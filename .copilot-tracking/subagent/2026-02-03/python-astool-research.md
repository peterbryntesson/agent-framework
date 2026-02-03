# Python `as_tool()` Implementation Research

**Date:** 2026-02-03  
**Status:** Complete  
**Output File:** `.copilot-tracking/subagent/2026-02-03/python-astool-research.md`

## Summary

The `as_tool()` method in the Python agent framework converts an agent into a `FunctionTool` that can be used by other agents. This enables hierarchical agent architectures where a coordinator agent can delegate tasks to specialized sub-agents.

## Complete Implementation

### Location

[python/packages/core/agent_framework/_agents.py#L411-L497](python/packages/core/agent_framework/_agents.py#L411-L497)

### Full Code

```python
def as_tool(
    self,
    *,
    name: str | None = None,
    description: str | None = None,
    arg_name: str = "task",
    arg_description: str | None = None,
    stream_callback: Callable[[AgentResponseUpdate], None]
    | Callable[[AgentResponseUpdate], Awaitable[None]]
    | None = None,
) -> FunctionTool[BaseModel, str]:
    """Create a FunctionTool that wraps this agent.

    Keyword Args:
        name: The name for the tool. If None, uses the agent's name.
        description: The description for the tool. If None, uses the agent's description or empty string.
        arg_name: The name of the function argument (default: "task").
        arg_description: The description for the function argument.
            If None, defaults to "Task for {tool_name}".
        stream_callback: Optional callback for streaming responses. If provided, uses run_stream.

    Returns:
        A FunctionTool that can be used as a tool by other agents.

    Raises:
        TypeError: If the agent does not implement AgentProtocol.
        ValueError: If the agent tool name cannot be determined.
    """
    # Verify that self implements AgentProtocol
    if not isinstance(self, AgentProtocol):
        raise TypeError(f"Agent {self.__class__.__name__} must implement AgentProtocol to be used as a tool")

    tool_name = name or _sanitize_agent_name(self.name)
    if tool_name is None:
        raise ValueError("Agent tool name cannot be None. Either provide a name parameter or set the agent's name.")
    tool_description = description or self.description or ""
    argument_description = arg_description or f"Task for {tool_name}"

    # Create dynamic input model with the specified argument name
    field_info = Field(..., description=argument_description)
    model_name = f"{name or _sanitize_agent_name(self.name) or 'agent'}_task"
    input_model = create_model(model_name, **{arg_name: (str, field_info)})  # type: ignore[call-overload]

    # Check if callback is async once, outside the wrapper
    is_async_callback = stream_callback is not None and inspect.iscoroutinefunction(stream_callback)

    async def agent_wrapper(**kwargs: Any) -> str:
        """Wrapper function that calls the agent."""
        # Extract the input from kwargs using the specified arg_name
        input_text = kwargs.get(arg_name, "")

        # Forward runtime context kwargs, excluding arg_name and conversation_id.
        forwarded_kwargs = {k: v for k, v in kwargs.items() if k not in (arg_name, "conversation_id")}

        if stream_callback is None:
            # Use non-streaming mode
            return (await self.run(input_text, **forwarded_kwargs)).text

        # Use streaming mode - accumulate updates and create final response
        response_updates: list[AgentResponseUpdate] = []
        async for update in self.run_stream(input_text, **forwarded_kwargs):
            response_updates.append(update)
            if is_async_callback:
                await stream_callback(update)  # type: ignore[misc]
            else:
                stream_callback(update)

        # Create final text from accumulated updates
        return AgentResponse.from_agent_run_response_updates(response_updates).text

    agent_tool: FunctionTool[BaseModel, str] = FunctionTool(
        name=tool_name,
        description=tool_description,
        func=agent_wrapper,
        input_model=input_model,  # type: ignore
        approval_mode="never_require",
    )
    agent_tool._forward_runtime_kwargs = True  # type: ignore
    return agent_tool
```

## Parameter Schema

The tool generates a **dynamic Pydantic model** as its input schema:

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `name` | `str \| None` | `None` | Tool name (defaults to sanitized agent name) |
| `description` | `str \| None` | `None` | Tool description (defaults to agent description) |
| `arg_name` | `str` | `"task"` | Name of the input argument in generated schema |
| `arg_description` | `str \| None` | `None` | Description for the input arg (defaults to "Task for {tool_name}") |
| `stream_callback` | `Callable \| None` | `None` | Optional callback for streaming responses |

### Generated Input Model Structure

```python
# Dynamic model created at runtime:
model_name = f"{tool_name}_task"
input_model = create_model(model_name, **{arg_name: (str, Field(..., description=arg_description))})

# Example JSON schema when arg_name="task":
{
    "properties": {
        "task": {
            "type": "string",
            "description": "Task for TestAgent"
        }
    },
    "required": ["task"]
}
```

## Agent Invocation Mechanism

When the tool is invoked by another agent:

1. **Input extraction:** The wrapper extracts the task text from kwargs using `arg_name`
2. **Runtime context forwarding:** All kwargs except `arg_name` and `conversation_id` are forwarded to the sub-agent
3. **Execution mode selection:**
   - **No stream_callback:** Calls `self.run(input_text, **forwarded_kwargs)` and returns `.text`
   - **With stream_callback:** Calls `self.run_stream()`, accumulates updates, invokes callback for each, returns final text

### Key Implementation Detail: `_forward_runtime_kwargs`

```python
agent_tool._forward_runtime_kwargs = True
```

This flag on `FunctionTool` enables automatic forwarding of runtime kwargs (like `api_token`, `user_id`, session context) through the tool invocation chain. This is critical for hierarchical agent patterns.

## Streaming Support

### Synchronous Callback

```python
def stream_callback(update: AgentResponseUpdate) -> None:
    print(update.text)

tool = agent.as_tool(stream_callback=stream_callback)
```

### Asynchronous Callback

```python
async def async_stream_callback(update: AgentResponseUpdate) -> None:
    await log_update(update)

tool = agent.as_tool(stream_callback=async_stream_callback)
```

The implementation automatically detects if the callback is async:

```python
is_async_callback = stream_callback is not None and inspect.iscoroutinefunction(stream_callback)
```

### Final Response Construction

Streaming updates are accumulated and combined using:

```python
return AgentResponse.from_agent_run_response_updates(response_updates).text
```

## Test Patterns Found

### Test File Locations

- [python/packages/core/tests/core/test_agents.py#L368-L498](python/packages/core/tests/core/test_agents.py#L368-L498)
- [python/packages/core/tests/core/test_as_tool_kwargs_propagation.py](python/packages/core/tests/core/test_as_tool_kwargs_propagation.py)

### Test Categories

#### 1. Basic Functionality Tests

```python
async def test_chat_agent_as_tool_basic(chat_client: ChatClientProtocol) -> None:
    """Test basic as_tool functionality."""
    agent = ChatAgent(chat_client=chat_client, name="TestAgent", description="Test agent for as_tool")

    tool = agent.as_tool()

    assert tool.name == "TestAgent"
    assert tool.description == "Test agent for as_tool"
    assert hasattr(tool, "func")
    assert hasattr(tool, "input_model")
```

#### 2. Custom Parameters Tests

```python
async def test_chat_agent_as_tool_custom_parameters(chat_client: ChatClientProtocol) -> None:
    """Test as_tool with custom parameters."""
    agent = ChatAgent(chat_client=chat_client, name="TestAgent", description="Original description")

    tool = agent.as_tool(
        name="CustomTool",
        description="Custom description",
        arg_name="query",
        arg_description="Custom input description",
    )

    assert tool.name == "CustomTool"
    assert tool.description == "Custom description"

    # Check that the input model has the custom field name
    schema = tool.input_model.model_json_schema()
    assert "query" in schema["properties"]
    assert schema["properties"]["query"]["description"] == "Custom input description"
```

#### 3. Function Execution Tests

```python
async def test_chat_agent_as_tool_function_execution(chat_client: ChatClientProtocol) -> None:
    """Test that the generated FunctionTool can be executed."""
    agent = ChatAgent(chat_client=chat_client, name="TestAgent", description="Test agent")

    tool = agent.as_tool()

    # Test function execution
    result = await tool.invoke(arguments=tool.input_model(task="Hello"))

    # Should return the agent's response text
    assert isinstance(result, str)
    assert result == "test response"  # From mock chat client
```

#### 4. Streaming Callback Tests

```python
async def test_chat_agent_as_tool_with_stream_callback(chat_client: ChatClientProtocol) -> None:
    """Test as_tool with stream callback functionality."""
    agent = ChatAgent(chat_client=chat_client, name="StreamingAgent")

    collected_updates: list[AgentResponseUpdate] = []

    def stream_callback(update: AgentResponseUpdate) -> None:
        collected_updates.append(update)

    tool = agent.as_tool(stream_callback=stream_callback)
    result = await tool.invoke(arguments=tool.input_model(task="Hello"))

    assert len(collected_updates) > 0
    assert isinstance(result, str)
    expected_text = "".join(update.text for update in collected_updates)
    assert result == expected_text
```

#### 5. Name Sanitization Tests

```python
async def test_chat_agent_as_tool_name_sanitization(chat_client: ChatClientProtocol) -> None:
    """Test as_tool name sanitization."""
    test_cases = [
        ("Invoice & Billing Agent", "Invoice_Billing_Agent"),
        ("Travel & Logistics Agent", "Travel_Logistics_Agent"),
        ("Agent@Company.com", "Agent_Company_com"),
        ("Agent___Multiple___Underscores", "Agent_Multiple_Underscores"),
        ("123Agent", "_123Agent"),  # Test digit prefix handling
        ("@@@", "agent"),  # Test empty sanitization fallback
    ]

    for agent_name, expected_tool_name in test_cases:
        agent = ChatAgent(chat_client=chat_client, name=agent_name)
        tool = agent.as_tool()
        assert tool.name == expected_tool_name
```

#### 6. Kwargs Propagation Tests

```python
async def test_as_tool_forwards_runtime_kwargs(self, chat_client: MockChatClient) -> None:
    """Test that runtime kwargs are forwarded through as_tool() to sub-agent."""
    captured_kwargs: dict[str, Any] = {}

    @agent_middleware
    async def capture_middleware(
        context: AgentRunContext, next: Callable[[AgentRunContext], Awaitable[None]]
    ) -> None:
        captured_kwargs.update(context.kwargs)
        await next(context)

    sub_agent = ChatAgent(chat_client=chat_client, name="sub_agent", middleware=[capture_middleware])
    tool = sub_agent.as_tool(name="delegate", arg_name="task")

    _ = await tool.invoke(
        arguments=tool.input_model(task="Test delegation"),
        api_token="secret-xyz-123",
        user_id="user-456",
    )

    assert captured_kwargs["api_token"] == "secret-xyz-123"
    assert captured_kwargs["user_id"] == "user-456"
```

#### 7. Nested Delegation Tests

```python
async def test_as_tool_nested_delegation_propagates_kwargs(self, chat_client: MockChatClient) -> None:
    """Test that kwargs propagate through multiple levels (A → B → C)."""
    # Creates agent C, agent B (with C as tool), invokes B's tool
    # Verifies kwargs like trace_id and tenant_id propagate to both levels
```

## Usage Example (From Samples)

From [python/samples/getting_started/middleware/runtime_context_delegation.py](python/samples/getting_started/middleware/runtime_context_delegation.py):

```python
# Create specialized sub-agents
email_agent = client.as_agent(
    name="email_agent",
    instructions="You send emails using the send_email tool.",
    tools=[send_email_tool],
)

sms_agent = client.as_agent(
    name="sms_agent",
    instructions="You send SMS messages using the send_sms tool.",
    tools=[send_sms_tool],
)

# Create coordinator that delegates to sub-agents
coordinator = client.as_agent(
    name="coordinator",
    instructions="You coordinate communication tasks.",
    tools=[
        email_agent.as_tool(
            name="email_sender",
            description="Send emails to recipients",
            arg_name="task",
        ),
        sms_agent.as_tool(
            name="sms_sender",
            description="Send SMS messages",
            arg_name="task",
        ),
    ],
)

# Runtime context automatically propagates through as_tool()
await coordinator.run(
    "Send an email to john@example.com",
    api_token="secret-token-abc",
    user_id="user-999",
)
```

## Key Findings

- **Dynamic schema generation:** Uses Pydantic's `create_model` to dynamically generate input models with configurable argument names
- **Runtime kwargs forwarding:** The `_forward_runtime_kwargs = True` flag enables automatic propagation of session context (api_token, user_id, etc.) through delegation chains
- **Dual execution modes:** Supports both non-streaming (`run()`) and streaming (`run_stream()`) with sync/async callback detection
- **Name sanitization:** Agent names are sanitized for use as function names (special chars → underscores, digit prefix → underscore prefix)
- **Protocol verification:** Validates that the agent implements `AgentProtocol` before creating the tool
- **Default approval mode:** Tools created via `as_tool()` use `approval_mode="never_require"` (sub-agent calls don't require separate approval)
- **Returns text only:** The tool wrapper returns `str` (the `.text` property of `AgentResponse`), not the full response object
- **Excluded kwargs:** The `arg_name` parameter and `conversation_id` are explicitly excluded from forwarded kwargs
