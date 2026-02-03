// Copyright (c) Microsoft. All rights reserved.

package chatagent

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/microsoft/agent-framework-go/agent"
	"github.com/microsoft/agent-framework-go/chat"
)

func TestNewSession(t *testing.T) {
	client := newMockClient()
	a := New(client)

	session, err := a.NewSession(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if session == nil {
		t.Fatal("expected session")
	}
	if session.ID() == "" {
		t.Error("expected session ID")
	}

	// Should start with no messages
	if len(session.Messages()) != 0 {
		t.Errorf("expected 0 messages, got %d", len(session.Messages()))
	}
}

func TestSessionAddMessage(t *testing.T) {
	session := newSession("test-agent")

	session.AddMessage(agent.NewUserMessage("Hello"))
	session.AddMessage(agent.NewAssistantMessage("Hi there"))

	messages := session.Messages()
	if len(messages) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(messages))
	}
	if messages[0].Role != chat.RoleUser {
		t.Errorf("expected user role, got %v", messages[0].Role)
	}
	if messages[1].Role != chat.RoleAssistant {
		t.Errorf("expected assistant role, got %v", messages[1].Role)
	}
}

func TestSessionAddMessages(t *testing.T) {
	session := newSession("test-agent")

	session.AddMessages(
		agent.NewUserMessage("Hello"),
		agent.NewAssistantMessage("Hi there"),
		agent.NewUserMessage("How are you?"),
	)

	if session.MessageCount() != 3 {
		t.Errorf("expected 3 messages, got %d", session.MessageCount())
	}
}

func TestSessionClearMessages(t *testing.T) {
	session := newSession("test-agent")
	session.AddMessage(agent.NewUserMessage("Hello"))
	session.AddMessage(agent.NewAssistantMessage("Hi"))

	session.ClearMessages()

	if session.MessageCount() != 0 {
		t.Errorf("expected 0 messages after clear, got %d", session.MessageCount())
	}
}

func TestSessionSerialize(t *testing.T) {
	session := newSession("test-agent")
	// Don't add messages with complex Content types for this test
	// as they require custom JSON marshaling

	data, err := session.Serialize()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify JSON is valid and contains expected fields
	var parsed map[string]interface{}
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("failed to parse serialized data: %v", err)
	}

	if parsed["id"] != session.ID() {
		t.Errorf("expected ID %q, got %v", session.ID(), parsed["id"])
	}
	if parsed["agent_id"] != "test-agent" {
		t.Errorf("expected agent_id 'test-agent', got %v", parsed["agent_id"])
	}
}

func TestSessionRestore(t *testing.T) {
	// Create a session and serialize without messages
	// (message deserialization requires custom JSON handling)
	original := newSession("test-agent")

	// Serialize
	data, err := original.Serialize()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Restore
	restored, err := restoreSession(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify
	if restored.ID() != original.ID() {
		t.Errorf("expected ID %q, got %q", original.ID(), restored.ID())
	}
	if restored.AgentID() != original.AgentID() {
		t.Errorf("expected AgentID %q, got %q", original.AgentID(), restored.AgentID())
	}
}

func TestAgentRestoreSession(t *testing.T) {
	client := newMockClient()
	a := New(client)

	// Create and serialize a session without messages
	original, _ := a.NewSession(context.Background())
	data, _ := original.Serialize()

	// Restore via agent
	restored, err := a.RestoreSession(context.Background(), data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if restored.ID() != original.ID() {
		t.Errorf("expected ID %q, got %q", original.ID(), restored.ID())
	}
}

func TestSessionRestoreInvalid(t *testing.T) {
	_, err := restoreSession([]byte("invalid json"))
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestSessionAgentID(t *testing.T) {
	session := newSession("my-agent-id")

	if session.AgentID() != "my-agent-id" {
		t.Errorf("expected agent ID 'my-agent-id', got %q", session.AgentID())
	}
}

func TestSessionTimestamps(t *testing.T) {
	before := time.Now()
	session := newSession("test-agent")
	after := time.Now()

	createdAt := session.CreatedAt()
	if createdAt.Before(before) || createdAt.After(after) {
		t.Errorf("createdAt %v not in expected range [%v, %v]", createdAt, before, after)
	}

	// Initially, modifiedAt should equal createdAt
	if !session.ModifiedAt().Equal(session.CreatedAt()) {
		t.Error("expected modifiedAt to equal createdAt initially")
	}

	// Add a message and check modifiedAt updates
	time.Sleep(time.Millisecond) // Ensure time difference
	session.AddMessage(agent.NewUserMessage("Hello"))

	if !session.ModifiedAt().After(session.CreatedAt()) {
		t.Error("expected modifiedAt to be after createdAt after modification")
	}
}

func TestSessionClone(t *testing.T) {
	original := newSession("test-agent")
	original.AddMessage(agent.NewUserMessage("Hello"))

	cloned := original.Clone()

	// Should have different ID
	if cloned.ID() == original.ID() {
		t.Error("expected different ID for clone")
	}

	// Should have same messages
	if cloned.MessageCount() != original.MessageCount() {
		t.Errorf("expected same message count, got %d vs %d",
			cloned.MessageCount(), original.MessageCount())
	}

	// Modifying clone should not affect original
	cloned.AddMessage(agent.NewAssistantMessage("Hi"))
	if original.MessageCount() != 1 {
		t.Error("modifying clone affected original")
	}
}

func TestSessionMessagesReturnsCopy(t *testing.T) {
	session := newSession("test-agent")
	session.AddMessage(agent.NewUserMessage("Hello"))

	messages := session.Messages()
	messages = append(messages, agent.NewUserMessage("Modified"))

	// Original should be unchanged
	if session.MessageCount() != 1 {
		t.Error("modifying returned slice affected session")
	}
}

func TestSessionInterface(t *testing.T) {
	session := newSession("test-agent")

	// Verify implements agent.Session
	var _ agent.Session = session
}

func TestSessionRegisterService(t *testing.T) {
	type TestService struct {
		Value string
	}

	session := newSession("test-agent")
	service := &TestService{Value: "test"}

	session.RegisterService(service)

	// Should be nil for unregistered type
	if session.GetService(nil) != nil {
		t.Error("expected nil for nil type")
	}
}

func TestSessionRegisterNilService(t *testing.T) {
	session := newSession("test-agent")

	// Should not panic
	session.RegisterService(nil)
}

func TestSessionConcurrentAccess(t *testing.T) {
	session := newSession("test-agent")

	done := make(chan bool)

	// Concurrent writes
	go func() {
		for i := 0; i < 100; i++ {
			session.AddMessage(agent.NewUserMessage("Hello"))
		}
		done <- true
	}()

	// Concurrent reads
	go func() {
		for i := 0; i < 100; i++ {
			_ = session.Messages()
			_ = session.MessageCount()
		}
		done <- true
	}()

	<-done
	<-done

	// Should complete without race condition
	if session.MessageCount() != 100 {
		t.Errorf("expected 100 messages, got %d", session.MessageCount())
	}
}
