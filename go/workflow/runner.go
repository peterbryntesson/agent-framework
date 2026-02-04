// Copyright (c) Microsoft. All rights reserved.

package workflow

import (
	"context"
	"errors"
)

// WorkflowRunner executes workflows.
// Full implementation is in Phase 3.
type WorkflowRunner struct {
	workflow *Workflow
	options  runnerOptions
}

// runnerOptions configures the WorkflowRunner.
type runnerOptions struct {
	maxSupersteps   int
	checkpointStore CheckpointStore
	runID           string
}

// RunnerOption configures the WorkflowRunner.
type RunnerOption func(*runnerOptions)

// WithMaxSupersteps sets the maximum number of supersteps.
func WithMaxSupersteps(max int) RunnerOption {
	return func(o *runnerOptions) {
		o.maxSupersteps = max
	}
}

// WithRunID sets a specific run ID instead of generating one.
func WithRunID(runID string) RunnerOption {
	return func(o *runnerOptions) {
		o.runID = runID
	}
}

// NewRunner creates a new WorkflowRunner.
func NewRunner(workflow *Workflow, opts ...RunnerOption) *WorkflowRunner {
	options := runnerOptions{
		maxSupersteps: 100,
	}
	for _, opt := range opts {
		opt(&options)
	}
	return &WorkflowRunner{
		workflow: workflow,
		options:  options,
	}
}

// Run executes the workflow with the given input.
// Placeholder implementation for Phase 1 - full implementation in Phase 3.
func (r *WorkflowRunner) Run(ctx context.Context, input string) (*WorkflowResult, error) {
	return nil, errors.New("workflow runner not implemented - see Phase 3")
}

// RunStream executes the workflow and returns a channel of events.
// Placeholder implementation for Phase 1 - full implementation in Phase 3.
func (r *WorkflowRunner) RunStream(ctx context.Context, input string) (<-chan WorkflowEvent, error) {
	return nil, errors.New("workflow runner not implemented - see Phase 3")
}
