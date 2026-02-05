// Copyright (c) Microsoft. All rights reserved.

package chatagent

import (
	"context"
	"sync"
	"testing"

	"github.com/microsoft/agent-framework-go/agent"
	"github.com/microsoft/agent-framework-go/chat"
)

// recordingChatMiddleware records invocations for testing.
type recordingChatMiddleware struct {
	mu          sync.Mutex
	invocations []string
	messages    [][]agent.Message
}

func (m *recordingChatMiddleware) Process(ctx context.Context, chatCtx *agent.ChatContext, next agent.ChatHandler) error {
	m.mu.Lock()
	m.invocations = append(m.invocations, "before")
	m.messages = append(m.messages, chatCtx.Messages)
	m.mu.Unlock()

	err := next(ctx, chatCtx)

	m.mu.Lock()
	m.invocations = append(m.invocations, "after")
	m.mu.Unlock()

	return err
}

func TestChatMiddleware_InvokedOnGetResponse(t *testing.T) {
	client := newMockClient()
	mw := &recordingChatMiddleware{}

	a := New(client, WithChatMiddleware(mw))

	messages := []agent.Message{
		agent.NewUserMessage("Hello"),
	}

	_, err := a.Run(context.Background(), messages)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(mw.invocations) < 2 {
		t.Errorf("expected at least 2 invocations (before/after), got %d", len(mw.invocations))
	}
	if mw.invocations[0] != "before" {
		t.Errorf("expected first invocation 'before', got %q", mw.invocations[0])
	}
	if mw.invocations[1] != "after" {
		t.Errorf("expected second invocation 'after', got %q", mw.invocations[1])
	}
}

func TestChatMiddleware_InvokedOnGetStreamingResponse(t *testing.T) {
	client := newMockClient()
	mw := &recordingChatMiddleware{}

	a := New(client, WithChatMiddleware(mw))

	messages := []agent.Message{
		agent.NewUserMessage("Hello"),
	}

	updates, err := a.RunStream(context.Background(), messages)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Drain the stream
	for range updates {
		// intentionally empty - drain streaming updates
	}

	if len(mw.invocations) < 2 {
		t.Errorf("expected at least 2 invocations (before/after), got %d", len(mw.invocations))
	}
}

func TestChatMiddleware_ReceivesMessages(t *testing.T) {
	client := newMockClient()
	mw := &recordingChatMiddleware{}

	a := New(client, WithChatMiddleware(mw))

	messages := []agent.Message{
		agent.NewUserMessage("Test message"),
	}

	_, err := a.Run(context.Background(), messages)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(mw.messages) == 0 {
		t.Fatal("expected messages to be recorded")
	}
	// The first recorded messages should be the input (plus any system message)
	if len(mw.messages[0]) == 0 {
		t.Error("expected non-empty messages slice")
	}
}

// cachingChatMiddleware caches responses to avoid calling next.
type cachingChatMiddleware struct {
	cache map[string]*agent.ChatResponse
}

func (m *cachingChatMiddleware) Process(ctx context.Context, chatCtx *agent.ChatContext, next agent.ChatHandler) error {
	// Simple cache key from message count
	key := "cache_key"

	if cached, ok := m.cache[key]; ok {
		chatCtx.Response = cached
		return nil // Short-circuit
	}

	err := next(ctx, chatCtx)
	if err != nil {
		return err
	}

	// Cache the response
	if chatCtx.Response != nil {
		m.cache[key] = chatCtx.Response
	}
	return nil
}

func TestChatMiddleware_CanShortCircuit(t *testing.T) {
	callCount := 0
	client := newMockClient()
	client.getResponseFn = func(ctx context.Context, messages []chat.Message, options *chat.Options) (*chat.Response, error) {
		callCount++
		return &chat.Response{
			Message: chat.NewAssistantMessage("Response " + string(rune('0'+callCount))),
			Usage:   &chat.UsageDetails{TotalTokens: 10},
		}, nil
	}

	mw := &cachingChatMiddleware{
		cache: map[string]*agent.ChatResponse{
			"cache_key": {
				Text:         "Cached response",
				FinishReason: "stop",
				RawResponse: &chat.Response{
					Message: chat.NewAssistantMessage("Cached response"),
				},
			},
		},
	}

	a := New(client, WithChatMiddleware(mw))

	messages := []agent.Message{
		agent.NewUserMessage("Test"),
	}

	resp, err := a.Run(context.Background(), messages)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Client should not have been called due to cache hit
	if callCount != 0 {
		t.Errorf("expected 0 client calls (cached), got %d", callCount)
	}

	// Should get cached response
	if resp.Text() != "Cached response" {
		t.Errorf("expected 'Cached response', got %q", resp.Text())
	}
}

func TestChatMiddleware_ExecutionOrder(t *testing.T) {
	order := []string{}
	mu := sync.Mutex{}

	mw1 := agent.ChatMiddlewareFunc(func(ctx context.Context, chatCtx *agent.ChatContext, next agent.ChatHandler) error {
		mu.Lock()
		order = append(order, "mw1-before")
		mu.Unlock()
		err := next(ctx, chatCtx)
		mu.Lock()
		order = append(order, "mw1-after")
		mu.Unlock()
		return err
	})

	mw2 := agent.ChatMiddlewareFunc(func(ctx context.Context, chatCtx *agent.ChatContext, next agent.ChatHandler) error {
		mu.Lock()
		order = append(order, "mw2-before")
		mu.Unlock()
		err := next(ctx, chatCtx)
		mu.Lock()
		order = append(order, "mw2-after")
		mu.Unlock()
		return err
	})

	client := newMockClient()
	a := New(client, WithChatMiddleware(mw1, mw2))

	messages := []agent.Message{
		agent.NewUserMessage("Test"),
	}

	_, err := a.Run(context.Background(), messages)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// First middleware should run first (outermost)
	expected := []string{"mw1-before", "mw2-before", "mw2-after", "mw1-after"}
	if len(order) != len(expected) {
		t.Errorf("order = %v, want %v", order, expected)
	}
	for i, v := range expected {
		if i < len(order) && order[i] != v {
			t.Errorf("order[%d] = %q, want %q", i, order[i], v)
		}
	}
}

func TestChatMiddleware_MetadataFlows(t *testing.T) {
	var receivedValue string

	mw1 := agent.ChatMiddlewareFunc(func(ctx context.Context, chatCtx *agent.ChatContext, next agent.ChatHandler) error {
		chatCtx.Metadata["key1"] = "value1"
		return next(ctx, chatCtx)
	})

	mw2 := agent.ChatMiddlewareFunc(func(ctx context.Context, chatCtx *agent.ChatContext, next agent.ChatHandler) error {
		if v, ok := chatCtx.Metadata["key1"].(string); ok {
			receivedValue = v
		}
		return next(ctx, chatCtx)
	})

	client := newMockClient()
	a := New(client, WithChatMiddleware(mw1, mw2))

	messages := []agent.Message{
		agent.NewUserMessage("Test"),
	}

	_, err := a.Run(context.Background(), messages)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if receivedValue != "value1" {
		t.Errorf("expected metadata value 'value1', got %q", receivedValue)
	}
}

func TestBuilder_UseChatMiddleware(t *testing.T) {
	mw := &recordingChatMiddleware{}
	client := newMockClient()

	a, err := NewBuilder(client).
		UseChatMiddleware(mw).
		Build()

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	messages := []agent.Message{
		agent.NewUserMessage("Test"),
	}

	_, err = a.Run(context.Background(), messages)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(mw.invocations) == 0 {
		t.Error("expected middleware to be invoked")
	}
}

func TestChatMiddleware_StreamingReceivesContext(t *testing.T) {
	var isStreaming bool

	mw := agent.ChatMiddlewareFunc(func(ctx context.Context, chatCtx *agent.ChatContext, next agent.ChatHandler) error {
		isStreaming = chatCtx.IsStreaming
		return next(ctx, chatCtx)
	})

	client := newMockClient()
	a := New(client, WithChatMiddleware(mw))

	messages := []agent.Message{
		agent.NewUserMessage("Test"),
	}

	updates, err := a.RunStream(context.Background(), messages)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Drain the stream
	for range updates {
		// intentionally empty - drain streaming updates
	}

	if !isStreaming {
		t.Error("expected IsStreaming to be true for streaming requests")
	}
}
