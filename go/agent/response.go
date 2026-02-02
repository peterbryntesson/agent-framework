// Copyright (c) Microsoft. All rights reserved.

package agent

import (
	"encoding/json"
	"time"
)

// Response represents the complete result of an agent run.
type Response struct {
	// Metadata contains response metadata.
	Metadata map[string]interface{}

	// AdditionalProperties for extensibility.
	AdditionalProperties map[string]interface{}

	// RawRepresentation holds provider-specific response data.
	RawRepresentation interface{}

	// Usage contains token usage information.
	Usage *UsageDetails

	// SessionState contains serialized session state.
	SessionState json.RawMessage

	// Messages contains the response messages.
	Messages []Message

	// ContinuationToken for resuming long-running operations.
	ContinuationToken string

	// ResponseID is the unique identifier for this response.
	ResponseID string

	// AgentID is the identifier of the agent that generated this response.
	AgentID string

	// CreatedAt is the timestamp when this response was created.
	CreatedAt time.Time

	// FinishReason indicates why generation stopped.
	FinishReason FinishReason
}

// Text returns the concatenated text content of all messages in the response.
// This is a convenience method for accessing plain text responses.
func (r *Response) Text() string {
	if r == nil || len(r.Messages) == 0 {
		return ""
	}
	var text string
	for i := range r.Messages {
		text += r.Messages[i].Text()
	}
	return text
}

// ToResponseUpdates converts this Response to a slice of ResponseUpdate for streaming compatibility.
// This enables non-streaming responses to be processed by streaming-compatible handlers.
// Each message in the response is converted to a MessageComplete update, followed by a Done update.
func (r *Response) ToResponseUpdates() []ResponseUpdate {
	if r == nil {
		return nil
	}

	updates := make([]ResponseUpdate, 0, len(r.Messages)+2)

	// Add a usage update if usage is available
	if r.Usage != nil {
		updates = append(updates, ResponseUpdate{
			Kind:       UpdateKindUsage,
			Usage:      r.Usage,
			ResponseID: r.ResponseID,
			CreatedAt:  r.CreatedAt,
		})
	}

	// Add MessageComplete updates for each message
	for i := range r.Messages {
		msg := &r.Messages[i]
		updates = append(updates, ResponseUpdate{
			Kind:       UpdateKindMessageComplete,
			Message:    msg,
			ResponseID: r.ResponseID,
			Role:       string(msg.Role),
			CreatedAt:  r.CreatedAt,
		})
	}

	// Add the final Done update
	updates = append(updates, ResponseUpdate{
		Kind:         UpdateKindDone,
		FinishReason: r.FinishReason,
		ResponseID:   r.ResponseID,
		CreatedAt:    r.CreatedAt,
	})

	return updates
}

// ResponseUpdate represents an incremental update during streaming.
type ResponseUpdate struct {
	// Metadata contains additional update metadata.
	Metadata map[string]interface{}

	// Error for error updates.
	Error error

	// Delta contains the incremental content for ContentDelta updates.
	Delta *ContentDelta

	// Message contains the complete message for MessageComplete updates.
	Message *Message

	// Usage contains token usage information for usage updates.
	Usage *UsageDetails

	// Kind indicates the type of update.
	Kind UpdateKind

	// FinishReason for completion updates.
	FinishReason FinishReason

	// Role indicates the role of the author (system, user, assistant, tool).
	Role string

	// AuthorName is an optional name for the message author.
	AuthorName string

	// ResponseID is the ID of the response of which this update is a part.
	ResponseID string

	// MessageID is the ID of the message of which this update is a part.
	MessageID string

	// CreatedAt is the timestamp when this update was created.
	CreatedAt time.Time
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

	// StatusCanceled indicates the run was canceled by the user.
	StatusCanceled AsyncRunStatus = "canceled"

	// StatusFailed indicates the run encountered an unrecoverable error.
	StatusFailed AsyncRunStatus = "failed"

	// StatusExpired indicates the run exceeded its time limit.
	StatusExpired AsyncRunStatus = "expired"
)

// IsTerminal returns true if the status represents a final state.
func (s AsyncRunStatus) IsTerminal() bool {
	switch s {
	case StatusCompleted, StatusCanceled, StatusFailed, StatusExpired:
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

	// ThreadID is the conversation thread associated with this run.
	ThreadID string `json:"thread_id,omitempty"`

	// Status indicates the current state of the async run.
	Status AsyncRunStatus `json:"status"`

	// ExpiresAt indicates when the run will expire if not completed.
	ExpiresAt *time.Time `json:"expires_at,omitempty"`

	// StartedAt indicates when the run started processing.
	StartedAt *time.Time `json:"started_at,omitempty"`

	// CompletedAt indicates when the run finished (success or failure).
	CompletedAt *time.Time `json:"completed_at,omitempty"`

	// Error contains error details if the run failed.
	Error *AsyncRunError `json:"error,omitempty"`
}
