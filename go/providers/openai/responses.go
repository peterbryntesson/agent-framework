// Copyright (c) Microsoft. All rights reserved.

package openai

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"

	"github.com/microsoft/agent-framework-go/chat"
	"github.com/microsoft/agent-framework-go/tool"
)

// ResponsesAPIPath is the endpoint path for the Responses API.
const ResponsesAPIPath = "/v1/responses"

// ResponsesClient implements chat.Client for OpenAI's Responses API.
// The Responses API supports stateful conversations, hosted tools (web search,
// code interpreter), and response continuation via response IDs.
//
// Unlike the Chat Completions API, the Responses API:
//   - Supports provider-hosted tools like web search and code interpreter
//   - Maintains conversation state via response IDs
//   - Uses a different message format and request structure
//
// Example:
//
//	client, err := openai.NewResponsesClient(
//	    openai.ResponsesWithAPIKey("sk-..."),
//	    openai.ResponsesWithModel("gpt-4o"),
//	)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	messages := []chat.Message{
//	    chat.NewUserMessage("What's the weather in Seattle?"),
//	}
//
//	// Use web search tool
//	tools := []tool.Tool{tool.NewHostedWebSearchTool()}
//	opts := &chat.Options{Instructions: "You are a helpful assistant."}
//	resp, err := client.GetResponseWithTools(ctx, messages, tools, opts)
type ResponsesClient struct {
	httpClient *http.Client
	model      string
	baseURL    string
	apiKey     string
	orgID      string
	metadata   chat.ClientMetadata

	// Configuration
	instructionRole string
	instructions    string

	// Responses API specific state
	mu                 sync.RWMutex
	previousResponseID string
}

// Ensure ResponsesClient implements chat.Client.
var _ chat.Client = (*ResponsesClient)(nil)

// NewResponsesClient creates a new Responses API client.
//
// The Responses API is a stateful conversation API that supports hosted tools
// like web search and code interpreter. Use this client when you need:
//   - Provider-hosted tool capabilities (web search, code interpreter)
//   - Conversation continuation via response IDs
//   - The newer Responses API features
//
// If no API key is provided via ResponsesWithAPIKey, the client falls back to
// the OPENAI_API_KEY environment variable. Returns an error if no API key is available.
//
// Example:
//
//	// Use environment variable
//	client, err := openai.NewResponsesClient()
//
//	// Explicit configuration
//	client, err := openai.NewResponsesClient(
//	    openai.ResponsesWithAPIKey("sk-..."),
//	    openai.ResponsesWithModel("gpt-4o"),
//	    openai.ResponsesWithInstructions("You are a helpful assistant."),
//	)
func NewResponsesClient(opts ...ResponsesOption) (*ResponsesClient, error) {
	cfg := defaultResponsesConfig()
	for _, opt := range opts {
		opt(cfg)
	}

	cfg.applyEnvDefaults()

	if cfg.apiKey == "" {
		return nil, errors.New("openai: API key required: use ResponsesWithAPIKey or set OPENAI_API_KEY")
	}

	httpClient := cfg.httpClient
	if httpClient == nil {
		httpClient = &http.Client{}
	}

	baseURL := cfg.baseURL
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}

	return &ResponsesClient{
		httpClient:      httpClient,
		model:           cfg.model,
		baseURL:         baseURL,
		apiKey:          cfg.apiKey,
		orgID:           cfg.orgID,
		instructionRole: cfg.instructionRole,
		instructions:    cfg.instructions,
		metadata: chat.ClientMetadata{
			ProviderName: "openai-responses",
			ModelID:      cfg.model,
			EndpointURI:  baseURL,
		},
	}, nil
}

// Metadata returns provider-specific metadata about this client.
// This includes the provider name ("openai-responses"), model ID, and endpoint information.
func (c *ResponsesClient) Metadata() chat.ClientMetadata {
	return c.metadata
}

// GetPreviousResponseID returns the ID of the last response, if available.
// This can be used for conversation continuation.
func (c *ResponsesClient) GetPreviousResponseID() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.previousResponseID
}

// SetPreviousResponseID sets the response ID for conversation continuation.
// Use this to resume a conversation from a previous response.
func (c *ResponsesClient) SetPreviousResponseID(id string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.previousResponseID = id
}

// ClearConversation clears the stored response ID, starting a fresh conversation.
func (c *ResponsesClient) ClearConversation() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.previousResponseID = ""
}

// GetResponse sends messages to the Responses API and returns a complete response.
// This method does not support hosted tools; use GetResponseWithTools for that.
func (c *ResponsesClient) GetResponse(ctx context.Context, messages []chat.Message, options *chat.Options) (*chat.Response, error) {
	return c.GetResponseWithTools(ctx, messages, nil, options)
}

// GetResponseWithTools sends messages with tools to the Responses API.
// This method supports both function tools and hosted tools (web search, code interpreter).
//
// The Responses API handles hosted tools differently from the Chat Completions API.
// Hosted tools are executed by the provider's infrastructure, not locally.
//
// Example:
//
//	tools := []tool.Tool{
//	    tool.NewHostedWebSearchTool(),
//	    myFunctionTool,
//	}
//	resp, err := client.GetResponseWithTools(ctx, messages, tools, opts)
func (c *ResponsesClient) GetResponseWithTools(ctx context.Context, messages []chat.Message, tools []tool.Tool, options *chat.Options) (*chat.Response, error) {
	req := c.buildResponsesRequest(messages, tools, options)

	// Use previous response ID for conversation continuation
	c.mu.RLock()
	if c.previousResponseID != "" {
		req["previous_response_id"] = c.previousResponseID
	}
	c.mu.RUnlock()

	respBody, err := c.doRequest(ctx, req)
	if err != nil {
		return nil, err
	}

	response, err := c.parseResponsesAPIResponse(respBody)
	if err != nil {
		return nil, err
	}

	return response, nil
}

// GetStreamingResponse returns streaming responses from the Responses API.
// The channel receives ResponseUpdate values as content is generated.
// The channel is closed when generation completes or an error occurs.
//
// Note: The Responses API streaming format differs from the Chat Completions API.
func (c *ResponsesClient) GetStreamingResponse(ctx context.Context, messages []chat.Message, options *chat.Options) (<-chan chat.ResponseUpdate, error) {
	return c.GetStreamingResponseWithTools(ctx, messages, nil, options)
}

// GetStreamingResponseWithTools returns streaming responses with tool support.
// This method supports both function tools and hosted tools.
func (c *ResponsesClient) GetStreamingResponseWithTools(ctx context.Context, messages []chat.Message, tools []tool.Tool, options *chat.Options) (<-chan chat.ResponseUpdate, error) {
	req := c.buildResponsesRequest(messages, tools, options)
	req["stream"] = true

	// Use previous response ID for conversation continuation
	c.mu.RLock()
	if c.previousResponseID != "" {
		req["previous_response_id"] = c.previousResponseID
	}
	c.mu.RUnlock()

	updates := make(chan chat.ResponseUpdate, 32)

	go func() {
		defer close(updates)

		respBody, err := c.doRequest(ctx, req)
		if err != nil {
			updates <- chat.ResponseUpdate{
				Kind:  chat.UpdateKindError,
				Error: err,
			}
			return
		}

		// Parse the streaming response
		c.processStreamingResponse(ctx, respBody, updates)
	}()

	return updates, nil
}

// buildResponsesRequest creates a request body for the Responses API.
func (c *ResponsesClient) buildResponsesRequest(messages []chat.Message, tools []tool.Tool, options *chat.Options) map[string]interface{} {
	model := c.model
	if options != nil && options.ModelID != "" {
		model = options.ModelID
	}

	req := map[string]interface{}{
		"model": model,
		"input": toResponsesInput(messages, c.instructionRole),
	}

	// Add instructions
	instructions := c.instructions
	if options != nil && options.Instructions != "" {
		instructions = options.Instructions
	}
	if instructions != "" {
		req["instructions"] = instructions
	}

	// Add options
	if options != nil {
		if options.MaxTokens > 0 {
			req["max_output_tokens"] = options.MaxTokens
		}
		if options.Temperature != 0 {
			req["temperature"] = options.Temperature
		}
		if options.TopP != 0 {
			req["top_p"] = options.TopP
		}
		if options.User != "" {
			req["user"] = options.User
		}
		if options.Store {
			req["store"] = true
		}
	}

	// Add tools
	if len(tools) > 0 {
		toolConfigs := c.buildToolConfigs(tools)
		if len(toolConfigs) > 0 {
			req["tools"] = toolConfigs
		}
	}

	return req
}

// buildToolConfigs creates tool configurations for the Responses API.
func (c *ResponsesClient) buildToolConfigs(tools []tool.Tool) []map[string]interface{} {
	var configs []map[string]interface{}

	for _, t := range tools {
		if ht, ok := t.(tool.HostedTool); ok && ht.IsHosted() {
			// Use the provider config directly for hosted tools
			configs = append(configs, ht.ProviderConfig())
		} else {
			// Convert function tools to Responses API format
			var params interface{}
			if t.Parameters() != nil {
				if err := json.Unmarshal(t.Parameters(), &params); err != nil {
					params = nil
				}
			}

			config := map[string]interface{}{
				"type": "function",
				"function": map[string]interface{}{
					"name":        t.Name(),
					"description": t.Description(),
				},
			}
			if params != nil {
				config["function"].(map[string]interface{})["parameters"] = params
			}
			configs = append(configs, config)
		}
	}

	return configs
}

// toResponsesInput converts chat messages to Responses API input format.
func toResponsesInput(messages []chat.Message, instructionRole string) []map[string]interface{} {
	input := make([]map[string]interface{}, 0, len(messages))

	for _, msg := range messages {
		item := map[string]interface{}{
			"role": convertRole(msg.Role, instructionRole),
		}

		// Handle content
		if len(msg.Contents) == 1 {
			if tc, ok := msg.Contents[0].(*chat.TextContent); ok {
				item["content"] = tc.Text
			} else {
				item["content"] = toResponsesContentArray(msg.Contents)
			}
		} else if len(msg.Contents) > 0 {
			item["content"] = toResponsesContentArray(msg.Contents)
		}

		// Handle tool calls
		if len(msg.ToolCalls) > 0 {
			toolCalls := make([]map[string]interface{}, len(msg.ToolCalls))
			for i, tc := range msg.ToolCalls {
				toolCalls[i] = map[string]interface{}{
					"id":   tc.ID,
					"type": "function",
					"function": map[string]interface{}{
						"name":      tc.Name,
						"arguments": string(tc.Arguments),
					},
				}
			}
			item["tool_calls"] = toolCalls
		}

		// Handle tool result
		if msg.Role == chat.RoleTool && msg.ToolCallID != "" {
			item["tool_call_id"] = msg.ToolCallID
		}

		input = append(input, item)
	}

	return input
}

// convertRole converts chat.Role to Responses API role string.
func convertRole(role chat.Role, instructionRole string) string {
	switch role {
	case chat.RoleSystem:
		if instructionRole == "developer" {
			return "developer"
		}
		return "system"
	case chat.RoleUser:
		return "user"
	case chat.RoleAssistant:
		return "assistant"
	case chat.RoleTool:
		return "tool"
	default:
		return string(role)
	}
}

// toResponsesContentArray converts content items to Responses API format.
func toResponsesContentArray(contents []chat.Content) []map[string]interface{} {
	items := make([]map[string]interface{}, 0, len(contents))

	for _, c := range contents {
		switch content := c.(type) {
		case *chat.TextContent:
			items = append(items, map[string]interface{}{
				"type": "text",
				"text": content.Text,
			})
		case *chat.ImageContent:
			if content.URL != "" {
				items = append(items, map[string]interface{}{
					"type": "image_url",
					"image_url": map[string]interface{}{
						"url":    content.URL,
						"detail": content.Detail,
					},
				})
			} else if content.Base64Data != "" {
				dataURL := "data:" + content.MediaType + ";base64," + content.Base64Data
				items = append(items, map[string]interface{}{
					"type": "image_url",
					"image_url": map[string]interface{}{
						"url":    dataURL,
						"detail": content.Detail,
					},
				})
			}
		}
	}

	return items
}

// doRequest performs an HTTP request to the Responses API.
func (c *ResponsesClient) doRequest(ctx context.Context, body map[string]interface{}) ([]byte, error) {
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("openai: failed to marshal request: %w", err)
	}

	url := c.baseURL + ResponsesAPIPath
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("openai: failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	if c.orgID != "" {
		req.Header.Set("OpenAI-Organization", c.orgID)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("openai: request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("openai: failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, c.parseAPIError(resp.StatusCode, respBody)
	}

	return respBody, nil
}

// parseAPIError parses an API error response.
func (c *ResponsesClient) parseAPIError(statusCode int, body []byte) error {
	var errorResp struct {
		Error struct {
			Message string `json:"message"`
			Type    string `json:"type"`
			Code    string `json:"code"`
		} `json:"error"`
	}

	if err := json.Unmarshal(body, &errorResp); err != nil {
		return fmt.Errorf("openai: API error (status %d): %s", statusCode, string(body))
	}

	return fmt.Errorf("openai: API error (%s): %s", errorResp.Error.Code, errorResp.Error.Message)
}

// parseResponsesAPIResponse parses a Responses API response.
func (c *ResponsesClient) parseResponsesAPIResponse(body []byte) (*chat.Response, error) {
	var resp responsesAPIResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("openai: failed to parse response: %w", err)
	}

	// Store response ID for conversation continuation
	if resp.ID != "" {
		c.mu.Lock()
		c.previousResponseID = resp.ID
		c.mu.Unlock()
	}

	message := c.convertResponseToMessage(resp)
	finishReason := c.convertFinishReason(resp.Status)

	return &chat.Response{
		Message:      message,
		FinishReason: finishReason,
		Usage: &chat.UsageDetails{
			InputTokens:  resp.Usage.InputTokens,
			OutputTokens: resp.Usage.OutputTokens,
			TotalTokens:  resp.Usage.TotalTokens,
		},
		RawRepresentation: resp,
	}, nil
}

// responsesAPIResponse represents the Responses API response structure.
type responsesAPIResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created_at"`
	Model   string `json:"model"`
	Status  string `json:"status"`
	Output  []struct {
		Type    string `json:"type"`
		ID      string `json:"id,omitempty"`
		Role    string `json:"role,omitempty"`
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text,omitempty"`
		} `json:"content,omitempty"`
		Name      string          `json:"name,omitempty"`
		Arguments json.RawMessage `json:"arguments,omitempty"`
		CallID    string          `json:"call_id,omitempty"`
		Output    string          `json:"output,omitempty"`
	} `json:"output"`
	Usage struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
		TotalTokens  int `json:"total_tokens"`
	} `json:"usage"`
}

// convertResponseToMessage converts a Responses API response to a chat.Message.
func (c *ResponsesClient) convertResponseToMessage(resp responsesAPIResponse) chat.Message {
	msg := chat.Message{
		Role: chat.RoleAssistant,
	}

	var textBuilder strings.Builder
	var toolCalls []chat.ToolCall

	for _, output := range resp.Output {
		switch output.Type {
		case "message":
			for _, content := range output.Content {
				if content.Type == "text" {
					textBuilder.WriteString(content.Text)
				}
			}
		case "function_call":
			toolCalls = append(toolCalls, chat.ToolCall{
				ID:        output.CallID,
				Name:      output.Name,
				Arguments: output.Arguments,
			})
		case "web_search_call", "code_interpreter_call", "file_search_call":
			// These are hosted tool calls - include their results
			if output.Output != "" {
				textBuilder.WriteString(output.Output)
			}
		}
	}

	if textBuilder.Len() > 0 {
		msg.Contents = []chat.Content{chat.NewTextContent(textBuilder.String())}
	}

	if len(toolCalls) > 0 {
		msg.ToolCalls = toolCalls
	}

	return msg
}

// convertFinishReason converts a Responses API status to chat.FinishReason.
func (c *ResponsesClient) convertFinishReason(status string) chat.FinishReason {
	switch status {
	case "completed":
		return chat.FinishReasonStop
	case "incomplete":
		return chat.FinishReasonLength
	case "cancelled":
		return chat.FinishReasonStop
	case "failed":
		return chat.FinishReasonStop
	default:
		return chat.FinishReasonStop
	}
}

// processStreamingResponse processes a streaming response from the Responses API.
func (c *ResponsesClient) processStreamingResponse(ctx context.Context, body []byte, updates chan<- chat.ResponseUpdate) {
	// For non-streaming responses that were requested with stream=true,
	// parse the complete response and emit updates
	var resp responsesAPIResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		updates <- chat.ResponseUpdate{
			Kind:  chat.UpdateKindError,
			Error: fmt.Errorf("openai: failed to parse streaming response: %w", err),
		}
		return
	}

	// Store response ID
	if resp.ID != "" {
		c.mu.Lock()
		c.previousResponseID = resp.ID
		c.mu.Unlock()
	}

	// Emit content updates
	for _, output := range resp.Output {
		switch output.Type {
		case "message":
			for _, content := range output.Content {
				if content.Type == "text" {
					updates <- chat.ResponseUpdate{
						Kind: chat.UpdateKindContentDelta,
						Delta: &chat.ContentDelta{
							Role:      chat.RoleAssistant,
							TextDelta: content.Text,
						},
					}
				}
			}
		case "function_call":
			updates <- chat.ResponseUpdate{
				Kind: chat.UpdateKindToolCall,
				Delta: &chat.ContentDelta{
					ToolCallID: output.CallID,
					Name:       output.Name,
				},
				Metadata: map[string]interface{}{
					"tool_call_id": output.CallID,
					"tool_name":    output.Name,
					"arguments":    string(output.Arguments),
				},
			}
		}
	}

	// Emit usage if available
	if resp.Usage.TotalTokens > 0 {
		updates <- chat.ResponseUpdate{
			Kind: chat.UpdateKindUsage,
			Usage: &chat.UsageDetails{
				InputTokens:  resp.Usage.InputTokens,
				OutputTokens: resp.Usage.OutputTokens,
				TotalTokens:  resp.Usage.TotalTokens,
			},
		}
	}

	// Emit completion
	updates <- chat.ResponseUpdate{
		Kind:         chat.UpdateKindMessageComplete,
		FinishReason: c.convertFinishReason(resp.Status),
	}

	updates <- chat.ResponseUpdate{
		Kind: chat.UpdateKindDone,
	}
}
