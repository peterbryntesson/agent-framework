# Python Declarative Agents Implementation Analysis

**Date**: 2026-02-04  
**Package**: `agent-framework-declarative`  
**Version**: `1.0.0b260128`

---

## 1. Package Structure

### Directory Layout

```
python/packages/declarative/
├── agent_framework_declarative/
│   ├── __init__.py                    # Public API exports
│   ├── _models.py                     # Schema/model definitions (1109 lines)
│   ├── _loader.py                     # AgentFactory implementation (850 lines)
│   └── _workflows/                    # Workflow support
│       ├── __init__.py
│       ├── _factory.py                # WorkflowFactory
│       ├── _declarative_builder.py    # Graph builder (974 lines)
│       ├── _executors_*.py            # Action executors
│       └── ...
├── tests/
├── pyproject.toml
└── README.md
```

### Dependencies

- `agent-framework-core`
- `powerfx>=0.0.31` (for PowerFx expression evaluation)
- `pyyaml>=6.0,<7.0`

---

## 2. Model Definitions (Schema)

**File**: [_models.py](python/packages/declarative/agent_framework_declarative/_models.py)

### 2.1 Core Agent Definition Classes

#### `PromptAgent` (Lines 810-860)

The primary agent definition class for YAML-based agents.

```python
class PromptAgent(AgentDefinition):
    """Object representing a prompt agent specification."""

    def __init__(
        self,
        kind: str = "Prompt",
        name: str | None = None,
        displayName: str | None = None,
        description: str | None = None,
        metadata: dict[str, Any] | None = None,
        inputSchema: PropertySchema | None = None,
        outputSchema: PropertySchema | None = None,
        model: Model | dict[str, Any] | None = None,
        tools: list[Tool] | None = None,
        template: Template | dict[str, Any] | None = None,
        instructions: str | None = None,
        additionalInstructions: str | None = None,
    ) -> None:
```

**Required Fields:**
- `kind`: Must be `"Prompt"` or `"Agent"`
- `name`: Agent identifier

**Optional Fields:**
- `displayName`: Human-readable name
- `description`: Agent description
- `instructions`: System prompt/instructions
- `model`: Model configuration
- `tools`: List of tool definitions
- `outputSchema`: Structured output schema
- `inputSchema`: Input validation schema
- `template`: Template configuration
- `metadata`: Custom metadata dictionary

#### `AgentDefinition` Base (Lines 467-503)

```python
class AgentDefinition(SerializationMixin):
    def __init__(
        self,
        kind: str | None = None,
        name: str | None = None,
        displayName: str | None = None,
        description: str | None = None,
        metadata: dict[str, Any] | None = None,
        inputSchema: PropertySchema | None = None,
        outputSchema: PropertySchema | None = None,
    ) -> None:
```

### 2.2 Model Configuration

#### `Model` Class (Lines 394-411)

```python
class Model(SerializationMixin):
    def __init__(
        self,
        id: str | None = None,             # Model/deployment ID
        provider: str | None = None,        # Provider name (e.g., "AzureOpenAI")
        apiType: str | None = None,         # API type (e.g., "Chat", "Assistants")
        connection: Connections | None = None,  # Connection settings
        options: ModelOptions | None = None,    # Model options
    ) -> None:
```

#### `ModelOptions` Class (Lines 358-380)

```python
class ModelOptions(SerializationMixin):
    def __init__(
        self,
        frequencyPenalty: float | None = None,
        maxOutputTokens: int | None = None,
        presencePenalty: float | None = None,
        seed: int | None = None,
        temperature: float | None = None,
        topK: int | None = None,
        topP: float | None = None,
        stopSequences: list[str] | None = None,
        allowMultipleToolCalls: bool | None = None,
        additionalProperties: dict[str, Any] | None = None,
    ) -> None:
```

### 2.3 Connection Types

**File**: [_models.py](python/packages/declarative/agent_framework_declarative/_models.py#L237-L350)

Four connection types are supported:

| Kind | Class | Purpose |
|------|-------|---------|
| `reference` | `ReferenceConnection` | Reference to named connection |
| `remote` | `RemoteConnection` | Remote endpoint with name |
| `key` / `apiKey` | `ApiKeyConnection` | API key authentication |
| `anonymous` | `AnonymousConnection` | No authentication |

#### `Connection` Base (Lines 237-282)

```python
class Connection(SerializationMixin):
    def __init__(
        self,
        kind: Literal["reference", "remote", "key", "anonymous"],
        authenticationMode: str | None = None,
        usageDescription: str | None = None,
    ) -> None:
```

#### `ApiKeyConnection` (Lines 315-333)

```python
class ApiKeyConnection(Connection):
    def __init__(
        self,
        kind: Literal["key"] = "key",
        authenticationMode: str | None = None,
        usageDescription: str | None = None,
        endpoint: str | None = None,
        apiKey: str | None = None,
        key: str | None = None,  # Alias for apiKey
    ) -> None:
```

### 2.4 Tool Definitions

**File**: [_models.py](python/packages/declarative/agent_framework_declarative/_models.py#L509-L695)

#### Tool Types (dispatched by `kind` field)

| Kind | Class | Description |
|------|-------|-------------|
| `function` | `FunctionTool` | Custom function tool |
| `custom` | `CustomTool` | Custom tool with connection |
| `web_search` | `WebSearchTool` | Web search capability |
| `file_search` | `FileSearchTool` | Vector store file search |
| `mcp` | `McpTool` | MCP server tool |
| `openapi` | `OpenApiTool` | OpenAPI specification tool |
| `code_interpreter` | `CodeInterpreterTool` | Code interpreter tool |

#### `FunctionTool` (Lines 582-610)

```python
class FunctionTool(Tool):
    def __init__(
        self,
        name: str | None = None,
        kind: str = "function",
        description: str | None = None,
        bindings: list[Binding] | None = None,
        parameters: PropertySchema | list[Property] | dict[str, Any] | None = None,
        strict: bool = False,
    ) -> None:
```

#### `McpTool` (Lines 732-765)

```python
class McpTool(Tool):
    def __init__(
        self,
        name: str | None = None,
        kind: str = "mcp",
        description: str | None = None,
        bindings: list[Binding] | None = None,
        connection: Connection | None = None,
        serverName: str | None = None,
        serverDescription: str | None = None,
        approvalMode: McpServerApprovalMode | None = None,
        allowedTools: list[str] | None = None,
        url: str | None = None,
    ) -> None:
```

### 2.5 Property Schema

**File**: [_models.py](python/packages/declarative/agent_framework_declarative/_models.py#L69-L230)

#### `Property` Class (Lines 81-108)

```python
class Property(SerializationMixin):
    def __init__(
        self,
        name: str | None = None,
        kind: str | None = None,  # "string", "number", "boolean", "array", "object"
        description: str | None = None,
        required: bool | None = None,
        default: Any | None = None,
        example: Any | None = None,
        enum: list[Any] | None = None,
    ) -> None:
```

#### `PropertySchema` Class (Lines 181-230)

```python
class PropertySchema(SerializationMixin):
    def __init__(
        self,
        examples: list[dict[str, Any]] | None = None,
        strict: bool = False,
        properties: list[Property] | dict[str, dict[str, Any]] | None = None,
    ) -> None:
```

### 2.6 PowerFx Expression Support

**File**: [_models.py](python/packages/declarative/agent_framework_declarative/_models.py#L37-L66)

The `_try_powerfx_eval` function evaluates PowerFx expressions in YAML values:

```python
def _try_powerfx_eval(value: str | None, log_value: bool = True) -> str | None:
    """Check if a value refers to a environment variable and parse it if so."""
    if value is None:
        return value
    if not value.startswith("="):  # PowerFx expressions start with "="
        return value
    # ... evaluates using PowerFx engine with Env symbol
```

**Usage in YAML:**
```yaml
model:
  id: =Env.AZURE_OPENAI_DEPLOYMENT_NAME
  connection:
    endpoint: =Env.AZURE_AI_PROJECT_ENDPOINT
```

---

## 3. AgentFactory Implementation

**File**: [_loader.py](python/packages/declarative/agent_framework_declarative/_loader.py)

### 3.1 Factory Class

```python
class AgentFactory:
    """Factory for creating ChatAgent instances from declarative YAML definitions."""

    def __init__(
        self,
        *,
        chat_client: ChatClientProtocol | None = None,
        bindings: Mapping[str, Any] | None = None,
        connections: Mapping[str, Any] | None = None,
        client_kwargs: Mapping[str, Any] | None = None,
        additional_mappings: Mapping[str, ProviderTypeMapping] | None = None,
        default_provider: str = "AzureAIClient",
        safe_mode: bool = True,
        env_file_path: str | None = None,
        env_file_encoding: str | None = None,
    ) -> None:
```

**Parameters:**
- `chat_client`: Pre-configured chat client (optional)
- `bindings`: Function bindings for tools (name → callable)
- `connections`: Connection objects for reference resolution
- `client_kwargs`: Arguments passed to chat client constructor
- `additional_mappings`: Custom provider type mappings
- `default_provider`: Default provider when not specified
- `safe_mode`: Controls PowerFx environment variable access
- `env_file_path`: Path to .env file

### 3.2 Creation Methods

**File**: [_loader.py](python/packages/declarative/agent_framework_declarative/_loader.py#L270-L430)

| Method | Input | Description |
|--------|-------|-------------|
| `create_agent_from_yaml_path()` | File path | Load YAML from file |
| `create_agent_from_yaml()` | YAML string | Parse YAML string |
| `create_agent_from_dict()` | Dictionary | Direct dict input |
| `create_agent_from_yaml_path_async()` | File path | Async version for providers |
| `create_agent_from_yaml_async()` | YAML string | Async version |
| `create_agent_from_dict_async()` | Dictionary | Async version |

### 3.3 Provider Type Mapping

**File**: [_loader.py](python/packages/declarative/agent_framework_declarative/_loader.py#L53-L100)

```python
PROVIDER_TYPE_OBJECT_MAPPING: dict[str, ProviderTypeMapping] = {
    "AzureOpenAI.Chat": {
        "package": "agent_framework.azure",
        "name": "AzureOpenAIChatClient",
        "model_id_field": "deployment_name",
    },
    "AzureOpenAI.Assistants": {...},
    "AzureOpenAI.Responses": {...},
    "OpenAI.Chat": {...},
    "OpenAI.Assistants": {...},
    "OpenAI.Responses": {...},
    "AzureAIAgentClient": {...},
    "AzureAIClient": {...},
    "AzureAI.ProjectProvider": {...},
    "Anthropic.Chat": {...},
}
```

**Lookup Logic** ([_loader.py#L825-L850](python/packages/declarative/agent_framework_declarative/_loader.py#L825-L850)):
```python
def _retrieve_provider_configuration(self, model: Model) -> ProviderTypeMapping:
    class_lookup = (
        f"{model.provider}.{model.apiType}"
        if model.apiType
        else f"{model.provider}"
        if model.provider
        else self.default_provider
    )
    # Check additional_mappings first, then PROVIDER_TYPE_OBJECT_MAPPING
```

### 3.4 Tool Parsing

**File**: [_loader.py](python/packages/declarative/agent_framework_declarative/_loader.py#L720-L820)

```python
def _parse_tool(self, tool_resource: Tool) -> ToolProtocol:
    match tool_resource:
        case FunctionTool():
            # Look up function from bindings
            func: Callable[..., Any] | None = None
            if self.bindings and tool_resource.bindings:
                for binding in tool_resource.bindings:
                    if binding.name and (func := self.bindings.get(binding.name)):
                        break
            return AFFunctionTool(
                name=tool_resource.name,
                description=tool_resource.description,
                input_model=tool_resource.parameters.to_json_schema() if tool_resource.parameters else None,
                func=func,
            )
        case WebSearchTool():
            return HostedWebSearchTool(...)
        case FileSearchTool():
            return HostedFileSearchTool(...)
        case CodeInterpreterTool():
            return HostedCodeInterpreterTool(...)
        case McpTool():
            return HostedMCPTool(...)
```

---

## 4. WorkflowFactory Implementation

**File**: [_workflows/_factory.py](python/packages/declarative/agent_framework_declarative/_workflows/_factory.py)

### 4.1 Factory Class

```python
class WorkflowFactory:
    """Factory for creating executable Workflow objects from YAML definitions."""

    def __init__(
        self,
        *,
        agent_factory: AgentFactory | None = None,
        agents: Mapping[str, AgentProtocol | AgentExecutor] | None = None,
        bindings: Mapping[str, Any] | None = None,
        env_file: str | None = None,
        checkpoint_storage: CheckpointStorage | None = None,
    ) -> None:
```

### 4.2 Workflow Creation

Methods mirror `AgentFactory`:
- `create_workflow_from_yaml_path()`
- `create_workflow_from_yaml()`
- `create_workflow_from_definition()`

### 4.3 Declarative Workflow Builder

**File**: [_workflows/_declarative_builder.py](python/packages/declarative/agent_framework_declarative/_workflows/_declarative_builder.py)

Transforms YAML actions into a workflow graph with:
- Executor nodes for each action
- Sequential flow edges
- Conditional branching (If/Switch)
- Loop edges (Foreach)
- Goto handling

---

## 5. YAML Schema Documentation

### 5.1 Minimal Agent Definition

```yaml
kind: Prompt
name: MyAgent
instructions: You are a helpful assistant.
```

### 5.2 Full Agent Definition

```yaml
kind: Prompt
name: Assistant
displayName: Helpful Assistant
description: A helpful assistant agent

instructions: |
  You are a helpful assistant.
  Answer questions clearly and concisely.

model:
  id: gpt-4o                     # Model/deployment ID
  provider: AzureOpenAI          # Provider name
  apiType: Chat                  # API type (Chat, Assistants, Responses)
  connection:
    kind: key                    # Connection type
    endpoint: https://...
    apiKey: =Env.AZURE_OPENAI_KEY
  options:
    temperature: 0.7
    topP: 0.95
    maxOutputTokens: 4096
    frequencyPenalty: 0.0
    presencePenalty: 0.0
    allowMultipleToolCalls: true

tools:
  - kind: function
    name: GetWeather
    description: Get weather for a location
    bindings:
      get_weather: get_weather   # Maps to Python function
    parameters:
      properties:
        location:
          kind: string
          description: City and state
          required: true
        unit:
          kind: string
          enum: [celsius, fahrenheit]
          required: false

  - kind: web_search
    description: Search the web

  - kind: file_search
    vectorStoreIds:
      - vs_abc123
    maximumResultCount: 10

  - kind: mcp
    name: MyMCPServer
    url: https://mcp.example.com
    allowedTools: [tool1, tool2]
    approvalMode: always  # or "never" or specify object

outputSchema:
  properties:
    answer:
      kind: string
      required: true
      description: The response text
    confidence:
      kind: number
      required: false
```

### 5.3 Provider-Specific Examples

**Azure OpenAI Chat:**
```yaml
model:
  id: =Env.AZURE_OPENAI_DEPLOYMENT_NAME
  provider: AzureOpenAI
  apiType: Chat
```

**Azure AI Foundry (Remote):**
```yaml
model:
  id: gpt-4.1-mini
  connection:
    kind: remote
    endpoint: =Env.AZURE_FOUNDRY_PROJECT_ENDPOINT
```

**OpenAI Responses:**
```yaml
model:
  id: gpt-4o
  provider: OpenAI
  apiType: Responses
```

### 5.4 Workflow YAML Schema

```yaml
kind: Workflow
name: my_workflow

agents:
  MyAgent:
    file: ./agents/my_agent.yaml   # Or inline definition

trigger:
  kind: OnConversationStart
  id: main_trigger
  actions:
    - kind: SetVariable
      id: init_var
      variable: Local.Counter
      value: 0

    - kind: InvokeAzureAgent
      id: invoke_agent
      agent:
        name: MyAgent
      input:
        arguments:
          query: =Local.UserInput

    - kind: SendActivity
      id: send_response
      activity: =Local.Response
```

---

## 6. Error Handling Patterns

**File**: [_loader.py](python/packages/declarative/agent_framework_declarative/_loader.py#L104-L115)

```python
class DeclarativeLoaderError(AgentFrameworkException):
    """Exception raised for errors in the declarative loader."""
    pass

class ProviderLookupError(DeclarativeLoaderError):
    """Exception raised for errors in provider type lookup."""
    pass
```

**Validation Points:**
1. YAML file existence check
2. PromptAgent kind validation
3. Provider type resolution
4. Connection reference resolution
5. Tool binding resolution

---

## 7. Usage Examples

### 7.1 Basic Agent from YAML File

```python
from agent_framework.declarative import AgentFactory

factory = AgentFactory()
agent = factory.create_agent_from_yaml_path("agent.yaml")

async for event in agent.run_stream("Hello!"):
    print(event)
```

### 7.2 Agent with Pre-configured Client

```python
from agent_framework.azure import AzureOpenAIResponsesClient
from agent_framework.declarative import AgentFactory

client = AzureOpenAIResponsesClient(credential=credential)
factory = AgentFactory(chat_client=client)
agent = factory.create_agent_from_yaml_path("agent.yaml")
```

### 7.3 Agent with Function Bindings

```python
def get_weather(location: str) -> str:
    return f"Weather in {location}: Sunny"

factory = AgentFactory(
    chat_client=client,
    bindings={"get_weather": get_weather},
)
agent = factory.create_agent_from_yaml_path("agent.yaml")
```

### 7.4 Inline YAML Definition

```python
yaml_content = """
kind: Prompt
name: MyAgent
instructions: You are helpful.
model:
  id: gpt-4o
  connection:
    kind: remote
    endpoint: =Env.AZURE_AI_PROJECT_ENDPOINT
"""

factory = AgentFactory(client_kwargs={"credential": credential})
agent = factory.create_agent_from_yaml(yaml_content)
```

### 7.5 Workflow Creation

```python
from agent_framework.declarative import WorkflowFactory

factory = WorkflowFactory(agents={"MyAgent": my_agent})
workflow = factory.create_workflow_from_yaml_path("workflow.yaml")

async for event in workflow.run_stream({"input": "Hello"}):
    print(event)
```

---

## 8. Key File References

| File | Lines | Description |
|------|-------|-------------|
| [_models.py](python/packages/declarative/agent_framework_declarative/_models.py) | 1-1109 | All schema/model definitions |
| [_loader.py](python/packages/declarative/agent_framework_declarative/_loader.py) | 1-850 | AgentFactory implementation |
| [_workflows/_factory.py](python/packages/declarative/agent_framework_declarative/_workflows/_factory.py) | 1-677 | WorkflowFactory |
| [_workflows/_declarative_builder.py](python/packages/declarative/agent_framework_declarative/_workflows/_declarative_builder.py) | 1-974 | Workflow graph builder |
| [__init__.py](python/packages/declarative/agent_framework_declarative/__init__.py) | 1-37 | Public API exports |

---

## 9. Summary

The Python declarative agents implementation provides:

1. **Schema-driven agent definitions** using plain Python classes with `SerializationMixin`
2. **Factory pattern** (`AgentFactory`, `WorkflowFactory`) for creating agents/workflows from YAML
3. **Provider registration** via `PROVIDER_TYPE_OBJECT_MAPPING` dictionary with dynamic import
4. **Tool factory** with match/case pattern for dispatching to specific tool types
5. **PowerFx expression support** for environment variable access in YAML values
6. **Connection abstraction** supporting API keys, remote endpoints, and references
7. **Workflow support** with graph-based execution and checkpointing

The design follows a clean separation between:
- **Models** (`_models.py`): Pure data classes for schema representation
- **Loader** (`_loader.py`): Factory logic for instantiation
- **Workflows** (`_workflows/`): Multi-agent orchestration support
