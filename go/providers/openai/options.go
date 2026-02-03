// Copyright (c) Microsoft. All rights reserved.

package openai

import (
	"net/http"
	"os"
)

// config holds the configuration for an OpenAI client.
type config struct {
	apiKey          string
	baseURL         string
	orgID           string
	model           string
	instructionRole string
	httpClient      *http.Client
}

// defaultConfig returns the default configuration for an OpenAI client.
func defaultConfig() *config {
	return &config{
		model:           DefaultModel,
		instructionRole: DefaultInstructionRole,
	}
}

// applyEnvDefaults applies environment variable defaults to the configuration.
func (c *config) applyEnvDefaults() {
	if c.apiKey == "" {
		c.apiKey = os.Getenv(EnvAPIKey)
	}
	if c.baseURL == "" {
		if envURL := os.Getenv(EnvBaseURL); envURL != "" {
			c.baseURL = envURL
		}
	}
	if c.orgID == "" {
		c.orgID = os.Getenv(EnvOrgID)
	}
}

// Option configures an OpenAI client.
type Option func(*config)

// WithAPIKey sets the OpenAI API key.
// If not provided, the client falls back to the OPENAI_API_KEY environment variable.
func WithAPIKey(key string) Option {
	return func(c *config) {
		c.apiKey = key
	}
}

// WithModel sets the model to use for chat completions.
// Examples: "gpt-4o", "gpt-4o-mini", "gpt-4-turbo", "gpt-3.5-turbo"
// Default: "gpt-4o"
func WithModel(model string) Option {
	return func(c *config) {
		c.model = model
	}
}

// WithBaseURL sets a custom API endpoint URL.
// Use this for OpenAI-compatible APIs, proxies, or local servers.
// If not provided, the default OpenAI API URL is used.
func WithBaseURL(url string) Option {
	return func(c *config) {
		c.baseURL = url
	}
}

// WithOrgID sets the OpenAI organization ID.
// Required for accounts with multiple organizations.
// If not provided, falls back to the OPENAI_ORG_ID environment variable.
func WithOrgID(orgID string) Option {
	return func(c *config) {
		c.orgID = orgID
	}
}

// WithInstructionRole sets the role used for system instructions.
// Valid values are "system" or "developer".
// The "developer" role is used by some newer models for instruction following.
// Default: "system"
func WithInstructionRole(role string) Option {
	return func(c *config) {
		c.instructionRole = role
	}
}

// WithHTTPClient sets a custom HTTP client for making requests.
// Use this to configure timeouts, transport settings, or proxies.
// If not provided, a default HTTP client is used.
func WithHTTPClient(client *http.Client) Option {
	return func(c *config) {
		c.httpClient = client
	}
}
