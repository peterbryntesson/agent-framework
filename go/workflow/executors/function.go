// Copyright (c) Microsoft. All rights reserved.

package executors

import (
	"context"

	"github.com/microsoft/agent-framework-go/agent"
	"github.com/microsoft/agent-framework-go/workflow"
)

// FunctionHandler is the signature for function executor handlers.
// It receives incoming messages and returns output messages.
// This signature enables simple transformation use cases without
// requiring a full agent implementation.
type FunctionHandler func(ctx context.Context, messages []agent.Message) ([]agent.Message, error)

// FunctionExecutor wraps a function as a workflow Executor.
// Use this for simple transformations, routing logic, or any
// processing that doesn't require a full agent implementation.
type FunctionExecutor struct {
	workflow.ExecutorBase
	handler FunctionHandler
}

// NewFunctionExecutor creates a FunctionExecutor from a handler function.
// The id parameter should be unique within the workflow.
// The handler is invoked for each superstep where this executor receives messages.
func NewFunctionExecutor(id string, handler FunctionHandler) *FunctionExecutor {
	return &FunctionExecutor{
		ExecutorBase: workflow.NewExecutorBase(id),
		handler:      handler,
	}
}

// Execute processes incoming messages through the wrapped function.
// Messages from the workflow context are extracted, passed to the handler,
// and the returned messages are sent to downstream executors.
func (fe *FunctionExecutor) Execute(ctx context.Context, wCtx *workflow.WorkflowContext) error {
	messages := wCtx.Messages()
	if len(messages) == 0 {
		return nil
	}

	// Convert workflow messages to agent messages
	agentMessages := make([]agent.Message, 0, len(messages))
	for _, msg := range messages {
		agentMessages = append(agentMessages, msg.Content)
	}

	// Execute the handler
	outputs, err := fe.handler(ctx, agentMessages)
	if err != nil {
		return err
	}

	// Send output messages
	for _, msg := range outputs {
		wCtx.Send("", msg)
	}

	return nil
}

// Handler returns the underlying handler function.
func (fe *FunctionExecutor) Handler() FunctionHandler {
	return fe.handler
}
