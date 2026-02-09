// Copyright (c) Microsoft. All rights reserved.

package declarative

import (
	"errors"
	"fmt"

	"github.com/microsoft/agent-framework-go/chat"
	"github.com/microsoft/agent-framework-go/providers/openai"
)

// buildOpenAIProvider creates an OpenAI provider from a model configuration.
func buildOpenAIProvider(model Model) (chat.Client, error) {
	apiKey := resolveAPIKey(model)
	apiType := normalizeAPIType(model.APIType)
	if apiType == "" {
		apiType = "chat"
	}

	if apiType == "responses" {
		opts := make([]openai.ResponsesOption, 0)
		if apiKey != "" {
			opts = append(opts, openai.ResponsesWithAPIKey(apiKey))
		}
		if model.ID != "" {
			opts = append(opts, openai.ResponsesWithModel(model.ID))
		}
		if model.Endpoint != "" {
			opts = append(opts, openai.ResponsesWithBaseURL(model.Endpoint))
		}
		return openai.NewResponsesClient(opts...)
	}
	if apiType != "chat" {
		return nil, fmt.Errorf("unsupported apiType %q for OpenAI provider", model.APIType)
	}

	opts := make([]openai.Option, 0)

	// Add API key if provided
	if apiKey != "" {
		opts = append(opts, openai.WithAPIKey(apiKey))
	}

	// Add model if specified
	if model.ID != "" {
		opts = append(opts, openai.WithModel(model.ID))
	}

	// Add endpoint if specified
	if model.Endpoint != "" {
		opts = append(opts, openai.WithBaseURL(model.Endpoint))
	}

	return openai.NewClient(opts...)
}

// buildAzureOpenAIProvider creates an Azure OpenAI provider from a model configuration.
// Azure OpenAI uses the same underlying client but with Azure-specific configuration.
func buildAzureOpenAIProvider(model Model) (chat.Client, error) {
	apiKey := resolveAPIKey(model)
	if apiKey == "" {
		return nil, errors.New("API key is required for Azure OpenAI provider")
	}

	endpoint := model.Endpoint
	if endpoint == "" {
		return nil, errors.New("endpoint is required for Azure OpenAI provider")
	}

	modelID := model.ID
	if modelID == "" {
		return nil, errors.New("model ID (deployment name) is required for Azure OpenAI provider")
	}

	apiType := normalizeAPIType(model.APIType)
	if apiType == "" {
		apiType = "chat"
	}
	if apiType == "responses" {
		opts := []openai.ResponsesOption{
			openai.ResponsesWithAPIKey(apiKey),
			openai.ResponsesWithModel(modelID),
			openai.ResponsesWithBaseURL(endpoint),
		}
		return openai.NewResponsesClient(opts...)
	}
	if apiType != "chat" {
		return nil, fmt.Errorf("unsupported apiType %q for Azure OpenAI provider", model.APIType)
	}

	// For Azure OpenAI, we use the OpenAI client with Azure-specific settings
	// The endpoint should include the deployment name path
	opts := []openai.Option{
		openai.WithAPIKey(apiKey),
		openai.WithModel(modelID),
		openai.WithBaseURL(endpoint),
	}

	return openai.NewClient(opts...)
}

// resolveAPIKey extracts the API key from the model connection.
// The key may be a direct value or an environment variable reference.
func resolveAPIKey(model Model) string {
	if model.Connection == nil {
		return ""
	}

	// Try key first, then source
	return model.Connection.GetKey()
}
