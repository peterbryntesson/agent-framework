// Copyright (c) Microsoft. All rights reserved.

package durable

import (
	"encoding/json"
	"time"

	"github.com/microsoft/agent-framework-go/chat"
)

// StateEntry is the interface for conversation history entries.
// Both requests and responses implement this interface.
type StateEntry interface {
	// EntryType returns the type discriminator for this entry ("request" or "response").
	EntryType() string

	// CreatedAt returns when this entry was created.
	CreatedAt() time.Time

	// ToChatMessages converts this entry to chat.Message slice.
	ToChatMessages() []chat.Message

	// MessageCount returns the number of messages in this entry.
	MessageCount() int
}

// stateEntryJSON is used for polymorphic JSON deserialization.
type stateEntryJSON struct {
	Type string `json:"$type"`
}

// RequestEntry represents a user request in the conversation history.
type RequestEntry struct {
	// Type is the discriminator field, always "request".
	Type string `json:"$type"`

	// Timestamp is when this request was created (RFC 3339 format).
	Timestamp time.Time `json:"createdAt"`

	// Messages contains the user messages for this request.
	Messages []StateMessage `json:"messages"`

	// CorrelationID is an optional ID to correlate requests and responses.
	CorrelationID string `json:"correlationId,omitempty"`

	// OrchestrationID is the ID of the orchestration that initiated this request.
	OrchestrationID string `json:"orchestrationId,omitempty"`

	// ResponseType is the expected type of response (e.g., "text", "json").
	ResponseType string `json:"responseType,omitempty"`

	// ResponseSchema defines the expected structure if ResponseType is "json".
	ResponseSchema json.RawMessage `json:"responseSchema,omitempty"`
}

// EntryType returns "request".
func (r *RequestEntry) EntryType() string {
	return "request"
}

// CreatedAt returns the timestamp of this request.
func (r *RequestEntry) CreatedAt() time.Time {
	return r.Timestamp
}

// ToChatMessages converts this request to chat.Message slice.
func (r *RequestEntry) ToChatMessages() []chat.Message {
	messages := make([]chat.Message, 0, len(r.Messages))
	for _, m := range r.Messages {
		messages = append(messages, m.ToChatMessage())
	}
	return messages
}

// MessageCount returns the number of messages in this request.
func (r *RequestEntry) MessageCount() int {
	return len(r.Messages)
}

// NewRequestEntry creates a new request entry from chat messages.
func NewRequestEntry(messages []chat.Message) *RequestEntry {
	stateMessages := make([]StateMessage, 0, len(messages))
	for _, m := range messages {
		stateMessages = append(stateMessages, FromChatMessage(m))
	}
	return &RequestEntry{
		Type:      "request",
		Timestamp: time.Now().UTC(),
		Messages:  stateMessages,
	}
}

// NewRequestEntryWithCorrelation creates a new request entry with correlation ID.
func NewRequestEntryWithCorrelation(messages []chat.Message, correlationID string) *RequestEntry {
	entry := NewRequestEntry(messages)
	entry.CorrelationID = correlationID
	return entry
}

// ResponseEntry represents an agent response in the conversation history.
type ResponseEntry struct {
	// Type is the discriminator field, always "response".
	Type string `json:"$type"`

	// Timestamp is when this response was created (RFC 3339 format).
	Timestamp time.Time `json:"createdAt"`

	// Messages contains the assistant messages for this response.
	Messages []StateMessage `json:"messages"`

	// CorrelationID matches the request's correlation ID.
	CorrelationID string `json:"correlationId,omitempty"`

	// Usage contains token usage statistics.
	Usage *UsageInfo `json:"usage,omitempty"`

	// IsError indicates whether this response represents an error.
	IsError bool `json:"isError,omitempty"`
}

// EntryType returns "response".
func (r *ResponseEntry) EntryType() string {
	return "response"
}

// CreatedAt returns the timestamp of this response.
func (r *ResponseEntry) CreatedAt() time.Time {
	return r.Timestamp
}

// ToChatMessages converts this response to chat.Message slice.
func (r *ResponseEntry) ToChatMessages() []chat.Message {
	messages := make([]chat.Message, 0, len(r.Messages))
	for _, m := range r.Messages {
		messages = append(messages, m.ToChatMessage())
	}
	return messages
}

// MessageCount returns the number of messages in this response.
func (r *ResponseEntry) MessageCount() int {
	return len(r.Messages)
}

// NewResponseEntry creates a new response entry from chat messages.
func NewResponseEntry(messages []chat.Message) *ResponseEntry {
	stateMessages := make([]StateMessage, 0, len(messages))
	for _, m := range messages {
		stateMessages = append(stateMessages, FromChatMessage(m))
	}
	return &ResponseEntry{
		Type:      "response",
		Timestamp: time.Now().UTC(),
		Messages:  stateMessages,
	}
}

// NewResponseEntryWithUsage creates a new response entry with usage information.
func NewResponseEntryWithUsage(messages []chat.Message, usage *UsageInfo) *ResponseEntry {
	entry := NewResponseEntry(messages)
	entry.Usage = usage
	return entry
}

// UsageInfo contains token usage statistics.
type UsageInfo struct {
	InputTokenCount  int `json:"inputTokenCount,omitempty"`
	OutputTokenCount int `json:"outputTokenCount,omitempty"`
	TotalTokenCount  int `json:"totalTokenCount,omitempty"`
}

// StateMessage represents a single message in the conversation history.
type StateMessage struct {
	// Role is the role of the message author (user, assistant, system, tool).
	Role string `json:"role"`

	// AuthorName is the optional name of the message author.
	AuthorName string `json:"authorName,omitempty"`

	// Contents are the structured content items in this message.
	Contents []ContentItem `json:"contents,omitempty"`

	// CreatedAt is when this message was created.
	CreatedAt *time.Time `json:"createdAt,omitempty"`
}

// ToChatMessage converts this state message to a chat.Message.
func (m *StateMessage) ToChatMessage() chat.Message {
	contents := make([]chat.Content, 0, len(m.Contents))
	var toolCalls []chat.ToolCall

	for _, c := range m.Contents {
		content := c.ToChatContent()
		if content != nil {
			contents = append(contents, content)
		}
		// Extract tool calls from function call contents
		if c.Type == ContentTypeFunctionCall {
			toolCalls = append(toolCalls, chat.ToolCall{
				ID:        c.CallID,
				Name:      c.FunctionName,
				Arguments: c.Arguments,
			})
		}
	}

	msg := chat.Message{
		Role:     chat.Role(m.Role),
		Name:     m.AuthorName,
		Contents: contents,
	}

	if len(toolCalls) > 0 {
		msg.ToolCalls = toolCalls
	}

	if m.CreatedAt != nil {
		msg.CreatedAt = *m.CreatedAt
	}

	return msg
}

// FromChatMessage creates a StateMessage from a chat.Message.
func FromChatMessage(msg chat.Message) StateMessage {
	contents := make([]ContentItem, 0, len(msg.Contents)+len(msg.ToolCalls))

	// Convert content items
	for _, c := range msg.Contents {
		item := FromChatContent(c)
		if item != nil {
			contents = append(contents, *item)
		}
	}

	// Convert tool calls to function call content items
	for _, tc := range msg.ToolCalls {
		contents = append(contents, ContentItem{
			Type:         ContentTypeFunctionCall,
			CallID:       tc.ID,
			FunctionName: tc.Name,
			Arguments:    tc.Arguments,
		})
	}

	sm := StateMessage{
		Role:       string(msg.Role),
		AuthorName: msg.Name,
		Contents:   contents,
	}

	if !msg.CreatedAt.IsZero() {
		createdAt := msg.CreatedAt
		sm.CreatedAt = &createdAt
	}

	return sm
}

// ContentType constants for state content items.
const (
	ContentTypeText              = "text"
	ContentTypeFunctionCall      = "functionCall"
	ContentTypeFunctionResult    = "functionResult"
	ContentTypeData              = "data"
	ContentTypeError             = "error"
	ContentTypeHostedFile        = "hostedFile"
	ContentTypeHostedVectorStore = "hostedVectorStore"
	ContentTypeUsage             = "usage"
	ContentTypeReasoning         = "reasoning"
	ContentTypeURI               = "uri"
	ContentTypeUnknown           = "unknown"
)

// ContentItem represents a content item in a state message.
type ContentItem struct {
	// Type is the discriminator for this content type.
	Type string `json:"$type"`

	// Text is the text content (for "text" and "reasoning" types).
	Text string `json:"text,omitempty"`

	// CallID is the function call identifier (for "functionCall" and "functionResult" types).
	CallID string `json:"callId,omitempty"`

	// FunctionName is the name of the function (for "functionCall" type).
	FunctionName string `json:"name,omitempty"`

	// Arguments is the JSON arguments payload (for "functionCall" type).
	Arguments json.RawMessage `json:"arguments,omitempty"`

	// Result is the function result (for "functionResult" type).
	Result json.RawMessage `json:"result,omitempty"`

	// URI is the resource URI (for "data" and "uri" types).
	URI string `json:"uri,omitempty"`

	// MediaType is the MIME type (for "data" and "uri" types).
	MediaType string `json:"mediaType,omitempty"`

	// FileID is the hosted file identifier (for "hostedFile" type).
	FileID string `json:"fileId,omitempty"`

	// VectorStoreID is the vector store identifier (for "hostedVectorStore" type).
	VectorStoreID string `json:"vectorStoreId,omitempty"`

	// ErrorMessage is the error message (for "error" type).
	ErrorMessage string `json:"message,omitempty"`

	// ErrorCode is the error code (for "error" type).
	ErrorCode string `json:"errorCode,omitempty"`

	// ErrorDetails is additional error details (for "error" type).
	ErrorDetails string `json:"details,omitempty"`

	// Usage is the token usage info (for "usage" type).
	Usage *UsageInfo `json:"usage,omitempty"`

	// Content holds unknown content (for "unknown" type).
	Content json.RawMessage `json:"content,omitempty"`
}

// ToChatContent converts this content item to a chat.Content.
func (c *ContentItem) ToChatContent() chat.Content {
	switch c.Type {
	case ContentTypeText:
		return chat.NewTextContent(c.Text)
	case ContentTypeReasoning:
		// Reasoning is represented as text in the chat package
		return chat.NewTextContent(c.Text)
	case ContentTypeData:
		if c.URI == "" {
			return nil
		}
		return chat.NewImageContentFromBase64(c.URI, c.MediaType)
	case ContentTypeURI:
		if c.URI == "" {
			return nil
		}
		image := chat.NewImageContentFromURL(c.URI)
		image.MediaType = c.MediaType
		return image
	case ContentTypeFunctionResult:
		return chat.NewToolResultContent(c.CallID, string(c.Result))
	// For types that don't have a direct chat.Content equivalent,
	// we return nil and handle them separately
	case ContentTypeFunctionCall, ContentTypeHostedFile, ContentTypeHostedVectorStore,
		ContentTypeUsage, ContentTypeUnknown, ContentTypeError:
		return nil
	default:
		return nil
	}
}

// FromChatContent creates a ContentItem from a chat.Content.
func FromChatContent(c chat.Content) *ContentItem {
	if c == nil {
		return nil
	}

	switch v := c.(type) {
	case *chat.TextContent:
		return &ContentItem{
			Type: ContentTypeText,
			Text: v.Text,
		}
	case *chat.ToolResultContent:
		return &ContentItem{
			Type:   ContentTypeFunctionResult,
			CallID: v.ToolCallID,
			Result: json.RawMessage(v.Content),
		}
	case *chat.ImageContent:
		if v.URL != "" {
			return &ContentItem{
				Type:      ContentTypeURI,
				URI:       v.URL,
				MediaType: v.MediaType,
			}
		}
		return &ContentItem{
			Type:      ContentTypeData,
			URI:       v.Base64Data,
			MediaType: v.MediaType,
		}
	case *chat.ToolCallContent:
		return &ContentItem{
			Type:         ContentTypeFunctionCall,
			CallID:       v.ToolCall.ID,
			FunctionName: v.ToolCall.Name,
			Arguments:    v.ToolCall.Arguments,
		}
	default:
		// Serialize unknown content types
		data, _ := json.Marshal(c)
		return &ContentItem{
			Type:    ContentTypeUnknown,
			Content: data,
		}
	}
}

// UnmarshalStateEntry deserializes a StateEntry from JSON.
// It determines the concrete type from the "$type" field.
func UnmarshalStateEntry(data []byte) (StateEntry, error) {
	var typeInfo stateEntryJSON
	if err := json.Unmarshal(data, &typeInfo); err != nil {
		return nil, err
	}

	switch typeInfo.Type {
	case "request":
		var entry RequestEntry
		if err := json.Unmarshal(data, &entry); err != nil {
			return nil, err
		}
		return &entry, nil
	case "response":
		var entry ResponseEntry
		if err := json.Unmarshal(data, &entry); err != nil {
			return nil, err
		}
		return &entry, nil
	default:
		// Default to request for unknown types
		var entry RequestEntry
		if err := json.Unmarshal(data, &entry); err != nil {
			return nil, err
		}
		return &entry, nil
	}
}
