<!-- markdownlint-disable-file -->
# Release Changes: TextSearchProvider Memory Options

**Related Plan**: N/A (Enhancement from review findings)
**Related Review**: 2026-02-04-go-epic3-dotnet-python-comparison-review.md
**Implementation Date**: 2026-02-04

## Summary

Added two missing configuration options to the Go TextSearchProvider identified during the Epic 3 comparison review. These options provide parity with the .NET implementation for memory management.

## Changes

### Added

* go/provider/textsearch/options.go - Added `RecentMessageMemoryLimit` option to limit memory size
* go/provider/textsearch/options.go - Added `RecentMessageRolesIncluded` option to filter message roles
* go/provider/textsearch/options.go - Added `WithRecentMessageMemoryLimit` functional option
* go/provider/textsearch/options.go - Added `WithRecentMessageRolesIncluded` functional option
* go/provider/textsearch/options.go - Added `DefaultRecentMessageRolesIncluded` constant
* go/provider/textsearch/provider.go - Added `isRoleIncluded` helper method

### Modified

* go/provider/textsearch/options.go - Updated `defaultOptions()` to include new memory options
* go/provider/textsearch/provider.go - Updated `Invoked` method to respect memory limit and role filtering
* go/provider/textsearch/provider.go - Updated `Invoked` to skip memory update on invocation errors
* go/provider/textsearch/provider_test.go - Updated existing tests to use memory limit option
* go/provider/textsearch/provider_test.go - Added 9 new tests for memory options
* go/provider/textsearch/integration_test.go - Updated integration tests to use memory options

## Additional or Deviating Changes

* Memory is now disabled by default (RecentMessageMemoryLimit = 0)
  * Reason: Aligns with .NET behavior where memory is opt-in
  * Existing tests updated to explicitly enable memory when needed

## Feature Details

### RecentMessageMemoryLimit

| Property | Value |
|----------|-------|
| Type | `int` |
| Default | `0` (disabled) |
| Purpose | Limits the number of recent messages retained in memory for context building |
| Behavior when 0 | Memory disabled, only current request messages used for search |

### RecentMessageRolesIncluded

| Property | Value |
|----------|-------|
| Type | `[]string` |
| Default | `["user"]` |
| Purpose | Filters which message roles are included in memory |
| Valid values | `chat.RoleUser`, `chat.RoleAssistant`, `chat.RoleSystem`, `chat.RoleTool` |

### Usage Examples

```go
// Enable memory with limit of 10 messages, user messages only
provider := textsearch.New(searchFn,
    textsearch.WithRecentMessageMemoryLimit(10),
)

// Enable memory with both user and assistant messages
provider := textsearch.New(searchFn,
    textsearch.WithRecentMessageMemoryLimit(20),
    textsearch.WithRecentMessageRolesIncluded(
        string(chat.RoleUser),
        string(chat.RoleAssistant),
    ),
)
```

## Release Summary

| Metric | Value |
|--------|-------|
| Files Modified | 4 |
| Files Added | 0 |
| Tests Added | 9 |
| Coverage | 95.1% |
| Build Status | Passing |
| Test Status | All passing |

### Test Results

```
ok      github.com/microsoft/agent-framework-go/provider/textsearch   6.532s   coverage: 95.1% of statements
```

### Feature Parity Update

After this change, the TextSearchProvider feature comparison with .NET is complete:

| Feature | Go | .NET |
|---------|:--:|:----:|
| RecentMessageMemoryLimit | ✅ | ✅ |
| RecentMessageRolesIncluded | ✅ | ✅ |
