// Copyright (c) Microsoft. All rights reserved.

package openai

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/microsoft/agent-framework-go/chat"
	"github.com/microsoft/agent-framework-go/tool"
)

func TestNewClient(t *testing.T) {
	t.Run("creates client with API key", func(t *testing.T) {
		client, err := NewClient(WithAPIKey("test-key"))
		if err != nil {
			t.Fatalf("NewClient() error = %v", err)
		}
		if client == nil {
			t.Fatal("NewClient() returned nil")
		}
	})

	t.Run("returns error without API key", func(t *testing.T) {
		// Temporarily unset environment variable
		t.Setenv("OPENAI_API_KEY", "")
		t.Setenv("OPENAI_BASE_URL", "")
		t.Setenv("OPENAI_ORG_ID", "")

		_, err := NewClient()
		if err == nil {
			t.Fatal("NewClient() expected error without API key")
		}
		if !strings.Contains(err.Error(), "API key required") {
			t.Errorf("error message should mention API key, got: %v", err)
		}
	})

	t.Run("uses environment variable for API key", func(t *testing.T) {
		t.Setenv("OPENAI_API_KEY", "env-test-key")

		client, err := NewClient()
		if err != nil {
			t.Fatalf("NewClient() error = %v", err)
		}
		if client == nil {
			t.Fatal("NewClient() returned nil")
		}
	})

	t.Run("applies all options", func(t *testing.T) {
		httpClient := &http.Client{}
		client, err := NewClient(
			WithAPIKey("test-key"),
			WithModel("gpt-4o-mini"),
			WithBaseURL("https://custom.api.com"),
			WithOrgID("org-123"),
			WithInstructionRole("developer"),
			WithHTTPClient(httpClient),
		)
		if err != nil {
			t.Fatalf("NewClient() error = %v", err)
		}

		metadata := client.Metadata()
		if metadata.ModelID != "gpt-4o-mini" {
			t.Errorf("ModelID = %q, want %q", metadata.ModelID, "gpt-4o-mini")
		}
		if metadata.EndpointURI != "https://custom.api.com" {
			t.Errorf("EndpointURI = %q, want %q", metadata.EndpointURI, "https://custom.api.com")
		}
		if metadata.ProviderName != "openai" {
			t.Errorf("ProviderName = %q, want %q", metadata.ProviderName, "openai")
		}
	})

	t.Run("uses default model", func(t *testing.T) {
		client, err := NewClient(WithAPIKey("test-key"))
		if err != nil {
			t.Fatalf("NewClient() error = %v", err)
		}

		metadata := client.Metadata()
		if metadata.ModelID != DefaultModel {
			t.Errorf("ModelID = %q, want default %q", metadata.ModelID, DefaultModel)
		}
	})

	t.Run("uses environment base URL", func(t *testing.T) {
		t.Setenv("OPENAI_API_KEY", "test-key")
		t.Setenv("OPENAI_BASE_URL", "https://env-custom.api.com")

		client, err := NewClient()
		if err != nil {
			t.Fatalf("NewClient() error = %v", err)
		}

		metadata := client.Metadata()
		if metadata.EndpointURI != "https://env-custom.api.com" {
			t.Errorf("EndpointURI = %q, want %q", metadata.EndpointURI, "https://env-custom.api.com")
		}
	})

	t.Run("option overrides environment variable", func(t *testing.T) {
		t.Setenv("OPENAI_API_KEY", "env-key")
		t.Setenv("OPENAI_BASE_URL", "https://env.api.com")

		client, err := NewClient(
			WithAPIKey("option-key"),
			WithBaseURL("https://option.api.com"),
		)
		if err != nil {
			t.Fatalf("NewClient() error = %v", err)
		}

		metadata := client.Metadata()
		if metadata.EndpointURI != "https://option.api.com" {
			t.Errorf("EndpointURI = %q, want %q from option override", metadata.EndpointURI, "https://option.api.com")
		}
	})
}

func TestNewClientWithHTTPClient(t *testing.T) {
	t.Run("creates client with custom HTTP client", func(t *testing.T) {
		httpClient := &http.Client{
			Timeout: 30 * time.Second,
		}
		client, err := NewClientWithHTTPClient(httpClient, WithAPIKey("test-key"))
		if err != nil {
			t.Fatalf("NewClientWithHTTPClient() error = %v", err)
		}
		if client == nil {
			t.Fatal("NewClientWithHTTPClient() returned nil")
		}
	})
}

func TestNewClientWithTimeout(t *testing.T) {
	t.Run("creates client with custom timeout", func(t *testing.T) {
		client, err := NewClientWithTimeout(30*time.Second, WithAPIKey("test-key"))
		if err != nil {
			t.Fatalf("NewClientWithTimeout() error = %v", err)
		}
		if client == nil {
			t.Fatal("NewClientWithTimeout() returned nil")
		}
	})
}

func TestClient_Metadata(t *testing.T) {
	client, err := NewClient(WithAPIKey("test-key"))
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	metadata := client.Metadata()

	if metadata.ProviderName != "openai" {
		t.Errorf("ProviderName = %q, want %q", metadata.ProviderName, "openai")
	}
	if metadata.ModelID != DefaultModel {
		t.Errorf("ModelID = %q, want %q", metadata.ModelID, DefaultModel)
	}
	if metadata.EndpointURI == "" {
		t.Error("EndpointURI should not be empty")
	}
}

func TestClient_GetResponse(t *testing.T) {
	t.Run("successful response", func(t *testing.T) {
		// Create mock server
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				t.Errorf("Method = %q, want POST", r.Method)
			}
			if !strings.HasSuffix(r.URL.Path, "/chat/completions") {
				t.Errorf("Path = %q, want suffix '/chat/completions'", r.URL.Path)
			}
			if r.Header.Get("Authorization") != "Bearer test-key" {
				t.Errorf("Authorization header missing or incorrect")
			}

			resp := `{
				"id": "chatcmpl-123",
				"object": "chat.completion",
				"created": 1677652288,
				"model": "gpt-4o",
				"choices": [{
					"index": 0,
					"message": {
						"role": "assistant",
						"content": "Hello! How can I help you today?"
					},
					"finish_reason": "stop"
				}],
				"usage": {
					"prompt_tokens": 9,
					"completion_tokens": 12,
					"total_tokens": 21
				}
			}`
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(resp))
		}))
		defer server.Close()

		client, err := NewClient(
			WithAPIKey("test-key"),
			WithBaseURL(server.URL),
		)
		if err != nil {
			t.Fatalf("NewClient() error = %v", err)
		}

		messages := []chat.Message{
			{Role: chat.RoleUser, Contents: []chat.Content{chat.NewTextContent("Hello")}},
		}

		resp, err := client.GetResponse(context.Background(), messages, nil)
		if err != nil {
			t.Fatalf("GetResponse() error = %v", err)
		}

		if resp.Message.Text() != "Hello! How can I help you today?" {
			t.Errorf("Message.Text() = %q, want %q", resp.Message.Text(), "Hello! How can I help you today?")
		}
		if resp.FinishReason != chat.FinishReasonStop {
			t.Errorf("FinishReason = %v, want %v", resp.FinishReason, chat.FinishReasonStop)
		}
		if resp.Usage == nil {
			t.Fatal("Usage should not be nil")
		}
		if resp.Usage.InputTokens != 9 {
			t.Errorf("Usage.InputTokens = %d, want %d", resp.Usage.InputTokens, 9)
		}
		if resp.Usage.OutputTokens != 12 {
			t.Errorf("Usage.OutputTokens = %d, want %d", resp.Usage.OutputTokens, 12)
		}
		if resp.Usage.TotalTokens != 21 {
			t.Errorf("Usage.TotalTokens = %d, want %d", resp.Usage.TotalTokens, 21)
		}
	})

	t.Run("response with tool calls", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			resp := `{
				"id": "chatcmpl-123",
				"object": "chat.completion",
				"created": 1677652288,
				"model": "gpt-4o",
				"choices": [{
					"index": 0,
					"message": {
						"role": "assistant",
						"content": null,
						"tool_calls": [{
							"id": "call_abc123",
							"type": "function",
							"function": {
								"name": "get_weather",
								"arguments": "{\"location\":\"New York\"}"
							}
						}]
					},
					"finish_reason": "tool_calls"
				}],
				"usage": {
					"prompt_tokens": 50,
					"completion_tokens": 25,
					"total_tokens": 75
				}
			}`
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(resp))
		}))
		defer server.Close()

		client, err := NewClient(
			WithAPIKey("test-key"),
			WithBaseURL(server.URL),
		)
		if err != nil {
			t.Fatalf("NewClient() error = %v", err)
		}

		messages := []chat.Message{
			{Role: chat.RoleUser, Contents: []chat.Content{chat.NewTextContent("What's the weather in NYC?")}},
		}

		resp, err := client.GetResponse(context.Background(), messages, nil)
		if err != nil {
			t.Fatalf("GetResponse() error = %v", err)
		}

		if resp.FinishReason != chat.FinishReasonToolCalls {
			t.Errorf("FinishReason = %v, want %v", resp.FinishReason, chat.FinishReasonToolCalls)
		}
		if len(resp.Message.ToolCalls) != 1 {
			t.Fatalf("len(ToolCalls) = %d, want 1", len(resp.Message.ToolCalls))
		}
		tc := resp.Message.ToolCalls[0]
		if tc.ID != "call_abc123" {
			t.Errorf("ToolCall.ID = %q, want %q", tc.ID, "call_abc123")
		}
		if tc.Name != "get_weather" {
			t.Errorf("ToolCall.Name = %q, want %q", tc.Name, "get_weather")
		}
		if string(tc.Arguments) != `{"location":"New York"}` {
			t.Errorf("ToolCall.Arguments = %q, want %q", string(tc.Arguments), `{"location":"New York"}`)
		}
	})

	t.Run("applies options", func(t *testing.T) {
		var requestBody map[string]interface{}
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			decoder := json.NewDecoder(r.Body)
			if err := decoder.Decode(&requestBody); err != nil {
				t.Errorf("failed to decode request body: %v", err)
			}

			resp := `{
				"id": "chatcmpl-123",
				"object": "chat.completion",
				"created": 1677652288,
				"model": "gpt-4o-mini",
				"choices": [{
					"index": 0,
					"message": {"role": "assistant", "content": "Response"},
					"finish_reason": "stop"
				}],
				"usage": {"prompt_tokens": 5, "completion_tokens": 5, "total_tokens": 10}
			}`
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(resp))
		}))
		defer server.Close()

		client, err := NewClient(
			WithAPIKey("test-key"),
			WithBaseURL(server.URL),
		)
		if err != nil {
			t.Fatalf("NewClient() error = %v", err)
		}

		seed := 42
		options := &chat.Options{
			MaxTokens:        100,
			Temperature:      0.7,
			TopP:             0.9,
			StopSequences:    []string{"\n\n"},
			FrequencyPenalty: 0.5,
			PresencePenalty:  0.3,
			Seed:             &seed,
			User:             "user-123",
			ModelID:          "gpt-4o-mini",
		}

		messages := []chat.Message{
			{Role: chat.RoleUser, Contents: []chat.Content{chat.NewTextContent("Test")}},
		}

		_, err = client.GetResponse(context.Background(), messages, options)
		if err != nil {
			t.Fatalf("GetResponse() error = %v", err)
		}

		// Verify request parameters
		if model, ok := requestBody["model"].(string); !ok || model != "gpt-4o-mini" {
			t.Errorf("model = %v, want 'gpt-4o-mini'", requestBody["model"])
		}
		if maxTokens, ok := requestBody["max_tokens"].(float64); !ok || maxTokens != 100 {
			t.Errorf("max_tokens = %v, want 100", requestBody["max_tokens"])
		}
		if temp, ok := requestBody["temperature"].(float64); !ok || temp != 0.7 {
			t.Errorf("temperature = %v, want 0.7", requestBody["temperature"])
		}
		if topP, ok := requestBody["top_p"].(float64); !ok || topP != 0.9 {
			t.Errorf("top_p = %v, want 0.9", requestBody["top_p"])
		}
		if user, ok := requestBody["user"].(string); !ok || user != "user-123" {
			t.Errorf("user = %v, want 'user-123'", requestBody["user"])
		}
	})

	t.Run("handles API error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = w.Write([]byte(`{"error": {"message": "Rate limit exceeded", "type": "rate_limit_error"}}`))
		}))
		defer server.Close()

		client, err := NewClient(
			WithAPIKey("test-key"),
			WithBaseURL(server.URL),
		)
		if err != nil {
			t.Fatalf("NewClient() error = %v", err)
		}

		messages := []chat.Message{
			{Role: chat.RoleUser, Contents: []chat.Content{chat.NewTextContent("Hello")}},
		}

		_, err = client.GetResponse(context.Background(), messages, nil)
		if err == nil {
			t.Fatal("GetResponse() expected error for rate limit")
		}
	})

	t.Run("handles empty choices", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			resp := `{
				"id": "chatcmpl-123",
				"object": "chat.completion",
				"created": 1677652288,
				"model": "gpt-4o",
				"choices": [],
				"usage": {"prompt_tokens": 5, "completion_tokens": 0, "total_tokens": 5}
			}`
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(resp))
		}))
		defer server.Close()

		client, err := NewClient(
			WithAPIKey("test-key"),
			WithBaseURL(server.URL),
		)
		if err != nil {
			t.Fatalf("NewClient() error = %v", err)
		}

		messages := []chat.Message{
			{Role: chat.RoleUser, Contents: []chat.Content{chat.NewTextContent("Hello")}},
		}

		_, err = client.GetResponse(context.Background(), messages, nil)
		if err == nil {
			t.Fatal("GetResponse() expected error for empty choices")
		}
		if !strings.Contains(err.Error(), "no choices returned") {
			t.Errorf("error should mention no choices, got: %v", err)
		}
	})

	t.Run("context cancellation", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(500 * time.Millisecond)
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		client, err := NewClient(
			WithAPIKey("test-key"),
			WithBaseURL(server.URL),
		)
		if err != nil {
			t.Fatalf("NewClient() error = %v", err)
		}

		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancel()

		messages := []chat.Message{
			{Role: chat.RoleUser, Contents: []chat.Content{chat.NewTextContent("Hello")}},
		}

		_, err = client.GetResponse(ctx, messages, nil)
		if err == nil {
			t.Fatal("GetResponse() expected error for context cancellation")
		}
	})
}

func TestClient_GetResponseWithTools(t *testing.T) {
	t.Run("converts tool.Tool to chat.ToolDefinition", func(t *testing.T) {
		var requestBody map[string]interface{}
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			decoder := json.NewDecoder(r.Body)
			if err := decoder.Decode(&requestBody); err != nil {
				t.Errorf("failed to decode request body: %v", err)
			}

			resp := `{
				"id": "chatcmpl-123",
				"object": "chat.completion",
				"created": 1677652288,
				"model": "gpt-4o",
				"choices": [{
					"index": 0,
					"message": {"role": "assistant", "content": "Response"},
					"finish_reason": "stop"
				}],
				"usage": {"prompt_tokens": 5, "completion_tokens": 5, "total_tokens": 10}
			}`
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(resp))
		}))
		defer server.Close()

		client, err := NewClient(
			WithAPIKey("test-key"),
			WithBaseURL(server.URL),
		)
		if err != nil {
			t.Fatalf("NewClient() error = %v", err)
		}

		tools := []tool.Tool{
			&mockTool{
				name:        "get_weather",
				description: "Get weather for a location",
				parameters:  json.RawMessage(`{"type":"object","properties":{"location":{"type":"string"}}}`),
			},
		}

		messages := []chat.Message{
			{Role: chat.RoleUser, Contents: []chat.Content{chat.NewTextContent("What's the weather?")}},
		}

		_, err = client.GetResponseWithTools(context.Background(), messages, tools, nil)
		if err != nil {
			t.Fatalf("GetResponseWithTools() error = %v", err)
		}

		// Verify tools were sent in request
		toolsRaw, ok := requestBody["tools"].([]interface{})
		if !ok {
			t.Fatal("tools should be present in request")
		}
		if len(toolsRaw) != 1 {
			t.Errorf("len(tools) = %d, want 1", len(toolsRaw))
		}
	})

	t.Run("skips hosted tools", func(t *testing.T) {
		var requestBody map[string]interface{}
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			decoder := json.NewDecoder(r.Body)
			_ = decoder.Decode(&requestBody)

			resp := `{
				"id": "chatcmpl-123",
				"object": "chat.completion",
				"created": 1677652288,
				"model": "gpt-4o",
				"choices": [{
					"index": 0,
					"message": {"role": "assistant", "content": "Response"},
					"finish_reason": "stop"
				}],
				"usage": {"prompt_tokens": 5, "completion_tokens": 5, "total_tokens": 10}
			}`
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(resp))
		}))
		defer server.Close()

		client, err := NewClient(
			WithAPIKey("test-key"),
			WithBaseURL(server.URL),
		)
		if err != nil {
			t.Fatalf("NewClient() error = %v", err)
		}

		tools := []tool.Tool{
			&mockTool{
				name:        "function_tool",
				description: "A function tool",
				parameters:  json.RawMessage(`{}`),
			},
			&mockHostedTool{
				name:   "web_search",
				config: map[string]interface{}{"type": "web_search"},
			},
		}

		messages := []chat.Message{
			{Role: chat.RoleUser, Contents: []chat.Content{chat.NewTextContent("Search the web")}},
		}

		_, err = client.GetResponseWithTools(context.Background(), messages, tools, nil)
		if err != nil {
			t.Fatalf("GetResponseWithTools() error = %v", err)
		}

		// Verify only function tool was sent (hosted tools skipped)
		toolsRaw, ok := requestBody["tools"].([]interface{})
		if !ok {
			t.Fatal("tools should be present in request")
		}
		if len(toolsRaw) != 1 {
			t.Errorf("len(tools) = %d, want 1 (hosted tools should be skipped)", len(toolsRaw))
		}
	})
}

func TestClient_GetStreamingResponse(t *testing.T) {
	t.Run("returns channel for streaming", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/event-stream")
			w.Header().Set("Cache-Control", "no-cache")
			w.Header().Set("Connection", "keep-alive")

			chunks := []string{
				`data: {"id":"chatcmpl-123","object":"chat.completion.chunk","created":1677652288,"model":"gpt-4o","choices":[{"index":0,"delta":{"role":"assistant","content":"Hello"},"finish_reason":null}]}`,
				`data: {"id":"chatcmpl-123","object":"chat.completion.chunk","created":1677652288,"model":"gpt-4o","choices":[{"index":0,"delta":{"content":" world"},"finish_reason":null}]}`,
				`data: {"id":"chatcmpl-123","object":"chat.completion.chunk","created":1677652288,"model":"gpt-4o","choices":[{"index":0,"delta":{},"finish_reason":"stop"}]}`,
				`data: [DONE]`,
			}

			flusher, ok := w.(http.Flusher)
			if !ok {
				t.Fatal("ResponseWriter does not support Flusher")
				return
			}

			for _, chunk := range chunks {
				_, _ = w.Write([]byte(chunk + "\n\n"))
				flusher.Flush()
				time.Sleep(10 * time.Millisecond)
			}
		}))
		defer server.Close()

		client, err := NewClient(
			WithAPIKey("test-key"),
			WithBaseURL(server.URL),
		)
		if err != nil {
			t.Fatalf("NewClient() error = %v", err)
		}

		messages := []chat.Message{
			{Role: chat.RoleUser, Contents: []chat.Content{chat.NewTextContent("Hello")}},
		}

		updates, err := client.GetStreamingResponse(context.Background(), messages, nil)
		if err != nil {
			t.Fatalf("GetStreamingResponse() error = %v", err)
		}

		var textContent strings.Builder
		var finishReason chat.FinishReason
		updateCount := 0

		for update := range updates {
			updateCount++
			if update.Kind == chat.UpdateKindContentDelta && update.Delta != nil {
				textContent.WriteString(update.Delta.TextDelta)
			}
			if update.Kind == chat.UpdateKindMessageComplete {
				finishReason = update.FinishReason
			}
			if update.Kind == chat.UpdateKindError {
				t.Errorf("unexpected error: %v", update.Error)
			}
		}

		if textContent.String() != "Hello world" {
			t.Errorf("collected text = %q, want %q", textContent.String(), "Hello world")
		}
		if finishReason != chat.FinishReasonStop {
			t.Errorf("finish reason = %v, want %v", finishReason, chat.FinishReasonStop)
		}
		if updateCount < 3 {
			t.Errorf("update count = %d, want at least 3", updateCount)
		}
	})

	t.Run("handles stream creation error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"error": {"message": "Internal server error"}}`))
		}))
		defer server.Close()

		client, err := NewClient(
			WithAPIKey("test-key"),
			WithBaseURL(server.URL),
		)
		if err != nil {
			t.Fatalf("NewClient() error = %v", err)
		}

		messages := []chat.Message{
			{Role: chat.RoleUser, Contents: []chat.Content{chat.NewTextContent("Hello")}},
		}

		_, err = client.GetStreamingResponse(context.Background(), messages, nil)
		if err == nil {
			t.Fatal("GetStreamingResponse() expected error")
		}
	})
}

func TestClient_ToolChoiceHandling(t *testing.T) {
	tests := []struct {
		name         string
		toolChoice   string
		expectedType string
	}{
		{"auto", "auto", "string"},
		{"required", "required", "string"},
		{"none", "none", "string"},
		{"specific tool", "get_weather", "object"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var requestBody map[string]interface{}
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				decoder := json.NewDecoder(r.Body)
				_ = decoder.Decode(&requestBody)

				resp := `{
					"id": "chatcmpl-123",
					"object": "chat.completion",
					"created": 1677652288,
					"model": "gpt-4o",
					"choices": [{
						"index": 0,
						"message": {"role": "assistant", "content": "Response"},
						"finish_reason": "stop"
					}],
					"usage": {"prompt_tokens": 5, "completion_tokens": 5, "total_tokens": 10}
				}`
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(resp))
			}))
			defer server.Close()

			client, err := NewClient(
				WithAPIKey("test-key"),
				WithBaseURL(server.URL),
			)
			if err != nil {
				t.Fatalf("NewClient() error = %v", err)
			}

			options := &chat.Options{
				Tools: []chat.ToolDefinition{
					{Name: "get_weather", Description: "Get weather"},
				},
				ToolChoice: tt.toolChoice,
			}

			messages := []chat.Message{
				{Role: chat.RoleUser, Contents: []chat.Content{chat.NewTextContent("Test")}},
			}

			_, err = client.GetResponse(context.Background(), messages, options)
			if err != nil {
				t.Fatalf("GetResponse() error = %v", err)
			}

			toolChoice := requestBody["tool_choice"]
			if toolChoice == nil {
				t.Fatal("tool_choice should be present in request")
			}

			switch tt.expectedType {
			case "string":
				if _, ok := toolChoice.(string); !ok {
					t.Errorf("tool_choice type = %T, want string", toolChoice)
				}
			case "object":
				if _, ok := toolChoice.(map[string]interface{}); !ok {
					t.Errorf("tool_choice type = %T, want object", toolChoice)
				}
			}
		})
	}
}

func TestClient_InterfaceCompliance(t *testing.T) {
	var _ chat.Client = (*Client)(nil)
}

func TestClient_GetStreamingResponseWithTools(t *testing.T) {
	t.Run("converts tool.Tool to chat.ToolDefinition for streaming", func(t *testing.T) {
		var requestBody map[string]interface{}
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			decoder := json.NewDecoder(r.Body)
			_ = decoder.Decode(&requestBody)

			w.Header().Set("Content-Type", "text/event-stream")
			w.Header().Set("Cache-Control", "no-cache")
			w.Header().Set("Connection", "keep-alive")

			chunks := []string{
				`data: {"id":"chatcmpl-123","object":"chat.completion.chunk","created":1677652288,"model":"gpt-4o","choices":[{"index":0,"delta":{"role":"assistant","content":"Hi"},"finish_reason":null}]}`,
				`data: {"id":"chatcmpl-123","object":"chat.completion.chunk","created":1677652288,"model":"gpt-4o","choices":[{"index":0,"delta":{},"finish_reason":"stop"}]}`,
				`data: [DONE]`,
			}

			flusher, ok := w.(http.Flusher)
			if !ok {
				t.Fatal("ResponseWriter does not support Flusher")
				return
			}

			for _, chunk := range chunks {
				_, _ = w.Write([]byte(chunk + "\n\n"))
				flusher.Flush()
				time.Sleep(5 * time.Millisecond)
			}
		}))
		defer server.Close()

		client, err := NewClient(
			WithAPIKey("test-key"),
			WithBaseURL(server.URL),
		)
		if err != nil {
			t.Fatalf("NewClient() error = %v", err)
		}

		tools := []tool.Tool{
			&mockTool{
				name:        "get_weather",
				description: "Get weather for a location",
				parameters:  json.RawMessage(`{"type":"object","properties":{"location":{"type":"string"}}}`),
			},
		}

		messages := []chat.Message{
			{Role: chat.RoleUser, Contents: []chat.Content{chat.NewTextContent("What's the weather?")}},
		}

		updates, err := client.GetStreamingResponseWithTools(context.Background(), messages, tools, nil)
		if err != nil {
			t.Fatalf("GetStreamingResponseWithTools() error = %v", err)
		}

		// Drain the channel
		for range updates {
			// intentionally empty - drain streaming updates
		}

		// Verify tools were sent in request
		toolsRaw, ok := requestBody["tools"].([]interface{})
		if !ok {
			t.Fatal("tools should be present in request")
		}
		if len(toolsRaw) != 1 {
			t.Errorf("len(tools) = %d, want 1", len(toolsRaw))
		}
	})

	t.Run("skips hosted tools in streaming", func(t *testing.T) {
		var requestBody map[string]interface{}
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			decoder := json.NewDecoder(r.Body)
			_ = decoder.Decode(&requestBody)

			w.Header().Set("Content-Type", "text/event-stream")
			chunks := []string{
				`data: {"id":"chatcmpl-123","object":"chat.completion.chunk","created":1677652288,"model":"gpt-4o","choices":[{"index":0,"delta":{"role":"assistant","content":"Hi"},"finish_reason":null}]}`,
				`data: {"id":"chatcmpl-123","object":"chat.completion.chunk","created":1677652288,"model":"gpt-4o","choices":[{"index":0,"delta":{},"finish_reason":"stop"}]}`,
				`data: [DONE]`,
			}

			flusher, _ := w.(http.Flusher)
			for _, chunk := range chunks {
				_, _ = w.Write([]byte(chunk + "\n\n"))
				if flusher != nil {
					flusher.Flush()
				}
			}
		}))
		defer server.Close()

		client, err := NewClient(
			WithAPIKey("test-key"),
			WithBaseURL(server.URL),
		)
		if err != nil {
			t.Fatalf("NewClient() error = %v", err)
		}

		tools := []tool.Tool{
			&mockTool{
				name:        "function_tool",
				description: "A function tool",
				parameters:  json.RawMessage(`{}`),
			},
			&mockHostedTool{
				name:   "web_search",
				config: map[string]interface{}{"type": "web_search"},
			},
		}

		messages := []chat.Message{
			{Role: chat.RoleUser, Contents: []chat.Content{chat.NewTextContent("Search the web")}},
		}

		updates, err := client.GetStreamingResponseWithTools(context.Background(), messages, tools, nil)
		if err != nil {
			t.Fatalf("GetStreamingResponseWithTools() error = %v", err)
		}

		// Drain the channel
		for range updates {
			// intentionally empty - drain streaming updates
		}

		// Verify only function tool was sent (hosted tools skipped)
		toolsRaw, ok := requestBody["tools"].([]interface{})
		if !ok {
			t.Fatal("tools should be present in request")
		}
		if len(toolsRaw) != 1 {
			t.Errorf("len(tools) = %d, want 1 (hosted tools should be skipped)", len(toolsRaw))
		}
	})
}

func TestUnmarshalParameters(t *testing.T) {
	t.Run("returns nil for empty bytes", func(t *testing.T) {
		result := unmarshalParameters(nil)
		if result != nil {
			t.Errorf("expected nil, got %v", result)
		}

		result = unmarshalParameters([]byte{})
		if result != nil {
			t.Errorf("expected nil for empty slice, got %v", result)
		}
	})

	t.Run("returns nil for invalid JSON", func(t *testing.T) {
		result := unmarshalParameters([]byte("not valid json"))
		if result != nil {
			t.Errorf("expected nil for invalid JSON, got %v", result)
		}
	})

	t.Run("parses valid JSON", func(t *testing.T) {
		params := []byte(`{"type":"object","properties":{"x":{"type":"string"}}}`)
		result := unmarshalParameters(params)
		if result == nil {
			t.Fatal("expected non-nil result")
		}
		if result["type"] != "object" {
			t.Errorf("expected type 'object', got %v", result["type"])
		}
	})
}
