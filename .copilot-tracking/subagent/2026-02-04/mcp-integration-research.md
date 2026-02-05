# MCP (Model Context Protocol) Integration Analysis

**Date**: 2026-02-04  
**Research Focus**: MCP client/server implementations in .NET and Python

---

## Executive Summary

Both .NET and Python implementations provide comprehensive MCP (Model Context Protocol) support with:

1. **MCP Client** - Connect to external MCP servers to use their tools
2. **MCP Server** - Expose agents as MCP servers for other applications
3. **Hosted MCP Tools** - Service-managed MCP tool execution (OpenAI Responses API pattern)
4. **Tool Bridging** - Automatic conversion between MCP tools and framework Tool interfaces

---

## 1. Python MCP Implementation

### 1.1 Core MCP Client Classes

**File**: [python/packages/core/agent_framework/_mcp.py](python/packages/core/agent_framework/_mcp.py)

#### Transport Options (Lines 900-1245)

```python
# Three transport implementations:
class MCPStdioTool(MCPTool):      # stdio-based (local processes)
class MCPStreamableHTTPTool(MCPTool):  # HTTP/SSE-based
class MCPWebsocketTool(MCPTool):   # WebSocket-based
```

#### Base MCPTool Class (Lines 306-360)

```python
class MCPTool:
    def __init__(
        self,
        name: str,
        description: str | None = None,
        approval_mode: (Literal["always_require", "never_require"] | HostedMCPSpecificApproval | None) = None,
        allowed_tools: Collection[str] | None = None,
        load_tools: bool = True,
        parse_tool_results: Literal[True] | Callable[[types.CallToolResult], Any] | None = True,
        load_prompts: bool = True,
        parse_prompt_results: Literal[True] | Callable[[types.GetPromptResult], Any] | None = True,
        session: ClientSession | None = None,
        request_timeout: int | None = None,
        chat_client: "ChatClientProtocol | None" = None,
        additional_properties: dict[str, Any] | None = None,
    ) -> None:
```

### 1.2 MCP Client Initialization Pattern (Lines 389-445)

```python
async def connect(self, *, reset: bool = False) -> None:
    """Connect to the MCP server."""
    if reset:
        await self._safe_close_exit_stack()
        self.session = None
        self.is_connected = False
        self._exit_stack = AsyncExitStack()
    if not self.session:
        try:
            transport = await self._exit_stack.enter_async_context(self.get_mcp_client())
        except Exception as ex:
            # Handle connection errors
            ...
        try:
            session = await self._exit_stack.enter_async_context(
                ClientSession(
                    read_stream=transport[0],
                    write_stream=transport[1],
                    read_timeout_seconds=(
                        timedelta(seconds=self.request_timeout) if self.request_timeout else None
                    ),
                    message_handler=self.message_handler,
                    logging_callback=self.logging_callback,
                    sampling_callback=self.sampling_callback,
                )
            )
        except Exception as ex:
            # Handle session creation errors
            ...
        await session.initialize()
        self.session = session
```

### 1.3 Tool Discovery (Lines 649-689)

```python
async def load_tools(self) -> None:
    """Load tools from the MCP server."""
    existing_names = {func.name for func in self._functions}
    params: types.PaginatedRequestParams | None = None
    while True:
        await self._ensure_connected()
        tool_list = await self.session.list_tools(params=params)
        
        for tool in tool_list.tools:
            local_name = _normalize_mcp_name(tool.name)
            if local_name in existing_names:
                continue
            input_model = _get_input_model_from_mcp_tool(tool)
            approval_mode = self._determine_approval_mode(local_name)
            func: FunctionTool = FunctionTool(
                func=partial(self.call_tool, tool.name),
                name=local_name,
                description=tool.description or "",
                approval_mode=approval_mode,
                input_model=input_model,
            )
            self._functions.append(func)
            existing_names.add(local_name)
        
        if not tool_list or not tool_list.nextCursor:
            break
        params = types.PaginatedRequestParams(cursor=tool_list.nextCursor)
```

### 1.4 Tool Call Forwarding (Lines 720-770)

```python
async def call_tool(self, tool_name: str, **kwargs: Any) -> list[Content] | Any | types.CallToolResult:
    """Call a tool with the given arguments."""
    # Filter out framework kwargs that cannot be serialized
    filtered_kwargs = {
        k: v for k, v in kwargs.items()
        if k not in {"chat_options", "tools", "tool_choice", "thread", "conversation_id", "options"}
    }
    
    # Try the operation, reconnecting once if connection is closed
    for attempt in range(2):
        try:
            result = await self.session.call_tool(tool_name, arguments=filtered_kwargs)
            if self.parse_tool_results is None:
                return result
            if self.parse_tool_results is True:
                return _parse_contents_from_mcp_tool_result(result)
            if callable(self.parse_tool_results):
                return self.parse_tool_results(result)
            return result
        except ClosedResourceError as cl_ex:
            if attempt == 0:
                await self.connect(reset=True)
                continue
            raise ToolExecutionException(...)
```

### 1.5 MCP Server Pattern (Agent as MCP Server)

**File**: [python/packages/core/agent_framework/_agents.py](python/packages/core/agent_framework/_agents.py#L1105-L1218)

```python
def as_mcp_server(
    self,
    *,
    server_name: str = "Agent",
    version: str | None = None,
    instructions: str | None = None,
    lifespan: Callable[["Server[Any]"], AbstractAsyncContextManager[Any]] | None = None,
    **kwargs: Any,
) -> "Server[Any]":
    """Create an MCP server from an agent instance."""
    server: "Server[Any]" = Server(**server_args)
    agent_tool = self.as_tool(name=self._get_agent_name())
    
    @server.list_tools()
    async def _list_tools() -> list[types.Tool]:
        schema = agent_tool.input_model.model_json_schema()
        tool = types.Tool(
            name=agent_tool.name,
            description=agent_tool.description,
            inputSchema={
                "type": "object",
                "properties": schema.get("properties", {}),
                "required": schema.get("required", []),
            },
        )
        return [tool]
    
    @server.call_tool()
    async def _call_tool(name: str, arguments: dict[str, Any]) -> Sequence[types.TextContent | ...]:
        args_instance = agent_tool.input_model(**arguments)
        result = await agent_tool.invoke(arguments=args_instance)
        if isinstance(result, str):
            return [types.TextContent(type="text", text=result)]
        return [types.TextContent(type="text", text=str(result))]
    
    return server
```

### 1.6 HostedMCPTool (Service-Managed Execution)

**File**: [python/packages/core/agent_framework/_tools.py](python/packages/core/agent_framework/_tools.py#L379-L475)

```python
class HostedMCPTool(BaseTool):
    """Represents a MCP tool that is managed and executed by the service."""
    
    def __init__(
        self,
        *,
        name: str,
        description: str | None = None,
        url: AnyUrl | str,
        approval_mode: Literal["always_require", "never_require"] | HostedMCPSpecificApproval | None = None,
        allowed_tools: Collection[str] | None = None,
        headers: dict[str, str] | None = None,
        additional_properties: dict[str, Any] | None = None,
        **kwargs: Any,
    ) -> None:
        """Create a hosted MCP tool (executed by OpenAI/Azure service)."""
```

---

## 2. .NET MCP Implementation

### 2.1 MCP Client Integration (Using ModelContextProtocol Package)

**Package Reference**: [dotnet/Directory.Packages.props](dotnet/Directory.Packages.props#L102)

```xml
<PackageVersion Include="ModelContextProtocol" Version="0.4.0-preview.3" />
```

### 2.2 Client Initialization Pattern

**File**: [dotnet/samples/GettingStarted/FoundryAgents/FoundryAgents_Step09_UsingMcpClientAsTools/Program.cs](dotnet/samples/GettingStarted/FoundryAgents/FoundryAgents_Step09_UsingMcpClientAsTools/Program.cs#L16-L25)

```csharp
// Create an MCPClient for the GitHub server
await using var mcpClient = await McpClient.CreateAsync(new StdioClientTransport(new()
{
    Name = "MCPServer",
    Command = "npx",
    Arguments = ["-y", "--verbose", "@modelcontextprotocol/server-github"],
}));

// Retrieve the list of tools available on the GitHub server
IList<McpClientTool> mcpTools = await mcpClient.ListToolsAsync();
```

### 2.3 Tool Discovery and Bridging

**File**: [dotnet/samples/GettingStarted/ModelContextProtocol/Agent_MCP_Server/Program.cs](dotnet/samples/GettingStarted/ModelContextProtocol/Agent_MCP_Server/Program.cs#L26-L34)

```csharp
// MCP tools implement AITool interface, enabling direct use with agents
AIAgent agent = new AzureOpenAIClient(
    new Uri(endpoint),
    new AzureCliCredential())
    .GetChatClient(deploymentName)
    .AsAIAgent(
        instructions: "You answer questions related to GitHub repositories only.",
        tools: [.. mcpTools.Cast<AITool>()]  // McpClientTool implements AITool
    );
```

### 2.4 HostedMcpServerTool (Service-Managed Execution)

**File**: [dotnet/samples/GettingStarted/ModelContextProtocol/ResponseAgent_Hosted_MCP/Program.cs](dotnet/samples/GettingStarted/ModelContextProtocol/ResponseAgent_Hosted_MCP/Program.cs#L21-L27)

```csharp
// Create an MCP tool definition that the agent can use
// The service (OpenAI Responses API) invokes MCP tools directly
var mcpTool = new HostedMcpServerTool(
    serverName: "microsoft_learn",
    serverAddress: "https://learn.microsoft.com/api/mcp")
{
    AllowedTools = ["microsoft_docs_search"],
    ApprovalMode = HostedMcpServerToolApprovalMode.NeverRequire
};
```

### 2.5 MCP Tool Approval Handling

**File**: [dotnet/samples/GettingStarted/ModelContextProtocol/ResponseAgent_Hosted_MCP/Program.cs](dotnet/samples/GettingStarted/ModelContextProtocol/ResponseAgent_Hosted_MCP/Program.cs#L66-L84)

```csharp
// Handle approval requests for MCP tool calls
var userInputResponses = userInputRequests
    .OfType<McpServerToolApprovalRequestContent>()
    .Select(approvalRequest =>
    {
        Console.WriteLine($"""
            The agent would like to invoke the following MCP Tool, please reply Y to approve.
            ServerName: {approvalRequest.ToolCall.ServerName}
            Name: {approvalRequest.ToolCall.ToolName}
            Arguments: {string.Join(", ", approvalRequest.ToolCall.Arguments?.Select(x => $"{x.Key}: {x.Value}") ?? [])}
            """);
        return new ChatMessage(ChatRole.User, 
            [approvalRequest.CreateResponse(Console.ReadLine()?.Equals("Y", StringComparison.OrdinalIgnoreCase) ?? false)]);
    })
    .ToList();
```

### 2.6 Declarative MCP Tool Extensions

**File**: [dotnet/src/Microsoft.Agents.AI.Declarative/Extensions/McpServerToolExtensions.cs](dotnet/src/Microsoft.Agents.AI.Declarative/Extensions/McpServerToolExtensions.cs#L1-L35)

```csharp
internal static class McpServerToolExtensions
{
    internal static HostedMcpServerTool CreateHostedMcpTool(this McpServerTool tool)
    {
        Throw.IfNull(tool);
        Throw.IfNull(tool.ServerName?.LiteralValue);
        Throw.IfNull(tool.Connection);

        var connection = tool.Connection as AnonymousConnection 
            ?? throw new ArgumentException("Only AnonymousConnection is supported");
        var serverUrl = connection.Endpoint?.LiteralValue;
        
        return new HostedMcpServerTool(tool.ServerName.LiteralValue, serverUrl)
        {
            ServerDescription = tool.ServerDescription?.LiteralValue,
            AllowedTools = tool.AllowedTools?.LiteralValue,
            ApprovalMode = tool.ApprovalMode?.AsHostedMcpServerToolApprovalMode(),
        };
    }
}
```

### 2.7 OAuth Authentication for MCP Servers

**File**: [dotnet/samples/GettingStarted/ModelContextProtocol/Agent_MCP_Server_Auth/Program.cs](dotnet/samples/GettingStarted/ModelContextProtocol/Agent_MCP_Server_Auth/Program.cs#L28-L43)

```csharp
// Create SSE client transport for the MCP server with OAuth
var transport = new HttpClientTransport(new()
{
    Endpoint = new Uri(serverUrl),
    Name = "Secure Weather Client",
    OAuth = new()
    {
        ClientId = "ProtectedMcpClient",
        RedirectUri = new Uri("http://localhost:1179/callback"),
        AuthorizationRedirectDelegate = HandleAuthorizationUrlAsync,
    }
}, httpClient, consoleLoggerFactory);

await using var mcpClient = await McpClient.CreateAsync(transport, loggerFactory: consoleLoggerFactory);
```

---

## 3. Declarative Agent MCP Tool Definition

### 3.1 Python Declarative Model

**File**: [python/packages/declarative/agent_framework_declarative/_models.py](python/packages/declarative/agent_framework_declarative/_models.py#L730-L766)

```python
class McpTool(Tool):
    """Object representing an MCP tool."""
    
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

### 3.2 Approval Mode Classes

**File**: [python/packages/declarative/agent_framework_declarative/_models.py](python/packages/declarative/agent_framework_declarative/_models.py#L697-L725)

```python
class McpServerToolAlwaysRequireApprovalMode(McpServerApprovalMode):
    def __init__(self, kind: str = "always") -> None: ...

class McpServerToolNeverRequireApprovalMode(McpServerApprovalMode):
    def __init__(self, kind: str = "never") -> None: ...

class McpServerToolSpecifyApprovalMode(McpServerApprovalMode):
    def __init__(
        self,
        kind: str = "specify",
        alwaysRequireApprovalTools: list[str] | None = None,
        neverRequireApprovalTools: list[str] | None = None,
    ) -> None: ...
```

### 3.3 Declarative Loader MCP Tool Conversion

**File**: [python/packages/declarative/agent_framework_declarative/_loader.py](python/packages/declarative/agent_framework_declarative/_loader.py#L760-L820)

```python
case McpTool():
    approval_mode: HostedMCPSpecificApproval | Literal["always_require", "never_require"] | None = None
    if tool_resource.approvalMode is not None:
        if tool_resource.approvalMode.kind == "always":
            approval_mode = "always_require"
        elif tool_resource.approvalMode.kind == "never":
            approval_mode = "never_require"
        elif isinstance(tool_resource.approvalMode, McpServerToolSpecifyApprovalMode):
            approval_mode = {}
            if tool_resource.approvalMode.alwaysRequireApprovalTools:
                approval_mode["always_require_approval"] = (
                    tool_resource.approvalMode.alwaysRequireApprovalTools
                )
            ...
    
    return HostedMCPTool(
        name=tool_resource.name,
        description=tool_resource.description,
        url=tool_resource.url,
        allowed_tools=tool_resource.allowedTools,
        approval_mode=approval_mode,
        headers=headers,
        additional_properties=additional_properties,
    )
```

---

## 4. Tool Bridging Patterns

### 4.1 Python: MCP Tool → FunctionTool Conversion

**File**: [python/packages/core/agent_framework/_mcp.py](python/packages/core/agent_framework/_mcp.py#L287-L298)

```python
def _get_input_model_from_mcp_tool(tool: types.Tool) -> type[BaseModel]:
    """Creates a Pydantic model from a tool's parameters."""
    return _build_pydantic_model_from_json_schema(tool.name, tool.inputSchema)
```

The MCP tool's JSON schema is converted to a Pydantic model for the FunctionTool.

### 4.2 .NET: McpClientTool → AITool Interface

MCP tools from the `ModelContextProtocol` package implement `AITool` interface (from `Microsoft.Extensions.AI`), enabling seamless integration:

```csharp
// McpClientTool already implements AITool, so direct casting works
tools: [.. mcpTools.Cast<AITool>()]
```

### 4.3 Content Type Conversion

**File**: [python/packages/core/agent_framework/_mcp.py](python/packages/core/agent_framework/_mcp.py#L62-L187)

```python
def _parse_content_from_mcp(mcp_type: ...) -> list[Content]:
    """Parse an MCP type into an Agent Framework type."""
    match mcp_type:
        case types.TextContent():
            return [Content.from_text(text=mcp_type.text, raw_representation=mcp_type)]
        case types.ImageContent() | types.AudioContent():
            data_bytes = base64.b64decode(mcp_type.data)
            return [Content.from_data(data=data_bytes, media_type=mcp_type.mimeType, ...)]
        case types.ResourceLink():
            return [Content.from_uri(uri=str(mcp_type.uri), media_type=mcp_type.mimeType, ...)]
        case types.ToolUseContent():
            return [Content.from_function_call(call_id=mcp_type.id, name=mcp_type.name, ...)]
        case types.EmbeddedResource():
            # Handle TextResourceContents and BlobResourceContents
            ...
```

---

## 5. Transport Layer Implementations

### 5.1 Python Transport Options

| Transport | Class | Use Case |
|-----------|-------|----------|
| **Stdio** | `MCPStdioTool` | Local process communication |
| **HTTP/SSE** | `MCPStreamableHTTPTool` | Remote HTTP servers |
| **WebSocket** | `MCPWebsocketTool` | Real-time bidirectional |

### 5.2 .NET Transport Options

| Transport | Class | Use Case |
|-----------|-------|----------|
| **Stdio** | `StdioClientTransport` | Local process communication |
| **HTTP** | `HttpClientTransport` | Remote HTTP servers with OAuth |

---

## 6. Key Differences: Python vs .NET

| Aspect | Python | .NET |
|--------|--------|------|
| **MCP SDK** | Uses `mcp` Python package | Uses `ModelContextProtocol` NuGet |
| **Client Class** | `MCPTool` base class with subclasses | `McpClient.CreateAsync()` factory |
| **Tool Interface** | `FunctionTool` wrapping MCP tools | `McpClientTool` implements `AITool` |
| **Server Exposure** | `agent.as_mcp_server()` method | No direct equivalent (uses samples) |
| **Hosted Tools** | `HostedMCPTool` class | `HostedMcpServerTool` class |
| **Approval Handling** | `approval_mode` dict/literal | `HostedMcpServerToolApprovalMode` enum |

---

## 7. Sample Code References

### 7.1 Python Samples

| Sample | Path |
|--------|------|
| GitHub MCP with PAT | [python/samples/getting_started/mcp/mcp_github_pat.py](python/samples/getting_started/mcp/mcp_github_pat.py) |
| Agent as MCP Server | [python/samples/getting_started/mcp/agent_as_mcp_server.py](python/samples/getting_started/mcp/agent_as_mcp_server.py) |
| OpenAI with Local MCP | [python/samples/getting_started/agents/openai/openai_responses_client_with_local_mcp.py](python/samples/getting_started/agents/openai/openai_responses_client_with_local_mcp.py) |
| Azure AI with Hosted MCP | [python/samples/getting_started/agents/azure_ai/azure_ai_with_hosted_mcp.py](python/samples/getting_started/agents/azure_ai/azure_ai_with_hosted_mcp.py) |

### 7.2 .NET Samples

| Sample | Path |
|--------|------|
| Basic MCP Server | [dotnet/samples/GettingStarted/ModelContextProtocol/Agent_MCP_Server/](dotnet/samples/GettingStarted/ModelContextProtocol/Agent_MCP_Server/) |
| MCP with OAuth | [dotnet/samples/GettingStarted/ModelContextProtocol/Agent_MCP_Server_Auth/](dotnet/samples/GettingStarted/ModelContextProtocol/Agent_MCP_Server_Auth/) |
| Responses Agent + Hosted MCP | [dotnet/samples/GettingStarted/ModelContextProtocol/ResponseAgent_Hosted_MCP/](dotnet/samples/GettingStarted/ModelContextProtocol/ResponseAgent_Hosted_MCP/) |
| Foundry with MCP Client | [dotnet/samples/GettingStarted/FoundryAgents/FoundryAgents_Step09_UsingMcpClientAsTools/](dotnet/samples/GettingStarted/FoundryAgents/FoundryAgents_Step09_UsingMcpClientAsTools/) |

---

## 8. Test Coverage

### 8.1 Python Tests

**File**: [python/packages/core/tests/core/test_mcp.py](python/packages/core/tests/core/test_mcp.py) (2517 lines)

- Name normalization tests
- Content type conversion tests
- Tool result parsing with metadata
- Session management tests
- Error handling tests

---

## 9. Conclusions

### 9.1 Architecture Summary

1. **MCP Client Pattern**: Both platforms use a similar pattern - initialize transport, create session, list tools, call tools
2. **Tool Bridging**: MCP tools are automatically converted to framework-native tool types
3. **Hosted MCP**: Both support service-managed MCP execution (OpenAI Responses pattern)
4. **Agent as Server**: Python has `as_mcp_server()` method; .NET requires manual server setup

### 9.2 Key Implementation Files

| Component | Python | .NET |
|-----------|--------|------|
| **MCP Client** | `_mcp.py` | External `ModelContextProtocol` package |
| **Hosted Tools** | `_tools.py` | External `Microsoft.Extensions.AI` |
| **Declarative** | `_loader.py`, `_models.py` | `McpServerToolExtensions.cs` |
| **Server Exposure** | `_agents.py` | N/A (sample only) |
