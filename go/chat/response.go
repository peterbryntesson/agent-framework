// Copyright (c) Microsoft. All rights reserved.

package chat

// Response represents a complete chat completion response from a provider.
// This contains the generated message, usage information, and provider-specific data.
type Response struct {
	// Message is the generated response message from the model.
	Message Message

	// FinishReason indicates why the model stopped generating content.
	FinishReason FinishReason

	// Usage contains token usage information for billing and monitoring.
	Usage *UsageDetails

	// RawRepresentation holds the original provider-specific response data.
	// This enables access to provider-specific fields not in the common schema.
	RawRepresentation interface{}
}

// Text returns the text content of the response message.
// This is a convenience method for accessing plain text responses.
func (r *Response) Text() string {
	if r == nil {
		return ""
	}
	return r.Message.Text()
}

// ResponseUpdate represents an incremental update during streaming.
// This is sent through the channel returned by GetStreamingResponse.
type ResponseUpdate struct {
	// Kind indicates the type of update.
	Kind UpdateKind

	// Delta contains the incremental content for ContentDelta updates.
	Delta *ContentDelta

	// Message contains the complete message for MessageComplete updates.
	Message *Message

	// Usage contains token usage information for usage updates.
	Usage *UsageDetails

	// FinishReason for completion updates.
	FinishReason FinishReason

	// Error for error updates.
	Error error

	// Metadata contains additional update metadata.
	Metadata map[string]interface{}
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
	Role Role

	// TextDelta is the incremental text content.
	TextDelta string

	// ToolCallID is the ID of the tool call for tool-related deltas.
	ToolCallID string

	// Name is the name of the tool or function.
	Name string

	// ArgsDelta contains incremental function arguments as JSON.
	ArgsDelta string
}

// FinishReason indicates why the model stopped generating content.
type FinishReason int

const (
	// FinishReasonStop indicates normal completion.
	FinishReasonStop FinishReason = iota

	// FinishReasonLength indicates the maximum token limit was reached.
	FinishReasonLength

	// FinishReasonToolCalls indicates the model is waiting for tool results.
	FinishReasonToolCalls

	// FinishReasonContentFilter indicates content was filtered by safety systems.
	FinishReasonContentFilter
)
