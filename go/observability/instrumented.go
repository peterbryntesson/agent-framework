// Copyright (c) Microsoft. All rights reserved.

package observability

import (
	"context"
	"time"

	"github.com/microsoft/agent-framework-go/chat"

	"go.opentelemetry.io/otel/attribute"
)

// InstrumentedClient wraps a chat.Client with OpenTelemetry instrumentation.
// It provides tracing and metrics for all chat operations.
type InstrumentedClient struct {
	inner   chat.Client
	metrics *Metrics

	// EnableSensitiveData controls whether message content is included in traces.
	// Default is false for security.
	EnableSensitiveData bool
}

// Ensure InstrumentedClient implements chat.Client.
var _ chat.Client = (*InstrumentedClient)(nil)

// NewInstrumentedClient creates an instrumented wrapper around a chat client.
func NewInstrumentedClient(client chat.Client) *InstrumentedClient {
	metrics, _ := NewMetrics()
	return &InstrumentedClient{
		inner:               client,
		metrics:             metrics,
		EnableSensitiveData: false,
	}
}

// NewInstrumentedClientWithMetrics creates an instrumented wrapper with custom metrics.
func NewInstrumentedClientWithMetrics(client chat.Client, metrics *Metrics) *InstrumentedClient {
	return &InstrumentedClient{
		inner:               client,
		metrics:             metrics,
		EnableSensitiveData: false,
	}
}

// GetResponse sends messages to the chat model and returns a complete response.
// The request is traced and metrics are recorded.
func (c *InstrumentedClient) GetResponse(ctx context.Context, messages []chat.Message, options *chat.Options) (*chat.Response, error) {
	metadata := c.inner.Metadata()
	ctx, span := StartChatSpan(ctx, metadata.ProviderName, metadata.ModelID)
	defer span.End()

	// Record request options if available
	if options != nil {
		RecordRequestOptions(span, options.MaxTokens, options.Temperature, options.TopP)
	}

	start := time.Now()

	resp, err := c.inner.GetResponse(ctx, messages, options)

	latency := time.Since(start).Seconds()

	// Record latency metric
	if c.metrics != nil {
		c.metrics.RecordLatency(ctx, latency, metadata.ProviderName, metadata.ModelID)
	}

	if err != nil {
		RecordError(span, err)
		if c.metrics != nil {
			c.metrics.RecordError(ctx, errorTypeName(err), metadata.ProviderName, metadata.ModelID)
		}
		return nil, err
	}

	// Record usage on span
	if resp.Usage != nil {
		RecordUsageDetails(span, resp.Usage)
		if c.metrics != nil {
			c.metrics.RecordTokenUsage(ctx, resp.Usage.InputTokens, resp.Usage.OutputTokens, metadata.ProviderName, metadata.ModelID)
		}
	}

	// Record response attributes
	RecordResponse(span, "", metadata.ModelID, finishReasonToString(resp.FinishReason))

	return resp, nil
}

// GetStreamingResponse sends messages and returns a channel of incremental updates.
// The stream is traced and metrics are recorded upon completion.
func (c *InstrumentedClient) GetStreamingResponse(ctx context.Context, messages []chat.Message, options *chat.Options) (<-chan chat.ResponseUpdate, error) {
	metadata := c.inner.Metadata()
	ctx, span := StartChatSpan(ctx, metadata.ProviderName, metadata.ModelID)

	// Record request options if available
	if options != nil {
		RecordRequestOptions(span, options.MaxTokens, options.Temperature, options.TopP)
	}

	start := time.Now()

	updates, err := c.inner.GetStreamingResponse(ctx, messages, options)
	if err != nil {
		RecordError(span, err)
		span.End()
		if c.metrics != nil {
			c.metrics.RecordError(ctx, errorTypeName(err), metadata.ProviderName, metadata.ModelID)
		}
		return nil, err
	}

	// Wrap channel to instrument completion
	instrumented := make(chan chat.ResponseUpdate, 32)
	go func() {
		defer close(instrumented)
		defer span.End()

		var totalInputTokens, totalOutputTokens int
		var finishReason chat.FinishReason
		var finishReasonSet bool

		for update := range updates {
			// Capture usage information
			if update.Kind == chat.UpdateKindUsage && update.Usage != nil {
				totalInputTokens = update.Usage.InputTokens
				totalOutputTokens = update.Usage.OutputTokens
			}

			// Capture finish reason from message complete updates
			if update.Kind == chat.UpdateKindMessageComplete || update.Kind == chat.UpdateKindDone {
				finishReason = update.FinishReason
				finishReasonSet = true
			}

			// Handle completion
			if update.Kind == chat.UpdateKindDone {
				latency := time.Since(start).Seconds()
				if c.metrics != nil {
					c.metrics.RecordLatency(ctx, latency, metadata.ProviderName, metadata.ModelID)
					if totalInputTokens > 0 || totalOutputTokens > 0 {
						c.metrics.RecordTokenUsage(ctx, totalInputTokens, totalOutputTokens, metadata.ProviderName, metadata.ModelID)
					}
				}

				// Record usage on span
				if totalInputTokens > 0 || totalOutputTokens > 0 {
					RecordUsage(span, totalInputTokens, totalOutputTokens)
				}

				// Record response attributes
				if finishReasonSet {
					RecordResponse(span, "", metadata.ModelID, finishReasonToString(finishReason))
				}
			}

			// Handle errors
			if update.Kind == chat.UpdateKindError && update.Error != nil {
				RecordError(span, update.Error)
				if c.metrics != nil {
					c.metrics.RecordError(ctx, errorTypeName(update.Error), metadata.ProviderName, metadata.ModelID)
				}
			}

			instrumented <- update
		}
	}()

	return instrumented, nil
}

// Metadata returns provider-specific metadata about this client.
func (c *InstrumentedClient) Metadata() chat.ClientMetadata {
	return c.inner.Metadata()
}

// InstrumentedClientOption configures an InstrumentedClient.
type InstrumentedClientOption func(*InstrumentedClient)

// WithSensitiveData enables inclusion of message content in traces.
func WithSensitiveData(enabled bool) InstrumentedClientOption {
	return func(c *InstrumentedClient) {
		c.EnableSensitiveData = enabled
	}
}

// WithMetrics sets custom metrics for the client.
func WithMetrics(metrics *Metrics) InstrumentedClientOption {
	return func(c *InstrumentedClient) {
		c.metrics = metrics
	}
}

// NewInstrumentedClientWithOptions creates an instrumented client with options.
func NewInstrumentedClientWithOptions(client chat.Client, opts ...InstrumentedClientOption) *InstrumentedClient {
	ic := NewInstrumentedClient(client)
	for _, opt := range opts {
		opt(ic)
	}
	return ic
}

// errorTypeName returns a string representation of an error type.
func errorTypeName(err error) string {
	if err == nil {
		return "unknown"
	}
	return "error"
}

// finishReasonToString converts a FinishReason to its string representation.
func finishReasonToString(reason chat.FinishReason) string {
	switch reason {
	case chat.FinishReasonStop:
		return "stop"
	case chat.FinishReasonLength:
		return "length"
	case chat.FinishReasonToolCalls:
		return "tool_calls"
	case chat.FinishReasonContentFilter:
		return "content_filter"
	default:
		return "unknown"
	}
}

// InstrumentClient is a convenience function to wrap a client with instrumentation.
func InstrumentClient(client chat.Client) chat.Client {
	return NewInstrumentedClient(client)
}

// InstrumentClientWithSensitiveData wraps a client with instrumentation that includes message content.
func InstrumentClientWithSensitiveData(client chat.Client) chat.Client {
	return NewInstrumentedClientWithOptions(client, WithSensitiveData(true))
}

// clientMetricsKey is the context key for passing metrics to callbacks.
type clientMetricsKey struct{}

// ContextWithMetrics returns a context with metrics attached.
func ContextWithMetrics(ctx context.Context, metrics *Metrics) context.Context {
	return context.WithValue(ctx, clientMetricsKey{}, metrics)
}

// MetricsFromContext retrieves metrics from context, or returns default metrics.
func MetricsFromContext(ctx context.Context) *Metrics {
	if m, ok := ctx.Value(clientMetricsKey{}).(*Metrics); ok {
		return m
	}
	return DefaultMetrics()
}

// RecordAgentRunFromContext records an agent run metric using context metadata.
func RecordAgentRunFromContext(ctx context.Context, providerName, modelID string) {
	if m := MetricsFromContext(ctx); m != nil {
		m.RecordAgentRun(ctx, providerName, modelID)
	}
}

// RecordTokenUsageFromContext records token usage from context.
func RecordTokenUsageFromContext(ctx context.Context, inputTokens, outputTokens int, providerName, modelID string) {
	if m := MetricsFromContext(ctx); m != nil {
		m.RecordTokenUsage(ctx, inputTokens, outputTokens, providerName, modelID)
	}
}

// RecordLatencyFromContext records latency from context.
func RecordLatencyFromContext(ctx context.Context, latencySeconds float64, providerName, modelID string) {
	if m := MetricsFromContext(ctx); m != nil {
		m.RecordLatency(ctx, latencySeconds, providerName, modelID)
	}
}

// RecordErrorFromContext records an error from context.
func RecordErrorFromContext(ctx context.Context, errorType, providerName, modelID string) {
	if m := MetricsFromContext(ctx); m != nil {
		m.RecordError(ctx, errorType, providerName, modelID)
	}
}

// RecordToolInvocationFromContext records a tool invocation from context.
func RecordToolInvocationFromContext(ctx context.Context, toolName string, success bool) {
	if m := MetricsFromContext(ctx); m != nil {
		m.RecordToolInvocation(ctx, toolName, success)
	}
}

// InstrumentedAgentClient is a helper for recording agent-level metrics.
type InstrumentedAgentClient struct {
	metrics *Metrics
}

// NewInstrumentedAgentClient creates a helper for agent-level instrumentation.
func NewInstrumentedAgentClient() *InstrumentedAgentClient {
	metrics, _ := NewMetrics()
	return &InstrumentedAgentClient{metrics: metrics}
}

// StartRun starts recording for an agent run, returning a function to call on completion.
func (c *InstrumentedAgentClient) StartRun(ctx context.Context, agentID, agentName, providerName, modelID string) (context.Context, func(err error, usage *chat.UsageDetails)) {
	newCtx, span := StartAgentSpanWithID(ctx, agentID, agentName, providerName)
	span.SetAttributes(attribute.String(GenAIRequestModelKey, modelID))

	start := time.Now()

	if c.metrics != nil {
		c.metrics.RecordAgentRun(ctx, providerName, modelID)
	}

	return newCtx, func(err error, usage *chat.UsageDetails) {
		latency := time.Since(start).Seconds()

		if c.metrics != nil {
			c.metrics.RecordLatency(ctx, latency, providerName, modelID)
		}

		if err != nil {
			RecordError(span, err)
			if c.metrics != nil {
				c.metrics.RecordError(ctx, errorTypeName(err), providerName, modelID)
			}
		}

		if usage != nil {
			RecordUsageDetails(span, usage)
			if c.metrics != nil {
				c.metrics.RecordTokenUsage(ctx, usage.InputTokens, usage.OutputTokens, providerName, modelID)
			}
		}

		span.End()
	}
}
