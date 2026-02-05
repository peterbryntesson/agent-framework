// Copyright (c) Microsoft. All rights reserved.

package observability

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSemanticConventions_MetricNames(t *testing.T) {
	// Assert
	assert.Equal(t, "gen_ai.agent.runs", MetricAgentRuns)
	assert.Equal(t, "gen_ai.usage.input_tokens", MetricTokensInput)
	assert.Equal(t, "gen_ai.usage.output_tokens", MetricTokensOutput)
	assert.Equal(t, "gen_ai.request.latency", MetricRequestLatency)
	assert.Equal(t, "gen_ai.errors", MetricErrors)
	assert.Equal(t, "gen_ai.tool.invocations", MetricToolInvocations)
}

func TestSemanticConventions_AgentAttributes(t *testing.T) {
	// Assert
	assert.Equal(t, "gen_ai.agent.id", GenAIAgentIDKey)
	assert.Equal(t, "gen_ai.agent.name", GenAIAgentNameKey)
	assert.Equal(t, "gen_ai.agent.description", GenAIAgentDescriptionKey)
}

func TestSemanticConventions_ProviderAttributes(t *testing.T) {
	// Assert
	assert.Equal(t, "gen_ai.provider.name", GenAIProviderNameKey)
}

func TestSemanticConventions_TokenUsageAttributes(t *testing.T) {
	// Assert
	assert.Equal(t, "gen_ai.usage.cached_tokens", GenAIUsageCachedTokensKey)
	assert.Equal(t, "gen_ai.usage.reasoning_tokens", GenAIUsageReasoningTokensKey)
	assert.Equal(t, "gen_ai.usage.total_tokens", GenAIUsageTotalTokensKey)
}

func TestSemanticConventions_ErrorAttributes(t *testing.T) {
	// Assert
	assert.Equal(t, "gen_ai.error.type", GenAIErrorTypeKey)
	assert.Equal(t, "gen_ai.error.message", GenAIErrorMessageKey)
}

func TestSemanticConventions_ToolAttributes(t *testing.T) {
	// Assert
	assert.Equal(t, "gen_ai.tool.name", GenAIToolNameKey)
	assert.Equal(t, "gen_ai.tool.call_id", GenAIToolCallIDKey)
}

func TestSemanticConventions_AllAttributesFollowNaming(t *testing.T) {
	// All GenAI attributes from semconv.go should start with gen_ai.
	attrs := []string{
		GenAIAgentIDKey,
		GenAIAgentNameKey,
		GenAIAgentDescriptionKey,
		GenAIProviderNameKey,
		GenAIUsageCachedTokensKey,
		GenAIUsageReasoningTokensKey,
		GenAIUsageTotalTokensKey,
		GenAIErrorTypeKey,
		GenAIErrorMessageKey,
		GenAIToolNameKey,
		GenAIToolCallIDKey,
	}

	for _, attr := range attrs {
		assert.Contains(t, attr, "gen_ai.", "Attribute %s should follow gen_ai. prefix convention", attr)
	}
}

func TestSemanticConventions_OtelGoConstants(t *testing.T) {
	// Verify constants from otel.go
	assert.Equal(t, "github.com/microsoft/agent-framework-go", InstrumentationName)
	assert.Equal(t, "gen_ai.system", GenAISystemKey)
	assert.Equal(t, "gen_ai.operation.name", GenAIOperationNameKey)
	assert.Equal(t, "gen_ai.request.model", GenAIRequestModelKey)
	assert.Equal(t, "gen_ai.response.model", GenAIResponseModelKey)
	assert.Equal(t, "gen_ai.usage.input_tokens", GenAIUsageInputTokensKey)
	assert.Equal(t, "gen_ai.usage.output_tokens", GenAIUsageOutputTokensKey)
}

func TestSemanticConventions_OperationNames(t *testing.T) {
	// Assert
	assert.Equal(t, "chat", OperationChat)
	assert.Equal(t, "agent.run", OperationAgentRun)
	assert.Equal(t, "agent.run_stream", OperationAgentRunStream)
	assert.Equal(t, "tool.call", OperationToolCall)
}
