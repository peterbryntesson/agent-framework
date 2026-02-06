// Copyright (c) Microsoft. All rights reserved.

package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

// WebSocketTransport communicates with an MCP server via WebSocket.
// It uses JSON-RPC messages over a single WebSocket connection and
// demultiplexes responses and notifications with a single reader.
type WebSocketTransport struct {
	endpoint string
	headers  http.Header
	dialer   *websocket.Dialer
	config   *transportConfig

	conn *websocket.Conn
	mu   sync.Mutex

	started  bool
	closed   bool
	closedCh chan struct{}

	readOnce    sync.Once
	readErrOnce sync.Once
	readErrCh   chan struct{}
	readErr     error

	pendingMu     sync.Mutex
	pending       map[string]chan *Response
	notifications chan *Notification
	writeMu       sync.Mutex
}

// WebSocketTransportOption configures a WebSocketTransport.
type WebSocketTransportOption func(*WebSocketTransport)

// NewWebSocketTransport creates a transport for a WebSocket-based MCP server.
func NewWebSocketTransport(endpoint string, opts ...WebSocketTransportOption) *WebSocketTransport {
	t := &WebSocketTransport{
		endpoint:      endpoint,
		headers:       make(http.Header),
		config:        defaultTransportConfig(),
		closedCh:      make(chan struct{}),
		readErrCh:     make(chan struct{}),
		pending:       make(map[string]chan *Response),
		notifications: make(chan *Notification, 100),
	}

	for _, opt := range opts {
		opt(t)
	}

	if t.dialer == nil {
		t.dialer = websocket.DefaultDialer
	}

	return t
}

// WithWebSocketHeaders sets custom headers for the WebSocket handshake.
func WithWebSocketHeaders(headers http.Header) WebSocketTransportOption {
	return func(t *WebSocketTransport) {
		for key, values := range headers {
			for _, value := range values {
				t.headers.Add(key, value)
			}
		}
	}
}

// WithWebSocketHeader sets a single custom header.
func WithWebSocketHeader(key, value string) WebSocketTransportOption {
	return func(t *WebSocketTransport) {
		t.headers.Set(key, value)
	}
}

// WithWebSocketDialer sets a custom WebSocket dialer.
func WithWebSocketDialer(dialer *websocket.Dialer) WebSocketTransportOption {
	return func(t *WebSocketTransport) {
		t.dialer = dialer
	}
}

// WithWebSocketTimeout sets the timeout for WebSocket transport operations.
func WithWebSocketTimeout(opts ...TransportOption) WebSocketTransportOption {
	return func(t *WebSocketTransport) {
		t.config.applyOptions(opts)
	}
}

// Start establishes the WebSocket connection.
func (t *WebSocketTransport) Start(ctx context.Context) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.closed {
		return newTransportError("start", ErrClosed)
	}

	if t.started {
		return newTransportError("start", errors.New("transport already started"))
	}

	conn, _, err := t.dialer.DialContext(ctx, t.endpoint, t.headers)
	if err != nil {
		return newTransportError("start", err)
	}

	t.conn = conn
	t.started = true

	t.readOnce.Do(func() {
		go t.readLoop()
	})

	return nil
}

// Send sends a JSON-RPC request and waits for the response.
func (t *WebSocketTransport) Send(ctx context.Context, req *Request) (*Response, error) {
	t.mu.Lock()
	if t.closed {
		t.mu.Unlock()
		return nil, newTransportError("send", ErrClosed)
	}
	if !t.started {
		t.mu.Unlock()
		return nil, newTransportError("send", ErrNotInitialized)
	}
	t.mu.Unlock()

	if req.ID == nil {
		req.ID = nextRequestID()
	}
	if req.JSONRPC == "" {
		req.JSONRPC = JSONRPCVersion
	}

	responseCh := make(chan *Response, 1)
	key := requestKey(req.ID)
	t.pendingMu.Lock()
	t.pending[key] = responseCh
	t.pendingMu.Unlock()

	t.writeMu.Lock()
	err := t.conn.WriteJSON(req)
	t.writeMu.Unlock()
	if err != nil {
		t.pendingMu.Lock()
		delete(t.pending, key)
		t.pendingMu.Unlock()
		return nil, newTransportError("send", err)
	}

	select {
	case <-ctx.Done():
		t.pendingMu.Lock()
		delete(t.pending, key)
		t.pendingMu.Unlock()
		return nil, newTransportError("send", ctx.Err())
	case <-t.closedCh:
		t.pendingMu.Lock()
		delete(t.pending, key)
		t.pendingMu.Unlock()
		return nil, newTransportError("send", ErrClosed)
	case <-t.readErrCh:
		t.pendingMu.Lock()
		delete(t.pending, key)
		t.pendingMu.Unlock()
		return nil, newTransportError("send", t.readErr)
	case resp, ok := <-responseCh:
		if !ok {
			return nil, newTransportError("send", t.readErr)
		}
		return resp, nil
	}
}

// Receive receives the next notification from the transport.
func (t *WebSocketTransport) Receive(ctx context.Context) (*Notification, error) {
	t.mu.Lock()
	if t.closed {
		t.mu.Unlock()
		return nil, newTransportError("receive", ErrClosed)
	}
	if !t.started {
		t.mu.Unlock()
		return nil, newTransportError("receive", ErrNotInitialized)
	}
	t.mu.Unlock()

	select {
	case <-ctx.Done():
		return nil, newTransportError("receive", ctx.Err())
	case <-t.closedCh:
		return nil, newTransportError("receive", ErrClosed)
	case <-t.readErrCh:
		return nil, newTransportError("receive", t.readErr)
	case notif := <-t.notifications:
		return notif, nil
	}
}

// Close terminates the WebSocket connection.
func (t *WebSocketTransport) Close() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.closed {
		return nil
	}

	t.closed = true
	close(t.closedCh)

	if t.conn != nil {
		_ = t.conn.Close()
	}

	return nil
}

func (t *WebSocketTransport) readLoop() {
	for {
		_, message, err := t.conn.ReadMessage()
		if err != nil {
			t.signalReadError(err)
			return
		}

		if len(message) == 0 {
			continue
		}

		t.dispatchMessage(message)
	}
}

func (t *WebSocketTransport) dispatchMessage(message []byte) {
	var envelope map[string]json.RawMessage
	if err := json.Unmarshal(message, &envelope); err != nil {
		return
	}

	_, hasMethod := envelope["method"]
	_, hasID := envelope["id"]

	if hasMethod {
		if hasID {
			// Server-initiated requests are not supported by this transport.
			return
		}
		var notif Notification
		if err := json.Unmarshal(message, &notif); err != nil {
			return
		}
		if notif.JSONRPC == "" {
			notif.JSONRPC = JSONRPCVersion
		}
		select {
		case t.notifications <- &notif:
		default:
		}
		return
	}

	if hasID {
		var resp Response
		if err := json.Unmarshal(message, &resp); err != nil {
			return
		}
		key := requestKey(resp.ID)
		t.pendingMu.Lock()
		responseCh, ok := t.pending[key]
		if ok {
			delete(t.pending, key)
		}
		t.pendingMu.Unlock()
		if ok {
			responseCh <- &resp
			close(responseCh)
		}
	}
}

func (t *WebSocketTransport) signalReadError(err error) {
	t.readErrOnce.Do(func() {
		t.readErr = err
		close(t.readErrCh)

		t.pendingMu.Lock()
		for key, responseCh := range t.pending {
			delete(t.pending, key)
			close(responseCh)
		}
		t.pendingMu.Unlock()
	})
}
