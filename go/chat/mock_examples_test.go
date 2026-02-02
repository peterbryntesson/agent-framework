// Copyright (c) Microsoft. All rights reserved.

package chat_test

import (
	"context"
	"errors"
	"testing"

	"github.com/microsoft/agent-framework-go/chat"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMockChatClient_GetResponse demonstrates table-driven testing for chat responses.
func TestMockChatClient_GetResponse(t *testing.T) {
	testError := errors.New("test error")
	testResponse := &chat.Response{
		Message:      chat.NewAssistantMessage("Hello, world!"),
		FinishReason: chat.FinishReasonStop,
	}

	tests := []struct {
		name       string
		setupMock  func() *MockChatClient
		messages   []chat.Message
		options    *chat.Options
		wantResp   *chat.Response
		wantErr    error
		wantErrMsg string
	}{
		{
			name: "default returns nil",
			setupMock: func() *MockChatClient {
				return NewMockChatClient()
			},
			messages: []chat.Message{chat.NewUserMessage("Hello")},
			options:  nil,
			wantResp: nil,
			wantErr:  nil,
		},
		{
			name: "configured response",
			setupMock: func() *MockChatClient {
				return NewMockChatClient().WithResponse(testResponse, nil)
			},
			messages: []chat.Message{chat.NewUserMessage("Hello")},
			options:  nil,
			wantResp: testResponse,
			wantErr:  nil,
		},
		{
			name: "configured error",
			setupMock: func() *MockChatClient {
				return NewMockChatClient().WithResponse(nil, testError)
			},
			messages:   []chat.Message{chat.NewUserMessage("Hello")},
			options:    nil,
			wantResp:   nil,
			wantErr:    testError,
			wantErrMsg: "test error",
		},
		{
			name: "custom function validates options",
			setupMock: func() *MockChatClient {
				return NewMockChatClient().WithResponseFunc(func(_ context.Context, _ []chat.Message, opts *chat.Options) (*chat.Response, error) {
					if opts != nil && opts.MaxTokens > 0 {
						return testResponse, nil
					}
					return nil, errors.New("max tokens required")
				})
			},
			messages:   []chat.Message{chat.NewUserMessage("Hello")},
			options:    &chat.Options{MaxTokens: 100},
			wantResp:   testResponse,
			wantErr:    nil,
			wantErrMsg: "",
		},
		{
			name: "custom function rejects missing options",
			setupMock: func() *MockChatClient {
				return NewMockChatClient().WithResponseFunc(func(_ context.Context, _ []chat.Message, opts *chat.Options) (*chat.Response, error) {
					if opts != nil && opts.MaxTokens > 0 {
						return testResponse, nil
					}
					return nil, errors.New("max tokens required")
				})
			},
			messages:   []chat.Message{chat.NewUserMessage("Hello")},
			options:    nil,
			wantResp:   nil,
			wantErr:    errors.New("max tokens required"),
			wantErrMsg: "max tokens required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := tt.setupMock()
			ctx := context.Background()

			resp, err := mock.GetResponse(ctx, tt.messages, tt.options)

			if tt.wantErr != nil {
				require.Error(t, err)
				assert.Equal(t, tt.wantErrMsg, err.Error())
			} else {
				require.NoError(t, err)
			}

			if tt.wantResp != nil {
				require.NotNil(t, resp)
				assert.Equal(t, tt.wantResp.FinishReason, resp.FinishReason)
			} else {
				assert.Nil(t, resp)
			}
		})
	}
}

// TestMockChatClient_GetStreamingResponse demonstrates table-driven testing for streaming.
func TestMockChatClient_GetStreamingResponse(t *testing.T) {
	testError := errors.New("stream error")

	tests := []struct {
		name        string
		setupMock   func() *MockChatClient
		wantUpdates []chat.ResponseUpdate
		wantErr     error
	}{
		{
			name: "default returns nil channel",
			setupMock: func() *MockChatClient {
				return NewMockChatClient()
			},
			wantUpdates: nil,
			wantErr:     nil,
		},
		{
			name: "configured stream updates",
			setupMock: func() *MockChatClient {
				return NewMockChatClient().WithStreamingUpdates([]chat.ResponseUpdate{
					{Kind: chat.UpdateKindContentDelta, Delta: &chat.ContentDelta{TextDelta: "Hello"}},
					{Kind: chat.UpdateKindContentDelta, Delta: &chat.ContentDelta{TextDelta: ", world!"}},
					{Kind: chat.UpdateKindDone},
				}, nil)
			},
			wantUpdates: []chat.ResponseUpdate{
				{Kind: chat.UpdateKindContentDelta, Delta: &chat.ContentDelta{TextDelta: "Hello"}},
				{Kind: chat.UpdateKindContentDelta, Delta: &chat.ContentDelta{TextDelta: ", world!"}},
				{Kind: chat.UpdateKindDone},
			},
			wantErr: nil,
		},
		{
			name: "stream error",
			setupMock: func() *MockChatClient {
				return NewMockChatClient().WithStreamingUpdates(nil, testError)
			},
			wantUpdates: nil,
			wantErr:     testError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := tt.setupMock()
			ctx := context.Background()

			ch, err := mock.GetStreamingResponse(ctx, []chat.Message{chat.NewUserMessage("Hello")}, nil)

			if tt.wantErr != nil {
				require.Error(t, err)
				assert.Nil(t, ch)
				return
			}

			require.NoError(t, err)

			if tt.wantUpdates == nil {
				assert.Nil(t, ch)
				return
			}

			require.NotNil(t, ch)

			var gotUpdates []chat.ResponseUpdate
			for update := range ch {
				gotUpdates = append(gotUpdates, update)
			}

			assert.Equal(t, len(tt.wantUpdates), len(gotUpdates))
			for i, want := range tt.wantUpdates {
				assert.Equal(t, want.Kind, gotUpdates[i].Kind)
			}
		})
	}
}

// TestMockChatClient_Metadata demonstrates table-driven testing for client metadata.
func TestMockChatClient_Metadata(t *testing.T) {
	tests := []struct {
		name      string
		setupMock func() *MockChatClient
		want      chat.ClientMetadata
	}{
		{
			name: "default metadata is empty",
			setupMock: func() *MockChatClient {
				return NewMockChatClient()
			},
			want: chat.ClientMetadata{},
		},
		{
			name: "configured metadata",
			setupMock: func() *MockChatClient {
				return NewMockChatClient().WithMetadata(chat.ClientMetadata{
					ProviderName: "openai",
					ModelID:      "gpt-4",
					EndpointURI:  "https://api.openai.com",
				})
			},
			want: chat.ClientMetadata{
				ProviderName: "openai",
				ModelID:      "gpt-4",
				EndpointURI:  "https://api.openai.com",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := tt.setupMock()

			got := mock.Metadata()

			assert.Equal(t, tt.want.ProviderName, got.ProviderName)
			assert.Equal(t, tt.want.ModelID, got.ModelID)
			assert.Equal(t, tt.want.EndpointURI, got.EndpointURI)
		})
	}
}

// TestMockChatClient_ImplementsInterface verifies the mock implements the Client interface.
func TestMockChatClient_ImplementsInterface(t *testing.T) {
	var c chat.Client = NewMockChatClient()
	assert.NotNil(t, c)
}
