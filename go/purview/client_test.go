// Copyright (c) Microsoft. All rights reserved.

package purview

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockCredential implements TokenCredential for testing.
type mockCredential struct {
	token string
	err   error
}

func (m *mockCredential) GetToken(_ context.Context, _ TokenRequestOptions) (AccessToken, error) {
	if m.err != nil {
		return AccessToken{}, m.err
	}
	return AccessToken{
		Token:     m.token,
		ExpiresOn: time.Now().Add(time.Hour),
	}, nil
}

func TestNewClient(t *testing.T) {
	cred := &mockCredential{token: "test-token"}
	client := NewClient(cred, "test-tenant")

	assert.NotNil(t, client)
	assert.Equal(t, "test-tenant", client.tenantID)
}

func TestClient_EvaluateContent(t *testing.T) {
	// Create mock server
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Contains(t, r.URL.Path, "/contentpolicies/")
		assert.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))

		// Return response
		resp := EvaluationResponse{
			IsViolation: false,
			PolicyName:  "test-policy",
			Scores:      map[string]float64{"PII": 0.1},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer mockServer.Close()

	cred := &mockCredential{token: "test-token"}
	client := NewClient(cred, "test-tenant").WithBaseURL(mockServer.URL)

	result, err := client.EvaluateContent(context.Background(), EvaluationRequest{
		Content:    "Test content",
		PolicyName: "test-policy",
	})

	require.NoError(t, err)
	assert.False(t, result.IsViolation)
	assert.Equal(t, "test-policy", result.PolicyName)
}

func TestClient_EvaluateContent_Violation(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		resp := EvaluationResponse{
			IsViolation: true,
			PolicyName:  "test-policy",
			Violations: []ViolationResponse{
				{
					Category:    "PII",
					Severity:    "high",
					Confidence:  0.95,
					Description: "Personal information detected",
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer mockServer.Close()

	cred := &mockCredential{token: "test-token"}
	client := NewClient(cred, "test-tenant").WithBaseURL(mockServer.URL)

	result, err := client.EvaluateContent(context.Background(), EvaluationRequest{
		Content:    "My SSN is 123-45-6789",
		PolicyName: "test-policy",
	})

	require.NoError(t, err)
	assert.True(t, result.IsViolation)
	assert.Len(t, result.Violations, 1)
	assert.Equal(t, "PII", result.Violations[0].Category)
	assert.Equal(t, "high", result.Violations[0].Severity)
}

func TestClient_EvaluateContent_Error(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Internal Server Error"))
	}))
	defer mockServer.Close()

	cred := &mockCredential{token: "test-token"}
	client := NewClient(cred, "test-tenant").WithBaseURL(mockServer.URL)

	result, err := client.EvaluateContent(context.Background(), EvaluationRequest{
		Content:    "Test content",
		PolicyName: "test-policy",
	})

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "500")
}

func TestClient_GetScopeDefinition(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Contains(t, r.URL.Path, "/scopes/test-scope")

		resp := ScopeDefinition{
			ID:         "scope-id",
			Name:       "test-scope",
			Categories: []string{"PII", "NSFW"},
		}
		w.Header().Set("ETag", "abc123")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer mockServer.Close()

	cred := &mockCredential{token: "test-token"}
	client := NewClient(cred, "test-tenant").WithBaseURL(mockServer.URL)

	result, err := client.GetScopeDefinition(context.Background(), "test-scope")

	require.NoError(t, err)
	assert.Equal(t, "test-scope", result.Name)
	assert.Contains(t, result.Categories, "PII")
}

func TestClient_TokenCaching(t *testing.T) {
	callCount := 0
	cred := &mockCredential{token: "test-token"}

	// Wrap to count calls
	countingCred := &countingCredential{
		inner: cred,
		count: &callCount,
	}

	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		resp := EvaluationResponse{IsViolation: false}
		json.NewEncoder(w).Encode(resp)
	}))
	defer mockServer.Close()

	client := NewClient(countingCred, "test-tenant").WithBaseURL(mockServer.URL)

	// Make multiple requests
	for i := 0; i < 3; i++ {
		_, err := client.EvaluateContent(context.Background(), EvaluationRequest{
			Content:    "Test",
			PolicyName: "policy",
		})
		require.NoError(t, err)
	}

	// Token should only be fetched once due to caching
	assert.Equal(t, 1, callCount)
}

type countingCredential struct {
	inner TokenCredential
	count *int
}

func (c *countingCredential) GetToken(ctx context.Context, opts TokenRequestOptions) (AccessToken, error) {
	*c.count++
	return c.inner.GetToken(ctx, opts)
}
