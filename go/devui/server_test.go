// Copyright (c) Microsoft. All rights reserved.

package devui

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewServer(t *testing.T) {
	server := NewServer()

	assert.NotNil(t, server)
	assert.NotNil(t, server.registry)
	assert.NotNil(t, server.collector)
	assert.Equal(t, 8080, server.port)
}

func TestNewServer_WithOptions(t *testing.T) {
	server := NewServer(
		WithPort(9000),
		WithTraceCollection(false),
		WithMaxTraces(500),
	)

	assert.Equal(t, 9000, server.port)
	assert.Nil(t, server.collector)
}

func TestServer_RegisterAgent(t *testing.T) {
	server := NewServer()
	agent := &mockAgentFull{id: "test-id", name: "test-agent"}

	server.RegisterAgent("test", agent)

	result := server.registry.Get("test")
	require.NotNil(t, result)
	assert.Equal(t, "test-id", result.ID())
}

func TestServer_UnregisterAgent(t *testing.T) {
	server := NewServer()
	agent := &mockAgentFull{id: "test-id", name: "test-agent"}

	server.RegisterAgent("test", agent)
	server.UnregisterAgent("test")

	result := server.registry.Get("test")
	assert.Nil(t, result)
}

func TestServer_HandleListAgents(t *testing.T) {
	server := NewServer()
	server.RegisterAgent("agent1", &mockAgentFull{id: "id1", name: "Agent 1", description: "First agent"})
	server.RegisterAgent("agent2", &mockAgentFull{id: "id2", name: "Agent 2", description: "Second agent"})

	req := httptest.NewRequest(http.MethodGet, "/api/agents", nil)
	w := httptest.NewRecorder()

	server.ServeHTTP(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var agentsResp AgentsResponse
	err := json.NewDecoder(resp.Body).Decode(&agentsResp)
	require.NoError(t, err)

	assert.Len(t, agentsResp.Agents, 2)
}

func TestServer_HandleGetAgent(t *testing.T) {
	server := NewServer()
	server.RegisterAgent("test", &mockAgentFull{id: "id1", name: "Test Agent", description: "A test agent"})

	req := httptest.NewRequest(http.MethodGet, "/api/agents/test", nil)
	w := httptest.NewRecorder()

	server.ServeHTTP(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var agentInfo AgentInfo
	err := json.NewDecoder(resp.Body).Decode(&agentInfo)
	require.NoError(t, err)

	assert.Equal(t, "test", agentInfo.Name)
	assert.Equal(t, "id1", agentInfo.ID)
}

func TestServer_HandleGetAgent_NotFound(t *testing.T) {
	server := NewServer()

	req := httptest.NewRequest(http.MethodGet, "/api/agents/nonexistent", nil)
	w := httptest.NewRecorder()

	server.ServeHTTP(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestServer_HandleRunAgent(t *testing.T) {
	server := NewServer()
	server.RegisterAgent("test", &mockAgentFull{id: "id1", name: "Test Agent"})

	body := `{"messages":[{"role":"user","content":"Hello"}]}`
	req := httptest.NewRequest(http.MethodPost, "/api/agents/test/run", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.ServeHTTP(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var runResp RunResponse
	err := json.NewDecoder(resp.Body).Decode(&runResp)
	require.NoError(t, err)

	assert.Equal(t, "Hello, World!", runResp.Content)
	assert.NotEmpty(t, runResp.ConversationID)
}

func TestServer_HandleRunAgent_NotFound(t *testing.T) {
	server := NewServer()

	body := `{"messages":[{"role":"user","content":"Hello"}]}`
	req := httptest.NewRequest(http.MethodPost, "/api/agents/nonexistent/run", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.ServeHTTP(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestServer_HandleRunAgent_InvalidBody(t *testing.T) {
	server := NewServer()
	server.RegisterAgent("test", &mockAgentFull{id: "id1", name: "Test Agent"})

	body := `invalid json`
	req := httptest.NewRequest(http.MethodPost, "/api/agents/test/run", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.ServeHTTP(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestServer_HandleRunAgent_NoMessages(t *testing.T) {
	server := NewServer()
	server.RegisterAgent("test", &mockAgentFull{id: "id1", name: "Test Agent"})

	body := `{"messages":[]}`
	req := httptest.NewRequest(http.MethodPost, "/api/agents/test/run", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.ServeHTTP(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestServer_HandleStreamAgent(t *testing.T) {
	server := NewServer()
	server.RegisterAgent("test", &mockAgentFull{id: "id1", name: "Test Agent"})

	body := `{"messages":[{"role":"user","content":"Hello"}]}`
	req := httptest.NewRequest(http.MethodPost, "/api/agents/test/stream", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.ServeHTTP(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "text/event-stream", resp.Header.Get("Content-Type"))

	bodyBytes, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	bodyStr := string(bodyBytes)

	// Check for SSE events
	assert.Contains(t, bodyStr, "event: start")
	assert.Contains(t, bodyStr, "event: delta")
	assert.Contains(t, bodyStr, "event: complete")
	assert.Contains(t, bodyStr, "event: done")
}

func TestServer_HandleListTraces(t *testing.T) {
	server := NewServer()

	req := httptest.NewRequest(http.MethodGet, "/api/traces", nil)
	w := httptest.NewRecorder()

	server.ServeHTTP(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var tracesResp TracesResponse
	err := json.NewDecoder(resp.Body).Decode(&tracesResp)
	require.NoError(t, err)

	assert.Equal(t, 0, tracesResp.Total)
}

func TestServer_HandleClearTraces(t *testing.T) {
	server := NewServer()

	req := httptest.NewRequest(http.MethodDelete, "/api/traces", nil)
	w := httptest.NewRecorder()

	server.ServeHTTP(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	assert.Equal(t, http.StatusNoContent, resp.StatusCode)
}

func TestServer_HandleHealth(t *testing.T) {
	server := NewServer()
	server.RegisterAgent("test", &mockAgentFull{id: "id1", name: "Test Agent"})

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	server.ServeHTTP(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	bodyBytes, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	assert.Contains(t, string(bodyBytes), `"status":"healthy"`)
}

func TestServer_DefaultPage(t *testing.T) {
	server := NewServer()

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	server.ServeHTTP(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "text/html; charset=utf-8", resp.Header.Get("Content-Type"))
}

func TestServer_CORS(t *testing.T) {
	server := NewServer(WithCORS("http://localhost:3000"))

	req := httptest.NewRequest(http.MethodOptions, "/api/agents", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	w := httptest.NewRecorder()

	server.ServeHTTP(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	assert.Equal(t, http.StatusNoContent, resp.StatusCode)
	assert.Equal(t, "http://localhost:3000", resp.Header.Get("Access-Control-Allow-Origin"))
}

func TestServer_Address(t *testing.T) {
	server := NewServer(WithPort(9000))
	assert.Equal(t, "http://localhost:9000", server.Address())

	server2 := NewServer(WithAddress("0.0.0.0:8888"))
	assert.Equal(t, "0.0.0.0:8888", server2.Address())
}

func TestServer_StopWithoutStart(t *testing.T) {
	server := NewServer()

	// Should not panic or error
	err := server.Stop(context.Background())
	assert.NoError(t, err)
}
