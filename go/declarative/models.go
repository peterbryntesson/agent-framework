// Copyright (c) Microsoft. All rights reserved.

package declarative

// PromptAgent represents a declarative agent definition.
// This struct maps to the YAML schema used for agent definitions
// and is compatible with .NET and Python implementations.
type PromptAgent struct {
	// Schema is the JSON Schema reference for validation.
	Schema string `yaml:"$schema,omitempty" json:"$schema,omitempty"`

	// Kind identifies the agent type. Must be "Prompt".
	Kind string `yaml:"kind" json:"kind"`

	// Name is the human-readable name of the agent.
	Name string `yaml:"name" json:"name"`

	// Description provides details about the agent's purpose.
	Description string `yaml:"description,omitempty" json:"description,omitempty"`

	// Instructions are the system instructions for the agent.
	Instructions string `yaml:"instructions" json:"instructions"`

	// Model defines the AI model configuration.
	Model Model `yaml:"model" json:"model"`

	// Tools lists the tools available to the agent.
	Tools []Tool `yaml:"tools,omitempty" json:"tools,omitempty"`

	// Inputs defines the expected input parameters.
	Inputs []PropertySchema `yaml:"inputs,omitempty" json:"inputs,omitempty"`

	// Outputs defines the expected output schema.
	Outputs []PropertySchema `yaml:"outputs,omitempty" json:"outputs,omitempty"`

	// OutputSchema defines the structured output schema (alternative to Outputs).
	OutputSchema *OutputSchema `yaml:"outputSchema,omitempty" json:"outputSchema,omitempty"`
}

// Model defines the AI model configuration.
type Model struct {
	// ID is the model or deployment identifier.
	// Supports =Env.VAR_NAME syntax for environment variable substitution.
	ID string `yaml:"id,omitempty" json:"id,omitempty"`

	// Provider specifies the AI provider: "OpenAI", "AzureOpenAI", "Anthropic".
	Provider string `yaml:"provider,omitempty" json:"provider,omitempty"`

	// APIType specifies the API type: "Chat", "Responses", "Assistants".
	APIType string `yaml:"apiType,omitempty" json:"apiType,omitempty"`

	// Endpoint is the API endpoint URL (optional, for custom endpoints).
	// Supports =Env.VAR_NAME syntax.
	Endpoint string `yaml:"endpoint,omitempty" json:"endpoint,omitempty"`

	// Connection defines authentication settings.
	Connection *Connection `yaml:"connection,omitempty" json:"connection,omitempty"`

	// Options contains model-specific configuration.
	Options *ModelOptions `yaml:"options,omitempty" json:"options,omitempty"`
}

// Connection defines authentication settings for the model.
type Connection struct {
	// Kind specifies the connection type: "key", "remote", "anonymous".
	Kind string `yaml:"kind,omitempty" json:"kind,omitempty"`

	// Type is an alias for Kind (for compatibility).
	Type string `yaml:"type,omitempty" json:"type,omitempty"`

	// Key is the API key or reference to an environment variable.
	// Supports =Env.VAR_NAME syntax.
	Key string `yaml:"key,omitempty" json:"key,omitempty"`

	// Source specifies the key source (for compatibility).
	Source string `yaml:"source,omitempty" json:"source,omitempty"`
}

// ModelOptions contains optional model configuration.
type ModelOptions struct {
	// Temperature controls randomness (0-2).
	Temperature *float64 `yaml:"temperature,omitempty" json:"temperature,omitempty"`

	// TopP controls nucleus sampling.
	TopP *float64 `yaml:"topP,omitempty" json:"topP,omitempty"`

	// MaxTokens limits the response length.
	MaxTokens *int `yaml:"maxTokens,omitempty" json:"maxTokens,omitempty"`

	// AllowMultipleToolCalls enables parallel tool calls.
	AllowMultipleToolCalls bool `yaml:"allowMultipleToolCalls,omitempty" json:"allowMultipleToolCalls,omitempty"`

	// ChatToolMode controls tool selection: "auto", "required", "none".
	ChatToolMode string `yaml:"chatToolMode,omitempty" json:"chatToolMode,omitempty"`
}

// Tool defines a tool available to the agent.
type Tool struct {
	// Kind specifies the tool type: "function", "mcp", "webSearch", "codeInterpreter", "fileSearch".
	Kind string `yaml:"kind" json:"kind"`

	// Name is the unique identifier for the tool.
	Name string `yaml:"name" json:"name"`

	// Description explains what the tool does.
	Description string `yaml:"description,omitempty" json:"description,omitempty"`

	// Server is the MCP server endpoint (for MCP tools).
	Server string `yaml:"server,omitempty" json:"server,omitempty"`

	// Binding is a single binding name for the tool implementation.
	Binding string `yaml:"binding,omitempty" json:"binding,omitempty"`

	// Bindings maps parameter names to function bindings.
	Bindings map[string]string `yaml:"bindings,omitempty" json:"bindings,omitempty"`

	// Parameters defines the tool's input parameters (for function tools).
	Parameters *ParameterSchema `yaml:"parameters,omitempty" json:"parameters,omitempty"`
}

// ParameterSchema defines a JSON Schema for tool parameters.
type ParameterSchema struct {
	// Type is the JSON Schema type (usually "object").
	Type string `yaml:"type,omitempty" json:"type,omitempty"`

	// Properties maps parameter names to their schemas.
	Properties map[string]PropertySchema `yaml:"properties,omitempty" json:"properties,omitempty"`

	// Required lists required parameter names.
	Required []string `yaml:"required,omitempty" json:"required,omitempty"`
}

// PropertySchema defines a parameter or output schema property.
type PropertySchema struct {
	// Name is the property name (used in arrays).
	Name string `yaml:"name,omitempty" json:"name,omitempty"`

	// Kind specifies the type: "string", "number", "boolean", "array", "object".
	Kind string `yaml:"kind,omitempty" json:"kind,omitempty"`

	// Type is an alias for Kind (JSON Schema compatibility).
	Type string `yaml:"type,omitempty" json:"type,omitempty"`

	// Description explains the property.
	Description string `yaml:"description,omitempty" json:"description,omitempty"`

	// Required indicates if the property is required.
	Required bool `yaml:"required,omitempty" json:"required,omitempty"`

	// Default is the default value if not provided.
	Default any `yaml:"default,omitempty" json:"default,omitempty"`

	// Enum lists allowed values.
	Enum []string `yaml:"enum,omitempty" json:"enum,omitempty"`

	// Items defines the schema for array elements.
	Items *PropertySchema `yaml:"items,omitempty" json:"items,omitempty"`

	// Properties defines nested object properties.
	Properties map[string]PropertySchema `yaml:"properties,omitempty" json:"properties,omitempty"`
}

// OutputSchema defines the structured output schema for the agent.
type OutputSchema struct {
	// Properties maps output field names to their schemas.
	Properties map[string]PropertySchema `yaml:"properties,omitempty" json:"properties,omitempty"`
}

// GetConnectionKind returns the connection kind, handling the Kind/Type alias.
func (c *Connection) GetConnectionKind() string {
	if c == nil {
		return ""
	}
	if c.Kind != "" {
		return c.Kind
	}
	return c.Type
}

// GetKey returns the API key, handling the Key/Source alias.
func (c *Connection) GetKey() string {
	if c == nil {
		return ""
	}
	if c.Key != "" {
		return c.Key
	}
	return c.Source
}

// GetPropertyType returns the property type, handling the Kind/Type alias.
func (p *PropertySchema) GetPropertyType() string {
	if p.Kind != "" {
		return p.Kind
	}
	return p.Type
}
