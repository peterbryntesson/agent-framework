<!-- markdownlint-disable-file -->
# Implementation Details: Go ContextProvider Pattern for Dynamic Context Injection

## Context Reference

Sources:
* [.copilot-tracking/research/2026-02-03-go-middleware-followup-research.md](.copilot-tracking/research/2026-02-03-go-middleware-followup-research.md)
* [.copilot-tracking/subagent/2026-02-03/python-context-provider-analysis.md](.copilot-tracking/subagent/2026-02-03/python-context-provider-analysis.md)
* [.copilot-tracking/subagent/2026-02-03/context-provider-codebase-research.md](.copilot-tracking/subagent/2026-02-03/context-provider-codebase-research.md)

---

## Implementation Phase 1: Context and ContextProvider Interfaces

<!-- parallelizable: true -->

### Step 1.1: Create Context struct in agent package

Create `go/agent/context_provider.go` with the Context struct that holds dynamic context to inject before agent invocations.

Files:
* `go/agent/context_provider.go` - New file for Context struct and ContextProvider interfaces

```go
// Copyright (c) Microsoft. All rights reserved.

package agent

import (
    "github.com/microsoft/agents/go/chat"
    "github.com/microsoft/agents/go/tool"
)

// Context represents additional context to inject before an agent invocation.
// Instructions are appended to the agent's base instructions.
// Messages are prepended before user messages.
// Tools are merged with the agent's configured tools.
type Context struct {
    // Instructions are additional system instructions to append.
    // Multiple provider instructions are concatenated with newlines.
    Instructions string

    // Messages are additional messages to prepend before user messages.
    // These appear after system instructions and session history.
    Messages []chat.Message

    // Tools are additional tools to make available for this invocation.
    // These are merged with the agent's base tools and run-level tools.
    Tools []tool.Tool
}
```

Success criteria:
* Context struct compiles with correct imports
* Fields match Python Context semantics (instructions, messages, tools)

Context references:
* [python-context-provider-analysis.md](.copilot-tracking/subagent/2026-02-03/python-context-provider-analysis.md) (Lines 50-72) - Python Context class definition

Dependencies:
* go/chat package for Message type
* go/tool package for Tool type

---

### Step 1.2: Create ContextProvider interface with required Invoking method

Add the ContextProvider interface to `go/agent/context_provider.go`. The Invoking method is called before each agent invocation.

```go
// ContextProvider injects dynamic context before agent invocations.
// Implement this interface to add user-specific personalization,
// retrieved documents (RAG), dynamic tool availability, or other
// context that should vary per invocation.
//
// The Invoking method is called before each agent Run or RunStream call.
// Return a Context with additional instructions, messages, or tools
// to inject into the invocation.
//
// Example implementations:
//   - User profile provider that adds personalization instructions
//   - RAG provider that retrieves relevant documents as messages
//   - Feature flag provider that enables/disables tools dynamically
//   - Time provider that adds current datetime to context
type ContextProvider interface {
    // Invoking is called just before the agent invokes the chat client.
    // The messages parameter contains the conversation messages for this invocation.
    // Return a Context with additional instructions, messages, or tools to inject.
    // Return an empty Context (or nil) to inject nothing.
    // Return an error to abort the invocation.
    Invoking(ctx context.Context, messages []Message) (*Context, error)
}

// ContextProviderFunc is a function adapter for ContextProvider.
// Use this for simple providers that only need the Invoking method.
type ContextProviderFunc func(ctx context.Context, messages []Message) (*Context, error)

// Invoking implements ContextProvider.
func (f ContextProviderFunc) Invoking(ctx context.Context, messages []Message) (*Context, error) {
    return f(ctx, messages)
}
```

Success criteria:
* ContextProvider interface defines Invoking method
* ContextProviderFunc adapter enables functional style
* Godoc comments explain purpose and use cases

Context references:
* [python-context-provider-analysis.md](.copilot-tracking/subagent/2026-02-03/python-context-provider-analysis.md) (Lines 18-42) - Python ContextProvider.invoking()
* [context-provider-codebase-research.md](.copilot-tracking/subagent/2026-02-03/context-provider-codebase-research.md) (Lines 63-80) - Go middleware func adapter pattern

Dependencies:
* Step 1.1 completion (Context struct)

---

### Step 1.3: Create optional lifecycle interface ContextProviderWithLifecycle

Add an extended interface for providers that need lifecycle hooks beyond Invoking.

```go
// ContextProviderWithLifecycle extends ContextProvider with lifecycle hooks.
// Implement this interface when your provider needs to:
//   - Track conversation history after invocations (Invoked)
//   - Initialize state when a new session is created (SessionCreated)
//
// These methods are optional. The base ContextProvider interface only
// requires Invoking. Use ContextProviderWithLifecycle when you need
// to update provider state based on agent responses.
type ContextProviderWithLifecycle interface {
    ContextProvider

    // Invoked is called after the agent receives a response from the chat client.
    // Use this to update provider state based on the conversation.
    //
    // Parameters:
    //   - request: The messages sent to the agent for this invocation
    //   - response: The messages returned by the agent (may be nil on error)
    //   - invokeErr: Any error that occurred during invocation (may be nil on success)
    //
    // The error returned by Invoked is logged but does not affect the agent response.
    // The agent will still return the successful response even if Invoked fails.
    Invoked(ctx context.Context, request []Message, response []Message, invokeErr error) error

    // SessionCreated is called when a new session is created or assigned an ID.
    // Use this to initialize session-specific state or register with external services.
    //
    // Parameters:
    //   - sessionID: The unique identifier for the session
    //
    // The error returned by SessionCreated is logged but does not prevent session creation.
    SessionCreated(ctx context.Context, sessionID string) error
}

// BaseContextProvider provides no-op implementations of lifecycle methods.
// Embed this in your provider to only override methods you need.
type BaseContextProvider struct{}

// Invoked is a no-op implementation.
func (BaseContextProvider) Invoked(ctx context.Context, request, response []Message, invokeErr error) error {
    return nil
}

// SessionCreated is a no-op implementation.
func (BaseContextProvider) SessionCreated(ctx context.Context, sessionID string) error {
    return nil
}
```

Success criteria:
* ContextProviderWithLifecycle extends ContextProvider
* Invoked and SessionCreated methods have clear semantics
* BaseContextProvider provides no-op implementations for embedding

Context references:
* [python-context-provider-analysis.md](.copilot-tracking/subagent/2026-02-03/python-context-provider-analysis.md) (Lines 23-35) - Python invoked/thread_created

Dependencies:
* Step 1.2 completion (ContextProvider interface)

---

### Step 1.4: Add unit tests for Context struct

Create `go/agent/context_provider_test.go` with tests for the Context struct and interface implementations.

Files:
* `go/agent/context_provider_test.go` - New test file

```go
// Copyright (c) Microsoft. All rights reserved.

package agent

import (
    "context"
    "testing"

    "github.com/microsoft/agents/go/chat"
    "github.com/microsoft/agents/go/tool"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestContext_ZeroValue(t *testing.T) {
    var ctx Context
    assert.Empty(t, ctx.Instructions)
    assert.Nil(t, ctx.Messages)
    assert.Nil(t, ctx.Tools)
}

func TestContextProviderFunc_Invoking(t *testing.T) {
    provider := ContextProviderFunc(func(ctx context.Context, messages []Message) (*Context, error) {
        return &Context{
            Instructions: "Test instructions",
        }, nil
    })

    result, err := provider.Invoking(context.Background(), nil)
    require.NoError(t, err)
    assert.Equal(t, "Test instructions", result.Instructions)
}

func TestBaseContextProvider_NoOpMethods(t *testing.T) {
    var base BaseContextProvider
    
    err := base.Invoked(context.Background(), nil, nil, nil)
    assert.NoError(t, err)
    
    err = base.SessionCreated(context.Background(), "session-123")
    assert.NoError(t, err)
}
```

Success criteria:
* Tests verify Context zero value behavior
* Tests verify ContextProviderFunc adapter works
* Tests verify BaseContextProvider no-op methods

Dependencies:
* Steps 1.1-1.3 completion

---

## Implementation Phase 2: AggregateContextProvider Implementation

<!-- parallelizable: true -->

### Step 2.1: Implement AggregateContextProvider struct

Add AggregateContextProvider to `go/agent/context_provider.go` for combining multiple providers.

```go
// AggregateContextProvider combines multiple ContextProviders.
// It invokes all providers concurrently and merges their results:
//   - Instructions are concatenated with newlines
//   - Messages are extended in provider order
//   - Tools are extended in provider order
//
// Use AggregateContextProvider when you need multiple sources of context,
// such as combining user personalization, RAG retrieval, and feature flags.
type AggregateContextProvider struct {
    providers []ContextProvider
}

// NewAggregateContextProvider creates an AggregateContextProvider
// from the given providers. Providers are invoked in the order given.
func NewAggregateContextProvider(providers ...ContextProvider) *AggregateContextProvider {
    return &AggregateContextProvider{
        providers: providers,
    }
}

// Add appends additional providers to the aggregate.
func (a *AggregateContextProvider) Add(providers ...ContextProvider) {
    a.providers = append(a.providers, providers...)
}

// Providers returns the list of providers in this aggregate.
func (a *AggregateContextProvider) Providers() []ContextProvider {
    return a.providers
}
```

Success criteria:
* AggregateContextProvider stores multiple providers
* NewAggregateContextProvider constructor works
* Add method appends providers

Context references:
* [python-context-provider-analysis.md](.copilot-tracking/subagent/2026-02-03/python-context-provider-analysis.md) (Lines 235-280) - Python AggregateContextProvider

Dependencies:
* Phase 1 completion

---

### Step 2.2: Implement concurrent provider invocation with goroutines

Implement the Invoking method with concurrent provider calls using goroutines and channels.

```go
import (
    "context"
    "strings"
    "sync"
)

// Invoking calls all providers concurrently and merges their contexts.
func (a *AggregateContextProvider) Invoking(ctx context.Context, messages []Message) (*Context, error) {
    if len(a.providers) == 0 {
        return &Context{}, nil
    }

    // For a single provider, call directly without goroutine overhead
    if len(a.providers) == 1 {
        return a.providers[0].Invoking(ctx, messages)
    }

    type result struct {
        index   int
        context *Context
        err     error
    }

    results := make(chan result, len(a.providers))
    var wg sync.WaitGroup

    for i, provider := range a.providers {
        wg.Add(1)
        go func(idx int, p ContextProvider) {
            defer wg.Done()
            c, err := p.Invoking(ctx, messages)
            results <- result{index: idx, context: c, err: err}
        }(i, provider)
    }

    // Wait for all goroutines then close channel
    go func() {
        wg.Wait()
        close(results)
    }()

    // Collect results
    collected := make([]result, 0, len(a.providers))
    for r := range results {
        collected = append(collected, r)
    }

    // Sort by original index to maintain deterministic order
    sort.Slice(collected, func(i, j int) bool {
        return collected[i].index < collected[j].index
    })

    // Check for errors (first error wins)
    for _, r := range collected {
        if r.err != nil {
            return nil, r.err
        }
    }

    // Merge contexts
    return a.mergeContexts(collected), nil
}
```

Success criteria:
* Providers are called concurrently using goroutines
* Results are collected via channel
* Original order is preserved for deterministic output
* First error aborts and returns

Dependencies:
* Step 2.1 completion

---

### Step 2.3: Implement context merging logic (concatenate instructions, extend messages/tools)

Add the mergeContexts helper method.

```go
// mergeContexts combines multiple contexts into one.
// Instructions are joined with newlines.
// Messages and Tools are concatenated in order.
func (a *AggregateContextProvider) mergeContexts(results []result) *Context {
    var instructions strings.Builder
    var messages []chat.Message
    var tools []tool.Tool

    for _, r := range results {
        if r.context == nil {
            continue
        }

        // Concatenate instructions with newlines
        if r.context.Instructions != "" {
            if instructions.Len() > 0 {
                instructions.WriteString("\n")
            }
            instructions.WriteString(r.context.Instructions)
        }

        // Extend messages
        if len(r.context.Messages) > 0 {
            messages = append(messages, r.context.Messages...)
        }

        // Extend tools
        if len(r.context.Tools) > 0 {
            tools = append(tools, r.context.Tools...)
        }
    }

    return &Context{
        Instructions: instructions.String(),
        Messages:     messages,
        Tools:        tools,
    }
}
```

Success criteria:
* Instructions are concatenated with newline separators
* Messages are extended preserving order
* Tools are extended preserving order
* Nil contexts are skipped

Context references:
* [python-context-provider-analysis.md](.copilot-tracking/subagent/2026-02-03/python-context-provider-analysis.md) (Lines 285-300) - Python aggregation rules

Dependencies:
* Step 2.2 completion

---

### Step 2.4: Add unit tests for AggregateContextProvider

Add tests to `go/agent/context_provider_test.go`.

```go
func TestAggregateContextProvider_Empty(t *testing.T) {
    agg := NewAggregateContextProvider()
    
    result, err := agg.Invoking(context.Background(), nil)
    require.NoError(t, err)
    assert.Empty(t, result.Instructions)
    assert.Empty(t, result.Messages)
    assert.Empty(t, result.Tools)
}

func TestAggregateContextProvider_SingleProvider(t *testing.T) {
    provider := ContextProviderFunc(func(ctx context.Context, messages []Message) (*Context, error) {
        return &Context{Instructions: "Hello"}, nil
    })
    agg := NewAggregateContextProvider(provider)
    
    result, err := agg.Invoking(context.Background(), nil)
    require.NoError(t, err)
    assert.Equal(t, "Hello", result.Instructions)
}

func TestAggregateContextProvider_MergesInstructions(t *testing.T) {
    p1 := ContextProviderFunc(func(ctx context.Context, messages []Message) (*Context, error) {
        return &Context{Instructions: "First"}, nil
    })
    p2 := ContextProviderFunc(func(ctx context.Context, messages []Message) (*Context, error) {
        return &Context{Instructions: "Second"}, nil
    })
    agg := NewAggregateContextProvider(p1, p2)
    
    result, err := agg.Invoking(context.Background(), nil)
    require.NoError(t, err)
    assert.Equal(t, "First\nSecond", result.Instructions)
}

func TestAggregateContextProvider_MergesMessages(t *testing.T) {
    msg1 := chat.NewUserMessage("User 1")
    msg2 := chat.NewUserMessage("User 2")
    
    p1 := ContextProviderFunc(func(ctx context.Context, messages []Message) (*Context, error) {
        return &Context{Messages: []chat.Message{msg1}}, nil
    })
    p2 := ContextProviderFunc(func(ctx context.Context, messages []Message) (*Context, error) {
        return &Context{Messages: []chat.Message{msg2}}, nil
    })
    agg := NewAggregateContextProvider(p1, p2)
    
    result, err := agg.Invoking(context.Background(), nil)
    require.NoError(t, err)
    require.Len(t, result.Messages, 2)
}

func TestAggregateContextProvider_ErrorReturnsFirst(t *testing.T) {
    expectedErr := errors.New("provider error")
    p1 := ContextProviderFunc(func(ctx context.Context, messages []Message) (*Context, error) {
        return nil, expectedErr
    })
    p2 := ContextProviderFunc(func(ctx context.Context, messages []Message) (*Context, error) {
        return &Context{Instructions: "OK"}, nil
    })
    agg := NewAggregateContextProvider(p1, p2)
    
    _, err := agg.Invoking(context.Background(), nil)
    assert.ErrorIs(t, err, expectedErr)
}

func TestAggregateContextProvider_PreservesOrder(t *testing.T) {
    // Test that despite concurrent execution, results are ordered by provider index
    results := make([]string, 3)
    for i := 0; i < 10; i++ { // Run multiple times to catch race conditions
        p1 := ContextProviderFunc(func(ctx context.Context, messages []Message) (*Context, error) {
            return &Context{Instructions: "A"}, nil
        })
        p2 := ContextProviderFunc(func(ctx context.Context, messages []Message) (*Context, error) {
            return &Context{Instructions: "B"}, nil
        })
        p3 := ContextProviderFunc(func(ctx context.Context, messages []Message) (*Context, error) {
            return &Context{Instructions: "C"}, nil
        })
        agg := NewAggregateContextProvider(p1, p2, p3)
        
        result, err := agg.Invoking(context.Background(), nil)
        require.NoError(t, err)
        assert.Equal(t, "A\nB\nC", result.Instructions)
    }
}
```

Success criteria:
* Empty provider list returns empty context
* Single provider bypasses concurrency overhead
* Multiple providers merge instructions with newlines
* Multiple providers extend messages in order
* First error is returned
* Order is preserved despite concurrent execution

Dependencies:
* Steps 2.1-2.3 completion

---

## Implementation Phase 3: ChatClientAgent Integration

<!-- parallelizable: false -->

### Step 3.1: Add contextProviders field to chatagent.config struct

Modify `go/chatagent/options.go` to add the contextProviders field to the config struct.

Files:
* `go/chatagent/options.go` - Add contextProviders to config

Current code at approximately line 10-25:
```go
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
```

Modified code:
```go
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
    contextProviders   []agent.ContextProvider
}
```

Success criteria:
* contextProviders field added to config struct
* Field type is []agent.ContextProvider

Dependencies:
* Phase 2 completion

---

### Step 3.2: Add WithContextProvider option function

Add the functional option to `go/chatagent/options.go`.

```go
// WithContextProvider adds context providers to the agent.
// Context providers inject dynamic instructions, messages, and tools
// before each agent invocation.
//
// Multiple providers can be added by calling this option multiple times
// or by passing multiple providers in a single call.
// Providers are invoked in the order they are added.
//
// Example:
//
//     agent := chatagent.NewBuilder(client).
//         WithContextProvider(userProfileProvider).
//         WithContextProvider(ragProvider, featureFlagProvider).
//         BuildAgent()
func WithContextProvider(providers ...agent.ContextProvider) Option {
    return func(c *config) {
        c.contextProviders = append(c.contextProviders, providers...)
    }
}
```

Success criteria:
* WithContextProvider option accepts variadic providers
* Providers are appended to allow multiple calls
* Godoc documents usage pattern

Dependencies:
* Step 3.1 completion

---

### Step 3.3: Store contextProviders in Agent struct

Modify `go/chatagent/agent.go` to store context providers.

Files:
* `go/chatagent/agent.go` - Add contextProvider field and initialization

Current Agent struct (approximately lines 20-35):
```go
type Agent struct {
    id                 string
    name               string
    description        string
    instructions       string
    client             chat.Client
    tools              []tool.Tool
    maxTurns           int
    invocationConfig   tool.InvocationConfig
    functionMiddleware agent.FunctionMiddleware
    chatMiddleware     agent.ChatMiddleware
}
```

Modified Agent struct:
```go
type Agent struct {
    id                 string
    name               string
    description        string
    instructions       string
    client             chat.Client
    tools              []tool.Tool
    maxTurns           int
    invocationConfig   tool.InvocationConfig
    functionMiddleware agent.FunctionMiddleware
    chatMiddleware     agent.ChatMiddleware
    contextProvider    agent.ContextProvider // nil or AggregateContextProvider
}
```

In NewAgent or builder construction (where config is applied):
```go
// After other config application
if len(cfg.contextProviders) > 0 {
    if len(cfg.contextProviders) == 1 {
        a.contextProvider = cfg.contextProviders[0]
    } else {
        a.contextProvider = agent.NewAggregateContextProvider(cfg.contextProviders...)
    }
}
```

Success criteria:
* Agent struct has contextProvider field
* Single provider stored directly (no wrapper overhead)
* Multiple providers wrapped in AggregateContextProvider

Dependencies:
* Step 3.2 completion

---

## Implementation Phase 4: Context Injection in Agent.Run

<!-- parallelizable: false -->

### Step 4.1: Create getProviderContext helper method on Agent

Add helper method to `go/chatagent/agent.go`.

```go
// getProviderContext invokes the context provider and returns the context to inject.
// Returns nil if no context provider is configured.
func (a *Agent) getProviderContext(ctx context.Context, messages []agent.Message) (*agent.Context, error) {
    if a.contextProvider == nil {
        return nil, nil
    }
    return a.contextProvider.Invoking(ctx, messages)
}
```

Success criteria:
* Returns nil when no provider configured
* Calls provider.Invoking with messages
* Returns error on provider failure

Dependencies:
* Phase 3 completion

---

### Step 4.2: Modify prepareMessages to inject provider context

Modify the prepareMessages method in `go/chatagent/agent.go` to inject provider context.

Current prepareMessages (approximately lines 161-178):
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

Modified prepareMessages signature and implementation:
```go
func (a *Agent) prepareMessages(messages []agent.Message, cfg *agent.RunConfig, providerCtx *agent.Context) []chat.Message {
    capacity := len(messages) + 1
    if cfg.Session != nil {
        capacity += len(cfg.Session.Messages())
    }
    if providerCtx != nil {
        capacity += len(providerCtx.Messages) + 1 // +1 for potential instructions
    }
    chatMessages := make([]chat.Message, 0, capacity)
    
    // 1. Add base agent instructions
    instructions := a.instructions
    
    // 2. Append provider instructions if present
    if providerCtx != nil && providerCtx.Instructions != "" {
        if instructions != "" {
            instructions = instructions + "\n" + providerCtx.Instructions
        } else {
            instructions = providerCtx.Instructions
        }
    }
    
    // 3. Add combined system instructions
    if instructions != "" {
        chatMessages = append(chatMessages, chat.NewSystemMessage(instructions))
    }
    
    // 4. Add provider messages (before session history)
    if providerCtx != nil && len(providerCtx.Messages) > 0 {
        chatMessages = append(chatMessages, providerCtx.Messages...)
    }
    
    // 5. Add session history if provided
    if cfg.Session != nil {
        chatMessages = append(chatMessages, cfg.Session.Messages()...)
    }
    
    // 6. Add new messages from this invocation
    chatMessages = append(chatMessages, messages...)
    return chatMessages
}
```

Message injection order:
1. Base agent instructions + provider instructions (as system message)
2. Provider messages (context, examples, retrieved documents)
3. Session history (previous conversation turns)
4. New messages (current user input)

Success criteria:
* Provider instructions are appended to agent instructions
* Provider messages are injected before session history
* Original message ordering is preserved
* Works correctly when provider context is nil

Context references:
* [python-context-provider-analysis.md](.copilot-tracking/subagent/2026-02-03/python-context-provider-analysis.md) (Lines 125-160) - Python message injection order

Dependencies:
* Step 4.1 completion

---

### Step 4.3: Modify prepareChatOptions to inject provider tools

Add provider tools to the chat options. This may require modifying how tools are collected.

Files:
* `go/chatagent/agent.go` - Modify tool collection

Current getAllTools method (if exists) or tool preparation:
```go
func (a *Agent) getAllTools(cfg *agent.RunConfig) []tool.Tool {
    tools := make([]tool.Tool, 0, len(a.tools)+len(cfg.Tools))
    tools = append(tools, a.tools...)
    if len(cfg.Tools) > 0 {
        // ... append run-level tools
    }
    return tools
}
```

Modified to accept provider context:
```go
func (a *Agent) getAllTools(cfg *agent.RunConfig, providerCtx *agent.Context) []tool.Tool {
    capacity := len(a.tools)
    if providerCtx != nil {
        capacity += len(providerCtx.Tools)
    }
    if len(cfg.Tools) > 0 {
        capacity += len(cfg.Tools)
    }
    
    tools := make([]tool.Tool, 0, capacity)
    
    // 1. Base agent tools
    tools = append(tools, a.tools...)
    
    // 2. Provider tools
    if providerCtx != nil && len(providerCtx.Tools) > 0 {
        tools = append(tools, providerCtx.Tools...)
    }
    
    // 3. Run-level tools (highest priority, can override)
    if len(cfg.Tools) > 0 {
        // ... append run-level tools
    }
    
    return tools
}
```

Success criteria:
* Provider tools are merged between base and run-level tools
* Tool collection capacity is pre-calculated for efficiency
* Works correctly when provider context is nil

Dependencies:
* Step 4.2 completion

---

### Step 4.4: Modify Agent.Run to call provider and pass context

Update the Run method to invoke the context provider.

Current Run method (approximately lines 95-105):
```go
func (a *Agent) Run(ctx context.Context, messages []agent.Message, opts ...agent.RunOption) (*agent.Response, error) {
    cfg := agent.ApplyRunOptions(opts...)
    chatMessages := a.prepareMessages(messages, cfg)
    chatOptions := a.prepareChatOptions(cfg)
    return a.runWithToolLoop(ctx, chatMessages, chatOptions, cfg)
}
```

Modified Run method:
```go
func (a *Agent) Run(ctx context.Context, messages []agent.Message, opts ...agent.RunOption) (*agent.Response, error) {
    cfg := agent.ApplyRunOptions(opts...)
    
    // Get dynamic context from provider
    providerCtx, err := a.getProviderContext(ctx, messages)
    if err != nil {
        return nil, fmt.Errorf("context provider error: %w", err)
    }
    
    // Prepare messages with provider context injected
    chatMessages := a.prepareMessages(messages, cfg, providerCtx)
    chatOptions := a.prepareChatOptions(cfg)
    
    // Run with tool loop
    response, err := a.runWithToolLoop(ctx, chatMessages, chatOptions, cfg, providerCtx)
    
    // Notify provider of completion (best effort, don't fail on error)
    if a.contextProvider != nil {
        if lcp, ok := a.contextProvider.(agent.ContextProviderWithLifecycle); ok {
            var responseMessages []agent.Message
            if response != nil {
                responseMessages = response.Messages
            }
            if notifyErr := lcp.Invoked(ctx, messages, responseMessages, err); notifyErr != nil {
                // Log but don't fail - the response is still valid
                // Consider: a.logger.Warn("context provider Invoked failed", "error", notifyErr)
            }
        }
    }
    
    return response, err
}
```

Success criteria:
* Provider is invoked before message preparation
* Provider error aborts the run
* Invoked lifecycle hook is called after response
* Invoked error is logged but doesn't fail the run

Dependencies:
* Steps 4.1-4.3 completion

---

### Step 4.5: Add integration tests for context injection

Create `go/chatagent/context_provider_test.go` with integration tests.

```go
// Copyright (c) Microsoft. All rights reserved.

package chatagent

import (
    "context"
    "testing"

    "github.com/microsoft/agents/go/agent"
    "github.com/microsoft/agents/go/chat"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

type mockChatClient struct {
    capturedMessages []chat.Message
    response         *chat.Response
}

func (m *mockChatClient) GetResponse(ctx context.Context, messages []chat.Message, opts *chat.Options) (*chat.Response, error) {
    m.capturedMessages = messages
    return m.response, nil
}

func TestAgent_WithContextProvider_InjectsInstructions(t *testing.T) {
    client := &mockChatClient{
        response: &chat.Response{
            Message: chat.NewAssistantMessage("Hello"),
        },
    }
    
    provider := agent.ContextProviderFunc(func(ctx context.Context, messages []agent.Message) (*agent.Context, error) {
        return &agent.Context{
            Instructions: "You are a helpful assistant.",
        }, nil
    })
    
    a := NewBuilder(client).
        WithInstructions("Base instructions.").
        WithContextProvider(provider).
        BuildAgent()
    
    _, err := a.Run(context.Background(), []agent.Message{
        agent.NewUserMessage("Hi"),
    })
    require.NoError(t, err)
    
    // Verify instructions were merged
    require.Len(t, client.capturedMessages, 2) // system + user
    assert.Equal(t, chat.RoleSystem, client.capturedMessages[0].Role)
    assert.Contains(t, client.capturedMessages[0].Content, "Base instructions.")
    assert.Contains(t, client.capturedMessages[0].Content, "You are a helpful assistant.")
}

func TestAgent_WithContextProvider_InjectsMessages(t *testing.T) {
    client := &mockChatClient{
        response: &chat.Response{
            Message: chat.NewAssistantMessage("Hello"),
        },
    }
    
    provider := agent.ContextProviderFunc(func(ctx context.Context, messages []agent.Message) (*agent.Context, error) {
        return &agent.Context{
            Messages: []chat.Message{
                chat.NewUserMessage("Context message"),
            },
        }, nil
    })
    
    a := NewBuilder(client).
        WithContextProvider(provider).
        BuildAgent()
    
    _, err := a.Run(context.Background(), []agent.Message{
        agent.NewUserMessage("User message"),
    })
    require.NoError(t, err)
    
    // Verify context message appears before user message
    require.Len(t, client.capturedMessages, 2)
    assert.Equal(t, "Context message", client.capturedMessages[0].Content)
    assert.Equal(t, "User message", client.capturedMessages[1].Content)
}

func TestAgent_WithContextProvider_Error_AbortsRun(t *testing.T) {
    client := &mockChatClient{}
    expectedErr := errors.New("provider failed")
    
    provider := agent.ContextProviderFunc(func(ctx context.Context, messages []agent.Message) (*agent.Context, error) {
        return nil, expectedErr
    })
    
    a := NewBuilder(client).
        WithContextProvider(provider).
        BuildAgent()
    
    _, err := a.Run(context.Background(), []agent.Message{
        agent.NewUserMessage("Hi"),
    })
    
    assert.ErrorIs(t, err, expectedErr)
}
```

Success criteria:
* Tests verify instructions are merged correctly
* Tests verify messages are injected in correct order
* Tests verify provider errors abort the run
* Tests use mock chat client to capture actual messages

Dependencies:
* Steps 4.1-4.4 completion

---

## Implementation Phase 5: Lifecycle Hooks Implementation

<!-- parallelizable: false -->

### Step 5.1: Call Invoked hook after agent response

This is already integrated in Step 4.4. Verify the implementation handles:
* Success case: response is not nil, err is nil
* Error case: response may be nil, err is not nil
* Provider returns error: logged but not propagated

Success criteria:
* Invoked is called after every Run completion
* Both request and response messages are passed
* Error is passed to allow provider to track failures

Dependencies:
* Phase 4 completion

---

### Step 5.2: Call SessionCreated hook when session is created

Modify session handling to call the lifecycle hook. This requires understanding where sessions are created.

Files:
* `go/chatagent/session.go` or `go/agent/session.go` - Add hook call

If sessions are created in the agent or passed in via options, add hook call:

```go
// When a new session is created or assigned
func (a *Agent) notifySessionCreated(ctx context.Context, session agent.Session) {
    if a.contextProvider == nil {
        return
    }
    if lcp, ok := a.contextProvider.(agent.ContextProviderWithLifecycle); ok {
        if err := lcp.SessionCreated(ctx, session.ID()); err != nil {
            // Log but don't fail
            // a.logger.Warn("context provider SessionCreated failed", "error", err)
        }
    }
}
```

Call this when:
* A session is first attached to a run
* A session is created by the agent

Success criteria:
* SessionCreated is called when session ID is available
* Error is logged but doesn't prevent session use

Dependencies:
* Step 5.1 completion

---

### Step 5.3: Add lifecycle hook tests

Add tests for lifecycle hooks to `go/chatagent/context_provider_test.go`.

```go
type lifecycleTracker struct {
    agent.BaseContextProvider
    invokedCalled       bool
    invokedRequest      []agent.Message
    invokedResponse     []agent.Message
    invokedErr          error
    sessionCreatedCalled bool
    sessionID           string
}

func (t *lifecycleTracker) Invoking(ctx context.Context, messages []agent.Message) (*agent.Context, error) {
    return &agent.Context{}, nil
}

func (t *lifecycleTracker) Invoked(ctx context.Context, request, response []agent.Message, err error) error {
    t.invokedCalled = true
    t.invokedRequest = request
    t.invokedResponse = response
    t.invokedErr = err
    return nil
}

func (t *lifecycleTracker) SessionCreated(ctx context.Context, sessionID string) error {
    t.sessionCreatedCalled = true
    t.sessionID = sessionID
    return nil
}

func TestAgent_LifecycleHooks_Invoked_CalledOnSuccess(t *testing.T) {
    client := &mockChatClient{
        response: &chat.Response{
            Message: chat.NewAssistantMessage("Response"),
        },
    }
    tracker := &lifecycleTracker{}
    
    a := NewBuilder(client).
        WithContextProvider(tracker).
        BuildAgent()
    
    _, err := a.Run(context.Background(), []agent.Message{
        agent.NewUserMessage("Request"),
    })
    require.NoError(t, err)
    
    assert.True(t, tracker.invokedCalled)
    assert.Len(t, tracker.invokedRequest, 1)
    assert.NotEmpty(t, tracker.invokedResponse)
    assert.Nil(t, tracker.invokedErr)
}

func TestAgent_LifecycleHooks_Invoked_CalledOnError(t *testing.T) {
    expectedErr := errors.New("chat client error")
    client := &mockChatClientWithError{err: expectedErr}
    tracker := &lifecycleTracker{}
    
    a := NewBuilder(client).
        WithContextProvider(tracker).
        BuildAgent()
    
    _, err := a.Run(context.Background(), []agent.Message{
        agent.NewUserMessage("Request"),
    })
    assert.Error(t, err)
    
    assert.True(t, tracker.invokedCalled)
    assert.ErrorIs(t, tracker.invokedErr, expectedErr)
}
```

Success criteria:
* Invoked is called on successful runs
* Invoked is called on failed runs with error
* Request and response messages are passed correctly

Dependencies:
* Steps 5.1-5.2 completion

---

## Implementation Phase 6: Documentation and Samples

<!-- parallelizable: true -->

### Step 6.1: Update go/README.md with ContextProvider section

Add documentation section to `go/README.md`.

```markdown
### Context Providers

Context providers enable dynamic context injection before each agent invocation.
Use them for:
- User personalization based on profile data
- RAG (Retrieval Augmented Generation) to inject relevant documents
- Dynamic tool availability based on feature flags
- Time-aware context with current date/time

#### Basic Usage

```go
// Create a simple context provider
timeProvider := agent.ContextProviderFunc(func(ctx context.Context, messages []agent.Message) (*agent.Context, error) {
    return &agent.Context{
        Instructions: fmt.Sprintf("Current time: %s", time.Now().Format(time.RFC3339)),
    }, nil
})

// Add to agent
agent := chatagent.NewBuilder(client).
    WithInstructions("You are a helpful assistant.").
    WithContextProvider(timeProvider).
    BuildAgent()
```

#### Multiple Providers

Multiple providers are invoked concurrently and their contexts are merged:

```go
agent := chatagent.NewBuilder(client).
    WithContextProvider(userProfileProvider).
    WithContextProvider(ragProvider).
    WithContextProvider(featureFlagProvider).
    BuildAgent()
```

Merging rules:
- **Instructions**: Concatenated with newlines
- **Messages**: Extended in provider order
- **Tools**: Extended in provider order

#### Lifecycle Hooks

Implement `ContextProviderWithLifecycle` for advanced scenarios:

```go
type UserInfoProvider struct {
    agent.BaseContextProvider // Provides no-op Invoked/SessionCreated
    userInfo map[string]string
}

func (p *UserInfoProvider) Invoking(ctx context.Context, messages []agent.Message) (*agent.Context, error) {
    // Inject user context
    return &agent.Context{
        Instructions: fmt.Sprintf("User name: %s", p.userInfo["name"]),
    }, nil
}

func (p *UserInfoProvider) Invoked(ctx context.Context, request, response []agent.Message, err error) error {
    // Extract and store user info from conversation
    // ...
    return nil
}
```

#### Context Injection Order

Messages are composed in this order:
1. System message (base instructions + provider instructions)
2. Provider messages (retrieved documents, examples)
3. Session history (previous conversation)
4. New messages (current user input)
```

Success criteria:
* README documents basic usage pattern
* README explains multiple provider merging
* README shows lifecycle hook implementation
* README explains message injection order

Dependencies:
* Phases 1-5 completion

---

### Step 6.2: Add godoc comments to all exported types

Ensure all exported types, methods, and functions have godoc comments following Go conventions.

Files to verify:
* `go/agent/context_provider.go` - All exported types and methods

Success criteria:
* All exported identifiers have godoc comments
* Comments follow Go conventions (start with identifier name)
* Examples are included where helpful

Dependencies:
* Phases 1-5 completion

---

### Step 6.3: Create sample context provider implementations

Create sample providers demonstrating common patterns.

Files:
* `go/samples/context_providers/` - New sample directory (if samples exist)
* Or document in README with inline examples

Sample providers:
1. **TimeContextProvider** - Adds current date/time
2. **PersonaContextProvider** - Adds personality/role instructions
3. **UserProfileProvider** - Stateful provider with lifecycle hooks

```go
// TimeContextProvider adds current date/time to agent context.
type TimeContextProvider struct{}

func (TimeContextProvider) Invoking(ctx context.Context, messages []agent.Message) (*agent.Context, error) {
    now := time.Now().Format("Monday, January 2, 2006 at 3:04 PM")
    return &agent.Context{
        Instructions: fmt.Sprintf("Current date and time: %s", now),
    }, nil
}

// PersonaContextProvider adds a persona to the agent.
type PersonaContextProvider struct {
    Persona string
}

func (p PersonaContextProvider) Invoking(ctx context.Context, messages []agent.Message) (*agent.Context, error) {
    return &agent.Context{
        Instructions: fmt.Sprintf("Your persona: %s", p.Persona),
    }, nil
}
```

Success criteria:
* Sample providers demonstrate key patterns
* Samples are simple and focused
* Code is documented with comments

Dependencies:
* Phases 1-5 completion

---

## Implementation Phase 7: Validation

<!-- parallelizable: false -->

### Step 7.1: Run full project validation

Execute all validation commands:

```bash
cd go

# Build all packages
go build ./...

# Run all tests
go test ./...

# Run vet for static analysis
go vet ./...

# Run tests with race detector
go test -race ./agent/... ./chatagent/...
```

Success criteria:
* All packages build without errors
* All tests pass
* No vet warnings for new code
* No race conditions detected

### Step 7.2: Fix minor validation issues

Iterate on any errors found:
* Fix lint errors and build warnings
* Apply fixes directly when straightforward
* Update tests if behavior changes

### Step 7.3: Report blocking issues

If issues exceed minor fixes:
* Document the issues and affected files
* Provide next steps
* Avoid large-scale refactoring within this phase

---

## Dependencies

* Go 1.21+ (for slices package and generics)
* Existing go/agent package
* Existing go/chatagent package
* Existing go/chat package
* Existing go/tool package
* github.com/stretchr/testify for testing

## Success Criteria

* ContextProvider interface enables dynamic instruction/message/tool injection
* AggregateContextProvider correctly merges multiple provider outputs concurrently
* ChatClientAgent invokes providers before each Run/RunStream call
* Lifecycle hooks (Invoked, SessionCreated) are called at appropriate times
* All existing tests continue to pass
* New unit tests achieve >80% coverage for context_provider.go
* README.md documents ContextProvider usage with examples
* Code follows existing patterns in the codebase (functional options, middleware)
