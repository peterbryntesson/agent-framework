# Cross-Cutting .NET Projects Research

> **Generated**: 2026-02-20
> **Scope**: `dotnet/src/Microsoft.Agents.AI.Abstractions`, `dotnet/src/Microsoft.Agents.AI.CosmosNoSql`, `dotnet/src/Microsoft.Agents.AI.Mem0`, `dotnet/src/Microsoft.Agents.AI.Purview`, `dotnet/src/Microsoft.Agents.AI.DevUI`

---

## 1. Microsoft.Agents.AI.Abstractions

**Purpose**: Core abstractions layer defining the agent programming model, session management, chat history, context providers, response types, and JSON serialization. All other projects depend on this.

**Dependencies**: `Microsoft.Extensions.AI.Abstractions`, `Microsoft.Extensions.Logging.Abstractions`

### 1.1 Core Agent Model

#### AIAgent (abstract class) — [AIAgent.cs](dotnet/src/Microsoft.Agents.AI.Abstractions/AIAgent.cs)

- **Lines 1–392** — Abstract base class for all AI agents
- Properties: `Id`, `Name`, `Description`, `Metadata` (`AIAgentMetadata`)
- Public overloaded methods (non-streaming):
  - `RunAsync(string, AgentSession?, AgentRunOptions?, CancellationToken)` → [L54](dotnet/src/Microsoft.Agents.AI.Abstractions/AIAgent.cs#L54)
  - `RunAsync(ChatMessage, AgentSession?, AgentRunOptions?, CancellationToken)` → [L79](dotnet/src/Microsoft.Agents.AI.Abstractions/AIAgent.cs#L79)
  - `RunAsync(IEnumerable<ChatMessage>, AgentSession?, AgentRunOptions?, CancellationToken)` → [L104](dotnet/src/Microsoft.Agents.AI.Abstractions/AIAgent.cs#L104)
- Public overloaded methods (streaming):
  - `RunStreamingAsync(string, AgentSession?, AgentRunOptions?, CancellationToken)` → [L136](dotnet/src/Microsoft.Agents.AI.Abstractions/AIAgent.cs#L136)
  - `RunStreamingAsync(ChatMessage, AgentSession?, AgentRunOptions?, CancellationToken)` → [L163](dotnet/src/Microsoft.Agents.AI.Abstractions/AIAgent.cs#L163)
  - `RunStreamingAsync(IEnumerable<ChatMessage>, AgentSession?, AgentRunOptions?, CancellationToken)` → [L190](dotnet/src/Microsoft.Agents.AI.Abstractions/AIAgent.cs#L190)
- Abstract methods (must implement):
  - `RunCoreAsync(IEnumerable<ChatMessage>, AgentSession?, AgentRunOptions?, CancellationToken)` → [L219](dotnet/src/Microsoft.Agents.AI.Abstractions/AIAgent.cs#L219)
  - `RunCoreStreamingAsync(IEnumerable<ChatMessage>, AgentSession?, AgentRunOptions?, CancellationToken)` → [L237](dotnet/src/Microsoft.Agents.AI.Abstractions/AIAgent.cs#L237)
  - `GetNewSessionAsync(CancellationToken)` → [L247](dotnet/src/Microsoft.Agents.AI.Abstractions/AIAgent.cs#L247)
  - `DeserializeSessionAsync(JsonElement, JsonSerializerOptions?, CancellationToken)` → [L260](dotnet/src/Microsoft.Agents.AI.Abstractions/AIAgent.cs#L260)
- Virtual: `GetService(Type, object?)` following `IChatClient.GetService` pattern

#### DelegatingAIAgent — [DelegatingAIAgent.cs](dotnet/src/Microsoft.Agents.AI.Abstractions/DelegatingAIAgent.cs)

- Decorator/middleware pattern for agent pipelines
- Delegates all operations to `InnerAgent`
- Enables composable agent middleware stacks

#### AIAgentMetadata — [AIAgentMetadata.cs](dotnet/src/Microsoft.Agents.AI.Abstractions/AIAgentMetadata.cs)

- `ProviderName` → Used for OpenTelemetry `gen_ai.system` semantic convention attribute

### 1.2 Session Model

#### AgentSession (abstract class) — [AgentSession.cs](dotnet/src/Microsoft.Agents.AI.Abstractions/AgentSession.cs)

- Serializable conversation state container
- `Serialize(JsonSerializerOptions?)` for persistence
- `GetService(Type, object?)` for discovering attached services
- Restored via `AIAgent.DeserializeSessionAsync()`

#### InMemoryAgentSession — [InMemoryAgentSession.cs](dotnet/src/Microsoft.Agents.AI.Abstractions/InMemoryAgentSession.cs)

- Concrete session storing messages locally via `InMemoryChatHistoryProvider`
- Serializable state: `InMemoryAgentSessionState { ChatHistoryProviderState }`
- Default session type for simple scenarios

#### ServiceIdAgentSession — [ServiceIdAgentSession.cs](dotnet/src/Microsoft.Agents.AI.Abstractions/ServiceIdAgentSession.cs)

- Session for remote-service-backed conversations (e.g., OpenAI Assistants, Azure AI Foundry)
- Stores only `ServiceSessionId` locally; actual state lives on the remote service

### 1.3 Chat History

#### ChatHistoryProvider (abstract class) — [ChatHistoryProvider.cs](dotnet/src/Microsoft.Agents.AI.Abstractions/ChatHistoryProvider.cs)

- **Lines 1–229** — Two-phase lifecycle for history management:
  - `InvokingAsync(InvokingContext)` → Returns `IEnumerable<ChatMessage>` for agent context
  - `InvokedAsync(InvokedContext)` → Stores request + response messages
- `InvokedContext` includes:
  - `RequestMessages` — Original request
  - `ChatHistoryProviderMessages` — Messages from history provider
  - `AIContextProviderMessages` — Messages from context providers
  - `ResponseMessages` — Agent response messages
  - `InvokeException` — Exception if agent invocation failed
- Serializable via `Serialize()` / `GetService()` pattern

#### InMemoryChatHistoryProvider — [InMemoryChatHistoryProvider.cs](dotnet/src/Microsoft.Agents.AI.Abstractions/InMemoryChatHistoryProvider.cs)

- Implements `ChatHistoryProvider` + `IList<ChatMessage>` to allow direct manipulation
- **Chat Reduction**: Supports `IChatReducer` with configurable `ChatReducerTriggerEvent`:
  - `AfterMessageAdded` — Reduce after storing messages
  - `BeforeMessagesRetrieval` — Reduce before returning messages to agent
- `InvokingAsync`: Optionally runs reducer before returning history
- `InvokedAsync`: Adds request + AIContextProvider + response messages, optionally reduces after

#### ChatHistoryProviderExtensions — [ChatHistoryProviderExtensions.cs](dotnet/src/Microsoft.Agents.AI.Abstractions/ChatHistoryProviderExtensions.cs)

- `WithMessageFilters()` — Decorator pattern for filtering messages in/out of history
- `WithAIContextProviderMessageRemoval()` — Filters out AIContextProvider messages from stored history

#### ChatHistoryProviderMessageFilter — [ChatHistoryProviderMessageFilter.cs](dotnet/src/Microsoft.Agents.AI.Abstractions/ChatHistoryProviderMessageFilter.cs)

- Internal decorator wrapping inner `ChatHistoryProvider` with `Func<>` filters

### 1.4 Context Providers

#### AIContext — [AIContext.cs](dotnet/src/Microsoft.Agents.AI.Abstractions/AIContext.cs)

- Transient context container passed to agents per-invocation
- `string? Instructions` — Transient system instructions
- `IList<ChatMessage>? Messages` — Become permanent in history
- `IList<AITool>? Tools` — Transient tools for this invocation only

#### AIContextProvider (abstract class) — [AIContextProvider.cs](dotnet/src/Microsoft.Agents.AI.Abstractions/AIContextProvider.cs)

- **Lines 1–206** — Two-phase lifecycle:
  - `InvokingAsync(InvokingContext)` → Returns `AIContext` (instructions, messages, tools)
  - `InvokedAsync(InvokedContext)` → Post-processing after agent invocation
- `InvokingContext`: Contains `RequestMessages`
- `InvokedContext`: Contains `RequestMessages`, `AIContextProviderMessages`, `ResponseMessages`, `InvokeException`
- Supports `Serialize()` and `GetService()` pattern

### 1.5 Response Types

#### AgentResponse — [AgentResponse.cs](dotnet/src/Microsoft.Agents.AI.Abstractions/AgentResponse.cs)

- **Lines 1–407** — Complete non-streaming response
- Key properties:
  - `Messages` (IReadOnlyList<ChatMessage>)
  - `Text` (concatenated text content)
  - `UserInputRequests` (IReadOnlyList<UserInputRequest>)
  - `AgentId`, `ResponseId`, `ContinuationToken`, `CreatedAt`, `Usage`
  - `RawRepresentation` (underlying raw object)
  - `AdditionalProperties`
- Constructor accepts `ChatResponse` for M.E.AI interop
- `Deserialize<T>()` / `TryDeserialize<T>()` for structured output
- `ToAgentResponseUpdates()` for streaming conversion

#### AgentResponse\<T> — [AgentResponse{T}.cs](dotnet/src/Microsoft.Agents.AI.Abstractions/AgentResponse{T}.cs)

- Generic typed response with `Result` property for structured output

#### AgentResponseUpdate — [AgentResponseUpdate.cs](dotnet/src/Microsoft.Agents.AI.Abstractions/AgentResponseUpdate.cs)

- Single streaming chunk
- Properties: `Role`, `Contents`, `Text`, `AuthorName`, `AgentId`, `ResponseId`, `MessageId`, `ContinuationToken`

#### AgentResponseExtensions — [AgentResponseExtensions.cs](dotnet/src/Microsoft.Agents.AI.Abstractions/AgentResponseExtensions.cs)

- **Lines 1–212** — Bidirectional conversions:
  - `AsChatResponse()` — AgentResponse → ChatResponse
  - `AsChatResponseUpdate()` — AgentResponseUpdate → ChatResponseUpdate
  - `ToAgentResponse()` — ChatResponse → AgentResponse
  - `ToAgentResponseAsync()` — IAsyncEnumerable\<AgentResponseUpdate> aggregated to AgentResponse

#### AgentRunOptions — [AgentRunOptions.cs](dotnet/src/Microsoft.Agents.AI.Abstractions/AgentRunOptions.cs)

- `ContinuationToken` — For resume/background operations
- `AllowBackgroundResponses` — Enables background processing
- `AdditionalProperties` — Extensible property bag

### 1.6 Utility & Serialization

#### AdditionalPropertiesExtensions — [AdditionalPropertiesExtensions.cs](dotnet/src/Microsoft.Agents.AI.Abstractions/AdditionalPropertiesExtensions.cs)

- Type-keyed property storage using `typeof(T).FullName!` as key
- `GetValue<T>()`, `SetValue<T>()`, `TryGetValue<T>()`

#### AIContentExtensions — [AIContentExtensions.cs](dotnet/src/Microsoft.Agents.AI.Abstractions/AIContentExtensions.cs)

- Internal text concatenation helpers for `AIContent` / `ChatMessage`

#### AgentAbstractionsJsonUtilities — [AgentAbstractionsJsonUtilities.cs](dotnet/src/Microsoft.Agents.AI.Abstractions/AgentAbstractionsJsonUtilities.cs)

- JSON serialization config with **source generators for AOT/trimming compatibility**
- Chains `AIJsonUtilities` resolver + local `AgentAbstractionsJsonContext` resolver
- Serializable types: `AgentRunOptions`, `AgentResponse`, `AgentResponseUpdate`, session states

---

## 2. Microsoft.Agents.AI.CosmosNoSql

**Purpose**: Azure Cosmos DB implementations for persistent chat history and workflow checkpoint storage.

**Dependencies**: `Microsoft.Agents.AI.Abstractions`, `Microsoft.Agents.AI` (for workflow types), `Azure.Cosmos` SDK

### 2.1 CosmosChatHistoryProvider — [CosmosChatHistoryProvider.cs](dotnet/src/Microsoft.Agents.AI.CosmosNoSql/CosmosChatHistoryProvider.cs)

- **Lines 1–692** — Full `ChatHistoryProvider` implementation with Cosmos DB storage
- **Authentication**: Connection string, `TokenCredential`, or existing `CosmosClient`
- **Partition Strategy**: Hierarchical partition keys (`tenantId/userId/conversationId`) or simple `conversationId`
- **Configuration**:
  - `MaxItemCount = 100` (query page size)
  - `MaxBatchSize = 100` (transactional batch size)
  - `MaxMessagesToRetrieve` (optional limit)
  - `MessageTtlSeconds = 86400` (24-hour TTL)
- **InvokingAsync**: SQL query with `ORDER BY timestamp`, optional `DESC` + reverse for most-recent-N messages
- **InvokedAsync**: Transactional batch writes with auto-split on `RequestEntityTooLarge` (413)
- **Utilities**: `GetMessageCountAsync()`, `ClearMessagesAsync()`
- **Document Model**: `CosmosMessageDocument` with `id`, `conversationId`, `timestamp`, `messageId`, `role`, `message` (serialized JSON), `type`, `ttl`, `tenantId`, `userId`, `sessionId`

### 2.2 CosmosCheckpointStore — [CosmosCheckpointStore.cs](dotnet/src/Microsoft.Agents.AI.CosmosNoSql/CosmosCheckpointStore.cs)

- **Lines 1–278** — `JsonCheckpointStore` implementation for durable workflow checkpointing
- `CreateCheckpointAsync()`: Creates document with `runId_checkpointId` as id
- `RetrieveCheckpointAsync()`: Point-read by id
- `RetrieveIndexAsync()`: Query checkpoints by `runId`, optionally filtered by `parentCheckpointId`
- Non-generic `CosmosCheckpointStore` inherits `CosmosCheckpointStore<JsonElement>`

### 2.3 Extension Methods

#### CosmosDBChatExtensions — [CosmosDBChatExtensions.cs](dotnet/src/Microsoft.Agents.AI.CosmosNoSql/CosmosDBChatExtensions.cs)

- `WithCosmosDBChatHistoryProvider()` on `ChatClientAgentOptions`
- `WithCosmosDBChatHistoryProviderUsingManagedIdentity()`

#### CosmosDBWorkflowExtensions — [CosmosDBWorkflowExtensions.cs](dotnet/src/Microsoft.Agents.AI.CosmosNoSql/CosmosDBWorkflowExtensions.cs)

- **Lines 1–235** — Static factory methods:
  - `CreateCheckpointStore()` / `CreateCheckpointStoreUsingManagedIdentity()`
  - Generic and non-generic variants

---

## 3. Microsoft.Agents.AI.Mem0

**Purpose**: `AIContextProvider` implementation backed by the Mem0 memory service for semantic memory retrieval and persistence.

**Dependencies**: `Microsoft.Agents.AI.Abstractions`, `Microsoft.Extensions.Logging.Abstractions`

### 3.1 Mem0Provider — [Mem0Provider.cs](dotnet/src/Microsoft.Agents.AI.Mem0/Mem0Provider.cs)

- **Lines 1–298** — `AIContextProvider` implementation
- **InvokingAsync**: Joins request message texts → semantic search via Mem0 → returns `AIContext` with retrieved memories as user message
- **InvokedAsync**: Persists user/assistant/system messages as Mem0 memories (fire-and-forget)
- Configurable context prompt template, sensitive data redaction
- `ClearStoredMemoriesAsync()` utility
- Serializable state: `Mem0State { StorageScope, SearchScope }`
- Scoping: `Mem0ProviderScope` with `ApplicationId`, `AgentId`, `ThreadId`, `UserId`

### 3.2 Mem0Client — [Mem0Client.cs](dotnet/src/Microsoft.Agents.AI.Mem0/Mem0Client.cs)

- Internal HTTP client for Mem0 REST API
- `SearchAsync()` → `POST /v1/memories/search/`
- `CreateMemoryAsync()` → `POST /v1/memories/`
- `ClearMemoryAsync()` → `DELETE /v1/memories/?scope_params`
- Source-generated JSON: `Mem0SourceGenerationContext`

### 3.3 Configuration

#### Mem0ProviderOptions — [Mem0ProviderOptions.cs](dotnet/src/Microsoft.Agents.AI.Mem0/Mem0ProviderOptions.cs)

- `ContextPrompt` — Template for injecting memories into context
- `EnableSensitiveTelemetryData` — Controls logging of sensitive data

#### Mem0ProviderScope — [Mem0ProviderScope.cs](dotnet/src/Microsoft.Agents.AI.Mem0/Mem0ProviderScope.cs)

- `ApplicationId`, `AgentId`, `ThreadId`, `UserId` — Scoping dimensions for memory isolation

### 3.4 Serialization

#### Mem0JsonUtilities — [Mem0JsonUtilities.cs](dotnet/src/Microsoft.Agents.AI.Mem0/Mem0JsonUtilities.cs)

- JSON serialization config chaining `AgentAbstractionsJsonUtilities` resolver for AOT support

---

## 4. Microsoft.Agents.AI.Purview

**Purpose**: Microsoft Purview Data Loss Prevention (DLP) integration for AI agents and chat clients. Acts as middleware that intercepts prompts and responses, evaluates them against Purview policies, and blocks content violating DLP policies.

**Dependencies**: `Microsoft.Agents.AI.Abstractions`, `Microsoft.Agents.AI`, `Azure.Identity`, `Microsoft.Extensions.AI`, `Microsoft.Extensions.Caching.Memory`, `Microsoft.Extensions.DependencyInjection`

**VersionSuffix**: `alpha`

### 4.1 Architecture Overview

The Purview project implements a **middleware pattern** at two levels:

1. **Agent level**: `PurviewAgent` wraps `AIAgent` → intercepts `RunCoreAsync` / `RunCoreStreamingAsync`
2. **Chat client level**: `PurviewChatClient` wraps `IChatClient` → intercepts `GetResponseAsync` / `GetStreamingResponseAsync`

Both delegate to `PurviewWrapper` which orchestrates the DLP evaluation flow.

### 4.2 Middleware Entry Points

#### PurviewAgent — [PurviewAgent.cs](dotnet/src/Microsoft.Agents.AI.Purview/PurviewAgent.cs)

- **Lines 1–80** — `AIAgent` subclass wrapping inner agent
- Intercepts `RunCoreAsync` → delegates to `PurviewWrapper.ProcessAgentContentAsync()`
- Streaming: Materializes full response then converts to `AgentResponseUpdate` stream
- Implements `IDisposable` for cleanup of wrapper and inner agent

#### PurviewChatClient — [PurviewChatClient.cs](dotnet/src/Microsoft.Agents.AI.Purview/PurviewChatClient.cs)

- `IChatClient` implementation wrapping inner chat client
- `GetResponseAsync` → delegates to `PurviewWrapper.ProcessChatContentAsync()`
- Streaming: Materializes full response then converts to `ChatResponseUpdate` stream
- `GetService()` delegates to inner client

#### PurviewExtensions — [PurviewExtensions.cs](dotnet/src/Microsoft.Agents.AI.Purview/PurviewExtensions.cs)

- `WithPurview(AIAgentBuilder, ...)` — Adds Purview middleware to agent builder pipeline
- `WithPurview(ChatClientBuilder, ...)` — Adds Purview middleware to chat client pipeline
- `PurviewAgentMiddleware(...)` — Creates standalone agent middleware delegate
- `PurviewChatMiddleware(...)` — Creates standalone chat client middleware delegate
- `SetUserId(ChatMessage, Guid)` — Tags messages with Entra user IDs
- Internally constructs `PurviewWrapper` via private DI container

### 4.3 Core Processing Pipeline

#### PurviewWrapper — [PurviewWrapper.cs](dotnet/src/Microsoft.Agents.AI.Purview/PurviewWrapper.cs)

- **Lines 1–210** — Central orchestrator
- **ProcessChatContentAsync / ProcessAgentContentAsync** flow:
  1. Process prompt messages through `IScopedContentProcessor.ProcessMessagesAsync()` → `shouldBlockPrompt`
  2. If blocked → Return `BlockedPromptMessage` from `PurviewSettings`
  3. If not blocked → Forward to inner agent/chat client
  4. Process response messages through `IScopedContentProcessor.ProcessMessagesAsync()` → `shouldBlockResponse`
  5. If blocked → Return `BlockedResponseMessage` from `PurviewSettings`
  6. If not blocked → Return original response
- `IgnoreExceptions` setting: Logs but swallows Purview errors
- Session ID resolution: From `ChatClientAgentSession.ConversationId`, message `AdditionalProperties`, or new GUID

#### ScopedContentProcessor — [ScopedContentProcessor.cs](dotnet/src/Microsoft.Agents.AI.Purview/ScopedContentProcessor.cs)

- **Lines 1–346** — Orchestrates protection scopes, process content, and content activities calls
- `ProcessMessagesAsync()`:
  1. Maps `ChatMessage` list → `List<ProcessContentRequest>`
  2. Resolves user ID from message `AdditionalProperties["userId"]`, `AuthorName` (GUID format), or token
  3. Resolves tenant ID from settings or token
  4. For each request: `ProcessContentWithProtectionScopesAsync()`
  5. Checks `PolicyActions` for `BlockAccess` or `RestrictionAction.Block`
- `ProcessContentWithProtectionScopesAsync()`:
  1. Creates `ProtectionScopesRequest` → Checks cache
  2. Gets or fetches protection scopes → Caches response
  3. `CheckApplicableScopes()` → Matches activities and locations
  4. If `EvaluateOffline` → Queues background `ProcessContentJob`
  5. If `EvaluateInline` → Calls `ProcessContentAsync()` synchronously
  6. If no scopes match → Queues background `ContentActivityJob`

### 4.4 Purview API Client

#### IPurviewClient — [IPurviewClient.cs](dotnet/src/Microsoft.Agents.AI.Purview/IPurviewClient.cs)

- `GetUserInfoFromTokenAsync()` → Extracts user/tenant/client info from JWT
- `ProcessContentAsync(ProcessContentRequest)` → `POST /users/{userId}/dataSecurityAndGovernance/processContent`
- `GetProtectionScopesAsync(ProtectionScopesRequest)` → `POST /users/{userId}/dataSecurityAndGovernance/protectionScopes/compute`
- `SendContentActivitiesAsync(ContentActivitiesRequest)` → `POST /{userId}/dataSecurityAndGovernance/activities/contentActivities`

#### PurviewClient — [PurviewClient.cs](dotnet/src/Microsoft.Agents.AI.Purview/PurviewClient.cs)

- **Lines 1–324** — HTTP implementation using `TokenCredential` for auth against Microsoft Graph
- JWT token parsing for user/tenant/client extraction
- Custom exception mapping: 429 → `PurviewRateLimitException`, 401/403 → `PurviewAuthenticationException`, 402 → `PurviewPaymentRequiredException`
- Uses `PurviewSerializationUtils.SerializationSettings` for all JSON serialization

### 4.5 Background Job Processing

#### IChannelHandler / ChannelHandler — [IChannelHandler.cs](dotnet/src/Microsoft.Agents.AI.Purview/IChannelHandler.cs), [ChannelHandler.cs](dotnet/src/Microsoft.Agents.AI.Purview/ChannelHandler.cs)

- `System.Threading.Channels`-based bounded job queue
- `QueueJob(BackgroundJobBase)` — Non-blocking write to channel
- `AddRunner(Func<Channel, Task>)` — Registers consumer tasks
- `StopAndWaitForCompletionAsync()` — Graceful shutdown

#### IBackgroundJobRunner / BackgroundJobRunner — [IBackgroundJobRunner.cs](dotnet/src/Microsoft.Agents.AI.Purview/IBackgroundJobRunner.cs), [BackgroundJobRunner.cs](dotnet/src/Microsoft.Agents.AI.Purview/BackgroundJobRunner.cs)

- Spawns `MaxConcurrentJobConsumers` (default 10) consumer tasks
- Routes jobs: `ProcessContentJob` → `ProcessContentAsync()`, `ContentActivityJob` → `SendContentActivitiesAsync()`

#### Job Types — [Models/Jobs/](dotnet/src/Microsoft.Agents.AI.Purview/Models/Jobs/)

- `BackgroundJobBase` — Abstract marker base
- `ProcessContentJob` — Wraps `ProcessContentRequest` for offline evaluation
- `ContentActivityJob` — Wraps `ContentActivitiesRequest` for audit/activity logging

### 4.6 Caching

#### ICacheProvider / CacheProvider — [ICacheProvider.cs](dotnet/src/Microsoft.Agents.AI.Purview/ICacheProvider.cs), [CacheProvider.cs](dotnet/src/Microsoft.Agents.AI.Purview/CacheProvider.cs)

- Generic typed cache over `IDistributedCache`
- Uses JSON serialization for both keys and values
- Default: In-memory cache with 100 MB size limit
- Configurable TTL (default 30 minutes)
- Primary use: Caching `ProtectionScopesResponse` to avoid redundant API calls

### 4.7 Configuration

#### PurviewSettings — [PurviewSettings.cs](dotnet/src/Microsoft.Agents.AI.Purview/PurviewSettings.cs)

| Property | Type | Default | Description |
|---|---|---|---|
| `AppName` | `string` | (required) | Publicly visible application name |
| `AppVersion` | `string?` | `null` | Application version string |
| `TenantId` | `string?` | `null` | Entra tenant ID (inferred from token if null) |
| `PurviewAppLocation` | `PurviewAppLocation?` | `null` | Policy location identifier |
| `IgnoreExceptions` | `bool` | `false` | Swallow Purview errors instead of throwing |
| `GraphBaseUri` | `Uri` | `https://graph.microsoft.com/v1.0/` | Microsoft Graph base URI |
| `BlockedPromptMessage` | `string` | `"Prompt blocked by policies"` | Message when prompt is blocked |
| `BlockedResponseMessage` | `string` | `"Response blocked by policies"` | Message when response is blocked |
| `InMemoryCacheSizeLimit` | `long?` | `100_000_000` | In-memory cache size limit (bytes) |
| `CacheTTL` | `TimeSpan` | `30 minutes` | Cache entry time-to-live |
| `PendingBackgroundJobLimit` | `int` | `100` | Max queued background jobs |
| `MaxConcurrentJobConsumers` | `int` | `10` | Concurrent job processor count |

#### PurviewAppLocation — [PurviewAppLocation.cs](dotnet/src/Microsoft.Agents.AI.Purview/PurviewAppLocation.cs)

- `PurviewLocationType` enum: `Application`, `Uri`, `Domain`
- Maps to OData `PolicyLocation` types

### 4.8 Exception Hierarchy

| Exception | Base | Description |
|---|---|---|
| `PurviewException` | `Exception` | Base exception |
| `PurviewRequestException` | `PurviewException` | HTTP request errors (includes `StatusCode`) |
| `PurviewRateLimitException` | `PurviewException` | HTTP 429 rate limiting |
| `PurviewAuthenticationException` | `PurviewException` | HTTP 401/403 auth errors |
| `PurviewPaymentRequiredException` | `PurviewException` | HTTP 402 payment required |
| `PurviewJobException` | `PurviewException` | Background job queue errors |
| `PurviewJobLimitExceededException` | `PurviewJobException` | Queue full |

### 4.9 Model Structure — [Models/](dotnet/src/Microsoft.Agents.AI.Purview/Models/)

- **Models/Common/**: `Activity` (enum), `DlpAction`, `DlpActionInfo`, `RestrictionAction`, `PolicyLocation`, `PolicyScopeBase`, `ContentToProcess`, `ContentBase`, `TokenInfo`, `DeviceMetadata`, `ProtectedAppMetadata`, `IntegratedAppMetadata`, `ActivityMetadata`, `ExecutionMode`, `ProtectionScopeActivities`, `ProtectionScopeState`, `ProcessingError`, etc.
- **Models/Requests/**: `ProcessContentRequest`, `ProtectionScopesRequest`, `ContentActivitiesRequest`
- **Models/Responses/**: `ProcessContentResponse`, `ProtectionScopesResponse`, `ContentActivitiesResponse`
- **Models/Jobs/**: `BackgroundJobBase`, `ProcessContentJob`, `ContentActivityJob`

### 4.10 Serialization

#### PurviewSerializationUtils — [Serialization/PurviewSerializationUtils.cs](dotnet/src/Microsoft.Agents.AI.Purview/Serialization/PurviewSerializationUtils.cs)

- Source-generated `SourceGenerationContext` for AOT compatibility
- `camelCase` property naming, null-ignoring serialization
- Registered types: all request, response, and cache key types

---

## 5. Microsoft.Agents.AI.DevUI

**Purpose**: Developer UI for exploring registered agents and workflows. Serves an embedded SPA with REST API endpoints for entity discovery and metadata inspection.

**Dependencies**: `Microsoft.Agents.AI.Abstractions`, `Microsoft.Agents.AI` (for workflow types), `Microsoft.AspNetCore.*`, `Microsoft.Extensions.DependencyInjection`

### 5.1 Middleware & Resource Serving

#### DevUIMiddleware — [DevUIMiddleware.cs](dotnet/src/Microsoft.Agents.AI.DevUI/DevUIMiddleware.cs)

- **Lines 1–263** — ASP.NET Core middleware serving embedded SPA resources
- Reads assembly manifest resources into `FrozenDictionary<string, CachedResource>`
- GZip compression for cacheable resources
- ETag support with `304 Not Modified` responses
- Client-side routing fallback to `index.html` for SPA navigation

### 5.2 Registration & Routing

#### DevUIExtensions — [DevUIExtensions.cs](dotnet/src/Microsoft.Agents.AI.DevUI/DevUIExtensions.cs)

- `MapDevUI(IEndpointRouteBuilder)` → Maps:
  - `/devui/{*path}` → SPA resources
  - `/meta` → Metadata endpoint
  - `/v1/entities` → Entity discovery

#### ServiceCollectionsExtensions — [ServiceCollectionsExtensions.cs](dotnet/src/Microsoft.Agents.AI.DevUI/ServiceCollectionsExtensions.cs)

- `AddDevUI(IServiceCollection)` — Registers keyed `AIAgent` factory
- Uses `KeyedService.AnyKey` to resolve agents by string key (agent ID)
- Resolves from `Workflow` or default `AIAgent` registrations

#### HostApplicationBuilderExtensions — [HostApplicationBuilderExtensions.cs](dotnet/src/Microsoft.Agents.AI.DevUI/HostApplicationBuilderExtensions.cs)

- `AddDevUI(IHostApplicationBuilder)` convenience method

### 5.3 API Endpoints

#### MetaApiExtensions — [MetaApiExtensions.cs](dotnet/src/Microsoft.Agents.AI.DevUI/MetaApiExtensions.cs)

- `GET /meta` → Returns `MetaResponse`:
  - `UiMode` (e.g., "agent", "workflow"), `Version`, `Framework`, `Runtime`
  - `Capabilities` dictionary, `AuthRequired` flag

#### EntitiesApiExtensions — [EntitiesApiExtensions.cs](dotnet/src/Microsoft.Agents.AI.DevUI/EntitiesApiExtensions.cs)

- **Lines 1–304** — Entity discovery endpoints:
  - `GET /v1/entities` → Lists all registered `AIAgent` and `Workflow` entities
  - `GET /v1/entities/{entityId}/info` → Detailed entity info
- Discovers agents from DI container, extracts tools from `ChatOptions`, instructions from `ChatClientAgent`

### 5.4 Data Models

#### EntityInfo — [EntityInfo.cs](dotnet/src/Microsoft.Agents.AI.DevUI/EntityInfo.cs)

- Rich entity model with:
  - Common: `Id`, `Type` (agent/workflow), `Name`, `Description`, `Framework`
  - Agent: `Instructions`, `ModelId`, `ChatClientType`, `ContextProviders`, `Middleware`
  - Workflow: `Executors`, `WorkflowDump`, `InputSchema`, `StartExecutorId`

#### MetaResponse — [MetaResponse.cs](dotnet/src/Microsoft.Agents.AI.DevUI/MetaResponse.cs)

- Record with `UiMode`, `Version`, `Framework`, `Runtime`, `Capabilities`, `AuthRequired`

### 5.5 Workflow Visualization

#### WorkflowSerializationExtensions — [WorkflowSerializationExtensions.cs](dotnet/src/Microsoft.Agents.AI.DevUI/WorkflowSerializationExtensions.cs)

- **Lines 1–208** — Converts `Workflow` objects to Python-compatible dictionary format
- Handles graph structures: `DirectEdge`, `FanOutEdge`, `FanInEdge`
- Produces serializable dictionaries for DevUI rendering

### 5.6 Serialization

#### EntitiesJsonContext — [EntitiesJsonContext.cs](dotnet/src/Microsoft.Agents.AI.DevUI/EntitiesJsonContext.cs)

- Source-generated JSON context for AOT compatibility
- Registered types: `EntityInfo`, `MetaResponse`, collections

---

## 6. Cross-Cutting Patterns

### 6.1 Middleware / Decorator Pattern

All middleware-capable projects follow the same pattern:

| Project | Agent Middleware | Chat Client Middleware |
|---|---|---|
| **Abstractions** | `DelegatingAIAgent` (base) | N/A |
| **Purview** | `PurviewAgent` (wraps `AIAgent`) | `PurviewChatClient` (wraps `IChatClient`) |

Builder extension methods: `WithPurview(AIAgentBuilder)`, `WithPurview(ChatClientBuilder)`

### 6.2 Two-Phase Lifecycle (Invoking/Invoked)

Shared by `ChatHistoryProvider` and `AIContextProvider`:

1. **InvokingAsync** — Called before agent invocation; provides context/history
2. **InvokedAsync** — Called after agent invocation; stores/processes results

Implementations:
- `InMemoryChatHistoryProvider` (Abstractions)
- `CosmosChatHistoryProvider` (CosmosNoSql)
- `Mem0Provider` (Mem0)

### 6.3 JSON Serialization Strategy

All projects use **System.Text.Json source generators** for AOT/trimming compatibility:

| Project | Context Class | Resolver Chain |
|---|---|---|
| Abstractions | `AgentAbstractionsJsonContext` | `AIJsonUtilities` → `AgentAbstractionsJsonContext` |
| Mem0 | `Mem0SourceGenerationContext` | `AgentAbstractionsJsonUtilities` → `Mem0SourceGenerationContext` |
| Purview | `SourceGenerationContext` | Standalone with `camelCase` policy |
| DevUI | `EntitiesJsonContext` | Standalone |

### 6.4 Service Discovery / GetService Pattern

Multiple types implement a `GetService(Type, object?)` method following the `Microsoft.Extensions.AI.IChatClient.GetService` convention:
- `AIAgent.GetService()`
- `AgentSession.GetService()`
- `ChatHistoryProvider.GetService()`
- `AIContextProvider.GetService()`

### 6.5 Authentication Patterns

| Project | Auth Method |
|---|---|
| CosmosNoSql | Connection string or `TokenCredential` |
| Mem0 | API key via HTTP header |
| Purview | `Azure.Core.TokenCredential` → Microsoft Graph JWT |

### 6.6 Caching

| Project | Cache Type | Strategy |
|---|---|---|
| DevUI | `FrozenDictionary` (in-memory) | GZip-compressed embedded resources with ETag |
| Purview | `IDistributedCache` (default: `MemoryDistributedCache`) | Protection scopes cached with configurable TTL |

### 6.7 Background Processing

| Project | Mechanism |
|---|---|
| Purview | `System.Threading.Channels` bounded channel with concurrent consumers |

---

## 7. Public API Surface Summary

### Public Types

| Type | Project | Visibility |
|---|---|---|
| `AIAgent` | Abstractions | `public abstract` |
| `AgentSession` | Abstractions | `public abstract` |
| `ChatHistoryProvider` | Abstractions | `public abstract` |
| `AIContextProvider` | Abstractions | `public abstract` |
| `AIContext` | Abstractions | `public` |
| `AgentResponse` / `AgentResponse<T>` | Abstractions | `public` |
| `AgentResponseUpdate` | Abstractions | `public` |
| `AgentRunOptions` | Abstractions | `public` |
| `DelegatingAIAgent` | Abstractions | `public` |
| `InMemoryAgentSession` | Abstractions | `public` |
| `InMemoryChatHistoryProvider` | Abstractions | `public` |
| `ServiceIdAgentSession` | Abstractions | `public` |
| `AIAgentMetadata` | Abstractions | `public` |
| `CosmosChatHistoryProvider` | CosmosNoSql | `public` |
| `CosmosCheckpointStore` / `CosmosCheckpointStore<T>` | CosmosNoSql | `public` |
| `CosmosDBChatExtensions` | CosmosNoSql | `public static` |
| `CosmosDBWorkflowExtensions` | CosmosNoSql | `public static` |
| `Mem0Provider` | Mem0 | `public` |
| `Mem0ProviderOptions` | Mem0 | `public` |
| `Mem0ProviderScope` | Mem0 | `public` |
| `PurviewExtensions` | Purview | `public static` |
| `PurviewSettings` | Purview | `public` |
| `PurviewAppLocation` | Purview | `public` |
| `PurviewLocationType` | Purview | `public enum` |
| `PurviewException` (+ subclasses) | Purview | `public` |
| `DevUIExtensions` | DevUI | `public static` |
| `ServiceCollectionsExtensions` | DevUI | `public static` |
| `HostApplicationBuilderExtensions` | DevUI | `public static` |

### Internal-Only Types

Most implementation types across all projects are `internal` (e.g., `PurviewAgent`, `PurviewClient`, `PurviewWrapper`, `ScopedContentProcessor`, `Mem0Client`, `DevUIMiddleware`, all Purview models). This keeps the public API surface minimal and focused on configuration/extension methods.

---

## 8. File Inventory

### Abstractions (20 files)

| File | Lines | Purpose |
|---|---|---|
| [AIAgent.cs](dotnet/src/Microsoft.Agents.AI.Abstractions/AIAgent.cs) | 392 | Abstract agent base class |
| [AIContext.cs](dotnet/src/Microsoft.Agents.AI.Abstractions/AIContext.cs) | ~30 | Transient invocation context |
| [AIContextProvider.cs](dotnet/src/Microsoft.Agents.AI.Abstractions/AIContextProvider.cs) | 206 | Abstract context provider |
| [AIContentExtensions.cs](dotnet/src/Microsoft.Agents.AI.Abstractions/AIContentExtensions.cs) | ~40 | Text concatenation helpers |
| [AIAgentMetadata.cs](dotnet/src/Microsoft.Agents.AI.Abstractions/AIAgentMetadata.cs) | ~20 | OpenTelemetry provider name |
| [AgentResponse.cs](dotnet/src/Microsoft.Agents.AI.Abstractions/AgentResponse.cs) | 407 | Non-streaming response |
| [AgentResponse{T}.cs](dotnet/src/Microsoft.Agents.AI.Abstractions/AgentResponse{T}.cs) | ~30 | Typed response |
| [AgentResponseUpdate.cs](dotnet/src/Microsoft.Agents.AI.Abstractions/AgentResponseUpdate.cs) | ~80 | Streaming update |
| [AgentResponseExtensions.cs](dotnet/src/Microsoft.Agents.AI.Abstractions/AgentResponseExtensions.cs) | 212 | Response type conversions |
| [AgentRunOptions.cs](dotnet/src/Microsoft.Agents.AI.Abstractions/AgentRunOptions.cs) | ~30 | Run configuration |
| [AgentSession.cs](dotnet/src/Microsoft.Agents.AI.Abstractions/AgentSession.cs) | ~60 | Abstract session |
| [ChatHistoryProvider.cs](dotnet/src/Microsoft.Agents.AI.Abstractions/ChatHistoryProvider.cs) | 229 | Abstract history provider |
| [ChatHistoryProviderExtensions.cs](dotnet/src/Microsoft.Agents.AI.Abstractions/ChatHistoryProviderExtensions.cs) | ~50 | Filter decorators |
| [ChatHistoryProviderMessageFilter.cs](dotnet/src/Microsoft.Agents.AI.Abstractions/ChatHistoryProviderMessageFilter.cs) | ~70 | Filter implementation |
| [DelegatingAIAgent.cs](dotnet/src/Microsoft.Agents.AI.Abstractions/DelegatingAIAgent.cs) | ~80 | Decorator base |
| [InMemoryAgentSession.cs](dotnet/src/Microsoft.Agents.AI.Abstractions/InMemoryAgentSession.cs) | ~80 | In-memory session |
| [InMemoryChatHistoryProvider.cs](dotnet/src/Microsoft.Agents.AI.Abstractions/InMemoryChatHistoryProvider.cs) | ~180 | In-memory history |
| [ServiceIdAgentSession.cs](dotnet/src/Microsoft.Agents.AI.Abstractions/ServiceIdAgentSession.cs) | ~50 | Remote session |
| [AdditionalPropertiesExtensions.cs](dotnet/src/Microsoft.Agents.AI.Abstractions/AdditionalPropertiesExtensions.cs) | ~50 | Type-keyed properties |
| [AgentAbstractionsJsonUtilities.cs](dotnet/src/Microsoft.Agents.AI.Abstractions/AgentAbstractionsJsonUtilities.cs) | ~60 | AOT JSON config |

### CosmosNoSql (4 files)

| File | Lines | Purpose |
|---|---|---|
| [CosmosChatHistoryProvider.cs](dotnet/src/Microsoft.Agents.AI.CosmosNoSql/CosmosChatHistoryProvider.cs) | 692 | Cosmos DB chat history |
| [CosmosCheckpointStore.cs](dotnet/src/Microsoft.Agents.AI.CosmosNoSql/CosmosCheckpointStore.cs) | 278 | Workflow checkpoints |
| [CosmosDBChatExtensions.cs](dotnet/src/Microsoft.Agents.AI.CosmosNoSql/CosmosDBChatExtensions.cs) | ~50 | Chat builder extensions |
| [CosmosDBWorkflowExtensions.cs](dotnet/src/Microsoft.Agents.AI.CosmosNoSql/CosmosDBWorkflowExtensions.cs) | 235 | Workflow builder extensions |

### Mem0 (5 files)

| File | Lines | Purpose |
|---|---|---|
| [Mem0Provider.cs](dotnet/src/Microsoft.Agents.AI.Mem0/Mem0Provider.cs) | 298 | AIContextProvider impl |
| [Mem0Client.cs](dotnet/src/Microsoft.Agents.AI.Mem0/Mem0Client.cs) | ~120 | HTTP client |
| [Mem0ProviderOptions.cs](dotnet/src/Microsoft.Agents.AI.Mem0/Mem0ProviderOptions.cs) | ~20 | Configuration |
| [Mem0ProviderScope.cs](dotnet/src/Microsoft.Agents.AI.Mem0/Mem0ProviderScope.cs) | ~20 | Memory scoping |
| [Mem0JsonUtilities.cs](dotnet/src/Microsoft.Agents.AI.Mem0/Mem0JsonUtilities.cs) | ~30 | AOT JSON |

### Purview (72 files)

| Category | Key Files | Purpose |
|---|---|---|
| **Core** | [PurviewAgent.cs](dotnet/src/Microsoft.Agents.AI.Purview/PurviewAgent.cs), [PurviewChatClient.cs](dotnet/src/Microsoft.Agents.AI.Purview/PurviewChatClient.cs) | Middleware wrappers |
| **Orchestration** | [PurviewWrapper.cs](dotnet/src/Microsoft.Agents.AI.Purview/PurviewWrapper.cs), [ScopedContentProcessor.cs](dotnet/src/Microsoft.Agents.AI.Purview/ScopedContentProcessor.cs) | DLP evaluation pipeline |
| **API Client** | [PurviewClient.cs](dotnet/src/Microsoft.Agents.AI.Purview/PurviewClient.cs), [IPurviewClient.cs](dotnet/src/Microsoft.Agents.AI.Purview/IPurviewClient.cs) | Microsoft Graph HTTP calls |
| **Background** | [BackgroundJobRunner.cs](dotnet/src/Microsoft.Agents.AI.Purview/BackgroundJobRunner.cs), [ChannelHandler.cs](dotnet/src/Microsoft.Agents.AI.Purview/ChannelHandler.cs) | Async job processing |
| **Cache** | [CacheProvider.cs](dotnet/src/Microsoft.Agents.AI.Purview/CacheProvider.cs) | Protection scope caching |
| **Config** | [PurviewSettings.cs](dotnet/src/Microsoft.Agents.AI.Purview/PurviewSettings.cs), [PurviewAppLocation.cs](dotnet/src/Microsoft.Agents.AI.Purview/PurviewAppLocation.cs), [PurviewExtensions.cs](dotnet/src/Microsoft.Agents.AI.Purview/PurviewExtensions.cs), [Constants.cs](dotnet/src/Microsoft.Agents.AI.Purview/Constants.cs) | Configuration & DI |
| **Exceptions** | 7 files in [Exceptions/](dotnet/src/Microsoft.Agents.AI.Purview/Exceptions/) | Error hierarchy |
| **Models** | ~46 files in [Models/](dotnet/src/Microsoft.Agents.AI.Purview/Models/) | Graph API request/response/common models |
| **Serialization** | [PurviewSerializationUtils.cs](dotnet/src/Microsoft.Agents.AI.Purview/Serialization/PurviewSerializationUtils.cs) | AOT JSON source gen |

### DevUI (10 files)

| File | Lines | Purpose |
|---|---|---|
| [DevUIMiddleware.cs](dotnet/src/Microsoft.Agents.AI.DevUI/DevUIMiddleware.cs) | 263 | Embedded SPA serving |
| [DevUIExtensions.cs](dotnet/src/Microsoft.Agents.AI.DevUI/DevUIExtensions.cs) | ~40 | Route mapping |
| [ServiceCollectionsExtensions.cs](dotnet/src/Microsoft.Agents.AI.DevUI/ServiceCollectionsExtensions.cs) | ~50 | DI registration |
| [HostApplicationBuilderExtensions.cs](dotnet/src/Microsoft.Agents.AI.DevUI/HostApplicationBuilderExtensions.cs) | ~30 | Builder extensions |
| [MetaApiExtensions.cs](dotnet/src/Microsoft.Agents.AI.DevUI/MetaApiExtensions.cs) | ~40 | /meta endpoint |
| [EntitiesApiExtensions.cs](dotnet/src/Microsoft.Agents.AI.DevUI/EntitiesApiExtensions.cs) | 304 | Entity discovery API |
| [EntitiesJsonContext.cs](dotnet/src/Microsoft.Agents.AI.DevUI/EntitiesJsonContext.cs) | ~15 | AOT JSON |
| [MetaResponse.cs](dotnet/src/Microsoft.Agents.AI.DevUI/MetaResponse.cs) | ~20 | Metadata model |
| [EntityInfo.cs](dotnet/src/Microsoft.Agents.AI.DevUI/EntityInfo.cs) | ~80 | Entity model |
| [WorkflowSerializationExtensions.cs](dotnet/src/Microsoft.Agents.AI.DevUI/WorkflowSerializationExtensions.cs) | 208 | Workflow → dict |
