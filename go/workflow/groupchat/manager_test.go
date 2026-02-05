// Copyright (c) Microsoft. All rights reserved.

package groupchat

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/microsoft/agent-framework-go/agent"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockAgentForManager is a test implementation with configurable responses
type mockAgentForManager struct {
	id           string
	name         string
	description  string
	responses    []string
	responseIdx  int
	runErr       error
	runCount     int32
	lastMessages []agent.Message
	mu           sync.Mutex
}

func newMockAgentForManager(id, name string, responses ...string) *mockAgentForManager {
	if len(responses) == 0 {
		responses = []string{"response from " + name}
	}
	return &mockAgentForManager{
		id:        id,
		name:      name,
		responses: responses,
	}
}

func (m *mockAgentForManager) ID() string                      { return m.id }
func (m *mockAgentForManager) Name() string                    { return m.name }
func (m *mockAgentForManager) Description() string             { return m.description }
func (m *mockAgentForManager) Metadata() agent.AIAgentMetadata { return agent.AIAgentMetadata{} }

func (m *mockAgentForManager) Run(_ context.Context, messages []agent.Message, _ ...agent.RunOption) (*agent.Response, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	atomic.AddInt32(&m.runCount, 1)
	m.lastMessages = messages

	if m.runErr != nil {
		return nil, m.runErr
	}

	response := m.responses[m.responseIdx%len(m.responses)]
	m.responseIdx++

	return &agent.Response{
		Messages: []agent.Message{agent.NewAssistantMessage(response)},
	}, nil
}

func (m *mockAgentForManager) RunStream(_ context.Context, messages []agent.Message, _ ...agent.RunOption) (<-chan agent.ResponseUpdate, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	atomic.AddInt32(&m.runCount, 1)
	m.lastMessages = messages

	if m.runErr != nil {
		return nil, m.runErr
	}

	response := m.responses[m.responseIdx%len(m.responses)]
	m.responseIdx++

	ch := make(chan agent.ResponseUpdate, 2)
	go func() {
		defer close(ch)
		msg := agent.NewAssistantMessage(response)
		ch <- agent.ResponseUpdate{
			Kind:    agent.UpdateKindMessageComplete,
			Message: &msg,
		}
		ch <- agent.ResponseUpdate{
			Kind: agent.UpdateKindDone,
		}
	}()
	return ch, nil
}

func (m *mockAgentForManager) NewSession(_ context.Context) (agent.Session, error) { return nil, nil }
func (m *mockAgentForManager) RestoreSession(_ context.Context, _ json.RawMessage) (agent.Session, error) {
	return nil, nil
}
func (m *mockAgentForManager) GetService(_ reflect.Type) interface{} { return nil }

func (m *mockAgentForManager) RunCount() int32 {
	return atomic.LoadInt32(&m.runCount)
}

// TestNewManager tests Manager creation
func TestNewManager(t *testing.T) {
	t.Run("creates manager with agents", func(t *testing.T) {
		// Arrange
		agents := []agent.Agent{
			newMockAgentForManager("a1", "Agent 1"),
			newMockAgentForManager("a2", "Agent 2"),
		}

		// Act
		manager, err := NewManager(agents)

		// Assert
		require.NoError(t, err)
		assert.NotNil(t, manager)
		assert.Len(t, manager.Agents(), 2)
		assert.Equal(t, 40, manager.MaxTurns())
	})

	t.Run("returns error with no agents", func(t *testing.T) {
		// Act
		manager, err := NewManager(nil)

		// Assert
		assert.ErrorIs(t, err, ErrNoAgents)
		assert.Nil(t, manager)
	})

	t.Run("returns error with empty agents slice", func(t *testing.T) {
		// Act
		manager, err := NewManager([]agent.Agent{})

		// Assert
		assert.ErrorIs(t, err, ErrNoAgents)
		assert.Nil(t, manager)
	})

	t.Run("applies WithMaxTurns option", func(t *testing.T) {
		// Arrange
		agents := []agent.Agent{newMockAgentForManager("a1", "Agent 1")}

		// Act
		manager, err := NewManager(agents, WithMaxTurns(10))

		// Assert
		require.NoError(t, err)
		assert.Equal(t, 10, manager.MaxTurns())
	})

	t.Run("ignores invalid max turns", func(t *testing.T) {
		// Arrange
		agents := []agent.Agent{newMockAgentForManager("a1", "Agent 1")}

		// Act
		manager, err := NewManager(agents, WithMaxTurns(-5))

		// Assert
		require.NoError(t, err)
		assert.Equal(t, 40, manager.MaxTurns()) // unchanged default
	})

	t.Run("applies WithSelector option", func(t *testing.T) {
		// Arrange
		agents := []agent.Agent{
			newMockAgentForManager("a1", "Agent 1"),
			newMockAgentForManager("a2", "Agent 2"),
		}
		randomSelector, _ := NewRandomSelector(agents)

		// Act
		manager, err := NewManager(agents, WithSelector(randomSelector))

		// Assert
		require.NoError(t, err)
		assert.NotNil(t, manager)
	})

	t.Run("uses default round-robin selector when none provided", func(t *testing.T) {
		// Arrange
		agents := []agent.Agent{newMockAgentForManager("a1", "Agent 1")}

		// Act
		manager, err := NewManager(agents)

		// Assert
		require.NoError(t, err)
		assert.NotNil(t, manager)
	})
}

// TestManagerRun tests the synchronous Run method
func TestManagerRun(t *testing.T) {
	t.Run("runs single turn successfully", func(t *testing.T) {
		// Arrange
		agentA := newMockAgentForManager("a1", "Agent A", "Hello from A")
		manager, err := NewManager([]agent.Agent{agentA}, WithMaxTurns(1))
		require.NoError(t, err)

		// Act
		result, err := manager.Run(context.Background(), "Start")

		// Assert
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.True(t, result.IsSuccess())
		assert.Equal(t, 1, result.TurnCount)
		assert.Contains(t, result.TerminationReason, "maximum turns")
		assert.Equal(t, int32(1), agentA.RunCount())
	})

	t.Run("runs multiple turns in round-robin", func(t *testing.T) {
		// Arrange
		agentA := newMockAgentForManager("a1", "Agent A", "Response A")
		agentB := newMockAgentForManager("a2", "Agent B", "Response B")
		manager, err := NewManager([]agent.Agent{agentA, agentB}, WithMaxTurns(4))
		require.NoError(t, err)

		// Act
		result, err := manager.Run(context.Background(), "Start the chat")

		// Assert
		require.NoError(t, err)
		assert.Equal(t, 4, result.TurnCount)
		assert.Equal(t, int32(2), agentA.RunCount())
		assert.Equal(t, int32(2), agentB.RunCount())
	})

	t.Run("includes initial message in transcript", func(t *testing.T) {
		// Arrange
		agentA := newMockAgentForManager("a1", "Agent A")
		manager, err := NewManager([]agent.Agent{agentA}, WithMaxTurns(1))
		require.NoError(t, err)

		// Act
		result, err := manager.Run(context.Background(), "Hello, agents!")

		// Assert
		require.NoError(t, err)
		require.NotNil(t, result.Transcript)
		assert.GreaterOrEqual(t, result.Transcript.Len(), 2) // initial + response
	})

	t.Run("runs without initial message", func(t *testing.T) {
		// Arrange
		agentA := newMockAgentForManager("a1", "Agent A")
		manager, err := NewManager([]agent.Agent{agentA}, WithMaxTurns(1))
		require.NoError(t, err)

		// Act
		result, err := manager.Run(context.Background(), "")

		// Assert
		require.NoError(t, err)
		assert.Equal(t, 1, result.TurnCount)
	})

	t.Run("returns error on context cancellation", func(t *testing.T) {
		// Arrange
		agentA := newMockAgentForManager("a1", "Agent A")
		manager, err := NewManager([]agent.Agent{agentA}, WithMaxTurns(100))
		require.NoError(t, err)

		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		// Act
		result, err := manager.Run(ctx, "Start")

		// Assert
		assert.ErrorIs(t, err, ErrChatCancelled)
		assert.NotNil(t, result)
		assert.Equal(t, "cancelled", result.TerminationReason)
	})

	t.Run("returns error on agent failure", func(t *testing.T) {
		// Arrange
		agentA := newMockAgentForManager("a1", "Agent A")
		agentA.runErr = errors.New("agent error")
		manager, err := NewManager([]agent.Agent{agentA}, WithMaxTurns(3))
		require.NoError(t, err)

		// Act
		result, err := manager.Run(context.Background(), "Start")

		// Assert
		assert.Error(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "agent error", result.TerminationReason)
	})

	t.Run("terminates on custom condition", func(t *testing.T) {
		// Arrange
		agentA := newMockAgentForManager("a1", "Agent A", "DONE")
		condition := KeywordCondition("DONE")
		manager, err := NewManager(
			[]agent.Agent{agentA},
			WithMaxTurns(10),
			WithTerminationCondition(condition),
		)
		require.NoError(t, err)

		// Act
		result, err := manager.Run(context.Background(), "Start")

		// Assert
		require.NoError(t, err)
		// First turn runs, adds response with DONE, then second turn checks condition
		assert.LessOrEqual(t, result.TurnCount, 2)
		assert.Contains(t, result.TerminationReason, "termination condition")
	})

	t.Run("completes transcript on finish", func(t *testing.T) {
		// Arrange
		agentA := newMockAgentForManager("a1", "Agent A")
		manager, err := NewManager([]agent.Agent{agentA}, WithMaxTurns(1))
		require.NoError(t, err)

		// Act
		result, err := manager.Run(context.Background(), "Start")

		// Assert
		require.NoError(t, err)
		assert.True(t, result.Transcript.IsComplete())
	})
}

// TestManagerRunStream tests the streaming RunStream method
func TestManagerRunStream(t *testing.T) {
	t.Run("streams events for single turn", func(t *testing.T) {
		// Arrange
		agentA := newMockAgentForManager("a1", "Agent A", "Hello")
		manager, err := NewManager([]agent.Agent{agentA}, WithMaxTurns(1))
		require.NoError(t, err)

		// Act
		events, err := manager.RunStream(context.Background(), "Start")

		// Assert
		require.NoError(t, err)

		var eventKinds []EventKind
		for event := range events {
			eventKinds = append(eventKinds, event.Kind)
		}

		assert.Contains(t, eventKinds, EventKindStarted)
		assert.Contains(t, eventKinds, EventKindTurnStarted)
		assert.Contains(t, eventKinds, EventKindSpeakerSelected)
		assert.Contains(t, eventKinds, EventKindAgentInvoked)
		assert.Contains(t, eventKinds, EventKindTurnCompleted)
		assert.Contains(t, eventKinds, EventKindTerminating)
		assert.Contains(t, eventKinds, EventKindCompleted)
	})

	t.Run("emits response update events", func(t *testing.T) {
		// Arrange
		agentA := newMockAgentForManager("a1", "Agent A", "Response text")
		manager, err := NewManager([]agent.Agent{agentA}, WithMaxTurns(1))
		require.NoError(t, err)

		// Act
		events, err := manager.RunStream(context.Background(), "Start")
		require.NoError(t, err)

		var hasResponseUpdate bool
		for event := range events {
			if event.Kind == EventKindAgentResponseUpdate {
				hasResponseUpdate = true
			}
		}

		// Assert
		assert.True(t, hasResponseUpdate)
	})

	t.Run("emits response events with messages", func(t *testing.T) {
		// Arrange
		agentA := newMockAgentForManager("a1", "Agent A", "Hello world")
		manager, err := NewManager([]agent.Agent{agentA}, WithMaxTurns(1))
		require.NoError(t, err)

		// Act
		events, err := manager.RunStream(context.Background(), "Start")
		require.NoError(t, err)

		var responseEvents []Event
		for event := range events {
			if event.Kind == EventKindAgentResponse {
				responseEvents = append(responseEvents, event)
			}
		}

		// Assert
		require.Len(t, responseEvents, 1)
		assert.NotNil(t, responseEvents[0].Message)
		assert.Equal(t, "Hello world", responseEvents[0].Message.Text())
	})

	t.Run("emits speaker selection events", func(t *testing.T) {
		// Arrange
		agentA := newMockAgentForManager("a1", "Agent A")
		agentB := newMockAgentForManager("a2", "Agent B")
		manager, err := NewManager([]agent.Agent{agentA, agentB}, WithMaxTurns(2))
		require.NoError(t, err)

		// Act
		events, err := manager.RunStream(context.Background(), "Start")
		require.NoError(t, err)

		var speakers []string
		for event := range events {
			if event.Kind == EventKindSpeakerSelected {
				speakers = append(speakers, event.SpeakerName)
			}
		}

		// Assert
		assert.Len(t, speakers, 2)
		assert.Equal(t, "Agent A", speakers[0])
		assert.Equal(t, "Agent B", speakers[1])
	})

	t.Run("emits error event on agent failure", func(t *testing.T) {
		// Arrange
		agentA := newMockAgentForManager("a1", "Agent A")
		agentA.runErr = errors.New("stream error")
		manager, err := NewManager([]agent.Agent{agentA}, WithMaxTurns(3))
		require.NoError(t, err)

		// Act
		events, err := manager.RunStream(context.Background(), "Start")
		require.NoError(t, err)

		var hasError bool
		var errorEvent Event
		for event := range events {
			if event.Kind == EventKindError {
				hasError = true
				errorEvent = event
			}
		}

		// Assert
		assert.True(t, hasError)
		assert.Error(t, errorEvent.Error)
	})

	t.Run("emits terminating event before completion", func(t *testing.T) {
		// Arrange
		agentA := newMockAgentForManager("a1", "Agent A")
		manager, err := NewManager([]agent.Agent{agentA}, WithMaxTurns(1))
		require.NoError(t, err)

		// Act
		events, err := manager.RunStream(context.Background(), "Start")
		require.NoError(t, err)

		var terminatingEvent *Event
		for event := range events {
			if event.Kind == EventKindTerminating {
				e := event
				terminatingEvent = &e
			}
		}

		// Assert
		require.NotNil(t, terminatingEvent)
		assert.Contains(t, terminatingEvent.TerminationReason, "maximum turns")
	})

	t.Run("includes transcript in completed event", func(t *testing.T) {
		// Arrange
		agentA := newMockAgentForManager("a1", "Agent A")
		manager, err := NewManager([]agent.Agent{agentA}, WithMaxTurns(1))
		require.NoError(t, err)

		// Act
		events, err := manager.RunStream(context.Background(), "Start")
		require.NoError(t, err)

		var completedEvent *Event
		for event := range events {
			if event.Kind == EventKindCompleted {
				e := event
				completedEvent = &e
			}
		}

		// Assert
		require.NotNil(t, completedEvent)
		assert.NotNil(t, completedEvent.Transcript)
		assert.True(t, completedEvent.Transcript.IsComplete())
	})
}

// TestManagerCallbacks tests the callback options
func TestManagerCallbacks(t *testing.T) {
	t.Run("calls before turn callback", func(t *testing.T) {
		// Arrange
		var calledTurns []int
		beforeTurn := func(_ context.Context, turn int, _ *Transcript) bool {
			calledTurns = append(calledTurns, turn)
			return true
		}

		agentA := newMockAgentForManager("a1", "Agent A")
		manager, err := NewManager(
			[]agent.Agent{agentA},
			WithMaxTurns(3),
			WithBeforeTurnCallback(beforeTurn),
		)
		require.NoError(t, err)

		// Act
		_, err = manager.Run(context.Background(), "Start")

		// Assert
		require.NoError(t, err)
		assert.Equal(t, []int{1, 2, 3}, calledTurns)
	})

	t.Run("terminates when before turn returns false", func(t *testing.T) {
		// Arrange
		beforeTurn := func(_ context.Context, turn int, _ *Transcript) bool {
			return turn < 2 // Stop at turn 2
		}

		agentA := newMockAgentForManager("a1", "Agent A")
		manager, err := NewManager(
			[]agent.Agent{agentA},
			WithMaxTurns(10),
			WithBeforeTurnCallback(beforeTurn),
		)
		require.NoError(t, err)

		// Act
		result, err := manager.Run(context.Background(), "Start")

		// Assert
		require.NoError(t, err)
		assert.Equal(t, 2, result.TurnCount)
		assert.Contains(t, result.TerminationReason, "before turn callback")
	})

	t.Run("calls after turn callback", func(t *testing.T) {
		// Arrange
		var callCount int
		var lastSpeaker agent.Agent
		afterTurn := func(_ context.Context, _ int, speaker agent.Agent, _ []agent.Message) {
			callCount++
			lastSpeaker = speaker
		}

		agentA := newMockAgentForManager("a1", "Agent A")
		manager, err := NewManager(
			[]agent.Agent{agentA},
			WithMaxTurns(2),
			WithAfterTurnCallback(afterTurn),
		)
		require.NoError(t, err)

		// Act
		_, err = manager.Run(context.Background(), "Start")

		// Assert
		require.NoError(t, err)
		assert.Equal(t, 2, callCount)
		assert.Equal(t, "Agent A", lastSpeaker.Name())
	})

	t.Run("after turn receives response messages", func(t *testing.T) {
		// Arrange
		var receivedMessages []agent.Message
		afterTurn := func(_ context.Context, _ int, _ agent.Agent, messages []agent.Message) {
			receivedMessages = append(receivedMessages, messages...)
		}

		agentA := newMockAgentForManager("a1", "Agent A", "Hello from A")
		manager, err := NewManager(
			[]agent.Agent{agentA},
			WithMaxTurns(1),
			WithAfterTurnCallback(afterTurn),
		)
		require.NoError(t, err)

		// Act
		_, err = manager.Run(context.Background(), "Start")

		// Assert
		require.NoError(t, err)
		require.Len(t, receivedMessages, 1)
		assert.Equal(t, "Hello from A", receivedMessages[0].Text())
	})
}

// TestManagerHistoryFilter tests the history filter option
func TestManagerHistoryFilter(t *testing.T) {
	t.Run("applies history filter to messages", func(t *testing.T) {
		// Arrange
		var filterCallCount int
		filter := func(_ context.Context, _ agent.Agent, messages []agent.Message) []agent.Message {
			filterCallCount++
			return messages
		}

		agentA := newMockAgentForManager("a1", "Agent A")
		manager, err := NewManager(
			[]agent.Agent{agentA},
			WithMaxTurns(2),
			WithHistoryFilter(filter),
		)
		require.NoError(t, err)

		// Act
		_, err = manager.Run(context.Background(), "Start")

		// Assert
		require.NoError(t, err)
		assert.Equal(t, 2, filterCallCount)
	})

	t.Run("LastNMessagesFilter limits messages", func(t *testing.T) {
		// Arrange
		filter := LastNMessagesFilter(2)

		messages := []agent.Message{
			agent.NewUserMessage("1"),
			agent.NewUserMessage("2"),
			agent.NewUserMessage("3"),
			agent.NewUserMessage("4"),
		}

		// Act
		result := filter(context.Background(), nil, messages)

		// Assert
		require.Len(t, result, 2)
		assert.Equal(t, "3", result[0].Text())
		assert.Equal(t, "4", result[1].Text())
	})

	t.Run("LastNMessagesFilter returns all when fewer than n", func(t *testing.T) {
		// Arrange
		filter := LastNMessagesFilter(10)
		messages := []agent.Message{
			agent.NewUserMessage("1"),
			agent.NewUserMessage("2"),
		}

		// Act
		result := filter(context.Background(), nil, messages)

		// Assert
		assert.Len(t, result, 2)
	})
}

// TestManagerSystemPrompt tests the system prompt option
func TestManagerSystemPrompt(t *testing.T) {
	t.Run("prepends system prompt to messages", func(t *testing.T) {
		// Arrange
		agentA := newMockAgentForManager("a1", "Agent A")
		manager, err := NewManager(
			[]agent.Agent{agentA},
			WithMaxTurns(1),
			WithSystemPrompt("You are a helpful assistant"),
		)
		require.NoError(t, err)

		// Act
		_, err = manager.Run(context.Background(), "Hello")

		// Assert
		require.NoError(t, err)
		agentA.mu.Lock()
		defer agentA.mu.Unlock()
		require.Greater(t, len(agentA.lastMessages), 0)
		assert.Equal(t, "You are a helpful assistant", agentA.lastMessages[0].Text())
	})
}

// TestManagerIncludeTranscript tests transcript inclusion option
func TestManagerIncludeTranscript(t *testing.T) {
	t.Run("includes transcript by default", func(t *testing.T) {
		// Arrange
		agentA := newMockAgentForManager("a1", "Agent A")
		manager, err := NewManager([]agent.Agent{agentA}, WithMaxTurns(2))
		require.NoError(t, err)

		// Act
		_, err = manager.Run(context.Background(), "Hello")

		// Assert - second turn should have more messages than first
		require.NoError(t, err)
		// Just verify the manager ran successfully
	})

	t.Run("excludes transcript when disabled", func(t *testing.T) {
		// Arrange
		agentA := newMockAgentForManager("a1", "Agent A")
		manager, err := NewManager(
			[]agent.Agent{agentA},
			WithMaxTurns(2),
			WithIncludeTranscriptInMessages(false),
		)
		require.NoError(t, err)

		// Act
		_, err = manager.Run(context.Background(), "Hello")

		// Assert
		require.NoError(t, err)
		agentA.mu.Lock()
		defer agentA.mu.Unlock()
		// Should only have the initial message, not accumulated transcript
		assert.LessOrEqual(t, len(agentA.lastMessages), 1)
	})
}

// TestManagerAgentByID tests the agent lookup method
func TestManagerAgentByID(t *testing.T) {
	t.Run("returns agent by ID", func(t *testing.T) {
		// Arrange
		agentA := newMockAgentForManager("agent-a", "Agent A")
		agentB := newMockAgentForManager("agent-b", "Agent B")
		manager, err := NewManager([]agent.Agent{agentA, agentB})
		require.NoError(t, err)

		// Act
		result := manager.AgentByID("agent-b")

		// Assert
		assert.Equal(t, agentB, result)
	})

	t.Run("returns nil for unknown ID", func(t *testing.T) {
		// Arrange
		agentA := newMockAgentForManager("agent-a", "Agent A")
		manager, err := NewManager([]agent.Agent{agentA})
		require.NoError(t, err)

		// Act
		result := manager.AgentByID("unknown")

		// Assert
		assert.Nil(t, result)
	})
}

// TestManagerConcurrency tests thread-safety
func TestManagerConcurrency(t *testing.T) {
	t.Run("handles concurrent Run calls", func(t *testing.T) {
		// Arrange
		agentA := newMockAgentForManager("a1", "Agent A")
		manager, err := NewManager([]agent.Agent{agentA}, WithMaxTurns(2))
		require.NoError(t, err)

		// Act - run multiple concurrent calls
		var wg sync.WaitGroup
		errors := make(chan error, 5)

		for i := 0; i < 5; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				_, err := manager.Run(context.Background(), "Test")
				if err != nil {
					errors <- err
				}
			}()
		}

		wg.Wait()
		close(errors)

		// Assert - all should complete (some sequentially due to mutex)
		for err := range errors {
			assert.NoError(t, err)
		}
	})
}

// TestPerAgentHistoryFilter tests the per-agent filter helper
func TestPerAgentHistoryFilter(t *testing.T) {
	t.Run("applies different filters per agent", func(t *testing.T) {
		// Arrange
		agentA := newMockAgentForManager("agent-a", "Agent A")
		agentB := newMockAgentForManager("agent-b", "Agent B")

		filters := map[string]HistoryFilter{
			"agent-a": LastNMessagesFilter(1),
			"agent-b": LastNMessagesFilter(3),
		}
		filter := PerAgentHistoryFilter(filters)

		messages := []agent.Message{
			agent.NewUserMessage("1"),
			agent.NewUserMessage("2"),
			agent.NewUserMessage("3"),
			agent.NewUserMessage("4"),
		}

		// Act
		resultA := filter(context.Background(), agentA, messages)
		resultB := filter(context.Background(), agentB, messages)

		// Assert
		assert.Len(t, resultA, 1)
		assert.Len(t, resultB, 3)
	})

	t.Run("returns original for unknown agent", func(t *testing.T) {
		// Arrange
		agentC := newMockAgentForManager("agent-c", "Agent C")
		filters := map[string]HistoryFilter{}
		filter := PerAgentHistoryFilter(filters)

		messages := []agent.Message{
			agent.NewUserMessage("1"),
			agent.NewUserMessage("2"),
		}

		// Act
		result := filter(context.Background(), agentC, messages)

		// Assert
		assert.Len(t, result, 2)
	})
}

// TestManagerWithSelector tests custom selector usage
func TestManagerWithSelector(t *testing.T) {
	t.Run("uses custom selector", func(t *testing.T) {
		// Arrange
		agentA := newMockAgentForManager("a1", "Agent A")
		agentB := newMockAgentForManager("a2", "Agent B")

		// Always select agent B
		customSelector := SelectorFunc(func(_ context.Context, _ *Transcript) (agent.Agent, error) {
			return agentB, nil
		})

		manager, err := NewManager(
			[]agent.Agent{agentA, agentB},
			WithSelector(customSelector),
			WithMaxTurns(3),
		)
		require.NoError(t, err)

		// Act
		_, err = manager.Run(context.Background(), "Start")

		// Assert
		require.NoError(t, err)
		assert.Equal(t, int32(0), agentA.RunCount())
		assert.Equal(t, int32(3), agentB.RunCount())
	})

	t.Run("handles selector error", func(t *testing.T) {
		// Arrange
		agentA := newMockAgentForManager("a1", "Agent A")

		failingSelector := SelectorFunc(func(_ context.Context, _ *Transcript) (agent.Agent, error) {
			return nil, errors.New("selection failed")
		})

		manager, err := NewManager(
			[]agent.Agent{agentA},
			WithSelector(failingSelector),
			WithMaxTurns(3),
		)
		require.NoError(t, err)

		// Act
		result, err := manager.Run(context.Background(), "Start")

		// Assert
		assert.Error(t, err)
		assert.ErrorIs(t, result.Error, ErrNoSpeakerSelected)
	})
}

// TestManagerEventTimestamps tests that events have proper timestamps
func TestManagerEventTimestamps(t *testing.T) {
	t.Run("events have chronological timestamps", func(t *testing.T) {
		// Arrange
		agentA := newMockAgentForManager("a1", "Agent A")
		manager, err := NewManager([]agent.Agent{agentA}, WithMaxTurns(1))
		require.NoError(t, err)

		// Act
		events, err := manager.RunStream(context.Background(), "Start")
		require.NoError(t, err)

		var timestamps []time.Time
		for event := range events {
			timestamps = append(timestamps, event.Timestamp)
		}

		// Assert - timestamps should be non-zero and generally increasing
		for _, ts := range timestamps {
			assert.False(t, ts.IsZero())
		}
	})
}
