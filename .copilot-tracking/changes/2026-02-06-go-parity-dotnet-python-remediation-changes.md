<!-- markdownlint-disable-file -->
# Release Changes: Go Parity Remediation

**Related Plan**: 2026-02-06-go-parity-dotnet-python-remediation-plan.instructions.md
**Implementation Date**: 2026-02-06

## Summary

Added MCP WebSocket transport, prompt/logging/sampling handlers, declarative parity updates, AG-UI run metadata handling, and OpenAI responses/conversations hosting endpoints with session persistence alignment.

## Changes

### Added

* go/mcp/transport_websocket.go - WebSocket MCP transport with response demultiplexing.
* go/mcp/server_notifications.go - Notification hub for SSE streaming.
* go/declarative/normalize.go - Normalization helpers for API and tool kinds.
* go/declarative/approval_mode.go - Decode MCP approval modes from scalar or object YAML.
* go/hosting/openai/responses_models.go - Minimal Responses API request/response models.
* go/hosting/openai/responses_service.go - In-memory Responses API service with streaming events.
* go/hosting/openai/responses_handler.go - Responses endpoints for create, get, cancel, delete, and input item listing.
* go/hosting/openai/conversations_models.go - Conversations API request/response models.
* go/hosting/openai/conversations_store.go - In-memory conversations store and agent index.
* go/hosting/openai/conversations_handler.go - Conversations endpoints for CRUD and item management.
* go/hosting/openai/util.go - Shared helpers for IDs and timestamps.
* go/hosting/openai/responses_test.go - Responses endpoint coverage.
* go/hosting/openai/conversations_test.go - Conversations endpoint coverage.

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
* go/protocol/agui/server.go - Expand run request fields and propagate metadata/tools into run options.
* go/protocol/agui/client.go - Add run metadata fields and honor custom HTTP client configuration.
* go/protocol/agui/client_test.go - Validate run metadata request shaping and client options.
* go/hosting/builder.go - Return errors for missing agent configuration.
* go/hosting/builder_test.go - Update builder tests for error-returning Build.
* go/hosting/doc.go - Refresh hosted agent builder usage example.
* go/hosting/openai/handler.go - Add responses/conversations routes and default services.
* go/hosting/openai/completions.go - Persist sessions by conversation ID and return header.
* go/hosting/openai/streaming.go - Persist sessions by conversation ID after streaming.
* go/hosting/openai/options.go - Add responses/conversations service options.
* go/hosting/openai/doc.go - Document responses and conversations endpoints.
* go/devui/doc.go - Document frontend asset injection requirements and parity rationale.
* go/purview/options.go - Align TokenCredential types with Azure SDK azcore/policy interfaces.
* go/purview/doc.go - Note Azure SDK TokenCredential alignment in Purview usage docs.
* go/go.mod - Add Azure SDK azcore dependency for Purview credential alignment.
* go/go.sum - Track Azure SDK azcore checksums.

### Removed

* None

## Additional or Deviating Changes

* Added minimal Responses/Conversations handler tests outside explicit plan scope
	* Ensures new endpoints have baseline coverage and reduces regression risk

## Release Summary

Pending remaining phases completion.
