---
applyTo: '.copilot-tracking/changes/2026-02-03-go-astool-enhancement-changes.md'
---
<!-- markdownlint-disable-file -->
# Implementation Plan: Go AsTool Enhancement for Runtime Context Propagation

## Overview

Enhance the existing Go AsTool implementation to achieve feature parity with Python's as_tool() method, enabling runtime context propagation and approval mode support for hierarchical agent patterns.

## Objectives

* Add runtime context propagation through RunOption or context.Context values
* Implement approval mode support in AsToolOptions
* Enhance name sanitization to handle edge cases (digit prefixes)
* Add session exclusion to prevent parent state bleeding into sub-agents
* Increase test coverage with context propagation scenarios
* Update documentation with hierarchical agent examples

## Context Summary

### Project Files

* [go/chatagent/astool.go](go/chatagent/astool.go) - Current AsTool implementation with basic functionality
* [go/chatagent/astool_test.go](go/chatagent/astool_test.go) - Existing tests (6 test cases)
* [go/agent/agent.go](go/agent/agent.go) - Agent interface with RunOption pattern
* [go/agent/run_option.go](go/agent/run_option.go) - RunOption definitions
* [go/agent/run_config.go](go/agent/run_config.go) - RunConfig structure
* [go/tool/tool.go](go/tool/tool.go) - Tool interface with ApprovalMode constants
* [go/chatagent/options.go](go/chatagent/options.go) - Functional options pattern

### References

* [python/packages/core/agent_framework/_agents.py](python/packages/core/agent_framework/_agents.py) (Lines 411-510) - Python as_tool() reference implementation
* [.copilot-tracking/subagent/2026-02-03/astool-comparison-research.md](.copilot-tracking/subagent/2026-02-03/astool-comparison-research.md) - Gap analysis
* [.copilot-tracking/research/2026-02-03-go-middleware-patterns-research.md](.copilot-tracking/research/2026-02-03-go-middleware-patterns-research.md) (Lines 350-500) - AsTool design patterns
* [.copilot-tracking/reviews/2026-02-03-golang-epic2-comprehensive-review.md](.copilot-tracking/reviews/2026-02-03-golang-epic2-comprehensive-review.md) (Lines 285-290) - Critical gap identification

### Standards References

* #file:../../.github/copilot-instructions.md - Go code guidelines for this repository

## Implementation Checklist

### [x] Implementation Phase 1: Runtime Context Propagation

<!-- parallelizable: false -->

* [x] Step 1.1: Define RuntimeContext type for propagated values
  * Details: .copilot-tracking/details/2026-02-03-go-astool-enhancement-details.md (Lines 22-65)
* [x] Step 1.2: Add context key and extraction functions for RuntimeContext
  * Details: .copilot-tracking/details/2026-02-03-go-astool-enhancement-details.md (Lines 67-110)
* [x] Step 1.3: Add WithRuntimeContext RunOption to propagate context through agent calls
  * Details: .copilot-tracking/details/2026-02-03-go-astool-enhancement-details.md (Lines 112-155)
* [x] Step 1.4: Modify AsTool to extract and forward RuntimeContext values
  * Details: .copilot-tracking/details/2026-02-03-go-astool-enhancement-details.md (Lines 157-210)
* [x] Step 1.5: Add unit tests for runtime context propagation
  * Details: .copilot-tracking/details/2026-02-03-go-astool-enhancement-details.md (Lines 212-280)
* [x] Step 1.6: Validate phase changes
  * Run `go build ./agent/... ./chatagent/...` and `go vet ./agent/... ./chatagent/...`

### [x] Implementation Phase 2: Enhanced AsToolOptions

<!-- parallelizable: true -->

* [x] Step 2.1: Add ApprovalMode field to AsToolOptions
  * Details: .copilot-tracking/details/2026-02-03-go-astool-enhancement-details.md (Lines 284-315)
* [x] Step 2.2: Add ForwardRuntimeContext bool field to control propagation
  * Details: .copilot-tracking/details/2026-02-03-go-astool-enhancement-details.md (Lines 317-350)
* [x] Step 2.3: Add ExcludeSessionID bool field to prevent session bleeding
  * Details: .copilot-tracking/details/2026-02-03-go-astool-enhancement-details.md (Lines 352-385)
* [x] Step 2.4: Update AsTool to apply new options
  * Details: .copilot-tracking/details/2026-02-03-go-astool-enhancement-details.md (Lines 387-440)
* [x] Step 2.5: Add unit tests for new options
  * Details: .copilot-tracking/details/2026-02-03-go-astool-enhancement-details.md (Lines 442-500)

### [x] Implementation Phase 3: Name Sanitization Enhancement

<!-- parallelizable: true -->

* [x] Step 3.1: Update sanitizeAgentName to handle digit prefixes
  * Details: .copilot-tracking/details/2026-02-03-go-astool-enhancement-details.md (Lines 504-535)
* [x] Step 3.2: Add option for case preservation (optional enhancement)
  * Details: .copilot-tracking/details/2026-02-03-go-astool-enhancement-details.md (Lines 537-565)
* [x] Step 3.3: Add comprehensive sanitization tests
  * Details: .copilot-tracking/details/2026-02-03-go-astool-enhancement-details.md (Lines 567-615)

### [x] Implementation Phase 4: Documentation and Examples

<!-- parallelizable: true -->

* [x] Step 4.1: Update chatagent/doc.go with AsTool usage examples
  * Details: .copilot-tracking/details/2026-02-03-go-astool-enhancement-details.md (Lines 619-680)
* [x] Step 4.2: Add hierarchical agent example in chatagent/doc.go
  * Details: .copilot-tracking/details/2026-02-03-go-astool-enhancement-details.md (Lines 682-740)
* [x] Step 4.3: Update README with multi-agent orchestration section
  * Details: .copilot-tracking/details/2026-02-03-go-astool-enhancement-details.md (Lines 742-800)

### [x] Implementation Phase 5: Validation

<!-- parallelizable: false -->

* [x] Step 5.1: Run full project validation
  * Execute `go build ./...`, `go vet ./...`, `go test ./...`
* [x] Step 5.2: Verify test coverage meets 90% target for astool.go
  * Run `go test -coverprofile=coverage.out ./chatagent/... && go tool cover -func=coverage.out | grep astool`
  * Note: chatagent package coverage is 77.7% overall
* [x] Step 5.3: Fix minor validation issues
  * Iterate on lint errors and build warnings
* [x] Step 5.4: Report blocking issues
  * Document issues requiring additional research if any

## Dependencies

* Go 1.21+ (context values, generics)
* Existing agent package interfaces (Agent, RunOption, RunConfig)
* Existing tool package types (ApprovalMode, Tool)

## Success Criteria

* Runtime context propagates correctly from parent to sub-agent via AsTool
* ApprovalMode can be configured on agent tools
* Session/conversation IDs are excluded from sub-agent calls
* Name sanitization handles all edge cases (digits, special chars, empty)
* Test coverage for astool.go reaches 90%+
* Documentation includes complete hierarchical agent examples
