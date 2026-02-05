// Copyright (c) Microsoft. All rights reserved.

package validation

import (
	"errors"
	"testing"

	"github.com/microsoft/agent-framework-go/chat"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testInterface is a sample interface for testing.
type testInterface interface {
	DoSomething()
}

// testStruct is a sample struct for testing.
type testStruct struct {
	Name string
}

func (t *testStruct) DoSomething() {}

func TestRequireNotNil_NonNilValue(t *testing.T) {
	// Arrange
	value := "test"

	// Act
	err := RequireNotNil(value, "value")

	// Assert
	assert.NoError(t, err)
}

func TestRequireNotNil_NonNilPointer(t *testing.T) {
	// Arrange
	s := &testStruct{Name: "test"}

	// Act
	err := RequireNotNil(s, "struct")

	// Assert
	assert.NoError(t, err)
}

func TestRequireNotNil_NonNilSlice(t *testing.T) {
	// Arrange
	slice := []string{"a", "b"}

	// Act
	err := RequireNotNil(slice, "slice")

	// Assert
	assert.NoError(t, err)
}

func TestRequireNotNil_NonNilMap(t *testing.T) {
	// Arrange
	m := map[string]int{"key": 1}

	// Act
	err := RequireNotNil(m, "map")

	// Assert
	assert.NoError(t, err)
}

func TestRequireNotNil_NonNilInterface(t *testing.T) {
	// Arrange
	var iface testInterface = &testStruct{Name: "test"}

	// Act
	err := RequireNotNil(iface, "interface")

	// Assert
	assert.NoError(t, err)
}

func TestRequireNotNil_NilInterface(t *testing.T) {
	// Arrange - nil interface
	var value interface{}

	// Act
	err := RequireNotNil(value, "value")

	// Assert
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrNilValue))

	var validationErr *ValidationError
	require.True(t, errors.As(err, &validationErr))
	assert.Equal(t, "value", validationErr.Field)
	assert.Contains(t, validationErr.Message, "must not be nil")
}

func TestRequireNotNil_NilPointer(t *testing.T) {
	// Arrange
	var ptr *testStruct

	// Act
	err := RequireNotNil(ptr, "pointer")

	// Assert
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrNilValue))

	var validationErr *ValidationError
	require.True(t, errors.As(err, &validationErr))
	assert.Equal(t, "pointer", validationErr.Field)
}

func TestRequireNotNil_NilSlice(t *testing.T) {
	// Arrange
	var slice []string

	// Act
	err := RequireNotNil(slice, "slice")

	// Assert
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrNilValue))
}

func TestRequireNotNil_NilMap(t *testing.T) {
	// Arrange
	var m map[string]int

	// Act
	err := RequireNotNil(m, "map")

	// Assert
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrNilValue))
}

func TestRequireNotNil_TypedNilInterface(t *testing.T) {
	// Arrange - interface containing typed nil
	var ptr *testStruct
	var iface testInterface = ptr

	// Act
	err := RequireNotNil(iface, "interface")

	// Assert
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrNilValue))
}

func TestRequireNotNil_NilChannel(t *testing.T) {
	// Arrange
	var ch chan int

	// Act
	err := RequireNotNil(ch, "channel")

	// Assert
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrNilValue))
}

func TestRequireNotNil_NilFunc(t *testing.T) {
	// Arrange
	var fn func()

	// Act
	err := RequireNotNil(fn, "function")

	// Assert
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrNilValue))
}

func TestRequireNotEmpty_NonEmptyString(t *testing.T) {
	// Arrange
	value := "test"

	// Act
	err := RequireNotEmpty(value, "value")

	// Assert
	assert.NoError(t, err)
}

func TestRequireNotEmpty_StringWithSpaces(t *testing.T) {
	// Arrange
	value := "  test  "

	// Act
	err := RequireNotEmpty(value, "value")

	// Assert
	assert.NoError(t, err)
}

func TestRequireNotEmpty_EmptyString(t *testing.T) {
	// Arrange
	value := ""

	// Act
	err := RequireNotEmpty(value, "value")

	// Assert
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrEmptyValue))

	var validationErr *ValidationError
	require.True(t, errors.As(err, &validationErr))
	assert.Equal(t, "value", validationErr.Field)
	assert.Contains(t, validationErr.Message, "must not be empty")
}

func TestRequireNotEmpty_FieldNameInError(t *testing.T) {
	// Arrange
	value := ""

	// Act
	err := RequireNotEmpty(value, "apiKey")

	// Assert
	require.Error(t, err)

	var validationErr *ValidationError
	require.True(t, errors.As(err, &validationErr))
	assert.Equal(t, "apiKey", validationErr.Field)
}

func TestValidateMessages_ValidSingleMessage(t *testing.T) {
	// Arrange
	messages := []chat.Message{
		chat.NewUserMessage("Hello"),
	}

	// Act
	err := ValidateMessages(messages)

	// Assert
	assert.NoError(t, err)
}

func TestValidateMessages_ValidMultipleMessages(t *testing.T) {
	// Arrange
	messages := []chat.Message{
		chat.NewSystemMessage("You are a helpful assistant"),
		chat.NewUserMessage("Hello"),
		chat.NewAssistantMessage("Hi there!"),
	}

	// Act
	err := ValidateMessages(messages)

	// Assert
	assert.NoError(t, err)
}

func TestValidateMessages_ValidWithToolRole(t *testing.T) {
	// Arrange
	messages := []chat.Message{
		chat.NewUserMessage("Get weather"),
		{Role: chat.RoleTool, ToolCallID: "call_123"},
	}

	// Act
	err := ValidateMessages(messages)

	// Assert
	assert.NoError(t, err)
}

func TestValidateMessages_NilMessages(t *testing.T) {
	// Arrange
	var messages []chat.Message

	// Act
	err := ValidateMessages(messages)

	// Assert
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrNilValue))

	var validationErr *ValidationError
	require.True(t, errors.As(err, &validationErr))
	assert.Equal(t, "messages", validationErr.Field)
}

func TestValidateMessages_EmptyMessages(t *testing.T) {
	// Arrange
	messages := []chat.Message{}

	// Act
	err := ValidateMessages(messages)

	// Assert
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrEmptyMessages))

	var validationErr *ValidationError
	require.True(t, errors.As(err, &validationErr))
	assert.Equal(t, "messages", validationErr.Field)
	assert.Contains(t, validationErr.Message, "at least one message")
}

func TestValidateMessages_EmptyRole(t *testing.T) {
	// Arrange
	messages := []chat.Message{
		chat.NewUserMessage("Hello"),
		{Role: "", Contents: nil},
	}

	// Act
	err := ValidateMessages(messages)

	// Assert
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrInvalidRole))

	var validationErr *ValidationError
	require.True(t, errors.As(err, &validationErr))
	assert.Equal(t, "messages[1].Role", validationErr.Field)
}

func TestValidateMessages_InvalidRole(t *testing.T) {
	// Arrange
	messages := []chat.Message{
		{Role: chat.Role("invalid")},
	}

	// Act
	err := ValidateMessages(messages)

	// Assert
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrInvalidRole))

	var validationErr *ValidationError
	require.True(t, errors.As(err, &validationErr))
	assert.Equal(t, "messages[0].Role", validationErr.Field)
	assert.Contains(t, validationErr.Message, "invalid")
}

func TestValidateMessages_AllValidRoles(t *testing.T) {
	tests := []struct {
		name string
		role chat.Role
	}{
		{"system role", chat.RoleSystem},
		{"user role", chat.RoleUser},
		{"assistant role", chat.RoleAssistant},
		{"tool role", chat.RoleTool},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			messages := []chat.Message{
				{Role: tt.role},
			}

			// Act
			err := ValidateMessages(messages)

			// Assert
			assert.NoError(t, err)
		})
	}
}

func TestValidateMessages_EmptyContentAllowed(t *testing.T) {
	// Arrange - messages without content are valid (e.g., tool calls)
	messages := []chat.Message{
		{Role: chat.RoleAssistant, Contents: nil},
	}

	// Act
	err := ValidateMessages(messages)

	// Assert
	assert.NoError(t, err)
}

func TestValidationError_ErrorMessage(t *testing.T) {
	// Arrange
	err := &ValidationError{
		Field:   "testField",
		Message: "test message",
		Err:     ErrNilValue,
	}

	// Act
	result := err.Error()

	// Assert
	assert.Contains(t, result, "testField")
	assert.Contains(t, result, "test message")
	assert.Contains(t, result, ErrNilValue.Error())
}

func TestValidationError_ErrorMessageWithoutWrappedError(t *testing.T) {
	// Arrange
	err := &ValidationError{
		Field:   "testField",
		Message: "test message",
		Err:     nil,
	}

	// Act
	result := err.Error()

	// Assert
	assert.Contains(t, result, "testField")
	assert.Contains(t, result, "test message")
	assert.NotContains(t, result, "(")
}

func TestValidationError_Unwrap(t *testing.T) {
	// Arrange
	wrappedErr := ErrNilValue
	err := &ValidationError{
		Field:   "testField",
		Message: "test message",
		Err:     wrappedErr,
	}

	// Act
	unwrapped := err.Unwrap()

	// Assert
	assert.Equal(t, wrappedErr, unwrapped)
}

func TestValidationError_ErrorsIs(t *testing.T) {
	// Arrange
	err := &ValidationError{
		Field:   "testField",
		Message: "test message",
		Err:     ErrNilValue,
	}

	// Act & Assert
	assert.True(t, errors.Is(err, ErrNilValue))
	assert.False(t, errors.Is(err, ErrEmptyValue))
}

func TestValidationError_ErrorsAs(t *testing.T) {
	// Arrange
	originalErr := &ValidationError{
		Field:   "testField",
		Message: "test message",
		Err:     ErrNilValue,
	}
	var err error = originalErr

	// Act
	var validationErr *ValidationError
	ok := errors.As(err, &validationErr)

	// Assert
	assert.True(t, ok)
	assert.Equal(t, "testField", validationErr.Field)
	assert.Equal(t, "test message", validationErr.Message)
}
