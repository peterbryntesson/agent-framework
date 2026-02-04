<!-- markdownlint-disable-file -->
# Task Research: Go Epic 4 - Workflow Orchestration and Protocols

Comprehensive research for implementing Epic 4 features in the Go Agent Framework SDK, with detailed comparison to existing .NET and Python implementations.

## Task Implementation Requests

* Research Feature 4.1: Workflow Engine Package with DAG-based orchestration
* Research Feature 4.2: A2A Protocol Implementation for agent-to-agent communication
* Research Feature 4.3: AG-UI Protocol Implementation for agent-UI streaming
* Research Feature 4.4: Group Chat Orchestration for multi-agent conversations
* Compare with .NET and Python implementations for feature parity

## Scope and Success Criteria

* Scope: Epic 4 features only (Workflow, A2A, AG-UI, Group Chat)
* Assumptions: Building on completed Epic 1-3 foundations (agent interfaces, providers, middleware)
* Success Criteria:
  * Complete API surface comparison with .NET and Python
  * Identified Go-specific patterns and idioms to apply
  * Documented interface definitions and type mappings
  * Clear implementation recommendations for each feature

## Outline

1. Feature 4.1: Workflow Engine Package
2. Feature 4.2: A2A Protocol Implementation
3. Feature 4.3: AG-UI Protocol Implementation
4. Feature 4.4: Group Chat Orchestration

### Potential Next Research

* Temporal.io integration patterns for Go (Feature 5.1)
* Durable workflow state management
* MCP protocol integration (Feature 5.4)

## Research Executed

### File Analysis

#### .NET Workflows Package
* `dotnet/src/Microsoft.Agents.AI.Workflows/` - Core workflow package (100+ files)
* `Executor.cs` (Lines 1-80) - Abstract base class for workflow nodes
* `WorkflowBuilder.cs` (Lines 1-100) - Fluent builder for workflow construction
* `Edge.cs` - Edge definitions with conditions
* `AgentWorkflowBuilder.cs` - High-level builder for agent workflows
* `GroupChatManager.cs` - Group chat orchestration
* `Checkpointing/` - Checkpoint storage implementations

#### .NET A2A Package
* `dotnet/src/Microsoft.Agents.AI.A2A/` - A2A client implementation
* `dotnet/src/Microsoft.Agents.AI.Hosting.A2A/` - A2A hosting core
* `dotnet/src/Microsoft.Agents.AI.Hosting.A2A.AspNetCore/` - ASP.NET Core endpoints

#### .NET AG-UI Package
* `dotnet/src/Microsoft.Agents.AI.AGUI/` - AG-UI core types
* `dotnet/src/Microsoft.Agents.AI.Hosting.AGUI.AspNetCore/` - AG-UI SSE hosting

#### Python Workflows Package
* `python/packages/core/agent_framework/_workflows/` - Core workflow module (38 files)
* `_workflow.py`, `_workflow_builder.py` - Workflow definition and builder
* `_executor.py`, `_function_executor.py` - Executor implementations
* `_edge.py` - Edge types (FanIn, FanOut, SwitchCase)
* `_group_chat.py` - Group chat orchestration
* `_checkpoint.py` - Checkpoint storage

#### Python A2A Package
* `python/packages/a2a/agent_framework_a2a/` - A2A agent implementation
* `_agent.py` (Lines 1-150) - A2AAgent wrapping external A2A client

#### Python AG-UI Package
* `python/packages/ag-ui/agent_framework_ag_ui/` - AG-UI implementation
* Multiple modules: `_agent.py`, `_client.py`, `_endpoint.py`, `_http_service.py`

### Project Conventions

* Standards referenced: Go 1.22+ idioms, functional options pattern
* Instructions followed: .github/copilot-instructions.md

## Key Discoveries

### Project Structure

Both .NET and Python have mature, comprehensive workflow implementations with:
* DAG-based execution using Pregel-like superstep model
* Multiple edge types (Direct, FanOut, FanIn, SwitchCase)
* Checkpointing for persistence and recovery
* Event streaming for observability
* High-level builders for common patterns

### Implementation Patterns

#### Pregel-like Execution Model
Both implementations use synchronized supersteps:
1. Executors process incoming messages
2. Executors send new messages to connected nodes
3. System iterates until convergence (no more messages)

#### Executor Pattern
* .NET: Abstract `Executor` class with `ConfigureRoutes(RouteBuilder)` method
* Python: `Executor` class with `@handler` decorator for message handlers

#### Edge Types
| Edge Type | .NET | Python | Go Recommendation |
|-----------|------|--------|-------------------|
| Direct | `DirectEdgeData` | `Edge` | `DirectEdge` |
| FanOut | `FanOutEdgeData` | `FanOutEdgeGroup` | `FanOutEdge` |
| FanIn | `FanInEdgeData` | `FanInEdgeGroup` | `FanInEdge` |
| Conditional | Built into Edge | `EdgeCondition` | `ConditionalEdge` |
| Switch/Case | `SwitchBuilder` | `SwitchCaseEdgeGroup` | `SwitchEdge` |

### API and Schema Documentation

#### A2A Protocol Specification
* Source: https://a2a-protocol.org/latest/
* Transport: JSON-RPC over HTTP with SSE for streaming
* Core types: AgentCard, Task, Message, Part, Artifact
* Task states: Pending, Running, Completed, Failed, Cancelled

#### AG-UI Protocol Events
| Category | Events |
|----------|--------|
| Lifecycle | `RUN_STARTED`, `RUN_FINISHED`, `RUN_ERROR` |
| Text | `TEXT_MESSAGE_START`, `TEXT_MESSAGE_CONTENT`, `TEXT_MESSAGE_END` |
| Tool | `TOOL_CALL_START`, `TOOL_CALL_ARGS`, `TOOL_CALL_END`, `TOOL_CALL_RESULT` |
| State | `STATE_SNAPSHOT`, `STATE_DELTA` |

## Technical Scenarios

### Scenario 4.1: Workflow Engine Package

**Description:** Implement DAG-based workflow orchestration with Pregel-like execution model.

**Requirements:**
* Workflow definition with nodes and edges
* WorkflowBuilder for fluent construction
* Executor interface for node implementations
* Edge types: Direct, FanOut, FanIn, Conditional, Switch
* Checkpointing for persistence
* Event streaming for observability

**Preferred Approach:** Follow Python architecture with Go idioms

```text
workflow/
├── workflow.go          # Workflow struct and definition
├── builder.go           # WorkflowBuilder fluent API
├── executor.go          # Executor interface
├── edge.go              # Edge types and conditions
├── runner.go            # WorkflowRunner execution
├── checkpoint.go        # CheckpointStore interface
├── events.go            # Workflow events
└── context.go           # WorkflowContext for executors
```

**Interface Definitions:**

```go
// Executor processes messages in a workflow node.
type Executor interface {
    ID() string
    Execute(ctx context.Context, wCtx *WorkflowContext) error
}

// Edge connects two executors with optional condition.
type Edge struct {
    From      string
    To        string
    Condition EdgeCondition
}

type EdgeCondition func(state any) bool

// WorkflowBuilder constructs workflows fluently.
type WorkflowBuilder struct {
    executors map[string]Executor
    edges     []Edge
    startID   string
}

func NewBuilder(start Executor) *WorkflowBuilder
func (b *WorkflowBuilder) AddExecutor(exec Executor) *WorkflowBuilder
func (b *WorkflowBuilder) AddEdge(from, to string, cond EdgeCondition) *WorkflowBuilder
func (b *WorkflowBuilder) AddFanOut(from string, targets ...string) *WorkflowBuilder
func (b *WorkflowBuilder) AddFanIn(sources []string, to string) *WorkflowBuilder
func (b *WorkflowBuilder) Build() (*Workflow, error)
```

**API Comparison:**

| Feature | .NET | Python | Go Recommendation |
|---------|------|--------|-------------------|
| Executor base | Abstract class | Class with decorators | Interface |
| Message routing | RouteBuilder | @handler decorator | Handler function |
| Edge conditions | Delegate | Callable | `func(any) bool` |
| Checkpointing | ICheckpointStorage | CheckpointStorage protocol | Interface |
| Events | IObserver pattern | AsyncIterable | Channel-based |

---

### Scenario 4.2: A2A Protocol Implementation

**Description:** Implement agent-to-agent communication following the A2A protocol specification.

**Requirements:**
* A2A Client for calling remote agents
* A2A Server for exposing local agents
* AgentCard for capability declaration
* Task lifecycle management
* Streaming message support

**Preferred Approach:** Hybrid of .NET (full client+server) and Python (agent wrapper)

```text
protocol/a2a/
├── types.go      # AgentCard, Task, Message, Part
├── client.go     # A2A Client implementation
├── server.go     # A2A HTTP Server
├── agent.go      # A2AAgent wrapper for agent.Agent
└── session.go    # A2ASession with context/task tracking
```

**Interface Definitions:**

```go
// Client connects to remote A2A agents.
type Client struct {
    baseURL    string
    httpClient *http.Client
}

func NewClient(baseURL string, opts ...Option) *Client
func (c *Client) GetAgentCard(ctx context.Context) (*AgentCard, error)
func (c *Client) CreateTask(ctx context.Context, req *CreateTaskRequest) (*Task, error)
func (c *Client) GetTask(ctx context.Context, taskID string) (*Task, error)
func (c *Client) SendMessage(ctx context.Context, taskID string, msg *Message) (*Task, error)
func (c *Client) SendMessageStream(ctx context.Context, taskID string, msg *Message) (<-chan *Message, error)
func (c *Client) CancelTask(ctx context.Context, taskID string) error

// Server exposes an agent via A2A protocol.
type Server struct {
    agent    agent.Agent
    card     *AgentCard
    sessions sync.Map
}

func NewServer(agent agent.Agent, opts ...Option) *Server
func (s *Server) Handler() http.Handler

// A2AAgent wraps a remote A2A agent as a local agent.Agent.
type A2AAgent struct {
    client *Client
}

func NewA2AAgent(client *Client) *A2AAgent
// Implements agent.Agent interface
```

**API Comparison:**

| Feature | .NET | Python | Go Recommendation |
|---------|------|--------|-------------------|
| Client | A2AAgent (Client wrapper) | A2AAgent (httpx-based) | Client struct |
| Server Hosting | MapA2A extension | Not implemented | http.Handler |
| Session Tracking | A2AAgentSession | Thread-based | A2ASession |
| External Dependency | A2A NuGet package | a2a-python package | Native implementation |
| Streaming | SSE via A2A client | AsyncIterable | Channel-based |

---

### Scenario 4.3: AG-UI Protocol Implementation

**Description:** Implement agent-to-UI streaming protocol with Server-Sent Events.

**Requirements:**
* AG-UI event types
* SSE streaming server
* Event conversion from agent responses
* Optional client for consuming streams

**Preferred Approach:** Keep events internal, expose framework-native types

```text
protocol/agui/
├── types.go       # Internal event types
├── events.go      # Event definitions and serialization
├── server.go      # AG-UI SSE server
├── converter.go   # Convert agent updates to AG-UI events
└── client.go      # AG-UI client (optional)
```

**Interface Definitions:**

```go
// Event types (internal)
type EventType string

const (
    EventRunStarted       EventType = "RUN_STARTED"
    EventRunFinished      EventType = "RUN_FINISHED"
    EventRunError         EventType = "RUN_ERROR"
    EventTextMessageStart EventType = "TEXT_MESSAGE_START"
    EventTextMessageContent EventType = "TEXT_MESSAGE_CONTENT"
    EventTextMessageEnd   EventType = "TEXT_MESSAGE_END"
    EventToolCallStart    EventType = "TOOL_CALL_START"
    EventToolCallArgs     EventType = "TOOL_CALL_ARGS"
    EventToolCallEnd      EventType = "TOOL_CALL_END"
    EventStateSnapshot    EventType = "STATE_SNAPSHOT"
    EventStateDelta       EventType = "STATE_DELTA"
)

// Server streams AG-UI events to clients.
type Server struct {
    agent agent.Agent
}

func NewServer(agent agent.Agent, opts ...Option) *Server
func (s *Server) Handler() http.Handler

// EventConverter transforms agent updates to AG-UI events.
type EventConverter struct{}

func (c *EventConverter) Convert(update *agent.ResponseUpdate) ([]Event, error)
```

**API Comparison:**

| Feature | .NET | Python | Go Recommendation |
|---------|------|--------|-------------------|
| Event Types | Internal EventType enum | ag-ui-protocol library | Internal constants |
| SSE Server | MapAGUI extension | FastAPI endpoint | http.Handler with SSE |
| Event Converter | AGUIEventConverter | AGUIEventConverter | EventConverter |
| Client | AGUIChatClient | AGUIChatClient | Optional Client |
| JSON Handling | Custom polymorphic | Pydantic | json.Marshaler |

---

### Scenario 4.4: Group Chat Orchestration

**Description:** Implement multi-agent conversation orchestration with pluggable selection strategies.

**Requirements:**
* GroupChatManager for orchestrating conversations
* Selector interface for speaker selection
* Built-in selectors: RoundRobin, Random, LLM-based
* Transcript tracking
* Termination conditions

**Preferred Approach:** Follow Python's pattern with Go idioms

```text
workflow/groupchat/
├── manager.go     # GroupChatManager
├── selector.go    # Selector interface and implementations
├── transcript.go  # Transcript tracking
└── options.go     # Configuration options
```

**Interface Definitions:**

```go
// Selector determines which agent speaks next.
type Selector interface {
    SelectNext(ctx context.Context, transcript *Transcript) (agent.Agent, error)
}

// RoundRobinSelector cycles through agents in order.
type RoundRobinSelector struct {
    agents []agent.Agent
    index  int
}

func NewRoundRobinSelector(agents ...agent.Agent) *RoundRobinSelector

// RandomSelector picks a random agent.
type RandomSelector struct {
    agents []agent.Agent
}

func NewRandomSelector(agents ...agent.Agent) *RandomSelector

// LLMSelector uses an LLM to decide the next speaker.
type LLMSelector struct {
    decisionAgent agent.Agent
    agents        []agent.Agent
}

func NewLLMSelector(decisionAgent agent.Agent, agents ...agent.Agent) *LLMSelector

// Manager orchestrates multi-agent conversations.
type Manager struct {
    agents     []agent.Agent
    selector   Selector
    maxTurns   int
    transcript *Transcript
}

func NewManager(agents []agent.Agent, opts ...Option) *Manager
func (m *Manager) Run(ctx context.Context, input string) (*Transcript, error)
func (m *Manager) RunStream(ctx context.Context, input string) (<-chan Event, error)

// Transcript records the conversation.
type Transcript struct {
    Entries []TranscriptEntry
}

type TranscriptEntry struct {
    Speaker   string
    Message   agent.Message
    Timestamp time.Time
}
```

**API Comparison:**

| Feature | .NET | Python | Go Recommendation |
|---------|------|--------|-------------------|
| Manager | GroupChatManager | BaseGroupChatOrchestrator | Manager struct |
| Selector | Override method | Callable function | Selector interface |
| RoundRobin | RoundRobinGroupChatManager | GroupChatBuilder | RoundRobinSelector |
| LLM-based | Custom subclass | AgentBasedGroupChatOrchestrator | LLMSelector |
| Termination | Delegate | Builder methods | Functional options |
| Streaming | WorkflowEvent | AsyncIterable | Channel-based |

---

## Cross-Platform Feature Summary

| Feature | .NET | Python | Go Epic 4 |
|---------|:----:|:------:|:---------:|
| **Workflow Engine** |
| DAG Execution | ✅ | ✅ | 🔲 Feature 4.1 |
| WorkflowBuilder | ✅ | ✅ | 🔲 Feature 4.1 |
| Executor Interface | ✅ | ✅ | 🔲 Feature 4.1 |
| Edge Types (Direct/FanOut/FanIn) | ✅ | ✅ | 🔲 Feature 4.1 |
| Checkpointing | ✅ | ✅ | 🔲 Feature 4.1 |
| Event Streaming | ✅ | ✅ | 🔲 Feature 4.1 |
| **A2A Protocol** |
| A2A Client | ✅ | ✅ | 🔲 Feature 4.2 |
| A2A Server | ✅ | ❌ | 🔲 Feature 4.2 |
| A2A Agent Wrapper | ✅ | ✅ | 🔲 Feature 4.2 |
| Task Management | ✅ | ✅ | 🔲 Feature 4.2 |
| SSE Streaming | ✅ | ✅ | 🔲 Feature 4.2 |
| **AG-UI Protocol** |
| AG-UI Server | ✅ | ✅ | 🔲 Feature 4.3 |
| AG-UI Client | ✅ | ✅ | 🔲 Feature 4.3 |
| Event Types | ✅ | ✅ | 🔲 Feature 4.3 |
| SSE Streaming | ✅ | ✅ | 🔲 Feature 4.3 |
| **Group Chat** |
| GroupChatManager | ✅ | ✅ | 🔲 Feature 4.4 |
| Selector Interface | ✅ | ✅ | 🔲 Feature 4.4 |
| RoundRobin Selector | ✅ | ✅ | 🔲 Feature 4.4 |
| LLM-based Selector | ✅ | ✅ | 🔲 Feature 4.4 |
| Transcript | ✅ | ✅ | 🔲 Feature 4.4 |

---

## Go Implementation Recommendations

### 1. Workflow Engine (Feature 4.1)

**Priority:** High - Foundation for all orchestration features

**Recommendations:**
1. Use interface-based design for Executor (not abstract class)
2. Implement Pregel-like superstep execution with channels
3. Use `context.Context` throughout for cancellation
4. Functional options for builder configuration
5. Start with InMemoryCheckpointStore, add Redis/file later

**Key Decisions:**
* Message passing: Use typed channels vs interface{}
* State management: Use sync.Map for concurrent access
* Event streaming: Return `<-chan WorkflowEvent`

### 2. A2A Protocol (Feature 4.2)

**Priority:** High - Required for agent-to-agent communication

**Recommendations:**
1. Implement native A2A types (don't depend on external package)
2. Build both Client and Server (unlike Python which only has client)
3. Use standard `net/http` for server
4. Implement SSE streaming with proper connection handling
5. Create `A2AAgent` that implements `agent.Agent` interface

**Key Decisions:**
* JSON-RPC handling: Custom or use existing library
* SSE client: Use `bufio.Scanner` for line-based parsing
* Session storage: Interface-based with in-memory default

### 3. AG-UI Protocol (Feature 4.3)

**Priority:** Medium - Important for UI integration

**Recommendations:**
1. Keep event types internal (framework-native public API)
2. Implement SSE server with proper flushing
3. EventConverter transforms agent responses to AG-UI events
4. Support both FastAPI-style endpoint and standalone server

**Key Decisions:**
* Event format: Follow AG-UI spec exactly
* Buffering: Use `bufio.Writer` with explicit flush
* Connection management: Handle client disconnection gracefully

### 4. Group Chat (Feature 4.4)

**Priority:** Medium - Builds on workflow engine

**Recommendations:**
1. Define `Selector` interface for flexibility
2. Implement RoundRobin and Random as built-ins
3. LLMSelector uses structured output for agent selection
4. Transcript as slice of entries (not map)
5. Functional options for termination conditions

**Key Decisions:**
* Max turns: Required option with sensible default
* Termination: Support both max turns and custom condition
* Streaming: Use channels for progressive updates

---

## Dependencies and Prerequisites

### Go Module Dependencies

```go
require (
    // Existing from Epic 1-3
    github.com/microsoft/agent-framework-go/agent
    github.com/microsoft/agent-framework-go/chat
    
    // New for Epic 4
    // None required - use standard library where possible
)
```

### External References

* A2A Protocol: https://a2a-protocol.org/latest/
* AG-UI Protocol: https://docs.ag-ui.com/
* Pregel Paper: https://research.google/pubs/pregel-a-system-for-large-scale-graph-processing/

---

## Success Metrics

1. **API Parity**: All features from .NET/Python have Go equivalents
2. **Test Coverage**: 90%+ coverage for all new packages
3. **Performance**: <5ms overhead for workflow execution
4. **Documentation**: Complete godoc for all public APIs
5. **Examples**: Working examples for each feature
