// Copyright (c) Microsoft. All rights reserved.

package observability

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/sdk/trace"
)

func TestDefaultSetupConfig_ReturnsCorrectDefaults(t *testing.T) {
	// Act
	cfg := DefaultSetupConfig("test-service")

	// Assert
	assert.Equal(t, "test-service", cfg.ServiceName)
	assert.True(t, cfg.EnableTracing)
	assert.True(t, cfg.EnableMetrics)
	assert.Empty(t, cfg.ServiceVersion)
	assert.Nil(t, cfg.TraceExporter)
	assert.Nil(t, cfg.MetricExporter)
	assert.Nil(t, cfg.Sampler)
	assert.Nil(t, cfg.Propagators)
}

func TestWithServiceVersion_SetsVersion(t *testing.T) {
	// Arrange
	cfg := DefaultSetupConfig("test-service")

	// Act
	WithServiceVersion("1.0.0")(&cfg)

	// Assert
	assert.Equal(t, "1.0.0", cfg.ServiceVersion)
}

func TestWithSampler_SetsSampler(t *testing.T) {
	// Arrange
	cfg := DefaultSetupConfig("test-service")
	sampler := trace.NeverSample()

	// Act
	WithSampler(sampler)(&cfg)

	// Assert
	assert.NotNil(t, cfg.Sampler)
}

func TestWithTracingEnabled_SetsFlag(t *testing.T) {
	// Arrange
	cfg := DefaultSetupConfig("test-service")

	// Act
	WithTracingEnabled(false)(&cfg)

	// Assert
	assert.False(t, cfg.EnableTracing)
}

func TestWithMetricsEnabled_SetsFlag(t *testing.T) {
	// Arrange
	cfg := DefaultSetupConfig("test-service")

	// Act
	WithMetricsEnabled(false)(&cfg)

	// Assert
	assert.False(t, cfg.EnableMetrics)
}

func TestSetup_WithBothDisabled_Succeeds(t *testing.T) {
	// Arrange
	ctx := context.Background()

	// Act
	shutdown, err := Setup(ctx, "test-service",
		WithTracingEnabled(false),
		WithMetricsEnabled(false),
	)

	// Assert
	require.NoError(t, err)
	require.NotNil(t, shutdown)

	// Clean up
	err = shutdown(ctx)
	assert.NoError(t, err)
}

func TestSetupTracing_ConvenienceFunction(t *testing.T) {
	// This test verifies the convenience function signature is correct
	// Actual setup would require OTLP endpoint, so we skip real testing

	// Verify the function exists and has correct signature
	var fn func(context.Context, string, ...SetupOption) (Shutdown, error) = SetupTracing
	assert.NotNil(t, fn)
}

func TestSetupMetrics_ConvenienceFunction(t *testing.T) {
	// This test verifies the convenience function signature is correct
	// Actual setup would require OTLP endpoint, so we skip real testing

	// Verify the function exists and has correct signature
	var fn func(context.Context, string, ...SetupOption) (Shutdown, error) = SetupMetrics
	assert.NotNil(t, fn)
}

func TestSetup_ConvenienceFunction(t *testing.T) {
	// This test verifies the main function signature is correct
	// Actual setup would require OTLP endpoint, so we skip real testing

	// Verify the function exists and has correct signature
	var fn func(context.Context, string, ...SetupOption) (Shutdown, error) = Setup
	assert.NotNil(t, fn)
}

func TestShutdown_TypeIsFunction(t *testing.T) {
	// Assert
	var shutdown Shutdown = func(ctx context.Context) error {
		return nil
	}
	assert.NotNil(t, shutdown)

	// Verify it can be called
	err := shutdown(context.Background())
	assert.NoError(t, err)
}

func TestSetupOptions_CanBeChained(t *testing.T) {
	// Arrange
	cfg := DefaultSetupConfig("test-service")
	sampler := trace.NeverSample()

	// Act - Apply multiple options
	options := []SetupOption{
		WithServiceVersion("1.0.0"),
		WithSampler(sampler),
		WithTracingEnabled(true),
		WithMetricsEnabled(false),
	}

	for _, opt := range options {
		opt(&cfg)
	}

	// Assert
	assert.Equal(t, "1.0.0", cfg.ServiceVersion)
	assert.NotNil(t, cfg.Sampler)
	assert.True(t, cfg.EnableTracing)
	assert.False(t, cfg.EnableMetrics)
}
