// Copyright (c) Microsoft. All rights reserved.

// Package openai provides HTTP handlers for hosting agents as OpenAI-compatible endpoints.
//
// This package enables agents to be exposed as REST APIs that follow the OpenAI
// Chat Completions API specification. Clients using OpenAI SDKs or compatible
// tools can interact with hosted agents without modification.
//
// # Endpoints
//
// The handler serves the following endpoints:
//
//   - POST /v1/chat/completions - Chat completions (streaming and non-streaming)
//   - GET /v1/models - List available models
//   - POST /v1/responses - Create responses (streaming and non-streaming)
//   - GET /v1/responses/{responseId} - Retrieve responses
//   - POST /v1/responses/{responseId}/cancel - Cancel responses
//   - DELETE /v1/responses/{responseId} - Delete responses
//   - GET /v1/responses/{responseId}/input_items - List response input items
//   - GET /v1/conversations - List conversations by agent ID
//   - POST /v1/conversations - Create conversations
//   - GET /v1/conversations/{conversationId} - Retrieve conversations
//   - POST /v1/conversations/{conversationId} - Update conversations
//   - DELETE /v1/conversations/{conversationId} - Delete conversations
//   - POST /v1/conversations/{conversationId}/items - Add conversation items
//   - GET /v1/conversations/{conversationId}/items - List conversation items
//
// # Basic Usage
//
//	// Create an agent
//	myAgent, _ := agent.NewBuilder("my-agent").
//	    WithClient(chatClient).
//	    WithInstructions("You are a helpful assistant").
//	    Build()
//
//	// Create OpenAI-compatible handler
//	handler := openai.NewHandler(myAgent,
//	    openai.WithModelName("my-agent-v1"),
//	    openai.WithStreamingEnabled(true),
//	)
//
//	// Start HTTP server
//	http.Handle("/", handler)
//	http.ListenAndServe(":8080", nil)
//
// # Streaming Responses
//
// When the request sets "stream": true, responses are sent as Server-Sent Events (SSE):
//
//	data: {"id":"...","object":"chat.completion.chunk","choices":[{"delta":{"content":"Hello"}}]}
//
//	data: {"id":"...","object":"chat.completion.chunk","choices":[{"delta":{"content":" world"}}]}
//
//	data: [DONE]
//
// # Session Management
//
// The handler integrates with the hosting.SessionStore interface for conversation
// persistence. Sessions are identified by the X-Conversation-ID header:
//
//	handler := openai.NewHandler(myAgent,
//	    openai.WithSessionStore(hosting.NewInMemorySessionStore()),
//	)
//
// # Error Handling
//
// Errors are returned in OpenAI-compatible format:
//
//	{
//	    "error": {
//	        "type": "invalid_request_error",
//	        "message": "...",
//	        "code": "..."
//	    }
//	}
package openai
