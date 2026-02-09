// Copyright (c) Microsoft. All rights reserved.

package agui

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"

	"github.com/google/uuid"
	"github.com/microsoft/agent-framework-go/agent"
)

// Server exposes an agent via the AG-UI protocol using Server-Sent Events (SSE).
// It handles incoming requests, runs the agent, and streams AG-UI events to clients.
type Server struct {
	agent agent.Agent

	// mu protects activeConnections during concurrent access.
	mu sync.Mutex

	// activeConnections tracks active SSE connections for lifecycle management.
	// Maps connectionID to cancel function.
	activeConnections map[string]context.CancelFunc
}

// ServerOption configures the AG-UI server.
type ServerOption func(*serverOptions)

type serverOptions struct {
	// Reserved for future options
}

// NewServer creates a new AG-UI server for the given agent.
func NewServer(a agent.Agent, opts ...ServerOption) *Server {
	options := serverOptions{}
	for _, opt := range opts {
		opt(&options)
	}

	return &Server{
		agent:             a,
		activeConnections: make(map[string]context.CancelFunc),
	}
}

// Handler returns an http.Handler for the AG-UI endpoints.
// The handler exposes a single endpoint for streaming agent responses via SSE.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /", s.handleRun)
	mux.HandleFunc("POST /run", s.handleRun)
	mux.HandleFunc("POST /stream", s.handleRunStream)
	return mux
}

// RunRequest represents a request to run the agent.
type RunRequest struct {
	// ThreadID identifies the conversation thread.
	ThreadID string `json:"threadId,omitempty"`

	// RunID identifies the specific run within the thread.
	RunID string `json:"runId,omitempty"`

	// State contains optional state data for the run.
	State json.RawMessage `json:"state,omitempty"`

	// Messages contains the conversation history and new user input.
	Messages []RequestMessage `json:"messages"`

	// Tools contains optional tool definitions for the run.
	Tools []json.RawMessage `json:"tools,omitempty"`

	// Context contains optional context items for the run.
	Context []json.RawMessage `json:"context,omitempty"`

	// ForwardedProps contains optional forwarded properties.
	ForwardedProps json.RawMessage `json:"forwardedProps,omitempty"`
}

// RequestMessage represents a message in a run request.
type RequestMessage struct {
	// Role indicates the message author (e.g., "user", "assistant", "system").
	Role string `json:"role"`

	// Content is the message content.
	Content string `json:"content"`
}

// handleRun handles non-streaming run requests.
func (s *Server) handleRun(w http.ResponseWriter, r *http.Request) {
	var req RunRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if req.ThreadID == "" {
		req.ThreadID = uuid.New().String()
	}
	if req.RunID == "" {
		req.RunID = uuid.New().String()
	}

	// Convert request messages to agent messages
	agentMessages := s.toAgentMessages(req.Messages)
	options := s.buildRunOptions(req)

	// Run the agent
	response, err := s.agent.Run(r.Context(), agentMessages, options...)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Convert response to AG-UI events
	converter := NewEventConverter(req.ThreadID, req.RunID)
	var events []Event

	// Add run started event
	events = append(events, converter.RunStarted())

	// Convert response messages to events
	for _, msg := range response.Messages {
		update := agent.ResponseUpdate{
			Kind:       agent.UpdateKindMessageComplete,
			Message:    &msg,
			ResponseID: response.ResponseID,
		}
		events = append(events, converter.Convert(update)...)
	}

	// Add run finished event
	events = append(events, converter.RunFinished())

	// Return events as JSON array
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(events); err != nil {
		http.Error(w, "failed to encode events", http.StatusInternalServerError)
	}
}

// handleRunStream handles streaming run requests using SSE.
func (s *Server) handleRunStream(w http.ResponseWriter, r *http.Request) {
	var req RunRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if req.ThreadID == "" {
		req.ThreadID = uuid.New().String()
	}
	if req.RunID == "" {
		req.RunID = uuid.New().String()
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

	// Create connection context with cancellation
	connectionID := uuid.New().String()
	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	// Register connection for lifecycle management
	s.registerConnection(connectionID, cancel)
	defer s.unregisterConnection(connectionID)

	// Create event converter
	converter := NewEventConverter(req.ThreadID, req.RunID)

	// Send run started event
	s.writeSSEEvent(w, flusher, converter.RunStarted())

	// Convert request messages to agent messages
	agentMessages := s.toAgentMessages(req.Messages)
	options := s.buildRunOptions(req)

	// Run the agent in streaming mode
	updates, err := s.agent.RunStream(ctx, agentMessages, options...)
	if err != nil {
		s.writeSSEEvent(w, flusher, NewRunErrorEvent(err.Error()))
		return
	}

	// Stream updates as AG-UI events
	for update := range updates {
		// Check for context cancellation
		select {
		case <-ctx.Done():
			s.writeSSEEvent(w, flusher, NewRunErrorEvent("connection closed"))
			return
		default:
		}

		// Convert update to AG-UI events
		events := converter.Convert(update)
		for _, event := range events {
			s.writeSSEEvent(w, flusher, event)
		}
	}

	// Flush any remaining active messages and tool calls
	for _, event := range converter.FlushActiveMessages() {
		s.writeSSEEvent(w, flusher, event)
	}
	for _, event := range converter.FlushActiveToolCalls() {
		s.writeSSEEvent(w, flusher, event)
	}

	// Send run finished event
	s.writeSSEEvent(w, flusher, converter.RunFinished())
}

// registerConnection adds a connection to the active connections map.
func (s *Server) registerConnection(connectionID string, cancel context.CancelFunc) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.activeConnections[connectionID] = cancel
}

// unregisterConnection removes a connection from the active connections map.
func (s *Server) unregisterConnection(connectionID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.activeConnections, connectionID)
}

// CancelConnection cancels an active connection by ID.
// Returns true if the connection was found and canceled.
func (s *Server) CancelConnection(connectionID string) bool {
	s.mu.Lock()
	cancel, ok := s.activeConnections[connectionID]
	s.mu.Unlock()

	if ok {
		cancel()
		return true
	}
	return false
}

// ActiveConnectionCount returns the number of active SSE connections.
func (s *Server) ActiveConnectionCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.activeConnections)
}

// toAgentMessages converts request messages to agent messages.
func (s *Server) toAgentMessages(messages []RequestMessage) []agent.Message {
	result := make([]agent.Message, 0, len(messages))
	for _, msg := range messages {
		result = append(result, s.toAgentMessage(msg))
	}
	return result
}

// toAgentMessage converts a single request message to an agent message.
func (s *Server) toAgentMessage(msg RequestMessage) agent.Message {
	switch msg.Role {
	case "user":
		return agent.NewUserMessage(msg.Content)
	case "assistant":
		return agent.NewAssistantMessage(msg.Content)
	case "system":
		return agent.NewSystemMessage(msg.Content)
	default:
		return agent.NewUserMessage(msg.Content)
	}
}

func (s *Server) buildRunOptions(req RunRequest) []agent.RunOption {
	metadata := map[string]interface{}{}

	if req.ThreadID != "" {
		metadata["ag_ui_thread_id"] = req.ThreadID
	}
	if req.RunID != "" {
		metadata["ag_ui_run_id"] = req.RunID
	}
	if len(req.State) > 0 {
		metadata["ag_ui_state"] = decodeRawJSON(req.State)
	}
	if len(req.Context) > 0 {
		metadata["ag_ui_context"] = decodeRawJSONArray(req.Context)
	}
	if len(req.ForwardedProps) > 0 {
		metadata["ag_ui_forwarded_properties"] = decodeRawJSON(req.ForwardedProps)
	}

	var opts []agent.RunOption
	if len(metadata) > 0 {
		opts = append(opts, agent.WithMetadata(metadata))
	}

	if len(req.Tools) > 0 {
		tools := make([]interface{}, 0, len(req.Tools))
		for _, raw := range req.Tools {
			if value := decodeRawJSON(raw); value != nil {
				tools = append(tools, value)
			}
		}
		if len(tools) > 0 {
			opts = append(opts, agent.WithTools(tools...))
		}
	}

	return opts
}

func decodeRawJSON(raw json.RawMessage) interface{} {
	if len(raw) == 0 {
		return nil
	}

	var value interface{}
	if err := json.Unmarshal(raw, &value); err != nil {
		return string(raw)
	}
	return value
}

func decodeRawJSONArray(values []json.RawMessage) []interface{} {
	if len(values) == 0 {
		return nil
	}

	decoded := make([]interface{}, 0, len(values))
	for _, raw := range values {
		if value := decodeRawJSON(raw); value != nil {
			decoded = append(decoded, value)
		}
	}
	return decoded
}

// writeSSEEvent writes an AG-UI event as an SSE message.
func (s *Server) writeSSEEvent(w http.ResponseWriter, f http.Flusher, event Event) {
	data, err := json.Marshal(event)
	if err != nil {
		return
	}
	fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event.Type(), data)
	f.Flush()
}
