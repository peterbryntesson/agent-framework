// Copyright (c) Microsoft. All rights reserved.

package executors

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/microsoft/agent-framework-go/agent"
	"github.com/microsoft/agent-framework-go/chat"
	"github.com/microsoft/agent-framework-go/workflow"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newAggTextMessage creates a message with text content for testing.
func newAggTextMessage(role chat.Role, text string) agent.Message {
	return agent.Message{
		Role:     role,
		Contents: []chat.Content{chat.NewTextContent(text)},
	}
}

func TestAggregatingExecutor(t *testing.T) {
	t.Run("returns correct ID", func(t *testing.T) {
		// Arrange
		executor := NewAggregatingExecutor("aggregator", []string{"a", "b"}, func(messages []agent.Message) ([]agent.Message, error) {
			return nil, nil
		})

		// Act
		id := executor.ID()

		// Assert
		assert.Equal(t, "aggregator", id)
	})

	t.Run("returns expected sources", func(t *testing.T) {
		// Arrange
		sources := []string{"source-a", "source-b", "source-c"}
		executor := NewAggregatingExecutor("aggregator", sources, func(messages []agent.Message) ([]agent.Message, error) {
			return nil, nil
		})

		// Act
		result := executor.ExpectedSources()

		// Assert
		assert.Equal(t, sources, result)
		// Verify it's a copy
		result[0] = "modified"
		assert.Equal(t, "source-a", executor.ExpectedSources()[0])
	})

	t.Run("does not aggregate until all sources send", func(t *testing.T) {
		// Arrange
		aggregatorCalled := false
		executor := NewAggregatingExecutor("aggregator", []string{"source-a", "source-b"}, func(messages []agent.Message) ([]agent.Message, error) {
			aggregatorCalled = true
			return nil, nil
		})

		// Only source-a sends
		inputMessages := []workflow.WorkflowMessage{
			{From: "source-a", To: "aggregator", Content: newAggTextMessage(chat.RoleAssistant, "from-a")},
		}
		wCtx := newAggregatingTestContext("aggregator", inputMessages)

		// Act
		err := executor.Execute(context.Background(), wCtx)

		// Assert
		require.NoError(t, err)
		assert.False(t, aggregatorCalled, "aggregator should not be called until all sources send")
		assert.Empty(t, wCtx.Outbox())
	})

	t.Run("aggregates when all sources send", func(t *testing.T) {
		// Arrange
		var receivedMessages []agent.Message
		executor := NewAggregatingExecutor("aggregator", []string{"source-a", "source-b"}, func(messages []agent.Message) ([]agent.Message, error) {
			receivedMessages = messages
			return []agent.Message{
				newAggTextMessage(chat.RoleAssistant, "aggregated"),
			}, nil
		})

		// First superstep: source-a sends
		msg1 := []workflow.WorkflowMessage{
			{From: "source-a", To: "aggregator", Content: newAggTextMessage(chat.RoleAssistant, "from-a")},
		}
		wCtx1 := newAggregatingTestContext("aggregator", msg1)
		err := executor.Execute(context.Background(), wCtx1)
		require.NoError(t, err)
		assert.Empty(t, wCtx1.Outbox())

		// Second superstep: source-b sends
		msg2 := []workflow.WorkflowMessage{
			{From: "source-b", To: "aggregator", Content: newAggTextMessage(chat.RoleAssistant, "from-b")},
		}
		wCtx2 := newAggregatingTestContext("aggregator", msg2)

		// Act
		err = executor.Execute(context.Background(), wCtx2)

		// Assert
		require.NoError(t, err)
		require.Len(t, receivedMessages, 2)
		assert.Equal(t, "from-a", receivedMessages[0].Text())
		assert.Equal(t, "from-b", receivedMessages[1].Text())

		outbox := wCtx2.Outbox()
		require.Len(t, outbox, 1)
		assert.Equal(t, "aggregated", outbox[0].Content.Text())
	})

	t.Run("aggregates messages in source order", func(t *testing.T) {
		// Arrange
		var receivedMessages []agent.Message
		executor := NewAggregatingExecutor("aggregator", []string{"first", "second", "third"}, func(messages []agent.Message) ([]agent.Message, error) {
			receivedMessages = messages
			return nil, nil
		})

		// Send in reverse order
		messages := []workflow.WorkflowMessage{
			{From: "third", Content: newAggTextMessage(chat.RoleAssistant, "msg-3")},
			{From: "second", Content: newAggTextMessage(chat.RoleAssistant, "msg-2")},
			{From: "first", Content: newAggTextMessage(chat.RoleAssistant, "msg-1")},
		}
		wCtx := newAggregatingTestContext("aggregator", messages)

		// Act
		err := executor.Execute(context.Background(), wCtx)

		// Assert
		require.NoError(t, err)
		require.Len(t, receivedMessages, 3)
		// Should be in source order: first, second, third
		assert.Equal(t, "msg-1", receivedMessages[0].Text())
		assert.Equal(t, "msg-2", receivedMessages[1].Text())
		assert.Equal(t, "msg-3", receivedMessages[2].Text())
	})

	t.Run("handles multiple messages from same source", func(t *testing.T) {
		// Arrange
		var receivedMessages []agent.Message
		executor := NewAggregatingExecutor("aggregator", []string{"source-a", "source-b"}, func(messages []agent.Message) ([]agent.Message, error) {
			receivedMessages = messages
			return nil, nil
		})

		// Both sources send multiple messages
		messages := []workflow.WorkflowMessage{
			{From: "source-a", Content: newAggTextMessage(chat.RoleAssistant, "a-1")},
			{From: "source-a", Content: newAggTextMessage(chat.RoleAssistant, "a-2")},
			{From: "source-b", Content: newAggTextMessage(chat.RoleAssistant, "b-1")},
		}
		wCtx := newAggregatingTestContext("aggregator", messages)

		// Act
		err := executor.Execute(context.Background(), wCtx)

		// Assert
		require.NoError(t, err)
		require.Len(t, receivedMessages, 3)
		// Messages from source-a should come first (in order), then source-b
		assert.Equal(t, "a-1", receivedMessages[0].Text())
		assert.Equal(t, "a-2", receivedMessages[1].Text())
		assert.Equal(t, "b-1", receivedMessages[2].Text())
	})

	t.Run("returns error from aggregator", func(t *testing.T) {
		// Arrange
		expectedErr := errors.New("aggregation error")
		executor := NewAggregatingExecutor("aggregator", []string{"source"}, func(messages []agent.Message) ([]agent.Message, error) {
			return nil, expectedErr
		})

		messages := []workflow.WorkflowMessage{
			{From: "source", Content: newAggTextMessage(chat.RoleAssistant, "test")},
		}
		wCtx := newAggregatingTestContext("aggregator", messages)

		// Act
		err := executor.Execute(context.Background(), wCtx)

		// Assert
		assert.Equal(t, expectedErr, err)
	})

	t.Run("clears collected messages after aggregation", func(t *testing.T) {
		// Arrange
		aggregationCount := 0
		executor := NewAggregatingExecutor("aggregator", []string{"source"}, func(messages []agent.Message) ([]agent.Message, error) {
			aggregationCount++
			return []agent.Message{
				newAggTextMessage(chat.RoleAssistant, "aggregated"),
			}, nil
		})

		// First aggregation
		msg1 := []workflow.WorkflowMessage{
			{From: "source", Content: newAggTextMessage(chat.RoleAssistant, "msg-1")},
		}
		wCtx1 := newAggregatingTestContext("aggregator", msg1)
		err := executor.Execute(context.Background(), wCtx1)
		require.NoError(t, err)
		require.Len(t, wCtx1.Outbox(), 1)

		// Second aggregation - should require new message from source
		wCtx2 := newAggregatingTestContext("aggregator", nil)
		err = executor.Execute(context.Background(), wCtx2)
		require.NoError(t, err)
		assert.Empty(t, wCtx2.Outbox()) // No new messages, should not aggregate

		// Third aggregation with new message
		msg3 := []workflow.WorkflowMessage{
			{From: "source", Content: newAggTextMessage(chat.RoleAssistant, "msg-2")},
		}
		wCtx3 := newAggregatingTestContext("aggregator", msg3)
		err = executor.Execute(context.Background(), wCtx3)
		require.NoError(t, err)
		require.Len(t, wCtx3.Outbox(), 1)

		// Assert
		assert.Equal(t, 2, aggregationCount)
	})

	t.Run("reset clears collected messages", func(t *testing.T) {
		// Arrange
		aggregatorCalled := false
		executor := NewAggregatingExecutor("aggregator", []string{"source-a", "source-b"}, func(messages []agent.Message) ([]agent.Message, error) {
			aggregatorCalled = true
			return nil, nil
		})

		// Collect from source-a
		msg1 := []workflow.WorkflowMessage{
			{From: "source-a", Content: newAggTextMessage(chat.RoleAssistant, "test")},
		}
		wCtx1 := newAggregatingTestContext("aggregator", msg1)
		err := executor.Execute(context.Background(), wCtx1)
		require.NoError(t, err)

		// Reset
		executor.Reset()

		// Now source-b sends, but source-a should be missing
		msg2 := []workflow.WorkflowMessage{
			{From: "source-b", Content: newAggTextMessage(chat.RoleAssistant, "test")},
		}
		wCtx2 := newAggregatingTestContext("aggregator", msg2)
		err = executor.Execute(context.Background(), wCtx2)
		require.NoError(t, err)

		// Assert
		assert.False(t, aggregatorCalled, "aggregator should not be called after reset (missing source-a)")
	})

	t.Run("handles empty sources list", func(t *testing.T) {
		// Arrange
		aggregatorCalled := false
		executor := NewAggregatingExecutor("aggregator", []string{}, func(messages []agent.Message) ([]agent.Message, error) {
			aggregatorCalled = true
			return nil, nil
		})
		wCtx := newAggregatingTestContext("aggregator", nil)

		// Act
		err := executor.Execute(context.Background(), wCtx)

		// Assert
		require.NoError(t, err)
		assert.True(t, aggregatorCalled, "aggregator should be called with empty sources")
	})

	t.Run("handles single source", func(t *testing.T) {
		// Arrange
		var receivedMessages []agent.Message
		executor := NewAggregatingExecutor("aggregator", []string{"only-source"}, func(messages []agent.Message) ([]agent.Message, error) {
			receivedMessages = messages
			return []agent.Message{
				newAggTextMessage(chat.RoleAssistant, "done"),
			}, nil
		})

		messages := []workflow.WorkflowMessage{
			{From: "only-source", Content: newAggTextMessage(chat.RoleAssistant, "single")},
		}
		wCtx := newAggregatingTestContext("aggregator", messages)

		// Act
		err := executor.Execute(context.Background(), wCtx)

		// Assert
		require.NoError(t, err)
		require.Len(t, receivedMessages, 1)
		assert.Equal(t, "single", receivedMessages[0].Text())
		assert.Len(t, wCtx.Outbox(), 1)
	})

	t.Run("thread-safe concurrent access", func(t *testing.T) {
		// Arrange
		executor := NewAggregatingExecutor("aggregator", []string{"source-a", "source-b", "source-c"}, func(messages []agent.Message) ([]agent.Message, error) {
			return []agent.Message{newAggTextMessage(chat.RoleAssistant, "done")}, nil
		})

		var wg sync.WaitGroup
		errors := make(chan error, 3)

		// Act - concurrent sends from different goroutines
		for _, source := range []string{"source-a", "source-b", "source-c"} {
			wg.Add(1)
			go func(src string) {
				defer wg.Done()
				messages := []workflow.WorkflowMessage{
					{From: src, Content: newAggTextMessage(chat.RoleAssistant, "test")},
				}
				wCtx := newAggregatingTestContext("aggregator", messages)
				if err := executor.Execute(context.Background(), wCtx); err != nil {
					errors <- err
				}
			}(source)
		}

		wg.Wait()
		close(errors)

		// Assert
		for err := range errors {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("messages from unknown source are collected but not waited for", func(t *testing.T) {
		// Arrange
		var receivedMessages []agent.Message
		executor := NewAggregatingExecutor("aggregator", []string{"expected-source"}, func(messages []agent.Message) ([]agent.Message, error) {
			receivedMessages = messages
			return nil, nil
		})

		messages := []workflow.WorkflowMessage{
			{From: "unknown-source", Content: newAggTextMessage(chat.RoleAssistant, "unknown")},
			{From: "expected-source", Content: newAggTextMessage(chat.RoleAssistant, "expected")},
		}
		wCtx := newAggregatingTestContext("aggregator", messages)

		// Act
		err := executor.Execute(context.Background(), wCtx)

		// Assert
		require.NoError(t, err)
		// Only messages from expected sources are passed to aggregator
		require.Len(t, receivedMessages, 1)
		assert.Equal(t, "expected", receivedMessages[0].Text())
	})
}

// newAggregatingTestContext creates a WorkflowContext for aggregating executor testing.
func newAggregatingTestContext(executorID string, messages []workflow.WorkflowMessage) *workflow.WorkflowContext {
	state := &sync.Map{}
	return workflow.NewWorkflowContextForTest(context.Background(), executorID, "test-run", 0, messages, state)
}
