# Workflow Engine Implementation Analysis

Research Date: 2026-02-04

## Executive Summary

Both .NET and Python implementations provide comprehensive workflow engines based on a DAG (Directed Acyclic Graph) execution model using Pregel-like supersteps. The implementations share core concepts but use language-idiomatic patterns.

---

## 1. File Locations

### .NET Implementation

| Component | File Path | Line Range |
|-----------|-----------|------------|
| **Core Workflow** | [dotnet/src/Microsoft.Agents.AI.Workflows/Workflow.cs](dotnet/src/Microsoft.Agents.AI.Workflows/Workflow.cs) | L1-218 |
| **WorkflowBuilder** | [dotnet/src/Microsoft.Agents.AI.Workflows/WorkflowBuilder.cs](dotnet/src/Microsoft.Agents.AI.Workflows/WorkflowBuilder.cs) | L1-613 |
| **AgentWorkflowBuilder** | [dotnet/src/Microsoft.Agents.AI.Workflows/AgentWorkflowBuilder.cs](dotnet/src/Microsoft.Agents.AI.Workflows/AgentWorkflowBuilder.cs) | L1-181 |
| **Executor Base** | [dotnet/src/Microsoft.Agents.AI.Workflows/Executor.cs](dotnet/src/Microsoft.Agents.AI.Workflows/Executor.cs) | L1-297 |
| **Edge Types** | [dotnet/src/Microsoft.Agents.AI.Workflows/Edge.cs](dotnet/src/Microsoft.Agents.AI.Workflows/Edge.cs) | L1-74 |
| **IWorkflowContext** | [dotnet/src/Microsoft.Agents.AI.Workflows/IWorkflowContext.cs](dotnet/src/Microsoft.Agents.AI.Workflows/IWorkflowContext.cs) | L1-198 |
| **Checkpoint** | [dotnet/src/Microsoft.Agents.AI.Workflows/Checkpointing/Checkpoint.cs](dotnet/src/Microsoft.Agents.AI.Workflows/Checkpointing/Checkpoint.cs) | L1-43 |
| **EdgeRunner Base** | [dotnet/src/Microsoft.Agents.AI.Workflows/Execution/EdgeRunner.cs](dotnet/src/Microsoft.Agents.AI.Workflows/Execution/EdgeRunner.cs) | L1-30 |
| **FanOutEdgeRunner** | [dotnet/src/Microsoft.Agents.AI.Workflows/Execution/FanOutEdgeRunner.cs](dotnet/src/Microsoft.Agents.AI.Workflows/Execution/FanOutEdgeRunner.cs) | L1-63 |
| **FanInEdgeRunner** | [dotnet/src/Microsoft.Agents.AI.Workflows/Execution/FanInEdgeRunner.cs](dotnet/src/Microsoft.Agents.AI.Workflows/Execution/FanInEdgeRunner.cs) | L1-77 |
| **WorkflowEvent** | [dotnet/src/Microsoft.Agents.AI.Workflows/WorkflowEvent.cs](dotnet/src/Microsoft.Agents.AI.Workflows/WorkflowEvent.cs) | L1-30 |
| **HandoffsWorkflowBuilder** | [dotnet/src/Microsoft.Agents.AI.Workflows/HandoffsWorkflowBuilder.cs](dotnet/src/Microsoft.Agents.AI.Workflows/HandoffsWorkflowBuilder.cs) | - |
| **GroupChatWorkflowBuilder** | [dotnet/src/Microsoft.Agents.AI.Workflows/GroupChatWorkflowBuilder.cs](dotnet/src/Microsoft.Agents.AI.Workflows/GroupChatWorkflowBuilder.cs) | - |

### Python Implementation

| Component | File Path | Line Range |
|-----------|-----------|------------|
| **Core Workflow** | [python/packages/core/agent_framework/_workflows/_workflow.py](python/packages/core/agent_framework/_workflows/_workflow.py) | L1-873 |
| **WorkflowBuilder** | [python/packages/core/agent_framework/_workflows/_workflow_builder.py](python/packages/core/agent_framework/_workflows/_workflow_builder.py) | L1-1322 |
| **Executor Base** | [python/packages/core/agent_framework/_workflows/_executor.py](python/packages/core/agent_framework/_workflows/_executor.py) | L1-630 |
| **Edge/EdgeGroup** | [python/packages/core/agent_framework/_workflows/_edge.py](python/packages/core/agent_framework/_workflows/_edge.py) | L1-942 |
| **WorkflowContext** | [python/packages/core/agent_framework/_workflows/_workflow_context.py](python/packages/core/agent_framework/_workflows/_workflow_context.py) | L1-505 |
| **Checkpoint** | [python/packages/core/agent_framework/_workflows/_checkpoint.py](python/packages/core/agent_framework/_workflows/_checkpoint.py) | L1-217 |
| **Runner** | [python/packages/core/agent_framework/_workflows/_runner.py](python/packages/core/agent_framework/_workflows/_runner.py) | L1-427 |
| **EdgeRunner** | [python/packages/core/agent_framework/_workflows/_edge_runner.py](python/packages/core/agent_framework/_workflows/_edge_runner.py) | L1-402 |
| **ConcurrentBuilder** | [python/packages/core/agent_framework/_workflows/_concurrent.py](python/packages/core/agent_framework/_workflows/_concurrent.py) | L1-578 |
| **SequentialBuilder** | [python/packages/core/agent_framework/_workflows/_sequential.py](python/packages/core/agent_framework/_workflows/_sequential.py) | L1-312 |
| **Events** | [python/packages/core/agent_framework/_workflows/_events.py](python/packages/core/agent_framework/_workflows/_events.py) | L1-396 |
| **Module Exports** | [python/packages/core/agent_framework/_workflows/__init__.py](python/packages/core/agent_framework/_workflows/__init__.py) | L1-225 |

---

## 2. API Surface Comparison

### Core Types Mapping

| Concept | .NET | Python |
|---------|------|--------|
| **Workflow** | `Workflow` class | `Workflow` class |
| **Builder** | `WorkflowBuilder` | `WorkflowBuilder` |
| **Agent Builder** | `AgentWorkflowBuilder` (static) | `ConcurrentBuilder`, `SequentialBuilder` |
| **Executor** | `Executor` abstract class | `Executor` class with `@handler` decorator |
| **Edge** | `Edge` class with `EdgeKind` enum | `Edge` class |
| **Edge Group** | `EdgeData` hierarchy | `EdgeGroup` hierarchy |
| **Context** | `IWorkflowContext` interface | `WorkflowContext` class |
| **Checkpoint** | `Checkpoint` record | `WorkflowCheckpoint` dataclass |
| **Events** | `WorkflowEvent` hierarchy | `WorkflowEvent` hierarchy |
| **Run Result** | `IAsyncEnumerable<WorkflowEvent>` | `WorkflowRunResult` (list subclass) |

### Edge Types

| Type | .NET | Python |
|------|------|--------|
| **Direct** | `DirectEdgeData` | `Edge` (single) |
| **Fan-Out** | `FanOutEdgeData` | `FanOutEdgeGroup` |
| **Fan-In** | `FanInEdgeData` | `FanInEdgeGroup` |
| **Switch/Case** | Via conditional edges | `SwitchCaseEdgeGroup` |

### Event Types

| Event | .NET | Python |
|-------|------|--------|
| **Workflow Started** | `WorkflowStartedEvent` | `WorkflowStartedEvent` |
| **Workflow Output** | `WorkflowOutputEvent` | `WorkflowOutputEvent` |
| **Workflow Error** | `WorkflowErrorEvent` | `WorkflowErrorEvent`, `WorkflowFailedEvent` |
| **Executor Invoked** | `ExecutorInvokedEvent` | `ExecutorInvokedEvent` |
| **Executor Completed** | `ExecutorCompletedEvent` | `ExecutorCompletedEvent` |
| **Executor Failed** | `ExecutorFailedEvent` | `ExecutorFailedEvent` |
| **SuperStep Started** | `SuperStepStartedEvent` | `SuperStepStartedEvent` |
| **SuperStep Completed** | `SuperStepCompletedEvent` | `SuperStepCompletedEvent` |
| **Request Info** | `RequestInfoEvent` | `RequestInfoEvent` |
| **Status** | Implicit in events | `WorkflowStatusEvent` with `WorkflowRunState` enum |

---

## 3. Key Patterns

### 3.1 Workflow Definition (Nodes/Edges)

#### .NET Pattern

```csharp
// WorkflowBuilder.cs L46-54
public WorkflowBuilder(ExecutorBinding start)
{
    this._startExecutorId = this.Track(start).Id;
}

// Edge types: EdgeKind enum (Direct, FanOut, FanIn)
public enum EdgeKind { Direct, FanOut, FanIn }

// Edge class (Edge.cs L36-44)
public sealed class Edge
{
    public EdgeKind Kind { get; init; }
    public EdgeData Data { get; init; }
}
```

#### Python Pattern

```python
# _edge.py L68-78
@dataclass(init=False)
class Edge(DictConvertible):
    source_id: str
    target_id: str
    condition_name: str | None
    _condition: EdgeCondition | None

# Edge groups provide structured edge collections
class SingleEdgeGroup(EdgeGroup): ...
class FanOutEdgeGroup(EdgeGroup): ...
class FanInEdgeGroup(EdgeGroup): ...
```

### 3.2 WorkflowBuilder API (Fluent Pattern)

#### .NET Pattern

```csharp
// WorkflowBuilder.cs fluent API
public WorkflowBuilder AddEdge(ExecutorBinding source, ExecutorBinding target)
public WorkflowBuilder AddFanOutEdge(ExecutorBinding source, IEnumerable<ExecutorBinding> targets)
public WorkflowBuilder AddFanInEdge(IEnumerable<ExecutorBinding> sources, ExecutorBinding target)
public WorkflowBuilder WithOutputFrom(params ExecutorBinding[] executors)
public WorkflowBuilder WithName(string name)
public Workflow Build()

// AgentWorkflowBuilder.cs convenience methods
public static Workflow BuildSequential(params IEnumerable<AIAgent> agents)
public static Workflow BuildConcurrent(IEnumerable<AIAgent> agents, ...)
public static HandoffsWorkflowBuilder CreateHandoffBuilderWith(AIAgent initialAgent)
public static GroupChatWorkflowBuilder CreateGroupChatBuilderWith(...)
```

#### Python Pattern

```python
# _workflow_builder.py fluent API
def register_executor(self, factory: Callable, name: str) -> Self
def add_edge(self, source: str, target: str, condition: EdgeCondition | None = None) -> Self
def add_fan_out_edge(self, source: str, targets: list[str]) -> Self
def add_fan_in_edge(self, sources: list[str], target: str) -> Self
def set_start_executor(self, executor: str | Executor) -> Self
def with_checkpointing(self, storage: CheckpointStorage) -> Self
def build(self) -> Workflow

# High-level builders
class ConcurrentBuilder:
    def participants(self, agents: list[AgentProtocol]) -> Self
    def with_aggregator(self, aggregator: Executor | Callable) -> Self
    def build(self) -> Workflow

class SequentialBuilder:
    def participants(self, agents: list[AgentProtocol]) -> Self
    def with_request_info(self, agents: list | None = None) -> Self
    def build(self) -> Workflow
```

### 3.3 Executor Interface

#### .NET Pattern

```csharp
// Executor.cs
public abstract class Executor : IIdentified
{
    public string Id { get; }
    protected ExecutorOptions Options { get; }
    
    // Route configuration (L57-58)
    protected abstract RouteBuilder ConfigureRoutes(RouteBuilder routeBuilder);
    
    // Message handling (L121-127)
    public async ValueTask<object?> ExecuteAsync(
        object message, 
        TypeId messageType, 
        IWorkflowContext context, 
        CancellationToken cancellationToken = default)
}
```

#### Python Pattern

```python
# _executor.py
class Executor(RequestInfoMixin, DictConvertible):
    def __init__(self, id: str, *, type: str | None = None, defer_discovery: bool = False):
        self.id = id
        self.type = type or self.__class__.__name__
    
    # Handler decorator pattern
    @handler
    async def process(self, message: str, ctx: WorkflowContext[str]) -> None:
        await ctx.send_message(message.upper())
    
    # Execute called by framework
    async def execute(self, message, source_ids, shared_state, ctx, **kwargs)
```

### 3.4 State Management & Checkpointing

#### .NET Pattern

```csharp
// IWorkflowContext.cs state methods
ValueTask<T?> ReadStateAsync<T>(string key, string? scopeName = null, ...)
ValueTask<T> ReadOrInitStateAsync<T>(string key, Func<T> initialStateFactory, ...)
ValueTask QueueStateUpdateAsync<T>(string key, T? value, string? scopeName = null, ...)

// Checkpoint.cs
internal sealed class Checkpoint
{
    public int StepNumber { get; }
    public WorkflowInfo Workflow { get; }
    public RunnerStateData RunnerData { get; }
    public Dictionary<ScopeKey, PortableValue> StateData { get; }
    public Dictionary<EdgeId, PortableValue> EdgeStateData { get; }
}
```

#### Python Pattern

```python
# _checkpoint.py
@dataclass(slots=True)
class WorkflowCheckpoint:
    checkpoint_id: str
    workflow_id: str
    timestamp: str
    messages: dict[str, list[dict[str, Any]]]
    shared_state: dict[str, Any]
    pending_request_info_events: dict[str, dict[str, Any]]
    iteration_count: int
    metadata: dict[str, Any]
    version: str = "1.0"

# CheckpointStorage protocol
class CheckpointStorage(Protocol):
    async def save_checkpoint(self, checkpoint: WorkflowCheckpoint) -> str
    async def load_checkpoint(self, checkpoint_id: str) -> WorkflowCheckpoint | None
    async def list_checkpoint_ids(self, workflow_id: str | None = None) -> list[str]
    async def delete_checkpoint(self, checkpoint_id: str) -> bool
```

### 3.5 Parallel Execution (Fan-Out/Fan-In)

#### .NET Pattern

```csharp
// FanOutEdgeRunner.cs
internal sealed class FanOutEdgeRunner(IRunnerContext runContext, FanOutEdgeData edgeData)
{
    protected internal override async ValueTask<DeliveryMapping?> ChaseEdgeAsync(
        MessageEnvelope envelope, IStepTracer? stepTracer)
    {
        IEnumerable<string> targetIds = this.EdgeData.EdgeAssigner is null
            ? this.EdgeData.SinkIds
            : this.EdgeData.EdgeAssigner(message, this.EdgeData.SinkIds.Count)
                .Select(i => this.EdgeData.SinkIds[i]);
        
        Executor[] result = await Task.WhenAll(targetIds
            .Where(IsValidTarget)
            .Select(tid => this.RunContext.EnsureExecutorAsync(tid, stepTracer).AsTask()));
        // ...
    }
}

// FanInEdgeRunner.cs - Buffered collection until all sources ready
internal sealed class FanInEdgeRunner : EdgeRunner<FanInEdgeData>, IStatefulEdgeRunner
{
    private FanInEdgeState _state;
    
    protected internal override async ValueTask<DeliveryMapping?> ChaseEdgeAsync(...)
    {
        IEnumerable<MessageEnvelope>? releasedMessages = 
            this._state.ProcessMessage(envelope.SourceId, envelope);
        if (releasedMessages is null) return null; // Not ready yet
        // ...
    }
}
```

#### Python Pattern

```python
# _edge_runner.py
class FanOutEdgeRunner(EdgeRunner):
    async def send_message(self, message, shared_state, ctx) -> bool:
        selection_results = (
            self._selection_func(message.data, self._target_ids) 
            if self._selection_func 
            else self._target_ids
        )
        # Deliver to all selected targets concurrently
        tasks = [self._execute_on_target(tid, ...) for tid in deliverable_targets]
        await asyncio.gather(*tasks)

# _concurrent.py - High-level concurrent orchestration
class ConcurrentBuilder:
    def build(self) -> Workflow:
        # Wires: dispatcher -> fan-out -> participants -> fan-in -> aggregator
```

### 3.6 Event Streaming

#### .NET Pattern

```csharp
// Workflow execution returns IAsyncEnumerable<WorkflowEvent>
public async IAsyncEnumerable<WorkflowEvent> RunStreamingAsync(...)

// Events derive from WorkflowEvent base
[JsonDerivedType(typeof(ExecutorEvent))]
[JsonDerivedType(typeof(SuperStepEvent))]
[JsonDerivedType(typeof(WorkflowStartedEvent))]
[JsonDerivedType(typeof(WorkflowErrorEvent))]
public class WorkflowEvent(object? data = null)
{
    public object? Data => data;
}
```

#### Python Pattern

```python
# _workflow.py execution methods
async def run(self, message, ...) -> WorkflowRunResult:
    """Execute to completion, returns WorkflowRunResult with all events."""

async def run_stream(self, message, ...) -> AsyncIterable[WorkflowEvent]:
    """Returns async generator yielding events as they occur."""

# WorkflowRunState enum for status tracking
class WorkflowRunState(str, Enum):
    STARTED = "STARTED"
    IN_PROGRESS = "IN_PROGRESS"
    IN_PROGRESS_PENDING_REQUESTS = "IN_PROGRESS_PENDING_REQUESTS"
    IDLE = "IDLE"
    IDLE_WITH_PENDING_REQUESTS = "IDLE_WITH_PENDING_REQUESTS"
    FAILED = "FAILED"
    CANCELLED = "CANCELLED"
```

---

## 4. Design Recommendations for Go Implementation

### 4.1 Package Structure

```
go/
├── workflow/
│   ├── workflow.go          # Core Workflow type
│   ├── builder.go           # WorkflowBuilder with fluent API
│   ├── executor.go          # Executor interface and base type
│   ├── context.go           # WorkflowContext interface
│   ├── edge.go              # Edge, EdgeKind, EdgeData types
│   ├── events.go            # WorkflowEvent hierarchy
│   ├── checkpoint.go        # Checkpoint and CheckpointStorage
│   └── runner/
│       ├── runner.go        # Pregel superstep runner
│       ├── edge_runner.go   # EdgeRunner interface
│       ├── direct_edge.go   # DirectEdgeRunner
│       ├── fanout_edge.go   # FanOutEdgeRunner
│       └── fanin_edge.go    # FanInEdgeRunner (with state)
├── agentworkflow/            # High-level agent workflow builders
│   ├── sequential.go
│   ├── concurrent.go
│   ├── handoffs.go
│   └── groupchat.go
```

### 4.2 Core Interface Recommendations

```go
// Executor interface
type Executor interface {
    ID() string
    Execute(ctx context.Context, message any, wfCtx WorkflowContext) error
    CanHandle(messageType reflect.Type) bool
}

// WorkflowContext interface
type WorkflowContext interface {
    SendMessage(ctx context.Context, message any, targetID *string) error
    YieldOutput(ctx context.Context, output any) error
    RequestHalt(ctx context.Context) error
    ReadState(ctx context.Context, key string, scope *string) (any, error)
    WriteState(ctx context.Context, key string, value any, scope *string) error
    AddEvent(ctx context.Context, event WorkflowEvent) error
}

// EdgeKind enum
type EdgeKind int
const (
    EdgeKindDirect EdgeKind = iota
    EdgeKindFanOut
    EdgeKindFanIn
)

// Edge structure
type Edge struct {
    Kind      EdgeKind
    SourceID  string
    TargetID  string  // or TargetIDs []string for FanOut
    Condition func(message any) bool
}
```

### 4.3 Builder Pattern

```go
// WorkflowBuilder with method chaining
type WorkflowBuilder struct {
    executors map[string]ExecutorBinding
    edges     map[string][]Edge
    startID   string
    name      string
}

func NewWorkflowBuilder(start ExecutorBinding) *WorkflowBuilder
func (b *WorkflowBuilder) AddEdge(source, target ExecutorBinding) *WorkflowBuilder
func (b *WorkflowBuilder) AddConditionalEdge(source, target ExecutorBinding, cond func(any) bool) *WorkflowBuilder
func (b *WorkflowBuilder) AddFanOutEdge(source ExecutorBinding, targets []ExecutorBinding) *WorkflowBuilder
func (b *WorkflowBuilder) AddFanInEdge(sources []ExecutorBinding, target ExecutorBinding) *WorkflowBuilder
func (b *WorkflowBuilder) WithName(name string) *WorkflowBuilder
func (b *WorkflowBuilder) WithOutput(executors ...ExecutorBinding) *WorkflowBuilder
func (b *WorkflowBuilder) Build() (*Workflow, error)

// High-level agent workflow builders
func BuildSequential(agents ...Agent) (*Workflow, error)
func BuildConcurrent(agents []Agent, aggregator AggregatorFunc) (*Workflow, error)
```

### 4.4 Key Implementation Considerations

1. **Use Go's context.Context** for cancellation and deadline propagation through all async operations.

2. **Channel-based event streaming**: Use Go channels for event streaming:
   ```go
   func (w *Workflow) RunStream(ctx context.Context, input any) (<-chan WorkflowEvent, error)
   ```

3. **Goroutine-based parallelism**: Use goroutines with `sync.WaitGroup` or `errgroup.Group` for fan-out execution.

4. **Interface-based design**: Define small, focused interfaces (`Executor`, `EdgeRunner`, `CheckpointStorage`) to enable testing and extensibility.

5. **Functional options** for configuration:
   ```go
   type WorkflowOption func(*workflowConfig)
   func WithCheckpointing(storage CheckpointStorage) WorkflowOption
   func WithMaxIterations(max int) WorkflowOption
   ```

6. **Type-safe message routing**: Consider using generics for type-safe handler registration:
   ```go
   type Handler[T any] func(ctx context.Context, msg T, wfCtx WorkflowContext) error
   ```

7. **Fan-In state management**: The FanInEdgeRunner needs to maintain state across supersteps. Use a mutex-protected map or atomic operations for thread safety.

8. **Checkpoint serialization**: Use `encoding/json` for checkpoint serialization, with custom marshalers for complex types.

---

## 5. Observations

### Similarities Between Implementations

1. **Pregel-like execution model** with supersteps
2. **DAG-based workflow graph** with nodes (executors) and edges
3. **Fan-out/fan-in patterns** for parallel execution
4. **Event streaming** for observability
5. **Checkpoint/restore** for durability
6. **Fluent builder APIs** for workflow construction
7. **Request/response pattern** for human-in-the-loop scenarios

### Key Differences

| Aspect | .NET | Python |
|--------|------|--------|
| **Handler Registration** | `ConfigureRoutes(RouteBuilder)` override | `@handler` decorator |
| **Edge Grouping** | Implicit in edge data types | Explicit `EdgeGroup` classes |
| **Status Tracking** | Events imply status | Explicit `WorkflowRunState` enum |
| **High-level Builders** | Static methods on `AgentWorkflowBuilder` | Separate builder classes |
| **Type Discovery** | Reflection at runtime | Decorator + inspection at init |
| **Async Pattern** | `ValueTask` / `IAsyncEnumerable` | `async/await` with `AsyncGenerator` |

---

## 6. References

- .NET Workflow Project: `dotnet/src/Microsoft.Agents.AI.Workflows/`
- Python Workflow Module: `python/packages/core/agent_framework/_workflows/`
- Execution Folder (.NET): `dotnet/src/Microsoft.Agents.AI.Workflows/Execution/`
- Checkpointing (.NET): `dotnet/src/Microsoft.Agents.AI.Workflows/Checkpointing/`
- Sample Orchestration (Python): `python/samples/getting_started/workflows/orchestration/`
- Unit Tests (.NET): `dotnet/tests/Microsoft.Agents.AI.Workflows.UnitTests/`
- Unit Tests (Python): `python/packages/core/tests/workflow/`
