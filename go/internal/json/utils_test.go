// Copyright (c) Microsoft. All rights reserved.

package json

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testStruct is a sample struct for testing JSON operations.
type testStruct struct {
	Name  string `json:"name"`
	Value int    `json:"value"`
}

func TestMarshalToRawMessage_Struct(t *testing.T) {
	// Arrange
	input := testStruct{Name: "test", Value: 42}

	// Act
	result, err := MarshalToRawMessage(input)

	// Assert
	require.NoError(t, err)
	assert.JSONEq(t, `{"name":"test","value":42}`, string(result))
}

func TestMarshalToRawMessage_Map(t *testing.T) {
	// Arrange
	input := map[string]interface{}{
		"key1": "value1",
		"key2": 123,
	}

	// Act
	result, err := MarshalToRawMessage(input)

	// Assert
	require.NoError(t, err)
	assert.JSONEq(t, `{"key1":"value1","key2":123}`, string(result))
}

func TestMarshalToRawMessage_Slice(t *testing.T) {
	// Arrange
	input := []string{"a", "b", "c"}

	// Act
	result, err := MarshalToRawMessage(input)

	// Assert
	require.NoError(t, err)
	assert.JSONEq(t, `["a","b","c"]`, string(result))
}

func TestMarshalToRawMessage_Primitives(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected string
	}{
		{"string", "hello", `"hello"`},
		{"int", 42, `42`},
		{"float", 3.14, `3.14`},
		{"bool_true", true, `true`},
		{"bool_false", false, `false`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Act
			result, err := MarshalToRawMessage(tt.input)

			// Assert
			require.NoError(t, err)
			assert.Equal(t, tt.expected, string(result))
		})
	}
}

func TestMarshalToRawMessage_NilInterface(t *testing.T) {
	// Arrange
	var input interface{}

	// Act
	result, err := MarshalToRawMessage(input)

	// Assert
	require.NoError(t, err)
	assert.Nil(t, result)
}

func TestMarshalToRawMessage_NilPointer(t *testing.T) {
	// Arrange
	var input *testStruct

	// Act
	result, err := MarshalToRawMessage(input)

	// Assert
	require.NoError(t, err)
	assert.Nil(t, result)
}

func TestMarshalToRawMessage_NonNilPointer(t *testing.T) {
	// Arrange
	input := &testStruct{Name: "pointer", Value: 100}

	// Act
	result, err := MarshalToRawMessage(input)

	// Assert
	require.NoError(t, err)
	assert.JSONEq(t, `{"name":"pointer","value":100}`, string(result))
}

func TestMarshalToRawMessage_EmptyStruct(t *testing.T) {
	// Arrange
	input := testStruct{}

	// Act
	result, err := MarshalToRawMessage(input)

	// Assert
	require.NoError(t, err)
	assert.JSONEq(t, `{"name":"","value":0}`, string(result))
}

func TestMarshalToRawMessage_EmptySlice(t *testing.T) {
	// Arrange
	input := []string{}

	// Act
	result, err := MarshalToRawMessage(input)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, "[]", string(result))
}

func TestMarshalToRawMessage_EmptyMap(t *testing.T) {
	// Arrange
	input := map[string]string{}

	// Act
	result, err := MarshalToRawMessage(input)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, "{}", string(result))
}

func TestMarshalToRawMessage_UnmarshalableValue(t *testing.T) {
	// Arrange - channels cannot be marshaled
	input := make(chan int)

	// Act
	result, err := MarshalToRawMessage(input)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestUnmarshalFromRawMessage_Struct(t *testing.T) {
	// Arrange
	data := json.RawMessage(`{"name":"test","value":42}`)
	var target testStruct

	// Act
	err := UnmarshalFromRawMessage(data, &target)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, "test", target.Name)
	assert.Equal(t, 42, target.Value)
}

func TestUnmarshalFromRawMessage_Map(t *testing.T) {
	// Arrange
	data := json.RawMessage(`{"key1":"value1","key2":"value2"}`)
	var target map[string]string

	// Act
	err := UnmarshalFromRawMessage(data, &target)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, "value1", target["key1"])
	assert.Equal(t, "value2", target["key2"])
}

func TestUnmarshalFromRawMessage_Slice(t *testing.T) {
	// Arrange
	data := json.RawMessage(`["a","b","c"]`)
	var target []string

	// Act
	err := UnmarshalFromRawMessage(data, &target)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, []string{"a", "b", "c"}, target)
}

func TestUnmarshalFromRawMessage_NilData(t *testing.T) {
	// Arrange
	var data json.RawMessage
	var target testStruct

	// Act
	err := UnmarshalFromRawMessage(data, &target)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, testStruct{}, target)
}

func TestUnmarshalFromRawMessage_EmptyData(t *testing.T) {
	// Arrange
	data := json.RawMessage{}
	var target testStruct

	// Act
	err := UnmarshalFromRawMessage(data, &target)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, testStruct{}, target)
}

func TestUnmarshalFromRawMessage_NilTarget(t *testing.T) {
	// Arrange
	data := json.RawMessage(`{"name":"test"}`)

	// Act
	err := UnmarshalFromRawMessage(data, nil)

	// Assert
	assert.ErrorIs(t, err, ErrNilTarget)
}

func TestUnmarshalFromRawMessage_NilPointerTarget(t *testing.T) {
	// Arrange
	data := json.RawMessage(`{"name":"test"}`)
	var target *testStruct

	// Act
	err := UnmarshalFromRawMessage(data, target)

	// Assert
	assert.ErrorIs(t, err, ErrNilTarget)
}

func TestUnmarshalFromRawMessage_NonPointerTarget(t *testing.T) {
	// Arrange
	data := json.RawMessage(`{"name":"test"}`)
	var target testStruct

	// Act
	err := UnmarshalFromRawMessage(data, target)

	// Assert
	assert.ErrorIs(t, err, ErrNonPointerTarget)
}

func TestUnmarshalFromRawMessage_InvalidJSON(t *testing.T) {
	// Arrange
	data := json.RawMessage(`{invalid json}`)
	var target testStruct

	// Act
	err := UnmarshalFromRawMessage(data, &target)

	// Assert
	assert.Error(t, err)
}

func TestUnmarshalFromRawMessage_TypeMismatch(t *testing.T) {
	// Arrange
	data := json.RawMessage(`"string"`)
	var target int

	// Act
	err := UnmarshalFromRawMessage(data, &target)

	// Assert
	assert.Error(t, err)
}

func TestRoundTrip_Struct(t *testing.T) {
	// Arrange
	original := testStruct{Name: "roundtrip", Value: 999}

	// Act
	data, err := MarshalToRawMessage(original)
	require.NoError(t, err)

	var result testStruct
	err = UnmarshalFromRawMessage(data, &result)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, original, result)
}

func TestRoundTrip_ComplexStructure(t *testing.T) {
	// Arrange
	type nested struct {
		Items []testStruct          `json:"items"`
		Meta  map[string]string     `json:"meta"`
		Tags  []string              `json:"tags"`
		Extra map[string]testStruct `json:"extra,omitempty"`
	}

	original := nested{
		Items: []testStruct{
			{Name: "first", Value: 1},
			{Name: "second", Value: 2},
		},
		Meta: map[string]string{
			"version": "1.0",
			"author":  "test",
		},
		Tags: []string{"go", "json", "test"},
		Extra: map[string]testStruct{
			"special": {Name: "extra", Value: 100},
		},
	}

	// Act
	data, err := MarshalToRawMessage(original)
	require.NoError(t, err)

	var result nested
	err = UnmarshalFromRawMessage(data, &result)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, original, result)
}

func TestSentinelErrors_HaveDescriptiveMessages(t *testing.T) {
	// Arrange & Act & Assert
	assert.Equal(t, "target must not be nil", ErrNilTarget.Error())
	assert.Equal(t, "target must be a pointer", ErrNonPointerTarget.Error())
}
