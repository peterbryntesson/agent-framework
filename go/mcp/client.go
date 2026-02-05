// Copyright (c) Microsoft. All rights reserved.

package mcp

import (
	"context"
	"encoding/json"
	"sync"
)

// Client connects to an MCP server and provides tool access.
// It handles the MCP protocol handshake and maintains the connection state.
// The client is safe for concurrent use.
type Client struct {
	transport   Transport
	serverInfo  *Implementation
	serverCaps  *ServerCapabilities
	clientInfo  Implementation
	mu          sync.RWMutex
	initialized bool
	closed      bool
}

// ClientOption configures a Client.
type ClientOption func(*clientConfig)

// clientConfig holds client configuration options.
type clientConfig struct {
	clientInfo Implementation
}

// defaultClientConfig returns the default client configuration.
func defaultClientConfig() *clientConfig {
	return &clientConfig{
		clientInfo: Implementation{
			Name:    "agent-framework-go",
			Version: "1.0.0",
		},
	}
}

// WithClientInfo sets the client implementation info sent during initialization.
func WithClientInfo(name, version string) ClientOption {
	return func(c *clientConfig) {
		c.clientInfo = Implementation{
			Name:    name,
			Version: version,
		}
	}
}

// NewClient creates a new MCP client with the given transport.
// It automatically starts the transport and initializes the connection
// by performing the MCP handshake with the server.
func NewClient(ctx context.Context, transport Transport, opts ...ClientOption) (*Client, error) {
	cfg := defaultClientConfig()
	for _, opt := range opts {
		opt(cfg)
	}

	client := &Client{
		transport:  transport,
		clientInfo: cfg.clientInfo,
	}

	// Start the transport
	if err := transport.Start(ctx); err != nil {
		return nil, NewMCPError("start", err)
	}

	// Perform initialization handshake
	if err := client.initialize(ctx); err != nil {
		// Clean up transport on initialization failure
		transport.Close()
		return nil, err
	}

	return client, nil
}

// initialize performs the MCP initialization handshake with the server.
func (c *Client) initialize(ctx context.Context) error {
	params := InitializeParams{
		ProtocolVersion: ProtocolVersion,
		Capabilities:    ClientCapabilities{},
		ClientInfo:      c.clientInfo,
	}

	paramsJSON, err := json.Marshal(params)
	if err != nil {
		return NewMCPError("initialize", err)
	}

	req := &Request{
		JSONRPC: JSONRPCVersion,
		ID:      nextRequestID(),
		Method:  "initialize",
		Params:  paramsJSON,
	}

	resp, err := c.transport.Send(ctx, req)
	if err != nil {
		return NewMCPError("initialize", err)
	}

	if resp.Error != nil {
		return NewMCPError("initialize", resp.Error)
	}

	var result InitializeResult
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return NewMCPError("initialize", err)
	}

	// Verify protocol version compatibility
	if result.ProtocolVersion != ProtocolVersion {
		return NewMCPError("initialize", ErrProtocolVersion)
	}

	c.mu.Lock()
	c.serverInfo = &result.ServerInfo
	c.serverCaps = &result.Capabilities
	c.initialized = true
	c.mu.Unlock()

	// Send initialized notification to complete handshake
	initNotification := &Request{
		JSONRPC: JSONRPCVersion,
		Method:  "notifications/initialized",
	}

	// Initialized is a notification, we don't wait for a response
	if _, err := c.transport.Send(ctx, initNotification); err != nil {
		// Log but don't fail - some servers may not require this
	}

	return nil
}

// checkInitialized returns an error if the client is not initialized or closed.
func (c *Client) checkInitialized() error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.closed {
		return ErrClosed
	}
	if !c.initialized {
		return ErrNotInitialized
	}
	return nil
}

// ListTools returns the available tools from the MCP server.
func (c *Client) ListTools(ctx context.Context) ([]ToolInfo, error) {
	if err := c.checkInitialized(); err != nil {
		return nil, err
	}

	req := &Request{
		JSONRPC: JSONRPCVersion,
		ID:      nextRequestID(),
		Method:  "tools/list",
	}

	resp, err := c.transport.Send(ctx, req)
	if err != nil {
		return nil, NewMCPError("tools/list", err)
	}

	if resp.Error != nil {
		return nil, NewMCPError("tools/list", resp.Error)
	}

	var result ListToolsResult
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return nil, NewMCPError("tools/list", err)
	}

	return result.Tools, nil
}

// CallTool invokes a tool on the MCP server.
func (c *Client) CallTool(ctx context.Context, name string, arguments json.RawMessage) (*CallToolResult, error) {
	if err := c.checkInitialized(); err != nil {
		return nil, err
	}

	params := CallToolParams{
		Name:      name,
		Arguments: arguments,
	}

	paramsJSON, err := json.Marshal(params)
	if err != nil {
		return nil, NewMCPError("tools/call", err)
	}

	req := &Request{
		JSONRPC: JSONRPCVersion,
		ID:      nextRequestID(),
		Method:  "tools/call",
		Params:  paramsJSON,
	}

	resp, err := c.transport.Send(ctx, req)
	if err != nil {
		return nil, NewMCPError("tools/call", err)
	}

	if resp.Error != nil {
		return nil, NewMCPError("tools/call", resp.Error)
	}

	var result CallToolResult
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return nil, NewMCPError("tools/call", err)
	}

	return &result, nil
}

// ListResources returns available resources from the MCP server.
func (c *Client) ListResources(ctx context.Context) ([]ResourceInfo, error) {
	if err := c.checkInitialized(); err != nil {
		return nil, err
	}

	req := &Request{
		JSONRPC: JSONRPCVersion,
		ID:      nextRequestID(),
		Method:  "resources/list",
	}

	resp, err := c.transport.Send(ctx, req)
	if err != nil {
		return nil, NewMCPError("resources/list", err)
	}

	if resp.Error != nil {
		return nil, NewMCPError("resources/list", resp.Error)
	}

	var result ListResourcesResult
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return nil, NewMCPError("resources/list", err)
	}

	return result.Resources, nil
}

// ReadResource reads a resource by URI from the MCP server.
func (c *Client) ReadResource(ctx context.Context, uri string) (*ResourceContent, error) {
	if err := c.checkInitialized(); err != nil {
		return nil, err
	}

	params := ReadResourceParams{
		URI: uri,
	}

	paramsJSON, err := json.Marshal(params)
	if err != nil {
		return nil, NewMCPError("resources/read", err)
	}

	req := &Request{
		JSONRPC: JSONRPCVersion,
		ID:      nextRequestID(),
		Method:  "resources/read",
		Params:  paramsJSON,
	}

	resp, err := c.transport.Send(ctx, req)
	if err != nil {
		return nil, NewMCPError("resources/read", err)
	}

	if resp.Error != nil {
		return nil, NewMCPError("resources/read", resp.Error)
	}

	var result ReadResourceResult
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return nil, NewMCPError("resources/read", err)
	}

	if len(result.Contents) == 0 {
		return nil, NewMCPError("resources/read", ErrResourceNotFound)
	}

	return &result.Contents[0], nil
}

// ServerInfo returns the server's implementation info.
// Returns nil if the client is not initialized.
func (c *Client) ServerInfo() *Implementation {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.serverInfo
}

// ServerCapabilities returns the server's capabilities.
// Returns nil if the client is not initialized.
func (c *Client) ServerCapabilities() *ServerCapabilities {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.serverCaps
}

// Close closes the client and transport.
// After Close is called, all operations will return ErrClosed.
func (c *Client) Close() error {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return nil
	}
	c.closed = true
	c.mu.Unlock()

	return c.transport.Close()
}
