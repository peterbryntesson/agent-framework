# Python Agent Framework — Architecture Research

> Generated: 2026-02-20  
> Source: `python/` directory of `agent-framework-peterbryntesson`  
> Scope: All packages, classes, protocols, patterns, and dependencies

---

## 1. Repository Structure

### 1.1 Workspace Layout

```
python/
├── pyproject.toml          # Meta-package "agent-framework" (uv workspace root)
├── README.md
├── CODING_STANDARD.md
├── shared_tasks.toml       # Poe task definitions
├── packages/
│   ├── core/               # agent-framework-core
│   ├── a2a/                # agent-framework-a2a
│   ├── ag-ui/              # agent-framework-ag-ui
│   ├── anthropic/          # agent-framework-anthropic
│   ├── azure-ai/           # agent-framework-azure-ai
│   ├── declarative/        # agent-framework-declarative
│   └── durabletask/        # agent-framework-durabletask
├── samples/
├── tests/
└── docs/
```

### 1.2 Build & Tooling

| Aspect | Value |
|---|---|
| Python version | `>=3.10` |
| Build backend | `flit-core` (most packages), `hatchling` (ag-ui) |
| Workspace manager | `uv` with `[tool.uv.workspace]` members |
| Linter/Formatter | `ruff` (line-length=120) |
| Type checking | `pyright` (strict), `mypy` (strict) |
| Testing | `pytest` + `pytest-asyncio` |
| Task runner | `poe` (poethepoet) |

---

## 2. Core Package (`agent-framework-core`)

### 2.1 Dependencies

| Dependency | Version | Purpose |
|---|---|---|
| `typing-extensions` | `>=4.12` | Backports for `Self`, `TypedDict`, `override`, `Unpack` |
| `pydantic` | `>=2,<3` | Models, settings, JSON schema, validation |
| `pydantic-settings` | `>=2,<3` | Environment-based configuration |
| `opentelemetry-api` | `>=1.39.0` | Distributed tracing API |
| `opentelemetry-sdk` | `>=1.39.0` | Tracing SDK |
| `opentelemetry-semantic-conventions-ai` | `>=0.4.13` | GenAI semantic conventions |
| `openai` | `>=1.99.0` | OpenAI SDK (core dependency) |
| `azure-identity` | `>=1` | Azure authentication |
| `mcp[ws]` | `>=1.24.0,<2` | Model Context Protocol |
| `python-dotenv` | `>=1.1.0` | `.env` file loading |

### 2.2 Module Map

| Module | Lines | Purpose |
|---|---|---|
| `_types.py` | 3176 | Core type definitions: Content, ChatMessage, ChatResponse, AgentResponse, ChatOptions, Role, FinishReason |
| `_tools.py` | 2325 | Tool definitions, function invocation, auto-invoke loop |
| `observability.py` | 2013 | OpenTelemetry instrumentation decorators, metric/span management |
| `_middleware.py` | 1589 | Middleware system: Agent, Chat, Function middleware pipelines |
| `_agents.py` | 1330 | Agent protocol and ChatAgent implementation |
| `_mcp.py` | 1245 | MCP integration: Stdio, WebSocket, StreamableHTTP tools |
| `_serialization.py` | 617 | SerializationMixin: to_dict/from_dict with dependency injection |
| `_threads.py` | 506 | Thread management: AgentThread, ChatMessageStore |
| `_clients.py` | ~500 | ChatClientProtocol and BaseChatClient |
| `_memory.py` | ~200 | Context/ContextProvider for memory augmentation |
| `_pydantic.py` | ~80 | AFBaseSettings, HTTPsUrl |
| `_telemetry.py` | ~50 | User-agent string management |
| `_logging.py` | ~40 | Centralized logging (`get_logger`, `setup_logging`) |
| `exceptions.py` | ~140 | Full exception hierarchy |
| `_workflows/` | 30+ files | Multi-agent orchestration engine |

---

## 3. Core Protocols & Abstract Base Classes

### 3.1 `AgentProtocol` (Protocol, runtime_checkable)

The top-level interface for all agents.

```python
@runtime_checkable
class AgentProtocol(Protocol):
    id: str
    name: str
    description: str | None

    def run(
        self,
        messages: str | ChatMessage | list[str] | list[ChatMessage] | None = None,
        *,
        thread: AgentThread | None = None,
        **kwargs: Any,
    ) -> AgentResponse: ...   # async

    def run_stream(
        self,
        messages: str | ChatMessage | list[str] | list[ChatMessage] | None = None,
        *,
        thread: AgentThread | None = None,
        **kwargs: Any,
    ) -> AsyncIterable[AgentResponseUpdate]: ...  # async generator

    def get_new_thread(self, **kwargs: Any) -> AgentThread: ...
```

### 3.2 `ChatClientProtocol[TOptions_contra]` (Protocol, runtime_checkable)

The interface for all chat clients.

```python
@runtime_checkable
class ChatClientProtocol(Protocol[TOptions_contra]):
    additional_properties: dict[str, Any]

    async def get_response(
        self,
        messages: str | ChatMessage | Sequence[str | ChatMessage],
        *,
        options: TOptions_contra | None = None,
        **kwargs: Any,
    ) -> ChatResponse: ...

    async def get_streaming_response(
        self,
        messages: str | ChatMessage | Sequence[str | ChatMessage],
        *,
        options: TOptions_contra | None = None,
        **kwargs: Any,
    ) -> AsyncIterable[ChatResponseUpdate]: ...
```

### 3.3 `ToolProtocol` (Protocol, runtime_checkable)

```python
@runtime_checkable
class ToolProtocol(Protocol):
    name: str
    description: str
    additional_properties: dict[str, Any]
```

### 3.4 `ChatMessageStoreProtocol` (Protocol)

```python
class ChatMessageStoreProtocol(Protocol):
    async def list_messages(self) -> list[ChatMessage]: ...
    async def add_messages(self, messages: Sequence[ChatMessage]) -> None: ...
    @classmethod
    async def deserialize(cls, serialized_store_state: MutableMapping[str, Any], **kwargs) -> Self: ...
    async def update_from_state(self, serialized_store_state: MutableMapping[str, Any], **kwargs) -> None: ...
    async def serialize(self, **kwargs) -> dict[str, Any]: ...
```

### 3.5 `ContextProvider` (ABC)

```python
class ContextProvider(ABC):
    DEFAULT_CONTEXT_PROMPT: ClassVar[str]

    async def thread_created(self, thread_id: str) -> None: ...
    async def invoked(self, messages: list[ChatMessage], **kwargs) -> None: ...
    @abstractmethod
    async def invoking(self, messages: list[ChatMessage], **kwargs) -> Context | None: ...
    # Also supports async context manager (__aenter__/__aexit__)
```

Returns `Context` containing optional `instructions`, `messages`, and `tools`.

### 3.6 Middleware ABCs

| Class | Context Type | Method |
|---|---|---|
| `AgentMiddleware` | `AgentRunContext` | `async process(context, next)` |
| `ChatMiddleware` | `ChatContext` | `async process(context, next)` |
| `FunctionMiddleware` | `FunctionInvocationContext` | `async process(context, next)` |

All use the pipeline pattern: call `next(context)` to proceed, inspect/modify `context.result` after.

### 3.7 `SerializationMixin`

Core serialization infrastructure providing:

- `to_dict(*, exclude, exclude_none)` — Deep serialization with nested object support
- `from_dict(value, *, dependencies)` — Deserialization with dependency injection system
- `to_json()` / `from_json()` — JSON convenience methods
- `DEFAULT_EXCLUDE: ClassVar[set[str]]` — Fields always excluded from serialization
- `INJECTABLE: ClassVar[set[str]]` — Fields injected at deserialization time, excluded from serialization
- Type identifier via `_get_type_identifier()` using snake_case class name

---

## 4. Core Implementations

### 4.1 `BaseAgent(SerializationMixin)`

Base class for all agents:

- Auto-generates `id` (UUID4)
- Holds `name`, `description`, `context_provider`, `middleware`, `additional_properties`
- `get_new_thread()` — Creates AgentThread
- `deserialize_thread()` — Restores thread from dict
- `as_tool()` — Converts agent into a `FunctionTool` for use by other agents
- `_notify_thread_of_new_messages()` — Updates thread history

### 4.2 `ChatAgent(BaseAgent, Generic[TOptions_co])`

Primary agent implementation. Decorated with `@use_agent_middleware` and `@use_agent_instrumentation`.

```python
class ChatAgent(BaseAgent, Generic[TOptions_co]):
    def __init__(
        self,
        chat_client: ChatClientProtocol,
        instructions: str | None = None,
        *,
        id: str | None = None,
        name: str | None = None,
        description: str | None = None,
        tools: Sequence[ToolProtocol | Callable] | None = None,
        default_options: dict[str, Any] | None = None,
        chat_message_store_factory: Callable[[], ChatMessageStoreProtocol] | None = None,
        context_provider: ContextProvider | None = None,
        middleware: Sequence[Middleware] | None = None,
        **kwargs: Any,
    ): ...
```

Key behaviors:

- `run()` and `run_stream()` with typed options overloads
- Supports async context manager for MCP tool lifecycle
- Manages thread with dual storage modes: service-managed (`service_thread_id`) or local (`message_store`)
- Merges `default_options` with per-call options
- Integrates `ContextProvider` for memory/RAG augmentation
- `as_mcp_server()` — Exposes agent as an MCP server

### 4.3 `BaseChatClient(SerializationMixin, ABC, Generic[TOptions_co])`

Abstract base for all chat clients:

- `OTEL_PROVIDER_NAME: ClassVar[str]` — Provider identifier for telemetry
- Abstract: `_inner_get_response()`, `_inner_get_streaming_response()`
- Public: `get_response()`, `get_streaming_response()` (with TypedDict-based options overloads)
- `service_url()` — Returns service endpoint
- `as_agent()` — Convenience to wrap in `ChatAgent`
- Middleware support via constructor

### 4.4 `FunctionTool(BaseTool, Generic[ArgsT, ReturnT])`

Wraps Python functions for AI model tool calling:

- `func` — The wrapped callable (sync or async)
- `input_model` — Auto-generated Pydantic model from function signature
- `approval_mode` — `"always_require"` or `"never_require"`
- `max_invocations` / `max_invocation_exceptions` — Rate limiting
- `invoke(*, arguments, **kwargs)` — Execute the function
- `parameters()` — Cached JSON schema for the function parameters
- `to_json_schema_spec()` — Full OpenAI-compatible tool spec
- `__call__()` — Direct callable support
- `__get__()` — Descriptor protocol for method binding
- OTel histogram for invocation duration tracking

### 4.5 Hosted Tools

- `HostedCodeInterpreterTool` — Server-side code execution
- `HostedWebSearchTool` — Server-side web search (Bing)
- `HostedImageGenerationTool` — Server-side image generation
- `HostedMCPTool` — Server-side MCP tool
- `HostedFileSearchTool` — Server-side file/vector search

### 4.6 MCP Tools

- `MCPStdioTool` — Connects to MCP servers via stdio
- `MCPStreamableHTTPTool` — Connects via StreamableHTTP
- `MCPWebsocketTool` — Connects via WebSocket

All support async context manager for connection lifecycle, expand to constituent `FunctionTool` instances.

---

## 5. Type System

### 5.1 `Content` (SerializationMixin)

Unified content container using a discriminated union pattern via `type` field.

**Content Types** (17 total):

| Type | Factory Method | Key Fields |
|---|---|---|
| `text` | `from_text()` | `text`, `protected_data` |
| `text_reasoning` | `from_text()` | `text`, `protected_data` |
| `data` | `from_data()` | `data`, `media_type`, `uri` |
| `uri` | `from_uri()` | `uri`, `media_type` |
| `error` | `from_error()` | `message`, `error_code`, `error_details` |
| `function_call` | `from_function_call()` | `call_id`, `name`, `arguments`, `exception` |
| `function_result` | `from_function_result()` | `call_id`, `name`, `result` |
| `usage` | `from_usage()` | `usage_details` |
| `hosted_file` | `from_hosted_file()` | `file_id`, `uri`, `media_type` |
| `hosted_vector_store` | `from_hosted_vector_store()` | `vector_store_id` |
| `code_interpreter_tool_call` | `from_code_interpreter_tool_call()` | `call_id`, `inputs` |
| `code_interpreter_tool_result` | `from_code_interpreter_tool_result()` | `call_id`, `outputs` |
| `image_generation_tool_call` | `from_image_generation_tool_call()` | `call_id` |
| `image_generation_tool_result` | `from_image_generation_tool_result()` | `call_id`, `image_id` |
| `mcp_server_tool_call` | `from_mcp_server_tool_call()` | `call_id`, `tool_name`, `server_name`, `arguments` |
| `mcp_server_tool_result` | `from_mcp_server_tool_result()` | `call_id`, `tool_name`, `server_name`, `output` |
| `function_approval_request` | `from_function_approval_request()` | `id`, `function_call`, `user_input_request` |
| `function_approval_response` | `from_function_approval_response()` | `id`, `approved`, `function_call` |

Supports `__add__()` for text, text_reasoning, function_call, usage — enabling stream chunk concatenation.

### 5.2 Message & Response Types

| Type | Purpose | Key Properties |
|---|---|---|
| `ChatMessage` | Single message | `role`, `contents: list[Content]`, `text`, `author_name`, `message_id` |
| `ChatResponse[T]` | Full response | `messages`, `text`, `value: T`, `finish_reason`, `usage_details`, `model_id`, `conversation_id` |
| `ChatResponseUpdate` | Streaming chunk | `contents`, `text`, `role`, `finish_reason`, `message_id` |
| `AgentResponse[T]` | Agent run result | `messages`, `text`, `value: T`, `usage_details`, `user_input_requests` |
| `AgentResponseUpdate` | Agent streaming chunk | `contents`, `text`, `role`, `user_input_requests` |

Both `ChatResponse` and `AgentResponse` support:

- `from_*_updates()` / `from_*_generator()` — Assemble from streaming chunks
- `try_parse_value(output_format_type)` — Parse text into Pydantic model
- Lazy `value` property with caching

### 5.3 Enum-Like Classes

Uses `EnumLike` metaclass:

- `Role` — `SYSTEM`, `USER`, `ASSISTANT`, `TOOL`
- `FinishReason` — `STOP`, `LENGTH`, `TOOL_CALLS`, `CONTENT_FILTER`

### 5.4 `ChatOptions` (TypedDict)

Common options across providers:

```python
class ChatOptions(TypedDict, total=False):
    model_id: str
    temperature: float
    top_p: float
    max_tokens: int
    stop: str | Sequence[str]
    seed: int
    logit_bias: dict[str | int, float]
    frequency_penalty: float
    presence_penalty: float
    tools: ToolProtocol | Callable | Sequence[...] | None
    tool_choice: ToolMode | Literal["auto", "required", "none"]
    allow_multiple_tool_calls: bool
    response_format: type[BaseModel] | Mapping[str, Any] | None
    metadata: dict[str, Any]
    user: str
    store: bool
    conversation_id: str
    instructions: str
```

Utilities: `validate_chat_options()`, `merge_chat_options()`, `normalize_tools()`, `validate_tools()`, `validate_tool_mode()`.

---

## 6. Middleware System

### 6.1 Architecture

Three-level middleware pipeline, each using the same pattern:

```
[Middleware 1] → [Middleware 2] → ... → [Final Handler]
     ↓                ↓                      ↓
  process()       process()              execute()
     ↓                ↓                      ↓
  next(ctx)       next(ctx)              ctx.result = ...
```

### 6.2 Context Objects

| Context | Key Fields |
|---|---|
| `AgentRunContext` | `agent`, `messages`, `thread`, `is_streaming`, `metadata`, `result`, `terminate` |
| `ChatContext` | `chat_client`, `messages`, `options`, `is_streaming`, `metadata`, `result`, `terminate` |
| `FunctionInvocationContext` | `function`, `arguments`, `metadata`, `result`, `terminate` |

### 6.3 Middleware Registration

- Class-based: Subclass `AgentMiddleware`/`ChatMiddleware`/`FunctionMiddleware`
- Function-based: Use `@agent_middleware`/`@chat_middleware`/`@function_middleware` decorators
- Auto-wrapped via `MiddlewareWrapper` for functions

### 6.4 Pipeline Executors

- `AgentMiddlewarePipeline` — For agent `run()`/`run_stream()`
- `ChatMiddlewarePipeline` — For chat client `get_response()`/`get_streaming_response()`
- `FunctionMiddlewarePipeline` — For function tool `invoke()`

### 6.5 Class Decorators

- `@use_agent_middleware` — Adds agent middleware pipeline to agent class
- `@use_chat_middleware` — Adds chat middleware pipeline to chat client class
- `@use_function_invocation` — Adds auto-invoke loop with function middleware to chat client

---

## 7. Observability

### 7.1 Key Exports

- `use_instrumentation` — Class decorator for chat clients, traces `get_response`/`get_streaming_response`
- `use_agent_instrumentation` — Class decorator for agents, traces `run`/`run_stream`
- `configure_otel_providers()` — Configure OTLP exporters from params or env vars
- `enable_instrumentation()` — Toggle instrumentation on/off
- `create_resource()` — Create `Resource` with service name/version
- `create_metric_views()` — Configure histogram buckets

### 7.2 `OtelAttr` Enum

Comprehensive enum mapping all GenAI semantic convention attributes:

- Operation: `gen_ai.operation.name`, `gen_ai.provider.name`
- Request: `gen_ai.request.seed`, `gen_ai.request.temperature`, etc.
- Response: `gen_ai.response.finish_reasons`, `gen_ai.response.id`
- Usage: `gen_ai.usage.input_tokens`, `gen_ai.usage.output_tokens`
- Tool: `gen_ai.tool.name`, `gen_ai.tool.call.arguments`, `gen_ai.tool.call.result`
- Agent: `gen_ai.agent.id`, `gen_ai.agent.name`
- Workflow: `workflow.id`, `workflow.name`, `executor.id`, `edge_group.type`

### 7.3 `ObservabilitySettings` (AFBaseSettings)

Configurable via `AGENT_FRAMEWORK_OBSERVABILITY_*` env vars. Controls enablement, sensitive data capture.

---

## 8. Thread Management

### 8.1 `AgentThread`

Dual-mode thread:

- **Service-managed**: Uses `service_thread_id` — server stores history
- **Local**: Uses `ChatMessageStoreProtocol` — client stores history
- Mutually exclusive: setting one clears/blocks the other

Methods: `on_new_messages()`, `serialize()`, `deserialize()`, `update_from_thread_state()`.

### 8.2 `ChatMessageStore`

Default in-memory implementation of `ChatMessageStoreProtocol`. Stores `list[ChatMessage]` with full serialization support via `ChatMessageStoreState`.

---

## 9. Workflows (`_workflows/`)

### 9.1 Architecture

Pregel-style superstep execution engine for multi-agent orchestration.

```
WorkflowBuilder → build() → Workflow → run() / run_stream()
                                           ↓
                                        Runner (Pregel-style superstep execution)
                                           ↓
                                    [Executor 1] → [Executor 2] → ...
                                    via message passing through edges
```

### 9.2 Key Classes

#### `Workflow`

```python
class Workflow:
    def run(
        self,
        *,
        inputs: dict[str, Any] | None = None,
        shared_state: SharedState | None = None,
        max_steps: int = 100,
    ) -> WorkflowRunResult: ...  # async

    async def run_stream(
        self,
        *,
        inputs: dict[str, Any] | None = None,
        shared_state: SharedState | None = None,
        max_steps: int = 100,
    ) -> AsyncIterable[WorkflowEvent]: ...
```

#### `WorkflowBuilder`

Fluent API for constructing workflows:

```python
builder = WorkflowBuilder()
builder.register_executor(executor_a)
builder.register_executor(executor_b)
builder.add_edge(executor_a, executor_b)
builder.add_fan_out_edge(executor_a, [executor_b, executor_c])
builder.add_fan_in_edge([executor_b, executor_c], executor_d)
builder.add_switch_case_edge(executor_a, cases={"route1": executor_b, "route2": executor_c})
builder.set_start_executor(executor_a)
workflow = builder.build()
```

#### `Executor` (Base Class)

Processing unit with typed message routing:

```python
class Executor:
    @handler
    async def handle_message(self, context: WorkflowContext, message: MyMessageType) -> None:
        # Process message
        context.send_message(target_executor, output_message)
        context.yield_output(output_event)
```

Uses `@handler` decorator for type-safe message routing. Each handler receives messages of a specific type.

#### `WorkflowContext[T_Out, T_W_Out]`

Generic context for handlers:

- `send_message(target, message)` — Route messages to other executors
- `yield_output(output)` — Emit workflow output events

#### `Runner`

Pregel-style superstep execution:

1. Deliver messages from previous step to target executors
2. Execute all active executors in parallel
3. Collect outgoing messages
4. Check convergence (no more messages)
5. Repeat until convergence or max_steps

Supports checkpointing via `CheckpointStorage`.

### 9.3 High-Level Builders

| Builder | Pattern | Description |
|---|---|---|
| `SequentialBuilder` | Chain | A → B → C (linear pipeline) |
| `ConcurrentBuilder` | Fan-out/Fan-in | A → [B, C, D] → E (parallel execution) |
| `HandoffBuilder` | Decentralized | Agents route to each other via tool calls |
| `GroupChatBuilder` | Centralized | Orchestrator selects next agent via selection function |
| `MagenticBuilder` | Magentic-One | Specialized orchestration pattern |

### 9.4 Executor Types

| Type | Description |
|---|---|
| `AgentExecutor` | Wraps `AgentProtocol` for workflow execution |
| `FunctionExecutor` / `@executor` | Wraps Python functions as executors |
| `WorkflowExecutor` | Sub-workflow composition (nested workflows) |

### 9.5 Edge Types

| Edge Type | Behavior |
|---|---|
| `SingleEdgeGroup` | 1:1 message routing |
| `FanOutEdgeGroup` | 1:N broadcast |
| `FanInEdgeGroup` | N:1 aggregation with barrier |
| `SwitchCaseEdgeGroup` | Conditional routing based on message content |

### 9.6 Events

`WorkflowEvent` hierarchy:

- `WorkflowStartedEvent`, `WorkflowCompletedEvent`, `WorkflowFailedEvent`
- `WorkflowOutputEvent` — Output from `yield_output()`
- `SuperStepStartedEvent`, `SuperStepCompletedEvent`
- `ExecutorInvokedEvent`, `ExecutorCompletedEvent`
- `RequestInfoEvent` — For user input during workflow

### 9.7 State & Checkpointing

- `SharedState` — Shared mutable state between executors
- `CheckpointStorage` (Protocol) — Interface for persistence
  - `FileCheckpointStorage` — File-based
  - `InMemoryCheckpointStorage` — Memory-based

---

## 10. External Packages

### 10.1 `agent-framework-a2a`

**Dependencies**: `agent-framework-core`, `a2a-sdk>=0.3.5`

**Key Class: `A2AAgent(BaseAgent)`**

Wraps Agent-to-Agent protocol (Google A2A SDK) for interoperability:

- `run()` — Collects streaming response into `AgentResponse`
- `run_stream()` — Yields `AgentResponseUpdate` from A2A streaming
- Converts `ChatMessage` ↔ A2A `Message`
- Supports initialization via URL, AgentCard, or Client
- Transport negotiation with fallback

### 10.2 `agent-framework-ag-ui`

**Dependencies**: `agent-framework-core`, `ag-ui-protocol>=0.1.9`, `fastapi>=0.115.0`, `uvicorn>=0.30.0`

**Key Classes:**

| Class | Purpose |
|---|---|
| `AgentFrameworkAgent` | Wraps `AgentProtocol` for AG-UI protocol (SSE events) |
| `AGUIChatClient(BaseChatClient)` | Chat client for AG-UI servers (SSE streaming) |
| `AGUIHttpService` | HTTP service wrapper |
| `AGUIEventConverter` | Converts AG-UI events to Agent Framework types |

**Key Function:**

```python
def add_agent_framework_fastapi_endpoint(
    app: FastAPI,
    agent: AgentProtocol | AgentFrameworkAgent,
    path: str = "/",
    state_schema: Any | None = None,
    predict_state_config: dict | None = None,
    allow_origins: list[str] | None = None,
    default_state: dict | None = None,
    tags: list[str] | None = None,
    dependencies: Sequence[Depends] | None = None,
) -> None
```

Creates FastAPI POST endpoint returning `StreamingResponse` with `text/event-stream`.

### 10.3 `agent-framework-anthropic`

**Dependencies**: `agent-framework-core`, `anthropic>=0.70.0,<1`

**Key Class: `AnthropicClient(BaseChatClient[TAnthropicOptions])`**

Decorated with `@use_function_invocation`, `@use_instrumentation`, `@use_chat_middleware`.

- Uses `AsyncAnthropic` client
- `AnthropicSettings(AFBaseSettings)` — env prefix `ANTHROPIC_`
- `AnthropicChatOptions(ChatOptions)` — Extends with `top_k`, `service_tier`, `thinking` (ThinkingConfig), `container`, `additional_beta_flags`
- Beta flags: `"mcp-client-2025-04-04"`, `"code-execution-2025-08-25"`
- `OTEL_PROVIDER_NAME = "anthropic"`

### 10.4 `agent-framework-azure-ai`

**Dependencies**: `agent-framework-core`, `azure-ai-projects>=2.0.0b3`, `azure-ai-agents==1.2.0b5`, `aiohttp`

**Key Classes:**

| Class | Purpose |
|---|---|
| `AzureAIAgentClient(BaseChatClient)` | Azure AI Agent Service chat client |
| `AzureAIClient` | Azure AI Foundry client |
| `AzureAIAgentsProvider` | Agent provider for Azure AI |
| `AzureAIProjectAgentProvider` | Project-level agent provider |
| `AzureAISettings(AFBaseSettings)` | Settings with `AZURE_AI_*` env vars |

`AzureAIAgentClient` features:

- Decorated with `@use_function_invocation`, `@use_instrumentation`, `@use_chat_middleware`
- Auto-creates/manages `AgentsClient` and agent instances
- `AzureAIAgentOptions(ChatOptions)` — Extends with `conversation_id`, `tool_resources`
- Option translation map: `model_id` → `model`, `max_tokens` → `max_completion_tokens`
- `OTEL_PROVIDER_NAME = "azure.ai"`

### 10.5 `agent-framework-declarative`

**Dependencies**: `agent-framework-core`, `powerfx>=0.0.31` (Python <3.14), `pyyaml>=6.0`

**Key Classes:**

| Class | Purpose |
|---|---|
| `AgentFactory` | Creates `ChatAgent` from YAML definitions |
| `WorkflowFactory` | Creates workflows from YAML definitions |

**`AgentFactory`** features:

- `create_agent_from_yaml_path(yaml_path)` → `ChatAgent`
- `create_agent_from_yaml(yaml_content)` → `ChatAgent`
- Provider type mapping for 10+ providers:
  - `AzureOpenAI.Chat`, `AzureOpenAI.Assistants`, `AzureOpenAI.Responses`
  - `OpenAI.Chat`, `OpenAI.Assistants`, `OpenAI.Responses`
  - `AzureAIAgentClient`, `AzureAIClient`, `AzureAI.ProjectProvider`
  - `Anthropic.Chat`
- Extensible via `additional_mappings` parameter
- PowerFx expressions support (with safe_mode toggle)
- Bindings and connections for dependency injection

### 10.6 `agent-framework-durabletask`

**Dependencies**: `agent-framework-core`, `durabletask>=1.3.0`, `durabletask-azuremanaged>=1.3.0`

**Key Classes:**

| Class | Purpose |
|---|---|
| `DurableAIAgentWorker` | Wraps TaskHubGrpcWorker to register agents as durable entities |
| `DurableAIAgentClient` | Client wrapper for interacting with durable agents via gRPC |
| `DurableAIAgent(AgentProtocol)` | Proxy that delegates execution to durable entity (sync `run()`, no streaming) |
| `AgentEntity` | Platform-agnostic agent execution logic within entity context |
| `AgentEntityStateProviderMixin` | State caching + (de)serialization for entities |
| `DurableAgentExecutor` | Abstract execution strategy (client vs orchestration) |
| `ClientAgentExecutor` | Execution via gRPC client (polling for result) |
| `DurableAgentTask` | Custom Task wrapping entity calls with typed AgentResponse |
| `DurableAgentState` | Conversation history state model |

**Architecture:**

```
DurableAIAgentWorker
  └── add_agent(agent) → creates ConfiguredAgentEntity
       └── registers with TaskHubGrpcWorker as "dafx-{agent_name}"

DurableAIAgentClient
  └── get_agent(name) → DurableAIAgent proxy
       └── run(messages) → calls entity via gRPC
```

---

## 11. Exception Hierarchy

```
AgentFrameworkException
├── AgentException
│   ├── AgentExecutionException
│   ├── AgentInitializationError
│   └── AgentThreadException
├── ChatClientException
│   └── ChatClientInitializationError
├── ServiceException
│   ├── ServiceInitializationError
│   ├── ServiceResponseException
│   │   ├── ServiceContentFilterException
│   │   ├── ServiceInvalidExecutionSettingsError
│   │   ├── ServiceInvalidRequestError
│   │   └── ServiceInvalidResponseError
│   └── ServiceInvalidAuthError
├── ToolException
│   └── ToolExecutionException
└── AdditionItemMismatch
```

---

## 12. Design Patterns

### 12.1 Structural Subtyping (Protocols)

All major interfaces use `Protocol` for structural subtyping rather than nominal inheritance. This allows:

- Third-party implementations without inheriting from framework classes
- `runtime_checkable` for isinstance checks
- Clean duck-typing integration

### 12.2 Decorator Composition

Chat clients and agents use stacked decorators for cross-cutting concerns:

```python
@use_function_invocation      # Auto-invoke tool calls
@use_instrumentation          # OpenTelemetry tracing
@use_chat_middleware           # Chat middleware pipeline
class MyClient(BaseChatClient):
    ...
```

### 12.3 TypedDict-Based Options

`ChatOptions` is a `TypedDict` (not a Pydantic model) — enabling `**opts` unpacking and Unpack typing while being lightweight. Provider-specific options extend it.

### 12.4 Generic Type Parameters

Extensive use of Generics:

- `BaseChatClient[TOptions_co]` — Options type
- `ChatResponse[TResponseModel]` — Structured output type
- `FunctionTool[ArgsT, ReturnT]` — Input/output types
- `WorkflowContext[T_Out, T_W_Out]` — Output types

### 12.5 Lazy Loading

`__init__.py` uses `__getattr__` for lazy-loading connector packages (OpenAI, Azure, Anthropic, etc.) — installed only when needed via the `[all]` extra.

### 12.6 Async-First

All I/O operations are async. Streaming uses `AsyncIterable[T]` / `AsyncGenerator[T, None]`.

### 12.7 Pregel-Style Workflow Engine

The workflow engine uses a Pregel-style superstep model:

1. Messages are buffered during a superstep
2. All executors with pending messages run in parallel
3. New messages are enqueued for the next superstep
4. Convergence when no new messages are produced

### 12.8 Function Invocation Loop

Chat clients decorated with `@use_function_invocation` get an auto-invoke loop:

1. Send messages to model
2. If model returns function_call content, invoke the tool
3. Send function_result back to model
4. Repeat until model returns non-tool response or max_iterations reached

Configurable via `FunctionInvocationConfiguration`:

- `max_iterations` (default 40)
- `max_consecutive_errors_per_request` (default 3)
- `terminate_on_unknown_calls`
- `additional_tools`
- `include_detailed_errors`

### 12.9 Dual-Mode Thread Storage

Threads support both:

- **Server-managed**: `service_thread_id` — history stored server-side
- **Client-managed**: `ChatMessageStoreProtocol` — history stored locally
- Mutually exclusive, auto-detected from service responses

---

## 13. Package Dependency Graph

```
agent-framework (meta)
└── agent-framework-core[all]
    ├── agent-framework-a2a ← a2a-sdk
    ├── agent-framework-ag-ui ← ag-ui-protocol, fastapi, uvicorn
    ├── agent-framework-anthropic ← anthropic
    ├── agent-framework-azure-ai ← azure-ai-projects, azure-ai-agents
    ├── agent-framework-declarative ← powerfx, pyyaml
    ├── agent-framework-durabletask ← durabletask, durabletask-azuremanaged
    ├── agent-framework-openai ← openai (already in core)
    ├── agent-framework-azure ← openai (Azure OpenAI wrappers)
    ├── agent-framework-foundry ← azure-ai-projects
    └── ... (17+ connector packages)
```

---

## 14. Key File Sizes Reference

| File | Lines | Core Types Defined |
|---|---|---|
| `_types.py` | 3176 | Content, ContentType, ChatMessage, ChatResponse, ChatResponseUpdate, AgentResponse, AgentResponseUpdate, ChatOptions, Role, FinishReason, ToolMode, UsageDetails |
| `_tools.py` | 2325 | FunctionTool, BaseTool, ToolProtocol, Hosted*Tool, FunctionInvocationConfiguration, tool decorator, use_function_invocation |
| `observability.py` | 2013 | OtelAttr, ObservabilitySettings, use_instrumentation, use_agent_instrumentation, configure_otel_providers |
| `_middleware.py` | 1589 | AgentMiddleware, ChatMiddleware, FunctionMiddleware, *Context, *Pipeline, middleware decorators |
| `_agents.py` | 1330 | AgentProtocol, BaseAgent, ChatAgent |
| `_mcp.py` | 1245 | MCPStdioTool, MCPStreamableHTTPTool, MCPWebsocketTool |
| `_serialization.py` | 617 | SerializationMixin, SerializationProtocol |
| `_threads.py` | 506 | AgentThread, ChatMessageStore, ChatMessageStoreProtocol |
| `_clients.py` | ~500 | ChatClientProtocol, BaseChatClient |
