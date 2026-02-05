// Copyright (c) Microsoft. All rights reserved.

package tool

// Default values for invocation configuration.
// These align with Python agent_framework defaults.
const (
	// DefaultMaxIterations is the default maximum number of tool invocation rounds.
	// Matches Python DEFAULT_MAX_ITERATIONS.
	DefaultMaxIterations = 40

	// DefaultMaxConsecutiveErrors is the default limit for consecutive tool errors.
	// Matches Python DEFAULT_MAX_CONSECUTIVE_ERRORS.
	DefaultMaxConsecutiveErrors = 3
)

// InvocationConfig controls how tools are invoked during agent runs.
// This configuration aligns with Python FunctionInvocationConfiguration.
type InvocationConfig struct {
	// Enabled controls whether automatic function invocation is active.
	// When false, tool calls are returned to the caller without execution.
	Enabled bool

	// MaxIterations limits the number of tool invocation rounds.
	// Each round may include multiple parallel tool calls.
	// Zero or negative values disable the limit.
	MaxIterations int

	// MaxConsecutiveErrors stops invocation after this many consecutive errors.
	// This prevents infinite loops when tools consistently fail.
	// Zero or negative values disable this safeguard.
	MaxConsecutiveErrors int

	// TerminateOnUnknownCalls stops if a tool call references an unknown tool.
	// When false, unknown tool calls return an error result to the model.
	TerminateOnUnknownCalls bool

	// IncludeDetailedErrors includes full error details in tool results.
	// When false, generic error messages are returned to protect sensitive information.
	IncludeDetailedErrors bool

	// AdditionalTools are extra tools available for this invocation only.
	// These are merged with the agent's base tools for a single run.
	AdditionalTools []Tool

	// ReturnIntermediateSteps includes tool calls and results in the response.
	// When true, the response contains the full conversation including tool interactions.
	ReturnIntermediateSteps bool

	// Timeout specifies the maximum duration for a single tool invocation.
	// Zero means no timeout (use context deadline instead).
	TimeoutSeconds int

	// ParallelToolCalls enables parallel execution of independent tool calls.
	// When false, tool calls are executed sequentially.
	ParallelToolCalls bool
}

// DefaultInvocationConfig returns sensible defaults for tool invocation.
// These defaults provide a safe and practical starting point.
func DefaultInvocationConfig() InvocationConfig {
	return InvocationConfig{
		Enabled:                 true,
		MaxIterations:           DefaultMaxIterations,
		MaxConsecutiveErrors:    DefaultMaxConsecutiveErrors,
		TerminateOnUnknownCalls: false,
		IncludeDetailedErrors:   false,
		AdditionalTools:         nil,
		ReturnIntermediateSteps: false,
		TimeoutSeconds:          0,
		ParallelToolCalls:       true,
	}
}

// DisabledInvocationConfig returns a configuration with invocation disabled.
// Use this when tool calls should be returned without execution.
func DisabledInvocationConfig() InvocationConfig {
	config := DefaultInvocationConfig()
	config.Enabled = false
	return config
}

// WithMaxIterations returns a copy with the specified max iterations.
func (c InvocationConfig) WithMaxIterations(max int) InvocationConfig {
	c.MaxIterations = max
	return c
}

// WithMaxConsecutiveErrors returns a copy with the specified error limit.
func (c InvocationConfig) WithMaxConsecutiveErrors(max int) InvocationConfig {
	c.MaxConsecutiveErrors = max
	return c
}

// WithAdditionalTools returns a copy with additional tools appended.
func (c InvocationConfig) WithAdditionalTools(tools ...Tool) InvocationConfig {
	c.AdditionalTools = append(c.AdditionalTools, tools...)
	return c
}

// WithDetailedErrors returns a copy with detailed errors enabled.
func (c InvocationConfig) WithDetailedErrors(enabled bool) InvocationConfig {
	c.IncludeDetailedErrors = enabled
	return c
}

// WithIntermediateSteps returns a copy with intermediate step tracking enabled.
func (c InvocationConfig) WithIntermediateSteps(enabled bool) InvocationConfig {
	c.ReturnIntermediateSteps = enabled
	return c
}

// WithTimeout returns a copy with the specified timeout in seconds.
func (c InvocationConfig) WithTimeout(seconds int) InvocationConfig {
	c.TimeoutSeconds = seconds
	return c
}

// WithParallelCalls returns a copy with parallel tool call execution setting.
func (c InvocationConfig) WithParallelCalls(enabled bool) InvocationConfig {
	c.ParallelToolCalls = enabled
	return c
}

// Merge combines this configuration with another, preferring non-zero values from other.
// This is useful for layering configuration from multiple sources.
func (c InvocationConfig) Merge(other InvocationConfig) InvocationConfig {
	result := c

	// Only override if explicitly set in other
	if !other.Enabled {
		result.Enabled = false
	}
	if other.MaxIterations > 0 {
		result.MaxIterations = other.MaxIterations
	}
	if other.MaxConsecutiveErrors > 0 {
		result.MaxConsecutiveErrors = other.MaxConsecutiveErrors
	}
	if other.TerminateOnUnknownCalls {
		result.TerminateOnUnknownCalls = true
	}
	if other.IncludeDetailedErrors {
		result.IncludeDetailedErrors = true
	}
	if len(other.AdditionalTools) > 0 {
		result.AdditionalTools = append(result.AdditionalTools, other.AdditionalTools...)
	}
	if other.ReturnIntermediateSteps {
		result.ReturnIntermediateSteps = true
	}
	if other.TimeoutSeconds > 0 {
		result.TimeoutSeconds = other.TimeoutSeconds
	}
	if !other.ParallelToolCalls {
		result.ParallelToolCalls = false
	}

	return result
}

// Validate checks the configuration for invalid values.
// Returns nil if the configuration is valid.
func (c InvocationConfig) Validate() error {
	// Currently no validation errors are possible,
	// but this method provides a hook for future validation.
	return nil
}
