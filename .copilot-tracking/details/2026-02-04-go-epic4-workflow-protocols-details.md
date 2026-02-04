<!-- markdownlint-disable-file -->
# Implementation Details: Go Epic 4 - Workflow Orchestration and Protocols

## Context Reference

Sources:

* [.copilot-tracking/research/2026-02-04-go-epic4-workflow-protocols-research.md](../research/2026-02-04-go-epic4-workflow-protocols-research.md)
* [dotnet/src/Microsoft.Agents.AI.Workflows/](../../dotnet/src/Microsoft.Agents.AI.Workflows/) - .NET workflow reference
* [dotnet/src/Microsoft.Agents.AI.A2A/](../../dotnet/src/Microsoft.Agents.AI.A2A/) - .NET A2A reference
* [dotnet/src/Microsoft.Agents.AI.AGUI/](../../dotnet/src/Microsoft.Agents.AI.AGUI/) - .NET AG-UI reference

---

## Implementation Phase 1: Workflow Core Types and Interfaces

<!-- parallelizable: false -->

### Step 1.1: Create workflow package structure

Create the workflow package directory structure and documentation file.

Files:

* `go/workflow/doc.go` - Package documentation

Success criteria:

* Package compiles with `go build`
* Documentation visible in godoc

```go
// Copyright (c) Microsoft. All rights reserved.

// Package workflow provides DAG-based workflow orchestration for AI agents.
//
// The workflow package implements a Pregel-like execution model where:
//   - Workflows are directed acyclic graphs (DAGs) of executors
//   - Executors process messages in synchronized supersteps
//   - Edges define message routing with optional conditions
//   - Checkpointing enables persistence and recovery
//
// Basic usage:
//
//	// Create a workflow with agents
//	wf, err := workflow.NewBuilder(agentA).
//	    AddExecutor(agentB).
//	    AddEdge(agentA.ID(), agentB.ID(), nil).
//	    Build()
//
//	// Run the workflow
//	result, err := wf.Run(ctx, input)
//
// The package also provides built-in executors for common patterns:
//   - AgentExecutor: Wraps an agent.Agent for workflow execution
//   - FunctionExecutor: Wraps a function for simple transformations
//   - AggregatingExecutor: Collects results from fan-out patterns
package workflow
```

---

### Step 1.2: Define WorkflowContext interface and struct

Create the context passed to executors during workflow execution.

Files:

* `go/workflow/context.go` - WorkflowContext definition

Success criteria:

* Context provides access to workflow state, messages, and output channel
* Supports cancellation via context.Context

Context references:

* Research document (Lines 142-170) - Pregel execution model description
* dotnet/src/Microsoft.Agents.AI.Workflows/IWorkflowContext.cs - .NET reference

```go
// Copyright (c) Microsoft. All rights reserved.

package workflow

import (
	"context"
	"sync"

	"github.com/microsoft/agent-framework-go/agent"
)

// WorkflowContext provides the execution context for an Executor.
// It is passed to each executor during workflow execution and provides
// access to workflow state, incoming messages, and output routing.
type WorkflowContext struct {
	// ctx is the parent context for cancellation and timeout
	ctx context.Context

	// executorID is the ID of the currently executing executor
	executorID string

	// runID is the unique identifier for this workflow run
	runID string

	// superstep is the current superstep number (0-based)
	superstep int

	// messages contains incoming messages for this executor
	messages []WorkflowMessage

	// state is the workflow-level shared state (thread-safe access required)
	state *sync.Map

	// outbox collects messages to send to other executors
	outbox []WorkflowMessage

	// mu protects outbox modifications
	mu sync.Mutex
}

// WorkflowMessage represents a message passed between executors.
type WorkflowMessage struct {
	// From is the executor ID that sent the message
	From string

	// To is the executor ID that should receive the message
	To string

	// Content contains the message payload
	Content agent.Message

	// Superstep is the superstep in which this message was created
	Superstep int
}

// Context returns the parent context for cancellation checking.
func (wc *WorkflowContext) Context() context.Context {
	return wc.ctx
}

// ExecutorID returns the ID of the currently executing executor.
func (wc *WorkflowContext) ExecutorID() string {
	return wc.executorID
}

// RunID returns the unique identifier for this workflow run.
func (wc *WorkflowContext) RunID() string {
	return wc.runID
}

// Superstep returns the current superstep number (0-based).
func (wc *WorkflowContext) Superstep() int {
	return wc.superstep
}

// Messages returns the incoming messages for this executor.
func (wc *WorkflowContext) Messages() []WorkflowMessage {
	return wc.messages
}

// GetState retrieves a value from the workflow-level shared state.
func (wc *WorkflowContext) GetState(key string) (interface{}, bool) {
	return wc.state.Load(key)
}

// SetState stores a value in the workflow-level shared state.
func (wc *WorkflowContext) SetState(key string, value interface{}) {
	wc.state.Store(key, value)
}

// Send queues a message to be sent to another executor.
// The message will be delivered in the next superstep.
func (wc *WorkflowContext) Send(to string, content agent.Message) {
	wc.mu.Lock()
	defer wc.mu.Unlock()
	wc.outbox = append(wc.outbox, WorkflowMessage{
		From:      wc.executorID,
		To:        to,
		Content:   content,
		Superstep: wc.superstep,
	})
}

// Outbox returns the messages queued for delivery.
// This is used internally by the workflow runner.
func (wc *WorkflowContext) Outbox() []WorkflowMessage {
	wc.mu.Lock()
	defer wc.mu.Unlock()
	result := make([]WorkflowMessage, len(wc.outbox))
	copy(result, wc.outbox)
	return result
}

// newWorkflowContext creates a new WorkflowContext for an executor.
func newWorkflowContext(
	ctx context.Context,
	executorID string,
	runID string,
	superstep int,
	messages []WorkflowMessage,
	state *sync.Map,
) *WorkflowContext {
	return &WorkflowContext{
		ctx:        ctx,
		executorID: executorID,
		runID:      runID,
		superstep:  superstep,
		messages:   messages,
		state:      state,
		outbox:     make([]WorkflowMessage, 0),
	}
}
```

---

### Step 1.3: Define Executor interface

Define the core Executor interface that all workflow nodes implement.

Files:

* `go/workflow/executor.go` - Executor interface definition

Success criteria:

* Interface is minimal and Go-idiomatic
* Supports identification and execution

Context references:

* Research document (Lines 176-210) - Executor pattern comparison
* dotnet/src/Microsoft.Agents.AI.Workflows/Executor.cs - .NET abstract class

```go
// Copyright (c) Microsoft. All rights reserved.

package workflow

import (
	"context"
)

// Executor processes messages in a workflow node.
// Implementations define the behavior of workflow nodes, receiving
// messages from connected edges and producing output messages.
//
// The Execute method is called during each superstep where the
// executor has pending messages. Executors should:
//   - Process incoming messages from wCtx.Messages()
//   - Send output messages using wCtx.Send()
//   - Use wCtx.Context() for cancellation checking
//   - Store state using wCtx.GetState()/SetState() if needed
type Executor interface {
	// ID returns the unique identifier for this executor.
	// IDs must be unique within a workflow.
	ID() string

	// Execute processes incoming messages and produces output.
	// The context provides access to messages, state, and output routing.
	// Returns an error if processing fails.
	Execute(ctx context.Context, wCtx *WorkflowContext) error
}

// ExecutorFunc is a function type that implements Executor.
// Use this for simple executors that don't need complex state.
type ExecutorFunc struct {
	id string
	fn func(ctx context.Context, wCtx *WorkflowContext) error
}

// NewExecutorFunc creates an Executor from a function.
func NewExecutorFunc(id string, fn func(ctx context.Context, wCtx *WorkflowContext) error) *ExecutorFunc {
	return &ExecutorFunc{id: id, fn: fn}
}

// ID returns the executor identifier.
func (ef *ExecutorFunc) ID() string {
	return ef.id
}

// Execute invokes the wrapped function.
func (ef *ExecutorFunc) Execute(ctx context.Context, wCtx *WorkflowContext) error {
	return ef.fn(ctx, wCtx)
}

// ExecutorBase provides common functionality for executor implementations.
// Embed this in custom executor types for convenience.
type ExecutorBase struct {
	id string
}

// NewExecutorBase creates a new ExecutorBase with the given ID.
func NewExecutorBase(id string) ExecutorBase {
	return ExecutorBase{id: id}
}

// ID returns the executor identifier.
func (eb *ExecutorBase) ID() string {
	return eb.id
}
```

---

### Step 1.4: Define Edge types and conditions

Define edge types for connecting executors with optional conditions.

Files:

* `go/workflow/edge.go` - Edge types and conditions

Success criteria:

* Support direct, conditional, fan-out, fan-in, and switch edges
* Conditions are Go functions for flexibility

Context references:

* Research document (Lines 78-95) - Edge type comparison table
* dotnet/src/Microsoft.Agents.AI.Workflows/Edge.cs - .NET edge definition

```go
// Copyright (c) Microsoft. All rights reserved.

package workflow

// EdgeCondition is a function that determines if an edge should be traversed.
// The state parameter contains the workflow's current state.
// Return true to allow message routing through this edge.
type EdgeCondition func(state map[string]interface{}) bool

// Edge defines a connection between two executors in a workflow.
type Edge struct {
	// From is the source executor ID
	From string

	// To is the target executor ID
	To string

	// Condition optionally guards message routing.
	// If nil, messages always flow through this edge.
	Condition EdgeCondition
}

// NewEdge creates a direct edge between two executors.
func NewEdge(from, to string) Edge {
	return Edge{From: from, To: to}
}

// NewConditionalEdge creates an edge with a routing condition.
func NewConditionalEdge(from, to string, condition EdgeCondition) Edge {
	return Edge{From: from, To: to, Condition: condition}
}

// EdgeGroup represents a collection of related edges for fan-out/fan-in patterns.
type EdgeGroup struct {
	// Type indicates the edge group pattern
	Type EdgeGroupType

	// From is the source executor ID (for fan-out)
	From string

	// To is the target executor ID (for fan-in)
	To string

	// Targets are the destination executor IDs (for fan-out)
	Targets []string

	// Sources are the source executor IDs (for fan-in)
	Sources []string
}

// EdgeGroupType defines the type of edge group pattern.
type EdgeGroupType int

const (
	// EdgeGroupTypeFanOut routes from one executor to multiple targets
	EdgeGroupTypeFanOut EdgeGroupType = iota

	// EdgeGroupTypeFanIn collects from multiple sources to one target
	EdgeGroupTypeFanIn

	// EdgeGroupTypeSwitch routes based on case matching
	EdgeGroupTypeSwitch
)

// FanOutEdge creates an edge group that routes from one executor to multiple targets.
func FanOutEdge(from string, targets ...string) EdgeGroup {
	return EdgeGroup{
		Type:    EdgeGroupTypeFanOut,
		From:    from,
		Targets: targets,
	}
}

// FanInEdge creates an edge group that collects from multiple sources to one target.
func FanInEdge(sources []string, to string) EdgeGroup {
	return EdgeGroup{
		Type:    EdgeGroupTypeFanIn,
		To:      to,
		Sources: sources,
	}
}

// SwitchCase represents a case in a switch edge.
type SwitchCase struct {
	// CaseValue is the value to match
	CaseValue interface{}

	// Target is the executor ID to route to when matched
	Target string
}

// SwitchEdge represents a switch/case routing pattern.
type SwitchEdge struct {
	// From is the source executor ID
	From string

	// Selector extracts the value to match from state
	Selector func(state map[string]interface{}) interface{}

	// Cases define the routing rules
	Cases []SwitchCase

	// Default is the fallback target if no case matches
	Default string
}

// NewSwitchEdge creates a switch edge with the given selector and cases.
func NewSwitchEdge(from string, selector func(state map[string]interface{}) interface{}) *SwitchEdge {
	return &SwitchEdge{
		From:     from,
		Selector: selector,
		Cases:    make([]SwitchCase, 0),
	}
}

// Case adds a case to the switch edge.
func (se *SwitchEdge) Case(value interface{}, target string) *SwitchEdge {
	se.Cases = append(se.Cases, SwitchCase{CaseValue: value, Target: target})
	return se
}

// DefaultCase sets the default target for the switch edge.
func (se *SwitchEdge) DefaultCase(target string) *SwitchEdge {
	se.Default = target
	return se
}

// Evaluate determines the target executor based on the current state.
func (se *SwitchEdge) Evaluate(state map[string]interface{}) string {
	if se.Selector == nil {
		return se.Default
	}
	value := se.Selector(state)
	for _, c := range se.Cases {
		if c.CaseValue == value {
			return c.Target
		}
	}
	return se.Default
}
```

---

### Step 1.5: Define Workflow struct and initialization

Define the Workflow struct that holds the complete workflow graph.

Files:

* `go/workflow/workflow.go` - Workflow struct definition

Success criteria:

* Workflow is immutable after creation
* Contains all executors, edges, and configuration

Context references:

* Research document (Lines 95-140) - Workflow package structure
* dotnet/src/Microsoft.Agents.AI.Workflows/Workflow.cs - .NET Workflow class

```go
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
```

---

### Step 1.6: Write unit tests for core types

Create unit tests for the workflow core types.

Files:

* `go/workflow/context_test.go` - WorkflowContext tests
* `go/workflow/executor_test.go` - Executor tests
* `go/workflow/edge_test.go` - Edge tests

Success criteria:

* All types are tested for basic functionality
* Tests cover edge cases and error conditions

Dependencies:

* Testing package

---

### Step 1.7: Validate Phase 1 changes

Run validation commands:

```bash
cd go
go build ./workflow/...
go test ./workflow/... -v
```

---

## Implementation Phase 2: Workflow Builder

<!-- parallelizable: false -->

### Step 2.1: Implement WorkflowBuilder core structure

Create the fluent builder for constructing workflows.

Files:

* `go/workflow/builder.go` - WorkflowBuilder implementation

Success criteria:

* Builder supports fluent API
* Validates configuration on Build()

Context references:

* Research document (Lines 176-200) - Builder API comparison
* dotnet/src/Microsoft.Agents.AI.Workflows/WorkflowBuilder.cs - .NET builder

```go
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
```

---

### Step 2.2: Implement AddExecutor and edge methods

Add edge configuration methods to the builder.

Files:

* `go/workflow/builder.go` - Edge methods (continuation)

Success criteria:

* AddEdge creates direct edges
* AddConditionalEdge creates conditional edges

```go
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
```

---

### Step 2.3: Implement FanOut and FanIn edge helpers

Add fan-out and fan-in pattern support.

Files:

* `go/workflow/builder.go` - FanOut/FanIn methods

Success criteria:

* FanOut routes to multiple targets
* FanIn collects from multiple sources

```go
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
```

---

### Step 2.4: Implement Switch/Case edge support

Add switch/case routing pattern.

Files:

* `go/workflow/builder.go` - Switch methods

Success criteria:

* Switch edges route based on state values
* Default case handles unmatched values

```go
// AddSwitch adds a switch edge for conditional routing.
func (b *WorkflowBuilder) AddSwitch(sw *SwitchEdge) *WorkflowBuilder {
	b.switchEdges = append(b.switchEdges, sw)
	return b
}

// SwitchFrom creates a new switch edge builder starting from an executor.
func (b *WorkflowBuilder) SwitchFrom(from string, selector func(state map[string]interface{}) interface{}) *SwitchEdgeBuilder {
	return &SwitchEdgeBuilder{
		parent:   b,
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
```

---

### Step 2.5: Implement Build validation and Workflow creation

Add build method with validation.

Files:

* `go/workflow/builder.go` - Build method

Success criteria:

* Build validates all executors and edges
* Returns immutable Workflow on success

```go
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
```

---

### Step 2.6: Write unit tests for WorkflowBuilder

Create comprehensive unit tests for the builder.

Files:

* `go/workflow/builder_test.go` - WorkflowBuilder tests

Success criteria:

* Test fluent API patterns
* Test validation errors
* Test all edge types

---

### Step 2.7: Validate Phase 2 changes

Run validation commands:

```bash
cd go
go build ./workflow/...
go test ./workflow/... -v
```

---

## Implementation Phase 3: Workflow Execution Engine

<!-- parallelizable: false -->

### Step 3.1: Implement WorkflowRunner core

Create the workflow execution engine.

Files:

* `go/workflow/runner.go` - WorkflowRunner implementation

Success criteria:

* Runner executes workflows step by step
* Supports cancellation via context

Context references:

* Research document (Lines 142-170) - Pregel execution model
* dotnet/src/Microsoft.Agents.AI.Workflows/Execution/InProcessExecution.cs - .NET execution

```go
// Copyright (c) Microsoft. All rights reserved.

package workflow

import (
	"context"
	"fmt"
	"sync"

	"github.com/google/uuid"
)

// WorkflowResult contains the final result of a workflow execution.
type WorkflowResult struct {
	// RunID is the unique identifier for this run
	RunID string

	// Outputs contains messages from output executors
	Outputs []WorkflowMessage

	// FinalState is the workflow state at completion
	FinalState map[string]interface{}

	// SuperstepCount is the number of supersteps executed
	SuperstepCount int
}

// RunnerOption configures the WorkflowRunner.
type RunnerOption func(*runnerOptions)

type runnerOptions struct {
	maxSupersteps   int
	checkpointStore CheckpointStore
	runID           string
}

// WithMaxSupersteps sets the maximum number of supersteps.
func WithMaxSupersteps(max int) RunnerOption {
	return func(o *runnerOptions) {
		o.maxSupersteps = max
	}
}

// WithCheckpointStore sets the checkpoint store for persistence.
func WithCheckpointStore(store CheckpointStore) RunnerOption {
	return func(o *runnerOptions) {
		o.checkpointStore = store
	}
}

// WithRunID sets a specific run ID instead of generating one.
func WithRunID(runID string) RunnerOption {
	return func(o *runnerOptions) {
		o.runID = runID
	}
}

// WorkflowRunner executes workflows using a Pregel-like superstep model.
type WorkflowRunner struct {
	workflow *Workflow
	options  runnerOptions
}

// NewRunner creates a new WorkflowRunner for the given workflow.
func NewRunner(wf *Workflow, opts ...RunnerOption) *WorkflowRunner {
	options := runnerOptions{
		maxSupersteps: 100, // Default limit
	}
	for _, opt := range opts {
		opt(&options)
	}
	return &WorkflowRunner{
		workflow: wf,
		options:  options,
	}
}
```

---

### Step 3.2: Implement Pregel-like superstep execution

Implement the core superstep execution loop.

Files:

* `go/workflow/runner.go` - Superstep execution

Success criteria:

* Executors process messages in parallel within supersteps
* Messages are delivered in the next superstep

```go
// Run executes the workflow and returns the final result.
func (r *WorkflowRunner) Run(ctx context.Context, input string) (*WorkflowResult, error) {
	runID := r.options.runID
	if runID == "" {
		runID = uuid.New().String()
	}

	state := &sync.Map{}
	var messages []WorkflowMessage
	var outputs []WorkflowMessage

	// Create initial message to start executor
	messages = append(messages, WorkflowMessage{
		From:      "__input__",
		To:        r.workflow.startID,
		Content:   newInputMessage(input),
		Superstep: -1,
	})

	superstep := 0
	for superstep < r.options.maxSupersteps {
		// Check for cancellation
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		// Execute superstep
		newMessages, stepOutputs, err := r.executeSuperstep(ctx, runID, superstep, messages, state)
		if err != nil {
			return nil, err
		}

		outputs = append(outputs, stepOutputs...)

		// Check for convergence (no new messages)
		if len(newMessages) == 0 {
			break
		}

		messages = newMessages
		superstep++
	}

	// Convert state to map
	finalState := make(map[string]interface{})
	state.Range(func(key, value interface{}) bool {
		finalState[key.(string)] = value
		return true
	})

	return &WorkflowResult{
		RunID:          runID,
		Outputs:        outputs,
		FinalState:     finalState,
		SuperstepCount: superstep + 1,
	}, nil
}

// executeSuperstep executes one superstep of the workflow.
func (r *WorkflowRunner) executeSuperstep(
	ctx context.Context,
	runID string,
	superstep int,
	messages []WorkflowMessage,
	state *sync.Map,
) ([]WorkflowMessage, []WorkflowMessage, error) {
	// Group messages by target executor
	messagesByExecutor := make(map[string][]WorkflowMessage)
	for _, msg := range messages {
		messagesByExecutor[msg.To] = append(messagesByExecutor[msg.To], msg)
	}

	// Execute each executor that has messages
	var allNewMessages []WorkflowMessage
	var outputs []WorkflowMessage
	var mu sync.Mutex
	var wg sync.WaitGroup
	var execErr error

	for executorID, execMessages := range messagesByExecutor {
		executor, ok := r.workflow.GetExecutor(executorID)
		if !ok {
			return nil, nil, fmt.Errorf("executor %q not found", executorID)
		}

		wg.Add(1)
		go func(exec Executor, msgs []WorkflowMessage) {
			defer wg.Done()

			wCtx := newWorkflowContext(ctx, exec.ID(), runID, superstep, msgs, state)

			if err := exec.Execute(ctx, wCtx); err != nil {
				mu.Lock()
				if execErr == nil {
					execErr = fmt.Errorf("executor %q failed: %w", exec.ID(), err)
				}
				mu.Unlock()
				return
			}

			outbox := wCtx.Outbox()

			mu.Lock()
			// Route messages based on edges
			stateMap := syncMapToMap(state)
			for _, msg := range outbox {
				if msg.To != "" {
					// Explicit target
					allNewMessages = append(allNewMessages, msg)
				} else {
					// Route via edges
					targets := r.workflow.GetTargets(exec.ID(), stateMap)
					for _, target := range targets {
						allNewMessages = append(allNewMessages, WorkflowMessage{
							From:      msg.From,
							To:        target,
							Content:   msg.Content,
							Superstep: superstep,
						})
					}
				}

				// Collect outputs
				if r.workflow.IsOutputExecutor(exec.ID()) {
					outputs = append(outputs, msg)
				}
			}
			mu.Unlock()
		}(executor, execMessages)
	}

	wg.Wait()

	if execErr != nil {
		return nil, nil, execErr
	}

	return allNewMessages, outputs, nil
}

func syncMapToMap(m *sync.Map) map[string]interface{} {
	result := make(map[string]interface{})
	m.Range(func(key, value interface{}) bool {
		result[key.(string)] = value
		return true
	})
	return result
}
```

---

### Step 3.3: Implement message routing between executors

Message routing is implemented in Step 3.2 as part of the superstep execution.

---

### Step 3.4: Implement convergence detection

Convergence is detected when no new messages are produced (implemented in Step 3.2).

---

### Step 3.5: Implement WorkflowEvent streaming via channels

Add streaming execution support.

Files:

* `go/workflow/events.go` - WorkflowEvent types
* `go/workflow/runner.go` - RunStream method

Success criteria:

* Events stream via channel
* Caller can observe workflow progress

```go
// go/workflow/events.go
// Copyright (c) Microsoft. All rights reserved.

package workflow

import "time"

// WorkflowEventKind indicates the type of workflow event.
type WorkflowEventKind int

const (
	// EventKindStarted indicates workflow execution started
	EventKindStarted WorkflowEventKind = iota

	// EventKindSuperstepStarted indicates a superstep began
	EventKindSuperstepStarted

	// EventKindExecutorInvoked indicates an executor was invoked
	EventKindExecutorInvoked

	// EventKindExecutorCompleted indicates an executor completed
	EventKindExecutorCompleted

	// EventKindExecutorFailed indicates an executor failed
	EventKindExecutorFailed

	// EventKindSuperstepCompleted indicates a superstep completed
	EventKindSuperstepCompleted

	// EventKindOutput indicates output from an output executor
	EventKindOutput

	// EventKindCompleted indicates workflow execution completed
	EventKindCompleted

	// EventKindError indicates workflow execution failed
	EventKindError
)

// WorkflowEvent represents an event during workflow execution.
type WorkflowEvent struct {
	// Kind is the type of event
	Kind WorkflowEventKind

	// RunID is the workflow run identifier
	RunID string

	// Superstep is the current superstep number
	Superstep int

	// ExecutorID is the relevant executor ID (if applicable)
	ExecutorID string

	// Message is the relevant message (if applicable)
	Message *WorkflowMessage

	// Error is the error (for error events)
	Error error

	// Result is the final result (for completed events)
	Result *WorkflowResult

	// Timestamp is when the event occurred
	Timestamp time.Time
}
```

```go
// go/workflow/runner.go (RunStream method)

// RunStream executes the workflow and returns a channel of events.
func (r *WorkflowRunner) RunStream(ctx context.Context, input string) (<-chan WorkflowEvent, error) {
	events := make(chan WorkflowEvent, 100)

	go func() {
		defer close(events)

		runID := r.options.runID
		if runID == "" {
			runID = uuid.New().String()
		}

		events <- WorkflowEvent{
			Kind:      EventKindStarted,
			RunID:     runID,
			Timestamp: time.Now(),
		}

		state := &sync.Map{}
		var messages []WorkflowMessage
		var outputs []WorkflowMessage

		// Create initial message
		messages = append(messages, WorkflowMessage{
			From:      "__input__",
			To:        r.workflow.startID,
			Content:   newInputMessage(input),
			Superstep: -1,
		})

		superstep := 0
		for superstep < r.options.maxSupersteps {
			if err := ctx.Err(); err != nil {
				events <- WorkflowEvent{
					Kind:      EventKindError,
					RunID:     runID,
					Error:     err,
					Timestamp: time.Now(),
				}
				return
			}

			events <- WorkflowEvent{
				Kind:      EventKindSuperstepStarted,
				RunID:     runID,
				Superstep: superstep,
				Timestamp: time.Now(),
			}

			newMessages, stepOutputs, err := r.executeSuperstepWithEvents(ctx, runID, superstep, messages, state, events)
			if err != nil {
				events <- WorkflowEvent{
					Kind:      EventKindError,
					RunID:     runID,
					Error:     err,
					Timestamp: time.Now(),
				}
				return
			}

			outputs = append(outputs, stepOutputs...)

			events <- WorkflowEvent{
				Kind:      EventKindSuperstepCompleted,
				RunID:     runID,
				Superstep: superstep,
				Timestamp: time.Now(),
			}

			if len(newMessages) == 0 {
				break
			}

			messages = newMessages
			superstep++
		}

		finalState := make(map[string]interface{})
		state.Range(func(key, value interface{}) bool {
			finalState[key.(string)] = value
			return true
		})

		result := &WorkflowResult{
			RunID:          runID,
			Outputs:        outputs,
			FinalState:     finalState,
			SuperstepCount: superstep + 1,
		}

		events <- WorkflowEvent{
			Kind:      EventKindCompleted,
			RunID:     runID,
			Result:    result,
			Timestamp: time.Now(),
		}
	}()

	return events, nil
}
```

---

### Step 3.6: Write unit tests for WorkflowRunner

Create comprehensive tests for the runner.

Files:

* `go/workflow/runner_test.go` - WorkflowRunner tests

---

### Step 3.7: Validate Phase 3 changes

Run validation commands:

```bash
cd go
go build ./workflow/...
go test ./workflow/... -v -cover
```

---

## Implementation Phase 4: Checkpointing

<!-- parallelizable: false -->

### Step 4.1: Define CheckpointStore interface

Create the checkpoint storage interface.

Files:

* `go/workflow/checkpoint.go` - CheckpointStore interface

Success criteria:

* Interface supports save/load/delete operations
* Supports listing checkpoints by run ID

```go
// Copyright (c) Microsoft. All rights reserved.

package workflow

import (
	"context"
	"encoding/json"
	"time"
)

// Checkpoint represents a saved workflow state.
type Checkpoint struct {
	// ID is the unique checkpoint identifier
	ID string `json:"id"`

	// RunID is the workflow run identifier
	RunID string `json:"run_id"`

	// Superstep is the superstep number when saved
	Superstep int `json:"superstep"`

	// State is the serialized workflow state
	State json.RawMessage `json:"state"`

	// PendingMessages are messages waiting for delivery
	PendingMessages []WorkflowMessage `json:"pending_messages"`

	// CreatedAt is when the checkpoint was created
	CreatedAt time.Time `json:"created_at"`
}

// CheckpointStore defines the interface for checkpoint persistence.
type CheckpointStore interface {
	// Save persists a checkpoint.
	Save(ctx context.Context, checkpoint *Checkpoint) error

	// Load retrieves a checkpoint by ID.
	Load(ctx context.Context, checkpointID string) (*Checkpoint, error)

	// LoadLatest retrieves the most recent checkpoint for a run.
	LoadLatest(ctx context.Context, runID string) (*Checkpoint, error)

	// Delete removes a checkpoint.
	Delete(ctx context.Context, checkpointID string) error

	// List returns all checkpoints for a run.
	List(ctx context.Context, runID string) ([]*Checkpoint, error)
}
```

---

### Step 4.2: Implement InMemoryCheckpointStore

Create an in-memory implementation for testing and simple use cases.

Files:

* `go/workflow/checkpoint.go` - InMemoryCheckpointStore

Success criteria:

* Thread-safe implementation
* Supports all CheckpointStore operations

```go
// InMemoryCheckpointStore provides an in-memory checkpoint store.
// This is suitable for testing and single-process applications.
type InMemoryCheckpointStore struct {
	checkpoints map[string]*Checkpoint
	byRunID     map[string][]string // runID -> checkpoint IDs
	mu          sync.RWMutex
}

// NewInMemoryCheckpointStore creates a new in-memory checkpoint store.
func NewInMemoryCheckpointStore() *InMemoryCheckpointStore {
	return &InMemoryCheckpointStore{
		checkpoints: make(map[string]*Checkpoint),
		byRunID:     make(map[string][]string),
	}
}

// Save persists a checkpoint to memory.
func (s *InMemoryCheckpointStore) Save(ctx context.Context, checkpoint *Checkpoint) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.checkpoints[checkpoint.ID] = checkpoint
	s.byRunID[checkpoint.RunID] = append(s.byRunID[checkpoint.RunID], checkpoint.ID)
	return nil
}

// Load retrieves a checkpoint by ID.
func (s *InMemoryCheckpointStore) Load(ctx context.Context, checkpointID string) (*Checkpoint, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	cp, ok := s.checkpoints[checkpointID]
	if !ok {
		return nil, fmt.Errorf("checkpoint %q not found", checkpointID)
	}
	return cp, nil
}

// LoadLatest retrieves the most recent checkpoint for a run.
func (s *InMemoryCheckpointStore) LoadLatest(ctx context.Context, runID string) (*Checkpoint, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	ids, ok := s.byRunID[runID]
	if !ok || len(ids) == 0 {
		return nil, fmt.Errorf("no checkpoints found for run %q", runID)
	}

	// Return the last checkpoint
	return s.checkpoints[ids[len(ids)-1]], nil
}

// Delete removes a checkpoint.
func (s *InMemoryCheckpointStore) Delete(ctx context.Context, checkpointID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	cp, ok := s.checkpoints[checkpointID]
	if !ok {
		return nil
	}

	delete(s.checkpoints, checkpointID)

	// Remove from byRunID
	ids := s.byRunID[cp.RunID]
	for i, id := range ids {
		if id == checkpointID {
			s.byRunID[cp.RunID] = append(ids[:i], ids[i+1:]...)
			break
		}
	}

	return nil
}

// List returns all checkpoints for a run.
func (s *InMemoryCheckpointStore) List(ctx context.Context, runID string) ([]*Checkpoint, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	ids, ok := s.byRunID[runID]
	if !ok {
		return nil, nil
	}

	result := make([]*Checkpoint, 0, len(ids))
	for _, id := range ids {
		result = append(result, s.checkpoints[id])
	}
	return result, nil
}
```

---

### Step 4.3: Implement checkpoint save/restore in WorkflowRunner

Add checkpointing support to the runner.

Files:

* `go/workflow/runner.go` - Checkpoint methods

---

### Step 4.4: Write unit tests for checkpointing

Create tests for checkpoint functionality.

Files:

* `go/workflow/checkpoint_test.go` - Checkpoint tests

---

### Step 4.5: Validate Phase 4 changes

Run validation commands:

```bash
cd go
go build ./workflow/...
go test ./workflow/... -v -cover
```

---

## Implementation Phase 5: Built-in Executors

<!-- parallelizable: true -->

### Step 5.1: Implement AgentExecutor

Create an executor that wraps an agent.Agent.

Files:

* `go/workflow/executors/agent.go` - AgentExecutor

Success criteria:

* Wraps any agent.Agent implementation
* Converts workflow messages to agent messages

```go
// Copyright (c) Microsoft. All rights reserved.

package executors

import (
	"context"
	"fmt"

	"github.com/microsoft/agent-framework-go/agent"
	"github.com/microsoft/agent-framework-go/workflow"
)

// AgentExecutor wraps an agent.Agent as a workflow Executor.
type AgentExecutor struct {
	workflow.ExecutorBase
	agent agent.Agent
}

// NewAgentExecutor creates an AgentExecutor from an agent.
func NewAgentExecutor(id string, a agent.Agent) *AgentExecutor {
	return &AgentExecutor{
		ExecutorBase: workflow.NewExecutorBase(id),
		agent:        a,
	}
}

// Execute processes incoming messages through the wrapped agent.
func (ae *AgentExecutor) Execute(ctx context.Context, wCtx *workflow.WorkflowContext) error {
	messages := wCtx.Messages()
	if len(messages) == 0 {
		return nil
	}

	// Convert workflow messages to agent messages
	agentMessages := make([]agent.Message, 0, len(messages))
	for _, msg := range messages {
		agentMessages = append(agentMessages, msg.Content)
	}

	// Run the agent
	response, err := ae.agent.Run(ctx, agentMessages)
	if err != nil {
		return fmt.Errorf("agent execution failed: %w", err)
	}

	// Send output messages
	for _, msg := range response.Messages {
		wCtx.Send("", msg)
	}

	return nil
}

// Agent returns the underlying agent.
func (ae *AgentExecutor) Agent() agent.Agent {
	return ae.agent
}
```

---

### Step 5.2: Implement FunctionExecutor

Create an executor that wraps a simple function.

Files:

* `go/workflow/executors/function.go` - FunctionExecutor

Success criteria:

* Wraps any function matching the signature
* Simple transformation use cases

```go
// Copyright (c) Microsoft. All rights reserved.

package executors

import (
	"context"

	"github.com/microsoft/agent-framework-go/agent"
	"github.com/microsoft/agent-framework-go/workflow"
)

// FunctionHandler is the signature for function executor handlers.
type FunctionHandler func(ctx context.Context, messages []agent.Message) ([]agent.Message, error)

// FunctionExecutor wraps a function as a workflow Executor.
type FunctionExecutor struct {
	workflow.ExecutorBase
	handler FunctionHandler
}

// NewFunctionExecutor creates a FunctionExecutor from a handler function.
func NewFunctionExecutor(id string, handler FunctionHandler) *FunctionExecutor {
	return &FunctionExecutor{
		ExecutorBase: workflow.NewExecutorBase(id),
		handler:      handler,
	}
}

// Execute processes incoming messages through the wrapped function.
func (fe *FunctionExecutor) Execute(ctx context.Context, wCtx *workflow.WorkflowContext) error {
	messages := wCtx.Messages()
	if len(messages) == 0 {
		return nil
	}

	// Convert workflow messages to agent messages
	agentMessages := make([]agent.Message, 0, len(messages))
	for _, msg := range messages {
		agentMessages = append(agentMessages, msg.Content)
	}

	// Execute the handler
	outputs, err := fe.handler(ctx, agentMessages)
	if err != nil {
		return err
	}

	// Send output messages
	for _, msg := range outputs {
		wCtx.Send("", msg)
	}

	return nil
}
```

---

### Step 5.3: Implement AggregatingExecutor

Create an executor that collects messages from fan-in patterns.

Files:

* `go/workflow/executors/aggregating.go` - AggregatingExecutor

Success criteria:

* Collects messages from multiple sources
* Aggregation function is configurable

```go
// Copyright (c) Microsoft. All rights reserved.

package executors

import (
	"context"

	"github.com/microsoft/agent-framework-go/agent"
	"github.com/microsoft/agent-framework-go/workflow"
)

// AggregateFunc combines multiple messages into one or more output messages.
type AggregateFunc func(messages []agent.Message) ([]agent.Message, error)

// AggregatingExecutor collects messages from multiple sources and aggregates them.
type AggregatingExecutor struct {
	workflow.ExecutorBase
	expectedSources []string
	aggregator      AggregateFunc
	collected       map[string][]agent.Message
}

// NewAggregatingExecutor creates an AggregatingExecutor.
func NewAggregatingExecutor(id string, sources []string, aggregator AggregateFunc) *AggregatingExecutor {
	return &AggregatingExecutor{
		ExecutorBase:    workflow.NewExecutorBase(id),
		expectedSources: sources,
		aggregator:      aggregator,
		collected:       make(map[string][]agent.Message),
	}
}

// Execute collects messages and aggregates when all sources have sent.
func (ae *AggregatingExecutor) Execute(ctx context.Context, wCtx *workflow.WorkflowContext) error {
	// Collect messages by source
	for _, msg := range wCtx.Messages() {
		ae.collected[msg.From] = append(ae.collected[msg.From], msg.Content)
	}

	// Check if all expected sources have sent
	allReceived := true
	for _, source := range ae.expectedSources {
		if len(ae.collected[source]) == 0 {
			allReceived = false
			break
		}
	}

	if !allReceived {
		return nil
	}

	// Aggregate all messages
	var allMessages []agent.Message
	for _, source := range ae.expectedSources {
		allMessages = append(allMessages, ae.collected[source]...)
	}

	outputs, err := ae.aggregator(allMessages)
	if err != nil {
		return err
	}

	// Send aggregated output
	for _, msg := range outputs {
		wCtx.Send("", msg)
	}

	// Clear collected messages
	ae.collected = make(map[string][]agent.Message)

	return nil
}
```

---

### Step 5.4: Write unit tests for built-in executors

Create tests for all executor types.

Files:

* `go/workflow/executors/agent_test.go`
* `go/workflow/executors/function_test.go`
* `go/workflow/executors/aggregating_test.go`

---

### Step 5.5: Validate Phase 5 changes

Run validation commands:

```bash
cd go
go build ./workflow/...
go test ./workflow/... -v -cover
```

---

## Implementation Phase 6: A2A Protocol Types

<!-- parallelizable: true -->

### Step 6.1: Create protocol/a2a package structure

Create the A2A protocol package.

Files:

* `go/protocol/a2a/doc.go` - Package documentation

Success criteria:

* Package structure matches plan
* Documentation explains A2A protocol

```go
// Copyright (c) Microsoft. All rights reserved.

// Package a2a provides client and server implementations for the A2A
// (Agent-to-Agent) protocol, enabling communication between AI agents.
//
// The A2A protocol defines a standard way for agents to:
//   - Discover capabilities via AgentCard
//   - Create and manage Tasks for work items
//   - Exchange Messages with text and structured content
//   - Stream responses via Server-Sent Events
//
// Client usage:
//
//	client := a2a.NewClient("https://agent.example.com/a2a")
//	card, _ := client.GetAgentCard(ctx)
//	task, _ := client.CreateTask(ctx, &a2a.CreateTaskRequest{...})
//
// Server usage:
//
//	server := a2a.NewServer(myAgent, a2a.WithAgentCard(card))
//	http.Handle("/a2a/", server.Handler())
//
// See https://a2a-protocol.org/latest/ for the full protocol specification.
package a2a
```

---

### Step 6.2: Define AgentCard type

Create the AgentCard type for capability declaration.

Files:

* `go/protocol/a2a/types.go` - AgentCard definition

Success criteria:

* Matches A2A protocol specification
* JSON serialization works correctly

```go
// Copyright (c) Microsoft. All rights reserved.

package a2a

import "time"

// AgentCard describes an agent's capabilities for discovery.
type AgentCard struct {
	// Name is the human-readable name of the agent
	Name string `json:"name"`

	// Description explains the agent's purpose
	Description string `json:"description,omitempty"`

	// URL is the base URL for the agent's A2A endpoints
	URL string `json:"url"`

	// Provider is the organization providing the agent
	Provider *AgentProvider `json:"provider,omitempty"`

	// Version is the agent's version string
	Version string `json:"version,omitempty"`

	// Capabilities describes what the agent can do
	Capabilities *AgentCapabilities `json:"capabilities,omitempty"`

	// Authentication describes how to authenticate
	Authentication *AgentAuthentication `json:"authentication,omitempty"`

	// Skills lists specific capabilities
	Skills []AgentSkill `json:"skills,omitempty"`
}

// AgentProvider identifies who provides the agent.
type AgentProvider struct {
	Name string `json:"name"`
	URL  string `json:"url,omitempty"`
}

// AgentCapabilities describes supported features.
type AgentCapabilities struct {
	// Streaming indicates SSE support
	Streaming bool `json:"streaming,omitempty"`

	// PushNotifications indicates webhook support
	PushNotifications bool `json:"push_notifications,omitempty"`

	// StateTransitions indicates supported state changes
	StateTransitions []string `json:"state_transitions,omitempty"`
}

// AgentAuthentication describes authentication requirements.
type AgentAuthentication struct {
	// Schemes lists supported auth schemes
	Schemes []string `json:"schemes,omitempty"`
}

// AgentSkill describes a specific agent capability.
type AgentSkill struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description,omitempty"`
	Tags        []string `json:"tags,omitempty"`
}
```

---

### Step 6.3: Define Task and TaskState types

Create Task-related types.

Files:

* `go/protocol/a2a/types.go` - Task types

Success criteria:

* Task lifecycle states match specification
* Task contains all required fields

```go
// TaskState represents the current state of a task.
type TaskState string

const (
	TaskStatePending   TaskState = "pending"
	TaskStateRunning   TaskState = "running"
	TaskStateCompleted TaskState = "completed"
	TaskStateFailed    TaskState = "failed"
	TaskStateCancelled TaskState = "cancelled"
)

// Task represents a unit of work in the A2A protocol.
type Task struct {
	// ID is the unique task identifier
	ID string `json:"id"`

	// ContextID groups related tasks (conversation)
	ContextID string `json:"context_id,omitempty"`

	// State is the current task state
	State TaskState `json:"state"`

	// Messages contains the conversation history
	Messages []Message `json:"messages,omitempty"`

	// Artifacts contains generated outputs
	Artifacts []Artifact `json:"artifacts,omitempty"`

	// Metadata contains additional properties
	Metadata map[string]interface{} `json:"metadata,omitempty"`

	// CreatedAt is when the task was created
	CreatedAt time.Time `json:"created_at"`

	// UpdatedAt is when the task was last modified
	UpdatedAt time.Time `json:"updated_at"`

	// Error contains error details if failed
	Error *TaskError `json:"error,omitempty"`
}

// TaskError describes why a task failed.
type TaskError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// CreateTaskRequest contains parameters for creating a task.
type CreateTaskRequest struct {
	ContextID string                 `json:"context_id,omitempty"`
	Message   *Message               `json:"message"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}
```

---

### Step 6.4: Define Message and Part types

Create Message-related types.

Files:

* `go/protocol/a2a/types.go` - Message types

Success criteria:

* Message parts support text and data
* JSON marshaling handles polymorphism

```go
// Message represents a message in the A2A protocol.
type Message struct {
	// Role indicates who sent the message
	Role MessageRole `json:"role"`

	// Parts contains the message content
	Parts []Part `json:"parts"`

	// Metadata contains additional properties
	Metadata map[string]interface{} `json:"metadata,omitempty"`

	// CreatedAt is when the message was created
	CreatedAt time.Time `json:"created_at,omitempty"`
}

// MessageRole indicates the sender of a message.
type MessageRole string

const (
	RoleUser      MessageRole = "user"
	RoleAssistant MessageRole = "assistant"
	RoleSystem    MessageRole = "system"
)

// Part is a component of a message (text, data, file, etc.).
type Part struct {
	// Type indicates the part type
	Type PartType `json:"type"`

	// Text contains text content (for text parts)
	Text string `json:"text,omitempty"`

	// Data contains structured data (for data parts)
	Data interface{} `json:"data,omitempty"`

	// MimeType indicates the content type
	MimeType string `json:"mime_type,omitempty"`

	// URI points to external content
	URI string `json:"uri,omitempty"`
}

// PartType indicates the type of message part.
type PartType string

const (
	PartTypeText PartType = "text"
	PartTypeData PartType = "data"
	PartTypeFile PartType = "file"
)

// NewTextPart creates a text message part.
func NewTextPart(text string) Part {
	return Part{Type: PartTypeText, Text: text}
}

// NewDataPart creates a data message part.
func NewDataPart(data interface{}, mimeType string) Part {
	return Part{Type: PartTypeData, Data: data, MimeType: mimeType}
}
```

---

### Step 6.5: Define Artifact type

Create the Artifact type for generated outputs.

Files:

* `go/protocol/a2a/types.go` - Artifact type

```go
// Artifact represents an output generated by an agent.
type Artifact struct {
	// ID is the unique artifact identifier
	ID string `json:"id"`

	// Name is a human-readable name
	Name string `json:"name,omitempty"`

	// Description explains the artifact
	Description string `json:"description,omitempty"`

	// Parts contains the artifact content
	Parts []Part `json:"parts"`

	// Metadata contains additional properties
	Metadata map[string]interface{} `json:"metadata,omitempty"`

	// CreatedAt is when the artifact was created
	CreatedAt time.Time `json:"created_at"`
}
```

---

### Step 6.6: Implement JSON marshaling for A2A types

Ensure proper JSON serialization.

Files:

* `go/protocol/a2a/types.go` - JSON methods

---

### Step 6.7: Write unit tests for A2A types

Create tests for type serialization.

Files:

* `go/protocol/a2a/types_test.go` - Type tests

---

### Step 6.8: Validate Phase 6 changes

Run validation commands:

```bash
cd go
go build ./protocol/a2a/...
go test ./protocol/a2a/... -v
```

---

## Implementation Phase 7: A2A Client

<!-- parallelizable: false -->

### Step 7.1: Implement A2A Client core

Create the HTTP client for A2A protocol.

Files:

* `go/protocol/a2a/client.go` - Client implementation

Success criteria:

* Client handles HTTP communication
* Supports functional options for configuration

```go
// Copyright (c) Microsoft. All rights reserved.

package a2a

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client provides methods for interacting with A2A agents.
type Client struct {
	baseURL    string
	httpClient *http.Client
	headers    map[string]string
}

// ClientOption configures the A2A client.
type ClientOption func(*clientOptions)

type clientOptions struct {
	httpClient *http.Client
	headers    map[string]string
	timeout    time.Duration
}

// WithHTTPClient sets a custom HTTP client.
func WithHTTPClient(client *http.Client) ClientOption {
	return func(o *clientOptions) {
		o.httpClient = client
	}
}

// WithHeader adds a header to all requests.
func WithHeader(key, value string) ClientOption {
	return func(o *clientOptions) {
		if o.headers == nil {
			o.headers = make(map[string]string)
		}
		o.headers[key] = value
	}
}

// WithTimeout sets the request timeout.
func WithTimeout(timeout time.Duration) ClientOption {
	return func(o *clientOptions) {
		o.timeout = timeout
	}
}

// NewClient creates a new A2A client.
func NewClient(baseURL string, opts ...ClientOption) *Client {
	options := clientOptions{
		timeout: 30 * time.Second,
	}
	for _, opt := range opts {
		opt(&options)
	}

	httpClient := options.httpClient
	if httpClient == nil {
		httpClient = &http.Client{
			Timeout: options.timeout,
		}
	}

	return &Client{
		baseURL:    baseURL,
		httpClient: httpClient,
		headers:    options.headers,
	}
}

// doRequest performs an HTTP request with common handling.
func (c *Client) doRequest(ctx context.Context, method, path string, body interface{}) (*http.Response, error) {
	var bodyReader io.Reader
	if body != nil {
		bodyBytes, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(bodyBytes)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	for key, value := range c.headers {
		req.Header.Set(key, value)
	}

	return c.httpClient.Do(req)
}
```

---

### Step 7.2: Implement GetAgentCard method

Add capability discovery.

Files:

* `go/protocol/a2a/client.go` - GetAgentCard method

```go
// GetAgentCard retrieves the agent's capability card.
func (c *Client) GetAgentCard(ctx context.Context) (*AgentCard, error) {
	resp, err := c.doRequest(ctx, http.MethodGet, "/.well-known/agent.json", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status: %s", resp.Status)
	}

	var card AgentCard
	if err := json.NewDecoder(resp.Body).Decode(&card); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return &card, nil
}
```

---

### Step 7.3: Implement CreateTask and GetTask methods

Add task management.

Files:

* `go/protocol/a2a/client.go` - Task methods

```go
// CreateTask creates a new task with the agent.
func (c *Client) CreateTask(ctx context.Context, req *CreateTaskRequest) (*Task, error) {
	resp, err := c.doRequest(ctx, http.MethodPost, "/tasks", req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("unexpected status: %s", resp.Status)
	}

	var task Task
	if err := json.NewDecoder(resp.Body).Decode(&task); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return &task, nil
}

// GetTask retrieves a task by ID.
func (c *Client) GetTask(ctx context.Context, taskID string) (*Task, error) {
	resp, err := c.doRequest(ctx, http.MethodGet, "/tasks/"+taskID, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status: %s", resp.Status)
	}

	var task Task
	if err := json.NewDecoder(resp.Body).Decode(&task); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return &task, nil
}
```

---

### Step 7.4: Implement SendMessage method

Add message sending.

Files:

* `go/protocol/a2a/client.go` - SendMessage method

```go
// SendMessageRequest contains parameters for sending a message.
type SendMessageRequest struct {
	Message  *Message               `json:"message"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// SendMessage sends a message to a task.
func (c *Client) SendMessage(ctx context.Context, taskID string, msg *Message) (*Task, error) {
	req := &SendMessageRequest{Message: msg}
	resp, err := c.doRequest(ctx, http.MethodPost, "/tasks/"+taskID+"/messages", req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status: %s", resp.Status)
	}

	var task Task
	if err := json.NewDecoder(resp.Body).Decode(&task); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return &task, nil
}
```

---

### Step 7.5: Implement SendMessageStream with SSE parsing

Add streaming message support.

Files:

* `go/protocol/a2a/client.go` - SendMessageStream method

```go
// SendMessageStream sends a message and streams the response.
func (c *Client) SendMessageStream(ctx context.Context, taskID string, msg *Message) (<-chan StreamEvent, error) {
	req := &SendMessageRequest{Message: msg}
	bodyBytes, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		c.baseURL+"/tasks/"+taskID+"/messages/stream",
		bytes.NewReader(bodyBytes),
	)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "text/event-stream")

	for key, value := range c.headers {
		httpReq.Header.Set(key, value)
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("unexpected status: %s", resp.Status)
	}

	events := make(chan StreamEvent, 100)
	go func() {
		defer close(events)
		defer resp.Body.Close()
		c.parseSSE(ctx, resp.Body, events)
	}()

	return events, nil
}

// StreamEvent represents an event from a streaming response.
type StreamEvent struct {
	Type    string          `json:"type"`
	Data    json.RawMessage `json:"data"`
	Message *Message        `json:"message,omitempty"`
	Task    *Task           `json:"task,omitempty"`
	Error   error           `json:"-"`
}

// parseSSE parses Server-Sent Events from a reader.
func (c *Client) parseSSE(ctx context.Context, r io.Reader, events chan<- StreamEvent) {
	scanner := bufio.NewScanner(r)
	var eventType string
	var data []byte

	for scanner.Scan() {
		if ctx.Err() != nil {
			return
		}

		line := scanner.Text()

		if line == "" {
			// End of event
			if len(data) > 0 {
				var event StreamEvent
				event.Type = eventType
				event.Data = data
				events <- event
			}
			eventType = ""
			data = nil
			continue
		}

		if strings.HasPrefix(line, "event:") {
			eventType = strings.TrimSpace(strings.TrimPrefix(line, "event:"))
		} else if strings.HasPrefix(line, "data:") {
			data = []byte(strings.TrimPrefix(line, "data:"))
		}
	}
}
```

---

### Step 7.6: Implement CancelTask method

Add task cancellation.

Files:

* `go/protocol/a2a/client.go` - CancelTask method

```go
// CancelTask cancels a running task.
func (c *Client) CancelTask(ctx context.Context, taskID string) error {
	resp, err := c.doRequest(ctx, http.MethodDelete, "/tasks/"+taskID, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("unexpected status: %s", resp.Status)
	}

	return nil
}
```

---

### Step 7.7: Write unit tests for A2A Client

Create comprehensive client tests.

Files:

* `go/protocol/a2a/client_test.go` - Client tests

---

### Step 7.8: Validate Phase 7 changes

Run validation commands:

```bash
cd go
go build ./protocol/a2a/...
go test ./protocol/a2a/... -v -cover
```

---

## Implementation Phase 8: A2A Server

<!-- parallelizable: false -->

### Step 8.1: Implement A2A Server core

Create the HTTP server for hosting agents via A2A.

Files:

* `go/protocol/a2a/server.go` - Server implementation

Success criteria:

* Server wraps any agent.Agent
* Exposes standard A2A endpoints

```go
// Copyright (c) Microsoft. All rights reserved.

package a2a

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/microsoft/agent-framework-go/agent"
)

// Server exposes an agent via the A2A protocol.
type Server struct {
	agent    agent.Agent
	card     *AgentCard
	tasks    sync.Map // taskID -> *Task
	sessions sync.Map // contextID -> *Session
}

// ServerOption configures the A2A server.
type ServerOption func(*serverOptions)

type serverOptions struct {
	card *AgentCard
}

// WithAgentCard sets a custom agent card.
func WithAgentCard(card *AgentCard) ServerOption {
	return func(o *serverOptions) {
		o.card = card
	}
}

// NewServer creates a new A2A server for the given agent.
func NewServer(a agent.Agent, opts ...ServerOption) *Server {
	options := serverOptions{}
	for _, opt := range opts {
		opt(&options)
	}

	card := options.card
	if card == nil {
		card = &AgentCard{
			Name:        a.Name(),
			Description: a.Description(),
			Capabilities: &AgentCapabilities{
				Streaming: true,
			},
		}
	}

	return &Server{
		agent: a,
		card:  card,
	}
}

// Handler returns an http.Handler for the A2A endpoints.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /.well-known/agent.json", s.handleGetAgentCard)
	mux.HandleFunc("POST /tasks", s.handleCreateTask)
	mux.HandleFunc("GET /tasks/{taskID}", s.handleGetTask)
	mux.HandleFunc("POST /tasks/{taskID}/messages", s.handleSendMessage)
	mux.HandleFunc("POST /tasks/{taskID}/messages/stream", s.handleSendMessageStream)
	mux.HandleFunc("DELETE /tasks/{taskID}", s.handleCancelTask)
	return mux
}
```

---

### Step 8.2: Implement AgentCard endpoint handler

Add capability discovery endpoint.

Files:

* `go/protocol/a2a/server.go` - GetAgentCard handler

```go
func (s *Server) handleGetAgentCard(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(s.card)
}
```

---

### Step 8.3: Implement Task management endpoints

Add task creation and retrieval.

Files:

* `go/protocol/a2a/server.go` - Task handlers

```go
func (s *Server) handleCreateTask(w http.ResponseWriter, r *http.Request) {
	var req CreateTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	task := &Task{
		ID:        uuid.New().String(),
		ContextID: req.ContextID,
		State:     TaskStatePending,
		Messages:  []Message{*req.Message},
		Metadata:  req.Metadata,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if task.ContextID == "" {
		task.ContextID = uuid.New().String()
	}

	s.tasks.Store(task.ID, task)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(task)
}

func (s *Server) handleGetTask(w http.ResponseWriter, r *http.Request) {
	taskID := r.PathValue("taskID")
	task, ok := s.tasks.Load(taskID)
	if !ok {
		http.Error(w, "task not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(task)
}
```

---

### Step 8.4: Implement SendMessage endpoint with SSE streaming

Add message handling with streaming support.

Files:

* `go/protocol/a2a/server.go` - SendMessage handlers

```go
func (s *Server) handleSendMessage(w http.ResponseWriter, r *http.Request) {
	taskID := r.PathValue("taskID")
	taskVal, ok := s.tasks.Load(taskID)
	if !ok {
		http.Error(w, "task not found", http.StatusNotFound)
		return
	}
	task := taskVal.(*Task)

	var req SendMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Add message to task
	task.Messages = append(task.Messages, *req.Message)
	task.State = TaskStateRunning
	task.UpdatedAt = time.Now()

	// Convert to agent messages and run
	agentMessages := s.toAgentMessages(task.Messages)
	response, err := s.agent.Run(r.Context(), agentMessages)
	if err != nil {
		task.State = TaskStateFailed
		task.Error = &TaskError{Code: "execution_error", Message: err.Error()}
		s.tasks.Store(taskID, task)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Add response messages to task
	for _, msg := range response.Messages {
		task.Messages = append(task.Messages, s.toA2AMessage(msg))
	}
	task.State = TaskStateCompleted
	task.UpdatedAt = time.Now()
	s.tasks.Store(taskID, task)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(task)
}

func (s *Server) handleSendMessageStream(w http.ResponseWriter, r *http.Request) {
	taskID := r.PathValue("taskID")
	taskVal, ok := s.tasks.Load(taskID)
	if !ok {
		http.Error(w, "task not found", http.StatusNotFound)
		return
	}
	task := taskVal.(*Task)

	var req SendMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Set up SSE
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

	// Add message and run stream
	task.Messages = append(task.Messages, *req.Message)
	task.State = TaskStateRunning
	s.tasks.Store(taskID, task)

	agentMessages := s.toAgentMessages(task.Messages)
	updates, err := s.agent.RunStream(r.Context(), agentMessages)
	if err != nil {
		s.writeSSEError(w, flusher, err)
		return
	}

	for update := range updates {
		s.writeSSEUpdate(w, flusher, &update)
	}

	task.State = TaskStateCompleted
	task.UpdatedAt = time.Now()
	s.tasks.Store(taskID, task)
}
```

---

### Step 8.5: Implement session and task storage

Add helper methods for conversion and storage.

Files:

* `go/protocol/a2a/server.go` - Helper methods

```go
func (s *Server) handleCancelTask(w http.ResponseWriter, r *http.Request) {
	taskID := r.PathValue("taskID")
	taskVal, ok := s.tasks.Load(taskID)
	if !ok {
		http.Error(w, "task not found", http.StatusNotFound)
		return
	}

	task := taskVal.(*Task)
	task.State = TaskStateCancelled
	task.UpdatedAt = time.Now()
	s.tasks.Store(taskID, task)

	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) toAgentMessages(messages []Message) []agent.Message {
	result := make([]agent.Message, 0, len(messages))
	for _, msg := range messages {
		result = append(result, s.toAgentMessage(msg))
	}
	return result
}

func (s *Server) toAgentMessage(msg Message) agent.Message {
	// Convert A2A message to agent message
	var text string
	for _, part := range msg.Parts {
		if part.Type == PartTypeText {
			text += part.Text
		}
	}

	switch msg.Role {
	case RoleUser:
		return agent.NewUserMessage(text)
	case RoleAssistant:
		return agent.NewAssistantMessage(text)
	case RoleSystem:
		return agent.NewSystemMessage(text)
	default:
		return agent.NewUserMessage(text)
	}
}

func (s *Server) toA2AMessage(msg agent.Message) Message {
	return Message{
		Role:  MessageRole(msg.Role),
		Parts: []Part{NewTextPart(msg.Text())},
	}
}

func (s *Server) writeSSEUpdate(w http.ResponseWriter, f http.Flusher, update *agent.ResponseUpdate) {
	data, _ := json.Marshal(update)
	fmt.Fprintf(w, "event: update\ndata: %s\n\n", data)
	f.Flush()
}

func (s *Server) writeSSEError(w http.ResponseWriter, f http.Flusher, err error) {
	fmt.Fprintf(w, "event: error\ndata: {\"message\":%q}\n\n", err.Error())
	f.Flush()
}
```

---

### Step 8.6: Write unit tests for A2A Server

Create comprehensive server tests.

Files:

* `go/protocol/a2a/server_test.go` - Server tests

---

### Step 8.7: Validate Phase 8 changes

Run validation commands:

```bash
cd go
go build ./protocol/a2a/...
go test ./protocol/a2a/... -v -cover
```

---

## Implementation Phase 9: A2A Agent Wrapper

<!-- parallelizable: false -->

### Step 9.1: Implement A2AAgent that wraps Client as agent.Agent

Create the agent wrapper for remote A2A agents.

Files:

* `go/protocol/a2a/agent.go` - A2AAgent implementation

Success criteria:

* Implements agent.Agent interface
* Wraps A2A Client for remote calls

```go
// Copyright (c) Microsoft. All rights reserved.

package a2a

import (
	"context"
	"encoding/json"
	"reflect"
	"time"

	"github.com/google/uuid"
	"github.com/microsoft/agent-framework-go/agent"
)

// A2AAgent wraps a remote A2A agent as a local agent.Agent.
type A2AAgent struct {
	client      *Client
	id          string
	name        string
	description string
	card        *AgentCard
}

// NewA2AAgent creates an agent that calls a remote A2A agent.
func NewA2AAgent(client *Client, opts ...A2AAgentOption) *A2AAgent {
	a := &A2AAgent{
		client: client,
		id:     uuid.New().String(),
	}
	for _, opt := range opts {
		opt(a)
	}
	return a
}

// A2AAgentOption configures an A2AAgent.
type A2AAgentOption func(*A2AAgent)

// WithA2AAgentID sets the agent ID.
func WithA2AAgentID(id string) A2AAgentOption {
	return func(a *A2AAgent) {
		a.id = id
	}
}

// WithA2AAgentName sets the agent name.
func WithA2AAgentName(name string) A2AAgentOption {
	return func(a *A2AAgent) {
		a.name = name
	}
}

// WithA2AAgentDescription sets the agent description.
func WithA2AAgentDescription(description string) A2AAgentOption {
	return func(a *A2AAgent) {
		a.description = description
	}
}

// ID returns the agent identifier.
func (a *A2AAgent) ID() string {
	return a.id
}

// Name returns the agent name.
func (a *A2AAgent) Name() string {
	if a.name != "" {
		return a.name
	}
	if a.card != nil {
		return a.card.Name
	}
	return ""
}

// Description returns the agent description.
func (a *A2AAgent) Description() string {
	if a.description != "" {
		return a.description
	}
	if a.card != nil {
		return a.card.Description
	}
	return ""
}

// Metadata returns provider-specific metadata.
func (a *A2AAgent) Metadata() agent.AIAgentMetadata {
	return agent.AIAgentMetadata{
		ProviderID: "a2a",
	}
}
```

---

### Step 9.2: Implement Run method using SendMessage

Add the non-streaming execution method.

Files:

* `go/protocol/a2a/agent.go` - Run method

```go
// Run executes the agent with the provided messages.
func (a *A2AAgent) Run(ctx context.Context, messages []agent.Message, opts ...agent.RunOption) (*agent.Response, error) {
	// Create task with first message or continue existing task
	var task *Task
	var err error

	// Convert messages to A2A format
	a2aMessages := make([]Message, 0, len(messages))
	for _, msg := range messages {
		a2aMessages = append(a2aMessages, a.toA2AMessage(msg))
	}

	// Create task with the conversation
	if len(a2aMessages) > 0 {
		lastMsg := &a2aMessages[len(a2aMessages)-1]
		task, err = a.client.CreateTask(ctx, &CreateTaskRequest{
			Message: lastMsg,
		})
		if err != nil {
			return nil, err
		}
	}

	// Wait for completion (simple polling for now)
	for task.State == TaskStatePending || task.State == TaskStateRunning {
		time.Sleep(100 * time.Millisecond)
		task, err = a.client.GetTask(ctx, task.ID)
		if err != nil {
			return nil, err
		}
	}

	if task.State == TaskStateFailed {
		return nil, fmt.Errorf("task failed: %s", task.Error.Message)
	}

	// Convert response messages
	responseMessages := make([]agent.Message, 0)
	for _, msg := range task.Messages {
		if msg.Role == RoleAssistant {
			responseMessages = append(responseMessages, a.fromA2AMessage(msg))
		}
	}

	return &agent.Response{
		ResponseID: task.ID,
		AgentID:    a.id,
		Messages:   responseMessages,
		CreatedAt:  time.Now(),
	}, nil
}

func (a *A2AAgent) toA2AMessage(msg agent.Message) Message {
	return Message{
		Role:  MessageRole(msg.Role),
		Parts: []Part{NewTextPart(msg.Text())},
	}
}

func (a *A2AAgent) fromA2AMessage(msg Message) agent.Message {
	var text string
	for _, part := range msg.Parts {
		if part.Type == PartTypeText {
			text += part.Text
		}
	}
	return agent.NewAssistantMessage(text)
}
```

---

### Step 9.3: Implement RunStream method using SendMessageStream

Add streaming execution method.

Files:

* `go/protocol/a2a/agent.go` - RunStream method

```go
// RunStream executes the agent and returns a channel of response updates.
func (a *A2AAgent) RunStream(ctx context.Context, messages []agent.Message, opts ...agent.RunOption) (<-chan agent.ResponseUpdate, error) {
	updates := make(chan agent.ResponseUpdate, 100)

	go func() {
		defer close(updates)

		// Create task
		a2aMessages := make([]Message, 0, len(messages))
		for _, msg := range messages {
			a2aMessages = append(a2aMessages, a.toA2AMessage(msg))
		}

		if len(a2aMessages) == 0 {
			return
		}

		lastMsg := &a2aMessages[len(a2aMessages)-1]
		task, err := a.client.CreateTask(ctx, &CreateTaskRequest{
			Message: lastMsg,
		})
		if err != nil {
			updates <- agent.ResponseUpdate{
				Kind:  agent.UpdateKindError,
				Error: err,
			}
			return
		}

		// Stream messages
		events, err := a.client.SendMessageStream(ctx, task.ID, lastMsg)
		if err != nil {
			updates <- agent.ResponseUpdate{
				Kind:  agent.UpdateKindError,
				Error: err,
			}
			return
		}

		for event := range events {
			if event.Error != nil {
				updates <- agent.ResponseUpdate{
					Kind:  agent.UpdateKindError,
					Error: event.Error,
				}
				return
			}

			// Convert A2A events to agent updates
			update := a.convertEvent(event)
			if update != nil {
				updates <- *update
			}
		}

		updates <- agent.ResponseUpdate{
			Kind:       agent.UpdateKindDone,
			ResponseID: task.ID,
		}
	}()

	return updates, nil
}

func (a *A2AAgent) convertEvent(event StreamEvent) *agent.ResponseUpdate {
	// Convert based on event type
	switch event.Type {
	case "message":
		var msg Message
		if err := json.Unmarshal(event.Data, &msg); err == nil {
			agentMsg := a.fromA2AMessage(msg)
			return &agent.ResponseUpdate{
				Kind:    agent.UpdateKindMessageComplete,
				Message: &agentMsg,
			}
		}
	case "text":
		var text string
		if err := json.Unmarshal(event.Data, &text); err == nil {
			return &agent.ResponseUpdate{
				Kind: agent.UpdateKindTextDelta,
				Text: text,
			}
		}
	}
	return nil
}
```

---

### Step 9.4: Implement session management

Add session support for the A2A agent.

Files:

* `go/protocol/a2a/agent.go` - Session methods
* `go/protocol/a2a/session.go` - A2ASession

```go
// NewSession creates a new session for the A2A agent.
func (a *A2AAgent) NewSession(ctx context.Context) (agent.Session, error) {
	return &A2ASession{
		id:        uuid.New().String(),
		contextID: uuid.New().String(),
	}, nil
}

// RestoreSession deserializes a previously saved session.
func (a *A2AAgent) RestoreSession(ctx context.Context, data json.RawMessage) (agent.Session, error) {
	var session A2ASession
	if err := json.Unmarshal(data, &session); err != nil {
		return nil, err
	}
	return &session, nil
}

// GetService retrieves a service from the agent.
func (a *A2AAgent) GetService(serviceType reflect.Type) interface{} {
	return nil
}
```

```go
// go/protocol/a2a/session.go
// Copyright (c) Microsoft. All rights reserved.

package a2a

import (
	"encoding/json"
	"time"

	"github.com/microsoft/agent-framework-go/agent"
)

// A2ASession maintains state for A2A agent conversations.
type A2ASession struct {
	id        string
	contextID string
	taskID    string
	messages  []agent.Message
	createdAt time.Time
}

// ID returns the session identifier.
func (s *A2ASession) ID() string {
	return s.id
}

// ContextID returns the A2A context identifier.
func (s *A2ASession) ContextID() string {
	return s.contextID
}

// Messages returns the conversation history.
func (s *A2ASession) Messages() []agent.Message {
	return s.messages
}

// AddMessage appends a message to the session.
func (s *A2ASession) AddMessage(msg agent.Message) {
	s.messages = append(s.messages, msg)
}

// Clear removes all messages from the session.
func (s *A2ASession) Clear() {
	s.messages = nil
}

// Serialize converts the session to JSON.
func (s *A2ASession) Serialize() (json.RawMessage, error) {
	return json.Marshal(s)
}
```

---

### Step 9.5: Write unit tests for A2AAgent

Create tests for the agent wrapper.

Files:

* `go/protocol/a2a/agent_test.go` - Agent tests

---

### Step 9.6: Validate Phase 9 changes

Run validation commands:

```bash
cd go
go build ./protocol/a2a/...
go test ./protocol/a2a/... -v -cover
```

---

## Implementation Phase 10-12: AG-UI Protocol

Phases 10-12 follow similar patterns to A2A but for AG-UI protocol. Key files:

* `go/protocol/agui/doc.go` - Package documentation
* `go/protocol/agui/types.go` - Event types (internal constants)
* `go/protocol/agui/events.go` - Event definitions
* `go/protocol/agui/converter.go` - ResponseUpdate to Event conversion
* `go/protocol/agui/server.go` - SSE streaming server

---

## Implementation Phase 13-15: Group Chat Orchestration

Phases 13-15 implement group chat functionality. Key files:

* `go/workflow/groupchat/manager.go` - GroupChatManager
* `go/workflow/groupchat/selector.go` - Selector interface and implementations
* `go/workflow/groupchat/transcript.go` - Transcript tracking
* `go/workflow/groupchat/options.go` - Functional options

Selector implementations:

* RoundRobinSelector - Cycles through agents in order
* RandomSelector - Picks a random agent
* LLMSelector - Uses an LLM to decide the next speaker

---

## Implementation Phase 16: Integration and Examples

Create example applications demonstrating each feature:

* `go/examples/workflow/` - DAG workflow example
* `go/examples/a2a-client/` - A2A client example
* `go/examples/a2a-server/` - A2A server example
* `go/examples/agui-server/` - AG-UI server example
* `go/examples/groupchat/` - Group chat example

Update README.md with Epic 4 feature documentation.

---

## Implementation Phase 17: Final Validation

<!-- parallelizable: false -->

### Step 17.1: Run full project validation

Execute all validation commands for the project:

```bash
cd go
go build ./...
go test ./... -v -cover
go vet ./...
golangci-lint run
```

### Step 17.2: Verify 90%+ test coverage

Generate coverage reports for new packages:

```bash
go test -coverprofile=coverage.out ./workflow/... ./protocol/...
go tool cover -html=coverage.out -o coverage.html
```

### Step 17.3: Fix minor validation issues

Iterate on lint errors, build warnings, and test failures. Apply fixes directly when corrections are straightforward and isolated.

### Step 17.4: Report blocking issues

When validation failures require changes beyond minor fixes:

* Document the issues and affected files
* Provide the user with next steps
* Recommend additional research and planning rather than inline fixes
* Avoid large-scale refactoring within this phase

---

## Dependencies

* Go 1.22+ with generics
* github.com/google/uuid - UUID generation
* Standard library: net/http, encoding/json, context, sync, bufio

## Success Criteria

* All workflow tests pass with 90%+ coverage
* A2A client/server tests pass with protocol compliance
* AG-UI server streams events correctly via SSE
* Group chat orchestrates multi-agent conversations
* All examples build and run successfully
* Documentation is complete and accurate
