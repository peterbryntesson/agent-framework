// Copyright (c) Microsoft. All rights reserved.

package hosting

import (
	"context"
	"fmt"
	"net/http"

	"github.com/microsoft/agent-framework-go/agent"
)

// HostedAgentBuilder builds a hosted agent with session management and HTTP middleware.
//
// Example:
//
//	hosted, err := hosting.NewHostedAgentBuilder("my-agent").
//	    WithAgent(myAgent).
//	    WithSessionStore(hosting.NewInMemorySessionStore()).
//	    WithMiddleware(loggingMiddleware).
//	    Build()
//	if err != nil {
//	    // handle error
//	}
type HostedAgentBuilder struct {
	name         string
	agent        agent.Agent
	sessionStore SessionStore
	middleware   []func(http.Handler) http.Handler
}

// NewHostedAgentBuilder creates a new builder with the specified name.
func NewHostedAgentBuilder(name string) *HostedAgentBuilder {
	return &HostedAgentBuilder{
		name:       name,
		middleware: make([]func(http.Handler) http.Handler, 0),
	}
}

// WithAgent sets the agent to host.
func (b *HostedAgentBuilder) WithAgent(a agent.Agent) *HostedAgentBuilder {
	b.agent = a
	return b
}

// WithSessionStore sets the session store for conversation persistence.
func (b *HostedAgentBuilder) WithSessionStore(s SessionStore) *HostedAgentBuilder {
	b.sessionStore = s
	return b
}

// WithMiddleware adds HTTP middleware to the handler chain.
// Middleware is applied in the order added (first added = outermost).
func (b *HostedAgentBuilder) WithMiddleware(m func(http.Handler) http.Handler) *HostedAgentBuilder {
	b.middleware = append(b.middleware, m)
	return b
}

// Build creates the hosted agent.
// Returns an error if no agent has been set.
func (b *HostedAgentBuilder) Build() (*HostedAgent, error) {
	if b.agent == nil {
		return nil, fmt.Errorf("hosted agent requires an agent to be set via WithAgent")
	}

	// Default to in-memory session store if not specified
	sessionStore := b.sessionStore
	if sessionStore == nil {
		sessionStore = NewInMemorySessionStore()
	}

	return &HostedAgent{
		Agent:        b.agent,
		name:         b.name,
		sessionStore: sessionStore,
		middleware:   b.middleware,
	}, nil
}

// HostedAgent wraps an agent with session management and hosting capabilities.
type HostedAgent struct {
	agent.Agent
	name         string
	sessionStore SessionStore
	middleware   []func(http.Handler) http.Handler
}

// HostedAgentName returns the name of the hosted agent.
func (h *HostedAgent) HostedAgentName() string {
	return h.name
}

// SessionStore returns the session store used by this hosted agent.
func (h *HostedAgent) SessionStore() SessionStore {
	return h.sessionStore
}

// Middleware returns the HTTP middleware chain.
func (h *HostedAgent) Middleware() []func(http.Handler) http.Handler {
	return h.middleware
}

// GetOrCreateSession retrieves or creates a session for the conversation.
// If the session exists in the store, it is restored.
// If not, a new session is created using the agent's NewSession method.
func (h *HostedAgent) GetOrCreateSession(ctx context.Context, conversationID string) (agent.Session, error) {
	return h.sessionStore.GetSession(ctx, h.Agent, conversationID)
}

// SaveSession persists the session state to the store.
func (h *HostedAgent) SaveSession(ctx context.Context, conversationID string, session agent.Session) error {
	return h.sessionStore.SaveSession(ctx, h.Agent, conversationID, session)
}

// DeleteSession removes a session from the store.
func (h *HostedAgent) DeleteSession(ctx context.Context, conversationID string) error {
	return h.sessionStore.DeleteSession(ctx, h.Agent, conversationID)
}

// WrapHandler applies the middleware chain to an HTTP handler.
// Middleware is applied in reverse order so that the first added middleware
// is the outermost (first to receive requests).
func (h *HostedAgent) WrapHandler(handler http.Handler) http.Handler {
	// Apply middleware in reverse order
	for i := len(h.middleware) - 1; i >= 0; i-- {
		handler = h.middleware[i](handler)
	}
	return handler
}
