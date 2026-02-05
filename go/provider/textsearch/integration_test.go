// Copyright (c) Microsoft. All rights reserved.

package textsearch_test

import (
	"context"
	"testing"

	"github.com/microsoft/agent-framework-go/agent"
	"github.com/microsoft/agent-framework-go/chat"
	"github.com/microsoft/agent-framework-go/provider/textsearch"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockSearchBackend simulates an external search service.
type mockSearchBackend struct {
	documents map[string]string
}

func newMockSearchBackend() *mockSearchBackend {
	return &mockSearchBackend{
		documents: map[string]string{
			"doc1": "Go is a statically typed, compiled programming language designed at Google.",
			"doc2": "Agents in AI are autonomous entities that can perceive, decide, and act.",
			"doc3": "Context providers inject additional information into agent conversations.",
		},
	}
}

func (m *mockSearchBackend) search(ctx context.Context, query string) ([]textsearch.SearchResult, error) {
	// Simple keyword matching for testing
	var results []textsearch.SearchResult
	for name, content := range m.documents {
		results = append(results, textsearch.SearchResult{
			Name:  name,
			Value: content,
			Link:  "http://docs.example.com/" + name,
		})
	}
	return results, nil
}

func TestIntegration_BeforeAIInvokeMode_InjectsContext(t *testing.T) {
	backend := newMockSearchBackend()
	provider := textsearch.New(backend.search,
		textsearch.WithMaxResults(2),
		textsearch.WithContextPrompt("Reference these documents:"),
		textsearch.WithRecentMessageMemoryLimit(10),
		textsearch.WithRecentMessageRolesIncluded(string(chat.RoleUser), string(chat.RoleAssistant)),
	)

	// Simulate user asking a question
	messages := []agent.Message{
		chat.NewUserMessage("What is Go programming language?"),
	}

	// Provider processes the messages
	ctx := context.Background()
	agentCtx, err := provider.Invoking(ctx, messages)

	require.NoError(t, err)
	require.Len(t, agentCtx.Messages, 1)

	// The injected context should contain search results
	injectedContent := agentCtx.Messages[0].Text()
	assert.Contains(t, injectedContent, "Reference these documents:")

	// After AI responds, provider should track the conversation
	response := []agent.Message{
		chat.NewAssistantMessage("Go is a programming language created at Google."),
	}

	err = provider.Invoked(ctx, messages, response, nil)
	require.NoError(t, err)

	// Memory should contain both user and assistant messages (not the injected context)
	assert.Equal(t, 2, provider.MemorySize())
}

func TestIntegration_OnDemandMode_ProvidesTool(t *testing.T) {
	backend := newMockSearchBackend()
	provider := textsearch.New(backend.search,
		textsearch.WithSearchBehavior(textsearch.OnDemandFunctionCalling),
		textsearch.WithSearchToolName("SearchDocs"),
		textsearch.WithSearchToolDescription("Search internal documentation"),
	)

	messages := []agent.Message{
		chat.NewUserMessage("Find information about agents"),
	}

	ctx := context.Background()
	agentCtx, err := provider.Invoking(ctx, messages)

	require.NoError(t, err)
	require.Len(t, agentCtx.Tools, 1)

	tool := agentCtx.Tools[0]
	assert.Equal(t, "SearchDocs", tool.Name())
	assert.Equal(t, "Search internal documentation", tool.Description())
}

func TestIntegration_StatePersistence_AcrossSessions(t *testing.T) {
	backend := newMockSearchBackend()

	// Session 1: Create provider and have a conversation
	provider1 := textsearch.New(backend.search,
		textsearch.WithRecentMessageMemoryLimit(10),
		textsearch.WithRecentMessageRolesIncluded(string(chat.RoleUser), string(chat.RoleAssistant)),
	)

	messages := []agent.Message{
		chat.NewUserMessage("Hello"),
	}
	response := []agent.Message{
		chat.NewAssistantMessage("Hi there!"),
	}

	err := provider1.Invoked(context.Background(), messages, response, nil)
	require.NoError(t, err)

	// Serialize state
	state, err := provider1.Serialize()
	require.NoError(t, err)

	// Session 2: Create new provider from saved state
	provider2, err := textsearch.NewFromState(backend.search, state,
		textsearch.WithRecentMessageMemoryLimit(10),
		textsearch.WithRecentMessageRolesIncluded(string(chat.RoleUser), string(chat.RoleAssistant)),
	)
	require.NoError(t, err)

	// Provider2 should have the same memory as provider1
	assert.Equal(t, provider1.MemorySize(), provider2.MemorySize())

	// Continue the conversation
	messages2 := []agent.Message{
		chat.NewUserMessage("What can you help me with?"),
	}
	response2 := []agent.Message{
		chat.NewAssistantMessage("I can help with many things!"),
	}

	err = provider2.Invoked(context.Background(), messages2, response2, nil)
	require.NoError(t, err)

	// Should now have 4 messages in memory (2 from each turn)
	assert.Equal(t, 4, provider2.MemorySize())
}

func TestIntegration_CustomFormatter_ChangesOutput(t *testing.T) {
	backend := newMockSearchBackend()

	customFormatter := func(results []textsearch.SearchResult) string {
		if len(results) == 0 {
			return ""
		}
		return "CUSTOM: Found " + string(rune(len(results)+'0')) + " documents"
	}

	provider := textsearch.New(backend.search,
		textsearch.WithResultFormatter(customFormatter),
	)

	messages := []agent.Message{
		chat.NewUserMessage("Search query"),
	}

	ctx := context.Background()
	agentCtx, err := provider.Invoking(ctx, messages)

	require.NoError(t, err)
	require.Len(t, agentCtx.Messages, 1)
	assert.Contains(t, agentCtx.Messages[0].Text(), "CUSTOM:")
}

func TestIntegration_EmptySearchResults_NoContextInjected(t *testing.T) {
	emptySearch := func(ctx context.Context, query string) ([]textsearch.SearchResult, error) {
		return []textsearch.SearchResult{}, nil
	}

	provider := textsearch.New(emptySearch)

	messages := []agent.Message{
		chat.NewUserMessage("Something with no results"),
	}

	ctx := context.Background()
	agentCtx, err := provider.Invoking(ctx, messages)

	require.NoError(t, err)
	assert.Len(t, agentCtx.Messages, 0)
	assert.Len(t, agentCtx.Tools, 0)
}

func TestIntegration_ContextCancellation(t *testing.T) {
	slowSearch := func(ctx context.Context, query string) ([]textsearch.SearchResult, error) {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			return []textsearch.SearchResult{{Value: "result"}}, nil
		}
	}

	provider := textsearch.New(slowSearch)

	// Cancel the context immediately
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	messages := []agent.Message{
		chat.NewUserMessage("Query"),
	}

	agentCtx, err := provider.Invoking(ctx, messages)

	// Should handle cancellation gracefully
	require.NoError(t, err)
	assert.Len(t, agentCtx.Messages, 0)
}

func TestIntegration_MultiTurnConversation(t *testing.T) {
	backend := newMockSearchBackend()
	provider := textsearch.New(backend.search,
		textsearch.WithRecentMessageMemoryLimit(10),
		textsearch.WithRecentMessageRolesIncluded(string(chat.RoleUser), string(chat.RoleAssistant)),
	)

	ctx := context.Background()

	// Turn 1
	messages1 := []agent.Message{
		chat.NewUserMessage("What is Go?"),
	}

	agentCtx1, err := provider.Invoking(ctx, messages1)
	require.NoError(t, err)
	require.Len(t, agentCtx1.Messages, 1) // Search results injected

	response1 := []agent.Message{
		chat.NewAssistantMessage("Go is a programming language."),
	}
	err = provider.Invoked(ctx, messages1, response1, nil)
	require.NoError(t, err)

	// Turn 2 - builds on conversation history
	messages2 := []agent.Message{
		chat.NewUserMessage("Tell me more about it."),
	}

	agentCtx2, err := provider.Invoking(ctx, messages2)
	require.NoError(t, err)
	require.Len(t, agentCtx2.Messages, 1) // New search results

	response2 := []agent.Message{
		chat.NewAssistantMessage("It was designed at Google."),
	}
	err = provider.Invoked(ctx, messages2, response2, nil)
	require.NoError(t, err)

	// Provider should have accumulated memory
	assert.Equal(t, 4, provider.MemorySize())
}
