// Copyright (c) Microsoft. All rights reserved.

package tool

import (
	"context"
	"encoding/json"
	"errors"
	"sync/atomic"
	"testing"
	"time"
)

// mockTool implements Tool for testing.
type mockTool struct {
	name        string
	description string
	parameters  json.RawMessage
	invokeFn    func(ctx context.Context, args json.RawMessage) (Result, error)
}

func (m *mockTool) Name() string                { return m.name }
func (m *mockTool) Description() string         { return m.description }
func (m *mockTool) Parameters() json.RawMessage { return m.parameters }
func (m *mockTool) Invoke(ctx context.Context, args json.RawMessage) (Result, error) {
	if m.invokeFn != nil {
		return m.invokeFn(ctx, args)
	}
	return NewResult("mock result"), nil
}

// mockHostedTool implements HostedTool for testing.
type mockHostedTool struct {
	mockTool
}

func (m *mockHostedTool) IsHosted() bool { return true }
func (m *mockHostedTool) ProviderConfig() map[string]interface{} {
	return map[string]interface{}{"type": "mock"}
}

func TestNewInvoker(t *testing.T) {
	t.Run("creates invoker with tools", func(t *testing.T) {
		tools := []Tool{
			&mockTool{name: "tool1"},
			&mockTool{name: "tool2"},
		}
		config := DefaultInvocationConfig()

		invoker := NewInvoker(tools, config)

		if invoker == nil {
			t.Fatal("expected non-nil invoker")
		}
		if len(invoker.ToolNames()) != 2 {
			t.Errorf("expected 2 tools, got %d", len(invoker.ToolNames()))
		}
		if !invoker.HasTool("tool1") {
			t.Error("expected tool1 to be registered")
		}
		if !invoker.HasTool("tool2") {
			t.Error("expected tool2 to be registered")
		}
	})

	t.Run("merges additional tools from config", func(t *testing.T) {
		tools := []Tool{
			&mockTool{name: "tool1"},
		}
		config := DefaultInvocationConfig()
		config.AdditionalTools = []Tool{
			&mockTool{name: "additional"},
		}

		invoker := NewInvoker(tools, config)

		if !invoker.HasTool("additional") {
			t.Error("expected additional tool to be registered")
		}
		if len(invoker.ToolNames()) != 2 {
			t.Errorf("expected 2 tools, got %d", len(invoker.ToolNames()))
		}
	})

	t.Run("handles nil tools in slice", func(t *testing.T) {
		tools := []Tool{
			&mockTool{name: "tool1"},
			nil,
			&mockTool{name: "tool2"},
		}
		config := DefaultInvocationConfig()

		invoker := NewInvoker(tools, config)

		if len(invoker.ToolNames()) != 2 {
			t.Errorf("expected 2 tools, got %d", len(invoker.ToolNames()))
		}
	})
}

func TestInvoker_Invoke(t *testing.T) {
	t.Run("invokes tool successfully", func(t *testing.T) {
		tool := &mockTool{
			name: "test_tool",
			invokeFn: func(_ context.Context, args json.RawMessage) (Result, error) {
				return NewResult("success"), nil
			},
		}
		invoker := NewInvoker([]Tool{tool}, DefaultInvocationConfig())

		result, err := invoker.Invoke(context.Background(), "test_tool", nil)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.Content != "success" {
			t.Errorf("expected 'success', got %q", result.Content)
		}
		if result.IsError {
			t.Error("expected IsError to be false")
		}
	})

	t.Run("returns error result for unknown tool when not terminating", func(t *testing.T) {
		config := DefaultInvocationConfig()
		config.TerminateOnUnknownCalls = false
		invoker := NewInvoker([]Tool{}, config)

		result, err := invoker.Invoke(context.Background(), "unknown", nil)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !result.IsError {
			t.Error("expected IsError to be true")
		}
	})

	t.Run("returns ErrUnknownTool when terminating on unknown", func(t *testing.T) {
		config := DefaultInvocationConfig()
		config.TerminateOnUnknownCalls = true
		invoker := NewInvoker([]Tool{}, config)

		_, err := invoker.Invoke(context.Background(), "unknown", nil)

		if err == nil {
			t.Fatal("expected error")
		}
		if !errors.Is(err, ErrUnknownTool) {
			t.Errorf("expected ErrUnknownTool, got %v", err)
		}
	})

	t.Run("returns error when invocation disabled", func(t *testing.T) {
		config := DisabledInvocationConfig()
		invoker := NewInvoker([]Tool{&mockTool{name: "test"}}, config)

		_, err := invoker.Invoke(context.Background(), "test", nil)

		if err == nil {
			t.Fatal("expected error")
		}
		if !errors.Is(err, ErrInvocationDisabled) {
			t.Errorf("expected ErrInvocationDisabled, got %v", err)
		}
	})

	t.Run("skips hosted tools", func(t *testing.T) {
		hostedTool := &mockHostedTool{
			mockTool: mockTool{name: "hosted"},
		}
		invoker := NewInvoker([]Tool{hostedTool}, DefaultInvocationConfig())

		result, err := invoker.Invoke(context.Background(), "hosted", nil)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !result.IsError {
			t.Error("expected IsError for hosted tool")
		}
	})

	t.Run("recovers from panic", func(t *testing.T) {
		tool := &mockTool{
			name: "panicker",
			invokeFn: func(_ context.Context, _ json.RawMessage) (Result, error) {
				panic("test panic")
			},
		}
		invoker := NewInvoker([]Tool{tool}, DefaultInvocationConfig())

		result, err := invoker.Invoke(context.Background(), "panicker", nil)

		if err == nil {
			t.Fatal("expected error from panic")
		}
		var panicErr *InvocationPanicError
		if !errors.As(err, &panicErr) {
			t.Errorf("expected InvocationPanicError, got %T", err)
		}
		if result.Content != "Tool invocation failed" {
			t.Errorf("expected generic error message, got %q", result.Content)
		}
	})

	t.Run("includes detailed error on panic when configured", func(t *testing.T) {
		tool := &mockTool{
			name: "panicker",
			invokeFn: func(_ context.Context, _ json.RawMessage) (Result, error) {
				panic("detailed panic")
			},
		}
		config := DefaultInvocationConfig()
		config.IncludeDetailedErrors = true
		invoker := NewInvoker([]Tool{tool}, config)

		result, _ := invoker.Invoke(context.Background(), "panicker", nil)

		if result.Metadata == nil {
			t.Fatal("expected metadata with detailed errors")
		}
		if result.Metadata["error_type"] != "panic" {
			t.Errorf("expected error_type 'panic', got %v", result.Metadata["error_type"])
		}
	})

	t.Run("wraps tool invocation errors", func(t *testing.T) {
		testErr := errors.New("tool failed")
		tool := &mockTool{
			name: "failing",
			invokeFn: func(_ context.Context, _ json.RawMessage) (Result, error) {
				return Result{}, testErr
			},
		}
		invoker := NewInvoker([]Tool{tool}, DefaultInvocationConfig())

		result, err := invoker.Invoke(context.Background(), "failing", nil)

		if err == nil {
			t.Fatal("expected error")
		}
		var invErr *InvocationError
		if !errors.As(err, &invErr) {
			t.Errorf("expected InvocationError, got %T", err)
		}
		if result.Content != "Tool invocation failed" {
			t.Errorf("expected generic error message, got %q", result.Content)
		}
	})

	t.Run("respects context cancellation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		tool := &mockTool{name: "test"}
		invoker := NewInvoker([]Tool{tool}, DefaultInvocationConfig())

		result, err := invoker.Invoke(ctx, "test", nil)

		if err == nil {
			t.Fatal("expected error from cancelled context")
		}
		if result.Content != "Context cancelled before tool invocation" {
			t.Errorf("unexpected content: %q", result.Content)
		}
	})
}

func TestInvoker_InvokeBatch(t *testing.T) {
	t.Run("executes multiple calls sequentially", func(t *testing.T) {
		var callOrder []string
		tool := &mockTool{
			name: "test",
			invokeFn: func(_ context.Context, args json.RawMessage) (Result, error) {
				var id string
				if err := json.Unmarshal(args, &id); err == nil {
					callOrder = append(callOrder, id)
				}
				return NewResult("ok"), nil
			},
		}
		config := DefaultInvocationConfig()
		config.ParallelToolCalls = false
		invoker := NewInvoker([]Tool{tool}, config)

		calls := []ToolCall{
			{ID: "1", Name: "test", Arguments: json.RawMessage(`"a"`)},
			{ID: "2", Name: "test", Arguments: json.RawMessage(`"b"`)},
			{ID: "3", Name: "test", Arguments: json.RawMessage(`"c"`)},
		}

		results := invoker.InvokeBatch(context.Background(), calls, false)

		if len(results) != 3 {
			t.Fatalf("expected 3 results, got %d", len(results))
		}
		for i, r := range results {
			if r.CallID != calls[i].ID {
				t.Errorf("result %d: expected CallID %q, got %q", i, calls[i].ID, r.CallID)
			}
		}
	})

	t.Run("executes in parallel when configured", func(t *testing.T) {
		var concurrent int32
		var maxConcurrent int32

		tool := &mockTool{
			name: "slow",
			invokeFn: func(_ context.Context, _ json.RawMessage) (Result, error) {
				current := atomic.AddInt32(&concurrent, 1)
				defer atomic.AddInt32(&concurrent, -1)

				// Track max concurrent calls
				for {
					max := atomic.LoadInt32(&maxConcurrent)
					if current <= max || atomic.CompareAndSwapInt32(&maxConcurrent, max, current) {
						break
					}
				}

				time.Sleep(10 * time.Millisecond)
				return NewResult("ok"), nil
			},
		}
		config := DefaultInvocationConfig()
		config.ParallelToolCalls = true
		invoker := NewInvoker([]Tool{tool}, config)

		calls := []ToolCall{
			{ID: "1", Name: "slow"},
			{ID: "2", Name: "slow"},
			{ID: "3", Name: "slow"},
		}

		results := invoker.InvokeBatch(context.Background(), calls, true)

		if len(results) != 3 {
			t.Fatalf("expected 3 results, got %d", len(results))
		}
		if maxConcurrent < 2 {
			t.Error("expected parallel execution, but calls were sequential")
		}
	})

	t.Run("handles empty call list", func(t *testing.T) {
		invoker := NewInvoker([]Tool{}, DefaultInvocationConfig())

		results := invoker.InvokeBatch(context.Background(), []ToolCall{}, true)

		if len(results) != 0 {
			t.Errorf("expected empty results, got %d", len(results))
		}
	})

	t.Run("cancels remaining calls on context cancellation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		callCount := 0

		tool := &mockTool{
			name: "test",
			invokeFn: func(_ context.Context, _ json.RawMessage) (Result, error) {
				callCount++
				if callCount == 1 {
					cancel() // Cancel after first call
				}
				return NewResult("ok"), nil
			},
		}
		config := DefaultInvocationConfig()
		config.ParallelToolCalls = false
		invoker := NewInvoker([]Tool{tool}, config)

		calls := []ToolCall{
			{ID: "1", Name: "test"},
			{ID: "2", Name: "test"},
			{ID: "3", Name: "test"},
		}

		results := invoker.InvokeBatch(ctx, calls, false)

		// First call succeeds, rest should be cancelled
		if !results[0].IsSuccess() {
			t.Error("expected first call to succeed")
		}
		for i := 1; i < len(results); i++ {
			if results[i].Error == nil {
				t.Errorf("expected call %d to have error", i+1)
			}
		}
	})
}

func TestInvoker_ToolManagement(t *testing.T) {
	t.Run("GetTool returns registered tool", func(t *testing.T) {
		tool := &mockTool{name: "test"}
		invoker := NewInvoker([]Tool{tool}, DefaultInvocationConfig())

		got := invoker.GetTool("test")

		if got != tool {
			t.Error("expected to get the registered tool")
		}
	})

	t.Run("GetTool returns nil for unknown tool", func(t *testing.T) {
		invoker := NewInvoker([]Tool{}, DefaultInvocationConfig())

		got := invoker.GetTool("unknown")

		if got != nil {
			t.Error("expected nil for unknown tool")
		}
	})

	t.Run("AddTool adds new tool", func(t *testing.T) {
		invoker := NewInvoker([]Tool{}, DefaultInvocationConfig())
		tool := &mockTool{name: "new"}

		invoker.AddTool(tool)

		if !invoker.HasTool("new") {
			t.Error("expected new tool to be added")
		}
	})

	t.Run("AddTool replaces existing tool", func(t *testing.T) {
		original := &mockTool{name: "test", description: "original"}
		replacement := &mockTool{name: "test", description: "replacement"}
		invoker := NewInvoker([]Tool{original}, DefaultInvocationConfig())

		invoker.AddTool(replacement)

		got := invoker.GetTool("test")
		if got.Description() != "replacement" {
			t.Error("expected tool to be replaced")
		}
	})

	t.Run("AddTool ignores nil", func(t *testing.T) {
		invoker := NewInvoker([]Tool{}, DefaultInvocationConfig())

		invoker.AddTool(nil)

		if len(invoker.ToolNames()) != 0 {
			t.Error("expected no tools after adding nil")
		}
	})

	t.Run("RemoveTool removes existing tool", func(t *testing.T) {
		tool := &mockTool{name: "test"}
		invoker := NewInvoker([]Tool{tool}, DefaultInvocationConfig())

		removed := invoker.RemoveTool("test")

		if !removed {
			t.Error("expected RemoveTool to return true")
		}
		if invoker.HasTool("test") {
			t.Error("expected tool to be removed")
		}
	})

	t.Run("RemoveTool returns false for unknown tool", func(t *testing.T) {
		invoker := NewInvoker([]Tool{}, DefaultInvocationConfig())

		removed := invoker.RemoveTool("unknown")

		if removed {
			t.Error("expected RemoveTool to return false")
		}
	})

	t.Run("Tools returns all registered tools", func(t *testing.T) {
		tools := []Tool{
			&mockTool{name: "tool1"},
			&mockTool{name: "tool2"},
		}
		invoker := NewInvoker(tools, DefaultInvocationConfig())

		got := invoker.Tools()

		if len(got) != 2 {
			t.Errorf("expected 2 tools, got %d", len(got))
		}
	})
}

func TestInvoker_InvokeToolCalls(t *testing.T) {
	t.Run("converts InvocationResult to ToolResult", func(t *testing.T) {
		tool := &mockTool{
			name: "test",
			invokeFn: func(_ context.Context, _ json.RawMessage) (Result, error) {
				return NewResult("result"), nil
			},
		}
		invoker := NewInvoker([]Tool{tool}, DefaultInvocationConfig())

		calls := []ToolCall{
			{ID: "call-1", Name: "test"},
			{ID: "call-2", Name: "test"},
		}

		results := invoker.InvokeToolCalls(context.Background(), calls)

		if len(results) != 2 {
			t.Fatalf("expected 2 results, got %d", len(results))
		}
		if results[0].CallID != "call-1" {
			t.Errorf("expected CallID 'call-1', got %q", results[0].CallID)
		}
		if results[0].Result.Content != "result" {
			t.Errorf("expected content 'result', got %q", results[0].Result.Content)
		}
	})
}

func TestInvocationResult(t *testing.T) {
	t.Run("IsSuccess returns true for successful result", func(t *testing.T) {
		r := InvocationResult{
			Result: NewResult("ok"),
			Error:  nil,
		}

		if !r.IsSuccess() {
			t.Error("expected IsSuccess to be true")
		}
	})

	t.Run("IsSuccess returns false when error present", func(t *testing.T) {
		r := InvocationResult{
			Result: NewResult("ok"),
			Error:  errors.New("error"),
		}

		if r.IsSuccess() {
			t.Error("expected IsSuccess to be false")
		}
	})

	t.Run("IsSuccess returns false when result is error", func(t *testing.T) {
		r := InvocationResult{
			Result: NewErrorResult("failed"),
			Error:  nil,
		}

		if r.IsSuccess() {
			t.Error("expected IsSuccess to be false")
		}
	})

	t.Run("IsError returns true when error present", func(t *testing.T) {
		r := InvocationResult{
			Error: errors.New("error"),
		}

		if !r.IsError() {
			t.Error("expected IsError to be true")
		}
	})
}
