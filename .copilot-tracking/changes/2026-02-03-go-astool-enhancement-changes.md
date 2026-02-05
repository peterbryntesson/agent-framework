<!-- markdownlint-disable-file -->
# Release Changes: Go AsTool Enhancement for Runtime Context Propagation

**Related Plan**: 2026-02-03-go-astool-enhancement-plan.instructions.md
**Implementation Date**: 2026-02-03

## Summary

Enhanced the Go AsTool implementation to achieve feature parity with Python's as_tool() method, enabling runtime context propagation and approval mode support for hierarchical agent patterns.

## Changes

### Added

* go/agent/runtime_context.go - New RuntimeContext type for carrying key-value pairs through agent delegation chains, with thread-safe access and immutability
* go/agent/runtime_context_test.go - Comprehensive unit tests for RuntimeContext type and context extraction functions

### Modified

* go/agent/options.go - Added RuntimeContext field to RunConfig, added WithRuntimeContext and WithRuntimeValue RunOption functions, updated WithRunConfig to forward RuntimeContext
* go/chatagent/astool.go - Enhanced AsToolOptions with ApprovalMode, ForwardRuntimeContext, ExcludeKeys, and PreserveCase fields; added excludeKey helper method; updated AsTool to forward runtime context with session key exclusion; added GetApprovalMode method to agentTool; enhanced sanitizeAgentName to handle digit prefixes
* go/chatagent/astool_test.go - Added tests for approval mode, runtime context propagation, session key exclusion, custom exclude keys, and digit prefix sanitization
* go/chatagent/doc.go - Added comprehensive documentation for Agent as Tool, Streaming Sub-Agents, Runtime Context Propagation, and Hierarchical Agent Orchestration patterns
* go/README.md - Added Multi-Agent Orchestration section with complete example

### Removed

## Additional or Deviating Changes

* Test coverage for chatagent package is 77.7% overall, which may be below the 90% target for astool.go specifically. Additional tests for error paths and edge cases could increase coverage.

## Release Summary

This release adds runtime context propagation to the Go AsTool implementation, enabling parent agents to forward context values (user IDs, API tokens, session data) to sub-agents without modifying function signatures.

**Files Created**: 2
* go/agent/runtime_context.go - RuntimeContext type and context key functions
* go/agent/runtime_context_test.go - Unit tests for RuntimeContext

**Files Modified**: 5
* go/agent/options.go - RuntimeContext field and RunOption functions
* go/chatagent/astool.go - Enhanced AsToolOptions and context forwarding logic
* go/chatagent/astool_test.go - New tests for all features
* go/chatagent/doc.go - Documentation updates
* go/README.md - Multi-agent orchestration documentation

**Key Features**:
* RuntimeContext type with immutable, thread-safe access
* WithRuntimeContext and WithRuntimeValue RunOption functions
* ForwardRuntimeContext option in AsToolOptions for context propagation
* Automatic exclusion of session-related keys (session_id, conversation_id, thread_id)
* Custom key exclusion via ExcludeKeys option
* ApprovalMode support in AsToolOptions
* Digit prefix handling in sanitizeAgentName

**Validation**:
* `go build ./...` - Passed
* `go vet ./...` - Passed
* `go test ./...` - All tests passed

