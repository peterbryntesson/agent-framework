---
applyTo: '.copilot-tracking/changes/2026-02-03-go-chatmiddleware-changes.md'
---
<!-- markdownlint-disable-file -->
# Implementation Plan: Go ChatMiddleware for Chat Client Interception

## Overview

Implement ChatMiddleware for the Go agent framework to intercept chat client requests (GetResponse/GetStreamingResponse), completing the middleware triad (AgentMiddleware, FunctionMiddleware, ChatMiddleware) identified in the middleware patterns research.

## Objectives

* Add ChatMiddleware interface to intercept chat.Client requests
* Add ChatContext struct to hold chat invocation context
* Integrate ChatMiddleware into chatagent's tool loop for request interception
* Provide ChatMiddlewareFunc adapter for function-based middleware
* Support both streaming and non-streaming invocations via context flags
* Align conceptually with Python's ChatMiddleware while remaining idiomatic Go

## Context Summary

### Project Files

* [go/agent/middleware.go](go/agent/middleware.go) - Existing AgentMiddleware and FunctionMiddleware interfaces (pattern reference)
* [go/agent/middleware_context.go](go/agent/middleware_context.go) - Existing AgentContext and FunctionContext structs (pattern reference)
* [go/agent/chain.go](go/agent/chain.go) - Existing middleware chain composition functions
* [go/chat/client.go](go/chat/client.go) - Chat Client interface with GetResponse/GetStreamingResponse
* [go/chat/response.go](go/chat/response.go) - Response and ResponseUpdate types
* [go/chatagent/toolloop.go](go/chatagent/toolloop.go) - Tool loop calling chat client (integration point)
* [go/chatagent/options.go](go/chatagent/options.go) - Functional options for agent configuration

### References

* [.copilot-tracking/research/2026-02-03-go-middleware-patterns-research.md](.copilot-tracking/research/2026-02-03-go-middleware-patterns-research.md) - Original middleware research identifying ChatMiddleware as follow-up
* [.copilot-tracking/reviews/2026-02-03-go-middleware-patterns-review.md](.copilot-tracking/reviews/2026-02-03-go-middleware-patterns-review.md) - Review confirming ChatMiddleware as deferred work
* [.copilot-tracking/subagent/2026-02-03/python-chatmiddleware-research.md](.copilot-tracking/subagent/2026-02-03/python-chatmiddleware-research.md) - Python ChatMiddleware implementation reference

### Standards References

* #file:../../.github/copilot-instructions.md - Go code conventions and copyright requirements

## Implementation Checklist

### [x] Implementation Phase 1: ChatMiddleware Core Types

<!-- parallelizable: false -->

* [x] Step 1.1: Create ChatContext struct in agent package
  * Details: .copilot-tracking/details/2026-02-03-go-chatmiddleware-details.md (Lines 30-72)
* [x] Step 1.2: Create ChatMiddleware interface in agent package
  * Details: .copilot-tracking/details/2026-02-03-go-chatmiddleware-details.md (Lines 74-110)
* [x] Step 1.3: Add ChatMiddlewareFunc adapter
  * Details: .copilot-tracking/details/2026-02-03-go-chatmiddleware-details.md (Lines 112-130)
* [x] Step 1.4: Add ChainChatMiddleware composition function
  * Details: .copilot-tracking/details/2026-02-03-go-chatmiddleware-details.md (Lines 132-165)
* [x] Step 1.5: Add unit tests for ChatMiddleware types
  * Details: .copilot-tracking/details/2026-02-03-go-chatmiddleware-details.md (Lines 167-210)
* [x] Step 1.6: Validate phase changes
  * Run `go build ./agent/...` and `go vet ./agent/...`
  * Run `go test ./agent/...`

### [x] Implementation Phase 2: ChatAgent Integration

<!-- parallelizable: false -->

* [x] Step 2.1: Add ChatMiddleware options to chatagent package
  * Details: .copilot-tracking/details/2026-02-03-go-chatmiddleware-details.md (Lines 215-255)
* [x] Step 2.2: Integrate ChatMiddleware into runWithToolLoop
  * Details: .copilot-tracking/details/2026-02-03-go-chatmiddleware-details.md (Lines 257-320)
* [x] Step 2.3: Integrate ChatMiddleware into runStreamWithToolLoop
  * Details: .copilot-tracking/details/2026-02-03-go-chatmiddleware-details.md (Lines 322-385)
* [x] Step 2.4: Add builder support for ChatMiddleware
  * Details: .copilot-tracking/details/2026-02-03-go-chatmiddleware-details.md (Lines 387-425)
* [x] Step 2.5: Add integration tests for ChatMiddleware
  * Details: .copilot-tracking/details/2026-02-03-go-chatmiddleware-details.md (Lines 427-490)
* [x] Step 2.6: Validate phase changes
  * Run `go build ./chatagent/...` and `go vet ./chatagent/...`
  * Run `go test ./chatagent/...`

### [x] Implementation Phase 3: Documentation

<!-- parallelizable: true -->

* [x] Step 3.1: Update go/agent/doc.go with ChatMiddleware documentation
  * Details: .copilot-tracking/details/2026-02-03-go-chatmiddleware-details.md (Lines 495-530)
* [x] Step 3.2: Update go/README.md with ChatMiddleware section
  * Details: .copilot-tracking/details/2026-02-03-go-chatmiddleware-details.md (Lines 532-590)
* [x] Step 3.3: Validate documentation
  * Ensure doc comments pass `go doc` checks

### [x] Implementation Phase 4: Validation

<!-- parallelizable: false -->

* [x] Step 4.1: Run full project validation
  * Execute `go build ./...` for full build
  * Execute `go vet ./...` for static analysis
  * Execute `go test ./agent/... ./chatagent/...` for test coverage
* [x] Step 4.2: Fix minor validation issues
  * Iterate on lint errors and build warnings
  * Apply fixes directly when corrections are straightforward
* [x] Step 4.3: Report blocking issues
  * Document issues requiring additional research
  * Provide user with next steps and recommended planning
  * Avoid large-scale fixes within this phase

## Dependencies

* Existing AgentMiddleware/FunctionMiddleware patterns in go/agent/middleware.go
* Existing chain composition in go/agent/chain.go
* chat.Client interface in go/chat/client.go
* chatagent.Agent implementation in go/chatagent/

## Success Criteria

* ChatMiddleware interface follows same pattern as AgentMiddleware and FunctionMiddleware
* ChatContext captures chat client, messages, options, and response fields
* ChatMiddleware can intercept all chat client calls in chatagent's tool loop
* Both streaming and non-streaming invocations are supported
* Middleware can modify requests, observe responses, or short-circuit execution
* All validation commands pass (`go build`, `go vet`, `go test`)
* Documentation updated in README.md and doc.go
