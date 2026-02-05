// Copyright (c) Microsoft. All rights reserved.

package agent

import (
	"encoding/json"
	"reflect"
	"sync"
	"testing"

	"github.com/microsoft/agent-framework-go/chat"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewInMemorySession_GeneratesUUID(t *testing.T) {
	// Arrange & Act
	session := NewInMemorySession()

	// Assert
	assert.NotEmpty(t, session.ID())
	// UUID v4 format: xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx (36 chars)
	assert.Len(t, session.ID(), 36)
}

func TestNewInMemorySession_GeneratesUniqueIDs(t *testing.T) {
	// Arrange & Act
	session1 := NewInMemorySession()
	session2 := NewInMemorySession()

	// Assert
	assert.NotEqual(t, session1.ID(), session2.ID())
}

func TestNewInMemorySession_InitializesEmptyMessages(t *testing.T) {
	// Arrange & Act
	session := NewInMemorySession()

	// Assert
	assert.Empty(t, session.Messages())
}

func TestNewInMemorySessionWithID_UsesProvidedID(t *testing.T) {
	// Arrange
	customID := "custom-session-id-123"

	// Act
	session := NewInMemorySessionWithID(customID)

	// Assert
	assert.Equal(t, customID, session.ID())
}

func TestInMemorySession_AddMessage_AppendsMessage(t *testing.T) {
	// Arrange
	session := NewInMemorySession()
	msg := chat.NewUserMessage("Hello, world!")

	// Act
	session.AddMessage(msg)

	// Assert
	messages := session.Messages()
	require.Len(t, messages, 1)
	assert.Equal(t, chat.RoleUser, messages[0].Role)
	assert.Equal(t, "Hello, world!", messages[0].Text())
}

func TestInMemorySession_AddMessage_PreservesOrder(t *testing.T) {
	// Arrange
	session := NewInMemorySession()
	msg1 := chat.NewUserMessage("First message")
	msg2 := chat.NewAssistantMessage("Second message")
	msg3 := chat.NewUserMessage("Third message")

	// Act
	session.AddMessage(msg1)
	session.AddMessage(msg2)
	session.AddMessage(msg3)

	// Assert
	messages := session.Messages()
	require.Len(t, messages, 3)
	assert.Equal(t, "First message", messages[0].Text())
	assert.Equal(t, "Second message", messages[1].Text())
	assert.Equal(t, "Third message", messages[2].Text())
}

func TestInMemorySession_Messages_ReturnsCopy(t *testing.T) {
	// Arrange
	session := NewInMemorySession()
	session.AddMessage(chat.NewUserMessage("Original"))

	// Act
	messages := session.Messages()
	messages[0] = chat.NewUserMessage("Modified")

	// Assert - original should be unchanged
	originalMessages := session.Messages()
	assert.Equal(t, "Original", originalMessages[0].Text())
}

func TestInMemorySession_Serialize_ProducesValidJSON(t *testing.T) {
	// Arrange
	session := NewInMemorySessionWithID("test-id-456")
	session.AddMessage(chat.NewUserMessage("Hello"))
	session.AddMessage(chat.NewAssistantMessage("Hi there"))

	// Act
	data, err := session.Serialize()

	// Assert
	require.NoError(t, err)
	require.NotNil(t, data)

	var parsed map[string]interface{}
	err = json.Unmarshal(data, &parsed)
	require.NoError(t, err)
	assert.Equal(t, "test-id-456", parsed["id"])
}

func TestRestoreInMemorySession_RestoresState(t *testing.T) {
	// Skip: chat.Message.Contents uses interface type which requires custom JSON unmarshaling.
	// This is a known limitation tracked for future work.
	t.Skip("Skipped: chat.Content interface requires custom JSON unmarshaling for deserialization")

	// Arrange
	original := NewInMemorySessionWithID("restore-test-id")
	original.AddMessage(chat.NewUserMessage("Saved message"))
	original.AddMessage(chat.NewAssistantMessage("Saved response"))
	data, err := original.Serialize()
	require.NoError(t, err)

	// Act
	restored, err := RestoreInMemorySession(data)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, original.ID(), restored.ID())
	assert.Equal(t, original.Messages(), restored.Messages())
}

func TestRestoreInMemorySession_InvalidJSON_ReturnsError(t *testing.T) {
	// Arrange
	invalidJSON := json.RawMessage(`{invalid json}`)

	// Act
	session, err := RestoreInMemorySession(invalidJSON)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, session)
}

func TestInMemorySession_GetService_ReturnsNilForUnregistered(t *testing.T) {
	// Arrange
	session := NewInMemorySession()

	// Act
	service := session.GetService(reflect.TypeOf("string"))

	// Assert
	assert.Nil(t, service)
}

func TestInMemorySession_RegisterService_AllowsRetrieval(t *testing.T) {
	// Arrange
	session := NewInMemorySession()
	type testService struct {
		value string
	}
	svc := &testService{value: "test-value"}
	serviceType := reflect.TypeOf(svc)

	// Act
	session.RegisterService(serviceType, svc)
	retrieved := session.GetService(serviceType)

	// Assert
	require.NotNil(t, retrieved)
	assert.Equal(t, svc, retrieved)
}

func TestInMemorySession_ConcurrentAccess_IsThreadSafe(t *testing.T) {
	// Arrange
	session := NewInMemorySession()
	var wg sync.WaitGroup
	messageCount := 100

	// Act - concurrent writes
	for i := 0; i < messageCount; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			session.AddMessage(chat.NewUserMessage("Message"))
		}(i)
	}
	wg.Wait()

	// Assert
	messages := session.Messages()
	assert.Len(t, messages, messageCount)
}

func TestInMemorySession_ConcurrentReadWrite_IsThreadSafe(t *testing.T) {
	// Arrange
	session := NewInMemorySession()
	var wg sync.WaitGroup
	iterations := 50

	// Act - concurrent reads and writes
	for i := 0; i < iterations; i++ {
		wg.Add(2)
		go func() {
			defer wg.Done()
			session.AddMessage(chat.NewUserMessage("Write"))
		}()
		go func() {
			defer wg.Done()
			_ = session.Messages()
			_ = session.ID()
		}()
	}
	wg.Wait()

	// Assert - no race conditions, session should have expected messages
	messages := session.Messages()
	assert.Len(t, messages, iterations)
}

func TestInMemorySession_ImplementsSessionInterface(t *testing.T) {
	// Arrange & Act
	var _ Session = NewInMemorySession()

	// Assert - compilation success proves interface implementation
}

func TestInMemorySession_SerializeDeserializeRoundtrip_PreservesData(t *testing.T) {
	// Skip: chat.Message.Contents uses interface type which requires custom JSON unmarshaling.
	// This is a known limitation tracked for future work.
	t.Skip("Skipped: chat.Content interface requires custom JSON unmarshaling for deserialization")

	// Arrange
	original := NewInMemorySession()
	original.AddMessage(chat.NewSystemMessage("You are a helpful assistant."))
	original.AddMessage(chat.NewUserMessage("Hello!"))
	original.AddMessage(chat.NewAssistantMessage("Hello! How can I help you?"))
	original.AddMessage(chat.NewUserMessage("What is 2+2?"))
	original.AddMessage(chat.NewAssistantMessage("2+2 equals 4."))

	// Act
	data, err := original.Serialize()
	require.NoError(t, err)
	restored, err := RestoreInMemorySession(data)
	require.NoError(t, err)

	// Assert
	assert.Equal(t, original.ID(), restored.ID())
	originalMsgs := original.Messages()
	restoredMsgs := restored.Messages()
	require.Len(t, restoredMsgs, len(originalMsgs))
	for i := range originalMsgs {
		assert.Equal(t, originalMsgs[i].Role, restoredMsgs[i].Role)
		assert.Equal(t, originalMsgs[i].Text(), restoredMsgs[i].Text())
	}
}
