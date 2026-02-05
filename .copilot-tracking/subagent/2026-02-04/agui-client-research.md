# AG-UI Client Research for Go Implementation

**Date**: 2026-02-04  
**Status**: Research Complete  
**Purpose**: Analyze .NET and Python AG-UI Client implementations to inform Go client design

## Executive Summary

The AG-UI Client is a chat client that communicates with AG-UI protocol-compliant servers. It:

1. Sends requests with messages, tools, and state
2. Parses Server-Sent Events (SSE) streams
3. Converts AG-UI events to framework-native response updates
4. Handles hybrid tool execution (client + server tools)
5. Manages thread/conversation continuity

## 1. Method Signatures

### .NET AGUIChatClient

```csharp
// Core client implementation
public sealed class AGUIChatClient : DelegatingChatClient
{
    // Constructor
    public AGUIChatClient(
        HttpClient httpClient,
        string endpoint,
        ILoggerFactory? loggerFactory = null,
        JsonSerializerOptions? jsonSerializerOptions = null,
        IServiceProvider? serviceProvider = null);

    // Non-streaming response (calls streaming internally)
    public override Task<ChatResponse> GetResponseAsync(
        IEnumerable<ChatMessage> messages, 
        ChatOptions? options = null, 
        CancellationToken cancellationToken = default);

    // Streaming response - main implementation
    public override IAsyncEnumerable<ChatResponseUpdate> GetStreamingResponseAsync(
        IEnumerable<ChatMessage> messages,
        ChatOptions? options = null,
        CancellationToken cancellationToken = default);
}

// Internal handler
internal sealed class AGUIChatClientHandler : IChatClient
{
    // Metadata
    public ChatClientMetadata Metadata { get; }
    
    // Both methods delegate to streaming
    public Task<ChatResponse> GetResponseAsync(...);
    public IAsyncEnumerable<ChatResponseUpdate> GetStreamingResponseAsync(...);
}

// HTTP service
internal sealed class AGUIHttpService
{
    public IAsyncEnumerable<BaseEvent> PostRunAsync(
        RunAgentInput input,
        CancellationToken cancellationToken);
}
```

### Python AGUIChatClient

```python
class AGUIChatClient(BaseChatClient[TAGUIChatOptions], Generic[TAGUIChatOptions]):
    """Chat client for AG-UI compliant servers."""
    
    OTEL_PROVIDER_NAME = "agui"
    
    def __init__(
        self,
        *,
        endpoint: str,
        http_client: httpx.AsyncClient | None = None,
        timeout: float = 60.0,
        additional_properties: dict[str, Any] | None = None,
        **kwargs: Any,
    ) -> None: ...
    
    async def close(self) -> None: ...
    
    async def __aenter__(self) -> Self: ...
    
    async def __aexit__(self, *args: Any) -> None: ...
    
    # Internal methods (BaseChatClient interface)
    async def _inner_get_response(
        self,
        *,
        messages: MutableSequence[ChatMessage],
        options: dict[str, Any],
        **kwargs: Any,
    ) -> ChatResponse: ...
    
    async def _inner_get_streaming_response(
        self,
        *,
        messages: MutableSequence[ChatMessage],
        options: dict[str, Any],
        **kwargs: Any,
    ) -> AsyncIterable[ChatResponseUpdate]: ...

# HTTP service
class AGUIHttpService:
    def __init__(
        self,
        endpoint: str,
        http_client: httpx.AsyncClient | None = None,
        timeout: float = 60.0,
    ) -> None: ...
    
    async def post_run(
        self,
        thread_id: str,
        run_id: str,
        messages: list[dict[str, Any]],
        state: dict[str, Any] | None = None,
        tools: list[dict[str, Any]] | None = None,
    ) -> AsyncIterable[dict[str, Any]]: ...
    
    async def close(self) -> None: ...

# Event converter
class AGUIEventConverter:
    def __init__(self) -> None: ...
    
    def convert_event(self, event: dict[str, Any]) -> ChatResponseUpdate | None: ...
```

## 2. SSE Parsing Approach

### .NET Approach

Uses `System.Net.ServerSentEvents.SseParser`:

```csharp
// In AGUIHttpService.PostRunAsync
Stream responseStream = await response.Content.ReadAsStreamAsync(cancellationToken);
var items = SseParser.Create(responseStream, ItemParser).EnumerateAsync(cancellationToken);
await foreach (var sseItem in items)
{
    yield return sseItem.Data;
}

private static BaseEvent ItemParser(string type, ReadOnlySpan<byte> data)
{
    return JsonSerializer.Deserialize(data, AGUIJsonSerializerContext.Default.BaseEvent);
}
```

### Python Approach

Manual line-by-line parsing with httpx streaming:

```python
async with self.http_client.stream(
    "POST",
    self.endpoint,
    json=request_data,
    headers={"Accept": "text/event-stream"},
) as response:
    response.raise_for_status()
    
    async for line in response.aiter_lines():
        if line.startswith("data: "):
            data = line[6:]  # Remove "data: " prefix
            try:
                event = json.loads(data)
                yield event
            except json.JSONDecodeError:
                continue  # Skip malformed events
```

### SSE Format

The server sends events in this format:
```
event: RUN_STARTED
data: {"type":"RUN_STARTED","threadId":"t1","runId":"r1"}

event: TEXT_MESSAGE_START
data: {"type":"TEXT_MESSAGE_START","messageId":"m1","role":"assistant"}

event: TEXT_MESSAGE_CONTENT
data: {"type":"TEXT_MESSAGE_CONTENT","messageId":"m1","delta":"Hello"}
```

### Recommended Go Approach

Use `bufio.Scanner` or a dedicated SSE library:

```go
func (s *AGUIHttpService) PostRun(
    ctx context.Context,
    input RunAgentInput,
) (<-chan Event, error) {
    events := make(chan Event)
    
    go func() {
        defer close(events)
        
        resp, err := s.client.Do(req)
        if err != nil {
            events <- NewRunErrorEvent(err.Error())
            return
        }
        defer resp.Body.Close()
        
        scanner := bufio.NewScanner(resp.Body)
        for scanner.Scan() {
            line := scanner.Text()
            if strings.HasPrefix(line, "data: ") {
                data := line[6:]
                event, err := ParseEvent([]byte(data))
                if err != nil {
                    continue
                }
                select {
                case events <- event:
                case <-ctx.Done():
                    return
                }
            }
        }
    }()
    
    return events, nil
}
```

## 3. Event-to-Message Conversion Logic

### Event Types and Mapping

| AG-UI Event | .NET Output | Python Output |
|-------------|-------------|---------------|
| `RUN_STARTED` | `ChatResponseUpdate` with empty contents, sets `ConversationId`, `ResponseId` | `ChatResponseUpdate` with `thread_id`, `run_id` in `additional_properties` |
| `TEXT_MESSAGE_START` | Tracks in `TextMessageBuilder`, no immediate output | `ChatResponseUpdate` with empty contents |
| `TEXT_MESSAGE_CONTENT` | `ChatResponseUpdate` with text delta | `ChatResponseUpdate` with `Content.from_text()` |
| `TEXT_MESSAGE_END` | Resets builder state | Returns `None` |
| `TOOL_CALL_START` | Tracks in `ToolCallBuilder` | `ChatResponseUpdate` with `Content.from_function_call()` |
| `TOOL_CALL_ARGS` | Accumulates args in builder | `ChatResponseUpdate` with accumulated args |
| `TOOL_CALL_END` | Emits `FunctionCallContent` | Returns `None` |
| `TOOL_CALL_RESULT` | `ChatResponseUpdate` with `FunctionResultContent`, role=Tool | `ChatResponseUpdate` with `Content.from_function_result()` |
| `RUN_FINISHED` | `ChatResponseUpdate` with result text | `ChatResponseUpdate` with `FinishReason.STOP` |
| `RUN_ERROR` | `ChatResponseUpdate` with `ErrorContent` | `ChatResponseUpdate` with `Content.from_error()` |
| `STATE_SNAPSHOT` | `ChatResponseUpdate` with `DataContent` (application/json) | Not implemented in converter |
| `STATE_DELTA` | `ChatResponseUpdate` with `DataContent` (application/json-patch+json) | Not implemented in converter |

### .NET Conversion Logic (ChatResponseUpdateAGUIExtensions.cs)

```csharp
public static async IAsyncEnumerable<ChatResponseUpdate> AsChatResponseUpdatesAsync(
    this IAsyncEnumerable<BaseEvent> events,
    JsonSerializerOptions jsonSerializerOptions,
    CancellationToken cancellationToken = default)
{
    string? conversationId = null;
    string? responseId = null;
    var textMessageBuilder = new TextMessageBuilder();
    var toolCallAccumulator = new ToolCallBuilder();
    
    await foreach (var evt in events)
    {
        switch (evt)
        {
            case RunStartedEvent runStarted:
                conversationId = runStarted.ThreadId;
                responseId = runStarted.RunId;
                yield return ValidateAndEmitRunStart(runStarted);
                break;
                
            case TextMessageContentEvent textContent:
                yield return textMessageBuilder.EmitTextUpdate(textContent);
                break;
                
            case ToolCallEndEvent toolCallEnd:
                yield return toolCallAccumulator.EmitToolCallUpdate(toolCallEnd, options);
                break;
                
            // ... other cases
        }
    }
}
```

### Python Conversion Logic (AGUIEventConverter)

```python
class AGUIEventConverter:
    def convert_event(self, event: dict[str, Any]) -> ChatResponseUpdate | None:
        event_type = event.get("type", "")
        
        if event_type == "RUN_STARTED":
            return self._handle_run_started(event)
        elif event_type == "TEXT_MESSAGE_CONTENT":
            return self._handle_text_message_content(event)
        # ... etc
```

### Key Differences

1. **.NET** emits updates incrementally for text, accumulates tool args until `TOOL_CALL_END`
2. **Python** emits on every event including `TOOL_CALL_ARGS` deltas
3. Both track `thread_id`/`run_id` for conversation continuity
4. Both handle client vs server tool distinction

## 4. Configuration Options

### .NET Options

```csharp
// Constructor parameters
- HttpClient httpClient           // Required: HTTP client for requests
- string endpoint                 // Required: AG-UI server URL
- ILoggerFactory? loggerFactory   // Optional: For logging
- JsonSerializerOptions? options  // Optional: Custom JSON settings
- IServiceProvider? serviceProvider // Optional: DI container

// Chat options (via ChatOptions)
- ConversationId                  // Maps to thread_id
- Tools                           // Client-side tools to advertise
- AdditionalProperties["agui_thread_id"] // Thread ID tracking
```

### Python Options

```python
# Constructor parameters
endpoint: str                     # Required: AG-UI server URL
http_client: httpx.AsyncClient    # Optional: Custom HTTP client
timeout: float = 60.0             # Optional: Request timeout
additional_properties: dict       # Optional: Extra properties

# Chat options (AGUIChatOptions TypedDict)
class AGUIChatOptions(ChatOptions, total=False):
    # Inherited from ChatOptions
    model_id: str
    temperature: float
    top_p: float
    max_tokens: int
    stop: list[str]
    tools: list[Tool]
    tool_choice: str
    metadata: dict  # Contains thread_id
    
    # AG-UI specific
    forward_props: dict[str, Any]  # Additional props to server
    context: dict[str, Any]        # Shared context/state
```

### Recommended Go Options

```go
// ClientConfig holds AG-UI client configuration
type ClientConfig struct {
    // Endpoint is the AG-UI server URL (required)
    Endpoint string
    
    // HTTPClient is the HTTP client to use (optional, defaults to http.DefaultClient)
    HTTPClient *http.Client
    
    // Timeout for requests (optional, defaults to 60s)
    Timeout time.Duration
    
    // Logger for debug output (optional)
    Logger *slog.Logger
}

// RunOptions holds per-request options
type RunOptions struct {
    // ThreadID for conversation continuity (auto-generated if empty)
    ThreadID string
    
    // Tools to advertise to the server
    Tools []tool.Definition
    
    // State to send to the server
    State any
    
    // Context items for the agent
    Context []ContextItem
    
    // ForwardedProps for custom server parameters
    ForwardedProps map[string]any
}
```

## 5. Error Handling Patterns

### .NET Error Handling

```csharp
// HTTP errors - exception propagation
response.EnsureSuccessStatusCode();

// SSE parsing errors - via JsonException
private static BaseEvent ItemParser(string type, ReadOnlySpan<byte> data)
{
    return JsonSerializer.Deserialize(data, ...) ??
        throw new InvalidOperationException("Failed to deserialize SSE item.");
}

// Protocol errors - via ErrorContent
case RunErrorEvent runError:
    yield return new ChatResponseUpdate(
        ChatRole.Assistant, 
        [(new ErrorContent(runError.Message) { ErrorCode = runError.Code })]);

// State validation
if (!string.Equals(runFinished.ThreadId, conversationId))
    throw new InvalidOperationException("Thread ID mismatch");
```

### Python Error Handling

```python
# HTTP errors
try:
    response.raise_for_status()
except httpx.HTTPStatusError as e:
    logger.error(f"HTTP request failed: {e.response.status_code}")
    raise

# SSE parsing - continue on error
try:
    event = json.loads(data)
    yield event
except json.JSONDecodeError as e:
    logger.warning(f"Failed to parse SSE data: {data}")
    continue  # Don't fail entire stream

# Protocol errors - via error content
def _handle_run_error(self, event: dict) -> ChatResponseUpdate:
    return ChatResponseUpdate(
        role=Role.ASSISTANT,
        finish_reason=FinishReason.CONTENT_FILTER,
        contents=[Content.from_error(message=error_message, error_code="RUN_ERROR")],
    )
```

### Recommended Go Error Handling

```go
// Define specific error types
var (
    ErrSSEParsing    = errors.New("failed to parse SSE event")
    ErrHTTPRequest   = errors.New("HTTP request failed")
    ErrProtocol      = errors.New("AG-UI protocol error")
    ErrThreadMismatch = errors.New("thread ID mismatch")
)

// Error handling approach:
// 1. HTTP errors: return immediately
// 2. SSE parsing errors: log and skip, continue stream
// 3. Protocol errors: emit via RunErrorEvent, then stop
// 4. Context cancellation: clean shutdown

func (c *Client) Run(ctx context.Context, messages []agent.Message, opts *RunOptions) (
    <-chan agent.ResponseUpdate, error) {
    
    updates := make(chan agent.ResponseUpdate)
    
    go func() {
        defer close(updates)
        
        events, err := c.httpService.PostRun(ctx, input)
        if err != nil {
            updates <- agent.ResponseUpdate{
                Kind:  agent.UpdateKindError,
                Error: fmt.Errorf("%w: %v", ErrHTTPRequest, err),
            }
            return
        }
        
        for event := range events {
            update := c.converter.Convert(event)
            if update != nil {
                select {
                case updates <- *update:
                case <-ctx.Done():
                    return
                }
            }
        }
    }()
    
    return updates, nil
}
```

## 6. Hybrid Tool Execution Pattern

### Overview

AG-UI supports hybrid tool execution where:
- **Client tools**: Defined on client, executed locally via function invocation middleware
- **Server tools**: Defined on server, executed remotely

### .NET Implementation

```csharp
// Build set of client tool names
var clientToolSet = new HashSet<string>();
foreach (var tool in options?.Tools ?? [])
{
    clientToolSet.Add(tool.Name);
}

// When receiving function call
if (update.Contents[0] is FunctionCallContent fcc)
{
    if (clientToolSet.Contains(fcc.Name))
    {
        // Client tool - pass to FunctionInvokingChatClient
        fcc.AdditionalProperties["agui_thread_id"] = threadId;
    }
    else
    {
        // Server tool - wrap to hide from FunctionInvokingChatClient
        update.Contents[0] = new ServerFunctionCallContent(fcc);
    }
}
```

### Python Implementation

```python
# Build client tool set
client_tool_set: set[str] = set()
if tools := options.get("tools"):
    for tool in tools:
        if hasattr(tool, "name"):
            client_tool_set.add(tool.name)

# When receiving function call
for i, content in enumerate(update.contents):
    if content.type == "function_call":
        if content.name in client_tool_set:
            # Client tool - let @use_function_invocation handle it
            content.additional_properties["agui_thread_id"] = thread_id
        else:
            # Server tool - wrap to prevent local execution
            self._register_server_tool_placeholder(content.name)
            update.contents[i] = Content(type="server_function_call", function_call=content)
```

### Recommended Go Pattern

```go
type Client struct {
    httpService *HTTPService
    converter   *EventConverter
    clientTools map[string]struct{} // Set of client tool names
}

func (c *Client) distinguishToolCall(tc *agent.ToolCallDelta) ToolCallType {
    if _, isClient := c.clientTools[tc.Name]; isClient {
        return ToolCallTypeClient
    }
    return ToolCallTypeServer
}

type ToolCallType int

const (
    ToolCallTypeClient ToolCallType = iota
    ToolCallTypeServer
)
```

## 7. Recommended Go Implementation Structure

### Package Structure

```
go/protocol/agui/
├── client.go           # AGUIClient implementation
├── client_test.go      # Client tests
├── http_service.go     # HTTP + SSE handling
├── http_service_test.go
├── event_parser.go     # SSE event parsing
├── event_parser_test.go
├── response_converter.go  # Event → ResponseUpdate
├── response_converter_test.go
├── types.go            # Request/response types
├── events.go           # (existing) Event types
├── converter.go        # (existing) ResponseUpdate → Event
├── server.go           # (existing) Server implementation
```

### Core Interfaces

```go
// Client is an AG-UI protocol client that can communicate with AG-UI servers.
type Client struct {
    config      ClientConfig
    httpService *HTTPService
    converter   *ResponseConverter
    clientTools map[string]struct{}
}

// ClientOption configures the client.
type ClientOption func(*Client)

// NewClient creates a new AG-UI client.
func NewClient(endpoint string, opts ...ClientOption) *Client

// Run executes a streaming request to the AG-UI server.
func (c *Client) Run(ctx context.Context, messages []agent.Message, opts *RunOptions) (
    <-chan agent.ResponseUpdate, error)

// RunBlocking executes a request and waits for completion.
func (c *Client) RunBlocking(ctx context.Context, messages []agent.Message, opts *RunOptions) (
    *agent.Response, error)
```

### HTTP Service

```go
// HTTPService handles HTTP communication with AG-UI servers.
type HTTPService struct {
    client   *http.Client
    endpoint string
    logger   *slog.Logger
}

// PostRun sends a run request and returns SSE events.
func (s *HTTPService) PostRun(ctx context.Context, input *RunInput) (<-chan Event, error)
```

### Response Converter

```go
// ResponseConverter converts AG-UI events to agent.ResponseUpdate.
// This is the inverse of the existing EventConverter.
type ResponseConverter struct {
    threadID           string
    runID              string
    currentMessageID   string
    currentToolCallID  string
    currentToolName    string
    accumulatedArgs    strings.Builder
}

// NewResponseConverter creates a new converter.
func NewResponseConverter() *ResponseConverter

// Convert converts an AG-UI event to a ResponseUpdate.
func (c *ResponseConverter) Convert(event Event) *agent.ResponseUpdate
```

## 8. Test Strategy

### Unit Tests

1. **HTTP Service Tests**
   - Mock HTTP responses
   - Test SSE parsing with various formats
   - Test error handling (HTTP errors, malformed SSE)
   - Test context cancellation

2. **Response Converter Tests**
   - Test each event type conversion
   - Test state management across events
   - Test tool call accumulation
   - Test error event handling

3. **Client Tests**
   - Test Run with mock HTTP service
   - Test RunBlocking aggregation
   - Test client vs server tool distinction
   - Test thread ID management
   - Test option propagation

### Integration Tests

1. **End-to-End with Server**
   - Start Go AG-UI server
   - Connect with Go AG-UI client
   - Test full conversation flow
   - Test tool execution round-trip

2. **Cross-Platform Tests**
   - Go client → .NET server
   - Go client → Python server (if available)

### Test Data

Use existing test fixtures from:
- `go/protocol/agui/testdata/` (if exists)
- Create new fixtures matching .NET/Python test cases

### Example Test

```go
func TestResponseConverter_TextMessage(t *testing.T) {
    converter := NewResponseConverter()
    
    // Simulate RUN_STARTED
    startEvent := &RunStartedEvent{ThreadID: "t1", RunID: "r1"}
    update := converter.Convert(startEvent)
    require.NotNil(t, update)
    assert.Equal(t, "t1", update.AdditionalProperties["thread_id"])
    
    // Simulate TEXT_MESSAGE_START
    msgStart := &TextMessageStartEvent{MessageID: "m1", Role: "assistant"}
    update = converter.Convert(msgStart)
    // May return nil or empty update
    
    // Simulate TEXT_MESSAGE_CONTENT
    content := &TextMessageContentEvent{MessageID: "m1", Delta: "Hello"}
    update = converter.Convert(content)
    require.NotNil(t, update)
    assert.Equal(t, "Hello", update.Delta.TextDelta)
    
    // Simulate TEXT_MESSAGE_END
    msgEnd := &TextMessageEndEvent{MessageID: "m1"}
    update = converter.Convert(msgEnd)
    // May return nil
    
    // Simulate RUN_FINISHED
    finish := &RunFinishedEvent{ThreadID: "t1", RunID: "r1"}
    update = converter.Convert(finish)
    require.NotNil(t, update)
    assert.Equal(t, agent.UpdateKindDone, update.Kind)
}
```

## 9. Implementation Priority

### Phase 1: Core Client (High Priority)
1. `types.go` - Request/response types for client
2. `http_service.go` - SSE streaming HTTP client
3. `event_parser.go` - Parse JSON to Event types
4. `response_converter.go` - Event → ResponseUpdate

### Phase 2: Client API (High Priority)
1. `client.go` - Main client implementation
2. Client tests with mocked services

### Phase 3: Integration (Medium Priority)
1. Integration tests with existing server
2. Cross-platform compatibility verification

### Phase 4: Enhancements (Lower Priority)
1. State snapshot/delta handling
2. Context item support
3. Forwarded props support

## 10. Key Implementation Notes

1. **SSE Parsing**: Go stdlib doesn't have built-in SSE parser. Use `bufio.Scanner` or third-party library like `r3labs/sse`.

2. **Thread ID Management**: Both .NET and Python auto-generate thread IDs if not provided. The client should:
   - Accept thread ID in options
   - Generate `thread_<uuid>` if not provided
   - Return thread ID in first update for caller to capture

3. **Tool Distinction**: The client must track which tools are client-side to properly route function calls. Server tools should be marked or wrapped to prevent local execution.

4. **Channel-Based Streaming**: Unlike .NET's `IAsyncEnumerable` and Python's `AsyncIterable`, Go should use channels with proper context cancellation support.

5. **JSON Parsing**: Use the existing event types from `events.go`. Add `UnmarshalJSON` implementations for polymorphic event parsing (similar to .NET's `BaseEventJsonConverter`).
