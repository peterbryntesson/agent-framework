// Copyright (c) Microsoft. All rights reserved.

package purview

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

const (
	// purviewScope is the OAuth2 scope for Microsoft Purview API.
	purviewScope = "https://purview.azure.net/.default"

	// defaultBaseURL is the default Purview API base URL.
	defaultBaseURL = "https://api.purview.azure.com"
)

// Client provides access to Microsoft Purview content policy APIs.
type Client struct {
	credential  TokenCredential
	httpClient  *http.Client
	baseURL     string
	tenantID    string
	cache       ScopeCache
	cacheTTL    time.Duration

	// Token caching
	tokenMu     sync.RWMutex
	cachedToken *AccessToken
}

// NewClient creates a new Purview API client.
func NewClient(credential TokenCredential, tenantID string) *Client {
	return &Client{
		credential: credential,
		httpClient: &http.Client{Timeout: 30 * time.Second},
		baseURL:    defaultBaseURL,
		tenantID:   tenantID,
		cacheTTL:   1 * time.Hour,
	}
}

// WithHTTPClient sets a custom HTTP client.
func (c *Client) WithHTTPClient(client *http.Client) *Client {
	c.httpClient = client
	return c
}

// WithBaseURL sets a custom base URL (for testing).
func (c *Client) WithBaseURL(url string) *Client {
	c.baseURL = url
	return c
}

// WithCache sets a cache for scope definitions.
func (c *Client) WithCache(cache ScopeCache) *Client {
	c.cache = cache
	return c
}

// WithCacheTTL sets the cache TTL.
func (c *Client) WithCacheTTL(ttl time.Duration) *Client {
	c.cacheTTL = ttl
	return c
}

// EvaluateContent evaluates content against a policy.
func (c *Client) EvaluateContent(ctx context.Context, req EvaluationRequest) (*PolicyResult, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/contentpolicies/%s/evaluate", c.baseURL, req.PolicyName)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	if err := c.addAuthHeader(ctx, httpReq); err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to call Purview API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("Purview API returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var evalResp EvaluationResponse
	if err := json.NewDecoder(resp.Body).Decode(&evalResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return c.toResult(evalResp), nil
}

// GetScopeDefinition retrieves a scope definition, using cache if available.
func (c *Client) GetScopeDefinition(ctx context.Context, scopeName string) (*ScopeDefinition, error) {
	cacheKey := fmt.Sprintf("scope:%s", scopeName)

	// Check cache first
	if c.cache != nil {
		if data, _, ok := c.cache.Get(ctx, cacheKey); ok {
			var def ScopeDefinition
			if err := json.Unmarshal(data, &def); err == nil {
				return &def, nil
			}
		}
	}

	// Fetch from API
	url := fmt.Sprintf("%s/scopes/%s", c.baseURL, scopeName)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	if err := c.addAuthHeader(ctx, httpReq); err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to call Purview API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("Purview API returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var def ScopeDefinition
	if err := json.Unmarshal(bodyBytes, &def); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Cache the result
	if c.cache != nil {
		etag := resp.Header.Get("ETag")
		def.ETag = etag
		c.cache.Set(ctx, cacheKey, bodyBytes, etag, c.cacheTTL)
	}

	return &def, nil
}

// addAuthHeader adds the Authorization header with a bearer token.
func (c *Client) addAuthHeader(ctx context.Context, req *http.Request) error {
	token, err := c.getToken(ctx)
	if err != nil {
		return fmt.Errorf("failed to get access token: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	return nil
}

// getToken retrieves an access token, using a cached value if valid.
func (c *Client) getToken(ctx context.Context) (string, error) {
	c.tokenMu.RLock()
	if c.cachedToken != nil && time.Until(c.cachedToken.ExpiresOn) > time.Minute {
		token := c.cachedToken.Token
		c.tokenMu.RUnlock()
		return token, nil
	}
	c.tokenMu.RUnlock()

	c.tokenMu.Lock()
	defer c.tokenMu.Unlock()

	// Double-check after acquiring write lock
	if c.cachedToken != nil && time.Until(c.cachedToken.ExpiresOn) > time.Minute {
		return c.cachedToken.Token, nil
	}

	token, err := c.credential.GetToken(ctx, TokenRequestOptions{
		Scopes: []string{purviewScope},
	})
	if err != nil {
		return "", err
	}

	c.cachedToken = &token
	return token.Token, nil
}

// toResult converts an API response to a PolicyResult.
func (c *Client) toResult(resp EvaluationResponse) *PolicyResult {
	result := &PolicyResult{
		IsViolation: resp.IsViolation,
		PolicyName:  resp.PolicyName,
		Scores:      resp.Scores,
		ProcessedAt: time.Now(),
	}

	for _, v := range resp.Violations {
		violation := Violation{
			Category:    v.Category,
			Severity:    v.Severity,
			Confidence:  v.Confidence,
			Description: v.Description,
		}

		if v.StartOffset != nil && v.EndOffset != nil {
			violation.Location = &ContentLocation{
				StartOffset: *v.StartOffset,
				EndOffset:   *v.EndOffset,
				Text:        v.Text,
			}
		}

		result.Violations = append(result.Violations, violation)
	}

	return result
}
