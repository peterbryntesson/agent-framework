// Copyright (c) Microsoft. All rights reserved.

package agent

import (
	"errors"
	"fmt"
)

// Sentinel errors for common agent failure conditions.
// Use errors.Is to check for these conditions in error handling.
var (
	// ErrSessionNotFound indicates the requested session does not exist or has expired.
	ErrSessionNotFound = errors.New("session not found")

	// ErrInvalidInput indicates the input provided to the agent was malformed or invalid.
	ErrInvalidInput = errors.New("invalid input")

	// ErrRateLimited indicates the agent or provider has throttled requests due to rate limits.
	ErrRateLimited = errors.New("rate limited")

	// ErrProviderError indicates an error occurred in the underlying LLM provider.
	ErrProviderError = errors.New("provider error")

	// ErrToolInvocationFailed indicates a tool or function call failed during execution.
	ErrToolInvocationFailed = errors.New("tool invocation failed")
)

// Error provides structured error information for agent operations.
// It wraps an underlying error with contextual information about the operation
// and agent that encountered the error.
type Error struct {
	// Op is the operation that failed (e.g., "Run", "RunStream", "NewSession").
	Op string

	// AgentID is the identifier of the agent that encountered the error.
	AgentID string

	// Err is the underlying error.
	Err error
}

// Error returns a formatted error message including operation and agent context.
func (e *Error) Error() string {
	if e.AgentID != "" {
		return fmt.Sprintf("agent %s: %s: %v", e.AgentID, e.Op, e.Err)
	}
	return fmt.Sprintf("%s: %v", e.Op, e.Err)
}

// Unwrap returns the underlying error for use with errors.Is and errors.As.
func (e *Error) Unwrap() error {
	return e.Err
}

// NewError creates a new Error with the specified operation, agent ID, and underlying error.
func NewError(op string, agentID string, err error) *Error {
	return &Error{
		Op:      op,
		AgentID: agentID,
		Err:     err,
	}
}

// IsRetryable returns true if the error represents a condition that may succeed on retry.
// Errors such as rate limiting are considered retryable, while invalid input errors are not.
func IsRetryable(err error) bool {
	if err == nil {
		return false
	}

	// Check for known retryable sentinel errors
	if errors.Is(err, ErrRateLimited) {
		return true
	}

	// ErrProviderError may be retryable (transient provider issues)
	if errors.Is(err, ErrProviderError) {
		return true
	}

	// Non-retryable errors
	if errors.Is(err, ErrSessionNotFound) {
		return false
	}
	if errors.Is(err, ErrInvalidInput) {
		return false
	}
	if errors.Is(err, ErrToolInvocationFailed) {
		return false
	}

	// Default: unknown errors are not retryable
	return false
}
