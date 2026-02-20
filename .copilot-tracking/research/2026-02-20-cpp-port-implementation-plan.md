<!-- markdownlint-disable-file -->
# Implementation Plan: C++ Port of Agent Framework

Full C++ reimplementation of the Microsoft Agent Framework with feature parity across all .NET (27 projects, ~71K LOC) and Python (20 packages, ~79K LOC) components. Organized as phased milestones with explicit gate criteria.

## Decisions Log

| Decision | Choice | Rationale |
|---|---|---|
| Scope | Phased with milestone gates | Full parity over ~22 weeks, each phase builds on previous with quality gates |
| Expression engine | Lua/sol2 with PowerFx compatibility shim | See research AD-7 |
| CI/CD | GitHub Actions | Matrix builds MSVC, GCC 11+, Clang 14+ on Windows/Linux/macOS |
| Azure Functions hosting | Standalone daemon | Custom HTTP daemon mirroring trigger patterns; Azure Functions not directly portable to C++ |
| C++ Standard | C++20 minimum | Coroutines, concepts, ranges, `std::format`, `std::stop_token` |
| Build system | CMake 3.25+ with vcpkg manifest mode | Industry standard, FILE_SET support |
| Primary JSON | glaze 4.x | Tagged variant polymorphism, JSON Schema gen, 10x faster than nlohmann |
| HTTP / Async | Boost.Beast + Boost.Asio | Unified async model, no second event loop |
| Testing | Google Test + gmock 1.15+ | Industry standard |

## References

- Research document: [2026-02-20-cpp-port-research.md](2026-02-20-cpp-port-research.md)
- .NET structure analysis: [dotnet-structure-analysis.md](../subagent/2026-02-20/dotnet-structure-analysis.md)
- Python structure analysis: [python-structure-analysis.md](../subagent/2026-02-20/python-structure-analysis.md)
- Hosting protocols analysis: [hosting-protocols-analysis.md](../subagent/2026-02-20/hosting-protocols-analysis.md)

---

## Milestone Overview

| Phase | Name | Duration | Cumulative | C++ Libraries Delivered | Gate |
|---|---|---|---|---|---|
| 0 | Build Infrastructure | Week 1 | Week 1 | (tooling only) | CI green on 3 compilers × 3 OS |
| 1 | Async Primitives | Weeks 2–3 | Week 3 | `agents-async` | Unit tests for `AsyncGenerator`, SSE parser, HTTP client pass on all platforms |
| 2 | Content & Message Model | Weeks 3–4 | Week 4 | `agents-abstractions` | Round-trip JSON serialization tests for all 9 content types, polymorphic discriminator |
| 3 | Tool System | Week 5 | Week 5 | (extends `agents-abstractions`) | `make_function<>()` factory, JSON Schema generation, invocation tests |
| 4 | Core Agent Abstractions | Weeks 5–7 | Week 7 | `agents-ai` | `AIAgent`, `ChatClientAgent`, 3-level middleware, session lifecycle, decorator chain — all unit-tested |
| 5 | First Provider (OpenAI) | Weeks 7–9 | Week 9 | `agents-openai` | Chat Completions + Responses API, streaming, tool calling — integration tests against OpenAI API |
| 6 | Additional Providers | Weeks 9–12 | Week 12 | `agents-azure-ai`, `agents-anthropic`, `agents-copilotstudio`, `agents-github-copilot` | Each provider passes integration tests |
| 7 | Cross-Cutting Agents | Week 12 | Week 12 | (extends `agents-ai`) | `LoggingAgent`, `OpenTelemetryAgent`, `FunctionInvocationDelegatingAgent`, builder API |
| 8 | Hosting Layer | Weeks 13–16 | Week 16 | `agents-hosting`, `agents-hosting-openai`, `agents-hosting-a2a`, `agents-hosting-agui` | Interop tests: C++ server ↔ .NET/Python client for each protocol |
| 9 | Workflows | Weeks 16–19 | Week 19 | `agents-workflows`, `agents-workflows-declarative` | Graph executor + YAML-driven agent with Lua expressions, checkpoint round-trip |
| 10 | Storage & Integrations | Weeks 19–20 | Week 20 | `agents-storage-cosmos`, `agents-storage-redis`, `agents-mem0`, `agents-purview`, `agents-mcp` | Storage round-trip tests, MCP tool serving |
| 11 | Durable Tasks | Weeks 20–21 | Week 21 | `agents-durabletask`, `agents-hosting-daemon` | gRPC client, entity operations, standalone daemon integration test |
| 12 | Developer UI & Samples | Weeks 21–22 | Week 22 | `agents-devui` + all samples | All getting-started samples build and run, DevUI embedded SPA serves |

---

## Phase 0: Build Infrastructure (Week 1)

### Objective

Establish the CMake project skeleton, vcpkg manifest, CI pipeline, and coding standards so all subsequent phases can immediately compile, test, and distribute.

### Deliverables

| # | Task | Details | Effort |
|---|---|---|---|
| 0.1 | **Top-level CMake structure** | Create `cpp/CMakeLists.txt`, `cpp/CMakePresets.json` (Debug, Release, RelWithDebInfo, ASAN, TSAN configurations), `cpp/cmake/CompilerWarnings.cmake`, `cpp/cmake/Dependencies.cmake` | 0.5d |
| 0.2 | **vcpkg manifest** | `cpp/vcpkg.json` with all core dependencies: `boost-asio`, `boost-beast`, `glaze`, `nlohmann-json`, `spdlog`, `fmt`, `opentelemetry-cpp`, `openssl`, `gtest`. Feature flags for optional deps: `yaml-cpp`, `sol2`, `grpc`, `hiredis` | 0.5d |
| 0.3 | **Directory scaffold** | Create all `cpp/src/<LibraryName>/CMakeLists.txt` stubs (27 libraries mapping to .NET projects), empty `include/` and `src/` directories, `cpp/tests/` stubs, `cpp/samples/` stubs. Use the project structure from the research document | 0.5d |
| 0.4 | **Shared CMake modules** | `cpp/cmake/AgentFrameworkConfig.cmake.in` for `find_package()` support; `cpp/Directory.Build.cmake` for shared properties (C++20 standard, output directories, install rules) | 0.5d |
| 0.5 | **GitHub Actions CI** | `.github/workflows/cpp-build.yml`: matrix strategy for `{msvc-latest, gcc-11, clang-14}` × `{windows-latest, ubuntu-latest, macos-latest}` (9 combinations, skip invalid combos). Steps: checkout → vcpkg bootstrap → cmake configure → build → test → upload artifacts | 1d |
| 0.6 | **Code quality tooling** | clang-tidy configuration (`.clang-tidy`), clang-format (`.clang-format` matching project style), pre-commit hooks, sanitizer presets (ASAN, TSAN, UBSAN) | 0.5d |
| 0.7 | **Copyright header template** | `// Copyright (c) Microsoft. All rights reserved.` in all `.h`/`.cpp` files. Create a check script or clang-tidy check | 0.25d |
| 0.8 | **README.md** | `cpp/README.md` with build instructions, dependency list, architecture overview | 0.25d |

### File Tree

```text
cpp/
├── CMakeLists.txt
├── CMakePresets.json
├── vcpkg.json
├── Directory.Build.cmake
├── README.md
├── .clang-tidy
├── .clang-format
├── cmake/
│   ├── AgentFrameworkConfig.cmake.in
│   ├── CompilerWarnings.cmake
│   └── Dependencies.cmake
├── src/
│   ├── CMakeLists.txt
│   └── (27 library stubs with CMakeLists.txt each)
├── tests/
│   └── CMakeLists.txt
└── samples/
    └── CMakeLists.txt
```

### Gate Criteria

- [ ] `cmake --preset release` succeeds on MSVC, GCC 11, Clang 14
- [ ] `ctest` runs (even with 0 tests) on all 3 platforms
- [ ] CI pipeline green for all valid matrix entries
- [ ] vcpkg dependencies resolve without manual intervention
- [ ] clang-format and clang-tidy pass on scaffold files

---

## Phase 1: Async Primitives (Weeks 2–3)

### Objective

Build the foundational async types that every other component depends on: `Task<T>`, `AsyncGenerator<T>`, SSE stream parser, HTTP client wrapper, and concurrency utilities.

### Deliverables

| # | Task | Details | Maps To (.NET / Python) | Effort |
|---|---|---|---|---|
| 1.1 | **`Task<T>` alias and utilities** | `using Task<T> = asio::awaitable<T>`. Utility: `when_all()` wrapping `asio::experimental::make_parallel_group()`. `CancellationToken` = `std::stop_token` | `Task<T>` / `Coroutine[T]`, `Task.WhenAll` / `asyncio.gather` | 1d |
| 1.2 | **`AsyncGenerator<T>`** | Custom C++20 coroutine type supporting `co_yield`, pull-based async iteration via `co_await generator.next()`, proper RAII cleanup of coroutine handle, cancellation via `std::stop_token` | `IAsyncEnumerable<T>` / `AsyncIterable[T]` | 3d |
| 1.3 | **`AsyncChannel<T>`** | Thin wrapper over `asio::experimental::channel<void(error_code, T)>` with `send(T)`, `receive() -> Task<T>`, `close()` | `Channel<T>` / `asyncio.Queue` | 0.5d |
| 1.4 | **`AsyncSemaphore`** | Async-aware semaphore for concurrency limiting (Beast connection pools, fan-out) | `SemaphoreSlim` / `asyncio.Semaphore` | 0.5d |
| 1.5 | **HTTP client wrapper** | `class HttpClient` over Boost.Beast: async `post_json()`, `get()`, streaming `post_streaming()` → `AsyncGenerator<std::string_view>` (for SSE). Connection pooling, TLS via OpenSSL, timeout support, `std::stop_token` cancellation | `HttpClient` / `httpx.AsyncClient` | 3d |
| 1.6 | **SSE stream parser** | `parse_sse_stream(AsyncGenerator<std::string_view>) -> AsyncGenerator<SseEvent>`. `struct SseEvent { std::string event; std::string data; std::optional<std::string> id; }`. Handles multi-line `data:` fields, `event:` prefixes, blank-line delimiters per SSE spec | SSE parsing in all hosting projects | 1.5d |
| 1.7 | **Unit tests** | Google Test suite for all types: `AsyncGenerator` edge cases (empty, single, cancellation, exception propagation), SSE parser (multi-line, no event field, keepalive comments), HttpClient (mock server), `when_all` (success + partial failure) | — | 2d |

### Key Design Decisions

```cpp
// cpp/src/Agents.Async/include/agents/async/task.h
namespace agents::async {
    template <typename T>
    using Task = asio::awaitable<T>;

    using CancellationToken = std::stop_token;
    using CancellationSource = std::stop_source;
}

// cpp/src/Agents.Async/include/agents/async/async_generator.h
namespace agents::async {
    template <typename T>
    class AsyncGenerator {
    public:
        struct promise_type; // C++20 coroutine promise
        struct iterator;

        // Pull-based iteration
        Task<std::optional<T>> next();

        // Range-for support (if needed for sync contexts)
        iterator begin();
        iterator end();
    };
}
```

### Gate Criteria

- [ ] `AsyncGenerator<T>` passes: empty generator, single yield, multi-yield, exception propagation, early cancellation, coroutine cleanup
- [ ] SSE parser passes: standard events, multi-line data, missing event field, keepalive empty comments, UTF-8 data
- [ ] HTTP client makes real HTTPS request (integration test with a mock server)
- [ ] `when_all` handles both all-success and partial-failure cases
- [ ] All tests pass on MSVC, GCC, Clang
- [ ] No sanitizer warnings (ASAN, TSAN)

---

## Phase 2: Content & Message Model (Weeks 3–4)

### Objective

Implement the ~25 types from `Microsoft.Extensions.AI` that form the content/message foundation used by every agent and provider.

### Deliverables

| # | Task | Details | Maps To | Effort |
|---|---|---|---|---|
| 2.1 | **Content variant types** | 9 content structs: `TextContent`, `FunctionCallContent`, `FunctionResultContent`, `DataContent`, `ErrorContent`, `UsageContent`, `AudioContent`, `ImageContent`, `UriContent`. Each with `glz::meta` for `$type` discriminated union | `AIContent` subtypes / `Content` class | 2d |
| 2.2 | **`AIContent` variant** | `using AIContent = std::variant<TextContent, ...>` with glaze `tagged_variant` meta for polymorphic JSON via `$type` discriminator | `AIContent` abstract / `Content` | 0.5d |
| 2.3 | **`ChatMessage`** | `struct ChatMessage { ChatRole role; std::vector<AIContent> contents; std::string author_name; AdditionalProperties additional_properties; }` | `ChatMessage` | 0.5d |
| 2.4 | **`ChatRole`** | `enum class ChatRole { System, User, Assistant, Tool }` with string conversion | `ChatRole` / `Role` | 0.25d |
| 2.5 | **`ChatResponse`** | `struct ChatResponse { ChatMessage message; UsageDetails usage; std::optional<FinishReason> finish_reason; std::string model_id; std::string response_id; AdditionalProperties additional_properties; }` | `ChatResponse` / `ChatResponse` | 0.5d |
| 2.6 | **`ChatResponseUpdate`** | Streaming chunk: `struct ChatResponseUpdate { std::optional<ChatRole> role; std::vector<AIContent> contents; std::optional<FinishReason> finish_reason; std::string model_id; ... }` | `ChatResponseUpdate` | 0.5d |
| 2.7 | **`ChatOptions`** | `struct ChatOptions { std::optional<std::string> model_id; std::optional<float> temperature; std::optional<float> top_p; std::optional<int> max_output_tokens; std::vector<std::shared_ptr<AITool>> tools; ... }` | `ChatOptions` | 0.5d |
| 2.8 | **`UsageDetails`** | `struct UsageDetails { std::optional<int> input_tokens; std::optional<int> output_tokens; std::optional<int> total_tokens; AdditionalProperties additional_properties; }` | `UsageDetails` | 0.25d |
| 2.9 | **`FinishReason`** | `enum class FinishReason { Stop, Length, ToolCalls, ContentFilter, Error }` | `FinishReason` | 0.25d |
| 2.10 | **`AgentResponse` / `AgentResponseUpdate`** | `struct AgentResponse<T> { std::string agent_id; T response; FailureReason failure_reason; }` and streaming variant | `AgentResponse<T>` / `AgentResponse` | 0.5d |
| 2.11 | **`AdditionalProperties`** | `using AdditionalProperties = std::unordered_map<std::string, glz::json_t>` for extension data | `AdditionalPropertiesDictionary` | 0.25d |
| 2.12 | **`ResponseContinuationToken`** | `struct ResponseContinuationToken { std::string token; }` for multi-turn continuation | `ResponseContinuationToken` | 0.25d |
| 2.13 | **JSON serialization tests** | Round-trip tests for every type (serialize → deserialize → compare). Polymorphic discriminator tests: mixed content arrays. Cross-compatibility test: deserialize JSON produced by .NET serializer | — | 2d |

### Library: `agents-abstractions`

**Header location:** `cpp/src/Agents.Abstractions/include/agents/abstractions/`

```cpp
// content.h — core snippet
namespace agents {
    struct TextContent {
        std::string text;
    };

    struct FunctionCallContent {
        std::string call_id;
        std::string name;
        std::string arguments; // JSON string
    };

    // ... 7 more content types ...

    using AIContent = std::variant<
        TextContent, FunctionCallContent, FunctionResultContent,
        DataContent, ErrorContent, UsageContent,
        AudioContent, ImageContent, UriContent>;
}

template <>
struct glz::meta<agents::AIContent> {
    static constexpr std::string_view tag = "$type";
    static constexpr auto ids = std::array{
        "text"sv, "functionCall"sv, "functionResult"sv, "data"sv,
        "error"sv, "usage"sv, "audio"sv, "image"sv, "uri"sv
    };
};
```

### Gate Criteria

- [ ] All 9 content types serialize/deserialize with `$type` discriminator
- [ ] Mixed `std::vector<AIContent>` round-trips through JSON correctly
- [ ] `ChatMessage` with different roles and mixed content types serializes correctly
- [ ] `ChatResponse` and `ChatResponseUpdate` round-trip tests pass
- [ ] .NET-produced JSON can be deserialized by C++ (cross-compatibility)
- [ ] All tests pass on MSVC, GCC, Clang
- [ ] No memory leaks under ASAN

---

## Phase 3: Tool System (Week 5)

### Objective

Implement the AI tool/function abstraction, the factory for creating tools from C++ callables, and JSON Schema generation for function parameters.

### Deliverables

| # | Task | Details | Maps To | Effort |
|---|---|---|---|---|
| 3.1 | **`AITool` base class** | `class AITool { virtual string_view name(); virtual string_view description(); virtual json_t json_schema(); }` | `AITool` / `ToolProtocol` | 0.5d |
| 3.2 | **`AIFunction` class** | Extends `AITool` with `virtual Task<json_t> invoke_async(json_t const& args, CancellationToken)` | `AIFunction` / `FunctionTool` | 0.5d |
| 3.3 | **`make_function<>()` factory** | Template factory that creates `AIFunction` from any C++ callable. Extracts parameter names via macro or registration, generates JSON Schema for parameters. Supports `std::string`, `int`, `double`, `bool`, `json_t`, `std::optional<T>` parameter types | `AIFunctionFactory.Create()` / auto-schema from signatures | 3d |
| 3.4 | **JSON Schema generation** | `generate_json_schema<T>()` template function using `glz::write_json_schema<T>()`. Handles nested objects, arrays, enums. Used by `make_function` and by hosting layers to advertise tool schemas | `AIFunctionFactory` schema gen / Pydantic `model_json_schema()` | 1d |
| 3.5 | **`DelegatingAIFunction`** | Decorator for functions (pre/post processing) | `DelegatingAIFunction` | 0.25d |
| 3.6 | **`AGENT_TOOL` macro** | Convenience macro: `AGENT_TOOL("get_weather", "Gets the weather", get_weather_fn)` expands to `make_function(...)` | `@tool` decorator | 0.25d |
| 3.7 | **Unit tests** | Tests for: factory with different callable signatures, JSON Schema correctness, invocation with valid/invalid args, error handling, cancellation | — | 1.5d |

### Key Design

```cpp
// Tool factory — the hardest part is C++ reflection limitations
// Option A: Macro-based parameter registration
auto weather_tool = agents::make_function(
    "get_weather",
    "Gets the current weather for a location",
    [](std::string location, std::optional<std::string> units) -> Task<std::string> {
        co_return "72°F in " + location;
    },
    agents::params("location", "The city name")("units", "Temperature units (F/C)")
);

// Option B: Struct-based parameters with glaze reflection
struct GetWeatherParams {
    std::string location;
    std::optional<std::string> units;
};
auto weather_tool = agents::make_function<GetWeatherParams>(
    "get_weather", "Gets the current weather", handler_fn);
```

### Gate Criteria

- [ ] `make_function` creates working tools from lambdas, free functions, and struct-based params
- [ ] Generated JSON Schema matches OpenAI function calling format
- [ ] Invocation correctly deserializes JSON arguments and serializes results
- [ ] `DelegatingAIFunction` correctly wraps and delegates
- [ ] Error cases: missing required params, wrong types, cancellation

---

## Phase 4: Core Agent Abstractions (Weeks 5–7)

### Objective

Implement the entire agent class hierarchy, session management, middleware pipeline, context providers, and chat history providers. This is the largest single phase (maps to ~6,900 LOC of .NET Abstractions + AI).

### Deliverables

| # | Task | Details | Maps To (.NET) | Est. C++ LOC | Effort |
|---|---|---|---|---|---|
| 4.1 | **`IChatClient` interface** | Pure virtual: `get_response(messages, options, cancel) -> Task<ChatResponse>`, `get_streaming_response(messages, options, cancel) -> AsyncGenerator<ChatResponseUpdate>` | `IChatClient` | 50 | 0.5d |
| 4.2 | **`DelegatingChatClient`** | Holds `shared_ptr<IChatClient> inner_client_`; forwards by default, subclasses override to intercept | `DelegatingChatClient` | 80 | 0.5d |
| 4.3 | **`AIAgent` abstract base** | 4 `run_async` overloads (with/without session, with/without options), 2 core virtuals: `run_core_async`, `run_core_streaming_async`. `GetService()` template method | `AIAgent` (abstract) | 300 | 1.5d |
| 4.4 | **`DelegatingAIAgent`** | Holds `shared_ptr<AIAgent> inner_agent_`; decorator base. All calls forward to inner by default | `DelegatingAIAgent` | 150 | 0.5d |
| 4.5 | **`AgentSession` hierarchy** | Abstract `AgentSession`, concrete `InMemoryAgentSession` (client-side chat history), `ServiceIdAgentSession` (server-managed with ConversationId) | `AgentSession` hierarchy | 250 | 1d |
| 4.6 | **`ChatHistoryProvider`** | Two-phase lifecycle: `invoking_async(messages, options)` and `invoked_async(messages, response)`. In-memory default implementation | `ChatHistoryProvider` | 150 | 0.5d |
| 4.7 | **`AIContextProvider`** | Two-phase lifecycle: `invoking_async(context)` and `invoked_async(context)`. Injects instructions, messages, tools per invocation | `AIContextProvider` | 100 | 0.5d |
| 4.8 | **`AIContext`** | Transient context bag: additional instructions, messages, tools collected from providers | `AIContext` | 50 | 0.25d |
| 4.9 | **Three-level middleware pipeline** | Templated `MiddlewarePipeline<Context, Delegate, Middleware>`. Agent-level, chat-level, function-level. Onion model built inside-out | 3-level middleware | 200 | 1.5d |
| 4.10 | **`ChatClientAgent`** | **Largest single class.** Wraps `IChatClient`. Session lifecycle, options merging (instructions, tools, context provider data), function invocation loop (auto-invoke with iteration limit), streaming with function invocation. ~910 LOC in .NET, expect ~1200 LOC in C++ | `ChatClientAgent` (910 LOC) | 1200 | 4d |
| 4.11 | **`ServiceCollection` (minimal DI)** | Singleton/keyed registration, resolution by type or key. Needed for agent builder and hosting | `IServiceCollection` | 200 | 1d |
| 4.12 | **`AIAgentBuilder`** | Fluent builder: `.with_chat_client()`, `.with_instructions()`, `.with_tools()`, `.with_middleware()`, `.with_session_store()`, `.build()` | `AIAgentBuilder` | 150 | 0.5d |
| 4.13 | **Unit tests** | Comprehensive tests per component. Mock `IChatClient` for `ChatClientAgent` tests. Middleware ordering tests. Session lifecycle tests | — | — | 3d |

### Key Architecture

```cpp
// AIAgent abstract base (simplified)
namespace agents {
    class AIAgent {
    public:
        virtual ~AIAgent() = default;

        // Public API — 4 overloads
        Task<AgentResponse<ChatResponse>> run_async(
            std::vector<ChatMessage> messages,
            AgentSession* session = nullptr,
            ChatOptions options = {},
            CancellationToken cancel = {});

        AsyncGenerator<AgentResponseUpdate> run_streaming_async(
            std::vector<ChatMessage> messages,
            AgentSession* session = nullptr,
            ChatOptions options = {},
            CancellationToken cancel = {});

        // Service locator (matches .NET pattern)
        template <typename T>
        std::shared_ptr<T> get_service() const;

    protected:
        // Subclass implements these
        virtual Task<ChatResponse> run_core_async(
            std::vector<ChatMessage>& messages,
            ChatOptions& options,
            CancellationToken cancel) = 0;

        virtual AsyncGenerator<ChatResponseUpdate> run_core_streaming_async(
            std::vector<ChatMessage>& messages,
            ChatOptions& options,
            CancellationToken cancel) = 0;
    };
}
```

### Gate Criteria

- [ ] `ChatClientAgent` passes: single-turn, multi-turn, tool invocation loop (1 round, 3 rounds, max iterations), streaming, streaming with tools, cancellation
- [ ] Middleware pipeline executes in correct order (onion model verified)
- [ ] `DelegatingAIAgent` chain: `LoggingAgent → OtelAgent → ChatClientAgent` works
- [ ] Session lifecycle: `InMemoryAgentSession` accumulates messages correctly
- [ ] `ServiceIdAgentSession` creates and manages conversation IDs
- [ ] `AIContextProvider` injects context at invocation time
- [ ] `AIAgentBuilder` produces a correctly configured agent
- [ ] All tests pass on all 3 compilers

---

## Phase 5: First Provider — OpenAI (Weeks 7–9)

### Objective

Implement the first AI provider client (OpenAI) to validate the entire stack end-to-end: HTTP client → SSE parsing → IChatClient → ChatClientAgent → AgentResponse.

### Deliverables

| # | Task | Details | Effort |
|---|---|---|---|
| 5.1 | **OpenAI Chat Completions client** | `class OpenAIChatClient : public IChatClient`. REST POST to `https://api.openai.com/v1/chat/completions`. Non-streaming: parse full JSON response. Streaming: SSE parse → `AsyncGenerator<ChatResponseUpdate>` | 2d |
| 5.2 | **OpenAI Responses API client** | POST to `/v1/responses`. 19 SSE event types for streaming. Response lifecycle (created → in_progress → completed/failed) | 2d |
| 5.3 | **OpenAI Assistants API client** | Thread/message/run lifecycle. Polling or streaming for run status | 2d |
| 5.4 | **Azure OpenAI variant** | `class AzureOpenAIChatClient : public OpenAIChatClient`. Override base URL, add `api-version` query param, Azure AD token auth via `azure-identity-cpp` | 1d |
| 5.5 | **Tool/function calling** | Serialize `AITool` vector into OpenAI `tools` array format. Parse `tool_calls` from response. Feed `FunctionResultContent` back. Auto-invoke loop in `ChatClientAgent` | 1d |
| 5.6 | **Extension method pattern** | `openai_as_chat_client(api_key, model)` factory function that creates a configured `OpenAIChatClient`. `openai_as_agent(api_key, model, instructions)` that creates a full `ChatClientAgent` via builder | 0.5d |
| 5.7 | **Request/response models** | All OpenAI-specific request/response structs with glaze serialization: `CreateChatCompletion`, `ChatCompletion`, `ChatCompletionChunk`, `CreateResponse`, `Response`, tool types, etc. (~50 structs) | 2d |
| 5.8 | **Integration tests** | Tests against real OpenAI API (gated by `OPENAI_API_KEY` env var). Also mock-server unit tests for deterministic testing | 2d |

### Provider Architecture Pattern

```cpp
// Pattern for all providers: factory → IChatClient → ChatClientAgent
namespace agents::openai {
    // Creates IChatClient from OpenAI config
    std::shared_ptr<IChatClient> create_chat_client(OpenAIConfig config);

    // Convenience: creates a ChatClientAgent directly
    std::shared_ptr<AIAgent> create_agent(OpenAIConfig config, std::string instructions);

    struct OpenAIConfig {
        std::string api_key;
        std::string model = "gpt-4o";
        std::optional<std::string> base_url;       // For Azure OpenAI
        std::optional<std::string> api_version;     // For Azure OpenAI
        std::optional<std::string> organization;
    };
}
```

### Gate Criteria

- [ ] Non-streaming chat completion returns valid `ChatResponse` with content and usage
- [ ] Streaming chat completion yields `ChatResponseUpdate` chunks with correct ordering
- [ ] Tool calling: agent sends function definitions, receives tool_calls, invokes functions, sends results, gets final response
- [ ] Azure OpenAI variant works with Azure AD authentication
- [ ] Responses API streaming handles all 19 event types
- [ ] Integration tests pass against live OpenAI API
- [ ] Mock-server unit tests pass on all compilers

---

## Phase 6: Additional Providers (Weeks 9–12)

### Objective

Implement remaining AI provider integrations. Each follows the same pattern established in Phase 5.

### 6A: Azure AI (1.5 weeks)

| # | Task | Details | Effort |
|---|---|---|---|
| 6A.1 | **Azure AI Inference client** | `class AzureAIChatClient : public IChatClient`. REST calls to Azure AI Inference API. DelegatingChatClient pattern for tool validation and schema transformation (matches .NET's complex Azure AI adapter) | 2d |
| 6A.2 | **Persistent agent sessions** | `class AzureAIAgentSession : public ServiceIdAgentSession`. Server-managed threads via Azure AI Agent Service REST API | 1d |
| 6A.3 | **Tool validation & schema transforms** | Azure AI has stricter tool schema requirements than OpenAI. Transform tool definitions to comply | 1d |
| 6A.4 | **Azure identity integration** | Use `azure-identity-cpp` for `DefaultAzureCredential`. Token caching and refresh | 0.5d |
| 6A.5 | **Integration tests** | Against Azure AI endpoint (gated by env vars) | 1d |

### 6B: Anthropic (0.5 week)

| # | Task | Details | Effort |
|---|---|---|---|
| 6B.1 | **Anthropic Messages API client** | `class AnthropicChatClient : public IChatClient`. REST POST to `https://api.anthropic.com/v1/messages`. Handle Anthropic-specific content blocks | 1d |
| 6B.2 | **Streaming** | SSE with Anthropic event types: `message_start`, `content_block_start`, `content_block_delta`, `content_block_stop`, `message_delta`, `message_stop` | 0.5d |
| 6B.3 | **Tool use** | Anthropic tool_use/tool_result content blocks ↔ `FunctionCallContent`/`FunctionResultContent` | 0.5d |
| 6B.4 | **Tests** | Integration + mock | 0.5d |

### 6C: Copilot Studio (0.5 week)

| # | Task | Details | Effort |
|---|---|---|---|
| 6C.1 | **DirectLine protocol client** | `class CopilotStudioAgent : public AIAgent`. Direct subclass (not IChatClient-based). DirectLine REST API for conversation start, activity posting, response polling | 1.5d |
| 6C.2 | **Session management** | `CopilotStudioAgentSession` with DirectLine conversation ID | 0.5d |
| 6C.3 | **Tests** | Mock DirectLine server | 0.5d |

### 6D: GitHub Copilot (0.5 week)

| # | Task | Details | Effort |
|---|---|---|---|
| 6D.1 | **Channel-based streaming agent** | `class GitHubCopilotAgent : public AIAgent`. Direct subclass with channel-based streaming | 1d |
| 6D.2 | **Session management** | `GitHubCopilotAgentSession` | 0.5d |
| 6D.3 | **Tests** | Mock server | 0.5d |

### Gate Criteria (per provider)

- [ ] Non-streaming and streaming responses work
- [ ] Tool calling round-trips correctly
- [ ] Provider-specific session management works
- [ ] Integration tests pass against live endpoint (when env vars set)
- [ ] Mock tests pass on all compilers

---

## Phase 7: Cross-Cutting Agents (Week 12)

### Objective

Implement the decorator agents and the builder API. These are thin but important for the complete developer experience.

### Deliverables

| # | Task | Details | Maps To (.NET) | Effort |
|---|---|---|---|---|
| 7.1 | **`LoggingAgent`** | `DelegatingAIAgent` that logs invocations/responses via spdlog | `LoggingAgent` | 0.5d |
| 7.2 | **`OpenTelemetryAgent`** | `DelegatingAIAgent` that records OpenTelemetry spans with GenAI semantic conventions (`gen_ai.system`, `gen_ai.request.model`, `gen_ai.usage.input_tokens`, etc.) | `OpenTelemetryAgent` | 1d |
| 7.3 | **`FunctionInvocationDelegatingAgent`** | `DelegatingAIAgent` that intercepts function invocations for approval, logging, or modification | `FunctionInvocationDelegatingAgent` | 0.5d |
| 7.4 | **`AnonymousDelegatingAIAgent`** | `DelegatingAIAgent` initialized with lambdas for `run_core_async`/`run_core_streaming_async` | `AnonymousDelegatingAIAgent` | 0.25d |
| 7.5 | **`AIHostAgent`** | `DelegatingAIAgent` for hosting scenarios (routes requests to named agents) | `AIHostAgent` | 0.5d |
| 7.6 | **Builder API enhancements** | `AIAgentBuilder` methods for adding decorators: `.with_logging()`, `.with_telemetry()`, `.with_function_invocation_handler()` | `AIAgentBuilder` | 0.5d |
| 7.7 | **Unit tests** | Verify decorator chain ordering, telemetry span attributes, logging output | — | 1d |

### Gate Criteria

- [ ] `LoggingAgent` produces structured log output for invocations
- [ ] `OpenTelemetryAgent` creates spans with correct GenAI semantic convention attributes
- [ ] Decorator chain `Logging → OTel → FunctionInvocation → ChatClientAgent` works in correct order
- [ ] Builder API produces correctly layered agents
- [ ] OTel spans exportable via OTLP (verified with in-memory exporter in tests)

---

## Phase 8: Hosting Layer (Weeks 13–16)

### Objective

Build the HTTP server infrastructure and implement all three hosting protocols. This is the most complex phase due to protocol-specific SSE streaming, JSON-RPC, and the need for interop testing.

### 8A: HTTP Server Foundation (Week 13)

| # | Task | Details | Effort |
|---|---|---|---|
| 8A.1 | **`HttpRouter`** | Route registration with path parameters (`/{agentName}/v1/...`), HTTP verb matching, middleware pipeline. Built on Boost.Beast | 2d |
| 8A.2 | **`SseWriter`** | Chunked response streaming. `write_event(event, data)`, `close()`. Sets correct headers: `Content-Type: text/event-stream`, `Cache-Control: no-cache,no-store`, `Connection: keep-alive` | 1d |
| 8A.3 | **`AgentHost`** | Top-level server class: listens on port, accepts connections, dispatches to router. TLS support. Graceful shutdown via `std::stop_source` | 1.5d |
| 8A.4 | **`AgentHostBuilder`** | Fluent builder: `.with_port()`, `.with_tls()`, `.with_agent()`, `.with_middleware()`, `.map_openai()`, `.map_a2a()`, `.map_agui()`, `.build()` | 0.5d |
| 8A.5 | **Server middleware** | CORS, request logging, error handling, authentication middleware types | 1d |
| 8A.6 | **Tests** | Route matching, SSE write/read round-trip, concurrent connections | 1d |

### 8B: OpenAI-Compatible Hosting (Weeks 13–14)

| # | Task | Details | Effort |
|---|---|---|---|
| 8B.1 | **Chat Completions endpoint** | `POST /{agentName}/v1/chat/completions/`. Request → `ChatMessage` conversion → agent `run_async` / `run_streaming_async`. Non-streaming: JSON response. Streaming: SSE `ChatCompletionChunk` events | 2d |
| 8B.2 | **Request/response models** | ~12 model structs for Chat Completions format with glaze serialization | 1d |
| 8B.3 | **Responses API endpoints** | 5 endpoints: CreateResponse, GetResponse, CancelResponse, DeleteResponse, ListInputItems. 19 SSE event types for streaming | 3d |
| 8B.4 | **Responses streaming event generators** | Map internal `AgentResponseUpdate` to the 19 OpenAI Responses API event types. 11 distinct generator strategies (matching .NET `StreamingEventGenerator` classes) | 2d |
| 8B.5 | **Conversations API endpoints** | 8 endpoints for multi-turn conversation management with pagination | 2d |
| 8B.6 | **Interop tests** | C++ OpenAI-compatible server ↔ .NET/Python OpenAI client library | 1d |

### 8C: A2A Protocol Hosting (Week 15)

| # | Task | Details | Effort |
|---|---|---|---|
| 8C.1 | **Agent Card discovery** | `GET /.well-known/agent.json` returns agent capabilities | 0.5d |
| 8C.2 | **JSON-RPC 2.0 dispatcher** | `POST /` receives JSON-RPC 2.0 requests, routes to method handlers: `tasks/send`, `tasks/get`, `tasks/cancel`, `tasks/sendSubscribe`, `tasks/resubscribe` | 2d |
| 8C.3 | **Task lifecycle** | TaskManager: create, update, cancel tasks. Map to agent invocations | 1d |
| 8C.4 | **SSE streaming** | `tasks/sendSubscribe` → SSE stream of `AgentMessage`, `AgentTask`, `TaskUpdateEvent` | 1d |
| 8C.5 | **A2A client agent** | `class A2AAgent : public AIAgent` — remote proxy that calls another agent via A2A protocol. Continuation token support | 1d |
| 8C.6 | **Protocol types** | A2A protocol message types (~15 structs) with JSON serialization | 0.5d |
| 8C.7 | **Interop tests** | C++ A2A server ↔ .NET A2A client, and vice versa | 1d |

### 8D: AG-UI Protocol Hosting (Week 15–16)

| # | Task | Details | Effort |
|---|---|---|---|
| 8D.1 | **AG-UI SSE endpoint** | `POST /` → SSE stream with 12 event types: `RUN_STARTED`, `RUN_FINISHED`, `RUN_ERROR`, `TEXT_MESSAGE_START`, `TEXT_MESSAGE_CONTENT`, `TEXT_MESSAGE_END`, `TOOL_CALL_START`, `TOOL_CALL_ARGS`, `TOOL_CALL_END`, `STATE_SNAPSHOT`, `STATE_DELTA`, `CUSTOM` | 2d |
| 8D.2 | **AG-UI client** | `class AGUIChatClient : public DelegatingChatClient`. Wraps tool execution with AG-UI events. Maps `AgentResponseUpdate` to AG-UI SSE events | 1.5d |
| 8D.3 | **Protocol types** | 12 event structs, 5 message types, tool/context types (~20 structs total) | 0.5d |
| 8D.4 | **Interop tests** | C++ AG-UI server ↔ CopilotKit client | 1d |

### Gate Criteria

- [ ] `HttpRouter` correctly matches paths with parameters, returns 404 for unmatched
- [ ] SSE streaming: client receives all events in order, proper formatting
- [ ] OpenAI Chat Completions: request accepted by OpenAI Python client → C++ server
- [ ] OpenAI Responses: all 19 event types stream correctly
- [ ] A2A: agent card discovery + task send + task subscribe work
- [ ] AG-UI: all 12 event types stream with correct names and payloads
- [ ] Cross-language interop: .NET client ↔ C++ server passes for each protocol
- [ ] Concurrent request handling (multiple simultaneous SSE streams)
- [ ] Graceful shutdown: in-flight SSE streams complete or terminate cleanly

---

## Phase 9: Workflows (Weeks 16–19)

### Objective

Port the workflow engine (12,915 LOC) and declarative workflow layer (17,417 LOC) — the two largest components. This phase uses Lua/sol2 as the expression engine with a PowerFx compatibility shim.

### 9A: Graph Executor (Weeks 16–17)

| # | Task | Details | Effort |
|---|---|---|---|
| 9A.1 | **`WorkflowNode` base** | Abstract node with `execute_async(context) -> Task<void>`. Holds name, edges | 1d |
| 9A.2 | **Edge types** | `DirectEdge` (unconditional), `ConditionalEdge` (expression-based), `FanOutEdge` (parallel), `FanInEdge` (waits for all/any) | 2d |
| 9A.3 | **`WorkflowExecutor`** | Graph traversal engine. Executes nodes in topological order respecting edges. Fan-out uses `when_all`. Supports streaming | 3d |
| 9A.4 | **Builder APIs** | `build_sequential()`, `build_concurrent()`, `build_handoff()`, `build_group_chat()` — convenience builders | 2d |
| 9A.5 | **Checkpoint interface** | `class ICheckpointStore { virtual Task<void> save(...); virtual Task<json_t> load(...); }`. In-memory default | 0.5d |
| 9A.6 | **OpenTelemetry integration** | Spans per node execution, workflow-level parent span | 0.5d |
| 9A.7 | **Unit tests** | Linear flow, branching, fan-out/in, cycles (error), checkpointing | 2d |

### 9B: Declarative Workflows + Lua Engine (Weeks 17–19)

| # | Task | Details | Effort |
|---|---|---|---|
| 9B.1 | **YAML parsing** | yaml-cpp parsing of agent definitions, workflow definitions, steps, conditions, variable declarations | 2d |
| 9B.2 | **Lua expression engine** | `class ExpressionEngine` wrapping sol2. `evaluate(expr) -> json_t`, `evaluate_condition(expr) -> bool`, `evaluate_string(expr) -> string`. Variable set/get | 2d |
| 9B.3 | **PowerFx compatibility shim** | Translate PowerFx patterns to Lua: `Set(var, val)` → `var = val`, `If(cond, then, else)` → Lua if/then/else, `Concatenate(a, b)` → `a .. b`, `Text(n)` → `tostring(n)`, `Topic.var` → Lua table access. Regex-based translator | 2d |
| 9B.4 | **Variable scoping** | 5 scope levels matching .NET: Local, Global, System, Environment, Topic. Lua environment tables per scope | 1d |
| 9B.5 | **`WorkflowAgentProvider`** | Abstraction for resolving agent references in YAML to actual `AIAgent` instances | 0.5d |
| 9B.6 | **Declarative workflow builder** | YAML → `WorkflowExecutor`: parse steps, create nodes, wire edges based on conditions and flow control | 2d |
| 9B.7 | **Build-time code generation** | CMake custom command to replace Roslyn source generators. Scans for `[MessageHandler]`-equivalent attributes and generates routing code. Python script or dedicated tool | 2d |
| 9B.8 | **Unit tests** | YAML parsing, expression evaluation (both Lua native and PowerFx-translated), variable scoping, full declarative workflow execution | 2d |

### Gate Criteria

- [ ] Linear workflow: A → B → C executes in order
- [ ] Fan-out: A → {B, C} → D executes B and C concurrently, D waits for both
- [ ] Conditional edge: expression evaluated, correct branch taken
- [ ] Checkpoint save/load round-trips
- [ ] YAML-defined workflow matches manually-built equivalent
- [ ] PowerFx expressions (`Set`, `If`, `Concatenate`, `Topic.var`) evaluate correctly
- [ ] Lua native expressions evaluate correctly
- [ ] Variable scoping: Local isolates, Global shared, Environment reads env vars
- [ ] Build-time code generation produces correct routing code

---

## Phase 10: Storage & Integrations (Weeks 19–20)

### Objective

Implement external storage backends and integration libraries.

### Deliverables

| # | Task | Details | Effort |
|---|---|---|---|
| 10.1 | **In-memory stores** | `InMemoryChatHistoryStore`, `InMemoryCheckpointStore`, `InMemoryAgentSessionStore` — `std::unordered_map` with `std::mutex` | 0.5d |
| 10.2 | **Cosmos DB storage** | `class CosmosDbChatHistoryProvider : public ChatHistoryProvider`. REST API calls via `HttpClient` (no C++ Cosmos SDK). Container management, query by session ID, upsert messages | 2d |
| 10.3 | **Redis storage** | `class RedisChatHistoryProvider`, `class RedisCheckpointStore`. Use `hiredis` or `redis-plus-plus`. Async operations | 1.5d |
| 10.4 | **Mem0 integration** | `class Mem0ContextProvider : public AIContextProvider`. REST calls to Mem0 API for semantic memory search. Inject relevant memories as context | 1d |
| 10.5 | **Purview DLP middleware** | Agent middleware that calls Microsoft Graph API to check prompts/responses against Purview DLP policies. Block or allow based on response | 2d |
| 10.6 | **MCP tool serving** | Integration with `modelcontextprotocol/cpp-sdk` for serving agent tools via MCP protocol. `MCPToolServer` class | 1d |
| 10.7 | **Tests** | Storage round-trip tests (in-memory always, Cosmos/Redis gated by env vars), MCP tool discovery/invocation | 2d |

### Gate Criteria

- [ ] In-memory stores: concurrent read/write safety
- [ ] Cosmos DB: create container, store messages, query by session, delete
- [ ] Redis: store/retrieve chat history, checkpoint save/load
- [ ] Mem0: memory search returns relevant context, injected into agent
- [ ] Purview: DLP policy blocks prohibited content, allows safe content
- [ ] MCP: tool discovery and invocation via MCP protocol

---

## Phase 11: Durable Tasks (Weeks 20–21)

### Objective

Implement durable entity-backed agents and the standalone daemon that replaces Azure Functions hosting.

### Deliverables

| # | Task | Details | Effort |
|---|---|---|---|
| 11.1 | **gRPC Durable Task client** | Generate C++ stubs from `microsoft/durabletask-protobuf` proto files. `class DurableTaskClient` with entity operations: `signal_entity`, `get_entity_state`, `schedule_new_orchestration` | 2d |
| 11.2 | **`DurableAIAgent`** | `class DurableAIAgent : public AIAgent`. Entity-backed state. Entity ID = composite `AgentSessionId`. Operations: `run_async` signals entity, entity state machine processes | 2d |
| 11.3 | **`DurableAIAgentProxy`** | For calling durable agents from outside an orchestration context | 0.5d |
| 11.4 | **Entity state schema** | Match `schemas/durable-agent-entity-state.json`. JSON serialization of entity state | 0.5d |
| 11.5 | **Standalone daemon** | `class AgentDaemon` — HTTP server that mirrors Azure Functions trigger patterns. Receives HTTP requests, dispatches to durable task sidecar via gRPC. Replaces Azure Functions hosting for C++ | 2d |
| 11.6 | **Tests** | Unit tests with mock gRPC server. Integration tests with Durable Task sidecar | 2d |

### Gate Criteria

- [ ] gRPC client connects to Durable Task sidecar
- [ ] Entity signal/query round-trips correctly
- [ ] `DurableAIAgent` persists state across invocations
- [ ] Standalone daemon accepts HTTP requests and dispatches to sidecar
- [ ] Entity state matches JSON schema

---

## Phase 12: Developer UI & Samples (Weeks 21–22)

### Objective

Embed the developer UI SPA and create all sample applications that demonstrate the framework.

### Deliverables

| # | Task | Details | Effort |
|---|---|---|---|
| 12.1 | **Developer UI embedding** | Serve the existing SPA static files from the C++ HTTP server. REST API for entity discovery, agent listing, conversation management | 2d |
| 12.2 | **Getting Started samples** | 4 progressive samples matching .NET: Step1_Chat, Step2_Chat_Agent, Step3_Chat_Agent_Tools, Step4_Chat_Agent_OpenAI_Hosting. Each as standalone CMake executable with README | 2d |
| 12.3 | **Hosted Agent samples** | Multi-agent hosting: A2A, AG-UI, OpenAI-compatible. Agent-to-agent communication | 1d |
| 12.4 | **Durable Agent samples** | Long-running agent with checkpoint/resume | 1d |
| 12.5 | **Declarative workflow samples** | YAML-defined workflows with Lua expressions | 0.5d |
| 12.6 | **Sample README template** | Consistent format matching .NET sample READMEs: what it does, prerequisites, how to run, expected output | 0.5d |
| 12.7 | **Top-level README** | `cpp/samples/README.md` with index of all samples | 0.25d |
| 12.8 | **End-to-end validation** | Run all samples, verify output matches expectations | 1d |

### Gate Criteria

- [ ] DevUI serves and displays agent information
- [ ] All Getting Started samples compile and run
- [ ] Hosted Agent sample demonstrates multi-protocol hosting
- [ ] Durable Agent sample demonstrates persistence across restarts
- [ ] Each sample has a README with build/run instructions
- [ ] Samples work on all 3 platforms

---

## Cross-Cutting Concerns (All Phases)

### Testing Strategy

| Level | Framework | Coverage Target | Details |
|---|---|---|---|
| Unit tests | Google Test + gmock | Every public API method | Mock dependencies, test edge cases, error paths |
| Integration tests | Google Test | Per provider, per protocol | Gated by env vars (`OPENAI_API_KEY`, `AZURE_AI_ENDPOINT`, etc.) |
| Interop tests | Google Test + external runners | Per hosting protocol | C++ server ↔ .NET/Python client, C++ client ↔ .NET/Python server |
| Sanitizer tests | ASAN, TSAN, UBSAN | All unit tests | Run in CI as separate matrix entries |
| Benchmark tests | Google Benchmark | SSE parsing, JSON serialization, middleware pipeline | Track regressions |

### Documentation Strategy

| Artifact | Location | Format |
|---|---|---|
| API reference | Generated from Doxygen comments | HTML via Doxygen |
| Architecture guide | `cpp/docs/architecture.md` | Markdown |
| Migration guide (.NET → C++) | `cpp/docs/migration-from-dotnet.md` | Markdown with side-by-side examples |
| Migration guide (Python → C++) | `cpp/docs/migration-from-python.md` | Markdown with side-by-side examples |
| Per-sample README | `cpp/samples/*/README.md` | Markdown |

### Naming Conventions

| Concept | Convention | Example |
|---|---|---|
| Namespace | `agents::`, `agents::openai::`, `agents::hosting::` | `agents::AIAgent` |
| Header files | `snake_case.h` | `chat_client_agent.h` |
| Source files | `snake_case.cpp` | `chat_client_agent.cpp` |
| Class names | `PascalCase` | `ChatClientAgent` |
| Method names | `snake_case` | `run_async()`, `get_response()` |
| Member variables | `snake_case_` (trailing underscore) | `inner_agent_` |
| Constants | `kPascalCase` | `kMaxRetries` |
| Enum values | `PascalCase` | `ChatRole::Assistant` |
| CMake targets | `agents-<component>` | `agents-abstractions`, `agents-ai` |
| vcpkg port | `agent-framework-cpp` | — |

### Copyright Header

All `.h` and `.cpp` files:

```cpp
// Copyright (c) Microsoft. All rights reserved.
```

### Error Handling Strategy

| Approach | When |
|---|---|
| Exceptions (`std::runtime_error` subclasses) | Unrecoverable errors: network failures, invalid config, protocol violations |
| `std::expected<T, Error>` (C++23) or `Result<T>` | Recoverable errors where caller should handle: JSON parse failures, API 4xx responses |
| Error codes in response | Expected failure modes: `FinishReason::Error`, `FailureReason::AgentError` |
| `CancellationToken` / `std::stop_token` | Cooperative cancellation of async operations |

---

## Library Dependency Graph

```text
agents-async (Boost.Asio, Boost.Beast, OpenSSL)
    │
    ▼
agents-abstractions (glaze, nlohmann-json)
    │
    ▼
agents-ai (spdlog, fmt, opentelemetry-cpp)
    │
    ├──► agents-openai
    ├──► agents-azure-ai ──► agents-azure-ai-persistent
    ├──► agents-anthropic
    ├──► agents-copilotstudio
    ├──► agents-github-copilot
    │
    ├──► agents-workflows ──► agents-workflows-declarative (yaml-cpp, sol2, lua)
    │                         └──► agents-workflows-declarative-azure-ai
    │
    ├──► agents-hosting ──► agents-hosting-openai
    │                   ├──► agents-hosting-a2a
    │                   ├──► agents-hosting-agui
    │                   └──► agents-hosting-daemon
    │
    ├──► agents-a2a (A2A client)
    ├──► agents-agui (AG-UI client)
    │
    ├──► agents-durabletask (grpc, protobuf)
    │
    ├──► agents-storage-cosmos
    ├──► agents-storage-redis (hiredis)
    ├──► agents-mem0
    ├──► agents-purview
    ├──► agents-mcp (modelcontextprotocol-cpp-sdk)
    │
    └──► agents-devui
```

---

## Estimated LOC by Phase

| Phase | Est. C++ Header LOC | Est. C++ Source LOC | Est. Test LOC | Total |
|---|---|---|---|---|
| 0 Build infra | 0 | 0 | 0 | ~500 (CMake/config) |
| 1 Async | 800 | 1,200 | 1,000 | 3,000 |
| 2 Content model | 600 | 400 | 800 | 1,800 |
| 3 Tools | 400 | 600 | 600 | 1,600 |
| 4 Core agents | 1,500 | 2,500 | 2,000 | 6,000 |
| 5 OpenAI provider | 600 | 1,500 | 800 | 2,900 |
| 6 Other providers | 800 | 2,000 | 1,200 | 4,000 |
| 7 Cross-cutting | 300 | 500 | 400 | 1,200 |
| 8 Hosting | 1,200 | 4,000 | 2,000 | 7,200 |
| 9 Workflows | 1,500 | 5,000 | 2,000 | 8,500 |
| 10 Storage | 400 | 1,500 | 800 | 2,700 |
| 11 Durable | 400 | 1,200 | 600 | 2,200 |
| 12 DevUI & samples | 200 | 2,000 | 500 | 2,700 |
| **Total** | **~8,700** | **~22,400** | **~12,700** | **~44,300** |

*Note: C++ typically requires ~25-40% more code than C# for equivalent functionality due to header/source split, explicit templates, and less reflection.*

---

## Risk Mitigation Schedule

| Risk | Phase Affected | Mitigation Action | When |
|---|---|---|---|
| `AsyncGenerator<T>` coroutine edge cases | Phase 1 | Extensive fuzzing, sanitizer runs, compare with cppcoro reference | Week 2 |
| glaze API change | Phase 2 | Pin to specific version, write abstraction layer over serialization | Week 3 |
| No OpenAI C++ SDK | Phase 5 | Build from REST spec, validate against live API early | Week 7 |
| PowerFx ↔ Lua translation gaps | Phase 9 | Enumerate all PowerFx expressions used in .NET/Python YAML samples, verify each translates | Week 17 |
| A2A/AG-UI protocol compliance | Phase 8 | Use .NET/Python servers as reference implementations for interop testing | Week 14 |
| Cross-platform Beast server issues | Phase 8 | Test TLS on all 3 platforms in week 13, have libcurl fallback plan | Week 13 |
| Build time growth | All | Precompiled headers from Phase 0, monitor build times in CI, split large headers | Ongoing |

---

## Next Steps

1. **Clear context** by typing `/clear`.
2. Attach or open [2026-02-20-cpp-port-implementation-plan.md](.copilot-tracking/research/2026-02-20-cpp-port-implementation-plan.md).
3. Start implementation by typing `/task-plan` and selecting Phase 0 tasks.
