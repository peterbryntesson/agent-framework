// Copyright (c) Microsoft. All rights reserved.

package durable

import (
	"encoding/json"
	"reflect"
	"sync"

	"github.com/microsoft/agent-framework-go/chat"
)

// Session represents a durable agent session that maintains conversation state
// within a Temporal workflow.
type Session struct {
	// sessionID is the unique identifier for this session.
	sessionID SessionID

	// state contains the persisted conversation history.
	state *State

	// services stores registered service instances.
	services map[reflect.Type]interface{}

	// mu protects concurrent access.
	mu sync.RWMutex
}

// NewSession creates a new durable session with the given session ID.
func NewSession(sessionID SessionID) *Session {
	return &Session{
		sessionID: sessionID,
		state:     NewState(),
		services:  make(map[reflect.Type]interface{}),
	}
}

// NewSessionWithState creates a new durable session with existing state.
// This is used when restoring a session from persistent storage.
func NewSessionWithState(sessionID SessionID, state *State) *Session {
	return &Session{
		sessionID: sessionID,
		state:     state,
		services:  make(map[reflect.Type]interface{}),
	}
}

// ID returns the unique identifier for this session.
// For durable sessions, this is the workflow ID.
func (s *Session) ID() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.sessionID.String()
}

// SessionID returns the structured session identifier.
func (s *Session) SessionID() SessionID {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.sessionID
}

// Messages returns the complete conversation history as chat.Message slice.
// This aggregates all messages from requests and responses in the history.
func (s *Session) Messages() []chat.Message {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.state.BuildChatMessages()
}

// AddMessage appends a message to the session.
// For durable sessions, messages should be added via AppendRequest/AppendResponse
// for proper request/response grouping.
func (s *Session) AddMessage(msg chat.Message) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// For single message additions, wrap in a request or response entry
	// based on the role
	if msg.Role == chat.RoleUser || msg.Role == chat.RoleSystem {
		entry := NewRequestEntry([]chat.Message{msg})
		s.state.AppendRequest(entry)
	} else {
		entry := NewResponseEntry([]chat.Message{msg})
		s.state.AppendResponse(entry)
	}
}

// Serialize converts the session state to JSON for persistence.
func (s *Session) Serialize() (json.RawMessage, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return json.Marshal(s.state)
}

// GetService retrieves a service of the specified type from the session.
func (s *Session) GetService(serviceType reflect.Type) interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.services[serviceType]
}

// RegisterService registers a service instance for the specified type.
func (s *Session) RegisterService(serviceType reflect.Type, service interface{}) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.services[serviceType] = service
}

// State returns the underlying state object.
// This provides direct access for workflow operations.
func (s *Session) State() *State {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.state
}

// AppendRequest adds a request to the conversation history.
func (s *Session) AppendRequest(messages []chat.Message, correlationID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	entry := NewRequestEntryWithCorrelation(messages, correlationID)
	s.state.AppendRequest(entry)
}

// AppendResponse adds a response to the conversation history.
func (s *Session) AppendResponse(messages []chat.Message, usage *UsageInfo) {
	s.mu.Lock()
	defer s.mu.Unlock()

	entry := NewResponseEntryWithUsage(messages, usage)
	s.state.AppendResponse(entry)
}

// MessageCount returns the total number of messages in the session.
func (s *Session) MessageCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.state.MessageCount()
}

// RestoreSession deserializes a durable session from JSON data.
func RestoreSession(sessionID SessionID, data json.RawMessage) (*Session, error) {
	var state State
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, err
	}
	return NewSessionWithState(sessionID, &state), nil
}
