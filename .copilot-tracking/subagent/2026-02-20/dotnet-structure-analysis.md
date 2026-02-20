# .NET Agent Framework — Structure Analysis for C++ Port

> Generated: 2026-02-20
> Purpose: Inform a C++ port implementation plan by documenting the full .NET project structure,
> dependencies, public API surface, and complexity estimates.

---

## 1. Repository Overview

- **Framework purpose:** Building AI agents with pluggable model providers, workflows, hosting, and protocol support (A2A, AG-UI, OpenAI, MCP).
- **Source root:** `dotnet/src/` — 27 project directories + 1 shared code folder.
- **Total .cs files in src:** ~1,167
- **Total lines of C# in src:** ~71,168
- **Solution file:** `dotnet/agent-framework-dotnet.slnx` (465 lines, XML-based slnx format).

---

## 2. Project Inventory — `dotnet/src/`

### 2.1 Core / Abstractions Layer

| Project | Files | LOC | Public Types | Role (CMakeLists Equivalent) |
|---|---|---|---|---|
| **Microsoft.Agents.AI.Abstractions** | 30 | 2,827 | 23 | Core abstractions library — defines `AIAgent`, `AgentSession`, `AgentResponse`, `ChatHistoryProvider`, etc. No project dependencies. Foundation for all other projects. |
| **Microsoft.Agents.AI** | 37 | 4,065 | 25 | Core implementation — `ChatClientAgent`, OpenTelemetry instrumentation, text search, memory providers. Depends on Abstractions. |
| **LegacySupport** | 12 | 559 | 0 | Polyfill/shim types for older .NET TFMs (shared via file links, not a compiled project). |
| **Shared** | 17 | 2,024 | 0 | Shared helpers for tests/samples (not a NuGet package). |

### 2.2 Model Provider Libraries

| Project | Files | LOC | Public Types | Role |
|---|---|---|---|---|
| **Microsoft.Agents.AI.OpenAI** | 23 | 1,007 | 5 | OpenAI ChatClient/Responses/Assistants integration. Depends on AI. |
| **Microsoft.Agents.AI.AzureAI** | 18 | 1,020 | 0 (internal) | Azure AI Foundry project integration. Depends on AI. |
| **Microsoft.Agents.AI.AzureAI.Persistent** | 16 | 552 | 1 | Azure AI Agents (persistent) integration. Depends on AI. |
| **Microsoft.Agents.AI.Anthropic** | 18 | 358 | 2 | Anthropic Claude integration. Depends on AI. |
| **Microsoft.Agents.AI.CopilotStudio** | 13 | 335 | 2 | Microsoft Copilot Studio agent. Depends on Abstractions. |
| **Microsoft.Agents.AI.GitHub.Copilot** | 10 | 646 | 3 | GitHub Copilot SDK agent. Depends on Abstractions. |

### 2.3 Protocol / Interop Libraries

| Project | Files | LOC | Public Types | Role |
|---|---|---|---|---|
| **Microsoft.Agents.AI.A2A** | 24 | 937 | 5 | Agent-to-Agent (A2A) protocol client. Depends on Abstractions. |
| **Microsoft.Agents.AI.AGUI** | 44 | 1,900 | 1 | AG-UI protocol ChatClient implementation. Depends on Abstractions. |

### 2.4 Workflow Engine

| Project | Files | LOC | Public Types | Role |
|---|---|---|---|---|
| **Microsoft.Agents.AI.Workflows** | 200 | 12,915 | 104 | **Largest core project.** Workflow engine with executors, edges, state, streaming, checkpoints, group chat. Depends on AI + Abstractions. |
| **Microsoft.Agents.AI.Workflows.Declarative** | 140 | 17,417 | 38 | **Largest by LOC.** YAML/declarative workflow definitions, formula evaluation (PowerFx), code generation. Depends on Workflows. |
| **Microsoft.Agents.AI.Workflows.Declarative.AzureAI** | 11 | 367 | 1 | Azure AI agent provider for declarative workflows. Depends on AzureAI + Workflows.Declarative. |
| **Microsoft.Agents.AI.Workflows.Generators** | 23 | 1,656 | 2 | Roslyn source generator for executor route discovery. Standalone (no project refs). |

### 2.5 Hosting / ASP.NET Integration

| Project | Files | LOC | Public Types | Role |
|---|---|---|---|---|
| **Microsoft.Agents.AI.Hosting** | 24 | 771 | 12 | DI/hosting abstractions — `IHostedAgentBuilder`, session stores. Depends on AI + Abstractions + Workflows. |
| **Microsoft.Agents.AI.Hosting.OpenAI** | 93 | 10,668 | 2 | OpenAI-compatible HTTP API hosting (chat completions endpoint). Depends on Abstractions + Hosting. |
| **Microsoft.Agents.AI.Hosting.A2A** | 10 | 289 | 1 | A2A protocol hosting adapter. Depends on Abstractions + Hosting. |
| **Microsoft.Agents.AI.Hosting.A2A.AspNetCore** | 7 | 307 | 1 | ASP.NET Core endpoint routing for A2A. Depends on Hosting.A2A. |
| **Microsoft.Agents.AI.Hosting.AGUI.AspNetCore** | 11 | 399 | 2 | ASP.NET Core SSE endpoint for AG-UI. Depends on Hosting. |
| **Microsoft.Agents.AI.Hosting.AzureFunctions** | 22 | 970 | 6 | Azure Functions triggers for durable agents. Depends on DurableTask. |

### 2.6 Durable / Persistence

| Project | Files | LOC | Public Types | Role |
|---|---|---|---|---|
| **Microsoft.Agents.AI.DurableTask** | 50 | 2,799 | 12 | Durable Task Framework integration for long-running agents. Depends on AI. |
| **Microsoft.Agents.AI.CosmosNoSql** | 10 | 1,229 | 5 | Cosmos DB chat history + workflow checkpoint storage. Depends on Abstractions + Workflows. |

### 2.7 Specialized / Add-on Libraries

| Project | Files | LOC | Public Types | Role |
|---|---|---|---|---|
| **Microsoft.Agents.AI.Declarative** | 30 | 1,149 | 9 | YAML-based agent factory (non-workflow). Depends on AI + Abstractions. |
| **Microsoft.Agents.AI.DevUI** | 19 | 1,138 | 3 | Developer UI middleware. Depends on Hosting + Hosting.OpenAI. |
| **Microsoft.Agents.AI.Mem0** | 15 | 682 | 3 | Mem0 memory provider integration. Depends on Abstractions. |
| **Microsoft.Agents.AI.Purview** | 82 | 3,309 | 9 | Microsoft Purview compliance integration. Depends on AI + Abstractions. |

---

## 3. Project Dependency Graph (Simplified)

```text
Abstractions (root - no deps)
├── AI (core impl)
│   ├── OpenAI
│   ├── AzureAI
│   │   └── AzureAI.Persistent
│   ├── Anthropic
│   ├── DurableTask
│   │   └── Hosting.AzureFunctions
│   ├── Purview
│   └── Declarative (agent factory)
├── A2A
├── AGUI
├── CopilotStudio
├── GitHub.Copilot
├── Mem0
├── Workflows (+ AI)
│   ├── Workflows.Declarative
│   │   └── Workflows.Declarative.AzureAI
│   └── CosmosNoSql
├── Hosting (+ AI, Workflows)
│   ├── Hosting.OpenAI
│   ├── Hosting.A2A
│   │   └── Hosting.A2A.AspNetCore
│   ├── Hosting.AGUI.AspNetCore
│   └── DevUI (+ Hosting.OpenAI)
└── Workflows.Generators (standalone Roslyn generator)
```

---

## 4. NuGet Package Dependencies (from `Directory.Packages.props`)

### 4.1 AI / ML SDKs

| Package | Version | Used By |
|---|---|---|
| Microsoft.Extensions.AI | 10.2.0 | AI, AGUI, Anthropic, AzureAI, AzureAI.Persistent, Purview |
| Microsoft.Extensions.AI.Abstractions | 10.2.0 | Abstractions, Hosting.OpenAI |
| Microsoft.Extensions.AI.OpenAI | 10.2.0-preview | AzureAI, Hosting.OpenAI, OpenAI |
| OpenAI | 2.8.0 | AzureAI |
| Anthropic | 12.0.1 | Anthropic |
| Anthropic.Foundry | 0.1.0 | (samples) |
| Azure.AI.Projects | 1.2.0-beta.5 | AzureAI |
| Azure.AI.Projects.OpenAI | 1.0.0-beta.5 | AzureAI |
| Azure.AI.Agents.Persistent | 1.2.0-beta.8 | AzureAI.Persistent |
| Azure.AI.OpenAI | 2.8.0-beta.1 | (samples) |
| GitHub.Copilot.SDK | 0.1.18 | GitHub.Copilot |
| Microsoft.Agents.CopilotStudio.Client | 1.3.171-beta | CopilotStudio |

### 4.2 Infrastructure / Hosting

| Package | Version | Used By |
|---|---|---|
| Microsoft.Extensions.Hosting | 10.0.0 | Hosting |
| Microsoft.Extensions.DependencyInjection | 10.0.0 | Purview |
| Microsoft.Extensions.DependencyInjection.Abstractions | 10.0.2 | AI, Purview |
| Microsoft.Extensions.Logging.Abstractions | 10.0.2 | Abstractions, AI |
| Microsoft.Extensions.Configuration | 10.0.0 | Declarative, Workflows.Declarative |
| System.Text.Json | 10.0.2 | Hosting, Workflows |
| System.Threading.Channels | 10.0.2 | AGUI, Hosting, Workflows |

### 4.3 Azure Services

| Package | Version | Used By |
|---|---|---|
| Azure.Identity | 1.17.1 | CosmosNoSql, Workflows.Declarative.AzureAI, Purview |
| Microsoft.Azure.Cosmos | 3.54.0 | CosmosNoSql |

### 4.4 Protocols

| Package | Version | Used By |
|---|---|---|
| A2A | 0.3.3-preview | A2A, Hosting.A2A |
| A2A.AspNetCore | 0.3.3-preview | Hosting.A2A.AspNetCore |
| ModelContextProtocol | 0.4.0-preview.3 | (samples, Hosting.AzureFunctions) |
| System.Net.ServerSentEvents | 10.0.0 | AGUI, Hosting.AGUI.AspNetCore |

### 4.5 Durable Task / Azure Functions

| Package | Version | Used By |
|---|---|---|
| Microsoft.DurableTask.Client | 1.18.0 | DurableTask |
| Microsoft.DurableTask.Worker | 1.18.0 | DurableTask |
| Microsoft.Azure.Functions.Worker | 2.50.0 | Hosting.AzureFunctions |
| Microsoft.Azure.Functions.Worker.Extensions.DurableTask | 1.11.0 | Hosting.AzureFunctions |
| Microsoft.Azure.Functions.Worker.Extensions.Http | 3.3.0 | Hosting.AzureFunctions |
| Microsoft.Azure.Functions.Worker.Extensions.Mcp | 1.0.0 | Hosting.AzureFunctions |

### 4.6 Workflows / Declarative

| Package | Version | Used By |
|---|---|---|
| Microsoft.Agents.ObjectModel | 2026.1.2.3 | Declarative, Workflows.Declarative |
| Microsoft.Agents.ObjectModel.Json | 2026.1.2.3 | Declarative, Workflows.Declarative |
| Microsoft.Agents.ObjectModel.PowerFx | 2026.1.2.3 | Declarative, Workflows.Declarative |
| Microsoft.PowerFx.Interpreter | 1.5.0-build | Declarative, Workflows.Declarative |

### 4.7 Observability

| Package | Version | Used By |
|---|---|---|
| OpenTelemetry.Api | 1.13.1 | Workflows |
| System.Diagnostics.DiagnosticSource | 10.0.2 | AI, Hosting, Workflows, Purview |

### 4.8 Tooling / Analyzers (dev-only)

| Package | Version |
|---|---|
| Microsoft.CodeAnalysis.CSharp | 4.14.0 |
| Microsoft.CodeAnalysis.Analyzers | 3.11.0 |
| Microsoft.CodeAnalysis.NetAnalyzers | 10.0.100 |
| Roslynator.Analyzers | 4.14.1 |

---

## 5. Test Projects — `dotnet/tests/`

### 5.1 Unit Tests (22 projects)

| Test Project | .cs Files |
|---|---|
| Microsoft.Agents.AI.UnitTests | 33 |
| Microsoft.Agents.AI.Abstractions.UnitTests | 26 |
| Microsoft.Agents.AI.Workflows.UnitTests | 63 |
| Microsoft.Agents.AI.Workflows.Declarative.UnitTests | 96 |
| Microsoft.Agents.AI.Hosting.OpenAI.UnitTests | 30 |
| Microsoft.Agents.AI.A2A.UnitTests | 18 |
| Microsoft.Agents.AI.Hosting.A2A.UnitTests | 15 |
| Microsoft.Agents.AI.OpenAI.UnitTests | 14 |
| Microsoft.Agents.AI.AGUI.UnitTests | 13 |
| Microsoft.Agents.AI.Hosting.UnitTests | 13 |
| Microsoft.Agents.AI.Hosting.AGUI.AspNetCore.UnitTests | 13 |
| Microsoft.Agents.AI.DurableTask.UnitTests | 15 |
| Microsoft.Agents.AI.Hosting.AzureFunctions.UnitTests | 11 |
| Microsoft.Agents.AI.AzureAI.UnitTests | 11 |
| Microsoft.Agents.AI.GitHub.Copilot.UnitTests | 11 |
| Microsoft.Agents.AI.DevUI.UnitTests | 11 |
| Microsoft.Agents.AI.Declarative.UnitTests | 10 |
| Microsoft.Agents.AI.Purview.UnitTests | 9 |
| Microsoft.Agents.AI.CosmosNoSql.UnitTests | 9 |
| Microsoft.Agents.AI.Anthropic.UnitTests | 8 |
| Microsoft.Agents.AI.AzureAI.Persistent.UnitTests | 7 |
| Microsoft.Agents.AI.Mem0.UnitTests | 7 |
| Microsoft.Agents.AI.Workflows.Generators.UnitTests | 5 |

### 5.2 Integration Tests (14 projects)

| Test Project | .cs Files |
|---|---|
| Microsoft.Agents.AI.Workflows.Declarative.IntegrationTests | 25 |
| AgentConformance.IntegrationTests | 18 |
| Microsoft.Agents.AI.DurableTask.IntegrationTests | 18 |
| Microsoft.Agents.AI.Hosting.AGUI.AspNetCore.IntegrationTests | 13 |
| AzureAI.IntegrationTests | 12 |
| AzureAIAgentsPersistent.IntegrationTests | 12 |
| CopilotStudio.IntegrationTests | 12 |
| OpenAIAssistant.IntegrationTests | 12 |
| AnthropicChatCompletion.IntegrationTests | 11 |
| OpenAIChatCompletion.IntegrationTests | 11 |
| OpenAIResponse.IntegrationTests | 11 |
| Microsoft.Agents.AI.GitHub.Copilot.IntegrationTests | 10 |
| Microsoft.Agents.AI.Hosting.AzureFunctions.IntegrationTests | 10 |
| Microsoft.Agents.AI.Mem0.IntegrationTests | 7 |

---

## 6. Sample Projects — `dotnet/samples/`

### 6.1 Top-Level Sample Groups

| Directory | Description | Sample Count |
|---|---|---|
| **GettingStarted/Agents/** | Step-by-step agent tutorials | 20 |
| **GettingStarted/AgentProviders/** | Provider-specific samples (OpenAI, Azure, Anthropic, Ollama, ONNX, etc.) | 16 |
| **GettingStarted/FoundryAgents/** | Azure Foundry agent tutorials | 16 |
| **GettingStarted/Workflows/** | Workflow samples (concurrent, conditional, declarative, loop, checkpoint, HITL, observability, visualization) | ~30 |
| **GettingStarted/AGUI/** | AG-UI step-by-step (5 steps, client+server each) | 10 |
| **GettingStarted/A2A/** | A2A protocol samples | 2 |
| **GettingStarted/AgentWithOpenAI/** | OpenAI-specific agent steps | 5 |
| **GettingStarted/AgentWithAnthropic/** | Anthropic-specific agent steps | 3 |
| **GettingStarted/AgentWithMemory/** | Memory provider samples | 3 |
| **GettingStarted/AgentWithRAG/** | RAG pattern samples | 4 |
| **GettingStarted/DevUI/** | Developer UI sample | 1 |
| **GettingStarted/DeclarativeAgents/** | Declarative agent sample | 1 |
| **GettingStarted/ModelContextProtocol/** | MCP server samples | 4 |
| **GettingStarted/Observability/** | OpenTelemetry sample | 1 |
| **Durable/Agents/AzureFunctions/** | Durable agent Azure Functions samples | 8 |
| **Durable/Agents/ConsoleApps/** | Durable agent console samples | 7 |
| **A2AClientServer/** | Full A2A client/server demo | 2 |
| **AGUIClientServer/** | Full AG-UI client/server demo | 3 |
| **AgentWebChat/** | Aspire-based web chat demo | 4 |
| **AGUIWebChat/** | AG-UI web chat demo | — |
| **HostedAgents/** | Hosted agent scenarios | 3 |
| **M365Agent/** | Microsoft 365 agent | 1 |
| **Purview/** | Purview compliance sample | 1 |

---

## 7. Complexity Assessment for C++ Port

### 7.1 Tier 1 — Must Port (Core)

These form the essential foundation:

| Project | LOC | Complexity | C++ Considerations |
|---|---|---|---|
| **Abstractions** | 2,827 | Medium | Pure interfaces/abstracts; maps to C++ abstract classes + concepts. Biggest challenge: async patterns (`IAsyncEnumerable`, `Task<T>`). |
| **AI** | 4,065 | Medium–High | Uses `Microsoft.Extensions.AI` heavily; need C++ equivalent of `IChatClient`, DI, logging. |
| **Workflows** | 12,915 | **High** | State machines, streaming, checkpointing, OpenTelemetry. Most complex piece; source generators need compile-time alternative. |

### 7.2 Tier 2 — Important for Functionality

| Project | LOC | Complexity | Notes |
|---|---|---|---|
| **Hosting** | 771 | Low–Medium | DI/builder pattern; C++ would use different hosting model. |
| **OpenAI** | 1,007 | Low | Thin adapter layer over OpenAI C++ SDK. |
| **Hosting.OpenAI** | 10,668 | **High** | Large HTTP API surface; would need C++ HTTP framework (e.g., cpp-httplib, Boost.Beast). |
| **DurableTask** | 2,799 | Medium–High | Depends on Azure Durable Task SDK (no C++ equivalent). |

### 7.3 Tier 3 — Provider-Specific (Port as Needed)

| Project | LOC | Notes |
|---|---|---|
| AzureAI | 1,020 | Azure SDK dependency |
| AzureAI.Persistent | 552 | Azure SDK dependency |
| Anthropic | 358 | Thin wrapper |
| CopilotStudio | 335 | Microsoft SDK dependency |
| GitHub.Copilot | 646 | GitHub SDK dependency |
| A2A | 937 | Protocol library |
| AGUI | 1,900 | SSE-based protocol |
| CosmosNoSql | 1,229 | Azure Cosmos SDK |
| Purview | 3,309 | Microsoft compliance APIs |
| DevUI | 1,138 | Web UI middleware |
| Mem0 | 682 | HTTP-based memory service |

### 7.4 Tier 4 — Declarative / Tooling (Likely Skip or Redesign)

| Project | LOC | Notes |
|---|---|---|
| Workflows.Declarative | 17,417 | PowerFx dependency — no C++ equivalent; would need different expression engine. |
| Declarative | 1,149 | YAML agent factory — YAML parsing available in C++ (yaml-cpp). |
| Workflows.Generators | 1,656 | Roslyn source generator — N/A for C++; would use code generation or template metaprogramming. |

### 7.5 Key .NET Patterns Requiring C++ Equivalents

| .NET Pattern | C++ Equivalent Strategy |
|---|---|
| `async/await`, `Task<T>`, `ValueTask<T>` | `std::future`, coroutines (`co_await`), or callback-based |
| `IAsyncEnumerable<T>` | C++20 coroutines with `co_yield`, or custom async generator |
| Dependency Injection (`IServiceCollection`) | Manual DI, or libraries like Boost.DI / fruit |
| `ILogger` / `ILoggerFactory` | spdlog, or custom logging abstraction |
| `Microsoft.Extensions.AI` (`IChatClient`) | Custom abstract interface hierarchy |
| `System.Text.Json` serialization | nlohmann/json or simdjson |
| `System.Threading.Channels` | `std::queue` + `std::mutex` + `std::condition_variable`, or Boost.Lockfree |
| OpenTelemetry (.NET SDK) | OpenTelemetry C++ SDK |
| Source generators (Roslyn) | CMake code generation, or C++ templates |
| `IAsyncDisposable` / `using` | RAII pattern (destructors) |
| Records / `with` expressions | Structs with copy constructors |
| Central Package Management | vcpkg or Conan manifest |

---

## 8. Summary Statistics

| Metric | Value |
|---|---|
| Total src projects | 27 (+ LegacySupport + Shared) |
| Total .cs files (src) | ~1,167 |
| Total LOC (src) | ~71,168 |
| Total public types | ~244 |
| Total unit test projects | 22 |
| Total integration test projects | 14 |
| Total sample projects | ~145 |
| Key external dependencies | ~45 NuGet packages |
| Largest project (LOC) | Workflows.Declarative (17,417) |
| Largest project (types) | Workflows (104 public types) |
| Core portable LOC (Tiers 1+2) | ~34,112 |
