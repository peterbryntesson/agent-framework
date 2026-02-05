// Copyright (c) Microsoft. All rights reserved.

// Package tool provides interfaces and types for AI agent tools.
//
// This package defines the core abstractions for tools that can be invoked
// by AI agents during conversation. Tools enable agents to perform actions
// like searching the web, executing code, or calling external APIs.
//
// # Tool Interface
//
// The Tool interface represents any callable function that agents can invoke:
//
//	type Tool interface {
//	    Name() string
//	    Description() string
//	    Parameters() json.RawMessage
//	    Invoke(ctx context.Context, arguments json.RawMessage) (Result, error)
//	}
//
// # Hosted Tools
//
// HostedTool represents tools that run on the provider's infrastructure
// rather than locally. Examples include web search, code interpreter,
// and file search capabilities offered by providers like OpenAI.
//
// # Function Tools
//
// FunctionTool wraps Go functions to make them callable by AI models.
// Schema generation uses struct tags to define parameter types and validation.
// See the function.go file for the FunctionTool implementation and Func decorator.
//
// # Invocation
//
// The Invoker type handles safe tool invocation with error handling,
// panic recovery, and batch execution support. Configure invocation
// behavior using InvocationConfig.
//
// # Usage
//
// Create tools using the Func decorator:
//
//	type WeatherArgs struct {
//	    Location string `json:"location" description:"The city name" required:"true"`
//	    Unit     string `json:"unit" description:"Temperature unit" enum:"celsius,fahrenheit"`
//	}
//
//	func GetWeather(ctx context.Context, args WeatherArgs) (string, error) {
//	    return fmt.Sprintf("Weather in %s: 72°F", args.Location), nil
//	}
//
//	weatherTool := tool.Func(GetWeather, "Get current weather for a location")
//
// This package aligns with .NET AITool and Python ToolProtocol interfaces
// for consistency across the Agent Framework implementations.
package tool
