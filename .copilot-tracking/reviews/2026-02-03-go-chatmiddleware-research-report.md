# Go Agent Framework: ChatMiddleware Patterns Research Report

**Date:** 2026-02-03  
**Status:** Complete  
**Scope:** Analysis of existing middleware patterns and ChatMiddleware implementation

---

## Executive Summary

The Go agent framework **already has a fully implemented ChatMiddleware pattern** that follows the same design principles as AgentMiddleware and FunctionMiddleware. The implementation is well-integrated into the chatagent package and provides interception points for all chat client calls within the tool loop.

---

## 1. Current Middleware Pattern Analysis

### 1.1 Middleware Architecture Overview

The framework implements a **three-tier middleware hierarchy**:

| Layer | Interface | Intercepts | Level |
|-------|-----------|------------|-------|
| **Agent** | `AgentMiddleware` | Full agent invocations (Run/RunStream) | Highest |
| **Function** | `FunctionMiddleware` | Tool/function invocations | Middle |
| **Chat** | `ChatMiddleware` | Chat client requests (GetResponse/GetStreamingResponse) | Lowest |

### 1.2 Common Middleware Pattern

All three middleware types follow the same pattern:

```go
// Handler type - represents the next step in the chain
type XHandler func(ctx context.Context, xCtx *XContext) error

// Middleware interface - intercepts and can modify execution
type XMiddleware interface {
    Process(ctx context.Context, xCtx *XContext, next XHandler) error
}

// Function adapter - allows inline middleware creation
type XMiddlewareFunc func(ctx context.Context, xCtx *XContext, next XHandler) error

func (f XMiddlewareFunc) Process(ctx context.Context, xCtx *XContext, next XHandler) error {
    return f(ctx, xCtx, next)
}
```

### 1.3 Key Design Patterns

1. **Chain of Responsibility**: Middleware forms a chain where each can:
   - Pre-process before calling `next()`
   - Post-process after `next()` returns
   - Short-circuit by not calling `next()` and setting response directly

2. **Context Objects**: Each middleware type has a dedicated context struct containing:
   - Input data (messages, arguments, etc.)
   - Output fields (Response, Result, Error)
   - Metadata map for cross-middleware communication
   - Streaming flag and stream channel

3. **Functional Options**: Middleware is configured via `WithXMiddleware()` options during agent construction

---

## 2. ChatClient Interface Analysis

### 2.1 Client Interface

Location: [go/chat/client.go](go/chat/client.go)

```go
type Client interface {
    GetResponse(ctx context.Context, messages []Message, options *Options) (*Response, error)
    GetStreamingResponse(ctx context.Context, messages []Message, options *Options) (<-chan ResponseUpdate, error)
    Metadata() ClientMetadata
}
```

### 2.2 Key Characteristics

| Method | Purpose | Return Type |
|--------|---------|-------------|
| `GetResponse` | Synchronous completion | `*Response, error` |
| `GetStreamingResponse` | Streaming completion | `<-chan ResponseUpdate, error` |
| `Metadata` | Provider info | `ClientMetadata` |

### 2.3 Options Available for Interception

The `chat.Options` struct provides these configuration points:

- Model parameters: `MaxTokens`, `Temperature`, `TopP`, `Seed`
- Content control: `StopSequences`, `ResponseFormat`
- Tool configuration: `Tools`, `ToolChoice`
- Request metadata: `Instructions`, `ModelID`, `User`
- Persistence: `Store`, `ConversationID`

---

## 3. Existing ChatMiddleware Implementation

### 3.1 Core Types

Location: [go/agent/middleware.go](go/agent/middleware.go#L53-L79)

```go
// ChatHandler is the next handler in the chat middleware chain.
type ChatHandler func(ctx context.Context, chatCtx *ChatContext) error

// ChatMiddleware intercepts chat client requests.
type ChatMiddleware interface {
    Process(ctx context.Context, chatCtx *ChatContext, next ChatHandler) error
}

// ChatMiddlewareFunc is a function adapter for ChatMiddleware.
type ChatMiddlewareFunc func(ctx context.Context, chatCtx *ChatContext, next ChatHandler) error
```

### 3.2 ChatContext Structure

Location: [go/agent/middleware_context.go](go/agent/middleware_context.go#L61-L86)

```go
type ChatContext struct {
    // Input
    ClientMetadata ChatClientMetadata
    Messages       []Message
    Options        map[string]any
    Metadata       map[string]any  // For cross-middleware data

    // Output (non-streaming)
    Response *ChatResponse

    // Output (streaming)
    Stream      <-chan ChatResponseUpdate
    IsStreaming bool
}
```

### 3.3 Middleware Chaining

Location: [go/agent/chain.go](go/agent/chain.go#L64-L97)

```go
func ChainChatMiddleware(middlewares ...ChatMiddleware) ChatMiddleware {
    // Filters nil middlewares
    // Returns passthrough if empty
    // Builds chain from last to first (proper execution order)
}
```

### 3.4 Integration in ChatAgent

Location: [go/chatagent/toolloop.go](go/chatagent/toolloop.go#L454-L500)

The `invokeWithChatMiddleware` method:

1. Builds `ChatContext` from request data
2. Defines terminal handler that calls actual `chat.Client`
3. Executes through middleware chain
4. Handles both streaming and non-streaming responses

```go
func (a *Agent) invokeWithChatMiddleware(ctx context.Context, messages []chat.Message, 
    options *chat.Options, isStreaming bool) (*chat.Response, <-chan chat.ResponseUpdate, error) {
    
    chatCtx := &agent.ChatContext{
        ClientMetadata: agent.ChatClientMetadata{...},
        Messages:       convertMessagesToAgentMessages(messages),
        Options:        convertChatOptionsToMap(options),
        Metadata:       make(map[string]any),
        IsStreaming:    isStreaming,
    }

    terminal := func(ctx context.Context, c *agent.ChatContext) error {
        // Calls actual chat client
    }

    err := a.chatMiddleware.Process(ctx, chatCtx, terminal)
    // ...
}
```

---

## 4. Interception Points

### 4.1 Request Flow Diagram

```
Agent.Run()
    │
    ├── prepareMessages()
    ├── prepareChatOptions()
    │
    └── runWithToolLoop()
            │
            ├──── invokeWithChatMiddleware() ◄── ChatMiddleware intercepts here
            │         │
            │         ├── [Middleware 1: Before]
            │         ├── [Middleware 2: Before]
            │         ├── [Middleware N: Before]
            │         │
            │         ├── client.GetResponse() / GetStreamingResponse()
            │         │
            │         ├── [Middleware N: After]
            │         ├── [Middleware 2: After]
            │         └── [Middleware 1: After]
            │
            ├── Process response
            ├── invokeToolCalls() (if tool calls present)
            │         │
            │         └── FunctionMiddleware intercepts here
            │
            └── Loop back if more tool calls needed
```

### 4.2 Multiple Invocations per Agent.Run()

**Critical insight**: ChatMiddleware is invoked **once per chat client call**, not once per `Agent.Run()`. In a tool loop scenario:

- First iteration: ChatMiddleware sees initial user message
- After tool execution: ChatMiddleware sees conversation + tool results
- This continues until no more tool calls or max turns reached

This makes ChatMiddleware ideal for:

- Per-request metrics and logging
- Rate limiting per API call
- Response caching (with awareness of conversation context)

---

## 5. Use Cases and Benefits

### 5.1 Implemented Use Cases (from tests)

| Use Case | Implementation | File Reference |
|----------|----------------|----------------|
| Request/Response Logging | `recordingChatMiddleware` | [chat_middleware_test.go](go/chatagent/chat_middleware_test.go#L14-L33) |
| Response Caching | `cachingChatMiddleware` | [chat_middleware_test.go](go/chatagent/chat_middleware_test.go#L110-L133) |
| Metadata Propagation | Test verifies metadata flows | [chat_middleware_test.go](go/chatagent/chat_middleware_test.go#L233-L250) |
| Execution Order | Test verifies first-to-last order | [chat_middleware_test.go](go/chatagent/chat_middleware_test.go#L181-L230) |
| Short-circuit | Cache hit skips client call | [chat_middleware_test.go](go/chatagent/chat_middleware_test.go#L136-L177) |

### 5.2 Additional Use Cases (Documented)

From [go/README.md](go/README.md#L378-L418) and [go/agent/doc.go](go/agent/doc.go#L90-L128):

1. **Response Caching and Memoization**
   - Cache expensive API responses
   - Avoid redundant calls for identical prompts

2. **Rate Limiting and Throttling**
   - Implement token bucket or sliding window algorithms
   - Respect API quotas and rate limits

3. **Request/Response Logging and Metrics**
   - Log conversation history for debugging
   - Track latency, token usage, error rates

4. **Message Transformation**
   - Modify messages before sending (PII redaction, prompt injection)
   - Transform responses (formatting, filtering)

### 5.3 Potential Additional Use Cases

| Use Case | Description | Complexity |
|----------|-------------|------------|
| **Retry with Backoff** | Automatic retry on transient errors | Medium |
| **Circuit Breaker** | Fail fast when service is degraded | Medium |
| **Cost Tracking** | Track API costs based on token usage | Low |
| **Semantic Caching** | Cache based on semantic similarity | High |
| **A/B Testing** | Route requests to different models | Medium |
| **Fallback Models** | Switch to backup model on failure | Medium |
| **Request Validation** | Validate message content/length | Low |
| **Response Validation** | Ensure responses meet criteria | Low |
| **Observability** | OpenTelemetry spans for each call | Medium |

---

## 6. Implementation Analysis

### 6.1 Strengths

1. **Consistent Pattern**: Same middleware pattern across all three types
2. **Streaming Support**: First-class support for streaming responses
3. **Metadata Flow**: Cross-middleware data via `Metadata` map
4. **Nil-Safe Chaining**: `ChainChatMiddleware` filters nil middlewares
5. **Functional Adapters**: Easy inline middleware via `ChatMiddlewareFunc`
6. **Builder Integration**: `WithChatMiddleware()` option in chatagent

### 6.2 Potential Improvements

| Area | Current State | Suggestion |
|------|---------------|------------|
| **Pre-built Middleware** | Only examples in tests/docs | Create `middleware/` package with common implementations |
| **Metrics Integration** | Not integrated | Add OpenTelemetry ChatMiddleware in observability package |
| **Retry Middleware** | Not provided | Implement configurable retry with exponential backoff |
| **Rate Limiter** | Not provided | Implement token bucket rate limiter |
| **Options Access** | Options converted to `map[string]any` | Consider typed options in ChatContext |

---

## 7. Implementation Recommendations

### 7.1 Short-term Enhancements

1. **Logging Middleware** - Create `observability.ChatLoggingMiddleware`:

   ```go
   type ChatLoggingMiddleware struct {
       Logger *slog.Logger
       Level  slog.Level
   }
   ```

2. **Metrics Middleware** - Create `observability.ChatMetricsMiddleware`:
   - Track `llm.request.duration`
   - Track `llm.tokens.usage`
   - Track `llm.request.errors`

3. **Tracing Middleware** - Integrate with existing observability:
   - Create spans for each chat request
   - Record message counts, token usage as span attributes

### 7.2 Medium-term Enhancements

1. **Retry Middleware Package**:

   ```go
   type RetryMiddleware struct {
       MaxRetries  int
       BackoffFunc func(attempt int) time.Duration
       Retryable   func(error) bool
   }
   ```

2. **Rate Limiting Middleware**:

   ```go
   type RateLimitMiddleware struct {
       Limiter   *rate.Limiter
       WaitOnLimit bool
   }
   ```

3. **Caching Middleware Package**:

   ```go
   type CachingMiddleware struct {
       Cache     Cache
       KeyFunc   func(*ChatContext) string
       TTL       time.Duration
   }
   ```

### 7.3 Documentation Improvements

1. Add more comprehensive examples to package docs
2. Document streaming middleware considerations
3. Add troubleshooting guide for common middleware issues

---

## 8. Conclusion

The ChatMiddleware implementation in the Go agent framework is **complete and well-designed**. It follows established patterns from AgentMiddleware and FunctionMiddleware, providing a consistent developer experience.

### Key Findings

- ✅ ChatMiddleware interface is **already implemented**
- ✅ ChatContext provides comprehensive request/response access
- ✅ Middleware chaining works correctly with proper execution order
- ✅ Both streaming and non-streaming requests are supported
- ✅ Integration with chatagent is complete via `WithChatMiddleware()`

### Recommendations Priority

1. **High**: Add pre-built middleware implementations (logging, metrics, retry)
2. **Medium**: Integrate with observability package for OpenTelemetry
3. **Low**: Enhance ChatContext with typed options access

---

## Appendix: File References

| Component | File | Lines |
|-----------|------|-------|
| ChatMiddleware interface | [go/agent/middleware.go](go/agent/middleware.go) | 53-79 |
| ChatContext struct | [go/agent/middleware_context.go](go/agent/middleware_context.go) | 61-86 |
| ChainChatMiddleware | [go/agent/chain.go](go/agent/chain.go) | 64-97 |
| Agent integration | [go/chatagent/toolloop.go](go/chatagent/toolloop.go) | 454-500 |
| WithChatMiddleware option | [go/chatagent/options.go](go/chatagent/options.go) | 155-164 |
| Unit tests | [go/agent/chat_middleware_test.go](go/agent/chat_middleware_test.go) | 1-260 |
| Integration tests | [go/chatagent/chat_middleware_test.go](go/chatagent/chat_middleware_test.go) | 1-317 |
