// Copyright (c) Microsoft. All rights reserved.

package agent

import (
	"testing"
	"time"

	"github.com/microsoft/agent-framework-go/chat"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestToAgentResponse_EmptyUpdates(t *testing.T) {
	// Arrange
	var updates []ResponseUpdate

	// Act
	response := ToAgentResponse(updates)

	// Assert
	assert.Nil(t, response)
}

func TestToAgentResponse_SingleMessageComplete(t *testing.T) {
	// Arrange
	msg := chat.NewAssistantMessage("Hello, world!")
	updates := []ResponseUpdate{
		{
			Kind:       UpdateKindMessageComplete,
			Message:    &msg,
			ResponseID: "resp-123",
			CreatedAt:  time.Now(),
		},
		{
			Kind:         UpdateKindDone,
			FinishReason: FinishReasonStop,
		},
	}

	// Act
	response := ToAgentResponse(updates)

	// Assert
	require.NotNil(t, response)
	assert.Equal(t, "resp-123", response.ResponseID)
	assert.Len(t, response.Messages, 1)
	assert.Equal(t, FinishReasonStop, response.FinishReason)
}

func TestToAgentResponse_WithUsage(t *testing.T) {
	// Arrange
	updates := []ResponseUpdate{
		{
			Kind: UpdateKindUsage,
			Usage: &UsageDetails{
				InputTokens:  100,
				OutputTokens: 50,
				TotalTokens:  150,
			},
		},
		{
			Kind: UpdateKindDone,
		},
	}

	// Act
	response := ToAgentResponse(updates)

	// Assert
	require.NotNil(t, response)
	require.NotNil(t, response.Usage)
	assert.Equal(t, 100, response.Usage.InputTokens)
	assert.Equal(t, 50, response.Usage.OutputTokens)
	assert.Equal(t, 150, response.Usage.TotalTokens)
}

func TestToAgentResponse_MultipleUsageUpdates(t *testing.T) {
	// Arrange
	updates := []ResponseUpdate{
		{
			Kind: UpdateKindUsage,
			Usage: &UsageDetails{
				InputTokens:  100,
				OutputTokens: 50,
			},
		},
		{
			Kind: UpdateKindUsage,
			Usage: &UsageDetails{
				InputTokens:  50,
				OutputTokens: 25,
			},
		},
		{
			Kind: UpdateKindDone,
		},
	}

	// Act
	response := ToAgentResponse(updates)

	// Assert
	require.NotNil(t, response)
	require.NotNil(t, response.Usage)
	assert.Equal(t, 150, response.Usage.InputTokens)
	assert.Equal(t, 75, response.Usage.OutputTokens)
}

func TestToAgentResponse_WithError(t *testing.T) {
	// Arrange
	updates := []ResponseUpdate{
		{
			Kind:  UpdateKindError,
			Error: assert.AnError,
		},
		{
			Kind: UpdateKindDone,
		},
	}

	// Act
	response := ToAgentResponse(updates)

	// Assert
	require.NotNil(t, response)
	assert.Contains(t, response.Metadata, "error")
}

func TestToAgentResponseFromChannel(t *testing.T) {
	// Arrange
	ch := make(chan ResponseUpdate, 2)
	msg := chat.NewAssistantMessage("Test message")
	ch <- ResponseUpdate{
		Kind:    UpdateKindMessageComplete,
		Message: &msg,
	}
	ch <- ResponseUpdate{
		Kind:         UpdateKindDone,
		FinishReason: FinishReasonStop,
	}
	close(ch)

	// Act
	response := ToAgentResponseFromChannel(ch)

	// Assert
	require.NotNil(t, response)
	assert.Len(t, response.Messages, 1)
	assert.Equal(t, FinishReasonStop, response.FinishReason)
}

func TestResponse_ToResponseUpdates(t *testing.T) {
	// Arrange
	response := &Response{
		ResponseID: "resp-123",
		AgentID:    "agent-456",
		CreatedAt:  time.Now(),
		Messages: []Message{
			chat.NewAssistantMessage("Hello!"),
		},
		Usage: &UsageDetails{
			InputTokens:  10,
			OutputTokens: 5,
		},
		FinishReason: FinishReasonStop,
	}

	// Act
	updates := response.ToResponseUpdates()

	// Assert
	require.Len(t, updates, 3) // Usage + Message + Done
	assert.Equal(t, UpdateKindUsage, updates[0].Kind)
	assert.Equal(t, UpdateKindMessageComplete, updates[1].Kind)
	assert.Equal(t, UpdateKindDone, updates[2].Kind)
	assert.Equal(t, "resp-123", updates[0].ResponseID)
}

func TestResponse_ToResponseUpdates_NoUsage(t *testing.T) {
	// Arrange
	response := &Response{
		ResponseID: "resp-123",
		Messages: []Message{
			chat.NewAssistantMessage("Hello!"),
		},
		FinishReason: FinishReasonStop,
	}

	// Act
	updates := response.ToResponseUpdates()

	// Assert
	require.Len(t, updates, 2) // Message + Done
	assert.Equal(t, UpdateKindMessageComplete, updates[0].Kind)
	assert.Equal(t, UpdateKindDone, updates[1].Kind)
}

func TestResponse_ToResponseUpdates_NilResponse(t *testing.T) {
	// Arrange
	var response *Response

	// Act
	updates := response.ToResponseUpdates()

	// Assert
	assert.Nil(t, updates)
}
