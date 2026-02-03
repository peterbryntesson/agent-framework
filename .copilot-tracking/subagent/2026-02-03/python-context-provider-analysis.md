# Python ContextProvider Analysis

## Overview

This document analyzes the Python agent framework's `ContextProvider` implementation pattern for reference when implementing similar functionality in Go.

---

## 1. ContextProvider Interface

**File:** [python/packages/core/agent_framework/_memory.py](python/packages/core/agent_framework/_memory.py#L64-L161)

```python
class ContextProvider(ABC):
    """Base class for all context providers."""

    # Default prompt for assembling memories/instructions
    DEFAULT_CONTEXT_PROMPT: Final[str] = "## Memories\nConsider the following memories when answering user questions:"

    async def thread_created(self, thread_id: str | None) -> None:
        """Called just after a new thread is created."""
        pass  # Optional - default is no-op

    async def invoked(
        self,
        request_messages: ChatMessage | Sequence[ChatMessage],
        response_messages: ChatMessage | Sequence[ChatMessage] | None = None,
        invoke_exception: Exception | None = None,
        **kwargs: Any,
    ) -> None:
        """Called after the agent has received a response."""
        pass  # Optional - default is no-op

    @abstractmethod
    async def invoking(self, messages: ChatMessage | MutableSequence[ChatMessage], **kwargs: Any) -> Context:
        """Called just before the model/agent is invoked. REQUIRED."""
        pass

    async def __aenter__(self) -> "Self":
        """Enter async context manager."""
        return self

    async def __aexit__(self, exc_type, exc_val, exc_tb) -> None:
        """Exit async context manager."""
        pass
```

### Method Summary

| Method | Required | Purpose |
|--------|----------|---------|
| `invoking()` | Yes (abstract) | Called before agent invocation, returns `Context` with additional instructions/messages/tools |
| `invoked()` | No | Called after agent invocation with request/response for state tracking |
| `thread_created()` | No | Called when new thread is created for initialization |
| `__aenter__/__aexit__` | No | Async context manager for resource lifecycle |

---

## 2. Context Class

**File:** [python/packages/core/agent_framework/_memory.py](python/packages/core/agent_framework/_memory.py#L22-L60)

```python
class Context:
    def __init__(
        self,
        instructions: str | None = None,
        messages: Sequence[ChatMessage] | None = None,
        tools: Sequence["ToolProtocol"] | None = None,
    ):
        self.instructions = instructions
        self.messages: Sequence[ChatMessage] = messages or []
        self.tools: Sequence["ToolProtocol"] = tools or []
```

### Context Fields

| Field | Type | Purpose |
|-------|------|---------|
| `instructions` | `str \| None` | Additional system instructions to append |
| `messages` | `Sequence[ChatMessage]` | Additional messages to prepend to conversation |
| `tools` | `Sequence[ToolProtocol]` | Additional tools to make available for this run |

**Key Characteristics:**

- Context is **per-invocation** (not stored in chat history)
- Multiple contexts from different providers are **merged** before invocation
- Instructions are concatenated, messages and tools are extended

---

## 3. Integration Points

### 3.1 Agent Initialization

**File:** [python/packages/core/agent_framework/_agents.py](python/packages/core/agent_framework/_agents.py#L329-L351)

The `context_provider` is passed to `BaseAgent` and stored:

```python
def __init__(
    self,
    *,
    context_provider: ContextProvider | None = None,
    # ...
):
    self.context_provider = context_provider
```

### 3.2 Thread Creation

**File:** [python/packages/core/agent_framework/_agents.py](python/packages/core/agent_framework/_agents.py#L393)

When creating a new thread, the agent's context_provider is passed to it:

```python
def get_new_thread(self, **kwargs: Any) -> AgentThread:
    return AgentThread(**kwargs, context_provider=self.context_provider)
```

### 3.3 Before Invocation (invoking)

**File:** [python/packages/core/agent_framework/_agents.py](python/packages/core/agent_framework/_agents.py#L1295-L1312)

In `_prepare_thread_and_messages()`, the `invoking()` method is called:

```python
async def _prepare_thread_and_messages(
    self,
    *,
    thread: AgentThread | None,
    input_messages: list[ChatMessage] | None = None,
    **kwargs: Any,
) -> tuple[AgentThread, dict[str, Any], list[ChatMessage]]:
    # ...
    context: Context | None = None
    if self.context_provider:
        context = await self.context_provider.invoking(input_messages or [], **kwargs)
        if context:
            # Append context messages BEFORE input messages
            if context.messages:
                thread_messages.extend(context.messages)
            # Merge context tools with chat options
            if context.tools:
                if chat_options.get("tools") is not None:
                    chat_options["tools"].extend(context.tools)
                else:
                    chat_options["tools"] = list(context.tools)
            # Append context instructions
            if context.instructions:
                chat_options["instructions"] = (
                    context.instructions
                    if "instructions" not in chat_options
                    else f"{chat_options['instructions']}\n{context.instructions}"
                )
    thread_messages.extend(input_messages or [])
```

### 3.4 After Invocation (invoked)

**File:** [python/packages/core/agent_framework/_agents.py](python/packages/core/agent_framework/_agents.py#L381-L382)

In `_notify_thread_of_new_messages()`:

```python
async def _notify_thread_of_new_messages(
    self,
    thread: AgentThread,
    input_messages: ChatMessage | Sequence[ChatMessage],
    response_messages: ChatMessage | Sequence[ChatMessage],
    **kwargs: Any,
) -> None:
    # ...
    if thread.context_provider:
        await thread.context_provider.invoked(input_messages, response_messages, **kwargs)
```

### 3.5 Thread Created Hook

**File:** [python/packages/core/agent_framework/_agents.py](python/packages/core/agent_framework/_agents.py#L1238-L1241)

Called when service returns a conversation ID:

```python
if response_conversation_id is not None:
    thread.service_thread_id = response_conversation_id
    if thread.context_provider:
        await thread.context_provider.thread_created(thread.service_thread_id)
```

---

## 4. Lifecycle Methods

### Call Order During Agent.run()

```
1. _prepare_thread_and_messages()
   └── context_provider.invoking(input_messages)  → Returns Context
   └── Context merged into chat_options and thread_messages

2. chat_client.get_response(thread_messages, chat_options)

3. _update_thread_with_type_and_conversation_id()
   └── context_provider.thread_created(thread_id)  [if new thread]

4. _notify_thread_of_new_messages()
   └── thread.context_provider.invoked(request, response)
```

### Thread Lifecycle

The `thread_created()` hook is called:

1. When a new thread is assigned a `service_thread_id` from the service
2. When `_prepare_thread_and_messages()` detects an existing `service_thread_id`

---

## 5. Multiple Providers (Aggregation)

**File:** [python/samples/getting_started/context_providers/aggregate_context_provider.py](python/samples/getting_started/context_providers/aggregate_context_provider.py#L40-L115)

The `AggregateContextProvider` pattern combines multiple providers:

```python
class AggregateContextProvider(ContextProvider):
    def __init__(self, context_providers: ContextProvider | Sequence[ContextProvider] | None = None):
        self.providers = [context_providers] if isinstance(context_providers, ContextProvider) else (context_providers or [])

    async def thread_created(self, thread_id: str | None = None) -> None:
        # Call all providers in parallel
        await asyncio.gather(*[x.thread_created(thread_id) for x in self.providers])

    async def invoking(self, messages: ChatMessage | MutableSequence[ChatMessage], **kwargs: Any) -> Context:
        # Call all providers in parallel and merge results
        contexts = await asyncio.gather(*[provider.invoking(messages, **kwargs) for provider in self.providers])
        
        instructions: str = ""
        return_messages: list[ChatMessage] = []
        tools: list["ToolProtocol"] = []
        
        for ctx in contexts:
            if ctx.instructions:
                instructions += ctx.instructions  # Concatenate
            if ctx.messages:
                return_messages.extend(ctx.messages)  # Extend
            if ctx.tools:
                tools.extend(ctx.tools)  # Extend
        
        return Context(instructions=instructions, messages=return_messages, tools=tools)

    async def invoked(self, request_messages, response_messages, invoke_exception, **kwargs) -> None:
        # Call all providers in parallel
        await asyncio.gather(*[x.invoked(request_messages, response_messages, invoke_exception, **kwargs) for x in self.providers])
```

### Aggregation Rules

| Field | Aggregation Method |
|-------|-------------------|
| `instructions` | String concatenation |
| `messages` | List extend (order preserved) |
| `tools` | List extend |

---

## 6. Sample Implementations

### 6.1 Simple Context Provider (Time/Persona)

**File:** [python/samples/getting_started/context_providers/aggregate_context_provider.py](python/samples/getting_started/context_providers/aggregate_context_provider.py#L165-L210)

```python
class TimeContextProvider(ContextProvider):
    """Adds current time to context."""
    async def invoking(self, messages, **kwargs) -> Context:
        current_time = datetime.now().strftime("%Y-%m-%d %H:%M:%S")
        return Context(instructions=f"The current date and time is: {current_time}. ")


class PersonaContextProvider(ContextProvider):
    """Adds a persona to the agent."""
    def __init__(self, persona: str):
        self.persona = persona

    async def invoking(self, messages, **kwargs) -> Context:
        return Context(instructions=f"Your persona: {self.persona}. ")
```

### 6.2 Stateful Provider with Extraction

**File:** [python/samples/getting_started/context_providers/simple_context_provider.py](python/samples/getting_started/context_providers/simple_context_provider.py#L18-L82)

```python
class UserInfoMemory(ContextProvider):
    """Extracts and stores user info, provides it as context."""
    
    def __init__(self, chat_client, user_info=None, **kwargs):
        self._chat_client = chat_client
        self.user_info = user_info or UserInfo()

    async def invoked(self, request_messages, response_messages, invoke_exception, **kwargs) -> None:
        """Extract user information from messages after each call."""
        # Uses LLM to extract structured data from conversation
        if self.user_info.name is None or self.user_info.age is None:
            result = await self._chat_client.get_response(
                messages=request_messages,
                instructions="Extract the user's name and age...",
                options={"response_format": UserInfo},
            )
            if extracted := result.try_parse_value(UserInfo):
                self.user_info.name = self.user_info.name or extracted.name
                self.user_info.age = self.user_info.age or extracted.age

    async def invoking(self, messages, **kwargs) -> Context:
        """Provide user info as context before each call."""
        instructions = []
        if self.user_info.name:
            instructions.append(f"The user's name is {self.user_info.name}.")
        else:
            instructions.append("Ask for the user's name.")
        return Context(instructions=" ".join(instructions))
```

### 6.3 External Memory Provider (Mem0)

**File:** [python/packages/mem0/agent_framework_mem0/_provider.py](python/packages/mem0/agent_framework_mem0/_provider.py#L32-L135)

```python
class Mem0Provider(ContextProvider):
    """Memory provider using Mem0 service."""
    
    async def thread_created(self, thread_id: str | None = None) -> None:
        """Store thread ID for scoping."""
        self._per_operation_thread_id = thread_id

    async def invoked(self, request_messages, response_messages, **kwargs) -> None:
        """Store conversation in Mem0 after invocation."""
        messages = [*request_messages, *response_messages]
        await self.mem0_client.add(
            messages=messages,
            user_id=self.user_id,
            agent_id=self.agent_id,
            run_id=self._per_operation_thread_id,
        )

    async def invoking(self, messages, **kwargs) -> Context:
        """Search Mem0 for relevant memories."""
        input_text = "\n".join(msg.text for msg in messages if msg.text)
        search_results = await self.mem0_client.search(query=input_text, ...)
        
        if search_results:
            formatted = self._format_memories(search_results)
            return Context(instructions=f"{self.context_prompt}\n{formatted}")
        return Context()
```

### 6.4 Redis Vector Store Provider

**File:** [python/packages/redis/agent_framework_redis/_provider.py](python/packages/redis/agent_framework_redis/_provider.py#L34-L100)

```python
class RedisProvider(ContextProvider):
    """Redis context provider with vector search."""
    
    def __init__(
        self,
        redis_url: str = "redis://localhost:6379",
        redis_vectorizer: BaseVectorizer | None = None,
        context_prompt: str = ContextProvider.DEFAULT_CONTEXT_PROMPT,
        # Scope filters
        application_id: str | None = None,
        agent_id: str | None = None,
        user_id: str | None = None,
        thread_id: str | None = None,
    ):
        self.context_prompt = context_prompt
        # ... initialization
```

---

## 7. Go Implementation Recommendations

Based on the Python patterns, the Go implementation should:

### Interface Design

```go
// Context represents additional context for agent invocation
type Context struct {
    Instructions string
    Messages     []ChatMessage
    Tools        []Tool
}

// ContextProvider interface
type ContextProvider interface {
    // Required: Called before invocation
    Invoking(ctx context.Context, messages []ChatMessage) (Context, error)
    
    // Optional: Called after invocation
    Invoked(ctx context.Context, request []ChatMessage, response []ChatMessage, err error) error
    
    // Optional: Called when thread is created
    ThreadCreated(ctx context.Context, threadID string) error
}
```

### Integration Points

1. **Agent construction** - Accept `ContextProvider` option
2. **Thread creation** - Store provider reference
3. **Before Run** - Call `Invoking()` and merge context
4. **After Run** - Call `Invoked()` with request/response

### Aggregation

Provide an `AggregateContextProvider` that:

- Stores multiple providers
- Calls all providers concurrently (using goroutines)
- Merges results (concatenate instructions, extend messages/tools)

---

## References

- [_memory.py - ContextProvider base class](python/packages/core/agent_framework/_memory.py)
- [_agents.py - Integration in ChatAgent](python/packages/core/agent_framework/_agents.py)
- [_threads.py - AgentThread with context_provider](python/packages/core/agent_framework/_threads.py)
- [aggregate_context_provider.py - Aggregation sample](python/samples/getting_started/context_providers/aggregate_context_provider.py)
- [simple_context_provider.py - Basic sample](python/samples/getting_started/context_providers/simple_context_provider.py)
- [Mem0 provider](python/packages/mem0/agent_framework_mem0/_provider.py)
- [Redis provider](python/packages/redis/agent_framework_redis/_provider.py)
