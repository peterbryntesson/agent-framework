// Copyright (c) Microsoft. All rights reserved.

package tool

import (
	"encoding/json"
	"errors"
	"testing"
)

func TestNewResult(t *testing.T) {
	t.Run("creates successful result", func(t *testing.T) {
		// Act
		result := NewResult("success content")

		// Assert
		if result.Content != "success content" {
			t.Errorf("expected content 'success content', got %q", result.Content)
		}
		if result.IsError {
			t.Error("expected IsError to be false")
		}
	})
}

func TestNewResultWithMetadata(t *testing.T) {
	t.Run("creates result with metadata", func(t *testing.T) {
		// Arrange
		metadata := map[string]any{
			"source": "api",
			"count":  42,
		}

		// Act
		result := NewResultWithMetadata("content", metadata)

		// Assert
		if result.Content != "content" {
			t.Errorf("expected content 'content', got %q", result.Content)
		}
		if result.IsError {
			t.Error("expected IsError to be false")
		}
		if result.Metadata["source"] != "api" {
			t.Errorf("expected source 'api', got %v", result.Metadata["source"])
		}
		if result.Metadata["count"] != 42 {
			t.Errorf("expected count 42, got %v", result.Metadata["count"])
		}
	})
}

func TestNewResultFromValue(t *testing.T) {
	t.Run("handles nil value", func(t *testing.T) {
		// Act
		result, err := NewResultFromValue(nil)

		// Assert
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.Content != "" {
			t.Errorf("expected empty content, got %q", result.Content)
		}
	})

	t.Run("handles string value directly", func(t *testing.T) {
		// Act
		result, err := NewResultFromValue("hello")

		// Assert
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.Content != "hello" {
			t.Errorf("expected 'hello', got %q", result.Content)
		}
		if result.RawOutput != "hello" {
			t.Error("expected RawOutput to be set")
		}
	})

	t.Run("handles Stringer interface", func(t *testing.T) {
		// Arrange
		value := stringerType{"formatted value"}

		// Act
		result, err := NewResultFromValue(value)

		// Assert
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.Content != "Stringer: formatted value" {
			t.Errorf("expected 'Stringer: formatted value', got %q", result.Content)
		}
	})

	t.Run("JSON encodes other types", func(t *testing.T) {
		// Arrange
		value := map[string]int{"count": 42}

		// Act
		result, err := NewResultFromValue(value)

		// Assert
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.Content != `{"count":42}` {
			t.Errorf("expected JSON, got %q", result.Content)
		}
	})

	t.Run("handles struct types", func(t *testing.T) {
		// Arrange
		type testData struct {
			Name  string `json:"name"`
			Value int    `json:"value"`
		}
		value := testData{Name: "test", Value: 123}

		// Act
		result, err := NewResultFromValue(value)

		// Assert
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		var parsed testData
		if err := json.Unmarshal([]byte(result.Content), &parsed); err != nil {
			t.Fatalf("failed to parse result: %v", err)
		}
		if parsed.Name != "test" || parsed.Value != 123 {
			t.Errorf("unexpected parsed value: %+v", parsed)
		}
	})
}

type stringerType struct {
	value string
}

func (s stringerType) String() string {
	return "Stringer: " + s.value
}

func TestNewErrorResult(t *testing.T) {
	t.Run("creates error result", func(t *testing.T) {
		// Act
		result := NewErrorResult("error message")

		// Assert
		if result.Content != "error message" {
			t.Errorf("expected 'error message', got %q", result.Content)
		}
		if !result.IsError {
			t.Error("expected IsError to be true")
		}
	})
}

func TestNewErrorResultFromError(t *testing.T) {
	t.Run("creates result from error", func(t *testing.T) {
		// Arrange
		err := errors.New("test error")

		// Act
		result := NewErrorResultFromError(err)

		// Assert
		if result.Content != "test error" {
			t.Errorf("expected 'test error', got %q", result.Content)
		}
		if !result.IsError {
			t.Error("expected IsError to be true")
		}
		if result.RawOutput != err {
			t.Error("expected RawOutput to be the original error")
		}
	})

	t.Run("handles nil error", func(t *testing.T) {
		// Act
		result := NewErrorResultFromError(nil)

		// Assert
		if result.Content != "" {
			t.Errorf("expected empty content, got %q", result.Content)
		}
		if result.IsError {
			t.Error("expected IsError to be false for nil error")
		}
	})
}

func TestNewErrorResultWithDetails(t *testing.T) {
	t.Run("includes details when enabled", func(t *testing.T) {
		// Arrange
		err := errors.New("detailed error message")

		// Act
		result := NewErrorResultWithDetails(err, true)

		// Assert
		if result.Content != "detailed error message" {
			t.Errorf("expected 'detailed error message', got %q", result.Content)
		}
		if !result.IsError {
			t.Error("expected IsError to be true")
		}
	})

	t.Run("hides details when disabled", func(t *testing.T) {
		// Arrange
		err := errors.New("sensitive error message")

		// Act
		result := NewErrorResultWithDetails(err, false)

		// Assert
		if result.Content != "Tool invocation failed" {
			t.Errorf("expected generic message, got %q", result.Content)
		}
		if !result.IsError {
			t.Error("expected IsError to be true")
		}
	})

	t.Run("handles nil error", func(t *testing.T) {
		// Act
		result := NewErrorResultWithDetails(nil, true)

		// Assert
		if result.Content != "" {
			t.Errorf("expected empty content, got %q", result.Content)
		}
	})
}

func TestResult_String(t *testing.T) {
	t.Run("formats success result", func(t *testing.T) {
		// Arrange
		result := NewResult("success content")

		// Act
		str := result.String()

		// Assert
		if str != "success content" {
			t.Errorf("expected 'success content', got %q", str)
		}
	})

	t.Run("formats error result with prefix", func(t *testing.T) {
		// Arrange
		result := NewErrorResult("error content")

		// Act
		str := result.String()

		// Assert
		if str != "Error: error content" {
			t.Errorf("expected 'Error: error content', got %q", str)
		}
	})
}

func TestResult_WithMetadata(t *testing.T) {
	t.Run("adds metadata to result", func(t *testing.T) {
		// Arrange
		result := NewResult("content")

		// Act
		result = result.WithMetadata("key1", "value1").WithMetadata("key2", 42)

		// Assert
		if result.Metadata["key1"] != "value1" {
			t.Errorf("expected key1='value1', got %v", result.Metadata["key1"])
		}
		if result.Metadata["key2"] != 42 {
			t.Errorf("expected key2=42, got %v", result.Metadata["key2"])
		}
	})

	t.Run("initializes metadata if nil", func(t *testing.T) {
		// Arrange
		result := Result{Content: "test"}

		// Act
		result = result.WithMetadata("key", "value")

		// Assert
		if result.Metadata == nil {
			t.Error("expected metadata to be initialized")
		}
		if result.Metadata["key"] != "value" {
			t.Errorf("expected 'value', got %v", result.Metadata["key"])
		}
	})
}

func TestResult_GetMetadata(t *testing.T) {
	t.Run("returns metadata value", func(t *testing.T) {
		// Arrange
		result := NewResult("content").WithMetadata("key", "value")

		// Act
		value := result.GetMetadata("key")

		// Assert
		if value != "value" {
			t.Errorf("expected 'value', got %v", value)
		}
	})

	t.Run("returns nil for missing key", func(t *testing.T) {
		// Arrange
		result := NewResult("content")

		// Act
		value := result.GetMetadata("missing")

		// Assert
		if value != nil {
			t.Errorf("expected nil, got %v", value)
		}
	})

	t.Run("returns nil when metadata is nil", func(t *testing.T) {
		// Arrange
		result := Result{Content: "test"}

		// Act
		value := result.GetMetadata("key")

		// Assert
		if value != nil {
			t.Errorf("expected nil, got %v", value)
		}
	})
}

func TestResult_MarshalJSON(t *testing.T) {
	t.Run("marshals result correctly", func(t *testing.T) {
		// Arrange
		result := Result{
			Content:  "test content",
			IsError:  true,
			Metadata: map[string]any{"key": "value"},
		}

		// Act
		data, err := json.Marshal(result)

		// Assert
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		var parsed map[string]interface{}
		if err := json.Unmarshal(data, &parsed); err != nil {
			t.Fatalf("failed to parse: %v", err)
		}

		if parsed["content"] != "test content" {
			t.Errorf("expected content 'test content', got %v", parsed["content"])
		}
		if parsed["is_error"] != true {
			t.Errorf("expected is_error true, got %v", parsed["is_error"])
		}
	})
}

func TestResult_UnmarshalJSON(t *testing.T) {
	t.Run("unmarshals result correctly", func(t *testing.T) {
		// Arrange
		data := []byte(`{"content":"test","is_error":true,"metadata":{"key":"value"}}`)

		// Act
		var result Result
		err := json.Unmarshal(data, &result)

		// Assert
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.Content != "test" {
			t.Errorf("expected 'test', got %q", result.Content)
		}
		if !result.IsError {
			t.Error("expected IsError to be true")
		}
		if result.Metadata["key"] != "value" {
			t.Errorf("expected key='value', got %v", result.Metadata["key"])
		}
	})
}
