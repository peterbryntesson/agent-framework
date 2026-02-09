// Copyright (c) Microsoft. All rights reserved.

package openai

import (
	"encoding/json"
)

// ResponseStatus defines the state of a response.
type ResponseStatus string

const (
	ResponseStatusCompleted  ResponseStatus = "completed"
	ResponseStatusFailed     ResponseStatus = "failed"
	ResponseStatusInProgress ResponseStatus = "in_progress"
	ResponseStatusCancelled  ResponseStatus = "cancelled"
	ResponseStatusQueued     ResponseStatus = "queued"
	ResponseStatusIncomplete ResponseStatus = "incomplete"
)

// CreateResponse represents a request to create a response.
type CreateResponse struct {
	Input json.RawMessage `json:"input"`

	Model string `json:"model,omitempty"`

	Instructions string `json:"instructions,omitempty"`

	Stream *bool `json:"stream,omitempty"`

	PreviousResponseID string `json:"previous_response_id,omitempty"`

	Conversation *ConversationReference `json:"conversation,omitempty"`

	Background *bool `json:"background,omitempty"`

	Store *bool `json:"store,omitempty"`

	Tools []json.RawMessage `json:"tools,omitempty"`

	ToolChoice json.RawMessage `json:"tool_choice,omitempty"`

	Temperature *float64 `json:"temperature,omitempty"`

	TopP *float64 `json:"top_p,omitempty"`

	Metadata map[string]string `json:"metadata,omitempty"`

	Include []string `json:"include,omitempty"`
}

// Response represents a response from the Responses API.
type Response struct {
	ID string `json:"id"`

	Object string `json:"object"`

	CreatedAt int64 `json:"created_at"`

	Model string `json:"model,omitempty"`

	Status ResponseStatus `json:"status"`

	Error *ResponseError `json:"error,omitempty"`

	Output []ItemResource `json:"output"`

	Instructions string `json:"instructions,omitempty"`

	Usage *ResponseUsage `json:"usage,omitempty"`

	Conversation *ConversationReference `json:"conversation,omitempty"`

	PreviousResponseID string `json:"previous_response_id,omitempty"`

	Background *bool `json:"background,omitempty"`

	Metadata map[string]string `json:"metadata,omitempty"`
}

// ResponseUsage describes token usage.
type ResponseUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
	TotalTokens  int `json:"total_tokens"`
}

// ResponseError represents an error response.
type ResponseError struct {
	Message string `json:"message"`
	Code    string `json:"code,omitempty"`
}

// StreamingResponseEvent represents an SSE event for streaming responses.
type StreamingResponseEvent struct {
	Type           string        `json:"type"`
	SequenceNumber int           `json:"sequence_number"`
	Response       *Response     `json:"response,omitempty"`
	OutputIndex    int           `json:"output_index,omitempty"`
	ContentIndex   int           `json:"content_index,omitempty"`
	Item           *ItemResource `json:"item,omitempty"`
	Delta          string        `json:"delta,omitempty"`
	Text           string        `json:"text,omitempty"`
}

// ResponseInputMessage represents a single input message.
type ResponseInputMessage struct {
	Role    string      `json:"role"`
	Content interface{} `json:"content"`
	Name    string      `json:"name,omitempty"`
}

// ItemParam represents a request item for conversations or responses.
type ItemParam struct {
	Type      string      `json:"type"`
	Role      string      `json:"role,omitempty"`
	Content   interface{} `json:"content,omitempty"`
	CallID    string      `json:"call_id,omitempty"`
	Name      string      `json:"name,omitempty"`
	Arguments string      `json:"arguments,omitempty"`
	Output    string      `json:"output,omitempty"`
}

// ItemResource represents a stored item.
type ItemResource struct {
	ID        string      `json:"id"`
	Type      string      `json:"type"`
	Role      string      `json:"role,omitempty"`
	Content   interface{} `json:"content,omitempty"`
	CallID    string      `json:"call_id,omitempty"`
	Name      string      `json:"name,omitempty"`
	Arguments string      `json:"arguments,omitempty"`
	Output    string      `json:"output,omitempty"`
	CreatedAt int64       `json:"created_at"`
}

// ListItemsResponse represents a list response for items.
type ListItemsResponse struct {
	Object  string         `json:"object,omitempty"`
	Data    []ItemResource `json:"data"`
	FirstID string         `json:"first_id,omitempty"`
	LastID  string         `json:"last_id,omitempty"`
	HasMore bool           `json:"has_more"`
}

// DeleteResponse represents a delete result.
type DeleteResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Deleted bool   `json:"deleted"`
}
