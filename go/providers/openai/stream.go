// Copyright (c) Microsoft. All rights reserved.

package openai

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sync"

	"github.com/microsoft/agent-framework-go/chat"
	openailib "github.com/sashabaranov/go-openai"
)

// StreamProcessor handles the processing of OpenAI streaming responses.
// It provides utilities for accumulating deltas and managing stream state.
type StreamProcessor struct {
	// accumulated tracks tool call argument fragments during streaming.
	accumulated map[string]*toolCallAccumulator

	mu sync.Mutex
}

// toolCallAccumulator collects tool call data as it arrives in fragments.
type toolCallAccumulator struct {
	ID        string
	Name      string
	Arguments string
}

// NewStreamProcessor creates a new StreamProcessor instance.
func NewStreamProcessor() *StreamProcessor {
	return &StreamProcessor{
		accumulated: make(map[string]*toolCallAccumulator),
	}
}

// ProcessStream reads from an OpenAI stream and sends updates to the channel.
// This function handles context cancellation, EOF detection, and error recovery.
// The channel is closed when the stream ends or an error occurs.
func ProcessStream(ctx context.Context, stream *openailib.ChatCompletionStream, updates chan<- chat.ResponseUpdate) {
	processor := NewStreamProcessor()
	processor.process(ctx, stream, updates)
}

// process is the main loop that reads chunks from the stream.
func (sp *StreamProcessor) process(ctx context.Context, stream *openailib.ChatCompletionStream, updates chan<- chat.ResponseUpdate) {
	for {
		select {
		case <-ctx.Done():
			updates <- chat.ResponseUpdate{
				Kind:  chat.UpdateKindError,
				Error: ctx.Err(),
			}
			return
		default:
		}

		chunk, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			// Send any accumulated tool calls before finishing
			sp.flushToolCalls(updates)
			updates <- chat.ResponseUpdate{Kind: chat.UpdateKindDone}
			return
		}
		if err != nil {
			updates <- chat.ResponseUpdate{
				Kind:  chat.UpdateKindError,
				Error: fmt.Errorf("openai: stream error: %w", err),
			}
			return
		}

		sp.processChunk(chunk, updates)
	}
}

// processChunk handles a single chunk from the stream.
func (sp *StreamProcessor) processChunk(chunk openailib.ChatCompletionStreamResponse, updates chan<- chat.ResponseUpdate) {
	// Process choices in chunk
	for _, choice := range chunk.Choices {
		sp.processChoice(choice, updates)
	}

	// Usage in final chunk (if enabled via StreamOptions.IncludeUsage)
	if chunk.Usage != nil {
		updates <- chat.ResponseUpdate{
			Kind: chat.UpdateKindUsage,
			Usage: &chat.UsageDetails{
				InputTokens:  chunk.Usage.PromptTokens,
				OutputTokens: chunk.Usage.CompletionTokens,
				TotalTokens:  chunk.Usage.TotalTokens,
			},
		}
	}
}

// processChoice handles a single choice from a stream chunk.
func (sp *StreamProcessor) processChoice(choice openailib.ChatCompletionStreamChoice, updates chan<- chat.ResponseUpdate) {
	delta := choice.Delta

	// Handle text content delta
	if delta.Content != "" {
		updates <- chat.ResponseUpdate{
			Kind: chat.UpdateKindContentDelta,
			Delta: &chat.ContentDelta{
				Role:      chat.Role(delta.Role),
				TextDelta: delta.Content,
			},
		}
	}

	// Handle tool call deltas
	for _, tc := range delta.ToolCalls {
		sp.processToolCallDelta(tc, updates)
	}

	// Handle finish reason
	if choice.FinishReason != "" {
		// Flush accumulated tool calls before message complete
		if choice.FinishReason == openailib.FinishReasonToolCalls {
			sp.flushToolCalls(updates)
		}

		updates <- chat.ResponseUpdate{
			Kind:         chat.UpdateKindMessageComplete,
			FinishReason: fromOpenAIFinishReason(string(choice.FinishReason)),
		}
	}
}

// processToolCallDelta accumulates tool call fragments and sends updates.
func (sp *StreamProcessor) processToolCallDelta(tc openailib.ToolCall, updates chan<- chat.ResponseUpdate) {
	sp.mu.Lock()
	defer sp.mu.Unlock()

	// Get or create accumulator for this tool call
	acc, exists := sp.accumulated[tc.ID]
	if !exists && tc.ID != "" {
		acc = &toolCallAccumulator{ID: tc.ID}
		sp.accumulated[tc.ID] = acc
	} else if !exists {
		// Some deltas may have empty ID when streaming arguments for index-based calls
		// Use the index as a fallback key
		key := fmt.Sprintf("index_%d", tc.Index)
		acc, exists = sp.accumulated[key]
		if !exists {
			acc = &toolCallAccumulator{}
			sp.accumulated[key] = acc
		}
		if tc.ID != "" {
			acc.ID = tc.ID
		}
	}

	// Accumulate function name
	if tc.Function.Name != "" {
		acc.Name = tc.Function.Name

		// Send tool call start notification
		updates <- chat.ResponseUpdate{
			Kind: chat.UpdateKindToolCall,
			Delta: &chat.ContentDelta{
				ToolCallID: acc.ID,
				Name:       acc.Name,
			},
			Metadata: map[string]interface{}{
				"tool_call_id": acc.ID,
				"tool_name":    acc.Name,
			},
		}
	}

	// Accumulate arguments
	if tc.Function.Arguments != "" {
		acc.Arguments += tc.Function.Arguments

		// Send argument delta
		updates <- chat.ResponseUpdate{
			Kind: chat.UpdateKindContentDelta,
			Delta: &chat.ContentDelta{
				ToolCallID: acc.ID,
				ArgsDelta:  tc.Function.Arguments,
			},
		}
	}
}

// flushToolCalls sends any remaining accumulated tool call data.
func (sp *StreamProcessor) flushToolCalls(updates chan<- chat.ResponseUpdate) {
	sp.mu.Lock()
	defer sp.mu.Unlock()

	// Tool calls are already sent incrementally, so this is primarily
	// for cleanup and ensuring all state is flushed.
	sp.accumulated = make(map[string]*toolCallAccumulator)
}

// CollectStreamToResponse reads all updates from a stream channel and
// assembles them into a complete Response. This is useful for cases where
// streaming is used internally but a complete response is needed.
func CollectStreamToResponse(updates <-chan chat.ResponseUpdate) (*chat.Response, error) {
	var (
		textContent  string
		finishReason chat.FinishReason
		usage        *chat.UsageDetails
		toolCalls    []chat.ToolCall
		toolCallArgs = make(map[string]string)
		lastErr      error
	)

	for update := range updates {
		switch update.Kind {
		case chat.UpdateKindContentDelta:
			if update.Delta != nil {
				textContent += update.Delta.TextDelta

				// Accumulate tool call arguments
				if update.Delta.ToolCallID != "" && update.Delta.ArgsDelta != "" {
					toolCallArgs[update.Delta.ToolCallID] += update.Delta.ArgsDelta
				}
			}

		case chat.UpdateKindToolCall:
			if update.Delta != nil {
				tc := chat.ToolCall{
					ID:   update.Delta.ToolCallID,
					Name: update.Delta.Name,
				}
				toolCalls = append(toolCalls, tc)
			}

		case chat.UpdateKindMessageComplete:
			finishReason = update.FinishReason

		case chat.UpdateKindUsage:
			usage = update.Usage

		case chat.UpdateKindError:
			lastErr = update.Error

		case chat.UpdateKindDone:
			// Stream complete
		}
	}

	if lastErr != nil {
		return nil, lastErr
	}

	// Populate tool call arguments from accumulated data
	for i := range toolCalls {
		if args, ok := toolCallArgs[toolCalls[i].ID]; ok {
			toolCalls[i].Arguments = []byte(args)
		}
	}

	msg := chat.Message{
		Role: chat.RoleAssistant,
	}

	if textContent != "" {
		msg.Contents = []chat.Content{chat.NewTextContent(textContent)}
	}

	if len(toolCalls) > 0 {
		msg.ToolCalls = toolCalls
	}

	return &chat.Response{
		Message:      msg,
		FinishReason: finishReason,
		Usage:        usage,
	}, nil
}
