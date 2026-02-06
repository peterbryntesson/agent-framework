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

func TestNewState(t *testing.T) {
	state := NewState()

	assert.Equal(t, SchemaVersion, state.SchemaVersion)
	assert.NotNil(t, state.Data.ConversationHistory)
	assert.Empty(t, state.Data.ConversationHistory)
	assert.Nil(t, state.Data.ExpirationTimeUtc)
}

func TestState_AppendRequest(t *testing.T) {
	state := NewState()

	messages := []chat.Message{
		chat.NewUserMessage("Hello"),
	}
	entry := NewRequestEntry(messages)
	state.AppendRequest(entry)

	assert.Len(t, state.Data.ConversationHistory, 1)
	assert.Equal(t, "request", state.Data.ConversationHistory[0].EntryType())
}

func TestState_AppendResponse(t *testing.T) {
	state := NewState()

	messages := []chat.Message{
		chat.NewAssistantMessage("Hi there!"),
	}
	entry := NewResponseEntry(messages)
	state.AppendResponse(entry)

	assert.Len(t, state.Data.ConversationHistory, 1)
	assert.Equal(t, "response", state.Data.ConversationHistory[0].EntryType())
}

func TestState_BuildChatMessages(t *testing.T) {
	state := NewState()

	// Add a request
	reqMessages := []chat.Message{
		chat.NewUserMessage("Hello"),
	}
	state.AppendRequest(NewRequestEntry(reqMessages))

	// Add a response
	respMessages := []chat.Message{
		chat.NewAssistantMessage("Hi there!"),
	}
	state.AppendResponse(NewResponseEntry(respMessages))

	// Build chat messages
	messages := state.BuildChatMessages()

	assert.Len(t, messages, 2)
	assert.Equal(t, chat.RoleUser, messages[0].Role)
	assert.Equal(t, chat.RoleAssistant, messages[1].Role)
}

func TestState_BuildChatMessages_SkipsErrorResponses(t *testing.T) {
	state := NewState()

	state.AppendRequest(NewRequestEntry([]chat.Message{chat.NewUserMessage("Hello")}))
	state.AppendResponse(&ResponseEntry{
		Type:      "response",
		Timestamp: time.Now().UTC(),
		Messages: []StateMessage{
			{
				Role: "assistant",
				Contents: []ContentItem{
					{Type: ContentTypeText, Text: "Error response"},
				},
			},
		},
		IsError: true,
	})
	state.AppendResponse(NewResponseEntry([]chat.Message{chat.NewAssistantMessage("Hi")}))

	messages := state.BuildChatMessages()

	assert.Len(t, messages, 2)
	assert.Equal(t, chat.RoleUser, messages[0].Role)
	assert.Equal(t, chat.RoleAssistant, messages[1].Role)
}

func TestState_SetExpiration(t *testing.T) {
	state := NewState()
	expTime := time.Now().Add(1 * time.Hour)

	state.SetExpiration(expTime)

	require.NotNil(t, state.Data.ExpirationTimeUtc)
	assert.Equal(t, expTime.Unix(), state.Data.ExpirationTimeUtc.Unix())
}

func TestState_ClearExpiration(t *testing.T) {
	state := NewState()
	state.SetExpiration(time.Now().Add(1 * time.Hour))

	state.ClearExpiration()

	assert.Nil(t, state.Data.ExpirationTimeUtc)
}

func TestState_IsExpired(t *testing.T) {
	t.Run("no expiration", func(t *testing.T) {
		state := NewState()
		assert.False(t, state.IsExpired())
	})

	t.Run("expired", func(t *testing.T) {
		state := NewState()
		state.SetExpiration(time.Now().Add(-1 * time.Hour))
		assert.True(t, state.IsExpired())
	})

	t.Run("not expired", func(t *testing.T) {
		state := NewState()
		state.SetExpiration(time.Now().Add(1 * time.Hour))
		assert.False(t, state.IsExpired())
	})
}

func TestState_Clone(t *testing.T) {
	state := NewState()
	state.AppendRequest(NewRequestEntry([]chat.Message{chat.NewUserMessage("test")}))
	state.SetExpiration(time.Now().Add(1 * time.Hour))

	clone := state.Clone()

	assert.Equal(t, state.SchemaVersion, clone.SchemaVersion)
	assert.Len(t, clone.Data.ConversationHistory, 1)
	require.NotNil(t, clone.Data.ExpirationTimeUtc)

	// Ensure it's a deep copy
	state.AppendResponse(NewResponseEntry([]chat.Message{chat.NewAssistantMessage("reply")}))
	assert.Len(t, clone.Data.ConversationHistory, 1)
}

func TestState_MessageCount(t *testing.T) {
	state := NewState()

	assert.Equal(t, 0, state.MessageCount())

	state.AppendRequest(NewRequestEntry([]chat.Message{chat.NewUserMessage("test")}))
	assert.Equal(t, 1, state.MessageCount())

	state.AppendResponse(NewResponseEntry([]chat.Message{
		chat.NewAssistantMessage("reply1"),
		chat.NewAssistantMessage("reply2"),
	}))
	assert.Equal(t, 3, state.MessageCount())
}

func TestState_JSONSerialization(t *testing.T) {
	state := NewState()
	state.AppendRequest(NewRequestEntry([]chat.Message{chat.NewUserMessage("test")}))
	expTime := time.Now().Add(1 * time.Hour).UTC()
	state.SetExpiration(expTime)

	// Serialize
	data, err := json.Marshal(state)
	require.NoError(t, err)

	// Deserialize
	var restored State
	err = json.Unmarshal(data, &restored)
	require.NoError(t, err)

	assert.Equal(t, state.SchemaVersion, restored.SchemaVersion)
	assert.Len(t, restored.Data.ConversationHistory, 1)
	require.NotNil(t, restored.Data.ExpirationTimeUtc)
}
