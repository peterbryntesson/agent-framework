// Copyright (c) Microsoft. All rights reserved.

// Package agentframework provides the Microsoft Agent Framework SDK for Go.
//
// The Agent Framework enables developers to build AI agents that can
// interact with users through natural language conversations, execute
// tools, and orchestrate complex workflows.
//
// # Getting Started
//
// The most common way to create an agent is using the chatagent package
// with a chat client provider:
//
//	client, _ := openai.NewClient(
//	    openai.WithAPIKey("sk-..."),
//	    openai.WithModel("gpt-4o"),
//	)
//
//	agent := chatagent.New(client,
//	    chatagent.WithName("MyAgent"),
//	    chatagent.WithInstructions("You are a helpful assistant."),
//	)
//
//	resp, err := agent.Run(ctx, messages)
//
// # Package Organization
//
// The SDK is organized into the following packages:
//
//   - agent: Core agent interfaces and types
//   - chat: Chat client abstractions
//   - chatagent: ChatClient-backed agent implementation
//   - tool: Function calling and tool system
//   - middleware: Request/response interceptors
//   - memory: Context and history providers
//   - thread: Conversation threading
//   - workflow: DAG-based orchestration
//   - protocol/a2a: Agent-to-Agent protocol
//   - protocol/agui: Agent-UI protocol
//   - providers/*: LLM provider implementations
//   - observability: OpenTelemetry integration
//   - hosting: HTTP/gRPC server infrastructure
//
// # Providers
//
// The SDK supports multiple LLM providers:
//
//   - providers/openai: OpenAI API
//   - providers/azure: Azure OpenAI
//   - providers/anthropic: Anthropic Claude
//   - providers/bedrock: AWS Bedrock
//   - providers/ollama: Local models via Ollama
package agentframework
