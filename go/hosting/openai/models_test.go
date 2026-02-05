// Copyright (c) Microsoft. All rights reserved.

package openai

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/microsoft/agent-framework-go/chat"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestToAgentMessages_SimpleText(t *testing.T) {
	// Arrange
	messages := []ChatCompletionMessage{
		{Role: "system", Content: "You are a helpful assistant"},
		{Role: "user", Content: "Hello"},
	}

	// Act
	result := ToAgentMessages(messages)

	// Assert
	require.Len(t, result, 2)
	assert.Equal(t, chat.RoleSystem, result[0].Role)
	assert.Equal(t, "You are a helpful assistant", result[0].Text())
	assert.Equal(t, chat.RoleUser, result[1].Role)
	assert.Equal(t, "Hello", result[1].Text())
}

func TestToAgentMessages_WithToolCalls(t *testing.T) {
	// Arrange
	messages := []ChatCompletionMessage{
		{
			Role:    "assistant",
			Content: nil,
			ToolCalls: []ToolCallMessage{
				{
					ID:   "call_123",
					Type: "function",
					Function: FunctionCall{
						Name:      "get_weather",
						Arguments: `{"location": "Seattle"}`,
					},
				},
			},
		},
	}

	// Act
	result := ToAgentMessages(messages)

	// Assert
	require.Len(t, result, 1)
	assert.Equal(t, chat.RoleAssistant, result[0].Role)
	require.Len(t, result[0].ToolCalls, 1)
	assert.Equal(t, "call_123", result[0].ToolCalls[0].ID)
	assert.Equal(t, "get_weather", result[0].ToolCalls[0].Name)
}

func TestToAgentMessages_ToolResult(t *testing.T) {
	// Arrange
	messages := []ChatCompletionMessage{
		{
			Role:       "tool",
			Content:    `{"temperature": 72}`,
			ToolCallID: "call_123",
		},
	}

	// Act
	result := ToAgentMessages(messages)

	// Assert
	require.Len(t, result, 1)
	assert.Equal(t, chat.RoleTool, result[0].Role)
	assert.Equal(t, "call_123", result[0].ToolCallID)
	assert.Equal(t, `{"temperature": 72}`, result[0].Text())
}

func TestToAgentMessages_MultiPartContent(t *testing.T) {
	// Arrange
	messages := []ChatCompletionMessage{
		{
			Role: "user",
			Content: []any{
				map[string]any{"type": "text", "text": "What's in this image?"},
				map[string]any{"type": "image_url", "image_url": map[string]any{"url": "https://example.com/image.jpg"}},
			},
		},
	}

	// Act
	result := ToAgentMessages(messages)

	// Assert
	require.Len(t, result, 1)
	require.Len(t, result[0].Contents, 2)

	textContent, ok := result[0].Contents[0].(*chat.TextContent)
	require.True(t, ok)
	assert.Equal(t, "What's in this image?", textContent.Text)

	imageContent, ok := result[0].Contents[1].(*chat.ImageContent)
	require.True(t, ok)
	assert.Equal(t, "https://example.com/image.jpg", imageContent.URL)
}

func TestFromAgentMessage_SimpleText(t *testing.T) {
	// Arrange
	msg := chat.Message{
		Role:      chat.RoleAssistant,
		Contents:  []chat.Content{chat.NewTextContent("Hello, world!")},
		CreatedAt: time.Now(),
	}

	// Act
	result := FromAgentMessage(msg)

	// Assert
	assert.Equal(t, "assistant", result.Role)
	assert.Equal(t, "Hello, world!", result.Content)
}

func TestFromAgentMessage_WithToolCalls(t *testing.T) {
	// Arrange
	msg := chat.Message{
		Role: chat.RoleAssistant,
		ToolCalls: []chat.ToolCall{
			{
				ID:        "call_456",
				Name:      "search",
				Arguments: json.RawMessage(`{"query": "golang"}`),
			},
		},
		CreatedAt: time.Now(),
	}

	// Act
	result := FromAgentMessage(msg)

	// Assert
	assert.Equal(t, "assistant", result.Role)
	require.Len(t, result.ToolCalls, 1)
	assert.Equal(t, "call_456", result.ToolCalls[0].ID)
	assert.Equal(t, "function", result.ToolCalls[0].Type)
	assert.Equal(t, "search", result.ToolCalls[0].Function.Name)
	assert.Equal(t, `{"query": "golang"}`, result.ToolCalls[0].Function.Arguments)
}

func TestFromAgentMessage_ToolResult(t *testing.T) {
	// Arrange
	msg := chat.Message{
		Role:       chat.RoleTool,
		Contents:   []chat.Content{chat.NewTextContent(`{"result": "success"}`)},
		ToolCallID: "call_789",
		CreatedAt:  time.Now(),
	}

	// Act
	result := FromAgentMessage(msg)

	// Assert
	assert.Equal(t, "tool", result.Role)
	assert.Equal(t, "call_789", result.ToolCallID)
	assert.Equal(t, `{"result": "success"}`, result.Content)
}

func TestFromAgentMessage_MultiContent(t *testing.T) {
	// Arrange
	msg := chat.Message{
		Role: chat.RoleUser,
		Contents: []chat.Content{
			chat.NewTextContent("Look at this:"),
			chat.NewImageContentFromURL("https://example.com/photo.png"),
		},
		CreatedAt: time.Now(),
	}

	// Act
	result := FromAgentMessage(msg)

	// Assert
	assert.Equal(t, "user", result.Role)

	parts, ok := result.Content.([]ContentPart)
	require.True(t, ok)
	require.Len(t, parts, 2)
	assert.Equal(t, "text", parts[0].Type)
	assert.Equal(t, "Look at this:", parts[0].Text)
	assert.Equal(t, "image_url", parts[1].Type)
	assert.Equal(t, "https://example.com/photo.png", parts[1].ImageURL.URL)
}

func TestRoundTripConversion(t *testing.T) {
	// Arrange
	original := []ChatCompletionMessage{
		{Role: "system", Content: "System prompt"},
		{Role: "user", Content: "User message"},
		{
			Role:    "assistant",
			Content: "I'll help you",
			ToolCalls: []ToolCallMessage{
				{
					ID:   "call_1",
					Type: "function",
					Function: FunctionCall{
						Name:      "helper",
						Arguments: `{}`,
					},
				},
			},
		},
		{Role: "tool", Content: "Tool result", ToolCallID: "call_1"},
	}

	// Act
	agentMsgs := ToAgentMessages(original)
	var roundTripped []ChatCompletionMessage
	for _, m := range agentMsgs {
		roundTripped = append(roundTripped, FromAgentMessage(m))
	}

	// Assert
	require.Len(t, roundTripped, 4)
	assert.Equal(t, "system", roundTripped[0].Role)
	assert.Equal(t, "System prompt", roundTripped[0].Content)

	assert.Equal(t, "user", roundTripped[1].Role)
	assert.Equal(t, "User message", roundTripped[1].Content)

	assert.Equal(t, "assistant", roundTripped[2].Role)
	require.Len(t, roundTripped[2].ToolCalls, 1)
	assert.Equal(t, "call_1", roundTripped[2].ToolCalls[0].ID)

	assert.Equal(t, "tool", roundTripped[3].Role)
	assert.Equal(t, "call_1", roundTripped[3].ToolCallID)
}
