// Copyright (c) Microsoft. All rights reserved.

package chatagent

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"

	"github.com/microsoft/agent-framework-go/agent"
	"github.com/microsoft/agent-framework-go/tool"
)

// mockAgentForTool is a simple agent for testing AsTool.
type mockAgentForTool struct {
	id              string
	name            string
	description     string
	response        string
	runCalled       bool
	streamResp      []agent.ResponseUpdate
	capturedRunOpts []agent.RunOption
}

func (m *mockAgentForTool) ID() string          { return m.id }
func (m *mockAgentForTool) Name() string        { return m.name }
func (m *mockAgentForTool) Description() string { return m.description }
func (m *mockAgentForTool) Metadata() agent.AIAgentMetadata {
	return agent.AIAgentMetadata{}
}

func (m *mockAgentForTool) Run(ctx context.Context, msgs []agent.Message, opts ...agent.RunOption) (*agent.Response, error) {
	m.runCalled = true
	m.capturedRunOpts = opts
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
		{"123Agent", "_123agent"},
		{"1st Agent", "_1st_agent"},
		{"42", "_42"},
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

func TestAsTool_ApprovalMode(t *testing.T) {
	mockAgent := &mockAgentForTool{name: "ApprovalAgent", response: "Done"}

	// Default approval mode (empty string)
	tool1 := AsTool(mockAgent, AsToolOptions{})
	at1 := tool1.(*agentTool)
	if at1.approvalMode != "" {
		t.Errorf("Default ApprovalMode should be empty, got %q", at1.approvalMode)
	}

	// Explicit approval mode
	tool2 := AsTool(mockAgent, AsToolOptions{
		ApprovalMode: tool.ApprovalAlways,
	})
	at2 := tool2.(*agentTool)
	if at2.approvalMode != tool.ApprovalAlways {
		t.Errorf("ApprovalMode = %q, want %q", at2.approvalMode, tool.ApprovalAlways)
	}
}

func TestAsTool_GetApprovalMode(t *testing.T) {
	mockAgent := &mockAgentForTool{name: "TestAgent", response: "Done"}

	agentAsTool := AsTool(mockAgent, AsToolOptions{
		ApprovalMode: tool.ApprovalOnce,
	})

	at := agentAsTool.(*agentTool)
	if at.GetApprovalMode() != tool.ApprovalOnce {
		t.Errorf("GetApprovalMode() = %q, want %q", at.GetApprovalMode(), tool.ApprovalOnce)
	}
}

func TestAsTool_RuntimeContextPropagation(t *testing.T) {
	mockAgent := &mockAgentForTool{
		name:     "ContextAgent",
		response: "Done",
	}

	tool := AsTool(mockAgent, AsToolOptions{
		ForwardRuntimeContext: true,
	})

	// Create context with runtime values
	rtc := agent.NewRuntimeContext().
		With("user_id", "u123").
		With("api_token", "tok_abc")

	ctx := agent.WithRuntimeCtx(context.Background(), rtc)

	args := json.RawMessage(`{"task": "test"}`)
	_, err := tool.Invoke(ctx, args)

	if err != nil {
		t.Errorf("Invoke() error = %v", err)
	}

	if !mockAgent.runCalled {
		t.Error("agent.Run was not called")
	}

	// Verify run options were passed
	if len(mockAgent.capturedRunOpts) == 0 {
		t.Error("Expected run options to be passed with runtime context")
	}
}

func TestAsTool_RuntimeContextDisabled(t *testing.T) {
	mockAgent := &mockAgentForTool{
		name:     "NoContextAgent",
		response: "Done",
	}

	// ForwardRuntimeContext is false by default
	tool := AsTool(mockAgent, AsToolOptions{})

	rtc := agent.NewRuntimeContext().
		With("user_id", "u123")

	ctx := agent.WithRuntimeCtx(context.Background(), rtc)

	args := json.RawMessage(`{"task": "test"}`)
	_, _ = tool.Invoke(ctx, args)

	// No run options should be passed when forwarding is disabled
	if len(mockAgent.capturedRunOpts) != 0 {
		t.Error("Expected no run options when ForwardRuntimeContext is false")
	}
}

func TestAsTool_SessionKeyExclusion(t *testing.T) {
	mockAgent := &mockAgentForTool{
		name:     "ExcludeAgent",
		response: "Done",
	}

	tool := AsTool(mockAgent, AsToolOptions{
		ForwardRuntimeContext: true,
	})

	// Include session keys that should be excluded
	rtc := agent.NewRuntimeContext().
		With("user_id", "u123").
		With("session_id", "should_be_excluded").
		With("conversation_id", "should_be_excluded").
		With("thread_id", "should_be_excluded")

	ctx := agent.WithRuntimeCtx(context.Background(), rtc)

	args := json.RawMessage(`{"task": "test"}`)
	_, _ = tool.Invoke(ctx, args)

	// Agent should be called with filtered context
	if !mockAgent.runCalled {
		t.Error("agent.Run was not called")
	}

	// Verify that run options were passed (context has user_id)
	if len(mockAgent.capturedRunOpts) == 0 {
		t.Error("Expected run options to be passed")
	}
}

func TestAsTool_CustomExcludeKeys(t *testing.T) {
	mockAgent := &mockAgentForTool{
		name:     "CustomExcludeAgent",
		response: "Done",
	}

	tool := AsTool(mockAgent, AsToolOptions{
		ForwardRuntimeContext: true,
		ExcludeKeys:           []string{"secret_key", "internal_id"},
	})

	rtc := agent.NewRuntimeContext().
		With("user_id", "u123").
		With("secret_key", "should_be_excluded").
		With("internal_id", "should_be_excluded")

	ctx := agent.WithRuntimeCtx(context.Background(), rtc)

	args := json.RawMessage(`{"task": "test"}`)
	_, _ = tool.Invoke(ctx, args)

	if !mockAgent.runCalled {
		t.Error("agent.Run was not called")
	}
}

func TestAsTool_EmptyRuntimeContext(t *testing.T) {
	mockAgent := &mockAgentForTool{
		name:     "EmptyContextAgent",
		response: "Done",
	}

	tool := AsTool(mockAgent, AsToolOptions{
		ForwardRuntimeContext: true,
	})

	// Empty runtime context
	ctx := agent.WithRuntimeCtx(context.Background(), agent.NewRuntimeContext())

	args := json.RawMessage(`{"task": "test"}`)
	_, _ = tool.Invoke(ctx, args)

	// No run options should be passed when context is empty
	if len(mockAgent.capturedRunOpts) != 0 {
		t.Error("Expected no run options when runtime context is empty")
	}
}

func TestAsTool_AllSessionKeysExcludedEmptyResult(t *testing.T) {
	mockAgent := &mockAgentForTool{
		name:     "AllExcludedAgent",
		response: "Done",
	}

	tool := AsTool(mockAgent, AsToolOptions{
		ForwardRuntimeContext: true,
	})

	// Only session keys that get excluded
	rtc := agent.NewRuntimeContext().
		With("session_id", "excluded").
		With("conversation_id", "excluded")

	ctx := agent.WithRuntimeCtx(context.Background(), rtc)

	args := json.RawMessage(`{"task": "test"}`)
	_, _ = tool.Invoke(ctx, args)

	// No run options should be passed when all keys are excluded
	if len(mockAgent.capturedRunOpts) != 0 {
		t.Error("Expected no run options when all runtime context keys are excluded")
	}
}
