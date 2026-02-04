// Copyright (c) Microsoft. All rights reserved.

package openai

import (
	"bufio"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/microsoft/agent-framework-go/agent"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHandler_ChatCompletions_Streaming(t *testing.T) {
	// Arrange
	ag := &mockAgent{
		id:   "agent-1",
		name: "test-model",
		runStreamFn: func(ctx context.Context, messages []agent.Message, opts ...agent.RunOption) (<-chan agent.ResponseUpdate, error) {
			ch := make(chan agent.ResponseUpdate, 4)
			go func() {
				defer close(ch)
				ch <- agent.ResponseUpdate{
					Kind:  agent.UpdateKindContentDelta,
					Delta: &agent.ContentDelta{TextDelta: "Hello"},
				}
				ch <- agent.ResponseUpdate{
					Kind:  agent.UpdateKindContentDelta,
					Delta: &agent.ContentDelta{TextDelta: " world"},
				}
				ch <- agent.ResponseUpdate{
					Kind:         agent.UpdateKindDone,
					FinishReason: agent.FinishReasonStop,
				}
			}()
			return ch, nil
		},
	}
	handler := NewHandler(ag)

	reqBody := `{
		"model": "test-model",
		"messages": [{"role": "user", "content": "Hello"}],
		"stream": true
	}`

	req := httptest.NewRequest("POST", "/v1/chat/completions", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// Act
	handler.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "text/event-stream", w.Header().Get("Content-Type"))
	assert.Equal(t, "no-cache", w.Header().Get("Cache-Control"))

	// Parse SSE events
	events := parseSSEEvents(t, w.Body.String())
	require.GreaterOrEqual(t, len(events), 3) // At least role, content, done

	// Verify [DONE] marker
	assert.Equal(t, "[DONE]", events[len(events)-1])
}

func TestHandler_ChatCompletions_Streaming_WithToolCalls(t *testing.T) {
	// Arrange
	ag := &mockAgent{
		id:   "agent-1",
		name: "test-model",
		runStreamFn: func(ctx context.Context, messages []agent.Message, opts ...agent.RunOption) (<-chan agent.ResponseUpdate, error) {
			ch := make(chan agent.ResponseUpdate, 4)
			go func() {
				defer close(ch)
				ch <- agent.ResponseUpdate{
					Kind: agent.UpdateKindToolCall,
					Delta: &agent.ContentDelta{
						ToolCallID: "call_123",
						Name:       "get_weather",
						ArgsDelta:  `{"location":`,
					},
				}
				ch <- agent.ResponseUpdate{
					Kind: agent.UpdateKindToolCall,
					Delta: &agent.ContentDelta{
						ArgsDelta: `"Seattle"}`,
					},
				}
				ch <- agent.ResponseUpdate{
					Kind:         agent.UpdateKindDone,
					FinishReason: agent.FinishReasonToolCalls,
				}
			}()
			return ch, nil
		},
	}
	handler := NewHandler(ag)

	reqBody := `{
		"model": "test-model",
		"messages": [{"role": "user", "content": "Weather?"}],
		"stream": true
	}`

	req := httptest.NewRequest("POST", "/v1/chat/completions", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// Act
	handler.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "text/event-stream", w.Header().Get("Content-Type"))

	// Verify we get tool call chunks
	events := parseSSEEvents(t, w.Body.String())
	require.GreaterOrEqual(t, len(events), 3)

	// Check first tool call chunk has function name
	var firstChunk ChatCompletionChunk
	err := json.Unmarshal([]byte(events[0]), &firstChunk)
	require.NoError(t, err)
	require.Len(t, firstChunk.Choices, 1)
	require.Len(t, firstChunk.Choices[0].Delta.ToolCalls, 1)
	assert.Equal(t, "get_weather", firstChunk.Choices[0].Delta.ToolCalls[0].Function.Name)
}

func TestHandler_ChatCompletions_Streaming_Disabled(t *testing.T) {
	// Arrange
	ag := &mockAgent{id: "agent-1", name: "test-model"}
	handler := NewHandler(ag, WithStreamingEnabled(false))

	reqBody := `{
		"model": "test-model",
		"messages": [{"role": "user", "content": "Hello"}],
		"stream": true
	}`

	req := httptest.NewRequest("POST", "/v1/chat/completions", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// Act
	handler.ServeHTTP(w, req)

	// Assert - should fall back to non-streaming
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

	var resp ChatCompletionResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, "chat.completion", resp.Object)
}

func TestHandler_ChatCompletions_Streaming_WithUsage(t *testing.T) {
	// Arrange
	ag := &mockAgent{
		id:   "agent-1",
		name: "test-model",
		runStreamFn: func(ctx context.Context, messages []agent.Message, opts ...agent.RunOption) (<-chan agent.ResponseUpdate, error) {
			ch := make(chan agent.ResponseUpdate, 3)
			go func() {
				defer close(ch)
				ch <- agent.ResponseUpdate{
					Kind:  agent.UpdateKindContentDelta,
					Delta: &agent.ContentDelta{TextDelta: "Hi"},
				}
				ch <- agent.ResponseUpdate{
					Kind: agent.UpdateKindUsage,
					Usage: &agent.UsageDetails{
						InputTokens:  10,
						OutputTokens: 5,
						TotalTokens:  15,
					},
				}
				ch <- agent.ResponseUpdate{
					Kind:         agent.UpdateKindDone,
					FinishReason: agent.FinishReasonStop,
				}
			}()
			return ch, nil
		},
	}
	handler := NewHandler(ag)

	reqBody := `{
		"model": "test-model",
		"messages": [{"role": "user", "content": "Hi"}],
		"stream": true
	}`

	req := httptest.NewRequest("POST", "/v1/chat/completions", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// Act
	handler.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)

	events := parseSSEEvents(t, w.Body.String())

	// Find usage chunk
	var usageFound bool
	for _, event := range events {
		if event == "[DONE]" {
			continue
		}
		var chunk ChatCompletionChunk
		if err := json.Unmarshal([]byte(event), &chunk); err == nil {
			if chunk.Usage != nil {
				usageFound = true
				assert.Equal(t, 10, chunk.Usage.PromptTokens)
				assert.Equal(t, 5, chunk.Usage.CompletionTokens)
				assert.Equal(t, 15, chunk.Usage.TotalTokens)
			}
		}
	}
	assert.True(t, usageFound, "expected usage chunk in stream")
}

// parseSSEEvents parses SSE event data from the response body.
func parseSSEEvents(t *testing.T, body string) []string {
	t.Helper()
	var events []string
	scanner := bufio.NewScanner(strings.NewReader(body))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "data: ") {
			data := strings.TrimPrefix(line, "data: ")
			events = append(events, data)
		}
	}
	return events
}
