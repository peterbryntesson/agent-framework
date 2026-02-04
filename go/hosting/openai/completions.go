// Copyright (c) Microsoft. All rights reserved.

package openai

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/microsoft/agent-framework-go/agent"
	"github.com/microsoft/agent-framework-go/chat"
)

// handleChatCompletions handles POST /v1/chat/completions.
func (h *Handler) handleChatCompletions(w http.ResponseWriter, r *http.Request) {
	// Parse request body
	var req ChatCompletionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request_error", "Failed to parse request body: "+err.Error())
		return
	}

	// Validate required fields
	if len(req.Messages) == 0 {
		writeError(w, http.StatusBadRequest, "invalid_request_error", "messages is required")
		return
	}

	// Convert to agent messages
	messages := ToAgentMessages(req.Messages)

	// Get or create session
	session, err := h.getOrCreateSession(r.Context(), r)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "server_error", "Failed to get session: "+err.Error())
		return
	}

	// Route to streaming or non-streaming
	if req.Stream && h.streamingEnabled {
		h.streamCompletions(w, r.Context(), session, messages, req)
	} else {
		h.completeSync(w, r.Context(), session, messages, req)
	}
}

// completeSync handles non-streaming completions.
func (h *Handler) completeSync(
	w http.ResponseWriter,
	ctx context.Context,
	session agent.Session,
	messages []chat.Message,
	req ChatCompletionRequest,
) {
	// Build run options
	opts := h.buildRunOptions(session)

	// Execute agent
	response, err := h.agent.Run(ctx, messages, opts...)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "agent_error", "Agent execution failed: "+err.Error())
		return
	}

	// Save session if store is configured
	if err := h.saveSession(ctx, session); err != nil {
		writeError(w, http.StatusInternalServerError, "server_error", "Failed to save session: "+err.Error())
		return
	}

	// Convert to OpenAI response format
	resp := h.toCompletionResponse(response, req.Model)
	writeJSON(w, http.StatusOK, resp)
}

// getOrCreateSession retrieves an existing session or creates a new one.
func (h *Handler) getOrCreateSession(ctx context.Context, r *http.Request) (agent.Session, error) {
	convID := r.Header.Get("X-Conversation-ID")

	if h.sessionStore != nil && convID != "" {
		return h.sessionStore.GetSession(ctx, h.agent, convID)
	}

	// Create a new in-memory session
	return h.agent.NewSession(ctx)
}

// saveSession persists the session if a store is configured.
func (h *Handler) saveSession(ctx context.Context, session agent.Session) error {
	if h.sessionStore == nil {
		return nil
	}
	// We need to get the conversation ID from the request context
	// For now, we'll use the session ID as the conversation ID
	return h.sessionStore.SaveSession(ctx, h.agent, session.ID(), session)
}

// buildRunOptions creates agent run options.
func (h *Handler) buildRunOptions(session agent.Session) []agent.RunOption {
	var opts []agent.RunOption
	if session != nil {
		opts = append(opts, agent.WithSession(session))
	}
	return opts
}

// toCompletionResponse converts an agent response to OpenAI format.
func (h *Handler) toCompletionResponse(resp *agent.Response, model string) ChatCompletionResponse {
	if model == "" {
		model = h.modelName
	}

	responseID := resp.ResponseID
	if responseID == "" {
		responseID = "chatcmpl-" + uuid.New().String()
	}

	choices := make([]ChatCompletionChoice, 0, len(resp.Messages))
	for i, msg := range resp.Messages {
		choices = append(choices, ChatCompletionChoice{
			Index:        i,
			Message:      FromAgentMessage(msg),
			FinishReason: finishReasonToString(resp.FinishReason),
		})
	}

	// Ensure at least one choice
	if len(choices) == 0 {
		choices = append(choices, ChatCompletionChoice{
			Index:        0,
			Message:      ChatCompletionMessage{Role: "assistant", Content: ""},
			FinishReason: "stop",
		})
	}

	response := ChatCompletionResponse{
		ID:      responseID,
		Object:  "chat.completion",
		Created: time.Now().Unix(),
		Model:   model,
		Choices: choices,
	}

	// Add usage if available
	if resp.Usage != nil {
		response.Usage = &Usage{
			PromptTokens:     resp.Usage.InputTokens,
			CompletionTokens: resp.Usage.OutputTokens,
			TotalTokens:      resp.Usage.TotalTokens,
		}
	}

	return response
}

// finishReasonToString converts an agent finish reason to OpenAI format.
func finishReasonToString(reason agent.FinishReason) string {
	switch reason {
	case agent.FinishReasonStop:
		return "stop"
	case agent.FinishReasonLength:
		return "length"
	case agent.FinishReasonToolCalls:
		return "tool_calls"
	case agent.FinishReasonContentFilter:
		return "content_filter"
	default:
		return "stop"
	}
}

// writeJSON writes a JSON response.
func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

// writeError writes an error response in OpenAI format.
func writeError(w http.ResponseWriter, status int, errType, message string) {
	resp := ErrorResponse{
		Error: ErrorDetail{
			Type:    errType,
			Message: message,
		},
	}
	writeJSON(w, status, resp)
}
