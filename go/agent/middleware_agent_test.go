// Copyright (c) Microsoft. All rights reserved.

package agent

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"testing"
)

// testMiddlewareAgent is a minimal agent for testing middleware.
type testMiddlewareAgent struct {
	id          string
	name        string
	description string
	response    *Response
	runErr      error
}

func (a *testMiddlewareAgent) ID() string          { return a.id }
func (a *testMiddlewareAgent) Name() string        { return a.name }
func (a *testMiddlewareAgent) Description() string { return a.description }
func (a *testMiddlewareAgent) Metadata() AIAgentMetadata {
	return AIAgentMetadata{}
}

func (a *testMiddlewareAgent) Run(ctx context.Context, msgs []Message, opts ...RunOption) (*Response, error) {
	if a.runErr != nil {
		return nil, a.runErr
	}
	if a.response != nil {
		return a.response, nil
	}
	return &Response{}, nil
}

func (a *testMiddlewareAgent) RunStream(ctx context.Context, msgs []Message, opts ...RunOption) (<-chan ResponseUpdate, error) {
	ch := make(chan ResponseUpdate, 1)
	ch <- ResponseUpdate{Kind: UpdateKindDone}
	close(ch)
	return ch, nil
}

func (a *testMiddlewareAgent) NewSession(ctx context.Context) (Session, error) {
	return nil, nil
}

func (a *testMiddlewareAgent) RestoreSession(ctx context.Context, data json.RawMessage) (Session, error) {
	return nil, nil
}

func (a *testMiddlewareAgent) GetService(t reflect.Type) interface{} {
	return nil
}

func TestMiddlewareAgent_Run_NoMiddleware(t *testing.T) {
	inner := &testMiddlewareAgent{id: "test-agent"}
	agent := NewMiddlewareAgent(inner)

	resp, err := agent.Run(context.Background(), nil)
	if err != nil {
		t.Errorf("Run() error = %v", err)
	}
	if resp == nil {
		t.Error("Run() returned nil response")
	}
}

func TestMiddlewareAgent_Run_WithMiddleware(t *testing.T) {
	inner := &testMiddlewareAgent{id: "test-agent"}
	middlewareCalled := false

	mw := AgentMiddlewareFunc(func(ctx context.Context, agentCtx *AgentContext, next AgentHandler) error {
		middlewareCalled = true
		return next(ctx, agentCtx)
	})

	agent := NewMiddlewareAgent(inner, mw)

	_, err := agent.Run(context.Background(), nil)
	if err != nil {
		t.Errorf("Run() error = %v", err)
	}
	if !middlewareCalled {
		t.Error("middleware was not called")
	}
}

func TestMiddlewareAgent_Run_MiddlewareModifiesResponse(t *testing.T) {
	inner := &testMiddlewareAgent{
		id:       "test-agent",
		response: &Response{},
	}

	mw := AgentMiddlewareFunc(func(ctx context.Context, agentCtx *AgentContext, next AgentHandler) error {
		err := next(ctx, agentCtx)
		if err != nil {
			return err
		}
		// Modify the response after the inner agent runs
		agentCtx.Response = &Response{FinishReason: FinishReasonStop}
		return nil
	})

	agent := NewMiddlewareAgent(inner, mw)

	resp, err := agent.Run(context.Background(), nil)
	if err != nil {
		t.Errorf("Run() error = %v", err)
	}
	if resp.FinishReason != FinishReasonStop {
		t.Errorf("response was not modified by middleware")
	}
}

func TestMiddlewareAgent_Run_MiddlewareShortCircuits(t *testing.T) {
	innerCalled := false
	inner := &testMiddlewareAgent{id: "test-agent"}

	mw := AgentMiddlewareFunc(func(ctx context.Context, agentCtx *AgentContext, next AgentHandler) error {
		// Don't call next - set response directly
		agentCtx.Response = &Response{FinishReason: FinishReasonLength}
		return nil
	})

	// Wrap inner to detect if it's called
	wrappedInner := &callTracker{inner: inner, called: &innerCalled}
	agent := NewMiddlewareAgent(wrappedInner, mw)

	resp, err := agent.Run(context.Background(), nil)
	if err != nil {
		t.Errorf("Run() error = %v", err)
	}
	if innerCalled {
		t.Error("inner agent was called despite middleware short-circuit")
	}
	if resp.FinishReason != FinishReasonLength {
		t.Error("response was not set by middleware")
	}
}

func TestMiddlewareAgent_Run_PropagatesError(t *testing.T) {
	expectedErr := errors.New("middleware error")
	inner := &testMiddlewareAgent{id: "test-agent"}

	mw := AgentMiddlewareFunc(func(ctx context.Context, agentCtx *AgentContext, next AgentHandler) error {
		return expectedErr
	})

	agent := NewMiddlewareAgent(inner, mw)

	_, err := agent.Run(context.Background(), nil)
	if !errors.Is(err, expectedErr) {
		t.Errorf("Run() error = %v, want %v", err, expectedErr)
	}
}

func TestMiddlewareAgent_RunStream(t *testing.T) {
	inner := &testMiddlewareAgent{id: "test-agent"}
	middlewareCalled := false

	mw := AgentMiddlewareFunc(func(ctx context.Context, agentCtx *AgentContext, next AgentHandler) error {
		middlewareCalled = true
		if !agentCtx.IsStreaming {
			t.Error("IsStreaming should be true for RunStream")
		}
		return next(ctx, agentCtx)
	})

	agent := NewMiddlewareAgent(inner, mw)

	stream, err := agent.RunStream(context.Background(), nil)
	if err != nil {
		t.Errorf("RunStream() error = %v", err)
	}
	if !middlewareCalled {
		t.Error("middleware was not called")
	}

	// Drain the stream
	for range stream {
		// intentionally empty - drain streaming updates
	}
}

func TestMiddlewareAgent_Inner(t *testing.T) {
	inner := &testMiddlewareAgent{id: "test-agent"}
	agent := NewMiddlewareAgent(inner)

	if agent.Inner() != inner {
		t.Error("Inner() did not return the wrapped agent")
	}
}

func TestMiddlewareAgent_ForwardsMetadataMethods(t *testing.T) {
	inner := &testMiddlewareAgent{
		id:          "test-id",
		name:        "test-name",
		description: "test-desc",
	}
	agent := NewMiddlewareAgent(inner)

	if agent.ID() != "test-id" {
		t.Errorf("ID() = %q, want %q", agent.ID(), "test-id")
	}
	if agent.Name() != "test-name" {
		t.Errorf("Name() = %q, want %q", agent.Name(), "test-name")
	}
	if agent.Description() != "test-desc" {
		t.Errorf("Description() = %q, want %q", agent.Description(), "test-desc")
	}
}

// callTracker wraps an agent and tracks if it was called.
type callTracker struct {
	inner  Agent
	called *bool
}

func (c *callTracker) ID() string          { return c.inner.ID() }
func (c *callTracker) Name() string        { return c.inner.Name() }
func (c *callTracker) Description() string { return c.inner.Description() }
func (c *callTracker) Metadata() AIAgentMetadata {
	return c.inner.Metadata()
}

func (c *callTracker) Run(ctx context.Context, msgs []Message, opts ...RunOption) (*Response, error) {
	*c.called = true
	return c.inner.Run(ctx, msgs, opts...)
}

func (c *callTracker) RunStream(ctx context.Context, msgs []Message, opts ...RunOption) (<-chan ResponseUpdate, error) {
	*c.called = true
	return c.inner.RunStream(ctx, msgs, opts...)
}

func (c *callTracker) NewSession(ctx context.Context) (Session, error) {
	return c.inner.NewSession(ctx)
}

func (c *callTracker) RestoreSession(ctx context.Context, data json.RawMessage) (Session, error) {
	return c.inner.RestoreSession(ctx, data)
}

func (c *callTracker) GetService(t reflect.Type) interface{} {
	return c.inner.GetService(t)
}
