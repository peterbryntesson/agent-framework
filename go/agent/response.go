// Copyright (c) Microsoft. All rights reserved.

package agent

import "encoding/json"

// Response represents the complete result of an agent run.
// This is a stub that will be fully implemented in User Story 1.2.2.
type Response struct {
	Metadata     map[string]interface{}
	Usage        *UsageDetails
	SessionState json.RawMessage
	Messages     []Message
	FinishReason FinishReason
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
// This is a stub that will be fully implemented in User Story 1.2.2.
type ResponseUpdate struct {
	// Metadata contains additional update metadata.
	Metadata map[string]interface{}

	// Delta contains the incremental content for ContentDelta updates.
	Delta *ContentDelta

	// Message contains the complete message for MessageComplete updates.
	Message *Message

	// Kind indicates the type of update.
	Kind UpdateKind
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

	// UpdateKindError indicates an error occurred.
	UpdateKindError

	// UpdateKindDone indicates the stream is complete.
	UpdateKindDone
)

// ContentDelta represents incremental content in a streaming response.
// This is a stub that will be fully implemented in User Story 1.2.2.
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
