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
├── agent/          # Core Agent interface and response types
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
