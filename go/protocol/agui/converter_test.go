// Copyright (c) Microsoft. All rights reserved.

package agui

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/microsoft/agent-framework-go/agent"
	"github.com/microsoft/agent-framework-go/chat"
)

func TestNewEventConverter(t *testing.T) {
	t.Run("creates converter with thread and run IDs", func(t *testing.T) {
		converter := NewEventConverter("thread-123", "run-456")

		if converter.ThreadID() != "thread-123" {
			t.Errorf("ThreadID() = %q, want %q", converter.ThreadID(), "thread-123")
		}
		if converter.RunID() != "run-456" {
			t.Errorf("RunID() = %q, want %q", converter.RunID(), "run-456")
		}
	})

	t.Run("initializes empty active tracking maps", func(t *testing.T) {
		converter := NewEventConverter("thread", "run")

		if len(converter.activeMessages) != 0 {
			t.Errorf("activeMessages should be empty, got %d items", len(converter.activeMessages))
		}
		if len(converter.activeToolCalls) != 0 {
			t.Errorf("activeToolCalls should be empty, got %d items", len(converter.activeToolCalls))
		}
	})
}

func TestEventConverter_RunStarted(t *testing.T) {
	converter := NewEventConverter("thread-123", "run-456")
	event := converter.RunStarted()

	if event.Type() != EventTypeRunStarted {
		t.Errorf("Type() = %q, want %q", event.Type(), EventTypeRunStarted)
	}
	if event.ThreadID != "thread-123" {
		t.Errorf("ThreadID = %q, want %q", event.ThreadID, "thread-123")
	}
	if event.RunID != "run-456" {
		t.Errorf("RunID = %q, want %q", event.RunID, "run-456")
	}
}

func TestEventConverter_RunFinished(t *testing.T) {
	converter := NewEventConverter("thread-123", "run-456")
	event := converter.RunFinished()

	if event.Type() != EventTypeRunFinished {
		t.Errorf("Type() = %q, want %q", event.Type(), EventTypeRunFinished)
	}
	if event.ThreadID != "thread-123" {
		t.Errorf("ThreadID = %q, want %q", event.ThreadID, "thread-123")
	}
	if event.RunID != "run-456" {
		t.Errorf("RunID = %q, want %q", event.RunID, "run-456")
	}
}

func TestEventConverter_Convert_ContentDelta_TextDelta(t *testing.T) {
	t.Run("creates start and content events for new message", func(t *testing.T) {
		converter := NewEventConverter("thread", "run")

		update := agent.ResponseUpdate{
			Kind:      agent.UpdateKindContentDelta,
			MessageID: "msg-1",
			Delta: &agent.ContentDelta{
				TextDelta: "Hello",
				Role:      "assistant",
			},
		}

		events := converter.Convert(update)

		if len(events) != 2 {
			t.Fatalf("expected 2 events, got %d", len(events))
		}

		startEvent, ok := events[0].(*TextMessageStartEvent)
		if !ok {
			t.Fatalf("expected TextMessageStartEvent, got %T", events[0])
		}
		if startEvent.MessageID != "msg-1" {
			t.Errorf("MessageID = %q, want %q", startEvent.MessageID, "msg-1")
		}
		if startEvent.Role != "assistant" {
			t.Errorf("Role = %q, want %q", startEvent.Role, "assistant")
		}

		contentEvent, ok := events[1].(*TextMessageContentEvent)
		if !ok {
			t.Fatalf("expected TextMessageContentEvent, got %T", events[1])
		}
		if contentEvent.Delta != "Hello" {
			t.Errorf("Delta = %q, want %q", contentEvent.Delta, "Hello")
		}
	})

	t.Run("creates only content event for existing message", func(t *testing.T) {
		converter := NewEventConverter("thread", "run")
		converter.activeMessages["msg-1"] = "assistant"

		update := agent.ResponseUpdate{
			Kind:      agent.UpdateKindContentDelta,
			MessageID: "msg-1",
			Delta: &agent.ContentDelta{
				TextDelta: " world",
			},
		}

		events := converter.Convert(update)

		if len(events) != 1 {
			t.Fatalf("expected 1 event, got %d", len(events))
		}

		contentEvent, ok := events[0].(*TextMessageContentEvent)
		if !ok {
			t.Fatalf("expected TextMessageContentEvent, got %T", events[0])
		}
		if contentEvent.Delta != " world" {
			t.Errorf("Delta = %q, want %q", contentEvent.Delta, " world")
		}
	})

	t.Run("uses default role when not specified", func(t *testing.T) {
		converter := NewEventConverter("thread", "run")

		update := agent.ResponseUpdate{
			Kind:      agent.UpdateKindContentDelta,
			MessageID: "msg-1",
			Delta: &agent.ContentDelta{
				TextDelta: "Hello",
			},
		}

		events := converter.Convert(update)

		startEvent := events[0].(*TextMessageStartEvent)
		if startEvent.Role != "assistant" {
			t.Errorf("Role = %q, want default %q", startEvent.Role, "assistant")
		}
	})

	t.Run("generates message ID when not provided", func(t *testing.T) {
		converter := NewEventConverter("thread", "run")

		update := agent.ResponseUpdate{
			Kind: agent.UpdateKindContentDelta,
			Delta: &agent.ContentDelta{
				TextDelta: "Hello",
			},
		}

		events := converter.Convert(update)

		startEvent := events[0].(*TextMessageStartEvent)
		if startEvent.MessageID == "" {
			t.Error("MessageID should not be empty")
		}
	})

	t.Run("returns nil for nil delta", func(t *testing.T) {
		converter := NewEventConverter("thread", "run")

		update := agent.ResponseUpdate{
			Kind:  agent.UpdateKindContentDelta,
			Delta: nil,
		}

		events := converter.Convert(update)

		if events != nil {
			t.Errorf("expected nil, got %v", events)
		}
	})
}

func TestEventConverter_Convert_ContentDelta_ArgsDelta(t *testing.T) {
	t.Run("creates start and args events for new tool call", func(t *testing.T) {
		converter := NewEventConverter("thread", "run")

		update := agent.ResponseUpdate{
			Kind: agent.UpdateKindContentDelta,
			Delta: &agent.ContentDelta{
				ToolCallID: "tool-1",
				Name:       "get_weather",
				ArgsDelta:  `{"city":`,
			},
		}

		events := converter.Convert(update)

		if len(events) != 2 {
			t.Fatalf("expected 2 events, got %d", len(events))
		}

		startEvent, ok := events[0].(*ToolCallStartEvent)
		if !ok {
			t.Fatalf("expected ToolCallStartEvent, got %T", events[0])
		}
		if startEvent.ToolCallID != "tool-1" {
			t.Errorf("ToolCallID = %q, want %q", startEvent.ToolCallID, "tool-1")
		}
		if startEvent.ToolCallName != "get_weather" {
			t.Errorf("ToolCallName = %q, want %q", startEvent.ToolCallName, "get_weather")
		}

		argsEvent, ok := events[1].(*ToolCallArgsEvent)
		if !ok {
			t.Fatalf("expected ToolCallArgsEvent, got %T", events[1])
		}
		if argsEvent.Delta != `{"city":` {
			t.Errorf("Delta = %q, want %q", argsEvent.Delta, `{"city":`)
		}
	})

	t.Run("creates only args event for existing tool call", func(t *testing.T) {
		converter := NewEventConverter("thread", "run")
		converter.activeToolCalls["tool-1"] = "get_weather"

		update := agent.ResponseUpdate{
			Kind: agent.UpdateKindContentDelta,
			Delta: &agent.ContentDelta{
				ToolCallID: "tool-1",
				ArgsDelta:  `"Seattle"}`,
			},
		}

		events := converter.Convert(update)

		if len(events) != 1 {
			t.Fatalf("expected 1 event, got %d", len(events))
		}

		argsEvent, ok := events[0].(*ToolCallArgsEvent)
		if !ok {
			t.Fatalf("expected ToolCallArgsEvent, got %T", events[0])
		}
		if argsEvent.Delta != `"Seattle"}` {
			t.Errorf("Delta = %q, want %q", argsEvent.Delta, `"Seattle"}`)
		}
	})
}

func TestEventConverter_Convert_ToolCall(t *testing.T) {
	t.Run("creates start event for new tool call", func(t *testing.T) {
		converter := NewEventConverter("thread", "run")

		update := agent.ResponseUpdate{
			Kind: agent.UpdateKindToolCall,
			Delta: &agent.ContentDelta{
				ToolCallID: "tool-1",
				Name:       "calculator",
			},
		}

		events := converter.Convert(update)

		if len(events) != 1 {
			t.Fatalf("expected 1 event, got %d", len(events))
		}

		startEvent, ok := events[0].(*ToolCallStartEvent)
		if !ok {
			t.Fatalf("expected ToolCallStartEvent, got %T", events[0])
		}
		if startEvent.ToolCallID != "tool-1" {
			t.Errorf("ToolCallID = %q, want %q", startEvent.ToolCallID, "tool-1")
		}
	})

	t.Run("includes args event when arguments present", func(t *testing.T) {
		converter := NewEventConverter("thread", "run")

		update := agent.ResponseUpdate{
			Kind: agent.UpdateKindToolCall,
			Delta: &agent.ContentDelta{
				ToolCallID: "tool-1",
				Name:       "calculator",
				ArgsDelta:  `{"op": "add"}`,
			},
		}

		events := converter.Convert(update)

		if len(events) != 2 {
			t.Fatalf("expected 2 events, got %d", len(events))
		}

		argsEvent, ok := events[1].(*ToolCallArgsEvent)
		if !ok {
			t.Fatalf("expected ToolCallArgsEvent, got %T", events[1])
		}
		if argsEvent.Delta != `{"op": "add"}` {
			t.Errorf("Delta = %q, want %q", argsEvent.Delta, `{"op": "add"}`)
		}
	})

	t.Run("returns nil for nil delta", func(t *testing.T) {
		converter := NewEventConverter("thread", "run")

		update := agent.ResponseUpdate{
			Kind:  agent.UpdateKindToolCall,
			Delta: nil,
		}

		events := converter.Convert(update)

		if events != nil {
			t.Errorf("expected nil, got %v", events)
		}
	})

	t.Run("returns nil for empty tool call ID", func(t *testing.T) {
		converter := NewEventConverter("thread", "run")

		update := agent.ResponseUpdate{
			Kind: agent.UpdateKindToolCall,
			Delta: &agent.ContentDelta{
				Name: "test",
			},
		}

		events := converter.Convert(update)

		if events != nil {
			t.Errorf("expected nil, got %v", events)
		}
	})
}

func TestEventConverter_Convert_ToolResult(t *testing.T) {
	t.Run("creates end and result events for active tool call", func(t *testing.T) {
		converter := NewEventConverter("thread", "run")
		converter.activeToolCalls["tool-1"] = "get_weather"

		msg := chat.NewToolMessage("tool-1", "72°F and sunny")

		update := agent.ResponseUpdate{
			Kind:      agent.UpdateKindToolResult,
			Message:   &msg,
			MessageID: "result-msg-1",
		}

		events := converter.Convert(update)

		if len(events) != 2 {
			t.Fatalf("expected 2 events, got %d", len(events))
		}

		endEvent, ok := events[0].(*ToolCallEndEvent)
		if !ok {
			t.Fatalf("expected ToolCallEndEvent, got %T", events[0])
		}
		if endEvent.ToolCallID != "tool-1" {
			t.Errorf("ToolCallID = %q, want %q", endEvent.ToolCallID, "tool-1")
		}

		resultEvent, ok := events[1].(*ToolCallResultEvent)
		if !ok {
			t.Fatalf("expected ToolCallResultEvent, got %T", events[1])
		}
		if resultEvent.Content != "72°F and sunny" {
			t.Errorf("Content = %q, want %q", resultEvent.Content, "72°F and sunny")
		}
		if resultEvent.MessageID != "result-msg-1" {
			t.Errorf("MessageID = %q, want %q", resultEvent.MessageID, "result-msg-1")
		}
	})

	t.Run("creates only result event when tool call not active", func(t *testing.T) {
		converter := NewEventConverter("thread", "run")

		msg := chat.NewToolMessage("tool-1", "result")

		update := agent.ResponseUpdate{
			Kind:    agent.UpdateKindToolResult,
			Message: &msg,
		}

		events := converter.Convert(update)

		if len(events) != 1 {
			t.Fatalf("expected 1 event, got %d", len(events))
		}

		resultEvent, ok := events[0].(*ToolCallResultEvent)
		if !ok {
			t.Fatalf("expected ToolCallResultEvent, got %T", events[0])
		}
		if resultEvent.ToolCallID != "tool-1" {
			t.Errorf("ToolCallID = %q, want %q", resultEvent.ToolCallID, "tool-1")
		}
	})

	t.Run("returns nil for nil message", func(t *testing.T) {
		converter := NewEventConverter("thread", "run")

		update := agent.ResponseUpdate{
			Kind:    agent.UpdateKindToolResult,
			Message: nil,
		}

		events := converter.Convert(update)

		if events != nil {
			t.Errorf("expected nil, got %v", events)
		}
	})
}

func TestEventConverter_Convert_MessageComplete(t *testing.T) {
	t.Run("creates full lifecycle for non-streamed message", func(t *testing.T) {
		converter := NewEventConverter("thread", "run")

		msg := chat.NewAssistantMessage("Hello, world!")

		update := agent.ResponseUpdate{
			Kind:      agent.UpdateKindMessageComplete,
			Message:   &msg,
			MessageID: "msg-1",
		}

		events := converter.Convert(update)

		if len(events) != 3 {
			t.Fatalf("expected 3 events, got %d", len(events))
		}

		if events[0].Type() != EventTypeTextMessageStart {
			t.Errorf("events[0].Type() = %q, want %q", events[0].Type(), EventTypeTextMessageStart)
		}
		if events[1].Type() != EventTypeTextMessageContent {
			t.Errorf("events[1].Type() = %q, want %q", events[1].Type(), EventTypeTextMessageContent)
		}
		if events[2].Type() != EventTypeTextMessageEnd {
			t.Errorf("events[2].Type() = %q, want %q", events[2].Type(), EventTypeTextMessageEnd)
		}
	})

	t.Run("creates only end event for streamed message", func(t *testing.T) {
		converter := NewEventConverter("thread", "run")
		converter.activeMessages["msg-1"] = "assistant"

		msg := chat.NewAssistantMessage("Hello, world!")

		update := agent.ResponseUpdate{
			Kind:      agent.UpdateKindMessageComplete,
			Message:   &msg,
			MessageID: "msg-1",
		}

		events := converter.Convert(update)

		if len(events) != 1 {
			t.Fatalf("expected 1 event, got %d", len(events))
		}

		if events[0].Type() != EventTypeTextMessageEnd {
			t.Errorf("events[0].Type() = %q, want %q", events[0].Type(), EventTypeTextMessageEnd)
		}
	})

	t.Run("handles message with tool calls", func(t *testing.T) {
		converter := NewEventConverter("thread", "run")

		toolCalls := []chat.ToolCall{
			{ID: "tool-1", Name: "get_weather", Arguments: json.RawMessage(`{"city": "Seattle"}`)},
		}
		msg := chat.NewAssistantMessageWithToolCalls(toolCalls)

		update := agent.ResponseUpdate{
			Kind:    agent.UpdateKindMessageComplete,
			Message: &msg,
		}

		events := converter.Convert(update)

		// Should have: start, args, end for the tool call
		if len(events) < 3 {
			t.Fatalf("expected at least 3 events, got %d", len(events))
		}

		var foundStart, foundArgs, foundEnd bool
		for _, e := range events {
			switch e.Type() {
			case EventTypeToolCallStart:
				foundStart = true
			case EventTypeToolCallArgs:
				foundArgs = true
			case EventTypeToolCallEnd:
				foundEnd = true
			}
		}

		if !foundStart {
			t.Error("expected ToolCallStartEvent")
		}
		if !foundArgs {
			t.Error("expected ToolCallArgsEvent")
		}
		if !foundEnd {
			t.Error("expected ToolCallEndEvent")
		}
	})

	t.Run("handles message with tool result content", func(t *testing.T) {
		converter := NewEventConverter("thread", "run")

		msg := chat.Message{
			Role: chat.RoleTool,
			Contents: []chat.Content{
				chat.NewToolResultContent("tool-1", "result data"),
			},
		}

		update := agent.ResponseUpdate{
			Kind:    agent.UpdateKindMessageComplete,
			Message: &msg,
		}

		events := converter.Convert(update)

		var foundResult bool
		for _, e := range events {
			if e.Type() == EventTypeToolCallResult {
				foundResult = true
				resultEvent := e.(*ToolCallResultEvent)
				if resultEvent.Content != "result data" {
					t.Errorf("Content = %q, want %q", resultEvent.Content, "result data")
				}
			}
		}

		if !foundResult {
			t.Error("expected ToolCallResultEvent")
		}
	})

	t.Run("returns nil for nil message", func(t *testing.T) {
		converter := NewEventConverter("thread", "run")

		update := agent.ResponseUpdate{
			Kind:    agent.UpdateKindMessageComplete,
			Message: nil,
		}

		events := converter.Convert(update)

		if events != nil {
			t.Errorf("expected nil, got %v", events)
		}
	})
}

func TestEventConverter_Convert_Error(t *testing.T) {
	t.Run("creates error event with message", func(t *testing.T) {
		converter := NewEventConverter("thread", "run")

		update := agent.ResponseUpdate{
			Kind:  agent.UpdateKindError,
			Error: errors.New("something went wrong"),
		}

		events := converter.Convert(update)

		if len(events) != 1 {
			t.Fatalf("expected 1 event, got %d", len(events))
		}

		errorEvent, ok := events[0].(*RunErrorEvent)
		if !ok {
			t.Fatalf("expected RunErrorEvent, got %T", events[0])
		}
		if errorEvent.Message != "something went wrong" {
			t.Errorf("Message = %q, want %q", errorEvent.Message, "something went wrong")
		}
	})

	t.Run("handles nil error", func(t *testing.T) {
		converter := NewEventConverter("thread", "run")

		update := agent.ResponseUpdate{
			Kind:  agent.UpdateKindError,
			Error: nil,
		}

		events := converter.Convert(update)

		if len(events) != 1 {
			t.Fatalf("expected 1 event, got %d", len(events))
		}

		errorEvent := events[0].(*RunErrorEvent)
		if errorEvent.Message != "unknown error" {
			t.Errorf("Message = %q, want %q", errorEvent.Message, "unknown error")
		}
	})
}

func TestEventConverter_Convert_Done(t *testing.T) {
	t.Run("flushes active messages and creates finished event", func(t *testing.T) {
		converter := NewEventConverter("thread-123", "run-456")
		converter.activeMessages["msg-1"] = "assistant"

		update := agent.ResponseUpdate{
			Kind: agent.UpdateKindDone,
		}

		events := converter.Convert(update)

		// Should have: message end, run finished
		if len(events) != 2 {
			t.Fatalf("expected 2 events, got %d", len(events))
		}

		if events[0].Type() != EventTypeTextMessageEnd {
			t.Errorf("events[0].Type() = %q, want %q", events[0].Type(), EventTypeTextMessageEnd)
		}
		if events[1].Type() != EventTypeRunFinished {
			t.Errorf("events[1].Type() = %q, want %q", events[1].Type(), EventTypeRunFinished)
		}
	})

	t.Run("flushes active tool calls and creates finished event", func(t *testing.T) {
		converter := NewEventConverter("thread-123", "run-456")
		converter.activeToolCalls["tool-1"] = "test"

		update := agent.ResponseUpdate{
			Kind: agent.UpdateKindDone,
		}

		events := converter.Convert(update)

		// Should have: tool call end, run finished
		if len(events) != 2 {
			t.Fatalf("expected 2 events, got %d", len(events))
		}

		if events[0].Type() != EventTypeToolCallEnd {
			t.Errorf("events[0].Type() = %q, want %q", events[0].Type(), EventTypeToolCallEnd)
		}
		if events[1].Type() != EventTypeRunFinished {
			t.Errorf("events[1].Type() = %q, want %q", events[1].Type(), EventTypeRunFinished)
		}
	})

	t.Run("creates only finished event when nothing active", func(t *testing.T) {
		converter := NewEventConverter("thread-123", "run-456")

		update := agent.ResponseUpdate{
			Kind: agent.UpdateKindDone,
		}

		events := converter.Convert(update)

		if len(events) != 1 {
			t.Fatalf("expected 1 event, got %d", len(events))
		}

		if events[0].Type() != EventTypeRunFinished {
			t.Errorf("events[0].Type() = %q, want %q", events[0].Type(), EventTypeRunFinished)
		}
	})
}

func TestEventConverter_Convert_Usage(t *testing.T) {
	t.Run("returns nil for usage updates", func(t *testing.T) {
		converter := NewEventConverter("thread", "run")

		update := agent.ResponseUpdate{
			Kind: agent.UpdateKindUsage,
			Usage: &agent.UsageDetails{
				InputTokens:  100,
				OutputTokens: 50,
			},
		}

		events := converter.Convert(update)

		if events != nil {
			t.Errorf("expected nil, got %v", events)
		}
	})
}

func TestEventConverter_ConvertAll(t *testing.T) {
	t.Run("converts multiple updates", func(t *testing.T) {
		converter := NewEventConverter("thread", "run")

		msg := chat.NewAssistantMessage("Hello!")

		updates := []agent.ResponseUpdate{
			{
				Kind:      agent.UpdateKindContentDelta,
				MessageID: "msg-1",
				Delta:     &agent.ContentDelta{TextDelta: "Hello", Role: "assistant"},
			},
			{
				Kind:      agent.UpdateKindContentDelta,
				MessageID: "msg-1",
				Delta:     &agent.ContentDelta{TextDelta: "!"},
			},
			{
				Kind:      agent.UpdateKindMessageComplete,
				Message:   &msg,
				MessageID: "msg-1",
			},
			{
				Kind: agent.UpdateKindDone,
			},
		}

		events := converter.ConvertAll(updates)

		// Expect: start, content, content, end, finished
		if len(events) < 4 {
			t.Errorf("expected at least 4 events, got %d", len(events))
		}

		// Last event should be RunFinished
		lastEvent := events[len(events)-1]
		if lastEvent.Type() != EventTypeRunFinished {
			t.Errorf("last event Type() = %q, want %q", lastEvent.Type(), EventTypeRunFinished)
		}
	})

	t.Run("handles empty updates slice", func(t *testing.T) {
		converter := NewEventConverter("thread", "run")

		events := converter.ConvertAll(nil)

		if events != nil {
			t.Errorf("expected nil, got %v", events)
		}
	})
}

func TestEventConverter_FlushActiveMessages(t *testing.T) {
	t.Run("generates end events for all active messages", func(t *testing.T) {
		converter := NewEventConverter("thread", "run")
		converter.activeMessages["msg-1"] = "assistant"
		converter.activeMessages["msg-2"] = "user"

		events := converter.FlushActiveMessages()

		if len(events) != 2 {
			t.Fatalf("expected 2 events, got %d", len(events))
		}

		for _, e := range events {
			if e.Type() != EventTypeTextMessageEnd {
				t.Errorf("Type() = %q, want %q", e.Type(), EventTypeTextMessageEnd)
			}
		}

		if len(converter.activeMessages) != 0 {
			t.Errorf("activeMessages should be empty after flush")
		}
	})

	t.Run("returns nil when no active messages", func(t *testing.T) {
		converter := NewEventConverter("thread", "run")

		events := converter.FlushActiveMessages()

		if events != nil {
			t.Errorf("expected nil, got %v", events)
		}
	})
}

func TestEventConverter_FlushActiveToolCalls(t *testing.T) {
	t.Run("generates end events for all active tool calls", func(t *testing.T) {
		converter := NewEventConverter("thread", "run")
		converter.activeToolCalls["tool-1"] = "get_weather"
		converter.activeToolCalls["tool-2"] = "calculator"

		events := converter.FlushActiveToolCalls()

		if len(events) != 2 {
			t.Fatalf("expected 2 events, got %d", len(events))
		}

		for _, e := range events {
			if e.Type() != EventTypeToolCallEnd {
				t.Errorf("Type() = %q, want %q", e.Type(), EventTypeToolCallEnd)
			}
		}

		if len(converter.activeToolCalls) != 0 {
			t.Errorf("activeToolCalls should be empty after flush")
		}
	})

	t.Run("returns nil when no active tool calls", func(t *testing.T) {
		converter := NewEventConverter("thread", "run")

		events := converter.FlushActiveToolCalls()

		if events != nil {
			t.Errorf("expected nil, got %v", events)
		}
	})
}

func TestEventConverter_CompleteStreamingScenario(t *testing.T) {
	t.Run("simulates complete streaming response", func(t *testing.T) {
		converter := NewEventConverter("thread-123", "run-456")

		// Collect all events from a typical streaming scenario
		var allEvents []Event

		// 1. Run started
		allEvents = append(allEvents, converter.RunStarted())

		// 2. Text deltas
		updates := []agent.ResponseUpdate{
			{Kind: agent.UpdateKindContentDelta, MessageID: "msg-1", Delta: &agent.ContentDelta{TextDelta: "Hello", Role: "assistant"}},
			{Kind: agent.UpdateKindContentDelta, MessageID: "msg-1", Delta: &agent.ContentDelta{TextDelta: ", "}},
			{Kind: agent.UpdateKindContentDelta, MessageID: "msg-1", Delta: &agent.ContentDelta{TextDelta: "world!"}},
		}

		for _, u := range updates {
			allEvents = append(allEvents, converter.Convert(u)...)
		}

		// 3. Message complete and done
		msg := chat.NewAssistantMessage("Hello, world!")
		allEvents = append(allEvents, converter.Convert(agent.ResponseUpdate{
			Kind:      agent.UpdateKindMessageComplete,
			Message:   &msg,
			MessageID: "msg-1",
		})...)
		allEvents = append(allEvents, converter.Convert(agent.ResponseUpdate{Kind: agent.UpdateKindDone})...)

		// Verify event sequence
		expectedTypes := []EventType{
			EventTypeRunStarted,
			EventTypeTextMessageStart,
			EventTypeTextMessageContent, // "Hello"
			EventTypeTextMessageContent, // ", "
			EventTypeTextMessageContent, // "world!"
			EventTypeTextMessageEnd,
			EventTypeRunFinished,
		}

		if len(allEvents) != len(expectedTypes) {
			t.Fatalf("expected %d events, got %d", len(expectedTypes), len(allEvents))
		}

		for i, expected := range expectedTypes {
			if allEvents[i].Type() != expected {
				t.Errorf("event[%d].Type() = %q, want %q", i, allEvents[i].Type(), expected)
			}
		}
	})

	t.Run("simulates tool call scenario", func(t *testing.T) {
		converter := NewEventConverter("thread", "run")

		var allEvents []Event
		allEvents = append(allEvents, converter.RunStarted())

		// Tool call start
		allEvents = append(allEvents, converter.Convert(agent.ResponseUpdate{
			Kind: agent.UpdateKindContentDelta,
			Delta: &agent.ContentDelta{
				ToolCallID: "call-1",
				Name:       "get_weather",
				ArgsDelta:  `{"city":`,
			},
		})...)

		// More args
		allEvents = append(allEvents, converter.Convert(agent.ResponseUpdate{
			Kind: agent.UpdateKindContentDelta,
			Delta: &agent.ContentDelta{
				ToolCallID: "call-1",
				ArgsDelta:  `"Seattle"}`,
			},
		})...)

		// Tool result
		resultMsg := chat.NewToolMessage("call-1", "72°F")
		allEvents = append(allEvents, converter.Convert(agent.ResponseUpdate{
			Kind:    agent.UpdateKindToolResult,
			Message: &resultMsg,
		})...)

		// Done
		allEvents = append(allEvents, converter.Convert(agent.ResponseUpdate{Kind: agent.UpdateKindDone})...)

		// Verify we got the expected flow
		var hasToolStart, hasToolArgs, hasToolEnd, hasToolResult, hasRunFinished bool
		for _, e := range allEvents {
			switch e.Type() {
			case EventTypeToolCallStart:
				hasToolStart = true
			case EventTypeToolCallArgs:
				hasToolArgs = true
			case EventTypeToolCallEnd:
				hasToolEnd = true
			case EventTypeToolCallResult:
				hasToolResult = true
			case EventTypeRunFinished:
				hasRunFinished = true
			}
		}

		if !hasToolStart {
			t.Error("missing ToolCallStart event")
		}
		if !hasToolArgs {
			t.Error("missing ToolCallArgs event")
		}
		if !hasToolEnd {
			t.Error("missing ToolCallEnd event")
		}
		if !hasToolResult {
			t.Error("missing ToolCallResult event")
		}
		if !hasRunFinished {
			t.Error("missing RunFinished event")
		}
	})
}
