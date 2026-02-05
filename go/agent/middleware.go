// Copyright (c) Microsoft. All rights reserved.

package agent

import (
	"context"
)

// AgentHandler is the next handler in the agent middleware chain.
type AgentHandler func(ctx context.Context, agentCtx *AgentContext) error

// AgentMiddleware intercepts agent invocations.
// Implement this interface to add cross-cutting behavior like
// logging, security validation, or request modification.
type AgentMiddleware interface {
	// Process handles an agent invocation.
	// Call next to continue the chain, or return early to short-circuit.
	// The agentCtx contains input data and receives the response.
	Process(ctx context.Context, agentCtx *AgentContext, next AgentHandler) error
}

// AgentMiddlewareFunc is a function adapter for AgentMiddleware.
// Use this to create middleware from anonymous functions.
type AgentMiddlewareFunc func(ctx context.Context, agentCtx *AgentContext, next AgentHandler) error

// Process implements AgentMiddleware.
func (f AgentMiddlewareFunc) Process(ctx context.Context, agentCtx *AgentContext, next AgentHandler) error {
	return f(ctx, agentCtx, next)
}

// FunctionHandler is the next handler in the function middleware chain.
type FunctionHandler func(ctx context.Context, funcCtx *FunctionContext) error

// FunctionMiddleware intercepts tool/function invocations.
// Implement this interface to add validation, caching, or
// transformation of function calls and results.
type FunctionMiddleware interface {
	// Process handles a function invocation.
	// Call next to continue the chain, or return early to short-circuit.
	// Set funcCtx.Result or funcCtx.Error to provide a response.
	Process(ctx context.Context, funcCtx *FunctionContext, next FunctionHandler) error
}

// FunctionMiddlewareFunc is a function adapter for FunctionMiddleware.
type FunctionMiddlewareFunc func(ctx context.Context, funcCtx *FunctionContext, next FunctionHandler) error

// Process implements FunctionMiddleware.
func (f FunctionMiddlewareFunc) Process(ctx context.Context, funcCtx *FunctionContext, next FunctionHandler) error {
	return f(ctx, funcCtx, next)
}

// ChatHandler is the next handler in the chat middleware chain.
type ChatHandler func(ctx context.Context, chatCtx *ChatContext) error

// ChatMiddleware intercepts chat client requests.
// Implement this interface to add caching, request transformation,
// rate limiting, or logging at the chat client level.
// This operates at a lower level than AgentMiddleware, intercepting
// each individual GetResponse or GetStreamingResponse call.
type ChatMiddleware interface {
	// Process handles a chat client request.
	// Call next to continue the chain, or return early to short-circuit.
	// The chatCtx contains request data and receives the response.
	// For streaming requests, IsStreaming is true and Stream should be set.
	Process(ctx context.Context, chatCtx *ChatContext, next ChatHandler) error
}

// ChatMiddlewareFunc is a function adapter for ChatMiddleware.
// Use this to create middleware from anonymous functions.
type ChatMiddlewareFunc func(ctx context.Context, chatCtx *ChatContext, next ChatHandler) error

// Process implements ChatMiddleware.
func (f ChatMiddlewareFunc) Process(ctx context.Context, chatCtx *ChatContext, next ChatHandler) error {
	return f(ctx, chatCtx, next)
}

// Compile-time interface satisfaction checks.
var (
	_ ChatMiddleware = ChatMiddlewareFunc(nil)
)
