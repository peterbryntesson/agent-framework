<!-- markdownlint-disable-file -->
# Release Changes: Go Epic 5 Phase 5 - Durable Agents

**Related Plan**: 2026-02-04-go-epic5-implementation-plan.md
**Implementation Date**: 2026-02-05

## Summary

Implemented durable agent sessions for the Go Agent Framework using Temporal.io workflows. This enables persistent conversation sessions that survive process restarts and can be distributed across multiple workers.

## Changes

### Added

* go/durable/doc.go - Package documentation with usage examples
* go/durable/state.go - State types for persisted agent state matching cross-platform schema v1.1.0
* go/durable/state_entry.go - State entry types for requests, responses, and content items
* go/durable/sessionid.go - SessionID type for unique session identification with workflow ID generation
* go/durable/session.go - Durable session implementation maintaining conversation state
* go/durable/workflow.go - Temporal workflow for session management with update handlers
* go/durable/activity.go - Temporal activities for agent execution
* go/durable/agent.go - Durable agent wrapper implementing agent.Agent interface
* go/durable/worker.go - Temporal worker for hosting durable agent workflows
* go/durable/options.go - Configuration options for durable agents
* go/durable/state_test.go - Unit tests for state types
* go/durable/state_entry_test.go - Unit tests for state entry types
* go/durable/sessionid_test.go - Unit tests for session ID
* go/durable/session_test.go - Unit tests for durable session

### Modified

* go/go.mod - Added go.temporal.io/sdk v1.29.1 dependency
* go/go.sum - Updated with Temporal SDK and transitive dependencies

### Removed

* None

## Additional or Deviating Changes

* DataContent, URIContent, and ErrorContent types are not available in the Go chat package
  * Used nil fallback for these content types in ToChatContent
  * Serialization/deserialization handles these types for cross-platform compatibility
* Streaming through Temporal is not truly real-time
  * StreamingActivityResult collects all updates and returns them at once
  * RunStream method falls back to non-streaming Run and converts response to updates

## Release Summary

**Total Files Affected**: 16

**Files Created**:
* go/durable/doc.go - Package documentation
* go/durable/state.go - State persistence types
* go/durable/state_entry.go - Conversation history entry types
* go/durable/sessionid.go - Session identification
* go/durable/session.go - Session management
* go/durable/workflow.go - Temporal workflow definition
* go/durable/activity.go - Temporal activity implementations
* go/durable/agent.go - Durable agent wrapper
* go/durable/worker.go - Temporal worker management
* go/durable/options.go - Configuration options
* go/durable/state_test.go - State unit tests
* go/durable/state_entry_test.go - State entry unit tests
* go/durable/sessionid_test.go - SessionID unit tests
* go/durable/session_test.go - Session unit tests

**Files Modified**:
* go/go.mod - Added Temporal SDK dependency
* go/go.sum - Updated dependencies

**Dependencies Added**:
* go.temporal.io/sdk v1.29.1 - Temporal SDK for durable execution

**Deployment Notes**:
* Temporal server must be running and accessible (default: localhost:7233)
* Workers must be started to process durable agent requests
* Sessions persist in Temporal's storage backend
