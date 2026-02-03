// Copyright (c) Microsoft. All rights reserved.

package agent

import (
	"context"
	"errors"
	"testing"
)

func TestChatMiddlewareFunc_ImplementsInterface(t *testing.T) {
	// This is a compile-time check, but we test it explicitly for clarity
	var _ ChatMiddleware = ChatMiddlewareFunc(nil)
}

func TestChainChatMiddleware_Empty(t *testing.T) {
	chain := ChainChatMiddleware()
	called := false

	err := chain.Process(context.Background(), &ChatContext{}, func(ctx context.Context, chatCtx *ChatContext) error {
		called = true
		return nil
	})

	if err != nil {
		t.Errorf("ChainChatMiddleware() error = %v", err)
	}
	if !called {
		t.Error("ChainChatMiddleware() did not call next")
	}
}

func TestChainChatMiddleware_NilMiddleware(t *testing.T) {
	chain := ChainChatMiddleware(nil, nil)
	called := false

	err := chain.Process(context.Background(), &ChatContext{}, func(ctx context.Context, chatCtx *ChatContext) error {
		called = true
		return nil
	})

	if err != nil {
		t.Errorf("ChainChatMiddleware() with nils error = %v", err)
	}
	if !called {
		t.Error("ChainChatMiddleware() with nils did not call next")
	}
}

func TestChainChatMiddleware_Single(t *testing.T) {
	order := []string{}

	mw := ChatMiddlewareFunc(func(ctx context.Context, chatCtx *ChatContext, next ChatHandler) error {
		order = append(order, "mw1-before")
		err := next(ctx, chatCtx)
		order = append(order, "mw1-after")
		return err
	})

	chain := ChainChatMiddleware(mw)

	err := chain.Process(context.Background(), &ChatContext{}, func(ctx context.Context, chatCtx *ChatContext) error {
		order = append(order, "terminal")
		return nil
	})

	if err != nil {
		t.Errorf("ChainChatMiddleware() error = %v", err)
	}

	expected := []string{"mw1-before", "terminal", "mw1-after"}
	if len(order) != len(expected) {
		t.Errorf("order = %v, want %v", order, expected)
	}
	for i, v := range expected {
		if order[i] != v {
			t.Errorf("order[%d] = %q, want %q", i, order[i], v)
		}
	}
}

func TestChainChatMiddleware_Multiple(t *testing.T) {
	order := []string{}

	mw1 := ChatMiddlewareFunc(func(ctx context.Context, chatCtx *ChatContext, next ChatHandler) error {
		order = append(order, "mw1-before")
		err := next(ctx, chatCtx)
		order = append(order, "mw1-after")
		return err
	})

	mw2 := ChatMiddlewareFunc(func(ctx context.Context, chatCtx *ChatContext, next ChatHandler) error {
		order = append(order, "mw2-before")
		err := next(ctx, chatCtx)
		order = append(order, "mw2-after")
		return err
	})

	mw3 := ChatMiddlewareFunc(func(ctx context.Context, chatCtx *ChatContext, next ChatHandler) error {
		order = append(order, "mw3-before")
		err := next(ctx, chatCtx)
		order = append(order, "mw3-after")
		return err
	})

	chain := ChainChatMiddleware(mw1, mw2, mw3)

	err := chain.Process(context.Background(), &ChatContext{}, func(ctx context.Context, chatCtx *ChatContext) error {
		order = append(order, "terminal")
		return nil
	})

	if err != nil {
		t.Errorf("ChainChatMiddleware() error = %v", err)
	}

	// First middleware should run first (outermost)
	expected := []string{
		"mw1-before", "mw2-before", "mw3-before",
		"terminal",
		"mw3-after", "mw2-after", "mw1-after",
	}
	if len(order) != len(expected) {
		t.Errorf("order = %v, want %v", order, expected)
	}
	for i, v := range expected {
		if order[i] != v {
			t.Errorf("order[%d] = %q, want %q", i, order[i], v)
		}
	}
}

func TestChainChatMiddleware_ShortCircuit(t *testing.T) {
	nextCalled := false

	mw := ChatMiddlewareFunc(func(ctx context.Context, chatCtx *ChatContext, next ChatHandler) error {
		// Short-circuit by not calling next
		chatCtx.Response = &ChatResponse{Text: "cached"}
		return nil
	})

	chain := ChainChatMiddleware(mw)

	chatCtx := &ChatContext{}
	err := chain.Process(context.Background(), chatCtx, func(ctx context.Context, chatCtx *ChatContext) error {
		nextCalled = true
		return nil
	})

	if err != nil {
		t.Errorf("ChainChatMiddleware() error = %v", err)
	}
	if nextCalled {
		t.Error("next was called when it should have been short-circuited")
	}
	if chatCtx.Response == nil || chatCtx.Response.Text != "cached" {
		t.Error("response was not set by short-circuiting middleware")
	}
}

func TestChainChatMiddleware_ErrorPropagation(t *testing.T) {
	expectedErr := errors.New("test error")

	mw := ChatMiddlewareFunc(func(ctx context.Context, chatCtx *ChatContext, next ChatHandler) error {
		return expectedErr
	})

	chain := ChainChatMiddleware(mw)

	err := chain.Process(context.Background(), &ChatContext{}, func(ctx context.Context, chatCtx *ChatContext) error {
		return nil
	})

	if !errors.Is(err, expectedErr) {
		t.Errorf("error = %v, want %v", err, expectedErr)
	}
}

func TestChainChatMiddleware_ContextModification(t *testing.T) {
	// Test that middleware can modify context and changes flow through
	mw1 := ChatMiddlewareFunc(func(ctx context.Context, chatCtx *ChatContext, next ChatHandler) error {
		if chatCtx.Metadata == nil {
			chatCtx.Metadata = make(map[string]any)
		}
		chatCtx.Metadata["mw1"] = "value1"
		return next(ctx, chatCtx)
	})

	mw2 := ChatMiddlewareFunc(func(ctx context.Context, chatCtx *ChatContext, next ChatHandler) error {
		chatCtx.Metadata["mw2"] = "value2"
		return next(ctx, chatCtx)
	})

	chain := ChainChatMiddleware(mw1, mw2)

	chatCtx := &ChatContext{}
	err := chain.Process(context.Background(), chatCtx, func(ctx context.Context, chatCtx *ChatContext) error {
		// Verify both middleware values are present
		if chatCtx.Metadata["mw1"] != "value1" {
			t.Error("mw1 value not found")
		}
		if chatCtx.Metadata["mw2"] != "value2" {
			t.Error("mw2 value not found")
		}
		return nil
	})

	if err != nil {
		t.Errorf("ChainChatMiddleware() error = %v", err)
	}
}

func TestChatContext_StreamingFlag(t *testing.T) {
	// Test that IsStreaming flag is accessible
	ctx := &ChatContext{IsStreaming: true}
	if !ctx.IsStreaming {
		t.Error("IsStreaming should be true")
	}

	ctx.IsStreaming = false
	if ctx.IsStreaming {
		t.Error("IsStreaming should be false")
	}
}

func TestChatContext_Fields(t *testing.T) {
	// Test that all ChatContext fields are accessible
	ctx := &ChatContext{
		ClientMetadata: ChatClientMetadata{
			ProviderName: "openai",
			ModelID:      "gpt-4",
			EndpointURI:  "https://api.openai.com",
		},
		Messages: []Message{
			NewUserMessage("Hello"),
		},
		Options: map[string]any{
			"temperature": 0.7,
		},
		Metadata:    map[string]any{"key": "value"},
		IsStreaming: false,
	}

	if ctx.ClientMetadata.ProviderName != "openai" {
		t.Error("ProviderName not set correctly")
	}
	if ctx.ClientMetadata.ModelID != "gpt-4" {
		t.Error("ModelID not set correctly")
	}
	if len(ctx.Messages) != 1 {
		t.Error("Messages not set correctly")
	}
	if ctx.Options["temperature"] != 0.7 {
		t.Error("Options not set correctly")
	}
	if ctx.Metadata["key"] != "value" {
		t.Error("Metadata not set correctly")
	}
}
