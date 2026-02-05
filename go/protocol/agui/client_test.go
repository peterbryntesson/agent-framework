// Copyright (c) Microsoft. All rights reserved.

package agui

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/microsoft/agent-framework-go/agent"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewClient(t *testing.T) {
	t.Run("creates client with default options", func(t *testing.T) {
		// Act
		client := NewClient("http://localhost:8080")

		// Assert
		assert.NotNil(t, client)
		assert.Equal(t, "http://localhost:8080", client.endpoint)
		assert.NotNil(t, client.httpClient)
	})

	t.Run("creates client with custom headers", func(t *testing.T) {
		// Act
		client := NewClient("http://localhost:8080",
			WithHeader("Authorization", "Bearer token"),
		)

		// Assert
		assert.Equal(t, "Bearer token", client.options.headers["Authorization"])
	})
}

func TestClient_RunStream(t *testing.T) {
	t.Run("streams events from server", func(t *testing.T) {
		// Arrange
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/event-stream")
			w.WriteHeader(http.StatusOK)

			flusher, ok := w.(http.Flusher)
			require.True(t, ok)

			// Send events
			_, _ = w.Write([]byte("event:RUN_STARTED\ndata:{\"type\":\"RUN_STARTED\"}\n\n"))
			flusher.Flush()

			_, _ = w.Write([]byte("event:TEXT_MESSAGE_START\ndata:{\"type\":\"TEXT_MESSAGE_START\",\"messageId\":\"m1\",\"role\":\"assistant\"}\n\n"))
			flusher.Flush()

			_, _ = w.Write([]byte("event:TEXT_MESSAGE_CONTENT\ndata:{\"type\":\"TEXT_MESSAGE_CONTENT\",\"messageId\":\"m1\",\"delta\":\"Hello\"}\n\n"))
			flusher.Flush()

			_, _ = w.Write([]byte("event:TEXT_MESSAGE_END\ndata:{\"type\":\"TEXT_MESSAGE_END\",\"messageId\":\"m1\"}\n\n"))
			flusher.Flush()

			_, _ = w.Write([]byte("event:RUN_FINISHED\ndata:{\"type\":\"RUN_FINISHED\"}\n\n"))
			flusher.Flush()
		}))
		defer server.Close()

		client := NewClient(server.URL)

		// Act
		updates, err := client.RunStream(context.Background(), &ClientRunRequest{
			ThreadID: "test-thread",
			Messages: []agent.Message{
				agent.NewUserMessage("Hi"),
			},
		})

		// Assert
		require.NoError(t, err)

		var received []agent.ResponseUpdate
		for update := range updates {
			received = append(received, update)
		}

		assert.GreaterOrEqual(t, len(received), 2, "expected at least 2 updates")
	})

	t.Run("handles server error status", func(t *testing.T) {
		// Arrange
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer server.Close()

		client := NewClient(server.URL)

		// Act
		_, err := client.RunStream(context.Background(), &ClientRunRequest{
			ThreadID: "test-thread",
		})

		// Assert
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "500")
	})
}

func TestClient_Run(t *testing.T) {
	t.Run("collects streaming updates into response", func(t *testing.T) {
		// Arrange
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/event-stream")
			w.WriteHeader(http.StatusOK)

			flusher, _ := w.(http.Flusher)

			_, _ = w.Write([]byte("event:TEXT_MESSAGE_START\ndata:{\"type\":\"TEXT_MESSAGE_START\",\"messageId\":\"m1\",\"role\":\"assistant\"}\n\n"))
			flusher.Flush()

			_, _ = w.Write([]byte("event:TEXT_MESSAGE_CONTENT\ndata:{\"type\":\"TEXT_MESSAGE_CONTENT\",\"messageId\":\"m1\",\"delta\":\"Hello world\"}\n\n"))
			flusher.Flush()

			_, _ = w.Write([]byte("event:TEXT_MESSAGE_END\ndata:{\"type\":\"TEXT_MESSAGE_END\",\"messageId\":\"m1\"}\n\n"))
			flusher.Flush()

			_, _ = w.Write([]byte("event:RUN_FINISHED\ndata:{\"type\":\"RUN_FINISHED\"}\n\n"))
			flusher.Flush()
		}))
		defer server.Close()

		client := NewClient(server.URL)

		// Act
		response, err := client.Run(context.Background(), &ClientRunRequest{
			ThreadID: "test-thread",
			Messages: []agent.Message{agent.NewUserMessage("Hi")},
		})

		// Assert
		require.NoError(t, err)
		assert.NotNil(t, response)
		assert.Len(t, response.Messages, 1)
	})
}

func TestResponseConverter_Convert(t *testing.T) {
	t.Run("converts text message flow", func(t *testing.T) {
		// Arrange
		converter := newResponseConverter()

		// Act - Start message
		updates := converter.Convert(&TextMessageStartEvent{
			eventBase: eventBase{eventType: EventTypeTextMessageStart},
			MessageID: "m1",
			Role:      "assistant",
		})
		assert.Nil(t, updates, "start event should not produce updates")

		// Act - Content delta
		updates = converter.Convert(&TextMessageContentEvent{
			eventBase: eventBase{eventType: EventTypeTextMessageContent},
			MessageID: "m1",
			Delta:     "Hello",
		})
		require.Len(t, updates, 1)
		assert.Equal(t, agent.UpdateKindContentDelta, updates[0].Kind)

		// Act - End message
		updates = converter.Convert(&TextMessageEndEvent{
			eventBase: eventBase{eventType: EventTypeTextMessageEnd},
			MessageID: "m1",
		})
		require.Len(t, updates, 1)
		assert.Equal(t, agent.UpdateKindMessageComplete, updates[0].Kind)
		assert.NotNil(t, updates[0].Message)
	})

	t.Run("converts tool call flow", func(t *testing.T) {
		// Arrange
		converter := newResponseConverter()

		// Act - Start tool call
		updates := converter.Convert(&ToolCallStartEvent{
			eventBase:    eventBase{eventType: EventTypeToolCallStart},
			ToolCallID:   "tc1",
			ToolCallName: "get_weather",
		})
		require.Len(t, updates, 1)
		assert.Equal(t, agent.UpdateKindToolCall, updates[0].Kind)

		// Act - Tool call args
		updates = converter.Convert(&ToolCallArgsEvent{
			eventBase:  eventBase{eventType: EventTypeToolCallArgs},
			ToolCallID: "tc1",
			Delta:      `{"location":"Seattle"}`,
		})
		require.Len(t, updates, 1)
		assert.Equal(t, agent.UpdateKindContentDelta, updates[0].Kind)

		// Act - End tool call
		updates = converter.Convert(&ToolCallEndEvent{
			eventBase:  eventBase{eventType: EventTypeToolCallEnd},
			ToolCallID: "tc1",
		})
		require.Len(t, updates, 1)
		assert.Equal(t, agent.UpdateKindToolCall, updates[0].Kind)
	})

	t.Run("converts run error", func(t *testing.T) {
		// Arrange
		converter := newResponseConverter()

		// Act
		updates := converter.Convert(&RunErrorEvent{
			eventBase: eventBase{eventType: EventTypeRunError},
			Message:   "Something went wrong",
		})

		// Assert
		require.Len(t, updates, 1)
		assert.Equal(t, agent.UpdateKindError, updates[0].Kind)
		assert.NotNil(t, updates[0].Error)
	})
}

func TestClient_ParseEvent(t *testing.T) {
	tests := []struct {
		name      string
		eventType string
		data      string
		wantType  EventType
		wantError bool
	}{
		{
			name:      "run started",
			eventType: "RUN_STARTED",
			data:      `{"type":"RUN_STARTED"}`,
			wantType:  EventTypeRunStarted,
		},
		{
			name:      "text content",
			eventType: "TEXT_MESSAGE_CONTENT",
			data:      `{"type":"TEXT_MESSAGE_CONTENT","delta":"hello"}`,
			wantType:  EventTypeTextMessageContent,
		},
		{
			name:      "run finished",
			eventType: "RUN_FINISHED",
			data:      `{"type":"RUN_FINISHED"}`,
			wantType:  EventTypeRunFinished,
		},
		{
			name:      "invalid json",
			eventType: "RUN_STARTED",
			data:      `{invalid}`,
			wantError: true,
		},
	}

	client := NewClient("http://localhost")

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Act
			event, err := client.parseEvent(tt.eventType, tt.data)

			// Assert
			if tt.wantError {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			if event != nil {
				assert.Equal(t, tt.wantType, event.Type())
			}
		})
	}
}

func TestClient_BuildRequestBody(t *testing.T) {
	t.Run("builds basic request", func(t *testing.T) {
		// Arrange
		client := NewClient("http://localhost")
		req := &ClientRunRequest{
			ThreadID: "thread-123",
		}

		// Act
		body := client.buildRequestBody(req)

		// Assert
		assert.Equal(t, "thread-123", body["threadId"])
	})

	t.Run("includes messages", func(t *testing.T) {
		// Arrange
		client := NewClient("http://localhost")
		req := &ClientRunRequest{
			ThreadID: "thread-123",
			Messages: []agent.Message{
				agent.NewUserMessage("Hello"),
			},
		}

		// Act
		body := client.buildRequestBody(req)

		// Assert
		msgs, ok := body["messages"].([]map[string]interface{})
		require.True(t, ok)
		assert.Len(t, msgs, 1)
	})

	t.Run("includes state", func(t *testing.T) {
		// Arrange
		client := NewClient("http://localhost")
		req := &ClientRunRequest{
			ThreadID: "thread-123",
			State: map[string]interface{}{
				"key": "value",
			},
		}

		// Act
		body := client.buildRequestBody(req)

		// Assert
		state, ok := body["state"].(map[string]interface{})
		require.True(t, ok)
		assert.Equal(t, "value", state["key"])
	})
}
