// Copyright (c) Microsoft. All rights reserved.

package resilience

import (
	"context"
	"sync"

	"github.com/microsoft/agent-framework-go/agent"
	"golang.org/x/time/rate"
)

// RateLimiter provides token bucket rate limiting for both global and per-client throttling.
// It is safe for concurrent use by multiple goroutines.
type RateLimiter struct {
	global    *rate.Limiter
	perClient map[string]*rate.Limiter
	rps       float64
	burst     int
	mu        sync.RWMutex
}

// RateLimiterOption configures a RateLimiter.
type RateLimiterOption func(*RateLimiter)

// NewRateLimiter creates a new rate limiter with the specified requests per second and burst size.
// The rps parameter sets the sustainable rate, while burst allows temporary spikes above that rate.
//
// Example:
//
//	limiter := NewRateLimiter(100, 10) // 100 requests/second, burst of 10
func NewRateLimiter(rps float64, burst int, opts ...RateLimiterOption) *RateLimiter {
	rl := &RateLimiter{
		global:    rate.NewLimiter(rate.Limit(rps), burst),
		perClient: make(map[string]*rate.Limiter),
		rps:       rps,
		burst:     burst,
	}
	for _, opt := range opts {
		opt(rl)
	}
	return rl
}

// Wait blocks until the global rate limiter allows an event or the context is canceled.
// Returns nil if the event is allowed, or an error if the context is canceled/deadline exceeded.
func (r *RateLimiter) Wait(ctx context.Context) error {
	return r.global.Wait(ctx)
}

// WaitForClient blocks until the per-client rate limiter allows an event.
// Each clientID gets its own limiter with the same rps and burst as the global limiter.
// Per-client limiters are created lazily on first access.
func (r *RateLimiter) WaitForClient(ctx context.Context, clientID string) error {
	limiter := r.getOrCreateClientLimiter(clientID)
	return limiter.Wait(ctx)
}

// Allow reports whether a global event may happen now.
// Use this for non-blocking rate limit checks.
func (r *RateLimiter) Allow() bool {
	return r.global.Allow()
}

// AllowForClient reports whether an event for the specified client may happen now.
func (r *RateLimiter) AllowForClient(clientID string) bool {
	limiter := r.getOrCreateClientLimiter(clientID)
	return limiter.Allow()
}

// Middleware returns a ChatMiddleware that applies global rate limiting.
// Requests are blocked until the rate limiter allows them or the context is canceled.
func (r *RateLimiter) Middleware() agent.ChatMiddleware {
	return agent.ChatMiddlewareFunc(func(ctx context.Context, chatCtx *agent.ChatContext, next agent.ChatHandler) error {
		if err := r.Wait(ctx); err != nil {
			return err
		}
		return next(ctx, chatCtx)
	})
}

// ClientMiddleware returns a ChatMiddleware that applies per-client rate limiting.
// The clientIDFunc extracts the client ID from the context for per-client limiting.
func (r *RateLimiter) ClientMiddleware(clientIDFunc func(context.Context) string) agent.ChatMiddleware {
	return agent.ChatMiddlewareFunc(func(ctx context.Context, chatCtx *agent.ChatContext, next agent.ChatHandler) error {
		clientID := clientIDFunc(ctx)
		if clientID != "" {
			if err := r.WaitForClient(ctx, clientID); err != nil {
				return err
			}
		} else {
			if err := r.Wait(ctx); err != nil {
				return err
			}
		}
		return next(ctx, chatCtx)
	})
}

// getOrCreateClientLimiter returns the limiter for the given clientID, creating one if needed.
func (r *RateLimiter) getOrCreateClientLimiter(clientID string) *rate.Limiter {
	r.mu.RLock()
	limiter, exists := r.perClient[clientID]
	r.mu.RUnlock()

	if exists {
		return limiter
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	// Double-check after acquiring write lock
	if limiter, exists = r.perClient[clientID]; exists {
		return limiter
	}

	limiter = rate.NewLimiter(rate.Limit(r.rps), r.burst)
	r.perClient[clientID] = limiter
	return limiter
}

// RemoveClient removes the per-client limiter for the specified clientID.
// This can be used to clean up limiters for clients that are no longer active.
func (r *RateLimiter) RemoveClient(clientID string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.perClient, clientID)
}

// ClientCount returns the number of per-client limiters currently tracked.
func (r *RateLimiter) ClientCount() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.perClient)
}
