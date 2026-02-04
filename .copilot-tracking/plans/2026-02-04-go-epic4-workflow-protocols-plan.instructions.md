---
applyTo: '.copilot-tracking/changes/2026-02-04-go-epic4-workflow-protocols-changes.md'
---
<!-- markdownlint-disable-file -->
# Implementation Plan: Go Epic 4 - Workflow Orchestration and Protocols

## Overview

Implement comprehensive workflow orchestration and communication protocols for the Go Agent Framework SDK, including DAG-based workflow engine, A2A agent-to-agent protocol, AG-UI streaming protocol, and group chat orchestration with pluggable selection strategies.

## Objectives

* Implement Feature 4.1: Workflow Engine Package with Pregel-like DAG execution model
* Implement Feature 4.2: A2A Protocol client and server for agent-to-agent communication
* Implement Feature 4.3: AG-UI Protocol server for agent-to-UI streaming via SSE
* Implement Feature 4.4: Group Chat Orchestration with pluggable selector strategies
* Achieve 90%+ test coverage for all new packages
* Maintain Go idioms (interfaces, channels, functional options pattern)
* Ensure compatibility with existing Epic 1-3 foundations (agent interfaces, providers, middleware)

## Context Summary

### Research Files

* [.copilot-tracking/research/2026-02-04-go-epic4-workflow-protocols-research.md](../research/2026-02-04-go-epic4-workflow-protocols-research.md) - Comprehensive research with .NET/Python comparisons

### Project Files

* [go/agent/agent.go](../../go/agent/agent.go) - Core Agent interface to integrate with
* [go/agent/response.go](../../go/agent/response.go) - Response and ResponseUpdate types
* [go/agent/message.go](../../go/agent/message.go) - Message types and aliases
* [go/hosting/sessionstore.go](../../go/hosting/sessionstore.go) - Session store patterns

### External References

* A2A Protocol Specification: https://a2a-protocol.org/latest/
* AG-UI Protocol Documentation: https://docs.ag-ui.com/
* Pregel Paper: https://research.google/pubs/pregel-a-system-for-large-scale-graph-processing/

### Standards References

* #file:../../.github/copilot-instructions.md - Go code guidelines
* Reference: [dotnet/src/Microsoft.Agents.AI.Workflows/Executor.cs](../../dotnet/src/Microsoft.Agents.AI.Workflows/Executor.cs) - .NET Executor pattern
* Reference: [dotnet/src/Microsoft.Agents.AI.Workflows/WorkflowBuilder.cs](../../dotnet/src/Microsoft.Agents.AI.Workflows/WorkflowBuilder.cs) - .NET Builder pattern
* Reference: [dotnet/src/Microsoft.Agents.AI.Workflows/GroupChatManager.cs](../../dotnet/src/Microsoft.Agents.AI.Workflows/GroupChatManager.cs) - .NET Group Chat pattern
* Reference: [dotnet/src/Microsoft.Agents.AI.A2A/A2AAgent.cs](../../dotnet/src/Microsoft.Agents.AI.A2A/A2AAgent.cs) - .NET A2A implementation

## Implementation Checklist

### [x] Implementation Phase 1: Workflow Core Types and Interfaces

<!-- parallelizable: false -->

* [x] Step 1.1: Create workflow package structure
  * Details: .copilot-tracking/details/2026-02-04-go-epic4-workflow-protocols-details.md (Lines 30-75)
* [x] Step 1.2: Define WorkflowContext interface and struct
  * Details: .copilot-tracking/details/2026-02-04-go-epic4-workflow-protocols-details.md (Lines 77-150)
* [x] Step 1.3: Define Executor interface
  * Details: .copilot-tracking/details/2026-02-04-go-epic4-workflow-protocols-details.md (Lines 152-220)
* [x] Step 1.4: Define Edge types and conditions
  * Details: .copilot-tracking/details/2026-02-04-go-epic4-workflow-protocols-details.md (Lines 222-320)
* [x] Step 1.5: Define Workflow struct and initialization
  * Details: .copilot-tracking/details/2026-02-04-go-epic4-workflow-protocols-details.md (Lines 322-400)
* [x] Step 1.6: Write unit tests for core types
  * Details: .copilot-tracking/details/2026-02-04-go-epic4-workflow-protocols-details.md (Lines 402-460)
* [x] Step 1.7: Validate Phase 1 changes
  * Run `go build ./workflow/...` and `go test ./workflow/...`
  * Verify types compile and basic tests pass

### [x] Implementation Phase 2: Workflow Builder

<!-- parallelizable: false -->

* [x] Step 2.1: Implement WorkflowBuilder core structure
  * Details: .copilot-tracking/details/2026-02-04-go-epic4-workflow-protocols-details.md (Lines 470-560)
* [x] Step 2.2: Implement AddExecutor and edge methods
  * Details: .copilot-tracking/details/2026-02-04-go-epic4-workflow-protocols-details.md (Lines 562-660)
* [x] Step 2.3: Implement FanOut and FanIn edge helpers
  * Details: .copilot-tracking/details/2026-02-04-go-epic4-workflow-protocols-details.md (Lines 662-750)
* [x] Step 2.4: Implement Switch/Case edge support
  * Details: .copilot-tracking/details/2026-02-04-go-epic4-workflow-protocols-details.md (Lines 752-830)
* [x] Step 2.5: Implement Build validation and Workflow creation
  * Details: .copilot-tracking/details/2026-02-04-go-epic4-workflow-protocols-details.md (Lines 832-920)
* [x] Step 2.6: Write unit tests for WorkflowBuilder
  * Details: .copilot-tracking/details/2026-02-04-go-epic4-workflow-protocols-details.md (Lines 922-1000)
* [x] Step 2.7: Validate Phase 2 changes
  * Run `go build ./workflow/...` and `go test ./workflow/...`
  * Verify builder patterns work correctly

### [x] Implementation Phase 3: Workflow Execution Engine

<!-- parallelizable: false -->

* [x] Step 3.1: Implement WorkflowRunner core
  * Details: .copilot-tracking/details/2026-02-04-go-epic4-workflow-protocols-details.md (Lines 1010-1120)
* [x] Step 3.2: Implement Pregel-like superstep execution
  * Details: .copilot-tracking/details/2026-02-04-go-epic4-workflow-protocols-details.md (Lines 1122-1240)
* [x] Step 3.3: Implement message routing between executors
  * Details: .copilot-tracking/details/2026-02-04-go-epic4-workflow-protocols-details.md (Lines 1242-1340)
* [x] Step 3.4: Implement convergence detection
  * Details: .copilot-tracking/details/2026-02-04-go-epic4-workflow-protocols-details.md (Lines 1342-1420)
* [x] Step 3.5: Implement WorkflowEvent streaming via channels
  * Details: .copilot-tracking/details/2026-02-04-go-epic4-workflow-protocols-details.md (Lines 1422-1520)
* [x] Step 3.6: Write unit tests for WorkflowRunner
  * Details: .copilot-tracking/details/2026-02-04-go-epic4-workflow-protocols-details.md (Lines 1522-1620)
* [x] Step 3.7: Validate Phase 3 changes
  * Run full workflow package tests with coverage
  * Verify superstep execution works correctly

### [x] Implementation Phase 4: Checkpointing

<!-- parallelizable: false -->

* [x] Step 4.1: Define CheckpointStore interface
  * Details: .copilot-tracking/details/2026-02-04-go-epic4-workflow-protocols-details.md (Lines 1630-1700)
* [x] Step 4.2: Implement InMemoryCheckpointStore
  * Details: .copilot-tracking/details/2026-02-04-go-epic4-workflow-protocols-details.md (Lines 1702-1790)
* [x] Step 4.3: Implement checkpoint save/restore in WorkflowRunner
  * Details: .copilot-tracking/details/2026-02-04-go-epic4-workflow-protocols-details.md (Lines 1792-1890)
* [x] Step 4.4: Write unit tests for checkpointing
  * Details: .copilot-tracking/details/2026-02-04-go-epic4-workflow-protocols-details.md (Lines 1892-1960)
* [x] Step 4.5: Validate Phase 4 changes
  * Run checkpoint tests
  * Verify state persistence and recovery

### [x] Implementation Phase 5: Built-in Executors

<!-- parallelizable: true -->

* [x] Step 5.1: Implement AgentExecutor
  * Details: .copilot-tracking/details/2026-02-04-go-epic4-workflow-protocols-details.md (Lines 1970-2070)
* [x] Step 5.2: Implement FunctionExecutor
  * Details: .copilot-tracking/details/2026-02-04-go-epic4-workflow-protocols-details.md (Lines 2072-2160)
* [x] Step 5.3: Implement AggregatingExecutor
  * Details: .copilot-tracking/details/2026-02-04-go-epic4-workflow-protocols-details.md (Lines 2162-2250)
* [x] Step 5.4: Write unit tests for built-in executors
  * Details: .copilot-tracking/details/2026-02-04-go-epic4-workflow-protocols-details.md (Lines 2252-2350)
* [x] Step 5.5: Validate Phase 5 changes
  * Run executor tests
  * Verify integration with workflow runner

### [x] Implementation Phase 6: A2A Protocol Types

<!-- parallelizable: true -->

* [x] Step 6.1: Create protocol/a2a package structure
  * Details: .copilot-tracking/details/2026-02-04-go-epic4-workflow-protocols-details.md (Lines 2360-2420)
* [x] Step 6.2: Define AgentCard type
  * Details: .copilot-tracking/details/2026-02-04-go-epic4-workflow-protocols-details.md (Lines 2422-2520)
* [x] Step 6.3: Define Task and TaskState types
  * Details: .copilot-tracking/details/2026-02-04-go-epic4-workflow-protocols-details.md (Lines 2522-2620)
* [x] Step 6.4: Define Message and Part types
  * Details: .copilot-tracking/details/2026-02-04-go-epic4-workflow-protocols-details.md (Lines 2622-2740)
* [x] Step 6.5: Define Artifact type
  * Details: .copilot-tracking/details/2026-02-04-go-epic4-workflow-protocols-details.md (Lines 2742-2820)
* [x] Step 6.6: Implement JSON marshaling for A2A types
  * Details: .copilot-tracking/details/2026-02-04-go-epic4-workflow-protocols-details.md (Lines 2822-2920)
* [x] Step 6.7: Write unit tests for A2A types
  * Details: .copilot-tracking/details/2026-02-04-go-epic4-workflow-protocols-details.md (Lines 2922-3000)
* [x] Step 6.8: Validate Phase 6 changes
  * Run A2A type tests
  * Verify JSON serialization matches spec

### [x] Implementation Phase 7: A2A Client

<!-- parallelizable: false -->

* [x] Step 7.1: Implement A2A Client core
  * Details: .copilot-tracking/details/2026-02-04-go-epic4-workflow-protocols-details.md (Lines 3010-3120)
* [x] Step 7.2: Implement GetAgentCard method
  * Details: .copilot-tracking/details/2026-02-04-go-epic4-workflow-protocols-details.md (Lines 3122-3190)
* [x] Step 7.3: Implement CreateTask and GetTask methods
  * Details: .copilot-tracking/details/2026-02-04-go-epic4-workflow-protocols-details.md (Lines 3192-3300)
* [x] Step 7.4: Implement SendMessage method
  * Details: .copilot-tracking/details/2026-02-04-go-epic4-workflow-protocols-details.md (Lines 3302-3400)
* [x] Step 7.5: Implement SendMessageStream with SSE parsing
  * Details: .copilot-tracking/details/2026-02-04-go-epic4-workflow-protocols-details.md (Lines 3402-3540)
* [x] Step 7.6: Implement CancelTask method
  * Details: .copilot-tracking/details/2026-02-04-go-epic4-workflow-protocols-details.md (Lines 3542-3600)
* [x] Step 7.7: Write unit tests for A2A Client
  * Details: .copilot-tracking/details/2026-02-04-go-epic4-workflow-protocols-details.md (Lines 3602-3720)
* [x] Step 7.8: Validate Phase 7 changes
  * Run A2A client tests
  * Verify protocol compliance

### [x] Implementation Phase 8: A2A Server

<!-- parallelizable: false -->

* [x] Step 8.1: Implement A2A Server core
  * Details: .copilot-tracking/details/2026-02-04-go-epic4-workflow-protocols-details.md (Lines 3730-3840)
* [x] Step 8.2: Implement AgentCard endpoint handler
  * Details: .copilot-tracking/details/2026-02-04-go-epic4-workflow-protocols-details.md (Lines 3842-3920)
* [x] Step 8.3: Implement Task management endpoints
  * Details: .copilot-tracking/details/2026-02-04-go-epic4-workflow-protocols-details.md (Lines 3922-4050)
* [x] Step 8.4: Implement SendMessage endpoint with SSE streaming
  * Details: .copilot-tracking/details/2026-02-04-go-epic4-workflow-protocols-details.md (Lines 4052-4200)
* [x] Step 8.5: Implement session and task storage
  * Details: .copilot-tracking/details/2026-02-04-go-epic4-workflow-protocols-details.md (Lines 4202-4300)
* [x] Step 8.6: Write unit tests for A2A Server
  * Details: .copilot-tracking/details/2026-02-04-go-epic4-workflow-protocols-details.md (Lines 4302-4420)
* [x] Step 8.7: Validate Phase 8 changes
  * Run A2A server tests
  * Verify HTTP endpoint compliance

### [x] Implementation Phase 9: A2A Agent Wrapper

<!-- parallelizable: false -->

* [x] Step 9.1: Implement A2AAgent that wraps Client as agent.Agent
  * Details: .copilot-tracking/details/2026-02-04-go-epic4-workflow-protocols-details.md (Lines 4430-4560)
* [x] Step 9.2: Implement Run method using SendMessage
  * Details: .copilot-tracking/details/2026-02-04-go-epic4-workflow-protocols-details.md (Lines 4562-4660)
* [x] Step 9.3: Implement RunStream method using SendMessageStream
  * Details: .copilot-tracking/details/2026-02-04-go-epic4-workflow-protocols-details.md (Lines 4662-4780)
* [x] Step 9.4: Implement session management
  * Details: .copilot-tracking/details/2026-02-04-go-epic4-workflow-protocols-details.md (Lines 4782-4860)
* [x] Step 9.5: Write unit tests for A2AAgent
  * Details: .copilot-tracking/details/2026-02-04-go-epic4-workflow-protocols-details.md (Lines 4862-4960)
* [x] Step 9.6: Validate Phase 9 changes
  * Run A2A agent tests
  * Verify agent.Agent interface compliance

### [x] Implementation Phase 10: AG-UI Protocol Types

<!-- parallelizable: true -->

* [x] Step 10.1: Create protocol/agui package structure
  * Details: .copilot-tracking/details/2026-02-04-go-epic4-workflow-protocols-details.md (Lines 4970-5030)
* [x] Step 10.2: Define internal event types
  * Details: .copilot-tracking/details/2026-02-04-go-epic4-workflow-protocols-details.md (Lines 5032-5180)
* [x] Step 10.3: Implement Event JSON marshaling
  * Details: .copilot-tracking/details/2026-02-04-go-epic4-workflow-protocols-details.md (Lines 5182-5280)
* [x] Step 10.4: Write unit tests for AG-UI types
  * Details: .copilot-tracking/details/2026-02-04-go-epic4-workflow-protocols-details.md (Lines 5282-5350)
* [x] Step 10.5: Validate Phase 10 changes
  * Run AG-UI type tests
  * Verify event serialization

### [x] Implementation Phase 11: AG-UI Event Converter

<!-- parallelizable: false -->

* [x] Step 11.1: Implement EventConverter core
  * Details: .copilot-tracking/details/2026-02-04-go-epic4-workflow-protocols-details.md (Lines 5360-5460)
* [x] Step 11.2: Convert ResponseUpdate to lifecycle events
  * Details: .copilot-tracking/details/2026-02-04-go-epic4-workflow-protocols-details.md (Lines 5462-5560)
* [x] Step 11.3: Convert messages to text events
  * Details: .copilot-tracking/details/2026-02-04-go-epic4-workflow-protocols-details.md (Lines 5562-5660)
* [x] Step 11.4: Convert tool calls to tool events
  * Details: .copilot-tracking/details/2026-02-04-go-epic4-workflow-protocols-details.md (Lines 5662-5780)
* [x] Step 11.5: Write unit tests for EventConverter
  * Details: .copilot-tracking/details/2026-02-04-go-epic4-workflow-protocols-details.md (Lines 5782-5880)
* [x] Step 11.6: Validate Phase 11 changes
  * Run converter tests
  * Verify event mapping accuracy

### [x] Implementation Phase 12: AG-UI Server

<!-- parallelizable: false -->

* [x] Step 12.1: Implement AG-UI Server core
  * Details: .copilot-tracking/details/2026-02-04-go-epic4-workflow-protocols-details.md (Lines 5890-5990)
* [x] Step 12.2: Implement SSE streaming handler
  * Details: .copilot-tracking/details/2026-02-04-go-epic4-workflow-protocols-details.md (Lines 5992-6120)
* [x] Step 12.3: Implement connection lifecycle management
  * Details: .copilot-tracking/details/2026-02-04-go-epic4-workflow-protocols-details.md (Lines 6122-6220)
* [x] Step 12.4: Implement agent response to SSE conversion
  * Details: .copilot-tracking/details/2026-02-04-go-epic4-workflow-protocols-details.md (Lines 6222-6340)
* [x] Step 12.5: Write unit tests for AG-UI Server
  * Details: .copilot-tracking/details/2026-02-04-go-epic4-workflow-protocols-details.md (Lines 6342-6460)
* [x] Step 12.6: Validate Phase 12 changes
  * Run AG-UI server tests
  * Verify SSE streaming works correctly

### [ ] Implementation Phase 13: Group Chat Core

<!-- parallelizable: false -->

* [ ] Step 13.1: Create workflow/groupchat package structure
  * Details: .copilot-tracking/details/2026-02-04-go-epic4-workflow-protocols-details.md (Lines 6470-6530)
* [ ] Step 13.2: Define Selector interface
  * Details: .copilot-tracking/details/2026-02-04-go-epic4-workflow-protocols-details.md (Lines 6532-6610)
* [ ] Step 13.3: Define Transcript and TranscriptEntry types
  * Details: .copilot-tracking/details/2026-02-04-go-epic4-workflow-protocols-details.md (Lines 6612-6700)
* [ ] Step 13.4: Define GroupChatEvent types
  * Details: .copilot-tracking/details/2026-02-04-go-epic4-workflow-protocols-details.md (Lines 6702-6780)
* [ ] Step 13.5: Write unit tests for group chat types
  * Details: .copilot-tracking/details/2026-02-04-go-epic4-workflow-protocols-details.md (Lines 6782-6850)
* [ ] Step 13.6: Validate Phase 13 changes
  * Run group chat type tests

### [ ] Implementation Phase 14: Built-in Selectors

<!-- parallelizable: true -->

* [ ] Step 14.1: Implement RoundRobinSelector
  * Details: .copilot-tracking/details/2026-02-04-go-epic4-workflow-protocols-details.md (Lines 6860-6960)
* [ ] Step 14.2: Implement RandomSelector
  * Details: .copilot-tracking/details/2026-02-04-go-epic4-workflow-protocols-details.md (Lines 6962-7050)
* [ ] Step 14.3: Implement LLMSelector
  * Details: .copilot-tracking/details/2026-02-04-go-epic4-workflow-protocols-details.md (Lines 7052-7200)
* [ ] Step 14.4: Write unit tests for selectors
  * Details: .copilot-tracking/details/2026-02-04-go-epic4-workflow-protocols-details.md (Lines 7202-7320)
* [ ] Step 14.5: Validate Phase 14 changes
  * Run selector tests
  * Verify selection logic

### [ ] Implementation Phase 15: Group Chat Manager

<!-- parallelizable: false -->

* [ ] Step 15.1: Implement Manager core structure
  * Details: .copilot-tracking/details/2026-02-04-go-epic4-workflow-protocols-details.md (Lines 7330-7440)
* [ ] Step 15.2: Implement Run method with turn loop
  * Details: .copilot-tracking/details/2026-02-04-go-epic4-workflow-protocols-details.md (Lines 7442-7580)
* [ ] Step 15.3: Implement RunStream method with channel output
  * Details: .copilot-tracking/details/2026-02-04-go-epic4-workflow-protocols-details.md (Lines 7582-7720)
* [ ] Step 15.4: Implement termination conditions
  * Details: .copilot-tracking/details/2026-02-04-go-epic4-workflow-protocols-details.md (Lines 7722-7820)
* [ ] Step 15.5: Implement functional options for configuration
  * Details: .copilot-tracking/details/2026-02-04-go-epic4-workflow-protocols-details.md (Lines 7822-7920)
* [ ] Step 15.6: Write unit tests for Manager
  * Details: .copilot-tracking/details/2026-02-04-go-epic4-workflow-protocols-details.md (Lines 7922-8060)
* [ ] Step 15.7: Validate Phase 15 changes
  * Run manager tests
  * Verify multi-agent orchestration

### [ ] Implementation Phase 16: Integration and Examples

<!-- parallelizable: true -->

* [ ] Step 16.1: Create workflow example
  * Details: .copilot-tracking/details/2026-02-04-go-epic4-workflow-protocols-details.md (Lines 8070-8180)
* [ ] Step 16.2: Create A2A client/server example
  * Details: .copilot-tracking/details/2026-02-04-go-epic4-workflow-protocols-details.md (Lines 8182-8300)
* [ ] Step 16.3: Create AG-UI server example
  * Details: .copilot-tracking/details/2026-02-04-go-epic4-workflow-protocols-details.md (Lines 8302-8400)
* [ ] Step 16.4: Create group chat example
  * Details: .copilot-tracking/details/2026-02-04-go-epic4-workflow-protocols-details.md (Lines 8402-8520)
* [ ] Step 16.5: Update README with Epic 4 features
  * Details: .copilot-tracking/details/2026-02-04-go-epic4-workflow-protocols-details.md (Lines 8522-8600)
* [ ] Step 16.6: Validate Phase 16 changes
  * Build and run examples
  * Verify documentation accuracy

### [ ] Implementation Phase 17: Final Validation

<!-- parallelizable: false -->

* [ ] Step 17.1: Run full project validation
  * Execute `go build ./...` for all packages
  * Execute `go test ./...` with coverage
  * Run `go vet ./...` for static analysis
* [ ] Step 17.2: Verify 90%+ test coverage
  * Run `go test -cover ./workflow/...`
  * Run `go test -cover ./protocol/...`
  * Generate coverage report
* [ ] Step 17.3: Fix minor validation issues
  * Iterate on lint errors and build warnings
  * Apply fixes directly when corrections are straightforward
* [ ] Step 17.4: Report blocking issues
  * Document issues requiring additional research
  * Provide user with next steps and recommended planning
  * Avoid large-scale fixes within this phase

## Dependencies

* Go 1.22+ with generics and improved error handling
* Existing Epic 1-3 packages: agent, chat, chatagent, tool, hosting, provider
* Standard library: net/http, encoding/json, context, sync, bufio
* No external dependencies required (standard library preferred)

## Success Criteria

* All workflow executors process messages correctly via Pregel-like supersteps
* A2A Client successfully communicates with A2A-compliant servers
* A2A Server exposes local agents via A2A protocol endpoints
* AG-UI Server streams events to UI clients via SSE
* Group Chat Manager orchestrates multi-agent conversations with pluggable selection
* 90%+ test coverage for all new packages
* Complete godoc documentation for all public APIs
* Working examples for each feature
* API parity with .NET and Python implementations (Go-idiomatic)

## Package Structure

```text
go/
├── workflow/                      # Feature 4.1: Workflow Engine
│   ├── doc.go
│   ├── workflow.go                # Workflow struct
│   ├── builder.go                 # WorkflowBuilder
│   ├── executor.go                # Executor interface
│   ├── edge.go                    # Edge types
│   ├── runner.go                  # WorkflowRunner
│   ├── context.go                 # WorkflowContext
│   ├── checkpoint.go              # CheckpointStore interface
│   ├── events.go                  # WorkflowEvent types
│   ├── executors/                 # Built-in executors
│   │   ├── agent.go               # AgentExecutor
│   │   ├── function.go            # FunctionExecutor
│   │   └── aggregating.go         # AggregatingExecutor
│   ├── groupchat/                 # Feature 4.4: Group Chat
│   │   ├── manager.go             # Manager
│   │   ├── selector.go            # Selector interface + implementations
│   │   ├── transcript.go          # Transcript types
│   │   └── options.go             # Functional options
│   └── *_test.go                  # Unit tests
├── protocol/
│   ├── a2a/                       # Feature 4.2: A2A Protocol
│   │   ├── doc.go
│   │   ├── types.go               # AgentCard, Task, Message, Part
│   │   ├── client.go              # A2A Client
│   │   ├── server.go              # A2A Server
│   │   ├── agent.go               # A2AAgent wrapper
│   │   ├── session.go             # A2ASession
│   │   └── *_test.go              # Unit tests
│   └── agui/                      # Feature 4.3: AG-UI Protocol
│       ├── doc.go
│       ├── types.go               # Event types (internal)
│       ├── events.go              # Event definitions
│       ├── server.go              # AG-UI SSE Server
│       ├── converter.go           # ResponseUpdate to Event converter
│       └── *_test.go              # Unit tests
```

## API Comparison Summary

| Feature | .NET | Python | Go Implementation |
|---------|------|--------|-------------------|
| Executor | Abstract class | Class with decorators | Interface |
| WorkflowBuilder | Fluent class | Fluent class | Fluent struct with functional options |
| Edge conditions | Delegate | Callable | `func(any) bool` |
| Checkpointing | ICheckpointStorage | CheckpointStorage protocol | Interface |
| Workflow events | IObserver pattern | AsyncIterable | `<-chan WorkflowEvent` |
| A2A Client | A2AAgent wrapper | A2AAgent | Client struct |
| A2A Server | MapA2A extension | Not implemented | http.Handler |
| AG-UI Server | MapAGUI extension | FastAPI endpoint | http.Handler with SSE |
| AG-UI Events | Internal EventType enum | ag-ui-protocol library | Internal constants |
| GroupChatManager | Abstract class | BaseGroupChatOrchestrator | Manager struct |
| Selector | Override method | Callable function | Selector interface |
