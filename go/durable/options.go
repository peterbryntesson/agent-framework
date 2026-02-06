// Copyright (c) Microsoft. All rights reserved.

package durable

import (
	"encoding/json"
	"time"

	"github.com/microsoft/agent-framework-go/agent"
)

const (
	durableCorrelationIDKey   = "durable_correlation_id"
	durableResponseTypeKey    = "durable_response_type"
	durableResponseSchemaKey  = "durable_response_schema"
	durableOrchestrationIDKey = "durable_orchestration_id"
	durableRequestCreatedAt   = "durable_request_created_at"
	durableWaitForResponseKey = "durable_wait_for_response"
	durableEnableToolCallsKey = "durable_enable_tool_calls"
	durableRequestOptionsKey  = "durable_request_options"
)

// WithDurableSessionID creates a RunOption that sets the session ID for this run.
// If not set, a new session ID will be generated.
func WithDurableSessionID(sessionID SessionID) agent.RunOption {
	return func(cfg *agent.RunConfig) {
		if cfg.Metadata == nil {
			cfg.Metadata = make(map[string]interface{})
		}
		cfg.Metadata["durable_session_id"] = sessionID
	}
}

// WithDurableCorrelationID sets the correlation ID for a durable run.
func WithDurableCorrelationID(correlationID string) agent.RunOption {
	return func(cfg *agent.RunConfig) {
		if cfg.Metadata == nil {
			cfg.Metadata = make(map[string]interface{})
		}
		cfg.Metadata[durableCorrelationIDKey] = correlationID
	}
}

// WithDurableResponseType sets the expected response type for a durable run.
func WithDurableResponseType(responseType string) agent.RunOption {
	return func(cfg *agent.RunConfig) {
		if cfg.Metadata == nil {
			cfg.Metadata = make(map[string]interface{})
		}
		cfg.Metadata[durableResponseTypeKey] = responseType
	}
}

// WithDurableResponseSchema sets the expected response schema for a durable run.
func WithDurableResponseSchema(schema json.RawMessage) agent.RunOption {
	return func(cfg *agent.RunConfig) {
		if cfg.Metadata == nil {
			cfg.Metadata = make(map[string]interface{})
		}
		cfg.Metadata[durableResponseSchemaKey] = schema
	}
}

// WithDurableOrchestrationID sets the orchestration ID for a durable run.
func WithDurableOrchestrationID(orchestrationID string) agent.RunOption {
	return func(cfg *agent.RunConfig) {
		if cfg.Metadata == nil {
			cfg.Metadata = make(map[string]interface{})
		}
		cfg.Metadata[durableOrchestrationIDKey] = orchestrationID
	}
}

// WithDurableRequestCreatedAt sets the created-at time for a durable run request.
func WithDurableRequestCreatedAt(createdAt time.Time) agent.RunOption {
	return func(cfg *agent.RunConfig) {
		if cfg.Metadata == nil {
			cfg.Metadata = make(map[string]interface{})
		}
		cfg.Metadata[durableRequestCreatedAt] = createdAt
	}
}

// WithDurableWaitForResponse controls whether the caller waits for the durable response.
func WithDurableWaitForResponse(wait bool) agent.RunOption {
	return func(cfg *agent.RunConfig) {
		if cfg.Metadata == nil {
			cfg.Metadata = make(map[string]interface{})
		}
		cfg.Metadata[durableWaitForResponseKey] = wait
	}
}

// WithDurableEnableToolCalls controls whether tool calls are enabled for a durable run.
func WithDurableEnableToolCalls(enable bool) agent.RunOption {
	return func(cfg *agent.RunConfig) {
		if cfg.Metadata == nil {
			cfg.Metadata = make(map[string]interface{})
		}
		cfg.Metadata[durableEnableToolCallsKey] = enable
	}
}

// WithDurableRequestOptions sets additional request options for a durable run.
func WithDurableRequestOptions(options map[string]interface{}) agent.RunOption {
	return func(cfg *agent.RunConfig) {
		if cfg.Metadata == nil {
			cfg.Metadata = make(map[string]interface{})
		}
		cfg.Metadata[durableRequestOptionsKey] = options
	}
}

// getDurableSessionID extracts the session ID from a RunConfig if set.
func getDurableSessionID(cfg *agent.RunConfig, agentName string) (SessionID, bool) {
	if cfg == nil || cfg.Metadata == nil {
		return SessionID{}, false
	}
	if id, ok := cfg.Metadata["durable_session_id"].(SessionID); ok {
		return id, true
	}
	if id, ok := cfg.Metadata["durable_session_id"].(string); ok {
		parsed, err := ParseSessionID(id)
		if err == nil {
			return parsed, true
		}
		if agentName != "" {
			return SessionID{Name: agentName, Key: id}, true
		}
	}
	return SessionID{}, false
}

type durableRunRequestOptions struct {
	CorrelationID  string
	ResponseType   string
	ResponseSchema json.RawMessage
	Orchestration  string
	CreatedAt      *time.Time
	WaitForReply   bool
	EnableTools    bool
	Options        map[string]interface{}
}

func getDurableRunRequestOptions(cfg *agent.RunConfig) durableRunRequestOptions {
	options := durableRunRequestOptions{
		WaitForReply: true,
		EnableTools:  true,
	}
	if cfg == nil || cfg.Metadata == nil {
		return options
	}

	if correlationID, ok := cfg.Metadata[durableCorrelationIDKey].(string); ok {
		options.CorrelationID = correlationID
	}
	if responseType, ok := cfg.Metadata[durableResponseTypeKey].(string); ok {
		options.ResponseType = responseType
	}
	if responseSchema, ok := cfg.Metadata[durableResponseSchemaKey].(json.RawMessage); ok {
		options.ResponseSchema = responseSchema
	}
	if orchestrationID, ok := cfg.Metadata[durableOrchestrationIDKey].(string); ok {
		options.Orchestration = orchestrationID
	}
	if createdAt, ok := cfg.Metadata[durableRequestCreatedAt].(time.Time); ok {
		options.CreatedAt = &createdAt
	}
	if wait, ok := cfg.Metadata[durableWaitForResponseKey].(bool); ok {
		options.WaitForReply = wait
	}
	if enableTools, ok := cfg.Metadata[durableEnableToolCallsKey].(bool); ok {
		options.EnableTools = enableTools
	}
	if requestOptions, ok := cfg.Metadata[durableRequestOptionsKey].(map[string]interface{}); ok {
		options.Options = requestOptions
	}

	return options
}

// Options contains configuration for durable agents.
type Options struct {
	// TaskQueue is the Temporal task queue name.
	TaskQueue string

	// TimeToLive is the TTL for sessions.
	TimeToLive time.Duration

	// ActivityTimeout is the timeout for agent activities.
	ActivityTimeout time.Duration

	// MaxRetryAttempts is the maximum retry attempts for activities.
	MaxRetryAttempts int
}

// DefaultOptions returns sensible default options.
func DefaultOptions() Options {
	return Options{
		TaskQueue:        "durable-agent-queue",
		TimeToLive:       0, // No TTL by default
		ActivityTimeout:  DefaultActivityTimeout,
		MaxRetryAttempts: DefaultMaxRetryAttempts,
	}
}

// ClientOptions contains options for creating a Temporal client.
type ClientOptions struct {
	// HostPort is the Temporal server address.
	HostPort string

	// Namespace is the Temporal namespace.
	Namespace string

	// EnableTLS enables TLS for the connection.
	EnableTLS bool
}

// DefaultClientOptions returns default client options for local development.
func DefaultClientOptions() ClientOptions {
	return ClientOptions{
		HostPort:  "localhost:7233",
		Namespace: "default",
		EnableTLS: false,
	}
}
