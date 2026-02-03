// Copyright (c) Microsoft. All rights reserved.

package openai

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/microsoft/agent-framework-go/chat"
	"github.com/microsoft/agent-framework-go/tool"
)

func TestNewResponsesClient(t *testing.T) {
	t.Run("creates client with API key", func(t *testing.T) {
		client, err := NewResponsesClient(ResponsesWithAPIKey("test-key"))
		if err != nil {
			t.Fatalf("NewResponsesClient() error = %v", err)
		}
		if client == nil {
			t.Fatal("NewResponsesClient() returned nil")
		}
	})

	t.Run("returns error without API key", func(t *testing.T) {
		// Temporarily unset environment variable
		t.Setenv("OPENAI_API_KEY", "")

		_, err := NewResponsesClient()
		if err == nil {
			t.Fatal("NewResponsesClient() expected error without API key")
		}
	})

	t.Run("applies all options", func(t *testing.T) {
		httpClient := &http.Client{}
		client, err := NewResponsesClient(
			ResponsesWithAPIKey("test-key"),
			ResponsesWithModel("gpt-4o-mini"),
			ResponsesWithBaseURL("https://custom.api.com"),
			ResponsesWithOrgID("org-123"),
			ResponsesWithInstructionRole("developer"),
			ResponsesWithHTTPClient(httpClient),
			ResponsesWithInstructions("You are helpful."),
		)
		if err != nil {
			t.Fatalf("NewResponsesClient() error = %v", err)
		}

		metadata := client.Metadata()
		if metadata.ModelID != "gpt-4o-mini" {
			t.Errorf("ModelID = %q, want %q", metadata.ModelID, "gpt-4o-mini")
		}
		if metadata.EndpointURI != "https://custom.api.com" {
			t.Errorf("EndpointURI = %q, want %q", metadata.EndpointURI, "https://custom.api.com")
		}
		if metadata.ProviderName != "openai-responses" {
			t.Errorf("ProviderName = %q, want %q", metadata.ProviderName, "openai-responses")
		}
	})
}

func TestResponsesClient_Metadata(t *testing.T) {
	client, err := NewResponsesClient(ResponsesWithAPIKey("test-key"))
	if err != nil {
		t.Fatalf("NewResponsesClient() error = %v", err)
	}

	metadata := client.Metadata()

	if metadata.ProviderName != "openai-responses" {
		t.Errorf("ProviderName = %q, want %q", metadata.ProviderName, "openai-responses")
	}
	if metadata.ModelID != ResponsesDefaultModel {
		t.Errorf("ModelID = %q, want %q", metadata.ModelID, ResponsesDefaultModel)
	}
	if metadata.EndpointURI != DefaultBaseURL {
		t.Errorf("EndpointURI = %q, want %q", metadata.EndpointURI, DefaultBaseURL)
	}
}

func TestResponsesClient_ResponseIDManagement(t *testing.T) {
	client, err := NewResponsesClient(ResponsesWithAPIKey("test-key"))
	if err != nil {
		t.Fatalf("NewResponsesClient() error = %v", err)
	}

	t.Run("initially empty", func(t *testing.T) {
		if id := client.GetPreviousResponseID(); id != "" {
			t.Errorf("GetPreviousResponseID() = %q, want empty", id)
		}
	})

	t.Run("set and get", func(t *testing.T) {
		client.SetPreviousResponseID("resp_123")
		if id := client.GetPreviousResponseID(); id != "resp_123" {
			t.Errorf("GetPreviousResponseID() = %q, want %q", id, "resp_123")
		}
	})

	t.Run("clear conversation", func(t *testing.T) {
		client.SetPreviousResponseID("resp_123")
		client.ClearConversation()
		if id := client.GetPreviousResponseID(); id != "" {
			t.Errorf("GetPreviousResponseID() after clear = %q, want empty", id)
		}
	})
}

func TestResponsesClient_GetResponse(t *testing.T) {
	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request
		if r.Method != http.MethodPost {
			t.Errorf("Method = %q, want POST", r.Method)
		}
		if r.URL.Path != ResponsesAPIPath {
			t.Errorf("Path = %q, want %q", r.URL.Path, ResponsesAPIPath)
		}
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Errorf("Authorization header missing or incorrect")
		}

		// Send mock response
		resp := responsesAPIResponse{
			ID:      "resp_001",
			Object:  "realtime.response",
			Created: 1234567890,
			Model:   "gpt-4o",
			Status:  "completed",
			Output: []struct {
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
			}{
				{
					Type: "message",
					Role: "assistant",
					Content: []struct {
						Type string `json:"type"`
						Text string `json:"text,omitempty"`
					}{
						{Type: "text", Text: "Hello! How can I help you?"},
					},
				},
			},
			Usage: struct {
				InputTokens  int `json:"input_tokens"`
				OutputTokens int `json:"output_tokens"`
				TotalTokens  int `json:"total_tokens"`
			}{
				InputTokens:  10,
				OutputTokens: 8,
				TotalTokens:  18,
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client, err := NewResponsesClient(
		ResponsesWithAPIKey("test-key"),
		ResponsesWithBaseURL(server.URL),
	)
	if err != nil {
		t.Fatalf("NewResponsesClient() error = %v", err)
	}

	messages := []chat.Message{
		chat.NewUserMessage("Hello!"),
	}

	resp, err := client.GetResponse(context.Background(), messages, nil)
	if err != nil {
		t.Fatalf("GetResponse() error = %v", err)
	}

	// Verify response
	if resp.FinishReason != chat.FinishReasonStop {
		t.Errorf("FinishReason = %v, want %v", resp.FinishReason, chat.FinishReasonStop)
	}

	if len(resp.Message.Contents) == 0 {
		t.Fatal("Message.Contents is empty")
	}

	tc, ok := resp.Message.Contents[0].(*chat.TextContent)
	if !ok {
		t.Fatalf("Contents[0] type = %T, want *chat.TextContent", resp.Message.Contents[0])
	}
	if tc.Text != "Hello! How can I help you?" {
		t.Errorf("Text = %q, want %q", tc.Text, "Hello! How can I help you?")
	}

	// Verify usage
	if resp.Usage.InputTokens != 10 {
		t.Errorf("InputTokens = %d, want %d", resp.Usage.InputTokens, 10)
	}
	if resp.Usage.OutputTokens != 8 {
		t.Errorf("OutputTokens = %d, want %d", resp.Usage.OutputTokens, 8)
	}

	// Verify response ID stored
	if client.GetPreviousResponseID() != "resp_001" {
		t.Errorf("PreviousResponseID = %q, want %q", client.GetPreviousResponseID(), "resp_001")
	}
}

func TestResponsesClient_GetResponseWithTools(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var reqBody map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
			t.Errorf("Failed to decode request body: %v", err)
		}

		// Verify tools are included
		tools, ok := reqBody["tools"].([]interface{})
		if !ok {
			t.Error("tools not found in request")
		}
		if len(tools) != 2 {
			t.Errorf("len(tools) = %d, want 2", len(tools))
		}

		// Verify instructions
		if instructions, ok := reqBody["instructions"].(string); !ok || instructions != "Be helpful" {
			t.Errorf("instructions = %v, want %q", reqBody["instructions"], "Be helpful")
		}

		// Send response with function call
		resp := map[string]interface{}{
			"id":      "resp_002",
			"object":  "realtime.response",
			"status":  "completed",
			"model":   "gpt-4o",
			"output": []map[string]interface{}{
				{
					"type":      "function_call",
					"call_id":   "call_123",
					"name":      "get_weather",
					"arguments": `{"location":"Seattle"}`,
				},
			},
			"usage": map[string]int{
				"input_tokens":  15,
				"output_tokens": 12,
				"total_tokens":  27,
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client, err := NewResponsesClient(
		ResponsesWithAPIKey("test-key"),
		ResponsesWithBaseURL(server.URL),
	)
	if err != nil {
		t.Fatalf("NewResponsesClient() error = %v", err)
	}

	// Create tools - one hosted, one function
	webSearch := tool.NewHostedWebSearchTool()
	funcTool := createMockFunctionTool("get_weather", "Get weather for a location")

	tools := []tool.Tool{webSearch, funcTool}
	messages := []chat.Message{
		chat.NewUserMessage("What's the weather in Seattle?"),
	}
	opts := &chat.Options{Instructions: "Be helpful"}

	resp, err := client.GetResponseWithTools(context.Background(), messages, tools, opts)
	if err != nil {
		t.Fatalf("GetResponseWithTools() error = %v", err)
	}

	// Verify tool call in response
	if len(resp.Message.ToolCalls) == 0 {
		t.Fatal("ToolCalls is empty")
	}

	tc := resp.Message.ToolCalls[0]
	if tc.Name != "get_weather" {
		t.Errorf("ToolCall.Name = %q, want %q", tc.Name, "get_weather")
	}
	if tc.ID != "call_123" {
		t.Errorf("ToolCall.ID = %q, want %q", tc.ID, "call_123")
	}
}

func TestResponsesClient_ConversationContinuation(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var reqBody map[string]interface{}
		json.NewDecoder(r.Body).Decode(&reqBody)

		callCount++
		responseID := "resp_" + string(rune('0'+callCount))

		if callCount == 2 {
			// Second call should have previous_response_id
			prevID, ok := reqBody["previous_response_id"].(string)
			if !ok || prevID != "resp_1" {
				t.Errorf("previous_response_id = %v, want %q", reqBody["previous_response_id"], "resp_1")
			}
		}

		resp := map[string]interface{}{
			"id":     responseID,
			"status": "completed",
			"output": []map[string]interface{}{
				{
					"type": "message",
					"content": []map[string]interface{}{
						{"type": "text", "text": "Response " + string(rune('0'+callCount))},
					},
				},
			},
			"usage": map[string]int{"total_tokens": 10},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client, err := NewResponsesClient(
		ResponsesWithAPIKey("test-key"),
		ResponsesWithBaseURL(server.URL),
	)
	if err != nil {
		t.Fatalf("NewResponsesClient() error = %v", err)
	}

	// First request
	messages := []chat.Message{chat.NewUserMessage("Hello")}
	_, err = client.GetResponse(context.Background(), messages, nil)
	if err != nil {
		t.Fatalf("First GetResponse() error = %v", err)
	}

	// Second request should include previous response ID
	_, err = client.GetResponse(context.Background(), messages, nil)
	if err != nil {
		t.Fatalf("Second GetResponse() error = %v", err)
	}

	if callCount != 2 {
		t.Errorf("callCount = %d, want 2", callCount)
	}
}

func TestResponsesClient_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": map[string]interface{}{
				"message": "Invalid request",
				"type":    "invalid_request_error",
				"code":    "invalid_api_key",
			},
		})
	}))
	defer server.Close()

	client, err := NewResponsesClient(
		ResponsesWithAPIKey("bad-key"),
		ResponsesWithBaseURL(server.URL),
	)
	if err != nil {
		t.Fatalf("NewResponsesClient() error = %v", err)
	}

	messages := []chat.Message{chat.NewUserMessage("Hello")}
	_, err = client.GetResponse(context.Background(), messages, nil)
	if err == nil {
		t.Fatal("GetResponse() expected error for bad API key")
	}
}

func TestResponsesClient_GetStreamingResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var reqBody map[string]interface{}
		json.NewDecoder(r.Body).Decode(&reqBody)

		// Verify stream is requested
		if stream, ok := reqBody["stream"].(bool); !ok || !stream {
			t.Errorf("stream = %v, want true", reqBody["stream"])
		}

		// Return a complete response (streaming would be SSE in real API)
		resp := map[string]interface{}{
			"id":     "resp_stream",
			"status": "completed",
			"output": []map[string]interface{}{
				{
					"type": "message",
					"content": []map[string]interface{}{
						{"type": "text", "text": "Streamed response"},
					},
				},
			},
			"usage": map[string]int{
				"input_tokens":  5,
				"output_tokens": 3,
				"total_tokens":  8,
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client, err := NewResponsesClient(
		ResponsesWithAPIKey("test-key"),
		ResponsesWithBaseURL(server.URL),
	)
	if err != nil {
		t.Fatalf("NewResponsesClient() error = %v", err)
	}

	messages := []chat.Message{chat.NewUserMessage("Hello")}
	updates, err := client.GetStreamingResponse(context.Background(), messages, nil)
	if err != nil {
		t.Fatalf("GetStreamingResponse() error = %v", err)
	}

	var gotContent bool
	var gotDone bool

	for update := range updates {
		switch update.Kind {
		case chat.UpdateKindContentDelta:
			gotContent = true
			if update.Delta.TextDelta != "Streamed response" {
				t.Errorf("TextDelta = %q, want %q", update.Delta.TextDelta, "Streamed response")
			}
		case chat.UpdateKindDone:
			gotDone = true
		case chat.UpdateKindError:
			t.Errorf("Unexpected error: %v", update.Error)
		}
	}

	if !gotContent {
		t.Error("Did not receive content update")
	}
	if !gotDone {
		t.Error("Did not receive done update")
	}
}

func TestBuildResponsesRequest(t *testing.T) {
	client, err := NewResponsesClient(
		ResponsesWithAPIKey("test-key"),
		ResponsesWithModel("gpt-4o"),
		ResponsesWithInstructions("Default instructions"),
	)
	if err != nil {
		t.Fatalf("NewResponsesClient() error = %v", err)
	}

	messages := []chat.Message{
		chat.NewUserMessage("Hello"),
	}

	t.Run("basic request", func(t *testing.T) {
		req := client.buildResponsesRequest(messages, nil, nil)

		if req["model"] != "gpt-4o" {
			t.Errorf("model = %v, want %q", req["model"], "gpt-4o")
		}
		if req["instructions"] != "Default instructions" {
			t.Errorf("instructions = %v, want %q", req["instructions"], "Default instructions")
		}
	})

	t.Run("with options override", func(t *testing.T) {
		opts := &chat.Options{
			ModelID:      "gpt-4o-mini",
			Instructions: "Override instructions",
			MaxTokens:    100,
			Temperature:  0.7,
		}
		req := client.buildResponsesRequest(messages, nil, opts)

		if req["model"] != "gpt-4o-mini" {
			t.Errorf("model = %v, want %q", req["model"], "gpt-4o-mini")
		}
		if req["instructions"] != "Override instructions" {
			t.Errorf("instructions = %v, want %q", req["instructions"], "Override instructions")
		}
		if req["max_output_tokens"] != 100 {
			t.Errorf("max_output_tokens = %v, want %d", req["max_output_tokens"], 100)
		}
	})

	t.Run("with tools", func(t *testing.T) {
		webSearch := tool.NewHostedWebSearchTool()
		tools := []tool.Tool{webSearch}

		req := client.buildResponsesRequest(messages, tools, nil)

		toolConfigs, ok := req["tools"].([]map[string]interface{})
		if !ok {
			t.Fatal("tools not present or wrong type")
		}
		if len(toolConfigs) != 1 {
			t.Errorf("len(tools) = %d, want 1", len(toolConfigs))
		}
		if toolConfigs[0]["type"] != "web_search" {
			t.Errorf("tool type = %v, want %q", toolConfigs[0]["type"], "web_search")
		}
	})
}

func TestToResponsesInput(t *testing.T) {
	t.Run("single text message", func(t *testing.T) {
		messages := []chat.Message{
			chat.NewUserMessage("Hello"),
		}

		input := toResponsesInput(messages, "system")

		if len(input) != 1 {
			t.Fatalf("len(input) = %d, want 1", len(input))
		}
		if input[0]["role"] != "user" {
			t.Errorf("role = %v, want %q", input[0]["role"], "user")
		}
		if input[0]["content"] != "Hello" {
			t.Errorf("content = %v, want %q", input[0]["content"], "Hello")
		}
	})

	t.Run("developer role instruction", func(t *testing.T) {
		messages := []chat.Message{
			{Role: chat.RoleSystem, Contents: []chat.Content{chat.NewTextContent("Be helpful")}},
		}

		input := toResponsesInput(messages, "developer")

		if input[0]["role"] != "developer" {
			t.Errorf("role = %v, want %q", input[0]["role"], "developer")
		}
	})

	t.Run("message with tool calls", func(t *testing.T) {
		messages := []chat.Message{
			{
				Role: chat.RoleAssistant,
				ToolCalls: []chat.ToolCall{
					{ID: "call_1", Name: "get_weather", Arguments: json.RawMessage(`{"city":"Seattle"}`)},
				},
			},
		}

		input := toResponsesInput(messages, "system")

		toolCalls, ok := input[0]["tool_calls"].([]map[string]interface{})
		if !ok {
			t.Fatal("tool_calls not present or wrong type")
		}
		if len(toolCalls) != 1 {
			t.Fatalf("len(tool_calls) = %d, want 1", len(toolCalls))
		}
		if toolCalls[0]["id"] != "call_1" {
			t.Errorf("id = %v, want %q", toolCalls[0]["id"], "call_1")
		}
	})
}

func TestConvertRole(t *testing.T) {
	tests := []struct {
		role            chat.Role
		instructionRole string
		want            string
	}{
		{chat.RoleSystem, "system", "system"},
		{chat.RoleSystem, "developer", "developer"},
		{chat.RoleUser, "system", "user"},
		{chat.RoleAssistant, "system", "assistant"},
		{chat.RoleTool, "system", "tool"},
	}

	for _, tt := range tests {
		t.Run(string(tt.role)+"_"+tt.instructionRole, func(t *testing.T) {
			got := convertRole(tt.role, tt.instructionRole)
			if got != tt.want {
				t.Errorf("convertRole() = %q, want %q", got, tt.want)
			}
		})
	}
}

// mockFunctionTool is a simple function tool for testing.
type mockFunctionTool struct {
	name        string
	description string
}

func createMockFunctionTool(name, description string) *mockFunctionTool {
	return &mockFunctionTool{name: name, description: description}
}

func (t *mockFunctionTool) Name() string           { return t.name }
func (t *mockFunctionTool) Description() string    { return t.description }
func (t *mockFunctionTool) Parameters() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{"location":{"type":"string"}}}`)
}
func (t *mockFunctionTool) Invoke(ctx context.Context, args json.RawMessage) (tool.Result, error) {
	return tool.Result{Content: "result"}, nil
}
