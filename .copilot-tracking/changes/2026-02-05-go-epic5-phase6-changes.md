<!-- markdownlint-disable-file -->
# Release Changes: Go Epic 5 Phase 6 - Enterprise Features

**Related Plan**: 2026-02-04-go-epic5-implementation-plan.md  
**Implementation Date**: 2026-02-05

## Summary

Implemented Phase 6 Enterprise Features including the DevUI development server for testing and debugging AI agents, and the Purview middleware for Microsoft Purview content policy integration.

## Changes

### Added

* go/devui/doc.go - Package documentation with overview, quick start, and API endpoint descriptions
* go/devui/options.go - Functional options for server configuration (port, tracing, CORS, middleware)
* go/devui/types.go - API request/response types (AgentInfo, RunRequest, RunResponse, TraceInfo, etc.)
* go/devui/discovery.go - Agent registry for managing registered agents with thread-safe operations
* go/devui/tracing.go - OpenTelemetry trace collector with LRU eviction for visualization
* go/devui/server.go - HTTP server implementation with route setup, CORS, and lifecycle management
* go/devui/handlers.go - API endpoint handlers for agents, traces, and health endpoints
* go/devui/mock_test.go - Mock agent implementation for testing
* go/devui/discovery_test.go - Unit tests for agent registry
* go/devui/tracing_test.go - Unit tests for trace collector
* go/devui/server_test.go - Integration tests for all API endpoints
* go/purview/doc.go - Package documentation with overview and usage examples
* go/purview/options.go - Functional options and TokenCredential interface
* go/purview/types.go - Policy evaluation types (Settings, PolicyResult, Violation, etc.)
* go/purview/client.go - Purview API client with authentication and caching
* go/purview/middleware.go - Agent middleware for content policy evaluation
* go/purview/client_test.go - Unit tests for Purview client
* go/purview/middleware_test.go - Unit tests for Purview middleware

### Modified

None

### Removed

None

## Additional or Deviating Changes

* Frontend assets were not embedded in the DevUI package as originally planned in Task 5.5.1.5. Instead, a default HTML page is served that lists available API endpoints. Custom frontends can be provided via the `WithFrontendFS` option.
  * Reason: Frontend development requires separate tooling and would add complexity. The API-first approach allows flexible frontend integration.

* The Purview middleware uses a simplified TokenCredential interface instead of directly depending on Azure SDK.
  * Reason: Reduces external dependencies and allows for easier testing and custom credential implementations.

## Release Summary

**Total Files Affected**: 17 files created

**Files Created**:
* go/devui/ - 11 files (doc.go, options.go, types.go, discovery.go, tracing.go, server.go, handlers.go, mock_test.go, discovery_test.go, tracing_test.go, server_test.go)
* go/purview/ - 6 files (doc.go, options.go, types.go, client.go, middleware.go, client_test.go, middleware_test.go)

**Dependencies**:
* No new external dependencies added (uses existing go.opentelemetry.io/otel/sdk/trace)

**Test Results**:
* DevUI: 37 tests passed
* Purview: 16 tests passed
* Total: 53 tests, all passing

**Key Features Implemented**:
1. DevUI Server (Task 5.5.1)
   * Agent registration and discovery
   * REST API for agent execution (streaming and non-streaming)
   * OpenTelemetry trace collection with LRU eviction
   * SSE streaming for real-time updates
   * CORS support and custom middleware hooks
   * Health check endpoint

2. Purview Middleware (Task 5.5.2)
   * Content policy evaluation against Microsoft Purview
   * Pre and post-processing of agent messages
   * Configurable blocking behavior on violations
   * Custom violation handlers
   * Token caching for API authentication
   * Scope definition caching with ETag support
