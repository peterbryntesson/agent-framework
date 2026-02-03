<!-- markdownlint-disable-file -->
# Implementation Details: Go ChatMiddleware for Chat Client Interception

## Context Reference

Sources:

* [.copilot-tracking/research/2026-02-03-go-middleware-patterns-research.md](.copilot-tracking/research/2026-02-03-go-middleware-patterns-research.md) - Original middleware design research
* [.copilot-tracking/reviews/2026-02-03-go-middleware-patterns-review.md](.copilot-tracking/reviews/2026-02-03-go-middleware-patterns-review.md) - Review identifying ChatMiddleware as follow-up
* [.copilot-tracking/subagent/2026-02-03/python-chatmiddleware-research.md](.copilot-tracking/subagent/2026-02-03/python-chatmiddleware-research.md) - Python implementation reference
* [go/agent/middleware.go](go/agent/middleware.go) (Lines 1-50) - Existing middleware interface patterns
* [go/agent/middleware_context.go](go/agent/middleware_context.go) (Lines 1-60) - Existing context struct patterns
* [go/chat/client.go](go/chat/client.go) (Lines 1-120) - Chat Client interface definition

## Implementation Phase 1: ChatMiddleware Core Types

<!-- parallelizable: false -->

### Step 1.1: Create ChatContext struct in agent package

Add ChatContext struct to `go/agent/middleware_context.go` following the existing pattern from AgentContext and FunctionContext. ChatContext holds context for chat client middleware invocations.

Files:

* `go/agent/middleware_context.go` - Add ChatContext struct

Success criteria:

* ChatContext struct captures chat client metadata, messages, options, and response fields
* Supports both streaming and non-streaming via IsStreaming flag
* Includes Metadata map for middleware data sharing
* Follows existing context struct patterns

Context references:

* go/agent/middleware_context.go (Lines 10-35) - AgentContext pattern
* go/chat/client.go (Lines 17-32) - Client interface methods
* python-chatmiddleware-research.md (Lines 100-112) - Python ChatContext fields

Implementation:

```go
// ChatContext holds context for chat middleware invocations.
// It provides access to the chat client request parameters
// and fields for capturing the response.
type ChatContext struct {
    // ClientMetadata contains information about the chat client.
    ClientMetadata ClientMetadata

    // Messages contains the input messages for this request.
    Messages []Message

    // Options contains chat request options (temperature, max_tokens, etc.).
    Options map[string]any

    // Metadata allows middleware to attach arbitrary data.
    Metadata map[string]any

    // Response holds the result for non-streaming invocations.
    // Set by the terminal handler or by middleware to override the response.
    Response *ChatResponse

    // Stream holds the result channel for streaming invocations.
    // Set by the terminal handler when IsStreaming is true.
    Stream <-chan ChatResponseUpdate

    // IsStreaming indicates whether this is a streaming invocation.
    IsStreaming bool
}

// ClientMetadata mirrors chat.ClientMetadata for use in middleware context.
type ClientMetadata struct {
    ProviderName string
    ModelID      string
    EndpointURI  string
}

// ChatResponse wraps chat.Response for middleware context.
type ChatResponse = chat.Response

// ChatResponseUpdate wraps chat.ResponseUpdate for middleware context.
type ChatResponseUpdate = chat.ResponseUpdate
```

Note: Consider importing from chat package vs duplicating types. For initial implementation, use type aliases to avoid circular imports while maintaining type safety.

Dependencies:

* None (first step)

### Step 1.2: Create ChatMiddleware interface in agent package

Add ChatMiddleware interface to `go/agent/middleware.go` following the existing AgentMiddleware and FunctionMiddleware pattern. ChatMiddleware intercepts chat client requests at a lower level than AgentMiddleware.

Files:

* `go/agent/middleware.go` - Add ChatMiddleware interface and ChatHandler type

Success criteria:

* ChatMiddleware interface follows same pattern as AgentMiddleware
* ChatHandler type defines the next handler signature
* Interface documentation explains use cases (request modification, caching, logging)
* Supports short-circuiting by not calling next

Context references:

* go/agent/middleware.go (Lines 9-28) - AgentMiddleware pattern
* go/agent/middleware.go (Lines 30-45) - FunctionMiddleware pattern

Implementation:

```go
// ChatHandler is the next handler in the chat middleware chain.
type ChatHandler func(ctx context.Context, chatCtx *ChatContext) error

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

// ChatMiddlewareFunc is a function adapter for ChatMiddleware.
// Use this to create middleware from anonymous functions.
type ChatMiddlewareFunc func(ctx context.Context, chatCtx *ChatContext, next ChatHandler) error

// Process implements ChatMiddleware.
func (f ChatMiddlewareFunc) Process(ctx context.Context, chatCtx *ChatContext, next ChatHandler) error {
    return f(ctx, chatCtx, next)
}
```

Dependencies:

* Step 1.1 completion (ChatContext struct)

### Step 1.3: Add ChatMiddlewareFunc adapter

Already included in Step 1.2. This step validates the adapter works correctly and provides example usage in documentation.

Files:

* `go/agent/middleware.go` - Verify ChatMiddlewareFunc implements ChatMiddleware

Success criteria:

* ChatMiddlewareFunc satisfies ChatMiddleware interface
* Interface satisfaction verified with compile-time check
* Example usage documented in doc comment

Context references:

* go/agent/middleware.go (Lines 22-28) - AgentMiddlewareFunc pattern

Implementation verification:

```go
// Compile-time interface satisfaction check
var _ ChatMiddleware = ChatMiddlewareFunc(nil)
```

Dependencies:

* Step 1.2 completion

### Step 1.4: Add ChainChatMiddleware composition function

Add ChainChatMiddleware function to `go/agent/chain.go` following the existing ChainAgentMiddleware and ChainFunctionMiddleware patterns.

Files:

* `go/agent/chain.go` - Add ChainChatMiddleware function

Success criteria:

* ChainChatMiddleware composes multiple ChatMiddleware into single ChatMiddleware
* Applies middlewares in correct order (first added = outermost)
* Returns passthrough for empty/nil input
* Follows existing chain function patterns

Context references:

* go/agent/chain.go (Lines 10-35) - ChainAgentMiddleware pattern
* go/agent/chain.go (Lines 37-60) - ChainFunctionMiddleware pattern

Implementation:

```go
// ChainChatMiddleware composes multiple ChatMiddleware into a single ChatMiddleware.
// The middlewares are executed in order, with the first middleware being the outermost.
// An empty or nil slice returns a passthrough middleware that calls next directly.
func ChainChatMiddleware(middlewares ...ChatMiddleware) ChatMiddleware {
    // Filter out nil middlewares
    filtered := make([]ChatMiddleware, 0, len(middlewares))
    for _, m := range middlewares {
        if m != nil {
            filtered = append(filtered, m)
        }
    }

    if len(filtered) == 0 {
        return ChatMiddlewareFunc(func(ctx context.Context, chatCtx *ChatContext, next ChatHandler) error {
            return next(ctx, chatCtx)
        })
    }

    if len(filtered) == 1 {
        return filtered[0]
    }

    return ChatMiddlewareFunc(func(ctx context.Context, chatCtx *ChatContext, next ChatHandler) error {
        // Build handler chain in reverse order
        handler := next
        for i := len(filtered) - 1; i >= 0; i-- {
            m := filtered[i]
            currentHandler := handler
            handler = func(ctx context.Context, chatCtx *ChatContext) error {
                return m.Process(ctx, chatCtx, currentHandler)
            }
        }
        return handler(ctx, chatCtx)
    })
}
```

Dependencies:

* Step 1.2 completion (ChatMiddleware interface)

### Step 1.5: Add unit tests for ChatMiddleware types

Create comprehensive unit tests for ChatMiddleware types and chain composition.

Files:

* `go/agent/chat_middleware_test.go` - New test file for ChatMiddleware tests

Success criteria:

* Tests verify ChatMiddlewareFunc implements ChatMiddleware
* Tests verify ChainChatMiddleware execution order
* Tests verify short-circuiting behavior
* Tests verify context modification flows through chain
* Tests cover both streaming and non-streaming scenarios

Context references:

* go/agent/middleware_agent_test.go - Existing middleware test patterns
* go/agent/chain_test.go - Existing chain test patterns

Implementation outline:

```go
package agent

import (
    "context"
    "testing"
)

func TestChatMiddlewareFunc_ImplementsInterface(t *testing.T) {
    var _ ChatMiddleware = ChatMiddlewareFunc(nil)
}

func TestChainChatMiddleware_Empty(t *testing.T) {
    chain := ChainChatMiddleware()
    // Verify passthrough behavior
}

func TestChainChatMiddleware_Single(t *testing.T) {
    // Verify single middleware is returned directly
}

func TestChainChatMiddleware_ExecutionOrder(t *testing.T) {
    // Verify middlewares execute in correct order
    // First added = outermost = first to receive request
}

func TestChainChatMiddleware_ShortCircuit(t *testing.T) {
    // Verify middleware can short-circuit by not calling next
}

func TestChatContext_StreamingFlag(t *testing.T) {
    // Verify IsStreaming flag is set correctly for streaming calls
}
```

Dependencies:

* Step 1.4 completion

### Step 1.6: Validate phase changes

Run validation commands for agent package.

Validation commands:

* `go build ./agent/...` - Compile check
* `go vet ./agent/...` - Static analysis
* `go test ./agent/...` - Unit tests

## Implementation Phase 2: ChatAgent Integration

<!-- parallelizable: false -->

### Step 2.1: Add ChatMiddleware options to chatagent package

Add functional options and agent struct fields for ChatMiddleware configuration.

Files:

* `go/chatagent/options.go` - Add WithChatMiddleware option
* `go/chatagent/agent.go` - Add chatMiddleware field to Agent struct

Success criteria:

* WithChatMiddleware option accepts variadic ChatMiddleware
* Agent struct stores chat middleware slice
* Option follows existing pattern from WithFunctionMiddleware

Context references:

* go/chatagent/options.go (Lines 80-95) - WithFunctionMiddleware pattern
* go/chatagent/agent.go (Lines 30-50) - Agent struct fields

Implementation for options.go:

```go
// WithChatMiddleware adds middleware to intercept chat client requests.
// Middleware is executed in order, with the first middleware being outermost.
// ChatMiddleware operates at a lower level than AgentMiddleware, intercepting
// each individual GetResponse or GetStreamingResponse call within the tool loop.
func WithChatMiddleware(middleware ...agent.ChatMiddleware) Option {
    return func(a *Agent) {
        a.chatMiddleware = append(a.chatMiddleware, middleware...)
    }
}
```

Implementation for agent.go:

```go
type Agent struct {
    // ... existing fields ...
    
    // chatMiddleware intercepts chat client requests.
    chatMiddleware []agent.ChatMiddleware
}
```

Dependencies:

* Phase 1 completion

### Step 2.2: Integrate ChatMiddleware into runWithToolLoop

Modify runWithToolLoop to wrap chat client calls with ChatMiddleware execution.

Files:

* `go/chatagent/toolloop.go` - Wrap GetResponse calls with middleware

Success criteria:

* ChatMiddleware is executed before each GetResponse call
* ChatContext is populated with request parameters
* Response is retrieved from ChatContext after middleware execution
* Middleware can modify messages, options, or short-circuit with cached response
* Original behavior preserved when no middleware configured

Context references:

* go/chatagent/toolloop.go (Lines 30-45) - Current GetResponse call location
* go/chatagent/toolloop.go (Lines 17-60) - runWithToolLoop structure

Implementation approach:

```go
// In runWithToolLoop, replace direct GetResponse call:

// Before:
// resp, err := a.client.GetResponse(ctx, currentMessages, options)

// After:
resp, err := a.invokeWithChatMiddleware(ctx, currentMessages, options, false)

// New helper method:
func (a *Agent) invokeWithChatMiddleware(
    ctx context.Context,
    messages []chat.Message,
    options *chat.Options,
    isStreaming bool,
) (*chat.Response, <-chan chat.ResponseUpdate, error) {
    if len(a.chatMiddleware) == 0 {
        if isStreaming {
            stream, err := a.client.GetStreamingResponse(ctx, messages, options)
            return nil, stream, err
        }
        resp, err := a.client.GetResponse(ctx, messages, options)
        return resp, nil, err
    }

    // Build ChatContext
    chatCtx := &agent.ChatContext{
        ClientMetadata: a.buildClientMetadata(),
        Messages:       convertMessages(messages),
        Options:        convertOptions(options),
        Metadata:       make(map[string]any),
        IsStreaming:    isStreaming,
    }

    // Build middleware chain
    chain := agent.ChainChatMiddleware(a.chatMiddleware...)
    
    // Terminal handler calls actual client
    terminal := func(ctx context.Context, c *agent.ChatContext) error {
        if c.IsStreaming {
            stream, err := a.client.GetStreamingResponse(ctx, revertMessages(c.Messages), revertOptions(c.Options))
            if err != nil {
                return err
            }
            c.Stream = stream
        } else {
            resp, err := a.client.GetResponse(ctx, revertMessages(c.Messages), revertOptions(c.Options))
            if err != nil {
                return err
            }
            c.Response = resp
        }
        return nil
    }

    err := chain.Process(ctx, chatCtx, terminal)
    if err != nil {
        return nil, nil, err
    }

    if isStreaming {
        return nil, chatCtx.Stream, nil
    }
    return chatCtx.Response, nil, nil
}
```

Note: Message and options type conversions may be needed. Consider using type aliases or direct references to avoid conversions if imports can be structured appropriately.

Dependencies:

* Step 2.1 completion

### Step 2.3: Integrate ChatMiddleware into runStreamWithToolLoop

Modify runStreamWithToolLoop to wrap streaming chat client calls with ChatMiddleware execution.

Files:

* `go/chatagent/toolloop.go` - Wrap GetStreamingResponse calls with middleware

Success criteria:

* ChatMiddleware is executed before each GetStreamingResponse call
* IsStreaming flag is set to true in ChatContext
* Stream channel is retrieved from ChatContext after middleware execution
* Middleware can intercept, modify, or replace stream
* Original streaming behavior preserved when no middleware configured

Context references:

* go/chatagent/toolloop.go (Lines 120-135) - Current GetStreamingResponse call location
* go/chatagent/toolloop.go (Lines 100-200) - runStreamWithToolLoop structure

Implementation approach:

Reuse the `invokeWithChatMiddleware` helper from Step 2.2 with `isStreaming: true`:

```go
// In runStreamWithToolLoop, replace:
// streamUpdates, err := a.client.GetStreamingResponse(ctx, currentMessages, options)

// With:
_, streamUpdates, err := a.invokeWithChatMiddleware(ctx, currentMessages, options, true)
```

Dependencies:

* Step 2.2 completion

### Step 2.4: Add builder support for ChatMiddleware

Extend chatagent.Builder to support ChatMiddleware via UseChatMiddleware method.

Files:

* `go/chatagent/builder.go` - Add UseChatMiddleware method

Success criteria:

* UseChatMiddleware accepts variadic ChatMiddleware
* Method returns builder for chaining
* Middleware is passed to agent via WithChatMiddleware option
* Follows existing UseMiddleware/UseFunctionMiddleware patterns

Context references:

* go/chatagent/builder.go (Lines 80-100) - Existing UseFunctionMiddleware pattern

Implementation:

```go
// UseChatMiddleware adds middleware to intercept chat client requests.
// ChatMiddleware operates at the lowest level, intercepting each individual
// GetResponse or GetStreamingResponse call to the underlying chat client.
// Use this for caching, rate limiting, or request/response transformation.
func (b *Builder) UseChatMiddleware(middleware ...agent.ChatMiddleware) *Builder {
    b.options = append(b.options, WithChatMiddleware(middleware...))
    return b
}
```

Dependencies:

* Step 2.1 completion

### Step 2.5: Add integration tests for ChatMiddleware

Create integration tests verifying ChatMiddleware works end-to-end with ChatClientAgent.

Files:

* `go/chatagent/chat_middleware_test.go` - New test file for ChatMiddleware integration

Success criteria:

* Tests verify middleware is invoked for GetResponse calls
* Tests verify middleware is invoked for GetStreamingResponse calls
* Tests verify request modification (messages, options)
* Tests verify response override (short-circuiting)
* Tests verify middleware execution order
* Tests verify context.Metadata flows between middleware

Context references:

* go/chatagent/builder_test.go - Existing integration test patterns
* go/chatagent/toolloop_test.go - Mock client patterns

Implementation outline:

```go
package chatagent

import (
    "context"
    "testing"

    "github.com/microsoft/agent-framework-go/agent"
    "github.com/microsoft/agent-framework-go/chat"
)

// recordingChatMiddleware records invocations for testing.
type recordingChatMiddleware struct {
    invocations []string
}

func (m *recordingChatMiddleware) Process(ctx context.Context, chatCtx *agent.ChatContext, next agent.ChatHandler) error {
    m.invocations = append(m.invocations, "before")
    err := next(ctx, chatCtx)
    m.invocations = append(m.invocations, "after")
    return err
}

func TestChatMiddleware_InvokedOnGetResponse(t *testing.T) {
    // Test middleware is called for non-streaming requests
}

func TestChatMiddleware_InvokedOnGetStreamingResponse(t *testing.T) {
    // Test middleware is called for streaming requests
}

func TestChatMiddleware_CanModifyMessages(t *testing.T) {
    // Test middleware can modify messages before sending
}

func TestChatMiddleware_CanShortCircuit(t *testing.T) {
    // Test middleware can return cached response without calling next
}

func TestChatMiddleware_ExecutionOrder(t *testing.T) {
    // Test multiple middlewares execute in correct order
}

func TestBuilder_UseChatMiddleware(t *testing.T) {
    // Test builder method configures middleware correctly
}
```

Dependencies:

* Step 2.4 completion

### Step 2.6: Validate phase changes

Run validation commands for chatagent package.

Validation commands:

* `go build ./chatagent/...` - Compile check
* `go vet ./chatagent/...` - Static analysis
* `go test ./chatagent/...` - Unit and integration tests

## Implementation Phase 3: Documentation

<!-- parallelizable: true -->

### Step 3.1: Update go/agent/doc.go with ChatMiddleware documentation

Add ChatMiddleware section to the agent package documentation.

Files:

* `go/agent/doc.go` - Add ChatMiddleware documentation section

Success criteria:

* Documents ChatMiddleware purpose and use cases
* Explains difference from AgentMiddleware (lower level)
* Shows example middleware implementation
* Follows existing documentation style

Context references:

* go/agent/doc.go (Lines 30-60) - Existing middleware documentation

Implementation:

Add documentation section similar to AgentMiddleware and FunctionMiddleware:

```go
// # ChatMiddleware
//
// ChatMiddleware intercepts chat client requests at the lowest level,
// allowing interception of each GetResponse or GetStreamingResponse call.
// This is useful for:
//
//   - Request caching and memoization
//   - Rate limiting and throttling
//   - Request/response logging
//   - Message transformation before sending
//   - Response transformation after receiving
//
// ChatMiddleware operates below AgentMiddleware. While AgentMiddleware
// intercepts the entire agent invocation (which may involve multiple
// chat requests in a tool loop), ChatMiddleware intercepts each
// individual chat request.
//
// Example logging middleware:
//
//     type LoggingChatMiddleware struct {
//         Logger *slog.Logger
//     }
//
//     func (m *LoggingChatMiddleware) Process(ctx context.Context, chatCtx *ChatContext, next ChatHandler) error {
//         m.Logger.Info("Chat request", "messages", len(chatCtx.Messages))
//         start := time.Now()
//         err := next(ctx, chatCtx)
//         m.Logger.Info("Chat response", "duration", time.Since(start))
//         return err
//     }
```

Dependencies:

* Phase 1 completion

### Step 3.2: Update go/README.md with ChatMiddleware section

Add ChatMiddleware examples to the README middleware section.

Files:

* `go/README.md` - Add ChatMiddleware section under Middleware heading

Success criteria:

* Documents ChatMiddleware purpose
* Shows practical example with caching or logging
* Explains when to use ChatMiddleware vs AgentMiddleware
* Follows existing README documentation style

Context references:

* go/README.md (Lines 303-380) - Existing Middleware section

Implementation:

Add new subsection under Middleware:

```markdown
### Chat Middleware

ChatMiddleware intercepts individual chat client requests, operating at a lower
level than AgentMiddleware. Use ChatMiddleware for:

- Response caching to avoid redundant API calls
- Rate limiting and throttling
- Request/response logging and metrics
- Message transformation

```go
// Caching middleware example
type CachingMiddleware struct {
    cache map[string]*chat.Response
    mu    sync.RWMutex
}

func (m *CachingMiddleware) Process(ctx context.Context, chatCtx *agent.ChatContext, next agent.ChatHandler) error {
    // Skip caching for streaming
    if chatCtx.IsStreaming {
        return next(ctx, chatCtx)
    }

    key := computeCacheKey(chatCtx.Messages)
    
    m.mu.RLock()
    if cached, ok := m.cache[key]; ok {
        m.mu.RUnlock()
        chatCtx.Response = cached
        return nil // Short-circuit
    }
    m.mu.RUnlock()

    if err := next(ctx, chatCtx); err != nil {
        return err
    }

    m.mu.Lock()
    m.cache[key] = chatCtx.Response
    m.mu.Unlock()
    return nil
}

// Usage
agent := chatagent.NewBuilder(client).
    Name("CachedAgent").
    UseChatMiddleware(&CachingMiddleware{cache: make(map[string]*chat.Response)}).
    Build()
```
```

Dependencies:

* Phase 1 and Phase 2 completion

### Step 3.3: Validate documentation

Verify documentation is correct and complete.

Validation commands:

* `go doc ./agent/...` - Verify doc comments render correctly
* Manual review of README.md examples

## Implementation Phase 4: Validation

<!-- parallelizable: false -->

### Step 4.1: Run full project validation

Execute all validation commands for the complete project.

Validation commands:

* `go build ./...` - Full project build
* `go vet ./...` - Static analysis for all packages
* `go test ./agent/... ./chatagent/...` - Run all relevant tests

### Step 4.2: Fix minor validation issues

Iterate on any lint errors, build warnings, or test failures discovered during validation. Apply fixes directly when corrections are straightforward and isolated.

### Step 4.3: Report blocking issues

When validation failures require changes beyond minor fixes:

* Document the issues and affected files
* Provide the user with next steps
* Recommend additional research and planning rather than inline fixes
* Avoid large-scale refactoring within this phase

## Dependencies

* go/agent/middleware.go - Existing middleware patterns
* go/agent/middleware_context.go - Existing context patterns
* go/agent/chain.go - Existing chain composition
* go/chat/client.go - Chat Client interface
* go/chatagent/toolloop.go - Tool loop integration point

## Success Criteria

* ChatMiddleware interface added to agent package
* ChatContext struct captures all request/response data
* ChainChatMiddleware composes middleware correctly
* ChatMiddleware integrated into chatagent tool loop
* Builder.UseChatMiddleware() method available
* All unit and integration tests pass
* Documentation updated in doc.go and README.md
* All validation commands pass
