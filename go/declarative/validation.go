// Copyright (c) Microsoft. All rights reserved.

package declarative

import (
	"errors"
	"fmt"
	"strings"
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

	// Validate model
	if err := validateModel(agent.Model); err != nil {
		errs = append(errs, err)
	}

	// Validate inputs and outputs schemas
	for i, input := range agent.Inputs {
		validatePropertySchema(fmt.Sprintf("inputs[%d]", i), input, &errs)
	}
	for i, output := range agent.Outputs {
		validatePropertySchema(fmt.Sprintf("outputs[%d]", i), output, &errs)
	}
	if agent.OutputSchema != nil {
		for name, prop := range agent.OutputSchema.Properties {
			validatePropertySchema(fmt.Sprintf("outputSchema.properties.%s", name), prop, &errs)
		}
	}

	// Validate tools
	for i, t := range agent.Tools {
		kind := normalizeToolKind(t.Kind)
		if kind == "" {
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

		switch kind {
		case "function", "mcp", "websearch", "filesearch", "codeinterpreter", "":
			// supported
		default:
			errs = append(errs, ValidationError{
				Field:   fmt.Sprintf("tools[%d].kind", i),
				Message: fmt.Sprintf("unsupported kind %q", t.Kind),
			})
		}

		if kind == "mcp" {
			serverURL := t.Server
			if t.URL != "" {
				serverURL = t.URL
			}
			if serverURL == "" {
				errs = append(errs, ValidationError{
					Field:   fmt.Sprintf("tools[%d].url", i),
					Message: "required",
				})
			}
			if t.ApprovalMode != nil && t.ApprovalMode.Kind != "" {
				kindValue := strings.ToLower(strings.TrimSpace(t.ApprovalMode.Kind))
				if kindValue != "never" && kindValue != "always" && kindValue != "specify" {
					errs = append(errs, ValidationError{
						Field:   fmt.Sprintf("tools[%d].approvalMode.kind", i),
						Message: fmt.Sprintf("unsupported approval mode %q", t.ApprovalMode.Kind),
					})
				}
			}
		}

		if kind == "function" && t.Parameters != nil {
			validateParameterSchema(fmt.Sprintf("tools[%d].parameters", i), t.Parameters, &errs)
		}
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}

	return nil
}

func validateModel(model Model) error {
	var errs []error

	provider := strings.TrimSpace(model.Provider)
	if provider != "" {
		switch strings.ToLower(provider) {
		case "openai", "azureopenai", "azure_openai", "azure-openai":
			// supported
		default:
			errs = append(errs, ValidationError{Field: "model.provider", Message: fmt.Sprintf("unsupported provider %q", model.Provider)})
		}
	}

	apiType := normalizeAPIType(model.APIType)
	if apiType != "" {
		switch apiType {
		case "chat", "responses":
			// supported
		default:
			errs = append(errs, ValidationError{Field: "model.apiType", Message: fmt.Sprintf("unsupported apiType %q", model.APIType)})
		}
	}

	if strings.TrimSpace(model.ID) == "" && provider == "" && apiType == "" && strings.TrimSpace(model.Endpoint) == "" && model.Connection == nil && model.Options == nil {
		errs = append(errs, ValidationError{Field: "model", Message: "required"})
	}

	if strings.EqualFold(provider, "AzureOpenAI") || strings.EqualFold(provider, "azure_openai") || strings.EqualFold(provider, "azure-openai") {
		if strings.TrimSpace(model.Endpoint) == "" {
			errs = append(errs, ValidationError{Field: "model.endpoint", Message: "required for AzureOpenAI"})
		}
		if strings.TrimSpace(model.ID) == "" {
			errs = append(errs, ValidationError{Field: "model.id", Message: "required for AzureOpenAI"})
		}
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	return nil
}

func validateParameterSchema(path string, schema *ParameterSchema, errs *[]error) {
	if schema == nil {
		return
	}
	if schema.Type != "" && schema.Type != "object" {
		*errs = append(*errs, ValidationError{Field: path + ".type", Message: "expected object"})
	}
	for name, prop := range schema.Properties {
		validatePropertySchema(fmt.Sprintf("%s.properties.%s", path, name), prop, errs)
	}
}

func validatePropertySchema(path string, prop PropertySchema, errs *[]error) {
	propType := strings.TrimSpace(prop.GetPropertyType())
	if propType == "" {
		*errs = append(*errs, ValidationError{Field: path + ".type", Message: "required"})
		return
	}

	switch propType {
	case "string", "number", "integer", "boolean":
		return
	case "array":
		if prop.Items == nil {
			*errs = append(*errs, ValidationError{Field: path + ".items", Message: "required for array"})
			return
		}
		validatePropertySchema(path+".items", *prop.Items, errs)
	case "object":
		if len(prop.Properties) == 0 {
			*errs = append(*errs, ValidationError{Field: path + ".properties", Message: "required for object"})
			return
		}
		for name, child := range prop.Properties {
			validatePropertySchema(fmt.Sprintf("%s.properties.%s", path, name), child, errs)
		}
	default:
		*errs = append(*errs, ValidationError{Field: path + ".type", Message: fmt.Sprintf("unsupported type %q", propType)})
	}
}

// ValidateAgent validates a PromptAgent and returns all errors found.
func ValidateAgent(agent *PromptAgent) error {
	return validate(agent)
}
