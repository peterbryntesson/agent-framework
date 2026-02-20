# .NET Core Abstractions Research

> **Generated**: 2026-02-20
> **Scope**: `Microsoft.Agents.AI.Abstractions`, `Microsoft.Agents.AI`, `Microsoft.Agents.AI.Hosting`, `Shared/`
> **Purpose**: Comprehensive analysis of all interfaces, abstract classes, enums, delegates, and design patterns in the core agent framework.

---

## Table of Contents

1. [Microsoft.Agents.AI.Abstractions](#1-microsoftagentsaiabstractions)
2. [Microsoft.Agents.AI](#2-microsoftagentsai)
3. [Microsoft.Agents.AI.Hosting](#3-microsoftagentsaihosting)
4. [Shared/](#4-shared)
5. [Inheritance Hierarchies](#5-inheritance-hierarchies)
6. [Design Patterns](#6-design-patterns)
7. [External Dependencies](#7-external-dependencies)

---

## 1. Microsoft.Agents.AI.Abstractions

**Project Path**: `dotnet/src/Microsoft.Agents.AI.Abstractions/`
**Namespace**: `Microsoft.Agents.AI`

### 1.1 Abstract Classes

#### `AIAgent` — [dotnet/src/Microsoft.Agents.AI.Abstractions/AIAgent.cs](dotnet/src/Microsoft.Agents.AI.Abstractions/AIAgent.cs)

The foundational abstract class for the entire agent framework. All agents derive from this.

**Properties:**

| Property | Type | Modifiers | Description |
|---|---|---|---|
| `Id` | `string` | public, get | Auto-generated GUID; delegates to `IdCore` if non-null |
| `IdCore` | `string?` | protected virtual, get | Override point for custom IDs |
| `Name` | `string?` | public virtual, get | Agent display name |
| `Description` | `string?` | public virtual, get | Agent description |

**Methods:**

| Method | Return Type | Modifiers | Signature |
|---|---|---|---|
| `GetService` | `object?` | public virtual | `GetService(Type serviceType, object? serviceKey = null)` |
| `GetService<T>` | `T?` | public | `GetService<T>(object? serviceKey = null)` |
| `GetNewSessionAsync` | `Task<AgentSession>` | public abstract | `GetNewSessionAsync(CancellationToken)` |
| `DeserializeSessionAsync` | `Task<AgentSession>` | public abstract | `DeserializeSessionAsync(JsonElement serializedSession, JsonSerializerOptions?, CancellationToken)` |
| `RunAsync` (4 overloads) | `Task<AgentResponse>` | public | `RunAsync()`, `RunAsync(string)`, `RunAsync(ChatMessage, ...)`, `RunAsync(IEnumerable<ChatMessage>, ...)` |
| `RunStreamingAsync` (4 overloads) | `IAsyncEnumerable<AgentResponseUpdate>` | public | Mirrors `RunAsync` overloads |
| `RunCoreAsync` | `Task<AgentResponse>` | protected abstract | `RunCoreAsync(IEnumerable<ChatMessage>, AgentSession?, AgentRunOptions?, CancellationToken)` |
| `RunCoreStreamingAsync` | `IAsyncEnumerable<AgentResponseUpdate>` | protected abstract | `RunCoreStreamingAsync(IEnumerable<ChatMessage>, AgentSession?, AgentRunOptions?, CancellationToken)` |

**Key Details:**

- Lines 1-392. All `RunAsync`/`RunStreamingAsync` public overloads funnel to `RunCoreAsync`/`RunCoreStreamingAsync`.
- `Id` is lazily initialized from `IdCore ?? Guid.NewGuid().ToString("N")`.
- `GetService` pattern enables service locator from any agent; defaults return `null`.

---

#### `AgentSession` — [dotnet/src/Microsoft.Agents.AI.Abstractions/AgentSession.cs](dotnet/src/Microsoft.Agents.AI.Abstractions/AgentSession.cs)

Base class for all conversation/session state.

**Methods:**

| Method | Return Type | Modifiers | Signature |
|---|---|---|---|
| `Serialize` | `JsonElement` | public virtual | `Serialize(JsonSerializerOptions? options = null)` |
| `GetService` | `object?` | public virtual | `GetService(Type serviceType, object? serviceKey = null)` |
| `GetService<T>` | `T?` | public | `GetService<T>(object? serviceKey = null)` |

---

#### `AgentResponse<T>` — [dotnet/src/Microsoft.Agents.AI.Abstractions/AgentResponse{T}.cs](dotnet/src/Microsoft.Agents.AI.Abstractions/AgentResponse%7BT%7D.cs)

Generic typed response base.

**Properties:**

| Property | Type | Modifiers |
|---|---|---|
| `Result` | `T` | abstract, get |

**Inherits**: `AgentResponse`

---

#### `DelegatingAIAgent` — [dotnet/src/Microsoft.Agents.AI.Abstractions/DelegatingAIAgent.cs](dotnet/src/Microsoft.Agents.AI.Abstractions/DelegatingAIAgent.cs)

Decorator pattern base class. Delegates all operations to an inner agent.

**Properties:**

| Property | Type | Modifiers |
|---|---|---|
| `InnerAgent` | `AIAgent` | protected, get |

**Overrides**: `IdCore`, `Name`, `Description`, `GetService`, `GetNewSessionAsync`, `DeserializeSessionAsync`, `RunCoreAsync`, `RunCoreStreamingAsync` — all delegate to `InnerAgent`.

---

#### `AIContextProvider` — [dotnet/src/Microsoft.Agents.AI.Abstractions/AIContextProvider.cs](dotnet/src/Microsoft.Agents.AI.Abstractions/AIContextProvider.cs)

Two-phase lifecycle context provider. Supplies dynamic context (instructions, messages, tools) to agents.

**Methods:**

| Method | Return Type | Modifiers | Signature |
|---|---|---|---|
| `InvokingAsync` | `ValueTask<AIContext>` | public abstract | `InvokingAsync(InvokingContext context, CancellationToken)` |
| `InvokedAsync` | `ValueTask` | public virtual | `InvokedAsync(InvokedContext context, CancellationToken)` |
| `Serialize` | `JsonElement?` | public virtual | `Serialize(JsonSerializerOptions?)` |
| `GetService` | `object?` | public virtual | `GetService(Type, object?)` |

**Nested Types:**

- `sealed class InvokingContext` — Properties: `RequestMessages` (`IEnumerable<ChatMessage>`)
- `sealed class InvokedContext` — Properties: `RequestMessages`, `AIContextProviderMessages` (`IList<ChatMessage>?`), `ResponseMessages` (`IList<ChatMessage>`), `InvokeException` (`Exception?`)

---

#### `ChatHistoryProvider` — [dotnet/src/Microsoft.Agents.AI.Abstractions/ChatHistoryProvider.cs](dotnet/src/Microsoft.Agents.AI.Abstractions/ChatHistoryProvider.cs)

Abstract contract for chat history management.

**Methods:**

| Method | Return Type | Modifiers | Signature |
|---|---|---|---|
| `InvokingAsync` | `ValueTask<IEnumerable<ChatMessage>>` | public abstract | `InvokingAsync(InvokingContext, CancellationToken)` |
| `InvokedAsync` | `ValueTask` | public abstract | `InvokedAsync(InvokedContext, CancellationToken)` |
| `Serialize` | `JsonElement?` | public abstract | `Serialize(JsonSerializerOptions?)` |
| `GetService` | `object?` | public virtual | `GetService(Type, object?)` |

**Nested Types:**

- `sealed class InvokingContext` — Properties: `RequestMessages` (`IEnumerable<ChatMessage>`)
- `sealed class InvokedContext` — Properties: `RequestMessages`, `ChatHistoryProviderMessages` (`IEnumerable<ChatMessage>`), `AIContextProviderMessages` (`IList<ChatMessage>?`), `ResponseMessages` (`IList<ChatMessage>`), `InvokeException` (`Exception?`)

---

#### `InMemoryAgentSession` — [dotnet/src/Microsoft.Agents.AI.Abstractions/InMemoryAgentSession.cs](dotnet/src/Microsoft.Agents.AI.Abstractions/InMemoryAgentSession.cs)

Base class for sessions that store chat history in-memory.

**Properties:**

| Property | Type | Modifiers |
|---|---|---|
| `ChatHistoryProvider` | `InMemoryChatHistoryProvider` | public, get |

**Constructors:** Default (creates new provider), copy (from existing provider), deserialization (from `JsonElement` state).

**Nested Types:**

- `sealed class InMemoryAgentSessionState` — Properties: `ChatHistoryProviderState` (`InMemoryChatHistoryProvider.State?`)

**Inherits**: `AgentSession`

---

#### `ServiceIdAgentSession` — [dotnet/src/Microsoft.Agents.AI.Abstractions/ServiceIdAgentSession.cs](dotnet/src/Microsoft.Agents.AI.Abstractions/ServiceIdAgentSession.cs)

Base class for sessions backed by remote service (server-side history).

**Properties:**

| Property | Type | Modifiers |
|---|---|---|
| `ServiceSessionId` | `string?` | protected, get/set |

**Nested Types:**

- `sealed class ServiceIdAgentSessionState` — Properties: `ServiceSessionId` (`string?`)

**Inherits**: `AgentSession`

---

### 1.2 Concrete/Sealed Classes

#### `AgentResponse` — [dotnet/src/Microsoft.Agents.AI.Abstractions/AgentResponse.cs](dotnet/src/Microsoft.Agents.AI.Abstractions/AgentResponse.cs)

Response container returned from non-streaming agent runs.

**Properties:**

| Property | Type | Description |
|---|---|---|
| `Messages` | `IList<ChatMessage>` | Response messages |
| `Text` | `string` | Concatenated text from all messages |
| `UserInputRequests` | `IEnumerable<UserInputRequestContent>` | User input requests in response |
| `AgentId` | `string?` | Originating agent ID |
| `ResponseId` | `string?` | Response identifier |
| `ContinuationToken` | `ResponseContinuationToken?` | For resumable operations |
| `CreatedAt` | `DateTimeOffset?` | Creation timestamp |
| `Usage` | `UsageDetails?` | Token usage details |
| `RawRepresentation` | `object?` | Provider-specific raw response |
| `AdditionalProperties` | `AdditionalPropertiesDictionary?` | Extension data |

**Constructors:** Default, `(ChatMessage)`, `(ChatResponse)`, `(IList<ChatMessage>)`

**Methods:** `ToAgentResponseUpdates()`, `Deserialize<T>()`, `TryDeserialize<T>()`

**Private Enum:**

- `FailureReason { ResultDidNotContainJson, DeserializationProducedNull }`

---

#### `AgentResponseUpdate` — [dotnet/src/Microsoft.Agents.AI.Abstractions/AgentResponseUpdate.cs](dotnet/src/Microsoft.Agents.AI.Abstractions/AgentResponseUpdate.cs)

Streaming response chunk from agents.

**Properties:**

| Property | Type |
|---|---|
| `AuthorName` | `string?` |
| `Role` | `ChatRole?` |
| `Text` | `string` |
| `UserInputRequests` | `IEnumerable<UserInputRequestContent>` |
| `Contents` | `IList<AIContent>` |
| `RawRepresentation` | `object?` |
| `AdditionalProperties` | `AdditionalPropertiesDictionary?` |
| `AgentId` | `string?` |
| `ResponseId` | `string?` |
| `MessageId` | `string?` |
| `CreatedAt` | `DateTimeOffset?` |
| `ContinuationToken` | `ResponseContinuationToken?` |

---

#### `AgentRunOptions` — [dotnet/src/Microsoft.Agents.AI.Abstractions/AgentRunOptions.cs](dotnet/src/Microsoft.Agents.AI.Abstractions/AgentRunOptions.cs)

Options passed to `RunAsync`/`RunStreamingAsync`.

**Properties:**

| Property | Type | Description |
|---|---|---|
| `ContinuationToken` | `ResponseContinuationToken?` | For resuming operations |
| `AllowBackgroundResponses` | `bool?` | Whether to allow background (non-final) responses |
| `AdditionalProperties` | `AdditionalPropertiesDictionary?` | Extension data |

---

#### `AIContext` — [dotnet/src/Microsoft.Agents.AI.Abstractions/AIContext.cs](dotnet/src/Microsoft.Agents.AI.Abstractions/AIContext.cs)

Container for dynamic context injected by `AIContextProvider`.

**Properties:**

| Property | Type |
|---|---|
| `Instructions` | `string?` |
| `Messages` | `IList<ChatMessage>?` |
| `Tools` | `IList<AITool>?` |

---

#### `AIAgentMetadata` — [dotnet/src/Microsoft.Agents.AI.Abstractions/AIAgentMetadata.cs](dotnet/src/Microsoft.Agents.AI.Abstractions/AIAgentMetadata.cs)

Metadata attached to agents.

**Properties:**

| Property | Type |
|---|---|
| `ProviderName` | `string?` |

---

#### `InMemoryChatHistoryProvider` — [dotnet/src/Microsoft.Agents.AI.Abstractions/InMemoryChatHistoryProvider.cs](dotnet/src/Microsoft.Agents.AI.Abstractions/InMemoryChatHistoryProvider.cs)

In-memory chat history storage implementing `IList<ChatMessage>`.

**Implements:** `ChatHistoryProvider`, `IList<ChatMessage>`, `IReadOnlyList<ChatMessage>`

**Properties:**

| Property | Type | Description |
|---|---|---|
| `ChatReducer` | `IChatReducer?` | Optional message reducer |
| `ReducerTriggerEvent` | `ChatReducerTriggerEvent` | When to trigger reduction |

**Nested Types:**

- `enum ChatReducerTriggerEvent { AfterMessageAdded, BeforeMessagesRetrieval }`
- `sealed class State` — Properties: `Messages` (`List<ChatMessage>?`)

---

#### `ChatHistoryProviderMessageFilter` — [dotnet/src/Microsoft.Agents.AI.Abstractions/ChatHistoryProviderMessageFilter.cs](dotnet/src/Microsoft.Agents.AI.Abstractions/ChatHistoryProviderMessageFilter.cs)

Decorator that filters messages from chat history providers.

**Inherits**: `ChatHistoryProvider`

---

### 1.3 Interfaces

> No standalone interfaces are defined in the Abstractions project. The `IChatReducer` referenced by `InMemoryChatHistoryProvider` comes from `Microsoft.Agents.AI` or another project. All contracts are expressed as abstract classes.

---

### 1.4 Extension Classes

#### `AdditionalPropertiesExtensions` — [dotnet/src/Microsoft.Agents.AI.Abstractions/AdditionalPropertiesExtensions.cs](dotnet/src/Microsoft.Agents.AI.Abstractions/AdditionalPropertiesExtensions.cs)

Static extension methods on `AdditionalPropertiesDictionary`:

- `Add<T>(key, value)` / `TryAdd<T>(key, value)` / `TryGetValue<T>(key, out value)` / `Contains<T>(key)` / `Remove<T>(key)`

#### `AgentResponseExtensions` — [dotnet/src/Microsoft.Agents.AI.Abstractions/AgentResponseExtensions.cs](dotnet/src/Microsoft.Agents.AI.Abstractions/AgentResponseExtensions.cs)

Conversions between framework types and `Microsoft.Extensions.AI` types:

- `AsChatResponse(AgentResponse)` → `ChatResponse`
- `AsChatResponseUpdate(AgentResponseUpdate)` → `ChatResponseUpdate`
- `AsChatResponseUpdatesAsync(IAsyncEnumerable<AgentResponseUpdate>)` → `IAsyncEnumerable<ChatResponseUpdate>`
- `ToAgentResponse(ChatResponse)` → `AgentResponse`
- `ToAgentResponseAsync(IAsyncEnumerable<ChatResponseUpdate>)` → `IAsyncEnumerable<AgentResponseUpdate>`

#### `ChatHistoryProviderExtensions` — [dotnet/src/Microsoft.Agents.AI.Abstractions/ChatHistoryProviderExtensions.cs](dotnet/src/Microsoft.Agents.AI.Abstractions/ChatHistoryProviderExtensions.cs)

- `WithMessageFilters(ChatHistoryProvider, params Func<IEnumerable<ChatMessage>, IEnumerable<ChatMessage>>[])` → `ChatHistoryProvider`
- `WithAIContextProviderMessageRemoval(ChatHistoryProvider)` → `ChatHistoryProvider`

#### `AIContentExtensions` — [dotnet/src/Microsoft.Agents.AI.Abstractions/AIContentExtensions.cs](dotnet/src/Microsoft.Agents.AI.Abstractions/AIContentExtensions.cs) (internal)

- `ConcatText(IEnumerable<AIContent>)` → `string`
- `ConcatText(IList<ChatMessage>)` → `string`

#### `AgentAbstractionsJsonUtilities` — [dotnet/src/Microsoft.Agents.AI.Abstractions/AgentAbstractionsJsonUtilities.cs](dotnet/src/Microsoft.Agents.AI.Abstractions/AgentAbstractionsJsonUtilities.cs)

- Static `DefaultOptions` property (pre-configured `JsonSerializerOptions`)
- Source-generated `JsonSerializerContext` for framework types

---

## 2. Microsoft.Agents.AI

**Project Path**: `dotnet/src/Microsoft.Agents.AI/`
**Namespace**: `Microsoft.Agents.AI`

### 2.1 Concrete Agent Implementations

#### `ChatClientAgent` — [dotnet/src/Microsoft.Agents.AI/ChatClient/ChatClientAgent.cs](dotnet/src/Microsoft.Agents.AI/ChatClient/ChatClientAgent.cs)

The primary concrete agent implementation. Wraps an `IChatClient`.

**Sealed partial class, 910 lines.**

**Properties:**

| Property | Type | Modifiers |
|---|---|---|
| `ChatClient` | `IChatClient` | internal, get |
| `Instructions` | `string?` | overrides `AIAgent` via IdCore pattern |
| `ChatOptions` | `ChatOptions?` | internal |

**Constructors:**

```csharp
public ChatClientAgent(
    IChatClient chatClient,
    ChatClientAgentOptions? options = null,
    ILoggerFactory? loggerFactory = null,
    IServiceProvider? serviceProvider = null)

public ChatClientAgent(
    IChatClient chatClient,
    string? instructions = null,
    string? name = null,
    string? description = null,
    IList<AITool>? tools = null,
    ILoggerFactory? loggerFactory = null,
    IServiceProvider? serviceProvider = null)
```

**Key Behavior:**

- `RunCoreAsync`: Orchestrates invocation lifecycle:
  1. Creates session if none provided
  2. Calls `ChatHistoryProvider.InvokingAsync` to get history
  3. Calls `AIContextProvider.InvokingAsync` to get dynamic context
  4. Merges ChatOptions (agent defaults + runtime overrides)
  5. Builds message list: history + context messages + request messages
  6. Calls `IChatClient.GetResponseAsync`
  7. Calls `ChatHistoryProvider.InvokedAsync` and `AIContextProvider.InvokedAsync`
  8. Returns `AgentResponse`
- `RunCoreStreamingAsync`: Same lifecycle but uses `IChatClient.GetStreamingResponseAsync`
- Supports both server-side (ConversationId) and client-side (ChatHistoryProvider) session management

**Session Factory Methods:**

- `GetNewSessionAsync()` — uses factory from options or creates default
- `GetNewSessionAsync(string conversationId)` — server-side session
- `GetNewSessionAsync(ChatHistoryProvider)` — client-side session

---

#### `ChatClientAgentSession` — [dotnet/src/Microsoft.Agents.AI/ChatClient/ChatClientAgentSession.cs](dotnet/src/Microsoft.Agents.AI/ChatClient/ChatClientAgentSession.cs)

Session type for `ChatClientAgent`. Supports two mutually exclusive modes.

**Sealed class.**

**Properties:**

| Property | Type | Description |
|---|---|---|
| `ConversationId` | `string?` | Server-side session ID (mutual exclusion with ChatHistoryProvider) |
| `ChatHistoryProvider` | `ChatHistoryProvider?` | Client-side history storage |
| `AIContextProvider` | `AIContextProvider?` | Optional context provider |

**Nested Types:**

- `sealed class SessionState` — Properties: `ConversationId`, `ChatHistoryProviderState` (`JsonElement?`), `AIContextProviderState` (`JsonElement?`)

---

#### `ChatClientAgentOptions` — [dotnet/src/Microsoft.Agents.AI/ChatClient/ChatClientAgentOptions.cs](dotnet/src/Microsoft.Agents.AI/ChatClient/ChatClientAgentOptions.cs)

Configuration for `ChatClientAgent`.

**Sealed class.**

**Properties:**

| Property | Type |
|---|---|
| `Id` | `string?` |
| `Name` | `string?` |
| `Description` | `string?` |
| `ChatOptions` | `ChatOptions?` |
| `ChatHistoryProviderFactory` | `Func<ChatHistoryProviderFactoryContext, ChatHistoryProvider>?` |
| `AIContextProviderFactory` | `Func<AIContextProviderFactoryContext, AIContextProvider?>?` |
| `UseProvidedChatClientAsIs` | `bool` |

**Nested Types (delegates via factory contexts):**

- `sealed class AIContextProviderFactoryContext` — `SerializedState`, `JsonSerializerOptions`
- `sealed class ChatHistoryProviderFactoryContext` — `SerializedState`, `JsonSerializerOptions`

---

#### `ChatClientAgentRunOptions` — [dotnet/src/Microsoft.Agents.AI/ChatClient/ChatClientAgentRunOptions.cs](dotnet/src/Microsoft.Agents.AI/ChatClient/ChatClientAgentRunOptions.cs)

Extended run options for `ChatClientAgent`.

**Sealed class, inherits `AgentRunOptions`.**

**Properties:**

| Property | Type |
|---|---|
| `ChatOptions` | `ChatOptions?` |
| `ChatClientFactory` | `Func<IChatClient, IChatClient>?` |

---

### 2.2 Builder

#### `AIAgentBuilder` — [dotnet/src/Microsoft.Agents.AI/AIAgentBuilder.cs](dotnet/src/Microsoft.Agents.AI/AIAgentBuilder.cs)

Builder for composing agent middleware pipelines.

**Sealed class.**

**Methods:**

| Method | Return Type | Signature |
|---|---|---|
| `Build` | `AIAgent` | `Build(IServiceProvider? serviceProvider = null)` |
| `Use` | `AIAgentBuilder` | `Use(Func<AIAgent, AIAgent> agentFactory)` |
| `Use` | `AIAgentBuilder` | `Use(Func<AIAgent, IServiceProvider, AIAgent> agentFactory)` |
| `Use` (shared) | `AIAgentBuilder` | `Use(Func<IEnumerable<ChatMessage>, AgentSession?, AgentRunOptions?, AIAgent, CancellationToken, object> sharedFunc)` |
| `Use` (split) | `AIAgentBuilder` | `Use(runFunc, runStreamingFunc)` — separate handlers for run/streaming |

**Key Behavior:**

- Stores middleware factories in a list
- `Build()` applies factories in reverse order (first added = outermost)
- Uses `AnonymousDelegatingAIAgent` internally for delegate-based middleware

---

#### `AnonymousDelegatingAIAgent` — [dotnet/src/Microsoft.Agents.AI/AnonymousDelegatingAIAgent.cs](dotnet/src/Microsoft.Agents.AI/AnonymousDelegatingAIAgent.cs) (internal)

Internal sealed class implementing delegate-based middleware for the builder.

**Inherits**: `DelegatingAIAgent`

---

### 2.3 Decorator/Middleware Agents

#### `LoggingAgent` — [dotnet/src/Microsoft.Agents.AI/LoggingAgent.cs](dotnet/src/Microsoft.Agents.AI/LoggingAgent.cs)

Logs agent operations to `ILogger`.

**Sealed partial class, inherits `DelegatingAIAgent`.**

- Supports `Trace` level for sensitive data logging
- Builder extension: `UseLogging(ILoggerFactory?)`

---

#### `OpenTelemetryAgent` — [dotnet/src/Microsoft.Agents.AI/OpenTelemetryAgent.cs](dotnet/src/Microsoft.Agents.AI/OpenTelemetryAgent.cs)

OpenTelemetry instrumentation wrapper.

**Sealed class, inherits `DelegatingAIAgent`, implements `IDisposable`.**

**Properties:**

| Property | Type |
|---|---|
| `EnableSensitiveData` | `bool` |

- Uses internal `ForwardingChatClient` that delegates through `OpenTelemetryChatClient`
- Builder extension: `UseOpenTelemetry(ILoggerFactory?, enableSensitiveData?)`

---

#### `FunctionInvocationDelegatingAgent` — [dotnet/src/Microsoft.Agents.AI/FunctionInvocationDelegatingAgent.cs](dotnet/src/Microsoft.Agents.AI/FunctionInvocationDelegatingAgent.cs) (internal)

Wraps agent tools with middleware functions.

**Internal sealed class, inherits `DelegatingAIAgent`.**

**Nested Types:**

- `sealed class MiddlewareEnabledFunction : DelegatingAIFunction` — Wraps each tool's invocation with middleware

---

### 2.4 Memory/RAG Providers

#### `TextSearchProvider` — [dotnet/src/Microsoft.Agents.AI/Memory/TextSearchProvider.cs](dotnet/src/Microsoft.Agents.AI/Memory/TextSearchProvider.cs)

RAG context provider using text search.

**Sealed class, inherits `AIContextProvider`.**

**Two modes:**

- `BeforeAIInvoke` — automatically injects search results as context
- `OnDemandFunctionCalling` — exposes search as tool for the agent

**Nested Types:**

- `sealed class TextSearchResult` — Properties: `Content`, `Link`
- `sealed class TextSearchProviderState` — Serializable state

---

#### `TextSearchProviderOptions` — [dotnet/src/Microsoft.Agents.AI/Memory/TextSearchProviderOptions.cs](dotnet/src/Microsoft.Agents.AI/Memory/TextSearchProviderOptions.cs)

**Nested Enum:**

- `TextSearchBehavior { BeforeAIInvoke, OnDemandFunctionCalling }`

---

#### `ChatHistoryMemoryProvider` — [dotnet/src/Microsoft.Agents.AI/Memory/ChatHistoryMemoryProvider.cs](dotnet/src/Microsoft.Agents.AI/Memory/ChatHistoryMemoryProvider.cs)

Semantic memory over chat history using vector store.

**Sealed class, inherits `AIContextProvider`, implements `IDisposable`.**

- Uses `VectorStore` and `VectorStoreCollection` for embedding-based retrieval
- Two modes: `BeforeAIInvoke`, `OnDemandFunctionCalling`

---

#### `ChatHistoryMemoryProviderOptions` — [dotnet/src/Microsoft.Agents.AI/Memory/ChatHistoryMemoryProviderOptions.cs](dotnet/src/Microsoft.Agents.AI/Memory/ChatHistoryMemoryProviderOptions.cs)

**Nested Enum:**

- `SearchBehavior { BeforeAIInvoke, OnDemandFunctionCalling }`

---

#### `ChatHistoryMemoryProviderScope` — [dotnet/src/Microsoft.Agents.AI/Memory/ChatHistoryMemoryProviderScope.cs](dotnet/src/Microsoft.Agents.AI/Memory/ChatHistoryMemoryProviderScope.cs)

Scoping for memory search.

**Sealed class.**

**Properties:**

| Property | Type |
|---|---|
| `ApplicationId` | `string?` |
| `AgentId` | `string?` |
| `SessionId` | `string?` |
| `UserId` | `string?` |

---

### 2.5 Extension Methods

#### `AIAgentExtensions` — [dotnet/src/Microsoft.Agents.AI/AgentExtensions.cs](dotnet/src/Microsoft.Agents.AI/AgentExtensions.cs)

- `AsBuilder(this AIAgent)` → `AIAgentBuilder` — wraps existing agent in builder
- `AsAIFunction(this AIAgent, options?, session?)` → `AIFunction` — converts agent into callable function (for multi-agent composition)

#### `ChatClientBuilderExtensions` — [dotnet/src/Microsoft.Agents.AI/ChatClient/ChatClientBuilderExtensions.cs](dotnet/src/Microsoft.Agents.AI/ChatClient/ChatClientBuilderExtensions.cs)

- `UseFunctionInvocationMiddleware(...)` — adds function invocation middleware to chat client builder

#### `ChatClientExtensions` — [dotnet/src/Microsoft.Agents.AI/ChatClient/ChatClientExtensions.cs](dotnet/src/Microsoft.Agents.AI/ChatClient/ChatClientExtensions.cs)

- Extensions on `IChatClient` for direct agent operations

#### Builder Extensions (Logging, OpenTelemetry, FunctionInvocation):

- `LoggingAgentBuilderExtensions` — `UseLogging(AIAgentBuilder, ILoggerFactory?)`
- `OpenTelemetryAgentBuilderExtensions` — `UseOpenTelemetry(AIAgentBuilder, ILoggerFactory?, enableSensitiveData?)`
- `FunctionInvocationDelegatingAgentBuilderExtensions` — `UseFunctionInvocationMiddleware(AIAgentBuilder, ...)`

---

## 3. Microsoft.Agents.AI.Hosting

**Project Path**: `dotnet/src/Microsoft.Agents.AI.Hosting/`
**Namespace**: `Microsoft.Agents.AI.Hosting`

### 3.1 Interfaces

#### `IHostedAgentBuilder` — [dotnet/src/Microsoft.Agents.AI.Hosting/IHostedAgentBuilder.cs](dotnet/src/Microsoft.Agents.AI.Hosting/IHostedAgentBuilder.cs)

DI builder interface for registering agents.

**Properties:**

| Property | Type |
|---|---|
| `Name` | `string` |
| `ServiceCollection` | `IServiceCollection` |

---

#### `IHostedWorkflowBuilder` — [dotnet/src/Microsoft.Agents.AI.Hosting/IHostedWorkflowBuilder.cs](dotnet/src/Microsoft.Agents.AI.Hosting/IHostedWorkflowBuilder.cs)

DI builder interface for registering workflows.

**Properties:**

| Property | Type |
|---|---|
| `Name` | `string` |
| `HostApplicationBuilder` | `IHostApplicationBuilder` |

---

### 3.2 Abstract Classes

#### `AgentSessionStore` — [dotnet/src/Microsoft.Agents.AI.Hosting/AgentSessionStore.cs](dotnet/src/Microsoft.Agents.AI.Hosting/AgentSessionStore.cs)

Abstract contract for session persistence.

**Methods:**

| Method | Return Type | Modifiers | Signature |
|---|---|---|---|
| `SaveSessionAsync` | `ValueTask` | public abstract | `SaveSessionAsync(AIAgent, string conversationId, AgentSession, CancellationToken)` |
| `GetSessionAsync` | `ValueTask<AgentSession>` | public abstract | `GetSessionAsync(AIAgent, string conversationId, CancellationToken)` |

---

#### `WorkflowCatalog` — [dotnet/src/Microsoft.Agents.AI.Hosting/WorkflowCatalog.cs](dotnet/src/Microsoft.Agents.AI.Hosting/WorkflowCatalog.cs)

Abstract catalog of registered workflows.

**Methods:**

| Method | Return Type | Modifiers |
|---|---|---|
| `GetWorkflowsAsync` | `IAsyncEnumerable<Workflow>` | public abstract |

---

### 3.3 Concrete/Sealed Classes

#### `AIHostAgent` — [dotnet/src/Microsoft.Agents.AI.Hosting/AIHostAgent.cs](dotnet/src/Microsoft.Agents.AI.Hosting/AIHostAgent.cs)

Hosting wrapper that adds session persistence via `AgentSessionStore`.

**Inherits**: `DelegatingAIAgent`

**Methods:**

| Method | Return Type | Signature |
|---|---|---|
| `GetOrCreateSessionAsync` | `Task<AgentSession>` | `GetOrCreateSessionAsync(string conversationId, CancellationToken)` |
| `SaveSessionAsync` | `Task` | `SaveSessionAsync(string conversationId, AgentSession, CancellationToken)` |

**Overrides**: `RunCoreAsync`, `RunCoreStreamingAsync` — loads session before run, saves after.

---

#### `NoopAgentSessionStore` — [dotnet/src/Microsoft.Agents.AI.Hosting/NoopAgentSessionStore.cs](dotnet/src/Microsoft.Agents.AI.Hosting/NoopAgentSessionStore.cs)

No-op session store. Always creates new sessions; never persists.

**Sealed class, inherits `AgentSessionStore`.**

---

#### `InMemoryAgentSessionStore` — [dotnet/src/Microsoft.Agents.AI.Hosting/Local/InMemoryAgentSessionStore.cs](dotnet/src/Microsoft.Agents.AI.Hosting/Local/InMemoryAgentSessionStore.cs)

In-memory session store using `ConcurrentDictionary`.

**Sealed class, inherits `AgentSessionStore`.**

**Key behavior:**

- Stores serialized `JsonElement` values keyed by `"{agentId}:{conversationId}"`
- `GetSessionAsync`: Deserializes existing or creates new session
- `SaveSessionAsync`: Serializes and stores

---

#### `HostedAgentBuilder` — [dotnet/src/Microsoft.Agents.AI.Hosting/HostedAgentBuilder.cs](dotnet/src/Microsoft.Agents.AI.Hosting/HostedAgentBuilder.cs)

Internal sealed implementation of `IHostedAgentBuilder`.

---

#### `HostedWorkflowBuilder` — [dotnet/src/Microsoft.Agents.AI.Hosting/HostedWorkflowBuilder.cs](dotnet/src/Microsoft.Agents.AI.Hosting/HostedWorkflowBuilder.cs)

Internal sealed implementation of `IHostedWorkflowBuilder`.

---

### 3.4 Extension Methods

#### `AgentHostingServiceCollectionExtensions` — [dotnet/src/Microsoft.Agents.AI.Hosting/AgentHostingServiceCollectionExtensions.cs](dotnet/src/Microsoft.Agents.AI.Hosting/AgentHostingServiceCollectionExtensions.cs)

DI registration for agents on `IServiceCollection`:

- `AddAIAgent(IServiceCollection, string name, Func<IServiceProvider, AIAgent> factory)` — base overload
- `AddAIAgent(IServiceCollection, string name, Func<IServiceProvider, AIAgent> factory, Action<IHostedAgentBuilder> configure)` — with builder configuration
- `AddAIAgent<TAgent>(IServiceCollection, string name)` — generic registration
- `AddAIAgent<TAgent>(IServiceCollection, string name, Action<IHostedAgentBuilder> configure)` — generic with configuration
- `AddAIAgent(IServiceCollection, string name, Func<IServiceProvider, AIAgentBuilder> factory, Action<IHostedAgentBuilder>? configure)` — builder-based

**Key Behavior:**

- Registers agents as **keyed singletons** using `ServiceDescriptor.KeyedSingleton`
- Wraps agents with `AIHostAgent` for session management
- Registers `AgentSessionStore` as scoped if not already registered (defaults to `NoopAgentSessionStore`)

---

#### `HostApplicationBuilderAgentExtensions` — [dotnet/src/Microsoft.Agents.AI.Hosting/HostApplicationBuilderAgentExtensions.cs](dotnet/src/Microsoft.Agents.AI.Hosting/HostApplicationBuilderAgentExtensions.cs)

DI registration for agents on `IHostApplicationBuilder`:

- Overloads mirror `AgentHostingServiceCollectionExtensions` but target `IHostApplicationBuilder`
- Delegates to the `IServiceCollection` extension methods

---

#### `HostedAgentBuilderExtensions` — [dotnet/src/Microsoft.Agents.AI.Hosting/HostedAgentBuilderExtensions.cs](dotnet/src/Microsoft.Agents.AI.Hosting/HostedAgentBuilderExtensions.cs)

Configuration extensions for `IHostedAgentBuilder`:

- `WithInMemorySessionStore(IHostedAgentBuilder)` — registers `InMemoryAgentSessionStore`
- `WithSessionStore<TStore>(IHostedAgentBuilder)` — registers custom session store
- `WithAITool<TTool>(IHostedAgentBuilder)` — registers singleton `AITool`
- `WithAITools(IHostedAgentBuilder, Assembly)` — registers all `AIFunction` tools from assembly

---

#### `HostApplicationBuilderWorkflowExtensions` — [dotnet/src/Microsoft.Agents.AI.Hosting/HostApplicationBuilderWorkflowExtensions.cs](dotnet/src/Microsoft.Agents.AI.Hosting/HostApplicationBuilderWorkflowExtensions.cs)

- `AddWorkflow(IHostApplicationBuilder, string name, Func<IServiceProvider, Workflow> factory)` — registers workflow builder

---

#### `HostedWorkflowBuilderExtensions` — [dotnet/src/Microsoft.Agents.AI.Hosting/HostedWorkflowBuilderExtensions.cs](dotnet/src/Microsoft.Agents.AI.Hosting/HostedWorkflowBuilderExtensions.cs)

- `AddAsAIAgent(IHostedWorkflowBuilder, string? name)` — converts workflow into an `AIAgent` registration

---

## 4. Shared/

**Project Path**: `dotnet/src/Shared/`
**Purpose**: Shared utilities, test infrastructure, and sample helpers. Not a standalone NuGet package.

### 4.1 Throw Utilities

#### `Throw` — [dotnet/src/Shared/Throw/Throw.cs](dotnet/src/Shared/Throw/Throw.cs)

Internal static partial class (`Microsoft.Shared.Diagnostics` namespace). Provides argument validation helpers used throughout the framework.

**Key Methods (971 lines total):**

| Method | Description |
|---|---|
| `IfNull<T>(argument)` | Throws `ArgumentNullException` if null |
| `IfNullOrMemberNull<TParameter, TMember>(argument, member)` | Validates both object and member |
| `IfMemberNull<TParameter, TMember>(argument, member)` | Validates member only |
| `IfNullOrWhitespace(string?)` | Throws for null/whitespace strings |
| `IfNullOrEmpty(string?)` | Throws for null/empty strings |
| `IfBufferTooSmall(bufferSize, requiredSize)` | Buffer size validation |

Uses `[CallerArgumentExpression]` for automatic parameter name capture. Marked `[ExcludeFromCodeCoverage]`.

---

### 4.2 Workflow Utilities

#### `WorkflowFactory` — [dotnet/src/Shared/Workflows/Execution/WorkflowFactory.cs](dotnet/src/Shared/Workflows/Execution/WorkflowFactory.cs)

Internal sealed class for creating workflows from declarative YAML.

**Properties:**

| Property | Type |
|---|---|
| `Functions` | `IList<AIFunction>` |
| `Configuration` | `IConfiguration?` |
| `ConversationId` | `string?` |
| `LoggerFactory` | `ILoggerFactory` |

**Methods:**

- `CreateWorkflow()` → `Workflow` — Uses `DeclarativeWorkflowBuilder.Build<string>()` with `AzureAgentProvider`

---

#### `WorkflowRunner` — [dotnet/src/Shared/Workflows/Execution/WorkflowRunner.cs](dotnet/src/Shared/Workflows/Execution/WorkflowRunner.cs)

Internal sealed class for executing workflows with checkpoint support.

**Properties:**

| Property | Type | Description |
|---|---|---|
| `UseJsonCheckpoints` | `bool` | When true, persists checkpoints to disk |

**Methods:**

| Method | Return Type | Description |
|---|---|---|
| `Notify(string, ConsoleColor?)` | `void` (static) | Console output utility |
| `ExecuteAsync(Func<Workflow>, string)` | `Task` | Full workflow execution with checkpoint management |
| `MonitorAndDisposeWorkflowRunAsync(...)` | `Task<ExternalRequest?>` | Stream monitoring loop |
| `HandleExternalRequestAsync(ExternalRequest)` | `ValueTask<ExternalInputResponse>` | Handles user input/tool approval |
| `ProcessInputMessageAsync(ChatMessage)` | `IAsyncEnumerable<ChatMessage>` | Routes function calls, approvals, etc. |

**Key Workflow Event Types Handled:**

- `ExecutorInvokedEvent`, `ExecutorCompletedEvent`
- `DeclarativeActionInvokedEvent`, `DeclarativeActionCompletedEvent`
- `ExecutorFailedEvent`, `WorkflowErrorEvent`
- `SuperStepCompletedEvent` (checkpoint)
- `RequestInfoEvent` (external input)
- `ConversationUpdateEvent`, `MessageActivityEvent`
- `AgentResponseUpdateEvent`, `AgentResponseEvent`

---

#### `Application` — [dotnet/src/Shared/Workflows/Settings/Application.cs](dotnet/src/Shared/Workflows/Settings/Application.cs)

Internal static class with configuration helpers.

**Nested Class `Settings` (constants):**

| Constant | Value |
|---|---|
| `FoundryEndpoint` | `"FOUNDRY_PROJECT_ENDPOINT"` |
| `FoundryModelMini` | `"FOUNDRY_MODEL_DEPLOYMENT_NAME"` |
| `FoundryModelFull` | `"FOUNDRY_MEDIA_DEPLOYMENT_NAME"` |
| `FoundryGroundingTool` | `"FOUNDRY_CONNECTION_GROUNDING_TOOL"` |

**Methods:** `GetInput(string[])`, `GetRepoFolder()`, `GetValue(IConfiguration, string)`, `InitializeConfig()`

---

### 4.3 Foundry Agents

#### `AgentFactory` — [dotnet/src/Shared/Foundry/Agents/AgentFactory.cs](dotnet/src/Shared/Foundry/Agents/AgentFactory.cs)

Internal static class with extension method for `AIProjectClient`:

- `CreateAgentAsync(AIProjectClient, string agentName, AgentDefinition, string description)` → `ValueTask<AgentVersion>`

---

### 4.4 Test Infrastructure

#### `BaseSample` — [dotnet/src/Shared/Samples/BaseSample.cs](dotnet/src/Shared/Samples/BaseSample.cs)

Abstract base class for xUnit test samples. Inherits `TextWriter` to redirect `Console` output.

**Properties:**

| Property | Type |
|---|---|
| `Output` | `ITestOutputHelper` |
| `LoggerFactory` | `ILoggerFactory` |
| `Console` | `BaseSample` (returns `this`) |

**Methods:**

- `WriteUserMessage(string)` — Formats user message
- `WriteResponseOutput(AgentResponse, bool?)` — Formats agent response with usage
- `WriteMessageOutput(ChatMessage)` — Formats any chat message
- `WriteAgentOutput(AgentResponseUpdate)` — Formats streaming update

---

#### `OrchestrationSample` — [dotnet/src/Shared/Samples/OrchestrationSample.cs](dotnet/src/Shared/Samples/OrchestrationSample.cs)

Abstract base for orchestration test samples. Inherits `BaseSample`.

**Methods:**

- `CreateAgent(string instructions, string? description, string? name, params AIFunction[])` → `ChatClientAgent`
- `CreateChatClient()` → `IChatClient`
- `DisplayHistory(IEnumerable<ChatMessage>)`
- `WriteResponse(IEnumerable<ChatMessage>)` (static)
- `WriteStreamedResponse(IEnumerable<AgentResponseUpdate>)` (static)

**Nested Types:**

- `sealed class OrchestrationMonitor` — Properties: `StreamedResponses`, `History`; Methods: `ResponseCallbackAsync`, `StreamingResultCallbackAsync`

---

#### `Compiler` — [dotnet/src/Shared/CodeTests/Compiler.cs](dotnet/src/Shared/CodeTests/Compiler.cs)

Internal static class for dynamic code compilation in tests.

- `RepoDependencies(params IEnumerable<Type>)` → `IEnumerable<Assembly>`
- `Build(string code, params IEnumerable<Assembly>)` → `Assembly`

Uses Roslyn (`Microsoft.CodeAnalysis.CSharp`) for runtime compilation.

---

#### `SampleEnvironment` — [dotnet/src/Shared/Demos/SampleEnvironment.cs](dotnet/src/Shared/Demos/SampleEnvironment.cs)

Internal static class wrapping `System.Environment` with interactive prompting for missing environment variables. Used by demo/sample projects.

---

## 5. Inheritance Hierarchies

### Agent Hierarchy

```
AIAgent (abstract)
├── ChatClientAgent (sealed) — Primary concrete agent
├── DelegatingAIAgent (abstract) — Decorator base
│   ├── LoggingAgent (sealed) — ILogger instrumentation
│   ├── OpenTelemetryAgent (sealed, IDisposable) — OTel instrumentation
│   ├── FunctionInvocationDelegatingAgent (internal sealed) — Tool middleware
│   ├── AnonymousDelegatingAIAgent (internal sealed) — Builder middleware
│   └── AIHostAgent — Hosting session wrapper
```

### Session Hierarchy

```
AgentSession (abstract)
├── InMemoryAgentSession (abstract) — In-memory history
├── ServiceIdAgentSession (abstract) — Remote service-side history
└── ChatClientAgentSession (sealed) — ChatClientAgent session (dual mode)
```

### ChatHistoryProvider Hierarchy

```
ChatHistoryProvider (abstract)
├── InMemoryChatHistoryProvider (sealed, IList<ChatMessage>) — In-memory storage with reducer
└── ChatHistoryProviderMessageFilter (sealed) — Decorator for filtering
```

### AIContextProvider Hierarchy

```
AIContextProvider (abstract)
├── TextSearchProvider (sealed) — RAG via text search
└── ChatHistoryMemoryProvider (sealed, IDisposable) — Vector store memory
```

### Session Store Hierarchy

```
AgentSessionStore (abstract)
├── NoopAgentSessionStore (sealed) — No persistence
└── InMemoryAgentSessionStore (sealed) — ConcurrentDictionary storage
```

### Response Hierarchy

```
AgentResponse (class)
└── AgentResponse<T> (abstract) — Typed response
```

---

## 6. Design Patterns

### 6.1 Decorator Pattern

The `DelegatingAIAgent` class provides the foundation for the decorator pattern. Each decorator wraps an `InnerAgent` and can intercept, modify, or observe operations:

- **LoggingAgent**: Adds logging before/after operations
- **OpenTelemetryAgent**: Adds distributed tracing spans
- **FunctionInvocationDelegatingAgent**: Wraps tool invocations with middleware
- **AIHostAgent**: Adds session persistence around operations
- **AnonymousDelegatingAIAgent**: Lambda-based interception

### 6.2 Builder Pattern

`AIAgentBuilder` provides a fluent pipeline builder:

```csharp
AIAgent agent = new ChatClientAgent(chatClient)
    .AsBuilder()
    .UseLogging(loggerFactory)
    .UseOpenTelemetry()
    .UseFunctionInvocationMiddleware(...)
    .Build();
```

Factories are applied in reverse order, creating a layered pipeline (first = outermost).

### 6.3 Factory Pattern

Multiple factory patterns are used:

- `ChatClientAgentOptions.ChatHistoryProviderFactory` — creates chat history providers
- `ChatClientAgentOptions.AIContextProviderFactory` — creates context providers
- `AgentHostingServiceCollectionExtensions.AddAIAgent(name, factory)` — factory-based DI registration
- `ChatClientAgentRunOptions.ChatClientFactory` — runtime client wrapping

### 6.4 Service Locator Pattern

`GetService(Type, object?)` is defined on `AIAgent`, `AgentSession`, `AIContextProvider`, and `ChatHistoryProvider`. Enables runtime service resolution without tight coupling.

### 6.5 Two-Phase Lifecycle (Invoking/Invoked)

Both `AIContextProvider` and `ChatHistoryProvider` implement a two-phase pattern:

1. `InvokingAsync` — called before AI model invocation; returns context/history
2. `InvokedAsync` — called after AI model invocation; receives results and any exception

This enables providers to maintain state, reduce history, persist results, etc.

### 6.6 Middleware Pipeline

The agent builder creates a middleware pipeline similar to ASP.NET Core middleware. Each `Use()` call adds a layer. The `AnonymousDelegatingAIAgent` enables inline delegate-based handlers.

### 6.7 Dual Session Strategy

`ChatClientAgentSession` supports two mutually exclusive modes:

- **Server-side**: `ConversationId` — history managed by the AI service
- **Client-side**: `ChatHistoryProvider` — history managed locally

### 6.8 Keyed DI Registration

The hosting extensions register agents as **keyed singletons** in `IServiceCollection`, enabling multiple named agents in a single DI container.

---

## 7. External Dependencies

### Microsoft.Extensions.AI (MEAI)

The framework builds on top of `Microsoft.Extensions.AI` abstractions:

| Type | Usage |
|---|---|
| `IChatClient` | Core model interaction interface; wrapped by `ChatClientAgent` |
| `ChatMessage` | Message representation throughout the framework |
| `ChatRole` | Role enumeration (User, Assistant, System, Tool) |
| `ChatOptions` | Model invocation options (temperature, tools, etc.) |
| `ChatResponse` | Non-streaming model response |
| `ChatResponseUpdate` | Streaming model response chunk |
| `AITool` / `AIFunction` | Tool/function abstractions |
| `AIContent` | Base content type (text, image, function call, etc.) |
| `TextContent` | Text content |
| `FunctionCallContent` | Function invocation request |
| `FunctionResultContent` | Function invocation result |
| `UsageDetails` | Token usage statistics |
| `AdditionalPropertiesDictionary` | Extension property bag |
| `FunctionInvokingChatClient` | Auto function execution wrapper |
| `DelegatingAIFunction` | Function decorator base |

### System.Text.Json

Used for all serialization: session state, context provider state, structured output deserialization.

### Microsoft.Extensions.Hosting / DependencyInjection

Used by the Hosting project for DI integration:

- `IServiceCollection`, `IServiceProvider`, `IHostApplicationBuilder`
- Keyed service registration
- Scoped/singleton lifetime management

---

## Summary Statistics

| Category | Count |
|---|---|
| Abstract classes | 8 (`AIAgent`, `AgentSession`, `AgentResponse<T>`, `DelegatingAIAgent`, `AIContextProvider`, `ChatHistoryProvider`, `InMemoryAgentSession`, `ServiceIdAgentSession`, `AgentSessionStore`, `WorkflowCatalog`) |
| Interfaces | 2 (`IHostedAgentBuilder`, `IHostedWorkflowBuilder`) |
| Sealed concrete classes | ~20+ |
| Enums | 4 (`FailureReason`, `ChatReducerTriggerEvent`, `TextSearchBehavior`, `SearchBehavior`) |
| Extension method classes | ~10 |
| Design patterns identified | 8 (Decorator, Builder, Factory, Service Locator, Two-Phase Lifecycle, Middleware Pipeline, Dual Session Strategy, Keyed DI) |
