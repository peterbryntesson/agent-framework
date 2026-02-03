<!-- markdownlint-disable-file -->
# Release Changes: Go ContextProvider Pattern for Dynamic Context Injection

**Related Plan**: 2026-02-03-go-context-provider-plan.instructions.md
**Implementation Date**: 2026-02-03

## Summary

Implements the ContextProvider pattern for the Go agent framework, enabling dynamic context injection (instructions, messages, tools) before agent invocations with lifecycle hooks for session tracking.

## Changes

### Added

* [go/agent/context_provider.go](go/agent/context_provider.go) - New file containing Context struct, ContextProvider interface, ContextProviderFunc adapter, ContextProviderWithLifecycle interface, BaseContextProvider, and AggregateContextProvider
* [go/agent/context_provider_test.go](go/agent/context_provider_test.go) - Unit tests for Context struct, ContextProviderFunc, BaseContextProvider, and AggregateContextProvider
* [go/chatagent/context_provider_test.go](go/chatagent/context_provider_test.go) - Integration tests for context provider injection in ChatClientAgent

### Modified

* [go/chatagent/options.go](go/chatagent/options.go) - Added contextProviders field to config struct and WithContextProvider option function
* [go/chatagent/agent.go](go/chatagent/agent.go) - Added contextProvider field to Agent struct, modified Run/RunStream to invoke providers, updated prepareMessages and prepareChatOptions to inject provider context, added getProviderContext and notifyProviderInvoked helper methods
* [go/README.md](go/README.md) - Added Context Providers section with documentation and examples

### Removed

## Additional or Deviating Changes

* Lifecycle hooks (Invoked, SessionCreated) integrated within Phase 4 implementation rather than as separate Phase 5
  * The Invoked hook is called in the Run method after the tool loop completes
  * SessionCreated hook notification helper added but session creation integration deferred to future work

## Release Summary

Total files affected: 7

**Files Created (3)**:
* `go/agent/context_provider.go` - Core ContextProvider types and interfaces
* `go/agent/context_provider_test.go` - Unit tests for agent package context provider code
* `go/chatagent/context_provider_test.go` - Integration tests for chatagent context provider support

**Files Modified (3)**:
* `go/chatagent/options.go` - Added contextProviders field and WithContextProvider option
* `go/chatagent/agent.go` - Integrated context provider invocation into Run/RunStream flow
* `go/README.md` - Added documentation section for Context Providers

**Validation Results**:
* `go build ./...` - PASS
* `go vet ./...` - PASS
* `go test ./agent/... ./chatagent/...` - PASS (all tests passing)

