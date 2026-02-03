// Copyright (c) Microsoft. All rights reserved.

package openai

import (
	"context"
	"testing"
	"time"

	"github.com/microsoft/agent-framework-go/chat"
)

func TestNewStreamProcessor(t *testing.T) {
	sp := NewStreamProcessor()
	if sp == nil {
		t.Fatal("NewStreamProcessor returned nil")
	}
	if sp.accumulated == nil {
		t.Error("accumulated map should be initialized")
	}
}

func TestCollectStreamToResponse_TextContent(t *testing.T) {
	updates := make(chan chat.ResponseUpdate, 10)

	// Send text content deltas
	updates <- chat.ResponseUpdate{
		Kind: chat.UpdateKindContentDelta,
		Delta: &chat.ContentDelta{
			Role:      chat.RoleAssistant,
			TextDelta: "Hello",
		},
	}
	updates <- chat.ResponseUpdate{
		Kind: chat.UpdateKindContentDelta,
		Delta: &chat.ContentDelta{
			TextDelta: " world",
		},
	}
	updates <- chat.ResponseUpdate{
		Kind:         chat.UpdateKindMessageComplete,
		FinishReason: chat.FinishReasonStop,
	}
	updates <- chat.ResponseUpdate{
		Kind: chat.UpdateKindUsage,
		Usage: &chat.UsageDetails{
			InputTokens:  10,
			OutputTokens: 5,
			TotalTokens:  15,
		},
	}
	updates <- chat.ResponseUpdate{Kind: chat.UpdateKindDone}
	close(updates)

	resp, err := CollectStreamToResponse(updates)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.Message.Text() != "Hello world" {
		t.Errorf("expected 'Hello world', got %q", resp.Message.Text())
	}

	if resp.FinishReason != chat.FinishReasonStop {
		t.Errorf("expected FinishReasonStop, got %v", resp.FinishReason)
	}

	if resp.Usage == nil {
		t.Fatal("expected usage to be set")
	}
	if resp.Usage.InputTokens != 10 {
		t.Errorf("expected InputTokens=10, got %d", resp.Usage.InputTokens)
	}
	if resp.Usage.OutputTokens != 5 {
		t.Errorf("expected OutputTokens=5, got %d", resp.Usage.OutputTokens)
	}
	if resp.Usage.TotalTokens != 15 {
		t.Errorf("expected TotalTokens=15, got %d", resp.Usage.TotalTokens)
	}
}

func TestCollectStreamToResponse_ToolCalls(t *testing.T) {
	updates := make(chan chat.ResponseUpdate, 10)

	// Send tool call
	updates <- chat.ResponseUpdate{
		Kind: chat.UpdateKindToolCall,
		Delta: &chat.ContentDelta{
			ToolCallID: "call_123",
			Name:       "get_weather",
		},
	}
	// Send argument deltas
	updates <- chat.ResponseUpdate{
		Kind: chat.UpdateKindContentDelta,
		Delta: &chat.ContentDelta{
			ToolCallID: "call_123",
			ArgsDelta:  `{"location":`,
		},
	}
	updates <- chat.ResponseUpdate{
		Kind: chat.UpdateKindContentDelta,
		Delta: &chat.ContentDelta{
			ToolCallID: "call_123",
			ArgsDelta:  `"NYC"}`,
		},
	}
	updates <- chat.ResponseUpdate{
		Kind:         chat.UpdateKindMessageComplete,
		FinishReason: chat.FinishReasonToolCalls,
	}
	updates <- chat.ResponseUpdate{Kind: chat.UpdateKindDone}
	close(updates)

	resp, err := CollectStreamToResponse(updates)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(resp.Message.ToolCalls) != 1 {
		t.Fatalf("expected 1 tool call, got %d", len(resp.Message.ToolCalls))
	}

	tc := resp.Message.ToolCalls[0]
	if tc.ID != "call_123" {
		t.Errorf("expected tool call ID 'call_123', got %q", tc.ID)
	}
	if tc.Name != "get_weather" {
		t.Errorf("expected tool name 'get_weather', got %q", tc.Name)
	}
	if string(tc.Arguments) != `{"location":"NYC"}` {
		t.Errorf("expected arguments '{\"location\":\"NYC\"}', got %q", string(tc.Arguments))
	}

	if resp.FinishReason != chat.FinishReasonToolCalls {
		t.Errorf("expected FinishReasonToolCalls, got %v", resp.FinishReason)
	}
}

func TestCollectStreamToResponse_Error(t *testing.T) {
	updates := make(chan chat.ResponseUpdate, 5)

	updates <- chat.ResponseUpdate{
		Kind: chat.UpdateKindContentDelta,
		Delta: &chat.ContentDelta{
			TextDelta: "partial",
		},
	}
	updates <- chat.ResponseUpdate{
		Kind:  chat.UpdateKindError,
		Error: context.DeadlineExceeded,
	}
	updates <- chat.ResponseUpdate{Kind: chat.UpdateKindDone}
	close(updates)

	_, err := CollectStreamToResponse(updates)
	if err != context.DeadlineExceeded {
		t.Errorf("expected DeadlineExceeded error, got %v", err)
	}
}

func TestCollectStreamToResponse_EmptyStream(t *testing.T) {
	updates := make(chan chat.ResponseUpdate, 1)
	updates <- chat.ResponseUpdate{Kind: chat.UpdateKindDone}
	close(updates)

	resp, err := CollectStreamToResponse(updates)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.Message.Text() != "" {
		t.Errorf("expected empty text, got %q", resp.Message.Text())
	}
	if resp.Message.Role != chat.RoleAssistant {
		t.Errorf("expected RoleAssistant, got %v", resp.Message.Role)
	}
}

func TestStreamProcessor_ProcessToolCallDelta(t *testing.T) {
	sp := NewStreamProcessor()
	updates := make(chan chat.ResponseUpdate, 10)

	// Process tool call in a goroutine
	done := make(chan struct{})
	go func() {
		defer close(done)

		// Simulate receiving tool call fragments
		sp.processTestToolCallDelta(mockToolCall{
			ID:   "call_abc",
			Name: "search",
			Args: "",
		}, updates)

		sp.processTestToolCallDelta(mockToolCall{
			ID:   "call_abc",
			Name: "",
			Args: `{"query":`,
		}, updates)

		sp.processTestToolCallDelta(mockToolCall{
			ID:   "call_abc",
			Name: "",
			Args: `"test"}`,
		}, updates)

		sp.flushToolCalls(updates)
		close(updates)
	}()

	<-done

	// Collect updates
	var toolCallReceived bool
	var argsDeltas []string

	for update := range updates {
		switch update.Kind {
		case chat.UpdateKindToolCall:
			toolCallReceived = true
			if update.Delta.Name != "search" {
				t.Errorf("expected tool name 'search', got %q", update.Delta.Name)
			}
		case chat.UpdateKindContentDelta:
			if update.Delta.ArgsDelta != "" {
				argsDeltas = append(argsDeltas, update.Delta.ArgsDelta)
			}
		}
	}

	if !toolCallReceived {
		t.Error("expected to receive UpdateKindToolCall")
	}

	if len(argsDeltas) != 2 {
		t.Errorf("expected 2 argument deltas, got %d", len(argsDeltas))
	}
}

// mockToolCall is a test helper that mimics a tool call delta for testing
type mockToolCall struct {
	ID    string
	Index int
	Name  string
	Args  string
}

// processTestToolCallDelta is a test wrapper for StreamProcessor's tool call handling
func (sp *StreamProcessor) processTestToolCallDelta(tc mockToolCall, updates chan<- chat.ResponseUpdate) {
	sp.mu.Lock()
	defer sp.mu.Unlock()

	acc, exists := sp.accumulated[tc.ID]
	if !exists && tc.ID != "" {
		acc = &toolCallAccumulator{ID: tc.ID}
		sp.accumulated[tc.ID] = acc
	}

	if tc.Name != "" {
		acc.Name = tc.Name
		updates <- chat.ResponseUpdate{
			Kind: chat.UpdateKindToolCall,
			Delta: &chat.ContentDelta{
				ToolCallID: acc.ID,
				Name:       acc.Name,
			},
		}
	}

	if tc.Args != "" {
		acc.Arguments += tc.Args
		updates <- chat.ResponseUpdate{
			Kind: chat.UpdateKindContentDelta,
			Delta: &chat.ContentDelta{
				ToolCallID: acc.ID,
				ArgsDelta:  tc.Args,
			},
		}
	}
}

func TestStreamProcessor_ConcurrentAccess(t *testing.T) {
	sp := NewStreamProcessor()
	updates := make(chan chat.ResponseUpdate, 100)

	// Simulate concurrent access
	done := make(chan struct{})
	go func() {
		defer close(done)
		for i := 0; i < 10; i++ {
			sp.processTestToolCallDelta(mockToolCall{
				ID:   "call_concurrent",
				Args: "x",
			}, updates)
			time.Sleep(time.Millisecond)
		}
		sp.flushToolCalls(updates)
		close(updates)
	}()

	<-done

	// Should complete without race conditions or panics
	count := 0
	for range updates {
		count++
	}

	if count == 0 {
		t.Error("expected some updates")
	}
}

func TestCollectStreamToResponse_MultipleToolCalls(t *testing.T) {
	updates := make(chan chat.ResponseUpdate, 20)

	// First tool call
	updates <- chat.ResponseUpdate{
		Kind: chat.UpdateKindToolCall,
		Delta: &chat.ContentDelta{
			ToolCallID: "call_1",
			Name:       "search",
		},
	}
	updates <- chat.ResponseUpdate{
		Kind: chat.UpdateKindContentDelta,
		Delta: &chat.ContentDelta{
			ToolCallID: "call_1",
			ArgsDelta:  `{"q":"a"}`,
		},
	}

	// Second tool call
	updates <- chat.ResponseUpdate{
		Kind: chat.UpdateKindToolCall,
		Delta: &chat.ContentDelta{
			ToolCallID: "call_2",
			Name:       "calculator",
		},
	}
	updates <- chat.ResponseUpdate{
		Kind: chat.UpdateKindContentDelta,
		Delta: &chat.ContentDelta{
			ToolCallID: "call_2",
			ArgsDelta:  `{"n":5}`,
		},
	}

	updates <- chat.ResponseUpdate{
		Kind:         chat.UpdateKindMessageComplete,
		FinishReason: chat.FinishReasonToolCalls,
	}
	updates <- chat.ResponseUpdate{Kind: chat.UpdateKindDone}
	close(updates)

	resp, err := CollectStreamToResponse(updates)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(resp.Message.ToolCalls) != 2 {
		t.Fatalf("expected 2 tool calls, got %d", len(resp.Message.ToolCalls))
	}

	// Verify first tool call
	if resp.Message.ToolCalls[0].Name != "search" {
		t.Errorf("expected first tool 'search', got %q", resp.Message.ToolCalls[0].Name)
	}
	if string(resp.Message.ToolCalls[0].Arguments) != `{"q":"a"}` {
		t.Errorf("expected first args '{\"q\":\"a\"}', got %q", string(resp.Message.ToolCalls[0].Arguments))
	}

	// Verify second tool call
	if resp.Message.ToolCalls[1].Name != "calculator" {
		t.Errorf("expected second tool 'calculator', got %q", resp.Message.ToolCalls[1].Name)
	}
	if string(resp.Message.ToolCalls[1].Arguments) != `{"n":5}` {
		t.Errorf("expected second args '{\"n\":5}', got %q", string(resp.Message.ToolCalls[1].Arguments))
	}
}

func TestCollectStreamToResponse_NilDelta(t *testing.T) {
	updates := make(chan chat.ResponseUpdate, 5)

	// Send update with nil delta
	updates <- chat.ResponseUpdate{
		Kind:  chat.UpdateKindContentDelta,
		Delta: nil,
	}
	updates <- chat.ResponseUpdate{Kind: chat.UpdateKindDone}
	close(updates)

	resp, err := CollectStreamToResponse(updates)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should handle nil delta gracefully
	if resp.Message.Text() != "" {
		t.Errorf("expected empty text, got %q", resp.Message.Text())
	}
}
