// Copyright (c) Microsoft. All rights reserved.

package observability

import (
	"context"
	"fmt"
	"time"

	"github.com/microsoft/agent-framework-go/agent"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// TelemetryMiddleware provides OpenTelemetry instrumentation for agent invocations.
// It creates spans and records metrics for each agent run, integrating with
// the existing observability infrastructure.
type TelemetryMiddleware struct {
	metrics             *Metrics
	enableSensitiveData bool
	sourceName          string
}

// TelemetryOption configures TelemetryMiddleware behavior.
type TelemetryOption func(*TelemetryMiddleware)

// WithTelemetryMetrics configures the metrics instance to use.
// If not set, DefaultMetrics() is used.
func WithTelemetryMetrics(m *Metrics) TelemetryOption {
	return func(tm *TelemetryMiddleware) {
		tm.metrics = m
	}
}

// WithTelemetrySensitiveData enables recording of message content in traces.
// Default is false for security.
func WithTelemetrySensitiveData(enabled bool) TelemetryOption {
	return func(tm *TelemetryMiddleware) {
		tm.enableSensitiveData = enabled
	}
}

// WithSourceName sets a custom source name for tracing attribution.
func WithSourceName(name string) TelemetryOption {
	return func(tm *TelemetryMiddleware) {
		tm.sourceName = name
	}
}

// NewTelemetryMiddleware creates a new telemetry middleware with the provided options.
func NewTelemetryMiddleware(opts ...TelemetryOption) *TelemetryMiddleware {
	tm := &TelemetryMiddleware{
		enableSensitiveData: false,
		sourceName:          InstrumentationName,
	}

	for _, opt := range opts {
		opt(tm)
	}

	// Use default metrics if not provided
	if tm.metrics == nil {
		tm.metrics = DefaultMetrics()
	}

	return tm
}

// Ensure TelemetryMiddleware implements agent.AgentMiddleware.
var _ agent.AgentMiddleware = (*TelemetryMiddleware)(nil)

// Process implements agent.AgentMiddleware.
// It wraps agent invocations with OpenTelemetry spans and metrics.
func (m *TelemetryMiddleware) Process(ctx context.Context, agentCtx *agent.AgentContext, next agent.AgentHandler) error {
	// Determine operation name based on streaming
	operationName := OperationAgentRun
	if agentCtx.IsStreaming {
		operationName = OperationAgentRunStream
	}

	// Get agent metadata for attributes
	agentName := ""
	agentID := ""
	providerName := ""
	if agentCtx.Agent != nil {
		agentName = agentCtx.Agent.Name()
		agentID = agentCtx.Agent.ID()
		metadata := agentCtx.Agent.Metadata()
		providerName = metadata.ProviderName
	}

	// Try to get model ID from context metadata
	modelID := ""
	if agentCtx.Metadata != nil {
		if id, ok := agentCtx.Metadata["model_id"].(string); ok {
			modelID = id
		}
	}

	// Start span
	ctx, span := StartAgentSpan(ctx, operationName, agentName)
	defer span.End()

	// Record agent attributes
	if agentCtx.Agent != nil {
		span.SetAttributes(
			attribute.String(AgentIDKey, agentID),
			attribute.String(GenAISystemKey, providerName),
		)
		if modelID != "" {
			span.SetAttributes(attribute.String(GenAIRequestModelKey, modelID))
		}
	}

	// Record agent run metric
	if m.metrics != nil {
		m.metrics.RecordAgentRun(ctx, providerName, modelID)
	}

	start := time.Now()

	// Call next handler
	err := next(ctx, agentCtx)

	latency := time.Since(start).Seconds()

	// Record latency metric
	if m.metrics != nil {
		m.metrics.RecordLatency(ctx, latency, providerName, modelID)
	}

	if err != nil {
		RecordError(span, err)
		if m.metrics != nil {
			m.metrics.RecordError(ctx, agentErrorTypeName(err), providerName, modelID)
		}
		return err
	}

	// Record response details
	m.recordResponseDetails(ctx, span, agentCtx, providerName, modelID)

	return nil
}

// recordResponseDetails records span attributes from the agent response.
func (m *TelemetryMiddleware) recordResponseDetails(ctx context.Context, span trace.Span, agentCtx *agent.AgentContext, providerName, modelID string) {
	if agentCtx.Response == nil {
		return
	}

	resp := agentCtx.Response

	// Record finish reason
	finishReasonStr := agentFinishReasonToString(resp.FinishReason)
	if finishReasonStr != "" {
		span.SetAttributes(
			attribute.String(GenAIResponseFinishReasonsKey, finishReasonStr),
		)
	}

	// Record usage if available
	if resp.Usage != nil {
		RecordUsage(span, resp.Usage.InputTokens, resp.Usage.OutputTokens)

		if m.metrics != nil {
			m.metrics.RecordTokenUsage(ctx, resp.Usage.InputTokens, resp.Usage.OutputTokens, providerName, modelID)
		}
	}
}

// agentFinishReasonToString converts an agent.FinishReason to its string representation.
func agentFinishReasonToString(reason agent.FinishReason) string {
	switch reason {
	case agent.FinishReasonStop:
		return "stop"
	case agent.FinishReasonLength:
		return "length"
	case agent.FinishReasonToolCalls:
		return "tool_calls"
	case agent.FinishReasonContentFilter:
		return "content_filter"
	default:
		return ""
	}
}

// agentErrorTypeName extracts the type name from an error for metrics.
func agentErrorTypeName(err error) string {
	if err == nil {
		return ""
	}
	return fmt.Sprintf("%T", err)
}
