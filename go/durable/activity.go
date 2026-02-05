// Copyright (c) Microsoft. All rights reserved.

package durable

import (
	"context"
	"fmt"

	"github.com/microsoft/agent-framework-go/agent"
	"github.com/microsoft/agent-framework-go/chat"
	"go.temporal.io/sdk/activity"
)

// RunAgentActivityName is the registered name for the agent execution activity.
const RunAgentActivityName = "RunAgentActivity"

// ActivityInput is the input for the RunAgentActivity.
type ActivityInput struct {
	// SessionID identifies the session.
	SessionID SessionID `json:"sessionId"`

	// Messages is the complete conversation history to pass to the agent.
	Messages []chat.Message `json:"messages"`
}

// ActivityResult is the result of the RunAgentActivity.
type ActivityResult struct {
	// Messages contains the response messages from the agent.
	Messages []StateMessage `json:"messages"`

	// Usage contains token usage statistics.
	Usage *UsageInfo `json:"usage,omitempty"`

	// Error contains any error message.
	Error string `json:"error,omitempty"`
}

// agentContextKey is the context key for the agent instance.
type agentContextKey struct{}

// activityAgentKey is used to store the agent in activity context.
var activityAgentKey = agentContextKey{}

// ActivityContext provides context for activity execution.
type ActivityContext struct {
	// Agent is the agent instance to execute.
	Agent agent.Agent
}

// WithActivityContext returns a context with the activity context attached.
func WithActivityContext(ctx context.Context, actCtx ActivityContext) context.Context {
	return context.WithValue(ctx, activityAgentKey, actCtx)
}

// GetActivityContext retrieves the activity context from a context.
func GetActivityContext(ctx context.Context) (ActivityContext, bool) {
	actCtx, ok := ctx.Value(activityAgentKey).(ActivityContext)
	return actCtx, ok
}

// RunAgentActivity executes the agent and returns the response.
// This activity is registered with the Temporal worker and invoked by the workflow.
func RunAgentActivity(ctx context.Context, input ActivityInput) (ActivityResult, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Executing agent activity", "sessionId", input.SessionID)

	// Get the agent from activity context
	actCtx, ok := GetActivityContext(ctx)
	if !ok {
		return ActivityResult{Error: "agent not found in activity context"}, fmt.Errorf("agent not found in activity context")
	}

	if actCtx.Agent == nil {
		return ActivityResult{Error: "agent is nil"}, fmt.Errorf("agent is nil")
	}

	// agent.Message is an alias for chat.Message, so we can pass directly
	response, err := actCtx.Agent.Run(ctx, input.Messages)
	if err != nil {
		logger.Error("Agent execution failed", "error", err)
		return ActivityResult{Error: err.Error()}, err
	}

	// Convert response messages to state messages
	// agent.Message is an alias for chat.Message
	stateMessages := make([]StateMessage, 0, len(response.Messages))
	for _, msg := range response.Messages {
		stateMessages = append(stateMessages, FromChatMessage(msg))
	}

	// Extract usage information if available
	var usageInfo *UsageInfo
	if response.Usage != nil {
		usageInfo = &UsageInfo{
			InputTokenCount:  response.Usage.InputTokens,
			OutputTokenCount: response.Usage.OutputTokens,
			TotalTokenCount:  response.Usage.TotalTokens,
		}
	}

	logger.Info("Agent activity completed", "messageCount", len(stateMessages))

	return ActivityResult{
		Messages: stateMessages,
		Usage:    usageInfo,
	}, nil
}

// StreamingActivityResult is the result for streaming agent execution.
// Note: Temporal activities don't support true streaming, so this collects
// all updates and returns them at once.
type StreamingActivityResult struct {
	// Updates contains all streaming updates.
	Updates []StreamingUpdate `json:"updates"`

	// FinalMessages contains the final response messages.
	FinalMessages []StateMessage `json:"finalMessages"`

	// Usage contains token usage statistics.
	Usage *UsageInfo `json:"usage,omitempty"`

	// Error contains any error message.
	Error string `json:"error,omitempty"`
}

// StreamingUpdate represents a single streaming update.
type StreamingUpdate struct {
	// Type is the update type.
	Type string `json:"type"`

	// Content is the content delta.
	Content string `json:"content,omitempty"`

	// Message is the complete message (for message-complete updates).
	Message *StateMessage `json:"message,omitempty"`
}

// RunAgentStreamingActivityName is the registered name for the streaming agent activity.
const RunAgentStreamingActivityName = "RunAgentStreamingActivity"

// RunAgentStreamingActivity executes the agent with streaming and collects updates.
// Since Temporal activities can't stream, this collects all updates and returns them.
func RunAgentStreamingActivity(ctx context.Context, input ActivityInput) (StreamingActivityResult, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Executing streaming agent activity", "sessionId", input.SessionID)

	// Get the agent from activity context
	actCtx, ok := GetActivityContext(ctx)
	if !ok {
		return StreamingActivityResult{Error: "agent not found in activity context"}, fmt.Errorf("agent not found in activity context")
	}

	if actCtx.Agent == nil {
		return StreamingActivityResult{Error: "agent is nil"}, fmt.Errorf("agent is nil")
	}

	// agent.Message is an alias for chat.Message, so we can pass directly
	updateChan, err := actCtx.Agent.RunStream(ctx, input.Messages)
	if err != nil {
		logger.Error("Agent streaming execution failed", "error", err)
		return StreamingActivityResult{Error: err.Error()}, err
	}

	// Collect all updates
	var updates []StreamingUpdate
	var finalMessages []StateMessage
	var usageInfo *UsageInfo

	for update := range updateChan {
		switch update.Kind {
		case agent.UpdateKindContentDelta:
			var content string
			if update.Delta != nil {
				content = update.Delta.TextDelta
			}
			updates = append(updates, StreamingUpdate{
				Type:    "content_delta",
				Content: content,
			})
		case agent.UpdateKindMessageComplete:
			if update.Message != nil {
				// agent.Message is an alias for chat.Message
				stateMsg := FromChatMessage(*update.Message)
				updates = append(updates, StreamingUpdate{
					Type:    "message_complete",
					Message: &stateMsg,
				})
				finalMessages = append(finalMessages, stateMsg)
			}
		case agent.UpdateKindUsage:
			if update.Usage != nil {
				usageInfo = &UsageInfo{
					InputTokenCount:  update.Usage.InputTokens,
					OutputTokenCount: update.Usage.OutputTokens,
					TotalTokenCount:  update.Usage.TotalTokens,
				}
			}
		case agent.UpdateKindDone:
			// End of stream
		}
	}

	logger.Info("Streaming agent activity completed", "updateCount", len(updates), "messageCount", len(finalMessages))

	return StreamingActivityResult{
		Updates:       updates,
		FinalMessages: finalMessages,
		Usage:         usageInfo,
	}, nil
}
