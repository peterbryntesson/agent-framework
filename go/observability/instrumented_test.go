// Copyright (c) Microsoft. All rights reserved.

package observability

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/microsoft/agent-framework-go/chat"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockChatClient is a test implementation of chat.Client.
type mockChatClient struct {
	metadata        chat.ClientMetadata
	getResponseFunc func(ctx context.Context, messages []chat.Message, options *chat.Options) (*chat.Response, error)
	streamingFunc   func(ctx context.Context, messages []chat.Message, options *chat.Options) (<-chan chat.ResponseUpdate, error)
}

func (m *mockChatClient) GetResponse(ctx context.Context, messages []chat.Message, options *chat.Options) (*chat.Response, error) {
	if m.getResponseFunc != nil {
		return m.getResponseFunc(ctx, messages, options)
	}
	return &chat.Response{
		Message: chat.Message{
			Role:     chat.RoleAssistant,
			Contents: []chat.Content{chat.TextContent{Text: "Hello"}},
		},
		FinishReason: chat.FinishReasonStop,
		Usage: &chat.UsageDetails{
			InputTokens:  100,
			OutputTokens: 50,
			TotalTokens:  150,
		},
	}, nil
}

func (m *mockChatClient) GetStreamingResponse(ctx context.Context, messages []chat.Message, options *chat.Options) (<-chan chat.ResponseUpdate, error) {
	if m.streamingFunc != nil {
		return m.streamingFunc(ctx, messages, options)
	}
	updates := make(chan chat.ResponseUpdate, 3)
	go func() {
		defer close(updates)
		updates <- chat.ResponseUpdate{
			Kind: chat.UpdateKindContentDelta,
			Delta: &chat.ContentDelta{
				TextDelta: "Hello",
			},
		}
		updates <- chat.ResponseUpdate{
			Kind: chat.UpdateKindUsage,
			Usage: &chat.UsageDetails{
				InputTokens:  100,
				OutputTokens: 50,
			},
		}
		updates <- chat.ResponseUpdate{
			Kind: chat.UpdateKindDone,
		}
	}()
	return updates, nil
}

func (m *mockChatClient) Metadata() chat.ClientMetadata {
	return m.metadata
}

func TestNewInstrumentedClient_CreatesWrapper(t *testing.T) {
	// Arrange
	inner := &mockChatClient{
		metadata: chat.ClientMetadata{
			ProviderName: "openai",
			ModelID:      "gpt-4",
		},
	}

	// Act
	client := NewInstrumentedClient(inner)

	// Assert
	assert.NotNil(t, client)
	assert.NotNil(t, client.inner)
	assert.NotNil(t, client.metrics)
	assert.False(t, client.EnableSensitiveData)
}

func TestNewInstrumentedClientWithMetrics_UsesProvidedMetrics(t *testing.T) {
	// Arrange
	inner := &mockChatClient{}
	metrics, err := NewMetrics()
	require.NoError(t, err)

	// Act
	client := NewInstrumentedClientWithMetrics(inner, metrics)

	// Assert
	assert.Same(t, metrics, client.metrics)
}

func TestNewInstrumentedClientWithOptions_AppliesOptions(t *testing.T) {
	// Arrange
	inner := &mockChatClient{}
	metrics, _ := NewMetrics()

	// Act
	client := NewInstrumentedClientWithOptions(inner,
		WithSensitiveData(true),
		WithMetrics(metrics),
	)

	// Assert
	assert.True(t, client.EnableSensitiveData)
	assert.Same(t, metrics, client.metrics)
}

func TestInstrumentedClient_ImplementsChatClient(t *testing.T) {
	// Arrange
	inner := &mockChatClient{}
	client := NewInstrumentedClient(inner)

	// Assert - should compile and satisfy interface
	var _ chat.Client = client
}

func TestInstrumentedClient_Metadata_DelegatestoInner(t *testing.T) {
	// Arrange
	inner := &mockChatClient{
		metadata: chat.ClientMetadata{
			ProviderName: "openai",
			ModelID:      "gpt-4",
			EndpointURI:  "https://api.openai.com",
		},
	}
	client := NewInstrumentedClient(inner)

	// Act
	metadata := client.Metadata()

	// Assert
	assert.Equal(t, "openai", metadata.ProviderName)
	assert.Equal(t, "gpt-4", metadata.ModelID)
	assert.Equal(t, "https://api.openai.com", metadata.EndpointURI)
}

func TestInstrumentedClient_GetResponse_Success(t *testing.T) {
	// Arrange
	expectedResponse := &chat.Response{
		Message: chat.Message{
			Role:     chat.RoleAssistant,
			Contents: []chat.Content{chat.TextContent{Text: "Hello, world!"}},
		},
		FinishReason: chat.FinishReasonStop,
		Usage: &chat.UsageDetails{
			InputTokens:  100,
			OutputTokens: 50,
		},
	}

	inner := &mockChatClient{
		metadata: chat.ClientMetadata{
			ProviderName: "openai",
			ModelID:      "gpt-4",
		},
		getResponseFunc: func(ctx context.Context, messages []chat.Message, options *chat.Options) (*chat.Response, error) {
			return expectedResponse, nil
		},
	}
	client := NewInstrumentedClient(inner)
	ctx := context.Background()
	messages := []chat.Message{{Role: chat.RoleUser, Contents: []chat.Content{chat.TextContent{Text: "Hello"}}}}

	// Act
	resp, err := client.GetResponse(ctx, messages, nil)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, expectedResponse, resp)
}

func TestInstrumentedClient_GetResponse_WithOptions(t *testing.T) {
	// Arrange
	inner := &mockChatClient{
		metadata: chat.ClientMetadata{
			ProviderName: "openai",
			ModelID:      "gpt-4",
		},
	}
	client := NewInstrumentedClient(inner)
	ctx := context.Background()
	messages := []chat.Message{{Role: chat.RoleUser, Contents: []chat.Content{chat.TextContent{Text: "Hello"}}}}
	options := &chat.Options{
		MaxTokens:   1000,
		Temperature: 0.7,
		TopP:        0.9,
	}

	// Act
	resp, err := client.GetResponse(ctx, messages, options)

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, resp)
}

func TestInstrumentedClient_GetResponse_Error(t *testing.T) {
	// Arrange
	expectedErr := errors.New("API error")
	inner := &mockChatClient{
		metadata: chat.ClientMetadata{
			ProviderName: "openai",
			ModelID:      "gpt-4",
		},
		getResponseFunc: func(ctx context.Context, messages []chat.Message, options *chat.Options) (*chat.Response, error) {
			return nil, expectedErr
		},
	}
	client := NewInstrumentedClient(inner)
	ctx := context.Background()
	messages := []chat.Message{{Role: chat.RoleUser, Contents: []chat.Content{chat.TextContent{Text: "Hello"}}}}

	// Act
	resp, err := client.GetResponse(ctx, messages, nil)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Equal(t, expectedErr, err)
}

func TestInstrumentedClient_GetStreamingResponse_Success(t *testing.T) {
	// Arrange
	inner := &mockChatClient{
		metadata: chat.ClientMetadata{
			ProviderName: "openai",
			ModelID:      "gpt-4",
		},
	}
	client := NewInstrumentedClient(inner)
	ctx := context.Background()
	messages := []chat.Message{{Role: chat.RoleUser, Contents: []chat.Content{chat.TextContent{Text: "Hello"}}}}

	// Act
	updates, err := client.GetStreamingResponse(ctx, messages, nil)

	// Assert
	require.NoError(t, err)
	require.NotNil(t, updates)

	// Drain updates
	var count int
	for range updates {
		count++
	}
	assert.Greater(t, count, 0)
}

func TestInstrumentedClient_GetStreamingResponse_Error(t *testing.T) {
	// Arrange
	expectedErr := errors.New("stream error")
	inner := &mockChatClient{
		metadata: chat.ClientMetadata{
			ProviderName: "openai",
			ModelID:      "gpt-4",
		},
		streamingFunc: func(ctx context.Context, messages []chat.Message, options *chat.Options) (<-chan chat.ResponseUpdate, error) {
			return nil, expectedErr
		},
	}
	client := NewInstrumentedClient(inner)
	ctx := context.Background()
	messages := []chat.Message{{Role: chat.RoleUser, Contents: []chat.Content{chat.TextContent{Text: "Hello"}}}}

	// Act
	updates, err := client.GetStreamingResponse(ctx, messages, nil)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, updates)
	assert.Equal(t, expectedErr, err)
}

func TestInstrumentedClient_GetStreamingResponse_WithError(t *testing.T) {
	// Arrange
	inner := &mockChatClient{
		metadata: chat.ClientMetadata{
			ProviderName: "openai",
			ModelID:      "gpt-4",
		},
		streamingFunc: func(ctx context.Context, messages []chat.Message, options *chat.Options) (<-chan chat.ResponseUpdate, error) {
			updates := make(chan chat.ResponseUpdate, 2)
			go func() {
				defer close(updates)
				updates <- chat.ResponseUpdate{
					Kind:  chat.UpdateKindError,
					Error: errors.New("stream processing error"),
				}
				updates <- chat.ResponseUpdate{
					Kind: chat.UpdateKindDone,
				}
			}()
			return updates, nil
		},
	}
	client := NewInstrumentedClient(inner)
	ctx := context.Background()
	messages := []chat.Message{{Role: chat.RoleUser, Contents: []chat.Content{chat.TextContent{Text: "Hello"}}}}

	// Act
	updates, err := client.GetStreamingResponse(ctx, messages, nil)

	// Assert
	require.NoError(t, err)
	require.NotNil(t, updates)

	// Drain updates
	for range updates {
	}
}

func TestInstrumentClient_ConvenienceFunction(t *testing.T) {
	// Arrange
	inner := &mockChatClient{}

	// Act
	client := InstrumentClient(inner)

	// Assert
	assert.NotNil(t, client)
	_, ok := client.(*InstrumentedClient)
	assert.True(t, ok)
}

func TestInstrumentClientWithSensitiveData_ConvenienceFunction(t *testing.T) {
	// Arrange
	inner := &mockChatClient{}

	// Act
	client := InstrumentClientWithSensitiveData(inner)

	// Assert
	assert.NotNil(t, client)
	ic, ok := client.(*InstrumentedClient)
	assert.True(t, ok)
	assert.True(t, ic.EnableSensitiveData)
}

func TestContextWithMetrics_StoresMetrics(t *testing.T) {
	// Arrange
	metrics, _ := NewMetrics()
	ctx := context.Background()

	// Act
	newCtx := ContextWithMetrics(ctx, metrics)
	retrieved := MetricsFromContext(newCtx)

	// Assert
	assert.Same(t, metrics, retrieved)
}

func TestMetricsFromContext_ReturnsDefaultWhenNotSet(t *testing.T) {
	// Arrange
	// Reset default metrics
	defaultMetrics = nil
	ctx := context.Background()

	// Act
	metrics := MetricsFromContext(ctx)

	// Assert
	assert.NotNil(t, metrics)
}

func TestRecordAgentRunFromContext_WithMetrics(t *testing.T) {
	// Arrange
	metrics, _ := NewMetrics()
	ctx := ContextWithMetrics(context.Background(), metrics)

	// Act - should not panic
	RecordAgentRunFromContext(ctx, "openai", "gpt-4")
}

func TestRecordTokenUsageFromContext_WithMetrics(t *testing.T) {
	// Arrange
	metrics, _ := NewMetrics()
	ctx := ContextWithMetrics(context.Background(), metrics)

	// Act - should not panic
	RecordTokenUsageFromContext(ctx, 100, 50, "openai", "gpt-4")
}

func TestRecordLatencyFromContext_WithMetrics(t *testing.T) {
	// Arrange
	metrics, _ := NewMetrics()
	ctx := ContextWithMetrics(context.Background(), metrics)

	// Act - should not panic
	RecordLatencyFromContext(ctx, 1.5, "openai", "gpt-4")
}

func TestRecordErrorFromContext_WithMetrics(t *testing.T) {
	// Arrange
	metrics, _ := NewMetrics()
	ctx := ContextWithMetrics(context.Background(), metrics)

	// Act - should not panic
	RecordErrorFromContext(ctx, "timeout", "openai", "gpt-4")
}

func TestRecordToolInvocationFromContext_WithMetrics(t *testing.T) {
	// Arrange
	metrics, _ := NewMetrics()
	ctx := ContextWithMetrics(context.Background(), metrics)

	// Act - should not panic
	RecordToolInvocationFromContext(ctx, "get_weather", true)
}

func TestNewInstrumentedAgentClient_CreatesClient(t *testing.T) {
	// Act
	client := NewInstrumentedAgentClient()

	// Assert
	assert.NotNil(t, client)
	assert.NotNil(t, client.metrics)
}

func TestInstrumentedAgentClient_StartRun(t *testing.T) {
	// Arrange
	client := NewInstrumentedAgentClient()
	ctx := context.Background()

	// Act
	newCtx, finish := client.StartRun(ctx, "agent-1", "TestAgent", "openai", "gpt-4")

	// Assert
	assert.NotNil(t, newCtx)
	assert.NotNil(t, finish)

	// Call finish to clean up
	finish(nil, nil)
}

func TestInstrumentedAgentClient_StartRun_WithError(t *testing.T) {
	// Arrange
	client := NewInstrumentedAgentClient()
	ctx := context.Background()

	// Act
	_, finish := client.StartRun(ctx, "agent-1", "TestAgent", "openai", "gpt-4")

	// Finish with error
	finish(errors.New("test error"), nil)
}

func TestInstrumentedAgentClient_StartRun_WithUsage(t *testing.T) {
	// Arrange
	client := NewInstrumentedAgentClient()
	ctx := context.Background()
	usage := &chat.UsageDetails{
		InputTokens:  100,
		OutputTokens: 50,
	}

	// Act
	_, finish := client.StartRun(ctx, "agent-1", "TestAgent", "openai", "gpt-4")

	// Finish with usage
	finish(nil, usage)
}

func TestInstrumentedClient_ConcurrentRequests(t *testing.T) {
	// Arrange
	inner := &mockChatClient{
		metadata: chat.ClientMetadata{
			ProviderName: "openai",
			ModelID:      "gpt-4",
		},
	}
	client := NewInstrumentedClient(inner)
	ctx := context.Background()
	messages := []chat.Message{{Role: chat.RoleUser, Contents: []chat.Content{chat.TextContent{Text: "Hello"}}}}

	var wg sync.WaitGroup
	numRequests := 10

	// Act - Make concurrent requests
	for i := 0; i < numRequests; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := client.GetResponse(ctx, messages, nil)
			assert.NoError(t, err)
		}()
	}

	wg.Wait()
}

func TestErrorTypeName_WithNilError(t *testing.T) {
	// Act
	result := errorTypeName(nil)

	// Assert
	assert.Equal(t, "unknown", result)
}

func TestErrorTypeName_WithError(t *testing.T) {
	// Act
	result := errorTypeName(errors.New("test"))

	// Assert
	assert.Equal(t, "error", result)
}
