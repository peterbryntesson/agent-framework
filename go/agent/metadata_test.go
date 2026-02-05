// Copyright (c) Microsoft. All rights reserved.

package agent

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewAIAgentMetadata_SetsProviderName(t *testing.T) {
	// Arrange
	providerName := "openai"

	// Act
	metadata := NewAIAgentMetadata(providerName)

	// Assert
	assert.Equal(t, providerName, metadata.ProviderName)
}

func TestNewAIAgentMetadata_EmptyProviderName(t *testing.T) {
	// Arrange & Act
	metadata := NewAIAgentMetadata("")

	// Assert
	assert.Equal(t, "", metadata.ProviderName)
}

func TestAIAgentMetadata_DifferentProviders(t *testing.T) {
	testCases := []struct {
		name         string
		providerName string
	}{
		{"OpenAI", "openai"},
		{"Azure", "azure"},
		{"Anthropic", "anthropic"},
		{"Ollama", "ollama"},
		{"Bedrock", "bedrock"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Act
			metadata := NewAIAgentMetadata(tc.providerName)

			// Assert
			assert.Equal(t, tc.providerName, metadata.ProviderName)
		})
	}
}
