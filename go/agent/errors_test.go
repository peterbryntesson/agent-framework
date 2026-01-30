// Copyright (c) Microsoft. All rights reserved.

package agent

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSentinelErrors_AreDistinct(t *testing.T) {
	// Arrange
	sentinels := []error{
		ErrSessionNotFound,
		ErrInvalidInput,
		ErrRateLimited,
		ErrProviderError,
		ErrToolInvocationFailed,
	}

	// Act & Assert
	for i, err1 := range sentinels {
		for j, err2 := range sentinels {
			if i != j {
				assert.False(t, errors.Is(err1, err2), "sentinel errors should be distinct: %v vs %v", err1, err2)
			}
		}
	}
}

func TestSentinelErrors_HaveDescriptiveMessages(t *testing.T) {
	// Arrange & Act & Assert
	assert.Equal(t, "session not found", ErrSessionNotFound.Error())
	assert.Equal(t, "invalid input", ErrInvalidInput.Error())
	assert.Equal(t, "rate limited", ErrRateLimited.Error())
	assert.Equal(t, "provider error", ErrProviderError.Error())
	assert.Equal(t, "tool invocation failed", ErrToolInvocationFailed.Error())
}

func TestAgentError_ErrorWithAgentID(t *testing.T) {
	// Arrange
	agentErr := &AgentError{
		Op:      "Run",
		AgentID: "agent-123",
		Err:     ErrProviderError,
	}

	// Act
	message := agentErr.Error()

	// Assert
	assert.Equal(t, "agent agent-123: Run: provider error", message)
}

func TestAgentError_ErrorWithoutAgentID(t *testing.T) {
	// Arrange
	agentErr := &AgentError{
		Op:      "RunStream",
		AgentID: "",
		Err:     ErrRateLimited,
	}

	// Act
	message := agentErr.Error()

	// Assert
	assert.Equal(t, "RunStream: rate limited", message)
}

func TestAgentError_Unwrap(t *testing.T) {
	// Arrange
	underlying := ErrSessionNotFound
	agentErr := &AgentError{
		Op:      "RestoreSession",
		AgentID: "agent-456",
		Err:     underlying,
	}

	// Act
	unwrapped := agentErr.Unwrap()

	// Assert
	assert.Equal(t, underlying, unwrapped)
}

func TestAgentError_WorksWithErrorsIs(t *testing.T) {
	// Arrange
	agentErr := &AgentError{
		Op:      "NewSession",
		AgentID: "agent-789",
		Err:     ErrInvalidInput,
	}

	// Act & Assert
	assert.True(t, errors.Is(agentErr, ErrInvalidInput))
	assert.False(t, errors.Is(agentErr, ErrProviderError))
}

func TestAgentError_WorksWithErrorsAs(t *testing.T) {
	// Arrange
	agentErr := &AgentError{
		Op:      "Run",
		AgentID: "agent-abc",
		Err:     ErrToolInvocationFailed,
	}
	wrappedErr := fmt.Errorf("outer error: %w", agentErr)

	// Act
	var target *AgentError
	found := errors.As(wrappedErr, &target)

	// Assert
	require.True(t, found)
	assert.Equal(t, "Run", target.Op)
	assert.Equal(t, "agent-abc", target.AgentID)
	assert.Equal(t, ErrToolInvocationFailed, target.Err)
}

func TestAgentError_NestedWrapping(t *testing.T) {
	// Arrange
	innerErr := &AgentError{
		Op:      "ToolInvoke",
		AgentID: "inner-agent",
		Err:     ErrToolInvocationFailed,
	}
	outerErr := &AgentError{
		Op:      "Run",
		AgentID: "outer-agent",
		Err:     innerErr,
	}

	// Act & Assert
	assert.True(t, errors.Is(outerErr, ErrToolInvocationFailed))
	assert.Contains(t, outerErr.Error(), "outer-agent")
}

func TestNewAgentError_CreatesCorrectError(t *testing.T) {
	// Arrange
	op := "GetService"
	agentID := "test-agent"
	err := ErrSessionNotFound

	// Act
	agentErr := NewAgentError(op, agentID, err)

	// Assert
	require.NotNil(t, agentErr)
	assert.Equal(t, op, agentErr.Op)
	assert.Equal(t, agentID, agentErr.AgentID)
	assert.Equal(t, err, agentErr.Err)
}

func TestIsRetryable_NilError(t *testing.T) {
	// Act & Assert
	assert.False(t, IsRetryable(nil))
}

func TestIsRetryable_RateLimited(t *testing.T) {
	// Arrange
	err := ErrRateLimited

	// Act & Assert
	assert.True(t, IsRetryable(err))
}

func TestIsRetryable_WrappedRateLimited(t *testing.T) {
	// Arrange
	err := &AgentError{
		Op:      "Run",
		AgentID: "agent-123",
		Err:     ErrRateLimited,
	}

	// Act & Assert
	assert.True(t, IsRetryable(err))
}

func TestIsRetryable_ProviderError(t *testing.T) {
	// Arrange
	err := ErrProviderError

	// Act & Assert
	assert.True(t, IsRetryable(err))
}

func TestIsRetryable_WrappedProviderError(t *testing.T) {
	// Arrange
	err := fmt.Errorf("failed: %w", ErrProviderError)

	// Act & Assert
	assert.True(t, IsRetryable(err))
}

func TestIsRetryable_SessionNotFound(t *testing.T) {
	// Arrange
	err := ErrSessionNotFound

	// Act & Assert
	assert.False(t, IsRetryable(err))
}

func TestIsRetryable_InvalidInput(t *testing.T) {
	// Arrange
	err := ErrInvalidInput

	// Act & Assert
	assert.False(t, IsRetryable(err))
}

func TestIsRetryable_ToolInvocationFailed(t *testing.T) {
	// Arrange
	err := ErrToolInvocationFailed

	// Act & Assert
	assert.False(t, IsRetryable(err))
}

func TestIsRetryable_UnknownError(t *testing.T) {
	// Arrange
	err := errors.New("unknown error")

	// Act & Assert
	assert.False(t, IsRetryable(err))
}

func TestIsRetryable_DeeplyNestedRetryable(t *testing.T) {
	// Arrange
	innerErr := &AgentError{
		Op:      "InternalCall",
		AgentID: "inner",
		Err:     ErrRateLimited,
	}
	outerErr := fmt.Errorf("outer: %w", innerErr)

	// Act & Assert
	assert.True(t, IsRetryable(outerErr))
}

func TestAgentError_AllFieldsSet(t *testing.T) {
	// Arrange
	agentErr := &AgentError{
		Op:      "RunStream",
		AgentID: "full-test-agent",
		Err:     ErrProviderError,
	}

	// Act & Assert
	assert.Equal(t, "RunStream", agentErr.Op)
	assert.Equal(t, "full-test-agent", agentErr.AgentID)
	assert.Equal(t, ErrProviderError, agentErr.Err)
	assert.Equal(t, "agent full-test-agent: RunStream: provider error", agentErr.Error())
}

func TestAgentError_ImplementsErrorInterface(t *testing.T) {
	// Arrange
	var _ error = &AgentError{}

	// Act & Assert - compilation proves implementation
	agentErr := NewAgentError("Test", "agent-id", ErrInvalidInput)
	assert.Implements(t, (*error)(nil), agentErr)
}
