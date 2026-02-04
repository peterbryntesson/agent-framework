// Copyright (c) Microsoft. All rights reserved.

package agui

import (
	"encoding/json"
	"testing"
)

// TestEventTypeConstants tests that event type constants have correct values.
func TestEventTypeConstants(t *testing.T) {
	tests := []struct {
		name     string
		got      EventType
		expected string
	}{
		{"RunStarted", EventTypeRunStarted, "RUN_STARTED"},
		{"RunFinished", EventTypeRunFinished, "RUN_FINISHED"},
		{"RunError", EventTypeRunError, "RUN_ERROR"},
		{"TextMessageStart", EventTypeTextMessageStart, "TEXT_MESSAGE_START"},
		{"TextMessageContent", EventTypeTextMessageContent, "TEXT_MESSAGE_CONTENT"},
		{"TextMessageEnd", EventTypeTextMessageEnd, "TEXT_MESSAGE_END"},
		{"ToolCallStart", EventTypeToolCallStart, "TOOL_CALL_START"},
		{"ToolCallArgs", EventTypeToolCallArgs, "TOOL_CALL_ARGS"},
		{"ToolCallEnd", EventTypeToolCallEnd, "TOOL_CALL_END"},
		{"ToolCallResult", EventTypeToolCallResult, "TOOL_CALL_RESULT"},
		{"StateSnapshot", EventTypeStateSnapshot, "STATE_SNAPSHOT"},
		{"StateDelta", EventTypeStateDelta, "STATE_DELTA"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.got) != tt.expected {
				t.Errorf("EventType %s = %q, want %q", tt.name, tt.got, tt.expected)
			}
		})
	}
}

// TestRunStartedEvent tests RunStartedEvent creation and serialization.
func TestRunStartedEvent(t *testing.T) {
	t.Run("creates event with correct values", func(t *testing.T) {
		event := NewRunStartedEvent("thread-123", "run-456")

		if event.Type() != EventTypeRunStarted {
			t.Errorf("Type() = %q, want %q", event.Type(), EventTypeRunStarted)
		}
		if event.ThreadID != "thread-123" {
			t.Errorf("ThreadID = %q, want %q", event.ThreadID, "thread-123")
		}
		if event.RunID != "run-456" {
			t.Errorf("RunID = %q, want %q", event.RunID, "run-456")
		}
	})

	t.Run("serializes to JSON correctly", func(t *testing.T) {
		event := NewRunStartedEvent("thread-123", "run-456")

		data, err := json.Marshal(event)
		if err != nil {
			t.Fatalf("Marshal error: %v", err)
		}

		var parsed map[string]interface{}
		if err := json.Unmarshal(data, &parsed); err != nil {
			t.Fatalf("Unmarshal error: %v", err)
		}

		if parsed["type"] != "RUN_STARTED" {
			t.Errorf("type = %v, want %q", parsed["type"], "RUN_STARTED")
		}
		if parsed["threadId"] != "thread-123" {
			t.Errorf("threadId = %v, want %q", parsed["threadId"], "thread-123")
		}
		if parsed["runId"] != "run-456" {
			t.Errorf("runId = %v, want %q", parsed["runId"], "run-456")
		}
	})
}

// TestRunFinishedEvent tests RunFinishedEvent creation and serialization.
func TestRunFinishedEvent(t *testing.T) {
	t.Run("creates event with correct values", func(t *testing.T) {
		event := NewRunFinishedEvent("thread-123", "run-456")

		if event.Type() != EventTypeRunFinished {
			t.Errorf("Type() = %q, want %q", event.Type(), EventTypeRunFinished)
		}
		if event.ThreadID != "thread-123" {
			t.Errorf("ThreadID = %q, want %q", event.ThreadID, "thread-123")
		}
		if event.RunID != "run-456" {
			t.Errorf("RunID = %q, want %q", event.RunID, "run-456")
		}
	})

	t.Run("supports result data", func(t *testing.T) {
		result := map[string]string{"status": "success"}
		event := NewRunFinishedEvent("thread-123", "run-456").WithResult(result)

		if event.Result == nil {
			t.Fatal("Result should not be nil")
		}

		var parsed map[string]string
		if err := json.Unmarshal(event.Result, &parsed); err != nil {
			t.Fatalf("Unmarshal result error: %v", err)
		}
		if parsed["status"] != "success" {
			t.Errorf("result.status = %q, want %q", parsed["status"], "success")
		}
	})

	t.Run("serializes to JSON correctly", func(t *testing.T) {
		event := NewRunFinishedEvent("thread-123", "run-456")

		data, err := json.Marshal(event)
		if err != nil {
			t.Fatalf("Marshal error: %v", err)
		}

		var parsed map[string]interface{}
		if err := json.Unmarshal(data, &parsed); err != nil {
			t.Fatalf("Unmarshal error: %v", err)
		}

		if parsed["type"] != "RUN_FINISHED" {
			t.Errorf("type = %v, want %q", parsed["type"], "RUN_FINISHED")
		}
	})
}

// TestRunErrorEvent tests RunErrorEvent creation and serialization.
func TestRunErrorEvent(t *testing.T) {
	t.Run("creates event with message", func(t *testing.T) {
		event := NewRunErrorEvent("Something went wrong")

		if event.Type() != EventTypeRunError {
			t.Errorf("Type() = %q, want %q", event.Type(), EventTypeRunError)
		}
		if event.Message != "Something went wrong" {
			t.Errorf("Message = %q, want %q", event.Message, "Something went wrong")
		}
	})

	t.Run("supports error code", func(t *testing.T) {
		event := NewRunErrorEvent("Error").WithCode("ERR_001")

		if event.Code != "ERR_001" {
			t.Errorf("Code = %q, want %q", event.Code, "ERR_001")
		}
	})

	t.Run("serializes to JSON correctly", func(t *testing.T) {
		event := NewRunErrorEvent("Error message").WithCode("ERR_500")

		data, err := json.Marshal(event)
		if err != nil {
			t.Fatalf("Marshal error: %v", err)
		}

		var parsed map[string]interface{}
		if err := json.Unmarshal(data, &parsed); err != nil {
			t.Fatalf("Unmarshal error: %v", err)
		}

		if parsed["type"] != "RUN_ERROR" {
			t.Errorf("type = %v, want %q", parsed["type"], "RUN_ERROR")
		}
		if parsed["message"] != "Error message" {
			t.Errorf("message = %v, want %q", parsed["message"], "Error message")
		}
		if parsed["code"] != "ERR_500" {
			t.Errorf("code = %v, want %q", parsed["code"], "ERR_500")
		}
	})

	t.Run("omits empty code", func(t *testing.T) {
		event := NewRunErrorEvent("Error")

		data, err := json.Marshal(event)
		if err != nil {
			t.Fatalf("Marshal error: %v", err)
		}

		var parsed map[string]interface{}
		if err := json.Unmarshal(data, &parsed); err != nil {
			t.Fatalf("Unmarshal error: %v", err)
		}

		if _, exists := parsed["code"]; exists && parsed["code"] != "" {
			t.Errorf("code should be omitted or empty, got %v", parsed["code"])
		}
	})
}

// TestTextMessageStartEvent tests TextMessageStartEvent creation and serialization.
func TestTextMessageStartEvent(t *testing.T) {
	t.Run("creates event with correct values", func(t *testing.T) {
		event := NewTextMessageStartEvent("msg-123", "assistant")

		if event.Type() != EventTypeTextMessageStart {
			t.Errorf("Type() = %q, want %q", event.Type(), EventTypeTextMessageStart)
		}
		if event.MessageID != "msg-123" {
			t.Errorf("MessageID = %q, want %q", event.MessageID, "msg-123")
		}
		if event.Role != "assistant" {
			t.Errorf("Role = %q, want %q", event.Role, "assistant")
		}
	})

	t.Run("serializes to JSON correctly", func(t *testing.T) {
		event := NewTextMessageStartEvent("msg-123", "assistant")

		data, err := json.Marshal(event)
		if err != nil {
			t.Fatalf("Marshal error: %v", err)
		}

		var parsed map[string]interface{}
		if err := json.Unmarshal(data, &parsed); err != nil {
			t.Fatalf("Unmarshal error: %v", err)
		}

		if parsed["type"] != "TEXT_MESSAGE_START" {
			t.Errorf("type = %v, want %q", parsed["type"], "TEXT_MESSAGE_START")
		}
		if parsed["messageId"] != "msg-123" {
			t.Errorf("messageId = %v, want %q", parsed["messageId"], "msg-123")
		}
		if parsed["role"] != "assistant" {
			t.Errorf("role = %v, want %q", parsed["role"], "assistant")
		}
	})
}

// TestTextMessageContentEvent tests TextMessageContentEvent creation and serialization.
func TestTextMessageContentEvent(t *testing.T) {
	t.Run("creates event with correct values", func(t *testing.T) {
		event := NewTextMessageContentEvent("msg-123", "Hello, ")

		if event.Type() != EventTypeTextMessageContent {
			t.Errorf("Type() = %q, want %q", event.Type(), EventTypeTextMessageContent)
		}
		if event.MessageID != "msg-123" {
			t.Errorf("MessageID = %q, want %q", event.MessageID, "msg-123")
		}
		if event.Delta != "Hello, " {
			t.Errorf("Delta = %q, want %q", event.Delta, "Hello, ")
		}
	})

	t.Run("serializes to JSON correctly", func(t *testing.T) {
		event := NewTextMessageContentEvent("msg-123", "world!")

		data, err := json.Marshal(event)
		if err != nil {
			t.Fatalf("Marshal error: %v", err)
		}

		var parsed map[string]interface{}
		if err := json.Unmarshal(data, &parsed); err != nil {
			t.Fatalf("Unmarshal error: %v", err)
		}

		if parsed["type"] != "TEXT_MESSAGE_CONTENT" {
			t.Errorf("type = %v, want %q", parsed["type"], "TEXT_MESSAGE_CONTENT")
		}
		if parsed["delta"] != "world!" {
			t.Errorf("delta = %v, want %q", parsed["delta"], "world!")
		}
	})
}

// TestTextMessageEndEvent tests TextMessageEndEvent creation and serialization.
func TestTextMessageEndEvent(t *testing.T) {
	t.Run("creates event with correct values", func(t *testing.T) {
		event := NewTextMessageEndEvent("msg-123")

		if event.Type() != EventTypeTextMessageEnd {
			t.Errorf("Type() = %q, want %q", event.Type(), EventTypeTextMessageEnd)
		}
		if event.MessageID != "msg-123" {
			t.Errorf("MessageID = %q, want %q", event.MessageID, "msg-123")
		}
	})

	t.Run("serializes to JSON correctly", func(t *testing.T) {
		event := NewTextMessageEndEvent("msg-123")

		data, err := json.Marshal(event)
		if err != nil {
			t.Fatalf("Marshal error: %v", err)
		}

		var parsed map[string]interface{}
		if err := json.Unmarshal(data, &parsed); err != nil {
			t.Fatalf("Unmarshal error: %v", err)
		}

		if parsed["type"] != "TEXT_MESSAGE_END" {
			t.Errorf("type = %v, want %q", parsed["type"], "TEXT_MESSAGE_END")
		}
		if parsed["messageId"] != "msg-123" {
			t.Errorf("messageId = %v, want %q", parsed["messageId"], "msg-123")
		}
	})
}

// TestToolCallStartEvent tests ToolCallStartEvent creation and serialization.
func TestToolCallStartEvent(t *testing.T) {
	t.Run("creates event with correct values", func(t *testing.T) {
		event := NewToolCallStartEvent("call-123", "get_weather")

		if event.Type() != EventTypeToolCallStart {
			t.Errorf("Type() = %q, want %q", event.Type(), EventTypeToolCallStart)
		}
		if event.ToolCallID != "call-123" {
			t.Errorf("ToolCallID = %q, want %q", event.ToolCallID, "call-123")
		}
		if event.ToolCallName != "get_weather" {
			t.Errorf("ToolCallName = %q, want %q", event.ToolCallName, "get_weather")
		}
	})

	t.Run("supports parent message ID", func(t *testing.T) {
		event := NewToolCallStartEvent("call-123", "get_weather").WithParentMessageID("msg-456")

		if event.ParentMessageID != "msg-456" {
			t.Errorf("ParentMessageID = %q, want %q", event.ParentMessageID, "msg-456")
		}
	})

	t.Run("serializes to JSON correctly", func(t *testing.T) {
		event := NewToolCallStartEvent("call-123", "get_weather").WithParentMessageID("msg-456")

		data, err := json.Marshal(event)
		if err != nil {
			t.Fatalf("Marshal error: %v", err)
		}

		var parsed map[string]interface{}
		if err := json.Unmarshal(data, &parsed); err != nil {
			t.Fatalf("Unmarshal error: %v", err)
		}

		if parsed["type"] != "TOOL_CALL_START" {
			t.Errorf("type = %v, want %q", parsed["type"], "TOOL_CALL_START")
		}
		if parsed["toolCallId"] != "call-123" {
			t.Errorf("toolCallId = %v, want %q", parsed["toolCallId"], "call-123")
		}
		if parsed["toolCallName"] != "get_weather" {
			t.Errorf("toolCallName = %v, want %q", parsed["toolCallName"], "get_weather")
		}
		if parsed["parentMessageId"] != "msg-456" {
			t.Errorf("parentMessageId = %v, want %q", parsed["parentMessageId"], "msg-456")
		}
	})
}

// TestToolCallArgsEvent tests ToolCallArgsEvent creation and serialization.
func TestToolCallArgsEvent(t *testing.T) {
	t.Run("creates event with correct values", func(t *testing.T) {
		event := NewToolCallArgsEvent("call-123", `{"city":`)

		if event.Type() != EventTypeToolCallArgs {
			t.Errorf("Type() = %q, want %q", event.Type(), EventTypeToolCallArgs)
		}
		if event.ToolCallID != "call-123" {
			t.Errorf("ToolCallID = %q, want %q", event.ToolCallID, "call-123")
		}
		if event.Delta != `{"city":` {
			t.Errorf("Delta = %q, want %q", event.Delta, `{"city":`)
		}
	})

	t.Run("serializes to JSON correctly", func(t *testing.T) {
		event := NewToolCallArgsEvent("call-123", `"Seattle"}`)

		data, err := json.Marshal(event)
		if err != nil {
			t.Fatalf("Marshal error: %v", err)
		}

		var parsed map[string]interface{}
		if err := json.Unmarshal(data, &parsed); err != nil {
			t.Fatalf("Unmarshal error: %v", err)
		}

		if parsed["type"] != "TOOL_CALL_ARGS" {
			t.Errorf("type = %v, want %q", parsed["type"], "TOOL_CALL_ARGS")
		}
		if parsed["delta"] != `"Seattle"}` {
			t.Errorf("delta = %v, want %q", parsed["delta"], `"Seattle"}`)
		}
	})
}

// TestToolCallEndEvent tests ToolCallEndEvent creation and serialization.
func TestToolCallEndEvent(t *testing.T) {
	t.Run("creates event with correct values", func(t *testing.T) {
		event := NewToolCallEndEvent("call-123")

		if event.Type() != EventTypeToolCallEnd {
			t.Errorf("Type() = %q, want %q", event.Type(), EventTypeToolCallEnd)
		}
		if event.ToolCallID != "call-123" {
			t.Errorf("ToolCallID = %q, want %q", event.ToolCallID, "call-123")
		}
	})

	t.Run("serializes to JSON correctly", func(t *testing.T) {
		event := NewToolCallEndEvent("call-123")

		data, err := json.Marshal(event)
		if err != nil {
			t.Fatalf("Marshal error: %v", err)
		}

		var parsed map[string]interface{}
		if err := json.Unmarshal(data, &parsed); err != nil {
			t.Fatalf("Unmarshal error: %v", err)
		}

		if parsed["type"] != "TOOL_CALL_END" {
			t.Errorf("type = %v, want %q", parsed["type"], "TOOL_CALL_END")
		}
		if parsed["toolCallId"] != "call-123" {
			t.Errorf("toolCallId = %v, want %q", parsed["toolCallId"], "call-123")
		}
	})
}

// TestToolCallResultEvent tests ToolCallResultEvent creation and serialization.
func TestToolCallResultEvent(t *testing.T) {
	t.Run("creates event with correct values", func(t *testing.T) {
		event := NewToolCallResultEvent("call-123", "72°F, Sunny")

		if event.Type() != EventTypeToolCallResult {
			t.Errorf("Type() = %q, want %q", event.Type(), EventTypeToolCallResult)
		}
		if event.ToolCallID != "call-123" {
			t.Errorf("ToolCallID = %q, want %q", event.ToolCallID, "call-123")
		}
		if event.Content != "72°F, Sunny" {
			t.Errorf("Content = %q, want %q", event.Content, "72°F, Sunny")
		}
	})

	t.Run("supports optional fields", func(t *testing.T) {
		event := NewToolCallResultEvent("call-123", "result").
			WithMessageID("msg-789").
			WithRole("tool")

		if event.MessageID != "msg-789" {
			t.Errorf("MessageID = %q, want %q", event.MessageID, "msg-789")
		}
		if event.Role != "tool" {
			t.Errorf("Role = %q, want %q", event.Role, "tool")
		}
	})

	t.Run("serializes to JSON correctly", func(t *testing.T) {
		event := NewToolCallResultEvent("call-123", "success").
			WithMessageID("msg-789").
			WithRole("tool")

		data, err := json.Marshal(event)
		if err != nil {
			t.Fatalf("Marshal error: %v", err)
		}

		var parsed map[string]interface{}
		if err := json.Unmarshal(data, &parsed); err != nil {
			t.Fatalf("Unmarshal error: %v", err)
		}

		if parsed["type"] != "TOOL_CALL_RESULT" {
			t.Errorf("type = %v, want %q", parsed["type"], "TOOL_CALL_RESULT")
		}
		if parsed["content"] != "success" {
			t.Errorf("content = %v, want %q", parsed["content"], "success")
		}
		if parsed["messageId"] != "msg-789" {
			t.Errorf("messageId = %v, want %q", parsed["messageId"], "msg-789")
		}
		if parsed["role"] != "tool" {
			t.Errorf("role = %v, want %q", parsed["role"], "tool")
		}
	})
}

// TestStateSnapshotEvent tests StateSnapshotEvent creation and serialization.
func TestStateSnapshotEvent(t *testing.T) {
	t.Run("creates event with state data", func(t *testing.T) {
		state := map[string]interface{}{
			"counter": 42,
			"status":  "running",
		}
		event := NewStateSnapshotEvent(state)

		if event.Type() != EventTypeStateSnapshot {
			t.Errorf("Type() = %q, want %q", event.Type(), EventTypeStateSnapshot)
		}
		if event.Snapshot == nil {
			t.Fatal("Snapshot should not be nil")
		}
	})

	t.Run("handles nil state", func(t *testing.T) {
		event := NewStateSnapshotEvent(nil)

		if event.Snapshot != nil {
			t.Errorf("Snapshot should be nil for nil input")
		}
	})

	t.Run("serializes to JSON correctly", func(t *testing.T) {
		state := map[string]int{"value": 100}
		event := NewStateSnapshotEvent(state)

		data, err := json.Marshal(event)
		if err != nil {
			t.Fatalf("Marshal error: %v", err)
		}

		var parsed map[string]interface{}
		if err := json.Unmarshal(data, &parsed); err != nil {
			t.Fatalf("Unmarshal error: %v", err)
		}

		if parsed["type"] != "STATE_SNAPSHOT" {
			t.Errorf("type = %v, want %q", parsed["type"], "STATE_SNAPSHOT")
		}

		snapshot, ok := parsed["snapshot"].(map[string]interface{})
		if !ok {
			t.Fatalf("snapshot is not a map")
		}
		if snapshot["value"] != float64(100) {
			t.Errorf("snapshot.value = %v, want %v", snapshot["value"], 100)
		}
	})
}

// TestStateDeltaEvent tests StateDeltaEvent creation and serialization.
func TestStateDeltaEvent(t *testing.T) {
	t.Run("creates event with delta data", func(t *testing.T) {
		delta := map[string]interface{}{
			"counter": 1,
		}
		event := NewStateDeltaEvent(delta)

		if event.Type() != EventTypeStateDelta {
			t.Errorf("Type() = %q, want %q", event.Type(), EventTypeStateDelta)
		}
		if event.Delta == nil {
			t.Fatal("Delta should not be nil")
		}
	})

	t.Run("handles nil delta", func(t *testing.T) {
		event := NewStateDeltaEvent(nil)

		if event.Delta != nil {
			t.Errorf("Delta should be nil for nil input")
		}
	})

	t.Run("serializes to JSON correctly", func(t *testing.T) {
		delta := map[string]string{"action": "increment"}
		event := NewStateDeltaEvent(delta)

		data, err := json.Marshal(event)
		if err != nil {
			t.Fatalf("Marshal error: %v", err)
		}

		var parsed map[string]interface{}
		if err := json.Unmarshal(data, &parsed); err != nil {
			t.Fatalf("Unmarshal error: %v", err)
		}

		if parsed["type"] != "STATE_DELTA" {
			t.Errorf("type = %v, want %q", parsed["type"], "STATE_DELTA")
		}

		deltaData, ok := parsed["delta"].(map[string]interface{})
		if !ok {
			t.Fatalf("delta is not a map")
		}
		if deltaData["action"] != "increment" {
			t.Errorf("delta.action = %v, want %q", deltaData["action"], "increment")
		}
	})
}

// TestEventInterface tests that all event types implement the Event interface.
func TestEventInterface(t *testing.T) {
	events := []Event{
		NewRunStartedEvent("thread", "run"),
		NewRunFinishedEvent("thread", "run"),
		NewRunErrorEvent("error"),
		NewTextMessageStartEvent("msg", "role"),
		NewTextMessageContentEvent("msg", "delta"),
		NewTextMessageEndEvent("msg"),
		NewToolCallStartEvent("call", "name"),
		NewToolCallArgsEvent("call", "delta"),
		NewToolCallEndEvent("call"),
		NewToolCallResultEvent("call", "content"),
		NewStateSnapshotEvent(nil),
		NewStateDeltaEvent(nil),
	}

	expectedTypes := []EventType{
		EventTypeRunStarted,
		EventTypeRunFinished,
		EventTypeRunError,
		EventTypeTextMessageStart,
		EventTypeTextMessageContent,
		EventTypeTextMessageEnd,
		EventTypeToolCallStart,
		EventTypeToolCallArgs,
		EventTypeToolCallEnd,
		EventTypeToolCallResult,
		EventTypeStateSnapshot,
		EventTypeStateDelta,
	}

	for i, event := range events {
		if event.Type() != expectedTypes[i] {
			t.Errorf("Event %d Type() = %q, want %q", i, event.Type(), expectedTypes[i])
		}
	}
}

// TestAllEventsSerializeCorrectly tests that all events serialize to valid JSON.
func TestAllEventsSerializeCorrectly(t *testing.T) {
	events := []Event{
		NewRunStartedEvent("thread", "run"),
		NewRunFinishedEvent("thread", "run"),
		NewRunErrorEvent("error"),
		NewTextMessageStartEvent("msg", "role"),
		NewTextMessageContentEvent("msg", "delta"),
		NewTextMessageEndEvent("msg"),
		NewToolCallStartEvent("call", "name"),
		NewToolCallArgsEvent("call", "delta"),
		NewToolCallEndEvent("call"),
		NewToolCallResultEvent("call", "content"),
		NewStateSnapshotEvent(map[string]int{"key": 1}),
		NewStateDeltaEvent(map[string]int{"key": 2}),
	}

	for _, event := range events {
		t.Run(string(event.Type()), func(t *testing.T) {
			data, err := json.Marshal(event)
			if err != nil {
				t.Fatalf("Marshal error: %v", err)
			}

			var parsed map[string]interface{}
			if err := json.Unmarshal(data, &parsed); err != nil {
				t.Fatalf("Unmarshal error: %v", err)
			}

			if parsed["type"] != string(event.Type()) {
				t.Errorf("type = %v, want %q", parsed["type"], event.Type())
			}
		})
	}
}
