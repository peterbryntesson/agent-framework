// Copyright (c) Microsoft. All rights reserved.

package agent

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"
)

type mockAgent struct {
	id           string
	name         string
	description  string
	runCalled    bool
	streamCalled bool
}

func (m *mockAgent) ID() string          { return m.id }
func (m *mockAgent) Name() string        { return m.name }
func (m *mockAgent) Description() string { return m.description }
func (m *mockAgent) Metadata() AIAgentMetadata {
	return AIAgentMetadata{}
}

func (m *mockAgent) Run(ctx context.Context, msgs []Message, opts ...RunOption) (*Response, error) {
	m.runCalled = true
	return &Response{}, nil
}

func (m *mockAgent) RunStream(ctx context.Context, msgs []Message, opts ...RunOption) (<-chan ResponseUpdate, error) {
	m.streamCalled = true
	ch := make(chan ResponseUpdate)
	close(ch)
	return ch, nil
}

func (m *mockAgent) NewSession(ctx context.Context) (Session, error) {
	return nil, nil
}

func (m *mockAgent) RestoreSession(ctx context.Context, data json.RawMessage) (Session, error) {
	return nil, nil
}

func (m *mockAgent) GetService(t reflect.Type) interface{} {
	return nil
}

func TestDelegatingAgent_ForwardsAllMethods(t *testing.T) {
	inner := &mockAgent{id: "test-id", name: "test-name", description: "test-desc"}
	delegating := NewDelegatingAgent(inner)

	if delegating.ID() != "test-id" {
		t.Errorf("ID() = %q, want %q", delegating.ID(), "test-id")
	}
	if delegating.Name() != "test-name" {
		t.Errorf("Name() = %q, want %q", delegating.Name(), "test-name")
	}
	if delegating.Description() != "test-desc" {
		t.Errorf("Description() = %q, want %q", delegating.Description(), "test-desc")
	}
}

func TestDelegatingAgent_Inner(t *testing.T) {
	inner := &mockAgent{id: "inner-id"}
	delegating := NewDelegatingAgent(inner)

	if delegating.Inner() != inner {
		t.Error("Inner() did not return the wrapped agent")
	}
}

func TestDelegatingAgent_Run(t *testing.T) {
	inner := &mockAgent{}
	delegating := NewDelegatingAgent(inner)

	_, err := delegating.Run(context.Background(), nil)
	if err != nil {
		t.Errorf("Run() error = %v", err)
	}
	if !inner.runCalled {
		t.Error("Run() did not forward to inner agent")
	}
}

func TestDelegatingAgent_RunStream(t *testing.T) {
	inner := &mockAgent{}
	delegating := NewDelegatingAgent(inner)

	_, err := delegating.RunStream(context.Background(), nil)
	if err != nil {
		t.Errorf("RunStream() error = %v", err)
	}
	if !inner.streamCalled {
		t.Error("RunStream() did not forward to inner agent")
	}
}

func TestDelegatingAgent_NewSession(t *testing.T) {
	inner := &mockAgent{}
	delegating := NewDelegatingAgent(inner)

	_, err := delegating.NewSession(context.Background())
	if err != nil {
		t.Errorf("NewSession() error = %v", err)
	}
}

func TestDelegatingAgent_RestoreSession(t *testing.T) {
	inner := &mockAgent{}
	delegating := NewDelegatingAgent(inner)

	_, err := delegating.RestoreSession(context.Background(), nil)
	if err != nil {
		t.Errorf("RestoreSession() error = %v", err)
	}
}

func TestDelegatingAgent_GetService(t *testing.T) {
	inner := &mockAgent{}
	delegating := NewDelegatingAgent(inner)

	result := delegating.GetService(reflect.TypeOf(""))
	if result != nil {
		t.Errorf("GetService() = %v, want nil", result)
	}
}

func TestDelegatingAgent_Metadata(t *testing.T) {
	inner := &mockAgent{}
	delegating := NewDelegatingAgent(inner)

	metadata := delegating.Metadata()
	if metadata.ProviderName != "" {
		t.Errorf("Metadata().ProviderName = %q, want empty", metadata.ProviderName)
	}
}
