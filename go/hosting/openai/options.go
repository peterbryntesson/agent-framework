// Copyright (c) Microsoft. All rights reserved.

package openai

import (
	"github.com/microsoft/agent-framework-go/hosting"
)

// Option configures a Handler.
type Option func(*Handler)

// WithSessionStore sets the session store for conversation persistence.
// When set, conversations can be resumed across multiple requests using
// the X-Conversation-ID header.
func WithSessionStore(store hosting.SessionStore) Option {
	return func(h *Handler) {
		h.sessionStore = store
	}
}

// WithModelName sets the model name returned in API responses.
// This appears in the "model" field of completion responses.
// Defaults to the agent's name if not specified.
func WithModelName(name string) Option {
	return func(h *Handler) {
		h.modelName = name
	}
}

// WithStreamingEnabled enables or disables streaming responses.
// When enabled, requests with "stream": true will receive SSE responses.
// Defaults to true.
func WithStreamingEnabled(enabled bool) Option {
	return func(h *Handler) {
		h.streamingEnabled = enabled
	}
}

// WithBasePath sets the base path for API endpoints.
// Defaults to "/v1".
func WithBasePath(path string) Option {
	return func(h *Handler) {
		h.basePath = path
	}
}

// WithResponsesService sets the responses service for Responses endpoints.
func WithResponsesService(service ResponsesService) Option {
	return func(h *Handler) {
		h.responsesService = service
	}
}

// WithConversationStore sets the conversation store for Conversations endpoints.
func WithConversationStore(store ConversationStore) Option {
	return func(h *Handler) {
		h.conversationStore = store
	}
}

// WithConversationIndex sets the conversation index for agent listing.
func WithConversationIndex(index AgentConversationIndex) Option {
	return func(h *Handler) {
		h.conversationIndex = index
	}
}
