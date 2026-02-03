<!-- markdownlint-disable-file -->
# Release Changes: Go Middleware Patterns for Agent Framework

**Related Plan**: 2026-02-03-go-middleware-patterns-plan.instructions.md
**Implementation Date**: 2026-02-03

## Summary

Implemented middleware patterns for the Go agent framework including AgentMiddleware, FunctionMiddleware, DelegatingAgent, Builder.Use() method, and AsTool() conversion to enable flexible agent composition and interception. This enables cross-cutting concerns like logging, validation, and security to be added to agents declaratively.

## Changes

### Added

* `go/agent/middleware_context.go` - AgentContext and FunctionContext structs for middleware invocations
* `go/agent/middleware.go` - AgentMiddleware and FunctionMiddleware interfaces with function adapters
* `go/agent/delegating.go` - DelegatingAgent base type for decorator pattern
* `go/agent/delegating_test.go` - Unit tests for DelegatingAgent
* `go/agent/chain.go` - ChainAgentMiddleware and ChainFunctionMiddleware composition utilities
* `go/agent/chain_test.go` - Unit tests for middleware chaining
* `go/agent/middleware_agent.go` - MiddlewareAgent decorator that applies middleware to Run/RunStream
* `go/agent/middleware_agent_test.go` - Unit tests for MiddlewareAgent
* `go/agent/builder.go` - AgentBuilder with Use() and UseMiddleware() methods for pipeline composition
* `go/agent/builder_test.go` - Unit tests for AgentBuilder
* `go/chatagent/astool.go` - AsTool() function for agent-to-tool conversion enabling hierarchical agents
* `go/chatagent/astool_test.go` - Unit tests for AsTool with streaming scenarios

### Modified

* `go/agent/options.go` - Added WithRunConfig() option for middleware configuration passthrough
* `go/chatagent/options.go` - Added functionMiddleware field and WithFunctionMiddleware() option
* `go/chatagent/agent.go` - Added functionMiddleware field to Agent struct and integration in New()
* `go/chatagent/builder.go` - Added AgentFactory type, Use(), UseMiddleware(), UseFunctionMiddleware(), BuildAgent(), and MustBuildAgent() methods
* `go/chatagent/builder_test.go` - Added tests for Use(), UseMiddleware(), and BuildAgent() methods
* `go/chatagent/toolloop.go` - Added invokeSingleToolCall() method with FunctionMiddleware integration; refactored invokeToolCalls() and invokeToolCallsParallel() to use middleware
* `go/README.md` - Added Middleware and Hierarchical Agents documentation sections

### Removed

None

## Additional or Deviating Changes

* FunctionMiddleware integration was implemented in the chatagent package rather than the tool package to avoid circular imports. The tool.InvocationConfig was not modified.
  * Reason: The agent package contains FunctionMiddleware and FunctionContext, while tool package should remain independent. Middleware is applied at the chatagent layer where tool invocations occur.

* Race detector tests were skipped due to CGO requirement on Windows.
  * Reason: go test -race requires cgo which is not available without a C compiler configured.

## Release Summary

Total files affected: 17

Files created (12):
* `go/agent/middleware_context.go` - Middleware context types
* `go/agent/middleware.go` - Middleware interfaces
* `go/agent/delegating.go` - DelegatingAgent base type
* `go/agent/delegating_test.go` - DelegatingAgent tests
* `go/agent/chain.go` - Middleware chain utilities
* `go/agent/chain_test.go` - Chain tests
* `go/agent/middleware_agent.go` - MiddlewareAgent decorator
* `go/agent/middleware_agent_test.go` - MiddlewareAgent tests
* `go/agent/builder.go` - AgentBuilder pipeline builder
* `go/agent/builder_test.go` - AgentBuilder tests
* `go/chatagent/astool.go` - Agent-to-tool conversion
* `go/chatagent/astool_test.go` - AsTool tests

Files modified (5):
* `go/agent/options.go` - Added WithRunConfig option
* `go/chatagent/options.go` - Added FunctionMiddleware support
* `go/chatagent/agent.go` - Added middleware field
* `go/chatagent/builder.go` - Added Use/BuildAgent methods
* `go/chatagent/toolloop.go` - Integrated FunctionMiddleware
* `go/chatagent/builder_test.go` - Added new builder tests
* `go/README.md` - Added middleware documentation

Validation results:
* `go build ./...` - Passed
* `go vet ./...` - Passed
* `go test ./...` - All tests passed

