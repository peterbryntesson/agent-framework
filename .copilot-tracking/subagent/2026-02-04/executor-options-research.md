# ExecutorOptions and WorkflowContext Enhancements Research

**Date:** 2026-02-04
**Focus:** ExecutorOptions, YieldOutput, RequestHalt, and State Scope methods for Go implementation

---

## 1. ExecutorOptions from .NET

### Source File

[dotnet/src/Microsoft.Agents.AI.Workflows/ExecutorOptions.cs](../../../dotnet/src/Microsoft.Agents.AI.Workflows/ExecutorOptions.cs)

### Complete ExecutorOptions Definition

```csharp
public class ExecutorOptions
{
    public static ExecutorOptions Default { get; } = new();

    internal ExecutorOptions() { }

    // If true, the result of a message handler that returns a value 
    // will be sent as a message from the executor
    public bool AutoSendMessageHandlerResultObject { get; set; } = true;

    // If true, the result of a message handler that returns a value 
    // will be yielded as an output of the executor
    public bool AutoYieldOutputHandlerResultObject { get; set; } = true;
}
```

### Options Summary

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `AutoSendMessageHandlerResultObject` | `bool` | `true` | Auto-send handler return values as messages to downstream executors |
| `AutoYieldOutputHandlerResultObject` | `bool` | `true` | Auto-yield handler return values as workflow outputs (WorkflowOutputEvent) |

### Go Implementation Recommendation

```go
// ExecutorOptions configures executor behavior within workflows.
type ExecutorOptions struct {
    // AutoSendMessageResult controls whether handler return values
    // are automatically sent as messages to downstream executors.
    // Default: true
    AutoSendMessageResult bool

    // AutoYieldOutputResult controls whether handler return values
    // are automatically yielded as workflow outputs.
    // Default: true
    AutoYieldOutputResult bool
}

// DefaultExecutorOptions returns the default executor options.
func DefaultExecutorOptions() ExecutorOptions {
    return ExecutorOptions{
        AutoSendMessageResult: true,
        AutoYieldOutputResult: true,
    }
}
```

---

## 2. YieldOutput Semantics

### How YieldOutput Works

#### .NET Implementation

**Source:** [IWorkflowContext.cs#L45-50](../../../dotnet/src/Microsoft.Agents.AI.Workflows/IWorkflowContext.cs#L45-50)

```csharp
/// <summary>
/// Adds an output value to the workflow's output queue. These outputs will be 
/// bubbled out of the workflow using the WorkflowOutputEvent
/// </summary>
ValueTask YieldOutputAsync(object output, CancellationToken cancellationToken = default);
```

**Internal Implementation:** [InProcessRunnerContext.cs#L231-244](../../../dotnet/src/Microsoft.Agents.AI.Workflows/InProc/InProcessRunnerContext.cs#L231-244)

```csharp
private async ValueTask YieldOutputAsync(string sourceId, object output, CancellationToken ct)
{
    this.CheckEnded();
    Throw.IfNull(output);

    Executor sourceExecutor = await this.EnsureExecutorAsync(sourceId, tracer: null, ct);
    if (!sourceExecutor.CanOutput(output.GetType()))
    {
        throw new InvalidOperationException($"Cannot output object of type {output.GetType().Name}");
    }

    if (this._outputFilter.CanOutput(sourceId, output))
    {
        await this.AddEventAsync(new WorkflowOutputEvent(output, sourceId), ct);
    }
}
```

#### Python Implementation

**Source:** [_workflow_context.py#L350-363](../../../python/packages/core/agent_framework/_workflows/_workflow_context.py#L350-363)

```python
async def yield_output(self, output: T_W_Out) -> None:
    """Set the output of the workflow."""
    # Track yielded output for ExecutorCompletedEvent
    self._yielded_outputs.append(copy.deepcopy(output))

    with _framework_event_origin():
        event = WorkflowOutputEvent(data=output, executor_id=self._executor_id)
    await self._runner_context.add_event(event)
```

### WorkflowOutputEvent Structure

**Source:** [WorkflowOutputEvent.cs](../../../dotnet/src/Microsoft.Agents.AI.Workflows/WorkflowOutputEvent.cs)

```csharp
public sealed class WorkflowOutputEvent : WorkflowEvent
{
    internal WorkflowOutputEvent(object data, string sourceId) : base(data)
    {
        this.SourceId = sourceId;
    }

    public string SourceId { get; }
    
    public bool Is<T>() => this.IsType(typeof(T));
    public bool Is<T>([NotNullWhen(true)] out T? maybeValue) { ... }
    public T? As<T>() => this.Data is T value ? value : default;
}
```

### Key Semantics

1. **Event-based output** - YieldOutput creates a `WorkflowOutputEvent` containing the data and source executor ID
2. **Type validation** - The executor validates that the output type matches declared output types
3. **Output filtering** - An output filter can conditionally allow/block outputs
4. **Event queue** - Output events are added to the workflow's event queue for streaming to callers
5. **Tracking** - Python tracks yielded outputs for `ExecutorCompletedEvent`

### Go Implementation Recommendation

```go
// WorkflowOutputEvent represents output yielded by an executor.
type WorkflowOutputEvent struct {
    // Data is the output value
    Data interface{}
    
    // SourceID is the executor that yielded this output
    SourceID string
}

// YieldOutput queues an output to be emitted as a WorkflowOutputEvent.
func (wc *WorkflowContext) YieldOutput(output interface{}) {
    wc.mu.Lock()
    defer wc.mu.Unlock()
    wc.outputs = append(wc.outputs, WorkflowOutputEvent{
        Data:     output,
        SourceID: wc.executorID,
    })
}

// Outputs returns the outputs queued during this handler execution.
func (wc *WorkflowContext) Outputs() []WorkflowOutputEvent {
    wc.mu.Lock()
    defer wc.mu.Unlock()
    result := make([]WorkflowOutputEvent, len(wc.outputs))
    copy(result, wc.outputs)
    return result
}
```

---

## 3. RequestHalt Semantics

### How RequestHalt Works

#### .NET Implementation

**Declaration:** [IWorkflowContext.cs#L53-55](../../../dotnet/src/Microsoft.Agents.AI.Workflows/IWorkflowContext.cs#L53-55)

```csharp
/// <summary>
/// Adds a request to "halt" workflow execution at the end of the current SuperStep.
/// </summary>
ValueTask RequestHaltAsync();
```

**Implementation:** [InProcessRunnerContext.cs#L308](../../../dotnet/src/Microsoft.Agents.AI.Workflows/InProc/InProcessRunnerContext.cs#L308)

```csharp
public ValueTask RequestHaltAsync() => this.AddEventAsync(new RequestHaltEvent());
```

#### RequestHaltEvent Definition

**Source:** [RequestHaltEvent.cs](../../../dotnet/src/Microsoft.Agents.AI.Workflows/RequestHaltEvent.cs)

```csharp
/// <summary>
/// Event triggered when a workflow completes execution.
/// </summary>
internal sealed class RequestHaltEvent : WorkflowEvent
{
    internal RequestHaltEvent(object? result = null) : base(result) { }
}
```

### How the Execution Loop Handles RequestHalt

**Source:** [LockstepRunEventStream.cs#L96-122](../../../dotnet/src/Microsoft.Agents.AI.Workflows/Execution/LockstepRunEventStream.cs#L96-122)

```csharp
bool hadRequestHaltEvent = false;
foreach (WorkflowEvent raisedEvent in Interlocked.Exchange(ref eventSink, []))
{
    if (linkedSource.Token.IsCancellationRequested)
    {
        yield break; // Exit if cancellation is requested
    }

    // Interpret RequestHaltEvent as a termination request
    if (raisedEvent is RequestHaltEvent)
    {
        hadRequestHaltEvent = true;
    }
    else
    {
        yield return raisedEvent;
    }
}

if (hadRequestHaltEvent || linkedSource.Token.IsCancellationRequested)
{
    // If we had a halt event, we are done.
    yield break;
}
```

### Key Semantics

1. **Internal event** - `RequestHaltEvent` is an internal signaling event, not exposed to callers
2. **Deferred halt** - The halt happens at the end of the current superstep, not immediately
3. **Event filtering** - The event is NOT yielded to the event stream, only used for control flow
4. **Clean termination** - The workflow terminates gracefully after processing pending messages
5. **Result passing** - Can optionally pass a result object with the halt event

### Python Differences

Python does not currently implement `RequestHalt` - the workflow runs until convergence (no messages) or status becomes `IDLE`.

### Go Implementation Recommendation

```go
// Add to WorkflowContext
type WorkflowContext struct {
    // ... existing fields ...
    
    // haltRequested signals that the workflow should stop after this superstep
    haltRequested bool
}

// RequestHalt requests the workflow to halt after the current superstep.
// This is a graceful termination - pending messages in this superstep are processed.
func (wc *WorkflowContext) RequestHalt() {
    wc.mu.Lock()
    defer wc.mu.Unlock()
    wc.haltRequested = true
}

// HaltRequested returns true if RequestHalt was called.
func (wc *WorkflowContext) HaltRequested() bool {
    wc.mu.Lock()
    defer wc.mu.Unlock()
    return wc.haltRequested
}
```

**Runner Integration:**

```go
// In executeSuperstep, after executing all executors:
for _, wCtx := range executedContexts {
    if wCtx.HaltRequested() {
        return nil, outputs, ErrHaltRequested
    }
}
```

---

## 4. State Scope Methods

### .NET IWorkflowContext State Methods

**Source:** [IWorkflowContext.cs#L60-185](../../../dotnet/src/Microsoft.Agents.AI.Workflows/IWorkflowContext.cs#L60-185)

| Method | Description |
|--------|-------------|
| `ReadStateAsync<T>(key, scopeName?)` | Read a state value from the workflow's state store |
| `ReadOrInitStateAsync<T>(key, factory, scopeName?)` | Read state or initialize if not exists |
| `ReadStateKeysAsync(scopeName?)` | Get all state keys in a scope |
| `QueueStateUpdateAsync<T>(key, value, scopeName?)` | Queue a state update (visible next superstep) |
| `QueueClearScopeAsync(scopeName?)` | Clear all state entries in a scope |

### State Scope Concept

The .NET implementation uses **scoped state**:

- Each executor has a **default scope** based on its ID
- State can be read/written to custom named scopes
- State updates are queued and published at superstep boundaries
- Other executors see state changes in the next superstep

**StateManager Implementation:** [Execution/StateManager.cs](../../../dotnet/src/Microsoft.Agents.AI.Workflows/Execution/StateManager.cs)

```csharp
internal sealed class StateManager
{
    private readonly Dictionary<ScopeId, StateScope> _scopes = [];
    private readonly Dictionary<UpdateKey, StateUpdate> _queuedUpdates = [];

    // ScopeId combines executorId + optional scopeName
    // Updates are queued and published at superstep boundaries
}
```

### Python Shared State

**Source:** [_workflow_context.py#L413-420](../../../python/packages/core/agent_framework/_workflows/_workflow_context.py#L413-420)

```python
async def get_shared_state(self, key: str) -> Any:
    """Get a value from the shared state."""
    return await self._shared_state.get(key)

async def set_shared_state(self, key: str, value: Any) -> None:
    """Set a value in the shared state."""
    await self._shared_state.set(key, value)
```

Python uses a simpler **SharedState** model without explicit scoping.

### Current Go Implementation

**Source:** [context.go#L77-83](../../../go/workflow/context.go#L77-83)

```go
func (wc *WorkflowContext) GetState(key string) (interface{}, bool) {
    return wc.state.Load(key)
}

func (wc *WorkflowContext) SetState(key string, value interface{}) {
    wc.state.Store(key, value)
}
```

Current Go implementation:
- Uses `sync.Map` for thread-safe access
- No scoping mechanism
- Immediate visibility (no superstep boundaries)

### Go Implementation Recommendation

For parity with .NET, add scoped state support:

```go
// StateScope represents a scoped state namespace.
type StateScope struct {
    ExecutorID string
    ScopeName  string // Empty string = default scope
}

// ScopedState manages scoped workflow state with deferred updates.
type ScopedState struct {
    mu           sync.RWMutex
    scopes       map[StateScope]map[string]interface{}
    queuedUpdate map[StateScope]map[string]*stateUpdate
}

type stateUpdate struct {
    value    interface{}
    isDelete bool
}

// WorkflowContext enhancements
func (wc *WorkflowContext) ReadState(key string, scopeName ...string) (interface{}, bool)
func (wc *WorkflowContext) ReadOrInitState(key string, factory func() interface{}, scopeName ...string) interface{}
func (wc *WorkflowContext) QueueStateUpdate(key string, value interface{}, scopeName ...string)
func (wc *WorkflowContext) QueueClearScope(scopeName ...string)
func (wc *WorkflowContext) ReadStateKeys(scopeName ...string) []string
```

---

## 5. Integration with Execution Loop

### Current Go Runner Flow

1. **Create initial message** to start executor
2. **Execute supersteps** until convergence or max limit
3. Each superstep:
   - Group messages by target executor
   - Execute all executors in parallel
   - Collect outbox messages for next superstep
   - Collect outputs for result

### Required Changes for New Features

```go
// executeSuperstep modifications:

func (r *WorkflowRunner) executeSuperstep(...) ([]WorkflowMessage, []WorkflowOutputEvent, bool, error) {
    // ... existing message grouping ...

    var allNewMessages []WorkflowMessage
    var outputs []WorkflowOutputEvent      // Changed from []WorkflowMessage
    var haltRequested bool                  // New: track halt requests
    
    for executorID, execMessages := range messagesByExecutor {
        // ... execute executor ...
        
        // Collect outputs (new)
        outputs = append(outputs, wCtx.Outputs()...)
        
        // Check for halt request (new)
        if wCtx.HaltRequested() {
            haltRequested = true
        }
        
        // Apply executor options for auto-send/auto-yield (new)
        if r.options.executorOptions.AutoSendMessageResult {
            // Handle return value as message
        }
        if r.options.executorOptions.AutoYieldOutputResult {
            // Handle return value as output
        }
    }

    // Publish state updates (new)
    r.stateManager.PublishUpdates()

    return allNewMessages, outputs, haltRequested, nil
}
```

### Run Loop with Halt Support

```go
func (r *WorkflowRunner) Run(ctx context.Context, input string) (*WorkflowResult, error) {
    // ... initialization ...

    for superstep := 0; superstep < r.options.maxSupersteps; superstep++ {
        newMessages, stepOutputs, haltRequested, err := r.executeSuperstep(...)
        if err != nil {
            return nil, err
        }

        outputs = append(outputs, stepOutputs...)

        // Check for halt request (new)
        if haltRequested {
            break
        }

        // Check for convergence
        if len(newMessages) == 0 {
            break
        }

        messages = newMessages
    }

    return &WorkflowResult{...}, nil
}
```

---

## 6. Summary of Recommended Changes

### WorkflowContext Additions

| Method | Purpose | Priority |
|--------|---------|----------|
| `YieldOutput(output interface{})` | Emit workflow output event | High |
| `RequestHalt()` | Request graceful termination | High |
| `HaltRequested() bool` | Check if halt was requested | High |
| `Outputs() []WorkflowOutputEvent` | Get queued outputs | High |
| `ReadState(key, scope?) interface{}` | Read scoped state | Medium |
| `QueueStateUpdate(key, value, scope?)` | Queue state change | Medium |
| `QueueClearScope(scope?)` | Clear scope state | Low |
| `ReadStateKeys(scope?) []string` | List keys in scope | Low |

### New Types

| Type | Purpose |
|------|---------|
| `ExecutorOptions` | Configure auto-send/auto-yield behavior |
| `WorkflowOutputEvent` | Represent yielded output with source ID |
| `ScopedState` | Manage scoped state with deferred updates |

### Runner Modifications

1. **Add `ExecutorOptions`** to runner configuration
2. **Collect outputs** from `WorkflowContext.Outputs()` after execution
3. **Check halt requests** and break loop if requested
4. **Publish state updates** at superstep boundaries
5. **Emit `WorkflowOutputEvent`** for streaming runs

---

## 7. Implementation Priority

1. **Phase 1: YieldOutput and RequestHalt** (High priority)
   - Add `YieldOutput()` to WorkflowContext
   - Add `WorkflowOutputEvent` type
   - Add `RequestHalt()` to WorkflowContext
   - Update runner to handle halt requests

2. **Phase 2: ExecutorOptions** (Medium priority)
   - Add `ExecutorOptions` struct
   - Integrate options into runner execution
   - Handle auto-send/auto-yield behavior

3. **Phase 3: Scoped State** (Lower priority)
   - Implement `ScopedState` manager
   - Add scoped state methods to WorkflowContext
   - Implement deferred update publication
