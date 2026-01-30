// Copyright (c) Microsoft. All rights reserved.

package agent

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestApplyRunOptions_DefaultConfig(t *testing.T) {
	// Arrange - no options provided

	// Act
	cfg := ApplyRunOptions()

	// Assert
	assert.NotNil(t, cfg, "config should not be nil")
	assert.Nil(t, cfg.Session, "session should be nil by default")
	assert.Nil(t, cfg.Tools, "tools should be nil by default")
	assert.Equal(t, 0, cfg.MaxTokens, "max tokens should default to 0")
	assert.Equal(t, float32(0), cfg.Temperature, "temperature should default to 0")
	assert.NotNil(t, cfg.Metadata, "metadata should be initialized")
	assert.Empty(t, cfg.Metadata, "metadata should be empty by default")
}

func TestApplyRunOptions_NilOption(t *testing.T) {
	// Arrange
	var nilOption RunOption

	// Act - should not panic
	cfg := ApplyRunOptions(nilOption)

	// Assert
	assert.NotNil(t, cfg, "config should not be nil even with nil option")
}

func TestWithSession_SetsSession(t *testing.T) {
	// Arrange
	session := NewInMemorySession()

	// Act
	cfg := ApplyRunOptions(WithSession(session))

	// Assert
	assert.Equal(t, session, cfg.Session, "session should be set")
	assert.Equal(t, session.ID(), cfg.Session.ID(), "session ID should match")
}

func TestWithSession_NilSession(t *testing.T) {
	// Arrange
	var session Session

	// Act
	cfg := ApplyRunOptions(WithSession(session))

	// Assert
	assert.Nil(t, cfg.Session, "session should be nil when nil is passed")
}

func TestWithTools_AddsSingleTool(t *testing.T) {
	// Arrange
	tool := "mock-tool"

	// Act
	cfg := ApplyRunOptions(WithTools(tool))

	// Assert
	require.Len(t, cfg.Tools, 1, "should have one tool")
	assert.Equal(t, "mock-tool", cfg.Tools[0], "tool should match")
}

func TestWithTools_AddsMultipleTools(t *testing.T) {
	// Arrange
	tool1 := "tool1"
	tool2 := "tool2"
	tool3 := "tool3"

	// Act
	cfg := ApplyRunOptions(WithTools(tool1, tool2, tool3))

	// Assert
	require.Len(t, cfg.Tools, 3, "should have three tools")
	assert.Equal(t, "tool1", cfg.Tools[0])
	assert.Equal(t, "tool2", cfg.Tools[1])
	assert.Equal(t, "tool3", cfg.Tools[2])
}

func TestWithTools_Accumulates(t *testing.T) {
	// Arrange
	tool1 := "tool1"
	tool2 := "tool2"

	// Act - multiple WithTools calls should accumulate
	cfg := ApplyRunOptions(
		WithTools(tool1),
		WithTools(tool2),
	)

	// Assert
	require.Len(t, cfg.Tools, 2, "should have two tools from accumulated calls")
	assert.Equal(t, "tool1", cfg.Tools[0])
	assert.Equal(t, "tool2", cfg.Tools[1])
}

func TestWithTools_EmptyVariadic(t *testing.T) {
	// Arrange - no tools in variadic call

	// Act
	cfg := ApplyRunOptions(WithTools())

	// Assert
	assert.Nil(t, cfg.Tools, "tools should be nil when no tools provided")
}

func TestWithMaxTokens_SetsValue(t *testing.T) {
	// Arrange
	maxTokens := 4096

	// Act
	cfg := ApplyRunOptions(WithMaxTokens(maxTokens))

	// Assert
	assert.Equal(t, 4096, cfg.MaxTokens, "max tokens should be set")
}

func TestWithMaxTokens_ZeroValue(t *testing.T) {
	// Arrange
	maxTokens := 0

	// Act
	cfg := ApplyRunOptions(WithMaxTokens(maxTokens))

	// Assert
	assert.Equal(t, 0, cfg.MaxTokens, "max tokens should be 0")
}

func TestWithMaxTokens_NegativeValue(t *testing.T) {
	// Arrange - negative values are allowed (validation is caller's responsibility)
	maxTokens := -1

	// Act
	cfg := ApplyRunOptions(WithMaxTokens(maxTokens))

	// Assert
	assert.Equal(t, -1, cfg.MaxTokens, "negative max tokens should be set")
}

func TestWithTemperature_SetsValue(t *testing.T) {
	// Arrange
	temperature := float32(0.7)

	// Act
	cfg := ApplyRunOptions(WithTemperature(temperature))

	// Assert
	assert.Equal(t, float32(0.7), cfg.Temperature, "temperature should be set")
}

func TestWithTemperature_ZeroValue(t *testing.T) {
	// Arrange
	temperature := float32(0.0)

	// Act
	cfg := ApplyRunOptions(WithTemperature(temperature))

	// Assert
	assert.Equal(t, float32(0.0), cfg.Temperature, "temperature should be 0")
}

func TestWithTemperature_MaxValue(t *testing.T) {
	// Arrange
	temperature := float32(2.0) // Some providers support up to 2.0

	// Act
	cfg := ApplyRunOptions(WithTemperature(temperature))

	// Assert
	assert.Equal(t, float32(2.0), cfg.Temperature, "high temperature should be set")
}

func TestWithMetadata_AddsEntries(t *testing.T) {
	// Arrange
	metadata := map[string]interface{}{
		"key1": "value1",
		"key2": 42,
	}

	// Act
	cfg := ApplyRunOptions(WithMetadata(metadata))

	// Assert
	require.Len(t, cfg.Metadata, 2, "should have two metadata entries")
	assert.Equal(t, "value1", cfg.Metadata["key1"])
	assert.Equal(t, 42, cfg.Metadata["key2"])
}

func TestWithMetadata_MergesMultipleCalls(t *testing.T) {
	// Arrange
	metadata1 := map[string]interface{}{"key1": "value1"}
	metadata2 := map[string]interface{}{"key2": "value2"}

	// Act
	cfg := ApplyRunOptions(
		WithMetadata(metadata1),
		WithMetadata(metadata2),
	)

	// Assert
	require.Len(t, cfg.Metadata, 2, "should have merged metadata")
	assert.Equal(t, "value1", cfg.Metadata["key1"])
	assert.Equal(t, "value2", cfg.Metadata["key2"])
}

func TestWithMetadata_OverwritesDuplicateKeys(t *testing.T) {
	// Arrange
	metadata1 := map[string]interface{}{"key": "original"}
	metadata2 := map[string]interface{}{"key": "updated"}

	// Act
	cfg := ApplyRunOptions(
		WithMetadata(metadata1),
		WithMetadata(metadata2),
	)

	// Assert
	require.Len(t, cfg.Metadata, 1, "should have one entry after overwrite")
	assert.Equal(t, "updated", cfg.Metadata["key"], "later value should overwrite")
}

func TestWithMetadata_EmptyMap(t *testing.T) {
	// Arrange
	metadata := map[string]interface{}{}

	// Act
	cfg := ApplyRunOptions(WithMetadata(metadata))

	// Assert
	assert.NotNil(t, cfg.Metadata, "metadata should not be nil")
	assert.Empty(t, cfg.Metadata, "metadata should be empty")
}

func TestWithMetadata_NilMap(t *testing.T) {
	// Arrange
	var metadata map[string]interface{}

	// Act
	cfg := ApplyRunOptions(WithMetadata(metadata))

	// Assert
	assert.NotNil(t, cfg.Metadata, "metadata should not be nil")
	assert.Empty(t, cfg.Metadata, "metadata should be empty when nil passed")
}

func TestApplyRunOptions_CombinedOptions(t *testing.T) {
	// Arrange
	session := NewInMemorySession()
	tool1 := "weather"
	tool2 := "calculator"
	maxTokens := 1000
	temperature := float32(0.5)
	metadata := map[string]interface{}{
		"user_id":    "user-123",
		"request_id": "req-456",
	}

	// Act
	cfg := ApplyRunOptions(
		WithSession(session),
		WithTools(tool1, tool2),
		WithMaxTokens(maxTokens),
		WithTemperature(temperature),
		WithMetadata(metadata),
	)

	// Assert
	assert.Equal(t, session, cfg.Session, "session should be set")
	require.Len(t, cfg.Tools, 2, "should have two tools")
	assert.Equal(t, 1000, cfg.MaxTokens, "max tokens should be set")
	assert.Equal(t, float32(0.5), cfg.Temperature, "temperature should be set")
	assert.Len(t, cfg.Metadata, 2, "should have two metadata entries")
}

func TestApplyRunOptions_OptionOrder(t *testing.T) {
	// Arrange - later options should override earlier ones for same property

	// Act
	cfg := ApplyRunOptions(
		WithMaxTokens(100),
		WithMaxTokens(200),
		WithTemperature(0.3),
		WithTemperature(0.9),
	)

	// Assert
	assert.Equal(t, 200, cfg.MaxTokens, "later max tokens should override")
	assert.Equal(t, float32(0.9), cfg.Temperature, "later temperature should override")
}

func TestRunOptionType(t *testing.T) {
	// Arrange - verify RunOption is a function type

	// Act
	var opt RunOption = func(cfg *RunConfig) {
		cfg.MaxTokens = 999
	}
	cfg := ApplyRunOptions(opt)

	// Assert
	assert.Equal(t, 999, cfg.MaxTokens, "custom option should work")
}

func TestRunConfig_FieldsAccessible(t *testing.T) {
	// Arrange
	cfg := &RunConfig{
		Session:     NewInMemorySession(),
		Tools:       []interface{}{"tool1"},
		MaxTokens:   500,
		Temperature: 0.8,
		Metadata:    map[string]interface{}{"k": "v"},
	}

	// Assert - all fields are accessible
	assert.NotNil(t, cfg.Session)
	assert.Len(t, cfg.Tools, 1)
	assert.Equal(t, 500, cfg.MaxTokens)
	assert.Equal(t, float32(0.8), cfg.Temperature)
	assert.Len(t, cfg.Metadata, 1)
}
