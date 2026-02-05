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

// newFuncTextMessage creates a message with text content for testing.
func newFuncTextMessage(role chat.Role, text string) agent.Message {
	return agent.Message{
		Role:     role,
		Contents: []chat.Content{chat.NewTextContent(text)},
	}
}

func TestFunctionExecutor(t *testing.T) {
	t.Run("returns correct ID", func(t *testing.T) {
		// Arrange
		executor := NewFunctionExecutor("func-executor", func(ctx context.Context, messages []agent.Message) ([]agent.Message, error) {
			return nil, nil
		})

		// Act
		id := executor.ID()

		// Assert
		assert.Equal(t, "func-executor", id)
	})

	t.Run("returns underlying handler", func(t *testing.T) {
		// Arrange
		handler := func(ctx context.Context, messages []agent.Message) ([]agent.Message, error) {
			return nil, nil
		}
		executor := NewFunctionExecutor("func-executor", handler)

		// Act
		h := executor.Handler()

		// Assert
		assert.NotNil(t, h)
	})

	t.Run("returns nil with no messages", func(t *testing.T) {
		// Arrange
		handlerCalled := false
		executor := NewFunctionExecutor("func-executor", func(ctx context.Context, messages []agent.Message) ([]agent.Message, error) {
			handlerCalled = true
			return nil, nil
		})
		wCtx := newFunctionTestContext("func-executor", nil)

		// Act
		err := executor.Execute(context.Background(), wCtx)

		// Assert
		require.NoError(t, err)
		assert.False(t, handlerCalled, "handler should not be called when no messages")
	})

	t.Run("passes messages to handler and sends response", func(t *testing.T) {
		// Arrange
		var receivedMessages []agent.Message
		executor := NewFunctionExecutor("func-executor", func(ctx context.Context, messages []agent.Message) ([]agent.Message, error) {
			receivedMessages = messages
			return []agent.Message{
				newFuncTextMessage(chat.RoleAssistant, "transformed"),
			}, nil
		})

		inputMessages := []workflow.WorkflowMessage{
			{From: "input", To: "func-executor", Content: newFuncTextMessage(chat.RoleUser, "input-1")},
			{From: "input", To: "func-executor", Content: newFuncTextMessage(chat.RoleUser, "input-2")},
		}
		wCtx := newFunctionTestContext("func-executor", inputMessages)

		// Act
		err := executor.Execute(context.Background(), wCtx)

		// Assert
		require.NoError(t, err)
		require.Len(t, receivedMessages, 2)
		assert.Equal(t, "input-1", receivedMessages[0].Text())
		assert.Equal(t, "input-2", receivedMessages[1].Text())

		outbox := wCtx.Outbox()
		require.Len(t, outbox, 1)
		assert.Equal(t, "transformed", outbox[0].Content.Text())
	})

	t.Run("returns error from handler", func(t *testing.T) {
		// Arrange
		expectedErr := errors.New("handler error")
		executor := NewFunctionExecutor("func-executor", func(ctx context.Context, messages []agent.Message) ([]agent.Message, error) {
			return nil, expectedErr
		})

		inputMessages := []workflow.WorkflowMessage{
			{From: "input", To: "func-executor", Content: newFuncTextMessage(chat.RoleUser, "test")},
		}
		wCtx := newFunctionTestContext("func-executor", inputMessages)

		// Act
		err := executor.Execute(context.Background(), wCtx)

		// Assert
		assert.Equal(t, expectedErr, err)
	})

	t.Run("handles nil output from handler", func(t *testing.T) {
		// Arrange
		executor := NewFunctionExecutor("func-executor", func(ctx context.Context, messages []agent.Message) ([]agent.Message, error) {
			return nil, nil
		})

		inputMessages := []workflow.WorkflowMessage{
			{From: "input", To: "func-executor", Content: newFuncTextMessage(chat.RoleUser, "test")},
		}
		wCtx := newFunctionTestContext("func-executor", inputMessages)

		// Act
		err := executor.Execute(context.Background(), wCtx)

		// Assert
		require.NoError(t, err)
		assert.Empty(t, wCtx.Outbox())
	})

	t.Run("handles multiple output messages", func(t *testing.T) {
		// Arrange
		executor := NewFunctionExecutor("func-executor", func(ctx context.Context, messages []agent.Message) ([]agent.Message, error) {
			return []agent.Message{
				newFuncTextMessage(chat.RoleAssistant, "out-1"),
				newFuncTextMessage(chat.RoleAssistant, "out-2"),
				newFuncTextMessage(chat.RoleAssistant, "out-3"),
			}, nil
		})

		inputMessages := []workflow.WorkflowMessage{
			{From: "input", To: "func-executor", Content: newFuncTextMessage(chat.RoleUser, "test")},
		}
		wCtx := newFunctionTestContext("func-executor", inputMessages)

		// Act
		err := executor.Execute(context.Background(), wCtx)

		// Assert
		require.NoError(t, err)
		outbox := wCtx.Outbox()
		require.Len(t, outbox, 3)
		assert.Equal(t, "out-1", outbox[0].Content.Text())
		assert.Equal(t, "out-2", outbox[1].Content.Text())
		assert.Equal(t, "out-3", outbox[2].Content.Text())
	})

	t.Run("message transformation use case", func(t *testing.T) {
		// Arrange - uppercase transformer
		executor := NewFunctionExecutor("uppercase", func(ctx context.Context, messages []agent.Message) ([]agent.Message, error) {
			result := make([]agent.Message, len(messages))
			for i, msg := range messages {
				text := msg.Text()
				result[i] = newFuncTextMessage(msg.Role, "[UPPER] "+text)
			}
			return result, nil
		})

		inputMessages := []workflow.WorkflowMessage{
			{From: "input", Content: newFuncTextMessage(chat.RoleUser, "hello")},
		}
		wCtx := newFunctionTestContext("uppercase", inputMessages)

		// Act
		err := executor.Execute(context.Background(), wCtx)

		// Assert
		require.NoError(t, err)
		outbox := wCtx.Outbox()
		require.Len(t, outbox, 1)
		assert.Equal(t, "[UPPER] hello", outbox[0].Content.Text())
	})
}

// newFunctionTestContext creates a WorkflowContext for function executor testing.
func newFunctionTestContext(executorID string, messages []workflow.WorkflowMessage) *workflow.WorkflowContext {
	state := &sync.Map{}
	return workflow.NewWorkflowContextForTest(context.Background(), executorID, "test-run", 0, messages, state)
}
