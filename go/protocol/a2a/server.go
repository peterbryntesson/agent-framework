// Copyright (c) Microsoft. All rights reserved.

package a2a

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/microsoft/agent-framework-go/agent"
)

// Server exposes an agent via the A2A protocol.
type Server struct {
	agent    agent.Agent
	card     *AgentCard
	tasks    sync.Map // taskID -> *Task
	sessions sync.Map // contextID -> []string (task IDs)
}

// ServerOption configures the A2A server.
type ServerOption func(*serverOptions)

type serverOptions struct {
	card *AgentCard
}

// WithAgentCard sets a custom agent card for the server.
func WithAgentCard(card *AgentCard) ServerOption {
	return func(o *serverOptions) {
		o.card = card
	}
}

// NewServer creates a new A2A server for the given agent.
func NewServer(a agent.Agent, opts ...ServerOption) *Server {
	options := serverOptions{}
	for _, opt := range opts {
		opt(&options)
	}

	card := options.card
	if card == nil {
		card = &AgentCard{
			Name:        a.Name(),
			Description: a.Description(),
			Capabilities: &AgentCapabilities{
				Streaming: true,
			},
		}
	}

	return &Server{
		agent: a,
		card:  card,
	}
}

// Handler returns an http.Handler for the A2A endpoints.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /.well-known/agent.json", s.handleGetAgentCard)
	mux.HandleFunc("POST /tasks", s.handleCreateTask)
	mux.HandleFunc("GET /tasks/{taskID}", s.handleGetTask)
	mux.HandleFunc("POST /tasks/{taskID}/messages", s.handleSendMessage)
	mux.HandleFunc("POST /tasks/{taskID}/messages/stream", s.handleSendMessageStream)
	mux.HandleFunc("DELETE /tasks/{taskID}", s.handleCancelTask)
	return mux
}

// handleGetAgentCard returns the agent's capability card.
func (s *Server) handleGetAgentCard(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(s.card); err != nil {
		http.Error(w, "failed to encode agent card", http.StatusInternalServerError)
	}
}

// handleCreateTask creates a new task for the agent.
func (s *Server) handleCreateTask(w http.ResponseWriter, r *http.Request) {
	var req CreateTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if req.Message == nil {
		http.Error(w, "message is required", http.StatusBadRequest)
		return
	}

	task := &Task{
		ID:        uuid.New().String(),
		ContextID: req.ContextID,
		State:     TaskStatePending,
		Messages:  []Message{*req.Message},
		Metadata:  req.Metadata,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if task.ContextID == "" {
		task.ContextID = uuid.New().String()
	}

	s.tasks.Store(task.ID, task)
	s.addTaskToSession(task.ContextID, task.ID)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(task); err != nil {
		http.Error(w, "failed to encode task", http.StatusInternalServerError)
	}
}

// handleGetTask retrieves a task by ID.
func (s *Server) handleGetTask(w http.ResponseWriter, r *http.Request) {
	taskID := r.PathValue("taskID")
	task, ok := s.tasks.Load(taskID)
	if !ok {
		http.Error(w, "task not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(task); err != nil {
		http.Error(w, "failed to encode task", http.StatusInternalServerError)
	}
}

// handleSendMessage sends a message to a task and runs the agent.
func (s *Server) handleSendMessage(w http.ResponseWriter, r *http.Request) {
	taskID := r.PathValue("taskID")
	taskVal, ok := s.tasks.Load(taskID)
	if !ok {
		http.Error(w, "task not found", http.StatusNotFound)
		return
	}
	task := taskVal.(*Task)

	var req SendMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if req.Message == nil {
		http.Error(w, "message is required", http.StatusBadRequest)
		return
	}

	// Add message to task
	task.Messages = append(task.Messages, *req.Message)
	task.State = TaskStateRunning
	task.UpdatedAt = time.Now()
	s.tasks.Store(taskID, task)

	// Convert to agent messages and run
	agentMessages := s.toAgentMessages(task.Messages)
	response, err := s.agent.Run(r.Context(), agentMessages)
	if err != nil {
		task.State = TaskStateFailed
		task.Error = &TaskError{Code: "execution_error", Message: err.Error()}
		task.UpdatedAt = time.Now()
		s.tasks.Store(taskID, task)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Add response messages to task
	for _, msg := range response.Messages {
		task.Messages = append(task.Messages, s.toA2AMessage(msg))
	}
	task.State = TaskStateCompleted
	task.UpdatedAt = time.Now()
	s.tasks.Store(taskID, task)

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(task); err != nil {
		http.Error(w, "failed to encode task", http.StatusInternalServerError)
	}
}

// handleSendMessageStream sends a message and streams the response via SSE.
func (s *Server) handleSendMessageStream(w http.ResponseWriter, r *http.Request) {
	taskID := r.PathValue("taskID")
	taskVal, ok := s.tasks.Load(taskID)
	if !ok {
		http.Error(w, "task not found", http.StatusNotFound)
		return
	}
	task := taskVal.(*Task)

	var req SendMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if req.Message == nil {
		http.Error(w, "message is required", http.StatusBadRequest)
		return
	}

	// Set up SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

	// Add message and update state
	task.Messages = append(task.Messages, *req.Message)
	task.State = TaskStateRunning
	task.UpdatedAt = time.Now()
	s.tasks.Store(taskID, task)

	// Convert to agent messages and run stream
	agentMessages := s.toAgentMessages(task.Messages)
	updates, err := s.agent.RunStream(r.Context(), agentMessages)
	if err != nil {
		task.State = TaskStateFailed
		task.Error = &TaskError{Code: "execution_error", Message: err.Error()}
		task.UpdatedAt = time.Now()
		s.tasks.Store(taskID, task)
		s.writeSSEError(w, flusher, err)
		return
	}

	// Stream updates to client
	var accumulatedText string
	for update := range updates {
		if update.Error != nil {
			task.State = TaskStateFailed
			task.Error = &TaskError{Code: "stream_error", Message: update.Error.Error()}
			task.UpdatedAt = time.Now()
			s.tasks.Store(taskID, task)
			s.writeSSEError(w, flusher, update.Error)
			return
		}

		// Accumulate text from deltas
		if update.Kind == agent.UpdateKindContentDelta && update.Delta != nil {
			accumulatedText += update.Delta.TextDelta
		}

		// If message complete, add to task
		if update.Kind == agent.UpdateKindMessageComplete && update.Message != nil {
			task.Messages = append(task.Messages, s.toA2AMessage(*update.Message))
			task.UpdatedAt = time.Now()
			s.tasks.Store(taskID, task)
		}

		s.writeSSEUpdate(w, flusher, &update)
	}

	// Add accumulated text as message if no complete message was received
	if accumulatedText != "" {
		hasCompleteMessage := false
		for _, msg := range task.Messages {
			if msg.Role == RoleAssistant {
				hasCompleteMessage = true
				break
			}
		}
		if !hasCompleteMessage {
			assistantMsg := NewAssistantMessage(accumulatedText)
			task.Messages = append(task.Messages, assistantMsg)
		}
	}

	task.State = TaskStateCompleted
	task.UpdatedAt = time.Now()
	s.tasks.Store(taskID, task)

	// Send done event
	s.writeSSEDone(w, flusher, task)
}

// handleCancelTask cancels a running task.
func (s *Server) handleCancelTask(w http.ResponseWriter, r *http.Request) {
	taskID := r.PathValue("taskID")
	taskVal, ok := s.tasks.Load(taskID)
	if !ok {
		http.Error(w, "task not found", http.StatusNotFound)
		return
	}

	task := taskVal.(*Task)
	task.State = TaskStateCancelled
	task.UpdatedAt = time.Now()
	s.tasks.Store(taskID, task)

	w.WriteHeader(http.StatusNoContent)
}

// sessionTasks wraps a slice of task IDs for use in sync.Map.
// This wrapper enables thread-safe session management with mutex protection.
type sessionTasks struct {
	mu      sync.Mutex
	taskIDs []string
}

// addTaskToSession adds a task ID to a session's task list.
func (s *Server) addTaskToSession(contextID, taskID string) {
	// Try to load existing session or create new one
	existing, loaded := s.sessions.LoadOrStore(contextID, &sessionTasks{taskIDs: []string{taskID}})
	if !loaded {
		// New session was created with the task ID
		return
	}

	// Add to existing session with mutex protection
	session := existing.(*sessionTasks)
	session.mu.Lock()
	session.taskIDs = append(session.taskIDs, taskID)
	session.mu.Unlock()
}

// toAgentMessages converts A2A messages to agent messages.
func (s *Server) toAgentMessages(messages []Message) []agent.Message {
	result := make([]agent.Message, 0, len(messages))
	for _, msg := range messages {
		result = append(result, s.toAgentMessage(msg))
	}
	return result
}

// toAgentMessage converts a single A2A message to an agent message.
func (s *Server) toAgentMessage(msg Message) agent.Message {
	text := msg.Text()

	switch msg.Role {
	case RoleUser:
		return agent.NewUserMessage(text)
	case RoleAssistant:
		return agent.NewAssistantMessage(text)
	case RoleSystem:
		return agent.NewSystemMessage(text)
	default:
		return agent.NewUserMessage(text)
	}
}

// toA2AMessage converts an agent message to an A2A message.
func (s *Server) toA2AMessage(msg agent.Message) Message {
	return Message{
		Role:      MessageRole(msg.Role),
		Parts:     []Part{NewTextPart(msg.Text())},
		CreatedAt: msg.CreatedAt,
	}
}

// writeSSEUpdate writes a response update as an SSE event.
func (s *Server) writeSSEUpdate(w http.ResponseWriter, f http.Flusher, update *agent.ResponseUpdate) {
	data, err := json.Marshal(update)
	if err != nil {
		return
	}
	fmt.Fprintf(w, "event: update\ndata: %s\n\n", data)
	f.Flush()
}

// writeSSEError writes an error as an SSE event.
func (s *Server) writeSSEError(w http.ResponseWriter, f http.Flusher, err error) {
	errData := map[string]string{"message": err.Error()}
	data, _ := json.Marshal(errData)
	fmt.Fprintf(w, "event: error\ndata: %s\n\n", data)
	f.Flush()
}

// writeSSEDone writes a done event with the final task state.
func (s *Server) writeSSEDone(w http.ResponseWriter, f http.Flusher, task *Task) {
	data, _ := json.Marshal(task)
	fmt.Fprintf(w, "event: done\ndata: %s\n\n", data)
	f.Flush()
}

// GetTask retrieves a task by ID (for testing and external use).
func (s *Server) GetTask(taskID string) (*Task, bool) {
	taskVal, ok := s.tasks.Load(taskID)
	if !ok {
		return nil, false
	}
	return taskVal.(*Task), true
}

// GetSessionTasks returns all task IDs for a session (for testing and external use).
func (s *Server) GetSessionTasks(contextID string) []string {
	val, ok := s.sessions.Load(contextID)
	if !ok {
		return nil
	}
	session := val.(*sessionTasks)
	session.mu.Lock()
	defer session.mu.Unlock()
	result := make([]string, len(session.taskIDs))
	copy(result, session.taskIDs)
	return result
}
