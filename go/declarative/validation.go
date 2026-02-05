// Copyright (c) Microsoft. All rights reserved.

package declarative

import (
	"errors"
	"fmt"
)

// ValidationError represents an error in the agent definition.
type ValidationError struct {
	Field   string
	Message string
}

func (e ValidationError) Error() string {
	if e.Field != "" {
		return fmt.Sprintf("%s: %s", e.Field, e.Message)
	}
	return e.Message
}

// validate checks the PromptAgent for required fields and valid values.
func validate(agent *PromptAgent) error {
	var errs []error

	// Check kind
	if agent.Kind == "" {
		errs = append(errs, ValidationError{Field: "kind", Message: "required"})
	} else if agent.Kind != "Prompt" {
		errs = append(errs, ValidationError{Field: "kind", Message: fmt.Sprintf("unsupported kind %q, expected \"Prompt\"", agent.Kind)})
	}

	// Check name
	if agent.Name == "" {
		errs = append(errs, ValidationError{Field: "name", Message: "required"})
	}

	// Check instructions
	if agent.Instructions == "" {
		errs = append(errs, ValidationError{Field: "instructions", Message: "required"})
	}

	// Validate tools
	for i, t := range agent.Tools {
		if t.Kind == "" {
			errs = append(errs, ValidationError{
				Field:   fmt.Sprintf("tools[%d].kind", i),
				Message: "required",
			})
		}
		if t.Name == "" {
			errs = append(errs, ValidationError{
				Field:   fmt.Sprintf("tools[%d].name", i),
				Message: "required",
			})
		}
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}

	return nil
}

// ValidateAgent validates a PromptAgent and returns all errors found.
func ValidateAgent(agent *PromptAgent) error {
	return validate(agent)
}
