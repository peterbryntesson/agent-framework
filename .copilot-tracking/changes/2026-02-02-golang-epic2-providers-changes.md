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

### Modified

### Removed

## Additional or Deviating Changes

* Extended InvocationConfig beyond the minimal specification to include ReturnIntermediateSteps, TimeoutSeconds, and ParallelToolCalls fields for feature parity with advanced provider capabilities
  * These fields support common use cases like debugging agent interactions, enforcing time limits, and controlling execution parallelism

## Release Summary

<!-- Include after final phase: total files affected, files created/modified/removed with paths and purposes, dependency and infrastructure changes, deployment notes -->
