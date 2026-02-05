<!-- markdownlint-disable-file -->
# Release Changes: TelemetryMiddleware for OpenTelemetry Observability

**Related Plan**: 2026-02-03-go-telemetry-middleware-plan.instructions.md
**Implementation Date**: 2026-02-03

## Summary

Implemented `TelemetryMiddleware` and `FunctionTelemetryMiddleware` as composable middleware for OpenTelemetry instrumentation of agent and tool invocations. The middleware integrates with the existing observability infrastructure and follows established patterns.

## Changes

### Added

* [go/observability/agent_middleware.go](../../go/observability/agent_middleware.go) - TelemetryMiddleware implementing AgentMiddleware interface with OpenTelemetry span creation and metrics recording
* [go/observability/function_middleware.go](../../go/observability/function_middleware.go) - FunctionTelemetryMiddleware implementing FunctionMiddleware interface for tool call instrumentation
* [go/observability/agent_middleware_test.go](../../go/observability/agent_middleware_test.go) - Unit tests for TelemetryMiddleware with mock agent implementation
* [go/observability/function_middleware_test.go](../../go/observability/function_middleware_test.go) - Unit tests for FunctionTelemetryMiddleware

### Modified

* [go/README.md](../../go/README.md) - Added Observability Middleware section documenting TelemetryMiddleware and FunctionTelemetryMiddleware usage

### Removed

None

## Additional or Deviating Changes

* Renamed option functions to avoid conflicts with existing `instrumented.go` functions:
  * `WithMetrics` → `WithTelemetryMetrics`
  * `WithSensitiveData` → `WithTelemetrySensitiveData`
  * `errorTypeName` → `agentErrorTypeName`
  * `finishReasonToString` → `agentFinishReasonToString`
  * Reason: The existing `instrumented.go` already defines these functions for `InstrumentedClient`, and they work with `chat.FinishReason` rather than `agent.FinishReason`

## Release Summary

**Total files affected**: 5 (4 created, 1 modified)

**Files created**:
* `go/observability/agent_middleware.go` - TelemetryMiddleware with Process method for agent invocation tracing
* `go/observability/function_middleware.go` - FunctionTelemetryMiddleware for tool call tracing
* `go/observability/agent_middleware_test.go` - 9 unit tests for TelemetryMiddleware
* `go/observability/function_middleware_test.go` - 7 unit tests for FunctionTelemetryMiddleware

**Files modified**:
* `go/README.md` - Added documentation for middleware-based observability

**Dependencies**: No new dependencies added; uses existing go.opentelemetry.io/otel packages

**Validation**:
* `go build ./...` - Passed
* `go test ./observability/...` - All tests pass
* `go vet ./observability/...` - No issues
* `go test ./agent/...` - Agent package tests still pass
