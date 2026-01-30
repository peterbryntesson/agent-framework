// Copyright (c) Microsoft. All rights reserved.

package agent

import (
	"encoding/json"
	"reflect"
	"sync"
	"testing"

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
	msg := Message{Role: "user", Content: "Hello, world!"}

	// Act
	session.AddMessage(msg)

	// Assert
	messages := session.Messages()
	require.Len(t, messages, 1)
	assert.Equal(t, "user", messages[0].Role)
	assert.Equal(t, "Hello, world!", messages[0].Content)
}

func TestInMemorySession_AddMessage_PreservesOrder(t *testing.T) {
	// Arrange
	session := NewInMemorySession()
	msg1 := Message{Role: "user", Content: "First message"}
	msg2 := Message{Role: "assistant", Content: "Second message"}
	msg3 := Message{Role: "user", Content: "Third message"}

	// Act
	session.AddMessage(msg1)
	session.AddMessage(msg2)
	session.AddMessage(msg3)

	// Assert
	messages := session.Messages()
	require.Len(t, messages, 3)
	assert.Equal(t, "First message", messages[0].Content)
	assert.Equal(t, "Second message", messages[1].Content)
	assert.Equal(t, "Third message", messages[2].Content)
}

func TestInMemorySession_Messages_ReturnsCopy(t *testing.T) {
	// Arrange
	session := NewInMemorySession()
	session.AddMessage(Message{Role: "user", Content: "Original"})

	// Act
	messages := session.Messages()
	messages[0] = Message{Role: "modified", Content: "Modified"}

	// Assert - original should be unchanged
	originalMessages := session.Messages()
	assert.Equal(t, "Original", originalMessages[0].Content)
}

func TestInMemorySession_Serialize_ProducesValidJSON(t *testing.T) {
	// Arrange
	session := NewInMemorySessionWithID("test-id-456")
	session.AddMessage(Message{Role: "user", Content: "Hello"})
	session.AddMessage(Message{Role: "assistant", Content: "Hi there"})

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
	// Arrange
	original := NewInMemorySessionWithID("restore-test-id")
	original.AddMessage(Message{Role: "user", Content: "Saved message"})
	original.AddMessage(Message{Role: "assistant", Content: "Saved response"})
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
			session.AddMessage(Message{Role: "user", Content: "Message"})
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
			session.AddMessage(Message{Role: "user", Content: "Write"})
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
	// Arrange
	original := NewInMemorySession()
	original.AddMessage(Message{Role: "system", Content: "You are a helpful assistant."})
	original.AddMessage(Message{Role: "user", Content: "Hello!"})
	original.AddMessage(Message{Role: "assistant", Content: "Hello! How can I help you?"})
	original.AddMessage(Message{Role: "user", Content: "What is 2+2?"})
	original.AddMessage(Message{Role: "assistant", Content: "2+2 equals 4."})

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
		assert.Equal(t, originalMsgs[i].Content, restoredMsgs[i].Content)
	}
}
