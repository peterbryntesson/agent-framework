// Copyright (c) Microsoft. All rights reserved.

package workflow

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// TracerName is the name for workflow instrumentation.
const TracerName = "github.com/microsoft/agent-framework-go/workflow"

// Span names for workflow operations.
const (
	SpanWorkflowBuild   = "workflow.build"
	SpanWorkflowRun     = "workflow.run"
	SpanSuperstep       = "workflow.superstep"
	SpanExecutorProcess = "executor.process"
	SpanMessageSend     = "message.send"
)

// Attribute keys for workflow observability.
const (
	AttrWorkflowID    = "workflow.id"
	AttrWorkflowName  = "workflow.name"
	AttrRunID         = "run.id"
	AttrSuperstep     = "superstep.number"
	AttrExecutorID    = "executor.id"
	AttrExecutorType  = "executor.type"
	AttrMessageCount  = "message.count"
	AttrMessageFrom   = "message.from"
	AttrMessageTo     = "message.to"
	AttrHaltRequested = "halt.requested"
	AttrOutputCount   = "output.count"
)

// getTracer returns the tracer for workflow instrumentation.
func getTracer() trace.Tracer {
	return otel.Tracer(TracerName)
}

// StartWorkflowRunSpan starts a root span for workflow execution.
// Returns the context with the span and the span itself for ending.
func StartWorkflowRunSpan(
	ctx context.Context,
	wf *Workflow,
	runID string,
) (context.Context, trace.Span) {
	tracer := getTracer()

	ctx, span := tracer.Start(ctx, SpanWorkflowRun,
		trace.WithAttributes(
			attribute.String(AttrWorkflowID, wf.Name()),
			attribute.String(AttrWorkflowName, wf.Name()),
			attribute.String(AttrRunID, runID),
		),
	)

	return ctx, span
}

// StartSuperstepSpan starts a span for a superstep.
func StartSuperstepSpan(
	ctx context.Context,
	superstep int,
) (context.Context, trace.Span) {
	tracer := getTracer()

	ctx, span := tracer.Start(ctx, SpanSuperstep,
		trace.WithAttributes(
			attribute.Int(AttrSuperstep, superstep),
		),
	)

	return ctx, span
}

// StartExecutorSpan starts a span for executor processing.
// Uses span links instead of parent-child for parallel execution visibility.
func StartExecutorSpan(
	ctx context.Context,
	executorID string,
	messageCount int,
) (context.Context, trace.Span) {
	tracer := getTracer()

	ctx, span := tracer.Start(ctx, SpanExecutorProcess,
		trace.WithAttributes(
			attribute.String(AttrExecutorID, executorID),
			attribute.Int(AttrMessageCount, messageCount),
		),
	)

	return ctx, span
}

// StartMessageSendSpan starts a span for message transmission.
func StartMessageSendSpan(
	ctx context.Context,
	from, to string,
) (context.Context, trace.Span) {
	tracer := getTracer()

	ctx, span := tracer.Start(ctx, SpanMessageSend,
		trace.WithAttributes(
			attribute.String(AttrMessageFrom, from),
			attribute.String(AttrMessageTo, to),
		),
	)

	return ctx, span
}

// RecordSuperstepCompletion records superstep completion metrics on a span.
func RecordSuperstepCompletion(
	span trace.Span,
	messageCount int,
	outputCount int,
	haltRequested bool,
) {
	span.SetAttributes(
		attribute.Int(AttrMessageCount, messageCount),
		attribute.Int(AttrOutputCount, outputCount),
		attribute.Bool(AttrHaltRequested, haltRequested),
	)
}

// RecordExecutorError records an executor error on a span.
func RecordExecutorError(span trace.Span, err error) {
	span.RecordError(err)
}
