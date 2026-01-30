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
