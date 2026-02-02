// Copyright (c) Microsoft. All rights reserved.

package chat_test

import (
	"context"

	"github.com/microsoft/agent-framework-go/chat"
)

// MockChatClient is a mock implementation of the Client interface for testing.
// All methods delegate to configurable function fields, allowing tests to
// customize behavior without creating separate mock types.
type MockChatClient struct {
	// GetResponseFunc is called by GetResponse(). Returns nil, nil if nil.
	GetResponseFunc func(ctx context.Context, messages []chat.Message, options *chat.Options) (*chat.Response, error)

	// GetStreamingResponseFunc is called by GetStreamingResponse(). Returns nil, nil if nil.
	GetStreamingResponseFunc func(ctx context.Context, messages []chat.Message, options *chat.Options) (<-chan chat.ResponseUpdate, error)

	// MetadataFunc is called by Metadata(). Returns zero value if nil.
	MetadataFunc func() chat.ClientMetadata
}

// GetResponse sends messages to the chat model and returns a complete response.
func (m *MockChatClient) GetResponse(ctx context.Context, messages []chat.Message, options *chat.Options) (*chat.Response, error) {
	if m.GetResponseFunc != nil {
		return m.GetResponseFunc(ctx, messages, options)
	}
	return nil, nil
}

// GetStreamingResponse sends messages and returns a channel of incremental updates.
func (m *MockChatClient) GetStreamingResponse(ctx context.Context, messages []chat.Message, options *chat.Options) (<-chan chat.ResponseUpdate, error) {
	if m.GetStreamingResponseFunc != nil {
		return m.GetStreamingResponseFunc(ctx, messages, options)
	}
	return nil, nil
}

// Metadata returns provider-specific metadata about this client.
func (m *MockChatClient) Metadata() chat.ClientMetadata {
	if m.MetadataFunc != nil {
		return m.MetadataFunc()
	}
	return chat.ClientMetadata{}
}

// Verify MockChatClient implements chat.Client interface at compile time.
var _ chat.Client = (*MockChatClient)(nil)

// NewMockChatClient creates a MockChatClient with default no-op behavior.
func NewMockChatClient() *MockChatClient {
	return &MockChatClient{}
}

// WithMetadata configures the MockChatClient to return the specified metadata.
func (m *MockChatClient) WithMetadata(meta chat.ClientMetadata) *MockChatClient {
	m.MetadataFunc = func() chat.ClientMetadata { return meta }
	return m
}

// WithResponse configures the MockChatClient to return the specified response from GetResponse().
func (m *MockChatClient) WithResponse(resp *chat.Response, err error) *MockChatClient {
	m.GetResponseFunc = func(_ context.Context, _ []chat.Message, _ *chat.Options) (*chat.Response, error) {
		return resp, err
	}
	return m
}

// WithStreamingUpdates configures the MockChatClient to return the specified updates from GetStreamingResponse().
func (m *MockChatClient) WithStreamingUpdates(updates []chat.ResponseUpdate, err error) *MockChatClient {
	m.GetStreamingResponseFunc = func(_ context.Context, _ []chat.Message, _ *chat.Options) (<-chan chat.ResponseUpdate, error) {
		if err != nil {
			return nil, err
		}
		ch := make(chan chat.ResponseUpdate, len(updates))
		for _, u := range updates {
			ch <- u
		}
		close(ch)
		return ch, nil
	}
	return m
}

// WithResponseFunc configures the MockChatClient to use a custom response function.
// This is useful for tests that need to validate input parameters.
func (m *MockChatClient) WithResponseFunc(fn func(ctx context.Context, messages []chat.Message, options *chat.Options) (*chat.Response, error)) *MockChatClient {
	m.GetResponseFunc = fn
	return m
}

// WithStreamingFunc configures the MockChatClient to use a custom streaming function.
// This is useful for tests that need to validate input parameters or simulate complex streaming scenarios.
func (m *MockChatClient) WithStreamingFunc(fn func(ctx context.Context, messages []chat.Message, options *chat.Options) (<-chan chat.ResponseUpdate, error)) *MockChatClient {
	m.GetStreamingResponseFunc = fn
	return m
}
