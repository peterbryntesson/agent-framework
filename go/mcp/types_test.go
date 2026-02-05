// Copyright (c) Microsoft. All rights reserved.

package mcp

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewTextContent(t *testing.T) {
	t.Run("creates text content with correct type", func(t *testing.T) {
		// Act
		content := NewTextContent("hello world")

		// Assert
		assert.Equal(t, ContentTypeText, content.Type)
		assert.Equal(t, "hello world", content.Text)
		assert.Empty(t, content.Data)
		assert.Empty(t, content.MimeType)
		assert.Empty(t, content.URI)
	})

	t.Run("handles empty text", func(t *testing.T) {
		// Act
		content := NewTextContent("")

		// Assert
		assert.Equal(t, ContentTypeText, content.Type)
		assert.Empty(t, content.Text)
	})
}

func TestNewImageContent(t *testing.T) {
	t.Run("creates image content with data and mime type", func(t *testing.T) {
		// Act
		content := NewImageContent("base64data", "image/png")

		// Assert
		assert.Equal(t, ContentTypeImage, content.Type)
		assert.Equal(t, "base64data", content.Data)
		assert.Equal(t, "image/png", content.MimeType)
		assert.Empty(t, content.Text)
		assert.Empty(t, content.URI)
	})

	t.Run("handles different mime types", func(t *testing.T) {
		tests := []struct {
			name     string
			mimeType string
		}{
			{"jpeg", "image/jpeg"},
			{"gif", "image/gif"},
			{"webp", "image/webp"},
		}

		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				content := NewImageContent("data", tc.mimeType)
				assert.Equal(t, tc.mimeType, content.MimeType)
			})
		}
	})
}

func TestToolInfo_JSONSerialization(t *testing.T) {
	t.Run("marshals with all fields", func(t *testing.T) {
		// Arrange
		info := ToolInfo{
			Name:        "test_tool",
			Description: "A test tool",
			InputSchema: json.RawMessage(`{"type":"object"}`),
		}

		// Act
		data, err := json.Marshal(info)

		// Assert
		require.NoError(t, err)
		assert.Contains(t, string(data), `"name":"test_tool"`)
		assert.Contains(t, string(data), `"description":"A test tool"`)
		assert.Contains(t, string(data), `"inputSchema":{"type":"object"}`)
	})

	t.Run("omits empty description", func(t *testing.T) {
		// Arrange
		info := ToolInfo{
			Name: "simple_tool",
		}

		// Act
		data, err := json.Marshal(info)

		// Assert
		require.NoError(t, err)
		assert.Contains(t, string(data), `"name":"simple_tool"`)
		assert.NotContains(t, string(data), `"description"`)
	})

	t.Run("unmarshals correctly", func(t *testing.T) {
		// Arrange
		jsonData := `{"name":"parsed_tool","description":"Parsed description","inputSchema":{"type":"string"}}`

		// Act
		var info ToolInfo
		err := json.Unmarshal([]byte(jsonData), &info)

		// Assert
		require.NoError(t, err)
		assert.Equal(t, "parsed_tool", info.Name)
		assert.Equal(t, "Parsed description", info.Description)
		assert.JSONEq(t, `{"type":"string"}`, string(info.InputSchema))
	})
}

func TestContent_JSONSerialization(t *testing.T) {
	t.Run("marshals text content", func(t *testing.T) {
		// Arrange
		content := NewTextContent("test message")

		// Act
		data, err := json.Marshal(content)

		// Assert
		require.NoError(t, err)
		assert.Contains(t, string(data), `"type":"text"`)
		assert.Contains(t, string(data), `"text":"test message"`)
	})

	t.Run("marshals image content", func(t *testing.T) {
		// Arrange
		content := NewImageContent("abc123", "image/png")

		// Act
		data, err := json.Marshal(content)

		// Assert
		require.NoError(t, err)
		assert.Contains(t, string(data), `"type":"image"`)
		assert.Contains(t, string(data), `"data":"abc123"`)
		assert.Contains(t, string(data), `"mimeType":"image/png"`)
	})

	t.Run("marshals resource content", func(t *testing.T) {
		// Arrange
		content := Content{
			Type: ContentTypeResource,
			URI:  "file:///path/to/file",
		}

		// Act
		data, err := json.Marshal(content)

		// Assert
		require.NoError(t, err)
		assert.Contains(t, string(data), `"type":"resource"`)
		assert.Contains(t, string(data), `"uri":"file:///path/to/file"`)
	})

	t.Run("unmarshals mixed content", func(t *testing.T) {
		// Arrange
		jsonData := `{"type":"text","text":"hello","mimeType":"text/plain"}`

		// Act
		var content Content
		err := json.Unmarshal([]byte(jsonData), &content)

		// Assert
		require.NoError(t, err)
		assert.Equal(t, "text", content.Type)
		assert.Equal(t, "hello", content.Text)
		assert.Equal(t, "text/plain", content.MimeType)
	})
}

func TestCallToolResult_JSONSerialization(t *testing.T) {
	t.Run("marshals success result", func(t *testing.T) {
		// Arrange
		result := CallToolResult{
			Content: []Content{NewTextContent("success")},
			IsError: false,
		}

		// Act
		data, err := json.Marshal(result)

		// Assert
		require.NoError(t, err)
		assert.Contains(t, string(data), `"content":[{"type":"text","text":"success"}]`)
		// IsError should be omitted when false (omitempty)
		assert.NotContains(t, string(data), `"isError"`)
	})

	t.Run("marshals error result", func(t *testing.T) {
		// Arrange
		result := CallToolResult{
			Content: []Content{NewTextContent("something went wrong")},
			IsError: true,
		}

		// Act
		data, err := json.Marshal(result)

		// Assert
		require.NoError(t, err)
		assert.Contains(t, string(data), `"isError":true`)
	})

	t.Run("marshals multiple content items", func(t *testing.T) {
		// Arrange
		result := CallToolResult{
			Content: []Content{
				NewTextContent("line 1"),
				NewTextContent("line 2"),
				NewImageContent("imgdata", "image/jpeg"),
			},
		}

		// Act
		data, err := json.Marshal(result)

		// Assert
		require.NoError(t, err)
		var parsed CallToolResult
		err = json.Unmarshal(data, &parsed)
		require.NoError(t, err)
		assert.Len(t, parsed.Content, 3)
	})

	t.Run("unmarshals correctly", func(t *testing.T) {
		// Arrange
		jsonData := `{"content":[{"type":"text","text":"result"}],"isError":false}`

		// Act
		var result CallToolResult
		err := json.Unmarshal([]byte(jsonData), &result)

		// Assert
		require.NoError(t, err)
		require.Len(t, result.Content, 1)
		assert.Equal(t, "text", result.Content[0].Type)
		assert.Equal(t, "result", result.Content[0].Text)
		assert.False(t, result.IsError)
	})
}

func TestMCPError(t *testing.T) {
	t.Run("Error returns formatted message with cause", func(t *testing.T) {
		// Arrange
		cause := assert.AnError
		err := NewMCPError("tools/call", cause)

		// Act
		msg := err.Error()

		// Assert
		assert.Contains(t, msg, "mcp tools/call")
		assert.Contains(t, msg, cause.Error())
	})

	t.Run("Error returns formatted message without cause", func(t *testing.T) {
		// Arrange
		err := &MCPError{Operation: "initialize"}

		// Act
		msg := err.Error()

		// Assert
		assert.Equal(t, "mcp initialize failed", msg)
	})

	t.Run("Unwrap returns cause", func(t *testing.T) {
		// Arrange
		cause := assert.AnError
		err := NewMCPError("test", cause)

		// Act
		unwrapped := err.Unwrap()

		// Assert
		assert.Equal(t, cause, unwrapped)
	})
}

func TestResponseError(t *testing.T) {
	t.Run("Error returns formatted message", func(t *testing.T) {
		// Arrange
		err := &ResponseError{
			Code:    InvalidParams,
			Message: "missing required field",
		}

		// Act
		msg := err.Error()

		// Assert
		assert.Contains(t, msg, "MCP error")
		assert.Contains(t, msg, "-32602")
		assert.Contains(t, msg, "missing required field")
	})

	t.Run("JSON serialization round-trip", func(t *testing.T) {
		// Arrange
		err := &ResponseError{
			Code:    MethodNotFound,
			Message: "unknown method",
			Data:    json.RawMessage(`{"detail":"foo"}`),
		}

		// Act
		data, marshalErr := json.Marshal(err)
		require.NoError(t, marshalErr)

		var parsed ResponseError
		unmarshalErr := json.Unmarshal(data, &parsed)

		// Assert
		require.NoError(t, unmarshalErr)
		assert.Equal(t, MethodNotFound, parsed.Code)
		assert.Equal(t, "unknown method", parsed.Message)
		assert.JSONEq(t, `{"detail":"foo"}`, string(parsed.Data))
	})
}

func TestResourceInfo_JSONSerialization(t *testing.T) {
	t.Run("marshals with all fields", func(t *testing.T) {
		// Arrange
		info := ResourceInfo{
			URI:         "file:///test.txt",
			Name:        "Test File",
			Description: "A test resource",
			MimeType:    "text/plain",
		}

		// Act
		data, err := json.Marshal(info)

		// Assert
		require.NoError(t, err)
		assert.Contains(t, string(data), `"uri":"file:///test.txt"`)
		assert.Contains(t, string(data), `"name":"Test File"`)
		assert.Contains(t, string(data), `"description":"A test resource"`)
		assert.Contains(t, string(data), `"mimeType":"text/plain"`)
	})

	t.Run("omits empty optional fields", func(t *testing.T) {
		// Arrange
		info := ResourceInfo{
			URI:  "file:///minimal.txt",
			Name: "Minimal",
		}

		// Act
		data, err := json.Marshal(info)

		// Assert
		require.NoError(t, err)
		assert.NotContains(t, string(data), `"description"`)
		assert.NotContains(t, string(data), `"mimeType"`)
	})
}

func TestImplementation_JSONSerialization(t *testing.T) {
	t.Run("marshals correctly", func(t *testing.T) {
		// Arrange
		impl := Implementation{
			Name:    "test-client",
			Version: "1.0.0",
		}

		// Act
		data, err := json.Marshal(impl)

		// Assert
		require.NoError(t, err)
		assert.Contains(t, string(data), `"name":"test-client"`)
		assert.Contains(t, string(data), `"version":"1.0.0"`)
	})

	t.Run("unmarshals correctly", func(t *testing.T) {
		// Arrange
		jsonData := `{"name":"server","version":"2.0.0"}`

		// Act
		var impl Implementation
		err := json.Unmarshal([]byte(jsonData), &impl)

		// Assert
		require.NoError(t, err)
		assert.Equal(t, "server", impl.Name)
		assert.Equal(t, "2.0.0", impl.Version)
	})
}

func TestConstants(t *testing.T) {
	t.Run("ProtocolVersion", func(t *testing.T) {
		assert.Equal(t, "2024-11-05", ProtocolVersion)
	})

	t.Run("JSONRPCVersion", func(t *testing.T) {
		assert.Equal(t, "2.0", JSONRPCVersion)
	})

	t.Run("ContentType constants", func(t *testing.T) {
		assert.Equal(t, "text", ContentTypeText)
		assert.Equal(t, "image", ContentTypeImage)
		assert.Equal(t, "resource", ContentTypeResource)
	})

	t.Run("Error code constants", func(t *testing.T) {
		assert.Equal(t, -32700, ParseError)
		assert.Equal(t, -32600, InvalidRequest)
		assert.Equal(t, -32601, MethodNotFound)
		assert.Equal(t, -32602, InvalidParams)
		assert.Equal(t, -32603, InternalError)
	})
}

func TestSentinelErrors(t *testing.T) {
	tests := []struct {
		name string
		err  error
		msg  string
	}{
		{"ErrNotInitialized", ErrNotInitialized, "not initialized"},
		{"ErrClosed", ErrClosed, "closed"},
		{"ErrTimeout", ErrTimeout, "timed out"},
		{"ErrToolNotFound", ErrToolNotFound, "tool not found"},
		{"ErrResourceNotFound", ErrResourceNotFound, "resource not found"},
		{"ErrPromptNotFound", ErrPromptNotFound, "prompt not found"},
		{"ErrInvalidParams", ErrInvalidParams, "invalid parameters"},
		{"ErrProtocolVersion", ErrProtocolVersion, "protocol version"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Error(t, tc.err)
			assert.Contains(t, tc.err.Error(), tc.msg)
		})
	}
}

func TestRequest_JSONSerialization(t *testing.T) {
	t.Run("marshals full request", func(t *testing.T) {
		// Arrange
		req := Request{
			JSONRPC: JSONRPCVersion,
			ID:      int64(1),
			Method:  "tools/list",
			Params:  json.RawMessage(`{"cursor":"abc"}`),
		}

		// Act
		data, err := json.Marshal(req)

		// Assert
		require.NoError(t, err)
		assert.Contains(t, string(data), `"jsonrpc":"2.0"`)
		assert.Contains(t, string(data), `"id":1`)
		assert.Contains(t, string(data), `"method":"tools/list"`)
		assert.Contains(t, string(data), `"params":{"cursor":"abc"}`)
	})

	t.Run("marshals notification (no ID)", func(t *testing.T) {
		// Arrange
		req := Request{
			JSONRPC: JSONRPCVersion,
			Method:  "notifications/initialized",
		}

		// Act
		data, err := json.Marshal(req)

		// Assert
		require.NoError(t, err)
		assert.NotContains(t, string(data), `"id"`)
	})
}

func TestResponse_JSONSerialization(t *testing.T) {
	t.Run("marshals success response", func(t *testing.T) {
		// Arrange
		resp := Response{
			JSONRPC: JSONRPCVersion,
			ID:      int64(42),
			Result:  json.RawMessage(`{"tools":[]}`),
		}

		// Act
		data, err := json.Marshal(resp)

		// Assert
		require.NoError(t, err)
		assert.Contains(t, string(data), `"jsonrpc":"2.0"`)
		assert.Contains(t, string(data), `"id":42`)
		assert.Contains(t, string(data), `"result":{"tools":[]}`)
	})

	t.Run("marshals error response", func(t *testing.T) {
		// Arrange
		resp := Response{
			JSONRPC: JSONRPCVersion,
			ID:      int64(1),
			Error: &ResponseError{
				Code:    MethodNotFound,
				Message: "unknown method",
			},
		}

		// Act
		data, err := json.Marshal(resp)

		// Assert
		require.NoError(t, err)
		assert.Contains(t, string(data), `"error":`)
		assert.Contains(t, string(data), `"code":-32601`)
	})
}
