// Copyright (c) Microsoft. All rights reserved.

package agent

import (
	"context"
	"encoding/json"
	"reflect"
)

// MiddlewareAgent wraps an agent with middleware that intercepts Run and RunStream calls.
type MiddlewareAgent struct {
	inner      Agent
	middleware AgentMiddleware
}

// Compile-time check that MiddlewareAgent implements Agent.
var _ Agent = (*MiddlewareAgent)(nil)

// NewMiddlewareAgent creates an agent that applies the given middleware.
func NewMiddlewareAgent(inner Agent, middlewares ...AgentMiddleware) *MiddlewareAgent {
	return &MiddlewareAgent{
		inner:      inner,
		middleware: ChainAgentMiddleware(middlewares...),
	}
}

// ID returns the unique identifier from the inner agent.
func (m *MiddlewareAgent) ID() string {
	return m.inner.ID()
}

// Name returns the name from the inner agent.
func (m *MiddlewareAgent) Name() string {
	return m.inner.Name()
}

// Description returns the description from the inner agent.
func (m *MiddlewareAgent) Description() string {
	return m.inner.Description()
}

// Metadata returns the metadata from the inner agent.
func (m *MiddlewareAgent) Metadata() AIAgentMetadata {
	return m.inner.Metadata()
}

// NewSession forwards to the inner agent's NewSession method.
func (m *MiddlewareAgent) NewSession(ctx context.Context) (Session, error) {
	return m.inner.NewSession(ctx)
}

// RestoreSession forwards to the inner agent's RestoreSession method.
func (m *MiddlewareAgent) RestoreSession(ctx context.Context, data json.RawMessage) (Session, error) {
	return m.inner.RestoreSession(ctx, data)
}

// GetService forwards to the inner agent's GetService method.
func (m *MiddlewareAgent) GetService(serviceType reflect.Type) interface{} {
	return m.inner.GetService(serviceType)
}

// Run executes the agent through the middleware chain.
func (m *MiddlewareAgent) Run(ctx context.Context, messages []Message, opts ...RunOption) (*Response, error) {
	agentCtx := &AgentContext{
		Agent:       m.inner,
		Messages:    messages,
		Options:     ApplyRunOptions(opts...),
		Metadata:    make(map[string]any),
		IsStreaming: false,
	}

	// Terminal handler calls the actual agent
	terminal := func(ctx context.Context, agentCtx *AgentContext) error {
		resp, err := agentCtx.Agent.Run(ctx, agentCtx.Messages, WithRunConfig(agentCtx.Options))
		if err != nil {
			return err
		}
		agentCtx.Response = resp
		return nil
	}

	err := m.middleware.Process(ctx, agentCtx, terminal)
	if err != nil {
		return nil, err
	}
	return agentCtx.Response, nil
}

// RunStream executes the agent streaming through the middleware chain.
func (m *MiddlewareAgent) RunStream(ctx context.Context, messages []Message, opts ...RunOption) (<-chan ResponseUpdate, error) {
	agentCtx := &AgentContext{
		Agent:       m.inner,
		Messages:    messages,
		Options:     ApplyRunOptions(opts...),
		Metadata:    make(map[string]any),
		IsStreaming: true,
	}

	// Terminal handler calls the actual streaming agent
	terminal := func(ctx context.Context, agentCtx *AgentContext) error {
		stream, err := agentCtx.Agent.RunStream(ctx, agentCtx.Messages, WithRunConfig(agentCtx.Options))
		if err != nil {
			return err
		}
		agentCtx.Stream = stream
		return nil
	}

	err := m.middleware.Process(ctx, agentCtx, terminal)
	if err != nil {
		return nil, err
	}
	return agentCtx.Stream, nil
}

// Inner returns the wrapped agent.
func (m *MiddlewareAgent) Inner() Agent {
	return m.inner
}
