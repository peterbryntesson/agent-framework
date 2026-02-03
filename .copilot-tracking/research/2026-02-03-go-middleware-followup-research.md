<!-- markdownlint-disable-file -->
# Follow-up Research: Go Middleware Patterns

**Research Date**: 2026-02-03
**Related Review**: 2026-02-03-go-middleware-patterns-review.md
**Related Plan**: 2026-02-03-go-middleware-patterns-plan.instructions.md

## Research Summary

This research investigates the follow-up work items identified during the Go middleware patterns implementation review. Three areas were identified for future work: ChatMiddleware (status update), OpenTelemetry middleware integration, and Context Provider pattern implementation.

## Task Implementation Requests

* Update ChatMiddleware status from "Deferred" to "Implemented"
* Design OpenTelemetry middleware using AgentMiddleware pattern
* Design ContextProvider interface for dynamic context injection
* Document middleware execution order in README

## Key Findings

### 1. ChatMiddleware Status: Already Implemented

**Critical Discovery**: ChatMiddleware was already implemented as part of the middleware patterns work. The original review document incorrectly listed it as "Deferred".

#### Evidence

| Component | Status | Location |
|-----------|--------|----------|
| `ChatMiddleware` interface | ✅ Complete | [go/agent/middleware.go](../../go/agent/middleware.go) Lines 55-78 |
| `ChatContext` struct | ✅ Complete | [go/agent/middleware_context.go](../../go/agent/middleware_context.go) |
| `ChainChatMiddleware` | ✅ Complete | [go/agent/chain.go](../../go/agent/chain.go) Lines 64-95 |
| `WithChatMiddleware` option | ✅ Complete | [go/chatagent/options.go](../../go/chatagent/options.go) |
| Unit tests | ✅ Complete | [go/agent/chat_middleware_test.go](../../go/agent/chat_middleware_test.go) |
| Integration in toolloop | ✅ Complete | [go/chatagent/agent.go](../../go/chatagent/agent.go) Line 34 |

#### ChatMiddleware Interface

```go
// ChatMiddleware intercepts chat client requests.
// Implement this interface to add caching, request transformation,
// rate limiting, or logging at the chat client level.
// This operates at a lower level than AgentMiddleware, intercepting
// each individual GetResponse or GetStreamingResponse call.
type ChatMiddleware interface {
    // Process handles a chat client request.
    // Call next to continue the chain, or return early to short-circuit.
    // The chatCtx contains request data and receives the response.
    // For streaming requests, IsStreaming is true and Stream should be set.
    Process(ctx context.Context, chatCtx *ChatContext, next ChatHandler) error
}
```

#### Use Cases

* Response caching to avoid redundant API calls
* Request/response logging for debugging
* Rate limiting per client
* Retry with exponential backoff
* Cost tracking per request
* Metadata propagation between middleware

### 2. OpenTelemetry Middleware Integration

#### Current Observability Implementation

The `go/observability/` package provides:

| Component | Description |
|-----------|-------------|
| `InstrumentedClient` | A `chat.Client` wrapper that adds tracing/metrics at the chat level |
| `Metrics` struct | Counters and histograms for tokens, latency, errors |
| Span functions | `StartAgentSpan`, `StartChatSpan`, `StartToolSpan` |
| Semantic conventions | `semconv.go` with OpenTelemetry attribute names |

**Gap Identified**: No `InstrumentedAgent` wrapper like .NET's implementation.

#### Proposed Design: TelemetryMiddleware

Implement observability as `AgentMiddleware` for composability:

```go
// TelemetryMiddleware provides OpenTelemetry instrumentation for agent invocations.
type TelemetryMiddleware struct {
    metrics             *Metrics
    enableSensitiveData bool
    sourceName          string
}

// NewTelemetryMiddleware creates a new telemetry middleware.
func NewTelemetryMiddleware(opts ...TelemetryOption) *TelemetryMiddleware

// Process implements AgentMiddleware.
func (m *TelemetryMiddleware) Process(ctx context.Context, agentCtx *AgentContext, next AgentHandler) error {
    // Start span
    ctx, span := StartAgentSpan(ctx, agentCtx.AgentID, agentCtx.AgentName)
    defer span.End()
    
    start := time.Now()
    
    // Call next
    err := next(ctx, agentCtx)
    
    // Record metrics
    latency := time.Since(start).Seconds()
    m.recordMetrics(ctx, agentCtx, latency, err)
    
    // Record span attributes
    if err != nil {
        RecordError(span, err)
    }
    RecordResponse(span, agentCtx.Response)
    
    return err
}
```

#### Telemetry Data to Capture

| Category | Data |
|----------|------|
| **Spans** | `agent.run`, `agent.run_stream`, `tool.call` |
| **Attributes** | agent.id, agent.name, provider, model, tokens, finish_reason |
| **Metrics** | agent.runs count, input/output tokens, latency histogram, error count |

#### Benefits of Middleware Approach

1. **Composability**: Chain with logging, security, caching middleware
2. **Consistency**: Same pattern as .NET/Python implementations
3. **Flexibility**: Configure via functional options
4. **Separation**: Observability isolated from agent logic
5. **Testability**: Easy to mock and test in isolation

#### Implementation Recommendation

1. Create `go/observability/middleware.go` with `TelemetryMiddleware`
2. Add functional options for configuration
3. Integrate with existing `Metrics` struct
4. Update `go/README.md` with observability middleware examples
5. Create sample showing middleware-based observability

### 3. Context Provider Pattern

#### Python ContextProvider Analysis

The Python framework defines `ContextProvider` as an abstract base class in [_memory.py](../../python/packages/core/agent_framework/_memory.py):

```python
class ContextProvider(ABC):
    async def thread_created(self, thread_id: str | None) -> None:
        """Called after a new thread is created."""
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
    async def invoking(self, messages: ChatMessage | MutableSequence[ChatMessage], **kwargs: Any) -> Context:
        """Called just before the model/agent is invoked."""
        ...
```

The `Context` struct contains:
* `instructions`: Additional instructions to prepend
* `messages`: Additional messages to include
* `tools`: Additional tools to provide

#### Current Go Context Handling

* Static instructions/tools set at agent creation
* Session history for stateful conversations
* Run-level tools via `WithTools()` option
* No dynamic context injection before model invocation

#### Proposed Go Design

```go
// Context provides dynamic context to be injected before agent invocation.
type Context struct {
    Instructions string
    Messages     []chat.Message
    Tools        []tool.Tool
}

// ContextProvider injects dynamic context before agent invocations.
type ContextProvider interface {
    // Invoking is called just before the agent invokes the chat client.
    // Return context to be merged with the agent's base configuration.
    Invoking(ctx context.Context, messages []chat.Message) (*Context, error)
}

// ContextProviderWithLifecycle extends ContextProvider with lifecycle hooks.
type ContextProviderWithLifecycle interface {
    ContextProvider
    
    // SessionCreated is called when a new session is created.
    SessionCreated(ctx context.Context, sessionID string) error
    
    // Invoked is called after the agent receives a response.
    Invoked(ctx context.Context, request, response []chat.Message, err error) error
}

// AggregateContextProvider combines multiple context providers.
type AggregateContextProvider struct {
    providers []ContextProvider
}
```

#### Integration Points

```go
// In chatagent/options.go
func WithContextProvider(provider ContextProvider) Option

// In chatagent/agent.go - context injection order:
// 1. Base agent instructions
// 2. Provider instructions (from Invoking)
// 3. Provider messages (from Invoking)
// 4. Session history
// 5. Runtime messages (from Run call)
```

#### Use Cases

* **User-specific context**: Personalization based on user profile
* **Session state/memory**: Inject remembered facts from previous sessions
* **Dynamic tools**: Enable/disable tools based on feature flags
* **RAG integration**: Inject retrieved documents before each invocation

#### Implementation Recommendation

1. Create `go/agent/context_provider.go` with interfaces
2. Add `Context` struct to `go/agent/context.go`
3. Add `AggregateContextProvider` implementation
4. Add `WithContextProvider()` option to chatagent
5. Integrate context injection in `runWithToolLoop()`
6. Create sample showing user personalization pattern

### 4. Documentation Enhancement: Middleware Execution Order

The current README documents middleware chaining but doesn't explicitly explain execution order.

#### Recommendation

Add to `go/README.md` in the Middleware section:

```markdown
### Middleware Execution Order

Middlewares are executed in the order they are added, with the first middleware
being the outermost wrapper:

​```go
agent := chatagent.NewBuilder(client).
    UseMiddleware(loggingMiddleware).   // First to receive, last to return
    UseMiddleware(securityMiddleware).  // Second to receive, second-to-last to return
    UseMiddleware(cachingMiddleware).   // Last to receive, first to return
    BuildAgent()
​```

Execution flow:
1. loggingMiddleware.Process starts
2.   securityMiddleware.Process starts
3.     cachingMiddleware.Process starts
4.       → agent.Run executes
5.     cachingMiddleware.Process completes
6.   securityMiddleware.Process completes
7. loggingMiddleware.Process completes
```

## Priority Recommendations

| Item | Priority | Effort | Reasoning |
|------|----------|--------|-----------|
| Update review for ChatMiddleware | High | Low | Corrects documentation error |
| Documentation: execution order | High | Low | Improves developer understanding |
| TelemetryMiddleware | Medium | Medium | Provides observability parity with .NET |
| ContextProvider | Medium | Medium | Enables advanced personalization scenarios |

## Potential Next Research

* FunctionMiddleware for OpenTelemetry (trace tool invocations)
* ChatMiddleware for OpenTelemetry (trace API calls)
* ContextProvider samples for RAG integration
* Memory persistence with Redis/database backends

## Success Criteria

* ChatMiddleware correctly marked as implemented in review
* README includes middleware execution order documentation
* TelemetryMiddleware design validated against .NET InstrumentedAgent
* ContextProvider design validated against Python implementation
