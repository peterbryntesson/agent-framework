<!-- markdownlint-disable-file -->
# Release Changes: Go ChatMiddleware for Chat Client Interception

**Related Plan**: 2026-02-03-go-chatmiddleware-plan.instructions.md
**Implementation Date**: 2026-02-03

## Summary

Implemented ChatMiddleware for the Go agent framework to intercept chat client requests (GetResponse/GetStreamingResponse), completing the middleware triad (AgentMiddleware, FunctionMiddleware, ChatMiddleware).

## Changes

### Added

* go/agent/middleware_context.go - Added ChatContext, ChatClientMetadata, ChatResponse, and ChatResponseUpdate structs for middleware context
* go/agent/middleware.go - Added ChatHandler, ChatMiddleware interface, and ChatMiddlewareFunc adapter with compile-time interface check
* go/agent/chain.go - Added ChainChatMiddleware function to compose multiple chat middlewares
* go/agent/chat_middleware_test.go - Comprehensive unit tests for ChatMiddleware types and chain composition
* go/chatagent/options.go - Added chatMiddleware field to config and WithChatMiddleware option
* go/chatagent/toolloop.go - Added invokeWithChatMiddleware helper with conversion functions for messages, options, responses, and streams
* go/chatagent/builder.go - Added UseChatMiddleware method for builder pattern support
* go/chatagent/chat_middleware_test.go - Integration tests for ChatMiddleware in ChatClientAgent

### Modified

* go/chatagent/agent.go - Added chatMiddleware field to Agent struct and initialization in New function
* go/chatagent/toolloop.go - Modified runWithToolLoop and processStreamWithToolLoop to use invokeWithChatMiddleware instead of direct client calls; added nil check for Usage in response handling
* go/agent/doc.go - Added Middleware documentation section explaining all three middleware types with example
* go/README.md - Added Chat Middleware subsection with caching middleware example and updated middleware section header

### Removed

## Additional or Deviating Changes

* Added parseFinishReason, finishReasonToString, parseUpdateKind, and updateKindToString functions to properly convert between string-based middleware types and integer-based chat package types

## Release Summary

**Total Files Affected**: 10 files

**Files Created**: 2

* go/agent/chat_middleware_test.go - Unit tests for ChatMiddleware core types
* go/chatagent/chat_middleware_test.go - Integration tests for ChatMiddleware with ChatClientAgent

**Files Modified**: 8

* go/agent/middleware_context.go - Added ChatContext and related types
* go/agent/middleware.go - Added ChatMiddleware interface and adapter
* go/agent/chain.go - Added ChainChatMiddleware composition
* go/agent/doc.go - Added middleware documentation
* go/chatagent/options.go - Added WithChatMiddleware option
* go/chatagent/agent.go - Added chatMiddleware field
* go/chatagent/toolloop.go - Integrated ChatMiddleware into tool loop
* go/chatagent/builder.go - Added UseChatMiddleware builder method
* go/README.md - Added Chat Middleware documentation section

**Validation Results**:

* `go build ./...` - PASSED
* `go vet ./...` - PASSED
* `go test ./agent/... ./chatagent/...` - PASSED
