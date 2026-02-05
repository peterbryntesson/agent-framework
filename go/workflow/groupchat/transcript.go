// Copyright (c) Microsoft. All rights reserved.

package groupchat

import (
	"sync"
	"time"

	"github.com/microsoft/agent-framework-go/agent"
)

// Transcript records the conversation history of a group chat session.
// It is safe for concurrent read access but not for concurrent writes.
type Transcript struct {
	// Entries contains all messages in the conversation in chronological order.
	Entries []TranscriptEntry

	// StartTime is when the group chat started.
	StartTime time.Time

	// EndTime is when the group chat ended (zero if still running).
	EndTime time.Time

	// mu protects Entries during concurrent access.
	mu sync.RWMutex
}

// TranscriptEntry represents a single message in the group chat transcript.
type TranscriptEntry struct {
	// Index is the position of this entry in the transcript (0-based).
	Index int

	// Speaker is the ID of the agent that produced this message.
	Speaker string

	// SpeakerName is the human-readable name of the speaker.
	SpeakerName string

	// Message contains the actual message content.
	Message agent.Message

	// Timestamp is when this message was added to the transcript.
	Timestamp time.Time

	// Turn is the turn number in which this entry was created.
	Turn int
}

// NewTranscript creates a new empty transcript.
func NewTranscript() *Transcript {
	return &Transcript{
		Entries:   make([]TranscriptEntry, 0),
		StartTime: time.Now(),
	}
}

// AddEntry adds a new entry to the transcript.
// This method is not safe for concurrent use; external synchronization is required.
func (t *Transcript) AddEntry(entry TranscriptEntry) {
	t.mu.Lock()
	defer t.mu.Unlock()

	entry.Index = len(t.Entries)
	if entry.Timestamp.IsZero() {
		entry.Timestamp = time.Now()
	}
	t.Entries = append(t.Entries, entry)
}

// AddMessage is a convenience method to add a message from a speaker.
func (t *Transcript) AddMessage(speaker agent.Agent, message agent.Message, turn int) {
	entry := TranscriptEntry{
		Speaker:     speaker.ID(),
		SpeakerName: speaker.Name(),
		Message:     message,
		Turn:        turn,
	}
	t.AddEntry(entry)
}

// AddUserMessage adds a user input message to the transcript.
// User messages are attributed to "user" with no agent ID.
func (t *Transcript) AddUserMessage(message agent.Message, turn int) {
	entry := TranscriptEntry{
		Speaker:     "user",
		SpeakerName: "User",
		Message:     message,
		Turn:        turn,
	}
	t.AddEntry(entry)
}

// Len returns the number of entries in the transcript.
func (t *Transcript) Len() int {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return len(t.Entries)
}

// LastEntry returns the most recent entry in the transcript.
// Returns nil if the transcript is empty.
func (t *Transcript) LastEntry() *TranscriptEntry {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if len(t.Entries) == 0 {
		return nil
	}
	entry := t.Entries[len(t.Entries)-1]
	return &entry
}

// LastSpeaker returns the ID of the most recent speaker.
// Returns empty string if the transcript is empty.
func (t *Transcript) LastSpeaker() string {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if len(t.Entries) == 0 {
		return ""
	}
	return t.Entries[len(t.Entries)-1].Speaker
}

// EntriesBySpeaker returns all entries from a specific speaker.
func (t *Transcript) EntriesBySpeaker(speakerID string) []TranscriptEntry {
	t.mu.RLock()
	defer t.mu.RUnlock()

	result := make([]TranscriptEntry, 0)
	for _, entry := range t.Entries {
		if entry.Speaker == speakerID {
			result = append(result, entry)
		}
	}
	return result
}

// EntriesSince returns all entries added after the given timestamp.
func (t *Transcript) EntriesSince(since time.Time) []TranscriptEntry {
	t.mu.RLock()
	defer t.mu.RUnlock()

	result := make([]TranscriptEntry, 0)
	for _, entry := range t.Entries {
		if entry.Timestamp.After(since) {
			result = append(result, entry)
		}
	}
	return result
}

// ToMessages converts the transcript entries to a slice of agent.Message.
// This is useful for passing conversation history to agents.
func (t *Transcript) ToMessages() []agent.Message {
	t.mu.RLock()
	defer t.mu.RUnlock()

	messages := make([]agent.Message, len(t.Entries))
	for i, entry := range t.Entries {
		messages[i] = entry.Message
	}
	return messages
}

// Clone creates a deep copy of the transcript.
func (t *Transcript) Clone() *Transcript {
	t.mu.RLock()
	defer t.mu.RUnlock()

	entries := make([]TranscriptEntry, len(t.Entries))
	copy(entries, t.Entries)

	return &Transcript{
		Entries:   entries,
		StartTime: t.StartTime,
		EndTime:   t.EndTime,
	}
}

// Complete marks the transcript as complete by setting the end time.
func (t *Transcript) Complete() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.EndTime = time.Now()
}

// IsComplete returns true if the transcript has been marked as complete.
func (t *Transcript) IsComplete() bool {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return !t.EndTime.IsZero()
}

// Duration returns the duration of the group chat.
// If the chat is still running, it returns the elapsed time since start.
func (t *Transcript) Duration() time.Duration {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if t.EndTime.IsZero() {
		return time.Since(t.StartTime)
	}
	return t.EndTime.Sub(t.StartTime)
}
