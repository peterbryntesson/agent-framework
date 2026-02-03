// Copyright (c) Microsoft. All rights reserved.

package textsearch

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

// defaultOptions returns options with default values.
func defaultOptions() Options {
	return Options{
		MaxResults:            DefaultMaxResults,
		ContextPrompt:         DefaultContextPrompt,
		CitationsPrompt:       DefaultCitationsPrompt,
		SearchBehavior:        BeforeAIInvoke,
		SearchToolName:        DefaultSearchToolName,
		SearchToolDescription: DefaultSearchToolDescription,
	}
}
