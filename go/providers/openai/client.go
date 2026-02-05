// Copyright (c) Microsoft. All rights reserved.

package openai

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/microsoft/agent-framework-go/chat"
	"github.com/microsoft/agent-framework-go/tool"
	openailib "github.com/sashabaranov/go-openai"
)

// Default configuration values.
const (
	// DefaultModel is the default OpenAI model used for chat completions.
	DefaultModel = "gpt-4o"

	// DefaultInstructionRole is the default role used for system instructions.
	DefaultInstructionRole = "system"

	// DefaultBaseURL is the default OpenAI API base URL.
	DefaultBaseURL = "https://api.openai.com/v1"
)

// Environment variable names.
const (
	// EnvAPIKey is the environment variable name for the OpenAI API key.
	EnvAPIKey = "OPENAI_API_KEY" //nolint:gosec // G101: Not a credential, just an env var name

	// EnvBaseURL is the environment variable name for a custom API base URL.
	EnvBaseURL = "OPENAI_BASE_URL"

	// EnvOrgID is the environment variable name for the OpenAI organization ID.
	EnvOrgID = "OPENAI_ORG_ID"
)

// Client implements chat.Client for OpenAI's Chat Completions API.
// Aligns with Python OpenAIChatClient and .NET OpenAIChatClientExtensions.
type Client struct {
	client   *openailib.Client
	model    string
	metadata chat.ClientMetadata

	// Configuration
	instructionRole string // "system" or "developer"
}

// Ensure Client implements chat.Client.
var _ chat.Client = (*Client)(nil)

// NewClient creates a new OpenAI chat client.
//
// If no API key is provided via WithAPIKey, the client falls back to the
// OPENAI_API_KEY environment variable. Returns an error if no API key is available.
//
// Example:
//
//	// Use environment variable
//	client, err := openai.NewClient()
//
//	// Explicit configuration
//	client, err := openai.NewClient(
//	    openai.WithAPIKey("sk-..."),
//	    openai.WithModel("gpt-4o-mini"),
//	)
func NewClient(opts ...Option) (*Client, error) {
	cfg := defaultConfig()
	for _, opt := range opts {
		opt(cfg)
	}

	cfg.applyEnvDefaults()

	if cfg.apiKey == "" {
		return nil, errors.New("openai: API key required: use WithAPIKey or set OPENAI_API_KEY")
	}

	clientCfg := openailib.DefaultConfig(cfg.apiKey)

	if cfg.baseURL != "" {
		clientCfg.BaseURL = cfg.baseURL
	}

	if cfg.orgID != "" {
		clientCfg.OrgID = cfg.orgID
	}

	if cfg.httpClient != nil {
		clientCfg.HTTPClient = cfg.httpClient
	}

	endpointURI := clientCfg.BaseURL
	if endpointURI == "" {
		endpointURI = DefaultBaseURL
	}

	return &Client{
		client:          openailib.NewClientWithConfig(clientCfg),
		model:           cfg.model,
		instructionRole: cfg.instructionRole,
		metadata: chat.ClientMetadata{
			ProviderName: "openai",
			ModelID:      cfg.model,
			EndpointURI:  endpointURI,
		},
	}, nil
}

// Metadata returns provider-specific metadata about this client.
// This includes the provider name ("openai"), model ID, and endpoint information.
func (c *Client) Metadata() chat.ClientMetadata {
	return c.metadata
}

// GetResponse sends messages to the chat model and returns a complete response.
// The messages parameter contains the conversation history and system prompts.
// Options can configure model parameters like temperature, max tokens, and tools.
// Returns an error if the request fails or the context is canceled.
func (c *Client) GetResponse(ctx context.Context, messages []chat.Message, options *chat.Options) (*chat.Response, error) {
	req := c.buildRequest(messages, options)

	resp, err := c.client.CreateChatCompletion(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("openai: chat completion failed: %w", err)
	}

	if len(resp.Choices) == 0 {
		return nil, errors.New("openai: no choices returned in response")
	}

	choice := resp.Choices[0]
	return &chat.Response{
		Message:      fromOpenAIMessage(choice),
		FinishReason: fromOpenAIFinishReason(string(choice.FinishReason)),
		Usage: &chat.UsageDetails{
			InputTokens:  resp.Usage.PromptTokens,
			OutputTokens: resp.Usage.CompletionTokens,
			TotalTokens:  resp.Usage.TotalTokens,
		},
		RawRepresentation: resp,
	}, nil
}

// GetStreamingResponse sends messages and returns a channel of incremental updates.
// The channel receives ResponseUpdate values as content is generated.
// The channel is closed when generation completes or an error occurs.
// Callers must drain the channel completely to avoid resource leaks.
// Context cancellation stops the stream and closes the channel.
//
// The update kinds sent through the channel are:
//   - UpdateKindContentDelta: Incremental text content or tool call argument fragments
//   - UpdateKindToolCall: A tool call is being made (includes ID and name)
//   - UpdateKindMessageComplete: The message is complete (includes finish reason)
//   - UpdateKindUsage: Token usage information (sent if StreamOptions.IncludeUsage is true)
//   - UpdateKindError: An error occurred during streaming
//   - UpdateKindDone: The stream has completed successfully
func (c *Client) GetStreamingResponse(ctx context.Context, messages []chat.Message, options *chat.Options) (<-chan chat.ResponseUpdate, error) {
	req := c.buildRequest(messages, options)
	req.Stream = true
	req.StreamOptions = &openailib.StreamOptions{
		IncludeUsage: true,
	}

	stream, err := c.client.CreateChatCompletionStream(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("openai: stream creation failed: %w", err)
	}

	updates := make(chan chat.ResponseUpdate, 32) // Buffered for backpressure

	go func() {
		defer close(updates)
		defer stream.Close()

		ProcessStream(ctx, stream, updates)
	}()

	return updates, nil
}

// buildRequest creates an OpenAI chat completion request from messages and options.
func (c *Client) buildRequest(messages []chat.Message, options *chat.Options) openailib.ChatCompletionRequest {
	model := c.model
	if options != nil && options.ModelID != "" {
		model = options.ModelID
	}

	req := openailib.ChatCompletionRequest{
		Model:    model,
		Messages: toOpenAIMessages(messages, c.instructionRole),
	}

	if options == nil {
		return req
	}

	if options.MaxTokens > 0 {
		req.MaxTokens = options.MaxTokens
	}

	if options.Temperature != 0 {
		req.Temperature = options.Temperature
	}

	if options.TopP != 0 {
		req.TopP = options.TopP
	}

	if len(options.StopSequences) > 0 {
		req.Stop = options.StopSequences
	}

	if options.FrequencyPenalty != 0 {
		req.FrequencyPenalty = options.FrequencyPenalty
	}

	if options.PresencePenalty != 0 {
		req.PresencePenalty = options.PresencePenalty
	}

	if options.Seed != nil {
		req.Seed = options.Seed
	}

	if options.User != "" {
		req.User = options.User
	}

	if options.ResponseFormat != "" {
		req.ResponseFormat = &openailib.ChatCompletionResponseFormat{
			Type: openailib.ChatCompletionResponseFormatType(options.ResponseFormat),
		}
	}

	// Handle tools
	if len(options.Tools) > 0 {
		req.Tools = toOpenAITools(options.Tools)

		switch options.ToolChoice {
		case "auto":
			req.ToolChoice = "auto"
		case "required":
			req.ToolChoice = "required"
		case "none":
			req.ToolChoice = "none"
		default:
			if options.ToolChoice != "" {
				req.ToolChoice = openailib.ToolChoice{
					Type: openailib.ToolTypeFunction,
					Function: openailib.ToolFunction{
						Name: options.ToolChoice,
					},
				}
			}
		}
	}

	return req
}

// NewClientWithHTTPClient creates an OpenAI client with a custom HTTP configuration.
// This is a convenience function for common HTTP customization needs.
func NewClientWithHTTPClient(httpClient *http.Client, opts ...Option) (*Client, error) {
	opts = append([]Option{WithHTTPClient(httpClient)}, opts...)
	return NewClient(opts...)
}

// NewClientWithTimeout creates an OpenAI client with a custom timeout.
// This is a convenience function for setting request timeouts.
func NewClientWithTimeout(timeout time.Duration, opts ...Option) (*Client, error) {
	httpClient := &http.Client{
		Timeout: timeout,
	}
	return NewClientWithHTTPClient(httpClient, opts...)
}

// GetResponseWithTools sends messages with tool.Tool interfaces to the chat model.
// This is a convenience method that converts tool.Tool interfaces to the format
// expected by the Chat Completions API.
//
// For hosted tools (web search, code interpreter, etc.), use the Responses API
// client instead, as the Chat Completions API does not support hosted tools.
//
// Example:
//
//	tools := []tool.Tool{getWeatherTool, calculateTool}
//	resp, err := client.GetResponseWithTools(ctx, messages, tools, nil)
func (c *Client) GetResponseWithTools(ctx context.Context, messages []chat.Message, tools []tool.Tool, options *chat.Options) (*chat.Response, error) {
	if options == nil {
		options = &chat.Options{}
	}

	// Convert tool.Tool interfaces to chat.ToolDefinition
	toolDefs := make([]chat.ToolDefinition, 0, len(tools))
	for _, t := range tools {
		// Skip hosted tools - they're not supported in Chat Completions API
		if ht, ok := t.(tool.HostedTool); ok && ht.IsHosted() {
			continue
		}

		toolDefs = append(toolDefs, chat.ToolDefinition{
			Name:        t.Name(),
			Description: t.Description(),
			Parameters:  unmarshalParameters(t.Parameters()),
		})
	}

	// Merge with any existing tools in options
	if len(toolDefs) > 0 {
		options.Tools = append(options.Tools, toolDefs...)
	}

	return c.GetResponse(ctx, messages, options)
}

// GetStreamingResponseWithTools sends messages with tool.Tool interfaces and returns
// a streaming response. This is a convenience method that converts tool.Tool interfaces
// to the format expected by the Chat Completions API.
//
// For hosted tools (web search, code interpreter, etc.), use the Responses API
// client instead, as the Chat Completions API does not support hosted tools.
func (c *Client) GetStreamingResponseWithTools(ctx context.Context, messages []chat.Message, tools []tool.Tool, options *chat.Options) (<-chan chat.ResponseUpdate, error) {
	if options == nil {
		options = &chat.Options{}
	}

	// Convert tool.Tool interfaces to chat.ToolDefinition
	toolDefs := make([]chat.ToolDefinition, 0, len(tools))
	for _, t := range tools {
		// Skip hosted tools - they're not supported in Chat Completions API
		if ht, ok := t.(tool.HostedTool); ok && ht.IsHosted() {
			continue
		}

		toolDefs = append(toolDefs, chat.ToolDefinition{
			Name:        t.Name(),
			Description: t.Description(),
			Parameters:  unmarshalParameters(t.Parameters()),
		})
	}

	// Merge with any existing tools in options
	if len(toolDefs) > 0 {
		options.Tools = append(options.Tools, toolDefs...)
	}

	return c.GetStreamingResponse(ctx, messages, options)
}

// unmarshalParameters converts JSON Schema bytes to map for chat.ToolDefinition.
func unmarshalParameters(params []byte) map[string]interface{} {
	if len(params) == 0 {
		return nil
	}

	var result map[string]interface{}
	if err := json.Unmarshal(params, &result); err != nil {
		return nil
	}
	return result
}
