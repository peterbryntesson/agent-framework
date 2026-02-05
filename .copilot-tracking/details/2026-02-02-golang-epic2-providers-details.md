<!-- markdownlint-disable-file -->
# Implementation Details: Go Port - Epic 2: LLM Provider Implementations

## Context Reference

Sources:

* [.copilot-tracking/subagent/2026-02-02/dotnet-providers-research.md](.copilot-tracking/subagent/2026-02-02/dotnet-providers-research.md)
* [.copilot-tracking/subagent/2026-02-02/python-providers-research.md](.copilot-tracking/subagent/2026-02-02/python-providers-research.md)
* [.copilot-tracking/subagent/2026-02-02/go-current-state-research.md](.copilot-tracking/subagent/2026-02-02/go-current-state-research.md)

---

## Feature 2.1: Tool System Package

<!-- parallelizable: false -->

### Step 2.1.1: Define Tool interfaces and types

Create the tool package with core interface definitions aligned with .NET `AITool` and Python `ToolProtocol`.

Files:

* `go/tool/tool.go` - Core Tool interface
* `go/tool/result.go` - ToolResult struct
* `go/tool/config.go` - InvocationConfig struct
* `go/tool/doc.go` - Package documentation

#### Interface Definitions

```go
// tool/tool.go

// Tool represents a callable function that agents can invoke.
// Aligns with .NET AITool and Python ToolProtocol.
type Tool interface {
    // Name returns the unique identifier for this tool.
    Name() string
    
    // Description returns a human-readable description of what this tool does.
    Description() string
    
    // Parameters returns the JSON Schema describing the tool's input parameters.
    Parameters() json.RawMessage
    
    // Invoke executes the tool with the given arguments.
    Invoke(ctx context.Context, arguments json.RawMessage) (Result, error)
}

// HostedTool is a tool that runs on the provider's infrastructure (e.g., web search).
type HostedTool interface {
    Tool
    
    // IsHosted returns true, indicating this tool runs on the provider.
    IsHosted() bool
    
    // ProviderConfig returns provider-specific configuration for the hosted tool.
    ProviderConfig() map[string]interface{}
}

// Result represents the output of a tool invocation.
type Result struct {
    Content    string          `json:"content"`
    IsError    bool            `json:"is_error,omitempty"`
    Metadata   map[string]any  `json:"metadata,omitempty"`
    RawOutput  interface{}     `json:"-"`
}
```

#### Invocation Configuration

```go
// tool/config.go

// InvocationConfig controls how tools are invoked during agent runs.
// Aligns with Python FunctionInvocationConfiguration.
type InvocationConfig struct {
    // Enabled controls whether automatic function invocation is active.
    Enabled bool
    
    // MaxIterations limits the number of tool invocation rounds.
    MaxIterations int
    
    // MaxConsecutiveErrors stops invocation after this many consecutive errors.
    MaxConsecutiveErrors int
    
    // TerminateOnUnknownCalls stops if a tool call references an unknown tool.
    TerminateOnUnknownCalls bool
    
    // IncludeDetailedErrors includes error details in tool results.
    IncludeDetailedErrors bool
    
    // AdditionalTools are extra tools available for this invocation only.
    AdditionalTools []Tool
}

// DefaultInvocationConfig returns sensible defaults for tool invocation.
func DefaultInvocationConfig() InvocationConfig {
    return InvocationConfig{
        Enabled:              true,
        MaxIterations:        40,  // Matches Python DEFAULT_MAX_ITERATIONS
        MaxConsecutiveErrors: 3,   // Matches Python DEFAULT_MAX_CONSECUTIVE_ERRORS
    }
}
```

Success criteria:

* Tool interface matches .NET AITool method signatures
* HostedTool interface supports provider-specific configuration
* Result struct handles success, error, and metadata cases
* InvocationConfig has parity with Python FunctionInvocationConfiguration

Context references:

* [python/packages/core/agent_framework/_tools.py](python/packages/core/agent_framework/_tools.py) (Lines 157-180) - ToolProtocol
* [dotnet/src/Microsoft.Agents.AI/ChatClient/ChatClientExtensions.cs](dotnet/src/Microsoft.Agents.AI/ChatClient/ChatClientExtensions.cs) - Tool middleware pattern

Dependencies:

* None - foundational package

---

### Step 2.1.2: Implement FunctionTool with reflection

Create FunctionTool that wraps Go functions and generates JSON Schema using reflection.

Files:

* `go/tool/function.go` - FunctionTool implementation
* `go/tool/schema.go` - JSON Schema generation from Go types

#### FunctionTool Implementation

```go
// tool/function.go

// FunctionTool wraps a Go function to make it callable by AI models.
// Aligns with Python FunctionTool and .NET AIFunctionFactory.
type FunctionTool struct {
    name        string
    description string
    fn          reflect.Value
    inputType   reflect.Type
    schema      json.RawMessage
    
    // Configuration
    ApprovalMode   string // "always_require", "never_require", or ""
    MaxInvocations int
}

// NewFunctionTool creates a FunctionTool from a Go function.
// The function signature must be: func(ctx context.Context, args T) (R, error)
// where T is a struct with json tags defining parameter names and descriptions.
func NewFunctionTool(name, description string, fn interface{}) (*FunctionTool, error) {
    // Validate function signature
    // Extract parameter type via reflection
    // Generate JSON Schema from struct tags
    // Return configured FunctionTool
}

// Func is a decorator that creates a FunctionTool from a function.
// Usage: tool.Func(myFunction, "Description of what it does")
func Func(fn interface{}, description string, opts ...FuncOption) *FunctionTool

// FuncOption configures a FunctionTool created via Func().
type FuncOption func(*FunctionTool)

// WithName sets a custom name for the tool (default: function name).
func WithName(name string) FuncOption

// WithApprovalMode sets whether user approval is required.
func WithApprovalMode(mode string) FuncOption

// WithMaxInvocations limits how many times this tool can be called.
func WithMaxInvocations(max int) FuncOption
```

#### JSON Schema Generation

```go
// tool/schema.go

// GenerateSchema creates a JSON Schema from a Go struct type.
// Supports struct tags for customization:
//   - `json:"name"` for property names
//   - `description:"text"` for property descriptions
//   - `required:"true"` for required properties
//   - `enum:"a,b,c"` for enumerated values
func GenerateSchema(t reflect.Type) (json.RawMessage, error)

// Example struct with annotations:
type WeatherArgs struct {
    Location string `json:"location" description:"The city name" required:"true"`
    Unit     string `json:"unit" description:"Temperature unit" enum:"celsius,fahrenheit"`
}
```

Success criteria:

* FunctionTool validates function signature at creation time
* Schema generation handles struct, primitive, and slice types
* Struct tags control JSON Schema properties
* Func decorator provides convenient syntax

Context references:

* [python/packages/core/agent_framework/_tools.py](python/packages/core/agent_framework/_tools.py) (Lines 556-800) - FunctionTool implementation
* [python/packages/core/agent_framework/_tools.py](python/packages/core/agent_framework/_tools.py) (Lines 1218-1320) - @tool decorator

Dependencies:

* Step 2.1.1 completion

---

### Step 2.1.3: Implement hosted tool types

Create hosted tool implementations for provider-hosted capabilities.

Files:

* `go/tool/hosted.go` - All hosted tool implementations
* `go/tool/hosted_test.go` - Hosted tool tests

#### Hosted Tool Implementations

```go
// tool/hosted.go

// HostedWebSearchTool represents a provider-hosted web search capability.
// Aligns with Python HostedWebSearchTool and .NET ResponseWebSearchToolDefinition.
type HostedWebSearchTool struct {
    ToolName          string
    ToolDescription   string
    SearchContextSize string  // "low", "medium", "high"
    UserLocation      *UserLocation
}

type UserLocation struct {
    Type        string  // "approximate"
    City        string
    Region      string
    Country     string
    CountryCode string
    Timezone    string
}

func (t *HostedWebSearchTool) Name() string           { return t.ToolName }
func (t *HostedWebSearchTool) Description() string    { return t.ToolDescription }
func (t *HostedWebSearchTool) Parameters() json.RawMessage { return nil }
func (t *HostedWebSearchTool) Invoke(ctx context.Context, args json.RawMessage) (Result, error) {
    return Result{}, errors.New("hosted tools are invoked by the provider")
}
func (t *HostedWebSearchTool) IsHosted() bool { return true }
func (t *HostedWebSearchTool) ProviderConfig() map[string]interface{} { /* ... */ }

// HostedCodeInterpreterTool represents a provider-hosted code execution environment.
type HostedCodeInterpreterTool struct {
    ToolName        string
    ToolDescription string
    Container       *CodeInterpreterContainer
    FileIDs         []string
}

type CodeInterpreterContainer struct {
    Image   string
    EnvVars map[string]string
}

// HostedFileSearchTool represents a provider-hosted vector search capability.
type HostedFileSearchTool struct {
    ToolName        string
    ToolDescription string
    VectorStoreIDs  []string
    MaxResults      int
    Ranking         *FileSearchRanking
}

type FileSearchRanking struct {
    Ranker         string
    ScoreThreshold float64
}

// HostedMCPTool represents a Model Context Protocol server integration.
type HostedMCPTool struct {
    ToolName        string
    ToolDescription string
    ServerURL       string
    ServerLabel     string
    AllowedTools    []string
    Headers         map[string]string
    RequireApproval string // "never", "always"
}

// HostedImageGenerationTool represents a provider-hosted image generation capability.
type HostedImageGenerationTool struct {
    ToolName           string
    ToolDescription    string
    Quality            string // "low", "medium", "high", "auto"
    OutputFormat       string // "png", "jpeg", "webp"
    OutputCompression  int    // 0-100 for JPEG/WebP
    Background         string // "transparent", "opaque", "auto"
    Size               string // "1024x1024", "1792x1024", etc.
    PartialImages      bool
}
```

Success criteria:

* All hosted tools implement HostedTool interface
* ProviderConfig returns provider-specific JSON representation
* Configuration options match Python implementations

Context references:

* [python/packages/core/agent_framework/_tools.py](python/packages/core/agent_framework/_tools.py) (Lines 239-453) - Hosted tool classes

Dependencies:

* Step 2.1.1 completion

---

### Step 2.1.4: Implement function invocation utilities

Create utilities for safe function invocation with error handling.

Files:

* `go/tool/invoke.go` - Invocation utilities
* `go/tool/errors.go` - Tool-specific errors

#### Invocation Utilities

```go
// tool/invoke.go

// Invoker handles tool invocation with error handling and context support.
type Invoker struct {
    config InvocationConfig
    tools  map[string]Tool
}

// NewInvoker creates an Invoker with the given tools and configuration.
func NewInvoker(tools []Tool, config InvocationConfig) *Invoker

// Invoke executes a tool by name with the given arguments.
// Handles panic recovery, timeout, and error wrapping.
func (i *Invoker) Invoke(ctx context.Context, name string, arguments json.RawMessage) (Result, error) {
    tool, ok := i.tools[name]
    if !ok {
        if i.config.TerminateOnUnknownCalls {
            return Result{}, ErrUnknownTool
        }
        return Result{
            Content: fmt.Sprintf("Unknown tool: %s", name),
            IsError: true,
        }, nil
    }
    
    // Execute with panic recovery
    return i.invokeWithRecovery(ctx, tool, arguments)
}

func (i *Invoker) invokeWithRecovery(ctx context.Context, tool Tool, args json.RawMessage) (result Result, err error) {
    defer func() {
        if r := recover(); r != nil {
            err = &InvocationPanicError{
                ToolName: tool.Name(),
                Panic:    r,
                Stack:    debug.Stack(),
            }
            if i.config.IncludeDetailedErrors {
                result = Result{Content: fmt.Sprintf("Tool panicked: %v", r), IsError: true}
            } else {
                result = Result{Content: "Tool invocation failed", IsError: true}
            }
        }
    }()
    
    return tool.Invoke(ctx, args)
}

// InvokeBatch executes multiple tool calls, optionally in parallel.
func (i *Invoker) InvokeBatch(ctx context.Context, calls []ToolCall, parallel bool) []InvocationResult

type ToolCall struct {
    ID        string
    Name      string
    Arguments json.RawMessage
}

type InvocationResult struct {
    CallID string
    Result Result
    Error  error
}
```

#### Error Types

```go
// tool/errors.go

var (
    ErrUnknownTool       = errors.New("unknown tool")
    ErrInvalidArguments  = errors.New("invalid arguments")
    ErrMaxIterations     = errors.New("max iterations exceeded")
    ErrConsecutiveErrors = errors.New("max consecutive errors exceeded")
)

// InvocationError wraps errors from tool invocation.
type InvocationError struct {
    ToolName string
    Cause    error
}

// InvocationPanicError captures panic details from tool invocation.
type InvocationPanicError struct {
    ToolName string
    Panic    interface{}
    Stack    []byte
}
```

Success criteria:

* Invoker handles tool lookup, invocation, and error recovery
* Panic recovery prevents tool failures from crashing the agent
* Batch invocation supports parallel execution for independent calls
* Error types provide detailed diagnostic information

Context references:

* [dotnet/src/Microsoft.Agents.AI/FunctionInvocationDelegatingAgent.cs](dotnet/src/Microsoft.Agents.AI/FunctionInvocationDelegatingAgent.cs) - Function middleware pattern

Dependencies:

* Steps 2.1.1-2.1.3 completion

---

### Step 2.1.5: Add tool package tests

Create comprehensive tests for the tool package.

Files:

* `go/tool/tool_test.go` - Interface tests
* `go/tool/function_test.go` - FunctionTool tests
* `go/tool/schema_test.go` - Schema generation tests
* `go/tool/invoke_test.go` - Invoker tests

Test coverage requirements:

* FunctionTool creation with valid and invalid signatures
* Schema generation for various Go types
* Hosted tool configuration serialization
* Invoker error handling and panic recovery
* Batch invocation with parallel execution

Success criteria:

* 90%+ code coverage for tool package
* Table-driven tests for schema generation
* Mock-based tests for invocation scenarios

---

## Feature 2.2: OpenAI Provider Package

<!-- parallelizable: true -->

### Step 2.2.1: Create OpenAI client structure and options

Create the OpenAI provider package with client configuration.

Files:

* `go/providers/openai/client.go` - Client struct
* `go/providers/openai/options.go` - Configuration options
* `go/providers/openai/doc.go` - Package documentation

#### Client Structure

```go
// providers/openai/client.go

// Client implements chat.Client for OpenAI's Chat Completions API.
// Aligns with Python OpenAIChatClient and .NET OpenAIChatClientExtensions.
type Client struct {
    client   *openai.Client
    model    string
    metadata chat.ClientMetadata
    
    // Configuration
    instructionRole string // "system" or "developer"
}

// NewClient creates a new OpenAI chat client.
func NewClient(opts ...Option) (*Client, error) {
    cfg := defaultConfig()
    for _, opt := range opts {
        opt(cfg)
    }
    
    if cfg.apiKey == "" {
        cfg.apiKey = os.Getenv("OPENAI_API_KEY")
    }
    if cfg.apiKey == "" {
        return nil, errors.New("API key required: use WithAPIKey or set OPENAI_API_KEY")
    }
    
    clientCfg := openai.DefaultConfig(cfg.apiKey)
    if cfg.baseURL != "" {
        clientCfg.BaseURL = cfg.baseURL
    }
    if cfg.orgID != "" {
        clientCfg.OrgID = cfg.orgID
    }
    
    return &Client{
        client:          openai.NewClientWithConfig(clientCfg),
        model:           cfg.model,
        instructionRole: cfg.instructionRole,
        metadata: chat.ClientMetadata{
            ProviderName: "openai",
            ModelID:      cfg.model,
            EndpointURI:  clientCfg.BaseURL,
        },
    }, nil
}
```

#### Options Pattern

```go
// providers/openai/options.go

type config struct {
    apiKey          string
    baseURL         string
    orgID           string
    model           string
    instructionRole string
    httpClient      *http.Client
}

func defaultConfig() *config {
    return &config{
        model:           "gpt-4o",
        instructionRole: "system",
    }
}

type Option func(*config)

// WithAPIKey sets the OpenAI API key.
func WithAPIKey(key string) Option {
    return func(c *config) { c.apiKey = key }
}

// WithModel sets the model to use (e.g., "gpt-4o", "gpt-4o-mini").
func WithModel(model string) Option {
    return func(c *config) { c.model = model }
}

// WithBaseURL sets a custom API endpoint.
func WithBaseURL(url string) Option {
    return func(c *config) { c.baseURL = url }
}

// WithOrgID sets the OpenAI organization ID.
func WithOrgID(orgID string) Option {
    return func(c *config) { c.orgID = orgID }
}

// WithInstructionRole sets the role for system instructions ("system" or "developer").
func WithInstructionRole(role string) Option {
    return func(c *config) { c.instructionRole = role }
}

// WithHTTPClient sets a custom HTTP client for requests.
func WithHTTPClient(client *http.Client) Option {
    return func(c *config) { c.httpClient = client }
}
```

Success criteria:

* Client implements chat.Client interface
* Options pattern follows Go conventions
* Environment variable fallback for API key
* Metadata correctly populated

Context references:

* [python/packages/core/agent_framework/openai/_shared.py](python/packages/core/agent_framework/openai/_shared.py) (Lines 73-117) - OpenAISettings
* [dotnet/src/Microsoft.Agents.AI.OpenAI/Extensions/OpenAIChatClientExtensions.cs](dotnet/src/Microsoft.Agents.AI.OpenAI/Extensions/OpenAIChatClientExtensions.cs) - Extension pattern

Dependencies:

* Feature 2.1 completion (tool types)

---

### Step 2.2.2: Implement Chat Completions API integration

Implement non-streaming chat completion.

Files:

* `go/providers/openai/client.go` - GetResponse method
* `go/providers/openai/convert.go` - Message conversion utilities

#### Message Conversion

```go
// providers/openai/convert.go

// toOpenAIMessages converts chat.Message slice to OpenAI format.
func toOpenAIMessages(messages []chat.Message, instructionRole string) []openai.ChatCompletionMessage {
    result := make([]openai.ChatCompletionMessage, 0, len(messages))
    for _, msg := range messages {
        result = append(result, toOpenAIMessage(msg, instructionRole))
    }
    return result
}

func toOpenAIMessage(msg chat.Message, instructionRole string) openai.ChatCompletionMessage {
    role := string(msg.Role)
    if msg.Role == chat.RoleSystem && instructionRole == "developer" {
        role = "developer"
    }
    
    oaiMsg := openai.ChatCompletionMessage{
        Role: role,
        Name: msg.Name,
    }
    
    // Handle content types
    if len(msg.Contents) == 1 {
        if tc, ok := msg.Contents[0].(chat.TextContent); ok {
            oaiMsg.Content = tc.Text
            return oaiMsg
        }
    }
    
    // Multi-content or non-text content
    oaiMsg.MultiContent = toOpenAIContentParts(msg.Contents)
    
    // Handle tool calls
    if len(msg.ToolCalls) > 0 {
        oaiMsg.ToolCalls = toOpenAIToolCalls(msg.ToolCalls)
    }
    
    // Handle tool result
    if msg.ToolCallID != "" {
        oaiMsg.ToolCallID = msg.ToolCallID
    }
    
    return oaiMsg
}

func toOpenAIContentParts(contents []chat.Content) []openai.ChatMessagePart {
    // Convert TextContent, ImageContent to OpenAI parts
}

func toOpenAIToolCalls(calls []chat.ToolCall) []openai.ToolCall {
    // Convert chat.ToolCall to openai.ToolCall
}

// fromOpenAIMessage converts OpenAI response to chat.Message.
func fromOpenAIMessage(choice openai.ChatCompletionChoice) chat.Message {
    msg := chat.Message{
        Role:     chat.Role(choice.Message.Role),
        Contents: []chat.Content{chat.TextContent{Text: choice.Message.Content}},
    }
    
    if len(choice.Message.ToolCalls) > 0 {
        msg.ToolCalls = fromOpenAIToolCalls(choice.Message.ToolCalls)
    }
    
    return msg
}
```

#### GetResponse Implementation

```go
// providers/openai/client.go

func (c *Client) GetResponse(ctx context.Context, messages []chat.Message, options *chat.Options) (*chat.Response, error) {
    req := c.buildRequest(messages, options)
    
    resp, err := c.client.CreateChatCompletion(ctx, req)
    if err != nil {
        return nil, fmt.Errorf("openai: chat completion failed: %w", err)
    }
    
    if len(resp.Choices) == 0 {
        return nil, errors.New("openai: no choices returned")
    }
    
    choice := resp.Choices[0]
    return &chat.Response{
        Message:      fromOpenAIMessage(choice),
        FinishReason: chat.FinishReason(choice.FinishReason),
        Usage: chat.UsageDetails{
            InputTokens:  resp.Usage.PromptTokens,
            OutputTokens: resp.Usage.CompletionTokens,
            TotalTokens:  resp.Usage.TotalTokens,
        },
        RawRepresentation: resp,
    }, nil
}

func (c *Client) buildRequest(messages []chat.Message, options *chat.Options) openai.ChatCompletionRequest {
    req := openai.ChatCompletionRequest{
        Model:    c.model,
        Messages: toOpenAIMessages(messages, c.instructionRole),
    }
    
    if options != nil {
        if options.MaxTokens > 0 {
            req.MaxTokens = options.MaxTokens
        }
        if options.Temperature != 0 {
            req.Temperature = options.Temperature
        }
        if options.TopP != 0 {
            req.TopP = options.TopP
        }
        if len(options.StopSequences) > 0 {
            req.Stop = options.StopSequences
        }
    }
    
    return req
}
```

Success criteria:

* Message conversion handles all content types
* GetResponse returns properly structured chat.Response
* Usage details correctly populated
* Error handling includes context

Dependencies:

* Step 2.2.1 completion

---

### Step 2.2.3: Implement streaming response handling

Implement streaming chat completion with Go channels.

Files:

* `go/providers/openai/client.go` - GetStreamingResponse method
* `go/providers/openai/stream.go` - Stream processing utilities

#### Streaming Implementation

```go
// providers/openai/client.go

func (c *Client) GetStreamingResponse(ctx context.Context, messages []chat.Message, options *chat.Options) (<-chan chat.ResponseUpdate, error) {
    req := c.buildRequest(messages, options)
    req.Stream = true
    
    stream, err := c.client.CreateChatCompletionStream(ctx, req)
    if err != nil {
        return nil, fmt.Errorf("openai: stream creation failed: %w", err)
    }
    
    updates := make(chan chat.ResponseUpdate, 32) // Buffered for backpressure
    
    go func() {
        defer close(updates)
        defer stream.Close()
        
        c.processStream(ctx, stream, updates)
    }()
    
    return updates, nil
}

func (c *Client) processStream(ctx context.Context, stream *openai.ChatCompletionStream, updates chan<- chat.ResponseUpdate) {
    for {
        select {
        case <-ctx.Done():
            updates <- chat.ResponseUpdate{
                Kind:  chat.UpdateKindError,
                Error: ctx.Err(),
            }
            return
        default:
        }
        
        chunk, err := stream.Recv()
        if errors.Is(err, io.EOF) {
            updates <- chat.ResponseUpdate{Kind: chat.UpdateKindDone}
            return
        }
        if err != nil {
            updates <- chat.ResponseUpdate{
                Kind:  chat.UpdateKindError,
                Error: fmt.Errorf("openai: stream error: %w", err),
            }
            return
        }
        
        // Process chunk
        for _, choice := range chunk.Choices {
            update := c.parseStreamChunk(choice)
            updates <- update
            
            if choice.FinishReason != "" {
                updates <- chat.ResponseUpdate{
                    Kind:         chat.UpdateKindMessageComplete,
                    FinishReason: chat.FinishReason(choice.FinishReason),
                }
            }
        }
        
        // Usage in final chunk (if enabled)
        if chunk.Usage != nil {
            updates <- chat.ResponseUpdate{
                Kind: chat.UpdateKindUsage,
                Usage: &chat.UsageDetails{
                    InputTokens:  chunk.Usage.PromptTokens,
                    OutputTokens: chunk.Usage.CompletionTokens,
                    TotalTokens:  chunk.Usage.TotalTokens,
                },
            }
        }
    }
}

func (c *Client) parseStreamChunk(choice openai.ChatCompletionStreamChoice) chat.ResponseUpdate {
    delta := choice.Delta
    
    update := chat.ResponseUpdate{
        Kind: chat.UpdateKindContentDelta,
        Delta: &chat.ContentDelta{
            Role:      chat.Role(delta.Role),
            TextDelta: delta.Content,
        },
    }
    
    // Handle tool calls in delta
    if len(delta.ToolCalls) > 0 {
        for _, tc := range delta.ToolCalls {
            if tc.Function.Name != "" {
                update.Kind = chat.UpdateKindToolCall
                update.ToolCall = &chat.ToolCall{
                    ID:   tc.ID,
                    Name: tc.Function.Name,
                }
            }
            if tc.Function.Arguments != "" {
                update.Delta.ArgsDelta = tc.Function.Arguments
                update.Delta.ToolCallID = tc.ID
            }
        }
    }
    
    return update
}
```

Success criteria:

* Streaming uses buffered channels for backpressure handling
* Context cancellation properly stops the stream
* All update kinds (ContentDelta, ToolCall, MessageComplete, Usage, Error, Done) handled
* Goroutine cleanup on completion or error

Dependencies:

* Step 2.2.2 completion

---

### Step 2.2.4: Implement tool calling support

Add tool definitions and tool call handling to the OpenAI client.

Files:

* `go/providers/openai/tools.go` - Tool conversion utilities
* `go/providers/openai/client.go` - Tool-enabled request building

#### Tool Conversion

```go
// providers/openai/tools.go

import "github.com/microsoft/agent-framework-go/tool"

// toOpenAITools converts tool.Tool slice to OpenAI tool definitions.
func toOpenAITools(tools []tool.Tool) []openai.Tool {
    result := make([]openai.Tool, 0, len(tools))
    
    for _, t := range tools {
        // Skip hosted tools - they're added differently
        if ht, ok := t.(tool.HostedTool); ok && ht.IsHosted() {
            continue
        }
        
        result = append(result, openai.Tool{
            Type: openai.ToolTypeFunction,
            Function: &openai.FunctionDefinition{
                Name:        t.Name(),
                Description: t.Description(),
                Parameters:  t.Parameters(),
            },
        })
    }
    
    return result
}

// toOpenAIHostedTools returns hosted tool configurations for the request.
func toOpenAIHostedTools(tools []tool.Tool) []interface{} {
    var hosted []interface{}
    
    for _, t := range tools {
        if ht, ok := t.(tool.HostedTool); ok && ht.IsHosted() {
            hosted = append(hosted, ht.ProviderConfig())
        }
    }
    
    return hosted
}
```

#### Tool-Enabled Requests

```go
// providers/openai/client.go - extend buildRequest

func (c *Client) buildRequest(messages []chat.Message, options *chat.Options) openai.ChatCompletionRequest {
    req := openai.ChatCompletionRequest{
        Model:    c.model,
        Messages: toOpenAIMessages(messages, c.instructionRole),
    }
    
    // ... existing option handling ...
    
    // Add tools if provided
    if options != nil && len(options.Tools) > 0 {
        oaiTools := toOpenAITools(options.Tools)
        if len(oaiTools) > 0 {
            req.Tools = oaiTools
        }
        
        // Handle tool choice
        switch options.ToolChoice {
        case "auto":
            req.ToolChoice = "auto"
        case "required":
            req.ToolChoice = "required"
        case "none":
            req.ToolChoice = "none"
        default:
            if options.ToolChoice != "" {
                req.ToolChoice = openai.ToolChoice{
                    Type: openai.ToolTypeFunction,
                    Function: openai.ToolFunction{
                        Name: options.ToolChoice,
                    },
                }
            }
        }
    }
    
    return req
}
```

Success criteria:

* Function tools converted to OpenAI format
* Hosted tools separated for provider-specific handling
* Tool choice modes (auto, required, none, specific) supported
* JSON Schema parameters preserved

Dependencies:

* Steps 2.2.1-2.2.3 completion
* Feature 2.1 completion (tool types)

---

### Step 2.2.5: Implement Responses API client

Create a separate client for OpenAI's Responses API (stateful conversations).

Files:

* `go/providers/openai/responses.go` - Responses API client
* `go/providers/openai/responses_options.go` - Responses-specific options

#### Responses Client

```go
// providers/openai/responses.go

// ResponsesClient implements chat.Client for OpenAI's Responses API.
// Supports stateful conversations and hosted tools (web search, code interpreter).
type ResponsesClient struct {
    client   *openai.Client
    model    string
    metadata chat.ClientMetadata
    
    // Responses API specific
    previousResponseID string
}

// NewResponsesClient creates a new Responses API client.
func NewResponsesClient(opts ...ResponsesOption) (*ResponsesClient, error)

func (c *ResponsesClient) GetResponse(ctx context.Context, messages []chat.Message, options *chat.Options) (*chat.Response, error) {
    // Build Responses API request
    req := c.buildResponsesRequest(messages, options)
    
    // If we have a previous response ID, use it for continuation
    if c.previousResponseID != "" {
        req.PreviousResponseID = c.previousResponseID
    }
    
    resp, err := c.client.CreateResponse(ctx, req)
    if err != nil {
        return nil, fmt.Errorf("openai: responses API failed: %w", err)
    }
    
    // Store response ID for potential continuation
    c.previousResponseID = resp.ID
    
    return c.parseResponsesAPIResponse(resp)
}

func (c *ResponsesClient) buildResponsesRequest(messages []chat.Message, options *chat.Options) openai.CreateResponseRequest {
    req := openai.CreateResponseRequest{
        Model: c.model,
        Input: toResponsesInput(messages),
    }
    
    // Add hosted tools
    if options != nil && len(options.Tools) > 0 {
        for _, t := range options.Tools {
            if ht, ok := t.(tool.HostedTool); ok && ht.IsHosted() {
                switch ht.(type) {
                case *tool.HostedWebSearchTool:
                    req.Tools = append(req.Tools, map[string]interface{}{
                        "type": "web_search_preview",
                        // ... configuration
                    })
                case *tool.HostedCodeInterpreterTool:
                    req.Tools = append(req.Tools, map[string]interface{}{
                        "type": "code_interpreter",
                        // ... configuration
                    })
                }
            }
        }
    }
    
    return req
}
```

Success criteria:

* ResponsesClient implements chat.Client interface
* Stateful conversation continuation via response IDs
* Hosted tools (web search, code interpreter) configured correctly
* Response parsing handles Responses API format

Context references:

* [dotnet/src/Microsoft.Agents.AI.OpenAI/Extensions/OpenAIResponseClientExtensions.cs](dotnet/src/Microsoft.Agents.AI.OpenAI/Extensions/OpenAIResponseClientExtensions.cs) - Responses API pattern

Dependencies:

* Steps 2.2.1-2.2.4 completion

---

### Step 2.2.6: Add OpenAI provider tests

Create comprehensive tests for the OpenAI provider.

Files:

* `go/providers/openai/client_test.go` - Client unit tests
* `go/providers/openai/convert_test.go` - Conversion tests
* `go/providers/openai/integration_test.go` - Integration tests (build tag: integration)

Test coverage requirements:

* Client creation with various option combinations
* Message conversion for all content types
* Streaming response processing
* Tool call handling
* Error scenarios (rate limiting, network errors)

Success criteria:

* 90%+ code coverage for unit tests
* Integration tests with build tags
* Mock client for unit testing

---

## Feature 2.3: Azure OpenAI Provider Package

<!-- parallelizable: true -->

### Step 2.3.1: Create Azure OpenAI client structure and options

Create the Azure OpenAI provider package with configuration.

Files:

* `go/providers/azure/client.go` - Azure OpenAI client
* `go/providers/azure/options.go` - Configuration options
* `go/providers/azure/doc.go` - Package documentation

#### Client Structure

```go
// providers/azure/client.go

import (
    "github.com/Azure/azure-sdk-for-go/sdk/ai/azopenai"
    "github.com/Azure/azure-sdk-for-go/sdk/azcore"
)

// Client implements chat.Client for Azure OpenAI Service.
type Client struct {
    client     *azopenai.Client
    deployment string
    metadata   chat.ClientMetadata
}

// NewClient creates a new Azure OpenAI chat client.
func NewClient(opts ...Option) (*Client, error) {
    cfg := defaultConfig()
    for _, opt := range opts {
        opt(cfg)
    }
    
    var client *azopenai.Client
    var err error
    
    if cfg.credential != nil {
        // Azure AD authentication
        client, err = azopenai.NewClient(cfg.endpoint, cfg.credential, nil)
    } else if cfg.apiKey != "" {
        // API key authentication
        keyCredential := azcore.NewKeyCredential(cfg.apiKey)
        client, err = azopenai.NewClientWithKeyCredential(cfg.endpoint, keyCredential, nil)
    } else {
        return nil, errors.New("credential or API key required")
    }
    
    if err != nil {
        return nil, fmt.Errorf("azure: failed to create client: %w", err)
    }
    
    return &Client{
        client:     client,
        deployment: cfg.deployment,
        metadata: chat.ClientMetadata{
            ProviderName: "azure",
            ModelID:      cfg.deployment,
            EndpointURI:  cfg.endpoint,
        },
    }, nil
}
```

#### Options Pattern

```go
// providers/azure/options.go

import "github.com/Azure/azure-sdk-for-go/sdk/azcore"

type config struct {
    endpoint    string
    deployment  string
    apiKey      string
    credential  azcore.TokenCredential
    apiVersion  string
}

type Option func(*config)

// WithEndpoint sets the Azure OpenAI endpoint URL.
func WithEndpoint(endpoint string) Option

// WithDeployment sets the deployment name.
func WithDeployment(deployment string) Option

// WithAPIKey sets the API key for authentication.
func WithAPIKey(key string) Option

// WithTokenCredential sets an Azure AD credential for authentication.
func WithTokenCredential(cred azcore.TokenCredential) Option

// WithAPIVersion sets the API version (default: latest stable).
func WithAPIVersion(version string) Option
```

Success criteria:

* Client supports both API key and Azure AD authentication
* Environment variable fallbacks for endpoint and credentials
* Deployment name properly used in requests

Context references:

* [python/packages/core/agent_framework/azure/_chat_client.py](python/packages/core/agent_framework/azure/_chat_client.py) - Azure OpenAI client

Dependencies:

* Feature 2.1 completion

---

### Step 2.3.2: Implement Azure AD authentication

Add Azure AD authentication support with token refresh.

Files:

* `go/providers/azure/auth.go` - Authentication helpers
* `go/providers/azure/options.go` - Credential options

#### Authentication Helpers

```go
// providers/azure/auth.go

import (
    "github.com/Azure/azure-sdk-for-go/sdk/azidentity"
)

// NewDefaultCredential creates a DefaultAzureCredential for Azure AD auth.
// This credential chain tries:
// 1. Environment variables
// 2. Workload Identity
// 3. Managed Identity
// 4. Azure CLI
// 5. Azure Developer CLI
func NewDefaultCredential() (azcore.TokenCredential, error) {
    return azidentity.NewDefaultAzureCredential(nil)
}

// NewManagedIdentityCredential creates a credential for Azure Managed Identity.
func NewManagedIdentityCredential(clientID string) (azcore.TokenCredential, error) {
    opts := &azidentity.ManagedIdentityCredentialOptions{}
    if clientID != "" {
        opts.ID = azidentity.ClientID(clientID)
    }
    return azidentity.NewManagedIdentityCredential(opts)
}

// NewClientSecretCredential creates a credential for service principal auth.
func NewClientSecretCredential(tenantID, clientID, clientSecret string) (azcore.TokenCredential, error) {
    return azidentity.NewClientSecretCredential(tenantID, clientID, clientSecret, nil)
}
```

#### Credential Options

```go
// providers/azure/options.go - additional options

// WithDefaultCredential uses DefaultAzureCredential for authentication.
func WithDefaultCredential() Option {
    return func(c *config) {
        cred, err := NewDefaultCredential()
        if err == nil {
            c.credential = cred
        }
    }
}

// WithManagedIdentity uses Managed Identity with optional client ID.
func WithManagedIdentity(clientID string) Option {
    return func(c *config) {
        cred, err := NewManagedIdentityCredential(clientID)
        if err == nil {
            c.credential = cred
        }
    }
}
```

Success criteria:

* DefaultAzureCredential works in Azure-hosted environments
* Managed Identity supports user-assigned identities
* Token refresh handled automatically by SDK

Context references:

* [dotnet/src/Microsoft.Agents.AI.AzureAI/AzureAIProjectChatClient.cs](dotnet/src/Microsoft.Agents.AI.AzureAI/AzureAIProjectChatClient.cs) - Azure authentication pattern

Dependencies:

* Step 2.3.1 completion

---

### Step 2.3.3: Implement Azure OpenAI completions and streaming

Implement chat completion methods for Azure OpenAI.

Files:

* `go/providers/azure/client.go` - GetResponse and GetStreamingResponse
* `go/providers/azure/convert.go` - Azure-specific conversions

Implementation follows same patterns as OpenAI client but uses Azure SDK types.

Success criteria:

* GetResponse uses Azure SDK chat completions
* GetStreamingResponse handles Azure streaming format
* "On Your Data" configuration supported in options
* Tool calling works identically to OpenAI

Dependencies:

* Steps 2.3.1-2.3.2 completion

---

### Step 2.3.4: Implement Azure AI Foundry agent client

Create client for Azure AI Foundry Agent Service.

Files:

* `go/providers/azure/foundry.go` - Foundry agent client
* `go/providers/azure/foundry_options.go` - Foundry-specific options

#### Foundry Client

```go
// providers/azure/foundry.go

// FoundryClient implements chat.Client for Azure AI Foundry agents.
type FoundryClient struct {
    projectClient *aiprojects.Client
    agentRef      AgentReference
    metadata      chat.ClientMetadata
}

type AgentReference struct {
    AgentID     string
    AgentName   string
    Description string
}

// NewFoundryClient creates a client for an existing Foundry agent.
func NewFoundryClient(projectEndpoint string, agentRef AgentReference, opts ...FoundryOption) (*FoundryClient, error)

// GetOrCreateAgent retrieves an existing agent by name or creates a new one.
func GetOrCreateAgent(projectEndpoint, name, model, instructions string, opts ...FoundryOption) (*FoundryClient, error)

// CreateAgent creates a new agent in Azure AI Foundry.
func CreateAgent(projectEndpoint, name, model, instructions string, opts ...FoundryOption) (*FoundryClient, error)

func (c *FoundryClient) GetResponse(ctx context.Context, messages []chat.Message, options *chat.Options) (*chat.Response, error) {
    // Use Foundry agent API
}

func (c *FoundryClient) GetStreamingResponse(ctx context.Context, messages []chat.Message, options *chat.Options) (<-chan chat.ResponseUpdate, error) {
    // Use Foundry agent streaming API
}
```

Success criteria:

* FoundryClient implements chat.Client
* Agent creation and retrieval supported
* Conversation threading via agent threads

Context references:

* [dotnet/src/Microsoft.Agents.AI.AzureAI/AzureAIProjectChatClientExtensions.cs](dotnet/src/Microsoft.Agents.AI.AzureAI/AzureAIProjectChatClientExtensions.cs) - Foundry extensions

Dependencies:

* Steps 2.3.1-2.3.3 completion

---

### Step 2.3.5: Implement Persistent Agents support

Add support for Azure AI Foundry Persistent Agents.

Files:

* `go/providers/azure/persistent.go` - Persistent agent client
* `go/providers/azure/persistent_options.go` - Persistent agent options

#### Persistent Agent Client

```go
// providers/azure/persistent.go

// PersistentClient implements chat.Client for Azure AI Persistent Agents.
// Persistent agents maintain state and threads server-side.
type PersistentClient struct {
    client     *persistentagents.Client
    agentID    string
    threadID   string
    metadata   chat.ClientMetadata
}

// NewPersistentClient creates a client for an existing persistent agent.
func NewPersistentClient(endpoint, agentID string, opts ...PersistentOption) (*PersistentClient, error)

// CreatePersistentAgent creates a new persistent agent.
func CreatePersistentAgent(endpoint, name, model, instructions string, opts ...PersistentOption) (*PersistentClient, error)

// GetPersistentAgent retrieves an existing persistent agent by ID.
func GetPersistentAgent(endpoint, agentID string, opts ...PersistentOption) (*PersistentClient, error)

func (c *PersistentClient) GetResponse(ctx context.Context, messages []chat.Message, options *chat.Options) (*chat.Response, error) {
    // Create thread if needed
    if c.threadID == "" {
        thread, err := c.client.CreateThread(ctx, nil)
        if err != nil {
            return nil, fmt.Errorf("azure: failed to create thread: %w", err)
        }
        c.threadID = thread.ID
    }
    
    // Add messages to thread
    for _, msg := range messages {
        _, err := c.client.CreateMessage(ctx, c.threadID, toThreadMessage(msg))
        if err != nil {
            return nil, fmt.Errorf("azure: failed to add message: %w", err)
        }
    }
    
    // Run the agent
    run, err := c.client.CreateRun(ctx, c.threadID, c.agentID, nil)
    if err != nil {
        return nil, fmt.Errorf("azure: failed to create run: %w", err)
    }
    
    // Wait for completion
    return c.waitForRun(ctx, run.ID)
}
```

Success criteria:

* Persistent agents maintain server-side threads
* Agent creation, retrieval, and deletion supported
* Run polling with timeout handling

Context references:

* [dotnet/src/Microsoft.Agents.AI.AzureAI.Persistent/PersistentAgentsClientExtensions.cs](dotnet/src/Microsoft.Agents.AI.AzureAI.Persistent/PersistentAgentsClientExtensions.cs) - Persistent agents

Dependencies:

* Steps 2.3.1-2.3.4 completion

---

### Step 2.3.6: Add Azure provider tests

Create comprehensive tests for the Azure provider.

Files:

* `go/providers/azure/client_test.go` - Client unit tests
* `go/providers/azure/auth_test.go` - Authentication tests
* `go/providers/azure/integration_test.go` - Integration tests

Test coverage requirements:

* Both authentication methods (API key, Azure AD)
* Chat completion and streaming
* Foundry and Persistent agent clients
* Error scenarios

---

## Feature 2.4: Anthropic Provider Package

<!-- parallelizable: true -->

### Step 2.4.1: Create Anthropic client structure and options

Create the Anthropic provider package.

Files:

* `go/providers/anthropic/client.go` - Anthropic client
* `go/providers/anthropic/options.go` - Configuration options
* `go/providers/anthropic/doc.go` - Package documentation

#### Client Structure

```go
// providers/anthropic/client.go

import anthropic "github.com/liushuangls/go-anthropic/v2"

// Client implements chat.Client for Anthropic's Messages API.
type Client struct {
    client   *anthropic.Client
    model    string
    metadata chat.ClientMetadata
    
    // Anthropic-specific
    maxTokens       int    // Required for Anthropic
    additionalBetas []string
}

// NewClient creates a new Anthropic chat client.
func NewClient(opts ...Option) (*Client, error) {
    cfg := defaultConfig()
    for _, opt := range opts {
        opt(cfg)
    }
    
    if cfg.apiKey == "" {
        cfg.apiKey = os.Getenv("ANTHROPIC_API_KEY")
    }
    if cfg.apiKey == "" {
        return nil, errors.New("API key required: use WithAPIKey or set ANTHROPIC_API_KEY")
    }
    
    client := anthropic.NewClient(cfg.apiKey)
    
    return &Client{
        client:    client,
        model:     cfg.model,
        maxTokens: cfg.maxTokens,
        metadata: chat.ClientMetadata{
            ProviderName: "anthropic",
            ModelID:      cfg.model,
            EndpointURI:  "https://api.anthropic.com",
        },
    }, nil
}
```

#### Options

```go
// providers/anthropic/options.go

type config struct {
    apiKey          string
    model           string
    maxTokens       int
    additionalBetas []string
}

func defaultConfig() *config {
    return &config{
        model:     "claude-sonnet-4-20250514",
        maxTokens: 4096, // Required for Anthropic
    }
}

type Option func(*config)

func WithAPIKey(key string) Option
func WithModel(model string) Option
func WithMaxTokens(tokens int) Option
func WithBetaFlags(flags ...string) Option
```

Success criteria:

* Client implements chat.Client
* MaxTokens properly defaulted (Anthropic requires it)
* Model selection for Claude variants

Context references:

* [python/packages/anthropic/agent_framework_anthropic/_chat_client.py](python/packages/anthropic/agent_framework_anthropic/_chat_client.py) - Anthropic client

Dependencies:

* Feature 2.1 completion

---

### Step 2.4.2: Implement message format conversion

Convert between chat.Message and Anthropic message format.

Files:

* `go/providers/anthropic/convert.go` - Message conversions

#### Message Conversion

```go
// providers/anthropic/convert.go

// toAnthropicMessages converts chat.Message slice to Anthropic format.
// Note: Anthropic requires system messages as a separate parameter.
func toAnthropicMessages(messages []chat.Message) (systemPrompt string, msgs []anthropic.Message) {
    for _, msg := range messages {
        switch msg.Role {
        case chat.RoleSystem:
            // Anthropic takes system as separate parameter
            if text := extractText(msg); text != "" {
                systemPrompt += text + "\n"
            }
        case chat.RoleUser:
            msgs = append(msgs, anthropic.Message{
                Role:    anthropic.RoleUser,
                Content: toAnthropicContent(msg.Contents),
            })
        case chat.RoleAssistant:
            msgs = append(msgs, anthropic.Message{
                Role:    anthropic.RoleAssistant,
                Content: toAnthropicContent(msg.Contents),
            })
        case chat.RoleTool:
            // Tool results in Anthropic format
            msgs = append(msgs, anthropic.Message{
                Role: anthropic.RoleUser,
                Content: []anthropic.ContentBlock{{
                    Type:       "tool_result",
                    ToolUseID:  msg.ToolCallID,
                    Content:    extractText(msg),
                }},
            })
        }
    }
    return strings.TrimSpace(systemPrompt), msgs
}

func toAnthropicContent(contents []chat.Content) []anthropic.ContentBlock {
    var blocks []anthropic.ContentBlock
    
    for _, c := range contents {
        switch content := c.(type) {
        case chat.TextContent:
            blocks = append(blocks, anthropic.ContentBlock{
                Type: "text",
                Text: content.Text,
            })
        case chat.ImageContent:
            blocks = append(blocks, anthropic.ContentBlock{
                Type: "image",
                Source: &anthropic.ImageSource{
                    Type:      "base64",
                    MediaType: content.MediaType,
                    Data:      content.Base64Data,
                },
            })
        case chat.ToolCallContent:
            blocks = append(blocks, anthropic.ContentBlock{
                Type:  "tool_use",
                ID:    content.ToolCall.ID,
                Name:  content.ToolCall.Name,
                Input: content.ToolCall.Arguments,
            })
        }
    }
    
    return blocks
}

// fromAnthropicMessage converts Anthropic response to chat.Message.
func fromAnthropicMessage(resp *anthropic.MessagesResponse) chat.Message {
    msg := chat.Message{
        Role:     chat.RoleAssistant,
        Contents: make([]chat.Content, 0, len(resp.Content)),
    }
    
    for _, block := range resp.Content {
        switch block.Type {
        case "text":
            msg.Contents = append(msg.Contents, chat.TextContent{Text: block.Text})
        case "tool_use":
            tc := chat.ToolCall{
                ID:        block.ID,
                Name:      block.Name,
                Arguments: block.Input,
            }
            msg.ToolCalls = append(msg.ToolCalls, tc)
            msg.Contents = append(msg.Contents, chat.ToolCallContent{ToolCall: tc})
        }
    }
    
    return msg
}
```

Success criteria:

* System messages extracted as separate parameter
* Tool use blocks properly converted
* Image content uses base64 format

Dependencies:

* Step 2.4.1 completion

---

### Step 2.4.3: Implement SSE streaming

Implement Server-Sent Events streaming for Anthropic.

Files:

* `go/providers/anthropic/client.go` - GetStreamingResponse
* `go/providers/anthropic/stream.go` - SSE event parsing

#### Streaming Implementation

```go
// providers/anthropic/client.go

func (c *Client) GetStreamingResponse(ctx context.Context, messages []chat.Message, options *chat.Options) (<-chan chat.ResponseUpdate, error) {
    systemPrompt, anthropicMsgs := toAnthropicMessages(messages)
    
    req := anthropic.MessagesStreamRequest{
        Model:     c.model,
        Messages:  anthropicMsgs,
        MaxTokens: c.getMaxTokens(options),
        Stream:    true,
    }
    
    if systemPrompt != "" {
        req.System = systemPrompt
    }
    
    // Add tools
    if options != nil && len(options.Tools) > 0 {
        req.Tools = toAnthropicTools(options.Tools)
    }
    
    updates := make(chan chat.ResponseUpdate, 32)
    
    go func() {
        defer close(updates)
        
        stream, err := c.client.CreateMessagesStream(ctx, req)
        if err != nil {
            updates <- chat.ResponseUpdate{Kind: chat.UpdateKindError, Error: err}
            return
        }
        defer stream.Close()
        
        c.processAnthropicStream(ctx, stream, updates)
    }()
    
    return updates, nil
}

func (c *Client) processAnthropicStream(ctx context.Context, stream *anthropic.Stream, updates chan<- chat.ResponseUpdate) {
    for {
        select {
        case <-ctx.Done():
            updates <- chat.ResponseUpdate{Kind: chat.UpdateKindError, Error: ctx.Err()}
            return
        default:
        }
        
        event, err := stream.Recv()
        if errors.Is(err, io.EOF) {
            updates <- chat.ResponseUpdate{Kind: chat.UpdateKindDone}
            return
        }
        if err != nil {
            updates <- chat.ResponseUpdate{Kind: chat.UpdateKindError, Error: err}
            return
        }
        
        // Parse Anthropic events
        switch e := event.(type) {
        case *anthropic.MessageStartEvent:
            // Message metadata
        case *anthropic.ContentBlockStartEvent:
            if e.ContentBlock.Type == "tool_use" {
                updates <- chat.ResponseUpdate{
                    Kind: chat.UpdateKindToolCall,
                    ToolCall: &chat.ToolCall{
                        ID:   e.ContentBlock.ID,
                        Name: e.ContentBlock.Name,
                    },
                }
            }
        case *anthropic.ContentBlockDeltaEvent:
            if e.Delta.Type == "text_delta" {
                updates <- chat.ResponseUpdate{
                    Kind:  chat.UpdateKindContentDelta,
                    Delta: &chat.ContentDelta{TextDelta: e.Delta.Text},
                }
            } else if e.Delta.Type == "input_json_delta" {
                updates <- chat.ResponseUpdate{
                    Kind:  chat.UpdateKindContentDelta,
                    Delta: &chat.ContentDelta{ArgsDelta: e.Delta.PartialJSON},
                }
            }
        case *anthropic.MessageDeltaEvent:
            if e.Delta.StopReason != "" {
                updates <- chat.ResponseUpdate{
                    Kind:         chat.UpdateKindMessageComplete,
                    FinishReason: mapAnthropicStopReason(e.Delta.StopReason),
                }
            }
            if e.Usage != nil {
                updates <- chat.ResponseUpdate{
                    Kind: chat.UpdateKindUsage,
                    Usage: &chat.UsageDetails{
                        OutputTokens: e.Usage.OutputTokens,
                    },
                }
            }
        }
    }
}
```

Success criteria:

* SSE events properly parsed
* All event types handled (message_start, content_block_delta, etc.)
* Tool use blocks streamed correctly

Context references:

* [python/packages/anthropic/agent_framework_anthropic/_chat_client.py](python/packages/anthropic/agent_framework_anthropic/_chat_client.py) (Lines 354-360) - Streaming pattern

Dependencies:

* Steps 2.4.1-2.4.2 completion

---

### Step 2.4.4: Implement tool use blocks

Add tool calling support for Anthropic.

Files:

* `go/providers/anthropic/tools.go` - Tool conversion

#### Tool Conversion

```go
// providers/anthropic/tools.go

func toAnthropicTools(tools []tool.Tool) []anthropic.ToolDefinition {
    result := make([]anthropic.ToolDefinition, 0, len(tools))
    
    for _, t := range tools {
        if ht, ok := t.(tool.HostedTool); ok && ht.IsHosted() {
            continue // Anthropic hosted tools handled separately
        }
        
        result = append(result, anthropic.ToolDefinition{
            Name:        t.Name(),
            Description: t.Description(),
            InputSchema: t.Parameters(),
        })
    }
    
    return result
}

// toAnthropicHostedTools returns Anthropic-specific hosted tool configs.
func toAnthropicHostedTools(tools []tool.Tool) []anthropic.HostedTool {
    var hosted []anthropic.HostedTool
    
    for _, t := range tools {
        switch ht := t.(type) {
        case *tool.HostedCodeInterpreterTool:
            hosted = append(hosted, anthropic.HostedTool{
                Type: "code_execution",
                // Container config if provided
            })
        case *tool.HostedMCPTool:
            hosted = append(hosted, anthropic.HostedTool{
                Type:      "mcp",
                ServerURL: ht.ServerURL,
                // Additional config
            })
        }
    }
    
    return hosted
}
```

Success criteria:

* Tools converted to Anthropic input_schema format
* Hosted tools (code execution, MCP) supported
* Tool results properly formatted

Dependencies:

* Steps 2.4.1-2.4.3 completion

---

### Step 2.4.5: Add Anthropic provider tests

Create tests for the Anthropic provider.

Files:

* `go/providers/anthropic/client_test.go`
* `go/providers/anthropic/convert_test.go`
* `go/providers/anthropic/integration_test.go`

---

## Feature 2.5: AWS Bedrock Provider Package

<!-- parallelizable: true -->

### Step 2.5.1: Create Bedrock client structure and options

Create the AWS Bedrock provider package.

Files:

* `go/providers/bedrock/client.go` - Bedrock client
* `go/providers/bedrock/options.go` - Configuration options
* `go/providers/bedrock/doc.go` - Package documentation

#### Client Structure

```go
// providers/bedrock/client.go

import (
    "github.com/aws/aws-sdk-go-v2/config"
    "github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
)

// Client implements chat.Client for AWS Bedrock's Converse API.
type Client struct {
    client   *bedrockruntime.Client
    modelID  string
    metadata chat.ClientMetadata
}

// NewClient creates a new Bedrock chat client.
func NewClient(opts ...Option) (*Client, error) {
    cfg := defaultConfig()
    for _, opt := range opts {
        opt(cfg)
    }
    
    awsCfg, err := config.LoadDefaultConfig(context.Background(),
        config.WithRegion(cfg.region),
    )
    if err != nil {
        return nil, fmt.Errorf("bedrock: failed to load AWS config: %w", err)
    }
    
    client := bedrockruntime.NewFromConfig(awsCfg)
    
    return &Client{
        client:  client,
        modelID: cfg.modelID,
        metadata: chat.ClientMetadata{
            ProviderName: "aws.bedrock",
            ModelID:      cfg.modelID,
            EndpointURI:  fmt.Sprintf("https://bedrock-runtime.%s.amazonaws.com", cfg.region),
        },
    }, nil
}
```

Success criteria:

* Client uses AWS SDK v2
* Default credential chain resolution
* Region configuration

Context references:

* [python/packages/bedrock/agent_framework_bedrock/_chat_client.py](python/packages/bedrock/agent_framework_bedrock/_chat_client.py) - Bedrock client

Dependencies:

* Feature 2.1 completion

---

### Step 2.5.2: Implement AWS credential resolution

Add explicit credential configuration.

Files:

* `go/providers/bedrock/auth.go` - Credential helpers
* `go/providers/bedrock/options.go` - Credential options

#### Credential Options

```go
// providers/bedrock/options.go - additional options

// WithCredentials sets explicit AWS credentials.
func WithCredentials(accessKey, secretKey, sessionToken string) Option {
    return func(c *config) {
        c.accessKey = accessKey
        c.secretKey = secretKey
        c.sessionToken = sessionToken
    }
}

// WithProfile uses a specific AWS profile.
func WithProfile(profile string) Option {
    return func(c *config) {
        c.profile = profile
    }
}

// WithAssumeRole configures role assumption.
func WithAssumeRole(roleARN, externalID string) Option {
    return func(c *config) {
        c.roleARN = roleARN
        c.externalID = externalID
    }
}
```

Success criteria:

* Default credential chain (env, shared config, IAM role)
* Explicit credentials option
* Role assumption support

Dependencies:

* Step 2.5.1 completion

---

### Step 2.5.3: Implement Converse API integration

Implement chat completion using Bedrock's Converse API.

Files:

* `go/providers/bedrock/client.go` - GetResponse
* `go/providers/bedrock/convert.go` - Message conversions

#### Converse Implementation

```go
// providers/bedrock/client.go

func (c *Client) GetResponse(ctx context.Context, messages []chat.Message, options *chat.Options) (*chat.Response, error) {
    systemPrompts, converseMsgs := toBedrockMessages(messages)
    
    input := &bedrockruntime.ConverseInput{
        ModelId:  &c.modelID,
        Messages: converseMsgs,
    }
    
    if len(systemPrompts) > 0 {
        input.System = systemPrompts
    }
    
    // Add inference config
    if options != nil {
        input.InferenceConfig = &types.InferenceConfiguration{}
        if options.MaxTokens > 0 {
            input.InferenceConfig.MaxTokens = aws.Int32(int32(options.MaxTokens))
        }
        if options.Temperature != 0 {
            input.InferenceConfig.Temperature = aws.Float32(options.Temperature)
        }
    }
    
    // Add tools
    if options != nil && len(options.Tools) > 0 {
        input.ToolConfig = toBedrockToolConfig(options.Tools)
    }
    
    output, err := c.client.Converse(ctx, input)
    if err != nil {
        return nil, fmt.Errorf("bedrock: converse failed: %w", err)
    }
    
    return fromBedrockOutput(output)
}
```

Success criteria:

* Converse API properly used
* System prompts handled correctly
* Tool configuration included

Dependencies:

* Steps 2.5.1-2.5.2 completion

---

### Step 2.5.4: Implement streaming with chunk parsing

Implement streaming using ConverseStream API.

Files:

* `go/providers/bedrock/client.go` - GetStreamingResponse
* `go/providers/bedrock/stream.go` - Stream event parsing

#### Streaming Implementation

```go
// providers/bedrock/client.go

func (c *Client) GetStreamingResponse(ctx context.Context, messages []chat.Message, options *chat.Options) (<-chan chat.ResponseUpdate, error) {
    systemPrompts, converseMsgs := toBedrockMessages(messages)
    
    input := &bedrockruntime.ConverseStreamInput{
        ModelId:  &c.modelID,
        Messages: converseMsgs,
    }
    
    if len(systemPrompts) > 0 {
        input.System = systemPrompts
    }
    
    output, err := c.client.ConverseStream(ctx, input)
    if err != nil {
        return nil, fmt.Errorf("bedrock: stream failed: %w", err)
    }
    
    updates := make(chan chat.ResponseUpdate, 32)
    
    go func() {
        defer close(updates)
        
        stream := output.GetStream()
        for event := range stream.Events() {
            update := c.parseBedrockEvent(event)
            if update != nil {
                updates <- *update
            }
        }
        
        if err := stream.Err(); err != nil {
            updates <- chat.ResponseUpdate{Kind: chat.UpdateKindError, Error: err}
        } else {
            updates <- chat.ResponseUpdate{Kind: chat.UpdateKindDone}
        }
    }()
    
    return updates, nil
}

func (c *Client) parseBedrockEvent(event types.ConverseStreamOutput) *chat.ResponseUpdate {
    switch e := event.(type) {
    case *types.ConverseStreamOutputMemberContentBlockDelta:
        if textDelta, ok := e.Value.Delta.(*types.ContentBlockDeltaMemberText); ok {
            return &chat.ResponseUpdate{
                Kind:  chat.UpdateKindContentDelta,
                Delta: &chat.ContentDelta{TextDelta: textDelta.Value},
            }
        }
    case *types.ConverseStreamOutputMemberMessageStop:
        return &chat.ResponseUpdate{
            Kind:         chat.UpdateKindMessageComplete,
            FinishReason: mapBedrockStopReason(e.Value.StopReason),
        }
    case *types.ConverseStreamOutputMemberMetadata:
        if e.Value.Usage != nil {
            return &chat.ResponseUpdate{
                Kind: chat.UpdateKindUsage,
                Usage: &chat.UsageDetails{
                    InputTokens:  int(aws.ToInt32(e.Value.Usage.InputTokens)),
                    OutputTokens: int(aws.ToInt32(e.Value.Usage.OutputTokens)),
                },
            }
        }
    }
    return nil
}
```

Success criteria:

* ConverseStream events properly parsed
* All model types supported (Claude, Titan, Llama)
* Tool use events handled

Dependencies:

* Step 2.5.3 completion

---

### Step 2.5.5: Add Bedrock provider tests

Create tests for the Bedrock provider.

Files:

* `go/providers/bedrock/client_test.go`
* `go/providers/bedrock/convert_test.go`
* `go/providers/bedrock/integration_test.go`

---

## Feature 2.6: Ollama Provider Package

<!-- parallelizable: true -->

### Step 2.6.1: Create Ollama client structure and options

Create the Ollama provider package for local models.

Files:

* `go/providers/ollama/client.go` - Ollama client
* `go/providers/ollama/options.go` - Configuration options
* `go/providers/ollama/doc.go` - Package documentation

#### Client Structure

```go
// providers/ollama/client.go

// Client implements chat.Client for Ollama's Chat API.
type Client struct {
    baseURL    string
    model      string
    httpClient *http.Client
    metadata   chat.ClientMetadata
}

// NewClient creates a new Ollama chat client.
func NewClient(opts ...Option) (*Client, error) {
    cfg := defaultConfig()
    for _, opt := range opts {
        opt(cfg)
    }
    
    if cfg.baseURL == "" {
        cfg.baseURL = os.Getenv("OLLAMA_HOST")
        if cfg.baseURL == "" {
            cfg.baseURL = "http://localhost:11434"
        }
    }
    
    return &Client{
        baseURL:    cfg.baseURL,
        model:      cfg.model,
        httpClient: cfg.httpClient,
        metadata: chat.ClientMetadata{
            ProviderName: "ollama",
            ModelID:      cfg.model,
            EndpointURI:  cfg.baseURL,
        },
    }, nil
}
```

Success criteria:

* Client uses HTTP directly (no external dependency)
* Default localhost:11434 endpoint
* Model selection required

Context references:

* [python/packages/ollama/agent_framework_ollama/_chat_client.py](python/packages/ollama/agent_framework_ollama/_chat_client.py) - Ollama client

Dependencies:

* Feature 2.1 completion

---

### Step 2.6.2: Implement HTTP API integration

Implement chat completion via HTTP.

Files:

* `go/providers/ollama/client.go` - GetResponse
* `go/providers/ollama/api.go` - API types

#### API Types

```go
// providers/ollama/api.go

type chatRequest struct {
    Model    string          `json:"model"`
    Messages []ollamaMessage `json:"messages"`
    Stream   bool            `json:"stream"`
    Options  *modelOptions   `json:"options,omitempty"`
    Tools    []ollamaTool    `json:"tools,omitempty"`
}

type ollamaMessage struct {
    Role      string `json:"role"`
    Content   string `json:"content"`
    Images    []string `json:"images,omitempty"`
    ToolCalls []ollamaToolCall `json:"tool_calls,omitempty"`
}

type chatResponse struct {
    Model     string         `json:"model"`
    CreatedAt string         `json:"created_at"`
    Message   ollamaMessage  `json:"message"`
    Done      bool           `json:"done"`
    DoneReason string        `json:"done_reason,omitempty"`
    PromptEvalCount   int    `json:"prompt_eval_count,omitempty"`
    EvalCount         int    `json:"eval_count,omitempty"`
}

type modelOptions struct {
    NumPredict    int     `json:"num_predict,omitempty"`
    Temperature   float32 `json:"temperature,omitempty"`
    TopP          float32 `json:"top_p,omitempty"`
    TopK          int     `json:"top_k,omitempty"`
    RepeatPenalty float32 `json:"repeat_penalty,omitempty"`
}
```

#### GetResponse Implementation

```go
// providers/ollama/client.go

func (c *Client) GetResponse(ctx context.Context, messages []chat.Message, options *chat.Options) (*chat.Response, error) {
    req := chatRequest{
        Model:    c.model,
        Messages: toOllamaMessages(messages),
        Stream:   false,
    }
    
    if options != nil {
        req.Options = &modelOptions{}
        if options.MaxTokens > 0 {
            req.Options.NumPredict = options.MaxTokens
        }
        if options.Temperature != 0 {
            req.Options.Temperature = options.Temperature
        }
        if len(options.Tools) > 0 {
            req.Tools = toOllamaTools(options.Tools)
        }
    }
    
    body, err := json.Marshal(req)
    if err != nil {
        return nil, fmt.Errorf("ollama: marshal failed: %w", err)
    }
    
    httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/api/chat", bytes.NewReader(body))
    if err != nil {
        return nil, fmt.Errorf("ollama: request creation failed: %w", err)
    }
    httpReq.Header.Set("Content-Type", "application/json")
    
    httpResp, err := c.httpClient.Do(httpReq)
    if err != nil {
        return nil, fmt.Errorf("ollama: request failed: %w", err)
    }
    defer httpResp.Body.Close()
    
    if httpResp.StatusCode != http.StatusOK {
        return nil, fmt.Errorf("ollama: unexpected status %d", httpResp.StatusCode)
    }
    
    var resp chatResponse
    if err := json.NewDecoder(httpResp.Body).Decode(&resp); err != nil {
        return nil, fmt.Errorf("ollama: decode failed: %w", err)
    }
    
    return fromOllamaResponse(&resp)
}
```

Success criteria:

* HTTP POST to /api/chat endpoint
* Model options properly mapped
* Error responses handled

Dependencies:

* Step 2.6.1 completion

---

### Step 2.6.3: Implement NDJSON streaming

Implement streaming with newline-delimited JSON.

Files:

* `go/providers/ollama/client.go` - GetStreamingResponse
* `go/providers/ollama/stream.go` - NDJSON parsing

#### Streaming Implementation

```go
// providers/ollama/client.go

func (c *Client) GetStreamingResponse(ctx context.Context, messages []chat.Message, options *chat.Options) (<-chan chat.ResponseUpdate, error) {
    req := chatRequest{
        Model:    c.model,
        Messages: toOllamaMessages(messages),
        Stream:   true,
    }
    
    // ... options setup ...
    
    body, err := json.Marshal(req)
    if err != nil {
        return nil, fmt.Errorf("ollama: marshal failed: %w", err)
    }
    
    httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/api/chat", bytes.NewReader(body))
    if err != nil {
        return nil, fmt.Errorf("ollama: request creation failed: %w", err)
    }
    httpReq.Header.Set("Content-Type", "application/json")
    
    httpResp, err := c.httpClient.Do(httpReq)
    if err != nil {
        return nil, fmt.Errorf("ollama: request failed: %w", err)
    }
    
    if httpResp.StatusCode != http.StatusOK {
        httpResp.Body.Close()
        return nil, fmt.Errorf("ollama: unexpected status %d", httpResp.StatusCode)
    }
    
    updates := make(chan chat.ResponseUpdate, 32)
    
    go func() {
        defer close(updates)
        defer httpResp.Body.Close()
        
        c.processNDJSONStream(ctx, httpResp.Body, updates)
    }()
    
    return updates, nil
}

func (c *Client) processNDJSONStream(ctx context.Context, body io.Reader, updates chan<- chat.ResponseUpdate) {
    scanner := bufio.NewScanner(body)
    
    for scanner.Scan() {
        select {
        case <-ctx.Done():
            updates <- chat.ResponseUpdate{Kind: chat.UpdateKindError, Error: ctx.Err()}
            return
        default:
        }
        
        line := scanner.Bytes()
        if len(line) == 0 {
            continue
        }
        
        var chunk chatResponse
        if err := json.Unmarshal(line, &chunk); err != nil {
            updates <- chat.ResponseUpdate{Kind: chat.UpdateKindError, Error: err}
            return
        }
        
        if chunk.Message.Content != "" {
            updates <- chat.ResponseUpdate{
                Kind:  chat.UpdateKindContentDelta,
                Delta: &chat.ContentDelta{TextDelta: chunk.Message.Content},
            }
        }
        
        if chunk.Done {
            updates <- chat.ResponseUpdate{
                Kind:         chat.UpdateKindMessageComplete,
                FinishReason: chat.FinishReasonStop,
            }
            if chunk.EvalCount > 0 || chunk.PromptEvalCount > 0 {
                updates <- chat.ResponseUpdate{
                    Kind: chat.UpdateKindUsage,
                    Usage: &chat.UsageDetails{
                        InputTokens:  chunk.PromptEvalCount,
                        OutputTokens: chunk.EvalCount,
                    },
                }
            }
            updates <- chat.ResponseUpdate{Kind: chat.UpdateKindDone}
            return
        }
    }
    
    if err := scanner.Err(); err != nil {
        updates <- chat.ResponseUpdate{Kind: chat.UpdateKindError, Error: err}
    }
}
```

Success criteria:

* NDJSON lines properly parsed
* Buffered scanner for efficient reading
* Done flag triggers completion updates

Dependencies:

* Step 2.6.2 completion

---

### Step 2.6.4: Add Ollama provider tests

Create tests for the Ollama provider.

Files:

* `go/providers/ollama/client_test.go`
* `go/providers/ollama/integration_test.go`

---

## Feature 2.7: OpenTelemetry Observability Package

<!-- parallelizable: false -->

### Step 2.7.1: Define GenAI semantic conventions

Create semantic convention constants for AI observability.

Files:

* `go/observability/semconv.go` - Semantic conventions
* `go/observability/doc.go` - Updated package documentation

#### Semantic Conventions

```go
// observability/semconv.go

// GenAI semantic conventions for AI agent observability.
// Aligns with OpenTelemetry Semantic Conventions for GenAI.
const (
    // Operation names
    OperationInvokeAgent = "invoke_agent"
    OperationChatRequest = "chat"
    
    // Agent attributes
    AttributeAgentID          = "gen_ai.agent.id"
    AttributeAgentName        = "gen_ai.agent.name"
    AttributeAgentDescription = "gen_ai.agent.description"
    
    // Provider attributes
    AttributeProviderName = "gen_ai.provider.name"
    AttributeModelID      = "gen_ai.request.model"
    
    // Operation attributes
    AttributeOperationName = "gen_ai.operation.name"
    
    // Token usage attributes
    AttributeInputTokens    = "gen_ai.usage.input_tokens"
    AttributeOutputTokens   = "gen_ai.usage.output_tokens"
    AttributeCachedTokens   = "gen_ai.usage.cached_tokens"
    AttributeReasoningTokens = "gen_ai.usage.reasoning_tokens"
    
    // Response attributes
    AttributeFinishReason = "gen_ai.response.finish_reason"
    AttributeResponseID   = "gen_ai.response.id"
    
    // Error attributes
    AttributeErrorType    = "gen_ai.error.type"
    AttributeErrorMessage = "gen_ai.error.message"
)

// Tracer name for the agent framework.
const TracerName = "github.com/microsoft/agent-framework-go"

// Meter name for the agent framework.
const MeterName = "github.com/microsoft/agent-framework-go"
```

Success criteria:

* Constants align with OpenTelemetry GenAI conventions
* All relevant attributes defined
* Constants exported for external use

Context references:

* [dotnet/src/Microsoft.Agents.AI/OpenTelemetryConsts.cs](dotnet/src/Microsoft.Agents.AI/OpenTelemetryConsts.cs) - .NET conventions

Dependencies:

* None

---

### Step 2.7.2: Implement tracing instrumentation

Create tracing helpers and span management.

Files:

* `go/observability/tracing.go` - Tracing utilities
* `go/observability/setup.go` - Setup functions

#### Tracing Utilities

```go
// observability/tracing.go

import (
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/attribute"
    "go.opentelemetry.io/otel/trace"
)

// StartAgentSpan creates a span for agent invocation.
func StartAgentSpan(ctx context.Context, agentID, agentName, providerName string) (context.Context, trace.Span) {
    tracer := otel.Tracer(TracerName)
    
    ctx, span := tracer.Start(ctx, OperationInvokeAgent,
        trace.WithSpanKind(trace.SpanKindClient),
        trace.WithAttributes(
            attribute.String(AttributeOperationName, OperationInvokeAgent),
            attribute.String(AttributeAgentID, agentID),
            attribute.String(AttributeAgentName, agentName),
            attribute.String(AttributeProviderName, providerName),
        ),
    )
    
    return ctx, span
}

// StartChatSpan creates a span for chat client requests.
func StartChatSpan(ctx context.Context, providerName, modelID string) (context.Context, trace.Span) {
    tracer := otel.Tracer(TracerName)
    
    ctx, span := tracer.Start(ctx, OperationChatRequest,
        trace.WithSpanKind(trace.SpanKindClient),
        trace.WithAttributes(
            attribute.String(AttributeOperationName, OperationChatRequest),
            attribute.String(AttributeProviderName, providerName),
            attribute.String(AttributeModelID, modelID),
        ),
    )
    
    return ctx, span
}

// RecordUsage adds token usage attributes to the current span.
func RecordUsage(span trace.Span, usage *chat.UsageDetails) {
    if usage == nil {
        return
    }
    
    span.SetAttributes(
        attribute.Int(AttributeInputTokens, usage.InputTokens),
        attribute.Int(AttributeOutputTokens, usage.OutputTokens),
    )
    
    if usage.CachedTokens > 0 {
        span.SetAttributes(attribute.Int(AttributeCachedTokens, usage.CachedTokens))
    }
    if usage.ReasoningTokens > 0 {
        span.SetAttributes(attribute.Int(AttributeReasoningTokens, usage.ReasoningTokens))
    }
}

// RecordError adds error attributes to the current span.
func RecordError(span trace.Span, err error) {
    span.RecordError(err)
    span.SetAttributes(
        attribute.String(AttributeErrorType, fmt.Sprintf("%T", err)),
        attribute.String(AttributeErrorMessage, err.Error()),
    )
}
```

#### Setup Functions

```go
// observability/setup.go

import (
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
    "go.opentelemetry.io/otel/sdk/resource"
    "go.opentelemetry.io/otel/sdk/trace"
    semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

// SetupTracing configures OpenTelemetry tracing with OTLP export.
func SetupTracing(ctx context.Context, serviceName string, opts ...TracingOption) (shutdown func(context.Context) error, err error) {
    cfg := defaultTracingConfig()
    for _, opt := range opts {
        opt(cfg)
    }
    
    res, err := resource.New(ctx,
        resource.WithAttributes(
            semconv.ServiceNameKey.String(serviceName),
        ),
    )
    if err != nil {
        return nil, fmt.Errorf("failed to create resource: %w", err)
    }
    
    exporter, err := otlptracegrpc.New(ctx)
    if err != nil {
        return nil, fmt.Errorf("failed to create exporter: %w", err)
    }
    
    provider := trace.NewTracerProvider(
        trace.WithBatcher(exporter),
        trace.WithResource(res),
        trace.WithSampler(cfg.sampler),
    )
    
    otel.SetTracerProvider(provider)
    
    return provider.Shutdown, nil
}

type TracingOption func(*tracingConfig)

func WithSampler(sampler trace.Sampler) TracingOption
func WithExporter(exporter trace.SpanExporter) TracingOption
```

Success criteria:

* StartAgentSpan creates properly attributed spans
* RecordUsage adds token metrics
* SetupTracing configures OTLP export

Dependencies:

* Step 2.7.1 completion

---

### Step 2.7.3: Implement metrics collection

Create metrics for agent and chat operations.

Files:

* `go/observability/metrics.go` - Metrics definitions

#### Metrics Definitions

```go
// observability/metrics.go

import (
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/metric"
)

// Metrics holds the instruments for observability.
type Metrics struct {
    AgentRuns       metric.Int64Counter
    TokensInput     metric.Int64Histogram
    TokensOutput    metric.Int64Histogram
    RequestLatency  metric.Float64Histogram
    Errors          metric.Int64Counter
}

// NewMetrics creates the metrics instruments.
func NewMetrics() (*Metrics, error) {
    meter := otel.Meter(MeterName)
    
    agentRuns, err := meter.Int64Counter(
        "gen_ai.agent.runs",
        metric.WithDescription("Number of agent runs"),
        metric.WithUnit("{run}"),
    )
    if err != nil {
        return nil, err
    }
    
    tokensInput, err := meter.Int64Histogram(
        "gen_ai.usage.input_tokens",
        metric.WithDescription("Input tokens per request"),
        metric.WithUnit("{token}"),
    )
    if err != nil {
        return nil, err
    }
    
    tokensOutput, err := meter.Int64Histogram(
        "gen_ai.usage.output_tokens",
        metric.WithDescription("Output tokens per request"),
        metric.WithUnit("{token}"),
    )
    if err != nil {
        return nil, err
    }
    
    requestLatency, err := meter.Float64Histogram(
        "gen_ai.request.latency",
        metric.WithDescription("Request latency in seconds"),
        metric.WithUnit("s"),
    )
    if err != nil {
        return nil, err
    }
    
    errors, err := meter.Int64Counter(
        "gen_ai.errors",
        metric.WithDescription("Number of errors"),
        metric.WithUnit("{error}"),
    )
    if err != nil {
        return nil, err
    }
    
    return &Metrics{
        AgentRuns:      agentRuns,
        TokensInput:    tokensInput,
        TokensOutput:   tokensOutput,
        RequestLatency: requestLatency,
        Errors:         errors,
    }, nil
}
```

Success criteria:

* Counter for agent runs
* Histograms for token usage
* Histogram for latency
* Counter for errors

Dependencies:

* Step 2.7.1 completion

---

### Step 2.7.4: Create instrumented client wrapper

Create a delegating wrapper that adds observability.

Files:

* `go/observability/instrumented.go` - Instrumented client wrapper

#### Instrumented Client

```go
// observability/instrumented.go

// InstrumentedClient wraps a chat.Client with OpenTelemetry instrumentation.
type InstrumentedClient struct {
    inner    chat.Client
    metrics  *Metrics
    
    // Configuration
    EnableSensitiveData bool
}

// NewInstrumentedClient creates an instrumented wrapper.
func NewInstrumentedClient(client chat.Client) *InstrumentedClient {
    metrics, _ := NewMetrics()
    return &InstrumentedClient{
        inner:   client,
        metrics: metrics,
    }
}

func (c *InstrumentedClient) GetResponse(ctx context.Context, messages []chat.Message, options *chat.Options) (*chat.Response, error) {
    metadata := c.inner.Metadata()
    ctx, span := StartChatSpan(ctx, metadata.ProviderName, metadata.ModelID)
    defer span.End()
    
    start := time.Now()
    
    resp, err := c.inner.GetResponse(ctx, messages, options)
    
    latency := time.Since(start).Seconds()
    c.metrics.RequestLatency.Record(ctx, latency,
        metric.WithAttributes(
            attribute.String(AttributeProviderName, metadata.ProviderName),
            attribute.String(AttributeModelID, metadata.ModelID),
        ),
    )
    
    if err != nil {
        RecordError(span, err)
        c.metrics.Errors.Add(ctx, 1)
        return nil, err
    }
    
    if resp.Usage.InputTokens > 0 {
        c.metrics.TokensInput.Record(ctx, int64(resp.Usage.InputTokens))
    }
    if resp.Usage.OutputTokens > 0 {
        c.metrics.TokensOutput.Record(ctx, int64(resp.Usage.OutputTokens))
    }
    RecordUsage(span, &resp.Usage)
    
    span.SetAttributes(
        attribute.String(AttributeFinishReason, string(resp.FinishReason)),
    )
    
    return resp, nil
}

func (c *InstrumentedClient) GetStreamingResponse(ctx context.Context, messages []chat.Message, options *chat.Options) (<-chan chat.ResponseUpdate, error) {
    metadata := c.inner.Metadata()
    ctx, span := StartChatSpan(ctx, metadata.ProviderName, metadata.ModelID)
    
    start := time.Now()
    
    updates, err := c.inner.GetStreamingResponse(ctx, messages, options)
    if err != nil {
        RecordError(span, err)
        span.End()
        return nil, err
    }
    
    // Wrap channel to instrument completion
    instrumented := make(chan chat.ResponseUpdate, 32)
    go func() {
        defer close(instrumented)
        defer span.End()
        
        var totalInputTokens, totalOutputTokens int
        
        for update := range updates {
            if update.Kind == chat.UpdateKindUsage && update.Usage != nil {
                totalInputTokens = update.Usage.InputTokens
                totalOutputTokens = update.Usage.OutputTokens
            }
            if update.Kind == chat.UpdateKindDone {
                latency := time.Since(start).Seconds()
                c.metrics.RequestLatency.Record(ctx, latency)
                
                if totalInputTokens > 0 {
                    c.metrics.TokensInput.Record(ctx, int64(totalInputTokens))
                }
                if totalOutputTokens > 0 {
                    c.metrics.TokensOutput.Record(ctx, int64(totalOutputTokens))
                }
            }
            if update.Kind == chat.UpdateKindError && update.Error != nil {
                RecordError(span, update.Error)
                c.metrics.Errors.Add(ctx, 1)
            }
            instrumented <- update
        }
    }()
    
    return instrumented, nil
}

func (c *InstrumentedClient) Metadata() chat.ClientMetadata {
    return c.inner.Metadata()
}
```

Success criteria:

* InstrumentedClient implements chat.Client
* Spans created for requests with proper attributes
* Metrics recorded for latency, tokens, errors
* Streaming properly instrumented

Context references:

* [dotnet/src/Microsoft.Agents.AI/OpenTelemetryAgent.cs](dotnet/src/Microsoft.Agents.AI/OpenTelemetryAgent.cs) - .NET OTEL agent

Dependencies:

* Steps 2.7.1-2.7.3 completion

---

### Step 2.7.5: Add observability tests

Create tests for observability package.

Files:

* `go/observability/tracing_test.go`
* `go/observability/metrics_test.go`
* `go/observability/instrumented_test.go`

---

## Feature 2.8: ChatClientAgent Implementation

<!-- parallelizable: false -->

### Step 2.8.1: Create ChatClientAgent structure

Create the main agent implementation that wraps chat clients.

Files:

* `go/chatagent/agent.go` - ChatClientAgent struct
* `go/chatagent/doc.go` - Package documentation

#### ChatClientAgent Structure

```go
// chatagent/agent.go

import (
    "github.com/microsoft/agent-framework-go/agent"
    "github.com/microsoft/agent-framework-go/chat"
    "github.com/microsoft/agent-framework-go/tool"
)

// Agent implements agent.Agent by wrapping a chat.Client.
// This is the primary agent implementation in the framework.
type Agent struct {
    id          string
    name        string
    description string
    client      chat.Client
    
    // Configuration
    instructions string
    tools        []tool.Tool
    maxTurns     int
    
    // Invocation configuration
    invocationConfig tool.InvocationConfig
    
    // Services
    services map[reflect.Type]interface{}
}

var _ agent.Agent = (*Agent)(nil)

// New creates a new ChatClientAgent from a chat.Client.
func New(client chat.Client, opts ...Option) *Agent {
    cfg := defaultConfig()
    for _, opt := range opts {
        opt(cfg)
    }
    
    a := &Agent{
        id:               cfg.id,
        name:             cfg.name,
        description:      cfg.description,
        client:           client,
        instructions:     cfg.instructions,
        tools:            cfg.tools,
        maxTurns:         cfg.maxTurns,
        invocationConfig: cfg.invocationConfig,
        services:         make(map[reflect.Type]interface{}),
    }
    
    if a.id == "" {
        a.id = uuid.NewString()
    }
    if a.name == "" {
        a.name = client.Metadata().ModelID
    }
    
    return a
}

func (a *Agent) ID() string                      { return a.id }
func (a *Agent) Name() string                    { return a.name }
func (a *Agent) Description() string             { return a.description }
func (a *Agent) Metadata() agent.AIAgentMetadata { /* ... */ }
func (a *Agent) GetService(t reflect.Type) interface{} { return a.services[t] }
```

Success criteria:

* Agent implements agent.Agent interface
* Configuration via functional options
* Default values for ID and name

Context references:

* [dotnet/src/Microsoft.Agents.AI/ChatClient/ChatClientAgent.cs](dotnet/src/Microsoft.Agents.AI/ChatClient/ChatClientAgent.cs) (Lines 57-106) - .NET constructor

Dependencies:

* Features 2.1-2.7 completion

---

### Step 2.8.2: Implement agent options and builder

Create configuration options and fluent builder.

Files:

* `go/chatagent/options.go` - Agent options
* `go/chatagent/builder.go` - Fluent builder

#### Options

```go
// chatagent/options.go

type config struct {
    id               string
    name             string
    description      string
    instructions     string
    tools            []tool.Tool
    maxTurns         int
    invocationConfig tool.InvocationConfig
}

func defaultConfig() *config {
    return &config{
        maxTurns:         10,
        invocationConfig: tool.DefaultInvocationConfig(),
    }
}

type Option func(*config)

func WithID(id string) Option
func WithName(name string) Option
func WithDescription(description string) Option
func WithInstructions(instructions string) Option
func WithTools(tools ...tool.Tool) Option
func WithMaxTurns(maxTurns int) Option
func WithInvocationConfig(cfg tool.InvocationConfig) Option
```

#### Builder Pattern

```go
// chatagent/builder.go

// Builder provides fluent configuration for ChatClientAgent.
type Builder struct {
    client chat.Client
    opts   []Option
}

// NewBuilder creates a new agent builder.
func NewBuilder(client chat.Client) *Builder {
    return &Builder{client: client}
}

func (b *Builder) ID(id string) *Builder {
    b.opts = append(b.opts, WithID(id))
    return b
}

func (b *Builder) Name(name string) *Builder {
    b.opts = append(b.opts, WithName(name))
    return b
}

func (b *Builder) Description(description string) *Builder {
    b.opts = append(b.opts, WithDescription(description))
    return b
}

func (b *Builder) Instructions(instructions string) *Builder {
    b.opts = append(b.opts, WithInstructions(instructions))
    return b
}

func (b *Builder) Tools(tools ...tool.Tool) *Builder {
    b.opts = append(b.opts, WithTools(tools...))
    return b
}

func (b *Builder) MaxTurns(maxTurns int) *Builder {
    b.opts = append(b.opts, WithMaxTurns(maxTurns))
    return b
}

// Build creates the configured agent.
func (b *Builder) Build() (*Agent, error) {
    if b.client == nil {
        return nil, errors.New("chat client is required")
    }
    return New(b.client, b.opts...), nil
}
```

Success criteria:

* All configuration options available
* Builder provides fluent API
* Build validates required configuration

Dependencies:

* Step 2.8.1 completion

---

### Step 2.8.3: Implement Run and RunStream methods

Implement the core agent execution methods.

Files:

* `go/chatagent/agent.go` - Run and RunStream methods

#### Run Implementation

```go
// chatagent/agent.go

func (a *Agent) Run(ctx context.Context, messages []agent.Message, opts ...agent.RunOption) (*agent.Response, error) {
    cfg := agent.NewRunConfig(opts...)
    
    // Prepare messages with instructions
    chatMessages := a.prepareMessages(messages)
    
    // Prepare options with tools
    chatOptions := a.prepareChatOptions(cfg)
    
    // Execute with tool invocation loop
    return a.runWithToolLoop(ctx, chatMessages, chatOptions, cfg)
}

func (a *Agent) prepareMessages(messages []agent.Message) []chat.Message {
    chatMessages := make([]chat.Message, 0, len(messages)+1)
    
    // Add system instructions if configured
    if a.instructions != "" {
        chatMessages = append(chatMessages, chat.NewSystemMessage(a.instructions))
    }
    
    // Convert agent messages to chat messages
    for _, msg := range messages {
        chatMessages = append(chatMessages, toChatMessage(msg))
    }
    
    return chatMessages
}

func (a *Agent) prepareChatOptions(cfg *agent.RunConfig) *chat.Options {
    opts := &chat.Options{}
    
    // Merge agent tools with run-specific tools
    allTools := make([]tool.Tool, 0, len(a.tools)+len(cfg.Tools))
    allTools = append(allTools, a.tools...)
    for _, t := range cfg.Tools {
        if ft, ok := t.(tool.Tool); ok {
            allTools = append(allTools, ft)
        }
    }
    opts.Tools = allTools
    
    if cfg.MaxTokens > 0 {
        opts.MaxTokens = cfg.MaxTokens
    }
    if cfg.Temperature != 0 {
        opts.Temperature = cfg.Temperature
    }
    
    return opts
}

func (a *Agent) RunStream(ctx context.Context, messages []agent.Message, opts ...agent.RunOption) (<-chan agent.ResponseUpdate, error) {
    cfg := agent.NewRunConfig(opts...)
    
    chatMessages := a.prepareMessages(messages)
    chatOptions := a.prepareChatOptions(cfg)
    
    return a.runStreamWithToolLoop(ctx, chatMessages, chatOptions, cfg)
}
```

Success criteria:

* Instructions prepended to messages
* Tools merged from agent and run options
* Options properly translated

Context references:

* [dotnet/src/Microsoft.Agents.AI/ChatClient/ChatClientAgent.cs](dotnet/src/Microsoft.Agents.AI/ChatClient/ChatClientAgent.cs) (Lines 157-190) - .NET Run implementation

Dependencies:

* Steps 2.8.1-2.8.2 completion

---

### Step 2.8.4: Implement automatic tool invocation loop

Create the tool invocation loop that handles tool calls automatically.

Files:

* `go/chatagent/toolloop.go` - Tool invocation loop

#### Tool Loop Implementation

```go
// chatagent/toolloop.go

func (a *Agent) runWithToolLoop(ctx context.Context, messages []chat.Message, options *chat.Options, cfg *agent.RunConfig) (*agent.Response, error) {
    invoker := tool.NewInvoker(options.Tools, a.invocationConfig)
    
    currentMessages := messages
    var allResponseMessages []chat.Message
    var finalUsage chat.UsageDetails
    var consecutiveErrors int
    
    for turn := 0; turn < a.maxTurns; turn++ {
        // Get response from chat client
        resp, err := a.client.GetResponse(ctx, currentMessages, options)
        if err != nil {
            return nil, fmt.Errorf("chat request failed: %w", err)
        }
        
        // Accumulate usage
        finalUsage.InputTokens += resp.Usage.InputTokens
        finalUsage.OutputTokens += resp.Usage.OutputTokens
        finalUsage.TotalTokens += resp.Usage.TotalTokens
        
        allResponseMessages = append(allResponseMessages, resp.Message)
        
        // Check for tool calls
        if len(resp.Message.ToolCalls) == 0 || resp.FinishReason != chat.FinishReasonToolCalls {
            // No tool calls, we're done
            return a.buildResponse(allResponseMessages, finalUsage, resp.FinishReason), nil
        }
        
        // Invoke tools
        toolResults, err := a.invokeTools(ctx, invoker, resp.Message.ToolCalls)
        if err != nil {
            consecutiveErrors++
            if consecutiveErrors >= a.invocationConfig.MaxConsecutiveErrors {
                return nil, fmt.Errorf("max consecutive errors exceeded: %w", err)
            }
        } else {
            consecutiveErrors = 0
        }
        
        // Append assistant message and tool results
        currentMessages = append(currentMessages, resp.Message)
        for _, result := range toolResults {
            currentMessages = append(currentMessages, chat.NewToolMessage(result.CallID, result.Content))
        }
    }
    
    return nil, tool.ErrMaxIterations
}

func (a *Agent) invokeTools(ctx context.Context, invoker *tool.Invoker, toolCalls []chat.ToolCall) ([]toolResult, error) {
    results := make([]toolResult, 0, len(toolCalls))
    
    for _, tc := range toolCalls {
        result, err := invoker.Invoke(ctx, tc.Name, tc.Arguments)
        if err != nil && a.invocationConfig.TerminateOnUnknownCalls {
            return nil, err
        }
        
        content := result.Content
        if result.IsError && a.invocationConfig.IncludeDetailedErrors {
            content = fmt.Sprintf("Error: %s", content)
        }
        
        results = append(results, toolResult{
            CallID:  tc.ID,
            Content: content,
            IsError: result.IsError,
        })
    }
    
    return results, nil
}

type toolResult struct {
    CallID  string
    Content string
    IsError bool
}
```

Success criteria:

* Loop terminates on non-tool-call responses
* Tool results appended correctly
* MaxTurns enforced
* Consecutive error handling

Context references:

* [dotnet/src/Microsoft.Agents.AI/ChatClient/ChatClientExtensions.cs](dotnet/src/Microsoft.Agents.AI/ChatClient/ChatClientExtensions.cs) - FunctionInvokingChatClient pattern

Dependencies:

* Step 2.8.3 completion

---

### Step 2.8.5: Implement session management

Implement session creation and restoration.

Files:

* `go/chatagent/session.go` - ChatClientAgentSession

#### Session Implementation

```go
// chatagent/session.go

// Session implements agent.Session for ChatClientAgent.
type Session struct {
    id              string
    conversationID  string
    messages        []chat.Message
    mu              sync.RWMutex
    
    services map[reflect.Type]interface{}
}

var _ agent.Session = (*Session)(nil)

func newSession() *Session {
    return &Session{
        id:       uuid.NewString(),
        messages: make([]chat.Message, 0),
        services: make(map[reflect.Type]interface{}),
    }
}

func (s *Session) ID() string {
    return s.id
}

func (s *Session) Messages() []agent.Message {
    s.mu.RLock()
    defer s.mu.RUnlock()
    
    result := make([]agent.Message, len(s.messages))
    for i, msg := range s.messages {
        result[i] = toAgentMessage(msg)
    }
    return result
}

func (s *Session) AddMessage(msg agent.Message) {
    s.mu.Lock()
    defer s.mu.Unlock()
    
    s.messages = append(s.messages, toChatMessage(msg))
}

func (s *Session) Serialize() (json.RawMessage, error) {
    s.mu.RLock()
    defer s.mu.RUnlock()
    
    data := sessionData{
        ID:             s.id,
        ConversationID: s.conversationID,
        Messages:       s.messages,
    }
    
    return json.Marshal(data)
}

func (s *Session) GetService(t reflect.Type) interface{} {
    return s.services[t]
}

type sessionData struct {
    ID             string         `json:"id"`
    ConversationID string         `json:"conversation_id,omitempty"`
    Messages       []chat.Message `json:"messages"`
}

// Agent session methods

func (a *Agent) NewSession(ctx context.Context) (agent.Session, error) {
    return newSession(), nil
}

func (a *Agent) RestoreSession(ctx context.Context, data json.RawMessage) (agent.Session, error) {
    var sd sessionData
    if err := json.Unmarshal(data, &sd); err != nil {
        return nil, fmt.Errorf("failed to deserialize session: %w", err)
    }
    
    return &Session{
        id:             sd.ID,
        conversationID: sd.ConversationID,
        messages:       sd.Messages,
        services:       make(map[reflect.Type]interface{}),
    }, nil
}
```

Success criteria:

* Session is thread-safe
* Serialization/deserialization works
* Messages properly converted

Context references:

* [dotnet/src/Microsoft.Agents.AI/ChatClient/ChatClientAgentSession.cs](dotnet/src/Microsoft.Agents.AI/ChatClient/ChatClientAgentSession.cs) - .NET session

Dependencies:

* Steps 2.8.1-2.8.4 completion

---

### Step 2.8.6: Add ChatClientAgent tests

Create comprehensive tests for ChatClientAgent.

Files:

* `go/chatagent/agent_test.go` - Unit tests
* `go/chatagent/session_test.go` - Session tests
* `go/chatagent/toolloop_test.go` - Tool loop tests
* `go/chatagent/integration_test.go` - Integration tests

Test coverage requirements:

* Agent creation with various options
* Run with and without tools
* Streaming execution
* Tool invocation loop
* Session management
* Error scenarios

Success criteria:

* 90%+ code coverage
* Mock chat client for unit tests
* Integration tests with real providers

---

## Dependencies

* Go 1.22+ with generic type constraints
* `github.com/sashabaranov/go-openai` v1.x - OpenAI Go client
* `github.com/liushuangls/go-anthropic/v2` - Anthropic Go client
* `github.com/Azure/azure-sdk-for-go/sdk/ai/azopenai` - Azure OpenAI SDK
* `github.com/Azure/azure-sdk-for-go/sdk/azidentity` - Azure authentication
* `github.com/aws/aws-sdk-go-v2` - AWS SDK for Bedrock
* `go.opentelemetry.io/otel` v1.x - OpenTelemetry SDK

## Success Criteria

* All providers implement chat.Client interface with feature parity
* Tool system supports function tools and hosted tools
* OpenTelemetry instrumentation follows GenAI semantic conventions
* ChatClientAgent provides automatic tool invocation loop
* 90%+ test coverage across all packages
* API patterns align with .NET and Python implementations
