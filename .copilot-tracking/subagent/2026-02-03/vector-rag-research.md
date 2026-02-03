# Vector Search and RAG Integration Research

## Research Summary

**Date:** 2026-02-03  
**Epic:** 3 - Memory & Context Providers  
**Feature:** 3.5 - Vector Search and RAG Integration

---

## 1. Vector/RAG Types in Each Language

### .NET Implementation

#### Types Found

| Type | Location | Purpose |
|------|----------|---------|
| `TextSearchProvider` | [TextSearchProvider.cs](dotnet/src/Microsoft.Agents.AI/TextSearchProvider.cs) | RAG provider using pluggable text search backend |
| `ChatHistoryMemoryProvider` | [ChatHistoryMemoryProvider.cs](dotnet/src/Microsoft.Agents.AI/Memory/ChatHistoryMemoryProvider.cs) | Vector store-backed chat history retrieval |
| `HostedFileSearchTool` | Multiple locations (declarative extensions, OpenAI extensions) | Provider-hosted vector search tool |
| `HostedVectorStoreContent` | [DurableAgentStateHostedVectorStoreContent.cs](dotnet/src/Microsoft.Agents.AI.DurableTask/State/DurableAgentStateHostedVectorStoreContent.cs) | Vector store content wrapper for durable agents |
| `FileSearchToolDefinition` | OpenAI integration | File search tool definition for OpenAI Assistants |
| `VectorStore` | Via `Microsoft.Extensions.VectorData` NuGet package | Generic vector store abstraction |
| `VectorStoreCollection` | Via `Microsoft.Extensions.VectorData` NuGet package | Generic vector store collection interface |

#### Key Patterns in .NET

1. **TextSearchProvider** - Extensible RAG with pluggable search backend
   - Supports two behaviors: `BeforeAIInvoke` (automatic) and `OnDemandFunctionCalling` (tool-based)
   - Uses `Func<string, CancellationToken, Task<IEnumerable<TextSearchResult>>>` delegate for search
   - Maintains recent message memory for multi-turn context
   - Formats search results with configurable context and citations prompts

2. **ChatHistoryMemoryProvider** - Semantic search over conversation history
   - Uses `Microsoft.Extensions.VectorData.VectorStore` abstraction
   - Stores chat messages with embeddings for similarity search
   - Supports scoped storage/search by user, session, or custom dimensions
   - Configurable max results and context prompt

3. **Hosted File Search** - Provider-managed vector search
   - `HostedFileSearchTool` maps to OpenAI's File Search / Azure AI vector store
   - `HostedVectorStoreContent` wraps vector store IDs for tool inputs
   - Managed by provider infrastructure (OpenAI, Azure AI)

---

### Python Implementation

#### Types Found

| Type | Location | Purpose |
|------|----------|---------|
| `Mem0Provider` | [_provider.py](python/packages/mem0/agent_framework_mem0/_provider.py) | Mem0 memory service integration |
| `HostedWebSearchTool` | [_tools.py](python/packages/core/agent_framework/_tools.py#L284) | Provider-hosted web search |
| `HostedFileSearchTool` | [_tools.py](python/packages/core/agent_framework/_tools.py#L478) | Provider-hosted file/vector search |
| `HostedVectorStoreContent` | Core framework content types | Vector store content wrapper |
| `ContextProvider` | Core framework protocol | Base class for context providers |

#### Key Patterns in Python

1. **Mem0Provider** - External memory service integration
   - Integrates with Mem0 cloud or open-source memory service
   - Uses `search()` API for semantic memory retrieval in `invoking()`
   - Uses `add()` API to store conversation memories in `invoked()`
   - Supports scoping by `user_id`, `agent_id`, `application_id`, `thread_id`
   - Formats retrieved memories into context messages

2. **HostedFileSearchTool** - Provider-managed file search
   - Wraps vector store IDs as `HostedVectorStoreContent`
   - Configurable `max_results` parameter
   - Used with OpenAI Assistants and Azure AI agents

3. **ContextProvider Protocol** - Extensible context injection
   - `invoking()` method for pre-invocation context injection
   - `invoked()` method for post-invocation memory storage
   - `AggregateContextProvider` for combining multiple providers

---

### Go Implementation

#### Types Found

| Type | Location | Purpose |
|------|----------|---------|
| `HostedWebSearchTool` | [hosted.go](go/tool/hosted.go#L37) | Provider-hosted web search |
| `HostedFileSearchTool` | [hosted.go](go/tool/hosted.go#L237) | Provider-hosted vector/file search |
| `HostedVectorStoreContent` | Not yet implemented | Vector store content wrapper |
| `ContextProvider` | [context_provider.go](go/agent/context_provider.go) | Base interface for context providers |
| `ContextProviderWithLifecycle` | [context_provider.go](go/agent/context_provider.go#L67) | Extended interface with `Invoked` and `SessionCreated` |
| `AggregateContextProvider` | [context_provider.go](go/agent/context_provider.go#L109) | Combines multiple context providers |
| `FileSearchRanking` | [hosted.go](go/tool/hosted.go#L233) | Ranking configuration for file search |

#### Current Go Implementation Status

1. **HostedFileSearchTool** ✅ Implemented
   - Located in `go/tool/hosted.go`
   - Supports `VectorStoreIDs`, `MaxResults`, `Ranking` configuration
   - Implements `HostedTool` interface with `ProviderConfig()` method
   - Aligns with Python `HostedFileSearchTool`

2. **ContextProvider Interface** ✅ Implemented
   - Base `ContextProvider` interface with `Invoking()` method
   - Extended `ContextProviderWithLifecycle` with `Invoked()` and `SessionCreated()`
   - `Context` struct with `Instructions`, `Messages`, `Tools` fields
   - `AggregateContextProvider` for combining multiple providers

3. **Concrete Providers** ❌ Not Implemented
   - No Mem0 provider
   - No TextSearchProvider equivalent
   - No ChatHistoryMemoryProvider equivalent
   - No Azure AI Search integration

---

## 2. Integration Patterns

### Pattern 1: Provider-Hosted Vector Search (All Languages)

```
┌─────────────┐     ┌──────────────────────┐     ┌─────────────────┐
│   Agent     │────>│ HostedFileSearchTool │────>│ Provider API    │
│             │     │ (vector_store_ids)   │     │ (OpenAI/Azure)  │
└─────────────┘     └──────────────────────┘     └─────────────────┘
                              │                           │
                              │     ┌─────────────────────┤
                              │     │ Vector Store        │
                              │     │ (managed by provider)│
                              └────>└─────────────────────┘
```

**Usage:**
- Create vector store via provider SDK
- Upload documents
- Create `HostedFileSearchTool` with vector store IDs
- Add tool to agent configuration

### Pattern 2: Context Provider RAG (TextSearchProvider/.NET)

```
┌─────────────┐     ┌─────────────────────┐     ┌─────────────────┐
│   Agent     │────>│ TextSearchProvider  │────>│ Search Backend  │
│             │     │ (AIContextProvider) │     │ (Azure Search,  │
└─────────────┘     └─────────────────────┘     │  Custom, etc.)  │
       │                      │                  └─────────────────┘
       │                      │
       │              InvokingAsync()
       │                      │
       │            ┌─────────┴─────────┐
       │            │                   │
       │     BeforeAIInvoke      OnDemandFunctionCalling
       │     (auto inject)       (expose as tool)
       └────────────────────────────────────────────────────────────
```

**Behaviors:**
1. `BeforeAIInvoke`: Automatically search before each invocation, inject results as messages
2. `OnDemandFunctionCalling`: Expose search as a function tool the model can call

### Pattern 3: Memory Service Integration (Mem0/Python)

```
┌─────────────┐     ┌───────────────┐     ┌─────────────────┐
│   Agent     │────>│ Mem0Provider  │────>│ Mem0 API        │
│             │     │               │     │ (Cloud/OSS)     │
└─────────────┘     └───────────────┘     └─────────────────┘
       │                    │                      │
       │            invoking()                     │
       │            ├── search() ─────────────────>│
       │            └── inject context             │
       │                                           │
       │            invoked()                      │
       │            └── add() ────────────────────>│
       └───────────────────────────────────────────┘
```

**Features:**
- Semantic search for relevant memories
- Automatic memory extraction from conversations
- Scoping by user/agent/application/thread

### Pattern 4: Vector Store Memory Provider (ChatHistoryMemoryProvider/.NET)

```
┌─────────────┐     ┌────────────────────────┐     ┌─────────────────┐
│   Agent     │────>│ ChatHistoryMemoryProvider│───>│ VectorStore     │
│             │     │ (AIContextProvider)      │    │ (MEAI abstract) │
└─────────────┘     └────────────────────────┘     └─────────────────┘
       │                      │                            │
       │              InvokingAsync()                      │
       │              ├── Semantic search ────────────────>│
       │              └── Return related history           │
       │                                                   │
       │              InvokedAsync()                       │
       │              └── Store messages ─────────────────>│
       └───────────────────────────────────────────────────┘
```

**Features:**
- Uses `Microsoft.Extensions.VectorData.VectorStore` abstraction
- Compatible with any MEAI vector store implementation
- Scoped storage and search

---

## 3. Implementation Status in Go

### ✅ Implemented

| Component | File | Status |
|-----------|------|--------|
| `HostedFileSearchTool` | `go/tool/hosted.go` | Complete |
| `HostedWebSearchTool` | `go/tool/hosted.go` | Complete |
| `HostedCodeInterpreterTool` | `go/tool/hosted.go` | Complete |
| `HostedMCPTool` | `go/tool/hosted.go` | Complete |
| `HostedImageGenerationTool` | `go/tool/hosted.go` | Complete |
| `FileSearchRanking` | `go/tool/hosted.go` | Complete |
| `ContextProvider` interface | `go/agent/context_provider.go` | Complete |
| `ContextProviderWithLifecycle` | `go/agent/context_provider.go` | Complete |
| `AggregateContextProvider` | `go/agent/context_provider.go` | Complete |
| `BaseContextProvider` | `go/agent/context_provider.go` | Complete |
| `Context` struct | `go/agent/context_provider.go` | Complete |

### ❌ Not Implemented (Required for Feature 3.5)

| Component | Reference | Priority |
|-----------|-----------|----------|
| `TextSearchProvider` | .NET implementation | High |
| `ChatHistoryMemoryProvider` / `VectorStoreProvider` | .NET implementation | Medium |
| `Mem0Provider` | Python implementation | Medium |
| `VectorStore` interface | MEAI VectorData | High |
| `VectorStoreCollection` interface | MEAI VectorData | High |
| Azure AI Search integration | .NET/Python samples | Low |
| `HostedVectorStoreContent` content type | Python/DurableTask | Medium |

---

## 4. Recommendations for Go Implementation

### Phase 1: Core Abstractions

1. **VectorStore Interface** (High Priority)
   ```go
   package vectorstore
   
   type VectorStore interface {
       GetCollection(ctx context.Context, name string, dims int) (Collection, error)
   }
   
   type Collection interface {
       Upsert(ctx context.Context, records []Record) error
       Search(ctx context.Context, vector []float32, opts SearchOptions) ([]SearchResult, error)
       Delete(ctx context.Context, ids []string) error
   }
   ```

2. **TextSearchProvider** (High Priority)
   ```go
   package memory
   
   type TextSearchProvider struct {
       searchFunc func(ctx context.Context, query string) ([]TextSearchResult, error)
       behavior   SearchBehavior // BeforeInvoke or OnDemand
       // ...
   }
   ```

### Phase 2: Memory Providers

3. **Mem0Provider** (Medium Priority)
   - Port from Python implementation
   - Support cloud and OSS Mem0 backends

4. **ChatHistoryMemoryProvider** (Medium Priority)
   - Use VectorStore interface
   - Support scoped storage/search

### Phase 3: Provider Integrations

5. **Azure AI Search Client** (Lower Priority)
   - Implement VectorStore interface for Azure AI Search

6. **In-Memory Vector Store** (Testing)
   - Simple implementation for testing and examples

---

## 5. References

### .NET Files
- [TextSearchProvider.cs](dotnet/src/Microsoft.Agents.AI/TextSearchProvider.cs)
- [ChatHistoryMemoryProvider.cs](dotnet/src/Microsoft.Agents.AI/Memory/ChatHistoryMemoryProvider.cs)
- [AIContextProvider.cs](dotnet/src/Microsoft.Agents.AI.Abstractions/AIContextProvider.cs)
- [OpenAIAssistantClientExtensions.cs](dotnet/src/Microsoft.Agents.AI.OpenAI/Extensions/OpenAIAssistantClientExtensions.cs)

### Python Files
- [_provider.py](python/packages/mem0/agent_framework_mem0/_provider.py) - Mem0 provider
- [_tools.py](python/packages/core/agent_framework/_tools.py) - Hosted tools
- [mem0_basic.py](python/samples/getting_started/context_providers/mem0/mem0_basic.py) - Sample
- [openai_assistants_with_file_search.py](python/samples/getting_started/agents/openai/openai_assistants_with_file_search.py) - Sample

### Go Files
- [hosted.go](go/tool/hosted.go) - Hosted tools including HostedFileSearchTool
- [context_provider.go](go/agent/context_provider.go) - ContextProvider interface and implementations

### Documentation
- [golang-port-plan.md](docs/design/golang-port-plan.md) - Overall Go port plan
- [memory-context-providers-research.md](.copilot-tracking/subagent/2026-02-03/memory-context-providers-research.md) - Related research

---

## 6. Conclusion

The Go implementation has a solid foundation for vector search and RAG integration:

- ✅ **Hosted tools are complete** - `HostedFileSearchTool`, `HostedWebSearchTool` fully implemented
- ✅ **ContextProvider interface is complete** - Matches .NET/Python patterns
- ❌ **No concrete providers** - TextSearchProvider, Mem0Provider need implementation
- ❌ **No VectorStore abstraction** - Need Go equivalent of MEAI VectorData

**Recommended Implementation Order:**
1. Define `VectorStore` and `VectorStoreCollection` interfaces
2. Implement `TextSearchProvider` with pluggable search backend
3. Create in-memory vector store for testing
4. Port `Mem0Provider` from Python
5. Implement `ChatHistoryMemoryProvider` using VectorStore interface
