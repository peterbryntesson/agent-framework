// Copyright (c) Microsoft. All rights reserved.

package groupchat

import (
	"context"

	"github.com/microsoft/agent-framework-go/agent"
)

// HistoryFilter is a function that transforms the message history before
// it is passed to an agent. This can be used to limit context, add system
// messages, or modify the conversation for specific agents.
type HistoryFilter func(ctx context.Context, speaker agent.Agent, messages []agent.Message) []agent.Message

// BeforeTurnCallback is called before each turn begins.
// Return false to skip this turn (move to termination check).
type BeforeTurnCallback func(ctx context.Context, turnCount int, transcript *Transcript) bool

// AfterTurnCallback is called after each turn completes.
// It receives the speaker and their response for post-processing.
type AfterTurnCallback func(ctx context.Context, turnCount int, speaker agent.Agent, messages []agent.Message)

// WithHistoryFilter sets a function to filter/transform message history
// before passing it to each agent.
func WithHistoryFilter(filter HistoryFilter) ManagerOption {
	return func(m *Manager) {
		m.historyFilter = filter
	}
}

// WithBeforeTurnCallback sets a callback to execute before each turn.
func WithBeforeTurnCallback(callback BeforeTurnCallback) ManagerOption {
	return func(m *Manager) {
		m.beforeTurn = callback
	}
}

// WithAfterTurnCallback sets a callback to execute after each turn.
func WithAfterTurnCallback(callback AfterTurnCallback) ManagerOption {
	return func(m *Manager) {
		m.afterTurn = callback
	}
}

// WithSystemPrompt sets a system prompt to prepend to agent messages.
// This is added to every agent invocation as the first message.
func WithSystemPrompt(prompt string) ManagerOption {
	return func(m *Manager) {
		m.systemPrompt = prompt
	}
}

// WithIncludeTranscriptInMessages controls whether the full transcript
// is included in messages sent to agents. Default is true.
// When false, agents only receive the initial message.
func WithIncludeTranscriptInMessages(include bool) ManagerOption {
	return func(m *Manager) {
		m.includeTranscript = include
	}
}

// LastNMessagesFilter returns a HistoryFilter that limits history to the
// last n messages. Useful for limiting context window usage.
func LastNMessagesFilter(n int) HistoryFilter {
	return func(_ context.Context, _ agent.Agent, messages []agent.Message) []agent.Message {
		if n <= 0 || len(messages) <= n {
			return messages
		}
		return messages[len(messages)-n:]
	}
}

// PerAgentHistoryFilter returns a HistoryFilter that applies different
// filters based on the agent ID.
func PerAgentHistoryFilter(filters map[string]HistoryFilter) HistoryFilter {
	return func(ctx context.Context, speaker agent.Agent, messages []agent.Message) []agent.Message {
		if filter, ok := filters[speaker.ID()]; ok {
			return filter(ctx, speaker, messages)
		}
		return messages
	}
}
