// Copyright (c) Microsoft. All rights reserved.

package tool

import (
	"testing"
)

func TestDefaultInvocationConfig(t *testing.T) {
	t.Run("returns correct defaults", func(t *testing.T) {
		// Act
		config := DefaultInvocationConfig()

		// Assert
		if !config.Enabled {
			t.Error("expected Enabled to be true")
		}
		if config.MaxIterations != DefaultMaxIterations {
			t.Errorf("expected MaxIterations %d, got %d", DefaultMaxIterations, config.MaxIterations)
		}
		if config.MaxConsecutiveErrors != DefaultMaxConsecutiveErrors {
			t.Errorf("expected MaxConsecutiveErrors %d, got %d", DefaultMaxConsecutiveErrors, config.MaxConsecutiveErrors)
		}
		if config.TerminateOnUnknownCalls {
			t.Error("expected TerminateOnUnknownCalls to be false")
		}
		if config.IncludeDetailedErrors {
			t.Error("expected IncludeDetailedErrors to be false")
		}
		if config.ParallelToolCalls != true {
			t.Error("expected ParallelToolCalls to be true")
		}
	})
}

func TestDisabledInvocationConfig(t *testing.T) {
	t.Run("returns config with Enabled=false", func(t *testing.T) {
		// Act
		config := DisabledInvocationConfig()

		// Assert
		if config.Enabled {
			t.Error("expected Enabled to be false")
		}
	})
}

func TestInvocationConfig_WithMaxIterations(t *testing.T) {
	t.Run("sets max iterations", func(t *testing.T) {
		// Arrange
		config := DefaultInvocationConfig()

		// Act
		result := config.WithMaxIterations(100)

		// Assert
		if result.MaxIterations != 100 {
			t.Errorf("expected 100, got %d", result.MaxIterations)
		}
		// Original should be unchanged
		if config.MaxIterations != DefaultMaxIterations {
			t.Error("original config was modified")
		}
	})
}

func TestInvocationConfig_WithMaxConsecutiveErrors(t *testing.T) {
	t.Run("sets max consecutive errors", func(t *testing.T) {
		// Arrange
		config := DefaultInvocationConfig()

		// Act
		result := config.WithMaxConsecutiveErrors(5)

		// Assert
		if result.MaxConsecutiveErrors != 5 {
			t.Errorf("expected 5, got %d", result.MaxConsecutiveErrors)
		}
	})
}

func TestInvocationConfig_WithAdditionalTools(t *testing.T) {
	t.Run("appends additional tools", func(t *testing.T) {
		// Arrange
		config := DefaultInvocationConfig()
		tool1 := &testTool{name: "tool1"}
		tool2 := &testTool{name: "tool2"}

		// Act
		result := config.WithAdditionalTools(tool1, tool2)

		// Assert
		if len(result.AdditionalTools) != 2 {
			t.Errorf("expected 2 tools, got %d", len(result.AdditionalTools))
		}
	})
}

func TestInvocationConfig_WithDetailedErrors(t *testing.T) {
	t.Run("sets detailed errors flag", func(t *testing.T) {
		// Arrange
		config := DefaultInvocationConfig()

		// Act
		result := config.WithDetailedErrors(true)

		// Assert
		if !result.IncludeDetailedErrors {
			t.Error("expected IncludeDetailedErrors to be true")
		}
	})
}

func TestInvocationConfig_WithIntermediateSteps(t *testing.T) {
	t.Run("sets intermediate steps flag", func(t *testing.T) {
		// Arrange
		config := DefaultInvocationConfig()

		// Act
		result := config.WithIntermediateSteps(true)

		// Assert
		if !result.ReturnIntermediateSteps {
			t.Error("expected ReturnIntermediateSteps to be true")
		}
	})
}

func TestInvocationConfig_WithTimeout(t *testing.T) {
	t.Run("sets timeout", func(t *testing.T) {
		// Arrange
		config := DefaultInvocationConfig()

		// Act
		result := config.WithTimeout(30)

		// Assert
		if result.TimeoutSeconds != 30 {
			t.Errorf("expected 30, got %d", result.TimeoutSeconds)
		}
	})
}

func TestInvocationConfig_WithParallelCalls(t *testing.T) {
	t.Run("sets parallel calls flag", func(t *testing.T) {
		// Arrange
		config := DefaultInvocationConfig()

		// Act
		result := config.WithParallelCalls(false)

		// Assert
		if result.ParallelToolCalls {
			t.Error("expected ParallelToolCalls to be false")
		}
	})
}

func TestInvocationConfig_Merge(t *testing.T) {
	t.Run("merges non-zero values from other", func(t *testing.T) {
		// Arrange
		base := DefaultInvocationConfig()
		other := InvocationConfig{
			Enabled:                 true,
			MaxIterations:           50,
			MaxConsecutiveErrors:    10,
			TerminateOnUnknownCalls: true,
			IncludeDetailedErrors:   true,
			ReturnIntermediateSteps: true,
			TimeoutSeconds:          60,
			ParallelToolCalls:       true,
		}

		// Act
		result := base.Merge(other)

		// Assert
		if result.MaxIterations != 50 {
			t.Errorf("expected MaxIterations 50, got %d", result.MaxIterations)
		}
		if result.MaxConsecutiveErrors != 10 {
			t.Errorf("expected MaxConsecutiveErrors 10, got %d", result.MaxConsecutiveErrors)
		}
		if !result.TerminateOnUnknownCalls {
			t.Error("expected TerminateOnUnknownCalls to be true")
		}
		if !result.IncludeDetailedErrors {
			t.Error("expected IncludeDetailedErrors to be true")
		}
		if result.TimeoutSeconds != 60 {
			t.Errorf("expected TimeoutSeconds 60, got %d", result.TimeoutSeconds)
		}
	})

	t.Run("merges additional tools", func(t *testing.T) {
		// Arrange
		base := DefaultInvocationConfig()
		base.AdditionalTools = []Tool{&testTool{name: "base_tool"}}
		other := InvocationConfig{
			Enabled:         true,
			AdditionalTools: []Tool{&testTool{name: "other_tool"}},
		}

		// Act
		result := base.Merge(other)

		// Assert
		if len(result.AdditionalTools) != 2 {
			t.Errorf("expected 2 tools, got %d", len(result.AdditionalTools))
		}
	})

	t.Run("disables when other.Enabled is false", func(t *testing.T) {
		// Arrange
		base := DefaultInvocationConfig()
		other := InvocationConfig{
			Enabled: false,
		}

		// Act
		result := base.Merge(other)

		// Assert
		if result.Enabled {
			t.Error("expected Enabled to be false")
		}
	})

	t.Run("disables parallel when other.ParallelToolCalls is false", func(t *testing.T) {
		// Arrange
		base := DefaultInvocationConfig()
		other := InvocationConfig{
			Enabled:           true,
			ParallelToolCalls: false,
		}

		// Act
		result := base.Merge(other)

		// Assert
		if result.ParallelToolCalls {
			t.Error("expected ParallelToolCalls to be false")
		}
	})
}

func TestInvocationConfig_Validate(t *testing.T) {
	t.Run("returns nil for valid config", func(t *testing.T) {
		// Arrange
		config := DefaultInvocationConfig()

		// Act
		err := config.Validate()

		// Assert
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})
}
