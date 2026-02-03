// Copyright (c) Microsoft. All rights reserved.

package chatagent

import (
	"github.com/microsoft/agent-framework-go/tool"
)

// config holds the configuration for creating a ChatClientAgent.
// All fields have sensible defaults.
type config struct {
	id               string
	name             string
	description      string
	instructions     string
	tools            []tool.Tool
	maxTurns         int
	invocationConfig tool.InvocationConfig
}

// defaultConfig returns a config with sensible default values.
func defaultConfig() *config {
	return &config{
		maxTurns:         10,
		invocationConfig: tool.DefaultInvocationConfig(),
	}
}

// Option is a functional option for configuring a ChatClientAgent.
type Option func(*config)

// WithID sets the unique identifier for the agent.
// If not set, a UUID will be generated.
func WithID(id string) Option {
	return func(c *config) {
		c.id = id
	}
}

// WithName sets the human-readable name for the agent.
// If not set, defaults to the model ID from the chat client.
func WithName(name string) Option {
	return func(c *config) {
		c.name = name
	}
}

// WithDescription sets the description of the agent's purpose.
func WithDescription(description string) Option {
	return func(c *config) {
		c.description = description
	}
}

// WithInstructions sets the system instructions for the agent.
// These are prepended as a system message to every conversation.
func WithInstructions(instructions string) Option {
	return func(c *config) {
		c.instructions = instructions
	}
}

// WithTools sets the tools available to the agent.
// Tools can be FunctionTools or HostedTools.
func WithTools(tools ...tool.Tool) Option {
	return func(c *config) {
		c.tools = append(c.tools, tools...)
	}
}

// WithMaxTurns sets the maximum number of conversation turns.
// A turn consists of one model response, potentially with tool calls.
// Default is 10 turns.
func WithMaxTurns(maxTurns int) Option {
	return func(c *config) {
		c.maxTurns = maxTurns
	}
}

// WithInvocationConfig sets the tool invocation configuration.
// This controls behavior like max iterations and error handling.
func WithInvocationConfig(cfg tool.InvocationConfig) Option {
	return func(c *config) {
		c.invocationConfig = cfg
	}
}

// WithInvocationEnabled controls whether automatic tool invocation is enabled.
// When disabled, tool calls are returned but not automatically executed.
func WithInvocationEnabled(enabled bool) Option {
	return func(c *config) {
		c.invocationConfig.Enabled = enabled
	}
}

// WithMaxConsecutiveErrors sets the maximum number of consecutive tool errors
// before stopping the invocation loop. Default is 3.
func WithMaxConsecutiveErrors(max int) Option {
	return func(c *config) {
		c.invocationConfig.MaxConsecutiveErrors = max
	}
}

// WithTerminateOnUnknownCalls sets whether to terminate when an unknown
// tool is called. Default is false (returns error result to model).
func WithTerminateOnUnknownCalls(terminate bool) Option {
	return func(c *config) {
		c.invocationConfig.TerminateOnUnknownCalls = terminate
	}
}

// WithIncludeDetailedErrors sets whether to include detailed error information
// in tool results. Default is false for security.
func WithIncludeDetailedErrors(include bool) Option {
	return func(c *config) {
		c.invocationConfig.IncludeDetailedErrors = include
	}
}

// WithToolTimeout sets the timeout in seconds for individual tool invocations.
// Zero means no timeout.
func WithToolTimeout(seconds int) Option {
	return func(c *config) {
		c.invocationConfig.TimeoutSeconds = seconds
	}
}

// WithParallelToolCalls sets whether to execute tool calls in parallel.
// Default is false (sequential execution).
func WithParallelToolCalls(parallel bool) Option {
	return func(c *config) {
		c.invocationConfig.ParallelToolCalls = parallel
	}
}

// WithReturnIntermediateSteps sets whether to include intermediate tool
// call/result messages in the response. Useful for debugging.
func WithReturnIntermediateSteps(include bool) Option {
	return func(c *config) {
		c.invocationConfig.ReturnIntermediateSteps = include
	}
}
