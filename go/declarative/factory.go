// Copyright (c) Microsoft. All rights reserved.

package declarative

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/microsoft/agent-framework-go/agent"
	"github.com/microsoft/agent-framework-go/chat"
	"github.com/microsoft/agent-framework-go/chatagent"
	"github.com/microsoft/agent-framework-go/tool"
)

// AgentFactory creates agent instances from declarative YAML definitions.
// It manages provider bindings and tool registrations needed to
// instantiate agents from YAML configuration.
type AgentFactory struct {
	bindings         map[string]interface{}
	providerBuilders map[string]ProviderBuilder
	toolParsers      map[string]ToolParser
}

// ProviderBuilder is a function that creates a chat client from a model configuration.
// The returned interface must implement chat.Client.
type ProviderBuilder func(model Model) (chat.Client, error)

// ToolParser is a function that creates a tool from a tool definition.
// The returned interface must implement tool.Tool.
type ToolParser func(t Tool) (tool.Tool, error)

// NewAgentFactory creates a new agent factory with default configurations.
// The factory is pre-configured with default provider builders for common
// providers (OpenAI, Azure OpenAI) and default tool parsers.
func NewAgentFactory() *AgentFactory {
	factory := &AgentFactory{
		bindings:         make(map[string]interface{}),
		providerBuilders: make(map[string]ProviderBuilder),
		toolParsers:      make(map[string]ToolParser),
	}

	// Register default providers
	factory.providerBuilders["openai"] = buildOpenAIProvider
	factory.providerBuilders["azure_openai"] = buildAzureOpenAIProvider

	// Register default tool parsers
	factory.toolParsers["function"] = parseFunction
	factory.toolParsers[""] = parseFunction // default to function

	return factory
}

// WithBinding adds a named binding to the factory.
// Bindings can be used to inject dependencies into agents.
func (f *AgentFactory) WithBinding(name string, value interface{}) *AgentFactory {
	f.bindings[name] = value
	return f
}

// WithProvider registers a custom provider builder for a specific provider kind.
// This overrides any default provider builder with the same name.
func (f *AgentFactory) WithProvider(kind string, builder ProviderBuilder) *AgentFactory {
	f.providerBuilders[kind] = builder
	return f
}

// WithToolParser registers a custom tool parser for a specific tool kind.
// This overrides any default tool parser with the same name.
func (f *AgentFactory) WithToolParser(kind string, parser ToolParser) *AgentFactory {
	f.toolParsers[kind] = parser
	return f
}

// Create instantiates an agent from a PromptAgent definition.
// The agent definition should be loaded and validated before calling Create.
func (f *AgentFactory) Create(ctx context.Context, def *PromptAgent) (agent.Agent, error) {
	// Evaluate environment variables
	if err := evaluateAgent(def); err != nil {
		return nil, fmt.Errorf("evaluating agent definition: %w", err)
	}

	// Create the provider (chat client)
	client, err := f.createProvider(def.Model)
	if err != nil {
		return nil, fmt.Errorf("creating provider: %w", err)
	}

	// Create tools
	var tools []tool.Tool
	for _, toolDef := range def.Tools {
		t, err := f.parseTool(toolDef)
		if err != nil {
			return nil, fmt.Errorf("parsing tool %q: %w", toolDef.Name, err)
		}
		if t != nil {
			tools = append(tools, t)
		}
	}

	// Create the agent options
	opts := []chatagent.Option{
		chatagent.WithName(def.Name),
		chatagent.WithInstructions(def.Instructions),
	}

	if def.Description != "" {
		opts = append(opts, chatagent.WithDescription(def.Description))
	}

	if len(tools) > 0 {
		opts = append(opts, chatagent.WithTools(tools...))
	}

	// Create the agent
	chatAgent := chatagent.New(client, opts...)

	return chatAgent, nil
}

// CreateFromFile loads and creates an agent from a YAML file path.
func (f *AgentFactory) CreateFromFile(ctx context.Context, path string) (agent.Agent, error) {
	file, err := os.Open(path) //nolint:gosec // G304: User-provided path is the intended use case
	if err != nil {
		return nil, fmt.Errorf("opening file: %w", err)
	}
	defer file.Close()

	return f.CreateFromReader(ctx, file)
}

// CreateFromReader loads and creates an agent from a reader.
func (f *AgentFactory) CreateFromReader(ctx context.Context, reader io.Reader) (agent.Agent, error) {
	def, err := LoadFromReader(reader)
	if err != nil {
		return nil, err
	}

	if err := ValidateAgent(def); err != nil {
		return nil, err
	}

	return f.Create(ctx, def)
}

// CreateFromString loads and creates an agent from a YAML string.
func (f *AgentFactory) CreateFromString(ctx context.Context, content string) (agent.Agent, error) {
	def, err := LoadFromString(content)
	if err != nil {
		return nil, err
	}

	if err := ValidateAgent(def); err != nil {
		return nil, err
	}

	return f.Create(ctx, def)
}

// createProvider creates a chat client from a model configuration.
func (f *AgentFactory) createProvider(model Model) (chat.Client, error) {
	kind := getProviderKind(model)

	builder, ok := f.providerBuilders[kind]
	if !ok {
		return nil, fmt.Errorf("unknown provider kind: %q", kind)
	}

	return builder(model)
}

// getProviderKind returns the provider kind from the model configuration.
func getProviderKind(model Model) string {
	// Check Provider field first
	if model.Provider != "" {
		switch model.Provider {
		case "OpenAI":
			return "openai"
		case "AzureOpenAI":
			return "azure_openai"
		default:
			return model.Provider
		}
	}

	// Check connection kind
	if model.Connection != nil {
		kind := model.Connection.GetConnectionKind()
		if kind != "" {
			return kind
		}
	}

	// Default to OpenAI
	return "openai"
}

// parseTool parses a tool definition into a tool.Tool.
func (f *AgentFactory) parseTool(t Tool) (tool.Tool, error) {
	kind := t.Kind
	if kind == "" {
		kind = "function"
	}

	parser, ok := f.toolParsers[kind]
	if !ok {
		return nil, fmt.Errorf("unknown tool kind: %q", kind)
	}

	return parser(t)
}
