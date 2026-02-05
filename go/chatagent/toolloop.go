// Copyright (c) Microsoft. All rights reserved.

package chatagent

import (
	"context"
	"fmt"
	"sync"

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

		// Get response from chat client through middleware
		resp, _, err := a.invokeWithChatMiddleware(ctx, currentMessages, options, false)
		if err != nil {
			return nil, fmt.Errorf("chat request failed: %w", err)
		}

		// Accumulate usage (may be nil for cached responses)
		if resp.Usage != nil {
			totalUsage.InputTokens += resp.Usage.InputTokens
			totalUsage.OutputTokens += resp.Usage.OutputTokens
			totalUsage.TotalTokens += resp.Usage.TotalTokens
		}

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

		// Get streaming response through middleware
		_, streamUpdates, err := a.invokeWithChatMiddleware(ctx, currentMessages, options, true)
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
		result, err := a.invokeSingleToolCall(ctx, invoker, tc)
		if err != nil {
			lastErr = err
			if a.invocationConfig.TerminateOnUnknownCalls {
				return nil, err
			}
		}

		results = append(results, result)
	}

	return results, lastErr
}

// invokeSingleToolCall executes a single tool call through middleware.
func (a *Agent) invokeSingleToolCall(ctx context.Context, invoker *tool.Invoker, tc chat.ToolCall) (toolCallResult, error) {
	// Build function context
	funcCtx := &agent.FunctionContext{
		FunctionName: tc.Name,
		Arguments:    tc.Arguments,
		Metadata:     make(map[string]any),
	}

	// Terminal handler calls the actual invoker
	terminal := func(ctx context.Context, funcCtx *agent.FunctionContext) error {
		result, err := invoker.Invoke(ctx, funcCtx.FunctionName, funcCtx.Arguments)
		if err != nil {
			funcCtx.Error = err
		}
		funcCtx.Result = result
		return nil // Don't propagate invoker errors - let middleware handle them
	}

	// Execute through middleware chain if configured
	if a.functionMiddleware != nil {
		err := a.functionMiddleware.Process(ctx, funcCtx, terminal)
		if err != nil {
			return toolCallResult{
				CallID:  tc.ID,
				Name:    tc.Name,
				Content: "Middleware error",
				IsError: true,
			}, err
		}
	} else {
		_ = terminal(ctx, funcCtx)
	}

	// Extract result from function context
	var result tool.Result
	switch v := funcCtx.Result.(type) {
	case tool.Result:
		result = v
	case string:
		result = tool.NewResult(v)
	default:
		if funcCtx.Result != nil {
			r, err := tool.NewResultFromValue(funcCtx.Result)
			if err != nil {
				result = tool.NewResult(fmt.Sprintf("%v", funcCtx.Result))
			} else {
				result = r
			}
		}
	}

	content := result.Content
	if result.IsError && a.invocationConfig.IncludeDetailedErrors {
		content = fmt.Sprintf("Error: %s", content)
	}

	return toolCallResult{
		CallID:  tc.ID,
		Name:    tc.Name,
		Content: content,
		IsError: result.IsError,
	}, funcCtx.Error
}

// invokeToolCallsParallel executes tool calls concurrently.
func (a *Agent) invokeToolCallsParallel(ctx context.Context, invoker *tool.Invoker, toolCalls []chat.ToolCall) ([]toolCallResult, error) {
	results := make([]toolCallResult, len(toolCalls))
	errs := make([]error, len(toolCalls))

	var wg sync.WaitGroup
	for i, tc := range toolCalls {
		wg.Add(1)
		go func(idx int, call chat.ToolCall) {
			defer wg.Done()
			result, err := a.invokeSingleToolCall(ctx, invoker, call)
			results[idx] = result
			errs[idx] = err
		}(i, tc)
	}
	wg.Wait()

	var lastErr error
	for _, err := range errs {
		if err != nil {
			lastErr = err
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

// invokeWithChatMiddleware wraps chat client calls with middleware.
// Returns the response for non-streaming calls, or the stream for streaming calls.
func (a *Agent) invokeWithChatMiddleware(ctx context.Context, messages []chat.Message, options *chat.Options, isStreaming bool) (*chat.Response, <-chan chat.ResponseUpdate, error) {
	// Build ChatContext for middleware
	chatCtx := &agent.ChatContext{
		ClientMetadata: agent.ChatClientMetadata{
			ProviderName: a.client.Metadata().ProviderName,
			ModelID:      a.client.Metadata().ModelID,
			EndpointURI:  a.client.Metadata().EndpointURI,
		},
		Messages:    convertMessagesToAgentMessages(messages),
		Options:     convertChatOptionsToMap(options),
		Metadata:    make(map[string]any),
		IsStreaming: isStreaming,
	}

	// Terminal handler calls the actual chat client
	terminal := func(ctx context.Context, c *agent.ChatContext) error {
		if c.IsStreaming {
			stream, err := a.client.GetStreamingResponse(ctx, messages, options)
			if err != nil {
				return err
			}
			// Wrap the stream channel for middleware
			c.Stream = wrapStreamChannel(stream)
		} else {
			resp, err := a.client.GetResponse(ctx, messages, options)
			if err != nil {
				return err
			}
			c.Response = convertResponseToMiddlewareResponse(resp)
		}
		return nil
	}

	// Execute through middleware chain
	err := a.chatMiddleware.Process(ctx, chatCtx, terminal)
	if err != nil {
		return nil, nil, err
	}

	if isStreaming {
		// Return the stream - may be from middleware or terminal handler
		return nil, unwrapStreamChannel(chatCtx.Stream), nil
	}

	// Convert middleware response back to chat.Response
	return convertMiddlewareResponseToChatResponse(chatCtx.Response), nil, nil
}

// convertMessagesToAgentMessages converts chat.Message slice to agent.Message slice.
func convertMessagesToAgentMessages(messages []chat.Message) []agent.Message {
	// agent.Message is an alias to chat.Message, so direct conversion works
	result := make([]agent.Message, len(messages))
	copy(result, messages)
	return result
}

// convertChatOptionsToMap extracts options into a map for middleware access.
func convertChatOptionsToMap(options *chat.Options) map[string]any {
	if options == nil {
		return nil
	}
	result := make(map[string]any)
	if options.MaxTokens > 0 {
		result["max_tokens"] = options.MaxTokens
	}
	if options.Temperature > 0 {
		result["temperature"] = options.Temperature
	}
	if options.TopP > 0 {
		result["top_p"] = options.TopP
	}
	if len(options.StopSequences) > 0 {
		result["stop"] = options.StopSequences
	}
	if options.ResponseFormat != "" {
		result["response_format"] = options.ResponseFormat
	}
	if options.Seed != nil {
		result["seed"] = *options.Seed
	}
	// Copy metadata
	for k, v := range options.Metadata {
		result[k] = v
	}
	return result
}

// convertResponseToMiddlewareResponse converts chat.Response to middleware format.
func convertResponseToMiddlewareResponse(resp *chat.Response) *agent.ChatResponse {
	if resp == nil {
		return nil
	}
	return &agent.ChatResponse{
		Text:         resp.Text(),
		FinishReason: finishReasonToString(resp.FinishReason),
		RawResponse:  resp,
	}
}

// finishReasonToString converts chat.FinishReason to string.
func finishReasonToString(fr chat.FinishReason) string {
	switch fr {
	case chat.FinishReasonStop:
		return "stop"
	case chat.FinishReasonLength:
		return "length"
	case chat.FinishReasonToolCalls:
		return "tool_calls"
	case chat.FinishReasonContentFilter:
		return "content_filter"
	default:
		return "stop"
	}
}

// convertMiddlewareResponseToChatResponse extracts chat.Response from middleware response.
func convertMiddlewareResponseToChatResponse(resp *agent.ChatResponse) *chat.Response {
	if resp == nil {
		return nil
	}
	// If RawResponse is available, use it directly
	if raw, ok := resp.RawResponse.(*chat.Response); ok {
		return raw
	}
	// Fallback: construct minimal response from available data
	return &chat.Response{
		Message: chat.Message{
			Role:     chat.RoleAssistant,
			Contents: []chat.Content{chat.NewTextContent(resp.Text)},
		},
		FinishReason: parseFinishReason(resp.FinishReason),
	}
}

// parseFinishReason converts a string finish reason to chat.FinishReason.
func parseFinishReason(s string) chat.FinishReason {
	switch s {
	case "stop":
		return chat.FinishReasonStop
	case "length":
		return chat.FinishReasonLength
	case "tool_calls":
		return chat.FinishReasonToolCalls
	case "content_filter":
		return chat.FinishReasonContentFilter
	default:
		return chat.FinishReasonStop
	}
}

// wrapStreamChannel wraps chat.ResponseUpdate channel for middleware.
func wrapStreamChannel(stream <-chan chat.ResponseUpdate) <-chan agent.ChatResponseUpdate {
	out := make(chan agent.ChatResponseUpdate, 32)
	go func() {
		defer close(out)
		for update := range stream {
			agentUpdate := agent.ChatResponseUpdate{
				Kind:      updateKindToString(update.Kind),
				RawUpdate: update,
			}
			if update.Delta != nil {
				agentUpdate.TextDelta = update.Delta.TextDelta
			}
			if update.Error != nil {
				agentUpdate.Error = update.Error
			}
			out <- agentUpdate
		}
	}()
	return out
}

// updateKindToString converts chat.UpdateKind to string.
func updateKindToString(uk chat.UpdateKind) string {
	switch uk {
	case chat.UpdateKindContentDelta:
		return "content_delta"
	case chat.UpdateKindToolCall:
		return "tool_call"
	case chat.UpdateKindToolResult:
		return "tool_result"
	case chat.UpdateKindMessageComplete:
		return "message_complete"
	case chat.UpdateKindUsage:
		return "usage"
	case chat.UpdateKindError:
		return "error"
	case chat.UpdateKindDone:
		return "done"
	default:
		return "content_delta"
	}
}

// unwrapStreamChannel extracts original stream from middleware channel.
func unwrapStreamChannel(stream <-chan agent.ChatResponseUpdate) <-chan chat.ResponseUpdate {
	out := make(chan chat.ResponseUpdate, 32)
	go func() {
		defer close(out)
		for update := range stream {
			// If RawUpdate is available, use it directly
			if raw, ok := update.RawUpdate.(chat.ResponseUpdate); ok {
				out <- raw
				continue
			}
			// Fallback: construct minimal update
			out <- chat.ResponseUpdate{
				Kind: parseUpdateKind(update.Kind),
				Delta: &chat.ContentDelta{
					TextDelta: update.TextDelta,
				},
				Error: update.Error,
			}
		}
	}()
	return out
}

// parseUpdateKind converts a string update kind to chat.UpdateKind.
func parseUpdateKind(s string) chat.UpdateKind {
	switch s {
	case "content_delta":
		return chat.UpdateKindContentDelta
	case "tool_call":
		return chat.UpdateKindToolCall
	case "tool_result":
		return chat.UpdateKindToolResult
	case "message_complete":
		return chat.UpdateKindMessageComplete
	case "usage":
		return chat.UpdateKindUsage
	case "error":
		return chat.UpdateKindError
	case "done":
		return chat.UpdateKindDone
	default:
		return chat.UpdateKindContentDelta
	}
}
