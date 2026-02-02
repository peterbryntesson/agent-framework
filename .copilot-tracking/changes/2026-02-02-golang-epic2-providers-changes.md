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

### Modified

### Removed

## Additional or Deviating Changes

* Extended InvocationConfig beyond the minimal specification to include ReturnIntermediateSteps, TimeoutSeconds, and ParallelToolCalls fields for feature parity with advanced provider capabilities
  * These fields support common use cases like debugging agent interactions, enforcing time limits, and controlling execution parallelism

## Release Summary

<!-- Include after final phase: total files affected, files created/modified/removed with paths and purposes, dependency and infrastructure changes, deployment notes -->
