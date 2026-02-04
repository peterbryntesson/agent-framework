// Copyright (c) Microsoft. All rights reserved.

package workflow

import (
	"context"
	"sync"
	"testing"

	"github.com/microsoft/agent-framework-go/agent"
	"github.com/microsoft/agent-framework-go/chat"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWorkflowContext_BasicOperations(t *testing.T) {
	t.Run("returns context values", func(t *testing.T) {
		// Arrange
		ctx := context.Background()
		state := &sync.Map{}
		messages := []WorkflowMessage{
			{From: "a", To: "b", Content: agent.NewUserMessage("test"), Superstep: 0},
		}

		// Act
		wCtx := newWorkflowContext(ctx, "executor-1", "run-123", 5, messages, state)

		// Assert
		assert.Equal(t, ctx, wCtx.Context())
		assert.Equal(t, "executor-1", wCtx.ExecutorID())
		assert.Equal(t, "run-123", wCtx.RunID())
		assert.Equal(t, 5, wCtx.Superstep())
		assert.Len(t, wCtx.Messages(), 1)
		assert.Equal(t, "test", getMessageText(wCtx.Messages()[0].Content))
	})

	t.Run("state operations work correctly", func(t *testing.T) {
		// Arrange
		ctx := context.Background()
		state := &sync.Map{}
		wCtx := newWorkflowContext(ctx, "executor-1", "run-123", 0, nil, state)

		// Act
		wCtx.SetState("key1", "value1")
		wCtx.SetState("key2", 42)

		// Assert
		val1, ok1 := wCtx.GetState("key1")
		require.True(t, ok1)
		assert.Equal(t, "value1", val1)

		val2, ok2 := wCtx.GetState("key2")
		require.True(t, ok2)
		assert.Equal(t, 42, val2)

		_, ok3 := wCtx.GetState("missing")
		assert.False(t, ok3)
	})

	t.Run("send queues messages correctly", func(t *testing.T) {
		// Arrange
		ctx := context.Background()
		state := &sync.Map{}
		wCtx := newWorkflowContext(ctx, "executor-1", "run-123", 3, nil, state)

		// Act
		wCtx.Send("executor-2", agent.NewUserMessage("message1"))
		wCtx.Send("executor-3", agent.NewUserMessage("message2"))

		// Assert
		outbox := wCtx.Outbox()
		require.Len(t, outbox, 2)

		assert.Equal(t, "executor-1", outbox[0].From)
		assert.Equal(t, "executor-2", outbox[0].To)
		assert.Equal(t, 3, outbox[0].Superstep)

		assert.Equal(t, "executor-1", outbox[1].From)
		assert.Equal(t, "executor-3", outbox[1].To)
	})

	t.Run("outbox returns copy", func(t *testing.T) {
		// Arrange
		ctx := context.Background()
		state := &sync.Map{}
		wCtx := newWorkflowContext(ctx, "executor-1", "run-123", 0, nil, state)
		wCtx.Send("executor-2", agent.NewUserMessage("test"))

		// Act
		outbox1 := wCtx.Outbox()
		outbox2 := wCtx.Outbox()

		// Assert - modifying one doesn't affect the other
		outbox1[0].To = "modified"
		assert.Equal(t, "executor-2", outbox2[0].To)
	})
}

func TestWorkflowContext_ConcurrentAccess(t *testing.T) {
	t.Run("concurrent sends are safe", func(t *testing.T) {
		// Arrange
		ctx := context.Background()
		state := &sync.Map{}
		wCtx := newWorkflowContext(ctx, "executor-1", "run-123", 0, nil, state)

		// Act
		var wg sync.WaitGroup
		for i := 0; i < 100; i++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()
				wCtx.Send("target", agent.NewUserMessage("msg"))
			}(i)
		}
		wg.Wait()

		// Assert
		assert.Len(t, wCtx.Outbox(), 100)
	})

	t.Run("concurrent state access is safe", func(t *testing.T) {
		// Arrange
		ctx := context.Background()
		state := &sync.Map{}
		wCtx := newWorkflowContext(ctx, "executor-1", "run-123", 0, nil, state)

		// Act
		var wg sync.WaitGroup
		for i := 0; i < 100; i++ {
			wg.Add(2)
			go func(idx int) {
				defer wg.Done()
				wCtx.SetState("key", idx)
			}(i)
			go func() {
				defer wg.Done()
				wCtx.GetState("key")
			}()
		}
		wg.Wait()

		// Assert - no panic means success
		_, ok := wCtx.GetState("key")
		assert.True(t, ok)
	})
}

// getMessageText extracts text from a message for testing
func getMessageText(msg agent.Message) string {
	for _, c := range msg.Contents {
		if tc, ok := c.(*chat.TextContent); ok {
			return tc.Text
		}
	}
	return ""
}
