// Copyright (c) Microsoft. All rights reserved.

package durable

import (
	"errors"
	"time"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

// Workflow constants.
const (
	// UpdateNameRun is the name of the workflow update for running the agent.
	UpdateNameRun = "Run"

	// DefaultActivityTimeout is the default timeout for agent activities.
	DefaultActivityTimeout = 5 * time.Minute

	// DefaultMaxRetryAttempts is the default maximum retry attempts for activities.
	DefaultMaxRetryAttempts = 3

	// DefaultContinueAsNewThreshold is the history length at which to continue-as-new.
	DefaultContinueAsNewThreshold = 1000
)

// RunRequest is the input for the Run workflow update.
type RunRequest struct {
	// Messages are the new messages to process.
	Messages []StateMessage `json:"messages"`

	// CorrelationID is an optional correlation identifier.
	CorrelationID string `json:"correlationId,omitempty"`

	// ResponseType is the expected response type (e.g., "text", "json").
	ResponseType string `json:"responseType,omitempty"`

	// OrchestrationID is the ID of the calling orchestration.
	OrchestrationID string `json:"orchestrationId,omitempty"`
}

// RunResponse is the result of the Run workflow update.
type RunResponse struct {
	// Messages are the response messages from the agent.
	Messages []StateMessage `json:"messages"`

	// Usage contains token usage statistics.
	Usage *UsageInfo `json:"usage,omitempty"`

	// Error contains any error message if the run failed.
	Error string `json:"error,omitempty"`
}

// WorkflowInput is the input for the SessionWorkflow.
type WorkflowInput struct {
	// SessionID identifies this session.
	SessionID SessionID `json:"sessionId"`

	// TimeToLive is the optional TTL for this session.
	TimeToLive time.Duration `json:"timeToLive,omitempty"`

	// ExistingState is optional state to restore when continuing-as-new.
	ExistingState *State `json:"existingState,omitempty"`
}

// SessionWorkflow is the main Temporal workflow for durable agent sessions.
// It maintains conversation state and handles Run updates to execute the agent.
func SessionWorkflow(ctx workflow.Context, input WorkflowInput) error {
	logger := workflow.GetLogger(ctx)
	logger.Info("Starting session workflow", "sessionId", input.SessionID)

	// Initialize or restore state
	var state *State
	if input.ExistingState != nil {
		state = input.ExistingState
	} else {
		state = NewState()
	}

	// Set up TTL if configured
	if input.TimeToLive > 0 {
		expiration := workflow.Now(ctx).Add(input.TimeToLive)
		state.SetExpiration(expiration)
	}

	// Track if we need to continue-as-new due to long history
	historyLength := 0

	// Register the Run update handler
	err := workflow.SetUpdateHandlerWithOptions(
		ctx,
		UpdateNameRun,
		func(ctx workflow.Context, req RunRequest) (RunResponse, error) {
			return handleRunUpdate(ctx, input.SessionID, state, req, input.TimeToLive)
		},
		workflow.UpdateHandlerOptions{
			Validator: func(ctx workflow.Context, req RunRequest) error {
				if len(req.Messages) == 0 {
					return errors.New("messages cannot be empty")
				}
				return nil
			},
		},
	)
	if err != nil {
		return err
	}

	// Wait for TTL expiration or continue-as-new condition
	for {
		// Check if we need to continue-as-new
		historyLength++
		if historyLength >= DefaultContinueAsNewThreshold {
			logger.Info("Continuing as new due to history length")
			return workflow.NewContinueAsNewError(ctx, SessionWorkflow, WorkflowInput{
				SessionID:     input.SessionID,
				TimeToLive:    input.TimeToLive,
				ExistingState: state,
			})
		}

		// Calculate sleep duration
		var sleepDuration time.Duration
		if state.Data.ExpirationTimeUtc != nil {
			sleepDuration = state.Data.ExpirationTimeUtc.Sub(workflow.Now(ctx))
			if sleepDuration <= 0 {
				// Session has expired
				logger.Info("Session expired", "sessionId", input.SessionID)
				return nil
			}
		} else {
			// No TTL, sleep for a long time and wake on updates
			sleepDuration = 24 * time.Hour
		}

		// Use a selector to wait for either timer or signal
		selector := workflow.NewSelector(ctx)

		timerFuture := workflow.NewTimer(ctx, sleepDuration)
		selector.AddFuture(timerFuture, func(f workflow.Future) {
			// Timer fired - check if we should expire or continue
		})

		selector.Select(ctx)

		// Check if session should expire
		if state.Data.ExpirationTimeUtc != nil && workflow.Now(ctx).After(*state.Data.ExpirationTimeUtc) {
			logger.Info("Session expired after timer", "sessionId", input.SessionID)
			return nil
		}
	}
}

// handleRunUpdate handles a Run update request.
func handleRunUpdate(
	ctx workflow.Context,
	sessionID SessionID,
	state *State,
	req RunRequest,
	ttl time.Duration,
) (RunResponse, error) {
	logger := workflow.GetLogger(ctx)
	logger.Info("Processing run update", "sessionId", sessionID, "messageCount", len(req.Messages))

	// Add request to conversation history
	requestEntry := &RequestEntry{
		Type:            "request",
		Timestamp:       workflow.Now(ctx),
		Messages:        req.Messages,
		CorrelationID:   req.CorrelationID,
		OrchestrationID: req.OrchestrationID,
		ResponseType:    req.ResponseType,
	}
	state.AppendRequest(requestEntry)

	// Build the full conversation history for the agent
	allMessages := state.BuildChatMessages()

	// Execute the agent activity
	activityInput := ActivityInput{
		SessionID: sessionID,
		Messages:  allMessages,
	}

	activityOpts := workflow.ActivityOptions{
		StartToCloseTimeout: DefaultActivityTimeout,
		RetryPolicy: &temporal.RetryPolicy{
			MaximumAttempts: DefaultMaxRetryAttempts,
		},
	}
	ctx = workflow.WithActivityOptions(ctx, activityOpts)

	var activityResult ActivityResult
	err := workflow.ExecuteActivity(ctx, RunAgentActivityName, activityInput).Get(ctx, &activityResult)
	if err != nil {
		logger.Error("Activity failed", "error", err)
		return RunResponse{Error: err.Error()}, nil
	}

	// Add response to conversation history
	responseEntry := &ResponseEntry{
		Type:          "response",
		Timestamp:     workflow.Now(ctx),
		Messages:      activityResult.Messages,
		CorrelationID: req.CorrelationID,
		Usage:         activityResult.Usage,
	}
	state.AppendResponse(responseEntry)

	// Update TTL expiration if configured
	if ttl > 0 {
		newExpiration := workflow.Now(ctx).Add(ttl)
		state.SetExpiration(newExpiration)
	}

	return RunResponse{
		Messages: activityResult.Messages,
		Usage:    activityResult.Usage,
	}, nil
}

// GetHistoryQuery is a query to retrieve the conversation history.
const GetHistoryQuery = "GetHistory"
