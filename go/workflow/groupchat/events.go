// Copyright (c) Microsoft. All rights reserved.

package groupchat

import (
	"time"

	"github.com/microsoft/agent-framework-go/agent"
)

// EventKind indicates the type of group chat event.
type EventKind int

const (
	// EventKindStarted indicates the group chat has started.
	EventKindStarted EventKind = iota

	// EventKindTurnStarted indicates a new turn has begun.
	EventKindTurnStarted

	// EventKindSpeakerSelected indicates the next speaker has been selected.
	EventKindSpeakerSelected

	// EventKindAgentInvoked indicates an agent is being invoked.
	EventKindAgentInvoked

	// EventKindAgentResponse indicates an agent has produced a response.
	EventKindAgentResponse

	// EventKindAgentResponseUpdate indicates a streaming update from an agent.
	EventKindAgentResponseUpdate

	// EventKindTurnCompleted indicates a turn has completed.
	EventKindTurnCompleted

	// EventKindTerminating indicates the chat is about to terminate.
	EventKindTerminating

	// EventKindCompleted indicates the group chat has completed successfully.
	EventKindCompleted

	// EventKindError indicates the group chat encountered an error.
	EventKindError
)

// String returns a string representation of the event kind.
func (k EventKind) String() string {
	switch k {
	case EventKindStarted:
		return "started"
	case EventKindTurnStarted:
		return "turn_started"
	case EventKindSpeakerSelected:
		return "speaker_selected"
	case EventKindAgentInvoked:
		return "agent_invoked"
	case EventKindAgentResponse:
		return "agent_response"
	case EventKindAgentResponseUpdate:
		return "agent_response_update"
	case EventKindTurnCompleted:
		return "turn_completed"
	case EventKindTerminating:
		return "terminating"
	case EventKindCompleted:
		return "completed"
	case EventKindError:
		return "error"
	default:
		return "unknown"
	}
}

// Event represents an event during group chat execution.
type Event struct {
	// Kind is the type of event.
	Kind EventKind

	// Timestamp is when the event occurred.
	Timestamp time.Time

	// Turn is the current turn number (1-based).
	Turn int

	// Speaker is the ID of the speaking agent (if applicable).
	Speaker string

	// SpeakerName is the human-readable name of the speaker.
	SpeakerName string

	// Message contains the response message (for AgentResponse events).
	Message *agent.Message

	// ResponseUpdate contains streaming update data (for AgentResponseUpdate events).
	ResponseUpdate *agent.ResponseUpdate

	// Error contains error information (for Error events).
	Error error

	// TerminationReason explains why the chat is terminating (for Terminating events).
	TerminationReason string

	// Transcript is included in Completed events for final state.
	Transcript *Transcript
}

// NewStartedEvent creates a new group chat started event.
func NewStartedEvent() Event {
	return Event{
		Kind:      EventKindStarted,
		Timestamp: time.Now(),
	}
}

// NewTurnStartedEvent creates a new turn started event.
func NewTurnStartedEvent(turn int) Event {
	return Event{
		Kind:      EventKindTurnStarted,
		Timestamp: time.Now(),
		Turn:      turn,
	}
}

// NewSpeakerSelectedEvent creates a new speaker selected event.
func NewSpeakerSelectedEvent(turn int, speakerID, speakerName string) Event {
	return Event{
		Kind:        EventKindSpeakerSelected,
		Timestamp:   time.Now(),
		Turn:        turn,
		Speaker:     speakerID,
		SpeakerName: speakerName,
	}
}

// NewAgentInvokedEvent creates a new agent invoked event.
func NewAgentInvokedEvent(turn int, speakerID, speakerName string) Event {
	return Event{
		Kind:        EventKindAgentInvoked,
		Timestamp:   time.Now(),
		Turn:        turn,
		Speaker:     speakerID,
		SpeakerName: speakerName,
	}
}

// NewAgentResponseEvent creates a new agent response event.
func NewAgentResponseEvent(turn int, speakerID, speakerName string, message agent.Message) Event {
	return Event{
		Kind:        EventKindAgentResponse,
		Timestamp:   time.Now(),
		Turn:        turn,
		Speaker:     speakerID,
		SpeakerName: speakerName,
		Message:     &message,
	}
}

// NewAgentResponseUpdateEvent creates a new streaming response update event.
func NewAgentResponseUpdateEvent(turn int, speakerID, speakerName string, update agent.ResponseUpdate) Event {
	return Event{
		Kind:           EventKindAgentResponseUpdate,
		Timestamp:      time.Now(),
		Turn:           turn,
		Speaker:        speakerID,
		SpeakerName:    speakerName,
		ResponseUpdate: &update,
	}
}

// NewTurnCompletedEvent creates a new turn completed event.
func NewTurnCompletedEvent(turn int, speakerID, speakerName string) Event {
	return Event{
		Kind:        EventKindTurnCompleted,
		Timestamp:   time.Now(),
		Turn:        turn,
		Speaker:     speakerID,
		SpeakerName: speakerName,
	}
}

// NewTerminatingEvent creates a new terminating event.
func NewTerminatingEvent(turn int, reason string) Event {
	return Event{
		Kind:              EventKindTerminating,
		Timestamp:         time.Now(),
		Turn:              turn,
		TerminationReason: reason,
	}
}

// NewCompletedEvent creates a new group chat completed event.
func NewCompletedEvent(turn int, transcript *Transcript) Event {
	return Event{
		Kind:       EventKindCompleted,
		Timestamp:  time.Now(),
		Turn:       turn,
		Transcript: transcript,
	}
}

// NewErrorEvent creates a new error event.
func NewErrorEvent(turn int, err error) Event {
	return Event{
		Kind:      EventKindError,
		Timestamp: time.Now(),
		Turn:      turn,
		Error:     err,
	}
}

// Result represents the final result of a group chat session.
type Result struct {
	// Transcript contains the complete conversation history.
	Transcript *Transcript

	// TurnCount is the total number of turns executed.
	TurnCount int

	// TerminationReason explains why the chat ended.
	TerminationReason string

	// Error contains any error that occurred (nil if successful).
	Error error
}

// IsSuccess returns true if the group chat completed without error.
func (r *Result) IsSuccess() bool {
	return r.Error == nil
}
