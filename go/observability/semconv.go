// Copyright (c) Microsoft. All rights reserved.

package observability

// Additional GenAI semantic conventions for AI agent observability.
// These complement the core constants defined in otel.go.
// See: https://opentelemetry.io/docs/specs/semconv/gen-ai/

// Metric names for GenAI observability.
const (
	// MetricAgentRuns counts the number of agent runs.
	MetricAgentRuns = "gen_ai.agent.runs"

	// MetricTokensInput is a histogram of input tokens per request.
	MetricTokensInput = "gen_ai.usage.input_tokens"

	// MetricTokensOutput is a histogram of output tokens per request.
	MetricTokensOutput = "gen_ai.usage.output_tokens"

	// MetricRequestLatency is a histogram of request latency in seconds.
	MetricRequestLatency = "gen_ai.request.latency"

	// MetricErrors counts the number of errors.
	MetricErrors = "gen_ai.errors"

	// MetricToolInvocations counts tool invocations.
	MetricToolInvocations = "gen_ai.tool.invocations"
)

// Additional attribute constants not in otel.go.
const (
	// GenAIAgentIDKey identifies the unique agent ID.
	GenAIAgentIDKey = "gen_ai.agent.id"

	// GenAIAgentNameKey identifies the human-readable agent name.
	GenAIAgentNameKey = "gen_ai.agent.name"

	// GenAIAgentDescriptionKey provides the agent's description.
	GenAIAgentDescriptionKey = "gen_ai.agent.description"

	// GenAIProviderNameKey identifies the LLM provider.
	GenAIProviderNameKey = "gen_ai.provider.name"

	// GenAIUsageCachedTokensKey is the number of cached tokens.
	GenAIUsageCachedTokensKey = "gen_ai.usage.cached_tokens"

	// GenAIUsageReasoningTokensKey is the number of reasoning tokens.
	GenAIUsageReasoningTokensKey = "gen_ai.usage.reasoning_tokens"

	// GenAIUsageTotalTokensKey is the total number of tokens.
	GenAIUsageTotalTokensKey = "gen_ai.usage.total_tokens"

	// GenAIErrorTypeKey identifies the error type.
	GenAIErrorTypeKey = "gen_ai.error.type"

	// GenAIErrorMessageKey provides the error message.
	GenAIErrorMessageKey = "gen_ai.error.message"

	// GenAIToolNameKey identifies the tool name.
	GenAIToolNameKey = "gen_ai.tool.name"

	// GenAIToolCallIDKey identifies the tool call.
	GenAIToolCallIDKey = "gen_ai.tool.call_id"
)
