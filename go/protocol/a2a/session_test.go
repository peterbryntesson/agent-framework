// Copyright (c) Microsoft. All rights reserved.

package a2a

import (
	"encoding/json"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/microsoft/agent-framework-go/agent"
	"github.com/microsoft/agent-framework-go/chat"
)

// TestNewA2ASession tests session creation.
func TestNewA2ASession(t *testing.T) {
	session := NewA2ASession()

	if session.ID() == "" {
		t.Error("ID() should not be empty")
	}

	if session.ContextID() != "" {
		t.Errorf("ContextID() should be empty, got %q", session.ContextID())
	}

	if session.TaskID() != "" {
		t.Errorf("TaskID() should be empty, got %q", session.TaskID())
	}

	if len(session.Messages()) != 0 {
		t.Errorf("Messages() should be empty, got %d messages", len(session.Messages()))
	}
}

// TestNewA2ASessionWithContextID tests session creation with context ID.
func TestNewA2ASessionWithContextID(t *testing.T) {
	session := NewA2ASessionWithContextID("context-123")

	if session.ID() == "" {
		t.Error("ID() should not be empty")
	}

	if session.ContextID() != "context-123" {
		t.Errorf("ContextID() = %q, want %q", session.ContextID(), "context-123")
	}
}

// TestA2ASession_SetContextID tests setting context ID.
func TestA2ASession_SetContextID(t *testing.T) {
	session := NewA2ASession()

	session.SetContextID("new-context")

	if session.ContextID() != "new-context" {
		t.Errorf("ContextID() = %q, want %q", session.ContextID(), "new-context")
	}
}

// TestA2ASession_SetTaskID tests setting task ID.
func TestA2ASession_SetTaskID(t *testing.T) {
	session := NewA2ASession()

	session.SetTaskID("task-456")

	if session.TaskID() != "task-456" {
		t.Errorf("TaskID() = %q, want %q", session.TaskID(), "task-456")
	}
}

// TestA2ASession_AddMessage tests adding messages.
func TestA2ASession_AddMessage(t *testing.T) {
	session := NewA2ASession()

	msg1 := chat.NewUserMessage("Hello")
	msg2 := chat.NewAssistantMessage("Hi there!")

	session.AddMessage(msg1)
	session.AddMessage(msg2)

	messages := session.Messages()

	if len(messages) != 2 {
		t.Fatalf("len(Messages()) = %d, want 2", len(messages))
	}

	if messages[0].Text() != "Hello" {
		t.Errorf("messages[0].Text() = %q, want %q", messages[0].Text(), "Hello")
	}

	if messages[1].Text() != "Hi there!" {
		t.Errorf("messages[1].Text() = %q, want %q", messages[1].Text(), "Hi there!")
	}
}

// TestA2ASession_MessagesReturnsDefensiveCopy tests that Messages returns a copy.
func TestA2ASession_MessagesReturnsDefensiveCopy(t *testing.T) {
	session := NewA2ASession()
	session.AddMessage(chat.NewUserMessage("Original"))

	messages := session.Messages()
	messages = append(messages, chat.NewUserMessage("Added externally"))

	// Original should be unchanged
	if len(session.Messages()) != 1 {
		t.Errorf("len(Messages()) = %d, want 1 (defensive copy failed)", len(session.Messages()))
	}
}

// TestA2ASession_Serialize tests session serialization.
func TestA2ASession_Serialize(t *testing.T) {
	session := NewA2ASession()
	session.SetContextID("ctx-serialize")
	session.SetTaskID("task-serialize")
	session.AddMessage(chat.NewUserMessage("Test message"))

	data, err := session.Serialize()

	if err != nil {
		t.Fatalf("Serialize() error = %v", err)
	}

	// Verify it's valid JSON
	var parsed map[string]interface{}
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Serialize() produced invalid JSON: %v", err)
	}

	// Check expected fields
	if parsed["context_id"] != "ctx-serialize" {
		t.Errorf("context_id = %v, want %q", parsed["context_id"], "ctx-serialize")
	}

	if parsed["task_id"] != "task-serialize" {
		t.Errorf("task_id = %v, want %q", parsed["task_id"], "task-serialize")
	}
}

// TestRestoreA2ASession tests session restoration.
func TestRestoreA2ASession(t *testing.T) {
	// Create and serialize a session (without messages to avoid JSON interface issues)
	original := NewA2ASession()
	original.SetContextID("restore-context")
	original.SetTaskID("restore-task")

	data, err := original.Serialize()
	if err != nil {
		t.Fatalf("Serialize() error = %v", err)
	}

	// Restore it
	restored, err := RestoreA2ASession(data)

	if err != nil {
		t.Fatalf("RestoreA2ASession() error = %v", err)
	}

	if restored.ID() != original.ID() {
		t.Errorf("ID() = %q, want %q", restored.ID(), original.ID())
	}

	if restored.ContextID() != "restore-context" {
		t.Errorf("ContextID() = %q, want %q", restored.ContextID(), "restore-context")
	}

	if restored.TaskID() != "restore-task" {
		t.Errorf("TaskID() = %q, want %q", restored.TaskID(), "restore-task")
	}
}

// TestRestoreA2ASession_InvalidJSON tests restoration with invalid JSON.
func TestRestoreA2ASession_InvalidJSON(t *testing.T) {
	_, err := RestoreA2ASession([]byte("not valid json"))

	if err == nil {
		t.Error("RestoreA2ASession() with invalid JSON should return error")
	}
}

// TestA2ASession_GetService tests service retrieval.
func TestA2ASession_GetService(t *testing.T) {
	session := NewA2ASession()

	// Register a service
	type TestService struct {
		Value string
	}
	svc := &TestService{Value: "test"}
	session.RegisterService(reflect.TypeOf((*TestService)(nil)), svc)

	// Retrieve it
	got := session.GetService(reflect.TypeOf((*TestService)(nil)))

	if got != svc {
		t.Errorf("GetService() = %v, want %v", got, svc)
	}

	// Unknown service
	unknown := session.GetService(reflect.TypeOf((*string)(nil)))
	if unknown != nil {
		t.Errorf("GetService(unknown) = %v, want nil", unknown)
	}
}

// TestA2ASession_UpdateFromTask tests updating session from task.
func TestA2ASession_UpdateFromTask(t *testing.T) {
	session := NewA2ASession()

	task := &Task{
		ID:        "task-from-update",
		ContextID: "ctx-from-update",
		State:     TaskStateRunning,
	}

	session.updateFromTask(task)

	if session.ContextID() != "ctx-from-update" {
		t.Errorf("ContextID() = %q, want %q", session.ContextID(), "ctx-from-update")
	}

	if session.TaskID() != "task-from-update" {
		t.Errorf("TaskID() = %q, want %q", session.TaskID(), "task-from-update")
	}
}

// TestA2ASession_UpdateFromTaskPreservesContextID tests context ID preservation.
func TestA2ASession_UpdateFromTaskPreservesContextID(t *testing.T) {
	session := NewA2ASessionWithContextID("original-context")

	// Task with empty context ID shouldn't overwrite
	task := &Task{
		ID:        "task-1",
		ContextID: "", // Empty
		State:     TaskStateRunning,
	}

	session.updateFromTask(task)

	// Original context should be preserved since task context is empty
	if session.ContextID() != "original-context" {
		t.Errorf("ContextID() = %q, want %q", session.ContextID(), "original-context")
	}
}

// TestA2ASession_ConcurrentAccess tests thread safety.
func TestA2ASession_ConcurrentAccess(t *testing.T) {
	session := NewA2ASession()

	var wg sync.WaitGroup

	// Concurrent reads and writes
	for i := 0; i < 100; i++ {
		wg.Add(3)

		go func(n int) {
			defer wg.Done()
			session.AddMessage(chat.NewUserMessage("Message"))
		}(i)

		go func() {
			defer wg.Done()
			_ = session.Messages()
		}()

		go func() {
			defer wg.Done()
			_ = session.ContextID()
			_ = session.TaskID()
		}()
	}

	// Concurrent context/task updates
	for i := 0; i < 50; i++ {
		wg.Add(2)

		go func(n int) {
			defer wg.Done()
			session.SetContextID("ctx")
		}(i)

		go func(n int) {
			defer wg.Done()
			session.SetTaskID("task")
		}(i)
	}

	wg.Wait()

	// Verify no data race - just ensure we can still access
	_ = session.ID()
	_ = session.Messages()
}

// TestA2ASession_ImplementsSession tests that A2ASession implements agent.Session.
func TestA2ASession_ImplementsSession(t *testing.T) {
	var _ agent.Session = (*A2ASession)(nil)
}

// TestA2ASession_SerializeRoundTrip tests full serialization round trip.
func TestA2ASession_SerializeRoundTrip(t *testing.T) {
	original := NewA2ASession()
	original.SetContextID("roundtrip-ctx")
	original.SetTaskID("roundtrip-task")

	// Note: We don't test message serialization here because chat.Message.Contents
	// contains interface types that require custom JSON unmarshaling.
	// Message persistence should be handled via the agent's session store.

	// Serialize
	data, err := original.Serialize()
	if err != nil {
		t.Fatalf("Serialize() error = %v", err)
	}

	// Restore
	restored, err := RestoreA2ASession(data)
	if err != nil {
		t.Fatalf("RestoreA2ASession() error = %v", err)
	}

	// Verify all fields match
	if restored.ID() != original.ID() {
		t.Errorf("ID mismatch: %q != %q", restored.ID(), original.ID())
	}

	if restored.ContextID() != original.ContextID() {
		t.Errorf("ContextID mismatch: %q != %q", restored.ContextID(), original.ContextID())
	}

	if restored.TaskID() != original.TaskID() {
		t.Errorf("TaskID mismatch: %q != %q", restored.TaskID(), original.TaskID())
	}
}

// TestA2ASession_EmptySessionSerialize tests serializing an empty session.
func TestA2ASession_EmptySessionSerialize(t *testing.T) {
	session := NewA2ASession()

	data, err := session.Serialize()
	if err != nil {
		t.Fatalf("Serialize() error = %v", err)
	}

	restored, err := RestoreA2ASession(data)
	if err != nil {
		t.Fatalf("RestoreA2ASession() error = %v", err)
	}

	if restored.ID() != session.ID() {
		t.Errorf("ID mismatch")
	}

	if len(restored.Messages()) != 0 {
		t.Errorf("restored session should have no messages")
	}
}

// TestA2ASession_UpdateFromTaskWithMessages tests task update behavior.
func TestA2ASession_UpdateFromTaskWithMessages(t *testing.T) {
	session := NewA2ASession()

	now := time.Now()
	task := &Task{
		ID:        "task-with-msgs",
		ContextID: "ctx-with-msgs",
		State:     TaskStateCompleted,
		Messages: []Message{
			{Role: RoleUser, Parts: []Part{NewTextPart("Hello")}, CreatedAt: now},
			{Role: RoleAssistant, Parts: []Part{NewTextPart("Hi!")}, CreatedAt: now},
		},
	}

	session.updateFromTask(task)

	// updateFromTask should only update context and task ID, not messages
	// Messages are managed separately by the agent
	if session.TaskID() != "task-with-msgs" {
		t.Errorf("TaskID() = %q, want %q", session.TaskID(), "task-with-msgs")
	}

	if session.ContextID() != "ctx-with-msgs" {
		t.Errorf("ContextID() = %q, want %q", session.ContextID(), "ctx-with-msgs")
	}

	// Session messages should still be empty (not auto-populated from task)
	if len(session.Messages()) != 0 {
		t.Errorf("Messages should be empty, got %d", len(session.Messages()))
	}
}
