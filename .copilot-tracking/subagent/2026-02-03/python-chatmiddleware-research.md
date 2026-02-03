# Python ChatMiddleware Implementation Research

**Date**: 2026-02-03  
**Purpose**: Document the Python ChatMiddleware implementation for reference when implementing Go equivalents

---

## Overview

The Python `agent_framework` provides `ChatMiddleware` as part of a comprehensive middleware system that intercepts chat client requests. ChatMiddleware specifically targets the `get_response()` and `get_streaming_response()` methods of chat clients.

**Key Source Files**:
- [_middleware.py](../../../python/packages/core/agent_framework/_middleware.py) - Core middleware definitions
- [_clients.py](../../../python/packages/core/agent_framework/_clients.py) - Chat client integration
- [chat_middleware.py](../../../python/samples/getting_started/middleware/chat_middleware.py) - Sample usage

---

## ChatMiddleware Interface/Protocol Definition

### Abstract Base Class

```python
class ChatMiddleware(ABC):
    """Abstract base class for chat middleware that can intercept chat client requests."""

    @abstractmethod
    async def process(
        self,
        context: ChatContext,
        next: Callable[[ChatContext], Awaitable[None]],
    ) -> None:
        """Process a chat client request.

        Args:
            context: Chat invocation context containing chat client, messages, options, and metadata.
                    Use context.is_streaming to determine if this is a streaming call.
                    Middleware can set context.result to override execution, or observe
                    the actual execution result after calling next().
                    For non-streaming: ChatResponse
                    For streaming: AsyncIterable[ChatResponseUpdate]
            next: Function to call the next middleware or final chat execution.
                  Does not return anything - all data flows through the context.

        Note:
            Middleware should not return anything. All data manipulation should happen
            within the context object. Set context.result to override execution,
            or observe context.result after calling next() for actual results.
        """
        ...
```

### Callable Type Alias

Function-based middleware is also supported:

```python
ChatMiddlewareCallable = Callable[
    [ChatContext, Callable[[ChatContext], Awaitable[None]]],
    Awaitable[None]
]
```

### Decorator for Function-Based Middleware

```python
def chat_middleware(func: ChatMiddlewareCallable) -> ChatMiddlewareCallable:
    """Decorator to mark a function as chat middleware."""
    func._middleware_type: MiddlewareType = MiddlewareType.CHAT
    return func
```

---

## ChatContext Structure

```python
class ChatContext(SerializationMixin):
    """Context object for chat middleware invocations."""

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

### Field Descriptions

| Field | Type | Description |
|-------|------|-------------|
| `chat_client` | `ChatClientProtocol` | The chat client being invoked |
| `messages` | `MutableSequence[ChatMessage]` | Messages being sent to chat client (mutable) |
| `options` | `Mapping[str, Any] \| None` | Request options as dict (model_id, temperature, etc.) |
| `is_streaming` | `bool` | Whether this is a streaming invocation |
| `metadata` | `dict[str, Any]` | Shared data between middleware |
| `result` | `ChatResponse \| AsyncIterable[ChatResponseUpdate] \| None` | Execution result (set after `next()` or to override) |
| `terminate` | `bool` | Flag to stop pipeline execution |
| `kwargs` | `dict[str, Any]` | Additional keyword arguments |

---

## ChatMiddlewarePipeline Execution

### Non-Streaming Execution

```python
class ChatMiddlewarePipeline(BaseMiddlewarePipeline):
    """Executes chat middleware in a chain."""

    async def execute(
        self,
        chat_client: "ChatClientProtocol",
        messages: "MutableSequence[ChatMessage]",
        options: Mapping[str, Any] | None,
        context: ChatContext,
        final_handler: Callable[[ChatContext], Awaitable["ChatResponse"]],
        **kwargs: Any,
    ) -> "ChatResponse":
        """Execute the chat middleware pipeline."""
        # Update context with chat client, messages, and options
        context.chat_client = chat_client
        context.messages = messages
        if options:
            context.options = options

        if not self._middleware:
            return await final_handler(context)

        # Build handler chain and execute
        result_container: dict[str, Any] = {"result": None}

        async def chat_final_handler(c: ChatContext) -> "ChatResponse":
            if c.terminate:
                return c.result  # Return override result or None
            return await final_handler(c)

        first_handler = self._create_handler_chain(chat_final_handler, result_container, "result")
        await first_handler(context)

        if context.result is not None:
            return context.result
        return result_container["result"]
```

### Streaming Execution

```python
async def execute_stream(
    self,
    chat_client: "ChatClientProtocol",
    messages: "MutableSequence[ChatMessage]",
    options: Mapping[str, Any] | None,
    context: ChatContext,
    final_handler: Callable[[ChatContext], AsyncIterable["ChatResponseUpdate"]],
    **kwargs: Any,
) -> AsyncIterable["ChatResponseUpdate"]:
    """Execute the chat middleware pipeline for streaming."""
    context.chat_client = chat_client
    context.messages = messages
    if options:
        context.options = options
    context.is_streaming = True

    if not self._middleware:
        async for update in final_handler(context):
            yield update
        return

    result_container: dict[str, Any] = {"result_stream": None}
    first_handler = self._create_streaming_handler_chain(final_handler, result_container, "result_stream")
    await first_handler(context)

    # Yield from context.result (if overridden) or result_container
    if context.result is not None and hasattr(context.result, "__aiter__"):
        async for update in context.result:
            yield update
        return

    result_stream = result_container["result_stream"]
    if result_stream:
        async for update in result_stream:
            yield update
```

---

## Integration with Chat Client

### The `use_chat_middleware` Decorator

Applied to `BaseChatClient` to enable middleware support:

```python
@use_chat_middleware
class BaseChatClient(SerializationMixin, ABC, Generic[TOptions_co]):
    """Base class for chat clients."""
```

### Middleware-Enabled get_response

The decorator wraps `get_response()` to:

1. Extract call-level and instance-level middleware
2. Categorize by type (agent, function, chat)
3. Build `ChatMiddlewarePipeline` with chat middleware
4. Create `ChatContext` with prepared messages
5. Execute pipeline with final handler calling original method

```python
async def middleware_enabled_get_response(
    self: Any,
    messages: Any,
    *,
    options: Mapping[str, Any] | None = None,
    **kwargs: Any,
) -> Any:
    call_middleware = kwargs.pop("middleware", None)
    instance_middleware = getattr(self, "middleware", None)

    middleware = categorize_middleware(instance_middleware, call_middleware)
    chat_middleware_list = middleware["chat"]

    if not chat_middleware_list:
        return await original_get_response(self, messages, options=options, **kwargs)

    pipeline = ChatMiddlewarePipeline(chat_middleware_list)
    context = ChatContext(
        chat_client=self,
        messages=prepare_messages(messages),
        options=options,
        is_streaming=False,
        kwargs=kwargs,
    )

    async def final_handler(ctx: ChatContext) -> Any:
        return await original_get_response(
            self, list(ctx.messages), options=ctx.options, **ctx.kwargs
        )

    return await pipeline.execute(
        chat_client=self,
        messages=context.messages,
        options=options,
        context=context,
        final_handler=final_handler,
        **kwargs,
    )
```

### Middleware-Enabled get_streaming_response

Similar pattern but returns an async generator:

```python
def middleware_enabled_get_streaming_response(
    self: Any,
    messages: Any,
    *,
    options: dict[str, Any] | None = None,
    **kwargs: Any,
) -> Any:
    async def _stream_generator() -> Any:
        # Same categorization logic...
        pipeline = ChatMiddlewarePipeline(chat_middleware_list)
        context = ChatContext(..., is_streaming=True, ...)

        def final_handler(ctx: ChatContext) -> Any:
            return original_get_streaming_response(
                self, list(ctx.messages), options=ctx.options, **ctx.kwargs
            )

        async for update in pipeline.execute_stream(
            chat_client=self, messages=context.messages, options=options,
            context=context, final_handler=final_handler, **kwargs,
        ):
            yield update

    return _stream_generator()
```

---

## Middleware Registration Points

### 1. Instance-Level (Chat Client Constructor)

```python
client = SomeChatClient(
    model_id="gpt-4",
    middleware=[MyMiddleware(), another_middleware]
)
```

### 2. Call-Level (Per Request)

```python
response = await client.get_response(
    messages,
    middleware=[SpecificMiddleware()]
)
```

### 3. Agent-Level (Flows to Chat Client)

```python
agent = ChatAgent(
    chat_client=client,
    middleware=[AgentMiddleware(), ChatMiddleware()]  # Both types supported
)
```

Middleware passed through agent flows down to chat client calls via kwargs.

---

## Streaming Considerations

1. **Context Flag**: `context.is_streaming = True` indicates streaming mode
2. **Result Type**: For streaming, `context.result` should be `AsyncIterable[ChatResponseUpdate]`
3. **Execution Order**: Middleware runs before streaming begins, then stream is yielded
4. **Post-Processing**: Cannot easily post-process individual chunks in current design
5. **Terminate Behavior**: Setting `terminate = True` prevents final handler execution

### Important Note on Streaming Timing

In streaming execution, the middleware `process()` method completes before chunks are yielded:

```python
# Execution order in streaming:
# 1. middleware.process() called
# 2. await next(context) called
# 3. final_handler sets up stream (does NOT consume it yet)
# 4. middleware.process() returns
# 5. Pipeline yields from stream
```

This means post-`next()` code in middleware runs BEFORE streaming data is consumed.

---

## Example Implementations

### Class-Based Middleware

```python
class InputObserverMiddleware(ChatMiddleware):
    def __init__(self, replacement: str | None = None):
        self.replacement = replacement

    async def process(
        self,
        context: ChatContext,
        next: Callable[[ChatContext], Awaitable[None]],
    ) -> None:
        # Pre-processing: observe/modify messages
        for i, message in enumerate(context.messages):
            print(f"Message {i + 1}: {message.text}")

        # Optionally modify messages
        if self.replacement:
            for msg in context.messages:
                if msg.role == Role.USER:
                    msg.text = self.replacement

        # Continue pipeline
        await next(context)

        # Post-processing: observe result
        print(f"Result: {context.result}")
```

### Function-Based Middleware with Override

```python
@chat_middleware
async def security_middleware(
    context: ChatContext,
    next: Callable[[ChatContext], Awaitable[None]],
) -> None:
    blocked_terms = ["password", "secret", "api_key"]

    for message in context.messages:
        if message.text and any(term in message.text.lower() for term in blocked_terms):
            # Override response and terminate
            context.result = ChatResponse(
                messages=[ChatMessage(role=Role.ASSISTANT, text="Blocked for security.")]
            )
            context.terminate = True
            return

    await next(context)
```

### Real-World Example: Purview Policy Middleware

```python
class PurviewChatPolicyMiddleware(ChatMiddleware):
    def __init__(self, credential, settings, cache_provider=None):
        self._client = PurviewClient(credential, settings)
        self._processor = ScopedContentProcessor(self._client, settings, cache_provider)
        self._settings = settings

    async def process(
        self,
        context: ChatContext,
        next: Callable[[ChatContext], Awaitable[None]],
    ) -> None:
        # Pre-check: evaluate prompt against policy
        should_block, user_id = await self._processor.process_messages(
            context.messages, Activity.UPLOAD_TEXT
        )
        if should_block:
            context.result = ChatResponse(messages=[
                ChatMessage(role="system", text=self._settings.blocked_prompt_message)
            ])
            context.terminate = True
            return

        await next(context)

        # Post-check: evaluate response (non-streaming only)
        if context.result and not context.is_streaming:
            messages = getattr(context.result, "messages", None)
            if messages:
                should_block, _ = await self._processor.process_messages(
                    messages, Activity.UPLOAD_TEXT, user_id=user_id
                )
                if should_block:
                    context.result = ChatResponse(messages=[
                        ChatMessage(role="system", text=self._settings.blocked_response_message)
                    ])
```

---

## Middleware Type Detection

The framework uses type detection to categorize middleware:

```python
def _determine_middleware_type(middleware: Any) -> MiddlewareType:
    """Determine middleware type using decorator and/or parameter type annotation."""
    # 1. Check for decorator marker (_middleware_type attribute)
    decorator_type = getattr(middleware, "_middleware_type", None)

    # 2. Check parameter type annotation
    sig = inspect.signature(middleware)
    params = list(sig.parameters.values())
    first_param = params[0]
    if first_param.annotation.__name__ == "ChatContext":
        param_type = MiddlewareType.CHAT
    # ... similar for AgentRunContext, FunctionInvocationContext

    # 3. Validate decorator and annotation match if both present
    # 4. Return determined type or raise MiddlewareException
```

---

## Comparison with Go Implementation Goals

| Python Feature | Go Equivalent |
|----------------|---------------|
| `ChatMiddleware` ABC | `ChatMiddleware` interface |
| `ChatContext` class | `ChatContext` struct |
| `process(context, next)` | `Process(ctx context.Context, chatCtx *ChatContext, next NextHandler)` |
| `context.terminate = True` | `chatCtx.Terminate = true` |
| `context.result = override` | `chatCtx.Result = override` |
| Callable middleware | Handler functions |
| `@chat_middleware` decorator | Type registration pattern |
| `ChatMiddlewarePipeline` | `ChatMiddlewarePipeline` struct |

---

## Key Design Patterns

1. **Context Object Pattern**: All data flows through mutable context object
2. **Chain of Responsibility**: Middleware forms a chain via `next()` calls
3. **Decorator Pattern**: Wraps original methods transparently
4. **Type Categorization**: Automatic routing to correct pipeline
5. **Override Mechanism**: Set `result` + `terminate` to short-circuit

---

## References

- Source: [python/packages/core/agent_framework/_middleware.py](../../../python/packages/core/agent_framework/_middleware.py#L196-L283)
- Sample: [python/samples/getting_started/middleware/chat_middleware.py](../../../python/samples/getting_started/middleware/chat_middleware.py)
- Tests: [python/packages/core/tests/core/test_middleware.py](../../../python/packages/core/tests/core/test_middleware.py#L527)
- Purview: [python/packages/purview/agent_framework_purview/_middleware.py](../../../python/packages/purview/agent_framework_purview/_middleware.py#L106)
