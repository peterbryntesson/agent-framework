// Copyright (c) Microsoft. All rights reserved.

// Package agent provides core abstractions for building AI agents.
package agent

import (
	"context"
	"encoding/json"
	"reflect"
)

// Agent defines the core interface for AI agent implementations.
// Implementations provide conversational AI capabilities with support for
// streaming responses, session management, and extensible service resolution.
type Agent interface {
	// ID returns the unique identifier for this agent.
	// Implementations should return a stable identifier, typically a UUID.
	ID() string

	// Name returns the human-readable name of the agent.
	// Returns empty string if no name is configured.
	Name() string

	// Description returns a description of the agent's purpose and capabilities.
	// Returns empty string if no description is configured.
	Description() string

	// Metadata returns provider-specific metadata about the agent.
	Metadata() AIAgentMetadata

	// Run executes the agent with the provided messages and returns a complete response.
	// The messages parameter contains the conversation history and new user input.
	// Options can modify the agent's behavior for this specific run.
	// Returns an error if the agent fails to process the request.
	Run(ctx context.Context, messages []Message, opts ...RunOption) (*Response, error)

	// RunStream executes the agent and returns a channel of incremental response updates.
	// The channel is closed when the response is complete or an error occurs.
	// Callers should drain the channel completely to avoid resource leaks.
	// Context cancellation stops the stream and closes the channel.
	RunStream(ctx context.Context, messages []Message, opts ...RunOption) (<-chan ResponseUpdate, error)

	// NewSession creates a new agent session for maintaining conversation state.
	// Sessions track conversation history and can be serialized for persistence.
	NewSession(ctx context.Context) (Session, error)

	// RestoreSession deserializes a previously saved session from JSON data.
	// Use this to resume conversations from persisted state.
	RestoreSession(ctx context.Context, data json.RawMessage) (Session, error)

	// GetService retrieves a service of the specified type from the agent.
	// Returns nil if the service type is not available.
	// This enables extensibility through a service locator pattern.
	GetService(serviceType reflect.Type) interface{}
}

// GetService is a generic helper to retrieve a typed service from an agent.
// Returns the zero value and false if the service is not available.
func GetService[T any](agent Agent) (T, bool) {
	var zero T
	serviceType := reflect.TypeOf((*T)(nil)).Elem()
	service := agent.GetService(serviceType)
	if service == nil {
		return zero, false
	}
	typed, ok := service.(T)
	return typed, ok
}
