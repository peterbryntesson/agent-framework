// Copyright (c) Microsoft. All rights reserved.

// Package hosting provides infrastructure components for hosted agent scenarios.
//
// This package includes:
//   - SessionStore: Interface for persisting agent sessions across requests
//   - InMemorySessionStore: Thread-safe in-memory session storage
//   - NoopSessionStore: No-op implementation for testing
//   - HostedAgentBuilder: Fluent builder for creating hosted agents with middleware
//   - HostedAgent: Wrapper that adds session management to agents
//
// These components are typically used when building HTTP or gRPC servers
// that expose agents as network services.
//
// # Session Stores
//
// Session stores enable persistent conversations across HTTP requests,
// application restarts, or different service instances:
//
//	store := hosting.NewInMemorySessionStore()
//	session, _ := store.GetSession(ctx, agent, "conversation-123")
//
// # Hosted Agent Builder
//
// The builder pattern simplifies creating hosted agents with session
// management and HTTP middleware:
//
//	hosted, err := hosting.NewHostedAgentBuilder("my-agent").
//	    WithAgent(myAgent).
//	    WithSessionStore(store).
//	    WithMiddleware(loggingMiddleware).
//	    Build()
//	if err != nil {
//	    // handle error
//	}
//
// See the hosting/openai subpackage for OpenAI-compatible HTTP endpoints.
package hosting
