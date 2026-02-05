// Copyright (c) Microsoft. All rights reserved.

package agent

// AgentFactory creates a decorated agent from an inner agent.
type AgentFactory func(inner Agent) Agent

// AgentBuilder builds agent pipelines with middleware and decorators.
type AgentBuilder struct {
	innerFactory func() Agent
	factories    []AgentFactory
}

// NewAgentBuilder creates a new agent builder.
// The createAgent function creates the base agent to be decorated.
func NewAgentBuilder(createAgent func() Agent) *AgentBuilder {
	return &AgentBuilder{
		innerFactory: createAgent,
		factories:    make([]AgentFactory, 0),
	}
}

// Use adds a decorator factory to the pipeline.
// Factories are applied in reverse order: first Use() is outermost.
func (b *AgentBuilder) Use(factory AgentFactory) *AgentBuilder {
	b.factories = append(b.factories, factory)
	return b
}

// UseMiddleware adds agent middleware to the pipeline.
// This is a convenience method that wraps the middleware in a MiddlewareAgent.
func (b *AgentBuilder) UseMiddleware(middlewares ...AgentMiddleware) *AgentBuilder {
	return b.Use(func(inner Agent) Agent {
		return NewMiddlewareAgent(inner, middlewares...)
	})
}

// Build creates the configured agent with all decorators applied.
func (b *AgentBuilder) Build() Agent {
	inner := b.innerFactory()

	// Apply in reverse order so first Use() is outermost
	for i := len(b.factories) - 1; i >= 0; i-- {
		inner = b.factories[i](inner)
	}

	return inner
}
