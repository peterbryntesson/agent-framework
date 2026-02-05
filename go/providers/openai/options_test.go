// Copyright (c) Microsoft. All rights reserved.

package openai

import (
	"net/http"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := defaultConfig()

	if cfg.model != DefaultModel {
		t.Errorf("model = %q, want %q", cfg.model, DefaultModel)
	}
	if cfg.instructionRole != DefaultInstructionRole {
		t.Errorf("instructionRole = %q, want %q", cfg.instructionRole, DefaultInstructionRole)
	}
	if cfg.apiKey != "" {
		t.Error("apiKey should be empty by default")
	}
	if cfg.baseURL != "" {
		t.Error("baseURL should be empty by default")
	}
	if cfg.orgID != "" {
		t.Error("orgID should be empty by default")
	}
	if cfg.httpClient != nil {
		t.Error("httpClient should be nil by default")
	}
}

func TestApplyEnvDefaults(t *testing.T) {
	t.Run("applies API key from env", func(t *testing.T) {
		t.Setenv("OPENAI_API_KEY", "env-api-key")
		t.Setenv("OPENAI_BASE_URL", "")
		t.Setenv("OPENAI_ORG_ID", "")

		cfg := defaultConfig()
		cfg.applyEnvDefaults()

		if cfg.apiKey != "env-api-key" {
			t.Errorf("apiKey = %q, want 'env-api-key'", cfg.apiKey)
		}
	})

	t.Run("applies base URL from env", func(t *testing.T) {
		t.Setenv("OPENAI_API_KEY", "key")
		t.Setenv("OPENAI_BASE_URL", "https://custom.api.com")
		t.Setenv("OPENAI_ORG_ID", "")

		cfg := defaultConfig()
		cfg.applyEnvDefaults()

		if cfg.baseURL != "https://custom.api.com" {
			t.Errorf("baseURL = %q, want 'https://custom.api.com'", cfg.baseURL)
		}
	})

	t.Run("applies org ID from env", func(t *testing.T) {
		t.Setenv("OPENAI_API_KEY", "key")
		t.Setenv("OPENAI_BASE_URL", "")
		t.Setenv("OPENAI_ORG_ID", "org-12345")

		cfg := defaultConfig()
		cfg.applyEnvDefaults()

		if cfg.orgID != "org-12345" {
			t.Errorf("orgID = %q, want 'org-12345'", cfg.orgID)
		}
	})

	t.Run("does not override existing values", func(t *testing.T) {
		t.Setenv("OPENAI_API_KEY", "env-key")
		t.Setenv("OPENAI_BASE_URL", "env-url")
		t.Setenv("OPENAI_ORG_ID", "env-org")

		cfg := defaultConfig()
		cfg.apiKey = "explicit-key"
		cfg.baseURL = "explicit-url"
		cfg.orgID = "explicit-org"
		cfg.applyEnvDefaults()

		if cfg.apiKey != "explicit-key" {
			t.Errorf("apiKey = %q, want 'explicit-key' (should not be overridden)", cfg.apiKey)
		}
		if cfg.baseURL != "explicit-url" {
			t.Errorf("baseURL = %q, want 'explicit-url' (should not be overridden)", cfg.baseURL)
		}
		if cfg.orgID != "explicit-org" {
			t.Errorf("orgID = %q, want 'explicit-org' (should not be overridden)", cfg.orgID)
		}
	})
}

func TestWithAPIKey(t *testing.T) {
	cfg := defaultConfig()
	WithAPIKey("test-key")(cfg)

	if cfg.apiKey != "test-key" {
		t.Errorf("apiKey = %q, want 'test-key'", cfg.apiKey)
	}
}

func TestWithModel(t *testing.T) {
	cfg := defaultConfig()
	WithModel("gpt-4o-mini")(cfg)

	if cfg.model != "gpt-4o-mini" {
		t.Errorf("model = %q, want 'gpt-4o-mini'", cfg.model)
	}
}

func TestWithBaseURL(t *testing.T) {
	cfg := defaultConfig()
	WithBaseURL("https://custom.api.com")(cfg)

	if cfg.baseURL != "https://custom.api.com" {
		t.Errorf("baseURL = %q, want 'https://custom.api.com'", cfg.baseURL)
	}
}

func TestWithOrgID(t *testing.T) {
	cfg := defaultConfig()
	WithOrgID("org-12345")(cfg)

	if cfg.orgID != "org-12345" {
		t.Errorf("orgID = %q, want 'org-12345'", cfg.orgID)
	}
}

func TestWithInstructionRole(t *testing.T) {
	cfg := defaultConfig()
	WithInstructionRole("developer")(cfg)

	if cfg.instructionRole != "developer" {
		t.Errorf("instructionRole = %q, want 'developer'", cfg.instructionRole)
	}
}

func TestWithHTTPClient(t *testing.T) {
	httpClient := &http.Client{}
	cfg := defaultConfig()
	WithHTTPClient(httpClient)(cfg)

	if cfg.httpClient != httpClient {
		t.Error("httpClient should be set to provided client")
	}
}

func TestOptions_ChainedApplication(t *testing.T) {
	cfg := defaultConfig()

	options := []Option{
		WithAPIKey("test-key"),
		WithModel("gpt-4o-mini"),
		WithBaseURL("https://custom.api.com"),
		WithOrgID("org-123"),
		WithInstructionRole("developer"),
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
}
