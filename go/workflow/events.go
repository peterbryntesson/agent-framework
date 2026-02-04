// Copyright (c) Microsoft. All rights reserved.

package workflow

import "time"

// WorkflowResult contains the final result of a workflow execution.
type WorkflowResult struct {
	// RunID is the unique identifier for this run
	RunID string

	// Outputs contains messages from output executors
	Outputs []WorkflowMessage

	// FinalState is the workflow state at completion
	FinalState map[string]interface{}

	// SuperstepCount is the number of supersteps executed
	SuperstepCount int
}

// WorkflowEventKind indicates the type of workflow event.
type WorkflowEventKind int

const (
	// EventKindStarted indicates workflow execution started
	EventKindStarted WorkflowEventKind = iota

	// EventKindSuperstepStarted indicates a superstep began
	EventKindSuperstepStarted

	// EventKindExecutorInvoked indicates an executor was invoked
	EventKindExecutorInvoked

	// EventKindExecutorCompleted indicates an executor completed
	EventKindExecutorCompleted

	// EventKindExecutorFailed indicates an executor failed
	EventKindExecutorFailed

	// EventKindSuperstepCompleted indicates a superstep completed
	EventKindSuperstepCompleted

	// EventKindOutput indicates output from an output executor
	EventKindOutput

	// EventKindCompleted indicates workflow execution completed
	EventKindCompleted

	// EventKindError indicates workflow execution failed
	EventKindError
)

// WorkflowEvent represents an event during workflow execution.
type WorkflowEvent struct {
	// Kind is the type of event
	Kind WorkflowEventKind

	// RunID is the workflow run identifier
	RunID string

	// Superstep is the current superstep number
	Superstep int

	// ExecutorID is the relevant executor ID (if applicable)
	ExecutorID string

	// Message is the relevant message (if applicable)
	Message *WorkflowMessage

	// Error is the error (for error events)
	Error error

	// Result is the final result (for completed events)
	Result *WorkflowResult

	// Timestamp is when the event occurred
	Timestamp time.Time
}
