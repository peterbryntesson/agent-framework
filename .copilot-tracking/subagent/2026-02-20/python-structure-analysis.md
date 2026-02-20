# Python Package Structure Analysis

> Generated: 2026-02-20
> Purpose: Support C++ port implementation planning

## Summary Metrics

| Metric | Count |
|---|---|
| Total packages | 20 |
| Total .py files | 411 |
| Source files (excl `__init__.py`, tests) | 168 |
| Test files | 181 |
| `__init__.py` files | 54 |
| Approximate source lines of code | ~78,800 |

## Package Inventory

### All Packages by Source File Count (descending)

| # | Package Dir | PyPI Name | Source Files | Test Files | Total .py |
|---|---|---|---|---|---|
| 1 | `core` | agent-framework-core | 62 | 69 | 149 |
| 2 | `ag-ui` | agent-framework-ag-ui | 28 | 16 | 49 |
| 3 | `declarative` | agent-framework-declarative | 17 | 14 | 33 |
| 4 | `devui` | agent-framework-devui | 13 | 12 | 29 |
| 5 | `durabletask` | agent-framework-durabletask | 11 | 18 | 30 |
| 6 | `purview` | agent-framework-purview | 7 | 9 | 17 |
| 7 | `lab` | agent-framework-lab | 6 | 5 | 25 |
| 8 | `azure-ai` | agent-framework-azure-ai | 5 | 6 | 12 |
| 9 | `azurefunctions` | agent-framework-azurefunctions | 4 | 13 | 18 |
| 10 | `redis` | agent-framework-redis | 2 | 2 | 5 |
| 11 | `chatkit` | agent-framework-chatkit | 2 | 2 | 5 |
| 12 | `copilotstudio` | agent-framework-copilotstudio | 2 | 3 | 6 |
| 13 | `github_copilot` | agent-framework-github-copilot | 2 | 1 | 5 |
| 14 | `bedrock` | agent-framework-bedrock | 1 | 2 | 6 |
| 15 | `foundry_local` | agent-framework-foundry-local | 1 | 2 | 5 |
| 16 | `anthropic` | agent-framework-anthropic | 1 | 2 | 4 |
| 17 | `ollama` | agent-framework-ollama | 1 | 2 | 4 |
| 18 | `azure-ai-search` | agent-framework-azure-ai-search | 1 | 1 | 3 |
| 19 | `mem0` | agent-framework-mem0 | 1 | 1 | 3 |
| 20 | `a2a` | agent-framework-a2a | 1 | 1 | 3 |

## Inter-Package Dependency Graph

```
agent-framework (meta)
└── agent-framework-core [all] (includes all optional packages)
    ├── agent-framework-a2a → core
    ├── agent-framework-ag-ui → core
    ├── agent-framework-anthropic → core
    ├── agent-framework-azure-ai → core
    ├── agent-framework-azure-ai-search → core
    ├── agent-framework-azurefunctions → core, durabletask
    ├── agent-framework-bedrock → core
    ├── agent-framework-chatkit → core
    ├── agent-framework-copilotstudio → core
    ├── agent-framework-declarative → core
    ├── agent-framework-devui → core
    ├── agent-framework-durabletask → core
    ├── agent-framework-foundry-local → core
    ├── agent-framework-github-copilot → core
    ├── agent-framework-lab → (sub-packages: gaia, lightning, tau2)
    ├── agent-framework-mem0 → core
    ├── agent-framework-ollama → core
    ├── agent-framework-purview → core
    └── agent-framework-redis → core
```

Key observation: **azurefunctions** is the only package with a multi-package dependency (core + durabletask). All others depend solely on core.

## Key Public Types per Package

### 1. core (62 source files) — The Foundation

The core package contains the primary abstractions and built-in implementations.

#### Core Abstractions (`_agents.py`, `_clients.py`, `_tools.py`, `_threads.py`, `_memory.py`, `_middleware.py`)

- **AgentProtocol** — Protocol defining the agent interface
- **BaseAgent** — Base class for all agents
- **ChatAgent** — Standard chat-based agent
- **ChatClientProtocol** / **BaseChatClient** — Chat client abstraction
- **ToolProtocol** / **BaseTool** — Tool abstraction
- **FunctionTool** — Function-based tool implementation
- **HostedCodeInterpreterTool**, **HostedWebSearchTool**, **HostedImageGenerationTool**, **HostedMCPTool**, **HostedFileSearchTool** — Hosted tool types
- **MCPTool**, **MCPStdioTool**, **MCPStreamableHTTPTool**, **MCPWebsocketTool** — MCP integration tools
- **ChatMessageStoreProtocol**, **ChatMessageStore** — Message storage abstraction
- **AgentThread**, **AgentThreadState** — Threading/conversation state
- **Context**, **ContextProvider** — Memory/context provider
- **AgentMiddleware**, **FunctionMiddleware**, **ChatMiddleware** — Middleware pipeline types
- **AgentMiddlewarePipeline**, **FunctionMiddlewarePipeline**, **ChatMiddlewarePipeline**

#### Core Types (`_types.py`)

- **ChatMessage**, **ChatResponse**, **ChatResponseUpdate** — Message/response types
- **AgentResponse**, **AgentResponseUpdate** — Agent-level response types
- **Content**, **Role**, **FinishReason**, **UsageDetails**, **Annotation**
- **ToolMode**, **TextSpanRegion**

#### OpenAI Providers (`openai/`)

- **AssistantsClient**, **ChatClient**, **ResponsesClient** — Three OpenAI client implementations
- Azure variants: **AzureAssistantsClient**, **AzureChatClient**, **AzureResponsesClient**

#### Workflows Subsystem (`_workflows/`, 35 source files)

Major orchestration types:
- **Workflow**, **WorkflowBuilder**, **WorkflowContext**, **WorkflowRunResult**
- **WorkflowAgent**, **WorkflowExecutor**, **WorkflowViz**
- **Runner**, **RunnerContext**, **InProcRunnerContext** — Execution runtime
- **Edge**, **EdgeGroup**, **EdgeRunner** — Graph edge types
- **SequentialBuilder**, **ConcurrentBuilder**, **HandoffBuilder**, **GroupChatBuilder**, **MagenticBuilder**
- **AgentExecutor**, **AgentApprovalExecutor**, **HandoffAgentExecutor**, **MagenticAgentExecutor**
- **GroupChatOrchestrator**, **AgentBasedGroupChatOrchestrator**, **MagenticOrchestrator**
- **CheckpointStorage**, **FileCheckpointStorage**, **InMemoryCheckpointStorage**, **WorkflowCheckpoint**
- **SharedState**, **OrchestrationState**
- ~100+ workflow-related classes total

### 2. ag-ui (28 source files)

- **AGUIChatClient**, **AGUIChatOptions** — AG-UI protocol chat client
- **AGUIHttpService** — HTTP service for AG-UI protocol
- **AGUIEventConverter** — Event conversion
- **AgentFrameworkAgent** — Agent wrapper for AG-UI
- **PredictiveStateHandler**, **PredictStateConfig** — State prediction

### 3. declarative (17 source files)

- **AgentDefinition**, **AgentManifest**, **AgentFactory**, **WorkflowFactory**
- **Parser** — YAML/JSON declarative agent parser
- **PromptAgent** — Declaratively configured agent
- **DeclarativeWorkflowBuilder**, **DeclarativeWorkflowState**
- **Connection** types: **AnonymousConnection**, **ApiKeyConnection**, **ReferenceConnection**, **RemoteConnection**
- Action executors (~20+): **QuestionExecutor**, **ConfirmationExecutor**, **SetVariableExecutor**, **EmitEventExecutor**, **InvokeToolExecutor**, **ForeachInitExecutor**, etc.
- **Tool** types: **FunctionTool**, **CodeInterpreterTool**, **McpTool**, **FileSearchTool**, **WebSearchTool**, **OpenApiTool**

### 4. devui (13 source files)

- **DevServer** — Development server for testing agents
- **DeploymentManager**, **DeploymentConfig** — Deployment management
- **ConversationStore**, **InMemoryConversationStore** — Conversation persistence
- **SessionManager** — Session handling
- **AgentFrameworkExecutor**, **OpenAIExecutor** — Execution backends
- **MessageMapper** — Message format conversion
- **EntityDiscovery** — Agent entity discovery

### 5. durabletask (11 source files)

- **DurableAgentExecutor**, **DurableAIAgent**, **DurableAIAgentClient**, **DurableAIAgentWorker**
- **DurableAgentThread**, **DurableAgentState** + state content types (~15 types)
- **AgentEntity**, **AgentCallbackContext**
- **DurableTaskEntityStateProvider**, **DurableAgentProvider**
- **OrchestrationAgentExecutor**, **ClientAgentExecutor**

### 6. purview (7 source files)

- **PurviewClient** — Microsoft Purview API client
- **PurviewChatPolicyMiddleware**, **PurviewPolicyMiddleware** — Policy enforcement middleware
- **PurviewSettings** — Configuration
- ~30 data model types for Purview API interactions

### 7. azure-ai (5 source files)

- **AzureAIClient** — Azure AI Foundry client
- **AzureAIAgentClient** — Agent management client
- **AzureAIAgentsProvider**, **AzureAIProjectAgentProvider** — Agent providers
- **AzureAISettings**, **AzureAIAgentOptions**, **AzureAIProjectAgentOptions**

### 8. azurefunctions (4 source files)

- **AgentFunctionApp** — Azure Functions host for agents
- **AzureFunctionsAgentExecutor** — Functions-based execution
- **AzureFunctionEntityStateProvider** — State persistence via Functions

### 9. Single-File Provider Packages

| Package | Key Type(s) |
|---|---|
| `anthropic` | **AnthropicClient**, **AnthropicChatOptions**, **AnthropicSettings** |
| `bedrock` | **BedrockChatClient**, **BedrockChatOptions**, **BedrockSettings** |
| `ollama` | **OllamaChatClient**, **OllamaChatOptions**, **OllamaSettings** |
| `foundry_local` | **FoundryLocalClient**, **FoundryLocalChatOptions**, **FoundryLocalSettings** |
| `copilotstudio` | **CopilotStudioAgent**, **CopilotStudioSettings** |
| `github_copilot` | **GitHubCopilotAgent**, **GitHubCopilotOptions**, **GitHubCopilotSettings** |
| `a2a` | **A2AAgent** |
| `redis` | **RedisChatMessageStore**, **RedisProvider**, **RedisStoreState** |
| `mem0` | **Mem0Provider**, **MemorySearchResponse_v1_1** |
| `azure-ai-search` | **AzureAISearchContextProvider**, **AzureAISearchSettings** |
| `chatkit` | **ThreadItemConverter** |

## Test Structure

Tests are co-located within each package under `packages/<name>/tests/`. Total: **181 test files** across 22 test directories.

Top test directories by file count:
- core: 69 test files
- durabletask: 18 test files
- ag-ui: 16 test files
- declarative: 14 test files
- azurefunctions: 13 test files
- devui: 12 test files
- purview: 9 test files

Additionally, `python/tests/samples/` exists for sample validation tests.

## Sample Structure

Samples reside under `python/samples/`:

- **getting_started/** — Core samples organized by feature:
  - `agents/`, `chat_client/`, `tools/`, `workflows/`, `threads/`
  - `middleware/`, `context_providers/`, `mcp/`
  - `declarative/`, `devui/`, `durabletask/`, `azure_functions/`
  - `multimodal_input/`, `observability/`, `evaluation/`, `purview_agent/`
- **demos/** — Larger demo scenarios:
  - `chatkit-integration/`, `hosted_agents/`, `m365-agent/`, `workflow_evaluation/`
- **autogen-migration/** — Migration guides from AutoGen
- **semantic-kernel-migration/** — Migration guides from Semantic Kernel

## Core External Dependencies

From the core `pyproject.toml`:
- `pydantic>=2,<3` + `pydantic-settings>=2,<3` — Data models and settings
- `openai>=1.99.0` — OpenAI API client
- `azure-identity>=1,<2` — Azure authentication
- `mcp[ws]>=1.24.0,<2` — Model Context Protocol support
- `opentelemetry-api/sdk/semantic-conventions-ai` — Observability
- `typing-extensions`, `packaging` — Utilities

## Python-Unique Features (Not in .NET)

Based on comparing the package lists:

| Python-Only | Notes |
|---|---|
| `bedrock` (AWS Bedrock) | No .NET equivalent |
| `ollama` | No .NET equivalent |
| `foundry_local` | No .NET equivalent |
| `redis` | .NET uses CosmosNoSql instead |
| `chatkit` | No direct .NET equivalent |
| `azure-ai-search` | Separate package in Python; may be combined in .NET |
| `lab` (gaia, lightning, tau2) | Experimental sub-packages, no .NET equivalent |

.NET has these without Python equivalents:
- `Microsoft.Agents.AI.Hosting` / `Hosting.OpenAI` — Explicit hosting abstractions
- `Microsoft.Agents.AI.AzureAI.Persistent` — Persistent agent state in Azure AI
- `Microsoft.Agents.AI.CosmosNoSql` — Cosmos DB storage (Python uses Redis)
- `Microsoft.Agents.AI.Workflows.Generators` — Source generators (compile-time)
- `Microsoft.Agents.AI.Workflows.Declarative.AzureAI` — Azure AI-specific declarative workflows
- Explicit `Abstractions` / `Shared` separation

## Architecture Observations for C++ Port

1. **Layered architecture**: All packages depend on `core`; only `azurefunctions` has a secondary dependency (on `durabletask`). This clean dependency graph maps well to C++ library dependencies.

2. **Protocol-based abstractions**: Python uses `Protocol` classes (structural typing). C++ equivalent: pure virtual abstract classes (interfaces).

3. **Pydantic models**: Heavy use of Pydantic for data validation and serialization. C++ equivalent: consider nlohmann/json with custom serialization or a code-generation approach.

4. **Async patterns**: Python uses `async/await` throughout. C++ equivalent: `std::future`/`std::coroutine` (C++20 coroutines) or a library like cppcoro.

5. **Streaming via AsyncIterator**: Agent responses use `AsyncIterator[AgentResponseUpdate]`. C++ equivalent: consider a callback-based or observer pattern, or C++20 ranges with coroutines.

6. **Middleware pipeline**: Functional middleware chain pattern. Maps naturally to C++ function objects / `std::function` chains.

7. **Workflows are the largest subsystem** (~35 files, ~100+ classes) — graph-based orchestration with builders, edges, executors, and checkpointing. This is the most complex component to port.

8. **Estimated core port scope**: ~62 source files, ~78,800 total lines (all packages). A minimal C++ port targeting core + OpenAI + workflows would cover ~60% of the codebase.

9. **Python-specific patterns requiring C++ alternatives**:
   - `__init__.py` star exports → C++ header organization with explicit includes
   - Decorators (`@tool`, `@executor`) → Template metaprogramming or macros
   - Dynamic typing in places → `std::variant` or `std::any`
   - Pydantic settings from env vars → Custom configuration loader
   - MCP protocol integration → Direct HTTP/WebSocket client implementation
