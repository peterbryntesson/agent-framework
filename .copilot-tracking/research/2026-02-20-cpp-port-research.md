<!-- markdownlint-disable-file -->
# Task Research: C++ Port of Agent Framework

Full port of the .NET and Python agent framework to C++ with feature parity. This document catalogs the existing implementations, maps every component to C++ equivalents, selects a technology stack, and identifies risks.

## Task Implementation Requests

* Port all core abstractions (AIAgent, ChatClient, Content model, sessions) to C++
* Port all AI provider integrations (OpenAI, Azure AI, Anthropic, Copilot Studio, GitHub Copilot)
* Port all hosting layers (OpenAI-compatible, A2A, AG-UI, Azure Functions)
* Port the workflow engine (graph executor, declarative YAML, PowerFx expressions)
* Port durable task / entity support
* Port cross-cutting concerns (OpenTelemetry, middleware pipeline, DI, serialization)
* Port storage backends (Cosmos DB, Mem0, Redis)
* Port the developer UI
* Establish build, packaging, and test infrastructure

## Scope and Success Criteria

* Scope: Complete C++ reimplementation targeting C++20 (minimum). Feature parity with .NET (28 projects) and Python (20 packages). Cross-platform (Windows, Linux, macOS).
* Assumptions:
  * C++20 coroutines are the async foundation (maps to `async`/`await` in C# and Python).
  * No official OpenAI, Anthropic, or Azure AI C++ SDKs exist — all AI provider clients must be built from REST APIs.
  * PowerFx has no C++ port — an alternative expression engine is required for declarative workflows.
  * The Durable Task sidecar pattern (gRPC client) is preferred over a full engine reimplementation.
* Success Criteria:
  * Every public API surface from .NET and Python has a C++ equivalent.
  * All agent samples (GettingStarted, HostedAgents, Durable, M365Agent, etc.) can be reproduced in C++.
  * All hosting protocols (OpenAI-compatible, A2A, AG-UI) pass interop tests with existing .NET/Python clients.
  * Build and test on MSVC, GCC 11+, and Clang 14+.
  * Distributable via vcpkg.

## Outline

1. [Existing Implementation Analysis](#existing-implementation-analysis)
2. [C++ Technology Stack Selection](#c-technology-stack-selection)
3. [Component Mapping: .NET/Python → C++](#component-mapping)
4. [Architecture Decisions](#architecture-decisions)
5. [Library Dependency Matrix](#library-dependency-matrix)
6. [Project Structure](#project-structure)
7. [Implementation Tiers](#implementation-tiers)
8. [Risk Assessment](#risk-assessment)
9. [Potential Next Research](#potential-next-research)

### Potential Next Research

* PowerFx subset evaluator design — detailed grammar and feature set needed
  * Reasoning: PowerFx is the hardest gap; need to define exact subset (boolean conditions, string interpolation, variable scopes) to scope the effort.
  * Reference: `dotnet/src/Microsoft.Agents.AI.Workflows.Declarative/` PowerFx integration code
* Durable Task proto definitions and client SDK architecture
  * Reasoning: Need to map `DurableAIAgent` entity operations to gRPC calls against the sidecar.
  * Reference: `github.com/microsoft/durabletask-protobuf`
* C++ source generator alternatives (for workflow message routing)
  * Reasoning: .NET uses Roslyn IIncrementalGenerator; need C++ compile-time or build-time equivalent.
  * Reference: `dotnet/src/Microsoft.Agents.AI.Workflows.Generators/`
* MCP C++ SDK integration testing
  * Reasoning: `modelcontextprotocol/cpp-sdk` exists but needs validation for all required MCP features.
  * Reference: `github.com/modelcontextprotocol/cpp-sdk`

---

## Existing Implementation Analysis

### .NET Architecture (28 Projects)

#### Core Layer

| Project | Purpose | Key Types |
|---|---|---|
| `Abstractions` | Core agent model | `AIAgent` (abstract), `AgentSession`, `AgentResponse<T>`, `ChatHistoryProvider`, `AIContextProvider`, `AgentSessionStore` |
| `AI` | Primary implementations | `ChatClientAgent` (910 LOC), `AIAgentBuilder`, `DelegatingAIAgent`, `LoggingAgent`, `OpenTelemetryAgent`, `FunctionInvocationDelegatingAgent`, `AnonymousDelegatingAIAgent`, `AIHostAgent` |
| `Hosting` | Hosting abstractions | `IHostedAgentBuilder`, `IHostedWorkflowBuilder` |

**Core Abstract Class Hierarchy:**
```
AIAgent (abstract)
├── ChatClientAgent (wraps IChatClient)
├── DelegatingAIAgent (decorator base)
│   ├── LoggingAgent
│   ├── OpenTelemetryAgent
│   ├── FunctionInvocationDelegatingAgent
│   ├── AnonymousDelegatingAIAgent
│   └── AIHostAgent
├── CopilotStudioAgent (direct implementation)
├── GitHubCopilotAgent (direct implementation)
├── A2AAgent (remote proxy)
└── DurableAIAgent (entity-backed)
```

**Session Hierarchy:**
```
AgentSession (abstract)
├── InMemoryAgentSession (client-side state)
└── ServiceIdAgentSession (server-managed with ConversationId)
    ├── AzureAIAgentSession
    ├── CopilotStudioAgentSession
    ├── GitHubCopilotAgentSession
    └── A2AAgentSession
```

**Key Design Patterns:**
1. **Decorator/Delegating pattern** — `DelegatingAIAgent` wraps `InnerAgent` for cross-cutting concerns
2. **Two-phase lifecycle** — `InvokingAsync`/`InvokedAsync` on `ChatHistoryProvider` and `AIContextProvider`
3. **Builder pattern** — `AIAgentBuilder` with fluent API
4. **Service locator** — `GetService(Type, object?)` on all core types
5. **Dual session strategy** — server-side `ConversationId` vs client-side `ChatHistoryProvider`
6. **Keyed DI services** — agents registered and resolved by name

#### Provider Layer (6 Projects)

| Project | Integration Tier | Key Pattern |
|---|---|---|
| `OpenAI` | IChatClient-based | Extension `AsAIAgent()` on OpenAI client SDK types |
| `AzureAI` | IChatClient-based | Most complex: custom `DelegatingChatClient`, tool validation, schema transforms |
| `AzureAI.Persistent` | IChatClient-based | Server-managed agents via Azure AI Agent Service |
| `Anthropic` | IChatClient-based | Extension on Anthropic SDK |
| `CopilotStudio` | Direct AIAgent subclass | Custom session with DirectLine protocol |
| `GitHub.Copilot` | Direct AIAgent subclass | Channel-based streaming |

**Two integration tiers:**
1. **IChatClient-based**: Provider SDK → extension method → `IChatClient` → `ChatClientAgent` (gets middleware, telemetry, sessions for free)
2. **Direct AIAgent subclass**: Custom `RunCoreAsync`/`RunCoreStreamingAsync` (more complex but more control)

#### Protocol/Hosting Layer (5 Projects)

| Project | Protocol | Transport |
|---|---|---|
| `Hosting.OpenAI` | Chat Completions, Responses, Conversations API | HTTP + SSE |
| `Hosting.A2A.AspNetCore` | Google A2A | HTTP + SSE (JSON-RPC 2.0) |
| `Hosting.AGUI.AspNetCore` | CopilotKit AG-UI | HTTP POST → SSE stream |
| `Hosting.AzureFunctions` | Dynamic function triggers | HTTP + MCP tool triggers |
| `Hosting.A2A` | A2A base | TaskManager abstraction |

**SSE streaming pattern used by all hosting projects:**
- Content-Type: `text/event-stream`
- Cache-Control: `no-cache,no-store`
- Chunked transfer encoding
- Line-delimited `data: <json>\n\n` events

#### Workflow Layer (5 Projects)

| Project | Purpose |
|---|---|
| `Workflows` | Graph-based executor. Nodes (`Executor`), edges (`DirectEdge`/`FanOutEdge`/`FanInEdge`). Builders for sequential, concurrent, handoffs, group chat. Checkpointing. OpenTelemetry. |
| `Workflows.Declarative` | YAML → `Workflow` via PowerFx. Variable scopes (Local/Global/System/Environment/Topic). `WorkflowAgentProvider` abstraction. |
| `Workflows.Declarative.AzureAI` | Azure AI Foundry declarative agents |
| `Workflows.Generators` | Roslyn source generators for `[MessageHandler]`, `[SendsMessage]`, `[YieldsOutput]` |
| `DurableTask` | Entity-backed agent state. `DurableAIAgent`/`DurableAIAgentProxy`. Composite `AgentSessionId`. |

#### Storage/Integration Layer (4 Projects)

| Project | Purpose | Key Dependencies |
|---|---|---|
| `CosmosNoSql` | Chat history + checkpoint storage | Azure.Cosmos SDK |
| `Mem0` | Semantic memory via REST API | HttpClient |
| `Purview` | DLP middleware (blocks/allows prompts/responses) | Microsoft Graph |
| `DevUI` | Embedded developer SPA | ASP.NET Core |

---

### Python Architecture (20 Packages)

#### Core Package

| Component | Key Types |
|---|---|
| Agent model | `AgentProtocol`, `BaseAgent`, `ChatAgent` |
| Chat client | `ChatClientProtocol[TOptions]`, `BaseChatClient` |
| Content model | `Content` (18 `ContentType` literals, single class with factory methods) |
| Messages | `ChatMessage`, `Role`, `FinishReason` (EnumLike metaclass) |
| Streaming | `ChatResponseUpdate`, `AgentResponseUpdate`, `AgentResponse.from_chat_response_updates()` |
| Tools | `ToolProtocol`, `FunctionTool` (auto-schema from signatures), `@tool` decorator |
| Middleware | 3-level pipeline (agent→chat→function), ABC + callable, onion model |
| Observability | OpenTelemetry decorators with GenAI semantic conventions |
| Threads | `AgentThread` (dual-mode: server-managed or local store) |
| Context | `ContextProvider` (injects instructions/messages/tools per run) |

#### External Packages

| Package | Maps To (.NET) |
|---|---|
| `a2a` | A2A protocol interop |
| `ag-ui` | AG-UI protocol (FastAPI SSE endpoints) |
| `anthropic` | Anthropic Claude client |
| `azure-ai` | Azure AI Agent Service client |
| `azure-ai-search` | Azure AI Search integration |
| `chatkit` | Chat UI components |
| `copilotstudio` | Copilot Studio connector |
| `declarative` | YAML-driven agents/workflows with PowerFx |
| `devui` | Developer UI |
| `durabletask` | Durable entity-backed agents via gRPC |
| `foundry_local` | Local Foundry runtime |
| `github_copilot` | GitHub Copilot agent |
| `lab` | Experimental features |
| `mem0` | Mem0 memory integration |
| `ollama` | Ollama local model client |
| `purview` | Data governance middleware |
| `redis` | Redis storage backend |
| `bedrock` | AWS Bedrock client |
| `azurefunctions` | Azure Functions hosting |

---

### Microsoft.Extensions.AI Types (Must Reimplement)

The .NET framework depends heavily on `Microsoft.Extensions.AI` v10.2.0 (~25 types). These are the foundation of the content/message model and **must be reimplemented in C++**:

| Category | Type | Purpose |
|---|---|---|
| Content | `AIContent` (abstract) | Base content type |
| | `TextContent` | Text payload |
| | `FunctionCallContent` | Tool invocation request |
| | `FunctionResultContent` | Tool invocation result |
| | `DataContent` | Binary data (images, files) |
| | `ErrorContent` | Error information |
| | `UsageContent` | Token usage data |
| | `AudioContent` | Audio data |
| | `ImageContent` | Image reference |
| | `UriContent` | URI reference |
| Messages | `ChatMessage` | role + `IList<AIContent>` |
| | `ChatRole` | user/assistant/system/tool |
| Response | `ChatResponse` | Full response (non-streaming) |
| | `ChatResponseUpdate` | Streaming chunk |
| Options | `ChatOptions` | Temperature, model, tools, etc. |
| Client | `IChatClient` | `GetResponseAsync` + `GetStreamingResponseAsync` |
| | `DelegatingChatClient` | Decorator base |
| Tools | `AITool` / `AIFunction` | Tool abstraction + invocation |
| | `AIFunctionFactory` | Creates AIFunction from delegates |
| Supporting | `UsageDetails` | Token counts |
| | `AdditionalPropertiesDictionary` | Extension data |
| | `ResponseContinuationToken` | Multi-turn continuation |

**Polymorphic JSON:** `AIContent` uses `$type` discriminator in JSON for polymorphic serialization.

---

## C++ Technology Stack Selection

### Selected Stack

| Concern | Library | Version | License | Rationale |
|---|---|---|---|---|
| **C++ Standard** | C++20 | — | — | Coroutines, concepts, ranges, `std::format`, `std::stop_token` |
| **Build** | CMake 3.25+ | Latest | BSD-3 | Industry standard, FILE_SET support, presets |
| **Package Manager** | vcpkg (manifest mode) | Latest | MIT | Microsoft-aligned, first-class CMake integration |
| **Async Runtime** | Boost.Asio | 1.87+ | BSL-1.0 | Mature, cross-platform, C++20 coroutines, built-in I/O, channels |
| **HTTP Client** | Boost.Beast | 1.87+ | BSL-1.0 | Built on Asio, HTTP/1.1 + WebSocket, SSE via incremental parser |
| **HTTP Server** | Boost.Beast | 1.87+ | BSL-1.0 | Same stack for consistency; DIY routing + middleware |
| **JSON (primary)** | glaze | 4.x+ | MIT | Tagged variant polymorphism, JSON Schema gen, C++20 compile-time reflection, 10x faster than nlohmann |
| **JSON (fallback)** | nlohmann/json | 3.11+ | MIT | Broader compatibility, community familiarity |
| **YAML** | yaml-cpp | 0.8+ | MIT | De facto standard C++ YAML |
| **Testing** | Google Test + gmock | 1.15+ | BSD-3 | Industry standard, built-in mocking |
| **Observability** | opentelemetry-cpp | 1.18+ | Apache-2.0 | Official OTEL SDK, GenAI semantic conventions via attributes |
| **Logging** | spdlog + fmt | 1.14+ / 11+ | MIT | Fast structured logging |
| **gRPC** | grpc/grpc | 1.70+ | Apache-2.0 | Durable Task sidecar communication |
| **MCP** | modelcontextprotocol/cpp-sdk | Latest | MIT | Official MCP C++ SDK for tool serving |
| **Expression Engine** | Lua via sol2 | 4.x | MIT | PowerFx alternative for declarative workflows |
| **Auth (Azure)** | azure-identity-cpp | Latest | MIT | `DefaultAzureCredential` for Azure services |
| **TLS** | OpenSSL | 3.x | Apache-2.0 | Used by all networking libraries |

### Alternative Evaluation

#### HTTP Framework: Drogon vs Boost.Beast

| Criterion | Drogon | Boost.Beast |
|---|---|---|
| Routing/Middleware | Built-in | Manual |
| SSE streaming | `newStreamResponse()` | Manual chunked |
| HTTP client | Built-in | Built-in |
| WebSocket | Built-in | Built-in |
| Event loop | Custom (not Asio) | Boost.Asio |
| Dependency weight | Medium (~20 deps) | Boost only |
| Ecosystem fit | Standalone | Integrates with all Boost libs |

**Decision: Boost.Beast.** While Drogon offers more batteries-included features, Boost.Beast provides:
- A unified Asio-based async model used across all components
- No second event loop to bridge
- Boost.Asio's `experimental::channel`, cancellation, and parallel_group
- Better long-term maintenance guarantee (Boost)
- More control over routing and middleware patterns (matches the framework's custom architectures)

The cost is more boilerplate for routing/middleware, but the framework already needs custom middleware pipelines — we build them once in a shared library.

#### JSON: glaze vs nlohmann/json

| Criterion | glaze | nlohmann/json |
|---|---|---|
| Polymorphic discriminators | Built-in `tagged_variant` | Manual registry |
| JSON Schema generation | Built-in `write_json_schema<T>()` | Not supported |
| Performance | 10x faster | Slower but sufficient |
| C++ standard | C++20 required | C++11+ |
| Community | Growing (2k stars) | Massive (43k stars) |
| API stability | Evolving | Stable |

**Decision: glaze (primary) with nlohmann/json (where needed).** glaze's built-in tagged variant and JSON Schema generation directly serve the framework's polymorphic content types and tool schema generation. Use nlohmann/json for integration with libraries that expect it (e.g., MCP C++ SDK).

#### Expression Engine: Lua/sol2 vs Custom Subset Evaluator

| Criterion | Lua/sol2 | Custom PowerFx Subset |
|---|---|---|
| Development effort | Low (mature library) | High (parser + evaluator) |
| Syntax compatibility | Different from PowerFx | Can match PowerFx exactly |
| Performance | Excellent (JIT via LuaJIT) | Depends on implementation |
| Extensibility | Full scripting language | Limited to implemented features |
| User familiarity | Common in gaming/embedded | PowerFx users only |

**Decision: Lua/sol2 with a PowerFx compatibility shim.** A thin wrapper layer translates common PowerFx patterns (`Set()`, `If()`, variable scoping, `Concatenate()`, `Text()`) to Lua equivalents. Users writing declarative YAML can use either PowerFx-like syntax (translated) or native Lua.

---

## Component Mapping

### Core Abstractions

| .NET / Python | C++ Equivalent |
|---|---|
| `AIAgent` / `AgentProtocol` | `class AIAgent` (abstract, virtual `run_async`/`run_streaming_async`) |
| `DelegatingAIAgent` | `class DelegatingAIAgent : public AIAgent` (holds `std::shared_ptr<AIAgent>`) |
| `IChatClient` / `ChatClientProtocol` | `class IChatClient` (pure virtual, `get_response` + `get_streaming_response`) |
| `DelegatingChatClient` / `BaseChatClient` | `class DelegatingChatClient : public IChatClient` |
| `ChatClientAgent` / `ChatAgent` | `class ChatClientAgent : public AIAgent` (wraps `IChatClient`) |
| `AgentSession` / `AgentThread` | `class AgentSession` (abstract, dual-mode) |
| `AIContextProvider` / `ContextProvider` | `class AIContextProvider` (virtual `invoking_async`/`invoked_async`) |
| `ChatHistoryProvider` / `ChatMessageStore` | `class ChatHistoryProvider` (virtual two-phase lifecycle) |
| `AIAgentBuilder` | `class AIAgentBuilder` (fluent builder) |

### Content/Message Model

| .NET (M.E.AI) / Python | C++ Type |
|---|---|
| `AIContent` / `Content` | `std::variant<TextContent, FunctionCallContent, FunctionResultContent, DataContent, ErrorContent, UsageContent, AudioContent, ImageContent, UriContent>` via glaze `tagged_variant` |
| `ChatMessage` / `ChatMessage` | `struct ChatMessage { ChatRole role; std::vector<AIContent> contents; }` |
| `ChatRole` / `Role` | `enum class ChatRole { System, User, Assistant, Tool }` |
| `ChatResponse` | `struct ChatResponse { ChatMessage message; UsageDetails usage; ... }` |
| `ChatResponseUpdate` / `ChatResponseUpdate` | `struct ChatResponseUpdate { ... }` (streaming chunk) |
| `ChatOptions` / `ChatOptions` | `struct ChatOptions { std::optional<std::string> model; ... }` |
| `AgentResponse` / `AgentResponse` | `struct AgentResponse { std::string agent_id; ChatResponse response; }` |
| `AgentResponseUpdate` / `AgentResponseUpdate` | `struct AgentResponseUpdate { ... }` |

### Async Primitives

| .NET / Python | C++ Equivalent |
|---|---|
| `Task<T>` / `Coroutine[T]` | `asio::awaitable<T>` or custom `Task<T>` |
| `IAsyncEnumerable<T>` / `AsyncIterable[T]` | Custom `AsyncGenerator<T>` with `co_yield` |
| `CancellationToken` | `std::stop_token` (C++20) |
| `CancellationTokenSource` | `std::stop_source` |
| `Channel<T>` | `asio::experimental::channel<void(error_code, T)>` |
| `Task.WhenAll` / `asyncio.gather` | `asio::experimental::make_parallel_group()` |
| `SemaphoreSlim` / `asyncio.Semaphore` | Custom `AsyncSemaphore` or cppcoro equivalent |

### Tools/Functions

| .NET / Python | C++ Equivalent |
|---|---|
| `AIFunction` / `FunctionTool` | `class AIFunction` with JSON Schema + invoke |
| `AIFunctionFactory.Create()` | Template factory: `make_function<ReturnType>(name, desc, callable)` |
| `AITool` / `ToolProtocol` | `class AITool` (base, with `name` and `description`) |
| Function schema from signatures | C++20 concepts + template introspection (limited vs reflection in C#/Python) |
| `@tool` decorator | `AGENT_TOOL(name, desc)` macro or `make_tool()` factory |
| MCP tool serving | Use `modelcontextprotocol/cpp-sdk` |

### Middleware

| .NET / Python | C++ Equivalent |
|---|---|
| Agent middleware | `using AgentMiddleware = std::function<Task<void>(AgentContext&, AgentDelegate)>` |
| Chat middleware | `using ChatMiddleware = std::function<Task<void>(ChatContext&, ChatDelegate)>` |
| Function middleware | `using FunctionMiddleware = std::function<Task<void>(FunctionContext&, FunctionDelegate)>` |
| `DelegatingHandler` chain | Onion pipeline built from inside-out with `std::function` lambdas |

### Hosting/Protocols

| .NET / Python | C++ Equivalent |
|---|---|
| ASP.NET Core endpoints | Boost.Beast HTTP server with router |
| FastAPI endpoints | Same Boost.Beast server |
| SSE streaming (`SseJsonResult<T>`) | Custom SSE writer on Beast response stream |
| OpenAI-compatible API | Custom routes mapping to agent invocations |
| A2A protocol | Custom JSON-RPC 2.0 over HTTP + SSE |
| AG-UI protocol | Custom SSE event stream (12 event types) |
| Azure Functions hosting | Not directly portable — provide standalone daemon alternative |

### Workflows

| .NET / Python | C++ Equivalent |
|---|---|
| `Workflow` graph executor | `class WorkflowExecutor` with node/edge graph |
| `Executor` nodes | `class WorkflowNode` (virtual `execute_async`) |
| `DirectEdge`/`FanOutEdge`/`FanInEdge` | Edge types with routing logic |
| PowerFx `RecalcEngine` | Lua via sol2 with compatibility shim |
| YAML declarations | yaml-cpp parsing into workflow/agent models |
| Roslyn source generators | CMake custom commands or build-time code generation |
| Checkpointing | `ICheckpointStore` interface with JSON serialization |

### Storage

| .NET / Python | C++ Equivalent |
|---|---|
| Cosmos DB (`ChatHistoryProvider`) | Azure Cosmos DB REST API client (no C++ SDK) or CosmosDB via HTTP |
| Mem0 REST API | HTTP client (Boost.Beast) |
| Redis | hiredis or redis-plus-plus |
| In-memory store | `std::unordered_map` with mutex |

---

## Architecture Decisions

### AD-1: Async Foundation — Boost.Asio + C++20 Coroutines

All async code uses `co_await`/`co_yield`/`co_return` with Boost.Asio as the executor.

**Core async types:**
```cpp
// Task equivalent (non-streaming)
template <typename T>
using Task = asio::awaitable<T>;

// Streaming equivalent (IAsyncEnumerable / AsyncIterable)
template <typename T>
class AsyncGenerator; // Custom implementation using C++20 coroutine machinery

// Cancellation
using CancellationToken = std::stop_token;
using CancellationSource = std::stop_source;
```

### AD-2: Content Model — glaze Tagged Variant

```cpp
// Polymorphic content type using std::variant + glaze discriminator
struct TextContent { std::string text; };
struct FunctionCallContent { std::string call_id; std::string name; std::string arguments; };
struct FunctionResultContent { std::string call_id; std::string result; };
struct DataContent { std::string media_type; std::vector<uint8_t> data; };
struct ErrorContent { std::string message; std::string code; };

using AIContent = std::variant<TextContent, FunctionCallContent, FunctionResultContent,
                                DataContent, ErrorContent, UsageContent, AudioContent,
                                ImageContent, UriContent>;

// glaze meta for polymorphic JSON (tagged_variant)
template <> struct glz::meta<AIContent> {
    static constexpr std::string_view tag = "$type";
    static constexpr auto ids = std::array{
        "text", "functionCall", "functionResult", "data", "error",
        "usage", "audio", "image", "uri"
    };
};
```

### AD-3: HTTP Architecture — Beast with Router Layer

Build a lightweight HTTP router on top of Boost.Beast:

```cpp
class HttpRouter {
public:
    void route(http::verb method, std::string_view pattern,
               std::function<Task<HttpResponse>(HttpRequest const&)> handler);
    void use(HttpMiddleware middleware);  // Global middleware
    Task<HttpResponse> dispatch(HttpRequest const& req);
};
```

SSE streaming via chunked response writer:

```cpp
class SseWriter {
public:
    explicit SseWriter(beast::tcp_stream& stream);
    Task<void> write_event(std::string_view event, std::string_view data);
    Task<void> close();
};
```

### AD-4: Dependency Injection — Manual ServiceCollection

```cpp
class ServiceCollection {
public:
    template <typename TInterface, typename TImpl, typename... Args>
    void add_singleton(Args&&... args);
    template <typename TInterface>
    void add_keyed_singleton(std::string_view key, std::shared_ptr<TInterface> instance);
    template <typename T>
    std::shared_ptr<T> get_service() const;
    template <typename T>
    std::shared_ptr<T> get_keyed_service(std::string_view key) const;
};
```

### AD-5: Middleware Pipeline — std::function Onion Model

Three-level middleware matching .NET/Python architecture:

```cpp
// Agent level
using AgentDelegate = std::function<Task<void>(AgentRunContext&)>;
using AgentMiddleware = std::function<Task<void>(AgentRunContext&, AgentDelegate)>;

// Chat level
using ChatDelegate = std::function<Task<void>(ChatContext&)>;
using ChatMiddleware = std::function<Task<void>(ChatContext&, ChatDelegate)>;

// Function invocation level
using FunctionDelegate = std::function<Task<void>(FunctionInvocationContext&)>;
using FunctionMiddleware = std::function<Task<void>(FunctionInvocationContext&, FunctionDelegate)>;

class MiddlewarePipeline<Delegate, Middleware, Context> {
    std::vector<Middleware> middlewares_;
public:
    void use(Middleware mw);
    Delegate build(Delegate terminal);
};
```

### AD-6: Tool/Function System

```cpp
class AITool {
public:
    virtual ~AITool() = default;
    virtual std::string_view name() const = 0;
    virtual std::string_view description() const = 0;
    virtual glz::json_t json_schema() const = 0;
};

class AIFunction : public AITool {
public:
    virtual Task<glz::json_t> invoke_async(glz::json_t const& arguments,
                                            CancellationToken cancel) = 0;
};

// Factory for creating AIFunction from C++ callables
template <typename Callable>
std::shared_ptr<AIFunction> make_function(
    std::string name, std::string description, Callable&& fn);

// Macro for quick tool definition
#define AGENT_TOOL(name, desc, fn) agents::make_function(name, desc, fn)
```

### AD-7: Expression Engine — Lua/sol2 with PowerFx Shim

```cpp
class ExpressionEngine {
public:
    void set_variable(std::string_view name, glz::json_t value);
    Task<glz::json_t> evaluate(std::string_view expression);
    Task<bool> evaluate_condition(std::string_view expression);
    Task<std::string> evaluate_string(std::string_view expression);
};

// PowerFx compatibility translations:
// PowerFx: Set(varName, value)       → Lua: varName = value
// PowerFx: If(cond, then, else)      → Lua: if cond then ... else ... end
// PowerFx: Concatenate(a, b)         → Lua: a .. b
// PowerFx: Text(number)              → Lua: tostring(number)
// PowerFx: Topic.varName             → Lua: Topic.varName (table access)
```

---

## Library Dependency Matrix

### Core Framework (no optional features)

```json
{
    "name": "agent-framework-cpp",
    "version": "1.0.0-preview",
    "dependencies": [
        {"name": "boost-asio"},
        {"name": "boost-beast"},
        {"name": "glaze"},
        {"name": "nlohmann-json"},
        {"name": "spdlog"},
        {"name": "fmt"},
        {"name": "opentelemetry-cpp", "features": ["otlp-http"]},
        {"name": "openssl"}
    ]
}
```

### Full Framework (all features)

Additional dependencies for optional packages:

| Feature | Additional vcpkg Dependency |
|---|---|
| YAML declarative | `yaml-cpp` |
| Expression engine | `lua`, `sol2` |
| Durable tasks | `grpc`, `protobuf` |
| MCP tools | `modelcontextprotocol-cpp-sdk` (or vendored) |
| Azure Identity | `azure-identity-cpp`, `azure-core-cpp` |
| Redis storage | `hiredis` or `redis-plus-plus` |
| Testing | `gtest` |
| Benchmarks | `benchmark` (Google Benchmark) |

---

## Project Structure

```
cpp/
├── CMakeLists.txt                              # Top-level (equivalent to .slnx)
├── CMakePresets.json                           # Build configurations
├── vcpkg.json                                  # Package manifest
├── Directory.Build.cmake                       # Shared properties
├── README.md
├── cmake/
│   ├── AgentFrameworkConfig.cmake.in           # For find_package() support
│   ├── CompilerWarnings.cmake                  # Shared warnings
│   └── Dependencies.cmake                      # Dependency versions
├── src/
│   ├── CMakeLists.txt
│   ├── Agents.Abstractions/                    # ≡ Microsoft.Agents.AI.Abstractions
│   │   ├── CMakeLists.txt
│   │   ├── include/agents/abstractions/
│   │   │   ├── ai_agent.h
│   │   │   ├── agent_session.h
│   │   │   ├── agent_response.h
│   │   │   ├── ai_context_provider.h
│   │   │   ├── chat_history_provider.h
│   │   │   ├── content.h                       # AIContent variant types
│   │   │   ├── chat_message.h
│   │   │   ├── chat_options.h
│   │   │   ├── chat_role.h
│   │   │   ├── ai_tool.h
│   │   │   ├── ai_function.h
│   │   │   └── middleware.h
│   │   └── src/
│   │       └── ...
│   ├── Agents.AI/                              # ≡ Microsoft.Agents.AI
│   │   ├── CMakeLists.txt
│   │   ├── include/agents/ai/
│   │   │   ├── chat_client.h                   # IChatClient interface
│   │   │   ├── chat_client_agent.h
│   │   │   ├── delegating_agent.h
│   │   │   ├── delegating_chat_client.h
│   │   │   ├── agent_builder.h
│   │   │   ├── logging_agent.h
│   │   │   ├── otel_agent.h
│   │   │   ├── function_invocation_agent.h
│   │   │   └── ai_host_agent.h
│   │   └── src/
│   ├── Agents.AI.OpenAI/                       # ≡ Microsoft.Agents.AI.OpenAI
│   │   ├── include/agents/openai/
│   │   │   ├── openai_chat_client.h
│   │   │   └── openai_extensions.h
│   │   └── src/
│   ├── Agents.AI.AzureAI/                      # ≡ Microsoft.Agents.AI.AzureAI
│   ├── Agents.AI.AzureAI.Persistent/
│   ├── Agents.AI.Anthropic/
│   ├── Agents.AI.CopilotStudio/
│   ├── Agents.AI.GitHub.Copilot/
│   ├── Agents.AI.Hosting/                      # HTTP server, router, SSE writer
│   │   ├── include/agents/hosting/
│   │   │   ├── http_router.h
│   │   │   ├── sse_writer.h
│   │   │   ├── agent_host.h
│   │   │   └── agent_host_builder.h
│   │   └── src/
│   ├── Agents.AI.Hosting.OpenAI/               # OpenAI-compatible API server
│   ├── Agents.AI.Hosting.A2A/                  # A2A protocol server
│   ├── Agents.AI.Hosting.AGUI/                 # AG-UI protocol server
│   ├── Agents.AI.Workflows/                    # Workflow engine
│   ├── Agents.AI.Workflows.Declarative/        # YAML + Lua expression engine
│   ├── Agents.AI.DurableTask/                  # gRPC Durable Task client
│   ├── Agents.AI.MCP/                          # MCP tool serving
│   ├── Agents.AI.Storage.CosmosDB/             # Cosmos DB storage
│   ├── Agents.AI.Storage.Redis/                # Redis storage
│   ├── Agents.AI.Mem0/                         # Mem0 memory integration
│   ├── Agents.AI.Purview/                      # DLP middleware
│   └── Agents.AI.DevUI/                        # Developer UI (embedded SPA)
├── samples/
│   ├── CMakeLists.txt
│   ├── GettingStarted/
│   │   ├── Step1_Chat/
│   │   ├── Step2_Chat_Agent/
│   │   ├── Step3_Chat_Agent_Tools/
│   │   └── Step4_Chat_Agent_OpenAI_Hosting/
│   ├── HostedAgents/
│   ├── Durable/
│   └── ...
└── tests/
    ├── CMakeLists.txt
    ├── Agents.Abstractions.UnitTests/
    ├── Agents.AI.UnitTests/
    ├── Agents.AI.OpenAI.UnitTests/
    └── ...
```

---

## Implementation Tiers

Implementation proceeds bottom-up in dependency order:

### Tier 0: Build Infrastructure (Week 1)

* CMake project structure with presets
* vcpkg manifest with all dependencies
* CI pipeline (build + test on Windows/Linux/macOS)
* Compiler warning presets and sanitizer configurations
* Google Test infrastructure

### Tier 1: Async Primitives (Week 1-2)

* `AsyncGenerator<T>` coroutine type (with `co_yield` and pull-based iteration)
* SSE stream parser (`parse_sse_stream` → `AsyncGenerator<SseEvent>`)
* HTTP client wrapper over Boost.Beast (async POST, streaming POST with SSE)
* `AsyncChannel<T>` (thin wrapper over `asio::experimental::channel`)
* `when_all` utility for concurrent operations

### Tier 2: Content & Message Model (Week 2-3)

* `AIContent` variant with all 9 subtypes
* glaze meta definitions for polymorphic JSON serialization with `$type` discriminator
* `ChatMessage`, `ChatRole`, `FinishReason`
* `ChatResponse`, `ChatResponseUpdate`
* `ChatOptions` with all parameters
* `UsageDetails`, `AdditionalProperties`
* `AgentResponse`, `AgentResponseUpdate`
* JSON Schema generation from content/tool types

### Tier 3: Tool System (Week 3-4)

* `AITool` and `AIFunction` interfaces
* `make_function<>()` factory from C++ callables
* JSON Schema generation for function parameters
* Function invocation with argument deserialization and result serialization
* `DelegatingAIFunction` decorator

### Tier 4: Core Agent Abstractions (Week 4-5)

* `IChatClient` interface
* `DelegatingChatClient` base
* `AIAgent` abstract base with `run_async`/`run_streaming_async` (4 overloads each)
* `DelegatingAIAgent` decorator base
* `AgentSession` hierarchy (abstract → in-memory → service-id)
* `ChatHistoryProvider` with two-phase lifecycle
* `AIContextProvider` with two-phase lifecycle
* `AIContext` (transient instructions + messages + tools)
* Three-level middleware pipeline (agent → chat → function)

### Tier 5: ChatClientAgent (Week 5-6)

* `ChatClientAgent` (primary concrete agent — largest single class)
* Session lifecycle management (dual-mode)
* Options merging (instructions, tools, context provider data)
* Function invocation loop (auto-invoke with iteration limit)
* Streaming with function invocation

### Tier 6: AI Provider Clients (Week 6-9)

* OpenAI chat client (REST API → IChatClient)
  * Chat Completions API
  * Responses API
  * Assistants API
  * Streaming (SSE parsing)
  * Tool/function calling
* Azure AI chat client
  * Azure AI Inference API
  * Tool validation, schema transformation
  * Persistent agent sessions
* Anthropic chat client
  * Messages API
  * Streaming
  * Tool use
* Copilot Studio agent (DirectLine protocol)
* GitHub Copilot agent

### Tier 7: Cross-Cutting Agents (Week 9-10)

* `LoggingAgent` (spdlog-based)
* `OpenTelemetryAgent` (GenAI semantic conventions)
* `FunctionInvocationDelegatingAgent`
* `AnonymousDelegatingAIAgent`
* `AIHostAgent`
* `AIAgentBuilder` fluent API

### Tier 8: Hosting Layer (Week 10-13)

* HTTP router (Boost.Beast based)
* SSE writer (chunked response streaming)
* `AgentHost` (listens and dispatches)
* OpenAI-compatible hosting (Chat Completions + Responses + Conversations endpoints)
* A2A protocol server (JSON-RPC 2.0 + SSE + agent card discovery)
* AG-UI protocol server (12 SSE event types)
* A2A client (`A2AAgent` remote proxy)
* AG-UI client (`AGUIChatClient`)

### Tier 9: Workflows (Week 13-16)

* Graph executor (nodes + edges)
* Edge types: `DirectEdge`, `FanOutEdge`, `FanInEdge`
* BuildSequential, BuildConcurrent, handoff, group chat helpers
* Checkpoint store interface + in-memory implementation
* YAML declarative parsing (yaml-cpp)
* Lua expression engine with PowerFx compatibility shim (sol2)
* Variable scoping (Local/Global/System/Environment/Topic)
* `WorkflowAgentProvider` abstraction

### Tier 10: Storage & Integrations (Week 16-18)

* In-memory chat history store
* Cosmos DB storage (REST API via Beast)
* Redis storage (hiredis)
* Mem0 integration (REST)
* Purview DLP middleware (Microsoft Graph REST)
* MCP tool serving (via cpp-sdk)

### Tier 11: Durable Tasks (Week 18-20)

* gRPC Durable Task client SDK (from proto definitions)
* `DurableAIAgent` with entity operations
* `DurableAIAgentProxy` for out-of-orchestration calls
* Composite `AgentSessionId`

### Tier 12: Developer UI & Samples (Week 20-22)

* Embedded developer UI (static files + REST API)
* Entity discovery API
* All getting-started samples
* Hosted agent samples
* Durable agent samples

---

## Risk Assessment

### High Risk

| Risk | Impact | Mitigation |
|---|---|---|
| **PowerFx has no C++ port** | Declarative workflows cannot evaluate expressions natively | Use Lua/sol2 with compatibility shim; accept syntax difference or implement subset parser |
| **No official AI provider C++ SDKs** | Must build REST clients from scratch for OpenAI, Azure AI, Anthropic | Well-documented REST APIs; validate against provider test endpoints |
| **C++20 coroutine ecosystem immaturity** | Custom `AsyncGenerator<T>` needs thorough testing; edge cases in coroutine lifetime | Extensive unit testing; consider vendoring cppcoro fork for when_all/async_mutex |
| **glaze API stability** | Breaking changes in a younger library | Pin version; consider nlohmann/json as fallback for stable features |

### Medium Risk

| Risk | Impact | Mitigation |
|---|---|---|
| **A2A and AG-UI have no C++ SDKs** | Must implement protocols from specification | Wire protocols are well-documented HTTP+SSE+JSON; write compliance tests against .NET/Python servers |
| **Durable Task sidecar protocol complexity** | Entity operations, replay logic, scheduling | Use gRPC client against existing sidecar; proto definitions available at microsoft/durabletask-protobuf |
| **Build time for large C++ project** | 28+ libraries with templates and coroutines | Use precompiled headers, module support (C++20), forward declarations, compile-time optimization |
| **Cross-platform Beast server** | Platform-specific edge cases (Schannel vs OpenSSL) | Test on CI with all 3 platforms; libcurl for client fallback if Beast has platform issues |

### Low Risk

| Risk | Impact | Mitigation |
|---|---|---|
| **CMake/vcpkg infrastructure** | Well-established, well-documented | Follow vcpkg manifest best practices |
| **Google Test adoption** | Industry standard | Straightforward |
| **OpenTelemetry C++ SDK** | Production-ready | Attribute-based API matches GenAI conventions |
| **yaml-cpp** | Mature, stable | No concerns |

---

## Research Executed

### File Analysis

* `dotnet/src/Microsoft.Agents.AI.Abstractions/` — All .cs files: AIAgent abstract base (RunAsync 4 overloads, RunCoreAsync/RunCoreStreamingAsync), AgentSession hierarchy, AgentResponse<T>, ChatHistoryProvider, AIContextProvider, AgentSessionStore, enums (FailureReason, ChatReducerTriggerEvent, TextSearchBehavior, SearchBehavior)
* `dotnet/src/Microsoft.Agents.AI/` — ChatClientAgent (910 LOC), AIAgentBuilder, DelegatingAIAgent, LoggingAgent, OpenTelemetryAgent, FunctionInvocationDelegatingAgent
* `dotnet/src/Microsoft.Agents.AI.OpenAI/` — Extension methods AsAIAgent() on OpenAI SDK types, IChatClient bridge
* `dotnet/src/Microsoft.Agents.AI.AzureAI/` — DelegatingChatClient, tool validation, schema transforms
* `dotnet/src/Microsoft.Agents.AI.Anthropic/` — IChatClient bridge for Anthropic SDK
* `dotnet/src/Microsoft.Agents.AI.CopilotStudio/` — Direct AIAgent subclass with DirectLine
* `dotnet/src/Microsoft.Agents.AI.GitHub.Copilot/` — Direct AIAgent subclass with Channel streaming
* `dotnet/src/Microsoft.Agents.AI.A2A/` — A2AAgent, A2AAgentSession, SSE streaming, continuation tokens
* `dotnet/src/Microsoft.Agents.AI.AGUI/` — AGUIChatClient (DelegatingChatClient), 12 SSE event types
* `dotnet/src/Microsoft.Agents.AI.Hosting.OpenAI/` — 3 API surfaces, 11 StreamingEventGenerator strategies
* `dotnet/src/Microsoft.Agents.AI.Hosting.A2A.AspNetCore/` — MapA2A(), TaskManager
* `dotnet/src/Microsoft.Agents.AI.Hosting.AGUI.AspNetCore/` — AGUIServerSentEventsResult
* `dotnet/src/Microsoft.Agents.AI.Hosting.AzureFunctions/` — IFunctionMetadataTransformer, MCP tool triggers
* `dotnet/src/Microsoft.Agents.AI.Workflows/` — Graph executor, edges, checkpointing
* `dotnet/src/Microsoft.Agents.AI.Workflows.Declarative/` — PowerFx RecalcEngine, variable scopes
* `dotnet/src/Microsoft.Agents.AI.DurableTask/` — DurableAIAgent, DurableAIAgentProxy, entity state
* `dotnet/src/Microsoft.Agents.AI.CosmosNoSql/` — ChatHistoryProvider + CheckpointStore
* `dotnet/src/Microsoft.Agents.AI.Mem0/` — AIContextProvider backed by Mem0 REST
* `dotnet/src/Microsoft.Agents.AI.Purview/` — DLP middleware, Graph API
* `dotnet/src/Microsoft.Agents.AI.DevUI/` — Embedded SPA, entity API
* `python/packages/core/` — AgentProtocol, BaseAgent, ChatAgent, Content (18 types), ChatMessage, tools, middleware, OpenTelemetry
* `python/packages/a2a/` — A2A protocol interop
* `python/packages/ag-ui/` — AG-UI protocol (FastAPI SSE)
* `python/packages/azure-ai/` — Azure AI client
* `python/packages/anthropic/` — Anthropic client
* `python/packages/declarative/` — YAML agents/workflows with PowerFx
* `python/packages/durabletask/` — Durable entities via gRPC

### Code Search Results

* `using Microsoft.Extensions.AI` — 25+ types used across all .NET projects
* `IChatClient` implementations — 6 providers, 2 integration tiers
* SSE streaming — all hosting projects use `text/event-stream` with `no-cache,no-store`
* Source-generated JSON — `JsonSerializerContext` with chained resolvers in every project
* Keyed DI — `GetRequiredKeyedService<AIAgent>` for agent resolution

### External Research

* C++ HTTP libraries — 9 libraries evaluated (cpp-httplib, Boost.Beast, libcurl, Drogon, Crow, Pistache, cpprestsdk, Poco, Azure SDK C++)
  * Source: GitHub repositories, TechEmpower benchmarks
* C++ JSON libraries — 5 libraries evaluated (nlohmann/json, glaze, simdjson, RapidJSON, Boost.JSON)
  * Source: GitHub repositories, benchmark comparisons
* C++ async/coroutines — cppcoro, folly::coro, Boost.Asio, custom implementations
  * Source: C++20 standard, library documentation
* Protocol SDKs — A2A (none for C++), AG-UI (none for C++), MCP (official cpp-sdk exists), gRPC (official)
  * Source: GitHub repositories for google/A2A, CopilotKit/ag-ui, modelcontextprotocol/cpp-sdk
* PowerFx — no C++ port exists; Lua/sol2 identified as best alternative
  * Source: microsoft/Power-Fx repository, sol2 documentation

### Project Conventions

* Standards referenced: C++20, ISO C++20 coroutines, Boost Asio design
* Instructions followed: `.github/copilot-instructions.md` (copyright headers, documentation, build verification, format)
