// Copyright (c) Microsoft. All rights reserved.

package tool

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
)

// Test argument types for FunctionTool tests.
type weatherArgs struct {
	Location string `json:"location" description:"The city name" required:"true"`
	Unit     string `json:"unit" description:"Temperature unit" enum:"celsius,fahrenheit"`
}

type calculatorArgs struct {
	A  float64 `json:"a" description:"First operand" required:"true"`
	B  float64 `json:"b" description:"Second operand" required:"true"`
	Op string  `json:"op" description:"Operation" enum:"add,subtract,multiply,divide"`
}

type emptyArgs struct{}

// Test functions with various signatures.
func getWeatherFunc(ctx context.Context, args weatherArgs) (string, error) {
	return "Weather in " + args.Location + ": 22°" + args.Unit, nil
}

func calculateFunc(args calculatorArgs) (float64, error) {
	switch args.Op {
	case "add":
		return args.A + args.B, nil
	case "subtract":
		return args.A - args.B, nil
	case "multiply":
		return args.A * args.B, nil
	case "divide":
		if args.B == 0 {
			return 0, errors.New("division by zero")
		}
		return args.A / args.B, nil
	default:
		return 0, errors.New("unknown operation")
	}
}

func noArgsFunc(ctx context.Context) (string, error) {
	return "no args result", nil
}

func pointerArgsFunc(_ context.Context, args *weatherArgs) (string, error) {
	return "Pointer args: " + args.Location, nil
}

func TestNewFunctionTool(t *testing.T) {
	t.Run("creates tool with valid function", func(t *testing.T) {
		// Act
		tool, err := NewFunctionTool("get_weather", "Get weather for a location", getWeatherFunc)

		// Assert
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if tool == nil {
			t.Fatal("expected non-nil tool")
		}
		if tool.Name() != "get_weather" {
			t.Errorf("expected name 'get_weather', got %q", tool.Name())
		}
		if tool.Description() != "Get weather for a location" {
			t.Errorf("expected description 'Get weather for a location', got %q", tool.Description())
		}
	})

	t.Run("generates parameters schema", func(t *testing.T) {
		// Act
		tool, err := NewFunctionTool("get_weather", "Get weather", getWeatherFunc)

		// Assert
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		params := tool.Parameters()
		if params == nil {
			t.Fatal("expected non-nil parameters")
		}

		var schema map[string]interface{}
		if err := json.Unmarshal(params, &schema); err != nil {
			t.Fatalf("failed to parse schema: %v", err)
		}

		if schema["type"] != "object" {
			t.Errorf("expected type 'object', got %v", schema["type"])
		}

		props, ok := schema["properties"].(map[string]interface{})
		if !ok {
			t.Fatal("expected properties to be a map")
		}
		if _, ok := props["location"]; !ok {
			t.Error("expected 'location' property")
		}
		if _, ok := props["unit"]; !ok {
			t.Error("expected 'unit' property")
		}
	})

	t.Run("returns error for nil function", func(t *testing.T) {
		// Act
		_, err := NewFunctionTool("test", "test", nil)

		// Assert
		if err == nil {
			t.Fatal("expected error for nil function")
		}
		if !errors.Is(err, ErrNilFunction) {
			t.Errorf("expected ErrNilFunction, got %v", err)
		}
	})

	t.Run("returns error for non-function", func(t *testing.T) {
		// Act
		_, err := NewFunctionTool("test", "test", "not a function")

		// Assert
		if err == nil {
			t.Fatal("expected error for non-function")
		}
		if !errors.Is(err, ErrInvalidSignature) {
			t.Errorf("expected ErrInvalidSignature, got %v", err)
		}
	})

	t.Run("returns error for invalid args type", func(t *testing.T) {
		// A function with non-struct args
		badFunc := func(ctx context.Context, args string) (string, error) {
			return args, nil
		}

		// Act
		_, err := NewFunctionTool("test", "test", badFunc)

		// Assert
		if err == nil {
			t.Fatal("expected error for invalid args type")
		}
		if !errors.Is(err, ErrInvalidSignature) {
			t.Errorf("expected ErrInvalidSignature, got %v", err)
		}
	})

	t.Run("returns error for too many return values", func(t *testing.T) {
		badFunc := func(args emptyArgs) (string, int, error) {
			return "", 0, nil
		}

		// Act
		_, err := NewFunctionTool("test", "test", badFunc)

		// Assert
		if err == nil {
			t.Fatal("expected error for too many return values")
		}
		if !errors.Is(err, ErrInvalidSignature) {
			t.Errorf("expected ErrInvalidSignature, got %v", err)
		}
	})

	t.Run("supports function without context", func(t *testing.T) {
		// Act
		tool, err := NewFunctionTool("calculate", "Calculate result", calculateFunc)

		// Assert
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if tool == nil {
			t.Fatal("expected non-nil tool")
		}
	})

	t.Run("supports function with pointer args", func(t *testing.T) {
		// Act
		tool, err := NewFunctionTool("pointer_test", "Test pointer args", pointerArgsFunc)

		// Assert
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if tool == nil {
			t.Fatal("expected non-nil tool")
		}
	})

	t.Run("supports no-arg function", func(t *testing.T) {
		// Act
		tool, err := NewFunctionTool("no_args", "No args function", noArgsFunc)

		// Assert
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if tool == nil {
			t.Fatal("expected non-nil tool")
		}

		// Verify empty schema
		params := tool.Parameters()
		if params == nil {
			t.Fatal("expected non-nil parameters")
		}

		var schema map[string]interface{}
		if err := json.Unmarshal(params, &schema); err != nil {
			t.Fatalf("failed to parse schema: %v", err)
		}
		if schema["type"] != "object" {
			t.Errorf("expected type 'object', got %v", schema["type"])
		}
	})
}

func TestFunctionTool_Invoke(t *testing.T) {
	t.Run("invokes function with arguments", func(t *testing.T) {
		// Arrange
		tool, _ := NewFunctionTool("get_weather", "Get weather", getWeatherFunc)
		args := json.RawMessage(`{"location":"Seattle","unit":"celsius"}`)

		// Act
		result, err := tool.Invoke(context.Background(), args)

		// Assert
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.IsError {
			t.Errorf("unexpected error result: %s", result.Content)
		}
		expected := "Weather in Seattle: 22°celsius"
		if result.Content != expected {
			t.Errorf("expected %q, got %q", expected, result.Content)
		}
	})

	t.Run("invokes function with nil arguments", func(t *testing.T) {
		// Arrange
		tool, _ := NewFunctionTool("get_weather", "Get weather", getWeatherFunc)

		// Act
		result, err := tool.Invoke(context.Background(), nil)

		// Assert
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// Function should execute with zero values
		if result.IsError {
			t.Errorf("unexpected error result: %s", result.Content)
		}
	})

	t.Run("invokes function with empty arguments", func(t *testing.T) {
		// Arrange
		tool, _ := NewFunctionTool("get_weather", "Get weather", getWeatherFunc)
		args := json.RawMessage(`{}`)

		// Act
		result, err := tool.Invoke(context.Background(), args)

		// Assert
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.IsError {
			t.Errorf("unexpected error result: %s", result.Content)
		}
	})

	t.Run("handles function returning error", func(t *testing.T) {
		// Arrange
		tool, _ := NewFunctionTool("calculate", "Calculate", calculateFunc)
		args := json.RawMessage(`{"a":10,"b":0,"op":"divide"}`)

		// Act
		result, err := tool.Invoke(context.Background(), args)

		// Assert
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !result.IsError {
			t.Error("expected error result for division by zero")
		}
		if result.Content != "division by zero" {
			t.Errorf("expected 'division by zero', got %q", result.Content)
		}
	})

	t.Run("handles numeric return value", func(t *testing.T) {
		// Arrange
		tool, _ := NewFunctionTool("calculate", "Calculate", calculateFunc)
		args := json.RawMessage(`{"a":10,"b":5,"op":"add"}`)

		// Act
		result, err := tool.Invoke(context.Background(), args)

		// Assert
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.IsError {
			t.Errorf("unexpected error result: %s", result.Content)
		}
		if result.Content != "15" {
			t.Errorf("expected '15', got %q", result.Content)
		}
	})

	t.Run("handles invalid JSON arguments", func(t *testing.T) {
		// Arrange
		tool, _ := NewFunctionTool("get_weather", "Get weather", getWeatherFunc)
		args := json.RawMessage(`{invalid json}`)

		// Act
		result, err := tool.Invoke(context.Background(), args)

		// Assert
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !result.IsError {
			t.Error("expected error result for invalid JSON")
		}
	})

	t.Run("invokes no-arg function", func(t *testing.T) {
		// Arrange
		tool, _ := NewFunctionTool("no_args", "No args", noArgsFunc)

		// Act
		result, err := tool.Invoke(context.Background(), nil)

		// Assert
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.IsError {
			t.Errorf("unexpected error result: %s", result.Content)
		}
		if result.Content != "no args result" {
			t.Errorf("expected 'no args result', got %q", result.Content)
		}
	})

	t.Run("invokes function with pointer args", func(t *testing.T) {
		// Arrange
		tool, _ := NewFunctionTool("pointer_test", "Test", pointerArgsFunc)
		args := json.RawMessage(`{"location":"Portland"}`)

		// Act
		result, err := tool.Invoke(context.Background(), args)

		// Assert
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.IsError {
			t.Errorf("unexpected error result: %s", result.Content)
		}
		expected := "Pointer args: Portland"
		if result.Content != expected {
			t.Errorf("expected %q, got %q", expected, result.Content)
		}
	})
}

func TestFunctionTool_MaxInvocations(t *testing.T) {
	t.Run("enforces max invocations limit", func(t *testing.T) {
		// Arrange
		tool, _ := NewFunctionTool("get_weather", "Get weather", getWeatherFunc)
		tool.MaxInvocations = 2
		args := json.RawMessage(`{"location":"Seattle"}`)

		// Act - First two invocations should succeed
		_, err1 := tool.Invoke(context.Background(), args)
		_, err2 := tool.Invoke(context.Background(), args)
		_, err3 := tool.Invoke(context.Background(), args)

		// Assert
		if err1 != nil {
			t.Fatalf("first invocation failed: %v", err1)
		}
		if err2 != nil {
			t.Fatalf("second invocation failed: %v", err2)
		}
		if err3 == nil {
			t.Fatal("expected error on third invocation")
		}
		if !errors.Is(err3, ErrMaxInvocationsReached) {
			t.Errorf("expected ErrMaxInvocationsReached, got %v", err3)
		}
	})

	t.Run("tracks invocation count", func(t *testing.T) {
		// Arrange
		tool, _ := NewFunctionTool("get_weather", "Get weather", getWeatherFunc)

		// Act
		if tool.InvocationCount() != 0 {
			t.Errorf("expected initial count 0, got %d", tool.InvocationCount())
		}

		tool.Invoke(context.Background(), nil)
		if tool.InvocationCount() != 1 {
			t.Errorf("expected count 1, got %d", tool.InvocationCount())
		}

		tool.Invoke(context.Background(), nil)
		if tool.InvocationCount() != 2 {
			t.Errorf("expected count 2, got %d", tool.InvocationCount())
		}
	})

	t.Run("resets invocation count", func(t *testing.T) {
		// Arrange
		tool, _ := NewFunctionTool("get_weather", "Get weather", getWeatherFunc)
		tool.Invoke(context.Background(), nil)
		tool.Invoke(context.Background(), nil)

		// Act
		tool.ResetInvocationCount()

		// Assert
		if tool.InvocationCount() != 0 {
			t.Errorf("expected count 0 after reset, got %d", tool.InvocationCount())
		}
	})

	t.Run("zero max invocations means unlimited", func(t *testing.T) {
		// Arrange
		tool, _ := NewFunctionTool("get_weather", "Get weather", getWeatherFunc)
		tool.MaxInvocations = 0

		// Act - Should succeed for many invocations
		for i := 0; i < 100; i++ {
			_, err := tool.Invoke(context.Background(), nil)
			if err != nil {
				t.Fatalf("invocation %d failed: %v", i, err)
			}
		}
	})
}

func TestFunc(t *testing.T) {
	t.Run("creates tool with extracted function name", func(t *testing.T) {
		// Act
		tool, err := Func(getWeatherFunc)

		// Assert
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// The function name should be extracted from runtime
		if tool.Name() == "" {
			t.Error("expected non-empty name")
		}
	})

	t.Run("applies WithName option", func(t *testing.T) {
		// Act
		tool, err := Func(getWeatherFunc, WithName("custom_name"))

		// Assert
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if tool.Name() != "custom_name" {
			t.Errorf("expected 'custom_name', got %q", tool.Name())
		}
	})

	t.Run("applies WithDescription option", func(t *testing.T) {
		// Act
		tool, err := Func(getWeatherFunc, WithDescription("Custom description"))

		// Assert
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if tool.Description() != "Custom description" {
			t.Errorf("expected 'Custom description', got %q", tool.Description())
		}
	})

	t.Run("applies WithApprovalMode option", func(t *testing.T) {
		// Act
		tool, err := Func(getWeatherFunc, WithApprovalMode(ApprovalAlways))

		// Assert
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if tool.ApprovalMode != ApprovalAlways {
			t.Errorf("expected ApprovalAlways, got %v", tool.ApprovalMode)
		}
	})

	t.Run("applies WithMaxInvocations option", func(t *testing.T) {
		// Act
		tool, err := Func(getWeatherFunc, WithMaxInvocations(5))

		// Assert
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if tool.MaxInvocations != 5 {
			t.Errorf("expected 5, got %d", tool.MaxInvocations)
		}
	})

	t.Run("applies WithProperties option", func(t *testing.T) {
		// Act
		props := AdditionalProperties{"key": "value"}
		tool, err := Func(getWeatherFunc, WithProperties(props))

		// Assert
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if tool.Properties["key"] != "value" {
			t.Errorf("expected 'value', got %v", tool.Properties["key"])
		}
	})

	t.Run("applies multiple options", func(t *testing.T) {
		// Act
		tool, err := Func(getWeatherFunc,
			WithName("weather"),
			WithDescription("Get weather info"),
			WithApprovalMode(ApprovalOnce),
			WithMaxInvocations(10),
		)

		// Assert
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if tool.Name() != "weather" {
			t.Errorf("expected 'weather', got %q", tool.Name())
		}
		if tool.Description() != "Get weather info" {
			t.Errorf("expected 'Get weather info', got %q", tool.Description())
		}
		if tool.ApprovalMode != ApprovalOnce {
			t.Errorf("expected ApprovalOnce, got %v", tool.ApprovalMode)
		}
		if tool.MaxInvocations != 10 {
			t.Errorf("expected 10, got %d", tool.MaxInvocations)
		}
	})

	t.Run("returns error for invalid function", func(t *testing.T) {
		// Act
		_, err := Func(nil)

		// Assert
		if err == nil {
			t.Fatal("expected error for nil function")
		}
	})
}

func TestMustFunc(t *testing.T) {
	t.Run("returns tool for valid function", func(t *testing.T) {
		// Act
		tool := MustFunc(getWeatherFunc, WithName("weather"))

		// Assert
		if tool == nil {
			t.Fatal("expected non-nil tool")
		}
		if tool.Name() != "weather" {
			t.Errorf("expected 'weather', got %q", tool.Name())
		}
	})

	t.Run("panics for invalid function", func(t *testing.T) {
		// Act & Assert
		defer func() {
			if r := recover(); r == nil {
				t.Error("expected panic for nil function")
			}
		}()

		MustFunc(nil)
	})
}

func TestFunctionTool_ImplementsToolInterface(t *testing.T) {
	// Arrange
	tool, err := NewFunctionTool("test", "Test tool", getWeatherFunc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Act & Assert - verify it satisfies the Tool interface
	var iface Tool = tool
	if iface.Name() != "test" {
		t.Errorf("expected 'test', got %q", iface.Name())
	}
	if iface.Description() != "Test tool" {
		t.Errorf("expected 'Test tool', got %q", iface.Description())
	}
	if iface.Parameters() == nil {
		t.Error("expected non-nil parameters")
	}
}
