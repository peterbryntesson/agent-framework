---
applyTo: '.copilot-tracking/changes/2026-02-03-go-context-provider-changes.md'
---
<!-- markdownlint-disable-file -->
# Implementation Plan: Go ContextProvider Pattern for Dynamic Context Injection

## Overview

Implement the ContextProvider pattern for the Go agent framework, enabling dynamic context injection (instructions, messages, tools) before agent invocations with lifecycle hooks for session tracking.

## Objectives

* Define ContextProvider interface matching Python pattern semantics
* Implement Context struct for returning dynamic instructions, messages, and tools
* Create AggregateContextProvider for combining multiple providers
* Integrate ContextProvider into ChatClientAgent run flow
* Add WithContextProvider option to chatagent builder
* Provide lifecycle hooks (Invoking, Invoked, SessionCreated) for state management

## Context Summary

### Project Files

* [go/agent/middleware_context.go](go/agent/middleware_context.go) - Existing context patterns (AgentContext, ChatContext)
* [go/chatagent/agent.go](go/chatagent/agent.go) - Agent.Run entry point and message preparation
* [go/chatagent/options.go](go/chatagent/options.go) - Functional option patterns (WithXxx)
* [go/chatagent/session.go](go/chatagent/session.go) - Session implementation
* [go/chatagent/toolloop.go](go/chatagent/toolloop.go) - Tool invocation loop
* [go/agent/session.go](go/agent/session.go) - Session interface definition
* [go/chat/message.go](go/chat/message.go) - Message type definitions
* [go/tool/tool.go](go/tool/tool.go) - Tool interface definition

### References

* [.copilot-tracking/research/2026-02-03-go-middleware-followup-research.md](.copilot-tracking/research/2026-02-03-go-middleware-followup-research.md) - ContextProvider design research
* [.copilot-tracking/subagent/2026-02-03/python-context-provider-analysis.md](.copilot-tracking/subagent/2026-02-03/python-context-provider-analysis.md) - Python ContextProvider analysis
* [.copilot-tracking/subagent/2026-02-03/context-provider-codebase-research.md](.copilot-tracking/subagent/2026-02-03/context-provider-codebase-research.md) - Go codebase integration points
* [python/packages/core/agent_framework/_memory.py](python/packages/core/agent_framework/_memory.py) - Python ContextProvider reference

### Standards References

* #file:../../.github/copilot-instructions.md - Go code guidelines for this repository

## Implementation Checklist

### [x] Implementation Phase 1: Context and ContextProvider Interfaces

<!-- parallelizable: true -->

* [x] Step 1.1: Create Context struct in agent package
  * Details: .copilot-tracking/details/2026-02-03-go-context-provider-details.md (Lines 22-58)
* [x] Step 1.2: Create ContextProvider interface with required Invoking method
  * Details: .copilot-tracking/details/2026-02-03-go-context-provider-details.md (Lines 60-110)
* [x] Step 1.3: Create optional lifecycle interface ContextProviderWithLifecycle
  * Details: .copilot-tracking/details/2026-02-03-go-context-provider-details.md (Lines 112-158)
* [x] Step 1.4: Add unit tests for Context struct
  * Details: .copilot-tracking/details/2026-02-03-go-context-provider-details.md (Lines 160-195)
* [x] Step 1.5: Validate phase changes
  * Run `go build ./agent/...` and `go vet ./agent/...`

### [x] Implementation Phase 2: AggregateContextProvider Implementation

<!-- parallelizable: true -->

* [x] Step 2.1: Implement AggregateContextProvider struct
  * Details: .copilot-tracking/details/2026-02-03-go-context-provider-details.md (Lines 199-270)
* [x] Step 2.2: Implement concurrent provider invocation with goroutines
  * Details: .copilot-tracking/details/2026-02-03-go-context-provider-details.md (Lines 272-330)
* [x] Step 2.3: Implement context merging logic (concatenate instructions, extend messages/tools)
  * Details: .copilot-tracking/details/2026-02-03-go-context-provider-details.md (Lines 332-385)
* [x] Step 2.4: Add unit tests for AggregateContextProvider
  * Details: .copilot-tracking/details/2026-02-03-go-context-provider-details.md (Lines 387-450)
* [x] Step 2.5: Validate phase changes
  * Run `go test ./agent/...`

### [x] Implementation Phase 3: ChatClientAgent Integration

<!-- parallelizable: false -->

* [x] Step 3.1: Add contextProviders field to chatagent.config struct
  * Details: .copilot-tracking/details/2026-02-03-go-context-provider-details.md (Lines 454-490)
* [x] Step 3.2: Add WithContextProvider option function
  * Details: .copilot-tracking/details/2026-02-03-go-context-provider-details.md (Lines 492-530)
* [x] Step 3.3: Store contextProviders in Agent struct
  * Details: .copilot-tracking/details/2026-02-03-go-context-provider-details.md (Lines 532-570)
* [x] Step 3.4: Validate phase changes
  * Run `go build ./chatagent/...`

### [x] Implementation Phase 4: Context Injection in Agent.Run

<!-- parallelizable: false -->

* [x] Step 4.1: Create getProviderContext helper method on Agent
  * Details: .copilot-tracking/details/2026-02-03-go-context-provider-details.md (Lines 574-630)
* [x] Step 4.2: Modify prepareMessages to inject provider context
  * Details: .copilot-tracking/details/2026-02-03-go-context-provider-details.md (Lines 632-695)
* [x] Step 4.3: Modify prepareChatOptions to inject provider tools
  * Details: .copilot-tracking/details/2026-02-03-go-context-provider-details.md (Lines 697-750)
* [x] Step 4.4: Add integration tests for context injection
  * Details: .copilot-tracking/details/2026-02-03-go-context-provider-details.md (Lines 752-820)
* [x] Step 4.5: Validate phase changes
  * Run `go test ./chatagent/...`

### [x] Implementation Phase 5: Lifecycle Hooks Implementation

<!-- parallelizable: false -->

* [x] Step 5.1: Call Invoked hook after agent response
  * Details: .copilot-tracking/details/2026-02-03-go-context-provider-details.md (Lines 824-880)
* [x] Step 5.2: Call SessionCreated hook when session is created
  * Details: .copilot-tracking/details/2026-02-03-go-context-provider-details.md (Lines 882-935)
* [x] Step 5.3: Add lifecycle hook tests
  * Details: .copilot-tracking/details/2026-02-03-go-context-provider-details.md (Lines 937-1000)
* [x] Step 5.4: Validate phase changes
  * Run `go test ./chatagent/...`

### [x] Implementation Phase 6: Documentation and Samples

<!-- parallelizable: true -->

* [x] Step 6.1: Update go/README.md with ContextProvider section
  * Details: .copilot-tracking/details/2026-02-03-go-context-provider-details.md (Lines 1004-1060)
* [x] Step 6.2: Add godoc comments to all exported types
  * Details: .copilot-tracking/details/2026-02-03-go-context-provider-details.md (Lines 1062-1100)
* [x] Step 6.3: Create sample context provider implementations
  * Details: .copilot-tracking/details/2026-02-03-go-context-provider-details.md (Lines 1102-1180)

### [x] Implementation Phase 7: Validation

<!-- parallelizable: false -->

* [x] Step 7.1: Run full project validation
  * Execute `go build ./...`
  * Execute `go test ./...`
  * Execute `go vet ./...`
* [x] Step 7.2: Fix minor validation issues
  * Iterate on lint errors and build warnings
  * Apply fixes directly when corrections are straightforward
* [x] Step 7.3: Report blocking issues
  * Document issues requiring additional research
  * Provide user with next steps and recommended planning
  * Avoid large-scale fixes within this phase

## Dependencies

* Go 1.21+ (for generics and slices package)
* Existing go/agent and go/chatagent packages
* Existing go/chat and go/tool packages

## Success Criteria

* ContextProvider interface enables dynamic instruction/message/tool injection
* AggregateContextProvider correctly merges multiple provider outputs concurrently
* ChatClientAgent invokes providers before each Run/RunStream call
* Lifecycle hooks (Invoked, SessionCreated) are called at appropriate times
* All existing tests continue to pass
* New unit tests achieve >80% coverage for context_provider.go
* README.md documents ContextProvider usage with examples
