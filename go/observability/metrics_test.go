// Copyright (c) Microsoft. All rights reserved.

package observability

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewMetrics_CreatesAllInstruments(t *testing.T) {
	// Act
	metrics, err := NewMetrics()

	// Assert
	require.NoError(t, err)
	require.NotNil(t, metrics)
	assert.NotNil(t, metrics.AgentRuns)
	assert.NotNil(t, metrics.TokensInput)
	assert.NotNil(t, metrics.TokensOutput)
	assert.NotNil(t, metrics.RequestLatency)
	assert.NotNil(t, metrics.Errors)
	assert.NotNil(t, metrics.ToolInvocations)
}

func TestMetrics_RecordAgentRun(t *testing.T) {
	// Arrange
	metrics, err := NewMetrics()
	require.NoError(t, err)
	ctx := context.Background()

	// Act - should not panic
	metrics.RecordAgentRun(ctx, "openai", "gpt-4")
}

func TestMetrics_RecordTokenUsage(t *testing.T) {
	// Arrange
	metrics, err := NewMetrics()
	require.NoError(t, err)
	ctx := context.Background()

	// Act - should not panic
	metrics.RecordTokenUsage(ctx, 100, 50, "openai", "gpt-4")
}

func TestMetrics_RecordTokenUsage_WithZeroValues(t *testing.T) {
	// Arrange
	metrics, err := NewMetrics()
	require.NoError(t, err)
	ctx := context.Background()

	// Act - should not record zero values
	metrics.RecordTokenUsage(ctx, 0, 0, "openai", "gpt-4")
}

func TestMetrics_RecordLatency(t *testing.T) {
	// Arrange
	metrics, err := NewMetrics()
	require.NoError(t, err)
	ctx := context.Background()

	// Act - should not panic
	metrics.RecordLatency(ctx, 1.5, "openai", "gpt-4")
}

func TestMetrics_RecordError(t *testing.T) {
	// Arrange
	metrics, err := NewMetrics()
	require.NoError(t, err)
	ctx := context.Background()

	// Act - should not panic
	metrics.RecordError(ctx, "timeout", "openai", "gpt-4")
}

func TestMetrics_RecordToolInvocation_Success(t *testing.T) {
	// Arrange
	metrics, err := NewMetrics()
	require.NoError(t, err)
	ctx := context.Background()

	// Act - should not panic
	metrics.RecordToolInvocation(ctx, "get_weather", true)
}

func TestMetrics_RecordToolInvocation_Failure(t *testing.T) {
	// Arrange
	metrics, err := NewMetrics()
	require.NoError(t, err)
	ctx := context.Background()

	// Act - should not panic
	metrics.RecordToolInvocation(ctx, "get_weather", false)
}

func TestDefaultMetrics_ReturnsSameInstance(t *testing.T) {
	// Reset for test isolation
	defaultMetrics = nil

	// Act
	m1 := DefaultMetrics()
	m2 := DefaultMetrics()

	// Assert
	assert.NotNil(t, m1)
	assert.Same(t, m1, m2)
}

func TestDefaultMetrics_CreatesMetrics(t *testing.T) {
	// Reset for test isolation
	defaultMetrics = nil

	// Act
	m := DefaultMetrics()

	// Assert
	assert.NotNil(t, m)
	assert.NotNil(t, m.AgentRuns)
	assert.NotNil(t, m.TokensInput)
	assert.NotNil(t, m.TokensOutput)
	assert.NotNil(t, m.RequestLatency)
	assert.NotNil(t, m.Errors)
	assert.NotNil(t, m.ToolInvocations)
}

func TestMetrics_MultipleRecordings(t *testing.T) {
	// Arrange
	metrics, err := NewMetrics()
	require.NoError(t, err)
	ctx := context.Background()

	// Act - Record multiple values
	for i := 0; i < 10; i++ {
		metrics.RecordAgentRun(ctx, "openai", "gpt-4")
		metrics.RecordTokenUsage(ctx, 100+i*10, 50+i*5, "openai", "gpt-4")
		metrics.RecordLatency(ctx, float64(i)*0.1, "openai", "gpt-4")
	}

	// No assertions needed - just verify no panics
}

func TestMetrics_DifferentProviders(t *testing.T) {
	// Arrange
	metrics, err := NewMetrics()
	require.NoError(t, err)
	ctx := context.Background()

	providers := []struct {
		name    string
		modelID string
	}{
		{"openai", "gpt-4"},
		{"anthropic", "claude-3"},
		{"azure", "gpt-4-deployment"},
	}

	// Act - Record for different providers
	for _, p := range providers {
		metrics.RecordAgentRun(ctx, p.name, p.modelID)
		metrics.RecordTokenUsage(ctx, 100, 50, p.name, p.modelID)
		metrics.RecordLatency(ctx, 1.0, p.name, p.modelID)
	}

	// No assertions needed - just verify no panics
}
