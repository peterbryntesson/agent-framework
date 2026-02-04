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
* go/workflow/checkpoint.go - Checkpoint struct and CheckpointStore interface (placeholder for Phase 4 implementation)
* go/workflow/context_test.go - Unit tests for WorkflowContext including concurrent access tests
* go/workflow/executor_test.go - Unit tests for Executor interface and implementations
* go/workflow/edge_test.go - Unit tests for Edge types including SwitchEdge evaluation
* go/workflow/workflow_test.go - Unit tests for Workflow struct including validation and target resolution
* go/workflow/builder.go - WorkflowBuilder fluent API with AddExecutor, AddEdge, AddConditionalEdge, AddFanOut, AddFanIn, SwitchFrom, MarkAsOutput, and Build methods
* go/workflow/builder_test.go - Comprehensive unit tests for WorkflowBuilder (26 test cases covering fluent API, validation, and complex workflows)
* go/workflow/runner_test.go - Comprehensive unit tests for WorkflowRunner (30+ test cases covering Run, RunStream, parallel execution, convergence, cancellation)

### Modified

* go/workflow/runner.go - Full WorkflowRunner implementation with Pregel-like superstep execution, message routing, convergence detection, and event streaming via channels
* go/workflow/workflow_test.go - Updated TestWorkflow_Run to work with actual runner implementation instead of stub
* go/workflow/checkpoint.go - Added InMemoryCheckpointStore implementation with Save, Load, LoadLatest, Delete, List methods; removed "Phase 4" placeholder comment
* go/workflow/runner.go - Added SaveCheckpoint, ResumeFromCheckpoint, and ResumeFromLatestCheckpoint methods for checkpoint persistence and recovery

### Added

* go/workflow/checkpoint_test.go - Comprehensive unit tests for checkpointing (14 test cases covering InMemoryCheckpointStore CRUD, thread-safety, WorkflowRunner checkpoint save/restore, and error cases)
* go/workflow/executors/doc.go - Package documentation for built-in executor implementations
* go/workflow/executors/agent.go - AgentExecutor wrapping agent.Agent for workflow participation
* go/workflow/executors/function.go - FunctionExecutor wrapping simple handler functions for transformations
* go/workflow/executors/aggregating.go - AggregatingExecutor for fan-in patterns collecting messages from multiple sources
* go/workflow/executors/agent_test.go - Unit tests for AgentExecutor (7 test cases)
* go/workflow/executors/function_test.go - Unit tests for FunctionExecutor (7 test cases)
* go/workflow/executors/aggregating_test.go - Unit tests for AggregatingExecutor (12 test cases)

### Modified

* go/workflow/context.go - Added NewWorkflowContextForTest helper for external package testing

### Removed

## Additional or Deviating Changes

* Phase 3 implementation replaced Phase 1 stub runner.go with full implementation
  * Phase 1 created placeholder that returned "not implemented" error
  * Phase 3 provides complete Pregel-like execution with parallel executor processing

* Phase 4 checkpoint.go already had Checkpoint struct and CheckpointStore interface from Phase 1
  * Step 4.1 was already complete, only needed InMemoryCheckpointStore implementation
  * Added runner methods for checkpoint integration

## Release Summary

