// Copyright (c) Microsoft. All rights reserved.

package openai

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/microsoft/agent-framework-go/agent"
	"github.com/microsoft/agent-framework-go/chat"
)

// streamCompletions handles streaming completions with SSE.
func (h *Handler) streamCompletions(
	w http.ResponseWriter,
	ctx context.Context,
	conversationID string,
	session agent.Session,
	messages []chat.Message,
	req ChatCompletionRequest,
) {
	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	// Get flusher for streaming
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, "streaming_unsupported",
			"Streaming is not supported by the server")
		return
	}

	// Build run options
	opts := h.buildRunOptions(session)

	// Start streaming agent execution
	stream, err := h.agent.RunStream(ctx, messages, opts...)
	if err != nil {
		writeSSEError(w, flusher, "agent_error", "Failed to start agent: "+err.Error())
		return
	}

	// Generate response ID
	model := req.Model
	if model == "" {
		model = h.modelName
	}
	responseID := "chatcmpl-" + uuid.New().String()
	created := time.Now().Unix()

	// Track if we've sent the role yet
	roleSent := false

	// Process stream updates
	for update := range stream {
		if update.Error != nil {
			writeSSEError(w, flusher, "stream_error", update.Error.Error())
			return
		}

		chunk := h.updateToChunk(update, model, responseID, created, &roleSent)
		if chunk != nil {
			writeSSEChunk(w, flusher, chunk)
		}
	}

	// Save session if store is configured
	if err := h.saveSession(ctx, conversationID, session); err != nil {
		writeSSEError(w, flusher, "session_error", "Failed to save session: "+err.Error())
		return
	}

	// Send [DONE] marker
	writeSSEDone(w, flusher)
}

// updateToChunk converts an agent ResponseUpdate to an OpenAI chunk.
func (h *Handler) updateToChunk(
	update agent.ResponseUpdate,
	model string,
	responseID string,
	created int64,
	roleSent *bool,
) *ChatCompletionChunk {
	switch update.Kind {
	case agent.UpdateKindContentDelta:
		if update.Delta == nil {
			return nil
		}

		delta := ChatCompletionDelta{}

		// Send role with first content
		if !*roleSent {
			delta.Role = "assistant"
			*roleSent = true
		}

		// Extract text from delta
		if update.Delta.TextDelta != "" {
			delta.Content = update.Delta.TextDelta
		}

		return &ChatCompletionChunk{
			ID:      responseID,
			Object:  "chat.completion.chunk",
			Created: created,
			Model:   model,
			Choices: []ChatCompletionChunkChoice{
				{
					Index:        0,
					Delta:        delta,
					FinishReason: nil,
				},
			},
		}

	case agent.UpdateKindToolCall:
		// Send role if not yet sent
		delta := ChatCompletionDelta{}
		if !*roleSent {
			delta.Role = "assistant"
			*roleSent = true
		}

		// Add tool call delta from ContentDelta
		if update.Delta != nil {
			toolDelta := ToolCallDelta{
				Index: 0,
				ID:    update.Delta.ToolCallID,
				Type:  "function",
			}
			if update.Delta.Name != "" || update.Delta.ArgsDelta != "" {
				toolDelta.Function = &FunctionCallDelta{
					Name:      update.Delta.Name,
					Arguments: update.Delta.ArgsDelta,
				}
			}
			delta.ToolCalls = []ToolCallDelta{toolDelta}
		}

		return &ChatCompletionChunk{
			ID:      responseID,
			Object:  "chat.completion.chunk",
			Created: created,
			Model:   model,
			Choices: []ChatCompletionChunkChoice{
				{
					Index:        0,
					Delta:        delta,
					FinishReason: nil,
				},
			},
		}

	case agent.UpdateKindDone:
		finishReason := finishReasonToString(update.FinishReason)
		return &ChatCompletionChunk{
			ID:      responseID,
			Object:  "chat.completion.chunk",
			Created: created,
			Model:   model,
			Choices: []ChatCompletionChunkChoice{
				{
					Index:        0,
					Delta:        ChatCompletionDelta{},
					FinishReason: &finishReason,
				},
			},
		}

	case agent.UpdateKindUsage:
		if update.Usage == nil {
			return nil
		}
		return &ChatCompletionChunk{
			ID:      responseID,
			Object:  "chat.completion.chunk",
			Created: created,
			Model:   model,
			Choices: []ChatCompletionChunkChoice{},
			Usage: &Usage{
				PromptTokens:     update.Usage.InputTokens,
				CompletionTokens: update.Usage.OutputTokens,
				TotalTokens:      update.Usage.TotalTokens,
			},
		}

	case agent.UpdateKindMessageComplete:
		// Don't emit chunks for complete messages in streaming mode
		return nil

	default:
		return nil
	}
}

// writeSSEChunk writes a single SSE chunk.
func writeSSEChunk(w http.ResponseWriter, flusher http.Flusher, chunk *ChatCompletionChunk) {
	data, err := json.Marshal(chunk)
	if err != nil {
		return
	}
	fmt.Fprintf(w, "data: %s\n\n", data)
	flusher.Flush()
}

// writeSSEError writes an SSE error event.
func writeSSEError(w http.ResponseWriter, flusher http.Flusher, errType, message string) {
	errResp := ErrorResponse{
		Error: ErrorDetail{
			Type:    errType,
			Message: message,
		},
	}
	data, _ := json.Marshal(errResp)
	fmt.Fprintf(w, "data: %s\n\n", data)
	flusher.Flush()
	writeSSEDone(w, flusher)
}

// writeSSEDone writes the [DONE] marker to indicate stream completion.
func writeSSEDone(w http.ResponseWriter, flusher http.Flusher) {
	fmt.Fprint(w, "data: [DONE]\n\n")
	flusher.Flush()
}
