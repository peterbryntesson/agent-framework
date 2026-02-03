// Copyright (c) Microsoft. All rights reserved.

package agent

import (
	"encoding/json"
)

// AgentContext holds context for agent middleware invocations.
// It provides access to the agent being invoked, the input messages,
// and fields for capturing the response.
type AgentContext struct {
	// Agent is the agent being invoked.
	Agent Agent

	// Messages contains the input messages for this invocation.
	Messages []Message

	// Session is the optional session for this invocation.
	Session Session

	// Options contains run configuration for this invocation.
	Options *RunConfig

	// Metadata allows middleware to attach arbitrary data.
	Metadata map[string]any

	// Response holds the result for non-streaming invocations.
	// Set by the terminal handler or by middleware to override the response.
	Response *Response

	// Stream holds the result channel for streaming invocations.
	// Set by the terminal handler when IsStreaming is true.
	Stream <-chan ResponseUpdate

	// IsStreaming indicates whether this is a streaming invocation.
	IsStreaming bool
}

// FunctionContext holds context for function middleware invocations.
// It provides access to the function being invoked, its arguments,
// and fields for capturing the result.
type FunctionContext struct {
	// FunctionName is the name of the function being invoked.
	FunctionName string

	// Arguments contains the JSON arguments for the function.
	Arguments json.RawMessage

	// Metadata allows middleware to attach arbitrary data.
	Metadata map[string]any

	// Result holds the function result after invocation.
	// Set by the terminal handler or by middleware to override.
	Result any

	// Error holds any error from the invocation.
	Error error
}

// ChatContext holds context for chat middleware invocations.
// It provides access to the chat client request parameters
// and fields for capturing the response.
type ChatContext struct {
	// ClientMetadata contains information about the chat client.
	ClientMetadata ChatClientMetadata

	// Messages contains the input messages for this request.
	Messages []Message

	// Options contains chat request options as key-value pairs.
	Options map[string]any

	// Metadata allows middleware to attach arbitrary data.
	Metadata map[string]any

	// Response holds the result for non-streaming invocations.
	// Set by the terminal handler or by middleware to override the response.
	Response *ChatResponse

	// Stream holds the result channel for streaming invocations.
	// Set by the terminal handler when IsStreaming is true.
	Stream <-chan ChatResponseUpdate

	// IsStreaming indicates whether this is a streaming invocation.
	IsStreaming bool
}

// ChatClientMetadata contains information about the chat client.
type ChatClientMetadata struct {
	// ProviderName identifies the LLM provider (e.g., "openai", "azure").
	ProviderName string

	// ModelID identifies the model being used.
	ModelID string

	// EndpointURI is the base URI of the API endpoint.
	EndpointURI string
}

// ChatResponse represents a chat completion response in middleware context.
// This is a lightweight wrapper that middleware can use without importing chat package.
type ChatResponse struct {
	// Text is the text content of the response.
	Text string

	// FinishReason indicates why the model stopped.
	FinishReason string

	// RawResponse holds the original response for type-specific access.
	RawResponse any
}

// ChatResponseUpdate represents an incremental streaming update.
type ChatResponseUpdate struct {
	// Kind indicates the type of update.
	Kind string

	// TextDelta contains incremental text content.
	TextDelta string

	// Error for error updates.
	Error error

	// RawUpdate holds the original update for type-specific access.
	RawUpdate any
}
