// Copyright (c) Microsoft. All rights reserved.

// Package declarative provides support for loading agents from YAML definitions.
//
// This package enables agents to be defined declaratively in YAML files and
// loaded dynamically at runtime. It provides schema-compatible definitions
// with the .NET and Python implementations.
//
// # YAML Schema
//
// Agents are defined using the PromptAgent schema:
//
//	kind: Prompt
//	name: MyAssistant
//	description: A helpful assistant
//	instructions: You are a helpful assistant.
//	model:
//	    id: gpt-4o
//	    provider: OpenAI
//	    apiType: Chat
//	    connection:
//	        kind: key
//	        key: =Env.OPENAI_API_KEY
//	tools:
//	  - kind: function
//	    name: get_weather
//	    description: Get current weather
//
// # Loading Agents
//
// Use the AgentFactory to load and instantiate agents from YAML:
//
//	factory := declarative.NewAgentFactory().
//	    WithBinding("get_weather", weatherFunc)
//
//	agent, err := factory.CreateFromFile(ctx, "agent.yaml")
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	response, err := agent.Run(ctx, messages)
//
// # Environment Variables
//
// The package supports PowerFx-style environment variable substitution
// using the =Env.VAR_NAME syntax:
//
//	model:
//	    id: =Env.OPENAI_MODEL
//	    connection:
//	        key: =Env.OPENAI_API_KEY
//
// # Provider Support
//
// Built-in provider support includes:
//   - OpenAI (Chat API)
//   - Azure OpenAI (Chat and Responses APIs)
//
// Custom providers can be registered using WithProvider.
package declarative
