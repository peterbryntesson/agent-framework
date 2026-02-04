// Copyright (c) Microsoft. All rights reserved.

package workflow

import (
	"context"
)

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
	id string
}

// NewExecutorBase creates a new ExecutorBase with the given ID.
func NewExecutorBase(id string) ExecutorBase {
	return ExecutorBase{id: id}
}

// ID returns the executor identifier.
func (eb *ExecutorBase) ID() string {
	return eb.id
}
