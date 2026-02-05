// Copyright (c) Microsoft. All rights reserved.

package declarative

import (
	"strings"
	"testing"
)

func TestValidateAgentValid(t *testing.T) {
	agent := &PromptAgent{
		Kind:         "Prompt",
		Name:         "ValidAgent",
		Instructions: "Be helpful",
		Model: Model{
			ID: "gpt-4o",
		},
	}

	err := ValidateAgent(agent)
	if err != nil {
		t.Errorf("ValidateAgent() error = %v, want nil", err)
	}
}

func TestValidateAgentMissingKind(t *testing.T) {
	agent := &PromptAgent{
		Name:         "NoKindAgent",
		Instructions: "Be helpful",
		Model: Model{
			ID: "gpt-4o",
		},
	}

	err := ValidateAgent(agent)
	if err == nil {
		t.Error("ValidateAgent() expected error for missing kind")
	}

	// Check that error message contains "kind"
	if !strings.Contains(err.Error(), "kind") {
		t.Errorf("error should mention 'kind': %v", err)
	}
}

func TestValidateAgentInvalidKind(t *testing.T) {
	agent := &PromptAgent{
		Kind:         "Invalid",
		Name:         "InvalidKindAgent",
		Instructions: "Be helpful",
		Model: Model{
			ID: "gpt-4o",
		},
	}

	err := ValidateAgent(agent)
	if err == nil {
		t.Error("ValidateAgent() expected error for invalid kind")
	}
}

func TestValidateAgentMissingName(t *testing.T) {
	agent := &PromptAgent{
		Kind:         "Prompt",
		Instructions: "Be helpful",
		Model: Model{
			ID: "gpt-4o",
		},
	}

	err := ValidateAgent(agent)
	if err == nil {
		t.Error("ValidateAgent() expected error for missing name")
	}

	// Check that error message contains "name"
	if !strings.Contains(err.Error(), "name") {
		t.Errorf("error should mention 'name': %v", err)
	}
}

func TestValidateAgentMissingInstructions(t *testing.T) {
	agent := &PromptAgent{
		Kind: "Prompt",
		Name: "NoInstructionsAgent",
		Model: Model{
			ID: "gpt-4o",
		},
	}

	err := ValidateAgent(agent)
	if err == nil {
		t.Error("ValidateAgent() expected error for missing instructions")
	}

	// Check that error message contains "instructions"
	if !strings.Contains(err.Error(), "instructions") {
		t.Errorf("error should mention 'instructions': %v", err)
	}
}

func TestValidateAgentToolMissingName(t *testing.T) {
	agent := &PromptAgent{
		Kind:         "Prompt",
		Name:         "ToolAgent",
		Instructions: "Use tools",
		Model: Model{
			ID: "gpt-4o",
		},
		Tools: []Tool{
			{
				Kind:        "function",
				Description: "Does something",
			},
		},
	}

	err := ValidateAgent(agent)
	if err == nil {
		t.Error("ValidateAgent() expected error for tool missing name")
	}

	// Check that error message contains "tools[0].name"
	if !strings.Contains(err.Error(), "tools[0].name") {
		t.Errorf("error should mention 'tools[0].name': %v", err)
	}
}

func TestValidationErrorMessage(t *testing.T) {
	err := ValidationError{
		Field:   "name",
		Message: "is required",
	}

	// Check that the error message contains both field and message
	errStr := err.Error()
	if !strings.Contains(errStr, "name") {
		t.Errorf("Error() = %q, should contain field name", errStr)
	}
	if !strings.Contains(errStr, "is required") {
		t.Errorf("Error() = %q, should contain message", errStr)
	}
}
