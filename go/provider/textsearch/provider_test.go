// Copyright (c) Microsoft. All rights reserved.

package textsearch

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/microsoft/agent-framework-go/agent"
	"github.com/microsoft/agent-framework-go/chat"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockSearchFunc creates a search function that returns the given results.
func mockSearchFunc(results []SearchResult, err error) SearchFunc {
	return func(ctx context.Context, query string) ([]SearchResult, error) {
		return results, err
	}
}

func TestNew_DefaultOptions(t *testing.T) {
	searchFn := mockSearchFunc(nil, nil)
	p := New(searchFn)

	assert.NotNil(t, p)
	assert.Equal(t, DefaultMaxResults, p.options.MaxResults)
	assert.Equal(t, DefaultContextPrompt, p.options.ContextPrompt)
	assert.Equal(t, DefaultCitationsPrompt, p.options.CitationsPrompt)
	assert.Equal(t, BeforeAIInvoke, p.options.SearchBehavior)
	assert.Equal(t, DefaultSearchToolName, p.options.SearchToolName)
}

func TestNew_WithOptions(t *testing.T) {
	searchFn := mockSearchFunc(nil, nil)
	p := New(searchFn,
		WithMaxResults(10),
		WithContextPrompt("Custom prompt"),
		WithCitationsPrompt("Custom citations"),
		WithSearchBehavior(OnDemandFunctionCalling),
		WithSearchToolName("CustomSearch"),
		WithSearchToolDescription("Custom description"),
	)

	assert.Equal(t, 10, p.options.MaxResults)
	assert.Equal(t, "Custom prompt", p.options.ContextPrompt)
	assert.Equal(t, "Custom citations", p.options.CitationsPrompt)
	assert.Equal(t, OnDemandFunctionCalling, p.options.SearchBehavior)
	assert.Equal(t, "CustomSearch", p.options.SearchToolName)
	assert.Equal(t, "Custom description", p.options.SearchToolDescription)
}

func TestNew_OnDemandCreatesSearchTool(t *testing.T) {
	searchFn := mockSearchFunc(nil, nil)
	p := New(searchFn, WithSearchBehavior(OnDemandFunctionCalling))

	assert.NotNil(t, p.searchTool)
	assert.Equal(t, DefaultSearchToolName, p.searchTool.Name())
}

func TestInvoking_BeforeAIInvoke_SearchesAndInjectsContext(t *testing.T) {
	results := []SearchResult{
		{Name: "Doc1", Link: "http://example.com/1", Value: "Content 1"},
		{Name: "Doc2", Value: "Content 2"},
	}
	searchFn := mockSearchFunc(results, nil)
	p := New(searchFn)

	messages := []agent.Message{
		chat.NewUserMessage("What is the weather?"),
	}

	ctx := context.Background()
	agentCtx, err := p.Invoking(ctx, messages)

	require.NoError(t, err)
	require.NotNil(t, agentCtx)
	require.Len(t, agentCtx.Messages, 1)

	msg := agentCtx.Messages[0]
	assert.Equal(t, chat.RoleUser, msg.Role)
	content := msg.Text()
	assert.Contains(t, content, "Doc1")
	assert.Contains(t, content, "Content 1")
	assert.Contains(t, content, "http://example.com/1")
	assert.Contains(t, content, "Doc2")
	assert.Contains(t, content, "Content 2")

	// Check marker
	assert.Equal(t, providerMarker, msg.Name)
}

func TestInvoking_BeforeAIInvoke_HandlesEmptyQuery(t *testing.T) {
	searchFn := mockSearchFunc(nil, nil)
	p := New(searchFn)

	// Empty messages
	messages := []agent.Message{}

	ctx := context.Background()
	agentCtx, err := p.Invoking(ctx, messages)

	require.NoError(t, err)
	assert.Len(t, agentCtx.Messages, 0)
}

func TestInvoking_BeforeAIInvoke_HandlesSearchError(t *testing.T) {
	searchFn := mockSearchFunc(nil, errors.New("search failed"))
	p := New(searchFn)

	messages := []agent.Message{
		chat.NewUserMessage("Query text"),
	}

	ctx := context.Background()
	agentCtx, err := p.Invoking(ctx, messages)

	// Should not fail, just return empty context
	require.NoError(t, err)
	assert.Len(t, agentCtx.Messages, 0)
}

func TestInvoking_BeforeAIInvoke_LimitsResults(t *testing.T) {
	results := []SearchResult{
		{Name: "Doc1", Value: "Content 1"},
		{Name: "Doc2", Value: "Content 2"},
		{Name: "Doc3", Value: "Content 3"},
		{Name: "Doc4", Value: "Content 4"},
		{Name: "Doc5", Value: "Content 5"},
	}
	searchFn := mockSearchFunc(results, nil)
	p := New(searchFn, WithMaxResults(2))

	messages := []agent.Message{
		chat.NewUserMessage("Query"),
	}

	ctx := context.Background()
	agentCtx, err := p.Invoking(ctx, messages)

	require.NoError(t, err)
	require.Len(t, agentCtx.Messages, 1)

	content := agentCtx.Messages[0].Text()
	assert.Contains(t, content, "Doc1")
	assert.Contains(t, content, "Doc2")
	assert.NotContains(t, content, "Doc3")
	assert.NotContains(t, content, "Doc4")
	assert.NotContains(t, content, "Doc5")
}

func TestInvoking_BeforeAIInvoke_HandlesNoResults(t *testing.T) {
	searchFn := mockSearchFunc([]SearchResult{}, nil)
	p := New(searchFn)

	messages := []agent.Message{
		chat.NewUserMessage("Query"),
	}

	ctx := context.Background()
	agentCtx, err := p.Invoking(ctx, messages)

	require.NoError(t, err)
	assert.Len(t, agentCtx.Messages, 0)
}

func TestInvoking_OnDemand_ReturnsTool(t *testing.T) {
	searchFn := mockSearchFunc(nil, nil)
	p := New(searchFn, WithSearchBehavior(OnDemandFunctionCalling))

	messages := []agent.Message{
		chat.NewUserMessage("Query"),
	}

	ctx := context.Background()
	agentCtx, err := p.Invoking(ctx, messages)

	require.NoError(t, err)
	require.Len(t, agentCtx.Tools, 1)
	assert.Equal(t, DefaultSearchToolName, agentCtx.Tools[0].Name())
}

func TestInvoked_UpdatesMemory(t *testing.T) {
	searchFn := mockSearchFunc(nil, nil)
	p := New(searchFn)

	assert.Equal(t, 0, p.MemorySize())

	request := []agent.Message{
		chat.NewUserMessage("Hello"),
	}
	response := []agent.Message{
		chat.NewAssistantMessage("Hi there"),
	}

	err := p.Invoked(context.Background(), request, response, nil)
	require.NoError(t, err)

	assert.Equal(t, 2, p.MemorySize())
}

func TestInvoked_FiltersOwnOutput(t *testing.T) {
	searchFn := mockSearchFunc(nil, nil)
	p := New(searchFn)

	request := []agent.Message{
		chat.NewUserMessage("Hello"),
		{
			Role:     chat.RoleUser,
			Contents: []chat.Content{chat.NewTextContent("Search results")},
			Name:     providerMarker,
		},
	}

	err := p.Invoked(context.Background(), request, nil, nil)
	require.NoError(t, err)

	// Should only add the first message (not our output)
	assert.Equal(t, 1, p.MemorySize())
}

func TestFormatResults_Default(t *testing.T) {
	searchFn := mockSearchFunc(nil, nil)
	p := New(searchFn)

	results := []SearchResult{
		{Name: "Document 1", Link: "http://example.com", Value: "This is the content."},
	}

	formatted := p.formatResults(results)

	assert.Contains(t, formatted, DefaultContextPrompt)
	assert.Contains(t, formatted, "**Document 1**")
	assert.Contains(t, formatted, "Source: http://example.com")
	assert.Contains(t, formatted, "This is the content.")
	assert.Contains(t, formatted, DefaultCitationsPrompt)
}

func TestFormatResults_Empty(t *testing.T) {
	searchFn := mockSearchFunc(nil, nil)
	p := New(searchFn)

	formatted := p.formatResults([]SearchResult{})
	assert.Equal(t, "", formatted)
}

func TestFormatResults_Custom(t *testing.T) {
	customFormatter := func(results []SearchResult) string {
		return "CUSTOM FORMAT"
	}

	searchFn := mockSearchFunc(nil, nil)
	p := New(searchFn, WithResultFormatter(customFormatter))

	results := []SearchResult{{Value: "test"}}
	formatted := p.formatResults(results)

	assert.Equal(t, "CUSTOM FORMAT", formatted)
}

func TestFormatResults_MultipleSeparator(t *testing.T) {
	searchFn := mockSearchFunc(nil, nil)
	p := New(searchFn)

	results := []SearchResult{
		{Value: "First"},
		{Value: "Second"},
	}

	formatted := p.formatResults(results)
	assert.Contains(t, formatted, "----")
}

func TestSerialize_Restore_RoundTrip(t *testing.T) {
	searchFn := mockSearchFunc(nil, nil)
	p := New(searchFn)

	// Add some messages to memory
	request := []agent.Message{
		chat.NewUserMessage("Test message"),
	}
	_ = p.Invoked(context.Background(), request, nil, nil)

	// Serialize
	data, err := p.Serialize()
	require.NoError(t, err)
	require.NotNil(t, data)

	// Create new provider and restore
	p2 := New(searchFn)
	err = p2.Restore(data)
	require.NoError(t, err)

	assert.Equal(t, 1, p2.MemorySize())
}

func TestNewFromState_RestoresProvider(t *testing.T) {
	// Create a state with valid message structure
	searchFn := mockSearchFunc(nil, nil)
	originalP := New(searchFn)
	_ = originalP.Invoked(context.Background(), []agent.Message{chat.NewUserMessage("Hello")}, nil, nil)
	state, _ := originalP.Serialize()

	p, err := NewFromState(searchFn, state, WithMaxResults(5))
	require.NoError(t, err)
	require.NotNil(t, p)

	assert.Equal(t, 5, p.options.MaxResults)
	assert.Equal(t, 1, p.MemorySize())
}

func TestNewFromState_InvalidState(t *testing.T) {
	state := json.RawMessage(`invalid json`)
	searchFn := mockSearchFunc(nil, nil)

	_, err := NewFromState(searchFn, state)
	require.Error(t, err)
}

func TestClearMemory(t *testing.T) {
	searchFn := mockSearchFunc(nil, nil)
	p := New(searchFn)

	// Add some memory
	request := []agent.Message{chat.NewUserMessage("Test")}
	_ = p.Invoked(context.Background(), request, nil, nil)
	assert.Equal(t, 1, p.MemorySize())

	// Clear
	p.ClearMemory()
	assert.Equal(t, 0, p.MemorySize())
}

func TestExtractMessageText_StringContent(t *testing.T) {
	msg := chat.NewUserMessage("Hello world")
	text := extractMessageText(msg)
	assert.Equal(t, "Hello world", text)
}

func TestExtractMessageText_EmptyMessage(t *testing.T) {
	msg := agent.Message{}
	text := extractMessageText(msg)
	assert.Equal(t, "", text)
}

func TestIsOurOutput_True(t *testing.T) {
	msg := agent.Message{
		Name: providerMarker,
	}
	assert.True(t, isOurOutput(msg))
}

func TestIsOurOutput_False(t *testing.T) {
	msg := agent.Message{
		Name: "other-name",
	}
	assert.False(t, isOurOutput(msg))
}

func TestIsOurOutput_EmptyName(t *testing.T) {
	msg := agent.Message{}
	assert.False(t, isOurOutput(msg))
}

func TestDefaultOptions(t *testing.T) {
	opts := defaultOptions()

	assert.Equal(t, DefaultMaxResults, opts.MaxResults)
	assert.Equal(t, DefaultContextPrompt, opts.ContextPrompt)
	assert.Equal(t, DefaultCitationsPrompt, opts.CitationsPrompt)
	assert.Equal(t, BeforeAIInvoke, opts.SearchBehavior)
	assert.Equal(t, DefaultSearchToolName, opts.SearchToolName)
	assert.Equal(t, DefaultSearchToolDescription, opts.SearchToolDescription)
	assert.Nil(t, opts.ResultFormatter)
}

func TestWithNilOption(t *testing.T) {
	searchFn := mockSearchFunc(nil, nil)

	// Should not panic with nil option
	p := New(searchFn, nil, WithMaxResults(5), nil)
	assert.Equal(t, 5, p.options.MaxResults)
}

func TestExtractTextFromMessages_WithMemory(t *testing.T) {
	searchFn := mockSearchFunc(nil, nil)
	p := New(searchFn)

	// Add to memory
	_ = p.Invoked(context.Background(), []agent.Message{
		chat.NewUserMessage("First message"),
	}, nil, nil)

	// Extract from both memory and new messages
	newMessages := []agent.Message{
		chat.NewUserMessage("Second message"),
	}

	text := p.extractTextFromMessages(newMessages)
	assert.Contains(t, text, "First message")
	assert.Contains(t, text, "Second message")
}
