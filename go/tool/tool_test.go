// Copyright (c) Microsoft. All rights reserved.

package tool

import (
	"context"
	"encoding/json"
	"testing"
)

func TestToolChoice_Constants(t *testing.T) {
	t.Run("ToolChoiceAuto has correct value", func(t *testing.T) {
		if ToolChoiceAuto != "auto" {
			t.Errorf("expected 'auto', got %q", ToolChoiceAuto)
		}
	})

	t.Run("ToolChoiceRequired has correct value", func(t *testing.T) {
		if ToolChoiceRequired != "required" {
			t.Errorf("expected 'required', got %q", ToolChoiceRequired)
		}
	})

	t.Run("ToolChoiceNone has correct value", func(t *testing.T) {
		if ToolChoiceNone != "none" {
			t.Errorf("expected 'none', got %q", ToolChoiceNone)
		}
	})
}

func TestApprovalMode_Constants(t *testing.T) {
	t.Run("ApprovalNever has correct value", func(t *testing.T) {
		if ApprovalNever != "never" {
			t.Errorf("expected 'never', got %q", ApprovalNever)
		}
	})

	t.Run("ApprovalAlways has correct value", func(t *testing.T) {
		if ApprovalAlways != "always" {
			t.Errorf("expected 'always', got %q", ApprovalAlways)
		}
	})

	t.Run("ApprovalOnce has correct value", func(t *testing.T) {
		if ApprovalOnce != "once" {
			t.Errorf("expected 'once', got %q", ApprovalOnce)
		}
	})
}

func TestToolType_Constants(t *testing.T) {
	tests := []struct {
		name     string
		toolType ToolType
		expected string
	}{
		{"ToolTypeFunction", ToolTypeFunction, "function"},
		{"ToolTypeHostedWebSearch", ToolTypeHostedWebSearch, "web_search"},
		{"ToolTypeHostedCodeInterpreter", ToolTypeHostedCodeInterpreter, "code_interpreter"},
		{"ToolTypeHostedFileSearch", ToolTypeHostedFileSearch, "file_search"},
		{"ToolTypeHostedMCP", ToolTypeHostedMCP, "mcp"},
		{"ToolTypeHostedImageGeneration", ToolTypeHostedImageGeneration, "image_generation"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.toolType != ToolType(tc.expected) {
				t.Errorf("expected %q, got %q", tc.expected, tc.toolType)
			}
		})
	}
}

func TestSpecificToolChoice(t *testing.T) {
	t.Run("stores tool name", func(t *testing.T) {
		choice := SpecificToolChoice{Name: "my_tool"}

		if choice.Name != "my_tool" {
			t.Errorf("expected 'my_tool', got %q", choice.Name)
		}
	})
}

func TestToolCall(t *testing.T) {
	t.Run("serializes to JSON", func(t *testing.T) {
		// Arrange
		call := ToolCall{
			ID:        "call_123",
			Name:      "get_weather",
			Arguments: json.RawMessage(`{"location":"Seattle"}`),
		}

		// Act
		data, err := json.Marshal(call)

		// Assert
		if err != nil {
			t.Fatalf("unexpected marshal error: %v", err)
		}

		var parsed map[string]interface{}
		if err := json.Unmarshal(data, &parsed); err != nil {
			t.Fatalf("unexpected unmarshal error: %v", err)
		}

		if parsed["id"] != "call_123" {
			t.Errorf("expected id 'call_123', got %v", parsed["id"])
		}
		if parsed["name"] != "get_weather" {
			t.Errorf("expected name 'get_weather', got %v", parsed["name"])
		}
	})

	t.Run("deserializes from JSON", func(t *testing.T) {
		// Arrange
		data := []byte(`{"id":"call_456","name":"search","arguments":{"query":"test"}}`)

		// Act
		var call ToolCall
		err := json.Unmarshal(data, &call)

		// Assert
		if err != nil {
			t.Fatalf("unexpected unmarshal error: %v", err)
		}
		if call.ID != "call_456" {
			t.Errorf("expected id 'call_456', got %q", call.ID)
		}
		if call.Name != "search" {
			t.Errorf("expected name 'search', got %q", call.Name)
		}
		if call.Arguments == nil {
			t.Error("expected non-nil arguments")
		}
	})
}

func TestToolResult(t *testing.T) {
	t.Run("serializes to JSON with all fields", func(t *testing.T) {
		// Arrange
		result := ToolResult{
			CallID: "call_123",
			Result: Result{
				Content:  "The weather is sunny",
				IsError:  false,
				Metadata: map[string]any{"source": "api"},
			},
		}

		// Act
		data, err := json.Marshal(result)

		// Assert
		if err != nil {
			t.Fatalf("unexpected marshal error: %v", err)
		}

		var parsed map[string]interface{}
		if err := json.Unmarshal(data, &parsed); err != nil {
			t.Fatalf("unexpected unmarshal error: %v", err)
		}

		if parsed["call_id"] != "call_123" {
			t.Errorf("expected call_id 'call_123', got %v", parsed["call_id"])
		}

		resultMap, ok := parsed["result"].(map[string]interface{})
		if !ok {
			t.Fatal("expected result to be a map")
		}
		if resultMap["content"] != "The weather is sunny" {
			t.Errorf("expected content 'The weather is sunny', got %v", resultMap["content"])
		}
	})

	t.Run("deserializes from JSON", func(t *testing.T) {
		// Arrange
		data := []byte(`{"call_id":"call_789","result":{"content":"error message","is_error":true}}`)

		// Act
		var result ToolResult
		err := json.Unmarshal(data, &result)

		// Assert
		if err != nil {
			t.Fatalf("unexpected unmarshal error: %v", err)
		}
		if result.CallID != "call_789" {
			t.Errorf("expected call_id 'call_789', got %q", result.CallID)
		}
		if !result.Result.IsError {
			t.Error("expected IsError to be true")
		}
		if result.Result.Content != "error message" {
			t.Errorf("expected content 'error message', got %q", result.Result.Content)
		}
	})
}

func TestAdditionalProperties(t *testing.T) {
	t.Run("stores and retrieves values", func(t *testing.T) {
		props := AdditionalProperties{
			"key1": "value1",
			"key2": 42,
			"key3": true,
		}

		if props["key1"] != "value1" {
			t.Errorf("expected 'value1', got %v", props["key1"])
		}
		if props["key2"] != 42 {
			t.Errorf("expected 42, got %v", props["key2"])
		}
		if props["key3"] != true {
			t.Errorf("expected true, got %v", props["key3"])
		}
	})

	t.Run("returns nil for missing keys", func(t *testing.T) {
		props := AdditionalProperties{"key1": "value1"}

		if props["missing"] != nil {
			t.Errorf("expected nil for missing key, got %v", props["missing"])
		}
	})
}

// testTool implements the Tool interface for testing.
type testTool struct {
	name        string
	description string
	parameters  json.RawMessage
	invokeFn    func(ctx context.Context, args json.RawMessage) (Result, error)
}

func (t *testTool) Name() string                { return t.name }
func (t *testTool) Description() string         { return t.description }
func (t *testTool) Parameters() json.RawMessage { return t.parameters }
func (t *testTool) Invoke(ctx context.Context, args json.RawMessage) (Result, error) {
	if t.invokeFn != nil {
		return t.invokeFn(ctx, args)
	}
	return NewResult("default"), nil
}

func TestToolInterface(t *testing.T) {
	t.Run("can be implemented", func(t *testing.T) {
		// Arrange
		tool := &testTool{
			name:        "test_tool",
			description: "A test tool",
			parameters:  json.RawMessage(`{"type":"object","properties":{}}`),
			invokeFn: func(_ context.Context, _ json.RawMessage) (Result, error) {
				return NewResult("test result"), nil
			},
		}

		// Act & Assert
		var iface Tool = tool
		if iface.Name() != "test_tool" {
			t.Errorf("expected name 'test_tool', got %q", iface.Name())
		}
		if iface.Description() != "A test tool" {
			t.Errorf("expected description 'A test tool', got %q", iface.Description())
		}
		if iface.Parameters() == nil {
			t.Error("expected non-nil parameters")
		}

		result, err := iface.Invoke(context.Background(), nil)
		if err != nil {
			t.Fatalf("unexpected invoke error: %v", err)
		}
		if result.Content != "test result" {
			t.Errorf("expected 'test result', got %q", result.Content)
		}
	})
}

// testHostedTool implements the HostedTool interface for testing.
type testHostedTool struct {
	testTool
	providerConfig map[string]interface{}
}

func (t *testHostedTool) IsHosted() bool                         { return true }
func (t *testHostedTool) ProviderConfig() map[string]interface{} { return t.providerConfig }

func TestHostedToolInterface(t *testing.T) {
	t.Run("extends Tool interface", func(t *testing.T) {
		// Arrange
		tool := &testHostedTool{
			testTool: testTool{
				name:        "hosted_test",
				description: "A hosted test tool",
			},
			providerConfig: map[string]interface{}{
				"type":   "test",
				"custom": "value",
			},
		}

		// Act & Assert
		var iface HostedTool = tool
		if !iface.IsHosted() {
			t.Error("expected IsHosted() to return true")
		}

		config := iface.ProviderConfig()
		if config["type"] != "test" {
			t.Errorf("expected type 'test', got %v", config["type"])
		}
		if config["custom"] != "value" {
			t.Errorf("expected custom 'value', got %v", config["custom"])
		}

		// Verify it also satisfies Tool interface
		var toolIface Tool = iface
		if toolIface.Name() != "hosted_test" {
			t.Errorf("expected name 'hosted_test', got %q", toolIface.Name())
		}
	})
}
