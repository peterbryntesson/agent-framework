// Copyright (c) Microsoft. All rights reserved.

package purview

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/microsoft/agent-framework-go/agent"
	"github.com/microsoft/agent-framework-go/chat"
)

// Middleware provides Purview content policy evaluation for agent invocations.
type Middleware struct {
	client           *Client
	settings         Settings
	httpClient       *http.Client
	violationHandler ViolationHandler
	cache            ScopeCache
	cacheTTL         time.Duration
	evaluateInput    bool
	evaluateOutput   bool
}

// NewMiddleware creates a new Purview middleware.
func NewMiddleware(credential TokenCredential, settings Settings, opts ...Option) *Middleware {
	m := &Middleware{
		settings:       settings,
		evaluateInput:  true,
		evaluateOutput: true,
		cacheTTL:       1 * time.Hour,
	}

	// Apply options
	for _, opt := range opts {
		opt(m)
	}

	// Create client with options
	m.client = NewClient(credential, settings.TenantID)
	if m.httpClient != nil {
		m.client = m.client.WithHTTPClient(m.httpClient)
	}
	if m.cache != nil {
		m.client = m.client.WithCache(m.cache).WithCacheTTL(m.cacheTTL)
	}

	return m
}

// Apply returns an agent middleware that applies Purview policy evaluation.
func (m *Middleware) Apply() agent.AgentMiddleware {
	return agent.AgentMiddlewareFunc(func(ctx context.Context, agentCtx *agent.AgentContext, next agent.AgentHandler) error {
		// Evaluate input content
		if m.evaluateInput && len(agentCtx.Messages) > 0 {
			inputContent := m.extractInputContent(agentCtx.Messages)
			if inputContent != "" {
				result, err := m.evaluateContent(ctx, inputContent)
				if err != nil {
					return err
				}

				if result.IsViolation {
					if err := m.handleViolation(ctx, result, "input"); err != nil {
						return err
					}
				}
			}
		}

		// Execute the next handler
		if err := next(ctx, agentCtx); err != nil {
			return err
		}

		// Evaluate output content
		if m.evaluateOutput && agentCtx.Response != nil {
			outputContent := agentCtx.Response.Text()
			if outputContent != "" {
				result, err := m.evaluateContent(ctx, outputContent)
				if err != nil {
					return err
				}

				if result.IsViolation {
					if err := m.handleViolation(ctx, result, "output"); err != nil {
						return err
					}
				}
			}
		}

		return nil
	})
}

// ProcessContent evaluates content against the configured policy.
// This can be used for standalone content evaluation outside the middleware.
func (m *Middleware) ProcessContent(ctx context.Context, content string) (*PolicyResult, error) {
	return m.evaluateContent(ctx, content)
}

// evaluateContent calls the Purview API to evaluate content.
func (m *Middleware) evaluateContent(ctx context.Context, content string) (*PolicyResult, error) {
	req := EvaluationRequest{
		Content:    content,
		PolicyName: m.settings.ContentPolicy,
		Categories: m.settings.Categories,
	}

	return m.client.EvaluateContent(ctx, req)
}

// handleViolation processes a policy violation.
func (m *Middleware) handleViolation(ctx context.Context, result *PolicyResult, contentType string) error {
	// Call custom violation handler if set
	if m.violationHandler != nil {
		return m.violationHandler(ctx, result)
	}

	// Default behavior: block if configured
	if m.settings.BlockOnViolation {
		return &PolicyViolationError{Result: result}
	}

	return nil
}

// extractInputContent extracts text content from input messages.
func (m *Middleware) extractInputContent(messages []agent.Message) string {
	var parts []string
	for _, msg := range messages {
		if msg.Role == chat.RoleUser || msg.Role == chat.RoleAssistant {
			text := msg.Text()
			if text != "" {
				parts = append(parts, text)
			}
		}
	}
	return strings.Join(parts, "\n")
}

// Ensure Middleware can produce agent.AgentMiddleware.
var _ agent.AgentMiddleware = (*middlewareAdapter)(nil)

type middlewareAdapter struct {
	m *Middleware
}

func (a *middlewareAdapter) Process(ctx context.Context, agentCtx *agent.AgentContext, next agent.AgentHandler) error {
	return a.m.Apply().Process(ctx, agentCtx, next)
}
