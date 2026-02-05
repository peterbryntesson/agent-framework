// Copyright (c) Microsoft. All rights reserved.

package validation

import (
	"errors"
	"fmt"
	"reflect"

	"github.com/microsoft/agent-framework-go/chat"
)

var (
	// ErrNilValue indicates that a required value was nil.
	ErrNilValue = errors.New("value must not be nil")

	// ErrEmptyValue indicates that a required string value was empty.
	ErrEmptyValue = errors.New("value must not be empty")

	// ErrEmptyMessages indicates that a messages slice was empty when at least one message is required.
	ErrEmptyMessages = errors.New("messages must not be empty")

	// ErrInvalidRole indicates that a message has an invalid or empty role.
	ErrInvalidRole = errors.New("message has invalid role")

	// ErrEmptyContent indicates that a message has no content.
	// This sentinel is provided for callers that need to validate message content.
	// ValidateMessages does not use this error because empty content is valid
	// for certain message types (e.g., tool calls without text).
	ErrEmptyContent = errors.New("message has empty content")
)

// ValidationError represents a validation error with context about the failed validation.
type ValidationError struct {
	// Field is the name of the field or parameter that failed validation.
	Field string

	// Message describes the validation failure.
	Message string

	// Err is the underlying error, if any.
	Err error
}

// Error returns the formatted error message.
func (e *ValidationError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("validation failed for %q: %s (%v)", e.Field, e.Message, e.Err)
	}
	return fmt.Sprintf("validation failed for %q: %s", e.Field, e.Message)
}

// Unwrap returns the underlying error for error chain inspection.
func (e *ValidationError) Unwrap() error {
	return e.Err
}

// RequireNotNil validates that the given value is not nil.
// Returns a ValidationError if v is nil (nil interface or nil pointer).
// The name parameter identifies the field or parameter being validated.
func RequireNotNil(v interface{}, name string) error {
	if v == nil {
		return &ValidationError{
			Field:   name,
			Message: "must not be nil",
			Err:     ErrNilValue,
		}
	}

	// Check for typed nil (e.g., (*SomeType)(nil))
	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.Ptr, reflect.Interface, reflect.Slice, reflect.Map, reflect.Chan, reflect.Func:
		if rv.IsNil() {
			return &ValidationError{
				Field:   name,
				Message: "must not be nil",
				Err:     ErrNilValue,
			}
		}
	}

	return nil
}

// RequireNotEmpty validates that the given string is not empty.
// Returns a ValidationError if s is the empty string ("").
// The name parameter identifies the field or parameter being validated.
//
// Note: This function does not trim whitespace. Strings containing only
// whitespace characters are considered non-empty.
func RequireNotEmpty(s string, name string) error {
	if s == "" {
		return &ValidationError{
			Field:   name,
			Message: "must not be empty",
			Err:     ErrEmptyValue,
		}
	}

	return nil
}

// ValidateMessages validates that the given messages slice is valid for use in agent operations.
// Returns an error if:
//   - The messages slice is nil or empty
//   - Any message has an invalid role
//
// Note: Empty content is allowed as some messages (like tool calls) may not have text content.
func ValidateMessages(messages []chat.Message) error {
	if messages == nil {
		return &ValidationError{
			Field:   "messages",
			Message: "must not be nil",
			Err:     ErrNilValue,
		}
	}

	if len(messages) == 0 {
		return &ValidationError{
			Field:   "messages",
			Message: "must contain at least one message",
			Err:     ErrEmptyMessages,
		}
	}

	validRoles := map[chat.Role]bool{
		chat.RoleSystem:    true,
		chat.RoleUser:      true,
		chat.RoleAssistant: true,
		chat.RoleTool:      true,
	}

	for i, msg := range messages {
		if msg.Role == "" {
			return &ValidationError{
				Field:   fmt.Sprintf("messages[%d].Role", i),
				Message: "must not be empty",
				Err:     ErrInvalidRole,
			}
		}

		if !validRoles[msg.Role] {
			return &ValidationError{
				Field:   fmt.Sprintf("messages[%d].Role", i),
				Message: fmt.Sprintf("invalid role %q", msg.Role),
				Err:     ErrInvalidRole,
			}
		}
	}

	return nil
}
