// Copyright (c) Microsoft. All rights reserved.

package observability

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

// Metrics holds the OpenTelemetry metric instruments for observability.
type Metrics struct {
	// AgentRuns counts the number of agent runs.
	AgentRuns metric.Int64Counter

	// TokensInput is a histogram of input tokens per request.
	TokensInput metric.Int64Histogram

	// TokensOutput is a histogram of output tokens per request.
	TokensOutput metric.Int64Histogram

	// RequestLatency is a histogram of request latency in seconds.
	RequestLatency metric.Float64Histogram

	// Errors counts the number of errors.
	Errors metric.Int64Counter

	// ToolInvocations counts tool invocations.
	ToolInvocations metric.Int64Counter
}

// NewMetrics creates the metrics instruments.
// Returns an error if any instrument creation fails.
func NewMetrics() (*Metrics, error) {
	meter := otel.Meter(InstrumentationName)

	agentRuns, err := meter.Int64Counter(
		MetricAgentRuns,
		metric.WithDescription("Number of agent runs"),
		metric.WithUnit("{run}"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create agent runs counter: %w", err)
	}

	tokensInput, err := meter.Int64Histogram(
		MetricTokensInput,
		metric.WithDescription("Input tokens per request"),
		metric.WithUnit("{token}"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create input tokens histogram: %w", err)
	}

	tokensOutput, err := meter.Int64Histogram(
		MetricTokensOutput,
		metric.WithDescription("Output tokens per request"),
		metric.WithUnit("{token}"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create output tokens histogram: %w", err)
	}

	requestLatency, err := meter.Float64Histogram(
		MetricRequestLatency,
		metric.WithDescription("Request latency in seconds"),
		metric.WithUnit("s"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create request latency histogram: %w", err)
	}

	errors, err := meter.Int64Counter(
		MetricErrors,
		metric.WithDescription("Number of errors"),
		metric.WithUnit("{error}"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create errors counter: %w", err)
	}

	toolInvocations, err := meter.Int64Counter(
		MetricToolInvocations,
		metric.WithDescription("Number of tool invocations"),
		metric.WithUnit("{invocation}"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create tool invocations counter: %w", err)
	}

	return &Metrics{
		AgentRuns:       agentRuns,
		TokensInput:     tokensInput,
		TokensOutput:    tokensOutput,
		RequestLatency:  requestLatency,
		Errors:          errors,
		ToolInvocations: toolInvocations,
	}, nil
}

// RecordAgentRun records a single agent run with the given attributes.
func (m *Metrics) RecordAgentRun(ctx context.Context, providerName, modelID string) {
	m.AgentRuns.Add(ctx, 1,
		metric.WithAttributes(
			attribute.String(GenAIProviderNameKey, providerName),
			attribute.String(GenAIRequestModelKey, modelID),
		),
	)
}

// RecordTokenUsage records token usage metrics.
func (m *Metrics) RecordTokenUsage(ctx context.Context, inputTokens, outputTokens int, providerName, modelID string) {
	attrs := metric.WithAttributes(
		attribute.String(GenAIProviderNameKey, providerName),
		attribute.String(GenAIRequestModelKey, modelID),
	)

	if inputTokens > 0 {
		m.TokensInput.Record(ctx, int64(inputTokens), attrs)
	}

	if outputTokens > 0 {
		m.TokensOutput.Record(ctx, int64(outputTokens), attrs)
	}
}

// RecordLatency records request latency in seconds.
func (m *Metrics) RecordLatency(ctx context.Context, latencySeconds float64, providerName, modelID string) {
	m.RequestLatency.Record(ctx, latencySeconds,
		metric.WithAttributes(
			attribute.String(GenAIProviderNameKey, providerName),
			attribute.String(GenAIRequestModelKey, modelID),
		),
	)
}

// RecordError records an error occurrence.
func (m *Metrics) RecordError(ctx context.Context, errorType, providerName, modelID string) {
	m.Errors.Add(ctx, 1,
		metric.WithAttributes(
			attribute.String(GenAIErrorTypeKey, errorType),
			attribute.String(GenAIProviderNameKey, providerName),
			attribute.String(GenAIRequestModelKey, modelID),
		),
	)
}

// RecordToolInvocation records a tool invocation.
func (m *Metrics) RecordToolInvocation(ctx context.Context, toolName string, success bool) {
	successAttr := "true"
	if !success {
		successAttr = "false"
	}

	m.ToolInvocations.Add(ctx, 1,
		metric.WithAttributes(
			attribute.String(GenAIToolNameKey, toolName),
			attribute.String("success", successAttr),
		),
	)
}

// defaultMetrics is a lazily initialized singleton for convenience.
var defaultMetrics *Metrics

// DefaultMetrics returns a shared Metrics instance.
// Creates the instance on first call. Returns nil if creation fails.
func DefaultMetrics() *Metrics {
	if defaultMetrics == nil {
		var err error
		defaultMetrics, err = NewMetrics()
		if err != nil {
			// In production, metrics creation failure should not crash the application
			return nil
		}
	}
	return defaultMetrics
}
