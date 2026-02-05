// Copyright (c) Microsoft. All rights reserved.

package durable

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/microsoft/agent-framework-go/chat"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRequestEntry(t *testing.T) {
	messages := []chat.Message{
		chat.NewUserMessage("Hello"),
	}

	entry := NewRequestEntry(messages)

	assert.Equal(t, "request", entry.Type)
	assert.Equal(t, "request", entry.EntryType())
	assert.Len(t, entry.Messages, 1)
	assert.False(t, entry.Timestamp.IsZero())
}

func TestNewRequestEntryWithCorrelation(t *testing.T) {
	messages := []chat.Message{
		chat.NewUserMessage("Hello"),
	}

	entry := NewRequestEntryWithCorrelation(messages, "corr-123")

	assert.Equal(t, "corr-123", entry.CorrelationID)
}

func TestNewResponseEntry(t *testing.T) {
	messages := []chat.Message{
		chat.NewAssistantMessage("Hi there!"),
	}

	entry := NewResponseEntry(messages)

	assert.Equal(t, "response", entry.Type)
	assert.Equal(t, "response", entry.EntryType())
	assert.Len(t, entry.Messages, 1)
	assert.False(t, entry.Timestamp.IsZero())
}

func TestNewResponseEntryWithUsage(t *testing.T) {
	messages := []chat.Message{
		chat.NewAssistantMessage("Hi there!"),
	}
	usage := &UsageInfo{
		InputTokenCount:  10,
		OutputTokenCount: 20,
		TotalTokenCount:  30,
	}

	entry := NewResponseEntryWithUsage(messages, usage)

	require.NotNil(t, entry.Usage)
	assert.Equal(t, 10, entry.Usage.InputTokenCount)
	assert.Equal(t, 20, entry.Usage.OutputTokenCount)
	assert.Equal(t, 30, entry.Usage.TotalTokenCount)
}

func TestRequestEntry_ToChatMessages(t *testing.T) {
	messages := []chat.Message{
		chat.NewUserMessage("Hello"),
		chat.NewUserMessage("World"),
	}
	entry := NewRequestEntry(messages)

	result := entry.ToChatMessages()

	assert.Len(t, result, 2)
	assert.Equal(t, chat.RoleUser, result[0].Role)
	assert.Equal(t, "Hello", result[0].Text())
	assert.Equal(t, "World", result[1].Text())
}

func TestResponseEntry_ToChatMessages(t *testing.T) {
	messages := []chat.Message{
		chat.NewAssistantMessage("Response 1"),
		chat.NewAssistantMessage("Response 2"),
	}
	entry := NewResponseEntry(messages)

	result := entry.ToChatMessages()

	assert.Len(t, result, 2)
	assert.Equal(t, chat.RoleAssistant, result[0].Role)
}

func TestRequestEntry_MessageCount(t *testing.T) {
	entry := NewRequestEntry([]chat.Message{
		chat.NewUserMessage("one"),
		chat.NewUserMessage("two"),
		chat.NewUserMessage("three"),
	})

	assert.Equal(t, 3, entry.MessageCount())
}

func TestResponseEntry_MessageCount(t *testing.T) {
	entry := NewResponseEntry([]chat.Message{
		chat.NewAssistantMessage("one"),
		chat.NewAssistantMessage("two"),
	})

	assert.Equal(t, 2, entry.MessageCount())
}

func TestFromChatMessage(t *testing.T) {
	t.Run("user message with text", func(t *testing.T) {
		msg := chat.NewUserMessage("Hello world")

		stateMsg := FromChatMessage(msg)

		assert.Equal(t, "user", stateMsg.Role)
		require.Len(t, stateMsg.Contents, 1)
		assert.Equal(t, ContentTypeText, stateMsg.Contents[0].Type)
		assert.Equal(t, "Hello world", stateMsg.Contents[0].Text)
	})

	t.Run("assistant message", func(t *testing.T) {
		msg := chat.NewAssistantMessage("Hi there")

		stateMsg := FromChatMessage(msg)

		assert.Equal(t, "assistant", stateMsg.Role)
	})

	t.Run("system message", func(t *testing.T) {
		msg := chat.NewSystemMessage("You are a helpful assistant")

		stateMsg := FromChatMessage(msg)

		assert.Equal(t, "system", stateMsg.Role)
	})

	t.Run("message with tool calls", func(t *testing.T) {
		msg := chat.Message{
			Role: chat.RoleAssistant,
			ToolCalls: []chat.ToolCall{
				{
					ID:        "call_123",
					Name:      "get_weather",
					Arguments: json.RawMessage(`{"city":"London"}`),
				},
			},
		}

		stateMsg := FromChatMessage(msg)

		require.Len(t, stateMsg.Contents, 1)
		assert.Equal(t, ContentTypeFunctionCall, stateMsg.Contents[0].Type)
		assert.Equal(t, "call_123", stateMsg.Contents[0].CallID)
		assert.Equal(t, "get_weather", stateMsg.Contents[0].FunctionName)
	})
}

func TestStateMessage_ToChatMessage(t *testing.T) {
	t.Run("text message", func(t *testing.T) {
		stateMsg := StateMessage{
			Role: "user",
			Contents: []ContentItem{
				{Type: ContentTypeText, Text: "Hello"},
			},
		}

		msg := stateMsg.ToChatMessage()

		assert.Equal(t, chat.RoleUser, msg.Role)
		assert.Equal(t, "Hello", msg.Text())
	})

	t.Run("message with function call", func(t *testing.T) {
		stateMsg := StateMessage{
			Role: "assistant",
			Contents: []ContentItem{
				{
					Type:         ContentTypeFunctionCall,
					CallID:       "call_456",
					FunctionName: "search",
					Arguments:    `{"query":"test"}`,
				},
			},
		}

		msg := stateMsg.ToChatMessage()

		assert.Equal(t, chat.RoleAssistant, msg.Role)
		require.Len(t, msg.ToolCalls, 1)
		assert.Equal(t, "call_456", msg.ToolCalls[0].ID)
		assert.Equal(t, "search", msg.ToolCalls[0].Name)
	})

	t.Run("message with author name", func(t *testing.T) {
		stateMsg := StateMessage{
			Role:       "user",
			AuthorName: "John",
			Contents: []ContentItem{
				{Type: ContentTypeText, Text: "Hi"},
			},
		}

		msg := stateMsg.ToChatMessage()

		assert.Equal(t, "John", msg.Name)
	})

	t.Run("message with createdAt", func(t *testing.T) {
		createdAt := time.Now()
		stateMsg := StateMessage{
			Role:      "user",
			CreatedAt: &createdAt,
			Contents: []ContentItem{
				{Type: ContentTypeText, Text: "Hi"},
			},
		}

		msg := stateMsg.ToChatMessage()

		assert.Equal(t, createdAt.Unix(), msg.CreatedAt.Unix())
	})
}

func TestContentItem_ToChatContent(t *testing.T) {
	t.Run("text content", func(t *testing.T) {
		item := ContentItem{Type: ContentTypeText, Text: "Hello"}

		content := item.ToChatContent()

		require.NotNil(t, content)
		textContent, ok := content.(*chat.TextContent)
		require.True(t, ok)
		assert.Equal(t, "Hello", textContent.Text)
	})

	t.Run("function result content", func(t *testing.T) {
		item := ContentItem{
			Type:   ContentTypeFunctionResult,
			CallID: "call_123",
			Result: json.RawMessage(`"result value"`),
		}

		content := item.ToChatContent()

		require.NotNil(t, content)
		resultContent, ok := content.(*chat.ToolResultContent)
		require.True(t, ok)
		assert.Equal(t, "call_123", resultContent.ToolCallID)
	})

	t.Run("function call returns nil", func(t *testing.T) {
		item := ContentItem{Type: ContentTypeFunctionCall}

		content := item.ToChatContent()

		assert.Nil(t, content)
	})
}

func TestFromChatContent(t *testing.T) {
	t.Run("text content", func(t *testing.T) {
		content := chat.NewTextContent("Hello world")

		item := FromChatContent(content)

		require.NotNil(t, item)
		assert.Equal(t, ContentTypeText, item.Type)
		assert.Equal(t, "Hello world", item.Text)
	})

	t.Run("tool result content", func(t *testing.T) {
		content := chat.NewToolResultContent("call_123", "result value")

		item := FromChatContent(content)

		require.NotNil(t, item)
		assert.Equal(t, ContentTypeFunctionResult, item.Type)
		assert.Equal(t, "call_123", item.CallID)
	})

	t.Run("nil content", func(t *testing.T) {
		item := FromChatContent(nil)

		assert.Nil(t, item)
	})
}

func TestUnmarshalStateEntry(t *testing.T) {
	t.Run("request entry", func(t *testing.T) {
		data := []byte(`{
			"$type": "request",
			"createdAt": "2024-01-01T00:00:00Z",
			"messages": []
		}`)

		entry, err := UnmarshalStateEntry(data)

		require.NoError(t, err)
		assert.Equal(t, "request", entry.EntryType())
	})

	t.Run("response entry", func(t *testing.T) {
		data := []byte(`{
			"$type": "response",
			"createdAt": "2024-01-01T00:00:00Z",
			"messages": []
		}`)

		entry, err := UnmarshalStateEntry(data)

		require.NoError(t, err)
		assert.Equal(t, "response", entry.EntryType())
	})

	t.Run("invalid JSON", func(t *testing.T) {
		data := []byte(`invalid`)

		_, err := UnmarshalStateEntry(data)

		assert.Error(t, err)
	})
}
