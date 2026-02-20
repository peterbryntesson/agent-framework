<!-- markdownlint-disable-file -->
# Implementation Details: C++ Port of Agent Framework

## Context Reference

Sources:
* .copilot-tracking/research/2026-02-20-cpp-port-implementation-plan.md — Full research document with code snippets and gate criteria
* .copilot-tracking/research/2026-02-20-cpp-port-research.md — Technology research and decisions
* .copilot-tracking/subagent/2026-02-20/dotnet-structure-analysis.md — .NET project inventory (27 projects, ~71K LOC)
* .copilot-tracking/subagent/2026-02-20/python-structure-analysis.md — Python package inventory (20 packages, ~79K LOC)
* .copilot-tracking/subagent/2026-02-20/hosting-protocols-analysis.md — Protocol specifications

## Implementation Phase 0: Build Infrastructure

<!-- parallelizable: false -->

### Step 0.1: Create top-level CMake structure with presets

Create the root CMake configuration for the C++ port.

Files:
* cpp/CMakeLists.txt — Root CMake file. Set `cmake_minimum_required(VERSION 3.25)`, `project(agent-framework-cpp)`, C++20 standard, add subdirectories
* cpp/CMakePresets.json — Preset configurations: Debug, Release, RelWithDebInfo, ASAN, TSAN, UBSAN. Each preset specifies generator, build directory, toolchain file for vcpkg, and sanitizer flags

Success criteria:
* `cmake --preset release` configures without error on MSVC, GCC 11, Clang 14
* Presets for all build types resolve correctly

Context references:
* .copilot-tracking/research/2026-02-20-cpp-port-implementation-plan.md (Lines 62-90) — Phase 0 deliverables and file tree

Dependencies:
* None — first step

### Step 0.2: Create vcpkg manifest with all dependencies and feature flags

Define all third-party dependencies in a vcpkg manifest with optional feature flags for components not needed in all builds.

Files:
* cpp/vcpkg.json — Core dependencies: `boost-asio`, `boost-beast`, `glaze`, `nlohmann-json`, `spdlog`, `fmt`, `opentelemetry-cpp`, `openssl`, `gtest`. Feature flags: `[yaml]` for yaml-cpp, `[lua]` for sol2, `[grpc]` for gRPC+protobuf, `[redis]` for hiredis

Success criteria:
* `vcpkg install` resolves all core dependencies on all platforms
* Feature flags enable/disable optional dependencies correctly

Context references:
* .copilot-tracking/research/2026-02-20-cpp-port-implementation-plan.md (Lines 10-18) — Decisions log with library choices

Dependencies:
* Step 0.1 — CMake must reference vcpkg toolchain

### Step 0.3: Scaffold all 27 library directories with CMakeLists.txt stubs

Create the directory structure for all C++ libraries, mapping 1:1 from .NET projects.

Files:
* cpp/src/CMakeLists.txt — Parent that adds all library subdirectories
* cpp/src/Agents.Async/CMakeLists.txt — `agents-async` target (Boost.Asio, Boost.Beast, OpenSSL)
* cpp/src/Agents.Abstractions/CMakeLists.txt — `agents-abstractions` target (glaze, nlohmann-json)
* cpp/src/Agents.AI/CMakeLists.txt — `agents-ai` target (spdlog, fmt, opentelemetry-cpp)
* cpp/src/Agents.OpenAI/CMakeLists.txt — `agents-openai` target
* cpp/src/Agents.AzureAI/CMakeLists.txt — `agents-azure-ai` target
* cpp/src/Agents.AzureAI.Persistent/CMakeLists.txt — `agents-azure-ai-persistent` target
* cpp/src/Agents.Anthropic/CMakeLists.txt — `agents-anthropic` target
* cpp/src/Agents.CopilotStudio/CMakeLists.txt — `agents-copilotstudio` target
* cpp/src/Agents.GitHub.Copilot/CMakeLists.txt — `agents-github-copilot` target
* cpp/src/Agents.A2A/CMakeLists.txt — `agents-a2a` target
* cpp/src/Agents.AGUI/CMakeLists.txt — `agents-agui` target
* cpp/src/Agents.Workflows/CMakeLists.txt — `agents-workflows` target
* cpp/src/Agents.Workflows.Declarative/CMakeLists.txt — `agents-workflows-declarative` target (yaml-cpp, sol2)
* cpp/src/Agents.Workflows.Declarative.AzureAI/CMakeLists.txt — `agents-workflows-declarative-azure-ai` target
* cpp/src/Agents.Hosting/CMakeLists.txt — `agents-hosting` target
* cpp/src/Agents.Hosting.OpenAI/CMakeLists.txt — `agents-hosting-openai` target
* cpp/src/Agents.Hosting.A2A/CMakeLists.txt — `agents-hosting-a2a` target
* cpp/src/Agents.Hosting.AGUI/CMakeLists.txt — `agents-hosting-agui` target
* cpp/src/Agents.Hosting.Daemon/CMakeLists.txt — `agents-hosting-daemon` target
* cpp/src/Agents.DurableTask/CMakeLists.txt — `agents-durabletask` target (gRPC, protobuf)
* cpp/src/Agents.Storage.Cosmos/CMakeLists.txt — `agents-storage-cosmos` target
* cpp/src/Agents.Storage.Redis/CMakeLists.txt — `agents-storage-redis` target (hiredis)
* cpp/src/Agents.Mem0/CMakeLists.txt — `agents-mem0` target
* cpp/src/Agents.Purview/CMakeLists.txt — `agents-purview` target
* cpp/src/Agents.MCP/CMakeLists.txt — `agents-mcp` target
* cpp/src/Agents.DevUI/CMakeLists.txt — `agents-devui` target

Each library stub directory contains:
* `CMakeLists.txt` — Defines the library target with `add_library()`, `target_sources(FILE_SET)`, `target_link_libraries()`
* `include/agents/<component>/` — Public headers directory (empty placeholder `.h`)
* `src/` — Private sources directory (empty placeholder `.cpp`)

Success criteria:
* All 27 targets are defined and can be configured (even if empty)
* Dependency graph between targets matches the diagram in the research document
* `cmake --build --preset release` succeeds with all stub libraries

Context references:
* .copilot-tracking/subagent/2026-02-20/dotnet-structure-analysis.md (Lines 28-85) — Full .NET project inventory to map from
* .copilot-tracking/research/2026-02-20-cpp-port-implementation-plan.md (Lines 765-795) — Library dependency graph

Dependencies:
* Step 0.1 — Root CMake must exist
* Step 0.2 — vcpkg manifest must declare dependencies

### Step 0.4: Create shared CMake modules

Create reusable CMake modules for compiler warnings, dependency management, and install/find_package support.

Files:
* cpp/cmake/CompilerWarnings.cmake — Function `set_project_warnings(target)`: enables `-Wall -Wextra -Wpedantic -Werror` (GCC/Clang), `/W4 /WX` (MSVC), with selective suppressions
* cpp/cmake/Dependencies.cmake — `find_package` calls for all vcpkg dependencies, wraps optional deps behind feature flags
* cpp/cmake/AgentFrameworkConfig.cmake.in — Template for `find_package(AgentFramework)` support in downstream projects
* cpp/Directory.Build.cmake — Shared properties: C++20 standard, output directories, install rules, copyright header check

Success criteria:
* Compiler warnings function applies correct flags per compiler
* `find_package(AgentFramework)` works after install

Context references:
* .copilot-tracking/research/2026-02-20-cpp-port-implementation-plan.md (Lines 68-72) — Shared CMake modules spec

Dependencies:
* Step 0.1

### Step 0.5: Create GitHub Actions CI workflow with compiler/OS matrix

Set up the CI pipeline for cross-platform, cross-compiler testing.

Files:
* .github/workflows/cpp-build.yml — Matrix strategy: `{msvc-latest, gcc-11, clang-14}` × `{windows-latest, ubuntu-latest, macos-latest}`. Skip invalid combos (msvc on Linux/macOS). Steps: checkout → vcpkg bootstrap → cmake configure → build → test → upload artifacts. Separate sanitizer jobs (ASAN, TSAN, UBSAN on GCC/Clang only)

Success criteria:
* CI pipeline triggers on push and PR to `main`
* All valid matrix entries complete successfully
* Sanitizer jobs run without failures on scaffold code
* Build artifacts uploaded for each platform

Context references:
* .copilot-tracking/research/2026-02-20-cpp-port-implementation-plan.md (Lines 73-76) — CI specification

Dependencies:
* Steps 0.1-0.4 — Complete CMake configuration

### Step 0.6: Add code quality tooling

Configure static analysis and formatting tools.

Files:
* cpp/.clang-tidy — Enable checks: `modernize-*`, `readability-*`, `bugprone-*`, `performance-*`, `cppcoreguidelines-*`. Disable: `modernize-use-trailing-return-type`, `readability-magic-numbers`
* cpp/.clang-format — Style: `BasedOnStyle: Google`, `IndentWidth: 4`, `ColumnLimit: 120`, `BreakBeforeBraces: Attach`, `AllowShortFunctionsOnASingleLine: Inline`
* cpp/.pre-commit-config.yaml — Pre-commit hooks for clang-format and clang-tidy

Success criteria:
* `clang-format --dry-run --Werror` passes on all scaffold files
* `clang-tidy` runs without errors on all scaffold files
* Pre-commit hooks integrate with Git workflow

Context references:
* .copilot-tracking/research/2026-02-20-cpp-port-implementation-plan.md (Lines 77-79) — Code quality tooling spec

Dependencies:
* Step 0.3 — Source files must exist to lint

### Step 0.7: Add copyright header template and check script

Enforce the Microsoft copyright header on all source files.

Files:
* cpp/cmake/CheckCopyright.cmake — CMake script or Python script that scans all `.h`/`.cpp` files for `// Copyright (c) Microsoft. All rights reserved.` header. Returns non-zero exit code if missing
* Integration into CI workflow (additional step)

Success criteria:
* Script detects missing copyright headers
* All scaffold files pass the check

Dependencies:
* Step 0.3

### Step 0.8: Write cpp/README.md

Create the top-level documentation for the C++ port.

Files:
* cpp/README.md — Sections: Overview, Prerequisites, Build Instructions (CMake presets), Running Tests, Project Structure (library list with descriptions), Architecture Overview (dependency graph), Contributing Guidelines, Naming Conventions table

Success criteria:
* README renders correctly in GitHub
* Build instructions reproduce a working build

Dependencies:
* Steps 0.1-0.6

## Implementation Phase 1: Async Primitives

<!-- parallelizable: false -->

### Step 1.1: Implement Task<T> alias, when_all(), CancellationToken wrappers

Create the foundational async type aliases that unify Boost.Asio coroutines with the framework's API.

Files:
* cpp/src/Agents.Async/include/agents/async/task.h — `template<typename T> using Task = asio::awaitable<T>`, `using CancellationToken = std::stop_token`, `using CancellationSource = std::stop_source`
* cpp/src/Agents.Async/include/agents/async/when_all.h — `when_all()` wrapping `asio::experimental::make_parallel_group()`. Returns `std::tuple<T...>` of results. Handles partial failures: if any throws, propagates first exception after all complete
* cpp/src/Agents.Async/src/when_all.cpp — Implementation if non-trivial template specializations needed

Success criteria:
* `Task<int>` compiles and `co_return` works
* `when_all` handles all-success and partial-failure cases
* `CancellationToken` cooperatively stops async operations

Context references:
* .copilot-tracking/research/2026-02-20-cpp-port-implementation-plan.md (Lines 114-140) — Phase 1 deliverables and code snippets
* dotnet/src/Microsoft.Agents.AI.Abstractions/ — .NET Task/CancellationToken patterns to match

Dependencies:
* Phase 0 complete

### Step 1.2: Implement AsyncGenerator<T> C++20 coroutine type

Build the most critical async primitive: a pull-based async generator supporting `co_yield`, cancellation, and RAII cleanup.

Files:
* cpp/src/Agents.Async/include/agents/async/async_generator.h — `class AsyncGenerator<T>` with:
  * `struct promise_type` — C++20 coroutine promise with `yield_value(T)`, `return_void()`, `unhandled_exception()`
  * `Task<std::optional<T>> next()` — Pull-based iteration, returns `std::nullopt` when exhausted
  * `struct iterator` — For range-for support in sync contexts
  * RAII cleanup of coroutine handle in destructor
  * `std::stop_token` cancellation support — checks before each yield point
  * Exception propagation — re-throws in `next()` caller
* cpp/src/Agents.Async/src/async_generator.cpp — Non-template implementation details if needed

Success criteria:
* Empty generator: `next()` returns `std::nullopt` immediately
* Single yield: produces one value then exhausts
* Multi-yield: produces correct sequence
* Exception propagation: exception in generator body is re-thrown to caller of `next()`
* Early cancellation: generator stops producing when `stop_token` is triggered
* Coroutine cleanup: no leaks when generator is destroyed before exhaustion

Context references:
* .copilot-tracking/research/2026-02-20-cpp-port-implementation-plan.md (Lines 141-162) — AsyncGenerator design
* .copilot-tracking/research/2026-02-20-cpp-port-research.md — Research on C++20 coroutine limitations

Dependencies:
* Step 1.1 — Task<T> alias

### Step 1.3: Implement AsyncChannel<T> and AsyncSemaphore

Build concurrency coordination primitives.

Files:
* cpp/src/Agents.Async/include/agents/async/async_channel.h — `class AsyncChannel<T>`: wrapper over `asio::experimental::channel<void(error_code, T)>` with `send(T)`, `Task<T> receive()`, `close()`. Bounded capacity
* cpp/src/Agents.Async/include/agents/async/async_semaphore.h — `class AsyncSemaphore`: async-aware semaphore for concurrency limiting. `Task<void> acquire()`, `void release()`, constructor takes `size_t initial_count`

Success criteria:
* Channel: send/receive works across coroutines, close unblocks waiting receivers
* Semaphore: limits concurrency to configured count

Context references:
* .copilot-tracking/research/2026-02-20-cpp-port-implementation-plan.md (Lines 126-131) — Channel and semaphore specs

Dependencies:
* Step 1.1

### Step 1.4: Implement HTTP client wrapper over Boost.Beast

Build the HTTP client that all provider libraries will use.

Files:
* cpp/src/Agents.Async/include/agents/async/http_client.h — `class HttpClient` with:
  * `Task<HttpResponse> get(std::string_view url, Headers headers, CancellationToken)`
  * `Task<HttpResponse> post_json(std::string_view url, std::string_view body, Headers headers, CancellationToken)`
  * `AsyncGenerator<std::string_view> post_streaming(std::string_view url, std::string_view body, Headers headers, CancellationToken)` — For SSE streams
  * Connection pooling with configurable pool size
  * TLS via OpenSSL context
  * Timeout support (connect, read, total)
  * `struct HttpResponse { int status_code; Headers headers; std::string body; }`
* cpp/src/Agents.Async/src/http_client.cpp — Full implementation using `beast::http::async_read`, `beast::http::async_write`, `ssl::stream`

Success criteria:
* GET/POST to HTTPS endpoints works
* Streaming POST yields chunks as `string_view`
* Connection pool reuses connections
* Timeout triggers correctly
* Cancellation stops in-flight requests

Context references:
* .copilot-tracking/research/2026-02-20-cpp-port-implementation-plan.md (Lines 132-137) — HTTP client spec

Dependencies:
* Steps 1.1, 1.2 — Task<T> and AsyncGenerator<T>

### Step 1.5: Implement SSE stream parser

Parse Server-Sent Events from raw HTTP streaming chunks.

Files:
* cpp/src/Agents.Async/include/agents/async/sse_parser.h — `AsyncGenerator<SseEvent> parse_sse_stream(AsyncGenerator<std::string_view> raw_chunks)`. `struct SseEvent { std::string event; std::string data; std::optional<std::string> id; }`
* cpp/src/Agents.Async/src/sse_parser.cpp — Implementation following SSE spec: multi-line `data:` fields (concatenate with newline), `event:` prefix, blank-line delimiters, ignore lines starting with `:` (keepalive comments), UTF-8 support

Success criteria:
* Standard events parse correctly
* Multi-line data fields concatenate with newlines
* Missing event field defaults to empty string
* Keepalive empty comments (`:` prefix) are ignored
* UTF-8 data preserved correctly

Context references:
* .copilot-tracking/research/2026-02-20-cpp-port-implementation-plan.md (Lines 138-142) — SSE parser spec
* .copilot-tracking/subagent/2026-02-20/hosting-protocols-analysis.md — SSE format details used in all protocols

Dependencies:
* Step 1.2 — AsyncGenerator<T>

### Step 1.6: Write unit tests for all async primitives

Comprehensive test suite for the async layer.

Files:
* cpp/tests/Agents.Async.Tests/async_generator_tests.cpp — Tests: empty, single yield, multi-yield, cancellation, exception propagation, early destruction, concurrent producers
* cpp/tests/Agents.Async.Tests/sse_parser_tests.cpp — Tests: standard events, multi-line data, no event field, keepalive comments, UTF-8 data, malformed input
* cpp/tests/Agents.Async.Tests/http_client_tests.cpp — Tests: GET, POST, streaming POST (mock server via Boost.Beast listener on localhost), TLS, timeout, cancellation
* cpp/tests/Agents.Async.Tests/when_all_tests.cpp — Tests: all success, partial failure, empty input
* cpp/tests/Agents.Async.Tests/channel_tests.cpp — Tests: send/receive, close, bounded capacity
* cpp/tests/Agents.Async.Tests/semaphore_tests.cpp — Tests: acquire/release, concurrency limit

Success criteria:
* All tests pass on MSVC, GCC, Clang
* No sanitizer warnings (ASAN, TSAN)
* Code coverage > 90% for async library

Dependencies:
* Steps 1.1-1.5

## Implementation Phase 2: Content and Message Model

<!-- parallelizable: false -->

### Step 2.1: Implement 9 content variant types with glaze discriminated union

Create all content types that form the foundation of the message model.

Files:
* cpp/src/Agents.Abstractions/include/agents/abstractions/content.h — 9 content structs:
  * `TextContent { std::string text; }`
  * `FunctionCallContent { std::string call_id; std::string name; std::string arguments; }`
  * `FunctionResultContent { std::string call_id; std::string name; std::string result; }`
  * `DataContent { std::vector<uint8_t> data; std::string media_type; }`
  * `ErrorContent { std::string message; std::string code; }`
  * `UsageContent { UsageDetails usage; }`
  * `AudioContent { std::vector<uint8_t> data; std::string format; }`
  * `ImageContent { std::vector<uint8_t> data; std::string format; }`
  * `UriContent { std::string uri; std::string media_type; }`
  * `using AIContent = std::variant<TextContent, FunctionCallContent, ..., UriContent>`
* cpp/src/Agents.Abstractions/include/agents/abstractions/content_serialization.h — `glz::meta<agents::AIContent>` specialization with `$type` discriminator and IDs: `"text"`, `"functionCall"`, `"functionResult"`, `"data"`, `"error"`, `"usage"`, `"audio"`, `"image"`, `"uri"`

Success criteria:
* All 9 types serialize/deserialize with `$type` discriminator
* Mixed `std::vector<AIContent>` round-trips correctly
* Polymorphic variant dispatch works

Context references:
* .copilot-tracking/research/2026-02-20-cpp-port-implementation-plan.md (Lines 168-230) — Content type definitions and glaze meta snippet
* dotnet/src/Microsoft.Agents.AI.Abstractions/ — .NET content type definitions to match

Dependencies:
* Phase 0, Phase 1

### Step 2.2: Implement ChatMessage, ChatRole, ChatResponse, ChatResponseUpdate

Create the core message and response types.

Files:
* cpp/src/Agents.Abstractions/include/agents/abstractions/chat_role.h — `enum class ChatRole { System, User, Assistant, Tool }` with `to_string()` / `from_string()`
* cpp/src/Agents.Abstractions/include/agents/abstractions/chat_message.h — `struct ChatMessage { ChatRole role; std::vector<AIContent> contents; std::string author_name; AdditionalProperties additional_properties; }`
* cpp/src/Agents.Abstractions/include/agents/abstractions/finish_reason.h — `enum class FinishReason { Stop, Length, ToolCalls, ContentFilter, Error }`
* cpp/src/Agents.Abstractions/include/agents/abstractions/chat_response.h — `struct ChatResponse { ChatMessage message; UsageDetails usage; std::optional<FinishReason> finish_reason; std::string model_id; std::string response_id; AdditionalProperties additional_properties; }`
* cpp/src/Agents.Abstractions/include/agents/abstractions/chat_response_update.h — `struct ChatResponseUpdate { std::optional<ChatRole> role; std::vector<AIContent> contents; std::optional<FinishReason> finish_reason; std::string model_id; ... }`

Success criteria:
* `ChatMessage` with different roles and mixed content serializes correctly
* `ChatResponse` and `ChatResponseUpdate` round-trip tests pass
* Enum string conversions match .NET values

Context references:
* .copilot-tracking/research/2026-02-20-cpp-port-implementation-plan.md (Lines 175-191) — ChatMessage and response specs

Dependencies:
* Step 2.1

### Step 2.3: Implement ChatOptions, UsageDetails, FinishReason, AdditionalProperties

Create the options and auxiliary types.

Files:
* cpp/src/Agents.Abstractions/include/agents/abstractions/additional_properties.h — `using AdditionalProperties = std::unordered_map<std::string, glz::json_t>`
* cpp/src/Agents.Abstractions/include/agents/abstractions/usage_details.h — `struct UsageDetails { std::optional<int> input_tokens; std::optional<int> output_tokens; std::optional<int> total_tokens; AdditionalProperties additional_properties; }`
* cpp/src/Agents.Abstractions/include/agents/abstractions/chat_options.h — `struct ChatOptions { std::optional<std::string> model_id; std::optional<float> temperature; std::optional<float> top_p; std::optional<int> max_output_tokens; std::vector<std::shared_ptr<AITool>> tools; ... }` (forward-declare `AITool`)

Success criteria:
* `ChatOptions` serializes with all optional fields correctly
* `AdditionalProperties` handles arbitrary JSON values

Dependencies:
* Step 2.1

### Step 2.4: Implement AgentResponse<T> and AgentResponseUpdate

Create the agent-level response wrapper types.

Files:
* cpp/src/Agents.Abstractions/include/agents/abstractions/agent_response.h — `template<typename T> struct AgentResponse { std::string agent_id; T response; FailureReason failure_reason; }`. `enum class FailureReason { None, AgentError, Cancelled, Timeout }`
* cpp/src/Agents.Abstractions/include/agents/abstractions/agent_response_update.h — Streaming variant for incremental updates
* cpp/src/Agents.Abstractions/include/agents/abstractions/response_continuation_token.h — `struct ResponseContinuationToken { std::string token; }`

Success criteria:
* `AgentResponse<ChatResponse>` wraps correctly
* `FailureReason` enum covers all .NET cases

Context references:
* .copilot-tracking/research/2026-02-20-cpp-port-implementation-plan.md (Lines 195-200) — AgentResponse spec

Dependencies:
* Steps 2.2, 2.3

### Step 2.5: Write JSON round-trip serialization tests for all types

Comprehensive serialization testing.

Files:
* cpp/tests/Agents.Abstractions.Tests/content_serialization_tests.cpp — Round-trip for each of 9 content types, mixed `vector<AIContent>`, polymorphic discriminator correctness
* cpp/tests/Agents.Abstractions.Tests/chat_message_tests.cpp — Round-trip for `ChatMessage`, `ChatResponse`, `ChatResponseUpdate`
* cpp/tests/Agents.Abstractions.Tests/cross_compatibility_tests.cpp — Deserialize JSON produced by .NET serializer (stored as test fixtures in `cpp/tests/fixtures/dotnet_json/`)

Success criteria:
* All round-trip tests pass
* .NET-produced JSON deserializes correctly
* No memory leaks under ASAN

Dependencies:
* Steps 2.1-2.4

## Implementation Phase 3: Tool System

<!-- parallelizable: false -->

### Step 3.1: Implement AITool base class and AIFunction with async invocation

Create the core tool abstraction.

Files:
* cpp/src/Agents.Abstractions/include/agents/abstractions/ai_tool.h — `class AITool { public: virtual ~AITool() = default; virtual std::string_view name() const = 0; virtual std::string_view description() const = 0; virtual glz::json_t json_schema() const = 0; }`
* cpp/src/Agents.Abstractions/include/agents/abstractions/ai_function.h — `class AIFunction : public AITool { public: virtual Task<glz::json_t> invoke_async(glz::json_t const& args, CancellationToken cancel) = 0; }`

Success criteria:
* Interface compiles on all platforms
* Virtual dispatch works correctly

Context references:
* .copilot-tracking/research/2026-02-20-cpp-port-implementation-plan.md (Lines 242-256) — Tool system deliverables

Dependencies:
* Phase 2

### Step 3.2: Implement make_function<>() template factory

Build the factory that creates `AIFunction` from C++ callables with automatic JSON Schema generation.

Files:
* cpp/src/Agents.Abstractions/include/agents/abstractions/make_function.h — Two overloads:
  * **Option A (lambda + params descriptor):** `make_function(name, description, callable, params(...))` where `params` is a variadic helper that registers parameter names and descriptions
  * **Option B (struct-based params):** `make_function<ParamStruct>(name, description, handler_fn)` where glaze reflection on `ParamStruct` generates both JSON Schema and deserialization
  * Internal: type-erased `AIFunction` implementation that deserializes JSON args → invokes callable → serializes result
  * Supported parameter types: `std::string`, `int`, `double`, `bool`, `glz::json_t`, `std::optional<T>`, custom structs with glaze meta

Success criteria:
* Creates working tools from lambdas, free functions, member functions
* Struct-based params auto-generate correct JSON Schema
* Lambda-based params with `params()` descriptor produce correct schema
* Invocation correctly deserializes JSON arguments and serializes results

Context references:
* .copilot-tracking/research/2026-02-20-cpp-port-implementation-plan.md (Lines 264-290) — make_function design with both options

Dependencies:
* Step 3.1

### Step 3.3: Implement JSON Schema generation

Template function for generating JSON Schema from C++ types.

Files:
* cpp/src/Agents.Abstractions/include/agents/abstractions/json_schema.h — `template<typename T> glz::json_t generate_json_schema()` using `glz::write_json_schema<T>()`. Handle nested objects, arrays, enums, optional fields (nullable), default values
* cpp/src/Agents.Abstractions/src/json_schema.cpp — Non-template utilities for schema manipulation (merging, adding descriptions)

Success criteria:
* Generated schema matches OpenAI function calling format (`type`, `properties`, `required`, `description`)
* Nested objects produce nested `$ref` or inline schemas
* `std::optional<T>` produces nullable fields
* Enum types produce `enum` arrays

Dependencies:
* Step 3.1

### Step 3.4: Implement DelegatingAIFunction and AGENT_TOOL macro

Create the decorator pattern for functions and a convenience macro.

Files:
* cpp/src/Agents.Abstractions/include/agents/abstractions/delegating_ai_function.h — `class DelegatingAIFunction : public AIFunction { protected: std::shared_ptr<AIFunction> inner_; ... }` — Forwards by default, subclasses override for pre/post processing
* cpp/src/Agents.Abstractions/include/agents/abstractions/agent_tool_macro.h — `#define AGENT_TOOL(name, desc, fn) agents::make_function(name, desc, fn)` — Convenience macro for tool registration

Success criteria:
* `DelegatingAIFunction` wraps and delegates correctly
* Pre/post processing hooks work
* Macro expands to valid `make_function` call

Dependencies:
* Steps 3.2, 3.3

### Step 3.5: Write unit tests for tool factory, schema, invocation

Files:
* cpp/tests/Agents.Abstractions.Tests/tool_factory_tests.cpp — Tests: lambda with struct params, lambda with params descriptor, free function, async callable, multiple parameter types
* cpp/tests/Agents.Abstractions.Tests/json_schema_tests.cpp — Tests: primitive types, nested structs, arrays, enums, optional fields, generated schema matches expected JSON
* cpp/tests/Agents.Abstractions.Tests/tool_invocation_tests.cpp — Tests: valid args, missing required params, wrong types, cancellation, `DelegatingAIFunction` chain

Success criteria:
* All factory patterns produce working tools
* Schema output matches OpenAI specification
* Error cases handled gracefully

Dependencies:
* Steps 3.1-3.4

## Implementation Phase 4: Core Agent Abstractions

<!-- parallelizable: false -->

### Step 4.1: Implement IChatClient interface and DelegatingChatClient

Create the chat client abstraction.

Files:
* cpp/src/Agents.AI/include/agents/ai/i_chat_client.h — `class IChatClient { public: virtual ~IChatClient() = default; virtual Task<ChatResponse> get_response(std::vector<ChatMessage> const& messages, ChatOptions const& options, CancellationToken cancel) = 0; virtual AsyncGenerator<ChatResponseUpdate> get_streaming_response(std::vector<ChatMessage> const& messages, ChatOptions const& options, CancellationToken cancel) = 0; }`
* cpp/src/Agents.AI/include/agents/ai/delegating_chat_client.h — `class DelegatingChatClient : public IChatClient { protected: std::shared_ptr<IChatClient> inner_client_; ... }` — Forwards by default

Success criteria:
* Interface defines both non-streaming and streaming methods
* `DelegatingChatClient` correctly delegates to inner client
* Override pattern works for intercepting calls

Context references:
* .copilot-tracking/research/2026-02-20-cpp-port-implementation-plan.md (Lines 308-318) — IChatClient and DelegatingChatClient

Dependencies:
* Phase 2 (ChatMessage, ChatResponse), Phase 1 (Task, AsyncGenerator)

### Step 4.2: Implement AIAgent abstract base

Create the core agent abstract class with 4 `run_async` overloads.

Files:
* cpp/src/Agents.AI/include/agents/ai/ai_agent.h — `class AIAgent` with:
  * Public: `Task<AgentResponse<ChatResponse>> run_async(messages, session, options, cancel)` (4 overloads with defaulted params)
  * Public: `AsyncGenerator<AgentResponseUpdate> run_streaming_async(messages, session, options, cancel)` (4 overloads)
  * Public: `template<typename T> std::shared_ptr<T> get_service() const` — Service locator
  * Protected virtual: `Task<ChatResponse> run_core_async(messages, options, cancel)` — Subclass implements
  * Protected virtual: `AsyncGenerator<ChatResponseUpdate> run_core_streaming_async(messages, options, cancel)` — Subclass implements
  * Session lifecycle orchestration in public methods (load history → merge context → call core → save history)
* cpp/src/Agents.AI/src/ai_agent.cpp — Implementation of public overloads, session lifecycle

Success criteria:
* 4 overloads resolve correctly at call sites
* Session lifecycle: load → merge → run → save
* Service locator resolves registered types
* Subclass pattern works (can implement just `run_core_async`)

Context references:
* .copilot-tracking/research/2026-02-20-cpp-port-implementation-plan.md (Lines 319-356) — AIAgent architecture snippet

Dependencies:
* Phase 2, Phase 1

### Step 4.3: Implement DelegatingAIAgent

Create the decorator base for agent chains.

Files:
* cpp/src/Agents.AI/include/agents/ai/delegating_ai_agent.h — `class DelegatingAIAgent : public AIAgent { protected: std::shared_ptr<AIAgent> inner_agent_; ... }` — All calls forward to inner by default. Subclasses override `run_core_async` / `run_core_streaming_async` to intercept

Success criteria:
* Chain `A → B → C` where A and B are delegating agents works correctly
* Override pattern allows interception at any level

Dependencies:
* Step 4.2

### Step 4.4: Implement AgentSession hierarchy

Create session management types.

Files:
* cpp/src/Agents.AI/include/agents/ai/agent_session.h — `class AgentSession { public: virtual ~AgentSession() = default; virtual std::string session_id() const = 0; virtual Task<std::vector<ChatMessage>> load_history(CancellationToken) = 0; virtual Task<void> save_history(std::vector<ChatMessage> const&, CancellationToken) = 0; }`
* cpp/src/Agents.AI/include/agents/ai/in_memory_agent_session.h — `class InMemoryAgentSession : public AgentSession` — Client-side chat history in `std::vector<ChatMessage>` with mutex
* cpp/src/Agents.AI/include/agents/ai/service_id_agent_session.h — `class ServiceIdAgentSession : public AgentSession` — Server-managed session with conversation ID, delegates storage to external service

Success criteria:
* `InMemoryAgentSession` accumulates messages correctly across invocations
* `ServiceIdAgentSession` creates and manages conversation IDs
* Thread safety for concurrent access

Dependencies:
* Phase 2

### Step 4.5: Implement ChatHistoryProvider and AIContextProvider

Create the two-phase lifecycle providers.

Files:
* cpp/src/Agents.AI/include/agents/ai/chat_history_provider.h — `class ChatHistoryProvider { public: virtual Task<void> invoking_async(std::vector<ChatMessage>& messages, ChatOptions& options, CancellationToken) = 0; virtual Task<void> invoked_async(std::vector<ChatMessage> const& messages, ChatResponse const& response, CancellationToken) = 0; }`
* cpp/src/Agents.AI/include/agents/ai/in_memory_chat_history_provider.h — Default implementation storing in `std::vector`
* cpp/src/Agents.AI/include/agents/ai/ai_context_provider.h — `class AIContextProvider { public: virtual Task<void> invoking_async(AIContext& context, CancellationToken) = 0; virtual Task<void> invoked_async(AIContext const& context, CancellationToken) = 0; }`
* cpp/src/Agents.AI/include/agents/ai/ai_context.h — `struct AIContext { std::string additional_instructions; std::vector<ChatMessage> additional_messages; std::vector<std::shared_ptr<AITool>> additional_tools; }`

Success criteria:
* Two-phase lifecycle called correctly: `invoking_async` before agent runs, `invoked_async` after
* `AIContext` data is merged into agent options at invocation time

Dependencies:
* Phase 2, Phase 3

### Step 4.6: Implement three-level middleware pipeline

Create the templated middleware system.

Files:
* cpp/src/Agents.AI/include/agents/ai/middleware.h — `template<typename Context, typename Next> class IMiddleware { public: virtual Task<void> invoke(Context& ctx, Next next, CancellationToken) = 0; }`
* cpp/src/Agents.AI/include/agents/ai/middleware_pipeline.h — `template<typename Context, typename Next, typename Middleware> class MiddlewarePipeline` — Builds onion model inside-out. Provides `Task<void> execute(Context& ctx, CancellationToken)`. Three instantiations:
  * Agent-level: `AgentMiddleware` (wraps `AIAgent::run_core_async`)
  * Chat-level: `ChatMiddleware` (wraps `IChatClient::get_response`)
  * Function-level: `FunctionMiddleware` (wraps `AIFunction::invoke_async`)

Success criteria:
* Onion model: middleware executes in registration order (first registered = outermost)
* Each level works independently
* `next()` properly delegates to inner middleware or terminal handler

Context references:
* .copilot-tracking/research/2026-02-20-cpp-port-implementation-plan.md (Lines 339-342) — Three-level middleware

Dependencies:
* Steps 4.1, 4.2

### Step 4.7: Implement ChatClientAgent

The largest single class — wraps `IChatClient` with full session lifecycle, options merging, function auto-invocation loop, and streaming.

Files:
* cpp/src/Agents.AI/include/agents/ai/chat_client_agent.h — Header with class declaration (~150 LOC)
* cpp/src/Agents.AI/src/chat_client_agent.cpp — Implementation (~1050 LOC). Key behaviors:
  * `run_core_async`:
    1. Merge session history, context provider data (instructions, tools, messages)
    2. Build `ChatOptions` merging agent-configured tools, user-provided tools, context tools
    3. Call `IChatClient::get_response()`
    4. If response contains `FunctionCallContent`:
       a. Invoke each function (with function-level middleware)
       b. Append `FunctionResultContent` to messages
       c. Re-call `get_response()` (loop up to `max_auto_invoke_iterations`)
    5. Return final `ChatResponse`
  * `run_core_streaming_async`:
    1. Same merging as above
    2. Call `IChatClient::get_streaming_response()`
    3. Accumulate chunks to detect tool calls
    4. If tool calls detected: invoke functions → re-stream
    5. Yield `ChatResponseUpdate` chunks to caller
  * Session management: load history before, save after (including assistant response and function results)

Success criteria:
* Single-turn: user message → assistant response
* Multi-turn: session accumulates messages correctly
* Tool invocation: 1 round, 3 rounds, max iterations reached
* Streaming: chunks yield in correct order
* Streaming with tools: detects tool calls, invokes, re-streams
* Cancellation: stops mid-invocation
* Options merging: agent instructions + user instructions + context provider instructions all appear

Context references:
* .copilot-tracking/research/2026-02-20-cpp-port-implementation-plan.md (Lines 343-348) — ChatClientAgent spec (~910 LOC in .NET)
* dotnet/src/Microsoft.Agents.AI/ — .NET ChatClientAgent implementation for behavior reference

Dependencies:
* Steps 4.1-4.6

### Step 4.8: Implement ServiceCollection and AIAgentBuilder

Create minimal DI and fluent builder.

Files:
* cpp/src/Agents.AI/include/agents/ai/service_collection.h — `class ServiceCollection` — Singleton and keyed registration: `register_singleton<T>(instance)`, `register_keyed<T>(key, instance)`. Resolution: `resolve<T>()`, `resolve<T>(key)`. Uses `std::type_index` as key, `std::any` as storage
* cpp/src/Agents.AI/include/agents/ai/ai_agent_builder.h — `class AIAgentBuilder`:
  * `.with_chat_client(shared_ptr<IChatClient>)`
  * `.with_instructions(string)`
  * `.with_tools(vector<shared_ptr<AITool>>)`
  * `.with_middleware(shared_ptr<AgentMiddleware>)`
  * `.with_session_store(shared_ptr<...>)`
  * `.with_context_provider(shared_ptr<AIContextProvider>)`
  * `.build() -> shared_ptr<AIAgent>` — Creates `ChatClientAgent` with all configuration applied

Success criteria:
* `ServiceCollection` registers and resolves types correctly
* Builder fluent API produces a correctly configured `ChatClientAgent`
* Missing required configuration (no chat client) throws at build time

Dependencies:
* Steps 4.1-4.7

### Step 4.9: Write comprehensive unit tests

Files:
* cpp/tests/Agents.AI.Tests/chat_client_agent_tests.cpp — Tests with mock `IChatClient`: single-turn, multi-turn, tool invocation (1 round, 3 rounds, max iterations), streaming, streaming with tools, cancellation, options merging
* cpp/tests/Agents.AI.Tests/middleware_tests.cpp — Tests: single middleware, multiple middleware (verify ordering), agent-level + chat-level interaction
* cpp/tests/Agents.AI.Tests/session_tests.cpp — Tests: InMemoryAgentSession lifecycle, message accumulation, concurrent access
* cpp/tests/Agents.AI.Tests/builder_tests.cpp — Tests: builder produces correct agent, missing config errors, multiple tools/middleware
* cpp/tests/Agents.AI.Tests/context_provider_tests.cpp — Tests: context injection, two-phase lifecycle ordering

Success criteria:
* All tests pass on MSVC, GCC, Clang
* Mock `IChatClient` used consistently (no real API calls)
* Middleware onion ordering verified

Dependencies:
* Steps 4.1-4.8

## Implementation Phase 5: First Provider — OpenAI

<!-- parallelizable: false -->

### Step 5.1: Implement OpenAI request/response model structs

Create all OpenAI-specific types required for API communication.

Files:
* cpp/src/Agents.OpenAI/include/agents/openai/models/ — ~50 structs in organized headers:
  * `chat_completion_request.h` — `CreateChatCompletionRequest { model, messages, tools, ... }`
  * `chat_completion_response.h` — `ChatCompletion { id, choices, usage, ... }`
  * `chat_completion_chunk.h` — `ChatCompletionChunk { id, choices, ... }` (streaming)
  * `responses_request.h` — `CreateResponseRequest { model, input, ... }`
  * `responses_response.h` — `Response { id, output, status, ... }`
  * `tool_types.h` — OpenAI tool/function definitions
  * `common.h` — Shared types: `OpenAIMessage`, `ToolCall`, `FunctionObject`
  * Each with `glz::meta` for JSON serialization

Success criteria:
* All structs serialize to match OpenAI API JSON format
* All structs deserialize from real OpenAI API responses

Context references:
* .copilot-tracking/research/2026-02-20-cpp-port-implementation-plan.md (Lines 367-378) — Provider architecture

Dependencies:
* Phase 2

### Step 5.2: Implement OpenAIChatClient for Chat Completions

The primary OpenAI provider implementation.

Files:
* cpp/src/Agents.OpenAI/include/agents/openai/openai_chat_client.h — `class OpenAIChatClient : public IChatClient`
* cpp/src/Agents.OpenAI/src/openai_chat_client.cpp — Implementation:
  * `get_response()`: Convert `ChatMessage` → OpenAI messages format, POST to `/v1/chat/completions`, parse response → `ChatResponse`
  * `get_streaming_response()`: POST with `stream: true`, SSE parse → yield `ChatResponseUpdate`
  * Message conversion: `ChatRole` → OpenAI role strings, `AIContent` variants → OpenAI content blocks
  * Auth: `Authorization: Bearer <api_key>` header

Success criteria:
* Non-streaming returns valid `ChatResponse` with content and usage
* Streaming yields `ChatResponseUpdate` chunks in correct order
* Message format matches OpenAI API specification

Dependencies:
* Phase 4 (IChatClient), Phase 1 (HttpClient, SSE parser)

### Step 5.3: Implement OpenAI Responses API client

Support the newer Responses API with its 19 SSE event types.

Files:
* cpp/src/Agents.OpenAI/include/agents/openai/openai_responses_client.h — Additional client class or method overloads on `OpenAIChatClient`
* cpp/src/Agents.OpenAI/src/openai_responses_client.cpp — POST to `/v1/responses`. Parse 19 SSE event types: `response.created`, `response.in_progress`, `response.completed`, `response.failed`, `response.output_item.added`, `response.output_item.done`, `response.content_part.added`, `response.content_part.done`, `response.output_text.delta`, `response.output_text.done`, `response.function_call_arguments.delta`, `response.function_call_arguments.done`, `response.refusal.delta`, `response.refusal.done`, `response.file_search_call.started`, `response.file_search_call.completed`, `response.web_search_call.started`, `response.web_search_call.completed`, `rate_limits.updated`

Success criteria:
* All 19 event types parse correctly
* Response lifecycle (created → in_progress → completed/failed) tracked
* Function call arguments accumulate from delta events

Dependencies:
* Step 5.2

### Step 5.4: Implement OpenAI Assistants API client

Support the Assistants API with thread/message/run lifecycle.

Files:
* cpp/src/Agents.OpenAI/include/agents/openai/openai_assistants_client.h — `class OpenAIAssistantsClient`
* cpp/src/Agents.OpenAI/src/openai_assistants_client.cpp — Thread management: create thread, add messages, create run, poll/stream status, retrieve results

Success criteria:
* Thread lifecycle: create → add message → create run → poll → retrieve
* Streaming support for run status events
* Tool invocation via `requires_action` status

Dependencies:
* Steps 5.2, 5.3

### Step 5.5: Implement AzureOpenAIChatClient variant

Azure-specific subclass.

Files:
* cpp/src/Agents.OpenAI/include/agents/openai/azure_openai_chat_client.h — `class AzureOpenAIChatClient : public OpenAIChatClient`
* cpp/src/Agents.OpenAI/src/azure_openai_chat_client.cpp — Override base URL to `https://<resource>.openai.azure.com/openai/deployments/<deployment>`, add `api-version` query parameter, Azure AD token auth via `azure-identity-cpp` `DefaultAzureCredential`

Success criteria:
* Request URLs match Azure OpenAI format
* Azure AD token acquisition and refresh works
* Falls back to API key auth when no credential

Dependencies:
* Step 5.2

### Step 5.6: Implement tool/function calling serialization and auto-invoke loop

Wire up tool definitions and invocation with the ChatClientAgent loop.

Files:
* cpp/src/Agents.OpenAI/src/openai_chat_client.cpp — Additions to serialize `vector<shared_ptr<AITool>>` into OpenAI `tools` JSON array. Parse `tool_calls` from response. Convert `FunctionCallContent` ↔ OpenAI tool call format. Feed `FunctionResultContent` back as tool role message

Success criteria:
* Tool definitions appear in request JSON matching OpenAI format
* `tool_calls` in response create `FunctionCallContent` objects
* `FunctionResultContent` serializes as tool role message
* End-to-end: agent sends tools → model calls tool → agent invokes → sends result → model responds

Dependencies:
* Steps 5.2, Phase 3

### Step 5.7: Create extension factory functions

Convenience API for quick setup.

Files:
* cpp/src/Agents.OpenAI/include/agents/openai/openai_extensions.h — Factory functions:
  * `std::shared_ptr<IChatClient> openai_as_chat_client(OpenAIConfig config)` — Creates configured `OpenAIChatClient`
  * `std::shared_ptr<AIAgent> openai_as_agent(OpenAIConfig config, std::string instructions)` — Creates `ChatClientAgent` via builder
  * `struct OpenAIConfig { std::string api_key; std::string model = "gpt-4o"; std::optional<std::string> base_url; std::optional<std::string> api_version; std::optional<std::string> organization; }`

Success criteria:
* `openai_as_agent` produces a working agent in 1 call
* Config defaults work correctly

Dependencies:
* Steps 5.2-5.6, Phase 4

### Step 5.8: Write integration and mock-server tests

Files:
* cpp/tests/Agents.OpenAI.Tests/openai_chat_client_tests.cpp — Mock server tests: non-streaming, streaming, tool calling, error responses
* cpp/tests/Agents.OpenAI.Tests/openai_responses_tests.cpp — Mock server tests: all 19 event types
* cpp/tests/Agents.OpenAI.Tests/openai_integration_tests.cpp — Live API tests (gated by `OPENAI_API_KEY` env var): chat, streaming, tool calling
* cpp/tests/Agents.OpenAI.Tests/mock_openai_server.h — Boost.Beast-based HTTP server returning canned responses

Success criteria:
* Mock tests pass on all compilers without API key
* Integration tests pass against live OpenAI API
* End-to-end stack validated: HTTP → SSE → IChatClient → ChatClientAgent → AgentResponse

Dependencies:
* Steps 5.1-5.7

## Implementation Phase 6A: Azure AI Provider

<!-- parallelizable: true -->

### Step 6A.1: Implement AzureAIChatClient with REST calls and tool schema transforms

Files:
* cpp/src/Agents.AzureAI/include/agents/azure_ai/azure_ai_chat_client.h — `class AzureAIChatClient : public IChatClient`
* cpp/src/Agents.AzureAI/src/azure_ai_chat_client.cpp — REST calls to Azure AI Inference API. `DelegatingChatClient` pattern for tool validation: Azure AI has stricter tool schema requirements than OpenAI. Transform tool definitions to comply with Azure-specific constraints (required properties, enum restrictions)

Success criteria:
* Chat completions work against Azure AI Inference endpoint
* Tool schema transforms produce Azure-compliant definitions
* Streaming works

Context references:
* .copilot-tracking/research/2026-02-20-cpp-port-implementation-plan.md (Lines 402-418) — Azure AI provider spec

Dependencies:
* Phase 5

### Step 6A.2: Implement AzureAIAgentSession for server-managed threads

Files:
* cpp/src/Agents.AzureAI.Persistent/include/agents/azure_ai/azure_ai_agent_session.h — `class AzureAIAgentSession : public ServiceIdAgentSession`
* cpp/src/Agents.AzureAI.Persistent/src/azure_ai_agent_session.cpp — Server-managed threads via Azure AI Agent Service REST API. Create thread, post message, retrieve thread messages

Success criteria:
* Thread creation returns valid thread ID
* Messages persist server-side across requests

Dependencies:
* Step 6A.1, Phase 4

### Step 6A.3: Integrate azure-identity-cpp for DefaultAzureCredential

Files:
* cpp/src/Agents.AzureAI/src/azure_identity_adapter.h — Adapter wrapping `azure-identity-cpp` `DefaultAzureCredential`. Token caching and refresh. `Task<std::string> get_token(std::string scope, CancellationToken)`

Success criteria:
* Token acquired via environment credentials, managed identity, or CLI
* Token cached and refreshed before expiry

Dependencies:
* Step 6A.1

### Step 6A.4: Write integration tests

Files:
* cpp/tests/Agents.AzureAI.Tests/ — Tests gated by `AZURE_AI_ENDPOINT` and `AZURE_AI_API_KEY` env vars. Chat, streaming, tool calling, session management

Success criteria:
* All tests pass against live Azure AI endpoint

Dependencies:
* Steps 6A.1-6A.3

## Implementation Phase 6B: Anthropic Provider

<!-- parallelizable: true -->

### Step 6B.1: Implement AnthropicChatClient with Messages API and streaming

Files:
* cpp/src/Agents.Anthropic/include/agents/anthropic/anthropic_chat_client.h — `class AnthropicChatClient : public IChatClient`
* cpp/src/Agents.Anthropic/src/anthropic_chat_client.cpp — POST to `https://api.anthropic.com/v1/messages`. Handle content blocks: `text`, `tool_use`, `tool_result`. Streaming SSE events: `message_start`, `content_block_start`, `content_block_delta`, `content_block_stop`, `message_delta`, `message_stop`

Success criteria:
* Non-streaming returns valid response with text content
* Streaming yields updates for each content block

Context references:
* .copilot-tracking/research/2026-02-20-cpp-port-implementation-plan.md (Lines 420-432) — Anthropic provider spec

Dependencies:
* Phase 5

### Step 6B.2: Implement Anthropic tool_use/tool_result mapping

Files:
* cpp/src/Agents.Anthropic/src/anthropic_chat_client.cpp — Map `tool_use` content blocks → `FunctionCallContent`. Map `FunctionResultContent` → `tool_result` content block for next request

Success criteria:
* Tool calling round-trips correctly through Anthropic API format

Dependencies:
* Step 6B.1

### Step 6B.3: Write integration and mock tests

Files:
* cpp/tests/Agents.Anthropic.Tests/ — Mock server tests + integration tests gated by `ANTHROPIC_API_KEY`

Success criteria:
* All tests pass

Dependencies:
* Steps 6B.1-6B.2

## Implementation Phase 6C: Copilot Studio and GitHub Copilot Providers

<!-- parallelizable: true -->

### Step 6C.1: Implement CopilotStudioAgent with DirectLine protocol

Files:
* cpp/src/Agents.CopilotStudio/include/agents/copilotstudio/copilot_studio_agent.h — `class CopilotStudioAgent : public AIAgent` — Direct subclass (not IChatClient-based)
* cpp/src/Agents.CopilotStudio/src/copilot_studio_agent.cpp — DirectLine REST API: start conversation, post activity, poll for response activities. `CopilotStudioAgentSession` with DirectLine conversation ID

Success criteria:
* Conversation start returns valid conversation ID
* Activity posting and response polling work
* Session management ties to conversation lifecycle

Context references:
* .copilot-tracking/research/2026-02-20-cpp-port-implementation-plan.md (Lines 434-441) — Copilot Studio spec

Dependencies:
* Phase 4

### Step 6C.2: Implement GitHubCopilotAgent with channel-based streaming

Files:
* cpp/src/Agents.GitHub.Copilot/include/agents/github_copilot/github_copilot_agent.h — `class GitHubCopilotAgent : public AIAgent` — Direct subclass with channel-based streaming
* cpp/src/Agents.GitHub.Copilot/src/github_copilot_agent.cpp — Channel-based streaming pattern. `GitHubCopilotAgentSession` for session management

Success criteria:
* Streaming produces correct updates via channels
* Session management works

Dependencies:
* Phase 4

### Step 6C.3: Write mock tests for both providers

Files:
* cpp/tests/Agents.CopilotStudio.Tests/ — Mock DirectLine server tests
* cpp/tests/Agents.GitHub.Copilot.Tests/ — Mock server tests

Success criteria:
* All tests pass on all compilers

Dependencies:
* Steps 6C.1-6C.2

## Implementation Phase 7: Cross-Cutting Agents

<!-- parallelizable: false -->

### Step 7.1: Implement LoggingAgent via spdlog

Files:
* cpp/src/Agents.AI/include/agents/ai/logging_agent.h — `class LoggingAgent : public DelegatingAIAgent`
* cpp/src/Agents.AI/src/logging_agent.cpp — Logs: invocation start (agent name, message count), invocation end (response summary, duration), errors. Uses spdlog structured logging

Success criteria:
* Structured log output for each invocation
* Configurable log level

Dependencies:
* Phase 4

### Step 7.2: Implement OpenTelemetryAgent with GenAI semantic conventions

Files:
* cpp/src/Agents.AI/include/agents/ai/opentelemetry_agent.h — `class OpenTelemetryAgent : public DelegatingAIAgent`
* cpp/src/Agents.AI/src/opentelemetry_agent.cpp — Creates spans per invocation with GenAI semantic convention attributes:
  * `gen_ai.system` — Provider name
  * `gen_ai.request.model` — Model ID
  * `gen_ai.request.max_tokens` — Max output tokens
  * `gen_ai.request.temperature` — Temperature
  * `gen_ai.usage.input_tokens` — Input token count
  * `gen_ai.usage.output_tokens` — Output token count
  * `gen_ai.response.finish_reasons` — Finish reasons array
  * Span status: OK on success, ERROR on failure

Success criteria:
* Spans created with correct attributes
* Spans exported via OTLP (verified with in-memory exporter)
* Nested spans for agent chains

Dependencies:
* Phase 4

### Step 7.3: Implement FunctionInvocationDelegatingAgent and AnonymousDelegatingAIAgent

Files:
* cpp/src/Agents.AI/include/agents/ai/function_invocation_delegating_agent.h — `class FunctionInvocationDelegatingAgent : public DelegatingAIAgent` — Intercepts function invocations for approval/logging/modification
* cpp/src/Agents.AI/include/agents/ai/anonymous_delegating_ai_agent.h — `class AnonymousDelegatingAIAgent : public DelegatingAIAgent` — Initialized with lambdas for `run_core_async` / `run_core_streaming_async`

Success criteria:
* Function invocation interception works (can approve/deny/modify)
* Anonymous agent wraps lambdas correctly

Dependencies:
* Phase 4

### Step 7.4: Implement AIHostAgent and builder API enhancements

Files:
* cpp/src/Agents.AI/include/agents/ai/ai_host_agent.h — `class AIHostAgent : public DelegatingAIAgent` — Routes requests to named agents based on configuration
* cpp/src/Agents.AI/include/agents/ai/ai_agent_builder.h — Add methods: `.with_logging()`, `.with_telemetry(TracerProvider)`, `.with_function_invocation_handler(handler)`

Success criteria:
* `AIHostAgent` routes to correct agent by name
* Builder methods produce correctly layered decorator chains

Dependencies:
* Steps 7.1-7.3

### Step 7.5: Write unit tests for all cross-cutting agents

Files:
* cpp/tests/Agents.AI.Tests/logging_agent_tests.cpp — Verify structured log output
* cpp/tests/Agents.AI.Tests/otel_agent_tests.cpp — Verify span attributes with in-memory exporter
* cpp/tests/Agents.AI.Tests/decorator_chain_tests.cpp — Verify chain: `Logging → OTel → FunctionInvocation → ChatClientAgent` executes in correct order
* cpp/tests/Agents.AI.Tests/builder_decorator_tests.cpp — Verify builder produces correct layering

Success criteria:
* All decorator behaviors verified
* Chain ordering correct

Dependencies:
* Steps 7.1-7.4

## Implementation Phase 8A: HTTP Server Foundation

<!-- parallelizable: false -->

### Step 8A.1: Implement HttpRouter with path parameters and middleware pipeline

Files:
* cpp/src/Agents.Hosting/include/agents/hosting/http_router.h — `class HttpRouter` — Route registration: `add_route(method, path_pattern, handler)`. Path parameters: `/{agentName}/v1/...` extracts `agentName`. Middleware pipeline: pre/post processing per route or global
* cpp/src/Agents.Hosting/src/http_router.cpp — Trie-based path matching with parameter extraction, method filtering

Success criteria:
* Path parameters extracted correctly
* 404 for unmatched routes
* Method filtering (GET vs POST)
* Middleware executes in order

Dependencies:
* Phase 1

### Step 8A.2: Implement SseWriter for chunked SSE response streaming

Files:
* cpp/src/Agents.Hosting/include/agents/hosting/sse_writer.h — `class SseWriter` — `Task<void> write_event(std::string_view event, std::string_view data)`, `Task<void> close()`. Sets headers: `Content-Type: text/event-stream`, `Cache-Control: no-cache,no-store`, `Connection: keep-alive`. Uses Beast chunked encoding

Success criteria:
* Events written in correct SSE format (`event:`, `data:`, blank line)
* Client receives all events in order
* Proper connection cleanup on close

Dependencies:
* Phase 1

### Step 8A.3: Implement AgentHost server and AgentHostBuilder

Files:
* cpp/src/Agents.Hosting/include/agents/hosting/agent_host.h — `class AgentHost` — Listens on port, accepts connections, dispatches to router. TLS support via OpenSSL. Graceful shutdown via `std::stop_source`
* cpp/src/Agents.Hosting/include/agents/hosting/agent_host_builder.h — `class AgentHostBuilder` — Fluent: `.with_port(port)`, `.with_tls(cert, key)`, `.with_agent(name, agent)`, `.with_middleware(mw)`, `.map_openai()`, `.map_a2a()`, `.map_agui()`, `.build() -> AgentHost`
* cpp/src/Agents.Hosting/src/agent_host.cpp — Boost.Beast HTTP/HTTPS acceptor, connection handling, dispatch to router

Success criteria:
* Server accepts HTTP and HTTPS connections
* Routes dispatch to correct handlers
* Graceful shutdown: in-flight requests complete or terminate cleanly
* Builder produces configured server

Dependencies:
* Steps 8A.1, 8A.2

### Step 8A.4: Implement server middleware

Files:
* cpp/src/Agents.Hosting/include/agents/hosting/middleware/ — CORS middleware (configurable origins, methods, headers), request logging middleware (spdlog), error handling middleware (catch exceptions, return 500 JSON), authentication middleware (Bearer token validation)

Success criteria:
* CORS headers set correctly for preflight and actual requests
* Errors produce structured JSON error responses
* Auth middleware rejects invalid tokens

Dependencies:
* Step 8A.1

### Step 8A.5: Write tests for route matching, SSE round-trip, concurrent connections

Files:
* cpp/tests/Agents.Hosting.Tests/router_tests.cpp — Route matching, parameter extraction, 404, method filtering
* cpp/tests/Agents.Hosting.Tests/sse_tests.cpp — Write events, read back, verify format and ordering
* cpp/tests/Agents.Hosting.Tests/server_tests.cpp — Concurrent connections, graceful shutdown, TLS

Success criteria:
* All routing patterns match correctly
* SSE round-trip preserves event data
* Concurrent requests handled without data corruption

Dependencies:
* Steps 8A.1-8A.4

## Implementation Phase 8B: OpenAI-Compatible Hosting

<!-- parallelizable: false -->

### Step 8B.1: Implement Chat Completions endpoint

Files:
* cpp/src/Agents.Hosting.OpenAI/include/agents/hosting/openai/ — Route handler headers
* cpp/src/Agents.Hosting.OpenAI/src/chat_completions_handler.cpp — `POST /{agentName}/v1/chat/completions/`:
  1. Deserialize `CreateChatCompletionRequest`
  2. Convert to `vector<ChatMessage>` + `ChatOptions`
  3. Call agent `run_async()` (non-streaming) or `run_streaming_async()` (when `stream: true`)
  4. Non-streaming: serialize `ChatCompletion` JSON response
  5. Streaming: SSE `ChatCompletionChunk` events, final `[DONE]` sentinel
* cpp/src/Agents.Hosting.OpenAI/src/models/ — ~12 model structs for Chat Completions format

Success criteria:
* Request accepted by OpenAI Python client library (format compatibility)
* Non-streaming response matches OpenAI JSON format
* Streaming events match OpenAI SSE format

Context references:
* .copilot-tracking/research/2026-02-20-cpp-port-implementation-plan.md (Lines 501-520) — OpenAI hosting spec
* .copilot-tracking/subagent/2026-02-20/hosting-protocols-analysis.md — Protocol format details

Dependencies:
* Phase 8A, Phase 4

### Step 8B.2: Implement Responses API endpoints

Files:
* cpp/src/Agents.Hosting.OpenAI/src/responses_handler.cpp — 5 endpoints: `CreateResponse`, `GetResponse`, `CancelResponse`, `DeleteResponse`, `ListInputItems`. Streaming: map internal `AgentResponseUpdate` to 19 OpenAI Responses API event types using 11 distinct generator strategies (matching .NET `StreamingEventGenerator` classes)

Success criteria:
* All 19 event types stream in correct order
* Response lifecycle correctly managed
* Cancel and delete endpoints work

Dependencies:
* Step 8B.1

### Step 8B.3: Implement Conversations API endpoints

Files:
* cpp/src/Agents.Hosting.OpenAI/src/conversations_handler.cpp — 8 endpoints for multi-turn conversation management: create, get, list, delete conversations; add, get, list, delete messages. Pagination via cursor-based approach

Success criteria:
* CRUD operations for conversations and messages work
* Pagination returns correct pages

Dependencies:
* Step 8B.1

### Step 8B.4: Write interop tests

Files:
* cpp/tests/Agents.Hosting.OpenAI.Tests/ — Start C++ server, call with OpenAI Python/JS client library, verify responses. Test: chat completions, streaming, tool calling, conversations

Success criteria:
* OpenAI Python client successfully communicates with C++ server
* All response formats match expected specification

Dependencies:
* Steps 8B.1-8B.3

## Implementation Phase 8C: A2A Protocol Hosting

<!-- parallelizable: true -->

### Step 8C.1: Implement Agent Card discovery and JSON-RPC 2.0 dispatcher

Files:
* cpp/src/Agents.Hosting.A2A/include/agents/hosting/a2a/ — A2A hosting headers
* cpp/src/Agents.Hosting.A2A/src/agent_card_handler.cpp — `GET /.well-known/agent.json` returns agent capabilities (name, description, URL, supported protocols, skills)
* cpp/src/Agents.Hosting.A2A/src/jsonrpc_dispatcher.cpp — `POST /` receives JSON-RPC 2.0 requests, routes to methods: `tasks/send`, `tasks/get`, `tasks/cancel`, `tasks/sendSubscribe`, `tasks/resubscribe`. Validates JSON-RPC format (id, method, params, jsonrpc: "2.0")

Success criteria:
* Agent card served at well-known URL
* JSON-RPC dispatch routes to correct handler
* Invalid requests return JSON-RPC error responses

Dependencies:
* Phase 8A

### Step 8C.2: Implement task lifecycle manager and SSE streaming

Files:
* cpp/src/Agents.Hosting.A2A/src/task_manager.cpp — `class TaskManager`: create task, update task status, cancel task. Map task lifecycle to agent invocations. `tasks/sendSubscribe` → SSE stream of `AgentMessage`, `AgentTask`, `TaskUpdateEvent`

Success criteria:
* Task state transitions: submitted → working → completed/failed/canceled
* SSE streaming delivers updates in real-time
* Task cancellation propagates to agent

Dependencies:
* Step 8C.1

### Step 8C.3: Implement A2AAgent remote proxy

Files:
* cpp/src/Agents.A2A/include/agents/a2a/a2a_agent.h — `class A2AAgent : public AIAgent` — Remote proxy calling another agent via A2A protocol. Continuation token support for multi-turn
* cpp/src/Agents.A2A/src/a2a_agent.cpp — HTTP client calls to remote A2A server, JSON-RPC request construction, SSE response parsing

Success criteria:
* Remote agent invocation works end-to-end
* Continuation tokens enable multi-turn conversations
* Streaming via SSE works for remote agents

Dependencies:
* Phase 4, Phase 1

### Step 8C.4: Write interop tests

Files:
* cpp/tests/Agents.Hosting.A2A.Tests/ — C++ A2A server ↔ .NET A2A client, C++ A2A client ↔ .NET A2A server

Success criteria:
* Cross-language A2A communication works

Dependencies:
* Steps 8C.1-8C.3

## Implementation Phase 8D: AG-UI Protocol Hosting

<!-- parallelizable: true -->

### Step 8D.1: Implement AG-UI SSE endpoint with 12 event types

Files:
* cpp/src/Agents.Hosting.AGUI/include/agents/hosting/agui/ — AG-UI hosting headers
* cpp/src/Agents.Hosting.AGUI/src/agui_handler.cpp — `POST /` → SSE stream. 12 event types:
  * `RUN_STARTED`, `RUN_FINISHED`, `RUN_ERROR`
  * `TEXT_MESSAGE_START`, `TEXT_MESSAGE_CONTENT`, `TEXT_MESSAGE_END`
  * `TOOL_CALL_START`, `TOOL_CALL_ARGS`, `TOOL_CALL_END`
  * `STATE_SNAPSHOT`, `STATE_DELTA`, `CUSTOM`
* cpp/src/Agents.Hosting.AGUI/src/agui_protocol_types.h — 12 event structs, 5 message types, tool/context types (~20 structs)

Success criteria:
* All 12 event types stream with correct names and payloads
* Event ordering matches protocol specification

Dependencies:
* Phase 8A

### Step 8D.2: Implement AGUIChatClient

Files:
* cpp/src/Agents.AGUI/include/agents/agui/agui_chat_client.h — `class AGUIChatClient : public DelegatingChatClient`
* cpp/src/Agents.AGUI/src/agui_chat_client.cpp — Wraps tool execution with AG-UI events. Maps `AgentResponseUpdate` to AG-UI SSE events. Manages message lifecycle and state snapshots

Success criteria:
* Tool invocations produce correct AG-UI event sequences
* State management works

Dependencies:
* Phase 4

### Step 8D.3: Write interop tests

Files:
* cpp/tests/Agents.Hosting.AGUI.Tests/ — C++ AG-UI server ↔ CopilotKit client

Success criteria:
* CopilotKit client successfully communicates with C++ server

Dependencies:
* Steps 8D.1-8D.2

## Implementation Phase 9A: Workflow Graph Executor

<!-- parallelizable: false -->

### Step 9A.1: Implement WorkflowNode and edge types

Files:
* cpp/src/Agents.Workflows/include/agents/workflows/workflow_node.h — `class WorkflowNode { public: virtual ~WorkflowNode() = default; virtual Task<void> execute_async(WorkflowContext& context, CancellationToken) = 0; std::string name; std::vector<std::shared_ptr<Edge>> edges; }`
* cpp/src/Agents.Workflows/include/agents/workflows/edges.h — Edge types:
  * `DirectEdge` — Unconditional transition to next node
  * `ConditionalEdge` — Expression-based routing (evaluates condition, routes to matching target)
  * `FanOutEdge` — Parallel execution: spawns multiple child nodes
  * `FanInEdge` — Waits for all/any parallel children to complete

Success criteria:
* Node execution invokes virtual method
* Edge traversal routes correctly
* Fan-out spawns concurrent tasks

Context references:
* .copilot-tracking/research/2026-02-20-cpp-port-implementation-plan.md (Lines 560-575) — Workflow executor spec

Dependencies:
* Phase 4

### Step 9A.2: Implement WorkflowExecutor graph traversal engine

Files:
* cpp/src/Agents.Workflows/include/agents/workflows/workflow_executor.h — `class WorkflowExecutor`
* cpp/src/Agents.Workflows/src/workflow_executor.cpp — Graph traversal: topological execution respecting edges. Fan-out uses `when_all()`. Streaming support: yield updates as nodes complete. Cycle detection (error on cycles, these are DAGs). Context propagation between nodes

Success criteria:
* Linear execution: A → B → C in order
* Fan-out: A → {B, C} → D concurrently
* Conditional: correct branch selected
* Cycle detection raises error

Dependencies:
* Step 9A.1

### Step 9A.3: Implement builder APIs and checkpoint interface

Files:
* cpp/src/Agents.Workflows/include/agents/workflows/workflow_builder.h — `build_sequential(nodes)`, `build_concurrent(nodes)`, `build_handoff(agents)`, `build_group_chat(agents, strategy)`
* cpp/src/Agents.Workflows/include/agents/workflows/checkpoint_store.h — `class ICheckpointStore { public: virtual Task<void> save(std::string key, glz::json_t state, CancellationToken) = 0; virtual Task<glz::json_t> load(std::string key, CancellationToken) = 0; }`
* cpp/src/Agents.Workflows/src/in_memory_checkpoint_store.cpp — Default in-memory implementation

Success criteria:
* Builder convenience methods produce correct graph topologies
* Checkpoint save/load round-trips correctly

Dependencies:
* Step 9A.2

### Step 9A.4: Add OpenTelemetry integration

Files:
* cpp/src/Agents.Workflows/src/workflow_executor.cpp — Add span creation per node execution with parent span for workflow. Attributes: `workflow.node.name`, `workflow.node.duration`, `workflow.name`

Success criteria:
* Spans appear for each node execution
* Parent-child span relationship correct

Dependencies:
* Step 9A.2, Phase 7

### Step 9A.5: Write unit tests

Files:
* cpp/tests/Agents.Workflows.Tests/executor_tests.cpp — Linear flow, branching, fan-out/in, cycle detection
* cpp/tests/Agents.Workflows.Tests/builder_tests.cpp — Builder convenience methods produce correct graphs
* cpp/tests/Agents.Workflows.Tests/checkpoint_tests.cpp — Save/load round-trip, missing key handling

Success criteria:
* All graph patterns execute correctly
* Checkpoint data survives round-trip

Dependencies:
* Steps 9A.1-9A.4

## Implementation Phase 9B: Declarative Workflows and Lua Engine

<!-- parallelizable: false -->

### Step 9B.1: Implement YAML parsing of agent and workflow definitions

Files:
* cpp/src/Agents.Workflows.Declarative/include/agents/workflows/declarative/yaml_parser.h — Parse YAML files defining agents, workflows, steps, conditions, variable declarations. Produce intermediate representation (IR) structs
* cpp/src/Agents.Workflows.Declarative/src/yaml_parser.cpp — yaml-cpp based parsing. Handle: agent definitions (model, instructions, tools), workflow steps (sequential, parallel, conditional), variable references, tool references

Success criteria:
* Sample YAML files from `agent-samples/` and `workflow-samples/` parse correctly
* IR structs contain all required data

Context references:
* agent-samples/ — YAML agent definition examples
* workflow-samples/ — YAML workflow definition examples

Dependencies:
* Phase 9A

### Step 9B.2: Implement Lua expression engine via sol2

Files:
* cpp/src/Agents.Workflows.Declarative/include/agents/workflows/declarative/expression_engine.h — `class ExpressionEngine`
* cpp/src/Agents.Workflows.Declarative/src/expression_engine.cpp — Wraps sol2 state. Methods:
  * `glz::json_t evaluate(std::string_view expr)` — Evaluate arbitrary expression
  * `bool evaluate_condition(std::string_view expr)` — Evaluate boolean condition
  * `std::string evaluate_string(std::string_view expr)` — Evaluate string expression
  * Variable set/get via Lua global table

Success criteria:
* Lua expressions evaluate correctly
* Variables set and retrieved across evaluations
* Error handling for invalid expressions

Dependencies:
* Step 9B.1

### Step 9B.3: Implement PowerFx compatibility shim

Files:
* cpp/src/Agents.Workflows.Declarative/include/agents/workflows/declarative/powerfx_translator.h — Regex-based translator
* cpp/src/Agents.Workflows.Declarative/src/powerfx_translator.cpp — Translation rules:
  * `Set(var, val)` → `var = val`
  * `If(cond, then, else)` → Lua `if cond then ... else ... end`
  * `Concatenate(a, b, ...)` → `a .. b .. ...`
  * `Text(n)` → `tostring(n)`
  * `Topic.var` → Lua table access `Topic.var`
  * `Lower(s)` → `string.lower(s)`
  * `Upper(s)` → `string.upper(s)`
  * `Len(s)` → `#s`
  * Enumerate all PowerFx expressions used in .NET/Python YAML samples as translation targets

Success criteria:
* All PowerFx expressions from existing YAML samples translate correctly
* Translated expressions evaluate to same results as .NET PowerFx engine

Dependencies:
* Step 9B.2

### Step 9B.4: Implement variable scoping and WorkflowAgentProvider

Files:
* cpp/src/Agents.Workflows.Declarative/src/variable_scope.cpp — 5 scope levels matching .NET: Local, Global, System, Environment, Topic. Lua environment tables per scope. Scope resolution: Local → Topic → Global → System → Environment
* cpp/src/Agents.Workflows.Declarative/include/agents/workflows/declarative/workflow_agent_provider.h — `class WorkflowAgentProvider` — Resolves agent references in YAML to `AIAgent` instances. Registry pattern with named agents

Success criteria:
* Local scope isolates between workflow steps
* Global scope shared across workflow
* Environment scope reads env vars
* Agent references resolve to registered agents

Dependencies:
* Steps 9B.2, 9B.3

### Step 9B.5: Implement declarative workflow builder

Files:
* cpp/src/Agents.Workflows.Declarative/src/declarative_workflow_builder.cpp — YAML IR → `WorkflowExecutor`:
  1. Parse steps into `WorkflowNode` instances
  2. Wire edges based on conditions (PowerFx or Lua expressions)
  3. Handle flow control: goto, loop, break, fork/join
  4. Resolve agent and tool references via providers
  5. Return configured `WorkflowExecutor`

Success criteria:
* YAML-defined workflow matches manually-built equivalent in behavior
* All flow control patterns work

Dependencies:
* Steps 9B.1-9B.4, Phase 9A

### Step 9B.6: Implement build-time code generation

Files:
* cpp/cmake/GenerateRouting.cmake — CMake custom command that scans source files for routing attribute markers and generates routing dispatch code. Replaces Roslyn source generators from .NET
* cpp/tools/generate_routing.py — Python script invoked by CMake. Scans for marked handler functions, generates C++ routing switch/dispatch

Success criteria:
* Generated routing code compiles and dispatches correctly
* CMake re-generates when source files change

Dependencies:
* Phase 9A

### Step 9B.7: Write unit tests

Files:
* cpp/tests/Agents.Workflows.Declarative.Tests/yaml_parser_tests.cpp — Parse sample YAML files, verify IR structs
* cpp/tests/Agents.Workflows.Declarative.Tests/expression_tests.cpp — Lua native + PowerFx-translated expressions
* cpp/tests/Agents.Workflows.Declarative.Tests/variable_scope_tests.cpp — Scope isolation, resolution order
* cpp/tests/Agents.Workflows.Declarative.Tests/declarative_workflow_tests.cpp — Full declarative workflow execution matching sample YAMLs

Success criteria:
* All YAML samples from repo parse and execute correctly
* Expression evaluation matches expected values

Dependencies:
* Steps 9B.1-9B.6

## Implementation Phase 10: Storage and Integrations

<!-- parallelizable: false -->

### Step 10.1: Implement in-memory stores

Files:
* cpp/src/Agents.AI/include/agents/ai/stores/ — `InMemoryChatHistoryStore`, `InMemoryCheckpointStore`, `InMemoryAgentSessionStore`. Each backed by `std::unordered_map` with `std::mutex` for thread safety

Success criteria:
* Concurrent read/write safety
* Store and retrieve operations work correctly

Dependencies:
* Phase 4

### Step 10.2: Implement Cosmos DB storage provider via REST API

Files:
* cpp/src/Agents.Storage.Cosmos/include/agents/storage/cosmos/cosmos_chat_history_provider.h — `class CosmosDbChatHistoryProvider : public ChatHistoryProvider`
* cpp/src/Agents.Storage.Cosmos/src/cosmos_chat_history_provider.cpp — REST API calls via `HttpClient` (no C++ Cosmos SDK exists). Container management: create if not exists. Query by session ID. Upsert messages. Cosmos DB REST auth: generate auth signature from master key

Success criteria:
* Create container succeeds
* Store messages and query by session ID
* Delete operations work
* REST auth signature generated correctly

Dependencies:
* Phase 4, Phase 1

### Step 10.3: Implement Redis storage provider

Files:
* cpp/src/Agents.Storage.Redis/include/agents/storage/redis/redis_chat_history_provider.h — `class RedisChatHistoryProvider : public ChatHistoryProvider`
* cpp/src/Agents.Storage.Redis/include/agents/storage/redis/redis_checkpoint_store.h — `class RedisCheckpointStore : public ICheckpointStore`
* cpp/src/Agents.Storage.Redis/src/ — Implementation using `hiredis` or `redis-plus-plus`. Async operations. Key pattern: `session:{id}:messages`, `checkpoint:{key}`

Success criteria:
* Store and retrieve chat history
* Checkpoint save/load works
* Async operations don't block event loop

Dependencies:
* Phase 4, Phase 9A

### Step 10.4: Implement Mem0 context provider and Purview DLP middleware

Files:
* cpp/src/Agents.Mem0/include/agents/mem0/mem0_context_provider.h — `class Mem0ContextProvider : public AIContextProvider`
* cpp/src/Agents.Mem0/src/mem0_context_provider.cpp — REST calls to Mem0 API for semantic memory search. Inject relevant memories as additional context in `invoking_async`
* cpp/src/Agents.Purview/include/agents/purview/purview_middleware.h — Agent middleware that calls Microsoft Graph API to check prompts/responses against Purview DLP policies
* cpp/src/Agents.Purview/src/purview_middleware.cpp — DLP policy evaluation: block or allow based on response. Graph API auth

Success criteria:
* Mem0: memory search returns relevant context, injected into agent
* Purview: DLP policy blocks prohibited content, allows safe content

Dependencies:
* Phase 4

### Step 10.5: Implement MCP tool serving integration

Files:
* cpp/src/Agents.MCP/include/agents/mcp/mcp_tool_server.h — `class MCPToolServer`
* cpp/src/Agents.MCP/src/mcp_tool_server.cpp — Integration with `modelcontextprotocol/cpp-sdk`. Serve agent tools via MCP protocol: tool discovery (list tools with schemas), tool invocation (call agent function and return result)

Success criteria:
* MCP tool discovery returns all registered tools
* MCP tool invocation calls correct agent function
* Results serialized in MCP format

Dependencies:
* Phase 3

### Step 10.6: Write storage and integration tests

Files:
* cpp/tests/Agents.Storage.Tests/in_memory_tests.cpp — Concurrent access, store/retrieve
* cpp/tests/Agents.Storage.Cosmos.Tests/ — Gated by `COSMOS_ENDPOINT` + `COSMOS_KEY` env vars
* cpp/tests/Agents.Storage.Redis.Tests/ — Gated by `REDIS_CONNECTION_STRING` env var
* cpp/tests/Agents.Mem0.Tests/ — Gated by `MEM0_API_KEY`
* cpp/tests/Agents.MCP.Tests/ — Tool discovery and invocation

Success criteria:
* In-memory tests always pass
* External storage tests pass when env vars set
* MCP protocol tests pass

Dependencies:
* Steps 10.1-10.5

## Implementation Phase 11: Durable Tasks

<!-- parallelizable: false -->

### Step 11.1: Generate gRPC stubs and implement DurableTaskClient

Files:
* cpp/src/Agents.DurableTask/proto/ — Copy or reference proto files from `microsoft/durabletask-protobuf`
* cpp/src/Agents.DurableTask/include/agents/durabletask/durable_task_client.h — `class DurableTaskClient`
* cpp/src/Agents.DurableTask/src/durable_task_client.cpp — gRPC client with entity operations: `signal_entity(entity_id, operation, input)`, `get_entity_state<T>(entity_id)`, `schedule_new_orchestration(name, input)`. Connection management, retry logic

Success criteria:
* gRPC stubs compile from proto files
* Client connects to Durable Task sidecar
* Entity signal and query operations work

Context references:
* .copilot-tracking/research/2026-02-20-cpp-port-implementation-plan.md (Lines 695-720) — Durable tasks spec
* schemas/durable-agent-entity-state.json — Entity state schema

Dependencies:
* Phase 4

### Step 11.2: Implement DurableAIAgent and DurableAIAgentProxy

Files:
* cpp/src/Agents.DurableTask/include/agents/durabletask/durable_ai_agent.h — `class DurableAIAgent : public AIAgent` — Entity-backed state. Entity ID = composite `AgentSessionId`. `run_async` signals entity, entity state machine processes the request
* cpp/src/Agents.DurableTask/include/agents/durabletask/durable_ai_agent_proxy.h — `class DurableAIAgentProxy` — For calling durable agents from outside orchestration context. Submits request and polls for completion

Success criteria:
* Agent state persists across invocations
* Entity operations route correctly
* Proxy submits and retrieves results

Dependencies:
* Step 11.1

### Step 11.3: Implement entity state schema

Files:
* cpp/src/Agents.DurableTask/include/agents/durabletask/entity_state.h — Entity state struct matching `schemas/durable-agent-entity-state.json`. glaze serialization. State transitions: idle → processing → waiting_for_input → completed

Success criteria:
* State serializes to JSON matching schema
* All state transitions valid

Context references:
* schemas/durable-agent-entity-state.json — Schema definition

Dependencies:
* Step 11.1

### Step 11.4: Implement standalone AgentDaemon

Files:
* cpp/src/Agents.Hosting.Daemon/include/agents/hosting/daemon/agent_daemon.h — `class AgentDaemon`
* cpp/src/Agents.Hosting.Daemon/src/agent_daemon.cpp — HTTP server that mirrors Azure Functions trigger patterns. Receives HTTP requests (matching Azure Functions HTTP trigger routes), dispatches to Durable Task sidecar via gRPC. Replaces Azure Functions hosting for C++. Configuration: durable task sidecar address, port, agent registrations

Success criteria:
* Daemon accepts HTTP requests
* Dispatches to Durable Task sidecar correctly
* Matches Azure Functions trigger URL patterns

Dependencies:
* Steps 11.1-11.3, Phase 8A

### Step 11.5: Write durable task tests

Files:
* cpp/tests/Agents.DurableTask.Tests/client_tests.cpp — Mock gRPC server, entity operations
* cpp/tests/Agents.DurableTask.Tests/agent_tests.cpp — DurableAIAgent state persistence
* cpp/tests/Agents.DurableTask.Tests/daemon_tests.cpp — HTTP → gRPC dispatch integration

Success criteria:
* Mock gRPC tests pass on all compilers
* Entity state round-trips through JSON matching schema
* Daemon integration test works with mock sidecar

Dependencies:
* Steps 11.1-11.4

## Implementation Phase 12: Developer UI and Samples

<!-- parallelizable: false -->

### Step 12.1: Implement Developer UI SPA embedding

Files:
* cpp/src/Agents.DevUI/include/agents/devui/devui_middleware.h — Middleware that serves static SPA files from embedded or directory-based location
* cpp/src/Agents.DevUI/src/devui_middleware.cpp — Static file serving, REST API for entity discovery (`GET /api/agents`), agent listing, conversation management. SPA routing: non-API routes → `index.html`

Success criteria:
* DevUI serves and displays agent information
* REST API returns agent metadata
* SPA routing works (client-side routes)

Dependencies:
* Phase 8A

### Step 12.2: Create 4 Getting Started samples

Files:
* cpp/samples/GettingStarted/Step1_Chat/ — Minimal: create chat client, send message, print response. `CMakeLists.txt`, `Program.cpp`, `README.md`
* cpp/samples/GettingStarted/Step2_Chat_Agent/ — Add agent wrapper: create agent, run with session. `CMakeLists.txt`, `Program.cpp`, `README.md`
* cpp/samples/GettingStarted/Step3_Chat_Agent_Tools/ — Add tools: define function tools, agent auto-invokes. `CMakeLists.txt`, `Program.cpp`, `README.md`
* cpp/samples/GettingStarted/Step4_Chat_Agent_OpenAI_Hosting/ — Add hosting: serve agent via OpenAI-compatible HTTP API. `CMakeLists.txt`, `Program.cpp`, `README.md`

Each follows the pattern:
* Single `Program.cpp` with all code
* Configuration via environment variables
* Clear comments explaining each step
* README with: what it does, prerequisites, how to run, expected output

Success criteria:
* All 4 samples compile and run on all platforms
* Output matches README examples

Context references:
* dotnet/samples/GettingStarted/ — .NET getting started samples to mirror

Dependencies:
* Phases 4-8

### Step 12.3: Create advanced samples

Files:
* cpp/samples/HostedAgents/ — Multi-agent hosting: A2A, AG-UI, OpenAI-compatible in single server. Agent-to-agent communication via A2A protocol
* cpp/samples/Durable/ — Long-running agent with checkpoint/resume via Durable Task
* cpp/samples/Declarative/ — YAML-defined workflows with Lua expressions

Success criteria:
* Each sample demonstrates its specific feature
* Samples run end-to-end

Dependencies:
* Phases 8-11

### Step 12.4: Write sample READMEs and index

Files:
* cpp/samples/*/README.md — Consistent format: ## What This Sample Does, ## Prerequisites, ## How to Run, ## Expected Output, ## Key Concepts
* cpp/samples/README.md — Index of all samples with descriptions and links

Success criteria:
* READMEs render correctly
* Instructions reproduce working runs

Dependencies:
* Steps 12.2-12.3

### Step 12.5: End-to-end validation

Run all samples, verify output matches expectations. Test on all 3 platforms.

Validation commands:
* `cmake --build --preset release --target all` — Build all samples
* Run each sample with configured environment variables
* Compare output to expected results in READMEs

Success criteria:
* All samples build and run on Windows, Linux, macOS
* Output matches documentation

Dependencies:
* Steps 12.1-12.4

## Implementation Phase 13: Final Validation

<!-- parallelizable: false -->

### Step 13.1: Run full project validation

Execute all validation commands for the project:
* `cmake --build --preset release` — Full build of all 27 libraries + tests + samples
* `ctest --preset release` — All unit tests
* `clang-tidy --config-file=cpp/.clang-tidy` — Static analysis on all source files
* `clang-format --dry-run --Werror` — Format check on all source files
* `cmake --build --preset asan && ctest --preset asan` — Address Sanitizer pass
* `cmake --build --preset tsan && ctest --preset tsan` — Thread Sanitizer pass
* `cmake --build --preset ubsan && ctest --preset ubsan` — Undefined Behavior Sanitizer pass

### Step 13.2: Fix minor validation issues

Iterate on lint errors, build warnings, and test failures. Apply fixes directly when corrections are straightforward and isolated.

### Step 13.3: Cross-language interop validation

Run all interop test suites:
* C++ OpenAI-compatible server ↔ .NET OpenAI client
* C++ OpenAI-compatible server ↔ Python OpenAI client
* C++ A2A server ↔ .NET A2A client
* C++ A2A client ↔ .NET A2A server
* C++ AG-UI server ↔ CopilotKit TypeScript client

### Step 13.4: Report blocking issues

When validation failures require changes beyond minor fixes:
* Document the issues and affected files
* Provide the user with next steps
* Recommend additional research and planning rather than inline fixes
* Avoid large-scale refactoring within this phase

## Dependencies

* CMake 3.25+
* vcpkg (manifest mode)
* Boost 1.84+ (Asio, Beast)
* glaze 4.x
* nlohmann-json 3.x
* spdlog 1.x + fmt
* opentelemetry-cpp 1.x
* OpenSSL 3.x
* Google Test + gmock 1.15+
* yaml-cpp 0.8+ (Phase 9)
* sol2 3.x + Lua 5.4 (Phase 9)
* gRPC 1.60+ + protobuf (Phase 11)
* hiredis / redis-plus-plus (Phase 10)
* azure-identity-cpp (Phases 5, 6)

## Success Criteria

* All 27 C++ library targets build on MSVC, GCC 11+, and Clang 14+ across Windows, Linux, and macOS
* Complete unit test coverage for all public API methods
* All hosting protocols interoperate with .NET and Python implementations
* Zero sanitizer warnings under ASAN, TSAN, UBSAN
* All samples compile and run on all 3 platforms
* CI pipeline green for all valid matrix entries
