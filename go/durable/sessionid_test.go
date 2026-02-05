// Copyright (c) Microsoft. All rights reserved.

package durable

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSessionID(t *testing.T) {
	sessionID := NewSessionID("my-agent", "user-123")

	assert.Equal(t, "my-agent", sessionID.Name)
	assert.Equal(t, "user-123", sessionID.Key)
}

func TestSessionID_WorkflowID(t *testing.T) {
	sessionID := NewSessionID("my-agent", "user-123")

	workflowID := sessionID.WorkflowID()

	assert.Equal(t, "dafx-my-agent-user-123", workflowID)
}

func TestSessionID_EntityName(t *testing.T) {
	sessionID := NewSessionID("my-agent", "user-123")

	entityName := sessionID.EntityName()

	assert.Equal(t, "dafx-my-agent", entityName)
}

func TestParseSessionID(t *testing.T) {
	t.Run("valid workflow ID", func(t *testing.T) {
		sessionID, err := ParseSessionID("dafx-myagent-user123")

		require.NoError(t, err)
		assert.Equal(t, "myagent", sessionID.Name)
		assert.Equal(t, "user123", sessionID.Key)
	})

	t.Run("workflow ID with dashes in key", func(t *testing.T) {
		sessionID, err := ParseSessionID("dafx-agent-user-123-extra")

		require.NoError(t, err)
		assert.Equal(t, "agent", sessionID.Name)
		assert.Equal(t, "user-123-extra", sessionID.Key)
	})

	t.Run("invalid prefix", func(t *testing.T) {
		_, err := ParseSessionID("invalid-workflow-id")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "missing prefix")
	})

	t.Run("missing separator", func(t *testing.T) {
		_, err := ParseSessionID("dafx-nokey")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "missing separator")
	})

	t.Run("empty name", func(t *testing.T) {
		_, err := ParseSessionID("dafx--key")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "empty name or key")
	})
}

func TestSessionID_String(t *testing.T) {
	sessionID := NewSessionID("my-agent", "user-123")

	assert.Equal(t, "dafx-my-agent-user-123", sessionID.String())
}

func TestSessionID_IsZero(t *testing.T) {
	t.Run("zero session ID", func(t *testing.T) {
		var sessionID SessionID
		assert.True(t, sessionID.IsZero())
	})

	t.Run("non-zero session ID", func(t *testing.T) {
		sessionID := NewSessionID("agent", "key")
		assert.False(t, sessionID.IsZero())
	})

	t.Run("partial session ID - name only", func(t *testing.T) {
		sessionID := SessionID{Name: "agent"}
		assert.False(t, sessionID.IsZero())
	})
}

func TestSessionID_Equals(t *testing.T) {
	t.Run("equal session IDs", func(t *testing.T) {
		id1 := NewSessionID("agent", "key")
		id2 := NewSessionID("agent", "key")
		assert.True(t, id1.Equals(id2))
	})

	t.Run("case-insensitive name comparison", func(t *testing.T) {
		id1 := NewSessionID("Agent", "key")
		id2 := NewSessionID("agent", "key")
		assert.True(t, id1.Equals(id2))
	})

	t.Run("case-sensitive key comparison", func(t *testing.T) {
		id1 := NewSessionID("agent", "Key")
		id2 := NewSessionID("agent", "key")
		assert.False(t, id1.Equals(id2))
	})

	t.Run("different names", func(t *testing.T) {
		id1 := NewSessionID("agent1", "key")
		id2 := NewSessionID("agent2", "key")
		assert.False(t, id1.Equals(id2))
	})
}

func TestSessionID_JSONSerialization(t *testing.T) {
	sessionID := NewSessionID("myagent", "user123")

	// Marshal
	data, err := sessionID.MarshalJSON()
	require.NoError(t, err)
	assert.Equal(t, `"dafx-myagent-user123"`, string(data))

	// Unmarshal
	var restored SessionID
	err = restored.UnmarshalJSON(data)
	require.NoError(t, err)
	assert.Equal(t, sessionID.Name, restored.Name)
	assert.Equal(t, sessionID.Key, restored.Key)
}

func TestSessionID_TextMarshaling(t *testing.T) {
	sessionID := NewSessionID("myagent", "user123")

	// MarshalText
	data, err := sessionID.MarshalText()
	require.NoError(t, err)
	assert.Equal(t, "dafx-myagent-user123", string(data))

	// UnmarshalText
	var restored SessionID
	err = restored.UnmarshalText(data)
	require.NoError(t, err)
	assert.Equal(t, sessionID.Name, restored.Name)
	assert.Equal(t, sessionID.Key, restored.Key)
}
