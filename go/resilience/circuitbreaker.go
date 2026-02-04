// Copyright (c) Microsoft. All rights reserved.

package resilience

import (
	"context"
	"errors"
	"time"

	"github.com/microsoft/agent-framework-go/agent"
	"github.com/sony/gobreaker/v2"
)

// ErrCircuitOpen is returned when the circuit breaker is open and not accepting requests.
var ErrCircuitOpen = errors.New("circuit breaker is open")

// CircuitBreaker wraps gobreaker with chat middleware integration.
// It provides automatic failure detection and recovery, preventing cascading failures
// when a service is experiencing issues.
//
// States:
//   - Closed: Normal operation, requests pass through
//   - Open: Too many failures, requests fail fast with ErrCircuitOpen
//   - Half-Open: Testing recovery, limited requests allowed through
type CircuitBreaker struct {
	cb *gobreaker.CircuitBreaker[any]
}

// CircuitBreakerOption configures a CircuitBreaker.
type CircuitBreakerOption func(*gobreaker.Settings)

// NewCircuitBreaker creates a circuit breaker with the given name and options.
// The name is used for logging and metrics.
//
// Example:
//
//	cb := NewCircuitBreaker("my-service",
//	    WithMaxRequests(5),
//	    WithTimeout(30 * time.Second),
//	)
func NewCircuitBreaker(name string, opts ...CircuitBreakerOption) *CircuitBreaker {
	settings := gobreaker.Settings{
		Name:        name,
		MaxRequests: DefaultCircuitBreakerMaxRequests,
		Timeout:     DefaultCircuitBreakerTimeout,
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			return counts.ConsecutiveFailures >= DefaultConsecutiveFailuresThreshold
		},
	}

	for _, opt := range opts {
		opt(&settings)
	}

	return &CircuitBreaker{
		cb: gobreaker.NewCircuitBreaker[any](settings),
	}
}

// WithMaxRequests sets the maximum number of requests allowed in half-open state.
// When the circuit is half-open, this many requests are allowed through to test
// if the service has recovered.
func WithMaxRequests(n uint32) CircuitBreakerOption {
	return func(s *gobreaker.Settings) {
		s.MaxRequests = n
	}
}

// WithTimeout sets the duration the circuit stays open before transitioning to half-open.
// After this timeout, the circuit breaker allows limited requests through to test recovery.
func WithTimeout(d time.Duration) CircuitBreakerOption {
	return func(s *gobreaker.Settings) {
		s.Timeout = d
	}
}

// WithInterval sets the cyclic period of the closed state for clearing internal counts.
// If zero, the circuit breaker doesn't clear internal counts during the closed state.
func WithInterval(d time.Duration) CircuitBreakerOption {
	return func(s *gobreaker.Settings) {
		s.Interval = d
	}
}

// WithReadyToTrip sets the function that determines when to trip the circuit.
// The function is called with the current failure counts whenever a request fails.
// If it returns true, the circuit transitions from closed to open.
func WithReadyToTrip(f func(counts gobreaker.Counts) bool) CircuitBreakerOption {
	return func(s *gobreaker.Settings) {
		s.ReadyToTrip = f
	}
}

// WithOnStateChange sets a callback that's invoked when the circuit state changes.
// This can be used for logging or metrics.
func WithOnStateChange(f func(name string, from, to gobreaker.State)) CircuitBreakerOption {
	return func(s *gobreaker.Settings) {
		s.OnStateChange = f
	}
}

// WithIsSuccessful sets the function that determines if a request was successful.
// By default, any nil error is considered successful.
func WithIsSuccessful(f func(error) bool) CircuitBreakerOption {
	return func(s *gobreaker.Settings) {
		s.IsSuccessful = f
	}
}

// Execute runs the function with circuit breaker protection.
// If the circuit is open, it returns ErrCircuitOpen immediately.
// The function result is used to determine success/failure for the circuit state.
func (cb *CircuitBreaker) Execute(fn func() error) error {
	_, err := cb.cb.Execute(func() (any, error) {
		return nil, fn()
	})
	if errors.Is(err, gobreaker.ErrOpenState) || errors.Is(err, gobreaker.ErrTooManyRequests) {
		return ErrCircuitOpen
	}
	return err
}

// Middleware returns a ChatMiddleware that applies circuit breaking.
// Requests are rejected with ErrCircuitOpen when the circuit is open.
func (cb *CircuitBreaker) Middleware() agent.ChatMiddleware {
	return agent.ChatMiddlewareFunc(func(ctx context.Context, chatCtx *agent.ChatContext, next agent.ChatHandler) error {
		return cb.Execute(func() error {
			return next(ctx, chatCtx)
		})
	})
}

// State returns the current circuit breaker state.
func (cb *CircuitBreaker) State() gobreaker.State {
	return cb.cb.State()
}

// Name returns the name of the circuit breaker.
func (cb *CircuitBreaker) Name() string {
	return cb.cb.Name()
}

// Counts returns the current internal counts.
func (cb *CircuitBreaker) Counts() gobreaker.Counts {
	return cb.cb.Counts()
}
