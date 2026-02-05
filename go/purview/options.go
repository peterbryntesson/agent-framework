// Copyright (c) Microsoft. All rights reserved.

package purview

import (
	"context"
	"net/http"
	"time"
)

// TokenCredential abstracts Azure Identity credential for authentication.
// This interface is compatible with azcore.TokenCredential from the Azure SDK.
type TokenCredential interface {
	// GetToken retrieves an access token for the specified scopes.
	GetToken(ctx context.Context, options TokenRequestOptions) (AccessToken, error)
}

// TokenRequestOptions contains options for token requests.
type TokenRequestOptions struct {
	Scopes []string
}

// AccessToken represents an Azure access token.
type AccessToken struct {
	Token     string
	ExpiresOn time.Time
}

// Option configures a Middleware.
type Option func(*Middleware)

// WithHTTPClient sets a custom HTTP client for API calls.
func WithHTTPClient(client *http.Client) Option {
	return func(m *Middleware) {
		m.httpClient = client
	}
}

// WithViolationHandler sets a custom handler for policy violations.
// Return nil to allow content despite violations, or an error to block.
func WithViolationHandler(handler ViolationHandler) Option {
	return func(m *Middleware) {
		m.violationHandler = handler
	}
}

// WithCache sets a cache for scope definitions.
func WithCache(cache ScopeCache) Option {
	return func(m *Middleware) {
		m.cache = cache
	}
}

// WithCacheTTL sets the TTL for cached scope definitions.
// Default is 1 hour.
func WithCacheTTL(ttl time.Duration) Option {
	return func(m *Middleware) {
		m.cacheTTL = ttl
	}
}

// WithEvaluateInput enables/disables evaluation of input content.
// Default is true.
func WithEvaluateInput(enabled bool) Option {
	return func(m *Middleware) {
		m.evaluateInput = enabled
	}
}

// WithEvaluateOutput enables/disables evaluation of output content.
// Default is true.
func WithEvaluateOutput(enabled bool) Option {
	return func(m *Middleware) {
		m.evaluateOutput = enabled
	}
}
