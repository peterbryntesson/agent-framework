// Copyright (c) Microsoft. All rights reserved.

package hosting

import (
	"context"
	"encoding/json"
	"reflect"
	"sync"
	"testing"

	"github.com/microsoft/agent-framework-go/agent"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockAgent implements agent.Agent for testing.
type mockAgent struct {
	id           string
	newSessionFn func() (agent.Session, error)
	restoreFn    func(data json.RawMessage) (agent.Session, error)
}

func (m *mockAgent) ID() string                      { return m.id }
func (m *mockAgent) Name() string                    { return "test-agent" }
func (m *mockAgent) Description() string             { return "" }
func (m *mockAgent) Metadata() agent.AIAgentMetadata { return agent.AIAgentMetadata{} }

func (m *mockAgent) Run(ctx context.Context, messages []agent.Message, opts ...agent.RunOption) (*agent.Response, error) {
	return nil, nil
}

func (m *mockAgent) RunStream(ctx context.Context, messages []agent.Message, opts ...agent.RunOption) (<-chan agent.ResponseUpdate, error) {
	return nil, nil
}

func (m *mockAgent) NewSession(ctx context.Context) (agent.Session, error) {
	if m.newSessionFn != nil {
		return m.newSessionFn()
	}
	return &mockSession{id: "new-session"}, nil
}

func (m *mockAgent) RestoreSession(ctx context.Context, data json.RawMessage) (agent.Session, error) {
	if m.restoreFn != nil {
		return m.restoreFn(data)
	}
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

// TestMakeKey verifies the key format.
func TestMakeKey(t *testing.T) {
	key := makeKey("agent-123", "conv-456")
	assert.Equal(t, "agent-123:conv-456", key)
}

// TestInMemorySessionStore_SaveAndGet tests the save and get round-trip.
func TestInMemorySessionStore_SaveAndGet(t *testing.T) {
	store := NewInMemorySessionStore()
	ctx := context.Background()

	sessionData := json.RawMessage(`{"id":"session-1","messages":[]}`)
	session := &mockSession{id: "session-1", data: sessionData}
	ag := &mockAgent{
		id: "agent-1",
		restoreFn: func(data json.RawMessage) (agent.Session, error) {
			return &mockSession{id: "session-1", data: data}, nil
		},
	}

	// Save session
	err := store.SaveSession(ctx, ag, "conv-1", session)
	require.NoError(t, err)

	// Get session
	retrieved, err := store.GetSession(ctx, ag, "conv-1")
	require.NoError(t, err)
	assert.Equal(t, "session-1", retrieved.ID())
}

// TestInMemorySessionStore_GetCreatesNewWhenNotFound tests that GetSession creates a new session when not found.
func TestInMemorySessionStore_GetCreatesNewWhenNotFound(t *testing.T) {
	store := NewInMemorySessionStore()
	ctx := context.Background()

	ag := &mockAgent{
		id: "agent-1",
		newSessionFn: func() (agent.Session, error) {
			return &mockSession{id: "new-session"}, nil
		},
	}

	session, err := store.GetSession(ctx, ag, "unknown-conv")
	require.NoError(t, err)
	assert.Equal(t, "new-session", session.ID())
}

// TestInMemorySessionStore_Delete tests that DeleteSession removes a session.
func TestInMemorySessionStore_Delete(t *testing.T) {
	store := NewInMemorySessionStore()
	ctx := context.Background()

	sessionData := json.RawMessage(`{}`)
	session := &mockSession{id: "session-1", data: sessionData}
	ag := &mockAgent{
		id: "agent-1",
		newSessionFn: func() (agent.Session, error) {
			return &mockSession{id: "new-session"}, nil
		},
	}

	// Save and then delete
	err := store.SaveSession(ctx, ag, "conv-1", session)
	require.NoError(t, err)

	err = store.DeleteSession(ctx, ag, "conv-1")
	require.NoError(t, err)

	// Get should create new session since it was deleted
	retrieved, err := store.GetSession(ctx, ag, "conv-1")
	require.NoError(t, err)
	assert.Equal(t, "new-session", retrieved.ID())
}

// TestInMemorySessionStore_ConcurrentAccess tests thread safety.
func TestInMemorySessionStore_ConcurrentAccess(t *testing.T) {
	store := NewInMemorySessionStore()
	ctx := context.Background()

	ag := &mockAgent{
		id: "agent-1",
		newSessionFn: func() (agent.Session, error) {
			return &mockSession{id: "new", data: json.RawMessage(`{}`)}, nil
		},
		restoreFn: func(data json.RawMessage) (agent.Session, error) {
			return &mockSession{id: "restored", data: data}, nil
		},
	}

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			session := &mockSession{id: "s", data: json.RawMessage(`{}`)}
			_ = store.SaveSession(ctx, ag, "conv", session)
			_, _ = store.GetSession(ctx, ag, "conv")
		}(i)
	}
	wg.Wait()
}

// TestInMemorySessionStore_ContextCancellation tests context cancellation handling.
func TestInMemorySessionStore_ContextCancellation(t *testing.T) {
	store := NewInMemorySessionStore()
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	ag := &mockAgent{id: "agent-1"}
	session := &mockSession{id: "session-1", data: json.RawMessage(`{}`)}

	// All operations should return context error
	err := store.SaveSession(ctx, ag, "conv-1", session)
	assert.Equal(t, context.Canceled, err)

	_, err = store.GetSession(ctx, ag, "conv-1")
	assert.Equal(t, context.Canceled, err)

	err = store.DeleteSession(ctx, ag, "conv-1")
	assert.Equal(t, context.Canceled, err)
}

// TestInMemorySessionStore_MultipleAgents tests isolation between agents.
func TestInMemorySessionStore_MultipleAgents(t *testing.T) {
	store := NewInMemorySessionStore()
	ctx := context.Background()

	agent1 := &mockAgent{
		id: "agent-1",
		restoreFn: func(data json.RawMessage) (agent.Session, error) {
			return &mockSession{id: "agent1-session", data: data}, nil
		},
	}
	agent2 := &mockAgent{
		id: "agent-2",
		restoreFn: func(data json.RawMessage) (agent.Session, error) {
			return &mockSession{id: "agent2-session", data: data}, nil
		},
		newSessionFn: func() (agent.Session, error) {
			return &mockSession{id: "new-agent2-session"}, nil
		},
	}

	// Save session for agent1
	session1 := &mockSession{id: "agent1-session", data: json.RawMessage(`{"agent":"1"}`)}
	err := store.SaveSession(ctx, agent1, "conv-1", session1)
	require.NoError(t, err)

	// Get session for agent1 should return saved session
	retrieved1, err := store.GetSession(ctx, agent1, "conv-1")
	require.NoError(t, err)
	assert.Equal(t, "agent1-session", retrieved1.ID())

	// Get session for agent2 with same conv-1 should create new session
	retrieved2, err := store.GetSession(ctx, agent2, "conv-1")
	require.NoError(t, err)
	assert.Equal(t, "new-agent2-session", retrieved2.ID())
}

// TestNoopSessionStore_AlwaysCreatesNew tests that NoopSessionStore always creates new sessions.
func TestNoopSessionStore_AlwaysCreatesNew(t *testing.T) {
	store := NewNoopSessionStore()
	ctx := context.Background()

	callCount := 0
	ag := &mockAgent{
		id: "agent-1",
		newSessionFn: func() (agent.Session, error) {
			callCount++
			return &mockSession{id: "new"}, nil
		},
	}

	// Save does nothing
	session := &mockSession{id: "saved", data: json.RawMessage(`{}`)}
	err := store.SaveSession(ctx, ag, "conv-1", session)
	require.NoError(t, err)

	// Get always creates new
	_, err = store.GetSession(ctx, ag, "conv-1")
	require.NoError(t, err)
	assert.Equal(t, 1, callCount)

	// Get again creates another new session
	_, err = store.GetSession(ctx, ag, "conv-1")
	require.NoError(t, err)
	assert.Equal(t, 2, callCount)
}

// TestNoopSessionStore_DeleteDoesNothing tests that DeleteSession is a no-op.
func TestNoopSessionStore_DeleteDoesNothing(t *testing.T) {
	store := NewNoopSessionStore()
	ctx := context.Background()

	ag := &mockAgent{id: "agent-1"}

	err := store.DeleteSession(ctx, ag, "conv-1")
	require.NoError(t, err)
}

// TestNewInMemorySessionStore tests constructor.
func TestNewInMemorySessionStore(t *testing.T) {
	store := NewInMemorySessionStore()
	assert.NotNil(t, store)
}

// TestNewNoopSessionStore tests constructor.
func TestNewNoopSessionStore(t *testing.T) {
	store := NewNoopSessionStore()
	assert.NotNil(t, store)
}
