// Copyright (c) Microsoft. All rights reserved.

package agent

import (
	"context"
	"sort"
	"strings"
	"sync"

	"github.com/microsoft/agent-framework-go/chat"
	"github.com/microsoft/agent-framework-go/tool"
)

// Context represents additional context to inject before an agent invocation.
// Instructions are appended to the agent's base instructions.
// Messages are prepended before user messages.
// Tools are merged with the agent's configured tools.
type Context struct {
	// Instructions are additional system instructions to append.
	// Multiple provider instructions are concatenated with newlines.
	Instructions string

	// Messages are additional messages to prepend before user messages.
	// These appear after system instructions and session history.
	Messages []chat.Message

	// Tools are additional tools to make available for this invocation.
	// These are merged with the agent's base tools and run-level tools.
	Tools []tool.Tool
}

// ContextProvider injects dynamic context before agent invocations.
// Implement this interface to add user-specific personalization,
// retrieved documents (RAG), dynamic tool availability, or other
// context that should vary per invocation.
//
// The Invoking method is called before each agent Run or RunStream call.
// Return a Context with additional instructions, messages, or tools
// to inject into the invocation.
//
// Example implementations:
//   - User profile provider that adds personalization instructions
//   - RAG provider that retrieves relevant documents as messages
//   - Feature flag provider that enables/disables tools dynamically
//   - Time provider that adds current datetime to context
type ContextProvider interface {
	// Invoking is called just before the agent invokes the chat client.
	// The messages parameter contains the conversation messages for this invocation.
	// Return a Context with additional instructions, messages, or tools to inject.
	// Return an empty Context (or nil) to inject nothing.
	// Return an error to abort the invocation.
	Invoking(ctx context.Context, messages []Message) (*Context, error)
}

// ContextProviderFunc is a function adapter for ContextProvider.
// Use this for simple providers that only need the Invoking method.
type ContextProviderFunc func(ctx context.Context, messages []Message) (*Context, error)

// Invoking implements ContextProvider.
func (f ContextProviderFunc) Invoking(ctx context.Context, messages []Message) (*Context, error) {
	return f(ctx, messages)
}

// ContextProviderWithLifecycle extends ContextProvider with lifecycle hooks.
// Implement this interface when your provider needs to:
//   - Track conversation history after invocations (Invoked)
//   - Initialize state when a new session is created (SessionCreated)
//
// These methods are optional. The base ContextProvider interface only
// requires Invoking. Use ContextProviderWithLifecycle when you need
// to update provider state based on agent responses.
type ContextProviderWithLifecycle interface {
	ContextProvider

	// Invoked is called after the agent receives a response from the chat client.
	// Use this to update provider state based on the conversation.
	//
	// Parameters:
	//   - request: The messages sent to the agent for this invocation
	//   - response: The messages returned by the agent (may be nil on error)
	//   - invokeErr: Any error that occurred during invocation (may be nil on success)
	//
	// The error returned by Invoked is logged but does not affect the agent response.
	// The agent will still return the successful response even if Invoked fails.
	Invoked(ctx context.Context, request []Message, response []Message, invokeErr error) error

	// SessionCreated is called when a new session is created or assigned an ID.
	// Use this to initialize session-specific state or register with external services.
	//
	// Parameters:
	//   - sessionID: The unique identifier for the session
	//
	// The error returned by SessionCreated is logged but does not prevent session creation.
	SessionCreated(ctx context.Context, sessionID string) error
}

// BaseContextProvider provides no-op implementations of lifecycle methods.
// Embed this in your provider to only override methods you need.
type BaseContextProvider struct{}

// Invoked is a no-op implementation.
func (BaseContextProvider) Invoked(ctx context.Context, request, response []Message, invokeErr error) error {
	return nil
}

// SessionCreated is a no-op implementation.
func (BaseContextProvider) SessionCreated(ctx context.Context, sessionID string) error {
	return nil
}

// AggregateContextProvider combines multiple ContextProviders.
// It invokes all providers concurrently and merges their results:
//   - Instructions are concatenated with newlines
//   - Messages are extended in provider order
//   - Tools are extended in provider order
//
// Use AggregateContextProvider when you need multiple sources of context,
// such as combining user personalization, RAG retrieval, and feature flags.
type AggregateContextProvider struct {
	providers []ContextProvider
}

// NewAggregateContextProvider creates an AggregateContextProvider
// from the given providers. Providers are invoked in the order given.
func NewAggregateContextProvider(providers ...ContextProvider) *AggregateContextProvider {
	return &AggregateContextProvider{
		providers: providers,
	}
}

// Add appends additional providers to the aggregate.
func (a *AggregateContextProvider) Add(providers ...ContextProvider) {
	a.providers = append(a.providers, providers...)
}

// Providers returns the list of providers in this aggregate.
func (a *AggregateContextProvider) Providers() []ContextProvider {
	return a.providers
}

// result holds the result from a single provider invocation.
type result struct {
	index   int
	context *Context
	err     error
}

// Invoking calls all providers concurrently and merges their contexts.
func (a *AggregateContextProvider) Invoking(ctx context.Context, messages []Message) (*Context, error) {
	if len(a.providers) == 0 {
		return &Context{}, nil
	}

	// For a single provider, call directly without goroutine overhead
	if len(a.providers) == 1 {
		return a.providers[0].Invoking(ctx, messages)
	}

	results := make(chan result, len(a.providers))
	var wg sync.WaitGroup

	for i, provider := range a.providers {
		wg.Add(1)
		go func(idx int, p ContextProvider) {
			defer wg.Done()
			c, err := p.Invoking(ctx, messages)
			results <- result{index: idx, context: c, err: err}
		}(i, provider)
	}

	// Wait for all goroutines then close channel
	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect results
	collected := make([]result, 0, len(a.providers))
	for r := range results {
		collected = append(collected, r)
	}

	// Sort by original index to maintain deterministic order
	sort.Slice(collected, func(i, j int) bool {
		return collected[i].index < collected[j].index
	})

	// Check for errors (first error wins)
	for _, r := range collected {
		if r.err != nil {
			return nil, r.err
		}
	}

	// Merge contexts
	return a.mergeContexts(collected), nil
}

// mergeContexts combines multiple contexts into one.
// Instructions are joined with newlines.
// Messages and Tools are concatenated in order.
func (a *AggregateContextProvider) mergeContexts(results []result) *Context {
	var instructions strings.Builder
	var messages []chat.Message
	var tools []tool.Tool

	for _, r := range results {
		if r.context == nil {
			continue
		}

		// Concatenate instructions with newlines
		if r.context.Instructions != "" {
			if instructions.Len() > 0 {
				instructions.WriteString("\n")
			}
			instructions.WriteString(r.context.Instructions)
		}

		// Extend messages
		if len(r.context.Messages) > 0 {
			messages = append(messages, r.context.Messages...)
		}

		// Extend tools
		if len(r.context.Tools) > 0 {
			tools = append(tools, r.context.Tools...)
		}
	}

	return &Context{
		Instructions: instructions.String(),
		Messages:     messages,
		Tools:        tools,
	}
}

// Invoked calls the Invoked lifecycle hook on all providers that implement it.
func (a *AggregateContextProvider) Invoked(ctx context.Context, request, response []Message, invokeErr error) error {
	for _, p := range a.providers {
		if lcp, ok := p.(ContextProviderWithLifecycle); ok {
			if err := lcp.Invoked(ctx, request, response, invokeErr); err != nil {
				return err
			}
		}
	}
	return nil
}

// SessionCreated calls the SessionCreated lifecycle hook on all providers that implement it.
func (a *AggregateContextProvider) SessionCreated(ctx context.Context, sessionID string) error {
	for _, p := range a.providers {
		if lcp, ok := p.(ContextProviderWithLifecycle); ok {
			if err := lcp.SessionCreated(ctx, sessionID); err != nil {
				return err
			}
		}
	}
	return nil
}
