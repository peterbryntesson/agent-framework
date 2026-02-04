// Copyright (c) Microsoft. All rights reserved.

package workflow

import (
	"context"
	"sync"

	"github.com/microsoft/agent-framework-go/agent"
)

// WorkflowContext provides the execution context for an Executor.
// It is passed to each executor during workflow execution and provides
// access to workflow state, incoming messages, and output routing.
type WorkflowContext struct {
	// ctx is the parent context for cancellation and timeout
	ctx context.Context

	// executorID is the ID of the currently executing executor
	executorID string

	// runID is the unique identifier for this workflow run
	runID string

	// superstep is the current superstep number (0-based)
	superstep int

	// messages contains incoming messages for this executor
	messages []WorkflowMessage

	// state is the workflow-level shared state (thread-safe access required)
	state *sync.Map

	// outbox collects messages to send to other executors
	outbox []WorkflowMessage

	// mu protects outbox modifications
	mu sync.Mutex
}

// WorkflowMessage represents a message passed between executors.
type WorkflowMessage struct {
	// From is the executor ID that sent the message
	From string

	// To is the executor ID that should receive the message
	To string

	// Content contains the message payload
	Content agent.Message

	// Superstep is the superstep in which this message was created
	Superstep int
}

// Context returns the parent context for cancellation checking.
func (wc *WorkflowContext) Context() context.Context {
	return wc.ctx
}

// ExecutorID returns the ID of the currently executing executor.
func (wc *WorkflowContext) ExecutorID() string {
	return wc.executorID
}

// RunID returns the unique identifier for this workflow run.
func (wc *WorkflowContext) RunID() string {
	return wc.runID
}

// Superstep returns the current superstep number (0-based).
func (wc *WorkflowContext) Superstep() int {
	return wc.superstep
}

// Messages returns the incoming messages for this executor.
func (wc *WorkflowContext) Messages() []WorkflowMessage {
	return wc.messages
}

// GetState retrieves a value from the workflow-level shared state.
func (wc *WorkflowContext) GetState(key string) (interface{}, bool) {
	return wc.state.Load(key)
}

// SetState stores a value in the workflow-level shared state.
func (wc *WorkflowContext) SetState(key string, value interface{}) {
	wc.state.Store(key, value)
}

// Send queues a message to be sent to another executor.
// The message will be delivered in the next superstep.
func (wc *WorkflowContext) Send(to string, content agent.Message) {
	wc.mu.Lock()
	defer wc.mu.Unlock()
	wc.outbox = append(wc.outbox, WorkflowMessage{
		From:      wc.executorID,
		To:        to,
		Content:   content,
		Superstep: wc.superstep,
	})
}

// Outbox returns the messages queued for delivery.
// This is used internally by the workflow runner.
func (wc *WorkflowContext) Outbox() []WorkflowMessage {
	wc.mu.Lock()
	defer wc.mu.Unlock()
	result := make([]WorkflowMessage, len(wc.outbox))
	copy(result, wc.outbox)
	return result
}

// newWorkflowContext creates a new WorkflowContext for an executor.
func newWorkflowContext(
	ctx context.Context,
	executorID string,
	runID string,
	superstep int,
	messages []WorkflowMessage,
	state *sync.Map,
) *WorkflowContext {
	return &WorkflowContext{
		ctx:        ctx,
		executorID: executorID,
		runID:      runID,
		superstep:  superstep,
		messages:   messages,
		state:      state,
		outbox:     make([]WorkflowMessage, 0),
	}
}

// NewWorkflowContextForTest creates a WorkflowContext for use in tests.
// This function is exported for testing purposes and should not be
// used in production code.
func NewWorkflowContextForTest(
	ctx context.Context,
	executorID string,
	runID string,
	superstep int,
	messages []WorkflowMessage,
	state *sync.Map,
) *WorkflowContext {
	if state == nil {
		state = &sync.Map{}
	}
	return newWorkflowContext(ctx, executorID, runID, superstep, messages, state)
}
