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
* go/protocol/a2a/doc.go - Package documentation for A2A protocol types with Client/Server usage examples
* go/protocol/a2a/types.go - A2A protocol type definitions (AgentCard, Task, Message, Part, Artifact, StreamEvent, request/response types, and helper constructors)
* go/protocol/a2a/types_test.go - Comprehensive unit tests for A2A type serialization (17 test cases covering JSON round-tripping for all types)
* go/protocol/a2a/client.go - A2A Client HTTP implementation with functional options, GetAgentCard, CreateTask, GetTask, SendMessage, SendMessageStream with SSE parsing, and CancelTask methods
* go/protocol/a2a/client_test.go - Comprehensive unit tests for A2A Client (24 test cases covering all client methods, SSE parsing, error handling, and context cancellation)
* go/protocol/a2a/agent.go - A2AAgent wrapper implementing agent.Agent interface using A2A Client for remote agent access with Run and RunStream methods
* go/protocol/a2a/session.go - A2ASession implementing agent.Session for maintaining A2A context ID and task ID across interactions
* go/protocol/a2a/agent_test.go - Comprehensive unit tests for A2AAgent (16 test cases covering agent creation, Run, RunStream, session management, and message conversion)
* go/protocol/a2a/session_test.go - Comprehensive unit tests for A2ASession (18 test cases covering session creation, serialization, thread-safety, and service registration)

### Modified

* go/workflow/context.go - Added NewWorkflowContextForTest helper for external package testing
* go/protocol/a2a/server.go - A2A Server HTTP implementation exposing agent.Agent via A2A protocol with AgentCard, Task management, SendMessage, SendMessageStream SSE streaming, and CancelTask endpoints
* go/protocol/a2a/server_test.go - Comprehensive unit tests for A2A Server (45+ test cases covering all endpoints, message conversion, session management, streaming, and integration flows)

### Removed

## Additional or Deviating Changes

* Phase 3 implementation replaced Phase 1 stub runner.go with full implementation
  * Phase 1 created placeholder that returned "not implemented" error
  * Phase 3 provides complete Pregel-like execution with parallel executor processing

* Phase 4 checkpoint.go already had Checkpoint struct and CheckpointStore interface from Phase 1
  * Step 4.1 was already complete, only needed InMemoryCheckpointStore implementation
  * Added runner methods for checkpoint integration

* Phase 8 session storage uses wrapper struct instead of raw slice
  * Go slices are not comparable, preventing use of sync.Map.CompareAndSwap
  * Added sessionTasks wrapper struct with mutex protection for thread-safe task ID management

* Phase 9 A2AAgent uses ImageContent instead of DataContent for file/data parts
  * chat package does not have a DataContent type
  * PartTypeFile maps to ImageContent for URL-based content, PartTypeData parts with structured data are currently not converted

