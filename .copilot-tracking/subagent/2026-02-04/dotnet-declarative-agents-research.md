# .NET Declarative Agents Implementation Analysis

**Date**: 2026-02-04  
**Objective**: Analyze the .NET Declarative agents implementation for YAML-based agent definitions.

---

## 1. Package Structure Overview

### 1.1 Microsoft.Agents.AI.Declarative

**Location**: [dotnet/src/Microsoft.Agents.AI.Declarative/](dotnet/src/Microsoft.Agents.AI.Declarative/)

Core files for declarative prompt-based agents:

| File | Purpose |
|------|---------|
| [AgentBotElementYaml.cs](dotnet/src/Microsoft.Agents.AI.Declarative/AgentBotElementYaml.cs) | YAML parsing and deserialization |
| [PromptAgentFactory.cs](dotnet/src/Microsoft.Agents.AI.Declarative/PromptAgentFactory.cs) | Abstract base factory for creating agents |
| [AggregatorPromptAgentFactory.cs](dotnet/src/Microsoft.Agents.AI.Declarative/AggregatorPromptAgentFactory.cs) | Aggregates multiple factories |
| [ChatClient/ChatClientPromptAgentFactory.cs](dotnet/src/Microsoft.Agents.AI.Declarative/ChatClient/ChatClientPromptAgentFactory.cs) | Creates ChatClientAgent instances |

**Extensions** ([dotnet/src/Microsoft.Agents.AI.Declarative/Extensions/](dotnet/src/Microsoft.Agents.AI.Declarative/Extensions/)):
- `YamlAgentFactoryExtensions.cs` - Extension method `CreateFromYamlAsync()`
- `PromptAgentExtensions.cs` - Converts `GptComponentMetadata` to `ChatOptions`
- `ModelOptionsExtensions.cs` - Model options conversion
- `FunctionToolExtensions.cs` - Function tool creation
- `McpServerToolExtensions.cs` - MCP server tool creation
- `WebSearchToolExtensions.cs` - Web search tool creation
- `FileSearchToolExtensions.cs` - File search tool creation
- `CodeInterpreterToolExtensions.cs` - Code interpreter tool creation

### 1.2 Microsoft.Agents.AI.Workflows.Declarative

**Location**: [dotnet/src/Microsoft.Agents.AI.Workflows.Declarative/](dotnet/src/Microsoft.Agents.AI.Workflows.Declarative/)

Core workflow components:

| File | Purpose |
|------|---------|
| [DeclarativeWorkflowBuilder.cs](dotnet/src/Microsoft.Agents.AI.Workflows.Declarative/DeclarativeWorkflowBuilder.cs) | Builds workflows from YAML definitions |
| [DeclarativeWorkflowOptions.cs](dotnet/src/Microsoft.Agents.AI.Workflows.Declarative/DeclarativeWorkflowOptions.cs) | Workflow configuration options |
| [DeclarativeWorkflowLanguage.cs](dotnet/src/Microsoft.Agents.AI.Workflows.Declarative/DeclarativeWorkflowLanguage.cs) | Code generation language enum (Python, CSharp, JavaScript) |
| [WorkflowAgentProvider.cs](dotnet/src/Microsoft.Agents.AI.Workflows.Declarative/WorkflowAgentProvider.cs) | Abstract base for workflow agent providers |

**Subdirectories**:
- `Interpreter/` - Workflow execution logic
- `ObjectModel/` - Action executors (InvokeAzureAgent, Question, SetVariable, etc.)
- `Kit/` - Core executor infrastructure
- `PowerFx/` - Power Fx formula evaluation
- `Events/` - Workflow event types
- `Entities/` - Entity extraction

---

## 2. YAML Schema Documentation

### 2.1 Prompt Agent Schema (kind: Prompt)

Root-level schema for declarative prompt agents:

```yaml
kind: Prompt                    # Required: "Prompt" for agent definitions
name: string                    # Required: Agent name
description: string             # Optional: Agent description
instructions: string            # Required: System instructions for the agent
additionalInstructions: string  # Optional: Extra instructions
model:                          # Required: Model configuration
    id: string                  # Required: Model identifier (e.g., "gpt-4o")
    provider: string            # Optional: "AzureOpenAI", "OpenAI"
    apiType: string             # Optional: "Chat", "Assistants", "Responses"
    options:                    # Optional: Model options
        temperature: number     # 0.0 - 2.0
        topP: number            # 0.0 - 1.0
        topK: number            # Integer
        maxOutputTokens: number # Integer
        frequencyPenalty: number
        presencePenalty: number
        seed: number
        responseFormat: string  # "text", "json_object"
        stopSequences: string[]
        allowMultipleToolCalls: boolean
        chatToolMode: string    # "auto", "none", "require_any", or specific tool name
    connection:                 # Optional: Connection configuration
        kind: string            # "ApiKey", "Remote", "AnonymousConnection"
        endpoint: string        # API endpoint URL
        key: string             # API key (for ApiKey kind)
tools: []                       # Optional: Array of tool definitions
inputSchema:                    # Optional: Input schema definition
    properties: {}
outputSchema:                   # Optional: Output schema for structured responses
    properties: {}
template:                       # Optional: Template configuration
    format: string              # "PowerFx", "Mustache"
    parser: string              # "None", "Prompty", "XML"
```

### 2.2 Tool Definitions

#### Function Tool

```yaml
- kind: function
  name: string                  # Required: Function name
  description: string           # Required: Function description
  bindings:                     # Optional: Function bindings
    function_name: implementation_name
  parameters:                   # Required: Parameter definitions
    properties:
      paramName:
        kind: string            # "string", "number", "boolean", "object", "array"
        description: string
        required: boolean
        enum: []                # Optional: Allowed values
```

#### MCP Server Tool

```yaml
- kind: mcp
  name: string                  # Server name (legacy: use serverName)
  serverName: string            # Server identifier
  serverDescription: string     # Server description
  url: string                   # Server URL (legacy: use connection)
  connection:
    kind: AnonymousConnection
    endpoint: string            # MCP server endpoint
  allowedTools: []              # List of allowed tool names
  approvalMode:
    kind: string                # "never", "always", or "HostedMcpServerToolRequireSpecificApprovalMode"
    AlwaysRequireApprovalToolNames: []
    NeverRequireApprovalToolNames: []
```

#### Web Search Tool

```yaml
- kind: webSearch
  name: string
  description: string
```

#### File Search Tool

```yaml
- kind: fileSearch
  name: string
  description: string
  ranker: string                # "default"
  scoreThreshold: number
  maxResults: number
  maxContentLength: number
  vectorStoreIds: []            # Array of vector store IDs
```

#### Code Interpreter Tool

```yaml
- kind: codeInterpreter
  inputs:
    - kind: HostedFileContent
      FileId: string
```

### 2.3 Workflow Schema (kind: Workflow)

```yaml
kind: Workflow
trigger:
  kind: OnConversationStart     # Trigger type
  id: string                    # Workflow identifier
  actions: []                   # Array of action definitions
```

#### Workflow Actions

| Action Kind | Description |
|-------------|-------------|
| `InvokeAzureAgent` | Invoke an Azure AI Foundry agent |
| `SetVariable` | Set a variable value |
| `SendActivity` | Send a message to the user |
| `ConditionGroup` | Conditional branching |
| `GotoAction` | Jump to another action |
| `Foreach` | Loop over a collection |
| `Question` | Request human input |
| `CreateConversation` | Create a new conversation |
| `EndWorkflow` | Terminate the workflow |

---

## 3. Loader Implementation

### 3.1 YAML Parsing Flow

**Entry Point**: [AgentBotElementYaml.FromYaml()](dotnet/src/Microsoft.Agents.AI.Declarative/AgentBotElementYaml.cs#L26-L40)

```csharp
public static GptComponentMetadata FromYaml(string text, IConfiguration? configuration = null)
{
    Throw.IfNullOrEmpty(text);

    using var yamlReader = new StringReader(text);
    BotElement rootElement = YamlSerializer.Deserialize<BotElement>(yamlReader) 
        ?? throw new InvalidDataException("Text does not contain a valid agent definition.");

    if (rootElement is not GptComponentMetadata promptAgent)
    {
        throw new InvalidDataException($"Unsupported root element: {rootElement.GetType().Name}. Expected an {nameof(GptComponentMetadata)}.");
    }

    var botDefinition = WrapPromptAgentWithBot(promptAgent, configuration);
    return botDefinition.Descendants().OfType<GptComponentMetadata>().First();
}
```

**Workflow Parsing**: [DeclarativeWorkflowBuilder.ReadWorkflow()](dotnet/src/Microsoft.Agents.AI.Workflows.Declarative/DeclarativeWorkflowBuilder.cs#L131-L141)

```csharp
private static AdaptiveDialog ReadWorkflow(TextReader yamlReader)
{
    BotElement rootElement = YamlSerializer.Deserialize<BotElement>(yamlReader) 
        ?? throw new DeclarativeModelException("Workflow undefined.");

    if (rootElement is not AdaptiveDialog workflowElement)
    {
        throw new DeclarativeModelException($"Unsupported root element: {rootElement.GetType().Name}. Expected an {nameof(Workflow)}.");
    }

    return workflowElement;
}
```

### 3.2 Extension Method for Agent Creation

**File**: [YamlAgentFactoryExtensions.cs](dotnet/src/Microsoft.Agents.AI.Declarative/Extensions/YamlAgentFactoryExtensions.cs#L15-L29)

```csharp
public static Task<AIAgent> CreateFromYamlAsync(
    this PromptAgentFactory agentFactory, 
    string agentYaml, 
    CancellationToken cancellationToken = default)
{
    Throw.IfNull(agentFactory);
    Throw.IfNullOrEmpty(agentYaml);

    var agentDefinition = AgentBotElementYaml.FromYaml(agentYaml);

    return agentFactory.CreateAsync(agentDefinition, cancellationToken);
}
```

### 3.3 Workflow Building

**File**: [DeclarativeWorkflowBuilder.Build()](dotnet/src/Microsoft.Agents.AI.Workflows.Declarative/DeclarativeWorkflowBuilder.cs#L42-L70)

```csharp
public static Workflow Build<TInput>(
    string workflowFile,
    DeclarativeWorkflowOptions options,
    Func<TInput, ChatMessage>? inputTransform = null)
    where TInput : notnull
{
    using StreamReader yamlReader = File.OpenText(workflowFile);
    return Build(yamlReader, options, inputTransform);
}
```

---

## 4. Factory Pattern Implementation

### 4.1 Abstract Factory

**File**: [PromptAgentFactory.cs](dotnet/src/Microsoft.Agents.AI.Declarative/PromptAgentFactory.cs)

```csharp
public abstract class PromptAgentFactory
{
    protected PromptAgentFactory(RecalcEngine? engine = null, IConfiguration? configuration = null)
    {
        this.Engine = engine ?? new RecalcEngine();

        if (configuration is not null)
        {
            foreach (var kvp in configuration.AsEnumerable())
            {
                this.Engine.UpdateVariable(kvp.Key, kvp.Value ?? string.Empty);
            }
        }
    }

    protected RecalcEngine Engine { get; }

    public async Task<AIAgent> CreateAsync(GptComponentMetadata promptAgent, CancellationToken cancellationToken = default)
    {
        Throw.IfNull(promptAgent);

        var agent = await this.TryCreateAsync(promptAgent, cancellationToken).ConfigureAwait(false);
        return agent ?? throw new NotSupportedException($"Agent type {promptAgent.Kind} is not supported.");
    }

    public abstract Task<AIAgent?> TryCreateAsync(GptComponentMetadata promptAgent, CancellationToken cancellationToken = default);
}
```

### 4.2 Aggregator Factory

**File**: [AggregatorPromptAgentFactory.cs](dotnet/src/Microsoft.Agents.AI.Declarative/AggregatorPromptAgentFactory.cs)

```csharp
public sealed class AggregatorPromptAgentFactory : PromptAgentFactory
{
    private readonly PromptAgentFactory[] _agentFactories;

    public AggregatorPromptAgentFactory(params PromptAgentFactory[] agentFactories)
    {
        Throw.IfNullOrEmpty(agentFactories);
        this._agentFactories = agentFactories;
    }

    public override async Task<AIAgent?> TryCreateAsync(GptComponentMetadata promptAgent, CancellationToken cancellationToken = default)
    {
        Throw.IfNull(promptAgent);

        foreach (var agentFactory in this._agentFactories)
        {
            var agent = await agentFactory.TryCreateAsync(promptAgent, cancellationToken).ConfigureAwait(false);
            if (agent is not null)
            {
                return agent;
            }
        }

        return null;
    }
}
```

### 4.3 ChatClient Factory

**File**: [ChatClientPromptAgentFactory.cs](dotnet/src/Microsoft.Agents.AI.Declarative/ChatClient/ChatClientPromptAgentFactory.cs)

```csharp
public sealed class ChatClientPromptAgentFactory : PromptAgentFactory
{
    private readonly IChatClient _chatClient;
    private readonly IList<AIFunction>? _functions;
    private readonly ILoggerFactory? _loggerFactory;

    public ChatClientPromptAgentFactory(
        IChatClient chatClient, 
        IList<AIFunction>? functions = null, 
        RecalcEngine? engine = null, 
        IConfiguration? configuration = null, 
        ILoggerFactory? loggerFactory = null) : base(engine, configuration)
    {
        this._chatClient = chatClient;
        this._functions = functions;
        this._loggerFactory = loggerFactory;
    }

    public override Task<AIAgent?> TryCreateAsync(GptComponentMetadata promptAgent, CancellationToken cancellationToken = default)
    {
        var options = new ChatClientAgentOptions()
        {
            Name = promptAgent.Name,
            Description = promptAgent.Description,
            ChatOptions = promptAgent.GetChatOptions(this.Engine, this._functions),
        };

        var agent = new ChatClientAgent(this._chatClient, options, this._loggerFactory);

        return Task.FromResult<AIAgent?>(agent);
    }
}
```

### 4.4 Workflow Agent Provider

**File**: [WorkflowAgentProvider.cs](dotnet/src/Microsoft.Agents.AI.Workflows.Declarative/WorkflowAgentProvider.cs)

Abstract base class for workflow execution:

```csharp
public abstract class WorkflowAgentProvider
{
    public IEnumerable<AIFunction>? Functions { get; init; }
    public bool AllowConcurrentInvocation { get; init; }
    public bool AllowMultipleToolCalls { get; init; }

    public abstract Task<string> CreateConversationAsync(CancellationToken cancellationToken = default);
    public abstract Task<ChatMessage> CreateMessageAsync(string conversationId, ChatMessage conversationMessage, CancellationToken cancellationToken = default);
    public abstract Task<ChatMessage> GetMessageAsync(string conversationId, string messageId, CancellationToken cancellationToken = default);
    public abstract IAsyncEnumerable<AgentResponseUpdate> InvokeAgentAsync(
        string agentId,
        string? agentVersion,
        string? conversationId,
        IEnumerable<ChatMessage>? messages,
        IDictionary<string, object?>? inputArguments,
        CancellationToken cancellationToken = default);
    public abstract IAsyncEnumerable<ChatMessage> GetMessagesAsync(
        string conversationId,
        int? limit = null,
        string? after = null,
        string? before = null,
        bool newestFirst = false,
        CancellationToken cancellationToken = default);
}
```

### 4.5 Azure Agent Provider Implementation

**File**: [AzureAgentProvider.cs](dotnet/src/Microsoft.Agents.AI.Workflows.Declarative.AzureAI/AzureAgentProvider.cs)

```csharp
public sealed class AzureAgentProvider(Uri projectEndpoint, TokenCredential projectCredentials) : WorkflowAgentProvider
{
    private readonly Dictionary<string, AgentVersion> _versionCache = [];
    private readonly Dictionary<string, AIAgent> _agentCache = [];

    public AIProjectClientOptions? AIProjectClientOptions { get; init; }
    public ProjectOpenAIClientOptions? OpenAIClientOptions { get; init; }
    public HttpClient? HttpClient { get; init; }

    public override async Task<string> CreateConversationAsync(CancellationToken cancellationToken = default)
    {
        ProjectConversation conversation =
            await this.GetConversationClient()
                .CreateProjectConversationAsync(options: null, cancellationToken).ConfigureAwait(false);

        return conversation.Id;
    }

    // ... implementation details
}
```

---

## 5. Tool Resolution Flow

### 5.1 GetAITools Method

**File**: [PromptAgentExtensions.cs](dotnet/src/Microsoft.Agents.AI.Declarative/Extensions/PromptAgentExtensions.cs#L58-L75)

```csharp
internal static List<AITool>? GetAITools(this GptComponentMetadata promptAgent, IList<AIFunction>? functions)
{
    return promptAgent.Tools.Select(tool =>
    {
        return tool switch
        {
            CodeInterpreterTool => ((CodeInterpreterTool)tool).AsCodeInterpreterTool(),
            InvokeClientTaskAction => ((InvokeClientTaskAction)tool).CreateOrGetAITool(functions),
            McpServerTool => ((McpServerTool)tool).CreateHostedMcpTool(),
            FileSearchTool => ((FileSearchTool)tool).CreateFileSearchTool(),
            WebSearchTool => ((WebSearchTool)tool).CreateWebSearchTool(),
            _ => throw new NotSupportedException($"Unable to create tool definition because of unsupported tool type: {tool.Kind}"),
        };
    }).ToList() ?? [];
}
```

### 5.2 Supported Tool Kinds

| Kind | YAML Value | Extension File |
|------|------------|----------------|
| Code Interpreter | `codeInterpreter` | `CodeInterpreterToolExtensions.cs` |
| Function | `function` | `FunctionToolExtensions.cs` |
| MCP Server | `mcp` | `McpServerToolExtensions.cs` |
| File Search | `fileSearch` | `FileSearchToolExtensions.cs` |
| Web Search | `webSearch` | `WebSearchToolExtensions.cs` |

---

## 6. Sample YAML Files

### 6.1 Agent Samples

**Location**: [agent-samples/](agent-samples/)

| File | Provider | Description |
|------|----------|-------------|
| [azure/AzureOpenAI.yaml](agent-samples/azure/AzureOpenAI.yaml) | Azure OpenAI | Basic chat agent with output schema |
| [azure/AzureOpenAIAssistants.yaml](agent-samples/azure/AzureOpenAIAssistants.yaml) | Azure OpenAI | Assistants API |
| [chatclient/GetWeather.yaml](agent-samples/chatclient/GetWeather.yaml) | ChatClient | Agent with function tools |
| [foundry/FoundryAgent.yaml](agent-samples/foundry/FoundryAgent.yaml) | AI Foundry | Remote connection |
| [foundry/MicrosoftLearnAgent.yaml](agent-samples/foundry/MicrosoftLearnAgent.yaml) | AI Foundry | MCP server tool example |

### 6.2 Workflow Samples

**Location**: [workflow-samples/](workflow-samples/)

| File | Description |
|------|-------------|
| [CustomerSupport.yaml](workflow-samples/CustomerSupport.yaml) | Multi-agent customer support workflow |
| [DeepResearch.yaml](workflow-samples/DeepResearch.yaml) | Complex orchestration with multiple agents |
| [MathChat.yaml](workflow-samples/MathChat.yaml) | Simple math conversation |
| [Marketing.yaml](workflow-samples/Marketing.yaml) | Marketing workflow example |

---

## 7. Expression Syntax

The declarative agents support Power Fx expressions for dynamic values:

| Syntax | Description | Example |
|--------|-------------|---------|
| `=Env.VAR_NAME` | Environment variable | `=Env.AZURE_OPENAI_DEPLOYMENT_NAME` |
| `=System.ConversationId` | System variable | `conversationId: =System.ConversationId` |
| `=Local.VariableName` | Local workflow variable | `=Local.ServiceParameters.IsResolved` |
| `=UserMessage(text)` | Create user message | `=UserMessage(Local.InputTask)` |
| `=Not(condition)` | Logical NOT | `=Not(Local.ServiceParameters.IsResolved)` |
| `=Concat(...)` | String concatenation | `=Concat(ForAll(...), Value, "\n")` |

---

## 8. Key Dependencies

From [Microsoft.Agents.AI.Declarative.csproj](dotnet/src/Microsoft.Agents.AI.Declarative/Microsoft.Agents.AI.Declarative.csproj):

```xml
<PackageReference Include="Microsoft.Agents.ObjectModel" />
<PackageReference Include="Microsoft.Agents.ObjectModel.Json" />
<PackageReference Include="Microsoft.Agents.ObjectModel.PowerFx" />
<PackageReference Include="Microsoft.PowerFx.Interpreter" />
<PackageReference Include="Microsoft.Extensions.Configuration" />
<PackageReference Include="Microsoft.Extensions.Logging" />
```

**Note**: The `Microsoft.Agents.ObjectModel` package provides the core schema types (`GptComponentMetadata`, `BotElement`, `YamlSerializer`, etc.) which are external dependencies.

---

## 9. Error Handling Patterns

### 9.1 Validation Errors

```csharp
// Invalid root element
throw new InvalidDataException($"Unsupported root element: {rootElement.GetType().Name}. Expected an {nameof(GptComponentMetadata)}.");

// Unsupported agent type
throw new NotSupportedException($"Agent type {promptAgent.Kind} is not supported.");

// Unsupported tool type
throw new NotSupportedException($"Unable to create tool definition because of unsupported tool type: {tool.Kind}");
```

### 9.2 Workflow Exceptions

**File**: [DeclarativeActionExecutor.cs](dotnet/src/Microsoft.Agents.AI.Workflows.Declarative/Interpreter/DeclarativeActionExecutor.cs#L31-L36)

```csharp
if (!model.HasRequiredProperties)
{
    throw new DeclarativeModelException($"Missing required properties for element: {model.GetId()} ({model.GetType().Name}).");
}
```

---

## 10. Summary

### Key Findings

1. **Two-Layer Architecture**: 
   - `Microsoft.Agents.AI.Declarative` handles prompt-based agents
   - `Microsoft.Agents.AI.Workflows.Declarative` handles multi-step workflows

2. **Factory Pattern**: Uses abstract factory pattern with aggregator support for extensibility

3. **External Object Model**: Core schema types (`GptComponentMetadata`, `BotElement`) are in `Microsoft.Agents.ObjectModel` package

4. **Power Fx Integration**: Expressions use Power Fx engine for dynamic evaluation

5. **Tool Types**: Supports 5 tool types (function, codeInterpreter, mcp, webSearch, fileSearch)

6. **Provider Pattern**: `WorkflowAgentProvider` abstraction enables different backends (Azure AI Foundry, etc.)

### File References Summary

| Component | Primary File |
|-----------|--------------|
| YAML Parsing | [AgentBotElementYaml.cs](dotnet/src/Microsoft.Agents.AI.Declarative/AgentBotElementYaml.cs) |
| Factory Base | [PromptAgentFactory.cs](dotnet/src/Microsoft.Agents.AI.Declarative/PromptAgentFactory.cs) |
| Aggregator | [AggregatorPromptAgentFactory.cs](dotnet/src/Microsoft.Agents.AI.Declarative/AggregatorPromptAgentFactory.cs) |
| ChatClient Factory | [ChatClientPromptAgentFactory.cs](dotnet/src/Microsoft.Agents.AI.Declarative/ChatClient/ChatClientPromptAgentFactory.cs) |
| Tool Resolution | [PromptAgentExtensions.cs](dotnet/src/Microsoft.Agents.AI.Declarative/Extensions/PromptAgentExtensions.cs) |
| Workflow Builder | [DeclarativeWorkflowBuilder.cs](dotnet/src/Microsoft.Agents.AI.Workflows.Declarative/DeclarativeWorkflowBuilder.cs) |
| Workflow Provider | [WorkflowAgentProvider.cs](dotnet/src/Microsoft.Agents.AI.Workflows.Declarative/WorkflowAgentProvider.cs) |
| Azure Provider | [AzureAgentProvider.cs](dotnet/src/Microsoft.Agents.AI.Workflows.Declarative.AzureAI/AzureAgentProvider.cs) |
