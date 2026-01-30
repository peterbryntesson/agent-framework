// Copyright (c) Microsoft. All rights reserved.

// Package observability provides OpenTelemetry integration for the Agent Framework.
//
// This package enables tracing, metrics, and logging for agent operations
// using the OpenTelemetry standard.
package observability

import (
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

// Tracer returns the global tracer for agent framework operations.
func Tracer() trace.Tracer {
	return otel.Tracer("github.com/microsoft/agent-framework-go")
}

// Meter returns the global meter for agent framework metrics.
func Meter() metric.Meter {
	return otel.Meter("github.com/microsoft/agent-framework-go")
}
