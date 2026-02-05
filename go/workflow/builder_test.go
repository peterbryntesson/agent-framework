// Copyright (c) Microsoft. All rights reserved.

package workflow

import (
	"context"
	"testing"
)

// builderTestExecutor is a simple executor for builder tests.
type builderTestExecutor struct {
	id string
}

func newBuilderTestExecutor(id string) *builderTestExecutor {
	return &builderTestExecutor{id: id}
}

func (bte *builderTestExecutor) ID() string {
	return bte.id
}

func (bte *builderTestExecutor) Execute(ctx context.Context, wCtx *WorkflowContext) error {
	return nil
}

func TestNewBuilder_CreatesBuilderWithStartExecutor(t *testing.T) {
	// Arrange
	start := newBuilderTestExecutor("start")

	// Act
	builder := NewBuilder(start)

	// Assert
	if builder == nil {
		t.Fatal("expected builder to be non-nil")
	}
	if builder.startExecutor != start {
		t.Error("expected start executor to be set")
	}
	if len(builder.executors) != 1 {
		t.Errorf("expected 1 executor, got %d", len(builder.executors))
	}
	if _, exists := builder.executors["start"]; !exists {
		t.Error("expected start executor to be in executors map")
	}
}

func TestNewBuilder_PanicsOnNilStart(t *testing.T) {
	// Arrange
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for nil start executor")
		}
	}()

	// Act
	NewBuilder(nil)
}

func TestWorkflowBuilder_WithName(t *testing.T) {
	// Arrange
	start := newBuilderTestExecutor("start")
	builder := NewBuilder(start)

	// Act
	result := builder.WithName("test-workflow")

	// Assert
	if result != builder {
		t.Error("expected fluent return")
	}
	if builder.name != "test-workflow" {
		t.Errorf("expected name 'test-workflow', got %q", builder.name)
	}
}

func TestWorkflowBuilder_WithDescription(t *testing.T) {
	// Arrange
	start := newBuilderTestExecutor("start")
	builder := NewBuilder(start)

	// Act
	result := builder.WithDescription("A test workflow")

	// Assert
	if result != builder {
		t.Error("expected fluent return")
	}
	if builder.description != "A test workflow" {
		t.Errorf("expected description 'A test workflow', got %q", builder.description)
	}
}

func TestWorkflowBuilder_AddExecutor(t *testing.T) {
	// Arrange
	start := newBuilderTestExecutor("start")
	exec := newBuilderTestExecutor("exec")
	builder := NewBuilder(start)

	// Act
	result := builder.AddExecutor(exec)

	// Assert
	if result != builder {
		t.Error("expected fluent return")
	}
	if len(builder.executors) != 2 {
		t.Errorf("expected 2 executors, got %d", len(builder.executors))
	}
	if _, exists := builder.executors["exec"]; !exists {
		t.Error("expected added executor to be in executors map")
	}
}

func TestWorkflowBuilder_AddExecutor_PanicsOnNil(t *testing.T) {
	// Arrange
	start := newBuilderTestExecutor("start")
	builder := NewBuilder(start)
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for nil executor")
		}
	}()

	// Act
	builder.AddExecutor(nil)
}

func TestWorkflowBuilder_AddExecutor_PanicsOnDuplicate(t *testing.T) {
	// Arrange
	start := newBuilderTestExecutor("start")
	duplicate := newBuilderTestExecutor("start")
	builder := NewBuilder(start)
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for duplicate executor ID")
		}
	}()

	// Act
	builder.AddExecutor(duplicate)
}

func TestWorkflowBuilder_AddExecutors(t *testing.T) {
	// Arrange
	start := newBuilderTestExecutor("start")
	exec1 := newBuilderTestExecutor("exec1")
	exec2 := newBuilderTestExecutor("exec2")
	builder := NewBuilder(start)

	// Act
	result := builder.AddExecutors(exec1, exec2)

	// Assert
	if result != builder {
		t.Error("expected fluent return")
	}
	if len(builder.executors) != 3 {
		t.Errorf("expected 3 executors, got %d", len(builder.executors))
	}
}

func TestWorkflowBuilder_AddEdge(t *testing.T) {
	// Arrange
	start := newBuilderTestExecutor("start")
	end := newBuilderTestExecutor("end")
	builder := NewBuilder(start).AddExecutor(end)

	// Act
	result := builder.AddEdge("start", "end")

	// Assert
	if result != builder {
		t.Error("expected fluent return")
	}
	if len(builder.edges) != 1 {
		t.Errorf("expected 1 edge, got %d", len(builder.edges))
	}
	edge := builder.edges[0]
	if edge.From != "start" || edge.To != "end" {
		t.Errorf("expected edge from 'start' to 'end', got from %q to %q", edge.From, edge.To)
	}
}

func TestWorkflowBuilder_AddConditionalEdge(t *testing.T) {
	// Arrange
	start := newBuilderTestExecutor("start")
	end := newBuilderTestExecutor("end")
	builder := NewBuilder(start).AddExecutor(end)
	condition := func(state map[string]interface{}) bool {
		return true
	}

	// Act
	result := builder.AddConditionalEdge("start", "end", condition)

	// Assert
	if result != builder {
		t.Error("expected fluent return")
	}
	if len(builder.edges) != 1 {
		t.Errorf("expected 1 edge, got %d", len(builder.edges))
	}
	edge := builder.edges[0]
	if edge.Condition == nil {
		t.Error("expected edge to have condition")
	}
}

func TestWorkflowBuilder_AddEdgeFromTo(t *testing.T) {
	// Arrange
	start := newBuilderTestExecutor("start")
	end := newBuilderTestExecutor("end")
	builder := NewBuilder(start).AddExecutor(end)

	// Act
	result := builder.AddEdgeFromTo(start, end)

	// Assert
	if result != builder {
		t.Error("expected fluent return")
	}
	if len(builder.edges) != 1 {
		t.Errorf("expected 1 edge, got %d", len(builder.edges))
	}
}

func TestWorkflowBuilder_AddFanOut(t *testing.T) {
	// Arrange
	start := newBuilderTestExecutor("start")
	target1 := newBuilderTestExecutor("target1")
	target2 := newBuilderTestExecutor("target2")
	builder := NewBuilder(start).AddExecutors(target1, target2)

	// Act
	result := builder.AddFanOut("start", "target1", "target2")

	// Assert
	if result != builder {
		t.Error("expected fluent return")
	}
	if len(builder.edgeGroups) != 1 {
		t.Errorf("expected 1 edge group, got %d", len(builder.edgeGroups))
	}
	group := builder.edgeGroups[0]
	if group.Type != EdgeGroupTypeFanOut {
		t.Error("expected fan-out edge group type")
	}
	if group.From != "start" {
		t.Errorf("expected from 'start', got %q", group.From)
	}
	if len(group.Targets) != 2 {
		t.Errorf("expected 2 targets, got %d", len(group.Targets))
	}
}

func TestWorkflowBuilder_AddFanIn(t *testing.T) {
	// Arrange
	source1 := newBuilderTestExecutor("source1")
	source2 := newBuilderTestExecutor("source2")
	target := newBuilderTestExecutor("target")
	builder := NewBuilder(source1).AddExecutors(source2, target)

	// Act
	result := builder.AddFanIn([]string{"source1", "source2"}, "target")

	// Assert
	if result != builder {
		t.Error("expected fluent return")
	}
	if len(builder.edgeGroups) != 1 {
		t.Errorf("expected 1 edge group, got %d", len(builder.edgeGroups))
	}
	group := builder.edgeGroups[0]
	if group.Type != EdgeGroupTypeFanIn {
		t.Error("expected fan-in edge group type")
	}
	if group.To != "target" {
		t.Errorf("expected to 'target', got %q", group.To)
	}
	if len(group.Sources) != 2 {
		t.Errorf("expected 2 sources, got %d", len(group.Sources))
	}
}

func TestWorkflowBuilder_AddFanOutFromTo(t *testing.T) {
	// Arrange
	start := newBuilderTestExecutor("start")
	target1 := newBuilderTestExecutor("target1")
	target2 := newBuilderTestExecutor("target2")
	builder := NewBuilder(start).AddExecutors(target1, target2)

	// Act
	result := builder.AddFanOutFromTo(start, target1, target2)

	// Assert
	if result != builder {
		t.Error("expected fluent return")
	}
	if len(builder.edgeGroups) != 1 {
		t.Errorf("expected 1 edge group, got %d", len(builder.edgeGroups))
	}
}

func TestWorkflowBuilder_AddFanInFromTo(t *testing.T) {
	// Arrange
	source1 := newBuilderTestExecutor("source1")
	source2 := newBuilderTestExecutor("source2")
	target := newBuilderTestExecutor("target")
	builder := NewBuilder(source1).AddExecutors(source2, target)

	// Act
	result := builder.AddFanInFromTo([]Executor{source1, source2}, target)

	// Assert
	if result != builder {
		t.Error("expected fluent return")
	}
	if len(builder.edgeGroups) != 1 {
		t.Errorf("expected 1 edge group, got %d", len(builder.edgeGroups))
	}
}

func TestWorkflowBuilder_SwitchFrom(t *testing.T) {
	// Arrange
	start := newBuilderTestExecutor("start")
	caseA := newBuilderTestExecutor("caseA")
	caseB := newBuilderTestExecutor("caseB")
	defaultCase := newBuilderTestExecutor("default")
	builder := NewBuilder(start).AddExecutors(caseA, caseB, defaultCase)
	selector := func(state map[string]interface{}) interface{} {
		return state["choice"]
	}

	// Act
	result := builder.SwitchFrom("start", selector).
		Case("a", "caseA").
		Case("b", "caseB").
		Default("default").
		End()

	// Assert
	if result != builder {
		t.Error("expected fluent return")
	}
	if len(builder.switchEdges) != 1 {
		t.Errorf("expected 1 switch edge, got %d", len(builder.switchEdges))
	}
	sw := builder.switchEdges[0]
	if sw.From != "start" {
		t.Errorf("expected from 'start', got %q", sw.From)
	}
	if len(sw.Cases) != 2 {
		t.Errorf("expected 2 cases, got %d", len(sw.Cases))
	}
	if sw.Default != "default" {
		t.Errorf("expected default 'default', got %q", sw.Default)
	}
}

func TestWorkflowBuilder_MarkAsOutput(t *testing.T) {
	// Arrange
	start := newBuilderTestExecutor("start")
	end := newBuilderTestExecutor("end")
	builder := NewBuilder(start).AddExecutor(end)

	// Act
	result := builder.MarkAsOutput("end")

	// Assert
	if result != builder {
		t.Error("expected fluent return")
	}
	if !builder.outputExecutors["end"] {
		t.Error("expected 'end' to be marked as output")
	}
}

func TestWorkflowBuilder_Build_Success(t *testing.T) {
	// Arrange
	start := newBuilderTestExecutor("start")
	end := newBuilderTestExecutor("end")
	builder := NewBuilder(start).
		WithName("test").
		WithDescription("test description").
		AddExecutor(end).
		AddEdge("start", "end")

	// Act
	wf, err := builder.Build()

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if wf == nil {
		t.Fatal("expected workflow to be non-nil")
	}
	if wf.Name() != "test" {
		t.Errorf("expected name 'test', got %q", wf.Name())
	}
	if wf.Description() != "test description" {
		t.Errorf("expected description 'test description', got %q", wf.Description())
	}
	if wf.StartID() != "start" {
		t.Errorf("expected start ID 'start', got %q", wf.StartID())
	}
}

func TestWorkflowBuilder_Build_AutoDetectsLeafNodes(t *testing.T) {
	// Arrange
	start := newBuilderTestExecutor("start")
	end := newBuilderTestExecutor("end")
	builder := NewBuilder(start).
		AddExecutor(end).
		AddEdge("start", "end")

	// Act
	wf, err := builder.Build()

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !wf.IsOutputExecutor("end") {
		t.Error("expected 'end' to be auto-detected as output executor")
	}
	if wf.IsOutputExecutor("start") {
		t.Error("expected 'start' to not be an output executor")
	}
}

func TestWorkflowBuilder_Build_ValidationError_MissingEdgeTarget(t *testing.T) {
	// Arrange
	start := newBuilderTestExecutor("start")
	builder := NewBuilder(start).
		AddEdge("start", "missing")

	// Act
	wf, err := builder.Build()

	// Assert
	if err == nil {
		t.Fatal("expected error for missing edge target")
	}
	if wf != nil {
		t.Error("expected workflow to be nil on error")
	}
}

func TestWorkflowBuilder_Build_ValidationError_MissingEdgeSource(t *testing.T) {
	// Arrange
	start := newBuilderTestExecutor("start")
	end := newBuilderTestExecutor("end")
	builder := NewBuilder(start).
		AddExecutor(end)
	builder.edges = append(builder.edges, NewEdge("missing", "end"))

	// Act
	wf, err := builder.Build()

	// Assert
	if err == nil {
		t.Fatal("expected error for missing edge source")
	}
	if wf != nil {
		t.Error("expected workflow to be nil on error")
	}
}

func TestWorkflowBuilder_Build_WithFanOutValidation(t *testing.T) {
	// Arrange
	start := newBuilderTestExecutor("start")
	target1 := newBuilderTestExecutor("target1")
	target2 := newBuilderTestExecutor("target2")
	builder := NewBuilder(start).
		AddExecutors(target1, target2).
		AddFanOut("start", "target1", "target2")

	// Act
	wf, err := builder.Build()

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	targets := wf.GetTargets("start", nil)
	if len(targets) != 2 {
		t.Errorf("expected 2 targets, got %d", len(targets))
	}
}

func TestWorkflowBuilder_Build_WithSwitchEdgeValidation(t *testing.T) {
	// Arrange
	start := newBuilderTestExecutor("start")
	caseA := newBuilderTestExecutor("caseA")
	caseB := newBuilderTestExecutor("caseB")
	selector := func(state map[string]interface{}) interface{} {
		return state["choice"]
	}
	builder := NewBuilder(start).
		AddExecutors(caseA, caseB).
		SwitchFrom("start", selector).
		Case("a", "caseA").
		Case("b", "caseB").
		End()

	// Act
	wf, err := builder.Build()

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	targets := wf.GetTargets("start", map[string]interface{}{"choice": "a"})
	if len(targets) != 1 || targets[0] != "caseA" {
		t.Errorf("expected targets ['caseA'], got %v", targets)
	}
}

func TestWorkflowBuilder_Build_ImmutableWorkflow(t *testing.T) {
	// Arrange
	start := newBuilderTestExecutor("start")
	end := newBuilderTestExecutor("end")
	builder := NewBuilder(start).
		AddExecutor(end).
		AddEdge("start", "end")

	// Act
	wf, err := builder.Build()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Modify builder after Build
	extra := newBuilderTestExecutor("extra")
	builder.AddExecutor(extra)

	// Assert - workflow should not be affected
	if _, exists := wf.executors["extra"]; exists {
		t.Error("expected workflow to be immutable after Build")
	}
}

func TestWorkflowBuilder_FindLeafNodes(t *testing.T) {
	// Arrange
	start := newBuilderTestExecutor("start")
	middle := newBuilderTestExecutor("middle")
	end1 := newBuilderTestExecutor("end1")
	end2 := newBuilderTestExecutor("end2")
	builder := NewBuilder(start).
		AddExecutors(middle, end1, end2).
		AddEdge("start", "middle").
		AddFanOut("middle", "end1", "end2")

	// Act
	leaves := builder.findLeafNodes()

	// Assert
	if len(leaves) != 2 {
		t.Errorf("expected 2 leaf nodes, got %d", len(leaves))
	}
	leafMap := make(map[string]bool)
	for _, id := range leaves {
		leafMap[id] = true
	}
	if !leafMap["end1"] || !leafMap["end2"] {
		t.Errorf("expected end1 and end2 as leaf nodes, got %v", leaves)
	}
}

func TestWorkflowBuilder_ComplexWorkflow(t *testing.T) {
	// Arrange - build a complex workflow with multiple patterns
	start := newBuilderTestExecutor("start")
	branch1 := newBuilderTestExecutor("branch1")
	branch2 := newBuilderTestExecutor("branch2")
	merge := newBuilderTestExecutor("merge")
	conditional := newBuilderTestExecutor("conditional")
	switchExec := newBuilderTestExecutor("switch")
	caseA := newBuilderTestExecutor("caseA")
	caseB := newBuilderTestExecutor("caseB")
	end := newBuilderTestExecutor("end")

	selector := func(state map[string]interface{}) interface{} {
		return state["route"]
	}
	condition := func(state map[string]interface{}) bool {
		return state["enabled"] == true
	}

	// Act
	wf, err := NewBuilder(start).
		WithName("complex-workflow").
		AddExecutors(branch1, branch2, merge, conditional, switchExec, caseA, caseB, end).
		AddFanOut("start", "branch1", "branch2").
		AddFanIn([]string{"branch1", "branch2"}, "merge").
		AddConditionalEdge("merge", "conditional", condition).
		AddEdge("conditional", "switch").
		SwitchFrom("switch", selector).
		Case("a", "caseA").
		Case("b", "caseB").
		Default("end").
		End().
		AddEdge("caseA", "end").
		AddEdge("caseB", "end").
		MarkAsOutput("end").
		Build()

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if wf.Name() != "complex-workflow" {
		t.Errorf("expected name 'complex-workflow', got %q", wf.Name())
	}
	if len(wf.Executors()) != 9 {
		t.Errorf("expected 9 executors, got %d", len(wf.Executors()))
	}
	if !wf.IsOutputExecutor("end") {
		t.Error("expected 'end' to be output executor")
	}
}
