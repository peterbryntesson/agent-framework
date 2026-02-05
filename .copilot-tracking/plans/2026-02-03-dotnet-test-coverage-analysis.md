# .NET Test Coverage Analysis Report for Epic 2 Features

**Date:** February 3, 2026  
**Purpose:** Analyze .NET test patterns to inform Go test coverage requirements

---

## 1. Test Files Overview

### Microsoft.Agents.AI.UnitTests

| Test Class | File Path | Focus Area |
|------------|-----------|------------|
| `ChatClientAgentTests` | [ChatClient/ChatClientAgentTests.cs](dotnet/tests/Microsoft.Agents.AI.UnitTests/ChatClient/ChatClientAgentTests.cs) | Core agent invocation |
| `ChatClientAgentOptionsTests` | [ChatClient/ChatClientAgentOptionsTests.cs](dotnet/tests/Microsoft.Agents.AI.UnitTests/ChatClient/ChatClientAgentOptionsTests.cs) | Options configuration |
| `ChatClientAgent_ChatOptionsMergingTests` | [ChatClient/ChatClientAgent_ChatOptionsMergingTests.cs](dotnet/tests/Microsoft.Agents.AI.UnitTests/ChatClient/ChatClientAgent_ChatOptionsMergingTests.cs) | Options merging precedence |
| `ChatClientAgentSessionTests` | [ChatClient/ChatClientAgentSessionTests.cs](dotnet/tests/Microsoft.Agents.AI.UnitTests/ChatClient/ChatClientAgentSessionTests.cs) | Session management |
| `FunctionInvocationDelegatingAgentTests` | [FunctionInvocationDelegatingAgentTests.cs](dotnet/tests/Microsoft.Agents.AI.UnitTests/FunctionInvocationDelegatingAgentTests.cs) | Tool invocation middleware |
| `AnonymousDelegatingAIAgentTests` | [AnonymousDelegatingAIAgentTests.cs](dotnet/tests/Microsoft.Agents.AI.UnitTests/AnonymousDelegatingAIAgentTests.cs) | Anonymous middleware patterns |
| `LoggingAgentTests` | [LoggingAgentTests.cs](dotnet/tests/Microsoft.Agents.AI.UnitTests/LoggingAgentTests.cs) | Logging middleware |
| `OpenTelemetryAgentTests` | [OpenTelemetryAgentTests.cs](dotnet/tests/Microsoft.Agents.AI.UnitTests/OpenTelemetryAgentTests.cs) | Observability instrumentation |
| `AIAgentBuilderTests` | [AIAgentBuilderTests.cs](dotnet/tests/Microsoft.Agents.AI.UnitTests/AIAgentBuilderTests.cs) | Agent builder patterns |

### Microsoft.Agents.AI.Abstractions.UnitTests

| Test Class | File Path | Focus Area |
|------------|-----------|------------|
| `AIAgentTests` | [AIAgentTests.cs](dotnet/tests/Microsoft.Agents.AI.Abstractions.UnitTests/AIAgentTests.cs) | Base agent behavior |
| `DelegatingAIAgentTests` | [DelegatingAIAgentTests.cs](dotnet/tests/Microsoft.Agents.AI.Abstractions.UnitTests/DelegatingAIAgentTests.cs) | Delegating agent pattern |
| `AIContextProviderTests` | [AIContextProviderTests.cs](dotnet/tests/Microsoft.Agents.AI.Abstractions.UnitTests/AIContextProviderTests.cs) | Context provider |
| `AgentResponseTests` | [AgentResponseTests.cs](dotnet/tests/Microsoft.Agents.AI.Abstractions.UnitTests/AgentResponseTests.cs) | Response serialization |
| `ChatHistoryProviderTests` | [ChatHistoryProviderTests.cs](dotnet/tests/Microsoft.Agents.AI.Abstractions.UnitTests/ChatHistoryProviderTests.cs) | Chat history provider |
| `InMemoryChatHistoryProviderTests` | [InMemoryChatHistoryProviderTests.cs](dotnet/tests/Microsoft.Agents.AI.Abstractions.UnitTests/InMemoryChatHistoryProviderTests.cs) | In-memory history management |

---

## 2. Key Test Scenarios Covered

### 2.1 ChatClientAgent Tests

#### Constructor Tests
- ✅ Agent definition with all properties (Id, Name, Description, Instructions)
- ✅ ChatClient wrapper verification

#### RunAsync Tests
- ✅ Basic invocation and response verification
- ✅ `ArgumentNullException` when messages is null
- ✅ Passing `ChatOptions` through `ChatClientAgentRunOptions`
- ✅ Null `ChatOptions` with regular `AgentRunOptions`
- ✅ Base instructions included in options
- ✅ Author name set on all response messages (including null case)
- ✅ Session with `ChatHistoryProvider` retrieves messages
- ✅ Works without instructions (null/empty)
- ✅ Works with empty message collection
- ✅ `AIContextProvider` invocation and context merging (messages, instructions, tools)
- ✅ `AIContextProvider` invocation on downstream failure

### 2.2 FunctionInvocationDelegatingAgent Tests (Tool Middleware)

#### Basic Functionality
- ✅ Constructor with valid parameters
- ✅ `ArgumentNullException` for null inner agent
- ✅ Properties delegation (Id, Name, Description)

#### Function Invocation
- ✅ Middleware invoked for function calls (no options)
- ✅ Middleware invoked with `AgentRunOptions`
- ✅ `NotSupportedException` for custom run options
- ✅ Middleware invoked with `ChatClientAgentRunOptions`
- ✅ Multiple function calls trigger middleware for each
- ✅ `FunctionInvocationContext` contains correct values (function name, arguments)

#### Error Handling
- ✅ Pre-invocation exceptions surface to caller
- ✅ Function exceptions can be handled by middleware
- ✅ Middleware can modify function results

#### Chaining
- ✅ Multiple function middleware execute in correct order
- ✅ Function middleware works with running middleware

#### Streaming
- ✅ Function middleware works with streaming responses

### 2.3 OpenTelemetry Agent Tests (Observability)

#### Constructor & Configuration
- ✅ `ArgumentNullException` for null inner agent
- ✅ Null source name is valid
- ✅ `EnableSensitiveData` property roundtrips

#### Telemetry Logging
- ✅ Activity creation with correct display name
- ✅ Server address and port tags
- ✅ Operation name, provider name, agent metadata tags
- ✅ Request model tags
- ✅ Response ID and token usage tags
- ✅ Sensitive data conditional logging (input/output messages, system instructions, tool definitions)
- ✅ Works without chat options
- ✅ Works with chat options (all parameters captured)
- ✅ Streaming vs non-streaming both tested
- ✅ Activity duration measurement

### 2.4 DelegatingAIAgent Tests

#### Property Delegation
- ✅ Id delegates to inner agent
- ✅ Name delegates to inner agent
- ✅ Description delegates to inner agent

#### Method Delegation
- ✅ `GetNewSessionAsync` delegates correctly
- ✅ `RunAsync` delegates with correct parameters
- ✅ `RunStreamingAsync` delegates correctly

#### GetService
- ✅ Returns self for compatible type with null key
- ✅ Delegates to inner if key is not null
- ✅ Delegates to inner if type is incompatible

### 2.5 AIAgentBuilder Tests

#### Constructor
- ✅ `ArgumentNullException` for null inner agent
- ✅ `ArgumentNullException` for null factory

#### Build Behavior
- ✅ Returns inner agent when no middleware
- ✅ Works with factory function

#### Use Method
- ✅ Simple factory applies middleware
- ✅ Service provider factory applies middleware
- ✅ Multiple middleware applied in correct order (first added is outermost)
- ✅ `ArgumentNullException` for null factories
- ✅ `InvalidOperationException` when middleware returns null

---

## 3. Mocking Patterns Used

### 3.1 Moq for Interface Mocking

```csharp
// IChatClient mocking pattern
Mock<IChatClient> mockService = new();
mockService.Setup(
    s => s.GetResponseAsync(
        It.IsAny<IEnumerable<ChatMessage>>(),
        It.IsAny<ChatOptions>(),
        It.IsAny<CancellationToken>()))
    .ReturnsAsync(new ChatResponse([new(ChatRole.Assistant, "response")]));
```

### 3.2 Protected Method Mocking

```csharp
// For abstract base classes with protected methods
this._innerAgentMock.Protected()
    .Setup<Task<AgentResponse>>("RunCoreAsync",
        ItExpr.IsAny<IEnumerable<ChatMessage>>(),
        ItExpr.IsAny<AgentSession?>(),
        ItExpr.IsAny<AgentRunOptions?>(),
        ItExpr.IsAny<CancellationToken>())
    .ReturnsAsync(this._testResponse);
```

### 3.3 Callback Capture Pattern

```csharp
// Capture parameters for verification
ChatOptions? capturedChatOptions = null;
mockService.Setup(...)
    .Callback<IEnumerable<ChatMessage>, ChatOptions, CancellationToken>((msgs, opts, ct) =>
        capturedChatOptions = opts)
    .ReturnsAsync(...);
```

### 3.4 Sequence Setup for Multi-Call Scenarios

```csharp
// For function call -> final response pattern
mockChatClient.SetupSequence(c => c.GetResponseAsync(...))
    .ReturnsAsync(responseWithFunctionCall)
    .ReturnsAsync(finalResponse);
```

### 3.5 Test Agent Implementation

```csharp
// TestAIAgent with function delegates for flexible test setup
var innerAgent = new TestAIAgent
{
    NameFunc = () => "TestAgent",
    DescriptionFunc = () => "This is a test agent.",
    RunAsyncFunc = async (messages, session, options, cancellationToken) =>
    {
        return new AgentResponse(...);
    },
    RunStreamingAsyncFunc = CallbackAsync,
};
```

---

## 4. Edge Cases Tested

### 4.1 Null and Empty Handling
- ✅ Null messages parameter throws `ArgumentNullException`
- ✅ Empty message collection works correctly
- ✅ Null instructions work correctly
- ✅ Null session handling
- ✅ Null chat options handling
- ✅ Null service key in `GetService`

### 4.2 Cancellation
- ✅ `OperationCanceledException` is logged appropriately
- ✅ Cancellation token is propagated through middleware chain

### 4.3 Exception Handling
- ✅ Middleware exceptions propagate correctly
- ✅ Inner agent exceptions can be caught by middleware
- ✅ Exception details logged at appropriate levels

### 4.4 Streaming Edge Cases
- ✅ Streaming with function calls
- ✅ Empty streaming responses
- ✅ Streaming with usage content

### 4.5 Serialization Edge Cases
- ✅ JSON serialization roundtrips
- ✅ Custom AI content types
- ✅ Experimental content types (`FunctionApprovalRequestContent`)

---

## 5. Go Test Coverage Gap Analysis

### 5.1 Currently Present in Go

| Feature | Go Test File | Status |
|---------|-------------|--------|
| FunctionTool creation | `tool/function_test.go` | ✅ Good coverage |
| HostedTool types | `tool/hosted_test.go` | ✅ Good coverage |
| AsTool functionality | `chatagent/astool_test.go` | ✅ Basic coverage |
| DelegatingAgent | `agent/delegating_test.go` | ✅ Basic coverage |
| MiddlewareAgent | `agent/middleware_agent_test.go` | ✅ Good coverage |

### 5.2 Recommended Additions for Go

#### High Priority

| Gap | .NET Reference | Recommendation |
|-----|---------------|----------------|
| **ChatOptions merging precedence** | `ChatClientAgent_ChatOptionsMergingTests` | Add tests for options merging (agent-level vs request-level vs run-options) |
| **AIContextProvider integration** | `ChatClientAgentTests` lines 340-380 | Add context provider invocation tests with message/tool merging |
| **Function invocation context validation** | `FunctionInvocationDelegatingAgentTests` lines 300-330 | Test that function context contains correct values |
| **Multiple function middleware chaining** | `FunctionInvocationDelegatingAgentTests` lines 640-680 | Test execution order with nested middleware |

#### Medium Priority

| Gap | .NET Reference | Recommendation |
|-----|---------------|----------------|
| **OpenTelemetry sensitive data toggle** | `OpenTelemetryAgentTests` lines 60-270 | Add tests for `EnableSensitiveData` flag |
| **Telemetry tag completeness** | `OpenTelemetryAgentTests` lines 400-600 | Verify all expected tags are present |
| **Agent builder null safety** | `AIAgentBuilderTests` | Add tests for null factory/agent handling |
| **Session serialization roundtrip** | `ChatClientAgentSessionTests` | Test session serialize/deserialize |

#### Lower Priority

| Gap | .NET Reference | Recommendation |
|-----|---------------|----------------|
| **Logging level tests** | `LoggingAgentTests` | Test Debug vs Trace level logging |
| **Chat history provider edge cases** | `InMemoryChatHistoryProviderTests` | Test reducer triggers and edge cases |
| **Response text concatenation** | `AgentResponseTests` | Test multi-message text aggregation |

### 5.3 Specific Test Cases to Add

#### For `chatagent/astool_test.go`

```go
// Add these test scenarios:

func TestAsTool_RunOptionsPassthrough(t *testing.T) {
    // Verify that run options are correctly passed to the inner agent
}

func TestAsTool_ErrorFromInnerAgent(t *testing.T) {
    // Verify error handling when inner agent returns an error
}

func TestAsTool_EmptyResponse(t *testing.T) {
    // Verify handling of empty/nil response from agent
}

func TestAsTool_MultiMessageResponse(t *testing.T) {
    // Verify text concatenation from multiple response messages
}
```

#### For `tool/function_test.go`

```go
// Add these test scenarios:

func TestFunctionTool_InvocationContext(t *testing.T) {
    // Verify context is passed correctly through invocation
}

func TestFunctionTool_ConcurrentInvocation(t *testing.T) {
    // Verify thread-safety of concurrent tool invocations
}

func TestFunctionTool_ResultSerialization(t *testing.T) {
    // Verify result is correctly serialized to JSON
}
```

#### For Observability Tests

```go
// Add new file: observability/otel_agent_test.go

func TestOtelAgent_SensitiveDataDisabled(t *testing.T) {
    // Verify no sensitive data in spans when disabled
}

func TestOtelAgent_AllTagsPresent(t *testing.T) {
    // Verify gen_ai.* tags are correctly set
}

func TestOtelAgent_ErrorSpanStatus(t *testing.T) {
    // Verify span status is ERROR on agent failure
}
```

---

## 6. Summary

### Key Takeaways for Go Implementation

1. **.NET uses extensive mock verification** - Go tests should similarly verify that inner agents are called with correct parameters

2. **.NET tests multiple configuration scenarios** - Go needs more tests for options merging and precedence

3. **.NET has comprehensive error path testing** - Go should add tests for error propagation through middleware chains

4. **.NET validates serialization roundtrips** - Go tests should verify JSON serialization for sessions and responses

5. **.NET tests both streaming and non-streaming** - Go should ensure parity in streaming test coverage

6. **.NET uses Theory/InlineData for parameterized tests** - Go should use table-driven tests for similar coverage

### Priority Action Items

1. **Add ChatOptions merging tests** to `chatagent/options_test.go`
2. **Add AIContextProvider integration tests** to `chatagent/context_provider_test.go`
3. **Add function invocation middleware tests** to `chatagent/toolloop_test.go`
4. **Add OpenTelemetry sensitive data tests** to `observability/` package
5. **Add AsTool run options passthrough tests** to `chatagent/astool_test.go`
