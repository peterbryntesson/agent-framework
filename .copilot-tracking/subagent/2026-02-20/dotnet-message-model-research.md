# .NET Agent Framework — Message & Content Type Model Research

> Generated: 2026-02-20
> Source: `dotnet/src/` in agent-framework-peterbryntesson
> Purpose: Full type-system reference for C++ port

---

## 1. External Dependency: `Microsoft.Extensions.AI` (M.E.AI)

**Package versions (from Directory.Packages.props):**

- `Microsoft.Extensions.AI` — 10.2.0
- `Microsoft.Extensions.AI.Abstractions` — 10.2.0
- `Microsoft.Extensions.AI.OpenAI` — 10.2.0-preview.1.26063.2

The agent framework's `Microsoft.Agents.AI.Abstractions` project depends on `Microsoft.Extensions.AI.Abstractions`. The `Microsoft.Agents.AI` project depends on `Microsoft.Extensions.AI` (the full package).

### 1.1 M.E.AI Types Used Throughout the Framework

These types are **not defined in this repo** — they come from Microsoft.Extensions.AI. The C++ port must reimplement these.

#### Core Message Types

| Type | Purpose |
|------|---------|
| `ChatMessage` | A single message in a conversation (role + contents list) |
| `ChatRole` | Value type/struct: `User`, `Assistant`, `System`, `Tool` |
| `ChatResponse` | Non-streaming response from IChatClient (messages + metadata) |
| `ChatResponse<T>` | Generic typed response that deserializes result to `T` |
| `ChatResponseUpdate` | Single streaming chunk from IChatClient |
| `ChatOptions` | Options passed to IChatClient calls (instructions, tools, temp, etc.) |
| `ResponseContinuationToken` | Token for resuming/polling background responses |

#### Content Types (AIContent Hierarchy)

All are subclasses of `AIContent` (abstract base):

| Type | Discriminator | Key Properties |
|------|--------------|----------------|
| `AIContent` | (base) | `RawRepresentation`, `AdditionalProperties` |
| `TextContent` | "text" | `Text: string` |
| `TextReasoningContent` | "reasoning" | `Text: string` |
| `DataContent` | "data" | `Uri: string`, `MediaType: string?` |
| `UriContent` | "uri" | `Uri: Uri`, `MediaType: string` |
| `FunctionCallContent` | "functionCall" | `CallId: string`, `Name: string`, `Arguments: IDictionary<string, object?>` |
| `FunctionResultContent` | "functionResult" | `CallId: string`, `Result: object?` |
| `ErrorContent` | "error" | `Message: string?`, `ErrorCode: string?`, `Details: string?` |
| `UsageContent` | "usage" | `Details: UsageDetails` |
| `HostedFileContent` | "hostedFile" | `FileId: string` |
| `HostedVectorStoreContent` | "hostedVectorStore" | `VectorStoreId: string` |
| `UserInputRequestContent` | — | Used for agent-to-user approval/input requests |
| `UserInputResponseContent` | — | User's response to an input request |

#### Tool / Function Types

| Type | Purpose |
|------|---------|
| `AITool` | Abstract base for tools available to the model |
| `AIFunction` | Abstract function tool with metadata + invocation |
| `DelegatingAIFunction` | Decorator pattern for AIFunction |
| `AIFunctionArguments` | Dictionary-like arguments container for function calls |
| `FunctionInvocationContext` | Context for middleware during function invocation |
| `FunctionInvokingChatClient` | IChatClient decorator that auto-invokes functions |

#### Client / Infrastructure Types

| Type | Purpose |
|------|---------|
| `IChatClient` | Core interface: `GetResponseAsync()`, `GetStreamingResponseAsync()` |
| `ChatClientMetadata` | Metadata about a chat client (provider name, model ID) |
| `AdditionalPropertiesDictionary` | `Dictionary<string, object?>` for extensibility metadata |
| `AdditionalPropertiesDictionary<T>` | Typed variant (e.g., `<long>` for usage counts) |
| `UsageDetails` | Token usage counts (input, output, total, additional) |
| `AIJsonUtilities` | JSON serialization utilities for M.E.AI types |
| `IChatReducer` | Interface for message reduction/summarization strategies |

#### ChatMessage Properties (from M.E.AI)

```
ChatMessage
├── Role: ChatRole                    // User, Assistant, System, Tool
├── AuthorName: string?               // Name of the message author
├── Contents: IList<AIContent>        // Content items (text, images, function calls, etc.)
├── Text: string                      // Convenience: concatenated text from TextContent items
├── RawRepresentation: object?        // Original underlying object
├── AdditionalProperties: AdditionalPropertiesDictionary?
├── MessageId: string?                // Unique ID for message boundary tracking
└── CreatedAt: DateTimeOffset?        // Timestamp
```

#### ChatOptions Properties (from M.E.AI)

```
ChatOptions
├── Instructions: string?             // System instructions / system prompt
├── Tools: IList<AITool>?             // Available tools (functions, etc.)
├── Temperature: float?               // Sampling temperature
├── TopP: float?                      // Nucleus sampling
├── TopK: int?                        // Top-K sampling
├── MaxOutputTokens: int?             // Maximum tokens in response
├── StopSequences: IList<string>?     // Stop sequences
├── FrequencyPenalty: float?          // Frequency penalty
├── PresencePenalty: float?           // Presence penalty
├── ModelId: string?                  // Target model ID
├── ResponseFormat: ChatResponseFormat? // JSON/text response format
├── AdditionalProperties: AdditionalPropertiesDictionary?
├── ConversationId: string?           // Service-side conversation/thread ID
└── Clone(): ChatOptions              // Deep clone method
```

#### ChatRole Values (from M.E.AI)

`ChatRole` is a readonly struct with string `Value`:

| Static Property | String Value |
|----------------|--------------|
| `ChatRole.User` | `"user"` |
| `ChatRole.Assistant` | `"assistant"` |
| `ChatRole.System` | `"system"` |
| `ChatRole.Tool` | `"tool"` |

Custom roles can be created via `new ChatRole("custom_value")`.

#### UsageDetails Properties (from M.E.AI)

```
UsageDetails
├── InputTokenCount: long?
├── OutputTokenCount: long?
├── TotalTokenCount: long?
└── AdditionalCounts: AdditionalPropertiesDictionary<long>?
```

---

## 2. Framework-Defined Types: `Microsoft.Agents.AI.Abstractions`

These types are defined in `dotnet/src/Microsoft.Agents.AI.Abstractions/`.

### 2.1 `AIAgent` (Abstract Base Class)

**File:** `AIAgent.cs` — The foundation of all agents.

```
AIAgent (abstract)
├── Id: string                         // Unique agent ID (GUID by default, overridable via IdCore)
├── IdCore: string? (protected virtual) // Override point for custom ID
├── Name: string? (virtual)            // Human-readable name
├── Description: string? (virtual)     // Agent description
│
├── GetNewSessionAsync() → ValueTask<AgentSession>         // Create new conversation session
├── DeserializeSessionAsync(JsonElement) → ValueTask<AgentSession>
│
├── RunAsync(string, session?, options?, ct) → Task<AgentResponse>
├── RunAsync(ChatMessage, session?, options?, ct) → Task<AgentResponse>
├── RunAsync(IEnumerable<ChatMessage>, session?, options?, ct)
│     └── delegates to RunCoreAsync (abstract)
│
├── RunStreamingAsync(string, ...) → IAsyncEnumerable<AgentResponseUpdate>
├── RunStreamingAsync(ChatMessage, ...) → IAsyncEnumerable<AgentResponseUpdate>
├── RunStreamingAsync(IEnumerable<ChatMessage>, ...)
│     └── delegates to RunCoreStreamingAsync (abstract)
│
├── RunCoreAsync(...) → Task<AgentResponse>                 // ABSTRACT - implement this
├── RunCoreStreamingAsync(...) → IAsyncEnumerable<AgentResponseUpdate>  // ABSTRACT - implement this
│
└── GetService(Type, object?) → object?   // Service locator pattern
```

**Key pattern**: `RunAsync(string)` wraps text as `new ChatMessage(ChatRole.User, message)` before calling core.

### 2.2 `AgentResponse`

**File:** `AgentResponse.cs` — Non-streaming response from an agent.

```
AgentResponse
├── Messages: IList<ChatMessage>       // Response messages (auto-creates empty list)
├── Text: string [JsonIgnore]          // Concatenated text from all messages
├── UserInputRequests: IEnumerable<UserInputRequestContent> [JsonIgnore]
├── AgentId: string?                   // ID of responding agent
├── ResponseId: string?                // Unique response ID
├── ContinuationToken: ResponseContinuationToken?  // For background/polling
├── CreatedAt: DateTimeOffset?         // When response was created
├── Usage: UsageDetails?               // Token usage
├── RawRepresentation: object? [JsonIgnore]   // Original underlying response
├── AdditionalProperties: AdditionalPropertiesDictionary?
│
├── ctor()
├── ctor(ChatMessage)                  // Single message response
├── ctor(ChatResponse)                 // Wrap a ChatResponse (copies all metadata)
├── ctor(IList<ChatMessage>?)
│
├── ToString() → Text
├── ToAgentResponseUpdates() → AgentResponseUpdate[]   // Convert to streaming format
├── Deserialize<T>() → T              // Deserialize response JSON to typed object
├── TryDeserialize<T>(out T?) → bool
└── AsChatResponse() → ChatResponse   // Extension method for conversion
```

### 2.3 `AgentResponse<T>` (Abstract Generic)

**File:** `AgentResponse{T}.cs`

```
AgentResponse<T> : AgentResponse
└── Result: T (abstract)              // Typed result value
```

Concrete implementation: `ChatClientAgentResponse<T>` wraps `ChatResponse<T>`.

### 2.4 `AgentResponseUpdate`

**File:** `AgentResponseUpdate.cs` — Streaming response chunk.

```
AgentResponseUpdate
├── AuthorName: string?                // Author name (null/whitespace → null)
├── Role: ChatRole?                    // Role of the author
├── Text: string [JsonIgnore]          // Concatenated text from TextContent in Contents
├── UserInputRequests: IEnumerable<UserInputRequestContent> [JsonIgnore]
├── Contents: IList<AIContent>         // Content items for this update
├── RawRepresentation: object? [JsonIgnore]
├── AdditionalProperties: AdditionalPropertiesDictionary?
├── AgentId: string?                   // Responding agent ID
├── ResponseId: string?                // Response ID this update belongs to
├── MessageId: string?                 // Message ID for grouping updates into messages
├── CreatedAt: DateTimeOffset?
├── ContinuationToken: ResponseContinuationToken?
│
├── ctor()                              // [JsonConstructor]
├── ctor(ChatRole?, string?)            // Role + text → wraps as TextContent
├── ctor(ChatRole?, IList<AIContent>?)  // Role + contents
├── ctor(ChatResponseUpdate)            // Wrap a ChatResponseUpdate (copies all metadata)
│
├── ToString() → Text
└── AsChatResponseUpdate() → ChatResponseUpdate   // Extension method for conversion
```

### 2.5 `AgentRunOptions`

**File:** `AgentRunOptions.cs` — Options for agent invocations.

```
AgentRunOptions
├── ContinuationToken: ResponseContinuationToken?  // For resume/polling
├── AllowBackgroundResponses: bool?                // Enable background/async responses
└── AdditionalProperties: AdditionalPropertiesDictionary?
```

### 2.6 `AgentSession` (Abstract)

**File:** `AgentSession.cs` — Base class for conversation sessions.

```
AgentSession (abstract)
├── Serialize(JsonSerializerOptions?) → JsonElement   // Serialize state
├── GetService(Type, object?) → object?               // Service locator
└── GetService<TService>(object?) → TService?

Concrete subclasses:
├── InMemoryAgentSession — stores history locally via InMemoryChatHistoryProvider
└── ServiceIdAgentSession — stores only service ID, state is remote
```

### 2.7 `AIContext`

**File:** `AIContext.cs` — Dynamic context for agent invocations.

```
AIContext (sealed)
├── Instructions: string?              // Transient additional instructions
├── Messages: IList<ChatMessage>?      // Permanent additions to history
└── Tools: IList<AITool>?             // Transient additional tools
```

### 2.8 `AIAgentMetadata`

**File:** `AIAgentMetadata.cs`

```
AIAgentMetadata (sealed)
└── ProviderName: string?             // e.g., "azure.ai.agents", for OpenTelemetry
```

### 2.9 Session Types

#### InMemoryAgentSession

```
InMemoryAgentSession : AgentSession (abstract)
├── ChatHistoryProvider: InMemoryChatHistoryProvider
├── Serialize() → JsonElement
└── ctor(InMemoryChatHistoryProvider?)
    ctor(IEnumerable<ChatMessage>)
    ctor(JsonElement, JsonSerializerOptions?, Func<...>?)
```

#### ServiceIdAgentSession

```
ServiceIdAgentSession : AgentSession (abstract)
├── ServiceSessionId: string? (protected)
├── Serialize() → JsonElement
└── ctor()
    ctor(string serviceSessionId)
    ctor(JsonElement, JsonSerializerOptions?)
```

### 2.10 ChatHistoryProvider

**File:** `ChatHistoryProvider.cs`

```
ChatHistoryProvider (abstract)
├── InvokingAsync(InvokingContext, ct) → ValueTask<IEnumerable<ChatMessage>>
├── InvokedAsync(InvokedContext, ct) → ValueTask
├── Serialize(JsonSerializerOptions?) → JsonElement
└── GetService(Type, object?) → object?

InMemoryChatHistoryProvider : ChatHistoryProvider, IList<ChatMessage>
├── ChatReducer: IChatReducer?
├── ReducerTriggerEvent: ChatReducerTriggerEvent
├── [all IList<ChatMessage> members]
└── Serialize() → JsonElement
```

### 2.11 AIContextProvider

**File:** `AIContextProvider.cs`

```
AIContextProvider (abstract)
├── InvokingAsync(InvokingContext, ct) → ValueTask<AIContext>
├── InvokedAsync(InvokedContext, ct) → ValueTask
├── Serialize(JsonSerializerOptions?) → JsonElement
└── GetService(Type, object?) → object?
```

---

## 3. Framework-Defined Types: `Microsoft.Agents.AI`

These types are in `dotnet/src/Microsoft.Agents.AI/`.

### 3.1 `ChatClientAgent`

**File:** `ChatClient/ChatClientAgent.cs` — The primary concrete agent implementation.

```
ChatClientAgent : AIAgent (sealed partial)
├── ChatClient: IChatClient            // Underlying chat client
├── Instructions: string?              // From ChatOptions.Instructions
├── ChatOptions: ChatOptions?          // Internal default options
│
├── ctor(IChatClient, string? instructions, string? name, string? description, IList<AITool>? tools, ILoggerFactory?, IServiceProvider?)
├── ctor(IChatClient, ChatClientAgentOptions?, ILoggerFactory?, IServiceProvider?)
│
├── RunCoreAsync(...) → Task<AgentResponse>          // Calls IChatClient.GetResponseAsync
├── RunCoreStreamingAsync(...) → IAsyncEnumerable     // Calls IChatClient.GetStreamingResponseAsync
│
├── RunAsync<T>(...) → Task<AgentResponse<T>>         // Structured output variant
│
└── GetService(Type, object?) → object?
    Returns: AIAgentMetadata, IChatClient, ChatOptions, ChatClientAgentOptions
```

### 3.2 `ChatClientAgentOptions`

**File:** `ChatClient/ChatClientAgentOptions.cs`

```
ChatClientAgentOptions (sealed)
├── Id: string?
├── Name: string?
├── Description: string?
├── ChatOptions: ChatOptions?
├── ChatHistoryProviderFactory: Func<ChatHistoryProviderFactoryContext, CancellationToken, ValueTask<ChatHistoryProvider>>?
├── AIContextProviderFactory: Func<AIContextProviderFactoryContext, CancellationToken, ValueTask<AIContextProvider>>?
├── UseProvidedChatClientAsIs: bool
└── Clone() → ChatClientAgentOptions
```

### 3.3 `ChatClientAgentRunOptions`

**File:** `ChatClient/ChatClientAgentRunOptions.cs`

```
ChatClientAgentRunOptions : AgentRunOptions (sealed)
├── ChatOptions: ChatOptions?               // Per-invocation chat options
└── ChatClientFactory: Func<IChatClient, IChatClient>?  // Per-request client decorator
```

### 3.4 `ChatClientAgentResponse<T>`

**File:** `ChatClient/ChatClientAgentRunResponse{T}.cs`

```
ChatClientAgentResponse<T> : AgentResponse<T> (sealed)
├── Result: T (override)   // Delegates to ChatResponse<T>.Result
└── ctor(ChatResponse<T>)
```

### 3.5 `DelegatingAIAgent`

**File in Abstractions:** `DelegatingAIAgent.cs` — Decorator pattern base.

```
DelegatingAIAgent : AIAgent (abstract)
├── InnerAgent: AIAgent (protected)
├── All methods delegate to InnerAgent by default
└── ctor(AIAgent innerAgent)
```

### 3.6 `FunctionInvocationDelegatingAgent`

**File:** `FunctionInvocationDelegatingAgent.cs` — Adds function invocation middleware.

```
FunctionInvocationDelegatingAgent : DelegatingAIAgent (internal sealed)
├── Wraps function calls through middleware pipeline
├── Uses ChatClientAgentRunOptions.ChatClientFactory to inject middleware
└── MiddlewareEnabledFunction : DelegatingAIFunction (private sealed)
```

---

## 4. Conversion Between Agent and M.E.AI Types

**File:** `AgentResponseExtensions.cs`

| Method | Direction | Notes |
|--------|-----------|-------|
| `AgentResponse.AsChatResponse()` | Agent → M.E.AI | Returns RawRepresentation if already ChatResponse |
| `AgentResponseUpdate.AsChatResponseUpdate()` | Agent → M.E.AI | Returns RawRepresentation if already ChatResponseUpdate |
| `IAsyncEnumerable<AgentResponseUpdate>.AsChatResponseUpdatesAsync()` | Streaming Agent → M.E.AI | Wraps each update |
| `IEnumerable<AgentResponseUpdate>.ToAgentResponse()` | Streaming → Non-streaming | Uses MessageId for grouping |
| `IAsyncEnumerable<AgentResponseUpdate>.ToAgentResponseAsync()` | Async streaming → Non-streaming | Same as above, async |
| `AgentResponse.ToAgentResponseUpdates()` | Non-streaming → Streaming | Each message → update |
| `new AgentResponse(ChatResponse)` | M.E.AI → Agent | Constructor copies all metadata |
| `new AgentResponseUpdate(ChatResponseUpdate)` | M.E.AI → Agent | Constructor copies all metadata |

---

## 5. Streaming Model

### Non-streaming Flow
```
IChatClient.GetResponseAsync(messages, chatOptions)
  → ChatResponse
  → new AgentResponse(chatResponse)
  → AgentResponse { Messages, Usage, ResponseId, ... }
```

### Streaming Flow
```
IChatClient.GetStreamingResponseAsync(messages, chatOptions)
  → IAsyncEnumerable<ChatResponseUpdate>
  → foreach update:
       yield new AgentResponseUpdate(chatResponseUpdate) { AgentId = ... }
  → Consumer accumulates updates
  → Optionally: .ToAgentResponseAsync() to reconstitute full AgentResponse
```

### Key Streaming Semantics
- `MessageId` groups updates into logical messages
- Multiple updates with the same `MessageId` = parts of the same message
- `ResponseId` groups all updates in a single response
- Contiguous `TextContent` items may be coalesced when aggregated
- `ContinuationToken` on each update enables stream resumption
- Last update has `ContinuationToken = null` to signal completion

---

## 6. Serialization Infrastructure

### JSON Serialization Chain

```
AIJsonUtilities.DefaultOptions          ← M.E.AI types (ChatMessage, AIContent, etc.)
  ↓ chained into
AgentAbstractionsJsonUtilities.DefaultOptions  ← Agent abstraction types (AgentResponse, etc.)
  ↓ chained into
AgentJsonUtilities.DefaultOptions       ← Agent implementation types (ChatClientAgentSession, etc.)
```

### Source-Generated Serializable Types

From `AgentAbstractionsJsonUtilities`:
- `AgentRunOptions`
- `AgentResponse`
- `AgentResponse[]`
- `AgentResponseUpdate`
- `AgentResponseUpdate[]`
- `ServiceIdAgentSession.ServiceIdAgentSessionState`
- `InMemoryAgentSession.InMemoryAgentSessionState`
- `InMemoryChatHistoryProvider.State`

### JSON Settings
- Defaults: `JsonSerializerDefaults.Web` (camelCase, case-insensitive)
- `DefaultIgnoreCondition = JsonIgnoreCondition.WhenWritingNull`
- `NumberHandling = JsonNumberHandling.AllowReadingFromString`
- `UseStringEnumConverter = true`
- `Encoder = JavaScriptEncoder.UnsafeRelaxedJsonEscaping`

### DurableTask State Serialization (Polymorphic)

The DurableTask project defines its own serializable mirror types for AIContent:

```
DurableAgentStateContent [JsonPolymorphic, $type discriminator]
├── DurableAgentStateTextContent ("text")         → TextContent
├── DurableAgentStateTextReasoningContent ("reasoning") → TextReasoningContent
├── DurableAgentStateDataContent ("data")         → DataContent
├── DurableAgentStateUriContent ("uri")           → UriContent
├── DurableAgentStateFunctionCallContent ("functionCall") → FunctionCallContent
├── DurableAgentStateFunctionResultContent ("functionResult") → FunctionResultContent
├── DurableAgentStateErrorContent ("error")       → ErrorContent
├── DurableAgentStateUsageContent ("usage")       → UsageContent
├── DurableAgentStateHostedFileContent ("hostedFile") → HostedFileContent
├── DurableAgentStateHostedVectorStoreContent ("hostedVectorStore") → HostedVectorStoreContent
└── DurableAgentStateUnknownContent ("unknown")   → fallback via AIJsonUtilities
```

Each has `FromXxxContent(M.E.AI type)` and `ToAIContent()` round-trip methods.

`DurableAgentStateMessage` mirrors `ChatMessage`:
```
DurableAgentStateMessage
├── AuthorName: string?
├── CreatedAt: DateTimeOffset?
├── Contents: IReadOnlyList<DurableAgentStateContent>
├── Role: string                    // Serialized as string, not ChatRole
└── FromChatMessage() / ToChatMessage()  // Round-trip conversion
```

`DurableAgentStateUsage` mirrors `UsageDetails`:
```
DurableAgentStateUsage
├── InputTokenCount: long?
├── OutputTokenCount: long?
├── TotalTokenCount: long?
└── FromUsage() / ToUsageDetails()
```

---

## 7. Complete Type Dependency Graph for C++ Port

### Tier 0: Must implement first (no framework dependencies)
```
ChatRole (struct, string Value)
AdditionalPropertiesDictionary (Dictionary<string, object?>)
AdditionalPropertiesDictionary<T>
UsageDetails
ResponseContinuationToken (abstract, ToBytes() method)
```

### Tier 1: Core content types (depend on Tier 0)
```
AIContent (abstract base)
├── TextContent
├── TextReasoningContent
├── DataContent
├── UriContent
├── FunctionCallContent
├── FunctionResultContent
├── ErrorContent
├── UsageContent
├── HostedFileContent
├── HostedVectorStoreContent
├── UserInputRequestContent
└── UserInputResponseContent
```

### Tier 2: Message types (depend on Tier 0 + 1)
```
ChatMessage (role + contents list)
ChatResponse (messages + metadata)
ChatResponse<T>
ChatResponseUpdate (streaming chunk)
ChatOptions (instructions, tools, temp, etc.)
```

### Tier 3: Tool types (depend on Tier 0 + 1)
```
AITool (abstract)
AIFunction (abstract, metadata + invoke)
DelegatingAIFunction (decorator)
AIFunctionArguments
FunctionInvocationContext
```

### Tier 4: Client interface (depend on Tier 2 + 3)
```
IChatClient
├── GetResponseAsync(messages, options, ct) → ChatResponse
├── GetStreamingResponseAsync(messages, options, ct) → IAsyncEnumerable<ChatResponseUpdate>
├── GetService(Type, object?) → object?
└── Dispose()
```

### Tier 5: Agent framework types (depend on all above)
```
AgentSession (abstract)
├── InMemoryAgentSession
└── ServiceIdAgentSession

ChatHistoryProvider (abstract)
└── InMemoryChatHistoryProvider

AIContextProvider (abstract)
AIContext

AgentRunOptions
AgentResponse
AgentResponse<T>
AgentResponseUpdate

AIAgent (abstract)
├── DelegatingAIAgent
├── ChatClientAgent
└── FunctionInvocationDelegatingAgent

ChatClientAgentOptions
ChatClientAgentRunOptions
AIAgentMetadata
```

---

## 8. Key Patterns for C++ Port

### 8.1 Content Polymorphism
Content types use M.E.AI's `AIContent` base with `[JsonPolymorphic]` for serialization. In C++, use a variant or polymorphic base with type discriminator.

### 8.2 Message as Content Container
`ChatMessage` is essentially `{ Role, IList<AIContent> }`. A single message can contain mixed content: text + function calls + images, etc.

### 8.3 Streaming via IAsyncEnumerable
The streaming model uses `IAsyncEnumerable<AgentResponseUpdate>`. C++ equivalent: coroutine generator, callback-based stream, or `std::generator<T>` (C++23).

### 8.4 Decorator/Middleware Pattern
`DelegatingAIAgent` wraps agents. `DelegatingAIFunction` wraps functions. Both use decorator pattern. `FunctionInvokingChatClient` wraps `IChatClient` for auto-invocation.

### 8.5 Service Locator Pattern
`GetService(Type)` on AIAgent, AgentSession, and providers allows runtime type discovery. C++ equivalent: `std::any`-based registry or template getters.

### 8.6 Continuation Token for Background Operations
`ResponseContinuationToken` is serialized to bytes for transport. Used for polling (non-streaming) and resumption (streaming).

### 8.7 Agent ↔ M.E.AI Conversion
`AgentResponse` / `AgentResponseUpdate` are thin wrappers over M.E.AI's `ChatResponse` / `ChatResponseUpdate`, with added `AgentId` and bidirectional conversion methods. The framework maintains compatibility with both its own types and M.E.AI types.
