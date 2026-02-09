---
title: Go Parity Remediation Plan
description: Plan to address parity gaps and issues from the 2026-02-06 Go Epic 5 review.
author: GitHub Copilot
ms.date: 2026-02-06
ms.topic: plan
keywords:
  - go
  - parity
  - durable
  - mcp
  - declarative
  - hosting
estimated_reading_time: 6
---

## Context

Use this plan to address parity gaps identified in the review for Go Epic 5 parity with .NET and Python. Prioritize items that block compatibility and API coverage, then close remaining deviations and documentation gaps.

## Goals

* Align durable agent session identity, serialization, and run options with .NET and Python
* Close MCP transport and protocol capability gaps
* Expand declarative agent tool mapping and provider coverage
* Add missing hosting endpoints and align AG-UI request models
* Resolve DevUI parity gaps and finalize follow-up fixes

## Scope

* In scope: Go durable, MCP, declarative, hosting, protocol, DevUI, and Purview fixes identified in the review
* Out of scope: New feature requests unrelated to parity gaps or review findings

## Implementation Checklist

### Phase 1: Durable agent parity

* [ ] Align session ID format with @name@key and ensure serialization includes session ID
* [ ] Register history query in workflow and validate GetSession path
* [ ] Expand run request options to match .NET and Python features
* [ ] Preserve content fields in state entries, including data, uri, error, usage, and hosted content
* [ ] Persist function call arguments as structured data, not strings
* [ ] Mark error responses and exclude them from history context

### Phase 2: MCP parity and transport reliability

* [x] Add WebSocket MCP transport for parity with Python tooling
* [x] Implement prompt, logging, and sampling support in client and server
* [x] Implement SSE endpoint or adjust transport pathing to avoid dead /sse usage
* [x] Add single-reader demultiplexing for stdio transport
* [x] Add hosted MCP tool support or document parity decision
* [x] Fix MCP HTTP transport documentation example to avoid /sse duplication

### Phase 3: Declarative agent parity

* [x] Expand validation to include model fields and schema requirements
* [x] Add tool parsing support for MCP, web search, file search, and code interpreter
* [x] Apply tool bindings when creating agents
* [x] Expand PowerFx evaluation beyond Env.VAR with safe environment access
* [x] Expand provider mapping to match Python providers and API types

### Phase 4: Hosting and protocol parity

* [x] Expand AG-UI RunRequest model to include state, tools, context, forwarded props, and run IDs
* [x] Add OpenAI Responses and Conversations hosting endpoints
* [x] Align OpenAI session persistence to save and load by conversation ID
* [x] Implement AG-UI client WithHTTPClient behavior
* [x] Replace HostedAgentBuilder panic with error return or explicit validation API

### Phase 5: DevUI and Purview parity

* [ ] Add embedded DevUI frontend assets or document injection requirements with a parity rationale
* [ ] Decide on TokenCredential interface alignment with Azure SDK patterns

### Phase 6: Validation and documentation

* [ ] Run Go lint and format checks relevant to modified packages
* [ ] Update or add docs for any behavior changes and parity decisions
* [ ] Capture a changes log for the remediation work

## Validation Plan

* Run Go linting and formatting for affected packages
* Run targeted tests for durable, MCP, declarative, and hosting modules
* Verify parity-sensitive behaviors against .NET and Python implementations

## Deliverables

* Updated Go durable, MCP, declarative, hosting, protocol, DevUI, and Purview implementations
* Updated documentation and examples reflecting new parity behavior
* Changes log for the remediation release
