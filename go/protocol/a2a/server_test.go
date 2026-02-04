// Copyright (c) Microsoft. All rights reserved.

package a2a

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/microsoft/agent-framework-go/agent"
)

// mockAgent implements agent.Agent for testing.
type mockAgent struct {
	id          string
	name        string
	description string
	runFn       func(ctx context.Context, messages []agent.Message, opts ...agent.RunOption) (*agent.Response, error)
	runStreamFn func(ctx context.Context, messages []agent.Message, opts ...agent.RunOption) (<-chan agent.ResponseUpdate, error)
}

func (m *mockAgent) ID() string                      { return m.id }
func (m *mockAgent) Name() string                    { return m.name }
func (m *mockAgent) Description() string             { return m.description }
func (m *mockAgent) Metadata() agent.AIAgentMetadata { return agent.AIAgentMetadata{} }
func (m *mockAgent) GetService(reflect.Type) any     { return nil }
func (m *mockAgent) NewSession(context.Context) (agent.Session, error) {
	return nil, errors.New("not implemented")
}
func (m *mockAgent) RestoreSession(context.Context, json.RawMessage) (agent.Session, error) {
	return nil, errors.New("not implemented")
}
func (m *mockAgent) Run(ctx context.Context, messages []agent.Message, opts ...agent.RunOption) (*agent.Response, error) {
	if m.runFn != nil {
		return m.runFn(ctx, messages, opts...)
	}
	return &agent.Response{
		ResponseID: "resp-123",
		AgentID:    m.id,
		Messages:   []agent.Message{agent.NewAssistantMessage("Hello back!")},
		CreatedAt:  time.Now(),
	}, nil
}
func (m *mockAgent) RunStream(ctx context.Context, messages []agent.Message, opts ...agent.RunOption) (<-chan agent.ResponseUpdate, error) {
	if m.runStreamFn != nil {
		return m.runStreamFn(ctx, messages, opts...)
	}
	updates := make(chan agent.ResponseUpdate, 10)
	go func() {
		defer close(updates)
		msg := agent.NewAssistantMessage("Streaming response")
		updates <- agent.ResponseUpdate{
			Kind:    agent.UpdateKindMessageComplete,
			Message: &msg,
		}
		updates <- agent.ResponseUpdate{
			Kind:       agent.UpdateKindDone,
			ResponseID: "stream-resp-123",
		}
	}()
	return updates, nil
}

func TestNewServer(t *testing.T) {
	t.Run("creates server with default agent card", func(t *testing.T) {
		mockAg := &mockAgent{
			id:          "agent-1",
			name:        "TestAgent",
			description: "A test agent",
		}
		server := NewServer(mockAg)

		if server == nil {
			t.Fatal("expected server to be created")
		}
		if server.agent != mockAg {
			t.Error("expected agent to be set")
		}
		if server.card.Name != "TestAgent" {
			t.Errorf("expected card name TestAgent, got %s", server.card.Name)
		}
		if server.card.Description != "A test agent" {
			t.Errorf("expected card description, got %s", server.card.Description)
		}
		if server.card.Capabilities == nil || !server.card.Capabilities.Streaming {
			t.Error("expected streaming capability to be enabled")
		}
	})

	t.Run("creates server with custom agent card", func(t *testing.T) {
		mockAg := &mockAgent{id: "agent-1", name: "Default"}
		customCard := &AgentCard{
			Name:        "CustomAgent",
			Description: "Custom description",
			URL:         "http://example.com",
		}
		server := NewServer(mockAg, WithAgentCard(customCard))

		if server.card.Name != "CustomAgent" {
			t.Errorf("expected card name CustomAgent, got %s", server.card.Name)
		}
		if server.card.URL != "http://example.com" {
			t.Errorf("expected card URL http://example.com, got %s", server.card.URL)
		}
	})
}

func TestServer_Handler(t *testing.T) {
	t.Run("returns http.Handler", func(t *testing.T) {
		server := NewServer(&mockAgent{id: "agent-1"})
		handler := server.Handler()

		if handler == nil {
			t.Fatal("expected handler to be created")
		}
	})
}

func TestServer_GetAgentCard(t *testing.T) {
	t.Run("returns agent card as JSON", func(t *testing.T) {
		mockAg := &mockAgent{
			id:          "agent-1",
			name:        "TestAgent",
			description: "A test agent",
		}
		server := NewServer(mockAg)
		handler := server.Handler()

		req := httptest.NewRequest(http.MethodGet, "/.well-known/agent.json", nil)
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", w.Code)
		}

		contentType := w.Header().Get("Content-Type")
		if contentType != "application/json" {
			t.Errorf("expected Content-Type application/json, got %s", contentType)
		}

		var card AgentCard
		if err := json.NewDecoder(w.Body).Decode(&card); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if card.Name != "TestAgent" {
			t.Errorf("expected name TestAgent, got %s", card.Name)
		}
	})
}

func TestServer_CreateTask(t *testing.T) {
	t.Run("creates task successfully", func(t *testing.T) {
		mockAg := &mockAgent{id: "agent-1", name: "TestAgent"}
		server := NewServer(mockAg)
		handler := server.Handler()

		msg := NewUserMessage("Hello")
		reqBody, _ := json.Marshal(CreateTaskRequest{
			Message:   &msg,
			ContextID: "session-123",
		})

		req := httptest.NewRequest(http.MethodPost, "/tasks", bytes.NewReader(reqBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		if w.Code != http.StatusCreated {
			t.Errorf("expected status 201, got %d", w.Code)
		}

		var task Task
		if err := json.NewDecoder(w.Body).Decode(&task); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if task.ID == "" {
			t.Error("expected task ID to be set")
		}
		if task.ContextID != "session-123" {
			t.Errorf("expected context ID session-123, got %s", task.ContextID)
		}
		if task.State != TaskStatePending {
			t.Errorf("expected state pending, got %s", task.State)
		}
		if len(task.Messages) != 1 {
			t.Errorf("expected 1 message, got %d", len(task.Messages))
		}
	})

	t.Run("generates context ID if not provided", func(t *testing.T) {
		mockAg := &mockAgent{id: "agent-1"}
		server := NewServer(mockAg)
		handler := server.Handler()

		msg := NewUserMessage("Hello")
		reqBody, _ := json.Marshal(CreateTaskRequest{Message: &msg})

		req := httptest.NewRequest(http.MethodPost, "/tasks", bytes.NewReader(reqBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		var task Task
		json.NewDecoder(w.Body).Decode(&task)

		if task.ContextID == "" {
			t.Error("expected context ID to be generated")
		}
	})

	t.Run("returns error for missing message", func(t *testing.T) {
		mockAg := &mockAgent{id: "agent-1"}
		server := NewServer(mockAg)
		handler := server.Handler()

		reqBody, _ := json.Marshal(CreateTaskRequest{})
		req := httptest.NewRequest(http.MethodPost, "/tasks", bytes.NewReader(reqBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", w.Code)
		}
	})

	t.Run("returns error for invalid JSON", func(t *testing.T) {
		mockAg := &mockAgent{id: "agent-1"}
		server := NewServer(mockAg)
		handler := server.Handler()

		req := httptest.NewRequest(http.MethodPost, "/tasks", strings.NewReader("invalid json"))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", w.Code)
		}
	})

	t.Run("stores task in sessions", func(t *testing.T) {
		mockAg := &mockAgent{id: "agent-1"}
		server := NewServer(mockAg)
		handler := server.Handler()

		msg := NewUserMessage("Hello")
		reqBody, _ := json.Marshal(CreateTaskRequest{
			Message:   &msg,
			ContextID: "session-abc",
		})

		req := httptest.NewRequest(http.MethodPost, "/tasks", bytes.NewReader(reqBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		var task Task
		json.NewDecoder(w.Body).Decode(&task)

		sessionTasks := server.GetSessionTasks("session-abc")
		if len(sessionTasks) != 1 {
			t.Errorf("expected 1 task in session, got %d", len(sessionTasks))
		}
		if sessionTasks[0] != task.ID {
			t.Errorf("expected task ID %s in session, got %s", task.ID, sessionTasks[0])
		}
	})
}

func TestServer_GetTask(t *testing.T) {
	t.Run("retrieves existing task", func(t *testing.T) {
		mockAg := &mockAgent{id: "agent-1"}
		server := NewServer(mockAg)
		handler := server.Handler()

		// Create a task first
		msg := NewUserMessage("Hello")
		createBody, _ := json.Marshal(CreateTaskRequest{Message: &msg})
		createReq := httptest.NewRequest(http.MethodPost, "/tasks", bytes.NewReader(createBody))
		createReq.Header.Set("Content-Type", "application/json")
		createW := httptest.NewRecorder()
		handler.ServeHTTP(createW, createReq)

		var createdTask Task
		json.NewDecoder(createW.Body).Decode(&createdTask)

		// Get the task
		req := httptest.NewRequest(http.MethodGet, "/tasks/"+createdTask.ID, nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", w.Code)
		}

		var task Task
		json.NewDecoder(w.Body).Decode(&task)

		if task.ID != createdTask.ID {
			t.Errorf("expected task ID %s, got %s", createdTask.ID, task.ID)
		}
	})

	t.Run("returns 404 for non-existent task", func(t *testing.T) {
		mockAg := &mockAgent{id: "agent-1"}
		server := NewServer(mockAg)
		handler := server.Handler()

		req := httptest.NewRequest(http.MethodGet, "/tasks/nonexistent", nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("expected status 404, got %d", w.Code)
		}
	})
}

func TestServer_SendMessage(t *testing.T) {
	t.Run("sends message and runs agent", func(t *testing.T) {
		var receivedMessages []agent.Message
		mockAg := &mockAgent{
			id: "agent-1",
			runFn: func(ctx context.Context, messages []agent.Message, opts ...agent.RunOption) (*agent.Response, error) {
				receivedMessages = messages
				return &agent.Response{
					ResponseID: "resp-123",
					Messages:   []agent.Message{agent.NewAssistantMessage("Response text")},
					CreatedAt:  time.Now(),
				}, nil
			},
		}
		server := NewServer(mockAg)
		handler := server.Handler()

		// Create a task first
		msg := NewUserMessage("Hello")
		createBody, _ := json.Marshal(CreateTaskRequest{Message: &msg})
		createReq := httptest.NewRequest(http.MethodPost, "/tasks", bytes.NewReader(createBody))
		createReq.Header.Set("Content-Type", "application/json")
		createW := httptest.NewRecorder()
		handler.ServeHTTP(createW, createReq)

		var createdTask Task
		json.NewDecoder(createW.Body).Decode(&createdTask)

		// Send a message
		followUp := NewUserMessage("Follow up")
		sendBody, _ := json.Marshal(SendMessageRequest{Message: &followUp})
		req := httptest.NewRequest(http.MethodPost, "/tasks/"+createdTask.ID+"/messages", bytes.NewReader(sendBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", w.Code)
		}

		var task Task
		json.NewDecoder(w.Body).Decode(&task)

		if task.State != TaskStateCompleted {
			t.Errorf("expected state completed, got %s", task.State)
		}
		if len(task.Messages) != 3 { // initial + follow up + response
			t.Errorf("expected 3 messages, got %d", len(task.Messages))
		}
		if len(receivedMessages) != 2 { // initial + follow up
			t.Errorf("expected 2 messages sent to agent, got %d", len(receivedMessages))
		}
	})

	t.Run("returns error on agent failure", func(t *testing.T) {
		mockAg := &mockAgent{
			id: "agent-1",
			runFn: func(ctx context.Context, messages []agent.Message, opts ...agent.RunOption) (*agent.Response, error) {
				return nil, errors.New("agent error")
			},
		}
		server := NewServer(mockAg)
		handler := server.Handler()

		// Create a task
		msg := NewUserMessage("Hello")
		createBody, _ := json.Marshal(CreateTaskRequest{Message: &msg})
		createReq := httptest.NewRequest(http.MethodPost, "/tasks", bytes.NewReader(createBody))
		createReq.Header.Set("Content-Type", "application/json")
		createW := httptest.NewRecorder()
		handler.ServeHTTP(createW, createReq)

		var createdTask Task
		json.NewDecoder(createW.Body).Decode(&createdTask)

		// Send message that causes error
		followUp := NewUserMessage("Trigger error")
		sendBody, _ := json.Marshal(SendMessageRequest{Message: &followUp})
		req := httptest.NewRequest(http.MethodPost, "/tasks/"+createdTask.ID+"/messages", bytes.NewReader(sendBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		if w.Code != http.StatusInternalServerError {
			t.Errorf("expected status 500, got %d", w.Code)
		}

		// Verify task is marked as failed
		task, _ := server.GetTask(createdTask.ID)
		if task.State != TaskStateFailed {
			t.Errorf("expected state failed, got %s", task.State)
		}
		if task.Error == nil {
			t.Error("expected error to be set")
		}
	})

	t.Run("returns 404 for non-existent task", func(t *testing.T) {
		mockAg := &mockAgent{id: "agent-1"}
		server := NewServer(mockAg)
		handler := server.Handler()

		msg := NewUserMessage("Hello")
		sendBody, _ := json.Marshal(SendMessageRequest{Message: &msg})
		req := httptest.NewRequest(http.MethodPost, "/tasks/nonexistent/messages", bytes.NewReader(sendBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("expected status 404, got %d", w.Code)
		}
	})

	t.Run("returns error for missing message", func(t *testing.T) {
		mockAg := &mockAgent{id: "agent-1"}
		server := NewServer(mockAg)
		handler := server.Handler()

		// Create a task
		msg := NewUserMessage("Hello")
		createBody, _ := json.Marshal(CreateTaskRequest{Message: &msg})
		createReq := httptest.NewRequest(http.MethodPost, "/tasks", bytes.NewReader(createBody))
		createReq.Header.Set("Content-Type", "application/json")
		createW := httptest.NewRecorder()
		handler.ServeHTTP(createW, createReq)

		var createdTask Task
		json.NewDecoder(createW.Body).Decode(&createdTask)

		// Send empty request
		sendBody, _ := json.Marshal(SendMessageRequest{})
		req := httptest.NewRequest(http.MethodPost, "/tasks/"+createdTask.ID+"/messages", bytes.NewReader(sendBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", w.Code)
		}
	})
}

func TestServer_SendMessageStream(t *testing.T) {
	t.Run("streams response via SSE", func(t *testing.T) {
		mockAg := &mockAgent{
			id: "agent-1",
			runStreamFn: func(ctx context.Context, messages []agent.Message, opts ...agent.RunOption) (<-chan agent.ResponseUpdate, error) {
				updates := make(chan agent.ResponseUpdate, 10)
				go func() {
					defer close(updates)
					updates <- agent.ResponseUpdate{
						Kind: agent.UpdateKindContentDelta,
						Delta: &agent.ContentDelta{
							TextDelta: "Hello ",
						},
					}
					updates <- agent.ResponseUpdate{
						Kind: agent.UpdateKindContentDelta,
						Delta: &agent.ContentDelta{
							TextDelta: "world!",
						},
					}
					msg := agent.NewAssistantMessage("Hello world!")
					updates <- agent.ResponseUpdate{
						Kind:    agent.UpdateKindMessageComplete,
						Message: &msg,
					}
				}()
				return updates, nil
			},
		}
		server := NewServer(mockAg)
		handler := server.Handler()

		// Create a task
		msg := NewUserMessage("Hello")
		createBody, _ := json.Marshal(CreateTaskRequest{Message: &msg})
		createReq := httptest.NewRequest(http.MethodPost, "/tasks", bytes.NewReader(createBody))
		createReq.Header.Set("Content-Type", "application/json")
		createW := httptest.NewRecorder()
		handler.ServeHTTP(createW, createReq)

		var createdTask Task
		json.NewDecoder(createW.Body).Decode(&createdTask)

		// Stream message
		followUp := NewUserMessage("Stream me")
		sendBody, _ := json.Marshal(SendMessageRequest{Message: &followUp})
		req := httptest.NewRequest(http.MethodPost, "/tasks/"+createdTask.ID+"/messages/stream", bytes.NewReader(sendBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", w.Code)
		}

		contentType := w.Header().Get("Content-Type")
		if contentType != "text/event-stream" {
			t.Errorf("expected Content-Type text/event-stream, got %s", contentType)
		}

		body := w.Body.String()
		if !strings.Contains(body, "event: update") {
			t.Error("expected update events in response")
		}
		if !strings.Contains(body, "event: done") {
			t.Error("expected done event in response")
		}

		// Verify task is completed
		task, _ := server.GetTask(createdTask.ID)
		if task.State != TaskStateCompleted {
			t.Errorf("expected state completed, got %s", task.State)
		}
	})

	t.Run("handles stream errors", func(t *testing.T) {
		mockAg := &mockAgent{
			id: "agent-1",
			runStreamFn: func(ctx context.Context, messages []agent.Message, opts ...agent.RunOption) (<-chan agent.ResponseUpdate, error) {
				updates := make(chan agent.ResponseUpdate, 10)
				go func() {
					defer close(updates)
					updates <- agent.ResponseUpdate{
						Kind:  agent.UpdateKindError,
						Error: errors.New("stream error"),
					}
				}()
				return updates, nil
			},
		}
		server := NewServer(mockAg)
		handler := server.Handler()

		// Create a task
		msg := NewUserMessage("Hello")
		createBody, _ := json.Marshal(CreateTaskRequest{Message: &msg})
		createReq := httptest.NewRequest(http.MethodPost, "/tasks", bytes.NewReader(createBody))
		createReq.Header.Set("Content-Type", "application/json")
		createW := httptest.NewRecorder()
		handler.ServeHTTP(createW, createReq)

		var createdTask Task
		json.NewDecoder(createW.Body).Decode(&createdTask)

		// Stream message
		followUp := NewUserMessage("Trigger error")
		sendBody, _ := json.Marshal(SendMessageRequest{Message: &followUp})
		req := httptest.NewRequest(http.MethodPost, "/tasks/"+createdTask.ID+"/messages/stream", bytes.NewReader(sendBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		body := w.Body.String()
		if !strings.Contains(body, "event: error") {
			t.Error("expected error event in response")
		}

		// Verify task is failed
		task, _ := server.GetTask(createdTask.ID)
		if task.State != TaskStateFailed {
			t.Errorf("expected state failed, got %s", task.State)
		}
	})

	t.Run("returns error when RunStream fails", func(t *testing.T) {
		mockAg := &mockAgent{
			id: "agent-1",
			runStreamFn: func(ctx context.Context, messages []agent.Message, opts ...agent.RunOption) (<-chan agent.ResponseUpdate, error) {
				return nil, errors.New("failed to start stream")
			},
		}
		server := NewServer(mockAg)
		handler := server.Handler()

		// Create a task
		msg := NewUserMessage("Hello")
		createBody, _ := json.Marshal(CreateTaskRequest{Message: &msg})
		createReq := httptest.NewRequest(http.MethodPost, "/tasks", bytes.NewReader(createBody))
		createReq.Header.Set("Content-Type", "application/json")
		createW := httptest.NewRecorder()
		handler.ServeHTTP(createW, createReq)

		var createdTask Task
		json.NewDecoder(createW.Body).Decode(&createdTask)

		// Stream message
		followUp := NewUserMessage("Stream me")
		sendBody, _ := json.Marshal(SendMessageRequest{Message: &followUp})
		req := httptest.NewRequest(http.MethodPost, "/tasks/"+createdTask.ID+"/messages/stream", bytes.NewReader(sendBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		body := w.Body.String()
		if !strings.Contains(body, "event: error") {
			t.Error("expected error event in response")
		}

		// Verify task is failed
		task, _ := server.GetTask(createdTask.ID)
		if task.State != TaskStateFailed {
			t.Errorf("expected state failed, got %s", task.State)
		}
	})

	t.Run("returns 404 for non-existent task", func(t *testing.T) {
		mockAg := &mockAgent{id: "agent-1"}
		server := NewServer(mockAg)
		handler := server.Handler()

		msg := NewUserMessage("Hello")
		sendBody, _ := json.Marshal(SendMessageRequest{Message: &msg})
		req := httptest.NewRequest(http.MethodPost, "/tasks/nonexistent/messages/stream", bytes.NewReader(sendBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("expected status 404, got %d", w.Code)
		}
	})
}

func TestServer_CancelTask(t *testing.T) {
	t.Run("cancels running task", func(t *testing.T) {
		mockAg := &mockAgent{id: "agent-1"}
		server := NewServer(mockAg)
		handler := server.Handler()

		// Create a task
		msg := NewUserMessage("Hello")
		createBody, _ := json.Marshal(CreateTaskRequest{Message: &msg})
		createReq := httptest.NewRequest(http.MethodPost, "/tasks", bytes.NewReader(createBody))
		createReq.Header.Set("Content-Type", "application/json")
		createW := httptest.NewRecorder()
		handler.ServeHTTP(createW, createReq)

		var createdTask Task
		json.NewDecoder(createW.Body).Decode(&createdTask)

		// Cancel the task
		req := httptest.NewRequest(http.MethodDelete, "/tasks/"+createdTask.ID, nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		if w.Code != http.StatusNoContent {
			t.Errorf("expected status 204, got %d", w.Code)
		}

		// Verify task is cancelled
		task, _ := server.GetTask(createdTask.ID)
		if task.State != TaskStateCancelled {
			t.Errorf("expected state cancelled, got %s", task.State)
		}
	})

	t.Run("returns 404 for non-existent task", func(t *testing.T) {
		mockAg := &mockAgent{id: "agent-1"}
		server := NewServer(mockAg)
		handler := server.Handler()

		req := httptest.NewRequest(http.MethodDelete, "/tasks/nonexistent", nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("expected status 404, got %d", w.Code)
		}
	})
}

func TestServer_MessageConversion(t *testing.T) {
	t.Run("converts user messages correctly", func(t *testing.T) {
		mockAg := &mockAgent{id: "agent-1"}
		server := NewServer(mockAg)

		a2aMsg := Message{
			Role:  RoleUser,
			Parts: []Part{NewTextPart("Hello user")},
		}
		agentMsg := server.toAgentMessage(a2aMsg)

		if agentMsg.Role != "user" {
			t.Errorf("expected role user, got %s", agentMsg.Role)
		}
		if agentMsg.Text() != "Hello user" {
			t.Errorf("expected text 'Hello user', got '%s'", agentMsg.Text())
		}
	})

	t.Run("converts assistant messages correctly", func(t *testing.T) {
		mockAg := &mockAgent{id: "agent-1"}
		server := NewServer(mockAg)

		a2aMsg := Message{
			Role:  RoleAssistant,
			Parts: []Part{NewTextPart("Hello assistant")},
		}
		agentMsg := server.toAgentMessage(a2aMsg)

		if agentMsg.Role != "assistant" {
			t.Errorf("expected role assistant, got %s", agentMsg.Role)
		}
	})

	t.Run("converts system messages correctly", func(t *testing.T) {
		mockAg := &mockAgent{id: "agent-1"}
		server := NewServer(mockAg)

		a2aMsg := Message{
			Role:  RoleSystem,
			Parts: []Part{NewTextPart("System prompt")},
		}
		agentMsg := server.toAgentMessage(a2aMsg)

		if agentMsg.Role != "system" {
			t.Errorf("expected role system, got %s", agentMsg.Role)
		}
	})

	t.Run("converts back to A2A message", func(t *testing.T) {
		mockAg := &mockAgent{id: "agent-1"}
		server := NewServer(mockAg)

		agentMsg := agent.NewAssistantMessage("Response text")
		a2aMsg := server.toA2AMessage(agentMsg)

		if a2aMsg.Role != RoleAssistant {
			t.Errorf("expected role assistant, got %s", a2aMsg.Role)
		}
		if a2aMsg.Text() != "Response text" {
			t.Errorf("expected text 'Response text', got '%s'", a2aMsg.Text())
		}
	})
}

func TestServer_GetTask_External(t *testing.T) {
	t.Run("returns task and true for existing task", func(t *testing.T) {
		mockAg := &mockAgent{id: "agent-1"}
		server := NewServer(mockAg)
		handler := server.Handler()

		// Create a task
		msg := NewUserMessage("Hello")
		createBody, _ := json.Marshal(CreateTaskRequest{Message: &msg})
		createReq := httptest.NewRequest(http.MethodPost, "/tasks", bytes.NewReader(createBody))
		createReq.Header.Set("Content-Type", "application/json")
		createW := httptest.NewRecorder()
		handler.ServeHTTP(createW, createReq)

		var createdTask Task
		json.NewDecoder(createW.Body).Decode(&createdTask)

		// Get task via external method
		task, ok := server.GetTask(createdTask.ID)
		if !ok {
			t.Error("expected task to be found")
		}
		if task.ID != createdTask.ID {
			t.Errorf("expected task ID %s, got %s", createdTask.ID, task.ID)
		}
	})

	t.Run("returns nil and false for non-existent task", func(t *testing.T) {
		mockAg := &mockAgent{id: "agent-1"}
		server := NewServer(mockAg)

		task, ok := server.GetTask("nonexistent")
		if ok {
			t.Error("expected task not to be found")
		}
		if task != nil {
			t.Error("expected task to be nil")
		}
	})
}

func TestServer_GetSessionTasks(t *testing.T) {
	t.Run("returns empty for non-existent session", func(t *testing.T) {
		mockAg := &mockAgent{id: "agent-1"}
		server := NewServer(mockAg)

		tasks := server.GetSessionTasks("nonexistent")
		if tasks != nil {
			t.Error("expected nil for non-existent session")
		}
	})

	t.Run("returns task IDs for existing session", func(t *testing.T) {
		mockAg := &mockAgent{id: "agent-1"}
		server := NewServer(mockAg)
		handler := server.Handler()

		// Create multiple tasks in same session
		for i := 0; i < 3; i++ {
			msg := NewUserMessage("Hello")
			createBody, _ := json.Marshal(CreateTaskRequest{
				Message:   &msg,
				ContextID: "session-xyz",
			})
			createReq := httptest.NewRequest(http.MethodPost, "/tasks", bytes.NewReader(createBody))
			createReq.Header.Set("Content-Type", "application/json")
			createW := httptest.NewRecorder()
			handler.ServeHTTP(createW, createReq)
		}

		tasks := server.GetSessionTasks("session-xyz")
		if len(tasks) != 3 {
			t.Errorf("expected 3 tasks, got %d", len(tasks))
		}
	})
}

func TestServer_Integration(t *testing.T) {
	t.Run("full conversation flow", func(t *testing.T) {
		responseCount := 0
		mockAg := &mockAgent{
			id:   "agent-1",
			name: "TestBot",
			runFn: func(ctx context.Context, messages []agent.Message, opts ...agent.RunOption) (*agent.Response, error) {
				responseCount++
				return &agent.Response{
					ResponseID: "resp-" + string(rune('0'+responseCount)),
					Messages:   []agent.Message{agent.NewAssistantMessage("Response " + string(rune('0'+responseCount)))},
					CreatedAt:  time.Now(),
				}, nil
			},
		}
		server := NewServer(mockAg)
		handler := server.Handler()

		// 1. Get agent card
		cardReq := httptest.NewRequest(http.MethodGet, "/.well-known/agent.json", nil)
		cardW := httptest.NewRecorder()
		handler.ServeHTTP(cardW, cardReq)

		if cardW.Code != http.StatusOK {
			t.Fatalf("failed to get agent card: %d", cardW.Code)
		}

		// 2. Create task
		msg1 := NewUserMessage("Hello")
		createBody, _ := json.Marshal(CreateTaskRequest{Message: &msg1})
		createReq := httptest.NewRequest(http.MethodPost, "/tasks", bytes.NewReader(createBody))
		createReq.Header.Set("Content-Type", "application/json")
		createW := httptest.NewRecorder()
		handler.ServeHTTP(createW, createReq)

		if createW.Code != http.StatusCreated {
			t.Fatalf("failed to create task: %d", createW.Code)
		}

		var task Task
		json.NewDecoder(createW.Body).Decode(&task)

		// 3. Send messages
		for i := 0; i < 3; i++ {
			msg := NewUserMessage("Message " + string(rune('0'+i)))
			sendBody, _ := json.Marshal(SendMessageRequest{Message: &msg})
			sendReq := httptest.NewRequest(http.MethodPost, "/tasks/"+task.ID+"/messages", bytes.NewReader(sendBody))
			sendReq.Header.Set("Content-Type", "application/json")
			sendW := httptest.NewRecorder()
			handler.ServeHTTP(sendW, sendReq)

			if sendW.Code != http.StatusOK {
				t.Fatalf("failed to send message %d: %d", i, sendW.Code)
			}
		}

		// 4. Get final task state
		getReq := httptest.NewRequest(http.MethodGet, "/tasks/"+task.ID, nil)
		getW := httptest.NewRecorder()
		handler.ServeHTTP(getW, getReq)

		var finalTask Task
		json.NewDecoder(getW.Body).Decode(&finalTask)

		// Initial message + 3 user messages + 3 responses = 7 messages
		if len(finalTask.Messages) != 7 {
			t.Errorf("expected 7 messages, got %d", len(finalTask.Messages))
		}
		if finalTask.State != TaskStateCompleted {
			t.Errorf("expected state completed, got %s", finalTask.State)
		}
	})
}

// Ensure mock implements interface at compile time
var _ agent.Agent = (*mockAgent)(nil)

// Suppress unused import warning
var _ = io.EOF
