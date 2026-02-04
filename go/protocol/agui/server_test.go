// Copyright (c) Microsoft. All rights reserved.

package agui

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
	"sync"
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
		Messages: []agent.Message{agent.NewAssistantMessage("Hello!")},
	}, nil
}

func (m *mockAgent) RunStream(ctx context.Context, messages []agent.Message, opts ...agent.RunOption) (<-chan agent.ResponseUpdate, error) {
	if m.runStreamFn != nil {
		return m.runStreamFn(ctx, messages, opts...)
	}
	updates := make(chan agent.ResponseUpdate, 10)
	go func() {
		defer close(updates)
		msg := agent.NewAssistantMessage("Hello!")
		updates <- agent.ResponseUpdate{
			Kind:    agent.UpdateKindMessageComplete,
			Message: &msg,
		}
		updates <- agent.ResponseUpdate{
			Kind: agent.UpdateKindDone,
		}
	}()
	return updates, nil
}

func TestNewServer(t *testing.T) {
	mockAgt := &mockAgent{
		id:          "test-agent",
		name:        "Test Agent",
		description: "A test agent",
	}

	server := NewServer(mockAgt)

	if server == nil {
		t.Fatal("expected server to be created")
	}
	if server.agent == nil {
		t.Error("expected agent to be set")
	}
	if server.activeConnections == nil {
		t.Error("expected activeConnections map to be initialized")
	}
}

func TestServer_Handler(t *testing.T) {
	mockAgt := &mockAgent{id: "test-agent"}
	server := NewServer(mockAgt)

	handler := server.Handler()
	if handler == nil {
		t.Fatal("expected handler to be returned")
	}
}

func TestServer_HandleRun_Success(t *testing.T) {
	mockAgt := &mockAgent{
		id: "test-agent",
		runFn: func(_ context.Context, messages []agent.Message, _ ...agent.RunOption) (*agent.Response, error) {
			return &agent.Response{
				ResponseID: "resp-123",
				Messages:   []agent.Message{agent.NewAssistantMessage("Hello from agent!")},
			}, nil
		},
	}

	server := NewServer(mockAgt)

	reqBody := RunRequest{
		ThreadID: "thread-123",
		Messages: []RequestMessage{
			{Role: "user", Content: "Hello!"},
		},
	}
	bodyBytes, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/run", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	var events []json.RawMessage
	if err := json.NewDecoder(rec.Body).Decode(&events); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(events) < 3 {
		t.Fatalf("expected at least 3 events, got %d", len(events))
	}

	// Verify first event is RUN_STARTED
	var firstEvent struct {
		Type string `json:"type"`
	}
	json.Unmarshal(events[0], &firstEvent)
	if firstEvent.Type != string(EventTypeRunStarted) {
		t.Errorf("expected first event to be RUN_STARTED, got %s", firstEvent.Type)
	}

	// Verify last event is RUN_FINISHED
	var lastEvent struct {
		Type string `json:"type"`
	}
	json.Unmarshal(events[len(events)-1], &lastEvent)
	if lastEvent.Type != string(EventTypeRunFinished) {
		t.Errorf("expected last event to be RUN_FINISHED, got %s", lastEvent.Type)
	}
}

func TestServer_HandleRun_InvalidJSON(t *testing.T) {
	mockAgt := &mockAgent{id: "test-agent"}
	server := NewServer(mockAgt)

	req := httptest.NewRequest(http.MethodPost, "/run", strings.NewReader("invalid json"))
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rec.Code)
	}
}

func TestServer_HandleRun_AgentError(t *testing.T) {
	mockAgt := &mockAgent{
		id: "test-agent",
		runFn: func(_ context.Context, _ []agent.Message, _ ...agent.RunOption) (*agent.Response, error) {
			return nil, context.DeadlineExceeded
		},
	}

	server := NewServer(mockAgt)

	reqBody := RunRequest{
		Messages: []RequestMessage{{Role: "user", Content: "Hello!"}},
	}
	bodyBytes, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/run", bytes.NewReader(bodyBytes))
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", rec.Code)
	}
}

func TestServer_HandleRun_GeneratesThreadID(t *testing.T) {
	mockAgt := &mockAgent{
		id: "test-agent",
		runFn: func(_ context.Context, _ []agent.Message, _ ...agent.RunOption) (*agent.Response, error) {
			return &agent.Response{
				Messages: []agent.Message{agent.NewAssistantMessage("Hello!")},
			}, nil
		},
	}

	server := NewServer(mockAgt)

	// Request without threadID
	reqBody := RunRequest{
		Messages: []RequestMessage{{Role: "user", Content: "Hello!"}},
	}
	bodyBytes, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/run", bytes.NewReader(bodyBytes))
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	// Verify events have a threadId
	var events []struct {
		Type     string `json:"type"`
		ThreadID string `json:"threadId"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&events); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(events) == 0 {
		t.Fatal("expected at least one event")
	}

	if events[0].ThreadID == "" {
		t.Error("expected threadId to be generated")
	}
}

func TestServer_HandleRunStream_Success(t *testing.T) {
	mockAgt := &mockAgent{
		id: "test-agent",
		runStreamFn: func(_ context.Context, _ []agent.Message, _ ...agent.RunOption) (<-chan agent.ResponseUpdate, error) {
			updates := make(chan agent.ResponseUpdate, 10)
			go func() {
				defer close(updates)
				updates <- agent.ResponseUpdate{
					Kind: agent.UpdateKindContentDelta,
					Delta: &agent.ContentDelta{
						TextDelta: "Hello",
						Role:      "assistant",
					},
				}
				updates <- agent.ResponseUpdate{
					Kind: agent.UpdateKindContentDelta,
					Delta: &agent.ContentDelta{
						TextDelta: " World!",
					},
				}
				msg := agent.NewAssistantMessage("Hello World!")
				updates <- agent.ResponseUpdate{
					Kind:    agent.UpdateKindMessageComplete,
					Message: &msg,
				}
			}()
			return updates, nil
		},
	}

	server := NewServer(mockAgt)

	reqBody := RunRequest{
		ThreadID: "thread-123",
		Messages: []RequestMessage{{Role: "user", Content: "Hi!"}},
	}
	bodyBytes, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/stream", bytes.NewReader(bodyBytes))
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	// Verify SSE headers
	if rec.Header().Get("Content-Type") != "text/event-stream" {
		t.Errorf("expected Content-Type text/event-stream, got %s", rec.Header().Get("Content-Type"))
	}
	if rec.Header().Get("Cache-Control") != "no-cache" {
		t.Errorf("expected Cache-Control no-cache, got %s", rec.Header().Get("Cache-Control"))
	}

	// Parse SSE events
	body := rec.Body.String()
	events := parseSSEEvents(body)

	if len(events) < 2 {
		t.Fatalf("expected at least 2 events, got %d: %s", len(events), body)
	}

	// Verify RUN_STARTED is first
	if events[0].eventType != string(EventTypeRunStarted) {
		t.Errorf("expected first event type RUN_STARTED, got %s", events[0].eventType)
	}

	// Verify RUN_FINISHED is last
	lastEvent := events[len(events)-1]
	if lastEvent.eventType != string(EventTypeRunFinished) {
		t.Errorf("expected last event type RUN_FINISHED, got %s", lastEvent.eventType)
	}
}

func TestServer_HandleRunStream_InvalidJSON(t *testing.T) {
	mockAgt := &mockAgent{id: "test-agent"}
	server := NewServer(mockAgt)

	req := httptest.NewRequest(http.MethodPost, "/stream", strings.NewReader("not json"))
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rec.Code)
	}
}

func TestServer_HandleRunStream_AgentError(t *testing.T) {
	mockAgt := &mockAgent{
		id: "test-agent",
		runStreamFn: func(_ context.Context, _ []agent.Message, _ ...agent.RunOption) (<-chan agent.ResponseUpdate, error) {
			return nil, context.DeadlineExceeded
		},
	}

	server := NewServer(mockAgt)

	reqBody := RunRequest{
		Messages: []RequestMessage{{Role: "user", Content: "Hi!"}},
	}
	bodyBytes, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/stream", bytes.NewReader(bodyBytes))
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	// Verify RUN_ERROR event was sent
	body := rec.Body.String()
	if !strings.Contains(body, string(EventTypeRunError)) {
		t.Errorf("expected RUN_ERROR event in response: %s", body)
	}
}

func TestServer_HandleRunStream_ErrorUpdate(t *testing.T) {
	mockAgt := &mockAgent{
		id: "test-agent",
		runStreamFn: func(_ context.Context, _ []agent.Message, _ ...agent.RunOption) (<-chan agent.ResponseUpdate, error) {
			updates := make(chan agent.ResponseUpdate, 10)
			go func() {
				defer close(updates)
				updates <- agent.ResponseUpdate{
					Kind:  agent.UpdateKindError,
					Error: context.DeadlineExceeded,
				}
			}()
			return updates, nil
		},
	}

	server := NewServer(mockAgt)

	reqBody := RunRequest{
		Messages: []RequestMessage{{Role: "user", Content: "Hi!"}},
	}
	bodyBytes, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/stream", bytes.NewReader(bodyBytes))
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	// Verify RUN_ERROR event was sent
	body := rec.Body.String()
	if !strings.Contains(body, string(EventTypeRunError)) {
		t.Errorf("expected RUN_ERROR event in response: %s", body)
	}
}

func TestServer_HandleRunStream_ToolCalls(t *testing.T) {
	mockAgt := &mockAgent{
		id: "test-agent",
		runStreamFn: func(_ context.Context, _ []agent.Message, _ ...agent.RunOption) (<-chan agent.ResponseUpdate, error) {
			updates := make(chan agent.ResponseUpdate, 10)
			go func() {
				defer close(updates)
				updates <- agent.ResponseUpdate{
					Kind: agent.UpdateKindToolCall,
					Delta: &agent.ContentDelta{
						ToolCallID: "call-123",
						Name:       "get_weather",
						ArgsDelta:  `{"city": "Seattle"}`,
					},
				}
				msg := agent.NewToolMessage("call-123", "72°F and sunny")
				updates <- agent.ResponseUpdate{
					Kind:    agent.UpdateKindToolResult,
					Message: &msg,
				}
			}()
			return updates, nil
		},
	}

	server := NewServer(mockAgt)

	reqBody := RunRequest{
		Messages: []RequestMessage{{Role: "user", Content: "What's the weather?"}},
	}
	bodyBytes, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/stream", bytes.NewReader(bodyBytes))
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	body := rec.Body.String()

	// Verify tool call events are present
	if !strings.Contains(body, string(EventTypeToolCallStart)) {
		t.Errorf("expected TOOL_CALL_START event in response: %s", body)
	}
	if !strings.Contains(body, string(EventTypeToolCallArgs)) {
		t.Errorf("expected TOOL_CALL_ARGS event in response: %s", body)
	}
}

func TestServer_ConnectionLifecycle(t *testing.T) {
	mockAgt := &mockAgent{
		id: "test-agent",
		runStreamFn: func(ctx context.Context, _ []agent.Message, _ ...agent.RunOption) (<-chan agent.ResponseUpdate, error) {
			updates := make(chan agent.ResponseUpdate)
			go func() {
				defer close(updates)
				// Wait for context cancellation or timeout
				select {
				case <-ctx.Done():
					return
				case <-time.After(5 * time.Second):
					return
				}
			}()
			return updates, nil
		},
	}

	server := NewServer(mockAgt)

	// Track connection count
	if server.ActiveConnectionCount() != 0 {
		t.Errorf("expected 0 active connections initially, got %d", server.ActiveConnectionCount())
	}

	// Start a streaming request in a goroutine
	var wg sync.WaitGroup
	wg.Add(1)

	reqBody := RunRequest{
		Messages: []RequestMessage{{Role: "user", Content: "Hi!"}},
	}
	bodyBytes, _ := json.Marshal(reqBody)

	go func() {
		defer wg.Done()
		req := httptest.NewRequest(http.MethodPost, "/stream", bytes.NewReader(bodyBytes))
		rec := httptest.NewRecorder()
		server.Handler().ServeHTTP(rec, req)
	}()

	// Wait a bit for the connection to be registered
	time.Sleep(50 * time.Millisecond)

	if server.ActiveConnectionCount() != 1 {
		t.Logf("expected 1 active connection, got %d", server.ActiveConnectionCount())
	}

	// Wait for cleanup
	wg.Wait()
}

func TestServer_toAgentMessages(t *testing.T) {
	mockAgt := &mockAgent{id: "test-agent"}
	server := NewServer(mockAgt)

	messages := []RequestMessage{
		{Role: "user", Content: "Hello"},
		{Role: "assistant", Content: "Hi there"},
		{Role: "system", Content: "You are helpful"},
		{Role: "unknown", Content: "Test"},
	}

	result := server.toAgentMessages(messages)

	if len(result) != 4 {
		t.Fatalf("expected 4 messages, got %d", len(result))
	}

	if result[0].Role != "user" {
		t.Errorf("expected first message role user, got %s", result[0].Role)
	}
	if result[1].Role != "assistant" {
		t.Errorf("expected second message role assistant, got %s", result[1].Role)
	}
	if result[2].Role != "system" {
		t.Errorf("expected third message role system, got %s", result[2].Role)
	}
	// Unknown role defaults to user
	if result[3].Role != "user" {
		t.Errorf("expected fourth message role user (default), got %s", result[3].Role)
	}
}

func TestServer_CancelConnection_NotFound(t *testing.T) {
	mockAgt := &mockAgent{id: "test-agent"}
	server := NewServer(mockAgt)

	cancelled := server.CancelConnection("nonexistent-id")
	if cancelled {
		t.Error("expected CancelConnection to return false for nonexistent connection")
	}
}

func TestServer_HandleRun_RootEndpoint(t *testing.T) {
	mockAgt := &mockAgent{
		id: "test-agent",
		runFn: func(_ context.Context, _ []agent.Message, _ ...agent.RunOption) (*agent.Response, error) {
			return &agent.Response{
				Messages: []agent.Message{agent.NewAssistantMessage("Root response")},
			}, nil
		},
	}

	server := NewServer(mockAgt)

	reqBody := RunRequest{
		Messages: []RequestMessage{{Role: "user", Content: "Hello!"}},
	}
	bodyBytes, _ := json.Marshal(reqBody)

	// Test root endpoint "/"
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(bodyBytes))
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200 for root endpoint, got %d", rec.Code)
	}
}

func TestServer_HandleRunStream_EmptyMessages(t *testing.T) {
	mockAgt := &mockAgent{
		id: "test-agent",
		runStreamFn: func(_ context.Context, messages []agent.Message, _ ...agent.RunOption) (<-chan agent.ResponseUpdate, error) {
			updates := make(chan agent.ResponseUpdate, 10)
			go func() {
				defer close(updates)
				msg := agent.NewAssistantMessage("No input received")
				updates <- agent.ResponseUpdate{
					Kind:    agent.UpdateKindMessageComplete,
					Message: &msg,
				}
			}()
			return updates, nil
		},
	}

	server := NewServer(mockAgt)

	reqBody := RunRequest{
		Messages: []RequestMessage{},
	}
	bodyBytes, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/stream", bytes.NewReader(bodyBytes))
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}
}

// sseEvent represents a parsed SSE event.
type sseEvent struct {
	eventType string
	data      string
}

// parseSSEEvents parses SSE events from a response body.
func parseSSEEvents(body string) []sseEvent {
	var events []sseEvent
	lines := strings.Split(body, "\n")

	var currentEvent sseEvent
	for _, line := range lines {
		if strings.HasPrefix(line, "event: ") {
			currentEvent.eventType = strings.TrimPrefix(line, "event: ")
		} else if strings.HasPrefix(line, "data: ") {
			currentEvent.data = strings.TrimPrefix(line, "data: ")
		} else if line == "" && currentEvent.eventType != "" {
			events = append(events, currentEvent)
			currentEvent = sseEvent{}
		}
	}

	return events
}

func TestServer_HandleRunStream_WithContextCancellation(t *testing.T) {
	startedCh := make(chan struct{})
	mockAgt := &mockAgent{
		id: "test-agent",
		runStreamFn: func(ctx context.Context, _ []agent.Message, _ ...agent.RunOption) (<-chan agent.ResponseUpdate, error) {
			updates := make(chan agent.ResponseUpdate)
			go func() {
				defer close(updates)
				close(startedCh)
				// Block until context is cancelled
				<-ctx.Done()
			}()
			return updates, nil
		},
	}

	server := NewServer(mockAgt)

	reqBody := RunRequest{
		Messages: []RequestMessage{{Role: "user", Content: "Hi!"}},
	}
	bodyBytes, _ := json.Marshal(reqBody)

	ctx, cancel := context.WithCancel(context.Background())
	req := httptest.NewRequest(http.MethodPost, "/stream", bytes.NewReader(bodyBytes)).WithContext(ctx)
	rec := httptest.NewRecorder()

	done := make(chan struct{})
	go func() {
		server.Handler().ServeHTTP(rec, req)
		close(done)
	}()

	// Wait for stream to start
	<-startedCh

	// Cancel the context
	cancel()

	// Wait for handler to complete
	select {
	case <-done:
		// Success
	case <-time.After(2 * time.Second):
		t.Fatal("handler did not complete after context cancellation")
	}
}

// flushingRecorder implements http.ResponseWriter and http.Flusher for testing.
type flushingRecorder struct {
	*httptest.ResponseRecorder
	flushed bool
}

func (f *flushingRecorder) Flush() {
	f.flushed = true
}

func TestServer_HandleRunStream_FlushCalled(t *testing.T) {
	mockAgt := &mockAgent{
		id: "test-agent",
		runStreamFn: func(_ context.Context, _ []agent.Message, _ ...agent.RunOption) (<-chan agent.ResponseUpdate, error) {
			updates := make(chan agent.ResponseUpdate, 10)
			go func() {
				defer close(updates)
				updates <- agent.ResponseUpdate{
					Kind: agent.UpdateKindDone,
				}
			}()
			return updates, nil
		},
	}

	server := NewServer(mockAgt)

	reqBody := RunRequest{
		Messages: []RequestMessage{{Role: "user", Content: "Hi!"}},
	}
	bodyBytes, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/stream", bytes.NewReader(bodyBytes))
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	// Verify response was written
	if rec.Body.Len() == 0 {
		t.Error("expected response body to contain events")
	}
}

func TestRunRequest_JSON(t *testing.T) {
	req := RunRequest{
		ThreadID: "thread-123",
		Messages: []RequestMessage{
			{Role: "user", Content: "Hello"},
		},
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var decoded RunRequest
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if decoded.ThreadID != req.ThreadID {
		t.Errorf("expected ThreadID %s, got %s", req.ThreadID, decoded.ThreadID)
	}
	if len(decoded.Messages) != 1 {
		t.Errorf("expected 1 message, got %d", len(decoded.Messages))
	}
	if decoded.Messages[0].Role != "user" {
		t.Errorf("expected role user, got %s", decoded.Messages[0].Role)
	}
}

func TestRequestMessage_JSON(t *testing.T) {
	msg := RequestMessage{
		Role:    "assistant",
		Content: "Hello, I'm an assistant!",
	}

	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var decoded RequestMessage
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if decoded.Role != msg.Role {
		t.Errorf("expected Role %s, got %s", msg.Role, decoded.Role)
	}
	if decoded.Content != msg.Content {
		t.Errorf("expected Content %s, got %s", msg.Content, decoded.Content)
	}
}

// BenchmarkServer_HandleRun benchmarks non-streaming run handling.
func BenchmarkServer_HandleRun(b *testing.B) {
	mockAgt := &mockAgent{
		id: "test-agent",
		runFn: func(_ context.Context, _ []agent.Message, _ ...agent.RunOption) (*agent.Response, error) {
			return &agent.Response{
				Messages: []agent.Message{agent.NewAssistantMessage("Hello!")},
			}, nil
		},
	}

	server := NewServer(mockAgt)
	handler := server.Handler()

	reqBody := RunRequest{
		ThreadID: "thread-123",
		Messages: []RequestMessage{{Role: "user", Content: "Hello!"}},
	}
	bodyBytes, _ := json.Marshal(reqBody)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodPost, "/run", bytes.NewReader(bodyBytes))
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
	}
}

// BenchmarkServer_HandleRunStream benchmarks streaming run handling.
func BenchmarkServer_HandleRunStream(b *testing.B) {
	mockAgt := &mockAgent{
		id: "test-agent",
		runStreamFn: func(_ context.Context, _ []agent.Message, _ ...agent.RunOption) (<-chan agent.ResponseUpdate, error) {
			updates := make(chan agent.ResponseUpdate, 10)
			go func() {
				defer close(updates)
				msg := agent.NewAssistantMessage("Hello!")
				updates <- agent.ResponseUpdate{
					Kind:    agent.UpdateKindMessageComplete,
					Message: &msg,
				}
			}()
			return updates, nil
		},
	}

	server := NewServer(mockAgt)
	handler := server.Handler()

	reqBody := RunRequest{
		ThreadID: "thread-123",
		Messages: []RequestMessage{{Role: "user", Content: "Hello!"}},
	}
	bodyBytes, _ := json.Marshal(reqBody)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodPost, "/stream", io.NopCloser(bytes.NewReader(bodyBytes)))
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
	}
}
