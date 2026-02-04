// Copyright (c) Microsoft. All rights reserved.

package workflow

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

func setupTestTracer(t *testing.T) *tracetest.InMemoryExporter {
	exporter := tracetest.NewInMemoryExporter()
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSyncer(exporter),
	)
	otel.SetTracerProvider(tp)
	t.Cleanup(func() {
		_ = tp.Shutdown(context.Background())
	})
	return exporter
}

func TestStartWorkflowRunSpan(t *testing.T) {
	// Arrange
	exporter := setupTestTracer(t)

	// Build workflow using builder
	startExec := NewExecutorFunc("start", func(ctx context.Context, wCtx *WorkflowContext) error {
		return nil
	})
	wf, err := NewBuilder(startExec).
		WithName("test-workflow").
		WithDescription("Test workflow").
		Build()
	require.NoError(t, err)

	// Act
	ctx, span := StartWorkflowRunSpan(context.Background(), wf, "run-123")
	span.End()

	// Assert
	spans := exporter.GetSpans()
	require.Len(t, spans, 1)
	assert.Equal(t, SpanWorkflowRun, spans[0].Name)
	assert.NotNil(t, ctx, "expected non-nil context")

	// Verify attributes
	attrs := spans[0].Attributes
	assert.Contains(t, attrsToMap(attrs), AttrWorkflowID)
	assert.Contains(t, attrsToMap(attrs), AttrWorkflowName)
	assert.Contains(t, attrsToMap(attrs), AttrRunID)
}

func TestStartSuperstepSpan(t *testing.T) {
	// Arrange
	exporter := setupTestTracer(t)

	// Act
	_, span := StartSuperstepSpan(context.Background(), 5)
	span.End()

	// Assert
	spans := exporter.GetSpans()
	require.Len(t, spans, 1)
	assert.Equal(t, SpanSuperstep, spans[0].Name)

	attrs := attrsToMap(spans[0].Attributes)
	assert.Contains(t, attrs, AttrSuperstep)
}

func TestStartExecutorSpan(t *testing.T) {
	// Arrange
	exporter := setupTestTracer(t)

	// Act
	_, span := StartExecutorSpan(context.Background(), "exec-1", 10)
	span.End()

	// Assert
	spans := exporter.GetSpans()
	require.Len(t, spans, 1)
	assert.Equal(t, SpanExecutorProcess, spans[0].Name)

	attrs := attrsToMap(spans[0].Attributes)
	assert.Contains(t, attrs, AttrExecutorID)
	assert.Contains(t, attrs, AttrMessageCount)
}

func TestStartMessageSendSpan(t *testing.T) {
	// Arrange
	exporter := setupTestTracer(t)

	// Act
	_, span := StartMessageSendSpan(context.Background(), "sender", "receiver")
	span.End()

	// Assert
	spans := exporter.GetSpans()
	require.Len(t, spans, 1)
	assert.Equal(t, SpanMessageSend, spans[0].Name)

	attrs := attrsToMap(spans[0].Attributes)
	assert.Contains(t, attrs, AttrMessageFrom)
	assert.Contains(t, attrs, AttrMessageTo)
}

func TestRecordSuperstepCompletion(t *testing.T) {
	// Arrange
	exporter := setupTestTracer(t)
	_, span := StartSuperstepSpan(context.Background(), 3)

	// Act
	RecordSuperstepCompletion(span, 5, 2, true)
	span.End()

	// Assert
	spans := exporter.GetSpans()
	require.Len(t, spans, 1)

	attrs := attrsToMap(spans[0].Attributes)
	assert.Contains(t, attrs, AttrMessageCount)
	assert.Contains(t, attrs, AttrOutputCount)
	assert.Contains(t, attrs, AttrHaltRequested)
}

func TestRecordExecutorError(t *testing.T) {
	// Arrange
	exporter := setupTestTracer(t)
	_, span := StartExecutorSpan(context.Background(), "exec-1", 1)

	// Act
	testErr := errors.New("test error")
	RecordExecutorError(span, testErr)
	span.End()

	// Assert
	spans := exporter.GetSpans()
	require.Len(t, spans, 1)

	events := spans[0].Events
	require.NotEmpty(t, events)
	assert.Equal(t, "exception", events[0].Name)
}

func TestGetTracer(t *testing.T) {
	// Act
	tracer := getTracer()

	// Assert
	assert.NotNil(t, tracer)
}

func TestSpanNameConstants(t *testing.T) {
	// Verify all span name constants are defined and non-empty
	assert.NotEmpty(t, SpanWorkflowBuild)
	assert.NotEmpty(t, SpanWorkflowRun)
	assert.NotEmpty(t, SpanSuperstep)
	assert.NotEmpty(t, SpanExecutorProcess)
	assert.NotEmpty(t, SpanMessageSend)
}

func TestAttributeKeyConstants(t *testing.T) {
	// Verify all attribute key constants are defined and non-empty
	assert.NotEmpty(t, AttrWorkflowID)
	assert.NotEmpty(t, AttrWorkflowName)
	assert.NotEmpty(t, AttrRunID)
	assert.NotEmpty(t, AttrSuperstep)
	assert.NotEmpty(t, AttrExecutorID)
	assert.NotEmpty(t, AttrExecutorType)
	assert.NotEmpty(t, AttrMessageCount)
	assert.NotEmpty(t, AttrMessageFrom)
	assert.NotEmpty(t, AttrMessageTo)
	assert.NotEmpty(t, AttrHaltRequested)
	assert.NotEmpty(t, AttrOutputCount)
}

// attrsToMap converts attribute slice to a map of key names for easier assertion.
func attrsToMap(attrs []attribute.KeyValue) map[string]bool {
	result := make(map[string]bool)
	for _, attr := range attrs {
		result[string(attr.Key)] = true
	}
	return result
}
