// Copyright (c) Microsoft. All rights reserved.

package agent_test

import (
	"context"
	"errors"
	"testing"

	"github.com/microsoft/agent-framework-go/agent"
	"github.com/microsoft/agent-framework-go/chat"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMockAgent_Identity demonstrates table-driven testing patterns for agent identity methods.
func TestMockAgent_Identity(t *testing.T) {
	tests := []struct {
		name            string
		setupMock       func() *MockAgent
		wantID          string
		wantName        string
		wantDescription string
	}{
		{
			name: "default values are empty",
			setupMock: func() *MockAgent {
				return NewMockAgent()
			},
			wantID:          "",
			wantName:        "",
			wantDescription: "",
		},
		{
			name: "configured identity values",
			setupMock: func() *MockAgent {
				return NewMockAgent().
					WithID("agent-123").
					WithName("Test Agent").
					WithDescription("A test agent for unit tests")
			},
			wantID:          "agent-123",
			wantName:        "Test Agent",
			wantDescription: "A test agent for unit tests",
		},
		{
			name: "custom function overrides",
			setupMock: func() *MockAgent {
				m := NewMockAgent()
				m.IDFunc = func() string { return "custom-id" }
				return m
			},
			wantID:          "custom-id",
			wantName:        "",
			wantDescription: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := tt.setupMock()

			assert.Equal(t, tt.wantID, mock.ID())
			assert.Equal(t, tt.wantName, mock.Name())
			assert.Equal(t, tt.wantDescription, mock.Description())
		})
	}
}

// TestMockAgent_Run demonstrates table-driven testing for agent execution.
func TestMockAgent_Run(t *testing.T) {
	testError := errors.New("test error")
	testResponse := &agent.Response{
		Messages: []agent.Message{
			chat.NewAssistantMessage("Hello, world!"),
		},
		FinishReason: agent.FinishReasonStop,
	}

	tests := []struct {
		name       string
		setupMock  func() *MockAgent
		messages   []agent.Message
		wantResp   *agent.Response
		wantErr    error
		wantErrMsg string
	}{
		{
			name: "default returns nil",
			setupMock: func() *MockAgent {
				return NewMockAgent()
			},
			messages: []agent.Message{chat.NewUserMessage("Hello")},
			wantResp: nil,
			wantErr:  nil,
		},
		{
			name: "configured response",
			setupMock: func() *MockAgent {
				return NewMockAgent().WithResponse(testResponse, nil)
			},
			messages: []agent.Message{chat.NewUserMessage("Hello")},
			wantResp: testResponse,
			wantErr:  nil,
		},
		{
			name: "configured error",
			setupMock: func() *MockAgent {
				return NewMockAgent().WithResponse(nil, testError)
			},
			messages:   []agent.Message{chat.NewUserMessage("Hello")},
			wantResp:   nil,
			wantErr:    testError,
			wantErrMsg: "test error",
		},
		{
			name: "custom function captures input",
			setupMock: func() *MockAgent {
				m := NewMockAgent()
				m.RunFunc = func(_ context.Context, messages []agent.Message, _ ...agent.RunOption) (*agent.Response, error) {
					return &agent.Response{
						Messages: []agent.Message{
							chat.NewAssistantMessage("Received: " + messages[0].Text()),
						},
					}, nil
				}
				return m
			},
			messages: []agent.Message{chat.NewUserMessage("Test input")},
			wantResp: &agent.Response{
				Messages: []agent.Message{
					chat.NewAssistantMessage("Received: Test input"),
				},
			},
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := tt.setupMock()
			ctx := context.Background()

			resp, err := mock.Run(ctx, tt.messages)

			if tt.wantErr != nil {
				require.Error(t, err)
				assert.Equal(t, tt.wantErrMsg, err.Error())
			} else {
				require.NoError(t, err)
			}

			if tt.wantResp != nil {
				require.NotNil(t, resp)
				assert.Equal(t, len(tt.wantResp.Messages), len(resp.Messages))
				for i, msg := range tt.wantResp.Messages {
					assert.Equal(t, msg.Text(), resp.Messages[i].Text())
				}
			} else {
				assert.Nil(t, resp)
			}
		})
	}
}

// TestMockAgent_RunStream demonstrates table-driven testing for streaming responses.
func TestMockAgent_RunStream(t *testing.T) {
	testError := errors.New("stream error")

	tests := []struct {
		name        string
		setupMock   func() *MockAgent
		wantUpdates []agent.ResponseUpdate
		wantErr     error
	}{
		{
			name: "default returns nil channel",
			setupMock: func() *MockAgent {
				return NewMockAgent()
			},
			wantUpdates: nil,
			wantErr:     nil,
		},
		{
			name: "configured stream updates",
			setupMock: func() *MockAgent {
				return NewMockAgent().WithStreamUpdates([]agent.ResponseUpdate{
					{Kind: agent.UpdateKindContentDelta, Delta: &agent.ContentDelta{TextDelta: "Hello"}},
					{Kind: agent.UpdateKindContentDelta, Delta: &agent.ContentDelta{TextDelta: ", world!"}},
					{Kind: agent.UpdateKindDone},
				}, nil)
			},
			wantUpdates: []agent.ResponseUpdate{
				{Kind: agent.UpdateKindContentDelta, Delta: &agent.ContentDelta{TextDelta: "Hello"}},
				{Kind: agent.UpdateKindContentDelta, Delta: &agent.ContentDelta{TextDelta: ", world!"}},
				{Kind: agent.UpdateKindDone},
			},
			wantErr: nil,
		},
		{
			name: "stream error",
			setupMock: func() *MockAgent {
				return NewMockAgent().WithStreamUpdates(nil, testError)
			},
			wantUpdates: nil,
			wantErr:     testError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := tt.setupMock()
			ctx := context.Background()

			ch, err := mock.RunStream(ctx, []agent.Message{chat.NewUserMessage("Hello")})

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

			var gotUpdates []agent.ResponseUpdate
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

// TestMockAgent_Session demonstrates table-driven testing for session management.
func TestMockAgent_Session(t *testing.T) {
	testSession := agent.NewInMemorySession()
	testError := errors.New("session error")

	tests := []struct {
		name        string
		setupMock   func() *MockAgent
		wantSession agent.Session
		wantErr     error
	}{
		{
			name: "default returns nil",
			setupMock: func() *MockAgent {
				return NewMockAgent()
			},
			wantSession: nil,
			wantErr:     nil,
		},
		{
			name: "configured session",
			setupMock: func() *MockAgent {
				return NewMockAgent().WithSession(testSession, nil)
			},
			wantSession: testSession,
			wantErr:     nil,
		},
		{
			name: "session error",
			setupMock: func() *MockAgent {
				return NewMockAgent().WithSession(nil, testError)
			},
			wantSession: nil,
			wantErr:     testError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := tt.setupMock()
			ctx := context.Background()

			session, err := mock.NewSession(ctx)

			if tt.wantErr != nil {
				require.Error(t, err)
				assert.Nil(t, session)
				return
			}

			require.NoError(t, err)

			if tt.wantSession != nil {
				require.NotNil(t, session)
				assert.Equal(t, tt.wantSession.ID(), session.ID())
			} else {
				assert.Nil(t, session)
			}
		})
	}
}

// TestMockAgent_ImplementsInterface verifies the mock implements the Agent interface.
func TestMockAgent_ImplementsInterface(t *testing.T) {
	var a agent.Agent = NewMockAgent()
	assert.NotNil(t, a)
}
