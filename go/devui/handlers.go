// Copyright (c) Microsoft. All rights reserved.

package devui

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/microsoft/agent-framework-go/agent"
	"github.com/microsoft/agent-framework-go/chat"
	"go.opentelemetry.io/otel/trace"
)

// handleListAgents handles GET /api/agents.
func (s *Server) handleListAgents(w http.ResponseWriter, _ *http.Request) {
	agents := s.registry.List()
	writeJSON(w, http.StatusOK, AgentsResponse{Agents: agents})
}

// handleGetAgent handles GET /api/agents/{name}.
func (s *Server) handleGetAgent(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	info, ok := s.registry.GetInfo(name)
	if !ok {
		writeError(w, http.StatusNotFound, "agent_not_found", "Agent not found: "+name)
		return
	}
	writeJSON(w, http.StatusOK, info)
}

// handleRunAgent handles POST /api/agents/{name}/run.
func (s *Server) handleRunAgent(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	a := s.registry.Get(name)
	if a == nil {
		writeError(w, http.StatusNotFound, "agent_not_found", "Agent not found: "+name)
		return
	}

	var req RunRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid request body: "+err.Error())
		return
	}

	if len(req.Messages) == 0 {
		writeError(w, http.StatusBadRequest, "invalid_request", "At least one message is required")
		return
	}

	// Convert messages
	messages := make([]agent.Message, len(req.Messages))
	for i, m := range req.Messages {
		messages[i] = chat.NewMessageWithContents(chat.Role(m.Role), chat.NewTextContent(m.Content))
	}

	// Generate conversation ID if not provided
	conversationID := req.ConversationID
	if conversationID == "" {
		conversationID = uuid.New().String()
	}

	// Run agent
	ctx := r.Context()
	response, err := a.Run(ctx, messages)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "agent_error", "Agent execution failed: "+err.Error())
		return
	}

	// Extract trace ID from context if available
	traceID := ""
	if span := trace.SpanFromContext(ctx); span.SpanContext().IsValid() {
		traceID = span.SpanContext().TraceID().String()
	}

	// Build response
	resp := RunResponse{
		ID:             response.ResponseID,
		ConversationID: conversationID,
		Content:        response.Text(),
		FinishReason:   finishReasonToString(response.FinishReason),
		TraceID:        traceID,
	}

	if response.Usage != nil {
		resp.Usage = &UsageInfo{
			InputTokens:  response.Usage.InputTokens,
			OutputTokens: response.Usage.OutputTokens,
			TotalTokens:  response.Usage.TotalTokens,
		}
	}

	writeJSON(w, http.StatusOK, resp)
}

// handleStreamAgent handles POST /api/agents/{name}/stream.
func (s *Server) handleStreamAgent(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	a := s.registry.Get(name)
	if a == nil {
		writeError(w, http.StatusNotFound, "agent_not_found", "Agent not found: "+name)
		return
	}

	var req RunRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid request body: "+err.Error())
		return
	}

	if len(req.Messages) == 0 {
		writeError(w, http.StatusBadRequest, "invalid_request", "At least one message is required")
		return
	}

	// Convert messages
	messages := make([]agent.Message, len(req.Messages))
	for i, m := range req.Messages {
		messages[i] = chat.NewMessageWithContents(chat.Role(m.Role), chat.NewTextContent(m.Content))
	}

	// Generate conversation ID if not provided
	conversationID := req.ConversationID
	if conversationID == "" {
		conversationID = uuid.New().String()
	}

	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, "streaming_unsupported", "Streaming not supported")
		return
	}

	// Create cancellable context
	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	// Start streaming
	updates, err := a.RunStream(ctx, messages)
	if err != nil {
		writeSSE(w, "error", fmt.Sprintf(`{"error":"%s"}`, err.Error()))
		flusher.Flush()
		return
	}

	// Send start event
	writeSSE(w, "start", fmt.Sprintf(`{"conversationId":"%s"}`, conversationID))
	flusher.Flush()

	// Stream updates
	for update := range updates {
		var eventType, data string

		switch update.Kind {
		case agent.UpdateKindContentDelta:
			if update.Delta != nil {
				eventType = "delta"
				dataBytes, _ := json.Marshal(map[string]string{
					"content": update.Delta.TextDelta,
				})
				data = string(dataBytes)
			}
		case agent.UpdateKindToolCall:
			if update.Delta != nil && update.Delta.ToolCallID != "" {
				eventType = "tool_call"
				dataBytes, _ := json.Marshal(map[string]string{
					"toolCallId": update.Delta.ToolCallID,
					"toolName":   update.Delta.Name,
				})
				data = string(dataBytes)
			}
		case agent.UpdateKindToolResult:
			eventType = "tool_result"
			dataBytes, _ := json.Marshal(map[string]string{
				"toolCallId": update.Delta.ToolCallID,
			})
			data = string(dataBytes)
		case agent.UpdateKindDone:
			eventType = "complete"
			responseData := map[string]interface{}{
				"finishReason": finishReasonToString(update.FinishReason),
			}
			if update.Usage != nil {
				responseData["usage"] = map[string]int{
					"inputTokens":  update.Usage.InputTokens,
					"outputTokens": update.Usage.OutputTokens,
					"totalTokens":  update.Usage.TotalTokens,
				}
			}
			dataBytes, _ := json.Marshal(responseData)
			data = string(dataBytes)
		case agent.UpdateKindError:
			eventType = "error"
			errMsg := "Unknown error"
			if update.Error != nil {
				errMsg = update.Error.Error()
			}
			dataBytes, _ := json.Marshal(map[string]string{
				"error": errMsg,
			})
			data = string(dataBytes)
		}

		if eventType != "" && data != "" {
			writeSSE(w, eventType, data)
			flusher.Flush()
		}
	}

	// Send done event
	writeSSE(w, "done", `{"status":"complete"}`)
	flusher.Flush()
}

// handleListTraces handles GET /api/traces.
func (s *Server) handleListTraces(w http.ResponseWriter, _ *http.Request) {
	if s.collector == nil {
		writeJSON(w, http.StatusOK, TracesResponse{Traces: []TraceInfo{}, Total: 0})
		return
	}

	traces := s.collector.List()
	writeJSON(w, http.StatusOK, TracesResponse{Traces: traces, Total: len(traces)})
}

// handleGetTrace handles GET /api/traces/{id}.
func (s *Server) handleGetTrace(w http.ResponseWriter, r *http.Request) {
	if s.collector == nil {
		writeError(w, http.StatusNotFound, "trace_not_found", "Trace collection is disabled")
		return
	}

	traceID := r.PathValue("id")
	detail, ok := s.collector.Get(traceID)
	if !ok {
		writeError(w, http.StatusNotFound, "trace_not_found", "Trace not found: "+traceID)
		return
	}

	writeJSON(w, http.StatusOK, detail)
}

// handleClearTraces handles DELETE /api/traces.
func (s *Server) handleClearTraces(w http.ResponseWriter, _ *http.Request) {
	if s.collector != nil {
		s.collector.Clear()
	}
	w.WriteHeader(http.StatusNoContent)
}

// writeJSON writes a JSON response.
func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// writeError writes an error response.
func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, ErrorResponse{
		Error:   code,
		Message: message,
		Code:    code,
	})
}

// writeSSE writes a server-sent event.
func writeSSE(w http.ResponseWriter, event, data string) {
	fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event, data)
}

// finishReasonToString converts a FinishReason to a string.
func finishReasonToString(fr agent.FinishReason) string {
	switch fr {
	case agent.FinishReasonStop:
		return "stop"
	case agent.FinishReasonLength:
		return "length"
	case agent.FinishReasonToolCalls:
		return "tool_calls"
	case agent.FinishReasonContentFilter:
		return "content_filter"
	default:
		return "unknown"
	}
}
