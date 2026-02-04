// Copyright (c) Microsoft. All rights reserved.

package openai

import (
	"encoding/json"
	"time"

	"github.com/microsoft/agent-framework-go/chat"
)

// ChatCompletionRequest represents the OpenAI chat completions request format.
type ChatCompletionRequest struct {
	// Model is the model identifier to use for completion.
	Model string `json:"model"`

	// Messages is the list of messages in the conversation.
	Messages []ChatCompletionMessage `json:"messages"`

	// Temperature controls randomness in the response (0-2).
	Temperature *float64 `json:"temperature,omitempty"`

	// MaxTokens limits the maximum tokens in the response.
	MaxTokens *int `json:"max_tokens,omitempty"`

	// Stream indicates whether to stream the response.
	Stream bool `json:"stream,omitempty"`

	// Tools is the list of tools available for the model.
	Tools []Tool `json:"tools,omitempty"`

	// ToolChoice controls how the model selects tools.
	ToolChoice any `json:"tool_choice,omitempty"`

	// TopP controls nucleus sampling.
	TopP *float64 `json:"top_p,omitempty"`

	// N specifies the number of completions to generate.
	N *int `json:"n,omitempty"`

	// Stop sequences that stop generation.
	Stop any `json:"stop,omitempty"`

	// PresencePenalty penalizes new tokens based on presence in text so far.
	PresencePenalty *float64 `json:"presence_penalty,omitempty"`

	// FrequencyPenalty penalizes new tokens based on frequency in text so far.
	FrequencyPenalty *float64 `json:"frequency_penalty,omitempty"`

	// User is an optional unique identifier for the end-user.
	User string `json:"user,omitempty"`
}

// ChatCompletionMessage represents a message in the conversation.
type ChatCompletionMessage struct {
	// Role indicates who authored the message (system, user, assistant, tool).
	Role string `json:"role"`

	// Content is the message content. Can be a string or array of content parts.
	Content any `json:"content"`

	// Name is an optional name for the message author.
	Name string `json:"name,omitempty"`

	// ToolCalls contains tool calls requested by the assistant.
	ToolCalls []ToolCallMessage `json:"tool_calls,omitempty"`

	// ToolCallID identifies which tool call this message responds to.
	ToolCallID string `json:"tool_call_id,omitempty"`
}

// ToolCallMessage represents a tool call in a message.
type ToolCallMessage struct {
	// ID is the unique identifier for this tool call.
	ID string `json:"id"`

	// Type is always "function" for function calls.
	Type string `json:"type"`

	// Function contains the function call details.
	Function FunctionCall `json:"function"`
}

// FunctionCall represents the function being called.
type FunctionCall struct {
	// Name is the function name.
	Name string `json:"name"`

	// Arguments is the JSON-encoded function arguments.
	Arguments string `json:"arguments"`
}

// Tool represents a tool available to the model.
type Tool struct {
	// Type is the tool type (always "function").
	Type string `json:"type"`

	// Function contains the function definition.
	Function FunctionDefinition `json:"function"`
}

// FunctionDefinition defines a function tool.
type FunctionDefinition struct {
	// Name is the function name.
	Name string `json:"name"`

	// Description explains what the function does.
	Description string `json:"description,omitempty"`

	// Parameters is the JSON Schema for the function parameters.
	Parameters any `json:"parameters,omitempty"`
}

// ContentPart represents a part of multi-modal content.
type ContentPart struct {
	// Type indicates the content type ("text" or "image_url").
	Type string `json:"type"`

	// Text is the text content (when Type is "text").
	Text string `json:"text,omitempty"`

	// ImageURL contains image details (when Type is "image_url").
	ImageURL *ImageURL `json:"image_url,omitempty"`
}

// ImageURL contains image URL details.
type ImageURL struct {
	// URL is the image URL or base64-encoded data.
	URL string `json:"url"`

	// Detail controls image detail level ("auto", "low", "high").
	Detail string `json:"detail,omitempty"`
}

// ChatCompletionResponse represents the API response.
type ChatCompletionResponse struct {
	// ID is the unique identifier for this completion.
	ID string `json:"id"`

	// Object is always "chat.completion".
	Object string `json:"object"`

	// Created is the Unix timestamp when this was created.
	Created int64 `json:"created"`

	// Model is the model used for the completion.
	Model string `json:"model"`

	// Choices contains the completion choices.
	Choices []ChatCompletionChoice `json:"choices"`

	// Usage contains token usage information.
	Usage *Usage `json:"usage,omitempty"`

	// SystemFingerprint is the backend configuration fingerprint.
	SystemFingerprint string `json:"system_fingerprint,omitempty"`
}

// ChatCompletionChoice represents a single completion choice.
type ChatCompletionChoice struct {
	// Index is the choice index.
	Index int `json:"index"`

	// Message is the generated message.
	Message ChatCompletionMessage `json:"message"`

	// FinishReason indicates why generation stopped.
	FinishReason string `json:"finish_reason"`

	// Logprobs contains log probability information if requested.
	Logprobs any `json:"logprobs,omitempty"`
}

// Usage contains token usage information.
type Usage struct {
	// PromptTokens is the number of tokens in the prompt.
	PromptTokens int `json:"prompt_tokens"`

	// CompletionTokens is the number of tokens in the completion.
	CompletionTokens int `json:"completion_tokens"`

	// TotalTokens is the total number of tokens used.
	TotalTokens int `json:"total_tokens"`
}

// ChatCompletionChunk represents a streaming response chunk.
type ChatCompletionChunk struct {
	// ID is the unique identifier for this completion.
	ID string `json:"id"`

	// Object is always "chat.completion.chunk".
	Object string `json:"object"`

	// Created is the Unix timestamp when this was created.
	Created int64 `json:"created"`

	// Model is the model used for the completion.
	Model string `json:"model"`

	// Choices contains the chunk choices.
	Choices []ChatCompletionChunkChoice `json:"choices"`

	// SystemFingerprint is the backend configuration fingerprint.
	SystemFingerprint string `json:"system_fingerprint,omitempty"`

	// Usage contains token usage information (only in final chunk with stream_options).
	Usage *Usage `json:"usage,omitempty"`
}

// ChatCompletionChunkChoice represents a choice in a streaming chunk.
type ChatCompletionChunkChoice struct {
	// Index is the choice index.
	Index int `json:"index"`

	// Delta contains the incremental message content.
	Delta ChatCompletionDelta `json:"delta"`

	// FinishReason indicates why generation stopped (null until done).
	FinishReason *string `json:"finish_reason"`

	// Logprobs contains log probability information if requested.
	Logprobs any `json:"logprobs,omitempty"`
}

// ChatCompletionDelta represents incremental content in a streaming response.
type ChatCompletionDelta struct {
	// Role indicates the message role (only in first chunk).
	Role string `json:"role,omitempty"`

	// Content is the incremental text content.
	Content string `json:"content,omitempty"`

	// ToolCalls contains incremental tool call information.
	ToolCalls []ToolCallDelta `json:"tool_calls,omitempty"`
}

// ToolCallDelta represents incremental tool call information.
type ToolCallDelta struct {
	// Index is the tool call index.
	Index int `json:"index"`

	// ID is the tool call ID (only in first chunk for this tool).
	ID string `json:"id,omitempty"`

	// Type is the tool type (only in first chunk for this tool).
	Type string `json:"type,omitempty"`

	// Function contains incremental function call details.
	Function *FunctionCallDelta `json:"function,omitempty"`
}

// FunctionCallDelta represents incremental function call information.
type FunctionCallDelta struct {
	// Name is the function name (only in first chunk for this tool).
	Name string `json:"name,omitempty"`

	// Arguments contains incremental argument content.
	Arguments string `json:"arguments,omitempty"`
}

// ErrorResponse represents an API error response.
type ErrorResponse struct {
	// Error contains the error details.
	Error ErrorDetail `json:"error"`
}

// ErrorDetail contains error details.
type ErrorDetail struct {
	// Type is the error type.
	Type string `json:"type"`

	// Message is a human-readable error message.
	Message string `json:"message"`

	// Code is an optional error code.
	Code string `json:"code,omitempty"`

	// Param is the parameter related to the error.
	Param string `json:"param,omitempty"`
}

// ModelsResponse represents the response from GET /v1/models.
type ModelsResponse struct {
	// Object is always "list".
	Object string `json:"object"`

	// Data contains the list of models.
	Data []ModelInfo `json:"data"`
}

// ModelInfo represents information about a model.
type ModelInfo struct {
	// ID is the model identifier.
	ID string `json:"id"`

	// Object is always "model".
	Object string `json:"object"`

	// Created is the Unix timestamp when the model was created.
	Created int64 `json:"created"`

	// OwnedBy indicates who owns the model.
	OwnedBy string `json:"owned_by"`
}

// ToAgentMessages converts OpenAI messages to agent messages.
func ToAgentMessages(messages []ChatCompletionMessage) []chat.Message {
	result := make([]chat.Message, 0, len(messages))
	for _, m := range messages {
		result = append(result, toAgentMessage(m))
	}
	return result
}

// toAgentMessage converts a single OpenAI message to an agent message.
func toAgentMessage(m ChatCompletionMessage) chat.Message {
	role := chat.Role(m.Role)

	var contents []chat.Content
	switch c := m.Content.(type) {
	case string:
		if c != "" {
			contents = append(contents, chat.NewTextContent(c))
		}
	case []any:
		for _, part := range c {
			if partMap, ok := part.(map[string]any); ok {
				contents = append(contents, contentPartToContent(partMap))
			}
		}
	}

	var toolCalls []chat.ToolCall
	for _, tc := range m.ToolCalls {
		toolCalls = append(toolCalls, chat.ToolCall{
			ID:        tc.ID,
			Name:      tc.Function.Name,
			Arguments: json.RawMessage(tc.Function.Arguments),
		})
	}

	return chat.Message{
		Role:       role,
		Contents:   contents,
		Name:       m.Name,
		ToolCalls:  toolCalls,
		ToolCallID: m.ToolCallID,
		CreatedAt:  time.Now(),
	}
}

// contentPartToContent converts a content part map to a Content interface.
func contentPartToContent(part map[string]any) chat.Content {
	partType, _ := part["type"].(string)
	switch partType {
	case "text":
		text, _ := part["text"].(string)
		return chat.NewTextContent(text)
	case "image_url":
		if imageURL, ok := part["image_url"].(map[string]any); ok {
			url, _ := imageURL["url"].(string)
			return chat.NewImageContentFromURL(url)
		}
	}
	return chat.NewTextContent("")
}

// FromAgentMessage converts an agent message to an OpenAI message.
func FromAgentMessage(m chat.Message) ChatCompletionMessage {
	msg := ChatCompletionMessage{
		Role:       string(m.Role),
		Name:       m.Name,
		ToolCallID: m.ToolCallID,
	}

	// Convert contents to appropriate format
	if len(m.Contents) == 1 {
		if tc, ok := m.Contents[0].(*chat.TextContent); ok {
			msg.Content = tc.Text
		}
	} else if len(m.Contents) > 1 {
		parts := make([]ContentPart, 0, len(m.Contents))
		for _, c := range m.Contents {
			parts = append(parts, contentToContentPart(c))
		}
		msg.Content = parts
	}

	// Convert tool calls
	for _, tc := range m.ToolCalls {
		msg.ToolCalls = append(msg.ToolCalls, ToolCallMessage{
			ID:   tc.ID,
			Type: "function",
			Function: FunctionCall{
				Name:      tc.Name,
				Arguments: string(tc.Arguments),
			},
		})
	}

	return msg
}

// contentToContentPart converts a Content to a ContentPart.
func contentToContentPart(c chat.Content) ContentPart {
	switch ct := c.(type) {
	case *chat.TextContent:
		return ContentPart{
			Type: "text",
			Text: ct.Text,
		}
	case *chat.ImageContent:
		url := ct.URL
		if url == "" && ct.Base64Data != "" {
			url = "data:" + ct.MediaType + ";base64," + ct.Base64Data
		}
		return ContentPart{
			Type:     "image_url",
			ImageURL: &ImageURL{URL: url},
		}
	default:
		return ContentPart{Type: "text"}
	}
}
