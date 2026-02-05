// Copyright (c) Microsoft. All rights reserved.

package groupchat

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"
	"time"

	"github.com/microsoft/agent-framework-go/agent"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockTranscriptAgent is a test implementation of agent.Agent for transcript tests
type mockTranscriptAgent struct {
	id   string
	name string
}

func newMockTranscriptAgent(id, name string) *mockTranscriptAgent {
	return &mockTranscriptAgent{id: id, name: name}
}

func (m *mockTranscriptAgent) ID() string                      { return m.id }
func (m *mockTranscriptAgent) Name() string                    { return m.name }
func (m *mockTranscriptAgent) Description() string             { return "" }
func (m *mockTranscriptAgent) Metadata() agent.AIAgentMetadata { return agent.AIAgentMetadata{} }
func (m *mockTranscriptAgent) Run(_ context.Context, _ []agent.Message, _ ...agent.RunOption) (*agent.Response, error) {
	return &agent.Response{}, nil
}
func (m *mockTranscriptAgent) RunStream(_ context.Context, _ []agent.Message, _ ...agent.RunOption) (<-chan agent.ResponseUpdate, error) {
	ch := make(chan agent.ResponseUpdate)
	close(ch)
	return ch, nil
}
func (m *mockTranscriptAgent) NewSession(_ context.Context) (agent.Session, error) { return nil, nil }
func (m *mockTranscriptAgent) RestoreSession(_ context.Context, _ json.RawMessage) (agent.Session, error) {
	return nil, nil
}
func (m *mockTranscriptAgent) GetService(_ reflect.Type) interface{} { return nil }

func TestNewTranscript(t *testing.T) {
	t.Run("creates empty transcript", func(t *testing.T) {
		// Act
		transcript := NewTranscript()

		// Assert
		require.NotNil(t, transcript)
		assert.Empty(t, transcript.Entries)
		assert.False(t, transcript.StartTime.IsZero())
		assert.True(t, transcript.EndTime.IsZero())
	})
}

func TestTranscript_AddEntry(t *testing.T) {
	t.Run("adds entry with auto index", func(t *testing.T) {
		// Arrange
		transcript := NewTranscript()
		entry := TranscriptEntry{
			Speaker:     "agent-1",
			SpeakerName: "Agent One",
			Message:     agent.NewAssistantMessage("Hello"),
			Turn:        1,
		}

		// Act
		transcript.AddEntry(entry)

		// Assert
		require.Len(t, transcript.Entries, 1)
		assert.Equal(t, 0, transcript.Entries[0].Index)
		assert.Equal(t, "agent-1", transcript.Entries[0].Speaker)
	})

	t.Run("increments index for multiple entries", func(t *testing.T) {
		// Arrange
		transcript := NewTranscript()

		// Act
		transcript.AddEntry(TranscriptEntry{Speaker: "agent-1"})
		transcript.AddEntry(TranscriptEntry{Speaker: "agent-2"})
		transcript.AddEntry(TranscriptEntry{Speaker: "agent-3"})

		// Assert
		require.Len(t, transcript.Entries, 3)
		assert.Equal(t, 0, transcript.Entries[0].Index)
		assert.Equal(t, 1, transcript.Entries[1].Index)
		assert.Equal(t, 2, transcript.Entries[2].Index)
	})

	t.Run("sets timestamp if not provided", func(t *testing.T) {
		// Arrange
		transcript := NewTranscript()
		entry := TranscriptEntry{Speaker: "agent-1"}

		// Act
		transcript.AddEntry(entry)

		// Assert
		assert.False(t, transcript.Entries[0].Timestamp.IsZero())
	})

	t.Run("preserves provided timestamp", func(t *testing.T) {
		// Arrange
		transcript := NewTranscript()
		ts := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)
		entry := TranscriptEntry{
			Speaker:   "agent-1",
			Timestamp: ts,
		}

		// Act
		transcript.AddEntry(entry)

		// Assert
		assert.Equal(t, ts, transcript.Entries[0].Timestamp)
	})
}

func TestTranscript_AddMessage(t *testing.T) {
	t.Run("adds message from agent", func(t *testing.T) {
		// Arrange
		transcript := NewTranscript()
		testAgent := newMockTranscriptAgent("agent-1", "Agent One")
		msg := agent.NewAssistantMessage("Hello world")

		// Act
		transcript.AddMessage(testAgent, msg, 1)

		// Assert
		require.Len(t, transcript.Entries, 1)
		assert.Equal(t, "agent-1", transcript.Entries[0].Speaker)
		assert.Equal(t, "Agent One", transcript.Entries[0].SpeakerName)
		assert.Equal(t, 1, transcript.Entries[0].Turn)
	})
}

func TestTranscript_AddUserMessage(t *testing.T) {
	t.Run("adds user message", func(t *testing.T) {
		// Arrange
		transcript := NewTranscript()
		msg := agent.NewUserMessage("What's the weather?")

		// Act
		transcript.AddUserMessage(msg, 0)

		// Assert
		require.Len(t, transcript.Entries, 1)
		assert.Equal(t, "user", transcript.Entries[0].Speaker)
		assert.Equal(t, "User", transcript.Entries[0].SpeakerName)
		assert.Equal(t, 0, transcript.Entries[0].Turn)
	})
}

func TestTranscript_Len(t *testing.T) {
	t.Run("returns zero for empty transcript", func(t *testing.T) {
		// Arrange
		transcript := NewTranscript()

		// Act & Assert
		assert.Equal(t, 0, transcript.Len())
	})

	t.Run("returns correct count", func(t *testing.T) {
		// Arrange
		transcript := NewTranscript()
		transcript.AddEntry(TranscriptEntry{Speaker: "agent-1"})
		transcript.AddEntry(TranscriptEntry{Speaker: "agent-2"})

		// Act & Assert
		assert.Equal(t, 2, transcript.Len())
	})
}

func TestTranscript_LastEntry(t *testing.T) {
	t.Run("returns nil for empty transcript", func(t *testing.T) {
		// Arrange
		transcript := NewTranscript()

		// Act
		entry := transcript.LastEntry()

		// Assert
		assert.Nil(t, entry)
	})

	t.Run("returns last entry", func(t *testing.T) {
		// Arrange
		transcript := NewTranscript()
		transcript.AddEntry(TranscriptEntry{Speaker: "agent-1"})
		transcript.AddEntry(TranscriptEntry{Speaker: "agent-2"})

		// Act
		entry := transcript.LastEntry()

		// Assert
		require.NotNil(t, entry)
		assert.Equal(t, "agent-2", entry.Speaker)
	})
}

func TestTranscript_LastSpeaker(t *testing.T) {
	t.Run("returns empty for empty transcript", func(t *testing.T) {
		// Arrange
		transcript := NewTranscript()

		// Act & Assert
		assert.Empty(t, transcript.LastSpeaker())
	})

	t.Run("returns last speaker ID", func(t *testing.T) {
		// Arrange
		transcript := NewTranscript()
		transcript.AddEntry(TranscriptEntry{Speaker: "agent-1"})
		transcript.AddEntry(TranscriptEntry{Speaker: "agent-2"})

		// Act & Assert
		assert.Equal(t, "agent-2", transcript.LastSpeaker())
	})
}

func TestTranscript_EntriesBySpeaker(t *testing.T) {
	t.Run("returns empty for no matches", func(t *testing.T) {
		// Arrange
		transcript := NewTranscript()
		transcript.AddEntry(TranscriptEntry{Speaker: "agent-1"})
		transcript.AddEntry(TranscriptEntry{Speaker: "agent-2"})

		// Act
		entries := transcript.EntriesBySpeaker("agent-3")

		// Assert
		assert.Empty(t, entries)
	})

	t.Run("returns matching entries", func(t *testing.T) {
		// Arrange
		transcript := NewTranscript()
		transcript.AddEntry(TranscriptEntry{Speaker: "agent-1"})
		transcript.AddEntry(TranscriptEntry{Speaker: "agent-2"})
		transcript.AddEntry(TranscriptEntry{Speaker: "agent-1"})

		// Act
		entries := transcript.EntriesBySpeaker("agent-1")

		// Assert
		require.Len(t, entries, 2)
		assert.Equal(t, 0, entries[0].Index)
		assert.Equal(t, 2, entries[1].Index)
	})
}

func TestTranscript_EntriesSince(t *testing.T) {
	t.Run("returns entries after timestamp", func(t *testing.T) {
		// Arrange
		transcript := NewTranscript()
		t1 := time.Now().Add(-2 * time.Hour)
		t2 := time.Now().Add(-1 * time.Hour)
		t3 := time.Now()

		transcript.AddEntry(TranscriptEntry{Speaker: "agent-1", Timestamp: t1})
		transcript.AddEntry(TranscriptEntry{Speaker: "agent-2", Timestamp: t2})
		transcript.AddEntry(TranscriptEntry{Speaker: "agent-3", Timestamp: t3})

		// Act
		entries := transcript.EntriesSince(t1.Add(30 * time.Minute))

		// Assert
		require.Len(t, entries, 2)
		assert.Equal(t, "agent-2", entries[0].Speaker)
		assert.Equal(t, "agent-3", entries[1].Speaker)
	})
}

func TestTranscript_ToMessages(t *testing.T) {
	t.Run("converts entries to messages", func(t *testing.T) {
		// Arrange
		transcript := NewTranscript()
		transcript.AddEntry(TranscriptEntry{
			Speaker: "agent-1",
			Message: agent.NewAssistantMessage("Hello"),
		})
		transcript.AddEntry(TranscriptEntry{
			Speaker: "user",
			Message: agent.NewUserMessage("Hi there"),
		})

		// Act
		messages := transcript.ToMessages()

		// Assert
		require.Len(t, messages, 2)
		assert.Equal(t, "Hello", messages[0].Text())
		assert.Equal(t, "Hi there", messages[1].Text())
	})

	t.Run("returns empty slice for empty transcript", func(t *testing.T) {
		// Arrange
		transcript := NewTranscript()

		// Act
		messages := transcript.ToMessages()

		// Assert
		assert.Empty(t, messages)
	})
}

func TestTranscript_Clone(t *testing.T) {
	t.Run("creates deep copy", func(t *testing.T) {
		// Arrange
		transcript := NewTranscript()
		transcript.AddEntry(TranscriptEntry{Speaker: "agent-1"})
		transcript.AddEntry(TranscriptEntry{Speaker: "agent-2"})

		// Act
		clone := transcript.Clone()

		// Assert
		require.NotNil(t, clone)
		assert.Equal(t, transcript.Len(), clone.Len())
		assert.Equal(t, transcript.StartTime, clone.StartTime)

		// Verify independence
		transcript.AddEntry(TranscriptEntry{Speaker: "agent-3"})
		assert.NotEqual(t, transcript.Len(), clone.Len())
	})
}

func TestTranscript_Complete(t *testing.T) {
	t.Run("sets end time", func(t *testing.T) {
		// Arrange
		transcript := NewTranscript()
		assert.True(t, transcript.EndTime.IsZero())

		// Act
		transcript.Complete()

		// Assert
		assert.False(t, transcript.EndTime.IsZero())
	})
}

func TestTranscript_IsComplete(t *testing.T) {
	t.Run("returns false for incomplete transcript", func(t *testing.T) {
		// Arrange
		transcript := NewTranscript()

		// Act & Assert
		assert.False(t, transcript.IsComplete())
	})

	t.Run("returns true for complete transcript", func(t *testing.T) {
		// Arrange
		transcript := NewTranscript()
		transcript.Complete()

		// Act & Assert
		assert.True(t, transcript.IsComplete())
	})
}

func TestTranscript_Duration(t *testing.T) {
	t.Run("returns elapsed time for running transcript", func(t *testing.T) {
		// Arrange
		transcript := NewTranscript()
		transcript.StartTime = time.Now().Add(-5 * time.Second)

		// Act
		duration := transcript.Duration()

		// Assert
		assert.GreaterOrEqual(t, duration.Seconds(), float64(5))
	})

	t.Run("returns fixed duration for complete transcript", func(t *testing.T) {
		// Arrange
		transcript := NewTranscript()
		transcript.StartTime = time.Now().Add(-10 * time.Second)
		transcript.EndTime = transcript.StartTime.Add(5 * time.Second)

		// Act
		duration := transcript.Duration()

		// Assert
		assert.Equal(t, 5*time.Second, duration)
	})
}
