// Copyright (c) Microsoft. All rights reserved.

package openai

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/microsoft/agent-framework-go/agent"
	"github.com/microsoft/agent-framework-go/hosting"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockAgent implements agent.Agent for testing.
type mockAgent struct {
	id          string
	name        string
	runFn       func(ctx context.Context, messages []agent.Message, opts ...agent.RunOption) (*agent.Response, error)
	runStreamFn func(ctx context.Context, messages []agent.Message, opts ...agent.RunOption) (<-chan agent.ResponseUpdate, error)
}

func (m *mockAgent) ID() string                      { return m.id }
func (m *mockAgent) Name() string                    { return m.name }
func (m *mockAgent) Description() string             { return "Test agent" }
func (m *mockAgent) Metadata() agent.AIAgentMetadata { return agent.AIAgentMetadata{} }

func (m *mockAgent) Run(ctx context.Context, messages []agent.Message, opts ...agent.RunOption) (*agent.Response, error) {
	if m.runFn != nil {
		return m.runFn(ctx, messages, opts...)
	}
	return &agent.Response{
		Messages: []agent.Message{
			agent.NewAssistantMessage("Hello from agent"),
		},
		FinishReason: agent.FinishReasonStop,
	}, nil
}

func (m *mockAgent) RunStream(ctx context.Context, messages []agent.Message, opts ...agent.RunOption) (<-chan agent.ResponseUpdate, error) {
	if m.runStreamFn != nil {
		return m.runStreamFn(ctx, messages, opts...)
	}
	ch := make(chan agent.ResponseUpdate, 2)
	go func() {
		ch <- agent.ResponseUpdate{
			Kind:  agent.UpdateKindContentDelta,
			Delta: &agent.ContentDelta{TextDelta: "Hello"},
		}
		ch <- agent.ResponseUpdate{
			Kind:         agent.UpdateKindDone,
			FinishReason: agent.FinishReasonStop,
		}
		close(ch)
	}()
	return ch, nil
}

func (m *mockAgent) NewSession(ctx context.Context) (agent.Session, error) {
	return &mockSession{id: "test-session"}, nil
}

func (m *mockAgent) RestoreSession(ctx context.Context, data json.RawMessage) (agent.Session, error) {
	return &mockSession{id: "restored-session", data: data}, nil
}

func (m *mockAgent) GetService(serviceType reflect.Type) interface{} { return nil }

// mockSession implements agent.Session for testing.
type mockSession struct {
	id   string
	data json.RawMessage
}

func (s *mockSession) ID() string                                      { return s.id }
func (s *mockSession) Messages() []agent.Message                       { return nil }
func (s *mockSession) AddMessage(msg agent.Message)                    {}
func (s *mockSession) Serialize() (json.RawMessage, error)             { return s.data, nil }
func (s *mockSession) GetService(serviceType reflect.Type) interface{} { return nil }

func TestNewHandler_DefaultOptions(t *testing.T) {
	// Arrange
	ag := &mockAgent{id: "agent-1", name: "Test Agent"}

	// Act
	handler := NewHandler(ag)

	// Assert
	assert.NotNil(t, handler)
	assert.Equal(t, "Test Agent", handler.modelName)
	assert.True(t, handler.streamingEnabled)
	assert.Equal(t, "/v1", handler.basePath)
}

func TestNewHandler_WithOptions(t *testing.T) {
	// Arrange
	ag := &mockAgent{id: "agent-1", name: "Test Agent"}
	store := hosting.NewInMemorySessionStore()

	// Act
	handler := NewHandler(ag,
		WithModelName("custom-model"),
		WithStreamingEnabled(false),
		WithSessionStore(store),
		WithBasePath("/api/v2"),
	)

	// Assert
	assert.Equal(t, "custom-model", handler.modelName)
	assert.False(t, handler.streamingEnabled)
	assert.Equal(t, "/api/v2", handler.basePath)
	assert.NotNil(t, handler.sessionStore)
}

func TestHandler_ListModels(t *testing.T) {
	// Arrange
	ag := &mockAgent{id: "agent-1", name: "my-model"}
	handler := NewHandler(ag)

	req := httptest.NewRequest("GET", "/v1/models", nil)
	w := httptest.NewRecorder()

	// Act
	handler.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

	var resp ModelsResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, "list", resp.Object)
	require.Len(t, resp.Data, 1)
	assert.Equal(t, "my-model", resp.Data[0].ID)
}

func TestHandler_GetModel(t *testing.T) {
	// Arrange
	ag := &mockAgent{id: "agent-1", name: "my-model"}
	handler := NewHandler(ag)

	req := httptest.NewRequest("GET", "/v1/models/my-model", nil)
	w := httptest.NewRecorder()

	// Act
	handler.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)

	var resp ModelInfo
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, "my-model", resp.ID)
	assert.Equal(t, "model", resp.Object)
}

func TestHandler_GetModel_NotFound(t *testing.T) {
	// Arrange
	ag := &mockAgent{id: "agent-1", name: "my-model"}
	handler := NewHandler(ag)

	req := httptest.NewRequest("GET", "/v1/models/unknown-model", nil)
	w := httptest.NewRecorder()

	// Act
	handler.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusNotFound, w.Code)

	var resp ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, "model_not_found", resp.Error.Type)
}

func TestHandler_ChatCompletions_NonStreaming(t *testing.T) {
	// Arrange
	ag := &mockAgent{id: "agent-1", name: "test-model"}
	handler := NewHandler(ag)

	reqBody := `{
		"model": "test-model",
		"messages": [
			{"role": "user", "content": "Hello"}
		],
		"stream": false
	}`

	req := httptest.NewRequest("POST", "/v1/chat/completions", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// Act
	handler.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

	var resp ChatCompletionResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, "chat.completion", resp.Object)
	assert.Equal(t, "test-model", resp.Model)
	require.Len(t, resp.Choices, 1)
	assert.Equal(t, "assistant", resp.Choices[0].Message.Role)
	assert.Equal(t, "Hello from agent", resp.Choices[0].Message.Content)
	assert.Equal(t, "stop", resp.Choices[0].FinishReason)
}

func TestHandler_ChatCompletions_InvalidRequest(t *testing.T) {
	// Arrange
	ag := &mockAgent{id: "agent-1", name: "test-model"}
	handler := NewHandler(ag)

	req := httptest.NewRequest("POST", "/v1/chat/completions", strings.NewReader("invalid json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// Act
	handler.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid_request_error", resp.Error.Type)
}

func TestHandler_ChatCompletions_EmptyMessages(t *testing.T) {
	// Arrange
	ag := &mockAgent{id: "agent-1", name: "test-model"}
	handler := NewHandler(ag)

	reqBody := `{"model": "test-model", "messages": []}`

	req := httptest.NewRequest("POST", "/v1/chat/completions", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// Act
	handler.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid_request_error", resp.Error.Type)
	assert.Contains(t, resp.Error.Message, "messages is required")
}

func TestHandler_ChatCompletions_WithToolCalls(t *testing.T) {
	// Arrange
	ag := &mockAgent{
		id:   "agent-1",
		name: "test-model",
		runFn: func(ctx context.Context, messages []agent.Message, opts ...agent.RunOption) (*agent.Response, error) {
			return &agent.Response{
				Messages: []agent.Message{
					agent.NewAssistantMessage("I'll check that for you"),
				},
				FinishReason: agent.FinishReasonToolCalls,
			}, nil
		},
	}
	handler := NewHandler(ag)

	reqBody := `{
		"model": "test-model",
		"messages": [{"role": "user", "content": "What's the weather?"}],
		"tools": [{"type": "function", "function": {"name": "get_weather"}}]
	}`

	req := httptest.NewRequest("POST", "/v1/chat/completions", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// Act
	handler.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandler_ChatCompletions_WithSessionStore(t *testing.T) {
	// Arrange
	store := hosting.NewInMemorySessionStore()
	ag := &mockAgent{id: "agent-1", name: "test-model"}
	handler := NewHandler(ag, WithSessionStore(store))

	reqBody := `{
		"model": "test-model",
		"messages": [{"role": "user", "content": "Hello"}]
	}`

	req := httptest.NewRequest("POST", "/v1/chat/completions", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Conversation-ID", "conv-123")
	w := httptest.NewRecorder()

	// Act
	handler.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestFinishReasonToString(t *testing.T) {
	tests := []struct {
		reason   agent.FinishReason
		expected string
	}{
		{agent.FinishReasonStop, "stop"},
		{agent.FinishReasonLength, "length"},
		{agent.FinishReasonToolCalls, "tool_calls"},
		{agent.FinishReasonContentFilter, "content_filter"},
		{agent.FinishReason(999), "stop"}, // Unknown defaults to stop
	}

	for _, tc := range tests {
		t.Run(tc.expected, func(t *testing.T) {
			result := finishReasonToString(tc.reason)
			assert.Equal(t, tc.expected, result)
		})
	}
}
