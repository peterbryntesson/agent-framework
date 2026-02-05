// Copyright (c) Microsoft. All rights reserved.

package executors

import (
	"context"
	"fmt"

	"github.com/microsoft/agent-framework-go/agent"
	"github.com/microsoft/agent-framework-go/workflow"
)

// AgentExecutor wraps an agent.Agent as a workflow Executor.
// This allows agents to participate in DAG-based workflow orchestration,
// receiving messages from upstream executors and passing responses downstream.
type AgentExecutor struct {
	workflow.ExecutorBase
	agent agent.Agent
}

// NewAgentExecutor creates an AgentExecutor from an agent.
// The id parameter should be unique within the workflow.
// The agent is invoked for each superstep where this executor receives messages.
func NewAgentExecutor(id string, a agent.Agent) *AgentExecutor {
	return &AgentExecutor{
		ExecutorBase: workflow.NewExecutorBase(id),
		agent:        a,
	}
}

// Execute processes incoming messages through the wrapped agent.
// Messages from the workflow context are converted to agent messages,
// passed to the agent's Run method, and the response messages are
// sent to downstream executors.
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
