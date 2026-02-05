# .NET Middleware Implementation Deep Dive

**Date:** 2026-02-03
**Purpose:** Document implementation details for Go port parity

---

## 1. DelegatingAIAgent Base Class

**File:** [dotnet/src/Microsoft.Agents.AI.Abstractions/DelegatingAIAgent.cs](dotnet/src/Microsoft.Agents.AI.Abstractions/DelegatingAIAgent.cs)

### Overview

The `DelegatingAIAgent` implements the **decorator pattern** for AI agents, enabling composable middleware pipelines. It's an abstract class that:

- Extends `AIAgent`
- Wraps an inner `AIAgent` instance
- Provides transparent pass-through by default

### Overridden Methods

```csharp
public abstract class DelegatingAIAgent : AIAgent
{
    // Constructor - requires inner agent
    protected DelegatingAIAgent(AIAgent innerAgent)
    
    // Exposed inner agent for derived classes
    protected AIAgent InnerAgent { get; }
    
    // Property overrides - delegate to inner agent
    protected override string? IdCore => this.InnerAgent.Id;
    public override string? Name => this.InnerAgent.Name;
    public override string? Description => this.InnerAgent.Description;
    
    // Service resolution with self-awareness
    public override object? GetService(Type serviceType, object? serviceKey = null)
    
    // Session handling - pass through
    public override ValueTask<AgentSession> GetNewSessionAsync(CancellationToken)
    public override ValueTask<AgentSession> DeserializeSessionAsync(JsonElement, JsonSerializerOptions?, CancellationToken)
    
    // Core interception points - CRITICAL for middleware
    protected override Task<AgentResponse> RunCoreAsync(
        IEnumerable<ChatMessage> messages,
        AgentSession? session = null,
        AgentRunOptions? options = null,
        CancellationToken cancellationToken = default)
        => this.InnerAgent.RunAsync(messages, session, options, cancellationToken);
    
    protected override IAsyncEnumerable<AgentResponseUpdate> RunCoreStreamingAsync(
        IEnumerable<ChatMessage> messages,
        AgentSession? session = null,
        AgentRunOptions? options = null,
        CancellationToken cancellationToken = default)
        => this.InnerAgent.RunStreamingAsync(messages, session, options, cancellationToken);
}
```

### Key Pattern: Public vs Protected Methods

The `AIAgent` base class defines the interception pattern:

| Method Type | Access | Purpose |
|-------------|--------|---------|
| `RunAsync` | `public` | Entry point - cannot be overridden |
| `RunCoreAsync` | `protected abstract` | Implementation hook - override this |
| `RunStreamingAsync` | `public` | Entry point - cannot be overridden |
| `RunCoreStreamingAsync` | `protected abstract` | Implementation hook - override this |

**Critical:** Middleware overrides `RunCoreAsync` and `RunCoreStreamingAsync`, NOT the public methods.

---

## 2. All DelegatingAIAgent Derived Classes

### Production Classes (in src/)

| Class | Location | Purpose | Visibility |
|-------|----------|---------|------------|
| `AnonymousDelegatingAIAgent` | [Microsoft.Agents.AI](dotnet/src/Microsoft.Agents.AI/AnonymousDelegatingAIAgent.cs) | Lambda-based middleware | `internal sealed` |
| `FunctionInvocationDelegatingAgent` | [Microsoft.Agents.AI](dotnet/src/Microsoft.Agents.AI/FunctionInvocationDelegatingAgent.cs) | Tool call interception | `internal sealed` |
| `LoggingAgent` | [Microsoft.Agents.AI](dotnet/src/Microsoft.Agents.AI/LoggingAgent.cs) | Debug/trace logging | `public sealed` |
| `OpenTelemetryAgent` | [Microsoft.Agents.AI](dotnet/src/Microsoft.Agents.AI/OpenTelemetryAgent.cs) | OpenTelemetry instrumentation | `public sealed` |
| `AIHostAgent` | [Microsoft.Agents.AI.Hosting](dotnet/src/Microsoft.Agents.AI.Hosting/AIHostAgent.cs) | Session persistence | `public` |

### Sample Classes (in samples/)

| Class | Sample | Purpose |
|-------|--------|---------|
| `WeatherForecastAgent` | M365Agent | Weather integration |
| `ServerFunctionApprovalAgent` | AGUI Step04 | Human-in-loop (server) |
| `ServerFunctionApprovalClientAgent` | AGUI Step04 | Human-in-loop (client) |
| `SharedStateAgent` | AGUI Step05 | State management |
| `OpenAIChatClientAgent` | AgentWithOpenAI | Direct OpenAI chat |
| `OpenAIResponseClientAgent` | AgentWithOpenAI | OpenAI responses |
| `AgenticUIAgent` | AGUIDojoServer | Agentic UI patterns |
| `PredictiveStateUpdatesAgent` | AGUIDojoServer | Predictive state |

---

## 3. Method Signatures for Interception

### RunCoreAsync Signature

```csharp
protected override Task<AgentResponse> RunCoreAsync(
    IEnumerable<ChatMessage> messages,
    AgentSession? session = null,
    AgentRunOptions? options = null,
    CancellationToken cancellationToken = default)
```

**Parameters:**

- `messages`: Input messages to process
- `session`: Conversation state (nullable)
- `options`: Runtime configuration (nullable, can be subclassed)
- `cancellationToken`: Cancellation support

**Return:** `Task<AgentResponse>` - Complete response

### RunCoreStreamingAsync Signature

```csharp
protected override IAsyncEnumerable<AgentResponseUpdate> RunCoreStreamingAsync(
    IEnumerable<ChatMessage> messages,
    AgentSession? session = null,
    AgentRunOptions? options = null,
    CancellationToken cancellationToken = default)
```

**Return:** `IAsyncEnumerable<AgentResponseUpdate>` - Streaming updates

---

## 4. Interception Patterns

### Pattern 1: Simple Pass-Through with Logging

**Example:** `LoggingAgent`

```csharp
protected override async Task<AgentResponse> RunCoreAsync(
    IEnumerable<ChatMessage> messages, 
    AgentSession? session = null, 
    AgentRunOptions? options = null, 
    CancellationToken cancellationToken = default)
{
    // PRE-PROCESSING
    if (this._logger.IsEnabled(LogLevel.Debug))
    {
        this.LogInvoked(nameof(RunAsync));
    }

    try
    {
        // DELEGATION - call base which calls inner agent
        AgentResponse response = await base.RunCoreAsync(messages, session, options, cancellationToken)
            .ConfigureAwait(false);

        // POST-PROCESSING
        if (this._logger.IsEnabled(LogLevel.Debug))
        {
            this.LogCompleted(nameof(RunAsync));
        }

        return response;
    }
    catch (OperationCanceledException)
    {
        this.LogInvocationCanceled(nameof(RunAsync));
        throw;
    }
    catch (Exception ex)
    {
        this.LogInvocationFailed(nameof(RunAsync), ex);
        throw;
    }
}
```

### Pattern 2: Options Modification

**Example:** `FunctionInvocationDelegatingAgent`

```csharp
protected override Task<AgentResponse> RunCoreAsync(
    IEnumerable<ChatMessage> messages, 
    AgentSession? session = null, 
    AgentRunOptions? options = null, 
    CancellationToken cancellationToken = default)
    => this.InnerAgent.RunAsync(
        messages, 
        session, 
        this.AgentRunOptionsWithFunctionMiddleware(options),  // Modified options!
        cancellationToken);

private AgentRunOptions? AgentRunOptionsWithFunctionMiddleware(AgentRunOptions? options)
{
    // Upgrade to ChatClientAgentRunOptions if needed
    if (options is null || options.GetType() == typeof(AgentRunOptions))
    {
        options = new ChatClientAgentRunOptions();
    }

    if (options is not ChatClientAgentRunOptions aco)
    {
        throw new NotSupportedException(...);
    }

    // Install middleware via ChatClientFactory
    var originalFactory = aco.ChatClientFactory;
    aco.ChatClientFactory = chatClient =>
    {
        var builder = chatClient.AsBuilder();
        if (originalFactory is not null)
        {
            builder.Use(originalFactory);
        }
        return builder.ConfigureOptions(co => 
            co.Tools = co.Tools?.Select(tool => 
                tool is AIFunction aiFunction
                    ? new MiddlewareEnabledFunction(...) // Wrap tools
                    : tool
            ).ToList()
        ).Build();
    };

    return options;
}
```

### Pattern 3: Streaming Interception with Channels

**Example:** `AnonymousDelegatingAIAgent` with shared func

```csharp
protected override IAsyncEnumerable<AgentResponseUpdate> RunCoreStreamingAsync(
    IEnumerable<ChatMessage> messages,
    AgentSession? session = null,
    AgentRunOptions? options = null,
    CancellationToken cancellationToken = default)
{
    var updates = Channel.CreateBounded<AgentResponseUpdate>(1);

    _ = ProcessAsync();
    async Task ProcessAsync()
    {
        Exception? error = null;
        try
        {
            await this._sharedFunc(messages, session, options, 
                async (messages, session, options, cancellationToken) =>
                {
                    // Forward streaming from inner agent through channel
                    await foreach (var update in this.InnerAgent.RunStreamingAsync(...))
                    {
                        await updates.Writer.WriteAsync(update, cancellationToken);
                    }
                }, 
                cancellationToken);
        }
        catch (Exception ex)
        {
            error = ex;
            throw;
        }
        finally
        {
            _ = updates.Writer.TryComplete(error);
        }
    }

    return updates.Reader.ReadAllAsync(cancellationToken);
}
```

### Pattern 4: Run-to-Stream / Stream-to-Run Fallback

**Example:** `AnonymousDelegatingAIAgent`

```csharp
// When only _runStreamingFunc is provided, implement Run via Stream
protected override Task<AgentResponse> RunCoreAsync(...)
{
    if (this._runFunc is not null)
    {
        return this._runFunc(messages, session, options, this.InnerAgent, cancellationToken);
    }
    else
    {
        // Convert streaming to single response
        return this._runStreamingFunc!(messages, session, options, this.InnerAgent, cancellationToken)
            .ToAgentResponseAsync(cancellationToken);
    }
}

// When only _runFunc is provided, implement Stream via Run
protected override IAsyncEnumerable<AgentResponseUpdate> RunCoreStreamingAsync(...)
{
    if (this._runStreamingFunc is not null)
    {
        return this._runStreamingFunc(messages, session, options, this.InnerAgent, cancellationToken);
    }
    else
    {
        return GetStreamingRunAsyncViaRunAsync(this._runFunc!(...));
        
        static async IAsyncEnumerable<AgentResponseUpdate> GetStreamingRunAsyncViaRunAsync(Task<AgentResponse> task)
        {
            AgentResponse response = await task;
            foreach (var update in response.ToAgentResponseUpdates())
            {
                yield return update;
            }
        }
    }
}
```

---

## 5. AgentRunOptions Inheritance

### Base Class: AgentRunOptions

```csharp
public class AgentRunOptions
{
    public ResponseContinuationToken? ContinuationToken { get; set; }
    public bool? AllowBackgroundResponses { get; set; }
    public AdditionalPropertiesDictionary? AdditionalProperties { get; set; }
}
```

### Derived: ChatClientAgentRunOptions

```csharp
public sealed class ChatClientAgentRunOptions : AgentRunOptions
{
    public ChatOptions? ChatOptions { get; set; }
    public Func<IChatClient, IChatClient>? ChatClientFactory { get; set; }
}
```

### Options Pass-Through Pattern

Middleware typically:

1. Receives `AgentRunOptions?` (could be base or derived)
2. Casts or upgrades to specific type if needed
3. Modifies properties
4. Passes to inner agent

---

## 6. AIAgentBuilder Integration

**File:** [dotnet/src/Microsoft.Agents.AI/AIAgentBuilder.cs](dotnet/src/Microsoft.Agents.AI/AIAgentBuilder.cs)

### Builder Use Methods

```csharp
// Factory-based - most flexible
public AIAgentBuilder Use(Func<AIAgent, AIAgent> agentFactory)
public AIAgentBuilder Use(Func<AIAgent, IServiceProvider, AIAgent> agentFactory)

// Shared delegate - same logic for Run and RunStream
public AIAgentBuilder Use(
    Func<IEnumerable<ChatMessage>, AgentSession?, AgentRunOptions?, 
         Func<IEnumerable<ChatMessage>, AgentSession?, AgentRunOptions?, CancellationToken, Task>, 
         CancellationToken, Task> sharedFunc)

// Separate delegates - different logic for Run vs RunStream
public AIAgentBuilder Use(
    Func<..., Task<AgentResponse>>? runFunc,
    Func<..., IAsyncEnumerable<AgentResponseUpdate>>? runStreamingFunc)
```

### Extension Methods for Middleware

| Extension | Creates |
|-----------|---------|
| `UseLogging()` | `LoggingAgent` |
| `UseOpenTelemetry()` | `OpenTelemetryAgent` |
| `Use(callback)` | `FunctionInvocationDelegatingAgent` |

### Build Order

```csharp
public AIAgent Build(IServiceProvider? services = null)
{
    var agent = this._innerAgentFactory(services);

    // Apply factories in REVERSE order
    // First added = outermost wrapper
    if (this._agentFactories is not null)
    {
        for (var i = this._agentFactories.Count - 1; i >= 0; i--)
        {
            agent = this._agentFactories[i](agent, services);
        }
    }

    return agent;
}
```

---

## 7. Go Port Recommendations

### Middleware Interface

```go
// AgentMiddleware wraps an agent with pre/post processing
type AgentMiddleware interface {
    Agent  // Embed base interface
    Inner() Agent
}

// DelegatingAgent provides default pass-through
type DelegatingAgent struct {
    inner Agent
}

func (d *DelegatingAgent) Run(ctx context.Context, messages []ChatMessage, 
    session *AgentSession, options *AgentRunOptions) (*AgentResponse, error) {
    return d.inner.Run(ctx, messages, session, options)
}

func (d *DelegatingAgent) RunStreaming(ctx context.Context, messages []ChatMessage,
    session *AgentSession, options *AgentRunOptions) (<-chan AgentResponseUpdate, error) {
    return d.inner.RunStreaming(ctx, messages, session, options)
}
```

### Builder Pattern

```go
type AgentBuilder struct {
    innerFactory func(sp ServiceProvider) Agent
    factories    []func(Agent, ServiceProvider) Agent
}

func (b *AgentBuilder) Use(factory func(Agent) Agent) *AgentBuilder {
    b.factories = append(b.factories, func(a Agent, _ ServiceProvider) Agent {
        return factory(a)
    })
    return b
}

func (b *AgentBuilder) Build(sp ServiceProvider) Agent {
    agent := b.innerFactory(sp)
    // Apply in reverse order
    for i := len(b.factories) - 1; i >= 0; i-- {
        agent = b.factories[i](agent, sp)
    }
    return agent
}
```

### Middleware Examples

```go
// Logging middleware
type LoggingMiddleware struct {
    *DelegatingAgent
    logger *slog.Logger
}

func (l *LoggingMiddleware) Run(ctx context.Context, ...) (*AgentResponse, error) {
    l.logger.Debug("RunAsync invoked")
    resp, err := l.DelegatingAgent.Run(ctx, ...)
    if err != nil {
        l.logger.Error("RunAsync failed", "error", err)
        return nil, err
    }
    l.logger.Debug("RunAsync completed")
    return resp, nil
}
```

---

## 8. Summary Table

| Feature | .NET Implementation | Go Equivalent |
|---------|---------------------|---------------|
| Base decorator | `DelegatingAIAgent` abstract class | `DelegatingAgent` struct with embedded interface |
| Override points | `RunCoreAsync`, `RunCoreStreamingAsync` | `Run`, `RunStreaming` methods |
| Inner access | `protected AIAgent InnerAgent` | `Inner() Agent` method or field |
| Options type | `AgentRunOptions` with subclasses | `AgentRunOptions` struct with embedded types |
| Anonymous middleware | `AnonymousDelegatingAIAgent` | Function-based middleware |
| Builder | `AIAgentBuilder` with `Use()` | `AgentBuilder` with `Use()` |
| Extension methods | `UseLogging()`, `UseOpenTelemetry()` | `WithLogging()`, `WithOTel()` functions |

---

## 9. Critical Implementation Notes

1. **Reverse Order Build:** Middleware added first becomes the outermost wrapper
2. **Options Upgrade:** Middleware may need to upgrade `AgentRunOptions` to a subclass
3. **Channel Pattern:** For streaming middleware, use channels to bridge async operations
4. **Fallback Conversion:** Support converting between Run and RunStreaming when only one is provided
5. **Error Propagation:** Ensure errors bubble up correctly through the middleware chain
6. **Service Resolution:** `GetService` should check self before delegating to inner
