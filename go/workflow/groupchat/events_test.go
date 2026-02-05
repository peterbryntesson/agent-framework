// Copyright (c) Microsoft. All rights reserved.

package groupchat

import (
	"testing"

	"github.com/microsoft/agent-framework-go/agent"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEventKind_String(t *testing.T) {
	tests := []struct {
		kind     EventKind
		expected string
	}{
		{EventKindStarted, "started"},
		{EventKindTurnStarted, "turn_started"},
		{EventKindSpeakerSelected, "speaker_selected"},
		{EventKindAgentInvoked, "agent_invoked"},
		{EventKindAgentResponse, "agent_response"},
		{EventKindAgentResponseUpdate, "agent_response_update"},
		{EventKindTurnCompleted, "turn_completed"},
		{EventKindTerminating, "terminating"},
		{EventKindCompleted, "completed"},
		{EventKindError, "error"},
		{EventKind(999), "unknown"},
	}

	for _, tc := range tests {
		t.Run(tc.expected, func(t *testing.T) {
			assert.Equal(t, tc.expected, tc.kind.String())
		})
	}
}

func TestNewStartedEvent(t *testing.T) {
	t.Run("creates started event", func(t *testing.T) {
		// Act
		event := NewStartedEvent()

		// Assert
		assert.Equal(t, EventKindStarted, event.Kind)
		assert.False(t, event.Timestamp.IsZero())
	})
}

func TestNewTurnStartedEvent(t *testing.T) {
	t.Run("creates turn started event", func(t *testing.T) {
		// Act
		event := NewTurnStartedEvent(3)

		// Assert
		assert.Equal(t, EventKindTurnStarted, event.Kind)
		assert.Equal(t, 3, event.Turn)
		assert.False(t, event.Timestamp.IsZero())
	})
}

func TestNewSpeakerSelectedEvent(t *testing.T) {
	t.Run("creates speaker selected event", func(t *testing.T) {
		// Act
		event := NewSpeakerSelectedEvent(2, "agent-1", "Agent One")

		// Assert
		assert.Equal(t, EventKindSpeakerSelected, event.Kind)
		assert.Equal(t, 2, event.Turn)
		assert.Equal(t, "agent-1", event.Speaker)
		assert.Equal(t, "Agent One", event.SpeakerName)
		assert.False(t, event.Timestamp.IsZero())
	})
}

func TestNewAgentInvokedEvent(t *testing.T) {
	t.Run("creates agent invoked event", func(t *testing.T) {
		// Act
		event := NewAgentInvokedEvent(1, "agent-2", "Agent Two")

		// Assert
		assert.Equal(t, EventKindAgentInvoked, event.Kind)
		assert.Equal(t, 1, event.Turn)
		assert.Equal(t, "agent-2", event.Speaker)
		assert.Equal(t, "Agent Two", event.SpeakerName)
	})
}

func TestNewAgentResponseEvent(t *testing.T) {
	t.Run("creates agent response event", func(t *testing.T) {
		// Arrange
		msg := agent.NewAssistantMessage("Hello world")

		// Act
		event := NewAgentResponseEvent(1, "agent-1", "Agent One", msg)

		// Assert
		assert.Equal(t, EventKindAgentResponse, event.Kind)
		assert.Equal(t, 1, event.Turn)
		assert.Equal(t, "agent-1", event.Speaker)
		require.NotNil(t, event.Message)
		assert.Equal(t, "Hello world", event.Message.Text())
	})
}

func TestNewAgentResponseUpdateEvent(t *testing.T) {
	t.Run("creates response update event", func(t *testing.T) {
		// Arrange
		update := agent.ResponseUpdate{
			Kind: agent.UpdateKindContentDelta,
		}

		// Act
		event := NewAgentResponseUpdateEvent(2, "agent-1", "Agent One", update)

		// Assert
		assert.Equal(t, EventKindAgentResponseUpdate, event.Kind)
		assert.Equal(t, 2, event.Turn)
		require.NotNil(t, event.ResponseUpdate)
		assert.Equal(t, agent.UpdateKindContentDelta, event.ResponseUpdate.Kind)
	})
}

func TestNewTurnCompletedEvent(t *testing.T) {
	t.Run("creates turn completed event", func(t *testing.T) {
		// Act
		event := NewTurnCompletedEvent(5, "agent-3", "Agent Three")

		// Assert
		assert.Equal(t, EventKindTurnCompleted, event.Kind)
		assert.Equal(t, 5, event.Turn)
		assert.Equal(t, "agent-3", event.Speaker)
	})
}

func TestNewTerminatingEvent(t *testing.T) {
	t.Run("creates terminating event", func(t *testing.T) {
		// Act
		event := NewTerminatingEvent(10, "max turns reached")

		// Assert
		assert.Equal(t, EventKindTerminating, event.Kind)
		assert.Equal(t, 10, event.Turn)
		assert.Equal(t, "max turns reached", event.TerminationReason)
	})
}

func TestNewCompletedEvent(t *testing.T) {
	t.Run("creates completed event with transcript", func(t *testing.T) {
		// Arrange
		transcript := NewTranscript()
		transcript.AddEntry(TranscriptEntry{Speaker: "agent-1"})

		// Act
		event := NewCompletedEvent(5, transcript)

		// Assert
		assert.Equal(t, EventKindCompleted, event.Kind)
		assert.Equal(t, 5, event.Turn)
		require.NotNil(t, event.Transcript)
		assert.Equal(t, 1, event.Transcript.Len())
	})
}

func TestNewErrorEvent(t *testing.T) {
	t.Run("creates error event", func(t *testing.T) {
		// Arrange
		err := assert.AnError

		// Act
		event := NewErrorEvent(3, err)

		// Assert
		assert.Equal(t, EventKindError, event.Kind)
		assert.Equal(t, 3, event.Turn)
		assert.Equal(t, err, event.Error)
	})
}

func TestResult_IsSuccess(t *testing.T) {
	t.Run("returns true when no error", func(t *testing.T) {
		// Arrange
		result := &Result{
			Transcript: NewTranscript(),
			TurnCount:  5,
		}

		// Act & Assert
		assert.True(t, result.IsSuccess())
	})

	t.Run("returns false when error", func(t *testing.T) {
		// Arrange
		result := &Result{
			Transcript: NewTranscript(),
			Error:      assert.AnError,
		}

		// Act & Assert
		assert.False(t, result.IsSuccess())
	})
}
