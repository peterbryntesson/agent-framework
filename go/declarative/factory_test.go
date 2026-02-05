// Copyright (c) Microsoft. All rights reserved.

package declarative

import (
	"testing"

	"github.com/microsoft/agent-framework-go/chat"
	"github.com/microsoft/agent-framework-go/tool"
)

func TestNewAgentFactory(t *testing.T) {
	factory := NewAgentFactory()

	if factory == nil {
		t.Fatal("NewAgentFactory() returned nil")
	}

	if factory.bindings == nil {
		t.Error("factory.bindings is nil")
	}
	if factory.providerBuilders == nil {
		t.Error("factory.providerBuilders is nil")
	}
	if factory.toolParsers == nil {
		t.Error("factory.toolParsers is nil")
	}

	// Check default providers are registered
	if _, ok := factory.providerBuilders["openai"]; !ok {
		t.Error("openai provider not registered")
	}
	if _, ok := factory.providerBuilders["azure_openai"]; !ok {
		t.Error("azure_openai provider not registered")
	}

	// Check default tool parsers are registered
	if _, ok := factory.toolParsers["function"]; !ok {
		t.Error("function tool parser not registered")
	}
	if _, ok := factory.toolParsers[""]; !ok {
		t.Error("default tool parser not registered")
	}
}

func TestAgentFactoryWithBinding(t *testing.T) {
	factory := NewAgentFactory()

	myValue := "test-value"
	result := factory.WithBinding("myKey", myValue)

	if result != factory {
		t.Error("WithBinding should return the same factory")
	}

	if factory.bindings["myKey"] != myValue {
		t.Errorf("binding not set correctly, got %v", factory.bindings["myKey"])
	}
}

func TestAgentFactoryWithProvider(t *testing.T) {
	factory := NewAgentFactory()

	customBuilder := func(model Model) (chat.Client, error) {
		return nil, nil
	}

	result := factory.WithProvider("custom", customBuilder)

	if result != factory {
		t.Error("WithProvider should return the same factory")
	}

	if _, ok := factory.providerBuilders["custom"]; !ok {
		t.Error("custom provider not registered")
	}
}

func TestAgentFactoryWithToolParser(t *testing.T) {
	factory := NewAgentFactory()

	customParser := func(t Tool) (tool.Tool, error) {
		return nil, nil
	}

	result := factory.WithToolParser("custom", customParser)

	if result != factory {
		t.Error("WithToolParser should return the same factory")
	}

	if _, ok := factory.toolParsers["custom"]; !ok {
		t.Error("custom tool parser not registered")
	}
}

func TestGetProviderKind(t *testing.T) {
	tests := []struct {
		name  string
		model Model
		want  string
	}{
		{
			name:  "provider OpenAI",
			model: Model{Provider: "OpenAI"},
			want:  "openai",
		},
		{
			name:  "provider AzureOpenAI",
			model: Model{Provider: "AzureOpenAI"},
			want:  "azure_openai",
		},
		{
			name:  "custom provider",
			model: Model{Provider: "custom"},
			want:  "custom",
		},
		{
			name: "connection kind",
			model: Model{
				Connection: &Connection{Kind: "remote"},
			},
			want: "remote",
		},
		{
			name:  "default to openai",
			model: Model{},
			want:  "openai",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getProviderKind(tt.model)
			if got != tt.want {
				t.Errorf("getProviderKind() = %q, want %q", got, tt.want)
			}
		})
	}
}
