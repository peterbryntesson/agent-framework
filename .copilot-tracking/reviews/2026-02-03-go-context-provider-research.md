# Go Context Provider Pattern Research Report

**Date**: February 3, 2026  
**Status**: Complete  
**Author**: GitHub Copilot

---

## Executive Summary

This report analyzes the Python `ContextProvider` pattern and proposes a Go implementation for dynamic context injection before agent runs. The proposed design follows Go idioms while providing equivalent functionality for runtime context modification.

---

## 1. Python ContextProvider Analysis

### 1.1 Core Interface

The Python `ContextProvider` is an abstract base class defined in [_memory.py](../../python/packages/core/agent_framework/_memory.py) with three lifecycle methods:

```python
class ContextProvider(ABC):
    DEFAULT_CONTEXT_PROMPT: Final[str] = "## Memories\n..."

    async def thread_created(self, thread_id: str | None) -> None: ...
    async def invoked(self, request_messages, response_messages, invoke_exception, **kwargs) -> None: ...
    @abstractmethod
    async def invoking(self, messages, **kwargs) -> Context: ...
```

### 1.2 Context Class

The `Context` class holds the data returned by a `ContextProvider`:

```python
class Context:
    def __init__(
        self,
        instructions: str | None = None,
        messages: Sequence[ChatMessage] | None = None,
        tools: Sequence[ToolProtocol] | None = None,
    ): ...
```

### 1.3 Lifecycle Methods

| Method | When Called | Purpose |
|--------|-------------|---------|
| `thread_created(thread_id)` | After new thread creation | Load relevant data from long-term storage |
| `invoking(messages)` | Just before model invocation | Return context (instructions, messages, tools) |
| `invoked(request, response, exception)` | After model response | Update provider state, persist data |

### 1.4 Key Patterns

1. **Async Context Manager**: Supports `async with` for setup/teardown
2. **Dynamic Context**: Context is computed at invocation time, not at agent creation
3. **Composable**: `AggregateContextProvider` combines multiple providers
4. **Stateful**: Providers can maintain state across invocations

### 1.5 Real-World Implementations

1. **UserInfoMemory** ([simple_context_provider.py](../../python/samples/getting_started/context_providers/simple_context_provider.py)):
   - Extracts user info from messages after calls
   - Injects personalized instructions before calls

2. **RedisProvider** ([_provider.py](../../python/packages/redis/agent_framework_redis/_provider.py)):
   - Stores/retrieves context from Redis
   - Supports vector search for relevant memories
   - Scopes by application, agent, user, and thread IDs

---

## 2. Current Go Context Handling Analysis

### 2.1 Agent Interface

The Go `Agent` interface in [agent/agent.go](../../go/agent/agent.go) defines:

```go
type Agent interface {
    Run(ctx context.Context, messages []Message, opts ...RunOption) (*Response, error)
    RunStream(ctx context.Context, messages []Message, opts ...RunOption) (<-chan ResponseUpdate, error)
    // ...
}
```

### 2.2 ChatAgent Context Handling

The `ChatAgent` in [chatagent/agent.go](../../go/chatagent/agent.go) handles context through:

1. **Static Instructions**: Set at agent creation via `WithInstructions()`
2. **Static Tools**: Set at agent creation via `WithTools()`
3. **Session History**: Managed via `RunOption` - `WithSession()`
4. **Run-time Tools**: Added via `WithTools()` run option

Key method ([chatagent/agent.go#L158-L180](../../go/chatagent/agent.go)):

```go
func (a *Agent) prepareMessages(messages []agent.Message, cfg *agent.RunConfig) []chat.Message {
    // 1. Add system instructions (static)
    if a.instructions != "" {
        chatMessages = append(chatMessages, chat.NewSystemMessage(a.instructions))
    }
    // 2. Add session history
    if cfg.Session != nil {
        chatMessages = append(chatMessages, cfg.Session.Messages()...)
    }
    // 3. Add new messages
    chatMessages = append(chatMessages, messages...)
    return chatMessages
}
```

### 2.3 Existing Middleware Patterns

The Go framework has three middleware types in [agent/middleware.go](../../go/agent/middleware.go):

| Middleware | Target | Use Case |
|------------|--------|----------|
| `AgentMiddleware` | Agent invocations | Request modification, validation, logging |
| `FunctionMiddleware` | Tool/function calls | Caching, validation, transformation |
| `ChatMiddleware` | Chat client requests | Rate limiting, caching, request transformation |

### 2.4 Gap Analysis

| Feature | Python | Go |
|---------|--------|-----|
| Dynamic instructions | ✅ ContextProvider.invoking() | ❌ Static at creation |
| Dynamic messages | ✅ Context.messages | ❌ Session-only |
| Dynamic tools | ✅ Context.tools | ⚠️ RunOption only |
| Thread lifecycle | ✅ thread_created() | ❌ None |
| Post-invocation hooks | ✅ invoked() | ⚠️ Via AgentMiddleware |

---

## 3. Proposed Go ContextProvider Design

### 3.1 Core Interface

```go
// Package agent

// Context holds dynamic context provided before an agent invocation.
// This is returned by ContextProvider.Invoking and merged with agent configuration.
type Context struct {
    // Instructions are additional system instructions to include.
    // These are prepended as a system message after the agent's base instructions.
    Instructions string
    
    // Messages are additional messages to include in the conversation.
    // These are inserted after instructions but before session history.
    Messages []Message
    
    // Tools are additional tools available for this invocation only.
    // These augment the agent's base tools and run-option tools.
    Tools []interface{}
}

// ContextProvider supplies dynamic context before agent invocations.
// Implementations can inject instructions, messages, or tools based on
// runtime state, user context, or external data sources.
type ContextProvider interface {
    // Invoking is called just before the agent invokes the model.
    // It receives the current messages and returns additional context.
    // The returned Context is merged with agent configuration.
    // Implementations should be fast to avoid blocking the agent run.
    Invoking(ctx context.Context, messages []Message) (*Context, error)
}

// ContextProviderWithLifecycle extends ContextProvider with lifecycle hooks.
// Implement this interface when you need session/thread tracking or
// post-invocation processing.
type ContextProviderWithLifecycle interface {
    ContextProvider
    
    // SessionCreated is called when a new session is created.
    // Use this to load session-specific data from storage.
    SessionCreated(ctx context.Context, sessionID string) error
    
    // Invoked is called after the agent receives a response.
    // Use this to update state, persist data, or log results.
    Invoked(ctx context.Context, request []Message, response *Response, err error) error
}
```

### 3.2 Functional Adapter

```go
// ContextProviderFunc is a function adapter for simple ContextProvider implementations.
// Use this when you only need the Invoking method.
type ContextProviderFunc func(ctx context.Context, messages []Message) (*Context, error)

// Invoking implements ContextProvider.
func (f ContextProviderFunc) Invoking(ctx context.Context, messages []Message) (*Context, error) {
    return f(ctx, messages)
}
```

### 3.3 Aggregate Provider

```go
// AggregateContextProvider combines multiple providers.
// Contexts are merged in order: later providers override earlier ones.
type AggregateContextProvider struct {
    providers []ContextProvider
}

// NewAggregateContextProvider creates a provider that combines multiple providers.
func NewAggregateContextProvider(providers ...ContextProvider) *AggregateContextProvider {
    return &AggregateContextProvider{providers: providers}
}

// Invoking calls all providers and merges their contexts.
func (a *AggregateContextProvider) Invoking(ctx context.Context, messages []Message) (*Context, error) {
    result := &Context{
        Messages: make([]Message, 0),
        Tools:    make([]interface{}, 0),
    }
    
    for _, p := range a.providers {
        c, err := p.Invoking(ctx, messages)
        if err != nil {
            return nil, err
        }
        if c == nil {
            continue
        }
        
        // Merge instructions (concatenate with newlines)
        if c.Instructions != "" {
            if result.Instructions != "" {
                result.Instructions += "\n" + c.Instructions
            } else {
                result.Instructions = c.Instructions
            }
        }
        
        // Append messages and tools
        result.Messages = append(result.Messages, c.Messages...)
        result.Tools = append(result.Tools, c.Tools...)
    }
    
    return result, nil
}
```

### 3.4 ChatAgent Integration

```go
// In chatagent/options.go
type config struct {
    // ... existing fields ...
    contextProvider agent.ContextProvider
}

// WithContextProvider sets the context provider for dynamic context injection.
func WithContextProvider(provider agent.ContextProvider) Option {
    return func(c *config) {
        c.contextProvider = provider
    }
}
```

```go
// In chatagent/agent.go

// prepareMessages prepares messages including dynamic context.
func (a *Agent) prepareMessages(ctx context.Context, messages []agent.Message, cfg *agent.RunConfig) ([]chat.Message, error) {
    capacity := len(messages) + 1
    if cfg.Session != nil {
        capacity += len(cfg.Session.Messages())
    }
    
    chatMessages := make([]chat.Message, 0, capacity)
    
    // 1. Add agent's base system instructions
    if a.instructions != "" {
        chatMessages = append(chatMessages, chat.NewSystemMessage(a.instructions))
    }
    
    // 2. Get dynamic context from provider
    if a.contextProvider != nil {
        providerCtx, err := a.contextProvider.Invoking(ctx, messages)
        if err != nil {
            return nil, fmt.Errorf("context provider: %w", err)
        }
        if providerCtx != nil {
            // Add provider instructions as system message
            if providerCtx.Instructions != "" {
                chatMessages = append(chatMessages, chat.NewSystemMessage(providerCtx.Instructions))
            }
            // Add provider messages
            for _, m := range providerCtx.Messages {
                chatMessages = append(chatMessages, m)
            }
            // Tools are handled separately in prepareChatOptions
        }
    }
    
    // 3. Add session history
    if cfg.Session != nil {
        chatMessages = append(chatMessages, cfg.Session.Messages()...)
    }
    
    // 4. Add new messages
    chatMessages = append(chatMessages, messages...)
    
    return chatMessages, nil
}
```

---

## 4. Integration Points

### 4.1 Where ContextProvider Fits

```
┌─────────────────────────────────────────────────────────────┐
│                      Agent.Run()                            │
├─────────────────────────────────────────────────────────────┤
│  1. AgentMiddleware.Process()  ◄── Cross-cutting concerns   │
│     │                                                        │
│     ▼                                                        │
│  2. ContextProvider.Invoking() ◄── Dynamic context injection │
│     │                                                        │
│     ▼                                                        │
│  3. prepareMessages()          ◄── Merge all context        │
│     │                                                        │
│     ▼                                                        │
│  4. ChatMiddleware.Process()   ◄── Request transformation   │
│     │                                                        │
│     ▼                                                        │
│  5. chat.Client.GetResponse()  ◄── Model invocation         │
│     │                                                        │
│     ▼                                                        │
│  6. Tool invocation loop       ◄── FunctionMiddleware       │
│     │                                                        │
│     ▼                                                        │
│  7. ContextProvider.Invoked()  ◄── Post-processing (if impl)│
└─────────────────────────────────────────────────────────────┘
```

### 4.2 Relationship with Existing Types

| Existing Type | Relationship with ContextProvider |
|---------------|-----------------------------------|
| `RunOption` | ContextProvider is set at agent creation, not per-run |
| `Session` | ContextProvider can access session via messages |
| `AgentMiddleware` | ContextProvider is called after AgentMiddleware |
| `ChatMiddleware` | ContextProvider affects messages before ChatMiddleware |

### 4.3 Context Merging Order

1. Agent base instructions (static)
2. ContextProvider instructions (dynamic)
3. ContextProvider messages (dynamic)
4. Session history (stateful)
5. Run-time messages (per-call)
6. Run-time tools (per-call) + ContextProvider tools (dynamic) + Agent tools (static)

---

## 5. Use Cases

### 5.1 User-Specific Context

```go
type UserContextProvider struct {
    userService UserService
}

func (p *UserContextProvider) Invoking(ctx context.Context, messages []agent.Message) (*agent.Context, error) {
    userID, ok := ctx.Value("user_id").(string)
    if !ok {
        return nil, nil // No user context available
    }
    
    user, err := p.userService.Get(ctx, userID)
    if err != nil {
        return nil, err
    }
    
    return &agent.Context{
        Instructions: fmt.Sprintf("The user's name is %s. Their preferences: %s",
            user.Name, user.Preferences),
    }, nil
}
```

### 5.2 Session State / Memory

```go
type MemoryProvider struct {
    store MemoryStore
}

func (p *MemoryProvider) Invoking(ctx context.Context, messages []agent.Message) (*agent.Context, error) {
    sessionID, _ := ctx.Value("session_id").(string)
    if sessionID == "" {
        return nil, nil
    }
    
    memories, err := p.store.Search(ctx, sessionID, getLastUserMessage(messages))
    if err != nil {
        return nil, err
    }
    
    if len(memories) == 0 {
        return nil, nil
    }
    
    return &agent.Context{
        Instructions: fmt.Sprintf("## Relevant Memories\n%s", formatMemories(memories)),
    }, nil
}

// Implement ContextProviderWithLifecycle for state persistence
func (p *MemoryProvider) Invoked(ctx context.Context, request []agent.Message, response *agent.Response, err error) error {
    if err != nil {
        return nil // Don't store failed interactions
    }
    
    sessionID, _ := ctx.Value("session_id").(string)
    return p.store.Add(ctx, sessionID, request, response.Messages)
}
```

### 5.3 Dynamic Tools

```go
type FeatureFlagToolProvider struct {
    flags FeatureFlagService
}

func (p *FeatureFlagToolProvider) Invoking(ctx context.Context, messages []agent.Message) (*agent.Context, error) {
    userID, _ := ctx.Value("user_id").(string)
    
    tools := make([]interface{}, 0)
    
    if p.flags.IsEnabled(ctx, "advanced_search", userID) {
        tools = append(tools, advancedSearchTool)
    }
    
    if p.flags.IsEnabled(ctx, "code_execution", userID) {
        tools = append(tools, codeExecutionTool)
    }
    
    return &agent.Context{Tools: tools}, nil
}
```

### 5.4 RAG (Retrieval-Augmented Generation)

```go
type RAGProvider struct {
    vectorStore VectorStore
    maxResults  int
}

func (p *RAGProvider) Invoking(ctx context.Context, messages []agent.Message) (*agent.Context, error) {
    query := getLastUserMessage(messages)
    if query == "" {
        return nil, nil
    }
    
    docs, err := p.vectorStore.Search(ctx, query, p.maxResults)
    if err != nil {
        return nil, err
    }
    
    if len(docs) == 0 {
        return nil, nil
    }
    
    return &agent.Context{
        Instructions: fmt.Sprintf("## Relevant Documents\n%s\n\nUse these documents to answer the user's question.",
            formatDocuments(docs)),
    }, nil
}
```

---

## 6. Design Decisions

### 6.1 Interface vs Struct

**Decision**: Use interface for maximum flexibility.

**Rationale**:
- Follows Go idioms (interfaces are implicit)
- Enables composition and decoration
- Allows different implementations (in-memory, Redis, database)

### 6.2 Separate Lifecycle Interface

**Decision**: Split lifecycle methods into `ContextProviderWithLifecycle`.

**Rationale**:
- Most providers only need `Invoking`
- Keeps the base interface simple
- Follows Interface Segregation Principle
- Agent can type-assert when needed

### 6.3 Error Handling

**Decision**: Return errors from `Invoking`.

**Rationale**:
- Allows provider to fail fast
- Agent can decide how to handle (retry, fallback, abort)
- Consistent with Go error handling patterns

### 6.4 Context Struct vs Map

**Decision**: Use a typed `Context` struct.

**Rationale**:
- Type safety at compile time
- Clear API contract
- Easier to document and test
- Can be extended with new fields without breaking changes

---

## 7. Implementation Plan

### Phase 1: Core Types
1. Add `Context` struct to `agent/context.go`
2. Add `ContextProvider` interface
3. Add `ContextProviderFunc` adapter
4. Add unit tests

### Phase 2: ChatAgent Integration
1. Add `contextProvider` field to `chatagent.config`
2. Add `WithContextProvider` option
3. Update `prepareMessages` to call provider
4. Update `prepareChatOptions` for dynamic tools
5. Add integration tests

### Phase 3: Lifecycle Support
1. Add `ContextProviderWithLifecycle` interface
2. Call `SessionCreated` in `NewSession`
3. Call `Invoked` after run completion
4. Add lifecycle tests

### Phase 4: Aggregate Provider
1. Implement `AggregateContextProvider`
2. Add merging logic tests
3. Add sample implementations

---

## 8. Comparison Summary

| Aspect | Python | Proposed Go |
|--------|--------|-------------|
| Interface | `ContextProvider(ABC)` | `ContextProvider` interface |
| Context type | `Context` class | `Context` struct |
| Required method | `invoking()` | `Invoking()` |
| Lifecycle hooks | Same class | Separate `ContextProviderWithLifecycle` |
| Aggregation | `AggregateContextProvider` | `AggregateContextProvider` |
| Async support | Native async/await | context.Context for cancellation |
| Type safety | Runtime (Protocol) | Compile-time (interface) |

---

## 9. Files to Create/Modify

| File | Action | Description |
|------|--------|-------------|
| `go/agent/context_provider.go` | Create | Core interfaces and types |
| `go/agent/context_provider_test.go` | Create | Unit tests |
| `go/agent/aggregate_context_provider.go` | Create | Aggregate implementation |
| `go/chatagent/options.go` | Modify | Add `WithContextProvider` |
| `go/chatagent/agent.go` | Modify | Call provider in prepareMessages |
| `go/chatagent/context_provider_test.go` | Create | Integration tests |

---

## 10. Open Questions

1. **Concurrent providers**: Should `AggregateContextProvider` call providers concurrently?
   - Recommendation: Sequential by default, add `ConcurrentAggregateContextProvider` if needed

2. **Provider ordering**: Should providers be explicitly ordered or rely on registration order?
   - Recommendation: Registration order (simpler, predictable)

3. **Error policy**: Should aggregate provider fail-fast or collect all errors?
   - Recommendation: Fail-fast (consistent with Go patterns)

4. **Run-level override**: Should `RunOption` allow per-run context providers?
   - Recommendation: Not initially; can be added later if needed
