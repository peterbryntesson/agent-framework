// Copyright (c) Microsoft. All rights reserved.

/*
Package durable provides durable agent sessions using Temporal.io workflows.

Durable agents enable persistent conversation sessions that survive process
restarts and can be distributed across multiple workers. Sessions maintain
conversation history in workflow state, enabling long-running agent interactions
with guaranteed durability and fault tolerance.

# Overview

The package provides three main components:

  - [Agent]: Wraps a standard agent with durable session management
  - [Worker]: Runs Temporal workflows and activities for durable agents
  - [Session]: Manages conversation state within a durable workflow

# State Format

Durable agent state follows the cross-platform schema defined in
schemas/durable-agent-entity-state.json. The state structure is:

	{
	    "schemaVersion": "1.1.0",
	    "data": {
	        "conversationHistory": [...],
	        "expirationTimeUtc": "2024-12-31T23:59:59Z"
	    }
	}

This format is compatible with .NET and Python implementations, enabling
cross-platform session migration.

# Basic Usage

Create a durable agent wrapper:

	// Create a Temporal client
	temporalClient, err := client.Dial(client.Options{
	    HostPort: "localhost:7233",
	})
	if err != nil {
	    log.Fatal(err)
	}
	defer temporalClient.Close()

	// Wrap your agent with durable session management
	durableAgent := durable.NewAgent(myAgent, temporalClient,
	    durable.WithTaskQueue("my-agent-queue"),
	    durable.WithName("my-agent"),
	)

	// Run the agent - session is managed automatically
	response, err := durableAgent.Run(ctx, messages,
	    durable.WithSessionID("user-123"),
	)

# Worker Setup

Run a worker to process durable agent requests:

	worker := durable.NewWorker(temporalClient, myAgent, "my-agent-queue")
	if err := worker.Start(); err != nil {
	    log.Fatal(err)
	}
	defer worker.Stop()

# Session ID

Sessions are identified by an [SessionID] which combines an agent name
and a unique key:

	sessionID := durable.NewSessionID("my-agent", "user-123")

The session ID is used to:
  - Route requests to the correct workflow instance
  - Enable session resumption across process restarts
  - Support multi-tenant agent deployments

# Time-to-Live

Sessions can have a time-to-live (TTL) that automatically expires idle sessions:

	durableAgent := durable.NewAgent(myAgent, temporalClient,
	    durable.WithTimeToLive(24 * time.Hour),
	)

When a session expires, its state is deleted and a new session must be created.

# Conversation History

The session maintains full conversation history, enabling the agent to
reference previous messages. History is persisted as workflow state and
survives process restarts.

Access conversation history:

	session, err := durableAgent.GetSession(ctx, sessionID)
	if err != nil {
	    return err
	}
	messages := session.Messages()

# Cross-Platform Compatibility

The state schema (version 1.1.0) is designed for cross-platform compatibility.
Sessions created by the Go implementation can be read by .NET and Python, and
vice versa. Content types are discriminated using the "$type" field:

  - "text": Plain text content
  - "functionCall": Function/tool call request
  - "functionResult": Function/tool call result
  - "data": Binary data with URI
  - "error": Error information
  - "usage": Token usage statistics

# Architecture

The package uses Temporal.io for durable execution:

  - SessionWorkflow: Long-running workflow that maintains session state
  - RunActivity: Activity that executes the agent for a single turn
  - Worker: Hosts workflows and activities

Requests flow through Temporal's update mechanism for synchronous responses
while maintaining durability guarantees.
*/
package durable
