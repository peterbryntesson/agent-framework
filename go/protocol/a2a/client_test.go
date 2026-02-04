// Copyright (c) Microsoft. All rights reserved.

package a2a

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestNewClient(t *testing.T) {
	t.Run("creates client with default settings", func(t *testing.T) {
		client := NewClient("http://localhost:8080")

		if client == nil {
			t.Fatal("expected client to be created")
		}
		if client.baseURL != "http://localhost:8080" {
			t.Errorf("expected baseURL http://localhost:8080, got %s", client.baseURL)
		}
		if client.httpClient == nil {
			t.Error("expected httpClient to be set")
		}
	})

	t.Run("creates client with custom HTTP client", func(t *testing.T) {
		customClient := &http.Client{Timeout: 60 * time.Second}
		client := NewClient("http://localhost:8080", WithHTTPClient(customClient))

		if client.httpClient != customClient {
			t.Error("expected custom HTTP client to be used")
		}
	})

	t.Run("creates client with custom headers", func(t *testing.T) {
		client := NewClient("http://localhost:8080",
			WithHeader("Authorization", "Bearer token"),
			WithHeader("X-Custom", "value"),
		)

		if client.headers["Authorization"] != "Bearer token" {
			t.Error("expected Authorization header to be set")
		}
		if client.headers["X-Custom"] != "value" {
			t.Error("expected X-Custom header to be set")
		}
	})

	t.Run("creates client with custom timeout", func(t *testing.T) {
		client := NewClient("http://localhost:8080", WithTimeout(60*time.Second))

		if client.httpClient.Timeout != 60*time.Second {
			t.Errorf("expected timeout 60s, got %v", client.httpClient.Timeout)
		}
	})
}

func TestClient_GetAgentCard(t *testing.T) {
	t.Run("retrieves agent card successfully", func(t *testing.T) {
		expectedCard := AgentCard{
			Name:        "TestAgent",
			Description: "A test agent",
			URL:         "http://localhost:8080",
			Capabilities: &AgentCapabilities{
				Streaming: true,
			},
		}

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/.well-known/agent.json" {
				t.Errorf("expected path /.well-known/agent.json, got %s", r.URL.Path)
			}
			if r.Method != http.MethodGet {
				t.Errorf("expected GET method, got %s", r.Method)
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(expectedCard)
		}))
		defer server.Close()

		client := NewClient(server.URL)
		card, err := client.GetAgentCard(context.Background())

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if card.Name != expectedCard.Name {
			t.Errorf("expected name %s, got %s", expectedCard.Name, card.Name)
		}
		if card.Description != expectedCard.Description {
			t.Errorf("expected description %s, got %s", expectedCard.Description, card.Description)
		}
	})

	t.Run("returns error on non-200 status", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		}))
		defer server.Close()

		client := NewClient(server.URL)
		_, err := client.GetAgentCard(context.Background())

		if err == nil {
			t.Fatal("expected error")
		}
		if !strings.Contains(err.Error(), "unexpected status") {
			t.Errorf("expected 'unexpected status' error, got: %v", err)
		}
	})

	t.Run("includes custom headers", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Header.Get("Authorization") != "Bearer test-token" {
				t.Error("expected Authorization header")
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(AgentCard{Name: "Test"})
		}))
		defer server.Close()

		client := NewClient(server.URL, WithHeader("Authorization", "Bearer test-token"))
		_, err := client.GetAgentCard(context.Background())

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

func TestClient_CreateTask(t *testing.T) {
	t.Run("creates task successfully", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/tasks" {
				t.Errorf("expected path /tasks, got %s", r.URL.Path)
			}
			if r.Method != http.MethodPost {
				t.Errorf("expected POST method, got %s", r.Method)
			}

			var req CreateTaskRequest
			json.NewDecoder(r.Body).Decode(&req)

			if req.Message == nil || req.Message.Text() != "Hello" {
				t.Error("expected message with text 'Hello'")
			}

			task := Task{
				ID:        "task-123",
				ContextID: "ctx-456",
				State:     TaskStatePending,
				Messages:  []Message{*req.Message},
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(task)
		}))
		defer server.Close()

		client := NewClient(server.URL)
		msg := NewUserMessage("Hello")
		task, err := client.CreateTask(context.Background(), &CreateTaskRequest{
			Message: &msg,
		})

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if task.ID != "task-123" {
			t.Errorf("expected task ID task-123, got %s", task.ID)
		}
		if task.State != TaskStatePending {
			t.Errorf("expected state pending, got %s", task.State)
		}
	})

	t.Run("returns error on failure", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusBadRequest)
		}))
		defer server.Close()

		client := NewClient(server.URL)
		msg := NewUserMessage("Hello")
		_, err := client.CreateTask(context.Background(), &CreateTaskRequest{
			Message: &msg,
		})

		if err == nil {
			t.Fatal("expected error")
		}
	})
}

func TestClient_GetTask(t *testing.T) {
	t.Run("retrieves task successfully", func(t *testing.T) {
		expectedTask := Task{
			ID:        "task-123",
			ContextID: "ctx-456",
			State:     TaskStateCompleted,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/tasks/task-123" {
				t.Errorf("expected path /tasks/task-123, got %s", r.URL.Path)
			}
			if r.Method != http.MethodGet {
				t.Errorf("expected GET method, got %s", r.Method)
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(expectedTask)
		}))
		defer server.Close()

		client := NewClient(server.URL)
		task, err := client.GetTask(context.Background(), "task-123")

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if task.ID != expectedTask.ID {
			t.Errorf("expected task ID %s, got %s", expectedTask.ID, task.ID)
		}
		if task.State != expectedTask.State {
			t.Errorf("expected state %s, got %s", expectedTask.State, task.State)
		}
	})

	t.Run("returns error when task not found", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		}))
		defer server.Close()

		client := NewClient(server.URL)
		_, err := client.GetTask(context.Background(), "nonexistent")

		if err == nil {
			t.Fatal("expected error")
		}
		if !strings.Contains(err.Error(), "unexpected status") {
			t.Errorf("expected 'unexpected status' error, got: %v", err)
		}
	})
}

func TestClient_SendMessage(t *testing.T) {
	t.Run("sends message successfully", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/tasks/task-123/messages" {
				t.Errorf("expected path /tasks/task-123/messages, got %s", r.URL.Path)
			}
			if r.Method != http.MethodPost {
				t.Errorf("expected POST method, got %s", r.Method)
			}

			var req SendMessageRequest
			json.NewDecoder(r.Body).Decode(&req)

			if req.Message == nil || req.Message.Text() != "How are you?" {
				t.Error("expected message with text 'How are you?'")
			}

			task := Task{
				ID:    "task-123",
				State: TaskStateCompleted,
				Messages: []Message{
					*req.Message,
					NewAssistantMessage("I'm doing well!"),
				},
				UpdatedAt: time.Now(),
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(task)
		}))
		defer server.Close()

		client := NewClient(server.URL)
		msg := NewUserMessage("How are you?")
		task, err := client.SendMessage(context.Background(), "task-123", &msg)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(task.Messages) != 2 {
			t.Errorf("expected 2 messages, got %d", len(task.Messages))
		}
		if task.State != TaskStateCompleted {
			t.Errorf("expected state completed, got %s", task.State)
		}
	})

	t.Run("returns error on failure", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer server.Close()

		client := NewClient(server.URL)
		msg := NewUserMessage("Hello")
		_, err := client.SendMessage(context.Background(), "task-123", &msg)

		if err == nil {
			t.Fatal("expected error")
		}
	})
}

func TestClient_SendMessageStream(t *testing.T) {
	t.Run("streams events successfully", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/tasks/task-123/messages/stream" {
				t.Errorf("expected path /tasks/task-123/messages/stream, got %s", r.URL.Path)
			}
			if r.Header.Get("Accept") != "text/event-stream" {
				t.Error("expected Accept: text/event-stream header")
			}

			w.Header().Set("Content-Type", "text/event-stream")
			w.Header().Set("Cache-Control", "no-cache")
			w.Header().Set("Connection", "keep-alive")

			flusher, ok := w.(http.Flusher)
			if !ok {
				t.Fatal("expected flusher support")
			}

			// Send message event
			msg := NewAssistantMessage("Hello!")
			msgData, _ := json.Marshal(msg)
			io.WriteString(w, "event:message\n")
			io.WriteString(w, "data:"+string(msgData)+"\n\n")
			flusher.Flush()

			// Send done event
			io.WriteString(w, "event:done\n")
			io.WriteString(w, "data:{}\n\n")
			flusher.Flush()
		}))
		defer server.Close()

		client := NewClient(server.URL)
		msg := NewUserMessage("Hi")
		events, err := client.SendMessageStream(context.Background(), "task-123", &msg)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		var receivedEvents []StreamEvent
		for event := range events {
			receivedEvents = append(receivedEvents, event)
		}

		if len(receivedEvents) != 2 {
			t.Fatalf("expected 2 events, got %d", len(receivedEvents))
		}

		if receivedEvents[0].Type != StreamEventTypeMessage {
			t.Errorf("expected message event, got %s", receivedEvents[0].Type)
		}
		if receivedEvents[0].Message == nil {
			t.Error("expected message to be parsed")
		}
		if receivedEvents[1].Type != StreamEventTypeDone {
			t.Errorf("expected done event, got %s", receivedEvents[1].Type)
		}
	})

	t.Run("handles context cancellation", func(t *testing.T) {
		serverStarted := make(chan struct{})
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/event-stream")
			w.WriteHeader(http.StatusOK)
			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			}
			close(serverStarted)

			// Wait for context cancellation or timeout
			select {
			case <-r.Context().Done():
			case <-time.After(2 * time.Second):
			}
		}))
		defer server.Close()

		ctx, cancel := context.WithCancel(context.Background())
		client := NewClient(server.URL, WithTimeout(5*time.Second))
		msg := NewUserMessage("Hi")
		events, err := client.SendMessageStream(ctx, "task-123", &msg)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Wait for server to start streaming
		<-serverStarted

		// Cancel the context
		cancel()

		// Drain the events channel
		var receivedEvents []StreamEvent
		for event := range events {
			receivedEvents = append(receivedEvents, event)
		}

		// Should receive an error event for context cancellation
		if len(receivedEvents) > 0 {
			lastEvent := receivedEvents[len(receivedEvents)-1]
			if lastEvent.Type == StreamEventTypeError && lastEvent.Error != nil {
				t.Logf("received expected error event: %v", lastEvent.Error)
			}
		}
	})

	t.Run("returns error on non-200 status", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		}))
		defer server.Close()

		client := NewClient(server.URL)
		msg := NewUserMessage("Hi")
		_, err := client.SendMessageStream(context.Background(), "task-123", &msg)

		if err == nil {
			t.Fatal("expected error")
		}
		if !strings.Contains(err.Error(), "unexpected status") {
			t.Errorf("expected 'unexpected status' error, got: %v", err)
		}
	})
}

func TestClient_CancelTask(t *testing.T) {
	t.Run("cancels task successfully", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/tasks/task-123" {
				t.Errorf("expected path /tasks/task-123, got %s", r.URL.Path)
			}
			if r.Method != http.MethodDelete {
				t.Errorf("expected DELETE method, got %s", r.Method)
			}

			w.WriteHeader(http.StatusNoContent)
		}))
		defer server.Close()

		client := NewClient(server.URL)
		err := client.CancelTask(context.Background(), "task-123")

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("returns error when task not found", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		}))
		defer server.Close()

		client := NewClient(server.URL)
		err := client.CancelTask(context.Background(), "nonexistent")

		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("accepts 200 OK status", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		client := NewClient(server.URL)
		err := client.CancelTask(context.Background(), "task-123")

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

func TestClient_parseSSE(t *testing.T) {
	t.Run("parses multi-line data events", func(t *testing.T) {
		sseData := `event:message
data:{"role":"assistant","parts":[{"type":"text","text":"Hello"}]}

event:done
data:{}

`
		client := NewClient("http://localhost")
		events := make(chan StreamEvent, 10)

		done := make(chan struct{})
		go func() {
			defer close(done)
			client.parseSSE(context.Background(), strings.NewReader(sseData), events)
			close(events)
		}()

		var receivedEvents []StreamEvent
		for event := range events {
			receivedEvents = append(receivedEvents, event)
		}
		<-done

		if len(receivedEvents) != 2 {
			t.Fatalf("expected 2 events, got %d", len(receivedEvents))
		}

		if receivedEvents[0].Type != StreamEventTypeMessage {
			t.Errorf("expected message event, got %s", receivedEvents[0].Type)
		}
		if receivedEvents[1].Type != StreamEventTypeDone {
			t.Errorf("expected done event, got %s", receivedEvents[1].Type)
		}
	})

	t.Run("parses task events", func(t *testing.T) {
		taskData := `{"id":"task-123","state":"completed"}`
		sseData := "event:task\ndata:" + taskData + "\n\n"

		client := NewClient("http://localhost")
		events := make(chan StreamEvent, 10)

		done := make(chan struct{})
		go func() {
			defer close(done)
			client.parseSSE(context.Background(), strings.NewReader(sseData), events)
			close(events)
		}()

		var receivedEvents []StreamEvent
		for event := range events {
			receivedEvents = append(receivedEvents, event)
		}
		<-done

		if len(receivedEvents) != 1 {
			t.Fatalf("expected 1 event, got %d", len(receivedEvents))
		}

		if receivedEvents[0].Type != StreamEventTypeTask {
			t.Errorf("expected task event, got %s", receivedEvents[0].Type)
		}
		if receivedEvents[0].Task == nil {
			t.Error("expected task to be parsed")
		} else if receivedEvents[0].Task.ID != "task-123" {
			t.Errorf("expected task ID task-123, got %s", receivedEvents[0].Task.ID)
		}
	})
}
