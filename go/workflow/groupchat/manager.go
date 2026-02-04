// Copyright (c) Microsoft. All rights reserved.

package groupchat

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/microsoft/agent-framework-go/agent"
)

// ErrMaxTurnsExceeded is returned when the maximum number of turns is reached.
var ErrMaxTurnsExceeded = errors.New("maximum turns exceeded")

// ErrNoSpeakerSelected is returned when the selector fails to choose a speaker.
var ErrNoSpeakerSelected = errors.New("no speaker selected")

// ErrChatCancelled is returned when the group chat is cancelled via context.
var ErrChatCancelled = errors.New("group chat cancelled")

// Manager orchestrates multi-agent group chat conversations.
// It uses a Selector to determine which agent speaks next and manages
// the conversation flow with configurable termination conditions.
type Manager struct {
	agents            []agent.Agent
	agentsByID        map[string]agent.Agent
	selector          Selector
	termination       TerminationCondition
	maxTurns          int
	historyFilter     HistoryFilter
	beforeTurn        BeforeTurnCallback
	afterTurn         AfterTurnCallback
	systemPrompt      string
	includeTranscript bool
	mu                sync.Mutex
}

// ManagerOption configures a Manager.
type ManagerOption func(*Manager)

// WithSelector sets the speaker selection strategy.
// If not provided, a round-robin selector is used by default.
func WithSelector(selector Selector) ManagerOption {
	return func(m *Manager) {
		m.selector = selector
	}
}

// WithMaxTurns sets the maximum number of turns before automatic termination.
// Default is 40 turns.
func WithMaxTurns(maxTurns int) ManagerOption {
	return func(m *Manager) {
		if maxTurns > 0 {
			m.maxTurns = maxTurns
		}
	}
}

// WithTerminationCondition sets a custom termination condition.
// This is evaluated after each turn in addition to the max turns limit.
func WithTerminationCondition(condition TerminationCondition) ManagerOption {
	return func(m *Manager) {
		m.termination = condition
	}
}

// NewManager creates a new group chat manager with the given agents.
// Returns an error if no agents are provided.
func NewManager(agents []agent.Agent, opts ...ManagerOption) (*Manager, error) {
	if len(agents) == 0 {
		return nil, ErrNoAgents
	}

	// Build lookup map by agent ID
	agentsByID := make(map[string]agent.Agent, len(agents))
	for _, a := range agents {
		agentsByID[a.ID()] = a
	}

	m := &Manager{
		agents:            agents,
		agentsByID:        agentsByID,
		maxTurns:          40, // default from .NET implementation
		includeTranscript: true,
	}

	// Apply options
	for _, opt := range opts {
		opt(m)
	}

	// Create default selector if not provided
	if m.selector == nil {
		selector, _ := NewRoundRobinSelector(agents...)
		m.selector = selector
	}

	return m, nil
}

// Run executes the group chat synchronously and returns the final result.
// The initialMessage is the starting prompt that kicks off the conversation.
// Use RunStream for real-time event streaming during execution.
func (m *Manager) Run(ctx context.Context, initialMessage string) (*Result, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Reset selector state
	m.selector.Reset()

	// Initialize transcript
	transcript := NewTranscript()

	// Add initial message if provided
	if initialMessage != "" {
		userMsg := agent.NewUserMessage(initialMessage)
		transcript.AddUserMessage(userMsg, 0)
	}

	turnCount := 0
	var terminationReason string

	// Main turn loop
	for {
		// Check context cancellation
		if ctx.Err() != nil {
			return &Result{
				Transcript:        transcript,
				TurnCount:         turnCount,
				TerminationReason: "cancelled",
				Error:             ErrChatCancelled,
			}, ErrChatCancelled
		}

		// Check max turns limit before incrementing (so turnCount reflects completed turns)
		if turnCount >= m.maxTurns {
			terminationReason = fmt.Sprintf("maximum turns (%d) reached", m.maxTurns)
			break
		}

		turnCount++

		// Check custom termination condition
		if m.termination != nil && m.termination(ctx, transcript, turnCount) {
			terminationReason = "termination condition met"
			break
		}

		// Execute before turn callback
		if m.beforeTurn != nil && !m.beforeTurn(ctx, turnCount, transcript) {
			terminationReason = "before turn callback returned false"
			break
		}

		// Select next speaker
		speaker, err := m.selector.SelectNext(ctx, transcript)
		if err != nil {
			return &Result{
				Transcript:        transcript,
				TurnCount:         turnCount,
				TerminationReason: "selection failed",
				Error:             fmt.Errorf("%w: %v", ErrNoSpeakerSelected, err),
			}, err
		}

		// Build messages for the agent
		messages := m.buildMessagesForAgent(ctx, speaker, transcript, initialMessage)

		// Invoke the selected agent
		response, err := speaker.Run(ctx, messages)
		if err != nil {
			return &Result{
				Transcript:        transcript,
				TurnCount:         turnCount,
				TerminationReason: "agent error",
				Error:             fmt.Errorf("agent %q failed: %w", speaker.Name(), err),
			}, err
		}

		// Add response messages to transcript
		for _, msg := range response.Messages {
			transcript.AddMessage(speaker, msg, turnCount)
		}

		// Execute after turn callback
		if m.afterTurn != nil {
			m.afterTurn(ctx, turnCount, speaker, response.Messages)
		}
	}

	// Mark transcript as complete
	transcript.Complete()

	return &Result{
		Transcript:        transcript,
		TurnCount:         turnCount,
		TerminationReason: terminationReason,
	}, nil
}

// RunStream executes the group chat and streams events through a channel.
// The channel is closed when the chat completes or encounters an error.
// Events include turn progress, speaker selection, agent responses, and completion.
func (m *Manager) RunStream(ctx context.Context, initialMessage string) (<-chan Event, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	events := make(chan Event, 100)

	go func() {
		defer close(events)
		m.runStreamInternal(ctx, initialMessage, events)
	}()

	return events, nil
}

// runStreamInternal contains the main streaming execution logic.
func (m *Manager) runStreamInternal(ctx context.Context, initialMessage string, events chan<- Event) {
	// Reset selector state
	m.selector.Reset()

	// Initialize transcript
	transcript := NewTranscript()

	// Emit started event
	events <- NewStartedEvent()

	// Add initial message if provided
	if initialMessage != "" {
		userMsg := agent.NewUserMessage(initialMessage)
		transcript.AddUserMessage(userMsg, 0)
	}

	turnCount := 0
	var terminationReason string

	// Main turn loop
	for {
		// Check context cancellation
		if ctx.Err() != nil {
			events <- NewErrorEvent(turnCount, ErrChatCancelled)
			return
		}

		// Check max turns limit before incrementing (so turnCount reflects completed turns)
		if turnCount >= m.maxTurns {
			terminationReason = fmt.Sprintf("maximum turns (%d) reached", m.maxTurns)
			events <- NewTerminatingEvent(turnCount, terminationReason)
			break
		}

		turnCount++

		// Check custom termination condition
		if m.termination != nil && m.termination(ctx, transcript, turnCount) {
			terminationReason = "termination condition met"
			events <- NewTerminatingEvent(turnCount, terminationReason)
			break
		}

		// Execute before turn callback
		if m.beforeTurn != nil && !m.beforeTurn(ctx, turnCount, transcript) {
			terminationReason = "before turn callback returned false"
			events <- NewTerminatingEvent(turnCount, terminationReason)
			break
		}

		// Emit turn started event
		events <- NewTurnStartedEvent(turnCount)

		// Select next speaker
		speaker, err := m.selector.SelectNext(ctx, transcript)
		if err != nil {
			events <- NewErrorEvent(turnCount, fmt.Errorf("%w: %v", ErrNoSpeakerSelected, err))
			return
		}

		speakerID := speaker.ID()
		speakerName := speaker.Name()
		if speakerName == "" {
			speakerName = speakerID
		}

		// Emit speaker selected event
		events <- NewSpeakerSelectedEvent(turnCount, speakerID, speakerName)

		// Emit agent invoked event
		events <- NewAgentInvokedEvent(turnCount, speakerID, speakerName)

		// Build messages for the agent
		messages := m.buildMessagesForAgent(ctx, speaker, transcript, initialMessage)

		// Invoke the selected agent with streaming
		updates, err := speaker.RunStream(ctx, messages)
		if err != nil {
			events <- NewErrorEvent(turnCount, fmt.Errorf("agent %q failed: %w", speakerName, err))
			return
		}

		// Collect response while streaming updates
		var responseMessages []agent.Message
		for update := range updates {
			// Emit streaming update event
			events <- NewAgentResponseUpdateEvent(turnCount, speakerID, speakerName, update)

			// Collect completed messages
			if update.Kind == agent.UpdateKindMessageComplete && update.Message != nil {
				responseMessages = append(responseMessages, *update.Message)
			}
		}

		// Add response messages to transcript and emit response events
		for _, msg := range responseMessages {
			transcript.AddMessage(speaker, msg, turnCount)
			events <- NewAgentResponseEvent(turnCount, speakerID, speakerName, msg)
		}

		// Execute after turn callback
		if m.afterTurn != nil {
			m.afterTurn(ctx, turnCount, speaker, responseMessages)
		}

		// Emit turn completed event
		events <- NewTurnCompletedEvent(turnCount, speakerID, speakerName)
	}

	// Mark transcript as complete
	transcript.Complete()

	// Emit completed event
	events <- NewCompletedEvent(turnCount, transcript)
}

// buildMessagesForAgent constructs the message list to send to an agent.
// It applies the system prompt, transcript inclusion, and history filter.
func (m *Manager) buildMessagesForAgent(ctx context.Context, speaker agent.Agent, transcript *Transcript, initialMessage string) []agent.Message {
	var messages []agent.Message

	// Add system prompt if configured
	if m.systemPrompt != "" {
		messages = append(messages, agent.NewSystemMessage(m.systemPrompt))
	}

	// Include transcript or just initial message
	if m.includeTranscript {
		messages = append(messages, transcript.ToMessages()...)
	} else if initialMessage != "" {
		messages = append(messages, agent.NewUserMessage(initialMessage))
	}

	// Apply history filter if configured
	if m.historyFilter != nil {
		messages = m.historyFilter(ctx, speaker, messages)
	}

	return messages
}

// Agents returns the list of agents participating in the group chat.
func (m *Manager) Agents() []agent.Agent {
	return m.agents
}

// AgentByID returns an agent by its ID, or nil if not found.
func (m *Manager) AgentByID(id string) agent.Agent {
	return m.agentsByID[id]
}

// MaxTurns returns the configured maximum number of turns.
func (m *Manager) MaxTurns() int {
	return m.maxTurns
}
