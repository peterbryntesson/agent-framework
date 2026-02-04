# .NET HTTP/gRPC Hosting Implementation Analysis

**Date**: 2026-02-04  
**Objective**: Analyze the .NET hosting packages for HTTP, gRPC, A2A, AG-UI, and OpenAI-compatible hosting.

---

## 1. Core Hosting Package

**Location**: [dotnet/src/Microsoft.Agents.AI.Hosting/](dotnet/src/Microsoft.Agents.AI.Hosting/)

### 1.1 Key Files

| File | Purpose |
|------|---------|
| [IHostedAgentBuilder.cs](dotnet/src/Microsoft.Agents.AI.Hosting/IHostedAgentBuilder.cs) | Builder interface for configuring agents |
| [HostedAgentBuilder.cs](dotnet/src/Microsoft.Agents.AI.Hosting/HostedAgentBuilder.cs) | Implementation of the builder pattern |
| [AgentHostingServiceCollectionExtensions.cs](dotnet/src/Microsoft.Agents.AI.Hosting/AgentHostingServiceCollectionExtensions.cs) | DI registration extensions |
| [HostApplicationBuilderAgentExtensions.cs](dotnet/src/Microsoft.Agents.AI.Hosting/HostApplicationBuilderAgentExtensions.cs) | Host builder extensions |
| [AIHostAgent.cs](dotnet/src/Microsoft.Agents.AI.Hosting/AIHostAgent.cs) | Wrapper with session persistence |
| [AgentSessionStore.cs](dotnet/src/Microsoft.Agents.AI.Hosting/AgentSessionStore.cs) | Abstract session storage |
| [NoopAgentSessionStore.cs](dotnet/src/Microsoft.Agents.AI.Hosting/NoopAgentSessionStore.cs) | No-op store implementation |
| [Local/InMemoryAgentSessionStore.cs](dotnet/src/Microsoft.Agents.AI.Hosting/Local/InMemoryAgentSessionStore.cs) | In-memory store for dev/test |

### 1.2 Interface Definitions

#### IHostedAgentBuilder (Lines 1-22)

```csharp
public interface IHostedAgentBuilder
{
    /// <summary>Gets the name of the agent being configured.</summary>
    string Name { get; }

    /// <summary>Gets the service collection for configuration.</summary>
    IServiceCollection ServiceCollection { get; }
}
```

#### AgentSessionStore (Lines 1-46)

```csharp
public abstract class AgentSessionStore
{
    public abstract ValueTask SaveSessionAsync(
        AIAgent agent,
        string conversationId,
        AgentSession session,
        CancellationToken cancellationToken = default);

    public abstract ValueTask<AgentSession> GetSessionAsync(
        AIAgent agent,
        string conversationId,
        CancellationToken cancellationToken = default);
}
```

### 1.3 Registration Patterns

The hosting package provides multiple registration overloads:

```csharp
// Name + Instructions (resolves IChatClient from DI)
services.AddAIAgent("agent-name", "instructions");

// Name + Instructions + ChatClient instance
services.AddAIAgent("agent-name", "instructions", chatClient);

// Name + Instructions + keyed ChatClient
services.AddAIAgent("agent-name", "instructions", chatClientServiceKey);

// Name + Instructions + Description + keyed ChatClient
services.AddAIAgent("agent-name", "instructions", "description", chatClientServiceKey);

// Custom factory delegate
services.AddAIAgent("agent-name", (sp, key) => new MyAgent(...));
```

### 1.4 Session Persistence Pattern

The `AIHostAgent` wraps any `AIAgent` and adds session persistence:

```csharp
public class AIHostAgent : DelegatingAIAgent
{
    private readonly AgentSessionStore _sessionStore;

    public ValueTask<AgentSession> GetOrCreateSessionAsync(string conversationId, CancellationToken ct);
    public ValueTask SaveSessionAsync(string conversationId, AgentSession session, CancellationToken ct);
}
```

---

## 2. A2A Protocol Hosting

**Locations**:
- [dotnet/src/Microsoft.Agents.AI.Hosting.A2A/](dotnet/src/Microsoft.Agents.AI.Hosting.A2A/)
- [dotnet/src/Microsoft.Agents.AI.Hosting.A2A.AspNetCore/](dotnet/src/Microsoft.Agents.AI.Hosting.A2A.AspNetCore/)

### 2.1 Key Files

| File | Purpose |
|------|---------|
| [AIAgentExtensions.cs](dotnet/src/Microsoft.Agents.AI.Hosting.A2A/AIAgentExtensions.cs#L1-L100) | Maps AIAgent to A2A TaskManager |
| [Converters/MessageConverter.cs](dotnet/src/Microsoft.Agents.AI.Hosting.A2A/Converters/MessageConverter.cs) | A2A ↔ ChatMessage conversion |
| [Converters/A2AMetadataExtensions.cs](dotnet/src/Microsoft.Agents.AI.Hosting.A2A/Converters/A2AMetadataExtensions.cs) | Metadata conversion |
| [EndpointRouteBuilderExtensions.cs](dotnet/src/Microsoft.Agents.AI.Hosting.A2A.AspNetCore/EndpointRouteBuilderExtensions.cs) | ASP.NET Core endpoint mapping |

### 2.2 Task Lifecycle Pattern

The A2A integration uses the `TaskManager` from the A2A SDK with event handlers:

```csharp
public static ITaskManager MapA2A(
    this AIAgent agent,
    ITaskManager? taskManager = null,
    ILoggerFactory? loggerFactory = null,
    AgentSessionStore? agentSessionStore = null)
{
    taskManager ??= new TaskManager();
    taskManager.OnMessageReceived += OnMessageReceivedAsync;
    return taskManager;

    async Task<A2AResponse> OnMessageReceivedAsync(MessageSendParams messageSendParams, CancellationToken ct)
    {
        var contextId = messageSendParams.Message.ContextId ?? Guid.NewGuid().ToString("N");
        var session = await hostAgent.GetOrCreateSessionAsync(contextId, ct);
        
        var response = await hostAgent.RunAsync(
            messageSendParams.ToChatMessages(),
            session: session,
            options: options,
            cancellationToken: ct);

        await hostAgent.SaveSessionAsync(contextId, session, ct);
        return new AgentMessage { ... };
    }
}
```

### 2.3 Endpoint Mapping Pattern

```csharp
// Map A2A with agent name
app.MapA2A(agentBuilder, "/a2a");

// Map A2A with AgentCard for discovery
app.MapA2A(agentBuilder, "/a2a", new AgentCard
{
    Name = "MyAgent",
    Description = "An example agent",
    Capabilities = new[] { "chat" }
});

// Map A2A with TaskManager configuration
app.MapA2A("agent-name", "/a2a", taskManager =>
{
    taskManager.OnAgentCardQuery += (context, query) => ...;
});
```

### 2.4 Message Conversion (Lines 12-53)

```csharp
internal static class MessageConverter
{
    public static List<Part> ToParts(this IList<ChatMessage> chatMessages);
    public static List<ChatMessage> ToChatMessages(this MessageSendParams messageSendParams);
}
```

---

## 3. AG-UI Hosting

**Location**: [dotnet/src/Microsoft.Agents.AI.Hosting.AGUI.AspNetCore/](dotnet/src/Microsoft.Agents.AI.Hosting.AGUI.AspNetCore/)

### 3.1 Key Files

| File | Purpose |
|------|---------|
| [AGUIEndpointRouteBuilderExtensions.cs](dotnet/src/Microsoft.Agents.AI.Hosting.AGUI.AspNetCore/AGUIEndpointRouteBuilderExtensions.cs) | Maps AG-UI endpoints |
| [AGUIServerSentEventsResult.cs](dotnet/src/Microsoft.Agents.AI.Hosting.AGUI.AspNetCore/AGUIServerSentEventsResult.cs) | SSE streaming result |
| [AGUIChatResponseUpdateStreamExtensions.cs](dotnet/src/Microsoft.Agents.AI.Hosting.AGUI.AspNetCore/AGUIChatResponseUpdateStreamExtensions.cs) | Filters client/server tools |
| [ServiceCollectionExtensions.cs](dotnet/src/Microsoft.Agents.AI.Hosting.AGUI.AspNetCore/ServiceCollectionExtensions.cs) | Service registration |
| [AGUIJsonSerializerOptions.cs](dotnet/src/Microsoft.Agents.AI.Hosting.AGUI.AspNetCore/AGUIJsonSerializerOptions.cs) | JSON serialization config |

### 3.2 SSE Streaming Implementation

The `AGUIServerSentEventsResult` class implements `IResult` for SSE streaming:

```csharp
internal sealed partial class AGUIServerSentEventsResult : IResult, IDisposable
{
    private readonly IAsyncEnumerable<BaseEvent> _events;
    
    public async Task ExecuteAsync(HttpContext httpContext)
    {
        httpContext.Response.ContentType = "text/event-stream";
        httpContext.Response.Headers.CacheControl = "no-cache,no-store";
        httpContext.Response.Headers.Pragma = "no-cache";

        await SseFormatter.WriteAsync(
            WrapEventsAsSseItemsAsync(_events, cancellationToken),
            body,
            SerializeEvent,
            cancellationToken);
    }
}
```

### 3.3 Endpoint Mapping Pattern

```csharp
public static IEndpointConventionBuilder MapAGUI(
    this IEndpointRouteBuilder endpoints,
    string pattern,
    AIAgent aiAgent)
{
    return endpoints.MapPost(pattern, async (RunAgentInput? input, HttpContext context, CancellationToken ct) =>
    {
        var messages = input.Messages.AsChatMessages(jsonSerializerOptions);
        var clientTools = input.Tools?.AsAITools().ToList();

        var runOptions = new ChatClientAgentRunOptions
        {
            ChatOptions = new ChatOptions
            {
                Tools = clientTools,
                AdditionalProperties = new AdditionalPropertiesDictionary
                {
                    ["ag_ui_state"] = input.State,
                    ["ag_ui_context"] = input.Context,
                    ["ag_ui_thread_id"] = input.ThreadId,
                    ["ag_ui_run_id"] = input.RunId
                }
            }
        };

        var events = aiAgent.RunStreamingAsync(messages, options: runOptions, cancellationToken: ct)
            .AsChatResponseUpdatesAsync()
            .FilterServerToolsFromMixedToolInvocationsAsync(clientTools, ct)
            .AsAGUIEventStreamAsync(input.ThreadId, input.RunId, jsonSerializerOptions, ct);

        return new AGUIServerSentEventsResult(events, logger);
    });
}
```

### 3.4 Tool Filtering (Client vs Server)

The `FilterServerToolsFromMixedToolInvocationsAsync` extension filters out server-side tool invocations when both client and server tools are present:

```csharp
public static async IAsyncEnumerable<ChatResponseUpdate> FilterServerToolsFromMixedToolInvocationsAsync(
    this IAsyncEnumerable<ChatResponseUpdate> updates,
    List<AITool>? clientTools,
    CancellationToken cancellationToken)
{
    // Filters FunctionCallContent to only include client tools when mixed
}
```

---

## 4. OpenAI-Compatible Hosting

**Location**: [dotnet/src/Microsoft.Agents.AI.Hosting.OpenAI/](dotnet/src/Microsoft.Agents.AI.Hosting.OpenAI/)

### 4.1 Key Files

| File | Purpose |
|------|---------|
| [EndpointRouteBuilderExtensions.ChatCompletions.cs](dotnet/src/Microsoft.Agents.AI.Hosting.OpenAI/EndpointRouteBuilderExtensions.ChatCompletions.cs) | Chat completions endpoints |
| [EndpointRouteBuilderExtensions.Responses.cs](dotnet/src/Microsoft.Agents.AI.Hosting.OpenAI/EndpointRouteBuilderExtensions.Responses.cs) | Responses API endpoints |
| [EndpointRouteBuilderExtensions.Conversations.cs](dotnet/src/Microsoft.Agents.AI.Hosting.OpenAI/EndpointRouteBuilderExtensions.Conversations.cs) | Conversations API endpoints |
| [SseJsonResult.cs](dotnet/src/Microsoft.Agents.AI.Hosting.OpenAI/SseJsonResult.cs) | Generic SSE result |
| [ServiceCollectionExtensions.cs](dotnet/src/Microsoft.Agents.AI.Hosting.OpenAI/ServiceCollectionExtensions.cs) | Service registration |
| [ChatCompletions/AIAgentChatCompletionsProcessor.cs](dotnet/src/Microsoft.Agents.AI.Hosting.OpenAI/ChatCompletions/AIAgentChatCompletionsProcessor.cs) | Request processing |
| [Responses/ResponsesHttpHandler.cs](dotnet/src/Microsoft.Agents.AI.Hosting.OpenAI/Responses/ResponsesHttpHandler.cs) | Responses API handler |
| [Responses/IResponsesService.cs](dotnet/src/Microsoft.Agents.AI.Hosting.OpenAI/Responses/IResponsesService.cs) | Service interface |
| [Responses/AIAgentResponseExecutor.cs](dotnet/src/Microsoft.Agents.AI.Hosting.OpenAI/Responses/AIAgentResponseExecutor.cs) | Local execution |
| [Responses/InMemoryResponsesService.cs](dotnet/src/Microsoft.Agents.AI.Hosting.OpenAI/Responses/InMemoryResponsesService.cs) | In-memory service |
| [Conversations/IConversationStorage.cs](dotnet/src/Microsoft.Agents.AI.Hosting.OpenAI/Conversations/IConversationStorage.cs) | Conversation storage |

### 4.2 Chat Completions Endpoint Pattern

```csharp
public static IEndpointConventionBuilder MapOpenAIChatCompletions(
    this IEndpointRouteBuilder endpoints,
    AIAgent agent,
    string? path)
{
    path ??= $"/{agent.Name}/v1/chat/completions";
    var group = endpoints.MapGroup(path);

    group.MapPost("/", async (CreateChatCompletion request, CancellationToken ct)
        => await AIAgentChatCompletionsProcessor.CreateChatCompletionAsync(agent, request, ct))
        .WithName(agent.Name + "/CreateChatCompletion");

    return group;
}
```

### 4.3 Responses API Endpoints

```csharp
public static IEndpointConventionBuilder MapOpenAIResponses(
    this IEndpointRouteBuilder endpoints,
    AIAgent agent,
    string? responsesPath)
{
    responsesPath ??= $"/{agent.Name}/v1/responses";
    var group = endpoints.MapGroup(responsesPath);

    // POST / - Create response
    group.MapPost("/", handlers.CreateResponseAsync);
    
    // GET {responseId} - Get response
    group.MapGet("{responseId}", handlers.GetResponseAsync);
    
    // POST {responseId}/cancel - Cancel response
    group.MapPost("{responseId}/cancel", handlers.CancelResponseAsync);
    
    // DELETE {responseId} - Delete response
    group.MapDelete("{responseId}", handlers.DeleteResponseAsync);
    
    // GET {responseId}/input_items - List input items
    group.MapGet("{responseId}/input_items", handlers.ListResponseInputItemsAsync);

    return group;
}
```

### 4.4 Conversations API Endpoints

```csharp
public static IEndpointConventionBuilder MapOpenAIConversations(this IEndpointRouteBuilder endpoints)
{
    var group = endpoints.MapGroup("/v1/conversations");

    // Conversation CRUD
    group.MapGet("", handlers.ListConversationsByAgentAsync);
    group.MapPost("", handlers.CreateConversationAsync);
    group.MapGet("{conversationId}", handlers.GetConversationAsync);
    group.MapPost("{conversationId}", handlers.UpdateConversationAsync);
    group.MapDelete("{conversationId}", handlers.DeleteConversationAsync);

    // Item operations
    group.MapPost("{conversationId}/items", handlers.CreateItemsAsync);
    group.MapGet("{conversationId}/items", handlers.ListItemsAsync);
    group.MapGet("{conversationId}/items/{itemId}", handlers.GetItemAsync);
    group.MapDelete("{conversationId}/items/{itemId}", handlers.DeleteItemAsync);

    return group;
}
```

### 4.5 SSE Streaming Pattern (Generic)

```csharp
internal sealed class SseJsonResult<T> : IResult
{
    public async Task ExecuteAsync(HttpContext httpContext)
    {
        response.Headers.ContentType = "text/event-stream";
        response.Headers.CacheControl = "no-cache,no-store";
        response.Headers.Connection = "keep-alive";
        response.Headers.ContentEncoding = "identity";
        httpContext.Features.GetRequiredFeature<IHttpResponseBodyFeature>().DisableBuffering();

        await SseFormatter.WriteAsync(
            source: GetItemsAsync(),
            destination: response.Body,
            itemFormatter: FormatItem,
            cancellationToken);
    }
}
```

### 4.6 Response Executor Interface

```csharp
internal interface IResponseExecutor
{
    ValueTask<ResponseError?> ValidateRequestAsync(CreateResponse request, CancellationToken ct);
    
    IAsyncEnumerable<StreamingResponseEvent> ExecuteAsync(
        AgentInvocationContext context,
        CreateResponse request,
        CancellationToken ct);
}
```

### 4.7 Conversation Storage Interface

```csharp
internal interface IConversationStorage
{
    Task<Conversation> CreateConversationAsync(Conversation conversation, CancellationToken ct);
    Task<Conversation?> GetConversationAsync(string conversationId, CancellationToken ct);
    Task<Conversation?> UpdateConversationAsync(Conversation conversation, CancellationToken ct);
    Task<bool> DeleteConversationAsync(string conversationId, CancellationToken ct);
    
    Task AddItemsAsync(string conversationId, IEnumerable<ItemResource> items, CancellationToken ct);
    Task<ItemResource?> GetItemAsync(string conversationId, string itemId, CancellationToken ct);
    Task<ListResponse<ItemResource>> ListItemsAsync(string conversationId, int? limit, SortOrder? order, string? after, CancellationToken ct);
    Task<bool> DeleteItemAsync(string conversationId, string itemId, CancellationToken ct);
}
```

---

## 5. Azure Functions Hosting

**Location**: [dotnet/src/Microsoft.Agents.AI.Hosting.AzureFunctions/](dotnet/src/Microsoft.Agents.AI.Hosting.AzureFunctions/)

### 5.1 Key Files

| File | Purpose |
|------|---------|
| [FunctionsApplicationBuilderExtensions.cs](dotnet/src/Microsoft.Agents.AI.Hosting.AzureFunctions/FunctionsApplicationBuilderExtensions.cs) | Builder extensions |
| [FunctionsAgentOptions.cs](dotnet/src/Microsoft.Agents.AI.Hosting.AzureFunctions/FunctionsAgentOptions.cs) | Agent options |
| [HttpTriggerOptions.cs](dotnet/src/Microsoft.Agents.AI.Hosting.AzureFunctions/HttpTriggerOptions.cs) | HTTP trigger config |
| [McpToolTriggerOptions.cs](dotnet/src/Microsoft.Agents.AI.Hosting.AzureFunctions/McpToolTriggerOptions.cs) | MCP tool trigger config |
| [BuiltInFunctions.cs](dotnet/src/Microsoft.Agents.AI.Hosting.AzureFunctions/BuiltInFunctions.cs) | HTTP/Entity/MCP handlers |
| [BuiltInFunctionExecutor.cs](dotnet/src/Microsoft.Agents.AI.Hosting.AzureFunctions/BuiltInFunctionExecutor.cs) | Function execution |
| [DurableTaskClientExtensions.cs](dotnet/src/Microsoft.Agents.AI.Hosting.AzureFunctions/DurableTaskClientExtensions.cs) | Durable proxy extensions |
| [DurableAgentFunctionMetadataTransformer.cs](dotnet/src/Microsoft.Agents.AI.Hosting.AzureFunctions/DurableAgentFunctionMetadataTransformer.cs) | Metadata generation |
| [DurableAgentsOptionsExtensions.cs](dotnet/src/Microsoft.Agents.AI.Hosting.AzureFunctions/DurableAgentsOptionsExtensions.cs) | Agent registration |
| [Middlewares/BuiltInFunctionExecutionMiddleware.cs](dotnet/src/Microsoft.Agents.AI.Hosting.AzureFunctions/Middlewares/BuiltInFunctionExecutionMiddleware.cs) | Custom executor middleware |

### 5.2 Configuration Pattern

```csharp
public static FunctionsApplicationBuilder ConfigureDurableAgents(
    this FunctionsApplicationBuilder builder,
    Action<DurableAgentsOptions> configure)
{
    // Register agent services
    builder.Services.ConfigureDurableAgents(configure);

    // Register options provider
    builder.Services.TryAddSingleton<IFunctionsAgentOptionsProvider>(...);

    // Register metadata transformer for dynamic function generation
    builder.Services.AddSingleton<IFunctionMetadataTransformer, DurableAgentFunctionMetadataTransformer>();

    // Middleware for built-in function execution
    builder.UseWhen<BuiltInFunctionExecutionMiddleware>(context =>
        context.FunctionDefinition.EntryPoint == BuiltInFunctions.RunAgentHttpFunctionEntryPoint ||
        context.FunctionDefinition.EntryPoint == BuiltInFunctions.RunAgentMcpToolFunctionEntryPoint ||
        context.FunctionDefinition.EntryPoint == BuiltInFunctions.RunAgentEntityFunctionEntryPoint);

    return builder;
}
```

### 5.3 Agent Registration with Triggers

```csharp
// Add agent with custom trigger configuration
options.AddAIAgent(agent, agentOptions =>
{
    agentOptions.HttpTrigger.IsEnabled = true;
    agentOptions.McpToolTrigger.IsEnabled = true;
});

// Or with explicit flags
options.AddAIAgent(agent, enableHttpTrigger: true, enableMcpToolTrigger: false);
```

### 5.4 HTTP Trigger Handler (Lines 46-145)

```csharp
public static async Task<HttpResponseData> RunAgentHttpAsync(
    [HttpTrigger] HttpRequestData req,
    [DurableClient] DurableTaskClient client,
    FunctionContext context)
{
    // Parse request (JSON or plain text)
    string? message = await ParseRequestBodyAsync(req, context);
    string? threadId = req.Query["thread_id"] ?? threadIdFromBody;

    // Create session ID
    AgentSessionId sessionId = new AgentSessionId(agentName, threadId ?? context.InvocationId);

    // Check for fire-and-forget mode
    bool waitForResponse = req.Headers.GetValues("x-ms-wait-for-response") != "false";

    // Create durable agent proxy
    AIAgent agentProxy = client.AsDurableAgentProxy(context, agentName);

    if (waitForResponse)
    {
        AgentResponse response = await agentProxy.RunAsync(
            message: new ChatMessage(ChatRole.User, message),
            session: new DurableAgentSession(sessionId),
            options: new DurableAgentRunOptions { IsFireAndForget = false });
        return CreateSuccessResponseAsync(req, context, sessionId.Key, response);
    }

    // Fire and forget - return 202 Accepted
    await agentProxy.RunAsync(..., options: new DurableAgentRunOptions { IsFireAndForget = true });
    return CreateAcceptedResponseAsync(req, context, sessionId.Key);
}
```

### 5.5 Entity Trigger Handler

```csharp
public static Task<string> InvokeAgentAsync(
    [DurableClient] DurableTaskClient client,
    string encodedEntityRequest,
    FunctionContext functionContext)
{
    IServiceProvider combinedServiceProvider = new CombinedServiceProvider(
        functionContext.InstanceServices, 
        client);

    AgentEntity entity = new(combinedServiceProvider, functionContext.CancellationToken);
    return GrpcEntityRunner.LoadAndRunAsync(encodedEntityRequest, entity, combinedServiceProvider);
}
```

### 5.6 MCP Tool Trigger Handler

```csharp
public static async Task<string?> RunMcpToolAsync(
    [McpToolTrigger("BuiltInMcpTool")] ToolInvocationContext context,
    [DurableClient] DurableTaskClient client,
    FunctionContext functionContext)
{
    string query = (string)context.Arguments["query"];
    string agentName = context.Name;

    AgentSessionId sessionId = context.Arguments.TryGetValue("threadId", out var threadId)
        ? AgentSessionId.Parse((string)threadId)
        : new AgentSessionId(agentName, functionContext.InvocationId);

    AIAgent agentProxy = client.AsDurableAgentProxy(functionContext, agentName);

    AgentResponse response = await agentProxy.RunAsync(
        message: new ChatMessage(ChatRole.User, query),
        session: new DurableAgentSession(sessionId));

    return response.Text;
}
```

### 5.7 Dynamic Function Metadata Generation

The `DurableAgentFunctionMetadataTransformer` dynamically creates Azure Functions for each registered agent:

```csharp
public void Transform(IList<IFunctionMetadata> original)
{
    foreach (var (agentName, factory) in _agents)
    {
        // Always create entity trigger
        original.Add(CreateAgentTrigger(agentName));

        // Optionally create HTTP trigger
        if (options.HttpTrigger.IsEnabled)
            original.Add(CreateHttpTrigger(agentName, $"agents/{agentName}/run"));

        // Optionally create MCP tool trigger
        if (options.McpToolTrigger.IsEnabled)
            original.Add(CreateMcpToolTrigger(agentName, agent.Description));
    }
}
```

### 5.8 Durable Agent Proxy Pattern

```csharp
public static AIAgent AsDurableAgentProxy(
    this DurableTaskClient durableClient,
    FunctionContext context,
    string agentName)
{
    // Validate agent is registered
    ServiceCollectionExtensions.ValidateAgentIsRegistered(context.InstanceServices, agentName);

    // Create durable agent client
    DefaultDurableAgentClient agentClient = ActivatorUtilities.CreateInstance<DefaultDurableAgentClient>(
        context.InstanceServices,
        durableClient);

    return new DurableAIAgentProxy(agentName, agentClient);
}
```

---

## 6. Key Patterns Summary

### 6.1 Handler Interface Patterns

| Protocol | Handler Type | Interface |
|----------|--------------|-----------|
| Core | Session Store | `AgentSessionStore` (abstract class) |
| A2A | TaskManager | `ITaskManager` (from A2A SDK) |
| AG-UI | IResult | `AGUIServerSentEventsResult` |
| OpenAI ChatCompletions | Processor | `AIAgentChatCompletionsProcessor` (static) |
| OpenAI Responses | Service | `IResponsesService` |
| OpenAI Responses | Executor | `IResponseExecutor` |
| OpenAI Conversations | Storage | `IConversationStorage` |
| Azure Functions | Executor | `IFunctionExecutor` |

### 6.2 Middleware Pipeline Integration

- **Core Hosting**: Uses DI keyed services for agent resolution
- **A2A**: Event-based (OnMessageReceived, OnAgentCardQuery)
- **AG-UI**: Uses ASP.NET Core minimal APIs with MapPost
- **OpenAI**: Uses ASP.NET Core minimal APIs with MapGroup
- **Azure Functions**: Uses IFunctionsWorkerMiddleware and IFunctionMetadataTransformer

### 6.3 Request/Response Serialization

| Protocol | Serialization |
|----------|---------------|
| A2A | A2AJsonUtilities, custom converters |
| AG-UI | AGUIJsonSerializerContext, System.Text.Json |
| OpenAI | ChatCompletionsJsonContext, OpenAIHostingJsonContext |
| Azure Functions | System.Text.Json with function bindings |

### 6.4 Streaming Patterns

| Protocol | Pattern | Implementation |
|----------|---------|----------------|
| A2A | Not implemented (sync response) | TaskManager events |
| AG-UI | SSE | AGUIServerSentEventsResult + SseFormatter |
| OpenAI ChatCompletions | SSE | SseFormatter + StreamingResponse class |
| OpenAI Responses | SSE | SseJsonResult<T> + IAsyncEnumerable |
| Azure Functions | Sync/Fire-and-forget | HTTP response or 202 Accepted |

### 6.5 Graceful Shutdown

- **A2A**: CancellationToken propagation
- **AG-UI**: Uses `httpContext.RequestAborted`, error event on failure
- **OpenAI**: CancellationToken propagation, error handling in try/catch
- **Azure Functions**: FunctionContext.CancellationToken, durable entity checkpointing

---

## 7. Clarifying Questions

1. **gRPC Hosting**: No dedicated gRPC hosting package was found. Is gRPC support planned, or should it be implemented as part of this research?

2. **A2A Streaming**: The current A2A implementation appears to be synchronous (returns single AgentMessage). Is streaming A2A support planned?

3. **Authentication/Authorization**: None of the hosting packages implement auth. Is this intentional (delegated to ASP.NET Core middleware)?

4. **Health Checks**: No health check endpoints are defined. Should these be added to the hosting packages?

5. **OpenTelemetry Integration**: Is there a separate package for observability, or should it be integrated into hosting packages?
