// Copyright (c) Microsoft. All rights reserved.

package agent

import (
	"encoding/json"
	"reflect"
)

// Session represents an agent conversation session.
// Sessions maintain conversation history and state across multiple agent runs.
// This interface will be fully implemented in User Story 1.2.3.
type Session interface {
	// ID returns the unique identifier for this session.
	ID() string

	// Messages returns the conversation history for this session.
	Messages() []Message

	// AddMessage appends a message to the session history.
	AddMessage(msg Message)

	// Serialize converts the session state to JSON for persistence.
	Serialize() (json.RawMessage, error)

	// GetService retrieves a service of the specified type from the session.
	// Returns nil if the service type is not available.
	GetService(serviceType reflect.Type) interface{}
}
