// Copyright (c) Microsoft. All rights reserved.

package devui

import (
	"context"
	"encoding/json"
	"reflect"

	"github.com/microsoft/agent-framework-go/agent"
	"github.com/microsoft/agent-framework-go/chat"
)

// mockAgentFull implements agent.Agent for testing.
type mockAgentFull struct {
	id           string
	name         string
	description  string
	providerName string

	runFunc       func(ctx context.Context, messages []agent.Message, opts ...agent.RunOption) (*agent.Response, error)
	runStreamFunc func(ctx context.Context, messages []agent.Message, opts ...agent.RunOption) (<-chan agent.ResponseUpdate, error)
}

func (m *mockAgentFull) ID() string {
	return m.id
}

func (m *mockAgentFull) Name() string {
	return m.name
}

func (m *mockAgentFull) Description() string {
	return m.description
}

func (m *mockAgentFull) Metadata() agent.AIAgentMetadata {
	return agent.AIAgentMetadata{
		ProviderName: m.providerName,
	}
}

func (m *mockAgentFull) Run(ctx context.Context, messages []agent.Message, opts ...agent.RunOption) (*agent.Response, error) {
	if m.runFunc != nil {
		return m.runFunc(ctx, messages, opts...)
	}
	return &agent.Response{
		ResponseID: "test-response-id",
		Messages: []agent.Message{
			chat.NewAssistantMessage("Hello, World!"),
		},
		FinishReason: agent.FinishReasonStop,
	}, nil
}

func (m *mockAgentFull) RunStream(ctx context.Context, messages []agent.Message, opts ...agent.RunOption) (<-chan agent.ResponseUpdate, error) {
	if m.runStreamFunc != nil {
		return m.runStreamFunc(ctx, messages, opts...)
	}

	ch := make(chan agent.ResponseUpdate, 3)
	go func() {
		defer close(ch)
		ch <- agent.ResponseUpdate{
			Kind:  agent.UpdateKindContentDelta,
			Delta: &agent.ContentDelta{TextDelta: "Hello"},
		}
		ch <- agent.ResponseUpdate{
			Kind:  agent.UpdateKindContentDelta,
			Delta: &agent.ContentDelta{TextDelta: ", World!"},
		}
		ch <- agent.ResponseUpdate{
			Kind:         agent.UpdateKindDone,
			FinishReason: agent.FinishReasonStop,
		}
	}()
	return ch, nil
}

func (m *mockAgentFull) NewSession(ctx context.Context) (agent.Session, error) {
	return agent.NewInMemorySession(), nil
}

func (m *mockAgentFull) RestoreSession(ctx context.Context, data json.RawMessage) (agent.Session, error) {
	return agent.RestoreInMemorySession(data)
}

func (m *mockAgentFull) GetService(serviceType reflect.Type) interface{} {
	return nil
}
