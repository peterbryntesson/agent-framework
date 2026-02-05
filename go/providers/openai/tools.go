// Copyright (c) Microsoft. All rights reserved.

package openai

import (
	"github.com/microsoft/agent-framework-go/tool"
	openailib "github.com/sashabaranov/go-openai"
)

// ConvertToolsToOpenAI converts tool.Tool slice to OpenAI tool definitions.
// This function filters out hosted tools, which are handled separately
// by ConvertHostedToolsToOpenAI.
//
// Example:
//
//	tools := []tool.Tool{myFunctionTool, webSearchTool}
//	oaiTools := openai.ConvertToolsToOpenAI(tools)
func ConvertToolsToOpenAI(tools []tool.Tool) []openailib.Tool {
	result := make([]openailib.Tool, 0, len(tools))

	for _, t := range tools {
		// Skip hosted tools - they're added differently via Responses API
		if ht, ok := t.(tool.HostedTool); ok && ht.IsHosted() {
			continue
		}

		result = append(result, openailib.Tool{
			Type: openailib.ToolTypeFunction,
			Function: &openailib.FunctionDefinition{
				Name:        t.Name(),
				Description: t.Description(),
				Parameters:  t.Parameters(),
			},
		})
	}

	return result
}

// ConvertHostedToolsToOpenAI returns hosted tool configurations for OpenAI requests.
// These configurations are used with the Responses API for provider-hosted tools
// like web search, code interpreter, and file search.
//
// Example:
//
//	tools := []tool.Tool{myFunctionTool, webSearchTool}
//	hostedConfigs := openai.ConvertHostedToolsToOpenAI(tools)
func ConvertHostedToolsToOpenAI(tools []tool.Tool) []map[string]interface{} {
	var hosted []map[string]interface{}

	for _, t := range tools {
		if ht, ok := t.(tool.HostedTool); ok && ht.IsHosted() {
			hosted = append(hosted, ht.ProviderConfig())
		}
	}

	return hosted
}

// HasHostedTools checks if any tools in the slice are hosted tools.
// This can be used to determine whether to use the Responses API
// (which supports hosted tools) vs the Chat Completions API.
func HasHostedTools(tools []tool.Tool) bool {
	for _, t := range tools {
		if ht, ok := t.(tool.HostedTool); ok && ht.IsHosted() {
			return true
		}
	}
	return false
}

// SeparateTools splits a slice of tools into function tools and hosted tools.
// This is useful when building requests that need to handle each type differently.
//
// Example:
//
//	tools := []tool.Tool{funcTool1, webSearch, funcTool2, codeInterpreter}
//	funcTools, hostedTools := openai.SeparateTools(tools)
//	// funcTools = [funcTool1, funcTool2]
//	// hostedTools = [webSearch, codeInterpreter]
func SeparateTools(tools []tool.Tool) (functionTools []tool.Tool, hostedTools []tool.HostedTool) {
	for _, t := range tools {
		if ht, ok := t.(tool.HostedTool); ok && ht.IsHosted() {
			hostedTools = append(hostedTools, ht)
		} else {
			functionTools = append(functionTools, t)
		}
	}
	return functionTools, hostedTools
}

// ToolChoiceForOpenAI converts a tool.ToolChoice or string to OpenAI format.
// Returns the appropriate value for the ToolChoice field in requests.
//
// Supported values:
//   - tool.ToolChoiceAuto, "auto": Model decides whether to call tools
//   - tool.ToolChoiceRequired, "required": Model must call at least one tool
//   - tool.ToolChoiceNone, "none": Model cannot call tools
//   - Any other string: Force the model to call the specified tool
func ToolChoiceForOpenAI(choice interface{}) interface{} {
	switch v := choice.(type) {
	case tool.ToolChoice:
		switch v {
		case tool.ToolChoiceAuto:
			return "auto"
		case tool.ToolChoiceRequired:
			return "required"
		case tool.ToolChoiceNone:
			return "none"
		default:
			return string(v)
		}
	case string:
		switch v {
		case "auto", "required", "none":
			return v
		case "":
			return nil
		default:
			// Specific tool name - force calling this tool
			return openailib.ToolChoice{
				Type: openailib.ToolTypeFunction,
				Function: openailib.ToolFunction{
					Name: v,
				},
			}
		}
	case tool.SpecificToolChoice:
		return openailib.ToolChoice{
			Type: openailib.ToolTypeFunction,
			Function: openailib.ToolFunction{
				Name: v.Name,
			},
		}
	default:
		return nil
	}
}

// ParallelToolCallsOption returns the appropriate setting for parallel tool calls.
// Some providers support executing multiple tool calls in parallel for efficiency.
// Returns a pointer to bool for optional field handling.
func ParallelToolCallsOption(enabled bool) *bool {
	return &enabled
}
