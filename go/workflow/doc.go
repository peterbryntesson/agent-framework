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
