---
applyTo: '.copilot-tracking/changes/2026-02-02-golang-epic2-providers-changes.md'
---
<!-- markdownlint-disable-file -->
# Implementation Plan: Go Port - Epic 2: LLM Provider Implementations

## Overview

Detailed implementation plan for Epic 2 of the Go SDK port, covering LLM provider implementations, tool system, and observability. Aligns with existing .NET and Python patterns for API consistency.

## Objectives

* Implement OpenAI, Azure OpenAI, Anthropic, Bedrock, and Ollama provider packages
* Create a tool system with function tools, hosted tools, and invocation middleware
* Add OpenTelemetry instrumentation following GenAI semantic conventions
* Achieve feature parity with .NET `Microsoft.Agents.AI.*` and Python `agent_framework` providers

## Context Summary

### Project Files

* [go/agent/](go/agent/) - Core agent interfaces (already implemented)
* [go/chat/](go/chat/) - Chat client interfaces (already implemented)
* [go/observability/](go/observability/) - Basic OTEL setup (needs expansion)

### Research References

* [.copilot-tracking/subagent/2026-02-02/dotnet-providers-research.md](.copilot-tracking/subagent/2026-02-02/dotnet-providers-research.md) - .NET provider patterns
* [.copilot-tracking/subagent/2026-02-02/python-providers-research.md](.copilot-tracking/subagent/2026-02-02/python-providers-research.md) - Python provider patterns
* [.copilot-tracking/subagent/2026-02-02/go-current-state-research.md](.copilot-tracking/subagent/2026-02-02/go-current-state-research.md) - Current Go implementation

### Standards References

* Go 1.22+ idioms and error handling patterns
* OpenTelemetry GenAI semantic conventions (`gen_ai.*` attributes)
* Functional options pattern for configuration

## Implementation Checklist

### [ ] Feature 2.1: Tool System Package

<!-- parallelizable: false -->
<!-- Priority: Implement first as providers depend on tool types -->

* [x] Step 2.1.1: Define Tool interfaces and types
  * Details: .copilot-tracking/details/2026-02-02-golang-epic2-providers-details.md (Lines 30-120)
* [ ] Step 2.1.2: Implement FunctionTool with reflection
  * Details: .copilot-tracking/details/2026-02-02-golang-epic2-providers-details.md (Lines 122-220)
* [ ] Step 2.1.3: Implement hosted tool types
  * Details: .copilot-tracking/details/2026-02-02-golang-epic2-providers-details.md (Lines 222-320)
* [ ] Step 2.1.4: Implement function invocation utilities
  * Details: .copilot-tracking/details/2026-02-02-golang-epic2-providers-details.md (Lines 322-400)
* [ ] Step 2.1.5: Add tool package tests
  * Details: .copilot-tracking/details/2026-02-02-golang-epic2-providers-details.md (Lines 402-450)
* [ ] Validate Feature 2.1
  * Run `go test ./tool/...` with 90%+ coverage

### [ ] Feature 2.2: OpenAI Provider Package

<!-- parallelizable: true -->

* [ ] Step 2.2.1: Create OpenAI client structure and options
  * Details: .copilot-tracking/details/2026-02-02-golang-epic2-providers-details.md (Lines 455-550)
* [ ] Step 2.2.2: Implement Chat Completions API integration
  * Details: .copilot-tracking/details/2026-02-02-golang-epic2-providers-details.md (Lines 552-650)
* [ ] Step 2.2.3: Implement streaming response handling
  * Details: .copilot-tracking/details/2026-02-02-golang-epic2-providers-details.md (Lines 652-750)
* [ ] Step 2.2.4: Implement tool calling support
  * Details: .copilot-tracking/details/2026-02-02-golang-epic2-providers-details.md (Lines 752-830)
* [ ] Step 2.2.5: Implement Responses API client
  * Details: .copilot-tracking/details/2026-02-02-golang-epic2-providers-details.md (Lines 832-920)
* [ ] Step 2.2.6: Add OpenAI provider tests
  * Details: .copilot-tracking/details/2026-02-02-golang-epic2-providers-details.md (Lines 922-980)
* [ ] Validate Feature 2.2
  * Run `go test ./providers/openai/...` with integration tests

### [ ] Feature 2.3: Azure OpenAI Provider Package

<!-- parallelizable: true -->

* [ ] Step 2.3.1: Create Azure OpenAI client structure and options
  * Details: .copilot-tracking/details/2026-02-02-golang-epic2-providers-details.md (Lines 985-1080)
* [ ] Step 2.3.2: Implement Azure AD authentication
  * Details: .copilot-tracking/details/2026-02-02-golang-epic2-providers-details.md (Lines 1082-1160)
* [ ] Step 2.3.3: Implement Azure OpenAI completions and streaming
  * Details: .copilot-tracking/details/2026-02-02-golang-epic2-providers-details.md (Lines 1162-1250)
* [ ] Step 2.3.4: Implement Azure AI Foundry agent client
  * Details: .copilot-tracking/details/2026-02-02-golang-epic2-providers-details.md (Lines 1252-1350)
* [ ] Step 2.3.5: Implement Persistent Agents support
  * Details: .copilot-tracking/details/2026-02-02-golang-epic2-providers-details.md (Lines 1352-1440)
* [ ] Step 2.3.6: Add Azure provider tests
  * Details: .copilot-tracking/details/2026-02-02-golang-epic2-providers-details.md (Lines 1442-1500)
* [ ] Validate Feature 2.3
  * Run `go test ./providers/azure/...` with integration tests

### [ ] Feature 2.4: Anthropic Provider Package

<!-- parallelizable: true -->

* [ ] Step 2.4.1: Create Anthropic client structure and options
  * Details: .copilot-tracking/details/2026-02-02-golang-epic2-providers-details.md (Lines 1505-1590)
* [ ] Step 2.4.2: Implement message format conversion
  * Details: .copilot-tracking/details/2026-02-02-golang-epic2-providers-details.md (Lines 1592-1680)
* [ ] Step 2.4.3: Implement SSE streaming
  * Details: .copilot-tracking/details/2026-02-02-golang-epic2-providers-details.md (Lines 1682-1770)
* [ ] Step 2.4.4: Implement tool use blocks
  * Details: .copilot-tracking/details/2026-02-02-golang-epic2-providers-details.md (Lines 1772-1850)
* [ ] Step 2.4.5: Add Anthropic provider tests
  * Details: .copilot-tracking/details/2026-02-02-golang-epic2-providers-details.md (Lines 1852-1910)
* [ ] Validate Feature 2.4
  * Run `go test ./providers/anthropic/...` with integration tests

### [ ] Feature 2.5: AWS Bedrock Provider Package

<!-- parallelizable: true -->

* [ ] Step 2.5.1: Create Bedrock client structure and options
  * Details: .copilot-tracking/details/2026-02-02-golang-epic2-providers-details.md (Lines 1915-2000)
* [ ] Step 2.5.2: Implement AWS credential resolution
  * Details: .copilot-tracking/details/2026-02-02-golang-epic2-providers-details.md (Lines 2002-2080)
* [ ] Step 2.5.3: Implement Converse API integration
  * Details: .copilot-tracking/details/2026-02-02-golang-epic2-providers-details.md (Lines 2082-2170)
* [ ] Step 2.5.4: Implement streaming with chunk parsing
  * Details: .copilot-tracking/details/2026-02-02-golang-epic2-providers-details.md (Lines 2172-2260)
* [ ] Step 2.5.5: Add Bedrock provider tests
  * Details: .copilot-tracking/details/2026-02-02-golang-epic2-providers-details.md (Lines 2262-2320)
* [ ] Validate Feature 2.5
  * Run `go test ./providers/bedrock/...` with integration tests

### [ ] Feature 2.6: Ollama Provider Package

<!-- parallelizable: true -->

* [ ] Step 2.6.1: Create Ollama client structure and options
  * Details: .copilot-tracking/details/2026-02-02-golang-epic2-providers-details.md (Lines 2325-2400)
* [ ] Step 2.6.2: Implement HTTP API integration
  * Details: .copilot-tracking/details/2026-02-02-golang-epic2-providers-details.md (Lines 2402-2480)
* [ ] Step 2.6.3: Implement NDJSON streaming
  * Details: .copilot-tracking/details/2026-02-02-golang-epic2-providers-details.md (Lines 2482-2560)
* [ ] Step 2.6.4: Add Ollama provider tests
  * Details: .copilot-tracking/details/2026-02-02-golang-epic2-providers-details.md (Lines 2562-2620)
* [ ] Validate Feature 2.6
  * Run `go test ./providers/ollama/...` with integration tests

### [ ] Feature 2.7: OpenTelemetry Observability Package

<!-- parallelizable: false -->

* [ ] Step 2.7.1: Define GenAI semantic conventions
  * Details: .copilot-tracking/details/2026-02-02-golang-epic2-providers-details.md (Lines 2625-2720)
* [ ] Step 2.7.2: Implement tracing instrumentation
  * Details: .copilot-tracking/details/2026-02-02-golang-epic2-providers-details.md (Lines 2722-2820)
* [ ] Step 2.7.3: Implement metrics collection
  * Details: .copilot-tracking/details/2026-02-02-golang-epic2-providers-details.md (Lines 2822-2900)
* [ ] Step 2.7.4: Create instrumented client wrapper
  * Details: .copilot-tracking/details/2026-02-02-golang-epic2-providers-details.md (Lines 2902-2980)
* [ ] Step 2.7.5: Add observability tests
  * Details: .copilot-tracking/details/2026-02-02-golang-epic2-providers-details.md (Lines 2982-3040)
* [ ] Validate Feature 2.7
  * Run `go test ./observability/...` with telemetry verification

### [ ] Feature 2.8: ChatClientAgent Implementation

<!-- parallelizable: false -->
<!-- Depends on: Features 2.1-2.7 -->

* [ ] Step 2.8.1: Create ChatClientAgent structure
  * Details: .copilot-tracking/details/2026-02-02-golang-epic2-providers-details.md (Lines 3045-3140)
* [ ] Step 2.8.2: Implement agent options and builder
  * Details: .copilot-tracking/details/2026-02-02-golang-epic2-providers-details.md (Lines 3142-3240)
* [ ] Step 2.8.3: Implement Run and RunStream methods
  * Details: .copilot-tracking/details/2026-02-02-golang-epic2-providers-details.md (Lines 3242-3350)
* [ ] Step 2.8.4: Implement automatic tool invocation loop
  * Details: .copilot-tracking/details/2026-02-02-golang-epic2-providers-details.md (Lines 3352-3450)
* [ ] Step 2.8.5: Implement session management
  * Details: .copilot-tracking/details/2026-02-02-golang-epic2-providers-details.md (Lines 3452-3530)
* [ ] Step 2.8.6: Add ChatClientAgent tests
  * Details: .copilot-tracking/details/2026-02-02-golang-epic2-providers-details.md (Lines 3532-3600)
* [ ] Validate Feature 2.8
  * Run `go test ./chatagent/...` with end-to-end tests

### [ ] Final Validation Phase

<!-- parallelizable: false -->

* [ ] Step F.1: Run full Epic 2 validation
  * Execute `go build ./...` for all packages
  * Execute `go test -cover ./...` for all tests
  * Execute `go vet ./...` and `golangci-lint run`
* [ ] Step F.2: Fix minor validation issues
  * Iterate on lint errors and test failures
  * Apply fixes directly when corrections are straightforward
* [ ] Step F.3: Run integration tests
  * Test OpenAI provider with live API
  * Test Azure OpenAI with Azure credentials
  * Test other providers in available environments
* [ ] Step F.4: Report blocking issues
  * Document issues requiring additional research
  * Provide next steps and recommended planning

## Dependencies

* Go 1.22+ with generic type constraints
* `github.com/sashabaranov/go-openai` - OpenAI Go client
* `github.com/Azure/azure-sdk-for-go/sdk/azidentity` - Azure authentication
* `github.com/Azure/azure-sdk-for-go/sdk/ai/azopenai` - Azure OpenAI SDK
* `github.com/aws/aws-sdk-go-v2` - AWS SDK for Bedrock
* `github.com/liushuangls/go-anthropic/v2` - Anthropic Go client
* `go.opentelemetry.io/otel` - OpenTelemetry SDK

## Success Criteria

* All provider packages implement `chat.Client` interface
* Tool system supports function tools and hosted tools
* OpenTelemetry instrumentation follows GenAI semantic conventions
* 90%+ test coverage across all packages
* Integration tests pass for each provider
* API patterns align with .NET and Python implementations
