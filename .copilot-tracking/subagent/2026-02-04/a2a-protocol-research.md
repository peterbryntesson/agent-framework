# A2A Protocol Implementation Analysis

**Date:** 2026-02-04
**Status:** Complete
**Researcher:** GitHub Copilot

## Executive Summary

The Microsoft Agent Framework implements comprehensive A2A (Agent-to-Agent) protocol support in both .NET and Python. The implementations rely on external A2A SDK libraries (`A2A` NuGet package for .NET, `a2a-sdk` Python package) and provide adapters to bridge A2A protocol types with the framework's native agent abstractions.

---

## 1. Protocol Specification Summary

### 1.1 A2A Protocol Overview

The A2A protocol is a standardized communication protocol for agent interoperability. Key specification references:

- **Official Specification**: <https://a2a-protocol.org/latest/>
- **GitHub Reference**: <https://github.com/a2aproject/A2A>

### 1.2 Core Concepts

| Concept | Description |
|---------|-------------|
| **AgentCard** | Metadata describing agent capabilities, exposed at `/.well-known/agent.json` |
| **Task** | Long-running operation with ID, status, and history |
| **Message** | Communication unit with parts (text, file, data) |
| **Part** | Content container: `TextPart`, `FilePart`, `DataPart` |
| **TaskStatus** | State enumeration: `submitted`, `working`, `completed`, `failed`, `canceled`, `rejected` |
| **Artifact** | Output produced by a task |

### 1.3 Transport Protocols

- **JSON-RPC** over HTTP (primary)
- **SSE (Server-Sent Events)** for streaming responses

### 1.4 Discovery Mechanisms

1. **Well-Known URI**: `GET /.well-known/agent.json`
2. **Curated Registries**: Catalog-based discovery
3. **Direct Configuration**: Private/manual discovery

---

## 2. .NET Implementation

### 2.1 Package Structure

| Package | Purpose |
|---------|---------|
| `Microsoft.Agents.AI.A2A` | Client-side A2A agent wrapper |
| `Microsoft.Agents.AI.Hosting.A2A` | Server-side hosting extensions |
| `Microsoft.Agents.AI.Hosting.A2A.AspNetCore` | ASP.NET Core endpoint mapping |

### 2.2 Key Files and Interfaces

#### A2A Client (Calling Remote Agents)

| File | Line Range | Description |
|------|------------|-------------|
| [A2AAgent.cs](dotnet/src/Microsoft.Agents.AI.A2A/A2AAgent.cs#L1-L350) | L1-350 | Main A2A agent implementation wrapping `A2AClient` |
| [A2AAgentSession.cs](dotnet/src/Microsoft.Agents.AI.A2A/A2AAgentSession.cs#L1-L67) | L1-67 | Session state with `ContextId` and `TaskId` |
| [A2AContinuationToken.cs](dotnet/src/Microsoft.Agents.AI.A2A/A2AContinuationToken.cs#L1-L82) | L1-82 | Token for resuming background task responses |

#### Extension Methods

| File | Line Range | Description |
|------|------------|-------------|
| [A2AClientExtensions.cs](dotnet/src/Microsoft.Agents.AI.A2A/Extensions/A2AClientExtensions.cs#L1-L44) | L1-44 | `A2AClient.AsAIAgent()` wrapper |
| [A2ACardResolverExtensions.cs](dotnet/src/Microsoft.Agents.AI.A2A/Extensions/A2ACardResolverExtensions.cs#L1-L48) | L1-48 | `A2ACardResolver.GetAIAgentAsync()` |
| [A2AAgentCardExtensions.cs](dotnet/src/Microsoft.Agents.AI.A2A/Extensions/A2AAgentCardExtensions.cs#L1-L42) | L1-42 | `AgentCard.AsAIAgent()` |
| [A2AAgentTaskExtensions.cs](dotnet/src/Microsoft.Agents.AI.A2A/Extensions/A2AAgentTaskExtensions.cs#L1-L47) | L1-47 | Task to `ChatMessage` conversion |
| [A2AArtifactExtensions.cs](dotnet/src/Microsoft.Agents.AI.A2A/Extensions/A2AArtifactExtensions.cs#L1-L28) | L1-28 | Artifact to `AIContent` conversion |
| [ChatMessageExtensions.cs](dotnet/src/Microsoft.Agents.AI.A2A/Extensions/ChatMessageExtensions.cs#L1-L37) | L1-37 | Framework messages to A2A `AgentMessage` |

#### A2A Server (Exposing Agents)

| File | Line Range | Description |
|------|------------|-------------|
| [AIAgentExtensions.cs](dotnet/src/Microsoft.Agents.AI.Hosting.A2A/AIAgentExtensions.cs#L1-L100) | L1-100 | `AIAgent.MapA2A()` attaches A2A capabilities |
| [EndpointRouteBuilderExtensions.cs](dotnet/src/Microsoft.Agents.AI.Hosting.A2A.AspNetCore/EndpointRouteBuilderExtensions.cs#L1-L247) | L1-247 | ASP.NET Core `MapA2A()` endpoint routing |
| [MessageConverter.cs](dotnet/src/Microsoft.Agents.AI.Hosting.A2A/Converters/MessageConverter.cs#L1-L53) | L1-53 | Bidirectional message conversion |

### 2.3 A2AAgent Class API

```csharp
public sealed class A2AAgent : AIAgent
{
    // Constructor
    public A2AAgent(
        A2AClient a2aClient,
        string? id = null,
        string? name = null,
        string? description = null,
        ILoggerFactory? loggerFactory = null);

    // Session Management
    public override ValueTask<AgentSession> GetNewSessionAsync(CancellationToken ct);
    public ValueTask<AgentSession> GetNewSessionAsync(string contextId);
    public override ValueTask<AgentSession> DeserializeSessionAsync(JsonElement serialized, ...);

    // Agent Execution (inherited from AIAgent)
    protected override Task<AgentResponse> RunCoreAsync(
        IEnumerable<ChatMessage> messages,
        AgentSession? session,
        AgentRunOptions? options,
        CancellationToken ct);

    protected override IAsyncEnumerable<AgentResponseUpdate> RunCoreStreamingAsync(
        IEnumerable<ChatMessage> messages,
        AgentSession? session,
        AgentRunOptions? options,
        CancellationToken ct);
}
```

### 2.4 Server-Side Hosting API

```csharp
// Attach A2A to an agent
ITaskManager taskManager = agent.MapA2A(
    agentCard: agentCard,
    taskManager: null,
    loggerFactory: loggerFactory,
    agentSessionStore: sessionStore);

// Map endpoints in ASP.NET Core
app.MapA2A(agent, "/a2a/myagent");
app.MapA2A(agent, "/a2a/myagent", agentCard);
app.MapA2A(agentBuilder, "/a2a/myagent", configureTaskManager);
```

### 2.5 Dependencies

```xml
<PackageReference Include="A2A" />
<ProjectReference Include="..\Microsoft.Agents.AI.Abstractions\..." />
```

---

## 3. Python Implementation

### 3.1 Package Structure

| Package | Purpose |
|---------|---------|
| `agent-framework-a2a` | A2A agent integration package |
| `agent-framework-core` (re-export) | Core package re-exports `A2AAgent` via `agent_framework.a2a` |

### 3.2 Key Files and Interfaces

| File | Line Range | Description |
|------|------------|-------------|
| [_agent.py](python/packages/a2a/agent_framework_a2a/_agent.py#L1-L440) | L1-440 | Complete `A2AAgent` implementation |
| [__init__.py](python/packages/a2a/agent_framework_a2a/__init__.py#L1-L16) | L1-16 | Package exports |

### 3.3 A2AAgent Class API

```python
class A2AAgent(BaseAgent):
    """A2A protocol agent wrapper."""

    AGENT_PROVIDER_NAME: Final[str] = "A2A"

    def __init__(
        self,
        *,
        name: str | None = None,
        id: str | None = None,
        description: str | None = None,
        agent_card: AgentCard | None = None,
        url: str | None = None,
        client: Client | None = None,
        http_client: httpx.AsyncClient | None = None,
        auth_interceptor: AuthInterceptor | None = None,
        timeout: float | httpx.Timeout | None = None,
        **kwargs: Any,
    ) -> None: ...

    async def run(
        self,
        messages: str | ChatMessage | Sequence[str | ChatMessage] | None = None,
        *,
        thread: AgentThread | None = None,
        **kwargs: Any,
    ) -> AgentResponse: ...

    async def run_stream(
        self,
        messages: str | ChatMessage | Sequence[str | ChatMessage] | None = None,
        *,
        thread: AgentThread | None = None,
        **kwargs: Any,
    ) -> AsyncIterable[AgentResponseUpdate]: ...

    # Internal conversion methods
    def _prepare_message_for_a2a(self, message: ChatMessage) -> A2AMessage: ...
    def _parse_contents_from_a2a(self, parts: Sequence[A2APart]) -> list[Content]: ...
    def _parse_messages_from_task(self, task: Task) -> list[ChatMessage]: ...
```

### 3.4 Content Type Mapping

| Framework Content Type | A2A Part Type |
|------------------------|---------------|
| `text` | `TextPart` |
| `error` | `TextPart` (with error message) |
| `uri` | `FilePart(FileWithUri)` |
| `data` | `FilePart(FileWithBytes)` |
| `hosted_file` | `FilePart(FileWithUri)` |

### 3.5 Dependencies

```toml
dependencies = [
    "agent-framework-core",
    "a2a-sdk>=0.3.5",
]
```

---

## 4. API Surface Comparison

### 4.1 Agent Creation

| Aspect | .NET | Python |
|--------|------|--------|
| **Constructor** | `new A2AAgent(a2aClient, id, name, description)` | `A2AAgent(url=..., agent_card=..., client=...)` |
| **From URL** | Via `A2ACardResolver.GetAIAgentAsync()` | `A2AAgent(url="...")` |
| **From AgentCard** | `agentCard.AsAIAgent()` | `A2AAgent(agent_card=card)` |
| **From Client** | `a2aClient.AsAIAgent()` | `A2AAgent(client=client)` |

### 4.2 Execution Methods

| Method | .NET | Python |
|--------|------|--------|
| **Sync Run** | `RunAsync(messages, session, options)` | `await agent.run(messages, thread=thread)` |
| **Stream Run** | `RunStreamingAsync(messages, session, options)` | `async for update in agent.run_stream(...)` |
| **Response Type** | `AgentResponse` | `AgentResponse` |
| **Update Type** | `AgentResponseUpdate` | `AgentResponseUpdate` |

### 4.3 Session/Context Management

| Aspect | .NET | Python |
|--------|------|--------|
| **Session Type** | `A2AAgentSession` | Uses `AgentThread` (framework standard) |
| **Context ID** | `session.ContextId` | Managed internally via client |
| **Task ID** | `session.TaskId` | Managed internally via client |
| **Serialization** | `session.Serialize()` / `DeserializeSessionAsync()` | Not explicitly implemented |

### 4.4 Server Hosting

| Aspect | .NET | Python |
|--------|------|--------|
| **Hosting Support** | Yes - `MapA2A()` extension | Not implemented (client only) |
| **Card Endpoint** | `GET /.well-known/agent.json` or `/v1/card` | N/A |
| **Message Endpoint** | Via `ITaskManager` | N/A |
| **Streaming** | SSE via A2A SDK | N/A |

---

## 5. Message Format and Endpoint Definitions

### 5.1 A2A Message Structure

```json
{
  "messageId": "string",
  "contextId": "string (optional)",
  "role": "user" | "agent",
  "parts": [
    { "kind": "text", "text": "string", "metadata": {} },
    { "kind": "file", "file": { "uri": "string", "mimeType": "string" } },
    { "kind": "data", "data": {} }
  ],
  "metadata": {}
}
```

### 5.2 A2A Task Structure

```json
{
  "id": "string",
  "contextId": "string",
  "status": {
    "state": "submitted" | "working" | "completed" | "failed" | "canceled" | "rejected",
    "message": "string (optional)"
  },
  "artifacts": [...],
  "history": [...],
  "metadata": {}
}
```

### 5.3 AgentCard Structure

```json
{
  "name": "string",
  "description": "string",
  "url": "string",
  "version": "string",
  "capabilities": {...},
  "authentication": {...}
}
```

### 5.4 Endpoints (Server-Side)

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/.well-known/agent.json` | GET | Agent card discovery |
| `/v1/card` | GET | Agent card (alternative path) |
| `/v1/message` | POST | Send message (JSON-RPC) |
| `/v1/message/stream` | POST | Send message with SSE streaming |
| `/v1/task/{id}` | GET | Get task status |
| `/v1/task/{id}/subscribe` | GET | Subscribe to task updates (SSE) |

---

## 6. Design Recommendations for Go Implementation

### 6.1 Package Structure

```
protocol/
└── a2a/
    ├── doc.go           # Package documentation
    ├── types.go         # Core A2A types (AgentCard, Task, Message, Part)
    ├── agent.go         # A2AAgent implementing agent.Agent interface
    ├── client.go        # HTTP client for calling remote A2A agents
    ├── server.go        # HTTP server for exposing agents via A2A
    ├── session.go       # A2ASession with ContextID/TaskID tracking
    └── converter.go     # Type conversion helpers
```

### 6.2 Core Types

```go
// AgentCard represents A2A agent metadata
type AgentCard struct {
    Name           string                 `json:"name"`
    Description    string                 `json:"description,omitempty"`
    URL            string                 `json:"url"`
    Version        string                 `json:"version,omitempty"`
    Capabilities   map[string]interface{} `json:"capabilities,omitempty"`
    Authentication *AuthConfig            `json:"authentication,omitempty"`
}

// TaskState enumeration
type TaskState string
const (
    TaskStateSubmitted TaskState = "submitted"
    TaskStateWorking   TaskState = "working"
    TaskStateCompleted TaskState = "completed"
    TaskStateFailed    TaskState = "failed"
    TaskStateCanceled  TaskState = "canceled"
    TaskStateRejected  TaskState = "rejected"
)

// Task represents a long-running A2A operation
type Task struct {
    ID        string     `json:"id"`
    ContextID string     `json:"contextId,omitempty"`
    Status    TaskStatus `json:"status"`
    Artifacts []Artifact `json:"artifacts,omitempty"`
    History   []Message  `json:"history,omitempty"`
    Metadata  Metadata   `json:"metadata,omitempty"`
}

// Message represents an A2A protocol message
type Message struct {
    MessageID        string   `json:"messageId"`
    ContextID        string   `json:"contextId,omitempty"`
    Role             Role     `json:"role"`
    Parts            []Part   `json:"parts"`
    ReferenceTaskIDs []string `json:"referenceTaskIds,omitempty"`
    Metadata         Metadata `json:"metadata,omitempty"`
}

// Part is a discriminated union of content types
type Part struct {
    Kind     string      `json:"kind"` // "text", "file", "data"
    Text     string      `json:"text,omitempty"`
    File     *FilePart   `json:"file,omitempty"`
    Data     interface{} `json:"data,omitempty"`
    Metadata Metadata    `json:"metadata,omitempty"`
}
```

### 6.3 Client Interface

```go
type Client interface {
    // GetAgentCard retrieves the agent's metadata
    GetAgentCard(ctx context.Context) (*AgentCard, error)

    // SendMessage sends a message and returns immediate response
    SendMessage(ctx context.Context, message *Message, metadata Metadata) (*Response, error)

    // SendMessageStreaming sends a message and returns SSE event stream
    SendMessageStreaming(ctx context.Context, message *Message, metadata Metadata) (<-chan Event, error)

    // GetTask retrieves task status by ID
    GetTask(ctx context.Context, taskID string) (*Task, error)

    // SubscribeToTask subscribes to task updates via SSE
    SubscribeToTask(ctx context.Context, taskID string) (<-chan Event, error)
}

func NewClient(baseURL string, opts ...ClientOption) (*client, error)
```

### 6.4 Agent Implementation

```go
// A2AAgent wraps an A2A client as a local agent
type A2AAgent struct {
    client      Client
    id          string
    name        string
    description string
    logger      *slog.Logger
}

func NewA2AAgent(client Client, opts ...AgentOption) *A2AAgent

// Implements agent.Agent interface
func (a *A2AAgent) Run(ctx context.Context, messages []chat.Message, opts ...agent.RunOption) (*agent.Response, error)
func (a *A2AAgent) RunStream(ctx context.Context, messages []chat.Message, opts ...agent.RunOption) (<-chan agent.ResponseUpdate, error)
func (a *A2AAgent) GetSession(ctx context.Context) (agent.Session, error)
```

### 6.5 Server Implementation

```go
// Server exposes a local agent via A2A protocol
type Server struct {
    agent        agent.Agent
    agentCard    *AgentCard
    sessionStore SessionStore
    handler      http.Handler
}

func NewServer(agent agent.Agent, card *AgentCard, opts ...ServerOption) *Server

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request)

// Endpoints handled:
// GET  /.well-known/agent.json  -> GetAgentCard
// GET  /v1/card                 -> GetAgentCard (alias)
// POST /v1/message              -> HandleMessage
// POST /v1/message/stream       -> HandleMessageStreaming (SSE)
// GET  /v1/task/{id}            -> GetTask
// GET  /v1/task/{id}/subscribe  -> SubscribeToTask (SSE)
```

### 6.6 Session Management

```go
type A2ASession struct {
    id        string
    contextID string
    taskID    string
    messages  []chat.Message
    mu        sync.RWMutex
}

func (s *A2ASession) ID() string
func (s *A2ASession) ContextID() string
func (s *A2ASession) TaskID() string
func (s *A2ASession) Messages() []chat.Message
func (s *A2ASession) AddMessage(msg chat.Message)
func (s *A2ASession) Serialize() ([]byte, error)

func DeserializeSession(data []byte) (*A2ASession, error)
```

### 6.7 Key Implementation Notes

1. **Use Standard Library**: Prefer `net/http` and `encoding/json` for HTTP and JSON handling
2. **SSE Support**: Implement SSE client/server using chunked transfer encoding
3. **Error Handling**: Define sentinel errors: `ErrTaskNotFound`, `ErrAgentUnavailable`, etc.
4. **Context Propagation**: Pass `context.Context` through all operations
5. **Streaming**: Use channels for streaming responses with proper cleanup
6. **Type Conversion**: Create bidirectional converters between `chat.Message` and A2A `Message`
7. **Testing**: Mock the HTTP transport layer for unit testing

---

## 7. Sample Code References

### 7.1 .NET Sample

Location: `dotnet/samples/A2AClientServer/` (referenced in Python sample README)

### 7.2 Python Sample

Location: [python/samples/getting_started/agents/a2a/agent_with_a2a.py](python/samples/getting_started/agents/a2a/agent_with_a2a.py)

```python
# Example usage
async with httpx.AsyncClient(timeout=60.0) as http_client:
    resolver = A2ACardResolver(httpx_client=http_client, base_url=a2a_agent_host)
    agent_card = await resolver.get_agent_card()

    agent = A2AAgent(
        name=agent_card.name,
        description=agent_card.description,
        agent_card=agent_card,
        url=a2a_agent_host,
    )

    response = await agent.run("What are your capabilities?")
```

---

## 8. Gaps and Considerations

### 8.1 Python Server-Side

- Python currently implements **client-only** A2A support
- No equivalent to .NET's `MapA2A()` hosting extensions
- Server hosting would require integration with FastAPI/Starlette

### 8.2 Task Stream Resumption

- .NET explicitly notes (line 141-148 in A2AAgent.cs) that task stream resumption is not well-defined in A2A v2.*
- A2A v3.0 specification improves this with task stream reconnection
- Go implementation should plan for v3.0 compliance

### 8.3 Authentication

- Both implementations support auth interceptors
- Python: `AuthInterceptor` parameter
- .NET: Via `HttpClient` configuration
- Go: Should support `http.RoundTripper` middleware pattern

---

## 9. References

- [A2A Protocol Specification](https://a2a-protocol.org/latest/)
- [A2A GitHub Repository](https://github.com/a2aproject/A2A)
- [Agent Discovery Documentation](https://github.com/a2aproject/A2A/blob/main/docs/topics/agent-discovery.md)
- [Life of a Task](https://github.com/a2aproject/A2A/blob/main/docs/topics/life-of-a-task.md)
