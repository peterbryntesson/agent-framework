<!-- markdownlint-disable-file -->
# Task Research: Go Epic 5 - Enterprise Production Features

Comprehensive research for implementing Epic 5 of the Go Agent Framework port, covering enterprise production features including durable agents, declarative definitions, HTTP/gRPC hosting, MCP integration, protocol-specific hosting, and production hardening.

## Task Implementation Requests

* Research .NET DurableTask implementation for durable agents pattern
* Research Python durabletask implementation for comparison
* Research .NET Declarative agents YAML loading and factory patterns
* Research Python declarative agents implementation
* Research .NET HTTP/gRPC hosting patterns (A2A, AG-UI, OpenAI-compatible)
* Research MCP integration patterns from .NET and Python
* Research Protocol-Specific hosting implementations
* Research production hardening patterns (rate limiting, circuit breaker, retry)

## Scope and Success Criteria

* Scope: All 7 features of Epic 5 with detailed analysis of .NET and Python implementations
* Assumptions:
  * Go implementation should follow existing Go patterns in the codebase
  * Feature parity with .NET and Python is the goal
  * Temporal.io is the target for durable agents in Go (unlike DurableTask in .NET)
* Success Criteria:
  * Complete mapping of .NET/Python APIs to Go equivalents
  * Identified Go-specific adaptations needed
  * Implementation patterns documented with code examples
  * All 7 features have actionable implementation guidance

## Outline

1. Feature 5.1: Durable Agents - DurableTask vs Temporal.io
2. Feature 5.2: Declarative Agent Definitions - YAML loading patterns
3. Feature 5.3: HTTP and gRPC Hosting - Server infrastructure
4. Feature 5.4: MCP Integration - Client/Server patterns
5. Feature 5.5: Enterprise Integrations - Copilot Studio, GitHub, Purview
6. Feature 5.6: Protocol-Specific Hosting - A2A, AG-UI, OpenAI-compatible
7. Feature 5.7: Production Hardening - Resilience patterns

### Potential Next Research

* Temporal.io Go SDK specific patterns for agent state management
* MCP Go SDK availability (github.com/mark3labs/mcp-go)
* OpenAPI code generation for gRPC proto definitions

## Research Executed

### File Analysis

**DurableTask Packages:**
* `dotnet/src/Microsoft.Agents.AI.DurableTask/` - 14 files including DurableAIAgent, AgentEntity, State/
* `python/packages/durabletask/agent_framework_durabletask/` - 11 files with platform-agnostic patterns
* `python/packages/azurefunctions/` - Azure Functions specific integration

**Declarative Packages:**
* `dotnet/src/Microsoft.Agents.AI.Declarative/` - YAML parsing, factory patterns
* `dotnet/src/Microsoft.Agents.AI.Workflows.Declarative/` - Workflow orchestration
* `python/packages/declarative/agent_framework_declarative/` - _models.py (1109 lines), _loader.py (850 lines)

**Hosting Packages:**
* `dotnet/src/Microsoft.Agents.AI.Hosting/` - Core hosting with session stores
* `dotnet/src/Microsoft.Agents.AI.Hosting.A2A.AspNetCore/` - A2A protocol
* `dotnet/src/Microsoft.Agents.AI.Hosting.AGUI.AspNetCore/` - AG-UI SSE streaming
* `dotnet/src/Microsoft.Agents.AI.Hosting.OpenAI/` - OpenAI-compatible endpoints
* `dotnet/src/Microsoft.Agents.AI.Hosting.AzureFunctions/` - Azure Functions triggers

**Existing Go Protocol Packages:**
* `go/protocol/a2a/` - A2A server (server.go: 409 lines), client, types
* `go/protocol/agui/` - AG-UI server (server.go: 276 lines), client, events
* `go/hosting/` - SessionStore interface (170 lines)

### Code Search Results

**Durable State Schema:**
* Schema version: `1.1.0`
* Defined in: `schemas/durable-agent-entity-state.json`
* Content types: text, data, error, functionCall, functionResult, hostedFile, reasoning, usage

**YAML Sample Files:**
* `agent-samples/azure/` - Azure OpenAI, Assistants
* `agent-samples/foundry/` - AI Foundry agents
* `agent-samples/chatclient/` - Function tools
* `workflow-samples/` - Multi-agent workflows

### External Research

**Temporal.io Go SDK:**
* Package: `go.temporal.io/sdk`
* Key types: `workflow.Context`, `client.Client`, `activity.Activity`
* Workflow-per-session pattern recommended over entities

**Resilience Libraries:**
* Rate limiting: `golang.org/x/time/rate` (token bucket)
* Circuit breaker: `github.com/sony/gobreaker`
* Retry: `github.com/cenkalti/backoff/v5` (already indirect dependency)

### Project Conventions

* Standards referenced: Go idioms from existing codebase (`go/agent/`, `go/chat/`, `go/protocol/`)
* Patterns: Functional options, interface-based DI, context propagation
* Instructions followed: `.github/copilot-instructions.md` for C# parity guidelines

## Key Discoveries

### 1. Durable Agents Architecture (Feature 5.1)

**Cross-Platform State Schema:**

Both .NET and Python use identical JSON state schema (v1.1.0) with:
- `schemaVersion`: Forward compatibility marker
- `data.conversationHistory`: Array of request/response entries
- `data.expirationTimeUtc`: TTL for automatic cleanup
- Polymorphic `$type` discriminators: "request", "response"

| .NET Component | Python Equivalent | Temporal.io Go Mapping |
|----------------|-------------------|----------------------|
| `DurableAIAgent` | `DurableAIAgent` shim | Workflow activity wrapper |
| `AgentEntity` | `AgentEntity` + `StateProviderMixin` | Workflow with signals/updates |
| `DurableAgentSession` | `DurableAgentThread` | Workflow ID pattern |
| `AgentSessionId` | `AgentSessionId` | `{agentName}-{key}` workflow ID |
| Entity operations | Entity operations | Signal channels / Update handlers |
| `SignalEntity` with `SignalTime` | Scheduled signals | `workflow.Sleep` + continue-as-new |

**Go Implementation Approach:**

Use **Workflow-per-Session** pattern with Temporal.io:
```go
// Workflow definition
func AgentSessionWorkflow(ctx workflow.Context, sessionID AgentSessionId) error {
    state := NewDurableAgentState()
    
    // Register update handler for synchronous request/response
    workflow.SetUpdateHandler(ctx, "run", func(ctx workflow.Context, request RunRequest) (AgentResponse, error) {
        // 1. Append request to conversation history
        state.Data.ConversationHistory = append(state.Data.ConversationHistory, 
            DurableAgentStateRequest{...})
        
        // 2. Execute agent activity
        var response AgentResponse
        err := workflow.ExecuteActivity(ctx, RunAgentActivity, RunAgentInput{
            Messages: state.BuildChatMessages(),
            Options:  request.Options,
        }).Get(ctx, &response)
        
        // 3. Append response to history
        state.Data.ConversationHistory = append(state.Data.ConversationHistory,
            DurableAgentStateResponse{...})
        
        return response, err
    })
    
    // TTL expiration via sleep
    if state.Data.ExpirationTimeUtc != nil {
        selector := workflow.NewSelector(ctx)
        selector.AddFuture(workflow.NewTimer(ctx, time.Until(*state.Data.ExpirationTimeUtc)), 
            func(f workflow.Future) { /* cleanup */ })
        selector.Select(ctx)
    }
    
    return nil
}
```

**File Structure for Go:**
```
go/durable/
├── agent.go           # DurableAIAgent wrapper
├── session.go         # DurableAgentSession with session ID
├── sessionid.go       # AgentSessionId type
├── state.go           # DurableAgentState + entries
├── state_entry.go     # Request/Response entry types
├── worker.go          # Temporal worker setup
├── client.go          # DurableAgentClient for external callers
└── options.go         # Functional options
```

### 2. Declarative Agents Architecture (Feature 5.2)

**YAML Schema Comparison:**

| Field | .NET | Python | Go Recommendation |
|-------|------|--------|-------------------|
| `kind` | Required ("Prompt") | Required ("Prompt", "Agent") | Required ("Prompt") |
| `name` | Required | Required | Required |
| `instructions` | Required | Optional | Required |
| `model` | Required | Optional | Required |
| `model.provider` | "AzureOpenAI", "OpenAI" | Multiple providers | Same as .NET |
| `model.apiType` | "Chat", "Assistants", "Responses" | Same | Same |
| `model.connection` | ApiKey, Remote, Anonymous | key, remote, anonymous | Same |
| `tools[].kind` | function, mcp, webSearch, fileSearch, codeInterpreter | Same + openapi | Same as .NET |
| PowerFx expressions | `=Env.VAR_NAME` | `=Env.VAR_NAME` | Environment substitution |

**Factory Pattern:**

```go
// Go declarative package structure
go/declarative/
├── loader.go          # LoadFromFile, LoadFromReader, LoadFromString
├── models.go          # PromptAgent, Model, Tool, Connection structs
├── factory.go         # AgentFactory with provider registry
├── providers.go       # ProviderBuilder registry
├── tools.go           # Tool parsing and conversion
├── powerfx.go         # Expression evaluation (env vars)
└── validation.go      # Schema validation

// Factory interface
type AgentFactory struct {
    chatClient chat.Client              // Optional pre-configured client
    bindings   map[string]tool.Func     // Tool function bindings
    providers  map[string]ProviderBuilder
}

func (f *AgentFactory) CreateFromYAML(content string) (agent.Agent, error)
func (f *AgentFactory) CreateFromFile(path string) (agent.Agent, error)
func (f *AgentFactory) RegisterProvider(key string, builder ProviderBuilder)
```

**Tool Kind Mapping:**

| Kind | Go Implementation |
|------|-------------------|
| `function` | Look up from `bindings` map, create `tool.Func` |
| `mcp` | Create `HostedMCPTool` for service execution |
| `webSearch` | Create `HostedWebSearchTool` |
| `fileSearch` | Create `HostedFileSearchTool` with vector store IDs |
| `codeInterpreter` | Create `HostedCodeInterpreterTool` |

### 3. HTTP and gRPC Hosting (Feature 5.3)

**Existing Go Infrastructure:**

The Go codebase already has:
* `go/hosting/sessionstore.go` - SessionStore interface + InMemorySessionStore
* `go/protocol/a2a/server.go` - Full A2A server with task management
* `go/protocol/agui/server.go` - Full AG-UI server with SSE streaming

**Gaps to Fill:**

| Feature | .NET | Go Status | Priority |
|---------|------|-----------|----------|
| Core hosting builder | `IHostedAgentBuilder` | ❌ Missing | High |
| Session persistence | `AgentSessionStore` | ✅ Exists (`hosting.SessionStore`) | Done |
| HTTP handler base | `NewHandler()` | Partial (per-protocol) | Medium |
| gRPC server | proto + server | ❌ Missing | Medium |
| Health checks | `/health` | ❌ Missing | Medium |
| CORS | Middleware | ❌ Missing | Low |
| Graceful shutdown | Context-based | Partial | Medium |

**Recommended Go Structure:**

```
go/hosting/
├── doc.go             # Package documentation
├── sessionstore.go    # ✅ Already exists
├── builder.go         # NEW: HostedAgentBuilder
├── handler.go         # NEW: Generic HTTP handler factory
├── options.go         # NEW: Functional options
├── grpc/
│   ├── proto/
│   │   └── agent.proto    # Service definition
│   ├── server.go          # gRPC server implementation
│   └── gen/               # Generated code
└── middleware/
    ├── cors.go            # CORS middleware
    ├── logging.go         # Request logging
    └── recovery.go        # Panic recovery
```

**Builder Pattern for Go:**

```go
type HostedAgentBuilder struct {
    name         string
    agent        agent.Agent
    sessionStore SessionStore
    middleware   []Middleware
}

func NewHostedAgentBuilder(name string) *HostedAgentBuilder
func (b *HostedAgentBuilder) WithAgent(a agent.Agent) *HostedAgentBuilder
func (b *HostedAgentBuilder) WithSessionStore(s SessionStore) *HostedAgentBuilder
func (b *HostedAgentBuilder) Build() *HostedAgent

// HostedAgent wraps agent with session management
type HostedAgent struct {
    agent.Agent
    sessionStore SessionStore
}

func (h *HostedAgent) GetOrCreateSession(ctx context.Context, convID string) (agent.Session, error)
func (h *HostedAgent) SaveSession(ctx context.Context, convID string, session agent.Session) error
```

### 4. MCP Integration (Feature 5.4)

**Python vs .NET Comparison:**

| Feature | Python | .NET | Go Recommendation |
|---------|--------|------|-------------------|
| MCP Client | Native `MCPTool` class (1245 lines) | `ModelContextProtocol` NuGet | Use external Go MCP library |
| Transport: Stdio | `MCPStdioTool` | `StdioClientTransport` | Wrap exec.Command |
| Transport: HTTP/SSE | `MCPStreamableHTTPTool` | `HttpClientTransport` | HTTP client + SSE parser |
| Transport: WebSocket | `MCPWebsocketTool` | Available | gorilla/websocket |
| Tool Discovery | `load_tools()` with pagination | `ListToolsAsync()` | Same pattern |
| Tool Bridging | `FunctionTool` wrapper | `McpClientTool : AITool` | Implement `tool.Tool` |
| Agent as MCP Server | `as_mcp_server()` method | Sample-only | Implement method |
| Hosted MCP Tool | `HostedMCPTool` | `HostedMcpServerTool` | For Responses API |

**Go MCP Package Structure:**

```
go/mcp/
├── doc.go             # Package documentation
├── client.go          # MCPClient interface
├── transport.go       # Transport interface
├── transport_stdio.go # Stdio transport
├── transport_http.go  # HTTP/SSE transport
├── tool.go            # MCPTool implementing tool.Tool
├── server.go          # Agent as MCP server
├── types.go           # MCP types (Tool, Resource, etc.)
└── hosted.go          # HostedMCPTool for service execution
```

**Client Pattern:**

```go
// MCPClient wraps connection to an MCP server
type MCPClient struct {
    transport  Transport
    session    *ClientSession
    tools      []FunctionTool
    mu         sync.RWMutex
}

type ClientOption func(*clientOptions)

func NewClient(transport Transport, opts ...ClientOption) (*MCPClient, error)
func (c *MCPClient) Connect(ctx context.Context) error
func (c *MCPClient) ListTools(ctx context.Context) ([]ToolInfo, error)
func (c *MCPClient) CallTool(ctx context.Context, name string, args map[string]any) (Result, error)
func (c *MCPClient) Tools() []tool.Tool  // Returns bridge tools

// Transport implementations
func NewStdioTransport(command string, args ...string) Transport
func NewHTTPTransport(endpoint string, opts ...HTTPTransportOption) Transport
```

**Server Pattern (Agent as MCP Server):**

```go
// Extension method on Agent
func AsMCPServer(a agent.Agent, opts ...ServerOption) *MCPServer

type MCPServer struct {
    agent    agent.Agent
    tools    []MCPToolDefinition
    resources []MCPResource
}

func (s *MCPServer) ListTools() []types.Tool
func (s *MCPServer) CallTool(ctx context.Context, name string, args map[string]any) ([]Content, error)
func (s *MCPServer) ServeHTTP(w http.ResponseWriter, r *http.Request)  // JSON-RPC 2.0
```

### 5. Enterprise Integrations (Feature 5.5)

**Priority Ranking for Go:**

| Integration | Priority | Rationale | Complexity |
|-------------|----------|-----------|------------|
| DevUI | High | Developer experience critical | Medium |
| Purview | High | Enterprise compliance | Medium-High |
| Copilot Studio | Medium | Requires external SDK | Medium |
| GitHub Copilot | Low | CLI-based, complex permissions | High |
| Foundry Local | Low | Python-only, local execution | Low |

**DevUI Architecture:**

```
go/devui/
├── doc.go
├── server.go          # HTTP server with embedded UI
├── discovery.go       # Agent/executor discovery
├── tracing.go         # OpenTelemetry trace collector
├── handlers.go        # API handlers
├── frontend/          # Embedded static assets (from Python)
└── options.go
```

Key endpoints:
- `GET /meta` - List discovered agents
- `POST /run` - Run agent (OpenAI-compatible)
- `POST /stream` - Stream agent response
- `GET /traces` - Get collected traces

**Purview Middleware Pattern:**

```go
// Purview middleware wraps agent with policy evaluation
type PurviewMiddleware struct {
    client   *PurviewClient
    settings PurviewSettings
}

func WithPurview(settings PurviewSettings) agent.Middleware

type PurviewClient struct {
    httpClient *http.Client
    credential azidentity.TokenCredential
    graphURI   string
    cache      *scopeCache  // ETag-based caching
}

func (c *PurviewClient) ProcessContent(ctx context.Context, req ProcessContentRequest) (*ProcessContentResponse, error)
```

### 6. Protocol-Specific Hosting (Feature 5.6)

**Current Go Status:**

| Protocol | Status | Location |
|----------|--------|----------|
| A2A | ✅ Complete | `go/protocol/a2a/server.go` |
| AG-UI | ✅ Complete | `go/protocol/agui/server.go` |
| OpenAI-compatible | ❌ Missing | Need to implement |
| Azure Functions | ❌ Missing | Out of scope for Go |

**OpenAI-Compatible Endpoints to Implement:**

```
go/hosting/openai/
├── doc.go
├── handler.go         # Route handler factory
├── completions.go     # POST /v1/chat/completions
├── responses.go       # POST /v1/responses (streaming)
├── conversations.go   # CRUD /v1/conversations
├── models.go          # Request/Response types
├── streaming.go       # SSE formatting
└── options.go
```

**Handler Pattern:**

```go
// OpenAI-compatible handler for an agent
func NewHandler(a agent.Agent, opts ...Option) http.Handler {
    h := &handler{agent: a, opts: applyOptions(opts)}
    mux := http.NewServeMux()
    mux.HandleFunc("POST /v1/chat/completions", h.handleChatCompletions)
    mux.HandleFunc("POST /v1/responses", h.handleCreateResponse)
    mux.HandleFunc("GET /v1/responses/{id}", h.handleGetResponse)
    // ... more endpoints
    return mux
}

// SSE streaming response
func (h *handler) streamCompletions(w http.ResponseWriter, messages []chat.Message) error {
    w.Header().Set("Content-Type", "text/event-stream")
    w.Header().Set("Cache-Control", "no-cache")
    flusher := w.(http.Flusher)
    
    stream := h.agent.RunStreaming(ctx, messages)
    for update := range stream.Updates() {
        data, _ := json.Marshal(toChatCompletionChunk(update))
        fmt.Fprintf(w, "data: %s\n\n", data)
        flusher.Flush()
    }
    fmt.Fprintf(w, "data: [DONE]\n\n")
    return nil
}
```

### 7. Production Hardening (Feature 5.7)

**Current State Analysis:**

| Pattern | .NET | Python | Go | Implementation Needed |
|---------|------|--------|-----|----------------------|
| Rate limit detection | HTTP 429 | HTTP 429 | `ErrRateLimited` | ✅ Already exists |
| IsRetryable helper | ❌ | ❌ | `IsRetryable()` | ✅ Already exists |
| Rate limiter | ❌ | ❌ | ❌ | New middleware |
| Circuit breaker | Polly-based | ❌ | ❌ | New middleware |
| Retry middleware | Polly-based | Linear backoff | ❌ | New middleware |
| Connection pooling | HttpClient | httpx | http.Transport | Configuration helper |

**Go Resilience Package Structure:**

```
go/resilience/
├── doc.go
├── ratelimit.go       # Token bucket rate limiter
├── circuitbreaker.go  # Circuit breaker middleware
├── retry.go           # Retry policy with backoff
├── pool.go            # HTTP client pool configuration
└── options.go         # Common configuration types
```

**Rate Limiter Implementation:**

```go
import "golang.org/x/time/rate"

type RateLimiter struct {
    limiter *rate.Limiter
    perClient map[string]*rate.Limiter
    mu        sync.RWMutex
}

func NewRateLimiter(rps float64, burst int) *RateLimiter
func (r *RateLimiter) Wait(ctx context.Context) error
func (r *RateLimiter) WaitForClient(ctx context.Context, clientID string) error

// Middleware wrapper
func WithRateLimit(rps float64, burst int) chat.Middleware
```

**Circuit Breaker Implementation:**

```go
import "github.com/sony/gobreaker"

type CircuitBreaker struct {
    cb *gobreaker.CircuitBreaker
}

type CircuitBreakerOption func(*gobreaker.Settings)

func NewCircuitBreaker(opts ...CircuitBreakerOption) *CircuitBreaker
func WithMaxRequests(n uint32) CircuitBreakerOption
func WithTimeout(d time.Duration) CircuitBreakerOption
func WithReadyToTrip(f func(gobreaker.Counts) bool) CircuitBreakerOption

func (cb *CircuitBreaker) Execute(fn func() error) error

// Middleware wrapper
func WithCircuitBreaker(opts ...CircuitBreakerOption) chat.Middleware
```

**Retry Policy Implementation:**

```go
import "github.com/cenkalti/backoff/v5"

type RetryPolicy struct {
    maxAttempts int
    backoff     backoff.BackOff
}

type RetryOption func(*RetryPolicy)

func NewRetryPolicy(opts ...RetryOption) *RetryPolicy
func WithMaxAttempts(n int) RetryOption
func WithExponentialBackoff(initial, max time.Duration) RetryOption
func WithJitter(factor float64) RetryOption

func (p *RetryPolicy) Execute(ctx context.Context, fn func() error) error

// Middleware wrapper
func WithRetry(opts ...RetryOption) chat.Middleware
```

## Technical Scenarios

### Scenario 1: Implementing DurableAIAgent with Temporal.io

**Requirements:**
* Persist conversation state across process restarts
* Support human-in-the-loop wait states
* TTL-based session cleanup
* Cross-language state compatibility

**Preferred Approach:**
Use Workflow-per-Session with Update handlers (not entities, which are experimental in Temporal.io).

```
go/durable/
├── agent.go           # DurableAIAgent implementing agent.Agent
├── session.go         # DurableAgentSession with Temporal workflow handle
├── sessionid.go       # AgentSessionId (agentName + key)
├── state.go           # DurableAgentState matching JSON schema 1.1.0
├── worker.go          # NewTemporalWorker with activity registration
├── workflow.go        # AgentSessionWorkflow definition
├── activity.go        # RunAgentActivity executing real agent
└── client.go          # DurableAgentClient for external callers
```

**Key Implementation Details:**

1. **Workflow ID pattern**: `durable-agent-{agentName}-{sessionKey}`
2. **State schema**: Match .NET/Python `schemaVersion: "1.1.0"` for cross-platform compatibility
3. **Update handler**: Use `workflow.SetUpdateHandler` for synchronous run requests
4. **TTL handling**: `workflow.Sleep` until expiration, then signal cleanup
5. **Continue-as-new**: Handle long conversation histories by periodic workflow continuation

### Scenario 2: Implementing Declarative Agent Factory

**Requirements:**
* Parse YAML agent definitions
* Support multiple providers (AzureOpenAI, OpenAI, Anthropic)
* Tool binding to Go functions
* Environment variable substitution

**Preferred Approach:**
Factory pattern with provider registry and YAML unmarshaling via `gopkg.in/yaml.v3`.

```
go/declarative/
├── loader.go          # LoadFromFile, LoadFromReader, LoadFromString
├── models.go          # PromptAgent, Model, Tool, Connection, PropertySchema
├── factory.go         # AgentFactory with Create methods
├── providers.go       # ProviderBuilder registry (map[string]ProviderBuilder)
├── tools.go           # parseTool dispatching by Kind
├── eval.go            # tryPowerFxEval for =Env.VAR_NAME substitution
└── validation.go      # Validate against schema requirements
```

**Provider Registration:**

```go
var defaultProviders = map[string]ProviderBuilder{
    "AzureOpenAI.Chat":       newAzureOpenAIChatProvider,
    "AzureOpenAI.Responses":  newAzureOpenAIResponsesProvider,
    "OpenAI.Chat":            newOpenAIChatProvider,
    "Anthropic.Chat":         newAnthropicChatProvider,
}
```

### Scenario 3: Implementing OpenAI-Compatible Hosting

**Requirements:**
* `/v1/chat/completions` endpoint
* SSE streaming support
* Request/response format matching OpenAI API
* Integration with existing SessionStore

**Preferred Approach:**
Handler factory returning `http.Handler` with standard library patterns.

```go
// Usage
mux := http.NewServeMux()
mux.Handle("/api/agent/", hosting.NewOpenAIHandler(myAgent,
    hosting.WithSessionStore(sessionStore),
    hosting.WithStreamingEnabled(true),
))
http.ListenAndServe(":8080", mux)
```

**File Structure:**
```
go/hosting/openai/
├── handler.go         # NewHandler factory
├── completions.go     # Chat completions endpoint
├── responses.go       # Responses API endpoints
├── streaming.go       # SSE writer utilities
├── models.go          # OpenAI request/response types
└── options.go         # Handler options
```

## Implementation Priority Matrix

| Feature | Priority | Dependencies | Effort | Notes |
|---------|----------|--------------|--------|-------|
| 5.7 Production Hardening | P0 | None | Medium | Foundation for all features |
| 5.3 HTTP Hosting (OpenAI-compat) | P0 | Existing SessionStore | Medium | High value, builds on A2A/AGUI |
| 5.2 Declarative Agents | P1 | Provider packages | High | Enables no-code agent creation |
| 5.4 MCP Integration | P1 | External MCP library | High | Critical for tool ecosystem |
| 5.1 Durable Agents | P2 | Temporal.io SDK | High | Complex, needs Temporal infrastructure |
| 5.5 Enterprise: DevUI | P2 | OpenAI hosting | Medium | Developer experience |
| 5.5 Enterprise: Purview | P2 | Azure SDK | Medium | Enterprise compliance |
| 5.6 Azure Functions | P3 | N/A | N/A | Out of scope for Go |

## Dependencies

**New Go Modules Required:**

```go
// go.mod additions
require (
    go.temporal.io/sdk v1.29.1                      // Durable agents
    gopkg.in/yaml.v3 v3.0.1                         // Declarative YAML
    golang.org/x/time v0.5.0                        // Rate limiting
    github.com/sony/gobreaker v1.0.0                // Circuit breaker
    github.com/mark3labs/mcp-go v0.8.0              // MCP client/server
    google.golang.org/grpc v1.64.0                  // gRPC hosting
    google.golang.org/protobuf v1.34.2              // Proto definitions
)
```

**Existing Dependencies to Leverage:**

* `github.com/cenkalti/backoff/v5` - Already indirect, use for retry
* `github.com/google/uuid` - Session ID generation
* `go.opentelemetry.io/otel` - Tracing for DevUI

## Success Criteria

* 100% of Epic 5 features implemented with API parity
* Cross-platform state schema compatibility for durable agents
* 90%+ test coverage across all new packages
* All public APIs documented with godoc
* Integration tests for Temporal.io, MCP, and hosting endpoints
* Benchmark overhead < 10ms for simple synchronous runs
* Benchmark streaming overhead < 1ms per chunk
