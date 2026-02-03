---
applyTo: '.copilot-tracking/details/2026-02-03-go-telemetry-middleware-details.md'
---
<!-- markdownlint-disable-file -->
# Implementation Plan: TelemetryMiddleware for OpenTelemetry Observability

## Overview

Implement `TelemetryMiddleware` as an `AgentMiddleware` that provides OpenTelemetry instrumentation for agent invocations, enabling composable observability that can be chained with other middleware.

## Objectives

* Create `TelemetryMiddleware` implementing `AgentMiddleware` interface
* Provide functional options for configuration (sensitive data, metrics, source name)
* Integrate with existing `Metrics` struct and span functions from `go/observability/`
* Create corresponding `FunctionTelemetryMiddleware` for tool call instrumentation
* Add comprehensive unit tests following existing test patterns
* Update README documentation with usage examples

## Context Summary

### Project Files

* [go/observability/otel.go](../../go/observability/otel.go) - Span functions (`StartAgentSpan`, `StartChatSpan`, `RecordError`, `RecordUsage`)
* [go/observability/metrics.go](../../go/observability/metrics.go) - `Metrics` struct with `RecordAgentRun`, `RecordLatency`, `RecordError`, `RecordTokenUsage`
* [go/observability/semconv.go](../../go/observability/semconv.go) - Semantic conventions for GenAI attributes
* [go/agent/middleware.go](../../go/agent/middleware.go) - `AgentMiddleware` and `FunctionMiddleware` interfaces
* [go/agent/middleware_context.go](../../go/agent/middleware_context.go) - `AgentContext` and `FunctionContext` structs
* [go/observability/instrumented.go](../../go/observability/instrumented.go) - `InstrumentedClient` pattern reference

### References

* [2026-02-03-go-middleware-followup-research.md](../.copilot-tracking/research/2026-02-03-go-middleware-followup-research.md) - Design recommendations for TelemetryMiddleware
* OpenTelemetry Semantic Conventions for GenAI

### Standards References

* #file:../../.github/copilot-instructions.md - Repository coding conventions

## Implementation Checklist

### [x] Implementation Phase 1: TelemetryMiddleware Core

<!-- parallelizable: false -->

* [x] Step 1.1: Create TelemetryMiddleware struct and options
  * Details: .copilot-tracking/details/2026-02-03-go-telemetry-middleware-details.md (Lines 20-75)
* [x] Step 1.2: Implement Process method for AgentMiddleware interface
  * Details: .copilot-tracking/details/2026-02-03-go-telemetry-middleware-details.md (Lines 77-130)
* [x] Step 1.3: Add response and streaming attribute recording
  * Details: .copilot-tracking/details/2026-02-03-go-telemetry-middleware-details.md (Lines 132-175)

### [x] Implementation Phase 2: FunctionTelemetryMiddleware

<!-- parallelizable: true -->

* [x] Step 2.1: Create FunctionTelemetryMiddleware for tool call instrumentation
  * Details: .copilot-tracking/details/2026-02-03-go-telemetry-middleware-details.md (Lines 177-230)
* [x] Step 2.2: Integrate with existing tool span functions
  * Details: .copilot-tracking/details/2026-02-03-go-telemetry-middleware-details.md (Lines 232-270)

### [x] Implementation Phase 3: Unit Tests

<!-- parallelizable: true -->

* [x] Step 3.1: Create TelemetryMiddleware unit tests
  * Details: .copilot-tracking/details/2026-02-03-go-telemetry-middleware-details.md (Lines 272-340)
* [x] Step 3.2: Create FunctionTelemetryMiddleware unit tests
  * Details: .copilot-tracking/details/2026-02-03-go-telemetry-middleware-details.md (Lines 342-390)

### [x] Implementation Phase 4: Documentation

<!-- parallelizable: true -->

* [x] Step 4.1: Update go/README.md with TelemetryMiddleware usage examples
  * Details: .copilot-tracking/details/2026-02-03-go-telemetry-middleware-details.md (Lines 392-450)

### [x] Implementation Phase 5: Validation

<!-- parallelizable: false -->

* [x] Step 5.1: Run full project validation
  * Execute `go build ./...` for all packages
  * Execute `go test ./observability/...` for observability tests
  * Execute `go vet ./observability/...` for static analysis
* [x] Step 5.2: Fix minor validation issues
  * Iterate on lint errors and build warnings
  * Apply fixes directly when corrections are straightforward
* [x] Step 5.3: Report blocking issues
  * Document issues requiring additional research
  * Provide user with next steps and recommended planning

## Dependencies

* Go 1.21+
* go.opentelemetry.io/otel (already in go.mod)
* Existing `go/observability/` package components
* Existing `go/agent/` middleware interfaces

## Success Criteria

* `TelemetryMiddleware` compiles and passes all unit tests
* `FunctionTelemetryMiddleware` compiles and passes all unit tests
* Middleware integrates with existing `Metrics` and span functions
* README includes clear usage examples for middleware-based observability
* `go test ./...` passes without errors
