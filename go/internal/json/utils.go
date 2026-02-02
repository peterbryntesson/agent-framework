// Copyright (c) Microsoft. All rights reserved.

package json

import (
	"encoding/json"
	"errors"
	"reflect"
)

var (
	// ErrNilTarget indicates that the target parameter for unmarshaling is nil.
	ErrNilTarget = errors.New("target must not be nil")

	// ErrNonPointerTarget indicates that the target is not a pointer type.
	ErrNonPointerTarget = errors.New("target must be a pointer")
)

// MarshalToRawMessage marshals the given value to a json.RawMessage.
// Returns nil if the input value is nil (nil interface or nil pointer).
// Returns an error if marshaling fails.
func MarshalToRawMessage(v interface{}) (json.RawMessage, error) {
	if v == nil {
		return nil, nil
	}

	// Check for nil pointer or nil interface containing a typed nil
	rv := reflect.ValueOf(v)
	if rv.Kind() == reflect.Ptr && rv.IsNil() {
		return nil, nil
	}

	data, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}

	return json.RawMessage(data), nil
}

// UnmarshalFromRawMessage unmarshals the given json.RawMessage into the target value.
// Returns nil if the data is nil or empty.
// Returns ErrNilTarget if target is nil.
// Returns ErrNonPointerTarget if target is not a pointer.
// Returns an error if unmarshaling fails.
func UnmarshalFromRawMessage(data json.RawMessage, target interface{}) error {
	// Handle nil or empty data
	if len(data) == 0 {
		return nil
	}

	if target == nil {
		return ErrNilTarget
	}

	rv := reflect.ValueOf(target)
	if rv.Kind() != reflect.Ptr {
		return ErrNonPointerTarget
	}

	if rv.IsNil() {
		return ErrNilTarget
	}

	return json.Unmarshal(data, target)
}
