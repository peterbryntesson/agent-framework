// Copyright (c) Microsoft. All rights reserved.

package observability

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"testing"

	"github.com/microsoft/agent-framework-go/agent"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockAgent implements agent.Agent for testing.
type mockAgent struct {
	id          string
	name        string
	description string
	metadata    agent.AIAgentMetadata
}

func (m *mockAgent) ID() string                      { return m.id }
func (m *mockAgent) Name() string                    { return m.name }
func (m *mockAgent) Description() string             { return m.description }
func (m *mockAgent) Metadata() agent.AIAgentMetadata { return m.metadata }

// Stub the remaining interface methods.
func (m *mockAgent) Run(_ context.Context, _ []agent.Message, _ ...agent.RunOption) (*agent.Response, error) {
	return nil, nil
}

func (m *mockAgent) RunStream(_ context.Context, _ []agent.Message, _ ...agent.RunOption) (<-chan agent.ResponseUpdate, error) {
	return nil, nil
}

func (m *mockAgent) NewSession(_ context.Context) (agent.Session, error) {
	return nil, nil
}

func (m *mockAgent) RestoreSession(_ context.Context, _ json.RawMessage) (agent.Session, error) {
	return nil, nil
}

func (m *mockAgent) GetService(_ reflect.Type) interface{} {
	return nil
}

func TestNewTelemetryMiddleware_DefaultsAreSet(t *testing.T) {
	// Act
	mw := NewTelemetryMiddleware()

	// Assert
	assert.NotNil(t, mw)
	assert.False(t, mw.enableSensitiveData)
	assert.Equal(t, InstrumentationName, mw.sourceName)
}

func TestNewTelemetryMiddleware_WithOptions(t *testing.T) {
	// Arrange
	metrics, err := NewMetrics()
	require.NoError(t, err)

	// Act
	mw := NewTelemetryMiddleware(
		WithTelemetryMetrics(metrics),
		WithTelemetrySensitiveData(true),
		WithSourceName("custom-source"),
	)

	// Assert
	assert.Equal(t, metrics, mw.metrics)
	assert.True(t, mw.enableSensitiveData)
	assert.Equal(t, "custom-source", mw.sourceName)
}

func TestTelemetryMiddleware_Process_CallsNextHandler(t *testing.T) {
	// Arrange
	mw := NewTelemetryMiddleware()
	called := false

	agentCtx := &agent.AgentContext{
		Agent: &mockAgent{
			id:   "test-id",
			name: "test-agent",
			metadata: agent.AIAgentMetadata{
				ProviderName: "openai",
			},
		},
		Metadata: map[string]any{"model_id": "gpt-4"},
	}

	next := func(ctx context.Context, ac *agent.AgentContext) error {
		called = true
		return nil
	}

	// Act
	err := mw.Process(context.Background(), agentCtx, next)

	// Assert
	assert.NoError(t, err)
	assert.True(t, called)
}

func TestTelemetryMiddleware_Process_RecordsErrorOnFailure(t *testing.T) {
	// Arrange
	mw := NewTelemetryMiddleware()
	expectedErr := errors.New("test error")

	agentCtx := &agent.AgentContext{
		Agent: &mockAgent{
			id:   "test-id",
			name: "test-agent",
		},
	}

	next := func(ctx context.Context, ac *agent.AgentContext) error {
		return expectedErr
	}

	// Act
	err := mw.Process(context.Background(), agentCtx, next)

	// Assert
	assert.Equal(t, expectedErr, err)
}

func TestTelemetryMiddleware_Process_StreamingOperation(t *testing.T) {
	// Arrange
	mw := NewTelemetryMiddleware()

	agentCtx := &agent.AgentContext{
		Agent:       &mockAgent{id: "test-id", name: "test-agent"},
		IsStreaming: true,
	}

	next := func(ctx context.Context, ac *agent.AgentContext) error {
		return nil
	}

	// Act
	err := mw.Process(context.Background(), agentCtx, next)

	// Assert
	assert.NoError(t, err)
}

func TestTelemetryMiddleware_Process_NilAgent(t *testing.T) {
	// Arrange
	mw := NewTelemetryMiddleware()

	agentCtx := &agent.AgentContext{
		Agent: nil,
	}

	next := func(ctx context.Context, ac *agent.AgentContext) error {
		return nil
	}

	// Act
	err := mw.Process(context.Background(), agentCtx, next)

	// Assert
	assert.NoError(t, err)
}

func TestTelemetryMiddleware_Process_RecordsResponseDetails(t *testing.T) {
	// Arrange
	mw := NewTelemetryMiddleware()

	agentCtx := &agent.AgentContext{
		Agent: &mockAgent{id: "test-id", name: "test-agent"},
	}

	next := func(ctx context.Context, ac *agent.AgentContext) error {
		ac.Response = &agent.Response{
			FinishReason: agent.FinishReasonStop,
			Usage: &agent.UsageDetails{
				InputTokens:  100,
				OutputTokens: 50,
			},
		}
		return nil
	}

	// Act
	err := mw.Process(context.Background(), agentCtx, next)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, agentCtx.Response)
}

func TestAgentErrorTypeName_ReturnsTypeName(t *testing.T) {
	// Arrange
	testErr := errors.New("test error")

	// Act
	typeName := agentErrorTypeName(testErr)

	// Assert
	assert.Equal(t, "*errors.errorString", typeName)
}

func TestAgentErrorTypeName_NilError(t *testing.T) {
	// Act
	typeName := agentErrorTypeName(nil)

	// Assert
	assert.Equal(t, "", typeName)
}
