// Copyright (c) Microsoft. All rights reserved.

package openai

import (
	"encoding/json"
	"testing"

	"github.com/microsoft/agent-framework-go/chat"
	openailib "github.com/sashabaranov/go-openai"
)

func TestToOpenAIMessages(t *testing.T) {
	t.Run("converts simple text messages", func(t *testing.T) {
		messages := []chat.Message{
			{Role: chat.RoleUser, Contents: []chat.Content{chat.NewTextContent("Hello")}},
			{Role: chat.RoleAssistant, Contents: []chat.Content{chat.NewTextContent("Hi there!")}},
		}

		result := toOpenAIMessages(messages, "system")

		if len(result) != 2 {
			t.Fatalf("len(result) = %d, want 2", len(result))
		}
		if result[0].Role != "user" {
			t.Errorf("result[0].Role = %q, want 'user'", result[0].Role)
		}
		if result[0].Content != "Hello" {
			t.Errorf("result[0].Content = %q, want 'Hello'", result[0].Content)
		}
		if result[1].Role != "assistant" {
			t.Errorf("result[1].Role = %q, want 'assistant'", result[1].Role)
		}
	})

	t.Run("handles system role with default instruction role", func(t *testing.T) {
		messages := []chat.Message{
			{Role: chat.RoleSystem, Contents: []chat.Content{chat.NewTextContent("You are helpful.")}},
		}

		result := toOpenAIMessages(messages, "system")

		if result[0].Role != "system" {
			t.Errorf("result[0].Role = %q, want 'system'", result[0].Role)
		}
	})

	t.Run("converts system to developer role when specified", func(t *testing.T) {
		messages := []chat.Message{
			{Role: chat.RoleSystem, Contents: []chat.Content{chat.NewTextContent("You are helpful.")}},
		}

		result := toOpenAIMessages(messages, "developer")

		if result[0].Role != "developer" {
			t.Errorf("result[0].Role = %q, want 'developer'", result[0].Role)
		}
	})
}

func TestToOpenAIMessage(t *testing.T) {
	t.Run("handles single text content", func(t *testing.T) {
		msg := chat.Message{
			Role:     chat.RoleUser,
			Contents: []chat.Content{chat.NewTextContent("Hello world")},
		}

		result := toOpenAIMessage(msg, "system")

		if result.Role != "user" {
			t.Errorf("Role = %q, want 'user'", result.Role)
		}
		if result.Content != "Hello world" {
			t.Errorf("Content = %q, want 'Hello world'", result.Content)
		}
		if len(result.MultiContent) != 0 {
			t.Error("MultiContent should be empty for single text content")
		}
	})

	t.Run("handles multiple content items", func(t *testing.T) {
		msg := chat.Message{
			Role: chat.RoleUser,
			Contents: []chat.Content{
				chat.NewTextContent("First"),
				chat.NewTextContent("Second"),
			},
		}

		result := toOpenAIMessage(msg, "system")

		if len(result.MultiContent) != 2 {
			t.Fatalf("len(MultiContent) = %d, want 2", len(result.MultiContent))
		}
		if result.MultiContent[0].Type != openailib.ChatMessagePartTypeText {
			t.Errorf("MultiContent[0].Type = %v, want text", result.MultiContent[0].Type)
		}
	})

	t.Run("handles image content with URL", func(t *testing.T) {
		msg := chat.Message{
			Role: chat.RoleUser,
			Contents: []chat.Content{
				&chat.ImageContent{URL: "https://example.com/image.jpg", Detail: "high"},
			},
		}

		result := toOpenAIMessage(msg, "system")

		if len(result.MultiContent) != 1 {
			t.Fatalf("len(MultiContent) = %d, want 1", len(result.MultiContent))
		}
		if result.MultiContent[0].Type != openailib.ChatMessagePartTypeImageURL {
			t.Errorf("MultiContent[0].Type = %v, want image_url", result.MultiContent[0].Type)
		}
		if result.MultiContent[0].ImageURL.URL != "https://example.com/image.jpg" {
			t.Errorf("ImageURL.URL = %q, want 'https://example.com/image.jpg'", result.MultiContent[0].ImageURL.URL)
		}
		if result.MultiContent[0].ImageURL.Detail != "high" {
			t.Errorf("ImageURL.Detail = %q, want 'high'", result.MultiContent[0].ImageURL.Detail)
		}
	})

	t.Run("handles image content with base64 data", func(t *testing.T) {
		msg := chat.Message{
			Role: chat.RoleUser,
			Contents: []chat.Content{
				&chat.ImageContent{Base64Data: "base64encodeddata", MediaType: "image/png"},
			},
		}

		result := toOpenAIMessage(msg, "system")

		if len(result.MultiContent) != 1 {
			t.Fatalf("len(MultiContent) = %d, want 1", len(result.MultiContent))
		}
		expectedURL := "data:image/png;base64,base64encodeddata"
		if result.MultiContent[0].ImageURL.URL != expectedURL {
			t.Errorf("ImageURL.URL = %q, want %q", result.MultiContent[0].ImageURL.URL, expectedURL)
		}
	})

	t.Run("handles tool result messages", func(t *testing.T) {
		msg := chat.Message{
			Role:       chat.RoleTool,
			ToolCallID: "call_123",
			Contents:   []chat.Content{chat.NewTextContent("Tool result content")},
		}

		result := toOpenAIMessage(msg, "system")

		if result.Role != "tool" {
			t.Errorf("Role = %q, want 'tool'", result.Role)
		}
		if result.ToolCallID != "call_123" {
			t.Errorf("ToolCallID = %q, want 'call_123'", result.ToolCallID)
		}
		if result.Content != "Tool result content" {
			t.Errorf("Content = %q, want 'Tool result content'", result.Content)
		}
	})

	t.Run("handles assistant message with tool calls", func(t *testing.T) {
		msg := chat.Message{
			Role: chat.RoleAssistant,
			ToolCalls: []chat.ToolCall{
				{
					ID:        "call_abc",
					Name:      "get_weather",
					Arguments: json.RawMessage(`{"location":"NYC"}`),
				},
			},
		}

		result := toOpenAIMessage(msg, "system")

		if len(result.ToolCalls) != 1 {
			t.Fatalf("len(ToolCalls) = %d, want 1", len(result.ToolCalls))
		}
		if result.ToolCalls[0].ID != "call_abc" {
			t.Errorf("ToolCalls[0].ID = %q, want 'call_abc'", result.ToolCalls[0].ID)
		}
		if result.ToolCalls[0].Function.Name != "get_weather" {
			t.Errorf("ToolCalls[0].Function.Name = %q, want 'get_weather'", result.ToolCalls[0].Function.Name)
		}
		if result.ToolCalls[0].Function.Arguments != `{"location":"NYC"}` {
			t.Errorf("ToolCalls[0].Function.Arguments = %q", result.ToolCalls[0].Function.Arguments)
		}
	})

	t.Run("preserves message name", func(t *testing.T) {
		msg := chat.Message{
			Role:     chat.RoleUser,
			Name:     "John",
			Contents: []chat.Content{chat.NewTextContent("Hello")},
		}

		result := toOpenAIMessage(msg, "system")

		if result.Name != "John" {
			t.Errorf("Name = %q, want 'John'", result.Name)
		}
	})
}

func TestExtractTextContent(t *testing.T) {
	tests := []struct {
		name     string
		contents []chat.Content
		expected string
	}{
		{
			name:     "empty contents",
			contents: []chat.Content{},
			expected: "",
		},
		{
			name:     "single text content",
			contents: []chat.Content{chat.NewTextContent("Hello")},
			expected: "Hello",
		},
		{
			name: "multiple text contents",
			contents: []chat.Content{
				chat.NewTextContent("Hello"),
				chat.NewTextContent(" "),
				chat.NewTextContent("World"),
			},
			expected: "Hello World",
		},
		{
			name: "mixed content types",
			contents: []chat.Content{
				chat.NewTextContent("Text"),
				&chat.ImageContent{URL: "https://example.com/img.jpg"},
				chat.NewTextContent(" more"),
			},
			expected: "Text more",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractTextContent(tt.contents)
			if result != tt.expected {
				t.Errorf("extractTextContent() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestToOpenAIContentParts(t *testing.T) {
	t.Run("converts text content", func(t *testing.T) {
		contents := []chat.Content{chat.NewTextContent("Hello")}

		result := toOpenAIContentParts(contents)

		if len(result) != 1 {
			t.Fatalf("len(result) = %d, want 1", len(result))
		}
		if result[0].Type != openailib.ChatMessagePartTypeText {
			t.Errorf("Type = %v, want text", result[0].Type)
		}
		if result[0].Text != "Hello" {
			t.Errorf("Text = %q, want 'Hello'", result[0].Text)
		}
	})

	t.Run("converts image content with URL", func(t *testing.T) {
		contents := []chat.Content{
			&chat.ImageContent{URL: "https://example.com/img.jpg", Detail: "low"},
		}

		result := toOpenAIContentParts(contents)

		if len(result) != 1 {
			t.Fatalf("len(result) = %d, want 1", len(result))
		}
		if result[0].Type != openailib.ChatMessagePartTypeImageURL {
			t.Errorf("Type = %v, want image_url", result[0].Type)
		}
		if result[0].ImageURL.URL != "https://example.com/img.jpg" {
			t.Errorf("ImageURL.URL = %q", result[0].ImageURL.URL)
		}
	})

	t.Run("handles empty contents", func(t *testing.T) {
		result := toOpenAIContentParts([]chat.Content{})

		if len(result) != 0 {
			t.Errorf("len(result) = %d, want 0", len(result))
		}
	})
}

func TestToOpenAIToolCalls(t *testing.T) {
	t.Run("converts tool calls", func(t *testing.T) {
		calls := []chat.ToolCall{
			{
				ID:        "call_1",
				Name:      "get_weather",
				Arguments: json.RawMessage(`{"location":"NYC"}`),
			},
			{
				ID:        "call_2",
				Name:      "get_time",
				Arguments: json.RawMessage(`{"timezone":"EST"}`),
			},
		}

		result := toOpenAIToolCalls(calls)

		if len(result) != 2 {
			t.Fatalf("len(result) = %d, want 2", len(result))
		}

		if result[0].ID != "call_1" {
			t.Errorf("result[0].ID = %q, want 'call_1'", result[0].ID)
		}
		if result[0].Type != openailib.ToolTypeFunction {
			t.Errorf("result[0].Type = %v, want function", result[0].Type)
		}
		if result[0].Function.Name != "get_weather" {
			t.Errorf("result[0].Function.Name = %q", result[0].Function.Name)
		}
		if result[0].Function.Arguments != `{"location":"NYC"}` {
			t.Errorf("result[0].Function.Arguments = %q", result[0].Function.Arguments)
		}

		if result[1].ID != "call_2" {
			t.Errorf("result[1].ID = %q, want 'call_2'", result[1].ID)
		}
	})

	t.Run("handles empty tool calls", func(t *testing.T) {
		result := toOpenAIToolCalls([]chat.ToolCall{})

		if len(result) != 0 {
			t.Errorf("len(result) = %d, want 0", len(result))
		}
	})
}

func TestToOpenAITools(t *testing.T) {
	t.Run("converts tool definitions", func(t *testing.T) {
		tools := []chat.ToolDefinition{
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
		}

		result := toOpenAITools(tools)

		if len(result) != 1 {
			t.Fatalf("len(result) = %d, want 1", len(result))
		}
		if result[0].Type != openailib.ToolTypeFunction {
			t.Errorf("Type = %v, want function", result[0].Type)
		}
		if result[0].Function.Name != "get_weather" {
			t.Errorf("Function.Name = %q, want 'get_weather'", result[0].Function.Name)
		}
		if result[0].Function.Description != "Get weather for a location" {
			t.Errorf("Function.Description = %q", result[0].Function.Description)
		}
	})

	t.Run("handles nil parameters", func(t *testing.T) {
		tools := []chat.ToolDefinition{
			{
				Name:        "no_params",
				Description: "Tool with no parameters",
				Parameters:  nil,
			},
		}

		result := toOpenAITools(tools)

		if len(result) != 1 {
			t.Fatalf("len(result) = %d, want 1", len(result))
		}
		// Should not panic with nil parameters
	})
}

func TestFromOpenAIMessage(t *testing.T) {
	t.Run("converts text response", func(t *testing.T) {
		choice := openailib.ChatCompletionChoice{
			Message: openailib.ChatCompletionMessage{
				Role:    "assistant",
				Content: "Hello! How can I help?",
			},
		}

		result := fromOpenAIMessage(choice)

		if result.Role != chat.RoleAssistant {
			t.Errorf("Role = %v, want assistant", result.Role)
		}
		if result.Text() != "Hello! How can I help?" {
			t.Errorf("Text() = %q, want 'Hello! How can I help?'", result.Text())
		}
	})

	t.Run("converts response with tool calls", func(t *testing.T) {
		choice := openailib.ChatCompletionChoice{
			Message: openailib.ChatCompletionMessage{
				Role: "assistant",
				ToolCalls: []openailib.ToolCall{
					{
						ID:   "call_123",
						Type: openailib.ToolTypeFunction,
						Function: openailib.FunctionCall{
							Name:      "get_weather",
							Arguments: `{"location":"NYC"}`,
						},
					},
				},
			},
		}

		result := fromOpenAIMessage(choice)

		if len(result.ToolCalls) != 1 {
			t.Fatalf("len(ToolCalls) = %d, want 1", len(result.ToolCalls))
		}
		if result.ToolCalls[0].ID != "call_123" {
			t.Errorf("ToolCalls[0].ID = %q, want 'call_123'", result.ToolCalls[0].ID)
		}
		if result.ToolCalls[0].Name != "get_weather" {
			t.Errorf("ToolCalls[0].Name = %q", result.ToolCalls[0].Name)
		}
	})

	t.Run("handles empty content", func(t *testing.T) {
		choice := openailib.ChatCompletionChoice{
			Message: openailib.ChatCompletionMessage{
				Role:    "assistant",
				Content: "",
			},
		}

		result := fromOpenAIMessage(choice)

		if result.Role != chat.RoleAssistant {
			t.Errorf("Role = %v, want assistant", result.Role)
		}
		if len(result.Contents) != 0 {
			t.Errorf("len(Contents) = %d, want 0 for empty content", len(result.Contents))
		}
	})
}

func TestFromOpenAIToolCalls(t *testing.T) {
	t.Run("converts tool calls", func(t *testing.T) {
		calls := []openailib.ToolCall{
			{
				ID:   "call_1",
				Type: openailib.ToolTypeFunction,
				Function: openailib.FunctionCall{
					Name:      "func1",
					Arguments: `{"a":1}`,
				},
			},
			{
				ID:   "call_2",
				Type: openailib.ToolTypeFunction,
				Function: openailib.FunctionCall{
					Name:      "func2",
					Arguments: `{"b":2}`,
				},
			},
		}

		result := fromOpenAIToolCalls(calls)

		if len(result) != 2 {
			t.Fatalf("len(result) = %d, want 2", len(result))
		}
		if result[0].ID != "call_1" {
			t.Errorf("result[0].ID = %q, want 'call_1'", result[0].ID)
		}
		if result[0].Name != "func1" {
			t.Errorf("result[0].Name = %q, want 'func1'", result[0].Name)
		}
		if string(result[0].Arguments) != `{"a":1}` {
			t.Errorf("result[0].Arguments = %q", string(result[0].Arguments))
		}
	})
}

func TestFromOpenAIFinishReason(t *testing.T) {
	tests := []struct {
		reason   string
		expected chat.FinishReason
	}{
		{"stop", chat.FinishReasonStop},
		{"length", chat.FinishReasonLength},
		{"tool_calls", chat.FinishReasonToolCalls},
		{"function_call", chat.FinishReasonToolCalls},
		{"content_filter", chat.FinishReasonContentFilter},
		{"unknown", chat.FinishReasonStop},
		{"", chat.FinishReasonStop},
	}

	for _, tt := range tests {
		t.Run(tt.reason, func(t *testing.T) {
			result := fromOpenAIFinishReason(tt.reason)
			if result != tt.expected {
				t.Errorf("fromOpenAIFinishReason(%q) = %v, want %v", tt.reason, result, tt.expected)
			}
		})
	}
}

func TestMessageConversionRoundTrip(t *testing.T) {
	// Test that messages can be converted to OpenAI format and back
	t.Run("text message round trip", func(t *testing.T) {
		original := chat.Message{
			Role:     chat.RoleUser,
			Contents: []chat.Content{chat.NewTextContent("Hello world")},
		}

		// Convert to OpenAI
		oaiMsg := toOpenAIMessage(original, "system")

		// Simulate what OpenAI would return
		choice := openailib.ChatCompletionChoice{
			Message: openailib.ChatCompletionMessage{
				Role:    oaiMsg.Role,
				Content: oaiMsg.Content,
			},
		}

		// Convert back
		result := fromOpenAIMessage(choice)

		if result.Role != original.Role {
			t.Errorf("Role = %v, want %v", result.Role, original.Role)
		}
		if result.Text() != "Hello world" {
			t.Errorf("Text() = %q, want 'Hello world'", result.Text())
		}
	})
}
