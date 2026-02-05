// Copyright (c) Microsoft. All rights reserved.

package chat

import (
	"context"
)

// Client defines the interface for chat completion providers.
// Implementations connect to LLM APIs (OpenAI, Azure OpenAI, Anthropic, etc.)
// and provide a consistent interface for getting chat responses.
//
// This interface is the foundation for agent implementations that use
// chat completion APIs. The ChatClientAgent wraps a Client to provide
// the full Agent interface.
type Client interface {
	// GetResponse sends messages to the chat model and returns a complete response.
	// The messages parameter contains the conversation history and system prompts.
	// Options can configure model parameters like temperature, max tokens, and tools.
	// Returns an error if the request fails or the context is canceled.
	GetResponse(ctx context.Context, messages []Message, options *Options) (*Response, error)

	// GetStreamingResponse sends messages and returns a channel of incremental updates.
	// The channel receives ResponseUpdate values as content is generated.
	// The channel is closed when generation completes or an error occurs.
	// Callers must drain the channel completely to avoid resource leaks.
	// Context cancellation stops the stream and closes the channel.
	GetStreamingResponse(ctx context.Context, messages []Message, options *Options) (<-chan ResponseUpdate, error)

	// Metadata returns provider-specific metadata about this client.
	// This includes the provider name, model ID, and endpoint information.
	Metadata() ClientMetadata
}

// ClientMetadata contains information about a chat client and its configuration.
// This is used for observability, logging, and debugging purposes.
type ClientMetadata struct {
	// ProviderName identifies the LLM provider (e.g., "openai", "azure", "anthropic").
	// This is used for OpenTelemetry semantic conventions and logging.
	ProviderName string

	// ModelID identifies the specific model being used (e.g., "gpt-4", "claude-3-opus").
	// For deployments, this may be the deployment name rather than the model name.
	ModelID string

	// EndpointURI is the base URI of the API endpoint.
	// This may be empty for providers that use default endpoints.
	EndpointURI string
}

// Options configures a chat completion request.
// All fields are optional; zero values indicate the model's defaults should be used.
type Options struct {
	// MaxTokens limits the maximum number of tokens in the response.
	// Zero means use the model's default limit.
	MaxTokens int

	// Temperature controls randomness in the response.
	// Values range from 0.0 (deterministic) to 2.0 (very random).
	// Zero means use the model's default temperature.
	Temperature float32

	// TopP controls nucleus sampling.
	// Zero means use the model's default top_p.
	TopP float32

	// StopSequences are strings that cause the model to stop generating.
	StopSequences []string

	// ResponseFormat specifies the expected response format.
	// Supported values depend on the provider.
	ResponseFormat string

	// Metadata contains additional provider-specific options.
	// This allows extensibility without modifying the Options struct.
	Metadata map[string]interface{}

	// Seed is used for reproducible outputs.
	// If specified, the model will attempt to produce deterministic results.
	Seed *int

	// LogitBias modifies the likelihood of specified tokens appearing in the output.
	// Map from token ID (string or int) to bias value (-100 to 100).
	LogitBias map[string]float32

	// FrequencyPenalty penalizes tokens based on their frequency in the response so far.
	// Values range from -2.0 to 2.0.
	FrequencyPenalty float32

	// PresencePenalty penalizes tokens based on whether they appear in the response so far.
	// Values range from -2.0 to 2.0.
	PresencePenalty float32

	// Tools are the tools/functions available for the model to call.
	Tools []ToolDefinition

	// ToolChoice specifies how the model should use tools.
	// Values: "auto" (model decides), "none" (no tools), "required" (must use a tool),
	// or a specific tool name to force calling that tool.
	ToolChoice string

	// Instructions provides a system prompt or additional instructions for the request.
	// This is merged with any system messages in the conversation.
	Instructions string

	// ModelID overrides the model to use for this request.
	// If empty, the client's default model is used.
	ModelID string

	// User is an identifier for the end-user, used for abuse monitoring.
	User string

	// Store indicates whether to store the conversation for later retrieval.
	Store bool

	// ConversationID links this request to an existing conversation for persistence.
	ConversationID string
}

// ToolDefinition describes a tool that the model can call.
type ToolDefinition struct {
	// Name is the name of the tool/function.
	Name string `json:"name"`

	// Description explains what the tool does.
	Description string `json:"description,omitempty"`

	// Parameters is the JSON Schema describing the tool's parameters.
	Parameters map[string]interface{} `json:"parameters,omitempty"`
}

// NewOptions creates a new Options with default values.
// Use the With* methods to configure specific options.
func NewOptions() *Options {
	return &Options{
		Metadata: make(map[string]interface{}),
	}
}
