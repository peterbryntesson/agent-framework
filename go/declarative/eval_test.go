// Copyright (c) Microsoft. All rights reserved.

package declarative

import (
	"os"
	"testing"
)

func TestTryEvalEnv(t *testing.T) {
	tests := []struct {
		name     string
		expr     string
		envName  string
		envValue string
		want     string
		wantOk   bool
	}{
		{
			name:     "simple env var",
			expr:     "=Env.OPENAI_API_KEY",
			envName:  "OPENAI_API_KEY",
			envValue: "sk-test123",
			want:     "sk-test123",
			wantOk:   true,
		},
		{
			name:     "env var with underscores",
			expr:     "=Env.AZURE_OPENAI_ENDPOINT",
			envName:  "AZURE_OPENAI_ENDPOINT",
			envValue: "https://example.openai.azure.com",
			want:     "https://example.openai.azure.com",
			wantOk:   true,
		},
		{
			name:   "literal value",
			expr:   "literal-value",
			want:   "literal-value",
			wantOk: false,
		},
		{
			name:   "empty string",
			expr:   "",
			want:   "",
			wantOk: false,
		},
		{
			name:   "invalid format - no equal sign",
			expr:   "Env.SOME_VAR",
			want:   "Env.SOME_VAR",
			wantOk: false,
		},
		{
			name:   "invalid format - wrong prefix",
			expr:   "=ENV.SOME_VAR",
			want:   "=ENV.SOME_VAR",
			wantOk: false,
		},
		{
			name:     "missing env var returns empty",
			expr:     "=Env.NONEXISTENT_VAR_12345",
			envName:  "",
			envValue: "",
			want:     "",
			wantOk:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envName != "" {
				os.Setenv(tt.envName, tt.envValue)
				defer os.Unsetenv(tt.envName)
			}

			got, ok := TryEvalEnv(tt.expr)
			if ok != tt.wantOk {
				t.Errorf("TryEvalEnv() ok = %v, wantOk %v", ok, tt.wantOk)
			}
			if got != tt.want {
				t.Errorf("TryEvalEnv() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestEvalEnvOrDefault(t *testing.T) {
	tests := []struct {
		name     string
		expr     string
		envName  string
		envValue string
		want     string
	}{
		{
			name:     "env var found",
			expr:     "=Env.TEST_KEY",
			envName:  "TEST_KEY",
			envValue: "test-value",
			want:     "test-value",
		},
		{
			name: "literal value returned",
			expr: "my-literal",
			want: "my-literal",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envName != "" {
				os.Setenv(tt.envName, tt.envValue)
				defer os.Unsetenv(tt.envName)
			}

			got := EvalEnvOrDefault(tt.expr)
			if got != tt.want {
				t.Errorf("EvalEnvOrDefault() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestEvaluateAgent(t *testing.T) {
	os.Setenv("TEST_MODEL_ID", "gpt-4o")
	os.Setenv("TEST_ENDPOINT", "https://api.example.com")
	os.Setenv("TEST_API_KEY", "sk-secret")
	defer os.Unsetenv("TEST_MODEL_ID")
	defer os.Unsetenv("TEST_ENDPOINT")
	defer os.Unsetenv("TEST_API_KEY")

	agent := &PromptAgent{
		Name:         "Test Agent",
		Instructions: "Be helpful",
		Model: Model{
			ID:       "=Env.TEST_MODEL_ID",
			Endpoint: "=Env.TEST_ENDPOINT",
			Connection: &Connection{
				Key: "=Env.TEST_API_KEY",
			},
		},
		Tools: []Tool{
			{
				Name:   "test_tool",
				Server: "=Env.TEST_ENDPOINT",
			},
		},
	}

	err := evaluateAgent(agent)
	if err != nil {
		t.Fatalf("evaluateAgent() error = %v", err)
	}

	if agent.Model.ID != "gpt-4o" {
		t.Errorf("Model.ID = %q, want %q", agent.Model.ID, "gpt-4o")
	}
	if agent.Model.Endpoint != "https://api.example.com" {
		t.Errorf("Model.Endpoint = %q, want %q", agent.Model.Endpoint, "https://api.example.com")
	}
	if agent.Model.Connection.Key != "sk-secret" {
		t.Errorf("Model.Connection.Key = %q, want %q", agent.Model.Connection.Key, "sk-secret")
	}
	if agent.Tools[0].Server != "https://api.example.com" {
		t.Errorf("Tools[0].Server = %q, want %q", agent.Tools[0].Server, "https://api.example.com")
	}
}
