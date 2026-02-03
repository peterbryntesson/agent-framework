<!-- markdownlint-disable-file -->
# Release Changes: Go Port - Epic 2: LLM Provider Implementations

**Related Plan**: 2026-02-02-golang-epic2-providers-plan.instructions.md
**Implementation Date**: 2026-02-02

## Summary

Implementation of Epic 2 for the Go SDK port, covering LLM provider implementations, tool system, and observability. This change log tracks incremental progress as features and steps are completed.

## Changes

### Added

* go/tool/doc.go - Package documentation for the tool package explaining Tool, HostedTool, FunctionTool, and Invoker concepts with usage examples
* go/tool/tool.go - Core Tool and HostedTool interfaces with supporting types (ToolChoice, ApprovalMode, ToolType, ToolCall, ToolResult) aligned with .NET AITool and Python ToolProtocol
* go/tool/result.go - Result struct for tool invocation outputs with constructors for success, error, and value-based results, plus JSON serialization support
* go/tool/config.go - InvocationConfig struct controlling tool invocation behavior with defaults matching Python FunctionInvocationConfiguration, including fluent builder methods and Merge/Validate helpers
* go/tool/schema.go - JSON Schema generation from Go struct types using reflection, supporting struct tags for descriptions, enums, required fields, and defaults; handles nested structs, slices, and maps
* go/tool/function.go - FunctionTool implementation wrapping Go functions for AI model invocation with automatic signature validation, JSON Schema generation, and invocation counting; includes Func() decorator and FuncOption configuration pattern
* go/tool/hosted.go - Hosted tool implementations for provider-hosted capabilities (HostedWebSearchTool, HostedCodeInterpreterTool, HostedFileSearchTool, HostedMCPTool, HostedImageGenerationTool) with configuration types, constructors, and ProviderConfig serialization aligned with Python HostedTool classes
* go/tool/hosted_test.go - Comprehensive tests for hosted tool types covering interface compliance, ProviderConfig serialization, default names, and error handling (21 tests)
* go/tool/errors.go - Tool-specific error types including sentinel errors (ErrUnknownTool, ErrInvalidArguments, ErrMaxIterations, ErrConsecutiveErrors, ErrInvocationDisabled, ErrToolTimeout) and structured error types (InvocationError, InvocationPanicError, UnknownToolError, ArgumentError) with proper error wrapping and Is/Unwrap support
* go/tool/invoke.go - Invoker struct for safe tool invocation with panic recovery, timeout support, batch processing (sequential and parallel), context cancellation handling, tool management methods, and InvocationResult type for batch results
* go/tool/invoke_test.go - Comprehensive tests for Invoker covering creation, invocation, unknown tool handling, panic recovery, batch processing, tool management, context cancellation, and InvocationResult helpers (26 tests)
* go/tool/tool_test.go - Comprehensive tests for core Tool and HostedTool interfaces, type constants (ToolChoice, ApprovalMode, ToolType), ToolCall/ToolResult serialization, and AdditionalProperties (14 tests)
* go/tool/function_test.go - Comprehensive tests for FunctionTool creation, invocation, signature validation, max invocations limit, Func() decorator, MustFunc(), and option functions (32 tests)
* go/tool/schema_test.go - Comprehensive tests for JSON Schema generation covering basic types, numeric types, enum support, default values, nested structs, slices, maps, required fields, field visibility, pointer types, and unsupported types (23 tests)
* go/tool/config_test.go - Comprehensive tests for InvocationConfig including defaults, builder methods (With*), Merge, and Validate functions (15 tests)
* go/tool/result_test.go - Comprehensive tests for Result struct including constructors, String(), JSON serialization, WithMetadata, and GetMetadata (16 tests)
* go/tool/errors_test.go - Comprehensive tests for error types including sentinel errors, InvocationError, InvocationPanicError, UnknownToolError, and ArgumentError with proper error wrapping support (12 tests)
* go/providers/openai/doc.go - Package documentation for the OpenAI provider explaining client creation, configuration options, basic usage, streaming, tool calling, Responses API, error handling, and observability
* go/providers/openai/options.go - Configuration options using functional options pattern (WithAPIKey, WithModel, WithBaseURL, WithOrgID, WithInstructionRole, WithHTTPClient) with environment variable fallbacks
* go/providers/openai/client.go - OpenAI Client struct implementing chat.Client interface with GetResponse, GetStreamingResponse, and Metadata methods; includes request building, streaming processing, and convenience constructors (NewClientWithHTTPClient, NewClientWithTimeout)
* go/providers/openai/convert.go - Message conversion utilities for transforming between chat.Message and OpenAI format; handles text content, image content, tool calls, tool results, and finish reasons
* go/providers/openai/stream.go - Stream processing utilities including StreamProcessor for managing tool call accumulation during streaming, ProcessStream for handling OpenAI stream chunks, and CollectStreamToResponse for assembling complete responses from stream updates (9 tests in stream_test.go)
* go/providers/openai/tools.go - Tool conversion utilities for tool.Tool to OpenAI format transformation; includes ConvertToolsToOpenAI (function tools), ConvertHostedToolsToOpenAI (hosted tool configs), HasHostedTools, SeparateTools, ToolChoiceForOpenAI, and ParallelToolCallsOption helpers
* go/providers/openai/tools_test.go - Comprehensive tests for tool conversion utilities covering function tools, hosted tools, tool separation, tool choice conversion, and configuration helpers (17 tests)
* go/providers/openai/responses_options.go - Functional options pattern for ResponsesClient configuration including ResponsesWithAPIKey, ResponsesWithModel, ResponsesWithBaseURL, ResponsesWithOrgID, ResponsesWithInstructionRole, ResponsesWithHTTPClient, and ResponsesWithInstructions options
* go/providers/openai/responses.go - ResponsesClient implementing chat.Client interface for OpenAI's stateful Responses API; includes custom HTTP client (go-openai library doesn't support Responses API), conversation continuation via previous_response_id, hosted tool support (web_search, code_interpreter, file_search, computer_use, mcp, image_generation), and streaming response handling
* go/providers/openai/responses_test.go - Comprehensive tests for ResponsesClient covering client creation, option application, metadata, response ID management, GetResponse, GetResponseWithTools, conversation continuation, API errors, streaming, request building, message conversion, and role conversion (22 test cases)
* go/providers/openai/client_test.go - Comprehensive unit tests for Client covering creation with options, environment variables, GetResponse with mock servers, GetResponseWithTools, GetStreamingResponse, GetStreamingResponseWithTools, tool choice handling, and interface compliance (35 test cases)
* go/providers/openai/convert_test.go - Comprehensive tests for message conversion utilities covering toOpenAIMessages, toOpenAIMessage, extractTextContent, toOpenAIContentParts, toOpenAIToolCalls, toOpenAITools, fromOpenAIMessage, fromOpenAIToolCalls, fromOpenAIFinishReason, and round-trip conversion (25 test cases)
* go/providers/openai/options_test.go - Unit tests for configuration options including defaultConfig, applyEnvDefaults, WithAPIKey, WithModel, WithBaseURL, WithOrgID, WithInstructionRole, WithHTTPClient, and chained application (11 test cases)
* go/providers/openai/responses_options_test.go - Unit tests for ResponsesClient configuration options including defaultResponsesConfig, applyEnvDefaults, all ResponsesWith* option functions, and chained application (16 test cases)
* go/providers/openai/integration_test.go - Integration tests with //go:build integration tag for real API testing; covers Client.GetResponse, Client.GetStreamingResponse, ResponsesClient.GetResponse, conversation continuation, different models; requires OPENAI_API_KEY environment variable (15 test cases)
* go/chatagent/doc.go - Package documentation for ChatClientAgent with usage examples for basic agent creation, tool integration, streaming, session management, builder pattern, and observability
* go/chatagent/options.go - Functional options pattern for agent configuration (WithID, WithName, WithDescription, WithInstructions, WithTools, WithMaxTurns, WithInvocationConfig, WithInvocationEnabled, WithMaxConsecutiveErrors, WithTerminateOnUnknownCalls, WithIncludeDetailedErrors, WithToolTimeout, WithParallelToolCalls, WithReturnIntermediateSteps)
* go/chatagent/agent.go - Agent struct implementing agent.Agent interface with New constructor, Run, RunStream, NewSession, RestoreSession, GetService, RegisterService methods; message preparation and tool conversion utilities
* go/chatagent/builder.go - Fluent Builder pattern for agent configuration with NewBuilder, chained configuration methods, Build and MustBuild with validation
* go/chatagent/toolloop.go - Tool invocation loop implementation with runWithToolLoop, runStreamWithToolLoop, collectStreamUpdates, invokeToolCalls, and invokeToolCallsParallel
* go/chatagent/session.go - Session struct implementing agent.Session with thread-safe message storage, serialization, and management methods
* go/chatagent/agent_test.go - Unit tests for Agent covering creation, interface compliance, metadata, Run, RunStream, tools, and services (15 test cases)
* go/chatagent/options_test.go - Unit tests for option functions and default configuration (18 test cases)
* go/chatagent/builder_test.go - Unit tests for Builder pattern covering chaining, validation, and error handling (20 test cases)
* go/chatagent/session_test.go - Unit tests for Session covering creation, messages, serialization, timestamps, cloning, and concurrency (20 test cases)
* go/chatagent/toolloop_test.go - Unit tests for tool invocation loop covering single turn, tool calls, max turns, errors, parallel execution (12 test cases)

### Modified

* go/go.mod - Added github.com/sashabaranov/go-openai v1.41.2 dependency for OpenAI API client
* go/providers/openai/client.go - Added tool package import and GetResponseWithTools/GetStreamingResponseWithTools convenience methods for tool.Tool interface integration
* go/go.sum - Updated with go-openai dependency checksums

### Removed

## Additional or Deviating Changes

* Extended InvocationConfig beyond the minimal specification to include ReturnIntermediateSteps, TimeoutSeconds, and ParallelToolCalls fields for feature parity with advanced provider capabilities
  * These fields support common use cases like debugging agent interactions, enforcing time limits, and controlling execution parallelism
* Implemented streaming response handling in Step 2.2.1 rather than deferring to Step 2.2.3 because the Client interface requires GetStreamingResponse
  * This deviation ensures the chat.Client interface is fully satisfied from the initial implementation
* Implemented basic tool conversion in Step 2.2.1 rather than deferring to Step 2.2.4 because buildRequest needs tool support for Options.Tools
  * This keeps the request building logic cohesive and avoids placeholder implementations
* Step 2.2.2 (Chat Completions API integration) marked complete as all functionality was already implemented in Step 2.2.1
  * GetResponse method with full message conversion and error handling
  * Message conversion utilities (toOpenAIMessages, toOpenAIMessage, toOpenAIContentParts, toOpenAIToolCalls, fromOpenAIMessage, fromOpenAIToolCalls, fromOpenAIFinishReason)
  * Usage details correctly populated from API response
  * Verified with go build and go vet
* Step 2.2.3 (Streaming response handling) formalized with dedicated stream.go file
  * Created StreamProcessor struct for managing streaming state and tool call accumulation
  * Implemented ProcessStream function to handle OpenAI stream chunks with proper context cancellation
  * Added CollectStreamToResponse utility for converting stream updates to complete Response
  * Refactored client.go to use ProcessStream function (removed 85 lines of inline streaming logic)
  * Added 9 comprehensive stream_test.go tests covering text content, tool calls, errors, and concurrent access
  * Verified with go build, go vet, and go test (all 9 tests pass)
* Step 2.2.4 (Tool calling support) implemented with tool package integration
  * Created tools.go with tool.Tool to OpenAI format conversion utilities
  * Implemented ConvertToolsToOpenAI for function tools (filters out hosted tools)
  * Implemented ConvertHostedToolsToOpenAI for extracting hosted tool configurations
  * Added HasHostedTools helper to determine if Responses API is needed
  * Added SeparateTools helper to split tools by type
  * Added ToolChoiceForOpenAI for converting tool choice constants and specific tool names
  * Added ParallelToolCallsOption helper for optional bool field handling
  * Extended client.go with GetResponseWithTools and GetStreamingResponseWithTools convenience methods
  * Added tool package import to client.go for tool.Tool and tool.HostedTool interfaces
  * Created tools_test.go with 17 comprehensive tests covering all conversion utilities
  * Verified with go build, go vet, and go test (all 26 tests pass)
* Step 2.2.5 (Responses API client) implemented with custom HTTP client
  * Created responses_options.go with functional options pattern for ResponsesClient configuration
  * Created responses.go with ResponsesClient implementing chat.Client interface for OpenAI's Responses API
  * Used custom HTTP client because github.com/sashabaranov/go-openai v1.41.2 does not support Responses API natively
  * Implemented stateful conversation continuation via previous_response_id field with thread-safe access
  * Implemented hosted tool support for web_search, code_interpreter, file_search, computer_use, mcp, and image_generation tool types
  * Implemented GetResponse, GetResponseWithTools, GetStreamingResponse, and GetStreamingResponseWithTools methods
  * Added ClearConversation, GetPreviousResponseID, and SetPreviousResponseID for conversation state management
  * Created responses_test.go with 22 comprehensive test cases using httptest mock server
  * Verified with go build, go vet, and go test (all tests pass)
* Step 2.2.6 (Add OpenAI provider tests) implemented with comprehensive test coverage
  * Created client_test.go with 35 test cases covering client creation, GetResponse, GetStreamingResponse, GetResponseWithTools, GetStreamingResponseWithTools, tool choice handling, interface compliance, and unmarshalParameters
  * Created convert_test.go with 25 test cases covering message conversion, content extraction, tool call conversion, finish reason mapping, and round-trip conversion
  * Created options_test.go with 11 test cases covering config defaults, environment variable handling, and all option functions
  * Created responses_options_test.go with 16 test cases covering ResponsesClient configuration options and environment variable handling
  * Created integration_test.go with build tag `//go:build integration` containing 15 integration test cases for real API testing
  * Integration tests cover Client and ResponsesClient functionality including streaming, tool calls, and conversation continuation
  * Achieved 86.7% code coverage for unit tests (note: 90%+ target partially limited by internal streaming code paths requiring actual OpenAI stream objects)
  * Total test count: 113+ tests (226 test lines including subtests)
  * Verified with go build, go vet, and go test (all tests pass)
* Feature 2.7 (OpenTelemetry Observability Package) implemented:
  * go/observability/semconv.go - Additional GenAI semantic convention constants for metric names (MetricAgentRuns, MetricTokensInput, MetricTokensOutput, MetricRequestLatency, MetricErrors, MetricToolInvocations) and attribute keys (GenAIAgentIDKey, GenAIAgentNameKey, GenAIProviderNameKey, GenAIErrorTypeKey, GenAIToolNameKey, etc.) complementing otel.go
  * go/observability/otel.go - Extended with RecordUsageDetails (for chat.UsageDetails), RecordError (with status code), EndSpanWithError, StartToolSpan, SpanFromContext, and StartAgentSpanWithID functions; added chat package import and codes import for error handling
  * go/observability/metrics.go - Metrics struct with Int64Counter and Float64Histogram instruments; NewMetrics creates all instruments; RecordAgentRun, RecordTokenUsage, RecordLatency, RecordError, RecordToolInvocation methods; DefaultMetrics singleton pattern
  * go/observability/instrumented.go - InstrumentedClient wrapping chat.Client with automatic tracing and metrics; GetResponse and GetStreamingResponse with span creation, usage recording, latency measurement, error tracking; InstrumentedClientOption pattern with WithSensitiveData and WithMetrics; InstrumentedAgentClient for agent-level instrumentation with StartRun returning completion callback; context helpers (ContextWithMetrics, MetricsFromContext, RecordAgentRunFromContext, etc.)
  * go/observability/setup.go - Setup function for OpenTelemetry configuration with OTLP gRPC export; SetupConfig with ServiceName, ServiceVersion, EnableTracing, EnableMetrics; SetupOption pattern with WithServiceVersion, WithTraceExporter, WithMetricExporter, WithSampler, WithPropagators; SetupTracing and SetupMetrics convenience functions; buildResource, setupTracing, setupMetrics internal functions
  * go/observability/doc.go - Comprehensive package documentation with overview, quick start, semantic conventions, tracing, metrics, instrumented client, configuration, and environment variable sections
  * go/observability/semconv_test.go - 10 tests covering metric names, agent/provider/token/error/tool attributes, naming conventions, and otel.go constants
  * go/observability/metrics_test.go - 14 tests covering NewMetrics, RecordAgentRun, RecordTokenUsage, RecordLatency, RecordError, RecordToolInvocation, DefaultMetrics singleton, concurrent recordings, and multi-provider scenarios
  * go/observability/instrumented_test.go - 28 tests covering InstrumentedClient creation, interface compliance, GetResponse success/error, GetStreamingResponse success/error, streaming error handling, convenience functions, context helpers, InstrumentedAgentClient, and concurrent requests
  * go/observability/setup_test.go - 12 tests covering DefaultSetupConfig, option functions, setup with disabled providers, convenience functions, and option chaining
  * All 74 observability tests pass; go build and go vet clean
  * Added go.mod dependencies: go.opentelemetry.io/otel/sdk/metric v1.40.0, go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc v1.40.0, go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc v1.40.0
* Feature 2.8 (ChatClientAgent Implementation) implemented:
  * go/chatagent/doc.go - Comprehensive package documentation with usage examples for basic agent creation, tool integration, streaming, session management, builder pattern, and observability
  * go/chatagent/options.go - Functional options pattern (WithID, WithName, WithDescription, WithInstructions, WithTools, WithMaxTurns, WithInvocationConfig, WithInvocationEnabled, WithMaxConsecutiveErrors, WithTerminateOnUnknownCalls, WithIncludeDetailedErrors, WithToolTimeout, WithParallelToolCalls, WithReturnIntermediateSteps) with default configuration
  * go/chatagent/agent.go - Agent struct implementing agent.Agent interface; New constructor with functional options; ID, Name, Description, Metadata, Run, RunStream, NewSession, RestoreSession, GetService methods; message preparation with instructions and session history; chat options preparation with tool conversion
  * go/chatagent/builder.go - Fluent Builder pattern for agent configuration; NewBuilder constructor; chained configuration methods (ID, Name, Description, Instructions, Tools, MaxTurns, InvocationConfig, InvocationEnabled, ToolTimeout, ParallelToolCalls, IncludeDetailedErrors); Build and MustBuild methods with validation
  * go/chatagent/toolloop.go - Tool invocation loop implementation; runWithToolLoop for non-streaming execution with automatic tool invocation, usage accumulation, consecutive error handling; runStreamWithToolLoop and processStreamWithToolLoop for streaming execution; collectStreamUpdates for stream processing; invokeToolCalls for sequential execution; invokeToolCallsParallel for parallel tool execution
  * go/chatagent/session.go - Session struct implementing agent.Session with thread-safe message storage; newSession and restoreSession constructors; ID, AgentID, Messages, AddMessage, AddMessages, ClearMessages, MessageCount, Serialize, GetService, RegisterService, CreatedAt, ModifiedAt, Clone methods
  * go/chatagent/agent_test.go - 15 test cases covering agent creation, interface compliance, metadata, Run with various options, RunStream, tool definitions, and service registration
  * go/chatagent/options_test.go - 18 test cases covering default config, all option functions, chained options
  * go/chatagent/builder_test.go - 20 test cases covering builder methods, chaining, validation, nil client handling, MustBuild panic, error propagation
  * go/chatagent/session_test.go - 20 test cases covering session creation, message management, serialization, restoration, timestamps, cloning, concurrent access
  * go/chatagent/toolloop_test.go - 12 test cases covering single turn, tool call loop, max turns, invocation disabled, tool errors, consecutive errors, unknown tools, parallel execution, context cancellation, streaming
  * Achieved 79.1% code coverage for chatagent package
  * All 85+ tests pass; go build and go vet clean

<!-- Include after final phase: total files affected, files created/modified/removed with paths and purposes, dependency and infrastructure changes, deployment notes -->

