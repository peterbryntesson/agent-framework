// Copyright (c) Microsoft. All rights reserved.

package a2a

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/microsoft/agent-framework-go/agent"
	"github.com/microsoft/agent-framework-go/chat"
)

// A2AAgent wraps an A2A Client as an agent.Agent implementation.
// This enables remote A2A agents to be used seamlessly as local agents,
// supporting both synchronous and streaming interactions.
type A2AAgent struct {
	client      *Client
	id          string
	name        string
	description string
	card        *AgentCard
	cardMu      sync.RWMutex
}

// A2AAgentOption configures the A2AAgent.
type A2AAgentOption func(*a2aAgentOptions)

type a2aAgentOptions struct {
	id          string
	name        string
	description string
}

// WithAgentID sets a custom agent ID.
func WithAgentID(id string) A2AAgentOption {
	return func(o *a2aAgentOptions) {
		o.id = id
	}
}

// WithAgentName sets a custom agent name.
func WithAgentName(name string) A2AAgentOption {
	return func(o *a2aAgentOptions) {
		o.name = name
	}
}

// WithAgentDescription sets a custom agent description.
func WithAgentDescription(description string) A2AAgentOption {
	return func(o *a2aAgentOptions) {
		o.description = description
	}
}

// NewA2AAgent creates a new A2AAgent that wraps an A2A client.
// The agent implements the agent.Agent interface, enabling remote
// A2A agents to be used as local agents.
func NewA2AAgent(client *Client, opts ...A2AAgentOption) *A2AAgent {
	options := a2aAgentOptions{}
	for _, opt := range opts {
		opt(&options)
	}

	agentID := options.id
	if agentID == "" {
		agentID = uuid.New().String()
	}

	return &A2AAgent{
		client:      client,
		id:          agentID,
		name:        options.name,
		description: options.description,
	}
}

// ID returns the unique identifier for this agent.
func (a *A2AAgent) ID() string {
	return a.id
}

// Name returns the human-readable name of the agent.
// If not set explicitly, attempts to fetch from the AgentCard.
func (a *A2AAgent) Name() string {
	if a.name != "" {
		return a.name
	}

	a.cardMu.RLock()
	if a.card != nil {
		name := a.card.Name
		a.cardMu.RUnlock()
		return name
	}
	a.cardMu.RUnlock()

	return ""
}

// Description returns the description of the agent's purpose.
// If not set explicitly, attempts to fetch from the AgentCard.
func (a *A2AAgent) Description() string {
	if a.description != "" {
		return a.description
	}

	a.cardMu.RLock()
	if a.card != nil {
		desc := a.card.Description
		a.cardMu.RUnlock()
		return desc
	}
	a.cardMu.RUnlock()

	return ""
}

// Metadata returns provider-specific metadata about the agent.
func (a *A2AAgent) Metadata() agent.AIAgentMetadata {
	return agent.AIAgentMetadata{
		ProviderName: "a2a",
	}
}

// Run executes the agent with the provided messages and returns a complete response.
// It creates or continues a task on the remote A2A agent and waits for completion.
func (a *A2AAgent) Run(ctx context.Context, messages []agent.Message, opts ...agent.RunOption) (*agent.Response, error) {
	if len(messages) == 0 {
		return nil, fmt.Errorf("at least one message is required")
	}

	cfg := agent.ApplyRunOptions(opts...)
	session := a.getOrCreateSession(cfg)

	// Convert messages to A2A format
	a2aMessage := a.convertMessagesToA2A(messages)

	// Get the A2A session for context tracking
	a2aSession, ok := session.(*A2ASession)
	if !ok {
		return nil, fmt.Errorf("invalid session type: expected *A2ASession")
	}

	var task *Task
	var err error

	if a2aSession.TaskID() == "" {
		// Create a new task
		req := &CreateTaskRequest{
			ContextID: a2aSession.ContextID(),
			Message:   a2aMessage,
			Metadata:  cfg.Metadata,
		}
		task, err = a.client.CreateTask(ctx, req)
	} else {
		// Send message to existing task
		task, err = a.client.SendMessage(ctx, a2aSession.TaskID(), a2aMessage)
	}

	if err != nil {
		return nil, fmt.Errorf("a2a request failed: %w", err)
	}

	// Update session with task context
	a2aSession.updateFromTask(task)

	// Convert A2A task to agent response
	return a.convertTaskToResponse(task), nil
}

// RunStream executes the agent and returns a channel of incremental response updates.
// It creates or continues a streaming task on the remote A2A agent.
func (a *A2AAgent) RunStream(ctx context.Context, messages []agent.Message, opts ...agent.RunOption) (<-chan agent.ResponseUpdate, error) {
	if len(messages) == 0 {
		return nil, fmt.Errorf("at least one message is required")
	}

	cfg := agent.ApplyRunOptions(opts...)
	session := a.getOrCreateSession(cfg)

	// Convert messages to A2A format
	a2aMessage := a.convertMessagesToA2A(messages)

	// Get the A2A session for context tracking
	a2aSession, ok := session.(*A2ASession)
	if !ok {
		return nil, fmt.Errorf("invalid session type: expected *A2ASession")
	}

	var taskID string
	if a2aSession.TaskID() == "" {
		// Create a new task first
		req := &CreateTaskRequest{
			ContextID: a2aSession.ContextID(),
			Message:   a2aMessage,
			Metadata:  cfg.Metadata,
		}
		task, err := a.client.CreateTask(ctx, req)
		if err != nil {
			return nil, fmt.Errorf("create task failed: %w", err)
		}
		taskID = task.ID
		a2aSession.updateFromTask(task)
	} else {
		taskID = a2aSession.TaskID()
	}

	// Start streaming
	events, err := a.client.SendMessageStream(ctx, taskID, a2aMessage)
	if err != nil {
		return nil, fmt.Errorf("stream request failed: %w", err)
	}

	updates := make(chan agent.ResponseUpdate, 100)
	go func() {
		defer close(updates)
		a.processStreamEvents(ctx, events, updates, a2aSession)
	}()

	return updates, nil
}

// NewSession creates a new agent session for maintaining conversation state.
func (a *A2AAgent) NewSession(ctx context.Context) (agent.Session, error) {
	return NewA2ASession(), nil
}

// RestoreSession deserializes a previously saved session from JSON data.
func (a *A2AAgent) RestoreSession(ctx context.Context, data json.RawMessage) (agent.Session, error) {
	return RestoreA2ASession(data)
}

// GetService retrieves a service of the specified type from the agent.
func (a *A2AAgent) GetService(serviceType reflect.Type) interface{} {
	switch {
	case serviceType == reflect.TypeOf((*Client)(nil)):
		return a.client
	case serviceType == reflect.TypeOf((*AgentCard)(nil)):
		a.cardMu.RLock()
		card := a.card
		a.cardMu.RUnlock()
		return card
	}
	return nil
}

// FetchAgentCard fetches and caches the remote agent's capability card.
// This can be used to discover the agent's capabilities before using it.
func (a *A2AAgent) FetchAgentCard(ctx context.Context) (*AgentCard, error) {
	card, err := a.client.GetAgentCard(ctx)
	if err != nil {
		return nil, err
	}

	a.cardMu.Lock()
	a.card = card
	a.cardMu.Unlock()

	return card, nil
}

// getOrCreateSession gets the session from config or creates a new one.
func (a *A2AAgent) getOrCreateSession(cfg *agent.RunConfig) agent.Session {
	if cfg.Session != nil {
		return cfg.Session
	}
	return NewA2ASession()
}

// convertMessagesToA2A converts agent messages to A2A message format.
func (a *A2AAgent) convertMessagesToA2A(messages []agent.Message) *Message {
	if len(messages) == 0 {
		return nil
	}

	// Use the last message as the primary message to send
	lastMsg := messages[len(messages)-1]

	parts := make([]Part, 0, len(lastMsg.Contents))
	for _, content := range lastMsg.Contents {
		switch c := content.(type) {
		case *chat.TextContent:
			parts = append(parts, NewTextPart(c.Text))
		case *chat.ImageContent:
			// Convert image content to file part
			if c.URL != "" {
				parts = append(parts, NewFilePart(c.URL, c.MediaType))
			}
		}
	}

	// Map chat role to A2A role
	var role MessageRole
	switch lastMsg.Role {
	case chat.RoleUser:
		role = RoleUser
	case chat.RoleAssistant:
		role = RoleAssistant
	case chat.RoleSystem:
		role = RoleSystem
	default:
		role = RoleUser
	}

	return &Message{
		Role:      role,
		Parts:     parts,
		CreatedAt: lastMsg.CreatedAt,
	}
}

// convertTaskToResponse converts an A2A task to an agent response.
func (a *A2AAgent) convertTaskToResponse(task *Task) *agent.Response {
	messages := make([]agent.Message, 0, len(task.Messages))

	for _, msg := range task.Messages {
		// Only include assistant messages in the response
		if msg.Role == RoleAssistant {
			messages = append(messages, a.convertA2AMessageToAgent(&msg))
		}
	}

	resp := &agent.Response{
		AgentID:    a.id,
		ResponseID: task.ID,
		Messages:   messages,
		CreatedAt:  task.CreatedAt,
		Metadata:   task.Metadata,
	}

	// Set finish reason based on task state
	switch task.State {
	case TaskStateCompleted:
		resp.FinishReason = agent.FinishReasonStop
	case TaskStateFailed:
		resp.FinishReason = agent.FinishReasonStop
	case TaskStateCancelled:
		resp.FinishReason = agent.FinishReasonStop
	}

	return resp
}

// convertA2AMessageToAgent converts an A2A message to an agent message.
func (a *A2AAgent) convertA2AMessageToAgent(msg *Message) agent.Message {
	contents := make([]chat.Content, 0, len(msg.Parts))

	for _, part := range msg.Parts {
		switch part.Type {
		case PartTypeText:
			contents = append(contents, chat.NewTextContent(part.Text))
		case PartTypeFile:
			// Convert file part to image content if it has a URI
			if part.URI != "" {
				contents = append(contents, &chat.ImageContent{
					URL:       part.URI,
					MediaType: part.MimeType,
				})
			}
		}
	}

	// Map A2A role to chat role
	var role chat.Role
	switch msg.Role {
	case RoleUser:
		role = chat.RoleUser
	case RoleAssistant:
		role = chat.RoleAssistant
	case RoleSystem:
		role = chat.RoleSystem
	default:
		role = chat.RoleUser
	}

	return chat.Message{
		Role:      role,
		Contents:  contents,
		CreatedAt: msg.CreatedAt,
	}
}

// processStreamEvents processes A2A stream events and converts them to agent updates.
func (a *A2AAgent) processStreamEvents(ctx context.Context, events <-chan StreamEvent, updates chan<- agent.ResponseUpdate, session *A2ASession) {
	for {
		select {
		case <-ctx.Done():
			updates <- agent.ResponseUpdate{
				Kind:  agent.UpdateKindError,
				Error: ctx.Err(),
			}
			return
		case event, ok := <-events:
			if !ok {
				// Channel closed - send done
				updates <- agent.ResponseUpdate{
					Kind:      agent.UpdateKindDone,
					CreatedAt: time.Now(),
				}
				return
			}

			if event.Error != nil {
				updates <- agent.ResponseUpdate{
					Kind:  agent.UpdateKindError,
					Error: event.Error,
				}
				continue
			}

			switch event.Type {
			case StreamEventTypeMessage:
				if event.Message != nil {
					agentMsg := a.convertA2AMessageToAgent(event.Message)
					updates <- agent.ResponseUpdate{
						Kind:      agent.UpdateKindMessageComplete,
						Message:   &agentMsg,
						Role:      string(agentMsg.Role),
						CreatedAt: event.Message.CreatedAt,
					}
				}

			case StreamEventTypeTask:
				if event.Task != nil {
					session.updateFromTask(event.Task)
					// Convert task messages to updates
					for _, msg := range event.Task.Messages {
						if msg.Role == RoleAssistant {
							agentMsg := a.convertA2AMessageToAgent(&msg)
							updates <- agent.ResponseUpdate{
								Kind:       agent.UpdateKindMessageComplete,
								Message:    &agentMsg,
								Role:       string(agentMsg.Role),
								ResponseID: event.Task.ID,
								CreatedAt:  msg.CreatedAt,
							}
						}
					}
				}

			case StreamEventTypeDone:
				updates <- agent.ResponseUpdate{
					Kind:         agent.UpdateKindDone,
					FinishReason: agent.FinishReasonStop,
					CreatedAt:    time.Now(),
				}
				return

			case StreamEventTypeError:
				updates <- agent.ResponseUpdate{
					Kind:  agent.UpdateKindError,
					Error: fmt.Errorf("stream error from A2A agent"),
				}
			}
		}
	}
}

// Verify A2AAgent implements agent.Agent
var _ agent.Agent = (*A2AAgent)(nil)
