// Copyright (c) Microsoft. All rights reserved.

package agui

import (
	"github.com/google/uuid"
	"github.com/microsoft/agent-framework-go/agent"
	"github.com/microsoft/agent-framework-go/chat"
)

// EventConverter converts agent.ResponseUpdate values to AG-UI events.
// It maintains state to track message and tool call lifecycles, generating
// appropriate start/content/end events for streaming responses.
type EventConverter struct {
	// threadID identifies the conversation thread.
	threadID string

	// runID identifies this specific run within the thread.
	runID string

	// activeMessages tracks messages that have started but not ended.
	// Maps messageID to role for proper end event generation.
	activeMessages map[string]string

	// activeToolCalls tracks tool calls that have started but not ended.
	// Maps toolCallID to tool name for tracking.
	activeToolCalls map[string]string

	// messageCounter generates unique message IDs when not provided.
	messageCounter int
}

// NewEventConverter creates a new EventConverter for the given thread and run.
func NewEventConverter(threadID, runID string) *EventConverter {
	return &EventConverter{
		threadID:        threadID,
		runID:           runID,
		activeMessages:  make(map[string]string),
		activeToolCalls: make(map[string]string),
	}
}

// ThreadID returns the thread identifier.
func (c *EventConverter) ThreadID() string {
	return c.threadID
}

// RunID returns the run identifier.
func (c *EventConverter) RunID() string {
	return c.runID
}

// Convert converts a single ResponseUpdate to zero or more AG-UI events.
// The returned slice may contain multiple events for a single update
// (e.g., start and content events for a new message with text).
func (c *EventConverter) Convert(update agent.ResponseUpdate) []Event {
	switch update.Kind {
	case agent.UpdateKindContentDelta:
		return c.convertContentDelta(update)

	case agent.UpdateKindToolCall:
		return c.convertToolCall(update)

	case agent.UpdateKindToolResult:
		return c.convertToolResult(update)

	case agent.UpdateKindMessageComplete:
		return c.convertMessageComplete(update)

	case agent.UpdateKindError:
		return c.convertError(update)

	case agent.UpdateKindDone:
		return c.convertDone(update)

	case agent.UpdateKindUsage:
		// Usage updates don't map to AG-UI events
		return nil

	default:
		return nil
	}
}

// ConvertAll converts a slice of ResponseUpdate to AG-UI events.
// This is a convenience method for processing multiple updates at once.
func (c *EventConverter) ConvertAll(updates []agent.ResponseUpdate) []Event {
	var events []Event
	for _, update := range updates {
		events = append(events, c.Convert(update)...)
	}
	return events
}

// RunStarted returns a RunStartedEvent for this converter's thread and run.
// Call this at the beginning of a streaming response.
func (c *EventConverter) RunStarted() *RunStartedEvent {
	return NewRunStartedEvent(c.threadID, c.runID)
}

// RunFinished returns a RunFinishedEvent for this converter's thread and run.
// Call this at the end of a successful streaming response.
func (c *EventConverter) RunFinished() *RunFinishedEvent {
	return NewRunFinishedEvent(c.threadID, c.runID)
}

// FlushActiveMessages generates end events for any messages that were started
// but not explicitly ended. This should be called before RunFinished.
func (c *EventConverter) FlushActiveMessages() []Event {
	var events []Event
	for messageID := range c.activeMessages {
		events = append(events, NewTextMessageEndEvent(messageID))
		delete(c.activeMessages, messageID)
	}
	return events
}

// FlushActiveToolCalls generates end events for any tool calls that were started
// but not explicitly ended. This should be called before RunFinished.
func (c *EventConverter) FlushActiveToolCalls() []Event {
	var events []Event
	for toolCallID := range c.activeToolCalls {
		events = append(events, NewToolCallEndEvent(toolCallID))
		delete(c.activeToolCalls, toolCallID)
	}
	return events
}

// convertContentDelta handles incremental content updates.
func (c *EventConverter) convertContentDelta(update agent.ResponseUpdate) []Event {
	if update.Delta == nil {
		return nil
	}

	var events []Event
	delta := update.Delta

	// Handle text deltas
	if delta.TextDelta != "" {
		messageID := update.MessageID
		if messageID == "" {
			messageID = c.generateMessageID()
		}

		// Start message if not already active
		if _, active := c.activeMessages[messageID]; !active {
			role := delta.Role
			if role == "" {
				role = string(chat.RoleAssistant)
			}
			events = append(events, NewTextMessageStartEvent(messageID, role))
			c.activeMessages[messageID] = role
		}

		events = append(events, NewTextMessageContentEvent(messageID, delta.TextDelta))
	}

	// Handle tool call argument deltas
	if delta.ArgsDelta != "" && delta.ToolCallID != "" {
		// Start tool call if not already active
		if _, active := c.activeToolCalls[delta.ToolCallID]; !active {
			name := delta.Name
			if name == "" {
				name = "unknown"
			}
			events = append(events, NewToolCallStartEvent(delta.ToolCallID, name))
			c.activeToolCalls[delta.ToolCallID] = name
		}

		events = append(events, NewToolCallArgsEvent(delta.ToolCallID, delta.ArgsDelta))
	}

	return events
}

// convertToolCall handles tool call updates.
func (c *EventConverter) convertToolCall(update agent.ResponseUpdate) []Event {
	if update.Delta == nil {
		return nil
	}

	delta := update.Delta
	if delta.ToolCallID == "" {
		return nil
	}

	var events []Event

	// Start tool call if not already active
	if _, active := c.activeToolCalls[delta.ToolCallID]; !active {
		name := delta.Name
		if name == "" {
			name = "unknown"
		}
		events = append(events, NewToolCallStartEvent(delta.ToolCallID, name))
		c.activeToolCalls[delta.ToolCallID] = name
	}

	// If there are arguments, emit args event
	if delta.ArgsDelta != "" {
		events = append(events, NewToolCallArgsEvent(delta.ToolCallID, delta.ArgsDelta))
	}

	return events
}

// convertToolResult handles tool result updates.
func (c *EventConverter) convertToolResult(update agent.ResponseUpdate) []Event {
	if update.Message == nil {
		return nil
	}

	var events []Event

	// End any active tool call for this result
	toolCallID := update.Message.ToolCallID
	if toolCallID != "" {
		if _, active := c.activeToolCalls[toolCallID]; active {
			events = append(events, NewToolCallEndEvent(toolCallID))
			delete(c.activeToolCalls, toolCallID)
		}

		// Emit tool result event
		content := update.Message.Text()
		resultEvent := NewToolCallResultEvent(toolCallID, content)
		if update.MessageID != "" {
			resultEvent = resultEvent.WithMessageID(update.MessageID)
		}
		resultEvent = resultEvent.WithRole(string(update.Message.Role))
		events = append(events, resultEvent)
	}

	return events
}

// convertMessageComplete handles complete message updates.
func (c *EventConverter) convertMessageComplete(update agent.ResponseUpdate) []Event {
	if update.Message == nil {
		return nil
	}

	var events []Event
	msg := update.Message

	// Generate message ID
	messageID := update.MessageID
	if messageID == "" {
		messageID = c.generateMessageID()
	}

	// Handle text content
	text := msg.Text()
	if text != "" {
		role := string(msg.Role)
		if role == "" {
			role = string(chat.RoleAssistant)
		}

		// If this message wasn't streamed incrementally, emit full lifecycle
		if _, active := c.activeMessages[messageID]; !active {
			events = append(events, NewTextMessageStartEvent(messageID, role))
			events = append(events, NewTextMessageContentEvent(messageID, text))
			events = append(events, NewTextMessageEndEvent(messageID))
		} else {
			// Message was already started via deltas, just end it
			events = append(events, NewTextMessageEndEvent(messageID))
			delete(c.activeMessages, messageID)
		}
	}

	// Handle tool calls in the message
	for _, toolCall := range msg.ToolCalls {
		if _, active := c.activeToolCalls[toolCall.ID]; !active {
			events = append(events, NewToolCallStartEvent(toolCall.ID, toolCall.Name))
		}
		if len(toolCall.Arguments) > 0 {
			events = append(events, NewToolCallArgsEvent(toolCall.ID, string(toolCall.Arguments)))
		}
		events = append(events, NewToolCallEndEvent(toolCall.ID))
		delete(c.activeToolCalls, toolCall.ID)
	}

	// Handle tool result content
	for _, content := range msg.Contents {
		if trc, ok := content.(*chat.ToolResultContent); ok {
			events = append(events, NewToolCallResultEvent(trc.ToolCallID, trc.Content))
		}
	}

	return events
}

// convertError handles error updates.
func (c *EventConverter) convertError(update agent.ResponseUpdate) []Event {
	message := "unknown error"
	if update.Error != nil {
		message = update.Error.Error()
	}
	return []Event{NewRunErrorEvent(message)}
}

// convertDone handles completion updates.
func (c *EventConverter) convertDone(update agent.ResponseUpdate) []Event {
	var events []Event

	// Flush any remaining active messages and tool calls
	events = append(events, c.FlushActiveMessages()...)
	events = append(events, c.FlushActiveToolCalls()...)

	// Add run finished event
	events = append(events, c.RunFinished())

	return events
}

// generateMessageID creates a unique message ID.
func (c *EventConverter) generateMessageID() string {
	c.messageCounter++
	return uuid.New().String()
}
