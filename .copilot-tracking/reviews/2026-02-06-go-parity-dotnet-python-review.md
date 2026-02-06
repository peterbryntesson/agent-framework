<!-- markdownlint-disable-file -->
# Implementation Review: Go Parity with .NET and Python (Epic 5)

**Review Date**: 2026-02-06
**Related Plan**: 2026-02-04-go-epic5-implementation-plan.md
**Related Changes**: 2026-02-05-go-epic5-phase4-changes.md; 2026-02-05-go-epic5-phase5-changes.md; 2026-02-05-go-epic5-phase6-changes.md; 2026-02-05-golang-lint-remediation-changes.md
**Related Research**: 2026-02-04-go-epic5-enterprise-production-research.md

## Review Summary

Reviewed Go Epic 5 parity against .NET and Python for durable agents, declarative agents, hosting and protocols, MCP integration, DevUI, Purview, and resilience. Major parity gaps remain in durable session identity and run options, MCP transports and prompt support, declarative tool mapping and PowerFx, and OpenAI hosting endpoint coverage. Overall status is Needs Rework.

## Implementation Checklist

Items extracted from research and plan documents with validation status.

### From Research Document

* [ ] Durable state schema compatibility and workflow-per-session pattern
	* Source: 2026-02-04-go-epic5-enterprise-production-research.md (Lines 109-126)
	* Status: Partial
	* Evidence: [go/durable/state.go](go/durable/state.go#L10-L67), [go/durable/workflow.go](go/durable/workflow.go#L38-L150)
* [ ] Declarative YAML requirements and PowerFx evaluation parity
	* Source: 2026-02-04-go-epic5-enterprise-production-research.md (Lines 181-215)
	* Status: Partial
	* Evidence: [go/declarative/validation.go](go/declarative/validation.go#L20-L66), [go/declarative/eval.go](go/declarative/eval.go#L8-L61)
* [ ] Tool kind mapping for function, mcp, webSearch, fileSearch, codeInterpreter
	* Source: 2026-02-04-go-epic5-enterprise-production-research.md (Lines 218-226)
	* Status: Missing
	* Evidence: [go/declarative/factory.go](go/declarative/factory.go#L32-L60), [go/declarative/models.go](go/declarative/models.go#L70-L116)
* [ ] OpenAI hosting coverage and AG-UI request shape parity
	* Source: 2026-02-04-go-epic5-enterprise-production-research.md (Lines 428-470)
	* Status: Partial
	* Evidence: [go/hosting/openai/handler.go](go/hosting/openai/handler.go#L44-L83), [go/protocol/agui/server.go](go/protocol/agui/server.go#L48-L76)
* [ ] MCP integration with transports, tool discovery, and agent-as-server
	* Source: 2026-02-04-go-epic5-enterprise-production-research.md (Lines 298-362)
	* Status: Partial
	* Evidence: [go/mcp/doc.go](go/mcp/doc.go#L12-L70), [go/mcp/server.go](go/mcp/server.go#L40-L118), [go/mcp/types.go](go/mcp/types.go#L82-L180)
* [ ] DevUI server endpoints and frontend support
	* Source: 2026-02-04-go-epic5-enterprise-production-research.md (Lines 377-394)
	* Status: Partial
	* Evidence: [go/devui/server.go](go/devui/server.go#L50-L132), [go/devui/server.go](go/devui/server.go#L124-L180)
* [ ] Purview client with scope caching and policy evaluation
	* Source: 2026-02-04-go-epic5-enterprise-production-research.md (Lines 396-414)
	* Status: Verified
	* Evidence: [go/purview/client.go](go/purview/client.go#L33-L156)
* [ ] Production hardening middleware for retry, circuit breaker, and rate limiting
	* Source: 2026-02-04-go-epic5-enterprise-production-research.md (Lines 481-559)
	* Status: Verified
	* Evidence: [go/resilience/doc.go](go/resilience/doc.go#L1-L90), [go/resilience/retry.go](go/resilience/retry.go#L12-L140), [go/resilience/ratelimit.go](go/resilience/ratelimit.go#L12-L140), [go/resilience/circuitbreaker.go](go/resilience/circuitbreaker.go#L18-L170)

### From Implementation Plan

* [ ] Feature 5.7 production hardening package
	* Source: 2026-02-04-go-epic5-implementation-plan.md Phase 1
	* Status: Verified
	* Evidence: [go/resilience/doc.go](go/resilience/doc.go#L1-L90)
* [ ] Feature 5.3 OpenAI hosting endpoints
	* Source: 2026-02-04-go-epic5-implementation-plan.md Phase 2
	* Status: Partial
	* Evidence: [go/hosting/openai/handler.go](go/hosting/openai/handler.go#L44-L83)
* [ ] Feature 5.2 declarative agents factory and validation
	* Source: 2026-02-04-go-epic5-implementation-plan.md Phase 3
	* Status: Partial
	* Evidence: [go/declarative/factory.go](go/declarative/factory.go#L18-L124), [go/declarative/validation.go](go/declarative/validation.go#L20-L66)
* [ ] Feature 5.4 MCP integration
	* Source: 2026-02-04-go-epic5-implementation-plan.md Phase 4
	* Status: Partial
	* Evidence: [go/mcp/doc.go](go/mcp/doc.go#L12-L70), [go/mcp/server.go](go/mcp/server.go#L40-L118)
* [ ] Feature 5.1 durable agents
	* Source: 2026-02-04-go-epic5-implementation-plan.md Phase 5
	* Status: Partial
	* Evidence: [go/durable/agent.go](go/durable/agent.go#L24-L180), [go/durable/workflow.go](go/durable/workflow.go#L38-L150)
* [ ] Feature 5.5 DevUI and Purview
	* Source: 2026-02-04-go-epic5-implementation-plan.md Phase 6
	* Status: Partial
	* Evidence: [go/devui/server.go](go/devui/server.go#L50-L180), [go/purview/client.go](go/purview/client.go#L33-L156)

## Validation Results

### Convention Compliance

* Manual Go idiom review only, no automated lint or format checks

### Validation Commands

* Not run

## Additional or Deviating Changes

* MCP HTTP transport example in documentation uses a URL ending in /sse, while the transport appends /sse internally. Evidence: [go/mcp/doc.go](go/mcp/doc.go#L56-L62), [go/mcp/transport_http.go](go/mcp/transport_http.go#L20-L65)

## Findings

### Durable agents parity gaps

* Session ID format does not match .NET and Python, which use @name@key. Go serializes using a dafx-name-key workflow format, breaking cross-language session reuse. Evidence: [go/durable/sessionid.go](go/durable/sessionid.go#L10-L90), [dotnet/src/Microsoft.Agents.AI.DurableTask/AgentSessionId.cs](dotnet/src/Microsoft.Agents.AI.DurableTask/AgentSessionId.cs#L70-L120), [python/packages/durabletask/agent_framework_durabletask/_models.py](python/packages/durabletask/agent_framework_durabletask/_models.py#L200-L270)
* Session serialization omits session ID, while .NET and Python include it as part of serialized session state. Evidence: [go/durable/session.go](go/durable/session.go#L76-L118), [go/durable/agent.go](go/durable/agent.go#L150-L198), [dotnet/src/Microsoft.Agents.AI.DurableTask/DurableAgentSession.cs](dotnet/src/Microsoft.Agents.AI.DurableTask/DurableAgentSession.cs#L18-L64), [python/packages/durabletask/agent_framework_durabletask/_models.py](python/packages/durabletask/agent_framework_durabletask/_models.py#L250-L330)
* GetSession queries a workflow but the workflow does not register a history query, so the call cannot succeed as implemented. Evidence: [go/durable/agent.go](go/durable/agent.go#L190-L230), [go/durable/workflow.go](go/durable/workflow.go#L38-L140)
* Run request options lack tool enablement, response format, and fire-and-forget support present in .NET and Python. Evidence: [go/durable/workflow.go](go/durable/workflow.go#L28-L72), [dotnet/src/Microsoft.Agents.AI.DurableTask/RunRequest.cs](dotnet/src/Microsoft.Agents.AI.DurableTask/RunRequest.cs#L12-L70), [dotnet/src/Microsoft.Agents.AI.DurableTask/DurableAgentRunOptions.cs](dotnet/src/Microsoft.Agents.AI.DurableTask/DurableAgentRunOptions.cs#L10-L52), [python/packages/durabletask/agent_framework_durabletask/_models.py](python/packages/durabletask/agent_framework_durabletask/_models.py#L130-L210)
* Content conversion drops data, uri, error, usage, and hosted content, so conversation replay loses information compared to .NET and Python. Evidence: [go/durable/state_entry.go](go/durable/state_entry.go#L180-L320), [dotnet/src/Microsoft.Agents.AI.DurableTask/State/DurableAgentStateContent.cs](dotnet/src/Microsoft.Agents.AI.DurableTask/State/DurableAgentStateContent.cs#L10-L60), [python/packages/durabletask/agent_framework_durabletask/_durable_agent_state.py](python/packages/durabletask/agent_framework_durabletask/_durable_agent_state.py#L200-L300)
* Function call arguments are persisted as strings, while .NET and Python keep structured values. Evidence: [go/durable/state_entry.go](go/durable/state_entry.go#L210-L270), [dotnet/src/Microsoft.Agents.AI.DurableTask/State/DurableAgentStateContent.cs](dotnet/src/Microsoft.Agents.AI.DurableTask/State/DurableAgentStateContent.cs#L10-L60), [python/packages/durabletask/agent_framework_durabletask/_durable_agent_state.py](python/packages/durabletask/agent_framework_durabletask/_durable_agent_state.py#L560-L650)
* Error responses are not flagged and filtered from history, unlike Python which marks error responses and excludes them from context building. Evidence: [python/packages/durabletask/agent_framework_durabletask/_entities.py](python/packages/durabletask/agent_framework_durabletask/_entities.py#L120-L200)

### MCP parity and transport issues

* WebSocket MCP transport is not implemented in Go, while Python exposes MCPWebsocketTool. Evidence: [go/mcp/doc.go](go/mcp/doc.go#L12-L70), [python/packages/core/agent_framework/_mcp.py](python/packages/core/agent_framework/_mcp.py#L1120-L1205)
* Prompts, logging, and sampling capabilities are declared but not handled by the Go server and client. Evidence: [go/mcp/types.go](go/mcp/types.go#L120-L176), [go/mcp/server.go](go/mcp/server.go#L60-L118), [go/mcp/client.go](go/mcp/client.go#L120-L220), [python/packages/core/agent_framework/_mcp.py](python/packages/core/agent_framework/_mcp.py#L360-L520)
* HTTP transport always appends /sse for notifications, but the Go server only handles POST requests and never exposes an SSE endpoint. Evidence: [go/mcp/transport_http.go](go/mcp/transport_http.go#L70-L160), [go/mcp/server.go](go/mcp/server.go#L300-L360)
* Stdio transport reads stdout in Send and Receive concurrently without demultiplexing, which risks consuming responses as notifications. Evidence: [go/mcp/transport_stdio.go](go/mcp/transport_stdio.go#L90-L220)
* Hosted MCP tool support is available in Python for service-managed tools, but Go has no equivalent. Evidence: [python/packages/core/agent_framework/_tools.py](python/packages/core/agent_framework/_tools.py#L360-L480)

### Declarative parity gaps

* Validation only checks kind, name, instructions, and tool name or kind, leaving model fields and schema requirements unchecked. Evidence: [go/declarative/validation.go](go/declarative/validation.go#L20-L66)
* Tool parsing only supports function tools even though the schema lists MCP, web search, code interpreter, and file search. Evidence: [go/declarative/factory.go](go/declarative/factory.go#L32-L112), [go/declarative/models.go](go/declarative/models.go#L70-L116)
* Tool bindings are defined but never applied to attach handlers during Create. Evidence: [go/declarative/models.go](go/declarative/models.go#L90-L114), [go/declarative/tools.go](go/declarative/tools.go#L18-L90), [go/declarative/factory.go](go/declarative/factory.go#L70-L124)
* PowerFx evaluation is limited to =Env.VAR, while .NET and Python evaluate broader expressions with controlled environment access. Evidence: [go/declarative/eval.go](go/declarative/eval.go#L8-L61), [dotnet/src/Microsoft.Agents.AI.Declarative/PromptAgentFactory.cs](dotnet/src/Microsoft.Agents.AI.Declarative/PromptAgentFactory.cs#L18-L52), [python/packages/declarative/agent_framework_declarative/_models.py](python/packages/declarative/agent_framework_declarative/_models.py#L14-L80)
* Provider mapping is limited to OpenAI and AzureOpenAI, while Python supports additional providers and API types. Evidence: [go/declarative/factory.go](go/declarative/factory.go#L32-L60), [python/packages/declarative/agent_framework_declarative/_loader.py](python/packages/declarative/agent_framework_declarative/_loader.py#L22-L120)

### Hosting and protocol parity gaps

* AG-UI server RunRequest only accepts threadId and messages, missing state, tools, context, forwarded props, and run IDs present in .NET and Python. Evidence: [go/protocol/agui/server.go](go/protocol/agui/server.go#L48-L76), [dotnet/src/Microsoft.Agents.AI.Hosting.AGUI.AspNetCore/AGUIEndpointRouteBuilderExtensions.cs](dotnet/src/Microsoft.Agents.AI.Hosting.AGUI.AspNetCore/AGUIEndpointRouteBuilderExtensions.cs#L32-L90), [python/packages/ag-ui/agent_framework_ag_ui/_types.py](python/packages/ag-ui/agent_framework_ag_ui/_types.py#L30-L100)
* OpenAI hosting only supports chat completions and models, while .NET exposes Responses and Conversations APIs. Evidence: [go/hosting/openai/handler.go](go/hosting/openai/handler.go#L44-L83), [dotnet/src/Microsoft.Agents.AI.Hosting.OpenAI/EndpointRouteBuilderExtensions.Responses.cs](dotnet/src/Microsoft.Agents.AI.Hosting.OpenAI/EndpointRouteBuilderExtensions.Responses.cs#L22-L120), [dotnet/src/Microsoft.Agents.AI.Hosting.OpenAI/EndpointRouteBuilderExtensions.Conversations.cs](dotnet/src/Microsoft.Agents.AI.Hosting.OpenAI/EndpointRouteBuilderExtensions.Conversations.cs#L12-L80)
* Session persistence uses X-Conversation-ID for load but saves using session.ID, preventing restoring by the original conversation ID. Evidence: [go/hosting/openai/completions.go](go/hosting/openai/completions.go#L65-L112)
* AG-UI client WithHTTPClient is a no-op, so custom HTTP clients cannot be injected. Evidence: [go/protocol/agui/client.go](go/protocol/agui/client.go#L42-L68)
* HostedAgentBuilder.Build panics instead of returning an error, which is non-idiomatic for a builder API. Evidence: [go/hosting/builder.go](go/hosting/builder.go#L40-L72)

### DevUI and Purview parity gaps

* Go DevUI serves a fallback HTML page unless a frontend FS is injected, while .NET embeds full frontend assets. Evidence: [go/devui/server.go](go/devui/server.go#L120-L200), [dotnet/src/Microsoft.Agents.AI.DevUI/DevUIMiddleware.cs](dotnet/src/Microsoft.Agents.AI.DevUI/DevUIMiddleware.cs#L20-L120)
* Go Purview uses a local TokenCredential interface instead of an Azure SDK credential type, which may complicate parity with Azure SDK integration patterns. Evidence: [go/purview/options.go](go/purview/options.go#L8-L40)

## Missing Work

* Align durable session ID format, serialization, and query behavior with .NET and Python
* Add MCP WebSocket transport, prompt APIs, and logging or sampling support
* Expand declarative tool parsing and provider mapping to match Python
* Add OpenAI Responses and Conversations endpoints to Go hosting
* Expand AG-UI request model to accept state, tools, context, forwarded props, and run IDs

## Follow-Up Work

### Deferred from Current Scope

* Decide whether to keep the custom Go TokenCredential interface or align with Azure SDK credential interfaces

### Identified During Review

* Fix OpenAI session persistence to save and load by the same conversation ID
* Implement a single-reader demultiplexer for MCP stdio transport
* Correct MCP HTTP documentation example to avoid /sse duplication
* Replace HostedAgentBuilder panic with error return or explicit validation API

## Review Completion

**Overall Status**: Needs Rework
**Reviewer Notes**: Significant parity gaps remain across durable agents, MCP, declarative tooling, and hosting endpoints. Use the Missing Work list to plan the next iteration.
