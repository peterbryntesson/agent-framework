<!-- markdownlint-disable-file -->
# Release Changes: Go Parity Remediation

**Related Plan**: 2026-02-06-go-parity-dotnet-python-remediation-plan.instructions.md
**Implementation Date**: 2026-02-06

## Summary

Added MCP WebSocket transport, prompt/logging/sampling handlers, SSE notifications, and stdio demultiplexing updates.

## Changes

### Added

* go/mcp/transport_websocket.go - WebSocket MCP transport with response demultiplexing.
* go/mcp/server_notifications.go - Notification hub for SSE streaming.

### Modified

* go/mcp/transport.go - Add request ID key helper for transport demultiplexing.
* go/mcp/transport_http.go - Separate SSE endpoint handling and override option.
* go/mcp/transport_stdio.go - Single-reader demultiplexing for stdio transport.
* go/mcp/types.go - Add prompt, logging, and sampling types.
* go/mcp/client.go - Add prompt/logging/sampling client APIs and capabilities option.
* go/mcp/server.go - Add prompt/logging/sampling handlers and SSE endpoint.
* go/mcp/doc.go - Update HTTP transport example and document SSE handler usage.
* go/go.mod - Add gorilla/websocket dependency.
* go/go.sum - Track gorilla/websocket dependency checksum.

### Removed

* None

## Additional or Deviating Changes

* None

## Release Summary

Pending Phase 2 completion.
