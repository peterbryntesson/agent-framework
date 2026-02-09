// Copyright (c) Microsoft. All rights reserved.

package openai

import (
	"net/http"

	"github.com/microsoft/agent-framework-go/agent"
	"github.com/microsoft/agent-framework-go/hosting"
)

// Handler serves OpenAI-compatible endpoints for an agent.
//
// The handler implements http.Handler and routes requests to the appropriate
// endpoint handlers based on the request path.
type Handler struct {
	agent             agent.Agent
	sessionStore      hosting.SessionStore
	modelName         string
	streamingEnabled  bool
	basePath          string
	responsesService  ResponsesService
	conversationStore ConversationStore
	conversationIndex AgentConversationIndex
	mux               *http.ServeMux
}

// NewHandler creates an HTTP handler for the given agent.
//
// The handler serves OpenAI-compatible endpoints:
//   - POST /v1/chat/completions - Chat completions
//   - GET /v1/models - List available models
//
// Example:
//
//	handler := openai.NewHandler(myAgent,
//	    openai.WithModelName("my-model"),
//	    openai.WithStreamingEnabled(true),
//	    openai.WithSessionStore(store),
//	)
//	http.ListenAndServe(":8080", handler)
func NewHandler(a agent.Agent, opts ...Option) *Handler {
	h := &Handler{
		agent:            a,
		streamingEnabled: true,
		basePath:         "/v1",
	}

	// Apply options
	for _, opt := range opts {
		opt(h)
	}

	// Set default model name from agent
	if h.modelName == "" {
		h.modelName = a.Name()
		if h.modelName == "" {
			h.modelName = "agent"
		}
	}

	if h.conversationStore == nil {
		h.conversationStore = NewInMemoryConversationStore()
	}
	if h.responsesService == nil {
		h.responsesService = NewInMemoryResponsesService(a, h.conversationStore)
	}

	// Create router
	h.mux = http.NewServeMux()
	h.setupRoutes()

	return h
}

// setupRoutes configures the HTTP routes.
func (h *Handler) setupRoutes() {
	h.mux.HandleFunc("POST "+h.basePath+"/chat/completions", h.handleChatCompletions)
	h.mux.HandleFunc("GET "+h.basePath+"/models", h.handleListModels)
	h.mux.HandleFunc("GET "+h.basePath+"/models/{model}", h.handleGetModel)
	h.mux.HandleFunc("POST "+h.basePath+"/responses", h.handleResponsesCreate)
	h.mux.HandleFunc("GET "+h.basePath+"/responses/{responseId}", h.handleResponsesGet)
	h.mux.HandleFunc("POST "+h.basePath+"/responses/{responseId}/cancel", h.handleResponsesCancel)
	h.mux.HandleFunc("DELETE "+h.basePath+"/responses/{responseId}", h.handleResponsesDelete)
	h.mux.HandleFunc("GET "+h.basePath+"/responses/{responseId}/input_items", h.handleResponsesListInputItems)
	h.mux.HandleFunc("GET "+h.basePath+"/conversations", h.handleConversationsList)
	h.mux.HandleFunc("POST "+h.basePath+"/conversations", h.handleConversationsCreate)
	h.mux.HandleFunc("GET "+h.basePath+"/conversations/{conversationId}", h.handleConversationsGet)
	h.mux.HandleFunc("POST "+h.basePath+"/conversations/{conversationId}", h.handleConversationsUpdate)
	h.mux.HandleFunc("DELETE "+h.basePath+"/conversations/{conversationId}", h.handleConversationsDelete)
	h.mux.HandleFunc("POST "+h.basePath+"/conversations/{conversationId}/items", h.handleConversationsCreateItems)
	h.mux.HandleFunc("GET "+h.basePath+"/conversations/{conversationId}/items", h.handleConversationsListItems)
	h.mux.HandleFunc("GET "+h.basePath+"/conversations/{conversationId}/items/{itemId}", h.handleConversationsGetItem)
	h.mux.HandleFunc("DELETE "+h.basePath+"/conversations/{conversationId}/items/{itemId}", h.handleConversationsDeleteItem)
}

// ServeHTTP implements http.Handler.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.mux.ServeHTTP(w, r)
}

// handleListModels handles GET /v1/models.
func (h *Handler) handleListModels(w http.ResponseWriter, _ *http.Request) {
	resp := ModelsResponse{
		Object: "list",
		Data: []ModelInfo{
			{
				ID:      h.modelName,
				Object:  "model",
				Created: 0,
				OwnedBy: "agent-framework",
			},
		},
	}
	writeJSON(w, http.StatusOK, resp)
}

// handleGetModel handles GET /v1/models/{model}.
func (h *Handler) handleGetModel(w http.ResponseWriter, r *http.Request) {
	model := r.PathValue("model")
	if model != h.modelName {
		writeError(w, http.StatusNotFound, "model_not_found", "Model not found: "+model)
		return
	}

	resp := ModelInfo{
		ID:      h.modelName,
		Object:  "model",
		Created: 0,
		OwnedBy: "agent-framework",
	}
	writeJSON(w, http.StatusOK, resp)
}
