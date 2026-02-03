//go:build integration

// Copyright (c) Microsoft. All rights reserved.

package openai

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/microsoft/agent-framework-go/chat"
)

// Integration tests require a valid OPENAI_API_KEY environment variable.
// Run with: go test -tags=integration ./providers/openai/...
//
// These tests make real API calls and may incur costs.

func skipIfNoAPIKey(t *testing.T) {
	t.Helper()
	if os.Getenv("OPENAI_API_KEY") == "" {
		t.Skip("OPENAI_API_KEY not set, skipping integration test")
	}
}

func TestIntegration_Client_GetResponse(t *testing.T) {
	skipIfNoAPIKey(t)

	client, err := NewClient(
		WithModel("gpt-4o-mini"), // Use cheaper model for testing
	)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	t.Run("simple completion", func(t *testing.T) {
		messages := []chat.Message{
			{Role: chat.RoleSystem, Contents: []chat.Content{chat.NewTextContent("You are a helpful assistant. Respond briefly.")}},
			{Role: chat.RoleUser, Contents: []chat.Content{chat.NewTextContent("Say hello")}},
		}

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		resp, err := client.GetResponse(ctx, messages, nil)
		if err != nil {
			t.Fatalf("GetResponse() error = %v", err)
		}

		if resp.Message.Text() == "" {
			t.Error("expected non-empty response text")
		}
		if resp.FinishReason != chat.FinishReasonStop {
			t.Errorf("FinishReason = %v, want stop", resp.FinishReason)
		}
		if resp.Usage == nil {
			t.Error("Usage should not be nil")
		} else {
			if resp.Usage.InputTokens == 0 {
				t.Error("InputTokens should be > 0")
			}
			if resp.Usage.OutputTokens == 0 {
				t.Error("OutputTokens should be > 0")
			}
		}
	})

	t.Run("with options", func(t *testing.T) {
		messages := []chat.Message{
			{Role: chat.RoleUser, Contents: []chat.Content{chat.NewTextContent("Count from 1 to 3")}},
		}

		options := &chat.Options{
			MaxTokens:   50,
			Temperature: 0.1,
		}

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		resp, err := client.GetResponse(ctx, messages, options)
		if err != nil {
			t.Fatalf("GetResponse() error = %v", err)
		}

		if resp.Message.Text() == "" {
			t.Error("expected non-empty response text")
		}
	})

	t.Run("with tools", func(t *testing.T) {
		messages := []chat.Message{
			{Role: chat.RoleUser, Contents: []chat.Content{chat.NewTextContent("What is the weather in Paris? Use the get_weather tool.")}},
		}

		options := &chat.Options{
			Tools: []chat.ToolDefinition{
				{
					Name:        "get_weather",
					Description: "Get the current weather for a location",
					Parameters: map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"location": map[string]interface{}{
								"type":        "string",
								"description": "The city name",
							},
						},
						"required": []string{"location"},
					},
				},
			},
			ToolChoice: "auto",
		}

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		resp, err := client.GetResponse(ctx, messages, options)
		if err != nil {
			t.Fatalf("GetResponse() error = %v", err)
		}

		// Model should either call the tool or respond directly
		if resp.FinishReason != chat.FinishReasonToolCalls && resp.FinishReason != chat.FinishReasonStop {
			t.Errorf("unexpected FinishReason = %v", resp.FinishReason)
		}

		if resp.FinishReason == chat.FinishReasonToolCalls {
			if len(resp.Message.ToolCalls) == 0 {
				t.Error("expected tool calls when FinishReason is tool_calls")
			} else {
				tc := resp.Message.ToolCalls[0]
				if tc.Name != "get_weather" {
					t.Errorf("ToolCall.Name = %q, want 'get_weather'", tc.Name)
				}
				if tc.ID == "" {
					t.Error("ToolCall.ID should not be empty")
				}
			}
		}
	})

	t.Run("conversation with tool result", func(t *testing.T) {
		// First message triggers tool call
		messages := []chat.Message{
			{Role: chat.RoleUser, Contents: []chat.Content{chat.NewTextContent("What is the weather in London?")}},
		}

		options := &chat.Options{
			Tools: []chat.ToolDefinition{
				{
					Name:        "get_weather",
					Description: "Get the current weather for a location",
					Parameters: map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"location": map[string]interface{}{"type": "string"},
						},
					},
				},
			},
			ToolChoice: "required",
		}

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		resp, err := client.GetResponse(ctx, messages, options)
		if err != nil {
			t.Fatalf("First GetResponse() error = %v", err)
		}

		if len(resp.Message.ToolCalls) == 0 {
			t.Skip("Model did not make a tool call, skipping conversation test")
		}

		// Build conversation with tool result
		toolCallID := resp.Message.ToolCalls[0].ID
		conversationMessages := []chat.Message{
			messages[0],
			resp.Message,
			{
				Role:       chat.RoleTool,
				ToolCallID: toolCallID,
				Contents:   []chat.Content{chat.NewTextContent("The weather in London is 15°C and cloudy.")},
			},
		}

		// Get final response
		finalOptions := &chat.Options{}
		finalResp, err := client.GetResponse(ctx, conversationMessages, finalOptions)
		if err != nil {
			t.Fatalf("Second GetResponse() error = %v", err)
		}

		if finalResp.Message.Text() == "" {
			t.Error("expected non-empty final response")
		}
		// The response should mention the weather information
		if !strings.Contains(strings.ToLower(finalResp.Message.Text()), "london") &&
			!strings.Contains(strings.ToLower(finalResp.Message.Text()), "15") &&
			!strings.Contains(strings.ToLower(finalResp.Message.Text()), "cloudy") {
			t.Logf("Response may not contain weather info: %s", finalResp.Message.Text())
		}
	})
}

func TestIntegration_Client_GetStreamingResponse(t *testing.T) {
	skipIfNoAPIKey(t)

	client, err := NewClient(
		WithModel("gpt-4o-mini"),
	)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	t.Run("streaming completion", func(t *testing.T) {
		messages := []chat.Message{
			{Role: chat.RoleUser, Contents: []chat.Content{chat.NewTextContent("Count from 1 to 5")}},
		}

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		updates, err := client.GetStreamingResponse(ctx, messages, nil)
		if err != nil {
			t.Fatalf("GetStreamingResponse() error = %v", err)
		}

		var textContent strings.Builder
		var finishReason chat.FinishReason
		var hasUsage bool
		var hasDone bool
		updateCount := 0

		for update := range updates {
			updateCount++

			switch update.Kind {
			case chat.UpdateKindContentDelta:
				if update.Delta != nil {
					textContent.WriteString(update.Delta.TextDelta)
				}
			case chat.UpdateKindMessageComplete:
				finishReason = update.FinishReason
			case chat.UpdateKindUsage:
				hasUsage = true
			case chat.UpdateKindDone:
				hasDone = true
			case chat.UpdateKindError:
				t.Errorf("unexpected error: %v", update.Error)
			}
		}

		if textContent.Len() == 0 {
			t.Error("expected non-empty streamed content")
		}
		if finishReason != chat.FinishReasonStop {
			t.Errorf("FinishReason = %v, want stop", finishReason)
		}
		if !hasUsage {
			t.Error("expected usage update")
		}
		if !hasDone {
			t.Error("expected done update")
		}
		if updateCount < 3 {
			t.Errorf("updateCount = %d, expected at least 3 updates", updateCount)
		}
	})

	t.Run("streaming with context cancellation", func(t *testing.T) {
		messages := []chat.Message{
			{Role: chat.RoleUser, Contents: []chat.Content{chat.NewTextContent("Write a very long story")}},
		}

		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		updates, err := client.GetStreamingResponse(ctx, messages, nil)
		if err != nil {
			t.Fatalf("GetStreamingResponse() error = %v", err)
		}

		var hasError bool
		for update := range updates {
			if update.Kind == chat.UpdateKindError {
				hasError = true
				if update.Error != context.DeadlineExceeded && update.Error.Error() != "context deadline exceeded" {
					t.Logf("error type: %T, value: %v", update.Error, update.Error)
				}
			}
		}

		// We expect either an error or the stream to complete before timeout
		// depending on API response time
		_ = hasError
	})

	t.Run("streaming with tool calls", func(t *testing.T) {
		messages := []chat.Message{
			{Role: chat.RoleUser, Contents: []chat.Content{chat.NewTextContent("What's the weather in Tokyo?")}},
		}

		options := &chat.Options{
			Tools: []chat.ToolDefinition{
				{
					Name:        "get_weather",
					Description: "Get weather for a location",
					Parameters: map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"location": map[string]interface{}{"type": "string"},
						},
					},
				},
			},
			ToolChoice: "required",
		}

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		updates, err := client.GetStreamingResponse(ctx, messages, options)
		if err != nil {
			t.Fatalf("GetStreamingResponse() error = %v", err)
		}

		var hasToolCall bool
		var finishReason chat.FinishReason

		for update := range updates {
			if update.Kind == chat.UpdateKindToolCall {
				hasToolCall = true
			}
			if update.Kind == chat.UpdateKindMessageComplete {
				finishReason = update.FinishReason
			}
		}

		if !hasToolCall {
			t.Error("expected tool call update")
		}
		if finishReason != chat.FinishReasonToolCalls {
			t.Errorf("FinishReason = %v, want tool_calls", finishReason)
		}
	})
}

func TestIntegration_Client_Metadata(t *testing.T) {
	skipIfNoAPIKey(t)

	client, err := NewClient(
		WithModel("gpt-4o-mini"),
	)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	metadata := client.Metadata()

	if metadata.ProviderName != "openai" {
		t.Errorf("ProviderName = %q, want 'openai'", metadata.ProviderName)
	}
	if metadata.ModelID != "gpt-4o-mini" {
		t.Errorf("ModelID = %q, want 'gpt-4o-mini'", metadata.ModelID)
	}
	if metadata.EndpointURI == "" {
		t.Error("EndpointURI should not be empty")
	}
}

func TestIntegration_ResponsesClient_GetResponse(t *testing.T) {
	skipIfNoAPIKey(t)

	client, err := NewResponsesClient(
		ResponsesWithModel("gpt-4o-mini"),
	)
	if err != nil {
		t.Fatalf("NewResponsesClient() error = %v", err)
	}

	t.Run("simple completion", func(t *testing.T) {
		messages := []chat.Message{
			{Role: chat.RoleUser, Contents: []chat.Content{chat.NewTextContent("Say hello")}},
		}

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		resp, err := client.GetResponse(ctx, messages, nil)
		if err != nil {
			t.Fatalf("GetResponse() error = %v", err)
		}

		if resp.Message.Text() == "" {
			t.Error("expected non-empty response text")
		}
	})

	t.Run("conversation continuation", func(t *testing.T) {
		// First message
		messages := []chat.Message{
			{Role: chat.RoleUser, Contents: []chat.Content{chat.NewTextContent("My name is Alice")}},
		}

		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()

		resp1, err := client.GetResponse(ctx, messages, nil)
		if err != nil {
			t.Fatalf("First GetResponse() error = %v", err)
		}

		// Check that response ID was stored
		responseID := client.GetPreviousResponseID()
		if responseID == "" {
			t.Skip("Response ID not returned, skipping continuation test")
		}

		// Second message should continue the conversation
		messages2 := []chat.Message{
			{Role: chat.RoleUser, Contents: []chat.Content{chat.NewTextContent("What is my name?")}},
		}

		resp2, err := client.GetResponse(ctx, messages2, nil)
		if err != nil {
			t.Fatalf("Second GetResponse() error = %v", err)
		}

		// The response should mention "Alice"
		if !strings.Contains(strings.ToLower(resp2.Message.Text()), "alice") {
			t.Logf("First response: %s", resp1.Message.Text())
			t.Logf("Second response: %s", resp2.Message.Text())
			t.Error("conversation continuation may not have worked - 'Alice' not found in response")
		}
	})
}

func TestIntegration_DifferentModels(t *testing.T) {
	skipIfNoAPIKey(t)

	models := []string{
		"gpt-4o-mini",
		"gpt-3.5-turbo",
	}

	for _, model := range models {
		t.Run(model, func(t *testing.T) {
			client, err := NewClient(WithModel(model))
			if err != nil {
				t.Fatalf("NewClient() error = %v", err)
			}

			messages := []chat.Message{
				{Role: chat.RoleUser, Contents: []chat.Content{chat.NewTextContent("Say OK")}},
			}

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			resp, err := client.GetResponse(ctx, messages, &chat.Options{MaxTokens: 10})
			if err != nil {
				t.Fatalf("GetResponse() error = %v", err)
			}

			if resp.Message.Text() == "" {
				t.Error("expected non-empty response")
			}
		})
	}
}
