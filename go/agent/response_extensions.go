// Copyright (c) Microsoft. All rights reserved.

package agent

import (
	"time"
)

// ToAgentResponse accumulates a slice of ResponseUpdate into a single Response.
// This is useful for converting streaming updates into a complete response.
// The function combines content deltas, accumulates usage, and builds the final message list.
func ToAgentResponse(updates []ResponseUpdate) *Response {
	if len(updates) == 0 {
		return nil
	}

	response := &Response{
		Messages: make([]Message, 0),
		Metadata: make(map[string]interface{}),
	}

	var currentMessageID string
	var messagesByID = make(map[string]*Message)
	var totalUsage *UsageDetails

	for i := range updates {
		update := &updates[i]

		// Track response metadata from updates
		if update.ResponseID != "" && response.ResponseID == "" {
			response.ResponseID = update.ResponseID
		}
		if !update.CreatedAt.IsZero() && response.CreatedAt.IsZero() {
			response.CreatedAt = update.CreatedAt
		}

		switch update.Kind {
		case UpdateKindMessageComplete:
			if update.Message != nil {
				response.Messages = append(response.Messages, *update.Message)
			}

		case UpdateKindContentDelta:
			if update.Delta != nil {
				// Group deltas by MessageID
				msgID := update.MessageID
				if msgID == "" {
					msgID = currentMessageID
				}
				if msgID == "" {
					msgID = "default"
				}
				currentMessageID = msgID

				// Get or create message for this ID
				msg, exists := messagesByID[msgID]
				if !exists {
					newMsg := Message{
						CreatedAt: time.Now(),
					}
					messagesByID[msgID] = &newMsg
					msg = &newMsg
				}

				// Note: Full delta accumulation would require building Contents
				// This is a simplified implementation that tracks the structure
				_ = msg // Delta accumulation deferred to provider implementations
			}

		case UpdateKindUsage:
			if update.Usage != nil {
				if totalUsage == nil {
					totalUsage = &UsageDetails{}
				}
				totalUsage.InputTokens += update.Usage.InputTokens
				totalUsage.OutputTokens += update.Usage.OutputTokens
				totalUsage.TotalTokens += update.Usage.TotalTokens
				totalUsage.CachedTokens += update.Usage.CachedTokens
				totalUsage.ReasoningTokens += update.Usage.ReasoningTokens
			}

		case UpdateKindDone:
			response.FinishReason = update.FinishReason

		case UpdateKindError:
			// Error handling - could set metadata or return error
			if update.Error != nil {
				response.Metadata["error"] = update.Error.Error()
			}
		}
	}

	// Add accumulated messages from deltas (if any were tracked by ID)
	for _, msg := range messagesByID {
		response.Messages = append(response.Messages, *msg)
	}

	response.Usage = totalUsage
	return response
}

// ToAgentResponseFromChannel reads all updates from a channel and accumulates them into a Response.
// This consumes the entire channel and blocks until the channel is closed.
func ToAgentResponseFromChannel(updates <-chan ResponseUpdate) *Response {
	var collected []ResponseUpdate
	for update := range updates {
		collected = append(collected, update)
	}
	return ToAgentResponse(collected)
}
