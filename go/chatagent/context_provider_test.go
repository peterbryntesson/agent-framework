// Copyright (c) Microsoft. All rights reserved.

package chatagent

import (
	"context"
	"errors"
	"testing"

	"github.com/microsoft/agent-framework-go/agent"
	"github.com/microsoft/agent-framework-go/chat"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// captureClient is a mock client that captures messages sent to it.
type captureClient struct {
	capturedMessages []chat.Message
	capturedOptions  *chat.Options
	response         *chat.Response
}

func newCaptureClient() *captureClient {
	return &captureClient{
		response: &chat.Response{
			Message: chat.NewAssistantMessage("Test response"),
			Usage:   &chat.UsageDetails{InputTokens: 10, OutputTokens: 5, TotalTokens: 15},
		},
	}
}

func (c *captureClient) GetResponse(ctx context.Context, messages []chat.Message, options *chat.Options) (*chat.Response, error) {
	c.capturedMessages = messages
	c.capturedOptions = options
	return c.response, nil
}

func (c *captureClient) GetStreamingResponse(ctx context.Context, messages []chat.Message, options *chat.Options) (<-chan chat.ResponseUpdate, error) {
	c.capturedMessages = messages
	c.capturedOptions = options

	updates := make(chan chat.ResponseUpdate, 2)
	go func() {
		defer close(updates)
		updates <- chat.ResponseUpdate{
			Kind:  chat.UpdateKindContentDelta,
			Delta: &chat.ContentDelta{TextDelta: "Test response"},
		}
		updates <- chat.ResponseUpdate{
			Kind:         chat.UpdateKindDone,
			FinishReason: chat.FinishReasonStop,
		}
	}()

	return updates, nil
}

func (c *captureClient) Metadata() chat.ClientMetadata {
	return chat.ClientMetadata{
		ProviderName: "capture",
		ModelID:      "capture-model",
	}
}

func TestAgent_WithContextProvider_InjectsInstructions(t *testing.T) {
	client := newCaptureClient()

	provider := agent.ContextProviderFunc(func(ctx context.Context, messages []agent.Message) (*agent.Context, error) {
		return &agent.Context{
			Instructions: "You are a helpful assistant.",
		}, nil
	})

	a := New(client,
		WithInstructions("Base instructions."),
		WithContextProvider(provider),
	)

	_, err := a.Run(context.Background(), []agent.Message{
		agent.NewUserMessage("Hi"),
	})
	require.NoError(t, err)

	// Verify instructions were merged
	require.GreaterOrEqual(t, len(client.capturedMessages), 2) // system + user at minimum
	assert.Equal(t, chat.RoleSystem, client.capturedMessages[0].Role)
	systemText := client.capturedMessages[0].Text()
	assert.Contains(t, systemText, "Base instructions.")
	assert.Contains(t, systemText, "You are a helpful assistant.")
}

func TestAgent_WithContextProvider_InjectsMessages(t *testing.T) {
	client := newCaptureClient()

	contextMsg := chat.NewUserMessage("Context message")
	provider := agent.ContextProviderFunc(func(ctx context.Context, messages []agent.Message) (*agent.Context, error) {
		return &agent.Context{
			Messages: []chat.Message{contextMsg},
		}, nil
	})

	a := New(client,
		WithContextProvider(provider),
	)

	_, err := a.Run(context.Background(), []agent.Message{
		agent.NewUserMessage("User message"),
	})
	require.NoError(t, err)

	// Verify context message appears before user message
	require.Len(t, client.capturedMessages, 2)
	assert.Equal(t, "Context message", client.capturedMessages[0].Text())
	assert.Equal(t, "User message", client.capturedMessages[1].Text())
}

func TestAgent_WithContextProvider_ProviderOnlyInstructions(t *testing.T) {
	client := newCaptureClient()

	provider := agent.ContextProviderFunc(func(ctx context.Context, messages []agent.Message) (*agent.Context, error) {
		return &agent.Context{
			Instructions: "Provider-only instructions.",
		}, nil
	})

	a := New(client,
		WithContextProvider(provider),
	)

	_, err := a.Run(context.Background(), []agent.Message{
		agent.NewUserMessage("Hi"),
	})
	require.NoError(t, err)

	// Verify provider instructions are used even without base instructions
	require.GreaterOrEqual(t, len(client.capturedMessages), 2)
	assert.Equal(t, chat.RoleSystem, client.capturedMessages[0].Role)
	assert.Equal(t, "Provider-only instructions.", client.capturedMessages[0].Text())
}

func TestAgent_WithContextProvider_Error_AbortsRun(t *testing.T) {
	client := newCaptureClient()
	expectedErr := errors.New("provider failed")

	provider := agent.ContextProviderFunc(func(ctx context.Context, messages []agent.Message) (*agent.Context, error) {
		return nil, expectedErr
	})

	a := New(client,
		WithContextProvider(provider),
	)

	_, err := a.Run(context.Background(), []agent.Message{
		agent.NewUserMessage("Hi"),
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "context provider error")
}

func TestAgent_WithMultipleContextProviders_MergesContext(t *testing.T) {
	client := newCaptureClient()

	p1 := agent.ContextProviderFunc(func(ctx context.Context, messages []agent.Message) (*agent.Context, error) {
		return &agent.Context{
			Instructions: "First instruction.",
		}, nil
	})

	p2 := agent.ContextProviderFunc(func(ctx context.Context, messages []agent.Message) (*agent.Context, error) {
		return &agent.Context{
			Instructions: "Second instruction.",
		}, nil
	})

	a := New(client,
		WithInstructions("Base."),
		WithContextProvider(p1, p2),
	)

	_, err := a.Run(context.Background(), []agent.Message{
		agent.NewUserMessage("Hi"),
	})
	require.NoError(t, err)

	// Verify all instructions are merged
	systemText := client.capturedMessages[0].Text()
	assert.Contains(t, systemText, "Base.")
	assert.Contains(t, systemText, "First instruction.")
	assert.Contains(t, systemText, "Second instruction.")
}

func TestAgent_WithContextProvider_MessageOrder(t *testing.T) {
	client := newCaptureClient()

	provider := agent.ContextProviderFunc(func(ctx context.Context, messages []agent.Message) (*agent.Context, error) {
		return &agent.Context{
			Instructions: "Provider instructions",
			Messages:     []chat.Message{chat.NewUserMessage("Provider context")},
		}, nil
	})

	a := New(client,
		WithInstructions("Base instructions"),
		WithContextProvider(provider),
	)

	_, err := a.Run(context.Background(), []agent.Message{
		agent.NewUserMessage("User input"),
	})
	require.NoError(t, err)

	// Verify order: system (merged) -> provider messages -> user messages
	require.Len(t, client.capturedMessages, 3)
	assert.Equal(t, chat.RoleSystem, client.capturedMessages[0].Role)
	assert.Equal(t, "Provider context", client.capturedMessages[1].Text())
	assert.Equal(t, "User input", client.capturedMessages[2].Text())
}

func TestAgent_WithContextProvider_NilContext(t *testing.T) {
	client := newCaptureClient()

	provider := agent.ContextProviderFunc(func(ctx context.Context, messages []agent.Message) (*agent.Context, error) {
		return nil, nil // Provider returns nil context
	})

	a := New(client,
		WithInstructions("Base"),
		WithContextProvider(provider),
	)

	_, err := a.Run(context.Background(), []agent.Message{
		agent.NewUserMessage("Hi"),
	})
	require.NoError(t, err)

	// Should still work with just base instructions
	assert.Equal(t, chat.RoleSystem, client.capturedMessages[0].Role)
	assert.Equal(t, "Base", client.capturedMessages[0].Text())
}

func TestAgent_WithContextProvider_EmptyContext(t *testing.T) {
	client := newCaptureClient()

	provider := agent.ContextProviderFunc(func(ctx context.Context, messages []agent.Message) (*agent.Context, error) {
		return &agent.Context{}, nil // Empty context
	})

	a := New(client,
		WithInstructions("Base"),
		WithContextProvider(provider),
	)

	_, err := a.Run(context.Background(), []agent.Message{
		agent.NewUserMessage("Hi"),
	})
	require.NoError(t, err)

	// Should still work with just base instructions
	assert.Equal(t, chat.RoleSystem, client.capturedMessages[0].Role)
	assert.Equal(t, "Base", client.capturedMessages[0].Text())
}

// lifecycleProvider tracks lifecycle hook calls for testing.
type lifecycleProvider struct {
	agent.BaseContextProvider
	invokingCalled       bool
	invokedCalled        bool
	sessionCreatedCalled bool
	invokedRequest       []agent.Message
	invokedResponse      []agent.Message
	invokedError         error
}

func (p *lifecycleProvider) Invoking(ctx context.Context, messages []agent.Message) (*agent.Context, error) {
	p.invokingCalled = true
	return &agent.Context{Instructions: "Lifecycle test"}, nil
}

func (p *lifecycleProvider) Invoked(ctx context.Context, request, response []agent.Message, invokeErr error) error {
	p.invokedCalled = true
	p.invokedRequest = request
	p.invokedResponse = response
	p.invokedError = invokeErr
	return nil
}

func (p *lifecycleProvider) SessionCreated(ctx context.Context, sessionID string) error {
	p.sessionCreatedCalled = true
	return nil
}

func TestAgent_WithContextProvider_CallsLifecycleHooks(t *testing.T) {
	client := newCaptureClient()
	provider := &lifecycleProvider{}

	a := New(client,
		WithContextProvider(provider),
	)

	inputMessages := []agent.Message{agent.NewUserMessage("Hello")}
	_, err := a.Run(context.Background(), inputMessages)
	require.NoError(t, err)

	// Verify lifecycle hooks were called
	assert.True(t, provider.invokingCalled, "Invoking should be called")
	assert.True(t, provider.invokedCalled, "Invoked should be called")
	assert.Equal(t, inputMessages, provider.invokedRequest)
	assert.NotNil(t, provider.invokedResponse)
}

func TestAgent_RunStream_WithContextProvider(t *testing.T) {
	client := newCaptureClient()

	provider := agent.ContextProviderFunc(func(ctx context.Context, messages []agent.Message) (*agent.Context, error) {
		return &agent.Context{
			Instructions: "Stream provider instructions",
		}, nil
	})

	a := New(client,
		WithInstructions("Base"),
		WithContextProvider(provider),
	)

	updates, err := a.RunStream(context.Background(), []agent.Message{
		agent.NewUserMessage("Hi"),
	})
	require.NoError(t, err)

	// Drain the channel
	for range updates {
	}

	// Verify instructions were merged
	systemText := client.capturedMessages[0].Text()
	assert.Contains(t, systemText, "Base")
	assert.Contains(t, systemText, "Stream provider instructions")
}
