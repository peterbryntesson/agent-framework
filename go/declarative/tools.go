// Copyright (c) Microsoft. All rights reserved.

package declarative

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/microsoft/agent-framework-go/tool"
)

// parseFunction creates a function tool from a declarative tool definition.
// This creates a tool with a schema but requires a binding to provide the
// actual implementation.
func parseFunction(t Tool) (tool.Tool, error) {
	if t.Name == "" {
		return nil, fmt.Errorf("tool name is required")
	}

	// Build JSON Schema for parameters
	schema, err := buildParameterSchema(t.Parameters)
	if err != nil {
		return nil, fmt.Errorf("building parameter schema: %w", err)
	}

	// Create a declarative tool that uses bindings for execution
	return &declarativeTool{
		name:        t.Name,
		description: t.Description,
		schema:      schema,
		binding:     t.Binding,
	}, nil
}

// declarativeTool implements tool.Tool for declaratively defined tools.
type declarativeTool struct {
	name        string
	description string
	schema      json.RawMessage
	binding     string
	handler     func(ctx context.Context, args json.RawMessage) (tool.Result, error)
}

// Name returns the tool name.
func (t *declarativeTool) Name() string {
	return t.name
}

// Description returns the tool description.
func (t *declarativeTool) Description() string {
	return t.description
}

// Parameters returns the JSON schema for the tool parameters.
func (t *declarativeTool) Parameters() json.RawMessage {
	return t.schema
}

// Invoke executes the tool with the given input.
// For declarative tools without a handler, this returns an error.
func (t *declarativeTool) Invoke(ctx context.Context, arguments json.RawMessage) (tool.Result, error) {
	if t.handler == nil {
		return tool.Result{}, fmt.Errorf("tool %q has no handler bound", t.name)
	}
	return t.handler(ctx, arguments)
}

// SetHandler sets the execution handler for the tool.
func (t *declarativeTool) SetHandler(handler func(ctx context.Context, args json.RawMessage) (tool.Result, error)) {
	t.handler = handler
}

// Binding returns the binding name for the tool.
func (t *declarativeTool) Binding() string {
	return t.binding
}

// buildParameterSchema creates a JSON schema from a parameter definition.
func buildParameterSchema(params *ParameterSchema) (json.RawMessage, error) {
	if params == nil {
		return json.Marshal(map[string]interface{}{
			"type":       "object",
			"properties": map[string]interface{}{},
		})
	}

	properties := make(map[string]interface{})
	for name, prop := range params.Properties {
		properties[name] = map[string]interface{}{
			"type":        prop.GetPropertyType(),
			"description": prop.Description,
		}
	}

	schema := map[string]interface{}{
		"type":       "object",
		"properties": properties,
	}

	if len(params.Required) > 0 {
		schema["required"] = params.Required
	}

	return json.Marshal(schema)
}

// BindToolHandler binds a handler function to a tool by name.
// This is used to provide implementations for declaratively defined tools.
func BindToolHandler(tools []tool.Tool, toolName string, handler func(ctx context.Context, args json.RawMessage) (tool.Result, error)) error {
	for _, t := range tools {
		if t.Name() == toolName {
			if dt, ok := t.(*declarativeTool); ok {
				dt.SetHandler(handler)
				return nil
			}
			return fmt.Errorf("tool %q is not a declarative tool", toolName)
		}
	}
	return fmt.Errorf("tool %q not found", toolName)
}
