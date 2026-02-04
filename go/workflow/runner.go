// Copyright (c) Microsoft. All rights reserved.

package workflow

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/microsoft/agent-framework-go/agent"
)

// WorkflowRunner executes workflows using a Pregel-like superstep model.
// In each superstep, executors process their pending messages in parallel
// and produce output messages for the next superstep. Execution continues
// until no new messages are produced (convergence) or the maximum superstep
// limit is reached.
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
// If the workflow does not converge within this limit, execution stops.
// Default is 100 supersteps.
func WithMaxSupersteps(max int) RunnerOption {
	return func(o *runnerOptions) {
		o.maxSupersteps = max
	}
}

// WithCheckpointStore sets the checkpoint store for persistence.
// When set, the runner can save and restore workflow state.
func WithCheckpointStore(store CheckpointStore) RunnerOption {
	return func(o *runnerOptions) {
		o.checkpointStore = store
	}
}

// WithRunID sets a specific run ID instead of generating one.
// This is useful for resuming from checkpoints or correlating logs.
func WithRunID(runID string) RunnerOption {
	return func(o *runnerOptions) {
		o.runID = runID
	}
}

// NewRunner creates a new WorkflowRunner for the given workflow.
func NewRunner(workflow *Workflow, opts ...RunnerOption) *WorkflowRunner {
	options := runnerOptions{
		maxSupersteps: 100, // Default limit
	}
	for _, opt := range opts {
		opt(&options)
	}
	return &WorkflowRunner{
		workflow: workflow,
		options:  options,
	}
}

// Run executes the workflow with the given input and returns the final result.
// The workflow starts by sending the input to the start executor and continues
// executing supersteps until convergence (no new messages) or the maximum
// superstep limit is reached.
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
		From:      inputExecutorID,
		To:        r.workflow.startID,
		Content:   newInputMessage(input),
		Superstep: -1,
	})

	superstep := 0
	executedSupersteps := 0
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

		executedSupersteps++
		outputs = append(outputs, stepOutputs...)

		// Check for convergence (no new messages)
		if len(newMessages) == 0 {
			break
		}

		messages = newMessages
		superstep++
	}

	// Convert state to map
	finalState := syncMapToMap(state)

	return &WorkflowResult{
		RunID:          runID,
		Outputs:        outputs,
		FinalState:     finalState,
		SuperstepCount: executedSupersteps,
	}, nil
}

// RunStream executes the workflow and returns a channel of events.
// This allows callers to observe workflow progress in real-time.
// The channel is closed when execution completes or fails.
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
			From:      inputExecutorID,
			To:        r.workflow.startID,
			Content:   newInputMessage(input),
			Superstep: -1,
		})

		superstep := 0
		executedSupersteps := 0
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

			executedSupersteps++

			// Emit output events
			for i := range stepOutputs {
				events <- WorkflowEvent{
					Kind:      EventKindOutput,
					RunID:     runID,
					Superstep: superstep,
					Message:   &stepOutputs[i],
					Timestamp: time.Now(),
				}
			}

			outputs = append(outputs, stepOutputs...)

			events <- WorkflowEvent{
				Kind:      EventKindSuperstepCompleted,
				RunID:     runID,
				Superstep: superstep,
				Timestamp: time.Now(),
			}

			// Check for convergence
			if len(newMessages) == 0 {
				break
			}

			messages = newMessages
			superstep++
		}

		finalState := syncMapToMap(state)

		result := &WorkflowResult{
			RunID:          runID,
			Outputs:        outputs,
			FinalState:     finalState,
			SuperstepCount: executedSupersteps,
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

// inputExecutorID is the virtual executor ID for the workflow input.
const inputExecutorID = "__input__"

// newInputMessage creates a user message from the input string.
func newInputMessage(input string) agent.Message {
	return agent.NewUserMessage(input)
}

// executeSuperstep executes one superstep of the workflow.
// Returns new messages for the next superstep and any outputs.
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

			newMsgs, stepOutputs := r.routeMessages(exec.ID(), wCtx.Outbox(), superstep, state)

			mu.Lock()
			allNewMessages = append(allNewMessages, newMsgs...)
			outputs = append(outputs, stepOutputs...)
			mu.Unlock()
		}(executor, execMessages)
	}

	wg.Wait()

	if execErr != nil {
		return nil, nil, execErr
	}

	return allNewMessages, outputs, nil
}

// executeSuperstepWithEvents executes a superstep and emits events.
func (r *WorkflowRunner) executeSuperstepWithEvents(
	ctx context.Context,
	runID string,
	superstep int,
	messages []WorkflowMessage,
	state *sync.Map,
	events chan<- WorkflowEvent,
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

			// Emit executor invoked event
			events <- WorkflowEvent{
				Kind:       EventKindExecutorInvoked,
				RunID:      runID,
				Superstep:  superstep,
				ExecutorID: exec.ID(),
				Timestamp:  time.Now(),
			}

			wCtx := newWorkflowContext(ctx, exec.ID(), runID, superstep, msgs, state)

			if err := exec.Execute(ctx, wCtx); err != nil {
				events <- WorkflowEvent{
					Kind:       EventKindExecutorFailed,
					RunID:      runID,
					Superstep:  superstep,
					ExecutorID: exec.ID(),
					Error:      err,
					Timestamp:  time.Now(),
				}
				mu.Lock()
				if execErr == nil {
					execErr = fmt.Errorf("executor %q failed: %w", exec.ID(), err)
				}
				mu.Unlock()
				return
			}

			// Emit executor completed event
			events <- WorkflowEvent{
				Kind:       EventKindExecutorCompleted,
				RunID:      runID,
				Superstep:  superstep,
				ExecutorID: exec.ID(),
				Timestamp:  time.Now(),
			}

			newMsgs, stepOutputs := r.routeMessages(exec.ID(), wCtx.Outbox(), superstep, state)

			mu.Lock()
			allNewMessages = append(allNewMessages, newMsgs...)
			outputs = append(outputs, stepOutputs...)
			mu.Unlock()
		}(executor, execMessages)
	}

	wg.Wait()

	if execErr != nil {
		return nil, nil, execErr
	}

	return allNewMessages, outputs, nil
}

// routeMessages routes outbox messages to their targets based on edges.
// Returns the routed messages and any outputs from output executors.
func (r *WorkflowRunner) routeMessages(
	executorID string,
	outbox []WorkflowMessage,
	superstep int,
	state *sync.Map,
) ([]WorkflowMessage, []WorkflowMessage) {
	var routed []WorkflowMessage
	var outputs []WorkflowMessage

	stateMap := syncMapToMap(state)

	for _, msg := range outbox {
		if msg.To != "" {
			// Explicit target specified
			routed = append(routed, WorkflowMessage{
				From:      executorID,
				To:        msg.To,
				Content:   msg.Content,
				Superstep: superstep,
			})
		} else {
			// Route via edges
			targets := r.workflow.GetTargets(executorID, stateMap)
			for _, target := range targets {
				routed = append(routed, WorkflowMessage{
					From:      executorID,
					To:        target,
					Content:   msg.Content,
					Superstep: superstep,
				})
			}
		}

		// Collect outputs from output executors
		if r.workflow.IsOutputExecutor(executorID) {
			outputs = append(outputs, msg)
		}
	}

	return routed, outputs
}

// syncMapToMap converts a sync.Map to a regular map.
func syncMapToMap(m *sync.Map) map[string]interface{} {
	result := make(map[string]interface{})
	m.Range(func(key, value interface{}) bool {
		result[key.(string)] = value
		return true
	})
	return result
}
