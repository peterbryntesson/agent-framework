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
// # Agent as Tool
//
// The AsTool function converts an agent to a tool for use by other agents,
// enabling hierarchical agent patterns:
//
//	// Create a specialized research agent
//	researcher := chatagent.New(researchClient,
//	    chatagent.WithName("Researcher"),
//	    chatagent.WithDescription("Performs in-depth research on topics"),
//	    chatagent.WithInstructions("You are a research specialist..."),
//	)
//
//	// Convert to tool with custom options
//	researchTool := chatagent.AsTool(researcher, chatagent.AsToolOptions{
//	    Name:           "research",
//	    Description:    "Research a topic thoroughly",
//	    ArgName:        "topic",
//	    ArgDescription: "The topic to research",
//	})
//
//	// Use in an orchestrator agent
//	orchestrator := chatagent.New(client,
//	    chatagent.WithName("Orchestrator"),
//	    chatagent.WithTools(researchTool),
//	)
//
// # Streaming Sub-Agents
//
// Use StreamCallback to receive incremental updates from sub-agents:
//
//	researchTool := chatagent.AsTool(researcher, chatagent.AsToolOptions{
//	    StreamCallback: func(update agent.ResponseUpdate) {
//	        if update.Delta != nil && update.Delta.TextDelta != "" {
//	            fmt.Print(update.Delta.TextDelta)
//	        }
//	    },
//	})
//
// # Runtime Context Propagation
//
// Forward runtime context (user IDs, API tokens, session data) to sub-agents:
//
//	researchTool := chatagent.AsTool(researcher, chatagent.AsToolOptions{
//	    ForwardRuntimeContext: true,
//	})
//
//	// Run with runtime context that propagates to all sub-agents
//	ctx := agent.WithRuntimeCtx(context.Background(),
//	    agent.NewRuntimeContext().
//	        With("user_id", "u123").
//	        With("api_token", "tok_abc"),
//	)
//
//	resp, err := orchestrator.Run(ctx, messages)
//
// Session-related keys (session_id, conversation_id, thread_id) are automatically
// excluded from propagation. Use ExcludeKeys to exclude additional keys.
//
// # Hierarchical Agent Orchestration
//
// Complex tasks can be broken down using multiple specialized agents:
//
//	// Create specialized agents
//	researcher := chatagent.New(client,
//	    chatagent.WithName("Researcher"),
//	    chatagent.WithInstructions("You research topics thoroughly."),
//	)
//
//	writer := chatagent.New(client,
//	    chatagent.WithName("Writer"),
//	    chatagent.WithInstructions("You write clear, engaging content."),
//	)
//
//	coder := chatagent.New(client,
//	    chatagent.WithName("Coder"),
//	    chatagent.WithInstructions("You write clean, tested code."),
//	)
//
//	// Convert agents to tools with context forwarding
//	tools := []tool.Tool{
//	    chatagent.AsTool(researcher, chatagent.AsToolOptions{
//	        Name:                  "research",
//	        Description:           "Research a topic",
//	        ForwardRuntimeContext: true,
//	    }),
//	    chatagent.AsTool(writer, chatagent.AsToolOptions{
//	        Name:                  "write",
//	        Description:           "Write content based on research",
//	        ForwardRuntimeContext: true,
//	    }),
//	    chatagent.AsTool(coder, chatagent.AsToolOptions{
//	        Name:                  "code",
//	        Description:           "Implement code solutions",
//	        ForwardRuntimeContext: true,
//	    }),
//	}
//
//	// Create orchestrator that uses specialized agents
//	orchestrator := chatagent.New(client,
//	    chatagent.WithName("Orchestrator"),
//	    chatagent.WithInstructions(`You coordinate complex tasks by delegating to:
//	- research: for gathering information
//	- write: for creating content
//	- code: for implementing solutions`),
//	    chatagent.WithTools(tools...),
//	)
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
