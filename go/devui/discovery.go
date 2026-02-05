// Copyright (c) Microsoft. All rights reserved.

package devui

import (
	"sync"

	"github.com/microsoft/agent-framework-go/agent"
)

// Registry manages registered agents for the DevUI server.
type Registry struct {
	agents map[string]registeredAgent
	mu     sync.RWMutex
}

type registeredAgent struct {
	agent agent.Agent
	info  AgentInfo
}

// NewRegistry creates a new agent registry.
func NewRegistry() *Registry {
	return &Registry{
		agents: make(map[string]registeredAgent),
	}
}

// Register adds an agent to the registry.
// If an agent with the same name exists, it is replaced.
func (r *Registry) Register(name string, a agent.Agent) {
	r.mu.Lock()
	defer r.mu.Unlock()

	metadata := a.Metadata()
	metadataMap := make(map[string]string)
	if metadata.ProviderName != "" {
		metadataMap["provider"] = metadata.ProviderName
	}

	r.agents[name] = registeredAgent{
		agent: a,
		info: AgentInfo{
			Name:        name,
			ID:          a.ID(),
			Description: a.Description(),
			Metadata:    metadataMap,
		},
	}
}

// Unregister removes an agent from the registry.
func (r *Registry) Unregister(name string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.agents, name)
}

// Get retrieves an agent by name.
// Returns nil if not found.
func (r *Registry) Get(name string) agent.Agent {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if reg, ok := r.agents[name]; ok {
		return reg.agent
	}
	return nil
}

// GetInfo retrieves agent info by name.
// Returns zero value and false if not found.
func (r *Registry) GetInfo(name string) (AgentInfo, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if reg, ok := r.agents[name]; ok {
		return reg.info, true
	}
	return AgentInfo{}, false
}

// List returns info for all registered agents.
func (r *Registry) List() []AgentInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()

	infos := make([]AgentInfo, 0, len(r.agents))
	for _, reg := range r.agents {
		infos = append(infos, reg.info)
	}
	return infos
}

// Count returns the number of registered agents.
func (r *Registry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.agents)
}
