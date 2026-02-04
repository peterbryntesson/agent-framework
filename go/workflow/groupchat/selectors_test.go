// Copyright (c) Microsoft. All rights reserved.

package groupchat

import (
	"context"
	"encoding/json"
	"errors"
	"math/rand"
	"reflect"
	"sync"
	"testing"

	"github.com/microsoft/agent-framework-go/agent"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockAgentWithResponse extends mockAgent for LLMSelector tests
type mockAgentWithResponse struct {
	id          string
	name        string
	description string
	response    string
	runErr      error
}

func newMockAgentWithResponse(id, name, description, response string) *mockAgentWithResponse {
	return &mockAgentWithResponse{
		id:          id,
		name:        name,
		description: description,
		response:    response,
	}
}

func (m *mockAgentWithResponse) ID() string                      { return m.id }
func (m *mockAgentWithResponse) Name() string                    { return m.name }
func (m *mockAgentWithResponse) Description() string             { return m.description }
func (m *mockAgentWithResponse) Metadata() agent.AIAgentMetadata { return agent.AIAgentMetadata{} }
func (m *mockAgentWithResponse) Run(_ context.Context, _ []agent.Message, _ ...agent.RunOption) (*agent.Response, error) {
	if m.runErr != nil {
		return nil, m.runErr
	}
	return &agent.Response{
		Messages: []agent.Message{agent.NewAssistantMessage(m.response)},
	}, nil
}
func (m *mockAgentWithResponse) RunStream(_ context.Context, _ []agent.Message, _ ...agent.RunOption) (<-chan agent.ResponseUpdate, error) {
	ch := make(chan agent.ResponseUpdate)
	close(ch)
	return ch, nil
}
func (m *mockAgentWithResponse) NewSession(_ context.Context) (agent.Session, error) { return nil, nil }
func (m *mockAgentWithResponse) RestoreSession(_ context.Context, _ json.RawMessage) (agent.Session, error) {
	return nil, nil
}
func (m *mockAgentWithResponse) GetService(_ reflect.Type) interface{} { return nil }

// TestRoundRobinSelector tests the RoundRobinSelector implementation
func TestRoundRobinSelector(t *testing.T) {
	t.Run("creates selector with agents", func(t *testing.T) {
		// Arrange
		agents := []agent.Agent{
			newMockAgent("a1", "Agent 1"),
			newMockAgent("a2", "Agent 2"),
		}

		// Act
		selector, err := NewRoundRobinSelector(agents...)

		// Assert
		require.NoError(t, err)
		assert.NotNil(t, selector)
	})

	t.Run("returns error with no agents", func(t *testing.T) {
		// Act
		selector, err := NewRoundRobinSelector()

		// Assert
		assert.ErrorIs(t, err, ErrNoAgents)
		assert.Nil(t, selector)
	})

	t.Run("selects agents in round-robin order", func(t *testing.T) {
		// Arrange
		agentA := newMockAgent("a1", "Agent A")
		agentB := newMockAgent("a2", "Agent B")
		agentC := newMockAgent("a3", "Agent C")
		selector, err := NewRoundRobinSelector(agentA, agentB, agentC)
		require.NoError(t, err)

		transcript := NewTranscript()
		ctx := context.Background()

		// Act & Assert - cycle through all agents twice
		selected, err := selector.SelectNext(ctx, transcript)
		require.NoError(t, err)
		assert.Equal(t, agentA, selected)

		selected, err = selector.SelectNext(ctx, transcript)
		require.NoError(t, err)
		assert.Equal(t, agentB, selected)

		selected, err = selector.SelectNext(ctx, transcript)
		require.NoError(t, err)
		assert.Equal(t, agentC, selected)

		// Should wrap around
		selected, err = selector.SelectNext(ctx, transcript)
		require.NoError(t, err)
		assert.Equal(t, agentA, selected)

		selected, err = selector.SelectNext(ctx, transcript)
		require.NoError(t, err)
		assert.Equal(t, agentB, selected)
	})

	t.Run("reset restarts sequence", func(t *testing.T) {
		// Arrange
		agentA := newMockAgent("a1", "Agent A")
		agentB := newMockAgent("a2", "Agent B")
		selector, err := NewRoundRobinSelector(agentA, agentB)
		require.NoError(t, err)
		ctx := context.Background()

		// Advance selector
		_, _ = selector.SelectNext(ctx, nil)
		_, _ = selector.SelectNext(ctx, nil)

		// Act
		selector.Reset()
		selected, err := selector.SelectNext(ctx, nil)

		// Assert
		require.NoError(t, err)
		assert.Equal(t, agentA, selected)
	})

	t.Run("thread-safe selection", func(t *testing.T) {
		// Arrange
		agents := make([]agent.Agent, 5)
		for i := 0; i < 5; i++ {
			agents[i] = newMockAgent(string(rune('a'+i)), "Agent")
		}
		selector, err := NewRoundRobinSelector(agents...)
		require.NoError(t, err)
		ctx := context.Background()

		// Act - concurrent selections
		var wg sync.WaitGroup
		selections := make([]agent.Agent, 100)
		for i := 0; i < 100; i++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()
				selected, _ := selector.SelectNext(ctx, nil)
				selections[idx] = selected
			}(i)
		}
		wg.Wait()

		// Assert - all selections should be valid agents
		for _, sel := range selections {
			assert.Contains(t, agents, sel)
		}
	})
}

// TestRandomSelector tests the RandomSelector implementation
func TestRandomSelector(t *testing.T) {
	t.Run("creates selector with agents", func(t *testing.T) {
		// Arrange
		agents := []agent.Agent{
			newMockAgent("a1", "Agent 1"),
			newMockAgent("a2", "Agent 2"),
		}

		// Act
		selector, err := NewRandomSelector(agents)

		// Assert
		require.NoError(t, err)
		assert.NotNil(t, selector)
	})

	t.Run("returns error with no agents", func(t *testing.T) {
		// Act
		selector, err := NewRandomSelector(nil)

		// Assert
		assert.ErrorIs(t, err, ErrNoAgents)
		assert.Nil(t, selector)
	})

	t.Run("returns error with empty slice", func(t *testing.T) {
		// Act
		selector, err := NewRandomSelector([]agent.Agent{})

		// Assert
		assert.ErrorIs(t, err, ErrNoAgents)
		assert.Nil(t, selector)
	})

	t.Run("selects from available agents", func(t *testing.T) {
		// Arrange
		agentA := newMockAgent("a1", "Agent A")
		agentB := newMockAgent("a2", "Agent B")
		agentC := newMockAgent("a3", "Agent C")
		agents := []agent.Agent{agentA, agentB, agentC}
		selector, err := NewRandomSelector(agents)
		require.NoError(t, err)

		ctx := context.Background()
		transcript := NewTranscript()

		// Act - select many times
		for i := 0; i < 20; i++ {
			selected, err := selector.SelectNext(ctx, transcript)

			// Assert
			require.NoError(t, err)
			assert.Contains(t, agents, selected)
		}
	})

	t.Run("uses provided RNG for deterministic behavior", func(t *testing.T) {
		// Arrange
		agentA := newMockAgent("a1", "Agent A")
		agentB := newMockAgent("a2", "Agent B")
		agentC := newMockAgent("a3", "Agent C")
		agents := []agent.Agent{agentA, agentB, agentC}

		// Create two selectors with same seed
		rng1 := rand.New(rand.NewSource(42))
		rng2 := rand.New(rand.NewSource(42))

		selector1, err := NewRandomSelector(agents, WithRNG(rng1))
		require.NoError(t, err)
		selector2, err := NewRandomSelector(agents, WithRNG(rng2))
		require.NoError(t, err)

		ctx := context.Background()

		// Act & Assert - same sequence
		for i := 0; i < 10; i++ {
			s1, _ := selector1.SelectNext(ctx, nil)
			s2, _ := selector2.SelectNext(ctx, nil)
			assert.Equal(t, s1.ID(), s2.ID(), "Selection %d should match", i)
		}
	})

	t.Run("reset is no-op", func(t *testing.T) {
		// Arrange
		agents := []agent.Agent{newMockAgent("a1", "Agent 1")}
		selector, err := NewRandomSelector(agents)
		require.NoError(t, err)

		// Act & Assert - should not panic
		selector.Reset()
	})

	t.Run("thread-safe selection", func(t *testing.T) {
		// Arrange
		agents := make([]agent.Agent, 5)
		for i := 0; i < 5; i++ {
			agents[i] = newMockAgent(string(rune('a'+i)), "Agent")
		}
		selector, err := NewRandomSelector(agents)
		require.NoError(t, err)
		ctx := context.Background()

		// Act - concurrent selections
		var wg sync.WaitGroup
		selections := make([]agent.Agent, 100)
		for i := 0; i < 100; i++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()
				selected, _ := selector.SelectNext(ctx, nil)
				selections[idx] = selected
			}(i)
		}
		wg.Wait()

		// Assert - all selections should be valid agents
		for _, sel := range selections {
			assert.Contains(t, agents, sel)
		}
	})
}

// TestLLMSelector tests the LLMSelector implementation
func TestLLMSelector(t *testing.T) {
	t.Run("creates selector with decision agent and agents", func(t *testing.T) {
		// Arrange
		decisionAgent := newMockAgentWithResponse("decision", "Moderator", "Decision maker", "Agent A")
		agents := []agent.Agent{
			newMockAgent("a1", "Agent A"),
			newMockAgent("a2", "Agent B"),
		}

		// Act
		selector, err := NewLLMSelector(decisionAgent, agents)

		// Assert
		require.NoError(t, err)
		assert.NotNil(t, selector)
	})

	t.Run("returns error with nil decision agent", func(t *testing.T) {
		// Arrange
		agents := []agent.Agent{newMockAgent("a1", "Agent A")}

		// Act
		selector, err := NewLLMSelector(nil, agents)

		// Assert
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "decision agent cannot be nil")
		assert.Nil(t, selector)
	})

	t.Run("returns error with no agents", func(t *testing.T) {
		// Arrange
		decisionAgent := newMockAgentWithResponse("decision", "Moderator", "", "")

		// Act
		selector, err := NewLLMSelector(decisionAgent, nil)

		// Assert
		assert.ErrorIs(t, err, ErrNoAgents)
		assert.Nil(t, selector)
	})

	t.Run("selects agent by name from LLM response", func(t *testing.T) {
		// Arrange
		agentA := newMockAgent("a1", "Agent A")
		agentB := newMockAgent("a2", "Agent B")
		decisionAgent := newMockAgentWithResponse("decision", "Moderator", "", "Agent B")
		agents := []agent.Agent{agentA, agentB}

		selector, err := NewLLMSelector(decisionAgent, agents)
		require.NoError(t, err)

		ctx := context.Background()
		transcript := NewTranscript()

		// Act
		selected, err := selector.SelectNext(ctx, transcript)

		// Assert
		require.NoError(t, err)
		assert.Equal(t, agentB, selected)
	})

	t.Run("selects agent by name case-insensitive", func(t *testing.T) {
		// Arrange
		agentA := newMockAgent("a1", "Agent Alpha")
		decisionAgent := newMockAgentWithResponse("decision", "Moderator", "", "agent alpha")
		agents := []agent.Agent{agentA}

		selector, err := NewLLMSelector(decisionAgent, agents)
		require.NoError(t, err)

		ctx := context.Background()

		// Act
		selected, err := selector.SelectNext(ctx, nil)

		// Assert
		require.NoError(t, err)
		assert.Equal(t, agentA, selected)
	})

	t.Run("selects agent by ID when name not found", func(t *testing.T) {
		// Arrange
		agentA := newMockAgent("agent-alpha-123", "Agent Alpha")
		decisionAgent := newMockAgentWithResponse("decision", "Moderator", "", "agent-alpha-123")
		agents := []agent.Agent{agentA}

		selector, err := NewLLMSelector(decisionAgent, agents)
		require.NoError(t, err)

		ctx := context.Background()

		// Act
		selected, err := selector.SelectNext(ctx, nil)

		// Assert
		require.NoError(t, err)
		assert.Equal(t, agentA, selected)
	})

	t.Run("selects agent by partial name match", func(t *testing.T) {
		// Arrange
		agentA := newMockAgent("a1", "Python Expert")
		decisionAgent := newMockAgentWithResponse("decision", "Moderator", "", "I think Python Expert should speak next")
		agents := []agent.Agent{agentA}

		selector, err := NewLLMSelector(decisionAgent, agents)
		require.NoError(t, err)

		ctx := context.Background()

		// Act
		selected, err := selector.SelectNext(ctx, nil)

		// Assert
		require.NoError(t, err)
		assert.Equal(t, agentA, selected)
	})

	t.Run("returns error when agent not found", func(t *testing.T) {
		// Arrange
		agentA := newMockAgent("a1", "Agent A")
		decisionAgent := newMockAgentWithResponse("decision", "Moderator", "", "Unknown Agent")
		agents := []agent.Agent{agentA}

		selector, err := NewLLMSelector(decisionAgent, agents)
		require.NoError(t, err)

		ctx := context.Background()

		// Act
		selected, err := selector.SelectNext(ctx, nil)

		// Assert
		assert.ErrorIs(t, err, ErrSelectionFailed)
		assert.Nil(t, selected)
	})

	t.Run("returns error when decision agent fails", func(t *testing.T) {
		// Arrange
		decisionAgent := newMockAgentWithResponse("decision", "Moderator", "", "")
		decisionAgent.runErr = errors.New("LLM error")
		agents := []agent.Agent{newMockAgent("a1", "Agent A")}

		selector, err := NewLLMSelector(decisionAgent, agents)
		require.NoError(t, err)

		ctx := context.Background()

		// Act
		selected, err := selector.SelectNext(ctx, nil)

		// Assert
		assert.ErrorIs(t, err, ErrSelectionFailed)
		assert.Nil(t, selected)
	})

	t.Run("returns error for empty response", func(t *testing.T) {
		// Arrange
		decisionAgent := newMockAgentWithResponse("decision", "Moderator", "", "   ")
		agents := []agent.Agent{newMockAgent("a1", "Agent A")}

		selector, err := NewLLMSelector(decisionAgent, agents)
		require.NoError(t, err)

		ctx := context.Background()

		// Act
		selected, err := selector.SelectNext(ctx, nil)

		// Assert
		assert.ErrorIs(t, err, ErrSelectionFailed)
		assert.Nil(t, selected)
	})

	t.Run("uses custom instructions when provided", func(t *testing.T) {
		// Arrange
		decisionAgent := newMockAgentWithResponse("decision", "Moderator", "", "Agent A")
		agents := []agent.Agent{newMockAgent("a1", "Agent A")}
		customInstructions := "Custom selection instructions"

		selector, err := NewLLMSelector(decisionAgent, agents, WithSelectionInstructions(customInstructions))
		require.NoError(t, err)

		// Assert - instructions are stored
		assert.Equal(t, customInstructions, selector.instructions)
	})

	t.Run("includes conversation history in prompt", func(t *testing.T) {
		// Arrange
		decisionAgent := newMockAgentWithResponse("decision", "Moderator", "", "Agent A")
		agentA := newMockAgent("a1", "Agent A")
		agents := []agent.Agent{agentA}

		selector, err := NewLLMSelector(decisionAgent, agents)
		require.NoError(t, err)

		transcript := NewTranscript()
		transcript.AddMessage(agentA, agent.NewAssistantMessage("Hello from Agent A"), 1)

		// Act
		prompt := selector.buildSelectionPrompt(transcript)

		// Assert
		assert.Contains(t, prompt, "Agent A")
		assert.Contains(t, prompt, "Recent conversation")
		assert.Contains(t, prompt, "Hello from Agent A")
	})

	t.Run("limits conversation history to 10 entries", func(t *testing.T) {
		// Arrange
		decisionAgent := newMockAgentWithResponse("decision", "Moderator", "", "Agent A")
		agentA := newMockAgent("a1", "Agent A")
		agents := []agent.Agent{agentA}

		selector, err := NewLLMSelector(decisionAgent, agents)
		require.NoError(t, err)

		// Add 15 entries
		transcript := NewTranscript()
		for i := 0; i < 15; i++ {
			transcript.AddMessage(agentA, agent.NewAssistantMessage("Message "+string(rune('A'+i))), i+1)
		}

		// Act
		prompt := selector.buildSelectionPrompt(transcript)

		// Assert - should only include last 10 messages (F through O, since A=65, A+5=70='F')
		assert.NotContains(t, prompt, "Message A")
		assert.NotContains(t, prompt, "Message E")
		assert.Contains(t, prompt, "Message F") // 6th message (index 5)
	})

	t.Run("truncates long messages in prompt", func(t *testing.T) {
		// Arrange
		decisionAgent := newMockAgentWithResponse("decision", "Moderator", "", "Agent A")
		agentA := newMockAgent("a1", "Agent A")
		agents := []agent.Agent{agentA}

		selector, err := NewLLMSelector(decisionAgent, agents)
		require.NoError(t, err)

		// Add a very long message
		longMessage := ""
		for i := 0; i < 100; i++ {
			longMessage += "This is a very long message. "
		}
		transcript := NewTranscript()
		transcript.AddMessage(agentA, agent.NewAssistantMessage(longMessage), 1)

		// Act
		prompt := selector.buildSelectionPrompt(transcript)

		// Assert - message should be truncated to 200 chars + "..."
		assert.Contains(t, prompt, "...")
		assert.Less(t, len(prompt), len(longMessage))
	})

	t.Run("reset is no-op", func(t *testing.T) {
		// Arrange
		decisionAgent := newMockAgentWithResponse("decision", "Moderator", "", "Agent A")
		agents := []agent.Agent{newMockAgent("a1", "Agent A")}
		selector, err := NewLLMSelector(decisionAgent, agents)
		require.NoError(t, err)

		// Act & Assert - should not panic
		selector.Reset()
	})

	t.Run("thread-safe selection", func(t *testing.T) {
		// Arrange
		agentA := newMockAgent("a1", "Agent A")
		decisionAgent := newMockAgentWithResponse("decision", "Moderator", "", "Agent A")
		agents := []agent.Agent{agentA}

		selector, err := NewLLMSelector(decisionAgent, agents)
		require.NoError(t, err)
		ctx := context.Background()

		// Act - concurrent selections
		var wg sync.WaitGroup
		errs := make([]error, 20)
		for i := 0; i < 20; i++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()
				_, errs[idx] = selector.SelectNext(ctx, nil)
			}(i)
		}
		wg.Wait()

		// Assert - all selections should succeed
		for _, err := range errs {
			assert.NoError(t, err)
		}
	})
}

// TestParseStructuredResponse tests the structured response parsing helper
func TestParseStructuredResponse(t *testing.T) {
	t.Run("parses valid JSON response", func(t *testing.T) {
		// Arrange
		text := `{"selected_agent": "Agent A"}`

		// Act
		name, ok := parseStructuredResponse(text)

		// Assert
		assert.True(t, ok)
		assert.Equal(t, "Agent A", name)
	})

	t.Run("returns false for non-JSON text", func(t *testing.T) {
		// Arrange
		text := "Agent A"

		// Act
		name, ok := parseStructuredResponse(text)

		// Assert
		assert.False(t, ok)
		assert.Empty(t, name)
	})

	t.Run("returns false for invalid JSON", func(t *testing.T) {
		// Arrange
		text := `{"selected_agent": }`

		// Act
		name, ok := parseStructuredResponse(text)

		// Assert
		assert.False(t, ok)
		assert.Empty(t, name)
	})

	t.Run("returns false for empty selected_agent", func(t *testing.T) {
		// Arrange
		text := `{"selected_agent": ""}`

		// Act
		name, ok := parseStructuredResponse(text)

		// Assert
		assert.False(t, ok)
		assert.Empty(t, name)
	})

	t.Run("handles whitespace around JSON", func(t *testing.T) {
		// Arrange
		text := `  {"selected_agent": "Agent B"}  `

		// Act
		name, ok := parseStructuredResponse(text)

		// Assert
		assert.True(t, ok)
		assert.Equal(t, "Agent B", name)
	})
}
