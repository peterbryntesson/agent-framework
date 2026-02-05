<!-- markdownlint-disable-file -->
# TextSearchProvider Implementation Research

**Date:** 2026-02-03  
**Purpose:** Detailed implementation patterns for TextSearchProvider from .NET to inform Go implementation

---

## 1. Complete Interface of TextSearchProvider

### Base Class: AIContextProvider (Abstractions)

The .NET `AIContextProvider` is an abstract base class with the following interface:

```csharp
public abstract class AIContextProvider
{
    // Called at start of agent invocation to provide additional context
    public abstract ValueTask<AIContext> InvokingAsync(
        InvokingContext context, 
        CancellationToken cancellationToken = default);
    
    // Called at end of invocation (optional override)
    public virtual ValueTask InvokedAsync(
        InvokedContext context, 
        CancellationToken cancellationToken = default) => default;
    
    // Serialize state for persistence (optional override)
    public virtual JsonElement Serialize(
        JsonSerializerOptions? jsonSerializerOptions = null) => default;
    
    // Service locator pattern (optional override)
    public virtual object? GetService(Type serviceType, object? serviceKey = null);
}
```

### TextSearchProvider Class Signature

```csharp
public sealed class TextSearchProvider : AIContextProvider
{
    // Constructor
    public TextSearchProvider(
        Func<string, CancellationToken, Task<IEnumerable<TextSearchResult>>> searchAsync,
        JsonElement serializedState,
        JsonSerializerOptions? jsonSerializerOptions = null,
        TextSearchProviderOptions? options = null,
        ILoggerFactory? loggerFactory = null);
    
    // Overrides
    public override ValueTask<AIContext> InvokingAsync(...);
    public override ValueTask InvokedAsync(...);
    public override JsonElement Serialize(...);
    
    // Nested class
    public sealed class TextSearchResult { ... }
}
```

**Private Fields:**

| Field | Type | Purpose |
|-------|------|---------|
| `_searchAsync` | `Func<string, CancellationToken, Task<IEnumerable<TextSearchResult>>>` | Search delegate |
| `_logger` | `ILogger<TextSearchProvider>?` | Optional logger |
| `_tools` | `AITool[]` | Pre-created search tool for on-demand mode |
| `_recentMessagesText` | `Queue<string>` | Recent message memory |
| `_recentMessageRolesIncluded` | `List<ChatRole>` | Roles to include in memory |
| `_recentMessageMemoryLimit` | `int` | Max messages to retain |
| `_searchTime` | `TextSearchBehavior` | BeforeAIInvoke or OnDemandFunctionCalling |
| `_contextPrompt` | `string` | Prompt prepended to results |
| `_citationsPrompt` | `string` | Prompt appended to results |
| `_contextFormatter` | `Func<IList<TextSearchResult>, string>?` | Custom formatter |

---

## 2. BeforeAIInvoke Mode

### Behavior Flow

1. **Aggregate Input:** Combines memory (recent messages) + current request messages
2. **Execute Search:** Calls the search delegate with combined text
3. **Format Results:** Uses default formatting or custom formatter
4. **Inject Context:** Returns results as a user message in AIContext

### Implementation Details

```csharp
public override async ValueTask<AIContext> InvokingAsync(InvokingContext context, CancellationToken cancellationToken)
{
    if (_searchTime != TextSearchBehavior.BeforeAIInvoke)
    {
        // On-demand mode: return tools only
        return new AIContext { Tools = _tools };
    }

    // Build search input from memory + current request
    var sbInput = new StringBuilder();
    var requestMessagesText = context.RequestMessages
        .Where(x => !string.IsNullOrWhiteSpace(x?.Text))
        .Select(x => x.Text);
    
    foreach (var messageText in _recentMessagesText.Concat(requestMessagesText))
    {
        if (sbInput.Length > 0) sbInput.Append('\n');
        sbInput.Append(messageText);
    }

    try
    {
        // Execute search
        var results = await _searchAsync(sbInput.ToString(), cancellationToken);
        var materialized = results as IList<TextSearchResult> ?? results.ToList();

        if (materialized.Count == 0)
            return new AIContext();

        // Format and return as user message
        string formatted = FormatResults(materialized);
        return new AIContext
        {
            Messages = [new ChatMessage(ChatRole.User, formatted) { 
                AdditionalProperties = new AdditionalPropertiesDictionary() { 
                    ["IsTextSearchProviderOutput"] = true 
                } 
            }]
        };
    }
    catch (Exception ex)
    {
        _logger?.LogError(ex, "TextSearchProvider: Failed to search");
        return new AIContext(); // Graceful degradation
    }
}
```

### Key Points

- **Feedback Loop Prevention:** The injected message is marked with `IsTextSearchProviderOutput = true`
- **Graceful Degradation:** Search errors are logged but don't fail the invocation
- **Empty Results:** Returns empty AIContext when no results found

---

## 3. OnDemandFunctionCalling Mode

### Behavior Flow

1. Does NOT perform automatic search
2. Returns pre-created search tool in `AIContext.Tools`
3. Model decides when to invoke the search function

### Tool Creation (in Constructor)

```csharp
_tools = [
    AIFunctionFactory.Create(
        this.SearchAsync,
        name: options?.FunctionToolName ?? "Search",
        description: options?.FunctionToolDescription ?? 
            "Allows searching for additional information to help answer the user question.")
];
```

### Internal Search Method

```csharp
internal async Task<string> SearchAsync(string userQuestion, CancellationToken cancellationToken)
{
    var results = await _searchAsync(userQuestion, cancellationToken);
    IList<TextSearchResult> materialized = results as IList<TextSearchResult> ?? results.ToList();
    string outputText = FormatResults(materialized);

    _logger?.LogInformation("TextSearchProvider: Retrieved {Count} results.", materialized.Count);
    return outputText;
}
```

---

## 4. TextSearchResult Structure

```csharp
public sealed class TextSearchResult
{
    /// <summary>Display name of the source document (optional)</summary>
    public string? SourceName { get; set; }

    /// <summary>Link/URL to the source document (optional)</summary>
    public string? SourceLink { get; set; }

    /// <summary>Textual content of the retrieved chunk (required)</summary>
    public string Text { get; set; } = string.Empty;

    /// <summary>Raw representation from data source (optional)</summary>
    /// <remarks>
    /// If created from another object model, store the original here.
    /// Useful for debugging or custom formatters that need access to
    /// the underlying object model.
    /// </remarks>
    public object? RawRepresentation { get; set; }
}
```

### Go Equivalent

```go
// TextSearchResult represents a single retrieved search result.
type TextSearchResult struct {
    // SourceName is the display name of the source document (optional).
    SourceName string
    
    // SourceLink is the URL to the source document (optional).
    SourceLink string
    
    // Text is the textual content of the retrieved chunk (required).
    Text string
    
    // RawRepresentation is the raw data from the search provider (optional).
    // Use this for custom formatters that need access to the underlying data.
    RawRepresentation any
}
```

---

## 5. Result Formatting Logic

### Default Formatting

```csharp
private string FormatResults(IList<TextSearchResult> results)
{
    // Custom formatter takes precedence
    if (_contextFormatter is not null)
        return _contextFormatter(results) ?? string.Empty;

    if (results.Count == 0)
        return string.Empty;

    var sb = new StringBuilder();
    sb.AppendLine(_contextPrompt);
    
    for (int i = 0; i < results.Count; i++)
    {
        var result = results[i];
        if (!string.IsNullOrWhiteSpace(result.SourceName))
            sb.AppendLine($"SourceDocName: {result.SourceName}");
        if (!string.IsNullOrWhiteSpace(result.SourceLink))
            sb.AppendLine($"SourceDocLink: {result.SourceLink}");
        sb.AppendLine($"Contents: {result.Text}");
        sb.AppendLine("----");
    }
    
    sb.AppendLine(_citationsPrompt);
    sb.AppendLine();
    return sb.ToString();
}
```

### Default Prompts

```csharp
const string DefaultContextPrompt = 
    "## Additional Context\n" +
    "Consider the following information from source documents when responding to the user:";

const string DefaultCitationsPrompt = 
    "Include citations to the source document with document name and link " +
    "if document name and link is available.";
```

### Example Output

```markdown
## Additional Context
Consider the following information from source documents when responding to the user:
SourceDocName: Doc1
SourceDocLink: http://example.com/doc1
Contents: Content of Doc1
----
SourceDocName: Doc2
SourceDocLink: http://example.com/doc2
Contents: Content of Doc2
----
Include citations to the source document with document name and link if document name and link is available.

```

---

## 6. Memory Management (InvokedAsync)

### Purpose

Maintains recent conversation context for improved multi-turn search relevance.

### Implementation

```csharp
public override ValueTask InvokedAsync(InvokedContext context, CancellationToken cancellationToken)
{
    int limit = _recentMessageMemoryLimit;
    if (limit <= 0)
        return default; // Memory disabled

    if (context.InvokeException is not null)
        return default; // Skip on error

    var messagesText = context.RequestMessages
        .Concat(context.ResponseMessages ?? [])
        .Where(m =>
            _recentMessageRolesIncluded.Contains(m.Role) &&
            !string.IsNullOrWhiteSpace(m.Text) &&
            // Filter out search provider messages to prevent feedback loop
            (m.AdditionalProperties == null || 
             !m.AdditionalProperties.TryGetValue("IsTextSearchProviderOutput", out bool isOutput) || 
             !isOutput))
        .Select(m => m.Text)
        .ToList();

    // If current batch exceeds limit, keep only most recent
    if (messagesText.Count > limit)
        messagesText = messagesText.Skip(messagesText.Count - limit).ToList();

    // Enqueue new messages
    foreach (var message in messagesText)
        _recentMessagesText.Enqueue(message);

    // Dequeue old messages beyond limit
    while (_recentMessagesText.Count > limit)
        _recentMessagesText.Dequeue();

    return default;
}
```

### Key Points

- **Feedback Loop Prevention:** Filters out messages with `IsTextSearchProviderOutput = true`
- **Role Filtering:** Only includes messages from configured roles (default: User only)
- **Queue Behavior:** FIFO queue with configurable limit
- **Error Handling:** Skips memory update on failed invocations

---

## 7. Serialization (State Persistence)

### Implementation

```csharp
public override JsonElement Serialize(JsonSerializerOptions? jsonSerializerOptions = null)
{
    TextSearchProviderState state = new();
    if (_recentMessageMemoryLimit > 0 && _recentMessagesText.Count > 0)
    {
        state.RecentMessagesText = _recentMessagesText
            .Take(_recentMessageMemoryLimit)
            .ToList();
    }
    return JsonSerializer.SerializeToElement(state, AgentJsonUtilities.DefaultOptions.GetTypeInfo(typeof(TextSearchProviderState)));
}

internal sealed class TextSearchProviderState
{
    public List<string>? RecentMessagesText { get; set; }
}
```

### State Restoration (Constructor)

```csharp
// Restore recent messages from serialized state if provided
if (serializedState.ValueKind is JsonValueKind.Null or JsonValueKind.Undefined)
{
    _recentMessagesText = new();
}
else
{
    var state = serializedState.Deserialize<TextSearchProviderState>(...);
    _recentMessagesText = state?.RecentMessagesText is { Count: > 0 }
        ? new Queue<string>(state.RecentMessagesText.Take(_recentMessageMemoryLimit))
        : new Queue<string>();
}
```

---

## 8. TextSearchProviderOptions

```csharp
public sealed class TextSearchProviderOptions
{
    /// <summary>When to execute search (default: BeforeAIInvoke)</summary>
    public TextSearchBehavior SearchTime { get; set; } = TextSearchBehavior.BeforeAIInvoke;
    
    /// <summary>Tool name for OnDemand mode (default: "Search")</summary>
    public string? FunctionToolName { get; set; }
    
    /// <summary>Tool description for OnDemand mode</summary>
    public string? FunctionToolDescription { get; set; }
    
    /// <summary>Prompt prepended to results</summary>
    public string? ContextPrompt { get; set; }
    
    /// <summary>Prompt appended to results</summary>
    public string? CitationsPrompt { get; set; }
    
    /// <summary>Custom formatter (overrides ContextPrompt/CitationsPrompt)</summary>
    public Func<IList<TextSearchResult>, string>? ContextFormatter { get; set; }
    
    /// <summary>Max recent messages to retain (0 = disabled)</summary>
    public int RecentMessageMemoryLimit { get; set; }
    
    /// <summary>Roles to include in memory (default: [User])</summary>
    public List<ChatRole>? RecentMessageRolesIncluded { get; set; }
    
    public enum TextSearchBehavior
    {
        BeforeAIInvoke,
        OnDemandFunctionCalling
    }
}
```

---

## 9. Recommended Go Implementation Pattern

### File Structure

```text
go/agent/
├── text_search_provider.go       # Main implementation
├── text_search_provider_test.go  # Tests
└── text_search_options.go        # Options struct (optional, can be in same file)
```

### Core Types

```go
// TextSearchBehavior determines when search is executed.
type TextSearchBehavior int

const (
    // TextSearchBeforeInvoke executes search before each invocation
    // and injects results as a user message.
    TextSearchBeforeInvoke TextSearchBehavior = iota
    
    // TextSearchOnDemand exposes a search tool for the model to invoke.
    TextSearchOnDemand
)

// TextSearchResult represents a single retrieved search result.
type TextSearchResult struct {
    SourceName        string // Display name of source document
    SourceLink        string // URL to source document
    Text              string // Content of the retrieved chunk
    RawRepresentation any    // Optional raw data for custom formatters
}

// TextSearchFunc is the signature for the search delegate.
type TextSearchFunc func(ctx context.Context, query string) ([]TextSearchResult, error)

// TextSearchResultFormatter customizes how results are formatted.
type TextSearchResultFormatter func(results []TextSearchResult) string
```

### Options Struct

```go
// TextSearchProviderOptions configures TextSearchProvider behavior.
type TextSearchProviderOptions struct {
    // Behavior determines when search is executed.
    // Default: TextSearchBeforeInvoke
    Behavior TextSearchBehavior
    
    // FunctionToolName is the tool name for OnDemand mode.
    // Default: "Search"
    FunctionToolName string
    
    // FunctionToolDescription is the tool description for OnDemand mode.
    FunctionToolDescription string
    
    // ContextPrompt is prepended to formatted results.
    ContextPrompt string
    
    // CitationsPrompt is appended to formatted results.
    CitationsPrompt string
    
    // ContextFormatter customizes result formatting.
    // When set, ContextPrompt and CitationsPrompt are ignored.
    ContextFormatter TextSearchResultFormatter
    
    // RecentMessageMemoryLimit is the max messages to retain (0 = disabled).
    RecentMessageMemoryLimit int
    
    // RecentMessageRoles specifies which roles to include in memory.
    // Default: [RoleUser]
    RecentMessageRoles []chat.Role
}

// DefaultTextSearchProviderOptions returns options with default values.
func DefaultTextSearchProviderOptions() TextSearchProviderOptions {
    return TextSearchProviderOptions{
        Behavior:                TextSearchBeforeInvoke,
        FunctionToolName:        "Search",
        FunctionToolDescription: "Allows searching for additional information to help answer the user question.",
        ContextPrompt:           "## Additional Context\nConsider the following information from source documents when responding to the user:",
        CitationsPrompt:         "Include citations to the source document with document name and link if document name and link is available.",
        RecentMessageRoles:      []chat.Role{chat.RoleUser},
    }
}
```

### Provider Implementation

```go
// TextSearchProvider is a RAG context provider that performs text search
// and injects results into the AI invocation context.
type TextSearchProvider struct {
    BaseContextProvider
    
    searchFunc     TextSearchFunc
    options        TextSearchProviderOptions
    recentMessages []string // Ring buffer
    searchTool     tool.Tool
    mu             sync.Mutex
}

// NewTextSearchProvider creates a new TextSearchProvider.
func NewTextSearchProvider(searchFunc TextSearchFunc, opts *TextSearchProviderOptions) *TextSearchProvider {
    if searchFunc == nil {
        panic("searchFunc cannot be nil")
    }
    
    options := DefaultTextSearchProviderOptions()
    if opts != nil {
        // Merge provided options with defaults
        if opts.Behavior != 0 { options.Behavior = opts.Behavior }
        if opts.FunctionToolName != "" { options.FunctionToolName = opts.FunctionToolName }
        // ... etc
    }
    
    p := &TextSearchProvider{
        searchFunc:     searchFunc,
        options:        options,
        recentMessages: make([]string, 0),
    }
    
    // Create search tool for on-demand mode
    p.searchTool = tool.NewFunction(
        options.FunctionToolName,
        options.FunctionToolDescription,
        p.search,
    )
    
    return p
}

// Invoking implements ContextProvider.
func (p *TextSearchProvider) Invoking(ctx context.Context, messages []Message) (*Context, error) {
    if p.options.Behavior == TextSearchOnDemand {
        return &Context{Tools: []tool.Tool{p.searchTool}}, nil
    }
    
    // Build search input
    var sb strings.Builder
    p.mu.Lock()
    for _, msg := range p.recentMessages {
        if sb.Len() > 0 { sb.WriteByte('\n') }
        sb.WriteString(msg)
    }
    p.mu.Unlock()
    
    for _, msg := range messages {
        text := msg.Text()
        if text == "" { continue }
        if sb.Len() > 0 { sb.WriteByte('\n') }
        sb.WriteString(text)
    }
    
    // Execute search
    results, err := p.searchFunc(ctx, sb.String())
    if err != nil {
        // Log error but don't fail invocation
        return &Context{}, nil
    }
    
    if len(results) == 0 {
        return &Context{}, nil
    }
    
    // Format results
    formatted := p.formatResults(results)
    
    // Return as user message with marker
    return &Context{
        Messages: []chat.Message{
            chat.NewUserMessage(formatted).
                WithMetadata("IsTextSearchProviderOutput", true),
        },
    }, nil
}

// Invoked implements ContextProviderWithLifecycle.
func (p *TextSearchProvider) Invoked(ctx context.Context, request, response []Message, invokeErr error) error {
    limit := p.options.RecentMessageMemoryLimit
    if limit <= 0 || invokeErr != nil {
        return nil
    }
    
    p.mu.Lock()
    defer p.mu.Unlock()
    
    // Collect message text, filtering out search provider output
    for _, msg := range append(request, response...) {
        if !p.shouldIncludeRole(msg.Role()) { continue }
        text := msg.Text()
        if text == "" { continue }
        if isSearchOutput, _ := msg.Metadata("IsTextSearchProviderOutput").(bool); isSearchOutput {
            continue
        }
        p.recentMessages = append(p.recentMessages, text)
    }
    
    // Trim to limit
    if len(p.recentMessages) > limit {
        p.recentMessages = p.recentMessages[len(p.recentMessages)-limit:]
    }
    
    return nil
}

func (p *TextSearchProvider) shouldIncludeRole(role chat.Role) bool {
    for _, r := range p.options.RecentMessageRoles {
        if r == role { return true }
    }
    return false
}

func (p *TextSearchProvider) formatResults(results []TextSearchResult) string {
    if p.options.ContextFormatter != nil {
        return p.options.ContextFormatter(results)
    }
    
    var sb strings.Builder
    sb.WriteString(p.options.ContextPrompt)
    sb.WriteByte('\n')
    
    for _, r := range results {
        if r.SourceName != "" {
            sb.WriteString("SourceDocName: ")
            sb.WriteString(r.SourceName)
            sb.WriteByte('\n')
        }
        if r.SourceLink != "" {
            sb.WriteString("SourceDocLink: ")
            sb.WriteString(r.SourceLink)
            sb.WriteByte('\n')
        }
        sb.WriteString("Contents: ")
        sb.WriteString(r.Text)
        sb.WriteString("\n----\n")
    }
    
    sb.WriteString(p.options.CitationsPrompt)
    sb.WriteByte('\n')
    
    return sb.String()
}

// search is the tool handler for on-demand mode.
func (p *TextSearchProvider) search(ctx context.Context, args struct {
    UserQuestion string `json:"userQuestion" description:"The query to search for"`
}) (string, error) {
    results, err := p.searchFunc(ctx, args.UserQuestion)
    if err != nil {
        return "", err
    }
    return p.formatResults(results), nil
}
```

---

## 10. Test Coverage Requirements

Based on .NET tests, Go implementation should cover:

| Test Case | Description |
|-----------|-------------|
| `TestInvoking_BeforeAIInvoke_InjectsResults` | Formatted results injected as user message |
| `TestInvoking_OnDemand_ExposesSearchTool` | Returns tool with correct name/description |
| `TestInvoking_CustomPrompts` | ContextPrompt and CitationsPrompt override defaults |
| `TestInvoking_CustomFormatter` | ContextFormatter bypasses default formatting |
| `TestInvoking_EmptyResults` | Returns empty context when no results |
| `TestInvoking_SearchError_GracefulDegradation` | Logs error, returns empty context |
| `TestInvoked_MemoryLimit` | Correct queue behavior with limits |
| `TestInvoked_RoleFiltering` | Only includes configured message roles |
| `TestInvoked_FeedbackLoopPrevention` | Excludes provider-injected messages |
| `TestInvoking_RawRepresentation` | Custom formatters can access raw data |
| `TestSerialization_RoundTrip` | State correctly serialized/restored |

---

## Summary

The TextSearchProvider is a RAG-focused context provider with two operating modes:

1. **BeforeAIInvoke:** Automatically searches and injects results before each invocation
2. **OnDemandFunctionCalling:** Exposes a search tool for the model to use when needed

Key implementation considerations for Go:

- Use existing `ContextProviderWithLifecycle` interface
- Thread-safe memory management with `sync.Mutex`
- Graceful error handling (don't fail invocations on search errors)
- Metadata marking to prevent feedback loops
- Support for custom formatters
- State serialization for persistence
