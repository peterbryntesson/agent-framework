# StatefulExecutor Research for Go Implementation

**Date**: 2026-02-04  
**Purpose**: Research StatefulExecutor patterns from .NET and Python to guide Go implementation

---

## Executive Summary

StatefulExecutor extends the base Executor with type-safe, managed state that persists across workflow runs and supports checkpointing. It provides:

1. **Generic typed state** (`TState`) with initialization factory
2. **Automatic state caching** (for non-concurrent runs)
3. **State key management** with optional scoping
4. **Lifecycle hooks** for state read/write operations
5. **Reset capability** for workflow reuse

---

## 1. .NET Implementation Analysis

### 1.1 StatefulExecutor Class Definition

```csharp
// From: dotnet/src/Microsoft.Agents.AI.Workflows/StatefulExecutor.cs

public abstract class StatefulExecutor<TState> : Executor
{
    private readonly Func<TState> _initialStateFactory;
    private TState? _stateCache;

    protected StatefulExecutor(
        string id,
        Func<TState> initialStateFactory,
        StatefulExecutorOptions? options = null,
        bool declareCrossRunShareable = false)
        : base(id, options ?? new StatefulExecutorOptions(), declareCrossRunShareable)
    {
        this.Options = (StatefulExecutorOptions)base.Options;
        this._initialStateFactory = Throw.IfNull(initialStateFactory);
    }

    protected new StatefulExecutorOptions Options { get; }
    private string DefaultStateKey => $"{this.GetType().Name}.State";
    protected string StateKey => this.Options.StateKey ?? this.DefaultStateKey;
}
```

### 1.2 StatefulExecutorOptions

```csharp
// From: dotnet/src/Microsoft.Agents.AI.Workflows/StatefulExecutorOptions.cs

public class StatefulExecutorOptions : ExecutorOptions
{
    // Unique key for state identification
    // Default: "{ExecutorType}.State"
    public string? StateKey { get; set; }

    // Scope name for shared state access
    // Default: null (private to executor instance)
    public string? ScopeName { get; set; }
}
```

### 1.3 Base ExecutorOptions

```csharp
// From: dotnet/src/Microsoft.Agents.AI.Workflows/ExecutorOptions.cs

public class ExecutorOptions
{
    public static ExecutorOptions Default { get; } = new();

    // Auto-send handler results as messages to connected executors
    public bool AutoSendMessageHandlerResultObject { get; set; } = true;

    // Auto-yield handler results as workflow outputs
    public bool AutoYieldOutputHandlerResultObject { get; set; } = true;
}
```

---

## 2. State Persistence Mechanism

### 2.1 Core State Operations

StatefulExecutor provides three primary state operations:

```csharp
// READ: Get state, initializing if needed
protected async ValueTask<TState> ReadStateAsync(
    IWorkflowContext context,
    bool skipCache = false,
    CancellationToken cancellationToken = default)
{
    // Cache hit (for non-concurrent mode)
    if (!skipCache && this._stateCache is not null)
        return this._stateCache;

    // Read from context (initializes if missing)
    TState? state = await context.ReadOrInitStateAsync(
        this.StateKey,
        this._initialStateFactory,
        this.Options.ScopeName,
        cancellationToken).ConfigureAwait(false);

    // Update cache if not concurrent
    if (!context.ConcurrentRunsEnabled)
        this._stateCache = state;

    return state;
}

// WRITE: Queue state update
protected ValueTask QueueStateUpdateAsync(
    TState state,
    IWorkflowContext context,
    CancellationToken cancellationToken = default)
{
    if (!context.ConcurrentRunsEnabled)
        this._stateCache = state;

    return context.QueueStateUpdateAsync(
        this.StateKey,
        state,
        this.Options.ScopeName,
        cancellationToken);
}

// READ-MODIFY-WRITE: Atomic state transformation
protected async ValueTask InvokeWithStateAsync(
    Func<TState, IWorkflowContext, CancellationToken, ValueTask<TState?>> invocation,
    IWorkflowContext context,
    bool skipCache = false,
    CancellationToken cancellationToken = default);
```

### 2.2 IWorkflowContext State API

```csharp
// From: dotnet/src/Microsoft.Agents.AI.Workflows/IWorkflowContext.cs

public interface IWorkflowContext
{
    // Read state (returns null if not found)
    ValueTask<T?> ReadStateAsync<T>(string key, string? scopeName = null, CancellationToken ct = default);

    // Read or initialize state
    ValueTask<T> ReadOrInitStateAsync<T>(string key, Func<T> factory, string? scopeName = null, CancellationToken ct = default);

    // Queue state update (visible to this executor immediately, others next superstep)
    ValueTask QueueStateUpdateAsync<T>(string key, T? value, string? scopeName = null, CancellationToken ct = default);

    // Read all keys in scope
    ValueTask<HashSet<string>> ReadStateKeysAsync(string? scopeName = null, CancellationToken ct = default);

    // Clear entire scope
    ValueTask QueueClearScopeAsync(string? scopeName = null, CancellationToken ct = default);

    // Whether concurrent runs are enabled (affects caching)
    bool ConcurrentRunsEnabled { get; }
}
```

### 2.3 StateManager Implementation

The StateManager handles state storage with:

- **Scoped storage**: State is organized by `ScopeId` (executorId + optional scopeName)
- **Queued updates**: Changes are batched and published at superstep boundaries
- **Export/Import**: Full state can be serialized for checkpointing

```csharp
// From: dotnet/src/Microsoft.Agents.AI.Workflows/Execution/StateManager.cs

internal sealed class StateManager
{
    private readonly Dictionary<ScopeId, StateScope> _scopes = [];
    private readonly Dictionary<UpdateKey, StateUpdate> _queuedUpdates = [];

    // Export all state for checkpointing
    internal async ValueTask<Dictionary<ScopeKey, PortableValue>> ExportStateAsync();

    // Import state from checkpoint
    internal ValueTask ImportStateAsync(Checkpoint checkpoint);

    // Publish queued updates (called at superstep end)
    public async ValueTask PublishUpdatesAsync(IStepTracer? tracer);
}
```

---

## 3. Lifecycle Methods

### 3.1 Base Executor Lifecycle

```csharp
// From: dotnet/src/Microsoft.Agents.AI.Workflows/Executor.cs

public abstract class Executor
{
    // Called once per executor instance during initialization
    protected internal virtual ValueTask InitializeAsync(
        IWorkflowContext context,
        CancellationToken cancellationToken = default) => default;

    // Called before checkpoint save
    protected internal virtual ValueTask OnCheckpointingAsync(
        IWorkflowContext context,
        CancellationToken cancellationToken = default) => default;

    // Called after checkpoint restore
    protected internal virtual ValueTask OnCheckpointRestoredAsync(
        IWorkflowContext context,
        CancellationToken cancellationToken = default) => default;
}
```

### 3.2 IResettableExecutor Interface

```csharp
// From: dotnet/src/Microsoft.Agents.AI.Workflows/IResettableExecutor.cs

public interface IResettableExecutor
{
    // Reset executor to initial state (for workflow reuse)
    ValueTask ResetAsync()
#if NET
    {
        return default;
    }
#else
    ;
#endif
}
```

### 3.3 StatefulExecutor Reset

```csharp
// StatefulExecutor provides reset implementation
protected ValueTask ResetAsync()
{
    this._stateCache = this._initialStateFactory();
    return default;
}
```

---

## 4. Python Implementation Analysis

### 4.1 Executor Checkpoint Hooks

```python
# From: python/packages/core/agent_framework/_workflows/_executor.py

class Executor:
    async def on_checkpoint_save(self) -> dict[str, Any]:
        """Hook called when the workflow is being saved to a checkpoint.
        
        Override to return state dictionary for checkpointing.
        Only JSON-serializable data allowed.
        """
        return {}

    async def on_checkpoint_restore(self, state: dict[str, Any]) -> None:
        """Hook called when the workflow is restored from a checkpoint.
        
        Args:
            state: The state dictionary saved during checkpointing.
        """
        ...
```

### 4.2 AgentExecutor State Management

```python
# From: python/packages/core/agent_framework/_workflows/_agent_executor.py

class AgentExecutor(Executor):
    def __init__(self, agent, *, agent_thread=None, output_response=False, id=None):
        # Internal cache of messages between runs
        self._cache: list[ChatMessage] = []
        # Full conversation history
        self._full_conversation: list[ChatMessage] = []
        # Pending user input requests
        self._pending_agent_requests: dict[str, Content] = {}
        self._pending_responses_to_agent: list[Content] = []

    async def on_checkpoint_save(self) -> dict[str, Any]:
        serialized_thread = await self._agent_thread.serialize()
        return {
            "cache": encode_chat_messages(self._cache),
            "full_conversation": encode_chat_messages(self._full_conversation),
            "agent_thread": serialized_thread,
            "pending_agent_requests": encode_checkpoint_value(self._pending_agent_requests),
            "pending_responses_to_agent": encode_checkpoint_value(self._pending_responses_to_agent),
        }

    async def on_checkpoint_restore(self, state: dict[str, Any]) -> None:
        # Restore cache, full_conversation, agent_thread, pending requests...
        cache_payload = state.get("cache")
        if cache_payload:
            self._cache = decode_chat_messages(cache_payload)
        # ... etc

    def reset(self) -> None:
        """Reset the internal cache."""
        self._cache.clear()
```

### 4.3 SharedState for Workflow-Level State

```python
# From: python/packages/core/agent_framework/_workflows/_shared_state.py

class SharedState:
    """Thread-safe shared state across executors."""

    async def set(self, key: str, value: Any) -> None: ...
    async def get(self, key: str) -> Any: ...
    async def has(self, key: str) -> bool: ...
    async def delete(self, key: str) -> None: ...
    async def clear(self) -> None: ...
    async def export_state(self) -> dict[str, Any]: ...
    async def import_state(self, state: dict[str, Any]) -> None: ...
```

---

## 5. Key Differences: Executor vs StatefulExecutor

| Aspect | Executor | StatefulExecutor |
|--------|----------|------------------|
| **State** | None built-in | Typed `TState` with factory |
| **Caching** | N/A | Automatic cache (non-concurrent) |
| **State Key** | N/A | Auto-generated or custom |
| **Scoping** | N/A | Optional scope for sharing |
| **Reset** | Optional interface | Built-in reset method |
| **Concurrency** | Not state-aware | Handles concurrent runs |

---

## 6. Recommended Go Implementation

### 6.1 StatefulExecutor Interface

```go
// StatefulExecutor extends Executor with managed state.
type StatefulExecutor[TState any] interface {
    Executor

    // StateKey returns the key used for state storage.
    StateKey() string

    // InitialState creates the initial state value.
    InitialState() TState

    // ReadState retrieves current state from context.
    ReadState(ctx context.Context, wCtx *WorkflowContext) (TState, error)

    // WriteState queues a state update.
    WriteState(ctx context.Context, wCtx *WorkflowContext, state TState) error

    // Reset returns executor to initial state.
    Reset() error
}
```

### 6.2 StatefulExecutorBase Implementation

```go
// StatefulExecutorBase provides common stateful executor functionality.
type StatefulExecutorBase[TState any] struct {
    ExecutorBase
    opts             StatefulExecutorOptions
    initialFactory   func() TState
    stateCache       *TState
    stateCacheMu     sync.RWMutex
}

// StatefulExecutorOptions configures stateful executor behavior.
type StatefulExecutorOptions struct {
    ExecutorOptions

    // StateKey overrides the default state key ("{Type}.State")
    StateKey string

    // ScopeName for shared state access (nil = private)
    ScopeName string
}

// NewStatefulExecutorBase creates a new StatefulExecutorBase.
func NewStatefulExecutorBase[TState any](
    id string,
    initialFactory func() TState,
    opts *StatefulExecutorOptions,
) *StatefulExecutorBase[TState] {
    if opts == nil {
        opts = &StatefulExecutorOptions{}
    }
    return &StatefulExecutorBase[TState]{
        ExecutorBase:   NewExecutorBase(id),
        opts:           *opts,
        initialFactory: initialFactory,
    }
}

// StateKey returns the state storage key.
func (se *StatefulExecutorBase[TState]) StateKey() string {
    if se.opts.StateKey != "" {
        return se.opts.StateKey
    }
    return fmt.Sprintf("%T.State", *new(TState))
}

// ReadState retrieves state, initializing if needed.
func (se *StatefulExecutorBase[TState]) ReadState(
    ctx context.Context,
    wCtx *WorkflowContext,
) (TState, error) {
    // Check cache first (for non-concurrent mode)
    se.stateCacheMu.RLock()
    if se.stateCache != nil {
        cached := *se.stateCache
        se.stateCacheMu.RUnlock()
        return cached, nil
    }
    se.stateCacheMu.RUnlock()

    // Read from context
    state, err := wCtx.ReadOrInitState(
        se.StateKey(),
        se.opts.ScopeName,
        se.initialFactory,
    )
    if err != nil {
        return *new(TState), err
    }

    // Update cache
    se.stateCacheMu.Lock()
    se.stateCache = &state
    se.stateCacheMu.Unlock()

    return state, nil
}

// WriteState queues a state update.
func (se *StatefulExecutorBase[TState]) WriteState(
    ctx context.Context,
    wCtx *WorkflowContext,
    state TState,
) error {
    // Update cache
    se.stateCacheMu.Lock()
    se.stateCache = &state
    se.stateCacheMu.Unlock()

    return wCtx.QueueStateUpdate(se.StateKey(), se.opts.ScopeName, state)
}

// Reset returns to initial state.
func (se *StatefulExecutorBase[TState]) Reset() error {
    se.stateCacheMu.Lock()
    defer se.stateCacheMu.Unlock()
    initial := se.initialFactory()
    se.stateCache = &initial
    return nil
}
```

### 6.3 WorkflowContext State Extensions

```go
// Add to WorkflowContext for state management

// ReadOrInitState reads state or initializes with factory.
func (wc *WorkflowContext) ReadOrInitState[T any](
    key string,
    scope string,
    factory func() T,
) (T, error) {
    fullKey := wc.scopedKey(key, scope)
    if val, ok := wc.GetState(fullKey); ok {
        if typed, ok := val.(T); ok {
            return typed, nil
        }
        return *new(T), fmt.Errorf("state type mismatch for key %s", key)
    }
    initial := factory()
    wc.SetState(fullKey, initial)
    return initial, nil
}

// QueueStateUpdate queues a state update.
func (wc *WorkflowContext) QueueStateUpdate[T any](
    key string,
    scope string,
    value T,
) error {
    fullKey := wc.scopedKey(key, scope)
    wc.SetState(fullKey, value)
    return nil
}

func (wc *WorkflowContext) scopedKey(key, scope string) string {
    if scope == "" {
        return fmt.Sprintf("%s:%s", wc.executorID, key)
    }
    return fmt.Sprintf("%s:%s:%s", wc.executorID, scope, key)
}
```

### 6.4 Checkpointing Interfaces

```go
// Checkpointable allows executors to participate in checkpointing.
type Checkpointable interface {
    // OnCheckpointSave returns state to persist.
    OnCheckpointSave(ctx context.Context) (map[string]any, error)

    // OnCheckpointRestore restores from persisted state.
    OnCheckpointRestore(ctx context.Context, state map[string]any) error
}

// Resettable allows executors to be reset for reuse.
type Resettable interface {
    Reset() error
}
```

---

## 7. Test Strategy

### 7.1 Unit Tests

```go
// TestStatefulExecutorBase_ReadState
// - Verify initial state from factory
// - Verify cache hit on second read
// - Verify skipCache bypasses cache

// TestStatefulExecutorBase_WriteState
// - Verify state is written to context
// - Verify cache is updated
// - Verify subsequent reads return written value

// TestStatefulExecutorBase_Reset
// - Verify state returns to initial value
// - Verify cache is cleared

// TestStatefulExecutorBase_ConcurrentMode
// - Verify cache disabled when concurrent runs enabled
// - Verify multiple executors see consistent state
```

### 7.2 Integration Tests

```go
// TestStatefulExecutor_AcrossSupersteps
// - State persists across superstep boundaries
// - Updates visible to other executors in next superstep

// TestStatefulExecutor_Checkpointing
// - State survives checkpoint save/restore cycle
// - OnCheckpointSave captures all state
// - OnCheckpointRestore restores correctly

// TestStatefulExecutor_ScopedState
// - Private scope isolates state per executor
// - Named scope allows sharing between executors
```

### 7.3 Compatibility Tests

```go
// TestStatefulExecutor_WithAgentExecutor
// - AgentExecutor maintains conversation state
// - State persists across runs

// TestStatefulExecutor_WithWorkflowRunner
// - Full workflow execution with stateful executors
// - Checkpoint and resume works correctly
```

---

## 8. Implementation Notes

### 8.1 Go Generics Considerations

Go 1.18+ generics work differently from C# generics:
- Cannot use generic type parameters in interface methods directly
- Use concrete types or `any` with type assertions
- Consider separate generic wrapper vs interface approach

### 8.2 Concurrency

- Use `sync.RWMutex` for cache access (read-heavy)
- Respect `ConcurrentRunsEnabled` flag from context
- State updates are atomic within superstep

### 8.3 Serialization

For checkpointing, state must be JSON-serializable:
- Use `encoding/json` for marshal/unmarshal
- Define clear contracts for state types
- Consider `PortableValue` equivalent for type-safe variants

---

## 9. Files to Create/Modify

1. **New**: `go/workflow/stateful_executor.go` - StatefulExecutorBase implementation
2. **New**: `go/workflow/stateful_executor_options.go` - Options struct
3. **Modify**: `go/workflow/context.go` - Add state management methods
4. **New**: `go/workflow/checkpoint.go` - Checkpoint interfaces
5. **New**: `go/workflow/stateful_executor_test.go` - Unit tests
6. **New**: `go/workflow/executors/stateful_agent.go` - Stateful AgentExecutor

---

## 10. References

- [StatefulExecutor.cs](../../../dotnet/src/Microsoft.Agents.AI.Workflows/StatefulExecutor.cs)
- [StatefulExecutorOptions.cs](../../../dotnet/src/Microsoft.Agents.AI.Workflows/StatefulExecutorOptions.cs)
- [Executor.cs](../../../dotnet/src/Microsoft.Agents.AI.Workflows/Executor.cs)
- [IWorkflowContext.cs](../../../dotnet/src/Microsoft.Agents.AI.Workflows/IWorkflowContext.cs)
- [StateManager.cs](../../../dotnet/src/Microsoft.Agents.AI.Workflows/Execution/StateManager.cs)
- [_executor.py](../../../python/packages/core/agent_framework/_workflows/_executor.py)
- [_agent_executor.py](../../../python/packages/core/agent_framework/_workflows/_agent_executor.py)
- [_shared_state.py](../../../python/packages/core/agent_framework/_workflows/_shared_state.py)
