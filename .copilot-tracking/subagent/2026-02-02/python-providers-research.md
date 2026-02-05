# Python Provider Implementations Research

**Date:** 2026-02-02
**Scope:** Agent Framework Python provider implementations

---

## 1. Package Structure

### Root Package Location

```
python/packages/core/agent_framework/
```

### Provider Packages

| Package | Directory | Purpose |
|---------|-----------|---------|
| Core | `python/packages/core/` | Base protocols, types, tools, observability |
| OpenAI | `python/packages/core/agent_framework/openai/` | OpenAI Chat Completions API |
| Azure OpenAI | `python/packages/core/agent_framework/azure/` | Azure OpenAI Service |
| Anthropic | `python/packages/anthropic/` | Claude models via Anthropic API |
| Azure AI | `python/packages/azure-ai/` | Azure AI Foundry Agent Service |
| Bedrock | `python/packages/bedrock/` | AWS Bedrock Converse API |
| Ollama | `python/packages/ollama/` | Local Ollama models |

### Package File Structure Pattern

Each provider package follows this structure:

```
agent_framework_{provider}/
├── __init__.py           # Public exports
├── _chat_client.py       # Main chat client implementation
├── _shared.py            # Settings, utilities, shared code
├── py.typed              # Type hints marker
└── tests/                # Unit tests
```

---

## 2. ChatClient Protocol

### Location

[python/packages/core/agent_framework/_clients.py](python/packages/core/agent_framework/_clients.py#L76-L152)

### Protocol Definition

```python
@runtime_checkable
class ChatClientProtocol(Protocol[TOptions_contra]):
    """A protocol for a chat client that can generate responses."""

    additional_properties: dict[str, Any]

    @overload
    async def get_response(
        self,
        messages: str | ChatMessage | Sequence[str | ChatMessage],
        *,
        options: "ChatOptions[TResponseModelT]",
        **kwargs: Any,
    ) -> "ChatResponse[TResponseModelT]": ...

    @overload
    async def get_response(
        self,
        messages: str | ChatMessage | Sequence[str | ChatMessage],
        *,
        options: TOptions_contra | None = None,
        **kwargs: Any,
    ) -> ChatResponse: ...

    def get_streaming_response(
        self,
        messages: str | ChatMessage | Sequence[str | ChatMessage],
        *,
        options: TOptions_contra | None = None,
        **kwargs: Any,
    ) -> AsyncIterable[ChatResponseUpdate]: ...
```

### Key Methods

| Method | Return Type | Purpose |
|--------|-------------|---------|
| `get_response()` | `ChatResponse` | Non-streaming chat completion |
| `get_streaming_response()` | `AsyncIterable[ChatResponseUpdate]` | Streaming chat completion |

---

## 3. BaseChatClient Abstract Base Class

### Location

[python/packages/core/agent_framework/_clients.py](python/packages/core/agent_framework/_clients.py#L171-L250)

### Class Signature

```python
class BaseChatClient(SerializationMixin, ABC, Generic[TOptions_co]):
    """Base class for chat clients."""

    OTEL_PROVIDER_NAME: ClassVar[str] = "unknown"
    DEFAULT_EXCLUDE: ClassVar[set[str]] = {"additional_properties"}

    def __init__(
        self,
        *,
        middleware: Sequence[ChatMiddleware | ChatMiddlewareCallable | ...] | None = None,
        additional_properties: dict[str, Any] | None = None,
        **kwargs: Any,
    ) -> None: ...
```

### Abstract Methods (must be implemented)

```python
@abstractmethod
async def _inner_get_response(
    self,
    *,
    messages: MutableSequence[ChatMessage],
    options: dict[str, Any],
    **kwargs: Any,
) -> ChatResponse: ...

@abstractmethod
async def _inner_get_streaming_response(
    self,
    *,
    messages: MutableSequence[ChatMessage],
    options: dict[str, Any],
    **kwargs: Any,
) -> AsyncIterable[ChatResponseUpdate]: ...
```

### Utility Methods

| Method | Purpose |
|--------|---------|
| `as_agent()` | Creates a `ChatAgent` from the client |
| `service_url()` | Returns the service URL (override in subclass) |
| `to_dict()` | Serializes client configuration |

---

## 4. ChatOptions TypedDict

### Location

[python/packages/core/agent_framework/_types.py](python/packages/core/agent_framework/_types.py#L2815-L2883)

### Base Definition

```python
class _ChatOptionsBase(TypedDict, total=False):
    """Common request settings for AI services as a TypedDict."""

    # Model selection
    model_id: str

    # Generation parameters
    temperature: float
    top_p: float
    max_tokens: int
    stop: str | Sequence[str]
    seed: int
    logit_bias: dict[str | int, float]

    # Penalty parameters
    frequency_penalty: float
    presence_penalty: float

    # Tool configuration
    tools: "ToolProtocol | Callable[..., Any] | MutableMapping[str, Any] | Sequence[...] | None"
    tool_choice: ToolMode | Literal["auto", "required", "none"]
    allow_multiple_tool_calls: bool

    # Response configuration
    response_format: type[BaseModel] | Mapping[str, Any] | None

    # Metadata
    metadata: dict[str, Any]
    user: str
    store: bool
    conversation_id: str

    # System/instructions
    instructions: str
```

---

## 5. Provider Implementations

### 5.1 OpenAI Chat Client

#### Location

[python/packages/core/agent_framework/openai/_chat_client.py](python/packages/core/agent_framework/openai/_chat_client.py)

#### Class Declaration

```python
@use_function_invocation
@use_instrumentation
@use_chat_middleware
class OpenAIChatClient(OpenAIConfigMixin, OpenAIBaseChatClient[TOpenAIChatOptions], Generic[TOpenAIChatOptions]):
    """OpenAI Chat completion class."""

    def __init__(
        self,
        *,
        model_id: str | None = None,
        api_key: str | Callable[[], str | Awaitable[str]] | None = None,
        org_id: str | None = None,
        default_headers: Mapping[str, str] | None = None,
        async_client: AsyncOpenAI | None = None,
        instruction_role: str | None = None,
        base_url: str | None = None,
        env_file_path: str | None = None,
        env_file_encoding: str | None = None,
    ) -> None: ...
```

#### Options TypedDict

```python
class OpenAIChatOptions(ChatOptions[TResponseModel], Generic[TResponseModel], total=False):
    logit_bias: dict[str | int, float]
    logprobs: bool
    top_logprobs: int
    prediction: Prediction
```

#### Settings Class

Location: [python/packages/core/agent_framework/openai/_shared.py](python/packages/core/agent_framework/openai/_shared.py#L73-L117)

```python
class OpenAISettings(AFBaseSettings):
    env_prefix: ClassVar[str] = "OPENAI_"

    api_key: SecretStr | Callable[[], str | Awaitable[str]] | None = None
    base_url: str | None = None
    org_id: str | None = None
    chat_model_id: str | None = None
    responses_model_id: str | None = None
```

---

### 5.2 Azure OpenAI Chat Client

#### Location

[python/packages/core/agent_framework/azure/_chat_client.py](python/packages/core/agent_framework/azure/_chat_client.py)

#### Class Declaration

```python
@use_function_invocation
@use_instrumentation
@use_chat_middleware
class AzureOpenAIChatClient(
    AzureOpenAIConfigMixin, OpenAIBaseChatClient[TAzureOpenAIChatOptions], Generic[TAzureOpenAIChatOptions]
):
    """Azure OpenAI Chat completion class."""

    def __init__(
        self,
        *,
        api_key: str | None = None,
        deployment_name: str | None = None,
        endpoint: str | None = None,
        base_url: str | None = None,
        api_version: str | None = None,
        ad_token: str | None = None,
        ad_token_provider: AsyncAzureADTokenProvider | None = None,
        token_endpoint: str | None = None,
        credential: TokenCredential | None = None,
        default_headers: Mapping[str, str] | None = None,
        async_client: AsyncAzureOpenAI | None = None,
        env_file_path: str | None = None,
        env_file_encoding: str | None = None,
        instruction_role: str | None = None,
        **kwargs: Any,
    ) -> None: ...
```

#### Options TypedDict

```python
class AzureOpenAIChatOptions(OpenAIChatOptions[TResponseModel], Generic[TResponseModel], total=False):
    data_sources: list[dict[str, Any]]  # Azure "On Your Data"
    user_security_context: AzureUserSecurityContext
    n: int
```

---

### 5.3 Anthropic Chat Client

#### Location

[python/packages/anthropic/agent_framework_anthropic/_chat_client.py](python/packages/anthropic/agent_framework_anthropic/_chat_client.py)

#### Class Declaration

```python
@use_function_invocation
@use_instrumentation
@use_chat_middleware
class AnthropicClient(BaseChatClient[TAnthropicOptions], Generic[TAnthropicOptions]):
    """Anthropic Chat client."""

    OTEL_PROVIDER_NAME: ClassVar[str] = "anthropic"

    def __init__(
        self,
        *,
        api_key: str | None = None,
        model_id: str | None = None,
        anthropic_client: AsyncAnthropic | None = None,
        additional_beta_flags: list[str] | None = None,
        env_file_path: str | None = None,
        env_file_encoding: str | None = None,
        **kwargs: Any,
    ) -> None: ...
```

#### Options TypedDict

```python
class AnthropicChatOptions(ChatOptions[TResponseModel], Generic[TResponseModel], total=False):
    top_k: int
    service_tier: Literal["auto", "standard_only"]
    thinking: ThinkingConfig  # Extended thinking for Claude
    container: dict[str, Any]  # Container configuration for skills
    additional_beta_flags: list[str]

    # Unsupported options (set to None)
    logit_bias: None
    seed: None
    frequency_penalty: None
    presence_penalty: None
    store: None
    conversation_id: None
```

#### Settings Class

```python
class AnthropicSettings(AFBaseSettings):
    env_prefix: ClassVar[str] = "ANTHROPIC_"

    api_key: SecretStr | None = None
    chat_model_id: str | None = None
```

---

### 5.4 Bedrock Chat Client

#### Location

[python/packages/bedrock/agent_framework_bedrock/_chat_client.py](python/packages/bedrock/agent_framework_bedrock/_chat_client.py)

#### Class Declaration

```python
@use_function_invocation
@use_instrumentation
@use_chat_middleware
class BedrockChatClient(BaseChatClient[TBedrockChatOptions], Generic[TBedrockChatOptions]):
    """Async chat client for Amazon Bedrock's Converse API."""

    OTEL_PROVIDER_NAME: ClassVar[str] = "aws.bedrock"

    def __init__(
        self,
        *,
        region: str | None = None,
        model_id: str | None = None,
        access_key: str | None = None,
        secret_key: str | None = None,
        session_token: str | None = None,
        client: BaseClient | None = None,
        boto3_session: Boto3Session | None = None,
        env_file_path: str | None = None,
        env_file_encoding: str | None = None,
        **kwargs: Any,
    ) -> None: ...
```

#### Options TypedDict

```python
class BedrockChatOptions(ChatOptions[TResponseModel], Generic[TResponseModel], total=False):
    guardrailConfig: BedrockGuardrailConfig
    performanceConfig: dict[str, Any]
    requestMetadata: dict[str, str]
    promptVariables: dict[str, dict[str, str]]

    # Unsupported options
    seed: None
    frequency_penalty: None
    presence_penalty: None
    allow_multiple_tool_calls: None
    response_format: None
    user: None
    store: None
    logit_bias: None
```

#### Settings Class

```python
class BedrockSettings(AFBaseSettings):
    env_prefix: ClassVar[str] = "BEDROCK_"

    region: str = DEFAULT_REGION  # "us-east-1"
    chat_model_id: str | None = None
    access_key: SecretStr | None = None
    secret_key: SecretStr | None = None
    session_token: SecretStr | None = None
```

---

### 5.5 Ollama Chat Client

#### Location

[python/packages/ollama/agent_framework_ollama/_chat_client.py](python/packages/ollama/agent_framework_ollama/_chat_client.py)

#### Class Declaration

```python
@use_function_invocation
@use_instrumentation
@use_chat_middleware
class OllamaChatClient(BaseChatClient[TOllamaChatOptions], Generic[TOllamaChatOptions]):
    """Ollama Chat completion class."""

    OTEL_PROVIDER_NAME: ClassVar[str] = "ollama"

    def __init__(
        self,
        *,
        host: str | None = None,
        client: AsyncClient | None = None,
        model_id: str | None = None,
        env_file_path: str | None = None,
        env_file_encoding: str | None = None,
        **kwargs: Any,
    ) -> None: ...
```

#### Options TypedDict

```python
class OllamaChatOptions(ChatOptions[TResponseModel], Generic[TResponseModel], total=False):
    # Ollama model-specific options
    num_predict: int
    top_k: int
    min_p: float
    typical_p: float
    repeat_penalty: float
    repeat_last_n: int
    penalize_newline: bool
    num_ctx: int
    num_batch: int
    keep_alive: str | int
    think: bool  # For thinking models

    # Unsupported options
    tool_choice: None
    allow_multiple_tool_calls: None
    user: None
    store: None
    logit_bias: None
    metadata: None
```

#### Settings Class

```python
class OllamaSettings(AFBaseSettings):
    env_prefix: ClassVar[str] = "OLLAMA_"

    host: str | None = None
    model_id: str | None = None
```

---

### 5.6 Azure AI Agent Client

#### Location

[python/packages/azure-ai/agent_framework_azure_ai/_chat_client.py](python/packages/azure-ai/agent_framework_azure_ai/_chat_client.py)

#### Class Declaration

```python
@use_function_invocation
@use_instrumentation
@use_chat_middleware
class AzureAIAgentClient(BaseChatClient[TAzureAIAgentOptions], Generic[TAzureAIAgentOptions]):
    """Azure AI Agent Chat client."""

    OTEL_PROVIDER_NAME: ClassVar[str] = "azure.ai"

    def __init__(
        self,
        *,
        agents_client: AgentsClient | None = None,
        agent_id: str | None = None,
        agent_name: str | None = None,
        agent_description: str | None = None,
        thread_id: str | None = None,
        project_endpoint: str | None = None,
        model_deployment_name: str | None = None,
        credential: AsyncTokenCredential | None = None,
        should_cleanup_agent: bool = True,
        env_file_path: str | None = None,
        env_file_encoding: str | None = None,
        **kwargs: Any,
    ) -> None: ...
```

---

## 6. Tool System

### 6.1 ToolProtocol

#### Location

[python/packages/core/agent_framework/_tools.py](python/packages/core/agent_framework/_tools.py#L157-L180)

```python
@runtime_checkable
class ToolProtocol(Protocol):
    """Represents a generic tool."""

    name: str
    description: str
    additional_properties: dict[str, Any] | None

    def __str__(self) -> str: ...
```

### 6.2 FunctionTool Class

#### Location

[python/packages/core/agent_framework/_tools.py](python/packages/core/agent_framework/_tools.py#L556-L800)

```python
class FunctionTool(BaseTool, Generic[ArgsT, ReturnT]):
    """A tool that wraps a Python function to make it callable by AI models."""

    def __init__(
        self,
        *,
        name: str,
        description: str = "",
        approval_mode: Literal["always_require", "never_require"] | None = None,
        max_invocations: int | None = None,
        max_invocation_exceptions: int | None = None,
        additional_properties: dict[str, Any] | None = None,
        func: Callable[..., Awaitable[ReturnT] | ReturnT] | None = None,
        input_model: type[ArgsT] | Mapping[str, Any] | None = None,
        **kwargs: Any,
    ) -> None: ...

    async def invoke(
        self,
        *,
        arguments: ArgsT | None = None,
        **kwargs: Any,
    ) -> ReturnT: ...

    def parameters(self) -> dict[str, Any]: ...

    def to_json_schema_spec(self) -> dict[str, Any]: ...
```

### 6.3 Tool Decorator

#### Location

[python/packages/core/agent_framework/_tools.py](python/packages/core/agent_framework/_tools.py#L1218-L1320)

```python
def tool(
    func: Callable[..., ReturnT | Awaitable[ReturnT]] | None = None,
    *,
    name: str | None = None,
    description: str | None = None,
    approval_mode: Literal["always_require", "never_require"] | None = None,
    max_invocations: int | None = None,
    max_invocation_exceptions: int | None = None,
    additional_properties: dict[str, Any] | None = None,
) -> FunctionTool[Any, ReturnT] | Callable[...]: ...
```

#### Usage Example

```python
from agent_framework import tool
from typing import Annotated

@tool(approval_mode="never_require")
def get_weather(
    location: Annotated[str, "The city name"],
    unit: Annotated[str, "Temperature unit"] = "celsius",
) -> str:
    """Get the weather for a location."""
    return f"Weather in {location}: 22°{unit[0].upper()}"
```

### 6.4 Hosted Tool Types

| Class | Location | Purpose |
|-------|----------|---------|
| `HostedWebSearchTool` | `_tools.py:289` | Web search capability |
| `HostedCodeInterpreterTool` | `_tools.py:239` | Code execution |
| `HostedFileSearchTool` | `_tools.py:453` | File/vector search |
| `HostedMCPTool` | `_tools.py:380` | MCP server integration |
| `HostedImageGenerationTool` | `_tools.py:343` | Image generation |

### 6.5 Function Invocation Configuration

#### Location

[python/packages/core/agent_framework/_tools.py](python/packages/core/agent_framework/_tools.py#L1340-L1430)

```python
class FunctionInvocationConfiguration(SerializationMixin):
    """Configuration for function invocation in chat clients."""

    def __init__(
        self,
        enabled: bool = True,
        max_iterations: int = DEFAULT_MAX_ITERATIONS,  # 40
        max_consecutive_errors_per_request: int = DEFAULT_MAX_CONSECUTIVE_ERRORS_PER_REQUEST,  # 3
        terminate_on_unknown_calls: bool = False,
        additional_tools: Sequence[ToolProtocol] | None = None,
        include_detailed_errors: bool = False,
    ) -> None: ...
```

---

## 7. Streaming Implementation Patterns

### 7.1 Async Generator Pattern

All providers implement `_inner_get_streaming_response()` as an async generator:

```python
@override
async def _inner_get_streaming_response(
    self,
    *,
    messages: MutableSequence[ChatMessage],
    options: dict[str, Any],
    **kwargs: Any,
) -> AsyncIterable[ChatResponseUpdate]:
    # Prepare options
    options_dict = self._prepare_options(messages, options)

    # Execute streaming request
    async for chunk in await self.client.chat(..., stream=True):
        yield self._parse_streaming_response(chunk)
```

### 7.2 OpenAI Streaming Pattern

Location: [python/packages/core/agent_framework/openai/_chat_client.py](python/packages/core/agent_framework/openai/_chat_client.py#L151-L176)

```python
async for chunk in await client.chat.completions.create(stream=True, **options_dict):
    if len(chunk.choices) == 0 and chunk.usage is None:
        continue
    yield self._parse_response_update_from_openai(chunk)
```

### 7.3 Anthropic Streaming Pattern

Location: [python/packages/anthropic/agent_framework_anthropic/_chat_client.py](python/packages/anthropic/agent_framework_anthropic/_chat_client.py#L354-L360)

```python
async for chunk in await self.anthropic_client.beta.messages.create(**run_options, stream=True):
    parsed_chunk = self._process_stream_event(chunk)
    if parsed_chunk:
        yield parsed_chunk
```

### 7.4 Ollama Streaming Pattern

Location: [python/packages/ollama/agent_framework_ollama/_chat_client.py](python/packages/ollama/agent_framework_ollama/_chat_client.py#L345-L365)

```python
response_object: AsyncIterable[OllamaChatResponse] = await self.client.chat(
    stream=True,
    **options_dict,
    **kwargs,
)
async for part in response_object:
    yield self._parse_streaming_response_from_ollama(part)
```

---

## 8. Observability / OpenTelemetry

### 8.1 Location

[python/packages/core/agent_framework/observability.py](python/packages/core/agent_framework/observability.py)

### 8.2 Key Functions

| Function | Purpose |
|----------|---------|
| `enable_instrumentation()` | Enable tracing globally |
| `configure_otel_providers()` | Configure exporters and providers |
| `use_instrumentation` | Class decorator for chat clients |
| `use_agent_instrumentation` | Class decorator for agents |
| `get_tracer()` | Get OpenTelemetry tracer |
| `get_meter()` | Get OpenTelemetry meter |

### 8.3 use_instrumentation Decorator

Location: [python/packages/core/agent_framework/observability.py](python/packages/core/agent_framework/observability.py#L1232-L1290)

```python
def use_instrumentation(
    chat_client: type[TChatClient],
) -> type[TChatClient]:
    """Class decorator that enables OpenTelemetry observability for a chat client."""

    provider_name = str(getattr(chat_client, "OTEL_PROVIDER_NAME", "unknown"))

    chat_client.get_response = _trace_get_response(
        chat_client.get_response, provider_name=provider_name
    )
    chat_client.get_streaming_response = _trace_get_streaming_response(
        chat_client.get_streaming_response, provider_name=provider_name
    )

    setattr(chat_client, OPEN_TELEMETRY_CHAT_CLIENT_MARKER, True)
    return chat_client
```

### 8.4 OtelAttr Enum

Location: [python/packages/core/agent_framework/observability.py](python/packages/core/agent_framework/observability.py#L115-L198)

Key attributes:

```python
class OtelAttr(str, Enum):
    OPERATION = "gen_ai.operation.name"
    PROVIDER_NAME = "gen_ai.provider.name"
    INPUT_TOKENS = "gen_ai.usage.input_tokens"
    OUTPUT_TOKENS = "gen_ai.usage.output_tokens"
    TOOL_NAME = "gen_ai.tool.name"
    TOOL_ARGUMENTS = "gen_ai.tool.call.arguments"
    TOOL_RESULT = "gen_ai.tool.call.result"
    AGENT_ID = "gen_ai.agent.id"
    AGENT_NAME = "gen_ai.agent.name"
    CONVERSATION_ID = "gen_ai.conversation.id"
    # ...
```

### 8.5 ObservabilitySettings

Location: [python/packages/core/agent_framework/observability.py](python/packages/core/agent_framework/observability.py#L550-L620)

```python
class ObservabilitySettings(AFBaseSettings):
    env_prefix: ClassVar[str] = ""

    enable_instrumentation: bool = False
    enable_sensitive_data: bool = False
    enable_console_exporters: bool = False
    vs_code_extension_port: int | None = None
```

---

## 9. Middleware System

### 9.1 Location

[python/packages/core/agent_framework/_middleware.py](python/packages/core/agent_framework/_middleware.py)

### 9.2 Decorator Stacking Pattern

All chat clients use three decorators:

```python
@use_function_invocation    # Enables automatic function calling
@use_instrumentation        # Enables OpenTelemetry tracing
@use_chat_middleware        # Enables middleware pipeline
class SomeChatClient(BaseChatClient[TOptions], Generic[TOptions]):
    ...
```

### 9.3 Context Classes

| Context Class | Purpose |
|---------------|---------|
| `AgentRunContext` | Agent middleware invocations |
| `FunctionInvocationContext` | Function middleware invocations |
| `ChatContext` | Chat middleware invocations |

---

## 10. Configuration Patterns

### 10.1 AFBaseSettings Pattern

All settings classes inherit from `AFBaseSettings` (Pydantic BaseSettings):

```python
class ProviderSettings(AFBaseSettings):
    env_prefix: ClassVar[str] = "PROVIDER_"  # e.g., "OPENAI_", "ANTHROPIC_"

    api_key: SecretStr | None = None
    chat_model_id: str | None = None
    # ... other settings
```

### 10.2 Environment Variable Naming

| Provider | Prefix | Example Variables |
|----------|--------|-------------------|
| OpenAI | `OPENAI_` | `OPENAI_API_KEY`, `OPENAI_CHAT_MODEL_ID` |
| Azure OpenAI | `AZURE_OPENAI_` | `AZURE_OPENAI_ENDPOINT`, `AZURE_OPENAI_API_KEY` |
| Anthropic | `ANTHROPIC_` | `ANTHROPIC_API_KEY`, `ANTHROPIC_CHAT_MODEL_ID` |
| Bedrock | `BEDROCK_` | `BEDROCK_REGION`, `BEDROCK_CHAT_MODEL_ID` |
| Ollama | `OLLAMA_` | `OLLAMA_HOST`, `OLLAMA_MODEL_ID` |
| Azure AI | `AZURE_AI_` | `AZURE_AI_PROJECT_ENDPOINT`, `AZURE_AI_MODEL_DEPLOYMENT_NAME` |

### 10.3 Options Translation Pattern

Each provider maps common options to provider-specific API parameters:

```python
# OpenAI
OPTION_TRANSLATIONS: dict[str, str] = {
    "model_id": "model",
    "allow_multiple_tool_calls": "parallel_tool_calls",
    "max_tokens": "max_completion_tokens",
}

# Anthropic
OPTION_TRANSLATIONS: dict[str, str] = {
    "model_id": "model",
    "stop": "stop_sequences",
    "instructions": "system",
}

# Bedrock
BEDROCK_OPTION_TRANSLATIONS: dict[str, str] = {
    "model_id": "modelId",
    "max_tokens": "maxTokens",
    "top_p": "topP",
    "stop": "stopSequences",
}
```

---

## 11. Error Handling Patterns

### 11.1 Exception Types

Location: [python/packages/core/agent_framework/exceptions.py](python/packages/core/agent_framework/exceptions.py)

| Exception | Purpose |
|-----------|---------|
| `ServiceInitializationError` | Client initialization failures |
| `ServiceResponseException` | API response errors |
| `ServiceInvalidRequestError` | Invalid request parameters |
| `ToolException` | Tool execution failures |
| `ChatClientInitializationError` | Chat client setup errors |
| `ContentError` | Content parsing/validation errors |

### 11.2 Error Wrapping Pattern

```python
try:
    response = await client.chat.completions.create(**options)
except BadRequestError as ex:
    if ex.code == "content_filter":
        raise OpenAIContentFilterException(...) from ex
    raise ServiceResponseException(...) from ex
except Exception as ex:
    raise ServiceResponseException(...) from ex
```

---

## 12. Key Content Types

### Location

[python/packages/core/agent_framework/_types.py](python/packages/core/agent_framework/_types.py#L430-L700)

### Content Class Factory Methods

| Method | ContentType | Purpose |
|--------|-------------|---------|
| `Content.from_text()` | `"text"` | Text content |
| `Content.from_text_reasoning()` | `"text_reasoning"` | Reasoning/thinking |
| `Content.from_data()` | `"data"` | Binary data (base64) |
| `Content.from_uri()` | `"uri"` | URI reference |
| `Content.from_function_call()` | `"function_call"` | Tool invocation |
| `Content.from_function_result()` | `"function_result"` | Tool result |
| `Content.from_error()` | `"error"` | Error content |
| `Content.from_usage()` | `"usage"` | Token usage |
| `Content.from_hosted_file()` | `"hosted_file"` | Hosted file reference |
| `Content.from_code_interpreter_tool_call()` | `"code_interpreter_tool_call"` | Code execution |
| `Content.from_mcp_server_tool_call()` | `"mcp_server_tool_call"` | MCP tool call |

---

## 13. Summary: Provider Implementation Checklist

When implementing a new provider, include:

1. **Settings Class** - Inherit from `AFBaseSettings`, set `env_prefix`
2. **Options TypedDict** - Extend `ChatOptions`, mark unsupported as `None`
3. **Chat Client Class** - Inherit from `BaseChatClient[TOptions]`
4. **Decorators** - Apply `@use_function_invocation`, `@use_instrumentation`, `@use_chat_middleware`
5. **OTEL_PROVIDER_NAME** - Set class variable for telemetry
6. **_inner_get_response()** - Implement non-streaming
7. **_inner_get_streaming_response()** - Implement async generator
8. **_prepare_options()** - Translate options to provider API format
9. **_prepare_messages_for_provider()** - Convert ChatMessage to provider format
10. **_prepare_tools_for_provider()** - Convert tools to provider format
11. **_parse_response()** - Convert provider response to ChatResponse
12. **Role/FinishReason Maps** - Map provider values to framework enums
