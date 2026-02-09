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

// parseMCP creates a hosted MCP tool from a declarative tool definition.
func parseMCP(t Tool) (tool.Tool, error) {
	serverURL := t.Server
	if t.URL != "" {
		serverURL = t.URL
	}
	if t.Name == "" {
		return nil, fmt.Errorf("tool name is required")
	}
	if serverURL == "" {
		return nil, fmt.Errorf("mcp tool server url is required")
	}

	hosted := tool.NewHostedMCPTool(t.Name, serverURL)
	hosted.ToolDescription = t.Description
	if t.ServerName != "" {
		hosted.ServerLabel = t.ServerName
	} else if t.ServerDescription != "" {
		hosted.ServerLabel = t.ServerDescription
	}
	hosted.AllowedTools = t.AllowedTools

	if t.ApprovalMode != nil {
		always := t.ApprovalMode.AlwaysRequireApproval
		if len(always) == 0 {
			always = t.ApprovalMode.AlwaysRequireApprovalTools
		}
		never := t.ApprovalMode.NeverRequireApproval
		if len(never) == 0 {
			never = t.ApprovalMode.NeverRequireApprovalTools
		}
		if len(always) > 0 || len(never) > 0 {
			hosted.SpecificApproval = &tool.MCPSpecificApproval{
				AlwaysRequireApproval: always,
				NeverRequireApproval:  never,
			}
		} else if t.ApprovalMode.Kind != "" {
			switch normalizeAPIType(t.ApprovalMode.Kind) {
			case "never":
				hosted.RequireApproval = tool.MCPApprovalNever
			case "always":
				hosted.RequireApproval = tool.MCPApprovalAlways
			case "specify":
				// No specific tools provided; keep default approval behavior.
			default:
				return nil, fmt.Errorf("unsupported mcp approval mode %q", t.ApprovalMode.Kind)
			}
		}
	}

	return hosted, nil
}

// parseWebSearch creates a hosted web search tool.
func parseWebSearch(t Tool) (tool.Tool, error) {
	hosted := tool.NewHostedWebSearchTool()
	if t.Name != "" {
		hosted.ToolName = t.Name
	}
	hosted.ToolDescription = t.Description
	hosted.SearchContextSize = t.SearchContextSize
	if t.UserLocation != nil {
		hosted.UserLocation = &tool.UserLocation{
			Type:        t.UserLocation.Type,
			City:        t.UserLocation.City,
			Region:      t.UserLocation.Region,
			Country:     t.UserLocation.Country,
			CountryCode: t.UserLocation.CountryCode,
			Timezone:    t.UserLocation.Timezone,
		}
	}
	return hosted, nil
}

// parseFileSearch creates a hosted file search tool.
func parseFileSearch(t Tool) (tool.Tool, error) {
	hosted := tool.NewHostedFileSearchTool()
	if t.Name != "" {
		hosted.ToolName = t.Name
	}
	hosted.ToolDescription = t.Description
	hosted.VectorStoreIDs = t.VectorStoreIDs
	if t.MaximumResultCount > 0 {
		hosted.MaxResults = t.MaximumResultCount
	} else {
		hosted.MaxResults = t.MaxResults
	}
	if t.Ranking != nil || t.Ranker != "" || t.ScoreThreshold != nil {
		hosted.Ranking = &tool.FileSearchRanking{
			Ranker:         t.Ranker,
			ScoreThreshold: 0,
		}
		if t.Ranking != nil {
			hosted.Ranking.Ranker = t.Ranking.Ranker
			hosted.Ranking.ScoreThreshold = t.Ranking.ScoreThreshold
		}
		if t.ScoreThreshold != nil {
			hosted.Ranking.ScoreThreshold = *t.ScoreThreshold
		}
	}
	if len(t.Filters) > 0 {
		hosted.AdditionalProperties = map[string]interface{}{
			"filters": t.Filters,
		}
	}
	return hosted, nil
}

// parseCodeInterpreter creates a hosted code interpreter tool.
func parseCodeInterpreter(t Tool) (tool.Tool, error) {
	hosted := tool.NewHostedCodeInterpreterTool()
	if t.Name != "" {
		hosted.ToolName = t.Name
	}
	hosted.ToolDescription = t.Description
	if t.Container != nil {
		hosted.Container = &tool.CodeInterpreterContainer{
			Image:   t.Container.Image,
			EnvVars: t.Container.EnvVars,
		}
	}
	hosted.FileIDs = t.FileIDs
	return hosted, nil
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
