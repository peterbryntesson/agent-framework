// Copyright (c) Microsoft. All rights reserved.

package declarative

import (
	"os"
	"regexp"
)

// envPattern matches PowerFx-style environment variable expressions.
// Supported formats:
//   - =Env.VAR_NAME
//   - =Env("VAR_NAME")
//   - =Env['VAR_NAME']
var envPattern = regexp.MustCompile(`^=Env\.([A-Za-z_][A-Za-z0-9_]*)$`)
var envFuncPattern = regexp.MustCompile(`^=Env\(\s*['"]([A-Za-z_][A-Za-z0-9_]*)['"]\s*\)$`)
var envIndexPattern = regexp.MustCompile(`^=Env\[\s*['"]([A-Za-z_][A-Za-z0-9_]*)['"]\s*\]$`)

// TryEvalEnv evaluates a PowerFx-style environment variable expression.
// If the value matches =Env.VAR_NAME, it returns the environment variable value.
// Otherwise, it returns the original value unchanged.
//
// Examples:
//
//	TryEvalEnv("=Env.OPENAI_API_KEY") // returns os.Getenv("OPENAI_API_KEY"), true
//	TryEvalEnv("literal-value")       // returns "literal-value", false
func TryEvalEnv(expr string) (string, bool) {
	matches := envPattern.FindStringSubmatch(expr)
	if len(matches) == 2 {
		value := os.Getenv(matches[1])
		return value, true
	}

	matches = envFuncPattern.FindStringSubmatch(expr)
	if len(matches) == 2 {
		value := os.Getenv(matches[1])
		return value, true
	}

	matches = envIndexPattern.FindStringSubmatch(expr)
	if len(matches) == 2 {
		value := os.Getenv(matches[1])
		return value, true
	}
	return expr, false
}

// EvalEnvOrDefault evaluates an environment variable expression.
// If the expression is an =Env.VAR_NAME pattern, it returns the environment
// variable value. Otherwise, it returns the original value.
func EvalEnvOrDefault(expr string) string {
	if value, ok := TryEvalEnv(expr); ok {
		return value
	}
	return expr
}

// evaluateAgent recursively evaluates all string fields in the agent definition
// that may contain environment variable references.
func evaluateAgent(agent *PromptAgent) error {
	// Evaluate model ID
	if agent.Model.ID != "" {
		agent.Model.ID = EvalEnvOrDefault(agent.Model.ID)
	}

	// Evaluate endpoint
	if agent.Model.Endpoint != "" {
		agent.Model.Endpoint = EvalEnvOrDefault(agent.Model.Endpoint)
	}

	// Evaluate connection
	if agent.Model.Connection != nil {
		if agent.Model.Connection.Key != "" {
			agent.Model.Connection.Key = EvalEnvOrDefault(agent.Model.Connection.Key)
		}
		if agent.Model.Connection.Source != "" {
			agent.Model.Connection.Source = EvalEnvOrDefault(agent.Model.Connection.Source)
		}
	}

	// Evaluate tool servers (for MCP)
	for i := range agent.Tools {
		if agent.Tools[i].Server != "" {
			agent.Tools[i].Server = EvalEnvOrDefault(agent.Tools[i].Server)
		}
		if agent.Tools[i].URL != "" {
			agent.Tools[i].URL = EvalEnvOrDefault(agent.Tools[i].URL)
		}
	}

	return nil
}
