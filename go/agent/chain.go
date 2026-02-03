// Copyright (c) Microsoft. All rights reserved.

package agent

import (
	"context"
)

// ChainAgentMiddleware composes multiple AgentMiddleware into a single middleware.
// Middlewares execute in the order provided: first middleware runs first,
// and its next() calls the second middleware, and so on.
func ChainAgentMiddleware(middlewares ...AgentMiddleware) AgentMiddleware {
	n := len(middlewares)
	if n == 0 {
		return AgentMiddlewareFunc(func(ctx context.Context, agentCtx *AgentContext, next AgentHandler) error {
			return next(ctx, agentCtx)
		})
	}
	if n == 1 {
		return middlewares[0]
	}

	return AgentMiddlewareFunc(func(ctx context.Context, agentCtx *AgentContext, next AgentHandler) error {
		// Build chain from the last middleware backwards
		chain := next
		for i := n - 1; i >= 0; i-- {
			mw := middlewares[i]
			currentNext := chain
			chain = func(ctx context.Context, agentCtx *AgentContext) error {
				return mw.Process(ctx, agentCtx, currentNext)
			}
		}
		return chain(ctx, agentCtx)
	})
}

// ChainFunctionMiddleware composes multiple FunctionMiddleware into a single middleware.
// Middlewares execute in the order provided: first middleware runs first,
// and its next() calls the second middleware, and so on.
func ChainFunctionMiddleware(middlewares ...FunctionMiddleware) FunctionMiddleware {
	n := len(middlewares)
	if n == 0 {
		return FunctionMiddlewareFunc(func(ctx context.Context, funcCtx *FunctionContext, next FunctionHandler) error {
			return next(ctx, funcCtx)
		})
	}
	if n == 1 {
		return middlewares[0]
	}

	return FunctionMiddlewareFunc(func(ctx context.Context, funcCtx *FunctionContext, next FunctionHandler) error {
		chain := next
		for i := n - 1; i >= 0; i-- {
			mw := middlewares[i]
			currentNext := chain
			chain = func(ctx context.Context, funcCtx *FunctionContext) error {
				return mw.Process(ctx, funcCtx, currentNext)
			}
		}
		return chain(ctx, funcCtx)
	})
}

// ChainChatMiddleware composes multiple ChatMiddleware into a single middleware.
// Middlewares execute in the order provided: first middleware runs first,
// and its next() calls the second middleware, and so on.
// An empty or nil slice returns a passthrough middleware that calls next directly.
func ChainChatMiddleware(middlewares ...ChatMiddleware) ChatMiddleware {
	// Filter out nil middlewares
	filtered := make([]ChatMiddleware, 0, len(middlewares))
	for _, m := range middlewares {
		if m != nil {
			filtered = append(filtered, m)
		}
	}

	n := len(filtered)
	if n == 0 {
		return ChatMiddlewareFunc(func(ctx context.Context, chatCtx *ChatContext, next ChatHandler) error {
			return next(ctx, chatCtx)
		})
	}
	if n == 1 {
		return filtered[0]
	}

	return ChatMiddlewareFunc(func(ctx context.Context, chatCtx *ChatContext, next ChatHandler) error {
		// Build chain from the last middleware backwards
		chain := next
		for i := n - 1; i >= 0; i-- {
			mw := filtered[i]
			currentNext := chain
			chain = func(ctx context.Context, chatCtx *ChatContext) error {
				return mw.Process(ctx, chatCtx, currentNext)
			}
		}
		return chain(ctx, chatCtx)
	})
}
