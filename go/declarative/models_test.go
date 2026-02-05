// Copyright (c) Microsoft. All rights reserved.

package declarative

import (
	"testing"
)

func TestPromptAgentModelFields(t *testing.T) {
	agent := PromptAgent{
		Kind:         "Prompt",
		Name:         "TestAgent",
		Instructions: "Be helpful",
		Model: Model{
			ID:       "gpt-4o",
			Provider: "OpenAI",
			Connection: &Connection{
				Key: "sk-test",
			},
		},
	}

	if agent.Kind != "Prompt" {
		t.Errorf("Kind = %q, want %q", agent.Kind, "Prompt")
	}
	if agent.Name != "TestAgent" {
		t.Errorf("Name = %q, want %q", agent.Name, "TestAgent")
	}
	if agent.Model.ID != "gpt-4o" {
		t.Errorf("Model.ID = %q, want %q", agent.Model.ID, "gpt-4o")
	}
}

func TestConnectionGetConnectionKind(t *testing.T) {
	tests := []struct {
		name       string
		connection *Connection
		want       string
	}{
		{
			name:       "nil connection",
			connection: nil,
			want:       "",
		},
		{
			name:       "kind field",
			connection: &Connection{Kind: "key"},
			want:       "key",
		},
		{
			name:       "type field fallback",
			connection: &Connection{Type: "remote"},
			want:       "remote",
		},
		{
			name:       "kind takes precedence",
			connection: &Connection{Kind: "key", Type: "remote"},
			want:       "key",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.connection.GetConnectionKind()
			if got != tt.want {
				t.Errorf("GetConnectionKind() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestConnectionGetKey(t *testing.T) {
	tests := []struct {
		name       string
		connection *Connection
		want       string
	}{
		{
			name:       "nil connection",
			connection: nil,
			want:       "",
		},
		{
			name:       "key field",
			connection: &Connection{Key: "sk-test"},
			want:       "sk-test",
		},
		{
			name:       "source field fallback",
			connection: &Connection{Source: "=Env.API_KEY"},
			want:       "=Env.API_KEY",
		},
		{
			name:       "key takes precedence",
			connection: &Connection{Key: "sk-key", Source: "sk-source"},
			want:       "sk-key",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.connection.GetKey()
			if got != tt.want {
				t.Errorf("GetKey() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestPropertySchemaGetPropertyType(t *testing.T) {
	tests := []struct {
		name string
		prop PropertySchema
		want string
	}{
		{
			name: "kind field",
			prop: PropertySchema{Kind: "string"},
			want: "string",
		},
		{
			name: "type field fallback",
			prop: PropertySchema{Type: "number"},
			want: "number",
		},
		{
			name: "kind takes precedence",
			prop: PropertySchema{Kind: "boolean", Type: "string"},
			want: "boolean",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.prop.GetPropertyType()
			if got != tt.want {
				t.Errorf("GetPropertyType() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestModelOptions(t *testing.T) {
	temp := 0.7
	topP := 0.9
	maxTokens := 1000

	opts := &ModelOptions{
		Temperature:            &temp,
		TopP:                   &topP,
		MaxTokens:              &maxTokens,
		AllowMultipleToolCalls: true,
		ChatToolMode:           "auto",
	}

	if *opts.Temperature != 0.7 {
		t.Errorf("Temperature = %v, want %v", *opts.Temperature, 0.7)
	}
	if *opts.TopP != 0.9 {
		t.Errorf("TopP = %v, want %v", *opts.TopP, 0.9)
	}
	if *opts.MaxTokens != 1000 {
		t.Errorf("MaxTokens = %v, want %v", *opts.MaxTokens, 1000)
	}
	if !opts.AllowMultipleToolCalls {
		t.Error("AllowMultipleToolCalls = false, want true")
	}
	if opts.ChatToolMode != "auto" {
		t.Errorf("ChatToolMode = %q, want %q", opts.ChatToolMode, "auto")
	}
}
