# AsTool Comparison: Python vs Go Implementation

**Date:** 2026-02-03
**Status:** Complete
**Researcher:** GitHub Copilot

## Executive Summary

This document compares the `as_tool()` method implementation between Python (`python/packages/core/agent_framework/_agents.py`) and Go (`go/chatagent/astool.go`) to identify feature gaps and alignment opportunities.

---

## 1. Feature Comparison Matrix

| Feature | Python | Go | Gap? |
|---------|--------|-----|------|
| Basic name/description override | ✅ | ✅ | No |
| Custom argument name | ✅ (`arg_name`) | ✅ (`ArgName`) | No |
| Custom argument description | ✅ (`arg_description`) | ✅ (`ArgDescription`) | No |
| Streaming callback support | ✅ | ✅ | No |
| Async stream callback | ✅ | N/A (Go uses channels) | No |
| Name sanitization | ✅ | ✅ | **Partial** |
| Runtime kwargs propagation | ✅ | ❌ | **Yes** |
| Approval mode setting | ✅ (`never_require`) | ❌ | **Yes** |
| Forward runtime kwargs flag | ✅ (`_forward_runtime_kwargs`) | ❌ | **Yes** |
| Session/thread management | ✅ (via middleware) | ❌ | **Yes** |
| Context provider integration | ✅ | ❌ | **Yes** |
| conversation_id exclusion | ✅ | ❌ | **Yes** |

---

## 2. Features in Python Missing in Go

### 2.1 Runtime kwargs Propagation (Critical Gap)

**Python Implementation:**
```python
# In BaseAgent.as_tool()
async def agent_wrapper(**kwargs: Any) -> str:
    # Forward runtime context kwargs, excluding arg_name and conversation_id
    forwarded_kwargs = {k: v for k, v in kwargs.items() if k not in (arg_name, "conversation_id")}
    return (await self.run(input_text, **forwarded_kwargs)).text

agent_tool._forward_runtime_kwargs = True  # Flag for tool invocation
```

**Go Implementation:**
- No equivalent mechanism exists
- `AsTool` creates a simple wrapper that only passes the task message
- No support for propagating runtime context (API tokens, session IDs, etc.)

**Impact:** Hierarchical agent patterns cannot pass session-specific data through delegation chains.

### 2.2 Approval Mode Configuration

**Python:**
```python
FunctionTool(
    ...
    approval_mode="never_require",  # Explicit setting
)
```

**Go:**
- `agentTool` struct doesn't include `ApprovalMode` field
- No `AsToolOptions` field for approval mode configuration
- Tools created via `AsTool` have no approval control

### 2.3 Name Sanitization Differences

**Python Handles:**
- Digit prefix: `"123Agent"` → `"_123Agent"` (prefixed with underscore)
- Empty result: `"@@@"` → `"agent"` (fallback)
- Case preservation: Maintains original case

**Go Handles:**
- Empty result: `""` → `"agent"` (fallback)
- Forces lowercase: All names converted to lowercase
- **Missing:** Digit prefix handling (e.g., `"123Agent"` → `"123_agent"` instead of `"_123agent"`)

### 2.4 Session/Thread State Management

**Python:**
- `as_tool()` wraps work with middleware support
- Middleware can capture/inject session context
- `conversation_id` explicitly excluded from forwarded kwargs

**Go:**
- No middleware integration in tool wrapper
- No session context management
- Agent sessions exist (`NewSession`, `RestoreSession`) but not used in `AsTool`

### 2.5 Context Provider Integration

**Python:**
- Tools can trigger context provider lifecycle hooks
- `invoked()` method called after tool execution

**Go:**
- Context providers exist (`agent.ContextProvider`)
- Not integrated with `AsTool` wrapper

---

## 3. Test Coverage Differences

### Python Test Coverage

| Test Case | File | Lines |
|-----------|------|-------|
| Basic as_tool functionality | `test_agents.py` | 370-380 |
| Custom parameters | `test_agents.py` | 382-397 |
| Default parameters | `test_agents.py` | 400-412 |
| No name error handling | `test_agents.py` | 415-421 |
| Function execution | `test_agents.py` | 424-434 |
| Stream callback | `test_agents.py` | 437-458 |
| Async stream callback | `test_agents.py` | 476-492 |
| Name sanitization | `test_agents.py` | 495-507 |
| **Kwargs propagation** | `test_as_tool_kwargs_propagation.py` | 1-357 |
| Forward runtime kwargs | `test_as_tool_kwargs_propagation.py` | 17-56 |
| Excludes arg_name | `test_as_tool_kwargs_propagation.py` | 58-92 |
| Nested delegation | `test_as_tool_kwargs_propagation.py` | 97-158 |
| Streaming mode kwargs | `test_as_tool_kwargs_propagation.py` | 160-202 |
| Empty kwargs handling | `test_as_tool_kwargs_propagation.py` | 204-220 |
| Kwargs with chat_options | `test_as_tool_kwargs_propagation.py` | 222-260 |
| Kwargs isolation per call | `test_as_tool_kwargs_propagation.py` | 262-316 |
| Excludes conversation_id | `test_as_tool_kwargs_propagation.py` | 318-357 |

**Total Python Tests:** ~16 dedicated test cases

### Go Test Coverage

| Test Case | File | Lines |
|-----------|------|-------|
| Non-streaming invocation | `astool_test.go` | 60-84 |
| Custom options | `astool_test.go` | 86-117 |
| Streaming mode | `astool_test.go` | 119-147 |
| Name sanitization | `astool_test.go` | 149-173 |
| Default description | `astool_test.go` | 175-187 |
| sanitizeAgentName | `astool_test.go` | 189-221 |

**Total Go Tests:** 6 test cases

### Missing Go Test Coverage

1. **Kwargs/context propagation tests** - No equivalent tests
2. **Error handling edge cases** - Limited error path testing
3. **Nested delegation patterns** - Not tested
4. **Isolation between calls** - Not tested
5. **Approval mode tests** - Not applicable (feature missing)

---

## 4. Error Handling Approaches

### Python

```python
# In as_tool wrapper
async def agent_wrapper(**kwargs: Any) -> str:
    # No try/catch - errors propagate directly
    return (await self.run(input_text, **forwarded_kwargs)).text
```

- Errors propagate naturally through async/await
- `FunctionTool.invoke()` has detailed error handling with observability
- Tool invocation failures captured with OpenTelemetry

### Go

```go
// In agentTool.Invoke
func (t *agentTool) Invoke(ctx context.Context, arguments json.RawMessage) (tool.Result, error) {
    result, err := t.invoke(ctx, arguments)
    if err != nil {
        return tool.Result{
            Content: err.Error(),
            IsError: true,
        }, err
    }
    return tool.NewResult(result), nil
}
```

- Error wrapped in `tool.Result` with `IsError: true`
- Error message exposed in `Content` field
- Both error return value and result structure indicate failure

**Difference:** Go returns error in two places (struct + return value); Python relies on exception propagation.

---

## 5. Session/State Management Patterns

### Python Pattern

```python
# Via middleware in as_tool context
@agent_middleware
async def capture_middleware(context: AgentRunContext, next) -> None:
    # Access context.kwargs for session state
    session_id = context.kwargs.get("session_id")
    await next(context)
```

**Flow:**
1. Parent agent calls sub-agent tool
2. `_forward_runtime_kwargs = True` triggers kwargs forwarding
3. Sub-agent's middleware receives forwarded kwargs
4. Session context preserved across delegation chain

### Go Pattern (Current)

```go
// No context forwarding - only message passed
func fn(ctx context.Context, args json.RawMessage) (string, error) {
    input := parsed[argName]
    messages := []agent.Message{agent.NewUserMessage(input)}
    resp, err := a.Run(ctx, messages)  // No RunOptions for context
    return resp.Text(), nil
}
```

**Missing:**
- No `RunOption` passed to sub-agent's `Run()`
- No way to forward session context
- `context.Context` available but not used for custom context propagation

### Recommended Go Enhancement

```go
// Proposed addition to AsToolOptions
type AsToolOptions struct {
    // ... existing fields ...
    
    // ForwardMetadata enables metadata propagation to sub-agent runs.
    ForwardMetadata bool
    
    // Metadata provides static metadata to pass to sub-agent runs.
    Metadata map[string]interface{}
}

// In AsTool wrapper
fn := func(ctx context.Context, args json.RawMessage) (string, error) {
    // Extract metadata from context if ForwardMetadata enabled
    var runOpts []agent.RunOption
    if opts.ForwardMetadata {
        if md := agent.MetadataFromContext(ctx); md != nil {
            runOpts = append(runOpts, agent.WithMetadata(md))
        }
    }
    if opts.Metadata != nil {
        runOpts = append(runOpts, agent.WithMetadata(opts.Metadata))
    }
    
    resp, err := a.Run(ctx, messages, runOpts...)
    return resp.Text(), nil
}
```

---

## 6. Recommendations

### Priority 1: Runtime Context Propagation (High Impact)

1. Add `ForwardMetadata bool` to `AsToolOptions`
2. Add `Metadata map[string]interface{}` to `AsToolOptions`
3. Use Go `context.Context` for propagating runtime values
4. Consider adding `agent.MetadataFromContext(ctx)` helper

### Priority 2: Approval Mode Support (Medium Impact)

1. Add `ApprovalMode` field to `AsToolOptions`
2. Add `ApprovalMode() ApprovalMode` method to `agentTool`
3. Default to `ApprovalNever` matching Python behavior

### Priority 3: Name Sanitization Alignment (Low Impact)

1. Add digit prefix handling to `sanitizeAgentName`
2. Consider case preservation option

### Priority 4: Test Coverage Enhancement (Medium Impact)

1. Add context/metadata propagation tests
2. Add nested delegation tests
3. Add error handling edge case tests
4. Add isolation between calls tests

---

## 7. Clarifying Questions

1. **Context propagation mechanism:** Should Go use `context.Context` values or `RunOption` metadata for propagating runtime context through delegation chains?

2. **Case sensitivity:** Should Go preserve case like Python, or is lowercase normalization intentional for Go conventions?

3. **Middleware integration:** Should the Go `AsTool` wrapper integrate with agent middleware, or is this out of scope?

4. **Approval mode:** Is approval mode support needed for Go `AsTool`, or is this a Python-only feature?

---

## 8. Files Analyzed

### Python
- [_agents.py](../../../python/packages/core/agent_framework/_agents.py) - `as_tool()` implementation (lines 411-503)
- [_tools.py](../../../python/packages/core/agent_framework/_tools.py) - FunctionTool and `_forward_runtime_kwargs` (lines 670-785)
- [test_agents.py](../../../python/packages/core/tests/core/test_agents.py) - as_tool tests (lines 370-510)
- [test_as_tool_kwargs_propagation.py](../../../python/packages/core/tests/core/test_as_tool_kwargs_propagation.py) - kwargs propagation tests (357 lines)
- [runtime_context_delegation.py](../../../python/samples/getting_started/middleware/runtime_context_delegation.py) - sample patterns

### Go
- [astool.go](../../../go/chatagent/astool.go) - AsTool implementation (185 lines)
- [astool_test.go](../../../go/chatagent/astool_test.go) - tests (221 lines)
- [options.go](../../../go/agent/options.go) - RunOption and RunConfig definitions
- [tool.go](../../../go/tool/tool.go) - ApprovalMode type definition
