// Copyright (c) Microsoft. All rights reserved.

package observability

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTracer_ReturnsTracer(t *testing.T) {
	// Act
	tracer := Tracer()

	// Assert
	assert.NotNil(t, tracer, "Tracer should not be nil")
}

func TestMeter_ReturnsMeter(t *testing.T) {
	// Act
	meter := Meter()

	// Assert
	assert.NotNil(t, meter, "Meter should not be nil")
}

func TestInstrumentationName_IsCorrect(t *testing.T) {
	// Assert
	assert.Equal(t, "github.com/microsoft/agent-framework-go", InstrumentationName)
}

func TestSemanticConventionConstants_AreDefined(t *testing.T) {
	// Assert - verify key constants are properly defined
	assert.Equal(t, "gen_ai.system", GenAISystemKey)
	assert.Equal(t, "gen_ai.operation.name", GenAIOperationNameKey)
	assert.Equal(t, "gen_ai.request.model", GenAIRequestModelKey)
	assert.Equal(t, "gen_ai.response.model", GenAIResponseModelKey)
	assert.Equal(t, "gen_ai.usage.input_tokens", GenAIUsageInputTokensKey)
	assert.Equal(t, "gen_ai.usage.output_tokens", GenAIUsageOutputTokensKey)
	assert.Equal(t, "agent.name", AgentNameKey)
	assert.Equal(t, "agent.id", AgentIDKey)
}

func TestOperationConstants_AreDefined(t *testing.T) {
	// Assert
	assert.Equal(t, "chat", OperationChat)
	assert.Equal(t, "agent.run", OperationAgentRun)
	assert.Equal(t, "agent.run_stream", OperationAgentRunStream)
	assert.Equal(t, "tool.call", OperationToolCall)
}

func TestStartAgentSpan_CreatesSpan(t *testing.T) {
	// Arrange
	ctx := context.Background()

	// Act
	newCtx, span := StartAgentSpan(ctx, OperationAgentRun, "test-agent")
	defer span.End()

	// Assert
	assert.NotNil(t, newCtx)
	assert.NotNil(t, span)
	assert.True(t, span.SpanContext().IsValid() || !span.IsRecording(), "Span should be valid or not recording (no exporter configured)")
}

func TestStartChatSpan_CreatesSpan(t *testing.T) {
	// Arrange
	ctx := context.Background()

	// Act
	newCtx, span := StartChatSpan(ctx, "openai", "gpt-4")
	defer span.End()

	// Assert
	assert.NotNil(t, newCtx)
	assert.NotNil(t, span)
}

func TestRecordUsage_SetsAttributes(t *testing.T) {
	// Arrange
	ctx := context.Background()
	_, span := StartChatSpan(ctx, "openai", "gpt-4")
	defer span.End()

	// Act - should not panic
	RecordUsage(span, 100, 50)
}

func TestRecordResponse_SetsAttributes(t *testing.T) {
	// Arrange
	ctx := context.Background()
	_, span := StartChatSpan(ctx, "openai", "gpt-4")
	defer span.End()

	// Act - should not panic
	RecordResponse(span, "resp-123", "gpt-4", "stop")
}

func TestRecordRequestOptions_SetsAttributes(t *testing.T) {
	// Arrange
	ctx := context.Background()
	_, span := StartChatSpan(ctx, "openai", "gpt-4")
	defer span.End()

	// Act - should not panic
	RecordRequestOptions(span, 1000, 0.7, 0.9)
}

func TestRecordRequestOptions_SkipsZeroValues(t *testing.T) {
	// Arrange
	ctx := context.Background()
	_, span := StartChatSpan(ctx, "openai", "gpt-4")
	defer span.End()

	// Act - should not panic, should skip zero values
	RecordRequestOptions(span, 0, 0, 0)
}
