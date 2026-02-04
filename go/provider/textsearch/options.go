// Copyright (c) Microsoft. All rights reserved.

package textsearch

import (
	"github.com/microsoft/agent-framework-go/chat"
)

// DefaultRecentMessageRolesIncluded is the default list of roles to include in memory.
var DefaultRecentMessageRolesIncluded = []string{string(chat.RoleUser)}

// SearchBehavior controls when search is performed.
type SearchBehavior int

const (
	// BeforeAIInvoke automatically searches before each agent invocation
	// and injects results as context messages.
	BeforeAIInvoke SearchBehavior = iota

	// OnDemandFunctionCalling exposes search as a tool that the model
	// can invoke when it needs additional information.
	OnDemandFunctionCalling
)

// DefaultContextPrompt is the default prompt prepended to search results.
const DefaultContextPrompt = "## Related Information\nConsider the following information when answering:"

// DefaultCitationsPrompt is the default prompt appended after search results.
const DefaultCitationsPrompt = "When using the above information, cite sources by name or link when available."

// DefaultMaxResults is the default maximum number of search results.
const DefaultMaxResults = 3

// DefaultSearchToolName is the default name for the search tool.
const DefaultSearchToolName = "Search"

// DefaultSearchToolDescription is the default description for the search tool.
const DefaultSearchToolDescription = "Search for relevant information to help answer user questions."

// Options configures the text search provider.
type Options struct {
	// MaxResults limits the number of search results to include.
	// Default: 3
	MaxResults int

	// ContextPrompt is prepended before search results.
	// Default: DefaultContextPrompt
	ContextPrompt string

	// CitationsPrompt is appended after search results.
	// Default: DefaultCitationsPrompt
	CitationsPrompt string

	// SearchBehavior controls when search is performed.
	// Default: BeforeAIInvoke
	SearchBehavior SearchBehavior

	// SearchToolName is the name of the search tool (OnDemandFunctionCalling only).
	// Default: "Search"
	SearchToolName string

	// SearchToolDescription is the description for the search tool.
	// Default: DefaultSearchToolDescription
	SearchToolDescription string

	// ResultFormatter customizes how results are formatted.
	// If nil, the default formatter is used.
	ResultFormatter ResultFormatter

	// RecentMessageMemoryLimit sets the maximum number of recent messages to
	// retain in memory for context building. When set to 0 (default), memory
	// is disabled and only current request messages are used for search input.
	// The value is a count of individual messages, not turns.
	RecentMessageMemoryLimit int

	// RecentMessageRolesIncluded filters which message roles are included
	// when retaining recent messages in memory. This allows you to control
	// whether to include only user messages, only assistant messages, or both.
	// Default: []chat.Role{chat.RoleUser}
	//
	// Be careful when including assistant messages, as they may skew search
	// results towards information already provided rather than focusing on
	// the user's current needs.
	RecentMessageRolesIncluded []string
}

// Option is a functional option for configuring the provider.
type Option func(*Options)

// WithMaxResults sets the maximum number of search results.
func WithMaxResults(max int) Option {
	return func(o *Options) {
		o.MaxResults = max
	}
}

// WithContextPrompt sets the context prompt.
func WithContextPrompt(prompt string) Option {
	return func(o *Options) {
		o.ContextPrompt = prompt
	}
}

// WithCitationsPrompt sets the citations prompt.
func WithCitationsPrompt(prompt string) Option {
	return func(o *Options) {
		o.CitationsPrompt = prompt
	}
}

// WithSearchBehavior sets the search behavior.
func WithSearchBehavior(behavior SearchBehavior) Option {
	return func(o *Options) {
		o.SearchBehavior = behavior
	}
}

// WithSearchToolName sets the search tool name.
func WithSearchToolName(name string) Option {
	return func(o *Options) {
		o.SearchToolName = name
	}
}

// WithSearchToolDescription sets the search tool description.
func WithSearchToolDescription(desc string) Option {
	return func(o *Options) {
		o.SearchToolDescription = desc
	}
}

// WithResultFormatter sets a custom result formatter.
func WithResultFormatter(formatter ResultFormatter) Option {
	return func(o *Options) {
		o.ResultFormatter = formatter
	}
}

// WithRecentMessageMemoryLimit sets the maximum number of recent messages
// to retain in memory for context building. Set to 0 to disable memory
// (only current request messages will be used for search).
func WithRecentMessageMemoryLimit(limit int) Option {
	return func(o *Options) {
		if limit < 0 {
			limit = 0
		}
		o.RecentMessageMemoryLimit = limit
	}
}

// WithRecentMessageRolesIncluded sets the list of message roles to include
// when retaining recent messages in memory. Use chat.RoleUser, chat.RoleAssistant, etc.
// Defaults to only user messages when not specified.
func WithRecentMessageRolesIncluded(roles ...string) Option {
	return func(o *Options) {
		o.RecentMessageRolesIncluded = roles
	}
}

// defaultOptions returns options with default values.
func defaultOptions() Options {
	return Options{
		MaxResults:                 DefaultMaxResults,
		ContextPrompt:              DefaultContextPrompt,
		CitationsPrompt:            DefaultCitationsPrompt,
		SearchBehavior:             BeforeAIInvoke,
		SearchToolName:             DefaultSearchToolName,
		SearchToolDescription:      DefaultSearchToolDescription,
		RecentMessageMemoryLimit:   0, // Disabled by default
		RecentMessageRolesIncluded: DefaultRecentMessageRolesIncluded,
	}
}
