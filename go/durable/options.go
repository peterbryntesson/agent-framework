// Copyright (c) Microsoft. All rights reserved.

package durable

import (
	"time"

	"github.com/microsoft/agent-framework-go/agent"
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

// getDurableSessionID extracts the session ID from a RunConfig if set.
func getDurableSessionID(cfg *agent.RunConfig) (SessionID, bool) {
	if cfg == nil || cfg.Metadata == nil {
		return SessionID{}, false
	}
	if id, ok := cfg.Metadata["durable_session_id"].(SessionID); ok {
		return id, true
	}
	return SessionID{}, false
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
