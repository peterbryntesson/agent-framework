// Copyright (c) Microsoft. All rights reserved.

package chatagent

import (
	"testing"

	"github.com/microsoft/agent-framework-go/chat"
	"github.com/microsoft/agent-framework-go/tool"
)

func TestNewBuilder(t *testing.T) {
	client := newMockClient()
	builder := NewBuilder(client)

	if builder.client != client {
		t.Error("expected client to be set")
	}
	if len(builder.opts) != 0 {
		t.Errorf("expected empty opts, got %d", len(builder.opts))
	}
}

func TestBuilderID(t *testing.T) {
	client := newMockClient()
	agent, err := NewBuilder(client).
		ID("custom-id").
		Build()

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if agent.ID() != "custom-id" {
		t.Errorf("expected ID 'custom-id', got %q", agent.ID())
	}
}

func TestBuilderName(t *testing.T) {
	client := newMockClient()
	agent, err := NewBuilder(client).
		Name("MyAgent").
		Build()

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if agent.Name() != "MyAgent" {
		t.Errorf("expected name 'MyAgent', got %q", agent.Name())
	}
}

func TestBuilderDescription(t *testing.T) {
	client := newMockClient()
	agent, err := NewBuilder(client).
		Description("A helpful agent").
		Build()

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if agent.Description() != "A helpful agent" {
		t.Errorf("expected description, got %q", agent.Description())
	}
}

func TestBuilderInstructions(t *testing.T) {
	client := newMockClient()
	agent, err := NewBuilder(client).
		Instructions("Be helpful and concise").
		Build()

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if agent.Instructions() != "Be helpful and concise" {
		t.Errorf("expected instructions, got %q", agent.Instructions())
	}
}

func TestBuilderTools(t *testing.T) {
	client := newMockClient()
	mockTool := &mockFunctionTool{name: "test_tool"}

	agent, err := NewBuilder(client).
		Tools(mockTool).
		Build()

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(agent.Tools()) != 1 {
		t.Errorf("expected 1 tool, got %d", len(agent.Tools()))
	}
}

func TestBuilderMaxTurns(t *testing.T) {
	t.Run("valid max turns", func(t *testing.T) {
		client := newMockClient()
		agent, err := NewBuilder(client).
			MaxTurns(20).
			Build()

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if agent.maxTurns != 20 {
			t.Errorf("expected maxTurns 20, got %d", agent.maxTurns)
		}
	})

	t.Run("invalid max turns", func(t *testing.T) {
		client := newMockClient()
		_, err := NewBuilder(client).
			MaxTurns(0).
			Build()

		if err == nil {
			t.Fatal("expected error for invalid maxTurns")
		}
	})
}

func TestBuilderInvocationConfig(t *testing.T) {
	client := newMockClient()
	cfg := tool.InvocationConfig{
		Enabled:       false,
		MaxIterations: 50,
	}

	agent, err := NewBuilder(client).
		InvocationConfig(cfg).
		Build()

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if agent.invocationConfig.Enabled {
		t.Error("expected Enabled false")
	}
	if agent.invocationConfig.MaxIterations != 50 {
		t.Errorf("expected MaxIterations 50, got %d", agent.invocationConfig.MaxIterations)
	}
}

func TestBuilderInvocationEnabled(t *testing.T) {
	client := newMockClient()
	agent, err := NewBuilder(client).
		InvocationEnabled(false).
		Build()

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if agent.invocationConfig.Enabled {
		t.Error("expected Enabled false")
	}
}

func TestBuilderToolTimeout(t *testing.T) {
	client := newMockClient()
	agent, err := NewBuilder(client).
		ToolTimeout(30).
		Build()

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if agent.invocationConfig.TimeoutSeconds != 30 {
		t.Errorf("expected TimeoutSeconds 30, got %d", agent.invocationConfig.TimeoutSeconds)
	}
}

func TestBuilderParallelToolCalls(t *testing.T) {
	client := newMockClient()
	agent, err := NewBuilder(client).
		ParallelToolCalls(true).
		Build()

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !agent.invocationConfig.ParallelToolCalls {
		t.Error("expected ParallelToolCalls true")
	}
}

func TestBuilderIncludeDetailedErrors(t *testing.T) {
	client := newMockClient()
	agent, err := NewBuilder(client).
		IncludeDetailedErrors(true).
		Build()

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !agent.invocationConfig.IncludeDetailedErrors {
		t.Error("expected IncludeDetailedErrors true")
	}
}

func TestBuilderChaining(t *testing.T) {
	client := newMockClient()
	mockTool := &mockFunctionTool{name: "test_tool"}

	agent, err := NewBuilder(client).
		ID("my-agent").
		Name("MyAgent").
		Description("A helpful agent").
		Instructions("Be helpful").
		Tools(mockTool).
		MaxTurns(15).
		ToolTimeout(30).
		ParallelToolCalls(true).
		Build()

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if agent.ID() != "my-agent" {
		t.Errorf("expected ID 'my-agent', got %q", agent.ID())
	}
	if agent.Name() != "MyAgent" {
		t.Errorf("expected name 'MyAgent', got %q", agent.Name())
	}
	if agent.Description() != "A helpful agent" {
		t.Errorf("expected description, got %q", agent.Description())
	}
	if agent.Instructions() != "Be helpful" {
		t.Errorf("expected instructions, got %q", agent.Instructions())
	}
	if len(agent.Tools()) != 1 {
		t.Errorf("expected 1 tool, got %d", len(agent.Tools()))
	}
	if agent.maxTurns != 15 {
		t.Errorf("expected maxTurns 15, got %d", agent.maxTurns)
	}
}

func TestBuilderNilClient(t *testing.T) {
	_, err := NewBuilder(nil).Build()

	if err == nil {
		t.Fatal("expected error for nil client")
	}
	if err.Error() != "chat client is required" {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestBuilderMustBuild(t *testing.T) {
	t.Run("succeeds with valid config", func(t *testing.T) {
		client := newMockClient()
		agent := NewBuilder(client).MustBuild()

		if agent == nil {
			t.Fatal("expected agent")
		}
	})

	t.Run("panics with invalid config", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Fatal("expected panic")
			}
		}()

		NewBuilder(nil).MustBuild()
	})
}

func TestBuilderErrorPropagation(t *testing.T) {
	client := newMockClient()

	// Error at MaxTurns should propagate even after other calls
	_, err := NewBuilder(client).
		MaxTurns(-1).
		Name("Test").
		Build()

	if err == nil {
		t.Fatal("expected error to propagate")
	}
}

// mockClient implementation for builder tests
func init() {
	// Ensure mockClient implements chat.Client
	var _ chat.Client = (*mockClient)(nil)
}
