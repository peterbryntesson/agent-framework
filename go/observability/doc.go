// Copyright (c) Microsoft. All rights reserved.

// Package observability provides OpenTelemetry integration for the Agent Framework.
//
// This package enables tracing, metrics, and logging for agent operations
// using the OpenTelemetry standard. It follows the Semantic Conventions for
// Generative AI Systems as defined by OpenTelemetry.
//
// See: https://opentelemetry.io/docs/specs/semconv/gen-ai/
//
// # Overview
//
// The observability package provides three main capabilities:
//
// 1. Tracing: Distributed tracing for agent invocations, chat requests, and tool calls
// 2. Metrics: Counters and histograms for token usage, latency, and error rates
// 3. Instrumented Client: A wrapper that adds observability to any chat.Client
//
// # Quick Start
//
// To add observability to your application:
//
//	// Setup OpenTelemetry with OTLP export
//	shutdown, err := observability.Setup(ctx, "my-agent-app")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer shutdown(ctx)
//
//	// Wrap your chat client with instrumentation
//	chatClient := openai.NewClient(...)
//	instrumented := observability.InstrumentClient(chatClient)
//
//	// Use the instrumented client - tracing and metrics are automatic
//	resp, err := instrumented.GetResponse(ctx, messages, nil)
//
// # Semantic Conventions
//
// This package follows the OpenTelemetry Semantic Conventions for Generative AI.
// Key attributes include:
//
//	gen_ai.agent.id        - Unique agent identifier
//	gen_ai.agent.name      - Human-readable agent name
//	gen_ai.provider.name   - LLM provider (openai, anthropic, etc.)
//	gen_ai.request.model   - Model being used
//	gen_ai.usage.input_tokens   - Input token count
//	gen_ai.usage.output_tokens  - Output token count
//
// # Tracing
//
// The package provides span creation functions for different operation types:
//
//	// Agent invocation span
//	ctx, span := observability.StartAgentSpan(ctx, "agent-id", "AgentName", "openai")
//	defer span.End()
//
//	// Chat request span
//	ctx, span := observability.StartChatSpan(ctx, "openai", "gpt-4")
//	defer span.End()
//
//	// Tool invocation span
//	ctx, span := observability.StartToolSpan(ctx, "get_weather", "call-123")
//	defer span.End()
//
// # Metrics
//
// Available metrics:
//
//	gen_ai.agent.runs          - Counter of agent runs
//	gen_ai.usage.input_tokens  - Histogram of input tokens
//	gen_ai.usage.output_tokens - Histogram of output tokens
//	gen_ai.request.latency     - Histogram of request latency (seconds)
//	gen_ai.errors              - Counter of errors
//	gen_ai.tool.invocations    - Counter of tool invocations
//
// # Instrumented Client
//
// The InstrumentedClient wraps any chat.Client and automatically:
//
//   - Creates spans for all requests
//   - Records token usage metrics
//   - Tracks request latency
//   - Counts errors
//   - Propagates context for distributed tracing
//
// Example:
//
//	client := observability.NewInstrumentedClient(baseClient)
//
//	// Or with options:
//	client := observability.NewInstrumentedClientWithOptions(baseClient,
//	    observability.WithSensitiveData(true), // Include message content in traces
//	    observability.WithMetrics(customMetrics),
//	)
//
// # Configuration
//
// Setup accepts options to customize behavior:
//
//	shutdown, err := observability.Setup(ctx, "my-service",
//	    observability.WithServiceVersion("1.0.0"),
//	    observability.WithSampler(trace.AlwaysSample()),
//	    observability.WithTraceExporter(customExporter),
//	)
//
// # Environment Variables
//
// The package respects standard OpenTelemetry environment variables:
//
//	OTEL_EXPORTER_OTLP_ENDPOINT - OTLP endpoint URL
//	OTEL_SERVICE_NAME           - Service name (overridden by Setup parameter)
//	OTEL_INSTRUMENTATION_GENAI_CAPTURE_MESSAGE_CONTENT - Enable sensitive data
package observability
