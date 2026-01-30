<!-- markdownlint-disable-file -->
# Release Changes: Golang Port Plan Updates

**Related Plan**: docs/design/golang-port-plan.md
**Related Review**: .copilot-tracking/reviews/2026-01-30-golang-port-plan-review.md
**Implementation Date**: 2026-01-30

## Summary

Updated the Go port plan to address critical and major findings from the implementation review. Added missing provider packages, explicit hosted tool type definitions, AsyncRunContent type for long-running operations, expanded hosting section with protocol-specific packages, and scope exclusion notes for experimental features.

## Changes

### Modified

* [docs/design/golang-port-plan.md](../../docs/design/golang-port-plan.md) - Comprehensive updates to address review findings:

  **Section 1.3 Feature Matrix - Scope Exclusions Added:**
  - Added explicit exclusion rows for Lab/Experimental, Workflow Generators, and Legacy Support
  - Added scope exclusion notes explaining why these are excluded from initial Go port

  **Section 2.1 Proposed Module Layout - Missing Providers Added:**
  - `providers/bedrock/` - AWS Bedrock LLM integration (client.go, auth.go, options.go)
  - `providers/azure/persistent.go` - Azure AI Foundry Persistent Agents
  - `providers/azuresearch/` - Azure AI Search for RAG (client.go, index.go, options.go)
  - `providers/copilotstudio/` - Copilot Studio integration
  - `providers/githubcopilot/` - GitHub Copilot SDK
  - `providers/foundrylocal/` - Local model execution

  **Section 2.1 Proposed Module Layout - Memory Subdirectories Added:**
  - `memory/mem0/` - Mem0 memory provider
  - `memory/redis/` - Redis message store
  - `memory/cosmos/` - Cosmos DB NoSQL persistence

  **Section 2.1 Proposed Module Layout - Governance Packages Added:**
  - `governance/purview/` - Microsoft Purview integration (client.go, policy.go, options.go)
  - `governance/compliance.go` - Compliance interfaces

  **Section 2.1 Proposed Module Layout - Developer UI Added:**
  - `devui/` - Developer debugging UI (server.go, inspector.go, templates/)

  **Section 2.1 Proposed Module Layout - Hosting Expanded:**
  - `hosting/a2a/` - A2A protocol HTTP handler and routing
  - `hosting/agui/` - AG-UI SSE handler and support
  - `hosting/openaicompat/` - OpenAI-compatible API handler
  - `hosting/azurefunctions/` - Azure Functions trigger and bindings

  **Section 3.4 Response Types - AsyncRunContent Added:**
  - Added `AsyncRunContent` struct with RunID, Status, ThreadID, timestamps, and error info
  - Added `AsyncRunStatus` enum with Queued, InProgress, RequiresAction, Completed, Cancelled, Failed, Expired states
  - Added `AsyncRunError` struct for failed async runs
  - Added `IsTerminal()` helper method on AsyncRunStatus

  **Section 3.5 Tool Interface - Hosted Tool Types Added:**
  - Added `HostedTool` interface extending Tool with IsHosted() and RawRepresentation()
  - Added `HostedWebSearchTool` with SearchContextSize and UserLocation
  - Added `HostedCodeInterpreterTool` with Container and FileIDs
  - Added `HostedFileSearchTool` with VectorStoreIDs and MaxResults
  - Added `HostedMCPTool` for MCP server bridging
  - Added `UserLocation` struct for web search geographic context

  **Section 4 Phase 2 - Bedrock Provider Added:**
  - Added AWS Bedrock Provider deliverable with IAM authentication, Claude/Titan models, streaming

  **Section 4 Phase 3 - Vector Search/RAG Added:**
  - Added Vector Search/RAG deliverable with Azure AI Search, vector stores, hybrid search

  **Section 4 Phase 5 - Enterprise Features Expanded:**
  - Added Enterprise Integrations deliverable (Copilot Studio, GitHub Copilot, Purview, Azure AI Search)
  - Added Developer Tools deliverable (DevUI, Agent inspector, Foundry Local)
  - Added Protocol-Specific Hosting deliverable (A2A, AG-UI, OpenAI-compat, Azure Functions)

## Additional or Deviating Changes

* No deviations from the review recommendations

## Release Summary

**Files Affected:** 1 (docs/design/golang-port-plan.md)

**Additions to Plan:**
- 8 new provider packages (Bedrock, Azure Persistent, Azure Search, Copilot Studio, GitHub Copilot, Foundry Local, Mem0, Redis, Cosmos)
- 2 governance packages (Purview, Compliance)
- 1 developer UI package
- 4 protocol-specific hosting packages (A2A, AG-UI, OpenAI-compat, Azure Functions)
- 5 hosted tool type definitions
- 1 AsyncRunContent type with status enum
- 3 scope exclusion notes

**Review Findings Addressed:**
- [Critical] Missing Provider: Amazon Bedrock ✓
- [Critical] Missing Enterprise Integrations ✓
- [Critical] Missing Hosting Variants ✓
- [Major] Incomplete Tool Abstractions ✓
- [Major] Missing Long-Running Operation Types ✓
- [Major] Missing Vector Search/RAG ✓
- [Major] Missing Developer UI ✓
- [Minor] Memory Integration Details ✓
- [Minor] Lab/Experimental Scope ✓
