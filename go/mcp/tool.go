// Copyright (c) Microsoft. All rights reserved.

package mcp

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/microsoft/agent-framework-go/tool"
)

// ToolAdapter provides access to MCP tools as framework tools.
// It bridges MCP server tools to the framework's tool.Tool interface,
// enabling agents to use MCP tools seamlessly.
type ToolAdapter struct {
	client *Client
}

// NewToolAdapter creates a new adapter for the given client.
// The adapter uses the client to list and invoke MCP tools.
func NewToolAdapter(client *Client) *ToolAdapter {
	return &ToolAdapter{
		client: client,
	}
}

// Tools returns all MCP tools as framework tool.Tool implementations.
// It queries the MCP server for available tools and wraps each one.
func (a *ToolAdapter) Tools(ctx context.Context) ([]tool.Tool, error) {
	mcpTools, err := a.client.ListTools(ctx)
	if err != nil {
		return nil, err
	}

	tools := make([]tool.Tool, len(mcpTools))
	for i, t := range mcpTools {
		tools[i] = &mcpBridgeTool{
			client: a.client,
			info:   t,
		}
	}

	return tools, nil
}

// Tool returns a single MCP tool by name as a framework tool.
// Returns ErrToolNotFound if no tool with the given name exists.
func (a *ToolAdapter) Tool(ctx context.Context, name string) (tool.Tool, error) {
	mcpTools, err := a.client.ListTools(ctx)
	if err != nil {
		return nil, err
	}

	for _, t := range mcpTools {
		if t.Name == name {
			return &mcpBridgeTool{
				client: a.client,
				info:   t,
			}, nil
		}
	}

	return nil, ErrToolNotFound
}

// mcpBridgeTool wraps an MCP tool as a framework tool.Tool.
type mcpBridgeTool struct {
	client *Client
	info   ToolInfo
}

// Name returns the tool's unique identifier.
func (t *mcpBridgeTool) Name() string {
	return t.info.Name
}

// Description returns a human-readable description of what the tool does.
func (t *mcpBridgeTool) Description() string {
	return t.info.Description
}

// Parameters returns the JSON Schema describing the tool's input parameters.
func (t *mcpBridgeTool) Parameters() json.RawMessage {
	return t.info.InputSchema
}

// Invoke executes the tool on the MCP server and returns the result.
func (t *mcpBridgeTool) Invoke(ctx context.Context, arguments json.RawMessage) (tool.Result, error) {
	result, err := t.client.CallTool(ctx, t.info.Name, arguments)
	if err != nil {
		return tool.Result{}, err
	}

	return convertCallToolResult(result), nil
}

// convertCallToolResult converts an MCP CallToolResult to a framework tool.Result.
func convertCallToolResult(mcpResult *CallToolResult) tool.Result {
	if mcpResult == nil || len(mcpResult.Content) == 0 {
		return tool.Result{
			Content: "",
			IsError: false,
		}
	}

	// Collect all text content from the result
	var texts []string
	metadata := make(map[string]any)
	hasNonText := false

	for i, content := range mcpResult.Content {
		switch content.Type {
		case ContentTypeText:
			texts = append(texts, content.Text)

		case ContentTypeImage:
			hasNonText = true
			// Store image data in metadata
			key := "image"
			if len(mcpResult.Content) > 1 {
				key = formatIndexedKey("image", i)
			}
			metadata[key] = map[string]any{
				"data":     content.Data,
				"mimeType": content.MimeType,
			}

		case ContentTypeResource:
			hasNonText = true
			// Store resource reference in metadata
			key := "resource"
			if len(mcpResult.Content) > 1 {
				key = formatIndexedKey("resource", i)
			}
			metadata[key] = map[string]any{
				"uri":      content.URI,
				"mimeType": content.MimeType,
			}
		}
	}

	// Build the result content
	var content string
	switch {
	case len(texts) == 1:
		content = texts[0]
	case len(texts) > 1:
		content = strings.Join(texts, "\n")
	case hasNonText:
		// No text content, provide a summary
		content = "[non-text content - see metadata]"
	}

	result := tool.Result{
		Content: content,
		IsError: mcpResult.IsError,
	}

	if len(metadata) > 0 {
		result.Metadata = metadata
	}

	return result
}

// formatIndexedKey creates an indexed key for metadata entries.
func formatIndexedKey(prefix string, index int) string {
	return prefix + "_" + formatInt(index)
}

// formatInt converts an integer to a string without using fmt.
func formatInt(n int) string {
	if n == 0 {
		return "0"
	}
	if n < 0 {
		return "-" + formatInt(-n)
	}
	var digits []byte
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}
	return string(digits)
}
