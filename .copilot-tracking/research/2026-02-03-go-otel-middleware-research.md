# OpenTelemetry Integration Through Middleware for Go Agent Framework

This research document analyzes the current observability implementation and proposes an OpenTelemetry middleware design for the Go agent framework.

## Executive Summary

The Go agent framework has a well-designed `AgentMiddleware` interface that enables cross-cutting concerns like observability. The current observability package provides low-level instrumentation helpers (`InstrumentedClient`, `InstrumentedAgentClient`) but lacks a complete agent wrapper pattern. This research proposes implementing OpenTelemetry observability as `AgentMiddleware`, aligning with Go idioms while achieving parity with the .NET `OpenTelemetryAgent` implementation.

---

## 1. Current Observability Implementation Analysis

### 1.1 Package Structure

The `go/observability/` package contains:

| File | Purpose |
|------|---------|
| [otel.go](go/observability/otel.go) | Core OpenTelemetry integration: tracer/meter access, span creation, attribute recording |
| [metrics.go](go/observability/metrics.go) | Metric instruments: counters, histograms for agent runs, tokens, latency, errors |
| [semconv.go](go/observability/semconv.go) | Semantic convention constants for GenAI observability |
| [instrumented.go](go/observability/instrumented.go) | `InstrumentedClient` wrapper for `chat.Client` and `InstrumentedAgentClient` helper |
| [setup.go](go/observability/setup.go) | OpenTelemetry SDK setup with OTLP exporters |

### 1.2 Current Components

#### InstrumentedClient (chat.Client wrapper)

```go
type InstrumentedClient struct {
    inner   chat.Client
    metrics *Metrics
    EnableSensitiveData bool
}
```

**Strengths:**

- Implements `chat.Client` interface (transparent decorator)
- Records spans for `GetResponse` and `GetStreamingResponse`
- Captures token usage, latency, errors, and finish reasons
- Supports both streaming and non-streaming operations
- Options pattern with `WithSensitiveData`, `WithMetrics`

**Limitations:**

- Only instruments at the chat client level
- Does not capture agent-level context (agent ID, name, description)
- Requires manual wrapping of the chat client

#### InstrumentedAgentClient (helper, not wrapper)

```go
type InstrumentedAgentClient struct {
    metrics *Metrics
}

func (c *InstrumentedAgentClient) StartRun(ctx context.Context, agentID, agentName, providerName, modelID string) (context.Context, func(err error, usage *chat.UsageDetails))
```

**Strengths:**

- Provides agent-level span creation with proper semantic conventions
- Returns completion callback for recording results
- Records agent run metrics

**Limitations:**

- Helper only—does not implement `agent.Agent` interface
- Requires manual integration in agent implementations
- Not composable with middleware pattern

#### Span Creation Functions

```go
func StartAgentSpan(ctx, operationName, agentName string, opts...) (context.Context, trace.Span)
func StartAgentSpanWithID(ctx, agentID, agentName, providerName string, opts...) (context.Context, trace.Span)
func StartChatSpan(ctx, system, model string, opts...) (context.Context, trace.Span)
func StartToolSpan(ctx, toolName, callID string, opts...) (context.Context, trace.Span)
```

**Strengths:**

- Follow OpenTelemetry semantic conventions for GenAI
- Proper span kinds (client for external calls, internal for tool calls)
- Include standard attributes (operation name, model, provider)

### 1.3 Semantic Conventions Implemented

The package follows [OpenTelemetry Semantic Conventions for GenAI](https://opentelemetry.io/docs/specs/semconv/gen-ai/):

| Convention Key | Purpose |
|----------------|---------|
| `gen_ai.system` | AI system identifier (e.g., "openai") |
| `gen_ai.operation.name` | Operation type (chat, agent.run, tool.call) |
| `gen_ai.request.model` | Requested model ID |
| `gen_ai.usage.input_tokens` | Input token count |
| `gen_ai.usage.output_tokens` | Output token count |
| `gen_ai.response.finish_reasons` | Why generation stopped |
| `agent.id`, `agent.name` | Agent identification |

### 1.4 Metrics Implemented

| Metric | Type | Description |
|--------|------|-------------|
| `gen_ai.agent.runs` | Counter | Number of agent runs |
| `gen_ai.usage.input_tokens` | Histogram | Input tokens per request |
| `gen_ai.usage.output_tokens` | Histogram | Output tokens per request |
| `gen_ai.request.latency` | Histogram | Request latency in seconds |
| `gen_ai.errors` | Counter | Error count |
| `gen_ai.tool.invocations` | Counter | Tool invocation count |

---

## 2. AgentMiddleware Interface Analysis

### 2.1 Interface Definition

From [go/agent/middleware.go](go/agent/middleware.go):

```go
// AgentHandler is the next handler in the agent middleware chain.
type AgentHandler func(ctx context.Context, agentCtx *AgentContext) error

// AgentMiddleware intercepts agent invocations.
type AgentMiddleware interface {
    Process(ctx context.Context, agentCtx *AgentContext, next AgentHandler) error
}

// AgentMiddlewareFunc is a function adapter for AgentMiddleware.
type AgentMiddlewareFunc func(ctx context.Context, agentCtx *AgentContext, next AgentHandler) error
```

### 2.2 AgentContext Structure

From [go/agent/middleware_context.go](go/agent/middleware_context.go):

```go
type AgentContext struct {
    Agent       Agent              // The agent being invoked
    Messages    []Message          // Input messages
    Session     Session            // Optional session
    Options     *RunConfig         // Run configuration
    Metadata    map[string]any     // Middleware data attachment
    Response    *Response          // Result for non-streaming
    Stream      <-chan ResponseUpdate  // Result for streaming
    IsStreaming bool               // Streaming indicator
}
```

### 2.3 Middleware Execution Infrastructure

From [go/agent/middleware_agent.go](go/agent/middleware_agent.go):

```go
type MiddlewareAgent struct {
    inner      Agent
    middleware AgentMiddleware
}

func NewMiddlewareAgent(inner Agent, middlewares ...AgentMiddleware) *MiddlewareAgent
```

The `MiddlewareAgent` wraps an inner agent and applies middleware chain for both `Run` and `RunStream` operations.

### 2.4 Middleware Chaining

From [go/agent/chain.go](go/agent/chain.go):

```go
func ChainAgentMiddleware(middlewares ...AgentMiddleware) AgentMiddleware
func ChainFunctionMiddleware(middlewares ...FunctionMiddleware) FunctionMiddleware
func ChainChatMiddleware(middlewares ...ChatMiddleware) ChatMiddleware
```

---

## 3. .NET OpenTelemetryAgent Reference Analysis

From [dotnet/src/Microsoft.Agents.AI/OpenTelemetryAgent.cs](dotnet/src/Microsoft.Agents.AI/OpenTelemetryAgent.cs):

### 3.1 Key Implementation Patterns

1. **Decorating Pattern**: `OpenTelemetryAgent` extends `DelegatingAIAgent`, wrapping an inner agent
2. **Delegation to OpenTelemetryChatClient**: Leverages existing chat-level telemetry
3. **Activity Augmentation**: Updates current `Activity` with agent-specific attributes
4. **Sensitive Data Control**: `EnableSensitiveData` property controls content logging

### 3.2 Telemetry Data Captured

- Operation name: `invoke_agent`
- Agent ID, name, description
- Provider name
- All chat-level telemetry (tokens, latency, finish reason)

### 3.3 Builder Extension

```csharp
builder.UseOpenTelemetry(sourceName, configure)
```

---

## 4. OpenTelemetry Middleware Design Proposal

### 4.1 Design Goals

1. Implement as `AgentMiddleware` for composability
2. Provide both middleware and agent wrapper options
3. Follow Go idioms (functional options, interfaces)
4. Capture comprehensive telemetry data
5. Support streaming operations
6. Align with .NET implementation semantically

### 4.2 Proposed Interface

```go
// File: go/observability/middleware.go

package observability

import (
    "context"
    "time"

    "github.com/microsoft/agent-framework-go/agent"
    "go.opentelemetry.io/otel/attribute"
    "go.opentelemetry.io/otel/trace"
)

// TelemetryMiddleware implements AgentMiddleware for OpenTelemetry instrumentation.
type TelemetryMiddleware struct {
    metrics             *Metrics
    enableSensitiveData bool
    sourceName          string
}

// TelemetryMiddlewareOption configures TelemetryMiddleware.
type TelemetryMiddlewareOption func(*TelemetryMiddleware)

// WithTelemetrySensitiveData enables logging of message content.
func WithTelemetrySensitiveData(enabled bool) TelemetryMiddlewareOption {
    return func(m *TelemetryMiddleware) {
        m.enableSensitiveData = enabled
    }
}

// WithTelemetryMetrics sets custom metrics.
func WithTelemetryMetrics(metrics *Metrics) TelemetryMiddlewareOption {
    return func(m *TelemetryMiddleware) {
        m.metrics = metrics
    }
}

// WithTelemetrySourceName sets the telemetry source name.
func WithTelemetrySourceName(name string) TelemetryMiddlewareOption {
    return func(m *TelemetryMiddleware) {
        m.sourceName = name
    }
}

// NewTelemetryMiddleware creates OpenTelemetry instrumentation middleware.
func NewTelemetryMiddleware(opts ...TelemetryMiddlewareOption) *TelemetryMiddleware {
    m := &TelemetryMiddleware{
        sourceName: InstrumentationName,
    }
    for _, opt := range opts {
        opt(m)
    }
    if m.metrics == nil {
        m.metrics, _ = NewMetrics()
    }
    return m
}

// Process implements AgentMiddleware.
func (m *TelemetryMiddleware) Process(ctx context.Context, agentCtx *agent.AgentContext, next agent.AgentHandler) error {
    ag := agentCtx.Agent
    metadata := ag.Metadata()

    // Start agent span
    ctx, span := m.startAgentSpan(ctx, ag, metadata)
    defer span.End()

    start := time.Now()

    // Record agent run metric
    if m.metrics != nil {
        m.metrics.RecordAgentRun(ctx, metadata.ProviderName, metadata.ModelID)
    }

    // Execute next handler
    err := next(ctx, agentCtx)

    // Record latency
    latency := time.Since(start).Seconds()
    if m.metrics != nil {
        m.metrics.RecordLatency(ctx, latency, metadata.ProviderName, metadata.ModelID)
    }

    // Handle error
    if err != nil {
        RecordError(span, err)
        if m.metrics != nil {
            m.metrics.RecordError(ctx, errorTypeName(err), metadata.ProviderName, metadata.ModelID)
        }
        return err
    }

    // Record response details for non-streaming
    if !agentCtx.IsStreaming && agentCtx.Response != nil {
        m.recordResponse(ctx, span, agentCtx.Response, metadata)
    }

    // For streaming, wrap the stream to capture completion
    if agentCtx.IsStreaming && agentCtx.Stream != nil {
        agentCtx.Stream = m.wrapStream(ctx, span, agentCtx.Stream, metadata, start)
    }

    return nil
}

func (m *TelemetryMiddleware) startAgentSpan(ctx context.Context, ag agent.Agent, metadata agent.AIAgentMetadata) (context.Context, trace.Span) {
    spanName := OperationAgentRun
    if ag.Name() != "" {
        spanName = OperationAgentRun + " " + ag.Name()
    }

    attrs := []attribute.KeyValue{
        attribute.String(GenAIOperationNameKey, OperationAgentRun),
        attribute.String(AgentIDKey, ag.ID()),
    }

    if ag.Name() != "" {
        attrs = append(attrs, attribute.String(AgentNameKey, ag.Name()))
    }
    if ag.Description() != "" {
        attrs = append(attrs, attribute.String(GenAIAgentDescriptionKey, ag.Description()))
    }
    if metadata.ProviderName != "" {
        attrs = append(attrs, attribute.String(AgentProviderKey, metadata.ProviderName))
    }
    if metadata.ModelID != "" {
        attrs = append(attrs, attribute.String(GenAIRequestModelKey, metadata.ModelID))
    }

    return Tracer().Start(ctx, spanName,
        trace.WithSpanKind(trace.SpanKindClient),
        trace.WithAttributes(attrs...),
    )
}

func (m *TelemetryMiddleware) recordResponse(ctx context.Context, span trace.Span, resp *agent.Response, metadata agent.AIAgentMetadata) {
    if resp.Usage != nil {
        RecordUsage(span, resp.Usage.InputTokens, resp.Usage.OutputTokens)
        if m.metrics != nil {
            m.metrics.RecordTokenUsage(ctx, resp.Usage.InputTokens, resp.Usage.OutputTokens, metadata.ProviderName, metadata.ModelID)
        }
    }
    if resp.FinishReason != "" {
        span.SetAttributes(attribute.String(GenAIResponseFinishReasonsKey, string(resp.FinishReason)))
    }
}

func (m *TelemetryMiddleware) wrapStream(ctx context.Context, span trace.Span, stream <-chan agent.ResponseUpdate, metadata agent.AIAgentMetadata, start time.Time) <-chan agent.ResponseUpdate {
    wrapped := make(chan agent.ResponseUpdate, 32)
    go func() {
        defer close(wrapped)

        var totalInput, totalOutput int
        var finishReason string

        for update := range stream {
            // Capture usage
            if update.Usage != nil {
                totalInput = update.Usage.InputTokens
                totalOutput = update.Usage.OutputTokens
            }
            // Capture finish reason
            if update.FinishReason != "" {
                finishReason = string(update.FinishReason)
            }
            // Handle errors
            if update.Error != nil {
                RecordError(span, update.Error)
                if m.metrics != nil {
                    m.metrics.RecordError(ctx, errorTypeName(update.Error), metadata.ProviderName, metadata.ModelID)
                }
            }

            wrapped <- update
        }

        // Record final metrics
        latency := time.Since(start).Seconds()
        if m.metrics != nil {
            m.metrics.RecordLatency(ctx, latency, metadata.ProviderName, metadata.ModelID)
            if totalInput > 0 || totalOutput > 0 {
                m.metrics.RecordTokenUsage(ctx, totalInput, totalOutput, metadata.ProviderName, metadata.ModelID)
            }
        }
        if totalInput > 0 || totalOutput > 0 {
            RecordUsage(span, totalInput, totalOutput)
        }
        if finishReason != "" {
            span.SetAttributes(attribute.String(GenAIResponseFinishReasonsKey, finishReason))
        }
    }()
    return wrapped
}

// Ensure interface compliance
var _ agent.AgentMiddleware = (*TelemetryMiddleware)(nil)
```

### 4.3 Builder Extension

```go
// File: go/observability/builder.go

package observability

import (
    "github.com/microsoft/agent-framework-go/agent"
)

// UseOpenTelemetry adds OpenTelemetry instrumentation to an agent builder.
// This creates a TelemetryMiddleware and adds it to the middleware chain.
func UseOpenTelemetry(builder interface{ Use(...agent.AgentMiddleware) }, opts ...TelemetryMiddlewareOption) {
    builder.Use(NewTelemetryMiddleware(opts...))
}
```

### 4.4 Agent Wrapper (Alternative API)

```go
// File: go/observability/telemetry_agent.go

package observability

import (
    "github.com/microsoft/agent-framework-go/agent"
)

// TelemetryAgent wraps an agent with OpenTelemetry instrumentation.
// This is a convenience wrapper around MiddlewareAgent with TelemetryMiddleware.
type TelemetryAgent struct {
    *agent.MiddlewareAgent
    middleware *TelemetryMiddleware
}

// NewTelemetryAgent creates an agent with OpenTelemetry instrumentation.
func NewTelemetryAgent(inner agent.Agent, opts ...TelemetryMiddlewareOption) *TelemetryAgent {
    mw := NewTelemetryMiddleware(opts...)
    return &TelemetryAgent{
        MiddlewareAgent: agent.NewMiddlewareAgent(inner, mw),
        middleware:      mw,
    }
}

// EnableSensitiveData controls whether message content is included in traces.
func (a *TelemetryAgent) EnableSensitiveData(enabled bool) {
    a.middleware.enableSensitiveData = enabled
}
```

---

## 5. Telemetry Data to Capture

### 5.1 Spans

| Span Name | Kind | When Created |
|-----------|------|--------------|
| `agent.run {name}` | Client | Agent `Run`/`RunStream` invocation |
| `{provider} chat` | Client | Chat client requests (via InstrumentedClient) |
| `tool.call {name}` | Internal | Function/tool invocations |

### 5.2 Span Attributes

| Attribute | Type | Description |
|-----------|------|-------------|
| `gen_ai.operation.name` | string | Operation type (agent.run, chat, tool.call) |
| `gen_ai.system` | string | AI provider system name |
| `gen_ai.request.model` | string | Model ID requested |
| `gen_ai.response.model` | string | Model ID returned |
| `gen_ai.usage.input_tokens` | int | Input token count |
| `gen_ai.usage.output_tokens` | int | Output token count |
| `gen_ai.response.finish_reasons` | string | Why generation stopped |
| `agent.id` | string | Unique agent identifier |
| `agent.name` | string | Human-readable agent name |
| `gen_ai.agent.description` | string | Agent description |
| `agent.provider` | string | LLM provider name |
| `gen_ai.tool.name` | string | Tool/function name |
| `gen_ai.tool.call_id` | string | Tool call identifier |
| `gen_ai.error.type` | string | Error type on failure |
| `gen_ai.error.message` | string | Error message on failure |

### 5.3 Metrics

| Metric | Type | Attributes | Description |
|--------|------|------------|-------------|
| `gen_ai.agent.runs` | Counter | provider, model | Agent run count |
| `gen_ai.usage.input_tokens` | Histogram | provider, model | Input tokens distribution |
| `gen_ai.usage.output_tokens` | Histogram | provider, model | Output tokens distribution |
| `gen_ai.request.latency` | Histogram | provider, model | Latency distribution (seconds) |
| `gen_ai.errors` | Counter | error_type, provider, model | Error count |
| `gen_ai.tool.invocations` | Counter | tool_name, success | Tool invocation count |

### 5.4 Optional Sensitive Data

When `EnableSensitiveData` is true:

- Message content in span events
- Function call arguments
- Function call results

---

## 6. Implementation Approach

### 6.1 Phased Implementation

#### Phase 1: Core TelemetryMiddleware

1. Create `go/observability/middleware.go` with `TelemetryMiddleware`
2. Implement `Process` method for non-streaming operations
3. Add streaming support with channel wrapping
4. Write comprehensive unit tests

#### Phase 2: FunctionMiddleware for Tool Calls

1. Create `TelemetryFunctionMiddleware` implementing `FunctionMiddleware`
2. Start tool call spans
3. Record tool invocation metrics
4. Capture arguments/results when sensitive data enabled

#### Phase 3: Builder Integration

1. Add `UseOpenTelemetry` helper function
2. Create `TelemetryAgent` convenience wrapper
3. Update chatagent builder to support middleware

#### Phase 4: ChatMiddleware Layer (Optional)

1. Create `TelemetryChatMiddleware` for chat-level instrumentation
2. Alternative to `InstrumentedClient` wrapper pattern

### 6.2 File Structure

```
go/observability/
├── doc.go                    # Package documentation
├── instrumented.go           # Existing InstrumentedClient
├── instrumented_test.go
├── metrics.go                # Metrics instruments
├── metrics_test.go
├── middleware.go             # NEW: TelemetryMiddleware
├── middleware_test.go        # NEW: Middleware tests
├── otel.go                   # Core OTel functions
├── otel_test.go
├── semconv.go                # Semantic conventions
├── semconv_test.go
├── setup.go                  # SDK setup
├── setup_test.go
├── telemetry_agent.go        # NEW: Convenience wrapper
└── telemetry_agent_test.go   # NEW: Agent wrapper tests
```

---

## 7. Benefits of Middleware-Based Observability

### 7.1 Comparison: Current vs Middleware Approach

| Aspect | Current Approach | Middleware Approach |
|--------|------------------|---------------------|
| **Composability** | Manual wrapping required | Chain with other middleware |
| **Flexibility** | Fixed instrumentation | Configurable via options |
| **Integration** | Separate API | Uses standard middleware API |
| **Streaming** | Built into wrapper | Handled in Process method |
| **Consistency** | Different pattern than other cross-cutting concerns | Same pattern as logging, security, etc. |
| **Testing** | Test wrapper directly | Mock middleware chain |

### 7.2 Advantages of Middleware Pattern

1. **Separation of Concerns**: Observability logic isolated from agent implementation
2. **Composability**: Combine with logging, security, caching middleware
3. **Ordering Control**: Position telemetry at start or end of chain
4. **Optional Layers**: Add agent, function, and chat middleware as needed
5. **Consistency**: Same pattern across .NET, Python, and Go implementations
6. **Testability**: Easy to mock and test in isolation

### 7.3 When to Use Each Approach

| Use Case | Recommended Approach |
|----------|---------------------|
| Quick instrumentation of chat client | `InstrumentedClient` |
| Full agent observability | `TelemetryMiddleware` |
| Custom middleware chain | `TelemetryMiddleware` |
| Builder-based configuration | `UseOpenTelemetry` extension |
| Simple agent wrapper | `TelemetryAgent` |

---

## 8. Usage Examples

### 8.1 Using TelemetryMiddleware Directly

```go
import (
    "github.com/microsoft/agent-framework-go/agent"
    "github.com/microsoft/agent-framework-go/chatagent"
    "github.com/microsoft/agent-framework-go/observability"
)

// Create base agent
baseAgent := chatagent.New(chatClient)

// Wrap with telemetry middleware
instrumentedAgent := agent.NewMiddlewareAgent(
    baseAgent,
    observability.NewTelemetryMiddleware(
        observability.WithTelemetrySensitiveData(false),
    ),
)

// Use the instrumented agent
resp, err := instrumentedAgent.Run(ctx, messages)
```

### 8.2 Using TelemetryAgent Wrapper

```go
// Create instrumented agent directly
agent := observability.NewTelemetryAgent(
    chatagent.New(chatClient),
    observability.WithTelemetrySensitiveData(true),
)

resp, err := agent.Run(ctx, messages)
```

### 8.3 Using Builder Extension

```go
builder := chatagent.NewBuilder(chatClient).
    WithName("MyAgent").
    WithInstructions("You are helpful.")

observability.UseOpenTelemetry(builder,
    observability.WithTelemetrySensitiveData(false),
)

agent := builder.Build()
```

### 8.4 Combining Multiple Middleware

```go
instrumentedAgent := agent.NewMiddlewareAgent(
    baseAgent,
    loggingMiddleware,           // First: log start
    observability.NewTelemetryMiddleware(), // Second: start span
    securityMiddleware,          // Third: validate
    rateLimitMiddleware,         // Fourth: rate limit
)
```

---

## 9. Implementation Checklist

- [ ] Create `TelemetryMiddleware` in `go/observability/middleware.go`
- [ ] Implement non-streaming `Process` method
- [ ] Implement streaming support with channel wrapping
- [ ] Add `TelemetryMiddlewareOption` functions
- [ ] Create `TelemetryAgent` convenience wrapper
- [ ] Add `UseOpenTelemetry` builder extension
- [ ] Write unit tests for middleware
- [ ] Write integration tests with mock tracer
- [ ] Update package documentation
- [ ] Add sample demonstrating usage

---

## 10. References

- [OpenTelemetry Semantic Conventions for GenAI](https://opentelemetry.io/docs/specs/semconv/gen-ai/)
- [Go Agent Middleware Interface](go/agent/middleware.go)
- [Current Observability Package](go/observability/)
- [.NET OpenTelemetryAgent](dotnet/src/Microsoft.Agents.AI/OpenTelemetryAgent.cs)
- [Middleware Research](../research/2026-02-03-go-middleware-patterns-research.md)
