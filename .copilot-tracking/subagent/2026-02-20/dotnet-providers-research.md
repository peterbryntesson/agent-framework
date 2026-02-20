# .NET AI Provider Implementations Research

> Generated: 2026-02-20
> Scope: All provider-specific projects under `dotnet/src/`

## Table of Contents

- [Core Abstractions Summary](#core-abstractions-summary)
- [1. Microsoft.Agents.AI.OpenAI](#1-microsoftagentsaiopenai)
- [2. Microsoft.Agents.AI.AzureAI](#2-microsoftagentsaiazureai)
- [3. Microsoft.Agents.AI.AzureAI.Persistent](#3-microsoftagentsaiazureaipersistent)
- [4. Microsoft.Agents.AI.Anthropic](#4-microsoftagentsaianthropic)
- [5. Microsoft.Agents.AI.CopilotStudio](#5-microsoftagentsaicopilotstudio)
- [6. Microsoft.Agents.AI.GitHub.Copilot](#6-microsoftagentsaigithubcopilot)
- [Cross-Cutting Analysis](#cross-cutting-analysis)

---

## Core Abstractions Summary

Before diving into providers, the key base types they all integrate with:

### `AIAgent` (abstract base) — [AIAgent.cs](dotnet/src/Microsoft.Agents.AI.Abstractions/AIAgent.cs)

- `Id` property (auto-generated GUID or override via `IdCore`)
- `Name`, `Description` virtual properties
- `RunAsync(messages, session?, options?, ct)` → `AgentResponse` (calls `RunCoreAsync`)
- `RunStreamingAsync(messages, session?, options?, ct)` → `IAsyncEnumerable<AgentResponseUpdate>` (calls `RunCoreStreamingAsync`)
- `GetNewSessionAsync()` → `AgentSession`
- `DeserializeSessionAsync(JsonElement)` → `AgentSession`
- `GetService(Type, key?)` — service locator pattern

### `DelegatingAIAgent` (decorator base) — [DelegatingAIAgent.cs](dotnet/src/Microsoft.Agents.AI.Abstractions/DelegatingAIAgent.cs)

- Extends `AIAgent`, wraps an `InnerAgent`
- Transparent pass-through of all operations
- Overridable for custom middleware behavior

### `ChatClientAgent` (sealed) — [ChatClientAgent.cs](dotnet/src/Microsoft.Agents.AI/ChatClient/ChatClientAgent.cs)

- Extends `AIAgent`, delegates to `IChatClient` (from `Microsoft.Extensions.AI`)
- Accepts `ChatClientAgentOptions` (instructions, tools, chat history provider factory, context provider factory)
- Manages `InMemoryAgentSession` by default
- Auto-wraps `IChatClient` with `FunctionInvokingChatClient` if tools present and no existing function invoker
- **Most providers create `ChatClientAgent` instances** via extension methods converting their SDK clients → `IChatClient` → `ChatClientAgent`

### `ServiceIdAgentSession` (abstract) — [ServiceIdAgentSession.cs](dotnet/src/Microsoft.Agents.AI.Abstractions/ServiceIdAgentSession.cs)

- For agents where state is stored remotely; only stores a `ServiceSessionId` reference locally
- Used by `CopilotStudioAgentSession`

---

## 1. Microsoft.Agents.AI.OpenAI

### Project Configuration

- **csproj**: [Microsoft.Agents.AI.OpenAI.csproj](dotnet/src/Microsoft.Agents.AI.OpenAI/Microsoft.Agents.AI.OpenAI.csproj)
- **NuGet Dependencies**: `Microsoft.Extensions.AI.OpenAI`
- **Project References**: `Microsoft.Agents.AI`
- **Integration Pattern**: Extension methods on OpenAI SDK types → `IChatClient` → `ChatClientAgent`
- **Namespace Strategy**: Extension methods placed in the OpenAI SDK namespaces (`OpenAI.Chat`, `OpenAI.Responses`, `OpenAI.Assistants`) so they appear naturally on those types

### Files (8 total)

#### [OpenAIChatClientExtensions.cs](dotnet/src/Microsoft.Agents.AI.OpenAI/Extensions/OpenAIChatClientExtensions.cs) (L1-96)

- **Class**: `OpenAIChatClientExtensions` (static)
- **Namespace**: `OpenAI.Chat`
- **Methods**:
  - `AsAIAgent(this ChatClient, instructions?, name?, description?, tools?, clientFactory?, loggerFactory?, services?)` → `ChatClientAgent`
  - `AsAIAgent(this ChatClient, ChatClientAgentOptions, clientFactory?, loggerFactory?, services?)` → `ChatClientAgent`
- **Pattern**: `ChatClient.AsIChatClient()` → optional `clientFactory` transform → `new ChatClientAgent(chatClient, options, loggerFactory, services)`
- **Streaming**: Inherited from `ChatClientAgent` which uses `IChatClient.GetStreamingResponseAsync()`
- **Tool/Function Calling**: Tools passed via `ChatClientAgentOptions.ChatOptions.Tools`; `ChatClientAgent` auto-wraps with `FunctionInvokingChatClient`
- **Authentication**: Handled by consumer when constructing `OpenAI.Chat.ChatClient` (API key or Azure credential)

#### [OpenAIResponseClientExtensions.cs](dotnet/src/Microsoft.Agents.AI.OpenAI/Extensions/OpenAIResponseClientExtensions.cs) (L1-100)

- **Class**: `OpenAIResponseClientExtensions` (static)
- **Namespace**: `OpenAI.Responses`
- **Methods**:
  - `AsAIAgent(this ResponsesClient, instructions?, name?, description?, tools?, clientFactory?, loggerFactory?, services?)` → `ChatClientAgent`
  - `AsAIAgent(this ResponsesClient, ChatClientAgentOptions, clientFactory?, loggerFactory?, services?)` → `ChatClientAgent`
- **Pattern**: Same as ChatClient — `ResponsesClient.AsIChatClient()` → `ChatClientAgent`
- **Streaming**: Same as ChatClient path

#### [OpenAIAssistantClientExtensions.cs](dotnet/src/Microsoft.Agents.AI.OpenAI/Extensions/OpenAIAssistantClientExtensions.cs) (L1-431)

- **Class**: `OpenAIAssistantClientExtensions` (static)
- **Namespace**: `OpenAI.Assistants`
- **Deprecated**: All methods marked `[Obsolete("The Assistants API has been deprecated. Please use the Responses API instead.")]`
- **Methods**:
  - `AsAIAgent(this AssistantClient, ClientResult<Assistant>, chatOptions?, clientFactory?, services?)` → `ChatClientAgent`
  - `AsAIAgent(this AssistantClient, Assistant, chatOptions?, clientFactory?, services?)` → `ChatClientAgent`
  - `AsAIAgent(this AssistantClient, ClientResult<Assistant>, ChatClientAgentOptions, clientFactory?, services?)` → `ChatClientAgent`
  - `AsAIAgent(this AssistantClient, Assistant, ChatClientAgentOptions, clientFactory?, services?)` → `ChatClientAgent`
  - `GetAIAgentAsync(this AssistantClient, agentId, chatOptions?, clientFactory?, services?, ct)` → `Task<ChatClientAgent>`
  - `GetAIAgentAsync(this AssistantClient, agentId, ChatClientAgentOptions, clientFactory?, services?, ct)` → `Task<ChatClientAgent>`
  - `CreateAIAgentAsync(this AssistantClient, model, instructions?, name?, description?, tools?, clientFactory?, loggerFactory?, services?, ct)` → `Task<ChatClientAgent>`
  - `CreateAIAgentAsync(this AssistantClient, model, ChatClientAgentOptions, clientFactory?, loggerFactory?, services?, ct)` → `Task<ChatClientAgent>`
- **Pattern**: `AssistantClient.AsIChatClient(assistantId)` → `ChatClientAgent`
- **Tool Handling**: `ConvertAIToolsToToolDefinitions()` — converts `AITool` list to OpenAI `ToolDefinition` list:
  - `HostedCodeInterpreterTool` → `CodeInterpreterToolDefinition` + `ToolResources.CodeInterpreter`
  - `HostedFileSearchTool` → `FileSearchToolDefinition` + `ToolResources.FileSearch`
  - Default → kept as `AIFunction` for local invocation

#### [AIAgentWithOpenAIExtensions.cs](dotnet/src/Microsoft.Agents.AI.OpenAI/Extensions/AIAgentWithOpenAIExtensions.cs) (L1-124)

- **Class**: `AIAgentWithOpenAIExtensions` (static)
- **Namespace**: `Microsoft.Agents.AI`
- **Purpose**: Extensions on `AIAgent` to accept/return **native OpenAI types**
- **Methods**:
  - `RunAsync(this AIAgent, IEnumerable<ChatMessage>, session?, options?, ct)` → `Task<ChatCompletion>` — Convert OpenAI `ChatMessage` → MEAI messages → run → extract `ChatCompletion`
  - `RunStreamingAsync(this AIAgent, IEnumerable<ChatMessage>, session?, options?, ct)` → `AsyncCollectionResult<StreamingChatCompletionUpdate>` — returns `AsyncStreamingChatCompletionUpdateCollectionResult`
  - `RunAsync(this AIAgent, IEnumerable<ResponseItem>, session?, options?, ct)` → `Task<ResponseResult>` — Convert OpenAI `ResponseItem` → MEAI messages → run → extract `ResponseResult`
  - `RunStreamingAsync(this AIAgent, IEnumerable<ResponseItem>, session?, options?, ct)` → `AsyncCollectionResult<StreamingResponseUpdate>` — returns `AsyncStreamingResponseUpdateCollectionResult`

#### [AgentResponseExtensions.cs](dotnet/src/Microsoft.Agents.AI.OpenAI/Extensions/AgentResponseExtensions.cs) (L1-49)

- **Class**: `AgentResponseExtensions` (static)
- **Namespace**: `Microsoft.Agents.AI`
- **Methods**:
  - `AsOpenAIChatCompletion(this AgentResponse)` → `ChatCompletion` — extracts from `RawRepresentation` or converts via `AsChatResponse()`
  - `AsOpenAIResponse(this AgentResponse)` → `ResponseResult` — extracts from `RawRepresentation` or converts via `AsChatResponse()`

#### [AsyncStreamingChatCompletionUpdateCollectionResult.cs](dotnet/src/Microsoft.Agents.AI.OpenAI/ChatClient/AsyncStreamingChatCompletionUpdateCollectionResult.cs) (L1-33)

- **Class**: `AsyncStreamingChatCompletionUpdateCollectionResult` (internal, sealed)
- **Extends**: `AsyncCollectionResult<StreamingChatCompletionUpdate>`
- **Purpose**: Wraps `IAsyncEnumerable<AgentResponseUpdate>` as OpenAI native streaming type
- **Key Method**: `GetValuesFromPageAsync` — converts AgentResponseUpdate stream → `StreamingChatCompletionUpdate` using `AsChatResponseUpdatesAsync().AsOpenAIStreamingChatCompletionUpdatesAsync()`

#### [AsyncStreamingResponseUpdateCollectionResult.cs](dotnet/src/Microsoft.Agents.AI.OpenAI/ChatClient/AsyncStreamingResponseUpdateCollectionResult.cs) (L1-52)

- **Class**: `AsyncStreamingResponseUpdateCollectionResult` (internal, sealed)
- **Extends**: `AsyncCollectionResult<StreamingResponseUpdate>`
- **Purpose**: Wraps `IAsyncEnumerable<AgentResponseUpdate>` as OpenAI Responses API streaming type
- **Key Method**: `GetValuesFromPageAsync` — extracts `StreamingResponseUpdate` from `RawRepresentation` or nested `ChatResponseUpdate.RawRepresentation`

#### [StreamingUpdatePipelineResponse.cs](dotnet/src/Microsoft.Agents.AI.OpenAI/ChatClient/StreamingUpdatePipelineResponse.cs) (L1-81)

- **Class**: `StreamingUpdatePipelineResponse` (internal, sealed)
- **Extends**: `PipelineResponse` (from `System.ClientModel.Primitives`)
- **Purpose**: Stub `PipelineResponse` for streaming (status 200, empty headers/content, no buffering support)
- **Contains**: `EmptyPipelineResponseHeaders` inner class

---

## 2. Microsoft.Agents.AI.AzureAI

### Project Configuration

- **csproj**: [Microsoft.Agents.AI.AzureAI.csproj](dotnet/src/Microsoft.Agents.AI.AzureAI/Microsoft.Agents.AI.AzureAI.csproj)
- **Title**: "Microsoft Agent Framework for Foundry Agents"
- **NuGet Dependencies**: `Azure.AI.Projects`, `Azure.AI.Projects.OpenAI`, `Microsoft.Extensions.AI`, `Microsoft.Extensions.AI.OpenAI`, `OpenAI`
- **Project References**: `Microsoft.Agents.AI`
- **Integration Pattern**: Extension methods on `AIProjectClient` → custom `AzureAIProjectChatClient` (which extends `DelegatingChatClient`) → `ChatClientAgent`

### Files (3 total)

#### [AzureAIProjectChatClient.cs](dotnet/src/Microsoft.Agents.AI.AzureAI/AzureAIProjectChatClient.cs) (L1-155)

- **Class**: `AzureAIProjectChatClient` (internal, sealed)
- **Extends**: `DelegatingChatClient` (from `Microsoft.Extensions.AI`)
- **Constructors**:
  - `(AIProjectClient, AgentReference, defaultModelId?, chatOptions?)` — uses `aiProjectClient.GetProjectOpenAIClient().GetProjectResponsesClientForAgent(agentReference).AsIChatClient()` as inner client
  - `(AIProjectClient, AgentRecord, chatOptions?)` — delegates to AgentVersion constructor using `agentRecord.Versions.Latest`
  - `(AIProjectClient, AgentVersion, chatOptions?)` — extracts model from `PromptAgentDefinition`
- **Key Override**: `GetAgentEnabledChatOptions(options)`:
  - Clones base `_chatOptions`, nulls out agent-defined fields (instructions, tools, temperature, etc.)
  - Sets `ConversationId` from per-request options or client-level
  - Wraps `RawRepresentationFactory` to set `CreateResponseOptions.Agent` to the agent reference and patches out `$.model`
- **Streaming**: `GetStreamingResponseAsync` overridden to pass agent-enabled options
- **Service Resolution**: `GetService()` exposes `ChatClientMetadata`, `AIProjectClient`, `AgentVersion`, `AgentRecord`, `AgentReference`
- **Metadata**: Provider tagged as `"azure.ai.agents"`

#### [AzureAIProjectChatClientExtensions.cs](dotnet/src/Microsoft.Agents.AI.AzureAI/AzureAIProjectChatClientExtensions.cs) (L1-770)

- **Class**: `AzureAIProjectChatClientExtensions` (static, partial)
- **Namespace**: `Azure.AI.Projects`
- **Public Methods (16 total)**:
  - `AsAIAgent(this AIProjectClient, AgentReference, tools?, clientFactory?, services?)` → `ChatClientAgent`
  - `AsAIAgent(this AIProjectClient, AgentRecord, tools?, clientFactory?, services?)` → `ChatClientAgent`
  - `AsAIAgent(this AIProjectClient, AgentVersion, tools?, clientFactory?, services?)` → `ChatClientAgent`
  - `GetAIAgentAsync(this AIProjectClient, name, tools?, clientFactory?, services?, ct)` → `Task<ChatClientAgent>`
  - `GetAIAgentAsync(this AIProjectClient, ChatClientAgentOptions, clientFactory?, services?, ct)` → `Task<ChatClientAgent>`
  - `CreateAIAgentAsync(this AIProjectClient, name, model, instructions, description?, tools?, clientFactory?, services?, ct)` → `Task<ChatClientAgent>`
  - `CreateAIAgentAsync(this AIProjectClient, model, ChatClientAgentOptions, clientFactory?, services?, ct)` → `Task<ChatClientAgent>`
  - `CreateAIAgentAsync(this AIProjectClient, name, AgentVersionCreationOptions, clientFactory?, ct)` → `Task<ChatClientAgent>`
- **Private Helper Methods**:
  - `GetAgentRecordByNameAsync` — protocol-level GET with user-agent header
  - `CreateAgentVersionWithProtocolAsync` — protocol-level create with `ModelReaderWriter`
  - Multiple `AsChatClientAgent` overloads — construct `AzureAIProjectChatClient` + optional `clientFactory` → `ChatClientAgent`
  - `CreateChatClientAgentOptions(AgentVersion, ChatOptions, requireInvocableTools)`:
    - Extracts tools from `PromptAgentDefinition.Tools`
    - Matches provided `AIFunction` tools against definition `FunctionTool`s by name
    - Validates all required tools are provided
    - Throws `InvalidOperationException` for missing tools
  - `ApplyToolsToAgentDefinition(AgentDefinition, tools)` — sets tools on `PromptAgentDefinition`, validates `AIFunction` instances are invocable (not just declarations)
  - `ToOpenAIResponseTextFormat` — converts `ChatResponseFormat` → `ResponseTextFormat`
  - `StrictSchemaTransformCache` — JSON schema transformer for OpenAI structured output restrictions (removes unsupported properties, moves them to description)
- **Agent Name Validation**: Regex `^[a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?$` (1-63 chars, alphanumeric + hyphens)
- **Tool Handling**: Rich conversion — `FunctionTool`, hosted tools (`ResponseTool.AsAITool()`), validation that function tools are invocable `AIFunctions` not just declarations
- **Inner Class**: `NoOpChatClient` — dummy `IChatClient` used to safely invoke `RawRepresentationFactory`
- **Inner Class**: `AgentClientJsonContext` — source-generated JSON serializer context

#### [RequestOptionsExtensions.cs](dotnet/src/Microsoft.Agents.AI.AzureAI/RequestOptionsExtensions.cs) (L1-64)

- **Class**: `RequestOptionsExtensions` (internal, static)
- **Methods**:
  - `ToRequestOptions(this CancellationToken, streaming)` → `RequestOptions` — adds `MeaiUserAgentPolicy` per-call
- **Inner Class**: `MeaiUserAgentPolicy` (private, sealed) — `PipelinePolicy` that adds `User-Agent: MEAI/{version}` header
- **Authentication**: User-agent header injection; actual authentication is handled by `AIProjectClient` (Azure credential-based)

---

## 3. Microsoft.Agents.AI.AzureAI.Persistent

### Project Configuration

- **csproj**: [Microsoft.Agents.AI.AzureAI.Persistent.csproj](dotnet/src/Microsoft.Agents.AI.AzureAI.Persistent/Microsoft.Agents.AI.AzureAI.Persistent.csproj)
- **NuGet Dependencies**: `Azure.AI.Agents.Persistent`, `Microsoft.Extensions.AI`
- **Project References**: `Microsoft.Agents.AI`
- **Integration Pattern**: Extension methods on `PersistentAgentsClient` → `IChatClient` (via SDK's `AsIChatClient`) → `ChatClientAgent`

### Files (1 total)

#### [PersistentAgentsClientExtensions.cs](dotnet/src/Microsoft.Agents.AI.AzureAI.Persistent/PersistentAgentsClientExtensions.cs) (L1-430)

- **Class**: `PersistentAgentsClientExtensions` (static)
- **Namespace**: `Azure.AI.Agents.Persistent`
- **Public Methods (8 total)**:
  - `AsAIAgent(this PersistentAgentsClient, Response<PersistentAgent>, chatOptions?, clientFactory?, services?)` → `ChatClientAgent`
  - `AsAIAgent(this PersistentAgentsClient, PersistentAgent, chatOptions?, clientFactory?, services?)` → `ChatClientAgent`
  - `AsAIAgent(this PersistentAgentsClient, Response<PersistentAgent>, ChatClientAgentOptions, clientFactory?, services?)` → `ChatClientAgent`
  - `AsAIAgent(this PersistentAgentsClient, PersistentAgent, ChatClientAgentOptions, clientFactory?, services?)` → `ChatClientAgent`
  - `GetAIAgentAsync(this PersistentAgentsClient, agentId, chatOptions?, clientFactory?, services?, ct)` → `Task<ChatClientAgent>`
  - `GetAIAgentAsync(this PersistentAgentsClient, agentId, ChatClientAgentOptions, clientFactory?, services?, ct)` → `Task<ChatClientAgent>`
  - `CreateAIAgentAsync(this PersistentAgentsClient, model, name?, description?, instructions?, tools?, toolResources?, temperature?, topP?, responseFormat?, metadata?, clientFactory?, services?, ct)` → `Task<ChatClientAgent>`
  - `CreateAIAgentAsync(this PersistentAgentsClient, model, ChatClientAgentOptions, clientFactory?, services?, ct)` → `Task<ChatClientAgent>`
- **Pattern**: `PersistentAgentsClient.AsIChatClient(agentId)` → optional `clientFactory` → `new ChatClientAgent(chatClient, agentOptions, services)`
- **Tool Handling**: `ConvertAIToolsToToolDefinitions()` converts:
  - `HostedCodeInterpreterTool` → `CodeInterpreterToolDefinition` + `ToolResources.CodeInterpreter.FileIds`
  - `HostedFileSearchTool` → `FileSearchToolDefinition` + `ToolResources.FileSearch.VectorStoreIds`
  - `HostedWebSearchTool` with `connectionId` → `BingGroundingToolDefinition`
  - Default → local `AIFunction` tools
- **Agent Retrieval**: Uses `PersistentAgentsClient.Administration.GetAgentAsync(agentId, ct)` and `Administration.CreateAgentAsync(...)` 
- **Session**: No custom session type; uses default `InMemoryAgentSession` from `ChatClientAgent`
- **Authentication**: Handled by `PersistentAgentsClient` construction (Azure credential)

---

## 4. Microsoft.Agents.AI.Anthropic

### Project Configuration

- **csproj**: [Microsoft.Agents.AI.Anthropic.csproj](dotnet/src/Microsoft.Agents.AI.Anthropic/Microsoft.Agents.AI.Anthropic.csproj)
- **NuGet Dependencies**: `Microsoft.Extensions.AI`, `Anthropic` (NuGet package)
- **Project References**: `Microsoft.Agents.AI`
- **Integration Pattern**: Extension methods on Anthropic SDK types → `IChatClient` (via Anthropic SDK's `AsIChatClient`) → `ChatClientAgent`

### Files (3 total)

#### [AnthropicClientExtensions.cs](dotnet/src/Microsoft.Agents.AI.Anthropic/AnthropicClientExtensions.cs) (L1-99)

- **Class**: `AnthropicClientExtensions` (static)
- **Namespace**: `Anthropic`
- **Static Property**: `DefaultMaxTokens` = 4096 (settable)
- **Methods**:
  - `AsAIAgent(this IAnthropicClient, model, instructions?, name?, description?, tools?, defaultMaxTokens?, clientFactory?, loggerFactory?, services?)` → `ChatClientAgent`
  - `AsAIAgent(this IAnthropicClient, ChatClientAgentOptions, clientFactory?, loggerFactory?, services?)` → `ChatClientAgent`
- **Pattern**: `client.AsIChatClient(model, defaultMaxTokens)` or `client.AsIChatClient()` → optional `clientFactory` → `new ChatClientAgent(chatClient, options, loggerFactory, services)`
- **Tool Support**: Tools passed via `ChatClientAgentOptions.ChatOptions.Tools`
- **Authentication**: Handled by consumer when constructing `IAnthropicClient` (API key)

#### [AnthropicBetaServiceExtensions.cs](dotnet/src/Microsoft.Agents.AI.Anthropic/AnthropicBetaServiceExtensions.cs) (L1-104)

- **Class**: `AnthropicBetaServiceExtensions` (static)
- **Namespace**: `Anthropic.Services`
- **Static Property**: `DefaultMaxTokens` = 4096 (settable)
- **Methods**:
  - `AsAIAgent(this IBetaService, model, instructions?, name?, description?, tools?, defaultMaxTokens?, clientFactory?, loggerFactory?, services?)` → `ChatClientAgent`
  - `AsAIAgent(this IBetaService, ChatClientAgentOptions, clientFactory?, loggerFactory?, services?)` → `ChatClientAgent`
- **Pattern**: Same as `IAnthropicClient` — `betaService.AsIChatClient(model, maxTokens)` → `ChatClientAgent`

#### [AnthropicClientJsonContext.cs](dotnet/src/Microsoft.Agents.AI.Anthropic/AnthropicClientJsonContext.cs) (L1-14)

- **Class**: `AnthropicClientJsonContext` (internal, sealed, partial)
- **Purpose**: Source-generated JSON serializer context for `JsonElement`, `string`, `Dictionary<string, object?>`

---

## 5. Microsoft.Agents.AI.CopilotStudio

### Project Configuration

- **csproj**: [Microsoft.Agents.AI.CopilotStudio.csproj](dotnet/src/Microsoft.Agents.AI.CopilotStudio/Microsoft.Agents.AI.CopilotStudio.csproj)
- **NuGet Dependencies**: `Microsoft.Agents.CopilotStudio.Client`
- **Project References**: `Microsoft.Agents.AI.Abstractions` (**NOT** `Microsoft.Agents.AI`)
- **Integration Pattern**: **Direct `AIAgent` subclass** — does NOT use `IChatClient` or `ChatClientAgent`

### Files (3 total)

#### [CopilotStudioAgent.cs](dotnet/src/Microsoft.Agents.AI.CopilotStudio/CopilotStudioAgent.cs) (L1-151)

- **Class**: `CopilotStudioAgent` (public, non-sealed)
- **Extends**: `AIAgent` (directly)
- **Constructor**: `CopilotStudioAgent(CopilotClient, loggerFactory?)`
- **Properties**:
  - `Client` → `CopilotClient` (public)
- **Key Methods**:
  - `GetNewSessionAsync()` → `new CopilotStudioAgentSession()`
  - `GetNewSessionAsync(conversationId)` → session with existing conversation
  - `DeserializeSessionAsync(JsonElement, options?, ct)` → `new CopilotStudioAgentSession(serializedSession, options)`
  - `RunCoreAsync(messages, session?, options?, ct)`:
    - Gets/creates `CopilotStudioAgentSession`
    - Starts new conversation if `ConversationId` is null via `StartNewConversationAsync()`
    - Joins all message texts with `\n`
    - Calls `ActivityProcessor.ProcessActivityAsync(Client.AskQuestionAsync(...), streaming: false, logger)`
    - Returns `AgentResponse` with list of response `ChatMessage`s
  - `RunCoreStreamingAsync(messages, session?, options?, ct)`:
    - Same setup as `RunCoreAsync`
    - Calls `ActivityProcessor.ProcessActivityAsync(Client.AskQuestionAsync(...), streaming: true, logger)`
    - Yields `AgentResponseUpdate` for each message
  - `GetService(Type, key?)` — returns `CopilotClient` or `AIAgentMetadata("copilot-studio")`
- **Session Management**: `CopilotStudioAgentSession` stores `ConversationId`; conversationId obtained from `Client.StartConversationAsync()`
- **Streaming**: True streaming via `IAsyncEnumerable<IActivity>` from `CopilotClient.AskQuestionAsync()`; filtered by activity type (`"typing"` for streaming, `"message"` for non-streaming)
- **Tool/Function Calling**: NOT SUPPORTED — CopilotStudio handles tools server-side
- **Authentication**: Handled by consumer when constructing `CopilotClient`
- **No Extension Methods**: Agent is directly constructed, no `AsAIAgent()` pattern

#### [CopilotStudioAgentSession.cs](dotnet/src/Microsoft.Agents.AI.CopilotStudio/CopilotStudioAgentSession.cs) (L1-31)

- **Class**: `CopilotStudioAgentSession` (public, sealed)
- **Extends**: `ServiceIdAgentSession`
- **Properties**: `ConversationId` (maps to `ServiceSessionId`)
- **Constructors**: internal only (default, or from `JsonElement`)

#### [ActivityProcessor.cs](dotnet/src/Microsoft.Agents.AI.CopilotStudio/ActivityProcessor.cs) (L1-48)

- **Class**: `ActivityProcessor` (internal, static)
- **Methods**:
  - `ProcessActivityAsync(IAsyncEnumerable<IActivity>, streaming, logger)` → `IAsyncEnumerable<ChatMessage>`
    - Filters by activity type: `"message"` (non-streaming) or `"typing"` (streaming)
    - Creates `ChatMessage` with `ChatRole.Assistant`, `TextContent`, `AuthorName`, `MessageId`, `RawRepresentation`

---

## 6. Microsoft.Agents.AI.GitHub.Copilot

### Project Configuration

- **csproj**: [Microsoft.Agents.AI.GitHub.Copilot.csproj](dotnet/src/Microsoft.Agents.AI.GitHub.Copilot/Microsoft.Agents.AI.GitHub.Copilot.csproj)
- **Target**: `.NET 8.0+` only (`$(TargetFrameworksCore)`)
- **NuGet Dependencies**: `GitHub.Copilot.SDK`
- **Project References**: `Microsoft.Agents.AI.Abstractions` (**NOT** `Microsoft.Agents.AI`)
- **Integration Pattern**: **Direct `AIAgent` subclass** + extension methods; does NOT use `IChatClient` or `ChatClientAgent`

### Files (4 total)

#### [GitHubCopilotAgent.cs](dotnet/src/Microsoft.Agents.AI.GitHub.Copilot/GitHubCopilotAgent.cs) (L1-471)

- **Class**: `GitHubCopilotAgent` (public, sealed)
- **Extends**: `AIAgent`, implements `IAsyncDisposable`
- **Constructor**:
  - `(CopilotClient, sessionConfig?, ownsClient?, id?, name?, description?)`
  - `(CopilotClient, ownsClient?, id?, name?, description?, tools?, instructions?)` — builds `SessionConfig` from tools/instructions
- **Properties**: `IdCore`, `Name`, `Description` (from constructor args)
- **Key Methods**:
  - `GetNewSessionAsync()` → `new GitHubCopilotAgentSession()`
  - `GetNewSessionAsync(sessionId)` → session with existing session ID
  - `DeserializeSessionAsync(JsonElement)` → `new GitHubCopilotAgentSession(serializedThread)`
  - `RunCoreAsync(messages, session?, options?, ct)` → delegates to `RunCoreStreamingAsync().ToAgentResponseAsync()` (streaming-first design)
  - `RunCoreStreamingAsync(messages, session?, options?, ct)`:
    - Creates/resumes `CopilotSession` via `_copilotClient.CreateSessionAsync(config)` or `ResumeSessionAsync(sessionId, resumeConfig)`
    - Subscribes to session events via `copilotSession.On(evt => ...)` using Channel-based producer/consumer
    - Event handling:
      - `AssistantMessageDeltaEvent` → `AgentResponseUpdate` with delta text
      - `AssistantMessageEvent` → `AgentResponseUpdate` with full message
      - `AssistantUsageEvent` → `AgentResponseUpdate` with `UsageContent` (input/output/cached tokens, cost, duration)
      - `SessionIdleEvent` → final update + channel complete
      - `SessionErrorEvent` → error update + channel complete with exception
      - Other events → stored as `RawRepresentation`
    - Processes `DataContent` attachments → temp files → `UserMessageDataAttachmentsItem`
    - Sends via `copilotSession.SendAsync(messageOptions)` with prompt + attachments
  - `DisposeAsync()` — disposes `_copilotClient` if `_ownsClient`
- **Private Helpers**:
  - `EnsureClientStartedAsync()` — starts client if not `ConnectionState.Connected`
  - `GetSessionConfig(tools, instructions)` — maps `AIFunction` tools + optional system message
  - `ProcessDataContentAttachmentsAsync()` — writes `DataContent` to temp files, returns attachment list
  - `CleanupTempFiles()` — best-effort temp file deletion
  - Media type mapping for temp file extensions (png, jpg, gif, webp, svg, txt, html, md, json, xml, pdf)
- **Session Management**: `GitHubCopilotAgentSession` stores `SessionId`
- **Streaming**: Full streaming via Channel-based pattern with `copilotSession.On()` event subscription
- **Tool/Function Calling**: Tools passed to `SessionConfig.Tools` as `List<AIFunction>`; only `AIFunction` types supported
- **Authentication**: Handled by consumer when constructing `GitHub.Copilot.SDK.CopilotClient`

#### [GitHubCopilotAgentSession.cs](dotnet/src/Microsoft.Agents.AI.GitHub.Copilot/GitHubCopilotAgentSession.cs) (L1-57)

- **Class**: `GitHubCopilotAgentSession` (public, sealed)
- **Extends**: `AgentSession` (directly, not `ServiceIdAgentSession`)
- **Properties**: `SessionId` (internal set)
- **Serialization**: `Serialize()` returns JSON with camelCase `sessionId` property
- **Deserialization**: Constructor reads `sessionId` from `JsonElement`
- **Inner Class**: `State` (internal, sealed) — serialization DTO

#### [CopilotClientExtensions.cs](dotnet/src/Microsoft.Agents.AI.GitHub.Copilot/CopilotClientExtensions.cs) (L1-77)

- **Class**: `CopilotClientExtensions` (static)
- **Namespace**: `GitHub.Copilot.SDK`
- **Methods**:
  - `AsAIAgent(this CopilotClient, sessionConfig?, ownsClient?, id?, name?, description?)` → `AIAgent`
  - `AsAIAgent(this CopilotClient, ownsClient?, id?, name?, description?, tools?, instructions?)` → `AIAgent`
- **Return Type**: Returns `AIAgent` (not `ChatClientAgent`)

#### [GitHubCopilotJsonUtilities.cs](dotnet/src/Microsoft.Agents.AI.GitHub.Copilot/GitHubCopilotJsonUtilities.cs) (L1-51)

- **Class**: `GitHubCopilotJsonUtilities` (internal, static, partial)
- **Properties**: `DefaultOptions` (read-only `JsonSerializerOptions`)
- **Configuration**: Chains `AgentAbstractionsJsonUtilities.DefaultOptions.TypeInfoResolver` + source-generated `JsonContext`
- **Inner Class**: `JsonContext` (private, sealed, partial) — source-generated with `JsonSerializerDefaults.Web` configuration, `GitHubCopilotAgentSession.State` support

---

## Cross-Cutting Analysis

### Provider Integration Patterns

| Provider | Base Class | Uses IChatClient | Uses ChatClientAgent | Has Custom Session | Has Extension Methods |
|----------|-----------|-------------------|---------------------|--------------------|-----------------------|
| **OpenAI** | N/A (extensions only) | Yes (via MEAI.OpenAI) | Yes | No (InMemory) | Yes (on ChatClient, ResponsesClient, AssistantClient) |
| **AzureAI** | N/A (extensions only) | Yes (custom DelegatingChatClient) | Yes | No (InMemory) | Yes (on AIProjectClient) |
| **AzureAI.Persistent** | N/A (extensions only) | Yes (via SDK's AsIChatClient) | Yes | No (InMemory) | Yes (on PersistentAgentsClient) |
| **Anthropic** | N/A (extensions only) | Yes (via Anthropic SDK) | Yes | No (InMemory) | Yes (on IAnthropicClient, IBetaService) |
| **CopilotStudio** | AIAgent (direct) | **No** | **No** | Yes (CopilotStudioAgentSession → ServiceIdAgentSession) | No |
| **GitHub Copilot** | AIAgent (direct) | **No** | **No** | Yes (GitHubCopilotAgentSession → AgentSession) | Yes (on CopilotClient) |

### Streaming Support

| Provider | Streaming Approach |
|----------|-------------------|
| **OpenAI** | Via `IChatClient.GetStreamingResponseAsync()` through `ChatClientAgent` |
| **AzureAI** | Via `DelegatingChatClient.GetStreamingResponseAsync()` override; passes agent-enabled options |
| **AzureAI.Persistent** | Via `IChatClient.GetStreamingResponseAsync()` through `ChatClientAgent` |
| **Anthropic** | Via `IChatClient.GetStreamingResponseAsync()` through `ChatClientAgent` |
| **CopilotStudio** | Direct: `IAsyncEnumerable<IActivity>` filtered by `"typing"` activity type → `AgentResponseUpdate` |
| **GitHub Copilot** | `Channel<AgentResponseUpdate>` + `copilotSession.On()` event subscription pattern |

### Tool/Function Calling

| Provider | Tool Pattern |
|----------|-------------|
| **OpenAI Chat/Responses** | `AITool` in `ChatOptions.Tools` → `ChatClientAgent` auto-wraps with `FunctionInvokingChatClient` |
| **OpenAI Assistants** | `AITool` → `ToolDefinition` conversion (CodeInterpreter, FileSearch, Functions); hosted tools go server-side, functions stay local |
| **AzureAI (Foundry)** | Rich tool validation: matches declared `FunctionTool` against provided `AIFunction` by name; supports `ResponseTool.AsAITool()` roundtrip; validates invocability |
| **AzureAI.Persistent** | `AITool` → `ToolDefinition` conversion (CodeInterpreter, FileSearch, BingGrounding, Functions); similar to Assistants |
| **Anthropic** | `AITool` in `ChatOptions.Tools` → `ChatClientAgent` auto-wraps with `FunctionInvokingChatClient` |
| **CopilotStudio** | Server-side only; no local tool support |
| **GitHub Copilot** | `AIFunction` tools passed to `SessionConfig.Tools`; only `AIFunction` types supported |

### Authentication Patterns

| Provider | Auth Approach |
|----------|--------------|
| **OpenAI** | Consumer provides API key or Azure credential when constructing OpenAI SDK client |
| **AzureAI** | `AIProjectClient` handles Azure credential + connection string; `MeaiUserAgentPolicy` adds MEAI user-agent header |
| **AzureAI.Persistent** | `PersistentAgentsClient` handles Azure credential |
| **Anthropic** | Consumer provides API key when constructing `IAnthropicClient` |
| **CopilotStudio** | Consumer provides auth when constructing `CopilotClient` |
| **GitHub Copilot** | Consumer provides auth when constructing `CopilotClient` |

### NuGet Dependency Map

| Provider | Key NuGet Packages |
|----------|-------------------|
| **OpenAI** | `Microsoft.Extensions.AI.OpenAI` |
| **AzureAI** | `Azure.AI.Projects`, `Azure.AI.Projects.OpenAI`, `Microsoft.Extensions.AI`, `Microsoft.Extensions.AI.OpenAI`, `OpenAI` |
| **AzureAI.Persistent** | `Azure.AI.Agents.Persistent`, `Microsoft.Extensions.AI` |
| **Anthropic** | `Anthropic`, `Microsoft.Extensions.AI` |
| **CopilotStudio** | `Microsoft.Agents.CopilotStudio.Client` |
| **GitHub Copilot** | `GitHub.Copilot.SDK` |

### Key Architectural Observations

1. **Two integration tiers**: Providers that use `IChatClient`/`ChatClientAgent` (OpenAI, AzureAI, Persistent, Anthropic) benefit from automatic function invocation, in-memory session management, telemetry, and logging built into `ChatClientAgent`. Providers that subclass `AIAgent` directly (CopilotStudio, GitHub Copilot) must implement all of this themselves.

2. **`clientFactory` pattern**: All `ChatClientAgent`-based providers accept `Func<IChatClient, IChatClient>? clientFactory` enabling middleware insertion (logging, caching, rate limiting) between the SDK client and the agent.

3. **Dual overloads**: Most providers offer both simple parameter overloads (`instructions`, `name`, `description`, `tools`) and full `ChatClientAgentOptions` overloads for advanced configuration.

4. **Namespace strategy**: Extension methods are placed in the SDK's own namespace (e.g., `OpenAI.Chat`, `Azure.AI.Projects`, `Anthropic`) so they appear naturally via IntelliSense on SDK types.

5. **Session divergence**: CopilotStudio uses `ServiceIdAgentSession` (server-managed state with local ID reference), GitHub Copilot uses bare `AgentSession` with custom serialization, while all ChatClientAgent-based providers default to `InMemoryAgentSession`.

6. **AzureAI is the most complex**: Has its own `DelegatingChatClient` subclass to intercept requests and inject agent references, strict tool validation, agent name validation, protocol-level API calls with `ModelReaderWriter`, JSON schema transformation for OpenAI structured output, and handles both declarative and invocable tool modes.
