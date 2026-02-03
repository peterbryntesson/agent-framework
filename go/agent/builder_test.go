// Copyright (c) Microsoft. All rights reserved.

package agent

import (
	"context"
	"testing"
)

func TestAgentBuilder_Build_NoFactories(t *testing.T) {
	inner := &mockAgent{id: "test-id"}
	builder := NewAgentBuilder(func() Agent {
		return inner
	})

	result := builder.Build()
	if result.ID() != "test-id" {
		t.Errorf("Build() ID = %q, want %q", result.ID(), "test-id")
	}
}

func TestAgentBuilder_Use_SingleFactory(t *testing.T) {
	inner := &mockAgent{id: "inner-id", name: "inner-name"}
	wrapperCalled := false

	builder := NewAgentBuilder(func() Agent {
		return inner
	}).Use(func(a Agent) Agent {
		wrapperCalled = true
		return NewDelegatingAgent(a)
	})

	result := builder.Build()

	if !wrapperCalled {
		t.Error("factory was not called")
	}
	if result.ID() != "inner-id" {
		t.Errorf("Build() ID = %q, want %q", result.ID(), "inner-id")
	}
}

func TestAgentBuilder_Use_MultipleFactories_Order(t *testing.T) {
	order := []string{}
	inner := &mockAgent{id: "inner"}

	builder := NewAgentBuilder(func() Agent {
		order = append(order, "create-inner")
		return inner
	}).Use(func(a Agent) Agent {
		order = append(order, "wrap-1")
		return NewDelegatingAgent(a)
	}).Use(func(a Agent) Agent {
		order = append(order, "wrap-2")
		return NewDelegatingAgent(a)
	}).Use(func(a Agent) Agent {
		order = append(order, "wrap-3")
		return NewDelegatingAgent(a)
	})

	_ = builder.Build()

	// First Use() should be outermost (applied last)
	expected := []string{"create-inner", "wrap-3", "wrap-2", "wrap-1"}
	if len(order) != len(expected) {
		t.Errorf("order = %v, want %v", order, expected)
		return
	}
	for i, v := range expected {
		if order[i] != v {
			t.Errorf("order[%d] = %q, want %q", i, order[i], v)
		}
	}
}

func TestAgentBuilder_UseMiddleware(t *testing.T) {
	inner := &mockAgent{id: "test"}
	middlewareCalled := false

	mw := AgentMiddlewareFunc(func(ctx context.Context, agentCtx *AgentContext, next AgentHandler) error {
		middlewareCalled = true
		return next(ctx, agentCtx)
	})

	result := NewAgentBuilder(func() Agent {
		return inner
	}).UseMiddleware(mw).Build()

	_, err := result.Run(context.Background(), nil)
	if err != nil {
		t.Errorf("Run() error = %v", err)
	}
	if !middlewareCalled {
		t.Error("middleware was not called")
	}
}

func TestAgentBuilder_ChainedUseMiddleware(t *testing.T) {
	inner := &mockAgent{id: "test"}
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

	result := NewAgentBuilder(func() Agent {
		return inner
	}).UseMiddleware(mw1).UseMiddleware(mw2).Build()

	_, err := result.Run(context.Background(), nil)
	if err != nil {
		t.Errorf("Run() error = %v", err)
	}

	// First UseMiddleware should be outermost
	expected := []string{"mw1-before", "mw2-before", "mw2-after", "mw1-after"}
	if len(order) != len(expected) {
		t.Errorf("order = %v, want %v", order, expected)
		return
	}
	for i, v := range expected {
		if order[i] != v {
			t.Errorf("order[%d] = %q, want %q", i, order[i], v)
		}
	}
}

func TestAgentBuilder_UseAndUseMiddleware_Combined(t *testing.T) {
	inner := &mockAgent{id: "inner-id"}
	decoratorApplied := false
	middlewareCalled := false

	mw := AgentMiddlewareFunc(func(ctx context.Context, agentCtx *AgentContext, next AgentHandler) error {
		middlewareCalled = true
		return next(ctx, agentCtx)
	})

	result := NewAgentBuilder(func() Agent {
		return inner
	}).Use(func(a Agent) Agent {
		decoratorApplied = true
		return NewDelegatingAgent(a)
	}).UseMiddleware(mw).Build()

	_, err := result.Run(context.Background(), nil)
	if err != nil {
		t.Errorf("Run() error = %v", err)
	}
	if !decoratorApplied {
		t.Error("decorator was not applied")
	}
	if !middlewareCalled {
		t.Error("middleware was not called")
	}
}
