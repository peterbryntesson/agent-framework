// Copyright (c) Microsoft. All rights reserved.

package agui

import "encoding/json"

// EventType defines the type of AG-UI event.
type EventType string

// Event type constants matching the AG-UI protocol specification.
const (
	// EventTypeRunStarted indicates the start of an agent run.
	EventTypeRunStarted EventType = "RUN_STARTED"

	// EventTypeRunFinished indicates successful completion of a run.
	EventTypeRunFinished EventType = "RUN_FINISHED"

	// EventTypeRunError indicates an error occurred during the run.
	EventTypeRunError EventType = "RUN_ERROR"

	// EventTypeTextMessageStart marks the beginning of a text message.
	EventTypeTextMessageStart EventType = "TEXT_MESSAGE_START"

	// EventTypeTextMessageContent contains incremental text content.
	EventTypeTextMessageContent EventType = "TEXT_MESSAGE_CONTENT"

	// EventTypeTextMessageEnd marks the end of a text message.
	EventTypeTextMessageEnd EventType = "TEXT_MESSAGE_END"

	// EventTypeToolCallStart marks the beginning of a tool call.
	EventTypeToolCallStart EventType = "TOOL_CALL_START"

	// EventTypeToolCallArgs contains incremental tool call arguments.
	EventTypeToolCallArgs EventType = "TOOL_CALL_ARGS"

	// EventTypeToolCallEnd marks the end of a tool call.
	EventTypeToolCallEnd EventType = "TOOL_CALL_END"

	// EventTypeToolCallResult contains the result of a tool call.
	EventTypeToolCallResult EventType = "TOOL_CALL_RESULT"

	// EventTypeStateSnapshot contains a complete state snapshot.
	EventTypeStateSnapshot EventType = "STATE_SNAPSHOT"

	// EventTypeStateDelta contains an incremental state update.
	EventTypeStateDelta EventType = "STATE_DELTA"
)

// Event is the interface implemented by all AG-UI events.
// Each event type has a Type() method that returns its EventType.
type Event interface {
	// Type returns the event type identifier.
	Type() EventType

	// sealed prevents external implementations.
	sealed()
}

// eventBase provides the common implementation for all events.
type eventBase struct {
	eventType EventType
}

func (e eventBase) Type() EventType { return e.eventType }
func (eventBase) sealed()           {}

// RunStartedEvent indicates the start of an agent run.
type RunStartedEvent struct {
	eventBase

	// ThreadID identifies the conversation thread.
	ThreadID string `json:"threadId"`

	// RunID identifies this specific run within the thread.
	RunID string `json:"runId"`
}

// NewRunStartedEvent creates a new RunStartedEvent.
func NewRunStartedEvent(threadID, runID string) *RunStartedEvent {
	return &RunStartedEvent{
		eventBase: eventBase{eventType: EventTypeRunStarted},
		ThreadID:  threadID,
		RunID:     runID,
	}
}

// MarshalJSON implements json.Marshaler.
func (e *RunStartedEvent) MarshalJSON() ([]byte, error) {
	type alias RunStartedEvent
	return json.Marshal(&struct {
		Type EventType `json:"type"`
		*alias
	}{
		Type:  e.eventType,
		alias: (*alias)(e),
	})
}

// RunFinishedEvent indicates successful completion of a run.
type RunFinishedEvent struct {
	eventBase

	// ThreadID identifies the conversation thread.
	ThreadID string `json:"threadId"`

	// RunID identifies this specific run within the thread.
	RunID string `json:"runId"`

	// Result contains optional result data as raw JSON.
	Result json.RawMessage `json:"result,omitempty"`
}

// NewRunFinishedEvent creates a new RunFinishedEvent.
func NewRunFinishedEvent(threadID, runID string) *RunFinishedEvent {
	return &RunFinishedEvent{
		eventBase: eventBase{eventType: EventTypeRunFinished},
		ThreadID:  threadID,
		RunID:     runID,
	}
}

// WithResult sets the result data.
func (e *RunFinishedEvent) WithResult(result interface{}) *RunFinishedEvent {
	if result != nil {
		if data, err := json.Marshal(result); err == nil {
			e.Result = data
		}
	}
	return e
}

// MarshalJSON implements json.Marshaler.
func (e *RunFinishedEvent) MarshalJSON() ([]byte, error) {
	type alias RunFinishedEvent
	return json.Marshal(&struct {
		Type EventType `json:"type"`
		*alias
	}{
		Type:  e.eventType,
		alias: (*alias)(e),
	})
}

// RunErrorEvent indicates an error occurred during the run.
type RunErrorEvent struct {
	eventBase

	// Message is the error message.
	Message string `json:"message"`

	// Code is an optional error code.
	Code string `json:"code,omitempty"`
}

// NewRunErrorEvent creates a new RunErrorEvent.
func NewRunErrorEvent(message string) *RunErrorEvent {
	return &RunErrorEvent{
		eventBase: eventBase{eventType: EventTypeRunError},
		Message:   message,
	}
}

// WithCode sets the error code.
func (e *RunErrorEvent) WithCode(code string) *RunErrorEvent {
	e.Code = code
	return e
}

// MarshalJSON implements json.Marshaler.
func (e *RunErrorEvent) MarshalJSON() ([]byte, error) {
	type alias RunErrorEvent
	return json.Marshal(&struct {
		Type EventType `json:"type"`
		*alias
	}{
		Type:  e.eventType,
		alias: (*alias)(e),
	})
}

// TextMessageStartEvent marks the beginning of a text message.
type TextMessageStartEvent struct {
	eventBase

	// MessageID identifies the message.
	MessageID string `json:"messageId"`

	// Role indicates the message author (e.g., "assistant").
	Role string `json:"role"`
}

// NewTextMessageStartEvent creates a new TextMessageStartEvent.
func NewTextMessageStartEvent(messageID, role string) *TextMessageStartEvent {
	return &TextMessageStartEvent{
		eventBase: eventBase{eventType: EventTypeTextMessageStart},
		MessageID: messageID,
		Role:      role,
	}
}

// MarshalJSON implements json.Marshaler.
func (e *TextMessageStartEvent) MarshalJSON() ([]byte, error) {
	type alias TextMessageStartEvent
	return json.Marshal(&struct {
		Type EventType `json:"type"`
		*alias
	}{
		Type:  e.eventType,
		alias: (*alias)(e),
	})
}

// TextMessageContentEvent contains incremental text content.
type TextMessageContentEvent struct {
	eventBase

	// MessageID identifies the message this content belongs to.
	MessageID string `json:"messageId"`

	// Delta is the incremental text content.
	Delta string `json:"delta"`
}

// NewTextMessageContentEvent creates a new TextMessageContentEvent.
func NewTextMessageContentEvent(messageID, delta string) *TextMessageContentEvent {
	return &TextMessageContentEvent{
		eventBase: eventBase{eventType: EventTypeTextMessageContent},
		MessageID: messageID,
		Delta:     delta,
	}
}

// MarshalJSON implements json.Marshaler.
func (e *TextMessageContentEvent) MarshalJSON() ([]byte, error) {
	type alias TextMessageContentEvent
	return json.Marshal(&struct {
		Type EventType `json:"type"`
		*alias
	}{
		Type:  e.eventType,
		alias: (*alias)(e),
	})
}

// TextMessageEndEvent marks the end of a text message.
type TextMessageEndEvent struct {
	eventBase

	// MessageID identifies the message.
	MessageID string `json:"messageId"`
}

// NewTextMessageEndEvent creates a new TextMessageEndEvent.
func NewTextMessageEndEvent(messageID string) *TextMessageEndEvent {
	return &TextMessageEndEvent{
		eventBase: eventBase{eventType: EventTypeTextMessageEnd},
		MessageID: messageID,
	}
}

// MarshalJSON implements json.Marshaler.
func (e *TextMessageEndEvent) MarshalJSON() ([]byte, error) {
	type alias TextMessageEndEvent
	return json.Marshal(&struct {
		Type EventType `json:"type"`
		*alias
	}{
		Type:  e.eventType,
		alias: (*alias)(e),
	})
}

// ToolCallStartEvent marks the beginning of a tool call.
type ToolCallStartEvent struct {
	eventBase

	// ToolCallID identifies the tool call.
	ToolCallID string `json:"toolCallId"`

	// ToolCallName is the name of the tool being called.
	ToolCallName string `json:"toolCallName"`

	// ParentMessageID optionally links to a parent message.
	ParentMessageID string `json:"parentMessageId,omitempty"`
}

// NewToolCallStartEvent creates a new ToolCallStartEvent.
func NewToolCallStartEvent(toolCallID, toolCallName string) *ToolCallStartEvent {
	return &ToolCallStartEvent{
		eventBase:    eventBase{eventType: EventTypeToolCallStart},
		ToolCallID:   toolCallID,
		ToolCallName: toolCallName,
	}
}

// WithParentMessageID sets the parent message ID.
func (e *ToolCallStartEvent) WithParentMessageID(parentMessageID string) *ToolCallStartEvent {
	e.ParentMessageID = parentMessageID
	return e
}

// MarshalJSON implements json.Marshaler.
func (e *ToolCallStartEvent) MarshalJSON() ([]byte, error) {
	type alias ToolCallStartEvent
	return json.Marshal(&struct {
		Type EventType `json:"type"`
		*alias
	}{
		Type:  e.eventType,
		alias: (*alias)(e),
	})
}

// ToolCallArgsEvent contains incremental tool call arguments.
type ToolCallArgsEvent struct {
	eventBase

	// ToolCallID identifies the tool call.
	ToolCallID string `json:"toolCallId"`

	// Delta is the incremental arguments content.
	Delta string `json:"delta"`
}

// NewToolCallArgsEvent creates a new ToolCallArgsEvent.
func NewToolCallArgsEvent(toolCallID, delta string) *ToolCallArgsEvent {
	return &ToolCallArgsEvent{
		eventBase:  eventBase{eventType: EventTypeToolCallArgs},
		ToolCallID: toolCallID,
		Delta:      delta,
	}
}

// MarshalJSON implements json.Marshaler.
func (e *ToolCallArgsEvent) MarshalJSON() ([]byte, error) {
	type alias ToolCallArgsEvent
	return json.Marshal(&struct {
		Type EventType `json:"type"`
		*alias
	}{
		Type:  e.eventType,
		alias: (*alias)(e),
	})
}

// ToolCallEndEvent marks the end of a tool call.
type ToolCallEndEvent struct {
	eventBase

	// ToolCallID identifies the tool call.
	ToolCallID string `json:"toolCallId"`
}

// NewToolCallEndEvent creates a new ToolCallEndEvent.
func NewToolCallEndEvent(toolCallID string) *ToolCallEndEvent {
	return &ToolCallEndEvent{
		eventBase:  eventBase{eventType: EventTypeToolCallEnd},
		ToolCallID: toolCallID,
	}
}

// MarshalJSON implements json.Marshaler.
func (e *ToolCallEndEvent) MarshalJSON() ([]byte, error) {
	type alias ToolCallEndEvent
	return json.Marshal(&struct {
		Type EventType `json:"type"`
		*alias
	}{
		Type:  e.eventType,
		alias: (*alias)(e),
	})
}

// ToolCallResultEvent contains the result of a tool call.
type ToolCallResultEvent struct {
	eventBase

	// ToolCallID identifies the tool call.
	ToolCallID string `json:"toolCallId"`

	// Content is the tool result content.
	Content string `json:"content"`

	// MessageID optionally identifies the result message.
	MessageID string `json:"messageId,omitempty"`

	// Role optionally specifies the result role.
	Role string `json:"role,omitempty"`
}

// NewToolCallResultEvent creates a new ToolCallResultEvent.
func NewToolCallResultEvent(toolCallID, content string) *ToolCallResultEvent {
	return &ToolCallResultEvent{
		eventBase:  eventBase{eventType: EventTypeToolCallResult},
		ToolCallID: toolCallID,
		Content:    content,
	}
}

// WithMessageID sets the message ID.
func (e *ToolCallResultEvent) WithMessageID(messageID string) *ToolCallResultEvent {
	e.MessageID = messageID
	return e
}

// WithRole sets the role.
func (e *ToolCallResultEvent) WithRole(role string) *ToolCallResultEvent {
	e.Role = role
	return e
}

// MarshalJSON implements json.Marshaler.
func (e *ToolCallResultEvent) MarshalJSON() ([]byte, error) {
	type alias ToolCallResultEvent
	return json.Marshal(&struct {
		Type EventType `json:"type"`
		*alias
	}{
		Type:  e.eventType,
		alias: (*alias)(e),
	})
}

// StateSnapshotEvent contains a complete state snapshot.
type StateSnapshotEvent struct {
	eventBase

	// Snapshot contains the full state as raw JSON.
	Snapshot json.RawMessage `json:"snapshot,omitempty"`
}

// NewStateSnapshotEvent creates a new StateSnapshotEvent.
func NewStateSnapshotEvent(snapshot interface{}) *StateSnapshotEvent {
	event := &StateSnapshotEvent{
		eventBase: eventBase{eventType: EventTypeStateSnapshot},
	}
	if snapshot != nil {
		if data, err := json.Marshal(snapshot); err == nil {
			event.Snapshot = data
		}
	}
	return event
}

// MarshalJSON implements json.Marshaler.
func (e *StateSnapshotEvent) MarshalJSON() ([]byte, error) {
	type alias StateSnapshotEvent
	return json.Marshal(&struct {
		Type EventType `json:"type"`
		*alias
	}{
		Type:  e.eventType,
		alias: (*alias)(e),
	})
}

// StateDeltaEvent contains an incremental state update.
type StateDeltaEvent struct {
	eventBase

	// Delta contains the state delta as raw JSON.
	Delta json.RawMessage `json:"delta,omitempty"`
}

// NewStateDeltaEvent creates a new StateDeltaEvent.
func NewStateDeltaEvent(delta interface{}) *StateDeltaEvent {
	event := &StateDeltaEvent{
		eventBase: eventBase{eventType: EventTypeStateDelta},
	}
	if delta != nil {
		if data, err := json.Marshal(delta); err == nil {
			event.Delta = data
		}
	}
	return event
}

// MarshalJSON implements json.Marshaler.
func (e *StateDeltaEvent) MarshalJSON() ([]byte, error) {
	type alias StateDeltaEvent
	return json.Marshal(&struct {
		Type EventType `json:"type"`
		*alias
	}{
		Type:  e.eventType,
		alias: (*alias)(e),
	})
}

// Verify all event types implement Event interface.
var (
	_ Event = (*RunStartedEvent)(nil)
	_ Event = (*RunFinishedEvent)(nil)
	_ Event = (*RunErrorEvent)(nil)
	_ Event = (*TextMessageStartEvent)(nil)
	_ Event = (*TextMessageContentEvent)(nil)
	_ Event = (*TextMessageEndEvent)(nil)
	_ Event = (*ToolCallStartEvent)(nil)
	_ Event = (*ToolCallArgsEvent)(nil)
	_ Event = (*ToolCallEndEvent)(nil)
	_ Event = (*ToolCallResultEvent)(nil)
	_ Event = (*StateSnapshotEvent)(nil)
	_ Event = (*StateDeltaEvent)(nil)
)
