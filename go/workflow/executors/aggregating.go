// Copyright (c) Microsoft. All rights reserved.

package executors

import (
	"context"
	"sync"

	"github.com/microsoft/agent-framework-go/agent"
	"github.com/microsoft/agent-framework-go/workflow"
)

// AggregateFunc combines multiple messages into one or more output messages.
// This function is called when all expected source executors have sent their messages.
type AggregateFunc func(messages []agent.Message) ([]agent.Message, error)

// AggregatingExecutor collects messages from multiple sources and aggregates them.
// This executor is useful for fan-in patterns where multiple upstream executors
// produce results that need to be combined before continuing the workflow.
//
// The executor tracks which sources have sent messages and only invokes the
// aggregator function when all expected sources have contributed.
type AggregatingExecutor struct {
	workflow.ExecutorBase
	expectedSources []string
	aggregator      AggregateFunc
	collected       map[string][]agent.Message
	mu              sync.Mutex
}

// NewAggregatingExecutor creates an AggregatingExecutor.
// The id parameter should be unique within the workflow.
// The sources parameter lists executor IDs that this executor expects to receive messages from.
// The aggregator function is called when all sources have sent at least one message.
func NewAggregatingExecutor(id string, sources []string, aggregator AggregateFunc) *AggregatingExecutor {
	return &AggregatingExecutor{
		ExecutorBase:    workflow.NewExecutorBase(id),
		expectedSources: sources,
		aggregator:      aggregator,
		collected:       make(map[string][]agent.Message),
	}
}

// Execute collects messages and aggregates when all sources have sent.
// Messages are accumulated across supersteps until all expected sources
// have contributed. Once complete, the aggregator function is invoked
// and the collected messages are cleared for the next aggregation cycle.
func (ae *AggregatingExecutor) Execute(ctx context.Context, wCtx *workflow.WorkflowContext) error {
	ae.mu.Lock()
	defer ae.mu.Unlock()

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

	// Aggregate all messages in source order
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

	// Clear collected messages for next aggregation cycle
	ae.collected = make(map[string][]agent.Message)

	return nil
}

// ExpectedSources returns the list of expected source executor IDs.
func (ae *AggregatingExecutor) ExpectedSources() []string {
	ae.mu.Lock()
	defer ae.mu.Unlock()
	result := make([]string, len(ae.expectedSources))
	copy(result, ae.expectedSources)
	return result
}

// Reset clears all collected messages without invoking the aggregator.
// This is useful for testing or when resetting workflow state.
func (ae *AggregatingExecutor) Reset() {
	ae.mu.Lock()
	defer ae.mu.Unlock()
	ae.collected = make(map[string][]agent.Message)
}
