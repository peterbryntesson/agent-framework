# Context Provider Codebase Research

**Date**: 2026-02-03
**Purpose**: Research the Go agent framework codebase to understand patterns for implementing ContextProvider

---

## 1. Package Structure

### go/agent/ Directory (28 files)

| File | Purpose |
|------|---------|
| `agent.go` | Core `Agent` interface definition |
| `builder.go` | Generic agent builder utilities |
| `chain.go` | Middleware chaining functions |
| `delegating.go` | Delegating agent wrapper |
| `doc.go` | Package documentation |
| `errors.go` | Error types and handling |
| `message.go` | Message type definitions |
| `metadata.go` | Agent metadata types |
| `middleware.go` | Core middleware interface definitions |
| `middleware_agent.go` | MiddlewareAgent wrapper implementation |
| `middleware_context.go` | Context structs for middleware |
| `options.go` | RunOption functional options |
| `response.go` | Response and ResponseUpdate types |
| `response_extensions.go` | Response helper methods |
| `session.go` | Session interface and InMemorySession |
| `*_test.go` | Unit tests |

### go/chatagent/ Directory (14 files)

| File | Purpose |
|------|---------|
| `agent.go` | ChatClientAgent implementation |
| `astool.go` | Agent-as-tool functionality |
| `builder.go` | Fluent builder for ChatClientAgent |
| `doc.go` | Package documentation |
| `options.go` | Option functional options |
| `session.go` | Session implementation |
| `toolloop.go` | Tool invocation loop logic |
| `*_test.go` | Unit tests |

---

## 2. Middleware Pattern

### Middleware Interface Definitions ([middleware.go#L1-80](go/agent/middleware.go))

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

**Three Middleware Types** (Lines 9-75):

1. **AgentMiddleware** (Lines 9-27): Intercepts agent `Run`/`RunStream` calls
2. **FunctionMiddleware** (Lines 32-48): Intercepts tool/function invocations
3. **ChatMiddleware** (Lines 53-75): Intercepts chat client requests (GetResponse/GetStreamingResponse)

### Middleware Chaining ([chain.go#L1-97](go/agent/chain.go))

```go
// ChainAgentMiddleware composes multiple AgentMiddleware into a single middleware.
func ChainAgentMiddleware(middlewares ...AgentMiddleware) AgentMiddleware
func ChainFunctionMiddleware(middlewares ...FunctionMiddleware) FunctionMiddleware
func ChainChatMiddleware(middlewares ...ChatMiddleware) ChatMiddleware
```

**Pattern**: Chain builds from last to first, creating nested handlers where first middleware is outermost.

---

## 3. Context Structures

### AgentContext ([middleware_context.go#L9-38](go/agent/middleware_context.go))

```go
type AgentContext struct {
    Agent       Agent              // The agent being invoked
    Messages    []Message          // Input messages for this invocation
    Session     Session            // Optional session for this invocation
    Options     *RunConfig         // Run configuration
    Metadata    map[string]any     // Middleware can attach arbitrary data
    Response    *Response          // Result for non-streaming invocations
    Stream      <-chan ResponseUpdate  // Result channel for streaming
    IsStreaming bool               // Whether this is streaming
}
```

### FunctionContext ([middleware_context.go#L40-54](go/agent/middleware_context.go))

```go
type FunctionContext struct {
    FunctionName string           // Name of the function being invoked
    Arguments    json.RawMessage  // JSON arguments for the function
    Metadata     map[string]any   // Middleware can attach arbitrary data
    Result       any              // Function result after invocation
    Error        error            // Any error from the invocation
}
```

### ChatContext ([middleware_context.go#L56-85](go/agent/middleware_context.go))

```go
type ChatContext struct {
    ClientMetadata ChatClientMetadata  // Info about the chat client
    Messages       []Message           // Input messages for this request
    Options        map[string]any      // Chat request options
    Metadata       map[string]any      // Middleware can attach arbitrary data
    Response       *ChatResponse       // Result for non-streaming
    Stream         <-chan ChatResponseUpdate  // Result channel for streaming
    IsStreaming    bool                // Whether this is streaming
}
```

**Key Pattern**: All contexts have a `Metadata map[string]any` field for middleware to attach arbitrary data.

---

## 4. Agent Run Flow

### Entry Point: Agent.Run ([chatagent/agent.go#L95-105](go/chatagent/agent.go))

```go
func (a *Agent) Run(ctx context.Context, messages []agent.Message, opts ...agent.RunOption) (*agent.Response, error) {
    cfg := agent.ApplyRunOptions(opts...)
    chatMessages := a.prepareMessages(messages, cfg)
    chatOptions := a.prepareChatOptions(cfg)
    return a.runWithToolLoop(ctx, chatMessages, chatOptions, cfg)
}
```

### Tool Loop: runWithToolLoop ([chatagent/toolloop.go#L16-83](go/chatagent/toolloop.go))

```go
func (a *Agent) runWithToolLoop(ctx context.Context, messages []chat.Message, options *chat.Options, cfg *agent.RunConfig) (*agent.Response, error) {
    allTools := a.getAllTools(cfg)
    invoker := tool.NewInvoker(allTools, a.invocationConfig)
    currentMessages := messages
    
    for turn := 0; turn < a.maxTurns; turn++ {
        // 1. Check context cancellation
        if err := ctx.Err(); err != nil { return nil, err }
        
        // 2. Get response through ChatMiddleware (Line 33)
        resp, _, err := a.invokeWithChatMiddleware(ctx, currentMessages, options, false)
        
        // 3. Check for tool calls (Line 45)
        if len(resp.Message.ToolCalls) == 0 || resp.FinishReason != chat.FinishReasonToolCalls {
            return a.buildResponse(allResponseMessages, totalUsage, resp.FinishReason), nil
        }
        
        // 4. Execute tool calls through FunctionMiddleware (Line 56)
        toolResults, err := a.invokeToolCalls(ctx, invoker, resp.Message.ToolCalls)
        
        // 5. Append to conversation for next turn
        currentMessages = append(currentMessages, resp.Message)
        for _, result := range toolResults {
            toolMsg := chat.NewToolMessage(result.CallID, result.Content)
            currentMessages = append(currentMessages, toolMsg)
        }
    }
    return nil, tool.ErrMaxIterations
}
```

### Chat Middleware Invocation ([chatagent/toolloop.go#L452-498](go/chatagent/toolloop.go))

```go
func (a *Agent) invokeWithChatMiddleware(ctx context.Context, messages []chat.Message, options *chat.Options, isStreaming bool) (*chat.Response, <-chan chat.ResponseUpdate, error) {
    // Build ChatContext for middleware
    chatCtx := &agent.ChatContext{
        ClientMetadata: agent.ChatClientMetadata{...},
        Messages:    convertMessagesToAgentMessages(messages),
        Options:     convertChatOptionsToMap(options),
        Metadata:    make(map[string]any),
        IsStreaming: isStreaming,
    }

    // Terminal handler calls the actual chat client
    terminal := func(ctx context.Context, c *agent.ChatContext) error {
        if c.IsStreaming {
            stream, err := a.client.GetStreamingResponse(ctx, messages, options)
            c.Stream = wrapStreamChannel(stream)
        } else {
            resp, err := a.client.GetResponse(ctx, messages, options)
            c.Response = convertResponseToMiddlewareResponse(resp)
        }
        return nil
    }

    // Execute through middleware chain
    err := a.chatMiddleware.Process(ctx, chatCtx, terminal)
    ...
}
```

### Function Middleware Invocation ([chatagent/toolloop.go#L347-395](go/chatagent/toolloop.go))

```go
func (a *Agent) invokeSingleToolCall(ctx context.Context, invoker *tool.Invoker, tc chat.ToolCall) (toolCallResult, error) {
    // Build function context
    funcCtx := &agent.FunctionContext{
        FunctionName: tc.Name,
        Arguments:    tc.Arguments,
        Metadata:     make(map[string]any),
    }

    // Terminal handler calls the actual invoker
    terminal := func(ctx context.Context, funcCtx *agent.FunctionContext) error {
        result, err := invoker.Invoke(ctx, funcCtx.FunctionName, funcCtx.Arguments)
        funcCtx.Result = result
        funcCtx.Error = err
        return nil
    }

    // Execute through middleware chain if configured
    if a.functionMiddleware != nil {
        err := a.functionMiddleware.Process(ctx, funcCtx, terminal)
        ...
    }
    ...
}
```

---

## 5. Option Pattern

### Functional Option Pattern ([chatagent/options.go#L1-164](go/chatagent/options.go))

```go
// config holds private configuration
type config struct {
    id                 string
    name               string
    description        string
    instructions       string
    tools              []tool.Tool
    maxTurns           int
    invocationConfig   tool.InvocationConfig
    functionMiddleware []agent.FunctionMiddleware
    chatMiddleware     []agent.ChatMiddleware
}

// Option is a functional option for configuring a ChatClientAgent.
type Option func(*config)

// defaultConfig returns a config with sensible default values.
func defaultConfig() *config {
    return &config{
        maxTurns:         10,
        invocationConfig: tool.DefaultInvocationConfig(),
    }
}
```

### Option Examples (Lines 33-164)

```go
func WithID(id string) Option {
    return func(c *config) { c.id = id }
}

func WithName(name string) Option {
    return func(c *config) { c.name = name }
}

func WithInstructions(instructions string) Option {
    return func(c *config) { c.instructions = instructions }
}

func WithTools(tools ...tool.Tool) Option {
    return func(c *config) { c.tools = append(c.tools, tools...) }
}

func WithFunctionMiddleware(middlewares ...agent.FunctionMiddleware) Option {
    return func(c *config) { c.functionMiddleware = append(c.functionMiddleware, middlewares...) }
}

func WithChatMiddleware(middlewares ...agent.ChatMiddleware) Option {
    return func(c *config) { c.chatMiddleware = append(c.chatMiddleware, middlewares...) }
}
```

### Run Options ([agent/options.go#L1-84](go/agent/options.go))

```go
type RunOption func(*RunConfig)

type RunConfig struct {
    Session     Session
    Metadata    map[string]interface{}
    Tools       []interface{}
    MaxTokens   int
    Temperature float32
}

func WithSession(session Session) RunOption { ... }
func WithTools(tools ...interface{}) RunOption { ... }
func WithMaxTokens(maxTokens int) RunOption { ... }
func WithTemperature(temperature float32) RunOption { ... }
func WithMetadata(metadata map[string]interface{}) RunOption { ... }
```

---

## 6. Session Handling

### Session Interface ([agent/session.go#L14-30](go/agent/session.go))

```go
type Session interface {
    ID() string                           // Unique identifier
    Messages() []Message                   // Conversation history
    AddMessage(msg Message)                // Append a message
    Serialize() (json.RawMessage, error)   // Persist state
    GetService(serviceType reflect.Type) interface{}  // Service locator
}
```

### ChatAgent Session ([chatagent/session.go#L15-28](go/chatagent/session.go))

```go
type Session struct {
    id         string
    agentID    string
    messages   []chat.Message
    services   map[reflect.Type]interface{}  // Service registry
    mu         sync.RWMutex                  // Thread-safety
    createdAt  time.Time
    modifiedAt time.Time
}
```

### Session in Agent Run ([chatagent/agent.go#L161-178](go/chatagent/agent.go))

```go
func (a *Agent) prepareMessages(messages []agent.Message, cfg *agent.RunConfig) []chat.Message {
    capacity := len(messages) + 1
    if cfg.Session != nil {
        capacity += len(cfg.Session.Messages())
    }
    chatMessages := make([]chat.Message, 0, capacity)
    
    // Add system instructions
    if a.instructions != "" {
        chatMessages = append(chatMessages, chat.NewSystemMessage(a.instructions))
    }
    
    // Add session history if provided
    if cfg.Session != nil {
        chatMessages = append(chatMessages, cfg.Session.Messages()...)
    }
    
    // Add new messages
    chatMessages = append(chatMessages, messages...)
    return chatMessages
}
```

---

## 7. Integration Points for ContextProvider

### Recommended Integration Points

#### Option 1: AgentMiddleware (Highest Level)

**Location**: Wrap around `Agent.Run` / `Agent.RunStream`

**Pros**:
- Single injection point
- Access to full agent context
- Can modify messages before any processing

**Integration Pattern**:
```go
// In chatagent/options.go - Add new option
func WithContextProviders(providers ...ContextProvider) Option {
    return func(c *config) {
        c.contextProviders = append(c.contextProviders, providers...)
    }
}

// Create AgentMiddleware that invokes providers
type contextProviderMiddleware struct {
    providers []ContextProvider
}

func (m *contextProviderMiddleware) Process(ctx context.Context, agentCtx *AgentContext, next AgentHandler) error {
    // Get context from providers
    for _, p := range m.providers {
        additionalContext := p.GetContext(ctx, agentCtx)
        // Inject into messages or metadata
    }
    return next(ctx, agentCtx)
}
```

#### Option 2: Before runWithToolLoop (Agent Level)

**Location**: [chatagent/agent.go#L102](go/chatagent/agent.go#L102) - After `prepareMessages`, before `runWithToolLoop`

**Integration Pattern**:
```go
func (a *Agent) Run(ctx context.Context, messages []agent.Message, opts ...agent.RunOption) (*agent.Response, error) {
    cfg := agent.ApplyRunOptions(opts...)
    chatMessages := a.prepareMessages(messages, cfg)
    
    // NEW: Inject context from providers
    if len(a.contextProviders) > 0 {
        additionalMessages := a.getProviderContext(ctx, messages, cfg)
        chatMessages = a.injectContextMessages(chatMessages, additionalMessages)
    }
    
    chatOptions := a.prepareChatOptions(cfg)
    return a.runWithToolLoop(ctx, chatMessages, chatOptions, cfg)
}
```

#### Option 3: ChatMiddleware (Per-Request Level)

**Location**: [chatagent/toolloop.go#L452-498](go/chatagent/toolloop.go#L452-498) - `invokeWithChatMiddleware`

**Pros**:
- Executes on each LLM call
- Can adapt context based on conversation state

**Cons**:
- May add context redundantly across tool loop iterations

### Key Injection Points Summary

| Integration Point | File:Line | When Called | Best For |
|-------------------|-----------|-------------|----------|
| Agent creation | `chatagent/agent.go:40-65` | Once at construction | Static context providers |
| `prepareMessages` | `chatagent/agent.go:161-178` | Each Run call | Session-based context |
| `runWithToolLoop` start | `chatagent/toolloop.go:16-25` | Each Run call | Dynamic context injection |
| `invokeWithChatMiddleware` | `chatagent/toolloop.go:452-498` | Each LLM request | Per-request context |

### Recommended Approach

1. **Define ContextProvider interface** in `go/agent/context_provider.go`
2. **Add `WithContextProviders` option** in `go/chatagent/options.go`
3. **Store providers in Agent struct** in `go/chatagent/agent.go`
4. **Invoke providers in `prepareMessages`** or at start of `runWithToolLoop`
5. **Inject resulting context as system messages** before user messages

### Code Pattern to Follow

Based on existing patterns, ContextProvider should:

1. Use the `map[string]any` Metadata pattern for passing data through the chain
2. Follow the functional option pattern (`WithContextProviders`)
3. Support both synchronous and streaming flows
4. Be invoked early in the message preparation phase
5. Use the existing middleware chaining infrastructure if needed

---

## 8. Key Code References

| Pattern | File | Line Range |
|---------|------|------------|
| Middleware interfaces | `go/agent/middleware.go` | 1-80 |
| Context structures | `go/agent/middleware_context.go` | 1-115 |
| Middleware chaining | `go/agent/chain.go` | 1-97 |
| Functional options | `go/chatagent/options.go` | 1-164 |
| Agent construction | `go/chatagent/agent.go` | 40-65 |
| Message preparation | `go/chatagent/agent.go` | 161-178 |
| Tool loop | `go/chatagent/toolloop.go` | 16-83 |
| Chat middleware invocation | `go/chatagent/toolloop.go` | 452-498 |
| Function middleware invocation | `go/chatagent/toolloop.go` | 347-395 |
| Session interface | `go/agent/session.go` | 14-30 |
| Run options | `go/agent/options.go` | 1-84 |
