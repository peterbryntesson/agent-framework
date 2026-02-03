// Copyright (c) Microsoft. All rights reserved.

package chatagent

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"time"

	"github.com/google/uuid"
	"github.com/microsoft/agent-framework-go/agent"
	"github.com/microsoft/agent-framework-go/chat"
	"github.com/microsoft/agent-framework-go/tool"
)

// Agent implements agent.Agent by wrapping a chat.Client.
// This is the primary agent implementation in the framework, providing
// automatic tool invocation, session management, and streaming support.
type Agent struct {
	id          string
	name        string
	description string
	client      chat.Client

	// Configuration
	instructions string
	tools        []tool.Tool
	maxTurns     int

	// Invocation configuration
	invocationConfig   tool.InvocationConfig
	functionMiddleware agent.FunctionMiddleware
	chatMiddleware     agent.ChatMiddleware
	contextProvider    agent.ContextProvider

	// Services for extensibility
	services map[reflect.Type]interface{}
}

// Compile-time check that Agent implements agent.Agent.
var _ agent.Agent = (*Agent)(nil)

// New creates a new ChatClientAgent from a chat.Client.
// Use Option functions to configure the agent's behavior.
func New(client chat.Client, opts ...Option) *Agent {
	cfg := defaultConfig()
	for _, opt := range opts {
		if opt != nil {
			opt(cfg)
		}
	}

	a := &Agent{
		id:                 cfg.id,
		name:               cfg.name,
		description:        cfg.description,
		client:             client,
		instructions:       cfg.instructions,
		tools:              cfg.tools,
		maxTurns:           cfg.maxTurns,
		invocationConfig:   cfg.invocationConfig,
		functionMiddleware: agent.ChainFunctionMiddleware(cfg.functionMiddleware...),
		chatMiddleware:     agent.ChainChatMiddleware(cfg.chatMiddleware...),
		services:           make(map[reflect.Type]interface{}),
	}

	// Initialize context provider
	if len(cfg.contextProviders) > 0 {
		if len(cfg.contextProviders) == 1 {
			a.contextProvider = cfg.contextProviders[0]
		} else {
			a.contextProvider = agent.NewAggregateContextProvider(cfg.contextProviders...)
		}
	}

	// Generate ID if not provided
	if a.id == "" {
		a.id = uuid.NewString()
	}

	// Use model ID as name if not provided
	if a.name == "" && client != nil {
		a.name = client.Metadata().ModelID
	}

	return a
}

// ID returns the unique identifier for this agent.
func (a *Agent) ID() string {
	return a.id
}

// Name returns the human-readable name of the agent.
func (a *Agent) Name() string {
	return a.name
}

// Description returns the description of the agent's purpose.
func (a *Agent) Description() string {
	return a.description
}

// Metadata returns provider-specific metadata about the agent.
func (a *Agent) Metadata() agent.AIAgentMetadata {
	metadata := agent.AIAgentMetadata{}

	if a.client != nil {
		clientMeta := a.client.Metadata()
		metadata.ProviderName = clientMeta.ProviderName
	}

	return metadata
}

// Run executes the agent with the provided messages and returns a complete response.
func (a *Agent) Run(ctx context.Context, messages []agent.Message, opts ...agent.RunOption) (*agent.Response, error) {
	cfg := agent.ApplyRunOptions(opts...)

	// Get dynamic context from provider
	providerCtx, err := a.getProviderContext(ctx, messages)
	if err != nil {
		return nil, fmt.Errorf("context provider error: %w", err)
	}

	// Prepare messages with instructions and provider context
	chatMessages := a.prepareMessages(messages, cfg, providerCtx)

	// Prepare chat options with tools and provider tools
	chatOptions := a.prepareChatOptions(cfg, providerCtx)

	// Execute with tool invocation loop
	response, runErr := a.runWithToolLoop(ctx, chatMessages, chatOptions, cfg)

	// Notify provider of completion (best effort, don't fail on error)
	a.notifyProviderInvoked(ctx, messages, response, runErr)

	return response, runErr
}

// RunStream executes the agent and returns a channel of incremental response updates.
func (a *Agent) RunStream(ctx context.Context, messages []agent.Message, opts ...agent.RunOption) (<-chan agent.ResponseUpdate, error) {
	cfg := agent.ApplyRunOptions(opts...)

	// Get dynamic context from provider
	providerCtx, err := a.getProviderContext(ctx, messages)
	if err != nil {
		return nil, fmt.Errorf("context provider error: %w", err)
	}

	chatMessages := a.prepareMessages(messages, cfg, providerCtx)
	chatOptions := a.prepareChatOptions(cfg, providerCtx)

	return a.runStreamWithToolLoop(ctx, chatMessages, chatOptions, cfg)
}

// NewSession creates a new agent session for maintaining conversation state.
func (a *Agent) NewSession(ctx context.Context) (agent.Session, error) {
	return newSession(a.id), nil
}

// RestoreSession deserializes a previously saved session from JSON data.
func (a *Agent) RestoreSession(ctx context.Context, data json.RawMessage) (agent.Session, error) {
	return restoreSession(data)
}

// GetService retrieves a service of the specified type from the agent.
func (a *Agent) GetService(serviceType reflect.Type) interface{} {
	return a.services[serviceType]
}

// RegisterService registers a service that can be retrieved via GetService.
func (a *Agent) RegisterService(service interface{}) {
	if service == nil {
		return
	}
	serviceType := reflect.TypeOf(service)
	a.services[serviceType] = service
}

// Client returns the underlying chat.Client.
// This is useful for advanced scenarios requiring direct client access.
func (a *Agent) Client() chat.Client {
	return a.client
}

// Tools returns the tools configured for this agent.
func (a *Agent) Tools() []tool.Tool {
	return a.tools
}

// Instructions returns the system instructions for this agent.
func (a *Agent) Instructions() string {
	return a.instructions
}

// prepareMessages prepares messages for the chat client.
// Adds system instructions, provider context, and session history.
func (a *Agent) prepareMessages(messages []agent.Message, cfg *agent.RunConfig, providerCtx *agent.Context) []chat.Message {
	// Calculate capacity: instructions + provider messages + session messages + new messages
	capacity := len(messages) + 1
	if cfg.Session != nil {
		capacity += len(cfg.Session.Messages())
	}
	if providerCtx != nil {
		capacity += len(providerCtx.Messages) + 1 // +1 for potential provider instructions
	}

	chatMessages := make([]chat.Message, 0, capacity)

	// 1. Combine base agent instructions with provider instructions
	instructions := a.instructions
	if providerCtx != nil && providerCtx.Instructions != "" {
		if instructions != "" {
			instructions = instructions + "\n" + providerCtx.Instructions
		} else {
			instructions = providerCtx.Instructions
		}
	}

	// 2. Add combined system instructions
	if instructions != "" {
		chatMessages = append(chatMessages, chat.NewSystemMessage(instructions))
	}

	// 3. Add provider messages (context, examples, retrieved documents)
	if providerCtx != nil && len(providerCtx.Messages) > 0 {
		chatMessages = append(chatMessages, providerCtx.Messages...)
	}

	// 4. Add session history if provided
	if cfg.Session != nil {
		chatMessages = append(chatMessages, cfg.Session.Messages()...)
	}

	// 5. Add new messages from this invocation
	chatMessages = append(chatMessages, messages...)

	return chatMessages
}

// prepareChatOptions creates chat.Options from run configuration.
func (a *Agent) prepareChatOptions(cfg *agent.RunConfig, providerCtx *agent.Context) *chat.Options {
	opts := chat.NewOptions()

	// 1. Add base agent tools
	for _, t := range a.tools {
		opts.Tools = append(opts.Tools, toolToDefinition(t))
	}

	// 2. Add provider tools
	if providerCtx != nil {
		for _, t := range providerCtx.Tools {
			opts.Tools = append(opts.Tools, toolToDefinition(t))
		}
	}

	// 3. Add run-specific tools (highest priority)
	for _, t := range cfg.Tools {
		if ft, ok := t.(tool.Tool); ok {
			opts.Tools = append(opts.Tools, toolToDefinition(ft))
		}
	}

	// Apply run configuration
	if cfg.MaxTokens > 0 {
		opts.MaxTokens = cfg.MaxTokens
	}
	if cfg.Temperature != 0 {
		opts.Temperature = cfg.Temperature
	}

	return opts
}

// toolToDefinition converts a tool.Tool to chat.ToolDefinition.
func toolToDefinition(t tool.Tool) chat.ToolDefinition {
	var params map[string]interface{}
	if t.Parameters() != nil {
		_ = json.Unmarshal(t.Parameters(), &params)
	}

	return chat.ToolDefinition{
		Name:        t.Name(),
		Description: t.Description(),
		Parameters:  params,
	}
}

// buildResponse creates an agent.Response from accumulated data.
func (a *Agent) buildResponse(messages []chat.Message, usage chat.UsageDetails, finishReason chat.FinishReason) *agent.Response {
	resp := &agent.Response{
		AgentID:   a.id,
		CreatedAt: time.Now(),
		Messages:  messages,
		Usage: &agent.UsageDetails{
			InputTokens:  usage.InputTokens,
			OutputTokens: usage.OutputTokens,
			TotalTokens:  usage.TotalTokens,
		},
		FinishReason: convertFinishReason(finishReason),
	}

	// Generate response ID
	resp.ResponseID = uuid.NewString()

	return resp
}

// convertFinishReason converts chat.FinishReason to agent.FinishReason.
func convertFinishReason(fr chat.FinishReason) agent.FinishReason {
	switch fr {
	case chat.FinishReasonStop:
		return agent.FinishReasonStop
	case chat.FinishReasonLength:
		return agent.FinishReasonLength
	case chat.FinishReasonToolCalls:
		return agent.FinishReasonToolCalls
	case chat.FinishReasonContentFilter:
		return agent.FinishReasonContentFilter
	default:
		return agent.FinishReasonStop
	}
}

// getAllTools returns all tools including agent tools and run-specific tools.
func (a *Agent) getAllTools(cfg *agent.RunConfig) []tool.Tool {
	allTools := make([]tool.Tool, 0, len(a.tools)+len(cfg.Tools))
	allTools = append(allTools, a.tools...)

	for _, t := range cfg.Tools {
		if ft, ok := t.(tool.Tool); ok {
			allTools = append(allTools, ft)
		}
	}

	return allTools
}

// getProviderContext invokes the context provider and returns the context to inject.
// Returns nil if no context provider is configured.
func (a *Agent) getProviderContext(ctx context.Context, messages []agent.Message) (*agent.Context, error) {
	if a.contextProvider == nil {
		return nil, nil
	}
	return a.contextProvider.Invoking(ctx, messages)
}

// notifyProviderInvoked calls the Invoked lifecycle hook on the context provider.
// This is a best-effort notification; errors are logged but not returned.
func (a *Agent) notifyProviderInvoked(ctx context.Context, request []agent.Message, response *agent.Response, invokeErr error) {
	if a.contextProvider == nil {
		return
	}
	lcp, ok := a.contextProvider.(agent.ContextProviderWithLifecycle)
	if !ok {
		return
	}
	var responseMessages []agent.Message
	if response != nil {
		responseMessages = response.Messages
	}
	// Best effort - log error but don't propagate
	_ = lcp.Invoked(ctx, request, responseMessages, invokeErr)
}

// notifyProviderSessionCreated calls the SessionCreated lifecycle hook on the context provider.
// This is a best-effort notification; errors are logged but not returned.
func (a *Agent) notifyProviderSessionCreated(ctx context.Context, sessionID string) {
	if a.contextProvider == nil {
		return
	}
	lcp, ok := a.contextProvider.(agent.ContextProviderWithLifecycle)
	if !ok {
		return
	}
	// Best effort - log error but don't propagate
	_ = lcp.SessionCreated(ctx, sessionID)
}
