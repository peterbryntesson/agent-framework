// Copyright (c) Microsoft. All rights reserved.

// Package openai provides a chat.Client implementation for OpenAI's Chat Completions API.
//
// This package implements the [chat.Client] interface for OpenAI models including GPT-4o,
// GPT-4, and GPT-3.5-turbo. It supports both synchronous and streaming chat completions,
// tool/function calling, vision capabilities, and the Responses API for stateful conversations.
//
// # Client Creation
//
// Create a client using [NewClient] with functional options:
//
//	// Use OPENAI_API_KEY environment variable
//	client, err := openai.NewClient()
//
//	// Or provide API key explicitly
//	client, err := openai.NewClient(
//	    openai.WithAPIKey("sk-..."),
//	    openai.WithModel("gpt-4o"),
//	)
//
// # Configuration Options
//
// The client supports several configuration options:
//
//   - [WithAPIKey]: Set the OpenAI API key (falls back to OPENAI_API_KEY env var)
//   - [WithModel]: Set the model to use (default: "gpt-4o")
//   - [WithBaseURL]: Set a custom API endpoint for proxies or compatible APIs
//   - [WithOrgID]: Set the OpenAI organization ID for multi-org accounts
//   - [WithInstructionRole]: Set role for system instructions ("system" or "developer")
//   - [WithHTTPClient]: Provide a custom HTTP client for request handling
//
// # Basic Usage
//
// Send a chat completion request:
//
//	client, err := openai.NewClient()
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	messages := []chat.Message{
//	    chat.NewSystemMessage("You are a helpful assistant."),
//	    chat.NewUserMessage("Hello, how are you?"),
//	}
//
//	response, err := client.GetResponse(ctx, messages, nil)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	fmt.Println(response.Text())
//
// # Streaming Responses
//
// Stream responses for real-time output:
//
//	updates, err := client.GetStreamingResponse(ctx, messages, nil)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	for update := range updates {
//	    switch update.Kind {
//	    case chat.UpdateKindContentDelta:
//	        fmt.Print(update.Delta.TextDelta)
//	    case chat.UpdateKindDone:
//	        fmt.Println()
//	    case chat.UpdateKindError:
//	        log.Printf("Error: %v", update.Error)
//	    }
//	}
//
// # Tool Calling
//
// The client supports OpenAI's function calling / tools feature through the
// [chat.Options.Tools] field. Define tools using the tool package and pass
// them in the options:
//
//	import "github.com/microsoft/agent-framework-go/tool"
//
//	weatherTool := tool.Func(getWeather, "Get current weather for a location")
//
//	options := &chat.Options{
//	    Tools: []chat.ToolDefinition{
//	        {Name: weatherTool.Name(), Description: weatherTool.Description()},
//	    },
//	    ToolChoice: "auto",
//	}
//
//	response, err := client.GetResponse(ctx, messages, options)
//
// # Responses API
//
// For stateful conversations with hosted tools (web search, code interpreter),
// use [NewResponsesClient]:
//
//	responsesClient, err := openai.NewResponsesClient(
//	    openai.WithResponsesAPIKey("sk-..."),
//	    openai.WithResponsesModel("gpt-4o"),
//	)
//
// # Error Handling
//
// The client returns errors for various failure conditions:
//
//   - Missing API key: Returned when no API key is provided and OPENAI_API_KEY is not set
//   - Network errors: HTTP transport failures wrapped with context
//   - API errors: Rate limiting, authentication failures, and server errors
//   - Context cancellation: When the context is cancelled during a request
//
// # Observability
//
// The client populates [chat.ClientMetadata] with provider information for
// OpenTelemetry integration and logging. Access via the Metadata() method.
//
// # Compatibility
//
// This package is designed to work with OpenAI-compatible APIs (e.g., Azure OpenAI,
// local proxies) via the WithBaseURL option. However, for Azure OpenAI, prefer
// the dedicated azure package which handles authentication and deployment mapping.
package openai
