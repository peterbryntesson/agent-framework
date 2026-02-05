// Copyright (c) Microsoft. All rights reserved.

package resilience

import (
	"context"
	"math/rand/v2"
	"time"

	"github.com/cenkalti/backoff/v5"
	"github.com/microsoft/agent-framework-go/agent"
)

// RetryPolicy defines retry behavior with configurable backoff strategies.
// It supports exponential backoff, linear backoff, and jitter for distributed systems.
type RetryPolicy struct {
	maxAttempts    uint
	initialBackoff time.Duration
	maxBackoff     time.Duration
	jitterFactor   float64
	useLinear      bool
	isRetryable    func(error) bool
}

// RetryOption configures a RetryPolicy.
type RetryOption func(*RetryPolicy)

// NewRetryPolicy creates a retry policy with the given options.
// By default, it uses exponential backoff with 3 attempts.
//
// Example:
//
//	policy := NewRetryPolicy(
//	    WithMaxAttempts(5),
//	    WithExponentialBackoff(100*time.Millisecond, 30*time.Second),
//	    WithJitter(0.1),
//	)
func NewRetryPolicy(opts ...RetryOption) *RetryPolicy {
	p := &RetryPolicy{
		maxAttempts:    uint(DefaultMaxAttempts),
		initialBackoff: DefaultInitialBackoff,
		maxBackoff:     DefaultMaxBackoff,
		jitterFactor:   DefaultJitter,
		useLinear:      false,
		isRetryable:    defaultIsRetryable,
	}
	for _, opt := range opts {
		opt(p)
	}
	return p
}

// WithMaxAttempts sets the maximum number of retry attempts.
// A value of 1 means no retries (only the initial attempt).
func WithMaxAttempts(n int) RetryOption {
	return func(p *RetryPolicy) {
		if n < 1 {
			n = 1
		}
		p.maxAttempts = uint(n)
	}
}

// WithExponentialBackoff configures exponential backoff with initial and maximum durations.
// Each retry waits longer than the previous, up to the maximum.
func WithExponentialBackoff(initial, max time.Duration) RetryOption {
	return func(p *RetryPolicy) {
		p.initialBackoff = initial
		p.maxBackoff = max
		p.useLinear = false
	}
}

// WithLinearBackoff configures linear backoff with a fixed interval.
// Each retry waits the same amount of time.
func WithLinearBackoff(interval time.Duration) RetryOption {
	return func(p *RetryPolicy) {
		p.initialBackoff = interval
		p.maxBackoff = interval
		p.useLinear = true
	}
}

// WithJitter adds randomization to backoff intervals.
// The factor should be between 0.0 (no jitter) and 1.0 (full jitter).
// Jitter helps prevent thundering herd problems in distributed systems.
func WithJitter(factor float64) RetryOption {
	return func(p *RetryPolicy) {
		if factor < 0 {
			factor = 0
		}
		if factor > 1 {
			factor = 1
		}
		p.jitterFactor = factor
	}
}

// WithRetryableCheck sets the function to determine if an error is retryable.
// By default, all errors are considered retryable.
func WithRetryableCheck(f func(error) bool) RetryOption {
	return func(p *RetryPolicy) {
		p.isRetryable = f
	}
}

// Execute runs the function with retry logic.
// It respects context cancellation and returns immediately if the context is done.
// Returns the error from the last attempt if all retries fail.
func (p *RetryPolicy) Execute(ctx context.Context, fn func() error) error {
	b := p.createBackoff()

	operation := func() (struct{}, error) {
		err := fn()
		if err != nil && !p.isRetryable(err) {
			// Return permanent error to stop retrying
			return struct{}{}, backoff.Permanent(err)
		}
		return struct{}{}, err
	}

	_, err := backoff.Retry(ctx, operation,
		backoff.WithBackOff(b),
		backoff.WithMaxTries(p.maxAttempts),
	)
	return err
}

// Middleware returns a ChatMiddleware that applies retry logic.
// Failed requests are retried according to the policy configuration.
func (p *RetryPolicy) Middleware() agent.ChatMiddleware {
	return agent.ChatMiddlewareFunc(func(ctx context.Context, chatCtx *agent.ChatContext, next agent.ChatHandler) error {
		return p.Execute(ctx, func() error {
			return next(ctx, chatCtx)
		})
	})
}

// createBackoff creates the appropriate backoff strategy based on configuration.
func (p *RetryPolicy) createBackoff() backoff.BackOff {
	if p.useLinear {
		return backoff.NewConstantBackOff(p.initialBackoff)
	}

	// Create exponential backoff with configured values
	b := backoff.NewExponentialBackOff()
	b.InitialInterval = p.initialBackoff
	b.MaxInterval = p.maxBackoff
	b.RandomizationFactor = p.jitterFactor
	return b
}

// defaultIsRetryable considers all errors retryable by default.
func defaultIsRetryable(err error) bool {
	return err != nil
}

// applyJitter adds randomization to a duration.
func applyJitter(d time.Duration, factor float64) time.Duration {
	if factor <= 0 {
		return d
	}
	jitter := time.Duration(float64(d) * factor * (2*rand.Float64() - 1))
	return d + jitter
}
