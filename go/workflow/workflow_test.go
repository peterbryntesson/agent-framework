// Copyright (c) Microsoft. All rights reserved.

package workflow

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWorkflow_BasicProperties(t *testing.T) {
	t.Run("returns name and description", func(t *testing.T) {
		// Arrange
		wf := &Workflow{
			name:        "test-workflow",
			description: "A test workflow",
		}

		// Act & Assert
		assert.Equal(t, "test-workflow", wf.Name())
		assert.Equal(t, "A test workflow", wf.Description())
	})

	t.Run("returns start ID", func(t *testing.T) {
		// Arrange
		wf := &Workflow{
			startID: "start-executor",
		}

		// Act & Assert
		assert.Equal(t, "start-executor", wf.StartID())
	})
}

func TestWorkflow_Executors(t *testing.T) {
	t.Run("returns copy of executors", func(t *testing.T) {
		// Arrange
		exec1 := NewExecutorFunc("exec-1", nil)
		exec2 := NewExecutorFunc("exec-2", nil)
		wf := &Workflow{
			executors: map[string]Executor{
				"exec-1": exec1,
				"exec-2": exec2,
			},
		}

		// Act
		executors := wf.Executors()

		// Assert
		assert.Len(t, executors, 2)

		// Verify it's a copy
		delete(executors, "exec-1")
		assert.Len(t, wf.executors, 2)
	})

	t.Run("GetExecutor returns existing executor", func(t *testing.T) {
		// Arrange
		exec := NewExecutorFunc("exec-1", nil)
		wf := &Workflow{
			executors: map[string]Executor{
				"exec-1": exec,
			},
		}

		// Act
		found, ok := wf.GetExecutor("exec-1")

		// Assert
		require.True(t, ok)
		assert.Equal(t, exec, found)
	})

	t.Run("GetExecutor returns false for missing executor", func(t *testing.T) {
		// Arrange
		wf := &Workflow{
			executors: map[string]Executor{},
		}

		// Act
		_, ok := wf.GetExecutor("missing")

		// Assert
		assert.False(t, ok)
	})
}

func TestWorkflow_OutputExecutors(t *testing.T) {
	t.Run("identifies output executors", func(t *testing.T) {
		// Arrange
		wf := &Workflow{
			outputExecutors: map[string]bool{
				"output-1": true,
				"output-2": true,
			},
		}

		// Act & Assert
		assert.True(t, wf.IsOutputExecutor("output-1"))
		assert.True(t, wf.IsOutputExecutor("output-2"))
		assert.False(t, wf.IsOutputExecutor("not-output"))
	})
}

func TestWorkflow_GetEdgesFrom(t *testing.T) {
	t.Run("returns edges from executor", func(t *testing.T) {
		// Arrange
		wf := &Workflow{
			edges: []Edge{
				NewEdge("a", "b"),
				NewEdge("a", "c"),
				NewEdge("b", "d"),
			},
		}

		// Act
		edgesFromA := wf.GetEdgesFrom("a")
		edgesFromB := wf.GetEdgesFrom("b")

		// Assert
		assert.Len(t, edgesFromA, 2)
		assert.Len(t, edgesFromB, 1)
	})

	t.Run("returns empty for executor with no edges", func(t *testing.T) {
		// Arrange
		wf := &Workflow{
			edges: []Edge{
				NewEdge("a", "b"),
			},
		}

		// Act
		edgesFromC := wf.GetEdgesFrom("c")

		// Assert
		assert.Empty(t, edgesFromC)
	})
}

func TestWorkflow_GetTargets(t *testing.T) {
	t.Run("returns direct edge targets", func(t *testing.T) {
		// Arrange
		wf := &Workflow{
			edges: []Edge{
				NewEdge("a", "b"),
				NewEdge("a", "c"),
			},
		}

		// Act
		targets := wf.GetTargets("a", nil)

		// Assert
		assert.ElementsMatch(t, []string{"b", "c"}, targets)
	})

	t.Run("evaluates conditional edges", func(t *testing.T) {
		// Arrange
		wf := &Workflow{
			edges: []Edge{
				NewConditionalEdge("a", "b", func(state map[string]interface{}) bool {
					return state["go-b"] == true
				}),
				NewConditionalEdge("a", "c", func(state map[string]interface{}) bool {
					return state["go-c"] == true
				}),
			},
		}

		// Act & Assert
		targets1 := wf.GetTargets("a", map[string]interface{}{"go-b": true})
		assert.Equal(t, []string{"b"}, targets1)

		targets2 := wf.GetTargets("a", map[string]interface{}{"go-c": true})
		assert.Equal(t, []string{"c"}, targets2)

		targets3 := wf.GetTargets("a", map[string]interface{}{"go-b": true, "go-c": true})
		assert.ElementsMatch(t, []string{"b", "c"}, targets3)
	})

	t.Run("returns fan-out targets", func(t *testing.T) {
		// Arrange
		wf := &Workflow{
			edgeGroups: []EdgeGroup{
				FanOutEdge("a", "b", "c", "d"),
			},
		}

		// Act
		targets := wf.GetTargets("a", nil)

		// Assert
		assert.ElementsMatch(t, []string{"b", "c", "d"}, targets)
	})

	t.Run("evaluates switch edges", func(t *testing.T) {
		// Arrange
		sw := NewSwitchEdge("router", func(state map[string]interface{}) interface{} {
			return state["choice"]
		}).
			Case("x", "target-x").
			Case("y", "target-y").
			DefaultCase("default")

		wf := &Workflow{
			switchEdges: []*SwitchEdge{sw},
		}

		// Act & Assert
		assert.Equal(t, []string{"target-x"}, wf.GetTargets("router", map[string]interface{}{"choice": "x"}))
		assert.Equal(t, []string{"target-y"}, wf.GetTargets("router", map[string]interface{}{"choice": "y"}))
		assert.Equal(t, []string{"default"}, wf.GetTargets("router", map[string]interface{}{"choice": "z"}))
	})

	t.Run("combines all target types", func(t *testing.T) {
		// Arrange
		wf := &Workflow{
			edges: []Edge{
				NewEdge("a", "direct"),
			},
			edgeGroups: []EdgeGroup{
				FanOutEdge("a", "fan1", "fan2"),
			},
			switchEdges: []*SwitchEdge{
				NewSwitchEdge("a", func(state map[string]interface{}) interface{} {
					return "match"
				}).Case("match", "switch-target"),
			},
		}

		// Act
		targets := wf.GetTargets("a", nil)

		// Assert
		assert.ElementsMatch(t, []string{"direct", "fan1", "fan2", "switch-target"}, targets)
	})
}

func TestWorkflow_Validate(t *testing.T) {
	t.Run("fails without start ID", func(t *testing.T) {
		// Arrange
		wf := &Workflow{
			executors: map[string]Executor{
				"a": NewExecutorFunc("a", nil),
			},
		}

		// Act
		err := wf.Validate()

		// Assert
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no start executor")
	})

	t.Run("fails when start executor not found", func(t *testing.T) {
		// Arrange
		wf := &Workflow{
			startID:   "missing",
			executors: map[string]Executor{},
		}

		// Act
		err := wf.Validate()

		// Assert
		require.Error(t, err)
		assert.Contains(t, err.Error(), "start executor")
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("fails when edge source not found", func(t *testing.T) {
		// Arrange
		wf := &Workflow{
			startID: "a",
			executors: map[string]Executor{
				"a": NewExecutorFunc("a", nil),
				"b": NewExecutorFunc("b", nil),
			},
			edges: []Edge{
				NewEdge("missing", "b"),
			},
		}

		// Act
		err := wf.Validate()

		// Assert
		require.Error(t, err)
		assert.Contains(t, err.Error(), "edge source executor")
	})

	t.Run("fails when edge target not found", func(t *testing.T) {
		// Arrange
		wf := &Workflow{
			startID: "a",
			executors: map[string]Executor{
				"a": NewExecutorFunc("a", nil),
			},
			edges: []Edge{
				NewEdge("a", "missing"),
			},
		}

		// Act
		err := wf.Validate()

		// Assert
		require.Error(t, err)
		assert.Contains(t, err.Error(), "edge target executor")
	})

	t.Run("fails when fan-out source not found", func(t *testing.T) {
		// Arrange
		wf := &Workflow{
			startID: "a",
			executors: map[string]Executor{
				"a": NewExecutorFunc("a", nil),
				"b": NewExecutorFunc("b", nil),
			},
			edgeGroups: []EdgeGroup{
				FanOutEdge("missing", "b"),
			},
		}

		// Act
		err := wf.Validate()

		// Assert
		require.Error(t, err)
		assert.Contains(t, err.Error(), "fan-out source executor")
	})

	t.Run("fails when fan-out target not found", func(t *testing.T) {
		// Arrange
		wf := &Workflow{
			startID: "a",
			executors: map[string]Executor{
				"a": NewExecutorFunc("a", nil),
			},
			edgeGroups: []EdgeGroup{
				FanOutEdge("a", "missing"),
			},
		}

		// Act
		err := wf.Validate()

		// Assert
		require.Error(t, err)
		assert.Contains(t, err.Error(), "fan-out target executor")
	})

	t.Run("fails when fan-in source not found", func(t *testing.T) {
		// Arrange
		wf := &Workflow{
			startID: "a",
			executors: map[string]Executor{
				"a": NewExecutorFunc("a", nil),
				"b": NewExecutorFunc("b", nil),
			},
			edgeGroups: []EdgeGroup{
				FanInEdge([]string{"a", "missing"}, "b"),
			},
		}

		// Act
		err := wf.Validate()

		// Assert
		require.Error(t, err)
		assert.Contains(t, err.Error(), "fan-in source executor")
	})

	t.Run("fails when fan-in target not found", func(t *testing.T) {
		// Arrange
		wf := &Workflow{
			startID: "a",
			executors: map[string]Executor{
				"a": NewExecutorFunc("a", nil),
			},
			edgeGroups: []EdgeGroup{
				FanInEdge([]string{"a"}, "missing"),
			},
		}

		// Act
		err := wf.Validate()

		// Assert
		require.Error(t, err)
		assert.Contains(t, err.Error(), "fan-in target executor")
	})

	t.Run("fails when switch case target not found", func(t *testing.T) {
		// Arrange
		wf := &Workflow{
			startID: "router",
			executors: map[string]Executor{
				"router": NewExecutorFunc("router", nil),
			},
			switchEdges: []*SwitchEdge{
				NewSwitchEdge("router", nil).Case("x", "missing"),
			},
		}

		// Act
		err := wf.Validate()

		// Assert
		require.Error(t, err)
		assert.Contains(t, err.Error(), "switch case target executor")
	})

	t.Run("fails when switch default target not found", func(t *testing.T) {
		// Arrange
		wf := &Workflow{
			startID: "router",
			executors: map[string]Executor{
				"router": NewExecutorFunc("router", nil),
				"target": NewExecutorFunc("target", nil),
			},
			switchEdges: []*SwitchEdge{
				NewSwitchEdge("router", nil).Case("x", "target").DefaultCase("missing"),
			},
		}

		// Act
		err := wf.Validate()

		// Assert
		require.Error(t, err)
		assert.Contains(t, err.Error(), "switch default target executor")
	})

	t.Run("succeeds with valid workflow", func(t *testing.T) {
		// Arrange
		wf := &Workflow{
			startID: "a",
			executors: map[string]Executor{
				"a": NewExecutorFunc("a", nil),
				"b": NewExecutorFunc("b", nil),
				"c": NewExecutorFunc("c", nil),
			},
			edges: []Edge{
				NewEdge("a", "b"),
				NewEdge("b", "c"),
			},
		}

		// Act
		err := wf.Validate()

		// Assert
		require.NoError(t, err)
	})

	t.Run("succeeds with complex valid workflow", func(t *testing.T) {
		// Arrange
		wf := &Workflow{
			startID: "start",
			executors: map[string]Executor{
				"start":    NewExecutorFunc("start", nil),
				"worker1":  NewExecutorFunc("worker1", nil),
				"worker2":  NewExecutorFunc("worker2", nil),
				"worker3":  NewExecutorFunc("worker3", nil),
				"collect":  NewExecutorFunc("collect", nil),
				"router":   NewExecutorFunc("router", nil),
				"option-a": NewExecutorFunc("option-a", nil),
				"option-b": NewExecutorFunc("option-b", nil),
				"default":  NewExecutorFunc("default", nil),
			},
			edges: []Edge{
				NewEdge("start", "router"),
			},
			edgeGroups: []EdgeGroup{
				FanOutEdge("router", "worker1", "worker2", "worker3"),
				FanInEdge([]string{"worker1", "worker2", "worker3"}, "collect"),
			},
			switchEdges: []*SwitchEdge{
				NewSwitchEdge("collect", nil).
					Case("a", "option-a").
					Case("b", "option-b").
					DefaultCase("default"),
			},
		}

		// Act
		err := wf.Validate()

		// Assert
		require.NoError(t, err)
	})
}

func TestWorkflow_Run(t *testing.T) {
	t.Run("calls runner", func(t *testing.T) {
		// Arrange
		wf := &Workflow{
			startID: "a",
			executors: map[string]Executor{
				"a": NewExecutorFunc("a", nil),
			},
		}

		// Act - this will return error since runner is a stub
		_, err := wf.Run(context.Background(), "test input")

		// Assert - runner stub returns error
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not implemented")
	})
}
