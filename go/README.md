# Get Started with Microsoft Agent Framework for Go Developers

[![Go Reference](https://pkg.go.dev/badge/github.com/microsoft/agent-framework-go.svg)](https://pkg.go.dev/github.com/microsoft/agent-framework-go)
[![Go Report Card](https://goreportcard.com/badge/github.com/microsoft/agent-framework-go)](https://goreportcard.com/report/github.com/microsoft/agent-framework-go)

## Quick Install

Install the Agent Framework Go module using `go get`:

```bash
go get github.com/microsoft/agent-framework-go@latest
```

### Requirements

- Go 1.22 or later
- Supported OS: Windows, macOS, Linux

## Setup API Keys

Set environment variables for your preferred LLM provider:

```bash
# OpenAI
export OPENAI_API_KEY=sk-...
export OPENAI_MODEL=gpt-4o

# Azure OpenAI
export AZURE_OPENAI_ENDPOINT=https://your-resource.openai.azure.com/
export AZURE_OPENAI_DEPLOYMENT_NAME=your-deployment
export AZURE_OPENAI_API_KEY=...  # or use Azure AD authentication

# Anthropic
export ANTHROPIC_API_KEY=...
export ANTHROPIC_MODEL=claude-3-sonnet-20240229
```

## Create a Simple Agent

Create an agent and invoke it directly:

```go
package main

import (
    "context"
    "fmt"
    "log"
    "os"

    "github.com/microsoft/agent-framework-go/chatagent"
    "github.com/microsoft/agent-framework-go/providers/openai"
)

func main() {
    ctx := context.Background()

    // Create an OpenAI chat client
    client, err := openai.NewClient(
        openai.WithAPIKey(os.Getenv("OPENAI_API_KEY")),
        openai.WithModel("gpt-4o"),
    )
    if err != nil {
        log.Fatal(err)
    }

    // Create an agent with instructions
    agent := chatagent.New(client,
        chatagent.WithName("HaikuBot"),
        chatagent.WithInstructions("You are an upbeat assistant that writes beautifully."),
    )

    // Run the agent
    response, err := agent.Run(ctx, "Write a haiku about Microsoft Agent Framework.")
    if err != nil {
        log.Fatal(err)
    }

    fmt.Println(response.Text())
}
```

## Streaming Responses

Stream agent responses for real-time output:

```go
package main

import (
    "context"
    "fmt"
    "log"
    "os"

    "github.com/microsoft/agent-framework-go/chatagent"
    "github.com/microsoft/agent-framework-go/providers/azure"
)

func main() {
    ctx := context.Background()

    // Create an Azure OpenAI chat client
    client, err := azure.NewClient(
        azure.WithEndpoint(os.Getenv("AZURE_OPENAI_ENDPOINT")),
        azure.WithDeployment(os.Getenv("AZURE_OPENAI_DEPLOYMENT_NAME")),
        azure.WithAPIKey(os.Getenv("AZURE_OPENAI_API_KEY")),
    )
    if err != nil {
        log.Fatal(err)
    }

    // Create an agent
    agent := chatagent.New(client,
        chatagent.WithName("StreamingBot"),
        chatagent.WithInstructions("You are a helpful assistant."),
    )

    // Stream the response
    updates, err := agent.RunStream(ctx, "Explain Go concurrency in 3 sentences.")
    if err != nil {
        log.Fatal(err)
    }

    for update := range updates {
        if update.Delta != nil && update.Delta.TextDelta != "" {
            fmt.Print(update.Delta.TextDelta)
        }
    }
    fmt.Println()
}
```

## Build an Agent with Tools

Enhance your agent with custom function tools:

```go
package main

import (
    "context"
    "encoding/json"
    "fmt"
    "log"
    "os"

    "github.com/microsoft/agent-framework-go/chatagent"
    "github.com/microsoft/agent-framework-go/providers/openai"
    "github.com/microsoft/agent-framework-go/tool"
)

// WeatherArgs defines the parameters for the weather tool
type WeatherArgs struct {
    Location string `json:"location" description:"The location to get the weather for"`
}

func main() {
    ctx := context.Background()

    // Create a function tool
    weatherTool, err := tool.NewFunctionTool(
        "get_weather",
        "Get the current weather for a location",
        func(ctx context.Context, args WeatherArgs) (string, error) {
            // Simulated weather response
            return fmt.Sprintf("The weather in %s is sunny and 72°F.", args.Location), nil
        },
    )
    if err != nil {
        log.Fatal(err)
    }

    // Create an OpenAI chat client
    client, err := openai.NewClient(
        openai.WithAPIKey(os.Getenv("OPENAI_API_KEY")),
        openai.WithModel("gpt-4o"),
    )
    if err != nil {
        log.Fatal(err)
    }

    // Create an agent with tools
    agent := chatagent.New(client,
        chatagent.WithName("WeatherBot"),
        chatagent.WithInstructions("You are a helpful weather assistant."),
        chatagent.WithTools(weatherTool),
    )

    // Run the agent
    response, err := agent.Run(ctx, "What's the weather like in Seattle?")
    if err != nil {
        log.Fatal(err)
    }

    fmt.Println(response.Text())
}
```

## Multi-Agent Orchestration

The framework supports hierarchical agent patterns where one agent can use other agents as tools. This enables complex task decomposition and delegation.

```go
package main

import (
    "context"
    "fmt"
    "log"
    "os"

    "github.com/microsoft/agent-framework-go/agent"
    "github.com/microsoft/agent-framework-go/chatagent"
    "github.com/microsoft/agent-framework-go/providers/openai"
    "github.com/microsoft/agent-framework-go/tool"
)

func main() {
    ctx := context.Background()

    client, err := openai.NewClient(
        openai.WithAPIKey(os.Getenv("OPENAI_API_KEY")),
        openai.WithModel("gpt-4o"),
    )
    if err != nil {
        log.Fatal(err)
    }

    // Create specialized agents
    researcher := chatagent.New(client,
        chatagent.WithName("Researcher"),
        chatagent.WithInstructions("You research topics thoroughly and provide detailed information."),
    )

    writer := chatagent.New(client,
        chatagent.WithName("Writer"),
        chatagent.WithInstructions("You write clear, engaging content based on provided information."),
    )

    // Convert agents to tools with context forwarding
    tools := []tool.Tool{
        chatagent.AsTool(researcher, chatagent.AsToolOptions{
            Name:                  "research",
            Description:           "Research a topic in depth",
            ForwardRuntimeContext: true,
        }),
        chatagent.AsTool(writer, chatagent.AsToolOptions{
            Name:                  "write",
            Description:           "Write content based on research",
            ForwardRuntimeContext: true,
        }),
    }

    // Create orchestrator that delegates to specialized agents
    orchestrator := chatagent.New(client,
        chatagent.WithName("Orchestrator"),
        chatagent.WithInstructions(`You coordinate complex tasks by delegating to:
- research: for gathering information on topics
- write: for creating polished content`),
        chatagent.WithTools(tools...),
    )

    // Run with runtime context that propagates to all sub-agents
    rtc := agent.NewRuntimeContext().
        With("user_id", "u123").
        With("request_id", "req_abc")

    ctx = agent.WithRuntimeCtx(ctx, rtc)

    response, err := orchestrator.Run(ctx, "Research quantum computing and write a summary.")
    if err != nil {
        log.Fatal(err)
    }

    fmt.Println(response.Text())
}
```

See the [chatagent package documentation](./chatagent/doc.go) for complete AsTool options including streaming callbacks and custom key exclusion.

## Using Chat Clients Directly

Use chat clients directly without the agent abstraction:

```go
package main

import (
    "context"
    "fmt"
    "log"
    "os"

    "github.com/microsoft/agent-framework-go/chat"
    "github.com/microsoft/agent-framework-go/providers/openai"
)

func main() {
    ctx := context.Background()

    // Create an OpenAI chat client
    client, err := openai.NewClient(
        openai.WithAPIKey(os.Getenv("OPENAI_API_KEY")),
        openai.WithModel("gpt-4o"),
    )
    if err != nil {
        log.Fatal(err)
    }

    // Build messages
    messages := []chat.Message{
        chat.NewSystemMessage("You are a helpful assistant."),
        chat.NewUserMessage("Write a haiku about Go programming."),
    }

    // Get response
    response, err := client.GetResponse(ctx, messages, nil)
    if err != nil {
        log.Fatal(err)
    }

    fmt.Println(response.Message.Text())
}
```

## OpenTelemetry Observability

Enable distributed tracing and metrics for your agents:

```go
package main

import (
    "context"
    "log"
    "os"

    "github.com/microsoft/agent-framework-go/chatagent"
    "github.com/microsoft/agent-framework-go/observability"
    "github.com/microsoft/agent-framework-go/providers/openai"
)

func main() {
    ctx := context.Background()

    // Set up OpenTelemetry tracing
    shutdown, err := observability.SetupTracing("my-agent-service",
        observability.WithOTLPExporter("localhost:4317"),
    )
    if err != nil {
        log.Fatal(err)
    }
    defer shutdown()

    // Create a client and agent as usual
    client, _ := openai.NewClient(
        openai.WithAPIKey(os.Getenv("OPENAI_API_KEY")),
        openai.WithModel("gpt-4o"),
    )

    agent := chatagent.New(client,
        chatagent.WithName("TracedAgent"),
        chatagent.WithInstructions("You are a helpful assistant."),
    )

    // Agent runs are automatically traced
    response, _ := agent.Run(ctx, "Hello, world!")
    log.Println(response.Text())
}
```

## Samples

Explore the samples directory for complete examples:

- [Getting Started](./samples/getting_started/): Basic agent creation and usage
- [Agent Providers](./samples/providers/): Examples using different LLM providers (OpenAI, Azure, Anthropic, etc.)
- [Tool Usage](./samples/tools/): Function calling and tool integration
- [Workflows](./samples/workflows/): Multi-agent orchestration patterns
- [Observability](./samples/observability/): OpenTelemetry tracing and metrics

## Middleware

Add cross-cutting behavior to agents using middleware. The framework provides three types of middleware:

### Agent Middleware

Intercept agent `Run` and `RunStream` calls:

```go
package main

import (
    "context"
    "log"
    "time"

    "github.com/microsoft/agent-framework-go/agent"
    "github.com/microsoft/agent-framework-go/chatagent"
)

// LoggingMiddleware logs agent invocations with timing
func LoggingMiddleware() agent.AgentMiddleware {
    return agent.AgentMiddlewareFunc(func(
        ctx context.Context,
        agentCtx *agent.AgentContext,
        next agent.AgentHandler,
    ) error {
        start := time.Now()
        log.Printf("Agent %s: starting", agentCtx.Agent.Name())

        err := next(ctx, agentCtx)

        log.Printf("Agent %s: completed in %v", agentCtx.Agent.Name(), time.Since(start))
        return err
    })
}

func main() {
    // Use middleware with the builder
    agent := chatagent.NewBuilder(client).
        Name("MyAgent").
        UseMiddleware(LoggingMiddleware()).
        BuildAgent()
}
```

### Function Middleware

Intercept tool invocations:

```go
// ValidationMiddleware validates tool arguments before execution
func ValidationMiddleware() agent.FunctionMiddleware {
    return agent.FunctionMiddlewareFunc(func(
        ctx context.Context,
        funcCtx *agent.FunctionContext,
        next agent.FunctionHandler,
    ) error {
        log.Printf("Calling tool: %s", funcCtx.FunctionName)

        // Validate arguments before calling the tool
        if len(funcCtx.Arguments) == 0 {
            funcCtx.Error = errors.New("empty arguments not allowed")
            return nil
        }

        return next(ctx, funcCtx)
    })
}

func main() {
    agent := chatagent.NewBuilder(client).
        Name("ValidatedAgent").
        UseFunctionMiddleware(ValidationMiddleware()).
        Build()
}
```

### Chat Middleware

Intercept individual chat client requests (GetResponse/GetStreamingResponse). This operates at a lower level than AgentMiddleware, intercepting each chat request within the tool loop:

```go
// CachingMiddleware caches responses to avoid redundant API calls
type CachingMiddleware struct {
    cache map[string]*agent.ChatResponse
    mu    sync.RWMutex
}

func (m *CachingMiddleware) Process(ctx context.Context, chatCtx *agent.ChatContext, next agent.ChatHandler) error {
    // Skip caching for streaming requests
    if chatCtx.IsStreaming {
        return next(ctx, chatCtx)
    }

    key := computeCacheKey(chatCtx.Messages)

    m.mu.RLock()
    if cached, ok := m.cache[key]; ok {
        m.mu.RUnlock()
        chatCtx.Response = cached
        return nil // Short-circuit, don't call next
    }
    m.mu.RUnlock()

    if err := next(ctx, chatCtx); err != nil {
        return err
    }

    m.mu.Lock()
    m.cache[key] = chatCtx.Response
    m.mu.Unlock()
    return nil
}

func main() {
    caching := &CachingMiddleware{cache: make(map[string]*agent.ChatResponse)}
    agent := chatagent.NewBuilder(client).
        Name("CachedAgent").
        UseChatMiddleware(caching).
        Build()
}
```

Use ChatMiddleware for:

- Response caching and memoization
- Rate limiting and throttling
- Request/response logging and metrics
- Message transformation before sending

### Chaining Middleware

Chain multiple middleware using the builder or agent builder:

```go
// Using AgentBuilder for decorator pattern
result := agent.NewAgentBuilder(func() agent.Agent {
    return chatagent.New(client, chatagent.WithName("BaseAgent"))
}).
    UseMiddleware(LoggingMiddleware()).
    UseMiddleware(TracingMiddleware()).
    Use(func(inner agent.Agent) agent.Agent {
        return NewCustomDecorator(inner)
    }).
    Build()
```

### Observability Middleware

The framework provides telemetry middleware for OpenTelemetry integration:

```go
import "github.com/microsoft/agent-framework-go/observability"

// Create telemetry middleware with default settings
telemetry := observability.NewTelemetryMiddleware()

// Or with custom options
telemetry := observability.NewTelemetryMiddleware(
    observability.WithTelemetrySensitiveData(false), // Don't log message content
    observability.WithSourceName("my-app"),
)

// Add to agent builder
agent := chatagent.NewBuilder(client).
    UseMiddleware(telemetry).
    BuildAgent()
```

#### Function Telemetry

For tool/function call instrumentation:

```go
functionTelemetry := observability.NewFunctionTelemetryMiddleware()

agent := chatagent.NewBuilder(client).
    UseFunctionMiddleware(functionTelemetry).
    BuildAgent()
```

#### Telemetry Data Captured

| Category       | Data                                                                  |
| -------------- | --------------------------------------------------------------------- |
| **Spans**      | `agent.run`, `agent.run_stream`, `tool.call`                          |
| **Attributes** | agent.id, agent.name, provider, model, tokens, finish_reason          |
| **Metrics**    | agent.runs count, input/output tokens, latency histogram, error count |

### Middleware Execution Order

When an agent runs, middleware executes in a specific order. Understanding this order helps you design middleware that cooperates correctly.

#### Execution Flow

```text
┌─────────────────────────────────────────────────────────────────────┐
│ Agent.Run / Agent.RunStream                                         │
├─────────────────────────────────────────────────────────────────────┤
│  1. AgentMiddleware (outermost)                                     │
│     ├── Pre-processing (before next())                              │
│     │                                                               │
│     │  2. ContextProvider.Invoking                                  │
│     │     └── Injects instructions, messages, and tools             │
│     │                                                               │
│     │  3. Tool Loop (repeats until no tool calls)                   │
│     │     ├── ChatMiddleware                                        │
│     │     │   ├── Pre-processing                                    │
│     │     │   ├── ChatClient.GetResponse / GetStreamingResponse     │
│     │     │   └── Post-processing                                   │
│     │     │                                                         │
│     │     └── FunctionMiddleware (for each tool call)               │
│     │         ├── Pre-processing                                    │
│     │         ├── Tool.Invoke                                       │
│     │         └── Post-processing                                   │
│     │                                                               │
│     │  4. ContextProvider.Invoked (lifecycle hook)                  │
│     │     └── Receives request messages, response, and any error    │
│     │                                                               │
│     └── Post-processing (after next() returns)                      │
└─────────────────────────────────────────────────────────────────────┘
```

#### Middleware Levels

| Level                | Scope                          | Use Cases                                         |
| -------------------- | ------------------------------ | ------------------------------------------------- |
| **AgentMiddleware**  | Entire Run/RunStream call      | Logging, tracing, authentication, rate limiting  |
| **ContextProvider**  | Before/after agent invocation  | RAG, personalization, dynamic tool injection      |
| **ChatMiddleware**   | Each chat client request       | Caching, request transformation, retry logic     |
| **FunctionMiddleware** | Each tool invocation         | Validation, auditing, permission checks           |

#### Chaining Order

When chaining multiple middleware of the same type, they execute in registration order:

```go
agent := chatagent.NewBuilder(client).
    UseMiddleware(First()).   // Runs first (outermost)
    UseMiddleware(Second()).  // Runs second
    UseMiddleware(Third()).   // Runs third (innermost)
    BuildAgent()
```

For the call `agent.Run(ctx, messages)`:

1. `First` pre-processing → `Second` pre-processing → `Third` pre-processing
2. Actual agent execution
3. `Third` post-processing → `Second` post-processing → `First` post-processing

## Context Providers

Context providers enable dynamic context injection before each agent invocation. Use them to add user-specific personalization, retrieved documents (RAG), dynamic tool availability, or other context that varies per invocation.

### Basic Usage

```go
package main

import (
    "context"

    "github.com/microsoft/agent-framework-go/agent"
    "github.com/microsoft/agent-framework-go/chatagent"
)

func main() {
    // Create a simple context provider using a function
    userContextProvider := agent.ContextProviderFunc(
        func(ctx context.Context, messages []agent.Message) (*agent.Context, error) {
            // Retrieve user-specific context (e.g., from database)
            userPrefs := getUserPreferences(ctx)

            return &agent.Context{
                Instructions: "User prefers responses in " + userPrefs.Language,
                Messages:     nil, // Additional context messages if needed
                Tools:        nil, // Additional tools if needed
            }, nil
        },
    )

    // Create an agent with the context provider
    myAgent := chatagent.New(client,
        chatagent.WithInstructions("You are a helpful assistant."),
        chatagent.WithContextProvider(userContextProvider),
    )

    response, _ := myAgent.Run(ctx, "Hello!")
}
```

### RAG Context Provider Example

```go
// RAGProvider retrieves relevant documents for each query
type RAGProvider struct {
    vectorStore VectorStore
}

func (r *RAGProvider) Invoking(ctx context.Context, messages []agent.Message) (*agent.Context, error) {
    // Get the last user message as the query
    var query string
    for i := len(messages) - 1; i >= 0; i-- {
        if messages[i].Role == chat.RoleUser {
            query = messages[i].Text()
            break
        }
    }

    // Retrieve relevant documents
    docs, err := r.vectorStore.Search(ctx, query, 5)
    if err != nil {
        return nil, err
    }

    // Convert documents to context messages
    var contextMessages []chat.Message
    for _, doc := range docs {
        contextMessages = append(contextMessages, chat.NewUserMessage(
            fmt.Sprintf("Reference document: %s", doc.Content),
        ))
    }

    return &agent.Context{
        Instructions: "Use the reference documents to answer the user's question.",
        Messages:     contextMessages,
    }, nil
}

func main() {
    ragProvider := &RAGProvider{vectorStore: myVectorStore}

    agent := chatagent.New(client,
        chatagent.WithContextProvider(ragProvider),
    )
}
```

### Multiple Context Providers

Combine multiple providers for different concerns:

```go
agent := chatagent.New(client,
    chatagent.WithContextProvider(
        userProfileProvider,    // User personalization
        ragProvider,            // Document retrieval
        featureFlagProvider,    // Dynamic feature toggles
    ),
)
```

When using multiple providers, their contexts are merged:

- Instructions are concatenated with newlines
- Messages are appended in provider order
- Tools are merged in provider order

### Lifecycle Hooks

For providers that need to track conversation state, implement `ContextProviderWithLifecycle`:

```go
type ConversationTracker struct {
    agent.BaseContextProvider // Embed for default implementations
    history map[string][]agent.Message
}

func (c *ConversationTracker) Invoking(ctx context.Context, messages []agent.Message) (*agent.Context, error) {
    // Return context based on conversation history
    return &agent.Context{}, nil
}

func (c *ConversationTracker) Invoked(ctx context.Context, request, response []agent.Message, err error) error {
    // Track the conversation after each invocation
    sessionID := getSessionID(ctx)
    c.history[sessionID] = append(c.history[sessionID], request...)
    c.history[sessionID] = append(c.history[sessionID], response...)
    return nil
}

func (c *ConversationTracker) SessionCreated(ctx context.Context, sessionID string) error {
    // Initialize tracking for new sessions
    c.history[sessionID] = []agent.Message{}
    return nil
}
```

## Hierarchical Agents with AsTool

Convert agents to tools for hierarchical delegation:

```go
package main

import (
    "github.com/microsoft/agent-framework-go/chatagent"
)

func main() {
    // Create specialized agents
    researcher := chatagent.New(client,
        chatagent.WithName("Researcher"),
        chatagent.WithDescription("Researches topics in depth"),
        chatagent.WithInstructions("You perform detailed research on topics."),
    )

    writer := chatagent.New(client,
        chatagent.WithName("Writer"),
        chatagent.WithDescription("Writes content based on research"),
        chatagent.WithInstructions("You write polished content."),
    )

    // Convert agents to tools
    researchTool := chatagent.AsTool(researcher, chatagent.AsToolOptions{})
    writeTool := chatagent.AsTool(writer, chatagent.AsToolOptions{})

    // Create orchestrator that uses specialized agents as tools
    orchestrator := chatagent.New(client,
        chatagent.WithName("Orchestrator"),
        chatagent.WithTools(researchTool, writeTool),
        chatagent.WithInstructions("Coordinate research and writing tasks."),
    )

    response, _ := orchestrator.Run(ctx, "Research and write about AI agents")
}
```

## Workflow Orchestration

Build complex DAG-based workflows with multiple agents using the workflow package. Workflows use a Pregel-like execution model where executors process messages in synchronized supersteps.

```go
package main

import (
    "context"
    "fmt"
    "log"
    "os"

    "github.com/microsoft/agent-framework-go/agent"
    "github.com/microsoft/agent-framework-go/chatagent"
    "github.com/microsoft/agent-framework-go/providers/openai"
    "github.com/microsoft/agent-framework-go/workflow"
    "github.com/microsoft/agent-framework-go/workflow/executors"
)

func main() {
    ctx := context.Background()

    client, err := openai.NewClient(
        openai.WithAPIKey(os.Getenv("OPENAI_API_KEY")),
        openai.WithModel("gpt-4o"),
    )
    if err != nil {
        log.Fatal(err)
    }

    // Create specialized agents
    researcher := chatagent.New(client,
        chatagent.WithName("Researcher"),
        chatagent.WithInstructions("Research the topic and provide key facts."),
    )

    analyzer := chatagent.New(client,
        chatagent.WithName("Analyzer"),
        chatagent.WithInstructions("Analyze the research and identify patterns."),
    )

    writer := chatagent.New(client,
        chatagent.WithName("Writer"),
        chatagent.WithInstructions("Write a summary based on the analysis."),
    )

    // Wrap agents as workflow executors
    researchExec := executors.NewAgentExecutor("research", researcher)
    analyzeExec := executors.NewAgentExecutor("analyze", analyzer)
    writeExec := executors.NewAgentExecutor("write", writer)

    // Build the workflow DAG: research -> analyze -> write
    wf, err := workflow.NewBuilder(researchExec).
        WithName("ResearchPipeline").
        AddExecutor(analyzeExec).
        AddExecutor(writeExec).
        AddEdge("research", "analyze").
        AddEdge("analyze", "write").
        MarkAsOutput("write").
        Build()
    if err != nil {
        log.Fatal(err)
    }

    // Run the workflow
    runner := workflow.NewRunner(wf)
    result, err := runner.Run(ctx, "Explain the benefits of microservices architecture")
    if err != nil {
        log.Fatal(err)
    }

    // Print outputs from the final executor
    for _, msg := range result.Outputs {
        fmt.Println(msg.Content.Text())
    }
}
```

### Fan-Out / Fan-In Pattern

Process data through multiple agents in parallel:

```go
// Build workflow with parallel processing
wf, _ := workflow.NewBuilder(inputExec).
    AddExecutors(analysisA, analysisB, analysisC).
    AddExecutor(aggregator).
    AddFanOut("input", "analysisA", "analysisB", "analysisC").
    AddFanIn([]string{"analysisA", "analysisB", "analysisC"}, "aggregator").
    MarkAsOutput("aggregator").
    Build()
```

### Conditional Routing

Route messages based on content:

```go
// Add conditional edge based on message content
wf, _ := workflow.NewBuilder(classifier).
    AddExecutors(positiveHandler, negativeHandler, neutralHandler).
    SwitchFrom("classifier").
        Case(func(msg any) bool {
            return strings.Contains(msg.(agent.Message).Text(), "positive")
        }, "positiveHandler").
        Case(func(msg any) bool {
            return strings.Contains(msg.(agent.Message).Text(), "negative")
        }, "negativeHandler").
        Default("neutralHandler").
    Build()
```

## A2A Protocol (Agent-to-Agent)

The A2A protocol enables standardized communication between AI agents over HTTP. Use it to connect agents across services and organizations.

### A2A Server

Expose your agent as an A2A-compliant endpoint:

```go
package main

import (
    "log"
    "net/http"
    "os"

    "github.com/microsoft/agent-framework-go/chatagent"
    "github.com/microsoft/agent-framework-go/protocol/a2a"
    "github.com/microsoft/agent-framework-go/providers/openai"
)

func main() {
    client, _ := openai.NewClient(
        openai.WithAPIKey(os.Getenv("OPENAI_API_KEY")),
        openai.WithModel("gpt-4o"),
    )

    // Create your agent
    myAgent := chatagent.New(client,
        chatagent.WithName("HelpfulAssistant"),
        chatagent.WithDescription("A helpful AI assistant"),
        chatagent.WithInstructions("You are a helpful assistant."),
    )

    // Create A2A server with custom agent card
    server := a2a.NewServer(myAgent, a2a.WithAgentCard(&a2a.AgentCard{
        Name:        "HelpfulAssistant",
        Description: "A helpful AI assistant accessible via A2A protocol",
        Capabilities: &a2a.AgentCapabilities{
            Streaming: true,
        },
    }))

    // Mount the A2A handler
    http.Handle("/a2a/", http.StripPrefix("/a2a", server.Handler()))

    log.Println("A2A server listening on :8080")
    log.Fatal(http.ListenAndServe(":8080", nil))
}
```

### A2A Client

Connect to remote A2A agents:

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/microsoft/agent-framework-go/protocol/a2a"
)

func main() {
    ctx := context.Background()

    // Create A2A client
    client := a2a.NewClient("http://remote-agent.example.com/a2a")

    // Discover agent capabilities
    card, err := client.GetAgentCard(ctx)
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("Connected to: %s\n", card.Name)

    // Create a task
    task, err := client.CreateTask(ctx, &a2a.CreateTaskRequest{
        Message: a2a.NewTextMessage(a2a.RoleUser, "Hello, can you help me?"),
    })
    if err != nil {
        log.Fatal(err)
    }

    // Send follow-up messages
    task, err = client.SendMessage(ctx, task.ID, a2a.NewTextMessage(
        a2a.RoleUser,
        "What's the weather like today?",
    ))
    if err != nil {
        log.Fatal(err)
    }

    // Print agent response
    for _, msg := range task.Messages {
        if msg.Role == a2a.RoleAgent {
            fmt.Printf("Agent: %s\n", a2a.MessageText(&msg))
        }
    }
}
```

### A2A Agent Wrapper

Use remote A2A agents as local agents:

```go
// Create an A2A agent that wraps a remote endpoint
remoteAgent := a2a.NewA2AAgent("http://remote-agent.example.com/a2a")

// Use it like any other agent
response, _ := remoteAgent.Run(ctx, []agent.Message{
    agent.NewUserMessage("Tell me about AI agents"),
})
fmt.Println(response.Text())
```

## AG-UI Protocol (Agent-to-UI)

The AG-UI protocol provides real-time streaming of agent responses to UI clients using Server-Sent Events (SSE).

```go
package main

import (
    "log"
    "net/http"
    "os"

    "github.com/microsoft/agent-framework-go/chatagent"
    "github.com/microsoft/agent-framework-go/protocol/agui"
    "github.com/microsoft/agent-framework-go/providers/openai"
)

func main() {
    client, _ := openai.NewClient(
        openai.WithAPIKey(os.Getenv("OPENAI_API_KEY")),
        openai.WithModel("gpt-4o"),
    )

    // Create your agent
    myAgent := chatagent.New(client,
        chatagent.WithName("StreamingAssistant"),
        chatagent.WithInstructions("You are a helpful assistant."),
    )

    // Create AG-UI server
    server := agui.NewServer(myAgent)

    // Mount the AG-UI handler
    http.Handle("/agui/", http.StripPrefix("/agui", server.Handler()))

    // Serve a simple HTML page for testing
    http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Content-Type", "text/html")
        w.Write([]byte(`<!DOCTYPE html>
<html>
<body>
    <h1>AG-UI Demo</h1>
    <div id="output"></div>
    <script>
        fetch('/agui/stream', {
            method: 'POST',
            headers: {'Content-Type': 'application/json'},
            body: JSON.stringify({
                messages: [{role: 'user', content: 'Tell me a story'}]
            })
        }).then(response => {
            const reader = response.body.getReader();
            const decoder = new TextDecoder();
            function read() {
                reader.read().then(({done, value}) => {
                    if (done) return;
                    document.getElementById('output').innerHTML += decoder.decode(value);
                    read();
                });
            }
            read();
        });
    </script>
</body>
</html>`))
    })

    log.Println("AG-UI server listening on :8080")
    log.Fatal(http.ListenAndServe(":8080", nil))
}
```

AG-UI events include:

- `RUN_STARTED` / `RUN_FINISHED` / `RUN_ERROR` - Lifecycle events
- `TEXT_MESSAGE_START` / `TEXT_MESSAGE_CONTENT` / `TEXT_MESSAGE_END` - Text streaming
- `TOOL_CALL_START` / `TOOL_CALL_ARGS` / `TOOL_CALL_END` / `TOOL_CALL_RESULT` - Tool calls
- `STATE_SNAPSHOT` / `STATE_DELTA` - State synchronization

## Group Chat Orchestration

Orchestrate conversations between multiple agents with pluggable speaker selection strategies:

```go
package main

import (
    "context"
    "fmt"
    "log"
    "os"

    "github.com/microsoft/agent-framework-go/agent"
    "github.com/microsoft/agent-framework-go/chatagent"
    "github.com/microsoft/agent-framework-go/providers/openai"
    "github.com/microsoft/agent-framework-go/workflow/groupchat"
)

func main() {
    ctx := context.Background()

    client, _ := openai.NewClient(
        openai.WithAPIKey(os.Getenv("OPENAI_API_KEY")),
        openai.WithModel("gpt-4o"),
    )

    // Create participating agents
    techExpert := chatagent.New(client,
        chatagent.WithName("TechExpert"),
        chatagent.WithDescription("Expert in technology and software"),
        chatagent.WithInstructions("You are a technology expert. Provide technical insights."),
    )

    businessAnalyst := chatagent.New(client,
        chatagent.WithName("BusinessAnalyst"),
        chatagent.WithDescription("Expert in business strategy"),
        chatagent.WithInstructions("You are a business analyst. Focus on ROI and strategy."),
    )

    moderator := chatagent.New(client,
        chatagent.WithName("Moderator"),
        chatagent.WithDescription("Discussion moderator"),
        chatagent.WithInstructions("You moderate discussions and summarize key points."),
    )

    agents := []agent.Agent{techExpert, businessAnalyst, moderator}

    // Create group chat manager with round-robin selection
    selector, _ := groupchat.NewRoundRobinSelector(agents...)
    manager, _ := groupchat.NewManager(agents,
        groupchat.WithSelector(selector),
        groupchat.WithMaxTurns(6),
        groupchat.WithTerminationCondition(groupchat.KeywordCondition("DONE")),
    )

    // Run the group chat
    result, err := manager.Run(ctx, "Discuss the pros and cons of adopting AI in enterprise")
    if err != nil {
        log.Fatal(err)
    }

    // Print the conversation
    fmt.Println("=== Group Chat Transcript ===")
    for _, entry := range result.Transcript.Entries {
        fmt.Printf("[%s]: %s\n\n", entry.SpeakerName, entry.Message.Text())
    }
}
```

### LLM-Based Speaker Selection

Use an AI to intelligently select the next speaker:

```go
// Create a decision agent for speaker selection
decisionAgent := chatagent.New(client,
    chatagent.WithName("Moderator"),
    chatagent.WithInstructions("Select the most appropriate next speaker."),
)

// Create LLM selector
selector, _ := groupchat.NewLLMSelector(decisionAgent, agents,
    groupchat.WithSelectionInstructions(`
        Based on the conversation, select the participant who can best 
        contribute to the current topic. Consider expertise and recent participation.
    `),
)

manager, _ := groupchat.NewManager(agents,
    groupchat.WithSelector(selector),
    groupchat.WithMaxTurns(10),
)
```

### Streaming Group Chat

Stream events as the group chat progresses:

```go
events, _ := manager.RunStream(ctx, "Start the discussion")

for event := range events {
    switch event.Kind {
    case groupchat.EventKindSpeakerSelected:
        fmt.Printf("\n--- %s's turn ---\n", event.SpeakerName)
    case groupchat.EventKindAgentResponseUpdate:
        if event.ResponseUpdate != nil && event.ResponseUpdate.Delta != nil {
            fmt.Print(event.ResponseUpdate.Delta.TextDelta)
        }
    case groupchat.EventKindCompleted:
        fmt.Println("\n\n=== Discussion Complete ===")
    }
}
```

## Documentation

- [Go Package Documentation](https://pkg.go.dev/github.com/microsoft/agent-framework-go)
- [Agent Framework Overview](https://learn.microsoft.com/agent-framework/overview/agent-framework-overview)
- [Quick Start Guide](https://learn.microsoft.com/agent-framework/tutorials/quick-start)
- [User Guide](https://learn.microsoft.com/en-us/agent-framework/user-guide/overview)
- [Design Documents](../docs/design/)
- [Architectural Decision Records](../docs/decisions/)

## Package Structure

```text
github.com/microsoft/agent-framework-go/
├── agent/          # Core Agent interface, response types, and ContextProvider
├── chat/           # ChatClient interface and message types
├── chatagent/      # ChatClientAgent implementation
├── tool/           # Tool interface and FunctionTool
├── middleware/     # Middleware pipeline
├── memory/         # Context and history providers
├── workflow/       # DAG-based workflow orchestration
├── observability/  # OpenTelemetry integration
├── providers/      # LLM provider implementations
│   ├── openai/     # OpenAI API
│   ├── azure/      # Azure OpenAI
│   ├── anthropic/  # Anthropic Claude
│   ├── bedrock/    # AWS Bedrock
│   └── ollama/     # Local Ollama models
├── protocol/       # Protocol implementations
│   ├── a2a/        # Agent-to-Agent protocol
│   └── agui/       # Agent-UI protocol
└── hosting/        # HTTP/gRPC server hosting
```

## Contributing

See [CONTRIBUTING.md](../CONTRIBUTING.md) for guidelines on contributing to the Agent Framework.

## Community

- Join our [Discord channel](https://discord.gg/b5zjErwbQM) for discussions and support
- Participate in [weekly office hours](../COMMUNITY.md#public-community-office-hours)
- File issues on [GitHub](https://github.com/microsoft/agent-framework/issues)

## License

This project is licensed under the MIT License. See [LICENSE](../LICENSE) for details.
