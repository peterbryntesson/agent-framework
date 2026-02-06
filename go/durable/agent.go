// Copyright (c) Microsoft. All rights reserved.

package durable

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/microsoft/agent-framework-go/agent"
	"github.com/microsoft/agent-framework-go/chat"
	"go.temporal.io/sdk/client"
)

// Agent wraps a standard agent with durable session management.
// It implements the agent.Agent interface and persists conversations
// using Temporal workflows.
type Agent struct {
	// inner is the wrapped agent that performs actual AI processing.
	inner agent.Agent

	// temporalClient is the Temporal client for workflow operations.
	temporalClient client.Client

	// name is the name used for session ID generation.
	name string

	// taskQueue is the Temporal task queue name.
	taskQueue string

	// timeToLive is the optional TTL for sessions.
	timeToLive time.Duration
}

// Ensure Agent implements agent.Agent.
var _ agent.Agent = (*Agent)(nil)

// NewAgent creates a new durable agent wrapper.
func NewAgent(inner agent.Agent, temporalClient client.Client, opts ...AgentOption) *Agent {
	a := &Agent{
		inner:          inner,
		temporalClient: temporalClient,
		name:           inner.Name(),
		taskQueue:      "durable-agent-queue",
	}

	for _, opt := range opts {
		opt(a)
	}

	return a
}

// AgentOption configures a durable Agent.
type AgentOption func(*Agent)

// WithName sets the agent name used for session ID generation.
func WithName(name string) AgentOption {
	return func(a *Agent) {
		a.name = name
	}
}

// WithTaskQueue sets the Temporal task queue for this agent.
func WithTaskQueue(queue string) AgentOption {
	return func(a *Agent) {
		a.taskQueue = queue
	}
}

// WithTimeToLive sets the TTL for durable sessions.
func WithTimeToLive(ttl time.Duration) AgentOption {
	return func(a *Agent) {
		a.timeToLive = ttl
	}
}

// ID returns the unique identifier for this agent.
func (a *Agent) ID() string {
	return a.inner.ID()
}

// Name returns the human-readable name of the agent.
func (a *Agent) Name() string {
	return a.name
}

// Description returns a description of the agent's purpose and capabilities.
func (a *Agent) Description() string {
	return a.inner.Description()
}

// Metadata returns provider-specific metadata about the agent.
func (a *Agent) Metadata() agent.AIAgentMetadata {
	return a.inner.Metadata()
}

// Run executes the agent within a durable session.
func (a *Agent) Run(ctx context.Context, messages []agent.Message, opts ...agent.RunOption) (*agent.Response, error) {
	// Apply standard run options
	cfg := agent.ApplyRunOptions(opts...)

	// Extract session ID from config metadata or generate new one
	sessionID, ok := getDurableSessionID(cfg, a.name)
	if !ok {
		sessionID = SessionID{
			Name: a.name,
			Key:  uuid.New().String(),
		}
	}
	requestOptions := getDurableRunRequestOptions(cfg)
	if requestOptions.CorrelationID == "" {
		requestOptions.CorrelationID = uuid.New().String()
	}
	if requestOptions.CreatedAt == nil {
		now := time.Now().UTC()
		requestOptions.CreatedAt = &now
	}

	// Convert messages to state messages
	// agent.Message is an alias for chat.Message
	stateMessages := make([]StateMessage, len(messages))
	for i, m := range messages {
		stateMessages[i] = FromChatMessage(m)
	}

	// Start or get the workflow
	workflowID := sessionID.WorkflowID()
	workflowOpts := client.StartWorkflowOptions{
		ID:        workflowID,
		TaskQueue: a.taskQueue,
	}

	// Try to start the workflow (it may already exist)
	_, err := a.temporalClient.ExecuteWorkflow(ctx, workflowOpts, SessionWorkflow, WorkflowInput{
		SessionID:  sessionID,
		TimeToLive: a.timeToLive,
	})
	if err != nil && !isWorkflowAlreadyStartedError(err) {
		return nil, fmt.Errorf("failed to start workflow: %w", err)
	}

	// Send the run request via workflow update
	updateStage := client.WorkflowUpdateStageCompleted
	if !requestOptions.WaitForReply {
		updateStage = client.WorkflowUpdateStageAccepted
	}

	updateHandle, err := a.temporalClient.UpdateWorkflow(ctx, client.UpdateWorkflowOptions{
		WorkflowID: workflowID,
		UpdateName: UpdateNameRun,
		Args: []interface{}{RunRequest{
			Messages:        stateMessages,
			CorrelationID:   requestOptions.CorrelationID,
			ResponseType:    requestOptions.ResponseType,
			ResponseSchema:  requestOptions.ResponseSchema,
			OrchestrationID: requestOptions.Orchestration,
			CreatedAt:       requestOptions.CreatedAt,
			WaitForResponse: requestOptions.WaitForReply,
			EnableToolCalls: requestOptions.EnableTools,
			Options:         requestOptions.Options,
		}},
		WaitForStage: updateStage,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to send run update: %w", err)
	}

	if !requestOptions.WaitForReply {
		return &agent.Response{
			Messages: []agent.Message{chat.NewSystemMessage(
				fmt.Sprintf("Request accepted for processing (correlation_id: %s).", requestOptions.CorrelationID),
			)},
			CreatedAt: time.Now(),
		}, nil
	}

	// Wait for the response
	var response RunResponse
	if err := updateHandle.Get(ctx, &response); err != nil {
		return nil, fmt.Errorf("failed to get run response: %w", err)
	}

	if response.Error != "" {
		return nil, errors.New(response.Error)
	}

	// Convert response messages back to agent format
	return a.convertRunResponse(response), nil
}

// RunStream executes the agent with streaming within a durable session.
// Note: Due to Temporal limitations, true streaming is not supported.
// This method collects all updates and returns them via the channel.
func (a *Agent) RunStream(ctx context.Context, messages []agent.Message, opts ...agent.RunOption) (<-chan agent.ResponseUpdate, error) {
	// For now, fall back to non-streaming and convert
	response, err := a.Run(ctx, messages, opts...)
	if err != nil {
		return nil, err
	}

	// Create a channel and emit the response as updates
	updateChan := make(chan agent.ResponseUpdate, len(response.Messages)+2)
	go func() {
		defer close(updateChan)

		updates := response.ToResponseUpdates()
		for _, update := range updates {
			select {
			case updateChan <- update:
			case <-ctx.Done():
				return
			}
		}
	}()

	return updateChan, nil
}

// NewSession creates a new agent session for maintaining conversation state.
func (a *Agent) NewSession(ctx context.Context) (agent.Session, error) {
	sessionID := SessionID{
		Name: a.name,
		Key:  uuid.New().String(),
	}
	return NewSession(sessionID), nil
}

// RestoreSession deserializes a previously saved session from JSON data.
func (a *Agent) RestoreSession(ctx context.Context, data json.RawMessage) (agent.Session, error) {
	var payload sessionPayload
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, fmt.Errorf("failed to unmarshal session: %w", err)
	}

	resolvedID := SessionID{}
	if payload.DurableSessionID != "" {
		parsed, err := ParseSessionID(payload.DurableSessionID)
		if err != nil {
			return nil, fmt.Errorf("failed to parse session ID: %w", err)
		}
		resolvedID = parsed
	}

	return NewSessionWithState(resolvedID, &payload.State), nil
}

// GetService retrieves a service of the specified type from the agent.
func (a *Agent) GetService(serviceType reflect.Type) interface{} {
	return a.inner.GetService(serviceType)
}

// GetSession retrieves the session state for a given session ID.
func (a *Agent) GetSession(ctx context.Context, sessionID SessionID) (*Session, error) {
	workflowID := sessionID.WorkflowID()

	// Query the workflow for state
	response, err := a.temporalClient.QueryWorkflow(ctx, workflowID, "", GetHistoryQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to query workflow: %w", err)
	}

	var state State
	if err := response.Get(&state); err != nil {
		return nil, fmt.Errorf("failed to decode state: %w", err)
	}

	return NewSessionWithState(sessionID, &state), nil
}

// DeleteSession terminates a session workflow.
func (a *Agent) DeleteSession(ctx context.Context, sessionID SessionID) error {
	workflowID := sessionID.WorkflowID()
	return a.temporalClient.TerminateWorkflow(ctx, workflowID, "", "session deleted")
}

// convertRunResponse converts a RunResponse to an agent.Response.
func (a *Agent) convertRunResponse(response RunResponse) *agent.Response {
	// agent.Message is an alias for chat.Message
	messages := make([]agent.Message, len(response.Messages))
	for i, m := range response.Messages {
		messages[i] = m.ToChatMessage()
	}

	resp := &agent.Response{
		Messages:  messages,
		CreatedAt: time.Now(),
	}

	if response.Usage != nil {
		resp.Usage = &agent.UsageDetails{
			InputTokens:  response.Usage.InputTokenCount,
			OutputTokens: response.Usage.OutputTokenCount,
			TotalTokens:  response.Usage.TotalTokenCount,
		}
	}

	return resp
}

// isWorkflowAlreadyStartedError checks if an error indicates the workflow already exists.
func isWorkflowAlreadyStartedError(err error) bool {
	if err == nil {
		return false
	}
	// Check for Temporal's workflow already started error
	// The serviceerror package has WorkflowExecutionAlreadyStarted
	errStr := err.Error()
	return strings.Contains(errStr, "already started") || strings.Contains(errStr, "WorkflowExecutionAlreadyStarted")
}
