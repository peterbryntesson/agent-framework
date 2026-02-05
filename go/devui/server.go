// Copyright (c) Microsoft. All rights reserved.

package devui

import (
	"context"
	"fmt"
	"io/fs"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/microsoft/agent-framework-go/agent"
)

// Server provides a development UI for testing and debugging AI agents.
type Server struct {
	registry        *Registry
	collector       *TraceCollector
	port            int
	addr            string
	traceCollection bool
	maxTraces       int
	frontendFS      fs.FS
	middleware      []func(http.Handler) http.Handler
	corsOrigins     []string

	httpServer *http.Server
	mux        *http.ServeMux
	mu         sync.RWMutex
}

// NewServer creates a new DevUI server with the provided options.
func NewServer(opts ...Option) *Server {
	s := &Server{
		registry:        NewRegistry(),
		port:            8080,
		traceCollection: true,
		maxTraces:       1000,
		middleware:      make([]func(http.Handler) http.Handler, 0),
	}

	for _, opt := range opts {
		opt(s)
	}

	if s.traceCollection {
		s.collector = NewTraceCollector(s.maxTraces)
	}

	s.mux = http.NewServeMux()
	s.setupRoutes()

	return s
}

// RegisterAgent adds an agent to the DevUI server.
func (s *Server) RegisterAgent(name string, a agent.Agent) {
	s.registry.Register(name, a)
}

// UnregisterAgent removes an agent from the DevUI server.
func (s *Server) UnregisterAgent(name string) {
	s.registry.Unregister(name)
}

// TraceCollector returns the trace collector for custom integration.
// Returns nil if trace collection is disabled.
func (s *Server) TraceCollector() *TraceCollector {
	return s.collector
}

// setupRoutes configures the HTTP routes.
func (s *Server) setupRoutes() {
	// API routes
	s.mux.HandleFunc("GET /api/agents", s.handleListAgents)
	s.mux.HandleFunc("GET /api/agents/{name}", s.handleGetAgent)
	s.mux.HandleFunc("POST /api/agents/{name}/run", s.handleRunAgent)
	s.mux.HandleFunc("POST /api/agents/{name}/stream", s.handleStreamAgent)
	s.mux.HandleFunc("GET /api/traces", s.handleListTraces)
	s.mux.HandleFunc("GET /api/traces/{id}", s.handleGetTrace)
	s.mux.HandleFunc("DELETE /api/traces", s.handleClearTraces)

	// Health check
	s.mux.HandleFunc("GET /health", s.handleHealth)

	// Frontend - serve embedded or custom assets
	if s.frontendFS != nil {
		s.mux.Handle("/", http.FileServer(http.FS(s.frontendFS)))
	} else {
		// Serve a simple default page if no frontend is provided
		s.mux.HandleFunc("/", s.handleDefaultPage)
	}
}

// ServeHTTP implements http.Handler.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	handler := http.Handler(s.mux)

	// Apply CORS if configured
	if len(s.corsOrigins) > 0 {
		handler = s.corsMiddleware(handler)
	}

	// Apply custom middleware in reverse order (first added = outermost)
	for i := len(s.middleware) - 1; i >= 0; i-- {
		handler = s.middleware[i](handler)
	}

	handler.ServeHTTP(w, r)
}

// Start starts the HTTP server.
// This method blocks until the server is stopped or an error occurs.
func (s *Server) Start() error {
	addr := s.addr
	if addr == "" {
		addr = ":" + strconv.Itoa(s.port)
	}

	s.mu.Lock()
	s.httpServer = &http.Server{
		Addr:              addr,
		Handler:           s,
		ReadHeaderTimeout: 10 * time.Second,
	}
	s.mu.Unlock()

	return s.httpServer.ListenAndServe()
}

// StartAsync starts the HTTP server in a goroutine.
// Returns immediately after the server starts listening.
func (s *Server) StartAsync() error {
	errCh := make(chan error, 1)
	go func() {
		if err := s.Start(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	// Give the server a moment to start and report any immediate errors
	select {
	case err := <-errCh:
		return err
	case <-time.After(100 * time.Millisecond):
		return nil
	}
}

// Stop gracefully shuts down the server.
func (s *Server) Stop(ctx context.Context) error {
	s.mu.RLock()
	server := s.httpServer
	s.mu.RUnlock()

	if server != nil {
		return server.Shutdown(ctx)
	}
	return nil
}

// Address returns the server address.
// Returns empty string if the server hasn't been configured with an address.
func (s *Server) Address() string {
	if s.addr != "" {
		return s.addr
	}
	return fmt.Sprintf("http://localhost:%d", s.port)
}

// corsMiddleware adds CORS headers to responses.
func (s *Server) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		allowed := false

		for _, o := range s.corsOrigins {
			if o == "*" || o == origin {
				allowed = true
				break
			}
		}

		if allowed && origin != "" {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			w.Header().Set("Access-Control-Max-Age", "86400")
		}

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// handleDefaultPage serves a simple default page when no frontend is provided.
func (s *Server) handleDefaultPage(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	html := `<!DOCTYPE html>
<html>
<head>
    <title>Agent DevUI</title>
    <style>
        body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; margin: 40px; }
        h1 { color: #333; }
        .api-list { background: #f5f5f5; padding: 20px; border-radius: 8px; }
        code { background: #e0e0e0; padding: 2px 6px; border-radius: 4px; }
        a { color: #0066cc; }
    </style>
</head>
<body>
    <h1>🤖 Agent DevUI</h1>
    <p>Development server for testing AI agents.</p>
    
    <h2>Registered Agents</h2>
    <p>View at <a href="/api/agents">/api/agents</a></p>
    
    <h2>API Endpoints</h2>
    <div class="api-list">
        <p><code>GET /api/agents</code> - List registered agents</p>
        <p><code>GET /api/agents/{name}</code> - Get agent details</p>
        <p><code>POST /api/agents/{name}/run</code> - Run agent (non-streaming)</p>
        <p><code>POST /api/agents/{name}/stream</code> - Run agent (streaming)</p>
        <p><code>GET /api/traces</code> - List traces</p>
        <p><code>GET /api/traces/{id}</code> - Get trace details</p>
        <p><code>DELETE /api/traces</code> - Clear traces</p>
        <p><code>GET /health</code> - Health check</p>
    </div>
</body>
</html>`
	fmt.Fprint(w, html)
}

// handleHealth handles GET /health.
func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, `{"status":"healthy","agents":`, s.registry.Count(), `}`)
}
