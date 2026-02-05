// Copyright (c) Microsoft. All rights reserved.

package devui

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRegistry_Register(t *testing.T) {
	registry := NewRegistry()

	agent := &mockAgentFull{
		id:          "test-id",
		name:        "test-agent",
		description: "Test agent description",
	}

	registry.Register("test", agent)

	result := registry.Get("test")
	require.NotNil(t, result)
	assert.Equal(t, "test-id", result.ID())
}

func TestRegistry_Unregister(t *testing.T) {
	registry := NewRegistry()

	agent := &mockAgentFull{id: "test-id", name: "test"}
	registry.Register("test", agent)

	require.NotNil(t, registry.Get("test"))

	registry.Unregister("test")

	assert.Nil(t, registry.Get("test"))
}

func TestRegistry_List(t *testing.T) {
	registry := NewRegistry()

	registry.Register("agent1", &mockAgentFull{id: "id1", name: "Agent 1"})
	registry.Register("agent2", &mockAgentFull{id: "id2", name: "Agent 2"})

	list := registry.List()

	assert.Len(t, list, 2)
}

func TestRegistry_GetInfo(t *testing.T) {
	registry := NewRegistry()

	agent := &mockAgentFull{
		id:          "test-id",
		name:        "test-agent",
		description: "Test description",
	}
	registry.Register("test", agent)

	info, ok := registry.GetInfo("test")

	assert.True(t, ok)
	assert.Equal(t, "test", info.Name)
	assert.Equal(t, "test-id", info.ID)
	assert.Equal(t, "Test description", info.Description)
}

func TestRegistry_GetInfo_NotFound(t *testing.T) {
	registry := NewRegistry()

	info, ok := registry.GetInfo("nonexistent")

	assert.False(t, ok)
	assert.Empty(t, info.Name)
}

func TestRegistry_Count(t *testing.T) {
	registry := NewRegistry()

	assert.Equal(t, 0, registry.Count())

	registry.Register("agent1", &mockAgentFull{id: "id1"})
	assert.Equal(t, 1, registry.Count())

	registry.Register("agent2", &mockAgentFull{id: "id2"})
	assert.Equal(t, 2, registry.Count())
}

func TestRegistry_Replace(t *testing.T) {
	registry := NewRegistry()

	agent1 := &mockAgentFull{id: "id1", name: "Agent 1"}
	agent2 := &mockAgentFull{id: "id2", name: "Agent 2"}

	registry.Register("test", agent1)
	assert.Equal(t, "id1", registry.Get("test").ID())

	registry.Register("test", agent2)
	assert.Equal(t, "id2", registry.Get("test").ID())
	assert.Equal(t, 1, registry.Count())
}
