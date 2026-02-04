# .NET DurableTask Implementation Analysis for Go Temporal.io Port

**Date**: 2026-02-04  
**Purpose**: Analyze the .NET DurableTask implementation for durable agents to inform Go Temporal.io port

## Executive Summary

The .NET DurableTask implementation provides a robust framework for building stateful, durable AI agents using the Durable Task technology stack. The architecture centers around:

1. **Durable Entities** for stateful agent execution and conversation history
2. **Durable Orchestrations** for long-running agent workflows
3. **Entity State Management** with versioned JSON serialization
4. **Azure Functions Integration** for HTTP/MCP tool triggers

## Package Structure

### Core Package: `Microsoft.Agents.AI.DurableTask`

Location: [dotnet/src/Microsoft.Agents.AI.DurableTask/](dotnet/src/Microsoft.Agents.AI.DurableTask/)

| File | Purpose | Temporal.io Equivalent |
|------|---------|----------------------|
| `DurableAIAgent.cs` | Durable agent wrapper for orchestrations | Workflow activity wrapper |
| `DurableAgentSession.cs` | Session persistence | Workflow context/state |
| `AgentEntity.cs` | Entity implementation (state + operations) | Workflow definition + signals |
| `AgentSessionId.cs` | Session identifier (agent name + key) | Workflow ID |
| `DurableAgentContext.cs` | Thread-static context for tools | Workflow context |
| `IDurableAgentClient.cs` | Client interface for signaling entities | Temporal client |
| `RunRequest.cs` | Request DTO with messages + options | Workflow input |
| `AgentRunHandle.cs` | Handle for polling response | Workflow handle |

### State Package: `State/`

Location: [dotnet/src/Microsoft.Agents.AI.DurableTask/State/](dotnet/src/Microsoft.Agents.AI.DurableTask/State/)

| File | Purpose |
|------|---------|
| `DurableAgentState.cs` | Root state container with schema version |
| `DurableAgentStateData.cs` | State data with conversation history + expiration |
| `DurableAgentStateEntry.cs` | Polymorphic base for request/response entries |
| `DurableAgentStateRequest.cs` | User/system request entry |
| `DurableAgentStateResponse.cs` | Agent response entry |
| `DurableAgentStateMessage.cs` | Single chat message |
| `DurableAgentStateContent.cs` | Polymorphic content types (text, function call, etc.) |

### Azure Functions Package: `Microsoft.Agents.AI.Hosting.AzureFunctions`

Location: [dotnet/src/Microsoft.Agents.AI.Hosting.AzureFunctions/](dotnet/src/Microsoft.Agents.AI.Hosting.AzureFunctions/)

| File | Purpose |
|------|---------|
| `FunctionsApplicationBuilderExtensions.cs` | DI configuration |
| `BuiltInFunctions.cs` | HTTP/Entity/MCP triggers |
| `DurableAgentFunctionMetadataTransformer.cs` | Auto-generates function metadata |

---

## Core Abstractions

### 1. DurableAIAgent (Orchestration Context)

**File**: [DurableAIAgent.cs](dotnet/src/Microsoft.Agents.AI.DurableTask/DurableAIAgent.cs)  
**Lines**: 1-254

The `DurableAIAgent` wraps an AI agent for use within durable orchestrations. It:
- Takes a `TaskOrchestrationContext` and agent name
- Generates deterministic session IDs using orchestration context
- Calls entity operations via `context.Entities.CallEntityAsync`

```csharp
public sealed class DurableAIAgent : AIAgent
{
    private readonly TaskOrchestrationContext _context;
    private readonly string _agentName;

    internal DurableAIAgent(TaskOrchestrationContext context, string agentName)
    {
        this._context = context;
        this._agentName = agentName;
    }

    protected override async Task<AgentResponse> RunCoreAsync(
        IEnumerable<ChatMessage> messages,
        AgentSession? session = null,
        AgentRunOptions? options = null,
        CancellationToken cancellationToken = default)
    {
        // ... validation ...
        
        RunRequest request = new([.. messages], responseFormat, enableToolCalls, enableToolNames)
        {
            OrchestrationId = this._context.InstanceId
        };

        return await this._context.Entities.CallEntityAsync<AgentResponse>(
            durableSession.SessionId,
            nameof(AgentEntity.Run),
            request);
    }
}
```

**Temporal.io Mapping**:
- `TaskOrchestrationContext` → `workflow.Context`
- `context.Entities.CallEntityAsync` → Signal to a separate workflow or update handler
- `context.NewGuid()` → `workflow.SideEffect` for deterministic ID generation

### 2. AgentEntity (Stateful Entity)

**File**: [AgentEntity.cs](dotnet/src/Microsoft.Agents.AI.DurableTask/AgentEntity.cs)  
**Lines**: 1-234

The `AgentEntity` is a Durable Entity that:
- Maintains conversation history in `DurableAgentState`
- Processes `Run` operations to invoke the underlying AI agent
- Supports TTL-based expiration with self-signaling

```csharp
internal class AgentEntity(IServiceProvider services, CancellationToken cancellationToken = default) 
    : TaskEntity<DurableAgentState>
{
    public async Task<AgentResponse> Run(RunRequest request)
    {
        AgentSessionId sessionId = this.Context.Id;
        AIAgent agent = this.GetAgent(sessionId);
        
        // Persist request to conversation history
        this.State.Data.ConversationHistory.Add(DurableAgentStateRequest.FromRunRequest(request));

        // Set context for tool access
        DurableAgentContext.SetCurrent(agentContext);
        
        try
        {
            // Run the agent with full conversation history
            IAsyncEnumerable<AgentResponseUpdate> responseStream = agentWrapper.RunStreamingAsync(
                this.State.Data.ConversationHistory.SelectMany(e => e.Messages).Select(m => m.ToChatMessage()),
                await agentWrapper.GetNewSessionAsync(cancellationToken).ConfigureAwait(false),
                options: null,
                this._cancellationToken);

            AgentResponse response = await responseStream.ToAgentResponseAsync(this._cancellationToken);

            // Persist response to conversation history
            this.State.Data.ConversationHistory.Add(
                DurableAgentStateResponse.FromResponse(request.CorrelationId, response));

            return response;
        }
        finally
        {
            DurableAgentContext.ClearCurrent();
        }
    }

    public void CheckAndDeleteIfExpired()
    {
        // TTL expiration check - deletes entity if expired
        if (currentTime >= expirationTime.Value)
        {
            this.State = null!;  // Deletes entity
        }
        else
        {
            // Reschedule deletion check
            this.Context.SignalEntity(this.Context.Id, nameof(CheckAndDeleteIfExpired), 
                options: new SignalEntityOptions { SignalTime = scheduledTime });
        }
    }
}
```

**Temporal.io Mapping**:
- `TaskEntity<TState>` → Workflow with state stored in workflow variables
- `this.State` → Workflow local variables persisted automatically
- Entity operations → Signals or Update handlers
- `SignalEntity` with `SignalTime` → `workflow.Sleep` + continue-as-new pattern

### 3. DurableAgentSession

**File**: [DurableAgentSession.cs](dotnet/src/Microsoft.Agents.AI.DurableTask/DurableAgentSession.cs)  
**Lines**: 1-68

Session wrapper containing `AgentSessionId` for serialization/deserialization.

```csharp
public sealed class DurableAgentSession : AgentSession
{
    internal AgentSessionId SessionId { get; }

    public override JsonElement Serialize(JsonSerializerOptions? jsonSerializerOptions = null)
    {
        return JsonSerializer.SerializeToElement(this, ...);
    }

    internal static DurableAgentSession Deserialize(JsonElement serializedSession, ...)
    {
        // Parse sessionId from JSON
        AgentSessionId sessionId = AgentSessionId.Parse(sessionIdString);
        return new DurableAgentSession(sessionId);
    }
}
```

### 4. AgentSessionId

**File**: [AgentSessionId.cs](dotnet/src/Microsoft.Agents.AI.DurableTask/AgentSessionId.cs)  
**Lines**: 1-170

Immutable identifier combining agent name + unique key:

```csharp
public readonly struct AgentSessionId : IEquatable<AgentSessionId>
{
    private const string EntityNamePrefix = "dafx-";
    
    public AgentSessionId(string name, string key)
    {
        this.Name = name;
        this._entityId = new EntityInstanceId(ToEntityName(name), key);
    }

    public string Name { get; }
    public string Key => this._entityId.Key;

    internal static string ToEntityName(string name) => $"{EntityNamePrefix}{name}";
    
    public static AgentSessionId WithRandomKey(string name) =>
        new(name, Guid.NewGuid().ToString("N"));
}
```

**Temporal.io Mapping**:
- Entity name prefix `dafx-{agentName}` → Workflow type ID pattern
- Session key → Workflow ID suffix
- Full ID: `dafx-{agentName}@{key}` → `{agentName}-{key}` workflow ID

### 5. DurableAgentContext

**File**: [DurableAgentContext.cs](dotnet/src/Microsoft.Agents.AI.DurableTask/DurableAgentContext.cs)  
**Lines**: 1-162

Thread-static context providing tools access to orchestration capabilities:

```csharp
public class DurableAgentContext
{
    private static readonly AsyncLocal<DurableAgentContext?> s_currentContext = new();

    public static DurableAgentContext Current => s_currentContext.Value ??
        throw new InvalidOperationException("No agent context found!");

    public TaskEntityContext EntityContext { get; }
    public DurableTaskClient Client { get; }
    public DurableAgentSession CurrentSession { get; }

    // Schedule new orchestration from within a tool
    public string ScheduleNewOrchestration(TaskName name, object? input = null, ...)
    {
        return this.EntityContext.ScheduleNewOrchestration(name, input, options);
    }

    // Raise event on orchestration
    public Task RaiseOrchestrationEventAsync(string instanceId, string eventName, object? eventData = null)
    {
        return this.Client.RaiseEventAsync(instanceId, eventName, eventData, ...);
    }
}
```

**Temporal.io Mapping**:
- `AsyncLocal<>` context → `workflow.Context()` passed through activity context
- `ScheduleNewOrchestration` → `client.ExecuteWorkflow` from activity
- `RaiseOrchestrationEventAsync` → `client.SignalWorkflow`

---

## State Persistence Pattern

### State Schema

**File**: [schemas/durable-agent-entity-state.json](schemas/durable-agent-entity-state.json)  
**Lines**: 1-218

The state schema defines:

```json
{
  "schemaVersion": "1.1.0",
  "data": {
    "conversationHistory": [
      {
        "$type": "request",
        "correlationId": "...",
        "createdAt": "...",
        "messages": [...]
      },
      {
        "$type": "response",
        "correlationId": "...",
        "usage": { "inputTokenCount": 595, "outputTokenCount": 63 },
        "messages": [...]
      }
    ],
    "expirationTimeUtc": "2026-02-18T00:00:00Z"
  }
}
```

### State Classes

**Root State** ([DurableAgentState.cs#L1-28](dotnet/src/Microsoft.Agents.AI.DurableTask/State/DurableAgentState.cs)):
```csharp
internal sealed class DurableAgentState
{
    public DurableAgentStateData Data { get; init; } = new();
    public string SchemaVersion { get; init; } = "1.1.0";
}
```

**State Data** ([DurableAgentStateData.cs#L1-32](dotnet/src/Microsoft.Agents.AI.DurableTask/State/DurableAgentStateData.cs)):
```csharp
internal sealed class DurableAgentStateData
{
    public IList<DurableAgentStateEntry> ConversationHistory { get; init; } = [];
    public DateTime? ExpirationTimeUtc { get; set; }
    [JsonExtensionData]
    public IDictionary<string, JsonElement>? ExtensionData { get; set; }
}
```

**Polymorphic Entries** ([DurableAgentStateEntry.cs#L1-44](dotnet/src/Microsoft.Agents.AI.DurableTask/State/DurableAgentStateEntry.cs)):
```csharp
[JsonPolymorphic(TypeDiscriminatorPropertyName = "$type")]
[JsonDerivedType(typeof(DurableAgentStateRequest), "request")]
[JsonDerivedType(typeof(DurableAgentStateResponse), "response")]
internal abstract class DurableAgentStateEntry
{
    public required string CorrelationId { get; init; }
    public required DateTimeOffset CreatedAt { get; init; }
    public IReadOnlyList<DurableAgentStateMessage> Messages { get; init; } = [];
}
```

**Temporal.io Mapping**:
- Entity state → Workflow local variables
- Schema versioning → Workflow versioning with `GetVersion`
- `JsonExtensionData` → Preserve unknown fields for forward compatibility

---

## Human-in-the-Loop Pattern

**Sample**: [05_AgentOrchestration_HITL](dotnet/samples/Durable/Agents/AzureFunctions/05_AgentOrchestration_HITL/)

### Orchestration Pattern

```csharp
[Function(nameof(RunOrchestrationAsync))]
public static async Task<object> RunOrchestrationAsync(
    [OrchestrationTrigger] TaskOrchestrationContext context)
{
    DurableAIAgent writerAgent = context.GetAgent("WriterAgent");
    AgentSession writerSession = await writerAgent.GetNewSessionAsync();

    // Generate content
    AgentResponse<GeneratedContent> writerResponse = await writerAgent.RunAsync<GeneratedContent>(
        message: $"Write a short article about '{input.Topic}'.",
        session: writerSession);

    while (iterationCount++ < input.MaxReviewAttempts)
    {
        // Notify user
        await context.CallActivityAsync(nameof(NotifyUserForApproval), content);

        // Wait for external event (human approval)
        HumanApprovalResponse humanResponse = await context.WaitForExternalEvent<HumanApprovalResponse>(
            eventName: "HumanApproval",
            timeout: TimeSpan.FromHours(input.ApprovalTimeoutHours));

        if (humanResponse.Approved)
        {
            await context.CallActivityAsync(nameof(PublishContent), content);
            return new { content = content.Content };
        }

        // Incorporate feedback and regenerate
        writerResponse = await writerAgent.RunAsync<GeneratedContent>(
            message: $"Rewrite incorporating feedback: {humanResponse.Feedback}",
            session: writerSession);
    }
}
```

### External Event Handling

```csharp
[Function(nameof(SendHumanApprovalAsync))]
public static async Task<HttpResponseData> SendHumanApprovalAsync(
    [HttpTrigger(...)] HttpRequestData req,
    string instanceId,
    [DurableClient] DurableTaskClient client)
{
    HumanApprovalResponse? approvalResponse = await req.ReadFromJsonAsync<HumanApprovalResponse>();
    await client.RaiseEventAsync(instanceId, "HumanApproval", approvalResponse);
    // ...
}
```

**Temporal.io Mapping**:
- `WaitForExternalEvent` → `workflow.GetSignalChannel` + `workflow.Await`
- `RaiseEventAsync` → `client.SignalWorkflow`
- Timeout handling → `workflow.AwaitWithTimeout` or selector pattern

---

## Client/Proxy Pattern

### DurableAIAgentProxy

**File**: [DurableAIAgentProxy.cs](dotnet/src/Microsoft.Agents.AI.DurableTask/DurableAIAgentProxy.cs)  
**Lines**: 1-78

External client proxy that signals entities and polls for responses:

```csharp
internal class DurableAIAgentProxy(string name, IDurableAgentClient agentClient) : AIAgent
{
    protected override async Task<AgentResponse> RunCoreAsync(...)
    {
        RunRequest request = new([.. messages], responseFormat, enableToolCalls, enableToolNames);
        AgentSessionId sessionId = durableSession.SessionId;

        AgentRunHandle agentRunHandle = await this._agentClient.RunAgentAsync(sessionId, request, cancellationToken);

        if (isFireAndForget)
        {
            return new AgentResponse();
        }

        return await agentRunHandle.ReadAgentResponseAsync(cancellationToken);
    }
}
```

### DefaultDurableAgentClient

**File**: [DefaultDurableAgentClient.cs](dotnet/src/Microsoft.Agents.AI.DurableTask/DefaultDurableAgentClient.cs)  
**Lines**: 1-32

```csharp
internal class DefaultDurableAgentClient(DurableTaskClient client, ILoggerFactory loggerFactory) : IDurableAgentClient
{
    public async Task<AgentRunHandle> RunAgentAsync(AgentSessionId sessionId, RunRequest request, ...)
    {
        await this._client.Entities.SignalEntityAsync(
            sessionId,
            nameof(AgentEntity.Run),
            request,
            cancellation: cancellationToken);

        return new AgentRunHandle(this._client, this._logger, sessionId, request.CorrelationId);
    }
}
```

### AgentRunHandle (Polling)

**File**: [AgentRunHandle.cs](dotnet/src/Microsoft.Agents.AI.DurableTask/AgentRunHandle.cs)  
**Lines**: 1-84

```csharp
internal sealed class AgentRunHandle
{
    public async Task<AgentResponse> ReadAgentResponseAsync(CancellationToken cancellationToken = default)
    {
        TimeSpan pollInterval = TimeSpan.FromMilliseconds(50);
        TimeSpan maxPollInterval = TimeSpan.FromSeconds(3);

        while (true)
        {
            EntityMetadata<DurableAgentState>? entityResponse = await this._client.Entities
                .GetEntityAsync<DurableAgentState>(this.SessionId, cancellation: cancellationToken);

            if (state?.Data.ConversationHistory is not null)
            {
                DurableAgentStateResponse? response = state.Data.ConversationHistory
                    .OfType<DurableAgentStateResponse>()
                    .FirstOrDefault(r => r.CorrelationId == this.CorrelationId);

                if (response is not null)
                {
                    return response.ToResponse();
                }
            }

            await Task.Delay(pollInterval, cancellationToken);
            pollInterval = TimeSpan.FromMilliseconds(
                Math.Min(pollInterval.TotalMilliseconds * 2, maxPollInterval.TotalMilliseconds));
        }
    }
}
```

**Temporal.io Mapping**:
- `SignalEntityAsync` → `client.SignalWorkflow`
- Polling for response → Use Temporal queries or update handlers instead
- Better pattern: Use `UpdateWithStartWorkflow` for synchronous request/response

---

## Registration and Configuration

### DurableAgentsOptions

**File**: [DurableAgentsOptions.cs](dotnet/src/Microsoft.Agents.AI.DurableTask/DurableAgentsOptions.cs)  
**Lines**: 1-145

```csharp
public sealed class DurableAgentsOptions
{
    private readonly Dictionary<string, Func<IServiceProvider, AIAgent>> _agentFactories = new(...);
    private readonly Dictionary<string, TimeSpan?> _agentTimeToLive = new(...);

    public TimeSpan? DefaultTimeToLive { get; set; } = TimeSpan.FromDays(14);
    public TimeSpan MinimumTimeToLiveSignalDelay { get; set; } = TimeSpan.FromMinutes(5);

    public DurableAgentsOptions AddAIAgentFactory(string name, Func<IServiceProvider, AIAgent> factory, TimeSpan? timeToLive = null)
    {
        this._agentFactories.Add(name, factory);
        if (timeToLive.HasValue)
        {
            this._agentTimeToLive[name] = timeToLive;
        }
        return this;
    }

    public DurableAgentsOptions AddAIAgent(AIAgent agent, TimeSpan? timeToLive = null)
    {
        this._agentFactories.Add(agent.Name, sp => agent);
        // ...
    }
}
```

### ServiceCollectionExtensions

**File**: [ServiceCollectionExtensions.cs](dotnet/src/Microsoft.Agents.AI.DurableTask/ServiceCollectionExtensions.cs)  
**Lines**: 1-100

```csharp
public static IServiceCollection ConfigureDurableAgents(
    this IServiceCollection services,
    Action<DurableAgentsOptions> configure,
    Action<IDurableTaskWorkerBuilder>? workerBuilder = null,
    Action<IDurableTaskClientBuilder>? clientBuilder = null)
{
    DurableAgentsOptions options = services.ConfigureDurableAgents(configure);

    services.AddDurableTaskWorker(builder =>
    {
        builder.AddTasks(registry =>
        {
            foreach (string name in options.GetAgentFactories().Keys)
            {
                registry.AddEntity<AgentEntity>(AgentSessionId.ToEntityName(name));
            }
        });
    });

    services.AddSingleton<IDurableAgentClient, DefaultDurableAgentClient>();
    return services;
}
```

---

## Azure Functions Integration

### FunctionsApplicationBuilderExtensions

**File**: [FunctionsApplicationBuilderExtensions.cs](dotnet/src/Microsoft.Agents.AI.Hosting.AzureFunctions/FunctionsApplicationBuilderExtensions.cs)  
**Lines**: 1-46

```csharp
public static FunctionsApplicationBuilder ConfigureDurableAgents(
    this FunctionsApplicationBuilder builder,
    Action<DurableAgentsOptions> configure)
{
    builder.Services.ConfigureDurableAgents(configure);
    
    // Auto-generate function metadata for HTTP/MCP/Entity triggers
    builder.Services.AddSingleton<IFunctionMetadataTransformer, DurableAgentFunctionMetadataTransformer>();

    // Middleware for built-in function execution
    builder.UseWhen<BuiltInFunctionExecutionMiddleware>(context =>
        context.FunctionDefinition.EntryPoint == BuiltInFunctions.RunAgentHttpFunctionEntryPoint ||
        context.FunctionDefinition.EntryPoint == BuiltInFunctions.RunAgentMcpToolFunctionEntryPoint ||
        context.FunctionDefinition.EntryPoint == BuiltInFunctions.RunAgentEntityFunctionEntryPoint);

    return builder;
}
```

### BuiltInFunctions

**File**: [BuiltInFunctions.cs](dotnet/src/Microsoft.Agents.AI.Hosting.AzureFunctions/BuiltInFunctions.cs)  
**Lines**: 1-120

```csharp
internal static class BuiltInFunctions
{
    // Entity trigger - invokes agent entity
    public static Task<string> InvokeAgentAsync(
        [DurableClient] DurableTaskClient client,
        string encodedEntityRequest,
        FunctionContext functionContext)
    {
        AgentEntity entity = new(combinedServiceProvider, functionContext.CancellationToken);
        return GrpcEntityRunner.LoadAndRunAsync(encodedEntityRequest, entity, combinedServiceProvider);
    }

    // HTTP trigger - creates or signals agent session
    public static async Task<HttpResponseData> RunAgentHttpAsync(
        [HttpTrigger] HttpRequestData req,
        [DurableClient] DurableTaskClient client,
        FunctionContext context)
    {
        // Parse request, create session ID
        AgentSessionId sessionId = new(agentName, threadIdValue ?? context.InvocationId);
        
        AIAgent agentProxy = client.AsDurableAgentProxy(context, agentName);
        
        if (waitForResponse)
        {
            AgentResponse response = await agentProxy.RunAsync(message, session);
            // Return response
        }
        else
        {
            await agentProxy.RunAsync(message, session, new DurableAgentRunOptions { IsFireAndForget = true });
            // Return accepted
        }
    }
}
```

---

## Temporal.io Mapping Summary

| .NET DurableTask Concept | Temporal.io Equivalent |
|-------------------------|------------------------|
| `TaskEntity<TState>` | Workflow with local state variables |
| Entity operations (Run, CheckAndDeleteIfExpired) | Signals + Update handlers |
| `TaskOrchestrationContext` | `workflow.Context` |
| `context.Entities.CallEntityAsync` | Child workflow or signal + query |
| `context.WaitForExternalEvent` | `workflow.GetSignalChannel().Receive()` |
| `client.RaiseEventAsync` | `client.SignalWorkflow` |
| `context.CallActivityAsync` | `workflow.ExecuteActivity` |
| `context.NewGuid()` | `workflow.SideEffect` |
| Entity TTL with self-signaling | `workflow.Sleep` + continue-as-new |
| State serialization | Workflow history (automatic) |
| Polling for response | Queries or Update handlers |

---

## Recommended Go Implementation Approach

### 1. Workflow-per-Session Pattern

Instead of using Durable Entities (not available in Temporal), use a workflow-per-session:

```go
type AgentSessionWorkflow struct {
    State AgentState
}

func (w *AgentSessionWorkflow) Run(ctx workflow.Context) error {
    // Handle signals for Run operations
    runChannel := workflow.GetSignalChannel(ctx, "run")
    
    for {
        var request RunRequest
        runChannel.Receive(ctx, &request)
        
        response := workflow.ExecuteActivity(ctx, RunAgentActivity, w.State, request).Get(ctx, &response)
        w.State.ConversationHistory = append(w.State.ConversationHistory, response)
    }
}
```

### 2. Update Handlers for Synchronous Responses

```go
func (w *AgentSessionWorkflow) Run(ctx workflow.Context) error {
    err := workflow.SetUpdateHandler(ctx, "run", func(ctx workflow.Context, request RunRequest) (AgentResponse, error) {
        var response AgentResponse
        err := workflow.ExecuteActivity(ctx, RunAgentActivity, w.State, request).Get(ctx, &response)
        if err == nil {
            w.State.ConversationHistory = append(w.State.ConversationHistory, response)
        }
        return response, err
    })
    
    // Wait for TTL expiration
    workflow.AwaitWithTimeout(ctx, w.State.TTL, func() bool { return false })
    return nil
}
```

### 3. State Serialization

Use the same JSON schema from [schemas/durable-agent-entity-state.json](schemas/durable-agent-entity-state.json) for Go structs.

---

## Open Questions

1. **Streaming Support**: The .NET implementation notes that "Streaming is not supported for durable agents" ([DurableAIAgent.cs#L118](dotnet/src/Microsoft.Agents.AI.DurableTask/DurableAIAgent.cs#L118)). Should Go support streaming via Temporal update handlers or signals?

2. **TTL Implementation**: The self-signaling pattern for TTL in entities works well. In Temporal, should we use:
   - `workflow.Sleep` + check expiration
   - Continue-as-new to reset history
   - Workflow idle timeout configuration

3. **Client Polling vs Updates**: The .NET proxy polls entity state for responses. Temporal Update handlers provide synchronous request/response - should we prefer that pattern?

4. **Multi-Agent Orchestrations**: The .NET HITL sample shows orchestrations calling multiple agents. How should this map to Temporal child workflows vs activities?

---

## References

- [DurableTask README](dotnet/src/Microsoft.Agents.AI.DurableTask/README.md)
- [State README](dotnet/src/Microsoft.Agents.AI.DurableTask/State/README.md)
- [TTL Documentation](docs/features/durable-agents/durable-agents-ttl.md)
- [Azure Functions Samples](dotnet/samples/Durable/Agents/AzureFunctions/README.md)
- [State Schema](schemas/durable-agent-entity-state.json)
