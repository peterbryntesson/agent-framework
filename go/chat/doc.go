// Copyright (c) Microsoft. All rights reserved.

/*
Package chat provides abstractions for chat completion providers.

The chat package defines the [Client] interface that LLM providers implement
to provide a consistent API for chat completions. This abstraction enables
agents to work with any provider (OpenAI, Azure OpenAI, Anthropic, etc.)
through a common interface.

# Client Interface

The [Client] interface is the primary abstraction for chat providers:

	type Client interface {
		GetResponse(ctx context.Context, messages []Message, options *Options) (*Response, error)
		GetStreamingResponse(ctx context.Context, messages []Message, options *Options) (<-chan ResponseUpdate, error)
		Metadata() ClientMetadata
	}

# Getting Responses

Use GetResponse for synchronous chat completions:

	client := openai.NewClient(openai.WithAPIKey(apiKey))

	messages := []chat.Message{
		chat.NewSystemMessage("You are a helpful assistant."),
		chat.NewUserMessage("Hello, how are you?"),
	}

	response, err := client.GetResponse(ctx, messages, nil)
	if err != nil {
		return err
	}
	fmt.Println(response.Text())

# Streaming Responses

Use GetStreamingResponse for incremental content delivery:

	updates, err := client.GetStreamingResponse(ctx, messages, nil)
	if err != nil {
		return err
	}
	for update := range updates {
		switch update.Kind {
		case chat.UpdateKindContentDelta:
			fmt.Print(update.Delta.TextDelta)
		case chat.UpdateKindError:
			return update.Error
		}
	}

# Options

Configure requests using [Options]:

	opts := &chat.Options{
		MaxTokens:   1000,
		Temperature: 0.7,
	}
	response, err := client.GetResponse(ctx, messages, opts)

# Provider Metadata

Access provider information for logging and observability:

	meta := client.Metadata()
	fmt.Printf("Provider: %s, Model: %s\n", meta.ProviderName, meta.ModelID)

# Message Types

Create messages using convenience constructors:

	system := chat.NewSystemMessage("You are a helpful assistant.")
	user := chat.NewUserMessage("What is the weather?")
	assistant := chat.NewAssistantMessage("I'd be happy to help with the weather.")

See the subpackages for specific provider implementations:

  - providers/openai: OpenAI API client
  - providers/azure: Azure OpenAI API client
  - providers/anthropic: Anthropic Claude API client
*/
package chat
