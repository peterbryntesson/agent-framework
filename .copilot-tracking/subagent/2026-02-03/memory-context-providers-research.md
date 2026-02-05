# Memory and Context Provider Research for Epic 3 Feature 3.2

> **Research Date:** 2026-02-03
> **Purpose:** Document Memory and Context Provider implementations across .NET, Python, and Go for Epic 3 Feature 3.2 alignment.

## Executive Summary

All three languages implement a **ContextProvider** pattern with similar lifecycle methods. Go has a complete implementation with `ContextProvider`, `ContextProviderWithLifecycle`, and `AggregateContextProvider`. The pattern is well-established and provides hooks for:

1. **Invoking** - Called before agent invocation to provide context
2. **Invoked** - Called after agent invocation to process results
3. **Session/Thread Created** - Called when a new session is created

---

## 1. .NET Context/Memory Provider Types

### Core Abstract Class

**File:** [AIContextProvider.cs](dotnet/src/Microsoft.Agents.AI.Abstractions/AIContextProvider.cs)

```csharp
public abstract class AIContextProvider
{
    // Called before agent invocation
    public abstract ValueTask<AIContext> InvokingAsync(InvokingContext context, CancellationToken cancellationToken = default);
    
    // Called after agent invocation (optional)
    public virtual ValueTask InvokedAsync(InvokedContext context, CancellationToken cancellationToken = default) => default;
    
    // Serialization support
    public virtual JsonElement Serialize(JsonSerializerOptions? jsonSerializerOptions = null) => default;
    
    // Service resolution
    public virtual object? GetService(Type serviceType, object? serviceKey = null);
}
```

### Context Classes

**File:** [AIContext.cs](dotnet/src/Microsoft.Agents.AI.Abstractions/AIContext.cs)

```csharp
public sealed class AIContext
{
    public string? Instructions { get; set; }           // Transient instructions
    public IList<ChatMessage>? Messages { get; set; }   // Permanent history additions
    public IList<AITool>? Tools { get; set; }           // Transient tools
}
```

### Provider Implementations

| Provider | File | Purpose |
|----------|------|---------|
| `Mem0Provider` | [Mem0Provider.cs](dotnet/src/Microsoft.Agents.AI.Mem0/Mem0Provider.cs) | Persists conversation messages to Mem0 service |
| `TextSearchProvider` | [TextSearchProvider.cs](dotnet/src/Microsoft.Agents.AI/TextSearchProvider.cs) | RAG with text search (before invoke or on-demand) |
| `ChatHistoryMemoryProvider` | [ChatHistoryMemoryProvider.cs](dotnet/src/Microsoft.Agents.AI/Memory/ChatHistoryMemoryProvider.cs) | Vector store-backed chat history retrieval |

### Key Patterns

- Uses factory pattern for provider creation via `AIContextProviderFactory`
- Supports state serialization/deserialization
- Providers are instantiated per-session via factory context
- No built-in AggregateContextProvider (handled at agent level)

---

## 2. Python Context/Memory Provider Types

### Core Abstract Class

**File:** [_memory.py](python/packages/core/agent_framework/_memory.py)

```python
class ContextProvider(ABC):
    DEFAULT_CONTEXT_PROMPT: Final[str] = "## Memories\nConsider the following memories..."
    
    async def thread_created(self, thread_id: str | None) -> None:
        """Called when a new thread is created."""
        pass
    
    async def invoked(
        self,
        request_messages: ChatMessage | Sequence[ChatMessage],
        response_messages: ChatMessage | Sequence[ChatMessage] | None = None,
        invoke_exception: Exception | None = None,
        **kwargs: Any,
    ) -> None:
        """Called after agent invocation."""
        pass
    
    @abstractmethod
    async def invoking(self, messages: ChatMessage | MutableSequence[ChatMessage], **kwargs: Any) -> Context:
        """Called before agent invocation."""
        pass
    
    async def __aenter__(self) -> Self: ...
    async def __aexit__(self, ...): ...
```

### Context Class

**File:** [_memory.py](python/packages/core/agent_framework/_memory.py)

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

### Provider Implementations

| Provider | Package | Purpose |
|----------|---------|---------|
| `Mem0Provider` | [agent_framework_mem0](python/packages/mem0/agent_framework_mem0/_provider.py) | Mem0 memory integration |
| `RedisProvider` | [agent_framework_redis](python/packages/redis/agent_framework_redis/_provider.py) | Redis-backed context with hybrid search |
| `AzureAISearchContextProvider` | [agent_framework_azure_ai_search](python/packages/azure-ai-search/agent_framework_azure_ai_search/_search_provider.py) | Azure AI Search with semantic/agentic modes |

### Key Patterns

- Async context manager support (`__aenter__`, `__aexit__`)
- Provider attached to `AgentThread` for lifecycle management
- Single `context_provider` per agent (no built-in aggregation)
- Uses `ContextProvider.DEFAULT_CONTEXT_PROMPT` for consistency

---

## 3. Go Context/Memory Provider Types

### Core Interfaces

**File:** [context_provider.go](go/agent/context_provider.go)

```go
// Basic interface - only requires Invoking
type ContextProvider interface {
    Invoking(ctx context.Context, messages []Message) (*Context, error)
}

// Extended interface with lifecycle hooks
type ContextProviderWithLifecycle interface {
    ContextProvider
    Invoked(ctx context.Context, request []Message, response []Message, invokeErr error) error
    SessionCreated(ctx context.Context, sessionID string) error
}

// Function adapter for simple providers
type ContextProviderFunc func(ctx context.Context, messages []Message) (*Context, error)
```

### Context Struct

```go
type Context struct {
    Instructions string         // Additional system instructions
    Messages     []chat.Message // Prepended before user messages
    Tools        []tool.Tool    // Merged with agent tools
}
```

### Helper Types

| Type | Purpose |
|------|---------|
| `BaseContextProvider` | Embeddable struct with no-op lifecycle implementations |
| `AggregateContextProvider` | Combines multiple providers with concurrent invocation |
| `ContextProviderFunc` | Function adapter for inline provider creation |

### AggregateContextProvider Details

**File:** [context_provider.go](go/agent/context_provider.go#L112-L262)

```go
type AggregateContextProvider struct {
    providers []ContextProvider
}

func NewAggregateContextProvider(providers ...ContextProvider) *AggregateContextProvider

func (a *AggregateContextProvider) Invoking(ctx context.Context, messages []Message) (*Context, error)
func (a *AggregateContextProvider) Invoked(ctx context.Context, request, response []Message, invokeErr error) error
func (a *AggregateContextProvider) SessionCreated(ctx context.Context, sessionID string) error
```

**Behavior:**
- Invokes all providers **concurrently** using goroutines
- Merges results in **deterministic order** (sorted by provider index)
- Instructions concatenated with newlines
- Messages and Tools extended in order
- First error wins (aborts invocation)

### Agent Integration

**File:** [agent.go](go/chatagent/agent.go#L66-L71)

```go
// In New():
if len(cfg.contextProviders) > 0 {
    if len(cfg.contextProviders) == 1 {
        a.contextProvider = cfg.contextProviders[0]
    } else {
        a.contextProvider = agent.NewAggregateContextProvider(cfg.contextProviders...)
    }
}
```

---

## 4. Cross-Language Interface Comparison

| Feature | .NET | Python | Go |
|---------|------|--------|-------|
| **Base Interface/Class** | `AIContextProvider` (abstract class) | `ContextProvider` (ABC) | `ContextProvider` (interface) |
| **Invoking Method** | `InvokingAsync` | `invoking` | `Invoking` |
| **Invoked Method** | `InvokedAsync` (virtual) | `invoked` (optional) | `Invoked` (via `ContextProviderWithLifecycle`) |
| **Session Created** | Via factory context | `thread_created` | `SessionCreated` |
| **Context Class** | `AIContext` | `Context` | `Context` |
| **Aggregation** | At agent level | At agent level | `AggregateContextProvider` |
| **Async Support** | ValueTask | async/await | Context-based cancellation |
| **Serialization** | `Serialize()` method | Not built-in | Not implemented |
| **Resource Cleanup** | `IDisposable` where needed | `__aenter__`/`__aexit__` | Not built-in |

---

## 5. Go Implementation Status

### Fully Implemented ✅

- [x] `ContextProvider` interface
- [x] `ContextProviderWithLifecycle` interface
- [x] `ContextProviderFunc` function adapter
- [x] `BaseContextProvider` embeddable struct
- [x] `AggregateContextProvider` with concurrent invocation
- [x] `Context` struct with Instructions, Messages, Tools
- [x] Agent integration via `WithContextProvider` option
- [x] Lifecycle hook invocations in agent Run/RunStream

### Not Yet Implemented ❌

- [ ] Concrete provider implementations (Mem0, Redis, Azure Search)
- [ ] State serialization/deserialization
- [ ] Resource cleanup/disposal patterns

### Test Coverage

**File:** [context_provider_test.go](go/chatagent/context_provider_test.go)

Tests cover:
- Instruction injection
- Message injection
- Provider-only instructions (no base instructions)
- Error handling (aborts run)
- Multiple provider merging
- Message ordering
- Nil context handling

---

## 6. Recommendations for Epic 3 Feature 3.2

### Priority 1: Concrete Provider Implementations

Based on .NET and Python patterns, consider implementing:

1. **TextSearchProvider** - RAG with pluggable search backend
   - Before-invoke and on-demand modes
   - Recent message memory for multi-turn context

2. **VectorStoreProvider** - Chat history with semantic search
   - Uses generic VectorStore interface
   - Scoped by application/agent/user/thread

### Priority 2: Serialization Support

Add state serialization to `Context` and providers:

```go
type SerializableContextProvider interface {
    ContextProvider
    Serialize() (json.RawMessage, error)
}
```

### Priority 3: Resource Management

Consider adding cleanup support:

```go
type CloseableContextProvider interface {
    ContextProvider
    Close() error
}
```

---

## 7. File References

### .NET

- [AIContextProvider.cs](dotnet/src/Microsoft.Agents.AI.Abstractions/AIContextProvider.cs) - Base class
- [AIContext.cs](dotnet/src/Microsoft.Agents.AI.Abstractions/AIContext.cs) - Context struct
- [Mem0Provider.cs](dotnet/src/Microsoft.Agents.AI.Mem0/Mem0Provider.cs) - Mem0 implementation
- [TextSearchProvider.cs](dotnet/src/Microsoft.Agents.AI/TextSearchProvider.cs) - RAG provider
- [ChatHistoryMemoryProvider.cs](dotnet/src/Microsoft.Agents.AI/Memory/ChatHistoryMemoryProvider.cs) - Vector store provider

### Python

- [_memory.py](python/packages/core/agent_framework/_memory.py) - Base class and Context
- [_provider.py (mem0)](python/packages/mem0/agent_framework_mem0/_provider.py) - Mem0 implementation
- [_provider.py (redis)](python/packages/redis/agent_framework_redis/_provider.py) - Redis implementation
- [_search_provider.py](python/packages/azure-ai-search/agent_framework_azure_ai_search/_search_provider.py) - Azure Search

### Go

- [context_provider.go](go/agent/context_provider.go) - All interfaces and types
- [options.go](go/chatagent/options.go) - WithContextProvider option
- [agent.go](go/chatagent/agent.go) - Agent integration
- [context_provider_test.go](go/chatagent/context_provider_test.go) - Test coverage

---

## 8. Summary

The Go implementation has a **complete foundation** for context providers with proper interface design, lifecycle hooks, and aggregation support. The main gaps are:

1. **No concrete provider implementations** (Mem0, search, vector store)
2. **No serialization support** for state persistence
3. **No resource cleanup patterns** for providers with external connections

For Epic 3 Feature 3.2, focus on:
- Implementing `TextSearchProvider` or equivalent for RAG scenarios
- Adding serialization interfaces if session persistence is needed
- Following the existing Go patterns (interfaces over abstract classes, functional options)
