// Copyright (c) Microsoft. All rights reserved.

package chatagent

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"

	"github.com/microsoft/agent-framework-go/agent"
)

// mockAgentForTool is a simple agent for testing AsTool.
type mockAgentForTool struct {
	id          string
	name        string
	description string
	response    string
	runCalled   bool
	streamResp  []agent.ResponseUpdate
}

func (m *mockAgentForTool) ID() string          { return m.id }
func (m *mockAgentForTool) Name() string        { return m.name }
func (m *mockAgentForTool) Description() string { return m.description }
func (m *mockAgentForTool) Metadata() agent.AIAgentMetadata {
	return agent.AIAgentMetadata{}
}

func (m *mockAgentForTool) Run(ctx context.Context, msgs []agent.Message, opts ...agent.RunOption) (*agent.Response, error) {
	m.runCalled = true
	return &agent.Response{
		Messages: []agent.Message{
			agent.NewAssistantMessage(m.response),
		},
	}, nil
}

func (m *mockAgentForTool) RunStream(ctx context.Context, msgs []agent.Message, opts ...agent.RunOption) (<-chan agent.ResponseUpdate, error) {
	ch := make(chan agent.ResponseUpdate, len(m.streamResp)+1)
	for _, u := range m.streamResp {
		ch <- u
	}
	ch <- agent.ResponseUpdate{Kind: agent.UpdateKindDone}
	close(ch)
	return ch, nil
}

func (m *mockAgentForTool) NewSession(ctx context.Context) (agent.Session, error) {
	return nil, nil
}

func (m *mockAgentForTool) RestoreSession(ctx context.Context, data json.RawMessage) (agent.Session, error) {
	return nil, nil
}

func (m *mockAgentForTool) GetService(t reflect.Type) interface{} {
	return nil
}

func TestAsTool_NonStreaming(t *testing.T) {
	mockAgent := &mockAgentForTool{
		id:          "test-id",
		name:        "Research Agent",
		description: "Performs research on topics",
		response:    "Research results for the topic",
	}

	tool := AsTool(mockAgent, AsToolOptions{})

	if tool.Name() != "research_agent" {
		t.Errorf("Name() = %q, want %q", tool.Name(), "research_agent")
	}
	if tool.Description() != "Performs research on topics" {
		t.Errorf("Description() = %q, want %q", tool.Description(), "Performs research on topics")
	}

	// Test invocation
	args := json.RawMessage(`{"task": "research AI"}`)
	result, err := tool.Invoke(context.Background(), args)

	if err != nil {
		t.Errorf("Invoke() error = %v", err)
	}
	if !mockAgent.runCalled {
		t.Error("agent.Run was not called")
	}
	if result.Content != "Research results for the topic" {
		t.Errorf("result.Content = %q, want %q", result.Content, "Research results for the topic")
	}
}

func TestAsTool_WithCustomOptions(t *testing.T) {
	mockAgent := &mockAgentForTool{
		id:       "test-id",
		name:     "Helper",
		response: "Done",
	}

	tool := AsTool(mockAgent, AsToolOptions{
		Name:           "custom_tool",
		Description:    "Custom description",
		ArgName:        "query",
		ArgDescription: "The query to process",
	})

	if tool.Name() != "custom_tool" {
		t.Errorf("Name() = %q, want %q", tool.Name(), "custom_tool")
	}
	if tool.Description() != "Custom description" {
		t.Errorf("Description() = %q, want %q", tool.Description(), "Custom description")
	}

	// Check parameters schema contains custom arg name
	params := tool.Parameters()
	var schema map[string]interface{}
	if err := json.Unmarshal(params, &schema); err != nil {
		t.Fatalf("Failed to unmarshal parameters: %v", err)
	}

	props := schema["properties"].(map[string]interface{})
	if _, ok := props["query"]; !ok {
		t.Error("parameters schema should contain 'query' property")
	}
}

func TestAsTool_Streaming(t *testing.T) {
	mockAgent := &mockAgentForTool{
		id:   "test-id",
		name: "Stream Agent",
		streamResp: []agent.ResponseUpdate{
			{Kind: agent.UpdateKindContentDelta, Delta: &agent.ContentDelta{TextDelta: "Hello "}},
			{Kind: agent.UpdateKindContentDelta, Delta: &agent.ContentDelta{TextDelta: "World"}},
		},
	}

	var receivedUpdates []agent.ResponseUpdate
	tool := AsTool(mockAgent, AsToolOptions{
		StreamCallback: func(update agent.ResponseUpdate) {
			receivedUpdates = append(receivedUpdates, update)
		},
	})

	args := json.RawMessage(`{"task": "stream test"}`)
	result, err := tool.Invoke(context.Background(), args)

	if err != nil {
		t.Errorf("Invoke() error = %v", err)
	}
	if result.Content != "Hello World" {
		t.Errorf("result.Content = %q, want %q", result.Content, "Hello World")
	}
	if len(receivedUpdates) < 2 {
		t.Errorf("received %d updates, want at least 2", len(receivedUpdates))
	}
}

func TestAsTool_NameSanitization(t *testing.T) {
	tests := []struct {
		agentName string
		wantName  string
	}{
		{"Research Agent", "research_agent"},
		{"code-helper", "code_helper"},
		{"My Super Agent!", "my_super_agent"},
		{"", "agent"},
		{"___", "agent"},
		{"Agent123", "agent123"},
	}

	for _, tt := range tests {
		t.Run(tt.agentName, func(t *testing.T) {
			mockAgent := &mockAgentForTool{name: tt.agentName}
			tool := AsTool(mockAgent, AsToolOptions{})

			if tool.Name() != tt.wantName {
				t.Errorf("Name() = %q, want %q", tool.Name(), tt.wantName)
			}
		})
	}
}

func TestAsTool_DefaultDescription(t *testing.T) {
	mockAgent := &mockAgentForTool{
		name:        "Helper",
		description: "", // Empty description
	}

	tool := AsTool(mockAgent, AsToolOptions{})

	expectedDesc := "Delegate task to helper agent"
	if tool.Description() != expectedDesc {
		t.Errorf("Description() = %q, want %q", tool.Description(), expectedDesc)
	}
}

func TestSanitizeAgentName(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"Simple", "simple"},
		{"With Spaces", "with_spaces"},
		{"special!@#chars", "special_chars"},
		{"multiple___underscores", "multiple_underscores"},
		{"_leading_trailing_", "leading_trailing"},
		{"", "agent"},
		{"MixedCase123", "mixedcase123"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := sanitizeAgentName(tt.input)
			if got != tt.want {
				t.Errorf("sanitizeAgentName(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
