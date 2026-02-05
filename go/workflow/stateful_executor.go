// Copyright (c) Microsoft. All rights reserved.

package workflow

import (
	"context"
	"encoding/json"
	"fmt"
)

// StatefulExecutor wraps an executor with typed persistent state.
// The state type T must be JSON-serializable for checkpoint persistence.
type StatefulExecutor[T any] struct {
	inner       Executor
	stateKey    string
	scopeName   string
	initFactory func() T
	stateCache  *T
	options     StatefulExecutorOptions
}

// StatefulExecutorOptions configures StatefulExecutor behavior.
type StatefulExecutorOptions struct {
	ExecutorOptions

	// StateKey is the key used for state storage.
	// Default: "{executorID}.State"
	StateKey string

	// ScopeName is an optional scope for state isolation.
	// Default: "" (private to executor)
	ScopeName string

	// CacheState when true, caches state in memory between invocations.
	// Disable for concurrent execution scenarios.
	// Default: true
	CacheState bool
}

// DefaultStatefulExecutorOptions returns default options.
func DefaultStatefulExecutorOptions() StatefulExecutorOptions {
	return StatefulExecutorOptions{
		ExecutorOptions: DefaultExecutorOptions(),
		CacheState:      true,
	}
}

// StatefulExecutorOption configures a StatefulExecutor.
type StatefulExecutorOption func(*StatefulExecutorOptions)

// WithStateKey sets a custom state key.
func WithStateKey(key string) StatefulExecutorOption {
	return func(o *StatefulExecutorOptions) {
		o.StateKey = key
	}
}

// WithScopeName sets a scope name for state isolation.
func WithScopeName(name string) StatefulExecutorOption {
	return func(o *StatefulExecutorOptions) {
		o.ScopeName = name
	}
}

// WithCacheState enables or disables state caching.
func WithCacheState(enabled bool) StatefulExecutorOption {
	return func(o *StatefulExecutorOptions) {
		o.CacheState = enabled
	}
}

// WithStatefulAutoSend sets auto-send behavior for the stateful executor.
func WithStatefulAutoSend(enabled bool) StatefulExecutorOption {
	return func(o *StatefulExecutorOptions) {
		o.AutoSendResult = enabled
	}
}

// WithStatefulAutoYield sets auto-yield behavior for the stateful executor.
func WithStatefulAutoYield(enabled bool) StatefulExecutorOption {
	return func(o *StatefulExecutorOptions) {
		o.AutoYieldResult = enabled
	}
}

// NewStatefulExecutor creates a new stateful executor wrapper.
func NewStatefulExecutor[T any](
	inner Executor,
	initFactory func() T,
	opts ...StatefulExecutorOption,
) *StatefulExecutor[T] {
	options := DefaultStatefulExecutorOptions()
	for _, opt := range opts {
		opt(&options)
	}

	// Default state key based on executor ID
	if options.StateKey == "" {
		options.StateKey = fmt.Sprintf("%s.State", inner.ID())
	}

	return &StatefulExecutor[T]{
		inner:       inner,
		stateKey:    options.StateKey,
		scopeName:   options.ScopeName,
		initFactory: initFactory,
		options:     options,
	}
}

// ID returns the wrapped executor's ID.
func (se *StatefulExecutor[T]) ID() string {
	return se.inner.ID()
}

// Options returns the executor options.
func (se *StatefulExecutor[T]) Options() ExecutorOptions {
	return se.options.ExecutorOptions
}

// stateStorageKey returns the full key used for state storage.
func (se *StatefulExecutor[T]) stateStorageKey() string {
	if se.scopeName != "" {
		return fmt.Sprintf("%s:%s", se.scopeName, se.stateKey)
	}
	return se.stateKey
}

// ReadState retrieves the current state from the workflow context.
// If no state exists, the init factory is called to create initial state.
func (se *StatefulExecutor[T]) ReadState(wCtx *WorkflowContext) (T, error) {
	// Check cache first
	if se.options.CacheState && se.stateCache != nil {
		return *se.stateCache, nil
	}

	key := se.stateStorageKey()
	rawVal, ok := wCtx.GetState(key)
	if !ok {
		// Initialize with factory
		initial := se.initFactory()
		if se.options.CacheState {
			se.stateCache = &initial
		}
		return initial, nil
	}

	// Deserialize from stored value
	var state T
	switch v := rawVal.(type) {
	case T:
		state = v
	case json.RawMessage:
		if err := json.Unmarshal(v, &state); err != nil {
			return state, fmt.Errorf("failed to unmarshal state: %w", err)
		}
	default:
		// Try JSON round-trip for compatibility
		data, err := json.Marshal(v)
		if err != nil {
			return state, fmt.Errorf("failed to convert state: %w", err)
		}
		if err := json.Unmarshal(data, &state); err != nil {
			return state, fmt.Errorf("failed to unmarshal state: %w", err)
		}
	}

	if se.options.CacheState {
		se.stateCache = &state
	}
	return state, nil
}

// QueueStateUpdate stores the state in the workflow context.
// The state will be persisted at the next checkpoint.
func (se *StatefulExecutor[T]) QueueStateUpdate(wCtx *WorkflowContext, state T) {
	key := se.stateStorageKey()
	wCtx.SetState(key, state)

	if se.options.CacheState {
		se.stateCache = &state
	}
}

// Execute runs the wrapped executor with state management.
// It reads state before execution and provides access via ReadState/QueueStateUpdate.
func (se *StatefulExecutor[T]) Execute(ctx context.Context, wCtx *WorkflowContext) error {
	// Pre-load state into cache if caching is enabled
	if se.options.CacheState {
		_, err := se.ReadState(wCtx)
		if err != nil {
			return fmt.Errorf("failed to read initial state: %w", err)
		}
	}

	// Execute the wrapped executor
	return se.inner.Execute(ctx, wCtx)
}

// InvokeWithState executes a function with the current state.
// This is a convenience method for executors that need direct state access.
func (se *StatefulExecutor[T]) InvokeWithState(
	wCtx *WorkflowContext,
	fn func(state T) (T, error),
) error {
	state, err := se.ReadState(wCtx)
	if err != nil {
		return err
	}

	newState, err := fn(state)
	if err != nil {
		return err
	}

	se.QueueStateUpdate(wCtx, newState)
	return nil
}

// ClearCache clears the cached state.
// Call this when the underlying state may have changed externally.
func (se *StatefulExecutor[T]) ClearCache() {
	se.stateCache = nil
}
