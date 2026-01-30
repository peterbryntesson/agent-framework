<!-- markdownlint-disable-file -->
# Implementation Details: Go Port - Epics, Features, and User Stories

## Context Reference

Sources: [docs/design/golang-port-plan.md](docs/design/golang-port-plan.md)

---

## Epic 1: Project Foundation and Core Abstractions (Phase 1, Weeks 1-4)

### Feature 1.1: Repository Setup and Module Initialization

Create the Go module structure and foundational project files.

Files:

* `go.mod` - Module definition with Go 1.22+ requirement
* `go.sum` - Dependency checksums
* `README.md` - Project documentation and quick start
* `LICENSE` - MIT license file
* `.github/workflows/ci.yml` - CI/CD pipeline configuration
* `Makefile` - Build automation commands

#### User Story 1.1.1: Initialize Go Module

As a developer, I want a properly configured Go module so that I can import and use the SDK.

Acceptance criteria:

* Module path set to `github.com/microsoft/agent-framework-go`
* Go version constraint set to 1.22+
* Initial dependencies declared for otel, testing

#### User Story 1.1.2: Configure CI Pipeline

As a maintainer, I want automated CI checks so that code quality is enforced.

Acceptance criteria:

* GitHub Actions workflow runs on PR and main
* Runs `go build`, `go test`, `go vet`, linting
* Enforces 90%+ coverage threshold

#### User Story 1.1.3: Create Project Documentation

As a developer, I want a README with quick start instructions so that I can begin using the SDK.

Acceptance criteria:

* Installation instructions via `go get`
* Basic usage example with ChatClientAgent
* Links to API documentation

### Feature 1.2: Core Agent Interface Package

Implement the `agent/` package with fundamental agent abstractions.

Files:

* `agent/agent.go` - Agent interface definition
* `agent/response.go` - AgentResponse and ResponseUpdate types
* `agent/session.go` - Session interface and InMemorySession
* `agent/options.go` - RunOption functional options pattern
* `agent/metadata.go` - AIAgentMetadata struct
* `agent/errors.go` - Agent-specific error types
* `agent/doc.go` - Package documentation

#### User Story 1.2.1: Define Agent Interface

As a developer, I want a core Agent interface so that I can build agent implementations.

Acceptance criteria:

* `Agent` interface with `ID()`, `Name()`, `Description()`, `Metadata()` methods
* `Run(ctx, messages, opts...) (*Response, error)` method signature
* `RunStream(ctx, messages, opts...) (<-chan ResponseUpdate, error)` method signature
* `NewSession(ctx) (Session, error)` method signature
* `RestoreSession(ctx, data) (Session, error)` method signature
* `GetService(serviceType) interface{}` method signature

#### User Story 1.2.2: Implement Response Types

As a developer, I want response types so that I can handle agent outputs.

Acceptance criteria:

* `Response` struct with Messages, Usage, FinishReason, SessionState, Metadata
* `Text()` method concatenates all text content
* `ResponseUpdate` struct with Kind, Delta, Message, Metadata
* `UpdateKind` constants: ContentDelta, ToolCall, ToolResult, MessageComplete, Error, Done
* `ContentDelta` struct with Role, TextDelta, ToolCallId, Name, ArgsDelta
* `AsyncRunContent` struct for long-running operations with status tracking

#### User Story 1.2.3: Implement Session Interface

As a developer, I want session management so that I can maintain conversation state.

Acceptance criteria:

* `Session` interface with ID(), Messages(), AddMessage(), Serialize() methods
* `InMemorySession` implementation for testing and simple use cases
* `NewInMemorySession()` constructor with UUID generation

#### User Story 1.2.4: Implement Options Pattern

As a developer, I want functional options so that I can configure agent runs flexibly.

Acceptance criteria:

* `RunOption` function type `func(*runConfig)`
* `WithSession(Session)` option
* `WithTools(...Tool)` option
* `WithMaxTokens(int)` option
* `WithTemperature(float32)` option
* `WithMetadata(map[string]interface{})` option

#### User Story 1.2.5: Implement Error Types

As a developer, I want structured errors so that I can handle failures appropriately.

Acceptance criteria:

* Sentinel errors: `ErrSessionNotFound`, `ErrInvalidInput`, `ErrRateLimited`, `ErrProviderError`, `ErrToolInvocationFailed`
* `AgentError` struct with Op, AgentID, Err fields
* `Error()` and `Unwrap()` methods for error wrapping
* `IsRetryable(error) bool` helper function

### Feature 1.3: Chat Client Abstractions Package

Implement the `chat/` package with chat completion abstractions.

Files:

* `chat/client.go` - ChatClient interface
* `chat/message.go` - Message, Content types
* `chat/response.go` - ChatResponse, ChatResponseUpdate
* `chat/options.go` - ChatOptions configuration
* `chat/tool.go` - Tool definition types
* `chat/usage.go` - UsageDetails struct
* `chat/doc.go` - Package documentation

#### User Story 1.3.1: Define ChatClient Interface

As a developer, I want a chat client interface so that providers can implement a common contract.

Acceptance criteria:

* `Client` interface with GetResponse, GetStreamingResponse, Metadata methods
* `GetResponse(ctx, messages, options) (*Response, error)` signature
* `GetStreamingResponse(ctx, messages, options) (<-chan ResponseUpdate, error)` signature
* `ClientMetadata` struct with ProviderName, ModelID, EndpointURI

#### User Story 1.3.2: Implement Message Types

As a developer, I want message types so that I can structure conversations.

Acceptance criteria:

* `Role` type with constants: System, User, Assistant, Tool
* `Message` struct with Role, Contents, Name, ToolCalls, ToolCallID, CreatedAt, RawRepresentation
* `Content` interface with Type() method and sealed marker
* `TextContent`, `ImageContent`, `ToolCallContent`, `ToolResultContent` implementations
* `NewUserMessage(text)`, `NewSystemMessage(text)`, `NewAssistantMessage(text)` constructors

#### User Story 1.3.3: Implement Response Types

As a developer, I want chat response types so that I can process completions.

Acceptance criteria:

* `Response` struct with Message, FinishReason, Usage, RawRepresentation
* `ResponseUpdate` struct for streaming chunks
* `FinishReason` constants: Stop, Length, ToolCalls, ContentFilter

#### User Story 1.3.4: Implement Usage Types

As a developer, I want usage tracking so that I can monitor token consumption.

Acceptance criteria:

* `UsageDetails` struct with InputTokens, OutputTokens, TotalTokens
* `CachedTokens`, `ReasoningTokens` optional fields

### Feature 1.4: Internal Utilities Package

Implement the `internal/` package with shared utilities.

Files:

* `internal/json/utils.go` - JSON marshaling utilities
* `internal/sync/pool.go` - Object pooling helpers
* `internal/validation/validate.go` - Input validation functions

#### User Story 1.4.1: Implement JSON Utilities

As a developer, I want JSON utilities so that I can serialize/deserialize efficiently.

Acceptance criteria:

* `MarshalToRawMessage(v interface{}) (json.RawMessage, error)`
* `UnmarshalFromRawMessage(data json.RawMessage, v interface{}) error`
* Proper handling of nil and empty values

#### User Story 1.4.2: Implement Validation Helpers

As a developer, I want validation helpers so that I can validate inputs consistently.

Acceptance criteria:

* `RequireNotNil(v interface{}, name string) error`
* `RequireNotEmpty(s string, name string) error`
* `ValidateMessages(messages []Message) error`

### Feature 1.5: Testing Infrastructure

Set up comprehensive testing infrastructure.

Files:

* `agent/agent_test.go` - Agent interface tests
* `agent/mock_test.go` - Mock implementations
* `chat/client_test.go` - Client interface tests
* `testdata/` - Test fixtures directory

#### User Story 1.5.1: Create Mock Implementations

As a tester, I want mock implementations so that I can test without external dependencies.

Acceptance criteria:

* `MockAgent` implementing `Agent` interface with configurable function fields
* `MockChatClient` implementing `Client` interface with configurable function fields
* Table-driven test patterns established

#### User Story 1.5.2: Establish Test Fixtures

As a tester, I want test fixtures so that I can use consistent test data.

Acceptance criteria:

* JSON fixtures for messages, responses, sessions
* Helper functions to load fixtures
* Coverage reporting configuration

---

## Epic 2: LLM Provider Implementations (Phase 2, Weeks 5-8)

### Feature 2.1: OpenAI Provider Package

Implement the `providers/openai/` package for OpenAI API integration.

Files:

* `providers/openai/client.go` - OpenAI ChatClient implementation
* `providers/openai/responses.go` - Responses API client
* `providers/openai/assistants.go` - Assistants API client
* `providers/openai/options.go` - OpenAI-specific options
* `providers/openai/doc.go` - Package documentation

#### User Story 2.1.1: Implement OpenAI Chat Completions

As a developer, I want OpenAI chat completions so that I can use GPT models.

Acceptance criteria:

* `NewClient(opts ...Option) (*Client, error)` constructor
* `WithAPIKey(key string)` option
* `WithModel(model string)` option
* `WithBaseURL(url string)` option
* Chat completions API integration via go-openai library
* Proper error handling and retry logic

#### User Story 2.1.2: Implement OpenAI Streaming

As a developer, I want streaming responses so that I can display incremental output.

Acceptance criteria:

* `GetStreamingResponse` returns channel of updates
* Proper channel closure on completion or error
* Context cancellation support
* Backpressure handling with buffered channel

#### User Story 2.1.3: Implement OpenAI Tool Calling

As a developer, I want tool calling so that agents can invoke functions.

Acceptance criteria:

* Tool definitions converted to OpenAI function format
* Tool call responses parsed and returned in updates
* Multiple parallel tool calls supported

#### User Story 2.1.4: Implement Responses API Client

As a developer, I want Responses API support so that I can use hosted tools.

Acceptance criteria:

* Responses API endpoint integration
* WebSearch hosted tool support
* Stateful conversation continuation

### Feature 2.2: Azure OpenAI Provider Package

Implement the `providers/azure/` package for Azure OpenAI integration.

Files:

* `providers/azure/client.go` - Azure OpenAI ChatClient
* `providers/azure/persistent.go` - Azure AI Foundry Persistent Agents
* `providers/azure/auth.go` - Azure authentication helpers
* `providers/azure/options.go` - Azure-specific options

#### User Story 2.2.1: Implement Azure OpenAI Client

As a developer, I want Azure OpenAI integration so that I can use Azure-hosted models.

Acceptance criteria:

* `NewClient(opts ...Option) (*Client, error)` constructor
* `WithEndpoint(endpoint string)` option
* `WithDeployment(deployment string)` option
* `WithAPIKey(key string)` option for API key auth
* All OpenAI features supported (completions, streaming, tools)

#### User Story 2.2.2: Implement Azure AD Authentication

As a developer, I want Azure AD auth so that I can use managed identities.

Acceptance criteria:

* `WithTokenCredential(cred azcore.TokenCredential)` option
* DefaultAzureCredential support
* Token refresh handling

#### User Story 2.2.3: Implement Azure AI Foundry Persistent Agents

As a developer, I want persistent agents so that I can use Azure-hosted agent infrastructure.

Acceptance criteria:

* `NewPersistentAgent(opts ...Option)` constructor
* Agent creation, listing, and deletion
* Thread management via Foundry API

### Feature 2.3: Anthropic Provider Package

Implement the `providers/anthropic/` package for Claude integration.

Files:

* `providers/anthropic/client.go` - Anthropic ChatClient
* `providers/anthropic/options.go` - Anthropic-specific options

#### User Story 2.3.1: Implement Anthropic Chat Client

As a developer, I want Anthropic integration so that I can use Claude models.

Acceptance criteria:

* `NewClient(opts ...Option) (*Client, error)` constructor
* `WithAPIKey(key string)` option
* `WithModel(model string)` option (claude-3-opus, claude-3-sonnet, etc.)
* Chat completions with proper message format conversion

#### User Story 2.3.2: Implement Anthropic Streaming

As a developer, I want streaming for Claude so that I can display incremental output.

Acceptance criteria:

* SSE-based streaming support
* Proper event parsing (message_start, content_block_delta, etc.)
* Context cancellation support

#### User Story 2.3.3: Implement Anthropic Tool Calling

As a developer, I want tool calling for Claude so that agents can invoke functions.

Acceptance criteria:

* Tool definitions converted to Anthropic format
* Tool use blocks parsed from responses
* Tool results submitted correctly

### Feature 2.4: AWS Bedrock Provider Package

Implement the `providers/bedrock/` package for AWS Bedrock integration.

Files:

* `providers/bedrock/client.go` - Bedrock ChatClient
* `providers/bedrock/auth.go` - AWS authentication
* `providers/bedrock/options.go` - Bedrock-specific options

#### User Story 2.4.1: Implement Bedrock Chat Client

As a developer, I want AWS Bedrock integration so that I can use Bedrock-hosted models.

Acceptance criteria:

* `NewClient(opts ...Option) (*Client, error)` constructor
* `WithRegion(region string)` option
* `WithModelID(modelID string)` option
* Support for Claude, Titan, and other Bedrock models
* Proper AWS credential chain resolution

#### User Story 2.4.2: Implement Bedrock Streaming

As a developer, I want streaming for Bedrock so that I can display incremental output.

Acceptance criteria:

* InvokeModelWithResponseStream API integration
* Proper chunk parsing for different model types
* Context cancellation support

### Feature 2.5: Ollama Provider Package

Implement the `providers/ollama/` package for local model support.

Files:

* `providers/ollama/client.go` - Ollama ChatClient
* `providers/ollama/options.go` - Ollama-specific options

#### User Story 2.5.1: Implement Ollama Chat Client

As a developer, I want Ollama integration so that I can use local models.

Acceptance criteria:

* `NewClient(opts ...Option) (*Client, error)` constructor
* `WithBaseURL(url string)` option (default: localhost:11434)
* `WithModel(model string)` option
* HTTP-based API integration

#### User Story 2.5.2: Implement Ollama Streaming

As a developer, I want streaming for Ollama so that I can display incremental output.

Acceptance criteria:

* NDJSON streaming response parsing
* Proper buffer handling for line-delimited JSON
* Context cancellation support

### Feature 2.6: Tool System Package

Implement the `tool/` package for function calling infrastructure.

Files:

* `tool/tool.go` - Tool interface definition
* `tool/function.go` - FunctionTool implementation
* `tool/hosted.go` - Hosted tools (WebSearch, CodeInterpreter, etc.)
* `tool/invoke.go` - Function invocation utilities
* `tool/decorator.go` - Tool decorators

#### User Story 2.6.1: Define Tool Interface

As a developer, I want a tool interface so that I can define callable functions.

Acceptance criteria:

* `Tool` interface with Name(), Description(), Parameters(), Invoke() methods
* `Result` struct with Content, IsError fields
* `Parameters()` returns JSON Schema for arguments

#### User Story 2.6.2: Implement FunctionTool

As a developer, I want FunctionTool so that I can wrap Go functions as tools.

Acceptance criteria:

* `NewFunctionTool(name, description string, fn interface{}) (*FunctionTool, error)`
* Reflection-based parameter schema generation
* `Func(fn interface{}, description string, opts ...FuncOption)` decorator
* Support for struct, primitive, and slice parameter types

#### User Story 2.6.3: Implement Hosted Tools

As a developer, I want hosted tools so that I can use provider-hosted capabilities.

Acceptance criteria:

* `HostedTool` interface extending Tool with `IsHosted() bool`
* `HostedWebSearchTool` with SearchContextSize, UserLocation
* `HostedCodeInterpreterTool` with Container, FileIDs
* `HostedFileSearchTool` with VectorStoreIDs, MaxResults
* `HostedMCPTool` with ServerURL, ServerLabel, AllowedTools

#### User Story 2.6.4: Implement Function Invocation

As a developer, I want safe function invocation so that tool calls are handled correctly.

Acceptance criteria:

* `InvokeFunction(ctx, tool, args json.RawMessage) (Result, error)`
* Proper JSON unmarshaling to function parameters
* Panic recovery with error result
* Context timeout support

### Feature 2.7: OpenTelemetry Observability Package

Implement the `observability/` package for telemetry.

Files:

* `observability/otel.go` - OpenTelemetry setup utilities
* `observability/traces.go` - Tracing helpers
* `observability/metrics.go` - Metrics definitions
* `observability/attributes.go` - Semantic conventions

#### User Story 2.7.1: Implement Tracing Setup

As a developer, I want tracing so that I can monitor agent execution.

Acceptance criteria:

* `SetupTracing(serviceName string, opts ...TracingOption) (shutdown func(), error)`
* Tracer provider configuration with OTLP exporter
* Agent run spans with proper hierarchy

#### User Story 2.7.2: Define Semantic Conventions

As a developer, I want consistent attributes so that telemetry is standardized.

Acceptance criteria:

* `AttributeAgentID`, `AttributeAgentName` constants
* `AttributeModelID`, `AttributeProviderName` constants
* `AttributeTokensInput`, `AttributeTokensOutput` constants
* OpenTelemetry semantic conventions alignment

#### User Story 2.7.3: Implement Metrics

As a developer, I want metrics so that I can track usage patterns.

Acceptance criteria:

* `agent.runs` counter with labels
* `agent.tokens.input` histogram
* `agent.tokens.output` histogram
* `agent.latency` histogram

---

## Epic 3: Advanced Agent Features (Phase 3, Weeks 9-12)

### Feature 3.1: Middleware Pipeline Package

Implement the `middleware/` package for request/response interceptors.

Files:

* `middleware/middleware.go` - Core middleware interfaces
* `middleware/agent.go` - AgentMiddleware types
* `middleware/chat.go` - ChatMiddleware types
* `middleware/function.go` - FunctionMiddleware types
* `middleware/context.go` - Middleware context types
* `middleware/pipeline.go` - Pipeline builder

#### User Story 3.1.1: Define Middleware Interfaces

As a developer, I want middleware interfaces so that I can intercept agent operations.

Acceptance criteria:

* `AgentMiddleware` interface with `Process(ctx, runCtx, next) error`
* `ChatMiddleware` interface with `Process(ctx, chatCtx, next) error`
* `FunctionMiddleware` interface with `Process(ctx, fnCtx, next) error`
* Next function types for chaining

#### User Story 3.1.2: Implement Middleware Contexts

As a developer, I want middleware contexts so that I can access and modify request state.

Acceptance criteria:

* `AgentRunContext` with Agent, Messages, Options, Response, Terminate fields
* `ChatContext` with Client, Messages, Options, Response, Metadata fields
* `FunctionContext` with Function, Arguments, Result, Metadata fields

#### User Story 3.1.3: Implement Pipeline Builder

As a developer, I want a pipeline builder so that I can compose middleware chains.

Acceptance criteria:

* `NewAgentPipeline(middlewares ...AgentMiddleware) AgentPipeline`
* `Execute(ctx, agent, messages, opts) (*Response, error)` method
* Proper ordering (first added = outermost)
* Error propagation through chain

#### User Story 3.1.4: Implement Common Middleware

As a developer, I want common middleware so that I have ready-to-use interceptors.

Acceptance criteria:

* `LoggingMiddleware` for request/response logging
* `RetryMiddleware` with configurable backoff
* `TimeoutMiddleware` for deadline enforcement
* `MetricsMiddleware` for telemetry collection

### Feature 3.2: Memory and Context Providers Package

Implement the `memory/` package for context and history management.

Files:

* `memory/context.go` - Context and ContextProvider interfaces
* `memory/history.go` - ChatHistoryProvider interface
* `memory/store.go` - Message store interfaces
* `memory/mem0/client.go` - Mem0 integration
* `memory/redis/store.go` - Redis message store
* `memory/cosmos/store.go` - Cosmos DB store

#### User Story 3.2.1: Define Context Provider Interface

As a developer, I want context providers so that I can inject dynamic context.

Acceptance criteria:

* `Context` struct with Content, Metadata fields
* `ContextProvider` interface with `GetContext(ctx, query) ([]Context, error)`
* Priority ordering for multiple providers

#### User Story 3.2.2: Define History Provider Interface

As a developer, I want history providers so that I can manage conversation history.

Acceptance criteria:

* `ChatHistoryProvider` interface with GetHistory, AddMessage, Clear methods
* `GetHistory(ctx, sessionID) ([]Message, error)` signature
* `AddMessage(ctx, sessionID, message) error` signature

#### User Story 3.2.3: Implement Mem0 Memory Provider

As a developer, I want Mem0 integration so that I can use semantic memory.

Acceptance criteria:

* `NewMem0Client(opts ...Option) (*Client, error)`
* `WithAPIKey(key string)` option
* `Add(ctx, content, metadata) error` method
* `Search(ctx, query) ([]Memory, error)` method

#### User Story 3.2.4: Implement Redis Message Store

As a developer, I want Redis storage so that I can persist messages across instances.

Acceptance criteria:

* `NewRedisStore(opts ...Option) (*Store, error)`
* `WithAddress(addr string)` option
* FIFO message list per session
* TTL support for session expiration

### Feature 3.3: Thread Management Package

Implement the `thread/` package for conversation threading.

Files:

* `thread/thread.go` - AgentThread implementation
* `thread/store.go` - ChatMessageStore interface
* `thread/inmemory.go` - InMemory implementations

#### User Story 3.3.1: Implement AgentThread

As a developer, I want AgentThread so that I can manage conversation state.

Acceptance criteria:

* `AgentThread` struct with ID, agent reference, message store
* `Run(ctx, message, opts) (*Response, error)` method appends to history
* `RunStream(ctx, message, opts) (<-chan ResponseUpdate, error)` streaming variant
* `GetMessages(ctx) ([]Message, error)` method

#### User Story 3.3.2: Implement InMemory Store

As a developer, I want in-memory storage so that I can test without external deps.

Acceptance criteria:

* `InMemoryChatMessageStore` implementation
* Thread-safe access with mutex
* `NewInMemoryChatMessageStore()` constructor

### Feature 3.4: ChatClientAgent Implementation Package

Implement the `chatagent/` package as the primary agent implementation.

Files:

* `chatagent/agent.go` - ChatClientAgent implementation
* `chatagent/options.go` - Agent options
* `chatagent/session.go` - Agent session implementation
* `chatagent/builder.go` - Fluent builder pattern

#### User Story 3.4.1: Implement ChatClientAgent

As a developer, I want ChatClientAgent so that I can create agents from chat clients.

Acceptance criteria:

* `New(client chat.Client, opts ...Option) *Agent` constructor
* Implements `agent.Agent` interface fully
* Automatic tool invocation loop
* Session state management

#### User Story 3.4.2: Implement Agent Options

As a developer, I want agent options so that I can configure agent behavior.

Acceptance criteria:

* `WithName(name string)` option
* `WithDescription(description string)` option
* `WithInstructions(instructions string)` option
* `WithTools(tools ...Tool)` option
* `WithMiddleware(middleware ...AgentMiddleware)` option
* `WithMaxTurns(turns int)` option

#### User Story 3.4.3: Implement Agent Session

As a developer, I want agent sessions so that I can maintain conversation context.

Acceptance criteria:

* `ChatClientAgentSession` implementing `agent.Session`
* Conversation history tracking
* Serialization/deserialization for persistence

#### User Story 3.4.4: Implement Agent Builder

As a developer, I want a builder pattern so that I can construct agents fluently.

Acceptance criteria:

* `NewBuilder(client chat.Client) *Builder`
* Method chaining: `builder.Name(...).Instructions(...).Tools(...).Build()`
* Validation on Build()

### Feature 3.5: Vector Search and RAG Integration

Implement vector search capabilities for RAG patterns.

Files:

* `providers/azuresearch/client.go` - Azure AI Search client
* `providers/azuresearch/index.go` - Index management
* `providers/azuresearch/options.go` - Search options

#### User Story 3.5.1: Implement Azure AI Search Client

As a developer, I want Azure AI Search integration so that I can perform semantic search.

Acceptance criteria:

* `NewClient(opts ...Option) (*Client, error)` constructor
* `WithEndpoint(endpoint string)` option
* `WithAPIKey(key string)` option
* `Search(ctx, query, opts) (*SearchResults, error)` method

#### User Story 3.5.2: Implement Index Management

As a developer, I want index management so that I can configure search indexes.

Acceptance criteria:

* `CreateIndex(ctx, definition) error` method
* `DeleteIndex(ctx, name) error` method
* `ListIndexes(ctx) ([]IndexInfo, error)` method

#### User Story 3.5.3: Implement Hybrid Search

As a developer, I want hybrid search so that I can combine keyword and semantic search.

Acceptance criteria:

* `WithSemanticConfiguration(config string)` option
* `WithVectorField(field string)` option
* Combined ranking of keyword and vector results

---

## Epic 4: Workflow Orchestration and Protocols (Phase 4, Weeks 13-18)

### Feature 4.1: Workflow Engine Package

Implement the `workflow/` package for DAG-based orchestration.

Files:

* `workflow/workflow.go` - Workflow definition
* `workflow/builder.go` - WorkflowBuilder
* `workflow/executor.go` - Executor interface
* `workflow/edge.go` - Edge and EdgeGroup types
* `workflow/checkpoint.go` - Checkpointing support
* `workflow/runner.go` - WorkflowRunner
* `workflow/events.go` - Workflow events

#### User Story 4.1.1: Define Workflow Structure

As a developer, I want workflow definitions so that I can orchestrate multiple agents.

Acceptance criteria:

* `Workflow` struct with Nodes, Edges, StartNode, EndNode
* `Node` struct with ID, Executor, Metadata
* `Edge` struct with From, To, Condition
* Serializable to/from JSON/YAML

#### User Story 4.1.2: Implement WorkflowBuilder

As a developer, I want a workflow builder so that I can construct workflows programmatically.

Acceptance criteria:

* `NewBuilder() *Builder` constructor
* `AddNode(id string, executor Executor) *Builder`
* `AddEdge(from, to string, condition EdgeCondition) *Builder`
* `AddConditionalEdges(from string, router func(state) string) *Builder`
* `Build() (*Workflow, error)` with validation

#### User Story 4.1.3: Implement Executor Interface

As a developer, I want executors so that I can define node behaviors.

Acceptance criteria:

* `Executor` interface with `Execute(ctx, state) (state, error)`
* `AgentExecutor` wrapping an Agent
* `FunctionExecutor` wrapping a function
* State passed through workflow

#### User Story 4.1.4: Implement Checkpointing

As a developer, I want checkpointing so that I can resume workflows.

Acceptance criteria:

* `Checkpoint` struct with WorkflowID, NodeID, State, Timestamp
* `CheckpointStore` interface with Save, Load, Delete methods
* `InMemoryCheckpointStore` implementation

#### User Story 4.1.5: Implement WorkflowRunner

As a developer, I want a runner so that I can execute workflows.

Acceptance criteria:

* `NewRunner(workflow *Workflow, opts ...RunnerOption) *Runner`
* `Run(ctx, initialState) (finalState, error)` method
* `RunStream(ctx, initialState) (<-chan Event, error)` streaming variant
* Parallel execution for independent branches
* FanOut/FanIn patterns

#### User Story 4.1.6: Implement Workflow Events

As a developer, I want workflow events so that I can monitor execution.

Acceptance criteria:

* `Event` struct with Type, NodeID, State, Timestamp, Error
* Event types: NodeStarted, NodeCompleted, NodeFailed, WorkflowCompleted
* Event channel for streaming consumers

### Feature 4.2: A2A Protocol Implementation

Implement the `protocol/a2a/` package for agent-to-agent communication.

Files:

* `protocol/a2a/agent.go` - A2A Agent wrapper
* `protocol/a2a/client.go` - A2A Client for remote agents
* `protocol/a2a/types.go` - A2A Protocol types
* `protocol/a2a/server.go` - A2A HTTP Server

#### User Story 4.2.1: Implement A2A Types

As a developer, I want A2A types so that I can exchange messages per protocol.

Acceptance criteria:

* `AgentCard` struct with capabilities declaration
* `Task` struct with ID, SessionID, Status, History
* `TaskStatus` enum: Pending, Running, Completed, Failed, Cancelled
* `Message` struct aligned with A2A specification

#### User Story 4.2.2: Implement A2A Client

As a developer, I want an A2A client so that I can call remote agents.

Acceptance criteria:

* `NewClient(baseURL string, opts ...Option) *Client`
* `GetAgentCard(ctx) (*AgentCard, error)` method
* `CreateTask(ctx, request) (*Task, error)` method
* `GetTask(ctx, taskID) (*Task, error)` method
* `SendMessage(ctx, taskID, message) (*Task, error)` method
* `CancelTask(ctx, taskID) error` method

#### User Story 4.2.3: Implement A2A Server

As a developer, I want an A2A server so that I can expose agents over HTTP.

Acceptance criteria:

* `NewServer(agent Agent, opts ...Option) *Server`
* `GET /.well-known/agent.json` endpoint for AgentCard
* `POST /tasks` endpoint for task creation
* `GET /tasks/{id}` endpoint for task status
* `POST /tasks/{id}/messages` endpoint for messages
* `DELETE /tasks/{id}` endpoint for cancellation
* Streaming response support

#### User Story 4.2.4: Implement A2A Agent Wrapper

As a developer, I want A2A agent wrapping so that I can use remote agents locally.

Acceptance criteria:

* `NewA2AAgent(client *Client) *A2AAgent`
* Implements `agent.Agent` interface
* Translates Run/RunStream to A2A calls

### Feature 4.3: AG-UI Protocol Implementation

Implement the `protocol/agui/` package for agent-UI communication.

Files:

* `protocol/agui/client.go` - AG-UI Client
* `protocol/agui/server.go` - AG-UI Server
* `protocol/agui/types.go` - AG-UI types

#### User Story 4.3.1: Implement AG-UI Types

As a developer, I want AG-UI types so that I can stream UI events.

Acceptance criteria:

* `Event` struct with Type, Data, Timestamp
* Event types: TextMessageStart, TextMessageContent, TextMessageEnd, ToolCallStart, ToolCallEnd, etc.
* `RunAgentInput` struct for run requests

#### User Story 4.3.2: Implement AG-UI Server

As a developer, I want an AG-UI server so that I can stream events to UIs.

Acceptance criteria:

* `NewServer(agent Agent, opts ...Option) *Server`
* `POST /` endpoint for run requests
* Server-Sent Events (SSE) response format
* Proper event formatting per AG-UI spec

#### User Story 4.3.3: Implement AG-UI Client

As a developer, I want an AG-UI client so that I can consume agent streams.

Acceptance criteria:

* `NewClient(baseURL string) *Client`
* `Run(ctx, input) (<-chan Event, error)` method
* SSE parsing and event deserialization

### Feature 4.4: Group Chat Orchestration

Implement the `workflow/groupchat/` package for multi-agent chat.

Files:

* `workflow/groupchat/manager.go` - GroupChatManager
* `workflow/groupchat/roundrobin.go` - RoundRobin selector
* `workflow/groupchat/selector.go` - Selector interface

#### User Story 4.4.1: Implement GroupChatManager

As a developer, I want group chat so that I can orchestrate multi-agent conversations.

Acceptance criteria:

* `NewManager(agents []Agent, opts ...Option) *Manager`
* `WithSelector(selector Selector)` option
* `Run(ctx, initialMessage) (*Transcript, error)` method
* `RunStream(ctx, initialMessage) (<-chan Event, error)` streaming variant

#### User Story 4.4.2: Implement Selector Interface

As a developer, I want selectors so that I can customize speaker selection.

Acceptance criteria:

* `Selector` interface with `SelectNext(transcript) (Agent, error)`
* `RoundRobinSelector` implementation
* `RandomSelector` implementation
* `LLMSelector` using an agent to decide

#### User Story 4.4.3: Implement Transcript

As a developer, I want transcripts so that I can review group chat history.

Acceptance criteria:

* `Transcript` struct with Messages, Participants
* `TranscriptEntry` with Speaker, Message, Timestamp
* Serializable to JSON

---

## Epic 5: Enterprise Production Features (Phase 5, Weeks 19-24)

### Feature 5.1: Durable Agents with Temporal.io

Implement the `durable/` package for long-running orchestration.

Files:

* `durable/agent.go` - DurableAIAgent implementation
* `durable/session.go` - DurableAgentSession
* `durable/entity.go` - Agent entity state
* `durable/temporal/worker.go` - Temporal worker setup

#### User Story 5.1.1: Implement Temporal Integration

As a developer, I want Temporal integration so that I can run durable workflows.

Acceptance criteria:

* `NewTemporalWorker(opts ...Option) (*Worker, error)`
* `WithNamespace(ns string)` option
* `WithTaskQueue(queue string)` option
* Workflow and activity registration

#### User Story 5.1.2: Implement Durable Agent

As a developer, I want durable agents so that they survive process restarts.

Acceptance criteria:

* `DurableAIAgent` wrapping a base agent
* State persisted via Temporal workflow
* Resume capability from persisted state
* Human-in-the-loop wait states

#### User Story 5.1.3: Implement Durable Session

As a developer, I want durable sessions so that conversation state persists.

Acceptance criteria:

* `DurableAgentSession` implementing `agent.Session`
* Session state stored in Temporal entity
* Automatic state synchronization

### Feature 5.2: Declarative Agent Definitions

Implement the `declarative/` package for YAML-based agents.

Files:

* `declarative/loader.go` - YAML loader
* `declarative/models.go` - Schema models
* `declarative/factory.go` - Agent factory

#### User Story 5.2.1: Define YAML Schema

As a developer, I want a YAML schema so that I can define agents declaratively.

Acceptance criteria:

* Agent definition schema with name, description, instructions
* Provider configuration (openai, azure, anthropic, etc.)
* Tool definitions in YAML
* Middleware configuration

#### User Story 5.2.2: Implement YAML Loader

As a developer, I want a YAML loader so that I can parse agent definitions.

Acceptance criteria:

* `LoadFromFile(path string) (*AgentDefinition, error)`
* `LoadFromReader(r io.Reader) (*AgentDefinition, error)`
* `LoadFromString(content string) (*AgentDefinition, error)`
* Schema validation with helpful errors

#### User Story 5.2.3: Implement Agent Factory

As a developer, I want an agent factory so that I can instantiate agents from definitions.

Acceptance criteria:

* `NewFactory(opts ...Option) *Factory`
* `RegisterProvider(name string, builder ProviderBuilder)` method
* `RegisterToolFactory(name string, factory ToolFactory)` method
* `CreateAgent(definition *AgentDefinition) (Agent, error)` method

### Feature 5.3: HTTP and gRPC Hosting

Implement the `hosting/` package for server infrastructure.

Files:

* `hosting/http/handler.go` - HTTP handlers
* `hosting/http/router.go` - HTTP router setup
* `hosting/grpc/server.go` - gRPC server
* `hosting/grpc/proto/` - Protocol buffer definitions

#### User Story 5.3.1: Implement HTTP Handler

As a developer, I want HTTP handlers so that I can expose agents over REST.

Acceptance criteria:

* `NewHandler(agent Agent, opts ...Option) http.Handler`
* `POST /run` endpoint for synchronous runs
* `POST /stream` endpoint for streaming (SSE)
* `GET /health` endpoint for health checks
* Request/response JSON schemas

#### User Story 5.3.2: Implement HTTP Router

As a developer, I want router setup so that I can configure the HTTP server.

Acceptance criteria:

* `NewRouter(opts ...Option) *Router`
* `Mount(path string, handler http.Handler)` method
* CORS configuration
* Middleware integration

#### User Story 5.3.3: Implement gRPC Server

As a developer, I want gRPC support so that I can use efficient binary protocols.

Acceptance criteria:

* Protocol buffer service definition
* `NewGRPCServer(agent Agent, opts ...Option) *grpc.Server`
* Streaming RPC for agent runs
* Health check service

#### User Story 5.3.4: Implement Graceful Shutdown

As a developer, I want graceful shutdown so that in-flight requests complete.

Acceptance criteria:

* Context-based shutdown signaling
* Configurable shutdown timeout
* Connection draining

### Feature 5.4: MCP Integration

Implement MCP (Model Context Protocol) support.

Files:

* `mcp/client.go` - MCP client
* `mcp/server.go` - MCP server
* `mcp/types.go` - MCP types
* `mcp/bridge.go` - Tool bridging

#### User Story 5.4.1: Implement MCP Client

As a developer, I want an MCP client so that I can connect to MCP servers.

Acceptance criteria:

* `NewClient(transport Transport) (*Client, error)`
* `Initialize(ctx) error` method
* `ListTools(ctx) ([]ToolInfo, error)` method
* `CallTool(ctx, name, args) (Result, error)` method
* `ListResources(ctx) ([]ResourceInfo, error)` method

#### User Story 5.4.2: Implement MCP Server

As a developer, I want an MCP server so that I can expose tools via MCP.

Acceptance criteria:

* `NewServer(opts ...Option) *Server`
* `AddTool(tool Tool)` method
* `AddResource(resource Resource)` method
* JSON-RPC 2.0 protocol handling

#### User Story 5.4.3: Implement Tool Bridging

As a developer, I want tool bridging so that MCP tools work with agents.

Acceptance criteria:

* `MCPToolBridge` adapting MCP tools to Tool interface
* Automatic tool discovery on client connect
* Tool invocation forwarding

### Feature 5.5: Enterprise Integrations

Implement enterprise platform integrations.

Files:

* `providers/copilotstudio/client.go` - Copilot Studio integration
* `providers/githubcopilot/client.go` - GitHub Copilot SDK
* `governance/purview/client.go` - Microsoft Purview
* `devui/server.go` - Developer debugging UI
* `providers/foundrylocal/client.go` - Local model execution

#### User Story 5.5.1: Implement Copilot Studio Integration

As a developer, I want Copilot Studio integration so that I can use Microsoft Copilots.

Acceptance criteria:

* `NewCopilotStudioClient(opts ...Option) (*Client, error)`
* `WithTenantID(tenantID string)` option
* `WithBotID(botID string)` option
* Conversation management API

#### User Story 5.5.2: Implement GitHub Copilot SDK

As a developer, I want GitHub Copilot SDK so that I can build Copilot extensions.

Acceptance criteria:

* `NewGitHubCopilotClient(opts ...Option) (*Client, error)`
* Chat completion interface
* Extension manifest generation

#### User Story 5.5.3: Implement Microsoft Purview Integration

As a developer, I want Purview integration so that I can enforce data governance.

Acceptance criteria:

* `NewPurviewClient(opts ...Option) (*Client, error)`
* Policy evaluation for data classification
* Audit logging for data access

#### User Story 5.5.4: Implement Developer Debugging UI

As a developer, I want a debugging UI so that I can inspect agent behavior.

Acceptance criteria:

* `NewDevUIServer(opts ...Option) *Server`
* Agent inspector showing conversation flow
* Request/response tracing visualization
* Local model execution support with Foundry Local

### Feature 5.6: Protocol-Specific Hosting

Implement dedicated hosting for A2A, AG-UI, and OpenAI-compatible APIs.

Files:

* `hosting/a2a/handler.go` - A2A protocol HTTP handler
* `hosting/a2a/router.go` - A2A endpoint routing
* `hosting/agui/handler.go` - AG-UI SSE handler
* `hosting/agui/sse.go` - Server-sent events support
* `hosting/openaicompat/handler.go` - OpenAI-compatible API handler
* `hosting/openaicompat/models.go` - OpenAI API models
* `hosting/azurefunctions/trigger.go` - Azure Functions trigger

#### User Story 5.6.1: Implement A2A Hosting

As a developer, I want A2A hosting so that I can expose agents via A2A protocol.

Acceptance criteria:

* `NewA2AHandler(agent Agent, opts ...Option) http.Handler`
* Full A2A protocol endpoint implementation
* Task lifecycle management
* Streaming message support

#### User Story 5.6.2: Implement AG-UI Hosting

As a developer, I want AG-UI hosting so that I can stream to UI clients.

Acceptance criteria:

* `NewAGUIHandler(agent Agent, opts ...Option) http.Handler`
* SSE response streaming
* AG-UI event format compliance

#### User Story 5.6.3: Implement OpenAI-Compatible API

As a developer, I want OpenAI-compatible APIs so that existing clients can connect.

Acceptance criteria:

* `NewOpenAICompatHandler(agent Agent, opts ...Option) http.Handler`
* `/v1/chat/completions` endpoint
* Request/response format matching OpenAI API
* Streaming support with SSE

#### User Story 5.6.4: Implement Azure Functions Hosting

As a developer, I want Azure Functions hosting so that I can run serverless agents.

Acceptance criteria:

* HTTP trigger for agent invocation
* Durable Functions integration for long-running tasks
* Azure bindings for configuration

### Feature 5.7: Production Hardening

Implement production-ready resilience patterns.

Files:

* `resilience/ratelimit.go` - Rate limiting
* `resilience/circuitbreaker.go` - Circuit breaker
* `resilience/retry.go` - Retry policies
* `resilience/pool.go` - Connection pooling

#### User Story 5.7.1: Implement Rate Limiting

As a developer, I want rate limiting so that I can protect against overload.

Acceptance criteria:

* `NewRateLimiter(rps float64, burst int) *RateLimiter`
* Token bucket algorithm
* Per-client rate limiting option
* Rate limit exceeded error type

#### User Story 5.7.2: Implement Circuit Breaker

As a developer, I want circuit breakers so that I can handle downstream failures.

Acceptance criteria:

* `NewCircuitBreaker(opts ...Option) *CircuitBreaker`
* States: Closed, Open, HalfOpen
* Configurable failure threshold and reset timeout
* Circuit open error type

#### User Story 5.7.3: Implement Retry Policies

As a developer, I want retry policies so that transient failures are handled.

Acceptance criteria:

* `NewRetryPolicy(opts ...Option) *RetryPolicy`
* `WithMaxAttempts(n int)` option
* `WithBackoff(strategy BackoffStrategy)` option
* Exponential and jittered backoff strategies

#### User Story 5.7.4: Implement Connection Pooling

As a developer, I want connection pooling so that resources are used efficiently.

Acceptance criteria:

* HTTP client connection pool configuration
* gRPC connection pool management
* Pool size and idle timeout configuration

---

## Dependencies

* Go 1.22+ with generic type constraints
* OpenTelemetry Go SDK v1.x
* github.com/sashabaranov/go-openai for OpenAI
* github.com/liushuangls/go-anthropic for Anthropic
* github.com/Azure/azure-sdk-for-go for Azure auth
* go.temporal.io/sdk for durable workflows
* gopkg.in/yaml.v3 for declarative agents
* google.golang.org/grpc for gRPC hosting
* github.com/redis/go-redis/v9 for Redis storage

## Success Criteria

* 100% of Phase 1-4 features implemented with API parity
* 90%+ test coverage across all packages
* All public APIs documented with godoc
* Integration tests pass for all LLM providers
* Benchmark overhead < 10ms for simple runs
* Benchmark streaming overhead < 1ms per chunk
* Memory per-agent < 1KB base
* Zero CVEs on release
