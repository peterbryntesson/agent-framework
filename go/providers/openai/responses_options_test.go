// Copyright (c) Microsoft. All rights reserved.

package openai

import (
	"net/http"
	"testing"
)

func TestDefaultResponsesConfig(t *testing.T) {
	cfg := defaultResponsesConfig()

	if cfg.model != ResponsesDefaultModel {
		t.Errorf("model = %q, want %q", cfg.model, ResponsesDefaultModel)
	}
	if cfg.instructionRole != DefaultInstructionRole {
		t.Errorf("instructionRole = %q, want %q", cfg.instructionRole, DefaultInstructionRole)
	}
	if cfg.baseURL != DefaultBaseURL {
		t.Errorf("baseURL = %q, want %q", cfg.baseURL, DefaultBaseURL)
	}
	if cfg.apiKey != "" {
		t.Error("apiKey should be empty by default")
	}
}

func TestResponsesConfig_ApplyEnvDefaults(t *testing.T) {
	t.Run("applies API key from env", func(t *testing.T) {
		t.Setenv("OPENAI_API_KEY", "env-api-key")
		t.Setenv("OPENAI_BASE_URL", "")
		t.Setenv("OPENAI_ORG_ID", "")

		cfg := defaultResponsesConfig()
		cfg.applyEnvDefaults()

		if cfg.apiKey != "env-api-key" {
			t.Errorf("apiKey = %q, want 'env-api-key'", cfg.apiKey)
		}
	})

	t.Run("applies base URL from env when default", func(t *testing.T) {
		t.Setenv("OPENAI_API_KEY", "key")
		t.Setenv("OPENAI_BASE_URL", "https://custom.api.com")
		t.Setenv("OPENAI_ORG_ID", "")

		cfg := defaultResponsesConfig()
		cfg.applyEnvDefaults()

		if cfg.baseURL != "https://custom.api.com" {
			t.Errorf("baseURL = %q, want 'https://custom.api.com'", cfg.baseURL)
		}
	})

	t.Run("applies base URL from env when empty", func(t *testing.T) {
		t.Setenv("OPENAI_API_KEY", "key")
		t.Setenv("OPENAI_BASE_URL", "https://other.api.com")
		t.Setenv("OPENAI_ORG_ID", "")

		cfg := defaultResponsesConfig()
		cfg.baseURL = "" // Force empty
		cfg.applyEnvDefaults()

		if cfg.baseURL != "https://other.api.com" {
			t.Errorf("baseURL = %q, want 'https://other.api.com'", cfg.baseURL)
		}
	})

	t.Run("applies org ID from env", func(t *testing.T) {
		t.Setenv("OPENAI_API_KEY", "key")
		t.Setenv("OPENAI_BASE_URL", "")
		t.Setenv("OPENAI_ORG_ID", "org-12345")

		cfg := defaultResponsesConfig()
		cfg.applyEnvDefaults()

		if cfg.orgID != "org-12345" {
			t.Errorf("orgID = %q, want 'org-12345'", cfg.orgID)
		}
	})

	t.Run("does not override explicit API key", func(t *testing.T) {
		t.Setenv("OPENAI_API_KEY", "env-key")

		cfg := defaultResponsesConfig()
		cfg.apiKey = "explicit-key"
		cfg.applyEnvDefaults()

		if cfg.apiKey != "explicit-key" {
			t.Errorf("apiKey = %q, want 'explicit-key' (should not be overridden)", cfg.apiKey)
		}
	})

	t.Run("does not override explicit base URL", func(t *testing.T) {
		t.Setenv("OPENAI_BASE_URL", "env-url")

		cfg := defaultResponsesConfig()
		cfg.baseURL = "https://explicit.api.com"
		cfg.applyEnvDefaults()

		if cfg.baseURL != "https://explicit.api.com" {
			t.Errorf("baseURL = %q, want 'https://explicit.api.com' (should not be overridden)", cfg.baseURL)
		}
	})
}

func TestResponsesWithAPIKey(t *testing.T) {
	cfg := defaultResponsesConfig()
	ResponsesWithAPIKey("test-key")(cfg)

	if cfg.apiKey != "test-key" {
		t.Errorf("apiKey = %q, want 'test-key'", cfg.apiKey)
	}
}

func TestResponsesWithModel(t *testing.T) {
	cfg := defaultResponsesConfig()
	ResponsesWithModel("gpt-4o-mini")(cfg)

	if cfg.model != "gpt-4o-mini" {
		t.Errorf("model = %q, want 'gpt-4o-mini'", cfg.model)
	}
}

func TestResponsesWithBaseURL(t *testing.T) {
	cfg := defaultResponsesConfig()
	ResponsesWithBaseURL("https://custom.api.com")(cfg)

	if cfg.baseURL != "https://custom.api.com" {
		t.Errorf("baseURL = %q, want 'https://custom.api.com'", cfg.baseURL)
	}
}

func TestResponsesWithOrgID(t *testing.T) {
	cfg := defaultResponsesConfig()
	ResponsesWithOrgID("org-12345")(cfg)

	if cfg.orgID != "org-12345" {
		t.Errorf("orgID = %q, want 'org-12345'", cfg.orgID)
	}
}

func TestResponsesWithInstructionRole(t *testing.T) {
	cfg := defaultResponsesConfig()
	ResponsesWithInstructionRole("developer")(cfg)

	if cfg.instructionRole != "developer" {
		t.Errorf("instructionRole = %q, want 'developer'", cfg.instructionRole)
	}
}

func TestResponsesWithHTTPClient(t *testing.T) {
	httpClient := &http.Client{}
	cfg := defaultResponsesConfig()
	ResponsesWithHTTPClient(httpClient)(cfg)

	if cfg.httpClient != httpClient {
		t.Error("httpClient should be set to provided client")
	}
}

func TestResponsesWithInstructions(t *testing.T) {
	cfg := defaultResponsesConfig()
	ResponsesWithInstructions("You are a helpful assistant.")(cfg)

	if cfg.instructions != "You are a helpful assistant." {
		t.Errorf("instructions = %q, want 'You are a helpful assistant.'", cfg.instructions)
	}
}

func TestResponsesOptions_ChainedApplication(t *testing.T) {
	cfg := defaultResponsesConfig()

	options := []ResponsesOption{
		ResponsesWithAPIKey("test-key"),
		ResponsesWithModel("gpt-4o-mini"),
		ResponsesWithBaseURL("https://custom.api.com"),
		ResponsesWithOrgID("org-123"),
		ResponsesWithInstructionRole("developer"),
		ResponsesWithInstructions("Be helpful."),
	}

	for _, opt := range options {
		opt(cfg)
	}

	if cfg.apiKey != "test-key" {
		t.Errorf("apiKey = %q, want 'test-key'", cfg.apiKey)
	}
	if cfg.model != "gpt-4o-mini" {
		t.Errorf("model = %q, want 'gpt-4o-mini'", cfg.model)
	}
	if cfg.baseURL != "https://custom.api.com" {
		t.Errorf("baseURL = %q, want 'https://custom.api.com'", cfg.baseURL)
	}
	if cfg.orgID != "org-123" {
		t.Errorf("orgID = %q, want 'org-123'", cfg.orgID)
	}
	if cfg.instructionRole != "developer" {
		t.Errorf("instructionRole = %q, want 'developer'", cfg.instructionRole)
	}
	if cfg.instructions != "Be helpful." {
		t.Errorf("instructions = %q, want 'Be helpful.'", cfg.instructions)
	}
}
