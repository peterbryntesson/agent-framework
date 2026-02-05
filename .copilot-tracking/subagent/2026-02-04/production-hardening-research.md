# Production Hardening Patterns Analysis

## Research Date: 2026-02-04

## Executive Summary

This document analyzes resilience patterns (rate limiting, circuit breaker, retry, connection pooling) across the .NET, Python, and Go implementations of the Agent Framework. The analysis reveals that while basic resilience infrastructure exists, there are opportunities to implement more comprehensive production hardening patterns.

---

## 1. Rate Limiting Patterns

### 1.1 .NET Implementation

**Status**: Basic HTTP 429 handling exists, no built-in rate limiter.

| File | Line | Pattern |
|------|------|---------|
| [PurviewClient.cs](dotnet/src/Microsoft.Agents.AI.Purview/PurviewClient.cs#L38) | 38 | HTTP 429 status detection |
| [PurviewRateLimitException.cs](dotnet/src/Microsoft.Agents.AI.Purview/Exceptions/PurviewRateLimitException.cs#L10) | 10-27 | Rate limit exception class |

**Code Pattern**:
```csharp
// PurviewClient.cs - Rate limit detection
switch ((int)statusCode)
{
    case 429:
        return new PurviewRateLimitException($"Rate limit exceeded for {endpointName}.");
    // ...
}
```

**Observations**:
- Rate limit detection via HTTP status code 429
- Custom exception hierarchy for error handling
- No built-in token bucket or sliding window implementation
- No automatic retry-after header parsing

### 1.2 Python Implementation

**Status**: Basic HTTP 429 handling exists, no built-in rate limiter.

| File | Line | Pattern |
|------|------|---------|
| [_exceptions.py](python/packages/purview/agent_framework_purview/_exceptions.py#L27) | 27 | `PurviewRateLimitError` class |
| [_client.py](python/packages/purview/agent_framework_purview/_client.py#L172) | 172 | HTTP 429 detection and exception raising |

**Code Pattern**:
```python
# _client.py - Rate limit detection
if resp.status_code == 429:
    raise PurviewRateLimitError(f"Rate limited {resp.status_code}: {resp.text}")
```

**Observations**:
- Matches .NET exception structure
- No built-in token bucket or sliding window implementation
- No automatic retry logic for rate limits

### 1.3 Go Implementation

**Status**: Sentinel error defined, retry detection built-in.

| File | Line | Pattern |
|------|------|---------|
| [errors.go](go/agent/errors.go#L20) | 20 | `ErrRateLimited` sentinel error |
| [errors.go](go/agent/errors.go#L65-L86) | 65-86 | `IsRetryable()` function |

**Code Pattern**:
```go
// ErrRateLimited indicates the agent or provider has throttled requests due to rate limits.
ErrRateLimited = errors.New("rate limited")

// IsRetryable returns true if the error represents a condition that may succeed on retry.
func IsRetryable(err error) bool {
    if errors.Is(err, ErrRateLimited) {
        return true
    }
    // ...
}
```

**Observations**:
- Idiomatic Go sentinel errors
- Built-in `IsRetryable()` helper function
- Ready for integration with retry middleware

### 1.4 Go Equivalent Recommendations

For implementing rate limiting in Go:

```go
// Recommended packages:
// - golang.org/x/time/rate (standard library token bucket)
// - github.com/uber-go/ratelimit (leaky bucket)

// Token bucket pattern
import "golang.org/x/time/rate"

type RateLimitedClient struct {
    limiter *rate.Limiter
    inner   ChatClient
}

func (c *RateLimitedClient) GetResponse(ctx context.Context, messages []Message) (*Response, error) {
    if err := c.limiter.Wait(ctx); err != nil {
        return nil, fmt.Errorf("rate limit: %w", err)
    }
    return c.inner.GetResponse(ctx, messages)
}
```

---

## 2. Circuit Breaker Patterns

### 2.1 .NET Implementation

**Status**: External library available via `Microsoft.Extensions.Http.Resilience`.

| File | Line | Pattern |
|------|------|---------|
| [Directory.Packages.props](dotnet/Directory.Packages.props#L76) | 76 | `Microsoft.Extensions.Http.Resilience` v10.0.0 |
| [ServiceDefaultsExtensions.cs](dotnet/samples/AgentWebChat/AgentWebChat.ServiceDefaults/ServiceDefaultsExtensions.cs#L31) | 31 | `AddStandardResilienceHandler()` |

**Code Pattern**:
```csharp
// ServiceDefaultsExtensions.cs - Adding resilience to HTTP clients
builder.Services.ConfigureHttpClientDefaults(http =>
{
    // Turn on resilience by default
    http.AddStandardResilienceHandler();
});
```

**Observations**:
- Uses Polly-based resilience pipeline from Microsoft.Extensions.Http.Resilience
- Standard resilience handler includes: retry, circuit breaker, timeout policies
- Only demonstrated in sample code, not in core library

### 2.2 Python Implementation

**Status**: No circuit breaker implementation found.

**Observations**:
- No usage of `pybreaker` or similar libraries
- No state machine implementation (Closed/Open/HalfOpen)
- Opportunity for middleware-based circuit breaker

### 2.3 Go Implementation

**Status**: No circuit breaker implementation found, but backoff library available.

| File | Line | Pattern |
|------|------|---------|
| [go.mod](go/go.mod#L19) | 19 | `github.com/cenkalti/backoff/v5` (indirect dependency) |

**Observations**:
- Backoff library available for retry logic
- No circuit breaker implementation found
- Could use `github.com/sony/gobreaker` for circuit breaker

### 2.4 Go Equivalent Recommendations

```go
// Recommended package: github.com/sony/gobreaker

import "github.com/sony/gobreaker"

type CircuitBreakerClient struct {
    cb    *gobreaker.CircuitBreaker
    inner ChatClient
}

func NewCircuitBreakerClient(inner ChatClient) *CircuitBreakerClient {
    settings := gobreaker.Settings{
        Name:        "agent-client",
        MaxRequests: 3,                    // Half-open max requests
        Interval:    10 * time.Second,     // Closed state interval
        Timeout:     30 * time.Second,     // Open state timeout
        ReadyToTrip: func(counts gobreaker.Counts) bool {
            return counts.ConsecutiveFailures > 5
        },
    }
    return &CircuitBreakerClient{
        cb:    gobreaker.NewCircuitBreaker(settings),
        inner: inner,
    }
}
```

---

## 3. Retry Policies

### 3.1 .NET Implementation

**Status**: Exponential backoff implemented for polling; Polly-based retry in samples.

| File | Line | Pattern |
|------|------|---------|
| [AgentRunHandle.cs](dotnet/src/Microsoft.Agents.AI.DurableTask/AgentRunHandle.cs#L42) | 42 | Exponential backoff documentation |
| [AgentRunHandle.cs](dotnet/src/Microsoft.Agents.AI.DurableTask/AgentRunHandle.cs#L76) | 76 | Exponential backoff polling |
| [CosmosChatHistoryProvider.cs](dotnet/src/Microsoft.Agents.AI.CosmosNoSql/CosmosChatHistoryProvider.cs#L464) | 464 | Split-and-retry pattern |

**Code Pattern**:
```csharp
// AgentRunHandle.cs - Exponential backoff for polling
TimeSpan pollInterval = TimeSpan.FromMilliseconds(50); // Start with 50ms
TimeSpan maxPollInterval = TimeSpan.FromSeconds(3);    // Maximum 3 seconds

while (true)
{
    // ... poll logic ...
    
    // Wait before polling again with exponential backoff
    await Task.Delay(pollInterval, cancellationToken);
    
    // Double the poll interval, but cap it at the maximum
    pollInterval = TimeSpan.FromMilliseconds(
        Math.Min(pollInterval.TotalMilliseconds * 2, maxPollInterval.TotalMilliseconds));
}
```

### 3.2 Python Implementation

**Status**: Basic retry loop with linear backoff in Magentic workflow.

| File | Line | Pattern |
|------|------|---------|
| [_magentic.py](python/packages/core/agent_framework/_workflows/_magentic.py#L703-L717) | 703-717 | Retry loop with backoff |

**Code Pattern**:
```python
# _magentic.py - Retry loop with linear backoff
attempts = 0
last_error: Exception | None = None
while attempts < self.progress_ledger_retry_count:
    raw = await self._complete([*magentic_context.chat_history, user_message])
    try:
        ledger_dict = _extract_json(raw.text)
        return _coerce_model(MagenticProgressLedger, ledger_dict)
    except Exception as ex:
        last_error = ex
        attempts += 1
        if attempts < self.progress_ledger_retry_count:
            # brief backoff before next try
            await asyncio.sleep(0.25 * attempts)  # Linear backoff

raise RuntimeError(f"Parse failed after {attempts} attempt(s)")
```

**Observations**:
- Uses linear backoff (0.25 * attempts seconds)
- No jitter implementation
- No tenacity library usage found

### 3.3 Go Implementation

**Status**: Tool invocation config with max iterations and consecutive error limits.

| File | Line | Pattern |
|------|------|---------|
| [config.go](go/tool/config.go#L27-L32) | 27-32 | MaxIterations, MaxConsecutiveErrors |
| [invoke.go](go/tool/invoke.go#L87-91) | 87-91 | Timeout handling |

**Code Pattern**:
```go
// config.go - Invocation configuration
type InvocationConfig struct {
    MaxIterations        int  // Limits tool invocation rounds
    MaxConsecutiveErrors int  // Stops after consecutive errors
    TimeoutSeconds       int  // Per-invocation timeout
    // ...
}

// Default values (matches Python)
const (
    DefaultMaxIterations        = 40
    DefaultMaxConsecutiveErrors = 3
)
```

### 3.4 Go Equivalent Recommendations

```go
// Using github.com/cenkalti/backoff/v5 (already in go.mod)
import "github.com/cenkalti/backoff/v5"

func RetryWithBackoff(ctx context.Context, operation func() error) error {
    b := backoff.NewExponentialBackOff()
    b.MaxElapsedTime = 2 * time.Minute
    b.InitialInterval = 100 * time.Millisecond
    b.MaxInterval = 30 * time.Second
    b.Multiplier = 2.0
    b.RandomizationFactor = 0.5  // Jitter
    
    return backoff.Retry(operation, backoff.WithContext(b, ctx))
}
```

---

## 4. Connection Pooling

### 4.1 .NET Implementation

**Status**: HttpClient injection pattern used; relies on .NET's connection pooling.

| File | Line | Pattern |
|------|------|---------|
| [PurviewClient.cs](dotnet/src/Microsoft.Agents.AI.Purview/PurviewClient.cs#L27) | 27 | HttpClient field |
| [PurviewClient.cs](dotnet/src/Microsoft.Agents.AI.Purview/PurviewClient.cs#L56) | 56 | Constructor injection |
| [PurviewExtensions.cs](dotnet/src/Microsoft.Agents.AI.Purview/PurviewExtensions.cs#L39) | 39 | `AddSingleton<HttpClient>()` |

**Observations**:
- Uses injected HttpClient (recommended for .NET connection pooling)
- No explicit `SocketsHttpHandler` configuration found
- Relies on .NET's default connection management

### 4.2 Python Implementation

**Status**: httpx.AsyncClient with configurable timeout; reusable client pattern.

| File | Line | Pattern |
|------|------|---------|
| [_client.py](python/packages/purview/agent_framework_purview/_client.py#L55) | 55 | `httpx.AsyncClient(timeout=timeout)` |
| [_agent.py](python/packages/a2a/agent_framework_a2a/_agent.py#L105-113) | 105-113 | Detailed timeout configuration |

**Code Pattern**:
```python
# _agent.py - Detailed timeout configuration
def _create_timeout_config(self, timeout: float | httpx.Timeout | None) -> httpx.Timeout:
    if timeout is None:
        return httpx.Timeout(
            connect=10.0,   # 10 seconds to establish connection
            read=60.0,      # 60 seconds to read response
            write=10.0,     # 10 seconds to send request
            pool=5.0,       # 5 seconds to get connection from pool
        )
```

**Observations**:
- Uses httpx.AsyncClient for connection pooling
- Configurable pool timeout
- Context manager pattern for cleanup (`__aenter__`, `__aexit__`)

### 4.3 Go Implementation

**Status**: Custom http.Client injection pattern; configurable timeouts.

| File | Line | Pattern |
|------|------|---------|
| [client.go](go/providers/openai/client.go#L268) | 268 | `NewClientWithHTTPClient` factory |
| [client.go](go/providers/openai/client.go#L276) | 276 | Default `http.Client` creation |
| [client.go](go/protocol/a2a/client.go#L68) | 68 | A2A client with timeout |
| [options.go](go/providers/openai/options.go#L94) | 94 | `WithHTTPClient` option |

**Code Pattern**:
```go
// client.go - HTTP client injection
func NewClientWithHTTPClient(httpClient *http.Client, opts ...Option) (*Client, error) {
    opts = append([]Option{WithHTTPClient(httpClient)}, opts...)
    return NewClient(opts...)
}

// Default client with timeout
httpClient := &http.Client{
    Timeout: options.timeout,
}
```

**Observations**:
- Follows Go idiom of injecting http.Client
- Default timeout configuration available
- No custom Transport configuration (e.g., MaxIdleConns, MaxIdleConnsPerHost)

### 4.4 Go Equivalent Recommendations

```go
// Recommended Transport configuration for production
transport := &http.Transport{
    MaxIdleConns:        100,
    MaxIdleConnsPerHost: 10,
    MaxConnsPerHost:     100,
    IdleConnTimeout:     90 * time.Second,
    TLSHandshakeTimeout: 10 * time.Second,
    ExpectContinueTimeout: 1 * time.Second,
    ForceAttemptHTTP2:   true,
}

client := &http.Client{
    Transport: transport,
    Timeout:   30 * time.Second,
}
```

---

## 5. Semaphore/Concurrency Control

### 5.1 .NET Implementation

| File | Line | Pattern |
|------|------|---------|
| [MemoryCacheExtensions.cs](dotnet/src/Microsoft.Agents.AI.Hosting.OpenAI/MemoryCacheExtensions.cs#L22) | 22 | Atomic cache operations with semaphore |
| [InputWaiter.cs](dotnet/src/Microsoft.Agents.AI.Workflows/Execution/InputWaiter.cs#L11) | 11 | Input signal semaphore |
| [ChatHistoryMemoryProvider.cs](dotnet/src/Microsoft.Agents.AI/Memory/ChatHistoryMemoryProvider.cs#L58) | 58 | Initialization lock |

**Code Pattern**:
```csharp
// MemoryCacheExtensions.cs - Atomic cache operations
private static readonly ConcurrentDictionary<(IMemoryCache, object), SemaphoreSlim> s_semaphores = new();

await semaphore.WaitAsync(cancellationToken).ConfigureAwait(false);
try
{
    // Critical section
}
finally
{
    semaphore.Release();
}
```

### 5.2 Python Implementation

| File | Line | Pattern |
|------|------|---------|
| [gaia.py](python/packages/lab/gaia/agent_framework_lab_gaia/gaia.py#L523) | 523 | `asyncio.Semaphore(parallel)` |

**Code Pattern**:
```python
# gaia.py - Parallel task execution with semaphore
semaphore = asyncio.Semaphore(parallel)
tasks_coroutines = [
    self._run_single_task(task, task_runner, semaphore, timeout) 
    for task in tasks
]

async def _run_single_task(self, task, task_runner, semaphore, timeout):
    async with semaphore:
        # Execute task with concurrency limit
```

---

## 6. Timeout Configuration

### 6.1 .NET Implementation

**Pattern**: CancellationToken throughout APIs.

- All async methods accept `CancellationToken`
- Uses `ConfigureAwait(false)` consistently
- No explicit timeout configuration classes found

### 6.2 Python Implementation

**Pattern**: Configurable timeouts via constructor.

| File | Line | Pattern |
|------|------|---------|
| [_client.py](python/packages/purview/agent_framework_purview/_client.py#L49) | 49 | `timeout: float | None = 10.0` |
| [_agent.py](python/packages/a2a/agent_framework_a2a/_agent.py#L82) | 82 | `timeout: float | httpx.Timeout | None` |
| [gaia.py](python/packages/lab/gaia/agent_framework_lab_gaia/gaia.py#L407) | 407 | `asyncio.wait_for(task, timeout=timeout)` |

### 6.3 Go Implementation

**Pattern**: Context-based timeouts.

| File | Line | Pattern |
|------|------|---------|
| [invoke.go](go/tool/invoke.go#L89) | 89 | `context.WithTimeout()` |
| [config.go](go/tool/config.go#L51) | 51 | `TimeoutSeconds` configuration |

---

## 7. Library Dependencies

### 7.1 .NET Resilience Libraries

| Package | Version | Purpose |
|---------|---------|---------|
| Microsoft.Extensions.Http.Resilience | 10.0.0 | HTTP resilience (Polly-based) |
| Microsoft.Extensions.Caching.Memory | 10.0.0 | In-memory caching |

### 7.2 Python Libraries (from pyproject.toml files)

| Package | Version | Purpose |
|---------|---------|---------|
| httpx | >=0.27.0 | Async HTTP client with connection pooling |
| pytest-retry | >=1 | Test retry support |
| pytest-timeout | >=2.3.1 | Test timeout support |

### 7.3 Go Libraries (from go.mod)

| Package | Version | Purpose |
|---------|---------|---------|
| github.com/cenkalti/backoff/v5 | v5.0.3 | Exponential backoff (indirect) |

---

## 8. Exception Hierarchies

### 8.1 .NET Exceptions

```
PurviewException
├── PurviewRateLimitException  (429 responses)
├── PurviewAuthenticationException  (401/403 responses)
├── PurviewPaymentRequiredException  (402 responses)
└── PurviewRequestException  (other HTTP errors)
```

### 8.2 Python Exceptions

```
AgentFrameworkException
├── AgentException
│   ├── AgentExecutionException
│   └── AgentInitializationError
├── ServiceException
│   ├── ServiceInitializationError
│   ├── ServiceResponseException
│   │   ├── ServiceContentFilterException
│   │   ├── PurviewServiceError
│   │   │   ├── PurviewAuthenticationError
│   │   │   ├── PurviewPaymentRequiredError
│   │   │   ├── PurviewRateLimitError
│   │   │   └── PurviewRequestError
│   │   └── ...
│   └── ServiceInvalidAuthError
├── ToolException
│   └── ToolExecutionException
└── MiddlewareException
```

### 8.3 Go Errors

```go
// Sentinel errors
var (
    ErrSessionNotFound      // Not retryable
    ErrInvalidInput         // Not retryable
    ErrRateLimited          // Retryable
    ErrProviderError        // Retryable
    ErrToolInvocationFailed // Not retryable
)

// Structured error wrapper
type Error struct {
    Op      string  // Operation that failed
    AgentID string  // Agent identifier
    Err     error   // Underlying error
}
```

---

## 9. Middleware Architecture

### 9.1 .NET Middleware

Pattern: Delegating handler chain via IChatClient pipeline.

```csharp
// PurviewChatClient wraps inner IChatClient
public class PurviewChatClient(IChatClient innerClient, ...) : DelegatingChatClient(innerClient)
```

### 9.2 Python Middleware

Pattern: Explicit middleware chain with context objects.

| Middleware Type | Context Class | Purpose |
|-----------------|---------------|---------|
| AgentMiddleware | AgentRunContext | Agent-level interception |
| ChatMiddleware | ChatContext | Chat client interception |
| FunctionMiddleware | FunctionInvocationContext | Tool execution interception |

### 9.3 Go Middleware

Pattern: Function chains via Next() pattern.

```go
// Agent middleware signature
type AgentMiddleware func(next AgentHandler) AgentHandler

// Chat middleware signature  
type ChatMiddleware func(next ChatHandler) ChatHandler
```

---

## 10. Recommendations for Go Implementation

### 10.1 Priority 1: Rate Limiting Middleware

```go
package middleware

import (
    "context"
    "golang.org/x/time/rate"
)

type RateLimiterMiddleware struct {
    limiter *rate.Limiter
}

func NewRateLimiter(requestsPerSecond float64, burst int) *RateLimiterMiddleware {
    return &RateLimiterMiddleware{
        limiter: rate.NewLimiter(rate.Limit(requestsPerSecond), burst),
    }
}
```

### 10.2 Priority 2: Circuit Breaker Middleware

Use `github.com/sony/gobreaker` to implement circuit breaker pattern.

### 10.3 Priority 3: Retry Middleware

Leverage existing `github.com/cenkalti/backoff/v5` dependency for exponential backoff with jitter.

### 10.4 Priority 4: Connection Pool Configuration

Add helper functions for creating production-ready http.Transport configurations.

---

## 11. Gap Analysis

| Pattern | .NET | Python | Go |
|---------|------|--------|-----|
| Rate Limit Detection | ✅ | ✅ | ✅ |
| Rate Limiter (Token Bucket) | ❌ | ❌ | ❌ |
| Circuit Breaker | ✅ (via Polly) | ❌ | ❌ |
| Retry with Backoff | ✅ | ✅ (basic) | ❌ |
| Retry with Jitter | ❌ | ❌ | ❌ |
| Connection Pooling | ✅ (HttpClient) | ✅ (httpx) | ✅ (http.Client) |
| Pool Configuration | ❌ | ✅ | ❌ |
| Timeout Configuration | ✅ | ✅ | ✅ |
| Concurrency Control | ✅ | ✅ | ❌ |
| Error Classification | ✅ | ✅ | ✅ |
| Retryable Detection | ❌ | ❌ | ✅ |

---

## 12. Clarifying Questions

1. **Rate Limiting Strategy**: Should Go implement client-side rate limiting (token bucket) or just honor server-side rate limits?

2. **Circuit Breaker Scope**: Should circuit breaker be per-agent, per-provider, or configurable?

3. **Retry Policy Defaults**: What should be the default max retries and backoff parameters?

4. **Middleware Order**: What is the recommended order for resilience middleware (rate limit → circuit breaker → retry)?

5. **Metrics Integration**: Should resilience patterns emit OpenTelemetry metrics?
