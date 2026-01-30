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

	logger, ok := agent.GetService[Logger](agent)
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

See the subpackages for specific agent implementations:

  - chatagent: ChatClientAgent for chat completion-based agents
  - providers/openai: OpenAI provider integration
  - providers/azure: Azure OpenAI provider integration
*/
package agent
