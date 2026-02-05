# Python Middleware Deep Dive

Research completed: 2026-02-03

## Overview

This document provides a deep analysis of the Python middleware implementation to ensure the Go port has full parity with the Python Agent Framework.

---

## 1. Middleware Context Object Structures

### AgentRunContext

**Source**: [_middleware.py#L56-L136](python/packages/core/agent_framework/_middleware.py#L56-L136)

```python
class AgentRunContext(SerializationMixin):
    INJECTABLE: ClassVar[set[str]] = {"agent", "thread", "result"}

    def __init__(
        self,
        agent: "AgentProtocol",
        messages: list[ChatMessage],
        thread: "AgentThread | None" = None,
        is_streaming: bool = False,
        metadata: dict[str, Any] | None = None,
        result: AgentResponse | AsyncIterable[AgentResponseUpdate] | None = None,
        terminate: bool = False,
        kwargs: dict[str, Any] | None = None,
    ) -> None:
```

**Fields**:

| Field | Type | Description |
|-------|------|-------------|
| `agent` | `AgentProtocol` | The agent being invoked |
| `messages` | `list[ChatMessage]` | Input messages |
| `thread` | `AgentThread \| None` | Session/thread for this invocation |
| `is_streaming` | `bool` | Whether streaming mode |
| `metadata` | `dict[str, Any]` | Middleware-to-middleware data sharing |
| `result` | `AgentResponse \| AsyncIterable[AgentResponseUpdate] \| None` | Result (set after next() or for override) |
| `terminate` | `bool` | Flag to stop pipeline execution |
| `kwargs` | `dict[str, Any]` | Additional runtime kwargs |

**Go Parity Status**: ✅ Exists as `AgentContext` in Go. Minor differences:
- Go uses `Response` and `Stream` as separate fields
- Python uses polymorphic `result` field (AgentResponse or AsyncIterable)
- Go lacks `terminate` flag (uses error return instead)

---

### FunctionInvocationContext

**Source**: [_middleware.py#L139-L197](python/packages/core/agent_framework/_middleware.py#L139-L197)

```python
class FunctionInvocationContext(SerializationMixin):
    INJECTABLE: ClassVar[set[str]] = {"function", "arguments", "result"}

    def __init__(
        self,
        function: "FunctionTool[Any, Any]",
        arguments: "BaseModel",
        metadata: dict[str, Any] | None = None,
        result: Any = None,
        terminate: bool = False,
        kwargs: dict[str, Any] | None = None,
    ) -> None:
```

**Fields**:

| Field | Type | Description |
|-------|------|-------------|
| `function` | `FunctionTool` | The function/tool being invoked |
| `arguments` | `BaseModel` | Validated Pydantic arguments model |
| `metadata` | `dict[str, Any]` | Middleware data sharing |
| `result` | `Any` | Function result (post-execution or override) |
| `terminate` | `bool` | Stop pipeline flag |
| `kwargs` | `dict[str, Any]` | Runtime kwargs |

**Go Parity Status**: ⚠️ Partial. Go `FunctionContext` uses:
- `FunctionName: string` (not full tool object)
- `Arguments: json.RawMessage` (not validated model)
- Has `Error` field instead of using `terminate`

---

### ChatContext

**Source**: [_middleware.py#L200-L269](python/packages/core/agent_framework/_middleware.py#L200-L269)

```python
class ChatContext(SerializationMixin):
    INJECTABLE: ClassVar[set[str]] = {"chat_client", "result"}

    def __init__(
        self,
        chat_client: "ChatClientProtocol",
        messages: "MutableSequence[ChatMessage]",
        options: Mapping[str, Any] | None,
        is_streaming: bool = False,
        metadata: dict[str, Any] | None = None,
        result: "ChatResponse | AsyncIterable[ChatResponseUpdate] | None" = None,
        terminate: bool = False,
        kwargs: dict[str, Any] | None = None,
    ) -> None:
```

**Fields**:

| Field | Type | Description |
|-------|------|-------------|
| `chat_client` | `ChatClientProtocol` | The chat client instance |
| `messages` | `MutableSequence[ChatMessage]` | Mutable message list (can modify) |
| `options` | `Mapping[str, Any]` | Chat request options |
| `is_streaming` | `bool` | Streaming mode flag |
| `metadata` | `dict[str, Any]` | Middleware data sharing |
| `result` | `ChatResponse \| AsyncIterable \| None` | Result storage |
| `terminate` | `bool` | Stop execution flag |
| `kwargs` | `dict[str, Any]` | Extra kwargs |

**Go Parity Status**: ✅ Exists as `ChatContext` with similar fields. Go uses `ClientMetadata` struct instead of full client reference.

---

## 2. Stream Handling Patterns

### Non-Streaming vs Streaming Result Storage

**Key Pattern**: Python uses a **polymorphic `result` field** that holds:
- Non-streaming: `AgentResponse` / `ChatResponse`
- Streaming: `AsyncIterable[AgentResponseUpdate]` / `AsyncIterable[ChatResponseUpdate]`

**Source**: [_middleware.py#L802-L849](python/packages/core/agent_framework/_middleware.py#L802-L849)

```python
async def execute_stream(
    self,
    agent: "AgentProtocol",
    messages: list[ChatMessage],
    context: AgentRunContext,
    final_handler: Callable[[AgentRunContext], AsyncIterable[AgentResponseUpdate]],
) -> AsyncIterable[AgentResponseUpdate]:
    # Update context with agent and messages
    context.agent = agent
    context.messages = messages
    context.is_streaming = True

    if not self._middleware:
        async for update in final_handler(context):
            yield update
        return

    # Store the final result
    result_container: dict[str, AsyncIterable[AgentResponseUpdate] | None] = {"result_stream": None}

    first_handler = self._create_streaming_handler_chain(final_handler, result_container, "result_stream")
    await first_handler(context)

    # Yield from the result stream in result container or overridden result
    if context.result is not None and hasattr(context.result, "__aiter__"):
        async for update in context.result:
            yield update
        return

    result_stream = result_container["result_stream"]
    if result_stream is None:
        return

    async for update in result_stream:
        yield update
```

### Streaming Handler Chain Creation

**Source**: [_middleware.py#L698-L736](python/packages/core/agent_framework/_middleware.py#L698-L736)

```python
def _create_streaming_handler_chain(
    self,
    final_handler: Callable[[Any], Any],
    result_container: dict[str, Any],
    result_key: str = "result_stream",
) -> Callable[[Any], Awaitable[None]]:

    def create_next_handler(index: int) -> Callable[[Any], Awaitable[None]]:
        if index >= len(self._middleware):

            async def final_wrapper(c: Any) -> None:
                # If terminate was set, skip execution
                if c.terminate:
                    return

                # Execute actual handler
                try:
                    result = await final_handler(c)
                except TypeError:
                    # Handle non-awaitable case (generator functions)
                    result = final_handler(c)
                result_container[result_key] = result
                c.result = result

            return final_wrapper

        middleware = self._middleware[index]
        next_handler = create_next_handler(index + 1)

        async def current_handler(c: Any) -> None:
            await middleware.process(c, next_handler)
            # If terminate is set, don't continue the pipeline
            if c.terminate:
                return

        return current_handler

    return create_next_handler(0)
```

**Key Insights for Go Port**:
1. Streaming uses a `result_container` dict pattern to capture stream reference
2. The `terminate` flag is checked at each middleware step
3. Stream result can be overridden by middleware setting `context.result`
4. Final handler may not be awaitable (generator function) - needs TypeError handling

**Go Gap**: Go returns channels directly, Python uses AsyncIterable pattern. Need to ensure Go properly supports middleware override of streams.

---

## 3. Middleware Chaining Logic

### Pipeline Execution Order

**Source**: [_middleware.py#L661-L693](python/packages/core/agent_framework/_middleware.py#L661-L693)

```python
def _create_handler_chain(
    self,
    final_handler: Callable[[Any], Awaitable[Any]],
    result_container: dict[str, Any],
    result_key: str = "result",
) -> Callable[[Any], Awaitable[None]]:

    def create_next_handler(index: int) -> Callable[[Any], Awaitable[None]]:
        if index >= len(self._middleware):

            async def final_wrapper(c: Any) -> None:
                result = await final_handler(c)
                result_container[result_key] = result
                c.result = result

            return final_wrapper

        middleware = self._middleware[index]
        next_handler = create_next_handler(index + 1)

        async def current_handler(c: Any) -> None:
            await middleware.process(c, next_handler)

        return current_handler

    return create_next_handler(0)
```

**Execution Pattern**:
1. Middleware are executed in registration order (first registered = first to execute)
2. Each middleware receives `context` and `next` function
3. Calling `await next(context)` invokes the next middleware in chain
4. NOT calling `next()` short-circuits the pipeline
5. Code after `await next(context)` runs after downstream middleware complete

**Go Parity**: ✅ Go `ChainAgentMiddleware` follows same pattern - builds chain from last to first, executes first to last.

---

### Middleware Categorization System

**Source**: [_middleware.py#L1482-L1534](python/packages/core/agent_framework/_middleware.py#L1482-L1534)

Python has a sophisticated middleware type detection system:

```python
def categorize_middleware(
    *middleware_sources: Middleware | None,
) -> MiddlewareDict:
    result: MiddlewareDict = {"agent": [], "function": [], "chat": []}

    for source in middleware_sources:
        if source:
            if isinstance(source, list):
                all_middleware.extend(source)
            else:
                all_middleware.append(source)

    for middleware in all_middleware:
        if isinstance(middleware, AgentMiddleware):
            result["agent"].append(middleware)
        elif isinstance(middleware, FunctionMiddleware):
            result["function"].append(middleware)
        elif isinstance(middleware, ChatMiddleware):
            result["chat"].append(middleware)
        elif callable(middleware):
            middleware_type = _determine_middleware_type(middleware)
            # Categorize based on type...
```

**Type Detection** ([_middleware.py#L1082-L1143](python/packages/core/agent_framework/_middleware.py#L1082-L1143)):
1. Check for `_middleware_type` marker attribute (from decorators)
2. Inspect function signature for context parameter type annotation
3. Must match - decorator type and annotation type must agree
4. Throws `MiddlewareException` if type cannot be determined

**Go Gap**: ❌ Go requires explicit middleware interfaces. No automatic detection from function signatures. This is acceptable - Go's type system makes this unnecessary.

---

## 4. AgentMiddleware Processing (Streaming vs Non-Streaming)

### Non-Streaming Execute

**Source**: [_middleware.py#L755-L800](python/packages/core/agent_framework/_middleware.py#L755-L800)

```python
async def execute(
    self,
    agent: "AgentProtocol",
    messages: list[ChatMessage],
    context: AgentRunContext,
    final_handler: Callable[[AgentRunContext], Awaitable[AgentResponse]],
) -> AgentResponse | None:
    context.agent = agent
    context.messages = messages
    context.is_streaming = False

    if not self._middleware:
        return await final_handler(context)

    result_container: dict[str, AgentResponse | None] = {"result": None}

    async def agent_final_handler(c: AgentRunContext) -> AgentResponse:
        if c.terminate:
            if c.result is not None and isinstance(c.result, AgentResponse):
                return c.result
            return AgentResponse()
        return await final_handler(c)

    first_handler = self._create_handler_chain(agent_final_handler, result_container, "result")
    await first_handler(context)

    if context.result is not None and isinstance(context.result, AgentResponse):
        return context.result

    response = result_container.get("result")
    if response is None:
        return AgentResponse()
    return response
```

**Key Behaviors**:
1. If no middleware, bypass and call handler directly
2. `terminate=True` returns context.result or empty AgentResponse
3. Middleware can override by setting `context.result` after calling `next()`
4. Returns from result_container if context.result not set

---

### Streaming Execute

**Source**: [_middleware.py#L802-L849](python/packages/core/agent_framework/_middleware.py#L802-L849)

Key differences from non-streaming:
1. Sets `context.is_streaming = True`
2. Final handler returns `AsyncIterable[AgentResponseUpdate]`
3. Result is yielded via async for loop
4. Override detection: `hasattr(context.result, "__aiter__")`

---

## 5. FunctionMiddleware Integration with Tool Invocation

### Integration Point

**Source**: [_tools.py#L1597-L1620](python/packages/core/agent_framework/_tools.py#L1597-L1620)

```python
# Execute through middleware pipeline if available
from ._middleware import FunctionInvocationContext

middleware_context = FunctionInvocationContext(
    function=tool,
    arguments=args,
    kwargs=runtime_kwargs.copy(),
)

async def final_function_handler(context_obj: Any) -> Any:
    return await tool.invoke(
        arguments=context_obj.arguments,
        tool_call_id=function_call_content.call_id,
        **context_obj.kwargs if getattr(tool, "_forward_runtime_kwargs", False) else {},
    )

try:
    function_result = await middleware_pipeline.execute(
        function=tool,
        arguments=args,
        context=middleware_context,
        final_handler=final_function_handler,
    )
    return FunctionExecutionResult(
        content=Content.from_function_result(...),
        terminate=middleware_context.terminate,
    )
```

### Pipeline Propagation

**Source**: [_middleware.py#L1227](python/packages/core/agent_framework/_middleware.py#L1227)

```python
# Add function middleware pipeline to kwargs if available
if function_pipeline.has_middlewares:
    kwargs["_function_middleware_pipeline"] = function_pipeline
```

The function middleware pipeline is:
1. Created during agent middleware building
2. Passed via `kwargs["_function_middleware_pipeline"]`
3. Extracted in `_execute_function` and applied to each tool call
4. `terminate` flag from function context is returned in `FunctionExecutionResult`

**Go Gap**: ⚠️ Need to verify Go passes function middleware through to tool execution.

---

## 6. ChatMiddleware Integration

### Chat Client Decoration

**Source**: [_middleware.py#L1313-L1475](python/packages/core/agent_framework/_middleware.py#L1313-L1475)

Python uses `@use_chat_middleware` decorator on chat client classes:

```python
@use_chat_middleware
class BaseChatClient:
    ...
```

This wraps `get_response()` and `get_streaming_response()` methods to:
1. Extract middleware from call-level and instance-level
2. Categorize into chat/function types
3. Create `ChatMiddlewarePipeline`
4. Build `ChatContext` with request data
5. Execute through pipeline

### ChatMiddleware Pipeline Execute

**Source**: [_middleware.py#L978-L1030](python/packages/core/agent_framework/_middleware.py#L978-L1030)

```python
async def execute(
    self,
    chat_client: "ChatClientProtocol",
    messages: "MutableSequence[ChatMessage]",
    options: Mapping[str, Any] | None,
    context: ChatContext,
    final_handler: Callable[[ChatContext], Awaitable["ChatResponse"]],
    **kwargs: Any,
) -> "ChatResponse":
    context.chat_client = chat_client
    context.messages = messages
    if options:
        context.options = options

    if not self._middleware:
        return await final_handler(context)

    # ... middleware chain execution
```

**Go Parity**: ⚠️ Go has `ChatMiddleware` interface and `ChainChatMiddleware` but integration with chat clients needs verification.

---

## 7. Terminate Flag Semantics

### Usage Pattern

The `terminate` flag provides graceful short-circuiting:

```python
class SecurityMiddleware(AgentMiddleware):
    async def process(self, context: AgentRunContext, next):
        if contains_sensitive_data(context.messages):
            context.result = AgentResponse(messages=[...])
            context.terminate = True
            return  # Don't call next()

        await next(context)
```

**Key Behaviors**:
1. `terminate=True` + not calling `next()` = complete short-circuit
2. `terminate=True` + calling `next()` = downstream executes, but pipeline stops after current middleware returns
3. Result must be set if terminating, otherwise empty response returned

**Go Gap**: ❌ Go uses error return for termination. No `terminate` flag on context. This is a semantic difference - Go idiom vs Python idiom.

---

## 8. Middleware Type Decorators

Python provides decorator markers for function-based middleware:

**Source**: [_middleware.py#L475-L558](python/packages/core/agent_framework/_middleware.py#L475-L558)

```python
@agent_middleware
async def my_middleware(context: AgentRunContext, next):
    ...

@function_middleware
async def my_fn_middleware(context: FunctionInvocationContext, next):
    ...

@chat_middleware
async def my_chat_middleware(context: ChatContext, next):
    ...
```

These add `_middleware_type` attribute for categorization.

**Go Note**: Not needed - Go uses explicit interface types.

---

## 9. Injectable Fields (INJECTABLE Pattern)

**Source**: Context class definitions

```python
class AgentRunContext:
    INJECTABLE: ClassVar[set[str]] = {"agent", "thread", "result"}

class FunctionInvocationContext:
    INJECTABLE: ClassVar[set[str]] = {"function", "arguments", "result"}

class ChatContext:
    INJECTABLE: ClassVar[set[str]] = {"chat_client", "result"}
```

This pattern enables serialization/deserialization for durable agents. Fields marked as INJECTABLE are handled specially during state persistence.

**Go Gap**: ❓ Unknown if Go needs this for durable agents.

---

## 10. Summary: Go Parity Gaps

### Must Have (Core Functionality)

| Feature | Python | Go | Gap |
|---------|--------|-----|-----|
| AgentMiddleware interface | ✅ | ✅ | None |
| FunctionMiddleware interface | ✅ | ✅ | None |
| ChatMiddleware interface | ✅ | ✅ | None |
| Middleware chaining | ✅ | ✅ | None |
| Streaming support | ✅ | ✅ | Minor (channel vs AsyncIterable) |
| Terminate flag | ✅ | ❌ | Uses error return instead |
| Result override | ✅ | ✅ | None |
| Metadata dict | ✅ | ✅ | None |

### Nice to Have (Enhanced Features)

| Feature | Python | Go | Gap |
|---------|--------|-----|-----|
| Auto type detection | ✅ | ❌ | Not needed (Go types) |
| Decorator markers | ✅ | ❌ | Not needed |
| INJECTABLE pattern | ✅ | ❓ | Unknown |
| Full function tool in context | ✅ | ⚠️ | Go has name only |
| Validated arguments model | ✅ | ⚠️ | Go has raw JSON |

---

## 11. Recommended Go Improvements

1. **Add `Terminate` field to context structs** - Provides cleaner short-circuit semantics matching Python
2. **Enhance `FunctionContext`** - Include full tool reference if available
3. **Verify function middleware propagation** - Ensure `_function_middleware_pipeline` equivalent exists
4. **Verify chat middleware integration** - Ensure chat clients properly apply middleware chain
5. **Document streaming override pattern** - Show how middleware can provide custom stream

---

## Files Examined

- [python/packages/core/agent_framework/_middleware.py](python/packages/core/agent_framework/_middleware.py) - Full middleware module (1589 lines)
- [python/packages/core/agent_framework/_agents.py](python/packages/core/agent_framework/_agents.py) - Agent integration (1330 lines)
- [python/packages/core/agent_framework/_tools.py](python/packages/core/agent_framework/_tools.py) - Tool invocation with middleware
- [go/agent/middleware.go](go/agent/middleware.go) - Go middleware interfaces
- [go/agent/middleware_context.go](go/agent/middleware_context.go) - Go context structs
- [go/agent/middleware_agent.go](go/agent/middleware_agent.go) - Go MiddlewareAgent wrapper
- [go/agent/chain.go](go/agent/chain.go) - Go middleware chaining
