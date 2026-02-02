// Copyright (c) Microsoft. All rights reserved.

package tool

import (
	"context"
	"encoding/json"
)

// Tool represents a callable function that agents can invoke.
// This interface aligns with .NET AITool and Python ToolProtocol.
//
// Tools are the primary mechanism for agents to perform actions and
// interact with external systems. Each tool has a name, description,
// and JSON Schema defining its parameters.
type Tool interface {
	// Name returns the unique identifier for this tool.
	// The name should be a valid function identifier (alphanumeric and underscores).
	Name() string

	// Description returns a human-readable description of what this tool does.
	// This description is sent to the model to help it understand when to use the tool.
	Description() string

	// Parameters returns the JSON Schema describing the tool's input parameters.
	// Returns nil if the tool accepts no parameters.
	// The schema follows JSON Schema Draft 7 format.
	Parameters() json.RawMessage

	// Invoke executes the tool with the given arguments.
	// Arguments are provided as raw JSON matching the Parameters schema.
	// Returns the tool result or an error if invocation fails.
	Invoke(ctx context.Context, arguments json.RawMessage) (Result, error)
}

// HostedTool is a tool that runs on the provider's infrastructure.
// Examples include web search, code interpreter, and file search
// capabilities offered by providers like OpenAI and Azure.
//
// Hosted tools are not invoked locally; instead, the provider
// executes them and returns results as part of the response.
type HostedTool interface {
	Tool

	// IsHosted returns true, indicating this tool runs on the provider.
	// This method distinguishes hosted tools from locally-executed tools.
	IsHosted() bool

	// ProviderConfig returns provider-specific configuration for the hosted tool.
	// The returned map is serialized into the provider's tool definition format.
	// Keys and structure depend on the specific hosted tool type.
	ProviderConfig() map[string]interface{}
}

// AdditionalProperties contains optional metadata that can be attached to tools.
// This enables extensibility without modifying the core Tool interface.
type AdditionalProperties map[string]interface{}

// ToolChoice specifies how the model should select tools.
type ToolChoice string

const (
	// ToolChoiceAuto lets the model decide whether to call tools.
	ToolChoiceAuto ToolChoice = "auto"

	// ToolChoiceRequired forces the model to call at least one tool.
	ToolChoiceRequired ToolChoice = "required"

	// ToolChoiceNone prevents the model from calling any tools.
	ToolChoiceNone ToolChoice = "none"
)

// SpecificToolChoice forces the model to call a specific tool.
type SpecificToolChoice struct {
	// Name of the tool to call.
	Name string
}

// ApprovalMode specifies when user approval is required for tool invocation.
type ApprovalMode string

const (
	// ApprovalNever means the tool never requires user approval.
	ApprovalNever ApprovalMode = "never"

	// ApprovalAlways means the tool always requires user approval before execution.
	ApprovalAlways ApprovalMode = "always"

	// ApprovalOnce means approval is required once per session.
	ApprovalOnce ApprovalMode = "once"
)

// ToolType classifies the kind of tool for processing purposes.
type ToolType string

const (
	// ToolTypeFunction represents a locally-executed function tool.
	ToolTypeFunction ToolType = "function"

	// ToolTypeHostedWebSearch represents provider-hosted web search.
	ToolTypeHostedWebSearch ToolType = "web_search"

	// ToolTypeHostedCodeInterpreter represents provider-hosted code execution.
	ToolTypeHostedCodeInterpreter ToolType = "code_interpreter"

	// ToolTypeHostedFileSearch represents provider-hosted file/vector search.
	ToolTypeHostedFileSearch ToolType = "file_search"

	// ToolTypeHostedMCP represents a Model Context Protocol server tool.
	ToolTypeHostedMCP ToolType = "mcp"

	// ToolTypeHostedImageGeneration represents provider-hosted image generation.
	ToolTypeHostedImageGeneration ToolType = "image_generation"
)

// ToolCall represents a request from the model to invoke a tool.
// This is returned in chat responses when the model wants to use a tool.
type ToolCall struct {
	// ID uniquely identifies this tool call within the response.
	// This ID is used to correlate tool results with their calls.
	ID string `json:"id"`

	// Name is the name of the tool to invoke.
	Name string `json:"name"`

	// Arguments contains the tool arguments as raw JSON.
	Arguments json.RawMessage `json:"arguments"`
}

// ToolResult represents the outcome of a tool invocation.
// This is sent back to the model after executing a tool call.
type ToolResult struct {
	// CallID matches the ID from the corresponding ToolCall.
	CallID string `json:"call_id"`

	// Result contains the tool's output.
	Result Result `json:"result"`
}
