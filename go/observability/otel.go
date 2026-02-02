// Copyright (c) Microsoft. All rights reserved.

// Package observability provides OpenTelemetry integration for the Agent Framework.
//
// This package enables tracing, metrics, and logging for agent operations
// using the OpenTelemetry standard. It follows the Semantic Conventions for
// Generative AI Systems as defined by OpenTelemetry.
//
// See: https://opentelemetry.io/docs/specs/semconv/gen-ai/
package observability

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

// InstrumentationName is the name used for OpenTelemetry instrumentation.
const InstrumentationName = "github.com/microsoft/agent-framework-go"

// Semantic convention attribute keys for GenAI operations.
// These follow the OpenTelemetry Semantic Conventions for Generative AI.
const (
	// GenAISystemKey identifies the AI system (e.g., "openai", "anthropic").
	GenAISystemKey = "gen_ai.system"

	// GenAIOperationNameKey identifies the operation type.
	GenAIOperationNameKey = "gen_ai.operation.name"

	// GenAIRequestModelKey identifies the requested model.
	GenAIRequestModelKey = "gen_ai.request.model"

	// GenAIResponseModelKey identifies the model used in the response.
	GenAIResponseModelKey = "gen_ai.response.model"

	// GenAIRequestMaxTokensKey identifies the maximum tokens requested.
	GenAIRequestMaxTokensKey = "gen_ai.request.max_tokens"

	// GenAIRequestTemperatureKey identifies the temperature setting.
	GenAIRequestTemperatureKey = "gen_ai.request.temperature"

	// GenAIRequestTopPKey identifies the top_p setting.
	GenAIRequestTopPKey = "gen_ai.request.top_p"

	// GenAIUsageInputTokensKey identifies the number of input tokens.
	GenAIUsageInputTokensKey = "gen_ai.usage.input_tokens"

	// GenAIUsageOutputTokensKey identifies the number of output tokens.
	GenAIUsageOutputTokensKey = "gen_ai.usage.output_tokens"

	// GenAIResponseFinishReasonsKey identifies why generation stopped.
	GenAIResponseFinishReasonsKey = "gen_ai.response.finish_reasons"

	// GenAIResponseIDKey identifies the response ID.
	GenAIResponseIDKey = "gen_ai.response.id"

	// AgentNameKey identifies the agent name.
	AgentNameKey = "agent.name"

	// AgentIDKey identifies the agent ID.
	AgentIDKey = "agent.id"

	// AgentProviderKey identifies the agent provider.
	AgentProviderKey = "agent.provider"
)

// Operation names for GenAI operations.
const (
	// OperationChat represents a chat completion operation.
	OperationChat = "chat"

	// OperationAgentRun represents an agent run operation.
	OperationAgentRun = "agent.run"

	// OperationAgentRunStream represents a streaming agent run operation.
	OperationAgentRunStream = "agent.run_stream"

	// OperationToolCall represents a tool/function call operation.
	OperationToolCall = "tool.call"
)

// Tracer returns the global tracer for agent framework operations.
func Tracer() trace.Tracer {
	return otel.Tracer(InstrumentationName)
}

// Meter returns the global meter for agent framework metrics.
func Meter() metric.Meter {
	return otel.Meter(InstrumentationName)
}

// StartAgentSpan starts a new span for an agent operation.
// This creates a span with the appropriate semantic conventions for agent operations.
func StartAgentSpan(ctx context.Context, operationName string, agentName string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	return Tracer().Start(ctx, operationName,
		append(opts,
			trace.WithSpanKind(trace.SpanKindClient),
			trace.WithAttributes(
				attribute.String(GenAIOperationNameKey, operationName),
				attribute.String(AgentNameKey, agentName),
			),
		)...,
	)
}

// StartChatSpan starts a new span for a chat completion operation.
// This creates a span with the appropriate semantic conventions for chat operations.
func StartChatSpan(ctx context.Context, system string, model string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	spanName := system + " " + OperationChat
	return Tracer().Start(ctx, spanName,
		append(opts,
			trace.WithSpanKind(trace.SpanKindClient),
			trace.WithAttributes(
				attribute.String(GenAISystemKey, system),
				attribute.String(GenAIOperationNameKey, OperationChat),
				attribute.String(GenAIRequestModelKey, model),
			),
		)...,
	)
}

// RecordUsage records token usage attributes on the current span.
func RecordUsage(span trace.Span, inputTokens, outputTokens int) {
	span.SetAttributes(
		attribute.Int(GenAIUsageInputTokensKey, inputTokens),
		attribute.Int(GenAIUsageOutputTokensKey, outputTokens),
	)
}

// RecordResponse records response attributes on the current span.
func RecordResponse(span trace.Span, responseID string, model string, finishReason string) {
	span.SetAttributes(
		attribute.String(GenAIResponseIDKey, responseID),
		attribute.String(GenAIResponseModelKey, model),
		attribute.String(GenAIResponseFinishReasonsKey, finishReason),
	)
}

// RecordRequestOptions records request option attributes on the current span.
func RecordRequestOptions(span trace.Span, maxTokens int, temperature float32, topP float32) {
	attrs := make([]attribute.KeyValue, 0, 3)
	if maxTokens > 0 {
		attrs = append(attrs, attribute.Int(GenAIRequestMaxTokensKey, maxTokens))
	}
	if temperature > 0 {
		attrs = append(attrs, attribute.Float64(GenAIRequestTemperatureKey, float64(temperature)))
	}
	if topP > 0 {
		attrs = append(attrs, attribute.Float64(GenAIRequestTopPKey, float64(topP)))
	}
	if len(attrs) > 0 {
		span.SetAttributes(attrs...)
	}
}
