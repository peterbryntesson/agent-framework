# Group Chat Orchestration Implementation Analysis

**Date:** 2026-02-04  
**Status:** Complete  
**Research Task:** Analyze Group Chat Orchestration implementations in .NET and Python

---

## 1. Executive Summary

Both .NET and Python implementations provide robust group chat orchestration capabilities with:

- Abstract base classes for custom orchestrators
- Built-in round-robin selection
- LLM/Agent-based intelligent selection
- Termination condition handling
- Conversation history tracking and checkpointing

The Python implementation is more feature-rich with additional orchestration patterns (Magentic) and a more flexible builder API.

---

## 2. .NET Implementation

### 2.1 Core Files

| Component | File Path | Lines |
|-----------|-----------|-------|
| **GroupChatManager** (abstract base) | [dotnet/src/Microsoft.Agents.AI.Workflows/GroupChatManager.cs](dotnet/src/Microsoft.Agents.AI.Workflows/GroupChatManager.cs) | 1-79 |
| **RoundRobinGroupChatManager** | [dotnet/src/Microsoft.Agents.AI.Workflows/RoundRobinGroupChatManager.cs](dotnet/src/Microsoft.Agents.AI.Workflows/RoundRobinGroupChatManager.cs) | 1-72 |
| **GroupChatWorkflowBuilder** | [dotnet/src/Microsoft.Agents.AI.Workflows/GroupChatWorkflowBuilder.cs](dotnet/src/Microsoft.Agents.AI.Workflows/GroupChatWorkflowBuilder.cs) | 1-72 |
| **GroupChatHost** (internal executor) | [dotnet/src/Microsoft.Agents.AI.Workflows/Specialized/GroupChatHost.cs](dotnet/src/Microsoft.Agents.AI.Workflows/Specialized/GroupChatHost.cs) | 1-57 |
| **AgentWorkflowBuilder** (entry point) | [dotnet/src/Microsoft.Agents.AI.Workflows/AgentWorkflowBuilder.cs](dotnet/src/Microsoft.Agents.AI.Workflows/AgentWorkflowBuilder.cs#L175) | L175-181 |

### 2.2 GroupChatManager Abstract Class

```csharp
public abstract class GroupChatManager
{
    public int IterationCount { get; internal set; }
    public int MaximumIterationCount { get; set; } = 40;

    protected internal abstract ValueTask<AIAgent> SelectNextAgentAsync(
        IReadOnlyList<ChatMessage> history,
        CancellationToken cancellationToken = default);

    protected internal virtual ValueTask<IEnumerable<ChatMessage>> UpdateHistoryAsync(
        IReadOnlyList<ChatMessage> history,
        CancellationToken cancellationToken = default);

    protected internal virtual ValueTask<bool> ShouldTerminateAsync(
        IReadOnlyList<ChatMessage> history,
        CancellationToken cancellationToken = default);

    protected internal virtual void Reset();
}
```

**Key Features:**

- `SelectNextAgentAsync` - Abstract method for subclasses to implement selection logic
- `UpdateHistoryAsync` - Optional history filtering before passing to next agent
- `ShouldTerminateAsync` - Default checks `MaximumIterationCount`, overridable
- `Reset()` - Clears state for reuse

### 2.3 RoundRobinGroupChatManager

```csharp
public class RoundRobinGroupChatManager : GroupChatManager
{
    private readonly IReadOnlyList<AIAgent> _agents;
    private int _nextIndex;

    public RoundRobinGroupChatManager(
        IReadOnlyList<AIAgent> agents,
        Func<RoundRobinGroupChatManager, IEnumerable<ChatMessage>, CancellationToken, ValueTask<bool>>? shouldTerminateFunc = null);
}
```

**Selection Logic:** Cycles through agents in order using modulo arithmetic.

### 2.4 Builder Pattern

```csharp
// Entry point
AgentWorkflowBuilder.CreateGroupChatBuilderWith(agents => new RoundRobinGroupChatManager(agents) { MaximumIterationCount = 5 })
    .AddParticipants(agent1, agent2, agent3)
    .Build();
```

### 2.5 GroupChatHost (Internal Orchestration)

The `GroupChatHost` is an internal `Executor` that:

1. Handles incoming `TurnToken` messages to drive the conversation loop
2. Calls `manager.ShouldTerminateAsync()` to check termination
3. Calls `manager.UpdateHistoryAsync()` for history filtering
4. Calls `manager.SelectNextAgentAsync()` to get next speaker
5. Routes messages to selected agent executor
6. Yields output when terminated

---

## 3. Python Implementation

### 3.1 Core Files

| Component | File Path | Lines |
|-----------|-----------|-------|
| **BaseGroupChatOrchestrator** | [python/packages/core/agent_framework/_workflows/_base_group_chat_orchestrator.py](python/packages/core/agent_framework/_workflows/_base_group_chat_orchestrator.py) | 1-591 |
| **GroupChatOrchestrator** | [python/packages/core/agent_framework/_workflows/_group_chat.py](python/packages/core/agent_framework/_workflows/_group_chat.py#L83) | L83-250 |
| **AgentBasedGroupChatOrchestrator** | [python/packages/core/agent_framework/_workflows/_group_chat.py](python/packages/core/agent_framework/_workflows/_group_chat.py#L260) | L260-470 |
| **GroupChatBuilder** | [python/packages/core/agent_framework/_workflows/_group_chat.py](python/packages/core/agent_framework/_workflows/_group_chat.py#L476) | L476-996 |
| **MagenticOrchestrator** | [python/packages/core/agent_framework/_workflows/_magentic.py](python/packages/core/agent_framework/_workflows/_magentic.py) | 1-1983 |
| **ParticipantRegistry** | [python/packages/core/agent_framework/_workflows/_base_group_chat_orchestrator.py](python/packages/core/agent_framework/_workflows/_base_group_chat_orchestrator.py#L109) | L109-145 |

### 3.2 BaseGroupChatOrchestrator (Abstract Base)

```python
class BaseGroupChatOrchestrator(Executor, ABC):
    TERMINATION_CONDITION_MET_MESSAGE: ClassVar[str]
    MAX_ROUNDS_MET_MESSAGE: ClassVar[str]

    def __init__(
        self,
        id: str,
        participant_registry: ParticipantRegistry,
        *,
        name: str | None = None,
        max_rounds: int | None = None,
        termination_condition: TerminationCondition | None = None,
    ): ...

    # Abstract methods for subclasses
    async def _handle_messages(self, messages, ctx) -> None: ...
    async def _handle_response(self, response, ctx) -> None: ...

    # Conversation management (shared)
    def _append_messages(self, messages): ...
    def _get_conversation(self) -> list[ChatMessage]: ...
    async def _check_termination(self) -> bool: ...
    async def _check_round_limit_and_yield(self, ctx) -> bool: ...

    # Participant routing (shared)
    async def _broadcast_messages_to_participants(self, messages, ctx, participants=None): ...
    async def _send_request_to_participant(self, target, ctx, *, additional_instruction=None): ...

    # Checkpointing (shared)
    async def on_checkpoint_save(self) -> dict[str, Any]: ...
    async def on_checkpoint_restore(self, state: dict[str, Any]) -> None: ...
```

### 3.3 GroupChatSelectionFunction Type

```python
GroupChatSelectionFunction = Callable[[GroupChatState], Awaitable[str] | str]

@dataclass(frozen=True)
class GroupChatState:
    current_round: int
    participants: OrderedDict[str, str]  # name -> description
    conversation: list[ChatMessage]
```

### 3.4 GroupChatOrchestrator (Function-Based Selection)

```python
class GroupChatOrchestrator(BaseGroupChatOrchestrator):
    def __init__(
        self,
        id: str,
        participant_registry: ParticipantRegistry,
        selection_func: GroupChatSelectionFunction,
        *,
        name: str | None = None,
        max_rounds: int | None = None,
        termination_condition: TerminationCondition | None = None,
    ): ...
```

**Selection Pattern:** User-provided function receives `GroupChatState` and returns participant name.

### 3.5 AgentBasedGroupChatOrchestrator (LLM-Driven)

```python
class AgentOrchestrationOutput(BaseModel):
    terminate: bool
    reason: str
    next_speaker: str | None
    final_message: str | None

class AgentBasedGroupChatOrchestrator(BaseGroupChatOrchestrator):
    def __init__(
        self,
        agent: ChatAgent,
        participant_registry: ParticipantRegistry,
        *,
        max_rounds: int | None = None,
        termination_condition: TerminationCondition | None = None,
        retry_attempts: int | None = None,
        thread: AgentThread | None = None,
    ): ...
```

**Selection Pattern:** Uses structured output from LLM to decide next speaker and termination.

### 3.6 GroupChatBuilder API

```python
workflow = (
    GroupChatBuilder()
    # Option 1: Function-based selector
    .with_orchestrator(selection_func=round_robin_selector, orchestrator_name="Coordinator")
    # Option 2: Agent-based selector
    .with_orchestrator(agent=orchestrator_agent)
    # Option 3: Custom orchestrator
    .with_orchestrator(orchestrator=custom_orchestrator)

    .participants([agent1, agent2, agent3])
    .with_termination_condition(lambda conv: len(conv) >= 10)
    .with_max_rounds(20)
    .with_checkpointing(storage)
    .with_request_info(agents=["agent1"])  # Human-in-the-loop
    .build()
)
```

---

## 4. API Surface Comparison

| Feature | .NET | Python |
|---------|------|--------|
| **Base Class** | `GroupChatManager` (abstract) | `BaseGroupChatOrchestrator` (abstract, inherits `Executor`) |
| **Selection Method** | `SelectNextAgentAsync(history)` returns `AIAgent` | `GroupChatSelectionFunction(state)` returns `str` (participant name) |
| **Built-in Round Robin** | `RoundRobinGroupChatManager` | Via simple lambda function |
| **LLM-Based Selection** | Custom implementation required | `AgentBasedGroupChatOrchestrator` built-in |
| **Builder Entry Point** | `AgentWorkflowBuilder.CreateGroupChatBuilderWith()` | `GroupChatBuilder()` |
| **Participant Registration** | `.AddParticipants(...)` | `.participants(...)` or `.register_participants(...)` |
| **Max Iterations** | `MaximumIterationCount` property on manager | `.with_max_rounds()` on builder |
| **Termination Condition** | `ShouldTerminateAsync()` override or delegate | `.with_termination_condition()` on builder |
| **History Filtering** | `UpdateHistoryAsync()` override | Not exposed (handled internally) |
| **Checkpointing** | Via workflow infrastructure | `.with_checkpointing(storage)` on builder |
| **Human-in-the-Loop** | Via tool approval | `.with_request_info()` on builder |
| **Streaming Events** | `WorkflowEvent` hierarchy | `WorkflowEvent` hierarchy with `GroupChatRequestSentEvent`, `GroupChatResponseReceivedEvent` |
| **Advanced Patterns** | - | `MagenticOrchestrator` (sophisticated multi-step planning) |

---

## 5. Selector Patterns

### 5.1 Round-Robin Selection

**.NET:**
```csharp
new RoundRobinGroupChatManager(agents) { MaximumIterationCount = 10 }
```

**Python:**
```python
def round_robin_selector(state: GroupChatState) -> str:
    names = list(state.participants.keys())
    return names[state.current_round % len(names)]
```

### 5.2 Custom Logic Selection

**.NET (Custom Manager):**
```csharp
internal sealed class DeploymentGroupChatManager : GroupChatManager
{
    protected override ValueTask<AIAgent> SelectNextAgentAsync(IReadOnlyList<ChatMessage> history, ...)
    {
        if (this.IterationCount == 0)
            return new ValueTask<AIAgent>(_agents.First(a => a.Name == "QAEngineer"));
        return new ValueTask<AIAgent>(_agents.First(a => a.Name == "DevOpsEngineer"));
    }
}
```

**Python (Function):**
```python
def custom_selector(state: GroupChatState) -> str:
    if "error" in state.conversation[-1].text.lower():
        return "debugger"
    return "developer"
```

### 5.3 LLM-Based Selection

**Python (Built-in):**
```python
orchestrator_agent = ChatAgent(
    name="Orchestrator",
    instructions="Coordinate team conversation...",
    chat_client=chat_client,
)

workflow = GroupChatBuilder().with_orchestrator(agent=orchestrator_agent).participants([...]).build()
```

**.NET:** Requires custom `GroupChatManager` implementation that invokes an LLM.

---

## 6. Termination Condition Handling

### 6.1 .NET

```csharp
// Option 1: MaximumIterationCount
new RoundRobinGroupChatManager(agents) { MaximumIterationCount = 10 }

// Option 2: Delegate on RoundRobinGroupChatManager
new RoundRobinGroupChatManager(agents, shouldTerminateFunc: (manager, history, ct) =>
    new ValueTask<bool>(history.Any(m => m.Text.Contains("DONE"))));

// Option 3: Override in custom manager
protected override ValueTask<bool> ShouldTerminateAsync(IReadOnlyList<ChatMessage> history, ...)
{
    return new ValueTask<bool>(history.Last().Text.Contains("COMPLETE"));
}
```

### 6.2 Python

```python
# Option 1: max_rounds
.with_max_rounds(10)

# Option 2: Lambda/function
.with_termination_condition(lambda conv: len(conv) >= 20)

# Option 3: Async function
async def check_done(conversation: list[ChatMessage]) -> bool:
    return any("DONE" in m.text for m in conversation)
.with_termination_condition(check_done)

# Option 4: Agent-based orchestrator decides (built into AgentOrchestrationOutput.terminate)
```

---

## 7. Transcript/Conversation Tracking

### 7.1 .NET

- Conversation maintained in `GroupChatHost._pendingMessages`
- Messages accumulated and passed to manager for selection
- `UpdateHistoryAsync()` allows filtering before agent receives messages
- Final output yielded as `List<ChatMessage>`

### 7.2 Python

- Conversation maintained in `BaseGroupChatOrchestrator._full_conversation`
- Methods: `_append_messages()`, `_get_conversation()`, `_clear_conversation()`
- Checkpointing via `on_checkpoint_save()`/`on_checkpoint_restore()`
- Messages broadcast to all participants via `_broadcast_messages_to_participants()`
- Final output: `list[ChatMessage]`

---

## 8. Streaming Events

### 8.1 .NET Events

```csharp
// Via workflow event stream
await foreach (WorkflowEvent evt in run.WatchStreamAsync())
{
    if (evt is ExecutorCompletedEvent completed) { ... }
    if (evt is AgentResponseUpdateEvent update) { ... }
}
```

### 8.2 Python Events

```python
class GroupChatRequestSentEvent(GroupChatEvent):
    round_index: int
    participant_name: str

class GroupChatResponseReceivedEvent(GroupChatEvent):
    round_index: int
    participant_name: str

# Usage
async for event in workflow.run_stream(task):
    if isinstance(event, AgentRunUpdateEvent):
        print(f"{event.executor_id}: {event.data}")
    elif isinstance(event, WorkflowOutputEvent):
        print("Final output:", event.output)
```

---

## 9. Design Recommendations for Go Implementation

### 9.1 Core Types

```go
// Selector interface (like Python's function type)
type Selector interface {
    SelectNext(ctx context.Context, state GroupChatState) (participantID string, err error)
}

// GroupChatState passed to selector
type GroupChatState struct {
    CurrentRound int
    Participants map[string]string // ID -> description
    Conversation []ChatMessage
}

// TerminationCondition function type
type TerminationCondition func(conversation []ChatMessage) bool
```

### 9.2 Built-in Selectors

```go
// RoundRobinSelector
type RoundRobinSelector struct {
    participants []string
    index        int
}

func (s *RoundRobinSelector) SelectNext(ctx context.Context, state GroupChatState) (string, error) {
    participant := s.participants[s.index % len(s.participants)]
    s.index++
    return participant, nil
}

// RandomSelector
type RandomSelector struct {
    participants []string
    rng          *rand.Rand
}

// LLMSelector (future)
type LLMSelector struct {
    agent Agent
}
```

### 9.3 GroupChatManager

```go
type GroupChatManager struct {
    selector              Selector
    maxIterations         int
    terminationCondition  TerminationCondition
    iterationCount        int
    conversation          []ChatMessage
}

type GroupChatManagerOption func(*GroupChatManager)

func WithSelector(s Selector) GroupChatManagerOption { ... }
func WithMaxIterations(n int) GroupChatManagerOption { ... }
func WithTerminationCondition(tc TerminationCondition) GroupChatManagerOption { ... }

func NewGroupChatManager(participants []Agent, opts ...GroupChatManagerOption) *GroupChatManager { ... }

func (m *GroupChatManager) Run(ctx context.Context, initialMessages []ChatMessage) (*GroupChatResult, error) { ... }
func (m *GroupChatManager) RunStreaming(ctx context.Context, initialMessages []ChatMessage) (<-chan GroupChatEvent, error) { ... }
```

### 9.4 Key Design Principles

1. **Interface-first:** Define `Selector` interface for extensibility
2. **Functional options:** Use option pattern for configuration
3. **Context-aware:** All methods accept `context.Context`
4. **Streaming support:** Return channel for events
5. **Minimal API:** Start with essentials, expand later
6. **Match Python patterns:** Python is more complete, follow its lead
7. **State serialization:** Support checkpointing from start

### 9.5 Suggested Package Structure

```
go/
└── workflow/
    └── groupchat/
        ├── manager.go         # GroupChatManager
        ├── selector.go        # Selector interface
        ├── roundrobin.go      # RoundRobinSelector
        ├── random.go          # RandomSelector
        ├── state.go           # GroupChatState, GroupChatResult
        ├── events.go          # GroupChatEvent types
        └── options.go         # Functional options
```

---

## 10. Sample Code References

| Sample | .NET | Python |
|--------|------|--------|
| Round-robin selection | [dotnet/tests/.../06_GroupChat_Workflow.cs](dotnet/tests/Microsoft.Agents.AI.Workflows.UnitTests/Sample/06_GroupChat_Workflow.cs) | [python/samples/.../group_chat_simple_selector.py](python/samples/getting_started/workflows/orchestration/group_chat_simple_selector.py) |
| Custom selector | [dotnet/samples/.../DeploymentGroupChatManager.cs](dotnet/samples/GettingStarted/Workflows/Agents/GroupChatToolApproval/DeploymentGroupChatManager.cs) | Same file (lambda function) |
| Agent-based manager | - | [python/samples/.../group_chat_agent_manager.py](python/samples/getting_started/workflows/orchestration/group_chat_agent_manager.py) |
| Tool approval | [dotnet/samples/.../GroupChatToolApproval/](dotnet/samples/GettingStarted/Workflows/Agents/GroupChatToolApproval/) | [python/samples/.../group_chat_builder_tool_approval.py](python/samples/getting_started/workflows/tool-approval/group_chat_builder_tool_approval.py) |

---

## 11. Summary

The Python implementation is more mature with:

- Multiple orchestrator types (function-based, agent-based, Magentic)
- Fluent builder with factory support
- Built-in checkpointing
- Human-in-the-loop via `with_request_info()`

The .NET implementation is simpler with:

- Single abstract `GroupChatManager` class
- Built-in `RoundRobinGroupChatManager`
- Factory-based builder pattern

**For Go port:** Follow Python's architecture as the reference design, implementing the `Selector` interface pattern for extensibility.
