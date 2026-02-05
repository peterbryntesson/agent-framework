// Copyright (c) Microsoft. All rights reserved.

package declarative

import (
	"testing"
)

func TestResolveAPIKey(t *testing.T) {
	tests := []struct {
		name  string
		model Model
		want  string
	}{
		{
			name:  "nil connection",
			model: Model{},
			want:  "",
		},
		{
			name: "key field",
			model: Model{
				Connection: &Connection{Key: "sk-test"},
			},
			want: "sk-test",
		},
		{
			name: "source field",
			model: Model{
				Connection: &Connection{Source: "sk-from-source"},
			},
			want: "sk-from-source",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resolveAPIKey(tt.model)
			if got != tt.want {
				t.Errorf("resolveAPIKey() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestBuildOpenAIProvider(t *testing.T) {
	model := Model{
		ID: "gpt-4o-mini",
		Connection: &Connection{
			Key: "sk-test-key",
		},
	}

	client, err := buildOpenAIProvider(model)
	if err != nil {
		t.Fatalf("buildOpenAIProvider() error = %v", err)
	}

	if client == nil {
		t.Error("buildOpenAIProvider() returned nil client")
	}

	// Verify metadata
	metadata := client.Metadata()
	if metadata.ModelID != "gpt-4o-mini" {
		t.Errorf("ModelID = %q, want %q", metadata.ModelID, "gpt-4o-mini")
	}
}

func TestBuildOpenAIProviderWithEndpoint(t *testing.T) {
	model := Model{
		ID:       "custom-model",
		Endpoint: "https://custom.api.com/v1",
		Connection: &Connection{
			Key: "sk-test-key",
		},
	}

	client, err := buildOpenAIProvider(model)
	if err != nil {
		t.Fatalf("buildOpenAIProvider() error = %v", err)
	}

	if client == nil {
		t.Error("buildOpenAIProvider() returned nil client")
	}
}

func TestBuildAzureOpenAIProviderMissingAPIKey(t *testing.T) {
	model := Model{
		ID:       "deployment",
		Endpoint: "https://example.openai.azure.com",
	}

	_, err := buildAzureOpenAIProvider(model)
	if err == nil {
		t.Error("buildAzureOpenAIProvider() expected error for missing API key")
	}
}

func TestBuildAzureOpenAIProviderMissingEndpoint(t *testing.T) {
	model := Model{
		ID: "deployment",
		Connection: &Connection{
			Key: "sk-test",
		},
	}

	_, err := buildAzureOpenAIProvider(model)
	if err == nil {
		t.Error("buildAzureOpenAIProvider() expected error for missing endpoint")
	}
}

func TestBuildAzureOpenAIProviderMissingModelID(t *testing.T) {
	model := Model{
		Endpoint: "https://example.openai.azure.com",
		Connection: &Connection{
			Key: "sk-test",
		},
	}

	_, err := buildAzureOpenAIProvider(model)
	if err == nil {
		t.Error("buildAzureOpenAIProvider() expected error for missing model ID")
	}
}

func TestBuildAzureOpenAIProviderValid(t *testing.T) {
	model := Model{
		ID:       "gpt-4o-deployment",
		Endpoint: "https://example.openai.azure.com",
		Connection: &Connection{
			Key: "azure-api-key",
		},
	}

	client, err := buildAzureOpenAIProvider(model)
	if err != nil {
		t.Fatalf("buildAzureOpenAIProvider() error = %v", err)
	}

	if client == nil {
		t.Error("buildAzureOpenAIProvider() returned nil client")
	}
}
