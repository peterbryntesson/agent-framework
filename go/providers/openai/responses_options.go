// Copyright (c) Microsoft. All rights reserved.

package openai

import (
	"net/http"
	"os"
)

// ResponsesDefaultModel is the default model for the Responses API.
const ResponsesDefaultModel = "gpt-4o"

// responsesConfig holds the configuration for a Responses API client.
type responsesConfig struct {
	apiKey          string
	baseURL         string
	orgID           string
	model           string
	instructionRole string
	httpClient      *http.Client
	instructions    string
}

// defaultResponsesConfig returns the default configuration for a Responses API client.
func defaultResponsesConfig() *responsesConfig {
	return &responsesConfig{
		model:           ResponsesDefaultModel,
		instructionRole: DefaultInstructionRole,
		baseURL:         DefaultBaseURL,
	}
}

// applyEnvDefaults applies environment variable defaults to the responses configuration.
func (c *responsesConfig) applyEnvDefaults() {
	if c.apiKey == "" {
		c.apiKey = os.Getenv(EnvAPIKey)
	}
	if c.baseURL == "" || c.baseURL == DefaultBaseURL {
		if envURL := os.Getenv(EnvBaseURL); envURL != "" {
			c.baseURL = envURL
		}
	}
	if c.orgID == "" {
		c.orgID = os.Getenv(EnvOrgID)
	}
}

// ResponsesOption configures a Responses API client.
type ResponsesOption func(*responsesConfig)

// ResponsesWithAPIKey sets the OpenAI API key.
// If not provided, the client falls back to the OPENAI_API_KEY environment variable.
func ResponsesWithAPIKey(key string) ResponsesOption {
	return func(c *responsesConfig) {
		c.apiKey = key
	}
}

// ResponsesWithModel sets the model to use for the Responses API.
// Examples: "gpt-4o", "gpt-4o-mini"
// Default: "gpt-4o"
func ResponsesWithModel(model string) ResponsesOption {
	return func(c *responsesConfig) {
		c.model = model
	}
}

// ResponsesWithBaseURL sets a custom API endpoint URL.
// Use this for OpenAI-compatible APIs, proxies, or local servers.
// If not provided, the default OpenAI API URL is used.
func ResponsesWithBaseURL(url string) ResponsesOption {
	return func(c *responsesConfig) {
		c.baseURL = url
	}
}

// ResponsesWithOrgID sets the OpenAI organization ID.
// Required for accounts with multiple organizations.
// If not provided, falls back to the OPENAI_ORG_ID environment variable.
func ResponsesWithOrgID(orgID string) ResponsesOption {
	return func(c *responsesConfig) {
		c.orgID = orgID
	}
}

// ResponsesWithInstructionRole sets the role used for system instructions.
// Valid values are "system" or "developer".
// The "developer" role is used by some newer models for instruction following.
// Default: "system"
func ResponsesWithInstructionRole(role string) ResponsesOption {
	return func(c *responsesConfig) {
		c.instructionRole = role
	}
}

// ResponsesWithHTTPClient sets a custom HTTP client for making requests.
// Use this to configure timeouts, transport settings, or proxies.
// If not provided, a default HTTP client is used.
func ResponsesWithHTTPClient(client *http.Client) ResponsesOption {
	return func(c *responsesConfig) {
		c.httpClient = client
	}
}

// ResponsesWithInstructions sets default system instructions for the client.
// These instructions are included in every request unless overridden.
func ResponsesWithInstructions(instructions string) ResponsesOption {
	return func(c *responsesConfig) {
		c.instructions = instructions
	}
}
