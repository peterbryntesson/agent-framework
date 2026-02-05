// Copyright (c) Microsoft. All rights reserved.

/*
Package devui provides a development UI server for testing and debugging AI agents.

The devui package enables developers to interactively test agents during development,
visualize conversation traces, and inspect agent behavior without building a full
application. It provides a web-based interface for agent interaction and OpenTelemetry
trace collection.

# Overview

The DevUI server provides:

  - Agent registration and discovery
  - Interactive chat interface for testing
  - OpenTelemetry trace collection and visualization
  - SSE streaming for real-time updates
  - REST API for programmatic access

# Quick Start

To start a DevUI server with registered agents:

	// Create agents
	agent1 := chatagent.New("assistant", chatClient,
	    chatagent.WithInstructions("You are a helpful assistant."))

	// Create DevUI server
	server := devui.NewServer(
	    devui.WithPort(8080),
	)

	// Register agents
	server.RegisterAgent("assistant", agent1)

	// Start server
	if err := server.Start(); err != nil {
	    log.Fatal(err)
	}

# API Endpoints

The DevUI server exposes these endpoints:

	GET  /api/agents           - List registered agents
	GET  /api/agents/{name}    - Get agent details
	POST /api/agents/{name}/run     - Run agent (non-streaming)
	POST /api/agents/{name}/stream  - Run agent (streaming via SSE)
	GET  /api/traces           - List collected traces
	GET  /api/traces/{id}      - Get trace details
	GET  /                     - Serve frontend UI

# Trace Collection

DevUI integrates with OpenTelemetry to collect and display traces:

	server := devui.NewServer(
	    devui.WithTraceCollection(true),
	    devui.WithMaxTraces(1000),
	)

Traces are stored in memory and can be queried via the API or viewed in the UI.

# Frontend Customization

The default frontend can be replaced with custom assets:

	server := devui.NewServer(
	    devui.WithFrontendFS(customFS),
	)
*/
package devui
