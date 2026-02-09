// Copyright (c) Microsoft. All rights reserved.

package openai

import (
	"bufio"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHandler_ResponsesCreate_NonStreaming(t *testing.T) {
	// Arrange
	ag := &mockAgent{id: "agent-1", name: "test-model"}
	handler := NewHandler(ag)

	reqBody := `{"input":"Hello"}`
	req := httptest.NewRequest("POST", "/v1/responses", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// Act
	handler.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)
	var resp Response
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, "response", resp.Object)
	assert.Equal(t, ResponseStatusCompleted, resp.Status)
	require.NotEmpty(t, resp.Output)
	assert.Equal(t, "Hello from agent", resp.Output[0].Content)
}

func TestHandler_ResponsesCreate_Streaming(t *testing.T) {
	// Arrange
	ag := &mockAgent{id: "agent-1", name: "test-model"}
	handler := NewHandler(ag)

	reqBody := `{"input":"Hello","stream":true}`
	req := httptest.NewRequest("POST", "/v1/responses", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// Act
	handler.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)
	events := parseResponseEvents(w.Body.String())
	require.NotEmpty(t, events)
	assert.Contains(t, events, "response.created")
	assert.Contains(t, events, "response.completed")
}

func parseResponseEvents(body string) []string {
	var events []string
	scanner := bufio.NewScanner(strings.NewReader(body))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "event: ") {
			events = append(events, strings.TrimPrefix(line, "event: "))
		}
	}
	return events
}
