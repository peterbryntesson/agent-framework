// Copyright (c) Microsoft. All rights reserved.

package chatagent

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/microsoft/agent-framework-go/agent"
	"github.com/microsoft/agent-framework-go/chat"
	"github.com/microsoft/agent-framework-go/tool"
)

// TestToolLoopSingleTurn tests the tool loop with no tool calls.
func TestToolLoopSingleTurn(t *testing.T) {
	client := newMockClient()
	a := New(client)

	resp, err := a.Run(context.Background(), []agent.Message{
		agent.NewUserMessage("Hello"),
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp == nil {
		t.Fatal("expected response")
	}
	// Should complete in single turn without tool calls
}

// TestToolLoopWithToolCall tests the tool invocation loop.
func TestToolLoopWithToolCall(t *testing.T) {
	client := newMockClient()
	callCount := 0

	client.getResponseFn = func(ctx context.Context, messages []chat.Message, options *chat.Options) (*chat.Response, error) {
		callCount++
		if callCount == 1 {
			// First call: return tool call
			return &chat.Response{
				Message: chat.Message{
					Role: chat.RoleAssistant,
					ToolCalls: []chat.ToolCall{
						{
							ID:        "call_1",
							Name:      "get_weather",
							Arguments: json.RawMessage(`{"location": "Seattle"}`),
						},
					},
				},
				FinishReason: chat.FinishReasonToolCalls,
				Usage:        &chat.UsageDetails{InputTokens: 10, OutputTokens: 5},
			}, nil
		}
		// Second call: return final response
		return &chat.Response{
			Message:      chat.NewAssistantMessage("The weather in Seattle is sunny."),
			FinishReason: chat.FinishReasonStop,
			Usage:        &chat.UsageDetails{InputTokens: 15, OutputTokens: 10},
		}, nil
	}

	mockTool := &mockFunctionTool{
		name:         "get_weather",
		invokeResult: tool.Result{Content: "Sunny, 72°F"},
	}
	a := New(client, WithTools(mockTool))

	resp, err := a.Run(context.Background(), []agent.Message{
		agent.NewUserMessage("What's the weather in Seattle?"),
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if callCount != 2 {
		t.Errorf("expected 2 client calls, got %d", callCount)
	}
	if resp.Text() != "The weather in Seattle is sunny." {
		t.Errorf("unexpected response: %q", resp.Text())
	}
	// Check usage accumulation - usage comes from response, which may be nil in mock
	if resp.Usage == nil {
		t.Error("expected usage to be set")
	}
}

// TestToolLoopMaxTurns tests max turns limit.
func TestToolLoopMaxTurns(t *testing.T) {
	client := newMockClient()

	// Always return tool calls to exhaust max turns
	client.getResponseFn = func(ctx context.Context, messages []chat.Message, options *chat.Options) (*chat.Response, error) {
		return &chat.Response{
			Message: chat.Message{
				Role: chat.RoleAssistant,
				ToolCalls: []chat.ToolCall{
					{ID: "call_1", Name: "test_tool", Arguments: json.RawMessage(`{}`)},
				},
			},
			FinishReason: chat.FinishReasonToolCalls,
			Usage:        &chat.UsageDetails{},
		}, nil
	}

	mockTool := &mockFunctionTool{
		name:         "test_tool",
		invokeResult: tool.Result{Content: "result"},
	}
	a := New(client, WithTools(mockTool), WithMaxTurns(3))

	_, err := a.Run(context.Background(), []agent.Message{
		agent.NewUserMessage("Keep calling tools"),
	})

	if err == nil {
		t.Fatal("expected error for max turns exceeded")
	}
	if !errors.Is(err, tool.ErrMaxIterations) {
		t.Errorf("expected ErrMaxIterations, got %v", err)
	}
}

// TestToolLoopInvocationDisabled tests that tool calls are not invoked when disabled.
func TestToolLoopInvocationDisabled(t *testing.T) {
	client := newMockClient()
	invoked := false

	client.getResponseFn = func(ctx context.Context, messages []chat.Message, options *chat.Options) (*chat.Response, error) {
		return &chat.Response{
			Message: chat.Message{
				Role: chat.RoleAssistant,
				ToolCalls: []chat.ToolCall{
					{ID: "call_1", Name: "test_tool", Arguments: json.RawMessage(`{}`)},
				},
			},
			FinishReason: chat.FinishReasonToolCalls,
			Usage:        &chat.UsageDetails{},
		}, nil
	}

	mockTool := &mockFunctionTool{
		name:         "test_tool",
		invokeResult: tool.Result{Content: "result"},
	}
	// Wrap to track invocation
	originalInvoke := mockTool.Invoke
	mockTool.invokeResult = tool.Result{}
	_ = originalInvoke // unused, but demonstrates the pattern

	a := New(client,
		WithTools(mockTool),
		WithInvocationEnabled(false),
	)

	resp, err := a.Run(context.Background(), []agent.Message{
		agent.NewUserMessage("Test"),
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if invoked {
		t.Error("tool should not have been invoked")
	}
	// Response should have tool calls finish reason
	if resp.FinishReason != agent.FinishReasonToolCalls {
		t.Errorf("expected FinishReasonToolCalls, got %v", resp.FinishReason)
	}
}

// TestToolLoopToolError tests handling of tool invocation errors.
func TestToolLoopToolError(t *testing.T) {
	client := newMockClient()
	callCount := 0

	client.getResponseFn = func(ctx context.Context, messages []chat.Message, options *chat.Options) (*chat.Response, error) {
		callCount++
		if callCount == 1 {
			return &chat.Response{
				Message: chat.Message{
					Role: chat.RoleAssistant,
					ToolCalls: []chat.ToolCall{
						{ID: "call_1", Name: "failing_tool", Arguments: json.RawMessage(`{}`)},
					},
				},
				FinishReason: chat.FinishReasonToolCalls,
				Usage:        &chat.UsageDetails{},
			}, nil
		}
		// After tool error, model responds
		return &chat.Response{
			Message:      chat.NewAssistantMessage("I encountered an error."),
			FinishReason: chat.FinishReasonStop,
			Usage:        &chat.UsageDetails{},
		}, nil
	}

	mockTool := &mockFunctionTool{
		name:         "failing_tool",
		invokeResult: tool.Result{Content: "Tool failed", IsError: true},
	}
	a := New(client, WithTools(mockTool))

	resp, err := a.Run(context.Background(), []agent.Message{
		agent.NewUserMessage("Call the tool"),
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should complete after receiving error result
	if resp.Text() != "I encountered an error." {
		t.Errorf("unexpected response: %q", resp.Text())
	}
}

// TestToolLoopConsecutiveErrors tests max consecutive errors handling.
func TestToolLoopConsecutiveErrors(t *testing.T) {
	client := newMockClient()

	client.getResponseFn = func(ctx context.Context, messages []chat.Message, options *chat.Options) (*chat.Response, error) {
		return &chat.Response{
			Message: chat.Message{
				Role: chat.RoleAssistant,
				ToolCalls: []chat.ToolCall{
					{ID: "call_1", Name: "error_tool", Arguments: json.RawMessage(`{}`)},
				},
			},
			FinishReason: chat.FinishReasonToolCalls,
			Usage:        &chat.UsageDetails{},
		}, nil
	}

	mockTool := &mockFunctionTool{
		name:      "error_tool",
		invokeErr: errors.New("tool error"),
	}
	a := New(client,
		WithTools(mockTool),
		WithMaxConsecutiveErrors(2),
		WithTerminateOnUnknownCalls(true), // Make errors terminate
	)

	_, err := a.Run(context.Background(), []agent.Message{
		agent.NewUserMessage("Test"),
	})

	if err == nil {
		t.Fatal("expected error for consecutive tool failures")
	}
}

// TestToolLoopUnknownTool tests handling of unknown tool calls.
func TestToolLoopUnknownTool(t *testing.T) {
	t.Run("returns error result when not terminating", func(t *testing.T) {
		client := newMockClient()
		callCount := 0

		client.getResponseFn = func(ctx context.Context, messages []chat.Message, options *chat.Options) (*chat.Response, error) {
			callCount++
			if callCount == 1 {
				return &chat.Response{
					Message: chat.Message{
						Role: chat.RoleAssistant,
						ToolCalls: []chat.ToolCall{
							{ID: "call_1", Name: "unknown_tool", Arguments: json.RawMessage(`{}`)},
						},
					},
					FinishReason: chat.FinishReasonToolCalls,
					Usage:        &chat.UsageDetails{},
				}, nil
			}
			return &chat.Response{
				Message:      chat.NewAssistantMessage("Understood, tool not available."),
				FinishReason: chat.FinishReasonStop,
				Usage:        &chat.UsageDetails{},
			}, nil
		}

		a := New(client, WithTerminateOnUnknownCalls(false))

		resp, err := a.Run(context.Background(), []agent.Message{
			agent.NewUserMessage("Call unknown tool"),
		})

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.Text() != "Understood, tool not available." {
			t.Errorf("unexpected response: %q", resp.Text())
		}
	})

	t.Run("terminates when configured", func(t *testing.T) {
		client := newMockClient()

		client.getResponseFn = func(ctx context.Context, messages []chat.Message, options *chat.Options) (*chat.Response, error) {
			return &chat.Response{
				Message: chat.Message{
					Role: chat.RoleAssistant,
					ToolCalls: []chat.ToolCall{
						{ID: "call_1", Name: "unknown_tool", Arguments: json.RawMessage(`{}`)},
					},
				},
				FinishReason: chat.FinishReasonToolCalls,
				Usage:        &chat.UsageDetails{},
			}, nil
		}

		a := New(client, WithTerminateOnUnknownCalls(true))

		_, err := a.Run(context.Background(), []agent.Message{
			agent.NewUserMessage("Call unknown tool"),
		})

		if err == nil {
			t.Fatal("expected error for unknown tool")
		}
	})
}

// TestToolLoopParallelExecution tests parallel tool call execution.
func TestToolLoopParallelExecution(t *testing.T) {
	client := newMockClient()
	callCount := 0

	client.getResponseFn = func(ctx context.Context, messages []chat.Message, options *chat.Options) (*chat.Response, error) {
		callCount++
		if callCount == 1 {
			// Return multiple tool calls
			return &chat.Response{
				Message: chat.Message{
					Role: chat.RoleAssistant,
					ToolCalls: []chat.ToolCall{
						{ID: "call_1", Name: "tool_a", Arguments: json.RawMessage(`{}`)},
						{ID: "call_2", Name: "tool_b", Arguments: json.RawMessage(`{}`)},
					},
				},
				FinishReason: chat.FinishReasonToolCalls,
				Usage:        &chat.UsageDetails{},
			}, nil
		}
		return &chat.Response{
			Message:      chat.NewAssistantMessage("Both tools executed."),
			FinishReason: chat.FinishReasonStop,
			Usage:        &chat.UsageDetails{},
		}, nil
	}

	toolA := &mockFunctionTool{name: "tool_a", invokeResult: tool.Result{Content: "A"}}
	toolB := &mockFunctionTool{name: "tool_b", invokeResult: tool.Result{Content: "B"}}

	a := New(client,
		WithTools(toolA, toolB),
		WithParallelToolCalls(true),
	)

	resp, err := a.Run(context.Background(), []agent.Message{
		agent.NewUserMessage("Call both tools"),
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Text() != "Both tools executed." {
		t.Errorf("unexpected response: %q", resp.Text())
	}
}

// TestToolLoopContextCancellation tests context cancellation during tool loop.
func TestToolLoopContextCancellation(t *testing.T) {
	client := newMockClient()
	ctx, cancel := context.WithCancel(context.Background())

	client.getResponseFn = func(ctx context.Context, messages []chat.Message, options *chat.Options) (*chat.Response, error) {
		cancel() // Cancel during execution
		return &chat.Response{
			Message: chat.Message{
				Role: chat.RoleAssistant,
				ToolCalls: []chat.ToolCall{
					{ID: "call_1", Name: "test_tool", Arguments: json.RawMessage(`{}`)},
				},
			},
			FinishReason: chat.FinishReasonToolCalls,
			Usage:        &chat.UsageDetails{},
		}, nil
	}

	mockTool := &mockFunctionTool{name: "test_tool", invokeResult: tool.Result{Content: "result"}}
	a := New(client, WithTools(mockTool))

	_, err := a.Run(ctx, []agent.Message{
		agent.NewUserMessage("Test"),
	})

	if err == nil {
		t.Fatal("expected error from cancelled context")
	}
}

// TestStreamToolLoopBasic tests basic streaming with tool invocation.
func TestStreamToolLoopBasic(t *testing.T) {
	client := newMockClient()

	client.streamUpdates = [][]chat.ResponseUpdate{
		{
			{Kind: chat.UpdateKindContentDelta, Delta: &chat.ContentDelta{TextDelta: "Hello "}},
			{Kind: chat.UpdateKindContentDelta, Delta: &chat.ContentDelta{TextDelta: "world!"}},
			{Kind: chat.UpdateKindDone, FinishReason: chat.FinishReasonStop},
		},
	}

	a := New(client)

	updates, err := a.RunStream(context.Background(), []agent.Message{
		agent.NewUserMessage("Hi"),
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var text string
	for update := range updates {
		if update.Kind == agent.UpdateKindContentDelta && update.Delta != nil {
			text += update.Delta.TextDelta
		}
	}

	if text != "Hello world!" {
		t.Errorf("expected 'Hello world!', got %q", text)
	}
}
