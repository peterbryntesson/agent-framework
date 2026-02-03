// Copyright (c) Microsoft. All rights reserved.

package chatagent

import (
	"context"
	"fmt"

	"github.com/microsoft/agent-framework-go/agent"
	"github.com/microsoft/agent-framework-go/chat"
	"github.com/microsoft/agent-framework-go/tool"
)

// runWithToolLoop executes the agent with automatic tool invocation.
// It loops until the model generates a response without tool calls,
// or until maxTurns is reached.
func (a *Agent) runWithToolLoop(ctx context.Context, messages []chat.Message, options *chat.Options, cfg *agent.RunConfig) (*agent.Response, error) {
	allTools := a.getAllTools(cfg)
	invoker := tool.NewInvoker(allTools, a.invocationConfig)

	currentMessages := messages
	var allResponseMessages []chat.Message
	var totalUsage chat.UsageDetails
	var consecutiveErrors int

	for turn := 0; turn < a.maxTurns; turn++ {
		// Check context cancellation
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		// Get response from chat client
		resp, err := a.client.GetResponse(ctx, currentMessages, options)
		if err != nil {
			return nil, fmt.Errorf("chat request failed: %w", err)
		}

		// Accumulate usage
		totalUsage.InputTokens += resp.Usage.InputTokens
		totalUsage.OutputTokens += resp.Usage.OutputTokens
		totalUsage.TotalTokens += resp.Usage.TotalTokens

		// Store response message
		allResponseMessages = append(allResponseMessages, resp.Message)

		// Check if we need to invoke tools
		if len(resp.Message.ToolCalls) == 0 || resp.FinishReason != chat.FinishReasonToolCalls {
			// No tool calls or explicitly stopped - we're done
			return a.buildResponse(allResponseMessages, totalUsage, resp.FinishReason), nil
		}

		// Invoke tools if enabled
		if !a.invocationConfig.Enabled {
			// Return with pending tool calls
			return a.buildResponse(allResponseMessages, totalUsage, chat.FinishReasonToolCalls), nil
		}

		// Execute tool calls
		toolResults, err := a.invokeToolCalls(ctx, invoker, resp.Message.ToolCalls)
		if err != nil {
			consecutiveErrors++
			if consecutiveErrors >= a.invocationConfig.MaxConsecutiveErrors {
				return nil, fmt.Errorf("max consecutive errors (%d) exceeded: %w",
					a.invocationConfig.MaxConsecutiveErrors, err)
			}
		} else {
			consecutiveErrors = 0
		}

		// Append assistant message with tool calls to conversation
		currentMessages = append(currentMessages, resp.Message)

		// Append tool results as tool messages
		for _, result := range toolResults {
			toolMsg := chat.NewToolMessage(result.CallID, result.Content)
			currentMessages = append(currentMessages, toolMsg)

			// Include intermediate steps if configured
			if a.invocationConfig.ReturnIntermediateSteps {
				allResponseMessages = append(allResponseMessages, toolMsg)
			}
		}
	}

	// Max turns reached
	return nil, tool.ErrMaxIterations
}

// runStreamWithToolLoop executes the agent with streaming and tool invocation.
func (a *Agent) runStreamWithToolLoop(ctx context.Context, messages []chat.Message, options *chat.Options, cfg *agent.RunConfig) (<-chan agent.ResponseUpdate, error) {
	updates := make(chan agent.ResponseUpdate, 32)

	go func() {
		defer close(updates)
		a.processStreamWithToolLoop(ctx, messages, options, cfg, updates)
	}()

	return updates, nil
}

// processStreamWithToolLoop handles the streaming loop with tool invocation.
func (a *Agent) processStreamWithToolLoop(ctx context.Context, messages []chat.Message, options *chat.Options, cfg *agent.RunConfig, updates chan<- agent.ResponseUpdate) {
	allTools := a.getAllTools(cfg)
	invoker := tool.NewInvoker(allTools, a.invocationConfig)

	currentMessages := messages
	var totalUsage chat.UsageDetails
	var consecutiveErrors int

	for turn := 0; turn < a.maxTurns; turn++ {
		// Check context cancellation
		if ctx.Err() != nil {
			updates <- agent.ResponseUpdate{
				Kind:  agent.UpdateKindError,
				Error: ctx.Err(),
			}
			return
		}

		// Get streaming response
		streamUpdates, err := a.client.GetStreamingResponse(ctx, currentMessages, options)
		if err != nil {
			updates <- agent.ResponseUpdate{
				Kind:  agent.UpdateKindError,
				Error: fmt.Errorf("chat stream failed: %w", err),
			}
			return
		}

		// Collect the stream and forward updates
		responseMsg, usage, finishReason, err := a.collectStreamUpdates(ctx, streamUpdates, updates)
		if err != nil {
			updates <- agent.ResponseUpdate{
				Kind:  agent.UpdateKindError,
				Error: err,
			}
			return
		}

		// Accumulate usage
		if usage != nil {
			totalUsage.InputTokens += usage.InputTokens
			totalUsage.OutputTokens += usage.OutputTokens
			totalUsage.TotalTokens += usage.TotalTokens
		}

		// Check if we need to invoke tools
		if len(responseMsg.ToolCalls) == 0 || finishReason != chat.FinishReasonToolCalls {
			// Send final done update
			updates <- agent.ResponseUpdate{
				Kind:         agent.UpdateKindDone,
				FinishReason: convertFinishReason(finishReason),
				Usage: &agent.UsageDetails{
					InputTokens:  totalUsage.InputTokens,
					OutputTokens: totalUsage.OutputTokens,
					TotalTokens:  totalUsage.TotalTokens,
				},
			}
			return
		}

		// Invoke tools if enabled
		if !a.invocationConfig.Enabled {
			updates <- agent.ResponseUpdate{
				Kind:         agent.UpdateKindDone,
				FinishReason: agent.FinishReasonToolCalls,
			}
			return
		}

		// Execute tool calls
		toolResults, err := a.invokeToolCalls(ctx, invoker, responseMsg.ToolCalls)
		if err != nil {
			consecutiveErrors++
			if consecutiveErrors >= a.invocationConfig.MaxConsecutiveErrors {
				updates <- agent.ResponseUpdate{
					Kind:  agent.UpdateKindError,
					Error: fmt.Errorf("max consecutive errors exceeded: %w", err),
				}
				return
			}
		} else {
			consecutiveErrors = 0
		}

		// Append to conversation for next iteration
		currentMessages = append(currentMessages, responseMsg)

		for _, result := range toolResults {
			toolMsg := chat.NewToolMessage(result.CallID, result.Content)
			currentMessages = append(currentMessages, toolMsg)

			// Send tool result update
			updates <- agent.ResponseUpdate{
				Kind: agent.UpdateKindToolResult,
				Delta: &agent.ContentDelta{
					ToolCallID: result.CallID,
					Name:       result.Name,
				},
			}
		}
	}

	// Max turns reached
	updates <- agent.ResponseUpdate{
		Kind:  agent.UpdateKindError,
		Error: tool.ErrMaxIterations,
	}
}

// collectStreamUpdates collects updates from a stream and forwards them.
func (a *Agent) collectStreamUpdates(ctx context.Context, streamUpdates <-chan chat.ResponseUpdate, outUpdates chan<- agent.ResponseUpdate) (chat.Message, *chat.UsageDetails, chat.FinishReason, error) {
	var message chat.Message
	var usage *chat.UsageDetails
	var finishReason chat.FinishReason
	var textContent string
	toolCallsMap := make(map[int]*chat.ToolCall)

	for update := range streamUpdates {
		// Check context
		if ctx.Err() != nil {
			return message, usage, finishReason, ctx.Err()
		}

		switch update.Kind {
		case chat.UpdateKindContentDelta:
			if update.Delta != nil && update.Delta.TextDelta != "" {
				textContent += update.Delta.TextDelta
				outUpdates <- agent.ResponseUpdate{
					Kind: agent.UpdateKindContentDelta,
					Delta: &agent.ContentDelta{
						TextDelta: update.Delta.TextDelta,
						Role:      string(update.Delta.Role),
					},
				}
			}

		case chat.UpdateKindToolCall:
			if update.Delta != nil {
				// Use tool call ID as key for accumulation
				toolCallID := update.Delta.ToolCallID
				if toolCallID == "" {
					// Use name as fallback key if ID not yet available
					toolCallID = update.Delta.Name
				}

				// Find or create tool call entry using hash of ID
				idx := len(toolCallsMap) // Simple index for new entries
				for i, tc := range toolCallsMap {
					if tc.ID == update.Delta.ToolCallID || (tc.ID == "" && tc.Name == update.Delta.Name) {
						idx = i
						break
					}
				}
				if _, ok := toolCallsMap[idx]; !ok {
					toolCallsMap[idx] = &chat.ToolCall{}
				}
				tc := toolCallsMap[idx]
				if update.Delta.ToolCallID != "" {
					tc.ID = update.Delta.ToolCallID
				}
				if update.Delta.Name != "" {
					tc.Name = update.Delta.Name
				}
				if update.Delta.ArgsDelta != "" {
					tc.Arguments = append(tc.Arguments, []byte(update.Delta.ArgsDelta)...)
				}

				// Forward tool call update
				outUpdates <- agent.ResponseUpdate{
					Kind: agent.UpdateKindToolCall,
					Delta: &agent.ContentDelta{
						ToolCallID: update.Delta.ToolCallID,
						Name:       update.Delta.Name,
						ArgsDelta:  update.Delta.ArgsDelta,
					},
				}
			}

		case chat.UpdateKindUsage:
			if update.Usage != nil {
				usage = update.Usage
				outUpdates <- agent.ResponseUpdate{
					Kind: agent.UpdateKindUsage,
					Usage: &agent.UsageDetails{
						InputTokens:  update.Usage.InputTokens,
						OutputTokens: update.Usage.OutputTokens,
						TotalTokens:  update.Usage.TotalTokens,
					},
				}
			}

		case chat.UpdateKindMessageComplete:
			if update.Message != nil {
				message = *update.Message
			}

		case chat.UpdateKindError:
			return message, usage, finishReason, update.Error

		case chat.UpdateKindDone:
			finishReason = update.FinishReason
		}
	}

	// Build message if not complete
	if len(message.Contents) == 0 && textContent != "" {
		message.Role = chat.RoleAssistant
		message.Contents = []chat.Content{chat.NewTextContent(textContent)}
	}

	// Add collected tool calls
	if len(toolCallsMap) > 0 {
		message.ToolCalls = make([]chat.ToolCall, 0, len(toolCallsMap))
		for _, tc := range toolCallsMap {
			if tc.ID != "" || tc.Name != "" {
				message.ToolCalls = append(message.ToolCalls, *tc)
			}
		}
	}

	return message, usage, finishReason, nil
}

// invokeToolCalls executes multiple tool calls.
func (a *Agent) invokeToolCalls(ctx context.Context, invoker *tool.Invoker, toolCalls []chat.ToolCall) ([]toolCallResult, error) {
	results := make([]toolCallResult, 0, len(toolCalls))
	var lastErr error

	if a.invocationConfig.ParallelToolCalls && len(toolCalls) > 1 {
		// Parallel execution
		return a.invokeToolCallsParallel(ctx, invoker, toolCalls)
	}

	// Sequential execution
	for _, tc := range toolCalls {
		result, err := invoker.Invoke(ctx, tc.Name, tc.Arguments)
		if err != nil {
			lastErr = err
			if a.invocationConfig.TerminateOnUnknownCalls {
				return nil, err
			}
		}

		content := result.Content
		if result.IsError && a.invocationConfig.IncludeDetailedErrors {
			content = fmt.Sprintf("Error: %s", content)
		}

		results = append(results, toolCallResult{
			CallID:  tc.ID,
			Name:    tc.Name,
			Content: content,
			IsError: result.IsError,
		})
	}

	return results, lastErr
}

// invokeToolCallsParallel executes tool calls concurrently.
func (a *Agent) invokeToolCallsParallel(ctx context.Context, invoker *tool.Invoker, toolCalls []chat.ToolCall) ([]toolCallResult, error) {
	calls := make([]tool.ToolCall, len(toolCalls))
	for i, tc := range toolCalls {
		calls[i] = tool.ToolCall{
			ID:        tc.ID,
			Name:      tc.Name,
			Arguments: tc.Arguments,
		}
	}

	invokerResults := invoker.InvokeBatch(ctx, calls, true)

	results := make([]toolCallResult, len(invokerResults))
	var lastErr error

	for i, ir := range invokerResults {
		if ir.Error != nil {
			lastErr = ir.Error
		}

		content := ir.Result.Content
		if ir.Result.IsError && a.invocationConfig.IncludeDetailedErrors {
			content = fmt.Sprintf("Error: %s", content)
		}

		results[i] = toolCallResult{
			CallID:  calls[i].ID,
			Name:    calls[i].Name,
			Content: content,
			IsError: ir.Result.IsError,
		}
	}

	return results, lastErr
}

// toolCallResult represents the outcome of a single tool call.
type toolCallResult struct {
	CallID  string
	Name    string
	Content string
	IsError bool
}
