// Copyright (c) Microsoft. All rights reserved.

package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/microsoft/agent-framework-go/tool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewToolAdapter(t *testing.T) {
	t.Run("creates adapter with client", func(t *testing.T) {
		// Arrange
		mock := &mockTransport{sendResp: createSuccessfulInitResponse()}
		client, err := NewClient(context.Background(), mock)
		require.NoError(t, err)

		// Act
		adapter := NewToolAdapter(client)

		// Assert
		assert.NotNil(t, adapter)
		assert.Equal(t, client, adapter.client)
	})
}

func TestToolAdapter_Tools(t *testing.T) {
	t.Run("returns tools from client", func(t *testing.T) {
		// Arrange
		mock := &mockTransport{sendResp: createSuccessfulInitResponse()}
		client, err := NewClient(context.Background(), mock)
		require.NoError(t, err)

		expectedTools := []ToolInfo{
			{Name: "tool1", Description: "First", InputSchema: json.RawMessage(`{"type":"object"}`)},
			{Name: "tool2", Description: "Second"},
		}
		mock.sendResp = createToolsListResponse(expectedTools)

		adapter := NewToolAdapter(client)

		// Act
		tools, err := adapter.Tools(context.Background())

		// Assert
		require.NoError(t, err)
		require.Len(t, tools, 2)
		assert.Equal(t, "tool1", tools[0].Name())
		assert.Equal(t, "tool2", tools[1].Name())
	})

	t.Run("returns error from client", func(t *testing.T) {
		// Arrange
		mock := &mockTransport{sendResp: createSuccessfulInitResponse()}
		client, err := NewClient(context.Background(), mock)
		require.NoError(t, err)

		mock.sendErr = errors.New("list failed")
		adapter := NewToolAdapter(client)

		// Act
		tools, err := adapter.Tools(context.Background())

		// Assert
		assert.Nil(t, tools)
		assert.Error(t, err)
	})
}

func TestToolAdapter_Tool(t *testing.T) {
	t.Run("returns specific tool by name", func(t *testing.T) {
		// Arrange
		mock := &mockTransport{sendResp: createSuccessfulInitResponse()}
		client, err := NewClient(context.Background(), mock)
		require.NoError(t, err)

		toolInfos := []ToolInfo{
			{Name: "alpha", Description: "Alpha tool"},
			{Name: "beta", Description: "Beta tool"},
		}
		mock.sendResp = createToolsListResponse(toolInfos)
		adapter := NewToolAdapter(client)

		// Act
		foundTool, err := adapter.Tool(context.Background(), "beta")

		// Assert
		require.NoError(t, err)
		assert.Equal(t, "beta", foundTool.Name())
		assert.Equal(t, "Beta tool", foundTool.Description())
	})

	t.Run("returns ErrToolNotFound for missing tool", func(t *testing.T) {
		// Arrange
		mock := &mockTransport{sendResp: createSuccessfulInitResponse()}
		client, err := NewClient(context.Background(), mock)
		require.NoError(t, err)

		mock.sendResp = createToolsListResponse([]ToolInfo{})
		adapter := NewToolAdapter(client)

		// Act
		foundTool, err := adapter.Tool(context.Background(), "nonexistent")

		// Assert
		assert.Nil(t, foundTool)
		assert.ErrorIs(t, err, ErrToolNotFound)
	})
}

func TestMcpBridgeTool_Name(t *testing.T) {
	t.Run("returns tool name", func(t *testing.T) {
		// Arrange
		bridgeTool := &mcpBridgeTool{
			info: ToolInfo{Name: "my_tool"},
		}

		// Act
		name := bridgeTool.Name()

		// Assert
		assert.Equal(t, "my_tool", name)
	})
}

func TestMcpBridgeTool_Description(t *testing.T) {
	t.Run("returns tool description", func(t *testing.T) {
		// Arrange
		bridgeTool := &mcpBridgeTool{
			info: ToolInfo{Description: "Does something useful"},
		}

		// Act
		desc := bridgeTool.Description()

		// Assert
		assert.Equal(t, "Does something useful", desc)
	})

	t.Run("returns empty for no description", func(t *testing.T) {
		// Arrange
		bridgeTool := &mcpBridgeTool{
			info: ToolInfo{Name: "nodesc"},
		}

		// Act
		desc := bridgeTool.Description()

		// Assert
		assert.Empty(t, desc)
	})
}

func TestMcpBridgeTool_Parameters(t *testing.T) {
	t.Run("returns input schema", func(t *testing.T) {
		// Arrange
		schema := json.RawMessage(`{"type":"object","properties":{"x":{"type":"number"}}}`)
		bridgeTool := &mcpBridgeTool{
			info: ToolInfo{InputSchema: schema},
		}

		// Act
		params := bridgeTool.Parameters()

		// Assert
		assert.JSONEq(t, string(schema), string(params))
	})

	t.Run("returns nil for no schema", func(t *testing.T) {
		// Arrange
		bridgeTool := &mcpBridgeTool{
			info: ToolInfo{Name: "noschema"},
		}

		// Act
		params := bridgeTool.Parameters()

		// Assert
		assert.Nil(t, params)
	})
}

func TestMcpBridgeTool_Invoke(t *testing.T) {
	t.Run("invokes tool and returns result", func(t *testing.T) {
		// Arrange
		mock := &mockTransport{sendResp: createSuccessfulInitResponse()}
		client, err := NewClient(context.Background(), mock)
		require.NoError(t, err)

		bridgeTool := &mcpBridgeTool{
			client: client,
			info:   ToolInfo{Name: "test_tool"},
		}

		mock.sendResp = createCallToolResponse("tool output", false)

		// Act
		result, err := bridgeTool.Invoke(context.Background(), json.RawMessage(`{"input":"value"}`))

		// Assert
		require.NoError(t, err)
		assert.Equal(t, "tool output", result.Content)
		assert.False(t, result.IsError)
	})

	t.Run("handles error result from tool", func(t *testing.T) {
		// Arrange
		mock := &mockTransport{sendResp: createSuccessfulInitResponse()}
		client, err := NewClient(context.Background(), mock)
		require.NoError(t, err)

		bridgeTool := &mcpBridgeTool{
			client: client,
			info:   ToolInfo{Name: "failing_tool"},
		}

		mock.sendResp = createCallToolResponse("error occurred", true)

		// Act
		result, err := bridgeTool.Invoke(context.Background(), nil)

		// Assert
		require.NoError(t, err)
		assert.True(t, result.IsError)
		assert.Equal(t, "error occurred", result.Content)
	})

	t.Run("returns error on call failure", func(t *testing.T) {
		// Arrange
		mock := &mockTransport{sendResp: createSuccessfulInitResponse()}
		client, err := NewClient(context.Background(), mock)
		require.NoError(t, err)

		bridgeTool := &mcpBridgeTool{
			client: client,
			info:   ToolInfo{Name: "tool"},
		}

		mock.sendErr = errors.New("call failed")

		// Act
		result, err := bridgeTool.Invoke(context.Background(), nil)

		// Assert
		assert.Error(t, err)
		assert.Equal(t, tool.Result{}, result)
	})
}

func TestConvertCallToolResult(t *testing.T) {
	t.Run("handles nil result", func(t *testing.T) {
		// Act
		result := convertCallToolResult(nil)

		// Assert
		assert.Empty(t, result.Content)
		assert.False(t, result.IsError)
	})

	t.Run("handles empty content", func(t *testing.T) {
		// Arrange
		mcpResult := &CallToolResult{Content: []Content{}}

		// Act
		result := convertCallToolResult(mcpResult)

		// Assert
		assert.Empty(t, result.Content)
	})

	t.Run("converts single text content", func(t *testing.T) {
		// Arrange
		mcpResult := &CallToolResult{
			Content: []Content{NewTextContent("hello world")},
		}

		// Act
		result := convertCallToolResult(mcpResult)

		// Assert
		assert.Equal(t, "hello world", result.Content)
		assert.False(t, result.IsError)
	})

	t.Run("joins multiple text contents", func(t *testing.T) {
		// Arrange
		mcpResult := &CallToolResult{
			Content: []Content{
				NewTextContent("line 1"),
				NewTextContent("line 2"),
				NewTextContent("line 3"),
			},
		}

		// Act
		result := convertCallToolResult(mcpResult)

		// Assert
		assert.Equal(t, "line 1\nline 2\nline 3", result.Content)
	})

	t.Run("preserves isError flag", func(t *testing.T) {
		// Arrange
		mcpResult := &CallToolResult{
			Content: []Content{NewTextContent("error")},
			IsError: true,
		}

		// Act
		result := convertCallToolResult(mcpResult)

		// Assert
		assert.True(t, result.IsError)
	})

	t.Run("handles image content in metadata", func(t *testing.T) {
		// Arrange
		mcpResult := &CallToolResult{
			Content: []Content{
				NewImageContent("base64data", "image/png"),
			},
		}

		// Act
		result := convertCallToolResult(mcpResult)

		// Assert
		assert.Contains(t, result.Content, "non-text content")
		require.NotNil(t, result.Metadata)
		assert.Contains(t, result.Metadata, "image")

		imgData := result.Metadata["image"].(map[string]any)
		assert.Equal(t, "base64data", imgData["data"])
		assert.Equal(t, "image/png", imgData["mimeType"])
	})

	t.Run("handles resource content in metadata", func(t *testing.T) {
		// Arrange
		mcpResult := &CallToolResult{
			Content: []Content{
				{Type: ContentTypeResource, URI: "file:///doc.txt", MimeType: "text/plain"},
			},
		}

		// Act
		result := convertCallToolResult(mcpResult)

		// Assert
		require.NotNil(t, result.Metadata)
		assert.Contains(t, result.Metadata, "resource")

		resData := result.Metadata["resource"].(map[string]any)
		assert.Equal(t, "file:///doc.txt", resData["uri"])
	})

	t.Run("handles mixed content with indexed keys", func(t *testing.T) {
		// Arrange
		mcpResult := &CallToolResult{
			Content: []Content{
				NewTextContent("text part"),
				NewImageContent("img1", "image/png"),
				NewImageContent("img2", "image/jpeg"),
			},
		}

		// Act
		result := convertCallToolResult(mcpResult)

		// Assert
		assert.Equal(t, "text part", result.Content)
		require.NotNil(t, result.Metadata)
		// Should have indexed keys for multiple images
		assert.Contains(t, result.Metadata, "image_1")
		assert.Contains(t, result.Metadata, "image_2")
	})
}

func TestFormatIndexedKey(t *testing.T) {
	tests := []struct {
		prefix   string
		index    int
		expected string
	}{
		{"image", 0, "image_0"},
		{"image", 1, "image_1"},
		{"resource", 5, "resource_5"},
		{"item", 10, "item_10"},
		{"item", 123, "item_123"},
	}

	for _, tc := range tests {
		t.Run(tc.expected, func(t *testing.T) {
			result := formatIndexedKey(tc.prefix, tc.index)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestFormatInt(t *testing.T) {
	tests := []struct {
		input    int
		expected string
	}{
		{0, "0"},
		{1, "1"},
		{9, "9"},
		{10, "10"},
		{42, "42"},
		{100, "100"},
		{999, "999"},
		{-1, "-1"},
		{-42, "-42"},
	}

	for _, tc := range tests {
		t.Run(tc.expected, func(t *testing.T) {
			result := formatInt(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}
