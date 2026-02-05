// Copyright (c) Microsoft. All rights reserved.

package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockTransport is a test double for the Transport interface.
type mockTransport struct {
	startErr   error
	sendResp   *Response
	sendErr    error
	receiveErr error
	closeErr   error
	closed     bool
	sendCalls  []*Request
}

func (m *mockTransport) Start(ctx context.Context) error {
	return m.startErr
}

func (m *mockTransport) Send(ctx context.Context, req *Request) (*Response, error) {
	m.sendCalls = append(m.sendCalls, req)
	return m.sendResp, m.sendErr
}

func (m *mockTransport) Receive(ctx context.Context) (*Notification, error) {
	if m.receiveErr != nil {
		return nil, m.receiveErr
	}
	return nil, io.EOF
}

func (m *mockTransport) Close() error {
	m.closed = true
	return m.closeErr
}

// createSuccessfulInitResponse creates a mock initialize response.
func createSuccessfulInitResponse() *Response {
	result := InitializeResult{
		ProtocolVersion: ProtocolVersion,
		Capabilities:    ServerCapabilities{},
		ServerInfo: Implementation{
			Name:    "test-server",
			Version: "1.0.0",
		},
	}
	resultJSON, _ := json.Marshal(result)
	return &Response{
		JSONRPC: JSONRPCVersion,
		ID:      int64(1),
		Result:  resultJSON,
	}
}

// createToolsListResponse creates a mock tools/list response.
func createToolsListResponse(tools []ToolInfo) *Response {
	result := ListToolsResult{Tools: tools}
	resultJSON, _ := json.Marshal(result)
	return &Response{
		JSONRPC: JSONRPCVersion,
		ID:      int64(2),
		Result:  resultJSON,
	}
}

// createCallToolResponse creates a mock tools/call response.
func createCallToolResponse(content string, isError bool) *Response {
	result := CallToolResult{
		Content: []Content{NewTextContent(content)},
		IsError: isError,
	}
	resultJSON, _ := json.Marshal(result)
	return &Response{
		JSONRPC: JSONRPCVersion,
		ID:      int64(3),
		Result:  resultJSON,
	}
}

func TestNewClient_Success(t *testing.T) {
	t.Run("succeeds with valid transport", func(t *testing.T) {
		// Arrange
		mock := &mockTransport{
			sendResp: createSuccessfulInitResponse(),
		}

		// Act
		client, err := NewClient(context.Background(), mock)

		// Assert
		require.NoError(t, err)
		require.NotNil(t, client)
		assert.NotNil(t, client.ServerInfo())
		assert.Equal(t, "test-server", client.ServerInfo().Name)
	})

	t.Run("applies client options", func(t *testing.T) {
		// Arrange
		mock := &mockTransport{
			sendResp: createSuccessfulInitResponse(),
		}

		// Act
		client, err := NewClient(
			context.Background(),
			mock,
			WithClientInfo("custom-client", "2.0.0"),
		)

		// Assert
		require.NoError(t, err)
		require.NotNil(t, client)

		// Verify the client info was sent in the initialize request
		require.GreaterOrEqual(t, len(mock.sendCalls), 1)
		var params InitializeParams
		err = json.Unmarshal(mock.sendCalls[0].Params, &params)
		require.NoError(t, err)
		assert.Equal(t, "custom-client", params.ClientInfo.Name)
		assert.Equal(t, "2.0.0", params.ClientInfo.Version)
	})
}

func TestNewClient_Failures(t *testing.T) {
	t.Run("fails if Start fails", func(t *testing.T) {
		// Arrange
		mock := &mockTransport{
			startErr: errors.New("connection failed"),
		}

		// Act
		client, err := NewClient(context.Background(), mock)

		// Assert
		assert.Nil(t, client)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "start")
	})

	t.Run("fails if initialize Send fails", func(t *testing.T) {
		// Arrange
		mock := &mockTransport{
			sendErr: errors.New("send failed"),
		}

		// Act
		client, err := NewClient(context.Background(), mock)

		// Assert
		assert.Nil(t, client)
		assert.Error(t, err)
		assert.True(t, mock.closed, "transport should be closed on failure")
	})

	t.Run("fails if server returns error response", func(t *testing.T) {
		// Arrange
		mock := &mockTransport{
			sendResp: &Response{
				JSONRPC: JSONRPCVersion,
				ID:      int64(1),
				Error: &ResponseError{
					Code:    InternalError,
					Message: "server error",
				},
			},
		}

		// Act
		client, err := NewClient(context.Background(), mock)

		// Assert
		assert.Nil(t, client)
		assert.Error(t, err)
		assert.True(t, mock.closed)
	})

	t.Run("fails if protocol version mismatch", func(t *testing.T) {
		// Arrange
		result := InitializeResult{
			ProtocolVersion: "1999-01-01", // Wrong version
			ServerInfo: Implementation{
				Name:    "old-server",
				Version: "0.1.0",
			},
		}
		resultJSON, _ := json.Marshal(result)
		mock := &mockTransport{
			sendResp: &Response{
				JSONRPC: JSONRPCVersion,
				ID:      int64(1),
				Result:  resultJSON,
			},
		}

		// Act
		client, err := NewClient(context.Background(), mock)

		// Assert
		assert.Nil(t, client)
		assert.Error(t, err)
		assert.True(t, errors.Is(err, ErrProtocolVersion))
		assert.True(t, mock.closed)
	})

	t.Run("fails if result is invalid JSON", func(t *testing.T) {
		// Arrange
		mock := &mockTransport{
			sendResp: &Response{
				JSONRPC: JSONRPCVersion,
				ID:      int64(1),
				Result:  json.RawMessage(`{invalid json}`),
			},
		}

		// Act
		client, err := NewClient(context.Background(), mock)

		// Assert
		assert.Nil(t, client)
		assert.Error(t, err)
		assert.True(t, mock.closed)
	})
}

func TestWithClientInfo(t *testing.T) {
	t.Run("sets custom client info", func(t *testing.T) {
		// Arrange
		cfg := defaultClientConfig()

		// Act
		WithClientInfo("my-app", "3.0.0")(cfg)

		// Assert
		assert.Equal(t, "my-app", cfg.clientInfo.Name)
		assert.Equal(t, "3.0.0", cfg.clientInfo.Version)
	})
}

func TestClient_ListTools(t *testing.T) {
	t.Run("returns tools from server", func(t *testing.T) {
		// Arrange
		expectedTools := []ToolInfo{
			{Name: "tool1", Description: "First tool"},
			{Name: "tool2", Description: "Second tool"},
		}
		callCount := 0
		mock := &mockTransport{}
		mock.sendResp = createSuccessfulInitResponse()

		client, err := NewClient(context.Background(), mock)
		require.NoError(t, err)

		// Setup for ListTools call
		callCount = len(mock.sendCalls)
		mock.sendResp = createToolsListResponse(expectedTools)

		// Act
		tools, err := client.ListTools(context.Background())

		// Assert
		require.NoError(t, err)
		require.Len(t, tools, 2)
		assert.Equal(t, "tool1", tools[0].Name)
		assert.Equal(t, "tool2", tools[1].Name)

		// Verify the request was sent correctly
		require.Greater(t, len(mock.sendCalls), callCount)
		lastReq := mock.sendCalls[len(mock.sendCalls)-1]
		assert.Equal(t, "tools/list", lastReq.Method)
	})

	t.Run("fails when not initialized", func(t *testing.T) {
		// Arrange
		client := &Client{}

		// Act
		tools, err := client.ListTools(context.Background())

		// Assert
		assert.Nil(t, tools)
		assert.ErrorIs(t, err, ErrNotInitialized)
	})

	t.Run("fails when closed", func(t *testing.T) {
		// Arrange
		mock := &mockTransport{
			sendResp: createSuccessfulInitResponse(),
		}
		client, err := NewClient(context.Background(), mock)
		require.NoError(t, err)
		client.Close()

		// Act
		tools, err := client.ListTools(context.Background())

		// Assert
		assert.Nil(t, tools)
		assert.ErrorIs(t, err, ErrClosed)
	})
}

func TestClient_CallTool(t *testing.T) {
	t.Run("invokes tool and returns result", func(t *testing.T) {
		// Arrange
		mock := &mockTransport{
			sendResp: createSuccessfulInitResponse(),
		}
		client, err := NewClient(context.Background(), mock)
		require.NoError(t, err)

		mock.sendResp = createCallToolResponse("success result", false)

		// Act
		result, err := client.CallTool(
			context.Background(),
			"test_tool",
			json.RawMessage(`{"arg":"value"}`),
		)

		// Assert
		require.NoError(t, err)
		require.NotNil(t, result)
		require.Len(t, result.Content, 1)
		assert.Equal(t, "success result", result.Content[0].Text)
		assert.False(t, result.IsError)
	})

	t.Run("returns error result from tool", func(t *testing.T) {
		// Arrange
		mock := &mockTransport{
			sendResp: createSuccessfulInitResponse(),
		}
		client, err := NewClient(context.Background(), mock)
		require.NoError(t, err)

		mock.sendResp = createCallToolResponse("tool error", true)

		// Act
		result, err := client.CallTool(
			context.Background(),
			"failing_tool",
			nil,
		)

		// Assert
		require.NoError(t, err) // The call succeeded, tool returned error
		require.NotNil(t, result)
		assert.True(t, result.IsError)
		assert.Equal(t, "tool error", result.Content[0].Text)
	})

	t.Run("fails on transport error", func(t *testing.T) {
		// Arrange
		mock := &mockTransport{
			sendResp: createSuccessfulInitResponse(),
		}
		client, err := NewClient(context.Background(), mock)
		require.NoError(t, err)

		mock.sendResp = nil
		mock.sendErr = errors.New("network error")

		// Act
		result, err := client.CallTool(context.Background(), "tool", nil)

		// Assert
		assert.Nil(t, result)
		assert.Error(t, err)
	})

	t.Run("fails when not initialized", func(t *testing.T) {
		// Arrange
		client := &Client{}

		// Act
		result, err := client.CallTool(context.Background(), "tool", nil)

		// Assert
		assert.Nil(t, result)
		assert.ErrorIs(t, err, ErrNotInitialized)
	})
}

func TestClient_ListResources(t *testing.T) {
	t.Run("returns resources from server", func(t *testing.T) {
		// Arrange
		mock := &mockTransport{
			sendResp: createSuccessfulInitResponse(),
		}
		client, err := NewClient(context.Background(), mock)
		require.NoError(t, err)

		resources := []ResourceInfo{
			{URI: "file:///a.txt", Name: "A"},
			{URI: "file:///b.txt", Name: "B"},
		}
		result := ListResourcesResult{Resources: resources}
		resultJSON, _ := json.Marshal(result)
		mock.sendResp = &Response{
			JSONRPC: JSONRPCVersion,
			ID:      int64(2),
			Result:  resultJSON,
		}

		// Act
		res, err := client.ListResources(context.Background())

		// Assert
		require.NoError(t, err)
		require.Len(t, res, 2)
		assert.Equal(t, "file:///a.txt", res[0].URI)
	})
}

func TestClient_ReadResource(t *testing.T) {
	t.Run("returns resource content", func(t *testing.T) {
		// Arrange
		mock := &mockTransport{
			sendResp: createSuccessfulInitResponse(),
		}
		client, err := NewClient(context.Background(), mock)
		require.NoError(t, err)

		content := ResourceContent{
			URI:      "file:///test.txt",
			MimeType: "text/plain",
			Text:     "file contents",
		}
		result := ReadResourceResult{Contents: []ResourceContent{content}}
		resultJSON, _ := json.Marshal(result)
		mock.sendResp = &Response{
			JSONRPC: JSONRPCVersion,
			ID:      int64(2),
			Result:  resultJSON,
		}

		// Act
		res, err := client.ReadResource(context.Background(), "file:///test.txt")

		// Assert
		require.NoError(t, err)
		assert.Equal(t, "file:///test.txt", res.URI)
		assert.Equal(t, "file contents", res.Text)
	})

	t.Run("returns error for empty contents", func(t *testing.T) {
		// Arrange
		mock := &mockTransport{
			sendResp: createSuccessfulInitResponse(),
		}
		client, err := NewClient(context.Background(), mock)
		require.NoError(t, err)

		result := ReadResourceResult{Contents: []ResourceContent{}}
		resultJSON, _ := json.Marshal(result)
		mock.sendResp = &Response{
			JSONRPC: JSONRPCVersion,
			ID:      int64(2),
			Result:  resultJSON,
		}

		// Act
		res, err := client.ReadResource(context.Background(), "missing")

		// Assert
		assert.Nil(t, res)
		assert.Error(t, err)
	})
}

func TestClient_ServerInfo(t *testing.T) {
	t.Run("returns server info after initialization", func(t *testing.T) {
		// Arrange
		mock := &mockTransport{
			sendResp: createSuccessfulInitResponse(),
		}
		client, err := NewClient(context.Background(), mock)
		require.NoError(t, err)

		// Act
		info := client.ServerInfo()

		// Assert
		require.NotNil(t, info)
		assert.Equal(t, "test-server", info.Name)
		assert.Equal(t, "1.0.0", info.Version)
	})

	t.Run("returns nil before initialization", func(t *testing.T) {
		// Arrange
		client := &Client{}

		// Act
		info := client.ServerInfo()

		// Assert
		assert.Nil(t, info)
	})
}

func TestClient_ServerCapabilities(t *testing.T) {
	t.Run("returns capabilities after initialization", func(t *testing.T) {
		// Arrange
		result := InitializeResult{
			ProtocolVersion: ProtocolVersion,
			Capabilities: ServerCapabilities{
				Tools: &ToolsCapability{ListChanged: true},
			},
			ServerInfo: Implementation{Name: "server", Version: "1.0.0"},
		}
		resultJSON, _ := json.Marshal(result)
		mock := &mockTransport{
			sendResp: &Response{
				JSONRPC: JSONRPCVersion,
				ID:      int64(1),
				Result:  resultJSON,
			},
		}
		client, err := NewClient(context.Background(), mock)
		require.NoError(t, err)

		// Act
		caps := client.ServerCapabilities()

		// Assert
		require.NotNil(t, caps)
		require.NotNil(t, caps.Tools)
		assert.True(t, caps.Tools.ListChanged)
	})
}

func TestClient_Close(t *testing.T) {
	t.Run("closes transport", func(t *testing.T) {
		// Arrange
		mock := &mockTransport{
			sendResp: createSuccessfulInitResponse(),
		}
		client, err := NewClient(context.Background(), mock)
		require.NoError(t, err)

		// Act
		err = client.Close()

		// Assert
		assert.NoError(t, err)
		assert.True(t, mock.closed)
	})

	t.Run("subsequent calls are no-op", func(t *testing.T) {
		// Arrange
		mock := &mockTransport{
			sendResp: createSuccessfulInitResponse(),
		}
		client, err := NewClient(context.Background(), mock)
		require.NoError(t, err)

		// Act
		err1 := client.Close()
		err2 := client.Close()

		// Assert
		assert.NoError(t, err1)
		assert.NoError(t, err2)
	})

	t.Run("returns transport close error", func(t *testing.T) {
		// Arrange
		mock := &mockTransport{
			sendResp: createSuccessfulInitResponse(),
			closeErr: errors.New("close failed"),
		}
		client, err := NewClient(context.Background(), mock)
		require.NoError(t, err)

		// Act
		err = client.Close()

		// Assert
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "close failed")
	})
}

func TestClient_CheckInitialized(t *testing.T) {
	t.Run("returns ErrNotInitialized when not initialized", func(t *testing.T) {
		// Arrange
		client := &Client{}

		// Act
		err := client.checkInitialized()

		// Assert
		assert.ErrorIs(t, err, ErrNotInitialized)
	})

	t.Run("returns ErrClosed when closed", func(t *testing.T) {
		// Arrange
		client := &Client{
			initialized: true,
			closed:      true,
		}

		// Act
		err := client.checkInitialized()

		// Assert
		assert.ErrorIs(t, err, ErrClosed)
	})

	t.Run("returns nil when initialized and not closed", func(t *testing.T) {
		// Arrange
		client := &Client{
			initialized: true,
			closed:      false,
		}

		// Act
		err := client.checkInitialized()

		// Assert
		assert.NoError(t, err)
	})
}
