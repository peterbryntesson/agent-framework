// Copyright (c) Microsoft. All rights reserved.

// Package a2a provides client and server implementations for the A2A
// (Agent-to-Agent) protocol, enabling communication between AI agents.
//
// The A2A protocol defines a standard way for agents to:
//   - Discover capabilities via AgentCard
//   - Create and manage Tasks for work items
//   - Exchange Messages with text and structured content
//   - Stream responses via Server-Sent Events
//
// Client usage:
//
//	client := a2a.NewClient("https://agent.example.com/a2a")
//	card, _ := client.GetAgentCard(ctx)
//	task, _ := client.CreateTask(ctx, &a2a.CreateTaskRequest{...})
//
// Server usage:
//
//	server := a2a.NewServer(myAgent, a2a.WithAgentCard(card))
//	http.Handle("/a2a/", server.Handler())
//
// See https://a2a-protocol.org/latest/ for the full protocol specification.
package a2a
