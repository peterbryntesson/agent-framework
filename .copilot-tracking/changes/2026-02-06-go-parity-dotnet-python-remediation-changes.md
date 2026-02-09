<!-- markdownlint-disable-file -->
# Release Changes: Go Parity Remediation

**Related Plan**: 2026-02-06-go-parity-dotnet-python-remediation-plan.instructions.md
**Implementation Date**: 2026-02-06

## Summary

Added MCP WebSocket transport, prompt/logging/sampling handlers, and declarative parity updates for hosted tools, bindings, validation, and provider mapping.

## Changes

### Added

* go/mcp/transport_websocket.go - WebSocket MCP transport with response demultiplexing.
* go/mcp/server_notifications.go - Notification hub for SSE streaming.
* go/declarative/normalize.go - Normalization helpers for API and tool kinds.
* go/declarative/approval_mode.go - Decode MCP approval modes from scalar or object YAML.

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
* go/declarative/models.go - Expand declarative tool/model schema for hosted tools and MCP metadata.
* go/declarative/tools.go - Add hosted tool parsing for MCP, web search, file search, and code interpreter.
* go/declarative/factory.go - Apply tool bindings during agent creation and normalize tool kinds.
* go/declarative/eval.go - Support Env() and Env[] PowerFx-style environment evaluation.
* go/declarative/providers.go - Route OpenAI/AzureOpenAI apiType to chat or responses clients.
* go/declarative/validation.go - Validate model configuration, schema requirements, and MCP tool fields.

### Removed

* None

## Additional or Deviating Changes

* None

## Release Summary

Pending remaining phases completion.
