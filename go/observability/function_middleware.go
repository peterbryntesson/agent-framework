// Copyright (c) Microsoft. All rights reserved.

package observability

import (
	"context"
	"time"

	"github.com/microsoft/agent-framework-go/agent"

	"go.opentelemetry.io/otel/attribute"
)

// FunctionTelemetryMiddleware provides OpenTelemetry instrumentation for tool/function invocations.
type FunctionTelemetryMiddleware struct {
	metrics             *Metrics
	enableSensitiveData bool
}

// FunctionTelemetryOption configures FunctionTelemetryMiddleware behavior.
type FunctionTelemetryOption func(*FunctionTelemetryMiddleware)

// WithFunctionMetrics configures the metrics instance for function middleware.
func WithFunctionMetrics(m *Metrics) FunctionTelemetryOption {
	return func(fm *FunctionTelemetryMiddleware) {
		fm.metrics = m
	}
}

// WithFunctionSensitiveData enables recording of function arguments in traces.
func WithFunctionSensitiveData(enabled bool) FunctionTelemetryOption {
	return func(fm *FunctionTelemetryMiddleware) {
		fm.enableSensitiveData = enabled
	}
}

// NewFunctionTelemetryMiddleware creates a new function telemetry middleware.
func NewFunctionTelemetryMiddleware(opts ...FunctionTelemetryOption) *FunctionTelemetryMiddleware {
	fm := &FunctionTelemetryMiddleware{
		enableSensitiveData: false,
	}

	for _, opt := range opts {
		opt(fm)
	}

	if fm.metrics == nil {
		fm.metrics = DefaultMetrics()
	}

	return fm
}

// Ensure FunctionTelemetryMiddleware implements agent.FunctionMiddleware.
var _ agent.FunctionMiddleware = (*FunctionTelemetryMiddleware)(nil)

// Process implements agent.FunctionMiddleware.
func (m *FunctionTelemetryMiddleware) Process(ctx context.Context, funcCtx *agent.FunctionContext, next agent.FunctionHandler) error {
	// Get call ID from metadata if available
	callID := ""
	if funcCtx.Metadata != nil {
		if id, ok := funcCtx.Metadata["call_id"].(string); ok {
			callID = id
		}
	}

	// Start tool span
	ctx, span := StartToolSpan(ctx, funcCtx.FunctionName, callID)
	defer span.End()

	// Record function name
	span.SetAttributes(
		attribute.String(GenAIToolNameKey, funcCtx.FunctionName),
	)

	// Optionally record arguments if sensitive data is enabled
	if m.enableSensitiveData && len(funcCtx.Arguments) > 0 {
		span.SetAttributes(
			attribute.String("gen_ai.tool.arguments", string(funcCtx.Arguments)),
		)
	}

	start := time.Now()

	// Call next handler
	err := next(ctx, funcCtx)

	_ = time.Since(start).Seconds() // latency captured for potential future use

	// Record metrics
	success := err == nil && funcCtx.Error == nil
	if m.metrics != nil {
		m.metrics.RecordToolInvocation(ctx, funcCtx.FunctionName, success)
	}

	// Record error if present
	if err != nil {
		RecordError(span, err)
		return err
	}

	if funcCtx.Error != nil {
		RecordError(span, funcCtx.Error)
	}

	return nil
}
