# .NET Provider Implementations Research

## Directory Structure Overview

### Provider Packages Location

```
dotnet/src/
├── Microsoft.Agents.AI.Abstractions/     # Core abstractions (AIAgent, AgentSession, etc.)
├── Microsoft.Agents.AI/                  # Base implementations (ChatClientAgent, etc.)
├── Microsoft.Agents.AI.Anthropic/        # Anthropic provider
├── Microsoft.Agents.AI.AzureAI/          # Azure AI Project provider
├── Microsoft.Agents.AI.AzureAI.Persistent/ # Azure Persistent Agents provider
├── Microsoft.Agents.AI.OpenAI/           # OpenAI provider
├── Microsoft.Agents.AI.CopilotStudio/    # Copilot Studio provider
├── Microsoft.Agents.AI.GitHub.Copilot/   # GitHub Copilot provider
└── Shared/                               # Shared utilities
```

---

## Core Abstractions

### AIAgent Base Class

**File:** [dotnet/src/Microsoft.Agents.AI.Abstractions/AIAgent.cs](dotnet/src/Microsoft.Agents.AI.Abstractions/AIAgent.cs)

```csharp
public abstract class AIAgent
{
    // Identity
    public string Id { get; }           // Lines 37-41
    protected virtual string? IdCore => null;  // Lines 46-54
    public virtual string? Name { get; }       // Lines 59-65
    public virtual string? Description { get; } // Lines 70-76

    // Service Resolution
    public virtual object? GetService(Type serviceType, object? serviceKey = null); // Lines 82-91
    public TService? GetService<TService>(object? serviceKey = null); // Lines 97-99

    // Session Management
    public abstract ValueTask<AgentSession> GetNewSessionAsync(CancellationToken ct); // Lines 110-113
    public abstract ValueTask<AgentSession> DeserializeSessionAsync(JsonElement, JsonSerializerOptions?, CancellationToken); // Lines 121-125

    // Non-streaming Invocation
    public Task<AgentResponse> RunAsync(AgentSession? session, AgentRunOptions? options, CancellationToken ct); // Lines 158-163
    public Task<AgentResponse> RunAsync(string message, AgentSession?, AgentRunOptions?, CancellationToken); // Lines 180-189
    public Task<AgentResponse> RunAsync(ChatMessage message, AgentSession?, AgentRunOptions?, CancellationToken); // Lines 204-213
    public Task<AgentResponse> RunAsync(IEnumerable<ChatMessage> messages, AgentSession?, AgentRunOptions?, CancellationToken); // Lines 233-238
    protected abstract Task<AgentResponse> RunCoreAsync(IEnumerable<ChatMessage>, AgentSession?, AgentRunOptions?, CancellationToken); // Lines 257-261

    // Streaming Invocation
    public IAsyncEnumerable<AgentResponseUpdate> RunStreamingAsync(AgentSession?, AgentRunOptions?, CancellationToken); // Lines 274-278
    public IAsyncEnumerable<AgentResponseUpdate> RunStreamingAsync(string message, ...); // Lines 293-303
    public IAsyncEnumerable<AgentResponseUpdate> RunStreamingAsync(ChatMessage, ...); // Lines 317-326
    public IAsyncEnumerable<AgentResponseUpdate> RunStreamingAsync(IEnumerable<ChatMessage>, ...); // Lines 347-352
    protected abstract IAsyncEnumerable<AgentResponseUpdate> RunCoreStreamingAsync(IEnumerable<ChatMessage>, ...); // Lines 371-376
}
```

### DelegatingAIAgent Pattern

**File:** [dotnet/src/Microsoft.Agents.AI.Abstractions/DelegatingAIAgent.cs](dotnet/src/Microsoft.Agents.AI.Abstractions/DelegatingAIAgent.cs)

Decorator pattern base class for wrapping agents:

```csharp
public abstract class DelegatingAIAgent : AIAgent
{
    protected DelegatingAIAgent(AIAgent innerAgent); // Lines 34-37
    protected AIAgent InnerAgent { get; }  // Lines 45-49

    // All methods delegate to InnerAgent by default
    protected override Task<AgentResponse> RunCoreAsync(...) => InnerAgent.RunAsync(...); // Lines 81-86
    protected override IAsyncEnumerable<AgentResponseUpdate> RunCoreStreamingAsync(...) => InnerAgent.RunStreamingAsync(...); // Lines 89-94
}
```

### AgentSession

**File:** [dotnet/src/Microsoft.Agents.AI.Abstractions/AgentSession.cs](dotnet/src/Microsoft.Agents.AI.Abstractions/AgentSession.cs)

```csharp
public abstract class AgentSession
{
    public virtual JsonElement Serialize(JsonSerializerOptions? options); // Lines 57-58
    public virtual object? GetService(Type serviceType, object? serviceKey = null); // Lines 67-75
    public TService? GetService<TService>(object? serviceKey = null); // Lines 83-85
}
```

### AgentResponse and AgentResponseUpdate

**Files:**

- [dotnet/src/Microsoft.Agents.AI.Abstractions/AgentResponse.cs](dotnet/src/Microsoft.Agents.AI.Abstractions/AgentResponse.cs)
- [dotnet/src/Microsoft.Agents.AI.Abstractions/AgentResponseUpdate.cs](dotnet/src/Microsoft.Agents.AI.Abstractions/AgentResponseUpdate.cs)

```csharp
public class AgentResponse
{
    public IList<ChatMessage> Messages { get; set; }  // Lines 90-104
    public string Text { get; }                        // Concatenated text from messages
    public string? ResponseId { get; set; }
    public DateTimeOffset? CreatedAt { get; set; }
    public UsageDetails? Usage { get; set; }
    public object? RawRepresentation { get; set; }
    public ResponseContinuationToken? ContinuationToken { get; set; }

    public AgentResponse(ChatResponse response);      // Lines 57-77 - Wraps ChatResponse
}

public class AgentResponseUpdate
{
    public ChatRole? Role { get; set; }
    public string? AuthorName { get; set; }
    public IList<AIContent> Contents { get; set; }
    public string Text { get; }                        // Concatenated text from contents
    public string? ResponseId { get; set; }
    public DateTimeOffset? CreatedAt { get; set; }

    public AgentResponseUpdate(ChatResponseUpdate update); // Lines 63-77 - Wraps ChatResponseUpdate
}
```

---

## ChatClientAgent Implementation

### Core Implementation

**File:** [dotnet/src/Microsoft.Agents.AI/ChatClient/ChatClientAgent.cs](dotnet/src/Microsoft.Agents.AI/ChatClient/ChatClientAgent.cs) (910 lines)

#### Constructors

```csharp
// Simple constructor
public ChatClientAgent(
    IChatClient chatClient,
    string? instructions = null,
    string? name = null,
    string? description = null,
    IList<AITool>? tools = null,
    ILoggerFactory? loggerFactory = null,
    IServiceProvider? services = null); // Lines 57-68

// Options-based constructor
public ChatClientAgent(
    IChatClient chatClient,
    ChatClientAgentOptions? options,
    ILoggerFactory? loggerFactory = null,
    IServiceProvider? services = null); // Lines 90-106
```

#### Key Properties

```csharp
public IChatClient ChatClient { get; }    // Lines 113-119 - The underlying chat client
public string? Instructions => _agentOptions?.ChatOptions?.Instructions; // Lines 136-137
internal ChatOptions? ChatOptions => _agentOptions?.ChatOptions; // Line 142
```

#### Streaming Implementation Pattern

```csharp
protected override async IAsyncEnumerable<AgentResponseUpdate> RunCoreStreamingAsync(
    IEnumerable<ChatMessage> messages,
    AgentSession? session = null,
    AgentRunOptions? options = null,
    [EnumeratorCancellation] CancellationToken cancellationToken = default)
{
    // 1. Prepare session and messages (Lines 205-212)
    var (safeSession, chatOptions, inputMessagesForChatClient, aiContextProviderMessages, chatHistoryProviderMessages, continuationToken) =
        await PrepareSessionAndMessagesAsync(session, inputMessages, options, cancellationToken);

    // 2. Apply transformations (Line 217)
    chatClient = ApplyRunOptionsTransformations(options, chatClient);

    // 3. Get streaming response (Line 229)
    responseUpdatesEnumerator = chatClient.GetStreamingResponseAsync(inputMessagesForChatClient, chatOptions, cancellationToken).GetAsyncEnumerator();

    // 4. Iterate and yield updates (Lines 253-269)
    while (hasUpdates)
    {
        var update = responseUpdatesEnumerator.Current;
        update.AuthorName ??= this.Name;
        responseUpdates.Add(update);

        yield return new AgentResponseUpdate(update)
        {
            AgentId = this.Id,
            ContinuationToken = WrapContinuationToken(update.ContinuationToken, ...)
        };

        hasUpdates = await responseUpdatesEnumerator.MoveNextAsync();
    }

    // 5. Notify providers (Lines 283-290)
    await NotifyChatHistoryProviderOfNewMessagesAsync(...);
    await NotifyAIContextProviderOfSuccessAsync(...);
}
```

### ChatClientAgentOptions

**File:** [dotnet/src/Microsoft.Agents.AI/ChatClient/ChatClientAgentOptions.cs](dotnet/src/Microsoft.Agents.AI/ChatClient/ChatClientAgentOptions.cs)

```csharp
public sealed class ChatClientAgentOptions
{
    public string? Id { get; set; }                              // Line 24
    public string? Name { get; set; }                            // Line 29
    public string? Description { get; set; }                     // Line 34
    public ChatOptions? ChatOptions { get; set; }                // Line 39

    // Factory for chat history storage
    public Func<ChatHistoryProviderFactoryContext, CancellationToken, ValueTask<ChatHistoryProvider>>?
        ChatHistoryProviderFactory { get; set; }                 // Lines 44-46

    // Factory for context injection
    public Func<AIContextProviderFactoryContext, CancellationToken, ValueTask<AIContextProvider>>?
        AIContextProviderFactory { get; set; }                   // Lines 51-54

    // Disable default middleware wrapping
    public bool UseProvidedChatClientAsIs { get; set; }          // Lines 64-65

    public ChatClientAgentOptions Clone();                       // Lines 71-80
}
```

### ChatClientAgentSession

**File:** [dotnet/src/Microsoft.Agents.AI/ChatClient/ChatClientAgentSession.cs](dotnet/src/Microsoft.Agents.AI/ChatClient/ChatClientAgentSession.cs)

```csharp
public sealed class ChatClientAgentSession : AgentSession
{
    // For service-managed chat history
    public string? ConversationId { get; internal set; }  // Lines 28-66

    // For client-managed chat history
    public ChatHistoryProvider? ChatHistoryProvider { get; internal set; } // Lines 81-97

    // For context injection
    public AIContextProvider? AIContextProvider { get; internal set; }
}
```

---

## Provider Implementations

### OpenAI Provider

**Package:** `Microsoft.Agents.AI.OpenAI`

**Directory Structure:**

```
dotnet/src/Microsoft.Agents.AI.OpenAI/
├── ChatClient/
│   ├── AsyncStreamingChatCompletionUpdateCollectionResult.cs
│   ├── AsyncStreamingResponseUpdateCollectionResult.cs
│   └── StreamingUpdatePipelineResponse.cs
└── Extensions/
    ├── AgentResponseExtensions.cs
    ├── AIAgentWithOpenAIExtensions.cs
    ├── OpenAIAssistantClientExtensions.cs      # Deprecated
    ├── OpenAIChatClientExtensions.cs
    └── OpenAIResponseClientExtensions.cs
```

#### OpenAI Chat Completion Extensions

**File:** [dotnet/src/Microsoft.Agents.AI.OpenAI/Extensions/OpenAIChatClientExtensions.cs](dotnet/src/Microsoft.Agents.AI.OpenAI/Extensions/OpenAIChatClientExtensions.cs)

```csharp
public static class OpenAIChatClientExtensions
{
    // Simple agent creation
    public static ChatClientAgent AsAIAgent(
        this ChatClient client,
        string? instructions = null,
        string? name = null,
        string? description = null,
        IList<AITool>? tools = null,
        Func<IChatClient, IChatClient>? clientFactory = null,
        ILoggerFactory? loggerFactory = null,
        IServiceProvider? services = null); // Lines 36-55

    // Options-based agent creation
    public static ChatClientAgent AsAIAgent(
        this ChatClient client,
        ChatClientAgentOptions options,
        Func<IChatClient, IChatClient>? clientFactory = null,
        ILoggerFactory? loggerFactory = null,
        IServiceProvider? services = null); // Lines 69-84
}
```

#### OpenAI Response Client Extensions

**File:** [dotnet/src/Microsoft.Agents.AI.OpenAI/Extensions/OpenAIResponseClientExtensions.cs](dotnet/src/Microsoft.Agents.AI.OpenAI/Extensions/OpenAIResponseClientExtensions.cs)

```csharp
public static class OpenAIResponseClientExtensions
{
    public static ChatClientAgent AsAIAgent(
        this ResponsesClient client,
        string? instructions = null,
        string? name = null,
        string? description = null,
        IList<AITool>? tools = null,
        Func<IChatClient, IChatClient>? clientFactory = null,
        ILoggerFactory? loggerFactory = null,
        IServiceProvider? services = null); // Lines 37-62

    public static ChatClientAgent AsAIAgent(
        this ResponsesClient client,
        ChatClientAgentOptions options,
        Func<IChatClient, IChatClient>? clientFactory = null,
        ILoggerFactory? loggerFactory = null,
        IServiceProvider? services = null); // Lines 74-91
}
```

#### OpenAI Assistants Extensions (Deprecated)

**File:** [dotnet/src/Microsoft.Agents.AI.OpenAI/Extensions/OpenAIAssistantClientExtensions.cs](dotnet/src/Microsoft.Agents.AI.OpenAI/Extensions/OpenAIAssistantClientExtensions.cs)

```csharp
[Obsolete("The Assistants API has been deprecated. Please use the Responses API instead.")]
public static class OpenAIAssistantClientExtensions
{
    public static ChatClientAgent AsAIAgent(this AssistantClient client, ClientResult<Assistant> result, ...);
    public static ChatClientAgent AsAIAgent(this AssistantClient client, Assistant metadata, ...);
    public static Task<ChatClientAgent> GetAIAgentAsync(this AssistantClient client, string agentId, ...);
    public static Task<ChatClientAgent> CreateAIAgentAsync(this AssistantClient client, string name, ...);
}
```

---

### Azure AI Provider

**Package:** `Microsoft.Agents.AI.AzureAI`

**Directory Structure:**

```
dotnet/src/Microsoft.Agents.AI.AzureAI/
├── AzureAIProjectChatClient.cs
├── AzureAIProjectChatClientExtensions.cs
├── RequestOptionsExtensions.cs
└── Microsoft.Agents.AI.AzureAI.csproj
```

#### AzureAIProjectChatClient

**File:** [dotnet/src/Microsoft.Agents.AI.AzureAI/AzureAIProjectChatClient.cs](dotnet/src/Microsoft.Agents.AI.AzureAI/AzureAIProjectChatClient.cs)

Internal delegating chat client for Azure AI Project:

```csharp
internal sealed class AzureAIProjectChatClient : DelegatingChatClient
{
    private readonly AIProjectClient _agentClient;
    private readonly AgentReference _agentReference;
    private readonly ChatClientMetadata? _metadata;
    private readonly ChatOptions? _chatOptions;

    internal AzureAIProjectChatClient(
        AIProjectClient aiProjectClient,
        AgentReference agentReference,
        string? defaultModelId,
        ChatOptions? chatOptions); // Lines 36-50

    public override object? GetService(Type serviceType, object? serviceKey = null); // Lines 84-93

    public override async Task<ChatResponse> GetResponseAsync(...); // Lines 96-101
    public override async IAsyncEnumerable<ChatResponseUpdate> GetStreamingResponseAsync(...); // Lines 104-112
}
```

#### Azure AI Project Extensions

**File:** [dotnet/src/Microsoft.Agents.AI.AzureAI/AzureAIProjectChatClientExtensions.cs](dotnet/src/Microsoft.Agents.AI.AzureAI/AzureAIProjectChatClientExtensions.cs) (770 lines)

```csharp
public static partial class AzureAIProjectChatClientExtensions
{
    // Use existing agent by reference
    public static ChatClientAgent AsAIAgent(
        this AIProjectClient aiProjectClient,
        AgentReference agentReference,
        IList<AITool>? tools = null,
        Func<IChatClient, IChatClient>? clientFactory = null,
        IServiceProvider? services = null); // Lines 30-65

    // Get existing agent by name
    public static async Task<ChatClientAgent> GetAIAgentAsync(
        this AIProjectClient aiProjectClient,
        string name,
        IList<AITool>? tools = null,
        Func<IChatClient, IChatClient>? clientFactory = null,
        IServiceProvider? services = null,
        CancellationToken cancellationToken = default); // Lines 79-98

    // Use agent record
    public static ChatClientAgent AsAIAgent(
        this AIProjectClient aiProjectClient,
        AgentRecord agentRecord,
        IList<AITool>? tools = null,
        Func<IChatClient, IChatClient>? clientFactory = null,
        IServiceProvider? services = null); // Lines 114-130

    // Use agent version
    public static ChatClientAgent AsAIAgent(
        this AIProjectClient aiProjectClient,
        AgentVersion agentVersion,
        IList<AITool>? tools = null,
        Func<IChatClient, IChatClient>? clientFactory = null,
        IServiceProvider? services = null); // Lines 143-162

    // Create new agent
    public static Task<ChatClientAgent> CreateAIAgentAsync(
        this AIProjectClient aiProjectClient,
        string name,
        string model,
        string instructions,
        string? description = null,
        IList<AITool>? tools = null,
        Func<IChatClient, IChatClient>? clientFactory = null,
        IServiceProvider? services = null,
        CancellationToken cancellationToken = default); // Lines 222-249
}
```

---

### Azure AI Persistent Provider

**Package:** `Microsoft.Agents.AI.AzureAI.Persistent`

**File:** [dotnet/src/Microsoft.Agents.AI.AzureAI.Persistent/PersistentAgentsClientExtensions.cs](dotnet/src/Microsoft.Agents.AI.AzureAI.Persistent/PersistentAgentsClientExtensions.cs) (430 lines)

```csharp
public static class PersistentAgentsClientExtensions
{
    public static ChatClientAgent AsAIAgent(
        this PersistentAgentsClient persistentAgentsClient,
        Response<PersistentAgent> persistentAgentResponse,
        ChatOptions? chatOptions = null,
        Func<IChatClient, IChatClient>? clientFactory = null,
        IServiceProvider? services = null); // Lines 22-35

    public static ChatClientAgent AsAIAgent(
        this PersistentAgentsClient persistentAgentsClient,
        PersistentAgent persistentAgentMetadata,
        ChatOptions? chatOptions = null,
        Func<IChatClient, IChatClient>? clientFactory = null,
        IServiceProvider? services = null); // Lines 46-82

    public static async Task<ChatClientAgent> GetAIAgentAsync(
        this PersistentAgentsClient persistentAgentsClient,
        string agentId,
        ChatOptions? chatOptions = null,
        Func<IChatClient, IChatClient>? clientFactory = null,
        IServiceProvider? services = null,
        CancellationToken cancellationToken = default); // Lines 93-115
}
```

---

### Anthropic Provider

**Package:** `Microsoft.Agents.AI.Anthropic`

**Directory Structure:**

```
dotnet/src/Microsoft.Agents.AI.Anthropic/
├── AnthropicBetaServiceExtensions.cs
├── AnthropicClientExtensions.cs
├── AnthropicClientJsonContext.cs
└── Microsoft.Agents.AI.Anthropic.csproj
```

#### Anthropic Client Extensions

**File:** [dotnet/src/Microsoft.Agents.AI.Anthropic/AnthropicClientExtensions.cs](dotnet/src/Microsoft.Agents.AI.Anthropic/AnthropicClientExtensions.cs)

```csharp
public static class AnthropicClientExtensions
{
    public static int DefaultMaxTokens { get; set; } = 4096;

    public static ChatClientAgent AsAIAgent(
        this IAnthropicClient client,
        string model,
        string? instructions = null,
        string? name = null,
        string? description = null,
        IList<AITool>? tools = null,
        int? defaultMaxTokens = null,
        Func<IChatClient, IChatClient>? clientFactory = null,
        ILoggerFactory? loggerFactory = null,
        IServiceProvider? services = null); // Lines 31-71

    public static ChatClientAgent AsAIAgent(
        this IAnthropicClient client,
        ChatClientAgentOptions options,
        Func<IChatClient, IChatClient>? clientFactory = null,
        ILoggerFactory? loggerFactory = null,
        IServiceProvider? services = null); // Lines 82-99
}
```

#### Anthropic Beta Service Extensions

**File:** [dotnet/src/Microsoft.Agents.AI.Anthropic/AnthropicBetaServiceExtensions.cs](dotnet/src/Microsoft.Agents.AI.Anthropic/AnthropicBetaServiceExtensions.cs)

Similar pattern for `IBetaService` interface.

---

## Tool/Function System

### Tool Integration Pattern

The framework uses `Microsoft.Extensions.AI` abstractions:

- `AITool` - Base abstraction for tools
- `AIFunction` - Function-based tools
- `FunctionInvokingChatClient` - Middleware for automatic function invocation

#### Default Middleware Application

**File:** [dotnet/src/Microsoft.Agents.AI/ChatClient/ChatClientExtensions.cs](dotnet/src/Microsoft.Agents.AI/ChatClient/ChatClientExtensions.cs)

```csharp
internal static IChatClient WithDefaultAgentMiddleware(
    this IChatClient chatClient,
    ChatClientAgentOptions? options,
    IServiceProvider? services = null)
{
    var chatBuilder = chatClient.AsBuilder();

    // Add FunctionInvokingChatClient if not already present
    if (chatClient.GetService<FunctionInvokingChatClient>() is null)
    {
        chatBuilder.Use((innerClient, services) =>
        {
            var loggerFactory = services.GetService<ILoggerFactory>();
            return new FunctionInvokingChatClient(innerClient, loggerFactory, services);
        });
    }

    var agentChatClient = chatBuilder.Build(services);

    // Register tools from options
    if (options?.ChatOptions?.Tools is { Count: > 0 })
    {
        var functionService = agentChatClient.GetService<FunctionInvokingChatClient>();
        functionService!.AdditionalTools = options.ChatOptions.Tools;
    }

    return agentChatClient;
}
```

### Agent as Function

**File:** [dotnet/src/Microsoft.Agents.AI/AgentExtensions.cs](dotnet/src/Microsoft.Agents.AI/AgentExtensions.cs)

```csharp
public static AIFunction AsAIFunction(
    this AIAgent agent,
    AIFunctionFactoryOptions? options = null,
    AgentSession? session = null)
{
    Throw.IfNull(agent);

    [Description("Invoke an agent to retrieve some information.")]
    async Task<string> InvokeAgentAsync(
        [Description("Input query to invoke the agent.")] string query,
        CancellationToken cancellationToken)
    {
        // Propagate additional properties from parent agent
        AgentRunOptions? agentRunOptions = FunctionInvokingChatClient.CurrentContext?.Options?.AdditionalProperties is AdditionalPropertiesDictionary dict
            ? new AgentRunOptions { AdditionalProperties = dict }
            : null;

        var response = await agent.RunAsync(query, session: session, options: agentRunOptions, cancellationToken: cancellationToken);
        return response.Text;
    }

    options ??= new();
    options.Name ??= SanitizeAgentName(agent.Name);
    options.Description ??= agent.Description;

    return AIFunctionFactory.Create(InvokeAgentAsync, options);
}
```

### Function Invocation Middleware

**File:** [dotnet/src/Microsoft.Agents.AI/FunctionInvocationDelegatingAgent.cs](dotnet/src/Microsoft.Agents.AI/FunctionInvocationDelegatingAgent.cs)

```csharp
internal sealed class FunctionInvocationDelegatingAgent : DelegatingAIAgent
{
    private readonly Func<AIAgent, FunctionInvocationContext, Func<FunctionInvocationContext, CancellationToken, ValueTask<object?>>, CancellationToken, ValueTask<object?>> _delegateFunc;

    // Wraps tools with middleware
    private AgentRunOptions? AgentRunOptionsWithFunctionMiddleware(AgentRunOptions? options)
    {
        aco.ChatClientFactory = chatClient =>
        {
            var builder = chatClient.AsBuilder();
            return builder.ConfigureOptions(co
                => co.Tools = co.Tools?.Select(tool => tool is AIFunction aiFunction
                        ? new MiddlewareEnabledFunction(this.InnerAgent, aiFunction, this._delegateFunc)
                        : tool)
                    .ToList())
                .Build();
        };
        return options;
    }
}
```

---

## Observability

### OpenTelemetry Integration

**File:** [dotnet/src/Microsoft.Agents.AI/OpenTelemetryAgent.cs](dotnet/src/Microsoft.Agents.AI/OpenTelemetryAgent.cs)

```csharp
public sealed class OpenTelemetryAgent : DelegatingAIAgent, IDisposable
{
    private readonly OpenTelemetryChatClient _otelClient;
    private readonly string? _providerName;

    public OpenTelemetryAgent(AIAgent innerAgent, string? sourceName = null) : base(innerAgent)
    {
        _providerName = innerAgent.GetService<AIAgentMetadata>()?.ProviderName;

        _otelClient = new OpenTelemetryChatClient(
            new ForwardingChatClient(this),
            sourceName: string.IsNullOrEmpty(sourceName) ? OpenTelemetryConsts.DefaultSourceName : sourceName!);
    }

    // Control sensitive data logging
    public bool EnableSensitiveData
    {
        get => _otelClient.EnableSensitiveData;
        set => _otelClient.EnableSensitiveData = value;
    }

    // Augments activity with agent-specific tags
    private void UpdateCurrentActivity(Activity? previousActivity)
    {
        if (Activity.Current is not { } activity) return;

        activity.DisplayName = $"invoke_agent {this.Name}({this.Id})";
        activity.SetTag("gen_ai.operation.name", "invoke_agent");
        activity.SetTag("gen_ai.agent.id", this.Id);
        activity.SetTag("gen_ai.agent.name", this.Name);
        activity.SetTag("gen_ai.agent.description", this.Description);
        activity.SetTag("gen_ai.provider.name", _providerName);
    }
}
```

### OpenTelemetry Constants

**File:** [dotnet/src/Microsoft.Agents.AI/OpenTelemetryConsts.cs](dotnet/src/Microsoft.Agents.AI/OpenTelemetryConsts.cs)

```csharp
internal static class OpenTelemetryConsts
{
    public const string DefaultSourceName = "Experimental.Microsoft.Agents.AI";

    public static class GenAI
    {
        public const string InvokeAgent = "invoke_agent";

        public static class Agent
        {
            public const string Id = "gen_ai.agent.id";
            public const string Name = "gen_ai.agent.name";
            public const string Description = "gen_ai.agent.description";
        }

        public static class Operation { public const string Name = "gen_ai.operation.name"; }
        public static class Provider { public const string Name = "gen_ai.provider.name"; }
    }
}
```

### OpenTelemetry Builder Extension

**File:** [dotnet/src/Microsoft.Agents.AI/OpenTelemetryAgentBuilderExtensions.cs](dotnet/src/Microsoft.Agents.AI/OpenTelemetryAgentBuilderExtensions.cs)

```csharp
public static AIAgentBuilder UseOpenTelemetry(this AIAgentBuilder builder, string? sourceName = null)
{
    return builder.Use(innerAgent => new OpenTelemetryAgent(innerAgent, sourceName));
}
```

### Logging Agent

**File:** [dotnet/src/Microsoft.Agents.AI/LoggingAgent.cs](dotnet/src/Microsoft.Agents.AI/LoggingAgent.cs)

```csharp
public sealed partial class LoggingAgent : DelegatingAIAgent
{
    private readonly ILogger _logger;
    private JsonSerializerOptions _jsonSerializerOptions;

    public LoggingAgent(AIAgent innerAgent, ILogger logger) : base(innerAgent)
    {
        _logger = Throw.IfNull(logger);
        _jsonSerializerOptions = AgentJsonUtilities.DefaultOptions;
    }

    // Logs at Debug level, with Trace including message content
    protected override async Task<AgentResponse> RunCoreAsync(...)
    {
        if (_logger.IsEnabled(LogLevel.Debug))
        {
            if (_logger.IsEnabled(LogLevel.Trace))
            {
                LogInvokedSensitive(nameof(RunAsync), AsJson(messages), AsJson(options), ...);
            }
            else
            {
                LogInvoked(nameof(RunAsync));
            }
        }
        // ...
    }
}
```

---

## Chat History Provider Pattern

**File:** [dotnet/src/Microsoft.Agents.AI.Abstractions/ChatHistoryProvider.cs](dotnet/src/Microsoft.Agents.AI.Abstractions/ChatHistoryProvider.cs)

```csharp
public abstract class ChatHistoryProvider
{
    // Called at start of agent invocation - provide historical messages
    public abstract ValueTask<IEnumerable<ChatMessage>> InvokingAsync(
        InvokingContext context,
        CancellationToken cancellationToken = default);

    // Called at end of agent invocation - store new messages
    public abstract ValueTask InvokedAsync(
        InvokedContext context,
        CancellationToken cancellationToken = default);

    // Serialization for persistence
    public virtual JsonElement Serialize(JsonSerializerOptions? options);

    public sealed class InvokingContext
    {
        public IEnumerable<ChatMessage> RequestMessages { get; }
    }

    public sealed class InvokedContext
    {
        public IEnumerable<ChatMessage> RequestMessages { get; }
        public IEnumerable<ChatMessage> HistoryMessages { get; }
        public IEnumerable<ChatMessage>? ResponseMessages { get; set; }
        public IEnumerable<ChatMessage>? AIContextProviderMessages { get; set; }
        public Exception? InvokeException { get; set; }
    }
}
```

---

## AI Context Provider Pattern

**File:** [dotnet/src/Microsoft.Agents.AI.Abstractions/AIContextProvider.cs](dotnet/src/Microsoft.Agents.AI.Abstractions/AIContextProvider.cs)

```csharp
public abstract class AIContextProvider
{
    // Called at start of invocation - inject additional context
    public abstract ValueTask<AIContext> InvokingAsync(
        InvokingContext context,
        CancellationToken cancellationToken = default);

    // Called at end of invocation - process results
    public virtual ValueTask InvokedAsync(
        InvokedContext context,
        CancellationToken cancellationToken = default);

    public virtual JsonElement Serialize(JsonSerializerOptions? options);

    public sealed class InvokingContext
    {
        public IEnumerable<ChatMessage> RequestMessages { get; }
    }

    public sealed class InvokedContext
    {
        public IEnumerable<ChatMessage> RequestMessages { get; }
        public IEnumerable<ChatMessage>? AIContextProviderMessages { get; }
        public IEnumerable<ChatMessage>? ResponseMessages { get; set; }
        public Exception? InvokeException { get; set; }
    }
}

public sealed class AIContext
{
    public IList<ChatMessage>? Messages { get; set; }
    public IList<AITool>? Tools { get; set; }
    public string? Instructions { get; set; }
}
```

---

## Structured Output Support

**File:** [dotnet/src/Microsoft.Agents.AI/ChatClient/ChatClientAgentStructuredOutput.cs](dotnet/src/Microsoft.Agents.AI/ChatClient/ChatClientAgentStructuredOutput.cs)

```csharp
public sealed partial class ChatClientAgent
{
    public Task<ChatClientAgentResponse<T>> RunAsync<T>(
        AgentSession? session = null,
        JsonSerializerOptions? serializerOptions = null,
        AgentRunOptions? options = null,
        bool? useJsonSchemaResponseFormat = null,
        CancellationToken cancellationToken = default);

    public Task<ChatClientAgentResponse<T>> RunAsync<T>(
        string message,
        AgentSession? session = null,
        JsonSerializerOptions? serializerOptions = null,
        AgentRunOptions? options = null,
        bool? useJsonSchemaResponseFormat = null,
        CancellationToken cancellationToken = default);

    public Task<ChatClientAgentResponse<T>> RunAsync<T>(
        ChatMessage message,
        ...);

    public Task<ChatClientAgentResponse<T>> RunAsync<T>(
        IEnumerable<ChatMessage> messages,
        ...);
}
```

---

## Agent Builder Pattern

**File:** [dotnet/src/Microsoft.Agents.AI/AIAgentBuilder.cs](dotnet/src/Microsoft.Agents.AI/AIAgentBuilder.cs)

```csharp
public sealed class AIAgentBuilder
{
    private readonly Func<IServiceProvider, AIAgent> _innerAgentFactory;
    private List<Func<AIAgent, IServiceProvider, AIAgent>>? _agentFactories;

    public AIAgentBuilder(AIAgent innerAgent);
    public AIAgentBuilder(Func<IServiceProvider, AIAgent> innerAgentFactory);

    // Add middleware to the pipeline
    public AIAgentBuilder Use(Func<AIAgent, AIAgent> agentFactory);
    public AIAgentBuilder Use(Func<AIAgent, IServiceProvider, AIAgent> agentFactory);

    // Build the final agent pipeline
    public AIAgent Build(IServiceProvider? services = null);
}

// Extension to start building from an agent
public static AIAgentBuilder AsBuilder(this AIAgent innerAgent)
{
    return new AIAgentBuilder(innerAgent);
}
```

---

## Error Handling Patterns

### Session Error Handling

From [ChatClientAgent.cs](dotnet/src/Microsoft.Agents.AI/ChatClient/ChatClientAgent.cs) Lines 800-850:

```csharp
private static Task NotifyChatHistoryProviderOfFailureAsync(
    ChatClientAgentSession session,
    Exception ex,
    IEnumerable<ChatMessage> requestMessages,
    IEnumerable<ChatMessage>? chatHistoryProviderMessages,
    IEnumerable<ChatMessage>? aiContextProviderMessages,
    ChatOptions? chatOptions,
    CancellationToken cancellationToken)
{
    ChatHistoryProvider? provider = ResolveChatHistoryProvider(session, chatOptions);

    if (provider is not null)
    {
        var invokedContext = new ChatHistoryProvider.InvokedContext(requestMessages, chatHistoryProviderMessages!)
        {
            AIContextProviderMessages = aiContextProviderMessages,
            InvokeException = ex  // Exception is passed to provider
        };

        return provider.InvokedAsync(invokedContext, cancellationToken).AsTask();
    }

    return Task.CompletedTask;
}
```

### Session Type Validation

From [ChatClientAgent.cs](dotnet/src/Microsoft.Agents.AI/ChatClient/ChatClientAgent.cs) Lines 776-789:

```csharp
private async Task UpdateSessionWithTypeAndConversationIdAsync(
    ChatClientAgentSession session,
    string? responseConversationId,
    CancellationToken cancellationToken)
{
    if (string.IsNullOrWhiteSpace(responseConversationId) && !string.IsNullOrWhiteSpace(session.ConversationId))
    {
        // Service doesn't support service-managed history but session was configured for it
        throw new InvalidOperationException("Service did not return a valid conversation id when using an AgentSession with service managed chat history.");
    }
    // ...
}
```

---

## Sample Usage Patterns

### Simple Agent Creation (OpenAI)

```csharp
AIAgent agent = new OpenAIClient(apiKey)
    .GetChatClient("gpt-4o")
    .AsIChatClient()
    .AsAIAgent(instructions: "You are a helpful assistant.", name: "Assistant");

var response = await agent.RunAsync("Hello!");
Console.WriteLine(response.Text);
```

### Multi-Provider Workflow

```csharp
IChatClient aws = new AmazonBedrockRuntimeClient(...).AsIChatClient("amazon.nova-pro-v1:0");
IChatClient anthropic = new AnthropicClient(...).AsIChatClient("claude-sonnet-4-20250514");
IChatClient openai = new OpenAIClient(...).GetChatClient("gpt-4o-mini").AsIChatClient();

AIAgent researcher = new ChatClientAgent(aws, instructions: "Research the topic.", name: "researcher");
AIAgent factChecker = new ChatClientAgent(openai, instructions: "Verify claims.", name: "fact_checker");
AIAgent reporter = new ChatClientAgent(anthropic, instructions: "Summarize findings.", name: "reporter");

AIAgent workflowAgent = AgentWorkflowBuilder.BuildSequential(researcher, factChecker, reporter).AsAgent();
await foreach (var update in workflowAgent.RunStreamingAsync("Topic"))
{
    Console.Write(update.Text);
}
```

### Azure AI Project Agent

```csharp
var aiProjectClient = new AIProjectClient(new Uri(endpoint), new DefaultAzureCredential());

// Get existing agent
AIAgent agent = await aiProjectClient.GetAIAgentAsync(name: "MyAgent");

// Or create new agent
AIAgent agent = await aiProjectClient.CreateAIAgentAsync(
    name: "NewAgent",
    model: "gpt-4o",
    instructions: "You are a helpful assistant.");
```

### With OpenTelemetry

```csharp
AIAgent agent = new ChatClientAgent(chatClient, instructions: "...", name: "Agent")
    .AsBuilder()
    .UseOpenTelemetry()
    .Build();
```

---

## Key Design Patterns Summary

| Pattern | Implementation | Purpose |
|---------|---------------|---------|
| **Abstract Factory** | `AsAIAgent()` extensions | Provider-agnostic agent creation |
| **Decorator** | `DelegatingAIAgent` | Composable agent middleware |
| **Builder** | `AIAgentBuilder`, `ChatClientBuilder` | Fluent pipeline construction |
| **Strategy** | `ChatHistoryProvider`, `AIContextProvider` | Pluggable storage/context |
| **Adapter** | Provider extensions wrapping `IChatClient` | Unified interface across providers |
| **Template Method** | `RunCoreAsync`, `RunCoreStreamingAsync` | Extensible invocation logic |
