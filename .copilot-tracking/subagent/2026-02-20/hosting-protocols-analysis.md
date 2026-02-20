# Hosting Protocols Deep-Dive Analysis for C++ Port

**Date:** 2026-02-20
**Scope:** OpenAI-compatible, A2A, and AG-UI hosting protocol implementations

---

## 1. OpenAI-Compatible Hosting (`Microsoft.Agents.AI.Hosting.OpenAI`)

### 1.1 API Surfaces (3 distinct surfaces)

The OpenAI hosting layer exposes **three independent API surfaces**, each mapped via separate `EndpointRouteBuilderExtensions` partial classes.

#### Surface A: Chat Completions API

| Aspect | Detail |
|--------|--------|
| **Route** | `POST /{agentName}/v1/chat/completions/` |
| **Endpoints** | 1 |
| **Streaming** | SSE via `text/event-stream` when `request.Stream == true` |
| **Processor** | `AIAgentChatCompletionsProcessor` |

**Request model:** `CreateChatCompletion`
- `messages` (required): `IList<ChatCompletionRequestMessage>`
- `model` (required): string
- `stream`: bool?
- `temperature`, `top_p`, `frequency_penalty`, `max_completion_tokens`: numeric params
- `tools`: tool definitions
- `tool_choice`: tool selection strategy
- `response_format`, `stop`, `logit_bias`, `logprobs`, `audio`, etc.

**Response models:**
- Non-streaming: `ChatCompletion` (JSON object with `choices[]`)
- Streaming: SSE stream of `ChatCompletionChunk` objects (no event type name, just `data:` lines)

**SSE streaming format (Chat Completions):**
- Headers: `Content-Type: text/event-stream`, `Cache-Control: no-cache,no-store`, `Connection: keep-alive`, `Content-Encoding: identity`
- Each SSE item is a `ChatCompletionChunk` serialized as JSON
- No explicit `event:` field (uses default SSE event type)
- Chunks contain: `id`, `created`, `model`, `choices[]` with `delta` (role, content, tool_calls), `finish_reason`, `usage`

**Key model types (12 model files):**
- `CreateChatCompletion`, `ChatCompletion`, `ChatCompletionChunk`
- `ChatCompletionChoice`, `ChatCompletionChoiceChunk`
- `ChatCompletionRequestMessage`, `MessageContent`, `MessageContentPart`
- `CompletionUsage`, `Tool`, `ToolChoice`
- `ResponseFormat`, `StopSequences`

---

#### Surface B: Responses API

| Aspect | Detail |
|--------|--------|
| **Base Route** | `/{agentName}/v1/responses` or `/v1/responses` |
| **Endpoints** | 5 |
| **Streaming** | SSE via `SseJsonResult<StreamingResponseEvent>` |
| **Handler** | `ResponsesHttpHandler` → `IResponsesService` |

**Endpoints:**

| Method | Path | Name | Description |
|--------|------|------|-------------|
| `POST` | `/` | CreateResponse | Creates a model response (streaming or non-streaming) |
| `GET` | `/{responseId}` | GetResponse | Retrieves a response by ID (optionally streaming) |
| `POST` | `/{responseId}/cancel` | CancelResponse | Cancels an in-progress response |
| `DELETE` | `/{responseId}` | DeleteResponse | Deletes a response |
| `GET` | `/{responseId}/input_items` | ListResponseInputItems | Lists input items with pagination |

**Request model:** `CreateResponse`
- `input` (required): `ResponseInput` (string or array of `InputMessage`)
- `agent`: `AgentReference`
- `model`, `instructions`, `max_output_tokens`, `temperature`, `top_p`
- `reasoning`: `ReasoningOptions`
- `stream`: bool?
- `previous_response_id`: string (multi-turn)
- `conversation`: `ConversationReference`
- `parallel_tool_calls`, `metadata`, `store`
- `text`: `TextConfiguration`

**Response model:** `Response`
- `id`, `object` ("response"), `created_at`, `model`, `status` (enum: completed/failed/in_progress/cancelled/queued/incomplete)
- `agent`: `AgentId`, `error`: `ResponseError`, `output[]`: `ItemResource[]`
- `metadata`, `previous_response_id`, `usage`

**SSE streaming events (19 event types):**

| Event Type String | Class | Description |
|-------------------|-------|-------------|
| `response.created` | `StreamingResponseCreated` | Response created, streaming begun |
| `response.in_progress` | `StreamingResponseInProgress` | Response processing |
| `response.completed` | `StreamingResponseCompleted` | Response fully completed |
| `response.incomplete` | `StreamingResponseIncomplete` | Response ended incomplete |
| `response.failed` | `StreamingResponseFailed` | Response failed |
| `response.cancelled` | `StreamingResponseCancelled` | Response cancelled |
| `response.output_item.added` | `StreamingOutputItemAdded` | New output item added |
| `response.output_item.done` | `StreamingOutputItemDone` | Output item completed |
| `response.content_part.added` | `StreamingContentPartAdded` | Content part added to item |
| `response.content_part.done` | `StreamingContentPartDone` | Content part completed |
| `response.output_text.delta` | `StreamingOutputTextDelta` | Incremental text chunk |
| `response.output_text.done` | `StreamingOutputTextDone` | Text output completed |
| `response.function_call_arguments.delta` | `StreamingFunctionCallArgumentsDelta` | Function args delta |
| `response.function_call_arguments.done` | `StreamingFunctionCallArgumentsDone` | Function args complete |
| `response.reasoning_summary_text.delta` | `StreamingReasoningSummaryTextDelta` | Reasoning text delta |
| `response.reasoning_summary_text.done` | `StreamingReasoningSummaryTextDone` | Reasoning text done |
| `response.workflow_event.completed` | `StreamingWorkflowEventComplete` | Workflow lifecycle event |
| `response.function_approval.requested` | `StreamingFunctionApprovalRequested` | Human-in-the-loop approval request |
| `response.function_approval.responded` | `StreamingFunctionApprovalResponded` | Approval response |

SSE format uses `event:` field set to the type discriminator string, `data:` field contains JSON-serialized event object.

**Streaming event generators (11 generators):**
- `AssistantMessageEventGenerator`, `TextReasoningContentEventGenerator`
- `FunctionCallEventGenerator`, `FunctionResultEventGenerator`
- `FunctionApprovalRequestEventGenerator`, `FunctionApprovalResponseEventGenerator`
- `ImageContentEventGenerator`, `AudioContentEventGenerator`
- `FileContentEventGenerator`, `HostedFileContentEventGenerator`
- `ErrorContentEventGenerator`

**Supporting services:**
- `IResponsesService` — core service interface (7 methods)
- `InMemoryResponsesService` — default in-memory implementation
- `IResponseExecutor` / `AIAgentResponseExecutor` / `HostedAgentResponseExecutor` — agent invocation
- `IConversationStorage` — persistence abstraction
- `SequenceNumber` — thread-safe sequence numbering for events

---

#### Surface C: Conversations API

| Aspect | Detail |
|--------|--------|
| **Base Route** | `/v1/conversations` |
| **Endpoints** | 8 |
| **Streaming** | None (REST only) |
| **Handler** | `ConversationsHttpHandler` |

**Endpoints:**

| Method | Path | Name | Description |
|--------|------|------|-------------|
| `GET` | `/` | ListConversationsByAgent | List conversations by agent_id (non-standard extension) |
| `POST` | `/` | CreateConversation | Create a new conversation |
| `GET` | `/{conversationId}` | GetConversation | Retrieve conversation by ID |
| `POST` | `/{conversationId}` | UpdateConversation | Update conversation metadata |
| `DELETE` | `/{conversationId}` | DeleteConversation | Delete conversation and all items |
| `POST` | `/{conversationId}/items` | CreateItems | Add items to conversation |
| `GET` | `/{conversationId}/items` | ListItems | List items in conversation |
| `GET` | `/{conversationId}/items/{itemId}` | GetItem | Retrieve specific item |
| `DELETE` | `/{conversationId}/items/{itemId}` | DeleteItem | Delete specific item |

**Key models:**
- `Conversation`, `CreateConversationRequest`, `UpdateConversationRequest`, `AddMessageRequest`
- `IConversationStorage` / `InMemoryConversationStorage`
- `IAgentConversationIndex` / `InMemoryAgentConversationIndex`

---

### 1.2 OpenAI Hosting Summary

| Metric | Count |
|--------|-------|
| **Total HTTP endpoints** | 14 (1 + 5 + 8) |
| **API surfaces** | 3 (Chat Completions, Responses, Conversations) |
| **SSE streaming event types** | 19 (Responses API) + unnamed chunks (Chat Completions) |
| **Model types (approx)** | ~40+ across all three surfaces |
| **Service interfaces** | 4 (`IResponsesService`, `IResponseExecutor`, `IConversationStorage`, `IAgentConversationIndex`) |
| **JSON serialization contexts** | 2 (`ChatCompletionsJsonContext`, `OpenAIHostingJsonContext`) |

---

## 2. A2A Hosting (`Microsoft.Agents.AI.Hosting.A2A.AspNetCore`)

### 2.1 Protocol Overview

A2A (Agent-to-Agent) follows the [A2A Protocol specification](https://github.com/a2aproject/A2A). The hosting layer **delegates to the `A2A` and `A2A.AspNetCore` NuGet packages** for actual HTTP endpoint registration and JSON-RPC handling.

### 2.2 Architecture

```
MapA2A() extensions
    ├─ Creates AIHostAgent wrapper around AIAgent
    ├─ Creates/configures TaskManager
    │   ├─ OnMessageReceived event handler (agent invocation)
    │   └─ OnAgentCardQuery event handler (card discovery)
    ├─ Calls A2ARouteBuilderExtensions.MapA2A() (from A2A.AspNetCore NuGet)
    └─ Calls endpoints.MapHttpA2A() (HTTP endpoint registration)
```

### 2.3 Endpoints (defined by A2A protocol)

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/.well-known/agent-card.json` | `GET` | Agent card discovery (returned by `OnAgentCardQuery` handler) |
| `/{path}` | `POST` | JSON-RPC 2.0 endpoint (single POST, method in body) |

### 2.4 JSON-RPC 2.0 Methods

All A2A operations go through a **single POST** endpoint using JSON-RPC 2.0:

| JSON-RPC Method | Description |
|-----------------|-------------|
| `message/send` | Send a message to the agent (request/response) |
| `message/stream` | Send a message and receive SSE streaming response |
| `tasks/get` | Get the current state of a task |
| `tasks/cancel` | Cancel an in-progress task |

**JSON-RPC 2.0 message format:**
```json
{
    "id": "1",
    "jsonrpc": "2.0",
    "method": "message/send",
    "params": {
        "message": {
            "kind": "message",
            "role": "user",
            "messageId": "msg_1",
            "contextId": "...",
            "parts": [
                { "kind": "text", "text": "..." }
            ]
        },
        "metadata": {}
    }
}
```

### 2.5 A2A Response Types

| Response Type | Class | Description |
|---------------|-------|-------------|
| Message response | `AgentMessage` | Direct message with parts (text, data, file) |
| Task response | `AgentTask` | Long-running task with status, artifacts |
| Task update event | `TaskUpdateEvent` / `TaskArtifactUpdateEvent` | Streaming task updates |

### 2.6 SSE Streaming (message/stream)

The `message/stream` method returns SSE events where each event's `data:` field contains a JSON-RPC response wrapper around one of:
- `AgentMessage` — full message
- `AgentTask` — task state
- `TaskUpdateEvent` — incremental task update
- `TaskArtifactUpdateEvent` — artifact update

### 2.7 Key Types for C++ Port

| Type | Purpose |
|------|---------|
| `AgentCard` | Agent metadata (name, description, skills, capabilities URL) |
| `AgentMessage` | Message with parts, contextId, messageId, role |
| `AgentTask` | Task with id, status (state machine), contextId, artifacts |
| `TaskState` | Enum: Submitted, Working, Completed, Failed, Canceled |
| `MessageSendParams` | Send request params (message + metadata) |
| `A2AResponse` | Base type (AgentMessage or AgentTask) |
| `A2AEvent` | SSE event base (message, task, task update) |

**Note:** The actual JSON-RPC handler and SSE formatter are in the external `A2A` and `A2A.AspNetCore` NuGet packages. The framework layer provides the agent adapter (`AIHostAgent`, `AIAgentExtensions`) and ASP.NET Core registration (`MapA2A()`).

### 2.8 A2A Hosting Summary

| Metric | Count |
|--------|-------|
| **HTTP endpoints** | 2 (GET agent card + POST JSON-RPC) |
| **JSON-RPC methods** | 4 (message/send, message/stream, tasks/get, tasks/cancel) |
| **SSE event types** | 3 (AgentMessage, AgentTask, TaskUpdateEvent) |
| **External NuGet deps** | 2 (`A2A`, `A2A.AspNetCore`) |
| **Framework adapter types** | ~5 (`AIHostAgent`, `AIAgentExtensions`, session store, converters) |

---

## 3. AG-UI Hosting (`Microsoft.Agents.AI.Hosting.AGUI.AspNetCore`)

### 3.1 Protocol Overview

AG-UI (Agent-User Interface) protocol, originating from CopilotKit. Single POST endpoint that returns SSE stream.

### 3.2 Endpoint

| Endpoint | Method | Description |
|----------|--------|-------------|
| `POST {pattern}` | POST | Run agent, returns SSE event stream |

Single endpoint mapped via `MapAGUI(endpoints, pattern, aiAgent)`.

### 3.3 Request Model: `RunAgentInput`

```json
{
    "threadId": "string",
    "runId": "string",
    "state": { /* arbitrary JSON */ },
    "messages": [ /* AGUIMessage[] */ ],
    "tools": [ /* AGUITool[] */ ],
    "context": [ { "description": "...", "value": "..." } ],
    "forwardedProps": { /* arbitrary JSON */ }
}
```

**Message types:**
- `AGUIUserMessage`, `AGUIAssistantMessage`, `AGUISystemMessage`, `AGUIDeveloperMessage`, `AGUIToolMessage`
- All inherit `AGUIMessage` with role-based polymorphic deserialization (`AGUIMessageJsonConverter`)

**Tool definition:** `AGUITool` with `AGUIFunctionCall` (name, description, parameters)

### 3.4 SSE Event Types (12 types)

| Event Type Constant | Class | Description |
|---------------------|-------|-------------|
| `RUN_STARTED` | `RunStartedEvent` | Run lifecycle start (threadId, runId) |
| `RUN_FINISHED` | `RunFinishedEvent` | Run lifecycle end (threadId, runId) |
| `RUN_ERROR` | `RunErrorEvent` | Error occurred (code, message) |
| `TEXT_MESSAGE_START` | `TextMessageStartEvent` | New assistant text message (messageId, role) |
| `TEXT_MESSAGE_CONTENT` | `TextMessageContentEvent` | Text delta (messageId, delta) |
| `TEXT_MESSAGE_END` | `TextMessageEndEvent` | Text message complete (messageId) |
| `TOOL_CALL_START` | `ToolCallStartEvent` | Tool invocation begins (toolCallId, toolCallName, parentMessageId) |
| `TOOL_CALL_ARGS` | `ToolCallArgsEvent` | Tool arguments delta (toolCallId, delta) |
| `TOOL_CALL_END` | `ToolCallEndEvent` | Tool invocation complete (toolCallId) |
| `TOOL_CALL_RESULT` | `ToolCallResultEvent` | Tool execution result (toolCallId, messageId, content, role) |
| `STATE_SNAPSHOT` | `StateSnapshotEvent` | Full state snapshot (JSON element) |
| `STATE_DELTA` | `StateDeltaEvent` | State delta/patch (JSON patch element) |

### 3.5 SSE Format

- `Content-Type: text/event-stream`
- `Cache-Control: no-cache,no-store`
- `Pragma: no-cache`
- No explicit `event:` field per SSE item
- Each `data:` line contains a JSON-serialized `BaseEvent` with `type` discriminator field
- Error handling: if streaming fails, sends a `RUN_ERROR` event before closing
- Uses `SseFormatter.WriteAsync()` from .NET `System.Net.ServerSentEvents`

### 3.6 Server-Side Processing Pipeline

```
RunStreamingAsync(messages, options)       ← AIAgent streaming
    .AsChatResponseUpdatesAsync()          ← convert to ChatResponseUpdate stream
    .FilterServerToolsFromMixedToolInvocationsAsync()  ← separate client/server tools
    .AsAGUIEventStreamAsync(threadId, runId, ...)      ← convert to AG-UI events
```

Key concern: **mixed tool invocations** — AG-UI supports both client-side and server-side tool calls. The `FilterServerToolsFromMixedToolInvocationsAsync` extension strips server tools from mixed sets so only client tools reach the client.

### 3.7 AG-UI Hosting Summary

| Metric | Count |
|--------|-------|
| **HTTP endpoints** | 1 (POST) |
| **SSE event types** | 12 |
| **Message role types** | 5 (user, assistant, system, developer, tool) |
| **Request model fields** | 7 |
| **Shared model types** | ~20 (events + messages + tools + context) |
| **JSON serialization** | `AGUIJsonSerializerContext` (source-generated) + `BaseEventJsonConverter` + `AGUIMessageJsonConverter` |

---

## 4. A2A Client (`Microsoft.Agents.AI.A2A`)

### 4.1 Overview

`A2AAgent` extends `AIAgent` to invoke **remote** A2A-protocol agents as a client.

### 4.2 Key Components

| Type | Purpose |
|------|---------|
| `A2AAgent` | `AIAgent` subclass wrapping `A2AClient` |
| `A2AAgentSession` | Session with `ContextId` + `TaskId` for conversation continuity |
| `A2AContinuationToken` | Token for resuming background task polling |
| `A2AJsonUtilities` | JSON serialization options |

### 4.3 Operations

**`RunCoreAsync` (non-streaming):**
1. If continuation token present → `_a2aClient.GetTaskAsync(taskId)`
2. Otherwise → `_a2aClient.SendMessageAsync(sendParams)`
3. Response is `AgentMessage` or `AgentTask` → converted to `AgentResponse`

**`RunCoreStreamingAsync` (streaming):**
1. `_a2aClient.SendMessageStreamingAsync(sendParams)` → SSE stream
2. Each SSE event is `AgentMessage`, `AgentTask`, or `TaskUpdateEvent`
3. Converted to `AgentResponseUpdate` via helper methods

### 4.4 Extension Methods (9 files)

| Extension | Purpose |
|-----------|---------|
| `A2AAgentCardExtensions` | AgentCard ↔ framework conversion |
| `A2AAgentTaskExtensions` | AgentTask → ChatMessage[] conversion |
| `A2AAIContentExtensions` | A2A parts ↔ AIContent conversion |
| `A2AArtifactExtensions` | Artifact → AIContent[] conversion |
| `A2ACardResolverExtensions` | Dynamic agent card resolution |
| `A2AClientExtensions` | `A2AClient.AsAIAgent()` convenience |
| `A2AMetadataExtensions` | Metadata ↔ AdditionalProperties conversion |
| `ChatMessageExtensions` | ChatMessage → A2A AgentMessage conversion |
| `AdditionalPropertiesDictionaryExtensions` | Property bridge |

### 4.5 A2A Client Summary

| Metric | Count |
|--------|-------|
| **Key classes** | 4 (A2AAgent, A2AAgentSession, A2AContinuationToken, A2AJsonUtilities) |
| **Extension files** | 9 |
| **External NuGet dep** | 1 (`A2A`) |
| **Agent operations** | 2 (RunCore, RunCoreStreaming) |

---

## 5. AGUI Client (`Microsoft.Agents.AI.AGUI`)

### 5.1 Overview

`AGUIChatClient` implements `IChatClient` (from `Microsoft.Extensions.AI`) to communicate with remote AG-UI servers.

### 5.2 Key Components

| Type | Purpose |
|------|---------|
| `AGUIChatClient` | `DelegatingChatClient` wrapping inner handler |
| `AGUIChatClientHandler` | Inner `IChatClient` that performs HTTP POST + SSE parsing |
| `AGUIHttpService` | HTTP POST + SSE stream parsing |
| `FunctionInvokingChatClient` | Wraps handler for client-side tool execution |
| `ServerFunctionCallContent` | Marker to hide server tools from `FunctionInvokingChatClient` |

### 5.3 Communication Flow

```
AGUIChatClient
    → FunctionInvokingChatClient (handles client tool calls)
        → AGUIChatClientHandler
            → AGUIHttpService.PostRunAsync()
                → HTTP POST to AG-UI endpoint
                → SSE response parsed via SseParser
                → BaseEvent stream
            → .AsChatResponseUpdatesAsync() (converts events to ChatResponseUpdate)
```

### 5.4 Thread ID / Conversation Management

- AG-UI requires full message history on every turn (no server-side history)
- Thread ID is propagated via `ChatOptions.AdditionalProperties["agui_thread_id"]`
- Thread ID is also temporarily stored on `FunctionCallContent.AdditionalProperties` to survive `FunctionInvokingChatClient` round-trips
- `ConversationId` is cleared before yielding to `FunctionInvokingChatClient` (forces full history send)

### 5.5 State Management

- State is extracted from the last message's `DataContent` (application/json media type)
- Extracted state is removed from messages and placed in `RunAgentInput.State`

### 5.6 AGUI Client Summary

| Metric | Count |
|--------|-------|
| **Key classes** | 5 (AGUIChatClient, handler, HttpService, ServerFunctionCallContent, shared models) |
| **Shared model types** | ~20 (reused from Shared/) |
| **HTTP operations** | 1 (POST) |
| **SSE parsing** | Uses `SseParser.Create()` with custom `ItemParser` |

---

## 6. Cross-Protocol Comparison

| Dimension | OpenAI-Compatible | A2A | AG-UI |
|-----------|-------------------|-----|-------|
| **HTTP Endpoints** | 14 | 2 | 1 |
| **Transport** | REST + SSE | JSON-RPC 2.0 + SSE | POST → SSE |
| **SSE Event Types** | 19 (Responses) + chunks (Chat) | 3 (via A2A SDK) | 12 |
| **Request Complexity** | High (40+ model types) | Medium (via A2A SDK) | Low-Medium (~20 types) |
| **Multi-turn** | previous_response_id / conversations | contextId + referenceTaskIds | threadId + full history |
| **Tool Support** | Yes (function calling) | Via message parts | Client + server tools |
| **State Management** | Server-side (IConversationStorage) | Server-side (session store) | Client-side (state snapshot/delta) |
| **Background/Long-running** | Yes (background=true) | Yes (tasks with state machine) | No |
| **External Dependencies** | None (self-contained) | A2A, A2A.AspNetCore NuGets | None (self-contained) |

---

## 7. Complexity Assessment for C++ Port

### 7.1 Difficulty Ratings

| Component | Complexity | Rationale |
|-----------|------------|-----------|
| **SSE Streaming Infrastructure** | **HIGH** | Core to all protocols. Need async generator/coroutine support, chunked transfer encoding, proper connection management. C++ lacks `IAsyncEnumerable<T>` — need coroutine-based equivalent (C++20 coroutines or callback chains). |
| **JSON Serialization** | **HIGH** | Extensive polymorphic JSON. OpenAI alone has 40+ types with discriminator-based deserialization. Need nlohmann/json or simdjson with custom type handling. Source-generated contexts not available. |
| **OpenAI Chat Completions** | **MEDIUM** | Single endpoint, well-defined request/response. The streaming chunk format is straightforward. |
| **OpenAI Responses API** | **HIGH** | 5 endpoints, 19 streaming event types, complex state management (response lifecycle), pagination, conversation threading. Most complex single surface. |
| **OpenAI Conversations API** | **MEDIUM** | 8 REST endpoints, no streaming. Straightforward CRUD. Storage abstraction needed. |
| **A2A Hosting** | **MEDIUM-HIGH** | JSON-RPC 2.0 handler needed (or port the A2A C# SDK). Agent card discovery protocol. Task state machine. SSE for streaming. The A2A C# SDK handles most complexity — equivalent needed in C++. |
| **A2A Client** | **MEDIUM** | HTTP client with SSE parsing. Continuation token handling. Session management. Depends on A2A protocol types. |
| **AG-UI Hosting** | **MEDIUM** | Single endpoint, 12 event types. Mixed tool filtering logic is nuanced. SSE serialization straightforward. |
| **AG-UI Client** | **MEDIUM-HIGH** | HTTP + SSE parsing, `FunctionInvokingChatClient` wrapper pattern (auto tool execution), thread ID propagation, state extraction. The delegating/wrapping pattern is complex to port. |

### 7.2 Key C++ Challenges

1. **Async streaming** — All three protocols rely heavily on `IAsyncEnumerable<T>`. C++20 coroutines (`co_yield`) or Boost.Asio-based streaming needed.
2. **SSE formatting/parsing** — .NET's `SseFormatter`/`SseParser` with typed items. Need equivalent (possibly custom, ~200 LOC).
3. **JSON polymorphism** — `JsonPolymorphic` with type discriminators used extensively. Manual dispatch tables in C++.
4. **HTTP server framework** — ASP.NET Core minimal APIs → likely cpp-httplib, Crow, or Drogon.
5. **DI/Service resolution** — Keyed services, `IServiceProvider` patterns heavily used. Need a DI container or factory pattern.
6. **No A2A C++ SDK exists** — The entire A2A protocol layer (JSON-RPC 2.0, agent card, task lifecycle) must be implemented from scratch or ported.

### 7.3 Estimated Type Counts for C++ Port

| Category | Approximate Count |
|----------|-------------------|
| **OpenAI model structs** | ~50 |
| **OpenAI service interfaces** | ~4 |
| **A2A protocol types** | ~15 (if porting SDK) |
| **AG-UI event/message types** | ~20 |
| **Shared infrastructure** | ~10 (SSE, JSON, HTTP) |
| **Total estimated C++ types** | ~100 |

### 7.4 Recommended Port Order

1. **SSE infrastructure** (shared by all) → SSE formatter + parser
2. **AG-UI hosting** (simplest: 1 endpoint, 12 events) → proof of concept
3. **OpenAI Chat Completions** (1 endpoint, well-known format) → broad compatibility
4. **AG-UI client** (HTTP + SSE parsing) → enables agent composition
5. **OpenAI Responses API** (most complex surface) → full OpenAI compatibility
6. **OpenAI Conversations API** (REST CRUD) → state management
7. **A2A hosting + client** (requires protocol reimplementation) → agent-to-agent

---

## 8. File Inventory

### OpenAI Hosting (22 source files)
- 3 endpoint route builders (ChatCompletions, Responses, Conversations)
- 1 SSE result type (`SseJsonResult<T>`)
- 1 processor (`AIAgentChatCompletionsProcessor`)
- 1 handler (`ResponsesHttpHandler`)
- 1 handler (`ConversationsHttpHandler`)
- 12 chat completions models, 16 responses models, 4 conversation models
- 2 JSON contexts, converters, service interfaces, DI extensions
- 11 streaming event generators

### A2A Hosting (3 source files in AspNetCore + ~10 in Hosting.A2A)
- 1 endpoint route builder
- 1 agent extension (`AIAgentExtensions`)
- 1 host agent wrapper (`AIHostAgent`)
- Session store, converters, metadata extensions

### AG-UI Hosting (6 source files)
- 1 endpoint route builder
- 1 SSE result type (`AGUIServerSentEventsResult`)
- 1 stream filter extension
- 1 JSON serializer options, 1 service collection extensions

### A2A Client (6 source files + 9 extensions)
- `A2AAgent`, `A2AAgentSession`, `A2AContinuationToken`, `A2AJsonUtilities`
- 9 extension method files for type conversions

### AGUI Client (3 source files + ~20 shared types)
- `AGUIChatClient`, `AGUIHttpService`
- Shared: 12 event classes, 5 message classes, tool/context types, JSON converters
