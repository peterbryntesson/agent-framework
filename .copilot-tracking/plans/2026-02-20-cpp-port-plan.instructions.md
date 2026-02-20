---
applyTo: '.copilot-tracking/changes/2026-02-20-cpp-port-changes.md'
---
<!-- markdownlint-disable-file -->
# Implementation Plan: C++ Port of Agent Framework

## Overview

Full C++ reimplementation of the Microsoft Agent Framework with feature parity across all .NET (27 projects, ~71K LOC) and Python (20 packages, ~79K LOC) components, targeting ~44K C++ LOC delivered over 13 phases in 22 weeks.

## Objectives

* Deliver a complete C++ agent framework with the same capabilities as the .NET and Python implementations
* Support C++20 across MSVC, GCC 11+, and Clang 14+ on Windows, Linux, and macOS
* Achieve full interop with .NET/Python implementations via OpenAI, A2A, and AG-UI hosting protocols
* Provide idiomatic C++ APIs (coroutines, concepts, ranges) while maintaining architectural parity

## Context Summary

### Project Files

* dotnet/src/ (27 projects, ~71K LOC) — .NET reference implementation
* python/packages/ (20 packages, ~79K LOC) — Python reference implementation
* schemas/durable-agent-entity-state.json — Durable agent state schema

### References

* .copilot-tracking/research/2026-02-20-cpp-port-implementation-plan.md — Full research with per-phase deliverables, code snippets, and gate criteria
* .copilot-tracking/research/2026-02-20-cpp-port-research.md — Technology research and decisions
* .copilot-tracking/subagent/2026-02-20/dotnet-structure-analysis.md — .NET project inventory and LOC analysis
* .copilot-tracking/subagent/2026-02-20/python-structure-analysis.md — Python package inventory and dependency graph
* .copilot-tracking/subagent/2026-02-20/hosting-protocols-analysis.md — Protocol specifications for A2A, AG-UI, OpenAI hosting

### Standards References

* #file:../../.github/copilot-instructions.md — Repository contribution guidelines
* #file:../../.github/instructions/durabletask-dotnet.instructions.md — Durable task patterns (reference for C++ port)

### Technology Decisions

| Decision | Choice | Rationale |
|---|---|---|
| C++ Standard | C++20 minimum | Coroutines, concepts, ranges, `std::format`, `std::stop_token` |
| Build system | CMake 3.25+ with vcpkg manifest mode | Industry standard, FILE_SET support |
| JSON library | glaze 4.x | Tagged variant polymorphism, JSON Schema gen, 10x faster than nlohmann |
| HTTP / Async | Boost.Beast + Boost.Asio | Unified async model, no second event loop |
| Testing | Google Test + gmock 1.15+ | Industry standard |
| Expression engine | Lua/sol2 with PowerFx compatibility shim | Research decision AD-7 |
| CI/CD | GitHub Actions | Matrix builds MSVC, GCC 11+, Clang 14+ on Windows/Linux/macOS |

## Implementation Checklist

### [ ] Implementation Phase 0: Build Infrastructure (Week 1)

<!-- parallelizable: false -->

Establish CMake project skeleton, vcpkg manifest, CI pipeline, and coding standards.

* [ ] Step 0.1: Create top-level CMake structure with presets
  * Details: .copilot-tracking/details/2026-02-20-cpp-port-details.md (Lines 17-42)
* [ ] Step 0.2: Create vcpkg manifest with all dependencies and feature flags
  * Details: .copilot-tracking/details/2026-02-20-cpp-port-details.md (Lines 44-68)
* [ ] Step 0.3: Scaffold all 27 library directories with CMakeLists.txt stubs
  * Details: .copilot-tracking/details/2026-02-20-cpp-port-details.md (Lines 70-123)
* [ ] Step 0.4: Create shared CMake modules (find_package support, compiler warnings, dependencies)
  * Details: .copilot-tracking/details/2026-02-20-cpp-port-details.md (Lines 125-145)
* [ ] Step 0.5: Create GitHub Actions CI workflow with compiler/OS matrix
  * Details: .copilot-tracking/details/2026-02-20-cpp-port-details.md (Lines 147-175)
* [ ] Step 0.6: Add code quality tooling (clang-tidy, clang-format, sanitizer presets)
  * Details: .copilot-tracking/details/2026-02-20-cpp-port-details.md (Lines 177-199)
* [ ] Step 0.7: Add copyright header template and check script
  * Details: .copilot-tracking/details/2026-02-20-cpp-port-details.md (Lines 201-210)
* [ ] Step 0.8: Write cpp/README.md with build instructions and architecture overview
  * Details: .copilot-tracking/details/2026-02-20-cpp-port-details.md (Lines 212-224)
* [ ] Step 0.9: Validate phase — CI green on all matrix entries
  * `cmake --preset release` succeeds on MSVC, GCC 11, Clang 14
  * `ctest` runs on all 3 platforms
  * vcpkg dependencies resolve without manual intervention
  * clang-format and clang-tidy pass on scaffold files

### [ ] Implementation Phase 1: Async Primitives (Weeks 2-3)

<!-- parallelizable: false -->

Build foundational async types: `Task<T>`, `AsyncGenerator<T>`, SSE stream parser, HTTP client wrapper.

* [ ] Step 1.1: Implement `Task<T>` alias, `when_all()`, `CancellationToken` wrappers
  * Details: .copilot-tracking/details/2026-02-20-cpp-port-details.md (Lines 228-250)
* [ ] Step 1.2: Implement `AsyncGenerator<T>` C++20 coroutine type
  * Details: .copilot-tracking/details/2026-02-20-cpp-port-details.md (Lines 252-283)
* [ ] Step 1.3: Implement `AsyncChannel<T>` and `AsyncSemaphore`
  * Details: .copilot-tracking/details/2026-02-20-cpp-port-details.md (Lines 285-305)
* [ ] Step 1.4: Implement HTTP client wrapper over Boost.Beast
  * Details: .copilot-tracking/details/2026-02-20-cpp-port-details.md (Lines 307-340)
* [ ] Step 1.5: Implement SSE stream parser
  * Details: .copilot-tracking/details/2026-02-20-cpp-port-details.md (Lines 342-368)
* [ ] Step 1.6: Write unit tests for all async primitives
  * Details: .copilot-tracking/details/2026-02-20-cpp-port-details.md (Lines 370-395)
* [ ] Step 1.7: Validate phase — no sanitizer warnings (ASAN, TSAN), all compilers pass

### [ ] Implementation Phase 2: Content and Message Model (Weeks 3-4)

<!-- parallelizable: false -->

Implement the ~25 types from `Microsoft.Extensions.AI` that form the content/message foundation.

* [ ] Step 2.1: Implement 9 content variant types with glaze `$type` discriminated union
  * Details: .copilot-tracking/details/2026-02-20-cpp-port-details.md (Lines 399-435)
* [ ] Step 2.2: Implement `ChatMessage`, `ChatRole`, `ChatResponse`, `ChatResponseUpdate`
  * Details: .copilot-tracking/details/2026-02-20-cpp-port-details.md (Lines 437-468)
* [ ] Step 2.3: Implement `ChatOptions`, `UsageDetails`, `FinishReason`, `AdditionalProperties`
  * Details: .copilot-tracking/details/2026-02-20-cpp-port-details.md (Lines 470-497)
* [ ] Step 2.4: Implement `AgentResponse<T>` and `AgentResponseUpdate`
  * Details: .copilot-tracking/details/2026-02-20-cpp-port-details.md (Lines 499-517)
* [ ] Step 2.5: Write JSON round-trip serialization tests for all types
  * Details: .copilot-tracking/details/2026-02-20-cpp-port-details.md (Lines 519-540)
* [ ] Step 2.6: Validate phase — cross-compatibility with .NET-produced JSON, ASAN clean

### [ ] Implementation Phase 3: Tool System (Week 5)

<!-- parallelizable: false -->

Implement AI tool/function abstraction, factory for creating tools from C++ callables, and JSON Schema generation.

* [ ] Step 3.1: Implement `AITool` base class and `AIFunction` with async invocation
  * Details: .copilot-tracking/details/2026-02-20-cpp-port-details.md (Lines 544-565)
* [ ] Step 3.2: Implement `make_function<>()` template factory with both lambda and struct-based parameter styles
  * Details: .copilot-tracking/details/2026-02-20-cpp-port-details.md (Lines 567-607)
* [ ] Step 3.3: Implement JSON Schema generation via `glz::write_json_schema<T>()`
  * Details: .copilot-tracking/details/2026-02-20-cpp-port-details.md (Lines 609-629)
* [ ] Step 3.4: Implement `DelegatingAIFunction` and `AGENT_TOOL` convenience macro
  * Details: .copilot-tracking/details/2026-02-20-cpp-port-details.md (Lines 631-651)
* [ ] Step 3.5: Write unit tests for tool factory, schema generation, invocation, error handling
  * Details: .copilot-tracking/details/2026-02-20-cpp-port-details.md (Lines 653-673)
* [ ] Step 3.6: Validate phase — generated JSON Schema matches OpenAI function calling format

### [ ] Implementation Phase 4: Core Agent Abstractions (Weeks 5-7)

<!-- parallelizable: false -->

Implement the full agent class hierarchy, session management, middleware pipeline, context providers, and `ChatClientAgent` (~1200 C++ LOC).

* [ ] Step 4.1: Implement `IChatClient` interface and `DelegatingChatClient`
  * Details: .copilot-tracking/details/2026-02-20-cpp-port-details.md (Lines 677-701)
* [ ] Step 4.2: Implement `AIAgent` abstract base with 4 `run_async` overloads
  * Details: .copilot-tracking/details/2026-02-20-cpp-port-details.md (Lines 703-740)
* [ ] Step 4.3: Implement `DelegatingAIAgent` decorator base
  * Details: .copilot-tracking/details/2026-02-20-cpp-port-details.md (Lines 742-758)
* [ ] Step 4.4: Implement `AgentSession` hierarchy (`InMemoryAgentSession`, `ServiceIdAgentSession`)
  * Details: .copilot-tracking/details/2026-02-20-cpp-port-details.md (Lines 760-787)
* [ ] Step 4.5: Implement `ChatHistoryProvider` and `AIContextProvider` with two-phase lifecycle
  * Details: .copilot-tracking/details/2026-02-20-cpp-port-details.md (Lines 789-818)
* [ ] Step 4.6: Implement three-level middleware pipeline (agent, chat, function)
  * Details: .copilot-tracking/details/2026-02-20-cpp-port-details.md (Lines 820-853)
* [ ] Step 4.7: Implement `ChatClientAgent` — session lifecycle, options merging, function invocation loop, streaming
  * Details: .copilot-tracking/details/2026-02-20-cpp-port-details.md (Lines 855-907)
* [ ] Step 4.8: Implement `ServiceCollection` (minimal DI) and `AIAgentBuilder`
  * Details: .copilot-tracking/details/2026-02-20-cpp-port-details.md (Lines 909-940)
* [ ] Step 4.9: Write comprehensive unit tests with mock `IChatClient`
  * Details: .copilot-tracking/details/2026-02-20-cpp-port-details.md (Lines 942-970)
* [ ] Step 4.10: Validate phase — all 3 compilers pass, middleware onion model verified

### [ ] Implementation Phase 5: First Provider — OpenAI (Weeks 7-9)

<!-- parallelizable: false -->

Implement OpenAI Chat Completions, Responses API, and Assistants API to validate the full stack end-to-end.

* [ ] Step 5.1: Implement OpenAI request/response model structs (~50 structs with glaze serialization)
  * Details: .copilot-tracking/details/2026-02-20-cpp-port-details.md (Lines 974-997)
* [ ] Step 5.2: Implement `OpenAIChatClient` for Chat Completions (non-streaming and streaming)
  * Details: .copilot-tracking/details/2026-02-20-cpp-port-details.md (Lines 999-1027)
* [ ] Step 5.3: Implement OpenAI Responses API client with 19 SSE event types
  * Details: .copilot-tracking/details/2026-02-20-cpp-port-details.md (Lines 1029-1053)
* [ ] Step 5.4: Implement OpenAI Assistants API client (thread/message/run lifecycle)
  * Details: .copilot-tracking/details/2026-02-20-cpp-port-details.md (Lines 1055-1076)
* [ ] Step 5.5: Implement `AzureOpenAIChatClient` variant with Azure AD auth
  * Details: .copilot-tracking/details/2026-02-20-cpp-port-details.md (Lines 1078-1098)
* [ ] Step 5.6: Implement tool/function calling serialization and auto-invoke loop
  * Details: .copilot-tracking/details/2026-02-20-cpp-port-details.md (Lines 1100-1120)
* [ ] Step 5.7: Create extension factory functions (`openai_as_chat_client`, `openai_as_agent`)
  * Details: .copilot-tracking/details/2026-02-20-cpp-port-details.md (Lines 1122-1137)
* [ ] Step 5.8: Write integration tests (live API gated by env vars) and mock-server unit tests
  * Details: .copilot-tracking/details/2026-02-20-cpp-port-details.md (Lines 1139-1157)
* [ ] Step 5.9: Validate phase — end-to-end: HTTP → SSE → IChatClient → ChatClientAgent → AgentResponse

### [ ] Implementation Phase 6A: Azure AI Provider (Weeks 9-10.5)

<!-- parallelizable: true -->

* [ ] Step 6A.1: Implement `AzureAIChatClient` with REST calls and tool schema transforms
  * Details: .copilot-tracking/details/2026-02-20-cpp-port-details.md (Lines 1161-1185)
* [ ] Step 6A.2: Implement `AzureAIAgentSession` for server-managed threads
  * Details: .copilot-tracking/details/2026-02-20-cpp-port-details.md (Lines 1187-1202)
* [ ] Step 6A.3: Integrate `azure-identity-cpp` for `DefaultAzureCredential`
  * Details: .copilot-tracking/details/2026-02-20-cpp-port-details.md (Lines 1204-1218)
* [ ] Step 6A.4: Write integration tests against Azure AI endpoint
  * Details: .copilot-tracking/details/2026-02-20-cpp-port-details.md (Lines 1220-1230)

### [ ] Implementation Phase 6B: Anthropic Provider (Week 10.5-11)

<!-- parallelizable: true -->

* [ ] Step 6B.1: Implement `AnthropicChatClient` with Messages API and streaming
  * Details: .copilot-tracking/details/2026-02-20-cpp-port-details.md (Lines 1234-1258)
* [ ] Step 6B.2: Implement Anthropic tool_use/tool_result content block mapping
  * Details: .copilot-tracking/details/2026-02-20-cpp-port-details.md (Lines 1260-1275)
* [ ] Step 6B.3: Write integration and mock tests
  * Details: .copilot-tracking/details/2026-02-20-cpp-port-details.md (Lines 1277-1287)

### [ ] Implementation Phase 6C: Copilot Studio and GitHub Copilot Providers (Weeks 11-12)

<!-- parallelizable: true -->

* [ ] Step 6C.1: Implement `CopilotStudioAgent` with DirectLine protocol
  * Details: .copilot-tracking/details/2026-02-20-cpp-port-details.md (Lines 1291-1314)
* [ ] Step 6C.2: Implement `GitHubCopilotAgent` with channel-based streaming
  * Details: .copilot-tracking/details/2026-02-20-cpp-port-details.md (Lines 1316-1336)
* [ ] Step 6C.3: Write mock tests for both providers
  * Details: .copilot-tracking/details/2026-02-20-cpp-port-details.md (Lines 1338-1348)

### [ ] Implementation Phase 7: Cross-Cutting Agents (Week 12)

<!-- parallelizable: false -->

Implement decorator agents (`LoggingAgent`, `OpenTelemetryAgent`, etc.) and builder API enhancements.

* [ ] Step 7.1: Implement `LoggingAgent` via spdlog
  * Details: .copilot-tracking/details/2026-02-20-cpp-port-details.md (Lines 1352-1368)
* [ ] Step 7.2: Implement `OpenTelemetryAgent` with GenAI semantic conventions
  * Details: .copilot-tracking/details/2026-02-20-cpp-port-details.md (Lines 1370-1396)
* [ ] Step 7.3: Implement `FunctionInvocationDelegatingAgent` and `AnonymousDelegatingAIAgent`
  * Details: .copilot-tracking/details/2026-02-20-cpp-port-details.md (Lines 1398-1416)
* [ ] Step 7.4: Implement `AIHostAgent` and builder API enhancements
  * Details: .copilot-tracking/details/2026-02-20-cpp-port-details.md (Lines 1418-1436)
* [ ] Step 7.5: Write unit tests — decorator chain ordering, telemetry spans, logging output
  * Details: .copilot-tracking/details/2026-02-20-cpp-port-details.md (Lines 1438-1455)
* [ ] Step 7.6: Validate phase — OTel spans exportable via OTLP (in-memory exporter)

### [ ] Implementation Phase 8A: HTTP Server Foundation (Week 13)

<!-- parallelizable: false -->

* [ ] Step 8A.1: Implement `HttpRouter` with path parameters and middleware pipeline
  * Details: .copilot-tracking/details/2026-02-20-cpp-port-details.md (Lines 1459-1480)
* [ ] Step 8A.2: Implement `SseWriter` for chunked SSE response streaming
  * Details: .copilot-tracking/details/2026-02-20-cpp-port-details.md (Lines 1482-1500)
* [ ] Step 8A.3: Implement `AgentHost` server and `AgentHostBuilder`
  * Details: .copilot-tracking/details/2026-02-20-cpp-port-details.md (Lines 1502-1527)
* [ ] Step 8A.4: Implement server middleware (CORS, logging, error handling, auth)
  * Details: .copilot-tracking/details/2026-02-20-cpp-port-details.md (Lines 1529-1545)
* [ ] Step 8A.5: Write tests for route matching, SSE round-trip, concurrent connections
  * Details: .copilot-tracking/details/2026-02-20-cpp-port-details.md (Lines 1547-1562)

### [ ] Implementation Phase 8B: OpenAI-Compatible Hosting (Weeks 13-14)

<!-- parallelizable: false -->

* [ ] Step 8B.1: Implement Chat Completions endpoint with request conversion and streaming
  * Details: .copilot-tracking/details/2026-02-20-cpp-port-details.md (Lines 1566-1592)
* [ ] Step 8B.2: Implement Responses API endpoints (5 endpoints, 19 SSE event types)
  * Details: .copilot-tracking/details/2026-02-20-cpp-port-details.md (Lines 1594-1622)
* [ ] Step 8B.3: Implement Conversations API endpoints (8 endpoints with pagination)
  * Details: .copilot-tracking/details/2026-02-20-cpp-port-details.md (Lines 1624-1644)
* [ ] Step 8B.4: Write interop tests — C++ server validated by .NET/Python OpenAI client
  * Details: .copilot-tracking/details/2026-02-20-cpp-port-details.md (Lines 1646-1658)

### [ ] Implementation Phase 8C: A2A Protocol Hosting (Week 15)

<!-- parallelizable: true -->

* [ ] Step 8C.1: Implement Agent Card discovery and JSON-RPC 2.0 dispatcher
  * Details: .copilot-tracking/details/2026-02-20-cpp-port-details.md (Lines 1662-1688)
* [ ] Step 8C.2: Implement task lifecycle manager and SSE streaming for `tasks/sendSubscribe`
  * Details: .copilot-tracking/details/2026-02-20-cpp-port-details.md (Lines 1690-1710)
* [ ] Step 8C.3: Implement `A2AAgent` remote proxy with continuation token support
  * Details: .copilot-tracking/details/2026-02-20-cpp-port-details.md (Lines 1712-1729)
* [ ] Step 8C.4: Write interop tests — C++ A2A server ↔ .NET A2A client
  * Details: .copilot-tracking/details/2026-02-20-cpp-port-details.md (Lines 1731-1742)

### [ ] Implementation Phase 8D: AG-UI Protocol Hosting (Weeks 15-16)

<!-- parallelizable: true -->

* [ ] Step 8D.1: Implement AG-UI SSE endpoint with 12 event types
  * Details: .copilot-tracking/details/2026-02-20-cpp-port-details.md (Lines 1746-1770)
* [ ] Step 8D.2: Implement `AGUIChatClient` — wraps tool execution with AG-UI events
  * Details: .copilot-tracking/details/2026-02-20-cpp-port-details.md (Lines 1772-1790)
* [ ] Step 8D.3: Write interop tests — C++ AG-UI server ↔ CopilotKit client
  * Details: .copilot-tracking/details/2026-02-20-cpp-port-details.md (Lines 1792-1802)

### [ ] Implementation Phase 9A: Workflow Graph Executor (Weeks 16-17)

<!-- parallelizable: false -->

* [ ] Step 9A.1: Implement `WorkflowNode`, edge types (Direct, Conditional, FanOut, FanIn)
  * Details: .copilot-tracking/details/2026-02-20-cpp-port-details.md (Lines 1806-1838)
* [ ] Step 9A.2: Implement `WorkflowExecutor` graph traversal engine
  * Details: .copilot-tracking/details/2026-02-20-cpp-port-details.md (Lines 1840-1869)
* [ ] Step 9A.3: Implement builder APIs and checkpoint interface
  * Details: .copilot-tracking/details/2026-02-20-cpp-port-details.md (Lines 1871-1897)
* [ ] Step 9A.4: Add OpenTelemetry integration (per-node spans)
  * Details: .copilot-tracking/details/2026-02-20-cpp-port-details.md (Lines 1899-1912)
* [ ] Step 9A.5: Write unit tests — linear, branching, fan-out/in, cycles, checkpointing
  * Details: .copilot-tracking/details/2026-02-20-cpp-port-details.md (Lines 1914-1933)

### [ ] Implementation Phase 9B: Declarative Workflows and Lua Engine (Weeks 17-19)

<!-- parallelizable: false -->

* [ ] Step 9B.1: Implement YAML parsing of agent and workflow definitions
  * Details: .copilot-tracking/details/2026-02-20-cpp-port-details.md (Lines 1937-1960)
* [ ] Step 9B.2: Implement Lua expression engine via sol2
  * Details: .copilot-tracking/details/2026-02-20-cpp-port-details.md (Lines 1962-1990)
* [ ] Step 9B.3: Implement PowerFx compatibility shim (regex-based translator)
  * Details: .copilot-tracking/details/2026-02-20-cpp-port-details.md (Lines 1992-2020)
* [ ] Step 9B.4: Implement variable scoping (5 scope levels) and `WorkflowAgentProvider`
  * Details: .copilot-tracking/details/2026-02-20-cpp-port-details.md (Lines 2022-2046)
* [ ] Step 9B.5: Implement declarative workflow builder (YAML → `WorkflowExecutor`)
  * Details: .copilot-tracking/details/2026-02-20-cpp-port-details.md (Lines 2048-2068)
* [ ] Step 9B.6: Implement build-time code generation (CMake custom command, replaces Roslyn source generators)
  * Details: .copilot-tracking/details/2026-02-20-cpp-port-details.md (Lines 2070-2090)
* [ ] Step 9B.7: Write unit tests — YAML parsing, expressions, scoping, full declarative execution
  * Details: .copilot-tracking/details/2026-02-20-cpp-port-details.md (Lines 2092-2112)
* [ ] Step 9B.8: Validate phase — PowerFx expressions and Lua native expressions evaluate correctly

### [ ] Implementation Phase 10: Storage and Integrations (Weeks 19-20)

<!-- parallelizable: false -->

* [ ] Step 10.1: Implement in-memory stores (ChatHistory, Checkpoint, AgentSession)
  * Details: .copilot-tracking/details/2026-02-20-cpp-port-details.md (Lines 2116-2132)
* [ ] Step 10.2: Implement Cosmos DB storage provider via REST API
  * Details: .copilot-tracking/details/2026-02-20-cpp-port-details.md (Lines 2134-2157)
* [ ] Step 10.3: Implement Redis storage provider via hiredis
  * Details: .copilot-tracking/details/2026-02-20-cpp-port-details.md (Lines 2159-2180)
* [ ] Step 10.4: Implement Mem0 context provider and Purview DLP middleware
  * Details: .copilot-tracking/details/2026-02-20-cpp-port-details.md (Lines 2182-2210)
* [ ] Step 10.5: Implement MCP tool serving integration
  * Details: .copilot-tracking/details/2026-02-20-cpp-port-details.md (Lines 2212-2230)
* [ ] Step 10.6: Write tests — concurrent store access, storage round-trips (Cosmos/Redis gated by env vars)
  * Details: .copilot-tracking/details/2026-02-20-cpp-port-details.md (Lines 2232-2250)
* [ ] Step 10.7: Validate phase — in-memory concurrent safety, MCP tool discovery/invocation

### [ ] Implementation Phase 11: Durable Tasks (Weeks 20-21)

<!-- parallelizable: false -->

* [ ] Step 11.1: Generate C++ gRPC stubs from durabletask-protobuf and implement `DurableTaskClient`
  * Details: .copilot-tracking/details/2026-02-20-cpp-port-details.md (Lines 2254-2279)
* [ ] Step 11.2: Implement `DurableAIAgent` and `DurableAIAgentProxy`
  * Details: .copilot-tracking/details/2026-02-20-cpp-port-details.md (Lines 2281-2308)
* [ ] Step 11.3: Implement entity state schema matching `schemas/durable-agent-entity-state.json`
  * Details: .copilot-tracking/details/2026-02-20-cpp-port-details.md (Lines 2310-2325)
* [ ] Step 11.4: Implement standalone `AgentDaemon` (HTTP → durable task sidecar via gRPC)
  * Details: .copilot-tracking/details/2026-02-20-cpp-port-details.md (Lines 2327-2351)
* [ ] Step 11.5: Write tests — mock gRPC server, entity signal/query round-trip, daemon integration
  * Details: .copilot-tracking/details/2026-02-20-cpp-port-details.md (Lines 2353-2370)
* [ ] Step 11.6: Validate phase — entity state matches JSON schema, sidecar communication works

### [ ] Implementation Phase 12: Developer UI and Samples (Weeks 21-22)

<!-- parallelizable: false -->

* [ ] Step 12.1: Implement Developer UI SPA embedding with REST API
  * Details: .copilot-tracking/details/2026-02-20-cpp-port-details.md (Lines 2374-2395)
* [ ] Step 12.2: Create 4 Getting Started samples (Step1 through Step4)
  * Details: .copilot-tracking/details/2026-02-20-cpp-port-details.md (Lines 2397-2426)
* [ ] Step 12.3: Create Hosted Agent, Durable Agent, and Declarative Workflow samples
  * Details: .copilot-tracking/details/2026-02-20-cpp-port-details.md (Lines 2428-2456)
* [ ] Step 12.4: Write sample READMEs and top-level cpp/samples/README.md
  * Details: .copilot-tracking/details/2026-02-20-cpp-port-details.md (Lines 2458-2474)
* [ ] Step 12.5: End-to-end validation — run all samples, verify output
  * Details: .copilot-tracking/details/2026-02-20-cpp-port-details.md (Lines 2476-2490)

### [ ] Implementation Phase 13: Final Validation

<!-- parallelizable: false -->

* [ ] Step 13.1: Run full project validation
  * Run `cmake --build --preset release` for all targets
  * Run `ctest --preset release` for complete test suite
  * Run clang-tidy and clang-format checks on all source files
  * Run ASAN, TSAN, UBSAN sanitizer passes
* [ ] Step 13.2: Fix minor validation issues
  * Iterate on lint errors and build warnings
  * Apply fixes directly when corrections are straightforward
* [ ] Step 13.3: Cross-language interop validation
  * C++ OpenAI-compatible server ↔ .NET/Python OpenAI client
  * C++ A2A server ↔ .NET A2A client
  * C++ AG-UI server ↔ CopilotKit client
* [ ] Step 13.4: Report blocking issues
  * Document issues requiring additional research
  * Provide next steps and recommended planning
  * Avoid large-scale fixes within this phase

## Dependencies

* CMake 3.25+
* vcpkg (manifest mode)
* Boost 1.84+ (Asio, Beast)
* glaze 4.x
* nlohmann-json (for interop/fallback)
* spdlog + fmt
* opentelemetry-cpp
* OpenSSL
* Google Test + gmock 1.15+
* yaml-cpp (Phase 9)
* sol2 + Lua 5.4 (Phase 9)
* gRPC + protobuf (Phase 11)
* hiredis / redis-plus-plus (Phase 10)
* azure-identity-cpp (Phase 5/6)

## Success Criteria

* All 27 C++ library targets build on MSVC, GCC 11+, and Clang 14+ across Windows, Linux, and macOS
* Complete unit test coverage for all public API methods across all libraries
* All hosting protocols interoperate with .NET and Python implementations
* Zero sanitizer warnings (ASAN, TSAN, UBSAN)
* All Getting Started samples compile and run successfully on all 3 platforms
* CI pipeline green for all valid matrix entries
* API naming follows the project naming conventions (PascalCase classes, snake_case methods)
* All source files carry the Microsoft copyright header
