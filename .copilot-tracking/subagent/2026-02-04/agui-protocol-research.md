# AG-UI Protocol Implementation Analysis

**Date:** 2026-02-04
**Status:** Complete
**Scope:** .NET and Python AG-UI implementations in Microsoft Agent Framework SDK

---

## 1. Protocol Specification Summary

### 1.1 Overview

AG-UI (Agent-User Interaction) is an open, lightweight, event-based protocol for communication between AI agents and user-facing applications. Key characteristics:

- **SSE-Based Streaming**: Uses Server-Sent Events for real-time streaming
- **Event-Driven Architecture**: Supports nondeterministic agent behavior
- **Bidirectional Communication**: Client-server state synchronization
- **Protocol Interoperability**: Complements MCP (tool/context) and A2A (agent-to-agent) protocols

**Reference:** [docs/decisions/0010-ag-ui-support.md](../../../docs/decisions/0010-ag-ui-support.md)

### 1.2 Design Principles

From the ADR:
1. **Event Models as Internal Types** - AG-UI events are internal with framework-native public API
2. **No Custom Content Types** - Uses existing `ChatResponseUpdate` properties
3. **Agent Factory Pattern** - Supports multi-tenancy via factory functions
4. **Bidirectional Conversion** - Symmetric logic for server→client and client→server

---

## 2. Event Type Definitions

### 2.1 Complete Event Type Catalog

| Event Type | Description | Key Properties |
|------------|-------------|----------------|
| `RUN_STARTED` | Run lifecycle start | `threadId`, `runId` |
| `RUN_FINISHED` | Run lifecycle end | (none) |
| `RUN_ERROR` | Error during run | `message`, `code` |
| `TEXT_MESSAGE_START` | Text message begins | `messageId`, `role` |
| `TEXT_MESSAGE_CONTENT` | Text content delta | `messageId`, `delta` |
| `TEXT_MESSAGE_END` | Text message ends | (none) |
| `TOOL_CALL_START` | Tool call begins | `toolCallId`, `toolCallName`, `parentMessageId` |
| `TOOL_CALL_ARGS` | Tool argument delta | `toolCallId`, `delta` |
| `TOOL_CALL_END` | Tool call ends | `toolCallId` |
| `TOOL_CALL_RESULT` | Tool execution result | `toolCallId`, `content`, `role` |
| `STATE_SNAPSHOT` | Full state snapshot | `snapshot` (JsonElement) |
| `STATE_DELTA` | Incremental state update | `delta` (JsonElement) |

### 2.2 Event Flow Diagram

```
RUN_STARTED
    ├── TEXT_MESSAGE_START
    │   ├── TEXT_MESSAGE_CONTENT (1..N)
    │   └── TEXT_MESSAGE_END
    ├── TOOL_CALL_START
    │   ├── TOOL_CALL_ARGS (0..N)
    │   ├── TOOL_CALL_END
    │   └── TOOL_CALL_RESULT
    ├── STATE_SNAPSHOT (optional)
    └── STATE_DELTA (optional)
RUN_FINISHED | RUN_ERROR
```

---

## 3. .NET Implementation Analysis

### 3.1 Package Structure

| Package | Purpose |
|---------|---------|
| `Microsoft.Agents.AI.AGUI` | Client-side AG-UI consumption |
| `Microsoft.Agents.AI.Hosting.AGUI.AspNetCore` | Server-side AG-UI hosting |

### 3.2 Key Files and Interfaces

#### Client Package (`Microsoft.Agents.AI.AGUI`)

| File | Lines | Purpose |
|------|-------|---------|
| [AGUIChatClient.cs](../../../dotnet/src/Microsoft.Agents.AI.AGUI/AGUIChatClient.cs) | 380 | Main client implementing `IChatClient` |
| [AGUIHttpService.cs](../../../dotnet/src/Microsoft.Agents.AI.AGUI/AGUIHttpService.cs) | - | HTTP/SSE streaming service |
| [Shared/AGUIEventTypes.cs](../../../dotnet/src/Microsoft.Agents.AI.AGUI/Shared/AGUIEventTypes.cs#L1-L34) | 34 | Event type constants |
| [Shared/BaseEvent.cs](../../../dotnet/src/Microsoft.Agents.AI.AGUI/Shared/BaseEvent.cs#L1-L17) | 17 | Abstract base event class |
| [Shared/BaseEventJsonConverter.cs](../../../dotnet/src/Microsoft.Agents.AI.AGUI/Shared/BaseEventJsonConverter.cs#L1-L107) | 107 | Custom polymorphic JSON deserializer |

#### Event Types (Shared folder)

| File | Event Type |
|------|------------|
| [RunStartedEvent.cs](../../../dotnet/src/Microsoft.Agents.AI.AGUI/Shared/RunStartedEvent.cs#L1-L24) | `RUN_STARTED` |
| [RunFinishedEvent.cs](../../../dotnet/src/Microsoft.Agents.AI.AGUI/Shared/RunFinishedEvent.cs) | `RUN_FINISHED` |
| [RunErrorEvent.cs](../../../dotnet/src/Microsoft.Agents.AI.AGUI/Shared/RunErrorEvent.cs#L1-L24) | `RUN_ERROR` |
| [TextMessageStartEvent.cs](../../../dotnet/src/Microsoft.Agents.AI.AGUI/Shared/TextMessageStartEvent.cs#L1-L24) | `TEXT_MESSAGE_START` |
| [TextMessageContentEvent.cs](../../../dotnet/src/Microsoft.Agents.AI.AGUI/Shared/TextMessageContentEvent.cs#L1-L24) | `TEXT_MESSAGE_CONTENT` |
| [TextMessageEndEvent.cs](../../../dotnet/src/Microsoft.Agents.AI.AGUI/Shared/TextMessageEndEvent.cs) | `TEXT_MESSAGE_END` |
| [ToolCallStartEvent.cs](../../../dotnet/src/Microsoft.Agents.AI.AGUI/Shared/ToolCallStartEvent.cs#L1-L27) | `TOOL_CALL_START` |
| [ToolCallArgsEvent.cs](../../../dotnet/src/Microsoft.Agents.AI.AGUI/Shared/ToolCallArgsEvent.cs#L1-L24) | `TOOL_CALL_ARGS` |
| [ToolCallEndEvent.cs](../../../dotnet/src/Microsoft.Agents.AI.AGUI/Shared/ToolCallEndEvent.cs#L1-L22) | `TOOL_CALL_END` |
| [ToolCallResultEvent.cs](../../../dotnet/src/Microsoft.Agents.AI.AGUI/Shared/ToolCallResultEvent.cs#L1-L31) | `TOOL_CALL_RESULT` |
| [StateSnapshotEvent.cs](../../../dotnet/src/Microsoft.Agents.AI.AGUI/Shared/StateSnapshotEvent.cs#L1-L22) | `STATE_SNAPSHOT` |
| [StateDeltaEvent.cs](../../../dotnet/src/Microsoft.Agents.AI.AGUI/Shared/StateDeltaEvent.cs#L1-L22) | `STATE_DELTA` |

#### Server Package (`Microsoft.Agents.AI.Hosting.AGUI.AspNetCore`)

| File | Lines | Purpose |
|------|-------|---------|
| [AGUIEndpointRouteBuilderExtensions.cs](../../../dotnet/src/Microsoft.Agents.AI.Hosting.AGUI.AspNetCore/AGUIEndpointRouteBuilderExtensions.cs#L1-L82) | 82 | `MapAGUI()` extension method |
| [AGUIServerSentEventsResult.cs](../../../dotnet/src/Microsoft.Agents.AI.Hosting.AGUI.AspNetCore/AGUIServerSentEventsResult.cs#L1-L120) | 120 | SSE response formatting |
| [AGUIChatResponseUpdateStreamExtensions.cs](../../../dotnet/src/Microsoft.Agents.AI.Hosting.AGUI.AspNetCore/AGUIChatResponseUpdateStreamExtensions.cs#L1-L83) | 83 | Tool filtering for hybrid execution |

### 3.3 SSE Implementation Pattern (.NET)

```csharp
// Server-side SSE (AGUIServerSentEventsResult.cs:27-43)
httpContext.Response.ContentType = "text/event-stream";
httpContext.Response.Headers.CacheControl = "no-cache,no-store";
httpContext.Response.Headers.Pragma = "no-cache";

await SseFormatter.WriteAsync(
    WrapEventsAsSseItemsAsync(this._events, cancellationToken),
    body,
    this.SerializeEvent,
    cancellationToken);
```

### 3.4 Request/Response Model

**Request:** [RunAgentInput.cs](../../../dotnet/src/Microsoft.Agents.AI.AGUI/Shared/RunAgentInput.cs#L1-L39)
```json
{
  "threadId": "string",
  "runId": "string", 
  "messages": [...],
  "tools": [...],
  "state": {},
  "context": [],
  "forwardedProps": {}
}
```

---

## 4. Python Implementation Analysis

### 4.1 Package Structure

| Package | Location | Purpose |
|---------|----------|---------|
| `agent_framework_ag_ui` | `python/packages/ag-ui/` | Client + Server implementation |

### 4.2 Key Files and Interfaces

| File | Lines | Purpose |
|------|-------|---------|
| [\_\_init\_\_.py](../../../python/packages/ag-ui/agent_framework_ag_ui/__init__.py#L1-L35) | 35 | Public API exports |
| [_client.py](../../../python/packages/ag-ui/agent_framework_ag_ui/_client.py#L1-L429) | 429 | `AGUIChatClient` implementation |
| [_endpoint.py](../../../python/packages/ag-ui/agent_framework_ag_ui/_endpoint.py#L1-L103) | 103 | FastAPI endpoint helper |
| [_agent.py](../../../python/packages/ag-ui/agent_framework_ag_ui/_agent.py#L1-L112) | 112 | `AgentFrameworkAgent` wrapper |
| [_event_converters.py](../../../python/packages/ag-ui/agent_framework_ag_ui/_event_converters.py#L1-L207) | 207 | AG-UI events → Framework types |
| [_http_service.py](../../../python/packages/ag-ui/agent_framework_ag_ui/_http_service.py#L1-L162) | 162 | HTTP/SSE client service |
| [_types.py](../../../python/packages/ag-ui/agent_framework_ag_ui/_types.py#L1-L131) | 131 | Type definitions |
| [_run.py](../../../python/packages/ag-ui/agent_framework_ag_ui/_run.py#L1-L964) | 964 | Agent run orchestration |

### 4.3 Key Classes

| Class | File | Purpose |
|-------|------|---------|
| `AGUIChatClient` | _client.py | Client consuming AG-UI servers |
| `AGUIEventConverter` | _event_converters.py | Event to ChatResponseUpdate conversion |
| `AGUIHttpService` | _http_service.py | SSE stream parsing |
| `AgentFrameworkAgent` | _agent.py | Agent wrapper for protocol |
| `AGUIRequest` | _types.py | Pydantic request model |
| `FlowState` | _run.py | Orchestration state |

### 4.4 SSE Implementation Pattern (Python)

```python
# Server-side SSE (_endpoint.py:73-81)
return StreamingResponse(
    event_generator(),
    media_type="text/event-stream",
    headers={
        "Cache-Control": "no-cache",
        "Connection": "keep-alive",
        "X-Accel-Buffering": "no",
    },
)

# Client-side SSE parsing (_http_service.py:107-121)
async for line in response.aiter_lines():
    if line.startswith("data: "):
        data = line[6:]  # Remove "data: " prefix
        event = json.loads(data)
        yield event
```

### 4.5 Event Conversion Pattern (Python)

```python
# _event_converters.py:28-73
def convert_event(self, event: dict[str, Any]) -> ChatResponseUpdate | None:
    event_type = event.get("type", "")
    
    if event_type == "RUN_STARTED":
        return self._handle_run_started(event)
    elif event_type == "TEXT_MESSAGE_START":
        return self._handle_text_message_start(event)
    # ... etc
```

---

## 5. API Surface Comparison

### 5.1 Client APIs

| Feature | .NET | Python |
|---------|------|--------|
| Client Class | `AGUIChatClient` | `AGUIChatClient` |
| Base Interface | `IChatClient` (M.E.AI) | `BaseChatClient` |
| Streaming Method | `GetStreamingResponseAsync()` | `get_streaming_response()` |
| Non-streaming | `GetResponseAsync()` | `get_response()` |
| Context Manager | N/A (IDisposable) | `async with` support |
| Tool Invocation | `FunctionInvokingChatClient` wrapper | `@use_function_invocation` decorator |

### 5.2 Server APIs

| Feature | .NET | Python |
|---------|------|--------|
| Endpoint Registration | `app.MapAGUI("/", agent)` | `add_agent_framework_fastapi_endpoint(app, agent)` |
| Framework | ASP.NET Core | FastAPI |
| SSE Formatting | `SseFormatter` (System.Net) | `EventEncoder` (ag_ui) |
| Agent Factory | Via `MapAGUI` overloads | `AgentFrameworkAgent` wrapper |

### 5.3 Event Types

| Event | .NET Class | Python Handling |
|-------|------------|-----------------|
| `RUN_STARTED` | `RunStartedEvent` | `_handle_run_started()` |
| `RUN_FINISHED` | `RunFinishedEvent` | `_handle_run_finished()` |
| `RUN_ERROR` | `RunErrorEvent` | `_handle_run_error()` |
| `TEXT_MESSAGE_START` | `TextMessageStartEvent` | `_handle_text_message_start()` |
| `TEXT_MESSAGE_CONTENT` | `TextMessageContentEvent` | `_handle_text_message_content()` |
| `TEXT_MESSAGE_END` | `TextMessageEndEvent` | `_handle_text_message_end()` |
| `TOOL_CALL_START` | `ToolCallStartEvent` | `_handle_tool_call_start()` |
| `TOOL_CALL_ARGS` | `ToolCallArgsEvent` | `_handle_tool_call_args()` |
| `TOOL_CALL_END` | `ToolCallEndEvent` | `_handle_tool_call_end()` |
| `TOOL_CALL_RESULT` | `ToolCallResultEvent` | `_handle_tool_call_result()` |
| `STATE_SNAPSHOT` | `StateSnapshotEvent` | Via `ag_ui.core` |
| `STATE_DELTA` | `StateDeltaEvent` | Via `ag_ui.core` |

### 5.4 Type Definitions

| Concept | .NET Type | Python Type |
|---------|-----------|-------------|
| Request Model | `RunAgentInput` | `AGUIRequest` (Pydantic) |
| Base Event | `BaseEvent` (internal) | `BaseEvent` (from ag_ui.core) |
| Chat Options | `ChatOptions` (M.E.AI) | `AGUIChatOptions` (TypedDict) |
| Thread State | `FlowState` (internal) | `FlowState` (dataclass) |

---

## 6. Design Recommendations for Go Implementation

### 6.1 Package Structure

```
go/
├── protocol/
│   └── agui/
│       ├── types.go       # Event type definitions
│       ├── events.go      # Event structs (internal)
│       ├── client.go      # AG-UI client
│       ├── server.go      # SSE server helpers
│       └── converter.go   # Event ↔ ChatResponseUpdate
├── hosting/
│   └── agui/
│       ├── handler.go     # HTTP handler
│       └── sse.go         # SSE encoding/decoding
```

### 6.2 Event Types (types.go)

```go
// Event type constants (matches .NET/Python)
const (
    EventRunStarted         = "RUN_STARTED"
    EventRunFinished        = "RUN_FINISHED"
    EventRunError           = "RUN_ERROR"
    EventTextMessageStart   = "TEXT_MESSAGE_START"
    EventTextMessageContent = "TEXT_MESSAGE_CONTENT"
    EventTextMessageEnd     = "TEXT_MESSAGE_END"
    EventToolCallStart      = "TOOL_CALL_START"
    EventToolCallArgs       = "TOOL_CALL_ARGS"
    EventToolCallEnd        = "TOOL_CALL_END"
    EventToolCallResult     = "TOOL_CALL_RESULT"
    EventStateSnapshot      = "STATE_SNAPSHOT"
    EventStateDelta         = "STATE_DELTA"
)
```

### 6.3 Event Structs (events.go)

```go
// BaseEvent with type discriminator
type BaseEvent struct {
    Type string `json:"type"`
}

// RunStartedEvent
type RunStartedEvent struct {
    BaseEvent
    ThreadID string `json:"threadId"`
    RunID    string `json:"runId"`
}

// TextMessageContentEvent
type TextMessageContentEvent struct {
    BaseEvent
    MessageID string `json:"messageId"`
    Delta     string `json:"delta"`
}

// ToolCallStartEvent
type ToolCallStartEvent struct {
    BaseEvent
    ToolCallID      string  `json:"toolCallId"`
    ToolCallName    string  `json:"toolCallName"`
    ParentMessageID *string `json:"parentMessageId,omitempty"`
}
```

### 6.4 SSE Implementation

```go
// Server-side SSE writing
func WriteSSE(w http.ResponseWriter, events <-chan BaseEvent) error {
    w.Header().Set("Content-Type", "text/event-stream")
    w.Header().Set("Cache-Control", "no-cache")
    w.Header().Set("Connection", "keep-alive")
    
    flusher, ok := w.(http.Flusher)
    if !ok {
        return errors.New("streaming not supported")
    }
    
    for event := range events {
        data, _ := json.Marshal(event)
        fmt.Fprintf(w, "data: %s\n\n", data)
        flusher.Flush()
    }
    return nil
}

// Client-side SSE parsing
func ParseSSE(r io.Reader) <-chan BaseEvent {
    events := make(chan BaseEvent)
    scanner := bufio.NewScanner(r)
    
    go func() {
        defer close(events)
        for scanner.Scan() {
            line := scanner.Text()
            if strings.HasPrefix(line, "data: ") {
                var event BaseEvent
                json.Unmarshal([]byte(line[6:]), &event)
                events <- event
            }
        }
    }()
    return events
}
```

### 6.5 Key Design Decisions

1. **Internal Event Types**: Like .NET, keep event types internal; expose framework-native types
2. **Custom JSON Unmarshaling**: Use discriminator-based polymorphic deserialization
3. **Streaming-First**: Design around Go's channel patterns for SSE
4. **Thread Safety**: Ensure event converter maintains proper state for streaming
5. **Tool Filtering**: Implement hybrid tool execution (client vs server tools)

### 6.6 Implementation Priority

1. **Phase 1**: Event types and SSE utilities
2. **Phase 2**: Client implementation (`AGUIClient`)
3. **Phase 3**: Server implementation (`AGUIHandler`)
4. **Phase 4**: Integration with `ChatAgent`

---

## 7. Samples and Tests

### 7.1 .NET Samples

| Sample | Path |
|--------|------|
| Getting Started Server | [dotnet/samples/GettingStarted/AGUI/Step01_GettingStarted/Server/](../../../dotnet/samples/GettingStarted/AGUI/Step01_GettingStarted/Server/Program.cs) |
| Getting Started Client | [dotnet/samples/GettingStarted/AGUI/Step01_GettingStarted/Client/](../../../dotnet/samples/GettingStarted/AGUI/Step01_GettingStarted/Client/) |
| Backend Tools | [dotnet/samples/GettingStarted/AGUI/Step02_BackendTools/](../../../dotnet/samples/GettingStarted/AGUI/Step02_BackendTools/) |
| AGUI Client Server | [dotnet/samples/AGUIClientServer/](../../../dotnet/samples/AGUIClientServer/) |

### 7.2 Python Samples

| Sample | Path |
|--------|------|
| Getting Started | [python/packages/ag-ui/getting_started/](../../../python/packages/ag-ui/getting_started/) |
| Examples | [python/packages/ag-ui/agent_framework_ag_ui_examples/](../../../python/packages/ag-ui/agent_framework_ag_ui_examples/) |
| Tests | [python/packages/ag-ui/tests/](../../../python/packages/ag-ui/tests/) |

---

## 8. Key Findings Summary

### Architecture

- **Event-based protocol** with 12 defined event types
- **SSE streaming** for real-time communication
- **Internal events** with framework-native public API
- **Bidirectional** client and server support

### .NET Specifics

- Uses `Microsoft.Extensions.AI` abstractions (`IChatClient`)
- Custom `BaseEventJsonConverter` for polymorphic JSON
- `FunctionInvokingChatClient` wrapper for tool execution
- Preprocessor directives (`#if ASPNETCORE`) for shared code

### Python Specifics

- Uses `ag_ui.core` library for event types
- FastAPI `StreamingResponse` for SSE
- `@use_function_invocation` decorator pattern
- Pydantic models for request validation

### Go Recommendations

- Follow internal event pattern
- Use channels for streaming
- Implement custom JSON unmarshaler
- Integrate with existing `chat` package

---

**Research Complete:** 2026-02-04
