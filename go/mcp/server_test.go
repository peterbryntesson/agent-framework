// Copyright (c) Microsoft. All rights reserved.

package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/microsoft/agent-framework-go/tool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockTool is a test implementation of tool.Tool.
type mockTool struct {
	name   string
	desc   string
	params json.RawMessage
	result tool.Result
	err    error
}

func (m *mockTool) Name() string                { return m.name }
func (m *mockTool) Description() string         { return m.desc }
func (m *mockTool) Parameters() json.RawMessage { return m.params }
func (m *mockTool) Invoke(ctx context.Context, arguments json.RawMessage) (tool.Result, error) {
	return m.result, m.err
}

func TestNewServer(t *testing.T) {
	t.Run("creates server with default values", func(t *testing.T) {
		// Act
		server := NewServer()

		// Assert
		assert.NotNil(t, server)
		assert.Equal(t, "agent-framework-go", server.serverInfo.Name)
		assert.Equal(t, "1.0.0", server.serverInfo.Version)
	})

	t.Run("applies server options", func(t *testing.T) {
		// Arrange
		testTool := &mockTool{name: "test"}
		testResource := ResourceInfo{URI: "file:///test.txt", Name: "Test"}

		// Act
		server := NewServer(
			WithServerInfo("custom-server", "2.0.0"),
			WithTools(testTool),
			WithResources(testResource),
		)

		// Assert
		assert.Equal(t, "custom-server", server.serverInfo.Name)
		assert.Equal(t, "2.0.0", server.serverInfo.Version)
		require.Len(t, server.tools, 1)
		require.Len(t, server.resources, 1)
	})

	t.Run("sets capabilities based on configuration", func(t *testing.T) {
		// Arrange & Act
		serverWithTools := NewServer(WithTools(&mockTool{name: "t1"}))
		serverWithResources := NewServer(WithResources(ResourceInfo{URI: "x", Name: "X"}))
		serverWithBoth := NewServer(
			WithTools(&mockTool{name: "t2"}),
			WithResources(ResourceInfo{URI: "y", Name: "Y"}),
		)
		serverEmpty := NewServer()

		// Assert
		assert.NotNil(t, serverWithTools.capabilities.Tools)
		assert.Nil(t, serverWithTools.capabilities.Resources)

		assert.Nil(t, serverWithResources.capabilities.Tools)
		assert.NotNil(t, serverWithResources.capabilities.Resources)

		assert.NotNil(t, serverWithBoth.capabilities.Tools)
		assert.NotNil(t, serverWithBoth.capabilities.Resources)

		assert.Nil(t, serverEmpty.capabilities.Tools)
		assert.Nil(t, serverEmpty.capabilities.Resources)
	})
}

func TestWithServerInfo(t *testing.T) {
	t.Run("sets server info", func(t *testing.T) {
		// Act
		server := NewServer(WithServerInfo("my-server", "3.0.0"))

		// Assert
		assert.Equal(t, "my-server", server.serverInfo.Name)
		assert.Equal(t, "3.0.0", server.serverInfo.Version)
	})
}

func TestWithTools(t *testing.T) {
	t.Run("adds multiple tools", func(t *testing.T) {
		// Arrange
		tool1 := &mockTool{name: "tool1"}
		tool2 := &mockTool{name: "tool2"}

		// Act
		server := NewServer(WithTools(tool1, tool2))

		// Assert
		require.Len(t, server.tools, 2)
		assert.Equal(t, "tool1", server.tools[0].Name())
		assert.Equal(t, "tool2", server.tools[1].Name())
	})

	t.Run("can be called multiple times", func(t *testing.T) {
		// Act
		server := NewServer(
			WithTools(&mockTool{name: "a"}),
			WithTools(&mockTool{name: "b"}),
		)

		// Assert
		require.Len(t, server.tools, 2)
	})
}

func TestWithResources(t *testing.T) {
	t.Run("adds resources", func(t *testing.T) {
		// Arrange
		res := ResourceInfo{URI: "file:///doc.txt", Name: "Doc"}

		// Act
		server := NewServer(WithResources(res))

		// Assert
		require.Len(t, server.resources, 1)
		assert.Equal(t, "file:///doc.txt", server.resources[0].URI)
	})
}

func TestWithResourceHandler(t *testing.T) {
	t.Run("sets resource handler", func(t *testing.T) {
		// Arrange
		handler := func(ctx context.Context, uri string) (*ResourceContent, error) {
			return &ResourceContent{URI: uri, Text: "content"}, nil
		}

		// Act
		server := NewServer(WithResourceHandler(handler))

		// Assert
		assert.NotNil(t, server.resourceFn)
	})
}

func TestServer_Handle_Initialize(t *testing.T) {
	t.Run("handles initialize request", func(t *testing.T) {
		// Arrange
		server := NewServer(WithServerInfo("test-server", "1.0.0"))
		params := InitializeParams{
			ProtocolVersion: ProtocolVersion,
			ClientInfo:      Implementation{Name: "test-client", Version: "1.0.0"},
		}
		paramsJSON, _ := json.Marshal(params)
		req := &Request{
			JSONRPC: JSONRPCVersion,
			ID:      int64(1),
			Method:  "initialize",
			Params:  paramsJSON,
		}

		// Act
		resp := server.Handle(context.Background(), req)

		// Assert
		require.NotNil(t, resp)
		assert.Nil(t, resp.Error)

		var result InitializeResult
		err := json.Unmarshal(resp.Result, &result)
		require.NoError(t, err)
		assert.Equal(t, ProtocolVersion, result.ProtocolVersion)
		assert.Equal(t, "test-server", result.ServerInfo.Name)
	})

	t.Run("handles initialize with empty params", func(t *testing.T) {
		// Arrange
		server := NewServer()
		req := &Request{
			JSONRPC: JSONRPCVersion,
			ID:      int64(1),
			Method:  "initialize",
		}

		// Act
		resp := server.Handle(context.Background(), req)

		// Assert
		require.NotNil(t, resp)
		assert.Nil(t, resp.Error)
	})

	t.Run("returns error for invalid params", func(t *testing.T) {
		// Arrange
		server := NewServer()
		req := &Request{
			JSONRPC: JSONRPCVersion,
			ID:      int64(1),
			Method:  "initialize",
			Params:  json.RawMessage(`{invalid json`),
		}

		// Act
		resp := server.Handle(context.Background(), req)

		// Assert
		require.NotNil(t, resp)
		require.NotNil(t, resp.Error)
		assert.Equal(t, InvalidParams, resp.Error.Code)
	})
}

func TestServer_Handle_Initialized(t *testing.T) {
	t.Run("handles initialized notification", func(t *testing.T) {
		// Arrange
		server := NewServer()
		req := &Request{
			JSONRPC: JSONRPCVersion,
			Method:  "notifications/initialized",
		}

		// Act
		resp := server.Handle(context.Background(), req)

		// Assert
		assert.Nil(t, resp) // Notifications don't get responses
		assert.True(t, server.initialized)
	})
}

func TestServer_Handle_ToolsList(t *testing.T) {
	t.Run("returns list of tools", func(t *testing.T) {
		// Arrange
		tools := []tool.Tool{
			&mockTool{name: "tool1", desc: "First tool", params: json.RawMessage(`{"type":"object"}`)},
			&mockTool{name: "tool2", desc: "Second tool"},
		}
		server := NewServer(WithTools(tools...))
		req := &Request{
			JSONRPC: JSONRPCVersion,
			ID:      int64(1),
			Method:  "tools/list",
		}

		// Act
		resp := server.Handle(context.Background(), req)

		// Assert
		require.NotNil(t, resp)
		assert.Nil(t, resp.Error)

		var result ListToolsResult
		err := json.Unmarshal(resp.Result, &result)
		require.NoError(t, err)
		require.Len(t, result.Tools, 2)
		assert.Equal(t, "tool1", result.Tools[0].Name)
		assert.Equal(t, "First tool", result.Tools[0].Description)
	})

	t.Run("returns empty list when no tools", func(t *testing.T) {
		// Arrange
		server := NewServer()
		req := &Request{
			JSONRPC: JSONRPCVersion,
			ID:      int64(1),
			Method:  "tools/list",
		}

		// Act
		resp := server.Handle(context.Background(), req)

		// Assert
		require.NotNil(t, resp)
		var result ListToolsResult
		err := json.Unmarshal(resp.Result, &result)
		require.NoError(t, err)
		assert.Empty(t, result.Tools)
	})
}

func TestServer_Handle_ToolsCall(t *testing.T) {
	t.Run("invokes tool successfully", func(t *testing.T) {
		// Arrange
		testTool := &mockTool{
			name:   "greet",
			result: tool.NewResult("Hello, World!"),
		}
		server := NewServer(WithTools(testTool))

		params := CallToolParams{Name: "greet", Arguments: json.RawMessage(`{"name":"World"}`)}
		paramsJSON, _ := json.Marshal(params)
		req := &Request{
			JSONRPC: JSONRPCVersion,
			ID:      int64(1),
			Method:  "tools/call",
			Params:  paramsJSON,
		}

		// Act
		resp := server.Handle(context.Background(), req)

		// Assert
		require.NotNil(t, resp)
		assert.Nil(t, resp.Error)

		var result CallToolResult
		err := json.Unmarshal(resp.Result, &result)
		require.NoError(t, err)
		require.Len(t, result.Content, 1)
		assert.Equal(t, "Hello, World!", result.Content[0].Text)
		assert.False(t, result.IsError)
	})

	t.Run("handles tool invocation error", func(t *testing.T) {
		// Arrange
		testTool := &mockTool{
			name: "failing",
			err:  errors.New("tool failed"),
		}
		server := NewServer(WithTools(testTool))

		params := CallToolParams{Name: "failing"}
		paramsJSON, _ := json.Marshal(params)
		req := &Request{
			JSONRPC: JSONRPCVersion,
			ID:      int64(1),
			Method:  "tools/call",
			Params:  paramsJSON,
		}

		// Act
		resp := server.Handle(context.Background(), req)

		// Assert
		require.NotNil(t, resp)
		assert.Nil(t, resp.Error) // Not a protocol error

		var result CallToolResult
		err := json.Unmarshal(resp.Result, &result)
		require.NoError(t, err)
		assert.True(t, result.IsError)
		assert.Contains(t, result.Content[0].Text, "tool failed")
	})

	t.Run("returns error for unknown tool", func(t *testing.T) {
		// Arrange
		server := NewServer()
		params := CallToolParams{Name: "nonexistent"}
		paramsJSON, _ := json.Marshal(params)
		req := &Request{
			JSONRPC: JSONRPCVersion,
			ID:      int64(1),
			Method:  "tools/call",
			Params:  paramsJSON,
		}

		// Act
		resp := server.Handle(context.Background(), req)

		// Assert
		require.NotNil(t, resp)
		require.NotNil(t, resp.Error)
		assert.Equal(t, InvalidParams, resp.Error.Code)
		assert.Contains(t, resp.Error.Message, "not found")
	})

	t.Run("returns error for invalid params", func(t *testing.T) {
		// Arrange
		server := NewServer()
		req := &Request{
			JSONRPC: JSONRPCVersion,
			ID:      int64(1),
			Method:  "tools/call",
			Params:  json.RawMessage(`{invalid`),
		}

		// Act
		resp := server.Handle(context.Background(), req)

		// Assert
		require.NotNil(t, resp)
		require.NotNil(t, resp.Error)
		assert.Equal(t, InvalidParams, resp.Error.Code)
	})
}

func TestServer_Handle_ResourcesList(t *testing.T) {
	t.Run("returns list of resources", func(t *testing.T) {
		// Arrange
		resources := []ResourceInfo{
			{URI: "file:///a.txt", Name: "File A"},
			{URI: "file:///b.txt", Name: "File B", Description: "Second file"},
		}
		server := NewServer(WithResources(resources...))
		req := &Request{
			JSONRPC: JSONRPCVersion,
			ID:      int64(1),
			Method:  "resources/list",
		}

		// Act
		resp := server.Handle(context.Background(), req)

		// Assert
		require.NotNil(t, resp)
		assert.Nil(t, resp.Error)

		var result ListResourcesResult
		err := json.Unmarshal(resp.Result, &result)
		require.NoError(t, err)
		require.Len(t, result.Resources, 2)
	})
}

func TestServer_Handle_ResourcesRead(t *testing.T) {
	t.Run("reads resource via handler", func(t *testing.T) {
		// Arrange
		handler := func(ctx context.Context, uri string) (*ResourceContent, error) {
			if uri == "file:///test.txt" {
				return &ResourceContent{
					URI:      uri,
					MimeType: "text/plain",
					Text:     "file content",
				}, nil
			}
			return nil, nil
		}
		server := NewServer(WithResourceHandler(handler))

		params := ReadResourceParams{URI: "file:///test.txt"}
		paramsJSON, _ := json.Marshal(params)
		req := &Request{
			JSONRPC: JSONRPCVersion,
			ID:      int64(1),
			Method:  "resources/read",
			Params:  paramsJSON,
		}

		// Act
		resp := server.Handle(context.Background(), req)

		// Assert
		require.NotNil(t, resp)
		assert.Nil(t, resp.Error)

		var result ReadResourceResult
		err := json.Unmarshal(resp.Result, &result)
		require.NoError(t, err)
		require.Len(t, result.Contents, 1)
		assert.Equal(t, "file content", result.Contents[0].Text)
	})

	t.Run("reads static resource", func(t *testing.T) {
		// Arrange
		resources := []ResourceInfo{
			{URI: "file:///static.txt", Name: "Static", MimeType: "text/plain"},
		}
		server := NewServer(WithResources(resources...))

		params := ReadResourceParams{URI: "file:///static.txt"}
		paramsJSON, _ := json.Marshal(params)
		req := &Request{
			JSONRPC: JSONRPCVersion,
			ID:      int64(1),
			Method:  "resources/read",
			Params:  paramsJSON,
		}

		// Act
		resp := server.Handle(context.Background(), req)

		// Assert
		require.NotNil(t, resp)
		assert.Nil(t, resp.Error)

		var result ReadResourceResult
		err := json.Unmarshal(resp.Result, &result)
		require.NoError(t, err)
		require.Len(t, result.Contents, 1)
		assert.Equal(t, "file:///static.txt", result.Contents[0].URI)
	})

	t.Run("returns error for unknown resource", func(t *testing.T) {
		// Arrange
		server := NewServer()
		params := ReadResourceParams{URI: "file:///missing.txt"}
		paramsJSON, _ := json.Marshal(params)
		req := &Request{
			JSONRPC: JSONRPCVersion,
			ID:      int64(1),
			Method:  "resources/read",
			Params:  paramsJSON,
		}

		// Act
		resp := server.Handle(context.Background(), req)

		// Assert
		require.NotNil(t, resp)
		require.NotNil(t, resp.Error)
		assert.Equal(t, InvalidParams, resp.Error.Code)
		assert.Contains(t, resp.Error.Message, "not found")
	})

	t.Run("returns error from handler", func(t *testing.T) {
		// Arrange
		handler := func(ctx context.Context, uri string) (*ResourceContent, error) {
			return nil, errors.New("read failed")
		}
		server := NewServer(WithResourceHandler(handler))

		params := ReadResourceParams{URI: "file:///error.txt"}
		paramsJSON, _ := json.Marshal(params)
		req := &Request{
			JSONRPC: JSONRPCVersion,
			ID:      int64(1),
			Method:  "resources/read",
			Params:  paramsJSON,
		}

		// Act
		resp := server.Handle(context.Background(), req)

		// Assert
		require.NotNil(t, resp)
		require.NotNil(t, resp.Error)
		assert.Equal(t, InternalError, resp.Error.Code)
	})
}

func TestServer_Handle_UnknownMethod(t *testing.T) {
	t.Run("returns MethodNotFound for unknown method", func(t *testing.T) {
		// Arrange
		server := NewServer()
		req := &Request{
			JSONRPC: JSONRPCVersion,
			ID:      int64(1),
			Method:  "unknown/method",
		}

		// Act
		resp := server.Handle(context.Background(), req)

		// Assert
		require.NotNil(t, resp)
		require.NotNil(t, resp.Error)
		assert.Equal(t, MethodNotFound, resp.Error.Code)
		assert.Contains(t, resp.Error.Message, "unknown/method")
	})
}

func TestServer_HTTPHandler(t *testing.T) {
	t.Run("returns valid http.Handler", func(t *testing.T) {
		// Arrange
		server := NewServer()

		// Act
		handler := server.HTTPHandler()

		// Assert
		assert.NotNil(t, handler)
		assert.Implements(t, (*http.Handler)(nil), handler)
	})

	t.Run("handles POST request", func(t *testing.T) {
		// Arrange
		server := NewServer(WithServerInfo("http-server", "1.0.0"))
		handler := server.HTTPHandler()

		params := InitializeParams{ProtocolVersion: ProtocolVersion}
		paramsJSON, _ := json.Marshal(params)
		req := Request{
			JSONRPC: JSONRPCVersion,
			ID:      int64(1),
			Method:  "initialize",
			Params:  paramsJSON,
		}
		reqBody, _ := json.Marshal(req)

		httpReq := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(string(reqBody)))
		httpReq.Header.Set("Content-Type", "application/json")
		recorder := httptest.NewRecorder()

		// Act
		handler.ServeHTTP(recorder, httpReq)

		// Assert
		assert.Equal(t, http.StatusOK, recorder.Code)
		assert.Equal(t, "application/json", recorder.Header().Get("Content-Type"))

		var resp Response
		err := json.Unmarshal(recorder.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Nil(t, resp.Error)
	})

	t.Run("rejects non-POST requests", func(t *testing.T) {
		// Arrange
		server := NewServer()
		handler := server.HTTPHandler()

		httpReq := httptest.NewRequest(http.MethodGet, "/", nil)
		recorder := httptest.NewRecorder()

		// Act
		handler.ServeHTTP(recorder, httpReq)

		// Assert
		assert.Equal(t, http.StatusMethodNotAllowed, recorder.Code)
	})

	t.Run("handles invalid JSON", func(t *testing.T) {
		// Arrange
		server := NewServer()
		handler := server.HTTPHandler()

		httpReq := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{invalid`))
		recorder := httptest.NewRecorder()

		// Act
		handler.ServeHTTP(recorder, httpReq)

		// Assert
		var resp Response
		err := json.Unmarshal(recorder.Body.Bytes(), &resp)
		require.NoError(t, err)
		require.NotNil(t, resp.Error)
		assert.Equal(t, ParseError, resp.Error.Code)
	})

	t.Run("returns 204 for notifications", func(t *testing.T) {
		// Arrange
		server := NewServer()
		handler := server.HTTPHandler()

		req := Request{
			JSONRPC: JSONRPCVersion,
			Method:  "notifications/initialized",
		}
		reqBody, _ := json.Marshal(req)

		httpReq := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(string(reqBody)))
		recorder := httptest.NewRecorder()

		// Act
		handler.ServeHTTP(recorder, httpReq)

		// Assert
		assert.Equal(t, http.StatusNoContent, recorder.Code)
	})

	t.Run("handles read error", func(t *testing.T) {
		// Arrange
		server := NewServer()
		handler := server.HTTPHandler()

		// Create a request with a body that fails to read
		httpReq := httptest.NewRequest(http.MethodPost, "/", &errorReader{})
		recorder := httptest.NewRecorder()

		// Act
		handler.ServeHTTP(recorder, httpReq)

		// Assert
		var resp Response
		err := json.Unmarshal(recorder.Body.Bytes(), &resp)
		require.NoError(t, err)
		require.NotNil(t, resp.Error)
		assert.Equal(t, ParseError, resp.Error.Code)
	})
}

// errorReader is an io.Reader that always returns an error.
type errorReader struct{}

func (e *errorReader) Read(p []byte) (n int, err error) {
	return 0, io.ErrUnexpectedEOF
}

func TestServer_AddTool(t *testing.T) {
	t.Run("adds tool at runtime", func(t *testing.T) {
		// Arrange
		server := NewServer()
		assert.Empty(t, server.tools)
		assert.Nil(t, server.capabilities.Tools)

		newTool := &mockTool{name: "dynamic"}

		// Act
		server.AddTool(newTool)

		// Assert
		require.Len(t, server.tools, 1)
		assert.Equal(t, "dynamic", server.tools[0].Name())
		assert.NotNil(t, server.capabilities.Tools)
	})
}

func TestServer_AddResource(t *testing.T) {
	t.Run("adds resource at runtime", func(t *testing.T) {
		// Arrange
		server := NewServer()
		assert.Empty(t, server.resources)
		assert.Nil(t, server.capabilities.Resources)

		newResource := ResourceInfo{URI: "file:///new.txt", Name: "New"}

		// Act
		server.AddResource(newResource)

		// Assert
		require.Len(t, server.resources, 1)
		assert.Equal(t, "file:///new.txt", server.resources[0].URI)
		assert.NotNil(t, server.capabilities.Resources)
	})
}

func TestServer_SuccessResponse(t *testing.T) {
	t.Run("creates success response", func(t *testing.T) {
		// Arrange
		server := NewServer()
		result := map[string]string{"key": "value"}

		// Act
		resp := server.successResponse(int64(42), result)

		// Assert
		assert.Equal(t, JSONRPCVersion, resp.JSONRPC)
		assert.Equal(t, int64(42), resp.ID)
		assert.Nil(t, resp.Error)
		assert.Contains(t, string(resp.Result), `"key":"value"`)
	})
}

func TestServer_ErrorResponse(t *testing.T) {
	t.Run("creates error response", func(t *testing.T) {
		// Arrange
		server := NewServer()

		// Act
		resp := server.errorResponse(int64(1), MethodNotFound, "Method not found", nil)

		// Assert
		assert.Equal(t, JSONRPCVersion, resp.JSONRPC)
		assert.Equal(t, int64(1), resp.ID)
		require.NotNil(t, resp.Error)
		assert.Equal(t, MethodNotFound, resp.Error.Code)
		assert.Equal(t, "Method not found", resp.Error.Message)
	})

	t.Run("creates error response with data", func(t *testing.T) {
		// Arrange
		server := NewServer()
		data := map[string]string{"detail": "extra info"}

		// Act
		resp := server.errorResponse(int64(2), InternalError, "Error", data)

		// Assert
		require.NotNil(t, resp.Error)
		assert.Contains(t, string(resp.Error.Data), "extra info")
	})
}
