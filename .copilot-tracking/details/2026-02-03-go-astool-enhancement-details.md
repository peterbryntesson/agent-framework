<!-- markdownlint-disable-file -->
# Implementation Details: Go AsTool Enhancement for Runtime Context Propagation

## Context Reference

Sources:
- [go/chatagent/astool.go](go/chatagent/astool.go) - Current implementation
- [python/packages/core/agent_framework/_agents.py](python/packages/core/agent_framework/_agents.py) (Lines 411-510) - Python reference
- [.copilot-tracking/subagent/2026-02-03/astool-comparison-research.md](.copilot-tracking/subagent/2026-02-03/astool-comparison-research.md) - Gap analysis

## Implementation Phase 1: Runtime Context Propagation

<!-- parallelizable: false -->

### Step 1.1: Define RuntimeContext type for propagated values

Create a RuntimeContext type that can carry arbitrary key-value pairs through agent delegation chains, similar to Python's kwargs propagation.

Files:
* `go/agent/runtime_context.go` - New file for RuntimeContext type

Success criteria:
* RuntimeContext struct defined with thread-safe access
* Values method returns all stored key-value pairs
* Get method retrieves specific value by key
* With method creates new context with additional value

Context references:
* [python/packages/core/agent_framework/_agents.py](python/packages/core/agent_framework/_agents.py) (Lines 477-479) - `forwarded_kwargs = {k: v for k, v in kwargs.items() if k not in (arg_name, "conversation_id")}`

Dependencies:
* None (new standalone file)

```go
// Copyright (c) Microsoft. All rights reserved.

package agent

import "sync"

// RuntimeContext holds key-value pairs that propagate through agent delegation chains.
// This enables parent agents to forward context (user IDs, API tokens, session data)
// to sub-agents without modifying function signatures.
//
// RuntimeContext is immutable; With() returns a new context with additional values.
// Thread-safe for concurrent reads.
type RuntimeContext struct {
	values map[string]interface{}
	mu     sync.RWMutex
}

// NewRuntimeContext creates an empty RuntimeContext.
func NewRuntimeContext() *RuntimeContext {
	return &RuntimeContext{
		values: make(map[string]interface{}),
	}
}

// With returns a new RuntimeContext containing the original values plus the new key-value pair.
// The original context is not modified.
func (c *RuntimeContext) With(key string, value interface{}) *RuntimeContext {
	c.mu.RLock()
	defer c.mu.RUnlock()

	newValues := make(map[string]interface{}, len(c.values)+1)
	for k, v := range c.values {
		newValues[k] = v
	}
	newValues[key] = value

	return &RuntimeContext{values: newValues}
}

// Get retrieves a value by key. Returns nil and false if key is not present.
func (c *RuntimeContext) Get(key string) (interface{}, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	v, ok := c.values[key]
	return v, ok
}

// Values returns a copy of all key-value pairs in the context.
func (c *RuntimeContext) Values() map[string]interface{} {
	c.mu.RLock()
	defer c.mu.RUnlock()

	result := make(map[string]interface{}, len(c.values))
	for k, v := range c.values {
		result[k] = v
	}
	return result
}

// Len returns the number of values in the context.
func (c *RuntimeContext) Len() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.values)
}
```

### Step 1.2: Add context key and extraction functions for RuntimeContext

Add functions to store and retrieve RuntimeContext from context.Context, following Go's context value pattern.

Files:
* `go/agent/runtime_context.go` - Append to existing file

Success criteria:
* WithRuntimeCtx function adds RuntimeContext to context.Context
* RuntimeCtxFromContext extracts RuntimeContext from context.Context
* Returns empty RuntimeContext if not present (never nil)

Context references:
* [go/agent/middleware_context.go](go/agent/middleware_context.go) - Existing context value patterns

Dependencies:
* Step 1.1 completion

```go
// runtimeCtxKey is the context key for RuntimeContext values.
type runtimeCtxKey struct{}

// WithRuntimeCtx returns a new context.Context containing the RuntimeContext.
func WithRuntimeCtx(ctx context.Context, rtc *RuntimeContext) context.Context {
	return context.WithValue(ctx, runtimeCtxKey{}, rtc)
}

// RuntimeCtxFromContext extracts RuntimeContext from context.Context.
// Returns an empty RuntimeContext if not present.
func RuntimeCtxFromContext(ctx context.Context) *RuntimeContext {
	if rtc, ok := ctx.Value(runtimeCtxKey{}).(*RuntimeContext); ok && rtc != nil {
		return rtc
	}
	return NewRuntimeContext()
}
```

### Step 1.3: Add WithRuntimeContext RunOption to propagate context through agent calls

Create a RunOption that attaches RuntimeContext to RunConfig for propagation through Agent.Run calls.

Files:
* `go/agent/run_option.go` - Add new RunOption function
* `go/agent/run_config.go` - Add RuntimeContext field if needed

Success criteria:
* WithRuntimeContext RunOption function defined
* RunConfig stores RuntimeContext
* Context accessible in agent implementations

Context references:
* [go/agent/run_option.go](go/agent/run_option.go) - Existing RunOption patterns
* [go/chatagent/options.go](go/chatagent/options.go) - Functional options pattern

Dependencies:
* Step 1.2 completion

```go
// In run_config.go, add field to RunConfig:
type RunConfig struct {
	// ... existing fields ...
	
	// RuntimeContext contains key-value pairs to propagate through agent chains.
	RuntimeContext *RuntimeContext
}

// In run_option.go:

// WithRuntimeContext attaches a RuntimeContext to the run configuration.
// Use this to propagate context (user IDs, API tokens, session data) to sub-agents.
func WithRuntimeContext(rtc *RuntimeContext) RunOption {
	return func(cfg *RunConfig) {
		cfg.RuntimeContext = rtc
	}
}

// WithRuntimeValue adds a single key-value pair to the RuntimeContext.
// If no RuntimeContext exists, creates a new one.
func WithRuntimeValue(key string, value interface{}) RunOption {
	return func(cfg *RunConfig) {
		if cfg.RuntimeContext == nil {
			cfg.RuntimeContext = NewRuntimeContext()
		}
		cfg.RuntimeContext = cfg.RuntimeContext.With(key, value)
	}
}
```

### Step 1.4: Modify AsTool to extract and forward RuntimeContext values

Update the AsTool invoke function to extract RuntimeContext from context.Context and forward it to the sub-agent, excluding session-related keys.

Files:
* `go/chatagent/astool.go` - Modify invoke function

Success criteria:
* RuntimeContext extracted from context.Context
* Session-related keys excluded (session_id, conversation_id)
* RuntimeContext forwarded via RunOption
* Streaming callback still works with context forwarding

Context references:
* [python/packages/core/agent_framework/_agents.py](python/packages/core/agent_framework/_agents.py) (Lines 477-479) - kwargs forwarding with exclusions
* [go/chatagent/astool.go](go/chatagent/astool.go) (Lines 72-108) - Current invoke function

Dependencies:
* Step 1.3 completion

```go
// Modify the invoke function in AsTool:

// excludedKeys are keys that should not propagate to sub-agents
var excludedKeys = map[string]bool{
	"session_id":      true,
	"conversation_id": true,
	"thread_id":       true,
}

// Create a function that invokes the agent
fn := func(ctx context.Context, args json.RawMessage) (string, error) {
	// Parse args to get the task input
	var parsed map[string]string
	if err := json.Unmarshal(args, &parsed); err != nil {
		return "", fmt.Errorf("failed to parse arguments: %w", err)
	}

	input := parsed[argName]
	messages := []agent.Message{agent.NewUserMessage(input)}

	// Build run options with forwarded runtime context
	var runOpts []agent.RunOption
	
	if opts.ForwardRuntimeContext {
		// Extract runtime context from context.Context
		parentCtx := agent.RuntimeCtxFromContext(ctx)
		
		// Create filtered context excluding session-related keys
		filteredCtx := agent.NewRuntimeContext()
		for k, v := range parentCtx.Values() {
			if !excludedKeys[k] && !opts.excludeKey(k) {
				filteredCtx = filteredCtx.With(k, v)
			}
		}
		
		if filteredCtx.Len() > 0 {
			runOpts = append(runOpts, agent.WithRuntimeContext(filteredCtx))
		}
	}

	if opts.StreamCallback != nil {
		// Streaming mode
		updates, err := a.RunStream(ctx, messages, runOpts...)
		if err != nil {
			return "", err
		}

		var lastText string
		for update := range updates {
			opts.StreamCallback(update)
			if update.Delta != nil && update.Delta.TextDelta != "" {
				lastText += update.Delta.TextDelta
			}
		}

		return lastText, nil
	}

	// Non-streaming mode
	resp, err := a.Run(ctx, messages, runOpts...)
	if err != nil {
		return "", err
	}

	return resp.Text(), nil
}
```

### Step 1.5: Add unit tests for runtime context propagation

Create comprehensive tests for RuntimeContext type and AsTool context forwarding.

Files:
* `go/agent/runtime_context_test.go` - New test file
* `go/chatagent/astool_test.go` - Add context propagation tests

Success criteria:
* RuntimeContext creation, Get, With, Values tests pass
* Context extraction from context.Context works
* AsTool forwards runtime context to sub-agent
* Session keys are excluded from forwarding

Context references:
* [go/chatagent/astool_test.go](go/chatagent/astool_test.go) - Existing test patterns

Dependencies:
* Steps 1.1-1.4 completion

```go
// go/agent/runtime_context_test.go

// Copyright (c) Microsoft. All rights reserved.

package agent

import (
	"context"
	"testing"
)

func TestRuntimeContext_NewAndGet(t *testing.T) {
	rtc := NewRuntimeContext()
	
	_, ok := rtc.Get("missing")
	if ok {
		t.Error("Get should return false for missing key")
	}
	
	rtc2 := rtc.With("user_id", "u123")
	val, ok := rtc2.Get("user_id")
	if !ok || val != "u123" {
		t.Errorf("Get() = %v, %v; want u123, true", val, ok)
	}
	
	// Original unchanged
	_, ok = rtc.Get("user_id")
	if ok {
		t.Error("With should not modify original context")
	}
}

func TestRuntimeContext_Values(t *testing.T) {
	rtc := NewRuntimeContext().
		With("a", "1").
		With("b", "2")
	
	vals := rtc.Values()
	if len(vals) != 2 {
		t.Errorf("Values() len = %d; want 2", len(vals))
	}
	if vals["a"] != "1" || vals["b"] != "2" {
		t.Error("Values() content mismatch")
	}
}

func TestRuntimeCtxFromContext(t *testing.T) {
	ctx := context.Background()
	
	// No runtime context set
	rtc := RuntimeCtxFromContext(ctx)
	if rtc == nil || rtc.Len() != 0 {
		t.Error("Should return empty RuntimeContext when not set")
	}
	
	// With runtime context
	rtc2 := NewRuntimeContext().With("key", "value")
	ctx2 := WithRuntimeCtx(ctx, rtc2)
	
	extracted := RuntimeCtxFromContext(ctx2)
	val, ok := extracted.Get("key")
	if !ok || val != "value" {
		t.Error("Should extract RuntimeContext from context.Context")
	}
}

// go/chatagent/astool_test.go - Add test:

func TestAsTool_RuntimeContextPropagation(t *testing.T) {
	var capturedOpts []agent.RunOption
	
	mockAgent := &mockAgentForTool{
		name:     "ContextAgent",
		response: "Done",
		runInterceptor: func(opts []agent.RunOption) {
			capturedOpts = opts
		},
	}

	tool := AsTool(mockAgent, AsToolOptions{
		ForwardRuntimeContext: true,
	})

	// Create context with runtime values
	rtc := agent.NewRuntimeContext().
		With("user_id", "u123").
		With("api_token", "tok_abc").
		With("session_id", "should_be_excluded")
	
	ctx := agent.WithRuntimeCtx(context.Background(), rtc)

	args := json.RawMessage(`{"task": "test"}`)
	_, err := tool.Invoke(ctx, args)

	if err != nil {
		t.Errorf("Invoke() error = %v", err)
	}
	
	// Verify context was forwarded (excluding session_id)
	// Check capturedOpts contains WithRuntimeContext with user_id and api_token
	// but not session_id
}

func TestAsTool_SessionIDExclusion(t *testing.T) {
	// Test that session_id, conversation_id, thread_id are excluded
}
```

### Step 1.6: Validate phase changes

Run build and vet for agent and chatagent packages.

Validation commands:
* `go build ./agent/... ./chatagent/...`
* `go vet ./agent/... ./chatagent/...`

## Implementation Phase 2: Enhanced AsToolOptions

<!-- parallelizable: true -->

### Step 2.1: Add ApprovalMode field to AsToolOptions

Add ApprovalMode field to control whether the tool requires user approval before execution.

Files:
* `go/chatagent/astool.go` - Modify AsToolOptions struct

Success criteria:
* ApprovalMode field added to AsToolOptions
* Default value is ApprovalNever (matching Python's never_require)
* agentTool stores and returns approval mode

Context references:
* [go/tool/tool.go](go/tool/tool.go) (Lines 82-94) - ApprovalMode constants
* [python/packages/core/agent_framework/_agents.py](python/packages/core/agent_framework/_agents.py) (Line 505) - `approval_mode="never_require"`

Dependencies:
* None

```go
// AsToolOptions configures agent-to-tool conversion.
type AsToolOptions struct {
	// Name overrides the tool name (defaults to sanitized agent name).
	Name string

	// Description overrides the tool description (defaults to agent description).
	Description string

	// ArgName sets the parameter name for the task input (defaults to "task").
	ArgName string

	// ArgDescription describes the task parameter.
	ArgDescription string

	// StreamCallback receives streaming updates when set.
	// If nil, the tool uses non-streaming invocation.
	StreamCallback func(agent.ResponseUpdate)

	// ApprovalMode specifies when user approval is required.
	// Defaults to ApprovalNever.
	ApprovalMode tool.ApprovalMode

	// ForwardRuntimeContext enables runtime context propagation to sub-agent.
	// When true, RuntimeContext values are forwarded (excluding session keys).
	ForwardRuntimeContext bool

	// ExcludeKeys specifies additional keys to exclude from context forwarding.
	ExcludeKeys []string
}

// excludeKey checks if a key should be excluded from forwarding.
func (o *AsToolOptions) excludeKey(key string) bool {
	for _, k := range o.ExcludeKeys {
		if k == key {
			return true
		}
	}
	return false
}
```

### Step 2.2: Add ForwardRuntimeContext bool field to control propagation

Already included in Step 2.1 above. This step verifies the integration.

Files:
* `go/chatagent/astool.go` - Field already added in 2.1

Success criteria:
* ForwardRuntimeContext field controls whether context propagates
* Default is false (opt-in to prevent unintended propagation)

Dependencies:
* Step 2.1 completion

### Step 2.3: Add ExcludeSessionID bool field to prevent session bleeding

Replace with ExcludeKeys for more flexibility (already in Step 2.1).

Files:
* `go/chatagent/astool.go` - Field already added in 2.1

Success criteria:
* ExcludeKeys allows custom exclusion patterns
* Default excludedKeys always apply (session_id, conversation_id, thread_id)

Dependencies:
* Step 2.1 completion

### Step 2.4: Update AsTool to apply new options

Modify AsTool function to use ApprovalMode when creating the agentTool.

Files:
* `go/chatagent/astool.go` - Update AsTool function

Success criteria:
* ApprovalMode from options applied to agentTool
* agentTool implements ApprovalMode getter if needed

Context references:
* [go/chatagent/astool.go](go/chatagent/astool.go) (Lines 112-130) - agentTool struct

Dependencies:
* Step 2.1 completion

```go
// Update agentTool struct:
type agentTool struct {
	name         string
	description  string
	parameters   json.RawMessage
	invoke       func(ctx context.Context, args json.RawMessage) (string, error)
	approvalMode tool.ApprovalMode
}

// GetApprovalMode returns the tool's approval mode.
func (t *agentTool) GetApprovalMode() tool.ApprovalMode {
	return t.approvalMode
}

// Update AsTool return:
return &agentTool{
	name:         name,
	description:  desc,
	parameters:   paramsJSON,
	invoke:       fn,
	approvalMode: opts.ApprovalMode,
}
```

### Step 2.5: Add unit tests for new options

Add tests for ApprovalMode, ForwardRuntimeContext, and ExcludeKeys.

Files:
* `go/chatagent/astool_test.go` - Add new test cases

Success criteria:
* ApprovalMode is correctly set on tool
* ForwardRuntimeContext enables/disables propagation
* ExcludeKeys prevents specific keys from forwarding

Dependencies:
* Steps 2.1-2.4 completion

```go
func TestAsTool_ApprovalMode(t *testing.T) {
	mockAgent := &mockAgentForTool{name: "ApprovalAgent"}

	// Default approval mode
	tool1 := AsTool(mockAgent, AsToolOptions{})
	if tool1.(*agentTool).approvalMode != "" {
		t.Error("Default ApprovalMode should be empty (never)")
	}

	// Explicit approval mode
	tool2 := AsTool(mockAgent, AsToolOptions{
		ApprovalMode: tool.ApprovalAlways,
	})
	if tool2.(*agentTool).approvalMode != tool.ApprovalAlways {
		t.Error("ApprovalMode should be set to always")
	}
}

func TestAsTool_ExcludeKeys(t *testing.T) {
	// Test that custom exclude keys work
	mockAgent := &mockAgentForTool{name: "ExcludeAgent", response: "Done"}

	tool := AsTool(mockAgent, AsToolOptions{
		ForwardRuntimeContext: true,
		ExcludeKeys:           []string{"secret_key"},
	})

	rtc := agent.NewRuntimeContext().
		With("user_id", "u123").
		With("secret_key", "should_be_excluded")
	
	ctx := agent.WithRuntimeCtx(context.Background(), rtc)

	// Invoke and verify secret_key was excluded
	_, _ = tool.Invoke(ctx, json.RawMessage(`{"task": "test"}`))
	// Verification through mock interception
}
```

## Implementation Phase 3: Name Sanitization Enhancement

<!-- parallelizable: true -->

### Step 3.1: Update sanitizeAgentName to handle digit prefixes

Modify sanitizeAgentName to prefix names starting with digits with underscore, matching Python behavior.

Files:
* `go/chatagent/astool.go` - Modify sanitizeAgentName function

Success criteria:
* Names starting with digits are prefixed with underscore
* Existing behavior preserved for other cases
* Empty names still return "agent"

Context references:
* [go/chatagent/astool.go](go/chatagent/astool.go) (Lines 144-167) - Current sanitizeAgentName

Dependencies:
* None

```go
// sanitizeAgentName converts an agent name to a valid tool identifier.
func sanitizeAgentName(name string) string {
	// Replace spaces and special chars with underscores
	re := regexp.MustCompile(`[^a-zA-Z0-9_]`)
	sanitized := re.ReplaceAllString(name, "_")

	// Remove consecutive underscores
	sanitized = regexp.MustCompile(`_+`).ReplaceAllString(sanitized, "_")

	// Trim underscores from ends
	sanitized = strings.Trim(sanitized, "_")

	// Ensure lowercase
	sanitized = strings.ToLower(sanitized)

	// Handle empty result
	if sanitized == "" {
		return "agent"
	}

	// Prefix with underscore if starts with digit
	if len(sanitized) > 0 && sanitized[0] >= '0' && sanitized[0] <= '9' {
		sanitized = "_" + sanitized
	}

	return sanitized
}
```

### Step 3.2: Add option for case preservation (optional enhancement)

Add optional flag to preserve original casing if needed for specific use cases.

Files:
* `go/chatagent/astool.go` - Add PreserveCase to AsToolOptions

Success criteria:
* PreserveCase option skips lowercase conversion when true
* Default behavior (lowercase) unchanged

Dependencies:
* None

```go
// In AsToolOptions:
// PreserveCase keeps the original casing when sanitizing the agent name.
// Default is false (names are lowercased).
PreserveCase bool

// In sanitizeAgentName, add parameter:
func sanitizeAgentName(name string, preserveCase bool) string {
	// ... existing logic ...
	
	// Ensure lowercase (unless preserving case)
	if !preserveCase {
		sanitized = strings.ToLower(sanitized)
	}
	
	// ... rest of logic ...
}
```

### Step 3.3: Add comprehensive sanitization tests

Extend tests to cover all edge cases including digit prefixes.

Files:
* `go/chatagent/astool_test.go` - Extend TestSanitizeAgentName

Success criteria:
* Digit-prefix handling tested
* All special character replacements tested
* Empty and underscore-only names tested

Dependencies:
* Steps 3.1-3.2 completion

```go
func TestSanitizeAgentName_Comprehensive(t *testing.T) {
	tests := []struct {
		input       string
		wantName    string
		description string
	}{
		{"Research Agent", "research_agent", "spaces to underscores"},
		{"code-helper", "code_helper", "hyphens to underscores"},
		{"My Super Agent!", "my_super_agent", "special chars removed"},
		{"", "agent", "empty string"},
		{"___", "agent", "only underscores"},
		{"Agent123", "agent123", "trailing digits"},
		{"123Agent", "_123agent", "leading digit gets prefix"},
		{"1st Agent", "_1st_agent", "leading digit with spaces"},
		{"UPPERCASE", "uppercase", "case normalization"},
		{"a__b__c", "a_b_c", "consecutive underscores collapsed"},
		{"_leading", "leading", "leading underscore trimmed"},
		{"trailing_", "trailing", "trailing underscore trimmed"},
	}

	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			got := sanitizeAgentName(tt.input)
			if got != tt.wantName {
				t.Errorf("sanitizeAgentName(%q) = %q, want %q", tt.input, got, tt.wantName)
			}
		})
	}
}
```

## Implementation Phase 4: Documentation and Examples

<!-- parallelizable: true -->

### Step 4.1: Update chatagent/doc.go with AsTool usage examples

Add comprehensive AsTool documentation to the package documentation.

Files:
* `go/chatagent/doc.go` - Add AsTool examples section

Success criteria:
* Basic AsTool usage documented
* Custom options documented
* Streaming callback example included

Context references:
* [go/chatagent/doc.go](go/chatagent/doc.go) - Existing package documentation

Dependencies:
* None

```go
// Add to doc.go:

/*
Agent as Tool

The AsTool function converts an agent to a tool for use by other agents,
enabling hierarchical agent patterns:

	// Create a specialized research agent
	researcher := chatagent.New(researchClient,
		chatagent.WithName("Researcher"),
		chatagent.WithDescription("Performs in-depth research on topics"),
		chatagent.WithInstructions("You are a research specialist..."),
	)

	// Convert to tool with custom options
	researchTool := chatagent.AsTool(researcher, chatagent.AsToolOptions{
		Name:           "research",
		Description:    "Research a topic thoroughly",
		ArgName:        "topic",
		ArgDescription: "The topic to research",
	})

	// Use in an orchestrator agent
	orchestrator := chatagent.New(client,
		chatagent.WithName("Orchestrator"),
		chatagent.WithTools(researchTool),
	)

Streaming Support

Use StreamCallback to receive incremental updates from the sub-agent:

	researchTool := chatagent.AsTool(researcher, chatagent.AsToolOptions{
		StreamCallback: func(update agent.ResponseUpdate) {
			if update.Delta != nil && update.Delta.TextDelta != "" {
				fmt.Print(update.Delta.TextDelta)
			}
		},
	})
*/
```

### Step 4.2: Add hierarchical agent example in chatagent/doc.go

Add a complete multi-agent orchestration example.

Files:
* `go/chatagent/doc.go` - Add hierarchical example

Success criteria:
* Complete working example with multiple agents
* Shows context propagation
* Demonstrates orchestrator pattern

Dependencies:
* Step 4.1 completion

```go
/*
Hierarchical Agent Orchestration

Complex tasks can be broken down using multiple specialized agents:

	// Create specialized agents
	researcher := chatagent.New(client,
		chatagent.WithName("Researcher"),
		chatagent.WithInstructions("You research topics thoroughly."),
	)

	writer := chatagent.New(client,
		chatagent.WithName("Writer"),
		chatagent.WithInstructions("You write clear, engaging content."),
	)

	coder := chatagent.New(client,
		chatagent.WithName("Coder"),
		chatagent.WithInstructions("You write clean, tested code."),
	)

	// Convert agents to tools with context forwarding
	tools := []tool.Tool{
		chatagent.AsTool(researcher, chatagent.AsToolOptions{
			Name:                  "research",
			Description:           "Research a topic",
			ForwardRuntimeContext: true,
		}),
		chatagent.AsTool(writer, chatagent.AsToolOptions{
			Name:                  "write",
			Description:           "Write content based on research",
			ForwardRuntimeContext: true,
		}),
		chatagent.AsTool(coder, chatagent.AsToolOptions{
			Name:                  "code",
			Description:           "Implement code solutions",
			ForwardRuntimeContext: true,
		}),
	}

	// Create orchestrator that uses specialized agents
	orchestrator := chatagent.New(client,
		chatagent.WithName("Orchestrator"),
		chatagent.WithInstructions(`You coordinate complex tasks by delegating to:
- research: for gathering information
- write: for creating content
- code: for implementing solutions`),
		chatagent.WithTools(tools...),
	)

	// Run with runtime context that propagates to all sub-agents
	ctx := agent.WithRuntimeCtx(context.Background(),
		agent.NewRuntimeContext().
			With("user_id", "u123").
			With("api_token", "tok_abc"),
	)

	resp, err := orchestrator.Run(ctx, messages)
*/
```

### Step 4.3: Update README with multi-agent orchestration section

Add a section to the Go README about multi-agent orchestration.

Files:
* `go/README.md` - Add orchestration section

Success criteria:
* Clear explanation of AsTool pattern
* Link to chatagent documentation
* Brief code example

Dependencies:
* None

```markdown
## Multi-Agent Orchestration

The framework supports hierarchical agent patterns where one agent can use other agents as tools:

```go
// Create a specialized agent
researcher := chatagent.New(client,
    chatagent.WithName("Researcher"),
)

// Convert to tool for use by other agents
researchTool := chatagent.AsTool(researcher, chatagent.AsToolOptions{
    Name:        "research",
    Description: "Research a topic thoroughly",
})

// Use in orchestrator
orchestrator := chatagent.New(client,
    chatagent.WithTools(researchTool),
)
```

See the [chatagent package documentation](./chatagent/doc.go) for complete examples.
```

## Implementation Phase 5: Validation

<!-- parallelizable: false -->

### Step 5.1: Run full project validation

Execute all validation commands for the project:
* `go build ./...`
* `go vet ./...`
* `go test ./...`

### Step 5.2: Verify test coverage meets 90% target for astool.go

Run coverage analysis specifically for astool.go:
* `go test -coverprofile=coverage.out ./chatagent/...`
* `go tool cover -func=coverage.out | grep astool`

Expected coverage: 90%+

### Step 5.3: Fix minor validation issues

Iterate on lint errors, build warnings, and test failures. Apply fixes directly when corrections are straightforward and isolated.

### Step 5.4: Report blocking issues

When validation failures require changes beyond minor fixes:
* Document the issues and affected files.
* Provide the user with next steps.
* Recommend additional research and planning rather than inline fixes.
* Avoid large-scale refactoring within this phase.

## Dependencies

* Go 1.21+ (context values, generics)
* Existing agent package (Agent, RunOption, RunConfig, Message)
* Existing tool package (Tool, ApprovalMode, Result)

## Success Criteria

* All runtime context propagation tests pass
* ApprovalMode configurable on agent tools
* Session keys excluded from sub-agent calls
* Name sanitization handles all edge cases
* Test coverage for astool.go ≥ 90%
* Documentation includes complete examples
