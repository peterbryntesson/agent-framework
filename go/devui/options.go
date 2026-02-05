// Copyright (c) Microsoft. All rights reserved.

package devui

import (
	"io/fs"
	"net/http"
)

// Option configures a DevUI Server.
type Option func(*Server)

// WithPort sets the HTTP port for the server.
// Default is 8080.
func WithPort(port int) Option {
	return func(s *Server) {
		s.port = port
	}
}

// WithAddress sets the full address (host:port) for the server.
// This takes precedence over WithPort.
func WithAddress(addr string) Option {
	return func(s *Server) {
		s.addr = addr
	}
}

// WithTraceCollection enables OpenTelemetry trace collection.
// Default is true.
func WithTraceCollection(enabled bool) Option {
	return func(s *Server) {
		s.traceCollection = enabled
	}
}

// WithMaxTraces sets the maximum number of traces to retain in memory.
// Older traces are evicted when the limit is reached.
// Default is 1000.
func WithMaxTraces(max int) Option {
	return func(s *Server) {
		s.maxTraces = max
	}
}

// WithFrontendFS sets a custom filesystem for serving frontend assets.
// The filesystem should contain an index.html at the root.
func WithFrontendFS(fsys fs.FS) Option {
	return func(s *Server) {
		s.frontendFS = fsys
	}
}

// WithMiddleware adds HTTP middleware to the server.
// Middleware is applied in the order added (first added = outermost).
func WithMiddleware(m func(http.Handler) http.Handler) Option {
	return func(s *Server) {
		s.middleware = append(s.middleware, m)
	}
}

// WithCORS enables CORS with the specified allowed origins.
// Pass "*" to allow all origins (not recommended for production).
func WithCORS(allowedOrigins ...string) Option {
	return func(s *Server) {
		s.corsOrigins = allowedOrigins
	}
}
