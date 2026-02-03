// Copyright (c) Microsoft. All rights reserved.

/*
Package agent provides core abstractions for building AI agents in Go.

The agent package defines the fundamental [Agent] interface that all agent
implementations must satisfy. This interface provides a consistent contract
for conversational AI interactions, streaming responses, session management,
and service resolution.

# Agent Interface

The [Agent] interface is the primary abstraction for AI agents:

	type Agent interface {
		ID() string
		Name() string
		Description() string
		Metadata() AIAgentMetadata
		Run(ctx context.Context, messages []Message, opts ...RunOption) (*Response, error)
		RunStream(ctx context.Context, messages []Message, opts ...RunOption) (<-chan ResponseUpdate, error)
		NewSession(ctx context.Context) (Session, error)
		RestoreSession(ctx context.Context, data json.RawMessage) (Session, error)
		GetService(serviceType reflect.Type) interface{}
	}

# Running Agents

Use Run for synchronous execution that returns a complete response:

	response, err := agent.Run(ctx, messages)
	if err != nil {
		return err
	}
	fmt.Println(response.Text())

Use RunStream for incremental streaming of response content:

	updates, err := agent.RunStream(ctx, messages)
	if err != nil {
		return err
	}
	for update := range updates {
		if update.Kind == UpdateKindContentDelta {
			fmt.Print(update.Delta.TextDelta)
		}
	}

# Session Management

Sessions maintain conversation state across multiple agent runs:

	// Create a new session
	session, err := agent.NewSession(ctx)
	if err != nil {
		return err
	}

	// Run with session context
	response, err := agent.Run(ctx, messages, WithSession(session))

	// Persist session state
	data, err := session.Serialize()
	// ... save data ...

	// Restore session later
	restored, err := agent.RestoreSession(ctx, data)

# Service Resolution

The GetService method enables extensibility through a service locator pattern.
Use the generic [GetService] helper for type-safe service retrieval:

	logger, ok := GetService[Logger](agent)
	if ok {
		logger.Info("Agent running")
	}

# Options

Use [RunOption] functions to customize agent behavior:

	response, err := agent.Run(ctx, messages,
		WithMaxTokens(1000),
		WithTemperature(0.7),
		WithSession(session),
	)

# Middleware

The agent package provides three middleware types for intercepting different
levels of agent execution:

  - [AgentMiddleware]: Intercepts full agent invocations (Run/RunStream calls)
  - [FunctionMiddleware]: Intercepts tool/function invocations during execution
  - [ChatMiddleware]: Intercepts chat client requests (GetResponse/GetStreamingResponse)

Use [ChainAgentMiddleware], [ChainFunctionMiddleware], and [ChainChatMiddleware]
to compose multiple middlewares into a single middleware.

ChatMiddleware operates at the lowest level, intercepting each individual chat
client request. This is useful for:

  - Request caching and memoization
  - Rate limiting and throttling
  - Request/response logging
  - Message transformation

Example logging middleware:

	type LoggingChatMiddleware struct {
		Logger *slog.Logger
	}

	func (m *LoggingChatMiddleware) Process(ctx context.Context, chatCtx *ChatContext, next ChatHandler) error {
		m.Logger.Info("Chat request", "messages", len(chatCtx.Messages))
		start := time.Now()
		err := next(ctx, chatCtx)
		m.Logger.Info("Chat response", "duration", time.Since(start))
		return err
	}

See the subpackages for specific agent implementations:

  - chatagent: ChatClientAgent for chat completion-based agents
  - providers/openai: OpenAI provider integration
  - providers/azure: Azure OpenAI provider integration
*/
package agent
