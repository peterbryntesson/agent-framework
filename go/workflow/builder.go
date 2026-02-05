// Copyright (c) Microsoft. All rights reserved.

package workflow

import (
	"fmt"
)

// WorkflowBuilder provides a fluent API for constructing workflows.
type WorkflowBuilder struct {
	name            string
	description     string
	startExecutor   Executor
	executors       map[string]Executor
	edges           []Edge
	edgeGroups      []EdgeGroup
	switchEdges     []*SwitchEdge
	outputExecutors map[string]bool
}

// NewBuilder creates a new WorkflowBuilder with the specified start executor.
func NewBuilder(start Executor) *WorkflowBuilder {
	if start == nil {
		panic("start executor cannot be nil")
	}
	return &WorkflowBuilder{
		startExecutor:   start,
		executors:       map[string]Executor{start.ID(): start},
		edges:           make([]Edge, 0),
		edgeGroups:      make([]EdgeGroup, 0),
		switchEdges:     make([]*SwitchEdge, 0),
		outputExecutors: make(map[string]bool),
	}
}

// WithName sets the workflow name.
func (b *WorkflowBuilder) WithName(name string) *WorkflowBuilder {
	b.name = name
	return b
}

// WithDescription sets the workflow description.
func (b *WorkflowBuilder) WithDescription(description string) *WorkflowBuilder {
	b.description = description
	return b
}

// AddExecutor adds an executor to the workflow.
func (b *WorkflowBuilder) AddExecutor(exec Executor) *WorkflowBuilder {
	if exec == nil {
		panic("executor cannot be nil")
	}
	if _, exists := b.executors[exec.ID()]; exists {
		panic(fmt.Sprintf("executor with ID %q already exists", exec.ID()))
	}
	b.executors[exec.ID()] = exec
	return b
}

// AddExecutors adds multiple executors to the workflow.
func (b *WorkflowBuilder) AddExecutors(executors ...Executor) *WorkflowBuilder {
	for _, exec := range executors {
		b.AddExecutor(exec)
	}
	return b
}

// AddEdge adds a direct edge between two executors.
func (b *WorkflowBuilder) AddEdge(from, to string) *WorkflowBuilder {
	b.edges = append(b.edges, NewEdge(from, to))
	return b
}

// AddConditionalEdge adds an edge with a condition.
func (b *WorkflowBuilder) AddConditionalEdge(from, to string, condition EdgeCondition) *WorkflowBuilder {
	b.edges = append(b.edges, NewConditionalEdge(from, to, condition))
	return b
}

// AddEdgeFromTo adds an edge using executor references.
func (b *WorkflowBuilder) AddEdgeFromTo(from, to Executor) *WorkflowBuilder {
	return b.AddEdge(from.ID(), to.ID())
}

// AddFanOut creates edges from one executor to multiple targets.
func (b *WorkflowBuilder) AddFanOut(from string, targets ...string) *WorkflowBuilder {
	b.edgeGroups = append(b.edgeGroups, FanOutEdge(from, targets...))
	return b
}

// AddFanIn creates an edge group that collects from multiple sources.
func (b *WorkflowBuilder) AddFanIn(sources []string, to string) *WorkflowBuilder {
	b.edgeGroups = append(b.edgeGroups, FanInEdge(sources, to))
	return b
}

// AddFanOutFromTo creates edges from one executor to multiple targets using references.
func (b *WorkflowBuilder) AddFanOutFromTo(from Executor, targets ...Executor) *WorkflowBuilder {
	targetIDs := make([]string, len(targets))
	for i, t := range targets {
		targetIDs[i] = t.ID()
	}
	return b.AddFanOut(from.ID(), targetIDs...)
}

// AddFanInFromTo creates an edge group using executor references.
func (b *WorkflowBuilder) AddFanInFromTo(sources []Executor, to Executor) *WorkflowBuilder {
	sourceIDs := make([]string, len(sources))
	for i, s := range sources {
		sourceIDs[i] = s.ID()
	}
	return b.AddFanIn(sourceIDs, to.ID())
}

// AddSwitch adds a switch edge for conditional routing.
func (b *WorkflowBuilder) AddSwitch(sw *SwitchEdge) *WorkflowBuilder {
	b.switchEdges = append(b.switchEdges, sw)
	return b
}

// SwitchFrom creates a new switch edge builder starting from an executor.
func (b *WorkflowBuilder) SwitchFrom(from string, selector func(state map[string]interface{}) interface{}) *SwitchEdgeBuilder {
	return &SwitchEdgeBuilder{
		parent:     b,
		switchEdge: NewSwitchEdge(from, selector),
	}
}

// SwitchEdgeBuilder provides a fluent API for building switch edges.
type SwitchEdgeBuilder struct {
	parent     *WorkflowBuilder
	switchEdge *SwitchEdge
}

// Case adds a case to the switch edge.
func (seb *SwitchEdgeBuilder) Case(value interface{}, target string) *SwitchEdgeBuilder {
	seb.switchEdge.Case(value, target)
	return seb
}

// Default sets the default target.
func (seb *SwitchEdgeBuilder) Default(target string) *SwitchEdgeBuilder {
	seb.switchEdge.DefaultCase(target)
	return seb
}

// End finishes the switch edge and returns to the workflow builder.
func (seb *SwitchEdgeBuilder) End() *WorkflowBuilder {
	seb.parent.AddSwitch(seb.switchEdge)
	return seb.parent
}

// MarkAsOutput marks an executor as producing workflow output.
func (b *WorkflowBuilder) MarkAsOutput(executorID string) *WorkflowBuilder {
	b.outputExecutors[executorID] = true
	return b
}

// Build creates the Workflow from the builder configuration.
// Returns an error if the workflow is not valid.
func (b *WorkflowBuilder) Build() (*Workflow, error) {
	wf := &Workflow{
		name:            b.name,
		description:     b.description,
		startID:         b.startExecutor.ID(),
		executors:       make(map[string]Executor, len(b.executors)),
		edges:           make([]Edge, len(b.edges)),
		edgeGroups:      make([]EdgeGroup, len(b.edgeGroups)),
		switchEdges:     make([]*SwitchEdge, len(b.switchEdges)),
		outputExecutors: make(map[string]bool, len(b.outputExecutors)),
	}

	// Copy executors
	for id, exec := range b.executors {
		wf.executors[id] = exec
	}

	// Copy edges
	copy(wf.edges, b.edges)
	copy(wf.edgeGroups, b.edgeGroups)
	copy(wf.switchEdges, b.switchEdges)

	// Copy output executors
	for id, isOutput := range b.outputExecutors {
		wf.outputExecutors[id] = isOutput
	}

	// If no output executors are marked, find leaf nodes
	if len(wf.outputExecutors) == 0 {
		leafNodes := b.findLeafNodes()
		for _, id := range leafNodes {
			wf.outputExecutors[id] = true
		}
	}

	// Validate the workflow
	if err := wf.Validate(); err != nil {
		return nil, err
	}

	return wf, nil
}

// findLeafNodes returns executor IDs that have no outgoing edges.
func (b *WorkflowBuilder) findLeafNodes() []string {
	hasOutgoing := make(map[string]bool)

	for _, edge := range b.edges {
		hasOutgoing[edge.From] = true
	}
	for _, group := range b.edgeGroups {
		if group.Type == EdgeGroupTypeFanOut {
			hasOutgoing[group.From] = true
		}
	}
	for _, sw := range b.switchEdges {
		hasOutgoing[sw.From] = true
	}

	var leaves []string
	for id := range b.executors {
		if !hasOutgoing[id] {
			leaves = append(leaves, id)
		}
	}
	return leaves
}
