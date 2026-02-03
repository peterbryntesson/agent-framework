// Copyright (c) Microsoft. All rights reserved.

package openai

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/microsoft/agent-framework-go/tool"
	openailib "github.com/sashabaranov/go-openai"
)

// mockTool is a simple tool implementation for testing.
type mockTool struct {
	name        string
	description string
	parameters  json.RawMessage
}

func (t *mockTool) Name() string                { return t.name }
func (t *mockTool) Description() string         { return t.description }
func (t *mockTool) Parameters() json.RawMessage { return t.parameters }
func (t *mockTool) Invoke(ctx context.Context, args json.RawMessage) (tool.Result, error) {
	return tool.NewResult("result"), nil
}

// mockHostedTool is a hosted tool implementation for testing.
type mockHostedTool struct {
	name   string
	config map[string]interface{}
}

func (t *mockHostedTool) Name() string                { return t.name }
func (t *mockHostedTool) Description() string         { return "hosted tool" }
func (t *mockHostedTool) Parameters() json.RawMessage { return nil }
func (t *mockHostedTool) Invoke(ctx context.Context, args json.RawMessage) (tool.Result, error) {
	return tool.Result{}, nil
}
func (t *mockHostedTool) IsHosted() bool                       { return true }
func (t *mockHostedTool) ProviderConfig() map[string]interface{} { return t.config }

func TestConvertToolsToOpenAI(t *testing.T) {
	params := json.RawMessage(`{"type":"object","properties":{"x":{"type":"string"}}}`)

	tests := []struct {
		name     string
		tools    []tool.Tool
		expected int
	}{
		{
			name:     "empty slice",
			tools:    []tool.Tool{},
			expected: 0,
		},
		{
			name: "single function tool",
			tools: []tool.Tool{
				&mockTool{name: "test", description: "test desc", parameters: params},
			},
			expected: 1,
		},
		{
			name: "multiple function tools",
			tools: []tool.Tool{
				&mockTool{name: "tool1", description: "desc1", parameters: params},
				&mockTool{name: "tool2", description: "desc2", parameters: params},
			},
			expected: 2,
		},
		{
			name: "skips hosted tools",
			tools: []tool.Tool{
				&mockTool{name: "func_tool", description: "function", parameters: params},
				&mockHostedTool{name: "web_search", config: map[string]interface{}{"type": "web_search"}},
			},
			expected: 1,
		},
		{
			name: "only hosted tools returns empty",
			tools: []tool.Tool{
				&mockHostedTool{name: "web_search", config: map[string]interface{}{"type": "web_search"}},
				&mockHostedTool{name: "code_interpreter", config: map[string]interface{}{"type": "code_interpreter"}},
			},
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ConvertToolsToOpenAI(tt.tools)
			if len(result) != tt.expected {
				t.Errorf("expected %d tools, got %d", tt.expected, len(result))
			}

			// Verify tool properties for non-empty results
			for i, oaiTool := range result {
				if oaiTool.Type != openailib.ToolTypeFunction {
					t.Errorf("tool %d: expected type function, got %s", i, oaiTool.Type)
				}
				if oaiTool.Function == nil {
					t.Errorf("tool %d: function definition is nil", i)
				}
			}
		})
	}
}

func TestConvertToolsToOpenAIPreservesProperties(t *testing.T) {
	params := json.RawMessage(`{"type":"object","properties":{"location":{"type":"string"}}}`)
	mockT := &mockTool{
		name:        "get_weather",
		description: "Gets weather for a location",
		parameters:  params,
	}

	result := ConvertToolsToOpenAI([]tool.Tool{mockT})

	if len(result) != 1 {
		t.Fatalf("expected 1 tool, got %d", len(result))
	}

	oaiTool := result[0]
	if oaiTool.Function.Name != "get_weather" {
		t.Errorf("expected name 'get_weather', got '%s'", oaiTool.Function.Name)
	}
	if oaiTool.Function.Description != "Gets weather for a location" {
		t.Errorf("expected description mismatch")
	}
	// Compare parameters by re-marshaling to JSON
	paramBytes, ok := oaiTool.Function.Parameters.(json.RawMessage)
	if !ok {
		t.Errorf("expected Parameters to be json.RawMessage, got %T", oaiTool.Function.Parameters)
	} else if string(paramBytes) != string(params) {
		t.Errorf("expected parameters to match, got %s", string(paramBytes))
	}
}

func TestConvertHostedToolsToOpenAI(t *testing.T) {
	tests := []struct {
		name     string
		tools    []tool.Tool
		expected int
	}{
		{
			name:     "empty slice",
			tools:    []tool.Tool{},
			expected: 0,
		},
		{
			name: "no hosted tools",
			tools: []tool.Tool{
				&mockTool{name: "func_tool", description: "test"},
			},
			expected: 0,
		},
		{
			name: "single hosted tool",
			tools: []tool.Tool{
				&mockHostedTool{name: "web_search", config: map[string]interface{}{"type": "web_search"}},
			},
			expected: 1,
		},
		{
			name: "multiple hosted tools",
			tools: []tool.Tool{
				&mockHostedTool{name: "web_search", config: map[string]interface{}{"type": "web_search"}},
				&mockHostedTool{name: "code_interpreter", config: map[string]interface{}{"type": "code_interpreter"}},
			},
			expected: 2,
		},
		{
			name: "mixed tools returns only hosted",
			tools: []tool.Tool{
				&mockTool{name: "func_tool", description: "function"},
				&mockHostedTool{name: "web_search", config: map[string]interface{}{"type": "web_search"}},
				&mockTool{name: "func_tool2", description: "function2"},
			},
			expected: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ConvertHostedToolsToOpenAI(tt.tools)
			if len(result) != tt.expected {
				t.Errorf("expected %d hosted tools, got %d", tt.expected, len(result))
			}
		})
	}
}

func TestConvertHostedToolsPreservesConfig(t *testing.T) {
	config := map[string]interface{}{
		"type":                "web_search",
		"search_context_size": "high",
	}
	mockHT := &mockHostedTool{name: "web_search", config: config}

	result := ConvertHostedToolsToOpenAI([]tool.Tool{mockHT})

	if len(result) != 1 {
		t.Fatalf("expected 1 hosted tool config, got %d", len(result))
	}

	if result[0]["type"] != "web_search" {
		t.Errorf("expected type 'web_search', got '%v'", result[0]["type"])
	}
	if result[0]["search_context_size"] != "high" {
		t.Errorf("expected search_context_size 'high', got '%v'", result[0]["search_context_size"])
	}
}

func TestHasHostedTools(t *testing.T) {
	tests := []struct {
		name     string
		tools    []tool.Tool
		expected bool
	}{
		{
			name:     "empty slice",
			tools:    []tool.Tool{},
			expected: false,
		},
		{
			name: "only function tools",
			tools: []tool.Tool{
				&mockTool{name: "func1"},
				&mockTool{name: "func2"},
			},
			expected: false,
		},
		{
			name: "only hosted tools",
			tools: []tool.Tool{
				&mockHostedTool{name: "web_search", config: map[string]interface{}{}},
			},
			expected: true,
		},
		{
			name: "mixed tools",
			tools: []tool.Tool{
				&mockTool{name: "func1"},
				&mockHostedTool{name: "web_search", config: map[string]interface{}{}},
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := HasHostedTools(tt.tools)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestSeparateTools(t *testing.T) {
	funcTool1 := &mockTool{name: "func1"}
	funcTool2 := &mockTool{name: "func2"}
	hostedTool1 := &mockHostedTool{name: "web_search", config: map[string]interface{}{}}
	hostedTool2 := &mockHostedTool{name: "code_interpreter", config: map[string]interface{}{}}

	tests := []struct {
		name            string
		tools           []tool.Tool
		expectedFunc    int
		expectedHosted  int
	}{
		{
			name:           "empty slice",
			tools:          []tool.Tool{},
			expectedFunc:   0,
			expectedHosted: 0,
		},
		{
			name:           "only function tools",
			tools:          []tool.Tool{funcTool1, funcTool2},
			expectedFunc:   2,
			expectedHosted: 0,
		},
		{
			name:           "only hosted tools",
			tools:          []tool.Tool{hostedTool1, hostedTool2},
			expectedFunc:   0,
			expectedHosted: 2,
		},
		{
			name:           "mixed tools",
			tools:          []tool.Tool{funcTool1, hostedTool1, funcTool2, hostedTool2},
			expectedFunc:   2,
			expectedHosted: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			funcTools, hostedTools := SeparateTools(tt.tools)
			if len(funcTools) != tt.expectedFunc {
				t.Errorf("expected %d function tools, got %d", tt.expectedFunc, len(funcTools))
			}
			if len(hostedTools) != tt.expectedHosted {
				t.Errorf("expected %d hosted tools, got %d", tt.expectedHosted, len(hostedTools))
			}
		})
	}
}

func TestToolChoiceForOpenAI(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected interface{}
	}{
		{
			name:     "tool.ToolChoiceAuto",
			input:    tool.ToolChoiceAuto,
			expected: "auto",
		},
		{
			name:     "tool.ToolChoiceRequired",
			input:    tool.ToolChoiceRequired,
			expected: "required",
		},
		{
			name:     "tool.ToolChoiceNone",
			input:    tool.ToolChoiceNone,
			expected: "none",
		},
		{
			name:     "string auto",
			input:    "auto",
			expected: "auto",
		},
		{
			name:     "string required",
			input:    "required",
			expected: "required",
		},
		{
			name:     "string none",
			input:    "none",
			expected: "none",
		},
		{
			name:     "empty string",
			input:    "",
			expected: nil,
		},
		{
			name:  "specific tool name string",
			input: "get_weather",
			expected: openailib.ToolChoice{
				Type: openailib.ToolTypeFunction,
				Function: openailib.ToolFunction{
					Name: "get_weather",
				},
			},
		},
		{
			name:  "SpecificToolChoice struct",
			input: tool.SpecificToolChoice{Name: "my_function"},
			expected: openailib.ToolChoice{
				Type: openailib.ToolTypeFunction,
				Function: openailib.ToolFunction{
					Name: "my_function",
				},
			},
		},
		{
			name:     "unknown type returns nil",
			input:    123,
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ToolChoiceForOpenAI(tt.input)

			switch expected := tt.expected.(type) {
			case nil:
				if result != nil {
					t.Errorf("expected nil, got %v", result)
				}
			case string:
				if result != expected {
					t.Errorf("expected %q, got %v", expected, result)
				}
			case openailib.ToolChoice:
				resultChoice, ok := result.(openailib.ToolChoice)
				if !ok {
					t.Errorf("expected ToolChoice, got %T", result)
					return
				}
				if resultChoice.Type != expected.Type {
					t.Errorf("expected type %s, got %s", expected.Type, resultChoice.Type)
				}
				if resultChoice.Function.Name != expected.Function.Name {
					t.Errorf("expected function name %q, got %q", expected.Function.Name, resultChoice.Function.Name)
				}
			}
		})
	}
}

func TestParallelToolCallsOption(t *testing.T) {
	truePtr := ParallelToolCallsOption(true)
	if truePtr == nil || *truePtr != true {
		t.Error("expected pointer to true")
	}

	falsePtr := ParallelToolCallsOption(false)
	if falsePtr == nil || *falsePtr != false {
		t.Error("expected pointer to false")
	}
}
