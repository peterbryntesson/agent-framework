// Copyright (c) Microsoft. All rights reserved.

package durable

import (
	"context"

	"github.com/microsoft/agent-framework-go/agent"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
)

// Worker manages the Temporal worker lifecycle for durable agents.
// It registers the workflow and activity implementations and processes
// tasks from the configured task queue.
type Worker struct {
	// client is the Temporal client.
	client client.Client

	// worker is the Temporal worker instance.
	worker worker.Worker

	// agent is the agent to execute in activities.
	agent agent.Agent

	// taskQueue is the Temporal task queue name.
	taskQueue string

	// options contains worker configuration.
	options WorkerOptions
}

// WorkerOptions configures a Worker.
type WorkerOptions struct {
	// MaxConcurrentActivityExecutionSize is the maximum number of concurrent activities.
	MaxConcurrentActivityExecutionSize int

	// MaxConcurrentWorkflowTaskExecutionSize is the maximum number of concurrent workflow tasks.
	MaxConcurrentWorkflowTaskExecutionSize int

	// EnableSessionWorker enables sticky session support.
	EnableSessionWorker bool
}

// DefaultWorkerOptions returns sensible default options.
func DefaultWorkerOptions() WorkerOptions {
	return WorkerOptions{
		MaxConcurrentActivityExecutionSize:     10,
		MaxConcurrentWorkflowTaskExecutionSize: 100,
		EnableSessionWorker:                    false,
	}
}

// WorkerOption configures a Worker.
type WorkerOption func(*Worker)

// WithWorkerOptions sets the worker options.
func WithWorkerOptions(opts WorkerOptions) WorkerOption {
	return func(w *Worker) {
		w.options = opts
	}
}

// WithMaxConcurrentActivities sets the maximum concurrent activities.
func WithMaxConcurrentActivities(n int) WorkerOption {
	return func(w *Worker) {
		w.options.MaxConcurrentActivityExecutionSize = n
	}
}

// WithMaxConcurrentWorkflows sets the maximum concurrent workflow tasks.
func WithMaxConcurrentWorkflows(n int) WorkerOption {
	return func(w *Worker) {
		w.options.MaxConcurrentWorkflowTaskExecutionSize = n
	}
}

// NewWorker creates a new Temporal worker for durable agents.
func NewWorker(temporalClient client.Client, a agent.Agent, taskQueue string, opts ...WorkerOption) *Worker {
	w := &Worker{
		client:    temporalClient,
		agent:     a,
		taskQueue: taskQueue,
		options:   DefaultWorkerOptions(),
	}

	for _, opt := range opts {
		opt(w)
	}

	return w
}

// Start starts the worker.
// This blocks until the worker is stopped or an error occurs.
func (w *Worker) Start() error {
	// Create worker options
	workerOpts := worker.Options{
		MaxConcurrentActivityExecutionSize:     w.options.MaxConcurrentActivityExecutionSize,
		MaxConcurrentWorkflowTaskExecutionSize: w.options.MaxConcurrentWorkflowTaskExecutionSize,
		EnableSessionWorker:                    w.options.EnableSessionWorker,
	}

	// Create the Temporal worker
	w.worker = worker.New(w.client, w.taskQueue, workerOpts)

	// Register the workflow
	w.worker.RegisterWorkflow(SessionWorkflow)

	// Register activities with the agent context
	w.worker.RegisterActivityWithOptions(
		w.createActivityFunc(),
		activity.RegisterOptions{
			Name: RunAgentActivityName,
		},
	)

	w.worker.RegisterActivityWithOptions(
		w.createStreamingActivityFunc(),
		activity.RegisterOptions{
			Name: RunAgentStreamingActivityName,
		},
	)

	// Start the worker (blocks until stopped)
	return w.worker.Run(worker.InterruptCh())
}

// StartAsync starts the worker asynchronously.
// Returns immediately after starting the worker.
func (w *Worker) StartAsync() error {
	// Create worker options
	workerOpts := worker.Options{
		MaxConcurrentActivityExecutionSize:     w.options.MaxConcurrentActivityExecutionSize,
		MaxConcurrentWorkflowTaskExecutionSize: w.options.MaxConcurrentWorkflowTaskExecutionSize,
		EnableSessionWorker:                    w.options.EnableSessionWorker,
	}

	// Create the Temporal worker
	w.worker = worker.New(w.client, w.taskQueue, workerOpts)

	// Register the workflow
	w.worker.RegisterWorkflow(SessionWorkflow)

	// Register activities with the agent context
	w.worker.RegisterActivityWithOptions(
		w.createActivityFunc(),
		activity.RegisterOptions{
			Name: RunAgentActivityName,
		},
	)

	w.worker.RegisterActivityWithOptions(
		w.createStreamingActivityFunc(),
		activity.RegisterOptions{
			Name: RunAgentStreamingActivityName,
		},
	)

	// Start the worker asynchronously
	return w.worker.Start()
}

// Stop stops the worker gracefully.
func (w *Worker) Stop() {
	if w.worker != nil {
		w.worker.Stop()
	}
}

// createActivityFunc creates the activity function with agent context.
func (w *Worker) createActivityFunc() func(ctx context.Context, input ActivityInput) (ActivityResult, error) {
	return func(ctx context.Context, input ActivityInput) (ActivityResult, error) {
		// Inject the agent into the context
		ctx = WithActivityContext(ctx, ActivityContext{Agent: w.agent})
		return RunAgentActivity(ctx, input)
	}
}

// createStreamingActivityFunc creates the streaming activity function with agent context.
func (w *Worker) createStreamingActivityFunc() func(ctx context.Context, input ActivityInput) (StreamingActivityResult, error) {
	return func(ctx context.Context, input ActivityInput) (StreamingActivityResult, error) {
		// Inject the agent into the context
		ctx = WithActivityContext(ctx, ActivityContext{Agent: w.agent})
		return RunAgentStreamingActivity(ctx, input)
	}
}

// TaskQueue returns the task queue name.
func (w *Worker) TaskQueue() string {
	return w.taskQueue
}

// Client returns the Temporal client.
func (w *Worker) Client() client.Client {
	return w.client
}
