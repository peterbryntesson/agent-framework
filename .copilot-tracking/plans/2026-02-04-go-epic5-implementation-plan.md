<!-- markdownlint-disable-file -->
# Implementation Plan: Go Epic 5 - Enterprise Production Features

**Created:** 2026-02-04  
**Source Research:** [2026-02-04-go-epic5-enterprise-production-research.md](../research/2026-02-04-go-epic5-enterprise-production-research.md)  
**Status:** Ready for Execution

## Executive Summary

This plan implements 7 enterprise production features for the Go Agent Framework, prioritized for maximum value with minimum dependencies. The implementation follows existing Go idioms from the codebase and maintains API parity with .NET and Python implementations.

**Total Estimated Effort:** 18-22 developer weeks  
**Features:** 7 major features across 6 new packages

## Implementation Priority Order

| Priority | Feature | Package | Effort | Dependencies |
|----------|---------|---------|--------|--------------|
| P0-1 | 5.7 Production Hardening | `go/resilience/` | 2 weeks | None |
| P0-2 | 5.3 HTTP/gRPC Hosting | `go/hosting/openai/` | 2 weeks | Existing `hosting.SessionStore` |
| P1-1 | 5.2 Declarative Agents | `go/declarative/` | 3 weeks | Provider packages |
| P1-2 | 5.4 MCP Integration | `go/mcp/` | 3 weeks | External `mcp-go` library |
| P2-1 | 5.1 Durable Agents | `go/durable/` | 4 weeks | Temporal.io SDK |
| P2-2 | 5.5 Enterprise: DevUI | `go/devui/` | 2 weeks | OpenAI hosting |
| P2-3 | 5.5 Enterprise: Purview | `go/purview/` | 2 weeks | Azure SDK |

---

## Phase 1: Foundation (Weeks 1-2) ✅ COMPLETED

### Feature 5.7: Production Hardening ✅

**Objective:** Implement resilience patterns as reusable middleware for chat clients and agents.

**Status:** ✅ Completed on 2026-02-04  
**Changes Log:** [2026-02-04-go-epic5-phase1-changes.md](../changes/2026-02-04-go-epic5-phase1-changes.md)

#### Task 5.7.1: Create Resilience Package Structure ✅

**Files Created:**

```
go/resilience/
├── doc.go           ✅
├── ratelimit.go     ✅
├── ratelimit_test.go ✅
├── circuitbreaker.go ✅
├── circuitbreaker_test.go ✅
├── retry.go         ✅
├── retry_test.go    ✅
├── pool.go          ✅
├── pool_test.go     ✅
└── options.go       ✅
```

**Subtasks:**

| ID | Task | File | Acceptance Criteria | Status |
|----|------|------|---------------------|--------|
| 5.7.1.1 | Create package documentation | `doc.go` | Package overview with usage examples | ✅ |
| 5.7.1.2 | Create options types | `options.go` | Common configuration structs | ✅ |

#### Task 5.7.2: Implement Rate Limiter ✅

**File:** `go/resilience/ratelimit.go`

**Implementation Details:**

```go
package resilience

import (
    "context"
    "sync"
    "golang.org/x/time/rate"
    "github.com/microsoft/agent-framework-go/chat"
)

// RateLimiter provides token bucket rate limiting.
type RateLimiter struct {
    global    *rate.Limiter
    perClient map[string]*rate.Limiter
    rps       float64
    burst     int
    mu        sync.RWMutex
}

// RateLimiterOption configures a RateLimiter.
type RateLimiterOption func(*RateLimiter)

// NewRateLimiter creates a new rate limiter with the specified requests per second and burst size.
func NewRateLimiter(rps float64, burst int, opts ...RateLimiterOption) *RateLimiter

// Wait blocks until the rate limiter allows an event or the context is canceled.
func (r *RateLimiter) Wait(ctx context.Context) error

// WaitForClient blocks until the per-client rate limiter allows an event.
func (r *RateLimiter) WaitForClient(ctx context.Context, clientID string) error

// Middleware returns a chat.Middleware that applies rate limiting.
func (r *RateLimiter) Middleware() chat.Middleware
```

**Subtasks:**

| ID | Task | Acceptance Criteria | Status |
|----|------|---------------------|--------|
| 5.7.2.1 | Implement `NewRateLimiter` | Token bucket with global limiter | ✅ |
| 5.7.2.2 | Implement `Wait` | Blocks until allowed, respects context | ✅ |
| 5.7.2.3 | Implement `WaitForClient` | Per-client limiting with lazy initialization | ✅ |
| 5.7.2.4 | Implement `Middleware` | Returns `chat.Middleware` wrapper | ✅ |
| 5.7.2.5 | Write unit tests | 90%+ coverage, test context cancellation | ✅ |

**Dependencies:**
- Add `golang.org/x/time` to `go.mod` (direct dependency) ✅

#### Task 5.7.3: Implement Circuit Breaker ✅

**File:** `go/resilience/circuitbreaker.go`

**Implementation Details:**

```go
package resilience

import (
    "github.com/sony/gobreaker"
    "github.com/microsoft/agent-framework-go/chat"
)

// CircuitBreaker wraps gobreaker with chat middleware integration.
type CircuitBreaker struct {
    cb *gobreaker.CircuitBreaker
}

// CircuitBreakerOption configures a CircuitBreaker.
type CircuitBreakerOption func(*gobreaker.Settings)

// NewCircuitBreaker creates a circuit breaker with the given options.
func NewCircuitBreaker(name string, opts ...CircuitBreakerOption) *CircuitBreaker

// WithMaxRequests sets the max requests in half-open state.
func WithMaxRequests(n uint32) CircuitBreakerOption

// WithTimeout sets the timeout for open state before half-open.
func WithTimeout(d time.Duration) CircuitBreakerOption

// WithReadyToTrip sets the function that determines when to trip.
func WithReadyToTrip(f func(gobreaker.Counts) bool) CircuitBreakerOption

// Execute runs the function with circuit breaker protection.
func (cb *CircuitBreaker) Execute(fn func() error) error

// Middleware returns a chat.Middleware that applies circuit breaking.
func (cb *CircuitBreaker) Middleware() chat.Middleware

// State returns the current circuit breaker state.
func (cb *CircuitBreaker) State() gobreaker.State
```

**Subtasks:**

| ID | Task | Acceptance Criteria | Status |
|----|------|---------------------|--------|
| 5.7.3.1 | Implement `NewCircuitBreaker` | Configurable settings via options | ✅ |
| 5.7.3.2 | Implement option functions | MaxRequests, Timeout, ReadyToTrip | ✅ |
| 5.7.3.3 | Implement `Execute` | Wraps gobreaker.Execute | ✅ |
| 5.7.3.4 | Implement `Middleware` | Returns `chat.Middleware` wrapper | ✅ |
| 5.7.3.5 | Write unit tests | Test all states: closed, open, half-open | ✅ |

**Dependencies:**
- Add `github.com/sony/gobreaker/v2` to `go.mod` ✅

#### Task 5.7.4: Implement Retry Policy ✅

**File:** `go/resilience/retry.go`

**Implementation Details:**

```go
package resilience

import (
    "context"
    "time"
    "github.com/cenkalti/backoff/v5"
    "github.com/microsoft/agent-framework-go/chat"
)

// RetryPolicy defines retry behavior with backoff.
type RetryPolicy struct {
    maxAttempts int
    backoff     backoff.BackOff
    isRetryable func(error) bool
}

// RetryOption configures a RetryPolicy.
type RetryOption func(*RetryPolicy)

// NewRetryPolicy creates a retry policy with the given options.
func NewRetryPolicy(opts ...RetryOption) *RetryPolicy

// WithMaxAttempts sets the maximum number of retry attempts.
func WithMaxAttempts(n int) RetryOption

// WithExponentialBackoff configures exponential backoff.
func WithExponentialBackoff(initial, max time.Duration) RetryOption

// WithLinearBackoff configures linear backoff.
func WithLinearBackoff(interval time.Duration) RetryOption

// WithJitter adds randomization to backoff intervals.
func WithJitter(factor float64) RetryOption

// WithRetryableCheck sets the function to determine if an error is retryable.
func WithRetryableCheck(f func(error) bool) RetryOption

// Execute runs the function with retry logic.
func (p *RetryPolicy) Execute(ctx context.Context, fn func() error) error

// Middleware returns a chat.Middleware that applies retry logic.
func (p *RetryPolicy) Middleware() chat.Middleware
```

**Subtasks:**

| ID | Task | Acceptance Criteria | Status |
|----|------|---------------------|--------|
| 5.7.4.1 | Implement `NewRetryPolicy` | Default to exponential backoff | ✅ |
| 5.7.4.2 | Implement backoff options | Exponential, linear, jitter | ✅ |
| 5.7.4.3 | Implement `WithRetryableCheck` | Use existing `chat.IsRetryable` as default | ✅ |
| 5.7.4.4 | Implement `Execute` | Respect context cancellation | ✅ |
| 5.7.4.5 | Implement `Middleware` | Returns `chat.Middleware` wrapper | ✅ |
| 5.7.4.6 | Write unit tests | Test backoff timing, max attempts | ✅ |

**Dependencies:**
- Promote `github.com/cenkalti/backoff/v5` to direct dependency ✅

#### Task 5.7.5: Implement HTTP Client Pool Configuration ✅

**File:** `go/resilience/pool.go`

**Implementation Details:**

```go
package resilience

import (
    "net/http"
    "time"
)

// PoolConfig configures HTTP client connection pooling.
type PoolConfig struct {
    MaxIdleConns        int
    MaxIdleConnsPerHost int
    MaxConnsPerHost     int
    IdleConnTimeout     time.Duration
    TLSHandshakeTimeout time.Duration
    ResponseHeaderTimeout time.Duration
}

// DefaultPoolConfig returns sensible defaults for production use.
func DefaultPoolConfig() PoolConfig

// NewHTTPClient creates an http.Client with optimized connection pooling.
func NewHTTPClient(config PoolConfig) *http.Client

// NewHTTPTransport creates an http.Transport with optimized settings.
func NewHTTPTransport(config PoolConfig) *http.Transport
```

**Subtasks:**

| ID | Task | Acceptance Criteria | Status |
|----|------|---------------------|--------|
| 5.7.5.1 | Define `PoolConfig` | All relevant transport settings | ✅ |
| 5.7.5.2 | Implement `DefaultPoolConfig` | Production-ready defaults | ✅ |
| 5.7.5.3 | Implement `NewHTTPClient` | Configured client with transport | ✅ |
| 5.7.5.4 | Write unit tests | Verify configuration applied | ✅ |

---

## Phase 2: HTTP Hosting (Weeks 3-4) ✅ COMPLETED

### Feature 5.3: HTTP and gRPC Hosting ✅

**Objective:** Implement OpenAI-compatible HTTP endpoints for agent hosting.

**Status:** ✅ Completed on 2026-02-04  
**Changes Log:** [2026-02-04-go-epic5-phase2-changes.md](../changes/2026-02-04-go-epic5-phase2-changes.md)

#### Task 5.3.1: Create OpenAI Hosting Package Structure ✅

**Files Created:**

```
go/hosting/openai/
├── doc.go           ✅
├── handler.go       ✅
├── handler_test.go  ✅
├── completions.go   ✅
├── streaming.go     ✅
├── streaming_test.go ✅
├── models.go        ✅
├── models_test.go   ✅
└── options.go       ✅
```

**Subtasks:**

| ID | Task | File | Acceptance Criteria | Status |
|----|------|------|---------------------|--------|
| 5.3.1.1 | Create package documentation | `doc.go` | Overview with endpoint examples | ✅ |
| 5.3.1.2 | Define request/response models | `models.go` | OpenAI API-compatible structs | ✅ |
| 5.3.1.3 | Define handler options | `options.go` | Functional options pattern | ✅ |

#### Task 5.3.2: Implement OpenAI Request/Response Models ✅

**File:** `go/hosting/openai/models.go`

**Implementation Details:**

```go
package openai

// ChatCompletionRequest represents the OpenAI chat completions request.
type ChatCompletionRequest struct {
    Model       string                   `json:"model"`
    Messages    []ChatCompletionMessage  `json:"messages"`
    Temperature *float64                 `json:"temperature,omitempty"`
    MaxTokens   *int                     `json:"max_tokens,omitempty"`
    Stream      bool                     `json:"stream,omitempty"`
    Tools       []Tool                   `json:"tools,omitempty"`
    ToolChoice  any                      `json:"tool_choice,omitempty"`
}

// ChatCompletionMessage represents a message in the conversation.
type ChatCompletionMessage struct {
    Role       string     `json:"role"`
    Content    any        `json:"content"` // string or []ContentPart
    Name       string     `json:"name,omitempty"`
    ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
    ToolCallID string     `json:"tool_call_id,omitempty"`
}

// ChatCompletionResponse represents the API response.
type ChatCompletionResponse struct {
    ID      string                   `json:"id"`
    Object  string                   `json:"object"`
    Created int64                    `json:"created"`
    Model   string                   `json:"model"`
    Choices []ChatCompletionChoice   `json:"choices"`
    Usage   *Usage                   `json:"usage,omitempty"`
}

// ChatCompletionChunk represents a streaming response chunk.
type ChatCompletionChunk struct {
    ID      string                 `json:"id"`
    Object  string                 `json:"object"`
    Created int64                  `json:"created"`
    Model   string                 `json:"model"`
    Choices []ChatCompletionDelta  `json:"choices"`
}

// Additional types: Tool, ToolCall, Usage, ContentPart, etc.
```

**Subtasks:**

| ID | Task | Acceptance Criteria | Status |
|----|------|---------------------|--------|
| 5.3.2.1 | Define request types | Match OpenAI API spec | ✅ |
| 5.3.2.2 | Define response types | Include streaming chunks | ✅ |
| 5.3.2.3 | Add JSON tags | Proper serialization | ✅ |
| 5.3.2.4 | Add conversion helpers | To/from `chat.Message` types | ✅ |

#### Task 5.3.3: Implement Handler Factory ✅

**File:** `go/hosting/openai/handler.go`

**Implementation Details:**

```go
package openai

import (
    "net/http"
    "github.com/microsoft/agent-framework-go/agent"
    "github.com/microsoft/agent-framework-go/hosting"
)

// Handler serves OpenAI-compatible endpoints for an agent.
type Handler struct {
    agent        agent.Agent
    sessionStore hosting.SessionStore
    modelName    string
    streamingEnabled bool
}

// Option configures a Handler.
type Option func(*Handler)

// NewHandler creates an HTTP handler for the given agent.
func NewHandler(a agent.Agent, opts ...Option) http.Handler

// WithSessionStore sets the session store for conversation persistence.
func WithSessionStore(store hosting.SessionStore) Option

// WithModelName sets the model name returned in responses.
func WithModelName(name string) Option

// WithStreamingEnabled enables or disables streaming responses.
func WithStreamingEnabled(enabled bool) Option
```

**Subtasks:**

| ID | Task | Acceptance Criteria | Status |
|----|------|---------------------|--------|
| 5.3.3.1 | Implement `NewHandler` | Returns `http.Handler` with routes | ✅ |
| 5.3.3.2 | Configure route multiplexer | Standard library `http.ServeMux` | ✅ |
| 5.3.3.3 | Implement options | All options functional | ✅ |
| 5.3.3.4 | Write unit tests | Handler creation and routing | ✅ |

#### Task 5.3.4: Implement Chat Completions Endpoint ✅

**File:** `go/hosting/openai/completions.go`

**Implementation Details:**

```go
package openai

import (
    "encoding/json"
    "net/http"
)

// handleChatCompletions handles POST /v1/chat/completions
func (h *Handler) handleChatCompletions(w http.ResponseWriter, r *http.Request) {
    // 1. Parse request body
    var req ChatCompletionRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        h.writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
        return
    }

    // 2. Convert to chat.Message slice
    messages := h.convertMessages(req.Messages)

    // 3. Get or create session
    convID := r.Header.Get("X-Conversation-ID")
    session, err := h.getOrCreateSession(r.Context(), convID)
    
    // 4. Route to streaming or non-streaming
    if req.Stream {
        h.streamCompletions(w, r.Context(), session, messages, req)
    } else {
        h.completeSync(w, r.Context(), session, messages, req)
    }
}

func (h *Handler) completeSync(w http.ResponseWriter, ctx context.Context, 
    session agent.Session, messages []chat.Message, req ChatCompletionRequest) {
    // Execute agent
    response, err := h.agent.Run(ctx, messages, agent.WithSession(session))
    if err != nil {
        h.writeError(w, http.StatusInternalServerError, "agent_error", err.Error())
        return
    }
    
    // Convert to OpenAI response format
    resp := h.toCompletionResponse(response, req.Model)
    h.writeJSON(w, http.StatusOK, resp)
}
```

**Subtasks:**

| ID | Task | Acceptance Criteria | Status |
|----|------|---------------------|--------|
| 5.3.4.1 | Implement request parsing | Validate required fields | ✅ |
| 5.3.4.2 | Implement message conversion | Map OpenAI to chat.Message | ✅ |
| 5.3.4.3 | Implement session handling | Create/retrieve sessions | ✅ |
| 5.3.4.4 | Implement sync completion | Non-streaming response | ✅ |
| 5.3.4.5 | Implement response conversion | Map agent response to OpenAI format | ✅ |
| 5.3.4.6 | Write integration tests | Full request/response cycle | ✅ |

#### Task 5.3.5: Implement SSE Streaming ✅

**File:** `go/hosting/openai/streaming.go`

**Implementation Details:**

```go
package openai

import (
    "encoding/json"
    "fmt"
    "net/http"
)

// streamCompletions sends streaming SSE responses.
func (h *Handler) streamCompletions(w http.ResponseWriter, ctx context.Context,
    session agent.Session, messages []chat.Message, req ChatCompletionRequest) {
    
    // Set SSE headers
    w.Header().Set("Content-Type", "text/event-stream")
    w.Header().Set("Cache-Control", "no-cache")
    w.Header().Set("Connection", "keep-alive")
    
    flusher, ok := w.(http.Flusher)
    if !ok {
        h.writeError(w, http.StatusInternalServerError, "streaming_unsupported", 
            "streaming not supported")
        return
    }
    
    // Start streaming agent execution
    stream := h.agent.RunStreaming(ctx, messages, agent.WithSession(session))
    
    for update := range stream.Updates() {
        chunk := h.toCompletionChunk(update, req.Model)
        data, _ := json.Marshal(chunk)
        fmt.Fprintf(w, "data: %s\n\n", data)
        flusher.Flush()
    }
    
    // Send [DONE] marker
    fmt.Fprintf(w, "data: [DONE]\n\n")
    flusher.Flush()
}

// writeSSE writes a single SSE event.
func writeSSE(w http.ResponseWriter, event, data string) {
    if event != "" {
        fmt.Fprintf(w, "event: %s\n", event)
    }
    fmt.Fprintf(w, "data: %s\n\n", data)
}
```

**Subtasks:**

| ID | Task | Acceptance Criteria | Status |
|----|------|---------------------|--------|
| 5.3.5.1 | Set SSE headers | Content-Type, Cache-Control | ✅ |
| 5.3.5.2 | Implement chunk streaming | Convert updates to chunks | ✅ |
| 5.3.5.3 | Implement [DONE] marker | OpenAI-compatible termination | ✅ |
| 5.3.5.4 | Handle client disconnect | Context cancellation | ✅ |
| 5.3.5.5 | Write streaming tests | Verify chunk format | ✅ |

#### Task 5.3.6: Implement Hosting Builder Enhancement ✅

**File:** `go/hosting/builder.go`

**Implementation Details:**

```go
package hosting

import (
    "net/http"
    "github.com/microsoft/agent-framework-go/agent"
)

// HostedAgentBuilder builds a hosted agent with session management.
type HostedAgentBuilder struct {
    name         string
    agent        agent.Agent
    sessionStore SessionStore
    middleware   []func(http.Handler) http.Handler
}

// NewHostedAgentBuilder creates a new builder.
func NewHostedAgentBuilder(name string) *HostedAgentBuilder

// WithAgent sets the agent to host.
func (b *HostedAgentBuilder) WithAgent(a agent.Agent) *HostedAgentBuilder

// WithSessionStore sets the session store.
func (b *HostedAgentBuilder) WithSessionStore(s SessionStore) *HostedAgentBuilder

// WithMiddleware adds HTTP middleware.
func (b *HostedAgentBuilder) WithMiddleware(m func(http.Handler) http.Handler) *HostedAgentBuilder

// Build creates the hosted agent.
func (b *HostedAgentBuilder) Build() *HostedAgent

// HostedAgent wraps an agent with session management.
type HostedAgent struct {
    agent.Agent
    name         string
    sessionStore SessionStore
}

// GetOrCreateSession retrieves or creates a session for the conversation.
func (h *HostedAgent) GetOrCreateSession(ctx context.Context, convID string) (agent.Session, error)

// SaveSession persists the session state.
func (h *HostedAgent) SaveSession(ctx context.Context, convID string, session agent.Session) error
```

**Subtasks:**

| ID | Task | Acceptance Criteria | Status |
|----|------|---------------------|--------|
| 5.3.6.1 | Implement builder pattern | Fluent API | ✅ |
| 5.3.6.2 | Implement `Build` | Creates `HostedAgent` | ✅ |
| 5.3.6.3 | Implement session methods | Get/create/save sessions | ✅ |
| 5.3.6.4 | Write unit tests | Builder and session lifecycle | ✅ |

---

## Phase 3: Declarative Agents (Weeks 5-7)

### Feature 5.2: Declarative Agent Definitions

**Objective:** Enable agents to be defined in YAML files and loaded dynamically.

#### Task 5.2.1: Create Declarative Package Structure

**Files to Create:**

```
go/declarative/
├── doc.go
├── loader.go
├── loader_test.go
├── models.go
├── models_test.go
├── factory.go
├── factory_test.go
├── providers.go
├── providers_test.go
├── tools.go
├── tools_test.go
├── eval.go
├── eval_test.go
└── validation.go
```

#### Task 5.2.2: Define YAML Models

**File:** `go/declarative/models.go`

**Implementation Details:**

```go
package declarative

// PromptAgent represents a declarative agent definition.
type PromptAgent struct {
    Schema       string            `yaml:"$schema,omitempty"`
    Kind         string            `yaml:"kind"`           // "Prompt"
    Name         string            `yaml:"name"`
    Description  string            `yaml:"description,omitempty"`
    Instructions string            `yaml:"instructions"`
    Model        Model             `yaml:"model"`
    Tools        []Tool            `yaml:"tools,omitempty"`
    Inputs       []PropertySchema  `yaml:"inputs,omitempty"`
    Outputs      []PropertySchema  `yaml:"outputs,omitempty"`
}

// Model defines the AI model configuration.
type Model struct {
    Provider   string     `yaml:"provider"`   // "AzureOpenAI", "OpenAI", "Anthropic"
    APIType    string     `yaml:"apiType"`    // "Chat", "Responses", "Assistants"
    Connection Connection `yaml:"connection"`
    Endpoint   string     `yaml:"endpoint,omitempty"`
    Model      string     `yaml:"model"`      // Deployment or model name
    Config     any        `yaml:"config,omitempty"`
}

// Connection defines authentication settings.
type Connection struct {
    Type   string `yaml:"type"`   // "apiKey", "remote", "anonymous"
    Source string `yaml:"source"` // Environment variable or URL
}

// Tool defines a tool available to the agent.
type Tool struct {
    Kind        string          `yaml:"kind"`        // "function", "mcp", "webSearch", etc.
    Name        string          `yaml:"name"`
    Description string          `yaml:"description,omitempty"`
    Server      string          `yaml:"server,omitempty"`      // For MCP
    Parameters  []PropertySchema `yaml:"parameters,omitempty"` // For function
}

// PropertySchema defines a parameter or output schema.
type PropertySchema struct {
    Name        string `yaml:"name"`
    Type        string `yaml:"type"`
    Description string `yaml:"description,omitempty"`
    Required    bool   `yaml:"required,omitempty"`
    Default     any    `yaml:"default,omitempty"`
}
```

**Subtasks:**

| ID | Task | Acceptance Criteria |
|----|------|---------------------|
| 5.2.2.1 | Define `PromptAgent` struct | All YAML fields mapped |
| 5.2.2.2 | Define `Model` struct | Provider, connection, config |
| 5.2.2.3 | Define `Tool` struct | All tool kinds supported |
| 5.2.2.4 | Define `PropertySchema` | Parameter definitions |
| 5.2.2.5 | Add YAML tags | Proper unmarshaling |

#### Task 5.2.3: Implement YAML Loader

**File:** `go/declarative/loader.go`

**Implementation Details:**

```go
package declarative

import (
    "io"
    "os"
    "gopkg.in/yaml.v3"
)

// LoadFromFile loads a PromptAgent from a YAML file.
func LoadFromFile(path string) (*PromptAgent, error) {
    f, err := os.Open(path)
    if err != nil {
        return nil, fmt.Errorf("failed to open file: %w", err)
    }
    defer f.Close()
    return LoadFromReader(f)
}

// LoadFromReader loads a PromptAgent from a reader.
func LoadFromReader(r io.Reader) (*PromptAgent, error) {
    var agent PromptAgent
    decoder := yaml.NewDecoder(r)
    if err := decoder.Decode(&agent); err != nil {
        return nil, fmt.Errorf("failed to parse YAML: %w", err)
    }
    if err := validate(&agent); err != nil {
        return nil, err
    }
    return &agent, nil
}

// LoadFromString loads a PromptAgent from a YAML string.
func LoadFromString(content string) (*PromptAgent, error) {
    return LoadFromReader(strings.NewReader(content))
}
```

**Subtasks:**

| ID | Task | Acceptance Criteria |
|----|------|---------------------|
| 5.2.3.1 | Implement `LoadFromFile` | File reading with error handling |
| 5.2.3.2 | Implement `LoadFromReader` | YAML parsing with validation |
| 5.2.3.3 | Implement `LoadFromString` | String convenience method |
| 5.2.3.4 | Write loader tests | Test all sample YAML files |

#### Task 5.2.4: Implement Environment Variable Evaluation

**File:** `go/declarative/eval.go`

**Implementation Details:**

```go
package declarative

import (
    "os"
    "regexp"
    "strings"
)

var envPattern = regexp.MustCompile(`^=Env\.([A-Z_][A-Z0-9_]*)$`)

// tryPowerFxEval evaluates PowerFx-style expressions.
// Currently supports =Env.VAR_NAME for environment variable substitution.
func tryPowerFxEval(expr string) (string, bool) {
    matches := envPattern.FindStringSubmatch(expr)
    if len(matches) == 2 {
        value := os.Getenv(matches[1])
        return value, true
    }
    return expr, false
}

// evaluateStrings recursively evaluates all string fields in the agent definition.
func evaluateStrings(agent *PromptAgent) error {
    // Evaluate connection source
    if evaluated, ok := tryPowerFxEval(agent.Model.Connection.Source); ok {
        agent.Model.Connection.Source = evaluated
    }
    // Evaluate endpoint
    if evaluated, ok := tryPowerFxEval(agent.Model.Endpoint); ok {
        agent.Model.Endpoint = evaluated
    }
    // ... evaluate other fields
    return nil
}
```

**Subtasks:**

| ID | Task | Acceptance Criteria |
|----|------|---------------------|
| 5.2.4.1 | Implement `tryPowerFxEval` | Parse =Env.VAR_NAME expressions |
| 5.2.4.2 | Implement `evaluateStrings` | Recursive field evaluation |
| 5.2.4.3 | Write eval tests | Test environment variable substitution |

#### Task 5.2.5: Implement Agent Factory

**File:** `go/declarative/factory.go`

**Implementation Details:**

```go
package declarative

import (
    "context"
    "github.com/microsoft/agent-framework-go/agent"
    "github.com/microsoft/agent-framework-go/tool"
)

// AgentFactory creates agents from declarative definitions.
type AgentFactory struct {
    bindings  map[string]tool.Func
    providers map[string]ProviderBuilder
}

// ProviderBuilder creates a chat client for a provider configuration.
type ProviderBuilder func(model Model) (chat.Client, error)

// NewAgentFactory creates a new factory with default providers.
func NewAgentFactory() *AgentFactory

// WithBinding registers a function binding for tool lookup.
func (f *AgentFactory) WithBinding(name string, fn tool.Func) *AgentFactory

// WithProvider registers a custom provider builder.
func (f *AgentFactory) WithProvider(key string, builder ProviderBuilder) *AgentFactory

// Create creates an agent from a PromptAgent definition.
func (f *AgentFactory) Create(ctx context.Context, def *PromptAgent) (agent.Agent, error)

// CreateFromFile loads and creates an agent from a YAML file.
func (f *AgentFactory) CreateFromFile(ctx context.Context, path string) (agent.Agent, error)

// CreateFromString loads and creates an agent from a YAML string.
func (f *AgentFactory) CreateFromString(ctx context.Context, content string) (agent.Agent, error)
```

**Subtasks:**

| ID | Task | Acceptance Criteria |
|----|------|---------------------|
| 5.2.5.1 | Implement `NewAgentFactory` | Initialize with default providers |
| 5.2.5.2 | Implement `WithBinding` | Register function bindings |
| 5.2.5.3 | Implement `WithProvider` | Custom provider registration |
| 5.2.5.4 | Implement `Create` | Build agent from definition |
| 5.2.5.5 | Write factory tests | Test agent creation |

#### Task 5.2.6: Implement Default Provider Builders

**File:** `go/declarative/providers.go`

**Implementation Details:**

```go
package declarative

import (
    "github.com/microsoft/agent-framework-go/providers/azureopenai"
    "github.com/microsoft/agent-framework-go/providers/openai"
)

// defaultProviders maps provider keys to builder functions.
var defaultProviders = map[string]ProviderBuilder{
    "AzureOpenAI.Chat":      newAzureOpenAIChatProvider,
    "AzureOpenAI.Responses": newAzureOpenAIResponsesProvider,
    "OpenAI.Chat":           newOpenAIChatProvider,
    "OpenAI.Responses":      newOpenAIResponsesProvider,
}

func newAzureOpenAIChatProvider(model Model) (chat.Client, error) {
    apiKey, err := resolveAPIKey(model.Connection)
    if err != nil {
        return nil, err
    }
    return azureopenai.NewClient(
        model.Endpoint,
        model.Model,
        azureopenai.WithAPIKey(apiKey),
    )
}

func resolveAPIKey(conn Connection) (string, error) {
    switch conn.Type {
    case "apiKey":
        return conn.Source, nil
    default:
        return "", fmt.Errorf("unsupported connection type: %s", conn.Type)
    }
}
```

**Subtasks:**

| ID | Task | Acceptance Criteria |
|----|------|---------------------|
| 5.2.6.1 | Implement Azure OpenAI providers | Chat and Responses API types |
| 5.2.6.2 | Implement OpenAI providers | Direct OpenAI API |
| 5.2.6.3 | Implement `resolveAPIKey` | Handle connection types |
| 5.2.6.4 | Write provider tests | Test each provider type |

#### Task 5.2.7: Implement Tool Parsing

**File:** `go/declarative/tools.go`

**Implementation Details:**

```go
package declarative

import (
    "github.com/microsoft/agent-framework-go/tool"
)

// parseTool converts a declarative Tool to a tool.Tool.
func (f *AgentFactory) parseTool(t Tool) (tool.Tool, error) {
    switch t.Kind {
    case "function":
        return f.parseFunctionTool(t)
    case "mcp":
        return f.parseMCPTool(t)
    case "webSearch":
        return f.parseHostedTool(t, tool.WebSearchToolName)
    case "fileSearch":
        return f.parseHostedTool(t, tool.FileSearchToolName)
    case "codeInterpreter":
        return f.parseHostedTool(t, tool.CodeInterpreterToolName)
    default:
        return nil, fmt.Errorf("unsupported tool kind: %s", t.Kind)
    }
}

func (f *AgentFactory) parseFunctionTool(t Tool) (tool.Tool, error) {
    fn, ok := f.bindings[t.Name]
    if !ok {
        return nil, fmt.Errorf("no binding found for function: %s", t.Name)
    }
    return tool.NewFunctionTool(t.Name, t.Description, fn), nil
}

func (f *AgentFactory) parseMCPTool(t Tool) (tool.Tool, error) {
    // Create hosted MCP tool that will be executed by the service
    return tool.NewHostedMCPTool(t.Name, t.Server, t.Description), nil
}

func (f *AgentFactory) parseHostedTool(t Tool, hostedName string) (tool.Tool, error) {
    return tool.NewHostedTool(hostedName, t.Description), nil
}
```

**Subtasks:**

| ID | Task | Acceptance Criteria |
|----|------|---------------------|
| 5.2.7.1 | Implement `parseTool` dispatcher | Route by kind |
| 5.2.7.2 | Implement function tool parsing | Look up bindings |
| 5.2.7.3 | Implement MCP tool parsing | Create hosted MCP tool |
| 5.2.7.4 | Implement hosted tool parsing | Web search, file search, code interpreter |
| 5.2.7.5 | Write tool parsing tests | Test all tool kinds |

---

## Phase 4: MCP Integration (Weeks 8-10)

### Feature 5.4: MCP Integration

**Objective:** Enable agents to use tools from MCP servers and expose agents as MCP servers.

#### Task 5.4.1: Create MCP Package Structure

**Files to Create:**

```
go/mcp/
├── doc.go
├── client.go
├── client_test.go
├── transport.go
├── transport_stdio.go
├── transport_stdio_test.go
├── transport_http.go
├── transport_http_test.go
├── tool.go
├── tool_test.go
├── server.go
├── server_test.go
├── types.go
├── hosted.go
└── hosted_test.go
```

**Dependencies:**
- Add `github.com/mark3labs/mcp-go v0.8.0` to `go.mod`

#### Task 5.4.2: Define MCP Types

**File:** `go/mcp/types.go`

**Implementation Details:**

```go
package mcp

// ToolInfo describes an MCP tool.
type ToolInfo struct {
    Name        string          `json:"name"`
    Description string          `json:"description,omitempty"`
    InputSchema json.RawMessage `json:"inputSchema,omitempty"`
}

// Content represents MCP content.
type Content struct {
    Type string `json:"type"` // "text", "image", "resource"
    Text string `json:"text,omitempty"`
    Data string `json:"data,omitempty"` // Base64 for binary
    URI  string `json:"uri,omitempty"`
}

// CallToolResult is the result of calling an MCP tool.
type CallToolResult struct {
    Content []Content `json:"content"`
    IsError bool      `json:"isError,omitempty"`
}

// ResourceInfo describes an MCP resource.
type ResourceInfo struct {
    URI         string `json:"uri"`
    Name        string `json:"name"`
    Description string `json:"description,omitempty"`
    MimeType    string `json:"mimeType,omitempty"`
}
```

**Subtasks:**

| ID | Task | Acceptance Criteria |
|----|------|---------------------|
| 5.4.2.1 | Define `ToolInfo` | MCP tool definition |
| 5.4.2.2 | Define `Content` | Content types |
| 5.4.2.3 | Define `CallToolResult` | Tool result structure |
| 5.4.2.4 | Define `ResourceInfo` | Resource definition |

#### Task 5.4.3: Implement Transport Interface

**File:** `go/mcp/transport.go`

**Implementation Details:**

```go
package mcp

import (
    "context"
    "io"
)

// Transport defines the interface for MCP communication.
type Transport interface {
    // Start initializes the transport connection.
    Start(ctx context.Context) error
    
    // Send sends a JSON-RPC request and returns the response.
    Send(ctx context.Context, method string, params any) (json.RawMessage, error)
    
    // Close terminates the transport connection.
    Close() error
}

// TransportOption configures a transport.
type TransportOption func(*transportOptions)

type transportOptions struct {
    timeout time.Duration
    logger  Logger
}
```

#### Task 5.4.4: Implement Stdio Transport

**File:** `go/mcp/transport_stdio.go`

**Implementation Details:**

```go
package mcp

import (
    "bufio"
    "context"
    "encoding/json"
    "os/exec"
)

// StdioTransport communicates with an MCP server via stdin/stdout.
type StdioTransport struct {
    cmd      *exec.Cmd
    stdin    io.WriteCloser
    stdout   *bufio.Reader
    mu       sync.Mutex
    reqID    int64
}

// NewStdioTransport creates a transport for a stdio-based MCP server.
func NewStdioTransport(command string, args ...string) *StdioTransport

func (t *StdioTransport) Start(ctx context.Context) error {
    t.cmd = exec.CommandContext(ctx, t.command, t.args...)
    
    var err error
    t.stdin, err = t.cmd.StdinPipe()
    if err != nil {
        return fmt.Errorf("failed to get stdin: %w", err)
    }
    
    stdout, err := t.cmd.StdoutPipe()
    if err != nil {
        return fmt.Errorf("failed to get stdout: %w", err)
    }
    t.stdout = bufio.NewReader(stdout)
    
    return t.cmd.Start()
}

func (t *StdioTransport) Send(ctx context.Context, method string, params any) (json.RawMessage, error) {
    t.mu.Lock()
    defer t.mu.Unlock()
    
    id := atomic.AddInt64(&t.reqID, 1)
    req := jsonrpcRequest{
        JSONRPC: "2.0",
        ID:      id,
        Method:  method,
        Params:  params,
    }
    
    // Write request
    if err := json.NewEncoder(t.stdin).Encode(req); err != nil {
        return nil, err
    }
    
    // Read response
    var resp jsonrpcResponse
    if err := json.NewDecoder(t.stdout).Decode(&resp); err != nil {
        return nil, err
    }
    
    if resp.Error != nil {
        return nil, resp.Error
    }
    
    return resp.Result, nil
}
```

**Subtasks:**

| ID | Task | Acceptance Criteria |
|----|------|---------------------|
| 5.4.4.1 | Implement `NewStdioTransport` | Initialize command |
| 5.4.4.2 | Implement `Start` | Start process, get pipes |
| 5.4.4.3 | Implement `Send` | JSON-RPC request/response |
| 5.4.4.4 | Implement `Close` | Terminate process |
| 5.4.4.5 | Write stdio tests | Mock process for testing |

#### Task 5.4.5: Implement HTTP Transport

**File:** `go/mcp/transport_http.go`

**Implementation Details:**

```go
package mcp

import (
    "bufio"
    "context"
    "net/http"
)

// HTTPTransport communicates with an MCP server via HTTP/SSE.
type HTTPTransport struct {
    endpoint   string
    httpClient *http.Client
    headers    http.Header
    sseConn    *sseConnection
    mu         sync.Mutex
}

// NewHTTPTransport creates a transport for an HTTP-based MCP server.
func NewHTTPTransport(endpoint string, opts ...HTTPTransportOption) *HTTPTransport

// HTTPTransportOption configures an HTTPTransport.
type HTTPTransportOption func(*HTTPTransport)

// WithHTTPClient sets a custom HTTP client.
func WithHTTPClient(client *http.Client) HTTPTransportOption

// WithHeaders sets custom headers for requests.
func WithHeaders(headers http.Header) HTTPTransportOption

func (t *HTTPTransport) Start(ctx context.Context) error {
    // For streamable HTTP, establish SSE connection
    return t.connectSSE(ctx)
}

func (t *HTTPTransport) Send(ctx context.Context, method string, params any) (json.RawMessage, error) {
    // Send JSON-RPC via POST, receive response
    reqBody, _ := json.Marshal(jsonrpcRequest{
        JSONRPC: "2.0",
        ID:      atomic.AddInt64(&t.reqID, 1),
        Method:  method,
        Params:  params,
    })
    
    resp, err := t.httpClient.Post(t.endpoint, "application/json", bytes.NewReader(reqBody))
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()
    
    var rpcResp jsonrpcResponse
    if err := json.NewDecoder(resp.Body).Decode(&rpcResp); err != nil {
        return nil, err
    }
    
    return rpcResp.Result, nil
}
```

**Subtasks:**

| ID | Task | Acceptance Criteria |
|----|------|---------------------|
| 5.4.5.1 | Implement `NewHTTPTransport` | Configure endpoint |
| 5.4.5.2 | Implement `Start` | SSE connection |
| 5.4.5.3 | Implement `Send` | HTTP POST JSON-RPC |
| 5.4.5.4 | Implement SSE parsing | Event stream handling |
| 5.4.5.5 | Write HTTP tests | Mock server for testing |

#### Task 5.4.6: Implement MCP Client

**File:** `go/mcp/client.go`

**Implementation Details:**

```go
package mcp

import (
    "context"
    "sync"
    "github.com/microsoft/agent-framework-go/tool"
)

// Client connects to an MCP server and provides tool access.
type Client struct {
    transport Transport
    tools     []ToolInfo
    mu        sync.RWMutex
    connected bool
}

// ClientOption configures a Client.
type ClientOption func(*clientOptions)

// NewClient creates a new MCP client with the given transport.
func NewClient(transport Transport, opts ...ClientOption) *Client

// Connect establishes the connection and discovers tools.
func (c *Client) Connect(ctx context.Context) error {
    if err := c.transport.Start(ctx); err != nil {
        return err
    }
    
    // Initialize connection
    _, err := c.transport.Send(ctx, "initialize", initializeParams{
        ProtocolVersion: "2024-11-05",
        Capabilities:    clientCapabilities{},
        ClientInfo:      clientInfo{Name: "agent-framework-go", Version: "1.0.0"},
    })
    if err != nil {
        return fmt.Errorf("initialize failed: %w", err)
    }
    
    // Discover tools
    return c.refreshTools(ctx)
}

// ListTools returns the available tools from the MCP server.
func (c *Client) ListTools(ctx context.Context) ([]ToolInfo, error) {
    c.mu.RLock()
    defer c.mu.RUnlock()
    return c.tools, nil
}

// CallTool invokes a tool on the MCP server.
func (c *Client) CallTool(ctx context.Context, name string, args map[string]any) (*CallToolResult, error) {
    result, err := c.transport.Send(ctx, "tools/call", callToolParams{
        Name:      name,
        Arguments: args,
    })
    if err != nil {
        return nil, err
    }
    
    var callResult CallToolResult
    if err := json.Unmarshal(result, &callResult); err != nil {
        return nil, err
    }
    return &callResult, nil
}

// Tools returns the MCP tools as framework tool.Tool implementations.
func (c *Client) Tools() []tool.Tool {
    c.mu.RLock()
    defer c.mu.RUnlock()
    
    tools := make([]tool.Tool, len(c.tools))
    for i, t := range c.tools {
        tools[i] = newMCPBridgeTool(c, t)
    }
    return tools
}
```

**Subtasks:**

| ID | Task | Acceptance Criteria |
|----|------|---------------------|
| 5.4.6.1 | Implement `NewClient` | Initialize with transport |
| 5.4.6.2 | Implement `Connect` | Initialize and discover tools |
| 5.4.6.3 | Implement `ListTools` | Return cached tools |
| 5.4.6.4 | Implement `CallTool` | Invoke tool via transport |
| 5.4.6.5 | Implement `Tools` | Return bridged tool.Tool slice |
| 5.4.6.6 | Write client tests | Test full lifecycle |

#### Task 5.4.7: Implement MCP Bridge Tool

**File:** `go/mcp/tool.go`

**Implementation Details:**

```go
package mcp

import (
    "context"
    "github.com/microsoft/agent-framework-go/tool"
)

// mcpBridgeTool wraps an MCP tool as a framework tool.Tool.
type mcpBridgeTool struct {
    client *Client
    info   ToolInfo
}

func newMCPBridgeTool(client *Client, info ToolInfo) *mcpBridgeTool {
    return &mcpBridgeTool{client: client, info: info}
}

func (t *mcpBridgeTool) Name() string {
    return t.info.Name
}

func (t *mcpBridgeTool) Description() string {
    return t.info.Description
}

func (t *mcpBridgeTool) Parameters() tool.Schema {
    var schema tool.Schema
    json.Unmarshal(t.info.InputSchema, &schema)
    return schema
}

func (t *mcpBridgeTool) Invoke(ctx context.Context, args string) (tool.Result, error) {
    var argsMap map[string]any
    if err := json.Unmarshal([]byte(args), &argsMap); err != nil {
        return tool.Result{}, fmt.Errorf("invalid arguments: %w", err)
    }
    
    result, err := t.client.CallTool(ctx, t.info.Name, argsMap)
    if err != nil {
        return tool.Result{}, err
    }
    
    // Convert MCP content to tool result
    return convertResult(result), nil
}
```

**Subtasks:**

| ID | Task | Acceptance Criteria |
|----|------|---------------------|
| 5.4.7.1 | Implement `tool.Tool` interface | All methods |
| 5.4.7.2 | Implement argument parsing | JSON to map |
| 5.4.7.3 | Implement result conversion | MCP content to tool.Result |
| 5.4.7.4 | Write bridge tests | Tool invocation |

#### Task 5.4.8: Implement Agent as MCP Server

**File:** `go/mcp/server.go`

**Implementation Details:**

```go
package mcp

import (
    "context"
    "net/http"
    "github.com/microsoft/agent-framework-go/agent"
)

// Server exposes an agent as an MCP server.
type Server struct {
    agent     agent.Agent
    tools     []ToolInfo
    resources []ResourceInfo
}

// ServerOption configures a Server.
type ServerOption func(*Server)

// AsMCPServer wraps an agent as an MCP server.
func AsMCPServer(a agent.Agent, opts ...ServerOption) *Server {
    s := &Server{agent: a}
    for _, opt := range opts {
        opt(s)
    }
    s.initTools()
    return s
}

// WithResources adds resources to expose.
func WithResources(resources ...ResourceInfo) ServerOption

// ListTools returns the tools exposed by the agent.
func (s *Server) ListTools() []ToolInfo

// CallTool invokes a tool on the agent.
func (s *Server) CallTool(ctx context.Context, name string, args map[string]any) (*CallToolResult, error)

// ServeHTTP handles MCP JSON-RPC requests.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    // Parse JSON-RPC request
    var req jsonrpcRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        s.writeError(w, -32700, "Parse error", nil)
        return
    }
    
    // Dispatch by method
    var result any
    var err error
    switch req.Method {
    case "initialize":
        result = s.handleInitialize(req.Params)
    case "tools/list":
        result = s.handleToolsList(req.Params)
    case "tools/call":
        result, err = s.handleToolsCall(r.Context(), req.Params)
    default:
        s.writeError(w, -32601, "Method not found", nil)
        return
    }
    
    if err != nil {
        s.writeError(w, -32000, err.Error(), nil)
        return
    }
    
    s.writeResult(w, req.ID, result)
}
```

**Subtasks:**

| ID | Task | Acceptance Criteria |
|----|------|---------------------|
| 5.4.8.1 | Implement `AsMCPServer` | Create server from agent |
| 5.4.8.2 | Implement tool extraction | Extract agent tools as MCP tools |
| 5.4.8.3 | Implement `ServeHTTP` | JSON-RPC 2.0 handler |
| 5.4.8.4 | Implement method handlers | initialize, tools/list, tools/call |
| 5.4.8.5 | Write server tests | HTTP request/response |

---

## Phase 5: Durable Agents (Weeks 11-14)

### Feature 5.1: Durable Agents

**Objective:** Implement persistent agent sessions using Temporal.io workflows.

#### Task 5.1.1: Create Durable Package Structure

**Files to Create:**

```
go/durable/
├── doc.go
├── agent.go
├── agent_test.go
├── session.go
├── session_test.go
├── sessionid.go
├── sessionid_test.go
├── state.go
├── state_test.go
├── state_entry.go
├── workflow.go
├── workflow_test.go
├── activity.go
├── activity_test.go
├── worker.go
├── worker_test.go
├── client.go
├── client_test.go
└── options.go
```

**Dependencies:**
- Add `go.temporal.io/sdk v1.29.1` to `go.mod`

#### Task 5.1.2: Define State Types

**File:** `go/durable/state.go`

**Implementation Details:**

```go
package durable

import (
    "time"
)

// SchemaVersion is the current state schema version for cross-platform compatibility.
const SchemaVersion = "1.1.0"

// State represents the persisted agent state.
type State struct {
    SchemaVersion string    `json:"schemaVersion"`
    Data          StateData `json:"data"`
}

// StateData contains the conversation history and metadata.
type StateData struct {
    ConversationHistory []StateEntry `json:"conversationHistory"`
    ExpirationTimeUtc   *time.Time   `json:"expirationTimeUtc,omitempty"`
}

// NewState creates a new empty state.
func NewState() *State {
    return &State{
        SchemaVersion: SchemaVersion,
        Data: StateData{
            ConversationHistory: make([]StateEntry, 0),
        },
    }
}

// AppendRequest adds a request entry to the conversation history.
func (s *State) AppendRequest(entry RequestEntry)

// AppendResponse adds a response entry to the conversation history.
func (s *State) AppendResponse(entry ResponseEntry)

// BuildChatMessages converts the history to chat.Message slice.
func (s *State) BuildChatMessages() []chat.Message
```

#### Task 5.1.3: Define State Entry Types

**File:** `go/durable/state_entry.go`

**Implementation Details:**

```go
package durable

import (
    "time"
)

// StateEntry is the base interface for history entries.
type StateEntry interface {
    entryType() string
    timestamp() time.Time
}

// RequestEntry represents a user request in the history.
type RequestEntry struct {
    Type         string           `json:"$type"` // "request"
    Timestamp    time.Time        `json:"timestamp"`
    Contents     []ContentEntry   `json:"contents"`
    Instructions string           `json:"instructions,omitempty"`
}

// ResponseEntry represents an agent response in the history.
type ResponseEntry struct {
    Type       string         `json:"$type"` // "response"
    Timestamp  time.Time      `json:"timestamp"`
    Contents   []ContentEntry `json:"contents"`
}

// ContentEntry represents a content item (text, function call, etc.).
type ContentEntry struct {
    Type           string                 `json:"$type"` // "text", "functionCall", "functionResult", etc.
    Text           string                 `json:"text,omitempty"`
    FunctionName   string                 `json:"functionName,omitempty"`
    FunctionCallID string                 `json:"functionCallId,omitempty"`
    Arguments      map[string]any         `json:"arguments,omitempty"`
    Result         string                 `json:"result,omitempty"`
}
```

**Subtasks:**

| ID | Task | Acceptance Criteria |
|----|------|---------------------|
| 5.1.3.1 | Define `StateEntry` interface | Common methods |
| 5.1.3.2 | Define `RequestEntry` | Match JSON schema |
| 5.1.3.3 | Define `ResponseEntry` | Match JSON schema |
| 5.1.3.4 | Define `ContentEntry` | All content types |
| 5.1.3.5 | Write serialization tests | JSON round-trip with .NET/Python |

#### Task 5.1.4: Implement Session ID

**File:** `go/durable/sessionid.go`

**Implementation Details:**

```go
package durable

import (
    "fmt"
    "strings"
)

// SessionID uniquely identifies a durable agent session.
type SessionID struct {
    AgentName string
    Key       string
}

// NewSessionID creates a new session ID.
func NewSessionID(agentName, key string) SessionID {
    return SessionID{AgentName: agentName, Key: key}
}

// WorkflowID returns the Temporal workflow ID for this session.
func (id SessionID) WorkflowID() string {
    return fmt.Sprintf("durable-agent-%s-%s", id.AgentName, id.Key)
}

// ParseSessionID parses a workflow ID back to a SessionID.
func ParseSessionID(workflowID string) (SessionID, error) {
    const prefix = "durable-agent-"
    if !strings.HasPrefix(workflowID, prefix) {
        return SessionID{}, fmt.Errorf("invalid workflow ID format")
    }
    rest := strings.TrimPrefix(workflowID, prefix)
    parts := strings.SplitN(rest, "-", 2)
    if len(parts) != 2 {
        return SessionID{}, fmt.Errorf("invalid workflow ID format")
    }
    return SessionID{AgentName: parts[0], Key: parts[1]}, nil
}

// String returns the string representation.
func (id SessionID) String() string {
    return id.WorkflowID()
}
```

#### Task 5.1.5: Implement Temporal Workflow

**File:** `go/durable/workflow.go`

**Implementation Details:**

```go
package durable

import (
    "time"
    "go.temporal.io/sdk/workflow"
)

// SessionWorkflow is the Temporal workflow for a durable agent session.
func SessionWorkflow(ctx workflow.Context, sessionID SessionID) error {
    state := NewState()
    
    // Register update handler for synchronous run requests
    err := workflow.SetUpdateHandler(ctx, "run", func(ctx workflow.Context, req RunRequest) (RunResponse, error) {
        // Append request to history
        state.AppendRequest(RequestEntry{
            Type:      "request",
            Timestamp: workflow.Now(ctx),
            Contents:  convertToContentEntries(req.Messages),
        })
        
        // Execute agent activity
        var response RunResponse
        activityOpts := workflow.ActivityOptions{
            StartToCloseTimeout: 5 * time.Minute,
            RetryPolicy: &temporal.RetryPolicy{
                MaximumAttempts: 3,
            },
        }
        err := workflow.ExecuteActivity(
            workflow.WithActivityOptions(ctx, activityOpts),
            RunAgentActivity,
            RunAgentInput{
                SessionID: sessionID,
                Messages:  state.BuildChatMessages(),
                Options:   req.Options,
            },
        ).Get(ctx, &response)
        
        if err != nil {
            return RunResponse{}, err
        }
        
        // Append response to history
        state.AppendResponse(ResponseEntry{
            Type:      "response",
            Timestamp: workflow.Now(ctx),
            Contents:  convertFromAgentResponse(response),
        })
        
        return response, nil
    })
    if err != nil {
        return err
    }
    
    // Handle TTL expiration
    if state.Data.ExpirationTimeUtc != nil {
        expiresIn := time.Until(*state.Data.ExpirationTimeUtc)
        if expiresIn > 0 {
            _ = workflow.Sleep(ctx, expiresIn)
        }
    } else {
        // Wait indefinitely for signals
        workflow.Await(ctx, func() bool { return false })
    }
    
    return nil
}
```

**Subtasks:**

| ID | Task | Acceptance Criteria |
|----|------|---------------------|
| 5.1.5.1 | Implement workflow function | Temporal workflow definition |
| 5.1.5.2 | Implement update handler | Synchronous run requests |
| 5.1.5.3 | Implement TTL handling | Expiration via Sleep |
| 5.1.5.4 | Implement continue-as-new | Long history handling |
| 5.1.5.5 | Write workflow tests | Temporal test framework |

#### Task 5.1.6: Implement Agent Activity

**File:** `go/durable/activity.go`

**Implementation Details:**

```go
package durable

import (
    "context"
    "go.temporal.io/sdk/activity"
    "github.com/microsoft/agent-framework-go/agent"
)

// RunAgentInput is the input to the RunAgentActivity.
type RunAgentInput struct {
    SessionID SessionID
    Messages  []chat.Message
    Options   RunOptions
}

// RunAgentActivity executes the agent and returns the response.
func RunAgentActivity(ctx context.Context, input RunAgentInput) (RunResponse, error) {
    logger := activity.GetLogger(ctx)
    logger.Info("Executing agent", "sessionID", input.SessionID)
    
    // Get agent from activity context (registered during worker setup)
    a := getAgentFromContext(ctx)
    if a == nil {
        return RunResponse{}, fmt.Errorf("agent not found in activity context")
    }
    
    // Execute the agent
    response, err := a.Run(ctx, input.Messages)
    if err != nil {
        return RunResponse{}, err
    }
    
    return convertToRunResponse(response), nil
}
```

#### Task 5.1.7: Implement Durable Agent Wrapper

**File:** `go/durable/agent.go`

**Implementation Details:**

```go
package durable

import (
    "context"
    "github.com/microsoft/agent-framework-go/agent"
    "go.temporal.io/sdk/client"
)

// Agent wraps a regular agent with durable session management.
type Agent struct {
    inner       agent.Agent
    client      client.Client
    taskQueue   string
}

// AgentOption configures a durable Agent.
type AgentOption func(*Agent)

// NewAgent creates a durable agent wrapper.
func NewAgent(inner agent.Agent, temporalClient client.Client, opts ...AgentOption) *Agent

// WithTaskQueue sets the Temporal task queue.
func WithTaskQueue(queue string) AgentOption

// Run executes the agent within a durable session.
func (a *Agent) Run(ctx context.Context, messages []chat.Message, opts ...agent.RunOption) (*agent.Response, error) {
    runOpts := agent.ApplyRunOptions(opts)
    sessionID := a.getOrCreateSessionID(runOpts)
    
    // Start or get existing workflow
    workflowOpts := client.StartWorkflowOptions{
        ID:        sessionID.WorkflowID(),
        TaskQueue: a.taskQueue,
    }
    
    run, err := a.client.ExecuteWorkflow(ctx, workflowOpts, SessionWorkflow, sessionID)
    if err != nil {
        // Workflow may already exist, try to get handle
        run = a.client.GetWorkflow(ctx, sessionID.WorkflowID(), "")
    }
    
    // Send run request via update
    var response RunResponse
    handle, err := a.client.UpdateWorkflow(ctx, client.UpdateWorkflowOptions{
        WorkflowID:   sessionID.WorkflowID(),
        UpdateName:   "run",
        Args:         []any{RunRequest{Messages: messages, Options: runOpts}},
        WaitForStage: client.WorkflowUpdateStageCompleted,
    })
    if err != nil {
        return nil, err
    }
    
    if err := handle.Get(ctx, &response); err != nil {
        return nil, err
    }
    
    return convertFromRunResponse(response), nil
}

// RunStreaming executes the agent with streaming within a durable session.
func (a *Agent) RunStreaming(ctx context.Context, messages []chat.Message, opts ...agent.RunOption) *agent.StreamingResponse
```

**Subtasks:**

| ID | Task | Acceptance Criteria |
|----|------|---------------------|
| 5.1.7.1 | Implement `NewAgent` | Configure with Temporal client |
| 5.1.7.2 | Implement `Run` | Execute via workflow update |
| 5.1.7.3 | Implement `RunStreaming` | Streaming with activity signals |
| 5.1.7.4 | Implement session ID generation | From options or auto-generate |
| 5.1.7.5 | Write agent tests | Full integration tests |

#### Task 5.1.8: Implement Temporal Worker

**File:** `go/durable/worker.go`

**Implementation Details:**

```go
package durable

import (
    "go.temporal.io/sdk/client"
    "go.temporal.io/sdk/worker"
    "github.com/microsoft/agent-framework-go/agent"
)

// Worker manages Temporal worker lifecycle.
type Worker struct {
    client    client.Client
    worker    worker.Worker
    agent     agent.Agent
    taskQueue string
}

// WorkerOption configures a Worker.
type WorkerOption func(*Worker)

// NewWorker creates a new Temporal worker for durable agents.
func NewWorker(temporalClient client.Client, a agent.Agent, taskQueue string, opts ...WorkerOption) *Worker {
    w := &Worker{
        client:    temporalClient,
        agent:     a,
        taskQueue: taskQueue,
    }
    
    w.worker = worker.New(temporalClient, taskQueue, worker.Options{})
    
    // Register workflow
    w.worker.RegisterWorkflow(SessionWorkflow)
    
    // Register activity with agent context
    w.worker.RegisterActivityWithOptions(RunAgentActivity, activity.RegisterOptions{
        Name: "RunAgentActivity",
    })
    
    return w
}

// Start starts the worker.
func (w *Worker) Start() error {
    return w.worker.Start()
}

// Stop stops the worker gracefully.
func (w *Worker) Stop() {
    w.worker.Stop()
}
```

**Subtasks:**

| ID | Task | Acceptance Criteria |
|----|------|---------------------|
| 5.1.8.1 | Implement `NewWorker` | Register workflow and activities |
| 5.1.8.2 | Implement `Start` | Start worker |
| 5.1.8.3 | Implement `Stop` | Graceful shutdown |
| 5.1.8.4 | Write worker tests | Worker lifecycle |

---

## Phase 6: Enterprise Features (Weeks 15-18)

### Feature 5.5: Enterprise Integrations

#### Task 5.5.1: Implement DevUI Server

**Files to Create:**

```
go/devui/
├── doc.go
├── server.go
├── server_test.go
├── discovery.go
├── discovery_test.go
├── tracing.go
├── handlers.go
├── frontend/       # Embedded static assets
└── options.go
```

**Implementation Details:**

```go
package devui

import (
    "embed"
    "net/http"
    "github.com/microsoft/agent-framework-go/agent"
)

//go:embed frontend/*
var frontendFS embed.FS

// Server provides a development UI for agent testing.
type Server struct {
    agents    map[string]agent.Agent
    collector *TraceCollector
}

// NewServer creates a DevUI server.
func NewServer(opts ...Option) *Server

// RegisterAgent adds an agent to the DevUI.
func (s *Server) RegisterAgent(name string, a agent.Agent)

// ServeHTTP handles HTTP requests.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request)
```

**Subtasks:**

| ID | Task | Acceptance Criteria |
|----|------|---------------------|
| 5.5.1.1 | Implement server structure | HTTP handler with routes |
| 5.5.1.2 | Implement agent discovery | List registered agents |
| 5.5.1.3 | Implement trace collection | OpenTelemetry integration |
| 5.5.1.4 | Implement API handlers | /meta, /run, /stream, /traces |
| 5.5.1.5 | Embed frontend assets | Static file serving |
| 5.5.1.6 | Write server tests | API endpoint tests |

#### Task 5.5.2: Implement Purview Middleware

**Files to Create:**

```
go/purview/
├── doc.go
├── middleware.go
├── middleware_test.go
├── client.go
├── client_test.go
├── types.go
└── options.go
```

**Implementation Details:**

```go
package purview

import (
    "context"
    "github.com/microsoft/agent-framework-go/agent"
)

// Middleware provides Purview content policy evaluation.
type Middleware struct {
    client   *Client
    settings Settings
}

// Settings configures Purview middleware.
type Settings struct {
    Endpoint      string
    TenantID      string
    ContentPolicy string
    Categories    []string
}

// NewMiddleware creates Purview middleware.
func NewMiddleware(credential azcore.TokenCredential, settings Settings) *Middleware

// Apply returns an agent middleware function.
func (m *Middleware) Apply() agent.Middleware

// ProcessContent evaluates content against Purview policies.
func (m *Middleware) ProcessContent(ctx context.Context, content string) (*PolicyResult, error)
```

**Subtasks:**

| ID | Task | Acceptance Criteria |
|----|------|---------------------|
| 5.5.2.1 | Define Purview types | Request/response structures |
| 5.5.2.2 | Implement Purview client | Azure Graph API calls |
| 5.5.2.3 | Implement caching | ETag-based scope caching |
| 5.5.2.4 | Implement middleware | Pre/post processing |
| 5.5.2.5 | Write middleware tests | Policy evaluation |

---

## Dependency Updates

### go.mod Additions

```go
require (
    // New direct dependencies
    golang.org/x/time v0.5.0                        // Rate limiting
    github.com/sony/gobreaker v1.0.0                // Circuit breaker
    github.com/mark3labs/mcp-go v0.8.0              // MCP client/server
    go.temporal.io/sdk v1.29.1                      // Durable agents
    
    // Existing but promote to direct
    github.com/cenkalti/backoff/v5 v5.0.3           // Retry (currently indirect)
    gopkg.in/yaml.v3 v3.0.1                         // YAML parsing (currently indirect)
)
```

---

## Testing Strategy

### Unit Tests
- 90%+ code coverage target
- Use `testify` for assertions (already a dependency)
- Mock external dependencies (Temporal, HTTP clients)

### Integration Tests
- Temporal: Use Temporal test environment
- MCP: Mock stdio/HTTP servers
- HTTP Hosting: `httptest` package

### Cross-Platform Compatibility Tests
- Verify JSON state schema matches .NET/Python
- Test YAML parsing against sample files in `agent-samples/`

---

## Success Criteria

| Metric | Target |
|--------|--------|
| Code coverage | ≥ 90% |
| All packages documented | godoc for every public API |
| API parity | Feature-complete with .NET/Python |
| Performance overhead | < 10ms for sync runs |
| Streaming latency | < 1ms per chunk |
| Integration tests | Pass for Temporal, MCP, HTTP hosting |

---

## Handoff Checklist

Before starting implementation:

- [ ] Review existing Go code patterns in `go/agent/`, `go/chat/`, `go/protocol/`
- [ ] Set up Temporal.io local development environment
- [ ] Verify MCP Go library compatibility
- [ ] Create feature branches for each phase
- [ ] Set up CI for new packages

---

## Related Documents

- Research: [2026-02-04-go-epic5-enterprise-production-research.md](../research/2026-02-04-go-epic5-enterprise-production-research.md)
- Schema: [durable-agent-entity-state.json](../../schemas/durable-agent-entity-state.json)
- YAML Samples: [agent-samples/](../../agent-samples/)
