// Copyright (c) Microsoft. All rights reserved.

package agent

import (
	"context"
	"testing"
)

func TestChainAgentMiddleware_Empty(t *testing.T) {
	chain := ChainAgentMiddleware()
	called := false

	err := chain.Process(context.Background(), &AgentContext{}, func(ctx context.Context, agentCtx *AgentContext) error {
		called = true
		return nil
	})

	if err != nil {
		t.Errorf("ChainAgentMiddleware() error = %v", err)
	}
	if !called {
		t.Error("ChainAgentMiddleware() did not call next")
	}
}

func TestChainAgentMiddleware_Single(t *testing.T) {
	order := []string{}

	mw := AgentMiddlewareFunc(func(ctx context.Context, agentCtx *AgentContext, next AgentHandler) error {
		order = append(order, "mw1-before")
		err := next(ctx, agentCtx)
		order = append(order, "mw1-after")
		return err
	})

	chain := ChainAgentMiddleware(mw)

	err := chain.Process(context.Background(), &AgentContext{}, func(ctx context.Context, agentCtx *AgentContext) error {
		order = append(order, "terminal")
		return nil
	})

	if err != nil {
		t.Errorf("ChainAgentMiddleware() error = %v", err)
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

func TestChainAgentMiddleware_Multiple(t *testing.T) {
	order := []string{}

	mw1 := AgentMiddlewareFunc(func(ctx context.Context, agentCtx *AgentContext, next AgentHandler) error {
		order = append(order, "mw1-before")
		err := next(ctx, agentCtx)
		order = append(order, "mw1-after")
		return err
	})

	mw2 := AgentMiddlewareFunc(func(ctx context.Context, agentCtx *AgentContext, next AgentHandler) error {
		order = append(order, "mw2-before")
		err := next(ctx, agentCtx)
		order = append(order, "mw2-after")
		return err
	})

	mw3 := AgentMiddlewareFunc(func(ctx context.Context, agentCtx *AgentContext, next AgentHandler) error {
		order = append(order, "mw3-before")
		err := next(ctx, agentCtx)
		order = append(order, "mw3-after")
		return err
	})

	chain := ChainAgentMiddleware(mw1, mw2, mw3)

	err := chain.Process(context.Background(), &AgentContext{}, func(ctx context.Context, agentCtx *AgentContext) error {
		order = append(order, "terminal")
		return nil
	})

	if err != nil {
		t.Errorf("ChainAgentMiddleware() error = %v", err)
	}

	// First middleware should run first
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

func TestChainAgentMiddleware_ShortCircuit(t *testing.T) {
	order := []string{}

	mw1 := AgentMiddlewareFunc(func(ctx context.Context, agentCtx *AgentContext, next AgentHandler) error {
		order = append(order, "mw1-before")
		// Don't call next - short-circuit
		order = append(order, "mw1-after")
		return nil
	})

	mw2 := AgentMiddlewareFunc(func(ctx context.Context, agentCtx *AgentContext, next AgentHandler) error {
		order = append(order, "mw2")
		return next(ctx, agentCtx)
	})

	chain := ChainAgentMiddleware(mw1, mw2)

	err := chain.Process(context.Background(), &AgentContext{}, func(ctx context.Context, agentCtx *AgentContext) error {
		order = append(order, "terminal")
		return nil
	})

	if err != nil {
		t.Errorf("ChainAgentMiddleware() error = %v", err)
	}

	expected := []string{"mw1-before", "mw1-after"}
	if len(order) != len(expected) {
		t.Errorf("order = %v, want %v", order, expected)
	}
}

func TestChainFunctionMiddleware_Empty(t *testing.T) {
	chain := ChainFunctionMiddleware()
	called := false

	err := chain.Process(context.Background(), &FunctionContext{}, func(ctx context.Context, funcCtx *FunctionContext) error {
		called = true
		return nil
	})

	if err != nil {
		t.Errorf("ChainFunctionMiddleware() error = %v", err)
	}
	if !called {
		t.Error("ChainFunctionMiddleware() did not call next")
	}
}

func TestChainFunctionMiddleware_Multiple(t *testing.T) {
	order := []string{}

	mw1 := FunctionMiddlewareFunc(func(ctx context.Context, funcCtx *FunctionContext, next FunctionHandler) error {
		order = append(order, "mw1-before")
		err := next(ctx, funcCtx)
		order = append(order, "mw1-after")
		return err
	})

	mw2 := FunctionMiddlewareFunc(func(ctx context.Context, funcCtx *FunctionContext, next FunctionHandler) error {
		order = append(order, "mw2-before")
		err := next(ctx, funcCtx)
		order = append(order, "mw2-after")
		return err
	})

	chain := ChainFunctionMiddleware(mw1, mw2)

	err := chain.Process(context.Background(), &FunctionContext{}, func(ctx context.Context, funcCtx *FunctionContext) error {
		order = append(order, "terminal")
		return nil
	})

	if err != nil {
		t.Errorf("ChainFunctionMiddleware() error = %v", err)
	}

	expected := []string{"mw1-before", "mw2-before", "terminal", "mw2-after", "mw1-after"}
	if len(order) != len(expected) {
		t.Errorf("order = %v, want %v", order, expected)
	}
}
