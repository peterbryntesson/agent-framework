// Copyright (c) Microsoft. All rights reserved.

package durable

import (
	"encoding/json"
	"fmt"
	"strings"
)

// entityNamePrefix is the prefix used for Temporal workflow IDs.
const entityNamePrefix = "dafx-"

// SessionID uniquely identifies a durable agent session.
// It combines an agent name and a unique key to create a stable identifier
// that can be used to resume sessions across process restarts.
type SessionID struct {
	// Name is the name of the agent that owns this session (case-insensitive).
	Name string

	// Key is the unique key for this session (case-sensitive).
	Key string
}

// NewSessionID creates a new session ID with the given agent name and key.
func NewSessionID(name, key string) SessionID {
	return SessionID{Name: name, Key: key}
}

// WorkflowID returns the Temporal workflow ID for this session.
// The format is "@dafx-{name}@{key}" to align with cross-platform entity IDs.
func (id SessionID) WorkflowID() string {
	return fmt.Sprintf("@%s@%s", id.EntityName(), id.Key)
}

// EntityName returns the entity name portion used in the workflow ID.
// This matches the .NET convention of "dafx-{agentName}".
func (id SessionID) EntityName() string {
	return entityNamePrefix + id.Name
}

// ParseSessionID parses a session ID string back to a SessionID.
// Supported formats:
// - "@name@key" (preferred session ID format)
// - "@dafx-name@key" (entity workflow ID format)
// - "dafx-name-key" (legacy workflow ID format)
func ParseSessionID(sessionID string) (SessionID, error) {
	if strings.HasPrefix(sessionID, "@") {
		parts := strings.SplitN(sessionID[1:], "@", 2)
		if len(parts) != 2 {
			return SessionID{}, fmt.Errorf("invalid session ID format: missing separator")
		}
		name := parts[0]
		key := parts[1]
		if name == "" || key == "" {
			return SessionID{}, fmt.Errorf("invalid session ID format: empty name or key")
		}
		if strings.HasPrefix(name, entityNamePrefix) {
			name = strings.TrimPrefix(name, entityNamePrefix)
		}
		return SessionID{Name: name, Key: key}, nil
	}

	if !strings.HasPrefix(sessionID, entityNamePrefix) {
		return SessionID{}, fmt.Errorf("invalid session ID format: missing prefix %q", entityNamePrefix)
	}

	rest := strings.TrimPrefix(sessionID, entityNamePrefix)
	idx := strings.Index(rest, "-")
	if idx < 0 {
		return SessionID{}, fmt.Errorf("invalid session ID format: missing separator")
	}

	name := rest[:idx]
	key := rest[idx+1:]

	if name == "" || key == "" {
		return SessionID{}, fmt.Errorf("invalid session ID format: empty name or key")
	}

	return SessionID{Name: name, Key: key}, nil
}

// String returns the string representation of the session ID.
// The format is "@name@key".
func (id SessionID) String() string {
	return fmt.Sprintf("@%s@%s", id.Name, id.Key)
}

// IsZero returns true if this session ID is uninitialized.
func (id SessionID) IsZero() bool {
	return id.Name == "" && id.Key == ""
}

// Equals returns true if this session ID equals another.
// Comparison is case-insensitive for the name and case-sensitive for the key.
func (id SessionID) Equals(other SessionID) bool {
	return strings.EqualFold(id.Name, other.Name) && id.Key == other.Key
}

// MarshalJSON implements json.Marshaler.
// The session ID is serialized as its string representation.
func (id SessionID) MarshalJSON() ([]byte, error) {
	return json.Marshal(id.String())
}

// UnmarshalJSON implements json.Unmarshaler.
// The session ID is deserialized from its string representation.
func (id *SessionID) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}

	parsed, err := ParseSessionID(s)
	if err != nil {
		return err
	}

	*id = parsed
	return nil
}

// MarshalText implements encoding.TextMarshaler.
func (id SessionID) MarshalText() ([]byte, error) {
	return []byte(id.String()), nil
}

// UnmarshalText implements encoding.TextUnmarshaler.
func (id *SessionID) UnmarshalText(data []byte) error {
	parsed, err := ParseSessionID(string(data))
	if err != nil {
		return err
	}
	*id = parsed
	return nil
}
