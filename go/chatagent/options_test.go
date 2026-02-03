// Copyright (c) Microsoft. All rights reserved.

package chatagent

import (
	"testing"

	"github.com/microsoft/agent-framework-go/tool"
)

func TestDefaultConfig(t *testing.T) {
	cfg := defaultConfig()

	if cfg.maxTurns != 10 {
		t.Errorf("expected maxTurns 10, got %d", cfg.maxTurns)
	}
	if cfg.invocationConfig.MaxIterations != 40 {
		t.Errorf("expected MaxIterations 40, got %d", cfg.invocationConfig.MaxIterations)
	}
	if cfg.invocationConfig.MaxConsecutiveErrors != 3 {
		t.Errorf("expected MaxConsecutiveErrors 3, got %d", cfg.invocationConfig.MaxConsecutiveErrors)
	}
}

func TestWithID(t *testing.T) {
	cfg := defaultConfig()
	WithID("test-id")(cfg)

	if cfg.id != "test-id" {
		t.Errorf("expected id 'test-id', got %q", cfg.id)
	}
}

func TestWithName(t *testing.T) {
	cfg := defaultConfig()
	WithName("TestAgent")(cfg)

	if cfg.name != "TestAgent" {
		t.Errorf("expected name 'TestAgent', got %q", cfg.name)
	}
}

func TestWithDescription(t *testing.T) {
	cfg := defaultConfig()
	WithDescription("A test agent")(cfg)

	if cfg.description != "A test agent" {
		t.Errorf("expected description 'A test agent', got %q", cfg.description)
	}
}

func TestWithInstructions(t *testing.T) {
	cfg := defaultConfig()
	WithInstructions("Be helpful and concise")(cfg)

	if cfg.instructions != "Be helpful and concise" {
		t.Errorf("expected instructions 'Be helpful and concise', got %q", cfg.instructions)
	}
}

func TestWithTools(t *testing.T) {
	cfg := defaultConfig()
	mockTool := &mockFunctionTool{name: "test_tool"}
	WithTools(mockTool)(cfg)

	if len(cfg.tools) != 1 {
		t.Fatalf("expected 1 tool, got %d", len(cfg.tools))
	}
	if cfg.tools[0].Name() != "test_tool" {
		t.Errorf("expected tool name 'test_tool', got %q", cfg.tools[0].Name())
	}
}

func TestWithToolsMultiple(t *testing.T) {
	cfg := defaultConfig()
	tool1 := &mockFunctionTool{name: "tool1"}
	tool2 := &mockFunctionTool{name: "tool2"}
	WithTools(tool1, tool2)(cfg)

	if len(cfg.tools) != 2 {
		t.Fatalf("expected 2 tools, got %d", len(cfg.tools))
	}
}

func TestWithToolsChained(t *testing.T) {
	cfg := defaultConfig()
	tool1 := &mockFunctionTool{name: "tool1"}
	tool2 := &mockFunctionTool{name: "tool2"}
	WithTools(tool1)(cfg)
	WithTools(tool2)(cfg)

	if len(cfg.tools) != 2 {
		t.Fatalf("expected 2 tools after chaining, got %d", len(cfg.tools))
	}
}

func TestWithMaxTurns(t *testing.T) {
	cfg := defaultConfig()
	WithMaxTurns(20)(cfg)

	if cfg.maxTurns != 20 {
		t.Errorf("expected maxTurns 20, got %d", cfg.maxTurns)
	}
}

func TestWithInvocationConfig(t *testing.T) {
	cfg := defaultConfig()
	invCfg := tool.InvocationConfig{
		Enabled:       false,
		MaxIterations: 100,
	}
	WithInvocationConfig(invCfg)(cfg)

	if cfg.invocationConfig.Enabled {
		t.Error("expected Enabled false")
	}
	if cfg.invocationConfig.MaxIterations != 100 {
		t.Errorf("expected MaxIterations 100, got %d", cfg.invocationConfig.MaxIterations)
	}
}

func TestWithInvocationEnabled(t *testing.T) {
	cfg := defaultConfig()
	WithInvocationEnabled(false)(cfg)

	if cfg.invocationConfig.Enabled {
		t.Error("expected Enabled false")
	}
}

func TestWithMaxConsecutiveErrors(t *testing.T) {
	cfg := defaultConfig()
	WithMaxConsecutiveErrors(5)(cfg)

	if cfg.invocationConfig.MaxConsecutiveErrors != 5 {
		t.Errorf("expected MaxConsecutiveErrors 5, got %d", cfg.invocationConfig.MaxConsecutiveErrors)
	}
}

func TestWithTerminateOnUnknownCalls(t *testing.T) {
	cfg := defaultConfig()
	WithTerminateOnUnknownCalls(true)(cfg)

	if !cfg.invocationConfig.TerminateOnUnknownCalls {
		t.Error("expected TerminateOnUnknownCalls true")
	}
}

func TestWithIncludeDetailedErrors(t *testing.T) {
	cfg := defaultConfig()
	WithIncludeDetailedErrors(true)(cfg)

	if !cfg.invocationConfig.IncludeDetailedErrors {
		t.Error("expected IncludeDetailedErrors true")
	}
}

func TestWithToolTimeout(t *testing.T) {
	cfg := defaultConfig()
	WithToolTimeout(30)(cfg)

	if cfg.invocationConfig.TimeoutSeconds != 30 {
		t.Errorf("expected TimeoutSeconds 30, got %d", cfg.invocationConfig.TimeoutSeconds)
	}
}

func TestWithParallelToolCalls(t *testing.T) {
	cfg := defaultConfig()
	WithParallelToolCalls(true)(cfg)

	if !cfg.invocationConfig.ParallelToolCalls {
		t.Error("expected ParallelToolCalls true")
	}
}

func TestWithReturnIntermediateSteps(t *testing.T) {
	cfg := defaultConfig()
	WithReturnIntermediateSteps(true)(cfg)

	if !cfg.invocationConfig.ReturnIntermediateSteps {
		t.Error("expected ReturnIntermediateSteps true")
	}
}

func TestOptionChaining(t *testing.T) {
	cfg := defaultConfig()
	options := []Option{
		WithID("test-id"),
		WithName("TestAgent"),
		WithDescription("Test description"),
		WithInstructions("Be helpful"),
		WithMaxTurns(15),
		WithToolTimeout(60),
	}

	for _, opt := range options {
		opt(cfg)
	}

	if cfg.id != "test-id" {
		t.Errorf("expected id 'test-id', got %q", cfg.id)
	}
	if cfg.name != "TestAgent" {
		t.Errorf("expected name 'TestAgent', got %q", cfg.name)
	}
	if cfg.description != "Test description" {
		t.Errorf("expected description, got %q", cfg.description)
	}
	if cfg.instructions != "Be helpful" {
		t.Errorf("expected instructions, got %q", cfg.instructions)
	}
	if cfg.maxTurns != 15 {
		t.Errorf("expected maxTurns 15, got %d", cfg.maxTurns)
	}
	if cfg.invocationConfig.TimeoutSeconds != 60 {
		t.Errorf("expected TimeoutSeconds 60, got %d", cfg.invocationConfig.TimeoutSeconds)
	}
}
