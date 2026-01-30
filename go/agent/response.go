// Copyright (c) Microsoft. All rights reserved.

package agent

import (
	"encoding/json"
	"time"
)

// Response represents the complete result of an agent run.
type Response struct {
	Metadata     map[string]interface{}
	Usage        *UsageDetails
	SessionState json.RawMessage
	Messages     []Message
	FinishReason FinishReason

	// ContinuationToken for resuming long-running operations.
	ContinuationToken string

	// AdditionalProperties for extensibility.
	AdditionalProperties map[string]interface{}

	// RawRepresentation holds provider-specific response data.
	RawRepresentation interface{}
}

// Text returns the concatenated text content of all messages in the response.
// This is a convenience method for accessing plain text responses.
func (r *Response) Text() string {
	if r == nil || len(r.Messages) == 0 {
		return ""
	}
	var text string
	for _, msg := range r.Messages {
		text += msg.Content
	}
	return text
}

// ResponseUpdate represents an incremental update during streaming.
type ResponseUpdate struct {
	// Metadata contains additional update metadata.
	Metadata map[string]interface{}

	// Delta contains the incremental content for ContentDelta updates.
	Delta *ContentDelta

	// Message contains the complete message for MessageComplete updates.
	Message *Message

	// Kind indicates the type of update.
	Kind UpdateKind

	// Usage contains token usage information for usage updates.
	Usage *UsageDetails

	// FinishReason for completion updates.
	FinishReason FinishReason

	// Error for error updates.
	Error error
}

// UpdateKind represents the type of streaming update.
type UpdateKind int

const (
	// UpdateKindContentDelta indicates incremental content is available.
	UpdateKindContentDelta UpdateKind = iota

	// UpdateKindToolCall indicates a tool call is being made.
	UpdateKindToolCall

	// UpdateKindToolResult indicates a tool result is available.
	UpdateKindToolResult

	// UpdateKindMessageComplete indicates a complete message is available.
	UpdateKindMessageComplete

	// UpdateKindUsage indicates token usage information is available.
	UpdateKindUsage

	// UpdateKindError indicates an error occurred.
	UpdateKindError

	// UpdateKindDone indicates the stream is complete.
	UpdateKindDone
)

// ContentDelta represents incremental content in a streaming response.
type ContentDelta struct {
	// Role of the content author.
	Role string

	// TextDelta is the incremental text content.
	TextDelta string

	// ToolCallID is the ID of the tool call for tool-related deltas.
	ToolCallID string

	// Name is the name of the tool or function.
	Name string

	// ArgsDelta contains incremental function arguments as JSON.
	ArgsDelta string
}

// FinishReason indicates why the agent stopped generating content.
type FinishReason int

const (
	// FinishReasonStop indicates normal completion.
	FinishReasonStop FinishReason = iota

	// FinishReasonLength indicates the maximum token limit was reached.
	FinishReasonLength

	// FinishReasonToolCalls indicates the agent is waiting for tool results.
	FinishReasonToolCalls

	// FinishReasonContentFilter indicates content was filtered.
	FinishReasonContentFilter
)

// UsageDetails contains token usage information.
// This is a stub that will be fully implemented in User Story 1.3.4.
type UsageDetails struct {
	// InputTokens is the number of tokens in the input.
	InputTokens int

	// OutputTokens is the number of tokens in the output.
	OutputTokens int

	// TotalTokens is the total number of tokens used.
	TotalTokens int

	// CachedTokens is the number of cached tokens (optional).
	CachedTokens int

	// ReasoningTokens is the number of reasoning tokens (optional).
	ReasoningTokens int
}

// AsyncRunStatus represents the state of an async agent run.
type AsyncRunStatus string

const (
	// StatusQueued indicates the run is waiting to be processed.
	StatusQueued AsyncRunStatus = "queued"

	// StatusInProgress indicates the run is currently executing.
	StatusInProgress AsyncRunStatus = "in_progress"

	// StatusRequiresAction indicates the run needs user input (e.g., tool approval).
	StatusRequiresAction AsyncRunStatus = "requires_action"

	// StatusCompleted indicates the run finished successfully.
	StatusCompleted AsyncRunStatus = "completed"

	// StatusCancelled indicates the run was cancelled by the user.
	StatusCancelled AsyncRunStatus = "cancelled"

	// StatusFailed indicates the run encountered an unrecoverable error.
	StatusFailed AsyncRunStatus = "failed"

	// StatusExpired indicates the run exceeded its time limit.
	StatusExpired AsyncRunStatus = "expired"
)

// IsTerminal returns true if the status represents a final state.
func (s AsyncRunStatus) IsTerminal() bool {
	switch s {
	case StatusCompleted, StatusCancelled, StatusFailed, StatusExpired:
		return true
	default:
		return false
	}
}

// AsyncRunError contains details about a failed async run.
type AsyncRunError struct {
	// Code is the error code identifying the type of error.
	Code string `json:"code"`

	// Message is a human-readable description of the error.
	Message string `json:"message"`
}

// AsyncRunContent represents the status of a long-running agent operation.
// This is used when an agent run cannot complete synchronously and must be
// polled or awaited for completion.
type AsyncRunContent struct {
	// RunID is the unique identifier for the async run.
	RunID string `json:"run_id"`

	// Status indicates the current state of the async run.
	Status AsyncRunStatus `json:"status"`

	// ThreadID is the conversation thread associated with this run.
	ThreadID string `json:"thread_id,omitempty"`

	// ExpiresAt indicates when the run will expire if not completed.
	ExpiresAt *time.Time `json:"expires_at,omitempty"`

	// StartedAt indicates when the run started processing.
	StartedAt *time.Time `json:"started_at,omitempty"`

	// CompletedAt indicates when the run finished (success or failure).
	CompletedAt *time.Time `json:"completed_at,omitempty"`

	// Error contains error details if the run failed.
	Error *AsyncRunError `json:"error,omitempty"`
}
