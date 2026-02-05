// Copyright (c) Microsoft. All rights reserved.

package workflow

import (
	"context"
)

// ExecutorOptions configures executor behavior.
type ExecutorOptions struct {
	// AutoSendResult when true, automatically sends execute return values
	// as messages to connected executors. Default is true.
	AutoSendResult bool

	// AutoYieldResult when true, automatically yields execute return values
	// as workflow outputs. Default is true.
	AutoYieldResult bool
}

// DefaultExecutorOptions returns the default executor options.
// Both AutoSendResult and AutoYieldResult default to true, matching
// the .NET behavior.
func DefaultExecutorOptions() ExecutorOptions {
	return ExecutorOptions{
		AutoSendResult:  true,
		AutoYieldResult: true,
	}
}

// OptionsProvider is optionally implemented by executors that support options.
type OptionsProvider interface {
	Options() ExecutorOptions
}

// Executor processes messages in a workflow node.
// Implementations define the behavior of workflow nodes, receiving
// messages from connected edges and producing output messages.
//
// The Execute method is called during each superstep where the
// executor has pending messages. Executors should:
//   - Process incoming messages from wCtx.Messages()
//   - Send output messages using wCtx.Send()
//   - Use wCtx.Context() for cancellation checking
//   - Store state using wCtx.GetState()/SetState() if needed
type Executor interface {
	// ID returns the unique identifier for this executor.
	// IDs must be unique within a workflow.
	ID() string

	// Execute processes incoming messages and produces output.
	// The context provides access to messages, state, and output routing.
	// Returns an error if processing fails.
	Execute(ctx context.Context, wCtx *WorkflowContext) error
}

// ExecutorFunc is a function type that implements Executor.
// Use this for simple executors that don't need complex state.
type ExecutorFunc struct {
	id string
	fn func(ctx context.Context, wCtx *WorkflowContext) error
}

// NewExecutorFunc creates an Executor from a function.
func NewExecutorFunc(id string, fn func(ctx context.Context, wCtx *WorkflowContext) error) *ExecutorFunc {
	return &ExecutorFunc{id: id, fn: fn}
}

// ID returns the executor identifier.
func (ef *ExecutorFunc) ID() string {
	return ef.id
}

// Execute invokes the wrapped function.
func (ef *ExecutorFunc) Execute(ctx context.Context, wCtx *WorkflowContext) error {
	return ef.fn(ctx, wCtx)
}

// ExecutorBase provides common functionality for executor implementations.
// Embed this in custom executor types for convenience.
type ExecutorBase struct {
	id      string
	options ExecutorOptions
}

// ExecutorOption configures an ExecutorBase.
type ExecutorOption func(*ExecutorBase)

// NewExecutorBase creates a new ExecutorBase with the given ID.
// Options default to DefaultExecutorOptions() and can be customized
// using functional options.
func NewExecutorBase(id string, opts ...ExecutorOption) ExecutorBase {
	eb := ExecutorBase{
		id:      id,
		options: DefaultExecutorOptions(),
	}
	for _, opt := range opts {
		opt(&eb)
	}
	return eb
}

// ID returns the executor identifier.
func (eb *ExecutorBase) ID() string {
	return eb.id
}

// Options returns the executor options.
func (eb *ExecutorBase) Options() ExecutorOptions {
	return eb.options
}

// WithAutoSend sets whether executor return values are automatically sent
// as messages. Default is true.
func WithAutoSend(enabled bool) ExecutorOption {
	return func(eb *ExecutorBase) {
		eb.options.AutoSendResult = enabled
	}
}

// WithAutoYield sets whether executor return values are automatically yielded
// as outputs. Default is true.
func WithAutoYield(enabled bool) ExecutorOption {
	return func(eb *ExecutorBase) {
		eb.options.AutoYieldResult = enabled
	}
}
