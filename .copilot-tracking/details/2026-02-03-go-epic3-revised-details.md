<!-- markdownlint-disable-file -->
# Implementation Details: Go Port Epic 3 - Revised Advanced Agent Features

## Context Reference

Sources:
* [2026-02-03-go-epic3-advanced-features-research.md](../../research/2026-02-03-go-epic3-advanced-features-research.md)
* [thread-session-management-research.md](../../subagent/2026-02-03/thread-session-management-research.md)
* [vector-rag-research.md](../../subagent/2026-02-03/vector-rag-research.md)

---

## Implementation Phase 2: SessionStore Interface

<!-- parallelizable: false -->

### Step 2.1: Create hosting package structure

Create a new `hosting` package for server-side infrastructure components.

Files:
* `go/hosting/doc.go` - Package documentation
* `go/hosting/sessionstore.go` - SessionStore interface and implementations

Success criteria:
* Package compiles with no errors
* Package documented with godoc

Context references:
* [dotnet/src/Microsoft.Agents.AI.Hosting/AgentSessionStore.cs](dotnet/src/Microsoft.Agents.AI.Hosting/AgentSessionStore.cs) - .NET pattern

Dependencies:
* None (new package)

#### Implementation: doc.go

```go
// Copyright (c) Microsoft. All rights reserved.

// Package hosting provides infrastructure components for hosted agent scenarios.
//
// This package includes:
//   - SessionStore: Interface for persisting agent sessions across requests
//   - InMemorySessionStore: Thread-safe in-memory session storage
//   - NoopSessionStore: No-op implementation for testing
//
// These components are typically used when building HTTP or gRPC servers
// that expose agents as network services.
package hosting
```

---

### Step 2.2: Define SessionStore interface

Define the core SessionStore interface following .NET patterns.

Files:
* `go/hosting/sessionstore.go` - Interface definition

Success criteria:
* Interface defines SaveSession, GetSession, DeleteSession methods
* Methods accept context.Context for cancellation
* Methods use Agent interface for session creation/deserialization

Context references:
* .NET AgentSessionStore has SaveSessionAsync and GetSessionAsync
* Key format is `{agentID}:{conversationID}`

Dependencies:
* Step 2.1 completion

#### Implementation: SessionStore interface

```go
// Copyright (c) Microsoft. All rights reserved.

package hosting

import (
	"context"
	"encoding/json"
	"sync"

	"github.com/microsoft/agent-framework-go/agent"
)

// SessionStore defines the interface for storing and retrieving agent sessions.
//
// Session stores enable persistent conversations across HTTP requests,
// application restarts, or different service instances in hosted scenarios.
//
// Implementations must be safe for concurrent use from multiple goroutines.
type SessionStore interface {
	// SaveSession persists a session for the given agent and conversation.
	//
	// The session is serialized using session.Serialize() and stored
	// with a key derived from the agent ID and conversation ID.
	//
	// Returns an error if serialization or storage fails.
	SaveSession(ctx context.Context, ag agent.Agent, conversationID string, session agent.Session) error

	// GetSession retrieves a session for the given agent and conversation.
	//
	// If the session exists, it is deserialized using ag.RestoreSession().
	// If the session does not exist, a new session is created using ag.NewSession().
	//
	// Returns the session and any error that occurred.
	GetSession(ctx context.Context, ag agent.Agent, conversationID string) (agent.Session, error)

	// DeleteSession removes a session from storage.
	//
	// Returns nil if the session was deleted or did not exist.
	// Returns an error if the deletion failed.
	DeleteSession(ctx context.Context, ag agent.Agent, conversationID string) error
}

// makeKey computes the storage key from agent ID and conversation ID.
// Format: "{agentID}:{conversationID}" matching .NET convention.
func makeKey(agentID, conversationID string) string {
	return agentID + ":" + conversationID
}
```

---

### Step 2.3: Implement InMemorySessionStore

Implement a thread-safe in-memory session store using sync.Map.

Files:
* `go/hosting/sessionstore.go` - Add InMemorySessionStore

Success criteria:
* Thread-safe using sync.Map
* SaveSession serializes and stores session
* GetSession returns existing or creates new session
* DeleteSession removes session from storage

Context references:
* .NET uses ConcurrentDictionary<string, JsonElement>
* Go equivalent is sync.Map

Dependencies:
* Step 2.2 completion

#### Implementation: InMemorySessionStore

```go
// InMemorySessionStore provides a thread-safe in-memory session store.
//
// This implementation is suitable for single-instance deployments,
// development, and testing. For production multi-instance deployments,
// use a distributed session store implementation.
//
// Sessions are stored as json.RawMessage values, preserving the exact
// serialized form returned by Session.Serialize().
type InMemorySessionStore struct {
	sessions sync.Map // map[string]json.RawMessage
}

// NewInMemorySessionStore creates a new in-memory session store.
func NewInMemorySessionStore() *InMemorySessionStore {
	return &InMemorySessionStore{}
}

// SaveSession serializes and stores the session.
func (s *InMemorySessionStore) SaveSession(
	ctx context.Context,
	ag agent.Agent,
	conversationID string,
	session agent.Session,
) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}

	key := makeKey(ag.ID(), conversationID)

	data, err := session.Serialize()
	if err != nil {
		return err
	}

	s.sessions.Store(key, data)
	return nil
}

// GetSession retrieves an existing session or creates a new one.
func (s *InMemorySessionStore) GetSession(
	ctx context.Context,
	ag agent.Agent,
	conversationID string,
) (agent.Session, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	key := makeKey(ag.ID(), conversationID)

	if data, ok := s.sessions.Load(key); ok {
		rawData, _ := data.(json.RawMessage)
		return ag.RestoreSession(ctx, rawData)
	}

	// Session not found - create new session
	return ag.NewSession(ctx)
}

// DeleteSession removes a session from storage.
func (s *InMemorySessionStore) DeleteSession(
	ctx context.Context,
	ag agent.Agent,
	conversationID string,
) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}

	key := makeKey(ag.ID(), conversationID)
	s.sessions.Delete(key)
	return nil
}

// Compile-time interface check.
var _ SessionStore = (*InMemorySessionStore)(nil)
```

---

### Step 2.4: Implement NoopSessionStore

Implement a no-op session store that never persists sessions.

Files:
* `go/hosting/sessionstore.go` - Add NoopSessionStore

Success criteria:
* SaveSession does nothing and returns nil
* GetSession always creates a new session
* DeleteSession does nothing and returns nil

Context references:
* .NET NoopAgentSessionStore pattern

Dependencies:
* Step 2.2 completion

#### Implementation: NoopSessionStore

```go
// NoopSessionStore is a session store that never persists sessions.
//
// Use this implementation when:
//   - Session persistence is not required
//   - Each request should start with a fresh session
//   - Testing stateless agent behavior
type NoopSessionStore struct{}

// NewNoopSessionStore creates a new no-op session store.
func NewNoopSessionStore() *NoopSessionStore {
	return &NoopSessionStore{}
}

// SaveSession does nothing and returns nil.
func (s *NoopSessionStore) SaveSession(
	ctx context.Context,
	ag agent.Agent,
	conversationID string,
	session agent.Session,
) error {
	return nil
}

// GetSession always creates a new session.
func (s *NoopSessionStore) GetSession(
	ctx context.Context,
	ag agent.Agent,
	conversationID string,
) (agent.Session, error) {
	return ag.NewSession(ctx)
}

// DeleteSession does nothing and returns nil.
func (s *NoopSessionStore) DeleteSession(
	ctx context.Context,
	ag agent.Agent,
	conversationID string,
) error {
	return nil
}

// Compile-time interface check.
var _ SessionStore = (*NoopSessionStore)(nil)
```

---

### Step 2.5: Update Agent interface for session deserialization

Verify Agent interface supports RestoreSession method required by SessionStore.

Files:
* `go/agent/agent.go` - Verify RestoreSession exists

Success criteria:
* Agent interface includes RestoreSession method
* Method signature matches SessionStore usage

Context references:
* Existing Agent interface in [go/agent/agent.go](go/agent/agent.go)

Dependencies:
* None (verification step)

#### Verification

The existing Agent interface should already have:

```go
type Agent interface {
    // ... existing methods ...
    
    // NewSession creates a new session for this agent.
    NewSession(ctx context.Context) (Session, error)
    
    // RestoreSession restores a session from serialized data.
    RestoreSession(ctx context.Context, data json.RawMessage) (Session, error)
}
```

If RestoreSession is missing, add it. The chatagent.Agent already implements session restoration.

---

### Step 2.6: Write unit tests for SessionStore

Create comprehensive tests for SessionStore implementations.

Files:
* `go/hosting/sessionstore_test.go` - Unit tests

Success criteria:
* Test SaveSession and GetSession round-trip
* Test GetSession creates new session when not found
* Test DeleteSession removes session
* Test context cancellation handling
* Test concurrent access safety
* Achieve 90%+ coverage

Context references:
* Go testing patterns in existing test files

Dependencies:
* Steps 2.3 and 2.4 completion

#### Test Cases

```go
// Copyright (c) Microsoft. All rights reserved.

package hosting

import (
	"context"
	"encoding/json"
	"sync"
	"testing"

	"github.com/microsoft/agent-framework-go/agent"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockAgent implements agent.Agent for testing.
type mockAgent struct {
	id           string
	newSessionFn func() (agent.Session, error)
	restoreFn    func(data json.RawMessage) (agent.Session, error)
}

func (m *mockAgent) ID() string { return m.id }
func (m *mockAgent) Name() string { return "test-agent" }
func (m *mockAgent) Description() string { return "" }
func (m *mockAgent) Metadata() agent.AIAgentMetadata { return agent.AIAgentMetadata{} }
func (m *mockAgent) Run(ctx context.Context, messages []agent.Message, opts ...agent.RunOption) (*agent.Response, error) {
	return nil, nil
}
func (m *mockAgent) RunStream(ctx context.Context, messages []agent.Message, opts ...agent.RunOption) (<-chan agent.ResponseUpdate, error) {
	return nil, nil
}
func (m *mockAgent) NewSession(ctx context.Context) (agent.Session, error) {
	return m.newSessionFn()
}
func (m *mockAgent) RestoreSession(ctx context.Context, data json.RawMessage) (agent.Session, error) {
	return m.restoreFn(data)
}
func (m *mockAgent) GetService(serviceType reflect.Type) interface{} { return nil }

// mockSession implements agent.Session for testing.
type mockSession struct {
	id   string
	data json.RawMessage
}

func (s *mockSession) ID() string { return s.id }
func (s *mockSession) Messages() []agent.Message { return nil }
func (s *mockSession) AddMessage(msg agent.Message) {}
func (s *mockSession) Serialize() (json.RawMessage, error) { return s.data, nil }
func (s *mockSession) GetService(serviceType reflect.Type) interface{} { return nil }

func TestInMemorySessionStore_SaveAndGet(t *testing.T) {
	store := NewInMemorySessionStore()
	ctx := context.Background()

	session := &mockSession{id: "session-1", data: json.RawMessage(`{"id":"session-1"}`)}
	ag := &mockAgent{
		id: "agent-1",
		restoreFn: func(data json.RawMessage) (agent.Session, error) {
			return &mockSession{id: "session-1", data: data}, nil
		},
	}

	// Save session
	err := store.SaveSession(ctx, ag, "conv-1", session)
	require.NoError(t, err)

	// Get session
	retrieved, err := store.GetSession(ctx, ag, "conv-1")
	require.NoError(t, err)
	assert.Equal(t, "session-1", retrieved.ID())
}

func TestInMemorySessionStore_GetCreatesNewWhenNotFound(t *testing.T) {
	store := NewInMemorySessionStore()
	ctx := context.Background()

	ag := &mockAgent{
		id: "agent-1",
		newSessionFn: func() (agent.Session, error) {
			return &mockSession{id: "new-session"}, nil
		},
	}

	session, err := store.GetSession(ctx, ag, "unknown-conv")
	require.NoError(t, err)
	assert.Equal(t, "new-session", session.ID())
}

func TestInMemorySessionStore_Delete(t *testing.T) {
	store := NewInMemorySessionStore()
	ctx := context.Background()

	session := &mockSession{id: "session-1", data: json.RawMessage(`{}`)}
	ag := &mockAgent{
		id: "agent-1",
		newSessionFn: func() (agent.Session, error) {
			return &mockSession{id: "new-session"}, nil
		},
	}

	// Save and then delete
	_ = store.SaveSession(ctx, ag, "conv-1", session)
	err := store.DeleteSession(ctx, ag, "conv-1")
	require.NoError(t, err)

	// Get should create new session
	retrieved, err := store.GetSession(ctx, ag, "conv-1")
	require.NoError(t, err)
	assert.Equal(t, "new-session", retrieved.ID())
}

func TestInMemorySessionStore_ConcurrentAccess(t *testing.T) {
	store := NewInMemorySessionStore()
	ctx := context.Background()

	ag := &mockAgent{
		id: "agent-1",
		newSessionFn: func() (agent.Session, error) {
			return &mockSession{id: "new"}, nil
		},
		restoreFn: func(data json.RawMessage) (agent.Session, error) {
			return &mockSession{id: "restored", data: data}, nil
		},
	}

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			session := &mockSession{id: "s", data: json.RawMessage(`{}`)}
			_ = store.SaveSession(ctx, ag, "conv", session)
			_, _ = store.GetSession(ctx, ag, "conv")
		}(i)
	}
	wg.Wait()
}

func TestNoopSessionStore_AlwaysCreatesNew(t *testing.T) {
	store := NewNoopSessionStore()
	ctx := context.Background()

	callCount := 0
	ag := &mockAgent{
		id: "agent-1",
		newSessionFn: func() (agent.Session, error) {
			callCount++
			return &mockSession{id: "new"}, nil
		},
	}

	// Save does nothing
	session := &mockSession{id: "saved", data: json.RawMessage(`{}`)}
	err := store.SaveSession(ctx, ag, "conv-1", session)
	require.NoError(t, err)

	// Get always creates new
	_, err = store.GetSession(ctx, ag, "conv-1")
	require.NoError(t, err)
	assert.Equal(t, 1, callCount)
}
```

---

## Implementation Phase 3: TextSearchProvider

<!-- parallelizable: false -->

### Step 3.1: Define SearchResult and SearchFunc types

Create the core types for the text search provider.

Files:
* `go/provider/textsearch/types.go` - Type definitions

Success criteria:
* SearchResult struct with Name, Link, Value, Data fields
* SearchFunc type for pluggable search backends
* ResultFormatter type for custom result formatting

Context references:
* .NET TextSearchResult with Name, Link, Value, Data properties

Dependencies:
* None

#### Implementation: types.go

```go
// Copyright (c) Microsoft. All rights reserved.

// Package textsearch provides a RAG context provider with pluggable search backends.
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
```

---

### Step 3.2: Define TextSearchProviderOptions

Define configuration options for the text search provider.

Files:
* `go/provider/textsearch/options.go` - Options and functional options

Success criteria:
* SearchBehavior enum with BeforeAIInvoke and OnDemandFunctionCalling
* TextSearchProviderOptions struct with all configuration fields
* Functional options pattern for configuration

Context references:
* .NET TextSearchProviderOptions with SearchBehavior enum

Dependencies:
* Step 3.1 completion

#### Implementation: options.go

```go
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
```

---

### Step 3.3: Implement TextSearchProvider core

Implement the core provider structure and constructor.

Files:
* `go/provider/textsearch/provider.go` - Provider implementation

Success criteria:
* Provider implements ContextProviderWithLifecycle
* Constructor accepts SearchFunc and Options
* Provider maintains memory of recent messages
* Thread-safe implementation

Context references:
* .NET TextSearchProvider implementation pattern

Dependencies:
* Steps 3.1 and 3.2 completion

#### Implementation: provider.go (core structure)

```go
// Copyright (c) Microsoft. All rights reserved.

package textsearch

import (
	"context"
	"encoding/json"
	"strings"
	"sync"

	"github.com/microsoft/agent-framework-go/agent"
	"github.com/microsoft/agent-framework-go/chat"
	"github.com/microsoft/agent-framework-go/tool"
)

// Provider implements RAG (Retrieval Augmented Generation) by searching
// for relevant context before or during agent invocations.
//
// It supports two modes:
//   - BeforeAIInvoke: Automatically search and inject context before each invocation
//   - OnDemandFunctionCalling: Expose search as a tool the model can invoke
//
// The provider tracks conversation history to build effective search queries
// from the full context of the conversation.
type Provider struct {
	agent.BaseContextProvider

	search  SearchFunc
	options Options

	// Memory of recent messages for context building
	memory []agent.Message
	mu     sync.Mutex

	// Pre-built search tool for OnDemandFunctionCalling mode
	searchTool tool.Tool
}

// New creates a new text search provider.
//
// The search function is called to retrieve relevant documents or information.
// Configure behavior using Option functions.
func New(search SearchFunc, opts ...Option) *Provider {
	options := defaultOptions()
	for _, opt := range opts {
		if opt != nil {
			opt(&options)
		}
	}

	p := &Provider{
		search:  search,
		options: options,
		memory:  make([]agent.Message, 0),
	}

	// Pre-build search tool for on-demand mode
	if options.SearchBehavior == OnDemandFunctionCalling {
		p.searchTool = p.buildSearchTool()
	}

	return p
}

// buildSearchTool creates the search function tool.
func (p *Provider) buildSearchTool() tool.Tool {
	return tool.NewFunctionTool(
		p.options.SearchToolName,
		p.options.SearchToolDescription,
		func(ctx context.Context, args struct {
			Query string `json:"query" description:"The search query"`
		}) (string, error) {
			results, err := p.search(ctx, args.Query)
			if err != nil {
				return "", err
			}
			return p.formatResults(results), nil
		},
	)
}

// formatResults formats search results into a string.
func (p *Provider) formatResults(results []SearchResult) string {
	if p.options.ResultFormatter != nil {
		return p.options.ResultFormatter(results)
	}
	return p.defaultFormatResults(results)
}

// defaultFormatResults provides the default result formatting.
func (p *Provider) defaultFormatResults(results []SearchResult) string {
	if len(results) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString(p.options.ContextPrompt)
	sb.WriteString("\n\n")

	for i, result := range results {
		if i > 0 {
			sb.WriteString("\n----\n")
		}

		if result.Name != "" {
			sb.WriteString("**")
			sb.WriteString(result.Name)
			sb.WriteString("**\n")
		}

		if result.Link != "" {
			sb.WriteString("Source: ")
			sb.WriteString(result.Link)
			sb.WriteString("\n")
		}

		sb.WriteString(result.Value)
	}

	if p.options.CitationsPrompt != "" {
		sb.WriteString("\n\n")
		sb.WriteString(p.options.CitationsPrompt)
	}

	return sb.String()
}

// extractTextFromMessages builds a query string from messages.
func (p *Provider) extractTextFromMessages(messages []agent.Message) string {
	var sb strings.Builder

	// Include memory first
	for _, msg := range p.memory {
		if text := extractMessageText(msg); text != "" {
			sb.WriteString(text)
			sb.WriteString(" ")
		}
	}

	// Include current messages
	for _, msg := range messages {
		if text := extractMessageText(msg); text != "" {
			sb.WriteString(text)
			sb.WriteString(" ")
		}
	}

	return strings.TrimSpace(sb.String())
}

// extractMessageText extracts text content from a message.
func extractMessageText(msg agent.Message) string {
	// Handle text content from message
	if textContent, ok := msg.Content.(string); ok {
		return textContent
	}
	// Handle content parts if needed
	return ""
}
```

---

### Step 3.4: Implement BeforeAIInvoke behavior

Implement the Invoking method for automatic context injection.

Files:
* `go/provider/textsearch/provider.go` - Add Invoking method

Success criteria:
* Extracts query from memory + current messages
* Calls search function with query
* Formats results and returns as context message
* Handles errors gracefully (logs but doesn't fail)

Context references:
* .NET InvokingAsync implementation

Dependencies:
* Step 3.3 completion

#### Implementation: Invoking method

```go
// Invoking is called before each agent invocation.
//
// In BeforeAIInvoke mode, this method performs a search and injects
// results as a context message.
//
// In OnDemandFunctionCalling mode, this method returns the search tool.
func (p *Provider) Invoking(ctx context.Context, messages []agent.Message) (*agent.Context, error) {
	switch p.options.SearchBehavior {
	case BeforeAIInvoke:
		return p.invokingBeforeAI(ctx, messages)
	case OnDemandFunctionCalling:
		return p.invokingOnDemand(ctx, messages)
	default:
		return &agent.Context{}, nil
	}
}

// invokingBeforeAI implements automatic search and context injection.
func (p *Provider) invokingBeforeAI(ctx context.Context, messages []agent.Message) (*agent.Context, error) {
	// Build query from memory and current messages
	query := p.extractTextFromMessages(messages)
	if query == "" {
		return &agent.Context{}, nil
	}

	// Perform search
	results, err := p.search(ctx, query)
	if err != nil {
		// Log error but don't fail the invocation
		// In production, use a proper logger
		return &agent.Context{}, nil
	}

	if len(results) == 0 {
		return &agent.Context{}, nil
	}

	// Limit results
	if len(results) > p.options.MaxResults {
		results = results[:p.options.MaxResults]
	}

	// Format results
	formattedResults := p.formatResults(results)
	if formattedResults == "" {
		return &agent.Context{}, nil
	}

	// Create context message with metadata to prevent feedback loop
	contextMessage := agent.Message{
		Role:    chat.RoleUser,
		Content: formattedResults,
		Metadata: map[string]any{
			"isTextSearchProviderOutput": true,
		},
	}

	return &agent.Context{
		Messages: []chat.Message{contextMessage},
	}, nil
}
```

---

### Step 3.5: Implement OnDemandFunctionCalling behavior

Implement the on-demand search tool behavior.

Files:
* `go/provider/textsearch/provider.go` - Add on-demand methods

Success criteria:
* Returns search tool in Context.Tools
* Tool invocation triggers search
* Results formatted consistently with BeforeAIInvoke mode

Context references:
* .NET OnDemandFunctionCalling implementation

Dependencies:
* Step 3.4 completion

#### Implementation: OnDemand methods

```go
// invokingOnDemand returns the search tool for on-demand invocation.
func (p *Provider) invokingOnDemand(ctx context.Context, messages []agent.Message) (*agent.Context, error) {
	return &agent.Context{
		Tools: []tool.Tool{p.searchTool},
	}, nil
}
```

---

### Step 3.6: Implement state serialization

Implement the Invoked method for memory tracking and state serialization.

Files:
* `go/provider/textsearch/provider.go` - Add Invoked and serialization

Success criteria:
* Invoked updates memory with request/response messages
* State can be serialized to JSON
* State can be restored from JSON

Context references:
* .NET Serialize/Deserialize pattern

Dependencies:
* Step 3.5 completion

#### Implementation: State management

```go
// Invoked is called after each agent invocation.
//
// This method updates the provider's memory with the conversation
// to build better search queries for future invocations.
func (p *Provider) Invoked(
	ctx context.Context,
	request []agent.Message,
	response []agent.Message,
	invokeErr error,
) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Filter out our own output messages to prevent feedback loop
	for _, msg := range request {
		if !isOurOutput(msg) {
			p.memory = append(p.memory, msg)
		}
	}

	if response != nil {
		p.memory = append(p.memory, response...)
	}

	return nil
}

// isOurOutput checks if a message was generated by this provider.
func isOurOutput(msg agent.Message) bool {
	if msg.Metadata == nil {
		return false
	}
	if isOutput, ok := msg.Metadata["isTextSearchProviderOutput"].(bool); ok {
		return isOutput
	}
	return false
}

// State represents the serializable state of the provider.
type State struct {
	Memory []agent.Message `json:"memory"`
}

// Serialize returns the provider state as JSON.
func (p *Provider) Serialize() (json.RawMessage, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	state := State{
		Memory: p.memory,
	}

	return json.Marshal(state)
}

// Restore restores the provider state from JSON.
func (p *Provider) Restore(data json.RawMessage) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	var state State
	if err := json.Unmarshal(data, &state); err != nil {
		return err
	}

	p.memory = state.Memory
	return nil
}

// NewFromState creates a provider from serialized state.
func NewFromState(search SearchFunc, state json.RawMessage, opts ...Option) (*Provider, error) {
	p := New(search, opts...)
	if err := p.Restore(state); err != nil {
		return nil, err
	}
	return p, nil
}

// Compile-time interface check.
var _ agent.ContextProviderWithLifecycle = (*Provider)(nil)
```

---

### Step 3.7: Write unit tests for TextSearchProvider

Create comprehensive tests for the text search provider.

Files:
* `go/provider/textsearch/provider_test.go` - Unit tests

Success criteria:
* Test BeforeAIInvoke mode
* Test OnDemandFunctionCalling mode
* Test result formatting
* Test memory accumulation
* Test state serialization/restoration
* Achieve 90%+ coverage

Context references:
* Go testing patterns

Dependencies:
* Step 3.6 completion

#### Test Cases Overview

```go
// Test: NewProvider creates provider with defaults
// Test: NewProvider applies options
// Test: Invoking_BeforeAIInvoke_SearchesAndInjectsContext
// Test: Invoking_BeforeAIInvoke_HandlesEmptyQuery
// Test: Invoking_BeforeAIInvoke_HandlesSearchError
// Test: Invoking_BeforeAIInvoke_LimitsResults
// Test: Invoking_OnDemand_ReturnsTool
// Test: Invoked_UpdatesMemory
// Test: Invoked_FiltersOwnOutput
// Test: FormatResults_Default
// Test: FormatResults_Custom
// Test: Serialize_Restore_RoundTrip
// Test: NewFromState_RestoresProvider
```

---

### Step 3.8: Write integration tests with ChatClientAgent

Create integration tests showing end-to-end RAG flow.

Files:
* `go/provider/textsearch/integration_test.go` - Integration tests

Success criteria:
* Test provider with mock ChatClientAgent
* Verify context injection in BeforeAIInvoke mode
* Verify tool availability in OnDemand mode

Context references:
* Existing chatagent tests

Dependencies:
* Step 3.7 completion

---

## Implementation Phase 4: Final Validation

<!-- parallelizable: false -->

### Step 4.1: Run full project validation

Execute all validation commands:

```bash
cd go
go build ./...
go test -cover ./...
go vet ./...
golangci-lint run
```

### Step 4.2: Fix minor validation issues

Address any issues found during validation.

### Step 4.3: Update package documentation

Add godoc comments to all public types and update README.md.

### Step 4.4: Report blocking issues

Document any issues requiring additional research or planning.

---

## Dependencies

* Go 1.22+ with generics support
* Existing packages: `agent`, `chatagent`, `chat`, `tool`
* Standard library: `sync`, `encoding/json`, `context`, `strings`
* Testing: `github.com/stretchr/testify`

## Success Criteria

* SessionStore interface with SaveSession, GetSession, DeleteSession
* InMemorySessionStore with thread-safe sync.Map storage
* NoopSessionStore for stateless scenarios
* TextSearchProvider implementing ContextProviderWithLifecycle
* BeforeAIInvoke mode with automatic context injection
* OnDemandFunctionCalling mode with search tool
* State serialization and restoration
* 90%+ test coverage
* All public APIs documented with godoc
