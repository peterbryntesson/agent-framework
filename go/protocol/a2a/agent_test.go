// Copyright (c) Microsoft. All rights reserved.

package a2a

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/microsoft/agent-framework-go/agent"
	"github.com/microsoft/agent-framework-go/chat"
)

// TestNewA2AAgent tests agent creation with various options.
func TestNewA2AAgent(t *testing.T) {
	tests := []struct {
		name        string
		opts        []A2AAgentOption
		wantName    string
		wantDesc    string
		checkIDFunc func(id string) bool
	}{
		{
			name:        "default options",
			opts:        nil,
			wantName:    "",
			wantDesc:    "",
			checkIDFunc: func(id string) bool { return id != "" },
		},
		{
			name:        "with custom ID",
			opts:        []A2AAgentOption{WithAgentID("custom-id")},
			wantName:    "",
			wantDesc:    "",
			checkIDFunc: func(id string) bool { return id == "custom-id" },
		},
		{
			name:        "with name and description",
			opts:        []A2AAgentOption{WithAgentName("TestAgent"), WithAgentDescription("A test agent")},
			wantName:    "TestAgent",
			wantDesc:    "A test agent",
			checkIDFunc: func(id string) bool { return id != "" },
		},
		{
			name: "with all options",
			opts: []A2AAgentOption{
				WithAgentID("my-agent"),
				WithAgentName("MyAgent"),
				WithAgentDescription("My agent description"),
			},
			wantName:    "MyAgent",
			wantDesc:    "My agent description",
			checkIDFunc: func(id string) bool { return id == "my-agent" },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewClient("http://localhost:8080")
			a := NewA2AAgent(client, tt.opts...)

			if !tt.checkIDFunc(a.ID()) {
				t.Errorf("ID() = %q, check failed", a.ID())
			}
			if a.Name() != tt.wantName {
				t.Errorf("Name() = %q, want %q", a.Name(), tt.wantName)
			}
			if a.Description() != tt.wantDesc {
				t.Errorf("Description() = %q, want %q", a.Description(), tt.wantDesc)
			}
		})
	}
}

// TestA2AAgent_Metadata tests the Metadata method.
func TestA2AAgent_Metadata(t *testing.T) {
	client := NewClient("http://localhost:8080")
	a := NewA2AAgent(client)

	meta := a.Metadata()

	if meta.ProviderName != "a2a" {
		t.Errorf("Metadata().ProviderName = %q, want %q", meta.ProviderName, "a2a")
	}
}

// TestA2AAgent_GetService tests service retrieval.
func TestA2AAgent_GetService(t *testing.T) {
	client := NewClient("http://localhost:8080")
	a := NewA2AAgent(client)

	// Test getting the client
	got := a.GetService(reflect.TypeOf((*Client)(nil)))
	if got != client {
		t.Errorf("GetService(Client) = %v, want %v", got, client)
	}

	// Test getting unknown service
	unknownType := reflect.TypeOf((*string)(nil))
	if got := a.GetService(unknownType); got != nil {
		t.Errorf("GetService(unknown) = %v, want nil", got)
	}
}

// TestA2AAgent_NewSession tests session creation.
func TestA2AAgent_NewSession(t *testing.T) {
	client := NewClient("http://localhost:8080")
	a := NewA2AAgent(client)

	session, err := a.NewSession(context.Background())

	if err != nil {
		t.Fatalf("NewSession() error = %v", err)
	}

	if session == nil {
		t.Fatal("NewSession() returned nil")
	}

	if session.ID() == "" {
		t.Error("session.ID() is empty")
	}

	// Verify it's an A2ASession
	a2aSession, ok := session.(*A2ASession)
	if !ok {
		t.Fatalf("session is not *A2ASession, got %T", session)
	}

	if a2aSession.ContextID() != "" {
		t.Errorf("new session should have empty ContextID, got %q", a2aSession.ContextID())
	}
}

// TestA2AAgent_RestoreSession tests session restoration.
func TestA2AAgent_RestoreSession(t *testing.T) {
	client := NewClient("http://localhost:8080")
	a := NewA2AAgent(client)

	// Create and serialize a session
	original := NewA2ASessionWithContextID("ctx-123")
	original.SetTaskID("task-456")

	data, err := original.Serialize()
	if err != nil {
		t.Fatalf("Serialize() error = %v", err)
	}

	// Restore the session
	restored, err := a.RestoreSession(context.Background(), data)
	if err != nil {
		t.Fatalf("RestoreSession() error = %v", err)
	}

	restoredA2A, ok := restored.(*A2ASession)
	if !ok {
		t.Fatalf("restored session is not *A2ASession")
	}

	if restoredA2A.ContextID() != "ctx-123" {
		t.Errorf("ContextID() = %q, want %q", restoredA2A.ContextID(), "ctx-123")
	}

	if restoredA2A.TaskID() != "task-456" {
		t.Errorf("TaskID() = %q, want %q", restoredA2A.TaskID(), "task-456")
	}
}

// TestA2AAgent_Run tests the Run method with a mock server.
func TestA2AAgent_Run(t *testing.T) {
	// Create a mock A2A server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/tasks":
			// Create task endpoint
			task := Task{
				ID:        "task-123",
				ContextID: "ctx-456",
				State:     TaskStateCompleted,
				Messages: []Message{
					NewAssistantMessage("Hello! How can I help you?"),
				},
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(task)

		case r.Method == http.MethodPost && strings.HasPrefix(r.URL.Path, "/tasks/") && strings.HasSuffix(r.URL.Path, "/messages"):
			// Send message endpoint
			task := Task{
				ID:        "task-123",
				ContextID: "ctx-456",
				State:     TaskStateCompleted,
				Messages: []Message{
					NewAssistantMessage("I received your follow-up!"),
				},
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(task)

		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client := NewClient(server.URL)
	a := NewA2AAgent(client, WithAgentID("test-agent"))

	t.Run("successful run with new session", func(t *testing.T) {
		messages := []agent.Message{
			chat.NewUserMessage("Hello!"),
		}

		resp, err := a.Run(context.Background(), messages)

		if err != nil {
			t.Fatalf("Run() error = %v", err)
		}

		if resp == nil {
			t.Fatal("Run() returned nil response")
		}

		if resp.AgentID != "test-agent" {
			t.Errorf("AgentID = %q, want %q", resp.AgentID, "test-agent")
		}

		if resp.ResponseID != "task-123" {
			t.Errorf("ResponseID = %q, want %q", resp.ResponseID, "task-123")
		}

		if len(resp.Messages) != 1 {
			t.Fatalf("len(Messages) = %d, want 1", len(resp.Messages))
		}

		if resp.Messages[0].Text() != "Hello! How can I help you?" {
			t.Errorf("Message text = %q, want %q", resp.Messages[0].Text(), "Hello! How can I help you?")
		}
	})

	t.Run("run with existing session", func(t *testing.T) {
		session := NewA2ASessionWithContextID("ctx-456")
		session.SetTaskID("task-123")

		messages := []agent.Message{
			chat.NewUserMessage("Follow-up message"),
		}

		resp, err := a.Run(context.Background(), messages, agent.WithSession(session))

		if err != nil {
			t.Fatalf("Run() error = %v", err)
		}

		if len(resp.Messages) != 1 {
			t.Fatalf("len(Messages) = %d, want 1", len(resp.Messages))
		}

		if resp.Messages[0].Text() != "I received your follow-up!" {
			t.Errorf("Message text = %q", resp.Messages[0].Text())
		}
	})

	t.Run("run with empty messages", func(t *testing.T) {
		_, err := a.Run(context.Background(), nil)

		if err == nil {
			t.Error("Run() with empty messages should return error")
		}
	})
}

// TestA2AAgent_RunStream tests the RunStream method.
func TestA2AAgent_RunStream(t *testing.T) {
	// Create a mock A2A server with SSE streaming
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/tasks":
			// Create task endpoint
			task := Task{
				ID:        "task-stream-1",
				ContextID: "ctx-stream",
				State:     TaskStatePending,
				CreatedAt: time.Now(),
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(task)

		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/messages/stream"):
			// SSE streaming endpoint
			w.Header().Set("Content-Type", "text/event-stream")
			w.Header().Set("Cache-Control", "no-cache")
			flusher, ok := w.(http.Flusher)
			if !ok {
				http.Error(w, "SSE not supported", http.StatusInternalServerError)
				return
			}

			// Send a message event
			msg := Message{
				Role:      RoleAssistant,
				Parts:     []Part{NewTextPart("Streaming response!")},
				CreatedAt: time.Now(),
			}
			msgJSON, _ := json.Marshal(msg)
			fmt.Fprintf(w, "event:message\n")
			fmt.Fprintf(w, "data:%s\n\n", msgJSON)
			flusher.Flush()

			// Send done event
			fmt.Fprintf(w, "event:done\n")
			fmt.Fprintf(w, "data:{}\n\n")
			flusher.Flush()

		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client := NewClient(server.URL)
	a := NewA2AAgent(client)

	t.Run("successful streaming", func(t *testing.T) {
		messages := []agent.Message{
			chat.NewUserMessage("Hello stream!"),
		}

		updates, err := a.RunStream(context.Background(), messages)

		if err != nil {
			t.Fatalf("RunStream() error = %v", err)
		}

		var receivedUpdates []agent.ResponseUpdate
		for update := range updates {
			receivedUpdates = append(receivedUpdates, update)
		}

		if len(receivedUpdates) == 0 {
			t.Fatal("no updates received")
		}

		// Should have at least a message complete and done
		hasMessage := false
		hasDone := false
		for _, u := range receivedUpdates {
			if u.Kind == agent.UpdateKindMessageComplete {
				hasMessage = true
			}
			if u.Kind == agent.UpdateKindDone {
				hasDone = true
			}
		}

		if !hasMessage {
			t.Error("expected MessageComplete update")
		}
		if !hasDone {
			t.Error("expected Done update")
		}
	})

	t.Run("stream with empty messages", func(t *testing.T) {
		_, err := a.RunStream(context.Background(), nil)

		if err == nil {
			t.Error("RunStream() with empty messages should return error")
		}
	})

	t.Run("stream with context cancellation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		messages := []agent.Message{
			chat.NewUserMessage("Hello!"),
		}

		_, err := a.RunStream(ctx, messages)

		if err == nil {
			t.Error("RunStream() with cancelled context should return error")
		}
	})
}

// TestA2AAgent_FetchAgentCard tests fetching the agent card.
func TestA2AAgent_FetchAgentCard(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/.well-known/agent.json" {
			card := AgentCard{
				Name:        "Remote Agent",
				Description: "A remote A2A agent",
				URL:         "http://example.com",
				Version:     "1.0.0",
				Capabilities: &AgentCapabilities{
					Streaming: true,
				},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(card)
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	client := NewClient(server.URL)
	a := NewA2AAgent(client)

	card, err := a.FetchAgentCard(context.Background())

	if err != nil {
		t.Fatalf("FetchAgentCard() error = %v", err)
	}

	if card.Name != "Remote Agent" {
		t.Errorf("card.Name = %q, want %q", card.Name, "Remote Agent")
	}

	// Verify card is cached
	if a.Name() != "Remote Agent" {
		t.Errorf("Name() after fetch = %q, want %q", a.Name(), "Remote Agent")
	}

	if a.Description() != "A remote A2A agent" {
		t.Errorf("Description() after fetch = %q", a.Description())
	}
}

// TestA2AAgent_NameAndDescriptionFromCard tests name/description fallback to card.
func TestA2AAgent_NameAndDescriptionFromCard(t *testing.T) {
	client := NewClient("http://localhost:8080")
	a := NewA2AAgent(client)

	// Before fetching card, should return empty
	if a.Name() != "" {
		t.Errorf("Name() before card = %q, want empty", a.Name())
	}

	// Manually set the card
	a.cardMu.Lock()
	a.card = &AgentCard{
		Name:        "Card Agent",
		Description: "From the card",
	}
	a.cardMu.Unlock()

	// Now should return card values
	if a.Name() != "Card Agent" {
		t.Errorf("Name() with card = %q, want %q", a.Name(), "Card Agent")
	}

	if a.Description() != "From the card" {
		t.Errorf("Description() with card = %q, want %q", a.Description(), "From the card")
	}

	// Explicit name/description should override card
	a2 := NewA2AAgent(client, WithAgentName("Override"), WithAgentDescription("Override desc"))
	a2.cardMu.Lock()
	a2.card = &AgentCard{
		Name:        "Card Agent",
		Description: "From the card",
	}
	a2.cardMu.Unlock()

	if a2.Name() != "Override" {
		t.Errorf("Name() with explicit = %q, want %q", a2.Name(), "Override")
	}

	if a2.Description() != "Override desc" {
		t.Errorf("Description() with explicit = %q, want %q", a2.Description(), "Override desc")
	}
}

// TestA2AAgent_RunWithInvalidSession tests Run with wrong session type.
func TestA2AAgent_RunWithInvalidSession(t *testing.T) {
	client := NewClient("http://localhost:8080")
	a := NewA2AAgent(client)

	// Use a different session type
	wrongSession := agent.NewInMemorySession()

	messages := []agent.Message{
		chat.NewUserMessage("Hello!"),
	}

	_, err := a.Run(context.Background(), messages, agent.WithSession(wrongSession))

	if err == nil {
		t.Error("Run() with wrong session type should return error")
	}

	if !strings.Contains(err.Error(), "invalid session type") {
		t.Errorf("error should mention invalid session type, got: %v", err)
	}
}

// TestA2AAgent_ConvertMessagesToA2A tests message conversion.
func TestA2AAgent_ConvertMessagesToA2A(t *testing.T) {
	client := NewClient("http://localhost:8080")
	a := NewA2AAgent(client)

	tests := []struct {
		name     string
		messages []agent.Message
		wantRole MessageRole
		wantText string
	}{
		{
			name:     "user message",
			messages: []agent.Message{chat.NewUserMessage("Hello!")},
			wantRole: RoleUser,
			wantText: "Hello!",
		},
		{
			name:     "assistant message",
			messages: []agent.Message{chat.NewAssistantMessage("Hi there!")},
			wantRole: RoleAssistant,
			wantText: "Hi there!",
		},
		{
			name:     "system message",
			messages: []agent.Message{chat.NewSystemMessage("You are helpful.")},
			wantRole: RoleSystem,
			wantText: "You are helpful.",
		},
		{
			name: "multiple messages uses last",
			messages: []agent.Message{
				chat.NewUserMessage("First"),
				chat.NewUserMessage("Second"),
				chat.NewUserMessage("Third"),
			},
			wantRole: RoleUser,
			wantText: "Third",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := a.convertMessagesToA2A(tt.messages)

			if msg == nil {
				t.Fatal("convertMessagesToA2A returned nil")
			}

			if msg.Role != tt.wantRole {
				t.Errorf("Role = %v, want %v", msg.Role, tt.wantRole)
			}

			if msg.Text() != tt.wantText {
				t.Errorf("Text() = %q, want %q", msg.Text(), tt.wantText)
			}
		})
	}
}

// TestA2AAgent_ConvertTaskToResponse tests task to response conversion.
func TestA2AAgent_ConvertTaskToResponse(t *testing.T) {
	client := NewClient("http://localhost:8080")
	a := NewA2AAgent(client, WithAgentID("test-agent"))

	tests := []struct {
		name             string
		task             *Task
		wantMsgCount     int
		wantFinishReason agent.FinishReason
	}{
		{
			name: "completed task",
			task: &Task{
				ID:    "task-1",
				State: TaskStateCompleted,
				Messages: []Message{
					NewUserMessage("Hello"),
					NewAssistantMessage("Hi!"),
				},
			},
			wantMsgCount:     1, // Only assistant messages
			wantFinishReason: agent.FinishReasonStop,
		},
		{
			name: "failed task",
			task: &Task{
				ID:    "task-2",
				State: TaskStateFailed,
				Error: &TaskError{Code: "ERROR", Message: "Something went wrong"},
			},
			wantMsgCount:     0,
			wantFinishReason: agent.FinishReasonStop,
		},
		{
			name: "task with multiple assistant messages",
			task: &Task{
				ID:    "task-3",
				State: TaskStateCompleted,
				Messages: []Message{
					NewUserMessage("Q1"),
					NewAssistantMessage("A1"),
					NewUserMessage("Q2"),
					NewAssistantMessage("A2"),
				},
			},
			wantMsgCount:     2,
			wantFinishReason: agent.FinishReasonStop,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := a.convertTaskToResponse(tt.task)

			if resp.AgentID != "test-agent" {
				t.Errorf("AgentID = %q, want %q", resp.AgentID, "test-agent")
			}

			if resp.ResponseID != tt.task.ID {
				t.Errorf("ResponseID = %q, want %q", resp.ResponseID, tt.task.ID)
			}

			if len(resp.Messages) != tt.wantMsgCount {
				t.Errorf("len(Messages) = %d, want %d", len(resp.Messages), tt.wantMsgCount)
			}

			if resp.FinishReason != tt.wantFinishReason {
				t.Errorf("FinishReason = %v, want %v", resp.FinishReason, tt.wantFinishReason)
			}
		})
	}
}

// TestA2AAgent_ConcurrentAccess tests thread safety.
func TestA2AAgent_ConcurrentAccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		task := Task{
			ID:       "task-concurrent",
			State:    TaskStateCompleted,
			Messages: []Message{NewAssistantMessage("OK")},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(task)
	}))
	defer server.Close()

	client := NewClient(server.URL)
	a := NewA2AAgent(client)

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			messages := []agent.Message{
				chat.NewUserMessage("Concurrent request"),
			}

			_, err := a.Run(context.Background(), messages)
			if err != nil {
				t.Errorf("concurrent Run() error: %v", err)
			}
		}()
	}

	wg.Wait()
}

// Verify A2AAgent implements agent.Agent at compile time.
func TestA2AAgent_ImplementsAgent(t *testing.T) {
	var _ agent.Agent = (*A2AAgent)(nil)
}
