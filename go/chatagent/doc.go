// Copyright (c) Microsoft. All rights reserved.

// Package chatagent provides ChatClientAgent, the primary agent implementation
// that wraps a chat.Client to provide the full agent.Agent interface.
//
// ChatClientAgent enables building AI agents from any chat completion provider
// (OpenAI, Azure OpenAI, Anthropic, etc.) with automatic tool invocation,
// session management, and streaming support.
//
// # Basic Usage
//
// Create an agent from a chat client:
//
//	client, _ := openai.NewClient(openai.WithAPIKey("sk-..."))
//	agent := chatagent.New(client,
//	    chatagent.WithInstructions("You are a helpful assistant."),
//	)
//
//	resp, err := agent.Run(ctx, []agent.Message{
//	    agent.NewUserMessage("Hello!"),
//	})
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Println(resp.Text())
//
// # Tool Integration
//
// Add tools for the agent to use:
//
//	getWeather := tool.Func(weatherFunc, "Get current weather",
//	    tool.WithName("get_weather"),
//	)
//
//	agent := chatagent.New(client,
//	    chatagent.WithTools(getWeather),
//	    chatagent.WithMaxTurns(10),
//	)
//
// The agent automatically handles the tool invocation loop, calling tools
// and feeding results back to the model until a final response is generated.
//
// # Streaming
//
// Use RunStream for incremental response updates:
//
//	updates, err := agent.RunStream(ctx, messages)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	for update := range updates {
//	    if update.Kind == agent.UpdateKindContentDelta {
//	        fmt.Print(update.Delta.TextDelta)
//	    }
//	}
//
// # Session Management
//
// Sessions maintain conversation state across multiple runs:
//
//	session, _ := agent.NewSession(ctx)
//
//	resp1, _ := agent.Run(ctx, []agent.Message{
//	    agent.NewUserMessage("Remember my name is Alice."),
//	}, agent.WithSession(session))
//	session.AddMessage(resp1.Messages[len(resp1.Messages)-1])
//
//	resp2, _ := agent.Run(ctx, []agent.Message{
//	    agent.NewUserMessage("What's my name?"),
//	}, agent.WithSession(session))
//	// Response will know the name is Alice
//
// # Builder Pattern
//
// For more complex configuration, use the fluent builder:
//
//	agent, err := chatagent.NewBuilder(client).
//	    Name("WeatherBot").
//	    Instructions("You help users with weather queries.").
//	    Tools(weatherTool, forecastTool).
//	    MaxTurns(20).
//	    Build()
//
// # Observability
//
// ChatClientAgent works with the observability package for tracing and metrics:
//
//	instrumentedClient := observability.NewInstrumentedClient(client)
//	agent := chatagent.New(instrumentedClient, opts...)
//
// This automatically traces all chat completions and records token usage.
package chatagent
