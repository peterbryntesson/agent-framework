// Copyright (c) Microsoft. All rights reserved.

package workflow

import (
	"context"
	"fmt"
)

// Workflow represents a complete workflow graph ready for execution.
// Workflows are immutable after creation via WorkflowBuilder.
type Workflow struct {
	// name is the optional workflow name
	name string

	// description is the optional workflow description
	description string

	// startID is the ID of the entry point executor
	startID string

	// executors maps executor IDs to their implementations
	executors map[string]Executor

	// edges contains all direct edges
	edges []Edge

	// edgeGroups contains fan-out/fan-in patterns
	edgeGroups []EdgeGroup

	// switchEdges contains switch/case routing
	switchEdges []*SwitchEdge

	// outputExecutors are executors that produce workflow output
	outputExecutors map[string]bool
}

// Name returns the workflow name.
func (w *Workflow) Name() string {
	return w.name
}

// Description returns the workflow description.
func (w *Workflow) Description() string {
	return w.description
}

// StartID returns the entry point executor ID.
func (w *Workflow) StartID() string {
	return w.startID
}

// Executors returns a copy of the executor map.
func (w *Workflow) Executors() map[string]Executor {
	result := make(map[string]Executor, len(w.executors))
	for k, v := range w.executors {
		result[k] = v
	}
	return result
}

// GetExecutor returns the executor with the given ID.
func (w *Workflow) GetExecutor(id string) (Executor, bool) {
	exec, ok := w.executors[id]
	return exec, ok
}

// IsOutputExecutor returns true if the executor produces workflow output.
func (w *Workflow) IsOutputExecutor(id string) bool {
	return w.outputExecutors[id]
}

// GetEdgesFrom returns all edges originating from the given executor.
func (w *Workflow) GetEdgesFrom(executorID string) []Edge {
	var result []Edge
	for _, edge := range w.edges {
		if edge.From == executorID {
			result = append(result, edge)
		}
	}
	return result
}

// GetTargets returns all target executor IDs for messages from the given executor.
// This evaluates conditions and switch cases based on the provided state.
func (w *Workflow) GetTargets(executorID string, state map[string]interface{}) []string {
	var targets []string

	// Check direct edges
	for _, edge := range w.edges {
		if edge.From == executorID {
			if edge.Condition == nil || edge.Condition(state) {
				targets = append(targets, edge.To)
			}
		}
	}

	// Check fan-out groups
	for _, group := range w.edgeGroups {
		if group.Type == EdgeGroupTypeFanOut && group.From == executorID {
			targets = append(targets, group.Targets...)
		}
	}

	// Check switch edges
	for _, sw := range w.switchEdges {
		if sw.From == executorID {
			target := sw.Evaluate(state)
			if target != "" {
				targets = append(targets, target)
			}
		}
	}

	return targets
}

// Validate checks that the workflow is correctly configured.
func (w *Workflow) Validate() error {
	if w.startID == "" {
		return fmt.Errorf("workflow has no start executor")
	}
	if _, ok := w.executors[w.startID]; !ok {
		return fmt.Errorf("start executor %q not found", w.startID)
	}

	// Validate all edge targets exist
	for _, edge := range w.edges {
		if _, ok := w.executors[edge.From]; !ok {
			return fmt.Errorf("edge source executor %q not found", edge.From)
		}
		if _, ok := w.executors[edge.To]; !ok {
			return fmt.Errorf("edge target executor %q not found", edge.To)
		}
	}

	// Validate fan-out/fan-in groups
	for _, group := range w.edgeGroups {
		switch group.Type {
		case EdgeGroupTypeFanOut:
			if _, ok := w.executors[group.From]; !ok {
				return fmt.Errorf("fan-out source executor %q not found", group.From)
			}
			for _, target := range group.Targets {
				if _, ok := w.executors[target]; !ok {
					return fmt.Errorf("fan-out target executor %q not found", target)
				}
			}
		case EdgeGroupTypeFanIn:
			if _, ok := w.executors[group.To]; !ok {
				return fmt.Errorf("fan-in target executor %q not found", group.To)
			}
			for _, source := range group.Sources {
				if _, ok := w.executors[source]; !ok {
					return fmt.Errorf("fan-in source executor %q not found", source)
				}
			}
		}
	}

	// Validate switch edges
	for _, sw := range w.switchEdges {
		if _, ok := w.executors[sw.From]; !ok {
			return fmt.Errorf("switch source executor %q not found", sw.From)
		}
		for _, c := range sw.Cases {
			if _, ok := w.executors[c.Target]; !ok {
				return fmt.Errorf("switch case target executor %q not found", c.Target)
			}
		}
		if sw.Default != "" {
			if _, ok := w.executors[sw.Default]; !ok {
				return fmt.Errorf("switch default target executor %q not found", sw.Default)
			}
		}
	}

	return nil
}

// Run executes the workflow with the given input.
// This is a convenience method that creates a runner and executes.
func (w *Workflow) Run(ctx context.Context, input string, opts ...RunnerOption) (*WorkflowResult, error) {
	runner := NewRunner(w, opts...)
	return runner.Run(ctx, input)
}

// RunStream executes the workflow and returns a channel of events.
func (w *Workflow) RunStream(ctx context.Context, input string, opts ...RunnerOption) (<-chan WorkflowEvent, error) {
	runner := NewRunner(w, opts...)
	return runner.RunStream(ctx, input)
}
