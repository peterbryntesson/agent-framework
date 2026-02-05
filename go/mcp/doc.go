// Copyright (c) Microsoft. All rights reserved.

/*
Package mcp provides client and server implementations for the Model Context
Protocol (MCP), enabling agents to use external tools and expose capabilities.

MCP defines a standard way for AI agents to:
  - Connect to MCP servers that provide tools and resources
  - Invoke tools hosted by MCP servers
  - Access resources (files, data) from MCP servers
  - Expose agent capabilities as an MCP server

# Protocol Overview

MCP uses JSON-RPC 2.0 over various transports (stdio, HTTP with SSE) to enable
communication between clients and servers. The protocol supports tool discovery,
invocation, resource access, and capability negotiation.

# Key Types

The package provides these primary abstractions:

  - [Client]: Connects to MCP servers to invoke tools and access resources
  - [Server]: Exposes agent capabilities as an MCP server
  - [Transport]: Abstraction for communication (stdio, HTTP/SSE)

# Client Usage

Connect to an MCP server using stdio transport (e.g., spawning a process):

	transport := mcp.NewStdioTransport("npx", "-y", "@modelcontextprotocol/server-filesystem", "/tmp")
	client, err := mcp.NewClient(transport)
	if err != nil {
	    return err
	}
	defer client.Close()

	// List available tools
	tools, err := client.ListTools(ctx)
	if err != nil {
	    return err
	}
	for _, tool := range tools {
	    fmt.Printf("Tool: %s - %s\n", tool.Name, tool.Description)
	}

	// Call a tool
	result, err := client.CallTool(ctx, "read_file", json.RawMessage(`{"path": "/tmp/example.txt"}`))
	if err != nil {
	    return err
	}
	fmt.Println(result.Content[0].Text)

Connect to an MCP server using HTTP transport:

	transport := mcp.NewHTTPTransport("https://mcp.example.com/sse")
	client, err := mcp.NewClient(transport,
	    mcp.WithClientInfo("my-agent", "1.0.0"),
	)
	if err != nil {
	    return err
	}
	defer client.Close()

	// Use client as above...

# Wrapping MCP Tools

Convert MCP tools to agent tools for use with the agent framework:

	adapter := mcp.NewToolAdapter(client)
	tools := adapter.Tools() // Returns []tool.Tool

	agent := chatagent.New(chatClient,
	    chatagent.WithTools(tools...),
	)

# Server Usage

Expose agent capabilities as an MCP server:

	server := mcp.NewServer(
	    mcp.WithServerInfo("my-server", "1.0.0"),
	    mcp.WithTools(myTools...),
	    mcp.WithResources(myResources...),
	)

	// Serve via stdio (for CLI tool usage)
	if err := server.ServeStdio(ctx); err != nil {
	    return err
	}

	// Or serve via HTTP
	http.Handle("/mcp", server.HTTPHandler())

# Protocol Version

This implementation supports MCP protocol version 2024-11-05.

See https://modelcontextprotocol.io/docs for the full protocol specification.
*/
package mcp
