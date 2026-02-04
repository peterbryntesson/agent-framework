// Copyright (c) Microsoft. All rights reserved.

package resilience

import (
	"time"
)

// Common default values for resilience components.
const (
	// DefaultRPS is the default requests per second for rate limiting.
	DefaultRPS = 100.0

	// DefaultBurst is the default burst size for rate limiting.
	DefaultBurst = 10

	// DefaultMaxAttempts is the default maximum retry attempts.
	DefaultMaxAttempts = 3

	// DefaultInitialBackoff is the default initial backoff duration.
	DefaultInitialBackoff = 100 * time.Millisecond

	// DefaultMaxBackoff is the default maximum backoff duration.
	DefaultMaxBackoff = 30 * time.Second

	// DefaultJitter is the default jitter factor (0.0 to 1.0).
	DefaultJitter = 0.1

	// DefaultCircuitBreakerTimeout is the default timeout before half-open state.
	DefaultCircuitBreakerTimeout = 60 * time.Second

	// DefaultCircuitBreakerMaxRequests is the default max requests in half-open state.
	DefaultCircuitBreakerMaxRequests = 1

	// DefaultConsecutiveFailuresThreshold is the default failure count to trip the circuit.
	DefaultConsecutiveFailuresThreshold = 5
)
