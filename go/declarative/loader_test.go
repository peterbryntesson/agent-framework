// Copyright (c) Microsoft. All rights reserved.

package declarative

import (
	"strings"
	"testing"
)

func TestLoadFromString(t *testing.T) {
	yaml := `
kind: Prompt
name: TestAgent
instructions: Be helpful
model:
  id: gpt-4o
  provider: OpenAI
  connection:
    key: sk-test
`

	agent, err := LoadFromString(yaml)
	if err != nil {
		t.Fatalf("LoadFromString() error = %v", err)
	}

	if agent.Kind != "Prompt" {
		t.Errorf("Kind = %q, want %q", agent.Kind, "Prompt")
	}
	if agent.Name != "TestAgent" {
		t.Errorf("Name = %q, want %q", agent.Name, "TestAgent")
	}
	if agent.Instructions != "Be helpful" {
		t.Errorf("Instructions = %q, want %q", agent.Instructions, "Be helpful")
	}
	if agent.Model.ID != "gpt-4o" {
		t.Errorf("Model.ID = %q, want %q", agent.Model.ID, "gpt-4o")
	}
	if agent.Model.Provider != "OpenAI" {
		t.Errorf("Model.Provider = %q, want %q", agent.Model.Provider, "OpenAI")
	}
	if agent.Model.Connection == nil {
		t.Fatal("Model.Connection is nil")
	}
	if agent.Model.Connection.Key != "sk-test" {
		t.Errorf("Model.Connection.Key = %q, want %q", agent.Model.Connection.Key, "sk-test")
	}
}

func TestLoadFromReader(t *testing.T) {
	yaml := `
kind: Prompt
name: ReaderAgent
instructions: Test instructions
model:
  id: gpt-4
`

	reader := strings.NewReader(yaml)
	agent, err := LoadFromReader(reader)
	if err != nil {
		t.Fatalf("LoadFromReader() error = %v", err)
	}

	if agent.Name != "ReaderAgent" {
		t.Errorf("Name = %q, want %q", agent.Name, "ReaderAgent")
	}
}

func TestLoadFromBytes(t *testing.T) {
	data := []byte(`
kind: Prompt
name: ByteAgent
instructions: Byte test
model:
  id: gpt-4o-mini
`)

	agent, err := LoadFromBytes(data)
	if err != nil {
		t.Fatalf("LoadFromBytes() error = %v", err)
	}

	if agent.Name != "ByteAgent" {
		t.Errorf("Name = %q, want %q", agent.Name, "ByteAgent")
	}
	if agent.Model.ID != "gpt-4o-mini" {
		t.Errorf("Model.ID = %q, want %q", agent.Model.ID, "gpt-4o-mini")
	}
}

func TestLoadWithTools(t *testing.T) {
	yaml := `
kind: Prompt
name: ToolAgent
instructions: Use tools
model:
  id: gpt-4o
tools:
  - kind: function
    name: get_weather
    description: Get the current weather
    parameters:
      type: object
      properties:
        location:
          type: string
          description: The city name
      required:
        - location
`

	agent, err := LoadFromString(yaml)
	if err != nil {
		t.Fatalf("LoadFromString() error = %v", err)
	}

	if len(agent.Tools) != 1 {
		t.Fatalf("len(Tools) = %d, want 1", len(agent.Tools))
	}

	tool := agent.Tools[0]
	if tool.Kind != "function" {
		t.Errorf("Tool.Kind = %q, want %q", tool.Kind, "function")
	}
	if tool.Name != "get_weather" {
		t.Errorf("Tool.Name = %q, want %q", tool.Name, "get_weather")
	}
	if tool.Parameters == nil {
		t.Fatal("Tool.Parameters is nil")
	}
	if len(tool.Parameters.Properties) != 1 {
		t.Errorf("len(Parameters.Properties) = %d, want 1", len(tool.Parameters.Properties))
	}
	if len(tool.Parameters.Required) != 1 {
		t.Errorf("len(Parameters.Required) = %d, want 1", len(tool.Parameters.Required))
	}
}

func TestLoadInvalidYAML(t *testing.T) {
	yaml := `
kind: Prompt
  invalid: indentation
`

	_, err := LoadFromString(yaml)
	if err == nil {
		t.Error("LoadFromString() expected error for invalid YAML")
	}
}
