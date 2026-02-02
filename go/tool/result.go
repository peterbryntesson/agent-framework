// Copyright (c) Microsoft. All rights reserved.

package tool

import (
	"encoding/json"
	"fmt"
)

// Result represents the output of a tool invocation.
// This struct captures both successful outputs and error conditions.
type Result struct {
	// Content is the string representation of the tool's output.
	// For complex results, this may be JSON-encoded.
	Content string `json:"content"`

	// IsError indicates whether this result represents an error condition.
	// When true, Content contains the error message.
	IsError bool `json:"is_error,omitempty"`

	// Metadata contains additional key-value pairs about the result.
	// This can include debugging information, citations, or other context.
	Metadata map[string]any `json:"metadata,omitempty"`

	// RawOutput holds the original output value before string conversion.
	// This is not serialized and is available for programmatic access.
	RawOutput interface{} `json:"-"`
}

// NewResult creates a successful result with the given content.
func NewResult(content string) Result {
	return Result{
		Content: content,
		IsError: false,
	}
}

// NewResultWithMetadata creates a successful result with content and metadata.
func NewResultWithMetadata(content string, metadata map[string]any) Result {
	return Result{
		Content:  content,
		IsError:  false,
		Metadata: metadata,
	}
}

// NewResultFromValue creates a result from any value.
// Strings are used directly; other types are JSON-encoded.
func NewResultFromValue(value interface{}) (Result, error) {
	if value == nil {
		return NewResult(""), nil
	}

	// Handle string values directly
	if s, ok := value.(string); ok {
		return Result{
			Content:   s,
			IsError:   false,
			RawOutput: value,
		}, nil
	}

	// Handle types that implement Stringer
	if s, ok := value.(fmt.Stringer); ok {
		return Result{
			Content:   s.String(),
			IsError:   false,
			RawOutput: value,
		}, nil
	}

	// JSON-encode other types
	data, err := json.Marshal(value)
	if err != nil {
		return Result{}, fmt.Errorf("failed to marshal result: %w", err)
	}

	return Result{
		Content:   string(data),
		IsError:   false,
		RawOutput: value,
	}, nil
}

// NewErrorResult creates an error result with the given message.
func NewErrorResult(message string) Result {
	return Result{
		Content: message,
		IsError: true,
	}
}

// NewErrorResultFromError creates an error result from a Go error.
func NewErrorResultFromError(err error) Result {
	if err == nil {
		return NewResult("")
	}
	return Result{
		Content:   err.Error(),
		IsError:   true,
		RawOutput: err,
	}
}

// NewErrorResultWithDetails creates an error result with additional context.
// When includeDetails is false, only a generic message is returned.
func NewErrorResultWithDetails(err error, includeDetails bool) Result {
	if err == nil {
		return NewResult("")
	}

	content := "Tool invocation failed"
	if includeDetails {
		content = err.Error()
	}

	return Result{
		Content:   content,
		IsError:   true,
		RawOutput: err,
	}
}

// String returns a human-readable representation of the result.
func (r Result) String() string {
	if r.IsError {
		return fmt.Sprintf("Error: %s", r.Content)
	}
	return r.Content
}

// MarshalJSON implements json.Marshaler for Result.
func (r Result) MarshalJSON() ([]byte, error) {
	type resultAlias Result
	return json.Marshal(resultAlias(r))
}

// UnmarshalJSON implements json.Unmarshaler for Result.
func (r *Result) UnmarshalJSON(data []byte) error {
	type resultAlias Result
	aux := (*resultAlias)(r)
	return json.Unmarshal(data, aux)
}

// WithMetadata returns a copy of the result with additional metadata.
func (r Result) WithMetadata(key string, value any) Result {
	if r.Metadata == nil {
		r.Metadata = make(map[string]any)
	}
	r.Metadata[key] = value
	return r
}

// GetMetadata retrieves a metadata value by key.
// Returns nil if the key doesn't exist.
func (r Result) GetMetadata(key string) any {
	if r.Metadata == nil {
		return nil
	}
	return r.Metadata[key]
}
