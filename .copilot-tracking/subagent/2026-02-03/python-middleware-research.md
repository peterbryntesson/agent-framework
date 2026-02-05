# Python Middleware Research

**Date**: 2026-02-03  
**Focus**: Middleware patterns in Python agent framework implementation

---

## 1. Middleware Protocol Definitions

### 1.1 AgentMiddleware

**File**: [_middleware.py](../../../python/packages/core/agent_framework/_middleware.py#L283-L338)

```python
class AgentMiddleware(ABC):
    """Abstract base class for agent middleware that can intercept agent invocations."""

    @abstractmethod
    async def process(
        self,
        context: AgentRunContext,
        next: Callable[[AgentRunContext], Awaitable[None]],
    ) -> None:
        """Process an agent invocation.

        Args:
            context: Agent invocation context containing agent, messages, and metadata.
            next: Function to call the next middleware or final agent execution.
        """
        ...
```

**Context Object**: `AgentRunContext`
- Location: [_middleware.py#L60-L133](../../../python/packages/core/agent_framework/_middleware.py#L60-L133)
- Attributes:
  - `agent: AgentProtocol` - The agent being invoked
  - `messages: list[ChatMessage]` - Messages being sent
  - `thread: AgentThread | None` - Agent thread for this invocation
  - `is_streaming: bool` - Whether streaming mode
  - `metadata: dict[str, Any]` - For sharing data between middleware
  - `result: AgentResponse | AsyncIterable[AgentResponseUpdate] | None` - Can be set to override execution
  - `terminate: bool` - Flag to stop execution after current middleware
  - `kwargs: dict[str, Any]` - Additional kwargs passed to agent run

### 1.2 FunctionMiddleware

**File**: [_middleware.py](../../../python/packages/core/agent_framework/_middleware.py#L342-L404)

```python
class FunctionMiddleware(ABC):
    """Abstract base class for function middleware that can intercept function invocations."""

    @abstractmethod
    async def process(
        self,
        context: FunctionInvocationContext,
        next: Callable[[FunctionInvocationContext], Awaitable[None]],
    ) -> None:
        """Process a function invocation.

        Args:
            context: Function invocation context containing function, arguments, and metadata.
            next: Function to call the next middleware or final function execution.
        """
        ...
```

**Context Object**: `FunctionInvocationContext`
- Location: [_middleware.py#L136-L202](../../../python/packages/core/agent_framework/_middleware.py#L136-L202)
- Attributes:
  - `function: FunctionTool[Any, Any]` - The function being invoked
  - `arguments: BaseModel` - Validated arguments for the function
  - `metadata: dict[str, Any]` - For sharing data between middleware
  - `result: Any` - Can be set to override execution
  - `terminate: bool` - Flag to stop execution
  - `kwargs: dict[str, Any]` - Additional kwargs

### 1.3 ChatMiddleware

**File**: [_middleware.py](../../../python/packages/core/agent_framework/_middleware.py#L407-L468)

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
            next: Function to call the next middleware or final chat execution.
        """
        ...
```

**Context Object**: `ChatContext`
- Location: [_middleware.py#L205-L280](../../../python/packages/core/agent_framework/_middleware.py#L205-L280)
- Attributes:
  - `chat_client: ChatClientProtocol` - The chat client being invoked
  - `messages: MutableSequence[ChatMessage]` - Messages being sent
  - `options: Mapping[str, Any] | None` - Options for the chat request
  - `is_streaming: bool` - Whether streaming mode
  - `metadata: dict[str, Any]` - For sharing data between middleware
  - `result: ChatResponse | AsyncIterable[ChatResponseUpdate] | None` - Can be set to override
  - `terminate: bool` - Flag to stop execution
  - `kwargs: dict[str, Any]` - Additional kwargs

---

## 2. Type Aliases for Function-based Middleware

**File**: [_middleware.py#L471-L484](../../../python/packages/core/agent_framework/_middleware.py#L471-L484)

```python
# Pure function type definitions
AgentMiddlewareCallable = Callable[
    [AgentRunContext, Callable[[AgentRunContext], Awaitable[None]]], Awaitable[None]
]

FunctionMiddlewareCallable = Callable[
    [FunctionInvocationContext, Callable[[FunctionInvocationContext], Awaitable[None]]], Awaitable[None]
]

ChatMiddlewareCallable = Callable[
    [ChatContext, Callable[[ChatContext], Awaitable[None]]], Awaitable[None]
]

# Union type for all middleware
Middleware: TypeAlias = (
    AgentMiddleware | AgentMiddlewareCallable |
    FunctionMiddleware | FunctionMiddlewareCallable |
    ChatMiddleware | ChatMiddlewareCallable
)
```

---

## 3. Middleware Decorators for Function-based Middleware

**File**: [_middleware.py#L489-L592](../../../python/packages/core/agent_framework/_middleware.py#L489-L592)

### `@agent_middleware`
Marks a function as agent middleware by setting `_middleware_type = MiddlewareType.AGENT`.

### `@function_middleware`
Marks a function as function middleware by setting `_middleware_type = MiddlewareType.FUNCTION`.

### `@chat_middleware`
Marks a function as chat middleware by setting `_middleware_type = MiddlewareType.CHAT`.

**Usage Example**:
```python
from agent_framework import agent_middleware, AgentRunContext

@agent_middleware
async def logging_middleware(context: AgentRunContext, next):
    print(f"Before: {context.agent.name}")
    await next(context)
    print(f"After: {context.result}")
```

---

## 4. Middleware Pipeline Classes

### 4.1 BaseMiddlewarePipeline

**File**: [_middleware.py#L611-L745](../../../python/packages/core/agent_framework/_middleware.py#L611-L745)

Base class providing common functionality:
- `_middleware: list[Any]` - Internal middleware list
- `_register_middleware_with_wrapper()` - Wraps callable functions in `MiddlewareWrapper`
- `_create_handler_chain()` - Builds chain of middleware handlers for non-streaming
- `_create_streaming_handler_chain()` - Builds chain for streaming operations

### 4.2 AgentMiddlewarePipeline

**File**: [_middleware.py#L748-L875](../../../python/packages/core/agent_framework/_middleware.py#L748-L875)

```python
class AgentMiddlewarePipeline(BaseMiddlewarePipeline):
    async def execute(
        self,
        agent: AgentProtocol,
        messages: list[ChatMessage],
        context: AgentRunContext,
        final_handler: Callable[[AgentRunContext], Awaitable[AgentResponse]],
    ) -> AgentResponse | None:
        """Execute the agent middleware pipeline for non-streaming."""

    async def execute_stream(
        self,
        agent: AgentProtocol,
        messages: list[ChatMessage],
        context: AgentRunContext,
        final_handler: Callable[[AgentRunContext], AsyncIterable[AgentResponseUpdate]],
    ) -> AsyncIterable[AgentResponseUpdate]:
        """Execute the agent middleware pipeline for streaming."""
```

### 4.3 FunctionMiddlewarePipeline

**File**: [_middleware.py#L877-L948](../../../python/packages/core/agent_framework/_middleware.py#L877-L948)

```python
class FunctionMiddlewarePipeline(BaseMiddlewarePipeline):
    async def execute(
        self,
        function: Any,
        arguments: BaseModel,
        context: FunctionInvocationContext,
        final_handler: Callable[[FunctionInvocationContext], Awaitable[Any]],
    ) -> Any:
        """Execute the function middleware pipeline."""
```

### 4.4 ChatMiddlewarePipeline

**File**: [_middleware.py#L950-L1080](../../../python/packages/core/agent_framework/_middleware.py#L950-L1080)

```python
class ChatMiddlewarePipeline(BaseMiddlewarePipeline):
    async def execute(
        self,
        chat_client: ChatClientProtocol,
        messages: MutableSequence[ChatMessage],
        options: Mapping[str, Any] | None,
        context: ChatContext,
        final_handler: Callable[[ChatContext], Awaitable[ChatResponse]],
        **kwargs: Any,
    ) -> ChatResponse:
        """Execute the chat middleware pipeline."""

    async def execute_stream(
        self,
        chat_client: ChatClientProtocol,
        messages: MutableSequence[ChatMessage],
        options: Mapping[str, Any] | None,
        context: ChatContext,
        final_handler: Callable[[ChatContext], AsyncIterable[ChatResponseUpdate]],
        **kwargs: Any,
    ) -> AsyncIterable[ChatResponseUpdate]:
        """Execute the chat middleware pipeline for streaming."""
```

---

## 5. Middleware Registration and Invocation

### 5.1 Class Decorator: `@use_agent_middleware`

**File**: [_middleware.py#L1159-L1293](../../../python/packages/core/agent_framework/_middleware.py#L1159-L1293)

This decorator wraps the agent's `run()` and `run_stream()` methods to inject middleware support.

**Key Behavior**:
1. Builds fresh middleware pipelines for each invocation from agent-level + run-level middleware
2. Categorizes middleware by type (agent/function/chat)
3. Creates `AgentRunContext` and passes to pipeline
4. Executes pipeline with final handler that calls original method

```python
@use_agent_middleware
@use_agent_instrumentation(capture_usage=False)
class ChatAgent(BaseAgent, Generic[TOptions_co]):
    ...
```

### 5.2 Class Decorator: `@use_chat_middleware`

**File**: [_middleware.py#L1296-L1440](../../../python/packages/core/agent_framework/_middleware.py#L1296-L1440)

Wraps chat client's `get_response()` and `get_streaming_response()` methods.

### 5.3 Middleware Categorization

**File**: [_middleware.py#L1450-L1504](../../../python/packages/core/agent_framework/_middleware.py#L1450-L1504)

```python
def categorize_middleware(*middleware_sources: Middleware | None) -> MiddlewareDict:
    """Categorize middleware into agent, function, and chat types."""
    result: MiddlewareDict = {"agent": [], "function": [], "chat": []}
    
    for middleware in all_middleware:
        if isinstance(middleware, AgentMiddleware):
            result["agent"].append(middleware)
        elif isinstance(middleware, FunctionMiddleware):
            result["function"].append(middleware)
        elif isinstance(middleware, ChatMiddleware):
            result["chat"].append(middleware)
        elif callable(middleware):
            middleware_type = _determine_middleware_type(middleware)
            # Route to appropriate category
    
    return result
```

### 5.4 Type Detection for Functions

**File**: [_middleware.py#L1082-L1157](../../../python/packages/core/agent_framework/_middleware.py#L1082-L1157)

```python
def _determine_middleware_type(middleware: Any) -> MiddlewareType:
    """Determine middleware type using decorator and/or parameter type annotation."""
    # 1. Check for decorator marker (_middleware_type attribute)
    # 2. Check parameter type annotation (AgentRunContext, FunctionInvocationContext, ChatContext)
    # 3. Raise MiddlewareException if cannot determine
```

---

## 6. Middleware Invocation Chain Pattern

### Execution Flow (Non-Streaming)

```
Agent.run() called
    │
    ▼
@use_agent_middleware decorator intercepts
    │
    ▼
categorize_middleware() separates by type
    │
    ▼
AgentMiddlewarePipeline.execute()
    │
    ▼
_create_handler_chain() builds nested handlers
    │
    ▼
For each middleware (in order):
    │
    ├──▶ middleware.process(context, next)
    │        │
    │        ├── Pre-processing (before await next)
    │        │
    │        ├── await next(context)  ──┐
    │        │                          │
    │        │   (inner middleware)◀────┘
    │        │
    │        └── Post-processing (after await next)
    │
    ▼
Final handler: original run() method
    │
    ▼
Result stored in context.result
    │
    ▼
Return result to caller
```

### Termination Pattern

```python
async def process(self, context: AgentRunContext, next) -> None:
    if should_terminate:
        context.result = AgentResponse(messages=[...])
        context.terminate = True
        return  # Don't call next()
    
    await next(context)
```

---

## 7. `as_tool()` Method

**File**: [_agents.py#L411-L503](../../../python/packages/core/agent_framework/_agents.py#L411-L503)

Converts an agent into a `FunctionTool` that can be used by other agents.

### Signature

```python
def as_tool(
    self,
    *,
    name: str | None = None,
    description: str | None = None,
    arg_name: str = "task",
    arg_description: str | None = None,
    stream_callback: Callable[[AgentResponseUpdate], None]
        | Callable[[AgentResponseUpdate], Awaitable[None]]
        | None = None,
) -> FunctionTool[BaseModel, str]:
```

### Key Implementation Details

1. **Creates dynamic Pydantic model** for input using `create_model()`:
   ```python
   input_model = create_model(model_name, **{arg_name: (str, field_info)})
   ```

2. **Wrapper function** that calls agent and extracts text:
   ```python
   async def agent_wrapper(**kwargs: Any) -> str:
       input_text = kwargs.get(arg_name, "")
       if stream_callback is None:
           return (await self.run(input_text, **forwarded_kwargs)).text
       # Streaming mode with callback...
   ```

3. **Creates FunctionTool** with `approval_mode="never_require"`:
   ```python
   agent_tool: FunctionTool[BaseModel, str] = FunctionTool(
       name=tool_name,
       description=tool_description,
       func=agent_wrapper,
       input_model=input_model,
       approval_mode="never_require",
   )
   agent_tool._forward_runtime_kwargs = True
   ```

### Usage Example

```python
# Create an agent
agent = ChatAgent(chat_client=client, name="research-agent", description="Performs research")

# Convert to tool
research_tool = agent.as_tool()

# Use with another agent
coordinator = ChatAgent(chat_client=client, name="coordinator", tools=research_tool)
```

---

## 8. ContextProvider Pattern

**File**: [_memory.py#L70-L176](../../../python/packages/core/agent_framework/_memory.py#L70-L176)

### Abstract Base Class

```python
class ContextProvider(ABC):
    """Base class for all context providers."""

    async def thread_created(self, thread_id: str | None) -> None:
        """Called just after a new thread is created."""
        pass

    async def invoked(
        self,
        request_messages: ChatMessage | Sequence[ChatMessage],
        response_messages: ChatMessage | Sequence[ChatMessage] | None = None,
        invoke_exception: Exception | None = None,
        **kwargs: Any,
    ) -> None:
        """Called after the agent has received a response."""
        pass

    @abstractmethod
    async def invoking(
        self, messages: ChatMessage | MutableSequence[ChatMessage], **kwargs: Any
    ) -> Context:
        """Called just before the model/agent is invoked.
        
        Returns:
            Context object with instructions, messages, and tools to include.
        """
        pass
```

### Context Class

```python
class Context:
    def __init__(
        self,
        instructions: str | None = None,
        messages: Sequence[ChatMessage] | None = None,
        tools: Sequence[ToolProtocol] | None = None,
    ):
        self.instructions = instructions
        self.messages: Sequence[ChatMessage] = messages or []
        self.tools: Sequence[ToolProtocol] = tools or []
```

### Integration Point

**File**: [_agents.py#L1296](../../../python/packages/core/agent_framework/_agents.py#L1296)

```python
# In _prepare_thread_and_messages():
if self.context_provider:
    context = await self.context_provider.invoking(input_messages or [], **kwargs)
    if context:
        if context.messages:
            thread_messages.extend(context.messages)
        if context.tools:
            chat_options["tools"].extend(context.tools)
        if context.instructions:
            chat_options["instructions"] = (
                context.instructions if "instructions" not in chat_options
                else f"{chat_options['instructions']}\n{context.instructions}"
            )
```

### Lifecycle

1. **`thread_created()`** - Called when new thread created (for loading long-term state)
2. **`invoking()`** - Called before each invocation (returns additional context)
3. **`invoked()`** - Called after each invocation (for storing conversation updates)

### Implementations

| Provider | Package | File |
|----------|---------|------|
| `RedisProvider` | `agent_framework_redis` | [_provider.py#L34](../../../python/packages/redis/agent_framework_redis/_provider.py#L34) |
| `Mem0Provider` | `agent_framework_mem0` | [_provider.py#L32](../../../python/packages/mem0/agent_framework_mem0/_provider.py#L32) |
| `AzureAISearchContextProvider` | `agent_framework_azure_ai_search` | [_search_provider.py#L167](../../../python/packages/azure-ai-search/agent_framework_azure_ai_search/_search_provider.py#L167) |

---

## 9. Sample Implementations

### Class-based Agent Middleware

**File**: [class_based_middleware.py#L48-L74](../../../python/samples/getting_started/middleware/class_based_middleware.py#L48-L74)

```python
class SecurityAgentMiddleware(AgentMiddleware):
    async def process(
        self,
        context: AgentRunContext,
        next: Callable[[AgentRunContext], Awaitable[None]],
    ) -> None:
        last_message = context.messages[-1] if context.messages else None
        if last_message and last_message.text:
            query = last_message.text
            if "password" in query.lower() or "secret" in query.lower():
                context.result = AgentResponse(messages=[
                    ChatMessage(role=Role.ASSISTANT, text="Blocked for security.")
                ])
                return  # Don't call next() to block execution

        await next(context)
```

### Class-based Function Middleware

**File**: [class_based_middleware.py#L76-L95](../../../python/samples/getting_started/middleware/class_based_middleware.py#L76-L95)

```python
class LoggingFunctionMiddleware(FunctionMiddleware):
    async def process(
        self,
        context: FunctionInvocationContext,
        next: Callable[[FunctionInvocationContext], Awaitable[None]],
    ) -> None:
        print(f"Calling: {context.function.name}")
        start_time = time.time()
        
        await next(context)
        
        duration = time.time() - start_time
        print(f"Completed in {duration:.5f}s")
```

---

## 10. Key Files Reference

| Component | File Path | Line Range |
|-----------|-----------|------------|
| AgentMiddleware ABC | `python/packages/core/agent_framework/_middleware.py` | L283-L338 |
| FunctionMiddleware ABC | `python/packages/core/agent_framework/_middleware.py` | L342-L404 |
| ChatMiddleware ABC | `python/packages/core/agent_framework/_middleware.py` | L407-L468 |
| AgentRunContext | `python/packages/core/agent_framework/_middleware.py` | L60-L133 |
| FunctionInvocationContext | `python/packages/core/agent_framework/_middleware.py` | L136-L202 |
| ChatContext | `python/packages/core/agent_framework/_middleware.py` | L205-L280 |
| AgentMiddlewarePipeline | `python/packages/core/agent_framework/_middleware.py` | L748-L875 |
| FunctionMiddlewarePipeline | `python/packages/core/agent_framework/_middleware.py` | L877-L948 |
| ChatMiddlewarePipeline | `python/packages/core/agent_framework/_middleware.py` | L950-L1080 |
| @use_agent_middleware | `python/packages/core/agent_framework/_middleware.py` | L1159-L1293 |
| @use_chat_middleware | `python/packages/core/agent_framework/_middleware.py` | L1296-L1440 |
| categorize_middleware() | `python/packages/core/agent_framework/_middleware.py` | L1450-L1504 |
| as_tool() | `python/packages/core/agent_framework/_agents.py` | L411-L503 |
| ContextProvider ABC | `python/packages/core/agent_framework/_memory.py` | L70-L176 |
| Context class | `python/packages/core/agent_framework/_memory.py` | L23-L67 |

---

## Summary

The Python agent framework implements a comprehensive middleware system with:

1. **Three middleware types**: AgentMiddleware, FunctionMiddleware, ChatMiddleware
2. **Context objects** that flow through the pipeline containing all invocation state
3. **Pipeline classes** that build handler chains and execute middleware in sequence
4. **Class decorators** (`@use_agent_middleware`, `@use_chat_middleware`) that inject middleware support
5. **Automatic categorization** of mixed middleware lists by type detection
6. **Termination support** via `context.terminate` and `context.result` override
7. **`as_tool()` method** for agent-to-tool conversion with runtime kwargs forwarding
8. **ContextProvider pattern** for adding dynamic context (instructions, messages, tools) before invocation
