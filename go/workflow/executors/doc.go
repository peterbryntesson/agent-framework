// Copyright (c) Microsoft. All rights reserved.

// Package executors provides built-in Executor implementations for common
// workflow patterns in the agent-framework-go SDK.
//
// This package includes:
//   - AgentExecutor: Wraps an agent.Agent for use within workflows
//   - FunctionExecutor: Wraps a simple handler function for transformations
//   - AggregatingExecutor: Collects messages from fan-in patterns
//
// # AgentExecutor
//
// AgentExecutor adapts an agent.Agent to the workflow.Executor interface,
// allowing agents to participate in DAG-based workflow orchestration:
//
//	executor := executors.NewAgentExecutor("researcher", myAgent)
//	builder.AddExecutor(executor)
//
// # FunctionExecutor
//
// FunctionExecutor wraps a simple function for lightweight transformations:
//
//	executor := executors.NewFunctionExecutor("transform", func(ctx context.Context, msgs []agent.Message) ([]agent.Message, error) {
//	    // Transform messages
//	    return transformed, nil
//	})
//
// # AggregatingExecutor
//
// AggregatingExecutor collects messages from multiple sources before
// processing, useful for fan-in patterns:
//
//	executor := executors.NewAggregatingExecutor("summarizer", []string{"analyzer1", "analyzer2"},
//	    func(msgs []agent.Message) ([]agent.Message, error) {
//	        // Combine results from all analyzers
//	        return combined, nil
//	    })
package executors
