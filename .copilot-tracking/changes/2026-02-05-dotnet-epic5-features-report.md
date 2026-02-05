# .NET Epic 5 Features Implementation Report

**Date:** February 5, 2026  
**Purpose:** Document .NET implementation of Epic 5 features for comparison with Go implementation

---

## Executive Summary

The .NET implementation has **comprehensive** coverage of Epic 5 features, with mature implementations across most categories. Key findings:

| Feature | .NET Status | Go Equivalent |
|---------|-------------|---------------|
| Resilience/Production Hardening | ❌ **Not Found** | ✅ `resilience/` |
| HTTP/gRPC Hosting | ✅ **Complete** | ✅ `hosting/` |
| Declarative Agents | ✅ **Complete** | ✅ `declarative/` |
| MCP Integration | ✅ **Complete** (Azure Functions) | ✅ `mcp/` |
| Durable Agents | ✅ **Complete** | ✅ `durable/` |
| DevUI | ✅ **Complete** | ✅ `devui/` |
| Purview | ✅ **Complete** | ✅ `purview/` |
| A2A Protocol | ✅ **Complete** (Additional) | ⚠️ `protocol/a2a` |
| AG-UI Protocol | ✅ **Complete** (Additional) | ⚠️ Pending |

---

## 1. Resilience/Production Hardening

### Status: ❌ NOT FOUND

**Location Searched:** `dotnet/src/Microsoft.Agents.AI/`

**Findings:**
- No dedicated resilience package found in .NET
- No Polly integration detected
- No circuit breaker, retry policies, or rate limiting middleware
- Only exception found: `PurviewRateLimitException` in Purview package (domain-specific)

**Files Found in Microsoft.Agents.AI:**
```
AgentExtensions.cs
AgentJsonUtilities.cs
AIAgentBuilder.cs
AnonymousDelegatingAIAgent.cs
FunctionInvocationDelegatingAgent.cs
FunctionInvocationDelegatingAgentBuilderExtensions.cs
LoggingAgent.cs
LoggingAgentBuilderExtensions.cs
OpenTelemetryAgent.cs
OpenTelemetryAgentBuilderExtensions.cs
TextSearchProvider.cs
TextSearchProviderOptions.cs
ChatClient/
Memory/
```

**Go Comparison:** Go has `resilience/` package with:
- Circuit breaker pattern
- Retry policies with exponential backoff
- Rate limiting
- Configuration options

**Gap Analysis:** .NET is missing this capability entirely - significant gap.

---

## 2. HTTP/gRPC Hosting

### Status: ✅ COMPLETE

**Location:** `dotnet/src/Microsoft.Agents.AI.Hosting/` and `dotnet/src/Microsoft.Agents.AI.Hosting.OpenAI/`

### Package: Microsoft.Agents.AI.Hosting

**Files:**
```
AgentHostingServiceCollectionExtensions.cs
AgentSessionStore.cs
AIHostAgent.cs
HostApplicationBuilderAgentExtensions.cs
HostApplicationBuilderWorkflowExtensions.cs
HostedAgentBuilder.cs
HostedAgentBuilderExtensions.cs
HostedWorkflowBuilder.cs
HostedWorkflowBuilderExtensions.cs
IHostedAgentBuilder.cs
IHostedWorkflowBuilder.cs
NoopAgentSessionStore.cs
WorkflowCatalog.cs
Local/
```

**Key Public Types:**
- `IHostedAgentBuilder` - Interface for agent configuration in hosting
- `HostedAgentBuilder` - Builder pattern implementation
- `IHostedWorkflowBuilder` - Workflow hosting configuration

**Extension Methods:**
```csharp
builder.AddAIAgent(name, instructions)
builder.AddAIAgent(name, instructions, chatClient)
builder.AddAIAgent(name, instructions, chatClientServiceKey)
builder.AddAIAgent(name, createAgentDelegate)
```

### Package: Microsoft.Agents.AI.Hosting.OpenAI

**Files:**
```
EndpointRouteBuilderExtensions.ChatCompletions.cs
EndpointRouteBuilderExtensions.Conversations.cs
EndpointRouteBuilderExtensions.Responses.cs
HostApplicationBuilderExtensions.cs
ServiceCollectionExtensions.cs
IdGenerator.cs
InMemoryStorageOptions.cs
SseJsonResult.cs
ChatCompletions/
Conversations/
Responses/
Models/
```

**Key Public APIs:**
```csharp
// OpenAI Responses API (latest)
app.MapOpenAIResponses()
app.MapOpenAIResponses(agent)
app.MapOpenAIResponses(agentBuilder)

// OpenAI Conversations API
app.MapOpenAIConversations()

// OpenAI Chat Completions (legacy)
app.MapOpenAIChatCompletions()
```

**Endpoint Capabilities:**
- Create response endpoint: `POST /{agent}/v1/responses/`
- Get response: `GET /{agent}/v1/responses/{responseId}`
- Cancel response: `POST /{agent}/v1/responses/{responseId}/cancel`
- Delete response: `DELETE /{agent}/v1/responses/{responseId}`
- Conversation CRUD operations
- SSE streaming support

---

## 3. Declarative Agents

### Status: ✅ COMPLETE

**Location:** `dotnet/src/Microsoft.Agents.AI.Declarative/`

**Files:**
```
AgentBotElementYaml.cs
AggregatorPromptAgentFactory.cs
PromptAgentFactory.cs
ChatClient/
  └── ChatClientPromptAgentFactory.cs
Extensions/
```

**Key Public Types:**

```csharp
// Abstract base factory
public abstract class PromptAgentFactory
{
    protected RecalcEngine Engine { get; }  // Power Fx engine for expressions
    
    public Task<AIAgent> CreateAsync(GptComponentMetadata promptAgent, CancellationToken cancellationToken = default);
    public abstract Task<AIAgent?> TryCreateAsync(GptComponentMetadata promptAgent, CancellationToken cancellationToken = default);
}

// ChatClient-based factory
public sealed class ChatClientPromptAgentFactory : PromptAgentFactory
{
    public ChatClientPromptAgentFactory(IChatClient chatClient, IList<AIFunction>? functions = null, 
                                         RecalcEngine? engine = null, IConfiguration? configuration = null, 
                                         ILoggerFactory? loggerFactory = null);
}
```

**Notable Capabilities:**
- Power Fx expression evaluation for dynamic configurations
- IConfiguration integration for environment variables
- YAML/metadata-based agent definition support
- ChatClient and tool integration

---

## 4. MCP Integration

### Status: ✅ COMPLETE (via Azure Functions)

**Location:** `dotnet/src/Microsoft.Agents.AI.Hosting.AzureFunctions/`

**Files:**
```
BuiltInFunctionExecutor.cs
BuiltInFunctions.cs
DurableAgentFunctionMetadataTransformer.cs
DurableAgentsOptionsExtensions.cs
DurableTaskClientExtensions.cs
FunctionsAgentOptions.cs
FunctionsApplicationBuilderExtensions.cs
HttpTriggerOptions.cs
McpToolTriggerOptions.cs
IFunctionsAgentOptionsProvider.cs
Middlewares/
```

**Package Dependencies:**
```xml
<PackageReference Include="Microsoft.Azure.Functions.Worker.Extensions.Mcp" />
```

**MCP Features:**
- MCP Tool Trigger support via Azure Functions
- Agent exposure as MCP tools
- Built-in function execution middleware

**Integration Approach:**
Unlike Go's standalone `mcp/` package, .NET integrates MCP through Azure Functions Worker Extensions, providing:
- MCP tool triggers
- Automatic MCP extension loading
- Integration with durable agents

**Sample Projects Available:**
- `Agent_MCP_Server` - Basic MCP server
- `Agent_MCP_Server_Auth` - MCP with authentication
- `FoundryAgents_Step09_UsingMcpClientAsTools` - MCP as tools
- `07_AgentAsMcpTool` - Durable agent as MCP tool

---

## 5. Durable Agents

### Status: ✅ COMPLETE (Mature)

**Location:** `dotnet/src/Microsoft.Agents.AI.DurableTask/`

**Files:**
```
AgentEntity.cs
AgentNotRegisteredException.cs
AgentRunHandle.cs
AgentSessionId.cs
AIAgentExtensions.cs
DefaultDurableAgentClient.cs
DurableAgentContext.cs
DurableAgentJsonUtilities.cs
DurableAgentRunOptions.cs
DurableAgentSession.cs
DurableAgentsOptions.cs
DurableAIAgent.cs
DurableAIAgentProxy.cs
EntityAgentWrapper.cs
IAgentResponseHandler.cs
IDurableAgentClient.cs
RunRequest.cs
ServiceCollectionExtensions.cs
TaskOrchestrationContextExtensions.cs
State/
  ├── DurableAgentState.cs
  ├── DurableAgentStateData.cs
  ├── DurableAgentStateMessage.cs
  ├── DurableAgentStateRequest.cs
  ├── DurableAgentStateResponse.cs
  └── ... (20+ state management files)
```

**Key Public Types:**

```csharp
// Main durable agent
public sealed class DurableAIAgent : AIAgent
{
    public override ValueTask<AgentSession> GetNewSessionAsync(CancellationToken cancellationToken = default);
    public override ValueTask<AgentSession> DeserializeSessionAsync(JsonElement serializedSession, ...);
    protected override Task<AgentResponse> RunCoreAsync(IEnumerable<ChatMessage> messages, ...);
}

// Session and context
public class DurableAgentSession : AgentSession { }
public class DurableAgentContext { }

// Client interface
internal interface IDurableAgentClient
{
    Task<AgentRunHandle> RunAgentAsync(AgentSessionId sessionId, RunRequest request, ...);
}
```

**Capabilities:**
- **Durable Entities** for stateful agent execution
- **Durable Orchestrations** for long-running workflows
- Automatic conversation history management
- State persistence across distributed environments
- Integration with Durable Task Scheduler

**Usage Pattern:**
```csharp
using IHost app = FunctionsApplication
    .CreateBuilder(args)
    .ConfigureFunctionsWebApplication()
    .ConfigureDurableAgents(options =>
    {
        options.AddAIAgent(spamDetector);
        options.AddAIAgent(emailAssistant);
    })
    .Build();
```

**Orchestration Integration:**
```csharp
[Function(nameof(SpamDetectionOrchestration))]
public static async Task<string> SpamDetectionOrchestration(
    [OrchestrationTrigger] TaskOrchestrationContext context)
{
    DurableAIAgent spamDetectionAgent = context.GetAgent("SpamDetectionAgent");
    AgentSession spamSession = await spamDetectionAgent.GetNewSessionAsync();
    AgentResponse<DetectionResult> response = await spamDetectionAgent.RunAsync<DetectionResult>(...);
}
```

---

## 6. DevUI

### Status: ✅ COMPLETE

**Location:** `dotnet/src/Microsoft.Agents.AI.DevUI/`

**Files:**
```
DevUIExtensions.cs
DevUIMiddleware.cs
HostApplicationBuilderExtensions.cs
ServiceCollectionsExtensions.cs
EntitiesApiExtensions.cs
MetaApiExtensions.cs
Entities/
  ├── EntitiesJsonContext.cs
  ├── EntityInfo.cs
  ├── MetaResponse.cs
  └── WorkflowSerializationExtensions.cs
wwwroot/  (embedded frontend resources)
Properties/
```

**Key Public APIs:**

```csharp
// Service registration
builder.AddDevUI();

// Endpoint mapping
app.MapDevUI();              // Maps to /devui
app.MapDevUI("/custom-path"); // Custom path
```

**Internal Components:**
- `DevUIMiddleware` - Serves embedded frontend resources
- Gzip compression support
- Content-type detection
- Cache control headers

**Dependencies:**
- Requires OpenAI Responses service
- Requires OpenAI Conversations service

**Usage Pattern:**
```csharp
var builder = WebApplication.CreateBuilder(args);
builder.AddAIAgent("assistant", "You are a helpful assistant.");
if (builder.Environment.IsDevelopment())
{
    builder.AddDevUI();
}
builder.AddOpenAIResponses();
builder.AddOpenAIConversations();

var app = builder.Build();
app.MapOpenAIResponses();
app.MapOpenAIConversations();
if (builder.Environment.IsDevelopment())
{
    app.MapDevUI();
}
```

---

## 7. Purview Integration

### Status: ✅ COMPLETE

**Location:** `dotnet/src/Microsoft.Agents.AI.Purview/`

**Files:**
```
PurviewAgent.cs
PurviewChatClient.cs
PurviewClient.cs
PurviewExtensions.cs
PurviewSettings.cs
PurviewWrapper.cs
PurviewAppLocation.cs
PurviewLocationType.cs
BackgroundJobRunner.cs
CacheProvider.cs
ChannelHandler.cs
ScopedContentProcessor.cs
Exceptions/
  └── PurviewRateLimitException.cs
Models/
  ├── Common/
  ├── Jobs/
  ├── Requests/
  └── Responses/
Serialization/
```

**Key Public Types:**

```csharp
// Extension methods for builder pattern
public static class PurviewExtensions
{
    // For AIAgent
    public static AIAgentBuilder WithPurview(this AIAgentBuilder builder, 
        TokenCredential tokenCredential, 
        PurviewSettings purviewSettings, 
        ILogger? logger = null, 
        IDistributedCache? cache = null);
    
    // For ChatClient
    public static ChatClientBuilder WithPurview(this ChatClientBuilder builder, 
        TokenCredential tokenCredential, 
        PurviewSettings purviewSettings, 
        ILogger? logger = null, 
        IDistributedCache? cache = null);
}

// Configuration
public class PurviewSettings
{
    public int InMemoryCacheSizeLimit { get; set; }
    public int PendingBackgroundJobLimit { get; set; }
}
```

**Capabilities:**
- Middleware-based policy enforcement
- Prompt (ingress) filtering
- Response (egress) filtering
- TokenCredential authentication (Azure.Core)
- Distributed cache support (IDistributedCache)
- Background job processing
- Rate limit handling

**Authentication Requirements:**
- ProtectionScopes.Compute.All
- Content.Process.All
- ContentActivity.Write

**Usage Pattern:**
```csharp
IChatClient client = new AzureOpenAIClient(...)
    .GetResponsesClient(deploymentName)
    .AsIChatClient()
    .AsBuilder()
    .WithPurview(browserCredential, new PurviewSettings("My Sample App"))
    .Build();
```

---

## 8. Additional Features (Not in Go Comparison)

### A2A Protocol Integration

**Location:** 
- `dotnet/src/Microsoft.Agents.AI.A2A/`
- `dotnet/src/Microsoft.Agents.AI.Hosting.A2A/`
- `dotnet/src/Microsoft.Agents.AI.Hosting.A2A.AspNetCore/`

**Key Types:**
```csharp
public sealed class A2AAgent : AIAgent
{
    public A2AAgent(A2AClient a2aClient, string? id = null, string? name = null, ...);
    public ValueTask<AgentSession> GetNewSessionAsync(string contextId);
}

// Endpoint mapping
app.MapA2A(agentBuilder, path);
app.MapA2A(agentName, path, configureTaskManager);
```

### AG-UI Protocol Integration

**Location:**
- `dotnet/src/Microsoft.Agents.AI.AGUI/`
- `dotnet/src/Microsoft.Agents.AI.Hosting.AGUI.AspNetCore/`

**Key Types:**
```csharp
public sealed class AGUIChatClient : DelegatingChatClient { }

// Endpoint mapping
app.MapAGUI(pattern, aiAgent);
```

---

## Gap Analysis Summary

| Feature | .NET | Go | Gap |
|---------|------|----|----|
| **Resilience** | ❌ Missing | ✅ Complete | **Critical** - .NET needs Polly integration |
| **HTTP Hosting** | ✅ Complete | ✅ Complete | None |
| **Declarative** | ✅ Complete | ✅ Complete | None |
| **MCP** | ✅ Azure Functions | ✅ Standalone | Different approach - both valid |
| **Durable** | ✅ Mature | ✅ Complete | None |
| **DevUI** | ✅ Complete | ✅ Complete | None |
| **Purview** | ✅ Complete | ✅ Complete | None |
| **A2A** | ✅ Complete | ⚠️ Basic | Go has basic, .NET has full hosting |
| **AG-UI** | ✅ Complete | ❌ Not found | .NET only |

---

## Recommendations

1. **Priority 1:** Add resilience package to .NET with Polly integration
2. **Priority 2:** Consider standalone MCP package (not Azure Functions dependent)
3. **Priority 3:** Evaluate AG-UI for Go implementation
4. **Priority 4:** Enhance Go A2A with ASP.NET-style hosting patterns
