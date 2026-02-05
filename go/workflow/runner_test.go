// Copyright (c) Microsoft. All rights reserved.

package workflow

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/microsoft/agent-framework-go/agent"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRunner(t *testing.T) {
	wf := createSimpleWorkflow(t)

	t.Run("creates runner with defaults", func(t *testing.T) {
		// Arrange & Act
		runner := NewRunner(wf)

		// Assert
		require.NotNil(t, runner)
		assert.Equal(t, wf, runner.workflow)
		assert.Equal(t, 100, runner.options.maxSupersteps)
		assert.Nil(t, runner.options.checkpointStore)
		assert.Empty(t, runner.options.runID)
	})

	t.Run("applies WithMaxSupersteps option", func(t *testing.T) {
		// Arrange & Act
		runner := NewRunner(wf, WithMaxSupersteps(50))

		// Assert
		assert.Equal(t, 50, runner.options.maxSupersteps)
	})

	t.Run("applies WithRunID option", func(t *testing.T) {
		// Arrange & Act
		runner := NewRunner(wf, WithRunID("test-run-123"))

		// Assert
		assert.Equal(t, "test-run-123", runner.options.runID)
	})

	t.Run("applies multiple options", func(t *testing.T) {
		// Arrange & Act
		runner := NewRunner(wf,
			WithMaxSupersteps(25),
			WithRunID("multi-opt-run"),
		)

		// Assert
		assert.Equal(t, 25, runner.options.maxSupersteps)
		assert.Equal(t, "multi-opt-run", runner.options.runID)
	})
}

func TestWorkflowRunner_Run(t *testing.T) {
	t.Run("executes simple linear workflow", func(t *testing.T) {
		// Arrange
		wf := createSimpleWorkflow(t)
		runner := NewRunner(wf, WithRunID("linear-test"))

		// Act
		result, err := runner.Run(context.Background(), "hello")

		// Assert
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, "linear-test", result.RunID)
		assert.Greater(t, result.SuperstepCount, 0)
	})

	t.Run("collects outputs from output executors", func(t *testing.T) {
		// Arrange
		wf := createWorkflowWithOutput(t)
		runner := NewRunner(wf)

		// Act
		result, err := runner.Run(context.Background(), "test input")

		// Assert
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Greater(t, len(result.Outputs), 0)
	})

	t.Run("handles executor errors", func(t *testing.T) {
		// Arrange
		wf := createWorkflowWithError(t)
		runner := NewRunner(wf)

		// Act
		result, err := runner.Run(context.Background(), "input")

		// Assert
		require.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "error-executor")
	})

	t.Run("respects context cancellation", func(t *testing.T) {
		// Arrange
		wf := createSlowWorkflow(t)
		runner := NewRunner(wf)
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		// Act
		result, err := runner.Run(ctx, "input")

		// Assert
		require.Error(t, err)
		assert.Nil(t, result)
		assert.ErrorIs(t, err, context.Canceled)
	})

	t.Run("converges when no new messages", func(t *testing.T) {
		// Arrange
		wf := createTerminatingWorkflow(t)
		runner := NewRunner(wf)

		// Act
		result, err := runner.Run(context.Background(), "start")

		// Assert
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.LessOrEqual(t, result.SuperstepCount, 5)
	})

	t.Run("respects max supersteps limit", func(t *testing.T) {
		// Arrange
		wf := createInfiniteWorkflow(t)
		runner := NewRunner(wf, WithMaxSupersteps(3))

		// Act
		result, err := runner.Run(context.Background(), "input")

		// Assert
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, 3, result.SuperstepCount)
	})

	t.Run("generates run ID when not specified", func(t *testing.T) {
		// Arrange
		wf := createSimpleWorkflow(t)
		runner := NewRunner(wf)

		// Act
		result, err := runner.Run(context.Background(), "input")

		// Assert
		require.NoError(t, err)
		assert.NotEmpty(t, result.RunID)
	})

	t.Run("preserves workflow state across supersteps", func(t *testing.T) {
		// Arrange
		wf := createStatefulWorkflow(t)
		runner := NewRunner(wf)

		// Act
		result, err := runner.Run(context.Background(), "initial")

		// Assert
		require.NoError(t, err)
		require.NotNil(t, result)
		val, ok := result.FinalState["counter"]
		assert.True(t, ok)
		assert.Greater(t, val.(int), 0)
	})

	t.Run("handles missing executor error", func(t *testing.T) {
		// Arrange: Create a workflow where edge points to non-existent executor
		// This shouldn't happen with proper validation, but test the error path
		exec := NewExecutorFunc("start", func(ctx context.Context, wCtx *WorkflowContext) error {
			wCtx.Send("nonexistent", agent.NewAssistantMessage("test"))
			return nil
		})

		wf := &Workflow{
			startID:         "start",
			executors:       map[string]Executor{"start": exec},
			outputExecutors: map[string]bool{},
		}
		runner := NewRunner(wf)

		// Act
		result, err := runner.Run(context.Background(), "test")

		// Assert
		require.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "not found")
	})
}

func TestWorkflowRunner_RunStream(t *testing.T) {
	t.Run("emits started event", func(t *testing.T) {
		// Arrange
		wf := createSimpleWorkflow(t)
		runner := NewRunner(wf, WithRunID("stream-test"))

		// Act
		events, err := runner.RunStream(context.Background(), "hello")

		// Assert
		require.NoError(t, err)
		require.NotNil(t, events)

		var foundStarted bool
		for event := range events {
			if event.Kind == EventKindStarted {
				foundStarted = true
				assert.Equal(t, "stream-test", event.RunID)
			}
		}
		assert.True(t, foundStarted, "should emit started event")
	})

	t.Run("emits superstep events", func(t *testing.T) {
		// Arrange
		wf := createSimpleWorkflow(t)
		runner := NewRunner(wf)

		// Act
		events, err := runner.RunStream(context.Background(), "test")

		// Assert
		require.NoError(t, err)

		var superstepStarted, superstepCompleted int
		for event := range events {
			if event.Kind == EventKindSuperstepStarted {
				superstepStarted++
			}
			if event.Kind == EventKindSuperstepCompleted {
				superstepCompleted++
			}
		}
		assert.Greater(t, superstepStarted, 0)
		assert.Equal(t, superstepStarted, superstepCompleted)
	})

	t.Run("emits executor events", func(t *testing.T) {
		// Arrange
		wf := createSimpleWorkflow(t)
		runner := NewRunner(wf)

		// Act
		events, err := runner.RunStream(context.Background(), "test")

		// Assert
		require.NoError(t, err)

		var invoked, completed int
		for event := range events {
			if event.Kind == EventKindExecutorInvoked {
				invoked++
				assert.NotEmpty(t, event.ExecutorID)
			}
			if event.Kind == EventKindExecutorCompleted {
				completed++
				assert.NotEmpty(t, event.ExecutorID)
			}
		}
		assert.Greater(t, invoked, 0)
		assert.Equal(t, invoked, completed)
	})

	t.Run("emits completed event with result", func(t *testing.T) {
		// Arrange
		wf := createSimpleWorkflow(t)
		runner := NewRunner(wf)

		// Act
		events, err := runner.RunStream(context.Background(), "test")

		// Assert
		require.NoError(t, err)

		var foundCompleted bool
		for event := range events {
			if event.Kind == EventKindCompleted {
				foundCompleted = true
				require.NotNil(t, event.Result)
				assert.NotEmpty(t, event.Result.RunID)
			}
		}
		assert.True(t, foundCompleted, "should emit completed event")
	})

	t.Run("emits error event on failure", func(t *testing.T) {
		// Arrange
		wf := createWorkflowWithError(t)
		runner := NewRunner(wf)

		// Act
		events, err := runner.RunStream(context.Background(), "test")

		// Assert
		require.NoError(t, err)

		var foundError bool
		for event := range events {
			if event.Kind == EventKindError {
				foundError = true
				assert.NotNil(t, event.Error)
			}
		}
		assert.True(t, foundError, "should emit error event")
	})

	t.Run("emits output events for output executors", func(t *testing.T) {
		// Arrange
		wf := createWorkflowWithOutput(t)
		runner := NewRunner(wf)

		// Act
		events, err := runner.RunStream(context.Background(), "test")

		// Assert
		require.NoError(t, err)

		var outputCount int
		for event := range events {
			if event.Kind == EventKindOutput {
				outputCount++
				assert.NotNil(t, event.Message)
			}
		}
		assert.Greater(t, outputCount, 0, "should emit output events")
	})

	t.Run("emits executor failed event on error", func(t *testing.T) {
		// Arrange
		wf := createWorkflowWithError(t)
		runner := NewRunner(wf)

		// Act
		events, err := runner.RunStream(context.Background(), "test")

		// Assert
		require.NoError(t, err)

		var foundFailed bool
		for event := range events {
			if event.Kind == EventKindExecutorFailed {
				foundFailed = true
				assert.NotEmpty(t, event.ExecutorID)
				assert.NotNil(t, event.Error)
			}
		}
		assert.True(t, foundFailed, "should emit executor failed event")
	})

	t.Run("handles context cancellation", func(t *testing.T) {
		// Arrange
		wf := createSlowWorkflow(t)
		runner := NewRunner(wf)
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		// Act
		events, err := runner.RunStream(ctx, "test")

		// Assert
		require.NoError(t, err) // Channel is returned before execution starts

		var foundError bool
		for event := range events {
			if event.Kind == EventKindError {
				foundError = true
				assert.ErrorIs(t, event.Error, context.Canceled)
			}
		}
		assert.True(t, foundError, "should emit error event on cancellation")
	})

	t.Run("closes channel after completion", func(t *testing.T) {
		// Arrange
		wf := createSimpleWorkflow(t)
		runner := NewRunner(wf)

		// Act
		events, err := runner.RunStream(context.Background(), "test")

		// Assert
		require.NoError(t, err)

		// Drain events
		for range events {
			// intentionally empty - drain streaming events
		}

		// Channel should be closed
		_, ok := <-events
		assert.False(t, ok, "channel should be closed")
	})

	t.Run("events have timestamps", func(t *testing.T) {
		// Arrange
		wf := createSimpleWorkflow(t)
		runner := NewRunner(wf)
		before := time.Now()

		// Act
		events, err := runner.RunStream(context.Background(), "test")

		// Assert
		require.NoError(t, err)

		for event := range events {
			assert.False(t, event.Timestamp.IsZero())
			assert.True(t, event.Timestamp.After(before) || event.Timestamp.Equal(before))
		}
	})
}

func TestWorkflowRunner_ParallelExecution(t *testing.T) {
	t.Run("executes parallel executors concurrently", func(t *testing.T) {
		// Arrange
		var executionOrder []string
		var mu sync.Mutex

		exec1 := NewExecutorFunc("executor1", func(ctx context.Context, wCtx *WorkflowContext) error {
			time.Sleep(10 * time.Millisecond)
			mu.Lock()
			executionOrder = append(executionOrder, "executor1")
			mu.Unlock()
			return nil
		})
		exec2 := NewExecutorFunc("executor2", func(ctx context.Context, wCtx *WorkflowContext) error {
			time.Sleep(10 * time.Millisecond)
			mu.Lock()
			executionOrder = append(executionOrder, "executor2")
			mu.Unlock()
			return nil
		})
		start := NewExecutorFunc("start", func(ctx context.Context, wCtx *WorkflowContext) error {
			wCtx.Send("executor1", agent.NewAssistantMessage("go"))
			wCtx.Send("executor2", agent.NewAssistantMessage("go"))
			return nil
		})

		builder := NewBuilder(start).
			WithName("parallel-test").
			AddExecutor(exec1).
			AddExecutor(exec2).
			AddEdge("start", "executor1").
			AddEdge("start", "executor2")

		wf, err := builder.Build()
		require.NoError(t, err)

		runner := NewRunner(wf)

		// Act
		startTime := time.Now()
		result, err := runner.Run(context.Background(), "input")
		elapsed := time.Since(startTime)

		// Assert
		require.NoError(t, err)
		require.NotNil(t, result)

		// Both executors should complete within roughly the same time
		// If sequential, it would take ~20ms; if parallel, ~10ms
		assert.Less(t, elapsed, 30*time.Millisecond)
		assert.Len(t, executionOrder, 2)
	})
}

func TestSyncMapToMap(t *testing.T) {
	t.Run("converts sync.Map to regular map", func(t *testing.T) {
		// Arrange
		sm := &sync.Map{}
		sm.Store("key1", "value1")
		sm.Store("key2", 42)
		sm.Store("key3", true)

		// Act
		result := syncMapToMap(sm)

		// Assert
		assert.Len(t, result, 3)
		assert.Equal(t, "value1", result["key1"])
		assert.Equal(t, 42, result["key2"])
		assert.Equal(t, true, result["key3"])
	})

	t.Run("returns empty map for empty sync.Map", func(t *testing.T) {
		// Arrange
		sm := &sync.Map{}

		// Act
		result := syncMapToMap(sm)

		// Assert
		assert.NotNil(t, result)
		assert.Len(t, result, 0)
	})
}

func TestNewInputMessage(t *testing.T) {
	t.Run("creates user message from input", func(t *testing.T) {
		// Arrange & Act
		msg := newInputMessage("test input")

		// Assert
		assert.NotNil(t, msg)
		assert.Equal(t, "test input", msg.Text())
	})
}

// Helper functions to create test workflows

func createSimpleWorkflow(t *testing.T) *Workflow {
	t.Helper()

	exec := NewExecutorFunc("start", func(ctx context.Context, wCtx *WorkflowContext) error {
		// Simply process and don't send further messages (converge)
		return nil
	})

	builder := NewBuilder(exec).WithName("simple")
	wf, err := builder.Build()
	require.NoError(t, err)
	return wf
}

func createWorkflowWithOutput(t *testing.T) *Workflow {
	t.Helper()

	start := NewExecutorFunc("start", func(ctx context.Context, wCtx *WorkflowContext) error {
		wCtx.Send("output", agent.NewAssistantMessage("processed"))
		return nil
	})
	output := NewExecutorFunc("output", func(ctx context.Context, wCtx *WorkflowContext) error {
		// Output executor just captures the message
		for _, msg := range wCtx.Messages() {
			wCtx.Send("", msg.Content) // Send to outbox for capture
		}
		return nil
	})

	builder := NewBuilder(start).
		WithName("with-output").
		AddExecutor(output).
		AddEdge("start", "output").
		MarkAsOutput("output")

	wf, err := builder.Build()
	require.NoError(t, err)
	return wf
}

func createWorkflowWithError(t *testing.T) *Workflow {
	t.Helper()

	start := NewExecutorFunc("start", func(ctx context.Context, wCtx *WorkflowContext) error {
		wCtx.Send("error-executor", agent.NewAssistantMessage("trigger"))
		return nil
	})
	errorExec := NewExecutorFunc("error-executor", func(ctx context.Context, wCtx *WorkflowContext) error {
		return errors.New("intentional error")
	})

	builder := NewBuilder(start).
		WithName("with-error").
		AddExecutor(errorExec).
		AddEdge("start", "error-executor")

	wf, err := builder.Build()
	require.NoError(t, err)
	return wf
}

func createSlowWorkflow(t *testing.T) *Workflow {
	t.Helper()

	exec := NewExecutorFunc("start", func(ctx context.Context, wCtx *WorkflowContext) error {
		select {
		case <-time.After(5 * time.Second):
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	})

	builder := NewBuilder(exec).WithName("slow")
	wf, err := builder.Build()
	require.NoError(t, err)
	return wf
}

func createTerminatingWorkflow(t *testing.T) *Workflow {
	t.Helper()

	start := NewExecutorFunc("start", func(ctx context.Context, wCtx *WorkflowContext) error {
		wCtx.Send("step2", agent.NewAssistantMessage("go"))
		return nil
	})
	step2 := NewExecutorFunc("step2", func(ctx context.Context, wCtx *WorkflowContext) error {
		wCtx.Send("end", agent.NewAssistantMessage("done"))
		return nil
	})
	end := NewExecutorFunc("end", func(ctx context.Context, wCtx *WorkflowContext) error {
		// Don't send any messages - workflow converges
		return nil
	})

	builder := NewBuilder(start).
		WithName("terminating").
		AddExecutor(step2).
		AddExecutor(end).
		AddEdge("start", "step2").
		AddEdge("step2", "end")

	wf, err := builder.Build()
	require.NoError(t, err)
	return wf
}

func createInfiniteWorkflow(t *testing.T) *Workflow {
	t.Helper()

	// Creates a cycle that never converges
	ping := NewExecutorFunc("ping", func(ctx context.Context, wCtx *WorkflowContext) error {
		wCtx.Send("pong", agent.NewAssistantMessage("ping"))
		return nil
	})
	pong := NewExecutorFunc("pong", func(ctx context.Context, wCtx *WorkflowContext) error {
		wCtx.Send("ping", agent.NewAssistantMessage("pong"))
		return nil
	})

	// Build manually to set ping as start
	wf := &Workflow{
		startID: "ping",
		executors: map[string]Executor{
			"ping": ping,
			"pong": pong,
		},
		outputExecutors: map[string]bool{},
	}
	return wf
}

func createStatefulWorkflow(t *testing.T) *Workflow {
	t.Helper()

	counter := NewExecutorFunc("counter", func(ctx context.Context, wCtx *WorkflowContext) error {
		val, ok := wCtx.GetState("counter")
		count := 0
		if ok {
			count = val.(int)
		}
		count++
		wCtx.SetState("counter", count)

		if count < 3 {
			wCtx.Send("counter", agent.NewAssistantMessage("continue"))
		}
		return nil
	})

	// Build manually to set counter as start with self-loop
	wf := &Workflow{
		startID: "counter",
		executors: map[string]Executor{
			"counter": counter,
		},
		outputExecutors: map[string]bool{},
	}
	return wf
}
