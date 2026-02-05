// Copyright (c) Microsoft. All rights reserved.

package a2a

import (
	"encoding/json"
	"reflect"
	"sync"

	"github.com/google/uuid"
	"github.com/microsoft/agent-framework-go/agent"
)

// A2ASession implements agent.Session for A2A protocol interactions.
// It maintains the context ID and task ID for continuing conversations
// with remote A2A agents.
type A2ASession struct {
	id        string
	contextID string
	taskID    string
	messages  []agent.Message
	services  map[reflect.Type]interface{}
	mu        sync.RWMutex
}

// a2aSessionState represents the serializable state of an A2ASession.
type a2aSessionState struct {
	ID        string          `json:"id"`
	ContextID string          `json:"context_id,omitempty"`
	TaskID    string          `json:"task_id,omitempty"`
	Messages  []agent.Message `json:"messages,omitempty"`
}

// NewA2ASession creates a new A2ASession with a generated UUID.
func NewA2ASession() *A2ASession {
	return &A2ASession{
		id:       uuid.New().String(),
		messages: make([]agent.Message, 0),
		services: make(map[reflect.Type]interface{}),
	}
}

// NewA2ASessionWithContextID creates a new A2ASession with an existing context ID.
// This is used to continue a conversation that was started with a different session.
func NewA2ASessionWithContextID(contextID string) *A2ASession {
	return &A2ASession{
		id:        uuid.New().String(),
		contextID: contextID,
		messages:  make([]agent.Message, 0),
		services:  make(map[reflect.Type]interface{}),
	}
}

// RestoreA2ASession deserializes a session from JSON data.
func RestoreA2ASession(data json.RawMessage) (*A2ASession, error) {
	var state a2aSessionState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, err
	}
	return &A2ASession{
		id:        state.ID,
		contextID: state.ContextID,
		taskID:    state.TaskID,
		messages:  state.Messages,
		services:  make(map[reflect.Type]interface{}),
	}, nil
}

// ID returns the unique identifier for this session.
func (s *A2ASession) ID() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.id
}

// ContextID returns the A2A context ID for grouping related tasks.
func (s *A2ASession) ContextID() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.contextID
}

// TaskID returns the current A2A task ID.
func (s *A2ASession) TaskID() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.taskID
}

// SetContextID sets the A2A context ID.
func (s *A2ASession) SetContextID(contextID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.contextID = contextID
}

// SetTaskID sets the current A2A task ID.
func (s *A2ASession) SetTaskID(taskID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.taskID = taskID
}

// Messages returns a copy of the conversation history for this session.
func (s *A2ASession) Messages() []agent.Message {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]agent.Message, len(s.messages))
	copy(result, s.messages)
	return result
}

// AddMessage appends a message to the session history.
func (s *A2ASession) AddMessage(msg agent.Message) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.messages = append(s.messages, msg)
}

// Serialize converts the session state to JSON for persistence.
func (s *A2ASession) Serialize() (json.RawMessage, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	state := a2aSessionState{
		ID:        s.id,
		ContextID: s.contextID,
		TaskID:    s.taskID,
		Messages:  s.messages,
	}

	return json.Marshal(state)
}

// GetService retrieves a service of the specified type from the session.
func (s *A2ASession) GetService(serviceType reflect.Type) interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.services[serviceType]
}

// RegisterService registers a service with the session.
func (s *A2ASession) RegisterService(serviceType reflect.Type, service interface{}) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.services[serviceType] = service
}

// updateFromTask updates the session state from an A2A task.
func (s *A2ASession) updateFromTask(task *Task) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Update context ID if provided
	if task.ContextID != "" {
		s.contextID = task.ContextID
	}

	// Update task ID
	s.taskID = task.ID
}

// Verify A2ASession implements agent.Session
var _ agent.Session = (*A2ASession)(nil)
