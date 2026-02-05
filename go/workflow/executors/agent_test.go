// Copyright (c) Microsoft. All rights reserved.

package executors

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"sync"
	"testing"

	"github.com/microsoft/agent-framework-go/agent"
	"github.com/microsoft/agent-framework-go/chat"
	"github.com/microsoft/agent-framework-go/workflow"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockAgent is a test implementation of agent.Agent.
type mockAgent struct {
	id          string
	name        string
	description string
	runFunc     func(ctx context.Context, messages []agent.Message, opts ...agent.RunOption) (*agent.Response, error)
}

func (m *mockAgent) ID() string                          { return m.id }
func (m *mockAgent) Name() string                        { return m.name }
func (m *mockAgent) Description() string                 { return m.description }
func (m *mockAgent) Metadata() agent.AIAgentMetadata     { return agent.AIAgentMetadata{} }
func (m *mockAgent) GetService(reflect.Type) interface{} { return nil }

func (m *mockAgent) Run(ctx context.Context, messages []agent.Message, opts ...agent.RunOption) (*agent.Response, error) {
	if m.runFunc != nil {
		return m.runFunc(ctx, messages, opts...)
	}
	return &agent.Response{}, nil
}

func (m *mockAgent) RunStream(ctx context.Context, messages []agent.Message, opts ...agent.RunOption) (<-chan agent.ResponseUpdate, error) {
	return nil, errors.New("not implemented")
}

func (m *mockAgent) NewSession(ctx context.Context) (agent.Session, error) {
	return nil, errors.New("not implemented")
}

func (m *mockAgent) RestoreSession(ctx context.Context, data json.RawMessage) (agent.Session, error) {
	return nil, errors.New("not implemented")
}

// newTextMessage creates a message with text content for testing.
func newTextMessage(role chat.Role, text string) agent.Message {
	return agent.Message{
		Role:     role,
		Contents: []chat.Content{chat.NewTextContent(text)},
	}
}

func TestAgentExecutor(t *testing.T) {
	t.Run("returns correct ID", func(t *testing.T) {
		// Arrange
		mockAgent := &mockAgent{id: "test-agent"}
		executor := NewAgentExecutor("agent-executor", mockAgent)

		// Act
		id := executor.ID()

		// Assert
		assert.Equal(t, "agent-executor", id)
	})

	t.Run("returns underlying agent", func(t *testing.T) {
		// Arrange
		mockAgent := &mockAgent{id: "test-agent", name: "Test Agent"}
		executor := NewAgentExecutor("agent-executor", mockAgent)

		// Act
		ag := executor.Agent()

		// Assert
		assert.Equal(t, mockAgent, ag)
		assert.Equal(t, "Test Agent", ag.Name())
	})

	t.Run("returns nil with no messages", func(t *testing.T) {
		// Arrange
		runCalled := false
		mockAgent := &mockAgent{
			runFunc: func(ctx context.Context, messages []agent.Message, opts ...agent.RunOption) (*agent.Response, error) {
				runCalled = true
				return &agent.Response{}, nil
			},
		}
		executor := NewAgentExecutor("agent-executor", mockAgent)
		wCtx := newTestWorkflowContext("agent-executor", nil)

		// Act
		err := executor.Execute(context.Background(), wCtx)

		// Assert
		require.NoError(t, err)
		assert.False(t, runCalled, "agent.Run should not be called when no messages")
	})

	t.Run("passes messages to agent and sends response", func(t *testing.T) {
		// Arrange
		var receivedMessages []agent.Message
		mockAgent := &mockAgent{
			runFunc: func(ctx context.Context, messages []agent.Message, opts ...agent.RunOption) (*agent.Response, error) {
				receivedMessages = messages
				return &agent.Response{
					Messages: []agent.Message{
						newTextMessage(chat.RoleAssistant, "response"),
					},
				}, nil
			},
		}
		executor := NewAgentExecutor("agent-executor", mockAgent)

		inputMessages := []workflow.WorkflowMessage{
			{From: "input", To: "agent-executor", Content: newTextMessage(chat.RoleUser, "hello")},
			{From: "input", To: "agent-executor", Content: newTextMessage(chat.RoleUser, "world")},
		}
		wCtx := newTestWorkflowContext("agent-executor", inputMessages)

		// Act
		err := executor.Execute(context.Background(), wCtx)

		// Assert
		require.NoError(t, err)
		require.Len(t, receivedMessages, 2)
		assert.Equal(t, "hello", receivedMessages[0].Text())
		assert.Equal(t, "world", receivedMessages[1].Text())

		outbox := wCtx.Outbox()
		require.Len(t, outbox, 1)
		assert.Equal(t, "response", outbox[0].Content.Text())
	})

	t.Run("returns error from agent", func(t *testing.T) {
		// Arrange
		expectedErr := errors.New("agent error")
		mockAgent := &mockAgent{
			runFunc: func(ctx context.Context, messages []agent.Message, opts ...agent.RunOption) (*agent.Response, error) {
				return nil, expectedErr
			},
		}
		executor := NewAgentExecutor("agent-executor", mockAgent)

		inputMessages := []workflow.WorkflowMessage{
			{From: "input", To: "agent-executor", Content: newTextMessage(chat.RoleUser, "test")},
		}
		wCtx := newTestWorkflowContext("agent-executor", inputMessages)

		// Act
		err := executor.Execute(context.Background(), wCtx)

		// Assert
		require.Error(t, err)
		assert.Contains(t, err.Error(), "agent execution failed")
		assert.Contains(t, err.Error(), "agent error")
	})

	t.Run("respects context cancellation", func(t *testing.T) {
		// Arrange
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		mockAgent := &mockAgent{
			runFunc: func(ctx context.Context, messages []agent.Message, opts ...agent.RunOption) (*agent.Response, error) {
				if ctx.Err() != nil {
					return nil, ctx.Err()
				}
				return &agent.Response{}, nil
			},
		}
		executor := NewAgentExecutor("agent-executor", mockAgent)

		inputMessages := []workflow.WorkflowMessage{
			{From: "input", To: "agent-executor", Content: newTextMessage(chat.RoleUser, "test")},
		}
		wCtx := newTestWorkflowContext("agent-executor", inputMessages)

		// Act
		err := executor.Execute(ctx, wCtx)

		// Assert
		require.Error(t, err)
		assert.Contains(t, err.Error(), "canceled")
	})

	t.Run("handles multiple response messages", func(t *testing.T) {
		// Arrange
		mockAgent := &mockAgent{
			runFunc: func(ctx context.Context, messages []agent.Message, opts ...agent.RunOption) (*agent.Response, error) {
				return &agent.Response{
					Messages: []agent.Message{
						newTextMessage(chat.RoleAssistant, "first"),
						newTextMessage(chat.RoleAssistant, "second"),
						newTextMessage(chat.RoleAssistant, "third"),
					},
				}, nil
			},
		}
		executor := NewAgentExecutor("agent-executor", mockAgent)

		inputMessages := []workflow.WorkflowMessage{
			{From: "input", To: "agent-executor", Content: newTextMessage(chat.RoleUser, "test")},
		}
		wCtx := newTestWorkflowContext("agent-executor", inputMessages)

		// Act
		err := executor.Execute(context.Background(), wCtx)

		// Assert
		require.NoError(t, err)
		outbox := wCtx.Outbox()
		require.Len(t, outbox, 3)
		assert.Equal(t, "first", outbox[0].Content.Text())
		assert.Equal(t, "second", outbox[1].Content.Text())
		assert.Equal(t, "third", outbox[2].Content.Text())
	})
}

// newTestWorkflowContext creates a WorkflowContext for testing.
func newTestWorkflowContext(executorID string, messages []workflow.WorkflowMessage) *workflow.WorkflowContext {
	state := &sync.Map{}
	return workflow.NewWorkflowContextForTest(context.Background(), executorID, "test-run", 0, messages, state)
}
