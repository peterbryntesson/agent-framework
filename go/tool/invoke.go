// Copyright (c) Microsoft. All rights reserved.

package tool

import (
	"context"
	"encoding/json"
	"fmt"
	"runtime/debug"
	"sync"
	"time"
)

// Invoker handles tool invocation with error handling, timeout, and context support.
// It provides safe execution of tools with panic recovery, batch processing,
// and configurable error handling behavior.
//
// Invoker is safe for concurrent use.
type Invoker struct {
	config InvocationConfig
	tools  map[string]Tool
	mu     sync.RWMutex
}

// NewInvoker creates an Invoker with the given tools and configuration.
// Tools are indexed by name for efficient lookup.
func NewInvoker(tools []Tool, config InvocationConfig) *Invoker {
	toolMap := make(map[string]Tool, len(tools))
	for _, t := range tools {
		if t != nil {
			toolMap[t.Name()] = t
		}
	}

	// Merge additional tools from config
	for _, t := range config.AdditionalTools {
		if t != nil {
			toolMap[t.Name()] = t
		}
	}

	return &Invoker{
		config: config,
		tools:  toolMap,
	}
}

// Invoke executes a tool by name with the given arguments.
// Handles panic recovery, timeout, and error wrapping.
//
// If the tool is unknown:
//   - Returns ErrUnknownTool if TerminateOnUnknownCalls is true
//   - Returns an error Result if TerminateOnUnknownCalls is false
//
// If the tool invocation panics:
//   - Returns InvocationPanicError in the error return
//   - Also returns an error Result for the model
func (i *Invoker) Invoke(ctx context.Context, name string, arguments json.RawMessage) (Result, error) {
	if !i.config.Enabled {
		return Result{}, ErrInvocationDisabled
	}

	i.mu.RLock()
	tool, ok := i.tools[name]
	i.mu.RUnlock()

	if !ok {
		availableNames := i.ToolNames()
		if i.config.TerminateOnUnknownCalls {
			return Result{}, NewUnknownToolError(name, availableNames)
		}
		return Result{
			Content: fmt.Sprintf("Unknown tool: %s. Available tools: %v", name, availableNames),
			IsError: true,
		}, nil
	}

	// Skip hosted tools - they are invoked by the provider
	if ht, ok := tool.(HostedTool); ok && ht.IsHosted() {
		return Result{
			Content: fmt.Sprintf("Tool %q is a hosted tool and cannot be invoked locally", name),
			IsError: true,
		}, nil
	}

	// Apply timeout if configured
	if i.config.TimeoutSeconds > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, time.Duration(i.config.TimeoutSeconds)*time.Second)
		defer cancel()
	}

	// Execute with panic recovery
	return i.invokeWithRecovery(ctx, tool, arguments)
}

// invokeWithRecovery executes a tool with panic recovery.
func (i *Invoker) invokeWithRecovery(ctx context.Context, tool Tool, args json.RawMessage) (result Result, err error) {
	defer func() {
		if r := recover(); r != nil {
			panicErr := NewInvocationPanicError(tool.Name(), r, debug.Stack())
			err = panicErr

			if i.config.IncludeDetailedErrors {
				result = Result{
					Content: fmt.Sprintf("Tool panicked: %v", r),
					IsError: true,
					Metadata: map[string]any{
						"panic":      fmt.Sprintf("%v", r),
						"stack":      panicErr.StackTrace(),
						"tool_name":  tool.Name(),
						"error_type": "panic",
					},
				}
			} else {
				result = Result{
					Content: "Tool invocation failed",
					IsError: true,
				}
			}
		}
	}()

	// Check context before invocation
	if err := ctx.Err(); err != nil {
		return Result{
			Content: "Context cancelled before tool invocation",
			IsError: true,
		}, err
	}

	result, err = tool.Invoke(ctx, args)
	if err != nil {
		invErr := NewInvocationError(tool.Name(), err)
		if i.config.IncludeDetailedErrors {
			return Result{
				Content: fmt.Sprintf("Tool error: %v", err),
				IsError: true,
				Metadata: map[string]any{
					"error":      err.Error(),
					"tool_name":  tool.Name(),
					"error_type": "invocation",
				},
			}, invErr
		}
		return Result{
			Content: "Tool invocation failed",
			IsError: true,
		}, invErr
	}

	return result, nil
}

// InvokeBatch executes multiple tool calls, optionally in parallel.
// Results are returned in the same order as the input calls.
//
// If parallel is true and config.ParallelToolCalls is true, calls are
// executed concurrently. Otherwise, calls are executed sequentially.
func (i *Invoker) InvokeBatch(ctx context.Context, calls []ToolCall, parallel bool) []InvocationResult {
	results := make([]InvocationResult, len(calls))

	if len(calls) == 0 {
		return results
	}

	useParallel := parallel && i.config.ParallelToolCalls && len(calls) > 1

	if useParallel {
		i.invokeBatchParallel(ctx, calls, results)
	} else {
		i.invokeBatchSequential(ctx, calls, results)
	}

	return results
}

// invokeBatchSequential executes tool calls one at a time.
func (i *Invoker) invokeBatchSequential(ctx context.Context, calls []ToolCall, results []InvocationResult) {
	for idx, call := range calls {
		result, err := i.Invoke(ctx, call.Name, call.Arguments)
		results[idx] = InvocationResult{
			CallID: call.ID,
			Result: result,
			Error:  err,
		}

		// Check for context cancellation between calls
		if ctx.Err() != nil {
			for j := idx + 1; j < len(calls); j++ {
				results[j] = InvocationResult{
					CallID: calls[j].ID,
					Result: Result{
						Content: "Invocation cancelled",
						IsError: true,
					},
					Error: ctx.Err(),
				}
			}
			break
		}
	}
}

// invokeBatchParallel executes tool calls concurrently.
func (i *Invoker) invokeBatchParallel(ctx context.Context, calls []ToolCall, results []InvocationResult) {
	var wg sync.WaitGroup
	wg.Add(len(calls))

	for idx, call := range calls {
		go func(idx int, call ToolCall) {
			defer wg.Done()
			result, err := i.Invoke(ctx, call.Name, call.Arguments)
			results[idx] = InvocationResult{
				CallID: call.ID,
				Result: result,
				Error:  err,
			}
		}(idx, call)
	}

	wg.Wait()
}

// InvokeToolCalls processes a slice of ToolCall and returns ToolResult.
// This is a convenience method that wraps InvokeBatch and formats
// the results for sending back to the model.
func (i *Invoker) InvokeToolCalls(ctx context.Context, calls []ToolCall) []ToolResult {
	invocationResults := i.InvokeBatch(ctx, calls, true)

	toolResults := make([]ToolResult, len(invocationResults))
	for idx, ir := range invocationResults {
		toolResults[idx] = ToolResult{
			CallID: ir.CallID,
			Result: ir.Result,
		}
	}

	return toolResults
}

// ToolNames returns the names of all registered tools.
func (i *Invoker) ToolNames() []string {
	i.mu.RLock()
	defer i.mu.RUnlock()

	names := make([]string, 0, len(i.tools))
	for name := range i.tools {
		names = append(names, name)
	}
	return names
}

// GetTool returns the tool with the given name, or nil if not found.
func (i *Invoker) GetTool(name string) Tool {
	i.mu.RLock()
	defer i.mu.RUnlock()
	return i.tools[name]
}

// HasTool reports whether a tool with the given name is registered.
func (i *Invoker) HasTool(name string) bool {
	i.mu.RLock()
	defer i.mu.RUnlock()
	_, ok := i.tools[name]
	return ok
}

// Tools returns a slice of all registered tools.
func (i *Invoker) Tools() []Tool {
	i.mu.RLock()
	defer i.mu.RUnlock()

	tools := make([]Tool, 0, len(i.tools))
	for _, tool := range i.tools {
		tools = append(tools, tool)
	}
	return tools
}

// Config returns the current invocation configuration.
func (i *Invoker) Config() InvocationConfig {
	return i.config
}

// AddTool adds a tool to the invoker.
// If a tool with the same name exists, it is replaced.
func (i *Invoker) AddTool(tool Tool) {
	if tool == nil {
		return
	}
	i.mu.Lock()
	defer i.mu.Unlock()
	i.tools[tool.Name()] = tool
}

// RemoveTool removes a tool from the invoker by name.
// Returns true if the tool was found and removed.
func (i *Invoker) RemoveTool(name string) bool {
	i.mu.Lock()
	defer i.mu.Unlock()
	if _, ok := i.tools[name]; ok {
		delete(i.tools, name)
		return true
	}
	return false
}

// InvocationResult captures the outcome of a single tool call in a batch.
type InvocationResult struct {
	// CallID matches the ID from the corresponding ToolCall.
	CallID string

	// Result contains the tool's output.
	Result Result

	// Error contains any error that occurred during invocation.
	// This is non-nil for invocation failures, panics, or context cancellation.
	Error error
}

// IsSuccess reports whether the invocation succeeded without error.
func (r InvocationResult) IsSuccess() bool {
	return r.Error == nil && !r.Result.IsError
}

// IsError reports whether the invocation resulted in an error.
func (r InvocationResult) IsError() bool {
	return r.Error != nil || r.Result.IsError
}
