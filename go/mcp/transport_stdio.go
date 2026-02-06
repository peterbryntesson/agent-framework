// Copyright (c) Microsoft. All rights reserved.

package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"sync"
)

// StdioTransport communicates with an MCP server via stdin/stdout.
// This transport spawns an external process and communicates using
// JSON-RPC messages over the process's standard input and output streams.
type StdioTransport struct {
	command string
	args    []string
	env     []string
	workDir string
	config  *transportConfig

	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout *bufio.Reader
	stderr io.ReadCloser

	mu       sync.Mutex
	started  bool
	closed   bool
	closedCh chan struct{}

	readOnce      sync.Once
	readErrOnce   sync.Once
	readErrCh     chan struct{}
	readErr       error
	notifications chan *Notification
	pendingMu     sync.Mutex
	pending       map[string]chan *Response
}

// StdioTransportOption configures a StdioTransport.
type StdioTransportOption func(*StdioTransport)

// NewStdioTransport creates a transport for a stdio-based MCP server.
// The command is the executable to run, and args are the command-line arguments.
func NewStdioTransport(command string, args []string, opts ...StdioTransportOption) *StdioTransport {
	t := &StdioTransport{
		command:  command,
		args:     args,
		config:   defaultTransportConfig(),
		closedCh: make(chan struct{}),
	}

	for _, opt := range opts {
		opt(t)
	}

	return t
}

// WithStdioEnv sets environment variables for the spawned process.
// The environment is specified as a list of "KEY=VALUE" strings.
func WithStdioEnv(env []string) StdioTransportOption {
	return func(t *StdioTransport) {
		t.env = env
	}
}

// WithStdioWorkDir sets the working directory for the spawned process.
func WithStdioWorkDir(dir string) StdioTransportOption {
	return func(t *StdioTransport) {
		t.workDir = dir
	}
}

// WithStdioTimeout sets the request timeout for the transport.
func WithStdioTimeout(opts ...TransportOption) StdioTransportOption {
	return func(t *StdioTransport) {
		t.config.applyOptions(opts)
	}
}

// Start spawns the MCP server process and establishes communication pipes.
func (t *StdioTransport) Start(ctx context.Context) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.closed {
		return newTransportError("start", ErrClosed)
	}

	if t.started {
		return newTransportError("start", errors.New("transport already started"))
	}

	// Create the command
	t.cmd = exec.CommandContext(ctx, t.command, t.args...) //nolint:gosec // G204: Command is user-configured for MCP server
	if len(t.env) > 0 {
		t.cmd.Env = t.env
	}
	if t.workDir != "" {
		t.cmd.Dir = t.workDir
	}

	// Set up pipes
	var err error
	t.stdin, err = t.cmd.StdinPipe()
	if err != nil {
		return newTransportError("start", fmt.Errorf("create stdin pipe: %w", err))
	}

	stdout, err := t.cmd.StdoutPipe()
	if err != nil {
		t.stdin.Close()
		return newTransportError("start", fmt.Errorf("create stdout pipe: %w", err))
	}
	t.stdout = bufio.NewReader(stdout)

	t.stderr, err = t.cmd.StderrPipe()
	if err != nil {
		t.stdin.Close()
		return newTransportError("start", fmt.Errorf("create stderr pipe: %w", err))
	}

	// Start the process
	if err := t.cmd.Start(); err != nil {
		t.stdin.Close()
		return newTransportError("start", fmt.Errorf("start process: %w", err))
	}

	if t.pending == nil {
		t.pending = make(map[string]chan *Response)
	}
	if t.notifications == nil {
		t.notifications = make(chan *Notification, 100)
	}
	if t.readErrCh == nil {
		t.readErrCh = make(chan struct{})
	}
	// Start a single reader to demultiplex responses and notifications.
	t.readOnce.Do(func() {
		go t.readLoop()
	})

	t.started = true
	return nil
}

// Send sends a JSON-RPC request and waits for the response.
// The request is serialized to JSON and written to the process's stdin.
// The response is read from the process's stdout.
func (t *StdioTransport) Send(ctx context.Context, req *Request) (*Response, error) {
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

	// Check context before sending
	select {
	case <-ctx.Done():
		return nil, newTransportError("send", ctx.Err())
	default:
	}

	// Encode and send the request
	encoder := json.NewEncoder(t.stdin)
	responseCh := make(chan *Response, 1)
	key := requestKey(req.ID)
	t.pendingMu.Lock()
	t.pending[key] = responseCh
	t.pendingMu.Unlock()
	if err := encoder.Encode(req); err != nil {
		t.pendingMu.Lock()
		delete(t.pending, key)
		t.pendingMu.Unlock()
		return nil, newTransportError("send", fmt.Errorf("encode request: %w", err))
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
		return nil, newTransportError("send", io.EOF)
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

// Receive receives a notification from the transport.
// For stdio transports, notifications are read from stdout when they arrive.
// This method blocks until a notification is received or the context is cancelled.
// Returns io.EOF when the transport is closed.
func (t *StdioTransport) Receive(ctx context.Context) (*Notification, error) {
	// Check if already closed
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
	case <-t.readErrCh:
		return nil, newTransportError("receive", t.readErr)
	case notif := <-t.notifications:
		return notif, nil
	}
}

// Close terminates the transport connection and kills the process.
func (t *StdioTransport) Close() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.closed {
		return nil
	}

	t.closed = true
	close(t.closedCh)

	var errs []error

	// Close stdin to signal the process
	if t.stdin != nil {
		if err := t.stdin.Close(); err != nil {
			errs = append(errs, fmt.Errorf("close stdin: %w", err))
		}
	}

	// Close stderr reader
	if t.stderr != nil {
		if err := t.stderr.Close(); err != nil {
			errs = append(errs, fmt.Errorf("close stderr: %w", err))
		}
	}

	// Kill the process if it's still running
	if t.cmd != nil && t.cmd.Process != nil {
		if err := t.cmd.Process.Kill(); err != nil {
			// Ignore "process already finished" errors
			if !errors.Is(err, exec.ErrNotFound) {
				errs = append(errs, fmt.Errorf("kill process: %w", err))
			}
		}
		// Wait for the process to exit to clean up resources
		_ = t.cmd.Wait()
	}

	if len(errs) > 0 {
		return newTransportError("close", errors.Join(errs...))
	}

	return nil
}

func (t *StdioTransport) readLoop() {
	for {
		line, err := t.stdout.ReadBytes('\n')
		if err != nil {
			t.signalReadError(err)
			return
		}

		if len(line) == 0 || (len(line) == 1 && line[0] == '\n') {
			continue
		}

		t.dispatchLine(line)
	}
}

func (t *StdioTransport) dispatchLine(line []byte) {
	var envelope map[string]json.RawMessage
	if err := json.Unmarshal(line, &envelope); err != nil {
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
		if err := json.Unmarshal(line, &notif); err != nil {
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
		if err := json.Unmarshal(line, &resp); err != nil {
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

func (t *StdioTransport) signalReadError(err error) {
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

// Stderr returns a reader for the process's stderr output.
// This can be used for logging or debugging purposes.
// Returns nil if the transport has not been started.
func (t *StdioTransport) Stderr() io.Reader {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.stderr
}
