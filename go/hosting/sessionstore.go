// Copyright (c) Microsoft. All rights reserved.

package hosting

import (
	"context"
	"encoding/json"
	"sync"

	"github.com/microsoft/agent-framework-go/agent"
)

// SessionStore defines the interface for storing and retrieving agent sessions.
//
// Session stores enable persistent conversations across HTTP requests,
// application restarts, or different service instances in hosted scenarios.
//
// Implementations must be safe for concurrent use from multiple goroutines.
type SessionStore interface {
	// SaveSession persists a session for the given agent and conversation.
	//
	// The session is serialized using session.Serialize() and stored
	// with a key derived from the agent ID and conversation ID.
	//
	// Returns an error if serialization or storage fails.
	SaveSession(ctx context.Context, ag agent.Agent, conversationID string, session agent.Session) error

	// GetSession retrieves a session for the given agent and conversation.
	//
	// If the session exists, it is deserialized using ag.RestoreSession().
	// If the session does not exist, a new session is created using ag.NewSession().
	//
	// Returns the session and any error that occurred.
	GetSession(ctx context.Context, ag agent.Agent, conversationID string) (agent.Session, error)

	// DeleteSession removes a session from storage.
	//
	// Returns nil if the session was deleted or did not exist.
	// Returns an error if the deletion failed.
	DeleteSession(ctx context.Context, ag agent.Agent, conversationID string) error
}

// makeKey computes the storage key from agent ID and conversation ID.
// Format: "{agentID}:{conversationID}" matching .NET convention.
func makeKey(agentID, conversationID string) string {
	return agentID + ":" + conversationID
}

// InMemorySessionStore provides a thread-safe in-memory session store.
//
// This implementation is suitable for single-instance deployments,
// development, and testing. For production multi-instance deployments,
// use a distributed session store implementation.
//
// Sessions are stored as json.RawMessage values, preserving the exact
// serialized form returned by Session.Serialize().
type InMemorySessionStore struct {
	sessions sync.Map // map[string]json.RawMessage
}

// NewInMemorySessionStore creates a new in-memory session store.
func NewInMemorySessionStore() *InMemorySessionStore {
	return &InMemorySessionStore{}
}

// SaveSession serializes and stores the session.
func (s *InMemorySessionStore) SaveSession(
	ctx context.Context,
	ag agent.Agent,
	conversationID string,
	session agent.Session,
) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}

	key := makeKey(ag.ID(), conversationID)

	data, err := session.Serialize()
	if err != nil {
		return err
	}

	s.sessions.Store(key, data)
	return nil
}

// GetSession retrieves an existing session or creates a new one.
func (s *InMemorySessionStore) GetSession(
	ctx context.Context,
	ag agent.Agent,
	conversationID string,
) (agent.Session, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	key := makeKey(ag.ID(), conversationID)

	if data, ok := s.sessions.Load(key); ok {
		rawData, _ := data.(json.RawMessage)
		return ag.RestoreSession(ctx, rawData)
	}

	// Session not found - create new session
	return ag.NewSession(ctx)
}

// DeleteSession removes a session from storage.
func (s *InMemorySessionStore) DeleteSession(
	ctx context.Context,
	ag agent.Agent,
	conversationID string,
) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}

	key := makeKey(ag.ID(), conversationID)
	s.sessions.Delete(key)
	return nil
}

// Compile-time interface check.
var _ SessionStore = (*InMemorySessionStore)(nil)

// NoopSessionStore is a session store that never persists sessions.
//
// Use this implementation when:
//   - Session persistence is not required
//   - Each request should start with a fresh session
//   - Testing stateless agent behavior
type NoopSessionStore struct{}

// NewNoopSessionStore creates a new no-op session store.
func NewNoopSessionStore() *NoopSessionStore {
	return &NoopSessionStore{}
}

// SaveSession does nothing and returns nil.
func (s *NoopSessionStore) SaveSession(
	ctx context.Context,
	ag agent.Agent,
	conversationID string,
	session agent.Session,
) error {
	return nil
}

// GetSession always creates a new session.
func (s *NoopSessionStore) GetSession(
	ctx context.Context,
	ag agent.Agent,
	conversationID string,
) (agent.Session, error) {
	return ag.NewSession(ctx)
}

// DeleteSession does nothing and returns nil.
func (s *NoopSessionStore) DeleteSession(
	ctx context.Context,
	ag agent.Agent,
	conversationID string,
) error {
	return nil
}

// Compile-time interface check.
var _ SessionStore = (*NoopSessionStore)(nil)
