// Copyright (c) Microsoft. All rights reserved.

package durable

import (
	"encoding/json"
	"testing"

	"github.com/microsoft/agent-framework-go/chat"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSession(t *testing.T) {
	sessionID := NewSessionID("my-agent", "user-123")

	session := NewSession(sessionID)

	assert.Equal(t, sessionID.String(), session.ID())
	assert.Equal(t, sessionID, session.SessionID())
	assert.Empty(t, session.Messages())
	assert.Equal(t, 0, session.MessageCount())
}

func TestNewSessionWithState(t *testing.T) {
	sessionID := NewSessionID("my-agent", "user-123")
	state := NewState()
	state.AppendRequest(NewRequestEntry([]chat.Message{chat.NewUserMessage("test")}))

	session := NewSessionWithState(sessionID, state)

	assert.Len(t, session.Messages(), 1)
	assert.Equal(t, 1, session.MessageCount())
}

func TestSession_AddMessage(t *testing.T) {
	session := NewSession(NewSessionID("agent", "key"))

	// Add a user message
	session.AddMessage(chat.NewUserMessage("Hello"))

	messages := session.Messages()
	require.Len(t, messages, 1)
	assert.Equal(t, chat.RoleUser, messages[0].Role)
	assert.Equal(t, "Hello", messages[0].Text())
}

func TestSession_AddMessage_DifferentRoles(t *testing.T) {
	session := NewSession(NewSessionID("agent", "key"))

	// Add messages with different roles
	session.AddMessage(chat.NewSystemMessage("You are helpful"))
	session.AddMessage(chat.NewUserMessage("Hello"))
	session.AddMessage(chat.NewAssistantMessage("Hi there"))

	messages := session.Messages()
	assert.Len(t, messages, 3)
	assert.Equal(t, chat.RoleSystem, messages[0].Role)
	assert.Equal(t, chat.RoleUser, messages[1].Role)
	assert.Equal(t, chat.RoleAssistant, messages[2].Role)
}

func TestSession_AppendRequest(t *testing.T) {
	session := NewSession(NewSessionID("agent", "key"))

	session.AppendRequest([]chat.Message{
		chat.NewUserMessage("Hello"),
		chat.NewUserMessage("World"),
	}, "corr-123")

	assert.Equal(t, 2, session.MessageCount())
}

func TestSession_AppendResponse(t *testing.T) {
	session := NewSession(NewSessionID("agent", "key"))
	usage := &UsageInfo{InputTokenCount: 10, OutputTokenCount: 20, TotalTokenCount: 30}

	session.AppendResponse([]chat.Message{
		chat.NewAssistantMessage("Response"),
	}, usage)

	assert.Equal(t, 1, session.MessageCount())
}

func TestSession_Serialize(t *testing.T) {
	session := NewSession(NewSessionID("agent", "key"))
	session.AddMessage(chat.NewUserMessage("Hello"))
	session.AddMessage(chat.NewAssistantMessage("Hi"))

	data, err := session.Serialize()

	require.NoError(t, err)
	assert.NotEmpty(t, data)

	// Verify the JSON structure
	var parsed map[string]interface{}
	err = json.Unmarshal(data, &parsed)
	require.NoError(t, err)
	assert.Equal(t, SchemaVersion, parsed["schemaVersion"])
}

func TestSession_State(t *testing.T) {
	session := NewSession(NewSessionID("agent", "key"))
	session.AddMessage(chat.NewUserMessage("Hello"))

	state := session.State()

	require.NotNil(t, state)
	assert.Equal(t, SchemaVersion, state.SchemaVersion)
	assert.Len(t, state.Data.ConversationHistory, 1)
}

func TestRestoreSession(t *testing.T) {
	sessionID := NewSessionID("my-agent", "user-123")
	originalSession := NewSession(sessionID)
	originalSession.AddMessage(chat.NewUserMessage("Hello"))
	originalSession.AddMessage(chat.NewAssistantMessage("Hi there"))

	// Serialize
	data, err := originalSession.Serialize()
	require.NoError(t, err)

	// Restore
	restored, err := RestoreSession(sessionID, data)

	require.NoError(t, err)
	assert.Equal(t, sessionID.String(), restored.ID())
	assert.Equal(t, 2, restored.MessageCount())
}

func TestSession_GetService(t *testing.T) {
	session := NewSession(NewSessionID("agent", "key"))

	// No service registered
	service := session.GetService(nil)
	assert.Nil(t, service)
}

func TestSession_ConcurrentAccess(t *testing.T) {
	session := NewSession(NewSessionID("agent", "key"))

	// Run concurrent operations
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func(n int) {
			session.AddMessage(chat.NewUserMessage("Message"))
			_ = session.Messages()
			_ = session.MessageCount()
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Verify no race conditions occurred
	assert.LessOrEqual(t, session.MessageCount(), 10)
}
