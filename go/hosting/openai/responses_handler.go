// Copyright (c) Microsoft. All rights reserved.

package openai

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
)

// handleResponsesCreate handles POST /v1/responses.
func (h *Handler) handleResponsesCreate(w http.ResponseWriter, r *http.Request) {
	var req CreateResponse
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request_error", "Failed to parse request body: "+err.Error())
		return
	}

	stream := req.Stream != nil && *req.Stream
	if q := r.URL.Query().Get("stream"); q != "" {
		if parsed, err := strconv.ParseBool(q); err == nil {
			stream = parsed
		}
	}

	if stream {
		h.streamResponses(w, r, req)
		return
	}

	response, err := h.responsesService.CreateResponse(r.Context(), req)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "server_error", "Failed to create response: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, response)
}

// handleResponsesGet handles GET /v1/responses/{responseId}.
func (h *Handler) handleResponsesGet(w http.ResponseWriter, r *http.Request) {
	responseID := r.PathValue("responseId")
	stream := false
	if q := r.URL.Query().Get("stream"); q != "" {
		if parsed, err := strconv.ParseBool(q); err == nil {
			stream = parsed
		}
	}

	if stream {
		h.streamResponseByID(w, r, responseID)
		return
	}

	response, ok, err := h.responsesService.GetResponse(r.Context(), responseID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "server_error", "Failed to get response: "+err.Error())
		return
	}
	if !ok {
		writeError(w, http.StatusNotFound, "invalid_request_error", "Response not found: "+responseID)
		return
	}

	writeJSON(w, http.StatusOK, response)
}

// handleResponsesCancel handles POST /v1/responses/{responseId}/cancel.
func (h *Handler) handleResponsesCancel(w http.ResponseWriter, r *http.Request) {
	responseID := r.PathValue("responseId")
	response, err := h.responsesService.CancelResponse(r.Context(), responseID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request_error", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, response)
}

// handleResponsesDelete handles DELETE /v1/responses/{responseId}.
func (h *Handler) handleResponsesDelete(w http.ResponseWriter, r *http.Request) {
	responseID := r.PathValue("responseId")
	deleted, err := h.responsesService.DeleteResponse(r.Context(), responseID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "server_error", "Failed to delete response: "+err.Error())
		return
	}
	if !deleted {
		writeError(w, http.StatusNotFound, "invalid_request_error", "Response not found: "+responseID)
		return
	}

	writeJSON(w, http.StatusOK, DeleteResponse{ID: responseID, Object: "response", Deleted: true})
}

// handleResponsesListInputItems handles GET /v1/responses/{responseId}/input_items.
func (h *Handler) handleResponsesListInputItems(w http.ResponseWriter, r *http.Request) {
	responseID := r.PathValue("responseId")
	limit := 0
	if q := r.URL.Query().Get("limit"); q != "" {
		if parsed, err := strconv.Atoi(q); err == nil {
			limit = parsed
		}
	}

	order := SortOrderDesc
	if q := r.URL.Query().Get("order"); q != "" {
		switch q {
		case "asc":
			order = SortOrderAsc
		case "desc":
			order = SortOrderDesc
		}
	}

	after := r.URL.Query().Get("after")

	items, err := h.responsesService.ListResponseInputItems(r.Context(), responseID, limit, order, after)
	if err != nil {
		writeError(w, http.StatusNotFound, "invalid_request_error", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, items)
}

func (h *Handler) streamResponses(w http.ResponseWriter, r *http.Request, req CreateResponse) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, "streaming_unsupported", "Streaming is not supported by the server")
		return
	}

	stream, _, err := h.responsesService.CreateResponseStreaming(r.Context(), req)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "server_error", "Failed to create response: "+err.Error())
		return
	}

	for evt := range stream {
		writeResponseSSEEvent(w, flusher, evt)
	}
}

func (h *Handler) streamResponseByID(w http.ResponseWriter, r *http.Request, responseID string) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, "streaming_unsupported", "Streaming is not supported by the server")
		return
	}

	startingAfter := 0
	if q := r.URL.Query().Get("starting_after"); q != "" {
		if parsed, err := strconv.Atoi(q); err == nil {
			startingAfter = parsed
		}
	}

	stream, err := h.responsesService.GetResponseStreaming(r.Context(), responseID, startingAfter)
	if err != nil {
		writeError(w, http.StatusNotFound, "invalid_request_error", err.Error())
		return
	}

	for evt := range stream {
		writeResponseSSEEvent(w, flusher, evt)
	}
}

func writeResponseSSEEvent(w http.ResponseWriter, flusher http.Flusher, event StreamingResponseEvent) {
	data, err := json.Marshal(event)
	if err != nil {
		return
	}
	fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event.Type, data)
	flusher.Flush()
}
