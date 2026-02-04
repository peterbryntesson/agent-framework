// Copyright (c) Microsoft. All rights reserved.

package groupchat

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"strings"
	"sync"

	"github.com/microsoft/agent-framework-go/agent"
)

// ErrNoAgents is returned when a selector is created with no agents.
var ErrNoAgents = errors.New("no agents provided")

// ErrSelectionFailed is returned when the LLM selector fails to select an agent.
var ErrSelectionFailed = errors.New("failed to select next agent")

// RoundRobinSelector cycles through agents in order, returning to the first
// after the last agent has spoken.
type RoundRobinSelector struct {
	agents []agent.Agent
	index  int
	mu     sync.Mutex
}

// NewRoundRobinSelector creates a new round-robin selector with the given agents.
// Returns an error if no agents are provided.
func NewRoundRobinSelector(agents ...agent.Agent) (*RoundRobinSelector, error) {
	if len(agents) == 0 {
		return nil, ErrNoAgents
	}
	return &RoundRobinSelector{
		agents: agents,
		index:  0,
	}, nil
}

// SelectNext returns the next agent in the round-robin sequence.
func (s *RoundRobinSelector) SelectNext(_ context.Context, _ *Transcript) (agent.Agent, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	selected := s.agents[s.index]
	s.index = (s.index + 1) % len(s.agents)
	return selected, nil
}

// Reset restarts the round-robin sequence from the first agent.
func (s *RoundRobinSelector) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.index = 0
}

// RandomSelector picks a random agent from the available agents for each turn.
type RandomSelector struct {
	agents []agent.Agent
	rng    *rand.Rand
	mu     sync.Mutex
}

// RandomSelectorOption is a functional option for configuring a RandomSelector.
type RandomSelectorOption func(*RandomSelector)

// WithRNG sets a custom random number generator for the RandomSelector.
// This is useful for testing to ensure deterministic behavior.
func WithRNG(rng *rand.Rand) RandomSelectorOption {
	return func(s *RandomSelector) {
		s.rng = rng
	}
}

// NewRandomSelector creates a new random selector with the given agents.
// Returns an error if no agents are provided.
func NewRandomSelector(agents []agent.Agent, opts ...RandomSelectorOption) (*RandomSelector, error) {
	if len(agents) == 0 {
		return nil, ErrNoAgents
	}
	s := &RandomSelector{
		agents: agents,
	}
	for _, opt := range opts {
		opt(s)
	}
	return s, nil
}

// SelectNext returns a randomly selected agent.
func (s *RandomSelector) SelectNext(_ context.Context, _ *Transcript) (agent.Agent, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var idx int
	if s.rng != nil {
		idx = s.rng.Intn(len(s.agents))
	} else {
		idx = rand.Intn(len(s.agents))
	}
	return s.agents[idx], nil
}

// Reset is a no-op for RandomSelector as it has no state to reset.
func (s *RandomSelector) Reset() {}

// LLMSelector uses an LLM-based agent to decide which agent should speak next.
// It constructs a prompt describing the available agents and the conversation
// so far, then asks the decision agent to select the next speaker.
type LLMSelector struct {
	decisionAgent agent.Agent
	agents        []agent.Agent
	agentsByID    map[string]agent.Agent
	agentsByName  map[string]agent.Agent
	instructions  string
	mu            sync.Mutex
}

// LLMSelectorOption is a functional option for configuring an LLMSelector.
type LLMSelectorOption func(*LLMSelector)

// WithSelectionInstructions sets custom instructions for the decision agent.
// The default instructions ask the agent to select based on conversation context.
func WithSelectionInstructions(instructions string) LLMSelectorOption {
	return func(s *LLMSelector) {
		s.instructions = instructions
	}
}

// NewLLMSelector creates a new LLM-based selector.
// The decisionAgent is used to determine which agent should speak next.
// The agents list contains all agents that can be selected.
// Returns an error if the decision agent is nil or no agents are provided.
func NewLLMSelector(decisionAgent agent.Agent, agents []agent.Agent, opts ...LLMSelectorOption) (*LLMSelector, error) {
	if decisionAgent == nil {
		return nil, errors.New("decision agent cannot be nil")
	}
	if len(agents) == 0 {
		return nil, ErrNoAgents
	}

	// Build lookup maps for agents by ID and name
	agentsByID := make(map[string]agent.Agent, len(agents))
	agentsByName := make(map[string]agent.Agent, len(agents))
	for _, a := range agents {
		agentsByID[a.ID()] = a
		if name := a.Name(); name != "" {
			agentsByName[strings.ToLower(name)] = a
		}
	}

	s := &LLMSelector{
		decisionAgent: decisionAgent,
		agents:        agents,
		agentsByID:    agentsByID,
		agentsByName:  agentsByName,
		instructions:  defaultSelectionInstructions,
	}

	for _, opt := range opts {
		opt(s)
	}

	return s, nil
}

const defaultSelectionInstructions = `You are a conversation moderator. Your task is to select the next speaker in a group conversation.

Based on the conversation history and the available participants, select the most appropriate next speaker.
Consider:
- Who has relevant expertise for the current topic
- Who hasn't spoken recently and may have valuable input
- The natural flow of conversation

Respond with ONLY the name of the selected participant, nothing else.`

// SelectNext uses the decision agent to select the next speaker.
func (s *LLMSelector) SelectNext(ctx context.Context, transcript *Transcript) (agent.Agent, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Build the prompt for the decision agent
	prompt := s.buildSelectionPrompt(transcript)

	// Create messages for the decision agent
	messages := []agent.Message{
		agent.NewSystemMessage(s.instructions),
		agent.NewUserMessage(prompt),
	}

	// Call the decision agent
	response, err := s.decisionAgent.Run(ctx, messages)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrSelectionFailed, err)
	}

	// Parse the response to find the selected agent
	selectedName := strings.TrimSpace(response.Text())
	return s.resolveAgent(selectedName)
}

// Reset is a no-op for LLMSelector as it has no state to reset.
func (s *LLMSelector) Reset() {}

// buildSelectionPrompt creates the prompt for the decision agent.
func (s *LLMSelector) buildSelectionPrompt(transcript *Transcript) string {
	var sb strings.Builder

	// List available participants
	sb.WriteString("Available participants:\n")
	for _, a := range s.agents {
		name := a.Name()
		if name == "" {
			name = a.ID()
		}
		desc := a.Description()
		if desc != "" {
			sb.WriteString(fmt.Sprintf("- %s: %s\n", name, desc))
		} else {
			sb.WriteString(fmt.Sprintf("- %s\n", name))
		}
	}

	// Add conversation history if available
	if transcript != nil && transcript.Len() > 0 {
		sb.WriteString("\nRecent conversation:\n")
		entries := transcript.Entries
		// Limit to last 10 entries to avoid token limits
		start := 0
		if len(entries) > 10 {
			start = len(entries) - 10
		}
		for _, entry := range entries[start:] {
			speaker := entry.SpeakerName
			if speaker == "" {
				speaker = entry.Speaker
			}
			text := entry.Message.Text()
			if len(text) > 200 {
				text = text[:200] + "..."
			}
			sb.WriteString(fmt.Sprintf("%s: %s\n", speaker, text))
		}
	}

	sb.WriteString("\nSelect the next participant to speak:")
	return sb.String()
}

// resolveAgent finds an agent by name or ID from the selection response.
func (s *LLMSelector) resolveAgent(selectedName string) (agent.Agent, error) {
	if selectedName == "" {
		return nil, fmt.Errorf("%w: empty response from decision agent", ErrSelectionFailed)
	}

	// Try exact name match (case-insensitive)
	normalizedName := strings.ToLower(selectedName)
	if a, ok := s.agentsByName[normalizedName]; ok {
		return a, nil
	}

	// Try ID match
	if a, ok := s.agentsByID[selectedName]; ok {
		return a, nil
	}

	// Try partial name match
	for _, a := range s.agents {
		name := strings.ToLower(a.Name())
		if name != "" && strings.Contains(normalizedName, name) {
			return a, nil
		}
		if strings.Contains(normalizedName, strings.ToLower(a.ID())) {
			return a, nil
		}
	}

	return nil, fmt.Errorf("%w: agent %q not found", ErrSelectionFailed, selectedName)
}

// selectionResponse is used for structured output parsing from the LLM.
type selectionResponse struct {
	SelectedAgent string `json:"selected_agent"`
}

// parseStructuredResponse attempts to parse a JSON response from the LLM.
// This is a helper for cases where the decision agent returns structured output.
func parseStructuredResponse(text string) (string, bool) {
	text = strings.TrimSpace(text)
	if !strings.HasPrefix(text, "{") {
		return "", false
	}

	var resp selectionResponse
	if err := json.Unmarshal([]byte(text), &resp); err != nil {
		return "", false
	}

	return resp.SelectedAgent, resp.SelectedAgent != ""
}
