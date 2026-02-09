// Copyright (c) Microsoft. All rights reserved.

package purview

import (
	"net/http"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
)

// TokenCredential is the Azure SDK credential interface used for authentication.
type TokenCredential = azcore.TokenCredential

// TokenRequestOptions contains options for token requests.
type TokenRequestOptions = policy.TokenRequestOptions

// AccessToken represents an Azure access token.
type AccessToken = azcore.AccessToken

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
