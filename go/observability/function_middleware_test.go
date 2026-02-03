// Copyright (c) Microsoft. All rights reserved.

package observability

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/microsoft/agent-framework-go/agent"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewFunctionTelemetryMiddleware_DefaultsAreSet(t *testing.T) {
	// Act
	mw := NewFunctionTelemetryMiddleware()

	// Assert
	assert.NotNil(t, mw)
	assert.False(t, mw.enableSensitiveData)
}

func TestNewFunctionTelemetryMiddleware_WithOptions(t *testing.T) {
	// Arrange
	metrics, err := NewMetrics()
	require.NoError(t, err)

	// Act
	mw := NewFunctionTelemetryMiddleware(
		WithFunctionMetrics(metrics),
		WithFunctionSensitiveData(true),
	)

	// Assert
	assert.Equal(t, metrics, mw.metrics)
	assert.True(t, mw.enableSensitiveData)
}

func TestFunctionTelemetryMiddleware_Process_CallsNextHandler(t *testing.T) {
	// Arrange
	mw := NewFunctionTelemetryMiddleware()
	called := false

	funcCtx := &agent.FunctionContext{
		FunctionName: "get_weather",
		Arguments:    json.RawMessage(`{"location": "Seattle"}`),
	}

	next := func(ctx context.Context, fc *agent.FunctionContext) error {
		called = true
		fc.Result = map[string]string{"temp": "72F"}
		return nil
	}

	// Act
	err := mw.Process(context.Background(), funcCtx, next)

	// Assert
	assert.NoError(t, err)
	assert.True(t, called)
}

func TestFunctionTelemetryMiddleware_Process_RecordsError(t *testing.T) {
	// Arrange
	mw := NewFunctionTelemetryMiddleware()
	expectedErr := errors.New("function failed")

	funcCtx := &agent.FunctionContext{
		FunctionName: "get_weather",
	}

	next := func(ctx context.Context, fc *agent.FunctionContext) error {
		return expectedErr
	}

	// Act
	err := mw.Process(context.Background(), funcCtx, next)

	// Assert
	assert.Equal(t, expectedErr, err)
}

func TestFunctionTelemetryMiddleware_Process_RecordsFunctionContextError(t *testing.T) {
	// Arrange
	mw := NewFunctionTelemetryMiddleware()

	funcCtx := &agent.FunctionContext{
		FunctionName: "get_weather",
	}

	next := func(ctx context.Context, fc *agent.FunctionContext) error {
		fc.Error = errors.New("context error")
		return nil
	}

	// Act
	err := mw.Process(context.Background(), funcCtx, next)

	// Assert
	assert.NoError(t, err) // Handler returned nil, error is in context
}

func TestFunctionTelemetryMiddleware_Process_WithCallID(t *testing.T) {
	// Arrange
	mw := NewFunctionTelemetryMiddleware()

	funcCtx := &agent.FunctionContext{
		FunctionName: "get_weather",
		Metadata:     map[string]any{"call_id": "call-123"},
	}

	next := func(ctx context.Context, fc *agent.FunctionContext) error {
		return nil
	}

	// Act
	err := mw.Process(context.Background(), funcCtx, next)

	// Assert
	assert.NoError(t, err)
}

func TestFunctionTelemetryMiddleware_Process_SensitiveDataEnabled(t *testing.T) {
	// Arrange
	mw := NewFunctionTelemetryMiddleware(WithFunctionSensitiveData(true))

	funcCtx := &agent.FunctionContext{
		FunctionName: "get_weather",
		Arguments:    json.RawMessage(`{"location": "Seattle"}`),
	}

	next := func(ctx context.Context, fc *agent.FunctionContext) error {
		return nil
	}

	// Act
	err := mw.Process(context.Background(), funcCtx, next)

	// Assert
	assert.NoError(t, err)
}

func TestFunctionTelemetryMiddleware_Process_EmptyArguments(t *testing.T) {
	// Arrange
	mw := NewFunctionTelemetryMiddleware(WithFunctionSensitiveData(true))

	funcCtx := &agent.FunctionContext{
		FunctionName: "ping",
		Arguments:    nil,
	}

	next := func(ctx context.Context, fc *agent.FunctionContext) error {
		return nil
	}

	// Act
	err := mw.Process(context.Background(), funcCtx, next)

	// Assert
	assert.NoError(t, err)
}
