// Copyright (c) Microsoft. All rights reserved.

package chatagent

import (
	"errors"

	"github.com/microsoft/agent-framework-go/chat"
	"github.com/microsoft/agent-framework-go/tool"
)

// Builder provides a fluent API for configuring a ChatClientAgent.
// Use NewBuilder to create a builder, chain configuration methods,
// and call Build to create the agent.
type Builder struct {
	client chat.Client
	opts   []Option
	err    error
}

// NewBuilder creates a new agent builder with the given chat client.
func NewBuilder(client chat.Client) *Builder {
	return &Builder{
		client: client,
		opts:   make([]Option, 0),
	}
}

// ID sets the unique identifier for the agent.
func (b *Builder) ID(id string) *Builder {
	if b.err != nil {
		return b
	}
	b.opts = append(b.opts, WithID(id))
	return b
}

// Name sets the human-readable name for the agent.
func (b *Builder) Name(name string) *Builder {
	if b.err != nil {
		return b
	}
	b.opts = append(b.opts, WithName(name))
	return b
}

// Description sets the description of the agent's purpose.
func (b *Builder) Description(description string) *Builder {
	if b.err != nil {
		return b
	}
	b.opts = append(b.opts, WithDescription(description))
	return b
}

// Instructions sets the system instructions for the agent.
func (b *Builder) Instructions(instructions string) *Builder {
	if b.err != nil {
		return b
	}
	b.opts = append(b.opts, WithInstructions(instructions))
	return b
}

// Tools sets the tools available to the agent.
func (b *Builder) Tools(tools ...tool.Tool) *Builder {
	if b.err != nil {
		return b
	}
	b.opts = append(b.opts, WithTools(tools...))
	return b
}

// MaxTurns sets the maximum number of conversation turns.
func (b *Builder) MaxTurns(maxTurns int) *Builder {
	if b.err != nil {
		return b
	}
	if maxTurns <= 0 {
		b.err = errors.New("maxTurns must be positive")
		return b
	}
	b.opts = append(b.opts, WithMaxTurns(maxTurns))
	return b
}

// InvocationConfig sets the tool invocation configuration.
func (b *Builder) InvocationConfig(cfg tool.InvocationConfig) *Builder {
	if b.err != nil {
		return b
	}
	b.opts = append(b.opts, WithInvocationConfig(cfg))
	return b
}

// InvocationEnabled controls whether automatic tool invocation is enabled.
func (b *Builder) InvocationEnabled(enabled bool) *Builder {
	if b.err != nil {
		return b
	}
	b.opts = append(b.opts, WithInvocationEnabled(enabled))
	return b
}

// ToolTimeout sets the timeout for individual tool invocations.
func (b *Builder) ToolTimeout(seconds int) *Builder {
	if b.err != nil {
		return b
	}
	b.opts = append(b.opts, WithToolTimeout(seconds))
	return b
}

// ParallelToolCalls enables or disables parallel tool execution.
func (b *Builder) ParallelToolCalls(parallel bool) *Builder {
	if b.err != nil {
		return b
	}
	b.opts = append(b.opts, WithParallelToolCalls(parallel))
	return b
}

// IncludeDetailedErrors controls whether detailed errors are returned.
func (b *Builder) IncludeDetailedErrors(include bool) *Builder {
	if b.err != nil {
		return b
	}
	b.opts = append(b.opts, WithIncludeDetailedErrors(include))
	return b
}

// Build creates the configured agent.
// Returns an error if required configuration is missing or invalid.
func (b *Builder) Build() (*Agent, error) {
	if b.err != nil {
		return nil, b.err
	}
	if b.client == nil {
		return nil, errors.New("chat client is required")
	}
	return New(b.client, b.opts...), nil
}

// MustBuild creates the configured agent, panicking on error.
// Use this only when errors indicate programming mistakes.
func (b *Builder) MustBuild() *Agent {
	agent, err := b.Build()
	if err != nil {
		panic("chatagent: " + err.Error())
	}
	return agent
}
