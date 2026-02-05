---
applyTo: '.copilot-tracking/changes/2026-02-03-go-middleware-patterns-changes.md'
---
<!-- markdownlint-disable-file -->
# Implementation Plan: Go Middleware Patterns for Agent Framework

## Overview

Implement middleware patterns for the Go agent framework including AgentMiddleware, FunctionMiddleware, DelegatingAgent, Builder.Use() method, and AsTool() conversion to enable flexible agent composition and interception.

## Objectives

* Define idiomatic Go middleware interfaces for agent and function invocation interception
* Implement DelegatingAgent base type for the decorator pattern
* Extend AgentBuilder with Use() method for middleware chaining
* Implement AsTool() function for agent-to-tool conversion enabling hierarchical agents
* Maintain backward compatibility with existing agent.Agent interface

## Context Summary

### Project Files

* [go/agent/agent.go](go/agent/agent.go) - Core Agent interface definition (target for new middleware types)
* [go/chatagent/agent.go](go/chatagent/agent.go) - ChatClientAgent implementation (middleware integration point)
* [go/chatagent/builder.go](go/chatagent/builder.go) - Existing fluent builder API (extend with Use() method)
* [go/chatagent/toolloop.go](go/chatagent/toolloop.go) - Tool invocation loop (FunctionMiddleware insertion point)
* [go/tool/tool.go](go/tool/tool.go) - Tool interface definition (target for AsTool result)

### References

* [.copilot-tracking/research/2026-02-03-go-middleware-patterns-research.md](.copilot-tracking/research/2026-02-03-go-middleware-patterns-research.md) - Comprehensive research on cross-platform patterns
* [dotnet/src/Microsoft.Agents.AI.Abstractions/DelegatingAIAgent.cs](dotnet/src/Microsoft.Agents.AI.Abstractions/DelegatingAIAgent.cs) - .NET decorator pattern reference
* [python/packages/core/agent_framework/_middleware.py](python/packages/core/agent_framework/_middleware.py) - Python middleware system reference

### Standards References

* #file:../../.github/copilot-instructions.md - Go code guidelines under `go/` directory

## Implementation Checklist

### [x] Implementation Phase 1: Core Middleware Interfaces

<!-- parallelizable: true -->

* [x] Step 1.1: Create middleware context types in agent package
  * Details: .copilot-tracking/details/2026-02-03-go-middleware-patterns-details.md (Lines 23-72)
* [x] Step 1.2: Create AgentMiddleware interface
  * Details: .copilot-tracking/details/2026-02-03-go-middleware-patterns-details.md (Lines 74-108)
* [x] Step 1.3: Create FunctionMiddleware interface
  * Details: .copilot-tracking/details/2026-02-03-go-middleware-patterns-details.md (Lines 110-143)
* [x] Step 1.4: Validate phase changes
  * Run `go build ./agent/...` and `go vet ./agent/...`

### [x] Implementation Phase 2: DelegatingAgent Implementation

<!-- parallelizable: true -->

* [x] Step 2.1: Implement DelegatingAgent base type
  * Details: .copilot-tracking/details/2026-02-03-go-middleware-patterns-details.md (Lines 145-205)
* [x] Step 2.2: Add unit tests for DelegatingAgent
  * Details: .copilot-tracking/details/2026-02-03-go-middleware-patterns-details.md (Lines 207-240)
* [x] Step 2.3: Validate phase changes
  * Run `go test ./agent/...`

### [x] Implementation Phase 3: Middleware Chain Implementation

<!-- parallelizable: false -->

* [x] Step 3.1: Implement middleware chain composition
  * Details: .copilot-tracking/details/2026-02-03-go-middleware-patterns-details.md (Lines 242-290)
* [x] Step 3.2: Create MiddlewareAgent decorator
  * Details: .copilot-tracking/details/2026-02-03-go-middleware-patterns-details.md (Lines 292-350)
* [x] Step 3.3: Add middleware chain tests
  * Details: .copilot-tracking/details/2026-02-03-go-middleware-patterns-details.md (Lines 352-395)

### [x] Implementation Phase 4: AgentBuilder.Use() Extension

<!-- parallelizable: false -->

* [x] Step 4.1: Add AgentFactory type and Use() method to chatagent.Builder
  * Details: .copilot-tracking/details/2026-02-03-go-middleware-patterns-details.md (Lines 397-445)
* [x] Step 4.2: Create standalone AgentBuilder in agent package
  * Details: .copilot-tracking/details/2026-02-03-go-middleware-patterns-details.md (Lines 447-505)
* [x] Step 4.3: Add builder tests
  * Details: .copilot-tracking/details/2026-02-03-go-middleware-patterns-details.md (Lines 507-545)
* [x] Step 4.4: Validate phase changes
  * Run `go test ./agent/... ./chatagent/...`

### [x] Implementation Phase 5: FunctionMiddleware Integration

<!-- parallelizable: false -->

* [x] Step 5.1: Extend InvocationConfig to accept FunctionMiddleware
  * Details: .copilot-tracking/details/2026-02-03-go-middleware-patterns-details.md (Lines 547-585)
* [x] Step 5.2: Integrate FunctionMiddleware into toolloop
  * Details: .copilot-tracking/details/2026-02-03-go-middleware-patterns-details.md (Lines 587-640)
* [x] Step 5.3: Add FunctionMiddleware tests
  * Details: .copilot-tracking/details/2026-02-03-go-middleware-patterns-details.md (Lines 642-680)

### [x] Implementation Phase 6: AsTool() Implementation

<!-- parallelizable: true -->

* [x] Step 6.1: Create AsToolOptions type and AsTool() function
  * Details: .copilot-tracking/details/2026-02-03-go-middleware-patterns-details.md (Lines 682-750)
* [x] Step 6.2: Add AsTool tests with streaming scenarios
  * Details: .copilot-tracking/details/2026-02-03-go-middleware-patterns-details.md (Lines 752-805)
* [x] Step 6.3: Validate phase changes
  * Run `go test ./chatagent/...`

### [x] Implementation Phase 7: Documentation and Examples

<!-- parallelizable: true -->

* [x] Step 7.1: Update go/README.md with middleware documentation
  * Details: .copilot-tracking/details/2026-02-03-go-middleware-patterns-details.md (Lines 807-845)
* [x] Step 7.2: Add doc.go comments for new types
  * Details: .copilot-tracking/details/2026-02-03-go-middleware-patterns-details.md (Lines 847-880)

### [x] Implementation Phase 8: Validation

<!-- parallelizable: false -->

* [x] Step 8.1: Run full project validation
  * Execute `go build ./...`
  * Execute `go vet ./...`
  * Execute `go test ./...`
  * Execute `golint ./...` if available
* [x] Step 8.2: Fix minor validation issues
  * Iterate on lint errors and build warnings
  * Apply fixes directly when corrections are straightforward
* [x] Step 8.3: Report blocking issues
  * Document issues requiring additional research
  * Provide user with next steps and recommended planning
  * Avoid large-scale refactoring within this phase

## Dependencies

* Go 1.21+ (target version per research)
* Existing agent.Agent interface must remain backward compatible
* github.com/google/uuid (already in use)
* encoding/json standard library
* context standard library
* reflect standard library

## Success Criteria

* All new middleware interfaces compile and pass vet checks
* DelegatingAgent correctly forwards all Agent interface methods
* Builder.Use() chains decorators in correct order (first = outermost)
* AsTool() converts agents to functional tools usable by other agents
* Existing tests continue to pass without modification
* New tests achieve >80% coverage for middleware code
* Documentation provides clear usage examples
