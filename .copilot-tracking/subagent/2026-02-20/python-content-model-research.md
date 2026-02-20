# Python Core Package Research: Content Model, Streaming, and Tool System

**Package**: `python/packages/core/agent_framework/`
**Date**: 2026-02-20
**Scope**: Content types, message model, streaming, tool system, agent lifecycle, middleware

---

## 1. Content Types System

### Design Philosophy: Unified Content Class

The framework uses a **single unified `Content` class** (not discriminated union subclasses). All content varieties share one class with a `type` field acting as a Literal string discriminator. Fields irrelevant to a given type are simply `None`.

**File**: [_types.py](python/packages/core/agent_framework/_types.py)

### ContentType Literal

```python
ContentType = Literal[
    "text", "text_reasoning", "data", "uri", "error",
    "function_call", "function_result", "usage",
    "hosted_file", "hosted_vector_store",
    "code_interpreter_tool_call", "code_interpreter_tool_result",
    "image_generation_tool_call", "image_generation_tool_result",
    "mcp_server_tool_call", "mcp_server_tool_result",
    "function_approval_request", "function_approval_response",
]
```

18 content types covering text, binary data, URIs, errors, tool calls/results, hosted resources, approval workflows, and usage telemetry.

### Content Class Fields

| Field | Type | Used By |
|---|---|---|
| `type` | `ContentType` | All |
| `text` | `str \| None` | text, text_reasoning, error, function_result |
| `uri` | `str \| None` | uri |
| `data` | `str \| None` | data (base64) |
| `media_type` | `str \| None` | data, uri |
| `call_id` | `str \| None` | function_call, function_result, mcp_server_tool_call/result |
| `name` | `str \| None` | function_call, mcp_server_tool_call |
| `arguments` | `str \| None` | function_call |
| `result` | `str \| None` | function_result, code_interpreter_tool_result |
| `exception` | `str \| None` | function_result |
| `usage` | `UsageDetails \| None` | usage |
| `function_call` | `Content \| None` | function_approval_request, function_approval_response |
| `approved` | `bool \| None` | function_approval_response |
| `id` | `str \| None` | function_approval_request, function_approval_response |
| `annotations` | `list[Annotation] \| None` | text |
| `file_id` | `str \| None` | hosted_file |
| `vector_store_id` | `str \| None` | hosted_vector_store |
| `additional_properties` | `dict[str, Any]` | All (arbitrary extra data) |
| `raw_representation` | `Any` | All (original provider object) |

### Factory Methods (classmethods)

Each content type has a dedicated factory:

- `Content.from_text(text, annotations=None)` — text content with optional citation annotations
- `Content.from_text_reasoning(text)` — chain-of-thought/reasoning text
- `Content.from_data(data, media_type)` — base64-encoded binary (images, audio)
- `Content.from_uri(uri, media_type=None, text=None)` — resource URI reference
- `Content.from_error(error)` — error message
- `Content.from_function_call(call_id, name, arguments)` — model-generated tool invocation
- `Content.from_function_result(call_id, result, exception=None)` — tool execution result
- `Content.from_usage(usage)` — token usage telemetry
- `Content.from_hosted_file(file_id)` — service-side file reference
- `Content.from_hosted_vector_store(vector_store_id)` — service-side vector store
- `Content.from_code_interpreter_tool_call(call_id, arguments)` — hosted code interpreter call
- `Content.from_code_interpreter_tool_result(call_id, result)` — code interpreter output
- `Content.from_image_generation_tool_call(call_id, arguments)` — hosted image gen call
- `Content.from_image_generation_tool_result(call_id, result)` — hosted image gen output
- `Content.from_mcp_server_tool_call(call_id, name, arguments)` — MCP server tool call
- `Content.from_mcp_server_tool_result(call_id, result)` — MCP server tool result
- `Content.from_function_approval_request(id, function_call)` — approval request wrapping a function_call Content
- `Content.from_function_approval_response(id, function_call, approved)` — approval decision

### Content Addition (`__add__`)

Content supports merging via `+` for streaming chunk assembly:

- **text + text**: Concatenates `.text` fields
- **text_reasoning + text_reasoning**: Concatenates `.text` fields
- **function_call + function_call**: Concatenates `.arguments` fields (for streamed argument fragments)
- **usage + usage**: Sums token counts
- Other type mismatches raise `AdditionItemMismatch`

### Serialization

- `to_dict(exclude_none=True)` — dictionary representation; omits None fields
- `from_dict(data)` — reconstruction from dictionary
- `parse_arguments()` — for function_call type, JSON-parses `.arguments` string into dict

### Helper Types

- **`Annotation`**: TypedDict with `type`, `text`, `start_index`, `end_index`, `source` (uri-based citations)
- **`TextSpanRegion`**: TypedDict with `start`, `end` for annotation regions
- **`UsageDetails`**: TypedDict with `input_token_count`, `output_token_count`, `total_token_count`

---

## 2. Message Model

### ChatMessage

**File**: [_types.py](python/packages/core/agent_framework/_types.py)

```python
class ChatMessage:
    role: Role
    contents: list[Content]
    author_name: str | None
    message_id: str | None
    additional_properties: dict[str, Any]
    raw_representation: Any
```

**Constructor overloads**:

- `ChatMessage(role, text="Hello")` — creates single text Content from string shorthand
- `ChatMessage(role, contents=[Content.from_text("Hello"), Content.from_data(...)])` — explicit content list

**Properties**:

- `.text` — concatenates all `text`-type contents with newlines; returns `""` if none

### Role (EnumLike)

Custom metaclass `_EnumLikeMeta` creates class-level constants from `_constants` dict:

- `Role.SYSTEM` → `"system"`
- `Role.USER` → `"user"`
- `Role.ASSISTANT` → `"assistant"`
- `Role.TOOL` → `"tool"`

Supports equality comparison with strings: `role == "assistant"` works.

### FinishReason (EnumLike)

- `FinishReason.CONTENT_FILTER` → `"content_filter"`
- `FinishReason.LENGTH` → `"length"`
- `FinishReason.STOP` → `"stop"`
- `FinishReason.TOOL_CALLS` → `"tool_calls"`

### ChatOptions (_ChatOptionsBase TypedDict)

Passed as `options: dict[str, Any]` throughout the framework:

| Field | Type | Description |
|---|---|---|
| `model_id` | `str` | Model identifier |
| `temperature` | `float` | Sampling temperature |
| `top_p` | `float` | Nucleus sampling |
| `max_tokens` | `int` | Max output tokens |
| `stop` | `list[str]` | Stop sequences |
| `seed` | `int` | Deterministic seed |
| `logit_bias` | `dict[str, float]` | Token bias |
| `frequency_penalty` | `float` | Frequency penalty |
| `presence_penalty` | `float` | Presence penalty |
| `tools` | tools specification | Tools available to model |
| `tool_choice` | `ToolMode` | Tool selection mode |
| `response_format` | `type \| dict` | Structured output format |
| `instructions` | `str` | System instructions |
| `metadata` | `dict` | Provider metadata |
| `user` | `str` | End-user identifier |
| `store` | `bool` | Whether to store conversation |
| `conversation_id` | `str` | Service-side conversation ID |
| `allow_multiple_tool_calls` | `bool` | Allow parallel tool calls |

### ToolMode TypedDict

```python
class ToolMode(TypedDict, total=False):
    mode: Literal["auto", "required", "none"]
    required_function_name: str
```

---

## 3. Streaming Architecture

### Response Types

The framework uses a bidirectional assembly pattern: streaming chunks arrive as updates, which can be assembled into complete responses.

#### ChatResponseUpdate

Streaming chunk from chat clients:

```python
class ChatResponseUpdate:
    contents: list[Content]
    role: Role | None
    response_id: str | None
    message_id: str | None
    conversation_id: str | None
    model_id: str | None
    finish_reason: FinishReason | None
    created_at: str | None
    additional_properties: dict[str, Any]
    raw_representation: Any
```

#### ChatResponse[TResponseModel]

Complete response assembled from updates:

```python
class ChatResponse(Generic[TResponseModel]):
    messages: list[ChatMessage]
    response_id: str | None
    conversation_id: str | None
    model_id: str | None
    created_at: str | None
    finish_reason: FinishReason | None
    usage_details: UsageDetails | None
    additional_properties: dict[str, Any]
    raw_representation: Any
```

**Assembly methods**:

- `ChatResponse.from_chat_response_updates(updates: list[ChatResponseUpdate])` — assembles complete response from collected update list
- `ChatResponse.from_chat_response_generator(generator: AsyncIterable[ChatResponseUpdate])` — consumes entire async generator into response

**Generic `TResponseModel`**: When `response_format` specifies a Pydantic model, `ChatResponse.value` property calls `model.model_validate_json()` on the text content to produce typed structured output.

#### AgentResponseUpdate / AgentResponse[TResponseModel]

Mirror `ChatResponseUpdate`/`ChatResponse` but at the agent level, with additional support for `user_input_requests` (filtering approval request contents).

- `AgentResponse.from_agent_run_response_updates(updates)` — assembly from agent update list
- `AgentResponse.from_agent_response_generator(gen)` — assembly from async generator

### Streaming Assembly Process (_process_update)

The `_process_update()` function merges each incoming `ChatResponseUpdate` into a building response:

1. Sets `response_id`, `model_id`, `conversation_id`, `finish_reason`, `created_at` from first update that has them
2. For each content in the update:
   - **Same `message_id`**: Merges into existing message using `Content.__add__` (text concatenation, argument streaming)
   - **New `message_id`**: Creates new ChatMessage
3. `_coalesce_text_content()` merges adjacent text chunks within a message
4. `_finalize_response()` extracts `usage_details` from usage-type content, sets finish_reason

### Client-Level Streaming

`BaseChatClient.get_streaming_response()` returns `AsyncIterable[ChatResponseUpdate]`.

### Agent-Level Streaming

`ChatAgent.run_stream()` returns `AsyncIterable[AgentResponseUpdate]`:

1. Normalizes input messages, prepares thread
2. Connects MCP tools
3. Calls `chat_client.get_streaming_response()`
4. Maps each `ChatResponseUpdate` → `AgentResponseUpdate` (adds `agent_id`, `agent_name`)
5. Yields each update
6. After generator exhausts, assembles `ChatResponse.from_chat_response_updates()` for thread notification

---

## 4. Tool System

### Tool Hierarchy

**File**: [_tools.py](python/packages/core/agent_framework/_tools.py)

```
ToolProtocol (runtime_checkable Protocol)
├── BaseTool(SerializationMixin)
│   └── FunctionTool(BaseTool, Generic[ArgsT, ReturnT])
│       ├── Locally executed function tools
│       └── MCP-loaded function tools (created by MCPTool.load_tools())
├── HostedCodeInterpreterTool(BaseTool)
├── HostedWebSearchTool(BaseTool)
├── HostedImageGenerationTool(BaseTool)
├── HostedMCPTool(BaseTool)
└── HostedFileSearchTool(BaseTool)
```

### ToolProtocol

```python
@runtime_checkable
class ToolProtocol(Protocol):
    name: str
    description: str | None
    additional_properties: dict[str, Any]
```

### FunctionTool

Core local function wrapper. Key attributes:

- `func: Callable` — the wrapped Python function
- `input_model: type[BaseModel]` — Pydantic model for argument validation
- `approval_mode: Literal["always_require"] | None` — whether approval is required
- `declaration_only: bool` — if True, function is not auto-invoked
- `max_invocations: int | None` — invocation cap
- `invocation_count: int` — current count
- `max_invocation_exceptions: int | None` — max allowed errors

**Key methods**:

- `invoke(arguments: BaseModel, tool_call_id: str, **kwargs)` — validates, executes (async or sync), records OTel metrics
- `parameters()` — returns cached JSON Schema from Pydantic model
- `to_json_schema_spec()` — returns `{"type": "function", "function": {"name", "description", "parameters", "strict"}}` for OpenAI API
- `_resolve_input_model()` — creates Pydantic model from function signature if not provided

**Descriptor protocol**: `__get__` enables FunctionTool to work as bound method descriptors on classes.

### `@tool` Decorator

Converts Python functions into `FunctionTool` instances:

```python
@tool
def get_weather(location: Annotated[str, "The city name"]) -> str:
    return f"Weather in {location}: sunny"
```

- Uses `Annotated[type, "description"]` for parameter descriptions
- Introspects function signature via `inspect.signature()`
- Creates Pydantic model via `pydantic.create_model()` with `Field(description=...)` for annotated params
- Supports both `@tool` (bare) and `@tool(name="custom_name", description="...")` (parameterized) forms

### `_create_input_model_from_func()`

Builds a Pydantic model from any Python function signature:

1. Iterates parameters (skipping `self`, `cls`, `**kwargs`)
2. Extracts `Annotated` metadata for descriptions
3. Maps default values to Pydantic `Field(default=...)`
4. Parameters without defaults become required
5. Calls `pydantic.create_model(func_name, **fields)`

### `_build_pydantic_model_from_json_schema()`

Reconstructs a Pydantic model from a JSON Schema dict:

- Handles `$ref` resolution via `$defs`
- Nested `object` types become nested Pydantic models
- `array` types with `items` become `list[T]`
- `const` and `enum` become `Literal` types
- `oneOf` with `discriminator` property generates discriminated unions
- Required vs. optional fields respected

### Hosted Tool Classes

Marker/configuration classes for service-managed tools (not locally executed):

| Class | Purpose | Key Fields |
|---|---|---|
| `HostedCodeInterpreterTool` | Server-side code execution | `file_ids`, `type="code_interpreter"` |
| `HostedWebSearchTool` | Server-side web search | `type="web_search"` |
| `HostedImageGenerationTool` | Server-side image generation | `type="image_generation"` |
| `HostedMCPTool` | Server-side MCP tool access | `server_label`, `server_url`, `type="mcp"` |
| `HostedFileSearchTool` | Server-side file search | `vector_store_ids`, `type="file_search"` |

### Function Invocation Configuration

```python
class FunctionInvocationConfiguration:
    enabled: bool = True
    max_iterations: int = 40
    max_consecutive_errors_per_request: int = 3
    terminate_on_unknown_calls: bool = False
    additional_tools: list[BaseTool] | None = None
    include_detailed_errors: bool = False
```

Controls the auto-invoke loop behavior when the chat client receives function call responses from the model.

### Auto-Invocation Flow (`_handle_function_calls_response` / `_handle_function_calls_streaming_response`)

Applied via `@use_function_invocation` class decorator on chat clients:

1. Model returns response with `function_call` content
2. Framework extracts function calls, matches to tool map
3. Checks for `approval_mode="always_require"` → returns `function_approval_request` contents instead
4. Checks for `declaration_only` → returns function calls as-is for external handling
5. Executes approved functions concurrently via `asyncio.gather()`
6. Wraps results as `function_result` content
7. Appends to message history, loops back to model
8. Repeats up to `max_iterations` or until model stops calling tools
9. Failsafe: after max iterations, sets `tool_choice="none"` for final call

### Approval Workflow

1. Tool marked with `approval_mode="always_require"`
2. First invocation returns `function_approval_request` content to caller
3. Caller creates `function_approval_response` with `approved=True/False`
4. On next call, framework processes responses:
   - Approved → execute function, replace with result
   - Rejected → replace with error result "Tool call invocation was rejected by user."

---

## 5. Agent Lifecycle

### Agent Hierarchy

**File**: [_agents.py](python/packages/core/agent_framework/_agents.py)

```
AgentProtocol (runtime_checkable Protocol)
├── BaseAgent(SerializationMixin)
│   └── ChatAgent(BaseAgent, Generic[TOptions_co])
```

### AgentProtocol

```python
@runtime_checkable
class AgentProtocol(Protocol):
    id: str
    name: str
    description: str | None
    async def run(self, messages, *, thread, **kwargs) -> AgentResponse: ...
    def run_stream(self, messages, *, thread, **kwargs) -> AsyncIterable[AgentResponseUpdate]: ...
    async def get_new_thread(self, **kwargs) -> AgentThread: ...
```

### BaseAgent

- `id`: Auto-generated UUID
- `name`: Required string
- `description`: Optional
- `context_provider`: Optional `ContextProvider` for dynamic context injection
- `middleware`: Optional list of `Middleware` instances

**Key methods**:

- `as_tool(description, instructions, result_parser)` — wraps agent as a `FunctionTool` for multi-agent orchestration
- `get_new_thread(**kwargs)` — creates new `AgentThread` (with optional custom `ChatMessageStore`)
- `deserialize_thread(data, dependencies)` — reconstructs thread from serialized dict
- `_notify_thread_of_new_messages(thread, messages)` — updates thread message store after run

### ChatAgent

Primary agent implementation. Decorated with `@use_agent_middleware` and `@use_agent_instrumentation`.

**Constructor**:

```python
ChatAgent(
    chat_client: ChatClientProtocol,
    instructions: str | None = None,
    tools: list[ToolProtocol | Callable] | None = None,
    default_options: dict[str, Any] | None = None,
    chat_message_store_factory: Callable[[], ChatMessageStoreProtocol] | None = None,
    context_provider: ContextProvider | None = None,
    middleware: list[Middleware] | None = None,
    **kwargs,
)
```

Separates MCP tools from regular tools at init time. Regular tools go into `default_options["tools"]`.

### `run()` Method Flow

1. **Normalize messages**: Convert string/list input to `list[ChatMessage]`
2. **Prepare thread**: `_prepare_thread_and_messages()` assembles:
   - Thread's stored messages
   - Context provider data (`context_provider.invoking()`)
   - Input messages
   - System instructions prepended
3. **Merge options**: Combine `default_options` with call-level `options`
4. **Connect MCP tools**: `async with` each MCPTool, load remote tools into options
5. **Call chat client**: `chat_client.get_response(messages, options=options, **kwargs)`
6. **Update thread**: `_notify_thread_of_new_messages()` adds response messages to thread store
7. **Invoke context provider**: `context_provider.invoked()` for post-run hook
8. **Return**: Wrap `ChatResponse` in `AgentResponse`

### `run_stream()` Method Flow

Similar to `run()` but:

1. Uses `chat_client.get_streaming_response()` → `AsyncIterable[ChatResponseUpdate]`
2. Maps each update to `AgentResponseUpdate` (adds `agent_id`, `agent_name`)
3. Yields each update to caller
4. After exhaustion, assembles `ChatResponse.from_chat_response_updates()` for thread notification
5. Yields final `AgentResponseUpdate` with `is_complete=True`

### Thread Management

**File**: [_threads.py](python/packages/core/agent_framework/_threads.py)

**AgentThread** supports two modes:

1. **Service-managed**: Has `service_thread_id` — messages stored server-side (e.g., OpenAI Assistants)
2. **Local**: Has `message_store` (in-memory `ChatMessageStore`) — messages stored in-process

**ChatMessageStore** (default implementation): Simple `list[ChatMessage]` wrapper with:

- `list_messages()` → returns stored messages
- `add_messages(messages)` → appends to list
- `serialize()` / `deserialize()` — dict-based persistence

**AgentThread serialization**: Supports `serialize()` → dict with thread state, `deserialize()` → reconstruct.

### Context Provider

**File**: [_memory.py](python/packages/core/agent_framework/_memory.py)

```python
class Context:
    instructions: str | None
    messages: list[ChatMessage]
    tools: list[ToolProtocol | Callable]

class ContextProvider(ABC):
    async def thread_created(self, thread: AgentThread) -> None: ...
    async def invoked(self, agent, thread, messages) -> None: ...
    @abstractmethod
    async def invoking(self, agent, thread, messages) -> Context: ...
```

- `invoking()` is called before each agent run; returns `Context` with additional instructions, messages, and tools to inject
- `invoked()` is called after each run for post-processing (e.g., memory updates)
- `thread_created()` is called when a new thread is created

### `as_mcp_server()`

ChatAgent can expose itself as an MCP server:

```python
server = agent.as_mcp_server(server_name="my-agent")
```

Creates an MCP `Server` with a single tool (`agent.name`) that invokes `agent.run()` and returns text results. Enables agent-to-agent communication via MCP protocol.

### `as_tool()`

BaseAgent can wrap itself as a `FunctionTool`:

```python
agent_tool = agent.as_tool(description="Research agent", instructions="Do research")
```

Creates a FunctionTool with a single `request: str` parameter that calls `agent.run()` and returns text output. Enables multi-agent orchestration where agents invoke other agents as tools.

---

## 6. Middleware System

### Architecture Overview

**File**: [_middleware.py](python/packages/core/agent_framework/_middleware.py)

Three independent middleware levels, each with its own context and pipeline:

| Level | Context Class | Applied To |
|---|---|---|
| Agent | `AgentRunContext` | `agent.run()` / `agent.run_stream()` |
| Function | `FunctionInvocationContext` | Tool/function execution |
| Chat | `ChatContext` | `chat_client.get_response()` / `get_streaming_response()` |

### Context Classes

All contexts share common fields: `metadata`, `result`, `terminate`, `kwargs`.

#### AgentRunContext

```python
class AgentRunContext:
    agent: AgentProtocol
    messages: list[ChatMessage]
    thread: AgentThread | None
    is_streaming: bool
    metadata: dict[str, Any]
    result: AgentResponse | None
    terminate: bool  # Set True to stop pipeline
    kwargs: dict[str, Any]
```

#### FunctionInvocationContext

```python
class FunctionInvocationContext:
    function: FunctionTool
    arguments: BaseModel
    metadata: dict[str, Any]
    result: Any
    terminate: bool  # Set True to stop tool loop
    kwargs: dict[str, Any]
```

#### ChatContext

```python
class ChatContext:
    chat_client: ChatClientProtocol
    messages: list[ChatMessage]
    options: dict[str, Any] | None
    is_streaming: bool
    metadata: dict[str, Any]
    result: ChatResponse | None
    terminate: bool  # Set True to skip execution
    kwargs: dict[str, Any]
```

### Middleware Protocols

**Class-based** (ABC with `process` method):

```python
class AgentMiddleware(ABC):
    async def process(self, context: AgentRunContext, next: Callable) -> AgentResponse: ...

class FunctionMiddleware(ABC):
    async def process(self, context: FunctionInvocationContext, next: Callable) -> Any: ...

class ChatMiddleware(ABC):
    async def process(self, context: ChatContext, next: Callable) -> ChatResponse: ...
```

**Function-based** (callables decorated with type markers):

```python
@agent_middleware
async def my_agent_middleware(context: AgentRunContext, next: Callable):
    # pre-processing
    result = await next(context)
    # post-processing
    return result

@function_middleware
async def my_function_middleware(context: FunctionInvocationContext, next: Callable):
    result = await next(context)
    return result

@chat_middleware
async def my_chat_middleware(context: ChatContext, next: Callable):
    result = await next(context)
    return result
```

### Type Union

```python
Middleware = Union[
    AgentMiddleware, FunctionMiddleware, ChatMiddleware,
    AgentMiddlewareCallable, FunctionMiddlewareCallable, ChatMiddlewareCallable
]
```

All middleware types can be mixed in a single list; the framework categorizes them automatically.

### Middleware Pipelines

Three pipeline implementations, all inheriting from `BaseMiddlewarePipeline`:

- **`AgentMiddlewarePipeline`**: `execute()` / `execute_stream()` for agent middleware chains
- **`FunctionMiddlewarePipeline`**: `execute()` for function middleware chains
- **`ChatMiddlewarePipeline`**: `execute()` / `execute_stream()` for chat middleware chains

Pipeline execution creates a handler chain where each middleware wraps the next, onion-style.

### Middleware Type Detection

`_determine_middleware_type()` resolves type by:

1. Checking for `_middleware_type` attribute (set by decorators)
2. Inspecting first parameter type annotation (`AgentRunContext`, `FunctionInvocationContext`, `ChatContext`)
3. Both must agree if both present; raises `MiddlewareException` on mismatch

### Middleware Categorization

`categorize_middleware(*sources)` separates mixed middleware lists into `MiddlewareDict`:

```python
class MiddlewareDict(TypedDict):
    agent: list[AgentMiddleware | AgentMiddlewareCallable]
    function: list[FunctionMiddleware | FunctionMiddlewareCallable]
    chat: list[ChatMiddleware | ChatMiddlewareCallable]
```

### Class Decorators

- `@use_agent_middleware` — wraps `run()`/`run_stream()` with agent middleware pipeline execution. Applied to `ChatAgent`.
- `@use_chat_middleware` — wraps `get_response()`/`get_streaming_response()` with chat middleware pipeline. Applied to `BaseChatClient`.
- `@use_function_invocation` — wraps `get_response()`/`get_streaming_response()` with auto-invoke loop. Applied to chat client implementations.

### Middleware Application Points

Middleware can be specified at:

1. **Instance level**: `ChatAgent(middleware=[...])` or `BaseChatClient(middleware=[...])`
2. **Call level**: `agent.run(messages, middleware=[...])` or `client.get_response(messages, middleware=[...])`
3. Both are merged, with instance-level executing first

### Terminate Signal

Setting `context.terminate = True` in any middleware:

- **AgentMiddleware**: Stops the agent pipeline; returns current result
- **FunctionMiddleware**: Stops the tool auto-invoke loop; returns result without additional LLM calls
- **ChatMiddleware**: Skips actual chat client execution; returns context result

---

## 7. MCP (Model Context Protocol) Integration

**File**: [_mcp.py](python/packages/core/agent_framework/_mcp.py)

### MCPTool Hierarchy

```
MCPTool (base)
├── MCPStdioTool — stdio transport (subprocess)
├── MCPStreamableHTTPTool — HTTP/SSE transport
└── MCPWebsocketTool — WebSocket transport
```

### MCPTool Base

Manages MCP `ClientSession` and `AsyncExitStack`. Key operations:

- `connect()` — establishes transport, initializes session, loads tools/prompts
- `load_tools()` — converts MCP tool definitions to `FunctionTool` instances with auto-generated Pydantic models
- `load_prompts()` — converts MCP prompt definitions to `FunctionTool` instances
- `call_tool(name, arguments)` — invokes MCP tool with auto-reconnect on failure
- `get_prompt(name, arguments)` — retrieves MCP prompt
- `sampling_callback()` — handles MCP sampling requests using configured chat_client

### Content Conversion

Bidirectional conversion between framework Content and MCP SDK content types:

- `_parse_content_from_mcp()`: MCP → Content (TextContent→text, ImageContent→data, ResourceLink→uri, etc.)
- `_prepare_content_for_mcp()`: Content → MCP (text→TextContent, data→ImageContent, etc.)

### Transport Implementations

| Class | Transport | Key Config |
|---|---|---|
| `MCPStdioTool` | stdio (subprocess) | `command`, `args`, `env`, `StdioServerParameters` |
| `MCPStreamableHTTPTool` | HTTP/SSE | `url`, optional `httpx.AsyncClient` |
| `MCPWebsocketTool` | WebSocket | `url` |

All share constructor pattern: `load_tools`, `parse_tool_results`, `load_prompts`, `parse_prompt_results`, `approval_mode`, `allowed_tools`.

---

## 8. Serialization Framework

**File**: [_serialization.py](python/packages/core/agent_framework/_serialization.py)

### SerializationMixin

Applied to `BaseTool`, `BaseAgent`, `BaseChatClient`, and context objects.

**Key features**:

- `to_dict(exclude=None, exclude_none=True)` — recursive serialization with type discriminator
- `from_dict(data, dependencies=None)` — reconstruction with dependency injection
- `to_json()` / `from_json()` — convenience JSON wrappers
- `DEFAULT_EXCLUDE: set[str]` — fields always excluded from serialization
- `INJECTABLE: set[str]` — fields excluded from serialization but injected during deserialization

**Dependency injection patterns**:

1. **Simple**: `{"type_id": {"param": value}}`
2. **Dict merge**: `{"type_id": {"dict_param": {"key": value}}}`
3. **Instance-specific**: `{"type_id": {"field:value": {"param": injected_value}}}`

**Type identifier resolution** (`_get_type_identifier`):

1. `value["type"]` if present in dict
2. Instance `type` attribute
3. Class `TYPE` attribute
4. Fallback: CamelCase → snake_case conversion

---

## 9. Exception Hierarchy

**File**: [exceptions.py](python/packages/core/agent_framework/exceptions.py)

```
AgentFrameworkException (base, auto-logs at debug level)
├── AgentException
│   ├── AgentExecutionException
│   ├── AgentInitializationError
│   └── AgentThreadException
├── ChatClientException
│   └── ChatClientInitializationError
├── ServiceException
│   ├── ServiceInitializationError
│   └── ServiceResponseException
│       ├── ServiceContentFilterException
│       ├── ServiceInvalidAuthError
│       ├── ServiceInvalidExecutionSettingsError
│       ├── ServiceInvalidRequestError
│       └── ServiceInvalidResponseError
├── ToolException
│   └── ToolExecutionException
├── AdditionItemMismatch
├── MiddlewareException
└── ContentError
```

---

## 10. Key Design Patterns

### Unified Content over Discriminated Unions

Single Content class with optional fields; type field discriminates. Favors simplicity and serializability over type safety of union types.

### Protocol-First Design

`AgentProtocol`, `ChatClientProtocol`, `ToolProtocol`, `ChatMessageStoreProtocol` — all use `@runtime_checkable` Protocol classes. Enables structural typing without inheritance.

### Decorator-Based Extension

`@use_function_invocation`, `@use_agent_middleware`, `@use_chat_middleware`, `@use_agent_instrumentation` — class decorators wrap methods to add cross-cutting concerns (tool execution, middleware, telemetry).

### Async-First with Sync Fallback

All I/O methods are async. `FunctionTool.invoke()` handles sync functions via `asyncio.to_thread()` (or direct call if not a coroutine).

### Factory Classmethods

Content uses factory classmethods (`from_text`, `from_function_call`, etc.) instead of constructor overloading. Ensures `type` field is always set correctly.

### Context Provider Pattern

`ContextProvider` injects dynamic data (instructions, messages, tools) at each agent invocation. Separates context management from agent logic.
