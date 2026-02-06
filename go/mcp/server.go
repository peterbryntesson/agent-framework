// Copyright (c) Microsoft. All rights reserved.

package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/microsoft/agent-framework-go/tool"
)

// Server exposes tools and resources as an MCP server.
// It handles the MCP protocol handshake and provides tool invocation
// and resource access to connected clients. The server is safe for
// concurrent use.
type Server struct {
	serverInfo    Implementation
	capabilities  ServerCapabilities
	tools         []tool.Tool
	resources     []ResourceInfo
	prompts       []Prompt
	resourceFn    func(ctx context.Context, uri string) (*ResourceContent, error)
	loggingFn     func(level LoggingLevel)
	samplingFn    func(ctx context.Context, params CreateMessageParams) (CreateMessageResult, error)
	logLevel      LoggingLevel
	notifications *notificationHub
	mu            sync.RWMutex
	initialized   bool
}

// ServerOption configures a Server.
type ServerOption func(*Server)

// NewServer creates a new MCP server with the given options.
// The server must be configured with tools and/or resources before
// it can handle client requests.
func NewServer(opts ...ServerOption) *Server {
	s := &Server{
		serverInfo: Implementation{
			Name:    "agent-framework-go",
			Version: "1.0.0",
		},
		capabilities:  ServerCapabilities{},
		notifications: newNotificationHub(),
	}

	for _, opt := range opts {
		opt(s)
	}

	// Set capabilities based on configuration
	if len(s.tools) > 0 {
		s.capabilities.Tools = &ToolsCapability{}
	}
	if len(s.resources) > 0 || s.resourceFn != nil {
		s.capabilities.Resources = &ResourcesCapability{}
	}
	if len(s.prompts) > 0 {
		s.capabilities.Prompts = &PromptsCapability{}
	}
	if s.loggingFn != nil {
		s.capabilities.Logging = &LoggingCapability{}
	}

	return s
}

// WithServerInfo sets the server implementation info.
// This information is sent to clients during initialization.
func WithServerInfo(name, version string) ServerOption {
	return func(s *Server) {
		s.serverInfo = Implementation{
			Name:    name,
			Version: version,
		}
	}
}

// WithTools adds tools to the server.
// These tools will be available to clients via tools/list and tools/call.
func WithTools(tools ...tool.Tool) ServerOption {
	return func(s *Server) {
		s.tools = append(s.tools, tools...)
	}
}

// WithResources adds static resources to the server.
// These resources will be available to clients via resources/list and resources/read.
func WithResources(resources ...ResourceInfo) ServerOption {
	return func(s *Server) {
		s.resources = append(s.resources, resources...)
	}
}

// WithPrompts adds prompts to the server.
func WithPrompts(prompts ...Prompt) ServerOption {
	return func(s *Server) {
		s.prompts = append(s.prompts, prompts...)
	}
}

// WithResourceHandler sets a function to handle resource reads.
// The handler is called for each resources/read request and should
// return the resource content for the given URI.
func WithResourceHandler(fn func(ctx context.Context, uri string) (*ResourceContent, error)) ServerOption {
	return func(s *Server) {
		s.resourceFn = fn
	}
}

// WithLoggingHandler sets a handler for logging level changes.
func WithLoggingHandler(fn func(level LoggingLevel)) ServerOption {
	return func(s *Server) {
		s.loggingFn = fn
	}
}

// WithSamplingHandler sets a handler for sampling/createMessage requests.
func WithSamplingHandler(fn func(ctx context.Context, params CreateMessageParams) (CreateMessageResult, error)) ServerOption {
	return func(s *Server) {
		s.samplingFn = fn
	}
}

// Handle processes a JSON-RPC request and returns a response.
// This is the main entry point for handling MCP protocol messages.
// Returns nil for notifications that don't require a response.
func (s *Server) Handle(ctx context.Context, req *Request) *Response {
	switch req.Method {
	case "initialize":
		return s.handleInitialize(ctx, req)
	case "notifications/initialized":
		// No response needed for notifications
		s.mu.Lock()
		s.initialized = true
		s.mu.Unlock()
		return nil
	case "tools/list":
		return s.handleToolsList(ctx, req)
	case "tools/call":
		return s.handleToolsCall(ctx, req)
	case "resources/list":
		return s.handleResourcesList(ctx, req)
	case "resources/read":
		return s.handleResourcesRead(ctx, req)
	case "prompts/list":
		return s.handlePromptsList(ctx, req)
	case "prompts/get":
		return s.handlePromptsGet(ctx, req)
	case "logging/setLevel":
		return s.handleLoggingSetLevel(ctx, req)
	case "sampling/createMessage":
		return s.handleSamplingCreateMessage(ctx, req)
	default:
		return s.errorResponse(req.ID, MethodNotFound, "Method not found: "+req.Method, nil)
	}
}

// handleInitialize performs the MCP initialization handshake.
func (s *Server) handleInitialize(ctx context.Context, req *Request) *Response {
	var params InitializeParams
	if len(req.Params) > 0 {
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return s.errorResponse(req.ID, InvalidParams, "Invalid initialize params", nil)
		}
	}

	// Build the initialization result
	result := InitializeResult{
		ProtocolVersion: ProtocolVersion,
		Capabilities:    s.capabilities,
		ServerInfo:      s.serverInfo,
	}

	return s.successResponse(req.ID, result)
}

// handleToolsList returns the list of available tools.
func (s *Server) handleToolsList(ctx context.Context, req *Request) *Response {
	s.mu.RLock()
	tools := s.tools
	s.mu.RUnlock()

	toolInfos := make([]ToolInfo, len(tools))
	for i, t := range tools {
		toolInfos[i] = ToolInfo{
			Name:        t.Name(),
			Description: t.Description(),
			InputSchema: t.Parameters(),
		}
	}

	result := ListToolsResult{
		Tools: toolInfos,
	}

	return s.successResponse(req.ID, result)
}

// handleToolsCall invokes a tool and returns the result.
func (s *Server) handleToolsCall(ctx context.Context, req *Request) *Response {
	var params CallToolParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return s.errorResponse(req.ID, InvalidParams, "Invalid params", nil)
	}

	// Find the tool
	s.mu.RLock()
	var targetTool tool.Tool
	for _, t := range s.tools {
		if t.Name() == params.Name {
			targetTool = t
			break
		}
	}
	s.mu.RUnlock()

	if targetTool == nil {
		return s.errorResponse(req.ID, InvalidParams, "Tool not found: "+params.Name, nil)
	}

	// Invoke the tool
	result, err := targetTool.Invoke(ctx, params.Arguments)
	if err != nil {
		return s.successResponse(req.ID, CallToolResult{
			Content: []Content{NewTextContent(err.Error())},
			IsError: true,
		})
	}

	return s.successResponse(req.ID, CallToolResult{
		Content: []Content{NewTextContent(result.Content)},
		IsError: result.IsError,
	})
}

// handleResourcesList returns the list of available resources.
func (s *Server) handleResourcesList(ctx context.Context, req *Request) *Response {
	s.mu.RLock()
	resources := s.resources
	s.mu.RUnlock()

	result := ListResourcesResult{
		Resources: resources,
	}

	return s.successResponse(req.ID, result)
}

// handleResourcesRead reads a resource by URI.
func (s *Server) handleResourcesRead(ctx context.Context, req *Request) *Response {
	var params ReadResourceParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return s.errorResponse(req.ID, InvalidParams, "Invalid params", nil)
	}

	// Try the resource handler first
	s.mu.RLock()
	resourceFn := s.resourceFn
	resources := s.resources
	s.mu.RUnlock()

	if resourceFn != nil {
		content, err := resourceFn(ctx, params.URI)
		if err != nil {
			return s.errorResponse(req.ID, InternalError, err.Error(), nil)
		}
		if content != nil {
			return s.successResponse(req.ID, ReadResourceResult{
				Contents: []ResourceContent{*content},
			})
		}
	}

	// Check static resources
	for _, r := range resources {
		if r.URI == params.URI {
			// For static resources, return a basic content with the resource info
			content := ResourceContent{
				URI:      r.URI,
				MimeType: r.MimeType,
			}
			return s.successResponse(req.ID, ReadResourceResult{
				Contents: []ResourceContent{content},
			})
		}
	}

	return s.errorResponse(req.ID, InvalidParams, "Resource not found: "+params.URI, nil)
}

// handlePromptsList returns the list of available prompts.
func (s *Server) handlePromptsList(ctx context.Context, req *Request) *Response {
	s.mu.RLock()
	prompts := s.prompts
	s.mu.RUnlock()

	promptInfos := make([]PromptInfo, len(prompts))
	for i, p := range prompts {
		promptInfos[i] = PromptInfo{
			Name:        p.Name,
			Description: p.Description,
			Arguments:   p.Arguments,
		}
	}

	result := ListPromptsResult{
		Prompts: promptInfos,
	}

	return s.successResponse(req.ID, result)
}

// handlePromptsGet returns a prompt by name.
func (s *Server) handlePromptsGet(ctx context.Context, req *Request) *Response {
	var params GetPromptParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return s.errorResponse(req.ID, InvalidParams, "Invalid params", nil)
	}

	s.mu.RLock()
	for _, p := range s.prompts {
		if p.Name == params.Name {
			s.mu.RUnlock()
			return s.successResponse(req.ID, GetPromptResult{Prompt: p})
		}
	}
	s.mu.RUnlock()

	return s.errorResponse(req.ID, InvalidParams, "Prompt not found: "+params.Name, nil)
}

// handleLoggingSetLevel updates the logging level.
func (s *Server) handleLoggingSetLevel(ctx context.Context, req *Request) *Response {
	var params LoggingSetLevelParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return s.errorResponse(req.ID, InvalidParams, "Invalid params", nil)
	}

	s.mu.Lock()
	s.logLevel = params.Level
	loggingFn := s.loggingFn
	s.mu.Unlock()

	if loggingFn != nil {
		loggingFn(params.Level)
	}

	return s.successResponse(req.ID, map[string]string{"status": "ok"})
}

// handleSamplingCreateMessage returns a sampled response when configured.
func (s *Server) handleSamplingCreateMessage(ctx context.Context, req *Request) *Response {
	var params CreateMessageParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return s.errorResponse(req.ID, InvalidParams, "Invalid params", nil)
	}

	s.mu.RLock()
	samplingFn := s.samplingFn
	s.mu.RUnlock()

	if samplingFn == nil {
		return s.errorResponse(req.ID, MethodNotFound, "Sampling handler not configured", nil)
	}

	result, err := samplingFn(ctx, params)
	if err != nil {
		return s.errorResponse(req.ID, InternalError, err.Error(), nil)
	}

	return s.successResponse(req.ID, result)
}

// successResponse creates a successful JSON-RPC response.
func (s *Server) successResponse(id interface{}, result interface{}) *Response {
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return s.errorResponse(id, InternalError, "Failed to marshal result", nil)
	}

	return &Response{
		JSONRPC: JSONRPCVersion,
		ID:      id,
		Result:  resultJSON,
	}
}

// errorResponse creates an error JSON-RPC response.
func (s *Server) errorResponse(id interface{}, code int, message string, data interface{}) *Response {
	var dataJSON json.RawMessage
	if data != nil {
		dataJSON, _ = json.Marshal(data)
	}

	return &Response{
		JSONRPC: JSONRPCVersion,
		ID:      id,
		Error: &ResponseError{
			Code:    code,
			Message: message,
			Data:    dataJSON,
		},
	}
}

// ServeStdio runs the server reading JSON-RPC from stdin and writing to stdout.
// This is the standard way to expose an MCP server as a CLI tool.
// The server runs until the context is canceled or stdin reaches EOF.
func (s *Server) ServeStdio(ctx context.Context) error {
	reader := bufio.NewReader(os.Stdin)
	encoder := json.NewEncoder(os.Stdout)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		line, err := reader.ReadBytes('\n')
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}

		// Skip empty lines
		if len(line) == 0 || (len(line) == 1 && line[0] == '\n') {
			continue
		}

		var req Request
		if err := json.Unmarshal(line, &req); err != nil {
			// Skip malformed requests
			continue
		}

		resp := s.Handle(ctx, &req)
		if resp != nil { // Don't send response for notifications
			if err := encoder.Encode(resp); err != nil {
				return err
			}
		}
	}
}

// HTTPHandler returns an http.Handler for the server.
// The handler expects JSON-RPC requests in the request body and
// returns JSON-RPC responses. This enables HTTP-based MCP communication.
func (s *Server) HTTPHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			s.SSEHandler().ServeHTTP(w, r)
			return
		}
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Read and parse the request
		body, err := io.ReadAll(r.Body)
		if err != nil {
			s.writeErrorResponse(w, nil, ParseError, "Failed to read request body")
			return
		}

		var req Request
		if err := json.Unmarshal(body, &req); err != nil {
			s.writeErrorResponse(w, nil, ParseError, "Invalid JSON")
			return
		}

		// Handle the request
		resp := s.Handle(r.Context(), &req)

		// Write the response
		w.Header().Set("Content-Type", "application/json")
		if resp == nil {
			// For notifications, return 204 No Content
			w.WriteHeader(http.StatusNoContent)
			return
		}

		if err := json.NewEncoder(w).Encode(resp); err != nil {
			// Can't write error response at this point, just log
			return
		}
	})
}

// SSEHandler returns an http.Handler that streams notifications via SSE.
func (s *Server) SSEHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")

		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
			return
		}

		client := s.notifications.Subscribe()
		defer s.notifications.Unsubscribe(client)

		ctx := r.Context()
		keepAlive := time.NewTicker(30 * time.Second)
		defer keepAlive.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-keepAlive.C:
				_, _ = w.Write([]byte(": ping\n\n"))
				flusher.Flush()
			case payload := <-client:
				_, _ = w.Write([]byte("event: notification\n"))
				_, _ = w.Write([]byte("data: "))
				_, _ = w.Write(payload)
				_, _ = w.Write([]byte("\n\n"))
				flusher.Flush()
			}
		}
	})
}

// writeErrorResponse writes a JSON-RPC error response to the HTTP response writer.
func (s *Server) writeErrorResponse(w http.ResponseWriter, id interface{}, code int, message string) {
	resp := s.errorResponse(id, code, message, nil)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// AddTool adds a tool to the server at runtime.
// This is useful for dynamically registering tools after server creation.
func (s *Server) AddTool(t tool.Tool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.tools = append(s.tools, t)
	if s.capabilities.Tools == nil {
		s.capabilities.Tools = &ToolsCapability{}
	}
	s.notifications.Broadcast(Notification{JSONRPC: JSONRPCVersion, Method: "notifications/tools/list_changed"})
}

// AddResource adds a resource to the server at runtime.
// This is useful for dynamically registering resources after server creation.
func (s *Server) AddResource(r ResourceInfo) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.resources = append(s.resources, r)
	if s.capabilities.Resources == nil {
		s.capabilities.Resources = &ResourcesCapability{}
	}
	s.notifications.Broadcast(Notification{JSONRPC: JSONRPCVersion, Method: "notifications/resources/list_changed"})
}

// AddPrompt adds a prompt to the server at runtime.
func (s *Server) AddPrompt(p Prompt) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.prompts = append(s.prompts, p)
	if s.capabilities.Prompts == nil {
		s.capabilities.Prompts = &PromptsCapability{}
	}
	s.notifications.Broadcast(Notification{JSONRPC: JSONRPCVersion, Method: "notifications/prompts/list_changed"})
}

// LogMessage broadcasts a logging notification to connected clients.
func (s *Server) LogMessage(level LoggingLevel, data json.RawMessage) {
	params, err := json.Marshal(LoggingMessageNotificationParams{Level: level, Data: data})
	if err != nil {
		return
	}
	s.notifications.Broadcast(Notification{
		JSONRPC: JSONRPCVersion,
		Method:  "logging/message",
		Params:  params,
	})
}
