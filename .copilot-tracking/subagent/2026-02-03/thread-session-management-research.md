# Thread/Session Management Research

**Date**: 2026-02-03
**Purpose**: Research for Epic 3 Feature 3.3 (Thread/Session Management in Go Port)

---

## Executive Summary

The Agent Framework uses "Session" (in .NET/Go) or "Thread" (in Python) to manage conversation state across agent runs. All three implementations share common patterns:

- Sessions store conversation history and optional service references
- Sessions can be serialized/deserialized for persistence
- Two modes exist: service-managed (external thread ID) vs. locally-managed (in-memory message store)

---

## 1. .NET Session Management

### Core Interface

**File**: [dotnet/src/Microsoft.Agents.AI.Abstractions/AgentSession.cs](dotnet/src/Microsoft.Agents.AI.Abstractions/AgentSession.cs)

```csharp
public abstract class AgentSession
{
    // Serialize session state to JSON for persistence
    public virtual JsonElement Serialize(JsonSerializerOptions? jsonSerializerOptions = null);
    
    // Service locator pattern for extensibility
    public virtual object? GetService(Type serviceType, object? serviceKey = null);
    public TService? GetService<TService>(object? serviceKey = null);
}
```

### Key Session Implementations

| Class | Purpose | Storage |
|-------|---------|---------|
| `InMemoryAgentSession` | Base class for local history | In-memory `ChatHistoryProvider` |
| `ChatClientAgentSession` | ChatClient-based agents | Either `ConversationId` OR `ChatHistoryProvider` |
| `DurableAgentSession` | Durable Task orchestration | External durable storage |
| `A2AAgentSession` | Agent-to-Agent communication | Remote agent state |

### ChatClientAgentSession Key Properties

**File**: [dotnet/src/Microsoft.Agents.AI/ChatClient/ChatClientAgentSession.cs](dotnet/src/Microsoft.Agents.AI/ChatClient/ChatClientAgentSession.cs)

```csharp
public sealed class ChatClientAgentSession : AgentSession
{
    // Option 1: Service-managed thread (external storage)
    public string? ConversationId { get; }
    
    // Option 2: Local message storage
    public ChatHistoryProvider? ChatHistoryProvider { get; }
    
    // Optional context provider
    public AIContextProvider? AIContextProvider { get; }
}
```

**Critical Rule**: Either `ConversationId` OR `ChatHistoryProvider` can be set, but not both. Switching between them is not supported.

### Session Lifecycle in AIAgent

**File**: [dotnet/src/Microsoft.Agents.AI.Abstractions/AIAgent.cs](dotnet/src/Microsoft.Agents.AI.Abstractions/AIAgent.cs)

```csharp
public abstract class AIAgent
{
    // Create new session
    public abstract ValueTask<AgentSession> GetNewSessionAsync(CancellationToken cancellationToken = default);
    
    // Restore session from serialized state
    public abstract ValueTask<AgentSession> DeserializeSessionAsync(
        JsonElement serializedSession, 
        JsonSerializerOptions? jsonSerializerOptions = null, 
        CancellationToken cancellationToken = default);
}
```

---

## 2. .NET Persistence Patterns

### AgentSessionStore Interface

**File**: [dotnet/src/Microsoft.Agents.AI.Hosting/AgentSessionStore.cs](dotnet/src/Microsoft.Agents.AI.Hosting/AgentSessionStore.cs)

```csharp
public abstract class AgentSessionStore
{
    public abstract ValueTask SaveSessionAsync(
        AIAgent agent,
        string conversationId,
        AgentSession session,
        CancellationToken cancellationToken = default);

    public abstract ValueTask<AgentSession> GetSessionAsync(
        AIAgent agent,
        string conversationId,
        CancellationToken cancellationToken = default);
}
```

### InMemoryAgentSessionStore Implementation

**File**: [dotnet/src/Microsoft.Agents.AI.Hosting/Local/InMemoryAgentSessionStore.cs](dotnet/src/Microsoft.Agents.AI.Hosting/Local/InMemoryAgentSessionStore.cs)

```csharp
public sealed class InMemoryAgentSessionStore : AgentSessionStore
{
    private readonly ConcurrentDictionary<string, JsonElement> _threads = new();

    public override ValueTask SaveSessionAsync(...)
    {
        var key = GetKey(conversationId, agent.Id);
        this._threads[key] = session.Serialize();
        return default;
    }

    public override async ValueTask<AgentSession> GetSessionAsync(...)
    {
        // Returns existing or creates new session
    }

    private static string GetKey(string conversationId, string agentId) 
        => $"{agentId}:{conversationId}";
}
```

---

## 3. Python Thread Management

### Core Classes

**File**: [python/packages/core/agent_framework/_threads.py](python/packages/core/agent_framework/_threads.py)

```python
class AgentThread:
    """Either service-managed or locally managed thread."""
    
    def __init__(
        self,
        *,
        service_thread_id: str | None = None,
        message_store: ChatMessageStoreProtocol | None = None,
        context_provider: ContextProvider | None = None,
    ):
        # Same rule as .NET: service_thread_id OR message_store, not both
        if service_thread_id is not None and message_store is not None:
            raise AgentThreadException("...")

class ChatMessageStore:
    """In-memory implementation of ChatMessageStoreProtocol."""
    
    async def list_messages(self) -> list[ChatMessage]: ...
    async def add_messages(self, messages: Sequence[ChatMessage]) -> None: ...
    async def serialize(self, **kwargs) -> dict[str, Any]: ...
    @classmethod
    async def deserialize(cls, serialized_store_state, **kwargs) -> "ChatMessageStore": ...
```

### ChatMessageStoreProtocol

```python
class ChatMessageStoreProtocol(Protocol):
    async def list_messages(self) -> list[ChatMessage]: ...
    async def add_messages(self, messages: Sequence[ChatMessage]) -> None: ...
    async def serialize(self, **kwargs) -> dict[str, Any]: ...
    @classmethod
    async def deserialize(cls, serialized_store_state, **kwargs) -> "ChatMessageStoreProtocol": ...
    async def update_from_state(self, serialized_store_state, **kwargs) -> None: ...
```

### Redis Persistence

**File**: [python/packages/redis/agent_framework_redis/_chat_message_store.py](python/packages/redis/agent_framework_redis/_chat_message_store.py)

```python
class RedisChatMessageStore:
    """Redis-backed implementation using Redis Lists."""
    
    def __init__(
        self,
        redis_url: str | None = None,
        thread_id: str | None = None,
        key_prefix: str = "chat_messages",
        max_messages: int | None = None,
        ...
    ): ...
```

---

## 4. Go Implementation Status

### Session Interface

**File**: [go/agent/session.go](go/agent/session.go)

```go
type Session interface {
    ID() string
    Messages() []Message
    AddMessage(msg Message)
    Serialize() (json.RawMessage, error)
    GetService(serviceType reflect.Type) interface{}
}
```

### InMemorySession Implementation

**File**: [go/agent/session.go](go/agent/session.go)

```go
type InMemorySession struct {
    services map[reflect.Type]interface{}
    messages []Message
    id       string
    mu       sync.RWMutex
}

func NewInMemorySession() *InMemorySession
func NewInMemorySessionWithID(id string) *InMemorySession
func RestoreInMemorySession(data json.RawMessage) (*InMemorySession, error)
```

### ChatAgent Session

**File**: [go/chatagent/session.go](go/chatagent/session.go)

```go
type Session struct {
    id         string
    agentID    string
    messages   []chat.Message
    services   map[reflect.Type]interface{}
    mu         sync.RWMutex
    createdAt  time.Time
    modifiedAt time.Time
}

// Serialization state
type sessionState struct {
    ID         string         `json:"id"`
    AgentID    string         `json:"agent_id,omitempty"`
    Messages   []chat.Message `json:"messages"`
    CreatedAt  time.Time      `json:"created_at"`
    ModifiedAt time.Time      `json:"modified_at"`
}
```

### Agent Interface Methods

**File**: [go/agent/agent.go](go/agent/agent.go)

```go
type Agent interface {
    // ... other methods
    NewSession(ctx context.Context) (Session, error)
    RestoreSession(ctx context.Context, data json.RawMessage) (Session, error)
}
```

---

## 5. Gap Analysis for Go Implementation

### Currently Implemented ✅

| Feature | Status |
|---------|--------|
| `Session` interface | ✅ Complete |
| `InMemorySession` | ✅ Complete |
| `chatagent.Session` | ✅ Complete |
| JSON serialization/deserialization | ✅ Complete |
| Thread-safe operations (mutex) | ✅ Complete |
| Service locator pattern (`GetService`) | ✅ Complete |
| Timestamps (`createdAt`, `modifiedAt`) | ✅ Complete |

### Missing Features ❌

| Feature | .NET/Python | Go Status |
|---------|-------------|-----------|
| `SessionStore` interface | `AgentSessionStore` | ❌ Not implemented |
| `InMemorySessionStore` | `InMemoryAgentSessionStore` | ❌ Not implemented |
| Service-managed sessions (ConversationId) | Supported | ❌ Not implemented |
| `ChatHistoryProvider` equivalent | `ChatHistoryProvider` | ❌ Not implemented |
| `AIContextProvider` equivalent | `AIContextProvider` | ❌ Not implemented |
| Redis/persistent storage | `RedisChatMessageStore` | ❌ Not implemented |
| Session cloning | Limited in .NET | ✅ `Clone()` method exists |

---

## 6. Recommendations for Epic 3 Feature 3.3

### Priority 1: Core Interfaces

1. **Add `SessionStore` interface** (align with `AgentSessionStore`):

```go
type SessionStore interface {
    Save(ctx context.Context, agent Agent, conversationID string, session Session) error
    Get(ctx context.Context, agent Agent, conversationID string) (Session, error)
}
```

2. **Add `InMemorySessionStore` implementation**:

```go
type InMemorySessionStore struct {
    sessions sync.Map // key: "{agentID}:{conversationID}"
}
```

### Priority 2: Dual-Mode Sessions

3. **Support service-managed sessions** (add `ConversationID` field):

```go
type Session struct {
    // ... existing fields
    conversationID string // For service-managed sessions
    messageStore   MessageStore // For local sessions
}
```

### Priority 3: History Providers

4. **Add `ChatHistoryProvider` interface**:

```go
type ChatHistoryProvider interface {
    GetMessages(ctx context.Context) ([]chat.Message, error)
    AddMessages(ctx context.Context, msgs ...chat.Message) error
    Serialize() (json.RawMessage, error)
}
```

---

## 7. API Comparison Table

| Concept | .NET | Python | Go |
|---------|------|--------|-----|
| Session type | `AgentSession` | `AgentThread` | `Session` interface |
| In-memory session | `InMemoryAgentSession` | `AgentThread` + `ChatMessageStore` | `InMemorySession` |
| Chat client session | `ChatClientAgentSession` | `AgentThread` | `chatagent.Session` |
| Create session | `GetNewSessionAsync()` | `agent.get_new_thread()` | `NewSession(ctx)` |
| Restore session | `DeserializeSessionAsync()` | `AgentThread.deserialize()` | `RestoreSession(ctx, data)` |
| Session store | `AgentSessionStore` | N/A (via ChatMessageStore) | ❌ Not implemented |
| Service thread ID | `ConversationId` | `service_thread_id` | ❌ Not implemented |
| Message storage | `ChatHistoryProvider` | `ChatMessageStoreProtocol` | Direct `messages` field |

---

## 8. References

### Key Files

- .NET Session: [AgentSession.cs](dotnet/src/Microsoft.Agents.AI.Abstractions/AgentSession.cs)
- .NET ChatClientSession: [ChatClientAgentSession.cs](dotnet/src/Microsoft.Agents.AI/ChatClient/ChatClientAgentSession.cs)
- .NET SessionStore: [AgentSessionStore.cs](dotnet/src/Microsoft.Agents.AI.Hosting/AgentSessionStore.cs)
- Python Thread: [_threads.py](python/packages/core/agent_framework/_threads.py)
- Python Redis: [_chat_message_store.py](python/packages/redis/agent_framework_redis/_chat_message_store.py)
- Go Session: [session.go](go/agent/session.go)
- Go ChatAgent Session: [session.go](go/chatagent/session.go)
