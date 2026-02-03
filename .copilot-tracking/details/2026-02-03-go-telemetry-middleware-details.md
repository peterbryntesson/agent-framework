<!-- markdownlint-disable-file -->
# Implementation Details: TelemetryMiddleware for OpenTelemetry Observability

## Context Reference

Sources:
* [2026-02-03-go-middleware-followup-research.md](../research/2026-02-03-go-middleware-followup-research.md) - Design recommendations
* [go/observability/instrumented.go](../../go/observability/instrumented.go) - InstrumentedClient pattern
* [go/agent/middleware.go](../../go/agent/middleware.go) - Middleware interface definitions

## Implementation Phase 1: TelemetryMiddleware Core

<!-- parallelizable: false -->

### Step 1.1: Create TelemetryMiddleware struct and options

Create `go/observability/agent_middleware.go` with the TelemetryMiddleware struct and functional options pattern.

Files:
* go/observability/agent_middleware.go - New file with middleware implementation

Implementation:

```go
// Copyright (c) Microsoft. All rights reserved.

package observability

import (
    "context"
    "time"

    "github.com/microsoft/agent-framework-go/agent"

    "go.opentelemetry.io/otel/attribute"
    "go.opentelemetry.io/otel/trace"
)

// TelemetryMiddleware provides OpenTelemetry instrumentation for agent invocations.
// It creates spans and records metrics for each agent run, integrating with
// the existing observability infrastructure.
type TelemetryMiddleware struct {
    metrics             *Metrics
    enableSensitiveData bool
    sourceName          string
}

// TelemetryOption configures TelemetryMiddleware behavior.
type TelemetryOption func(*TelemetryMiddleware)

// WithMetrics configures the metrics instance to use.
// If not set, DefaultMetrics() is used.
func WithMetrics(m *Metrics) TelemetryOption {
    return func(tm *TelemetryMiddleware) {
        tm.metrics = m
    }
}

// WithSensitiveData enables recording of message content in traces.
// Default is false for security.
func WithSensitiveData(enabled bool) TelemetryOption {
    return func(tm *TelemetryMiddleware) {
        tm.enableSensitiveData = enabled
    }
}

// WithSourceName sets a custom source name for tracing attribution.
func WithSourceName(name string) TelemetryOption {
    return func(tm *TelemetryMiddleware) {
        tm.sourceName = name
    }
}

// NewTelemetryMiddleware creates a new telemetry middleware with the provided options.
func NewTelemetryMiddleware(opts ...TelemetryOption) *TelemetryMiddleware {
    tm := &TelemetryMiddleware{
        enableSensitiveData: false,
        sourceName:          InstrumentationName,
    }

    for _, opt := range opts {
        opt(tm)
    }

    // Use default metrics if not provided
    if tm.metrics == nil {
        tm.metrics = DefaultMetrics()
    }

    return tm
}
```

Success criteria:
* Struct compiles without errors
* Functional options pattern matches existing Go patterns
* Default metrics integration works

Context references:
* [go/observability/instrumented.go](../../go/observability/instrumented.go) (Lines 14-44) - InstrumentedClient pattern reference

Dependencies:
* Existing Metrics struct from metrics.go
* DefaultMetrics() function from metrics.go

### Step 1.2: Implement Process method for AgentMiddleware interface

Add the Process method to TelemetryMiddleware in `go/observability/agent_middleware.go`.

Files:
* go/observability/agent_middleware.go - Add Process method

Implementation:

```go
// Ensure TelemetryMiddleware implements agent.AgentMiddleware.
var _ agent.AgentMiddleware = (*TelemetryMiddleware)(nil)

// Process implements agent.AgentMiddleware.
// It wraps agent invocations with OpenTelemetry spans and metrics.
func (m *TelemetryMiddleware) Process(ctx context.Context, agentCtx *agent.AgentContext, next agent.AgentHandler) error {
    // Determine operation name based on streaming
    operationName := OperationAgentRun
    if agentCtx.IsStreaming {
        operationName = OperationAgentRunStream
    }

    // Get agent metadata for attributes
    agentName := ""
    agentID := ""
    providerName := ""
    if agentCtx.Agent != nil {
        agentName = agentCtx.Agent.Name()
        agentID = agentCtx.Agent.ID()
        metadata := agentCtx.Agent.Metadata()
        providerName = metadata.ProviderName
    }

    // Try to get model ID from context metadata
    modelID := ""
    if agentCtx.Metadata != nil {
        if id, ok := agentCtx.Metadata["model_id"].(string); ok {
            modelID = id
        }
    }

    // Start span
    ctx, span := StartAgentSpan(ctx, operationName, agentName)
    defer span.End()

    // Record agent attributes
    if agentCtx.Agent != nil {
        span.SetAttributes(
            attribute.String(AgentIDKey, agentID),
            attribute.String(GenAISystemKey, providerName),
        )
        if modelID != "" {
            span.SetAttributes(attribute.String(GenAIRequestModelKey, modelID))
        }
    }

    // Record agent run metric
    if m.metrics != nil {
        m.metrics.RecordAgentRun(ctx, providerName, modelID)
    }

    start := time.Now()

    // Call next handler
    err := next(ctx, agentCtx)

    latency := time.Since(start).Seconds()

    // Record latency metric
    if m.metrics != nil {
        m.metrics.RecordLatency(ctx, latency, providerName, modelID)
    }

    if err != nil {
        RecordError(span, err)
        if m.metrics != nil {
            m.metrics.RecordError(ctx, errorTypeName(err), providerName, modelID)
        }
        return err
    }

    // Record response details
    m.recordResponseDetails(ctx, span, agentCtx, providerName, modelID)

    return nil
}
```

Success criteria:
* Process method compiles without errors
* Span is created with correct operation name
* Metrics are recorded for agent runs and latency
* Errors are properly recorded on the span

Context references:
* [go/observability/otel.go](../../go/observability/otel.go) (Lines 89-103) - StartAgentSpan implementation
* [go/agent/middleware.go](../../go/agent/middleware.go) (Lines 8-20) - AgentMiddleware interface

Dependencies:
* Step 1.1 completion
* StartAgentSpan function from otel.go
* RecordError function from otel.go

### Step 1.3: Add response and streaming attribute recording

Add helper methods to record response details from AgentContext.

Files:
* go/observability/agent_middleware.go - Add recordResponseDetails method

Implementation:

```go
// recordResponseDetails records span attributes from the agent response.
func (m *TelemetryMiddleware) recordResponseDetails(ctx context.Context, span trace.Span, agentCtx *agent.AgentContext, providerName, modelID string) {
    if agentCtx.Response == nil {
        return
    }

    resp := agentCtx.Response

    // Record finish reason
    if resp.FinishReason != "" {
        span.SetAttributes(
            attribute.String(GenAIResponseFinishReasonsKey, string(resp.FinishReason)),
        )
    }

    // Record usage if available
    if resp.Usage != nil {
        RecordUsage(span, resp.Usage.InputTokens, resp.Usage.OutputTokens)

        if m.metrics != nil {
            m.metrics.RecordTokenUsage(ctx, resp.Usage.InputTokens, resp.Usage.OutputTokens, providerName, modelID)
        }
    }
}

// errorTypeName extracts the type name from an error for metrics.
func errorTypeName(err error) string {
    if err == nil {
        return ""
    }
    return fmt.Sprintf("%T", err)
}
```

Note: The `errorTypeName` function may already exist in instrumented.go. Check and reuse if available.

Success criteria:
* Response details are recorded on the span
* Token usage metrics are recorded
* Finish reason is captured as span attribute

Context references:
* [go/observability/otel.go](../../go/observability/otel.go) (Lines 121-127) - RecordUsage function
* [go/observability/instrumented.go](../../go/observability/instrumented.go) (Lines 67-82) - Response recording pattern

Dependencies:
* Step 1.2 completion
* RecordUsage function from otel.go

## Implementation Phase 2: FunctionTelemetryMiddleware

<!-- parallelizable: true -->

### Step 2.1: Create FunctionTelemetryMiddleware for tool call instrumentation

Create FunctionTelemetryMiddleware in `go/observability/function_middleware.go`.

Files:
* go/observability/function_middleware.go - New file with function middleware

Implementation:

```go
// Copyright (c) Microsoft. All rights reserved.

package observability

import (
    "context"
    "time"

    "github.com/microsoft/agent-framework-go/agent"

    "go.opentelemetry.io/otel/attribute"
    "go.opentelemetry.io/otel/trace"
)

// FunctionTelemetryMiddleware provides OpenTelemetry instrumentation for tool/function invocations.
type FunctionTelemetryMiddleware struct {
    metrics             *Metrics
    enableSensitiveData bool
}

// FunctionTelemetryOption configures FunctionTelemetryMiddleware behavior.
type FunctionTelemetryOption func(*FunctionTelemetryMiddleware)

// WithFunctionMetrics configures the metrics instance for function middleware.
func WithFunctionMetrics(m *Metrics) FunctionTelemetryOption {
    return func(fm *FunctionTelemetryMiddleware) {
        fm.metrics = m
    }
}

// WithFunctionSensitiveData enables recording of function arguments in traces.
func WithFunctionSensitiveData(enabled bool) FunctionTelemetryOption {
    return func(fm *FunctionTelemetryMiddleware) {
        fm.enableSensitiveData = enabled
    }
}

// NewFunctionTelemetryMiddleware creates a new function telemetry middleware.
func NewFunctionTelemetryMiddleware(opts ...FunctionTelemetryOption) *FunctionTelemetryMiddleware {
    fm := &FunctionTelemetryMiddleware{
        enableSensitiveData: false,
    }

    for _, opt := range opts {
        opt(fm)
    }

    if fm.metrics == nil {
        fm.metrics = DefaultMetrics()
    }

    return fm
}
```

Success criteria:
* Struct compiles without errors
* Options pattern consistent with TelemetryMiddleware
* Default metrics integration works

Context references:
* [go/agent/middleware.go](../../go/agent/middleware.go) (Lines 30-44) - FunctionMiddleware interface

Dependencies:
* Existing Metrics struct from metrics.go

### Step 2.2: Integrate with existing tool span functions

Add the Process method to FunctionTelemetryMiddleware.

Files:
* go/observability/function_middleware.go - Add Process method

Implementation:

```go
// Ensure FunctionTelemetryMiddleware implements agent.FunctionMiddleware.
var _ agent.FunctionMiddleware = (*FunctionTelemetryMiddleware)(nil)

// Process implements agent.FunctionMiddleware.
func (m *FunctionTelemetryMiddleware) Process(ctx context.Context, funcCtx *agent.FunctionContext, next agent.FunctionHandler) error {
    // Get call ID from metadata if available
    callID := ""
    if funcCtx.Metadata != nil {
        if id, ok := funcCtx.Metadata["call_id"].(string); ok {
            callID = id
        }
    }

    // Start tool span
    ctx, span := StartToolSpan(ctx, funcCtx.FunctionName, callID)
    defer span.End()

    // Record function name
    span.SetAttributes(
        attribute.String(GenAIToolNameKey, funcCtx.FunctionName),
    )

    // Optionally record arguments if sensitive data is enabled
    if m.enableSensitiveData && len(funcCtx.Arguments) > 0 {
        span.SetAttributes(
            attribute.String("gen_ai.tool.arguments", string(funcCtx.Arguments)),
        )
    }

    start := time.Now()

    // Call next handler
    err := next(ctx, funcCtx)

    latency := time.Since(start).Seconds()

    // Record metrics
    success := err == nil && funcCtx.Error == nil
    if m.metrics != nil {
        m.metrics.RecordToolInvocation(ctx, funcCtx.FunctionName, success)
    }

    // Record error if present
    if err != nil {
        RecordError(span, err)
        return err
    }

    if funcCtx.Error != nil {
        RecordError(span, funcCtx.Error)
    }

    return nil
}
```

Note: `StartToolSpan` needs to be added to otel.go if not present. Check existing functions.

Success criteria:
* Process method compiles without errors
* Tool invocations are traced with proper span
* Success/failure metrics are recorded
* Sensitive data is only recorded when enabled

Context references:
* [go/observability/semconv.go](../../go/observability/semconv.go) (Lines 54-60) - Tool-related semantic conventions
* [go/observability/metrics.go](../../go/observability/metrics.go) (Lines 153-165) - RecordToolInvocation method

Dependencies:
* Step 2.1 completion
* StartToolSpan function (add to otel.go if needed)

## Implementation Phase 3: Unit Tests

<!-- parallelizable: true -->

### Step 3.1: Create TelemetryMiddleware unit tests

Create `go/observability/agent_middleware_test.go` with comprehensive tests.

Files:
* go/observability/agent_middleware_test.go - New test file

Implementation:

```go
// Copyright (c) Microsoft. All rights reserved.

package observability

import (
    "context"
    "errors"
    "testing"

    "github.com/microsoft/agent-framework-go/agent"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

// mockAgent implements agent.Agent for testing.
type mockAgent struct {
    id          string
    name        string
    description string
    metadata    agent.AIAgentMetadata
}

func (m *mockAgent) ID() string                      { return m.id }
func (m *mockAgent) Name() string                    { return m.name }
func (m *mockAgent) Description() string             { return m.description }
func (m *mockAgent) Metadata() agent.AIAgentMetadata { return m.metadata }

// Implement remaining interface methods with stubs...

func TestNewTelemetryMiddleware_DefaultsAreSet(t *testing.T) {
    // Act
    mw := NewTelemetryMiddleware()

    // Assert
    assert.NotNil(t, mw)
    assert.False(t, mw.enableSensitiveData)
    assert.Equal(t, InstrumentationName, mw.sourceName)
}

func TestNewTelemetryMiddleware_WithOptions(t *testing.T) {
    // Arrange
    metrics, err := NewMetrics()
    require.NoError(t, err)

    // Act
    mw := NewTelemetryMiddleware(
        WithMetrics(metrics),
        WithSensitiveData(true),
        WithSourceName("custom-source"),
    )

    // Assert
    assert.Equal(t, metrics, mw.metrics)
    assert.True(t, mw.enableSensitiveData)
    assert.Equal(t, "custom-source", mw.sourceName)
}

func TestTelemetryMiddleware_Process_CallsNextHandler(t *testing.T) {
    // Arrange
    mw := NewTelemetryMiddleware()
    called := false

    agentCtx := &agent.AgentContext{
        Agent: &mockAgent{
            id:   "test-id",
            name: "test-agent",
            metadata: agent.AIAgentMetadata{
                ProviderName: "openai",
            },
        },
        Metadata: map[string]any{"model_id": "gpt-4"},
    }

    next := func(ctx context.Context, ac *agent.AgentContext) error {
        called = true
        return nil
    }

    // Act
    err := mw.Process(context.Background(), agentCtx, next)

    // Assert
    assert.NoError(t, err)
    assert.True(t, called)
}

func TestTelemetryMiddleware_Process_RecordsErrorOnFailure(t *testing.T) {
    // Arrange
    mw := NewTelemetryMiddleware()
    expectedErr := errors.New("test error")

    agentCtx := &agent.AgentContext{
        Agent: &mockAgent{
            id:   "test-id",
            name: "test-agent",
        },
    }

    next := func(ctx context.Context, ac *agent.AgentContext) error {
        return expectedErr
    }

    // Act
    err := mw.Process(context.Background(), agentCtx, next)

    // Assert
    assert.Equal(t, expectedErr, err)
}

func TestTelemetryMiddleware_Process_StreamingOperation(t *testing.T) {
    // Arrange
    mw := NewTelemetryMiddleware()

    agentCtx := &agent.AgentContext{
        Agent:       &mockAgent{id: "test-id", name: "test-agent"},
        IsStreaming: true,
    }

    next := func(ctx context.Context, ac *agent.AgentContext) error {
        return nil
    }

    // Act
    err := mw.Process(context.Background(), agentCtx, next)

    // Assert
    assert.NoError(t, err)
}
```

Success criteria:
* All tests pass
* Default configuration is verified
* Options are properly applied
* Error handling is tested
* Streaming vs non-streaming is differentiated

Context references:
* [go/observability/instrumented_test.go](../../go/observability/instrumented_test.go) (Lines 1-100) - Test patterns reference

Dependencies:
* Phase 1 completion
* testify/assert and testify/require packages

### Step 3.2: Create FunctionTelemetryMiddleware unit tests

Create `go/observability/function_middleware_test.go` with comprehensive tests.

Files:
* go/observability/function_middleware_test.go - New test file

Implementation:

```go
// Copyright (c) Microsoft. All rights reserved.

package observability

import (
    "context"
    "encoding/json"
    "errors"
    "testing"

    "github.com/microsoft/agent-framework-go/agent"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestNewFunctionTelemetryMiddleware_DefaultsAreSet(t *testing.T) {
    // Act
    mw := NewFunctionTelemetryMiddleware()

    // Assert
    assert.NotNil(t, mw)
    assert.False(t, mw.enableSensitiveData)
}

func TestNewFunctionTelemetryMiddleware_WithOptions(t *testing.T) {
    // Arrange
    metrics, err := NewMetrics()
    require.NoError(t, err)

    // Act
    mw := NewFunctionTelemetryMiddleware(
        WithFunctionMetrics(metrics),
        WithFunctionSensitiveData(true),
    )

    // Assert
    assert.Equal(t, metrics, mw.metrics)
    assert.True(t, mw.enableSensitiveData)
}

func TestFunctionTelemetryMiddleware_Process_CallsNextHandler(t *testing.T) {
    // Arrange
    mw := NewFunctionTelemetryMiddleware()
    called := false

    funcCtx := &agent.FunctionContext{
        FunctionName: "get_weather",
        Arguments:    json.RawMessage(`{"location": "Seattle"}`),
    }

    next := func(ctx context.Context, fc *agent.FunctionContext) error {
        called = true
        fc.Result = map[string]string{"temp": "72F"}
        return nil
    }

    // Act
    err := mw.Process(context.Background(), funcCtx, next)

    // Assert
    assert.NoError(t, err)
    assert.True(t, called)
}

func TestFunctionTelemetryMiddleware_Process_RecordsError(t *testing.T) {
    // Arrange
    mw := NewFunctionTelemetryMiddleware()
    expectedErr := errors.New("function failed")

    funcCtx := &agent.FunctionContext{
        FunctionName: "get_weather",
    }

    next := func(ctx context.Context, fc *agent.FunctionContext) error {
        return expectedErr
    }

    // Act
    err := mw.Process(context.Background(), funcCtx, next)

    // Assert
    assert.Equal(t, expectedErr, err)
}

func TestFunctionTelemetryMiddleware_Process_RecordsFunctionContextError(t *testing.T) {
    // Arrange
    mw := NewFunctionTelemetryMiddleware()

    funcCtx := &agent.FunctionContext{
        FunctionName: "get_weather",
    }

    next := func(ctx context.Context, fc *agent.FunctionContext) error {
        fc.Error = errors.New("context error")
        return nil
    }

    // Act
    err := mw.Process(context.Background(), funcCtx, next)

    // Assert
    assert.NoError(t, err) // Handler returned nil, error is in context
}
```

Success criteria:
* All tests pass
* Default configuration is verified
* Options are properly applied
* Both handler errors and context errors are tested

Context references:
* [go/agent/middleware_context.go](../../go/agent/middleware_context.go) (Lines 43-55) - FunctionContext struct

Dependencies:
* Phase 2 completion
* testify packages

## Implementation Phase 4: Documentation

<!-- parallelizable: true -->

### Step 4.1: Update go/README.md with TelemetryMiddleware usage examples

Update the README with middleware-based observability examples.

Files:
* go/README.md - Add observability middleware section

Add to the Middleware section of go/README.md:

```markdown
### Observability Middleware

The framework provides telemetry middleware for OpenTelemetry integration:

​```go
import "github.com/microsoft/agent-framework-go/observability"

// Create telemetry middleware with default settings
telemetry := observability.NewTelemetryMiddleware()

// Or with custom options
telemetry := observability.NewTelemetryMiddleware(
    observability.WithSensitiveData(false), // Don't log message content
    observability.WithSourceName("my-app"),
)

// Add to agent builder
agent := chatagent.NewBuilder(client).
    UseMiddleware(telemetry).
    BuildAgent()
​```

#### Function Telemetry

For tool/function call instrumentation:

​```go
functionTelemetry := observability.NewFunctionTelemetryMiddleware()

agent := chatagent.NewBuilder(client).
    UseFunctionMiddleware(functionTelemetry).
    BuildAgent()
​```

#### Telemetry Data Captured

| Category | Data |
|----------|------|
| **Spans** | `agent.run`, `agent.run_stream`, `tool.call` |
| **Attributes** | agent.id, agent.name, provider, model, tokens, finish_reason |
| **Metrics** | agent.runs count, input/output tokens, latency histogram, error count |
```

Success criteria:
* README includes clear usage examples
* Both TelemetryMiddleware and FunctionTelemetryMiddleware are documented
* Captured telemetry data is summarized

Context references:
* [go/README.md](../../go/README.md) - Existing README structure

Dependencies:
* Phase 1 and Phase 2 completion for accurate API documentation

## Implementation Phase 5: Validation

<!-- parallelizable: false -->

### Step 5.1: Run full project validation

Execute all validation commands for the Go packages:

Validation commands:
* `go build ./...` - Verify all packages compile
* `go test ./observability/...` - Run observability package tests
* `go vet ./observability/...` - Static analysis
* `go test ./agent/...` - Ensure agent package still works

### Step 5.2: Fix minor validation issues

Iterate on any build errors, test failures, or vet warnings. Apply fixes directly when corrections are straightforward and isolated.

### Step 5.3: Report blocking issues

When validation failures require changes beyond minor fixes:
* Document the issues and affected files
* Provide the user with next steps
* Recommend additional research and planning rather than inline fixes
* Avoid large-scale refactoring within this phase

## Dependencies

* Go 1.21+
* go.opentelemetry.io/otel packages
* github.com/stretchr/testify for testing

## Success Criteria

* All new files compile without errors
* All unit tests pass
* `go vet` reports no issues
* README documentation is accurate and helpful
* Middleware integrates correctly with existing observability infrastructure
