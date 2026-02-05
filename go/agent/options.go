// Copyright (c) Microsoft. All rights reserved.

package agent

// RunOption is a functional option for configuring an agent run.
// Use the With* functions to create options.
// This will be fully implemented in User Story 1.2.4.
type RunOption func(*RunConfig)

// RunConfig holds the configuration for an agent run.
// This type is exported for use by agent implementations.
type RunConfig struct {
	Session        Session
	Metadata       map[string]interface{}
	Tools          []interface{}
	MaxTokens      int
	Temperature    float32
	RuntimeContext *RuntimeContext
}

// ApplyRunOptions applies all options to a default configuration and returns the result.
// This is useful for agent implementations that need to process RunOption values.
func ApplyRunOptions(opts ...RunOption) *RunConfig {
	cfg := &RunConfig{
		MaxTokens:   0, // 0 means use model default
		Temperature: 0, // 0 means use model default
		Metadata:    make(map[string]interface{}),
	}
	for _, opt := range opts {
		if opt != nil {
			opt(cfg)
		}
	}
	return cfg
}

// WithSession sets the session for the agent run.
func WithSession(session Session) RunOption {
	return func(cfg *RunConfig) {
		cfg.Session = session
	}
}

// WithTools adds tools available for this run.
func WithTools(tools ...interface{}) RunOption {
	return func(cfg *RunConfig) {
		cfg.Tools = append(cfg.Tools, tools...)
	}
}

// WithMaxTokens sets the maximum tokens for the response.
func WithMaxTokens(maxTokens int) RunOption {
	return func(cfg *RunConfig) {
		cfg.MaxTokens = maxTokens
	}
}

// WithTemperature sets the temperature for response generation.
// Values typically range from 0.0 (deterministic) to 1.0 (creative).
func WithTemperature(temperature float32) RunOption {
	return func(cfg *RunConfig) {
		cfg.Temperature = temperature
	}
}

// WithMetadata adds metadata to the run configuration.
func WithMetadata(metadata map[string]interface{}) RunOption {
	return func(cfg *RunConfig) {
		for k, v := range metadata {
			cfg.Metadata[k] = v
		}
	}
}

// WithRunConfig applies an existing RunConfig as a run option.
// This is useful for middleware that needs to pass through configuration.
func WithRunConfig(config *RunConfig) RunOption {
	return func(cfg *RunConfig) {
		if config == nil {
			return
		}
		cfg.Session = config.Session
		cfg.MaxTokens = config.MaxTokens
		cfg.Temperature = config.Temperature
		cfg.Tools = append(cfg.Tools, config.Tools...)
		for k, v := range config.Metadata {
			cfg.Metadata[k] = v
		}
		if config.RuntimeContext != nil {
			cfg.RuntimeContext = config.RuntimeContext
		}
	}
}

// WithRuntimeContext attaches a RuntimeContext to the run configuration.
// Use this to propagate context (user IDs, API tokens, session data) to sub-agents.
func WithRuntimeContext(rtc *RuntimeContext) RunOption {
	return func(cfg *RunConfig) {
		cfg.RuntimeContext = rtc
	}
}

// WithRuntimeValue adds a single key-value pair to the RuntimeContext.
// If no RuntimeContext exists, creates a new one.
func WithRuntimeValue(key string, value interface{}) RunOption {
	return func(cfg *RunConfig) {
		if cfg.RuntimeContext == nil {
			cfg.RuntimeContext = NewRuntimeContext()
		}
		cfg.RuntimeContext = cfg.RuntimeContext.With(key, value)
	}
}
