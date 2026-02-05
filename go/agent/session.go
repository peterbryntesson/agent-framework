// Copyright (c) Microsoft. All rights reserved.

package agent

import (
	"encoding/json"
	"reflect"
	"sync"

	"github.com/google/uuid"
)

// Session represents an agent conversation session.
// Sessions maintain conversation history and state across multiple agent runs.
type Session interface {
	// ID returns the unique identifier for this session.
	ID() string

	// Messages returns the conversation history for this session.
	Messages() []Message

	// AddMessage appends a message to the session history.
	AddMessage(msg Message)

	// Serialize converts the session state to JSON for persistence.
	Serialize() (json.RawMessage, error)

	// GetService retrieves a service of the specified type from the session.
	// Returns nil if the service type is not available.
	GetService(serviceType reflect.Type) interface{}
}

// inMemorySessionState represents the serializable state of an InMemorySession.
type inMemorySessionState struct {
	ID       string    `json:"id"`
	Messages []Message `json:"messages"`
}

// InMemorySession is an in-memory implementation of the Session interface.
// It is thread-safe and suitable for testing and simple use cases where
// persistence across process restarts is not required.
type InMemorySession struct {
	services map[reflect.Type]interface{}
	messages []Message
	id       string
	mu       sync.RWMutex
}

// NewInMemorySession creates a new InMemorySession with a generated UUID.
func NewInMemorySession() *InMemorySession {
	return &InMemorySession{
		id:       uuid.New().String(),
		messages: make([]Message, 0),
		services: make(map[reflect.Type]interface{}),
	}
}

// NewInMemorySessionWithID creates a new InMemorySession with the specified ID.
// This is useful for restoring sessions with known identifiers.
func NewInMemorySessionWithID(id string) *InMemorySession {
	return &InMemorySession{
		id:       id,
		messages: make([]Message, 0),
		services: make(map[reflect.Type]interface{}),
	}
}

// RestoreInMemorySession deserializes a session from JSON data.
// Returns an error if the data cannot be unmarshaled.
func RestoreInMemorySession(data json.RawMessage) (*InMemorySession, error) {
	var state inMemorySessionState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, err
	}
	return &InMemorySession{
		id:       state.ID,
		messages: state.Messages,
		services: make(map[reflect.Type]interface{}),
	}, nil
}

// ID returns the unique identifier for this session.
func (s *InMemorySession) ID() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.id
}

// Messages returns a copy of the conversation history for this session.
// The returned slice is a copy to prevent external modification.
func (s *InMemorySession) Messages() []Message {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]Message, len(s.messages))
	copy(result, s.messages)
	return result
}

// AddMessage appends a message to the session history.
// This method is thread-safe.
func (s *InMemorySession) AddMessage(msg Message) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.messages = append(s.messages, msg)
}

// Serialize converts the session state to JSON for persistence.
// The serialized data can be used with RestoreInMemorySession to restore the session.
func (s *InMemorySession) Serialize() (json.RawMessage, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	state := inMemorySessionState{
		ID:       s.id,
		Messages: s.messages,
	}
	return json.Marshal(state)
}

// GetService retrieves a service of the specified type from the session.
// Returns nil if the service type is not registered.
func (s *InMemorySession) GetService(serviceType reflect.Type) interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.services[serviceType]
}

// RegisterService registers a service instance for the specified type.
// This allows the session to provide services to agent implementations.
func (s *InMemorySession) RegisterService(serviceType reflect.Type, service interface{}) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.services[serviceType] = service
}
