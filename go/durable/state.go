// Copyright (c) Microsoft. All rights reserved.

package durable

import (
	"encoding/json"
	"time"

	"github.com/microsoft/agent-framework-go/chat"
)

// SchemaVersion is the current state schema version for cross-platform compatibility.
// This version must match the schema version used by .NET and Python implementations.
const SchemaVersion = "1.1.0"

// State represents the persisted agent state.
// This structure matches the cross-platform schema in schemas/durable-agent-entity-state.json.
type State struct {
	// SchemaVersion indicates the version of the state schema.
	// This is used for cross-platform compatibility.
	SchemaVersion string `json:"schemaVersion"`

	// Data contains the actual state data including conversation history.
	Data StateData `json:"data"`
}

// StateData contains the conversation history and metadata.
type StateData struct {
	// ConversationHistory is the ordered list of conversation entries (requests and responses).
	ConversationHistory []StateEntry `json:"conversationHistory"`

	// ExpirationTimeUtc is the optional expiration time for this session.
	// If set and the current time exceeds this value, the session may be deleted.
	ExpirationTimeUtc *time.Time `json:"expirationTimeUtc,omitempty"`
}

// stateDataJSON is the JSON representation of StateData for custom unmarshaling.
type stateDataJSON struct {
	ConversationHistory []json.RawMessage `json:"conversationHistory"`
	ExpirationTimeUtc   *time.Time        `json:"expirationTimeUtc,omitempty"`
}

// UnmarshalJSON implements json.Unmarshaler for StateData.
func (d *StateData) UnmarshalJSON(data []byte) error {
	var aux stateDataJSON
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	d.ExpirationTimeUtc = aux.ExpirationTimeUtc
	d.ConversationHistory = make([]StateEntry, 0, len(aux.ConversationHistory))

	for _, raw := range aux.ConversationHistory {
		entry, err := UnmarshalStateEntry(raw)
		if err != nil {
			return err
		}
		d.ConversationHistory = append(d.ConversationHistory, entry)
	}

	return nil
}

// MarshalJSON implements json.Marshaler for StateData.
func (d StateData) MarshalJSON() ([]byte, error) {
	// We need to marshal StateEntry items as their concrete types
	history := make([]json.RawMessage, len(d.ConversationHistory))
	for i, entry := range d.ConversationHistory {
		data, err := json.Marshal(entry)
		if err != nil {
			return nil, err
		}
		history[i] = data
	}

	aux := stateDataJSON{
		ConversationHistory: history,
		ExpirationTimeUtc:   d.ExpirationTimeUtc,
	}
	return json.Marshal(aux)
}

// NewState creates a new empty state with the current schema version.
func NewState() *State {
	return &State{
		SchemaVersion: SchemaVersion,
		Data: StateData{
			ConversationHistory: make([]StateEntry, 0),
		},
	}
}

// AppendRequest adds a request entry to the conversation history.
func (s *State) AppendRequest(entry *RequestEntry) {
	if s.Data.ConversationHistory == nil {
		s.Data.ConversationHistory = make([]StateEntry, 0)
	}
	s.Data.ConversationHistory = append(s.Data.ConversationHistory, entry)
}

// AppendResponse adds a response entry to the conversation history.
func (s *State) AppendResponse(entry *ResponseEntry) {
	if s.Data.ConversationHistory == nil {
		s.Data.ConversationHistory = make([]StateEntry, 0)
	}
	s.Data.ConversationHistory = append(s.Data.ConversationHistory, entry)
}

// BuildChatMessages converts the conversation history to a slice of chat.Message.
// This is used to provide context to the agent when processing a new request.
func (s *State) BuildChatMessages() []chat.Message {
	var messages []chat.Message
	for _, entry := range s.Data.ConversationHistory {
		entryMessages := entry.ToChatMessages()
		messages = append(messages, entryMessages...)
	}
	return messages
}

// SetExpiration sets the expiration time for this session.
func (s *State) SetExpiration(t time.Time) {
	s.Data.ExpirationTimeUtc = &t
}

// ClearExpiration removes the expiration time for this session.
func (s *State) ClearExpiration() {
	s.Data.ExpirationTimeUtc = nil
}

// IsExpired returns true if the session has expired.
func (s *State) IsExpired() bool {
	if s.Data.ExpirationTimeUtc == nil {
		return false
	}
	return time.Now().UTC().After(*s.Data.ExpirationTimeUtc)
}

// MarshalJSON implements json.Marshaler.
func (s *State) MarshalJSON() ([]byte, error) {
	type Alias State
	return json.Marshal((*Alias)(s))
}

// UnmarshalJSON implements json.Unmarshaler.
func (s *State) UnmarshalJSON(data []byte) error {
	type Alias State
	aux := &struct {
		*Alias
	}{
		Alias: (*Alias)(s),
	}
	return json.Unmarshal(data, aux)
}

// Clone creates a deep copy of the state.
func (s *State) Clone() *State {
	if s == nil {
		return nil
	}

	clone := &State{
		SchemaVersion: s.SchemaVersion,
		Data: StateData{
			ConversationHistory: make([]StateEntry, len(s.Data.ConversationHistory)),
		},
	}

	copy(clone.Data.ConversationHistory, s.Data.ConversationHistory)

	if s.Data.ExpirationTimeUtc != nil {
		expTime := *s.Data.ExpirationTimeUtc
		clone.Data.ExpirationTimeUtc = &expTime
	}

	return clone
}

// MessageCount returns the total number of messages across all conversation entries.
func (s *State) MessageCount() int {
	count := 0
	for _, entry := range s.Data.ConversationHistory {
		count += entry.MessageCount()
	}
	return count
}
