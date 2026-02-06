// Copyright (c) Microsoft. All rights reserved.

package mcp

import (
	"context"
	"fmt"
	"io"
	"sync/atomic"
	"time"
)

// Transport defines the interface for MCP communication.
// Implementations handle the low-level details of sending and receiving
// JSON-RPC messages over various transport protocols (stdio, HTTP/SSE).
type Transport interface {
	// Start initializes the transport connection.
	// This must be called before Send or Receive.
	Start(ctx context.Context) error

	// Send sends a JSON-RPC request and waits for the response.
	// The request ID is automatically assigned if not set.
	Send(ctx context.Context, req *Request) (*Response, error)

	// Receive receives the next notification from the transport.
	// Returns io.EOF when the transport is closed.
	// For transports that don't support notifications, this may block
	// until Close is called.
	Receive(ctx context.Context) (*Notification, error)

	// Close terminates the transport connection and releases resources.
	// After Close is called, Send and Receive will return errors.
	Close() error
}

// TransportOption configures a transport.
type TransportOption func(*transportConfig)

// transportConfig holds common configuration for transports.
type transportConfig struct {
	timeout     time.Duration
	readTimeout time.Duration
}

// defaultTransportConfig returns the default transport configuration.
func defaultTransportConfig() *transportConfig {
	return &transportConfig{
		timeout:     30 * time.Second,
		readTimeout: 5 * time.Minute,
	}
}

// applyOptions applies the given options to the configuration.
func (c *transportConfig) applyOptions(opts []TransportOption) {
	for _, opt := range opts {
		opt(c)
	}
}

// WithTransportTimeout sets the request timeout for transport operations.
func WithTransportTimeout(d time.Duration) TransportOption {
	return func(c *transportConfig) {
		c.timeout = d
	}
}

// WithReadTimeout sets the timeout for reading responses and notifications.
func WithReadTimeout(d time.Duration) TransportOption {
	return func(c *transportConfig) {
		c.readTimeout = d
	}
}

// requestIDGenerator generates unique request IDs atomically.
var requestIDGenerator atomic.Int64

// nextRequestID returns the next unique request ID.
func nextRequestID() int64 {
	return requestIDGenerator.Add(1)
}

func requestKey(id interface{}) string {
	return fmt.Sprintf("%v", id)
}

// TransportError wraps transport-level errors with additional context.
type TransportError struct {
	// Op is the operation that failed (e.g., "start", "send", "receive", "close").
	Op string

	// Cause is the underlying error.
	Cause error
}

// Error implements the error interface.
func (e *TransportError) Error() string {
	if e.Cause != nil {
		return "mcp transport " + e.Op + ": " + e.Cause.Error()
	}
	return "mcp transport " + e.Op + " failed"
}

// Unwrap returns the underlying cause for errors.Is and errors.As.
func (e *TransportError) Unwrap() error {
	return e.Cause
}

// Is reports whether the error matches the target.
func (e *TransportError) Is(target error) bool {
	if target == io.EOF {
		return e.Cause == io.EOF
	}
	return false
}

// newTransportError creates a new TransportError.
func newTransportError(op string, cause error) *TransportError {
	return &TransportError{
		Op:    op,
		Cause: cause,
	}
}
