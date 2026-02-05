// Copyright (c) Microsoft. All rights reserved.

package purview

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/microsoft/agent-framework-go/agent"
	"github.com/microsoft/agent-framework-go/chat"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createMockServer(t *testing.T, isViolation bool) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		resp := EvaluationResponse{
			IsViolation: isViolation,
			PolicyName:  "test-policy",
		}
		if isViolation {
			resp.Violations = []ViolationResponse{
				{Category: "PII", Severity: "high", Confidence: 0.9},
			}
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
}

func TestNewMiddleware(t *testing.T) {
	cred := &mockCredential{token: "test-token"}
	settings := Settings{
		TenantID:      "test-tenant",
		ContentPolicy: "test-policy",
	}

	middleware := NewMiddleware(cred, settings)

	assert.NotNil(t, middleware)
	assert.NotNil(t, middleware.client)
	assert.True(t, middleware.evaluateInput)
	assert.True(t, middleware.evaluateOutput)
}

func TestNewMiddleware_WithOptions(t *testing.T) {
	cred := &mockCredential{token: "test-token"}
	settings := Settings{
		TenantID:      "test-tenant",
		ContentPolicy: "test-policy",
	}

	handler := func(_ context.Context, _ *PolicyResult) error {
		return nil
	}

	middleware := NewMiddleware(cred, settings,
		WithEvaluateInput(false),
		WithEvaluateOutput(true),
		WithViolationHandler(handler),
	)

	assert.False(t, middleware.evaluateInput)
	assert.True(t, middleware.evaluateOutput)
	assert.NotNil(t, middleware.violationHandler)
}

func TestMiddleware_Apply_NoViolation(t *testing.T) {
	mockServer := createMockServer(t, false)
	defer mockServer.Close()

	cred := &mockCredential{token: "test-token"}
	settings := Settings{
		TenantID:         "test-tenant",
		ContentPolicy:    "test-policy",
		BlockOnViolation: true,
	}

	middleware := NewMiddleware(cred, settings)
	middleware.client = middleware.client.WithBaseURL(mockServer.URL)

	agentMiddleware := middleware.Apply()

	// Create mock context
	agentCtx := &agent.AgentContext{
		Messages: []agent.Message{
			chat.NewUserMessage("Hello"),
		},
		Response: &agent.Response{
			Messages: []agent.Message{
				chat.NewAssistantMessage("Hello back!"),
			},
		},
	}

	nextCalled := false
	next := func(_ context.Context, _ *agent.AgentContext) error {
		nextCalled = true
		return nil
	}

	err := agentMiddleware.Process(context.Background(), agentCtx, next)

	require.NoError(t, err)
	assert.True(t, nextCalled)
}

func TestMiddleware_Apply_InputViolation_Block(t *testing.T) {
	mockServer := createMockServer(t, true)
	defer mockServer.Close()

	cred := &mockCredential{token: "test-token"}
	settings := Settings{
		TenantID:         "test-tenant",
		ContentPolicy:    "test-policy",
		BlockOnViolation: true,
	}

	middleware := NewMiddleware(cred, settings)
	middleware.client = middleware.client.WithBaseURL(mockServer.URL)

	agentMiddleware := middleware.Apply()

	agentCtx := &agent.AgentContext{
		Messages: []agent.Message{
			chat.NewUserMessage("My SSN is 123-45-6789"),
		},
	}

	nextCalled := false
	next := func(_ context.Context, _ *agent.AgentContext) error {
		nextCalled = true
		return nil
	}

	err := agentMiddleware.Process(context.Background(), agentCtx, next)

	require.Error(t, err)
	assert.False(t, nextCalled)

	var policyErr *PolicyViolationError
	assert.ErrorAs(t, err, &policyErr)
	assert.True(t, policyErr.Result.IsViolation)
}

func TestMiddleware_Apply_InputViolation_Allow(t *testing.T) {
	mockServer := createMockServer(t, true)
	defer mockServer.Close()

	cred := &mockCredential{token: "test-token"}
	settings := Settings{
		TenantID:         "test-tenant",
		ContentPolicy:    "test-policy",
		BlockOnViolation: false, // Don't block
	}

	middleware := NewMiddleware(cred, settings)
	middleware.client = middleware.client.WithBaseURL(mockServer.URL)

	agentMiddleware := middleware.Apply()

	agentCtx := &agent.AgentContext{
		Messages: []agent.Message{
			chat.NewUserMessage("My SSN is 123-45-6789"),
		},
		Response: &agent.Response{
			Messages: []agent.Message{
				chat.NewAssistantMessage("OK"),
			},
		},
	}

	nextCalled := false
	next := func(_ context.Context, _ *agent.AgentContext) error {
		nextCalled = true
		return nil
	}

	err := agentMiddleware.Process(context.Background(), agentCtx, next)

	require.NoError(t, err)
	assert.True(t, nextCalled)
}

func TestMiddleware_Apply_CustomHandler(t *testing.T) {
	mockServer := createMockServer(t, true)
	defer mockServer.Close()

	cred := &mockCredential{token: "test-token"}
	settings := Settings{
		TenantID:         "test-tenant",
		ContentPolicy:    "test-policy",
		BlockOnViolation: true,
	}

	handlerCalled := false
	var capturedResult *PolicyResult
	handler := func(_ context.Context, result *PolicyResult) error {
		handlerCalled = true
		capturedResult = result
		return nil // Allow despite violation
	}

	middleware := NewMiddleware(cred, settings, WithViolationHandler(handler))
	middleware.client = middleware.client.WithBaseURL(mockServer.URL)

	agentMiddleware := middleware.Apply()

	agentCtx := &agent.AgentContext{
		Messages: []agent.Message{
			chat.NewUserMessage("My SSN is 123-45-6789"),
		},
		Response: &agent.Response{
			Messages: []agent.Message{
				chat.NewAssistantMessage("OK"),
			},
		},
	}

	next := func(_ context.Context, _ *agent.AgentContext) error {
		return nil
	}

	err := agentMiddleware.Process(context.Background(), agentCtx, next)

	require.NoError(t, err)
	assert.True(t, handlerCalled)
	assert.NotNil(t, capturedResult)
	assert.True(t, capturedResult.IsViolation)
}

func TestMiddleware_Apply_SkipInput(t *testing.T) {
	callCount := 0
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		callCount++
		resp := EvaluationResponse{IsViolation: false}
		json.NewEncoder(w).Encode(resp)
	}))
	defer mockServer.Close()

	cred := &mockCredential{token: "test-token"}
	settings := Settings{
		TenantID:      "test-tenant",
		ContentPolicy: "test-policy",
	}

	middleware := NewMiddleware(cred, settings, WithEvaluateInput(false))
	middleware.client = middleware.client.WithBaseURL(mockServer.URL)

	agentMiddleware := middleware.Apply()

	agentCtx := &agent.AgentContext{
		Messages: []agent.Message{
			chat.NewUserMessage("Test input"),
		},
		Response: &agent.Response{
			Messages: []agent.Message{
				chat.NewAssistantMessage("Test output"),
			},
		},
	}

	next := func(_ context.Context, _ *agent.AgentContext) error {
		return nil
	}

	err := agentMiddleware.Process(context.Background(), agentCtx, next)

	require.NoError(t, err)
	// Only output should be evaluated
	assert.Equal(t, 1, callCount)
}

func TestMiddleware_ProcessContent(t *testing.T) {
	mockServer := createMockServer(t, false)
	defer mockServer.Close()

	cred := &mockCredential{token: "test-token"}
	settings := Settings{
		TenantID:      "test-tenant",
		ContentPolicy: "test-policy",
	}

	middleware := NewMiddleware(cred, settings)
	middleware.client = middleware.client.WithBaseURL(mockServer.URL)

	result, err := middleware.ProcessContent(context.Background(), "Test content")

	require.NoError(t, err)
	assert.False(t, result.IsViolation)
}

func TestPolicyViolationError_Error(t *testing.T) {
	err := &PolicyViolationError{
		Result: &PolicyResult{
			IsViolation: true,
			Violations: []Violation{
				{Category: "PII"},
			},
		},
	}

	assert.Equal(t, "content policy violation: PII", err.Error())
}

func TestPolicyViolationError_Error_NoViolations(t *testing.T) {
	err := &PolicyViolationError{
		Result: &PolicyResult{
			IsViolation: true,
			Violations:  []Violation{},
		},
	}

	assert.Equal(t, "content policy violation", err.Error())
}
