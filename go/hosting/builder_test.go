// Copyright (c) Microsoft. All rights reserved.

package hosting

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/google/uuid"
	"github.com/microsoft/agent-framework-go/agent"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// builderMockAgent implements agent.Agent for testing.
type builderMockAgent struct {
	id   string
	name string
}

func (m *builderMockAgent) ID() string                      { return m.id }
func (m *builderMockAgent) Name() string                    { return m.name }
func (m *builderMockAgent) Description() string             { return "" }
func (m *builderMockAgent) Metadata() agent.AIAgentMetadata { return agent.AIAgentMetadata{} }

func (m *builderMockAgent) Run(ctx context.Context, messages []agent.Message, opts ...agent.RunOption) (*agent.Response, error) {
	return nil, nil
}

func (m *builderMockAgent) RunStream(ctx context.Context, messages []agent.Message, opts ...agent.RunOption) (<-chan agent.ResponseUpdate, error) {
	return nil, nil
}

func (m *builderMockAgent) NewSession(ctx context.Context) (agent.Session, error) {
	return &builderMockSession{id: uuid.New().String()}, nil
}

func (m *builderMockAgent) RestoreSession(ctx context.Context, data json.RawMessage) (agent.Session, error) {
	var sessionData mockSessionData
	if err := json.Unmarshal(data, &sessionData); err != nil {
		return nil, err
	}
	return &builderMockSession{id: sessionData.ID}, nil
}

func (m *builderMockAgent) GetService(serviceType reflect.Type) interface{} { return nil }

// builderMockSession implements agent.Session for testing.
type builderMockSession struct {
	id   string
	data json.RawMessage
}

type mockSessionData struct {
	ID string `json:"id"`
}

func (s *builderMockSession) ID() string                   { return s.id }
func (s *builderMockSession) Messages() []agent.Message    { return nil }
func (s *builderMockSession) AddMessage(msg agent.Message) {}
func (s *builderMockSession) Serialize() (json.RawMessage, error) {
	return json.Marshal(mockSessionData{ID: s.id})
}
func (s *builderMockSession) GetService(serviceType reflect.Type) interface{} { return nil }

func TestHostedAgentBuilder_Build(t *testing.T) {
	// Arrange
	ag := &builderMockAgent{id: "agent-1", name: "Test Agent"}
	store := NewInMemorySessionStore()

	// Act
	hosted, err := NewHostedAgentBuilder("my-hosted-agent").
		WithAgent(ag).
		WithSessionStore(store).
		Build()
	require.NoError(t, err)

	// Assert
	assert.Equal(t, "my-hosted-agent", hosted.HostedAgentName())
	assert.Equal(t, "agent-1", hosted.ID())
	assert.Equal(t, "Test Agent", hosted.Name())
	assert.NotNil(t, hosted.SessionStore())
}

func TestHostedAgentBuilder_Build_DefaultSessionStore(t *testing.T) {
	// Arrange
	ag := &builderMockAgent{id: "agent-1", name: "Test Agent"}

	// Act
	hosted, err := NewHostedAgentBuilder("my-agent").
		WithAgent(ag).
		Build()
	require.NoError(t, err)

	// Assert
	assert.NotNil(t, hosted.SessionStore())
	_, isInMemory := hosted.SessionStore().(*InMemorySessionStore)
	assert.True(t, isInMemory)
}

func TestHostedAgentBuilder_Build_ReturnsErrorWithoutAgent(t *testing.T) {
	// Arrange
	builder := NewHostedAgentBuilder("my-agent")

	// Act
	_, err := builder.Build()

	// Assert
	assert.Error(t, err)
}

func TestHostedAgentBuilder_WithMiddleware(t *testing.T) {
	// Arrange
	ag := &builderMockAgent{id: "agent-1", name: "Test Agent"}

	middlewareCalled := false
	middleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			middlewareCalled = true
			next.ServeHTTP(w, r)
		})
	}

	// Act
	hosted, err := NewHostedAgentBuilder("my-agent").
		WithAgent(ag).
		WithMiddleware(middleware).
		Build()
	require.NoError(t, err)

	// Create a test handler
	innerHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrapped := hosted.WrapHandler(innerHandler)

	// Make a test request
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	wrapped.ServeHTTP(w, req)

	// Assert
	assert.True(t, middlewareCalled)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHostedAgentBuilder_MiddlewareOrder(t *testing.T) {
	// Arrange
	ag := &builderMockAgent{id: "agent-1", name: "Test Agent"}

	var callOrder []string

	middleware1 := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			callOrder = append(callOrder, "m1-before")
			next.ServeHTTP(w, r)
			callOrder = append(callOrder, "m1-after")
		})
	}

	middleware2 := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			callOrder = append(callOrder, "m2-before")
			next.ServeHTTP(w, r)
			callOrder = append(callOrder, "m2-after")
		})
	}

	// Act
	hosted, err := NewHostedAgentBuilder("my-agent").
		WithAgent(ag).
		WithMiddleware(middleware1).
		WithMiddleware(middleware2).
		Build()
	require.NoError(t, err)

	innerHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callOrder = append(callOrder, "handler")
	})

	wrapped := hosted.WrapHandler(innerHandler)

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	wrapped.ServeHTTP(w, req)

	// Assert - middleware1 is outermost (first added)
	expected := []string{"m1-before", "m2-before", "handler", "m2-after", "m1-after"}
	assert.Equal(t, expected, callOrder)
}

func TestHostedAgent_GetOrCreateSession(t *testing.T) {
	// Arrange
	ag := &builderMockAgent{id: "agent-1", name: "Test Agent"}
	store := NewInMemorySessionStore()

	hosted, err := NewHostedAgentBuilder("my-agent").
		WithAgent(ag).
		WithSessionStore(store).
		Build()
	require.NoError(t, err)

	ctx := context.Background()

	// Act - first call creates new session
	session1, err := hosted.GetOrCreateSession(ctx, "conv-1")
	require.NoError(t, err)
	require.NotNil(t, session1)

	// Save session
	err = hosted.SaveSession(ctx, "conv-1", session1)
	require.NoError(t, err)

	// Act - second call retrieves existing session
	session2, err := hosted.GetOrCreateSession(ctx, "conv-1")
	require.NoError(t, err)

	// Assert - should get same session ID back
	assert.Equal(t, session1.ID(), session2.ID())
}

func TestHostedAgent_DeleteSession(t *testing.T) {
	// Arrange
	ag := &builderMockAgent{id: "agent-1", name: "Test Agent"}
	store := NewInMemorySessionStore()

	hosted, err := NewHostedAgentBuilder("my-agent").
		WithAgent(ag).
		WithSessionStore(store).
		Build()
	require.NoError(t, err)

	ctx := context.Background()

	// Create and save session
	session, _ := hosted.GetOrCreateSession(ctx, "conv-1")
	_ = hosted.SaveSession(ctx, "conv-1", session)

	// Act
	err = hosted.DeleteSession(ctx, "conv-1")
	require.NoError(t, err)

	// Assert - next get should create new session
	session2, _ := hosted.GetOrCreateSession(ctx, "conv-1")
	assert.NotEqual(t, session.ID(), session2.ID())
}
