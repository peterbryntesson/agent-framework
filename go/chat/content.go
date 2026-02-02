// Copyright (c) Microsoft. All rights reserved.

package chat

import "encoding/json"

// ContentType represents the type of content in a message.
type ContentType string

const (
	// ContentTypeText indicates text content.
	ContentTypeText ContentType = "text"

	// ContentTypeImage indicates image content.
	ContentTypeImage ContentType = "image"

	// ContentTypeToolCall indicates a tool/function call.
	ContentTypeToolCall ContentType = "tool_call"

	// ContentTypeToolResult indicates results from a tool invocation.
	ContentTypeToolResult ContentType = "tool_result"
)

// Content is the interface for message content types.
// This is a sealed interface - only types in this package can implement it.
// Implementations include TextContent, ImageContent, ToolCallContent, and ToolResultContent.
type Content interface {
	// Type returns the content type identifier.
	Type() ContentType

	// sealed prevents external implementations of the Content interface.
	sealed()
}

// contentBase is an unexported base type for sealing the Content interface.
type contentBase struct{}

func (contentBase) sealed() {}

// TextContent represents plain text content in a message.
type TextContent struct {
	contentBase

	// Text is the text content.
	Text string `json:"text"`
}

// Type returns ContentTypeText.
func (TextContent) Type() ContentType {
	return ContentTypeText
}

// NewTextContent creates a new TextContent with the given text.
func NewTextContent(text string) *TextContent {
	return &TextContent{Text: text}
}

// ImageContent represents image content in a message.
type ImageContent struct {
	contentBase

	// URL is the URL of the image.
	// Either URL or Base64Data should be set, not both.
	URL string `json:"url,omitempty"`

	// Base64Data is the base64-encoded image data.
	// Either URL or Base64Data should be set, not both.
	Base64Data string `json:"base64_data,omitempty"`

	// MediaType is the MIME type of the image (e.g., "image/png", "image/jpeg").
	MediaType string `json:"media_type,omitempty"`

	// Detail specifies the detail level for image processing.
	// Supported values depend on the provider (e.g., "auto", "low", "high").
	Detail string `json:"detail,omitempty"`
}

// Type returns ContentTypeImage.
func (ImageContent) Type() ContentType {
	return ContentTypeImage
}

// NewImageContentFromURL creates a new ImageContent with the given URL.
func NewImageContentFromURL(url string) *ImageContent {
	return &ImageContent{URL: url}
}

// NewImageContentFromBase64 creates a new ImageContent with base64-encoded data.
func NewImageContentFromBase64(base64Data, mediaType string) *ImageContent {
	return &ImageContent{
		Base64Data: base64Data,
		MediaType:  mediaType,
	}
}

// ToolCall represents a request to invoke a tool/function.
type ToolCall struct {
	// ID is the unique identifier for this tool call.
	ID string `json:"id"`

	// Name is the name of the tool/function to invoke.
	Name string `json:"name"`

	// Arguments contains the function arguments as JSON.
	Arguments json.RawMessage `json:"arguments"`
}

// ToolCallContent represents a tool/function call request in a message.
type ToolCallContent struct {
	contentBase

	// ToolCall contains the tool call details.
	ToolCall ToolCall `json:"tool_call"`
}

// Type returns ContentTypeToolCall.
func (ToolCallContent) Type() ContentType {
	return ContentTypeToolCall
}

// NewToolCallContent creates a new ToolCallContent with the given tool call details.
func NewToolCallContent(id, name string, arguments json.RawMessage) *ToolCallContent {
	return &ToolCallContent{
		ToolCall: ToolCall{
			ID:        id,
			Name:      name,
			Arguments: arguments,
		},
	}
}

// ToolResultContent represents the result of a tool/function invocation.
type ToolResultContent struct {
	contentBase

	// ToolCallID is the ID of the tool call this result responds to.
	ToolCallID string `json:"tool_call_id"`

	// Content is the result content, typically as text or JSON.
	Content string `json:"content"`

	// IsError indicates whether the result represents an error.
	IsError bool `json:"is_error,omitempty"`
}

// Type returns ContentTypeToolResult.
func (ToolResultContent) Type() ContentType {
	return ContentTypeToolResult
}

// NewToolResultContent creates a new ToolResultContent with the given result.
func NewToolResultContent(toolCallID, content string) *ToolResultContent {
	return &ToolResultContent{
		ToolCallID: toolCallID,
		Content:    content,
		IsError:    false,
	}
}

// NewToolResultContentWithError creates a new ToolResultContent representing an error.
func NewToolResultContentWithError(toolCallID, errorMessage string) *ToolResultContent {
	return &ToolResultContent{
		ToolCallID: toolCallID,
		Content:    errorMessage,
		IsError:    true,
	}
}
