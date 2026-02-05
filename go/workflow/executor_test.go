// Copyright (c) Microsoft. All rights reserved.

package workflow

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExecutorFunc(t *testing.T) {
	t.Run("returns correct ID", func(t *testing.T) {
		// Arrange
		executor := NewExecutorFunc("test-executor", func(ctx context.Context, wCtx *WorkflowContext) error {
			return nil
		})

		// Act
		id := executor.ID()

		// Assert
		assert.Equal(t, "test-executor", id)
	})

	t.Run("executes wrapped function", func(t *testing.T) {
		// Arrange
		executed := false
		executor := NewExecutorFunc("test-executor", func(ctx context.Context, wCtx *WorkflowContext) error {
			executed = true
			return nil
		})

		// Act
		err := executor.Execute(context.Background(), nil)

		// Assert
		require.NoError(t, err)
		assert.True(t, executed)
	})

	t.Run("returns error from wrapped function", func(t *testing.T) {
		// Arrange
		expectedErr := errors.New("test error")
		executor := NewExecutorFunc("test-executor", func(ctx context.Context, wCtx *WorkflowContext) error {
			return expectedErr
		})

		// Act
		err := executor.Execute(context.Background(), nil)

		// Assert
		assert.Equal(t, expectedErr, err)
	})

	t.Run("has access to workflow context", func(t *testing.T) {
		// Arrange
		var receivedCtx *WorkflowContext
		executor := NewExecutorFunc("test-executor", func(ctx context.Context, wCtx *WorkflowContext) error {
			receivedCtx = wCtx
			return nil
		})
		wCtx := newWorkflowContext(context.Background(), "exec-1", "run-1", 0, nil, nil)

		// Act
		err := executor.Execute(context.Background(), wCtx)

		// Assert
		require.NoError(t, err)
		assert.Equal(t, wCtx, receivedCtx)
	})
}

func TestExecutorBase(t *testing.T) {
	t.Run("returns correct ID", func(t *testing.T) {
		// Arrange
		base := NewExecutorBase("my-executor")

		// Act
		id := base.ID()

		// Assert
		assert.Equal(t, "my-executor", id)
	})

	t.Run("can be embedded in custom executor", func(t *testing.T) {
		// Arrange
		type CustomExecutor struct {
			ExecutorBase
			value int
		}
		custom := &CustomExecutor{
			ExecutorBase: NewExecutorBase("custom"),
			value:        42,
		}

		// Act & Assert
		assert.Equal(t, "custom", custom.ID())
		assert.Equal(t, 42, custom.value)
	})

	t.Run("has default options", func(t *testing.T) {
		// Arrange
		base := NewExecutorBase("test")

		// Act
		opts := base.Options()

		// Assert
		assert.True(t, opts.AutoSendResult, "AutoSendResult should default to true")
		assert.True(t, opts.AutoYieldResult, "AutoYieldResult should default to true")
	})

	t.Run("options can be customized", func(t *testing.T) {
		// Arrange
		base := NewExecutorBase("test",
			WithAutoSend(false),
			WithAutoYield(false),
		)

		// Act
		opts := base.Options()

		// Assert
		assert.False(t, opts.AutoSendResult, "AutoSendResult should be false")
		assert.False(t, opts.AutoYieldResult, "AutoYieldResult should be false")
	})
}

func TestDefaultExecutorOptions(t *testing.T) {
	// Act
	opts := DefaultExecutorOptions()

	// Assert
	assert.True(t, opts.AutoSendResult, "expected AutoSendResult to be true by default")
	assert.True(t, opts.AutoYieldResult, "expected AutoYieldResult to be true by default")
}

func TestExecutorBaseWithOptions(t *testing.T) {
	// Arrange & Act
	eb := NewExecutorBase("test",
		WithAutoSend(false),
		WithAutoYield(false),
	)

	// Assert
	assert.Equal(t, "test", eb.ID())
	opts := eb.Options()
	assert.False(t, opts.AutoSendResult, "expected AutoSendResult to be false")
	assert.False(t, opts.AutoYieldResult, "expected AutoYieldResult to be false")
}

func TestExecutorBaseDefaultOptions(t *testing.T) {
	// Arrange & Act
	eb := NewExecutorBase("test")
	opts := eb.Options()

	// Assert
	assert.True(t, opts.AutoSendResult, "expected AutoSendResult to be true by default")
	assert.True(t, opts.AutoYieldResult, "expected AutoYieldResult to be true by default")
}

// testExecutor implements Executor for testing
type testExecutor struct {
	ExecutorBase
	executeFunc func(ctx context.Context, wCtx *WorkflowContext) error
}

func newTestExecutor(id string, fn func(ctx context.Context, wCtx *WorkflowContext) error) *testExecutor {
	return &testExecutor{
		ExecutorBase: NewExecutorBase(id),
		executeFunc:  fn,
	}
}

func (te *testExecutor) Execute(ctx context.Context, wCtx *WorkflowContext) error {
	if te.executeFunc != nil {
		return te.executeFunc(ctx, wCtx)
	}
	return nil
}

func TestExecutorInterface(t *testing.T) {
	t.Run("custom executor implements interface", func(t *testing.T) {
		// Arrange
		var executor Executor = newTestExecutor("custom", nil)

		// Assert
		assert.Equal(t, "custom", executor.ID())
	})

	t.Run("ExecutorFunc implements interface", func(t *testing.T) {
		// Arrange
		var executor Executor = NewExecutorFunc("func-exec", func(ctx context.Context, wCtx *WorkflowContext) error {
			return nil
		})

		// Assert
		assert.Equal(t, "func-exec", executor.ID())
	})
}
