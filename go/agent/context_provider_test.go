// Copyright (c) Microsoft. All rights reserved.

package agent

import (
	"context"
	"errors"
	"testing"

	"github.com/microsoft/agent-framework-go/chat"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestContext_ZeroValue(t *testing.T) {
	var ctx Context
	assert.Empty(t, ctx.Instructions)
	assert.Nil(t, ctx.Messages)
	assert.Nil(t, ctx.Tools)
}

func TestContextProviderFunc_Invoking(t *testing.T) {
	provider := ContextProviderFunc(func(ctx context.Context, messages []Message) (*Context, error) {
		return &Context{
			Instructions: "Test instructions",
		}, nil
	})

	result, err := provider.Invoking(context.Background(), nil)
	require.NoError(t, err)
	assert.Equal(t, "Test instructions", result.Instructions)
}

func TestContextProviderFunc_Invoking_WithMessages(t *testing.T) {
	inputMessages := []Message{NewUserMessage("Hello")}
	provider := ContextProviderFunc(func(ctx context.Context, messages []Message) (*Context, error) {
		// Verify messages are passed through
		assert.Len(t, messages, 1)
		assert.Equal(t, "Hello", messages[0].Text())
		return &Context{Instructions: "Based on messages"}, nil
	})

	result, err := provider.Invoking(context.Background(), inputMessages)
	require.NoError(t, err)
	assert.Equal(t, "Based on messages", result.Instructions)
}

func TestContextProviderFunc_Invoking_ReturnsError(t *testing.T) {
	expectedErr := errors.New("provider error")
	provider := ContextProviderFunc(func(ctx context.Context, messages []Message) (*Context, error) {
		return nil, expectedErr
	})

	result, err := provider.Invoking(context.Background(), nil)
	assert.ErrorIs(t, err, expectedErr)
	assert.Nil(t, result)
}

func TestBaseContextProvider_NoOpMethods(t *testing.T) {
	var base BaseContextProvider

	err := base.Invoked(context.Background(), nil, nil, nil)
	assert.NoError(t, err)

	err = base.SessionCreated(context.Background(), "session-123")
	assert.NoError(t, err)
}

func TestBaseContextProvider_Invoked_WithParams(t *testing.T) {
	var base BaseContextProvider
	request := []Message{NewUserMessage("Request")}
	response := []Message{NewAssistantMessage("Response")}
	invokeErr := errors.New("invoke error")

	err := base.Invoked(context.Background(), request, response, invokeErr)
	assert.NoError(t, err)
}

func TestAggregateContextProvider_Empty(t *testing.T) {
	agg := NewAggregateContextProvider()

	result, err := agg.Invoking(context.Background(), nil)
	require.NoError(t, err)
	assert.Empty(t, result.Instructions)
	assert.Empty(t, result.Messages)
	assert.Empty(t, result.Tools)
}

func TestAggregateContextProvider_SingleProvider(t *testing.T) {
	provider := ContextProviderFunc(func(ctx context.Context, messages []Message) (*Context, error) {
		return &Context{Instructions: "Hello"}, nil
	})
	agg := NewAggregateContextProvider(provider)

	result, err := agg.Invoking(context.Background(), nil)
	require.NoError(t, err)
	assert.Equal(t, "Hello", result.Instructions)
}

func TestAggregateContextProvider_MergesInstructions(t *testing.T) {
	p1 := ContextProviderFunc(func(ctx context.Context, messages []Message) (*Context, error) {
		return &Context{Instructions: "First"}, nil
	})
	p2 := ContextProviderFunc(func(ctx context.Context, messages []Message) (*Context, error) {
		return &Context{Instructions: "Second"}, nil
	})
	agg := NewAggregateContextProvider(p1, p2)

	result, err := agg.Invoking(context.Background(), nil)
	require.NoError(t, err)
	assert.Equal(t, "First\nSecond", result.Instructions)
}

func TestAggregateContextProvider_MergesMessages(t *testing.T) {
	msg1 := chat.NewUserMessage("User 1")
	msg2 := chat.NewUserMessage("User 2")

	p1 := ContextProviderFunc(func(ctx context.Context, messages []Message) (*Context, error) {
		return &Context{Messages: []chat.Message{msg1}}, nil
	})
	p2 := ContextProviderFunc(func(ctx context.Context, messages []Message) (*Context, error) {
		return &Context{Messages: []chat.Message{msg2}}, nil
	})
	agg := NewAggregateContextProvider(p1, p2)

	result, err := agg.Invoking(context.Background(), nil)
	require.NoError(t, err)
	require.Len(t, result.Messages, 2)
	assert.Equal(t, "User 1", result.Messages[0].Text())
	assert.Equal(t, "User 2", result.Messages[1].Text())
}

func TestAggregateContextProvider_ErrorReturnsFirst(t *testing.T) {
	expectedErr := errors.New("provider error")
	p1 := ContextProviderFunc(func(ctx context.Context, messages []Message) (*Context, error) {
		return nil, expectedErr
	})
	p2 := ContextProviderFunc(func(ctx context.Context, messages []Message) (*Context, error) {
		return &Context{Instructions: "OK"}, nil
	})
	agg := NewAggregateContextProvider(p1, p2)

	_, err := agg.Invoking(context.Background(), nil)
	assert.Error(t, err)
}

func TestAggregateContextProvider_PreservesOrder(t *testing.T) {
	// Test that despite concurrent execution, results are ordered by provider index
	for i := 0; i < 10; i++ { // Run multiple times to catch race conditions
		p1 := ContextProviderFunc(func(ctx context.Context, messages []Message) (*Context, error) {
			return &Context{Instructions: "A"}, nil
		})
		p2 := ContextProviderFunc(func(ctx context.Context, messages []Message) (*Context, error) {
			return &Context{Instructions: "B"}, nil
		})
		p3 := ContextProviderFunc(func(ctx context.Context, messages []Message) (*Context, error) {
			return &Context{Instructions: "C"}, nil
		})
		agg := NewAggregateContextProvider(p1, p2, p3)

		result, err := agg.Invoking(context.Background(), nil)
		require.NoError(t, err)
		assert.Equal(t, "A\nB\nC", result.Instructions)
	}
}

func TestAggregateContextProvider_Add(t *testing.T) {
	p1 := ContextProviderFunc(func(ctx context.Context, messages []Message) (*Context, error) {
		return &Context{Instructions: "First"}, nil
	})
	p2 := ContextProviderFunc(func(ctx context.Context, messages []Message) (*Context, error) {
		return &Context{Instructions: "Second"}, nil
	})

	agg := NewAggregateContextProvider(p1)
	agg.Add(p2)

	assert.Len(t, agg.Providers(), 2)

	result, err := agg.Invoking(context.Background(), nil)
	require.NoError(t, err)
	assert.Equal(t, "First\nSecond", result.Instructions)
}

func TestAggregateContextProvider_SkipsNilContexts(t *testing.T) {
	p1 := ContextProviderFunc(func(ctx context.Context, messages []Message) (*Context, error) {
		return &Context{Instructions: "Hello"}, nil
	})
	p2 := ContextProviderFunc(func(ctx context.Context, messages []Message) (*Context, error) {
		return nil, nil // Returns nil context
	})
	p3 := ContextProviderFunc(func(ctx context.Context, messages []Message) (*Context, error) {
		return &Context{Instructions: "World"}, nil
	})
	agg := NewAggregateContextProvider(p1, p2, p3)

	result, err := agg.Invoking(context.Background(), nil)
	require.NoError(t, err)
	assert.Equal(t, "Hello\nWorld", result.Instructions)
}

func TestAggregateContextProvider_SkipsEmptyInstructions(t *testing.T) {
	p1 := ContextProviderFunc(func(ctx context.Context, messages []Message) (*Context, error) {
		return &Context{Instructions: "Hello"}, nil
	})
	p2 := ContextProviderFunc(func(ctx context.Context, messages []Message) (*Context, error) {
		return &Context{Instructions: ""}, nil // Empty instructions
	})
	p3 := ContextProviderFunc(func(ctx context.Context, messages []Message) (*Context, error) {
		return &Context{Instructions: "World"}, nil
	})
	agg := NewAggregateContextProvider(p1, p2, p3)

	result, err := agg.Invoking(context.Background(), nil)
	require.NoError(t, err)
	assert.Equal(t, "Hello\nWorld", result.Instructions)
}

// mockLifecycleProvider implements ContextProviderWithLifecycle for testing
type mockLifecycleProvider struct {
	BaseContextProvider
	invokedCalled        bool
	sessionCreatedCalled bool
	invokedRequest       []Message
	invokedResponse      []Message
	invokedError         error
	sessionID            string
	returnError          error
}

func (m *mockLifecycleProvider) Invoking(ctx context.Context, messages []Message) (*Context, error) {
	return &Context{}, nil
}

func (m *mockLifecycleProvider) Invoked(ctx context.Context, request, response []Message, invokeErr error) error {
	m.invokedCalled = true
	m.invokedRequest = request
	m.invokedResponse = response
	m.invokedError = invokeErr
	return m.returnError
}

func (m *mockLifecycleProvider) SessionCreated(ctx context.Context, sessionID string) error {
	m.sessionCreatedCalled = true
	m.sessionID = sessionID
	return m.returnError
}

func TestAggregateContextProvider_Invoked_CallsLifecycleProviders(t *testing.T) {
	mock := &mockLifecycleProvider{}
	agg := NewAggregateContextProvider(mock)

	request := []Message{NewUserMessage("Hello")}
	response := []Message{NewAssistantMessage("Hi")}

	err := agg.Invoked(context.Background(), request, response, nil)
	require.NoError(t, err)
	assert.True(t, mock.invokedCalled)
	assert.Equal(t, request, mock.invokedRequest)
	assert.Equal(t, response, mock.invokedResponse)
}

func TestAggregateContextProvider_Invoked_WithError(t *testing.T) {
	mock := &mockLifecycleProvider{}
	agg := NewAggregateContextProvider(mock)

	expectedErr := errors.New("invoke error")
	err := agg.Invoked(context.Background(), nil, nil, expectedErr)
	require.NoError(t, err)
	assert.True(t, mock.invokedCalled)
	assert.Equal(t, expectedErr, mock.invokedError)
}

func TestAggregateContextProvider_Invoked_ReturnsError(t *testing.T) {
	expectedErr := errors.New("lifecycle error")
	mock := &mockLifecycleProvider{returnError: expectedErr}
	agg := NewAggregateContextProvider(mock)

	err := agg.Invoked(context.Background(), nil, nil, nil)
	assert.ErrorIs(t, err, expectedErr)
}

func TestAggregateContextProvider_SessionCreated_CallsLifecycleProviders(t *testing.T) {
	mock := &mockLifecycleProvider{}
	agg := NewAggregateContextProvider(mock)

	err := agg.SessionCreated(context.Background(), "session-456")
	require.NoError(t, err)
	assert.True(t, mock.sessionCreatedCalled)
	assert.Equal(t, "session-456", mock.sessionID)
}

func TestAggregateContextProvider_SessionCreated_ReturnsError(t *testing.T) {
	expectedErr := errors.New("session error")
	mock := &mockLifecycleProvider{returnError: expectedErr}
	agg := NewAggregateContextProvider(mock)

	err := agg.SessionCreated(context.Background(), "session-789")
	assert.ErrorIs(t, err, expectedErr)
}

func TestAggregateContextProvider_SkipsNonLifecycleProviders(t *testing.T) {
	// Simple provider without lifecycle
	simple := ContextProviderFunc(func(ctx context.Context, messages []Message) (*Context, error) {
		return &Context{}, nil
	})
	mock := &mockLifecycleProvider{}
	agg := NewAggregateContextProvider(simple, mock)

	err := agg.Invoked(context.Background(), nil, nil, nil)
	require.NoError(t, err)
	assert.True(t, mock.invokedCalled)

	mock.sessionCreatedCalled = false
	err = agg.SessionCreated(context.Background(), "test-session")
	require.NoError(t, err)
	assert.True(t, mock.sessionCreatedCalled)
}
