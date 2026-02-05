<!-- markdownlint-disable-file -->
# Implementation Details: Go Middleware Patterns for Agent Framework

## Context Reference

Sources:
* Research: [.copilot-tracking/research/2026-02-03-go-middleware-patterns-research.md](.copilot-tracking/research/2026-02-03-go-middleware-patterns-research.md)
* .NET Reference: [dotnet/src/Microsoft.Agents.AI.Abstractions/DelegatingAIAgent.cs](dotnet/src/Microsoft.Agents.AI.Abstractions/DelegatingAIAgent.cs)
* Python Reference: [python/packages/core/agent_framework/_middleware.py](python/packages/core/agent_framework/_middleware.py)

## Implementation Phase 1: Core Middleware Interfaces

<!-- parallelizable: true -->

### Step 1.1: Create middleware context types in agent package

Create context structs that carry invocation data through middleware chains. These types align with Python's `AgentRunContext` and `FunctionInvocationContext`.

Files:
* `go/agent/middleware_context.go` - New file containing context types

Success criteria:
* AgentContext struct holds agent reference, messages, session, options, and response fields
* FunctionContext struct holds function reference, arguments, metadata, and result fields
* Both structs support metadata extension via `map[string]any`
* Types are exported for use by middleware implementations

Implementation:

```go
// Copyright (c) Microsoft. All rights reserved.

package agent

import (
    "encoding/json"
)

// AgentContext holds context for agent middleware invocations.
// It provides access to the agent being invoked, the input messages,
// and fields for capturing the response.
type AgentContext struct {
    // Agent is the agent being invoked.
    Agent Agent

    // Messages contains the input messages for this invocation.
    Messages []Message

    // Session is the optional session for this invocation.
    Session Session

    // Options contains run configuration for this invocation.
    Options *RunConfig

    // Metadata allows middleware to attach arbitrary data.
    Metadata map[string]any

    // Response holds the result for non-streaming invocations.
    // Set by the terminal handler or by middleware to override the response.
    Response *Response

    // Stream holds the result channel for streaming invocations.
    // Set by the terminal handler when IsStreaming is true.
    Stream <-chan ResponseUpdate

    // IsStreaming indicates whether this is a streaming invocation.
    IsStreaming bool
}

// FunctionContext holds context for function middleware invocations.
// It provides access to the function being invoked, its arguments,
// and fields for capturing the result.
type FunctionContext struct {
    // FunctionName is the name of the function being invoked.
    FunctionName string

    // Arguments contains the JSON arguments for the function.
    Arguments json.RawMessage

    // Metadata allows middleware to attach arbitrary data.
    Metadata map[string]any

    // Result holds the function result after invocation.
    // Set by the terminal handler or by middleware to override.
    Result any

    // Error holds any error from the invocation.
    Error error
}
```

Context references:
* Research (Lines 180-215) - AgentContext and FunctionContext design

Dependencies:
* None (new file)

### Step 1.2: Create AgentMiddleware interface

Define the AgentMiddleware interface that intercepts agent Run/RunStream calls. Follow Go idioms with interface + function adapter pattern.

Files:
* `go/agent/middleware.go` - New file containing middleware interfaces

Success criteria:
* AgentMiddleware interface defines Process method with context, agent context, and next handler
* AgentHandler type alias defines the next handler signature
* AgentMiddlewareFunc provides function adapter for the interface
* Interface supports both streaming and non-streaming via IsStreaming flag

Implementation:

```go
// Copyright (c) Microsoft. All rights reserved.

package agent

import (
    "context"
)

// AgentHandler is the next handler in the agent middleware chain.
type AgentHandler func(ctx context.Context, agentCtx *AgentContext) error

// AgentMiddleware intercepts agent invocations.
// Implement this interface to add cross-cutting behavior like
// logging, security validation, or request modification.
type AgentMiddleware interface {
    // Process handles an agent invocation.
    // Call next to continue the chain, or return early to short-circuit.
    // The agentCtx contains input data and receives the response.
    Process(ctx context.Context, agentCtx *AgentContext, next AgentHandler) error
}

// AgentMiddlewareFunc is a function adapter for AgentMiddleware.
// Use this to create middleware from anonymous functions.
type AgentMiddlewareFunc func(ctx context.Context, agentCtx *AgentContext, next AgentHandler) error

// Process implements AgentMiddleware.
func (f AgentMiddlewareFunc) Process(ctx context.Context, agentCtx *AgentContext, next AgentHandler) error {
    return f(ctx, agentCtx, next)
}
```

Context references:
* Research (Lines 145-178) - AgentMiddleware interface design
* gRPC interceptor pattern analysis

Dependencies:
* Step 1.1 completion (AgentContext type)

### Step 1.3: Create FunctionMiddleware interface

Define the FunctionMiddleware interface that intercepts tool/function invocations during the tool loop.

Files:
* `go/agent/middleware.go` - Append to existing file from Step 1.2

Success criteria:
* FunctionMiddleware interface defines Process method
* FunctionHandler type alias defines next handler signature
* FunctionMiddlewareFunc provides function adapter
* Interface aligns with Python's FunctionMiddleware class

Implementation (append to middleware.go):

```go
// FunctionHandler is the next handler in the function middleware chain.
type FunctionHandler func(ctx context.Context, funcCtx *FunctionContext) error

// FunctionMiddleware intercepts tool/function invocations.
// Implement this interface to add validation, caching, or
// transformation of function calls and results.
type FunctionMiddleware interface {
    // Process handles a function invocation.
    // Call next to continue the chain, or return early to short-circuit.
    // Set funcCtx.Result or funcCtx.Error to provide a response.
    Process(ctx context.Context, funcCtx *FunctionContext, next FunctionHandler) error
}

// FunctionMiddlewareFunc is a function adapter for FunctionMiddleware.
type FunctionMiddlewareFunc func(ctx context.Context, funcCtx *FunctionContext, next FunctionHandler) error

// Process implements FunctionMiddleware.
func (f FunctionMiddlewareFunc) Process(ctx context.Context, funcCtx *FunctionContext, next FunctionHandler) error {
    return f(ctx, funcCtx, next)
}
```

Context references:
* Research (Lines 215-245) - FunctionMiddleware interface design

Dependencies:
* Step 1.1 completion (FunctionContext type)

### Step 1.4: Validate phase changes

Run build and vet checks for the agent package.

Validation commands:
* `go build ./agent/...` - Compile new middleware types
* `go vet ./agent/...` - Check for common issues

## Implementation Phase 2: DelegatingAgent Implementation

<!-- parallelizable: true -->

### Step 2.1: Implement DelegatingAgent base type

Create a base agent decorator that wraps an inner agent and forwards all method calls. This enables the decorator pattern for middleware.

Files:
* `go/agent/delegating.go` - New file containing DelegatingAgent

Success criteria:
* DelegatingAgent embeds or holds an inner Agent
* All Agent interface methods forward to inner agent
* Inner() method exposes the wrapped agent for debugging
* Type implements agent.Agent interface (compile-time check)

Implementation:

```go
// Copyright (c) Microsoft. All rights reserved.

package agent

import (
    "context"
    "encoding/json"
    "reflect"
)

// DelegatingAgent wraps an inner agent to enable the decorator pattern.
// Embed this type and override specific methods to customize behavior.
type DelegatingAgent struct {
    inner Agent
}

// Compile-time check that DelegatingAgent implements Agent.
var _ Agent = (*DelegatingAgent)(nil)

// NewDelegatingAgent creates a new delegating agent wrapping the inner agent.
func NewDelegatingAgent(inner Agent) *DelegatingAgent {
    return &DelegatingAgent{inner: inner}
}

// Inner returns the wrapped agent.
func (d *DelegatingAgent) Inner() Agent {
    return d.inner
}

// ID returns the unique identifier from the inner agent.
func (d *DelegatingAgent) ID() string {
    return d.inner.ID()
}

// Name returns the name from the inner agent.
func (d *DelegatingAgent) Name() string {
    return d.inner.Name()
}

// Description returns the description from the inner agent.
func (d *DelegatingAgent) Description() string {
    return d.inner.Description()
}

// Metadata returns the metadata from the inner agent.
func (d *DelegatingAgent) Metadata() AIAgentMetadata {
    return d.inner.Metadata()
}

// Run forwards to the inner agent's Run method.
func (d *DelegatingAgent) Run(ctx context.Context, messages []Message, opts ...RunOption) (*Response, error) {
    return d.inner.Run(ctx, messages, opts...)
}

// RunStream forwards to the inner agent's RunStream method.
func (d *DelegatingAgent) RunStream(ctx context.Context, messages []Message, opts ...RunOption) (<-chan ResponseUpdate, error) {
    return d.inner.RunStream(ctx, messages, opts...)
}

// NewSession forwards to the inner agent's NewSession method.
func (d *DelegatingAgent) NewSession(ctx context.Context) (Session, error) {
    return d.inner.NewSession(ctx)
}

// RestoreSession forwards to the inner agent's RestoreSession method.
func (d *DelegatingAgent) RestoreSession(ctx context.Context, data json.RawMessage) (Session, error) {
    return d.inner.RestoreSession(ctx, data)
}

// GetService forwards to the inner agent's GetService method.
func (d *DelegatingAgent) GetService(serviceType reflect.Type) interface{} {
    return d.inner.GetService(serviceType)
}
```

Context references:
* Research (Lines 247-310) - DelegatingAgent implementation
* .NET DelegatingAIAgent.cs pattern

Dependencies:
* Existing Agent interface in agent/agent.go

### Step 2.2: Add unit tests for DelegatingAgent

Create comprehensive tests for the DelegatingAgent type.

Files:
* `go/agent/delegating_test.go` - New test file

Success criteria:
* Test that all methods forward correctly to inner agent
* Test Inner() returns the wrapped agent
* Test that compile-time interface check works
* Use mock agent for isolation

Implementation:

```go
// Copyright (c) Microsoft. All rights reserved.

package agent

import (
    "context"
    "encoding/json"
    "reflect"
    "testing"
)

type mockAgent struct {
    id          string
    name        string
    description string
    runCalled   bool
    streamCalled bool
}

func (m *mockAgent) ID() string                        { return m.id }
func (m *mockAgent) Name() string                      { return m.name }
func (m *mockAgent) Description() string               { return m.description }
func (m *mockAgent) Metadata() AIAgentMetadata         { return AIAgentMetadata{} }
func (m *mockAgent) Run(ctx context.Context, msgs []Message, opts ...RunOption) (*Response, error) {
    m.runCalled = true
    return &Response{}, nil
}
func (m *mockAgent) RunStream(ctx context.Context, msgs []Message, opts ...RunOption) (<-chan ResponseUpdate, error) {
    m.streamCalled = true
    ch := make(chan ResponseUpdate)
    close(ch)
    return ch, nil
}
func (m *mockAgent) NewSession(ctx context.Context) (Session, error) { return nil, nil }
func (m *mockAgent) RestoreSession(ctx context.Context, data json.RawMessage) (Session, error) { return nil, nil }
func (m *mockAgent) GetService(t reflect.Type) interface{} { return nil }

func TestDelegatingAgent_ForwardsAllMethods(t *testing.T) {
    inner := &mockAgent{id: "test-id", name: "test-name", description: "test-desc"}
    delegating := NewDelegatingAgent(inner)

    if delegating.ID() != "test-id" {
        t.Errorf("ID() = %q, want %q", delegating.ID(), "test-id")
    }
    if delegating.Name() != "test-name" {
        t.Errorf("Name() = %q, want %q", delegating.Name(), "test-name")
    }
    if delegating.Description() != "test-desc" {
        t.Errorf("Description() = %q, want %q", delegating.Description(), "test-desc")
    }
}

func TestDelegatingAgent_Inner(t *testing.T) {
    inner := &mockAgent{id: "inner-id"}
    delegating := NewDelegatingAgent(inner)

    if delegating.Inner() != inner {
        t.Error("Inner() did not return the wrapped agent")
    }
}

func TestDelegatingAgent_Run(t *testing.T) {
    inner := &mockAgent{}
    delegating := NewDelegatingAgent(inner)

    _, err := delegating.Run(context.Background(), nil)
    if err != nil {
        t.Errorf("Run() error = %v", err)
    }
    if !inner.runCalled {
        t.Error("Run() did not forward to inner agent")
    }
}
```

Context references:
* Go testing conventions

Dependencies:
* Step 2.1 completion

### Step 2.3: Validate phase changes

Validation commands:
* `go test ./agent/...` - Run agent package tests

## Implementation Phase 3: Middleware Chain Implementation

<!-- parallelizable: false -->

### Step 3.1: Implement middleware chain composition

Create utilities for composing multiple middlewares into a single chain. Follow gRPC's ChainUnaryInterceptor pattern.

Files:
* `go/agent/chain.go` - New file for chain utilities

Success criteria:
* ChainAgentMiddleware composes multiple AgentMiddleware into one
* ChainFunctionMiddleware composes multiple FunctionMiddleware into one
* Middlewares execute in order (first added = first to run)
* Terminal handler is called at end of chain

Implementation:

```go
// Copyright (c) Microsoft. All rights reserved.

package agent

import (
    "context"
)

// ChainAgentMiddleware composes multiple AgentMiddleware into a single middleware.
// Middlewares execute in the order provided: first middleware runs first,
// and its next() calls the second middleware, and so on.
func ChainAgentMiddleware(middlewares ...AgentMiddleware) AgentMiddleware {
    n := len(middlewares)
    if n == 0 {
        return AgentMiddlewareFunc(func(ctx context.Context, agentCtx *AgentContext, next AgentHandler) error {
            return next(ctx, agentCtx)
        })
    }
    if n == 1 {
        return middlewares[0]
    }

    return AgentMiddlewareFunc(func(ctx context.Context, agentCtx *AgentContext, next AgentHandler) error {
        // Build chain from the last middleware backwards
        chain := next
        for i := n - 1; i >= 0; i-- {
            mw := middlewares[i]
            currentNext := chain
            chain = func(ctx context.Context, agentCtx *AgentContext) error {
                return mw.Process(ctx, agentCtx, currentNext)
            }
        }
        return chain(ctx, agentCtx)
    })
}

// ChainFunctionMiddleware composes multiple FunctionMiddleware into a single middleware.
func ChainFunctionMiddleware(middlewares ...FunctionMiddleware) FunctionMiddleware {
    n := len(middlewares)
    if n == 0 {
        return FunctionMiddlewareFunc(func(ctx context.Context, funcCtx *FunctionContext, next FunctionHandler) error {
            return next(ctx, funcCtx)
        })
    }
    if n == 1 {
        return middlewares[0]
    }

    return FunctionMiddlewareFunc(func(ctx context.Context, funcCtx *FunctionContext, next FunctionHandler) error {
        chain := next
        for i := n - 1; i >= 0; i-- {
            mw := middlewares[i]
            currentNext := chain
            chain = func(ctx context.Context, funcCtx *FunctionContext) error {
                return mw.Process(ctx, funcCtx, currentNext)
            }
        }
        return chain(ctx, funcCtx)
    })
}
```

Context references:
* Research (Lines 315-345) - Chain composition pattern
* gRPC ChainUnaryInterceptor analysis

Dependencies:
* Steps 1.2, 1.3 completion

### Step 3.2: Create MiddlewareAgent decorator

Create an agent decorator that applies middleware to Run/RunStream calls.

Files:
* `go/agent/middleware_agent.go` - New file for middleware-enabled agent

Success criteria:
* MiddlewareAgent decorates an inner agent with middleware chain
* Run() builds AgentContext and executes middleware chain
* RunStream() sets IsStreaming=true and executes same chain
* Correctly propagates response from AgentContext back to caller

Implementation:

```go
// Copyright (c) Microsoft. All rights reserved.

package agent

import (
    "context"
    "encoding/json"
    "reflect"
)

// MiddlewareAgent wraps an agent with middleware that intercepts Run and RunStream calls.
type MiddlewareAgent struct {
    inner      Agent
    middleware AgentMiddleware
}

// Compile-time check that MiddlewareAgent implements Agent.
var _ Agent = (*MiddlewareAgent)(nil)

// NewMiddlewareAgent creates an agent that applies the given middleware.
func NewMiddlewareAgent(inner Agent, middlewares ...AgentMiddleware) *MiddlewareAgent {
    return &MiddlewareAgent{
        inner:      inner,
        middleware: ChainAgentMiddleware(middlewares...),
    }
}

func (m *MiddlewareAgent) ID() string                        { return m.inner.ID() }
func (m *MiddlewareAgent) Name() string                      { return m.inner.Name() }
func (m *MiddlewareAgent) Description() string               { return m.inner.Description() }
func (m *MiddlewareAgent) Metadata() AIAgentMetadata         { return m.inner.Metadata() }
func (m *MiddlewareAgent) NewSession(ctx context.Context) (Session, error) { 
    return m.inner.NewSession(ctx) 
}
func (m *MiddlewareAgent) RestoreSession(ctx context.Context, data json.RawMessage) (Session, error) { 
    return m.inner.RestoreSession(ctx, data) 
}
func (m *MiddlewareAgent) GetService(serviceType reflect.Type) interface{} { 
    return m.inner.GetService(serviceType) 
}

// Run executes the agent through the middleware chain.
func (m *MiddlewareAgent) Run(ctx context.Context, messages []Message, opts ...RunOption) (*Response, error) {
    agentCtx := &AgentContext{
        Agent:       m.inner,
        Messages:    messages,
        Options:     ApplyRunOptions(opts...),
        Metadata:    make(map[string]any),
        IsStreaming: false,
    }

    // Terminal handler calls the actual agent
    terminal := func(ctx context.Context, agentCtx *AgentContext) error {
        resp, err := agentCtx.Agent.Run(ctx, agentCtx.Messages, WithRunConfig(agentCtx.Options))
        if err != nil {
            return err
        }
        agentCtx.Response = resp
        return nil
    }

    err := m.middleware.Process(ctx, agentCtx, terminal)
    if err != nil {
        return nil, err
    }
    return agentCtx.Response, nil
}

// RunStream executes the agent streaming through the middleware chain.
func (m *MiddlewareAgent) RunStream(ctx context.Context, messages []Message, opts ...RunOption) (<-chan ResponseUpdate, error) {
    agentCtx := &AgentContext{
        Agent:       m.inner,
        Messages:    messages,
        Options:     ApplyRunOptions(opts...),
        Metadata:    make(map[string]any),
        IsStreaming: true,
    }

    // Terminal handler calls the actual streaming agent
    terminal := func(ctx context.Context, agentCtx *AgentContext) error {
        stream, err := agentCtx.Agent.RunStream(ctx, agentCtx.Messages, WithRunConfig(agentCtx.Options))
        if err != nil {
            return err
        }
        agentCtx.Stream = stream
        return nil
    }

    err := m.middleware.Process(ctx, agentCtx, terminal)
    if err != nil {
        return nil, err
    }
    return agentCtx.Stream, nil
}
```

Context references:
* Research middleware execution patterns

Dependencies:
* Steps 3.1, 1.1-1.3 completion

### Step 3.3: Add middleware chain tests

Create tests for middleware chain composition and MiddlewareAgent.

Files:
* `go/agent/chain_test.go` - New test file
* `go/agent/middleware_agent_test.go` - New test file

Success criteria:
* Test middleware execution order
* Test short-circuit behavior when middleware doesn't call next
* Test response modification in middleware
* Test streaming path through middleware

Context references:
* Go testing patterns

Dependencies:
* Steps 3.1, 3.2 completion

## Implementation Phase 4: AgentBuilder.Use() Extension

<!-- parallelizable: false -->

### Step 4.1: Add AgentFactory type and Use() method to chatagent.Builder

Extend the existing chatagent.Builder to support middleware chaining via Use() method.

Files:
* `go/chatagent/builder.go` - Modify existing file

Success criteria:
* AgentFactory type alias for `func(agent.Agent) agent.Agent`
* Use() method appends factory to builder's factory list
* Build() applies factories in reverse order (first Use = outermost)
* Backward compatible (existing code works without Use)

Implementation additions to builder.go:

```go
// AgentFactory creates a decorated agent from an inner agent.
type AgentFactory func(inner agent.Agent) agent.Agent

// In Builder struct, add:
// factories []AgentFactory

// Use adds a decorator factory to the agent pipeline.
// Factories are applied in reverse order: first Use() becomes the outermost decorator.
func (b *Builder) Use(factory AgentFactory) *Builder {
    if b.err != nil {
        return b
    }
    b.factories = append(b.factories, factory)
    return b
}

// In Build() method, after creating the agent:
// result := agent.Agent(a)
// for i := len(b.factories) - 1; i >= 0; i-- {
//     result = b.factories[i](result)
// }
// return result, nil
```

Context references:
* Research (Lines 350-395) - Builder.Use() pattern
* .NET AIAgentBuilder.Use() implementation

Dependencies:
* Phase 2 completion (DelegatingAgent)

### Step 4.2: Create standalone AgentBuilder in agent package

Create a general-purpose agent builder that works with any agent factory.

Files:
* `go/agent/builder.go` - New file

Success criteria:
* AgentBuilder takes an inner agent factory function
* Use() method chains decorator factories
* Build() returns fully decorated agent.Agent
* Works independently of chatagent package

Implementation:

```go
// Copyright (c) Microsoft. All rights reserved.

package agent

// AgentFactory creates a decorated agent from an inner agent.
type AgentFactory func(inner Agent) Agent

// AgentBuilder builds agent pipelines with middleware and decorators.
type AgentBuilder struct {
    innerFactory func() Agent
    factories    []AgentFactory
}

// NewAgentBuilder creates a new agent builder.
// The createAgent function creates the base agent to be decorated.
func NewAgentBuilder(createAgent func() Agent) *AgentBuilder {
    return &AgentBuilder{
        innerFactory: createAgent,
        factories:    make([]AgentFactory, 0),
    }
}

// Use adds a decorator factory to the pipeline.
// Factories are applied in reverse order: first Use() is outermost.
func (b *AgentBuilder) Use(factory AgentFactory) *AgentBuilder {
    b.factories = append(b.factories, factory)
    return b
}

// UseMiddleware adds agent middleware to the pipeline.
// This is a convenience method that wraps the middleware in a MiddlewareAgent.
func (b *AgentBuilder) UseMiddleware(middlewares ...AgentMiddleware) *AgentBuilder {
    return b.Use(func(inner Agent) Agent {
        return NewMiddlewareAgent(inner, middlewares...)
    })
}

// Build creates the configured agent with all decorators applied.
func (b *AgentBuilder) Build() Agent {
    inner := b.innerFactory()

    // Apply in reverse order so first Use() is outermost
    for i := len(b.factories) - 1; i >= 0; i-- {
        inner = b.factories[i](inner)
    }

    return inner
}
```

Context references:
* Research (Lines 395-440) - AgentBuilder pattern

Dependencies:
* Phase 3 completion

### Step 4.3: Add builder tests

Create tests for both chatagent.Builder.Use() and agent.AgentBuilder.

Files:
* `go/chatagent/builder_test.go` - Add Use() tests
* `go/agent/builder_test.go` - New test file

Success criteria:
* Test that Use() chains in correct order
* Test multiple Use() calls
* Test UseMiddleware convenience method
* Test backward compatibility of chatagent.Builder

Context references:
* .NET AIAgentBuilder tests

Dependencies:
* Steps 4.1, 4.2 completion

### Step 4.4: Validate phase changes

Validation commands:
* `go test ./agent/... ./chatagent/...` - Run tests for both packages

## Implementation Phase 5: FunctionMiddleware Integration

<!-- parallelizable: false -->

### Step 5.1: Extend InvocationConfig to accept FunctionMiddleware

Add middleware field to the tool invocation configuration.

Files:
* `go/tool/config.go` - Modify existing file

Success criteria:
* InvocationConfig has Middleware field of type []agent.FunctionMiddleware
* Default configuration has empty middleware slice
* Middleware is applied during tool invocation

Implementation addition to config.go:

```go
import "github.com/microsoft/agent-framework-go/agent"

// In InvocationConfig struct, add:
// Middleware []agent.FunctionMiddleware
```

Context references:
* Research (Lines 545-585) - FunctionMiddleware integration

Dependencies:
* Phase 1 completion (FunctionMiddleware interface)

### Step 5.2: Integrate FunctionMiddleware into toolloop

Modify the tool invocation code to apply function middleware.

Files:
* `go/chatagent/toolloop.go` - Modify existing file
* `go/tool/invoke.go` - Modify existing file

Success criteria:
* Tool invocations pass through middleware chain
* FunctionContext is populated with function name and arguments
* Result and error from FunctionContext are used as invocation result
* Both streaming and non-streaming paths use middleware

Implementation approach:
* Modify `invokeToolCalls` to wrap each tool invocation with middleware
* Build FunctionContext before invocation
* Execute middleware chain with terminal handler calling actual tool

Context references:
* Research (Lines 587-640) - Middleware integration points

Dependencies:
* Step 5.1 completion

### Step 5.3: Add FunctionMiddleware tests

Create tests for function middleware integration.

Files:
* `go/chatagent/toolloop_test.go` - Add middleware tests
* `go/tool/invoke_test.go` - Add middleware tests

Success criteria:
* Test middleware intercepts tool calls
* Test middleware can modify arguments
* Test middleware can override results
* Test middleware can short-circuit execution

Context references:
* Research validation middleware scenario

Dependencies:
* Step 5.2 completion

## Implementation Phase 6: AsTool() Implementation

<!-- parallelizable: true -->

### Step 6.1: Create AsToolOptions type and AsTool() function

Implement agent-to-tool conversion for hierarchical agent patterns.

Files:
* `go/chatagent/astool.go` - New file

Success criteria:
* AsToolOptions configures tool name, description, and argument handling
* AsTool() returns a tool.Tool that invokes the agent
* Supports optional streaming via callback
* Sanitizes agent name for valid tool identifier

Implementation:

```go
// Copyright (c) Microsoft. All rights reserved.

package chatagent

import (
    "context"
    "encoding/json"
    "fmt"
    "regexp"
    "strings"

    "github.com/microsoft/agent-framework-go/agent"
    "github.com/microsoft/agent-framework-go/tool"
)

// AsToolOptions configures agent-to-tool conversion.
type AsToolOptions struct {
    // Name overrides the tool name (defaults to sanitized agent name).
    Name string

    // Description overrides the tool description (defaults to agent description).
    Description string

    // ArgName sets the parameter name for the task input (defaults to "task").
    ArgName string

    // ArgDescription describes the task parameter.
    ArgDescription string

    // StreamCallback receives streaming updates when set.
    // If nil, the tool uses non-streaming invocation.
    StreamCallback func(agent.ResponseUpdate)
}

// AsTool converts an agent to a tool for use by other agents.
// This enables hierarchical agent patterns where one agent can
// delegate work to specialized sub-agents.
func AsTool(a agent.Agent, opts AsToolOptions) tool.Tool {
    name := opts.Name
    if name == "" {
        name = sanitizeAgentName(a.Name())
    }

    desc := opts.Description
    if desc == "" {
        desc = a.Description()
    }
    if desc == "" {
        desc = fmt.Sprintf("Delegate task to %s agent", name)
    }

    argName := opts.ArgName
    if argName == "" {
        argName = "task"
    }

    argDesc := opts.ArgDescription
    if argDesc == "" {
        argDesc = fmt.Sprintf("Task for the %s agent to perform", name)
    }

    // Create parameters schema
    params := map[string]any{
        "type": "object",
        "properties": map[string]any{
            argName: map[string]any{
                "type":        "string",
                "description": argDesc,
            },
        },
        "required": []string{argName},
    }
    paramsJSON, _ := json.Marshal(params)

    return tool.NewFunction(name, desc, paramsJSON, func(ctx context.Context, args json.RawMessage) (string, error) {
        // Parse args to get the task input
        var parsed map[string]string
        if err := json.Unmarshal(args, &parsed); err != nil {
            return "", fmt.Errorf("failed to parse arguments: %w", err)
        }

        input := parsed[argName]
        messages := []agent.Message{agent.NewUserMessage(input)}

        if opts.StreamCallback != nil {
            // Streaming mode
            updates, err := a.RunStream(ctx, messages)
            if err != nil {
                return "", err
            }

            var lastText string
            for update := range updates {
                opts.StreamCallback(update)
                if update.Delta != nil && update.Delta.Text != "" {
                    lastText += update.Delta.Text
                }
            }

            return lastText, nil
        }

        // Non-streaming mode
        resp, err := a.Run(ctx, messages)
        if err != nil {
            return "", err
        }

        return resp.Text(), nil
    })
}

// sanitizeAgentName converts an agent name to a valid tool identifier.
func sanitizeAgentName(name string) string {
    // Replace spaces and special chars with underscores
    re := regexp.MustCompile(`[^a-zA-Z0-9_]`)
    sanitized := re.ReplaceAllString(name, "_")
    
    // Remove consecutive underscores
    sanitized = regexp.MustCompile(`_+`).ReplaceAllString(sanitized, "_")
    
    // Trim underscores from ends
    sanitized = strings.Trim(sanitized, "_")
    
    // Ensure lowercase
    sanitized = strings.ToLower(sanitized)
    
    if sanitized == "" {
        sanitized = "agent"
    }
    
    return sanitized
}
```

Context references:
* Research (Lines 682-750) - AsTool implementation
* Python as_tool() implementation

Dependencies:
* tool.NewFunction availability

### Step 6.2: Add AsTool tests with streaming scenarios

Create comprehensive tests for agent-to-tool conversion.

Files:
* `go/chatagent/astool_test.go` - New test file

Success criteria:
* Test non-streaming invocation
* Test streaming invocation with callback
* Test name sanitization
* Test default values for options
* Test error handling

Context references:
* Research AsTool scenario

Dependencies:
* Step 6.1 completion

### Step 6.3: Validate phase changes

Validation commands:
* `go test ./chatagent/...` - Run chatagent tests

## Implementation Phase 7: Documentation and Examples

<!-- parallelizable: true -->

### Step 7.1: Update go/README.md with middleware documentation

Add middleware usage documentation to the Go package README.

Files:
* `go/README.md` - Modify existing file

Success criteria:
* Document AgentMiddleware interface and usage
* Document FunctionMiddleware interface and usage
* Provide code examples for common patterns
* Document AsTool for hierarchical agents
* Document Builder.Use() for chaining

Content sections to add:
* Middleware section with interface descriptions
* Logging middleware example
* Validation middleware example
* Hierarchical agents with AsTool example

Context references:
* Research configuration examples

Dependencies:
* Phases 1-6 completion

### Step 7.2: Add doc.go comments for new types

Add package-level documentation for new files.

Files:
* `go/agent/middleware.go` - Add package doc
* `go/agent/chain.go` - Add file doc
* `go/chatagent/astool.go` - Add file doc

Success criteria:
* Each new file has appropriate documentation
* Public types have complete godoc comments
* Examples in documentation compile correctly

Context references:
* Go documentation conventions

Dependencies:
* Phases 1-6 completion

## Implementation Phase 8: Validation

<!-- parallelizable: false -->

### Step 8.1: Run full project validation

Execute all validation commands for the project:
* `go build ./...` - Build all packages
* `go vet ./...` - Check for common issues
* `go test ./...` - Run all tests
* `go test -race ./...` - Run with race detector
* `golint ./...` - Run linter if available

### Step 8.2: Fix minor validation issues

Iterate on lint errors, build warnings, and test failures. Apply fixes directly when corrections are straightforward and isolated.

### Step 8.3: Report blocking issues

When validation failures require changes beyond minor fixes:
* Document the issues and affected files
* Provide the user with next steps
* Recommend additional research and planning rather than inline fixes
* Avoid large-scale refactoring within this phase

## Dependencies

* Go 1.21+ for generics support (if used)
* Existing agent.Agent interface
* Existing tool.Tool interface
* context standard library
* encoding/json standard library
* reflect standard library

## Success Criteria

* All middleware interfaces are idiomatic Go and compile correctly
* DelegatingAgent forwards all Agent methods to inner agent
* MiddlewareAgent correctly executes middleware chain for Run and RunStream
* Builder.Use() applies decorators in correct order
* AsTool() creates functional tools from agents
* All existing tests pass
* New tests achieve >80% coverage
* Documentation is complete with examples
