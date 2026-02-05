// Copyright (c) Microsoft. All rights reserved.

package observability

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

// SetupConfig configures OpenTelemetry setup.
type SetupConfig struct {
	// ServiceName is the name of the service for telemetry.
	ServiceName string

	// ServiceVersion is the version of the service.
	ServiceVersion string

	// TraceExporter is a custom trace exporter. If nil, OTLP gRPC is used.
	TraceExporter trace.SpanExporter

	// MetricExporter is a custom metric exporter. If nil, OTLP gRPC is used.
	MetricExporter metric.Exporter

	// Sampler controls trace sampling. If nil, AlwaysSample is used.
	Sampler trace.Sampler

	// Propagators controls context propagation. If nil, default propagators are used.
	Propagators propagation.TextMapPropagator

	// EnableTracing controls whether tracing is enabled. Default is true.
	EnableTracing bool

	// EnableMetrics controls whether metrics are enabled. Default is true.
	EnableMetrics bool
}

// DefaultSetupConfig returns a default configuration.
func DefaultSetupConfig(serviceName string) SetupConfig {
	return SetupConfig{
		ServiceName:   serviceName,
		EnableTracing: true,
		EnableMetrics: true,
	}
}

// SetupOption configures the OpenTelemetry setup.
type SetupOption func(*SetupConfig)

// WithServiceVersion sets the service version.
func WithServiceVersion(version string) SetupOption {
	return func(c *SetupConfig) {
		c.ServiceVersion = version
	}
}

// WithTraceExporter sets a custom trace exporter.
func WithTraceExporter(exporter trace.SpanExporter) SetupOption {
	return func(c *SetupConfig) {
		c.TraceExporter = exporter
	}
}

// WithMetricExporter sets a custom metric exporter.
func WithMetricExporter(exporter metric.Exporter) SetupOption {
	return func(c *SetupConfig) {
		c.MetricExporter = exporter
	}
}

// WithSampler sets the trace sampler.
func WithSampler(sampler trace.Sampler) SetupOption {
	return func(c *SetupConfig) {
		c.Sampler = sampler
	}
}

// WithPropagators sets the context propagators.
func WithPropagators(propagators propagation.TextMapPropagator) SetupOption {
	return func(c *SetupConfig) {
		c.Propagators = propagators
	}
}

// WithTracingEnabled sets whether tracing is enabled.
func WithTracingEnabled(enabled bool) SetupOption {
	return func(c *SetupConfig) {
		c.EnableTracing = enabled
	}
}

// WithMetricsEnabled sets whether metrics are enabled.
func WithMetricsEnabled(enabled bool) SetupOption {
	return func(c *SetupConfig) {
		c.EnableMetrics = enabled
	}
}

// Shutdown is a function that shuts down OpenTelemetry providers.
type Shutdown func(context.Context) error

// Setup configures OpenTelemetry with OTLP export.
// Returns a shutdown function that should be called on application exit.
func Setup(ctx context.Context, serviceName string, opts ...SetupOption) (Shutdown, error) {
	cfg := DefaultSetupConfig(serviceName)
	for _, opt := range opts {
		opt(&cfg)
	}

	// Build resource with service information
	res, err := buildResource(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	var shutdownFuncs []func(context.Context) error

	// Setup tracing
	if cfg.EnableTracing {
		tracerShutdown, err := setupTracing(ctx, res, cfg)
		if err != nil {
			return nil, fmt.Errorf("failed to setup tracing: %w", err)
		}
		shutdownFuncs = append(shutdownFuncs, tracerShutdown)
	}

	// Setup metrics
	if cfg.EnableMetrics {
		meterShutdown, err := setupMetrics(ctx, res, cfg)
		if err != nil {
			// Clean up tracing if metrics fail
			for _, fn := range shutdownFuncs {
				_ = fn(ctx)
			}
			return nil, fmt.Errorf("failed to setup metrics: %w", err)
		}
		shutdownFuncs = append(shutdownFuncs, meterShutdown)
	}

	// Setup propagators
	if cfg.Propagators != nil {
		otel.SetTextMapPropagator(cfg.Propagators)
	} else {
		otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
			propagation.TraceContext{},
			propagation.Baggage{},
		))
	}

	// Return combined shutdown function
	return func(ctx context.Context) error {
		var errs []error
		for _, fn := range shutdownFuncs {
			if err := fn(ctx); err != nil {
				errs = append(errs, err)
			}
		}
		if len(errs) > 0 {
			return fmt.Errorf("shutdown errors: %v", errs)
		}
		return nil
	}, nil
}

func buildResource(ctx context.Context, cfg SetupConfig) (*resource.Resource, error) {
	attrs := []resource.Option{
		resource.WithAttributes(
			semconv.ServiceNameKey.String(cfg.ServiceName),
		),
	}

	if cfg.ServiceVersion != "" {
		attrs = append(attrs, resource.WithAttributes(
			semconv.ServiceVersionKey.String(cfg.ServiceVersion),
		))
	}

	attrs = append(attrs, resource.WithFromEnv())
	attrs = append(attrs, resource.WithTelemetrySDK())

	return resource.New(ctx, attrs...)
}

func setupTracing(ctx context.Context, res *resource.Resource, cfg SetupConfig) (func(context.Context) error, error) {
	var exporter trace.SpanExporter
	var err error

	if cfg.TraceExporter != nil {
		exporter = cfg.TraceExporter
	} else {
		exporter, err = otlptracegrpc.New(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to create OTLP trace exporter: %w", err)
		}
	}

	sampler := cfg.Sampler
	if sampler == nil {
		sampler = trace.AlwaysSample()
	}

	provider := trace.NewTracerProvider(
		trace.WithBatcher(exporter),
		trace.WithResource(res),
		trace.WithSampler(sampler),
	)

	otel.SetTracerProvider(provider)

	return provider.Shutdown, nil
}

func setupMetrics(ctx context.Context, res *resource.Resource, cfg SetupConfig) (func(context.Context) error, error) {
	var exporter metric.Exporter
	var err error

	if cfg.MetricExporter != nil {
		exporter = cfg.MetricExporter
	} else {
		exporter, err = otlpmetricgrpc.New(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to create OTLP metric exporter: %w", err)
		}
	}

	provider := metric.NewMeterProvider(
		metric.WithReader(metric.NewPeriodicReader(exporter)),
		metric.WithResource(res),
	)

	otel.SetMeterProvider(provider)

	return provider.Shutdown, nil
}

// SetupTracing configures only OpenTelemetry tracing with OTLP export.
// This is a convenience function for applications that only need tracing.
func SetupTracing(ctx context.Context, serviceName string, opts ...SetupOption) (Shutdown, error) {
	tracingOpts := append([]SetupOption{WithMetricsEnabled(false)}, opts...)
	return Setup(ctx, serviceName, tracingOpts...)
}

// SetupMetrics configures only OpenTelemetry metrics with OTLP export.
// This is a convenience function for applications that only need metrics.
func SetupMetrics(ctx context.Context, serviceName string, opts ...SetupOption) (Shutdown, error) {
	metricsOpts := append([]SetupOption{WithTracingEnabled(false)}, opts...)
	return Setup(ctx, serviceName, metricsOpts...)
}
