// Copyright (c) Microsoft. All rights reserved.

package textsearch

import (
	"context"
)

// SearchResult represents a single search result from a text search operation.
type SearchResult struct {
	// Name is an optional display name for the result.
	Name string

	// Link is an optional URL to the source document.
	Link string

	// Value is the text content of the result (required).
	Value string

	// Data contains optional raw data for custom formatters.
	Data any
}

// SearchFunc performs a text search and returns results.
//
// The query parameter contains the search query text.
// Returns a slice of SearchResults or an error if the search failed.
type SearchFunc func(ctx context.Context, query string) ([]SearchResult, error)

// ResultFormatter formats search results into a context string.
//
// The results parameter contains the search results to format.
// Returns the formatted string to inject into the agent context.
type ResultFormatter func(results []SearchResult) string
