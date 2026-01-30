# Microsoft Agent Framework - Go (Golang) Port Plan

## Executive Summary

This document outlines a comprehensive plan for porting the Microsoft Agent Framework SDK from its current C# and Python implementations to Go (Golang). The goal is to create a production-ready, idiomatic Go implementation that maintains feature parity while leveraging Go's strengths in performance, concurrency, and simplicity.

---

## 1. Codebase Analysis Summary

### 1.1 Current Architecture Overview

The Microsoft Agent Framework is organized into several key layers:

| Layer | C# Namespace | Python Module | Purpose |
|-------|-------------|---------------|---------|
| **Core Abstractions** | `Microsoft.Agents.AI.Abstractions` | `agent_framework._agents`, `_types` | Base agent interfaces, responses, sessions |
| **Agent Implementation** | `Microsoft.Agents.AI` | `agent_framework._clients` | ChatClientAgent, middleware, tools |
| **Providers** | `Microsoft.Agents.AI.OpenAI`, `.AzureAI`, `.Anthropic` | `agent_framework.openai`, `.azure`, `.anthropic` | LLM provider integrations |
| **Workflows** | `Microsoft.Agents.AI.Workflows` | `agent_framework._workflows` | Graph-based orchestration |
| **Protocols** | `Microsoft.Agents.AI.A2A`, `.AGUI` | `agent_framework.a2a`, `.ag_ui` | Inter-agent communication |
| **Hosting** | `Microsoft.Agents.AI.Hosting` | N/A | ASP.NET Core integration |
| **Durable Agents** | `Microsoft.Agents.AI.DurableTask` | `agent_framework.durabletask` | Long-running orchestration |
| **Declarative** | `Microsoft.Agents.AI.Declarative` | `agent_framework.declarative` | YAML-based agent definitions |
| **Observability** | Integrated in core | `agent_framework.observability` | OpenTelemetry integration |

### 1.2 Key Abstractions Identified

```
┌─────────────────────────────────────────────────────────────────────┐
│                          AIAgent (Base)                              │
│  - Id, Name, Description                                            │
│  - RunAsync(messages, session, options) → AgentResponse             │
│  - RunStreamingAsync(messages, session, options) → stream<Update>   │
│  - GetNewSessionAsync() → AgentSession                              │
└─────────────────────────────────────────────────────────────────────┘
                                    │
                    ┌───────────────┼───────────────┐
                    ▼               ▼               ▼
            ChatClientAgent    DurableAIAgent    A2AAgent
                    │
                    ▼
            ┌───────────────────────────────────────┐
            │         IChatClient (from MEAI)       │
            │  - GetResponse(messages) → Response   │
            │  - GetStreamingResponse() → stream    │
            └───────────────────────────────────────┘
                    │
        ┌───────────┼───────────┬───────────────┐
        ▼           ▼           ▼               ▼
    OpenAI      AzureOpenAI   Anthropic     Ollama
```

### 1.3 Feature Matrix

| Feature | C# | Python | Go (Planned) |
|---------|:--:|:------:|:------------:|
| Core Agent Abstraction | ✓ | ✓ | Phase 1 |
| Chat Client Protocol | ✓ | ✓ | Phase 1 |
| Agent Response/Streaming | ✓ | ✓ | Phase 1 |
| Agent Session/Thread | ✓ | ✓ | Phase 1 |
| OpenAI Provider | ✓ | ✓ | Phase 2 |
| Azure OpenAI Provider | ✓ | ✓ | Phase 2 |
| Anthropic Provider | ✓ | ✓ | Phase 2 |
| Ollama Provider | ✗ | ✓ | Phase 2 |
| Function/Tool Calling | ✓ | ✓ | Phase 2 |
| Middleware Pipeline | ✓ | ✓ | Phase 3 |
| Context Providers | ✓ | ✓ | Phase 3 |
| Memory/RAG | ✓ | ✓ | Phase 3 |
| Workflows (DAG) | ✓ | ✓ | Phase 4 |
| A2A Protocol | ✓ | ✓ | Phase 4 |
| AG-UI Protocol | ✓ | ✓ | Phase 4 |
| OpenTelemetry | ✓ | ✓ | Phase 2 |
| Durable Agents | ✓ | ✓ | Phase 5 |
| Declarative YAML | ✓ | ✓ | Phase 5 |
| MCP Integration | ✓ | ✓ | Phase 5 |
| **Explicitly Excluded** | | | |
| Lab/Experimental | ✗ | ✓ | Out of Scope* |
| Workflow Generators | ✓ | ✗ | Out of Scope* |
| Legacy Support | ✓ | ✗ | Not Applicable |

*\*Scope Exclusion Notes:*
- *Lab/Experimental*: Python `lab/` package (benchmarks, RL training, experimental features) is excluded from initial Go port. May be considered for future phases based on community demand.
- *Workflow Generators*: C# `Workflows.Generators` uses Roslyn for code generation. Go will use `go generate` patterns if needed; not a direct port target.
- *Legacy Support*: C# `LegacySupport/` packages are not applicable to Go. The Go implementation will target minimum Go 1.22+ from inception.

---

## 2. Go Package Structure

### 2.1 Proposed Module Layout

```
github.com/microsoft/agent-framework-go/
├── go.mod
├── go.sum
├── README.md
├── LICENSE
│
├── agent/                          # Core agent abstractions
│   ├── agent.go                    # AIAgent interface
│   ├── response.go                 # AgentResponse, AgentResponseUpdate
│   ├── session.go                  # AgentSession interface
│   ├── options.go                  # AgentRunOptions
│   ├── metadata.go                 # AIAgentMetadata
│   └── errors.go                   # Agent-specific errors
│
├── chat/                           # Chat client abstractions
│   ├── client.go                   # ChatClient interface
│   ├── message.go                  # ChatMessage, Content, Role
│   ├── response.go                 # ChatResponse, ChatResponseUpdate
│   ├── options.go                  # ChatOptions
│   ├── tool.go                     # Tool interface and types
│   └── usage.go                    # UsageDetails
│
├── chatagent/                      # ChatClient-backed agent
│   ├── agent.go                    # ChatClientAgent implementation
│   ├── options.go                  # ChatClientAgentOptions
│   ├── session.go                  # ChatClientAgentSession
│   └── builder.go                  # AgentBuilder pattern
│
├── middleware/                     # Middleware pipeline
│   ├── middleware.go               # Middleware interfaces
│   ├── agent.go                    # AgentMiddleware
│   ├── chat.go                     # ChatMiddleware
│   ├── function.go                 # FunctionMiddleware
│   ├── context.go                  # Context types
│   └── pipeline.go                 # Pipeline builder
│
├── tool/                           # Tool system
│   ├── tool.go                     # Tool interface
│   ├── function.go                 # FunctionTool
│   ├── hosted.go                   # Hosted tools (web search, etc.)
│   ├── invoke.go                   # Function invocation
│   └── decorator.go                # Tool decorators
│
├── memory/                         # Context and memory providers
│   ├── context.go                  # Context, ContextProvider
│   ├── history.go                  # ChatHistoryProvider
│   ├── store.go                    # Message store interfaces
│   ├── mem0/
│   │   ├── client.go               # Mem0 memory provider
│   │   └── options.go
│   ├── redis/
│   │   ├── store.go                # Redis message store
│   │   └── options.go
│   └── cosmos/
│       ├── store.go                # Cosmos DB NoSQL persistence
│       └── options.go
│
├── governance/                     # Data governance and security
│   ├── purview/
│   │   ├── client.go               # Microsoft Purview integration
│   │   ├── policy.go               # Data policy enforcement
│   │   └── options.go
│   └── compliance.go               # Compliance interfaces
│
├── devui/                          # Developer debugging UI
│   ├── server.go                   # Debug UI HTTP server
│   ├── inspector.go                # Agent inspector
│   └── templates/                  # UI templates
│
├── thread/                         # Conversation threading
│   ├── thread.go                   # AgentThread
│   ├── store.go                    # ChatMessageStore
│   └── inmemory.go                 # InMemory implementations
│
├── providers/                      # LLM provider implementations
│   ├── openai/
│   │   ├── client.go               # OpenAI ChatClient
│   │   ├── responses.go            # Responses API client
│   │   ├── assistants.go           # Assistants API client
│   │   └── options.go
│   ├── azure/
│   │   ├── client.go               # Azure OpenAI ChatClient
│   │   ├── persistent.go           # Azure AI Foundry Persistent Agents
│   │   ├── auth.go                 # Azure authentication
│   │   └── options.go
│   ├── anthropic/
│   │   ├── client.go               # Anthropic ChatClient
│   │   └── options.go
│   ├── bedrock/
│   │   ├── client.go               # AWS Bedrock ChatClient
│   │   ├── auth.go                 # AWS authentication
│   │   └── options.go
│   ├── ollama/
│   │   ├── client.go               # Ollama ChatClient
│   │   └── options.go
│   ├── azuresearch/
│   │   ├── client.go               # Azure AI Search for RAG
│   │   ├── index.go                # Index management
│   │   └── options.go
│   ├── copilotstudio/
│   │   ├── client.go               # Copilot Studio integration
│   │   └── options.go
│   ├── githubcopilot/
│   │   ├── client.go               # GitHub Copilot SDK
│   │   └── options.go
│   └── foundrylocal/
│       ├── client.go               # Local model execution
│       └── options.go
│
├── workflow/                       # Workflow orchestration
│   ├── workflow.go                 # Workflow definition
│   ├── builder.go                  # WorkflowBuilder
│   ├── executor.go                 # Executor interface
│   ├── edge.go                     # Edge, EdgeGroup
│   ├── checkpoint.go               # Checkpointing
│   ├── runner.go                   # WorkflowRunner
│   ├── events.go                   # Workflow events
│   └── groupchat/
│       ├── manager.go              # GroupChatManager
│       └── roundrobin.go           # RoundRobin implementation
│
├── protocol/                       # Inter-agent protocols
│   ├── a2a/
│   │   ├── agent.go                # A2A Agent
│   │   ├── client.go               # A2A Client
│   │   ├── types.go                # A2A Protocol types
│   │   └── server.go               # A2A Server
│   └── agui/
│       ├── client.go               # AG-UI Client
│       ├── server.go               # AG-UI Server
│       └── types.go                # AG-UI types
│
├── durable/                        # Durable agents (optional)
│   ├── agent.go                    # DurableAIAgent
│   ├── session.go                  # DurableAgentSession
│   ├── entity.go                   # Agent entity
│   └── temporal/                   # Temporal.io integration
│       └── worker.go
│
├── declarative/                    # Declarative agent definitions
│   ├── loader.go                   # YAML loader
│   ├── models.go                   # Schema models
│   └── factory.go                  # Agent factory
│
├── observability/                  # Telemetry
│   ├── otel.go                     # OpenTelemetry setup
│   ├── traces.go                   # Tracing utilities
│   ├── metrics.go                  # Metrics definitions
│   └── attributes.go               # Semantic conventions
│
├── hosting/                        # Server hosting
│   ├── http/
│   │   ├── handler.go              # HTTP handlers
│   │   └── router.go               # HTTP router setup
│   ├── grpc/
│   │   ├── server.go               # gRPC server
│   │   └── proto/                  # Protocol buffers
│   ├── a2a/
│   │   ├── handler.go              # A2A protocol HTTP handler
│   │   ├── router.go               # A2A endpoint routing
│   │   └── options.go              # A2A hosting options
│   ├── agui/
│   │   ├── handler.go              # AG-UI SSE handler
│   │   ├── sse.go                  # Server-sent events support
│   │   └── options.go              # AG-UI hosting options
│   ├── openaicompat/
│   │   ├── handler.go              # OpenAI-compatible API handler
│   │   ├── models.go               # OpenAI API models
│   │   └── options.go              # Compatibility options
│   └── azurefunctions/
│       ├── trigger.go              # Azure Functions trigger
│       ├── bindings.go             # Function bindings
│       └── options.go              # Functions hosting options
│
├── internal/                       # Internal utilities
│   ├── json/
│   │   └── utils.go                # JSON utilities
│   ├── sync/
│   │   └── pool.go                 # Object pooling
│   └── validation/
│       └── validate.go             # Input validation
│
└── examples/                       # Example applications
    ├── basic/
    ├── streaming/
    ├── tools/
    ├── workflow/
    └── server/
```

### 2.2 Go Naming Conventions Applied

| C#/Python Pattern | Go Equivalent |
|-------------------|---------------|
| `AIAgent` class | `agent.Agent` interface |
| `ChatClientAgent` class | `chatagent.Agent` struct |
| `IChatClient` interface | `chat.Client` interface |
| `ChatMessage` class | `chat.Message` struct |
| `AgentResponse` class | `agent.Response` struct |
| `RunAsync` method | `Run(ctx, ...) (..., error)` |
| `RunStreamingAsync` method | `RunStream(ctx, ...) (<-chan Update, error)` |
| `GetNewSessionAsync` method | `NewSession(ctx) (Session, error)` |
| Exception types | Error types implementing `error` |

---

## 3. Core Interface Definitions

### 3.1 Agent Interface (`agent/agent.go`)

```go
package agent

import (
    "context"
    "encoding/json"
    
    "github.com/microsoft/agent-framework-go/chat"
)

// Agent defines the core interface for AI agents.
// All agent implementations must satisfy this interface.
type Agent interface {
    // ID returns the unique identifier for this agent instance.
    ID() string
    
    // Name returns the human-readable name of the agent.
    Name() string
    
    // Description returns a description of the agent's purpose.
    Description() string
    
    // Run executes the agent with the provided messages and returns a response.
    Run(ctx context.Context, messages []chat.Message, opts ...RunOption) (*Response, error)
    
    // RunStream executes the agent and streams response updates.
    RunStream(ctx context.Context, messages []chat.Message, opts ...RunOption) (<-chan ResponseUpdate, error)
    
    // NewSession creates a new conversation session for this agent.
    NewSession(ctx context.Context) (Session, error)
    
    // RestoreSession restores a session from its serialized form.
    RestoreSession(ctx context.Context, data json.RawMessage) (Session, error)
    
    // GetService returns a service of the specified type, if available.
    GetService(serviceType interface{}) interface{}
}

// Metadata provides metadata about an agent.
type Metadata struct {
    ProviderName string
    ModelID      string
    Properties   map[string]interface{}
}
```

### 3.2 Chat Client Interface (`chat/client.go`)

```go
package chat

import "context"

// Client defines the interface for chat completion clients.
type Client interface {
    // GetResponse sends messages and returns a complete response.
    GetResponse(ctx context.Context, messages []Message, opts ...Option) (*Response, error)
    
    // GetStreamingResponse sends messages and returns a channel of updates.
    GetStreamingResponse(ctx context.Context, messages []Message, opts ...Option) (<-chan ResponseUpdate, error)
    
    // Metadata returns metadata about this client.
    Metadata() *ClientMetadata
}

// ClientMetadata contains information about a chat client.
type ClientMetadata struct {
    ProviderName string
    ModelID      string
    EndpointURI  string
}
```

### 3.3 Message Types (`chat/message.go`)

```go
package chat

import (
    "encoding/json"
    "time"
)

// Role represents the role of a message sender.
type Role string

const (
    RoleSystem    Role = "system"
    RoleUser      Role = "user"
    RoleAssistant Role = "assistant"
    RoleTool      Role = "tool"
)

// Message represents a single message in a conversation.
type Message struct {
    Role       Role      `json:"role"`
    Content    []Content `json:"content,omitempty"`
    Name       string    `json:"name,omitempty"`
    ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
    ToolCallID string    `json:"tool_call_id,omitempty"`
    
    // Additional properties for extensibility
    AdditionalProperties map[string]interface{} `json:"-"`
    
    // RawRepresentation holds the original provider-specific data
    RawRepresentation json.RawMessage `json:"-"`
}

// Content represents a piece of content within a message.
type Content interface {
    Type() string
    isContent()
}

// TextContent represents text content.
type TextContent struct {
    Text string `json:"text"`
}

func (TextContent) Type() string { return "text" }
func (TextContent) isContent()   {}

// ImageContent represents image content.
type ImageContent struct {
    URL       string `json:"url,omitempty"`
    Base64    string `json:"base64,omitempty"`
    MediaType string `json:"media_type,omitempty"`
}

func (ImageContent) Type() string { return "image" }
func (ImageContent) isContent()   {}

// ToolCallContent represents a tool invocation request.
type ToolCallContent struct {
    ID        string          `json:"id"`
    Name      string          `json:"name"`
    Arguments json.RawMessage `json:"arguments"`
}

func (ToolCallContent) Type() string { return "tool_call" }
func (ToolCallContent) isContent()   {}

// ToolResultContent represents the result of a tool invocation.
type ToolResultContent struct {
    ToolCallID string `json:"tool_call_id"`
    Content    string `json:"content"`
    IsError    bool   `json:"is_error,omitempty"`
}

func (ToolResultContent) Type() string { return "tool_result" }
func (ToolResultContent) isContent()   {}
```

### 3.4 Response Types (`agent/response.go`)

```go
package agent

import (
    "time"
    
    "github.com/microsoft/agent-framework-go/chat"
)

// Response represents the complete response from an agent run.
type Response struct {
    // Messages contains all messages produced by the agent.
    Messages []chat.Message
    
    // ResponseID is a unique identifier for this response.
    ResponseID string
    
    // CreatedAt indicates when the response was created.
    CreatedAt time.Time
    
    // Usage contains token usage information, if available.
    Usage *chat.UsageDetails
    
    // FinishReason indicates why the response ended.
    FinishReason chat.FinishReason
    
    // ContinuationToken for resuming long-running operations.
    ContinuationToken string
    
    // AdditionalProperties for extensibility.
    AdditionalProperties map[string]interface{}
    
    // RawRepresentation holds provider-specific response data.
    RawRepresentation interface{}
}

// Text returns the concatenated text content of all messages.
func (r *Response) Text() string {
    var result strings.Builder
    for _, msg := range r.Messages {
        for _, content := range msg.Content {
            if tc, ok := content.(chat.TextContent); ok {
                result.WriteString(tc.Text)
            }
        }
    }
    return result.String()
}

// ResponseUpdate represents a streaming update during agent execution.
type ResponseUpdate struct {
    // Kind indicates the type of update.
    Kind UpdateKind
    
    // Delta contains the incremental content, if applicable.
    Delta *ContentDelta
    
    // Message contains a complete message, for message-complete updates.
    Message *chat.Message
    
    // Usage contains token usage information.
    Usage *chat.UsageDetails
    
    // FinishReason for completion updates.
    FinishReason chat.FinishReason
    
    // Error for error updates.
    Error error
    
    // Metadata for the update.
    Metadata map[string]interface{}
}

// UpdateKind categorizes the type of streaming update.
type UpdateKind string

const (
    UpdateKindContentDelta    UpdateKind = "content_delta"
    UpdateKindMessageComplete UpdateKind = "message_complete"
    UpdateKindToolCall        UpdateKind = "tool_call"
    UpdateKindToolResult      UpdateKind = "tool_result"
    UpdateKindUsage           UpdateKind = "usage"
    UpdateKindError           UpdateKind = "error"
    UpdateKindDone            UpdateKind = "done"
)

// ContentDelta represents incremental content in a streaming response.
type ContentDelta struct {
    Role         chat.Role
    TextDelta    string
    ToolCallID   string
    ToolCallName string
    ArgsDelta    string
}

// AsyncRunContent represents the status of a long-running agent operation.
// This is used when an agent run cannot complete synchronously and must be
// polled or awaited for completion.
type AsyncRunContent struct {
    // RunID is the unique identifier for the async run.
    RunID string `json:"run_id"`
    
    // Status indicates the current state of the async run.
    Status AsyncRunStatus `json:"status"`
    
    // ThreadID is the conversation thread associated with this run.
    ThreadID string `json:"thread_id,omitempty"`
    
    // ExpiresAt indicates when the run will expire if not completed.
    ExpiresAt *time.Time `json:"expires_at,omitempty"`
    
    // StartedAt indicates when the run started processing.
    StartedAt *time.Time `json:"started_at,omitempty"`
    
    // CompletedAt indicates when the run finished (success or failure).
    CompletedAt *time.Time `json:"completed_at,omitempty"`
    
    // Error contains error details if the run failed.
    Error *AsyncRunError `json:"error,omitempty"`
}

// AsyncRunStatus represents the state of an async agent run.
type AsyncRunStatus string

const (
    // StatusQueued indicates the run is waiting to be processed.
    StatusQueued AsyncRunStatus = "queued"
    
    // StatusInProgress indicates the run is currently executing.
    StatusInProgress AsyncRunStatus = "in_progress"
    
    // StatusRequiresAction indicates the run needs user input (e.g., tool approval).
    StatusRequiresAction AsyncRunStatus = "requires_action"
    
    // StatusCompleted indicates the run finished successfully.
    StatusCompleted AsyncRunStatus = "completed"
    
    // StatusCancelled indicates the run was cancelled by the user.
    StatusCancelled AsyncRunStatus = "cancelled"
    
    // StatusFailed indicates the run encountered an unrecoverable error.
    StatusFailed AsyncRunStatus = "failed"
    
    // StatusExpired indicates the run exceeded its time limit.
    StatusExpired AsyncRunStatus = "expired"
)

// AsyncRunError contains details about a failed async run.
type AsyncRunError struct {
    Code    string `json:"code"`
    Message string `json:"message"`
}

// IsTerminal returns true if the status represents a final state.
func (s AsyncRunStatus) IsTerminal() bool {
    switch s {
    case StatusCompleted, StatusCancelled, StatusFailed, StatusExpired:
        return true
    default:
        return false
    }
}
```

### 3.5 Tool Interface (`tool/tool.go`)

```go
package tool

import (
    "context"
    "encoding/json"
)

// Tool defines the interface for agent tools.
type Tool interface {
    // Name returns the tool's name.
    Name() string
    
    // Description returns a description suitable for the model.
    Description() string
    
    // Parameters returns the JSON Schema for the tool's parameters.
    Parameters() json.RawMessage
    
    // Invoke executes the tool with the given arguments.
    Invoke(ctx context.Context, args json.RawMessage) (Result, error)
}

// Result represents the output of a tool invocation.
type Result struct {
    Content string
    IsError bool
}

// FunctionTool wraps a Go function as a Tool.
type FunctionTool struct {
    name        string
    description string
    parameters  json.RawMessage
    fn          interface{}
}

// NewFunctionTool creates a new FunctionTool from a Go function.
// The function signature is inspected to generate the parameters schema.
func NewFunctionTool(name, description string, fn interface{}) (*FunctionTool, error) {
    // Implementation generates JSON Schema from function signature
    // using reflection
}

// Func is a decorator for creating FunctionTools from functions.
// Usage: tool.Func(myFunction, "description")
func Func(fn interface{}, description string, opts ...FuncOption) *FunctionTool {
    // Implementation
}

// HostedTool represents a tool hosted by the LLM provider.
// Hosted tools are executed server-side by the provider.
type HostedTool interface {
    Tool
    // IsHosted returns true indicating this is a hosted tool.
    IsHosted() bool
    // RawRepresentation returns the provider-specific tool definition.
    RawRepresentation() interface{}
}

// HostedWebSearchTool represents a provider-hosted web search capability.
type HostedWebSearchTool struct {
    SearchContextSize string // "low", "medium", "high"
    UserLocation      *UserLocation
}

func (t *HostedWebSearchTool) Name() string        { return "web_search" }
func (t *HostedWebSearchTool) Description() string { return "Search the web for information" }
func (t *HostedWebSearchTool) IsHosted() bool      { return true }

// HostedCodeInterpreterTool represents a provider-hosted code execution sandbox.
type HostedCodeInterpreterTool struct {
    Container string   // Optional container type
    FileIDs   []string // Files accessible to the interpreter
}

func (t *HostedCodeInterpreterTool) Name() string        { return "code_interpreter" }
func (t *HostedCodeInterpreterTool) Description() string { return "Execute code in a sandbox" }
func (t *HostedCodeInterpreterTool) IsHosted() bool      { return true }

// HostedFileSearchTool represents a provider-hosted file/vector search.
type HostedFileSearchTool struct {
    VectorStoreIDs []string
    MaxResults     int
}

func (t *HostedFileSearchTool) Name() string        { return "file_search" }
func (t *HostedFileSearchTool) Description() string { return "Search uploaded files" }
func (t *HostedFileSearchTool) IsHosted() bool      { return true }

// HostedMCPTool represents an MCP server tool bridged through the provider.
type HostedMCPTool struct {
    ServerURL   string
    ServerLabel string
    AllowedTools []string
}

func (t *HostedMCPTool) Name() string        { return "mcp_" + t.ServerLabel }
func (t *HostedMCPTool) Description() string { return "MCP server: " + t.ServerLabel }
func (t *HostedMCPTool) IsHosted() bool      { return true }

// UserLocation provides geographic context for web search.
type UserLocation struct {
    Type        string // "approximate"
    City        string
    Region      string
    Country     string
    CountryCode string
}
```

### 3.6 Session Interface (`agent/session.go`)

```go
package agent

import (
    "context"
    "encoding/json"
    
    "github.com/microsoft/agent-framework-go/chat"
)

// Session represents a conversation session with an agent.
type Session interface {
    // ID returns the session identifier.
    ID() string
    
    // AddMessages adds messages to the session history.
    AddMessages(ctx context.Context, messages ...chat.Message) error
    
    // GetMessages returns all messages in the session.
    GetMessages(ctx context.Context) ([]chat.Message, error)
    
    // Serialize serializes the session state.
    Serialize() (json.RawMessage, error)
}

// InMemorySession provides a simple in-memory session implementation.
type InMemorySession struct {
    id       string
    messages []chat.Message
}

// NewInMemorySession creates a new in-memory session.
func NewInMemorySession() *InMemorySession {
    return &InMemorySession{
        id:       generateID(),
        messages: make([]chat.Message, 0),
    }
}
```

### 3.7 Middleware Interface (`middleware/middleware.go`)

```go
package middleware

import (
    "context"
    
    "github.com/microsoft/agent-framework-go/agent"
    "github.com/microsoft/agent-framework-go/chat"
)

// AgentMiddleware defines the interface for agent middleware.
type AgentMiddleware interface {
    // Process handles the agent invocation.
    Process(ctx context.Context, runCtx *AgentRunContext, next AgentNext) error
}

// AgentNext is the function to call the next middleware in the pipeline.
type AgentNext func(ctx context.Context, runCtx *AgentRunContext) error

// AgentRunContext contains the context for an agent run.
type AgentRunContext struct {
    Agent      agent.Agent
    Messages   []chat.Message
    Session    agent.Session
    IsStream   bool
    Result     *agent.Response
    StreamChan <-chan agent.ResponseUpdate
    Metadata   map[string]interface{}
    Terminate  bool
}

// ChatMiddleware defines the interface for chat client middleware.
type ChatMiddleware interface {
    // Process handles the chat completion request.
    Process(ctx context.Context, chatCtx *ChatContext, next ChatNext) error
}

// ChatNext is the function to call the next middleware in the pipeline.
type ChatNext func(ctx context.Context, chatCtx *ChatContext) error

// ChatContext contains the context for a chat completion.
type ChatContext struct {
    Client    chat.Client
    Messages  []chat.Message
    Options   *chat.Options
    Result    *chat.Response
    StreamChan <-chan chat.ResponseUpdate
    Metadata  map[string]interface{}
}

// FunctionMiddleware defines the interface for function invocation middleware.
type FunctionMiddleware interface {
    // Process handles the function invocation.
    Process(ctx context.Context, fnCtx *FunctionContext, next FunctionNext) error
}

// FunctionNext is the function to call the next middleware.
type FunctionNext func(ctx context.Context, fnCtx *FunctionContext) error

// FunctionContext contains the context for a function invocation.
type FunctionContext struct {
    Function  tool.Tool
    Arguments json.RawMessage
    Result    *tool.Result
    Metadata  map[string]interface{}
}
```

---

## 4. Implementation Phases

### Phase 1: Core Foundation (Weeks 1-4)

**Goal:** Establish the core abstractions and interfaces.

#### Deliverables:
1. **Core Packages**
   - `agent/` - Agent interface, Response, Session, Metadata
   - `chat/` - Client interface, Message, Content types
   - `internal/` - JSON utilities, validation helpers

2. **In-Memory Implementations**
   - `InMemorySession` for testing
   - `InMemoryChatMessageStore`

3. **Testing Infrastructure**
   - Unit test framework setup
   - Mock implementations for interfaces
   - Test fixtures

#### Acceptance Criteria:
- [ ] All core interfaces defined and documented
- [ ] In-memory implementations pass unit tests
- [ ] 90%+ code coverage on core packages
- [ ] Comprehensive godoc documentation

### Phase 2: Provider Implementations (Weeks 5-8)

**Goal:** Implement LLM provider integrations.

#### Deliverables:
1. **OpenAI Provider**
   - Chat completions API
   - Responses API
   - Streaming support
   - Tool/function calling

2. **Azure OpenAI Provider**
   - Azure authentication (Azure AD, API key)
   - All OpenAI features

3. **Anthropic Provider**
   - Claude chat completions
   - Streaming support
   - Tool calling

4. **AWS Bedrock Provider**
   - Amazon Bedrock LLM integration
   - AWS authentication (IAM, credentials)
   - Claude, Titan, and other Bedrock models
   - Streaming support

5. **Tool System**
   - FunctionTool implementation
   - Hosted tools (WebSearch, etc.)
   - Function invocation pipeline

5. **OpenTelemetry Integration**
   - Tracing for agent runs
   - Metrics for token usage
   - Semantic conventions

#### Acceptance Criteria:
- [ ] All providers pass integration tests
- [ ] Streaming works correctly for all providers
- [ ] Tool calling works with all providers
- [ ] OpenTelemetry exports correct spans/metrics

### Phase 3: Advanced Features (Weeks 9-12)

**Goal:** Implement middleware, memory, and advanced patterns.

#### Deliverables:
1. **Middleware Pipeline**
   - Agent middleware
   - Chat middleware  
   - Function middleware
   - Pipeline builder

2. **Memory/Context**
   - ContextProvider interface
   - ChatHistoryProvider
   - Memory implementations

3. **Thread Management**
   - AgentThread implementation
   - Persistent thread stores (Redis, etc.)

4. **Vector Search/RAG**
   - Azure AI Search integration
   - Vector store abstractions
   - Hybrid search (keyword + semantic)
   - RAG pattern implementation

5. **ChatClientAgent**
   - Full implementation with all features
   - Builder pattern
   - Option pattern

#### Acceptance Criteria:
- [ ] Middleware pipeline correctly chains handlers
- [ ] Context providers integrate with agent runs
- [ ] Thread persistence works with external stores
- [ ] End-to-end tests with middleware

### Phase 4: Workflows and Protocols (Weeks 13-18)

**Goal:** Implement workflow orchestration and inter-agent protocols.

#### Deliverables:
1. **Workflow Engine**
   - Workflow definition
   - WorkflowBuilder
   - Executor interface
   - Edge/EdgeGroup
   - Checkpointing
   - Event streaming

2. **A2A Protocol**
   - A2A Agent
   - A2A Client
   - A2A Server (HTTP)

3. **AG-UI Protocol**
   - AG-UI Client
   - AG-UI Server

4. **Group Chat**
   - GroupChatManager
   - Round-robin orchestrator
   - Custom selector support

#### Acceptance Criteria:
- [ ] Workflows execute correctly with DAG patterns
- [ ] Checkpointing and recovery works
- [ ] A2A agents can communicate across network
- [ ] AG-UI integration tests pass

### Phase 5: Enterprise Features (Weeks 19-24)

**Goal:** Production-ready enterprise features.

#### Deliverables:
1. **Durable Agents**
   - Temporal.io integration (preferred for Go)
   - Long-running orchestration
   - State management

2. **Declarative Agents**
   - YAML schema definitions
   - Agent loader
   - Factory pattern

3. **Hosting**
   - HTTP server with standard library
   - gRPC server option
   - Health checks
   - Graceful shutdown

4. **MCP Integration**
   - MCP client
   - MCP server
   - Tool bridging

5. **Enterprise Integrations**
   - Microsoft Copilot Studio integration
   - GitHub Copilot SDK support
   - Microsoft Purview data governance
   - Azure AI Search for RAG patterns

6. **Developer Tools**
   - Developer debugging UI (DevUI)
   - Agent inspector and tracing
   - Local model execution (Foundry Local)

7. **Protocol-Specific Hosting**
   - A2A protocol hosting (`hosting/a2a/`)
   - AG-UI SSE hosting (`hosting/agui/`)
   - OpenAI-compatible API hosting (`hosting/openaicompat/`)
   - Azure Functions hosting (`hosting/azurefunctions/`)

8. **Production Hardening**
   - Rate limiting
   - Circuit breakers
   - Retry policies
   - Connection pooling

#### Acceptance Criteria:
- [ ] Durable agents survive process restarts
- [ ] Declarative agents load from YAML correctly
- [ ] HTTP/gRPC servers pass load testing
- [ ] MCP tools work with agents

---

## 5. Go-Specific Design Patterns

### 5.1 Error Handling

```go
package agent

import (
    "errors"
    "fmt"
)

// Sentinel errors for common cases
var (
    ErrSessionNotFound     = errors.New("agent: session not found")
    ErrInvalidMessage      = errors.New("agent: invalid message")
    ErrProviderUnavailable = errors.New("agent: provider unavailable")
    ErrToolNotFound        = errors.New("agent: tool not found")
    ErrToolInvocationFailed = errors.New("agent: tool invocation failed")
)

// AgentError provides detailed error information.
type AgentError struct {
    Op      string // Operation that failed
    AgentID string // Agent involved
    Err     error  // Underlying error
}

func (e *AgentError) Error() string {
    return fmt.Sprintf("agent %s: %s: %v", e.AgentID, e.Op, e.Err)
}

func (e *AgentError) Unwrap() error {
    return e.Err
}

// IsRetryable returns true if the error is transient.
func IsRetryable(err error) bool {
    var agentErr *AgentError
    if errors.As(err, &agentErr) {
        // Check for retryable conditions
        return errors.Is(agentErr.Err, ErrProviderUnavailable)
    }
    return false
}
```

### 5.2 Options Pattern

```go
package agent

// RunOption configures a Run call.
type RunOption func(*runConfig)

type runConfig struct {
    session     Session
    tools       []tool.Tool
    maxTokens   int
    temperature float32
    metadata    map[string]interface{}
}

// WithSession sets the session for the run.
func WithSession(s Session) RunOption {
    return func(c *runConfig) {
        c.session = s
    }
}

// WithTools adds tools to the run.
func WithTools(tools ...tool.Tool) RunOption {
    return func(c *runConfig) {
        c.tools = append(c.tools, tools...)
    }
}

// WithMaxTokens sets the maximum tokens.
func WithMaxTokens(n int) RunOption {
    return func(c *runConfig) {
        c.maxTokens = n
    }
}

// WithTemperature sets the temperature.
func WithTemperature(t float32) RunOption {
    return func(c *runConfig) {
        c.temperature = t
    }
}

// Usage:
// resp, err := agent.Run(ctx, messages,
//     agent.WithSession(session),
//     agent.WithTools(weatherTool),
//     agent.WithMaxTokens(1000),
// )
```

### 5.3 Context Usage

```go
package chatagent

import (
    "context"
    "time"
)

// contextKey is used for context values.
type contextKey int

const (
    agentIDKey contextKey = iota
    sessionIDKey
    traceIDKey
)

// WithAgentID adds the agent ID to the context.
func WithAgentID(ctx context.Context, id string) context.Context {
    return context.WithValue(ctx, agentIDKey, id)
}

// AgentIDFromContext retrieves the agent ID from context.
func AgentIDFromContext(ctx context.Context) string {
    if id, ok := ctx.Value(agentIDKey).(string); ok {
        return id
    }
    return ""
}

// Run implementation with context handling
func (a *Agent) Run(ctx context.Context, messages []chat.Message, opts ...RunOption) (*agent.Response, error) {
    // Add timeout if not already set
    if _, ok := ctx.Deadline(); !ok {
        var cancel context.CancelFunc
        ctx, cancel = context.WithTimeout(ctx, 30*time.Second)
        defer cancel()
    }
    
    // Add agent ID to context for tracing
    ctx = WithAgentID(ctx, a.ID())
    
    // ... implementation
}
```

### 5.4 Concurrency Patterns

```go
package workflow

import (
    "context"
    "sync"
)

// FanOut executes multiple executors concurrently.
func (r *Runner) FanOut(ctx context.Context, executors []Executor, input interface{}) ([]interface{}, error) {
    results := make([]interface{}, len(executors))
    errs := make([]error, len(executors))
    
    var wg sync.WaitGroup
    wg.Add(len(executors))
    
    for i, exec := range executors {
        go func(idx int, e Executor) {
            defer wg.Done()
            result, err := e.Execute(ctx, input)
            results[idx] = result
            errs[idx] = err
        }(i, exec)
    }
    
    wg.Wait()
    
    // Collect errors
    var combinedErr error
    for _, err := range errs {
        if err != nil {
            combinedErr = errors.Join(combinedErr, err)
        }
    }
    
    return results, combinedErr
}

// Streaming with channels
func (a *Agent) RunStream(ctx context.Context, messages []chat.Message, opts ...RunOption) (<-chan agent.ResponseUpdate, error) {
    updates := make(chan agent.ResponseUpdate, 100)
    
    go func() {
        defer close(updates)
        
        // Get streaming response from underlying client
        stream, err := a.client.GetStreamingResponse(ctx, messages, chatOpts...)
        if err != nil {
            updates <- agent.ResponseUpdate{
                Kind:  agent.UpdateKindError,
                Error: err,
            }
            return
        }
        
        for update := range stream {
            select {
            case <-ctx.Done():
                updates <- agent.ResponseUpdate{
                    Kind:  agent.UpdateKindError,
                    Error: ctx.Err(),
                }
                return
            case updates <- convertUpdate(update):
            }
        }
        
        updates <- agent.ResponseUpdate{Kind: agent.UpdateKindDone}
    }()
    
    return updates, nil
}
```

### 5.5 Interface Composition

```go
package agent

// Small, focused interfaces that can be composed

// Runner can run with messages.
type Runner interface {
    Run(ctx context.Context, messages []chat.Message, opts ...RunOption) (*Response, error)
}

// StreamRunner can run with streaming.
type StreamRunner interface {
    RunStream(ctx context.Context, messages []chat.Message, opts ...RunOption) (<-chan ResponseUpdate, error)
}

// SessionProvider can create and restore sessions.
type SessionProvider interface {
    NewSession(ctx context.Context) (Session, error)
    RestoreSession(ctx context.Context, data json.RawMessage) (Session, error)
}

// Identifier provides agent identification.
type Identifier interface {
    ID() string
    Name() string
    Description() string
}

// ServiceProvider provides access to internal services.
type ServiceProvider interface {
    GetService(serviceType interface{}) interface{}
}

// Agent composes all the interfaces.
type Agent interface {
    Identifier
    Runner
    StreamRunner
    SessionProvider
    ServiceProvider
}
```

---

## 6. Testing Strategy

### 6.1 Test Types

| Test Type | Coverage Target | Location |
|-----------|-----------------|----------|
| Unit Tests | 90%+ | `*_test.go` alongside code |
| Integration Tests | Key flows | `integration_test.go` with build tag |
| End-to-End Tests | Critical paths | `e2e/` directory |
| Benchmark Tests | Performance-critical | `*_bench_test.go` |
| Fuzz Tests | Input parsing | `*_fuzz_test.go` |

### 6.2 Mock Strategy

```go
package agent_test

import (
    "context"
    
    "github.com/microsoft/agent-framework-go/agent"
    "github.com/microsoft/agent-framework-go/chat"
)

// MockAgent implements agent.Agent for testing.
type MockAgent struct {
    IDFunc           func() string
    NameFunc         func() string
    DescriptionFunc  func() string
    RunFunc          func(ctx context.Context, messages []chat.Message, opts ...agent.RunOption) (*agent.Response, error)
    RunStreamFunc    func(ctx context.Context, messages []chat.Message, opts ...agent.RunOption) (<-chan agent.ResponseUpdate, error)
    NewSessionFunc   func(ctx context.Context) (agent.Session, error)
}

func (m *MockAgent) ID() string {
    if m.IDFunc != nil {
        return m.IDFunc()
    }
    return "mock-agent"
}

func (m *MockAgent) Run(ctx context.Context, messages []chat.Message, opts ...agent.RunOption) (*agent.Response, error) {
    if m.RunFunc != nil {
        return m.RunFunc(ctx, messages, opts...)
    }
    return &agent.Response{Messages: []chat.Message{}}, nil
}

// ... other methods
```

### 6.3 Table-Driven Tests

```go
func TestAgent_Run(t *testing.T) {
    tests := []struct {
        name     string
        messages []chat.Message
        opts     []agent.RunOption
        want     *agent.Response
        wantErr  error
    }{
        {
            name: "simple message",
            messages: []chat.Message{
                {Role: chat.RoleUser, Content: []chat.Content{chat.TextContent{Text: "Hello"}}},
            },
            want: &agent.Response{
                Messages: []chat.Message{
                    {Role: chat.RoleAssistant, Content: []chat.Content{chat.TextContent{Text: "Hello!"}}},
                },
            },
        },
        {
            name:     "empty messages",
            messages: []chat.Message{},
            wantErr:  agent.ErrInvalidMessage,
        },
        // ... more test cases
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            a := NewTestAgent(t)
            got, err := a.Run(context.Background(), tt.messages, tt.opts...)
            
            if !errors.Is(err, tt.wantErr) {
                t.Errorf("Run() error = %v, wantErr %v", err, tt.wantErr)
                return
            }
            
            if diff := cmp.Diff(tt.want, got); diff != "" {
                t.Errorf("Run() mismatch (-want +got):\n%s", diff)
            }
        })
    }
}
```

### 6.4 Integration Test Example

```go
//go:build integration

package integration_test

import (
    "context"
    "os"
    "testing"
    "time"
    
    "github.com/microsoft/agent-framework-go/chatagent"
    "github.com/microsoft/agent-framework-go/chat"
    "github.com/microsoft/agent-framework-go/providers/openai"
)

func TestOpenAI_Integration(t *testing.T) {
    apiKey := os.Getenv("OPENAI_API_KEY")
    if apiKey == "" {
        t.Skip("OPENAI_API_KEY not set")
    }
    
    client, err := openai.NewClient(
        openai.WithAPIKey(apiKey),
        openai.WithModel("gpt-4o-mini"),
    )
    if err != nil {
        t.Fatalf("failed to create client: %v", err)
    }
    
    agent := chatagent.New(client,
        chatagent.WithName("TestAgent"),
        chatagent.WithInstructions("You are a helpful assistant."),
    )
    
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    
    resp, err := agent.Run(ctx, []chat.Message{
        chat.NewUserMessage("Say hello in exactly 3 words."),
    })
    if err != nil {
        t.Fatalf("Run failed: %v", err)
    }
    
    if len(resp.Messages) == 0 {
        t.Error("expected at least one response message")
    }
    
    t.Logf("Response: %s", resp.Text())
}
```

---

## 7. Documentation Requirements

### 7.1 Documentation Types

1. **Package Documentation** - Every package has a `doc.go` file
2. **API Documentation** - Comprehensive godoc for all public APIs
3. **Examples** - Runnable examples in `example_test.go` files
4. **Tutorials** - Step-by-step guides in `/docs`
5. **Architecture Decision Records** - For major design decisions

### 7.2 Example Documentation

```go
// Package agent provides the core abstractions for AI agents.
//
// The agent package defines the fundamental interfaces and types
// used throughout the Agent Framework. The central abstraction is
// the Agent interface, which all agent implementations must satisfy.
//
// # Creating an Agent
//
// The most common way to create an agent is using the chatagent package:
//
//	client, _ := openai.NewClient(openai.WithAPIKey("sk-..."))
//	agent := chatagent.New(client,
//	    chatagent.WithName("MyAgent"),
//	    chatagent.WithInstructions("You are helpful."),
//	)
//
// # Running an Agent
//
// Agents can be run synchronously or with streaming:
//
//	// Synchronous
//	resp, err := agent.Run(ctx, messages)
//
//	// Streaming
//	updates, err := agent.RunStream(ctx, messages)
//	for update := range updates {
//	    fmt.Print(update.Delta.TextDelta)
//	}
//
// # Sessions
//
// Agents maintain conversation state through sessions:
//
//	session, _ := agent.NewSession(ctx)
//	resp, _ := agent.Run(ctx, messages, agent.WithSession(session))
//
// # Middleware
//
// Agent behavior can be customized through middleware. See the
// middleware package for details.
package agent
```

---

## 8. Dependencies

### 8.1 Required Dependencies

| Dependency | Purpose | Version |
|------------|---------|---------|
| `go.opentelemetry.io/otel` | Observability | v1.x |
| `go.opentelemetry.io/otel/trace` | Tracing | v1.x |
| `go.opentelemetry.io/otel/metric` | Metrics | v1.x |
| Standard library | HTTP, JSON, etc. | Go 1.22+ |

### 8.2 Optional Dependencies

| Dependency | Purpose | When Needed |
|------------|---------|-------------|
| `github.com/sashabaranov/go-openai` | OpenAI client | OpenAI provider |
| `github.com/Azure/azure-sdk-for-go` | Azure auth | Azure OpenAI |
| `github.com/liushuangls/go-anthropic` | Anthropic client | Anthropic provider |
| `go.temporal.io/sdk` | Durable workflows | Durable agents |
| `github.com/redis/go-redis/v9` | Redis | Distributed sessions |
| `gopkg.in/yaml.v3` | YAML parsing | Declarative agents |
| `google.golang.org/grpc` | gRPC | gRPC hosting |

### 8.3 Minimum Go Version

**Go 1.22+** required for:
- Generic type constraints
- `errors.Join`
- Enhanced `net/http` features
- `slices` and `maps` packages

---

## 9. Release Strategy

### 9.1 Versioning

Follow semantic versioning (SemVer):
- **Major (1.x.x)**: Breaking API changes
- **Minor (x.1.x)**: New features, backwards compatible
- **Patch (x.x.1)**: Bug fixes

### 9.2 Pre-release Phases

1. **Alpha (v0.1.0-alpha.x)**: Core interfaces, basic providers
2. **Beta (v0.1.0-beta.x)**: Feature complete, API stabilizing
3. **RC (v0.1.0-rc.x)**: API frozen, bug fixes only
4. **GA (v1.0.0)**: Production ready

### 9.3 Release Checklist

- [ ] All tests passing
- [ ] Code coverage ≥ 90%
- [ ] Documentation complete
- [ ] CHANGELOG.md updated
- [ ] Examples verified
- [ ] Security scan passed
- [ ] Performance benchmarks acceptable
- [ ] API compatibility verified

---

## 10. Open Questions

1. **Temporal vs. Custom Durable Framework**: Should we use Temporal.io for durable agents (mature Go support) or build a custom solution for parity with the .NET Durable Task Framework?

2. **Generics Usage**: How extensively should we use Go generics vs. interface{} for flexibility?

3. **Error Wrapping Strategy**: Standard library `errors.Join` vs. custom error types for better diagnostics?

4. **Streaming Implementation**: Channels vs. iterator pattern (Go 1.23 rangefunc)?

5. **Code Generation**: Should we use code generation for provider clients or hand-write?

---

## 11. Success Metrics

| Metric | Target |
|--------|--------|
| Test Coverage | ≥ 90% |
| Benchmark: Simple Run | < 10ms overhead |
| Benchmark: Streaming | < 1ms per chunk overhead |
| Memory: Per-agent | < 1KB base |
| API Parity | 100% of Phase 1-4 features |
| Documentation Coverage | 100% public APIs |
| Zero CVEs | On release |

---

## 12. Team & Timeline

### Suggested Team Composition

- **Tech Lead**: 1 senior Go developer
- **Core Developers**: 2-3 Go developers
- **QA Engineer**: 1 engineer for test automation
- **Technical Writer**: Part-time for documentation

### Timeline Summary

| Phase | Duration | Deliverables |
|-------|----------|--------------|
| Phase 1: Core | 4 weeks | Interfaces, in-memory |
| Phase 2: Providers | 4 weeks | OpenAI, Azure, Anthropic, OTel |
| Phase 3: Advanced | 4 weeks | Middleware, Memory, Threads |
| Phase 4: Workflows | 6 weeks | Workflows, A2A, AG-UI |
| Phase 5: Enterprise | 6 weeks | Durable, Declarative, Hosting |
| **Total** | **24 weeks** | Full feature parity |

---

## Appendix A: File-by-File Mapping

| C# File | Python File | Go Package/File |
|---------|-------------|-----------------|
| `AIAgent.cs` | `_agents.py` | `agent/agent.go` |
| `AgentResponse.cs` | `_types.py` | `agent/response.go` |
| `AgentSession.cs` | `_threads.py` | `agent/session.go` |
| `ChatClientAgent.cs` | `_clients.py` | `chatagent/agent.go` |
| `IChatClient` (MEAI) | `ChatClientProtocol` | `chat/client.go` |
| `ChatMessage` (MEAI) | `ChatMessage` | `chat/message.go` |
| `AITool` (MEAI) | `ToolProtocol` | `tool/tool.go` |
| `Workflow.cs` | `_workflow.py` | `workflow/workflow.go` |
| `Middleware/*` | `_middleware.py` | `middleware/*.go` |
| `A2AAgent.cs` | `a2a/*.py` | `protocol/a2a/*.go` |

---

## Appendix B: Example Usage (Target API)

```go
package main

import (
    "context"
    "fmt"
    "log"
    
    "github.com/microsoft/agent-framework-go/chatagent"
    "github.com/microsoft/agent-framework-go/chat"
    "github.com/microsoft/agent-framework-go/providers/openai"
    "github.com/microsoft/agent-framework-go/tool"
)

func main() {
    ctx := context.Background()
    
    // Create OpenAI client
    client, err := openai.NewClient(
        openai.WithAPIKey("sk-..."),
        openai.WithModel("gpt-4o"),
    )
    if err != nil {
        log.Fatal(err)
    }
    
    // Define a tool
    weatherTool := tool.Func(getWeather, "Get the current weather for a location")
    
    // Create agent
    agent := chatagent.New(client,
        chatagent.WithName("WeatherBot"),
        chatagent.WithInstructions("You help users check the weather."),
        chatagent.WithTools(weatherTool),
    )
    
    // Create session
    session, _ := agent.NewSession(ctx)
    
    // Run agent
    resp, err := agent.Run(ctx,
        []chat.Message{chat.NewUserMessage("What's the weather in Seattle?")},
        chatagent.WithSession(session),
    )
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Println(resp.Text())
    
    // Streaming example
    updates, err := agent.RunStream(ctx,
        []chat.Message{chat.NewUserMessage("Tell me more about the forecast")},
        chatagent.WithSession(session),
    )
    if err != nil {
        log.Fatal(err)
    }
    
    for update := range updates {
        if update.Kind == agent.UpdateKindContentDelta {
            fmt.Print(update.Delta.TextDelta)
        }
    }
    fmt.Println()
}

func getWeather(location string) (string, error) {
    return fmt.Sprintf("The weather in %s is sunny, 72°F", location), nil
}
```

---

*Document Version: 1.0*
*Created: January 2026*
*Author: Microsoft Agent Framework Team*
