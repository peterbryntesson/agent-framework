# Enterprise Integrations Analysis

**Date**: 2026-02-04
**Objective**: Analyze enterprise integrations (Copilot Studio, GitHub Copilot, Purview, DevUI, Foundry Local) in .NET and Python for Go implementation recommendations.

---

## Table of Contents

1. [Copilot Studio Integration](#1-copilot-studio-integration)
2. [GitHub Copilot Integration](#2-github-copilot-integration)
3. [Microsoft Purview Integration](#3-microsoft-purview-integration)
4. [Developer UI (DevUI)](#4-developer-ui-devui)
5. [Foundry Local](#5-foundry-local)
6. [Go Implementation Recommendations](#6-go-implementation-recommendations)

---

## 1. Copilot Studio Integration

### Overview

Copilot Studio integration enables connection to Microsoft Power Platform hosted AI agents via the Direct-to-Engine protocol.

### .NET Implementation

**Key Files**:

- [CopilotStudioAgent.cs](dotnet/src/Microsoft.Agents.AI.CopilotStudio/CopilotStudioAgent.cs#L1-L166) - Main agent wrapper
- [CopilotStudioAgentSession.cs](dotnet/src/Microsoft.Agents.AI.CopilotStudio/CopilotStudioAgentSession.cs#L1-L30) - Session state management
- [ActivityProcessor.cs](dotnet/src/Microsoft.Agents.AI.CopilotStudio/ActivityProcessor.cs#L1-L47) - Activity-to-ChatMessage conversion

**Client Initialization Pattern** (.NET):

```csharp
// Uses Microsoft.Agents.CopilotStudio.Client package
public CopilotStudioAgent(CopilotClient client, ILoggerFactory? loggerFactory = null)
{
    this.Client = client;
    this._logger = (loggerFactory ?? NullLoggerFactory.Instance).CreateLogger<CopilotStudioAgent>();
}
```

**Session Management**:

- Extends `ServiceIdAgentSession` base class
- Stores `ConversationId` for multi-turn conversations
- Supports session serialization/deserialization via JSON

**API Patterns**:

| Pattern | Description |
|---------|-------------|
| `GetNewSessionAsync()` | Creates new empty session |
| `GetNewSessionAsync(conversationId)` | Resumes existing conversation |
| `DeserializeSessionAsync()` | Reconstructs session from JSON |
| `RunCoreAsync()` | Non-streaming execution |
| `RunCoreStreamingAsync()` | Streaming execution with activity processing |

**Activity Processing** (Lines 16-45):

```csharp
// Converts IActivity stream to ChatMessage stream
public static async IAsyncEnumerable<ChatMessage> ProcessActivityAsync(
    IAsyncEnumerable<IActivity> activities, 
    bool streaming, 
    ILogger logger)
{
    await foreach (IActivity activity in activities)
    {
        if (!string.IsNullOrWhiteSpace(activity.Text))
        {
            if ((activity.Type == "message" && !streaming) || 
                (activity.Type == "typing" && streaming))
            {
                yield return CreateChatMessageFromActivity(activity, [new TextContent(activity.Text)]);
            }
        }
    }
}
```

### Python Implementation

**Key Files**:

- [_agent.py](python/packages/copilotstudio/agent_framework_copilotstudio/_agent.py#L1-L340) - Main agent class
- [_acquire_token.py](python/packages/copilotstudio/agent_framework_copilotstudio/_acquire_token.py#L1-L96) - MSAL token acquisition
- [__init__.py](python/packages/copilotstudio/agent_framework_copilotstudio/__init__.py#L1-L13) - Public exports

**Settings Pattern** (Lines 25-66):

```python
class CopilotStudioSettings(AFBaseSettings):
    """Environment variable prefix: COPILOTSTUDIOAGENT__"""
    env_prefix: ClassVar[str] = "COPILOTSTUDIOAGENT__"
    
    environmentid: str | None = None      # Environment ID
    schemaname: str | None = None         # Agent identifier
    agentappid: str | None = None         # App Registration ID
    tenantid: str | None = None           # Tenant ID
```

**Agent Initialization** (Lines 68-200):

```python
class CopilotStudioAgent(BaseAgent):
    def __init__(
        self,
        client: CopilotClient | None = None,
        settings: ConnectionSettings | None = None,
        *,
        environment_id: str | None = None,
        agent_identifier: str | None = None,
        client_id: str | None = None,
        tenant_id: str | None = None,
        token: str | None = None,
        cloud: PowerPlatformCloud | None = None,
        ...
    )
```

**Token Acquisition** (Lines 25-96):

```python
def acquire_token(*, client_id: str, tenant_id: str, username: str | None = None, ...):
    """MSAL Public Client Application token flow:
    1. Try silent acquisition from cache
    2. Fall back to interactive authentication
    """
    pca = PublicClientApplication(client_id=client_id, authority=authority, token_cache=token_cache)
    # Silent -> Interactive fallback pattern
```

### Key Abstractions for Go

| Abstraction | Purpose |
|-------------|---------|
| `CopilotClient` | External SDK client wrapper |
| `CopilotStudioAgentSession` | Session state with ConversationId |
| `ActivityProcessor` | Activity-to-ChatMessage converter |
| `ConnectionSettings` | Environment/agent configuration |
| Token acquisition | MSAL-based auth flow |

---

## 2. GitHub Copilot Integration

### Overview

GitHub Copilot integration wraps the GitHub Copilot SDK to provide agentic capabilities with tool support, MCP servers, and streaming responses.

### .NET Implementation

**Key Files**:

- [GitHubCopilotAgent.cs](dotnet/src/Microsoft.Agents.AI.GitHub.Copilot/GitHubCopilotAgent.cs#L1-L471) - Main agent class
- [GitHubCopilotAgentSession.cs](dotnet/src/Microsoft.Agents.AI.GitHub.Copilot/GitHubCopilotAgentSession.cs#L1-L53) - Session management
- [CopilotClientExtensions.cs](dotnet/src/Microsoft.Agents.AI.GitHub.Copilot/CopilotClientExtensions.cs#L1-L77) - Client extension methods

**Agent Configuration** (Lines 32-77):

```csharp
public GitHubCopilotAgent(
    CopilotClient copilotClient,
    SessionConfig? sessionConfig = null,
    bool ownsClient = false,
    string? id = null,
    string? name = null,
    string? description = null,
    IList<AITool>? tools = null,
    string? instructions = null)
```

**SessionConfig Properties** (Lines 140-150):

```csharp
SessionConfig sessionConfig = new SessionConfig
{
    Model = ...,
    Tools = ...,
    SystemMessage = ...,
    AvailableTools = ...,
    ExcludedTools = ...,
    OnPermissionRequest = ...,
    McpServers = ...,
    CustomAgents = ...,
    SkillDirectories = ...,
    DisabledSkills = ...,
    Streaming = true
};
```

**Extension Method Pattern** (Lines 30-50):

```csharp
public static AIAgent AsAIAgent(
    this CopilotClient client,
    SessionConfig? sessionConfig = null,
    bool ownsClient = false,
    string? id = null, ...)
{
    return new GitHubCopilotAgent(client, sessionConfig, ownsClient, id, name, description);
}
```

### Python Implementation

**Key Files**:

- [_agent.py](python/packages/github_copilot/agent_framework_github_copilot/_agent.py#L1-L529) - Main agent class
- [_settings.py](python/packages/github_copilot/agent_framework_github_copilot/_settings.py#L1-L49) - Settings model

**Options TypedDict** (Lines 54-82):

```python
class GitHubCopilotOptions(TypedDict, total=False):
    instructions: str               # System message
    cli_path: str                   # Path to Copilot CLI
    model: str                      # Model (e.g., "gpt-5", "claude-sonnet-4")
    timeout: float                  # Request timeout
    log_level: str                  # CLI log level
    on_permission_request: PermissionHandlerType  # Permission handler
    mcp_servers: dict[str, MCPServerConfig]       # MCP server configs
```

**Settings Pattern** (Lines 7-49):

```python
class GitHubCopilotSettings(AFBaseSettings):
    env_prefix: ClassVar[str] = "GITHUB_COPILOT_"
    
    cli_path: str | None = None
    model: str | None = None
    timeout: float | None = None
    log_level: str | None = None
```

**Agent Lifecycle** (Lines 217-265):

```python
async def __aenter__(self) -> "GitHubCopilotAgent[TOptions]":
    await self.start()
    return self

async def __aexit__(self, exc_type, exc_val, exc_tb) -> None:
    await self.stop()

async def start(self) -> None:
    """Initialize CopilotClient and connect to CLI server"""
    self._client = CopilotClient(client_options)
    await self._client.start()

async def stop(self) -> None:
    """Stop client if owned"""
    if self._client and self._owns_client:
        await self._client.stop()
```

**Permission Handling** (Lines 50-52):

```python
PermissionHandlerType = Callable[[PermissionRequest, dict[str, str]], PermissionRequestResult]
```

### Key Abstractions for Go

| Abstraction | Purpose |
|-------------|---------|
| `CopilotClient` | SDK client with lifecycle management |
| `SessionConfig` | Tools, permissions, MCP servers |
| `GitHubCopilotAgentSession` | Session state with SessionId |
| `PermissionHandler` | Callback for permission requests |
| `MCPServerConfig` | MCP server connection configuration |
| Async context manager | Client lifecycle (start/stop) |

---

## 3. Microsoft Purview Integration

### Overview

Purview integration provides data security and governance policy evaluation as middleware, evaluating prompts and responses against enterprise DLP policies.

### .NET Implementation

**Key Files**:

- [PurviewAgent.cs](dotnet/src/Microsoft.Agents.AI.Purview/PurviewAgent.cs#L1-L73) - Middleware agent wrapper
- [PurviewClient.cs](dotnet/src/Microsoft.Agents.AI.Purview/PurviewClient.cs#L1-L324) - Graph API client
- [PurviewSettings.cs](dotnet/src/Microsoft.Agents.AI.Purview/PurviewSettings.cs#L1-L84) - Configuration
- [ScopedContentProcessor.cs](dotnet/src/Microsoft.Agents.AI.Purview/ScopedContentProcessor.cs#L1-L346) - Policy evaluation processor
- [PurviewExtensions.cs](dotnet/src/Microsoft.Agents.AI.Purview/PurviewExtensions.cs#L1-L119) - Builder extensions

**Middleware Pattern** (PurviewAgent Lines 18-66):

```csharp
internal class PurviewAgent : AIAgent, IDisposable
{
    private readonly AIAgent _innerAgent;
    private readonly PurviewWrapper _purviewWrapper;

    protected override Task<AgentResponse> RunCoreAsync(...)
    {
        // Wraps inner agent execution with Purview policy evaluation
        return this._purviewWrapper.ProcessAgentContentAsync(
            messages, session, options, this._innerAgent, cancellationToken);
    }
}
```

**Settings Configuration** (Lines 13-84):

```csharp
public class PurviewSettings
{
    public string AppName { get; set; }
    public string? AppVersion { get; set; }
    public string? TenantId { get; set; }
    public PurviewAppLocation? PurviewAppLocation { get; set; }
    public bool IgnoreExceptions { get; set; }
    public Uri GraphBaseUri { get; set; } = new Uri("https://graph.microsoft.com/v1.0/");
    public string BlockedPromptMessage { get; set; } = "Prompt blocked by policies";
    public string BlockedResponseMessage { get; set; } = "Response blocked by policies";
    public long? InMemoryCacheSizeLimit { get; set; } = 100_000_000;
    public TimeSpan CacheTTL { get; set; } = TimeSpan.FromMinutes(30);
}
```

**Graph API Client Pattern** (Lines 100-170):

```csharp
public async Task<ProcessContentResponse> ProcessContentAsync(ProcessContentRequest request, CancellationToken cancellationToken)
{
    var token = await this._tokenCredential.GetTokenAsync(...);
    string uri = $"{this._graphUri}/users/{userId}/dataSecurityAndGovernance/processContent";
    
    using (HttpRequestMessage message = new(HttpMethod.Post, new Uri(uri)))
    {
        message.Headers.Add("Authorization", $"Bearer {token.Token}");
        message.Headers.Add("If-None-Match", request.ScopeIdentifier);  // Caching
        // POST and handle response
    }
}
```

**Builder Extensions** (Lines 50-78):

```csharp
public static AIAgentBuilder WithPurview(
    this AIAgentBuilder builder, 
    TokenCredential tokenCredential, 
    PurviewSettings purviewSettings, 
    ILogger? logger = null, 
    IDistributedCache? cache = null)
{
    PurviewWrapper purviewWrapper = CreateWrapper(tokenCredential, purviewSettings, logger, cache);
    return builder.Use((innerAgent) => new PurviewAgent(innerAgent, purviewWrapper));
}
```

### Python Implementation

**Key Files**:

- [_middleware.py](python/packages/purview/agent_framework_purview/_middleware.py#L1-L192) - Middleware classes
- [_client.py](python/packages/purview/agent_framework_purview/_client.py#L1-L197) - Async Graph client
- [_settings.py](python/packages/purview/agent_framework_purview/_settings.py#L1-L90) - Pydantic settings
- [_processor.py](python/packages/purview/agent_framework_purview/_processor.py#L1-L341) - Content processing

**Middleware Classes** (Lines 18-100):

```python
class PurviewPolicyMiddleware(AgentMiddleware):
    """Agent middleware for Purview policy evaluation"""
    
    async def process(self, context: AgentRunContext, next: Callable[...]):
        # Pre-check (prompt)
        should_block_prompt, resolved_user_id = await self._processor.process_messages(
            context.messages, Activity.UPLOAD_TEXT)
        if should_block_prompt:
            context.result = AgentResponse(messages=[...blocked message...])
            context.terminate = True
            return
        
        await next(context)
        
        # Post-check (response)
        if context.result and not context.is_streaming:
            should_block_response, _ = await self._processor.process_messages(
                context.result.messages, Activity.UPLOAD_TEXT, user_id=resolved_user_id)
            if should_block_response:
                context.result = AgentResponse(messages=[...blocked message...])

class PurviewChatPolicyMiddleware(ChatMiddleware):
    """Chat middleware variant for chat clients"""
```

**Settings Pattern** (Lines 40-90):

```python
class PurviewSettings(AFBaseSettings):
    app_name: str = Field(...)
    app_version: str | None = Field(default=None)
    tenant_id: str | None = Field(default=None)
    purview_app_location: PurviewAppLocation | None = Field(default=None)
    graph_base_uri: str = Field(default="https://graph.microsoft.com/v1.0/")
    blocked_prompt_message: str = Field(default="Prompt blocked by policy")
    blocked_response_message: str = Field(default="Response blocked by policy")
    ignore_exceptions: bool = Field(default=False)
    ignore_payment_required: bool = Field(default=False)
    cache_ttl_seconds: int = Field(default=14400)  # 4 hours
    max_cache_size_bytes: int = Field(default=200 * 1024 * 1024)  # 200MB
```

**Async Client Pattern** (Lines 35-135):

```python
class PurviewClient:
    async def process_content(self, request: ProcessContentRequest) -> ProcessContentResponse:
        with get_tracer().start_as_current_span("purview.process_content"):
            token = await self._get_token(tenant_id=request.tenant_id)
            url = f"{self._graph_uri}/users/{request.user_id}/dataSecurityAndGovernance/processContent"
            # POST with headers and handle response

    async def get_protection_scopes(self, request: ProtectionScopesRequest) -> ProtectionScopesResponse:
        # GET protection scopes with ETag caching

    async def send_content_activities(self, request: ContentActivitiesRequest) -> ContentActivitiesResponse:
        # POST activity logs
```

### Key Abstractions for Go

| Abstraction | Purpose |
|-------------|---------|
| `PurviewMiddleware` | Agent/Chat middleware for policy evaluation |
| `PurviewClient` | Graph API client (processContent, protectionScopes, activities) |
| `PurviewSettings` | App name, location, blocked messages |
| `ScopedContentProcessor` | Message-to-request mapping with caching |
| `CacheProvider` | ETag-based caching for protection scopes |
| `DlpAction/RestrictionAction` | Block/Allow action enums |

**Graph API Endpoints**:

| Endpoint | Purpose |
|----------|---------|
| `POST /users/{id}/dataSecurityAndGovernance/processContent` | Evaluate content against policies |
| `POST /users/{id}/dataSecurityAndGovernance/protectionScopes/compute` | Get user's protection scopes |
| `POST /users/{id}/dataSecurityAndGovernance/activities/contentActivities` | Log activities |

---

## 4. Developer UI (DevUI)

### Overview

DevUI provides an interactive web interface for testing, debugging, and visualizing AI agent execution during development.

### .NET Implementation

**Key Files**:

- [DevUIMiddleware.cs](dotnet/src/Microsoft.Agents.AI.DevUI/DevUIMiddleware.cs#L1-L263) - Static file serving
- [DevUIExtensions.cs](dotnet/src/Microsoft.Agents.AI.DevUI/DevUIExtensions.cs#L1-L69) - Endpoint mapping
- [HostApplicationBuilderExtensions.cs](dotnet/src/Microsoft.Agents.AI.DevUI/HostApplicationBuilderExtensions.cs#L1-L23) - DI registration
- [ServiceCollectionsExtensions.cs](dotnet/src/Microsoft.Agents.AI.DevUI/ServiceCollectionsExtensions.cs#L1-L58) - Service registration
- [MetaApiExtensions.cs](dotnet/src/Microsoft.Agents.AI.DevUI/MetaApiExtensions.cs#L1-L64) - Metadata API
- [README.md](dotnet/src/Microsoft.Agents.AI.DevUI/README.md#L1-L51) - Usage documentation

**Endpoint Mapping** (Lines 28-43):

```csharp
public static IEndpointConventionBuilder MapDevUI(this IEndpointRouteBuilder endpoints)
{
    var group = endpoints.MapGroup("");
    group.MapDevUI(pattern: "/devui");
    group.MapMeta();
    group.MapEntities();
    return group;
}
```

**Static File Middleware** (Lines 30-60):

```csharp
internal sealed class DevUIMiddleware
{
    private readonly FrozenDictionary<string, ResourceEntry> _resourceCache;
    
    // Serves embedded resources from assembly with:
    // - ETag-based caching (304 responses)
    // - GZip compression support
    // - Client-side routing fallback to index.html
}
```

**Meta API Response** (Lines 40-60):

```csharp
var meta = new MetaResponse
{
    UiMode = "developer",           // "developer" | "user"
    Version = "0.1.0",
    Framework = "agent_framework",
    Runtime = "dotnet",
    Capabilities = new Dictionary<string, bool>
    {
        ["tracing"] = false,
        ["openai_proxy"] = false,
        ["deployment"] = false
    },
    AuthRequired = false
};
```

**Service Registration** (Lines 20-55):

```csharp
// Keyed service factory for agents/workflows
services.AddKeyedSingleton<AIAgent>(KeyedService.AnyKey, (sp, key) =>
{
    var workflow = sp.GetKeyedService<Workflow>(keyAsStr);
    if (workflow is not null) return workflow.AsAgent(name: workflow.Name);
    
    var agent = sp.GetService<AIAgent>();
    if (agent?.Name?.Equals(keyAsStr)) return agent;
    
    return null!;
});
```

### Python Implementation

**Key Files**:

- [_server.py](python/packages/devui/agent_framework_devui/_server.py#L1-L1330) - FastAPI server
- [_discovery.py](python/packages/devui/agent_framework_devui/_discovery.py#L1-L958) - Entity discovery
- [_tracing.py](python/packages/devui/agent_framework_devui/_tracing.py#L1-L169) - OpenTelemetry tracing
- [_session.py](python/packages/devui/agent_framework_devui/_session.py#L1-L192) - Session management
- [models/](python/packages/devui/agent_framework_devui/models/) - Data models

**DevServer Class** (Lines 43-130):

```python
class DevServer:
    def __init__(
        self,
        entities_dir: str | None = None,
        port: int = 8080,
        host: str = "127.0.0.1",
        cors_origins: list[str] | None = None,
        ui_enabled: bool = True,
        mode: str = "developer",  # "developer" | "user"
    ):
        # Smart CORS defaults: permissive for localhost
        cors_origins = ["*"] if host in ("127.0.0.1", "localhost") else []
        
        self.executor: AgentFrameworkExecutor | None = None
        self.openai_executor: OpenAIExecutor | None = None
        self.deployment_manager = DeploymentManager()
        self._running_tasks: dict[str, asyncio.Task[Any]] = {}
```

**Mode-Based Error Handling** (Lines 85-110):

```python
def _format_error(self, error: Exception, context: str = "Operation") -> str:
    if self._is_dev_mode():
        return f"{context} failed: {error!s}"  # Full details
    logger.error(f"{context} failed: {error}", exc_info=True)
    return f"{context} failed"  # Generic to user

def _require_developer_mode(self, feature: str = "operation") -> None:
    if self.mode == "user":
        raise HTTPException(status_code=403, detail={"error": {...}})
```

**Entity Discovery** (Lines 20-100):

```python
class EntityDiscovery:
    def __init__(self, entities_dir: str | None = None):
        self._entities: dict[str, EntityInfo] = {}
        self._loaded_objects: dict[str, Any] = {}

    async def discover_entities(self) -> list[EntityInfo]:
        """Scan directory for agents/workflows"""

    async def load_entity(self, entity_id: str, checkpoint_manager: Any = None) -> Any:
        """Lazy load with checkpoint injection for workflows"""
```

**Trace Collection** (Lines 18-100):

```python
class SimpleTraceCollector(SpanExporter):
    """OpenTelemetry span exporter for trace events"""
    
    def export(self, spans: Sequence[Any]) -> SpanExportResult:
        for span in spans:
            trace_event = self._convert_span_to_trace_event(span)
            self.collected_events.append(trace_event)

    def _convert_span_to_trace_event(self, span) -> ResponseTraceEvent:
        return ResponseTraceEvent(
            type="trace_event",
            data={
                "span_id": str(span.context.span_id),
                "trace_id": str(span.context.trace_id),
                "operation_name": span.name,
                "duration_ms": duration_ms,
                "attributes": dict(span.attributes),
                ...
            }
        )
```

**Session Management** (Lines 17-100):

```python
class SessionManager:
    def __init__(self):
        self.sessions: dict[str, SessionData] = {}
    
    def create_session(self, session_id: str | None = None) -> str
    def get_session(self, session_id: str) -> SessionData | None
    def close_session(self, session_id: str) -> None
    def add_request_record(self, session_id, entity_id, executor_name, request_input, model_id) -> str
```

### Key Abstractions for Go

| Abstraction | Purpose |
|-------------|---------|
| `DevServer` | HTTP server with embedded UI |
| `EntityDiscovery` | Directory scanning for agents/workflows |
| `SessionManager` | Request tracking per session |
| `TraceCollector` | OpenTelemetry span collection |
| `AgentFrameworkExecutor` | Agent execution engine |
| `MetaResponse` | Server capabilities and mode |

**API Endpoints**:

| Endpoint | Purpose |
|----------|---------|
| `GET /devui/*` | Static UI files |
| `GET /meta` | Server metadata |
| `GET /entities` | List discovered entities |
| `POST /responses` | Execute agent (OpenAI-compatible) |
| `POST /conversations` | Conversation management |

---

## 5. Foundry Local

### Overview

Foundry Local provides local model inference via the Azure AI Foundry Local SDK, enabling on-device AI execution with an OpenAI-compatible API.

### Python Implementation (Python-only)

**Key Files**:

- [_foundry_local_client.py](python/packages/foundry_local/agent_framework_foundry_local/_foundry_local_client.py#L1-L260) - Chat client wrapper
- [__init__.py](python/packages/foundry_local/agent_framework_foundry_local/__init__.py#L1-L17) - Public exports

**Options TypedDict** (Lines 33-85):

```python
class FoundryLocalChatOptions(ChatOptions[TResponseModel], Generic[TResponseModel], total=False):
    """OpenAI-compatible options for local inference"""
    # Inherited: model_id, temperature, top_p, max_tokens, stop, tools, tool_choice
    
    # Foundry Local-specific
    extra_body: dict[str, Any]  # Model-specific parameters
    
    # Not applicable locally
    user: None
    store: None
```

**Settings Pattern** (Lines 100-120):

```python
class FoundryLocalSettings(AFBaseSettings):
    env_prefix: ClassVar[str] = "FOUNDRY_LOCAL_"
    model_id: str  # Required model ID or alias (e.g., "phi-4-mini")
```

**Client Class** (Lines 125-200):

```python
@use_function_invocation
@use_instrumentation
@use_chat_middleware
class FoundryLocalClient(OpenAIBaseChatClient[TFoundryLocalChatOptions]):
    def __init__(
        self,
        model_id: str | None = None,
        *,
        bootstrap: bool = True,          # Start service if not running
        timeout: float | None = None,
        prepare_model: bool = True,      # Download/load model on init
        device: DeviceType | None = None,  # GPU, CPU, NPU selection
        ...
    ):
        # Uses FoundryLocalManager from foundry_local package
        self._manager = FoundryLocalManager(...)
        if bootstrap:
            self._manager.start()
        if prepare_model:
            self._manager.prepare_model(model_id, device)
        
        # OpenAI-compatible client pointing to local endpoint
        self._client = AsyncOpenAI(base_url=self._manager.endpoint, api_key="local")
```

**Key Features**:

- Extends `OpenAIBaseChatClient` for OpenAI API compatibility
- Uses `FoundryLocalManager` for service lifecycle
- Supports device selection (GPU, CPU, NPU)
- Model catalog querying via `manager.list_catalog_models()`
- Decorators for instrumentation and middleware

### Key Abstractions for Go

| Abstraction | Purpose |
|-------------|---------|
| `FoundryLocalClient` | OpenAI-compatible local client |
| `FoundryLocalManager` | Service lifecycle management |
| `DeviceType` | GPU/CPU/NPU selection |
| Model catalog | Available model listing |

---

## 6. Go Implementation Recommendations

### Priority Assessment

| Integration | Priority | Complexity | Notes |
|-------------|----------|------------|-------|
| DevUI | **High** | Medium | Essential for development experience |
| Purview | **High** | High | Enterprise compliance requirement |
| Copilot Studio | Medium | Medium | External SDK dependency |
| GitHub Copilot | Low | High | External SDK dependency, CLI-based |
| Foundry Local | Low | Low | Python-only currently |

### Recommended Package Structure

```
go/
├── enterprise/
│   ├── devui/
│   │   ├── server.go           # HTTP server with embedded UI
│   │   ├── discovery.go        # Entity discovery
│   │   ├── session.go          # Session management
│   │   ├── tracing.go          # OpenTelemetry integration
│   │   └── meta.go             # Metadata API
│   │
│   ├── purview/
│   │   ├── client.go           # Graph API client
│   │   ├── middleware.go       # Agent/Chat middleware
│   │   ├── settings.go         # Configuration
│   │   ├── processor.go        # Content processing
│   │   ├── cache.go            # ETag caching
│   │   └── models.go           # Request/Response types
│   │
│   ├── copilotstudio/
│   │   ├── agent.go            # CopilotStudioAgent
│   │   ├── session.go          # Session with ConversationId
│   │   ├── activity.go         # Activity processor
│   │   └── settings.go         # Connection settings
│   │
│   └── ghcopilot/
│       ├── agent.go            # GitHubCopilotAgent
│       ├── session.go          # Session management
│       ├── permissions.go      # Permission handling
│       └── settings.go         # CLI/model configuration
```

### Common Patterns to Implement

#### 1. Settings Pattern

```go
type Settings struct {
    EnvPrefix string
}

func LoadSettings[T any](prefix string) (*T, error) {
    // Load from environment variables with prefix
}
```

#### 2. Middleware Pattern

```go
type Middleware interface {
    Process(ctx context.Context, messages []ChatMessage, next NextFunc) (*AgentResponse, error)
}

type NextFunc func(context.Context, []ChatMessage) (*AgentResponse, error)
```

#### 3. Session Serialization

```go
type Session interface {
    Serialize() ([]byte, error)
    Deserialize([]byte) error
}
```

#### 4. Builder Extensions

```go
func (b *AgentBuilder) WithPurview(cred TokenCredential, settings PurviewSettings) *AgentBuilder {
    return b.Use(func(inner Agent) Agent {
        return NewPurviewAgent(inner, cred, settings)
    })
}
```

### Implementation Notes

1. **DevUI**: Start with embedded static files using `embed.FS`, implement OpenAI-compatible endpoints
2. **Purview**: Focus on Graph API client first, then middleware wrapper
3. **External SDKs**: Wait for Go SDKs or consider gRPC/REST bridges
4. **Tracing**: Leverage existing OpenTelemetry infrastructure

---

## Summary

| Integration | .NET Files | Python Files | Key Pattern |
|-------------|------------|--------------|-------------|
| Copilot Studio | 3 | 3 | Agent wrapper + Activity processing |
| GitHub Copilot | 4 | 3 | Agent wrapper + Session + Permissions |
| Purview | 15+ | 8 | Middleware + Graph API + Caching |
| DevUI | 7 | 10+ | HTTP server + Discovery + Tracing |
| Foundry Local | N/A | 2 | OpenAI-compatible local client |

All integrations follow consistent patterns: Settings classes, session management, middleware composition, and builder extensions. The Go implementation should prioritize DevUI and Purview for immediate enterprise value.
