---
applyTo: '.copilot-tracking/changes/2026-02-03-go-epic3-revised-changes.md'
---
<!-- markdownlint-disable-file -->
# Implementation Plan: Go Port Epic 3 - Revised Advanced Agent Features

## Overview

Revised implementation plan for Epic 3 based on research findings showing that Features 3.1 (Middleware) and 3.4 (ChatClientAgent) are complete. This plan focuses on the remaining work: SessionStore abstraction and TextSearchProvider for RAG.

## Objectives

* Complete Feature 3.3: Add SessionStore interface and InMemorySessionStore implementation
* Complete Feature 3.5: Implement TextSearchProvider for client-side RAG
* Update original Epic 3 documentation to reflect actual completion status
* Maintain 90%+ test coverage for all new code

## Context Summary

### Research Files

* [.copilot-tracking/research/2026-02-03-go-epic3-advanced-features-research.md](../../research/2026-02-03-go-epic3-advanced-features-research.md) - Main research findings
* [.copilot-tracking/subagent/2026-02-03/go-middleware-status-research.md](../../subagent/2026-02-03/go-middleware-status-research.md) - Middleware completion evidence
* [.copilot-tracking/subagent/2026-02-03/thread-session-management-research.md](../../subagent/2026-02-03/thread-session-management-research.md) - Session patterns

### Already Implemented (No Work Required)

| Feature | Status | Evidence |
|---------|--------|----------|
| 3.1 Middleware Pipeline | ✅ Complete | All middleware interfaces, chaining, builder integration, telemetry |
| 3.2 Context Providers (Core) | ✅ Complete | ContextProvider, AggregateContextProvider, lifecycle hooks |
| 3.4 ChatClientAgent | ✅ Complete | 382-line agent, 697-line toolloop, sessions, AsTool |

### Standards References

* #file:../../.github/copilot-instructions.md - Go code guidelines
* Reference: [dotnet/src/Microsoft.Agents.AI.Hosting/AgentSessionStore.cs](dotnet/src/Microsoft.Agents.AI.Hosting/AgentSessionStore.cs)
* Reference: [dotnet/src/Microsoft.Agents.AI/TextSearchProvider.cs](dotnet/src/Microsoft.Agents.AI/TextSearchProvider.cs)

## Implementation Checklist

### [x] Implementation Phase 1: Documentation Update

<!-- parallelizable: true -->

* [x] Step 1.1: Mark completed features in Epic 3 plan
  * Update Feature 3.1 status to COMPLETE
  * Update Feature 3.4 status to COMPLETE
  * Note: Research document already captures this; plan updates optional

### [ ] Implementation Phase 2: SessionStore Interface

<!-- parallelizable: false -->

* [ ] Step 2.1: Create hosting package structure
  * Details: .copilot-tracking/details/2026-02-03-go-epic3-revised-details.md (Lines 17-52)
* [ ] Step 2.2: Define SessionStore interface
  * Details: .copilot-tracking/details/2026-02-03-go-epic3-revised-details.md (Lines 54-125)
* [ ] Step 2.3: Implement InMemorySessionStore
  * Details: .copilot-tracking/details/2026-02-03-go-epic3-revised-details.md (Lines 127-229)
* [ ] Step 2.4: Implement NoopSessionStore
  * Details: .copilot-tracking/details/2026-02-03-go-epic3-revised-details.md (Lines 231-297)
* [ ] Step 2.5: Update Agent interface for session deserialization
  * Details: .copilot-tracking/details/2026-02-03-go-epic3-revised-details.md (Lines 299-334)
* [ ] Step 2.6: Write unit tests for SessionStore
  * Details: .copilot-tracking/details/2026-02-03-go-epic3-revised-details.md (Lines 336-529)
* [ ] Step 2.7: Validate Phase 2 changes
  * Run `go build ./...` and `go test ./hosting/...`
  * Verify 90%+ coverage on hosting package

### [ ] Implementation Phase 3: TextSearchProvider

<!-- parallelizable: false -->

* [ ] Step 3.1: Define SearchResult and SearchFunc types
  * Details: .copilot-tracking/details/2026-02-03-go-epic3-revised-details.md (Lines 531-589)
* [ ] Step 3.2: Define TextSearchProviderOptions
  * Details: .copilot-tracking/details/2026-02-03-go-epic3-revised-details.md (Lines 591-740)
* [ ] Step 3.3: Implement TextSearchProvider core
  * Details: .copilot-tracking/details/2026-02-03-go-epic3-revised-details.md (Lines 742-925)
* [ ] Step 3.4: Implement BeforeAIInvoke behavior
  * Details: .copilot-tracking/details/2026-02-03-go-epic3-revised-details.md (Lines 927-1012)
* [ ] Step 3.5: Implement OnDemandFunctionCalling behavior
  * Details: .copilot-tracking/details/2026-02-03-go-epic3-revised-details.md (Lines 1014-1043)
* [ ] Step 3.6: Implement state serialization
  * Details: .copilot-tracking/details/2026-02-03-go-epic3-revised-details.md (Lines 1045-1148)
* [ ] Step 3.7: Write unit tests for TextSearchProvider
  * Details: .copilot-tracking/details/2026-02-03-go-epic3-revised-details.md (Lines 1150-1189)
* [ ] Step 3.8: Write integration tests with ChatClientAgent
  * Details: .copilot-tracking/details/2026-02-03-go-epic3-revised-details.md (Lines 1191-1210)
* [ ] Step 3.9: Validate Phase 3 changes
  * Run `go build ./...` and `go test ./...`
  * Verify 90%+ coverage on provider package

### [ ] Implementation Phase 4: Final Validation

<!-- parallelizable: false -->

* [ ] Step 4.1: Run full project validation
  * Execute `go build ./...` for all packages
  * Execute `go test -cover ./...` for all tests
  * Execute `go vet ./...` and linting via `golangci-lint run`
* [ ] Step 4.2: Fix minor validation issues
  * Iterate on lint errors and test failures
  * Apply fixes directly when corrections are straightforward
* [ ] Step 4.3: Update package documentation
  * Add godoc comments to all public types
  * Update README.md with new features
* [ ] Step 4.4: Report blocking issues
  * Document issues requiring additional research
  * Provide next steps and recommended planning

## Dependencies

* Go 1.22+ with generics support
* Existing packages: `agent`, `chatagent`, `chat`, `tool`
* Standard library: `sync`, `encoding/json`, `context`
* No new external dependencies required

## Success Criteria

* SessionStore interface with SaveSession, GetSession, DeleteSession methods
* InMemorySessionStore using sync.Map for thread-safe storage
* NoopSessionStore for testing and stateless scenarios
* TextSearchProvider implementing ContextProviderWithLifecycle
* BeforeAIInvoke mode with automatic context injection
* OnDemandFunctionCalling mode with search tool exposure
* 90%+ test coverage across all new code
* All public APIs documented with godoc
* Integration tests verifying end-to-end RAG flow
