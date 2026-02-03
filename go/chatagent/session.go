// Copyright (c) Microsoft. All rights reserved.

package chatagent

import (
	"encoding/json"
	"reflect"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/microsoft/agent-framework-go/agent"
	"github.com/microsoft/agent-framework-go/chat"
)

// Session implements agent.Session for ChatClientAgent.
// It maintains conversation history and is thread-safe.
type Session struct {
	id         string
	agentID    string
	messages   []chat.Message
	services   map[reflect.Type]interface{}
	mu         sync.RWMutex
	createdAt  time.Time
	modifiedAt time.Time
}

// Compile-time check that Session implements agent.Session.
var _ agent.Session = (*Session)(nil)

// sessionState represents the serializable state of a Session.
type sessionState struct {
	ID         string         `json:"id"`
	AgentID    string         `json:"agent_id,omitempty"`
	Messages   []chat.Message `json:"messages"`
	CreatedAt  time.Time      `json:"created_at"`
	ModifiedAt time.Time      `json:"modified_at"`
}

// newSession creates a new Session for the given agent.
func newSession(agentID string) *Session {
	now := time.Now()
	return &Session{
		id:         uuid.NewString(),
		agentID:    agentID,
		messages:   make([]chat.Message, 0),
		services:   make(map[reflect.Type]interface{}),
		createdAt:  now,
		modifiedAt: now,
	}
}

// restoreSession deserializes a Session from JSON data.
func restoreSession(data json.RawMessage) (*Session, error) {
	var state sessionState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, err
	}

	return &Session{
		id:         state.ID,
		agentID:    state.AgentID,
		messages:   state.Messages,
		services:   make(map[reflect.Type]interface{}),
		createdAt:  state.CreatedAt,
		modifiedAt: state.ModifiedAt,
	}, nil
}

// ID returns the unique identifier for this session.
func (s *Session) ID() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.id
}

// AgentID returns the ID of the agent this session belongs to.
func (s *Session) AgentID() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.agentID
}

// Messages returns a copy of the conversation history.
func (s *Session) Messages() []agent.Message {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Return a copy to prevent external modification
	result := make([]agent.Message, len(s.messages))
	copy(result, s.messages)
	return result
}

// AddMessage appends a message to the session history.
func (s *Session) AddMessage(msg agent.Message) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.messages = append(s.messages, msg)
	s.modifiedAt = time.Now()
}

// AddMessages appends multiple messages to the session history.
func (s *Session) AddMessages(msgs ...agent.Message) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.messages = append(s.messages, msgs...)
	s.modifiedAt = time.Now()
}

// ClearMessages removes all messages from the session.
func (s *Session) ClearMessages() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.messages = make([]chat.Message, 0)
	s.modifiedAt = time.Now()
}

// MessageCount returns the number of messages in the session.
func (s *Session) MessageCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.messages)
}

// Serialize converts the session state to JSON for persistence.
func (s *Session) Serialize() (json.RawMessage, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	state := sessionState{
		ID:         s.id,
		AgentID:    s.agentID,
		Messages:   s.messages,
		CreatedAt:  s.createdAt,
		ModifiedAt: s.modifiedAt,
	}

	return json.Marshal(state)
}

// GetService retrieves a service of the specified type from the session.
func (s *Session) GetService(serviceType reflect.Type) interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.services[serviceType]
}

// RegisterService registers a service that can be retrieved via GetService.
func (s *Session) RegisterService(service interface{}) {
	if service == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	serviceType := reflect.TypeOf(service)
	s.services[serviceType] = service
}

// CreatedAt returns when this session was created.
func (s *Session) CreatedAt() time.Time {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.createdAt
}

// ModifiedAt returns when this session was last modified.
func (s *Session) ModifiedAt() time.Time {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.modifiedAt
}

// Clone creates a deep copy of this session with a new ID.
func (s *Session) Clone() *Session {
	s.mu.RLock()
	defer s.mu.RUnlock()

	now := time.Now()
	messagesCopy := make([]chat.Message, len(s.messages))
	copy(messagesCopy, s.messages)

	return &Session{
		id:         uuid.NewString(),
		agentID:    s.agentID,
		messages:   messagesCopy,
		services:   make(map[reflect.Type]interface{}),
		createdAt:  now,
		modifiedAt: now,
	}
}
