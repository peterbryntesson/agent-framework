// Copyright (c) Microsoft. All rights reserved.

package tool

import (
	"errors"
	"testing"
)

func TestSentinelErrors(t *testing.T) {
	tests := []struct {
		name string
		err  error
		msg  string
	}{
		{"ErrUnknownTool", ErrUnknownTool, "unknown tool"},
		{"ErrInvalidArguments", ErrInvalidArguments, "invalid arguments"},
		{"ErrMaxIterations", ErrMaxIterations, "max iterations exceeded"},
		{"ErrConsecutiveErrors", ErrConsecutiveErrors, "max consecutive errors exceeded"},
		{"ErrInvocationDisabled", ErrInvocationDisabled, "tool invocation is disabled"},
		{"ErrToolTimeout", ErrToolTimeout, "tool invocation timeout"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.err.Error() != tc.msg {
				t.Errorf("expected %q, got %q", tc.msg, tc.err.Error())
			}
		})
	}
}

func TestInvocationError(t *testing.T) {
	t.Run("Error formats with cause", func(t *testing.T) {
		// Arrange
		cause := errors.New("underlying error")
		err := NewInvocationError("my_tool", cause)

		// Act
		msg := err.Error()

		// Assert
		expected := `tool "my_tool" invocation failed: underlying error`
		if msg != expected {
			t.Errorf("expected %q, got %q", expected, msg)
		}
	})

	t.Run("Error formats without cause", func(t *testing.T) {
		// Arrange
		err := NewInvocationError("my_tool", nil)

		// Act
		msg := err.Error()

		// Assert
		expected := `tool "my_tool" invocation failed`
		if msg != expected {
			t.Errorf("expected %q, got %q", expected, msg)
		}
	})

	t.Run("Unwrap returns cause", func(t *testing.T) {
		// Arrange
		cause := errors.New("cause")
		err := NewInvocationError("tool", cause)

		// Act
		unwrapped := err.Unwrap()

		// Assert
		if unwrapped != cause {
			t.Error("expected unwrapped to be cause")
		}
	})

	t.Run("errors.Is works with wrapped error", func(t *testing.T) {
		// Arrange
		cause := ErrToolTimeout
		err := NewInvocationError("tool", cause)

		// Act & Assert
		if !errors.Is(err, ErrToolTimeout) {
			t.Error("expected errors.Is to match ErrToolTimeout")
		}
	})
}

func TestInvocationPanicError(t *testing.T) {
	t.Run("Error formats correctly", func(t *testing.T) {
		// Arrange
		err := NewInvocationPanicError("panic_tool", "panic value", []byte("stack trace"))

		// Act
		msg := err.Error()

		// Assert
		expected := `tool "panic_tool" panicked: panic value`
		if msg != expected {
			t.Errorf("expected %q, got %q", expected, msg)
		}
	})

	t.Run("StackTrace returns stack", func(t *testing.T) {
		// Arrange
		stack := []byte("goroutine 1 [running]:\nmain.panic()")
		err := NewInvocationPanicError("tool", "value", stack)

		// Act
		trace := err.StackTrace()

		// Assert
		if trace != string(stack) {
			t.Errorf("expected %q, got %q", string(stack), trace)
		}
	})
}

func TestUnknownToolError(t *testing.T) {
	t.Run("Error formats correctly", func(t *testing.T) {
		// Arrange
		err := NewUnknownToolError("missing_tool", []string{"tool1", "tool2"})

		// Act
		msg := err.Error()

		// Assert
		expected := `unknown tool: "missing_tool"`
		if msg != expected {
			t.Errorf("expected %q, got %q", expected, msg)
		}
	})

	t.Run("Is matches ErrUnknownTool", func(t *testing.T) {
		// Arrange
		err := NewUnknownToolError("missing", nil)

		// Act & Assert
		if !errors.Is(err, ErrUnknownTool) {
			t.Error("expected errors.Is to match ErrUnknownTool")
		}
	})

	t.Run("stores available tools", func(t *testing.T) {
		// Arrange
		available := []string{"tool1", "tool2", "tool3"}
		err := NewUnknownToolError("missing", available)

		// Assert
		if len(err.AvailableTools) != 3 {
			t.Errorf("expected 3 available tools, got %d", len(err.AvailableTools))
		}
	})
}

func TestArgumentError(t *testing.T) {
	t.Run("Error formats with cause", func(t *testing.T) {
		// Arrange
		cause := errors.New("JSON syntax error")
		err := NewArgumentError("my_tool", "failed to parse", cause)

		// Act
		msg := err.Error()

		// Assert
		expected := `invalid arguments for tool "my_tool": failed to parse: JSON syntax error`
		if msg != expected {
			t.Errorf("expected %q, got %q", expected, msg)
		}
	})

	t.Run("Error formats without cause", func(t *testing.T) {
		// Arrange
		err := NewArgumentError("my_tool", "missing required field", nil)

		// Act
		msg := err.Error()

		// Assert
		expected := `invalid arguments for tool "my_tool": missing required field`
		if msg != expected {
			t.Errorf("expected %q, got %q", expected, msg)
		}
	})

	t.Run("Unwrap returns cause", func(t *testing.T) {
		// Arrange
		cause := errors.New("cause")
		err := NewArgumentError("tool", "message", cause)

		// Act
		unwrapped := err.Unwrap()

		// Assert
		if unwrapped != cause {
			t.Error("expected unwrapped to be cause")
		}
	})

	t.Run("Is matches ErrInvalidArguments", func(t *testing.T) {
		// Arrange
		err := NewArgumentError("tool", "message", nil)

		// Act & Assert
		if !errors.Is(err, ErrInvalidArguments) {
			t.Error("expected errors.Is to match ErrInvalidArguments")
		}
	})
}
