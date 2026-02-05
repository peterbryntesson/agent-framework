// Copyright (c) Microsoft. All rights reserved.

// Package resilience provides production-ready resilience patterns for the Agent Framework.
//
// This package implements common resilience patterns including rate limiting,
// circuit breakers, retry policies, and HTTP client pool configuration.
// These patterns help build robust applications that gracefully handle
// failures, overload conditions, and transient errors.
//
// # Overview
//
// The resilience package provides four main capabilities:
//
// 1. Rate Limiting: Token bucket rate limiting for both global and per-client throttling
// 2. Circuit Breaker: Automatic failure detection and recovery with gobreaker
// 3. Retry Policy: Configurable retry with exponential/linear backoff and jitter
// 4. HTTP Pool: Optimized HTTP client connection pooling for production use
//
// # Rate Limiting
//
// Use [RateLimiter] to protect against request overload:
//
//	limiter := resilience.NewRateLimiter(100, 10) // 100 RPS, burst of 10
//
//	// Wait for rate limit before making request
//	if err := limiter.Wait(ctx); err != nil {
//	    return err // context cancelled or deadline exceeded
//	}
//
//	// Per-client rate limiting
//	if err := limiter.WaitForClient(ctx, "client-123"); err != nil {
//	    return err
//	}
//
//	// As middleware
//	agent := chatagent.New(client,
//	    chatagent.WithChatMiddleware(limiter.Middleware()),
//	)
//
// # Circuit Breaker
//
// Use [CircuitBreaker] to fail fast on repeated errors:
//
//	cb := resilience.NewCircuitBreaker("my-service",
//	    resilience.WithMaxRequests(5),
//	    resilience.WithTimeout(30 * time.Second),
//	    resilience.WithReadyToTrip(func(counts gobreaker.Counts) bool {
//	        return counts.ConsecutiveFailures > 3
//	    }),
//	)
//
//	err := cb.Execute(func() error {
//	    return client.GetResponse(ctx, messages, nil)
//	})
//
//	// As middleware
//	agent := chatagent.New(client,
//	    chatagent.WithChatMiddleware(cb.Middleware()),
//	)
//
// # Retry Policy
//
// Use [RetryPolicy] for automatic retries with backoff:
//
//	policy := resilience.NewRetryPolicy(
//	    resilience.WithMaxAttempts(3),
//	    resilience.WithExponentialBackoff(100*time.Millisecond, 10*time.Second),
//	    resilience.WithJitter(0.1),
//	)
//
//	err := policy.Execute(ctx, func() error {
//	    return client.GetResponse(ctx, messages, nil)
//	})
//
//	// As middleware
//	agent := chatagent.New(client,
//	    chatagent.WithChatMiddleware(policy.Middleware()),
//	)
//
// # HTTP Client Pool
//
// Use [PoolConfig] for optimized HTTP connections:
//
//	config := resilience.DefaultPoolConfig()
//	config.MaxIdleConnsPerHost = 100
//
//	httpClient := resilience.NewHTTPClient(config)
//
// # Combining Patterns
//
// Resilience patterns can be composed for defense in depth:
//
//	// Create resilience components
//	limiter := resilience.NewRateLimiter(100, 10)
//	cb := resilience.NewCircuitBreaker("api")
//	retry := resilience.NewRetryPolicy(resilience.WithMaxAttempts(3))
//
//	// Apply as middleware (order matters: rate limit -> circuit breaker -> retry)
//	agent := chatagent.New(client,
//	    chatagent.WithChatMiddleware(
//	        limiter.Middleware(),
//	        cb.Middleware(),
//	        retry.Middleware(),
//	    ),
//	)
//
// # Thread Safety
//
// All types in this package are safe for concurrent use.
package resilience
