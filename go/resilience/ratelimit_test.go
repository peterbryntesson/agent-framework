// Copyright (c) Microsoft. All rights reserved.

package resilience

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/microsoft/agent-framework-go/agent"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRateLimiter(t *testing.T) {
	limiter := NewRateLimiter(100.0, 10)

	assert.NotNil(t, limiter)
	assert.NotNil(t, limiter.global)
	assert.NotNil(t, limiter.perClient)
	assert.Equal(t, 100.0, limiter.rps)
	assert.Equal(t, 10, limiter.burst)
}

func TestRateLimiter_Wait_AllowsBurst(t *testing.T) {
	limiter := NewRateLimiter(10.0, 5) // 10 RPS, burst of 5

	ctx := context.Background()

	// Should allow burst without waiting
	for i := 0; i < 5; i++ {
		err := limiter.Wait(ctx)
		assert.NoError(t, err)
	}
}

func TestRateLimiter_Wait_RespectsContext(t *testing.T) {
	limiter := NewRateLimiter(0.1, 1) // Very slow rate

	// Exhaust burst
	err := limiter.Wait(context.Background())
	require.NoError(t, err)

	// Next wait should block; use cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	err = limiter.Wait(ctx)
	assert.ErrorIs(t, err, context.Canceled)
}

func TestRateLimiter_Wait_RespectsDeadline(t *testing.T) {
	limiter := NewRateLimiter(0.1, 1) // Very slow rate

	// Exhaust burst
	err := limiter.Wait(context.Background())
	require.NoError(t, err)

	// Next wait should block; use short deadline
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	err = limiter.Wait(ctx)
	// The rate limiter may return a wrapped context error or its own error
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "deadline")
}

func TestRateLimiter_WaitForClient(t *testing.T) {
	limiter := NewRateLimiter(100.0, 5)
	ctx := context.Background()

	// Different clients should have separate limits
	for i := 0; i < 5; i++ {
		err := limiter.WaitForClient(ctx, "client-a")
		assert.NoError(t, err)
	}

	for i := 0; i < 5; i++ {
		err := limiter.WaitForClient(ctx, "client-b")
		assert.NoError(t, err)
	}

	assert.Equal(t, 2, limiter.ClientCount())
}

func TestRateLimiter_WaitForClient_IsolatesClients(t *testing.T) {
	limiter := NewRateLimiter(0.1, 1) // Very slow rate

	ctx := context.Background()

	// Exhaust client-a's burst
	err := limiter.WaitForClient(ctx, "client-a")
	require.NoError(t, err)

	// client-b should still have burst available
	err = limiter.WaitForClient(ctx, "client-b")
	assert.NoError(t, err)
}

func TestRateLimiter_Allow(t *testing.T) {
	limiter := NewRateLimiter(100.0, 5)

	// Should allow burst
	for i := 0; i < 5; i++ {
		assert.True(t, limiter.Allow())
	}
}

func TestRateLimiter_AllowForClient(t *testing.T) {
	limiter := NewRateLimiter(100.0, 3)

	// Each client should have its own burst
	for i := 0; i < 3; i++ {
		assert.True(t, limiter.AllowForClient("client-1"))
	}

	for i := 0; i < 3; i++ {
		assert.True(t, limiter.AllowForClient("client-2"))
	}
}

func TestRateLimiter_Middleware(t *testing.T) {
	limiter := NewRateLimiter(100.0, 10)
	mw := limiter.Middleware()

	var called bool
	next := func(ctx context.Context, chatCtx *agent.ChatContext) error {
		called = true
		return nil
	}

	err := mw.Process(context.Background(), &agent.ChatContext{}, next)
	assert.NoError(t, err)
	assert.True(t, called)
}

func TestRateLimiter_Middleware_BlocksWhenExhausted(t *testing.T) {
	limiter := NewRateLimiter(0.1, 1) // Very slow rate
	mw := limiter.Middleware()

	// Exhaust burst
	err := mw.Process(context.Background(), &agent.ChatContext{}, func(ctx context.Context, chatCtx *agent.ChatContext) error {
		return nil
	})
	require.NoError(t, err)

	// Next request should block; use short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	err = mw.Process(ctx, &agent.ChatContext{}, func(ctx context.Context, chatCtx *agent.ChatContext) error {
		return nil
	})
	assert.Error(t, err)
}

func TestRateLimiter_ClientMiddleware(t *testing.T) {
	limiter := NewRateLimiter(100.0, 10)

	type clientIDKey struct{}
	clientIDFunc := func(ctx context.Context) string {
		if id, ok := ctx.Value(clientIDKey{}).(string); ok {
			return id
		}
		return ""
	}

	mw := limiter.ClientMiddleware(clientIDFunc)

	var called bool
	next := func(ctx context.Context, chatCtx *agent.ChatContext) error {
		called = true
		return nil
	}

	ctx := context.WithValue(context.Background(), clientIDKey{}, "test-client")
	err := mw.Process(ctx, &agent.ChatContext{}, next)
	assert.NoError(t, err)
	assert.True(t, called)
}

func TestRateLimiter_RemoveClient(t *testing.T) {
	limiter := NewRateLimiter(100.0, 5)
	ctx := context.Background()

	// Create some client limiters
	_ = limiter.WaitForClient(ctx, "client-a")
	_ = limiter.WaitForClient(ctx, "client-b")
	assert.Equal(t, 2, limiter.ClientCount())

	// Remove one
	limiter.RemoveClient("client-a")
	assert.Equal(t, 1, limiter.ClientCount())
}

func TestRateLimiter_ConcurrentAccess(t *testing.T) {
	limiter := NewRateLimiter(1000.0, 100)
	ctx := context.Background()

	var wg sync.WaitGroup
	var successCount atomic.Int32

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(clientID int) {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				if err := limiter.WaitForClient(ctx, "client-"+string(rune('a'+clientID%10))); err == nil {
					successCount.Add(1)
				}
			}
		}(i)
	}

	wg.Wait()
	assert.Greater(t, successCount.Load(), int32(0))
}
