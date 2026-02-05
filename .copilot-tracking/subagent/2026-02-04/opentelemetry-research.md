# OpenTelemetry Instrumentation Research for Go Workflow Package

**Date:** 2026-02-04
**Purpose:** Research OpenTelemetry instrumentation patterns for workflow orchestration by analyzing .NET implementation and Go OpenTelemetry best practices.

---

## Executive Summary

This document captures the OpenTelemetry instrumentation patterns used in the .NET workflow implementation and provides recommendations for implementing equivalent instrumentation in the Go workflow package. The Go observability package already has a solid foundation for GenAI/agent instrumentation that can be extended for workflow-specific spans and metrics.

---

## 1. .NET Span Hierarchy Analysis

### 1.1 Activity Names (Span Names)

From [ActivityNames.cs](dotnet/src/Microsoft.Agents.AI.Workflows/Observability/ActivityNames.cs):

| Span Name | Description |
|-----------|-------------|
| `workflow.build` | Workflow construction/validation phase |
| `workflow.run` | Root span for entire workflow execution |
| `message.send` | Individual message transmission between executors |
| `executor.process` | Processing of messages by an executor |
| `edge_group.process` | Edge routing and message delivery |

### 1.2 Span Hierarchy (Parent-Child Relationships)

```
workflow.build
  └── (build validation events)

workflow.run [root]
  ├── edge_group.process (initial message delivery)
  ├── executor.process (UppercaseExecutor)
  │     └── message.send
  │           └── edge_group.process
  ├── executor.process (ReverseTextExecutor)
  │     └── message.send
  └── (repeat per superstep...)
```

### 1.3 Span Kind

- `workflow.run`: Internal (orchestration span)
- `executor.process`: Internal
- `edge_group.process`: Internal
- `message.send`: Internal

### 1.4 Span Links

The .NET implementation uses **span links** (not parent-child) for executor processing spans. From [ActivityExtensions.cs](dotnet/src/Microsoft.Agents.AI.Workflows/Observability/ActivityExtensions.cs):

```csharp
internal static void CreateSourceLinks(this Activity? activity, IReadOnlyDictionary<string, string>? traceContext)
{
    // Extract propagation context from dictionary
    var propagationContext = Propagators.DefaultTextMapPropagator.Extract(...)
    // Create a link to the source activity
    activity.AddLink(new ActivityLink(propagationContext.ActivityContext));
}
```

**Rationale:** Executor processing spans are siblings (parallel execution), not nested. Links represent the causal relationship between message source and processor.

---

## 2. Span Attributes Analysis

### 2.1 Tags (Span Attributes)

From [Tags.cs](dotnet/src/Microsoft.Agents.AI.Workflows/Observability/Tags.cs):

| Attribute Key | Type | Description |
|---------------|------|-------------|
| `workflow.id` | string | Unique workflow identifier |
| `workflow.name` | string | Human-readable workflow name |
| `workflow.description` | string | Workflow description |
| `workflow.definition` | string | Workflow definition/graph structure |
| `run.id` | string | Unique run/execution identifier |
| `executor.id` | string | Executor identifier |
| `executor.type` | string | Executor type/class name |
| `message.type` | string | Message type being processed |
| `message.source_id` | string | Source executor ID |
| `message.target_id` | string | Target executor ID |
| `edge_group.type` | string | Type of edge group (Direct, FanIn, FanOut) |
| `edge_group.delivered` | bool | Whether message was delivered |
| `edge_group.delivery_status` | string | Detailed delivery status |
| `error.type` | string | Exception type on failure |
| `build.error.message` | string | Build/validation error message |
| `build.error.type` | string | Build error type |

### 2.2 Edge Delivery Status Values

From [EdgeRunnerDeliveryStatus.cs](dotnet/src/Microsoft.Agents.AI.Workflows/Observability/EdgeRunnerDeliveryStatus.cs):

| Status | Description |
|--------|-------------|
| `delivered` | Message successfully delivered |
| `dropped type mismatch` | Target cannot handle message type |
| `dropped target mismatch` | Message intended for different target |
| `dropped condition false` | Edge condition evaluated to false |
| `exception` | Error during delivery |
| `buffered` | Message buffered for fan-in |

---

## 3. Span Events

From [EventNames.cs](dotnet/src/Microsoft.Agents.AI.Workflows/Observability/EventNames.cs):

| Event Name | Description |
|------------|-------------|
| `build.started` | Workflow build phase started |
| `build.validation_completed` | Build validation completed |
| `build.completed` | Workflow successfully built |
| `build.error` | Build failed with error |
| `workflow.started` | Workflow execution started |
| `workflow.completed` | Workflow execution completed |
| `workflow.error` | Workflow execution error |

---

## 4. Existing Go Observability Patterns

### 4.1 Current Instrumentation Infrastructure

The Go `observability` package already provides:

**Tracer/Meter Access:**
```go
const InstrumentationName = "github.com/microsoft/agent-framework-go"

func Tracer() trace.Tracer {
    return otel.Tracer(InstrumentationName)
}

func Meter() metric.Meter {
    return otel.Meter(InstrumentationName)
}
```

**Span Creation Helpers:**
```go
func StartAgentSpan(ctx context.Context, operationName string, agentName string, opts ...trace.SpanStartOption) (context.Context, trace.Span)
func StartChatSpan(ctx context.Context, system string, model string, opts ...trace.SpanStartOption) (context.Context, trace.Span)
func StartToolSpan(ctx context.Context, toolName, callID string, opts ...trace.SpanStartOption) (context.Context, trace.Span)
```

**Error Recording:**
```go
func RecordError(span trace.Span, err error)
func EndSpanWithError(span trace.Span, err error)
```

### 4.2 Existing Metrics

From [semconv.go](go/observability/semconv.go):

| Metric Name | Type | Description |
|-------------|------|-------------|
| `gen_ai.agent.runs` | Counter | Number of agent runs |
| `gen_ai.usage.input_tokens` | Histogram | Input tokens per request |
| `gen_ai.usage.output_tokens` | Histogram | Output tokens per request |
| `gen_ai.request.latency` | Histogram | Request latency (seconds) |
| `gen_ai.errors` | Counter | Number of errors |
| `gen_ai.tool.invocations` | Counter | Tool invocations |

### 4.3 Middleware Pattern

The Go observability package uses a middleware pattern for instrumentation:

```go
type TelemetryMiddleware struct {
    metrics             *Metrics
    enableSensitiveData bool
    sourceName          string
}

func (m *TelemetryMiddleware) Process(ctx context.Context, agentCtx *agent.AgentContext, next agent.AgentHandler) error {
    ctx, span := StartAgentSpan(ctx, operationName, agentName)
    defer span.End()
    // ... record attributes and call next
}
```

---

## 5. Recommended Go Workflow Instrumentation

### 5.1 New Semantic Conventions for Workflow

Add to [semconv.go](go/observability/semconv.go) or new `workflow_semconv.go`:

```go
// Workflow span names
const (
    OperationWorkflowBuild     = "workflow.build"
    OperationWorkflowRun       = "workflow.run"
    OperationSuperstep         = "workflow.superstep"
    OperationExecutorProcess   = "executor.process"
    OperationEdgeProcess       = "edge.process"
    OperationMessageSend       = "message.send"
)

// Workflow attribute keys
const (
    WorkflowIDKey          = "workflow.id"
    WorkflowNameKey        = "workflow.name"
    WorkflowDescriptionKey = "workflow.description"
    RunIDKey               = "run.id"
    SuperstepKey           = "workflow.superstep"
    ExecutorIDKey          = "executor.id"
    ExecutorTypeKey        = "executor.type"
    MessageTypeKey         = "message.type"
    MessageSourceIDKey     = "message.source_id"
    MessageTargetIDKey     = "message.target_id"
    EdgeTypeKey            = "edge.type"
    EdgeDeliveredKey       = "edge.delivered"
    EdgeDeliveryStatusKey  = "edge.delivery_status"
)

// Workflow metric names
const (
    MetricWorkflowRuns       = "workflow.runs"
    MetricSuperstepCount     = "workflow.superstep.count"
    MetricSuperstepDuration  = "workflow.superstep.duration"
    MetricExecutorDuration   = "executor.duration"
    MetricExecutorErrors     = "executor.errors"
    MetricMessagesDelivered  = "workflow.messages.delivered"
    MetricMessagesDropped    = "workflow.messages.dropped"
)
```

### 5.2 Span Helper Functions

Add to new `workflow_otel.go`:

```go
// StartWorkflowRunSpan starts a new span for workflow execution.
func StartWorkflowRunSpan(ctx context.Context, workflowName, runID string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
    spanName := OperationWorkflowRun
    if workflowName != "" {
        spanName = OperationWorkflowRun + " " + workflowName
    }
    return Tracer().Start(ctx, spanName,
        append(opts,
            trace.WithSpanKind(trace.SpanKindInternal),
            trace.WithAttributes(
                attribute.String(WorkflowNameKey, workflowName),
                attribute.String(RunIDKey, runID),
            ),
        )...,
    )
}

// StartSuperstepSpan starts a span for a superstep execution.
func StartSuperstepSpan(ctx context.Context, runID string, superstep int, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
    return Tracer().Start(ctx, OperationSuperstep,
        append(opts,
            trace.WithSpanKind(trace.SpanKindInternal),
            trace.WithAttributes(
                attribute.String(RunIDKey, runID),
                attribute.Int(SuperstepKey, superstep),
            ),
        )...,
    )
}

// StartExecutorSpan starts a span for executor processing.
func StartExecutorSpan(ctx context.Context, executorID string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
    return Tracer().Start(ctx, OperationExecutorProcess,
        append(opts,
            trace.WithSpanKind(trace.SpanKindInternal),
            trace.WithAttributes(
                attribute.String(ExecutorIDKey, executorID),
            ),
        )...,
    )
}

// StartEdgeSpan starts a span for edge routing.
func StartEdgeSpan(ctx context.Context, sourceID, targetID, edgeType string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
    return Tracer().Start(ctx, OperationEdgeProcess,
        append(opts,
            trace.WithSpanKind(trace.SpanKindInternal),
            trace.WithAttributes(
                attribute.String(MessageSourceIDKey, sourceID),
                attribute.String(MessageTargetIDKey, targetID),
                attribute.String(EdgeTypeKey, edgeType),
            ),
        )...,
    )
}
```

### 5.3 Workflow Metrics Struct

Add to new `workflow_metrics.go`:

```go
// WorkflowMetrics holds workflow-specific metric instruments.
type WorkflowMetrics struct {
    WorkflowRuns       metric.Int64Counter
    SuperstepCount     metric.Int64Counter
    SuperstepDuration  metric.Float64Histogram
    ExecutorDuration   metric.Float64Histogram
    ExecutorErrors     metric.Int64Counter
    MessagesDelivered  metric.Int64Counter
    MessagesDropped    metric.Int64Counter
}

// NewWorkflowMetrics creates the workflow metrics instruments.
func NewWorkflowMetrics() (*WorkflowMetrics, error) {
    meter := otel.Meter(InstrumentationName)
    
    workflowRuns, err := meter.Int64Counter(
        MetricWorkflowRuns,
        metric.WithDescription("Number of workflow runs"),
        metric.WithUnit("{run}"),
    )
    if err != nil {
        return nil, fmt.Errorf("failed to create workflow runs counter: %w", err)
    }
    
    // ... create other metrics similarly
    
    return &WorkflowMetrics{
        WorkflowRuns:      workflowRuns,
        // ...
    }, nil
}

// RecordWorkflowRun records a workflow run with attributes.
func (m *WorkflowMetrics) RecordWorkflowRun(ctx context.Context, workflowName, status string) {
    m.WorkflowRuns.Add(ctx, 1,
        metric.WithAttributes(
            attribute.String(WorkflowNameKey, workflowName),
            attribute.String("status", status),
        ),
    )
}

// RecordSuperstep records superstep metrics.
func (m *WorkflowMetrics) RecordSuperstep(ctx context.Context, workflowName string, superstep int, durationSeconds float64) {
    m.SuperstepCount.Add(ctx, 1,
        metric.WithAttributes(
            attribute.String(WorkflowNameKey, workflowName),
            attribute.Int(SuperstepKey, superstep),
        ),
    )
    m.SuperstepDuration.Record(ctx, durationSeconds,
        metric.WithAttributes(
            attribute.String(WorkflowNameKey, workflowName),
        ),
    )
}

// RecordExecutor records executor execution metrics.
func (m *WorkflowMetrics) RecordExecutor(ctx context.Context, executorID string, durationSeconds float64, success bool) {
    m.ExecutorDuration.Record(ctx, durationSeconds,
        metric.WithAttributes(
            attribute.String(ExecutorIDKey, executorID),
        ),
    )
    if !success {
        m.ExecutorErrors.Add(ctx, 1,
            metric.WithAttributes(
                attribute.String(ExecutorIDKey, executorID),
            ),
        )
    }
}

// RecordMessageDelivery records message delivery outcome.
func (m *WorkflowMetrics) RecordMessageDelivery(ctx context.Context, delivered bool, edgeType, reason string) {
    if delivered {
        m.MessagesDelivered.Add(ctx, 1,
            metric.WithAttributes(
                attribute.String(EdgeTypeKey, edgeType),
            ),
        )
    } else {
        m.MessagesDropped.Add(ctx, 1,
            metric.WithAttributes(
                attribute.String(EdgeTypeKey, edgeType),
                attribute.String("reason", reason),
            ),
        )
    }
}
```

### 5.4 Instrumentation Points in WorkflowRunner

Modify [runner.go](go/workflow/runner.go) to add instrumentation:

```go
func (r *WorkflowRunner) Run(ctx context.Context, input string) (*WorkflowResult, error) {
    runID := r.options.runID
    if runID == "" {
        runID = uuid.New().String()
    }

    // Start workflow run span
    ctx, span := observability.StartWorkflowRunSpan(ctx, r.workflow.Name(), runID)
    defer span.End()
    
    // Add workflow started event
    span.AddEvent("workflow.started")

    // ... existing logic ...

    for superstep < r.options.maxSupersteps {
        // Start superstep span
        superstepCtx, superstepSpan := observability.StartSuperstepSpan(ctx, runID, superstep)
        
        newMessages, stepOutputs, err := r.executeSuperstep(superstepCtx, runID, superstep, messages, state)
        
        superstepSpan.End()
        
        if err != nil {
            observability.RecordError(span, err)
            span.AddEvent("workflow.error")
            return nil, err
        }
        // ...
    }

    span.AddEvent("workflow.completed")
    return result, nil
}
```

### 5.5 Executor-Level Instrumentation

```go
func (r *WorkflowRunner) executeSuperstep(...) (...) {
    // Inside the goroutine for each executor:
    go func(exec Executor, msgs []WorkflowMessage) {
        defer wg.Done()

        // Start executor span
        execCtx, execSpan := observability.StartExecutorSpan(ctx, exec.ID())
        defer execSpan.End()
        
        start := time.Now()

        wCtx := newWorkflowContext(execCtx, exec.ID(), runID, superstep, msgs, state)

        if err := exec.Execute(execCtx, wCtx); err != nil {
            observability.RecordError(execSpan, err)
            // ... error handling
            return
        }

        duration := time.Since(start).Seconds()
        
        // Record metrics if available
        if r.metrics != nil {
            r.metrics.RecordExecutor(ctx, exec.ID(), duration, true)
        }

        // ... route messages with edge spans
    }(executor, execMessages)
}
```

---

## 6. Context Propagation Across Async Operations

### 6.1 Go Context Pattern

The Go workflow already passes `context.Context` through the execution chain. The key additions:

1. **Span Context in Context:** The `trace.SpanFromContext(ctx)` function retrieves the current span.

2. **Cross-Goroutine Propagation:** When spawning goroutines for parallel executor execution, pass the parent context:

```go
go func(ctx context.Context, exec Executor, msgs []WorkflowMessage) {
    // ctx carries the span context from parent
    execCtx, execSpan := observability.StartExecutorSpan(ctx, exec.ID())
    // ...
}(ctx, executor, execMessages)
```

3. **Span Links for Parallel Operations:** For parallel executor processing (like .NET), consider using span links instead of parent-child:

```go
func StartExecutorSpanWithLinks(ctx context.Context, executorID string, sourceSpanContext trace.SpanContext) (context.Context, trace.Span) {
    opts := []trace.SpanStartOption{
        trace.WithSpanKind(trace.SpanKindInternal),
        trace.WithAttributes(attribute.String(ExecutorIDKey, executorID)),
    }
    
    // Add link to source span if valid
    if sourceSpanContext.IsValid() {
        opts = append(opts, trace.WithLinks(trace.Link{SpanContext: sourceSpanContext}))
    }
    
    return Tracer().Start(ctx, OperationExecutorProcess, opts...)
}
```

### 6.2 Trace Context in Messages

For message-based context propagation (like .NET's `TraceContext` dictionary):

```go
// Add to WorkflowMessage struct
type WorkflowMessage struct {
    From        string
    To          string
    Content     agent.Message
    Superstep   int
    TraceContext map[string]string // W3C trace context headers
}

// Inject trace context when sending
func injectTraceContext(ctx context.Context, msg *WorkflowMessage) {
    carrier := propagation.MapCarrier{}
    otel.GetTextMapPropagator().Inject(ctx, carrier)
    msg.TraceContext = carrier
}

// Extract trace context when receiving
func extractTraceContext(ctx context.Context, msg *WorkflowMessage) context.Context {
    if msg.TraceContext == nil {
        return ctx
    }
    carrier := propagation.MapCarrier(msg.TraceContext)
    return otel.GetTextMapPropagator().Extract(ctx, carrier)
}
```

---

## 7. Test Strategy for Telemetry

### 7.1 Unit Testing Patterns (from existing Go tests)

**Test Structure:**
```go
func TestStartWorkflowRunSpan_CreatesSpan(t *testing.T) {
    // Arrange
    ctx := context.Background()

    // Act
    newCtx, span := StartWorkflowRunSpan(ctx, "test-workflow", "run-123")
    defer span.End()

    // Assert
    assert.NotNil(t, newCtx)
    assert.NotNil(t, span)
}

func TestWorkflowMetrics_RecordWorkflowRun(t *testing.T) {
    // Arrange
    metrics, err := NewWorkflowMetrics()
    require.NoError(t, err)
    ctx := context.Background()

    // Act - should not panic
    metrics.RecordWorkflowRun(ctx, "test-workflow", "completed")
}
```

### 7.2 Integration Testing with Captured Spans

Use in-memory exporter for testing:

```go
func TestWorkflowRunner_CreatesProperSpanHierarchy(t *testing.T) {
    // Arrange - Set up span recorder
    sr := tracetest.NewSpanRecorder()
    tp := trace.NewTracerProvider(trace.WithSpanProcessor(sr))
    otel.SetTracerProvider(tp)
    defer func() {
        otel.SetTracerProvider(trace.NewNoopTracerProvider())
    }()

    // Build test workflow
    workflow := buildTestWorkflow(t)
    runner := NewRunner(workflow)

    // Act
    result, err := runner.Run(context.Background(), "test input")

    // Assert
    require.NoError(t, err)
    require.NotNil(t, result)

    // Verify spans
    spans := sr.Ended()
    
    // Find workflow.run span
    var workflowSpan *tracetest.SpanStub
    for _, s := range spans {
        if s.Name == OperationWorkflowRun {
            workflowSpan = &s
            break
        }
    }
    require.NotNil(t, workflowSpan, "workflow.run span should exist")

    // Verify workflow.run has expected attributes
    assertHasAttribute(t, workflowSpan, WorkflowNameKey)
    assertHasAttribute(t, workflowSpan, RunIDKey)

    // Verify executor spans exist
    executorSpanCount := 0
    for _, s := range spans {
        if s.Name == OperationExecutorProcess {
            executorSpanCount++
        }
    }
    assert.GreaterOrEqual(t, executorSpanCount, 1, "Should have at least one executor span")
}

func assertHasAttribute(t *testing.T, span *tracetest.SpanStub, key string) {
    for _, attr := range span.Attributes {
        if string(attr.Key) == key {
            return
        }
    }
    t.Errorf("Span %q missing attribute %q", span.Name, key)
}
```

### 7.3 Metrics Testing

```go
func TestWorkflowMetrics_Integration(t *testing.T) {
    // Arrange - Set up metric reader
    reader := metric.NewManualReader()
    mp := metric.NewMeterProvider(metric.WithReader(reader))
    otel.SetMeterProvider(mp)
    defer func() {
        otel.SetMeterProvider(metric.NewNoopMeterProvider())
    }()

    metrics, err := NewWorkflowMetrics()
    require.NoError(t, err)

    ctx := context.Background()

    // Act
    metrics.RecordWorkflowRun(ctx, "test-workflow", "completed")
    metrics.RecordSuperstep(ctx, "test-workflow", 0, 1.5)

    // Assert - Collect metrics
    rm := metricdata.ResourceMetrics{}
    err = reader.Collect(ctx, &rm)
    require.NoError(t, err)

    // Verify workflow.runs counter has value
    // ... metric verification logic
}
```

---

## 8. Implementation Recommendations

### 8.1 File Organization

```
go/observability/
├── otel.go                    # Existing - GenAI spans
├── metrics.go                 # Existing - GenAI metrics
├── semconv.go                 # Existing - Add workflow constants
├── workflow_otel.go           # NEW - Workflow span helpers
├── workflow_metrics.go        # NEW - Workflow metrics
├── workflow_otel_test.go      # NEW - Workflow span tests
└── workflow_metrics_test.go   # NEW - Workflow metrics tests
```

### 8.2 Implementation Order

1. **Phase 1: Semantic Conventions**
   - Add workflow constants to `semconv.go`
   - Define span names, attribute keys, metric names

2. **Phase 2: Span Helpers**
   - Create `workflow_otel.go` with span creation functions
   - Add unit tests

3. **Phase 3: Workflow Metrics**
   - Create `WorkflowMetrics` struct
   - Add recording methods
   - Add unit tests

4. **Phase 4: Runner Instrumentation**
   - Add optional metrics to `WorkflowRunner`
   - Instrument `Run()` and `RunStream()` methods
   - Instrument superstep and executor execution

5. **Phase 5: Integration Tests**
   - Add workflow observability tests
   - Verify span hierarchy and attributes

### 8.3 Configuration Pattern

Follow existing pattern with options:

```go
// WorkflowRunnerOption additions
func WithWorkflowMetrics(metrics *WorkflowMetrics) RunnerOption {
    return func(o *runnerOptions) {
        o.metrics = metrics
    }
}

func WithTracingEnabled(enabled bool) RunnerOption {
    return func(o *runnerOptions) {
        o.enableTracing = enabled
    }
}
```

---

## 9. Summary of Key Findings

| Aspect | .NET Pattern | Go Recommendation |
|--------|--------------|-------------------|
| **Tracer Access** | Static `ActivitySource` per class | Use global `observability.Tracer()` |
| **Span Hierarchy** | workflow.run → executor.process → message.send | Same hierarchy via context propagation |
| **Parallel Spans** | Span links for siblings | Same - use `trace.WithLinks()` |
| **Metrics** | No workflow-specific metrics found | Add WorkflowMetrics struct |
| **Context Propagation** | TraceContext dictionary in messages | W3C propagation via MapCarrier |
| **Error Recording** | CaptureException extension | Use existing `RecordError()` |
| **Events** | AddEvent for lifecycle markers | Use `span.AddEvent()` |
| **Attributes** | Tags.* constants | WorkflowNameKey, etc. constants |

---

## References

- [OpenTelemetry Semantic Conventions for GenAI](https://opentelemetry.io/docs/specs/semconv/gen-ai/)
- [go.opentelemetry.io/otel Documentation](https://pkg.go.dev/go.opentelemetry.io/otel)
- [.NET ActivitySource API](https://learn.microsoft.com/en-us/dotnet/core/diagnostics/distributed-tracing-instrumentation-walkthroughs)
