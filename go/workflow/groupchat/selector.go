// Copyright (c) Microsoft. All rights reserved.

package groupchat

import (
	"context"

	"github.com/microsoft/agent-framework-go/agent"
)

// Selector determines which agent speaks next in a group chat.
// Implementations can use various strategies including round-robin,
// random selection, or LLM-based decision making.
type Selector interface {
	// SelectNext chooses the next agent to speak based on the conversation transcript.
	// The implementation should return one of the registered agents.
	// Returns an error if selection fails or no suitable agent is available.
	SelectNext(ctx context.Context, transcript *Transcript) (agent.Agent, error)

	// Reset clears any internal state in the selector.
	// This is called when starting a new group chat session.
	Reset()
}

// SelectorFunc is a function type that implements the Selector interface.
// It enables simple function-based selectors without needing a full struct.
type SelectorFunc func(ctx context.Context, transcript *Transcript) (agent.Agent, error)

// SelectNext implements the Selector interface.
func (f SelectorFunc) SelectNext(ctx context.Context, transcript *Transcript) (agent.Agent, error) {
	return f(ctx, transcript)
}

// Reset implements the Selector interface (no-op for function-based selectors).
func (f SelectorFunc) Reset() {}

// TerminationCondition determines whether the group chat should end.
// Returns true if the chat should terminate, false otherwise.
type TerminationCondition func(ctx context.Context, transcript *Transcript, turnCount int) bool

// MaxTurnsCondition creates a termination condition that stops after maxTurns.
func MaxTurnsCondition(maxTurns int) TerminationCondition {
	return func(_ context.Context, _ *Transcript, turnCount int) bool {
		return turnCount >= maxTurns
	}
}

// KeywordCondition creates a termination condition that stops when a keyword is found
// in the last message of the transcript.
func KeywordCondition(keyword string) TerminationCondition {
	return func(_ context.Context, transcript *Transcript, _ int) bool {
		if len(transcript.Entries) == 0 {
			return false
		}
		lastEntry := transcript.Entries[len(transcript.Entries)-1]
		text := extractTextFromMessage(lastEntry.Message)
		return containsKeyword(text, keyword)
	}
}

// containsKeyword checks if the text contains the specified keyword.
func containsKeyword(text, keyword string) bool {
	return len(keyword) > 0 && len(text) >= len(keyword) && containsSubstring(text, keyword)
}

// containsSubstring checks if s contains substr (case-insensitive).
func containsSubstring(s, substr string) bool {
	if len(substr) > len(s) {
		return false
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		if equalsFoldASCII(s[i:i+len(substr)], substr) {
			return true
		}
	}
	return false
}

// equalsFoldASCII performs a case-insensitive ASCII comparison.
func equalsFoldASCII(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := 0; i < len(a); i++ {
		ca, cb := a[i], b[i]
		if ca >= 'A' && ca <= 'Z' {
			ca += 'a' - 'A'
		}
		if cb >= 'A' && cb <= 'Z' {
			cb += 'a' - 'A'
		}
		if ca != cb {
			return false
		}
	}
	return true
}

// extractTextFromMessage extracts the text content from a message.
// It uses the Message.Text() method which concatenates all text content.
func extractTextFromMessage(msg agent.Message) string {
	return msg.Text()
}

// CombineConditions creates a termination condition that triggers when any of
// the provided conditions returns true (OR logic).
func CombineConditions(conditions ...TerminationCondition) TerminationCondition {
	return func(ctx context.Context, transcript *Transcript, turnCount int) bool {
		for _, cond := range conditions {
			if cond(ctx, transcript, turnCount) {
				return true
			}
		}
		return false
	}
}

// AllConditions creates a termination condition that triggers only when all
// provided conditions return true (AND logic).
func AllConditions(conditions ...TerminationCondition) TerminationCondition {
	return func(ctx context.Context, transcript *Transcript, turnCount int) bool {
		for _, cond := range conditions {
			if !cond(ctx, transcript, turnCount) {
				return false
			}
		}
		return len(conditions) > 0
	}
}
