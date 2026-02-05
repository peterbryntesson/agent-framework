<!-- markdownlint-disable-file -->
# Release Changes: Go Epic 5 Phase 3 - Declarative Agents

**Related Plan**: 2026-02-04-go-epic5-implementation-plan.md
**Implementation Date**: 2026-02-04

## Summary

Implemented Feature 5.2: Declarative Agent Definitions for the Go Agent Framework. This enables agents to be defined in YAML files and loaded dynamically at runtime. The implementation includes YAML model definitions, a loader with validation, environment variable evaluation (PowerFx-style =Env.VAR_NAME syntax), an agent factory with provider and tool parsing, and comprehensive unit tests achieving 69.3% coverage.

## Changes

### Added

* go/declarative/doc.go - Package documentation with usage examples and YAML schema overview
* go/declarative/models.go - YAML model types (PromptAgent, Model, Connection, ModelOptions, Tool, ParameterSchema, PropertySchema, OutputSchema) with helper methods
* go/declarative/models_test.go - Unit tests for model types and helper methods
* go/declarative/loader.go - YAML loader functions (LoadFromFile, LoadFromReader, LoadFromString, LoadFromBytes)
* go/declarative/loader_test.go - Unit tests for YAML loading functionality
* go/declarative/validation.go - Validation logic with ValidationError type and field validation
* go/declarative/validation_test.go - Unit tests for validation logic
* go/declarative/eval.go - Environment variable evaluation (TryEvalEnv, EvalEnvOrDefault, evaluateAgent)
* go/declarative/eval_test.go - Unit tests for environment variable evaluation
* go/declarative/factory.go - AgentFactory with ProviderBuilder and ToolParser types, Create/CreateFromFile/CreateFromString methods
* go/declarative/factory_test.go - Unit tests for factory functionality
* go/declarative/providers.go - Default provider builders for OpenAI and Azure OpenAI, resolveAPIKey helper
* go/declarative/providers_test.go - Unit tests for provider builders
* go/declarative/tools.go - Tool parsing (parseFunction, declarativeTool, buildParameterSchema, BindToolHandler)
* go/declarative/tools_test.go - Unit tests for tool parsing

### Modified

* None

### Removed

* None

## Additional or Deviating Changes

* MCP tool parsing (Task 5.2.7.3) deferred to Phase 4 - MCP Integration
  * Reason: MCP package not yet implemented; will be completed in Phase 4
* Hosted tool parsing (Task 5.2.7.4) deferred
  * Reason: Requires additional provider-specific hosted tool support; will be added in later phases
* Azure OpenAI uses OpenAI client with endpoint configuration instead of dedicated Azure client
  * Reason: No dedicated Azure OpenAI package exists yet; existing OpenAI package supports custom endpoints

## Validation Results

```
go build ./declarative/...  - PASS
go vet ./declarative/...    - PASS
go test ./declarative/... -cover
  44 tests passing
  coverage: 69.3% of statements
```

## Release Summary

**Total Files Affected:** 14 files created

**Files Created:**
- go/declarative/doc.go - Package documentation
- go/declarative/models.go - YAML model types
- go/declarative/models_test.go - Model unit tests
- go/declarative/loader.go - YAML loader functions
- go/declarative/loader_test.go - Loader unit tests
- go/declarative/validation.go - Validation logic
- go/declarative/validation_test.go - Validation unit tests
- go/declarative/eval.go - Environment variable evaluation
- go/declarative/eval_test.go - Eval unit tests
- go/declarative/factory.go - Agent factory
- go/declarative/factory_test.go - Factory unit tests
- go/declarative/providers.go - Provider builders
- go/declarative/providers_test.go - Provider unit tests
- go/declarative/tools.go - Tool parsing
- go/declarative/tools_test.go - Tool parsing unit tests

**Dependencies:**
- gopkg.in/yaml.v3 (already indirect dependency, promoted to direct use)

**Key Features:**
- YAML schema compatible with .NET and Python implementations
- PowerFx-style environment variable substitution (=Env.VAR_NAME)
- Extensible provider and tool parser registration
- Built-in support for OpenAI and Azure OpenAI providers
- Function tools with declarative parameter schemas
- Validation with detailed error reporting

**Test Coverage:** 69.3% of statements covered
