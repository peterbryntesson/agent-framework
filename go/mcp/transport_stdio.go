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

	t.started = true
	return nil
}

// Send sends a JSON-RPC request and waits for the response.
// The request is serialized to JSON and written to the process's stdin.
// The response is read from the process's stdout.
func (t *StdioTransport) Send(ctx context.Context, req *Request) (*Response, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.closed {
		return nil, newTransportError("send", ErrClosed)
	}

	if !t.started {
		return nil, newTransportError("send", ErrNotInitialized)
	}

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
	if err := encoder.Encode(req); err != nil {
		return nil, newTransportError("send", fmt.Errorf("encode request: %w", err))
	}

	// Read the response
	line, err := t.stdout.ReadBytes('\n')
	if err != nil {
		if err == io.EOF {
			return nil, newTransportError("send", io.EOF)
		}
		return nil, newTransportError("send", fmt.Errorf("read response: %w", err))
	}

	// Parse the response
	var resp Response
	if err := json.Unmarshal(line, &resp); err != nil {
		return nil, newTransportError("send", fmt.Errorf("decode response: %w", err))
	}

	return &resp, nil
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

	// Use a goroutine to read so we can respect context cancellation
	type readResult struct {
		line []byte
		err  error
	}
	resultCh := make(chan readResult, 1)

	go func() {
		line, err := t.stdout.ReadBytes('\n')
		resultCh <- readResult{line: line, err: err}
	}()

	select {
	case <-ctx.Done():
		return nil, newTransportError("receive", ctx.Err())
	case <-t.closedCh:
		return nil, newTransportError("receive", io.EOF)
	case result := <-resultCh:
		if result.err != nil {
			if result.err == io.EOF {
				return nil, newTransportError("receive", io.EOF)
			}
			return nil, newTransportError("receive", fmt.Errorf("read notification: %w", result.err))
		}

		// Try to parse as notification first
		var notif Notification
		if err := json.Unmarshal(result.line, &notif); err != nil {
			return nil, newTransportError("receive", fmt.Errorf("decode notification: %w", err))
		}

		// Verify it's a notification (no ID field in the raw JSON)
		var raw map[string]json.RawMessage
		if err := json.Unmarshal(result.line, &raw); err == nil {
			if _, hasID := raw["id"]; hasID {
				// This is a response, not a notification
				return nil, newTransportError("receive", errors.New("received response instead of notification"))
			}
		}

		return &notif, nil
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

// Stderr returns a reader for the process's stderr output.
// This can be used for logging or debugging purposes.
// Returns nil if the transport has not been started.
func (t *StdioTransport) Stderr() io.Reader {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.stderr
}
