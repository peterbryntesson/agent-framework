# Python DurableTask Implementation Research

**Date:** 2026-02-04  
**Purpose:** Analyze Python durabletask implementation for durable agents to inform Go Temporal.io port  
**Status:** Complete

---

## Table of Contents

1. [Package Structure Overview](#1-package-structure-overview)
2. [Core Abstractions](#2-core-abstractions)
3. [State Persistence Pattern](#3-state-persistence-pattern)
4. [Execution Models](#4-execution-models)
5. [Azure Functions Integration](#5-azure-functions-integration)
6. [Key APIs](#6-key-apis)
7. [Temporal.io Mapping Suggestions](#7-temporalio-mapping-suggestions)
8. [Clarifying Questions](#8-clarifying-questions)

---

## 1. Package Structure Overview

### durabletask Package

**Location:** [python/packages/durabletask/agent_framework_durabletask/](python/packages/durabletask/agent_framework_durabletask/)

| File | Purpose |
|------|---------|
| [__init__.py](__init__.py) | Public API exports |
| [_entities.py](_entities.py#L1-L348) | `AgentEntity`, `AgentEntityStateProviderMixin`, `DurableTaskEntityStateProvider` |
| [_durable_agent_state.py](_durable_agent_state.py#L1-L1327) | State serialization classes conforming to JSON schema |
| [_worker.py](_worker.py#L1-L200) | `DurableAIAgentWorker` - worker wrapper for agent registration |
| [_client.py](_client.py#L1-L85) | `DurableAIAgentClient` - external client for agent interaction |
| [_executors.py](_executors.py#L1-L517) | Execution strategies: `ClientAgentExecutor`, `OrchestrationAgentExecutor` |
| [_orchestration_context.py](_orchestration_context.py#L1-L75) | `DurableAIAgentOrchestrationContext` for orchestration usage |
| [_shim.py](_shim.py#L1-L200) | `DurableAIAgent` proxy implementing `AgentProtocol` |
| [_models.py](_models.py#L1-L341) | `RunRequest`, `AgentSessionId`, `DurableAgentThread` |
| [_callbacks.py](_callbacks.py#L1-L40) | Callback protocols for streaming/response handling |
| [_constants.py](_constants.py) | Constants and field names |
| [_response_utils.py](_response_utils.py) | Response format utilities |

### azurefunctions Package

**Location:** [python/packages/azurefunctions/agent_framework_azurefunctions/](python/packages/azurefunctions/agent_framework_azurefunctions/)

| File | Purpose |
|------|---------|
| [_app.py](_app.py#L1-L1051) | `AgentFunctionApp` - main Azure Functions integration |
| [_entities.py](_entities.py#L1-L120) | Azure Functions entity factory using `azure.durable_functions` |
| [_orchestration.py](_orchestration.py#L1-L222) | `AzureFunctionsAgentExecutor`, `AgentTask` |
| [_errors.py](_errors.py) | Error handling |

---

## 2. Core Abstractions

### 2.1 AgentEntity (Core Execution Logic)

**File:** [_entities.py#L78-L348](python/packages/durabletask/agent_framework_durabletask/_entities.py#L78-L348)

Platform-agnostic agent execution logic that encapsulates:
- Running agent with message
- Managing conversation state
- Handling streaming responses
- Invoking callbacks

```python
class AgentEntity:
    """Platform-agnostic agent execution logic."""

    agent: AgentProtocol
    callback: AgentResponseCallbackProtocol | None

    def __init__(
        self,
        agent: AgentProtocol,
        callback: AgentResponseCallbackProtocol | None = None,
        *,
        state_provider: AgentEntityStateProviderMixin,
    ) -> None:
        self.agent = agent
        self.callback = callback
        self._state_provider = state_provider

    async def run(self, request: RunRequest | dict[str, Any] | str) -> AgentResponse:
        """Execute the agent with a message."""
        # 1. Parse request
        # 2. Append request to conversation history
        # 3. Build chat messages from non-error history
        # 4. Invoke agent (streaming preferred)
        # 5. Append response to history
        # 6. Persist state
        # 7. Return response
```

### 2.2 AgentEntityStateProviderMixin

**File:** [_entities.py#L33-L76](python/packages/durabletask/agent_framework_durabletask/_entities.py#L33-L76)

Abstract mixin for state management - concrete implementations provide storage backend:

```python
class AgentEntityStateProviderMixin:
    """Mixin implementing durable agent state caching + (de)serialization + persistence."""

    _state_cache: DurableAgentState | None = None

    # Abstract methods - must be implemented by subclasses
    def _get_state_dict(self) -> dict[str, Any]: ...
    def _set_state_dict(self, state: dict[str, Any]) -> None: ...
    def _get_thread_id_from_entity(self) -> str: ...

    @property
    def thread_id(self) -> str:
        return self._get_thread_id_from_entity()

    @property
    def state(self) -> DurableAgentState:
        if self._state_cache is None:
            raw_state = self._get_state_dict()
            self._state_cache = DurableAgentState.from_dict(raw_state) if raw_state else DurableAgentState()
        return self._state_cache

    def persist_state(self) -> None:
        """Persist the current state to the underlying storage provider."""
        if self._state_cache is None:
            self._state_cache = DurableAgentState()
        self._set_state_dict(self._state_cache.to_dict())

    def reset(self) -> None:
        """Clear conversation history by resetting state to a fresh DurableAgentState."""
        self._state_cache = DurableAgentState()
        self.persist_state()
```

### 2.3 DurableAIAgent (Agent Proxy/Shim)

**File:** [_shim.py#L43-L165](python/packages/durabletask/agent_framework_durabletask/_shim.py#L43-L165)

Implements `AgentProtocol` but with different return semantics:

```python
class DurableAIAgent(AgentProtocol, Generic[TaskT]):
    """A durable agent proxy that delegates execution to the provider."""

    def __init__(self, executor: DurableAgentExecutor[TaskT], name: str, *, agent_id: str | None = None):
        self._executor = executor
        self.name = name
        self.id = agent_id if agent_id is not None else name

    def run(
        self,
        messages: str | ChatMessage | list[str] | list[ChatMessage] | None = None,
        *,
        thread: AgentThread | None = None,
        options: dict[str, Any] | None = None,
    ) -> TaskT:
        """Execute the agent via the injected provider.
        
        NOTE: Returns TaskT (sync Task object for yielding), NOT Coroutine like AgentProtocol.
        """
        message_str = self._normalize_messages(messages)
        run_request = self._executor.get_run_request(message=message_str, options=options)
        return self._executor.run_durable_agent(
            agent_name=self.name,
            run_request=run_request,
            thread=thread,
        )

    def get_new_thread(self, **kwargs: Any) -> DurableAgentThread:
        """Create a new agent thread via the provider."""
        return self._executor.get_new_thread(self.name, **kwargs)
```

### 2.4 DurableAgentExecutor (Execution Strategy Interface)

**File:** [_executors.py#L94-L177](python/packages/durabletask/agent_framework_durabletask/_executors.py#L94-L177)

Abstract base class for execution strategies:

```python
class DurableAgentExecutor(ABC, Generic[TaskT]):
    """Abstract base class for durable agent execution strategies."""

    @abstractmethod
    def run_durable_agent(
        self,
        agent_name: str,
        run_request: RunRequest,
        thread: AgentThread | None = None,
    ) -> TaskT:
        """Execute the durable agent."""
        raise NotImplementedError

    def get_new_thread(self, agent_name: str, **kwargs: Any) -> DurableAgentThread:
        """Create a new DurableAgentThread with random session ID."""
        session_id = self._create_session_id(agent_name)
        return DurableAgentThread.from_session_id(session_id, **kwargs)

    def get_run_request(
        self,
        message: str,
        *,
        options: dict[str, Any] | None = None,
    ) -> RunRequest:
        """Create a RunRequest from message and options."""
        correlation_id = self.generate_unique_id()
        # Extract response_format, enable_tool_calls, wait_for_response from options
        return RunRequest(...)
```

---

## 3. State Persistence Pattern

### 3.1 State Schema

**Schema File:** [schemas/durable-agent-entity-state.json](schemas/durable-agent-entity-state.json)

**Schema Version:** `1.1.0`

```json
{
  "schemaVersion": "1.1.0",
  "data": {
    "conversationHistory": [
      {
        "$type": "request",
        "correlationId": "uuid",
        "createdAt": "2026-02-04T12:00:00Z",
        "messages": [
          {
            "role": "user",
            "contents": [
              { "$type": "text", "text": "Hello" }
            ]
          }
        ],
        "responseType": "text",
        "orchestrationId": "optional-orchestration-id"
      },
      {
        "$type": "response",
        "correlationId": "uuid",
        "createdAt": "2026-02-04T12:00:01Z",
        "messages": [
          {
            "role": "assistant",
            "contents": [
              { "$type": "text", "text": "Hello! How can I help?" }
            ]
          }
        ],
        "usage": {
          "inputTokenCount": 10,
          "outputTokenCount": 8,
          "totalTokenCount": 18
        }
      }
    ]
  }
}
```

### 3.2 DurableAgentState Class

**File:** [_durable_agent_state.py#L383-L446](python/packages/durabletask/agent_framework_durabletask/_durable_agent_state.py#L383-L446)

```python
class DurableAgentState:
    """Manages durable agent state conforming to the schema."""

    SCHEMA_VERSION: str = "1.1.0"
    data: DurableAgentStateData
    schema_version: str = SCHEMA_VERSION

    def __init__(self, schema_version: str = SCHEMA_VERSION):
        self.data = DurableAgentStateData()
        self.schema_version = schema_version

    def to_dict(self) -> dict[str, Any]:
        return {
            "schemaVersion": self.schema_version,
            "data": self.data.to_dict(),
        }

    @classmethod
    def from_dict(cls, state: dict[str, Any]) -> DurableAgentState:
        """Restore state from a dictionary."""
        schema_version = state.get("schemaVersion")
        if schema_version is None:
            logger.warning("Resetting state as it is incompatible")
            return cls()
        instance = cls(schema_version=state.get("schemaVersion", cls.SCHEMA_VERSION))
        instance.data = DurableAgentStateData.from_dict(state.get("data", {}))
        return instance

    def try_get_agent_response(self, correlation_id: str) -> AgentResponse | None:
        """Get response by correlation ID for polling."""
        for entry in self.data.conversation_history:
            if entry.correlation_id == correlation_id and isinstance(entry, DurableAgentStateResponse):
                return DurableAgentStateResponse.to_run_response(entry)
        return None
```

### 3.3 Content Type Hierarchy

**File:** [_durable_agent_state.py#L225-L310](python/packages/durabletask/agent_framework_durabletask/_durable_agent_state.py#L225-L310)

| Content Type | Class | Purpose |
|--------------|-------|---------|
| `text` | `DurableAgentStateTextContent` | Plain text content |
| `data` | `DurableAgentStateDataContent` | Data URIs with media type |
| `error` | `DurableAgentStateErrorContent` | Error information |
| `functionCall` | `DurableAgentStateFunctionCallContent` | Tool/function calls |
| `functionResult` | `DurableAgentStateFunctionResultContent` | Tool/function results |
| `hostedFile` | `DurableAgentStateHostedFileContent` | Hosted file references |
| `hostedVectorStore` | `DurableAgentStateHostedVectorStoreContent` | Vector store references |
| `reasoning` | `DurableAgentStateTextReasoningContent` | Reasoning/chain-of-thought |
| `uri` | `DurableAgentStateUriContent` | URI references |
| `usage` | `DurableAgentStateUsageContent` | Token usage statistics |
| `unknown` | `DurableAgentStateUnknownContent` | Unknown content passthrough |

---

## 4. Execution Models

### 4.1 Client Execution (External)

**File:** [_executors.py#L180-L390](python/packages/durabletask/agent_framework_durabletask/_executors.py#L180-L390)

For external clients using `TaskHubGrpcClient`:

```python
class ClientAgentExecutor(DurableAgentExecutor[AgentResponse]):
    """Execution strategy for external clients."""

    def run_durable_agent(
        self,
        agent_name: str,
        run_request: RunRequest,
        thread: AgentThread | None = None,
    ) -> AgentResponse:
        """Execute the agent via the durabletask client.
        
        1. Signal the agent entity with a message request
        2. If wait_for_response=False, return acceptance response immediately
        3. Otherwise, poll the entity state for the response
        """
        entity_id = self._signal_agent_entity(agent_name, run_request, thread)

        if not run_request.wait_for_response:
            return self._create_acceptance_response(run_request.correlation_id)

        agent_response = self._poll_for_agent_response(entity_id, run_request.correlation_id)
        return self._handle_agent_response(agent_response, run_request.response_format, run_request.correlation_id)

    def _poll_for_agent_response(self, entity_id, correlation_id) -> AgentResponse | None:
        """Poll with retries until response is available."""
        for attempt in range(1, self.max_poll_retries + 1):
            time.sleep(self.poll_interval_seconds)  # Sleep first
            response = self._poll_entity_for_response(entity_id, correlation_id)
            if response is not None:
                return response
        return None

    def _poll_entity_for_response(self, entity_id, correlation_id) -> AgentResponse | None:
        """Read entity state and extract response by correlation ID."""
        entity_metadata = self._client.get_entity(entity_id, include_state=True)
        if entity_metadata is None:
            return None
        state = DurableAgentState.from_json(entity_metadata.get_state())
        return state.try_get_agent_response(correlation_id)
```

### 4.2 Orchestration Execution (Internal)

**File:** [_executors.py#L393-L517](python/packages/durabletask/agent_framework_durabletask/_executors.py#L393-L517)

For use within orchestration functions:

```python
class OrchestrationAgentExecutor(DurableAgentExecutor[DurableAgentTask]):
    """Execution strategy for orchestrations (sync/yield)."""

    def __init__(self, context: OrchestrationContext):
        self._context = context

    def generate_unique_id(self) -> str:
        """Create replay-safe UUID."""
        return self._context.new_uuid()

    def run_durable_agent(
        self,
        agent_name: str,
        run_request: RunRequest,
        thread: AgentThread | None = None,
    ) -> DurableAgentTask:
        """Execute the agent via orchestration context.
        
        Returns a DurableAgentTask that must be YIELDED (not awaited).
        """
        entity_id = EntityInstanceId(entity=session_id.entity_name, key=session_id.key)

        if not run_request.wait_for_response:
            # Fire-and-forget: signal entity
            self._context.signal_entity(entity_id, "run", run_request.to_dict())
            entity_task = CompletableTask()
            entity_task.complete(self._create_acceptance_response(run_request.correlation_id))
        else:
            # Blocking: call entity and wait
            entity_task = self._context.call_entity(entity_id, "run", run_request.to_dict())

        return DurableAgentTask(entity_task, run_request.response_format, run_request.correlation_id)
```

### 4.3 DurableAgentTask (Composite Task)

**File:** [_executors.py#L36-L92](python/packages/durabletask/agent_framework_durabletask/_executors.py#L36-L92)

Wraps entity calls and provides typed `AgentResponse` results:

```python
class DurableAgentTask(CompositeTask[AgentResponse], CompletableTask[AgentResponse]):
    """Custom Task that wraps entity calls for typed results."""

    def __init__(
        self,
        entity_task: CompletableTask[Any],
        response_format: type[BaseModel] | None,
        correlation_id: str,
    ):
        self._response_format = response_format
        self._correlation_id = correlation_id
        super().__init__([entity_task])

    def on_child_completed(self, task: Task[Any]) -> None:
        """Handle completion of the underlying entity task."""
        if self.is_complete:
            return

        if task.is_failed:
            self.fail("call_entity Task failed", task.get_exception())
            return

        raw_result = task.get_result()
        try:
            response = load_agent_response(raw_result)
            if self._response_format is not None:
                ensure_response_format(self._response_format, self._correlation_id, response)
            self.complete(response)
        except Exception as ex:
            self.fail("Failed to convert result", ex)
```

---

## 5. Azure Functions Integration

### 5.1 AgentFunctionApp

**File:** [_app.py#L95-L300](python/packages/azurefunctions/agent_framework_azurefunctions/_app.py#L95-L300)

Main application class extending `df.DFApp`:

```python
class AgentFunctionApp(DFAppBase):
    """Main application class for durable agent function apps using Durable Entities."""

    def __init__(
        self,
        agents: list[AgentProtocol] | None = None,
        http_auth_level: func.AuthLevel = func.AuthLevel.FUNCTION,
        enable_health_check: bool = True,
        enable_http_endpoints: bool = True,
        max_poll_retries: int = DEFAULT_MAX_POLL_RETRIES,
        poll_interval_seconds: float = DEFAULT_POLL_INTERVAL_SECONDS,
        enable_mcp_tool_trigger: bool = False,
        default_callback: AgentResponseCallbackProtocol | None = None,
    ):
        super().__init__(http_auth_level=http_auth_level)
        # Register agents and setup functions

    def add_agent(
        self,
        agent: AgentProtocol,
        callback: AgentResponseCallbackProtocol | None = None,
        enable_http_endpoint: bool | None = None,
        enable_mcp_tool_trigger: bool | None = None,
    ) -> None:
        """Add an agent - creates HTTP endpoint, entity, and optionally MCP trigger."""
        # Creates:
        # - HTTP POST /api/agents/{name}/run
        # - Entity: dafx-{name}
        # - Optional MCP tool trigger

    def get_agent(
        self,
        context: AgentOrchestrationContextType,
        agent_name: str,
    ) -> DurableAIAgent[AgentTask]:
        """Return a DurableAIAgent proxy for orchestration use."""
        executor = AzureFunctionsAgentExecutor(context)
        return DurableAIAgent(executor, agent_name)
```

### 5.2 Azure Functions Entity Factory

**File:** [_entities.py#L50-L120](python/packages/azurefunctions/agent_framework_azurefunctions/_entities.py#L50-L120)

```python
class AzureFunctionEntityStateProvider(AgentEntityStateProviderMixin):
    """Azure Functions Durable Entity state provider."""

    def __init__(self, context: df.DurableEntityContext) -> None:
        self._context = context

    def _get_state_dict(self) -> dict[str, Any]:
        return self._context.get_state(lambda: {})

    def _set_state_dict(self, state: dict[str, Any]) -> None:
        self._context.set_state(state)

    def _get_thread_id_from_entity(self) -> str:
        return str(self._context.entity_key)


def create_agent_entity(
    agent: AgentProtocol,
    callback: AgentResponseCallbackProtocol | None = None,
) -> Callable[[df.DurableEntityContext], None]:
    """Factory function to create an agent entity."""

    async def _entity_coroutine(context: df.DurableEntityContext) -> None:
        state_provider = AzureFunctionEntityStateProvider(context)
        entity = AgentEntity(agent, callback, state_provider=state_provider)

        if context.operation_name in ("run", "run_agent"):
            result = await entity.run(context.get_input())
            context.set_result(result.to_dict())
        elif context.operation_name == "reset":
            entity.reset()
            context.set_result({"status": "reset"})

    def entity_function(context: df.DurableEntityContext) -> None:
        """Synchronous wrapper for Durable Functions runtime."""
        asyncio.run(_entity_coroutine(context))

    return entity_function
```

---

## 6. Key APIs

### 6.1 Worker Setup Pattern

**File:** [_worker.py#L23-L120](python/packages/durabletask/agent_framework_durabletask/_worker.py#L23-L120)

```python
from durabletask import TaskHubGrpcWorker
from agent_framework_durabletask import DurableAIAgentWorker

# Create underlying worker
worker = TaskHubGrpcWorker(host_address="localhost:4001")

# Wrap with agent worker
agent_worker = DurableAIAgentWorker(worker)

# Register agents
agent_worker.add_agent(my_agent)

# Start worker
worker.start()
```

### 6.2 Client Usage Pattern

**File:** [_client.py#L23-L85](python/packages/durabletask/agent_framework_durabletask/_client.py#L23-L85)

```python
from durabletask import TaskHubGrpcClient
from agent_framework_durabletask import DurableAIAgentClient

# Create client
client = TaskHubGrpcClient(host_address="localhost:4001")
agent_client = DurableAIAgentClient(client)

# Get agent proxy
agent = agent_client.get_agent("assistant")

# Run agent (blocking - polls for response)
response = agent.run("Hello, how are you?")

# Fire-and-forget mode
response = agent.run("Process this", options={"wait_for_response": False})
```

### 6.3 Orchestration Usage Pattern

**File:** [_orchestration_context.py#L18-L75](python/packages/durabletask/agent_framework_durabletask/_orchestration_context.py#L18-L75)

```python
from agent_framework_durabletask import DurableAIAgentOrchestrationContext

def my_orchestration(context: OrchestrationContext):
    # Wrap context
    agent_context = DurableAIAgentOrchestrationContext(context)
    
    # Get agent proxy
    agent = agent_context.get_agent("assistant")
    
    # Run agent - YIELD (not await)
    result = yield agent.run("Hello!")
    
    return result.text
```

### 6.4 Session/Thread Management

**File:** [_models.py#L219-L280](python/packages/durabletask/agent_framework_durabletask/_models.py#L219-L280)

```python
# AgentSessionId: Identifies an agent session (name + key)
session_id = AgentSessionId(name="assistant", key=uuid.uuid4().hex)
# Entity name format: "dafx-{agent_name}"
entity_name = session_id.entity_name  # "dafx-assistant"

# DurableAgentThread: Tracks session ID for conversation continuity
thread = DurableAgentThread(session_id=session_id)

# Can parse from string format "@name@key"
parsed = AgentSessionId.parse("@assistant@abc123")
```

### 6.5 Callback Protocol

**File:** [_callbacks.py#L7-L40](python/packages/durabletask/agent_framework_durabletask/_callbacks.py#L7-L40)

```python
class AgentResponseCallbackProtocol(Protocol):
    """Protocol for callbacks during agent execution."""

    async def on_streaming_response_update(
        self,
        update: AgentResponseUpdate,
        context: AgentCallbackContext,
    ) -> None:
        """Handle streaming response update."""

    async def on_agent_response(
        self,
        response: AgentResponse,
        context: AgentCallbackContext,
    ) -> None:
        """Handle final agent response."""

@dataclass(frozen=True)
class AgentCallbackContext:
    """Context for callback invocations."""
    agent_name: str
    correlation_id: str
    thread_id: str | None = None
    request_message: str | None = None
```

---

## 7. Temporal.io Mapping Suggestions

### 7.1 Concept Mappings

| Python DurableTask | Temporal.io Equivalent | Notes |
|-------------------|------------------------|-------|
| `DurableEntity` | Temporal Entity (experimental) or Workflow + Signal | Temporal Entities are still experimental; alternatively use Workflow with signals |
| `TaskHubGrpcWorker` | `temporal.Worker` | Temporal's worker for executing workflows/activities |
| `TaskHubGrpcClient` | `temporal.Client` | Client for starting workflows and sending signals |
| `OrchestrationContext` | `workflow.Context` | Workflow execution context |
| `call_entity` | `workflow.ExecuteActivity` or Entity call | Direct entity calls in Temporal |
| `signal_entity` | `workflow.SignalExternalWorkflow` or Entity signal | For fire-and-forget operations |
| `EntityInstanceId` | Entity ID or Workflow ID | Unique identifier for entity instances |
| `CompletableTask` / `CompositeTask` | Temporal's `Future` | Temporal uses Futures for async operations |

### 7.2 State Management Patterns

```go
// Temporal Entity (if using experimental entity feature)
type AgentEntityState struct {
    SchemaVersion       string                    `json:"schemaVersion"`
    Data                AgentStateData            `json:"data"`
}

type AgentStateData struct {
    ConversationHistory []ConversationEntry       `json:"conversationHistory"`
    ExpirationTimeUTC   *time.Time                `json:"expirationTimeUtc,omitempty"`
}

// Alternative: Workflow-based state with Continue-As-New for long conversations
type AgentWorkflow struct {
    state AgentEntityState
}

func (w *AgentWorkflow) Run(ctx workflow.Context) error {
    // Handle signals for requests
    // Persist state periodically
    // Use Continue-As-New when history gets too long
}
```

### 7.3 Execution Model Mapping

```go
// Worker setup (similar to DurableAIAgentWorker)
type DurableAgentWorker struct {
    worker          client.Worker
    registeredAgents map[string]AgentProtocol
}

func (w *DurableAgentWorker) AddAgent(agent AgentProtocol) error {
    // Register workflow/entity for agent
    w.worker.RegisterWorkflow(createAgentWorkflow(agent))
    w.registeredAgents[agent.Name()] = agent
    return nil
}

// Client usage (similar to DurableAIAgentClient)
type DurableAgentClient struct {
    client temporal.Client
}

func (c *DurableAgentClient) GetAgent(name string) *DurableAIAgent {
    return &DurableAIAgent{
        executor: &ClientAgentExecutor{client: c.client},
        name:     name,
    }
}
```

### 7.4 Long-Running Operations

For human-in-the-loop patterns:

```go
// Temporal Activity for long-running agent tasks
func AgentRunActivity(ctx context.Context, request RunRequest) (*AgentResponse, error) {
    // Execute agent
    // Heartbeat for long operations
    activity.RecordHeartbeat(ctx, "processing")
    return response, nil
}

// Workflow with signal handling for human approval
func AgentWorkflow(ctx workflow.Context, input AgentInput) (*AgentResponse, error) {
    // Run agent activity
    var response AgentResponse
    err := workflow.ExecuteActivity(ctx, AgentRunActivity, input.Request).Get(ctx, &response)
    
    // If requires approval, wait for signal
    if response.RequiresApproval {
        var approval ApprovalSignal
        workflow.GetSignalChannel(ctx, "approval").Receive(ctx, &approval)
        // Continue based on approval
    }
    
    return &response, nil
}
```

### 7.5 Key Differences to Consider

1. **Entity vs Workflow**: Temporal's Entity feature is experimental. Consider using Workflows with signals as the primary pattern.

2. **Replay Safety**: Temporal requires deterministic workflows. The `generate_unique_id()` pattern maps to `workflow.SideEffect()` or `workflow.GetInfo().WorkflowExecution.ID`.

3. **State Size**: Temporal has workflow history limits. Use Continue-As-New for long conversations.

4. **Polling Pattern**: Temporal supports synchronous query results via `workflow.Query`, eliminating need for explicit polling.

5. **TTL/Cleanup**: Use Temporal's Workflow Execution Timeout or Schedule for TTL-based cleanup.

---

## 8. Clarifying Questions

1. **Entity vs Workflow Decision**: Should the Go port use Temporal's experimental Entity feature, or implement using standard Workflows with signals? Workflows are more mature but require different patterns.

2. **State Limits**: What's the expected maximum conversation history size? This impacts whether Continue-As-New is needed.

3. **Streaming Support**: The Python implementation notes that streaming is "not supported for durable agents". Is this acceptable for the Go port, or should we explore Temporal's streaming capabilities?

4. **Cross-Platform Compatibility**: Should the Go port maintain the same state schema (`durable-agent-entity-state.json`) for potential cross-platform state sharing?

5. **Activity vs Entity Operations**: Should agent execution be modeled as a Temporal Activity (better for CPU-intensive work, supports heartbeats) or Entity operation (better for state management)?

6. **Human-in-the-Loop**: What specific human-in-the-loop patterns need to be supported? The Python implementation has `wait_for_response` for fire-and-forget, but deeper approval workflows may need different patterns.

---

## References

- [Python DurableTask Package](python/packages/durabletask/)
- [Python Azure Functions Package](python/packages/azurefunctions/)
- [State Schema](schemas/durable-agent-entity-state.json)
- [.NET DurableTask Implementation](dotnet/src/Microsoft.Agents.AI.DurableTask/)
- [Long-Running Operations ADR](docs/decisions/0009-support-long-running-operations.md)
- [TTL Design Doc](docs/features/durable-agents/durable-agents-ttl.md)
