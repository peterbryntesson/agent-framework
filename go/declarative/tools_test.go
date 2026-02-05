// Copyright (c) Microsoft. All rights reserved.

package declarative

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/microsoft/agent-framework-go/tool"
)

func TestParseFunction(t *testing.T) {
	toolDef := Tool{
		Name:        "get_weather",
		Description: "Get the current weather",
		Kind:        "function",
		Parameters: &ParameterSchema{
			Type: "object",
			Properties: map[string]PropertySchema{
				"location": {
					Type:        "string",
					Description: "The city name",
				},
			},
			Required: []string{"location"},
		},
	}

	parsed, err := parseFunction(toolDef)
	if err != nil {
		t.Fatalf("parseFunction() error = %v", err)
	}

	if parsed.Name() != "get_weather" {
		t.Errorf("Name() = %q, want %q", parsed.Name(), "get_weather")
	}
	if parsed.Description() != "Get the current weather" {
		t.Errorf("Description() = %q, want %q", parsed.Description(), "Get the current weather")
	}

	params := parsed.Parameters()
	if params == nil {
		t.Fatal("Parameters() returned nil")
	}

	// Verify schema structure
	var schema map[string]interface{}
	if err := json.Unmarshal(params, &schema); err != nil {
		t.Fatalf("Failed to unmarshal parameters: %v", err)
	}

	if schema["type"] != "object" {
		t.Errorf("schema.type = %v, want %q", schema["type"], "object")
	}

	props, ok := schema["properties"].(map[string]interface{})
	if !ok {
		t.Fatal("schema.properties is not a map")
	}

	if _, ok := props["location"]; !ok {
		t.Error("schema.properties.location not found")
	}
}

func TestParseFunctionMissingName(t *testing.T) {
	toolDef := Tool{
		Description: "No name tool",
	}

	_, err := parseFunction(toolDef)
	if err == nil {
		t.Error("parseFunction() expected error for missing name")
	}
}

func TestParseFunctionNilParameters(t *testing.T) {
	toolDef := Tool{
		Name:        "simple_tool",
		Description: "A simple tool",
	}

	parsed, err := parseFunction(toolDef)
	if err != nil {
		t.Fatalf("parseFunction() error = %v", err)
	}

	params := parsed.Parameters()
	if params == nil {
		t.Fatal("Parameters() returned nil for tool with no parameters")
	}

	var schema map[string]interface{}
	if err := json.Unmarshal(params, &schema); err != nil {
		t.Fatalf("Failed to unmarshal parameters: %v", err)
	}

	if schema["type"] != "object" {
		t.Errorf("schema.type = %v, want %q", schema["type"], "object")
	}
}

func TestDeclarativeToolInvokeWithoutHandler(t *testing.T) {
	dt := &declarativeTool{
		name:        "test_tool",
		description: "A test tool",
	}

	_, err := dt.Invoke(context.Background(), nil)
	if err == nil {
		t.Error("Invoke() expected error for tool without handler")
	}
}

func TestDeclarativeToolInvokeWithHandler(t *testing.T) {
	dt := &declarativeTool{
		name:        "test_tool",
		description: "A test tool",
	}

	handler := func(ctx context.Context, args json.RawMessage) (tool.Result, error) {
		return tool.Result{Content: "handler called"}, nil
	}
	dt.SetHandler(handler)

	result, err := dt.Invoke(context.Background(), nil)
	if err != nil {
		t.Fatalf("Invoke() error = %v", err)
	}

	if result.Content != "handler called" {
		t.Errorf("result.Content = %q, want %q", result.Content, "handler called")
	}
}

func TestDeclarativeToolBinding(t *testing.T) {
	dt := &declarativeTool{
		name:    "test_tool",
		binding: "myBinding",
	}

	if dt.Binding() != "myBinding" {
		t.Errorf("Binding() = %q, want %q", dt.Binding(), "myBinding")
	}
}

func TestBindToolHandler(t *testing.T) {
	tools := []tool.Tool{
		&declarativeTool{name: "tool1"},
		&declarativeTool{name: "tool2"},
	}

	handler := func(ctx context.Context, args json.RawMessage) (tool.Result, error) {
		return tool.Result{Content: "bound"}, nil
	}

	err := BindToolHandler(tools, "tool1", handler)
	if err != nil {
		t.Fatalf("BindToolHandler() error = %v", err)
	}

	// Verify handler was bound
	dt := tools[0].(*declarativeTool)
	result, err := dt.Invoke(context.Background(), nil)
	if err != nil {
		t.Fatalf("Invoke() error = %v", err)
	}

	if result.Content != "bound" {
		t.Errorf("result.Content = %q, want %q", result.Content, "bound")
	}
}

func TestBindToolHandlerNotFound(t *testing.T) {
	tools := []tool.Tool{
		&declarativeTool{name: "tool1"},
	}

	handler := func(ctx context.Context, args json.RawMessage) (tool.Result, error) {
		return tool.Result{}, nil
	}

	err := BindToolHandler(tools, "nonexistent", handler)
	if err == nil {
		t.Error("BindToolHandler() expected error for nonexistent tool")
	}
}

func TestBuildParameterSchema(t *testing.T) {
	params := &ParameterSchema{
		Type: "object",
		Properties: map[string]PropertySchema{
			"name": {
				Type:        "string",
				Description: "The name",
			},
			"age": {
				Type:        "integer",
				Description: "The age",
			},
		},
		Required: []string{"name"},
	}

	schema, err := buildParameterSchema(params)
	if err != nil {
		t.Fatalf("buildParameterSchema() error = %v", err)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal(schema, &parsed); err != nil {
		t.Fatalf("Failed to unmarshal schema: %v", err)
	}

	if parsed["type"] != "object" {
		t.Errorf("type = %v, want %q", parsed["type"], "object")
	}

	required, ok := parsed["required"].([]interface{})
	if !ok {
		t.Fatal("required is not an array")
	}

	if len(required) != 1 || required[0] != "name" {
		t.Errorf("required = %v, want [\"name\"]", required)
	}
}
