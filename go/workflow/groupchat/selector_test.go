// Copyright (c) Microsoft. All rights reserved.

package groupchat

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"testing"

	"github.com/microsoft/agent-framework-go/agent"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockAgent is a test implementation of agent.Agent
type mockAgent struct {
	id          string
	name        string
	description string
}

func newMockAgent(id, name string) *mockAgent {
	return &mockAgent{id: id, name: name}
}

func (m *mockAgent) ID() string                      { return m.id }
func (m *mockAgent) Name() string                    { return m.name }
func (m *mockAgent) Description() string             { return m.description }
func (m *mockAgent) Metadata() agent.AIAgentMetadata { return agent.AIAgentMetadata{} }
func (m *mockAgent) Run(_ context.Context, _ []agent.Message, _ ...agent.RunOption) (*agent.Response, error) {
	return &agent.Response{}, nil
}
func (m *mockAgent) RunStream(_ context.Context, _ []agent.Message, _ ...agent.RunOption) (<-chan agent.ResponseUpdate, error) {
	ch := make(chan agent.ResponseUpdate)
	close(ch)
	return ch, nil
}
func (m *mockAgent) NewSession(_ context.Context) (agent.Session, error) { return nil, nil }
func (m *mockAgent) RestoreSession(_ context.Context, _ json.RawMessage) (agent.Session, error) {
	return nil, nil
}
func (m *mockAgent) GetService(_ reflect.Type) interface{} { return nil }

func TestSelectorFunc(t *testing.T) {
	t.Run("selects agent from function", func(t *testing.T) {
		// Arrange
		agentA := newMockAgent("agent-a", "Agent A")
		selector := SelectorFunc(func(ctx context.Context, transcript *Transcript) (agent.Agent, error) {
			return agentA, nil
		})

		// Act
		selected, err := selector.SelectNext(context.Background(), NewTranscript())

		// Assert
		require.NoError(t, err)
		assert.Equal(t, agentA, selected)
	})

	t.Run("returns error from function", func(t *testing.T) {
		// Arrange
		expectedErr := errors.New("selection failed")
		selector := SelectorFunc(func(ctx context.Context, transcript *Transcript) (agent.Agent, error) {
			return nil, expectedErr
		})

		// Act
		_, err := selector.SelectNext(context.Background(), NewTranscript())

		// Assert
		assert.Equal(t, expectedErr, err)
	})

	t.Run("reset is no-op", func(t *testing.T) {
		// Arrange
		selector := SelectorFunc(func(ctx context.Context, transcript *Transcript) (agent.Agent, error) {
			return nil, nil
		})

		// Act & Assert - should not panic
		selector.Reset()
	})
}

func TestMaxTurnsCondition(t *testing.T) {
	t.Run("returns false before max turns", func(t *testing.T) {
		// Arrange
		condition := MaxTurnsCondition(10)

		// Act
		result := condition(context.Background(), NewTranscript(), 5)

		// Assert
		assert.False(t, result)
	})

	t.Run("returns true at max turns", func(t *testing.T) {
		// Arrange
		condition := MaxTurnsCondition(10)

		// Act
		result := condition(context.Background(), NewTranscript(), 10)

		// Assert
		assert.True(t, result)
	})

	t.Run("returns true after max turns", func(t *testing.T) {
		// Arrange
		condition := MaxTurnsCondition(10)

		// Act
		result := condition(context.Background(), NewTranscript(), 15)

		// Assert
		assert.True(t, result)
	})
}

func TestKeywordCondition(t *testing.T) {
	t.Run("returns false for empty transcript", func(t *testing.T) {
		// Arrange
		condition := KeywordCondition("DONE")

		// Act
		result := condition(context.Background(), NewTranscript(), 1)

		// Assert
		assert.False(t, result)
	})

	t.Run("returns true when keyword found", func(t *testing.T) {
		// Arrange
		condition := KeywordCondition("DONE")
		transcript := NewTranscript()
		transcript.AddEntry(TranscriptEntry{
			Speaker: "agent-1",
			Message: agent.NewAssistantMessage("The task is DONE now."),
		})

		// Act
		result := condition(context.Background(), transcript, 1)

		// Assert
		assert.True(t, result)
	})

	t.Run("returns false when keyword not found", func(t *testing.T) {
		// Arrange
		condition := KeywordCondition("DONE")
		transcript := NewTranscript()
		transcript.AddEntry(TranscriptEntry{
			Speaker: "agent-1",
			Message: agent.NewAssistantMessage("Still working on it."),
		})

		// Act
		result := condition(context.Background(), transcript, 1)

		// Assert
		assert.False(t, result)
	})

	t.Run("case insensitive matching", func(t *testing.T) {
		// Arrange
		condition := KeywordCondition("done")
		transcript := NewTranscript()
		transcript.AddEntry(TranscriptEntry{
			Speaker: "agent-1",
			Message: agent.NewAssistantMessage("The task is DONE now."),
		})

		// Act
		result := condition(context.Background(), transcript, 1)

		// Assert
		assert.True(t, result)
	})
}

func TestCombineConditions(t *testing.T) {
	t.Run("returns true when any condition is true", func(t *testing.T) {
		// Arrange
		condition := CombineConditions(
			MaxTurnsCondition(100),
			MaxTurnsCondition(5),
		)

		// Act
		result := condition(context.Background(), NewTranscript(), 10)

		// Assert
		assert.True(t, result)
	})

	t.Run("returns false when all conditions are false", func(t *testing.T) {
		// Arrange
		condition := CombineConditions(
			MaxTurnsCondition(100),
			MaxTurnsCondition(50),
		)

		// Act
		result := condition(context.Background(), NewTranscript(), 10)

		// Assert
		assert.False(t, result)
	})

	t.Run("empty conditions returns false", func(t *testing.T) {
		// Arrange
		condition := CombineConditions()

		// Act
		result := condition(context.Background(), NewTranscript(), 10)

		// Assert
		assert.False(t, result)
	})
}

func TestAllConditions(t *testing.T) {
	t.Run("returns true when all conditions are true", func(t *testing.T) {
		// Arrange
		condition := AllConditions(
			MaxTurnsCondition(5),
			MaxTurnsCondition(8),
		)

		// Act
		result := condition(context.Background(), NewTranscript(), 10)

		// Assert
		assert.True(t, result)
	})

	t.Run("returns false when any condition is false", func(t *testing.T) {
		// Arrange
		condition := AllConditions(
			MaxTurnsCondition(5),
			MaxTurnsCondition(100),
		)

		// Act
		result := condition(context.Background(), NewTranscript(), 10)

		// Assert
		assert.False(t, result)
	})

	t.Run("empty conditions returns false", func(t *testing.T) {
		// Arrange
		condition := AllConditions()

		// Act
		result := condition(context.Background(), NewTranscript(), 10)

		// Assert
		assert.False(t, result)
	})
}

func TestContainsKeyword(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		keyword  string
		expected bool
	}{
		{"empty keyword", "hello world", "", false},
		{"empty text", "", "hello", false},
		{"keyword found", "hello world", "world", true},
		{"keyword not found", "hello world", "foo", false},
		{"keyword at start", "hello world", "hello", true},
		{"keyword at end", "hello world", "world", true},
		{"case insensitive upper", "hello WORLD", "world", true},
		{"case insensitive lower", "HELLO world", "hello", true},
		{"keyword longer than text", "hi", "hello", false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := containsKeyword(tc.text, tc.keyword)
			assert.Equal(t, tc.expected, result)
		})
	}
}
