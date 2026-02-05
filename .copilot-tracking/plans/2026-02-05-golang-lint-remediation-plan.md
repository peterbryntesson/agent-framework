# Go Lint Remediation Plan

**Created:** 2026-02-05  
**Status:** Draft  
**Linter:** golangci-lint v1.64.8  
**Total Issues:** 181 issues across 56 files

---

## Executive Summary

The Go codebase has accumulated 181 lint issues across various categories. This plan organizes them into prioritized epics based on severity, impact, and effort required.

---

## Issue Categories and Counts

| Category | Count | Priority | Effort |
|----------|-------|----------|--------|
| **errcheck** (unchecked error returns) | 25 | High | Medium |
| **fieldalignment** (struct memory optimization) | 31 | Low | Medium |
| **misspell** (spelling errors) | 13 | High | Low |
| **goconst** (repeated string literals) | 16 | Medium | Low |
| **gocyclo** (high cyclomatic complexity) | 19 | Medium | High |
| **revive/exported** (stuttering type names) | 22 | Low | Medium |
| **unused** (unused code) | 15 | High | Low |
| **gosec** (security issues) | 14 | High | Medium |
| **prealloc** (slice pre-allocation) | 7 | Low | Low |
| **unparam** (unused parameters) | 14 | Medium | Low |
| **empty-block** (empty blocks) | 10 | Medium | Low |
| **var-declaration** (redundant declarations) | 8 | Low | Low |
| **gocritic** (code style improvements) | 7 | Medium | Low |
| **ineffassign** (ineffectual assignments) | 4 | High | Low |
| **bodyclose** (unclosed response bodies) | 1 | Critical | Low |
| **Other** (misc issues) | 5 | Medium | Low |

---

## Epic 1: Critical and Security Issues (Priority: Immediate)

**Estimated Effort:** 2-3 hours  
**Risk if not fixed:** Memory leaks, security vulnerabilities, data corruption

### 1.1 Response Body Not Closed (bodyclose) - CRITICAL

| File | Line | Issue |
|------|------|-------|
| [protocol/a2a/client.go](../../../go/protocol/a2a/client.go#L212) | 212 | Response body must be closed |

**Fix:** Add `defer resp.Body.Close()` after error check.

### 1.2 Security Issues (gosec)

| File | Line | Issue |
|------|------|-------|
| [providers/openai/client.go](../../../go/providers/openai/client.go#L33) | 33 | G101: Potential hardcoded credentials |
| [observability/otel.go](../../../go/observability/otel.go#L37) | 37 | G101: Potential hardcoded credentials |
| [observability/otel.go](../../../go/observability/otel.go#L46) | 46 | G101: Potential hardcoded credentials |
| [observability/otel.go](../../../go/observability/otel.go#L49) | 49 | G101: Potential hardcoded credentials |
| [observability/semconv.go](../../../go/observability/semconv.go#L15) | 15 | G101: Potential hardcoded credentials |
| [observability/semconv.go](../../../go/observability/semconv.go#L18) | 18 | G101: Potential hardcoded credentials |
| [observability/semconv.go](../../../go/observability/semconv.go#L45) | 45 | G101: Potential hardcoded credentials |
| [observability/semconv.go](../../../go/observability/semconv.go#L48) | 48 | G101: Potential hardcoded credentials |
| [observability/semconv.go](../../../go/observability/semconv.go#L51) | 51 | G101: Potential hardcoded credentials |
| [testutil/fixtures.go](../../../go/testutil/fixtures.go#L39) | 39 | G304: Potential file inclusion via variable |
| [declarative/factory.go](../../../go/declarative/factory.go#L124) | 124 | G304: Potential file inclusion via variable |
| [declarative/loader.go](../../../go/declarative/loader.go#L17) | 17 | G304: Potential file inclusion via variable |
| [workflow/file_checkpoint.go](../../../go/workflow/file_checkpoint.go#L32) | 32 | G301: Expect directory permissions 0750 or less |
| [workflow/file_checkpoint.go](../../../go/workflow/file_checkpoint.go#L75) | 75 | G306: Expect WriteFile permissions 0600 or less |
| [workflow/file_checkpoint.go](../../../go/workflow/file_checkpoint.go#L198) | 198 | G304: Potential file inclusion via variable |
| [mcp/transport_stdio.go](../../../go/mcp/transport_stdio.go#L93) | 93 | G204: Subprocess launched with potential tainted input |
| [resilience/retry.go](../../../go/resilience/retry.go#L163) | 163 | G404: Use of weak random number generator |
| [workflow/groupchat/selectors.go](../../../go/workflow/groupchat/selectors.go#L102) | 102 | G404: Use of weak random number generator |

**Fixes:**
- G101 issues are false positives for semantic convention constants (add `//nolint:gosec` comment or exclude in config)
- G304 issues require input validation or path sanitization
- G301/G306: Use stricter permissions (0750 for dirs, 0600 for files)
- G204: Validate command inputs
- G404: Use crypto/rand for security-sensitive randomness, or add nolint for non-security contexts

### 1.3 Ineffectual Assignments (ineffassign)

| File | Line | Issue |
|------|------|-------|
| [protocol/a2a/session_test.go](../../../go/protocol/a2a/session_test.go#L103) | 103 | Ineffectual assignment to messages |
| [chatagent/toolloop.go](../../../go/chatagent/toolloop.go#L247) | 247 | Ineffectual assignment to toolCallID |
| [chatagent/session_test.go](../../../go/chatagent/session_test.go#L218) | 218 | Ineffectual assignment to messages |

**Fix:** Remove unused assignments or use the variable.

---

## Epic 2: Error Handling Issues (Priority: High)

**Estimated Effort:** 3-4 hours  
**Risk if not fixed:** Silent failures, undefined behavior

### 2.1 Unchecked Error Returns (errcheck)

| File | Line | Function/Call |
|------|------|---------------|
| [providers/openai/client.go](../../../go/providers/openai/client.go#L178) | 178 | `stream.Close()` |
| [providers/openai/responses.go](../../../go/providers/openai/responses.go#L320) | 320 | Map assignment without error check |
| [providers/openai/responses.go](../../../go/providers/openai/responses.go#L454) | 454 | `resp.Body.Close()` |
| [observability/instrumented.go](../../../go/observability/instrumented.go#L30) | 30 | `NewMetrics()` |
| [observability/instrumented.go](../../../go/observability/instrumented.go#L297) | 297 | `NewMetrics()` |
| [chat/content_test.go](../../../go/chat/content_test.go#L558) | 558 | Type assertion result |

**Test Files (lower priority):**

| File | Line | Function/Call |
|------|------|---------------|
| [providers/openai/client_test.go](../../../go/providers/openai/client_test.go#L209) | 209 | `w.Write()` |
| [providers/openai/client_test.go](../../../go/providers/openai/client_test.go#L549) | 549 | `decoder.Decode()` |
| [providers/openai/client_test.go](../../../go/providers/openai/client_test.go#L724) | 724 | `decoder.Decode()` |
| [providers/openai/client_test.go](../../../go/providers/openai/client_test.go#L795) | 795 | `decoder.Decode()` |
| [providers/openai/client_test.go](../../../go/providers/openai/client_test.go#L864) | 864 | `decoder.Decode()` |
| [providers/openai/client_test.go](../../../go/providers/openai/client_test.go#L873) | 873 | Type assertion |
| [providers/openai/responses_test.go](../../../go/providers/openai/responses_test.go#L169) | 169 | `Encode()` |
| [providers/openai/responses_test.go](../../../go/providers/openai/responses_test.go#L264) | 264 | `Encode()` |
| [providers/openai/responses_test.go](../../../go/providers/openai/responses_test.go#L309) | 309 | `Decode()` |
| [providers/openai/responses_test.go](../../../go/providers/openai/responses_test.go#L337) | 337 | `Encode()` |
| [providers/openai/responses_test.go](../../../go/providers/openai/responses_test.go#L370) | 370 | `Encode()` |
| [providers/openai/responses_test.go](../../../go/providers/openai/responses_test.go#L398) | 398 | `Decode()` |
| [providers/openai/responses_test.go](../../../go/providers/openai/responses_test.go#L425) | 425 | `Encode()` |
| [protocol/a2a/agent_test.go](../../../go/protocol/a2a/agent_test.go#L194) | 194 | `Encode()` |
| [protocol/a2a/agent_test.go](../../../go/protocol/a2a/agent_test.go#L209) | 209 | `Encode()` |

**Fix:** Check and handle errors appropriately. For test files, use `require.NoError()` or explicit error handling.

---

## Epic 3: Spelling and Naming Consistency (Priority: High)

**Estimated Effort:** 1-2 hours  
**Risk if not fixed:** Inconsistency, confusion, potential API issues

### 3.1 Misspellings (cancelled → canceled)

| File | Line | Context |
|------|------|---------|
| [protocol/agui/server.go](../../../go/protocol/agui/server.go#L224) | 224 | Comment: "cancelled" |
| [protocol/agui/server_test.go](../../../go/protocol/agui/server_test.go#L532) | 532 | Variable: `cancelled` |
| [protocol/agui/server_test.go](../../../go/protocol/agui/server_test.go#L533) | 533 | Variable: `cancelled` |
| [protocol/agui/server_test.go](../../../go/protocol/agui/server_test.go#L635) | 635 | Comment: "cancelled" |
| [providers/openai/client.go](../../../go/providers/openai/client.go#L123) | 123 | Comment: "cancelled" |
| [providers/openai/doc.go](../../../go/providers/openai/doc.go#L110) | 110 | Comment: "cancelled" |
| [providers/openai/responses.go](../../../go/providers/openai/responses.go#L590) | 590 | String literal: "cancelled" |
| [chat/client.go](../../../go/chat/client.go#L20) | 20 | Comment: "cancelled" |
| [protocol/a2a/types.go](../../../go/protocol/a2a/types.go#L94) | 94 | Comment: "cancelled" |
| [protocol/a2a/types.go](../../../go/protocol/a2a/types.go#L95) | 95 | Constant value: "cancelled" |

**Note:** For `protocol/a2a/types.go:95` and `providers/openai/responses.go:590`, the string "cancelled" may be required by external API specifications. Verify before changing.

---

## Epic 4: Code Quality Improvements (Priority: Medium)

**Estimated Effort:** 4-6 hours

### 4.1 Define String Constants (goconst)

**providers/openai package:**

| Constant | Occurrences | Files |
|----------|-------------|-------|
| `"user"` | 4 | responses.go |
| `"required"` | 4 | client.go |
| `"developer"` | 10 | convert.go |
| `"auto"` | 4 | client.go |
| `"function_call"` | 3 | convert.go |
| `"none"` | 4 | client.go |

**Other packages:**

| Constant | Occurrences | Package |
|----------|-------------|---------|
| `"Prompt"` | 3 | declarative |
| `"Tool invocation failed"` | 4 | tool |
| `"integer"` | 4 | tool |
| `"content_delta"` | 3 | chatagent |
| `"stop"` | 3 | chatagent |
| `"web_search"` | 4 | tool (constant exists but not used) |
| `"file_search"` | 3 | tool (constant exists but not used) |
| `"image_generation"` | 3 | tool (constant exists but not used) |
| `"code_interpreter"` | 3 | tool (constant exists but not used) |

### 4.2 Empty Blocks (empty-block)

| File | Line | Context |
|------|------|---------|
| [providers/openai/client_test.go](../../../go/providers/openai/client_test.go#L847) | 847 | `for range updates {}` |
| [providers/openai/client_test.go](../../../go/providers/openai/client_test.go#L913) | 913 | `for range updates {}` |
| [observability/instrumented_test.go](../../../go/observability/instrumented_test.go#L320) | 320 | `for range updates {}` |
| [workflow/runner_test.go](../../../go/workflow/runner_test.go#L397) | 397 | `for range updates {}` |
| [chatagent/chat_middleware_test.go](../../../go/chatagent/chat_middleware_test.go#L78) | 78 | `for range updates {}` |
| [chatagent/chat_middleware_test.go](../../../go/chatagent/chat_middleware_test.go#L310) | 310 | `for range updates {}` |
| [chatagent/context_provider_test.go](../../../go/chatagent/context_provider_test.go#L332) | 332 | `for range updates {}` |
| [agent/middleware_agent_test.go](../../../go/agent/middleware_agent_test.go#L183) | 183 | `for range stream {}` |
| [mcp/client.go](../../../go/mcp/client.go#L133) | 133 | Empty error handling block |

**Fix:** Add comment explaining why the block is empty, or remove if not needed.

### 4.3 Variable Declarations (var-declaration)

| File | Line | Issue |
|------|------|-------|
| [chat/client_test.go](../../../go/chat/client_test.go#L213) | 213 | `var response *Response = nil` → `var response *Response` |
| [chat/content_test.go](../../../go/chat/content_test.go#L302) | 302 | `var msg *Message = nil` → `var msg *Message` |
| [internal/json/utils_test.go](../../../go/internal/json/utils_test.go#L85) | 85 | Drop `= nil` |
| [internal/json/utils_test.go](../../../go/internal/json/utils_test.go#L97) | 97 | Drop `= nil` |
| [internal/json/utils_test.go](../../../go/internal/json/utils_test.go#L210) | 210 | Drop `= nil` |
| [internal/json/utils_test.go](../../../go/internal/json/utils_test.go#L248) | 248 | Drop `= nil` |
| [observability/setup_test.go](../../../go/observability/setup_test.go#L98) | 98 | Omit type (can be inferred) |
| [observability/setup_test.go](../../../go/observability/setup_test.go#L107) | 107 | Omit type (can be inferred) |
| [observability/setup_test.go](../../../go/observability/setup_test.go#L116) | 116 | Omit type (can be inferred) |

### 4.4 Append Assignment Issue (gocritic)

| File | Line | Issue |
|------|------|-------|
| [observability/setup.go](../../../go/observability/setup.go#L247) | 247 | `allOpts := append(opts, ...)` - append result not assigned to same slice |
| [observability/setup.go](../../../go/observability/setup.go#L254) | 254 | Same issue |

**Fix:** Either assign to `opts` or use a different variable name and document the intent.

### 4.5 Case Order in Type Switch (gocritic)

| File | Line | Issue |
|------|------|-------|
| [workflow/stateful_executor.go](../../../go/workflow/stateful_executor.go#L153) | 153 | `case json.RawMessage` must go before `T` case |

### 4.6 Unlambda (gocritic)

| File | Line | Function |
|------|------|----------|
| [chat/mock_examples_test.go](../../../go/chat/mock_examples_test.go#L34) | 34 | Replace lambda with `NewMockChatClient` |
| [chat/mock_examples_test.go](../../../go/chat/mock_examples_test.go#L133) | 133 | Same |
| [chat/mock_examples_test.go](../../../go/chat/mock_examples_test.go#L209) | 209 | Same |
| [agent/mock_examples_test.go](../../../go/agent/mock_examples_test.go#L27) | 27 | Replace lambda with `NewMockAgent` |
| [agent/mock_examples_test.go](../../../go/agent/mock_examples_test.go#L90) | 90 | Same |
| [agent/mock_examples_test.go](../../../go/agent/mock_examples_test.go#L178) | 178 | Same |
| [agent/mock_examples_test.go](../../../go/agent/mock_examples_test.go#L258) | 258 | Same |

### 4.7 If-Else to Switch Statement (gocritic)

| File | Line | Issue |
|------|------|-------|
| [protocol/a2a/client.go](../../../go/protocol/a2a/client.go#L258) | 258 | Rewrite if-else to switch |
| [mcp/tool.go](../../../go/mcp/tool.go#L145) | 145 | Rewrite if-else to switch |

### 4.8 Loop Simplification (gosimple)

| File | Line | Issue |
|------|------|-------|
| [chatagent/toolloop.go](../../../go/chatagent/toolloop.go#L508) | 508 | Use `copy(to, from)` instead of loop |

### 4.9 Boolean Comparison (gosimple)

| File | Line | Issue |
|------|------|-------|
| [devui/tracing.go](../../../go/devui/tracing.go#L132) | 132 | Simplify `== false` to `!` |

### 4.10 Format Issues (gofmt)

| File | Line | Issue |
|------|------|-------|
| [tool/invoke_test.go](../../../go/tool/invoke_test.go#L22) | 22 | File not properly formatted |

---

## Epic 5: Unused Code Cleanup (Priority: Medium)

**Estimated Effort:** 2-3 hours

### 5.1 Unused Functions

| File | Line | Function |
|------|------|----------|
| [workflow/runner.go](../../../go/workflow/runner.go#L267) | 267 | `getExecutorOptions` |
| [durable/workflow.go](../../../go/durable/workflow.go#L227) | 227 | `registerHistoryQuery` |
| [chatagent/agent.go](../../../go/chatagent/agent.go#L371) | 371 | `notifyProviderSessionCreated` |

### 5.2 Unused Types

| File | Line | Type |
|------|------|------|
| [observability/otel_test.go](../../../go/observability/otel_test.go#L122) | 122 | `mockSpan` |
| [devui/discovery_test.go](../../../go/devui/discovery_test.go#L13) | 13 | `mockAgent` |
| [devui/discovery_test.go](../../../go/devui/discovery_test.go#L24) | 24 | `mockMetadata` |
| [mcp/tool_test.go](../../../go/mcp/tool_test.go#L17) | 17 | `mockClientForTool` |
| [tool/function_test.go](../../../go/tool/function_test.go#L24) | 24 | `optionalArgs` |
| [chatagent/astool.go](../../../go/chatagent/astool.go#L108) | 108 | `taskArgs` |

### 5.3 Unused Functions in Test Files

| File | Line | Function |
|------|------|----------|
| [observability/otel_test.go](../../../go/observability/otel_test.go#L127) | 127 | `(*mockSpan).SetAttributes` |
| [devui/discovery_test.go](../../../go/devui/discovery_test.go#L19) | 19 | `(*mockAgent).ID`, `Name`, `Description`, `Metadata` |
| [mcp/tool_test.go](../../../go/mcp/tool_test.go#L28) | 28 | `(*mockClientForTool).ListTools`, `CallTool` |
| [tool/function_test.go](../../../go/tool/function_test.go#L58) | 58 | `noReturnFunc` |
| [tool/function_test.go](../../../go/tool/function_test.go#L62) | 62 | `panicFunc` |

### 5.4 Unused Fields

| File | Line | Field |
|------|------|-------|
| [hosting/builder_test.go](../../../go/hosting/builder_test.go#L55) | 55 | `data` field |
| [tool/schema_test.go](../../../go/tool/schema_test.go#L76) | 76 | `private` field |

### 5.5 Unused Writes (govet)

| File | Line | Field |
|------|------|-------|
| [chat/client_test.go](../../../go/chat/client_test.go#L514) | 514 | `FinishReason` |
| [chat/client_test.go](../../../go/chat/client_test.go#L515) | 515 | `Error` |
| [chat/content_test.go](../../../go/chat/content_test.go#L483) | 483 | `Role` |

---

## Epic 6: Unused Parameters (Priority: Medium)

**Estimated Effort:** 2-3 hours

### 6.1 Production Code

| File | Line | Function | Parameter |
|------|------|----------|-----------|
| [protocol/agui/converter.go](../../../go/protocol/agui/converter.go#L303) | 303 | `convertDone` | `update` |
| [providers/openai/responses.go](../../../go/providers/openai/responses.go#L600) | 600 | `processStreamingResponse` | `ctx` |
| [providers/openai/stream.go](../../../go/providers/openai/stream.go#L191) | 191 | `flushToolCalls` | `updates` |
| [provider/textsearch/provider.go](../../../go/provider/textsearch/provider.go#L233) | 233 | `invokingOnDemand` | `ctx` |
| [purview/middleware.go](../../../go/purview/middleware.go#L117) | 117 | `handleViolation` | `contentType` |
| [mcp/server.go](../../../go/mcp/server.go#L124) | 124 | `handleInitialize` | `ctx` |
| [mcp/server.go](../../../go/mcp/server.go#L143) | 143 | `handleToolsList` | `ctx` |
| [mcp/server.go](../../../go/mcp/server.go#L202) | 202 | `handleResourcesList` | `ctx` |
| [mcp/transport_http.go](../../../go/mcp/transport_http.go#L224) | 224 | `handleSSEEvent` | `eventType` |
| [declarative/eval.go](../../../go/declarative/eval.go#L43) | 43 | `evaluateAgent` | return always nil |

### 6.2 Test Code (always receives same value)

| File | Line | Function | Parameter |
|------|------|----------|-----------|
| [protocol/a2a/session_test.go](../../../go/protocol/a2a/session_test.go#L256) | 256 | Anonymous func | `n` |
| [protocol/a2a/session_test.go](../../../go/protocol/a2a/session_test.go#L277) | 277 | Anonymous func | `n` |
| [protocol/a2a/session_test.go](../../../go/protocol/a2a/session_test.go#L282) | 282 | Anonymous func | `n` |
| [workflow/context_test.go](../../../go/workflow/context_test.go#L110) | 110 | Anonymous func | `idx` |
| [agent/session_test.go](../../../go/agent/session_test.go#L193) | 193 | Anonymous func | `index` |
| [durable/session_test.go](../../../go/durable/session_test.go#L145) | 145 | Anonymous func | `n` |
| [hosting/sessionstore_test.go](../../../go/hosting/sessionstore_test.go#L157) | 157 | Anonymous func | `i` |
| [workflow/executors/agent_test.go](../../../go/workflow/executors/agent_test.go#L228) | 228 | `newTestWorkflowContext` | `executorID` |
| [workflow/executors/aggregating_test.go](../../../go/workflow/executors/aggregating_test.go#L19) | 19 | `newAggTextMessage` | `role` |
| [workflow/executors/aggregating_test.go](../../../go/workflow/executors/aggregating_test.go#L364) | 364 | `newAggregatingTestContext` | `executorID` |
| [workflow/groupchat/selectors_test.go](../../../go/workflow/groupchat/selectors_test.go#L28) | 28 | `newMockAgentWithResponse` | `id` |
| [purview/middleware_test.go](../../../go/purview/middleware_test.go#L18) | 18 | `createMockServer` | `t` |

---

## Epic 7: High Cyclomatic Complexity (Priority: Medium-Low)

**Estimated Effort:** 8-12 hours (requires careful refactoring)

These functions have complexity > 15 and would benefit from refactoring:

| File | Line | Function | Complexity |
|------|------|----------|------------|
| [chatagent/toolloop.go](../../../go/chatagent/toolloop.go#L215) | 215 | `collectStreamUpdates` | 29 |
| [tool/function_test.go](../../../go/tool/function_test.go#L70) | 70 | `TestNewFunctionTool` | 29 |
| [tool/invoke_test.go](../../../go/tool/invoke_test.go#L101) | 101 | `TestInvoker_Invoke` | 22 |
| [tool/function_test.go](../../../go/tool/function_test.go#L240) | 240 | `TestFunctionTool_Invoke` | 22 |
| [workflow/runner_test.go](../../../go/workflow/runner_test.go#L203) | 203 | `TestWorkflowRunner_RunStream` | 22 |
| [workflow/workflow.go](../../../go/workflow/workflow.go#L119) | 119 | `Validate` | 21 |
| [agent/response_extensions.go](../../../go/agent/response_extensions.go#L12) | 12 | `ToAgentResponse` | 21 |
| [devui/handlers.go](../../../go/devui/handlers.go#L101) | 101 | `handleStreamAgent` | 21 |
| [protocol/a2a/client_test.go](../../../go/protocol/a2a/client_test.go#L322) | 322 | `TestClient_SendMessageStream` | 20 |
| [providers/openai/convert_test.go](../../../go/providers/openai/convert_test.go#L61) | 61 | `TestToOpenAIMessage` | 20 |
| [tool/function_test.go](../../../go/tool/function_test.go#L471) | 471 | `TestFunc` | 19 |
| [observability/instrumented.go](../../../go/observability/instrumented.go#L94) | 94 | `GetStreamingResponse` | 19 |
| [providers/openai/client.go](../../../go/providers/openai/client.go#L187) | 187 | `buildRequest` | 18 |
| [tool/schema_test.go](../../../go/tool/schema_test.go#L85) | 85 | `TestGenerateSchema_BasicTypes` | 18 |
| [chatagent/astool.go](../../../go/chatagent/astool.go#L70) | 70 | `AsTool` | 18 |
| [providers/openai/client_test.go](../../../go/providers/openai/client_test.go#L18) | 18 | `TestNewClient` | 17 |
| [providers/openai/stream.go](../../../go/providers/openai/stream.go#L203) | 203 | `CollectStreamToResponse` | 17 |
| [tool/schema_test.go](../../../go/tool/schema_test.go#L421) | 421 | `TestGenerateSchema_RequiredFields` | 17 |
| [workflow/groupchat/manager.go](../../../go/workflow/groupchat/manager.go#L221) | 221 | `runStreamInternal` | 17 |
| [protocol/a2a/server.go](../../../go/protocol/a2a/server.go#L188) | 188 | `handleSendMessageStream` | 16 |
| [protocol/a2a/agent_test.go](../../../go/protocol/a2a/agent_test.go#L285) | 285 | `TestA2AAgent_RunStream` | 16 |
| [protocol/a2a/agent_test.go](../../../go/protocol/a2a/agent_test.go#L177) | 177 | `TestA2AAgent_Run` | 16 |
| [chatagent/agent_test.go](../../../go/chatagent/agent_test.go#L180) | 180 | `TestAgentRun` | 16 |
| [devui/tracing.go](../../../go/devui/tracing.go#L43) | 43 | `ExportSpans` | 16 |

**Refactoring Strategies:**
1. Extract helper functions
2. Use early returns
3. Split into smaller, focused functions
4. Use strategy pattern for complex switch statements
5. For test functions, split into subtests

---

## Epic 8: Naming Conventions (Priority: Low)

**Estimated Effort:** 4-6 hours

### 8.1 Stuttering Type Names (revive/exported)

These type names stutter when used with the package name:

| Package | Current Name | Suggested Name |
|---------|--------------|----------------|
| a2a | `A2ASession` | `Session` |
| a2a | `A2AAgent` | `Agent` |
| a2a | `A2AAgentOption` | `AgentOption` |
| workflow | `WorkflowContext` | `Context` |
| workflow | `WorkflowMessage` | `Message` |
| workflow | `WorkflowResult` | `Result` |
| workflow | `WorkflowEventKind` | `EventKind` |
| workflow | `WorkflowEvent` | `Event` |
| workflow | `WorkflowOutputEvent` | `OutputEvent` |
| workflow | `WorkflowBuilder` | `Builder` |
| workflow | `WorkflowRunner` | `Runner` |
| tool | `ToolChoice` | `Choice` |
| tool | `ToolType` | `Type` |
| tool | `ToolCall` | `Call` |
| tool | `ToolResult` | `Result` |
| agent | `AgentFactory` | `Factory` |
| agent | `AgentBuilder` | `Builder` |
| agent | `AgentContext` | `Context` |
| agent | `AgentHandler` | `Handler` |
| agent | `AgentMiddleware` | `Middleware` |
| agent | `AgentMiddlewareFunc` | `MiddlewareFunc` |
| validation | `ValidationError` | `Error` |
| mcp | `MCPError` | `Error` |

**Note:** These are breaking API changes. Consider deprecating old names and adding type aliases.

### 8.2 Error Variable Naming (revive/error-naming)

| File | Line | Current | Suggested |
|------|------|---------|-----------|
| [tool/hosted.go](../../../go/tool/hosted.go#L12) | 12 | `hostedInvocationError` | `errHostedInvocation` |

### 8.3 Error String Capitalization (stylecheck)

| File | Line | Issue |
|------|------|-------|
| [purview/client.go](../../../go/purview/client.go#L100) | 100 | Error strings should not be capitalized |
| [purview/client.go](../../../go/purview/client.go#L144) | 144 | Same |

### 8.4 Context Parameter Position (revive/context-as-argument)

| File | Line | Function |
|------|------|----------|
| [hosting/openai/streaming.go](../../../go/hosting/openai/streaming.go#L20) | 20 | Context should be first parameter |
| [hosting/openai/completions.go](../../../go/hosting/openai/completions.go#L52) | 52 | Same |

### 8.5 Init Function (gochecknoinits)

| File | Line | Issue |
|------|------|-------|
| [chatagent/builder_test.go](../../../go/chatagent/builder_test.go#L374) | 374 | Don't use `init` function |

---

## Epic 9: Memory Optimization (Priority: Low)

**Estimated Effort:** 3-4 hours

### 9.1 Struct Field Alignment (fieldalignment)

These structs can be optimized by reordering fields:

| File | Line | Struct | Current | Optimal |
|------|------|--------|---------|---------|
| [chat/client.go](../../../go/chat/client.go#L53) | 53 | `Options` | 192 | 152 |
| [chat/message.go](../../../go/chat/message.go#L29) | 29 | `Message` | 136 | 120 |
| [chat/response.go](../../../go/chat/response.go#L7) | 7 | `Response` | 168 | 160 |
| [chat/response.go](../../../go/chat/response.go#L33) | 33 | `ResponseUpdate` | 64 | 48 |
| [chat/client.go](../../../go/chat/client.go#L121) | 121 | `ToolDefinition` | 40 | 32 |
| [providers/openai/options.go](../../../go/providers/openai/options.go#L11) | 11 | `config` | 88 | 80 |
| [providers/openai/responses.go](../../../go/providers/openai/responses.go#L50) | 50 | `ResponsesClient` | 184 | 160 |
| [providers/openai/responses.go](../../../go/providers/openai/responses.go#L515) | 515 | `responsesAPIResponse` | 80 | 72 |
| [providers/openai/responses.go](../../../go/providers/openai/responses.go#L521) | 521 | Anonymous struct | 136 | 128 |
| [protocol/a2a/client.go](../../../go/protocol/a2a/client.go#L18) | 18 | `Client` | 32 | 24 |
| [protocol/a2a/server.go](../../../go/protocol/a2a/server.go#L307) | 307 | `sessionTasks` | 16 | 8 |
| [protocol/a2a/types.go](../../../go/protocol/a2a/types.go#L46) | 46 | `AgentCapabilities` | 16 | 8 |
| [protocol/a2a/types.go](../../../go/protocol/a2a/types.go#L100) | 100 | `Task` | 160 | 144 |
| [protocol/a2a/types.go](../../../go/protocol/a2a/types.go#L139) | 139 | `CreateTaskRequest` | 32 | 24 |
| [protocol/a2a/types.go](../../../go/protocol/a2a/types.go#L166) | 166 | `Message` | 72 | 56 |
| [protocol/a2a/types.go](../../../go/protocol/a2a/types.go#L259) | 259 | `Artifact` | 104 | 88 |
| [protocol/a2a/types.go](../../../go/protocol/a2a/types.go#L298) | 298 | `StreamEvent` | 64 | 56 |
| [workflow/checkpoint.go](../../../go/workflow/checkpoint.go#L14) | 14 | `Checkpoint` | 112 | 88 |
| [observability/setup.go](../../../go/observability/setup.go#L20) | 20 | `SetupConfig` | 96 | 88 |

**Note:** Only apply to exported structs that are frequently allocated. Test code struct alignment is low priority.

---

## Epic 10: Performance Optimizations (Priority: Low)

**Estimated Effort:** 1-2 hours

### 10.1 Slice Pre-allocation (prealloc)

| File | Line | Variable |
|------|------|----------|
| [protocol/agui/converter.go](../../../go/protocol/agui/converter.go#L110) | 110 | `events` |
| [protocol/agui/converter.go](../../../go/protocol/agui/converter.go#L121) | 121 | `events` |
| [protocol/agui/converter.go](../../../go/protocol/agui/converter.go#L242) | 242 | `events` |
| [provider/textsearch/integration_test.go](../../../go/provider/textsearch/integration_test.go#L33) | 33 | `results` |
| [hosting/openai/models.go](../../../go/hosting/openai/models.go#L333) | 333 | `toolCalls` |
| [hosting/openai/models_test.go](../../../go/hosting/openai/models_test.go#L221) | 221 | `roundTripped` |
| [agent/response_extensions.go](../../../go/agent/response_extensions.go#L105) | 105 | `collected` |

---

## Implementation Order

### Phase 1: Critical Fixes (Week 1)
1. Epic 1 - Critical and Security Issues
2. Epic 2 - Error Handling Issues (production code only)

### Phase 2: High-Priority Cleanup (Week 2)
3. Epic 3 - Spelling and Naming Consistency
4. Epic 5 - Unused Code Cleanup
5. Epic 4.10 - Format Issues

### Phase 3: Code Quality (Week 3)
6. Epic 4 - Code Quality Improvements (remaining)
7. Epic 6 - Unused Parameters

### Phase 4: Refactoring (Week 4-5)
8. Epic 7 - High Cyclomatic Complexity (prioritize production code)

### Phase 5: Long-term Improvements (Future)
9. Epic 8 - Naming Conventions (requires API versioning strategy)
10. Epic 9 - Memory Optimization
11. Epic 10 - Performance Optimizations

---

## Configuration Recommendations

Consider adding a `.golangci.yml` configuration file to:
1. Exclude false positives (e.g., G101 for semantic conventions)
2. Exclude test files from certain linters
3. Set appropriate complexity thresholds
4. Configure field alignment for only exported types

Example configuration additions:

```yaml
linters-settings:
  gosec:
    excludes:
      - G101  # Hardcoded credentials (false positive for constants)
  gocyclo:
    min-complexity: 20  # Slightly more lenient threshold
  govet:
    enable:
      - fieldalignment
    settings:
      fieldalignment:
        suggest-new: true

issues:
  exclude-rules:
    - path: _test\.go
      linters:
        - errcheck
        - gocyclo
        - fieldalignment
    - path: testutil/
      linters:
        - gosec
```

---

## Success Metrics

- [ ] All critical issues (Epic 1) resolved
- [ ] All high-priority error handling issues resolved
- [ ] Zero spelling inconsistencies
- [ ] No unused production code
- [ ] Average cyclomatic complexity < 15
- [ ] golangci-lint passes with standard configuration
