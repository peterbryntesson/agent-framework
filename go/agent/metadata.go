// Copyright (c) Microsoft. All rights reserved.

package agent

// AIAgentMetadata contains provider-specific metadata about an agent.
// This information is useful for observability, telemetry, and diagnostics.
type AIAgentMetadata struct {
	// ProviderName identifies the AI service provider (e.g., "openai", "azure", "anthropic").
	// Used for OpenTelemetry semantic conventions and telemetry attribution.
	ProviderName string
}

// NewAIAgentMetadata creates a new AIAgentMetadata with the specified provider name.
func NewAIAgentMetadata(providerName string) AIAgentMetadata {
	return AIAgentMetadata{
		ProviderName: providerName,
	}
}
