<!-- markdownlint-disable-file -->
# Release Changes: Go Epic 4 - Workflow Orchestration and Protocols

**Related Plan**: 2026-02-04-go-epic4-workflow-protocols-plan.instructions.md
**Implementation Date**: 2026-02-04

## Summary

Implementing comprehensive workflow orchestration and communication protocols for the Go Agent Framework SDK, including DAG-based workflow engine, A2A agent-to-agent protocol, AG-UI streaming protocol, and group chat orchestration with pluggable selection strategies.

## Changes

### Added

* go/workflow/doc.go - Package documentation for workflow package with Pregel-like execution model overview
* go/workflow/context.go - WorkflowContext struct providing execution context for executors with state management, message passing, and outbox functionality
* go/workflow/executor.go - Executor interface, ExecutorFunc convenience wrapper, and ExecutorBase embeddable type
* go/workflow/edge.go - Edge types including direct edges, conditional edges, fan-out/fan-in EdgeGroup, and SwitchEdge for case-based routing
* go/workflow/workflow.go - Workflow struct with executor management, edge traversal, target resolution, and validation
* go/workflow/events.go - WorkflowResult and WorkflowEvent types for workflow execution results and streaming events
* go/workflow/runner.go - WorkflowRunner stub with RunnerOption functional options (placeholder for Phase 3 implementation)
* go/workflow/checkpoint.go - Checkpoint struct and CheckpointStore interface (placeholder for Phase 4 implementation)
* go/workflow/context_test.go - Unit tests for WorkflowContext including concurrent access tests
* go/workflow/executor_test.go - Unit tests for Executor interface and implementations
* go/workflow/edge_test.go - Unit tests for Edge types including SwitchEdge evaluation
* go/workflow/workflow_test.go - Unit tests for Workflow struct including validation and target resolution
* go/workflow/builder.go - WorkflowBuilder fluent API with AddExecutor, AddEdge, AddConditionalEdge, AddFanOut, AddFanIn, SwitchFrom, MarkAsOutput, and Build methods
* go/workflow/builder_test.go - Comprehensive unit tests for WorkflowBuilder (26 test cases covering fluent API, validation, and complex workflows)

### Modified

### Removed

## Additional or Deviating Changes

* Created runner.go and checkpoint.go stub files with placeholder implementations
  * Required for workflow.go to compile since Workflow.Run/RunStream reference WorkflowRunner
  * Full implementations will be added in Phase 3 (runner) and Phase 4 (checkpointing)

## Release Summary

