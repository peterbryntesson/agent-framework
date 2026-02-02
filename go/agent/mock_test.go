// Copyright (c) Microsoft. All rights reserved.

package agent_test

import (
	"context"
	"encoding/json"
	"reflect"

	"github.com/microsoft/agent-framework-go/agent"
)

// MockAgent is a mock implementation of the Agent interface for testing.
// All methods delegate to configurable function fields, allowing tests to
// customize behavior without creating separate mock types.
type MockAgent struct {
	// IDFunc is called by ID(). Returns empty string if nil.
	IDFunc func() string

	// NameFunc is called by Name(). Returns empty string if nil.
	NameFunc func() string

	// DescriptionFunc is called by Description(). Returns empty string if nil.
	DescriptionFunc func() string

	// MetadataFunc is called by Metadata(). Returns zero value if nil.
	MetadataFunc func() agent.AIAgentMetadata

	// RunFunc is called by Run(). Returns nil, nil if nil.
	RunFunc func(ctx context.Context, messages []agent.Message, opts ...agent.RunOption) (*agent.Response, error)

	// RunStreamFunc is called by RunStream(). Returns nil, nil if nil.
	RunStreamFunc func(ctx context.Context, messages []agent.Message, opts ...agent.RunOption) (<-chan agent.ResponseUpdate, error)

	// NewSessionFunc is called by NewSession(). Returns nil, nil if nil.
	NewSessionFunc func(ctx context.Context) (agent.Session, error)

	// RestoreSessionFunc is called by RestoreSession(). Returns nil, nil if nil.
	RestoreSessionFunc func(ctx context.Context, data json.RawMessage) (agent.Session, error)

	// GetServiceFunc is called by GetService(). Returns nil if nil.
	GetServiceFunc func(serviceType reflect.Type) interface{}
}

// ID returns the agent's unique identifier.
func (m *MockAgent) ID() string {
	if m.IDFunc != nil {
		return m.IDFunc()
	}
	return ""
}

// Name returns the agent's human-readable name.
func (m *MockAgent) Name() string {
	if m.NameFunc != nil {
		return m.NameFunc()
	}
	return ""
}

// Description returns the agent's description.
func (m *MockAgent) Description() string {
	if m.DescriptionFunc != nil {
		return m.DescriptionFunc()
	}
	return ""
}

// Metadata returns provider-specific metadata.
func (m *MockAgent) Metadata() agent.AIAgentMetadata {
	if m.MetadataFunc != nil {
		return m.MetadataFunc()
	}
	return agent.AIAgentMetadata{}
}

// Run executes the agent with the provided messages.
func (m *MockAgent) Run(ctx context.Context, messages []agent.Message, opts ...agent.RunOption) (*agent.Response, error) {
	if m.RunFunc != nil {
		return m.RunFunc(ctx, messages, opts...)
	}
	return nil, nil
}

// RunStream executes the agent and returns a channel of updates.
func (m *MockAgent) RunStream(ctx context.Context, messages []agent.Message, opts ...agent.RunOption) (<-chan agent.ResponseUpdate, error) {
	if m.RunStreamFunc != nil {
		return m.RunStreamFunc(ctx, messages, opts...)
	}
	return nil, nil
}

// NewSession creates a new agent session.
func (m *MockAgent) NewSession(ctx context.Context) (agent.Session, error) {
	if m.NewSessionFunc != nil {
		return m.NewSessionFunc(ctx)
	}
	return nil, nil
}

// RestoreSession deserializes a session from JSON data.
func (m *MockAgent) RestoreSession(ctx context.Context, data json.RawMessage) (agent.Session, error) {
	if m.RestoreSessionFunc != nil {
		return m.RestoreSessionFunc(ctx, data)
	}
	return nil, nil
}

// GetService retrieves a service of the specified type.
func (m *MockAgent) GetService(serviceType reflect.Type) interface{} {
	if m.GetServiceFunc != nil {
		return m.GetServiceFunc(serviceType)
	}
	return nil
}

// Verify MockAgent implements agent.Agent interface at compile time.
var _ agent.Agent = (*MockAgent)(nil)

// NewMockAgent creates a MockAgent with default no-op behavior.
func NewMockAgent() *MockAgent {
	return &MockAgent{}
}

// WithID configures the MockAgent to return the specified ID.
func (m *MockAgent) WithID(id string) *MockAgent {
	m.IDFunc = func() string { return id }
	return m
}

// WithName configures the MockAgent to return the specified name.
func (m *MockAgent) WithName(name string) *MockAgent {
	m.NameFunc = func() string { return name }
	return m
}

// WithDescription configures the MockAgent to return the specified description.
func (m *MockAgent) WithDescription(description string) *MockAgent {
	m.DescriptionFunc = func() string { return description }
	return m
}

// WithResponse configures the MockAgent to return the specified response from Run().
func (m *MockAgent) WithResponse(resp *agent.Response, err error) *MockAgent {
	m.RunFunc = func(_ context.Context, _ []agent.Message, _ ...agent.RunOption) (*agent.Response, error) {
		return resp, err
	}
	return m
}

// WithStreamUpdates configures the MockAgent to return the specified updates from RunStream().
func (m *MockAgent) WithStreamUpdates(updates []agent.ResponseUpdate, err error) *MockAgent {
	m.RunStreamFunc = func(_ context.Context, _ []agent.Message, _ ...agent.RunOption) (<-chan agent.ResponseUpdate, error) {
		if err != nil {
			return nil, err
		}
		ch := make(chan agent.ResponseUpdate, len(updates))
		for _, u := range updates {
			ch <- u
		}
		close(ch)
		return ch, nil
	}
	return m
}

// WithSession configures the MockAgent to return the specified session from NewSession().
func (m *MockAgent) WithSession(session agent.Session, err error) *MockAgent {
	m.NewSessionFunc = func(_ context.Context) (agent.Session, error) {
		return session, err
	}
	return m
}
