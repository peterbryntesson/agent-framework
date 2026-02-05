# Go Port Epic 1 - Python Parity Checklist

> **Research Date**: 2026-02-02
> **Based on**: Python `agent_framework` core package (`python/packages/core/agent_framework/`)

This checklist documents all Python core abstractions that should have equivalents in the Go port for Epic 1 (Core Abstractions).

---

## Table of Contents

1. [Agent Protocol](#1-agent-protocol)
2. [Chat Client Protocol](#2-chat-client-protocol)
3. [Base Types](#3-base-types)
4. [Content Types](#4-content-types)
5. [Response Types](#5-response-types)
6. [Session/Thread Management](#6-sessionthread-management)
7. [Tool Types](#7-tool-types)
8. [Options/Config Types](#8-optionsconfig-types)
9. [Error Types](#9-error-types)
10. [Serialization](#10-serialization)
11. [Utilities](#11-utilities)

---

## 1. Agent Protocol

### Python: `AgentProtocol` (Protocol class in `_agents.py`)

| Feature | Python | Go Status | Go Location | Notes |
|---------|--------|-----------|-------------|-------|
| **Properties** |||||
| `id: str` | ✅ | ✅ | `agent.Agent.ID()` | |
| `name: str \| None` | ✅ | ✅ | `agent.Agent.Name()` | |
| `description: str \| None` | ✅ | ✅ | `agent.Agent.Description()` | |
| **Methods** |||||
| `run(messages, thread, **kwargs) -> AgentResponse` | ✅ | ✅ | `agent.Agent.Run()` | |
| `run_stream(messages, thread, **kwargs) -> AsyncIterable[AgentResponseUpdate]` | ✅ | ✅ | `agent.Agent.RunStream()` | |
| `get_new_thread(**kwargs) -> AgentThread` | ✅ | ✅ | `agent.Agent.NewSession()` | Go uses Session term |
| **Python-only** |||||
| `@runtime_checkable` decorator | ✅ | N/A | N/A | Go uses interfaces |

### Python: `BaseAgent` (Abstract class in `_agents.py`)

| Feature | Python | Go Status | Notes |
|---------|--------|-----------|-------|
| `__init__(id, name, description, context_provider, middleware, additional_properties)` | ✅ | ⚠️ Partial | Go lacks base implementation |
| `context_provider: ContextProvider` | ✅ | ❌ Missing | |
| `middleware: list[Middleware]` | ✅ | ❌ Missing | |
| `additional_properties: dict[str, Any]` | ✅ | ❌ Missing | |
| `_notify_thread_of_new_messages()` | ✅ | ❌ Missing | |
| `deserialize_thread()` | ✅ | ✅ | `agent.Agent.RestoreSession()` |
| `as_tool()` | ✅ | ❌ Missing | Agent-as-tool pattern |
| `SerializationMixin` inheritance | ✅ | ❌ Missing | |

### Python: `ChatAgent` (Concrete class in `_agents.py`)

| Feature | Python | Go Status | Notes |
|---------|--------|-----------|-------|
| `chat_client: ChatClientProtocol` | ✅ | ❌ Missing | Go has no ChatClientAgent yet |
| `instructions: str` | ✅ | ❌ Missing | |
| `tools: list[ToolProtocol]` | ✅ | ❌ Missing | |
| `default_options: dict` | ✅ | ❌ Missing | |
| `mcp_tools: list[MCPTool]` | ✅ | ❌ Missing | |
| `chat_message_store_factory` | ✅ | ❌ Missing | |
| Context manager support (`__aenter__`, `__aexit__`) | ✅ | ❌ Missing | |
| Overloaded `run()` with typed options | ✅ | N/A | Go lacks overloads |

---

## 2. Chat Client Protocol

### Python: `ChatClientProtocol` (Protocol class in `_clients.py`)

| Feature | Python | Go Status | Go Location | Notes |
|---------|--------|-----------|-------------|-------|
| **Properties** |||||
| `additional_properties: dict[str, Any]` | ✅ | ❌ Missing | | |
| **Methods** |||||
| `get_response(messages, options, **kwargs) -> ChatResponse` | ✅ | ✅ | `chat.Client.GetResponse()` | |
| `get_streaming_response(messages, options, **kwargs) -> AsyncIterable[ChatResponseUpdate]` | ✅ | ✅ | `chat.Client.GetStreamingResponse()` | |
| **Go-only** |||||
| `Metadata() ClientMetadata` | N/A | ✅ | `chat.Client.Metadata()` | Good addition |

### Python: `BaseChatClient` (Abstract class in `_clients.py`)

| Feature | Python | Go Status | Notes |
|---------|--------|-----------|-------|
| `OTEL_PROVIDER_NAME: ClassVar[str]` | ✅ | ⚠️ Partial | Go has `ClientMetadata.ProviderName` |
| `middleware: Sequence[ChatMiddleware \| FunctionMiddleware]` | ✅ | ❌ Missing | |
| `function_invocation_configuration` | ✅ | ❌ Missing | |
| `_inner_get_response()` abstract | ✅ | N/A | Go pattern differs |
| `_inner_get_streaming_response()` abstract | ✅ | N/A | Go pattern differs |
| `service_url()` | ✅ | ✅ | `ClientMetadata.EndpointURI` |
| `as_agent()` convenience method | ✅ | ❌ Missing | |
| `SerializationMixin` inheritance | ✅ | ❌ Missing | |

### Go: `chat.ClientMetadata` (Struct)

| Field | Go | Python Equivalent | Notes |
|-------|-----|-------------------|-------|
| `ProviderName` | ✅ | `OTEL_PROVIDER_NAME` | |
| `ModelID` | ✅ | Via options | |
| `EndpointURI` | ✅ | `service_url()` | |

---

## 3. Base Types

### Python: `Role` (Class in `_types.py`)

| Feature | Python | Go Status | Go Location | Notes |
|---------|--------|-----------|-------------|-------|
| `SYSTEM` | ✅ | ✅ | `chat.RoleSystem` | |
| `USER` | ✅ | ✅ | `chat.RoleUser` | |
| `ASSISTANT` | ✅ | ✅ | `chat.RoleAssistant` | |
| `TOOL` | ✅ | ✅ | `chat.RoleTool` | |
| Custom role support | ✅ | ❌ Missing | Python allows `Role("custom")` |
| `value` property | ✅ | N/A | Go uses string type |
| `__eq__`, `__hash__` | ✅ | N/A | Go uses string comparison |

### Python: `FinishReason` (Class in `_types.py`)

| Feature | Python | Go Status | Go Location | Notes |
|---------|--------|-----------|-------------|-------|
| `STOP` | ✅ | ✅ | `chat.FinishReasonStop` | |
| `LENGTH` | ✅ | ✅ | `chat.FinishReasonLength` | |
| `TOOL_CALLS` | ✅ | ✅ | `chat.FinishReasonToolCalls` | |
| `CONTENT_FILTER` | ✅ | ✅ | `chat.FinishReasonContentFilter` | |
| Custom reason support | ✅ | ❌ Missing | |

### Python: `UsageDetails` (TypedDict in `_types.py`)

| Field | Python | Go Status | Go Location | Notes |
|-------|--------|-----------|-------------|-------|
| `input_token_count` | ✅ | ✅ | `InputTokens` | Different naming |
| `output_token_count` | ✅ | ✅ | `OutputTokens` | Different naming |
| `total_token_count` | ✅ | ✅ | `TotalTokens` | Different naming |
| `CachedTokens` | ❌ | ✅ | | Go has extra |
| `ReasoningTokens` | ❌ | ✅ | | Go has extra |
| Arbitrary extra fields | ✅ | ❌ | | Python TypedDict is extensible |

### Python: `add_usage_details()` (Function)

| Feature | Python | Go Status | Notes |
|---------|--------|-----------|-------|
| `add_usage_details(usage1, usage2) -> UsageDetails` | ✅ | ❌ Missing | Sums token counts |

---

## 4. Content Types

### Python: `Content` (Unified class in `_types.py`)

Python uses a single unified `Content` class with different `type` discriminator.

| Content Type | Python Factory | Go Equivalent | Go Status | Notes |
|--------------|----------------|---------------|-----------|-------|
| `text` | `Content.from_text()` | `TextContent` | ✅ | |
| `text_reasoning` | `Content.from_text_reasoning()` | ❌ | ❌ Missing | |
| `data` | `Content.from_data()` | `ImageContent` (base64) | ⚠️ Partial | Go only handles images |
| `uri` | `Content.from_uri()` | `ImageContent` (URL) | ⚠️ Partial | Go only handles images |
| `error` | `Content.from_error()` | ❌ | ❌ Missing | |
| `function_call` | `Content.from_function_call()` | `ToolCallContent` | ✅ | |
| `function_result` | `Content.from_function_result()` | `ToolResultContent` | ✅ | |
| `usage` | `Content.from_usage()` | ❌ | ❌ Missing | Embedded in UsageDetails |
| `hosted_file` | `Content.from_hosted_file()` | ❌ | ❌ Missing | |
| `hosted_vector_store` | `Content.from_hosted_vector_store()` | ❌ | ❌ Missing | |
| `code_interpreter_tool_call` | `Content.from_code_interpreter_tool_call()` | ❌ | ❌ Missing | |
| `code_interpreter_tool_result` | `Content.from_code_interpreter_tool_result()` | ❌ | ❌ Missing | |
| `image_generation_tool_call` | `Content.from_image_generation_tool_call()` | ❌ | ❌ Missing | |
| `image_generation_tool_result` | `Content.from_image_generation_tool_result()` | ❌ | ❌ Missing | |
| `mcp_server_tool_call` | `Content.from_mcp_server_tool_call()` | ❌ | ❌ Missing | |
| `mcp_server_tool_result` | `Content.from_mcp_server_tool_result()` | ❌ | ❌ Missing | |
| `function_approval_request` | `Content.from_function_approval_request()` | ❌ | ❌ Missing | |
| `function_approval_response` | `Content.from_function_approval_response()` | ❌ | ❌ Missing | |

### Python Content Common Fields

| Field | Python | Go Equivalent | Status |
|-------|--------|---------------|--------|
| `type` | ✅ | `Type()` method | ✅ |
| `annotations` | ✅ | ❌ | ❌ Missing |
| `additional_properties` | ✅ | ❌ | ❌ Missing |
| `raw_representation` | ✅ | ❌ | ❌ Missing |

### Python Content Methods

| Method | Python | Go Equivalent | Status |
|--------|--------|---------------|--------|
| `to_dict()` | ✅ | ❌ | ❌ Missing |
| `from_dict()` | ✅ | ❌ | ❌ Missing |
| `__add__()` content merging | ✅ | ❌ | ❌ Missing |
| `__eq__()` | ✅ | ❌ | ❌ Missing |
| `__str__()` | ✅ | ❌ | ❌ Missing |
| `has_top_level_media_type()` | ✅ | ❌ | ❌ Missing |
| `parse_arguments()` | ✅ | ❌ | ❌ Missing |

### Go Content Types (existing)

| Type | Fields | Notes |
|------|--------|-------|
| `TextContent` | `Text` | Basic text |
| `ImageContent` | `URL`, `Base64Data`, `MediaType`, `Detail` | Image only |
| `ToolCallContent` | `ToolCall{ID, Name, Arguments}` | |
| `ToolResultContent` | `ToolCallID`, `Content`, `IsError` | |

### Python: `Annotation` (TypedDict in `_types.py`)

| Field | Python | Go Status | Notes |
|-------|--------|-----------|-------|
| `type: Literal["citation"]` | ✅ | ❌ Missing | |
| `title`, `url`, `file_id`, `tool_name`, `snippet` | ✅ | ❌ Missing | |
| `annotated_regions: Sequence[TextSpanRegion]` | ✅ | ❌ Missing | |
| `additional_properties`, `raw_representation` | ✅ | ❌ Missing | |

### Python: `TextSpanRegion` (TypedDict)

| Field | Python | Go Status |
|-------|--------|-----------|
| `type: Literal["text_span"]` | ✅ | ❌ Missing |
| `start_index: int` | ✅ | ❌ Missing |
| `end_index: int` | ✅ | ❌ Missing |

---

## 5. Response Types

### Python: `ChatMessage` (Class in `_types.py`)

| Feature | Python | Go Equivalent | Status | Notes |
|---------|--------|---------------|--------|-------|
| **Fields** |||||
| `role: Role` | ✅ | `Message.Role` | ✅ | |
| `contents: list[Content]` | ✅ | `Message.Contents` | ✅ | |
| `author_name: str \| None` | ✅ | `Message.Name` | ✅ | |
| `message_id: str \| None` | ✅ | ❌ | ❌ Missing | |
| `additional_properties: dict` | ✅ | ❌ | ❌ Missing | |
| `raw_representation` | ✅ | `Message.RawRepresentation` | ✅ | |
| **Properties** |||||
| `.text` property | ✅ | `Message.Text()` | ✅ | |
| **Constructors** |||||
| `__init__(role, text=...)` | ✅ | `NewUserMessage()` etc. | ✅ | |
| `__init__(role, contents=...)` | ✅ | `NewMessageWithContents()` | ✅ | |
| **Message fields (Go extra)** |||||
| `ToolCalls []ToolCall` | ❌ | ✅ | Go extra | Python uses Contents |
| `ToolCallID string` | ❌ | ✅ | Go extra | Python uses Contents |
| `CreatedAt time.Time` | ❌ | ✅ | Go extra | |

### Python: `ChatResponse` (Class in `_types.py`)

| Feature | Python | Go Equivalent | Status | Notes |
|---------|--------|---------------|--------|-------|
| **Fields** |||||
| `messages: list[ChatMessage]` | ✅ | `Response.Message` | ⚠️ | Go has single message |
| `response_id: str` | ✅ | ❌ | ❌ Missing | |
| `conversation_id: str` | ✅ | ❌ | ❌ Missing | |
| `model_id: str` | ✅ | ❌ | ❌ Missing | |
| `created_at: str` | ✅ | ❌ | ❌ Missing | |
| `finish_reason: FinishReason` | ✅ | `Response.FinishReason` | ✅ | |
| `usage_details: UsageDetails` | ✅ | `Response.Usage` | ✅ | |
| `structured_output / value` | ✅ | ❌ | ❌ Missing | Pydantic model parsing |
| `additional_properties` | ✅ | ❌ | ❌ Missing | |
| `raw_representation` | ✅ | `Response.RawRepresentation` | ✅ | |
| **Properties** |||||
| `.text` | ✅ | `Response.Text()` | ✅ | |
| `.value` (typed output) | ✅ | ❌ | ❌ Missing | |
| **Factory Methods** |||||
| `from_chat_response_updates()` | ✅ | ❌ | ❌ Missing | Combines stream |
| `from_chat_response_generator()` | ✅ | ❌ | ❌ Missing | Async version |
| **Methods** |||||
| `try_parse_value()` | ✅ | ❌ | ❌ Missing | |

### Python: `ChatResponseUpdate` (Class in `_types.py`)

| Feature | Python | Go Equivalent | Status | Notes |
|---------|--------|---------------|--------|-------|
| `contents: list[Content]` | ✅ | `ResponseUpdate.Delta` | ⚠️ | Go uses Delta struct |
| `role: Role` | ✅ | `ResponseUpdate.Delta.Role` | ⚠️ | Nested |
| `author_name: str` | ✅ | ❌ | ❌ Missing | |
| `response_id: str` | ✅ | ❌ | ❌ Missing | |
| `message_id: str` | ✅ | ❌ | ❌ Missing | |
| `conversation_id: str` | ✅ | ❌ | ❌ Missing | |
| `model_id: str` | ✅ | ❌ | ❌ Missing | |
| `created_at: str` | ✅ | ❌ | ❌ Missing | |
| `finish_reason: FinishReason` | ✅ | `ResponseUpdate.FinishReason` | ✅ | |
| `.text` property | ✅ | ❌ | ❌ Missing | |

### Go: `ResponseUpdate` (Struct)

| Feature | Go | Python Equivalent | Notes |
|---------|-----|-------------------|-------|
| `Kind UpdateKind` | ✅ | Inferred from content | |
| `Delta *ContentDelta` | ✅ | `contents` list | Different structure |
| `Message *Message` | ✅ | Inferred from contents | |
| `Usage *UsageDetails` | ✅ | Inferred from contents | |
| `Error error` | ✅ | ❌ | Error in stream |
| `Metadata` | ✅ | `additional_properties` | |

### Go: `UpdateKind` (Constants)

| Constant | Go | Python Equivalent |
|----------|-----|-------------------|
| `UpdateKindContentDelta` | ✅ | Inferred from Content type |
| `UpdateKindToolCall` | ✅ | Content type `function_call` |
| `UpdateKindToolResult` | ✅ | Content type `function_result` |
| `UpdateKindMessageComplete` | ✅ | ❌ Not explicit in Python |
| `UpdateKindUsage` | ✅ | Content type `usage` |
| `UpdateKindError` | ✅ | Exception handling |
| `UpdateKindDone` | ✅ | Stream exhausted |

### Python: `AgentResponse` (Class in `_types.py`)

| Feature | Python | Go Equivalent | Status | Notes |
|---------|--------|---------------|--------|-------|
| `messages: list[ChatMessage]` | ✅ | `Response.Messages` | ✅ | |
| `response_id: str` | ✅ | ❌ | ❌ Missing | |
| `created_at: str` | ✅ | ❌ | ❌ Missing | |
| `usage_details: UsageDetails` | ✅ | `Response.Usage` | ✅ | |
| `value` (structured output) | ✅ | ❌ | ❌ Missing | |
| `.text` property | ✅ | `Response.Text()` | ✅ | |
| `.user_input_requests` property | ✅ | ❌ | ❌ Missing | |
| `from_agent_run_response_updates()` | ✅ | ❌ | ❌ Missing | |
| `from_agent_response_generator()` | ✅ | ❌ | ❌ Missing | |
| `try_parse_value()` | ✅ | ❌ | ❌ Missing | |

### Go Extra: `agent.Response`

| Field | Go | Python Equivalent | Notes |
|-------|-----|-------------------|-------|
| `SessionState json.RawMessage` | ✅ | Via thread serialization | |
| `ContinuationToken string` | ✅ | ❌ | For resumable runs |
| `AdditionalProperties` | ✅ | `additional_properties` | |

### Python: `AgentResponseUpdate` (Class in `_types.py`)

| Feature | Python | Go Equivalent | Status |
|---------|--------|---------------|--------|
| Similar structure to `ChatResponseUpdate` | ✅ | `agent.ResponseUpdate` | ⚠️ Partial |
| `.user_input_requests` property | ✅ | ❌ | ❌ Missing |

---

## 6. Session/Thread Management

### Python: `AgentThread` (Class in `_threads.py`)

| Feature | Python | Go Equivalent | Status | Notes |
|---------|--------|---------------|--------|-------|
| `id: str` | ✅ | `Session.ID()` | ✅ | |
| `service_thread_id: str` | ✅ | ❌ | ❌ Missing | For hosted threads |
| `chat_message_store: ChatMessageStoreProtocol` | ✅ | ❌ | ❌ Missing | |
| `context_provider: ContextProvider` | ✅ | ❌ | ❌ Missing | |
| `on_new_messages()` | ✅ | `Session.AddMessage()` | ⚠️ | Different pattern |
| `serialize()` | ✅ | `Session.Serialize()` | ✅ | |
| `update_from_thread_state()` | ✅ | Via `RestoreSession` | ⚠️ | |

### Python: `ChatMessageStoreProtocol` (Protocol in `_threads.py`)

| Method | Python | Go Status | Notes |
|--------|--------|-----------|-------|
| `list_messages() -> list[ChatMessage]` | ✅ | `Session.Messages()` | ✅ Equivalent |
| `add_messages(messages)` | ✅ | `Session.AddMessage()` | ⚠️ Single vs batch |
| `deserialize()` | ✅ | `RestoreInMemorySession()` | ✅ |
| `update_from_state()` | ✅ | ❌ | ❌ Missing |
| `serialize()` | ✅ | `Session.Serialize()` | ✅ |

### Python: `ChatMessageStore` (Implementation)

Equivalent to Go's `InMemorySession` - both are in-memory implementations.

### Go: `Session` Interface

| Method | Go | Python Equivalent |
|--------|-----|-------------------|
| `ID()` | ✅ | `AgentThread.id` |
| `Messages()` | ✅ | `chat_message_store.list_messages()` |
| `AddMessage(msg)` | ✅ | `on_new_messages()` |
| `Serialize()` | ✅ | `serialize()` |
| `GetService(type)` | ✅ | ❌ Not in Python |

### Go: `InMemorySession` Implementation

| Feature | Go | Notes |
|---------|-----|-------|
| Thread-safe via `sync.RWMutex` | ✅ | Good |
| `RegisterService()` | ✅ | Extensibility |
| `RestoreInMemorySession()` | ✅ | |

---

## 7. Tool Types

### Python: `ToolProtocol` (Protocol in `_tools.py`)

| Feature | Python | Go Status | Notes |
|---------|--------|-----------|-------|
| `name: str` | ✅ | ❌ Missing | No Tool interface in Go |
| `description: str` | ✅ | ❌ Missing | |
| `additional_properties: dict` | ✅ | ❌ Missing | |

### Python: `FunctionTool` (Class in `_tools.py`)

| Feature | Python | Go Status | Notes |
|---------|--------|-----------|-------|
| `name`, `description`, `func` | ✅ | ❌ Missing | |
| `input_model: BaseModel` | ✅ | ❌ Missing | JSON schema |
| `approval_mode` | ✅ | ❌ Missing | |
| Decorator pattern (`@tool`) | ✅ | ❌ Missing | |

### Python: Hosted Tool Types

| Type | Python | Go Status |
|------|--------|-----------|
| `HostedFileSearchTool` | ✅ | ❌ Missing |
| `HostedCodeInterpreterTool` | ✅ | ❌ Missing |
| `HostedWebSearchTool` | ✅ | ❌ Missing |
| `HostedMCPTool` | ✅ | ❌ Missing |
| `HostedImageGenerationTool` | ✅ | ❌ Missing |

### Go: `ToolCall` (Struct in `chat/content.go`)

| Feature | Go | Python Equivalent |
|---------|-----|-------------------|
| `ID` | ✅ | `Content.call_id` |
| `Name` | ✅ | `Content.name` |
| `Arguments json.RawMessage` | ✅ | `Content.arguments` |

---

## 8. Options/Config Types

### Python: `ChatOptions` (TypedDict in `_types.py`)

| Field | Python | Go `chat.Options` | Status |
|-------|--------|-------------------|--------|
| `model_id: str` | ✅ | ❌ | ❌ Missing |
| `temperature: float` | ✅ | `Temperature` | ✅ |
| `top_p: float` | ✅ | `TopP` | ✅ |
| `max_tokens: int` | ✅ | `MaxTokens` | ✅ |
| `stop: str \| Sequence[str]` | ✅ | `StopSequences` | ✅ |
| `seed: int` | ✅ | ❌ | ❌ Missing |
| `logit_bias: dict` | ✅ | ❌ | ❌ Missing |
| `frequency_penalty: float` | ✅ | ❌ | ❌ Missing |
| `presence_penalty: float` | ✅ | ❌ | ❌ Missing |
| `tools: ...` | ✅ | ❌ | ❌ Missing |
| `tool_choice: ToolMode` | ✅ | ❌ | ❌ Missing |
| `allow_multiple_tool_calls: bool` | ✅ | ❌ | ❌ Missing |
| `response_format: type[BaseModel]` | ✅ | `ResponseFormat` | ⚠️ String only |
| `metadata: dict` | ✅ | `Metadata` | ✅ |
| `user: str` | ✅ | ❌ | ❌ Missing |
| `store: bool` | ✅ | ❌ | ❌ Missing |
| `conversation_id: str` | ✅ | ❌ | ❌ Missing |
| `instructions: str` | ✅ | ❌ | ❌ Missing |

### Python: `ToolMode` (TypedDict)

| Field | Python | Go Status |
|-------|--------|-----------|
| `mode: Literal["auto", "required", "none"]` | ✅ | ❌ Missing |
| `required_function_name: str` | ✅ | ❌ Missing |

### Go: `agent.RunConfig` and `RunOption`s

| Field | Go | Python Equivalent |
|-------|-----|-------------------|
| `Session` | ✅ | `thread` parameter |
| `Metadata` | ✅ | `metadata` in options |
| `Tools` | ✅ | `tools` in options |
| `MaxTokens` | ✅ | `max_tokens` |
| `Temperature` | ✅ | `temperature` |

### Python Utilities

| Function | Python | Go Status |
|----------|--------|-----------|
| `validate_chat_options()` | ✅ | ❌ Missing |
| `normalize_tools()` | ✅ | ❌ Missing |
| `validate_tools()` | ✅ | ❌ Missing |
| `validate_tool_mode()` | ✅ | ❌ Missing |
| `merge_chat_options()` | ✅ | ❌ Missing |

---

## 9. Error Types

### Python Exceptions (`exceptions.py`)

| Exception | Python | Go Equivalent | Status |
|-----------|--------|---------------|--------|
| `AgentFrameworkException` | ✅ | `agent.Error` | ⚠️ Similar |
| `AgentException` | ✅ | `agent.Error` | ⚠️ |
| `AgentExecutionException` | ✅ | ❌ | ❌ Missing |
| `AgentInitializationError` | ✅ | ❌ | ❌ Missing |
| `AgentThreadException` | ✅ | `ErrSessionNotFound` | ⚠️ |
| `ChatClientException` | ✅ | `ErrProviderError` | ⚠️ |
| `ChatClientInitializationError` | ✅ | ❌ | ❌ Missing |
| `ServiceException` | ✅ | `ErrProviderError` | ⚠️ |
| `ServiceInitializationError` | ✅ | ❌ | ❌ Missing |
| `ServiceResponseException` | ✅ | `ErrProviderError` | ⚠️ |
| `ServiceContentFilterException` | ✅ | ❌ | ❌ Missing |
| `ServiceInvalidAuthError` | ✅ | ❌ | ❌ Missing |
| `ContentError` | ✅ | ❌ | ❌ Missing |
| `ToolException` | ✅ | `ErrToolInvocationFailed` | ✅ |

### Go Sentinel Errors (`agent/errors.go`)

| Error | Go | Python Equivalent |
|-------|-----|-------------------|
| `ErrSessionNotFound` | ✅ | `AgentThreadException` |
| `ErrInvalidInput` | ✅ | Validation errors |
| `ErrRateLimited` | ✅ | ❌ Not explicit |
| `ErrProviderError` | ✅ | `ServiceException` family |
| `ErrToolInvocationFailed` | ✅ | `ToolException` |

### Go Error Utilities

| Function | Go | Python Equivalent |
|----------|-----|-------------------|
| `IsRetryable(err)` | ✅ | ❌ Not in Python |
| `Error.Unwrap()` | ✅ | Exception chaining |

---

## 10. Serialization

### Python: `SerializationMixin` (`_serialization.py`)

| Feature | Python | Go Status | Notes |
|---------|--------|-----------|-------|
| `to_dict()` | ✅ | ❌ Missing | Convert to map |
| `to_json()` | ✅ | `json.Marshal()` | Native |
| `from_dict()` | ✅ | ❌ Missing | Type reconstruction |
| `from_json()` | ✅ | `json.Unmarshal()` | Native |
| Nested object serialization | ✅ | ❌ Missing | |
| Dependency injection | ✅ | ❌ Missing | |
| `DEFAULT_EXCLUDE` field exclusion | ✅ | JSON struct tags | Different approach |

### Python: `SerializationProtocol`

| Method | Python | Go Status |
|--------|--------|-----------|
| `to_dict(**kwargs)` | ✅ | ❌ |
| `from_dict(value, **kwargs)` | ✅ | ❌ |

### Go Serialization

Go uses standard `json.Marshal`/`Unmarshal` with struct tags. Missing:
- Custom type reconstruction from discriminators
- Nested object handling for Content types
- Extensible property handling

---

## 11. Utilities

### Python Utilities in `_types.py`

| Function | Python | Go Status | Notes |
|----------|--------|-----------|-------|
| `detect_media_type_from_base64()` | ✅ | ❌ Missing | Magic byte detection |
| `prepare_messages()` | ✅ | ❌ Missing | Message normalization |
| `normalize_messages()` | ✅ | ❌ Missing | |
| `prepend_instructions_to_messages()` | ✅ | ❌ Missing | |
| `prepare_function_call_results()` | ✅ | ❌ Missing | |
| `_parse_content_list()` | ✅ | ❌ Missing | |
| `_process_update()` | ✅ | ❌ Missing | Stream accumulation |
| `_finalize_response()` | ✅ | ❌ Missing | |
| `_coalesce_text_content()` | ✅ | ❌ Missing | Merge text chunks |

### Python Context/Memory (`_memory.py`)

| Feature | Python | Go Status |
|---------|--------|-----------|
| `ContextProvider` | ✅ | ❌ Missing |
| `Context` | ✅ | ❌ Missing |

### Python Middleware (`_middleware.py`)

| Feature | Python | Go Status |
|---------|--------|-----------|
| `Middleware` | ✅ | ❌ Missing |
| `ChatMiddleware` | ✅ | ❌ Missing |
| `FunctionMiddleware` | ✅ | ❌ Missing |

---

## Summary Statistics

### Epic 1 Coverage

| Category | Python Features | Go Implemented | Gap |
|----------|-----------------|----------------|-----|
| Agent Protocol | 15 | 8 | 7 |
| Chat Client | 12 | 5 | 7 |
| Role/FinishReason | 10 | 8 | 2 |
| UsageDetails | 5 | 5 | 0 |
| Content Types | 20+ | 4 | 16+ |
| ChatMessage | 8 | 6 | 2 |
| ChatResponse | 12 | 4 | 8 |
| ChatResponseUpdate | 10 | 5 | 5 |
| AgentResponse | 8 | 4 | 4 |
| Session/Thread | 10 | 6 | 4 |
| Options | 18 | 6 | 12 |
| Errors | 12 | 5 | 7 |
| Serialization | 6 | 2 | 4 |
| Utilities | 15 | 0 | 15 |

### Priority Gaps for Epic 1

1. **High Priority** (Core functionality):
   - [ ] Content types beyond text/image (error, function_call/result parity)
   - [ ] ChatResponse fields (response_id, conversation_id, model_id)
   - [ ] ChatOptions (model_id, tool_choice, instructions)
   - [ ] Response accumulation from stream updates

2. **Medium Priority** (Usability):
   - [ ] Annotation support
   - [ ] Serialization helpers (to_dict/from_dict patterns)
   - [ ] Message normalization utilities
   - [ ] Error type expansion

3. **Lower Priority** (Advanced features):
   - [ ] Tool protocol and FunctionTool
   - [ ] Middleware support
   - [ ] Context providers
   - [ ] Structured output (Pydantic model parsing)

---

*Generated from Python `agent_framework` core package analysis*
