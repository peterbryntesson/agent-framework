// Copyright (c) Microsoft. All rights reserved.

// Package textsearch provides a RAG (Retrieval Augmented Generation) context provider
// with pluggable search backends.
//
// The Provider implements agent.ContextProviderWithLifecycle and can operate in two modes:
//   - BeforeAIInvoke: Automatically search and inject context before each agent invocation
//   - OnDemandFunctionCalling: Expose search as a tool the model can invoke on demand
//
// Example usage with BeforeAIInvoke mode:
//
//	searchFn := func(ctx context.Context, query string) ([]textsearch.SearchResult, error) {
//	    // Call your search backend (Azure AI Search, Elasticsearch, etc.)
//	    return results, nil
//	}
//
//	provider := textsearch.New(searchFn,
//	    textsearch.WithMaxResults(5),
//	    textsearch.WithSearchBehavior(textsearch.BeforeAIInvoke),
//	)
//
//	agent := chatagent.New(client,
//	    chatagent.WithContextProvider(provider),
//	)
//
// Example usage with OnDemandFunctionCalling mode:
//
//	provider := textsearch.New(searchFn,
//	    textsearch.WithSearchBehavior(textsearch.OnDemandFunctionCalling),
//	    textsearch.WithSearchToolName("SearchKnowledgeBase"),
//	)
package textsearch
