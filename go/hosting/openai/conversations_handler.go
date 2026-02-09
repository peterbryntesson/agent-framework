// Copyright (c) Microsoft. All rights reserved.

package openai

import (
	"encoding/json"
	"net/http"
	"strconv"
)

// handleConversationsList handles GET /v1/conversations.
func (h *Handler) handleConversationsList(w http.ResponseWriter, r *http.Request) {
	agentID := r.URL.Query().Get("agent_id")
	if agentID == "" {
		writeError(w, http.StatusBadRequest, "invalid_request_error", "agent_id query parameter is required")
		return
	}

	if h.conversationIndex == nil {
		writeJSON(w, http.StatusOK, ListConversationsResponse{Data: []Conversation{}, HasMore: false})
		return
	}

	ids, err := h.conversationIndex.GetConversationIDs(r.Context(), agentID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "server_error", "Failed to list conversations: "+err.Error())
		return
	}

	conversations := make([]Conversation, 0, len(ids))
	for _, id := range ids {
		conversation, ok, err := h.conversationStore.GetConversation(r.Context(), id)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "server_error", "Failed to get conversation: "+err.Error())
			return
		}
		if ok {
			conversations = append(conversations, conversation)
		}
	}

	writeJSON(w, http.StatusOK, ListConversationsResponse{Data: conversations, HasMore: false})
}

// handleConversationsCreate handles POST /v1/conversations.
func (h *Handler) handleConversationsCreate(w http.ResponseWriter, r *http.Request) {
	var req CreateConversationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request_error", "Failed to parse request body: "+err.Error())
		return
	}

	conversation := Conversation{
		ID:       "conv_" + newID(),
		Object:   "conversation",
		Metadata: req.Metadata,
	}

	created, err := h.conversationStore.CreateConversation(r.Context(), conversation)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request_error", err.Error())
		return
	}

	if len(req.Items) > 0 {
		items := h.toItemResources(req.Items)
		if err := h.conversationStore.AddItems(r.Context(), created.ID, items); err != nil {
			writeError(w, http.StatusInternalServerError, "server_error", "Failed to add items: "+err.Error())
			return
		}
	}

	if h.conversationIndex != nil {
		if agentID, ok := created.Metadata["agent_id"]; ok && agentID != "" {
			_ = h.conversationIndex.AddConversation(r.Context(), agentID, created.ID)
		}
	}

	writeJSON(w, http.StatusOK, created)
}

// handleConversationsGet handles GET /v1/conversations/{conversationId}.
func (h *Handler) handleConversationsGet(w http.ResponseWriter, r *http.Request) {
	conversationID := r.PathValue("conversationId")
	conversation, ok, err := h.conversationStore.GetConversation(r.Context(), conversationID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "server_error", "Failed to get conversation: "+err.Error())
		return
	}
	if !ok {
		writeError(w, http.StatusNotFound, "invalid_request_error", "Conversation not found: "+conversationID)
		return
	}

	writeJSON(w, http.StatusOK, conversation)
}

// handleConversationsUpdate handles POST /v1/conversations/{conversationId}.
func (h *Handler) handleConversationsUpdate(w http.ResponseWriter, r *http.Request) {
	conversationID := r.PathValue("conversationId")
	var req UpdateConversationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request_error", "Failed to parse request body: "+err.Error())
		return
	}

	conversation, ok, err := h.conversationStore.GetConversation(r.Context(), conversationID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "server_error", "Failed to get conversation: "+err.Error())
		return
	}
	if !ok {
		writeError(w, http.StatusNotFound, "invalid_request_error", "Conversation not found: "+conversationID)
		return
	}

	conversation.Metadata = req.Metadata
	updated, _, err := h.conversationStore.UpdateConversation(r.Context(), conversation)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "server_error", "Failed to update conversation: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, updated)
}

// handleConversationsDelete handles DELETE /v1/conversations/{conversationId}.
func (h *Handler) handleConversationsDelete(w http.ResponseWriter, r *http.Request) {
	conversationID := r.PathValue("conversationId")
	conversation, ok, err := h.conversationStore.GetConversation(r.Context(), conversationID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "server_error", "Failed to get conversation: "+err.Error())
		return
	}

	deleted, err := h.conversationStore.DeleteConversation(r.Context(), conversationID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "server_error", "Failed to delete conversation: "+err.Error())
		return
	}
	if !deleted {
		writeError(w, http.StatusNotFound, "invalid_request_error", "Conversation not found: "+conversationID)
		return
	}

	if h.conversationIndex != nil {
		if agentID, ok := conversation.Metadata["agent_id"]; ok && agentID != "" {
			_ = h.conversationIndex.RemoveConversation(r.Context(), agentID, conversationID)
		}
	}

	writeJSON(w, http.StatusOK, DeleteResponse{ID: conversationID, Object: "conversation.deleted", Deleted: true})
}

// handleConversationsCreateItems handles POST /v1/conversations/{conversationId}/items.
func (h *Handler) handleConversationsCreateItems(w http.ResponseWriter, r *http.Request) {
	conversationID := r.PathValue("conversationId")
	var req CreateItemsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request_error", "Failed to parse request body: "+err.Error())
		return
	}

	items := h.toItemResources(req.Items)
	if err := h.conversationStore.AddItems(r.Context(), conversationID, items); err != nil {
		writeError(w, http.StatusNotFound, "invalid_request_error", err.Error())
		return
	}

	resp := ListItemsResponse{Object: "list", Data: items, HasMore: false}
	if len(items) > 0 {
		resp.FirstID = items[0].ID
		resp.LastID = items[len(items)-1].ID
	}
	writeJSON(w, http.StatusOK, resp)
}

// handleConversationsListItems handles GET /v1/conversations/{conversationId}/items.
func (h *Handler) handleConversationsListItems(w http.ResponseWriter, r *http.Request) {
	conversationID := r.PathValue("conversationId")
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

	items, err := h.conversationStore.ListItems(r.Context(), conversationID, limit, order, after)
	if err != nil {
		writeError(w, http.StatusNotFound, "invalid_request_error", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, items)
}

// handleConversationsGetItem handles GET /v1/conversations/{conversationId}/items/{itemId}.
func (h *Handler) handleConversationsGetItem(w http.ResponseWriter, r *http.Request) {
	conversationID := r.PathValue("conversationId")
	itemID := r.PathValue("itemId")
	item, ok, err := h.conversationStore.GetItem(r.Context(), conversationID, itemID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "server_error", "Failed to get item: "+err.Error())
		return
	}
	if !ok {
		writeError(w, http.StatusNotFound, "invalid_request_error", "Item not found: "+itemID)
		return
	}

	writeJSON(w, http.StatusOK, item)
}

// handleConversationsDeleteItem handles DELETE /v1/conversations/{conversationId}/items/{itemId}.
func (h *Handler) handleConversationsDeleteItem(w http.ResponseWriter, r *http.Request) {
	conversationID := r.PathValue("conversationId")
	itemID := r.PathValue("itemId")
	deleted, err := h.conversationStore.DeleteItem(r.Context(), conversationID, itemID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "server_error", "Failed to delete item: "+err.Error())
		return
	}
	if !deleted {
		writeError(w, http.StatusNotFound, "invalid_request_error", "Item not found: "+itemID)
		return
	}

	writeJSON(w, http.StatusOK, DeleteResponse{ID: itemID, Object: "conversation.item.deleted", Deleted: true})
}

func (h *Handler) toItemResources(items []ItemParam) []ItemResource {
	result := make([]ItemResource, 0, len(items))
	for _, item := range items {
		result = append(result, ItemResource{
			ID:        "item_" + newID(),
			Type:      item.Type,
			Role:      item.Role,
			Content:   item.Content,
			CallID:    item.CallID,
			Name:      item.Name,
			Arguments: item.Arguments,
			Output:    item.Output,
			CreatedAt: nowUnix(),
		})
	}
	return result
}
