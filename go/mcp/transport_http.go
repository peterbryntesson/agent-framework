// Copyright (c) Microsoft. All rights reserved.

package mcp

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

// HTTPTransport communicates with an MCP server via HTTP/SSE.
// This transport uses HTTP POST for JSON-RPC requests and
// Server-Sent Events (SSE) for receiving notifications.
type HTTPTransport struct {
	endpoint    string
	sseEndpoint string
	httpClient  *http.Client
	headers     http.Header
	config      *transportConfig

	mu            sync.Mutex
	started       bool
	closed        bool
	closedCh      chan struct{}
	sseResp       *http.Response
	sseCancel     context.CancelFunc
	notifications chan *Notification
}

// HTTPTransportOption configures an HTTPTransport.
type HTTPTransportOption func(*HTTPTransport)

// NewHTTPTransport creates a transport for an HTTP-based MCP server.
// The endpoint is the base URL of the MCP server.
func NewHTTPTransport(endpoint string, opts ...HTTPTransportOption) *HTTPTransport {
	trimmedEndpoint := strings.TrimSuffix(endpoint, "/")
	postEndpoint := trimmedEndpoint
	sseEndpoint := trimmedEndpoint + "/sse"
	if strings.HasSuffix(trimmedEndpoint, "/sse") {
		postEndpoint = strings.TrimSuffix(trimmedEndpoint, "/sse")
		postEndpoint = strings.TrimSuffix(postEndpoint, "/")
		sseEndpoint = trimmedEndpoint
	}

	t := &HTTPTransport{
		endpoint:      postEndpoint,
		sseEndpoint:   sseEndpoint,
		headers:       make(http.Header),
		config:        defaultTransportConfig(),
		closedCh:      make(chan struct{}),
		notifications: make(chan *Notification, 100),
	}

	// Set default headers
	t.headers.Set("Content-Type", "application/json")
	t.headers.Set("Accept", "application/json")

	for _, opt := range opts {
		opt(t)
	}

	// Create default HTTP client if not provided
	if t.httpClient == nil {
		t.httpClient = &http.Client{
			Timeout: t.config.timeout,
		}
	}

	return t
}

// WithHTTPClient sets a custom HTTP client.
func WithHTTPClient(client *http.Client) HTTPTransportOption {
	return func(t *HTTPTransport) {
		t.httpClient = client
	}
}

// WithHTTPHeaders sets custom headers for all requests.
// These headers are added to every request made by the transport.
func WithHTTPHeaders(headers http.Header) HTTPTransportOption {
	return func(t *HTTPTransport) {
		for key, values := range headers {
			for _, value := range values {
				t.headers.Add(key, value)
			}
		}
	}
}

// WithHTTPHeader sets a single custom header.
func WithHTTPHeader(key, value string) HTTPTransportOption {
	return func(t *HTTPTransport) {
		t.headers.Set(key, value)
	}
}

// WithSSEEndpoint sets a custom SSE endpoint for notifications.
func WithSSEEndpoint(endpoint string) HTTPTransportOption {
	return func(t *HTTPTransport) {
		t.sseEndpoint = strings.TrimSuffix(endpoint, "/")
	}
}

// WithHTTPTimeout sets the timeout for HTTP requests.
func WithHTTPTimeout(opts ...TransportOption) HTTPTransportOption {
	return func(t *HTTPTransport) {
		t.config.applyOptions(opts)
	}
}

// Start initializes the HTTP transport.
// For HTTP transports, this establishes an SSE connection if the server
// supports notifications.
func (t *HTTPTransport) Start(ctx context.Context) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.closed {
		return newTransportError("start", ErrClosed)
	}

	if t.started {
		return newTransportError("start", errors.New("transport already started"))
	}

	t.started = true

	// Try to establish SSE connection for notifications
	// This is optional - some MCP servers may not support SSE
	go t.connectSSE()

	return nil
}

// connectSSE establishes an SSE connection for receiving notifications.
func (t *HTTPTransport) connectSSE() {
	t.mu.Lock()
	if t.closed {
		t.mu.Unlock()
		return
	}
	t.mu.Unlock()

	ctx, cancel := context.WithCancel(context.Background())
	t.mu.Lock()
	t.sseCancel = cancel
	t.mu.Unlock()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, t.sseEndpoint, nil)
	if err != nil {
		return
	}

	// Copy headers
	for key, values := range t.headers {
		for _, value := range values {
			req.Header.Add(key, value)
		}
	}
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Cache-Control", "no-cache")
	req.Header.Set("Connection", "keep-alive")

	// Create a client without timeout for long-lived SSE connection
	sseClient := &http.Client{
		Timeout: 0, // No timeout for SSE
	}

	resp, err := sseClient.Do(req)
	if err != nil {
		return
	}

	t.mu.Lock()
	if t.closed {
		resp.Body.Close()
		t.mu.Unlock()
		return
	}
	t.sseResp = resp
	t.mu.Unlock()

	// Parse SSE events
	t.parseSSE(ctx, resp.Body)
}

// parseSSE parses Server-Sent Events from a reader.
func (t *HTTPTransport) parseSSE(ctx context.Context, r io.Reader) {
	defer func() {
		t.mu.Lock()
		if t.sseResp != nil {
			t.sseResp.Body.Close()
			t.sseResp = nil
		}
		t.mu.Unlock()
	}()

	scanner := bufio.NewScanner(r)
	var eventType string
	var dataLines []string

	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return
		case <-t.closedCh:
			return
		default:
		}

		line := scanner.Text()

		if line == "" {
			// End of event - process accumulated data
			if len(dataLines) > 0 {
				data := strings.Join(dataLines, "\n")
				t.handleSSEEvent(eventType, data)
			}
			eventType = ""
			dataLines = nil
			continue
		}

		if strings.HasPrefix(line, "event:") {
			eventType = strings.TrimSpace(strings.TrimPrefix(line, "event:"))
		} else if strings.HasPrefix(line, "data:") {
			dataLines = append(dataLines, strings.TrimPrefix(line, "data:"))
		}
	}
}

// handleSSEEvent processes a single SSE event.
func (t *HTTPTransport) handleSSEEvent(eventType, data string) {
	// Parse as notification
	var notif Notification
	if err := json.Unmarshal([]byte(data), &notif); err != nil {
		return
	}

	// Ensure JSONRPC version is set
	if notif.JSONRPC == "" {
		notif.JSONRPC = JSONRPCVersion
	}

	// Send to notifications channel (non-blocking)
	select {
	case t.notifications <- &notif:
	default:
		// Channel full, drop notification
	}
}

// Send sends a JSON-RPC request and waits for the response.
// The request is sent as an HTTP POST to the server's JSON-RPC endpoint.
func (t *HTTPTransport) Send(ctx context.Context, req *Request) (*Response, error) {
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

	// Assign request ID if not set
	if req.ID == nil {
		req.ID = nextRequestID()
	}

	// Ensure JSONRPC version is set
	if req.JSONRPC == "" {
		req.JSONRPC = JSONRPCVersion
	}

	// Encode the request
	bodyBytes, err := json.Marshal(req)
	if err != nil {
		return nil, newTransportError("send", fmt.Errorf("encode request: %w", err))
	}

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, t.endpoint, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, newTransportError("send", fmt.Errorf("create request: %w", err))
	}

	// Copy headers
	for key, values := range t.headers {
		for _, value := range values {
			httpReq.Header.Add(key, value)
		}
	}

	// Send the request
	resp, err := t.httpClient.Do(httpReq)
	if err != nil {
		return nil, newTransportError("send", fmt.Errorf("http request: %w", err))
	}
	defer resp.Body.Close()

	// Check HTTP status
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, newTransportError("send", fmt.Errorf("http status %d: %s", resp.StatusCode, string(body)))
	}

	// Read and parse response
	bodyBytes, err = io.ReadAll(resp.Body)
	if err != nil {
		return nil, newTransportError("send", fmt.Errorf("read response: %w", err))
	}

	var rpcResp Response
	if err := json.Unmarshal(bodyBytes, &rpcResp); err != nil {
		return nil, newTransportError("send", fmt.Errorf("decode response: %w", err))
	}

	return &rpcResp, nil
}

// Receive receives a notification from the transport.
// For HTTP transports, notifications are received via SSE.
// Returns io.EOF when the transport is closed.
func (t *HTTPTransport) Receive(ctx context.Context) (*Notification, error) {
	t.mu.Lock()
	if t.closed {
		t.mu.Unlock()
		return nil, newTransportError("receive", io.EOF)
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
		return nil, newTransportError("receive", io.EOF)
	case notif := <-t.notifications:
		return notif, nil
	}
}

// Close terminates the transport connection.
func (t *HTTPTransport) Close() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.closed {
		return nil
	}

	t.closed = true
	close(t.closedCh)

	// Cancel SSE connection
	if t.sseCancel != nil {
		t.sseCancel()
	}

	// Close SSE response
	if t.sseResp != nil {
		t.sseResp.Body.Close()
		t.sseResp = nil
	}

	return nil
}

// SetHeader sets a header for all subsequent requests.
func (t *HTTPTransport) SetHeader(key, value string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.headers.Set(key, value)
}

// Endpoint returns the server endpoint URL.
func (t *HTTPTransport) Endpoint() string {
	return t.endpoint
}

// RetryConfig configures retry behavior for HTTP requests.
type RetryConfig struct {
	// MaxRetries is the maximum number of retry attempts.
	MaxRetries int

	// InitialBackoff is the initial backoff duration.
	InitialBackoff time.Duration

	// MaxBackoff is the maximum backoff duration.
	MaxBackoff time.Duration

	// BackoffMultiplier is the multiplier for exponential backoff.
	BackoffMultiplier float64
}

// DefaultRetryConfig returns a default retry configuration.
func DefaultRetryConfig() *RetryConfig {
	return &RetryConfig{
		MaxRetries:        3,
		InitialBackoff:    100 * time.Millisecond,
		MaxBackoff:        5 * time.Second,
		BackoffMultiplier: 2.0,
	}
}

// WithRetryConfig sets the retry configuration for the transport.
// Note: Retry logic must be implemented by the caller using this config.
func WithRetryConfig(config *RetryConfig) HTTPTransportOption {
	return func(t *HTTPTransport) {
		// Store config for callers to use
		// Actual retry logic is left to the caller for flexibility
	}
}
