// Copyright (c) Microsoft. All rights reserved.

package agent

import (
	"context"
	"sync"
)

// RuntimeContext holds key-value pairs that propagate through agent delegation chains.
// This enables parent agents to forward context (user IDs, API tokens, session data)
// to sub-agents without modifying function signatures.
//
// RuntimeContext is immutable; With() returns a new context with additional values.
// Thread-safe for concurrent reads.
type RuntimeContext struct {
	values map[string]interface{}
	mu     sync.RWMutex
}

// NewRuntimeContext creates an empty RuntimeContext.
func NewRuntimeContext() *RuntimeContext {
	return &RuntimeContext{
		values: make(map[string]interface{}),
	}
}

// With returns a new RuntimeContext containing the original values plus the new key-value pair.
// The original context is not modified.
func (c *RuntimeContext) With(key string, value interface{}) *RuntimeContext {
	c.mu.RLock()
	defer c.mu.RUnlock()

	newValues := make(map[string]interface{}, len(c.values)+1)
	for k, v := range c.values {
		newValues[k] = v
	}
	newValues[key] = value

	return &RuntimeContext{values: newValues}
}

// Get retrieves a value by key. Returns nil and false if key is not present.
func (c *RuntimeContext) Get(key string) (interface{}, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	v, ok := c.values[key]
	return v, ok
}

// Values returns a copy of all key-value pairs in the context.
func (c *RuntimeContext) Values() map[string]interface{} {
	c.mu.RLock()
	defer c.mu.RUnlock()

	result := make(map[string]interface{}, len(c.values))
	for k, v := range c.values {
		result[k] = v
	}
	return result
}

// Len returns the number of values in the context.
func (c *RuntimeContext) Len() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.values)
}

// runtimeCtxKey is the context key for RuntimeContext values.
type runtimeCtxKey struct{}

// WithRuntimeCtx returns a new context.Context containing the RuntimeContext.
func WithRuntimeCtx(ctx context.Context, rtc *RuntimeContext) context.Context {
	return context.WithValue(ctx, runtimeCtxKey{}, rtc)
}

// RuntimeCtxFromContext extracts RuntimeContext from context.Context.
// Returns an empty RuntimeContext if not present.
func RuntimeCtxFromContext(ctx context.Context) *RuntimeContext {
	if rtc, ok := ctx.Value(runtimeCtxKey{}).(*RuntimeContext); ok && rtc != nil {
		return rtc
	}
	return NewRuntimeContext()
}
