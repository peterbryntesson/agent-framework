// Copyright (c) Microsoft. All rights reserved.

package workflow

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testState struct {
	Counter int    `json:"counter"`
	Name    string `json:"name"`
}

func TestStatefulExecutor_ReadState_Initial(t *testing.T) {
	// Arrange
	inner := NewExecutorFunc("test", func(ctx context.Context, wCtx *WorkflowContext) error {
		return nil
	})

	se := NewStatefulExecutor(inner, func() testState {
		return testState{Counter: 0, Name: "initial"}
	})

	wCtx := NewWorkflowContextForTest(
		context.Background(),
		"test",
		"run-1",
		0,
		nil,
		nil,
	)

	// Act
	state, err := se.ReadState(wCtx)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, 0, state.Counter)
	assert.Equal(t, "initial", state.Name)
}

func TestStatefulExecutor_QueueStateUpdate(t *testing.T) {
	// Arrange
	inner := NewExecutorFunc("test", func(ctx context.Context, wCtx *WorkflowContext) error {
		return nil
	})

	se := NewStatefulExecutor(inner, func() testState {
		return testState{}
	})

	wCtx := NewWorkflowContextForTest(
		context.Background(),
		"test",
		"run-1",
		0,
		nil,
		nil,
	)

	// Act
	se.QueueStateUpdate(wCtx, testState{Counter: 5, Name: "updated"})

	// Read back
	state, err := se.ReadState(wCtx)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, 5, state.Counter)
	assert.Equal(t, "updated", state.Name)
}

func TestStatefulExecutor_InvokeWithState(t *testing.T) {
	// Arrange
	inner := NewExecutorFunc("test", func(ctx context.Context, wCtx *WorkflowContext) error {
		return nil
	})

	se := NewStatefulExecutor(inner, func() testState {
		return testState{Counter: 10}
	})

	wCtx := NewWorkflowContextForTest(
		context.Background(),
		"test",
		"run-1",
		0,
		nil,
		nil,
	)

	// Act
	err := se.InvokeWithState(wCtx, func(s testState) (testState, error) {
		s.Counter++
		return s, nil
	})

	// Assert
	require.NoError(t, err)
	state, _ := se.ReadState(wCtx)
	assert.Equal(t, 11, state.Counter)
}

func TestStatefulExecutor_CustomStateKey(t *testing.T) {
	// Arrange
	inner := NewExecutorFunc("test", func(ctx context.Context, wCtx *WorkflowContext) error {
		return nil
	})

	se := NewStatefulExecutor(inner, func() testState {
		return testState{}
	}, WithStateKey("custom.key"))

	// Assert
	assert.Equal(t, "custom.key", se.stateStorageKey())
}

func TestStatefulExecutor_ScopedStateKey(t *testing.T) {
	// Arrange
	inner := NewExecutorFunc("test", func(ctx context.Context, wCtx *WorkflowContext) error {
		return nil
	})

	se := NewStatefulExecutor(inner, func() testState {
		return testState{}
	}, WithScopeName("shared"))

	// Assert
	expected := "shared:test.State"
	assert.Equal(t, expected, se.stateStorageKey())
}

func TestStatefulExecutor_DefaultStateKey(t *testing.T) {
	// Arrange
	inner := NewExecutorFunc("my-executor", func(ctx context.Context, wCtx *WorkflowContext) error {
		return nil
	})

	se := NewStatefulExecutor(inner, func() testState {
		return testState{}
	})

	// Assert
	assert.Equal(t, "my-executor.State", se.stateStorageKey())
}

func TestStatefulExecutor_CacheState(t *testing.T) {
	// Arrange
	inner := NewExecutorFunc("test", func(ctx context.Context, wCtx *WorkflowContext) error {
		return nil
	})

	se := NewStatefulExecutor(inner, func() testState {
		return testState{Counter: 100}
	})

	wCtx := NewWorkflowContextForTest(
		context.Background(),
		"test",
		"run-1",
		0,
		nil,
		nil,
	)

	// Act - First read loads initial state
	state1, err := se.ReadState(wCtx)
	require.NoError(t, err)
	assert.Equal(t, 100, state1.Counter)

	// Modify state in context directly (simulating external change)
	wCtx.SetState("test.State", testState{Counter: 200})

	// Act - Second read should return cached value
	state2, err := se.ReadState(wCtx)
	require.NoError(t, err)
	assert.Equal(t, 100, state2.Counter, "should return cached value")

	// Act - Clear cache and read again
	se.ClearCache()
	state3, err := se.ReadState(wCtx)
	require.NoError(t, err)
	assert.Equal(t, 200, state3.Counter, "should read from context after cache clear")
}

func TestStatefulExecutor_NoCacheState(t *testing.T) {
	// Arrange
	inner := NewExecutorFunc("test", func(ctx context.Context, wCtx *WorkflowContext) error {
		return nil
	})

	se := NewStatefulExecutor(inner, func() testState {
		return testState{Counter: 100}
	}, WithCacheState(false))

	wCtx := NewWorkflowContextForTest(
		context.Background(),
		"test",
		"run-1",
		0,
		nil,
		nil,
	)

	// Act - First read
	state1, err := se.ReadState(wCtx)
	require.NoError(t, err)

	// Update state via QueueStateUpdate
	se.QueueStateUpdate(wCtx, testState{Counter: 200})

	// Read again - should get the new value
	state2, err := se.ReadState(wCtx)
	require.NoError(t, err)

	// Assert
	assert.Equal(t, 100, state1.Counter)
	assert.Equal(t, 200, state2.Counter)
}

func TestStatefulExecutor_Execute(t *testing.T) {
	// Arrange
	executed := false
	inner := NewExecutorFunc("test", func(ctx context.Context, wCtx *WorkflowContext) error {
		executed = true
		return nil
	})

	se := NewStatefulExecutor(inner, func() testState {
		return testState{}
	})

	wCtx := NewWorkflowContextForTest(
		context.Background(),
		"test",
		"run-1",
		0,
		nil,
		nil,
	)

	// Act
	err := se.Execute(context.Background(), wCtx)

	// Assert
	require.NoError(t, err)
	assert.True(t, executed, "inner executor should be executed")
}

func TestStatefulExecutor_Options(t *testing.T) {
	// Arrange
	inner := NewExecutorFunc("test", func(ctx context.Context, wCtx *WorkflowContext) error {
		return nil
	})

	se := NewStatefulExecutor(inner, func() testState {
		return testState{}
	},
		WithStatefulAutoSend(false),
		WithStatefulAutoYield(false),
	)

	// Act
	opts := se.Options()

	// Assert
	assert.False(t, opts.AutoSendResult)
	assert.False(t, opts.AutoYieldResult)
}

func TestStatefulExecutor_ID(t *testing.T) {
	// Arrange
	inner := NewExecutorFunc("my-executor", func(ctx context.Context, wCtx *WorkflowContext) error {
		return nil
	})

	se := NewStatefulExecutor(inner, func() testState {
		return testState{}
	})

	// Assert
	assert.Equal(t, "my-executor", se.ID())
}
