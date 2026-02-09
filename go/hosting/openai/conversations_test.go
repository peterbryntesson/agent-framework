// Copyright (c) Microsoft. All rights reserved.

package openai

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHandler_ConversationsLifecycle(t *testing.T) {
	// Arrange
	ag := &mockAgent{id: "agent-1", name: "test-model"}
	index := NewInMemoryAgentConversationIndex()
	store := NewInMemoryConversationStore()
	handler := NewHandler(ag, WithConversationIndex(index), WithConversationStore(store))

	createBody := `{"metadata":{"agent_id":"agent-1"},"items":[{"type":"message","role":"user","content":"Hi"}]}`
	createReq := httptest.NewRequest("POST", "/v1/conversations", strings.NewReader(createBody))
	createReq.Header.Set("Content-Type", "application/json")
	createRec := httptest.NewRecorder()

	// Act - create conversation
	handler.ServeHTTP(createRec, createReq)

	// Assert
	assert.Equal(t, http.StatusOK, createRec.Code)
	var created Conversation
	err := json.Unmarshal(createRec.Body.Bytes(), &created)
	require.NoError(t, err)
	require.NotEmpty(t, created.ID)

	// Act - get conversation
	getReq := httptest.NewRequest("GET", "/v1/conversations/"+created.ID, nil)
	getRec := httptest.NewRecorder()
	handler.ServeHTTP(getRec, getReq)
	assert.Equal(t, http.StatusOK, getRec.Code)

	// Act - list items
	listReq := httptest.NewRequest("GET", "/v1/conversations/"+created.ID+"/items", nil)
	listRec := httptest.NewRecorder()
	handler.ServeHTTP(listRec, listReq)
	assert.Equal(t, http.StatusOK, listRec.Code)

	// Act - list conversations by agent
	listConvReq := httptest.NewRequest("GET", "/v1/conversations?agent_id=agent-1", nil)
	listConvRec := httptest.NewRecorder()
	handler.ServeHTTP(listConvRec, listConvReq)
	assert.Equal(t, http.StatusOK, listConvRec.Code)

	var listResp ListConversationsResponse
	err = json.Unmarshal(listConvRec.Body.Bytes(), &listResp)
	require.NoError(t, err)
	assert.Len(t, listResp.Data, 1)

	// Act - delete conversation
	deleteReq := httptest.NewRequest("DELETE", "/v1/conversations/"+created.ID, nil)
	deleteRec := httptest.NewRecorder()
	handler.ServeHTTP(deleteRec, deleteReq)
	assert.Equal(t, http.StatusOK, deleteRec.Code)
}
