// Copyright (c) Microsoft. All rights reserved.

package chatagent

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/microsoft/agent-framework-go/agent"
	"github.com/microsoft/agent-framework-go/tool"
)

// excludedSessionKeys are keys that should not propagate to sub-agents by default.
var excludedSessionKeys = map[string]bool{
	"session_id":      true,
	"conversation_id": true,
	"thread_id":       true,
}

// AsToolOptions configures agent-to-tool conversion.
type AsToolOptions struct {
	// Name overrides the tool name (defaults to sanitized agent name).
	Name string

	// Description overrides the tool description (defaults to agent description).
	Description string

	// ArgName sets the parameter name for the task input (defaults to "task").
	ArgName string

	// ArgDescription describes the task parameter.
	ArgDescription string

	// StreamCallback receives streaming updates when set.
	// If nil, the tool uses non-streaming invocation.
	StreamCallback func(agent.ResponseUpdate)

	// ApprovalMode specifies when user approval is required.
	// Defaults to ApprovalNever.
	ApprovalMode tool.ApprovalMode

	// ForwardRuntimeContext enables runtime context propagation to sub-agent.
	// When true, RuntimeContext values are forwarded (excluding session keys).
	ForwardRuntimeContext bool

	// ExcludeKeys specifies additional keys to exclude from context forwarding.
	ExcludeKeys []string

	// PreserveCase keeps the original casing when sanitizing the agent name.
	// Default is false (names are lowercased).
	PreserveCase bool
}

// excludeKey checks if a key should be excluded from forwarding.
func (o *AsToolOptions) excludeKey(key string) bool {
	for _, k := range o.ExcludeKeys {
		if k == key {
			return true
		}
	}
	return false
}

// AsTool converts an agent to a tool for use by other agents.
// This enables hierarchical agent patterns where one agent can
// delegate work to specialized sub-agents.
func AsTool(a agent.Agent, opts AsToolOptions) tool.Tool {
	name := opts.Name
	if name == "" {
		name = sanitizeAgentName(a.Name())
	}

	desc := opts.Description
	if desc == "" {
		desc = a.Description()
	}
	if desc == "" {
		desc = fmt.Sprintf("Delegate task to %s agent", name)
	}

	argName := opts.ArgName
	if argName == "" {
		argName = "task"
	}

	argDesc := opts.ArgDescription
	if argDesc == "" {
		argDesc = fmt.Sprintf("Task for the %s agent to perform", name)
	}

	// Create parameters schema
	params := map[string]any{
		"type": "object",
		"properties": map[string]any{
			argName: map[string]any{
				"type":        "string",
				"description": argDesc,
			},
		},
		"required": []string{argName},
	}
	paramsJSON, _ := json.Marshal(params)

	// Create a function that invokes the agent
	fn := func(ctx context.Context, args json.RawMessage) (string, error) {
		// Parse args to get the task input
		var parsed map[string]string
		if err := json.Unmarshal(args, &parsed); err != nil {
			return "", fmt.Errorf("failed to parse arguments: %w", err)
		}

		input := parsed[argName]
		messages := []agent.Message{agent.NewUserMessage(input)}

		// Build run options with forwarded runtime context
		var runOpts []agent.RunOption

		if opts.ForwardRuntimeContext {
			// Extract runtime context from context.Context
			parentCtx := agent.RuntimeCtxFromContext(ctx)

			// Create filtered context excluding session-related keys
			filteredCtx := agent.NewRuntimeContext()
			for k, v := range parentCtx.Values() {
				if !excludedSessionKeys[k] && !opts.excludeKey(k) {
					filteredCtx = filteredCtx.With(k, v)
				}
			}

			if filteredCtx.Len() > 0 {
				runOpts = append(runOpts, agent.WithRuntimeContext(filteredCtx))
			}
		}

		if opts.StreamCallback != nil {
			// Streaming mode
			updates, err := a.RunStream(ctx, messages, runOpts...)
			if err != nil {
				return "", err
			}

			var lastText string
			for update := range updates {
				opts.StreamCallback(update)
				if update.Delta != nil && update.Delta.TextDelta != "" {
					lastText += update.Delta.TextDelta
				}
			}

			return lastText, nil
		}

		// Non-streaming mode
		resp, err := a.Run(ctx, messages, runOpts...)
		if err != nil {
			return "", err
		}

		return resp.Text(), nil
	}

	return &agentTool{
		name:         name,
		description:  desc,
		parameters:   paramsJSON,
		invoke:       fn,
		approvalMode: opts.ApprovalMode,
	}
}

// agentTool implements tool.Tool for agent invocation.
type agentTool struct {
	name         string
	description  string
	parameters   json.RawMessage
	invoke       func(ctx context.Context, args json.RawMessage) (string, error)
	approvalMode tool.ApprovalMode
}

// Name returns the tool name.
func (t *agentTool) Name() string {
	return t.name
}

// Description returns the tool description.
func (t *agentTool) Description() string {
	return t.description
}

// Parameters returns the JSON schema for tool parameters.
func (t *agentTool) Parameters() json.RawMessage {
	return t.parameters
}

// GetApprovalMode returns the tool's approval mode.
func (t *agentTool) GetApprovalMode() tool.ApprovalMode {
	return t.approvalMode
}

// Invoke executes the tool.
func (t *agentTool) Invoke(ctx context.Context, arguments json.RawMessage) (tool.Result, error) {
	result, err := t.invoke(ctx, arguments)
	if err != nil {
		return tool.Result{
			Content: err.Error(),
			IsError: true,
		}, err
	}
	return tool.NewResult(result), nil
}

// sanitizeAgentName converts an agent name to a valid tool identifier.
func sanitizeAgentName(name string) string {
	// Replace spaces and special chars with underscores
	re := regexp.MustCompile(`[^a-zA-Z0-9_]`)
	sanitized := re.ReplaceAllString(name, "_")

	// Remove consecutive underscores
	sanitized = regexp.MustCompile(`_+`).ReplaceAllString(sanitized, "_")

	// Trim underscores from ends
	sanitized = strings.Trim(sanitized, "_")

	// Ensure lowercase
	sanitized = strings.ToLower(sanitized)

	// Handle empty result
	if sanitized == "" {
		return "agent"
	}

	// Prefix with underscore if starts with digit
	if len(sanitized) > 0 && sanitized[0] >= '0' && sanitized[0] <= '9' {
		sanitized = "_" + sanitized
	}

	return sanitized
}

// Compile-time check that agentTool implements Tool.
var _ tool.Tool = (*agentTool)(nil)
