# .NET Protocols, Hosting & Workflows Research

> **Date**: 2026-02-20
> **Scope**: 13 projects under `dotnet/src/` covering protocol clients, hosting endpoints, workflow execution, durable agents, declarative workflows, and source generators.

---

## Table of Contents

1. [Microsoft.Agents.AI.A2A](#1-microsoftagentsaia2a)
2. [Microsoft.Agents.AI.AGUI](#2-microsoftagentsaiagui)
3. [Microsoft.Agents.AI.Hosting.A2A](#3-microsoftagentsaihostinga2a)
4. [Microsoft.Agents.AI.Hosting.A2A.AspNetCore](#4-microsoftagentsaihostinga2aaspnetcore)
5. [Microsoft.Agents.AI.Hosting.AGUI.AspNetCore](#5-microsoftagentsaihostingaguiaspnetcore)
6. [Microsoft.Agents.AI.Hosting.OpenAI](#6-microsoftagentsaihostingopenai)
7. [Microsoft.Agents.AI.Hosting.AzureFunctions](#7-microsoftagentsaihostingazurefunctions)
8. [Microsoft.Agents.AI.Workflows](#8-microsoftagentsaiworkflows)
9. [Microsoft.Agents.AI.Workflows.Declarative](#9-microsoftagentsaiworkflowsdeclarative)
10. [Microsoft.Agents.AI.Workflows.Declarative.AzureAI](#10-microsoftagentsaiworkflowsdeclarativeazureai)
11. [Microsoft.Agents.AI.Workflows.Generators](#11-microsoftagentsaiworkflowsgenerators)
12. [Microsoft.Agents.AI.DurableTask](#12-microsoftagentsaidurabletask)
13. [Microsoft.Agents.AI.Declarative](#13-microsoftagentsaideclarative)
14. [Cross-Cutting Patterns](#14-cross-cutting-patterns)

---

## 1. Microsoft.Agents.AI.A2A

**Purpose**: Agent-to-Agent (A2A) protocol client — wraps remote A2A-compliant servers as local `AIAgent` instances.

### NuGet Dependencies

| Package | Notes |
|---------|-------|
| `A2A` | A2A protocol NuGet (provides `A2AClient`, `AgentCard`, `TaskManager`, etc.) |
| Project ref: `Microsoft.Agents.AI.Abstractions` | Core agent abstractions (`AIAgent`, `AgentSession`, `AgentResponse`, etc.) |

### Classes, Interfaces, Enums

| Type | Kind | Description |
|------|------|-------------|
| `A2AAgent` | `class : AIAgent` | Main agent. `RunCoreAsync` calls `A2AClient.SendMessageAsync()` (non-streaming). `RunCoreStreamingAsync` calls `A2AClient.SendMessageStreamingAsync()` (SSE streaming). Supports `AgentMessage` and `AgentTask` A2A response types. |
| `A2AAgentSession` | `class : AgentSession` | Holds `ContextId` (conversation context) and `TaskId` (long-running task tracking). |
| `A2AContinuationToken` | `class : ResponseContinuationToken` | Encodes `taskId` for polling long-running A2A tasks. |
| `A2AJsonUtilities` | `static partial class` | Source-generated `JsonSerializerContext` with `JsonSerializerDefaults.Web`, `WhenWritingNull`, `AllowReadingFromString`. |

### Extension Methods

| Class | Key Methods |
|-------|-------------|
| `A2AAgentCardExtensions` | `AsAIAgent()` — wraps an `AgentCard` + `HttpClient` as `A2AAgent` |
| `A2AClientExtensions` | `AsAIAgent()` — wraps an existing `A2AClient` as `A2AAgent` |
| `A2ACardResolverExtensions` | `GetAIAgentAsync()` — resolves agent card by URL and wraps as `A2AAgent` |
| `ChatMessageExtensions` | `ToA2AMessage()` — converts `ChatMessage` to A2A `Message` |
| `A2AMessagePartConverterExtensions` | Various metadata/content converters between MEAI and A2A types |

### Streaming Protocol

- Uses A2A SSE streaming via `A2AClient.SendMessageStreamingAsync()`.
- Emits `AgentResponseUpdate` per SSE event containing text parts, metadata.
- Wraps A2A `TaskStatusUpdateEvent` and `TaskArtifactUpdateEvent` into the framework's streaming model.

### Agent Registration/Resolution

- No DI registration in this package (client-side only).
- Agents instantiated via factory extensions (`AsAIAgent()`).

---

## 2. Microsoft.Agents.AI.AGUI

**Purpose**: AG-UI (Agent-User Interface) protocol client — implements `IChatClient` for communicating with AG-UI servers.

### NuGet Dependencies

| Package | Notes |
|---------|-------|
| `Microsoft.Extensions.AI` | MEAI chat client abstractions |
| `System.Net.ServerSentEvents` | SSE parsing (`SseParser.Create()`) |
| `System.Net.Http.Json` | HTTP JSON extensions |
| `System.Threading.Channels` | For async streaming pipelines |

### Classes, Interfaces, Enums

| Type | Kind | Description |
|------|------|-------------|
| `AGUIChatClient` | `class : DelegatingChatClient` | Main client. Posts `RunAgentInput` JSON to AG-UI server. Receives SSE stream. Wraps `FunctionInvokingChatClient` for auto tool calling. Manages thread IDs across turns (AG-UI requires full history each turn). |
| `AGUIHttpService` | `class` | HTTP transport. Posts `RunAgentInput`, receives SSE stream via `SseParser.Create()`. |
| `AGUIJsonSerializerContext` | Source-generated JSON context | Covers all AG-UI request/response types. |

### Shared Event Types (conditionally compiled with `ASPNETCORE` define)

These types live in `Shared/` and are included by both this project (client) and `Hosting.AGUI.AspNetCore` (server):

| Type | Kind | Description |
|------|------|-------------|
| `BaseEvent` | `abstract class` | Base for all AG-UI events. Has `Type` discriminator string. |
| `BaseEventJsonConverter` | `JsonConverter` | Polymorphic JSON deserialization based on `Type` field. |
| **Run events** | | `RunStartedEvent`, `RunFinishedEvent`, `RunErrorEvent` |
| **Text message events** | | `TextMessageStartEvent`, `TextMessageContentEvent`, `TextMessageEndEvent` |
| **Tool call events** | | `ToolCallStartEvent`, `ToolCallArgsEvent`, `ToolCallEndEvent`, `ToolCallResultEvent` |
| **State events** | | `StateSnapshotEvent`, `StateDeltaEvent` |
| `RunAgentInput` | `class` | Request body: `ThreadId`, `RunId`, `State`, `Messages[]`, `Tools[]`, `Context`, `ForwardedProperties` |
| **Message types** | | `AGUIUserMessage`, `AGUIAssistantMessage`, `AGUISystemMessage`, `AGUIDeveloperMessage`, `AGUIToolMessage` |

### Streaming Protocol

- **Client→Server**: `POST` with `RunAgentInput` JSON body.
- **Server→Client**: SSE stream of `BaseEvent` subclasses.
- Bidirectional conversion: `ChatResponseUpdateAGUIExtensions` converts between `ChatResponseUpdate` streams and AG-UI event streams.

---

## 3. Microsoft.Agents.AI.Hosting.A2A

**Purpose**: Host local `AIAgent` instances as A2A protocol servers.

### NuGet Dependencies

| Package | Notes |
|---------|-------|
| `A2A` | A2A protocol NuGet |
| Project ref: `Microsoft.Agents.AI.Abstractions` | |
| Project ref: `Microsoft.Agents.AI.Hosting` | Provides `AIHostAgent`, `AgentSessionStore` |

### Classes

| Type | Kind | Description |
|------|------|-------------|
| `AIAgentExtensions` | `static class` | `.MapA2A()` extension on `AIAgent`. Creates `AIHostAgent` wrapper, configures `TaskManager.OnMessageReceived` event handler. Converts `ChatMessage` ↔ A2A `Message`. |
| `MessageConverter` | `static class` | Converts `MessageSendParams` → `List<ChatMessage>` and `IList<ChatMessage>` → `List<Part>`. |

### Agent Registration

- Uses `AgentSessionStore` for session persistence.
- `AIHostAgent` wraps the target `AIAgent` with session management.

---

## 4. Microsoft.Agents.AI.Hosting.A2A.AspNetCore

**Purpose**: ASP.NET Core endpoint routing for A2A agent hosting.

### NuGet Dependencies

| Package | Notes |
|---------|-------|
| `A2A.AspNetCore` | Provides `A2ARouteBuilderExtensions.MapA2A()` |
| Project ref: `Microsoft.Agents.AI.Hosting.A2A` | |

### Classes

| Type | Kind | Description |
|------|------|-------------|
| `MicrosoftAgentAIHostingA2AEndpointRouteBuilderExtensions` | `static class` | Multiple `MapA2A()` overloads on `IEndpointRouteBuilder`. |

### HTTP Endpoint Patterns

- Delegates to `A2ARouteBuilderExtensions.MapA2A()` from the A2A NuGet + `MapHttpA2A()`.
- Standard A2A endpoints: JSON-RPC over HTTP, SSE for streaming.

### Agent Resolution

```csharp
endpoints.ServiceProvider.GetRequiredKeyedService<AIAgent>(agentName)
```

Agents are registered as **keyed services** in the DI container.

---

## 5. Microsoft.Agents.AI.Hosting.AGUI.AspNetCore

**Purpose**: Host `AIAgent` instances as AG-UI protocol servers via ASP.NET Core.

### NuGet Dependencies

| Package | Notes |
|---------|-------|
| ASP.NET Core framework reference | |
| Project ref: `Microsoft.Agents.AI.AGUI` | Shared event types via `<Compile Include>` with `ASPNETCORE` define |
| Project ref: `Microsoft.Agents.AI.Hosting` | |
| `System.Net.ServerSentEvents` | SSE formatting |

### Classes

| Type | Kind | Description |
|------|------|-------------|
| `AGUIEndpointRouteBuilderExtensions` | `static class` | `MapAGUI()` — maps `POST` endpoint. Deserializes `RunAgentInput`, runs agent streaming, converts to AG-UI events, returns SSE. |
| `AGUIServerSentEventsResult` | `class : IResult` | Streams AG-UI events as SSE. Sets `Content-Type: text/event-stream`, `Cache-Control: no-cache,no-store`. Uses `SseFormatter.WriteAsync()`. Sends `RunErrorEvent` on failure. |
| `AGUIChatResponseUpdateStreamExtensions` | `static class` | Filters server-side tools from mixed tool invocations. |
| `ServiceCollectionExtensions` | `static class` | `AddAGUI()` — configures JSON serializer options. |

### HTTP Endpoint Pattern

```
POST {pattern}
  Request:  RunAgentInput (JSON)
  Response: text/event-stream (SSE of BaseEvent subclasses)
  Headers:  Content-Type: text/event-stream; Cache-Control: no-cache,no-store
```

### Agent Resolution

- Keyed service: `GetRequiredKeyedService<AIAgent>(agentName)` or direct instance.

---

## 6. Microsoft.Agents.AI.Hosting.OpenAI

**Purpose**: OpenAI-compatible REST API hosting for agents. Provides three API surfaces.

### NuGet Dependencies

| Package | Notes |
|---------|-------|
| `Microsoft.Extensions.AI.OpenAI` | OpenAI-compatible types |
| `Microsoft.Extensions.AI.Abstractions` | |
| ASP.NET Core framework reference | |
| `System.Net.ServerSentEvents` | SSE formatting |

### API Surface 1: Chat Completions

**Endpoint**: `POST /{agentName}/v1/chat/completions/`

| Type | Kind | Description |
|------|------|-------------|
| `AIAgentChatCompletionsProcessor` | `class` | `CreateChatCompletionAsync()` — processes chat completion requests. Handles both streaming and non-streaming. |
| `CreateChatCompletion` | Model | Request body |
| `ChatCompletion` | Model | Non-streaming response |
| `ChatCompletionChunk` | Model | Streaming response chunk |
| `ChatCompletionMessage` | Model | Message within completion |
| Various role/content models | | Delta, choice, usage models |

### API Surface 2: Responses API

**Endpoints**:
- `POST /{agentName}/v1/responses/` — create response
- `GET /{agentName}/v1/responses/{responseId}` — get response
- `POST /{agentName}/v1/responses/{responseId}/cancel` — cancel
- `DELETE /{agentName}/v1/responses/{responseId}` — delete
- `GET /{agentName}/v1/responses/{responseId}/input_items` — get input items

| Type | Kind | Description |
|------|------|-------------|
| `InMemoryResponsesService` | `class` | In-memory response storage and execution |
| `AIAgentResponseExecutor` | `class` | Executes agent runs for the Responses API |

### API Surface 3: Conversations API

**Endpoints**: CRUD under `/v1/conversations`:
- List, create, get, update, delete conversations
- List, create, get, update, delete conversation items

| Type | Kind | Description |
|------|------|-------------|
| `IConversationStorage` | `interface` | Storage backend for conversations |
| `IAgentConversationIndex` | `interface` | Index for agent-conversation mapping |

### Streaming Protocol (SSE)

| Type | Kind | Description |
|------|------|-------------|
| `SseJsonResult<T>` | `class` | Generic SSE streamer. Sets `Content-Type: text/event-stream`, `Cache-Control: no-cache,no-store`, `Connection: keep-alive`. |

### Streaming Event Generators (Strategy Pattern)

| Generator | Handles |
|-----------|---------|
| `StreamingEventGenerator` | Abstract base |
| `AssistantMessageEventGenerator` | Text content |
| `AudioContentEventGenerator` | Audio |
| `ErrorContentEventGenerator` | Errors |
| `FileContentEventGenerator` | File references |
| `FunctionCallEventGenerator` | Tool calls |
| `FunctionResultEventGenerator` | Tool results |
| `FunctionApprovalRequestEventGenerator` | User approval requests |
| `FunctionApprovalResponseEventGenerator` | User approval responses |
| `HostedFileContentEventGenerator` | Hosted files |
| `ImageContentEventGenerator` | Images |
| `TextReasoningContentEventGenerator` | Chain-of-thought reasoning |

### Service Registration

```csharp
services.AddOpenAIChatCompletions();
services.AddOpenAIResponses();
services.AddOpenAIConversations();
```

### Agent Resolution

- `GetRequiredKeyedService<AIAgent>(agentName)` or direct `AIAgent` instance.

---

## 7. Microsoft.Agents.AI.Hosting.AzureFunctions

**Purpose**: Azure Functions hosting for durable agents with HTTP triggers, entity triggers, and MCP tool triggers.

### NuGet Dependencies

| Package | Notes |
|---------|-------|
| `Microsoft.Azure.Functions.Worker` | Azure Functions isolated worker |
| `Microsoft.Azure.Functions.Worker.Extensions.DurableTask` | Durable entities |
| `Microsoft.Azure.Functions.Worker.Extensions.Http` | HTTP triggers |
| `Microsoft.Azure.Functions.Worker.Extensions.Mcp` | MCP tool triggers |
| Project ref: `Microsoft.Agents.AI.DurableTask` | |

### Classes

| Type | Kind | Description |
|------|------|-------------|
| `BuiltInFunctions` | `static class` | Contains `RunAgentHttpAsync()` (HTTP trigger), `InvokeAgentAsync()` (entity trigger), `RunMcpToolAsync()` (MCP tool trigger). |
| `BuiltInFunctionExecutor` | `class : IFunctionExecutor` | Custom function executor for built-in static entry points. Routes to `BuiltInFunctions` methods based on entry point name. |
| `BuiltInFunctionExecutionMiddleware` | `class : IFunctionsWorkerMiddleware` | Azure Functions middleware that sets the custom `IFunctionExecutor` for built-in function invocations. |
| `DurableAgentFunctionMetadataTransformer` | `class : IFunctionMetadataTransformer` | Dynamically registers entity, HTTP, and MCP tool trigger functions for each configured agent at startup. |
| `DefaultFunctionsAgentOptionsProvider` | `class : IFunctionsAgentOptionsProvider` | Provides per-agent trigger configuration. Default: HTTP enabled, MCP disabled. |
| `FunctionsAgentOptions` | `class` | Options container with `HttpTriggerOptions` and `McpToolTriggerOptions`. |
| `FunctionsApplicationBuilderExtensions` | `static class` | `ConfigureDurableAgents()` on `FunctionsApplicationBuilder`. DI setup. |
| `DurableAgentsOptionsExtensions` | `static class` | `AddAIAgent()` extensions for registering agents with Functions-specific trigger configuration. |
| `DurableTaskClientExtensions` | `static class` | `AsDurableAgentProxy()` — converts `DurableTaskClient` into a durable agent proxy. |

### HTTP Endpoint Pattern

```
POST agents/{agentName}/run
  Request:  { "message": "...", "thread_id": "..." } (JSON) or plain text body
  Headers:  x-ms-wait-for-response (optional; if absent → 202 Accepted fire-and-forget)
  Query:    ?thread_id=... (alternative to JSON body)
  Response: AgentResponse JSON or 202 Accepted
```

### Dynamic Function Registration

The `DurableAgentFunctionMetadataTransformer` adds functions at startup:

1. **Entity trigger**: `{entityName}` — Durable Task entity for agent state
2. **HTTP trigger**: `http-{agentName}` at `agents/{agentName}/run` — REST API endpoint
3. **MCP tool trigger**: `mcptool-{agentName}` — MCP protocol tool endpoint

---

## 8. Microsoft.Agents.AI.Workflows

**Purpose**: Graph-based workflow execution engine for multi-agent orchestration. The largest project (100+ source files).

### NuGet Dependencies

| Package | Notes |
|---------|-------|
| `OpenTelemetry.Api` | Observability |
| Project ref: `Microsoft.Agents.AI.Abstractions` | |
| Project ref: `Microsoft.Agents.AI` | Core agent types |

### Target Frameworks

`net472`, `netstandard2.0`, `net8.0`, `net9.0`, `net10.0`

### Core Architecture

#### Workflow Definition

| Type | Kind | Description |
|------|------|-------------|
| `Workflow` | `class` | Core workflow definition. Contains a graph of executors connected by edges. Has `DescribeProtocolAsync()` for type checking. |
| `WorkflowBuilder` | `class` | Fluent API for constructing workflows. `AddEdge()`, `AddFanOutEdge()`, `AddFanInEdge()`, `WithOutputFrom()`, `Build()`. |
| `WorkflowBuilderExtensions` | `static class` | Additional builder convenience methods. |
| `SwitchBuilder` | `class` | Predicate-based routing. `AddCase()` + `WithDefault()`. Reduces to fan-out edges internally. |

#### Executor (Node) System

| Type | Kind | Description |
|------|------|-------------|
| `Executor` | `abstract class` | Base execution unit. Has `Id`, `ConfigureRoutes()`, `ExecuteAsync()`, message routing via `MessageRouter`, `InputTypes`/`OutputTypes` sets. Supports checkpointing (`OnCheckpointingAsync`/`OnCheckpointRestoredAsync`). |
| `Executor<TInput>` | `abstract class` | Single-input executor with `HandleAsync(TInput, IWorkflowContext, CT)`. |
| `Executor<TInput, TOutput>` | `abstract class` | Input/output executor with `HandleAsync` returning `ValueTask<TOutput>`. |
| `StatefulExecutor` | `abstract class` | Executor with per-run state management. |
| `FunctionExecutor` | `class` | Wraps a delegate as an executor. |
| `AggregatingExecutor` | `abstract class` | Executor that aggregates messages across invocations using state. |
| `ChatProtocolExecutor` | `class` | Executor that handles chat messages via the Chat Protocol. |
| `ChatForwardingExecutor` | `class` | Forwards incoming chat messages to downstream executors. |

#### Specialized Executors

| Type | Kind | Description |
|------|------|-------------|
| `AIAgentHostExecutor` | `class` | Hosts an `AIAgent` as an executor within a workflow. Supports streaming, event emission, message forwarding. |
| `HandoffAgentExecutor` | `class` | Handles agent handoff logic. |
| `HandoffsStartExecutor` / `HandoffsEndExecutor` | `class` | Start/end points for handoff workflows. |
| `GroupChatHost` | `class` | Hosts group chat sessions within workflows. |
| `ConcurrentEndExecutor` | `class` | Aggregates results from concurrent fan-out execution paths. |
| `AggregateTurnMessagesExecutor` | `class` | Batches messages from a single turn for aggregation. |
| `OutputMessagesExecutor` | `class` | Terminal executor that yields messages as workflow output. |
| `RequestInfoExecutor` | `class` | Handles external request port Info events. |
| `AIContentExternalHandler` | `class` | Handles `AIContent`-based external requests. |

#### Edge System (Graph Connectivity)

| Type | Kind | Description |
|------|------|-------------|
| `Edge` | `class` | Connection between executors. |
| `EdgeData` | `abstract class` | Base for edge configuration. |
| `DirectEdgeData` | `class : EdgeData` | Direct 1:1 edge with optional condition function. |
| `FanOutEdgeData` | `class : EdgeData` | 1:N edge with partitioner function. |
| `FanInEdgeData` | `class : EdgeData` | N:1 aggregation edge. |
| `EdgeId` | `record` | Unique edge identifier. |

#### Workflow Hosting

| Type | Kind | Description |
|------|------|-------------|
| `WorkflowHostAgent` | `internal sealed class : AIAgent` | Wraps a `Workflow` as an `AIAgent`. Validates ChatProtocol compatibility. Creates `WorkflowSession` per call. |
| `WorkflowSession` | `internal sealed class : AgentSession` | Manages workflow run state: checkpoint manager, chat history provider, run ID. `InvokeStageAsync()` drives the workflow and yields `AgentResponseUpdate`s. |
| `WorkflowHostingExtensions` | `static class` | `Workflow.AsAgent()` — entry point for converting a workflow to an `AIAgent`. |

#### Agent Workflow Builders (Convenience Patterns)

| Type | Kind | Description |
|------|------|-------------|
| `AgentWorkflowBuilder` | `static class` | `BuildSequential()` — pipeline of agents. `BuildConcurrent()` — parallel agents with aggregation. `CreateHandoffBuilderWith()` — handoff-based workflows. `CreateGroupChatBuilderWith()` — group chat workflows. |
| `HandoffsWorkflowBuilder` | `class` | Fluent builder for handoff workflows. Agents hand off to each other via `AITool` invocations. |
| `GroupChatWorkflowBuilder` | `class` | Fluent builder for group chat workflows with a `GroupChatManager`. |
| `GroupChatManager` | `abstract class` | Manages group chat turn selection. |
| `RoundRobinGroupChatManager` | `class` | Round-robin turn selection for group chats. |

#### Binding System

| Type | Kind | Description |
|------|------|-------------|
| `ExecutorBinding` | `record` | Binds an executor factory + ID + type + optional raw reference. |
| `ExecutorInstanceBinding` | `class` | Binds a specific executor instance. |
| `AIAgentBinding` | `record : ExecutorBinding` | Binds an `AIAgent` as an executor via `AIAgentHostExecutor`. |
| `SubworkflowBinding` | `class` | Binds a sub-workflow as an executor. |
| `ConfiguredExecutorBinding` | `class` | Binds with configuration. |
| `PortBinding` / `RequestPortBinding` | `class` | Binds external request ports. |

#### External Request/Response System

| Type | Kind | Description |
|------|------|-------------|
| `RequestPort` / `RequestPort<TReq, TRes>` | `record` | External request port definition with typed request/response. |
| `ExternalRequest` | `record` | Request to an external port. Contains `PortInfo`, `RequestId`, `Data` (as `PortableValue`). Converts to `FunctionCallContent` for MEAI compatibility. |
| `ExternalResponse` | `record` | Response from external port. |
| `PortableValue` | `sealed class` | Type-erased value with delayed deserialization support. Used for cross-boundary data. JSON-serializable. |
| `IExternalRequestContext` | `interface` | Context for handling external requests within executor configuration. |

#### Execution Engine

| Type | Kind | Description |
|------|------|-------------|
| `InProcessExecution` | `static class` | Provides three execution modes: `OffThread` (default), `Concurrent`, `Lockstep`. |
| `IWorkflowExecutionEnvironment` | `interface` | Abstraction for workflow execution backends. `StreamAsync()`, `ResumeStreamAsync()`. |
| `InProcessExecutionEnvironment` | `class` | In-process implementation. |
| `InProcessRunner` | `class` | Core runner that drives super-step execution. |
| `InProcessRunnerContext` | `class` | Context for in-process execution. |
| `Run` / `StreamingRun` | `class` | Workflow run instances. `WatchStreamAsync()` yields `WorkflowEvent`s. `TrySendMessageAsync()` for feeding messages/tokens. |
| `MessageRouter` | `class` | Routes messages to executor handlers by type. |
| `RouteBuilder` | `class` | Fluent builder for message routes: `AddHandler<T>()`. |
| `EdgeRunner` / `DirectEdgeRunner` / `FanOutEdgeRunner` / `FanInEdgeRunner` | `class` | Execute edge delivery logic. |
| `StateManager` / `StateScope` | `class` | Per-executor state management. |
| `StepContext` | `class` | Context for a single super-step execution. |

#### Workflow Events

| Type | Kind | Description |
|------|------|-------------|
| `WorkflowEvent` | `abstract class` | Base for all workflow events. |
| `AgentResponseUpdateEvent` | `class` | Wraps `AgentResponseUpdate` as workflow event. |
| `AgentResponseEvent` | `class` | Wraps `AgentResponse` as workflow event. |
| `WorkflowStartedEvent` | `class` | Workflow execution started. |
| `WorkflowErrorEvent` | `class` | Workflow error with exception. |
| `WorkflowWarningEvent` | `class` | Warning event. |
| `WorkflowOutputEvent` | `class` | Workflow output data. |
| `SuperStepStartedEvent` / `SuperStepCompletedEvent` | `class` | Super-step lifecycle events. |
| `ExecutorInvokedEvent` / `ExecutorCompletedEvent` / `ExecutorFailedEvent` | `class` | Executor lifecycle events. |
| `RequestInfoEvent` / `RequestHaltEvent` | `class` | External request port events. |
| `SubworkflowErrorEvent` / `SubworkflowWarningEvent` | `class` | Sub-workflow error events. |

#### Checkpointing

| Type | Kind | Description |
|------|------|-------------|
| `CheckpointManager` | `class` | Manages checkpoint lifecycle. |
| `ICheckpointManager` / `ICheckpointStore` | `interface` | Checkpoint persistence abstractions. |
| `InMemoryCheckpointManager` | `class` | In-memory checkpoint storage. |
| `FileSystemJsonCheckpointStore` | `class` | File-system JSON checkpoint persistence. |
| `JsonCheckpointStore` | `class` | JSON-based checkpoint store. |
| `JsonMarshaller` | `class` | JSON serialization for checkpoint data. |
| `Checkpoint` / `CheckpointInfo` | `class` / `record` | Checkpoint data models. |
| Various converters | | `CheckpointInfoConverter`, `EdgeIdConverter`, `ExecutorIdentityConverter`, `PortableValueConverter`, `ScopeKeyConverter` |

#### Observability (OpenTelemetry)

| Type | Kind | Description |
|------|------|-------------|
| `ActivityNames` | `static class` | Activity names for tracing. |
| `ActivityExtensions` | `static class` | OpenTelemetry activity helpers. |
| `Tags` | `static class` | Semantic tag names. |
| `EventNames` | `static class` | Event name constants. |

#### Reflection / Code Generation Support

| Type | Kind | Description |
|------|------|-------------|
| `IMessageHandler<T>` / `IMessageHandler<T, TResult>` | `interface` | Handler interfaces. |
| `ReflectingExecutor` | `class` | Executor that discovers handlers via reflection. |
| `MessageHandlerAttribute` | `attribute` | Marks methods as message handlers (consumed by source generator). |
| `SendsMessageAttribute` | `attribute` | Declares message types sent by executor. |
| `YieldsOutputAttribute` | `attribute` | Declares output types yielded by executor. |
| `StreamsMessageAttribute` | `attribute` | Declares streaming message types. |

#### Protocol Descriptor

| Type | Kind | Description |
|------|------|-------------|
| `ProtocolDescriptor` | `class` | Describes executor/workflow input/output type contracts. |
| `ChatProtocolExtensions` | `static class` | `IsChatProtocol()` / `ThrowIfNotChatProtocol()` — validates that a protocol supports `List<ChatMessage>` + `TurnToken` input. |
| `TurnToken` | `class` | Token controlling turn-based execution flow. |

---

## 9. Microsoft.Agents.AI.Workflows.Declarative

**Purpose**: YAML-based declarative workflow definitions using the Foundry object model and PowerFx expressions.

### NuGet Dependencies

| Package | Notes |
|---------|-------|
| `Microsoft.Agents.ObjectModel` | Foundry workflow object model |
| `Microsoft.Agents.ObjectModel.Json` | JSON serialization |
| `Microsoft.Agents.ObjectModel.PowerFx` | PowerFx integration |
| `Microsoft.PowerFx.Interpreter` | PowerFx expression evaluation |
| `Microsoft.Extensions.Configuration` | Configuration |
| `System.CodeDom` | Code generation |
| Project ref: `Microsoft.Agents.AI.Workflows` | |

### Key Classes

| Type | Kind | Description |
|------|------|-------------|
| `DeclarativeWorkflowBuilder` | `static class` | Entry point. `Build<TInput>()` reads YAML, creates `WorkflowFormulaState`, visits workflow elements, produces `Workflow`. |
| `DeclarativeWorkflowOptions` | `class` | Configuration: `AgentProvider`, `Configuration`, `ConversationId`, `MaximumCallDepth`, `MaximumExpressionLength`, `LoggerFactory`. |
| `DeclarativeWorkflowLanguage` | `enum` | `Python`, `CSharp`, `JavaScript` — target languages for workflow ejection. |
| `WorkflowAgentProvider` | `abstract class` | Base for providing agents to declarative workflows. Has `CreateConversationAsync()`, `CreateMessageAsync()`, `GetMessageAsync()`, etc. Also configures `Functions`, `AllowConcurrentInvocation`, `AllowMultipleToolCalls`. |

### PowerFx Integration

| Type | Kind | Description |
|------|------|-------------|
| `WorkflowFormulaState` | `sealed class` | Variable scope management. Scopes: `Local`, `Global`, `System`, `Environment`, `Topic`. Uses `RecalcEngine` for expression evaluation. Supports checkpoint/restore via `IWorkflowContext`. |
| `WorkflowExpressionEngine` | `sealed class` | Evaluates `BoolExpression`, `StringExpression`, `ValueExpression`, `IntExpression`, etc. |
| `RecalcEngineFactory` | `static class` | Creates configured `RecalcEngine` with `AgentMessage()`, `UserMessage()`, `MessageText()` functions, `Set()` function, `PowerFxV1` features. |
| `SystemScope` | `static class` | Initializes system variables: `Activity`, `Bot`, `Conversation`, `ConversationId`, `LastMessage`, `LastMessageText`, `Recognizer`, `User`, `UserLanguage`. |
| `TypeSchema` | `static class` | Defines message record types for PowerFx: `Discriminator`, `Message.Fields`, `MessageContent.Fields`. |
| `WorkflowDiagnostics` | `static class` | Semantic analysis. `Describe()` extracts environment variables and user variables. `Initialize()` sets up scopes from workflow element. |

### PowerFx Functions

| Type | Description |
|------|-------------|
| `AgentMessage` | Creates assistant-role message records from strings |
| `UserMessage` | Creates user-role message records from strings |
| `MessageText` | Extracts text from message records (3 overloads: string, record, table) |

### Declarative Action Executors

| Type | Description |
|------|-------------|
| `SetVariableExecutor` | Sets a variable from a value expression |
| `SetTextVariableExecutor` | Sets a variable from a text template |

### Interpreter/Visitor

The workflow interpretation uses a visitor pattern:
- `WorkflowActionVisitor` — visits workflow elements and constructs executor graph
- `WorkflowElementWalker` — walks the element tree
- Various executor types translate declarative actions into workflow graph nodes

---

## 10. Microsoft.Agents.AI.Workflows.Declarative.AzureAI

**Purpose**: Azure AI Foundry agent provider for declarative workflows.

### NuGet Dependencies

| Package | Notes |
|---------|-------|
| `Azure.Identity` | Azure authentication |
| `Azure.AI.Projects` | Foundry SDK |
| `Azure.AI.Projects.OpenAI` | OpenAI integration for Foundry |
| Project ref: `Microsoft.Agents.AI.AzureAI` | Azure AI agent support |
| Project ref: `Microsoft.Agents.AI.Workflows.Declarative` | |

### Classes

| Type | Kind | Description |
|------|------|-------------|
| `AzureAgentProvider` | `sealed class : WorkflowAgentProvider` | Connects to Azure AI Foundry projects. Uses `AIProjectClient` and `ProjectConversationsClient`. Caches agent versions and agent instances. Configurable with `AIProjectClientOptions`, `ProjectOpenAIClientOptions`, and optional `HttpClient`. |

---

## 11. Microsoft.Agents.AI.Workflows.Generators

**Purpose**: Roslyn incremental source generator for compile-time workflow route configuration.

### NuGet Dependencies

| Package | Notes |
|---------|-------|
| `Microsoft.CodeAnalysis.CSharp` v4.4.0 | Roslyn APIs |
| Targets `netstandard2.0` only | Required for analyzers/generators |

### Generator Pipeline

| Type | Kind | Description |
|------|------|-------------|
| `ExecutorRouteGenerator` | `class : IIncrementalGenerator` | Main generator. Three pipelines: (1) Methods with `[MessageHandler]`, (2) Classes with `[SendsMessage]`, (3) Classes with `[YieldsOutput]`. Combines results, generates `ConfigureRoutes()` overrides and protocol type overrides. |
| `SemanticAnalyzer` | `static class` | Analyzes methods/classes via Roslyn semantic model. `AnalyzeHandlerMethod()`, `AnalyzeClassProtocolAttribute()`, `CombineHandlerMethodResults()`, `CombineProtocolOnlyResults()`. |
| `SourceBuilder` | `static class` | Generates C# source code. Outputs partial class with `ConfigureRoutes()`, `ConfigureSentTypes()`, `ConfigureYieldTypes()` overrides. |

### Generated Code Pattern

For an executor class with `[MessageHandler]` attributed methods, the generator produces:

```csharp
partial class MyExecutor
{
    protected override RouteBuilder ConfigureRoutes(RouteBuilder routeBuilder)
    {
        return routeBuilder
            .AddHandler<InputType>(this.HandleAsync)
            .AddHandler<OtherInput, OutputType>(this.OtherHandleAsync);
    }

    protected override ISet<Type> ConfigureSentTypes() { ... }
    protected override ISet<Type> ConfigureYieldTypes() { ... }
}
```

### Model Types

| Type | Kind | Description |
|------|------|-------------|
| `ExecutorInfo` | `record` | All info needed to generate code for one executor class: Namespace, ClassName, GenericParameters, Handlers[], ClassSendTypes[], ClassYieldTypes[]. |
| `HandlerInfo` | `record` | Single handler method: MethodName, InputTypeName, OutputTypeName, SignatureKind, YieldTypes, SendTypes. |
| `HandlerSignatureKind` | `enum` | `VoidSync`, `VoidAsync`, `ResultSync`, `ResultAsync`. |
| `AnalysisResult` | `class` | Combines `ExecutorInfo` with `Diagnostic[]`. |
| `MethodAnalysisResult` | `record` | Per-method analysis: class context + handler info + diagnostics. |
| `ClassProtocolInfo` | `record` | Class-level `[SendsMessage]`/`[YieldsOutput]` info. |
| `DiagnosticInfo` / `DiagnosticLocationInfo` | `record` | Value-equatable diagnostic wrappers for caching. |
| `ImmutableEquatableArray<T>` | `class` | Immutable array with sequence equality (required for generator caching). |
| `ProtocolAttributeKind` | `enum` | `Send`, `Yield`. |

---

## 12. Microsoft.Agents.AI.DurableTask

**Purpose**: Distributed durable execution for agents using the Durable Task Framework. Agent state persists across crashes/restarts via entity-backed state.

### NuGet Dependencies

| Package | Notes |
|---------|-------|
| `Microsoft.DurableTask.Client` | Durable Task client |
| `Microsoft.DurableTask.Worker` | Durable Task worker (entities, orchestrations) |
| Project ref: `Microsoft.Agents.AI.Abstractions` | |
| Project ref: `Microsoft.Agents.AI` | |
| Project ref: `Microsoft.Agents.AI.Hosting` | |

### Core Classes

| Type | Kind | Description |
|------|------|-------------|
| `DurableAIAgent` | `sealed class : AIAgent` | Agent that communicates via Durable Task entity methods. Used within orchestrations. Gets session from `TaskOrchestrationContext.NewAgentSessionId()`. **Cancellation not supported** (throws `NotSupportedException`). |
| `DurableAIAgentProxy` | `class : AIAgent` | Proxy for out-of-orchestration interaction with durable agents. Uses `IDurableAgentClient` to signal entities. **Streaming not supported** (throws `NotSupportedException`). Supports fire-and-forget via `DurableAgentRunOptions.IsFireAndForget`. |
| `EntityAgentWrapper` | `sealed class : DelegatingAIAgent` | Wraps an inner `AIAgent` within entity context. Provides entity-scoped `IServiceProvider`. Sets entity ID as agent ID. |
| `AgentEntity` | `class` | Durable Task entity that holds agent state. Handles `RunRequest` signals, manages chat history, runs the inner agent, stores responses. |
| `DurableAgentSession` | `sealed class : AgentSession` | Session backed by `AgentSessionId`. Serializes/deserializes to JSON. |
| `DurableAgentContext` | `class` | Execution context for durable agent runs. |
| `AgentSessionId` | `record struct` | Composite key: `{agentName}:{sessionKey}`. `Parse()`, `ToString()`, `ToEntityName()`, `WithRandomKey()`. |
| `DurableAgentsOptions` | `sealed class` | Configuration: `AddAIAgentFactory()`, `AddAIAgent()`, `DefaultTimeToLive` (14 days default), `MinimumTimeToLiveSignalDelay`. |
| `DefaultDurableAgentClient` | `class : IDurableAgentClient` | Client that signals entities via `DurableTaskClient` and polls for responses. |
| `RunRequest` | `record` | Request payload: `Messages[]`, `ResponseFormat`, `EnableToolCalls`, `EnableToolNames[]`, `CorrelationId`, `OrchestrationId`. |
| `DurableAgentRunOptions` | `sealed class : AgentRunOptions` | Extended options: `EnableToolCalls`, `EnableToolNames`, `ResponseFormat`, `IsFireAndForget`. |
| `AgentRunHandle` | ` class` | Handle for reading agent responses after signalling. |

### State Model

| Type | Kind | Description |
|------|------|-------------|
| `DurableAgentState` | `class` | Root state: entries list, session metadata. |
| `DurableAgentStateEntry` | `abstract class` | Base entry (polymorphic: request or response). |
| `DurableAgentStateMessage` | `class` | Individual message within state. |
| `DurableAgentStateContent` | `abstract class` | Content hierarchy: `TextContent`, `ErrorContent`, `FunctionCallContent`, `FunctionResultContent`, `DataContent`, `UriContent`, `HostedFileContent`, `HostedVectorStoreContent`, `ReasoningContent`, `UsageContent`, `UnknownContent`. |

### Interfaces

| Type | Description |
|------|-------------|
| `IDurableAgentClient` | `RunAgentAsync(sessionId, request)` → `AgentRunHandle` |
| `IAgentResponseHandler` | `OnStreamingResponseUpdateAsync()`, `OnAgentResponseAsync()` — for processing agent responses (e.g., sending to user). |

### Serialization

- `DurableAgentJsonUtilities`: Source-generated `JsonSerializerContext` with `JsonSerializerDefaults.Web`. Chains `AgentAbstractionsJsonUtilities.DefaultOptions.TypeInfoResolver` for cross-package type coverage. AOT and trimming friendly.
- Custom `JsonStringEnumConverter` added when reflection is enabled.

### Service Registration

```csharp
services.ConfigureDurableAgents(options => { ... }, workerBuilder, clientBuilder);
services.GetDurableAgentProxy("agentName");
```

### Orchestration Extensions

```csharp
TaskOrchestrationContext.GetAgent("agentName") → DurableAIAgent
TaskOrchestrationContext.NewAgentSessionId("agentName") → AgentSessionId
```

---

## 13. Microsoft.Agents.AI.Declarative

**Purpose**: Declarative agent support — creates `AIAgent` instances from YAML definitions. **Not published as NuGet** (`IsPackable=false`).

### NuGet Dependencies

| Package | Notes |
|---------|-------|
| `Microsoft.Agents.ObjectModel` | Foundry object model |
| `Microsoft.Agents.ObjectModel.Yaml` | YAML deserialization |
| `Microsoft.PowerFx.Interpreter` | Expression evaluation |
| `Microsoft.Extensions.Configuration` | Configuration |
| Project ref: `Microsoft.Agents.AI` | |

### Factory Pattern

| Type | Kind | Description |
|------|------|-------------|
| `PromptAgentFactory` | `abstract class` | Base factory. Holds a `RecalcEngine` for PowerFx expression evaluation. `CreateAsync()` creates agent from `GptComponentMetadata`. `TryCreateAsync()` returns null if unsupported. |
| `AggregatorPromptAgentFactory` | `sealed class : PromptAgentFactory` | Aggregates multiple factories. First factory that supports the definition wins. |
| `ChatClientPromptAgentFactory` | `class : PromptAgentFactory` | Creates agents backed by `IChatClient`. Configures tools, chat options, response format from declarative definition. |

### YAML Support

| Type | Kind | Description |
|------|------|-------------|
| `AgentBotElementYaml` | `static class` | `FromYaml()` — parses YAML into `GptComponentMetadata` model. |
| `YamlAgentFactoryExtensions` | `static class` | `CreateFromYamlAsync()` — convenience extension on `PromptAgentFactory`. |

### Tool Extensions

| Type | Creates |
|------|---------|
| `CodeInterpreterToolExtensions` | `HostedCodeInterpreterTool` |
| `FileSearchToolExtensions` | `HostedFileSearchTool` |
| `FunctionToolExtensions` | `AIFunctionDeclaration` or matched `AIFunction` |
| `McpServerToolExtensions` | `HostedMcpServerTool` |
| `WebSearchToolExtensions` | `HostedWebSearchTool` |
| `McpServerToolApprovalModeExtensions` | `HostedMcpServerToolApprovalMode` |

### Expression Evaluator Extensions

| Type | Evaluates |
|------|-----------|
| `IntExpressionExtensions` | `IntExpression` → `long?` |
| `NumberExpressionExtensions` | `NumberExpression` → `double?` |
| `StringExpressionExtensions` | `StringExpression` → `string?` |
| `BoolExpressionExtensions` | `BoolExpression` → `bool?` |

### Other Extensions

| Type | Description |
|------|-------------|
| `ModelOptionsExtensions` | Converts `ModelOptions` → `ChatToolMode`, extracts `AdditionalProperties` |
| `PromptAgentExtensions` | `GetChatOptions()` — converts `GptComponentMetadata` → `ChatOptions` |
| `RecordDataTypeExtensions` | `AsChatResponseFormat()` → `ChatResponseFormat.ForJsonSchema()` |
| `PropertyInfoExtensions` | `AsObjectDictionary()` — converts property metadata to schema dictionaries |

---

## 14. Cross-Cutting Patterns

### Serialization

All projects consistently use **System.Text.Json** with **source-generated `JsonSerializerContext`** classes for AOT compatibility:

- `A2AJsonUtilities` (A2A)
- `AGUIJsonSerializerContext` (AGUI)
- `DurableAgentJsonUtilities` (DurableTask)
- `WorkflowsJsonUtilities` (Workflows)

Common configuration: `JsonSerializerDefaults.Web`, `WhenWritingNull`, `AllowReadingFromString`, `JavaScriptEncoder.UnsafeRelaxedJsonEscaping`. Type info resolver chaining for cross-package type support.

### Streaming Protocols

| Protocol | Transport | Content-Type | Implementation |
|----------|-----------|--------------|----------------|
| A2A | SSE via `A2AClient` | `text/event-stream` | `A2AAgent.RunCoreStreamingAsync()` |
| AG-UI | SSE via `SseParser`/`SseFormatter` | `text/event-stream` | `AGUIServerSentEventsResult` |
| OpenAI Chat Completions | SSE via `SseJsonResult<T>` | `text/event-stream` | `SseJsonResult<ChatCompletionChunk>` |
| OpenAI Responses | SSE via `SseJsonResult<T>` | `text/event-stream` | Same SSE infrastructure |
| Workflows | In-process `IAsyncEnumerable<WorkflowEvent>` via channels | N/A (in-process) | `StreamingRun.WatchStreamAsync()` |

All SSE implementations set: `Content-Type: text/event-stream`, `Cache-Control: no-cache,no-store`. OpenAI also adds `Connection: keep-alive`.

### Agent Registration & Resolution

| Hosting | Registration | Resolution |
|---------|-------------|------------|
| A2A AspNetCore | Keyed service in DI | `GetRequiredKeyedService<AIAgent>(agentName)` |
| AGUI AspNetCore | Keyed service in DI | `GetRequiredKeyedService<AIAgent>(agentName)` |
| OpenAI Hosting | Keyed service in DI | `GetRequiredKeyedService<AIAgent>(agentName)` |
| Azure Functions | `DurableAgentsOptions.AddAIAgent()` / `AddAIAgentFactory()` | Entity-based via `AgentSessionId` |
| DurableTask (library) | `ConfigureDurableAgents()` on `IServiceCollection` | `GetDurableAgentProxy()` or keyed service |
| Workflows | Direct `AIAgent` instances via `AIAgentBinding` | Executor graph resolution |

### Middleware Pipeline

| Hosting | Middleware |
|---------|-----------|
| Azure Functions | `BuiltInFunctionExecutionMiddleware` → sets `IFunctionExecutor` for custom entry points |
| ASP.NET Core (all) | Standard ASP.NET Core endpoint routing via `MapXxx()` extension methods |
| Workflows | Internal execution pipeline: `InProcessRunner` → super-step execution → edge runners → executor invocation → message routing |

### Target Framework Support

Most projects multi-target: `net472`, `netstandard2.0`, `net8.0`, `net9.0`, `net10.0`.

Exceptions:
- ASP.NET Core hosting projects: `net8.0`, `net9.0`, `net10.0` only
- Azure Functions: `net8.0` only
- Workflows.Generators: `netstandard2.0` only (Roslyn requirement)
- Workflows.Declarative.AzureAI: `net8.0`, `net9.0`, `net10.0`
