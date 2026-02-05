// Copyright (c) Microsoft. All rights reserved.

package resilience

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/microsoft/agent-framework-go/agent"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRetryPolicy(t *testing.T) {
	policy := NewRetryPolicy()

	assert.NotNil(t, policy)
	assert.Equal(t, uint(DefaultMaxAttempts), policy.maxAttempts)
	assert.Equal(t, DefaultInitialBackoff, policy.initialBackoff)
	assert.Equal(t, DefaultMaxBackoff, policy.maxBackoff)
	assert.Equal(t, DefaultJitter, policy.jitterFactor)
}

func TestRetryPolicy_WithMaxAttempts(t *testing.T) {
	policy := NewRetryPolicy(WithMaxAttempts(5))
	assert.Equal(t, uint(5), policy.maxAttempts)
}

func TestRetryPolicy_WithMaxAttempts_MinimumOne(t *testing.T) {
	policy := NewRetryPolicy(WithMaxAttempts(0))
	assert.Equal(t, uint(1), policy.maxAttempts)

	policy = NewRetryPolicy(WithMaxAttempts(-5))
	assert.Equal(t, uint(1), policy.maxAttempts)
}

func TestRetryPolicy_WithExponentialBackoff(t *testing.T) {
	policy := NewRetryPolicy(
		WithExponentialBackoff(50*time.Millisecond, 5*time.Second),
	)
	assert.Equal(t, 50*time.Millisecond, policy.initialBackoff)
	assert.Equal(t, 5*time.Second, policy.maxBackoff)
	assert.False(t, policy.useLinear)
}

func TestRetryPolicy_WithLinearBackoff(t *testing.T) {
	policy := NewRetryPolicy(
		WithLinearBackoff(100 * time.Millisecond),
	)
	assert.Equal(t, 100*time.Millisecond, policy.initialBackoff)
	assert.True(t, policy.useLinear)
}

func TestRetryPolicy_WithJitter(t *testing.T) {
	policy := NewRetryPolicy(WithJitter(0.5))
	assert.Equal(t, 0.5, policy.jitterFactor)
}

func TestRetryPolicy_WithJitter_ClampsValues(t *testing.T) {
	policy := NewRetryPolicy(WithJitter(-0.5))
	assert.Equal(t, 0.0, policy.jitterFactor)

	policy = NewRetryPolicy(WithJitter(1.5))
	assert.Equal(t, 1.0, policy.jitterFactor)
}

func TestRetryPolicy_WithRetryableCheck(t *testing.T) {
	isRetryable := func(err error) bool {
		return errors.Is(err, context.DeadlineExceeded)
	}
	policy := NewRetryPolicy(WithRetryableCheck(isRetryable))
	assert.NotNil(t, policy.isRetryable)
}

func TestRetryPolicy_Execute_SuccessNoRetry(t *testing.T) {
	policy := NewRetryPolicy(WithMaxAttempts(3))

	var attempts atomic.Int32
	err := policy.Execute(context.Background(), func() error {
		attempts.Add(1)
		return nil
	})

	assert.NoError(t, err)
	assert.Equal(t, int32(1), attempts.Load())
}

func TestRetryPolicy_Execute_RetryOnFailure(t *testing.T) {
	policy := NewRetryPolicy(
		WithMaxAttempts(3),
		WithLinearBackoff(1*time.Millisecond),
	)

	var attempts atomic.Int32
	err := policy.Execute(context.Background(), func() error {
		attempts.Add(1)
		if attempts.Load() < 3 {
			return errors.New("transient error")
		}
		return nil
	})

	assert.NoError(t, err)
	assert.Equal(t, int32(3), attempts.Load())
}

func TestRetryPolicy_Execute_MaxAttemptsReached(t *testing.T) {
	policy := NewRetryPolicy(
		WithMaxAttempts(3),
		WithLinearBackoff(1*time.Millisecond),
	)

	var attempts atomic.Int32
	expectedErr := errors.New("persistent error")
	err := policy.Execute(context.Background(), func() error {
		attempts.Add(1)
		return expectedErr
	})

	assert.ErrorIs(t, err, expectedErr)
	assert.Equal(t, int32(3), attempts.Load())
}

func TestRetryPolicy_Execute_RespectsContext(t *testing.T) {
	policy := NewRetryPolicy(
		WithMaxAttempts(10),
		WithLinearBackoff(100*time.Millisecond),
	)

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	var attempts atomic.Int32
	err := policy.Execute(ctx, func() error {
		attempts.Add(1)
		return errors.New("always fail")
	})

	// Should have stopped before reaching max attempts
	assert.Error(t, err)
	assert.Less(t, attempts.Load(), int32(10))
}

func TestRetryPolicy_Execute_NonRetryableError(t *testing.T) {
	nonRetryableErr := errors.New("non-retryable")

	policy := NewRetryPolicy(
		WithMaxAttempts(5),
		WithLinearBackoff(1*time.Millisecond),
		WithRetryableCheck(func(err error) bool {
			return !errors.Is(err, nonRetryableErr)
		}),
	)

	var attempts atomic.Int32
	err := policy.Execute(context.Background(), func() error {
		attempts.Add(1)
		return nonRetryableErr
	})

	assert.ErrorIs(t, err, nonRetryableErr)
	assert.Equal(t, int32(1), attempts.Load())
}

func TestRetryPolicy_Middleware(t *testing.T) {
	policy := NewRetryPolicy(
		WithMaxAttempts(3),
		WithLinearBackoff(1*time.Millisecond),
	)
	mw := policy.Middleware()

	var attempts atomic.Int32
	err := mw.Process(context.Background(), &agent.ChatContext{}, func(ctx context.Context, chatCtx *agent.ChatContext) error {
		attempts.Add(1)
		if attempts.Load() < 2 {
			return errors.New("transient")
		}
		return nil
	})

	assert.NoError(t, err)
	assert.Equal(t, int32(2), attempts.Load())
}

func TestRetryPolicy_Middleware_PropagatesError(t *testing.T) {
	policy := NewRetryPolicy(
		WithMaxAttempts(2),
		WithLinearBackoff(1*time.Millisecond),
	)
	mw := policy.Middleware()

	expectedErr := errors.New("persistent error")
	err := mw.Process(context.Background(), &agent.ChatContext{}, func(ctx context.Context, chatCtx *agent.ChatContext) error {
		return expectedErr
	})

	assert.ErrorIs(t, err, expectedErr)
}

func TestRetryPolicy_Execute_ExponentialBackoff(t *testing.T) {
	policy := NewRetryPolicy(
		WithMaxAttempts(3),
		WithExponentialBackoff(10*time.Millisecond, 1*time.Second),
		WithJitter(0), // No jitter for predictable timing
	)

	start := time.Now()
	var attempts atomic.Int32
	_ = policy.Execute(context.Background(), func() error {
		attempts.Add(1)
		if attempts.Load() < 3 {
			return errors.New("fail")
		}
		return nil
	})

	elapsed := time.Since(start)
	// Should have waited at least some time between retries
	assert.GreaterOrEqual(t, elapsed.Milliseconds(), int64(10))
}

func TestApplyJitter(t *testing.T) {
	base := 100 * time.Millisecond

	// No jitter
	result := applyJitter(base, 0)
	assert.Equal(t, base, result)

	// With jitter - result should be different but within range
	results := make(map[time.Duration]bool)
	for i := 0; i < 100; i++ {
		result := applyJitter(base, 0.5)
		results[result] = true
		// Should be within ±50% of base
		assert.GreaterOrEqual(t, result, base-50*time.Millisecond)
		assert.LessOrEqual(t, result, base+50*time.Millisecond)
	}
	// With enough samples, should see some variation
	assert.Greater(t, len(results), 1)
}

func TestDefaultIsRetryable(t *testing.T) {
	assert.True(t, defaultIsRetryable(errors.New("any error")))
	assert.False(t, defaultIsRetryable(nil))
}

func TestRetryPolicy_ConcurrentAccess(t *testing.T) {
	policy := NewRetryPolicy(
		WithMaxAttempts(2),
		WithLinearBackoff(1*time.Millisecond),
	)

	done := make(chan struct{})
	var successCount atomic.Int32

	for i := 0; i < 50; i++ {
		go func() {
			for j := 0; j < 10; j++ {
				if err := policy.Execute(context.Background(), func() error { return nil }); err == nil {
					successCount.Add(1)
				}
			}
			done <- struct{}{}
		}()
	}

	for i := 0; i < 50; i++ {
		<-done
	}

	assert.Equal(t, int32(500), successCount.Load())
}

func TestRetryPolicy_SingleAttempt(t *testing.T) {
	policy := NewRetryPolicy(WithMaxAttempts(1))

	var attempts atomic.Int32
	err := policy.Execute(context.Background(), func() error {
		attempts.Add(1)
		return errors.New("fail")
	})

	assert.Error(t, err)
	assert.Equal(t, int32(1), attempts.Load())
}

func TestRetryPolicy_ExecuteWithContextCancellation(t *testing.T) {
	policy := NewRetryPolicy(
		WithMaxAttempts(100),
		WithLinearBackoff(50*time.Millisecond),
	)

	ctx, cancel := context.WithCancel(context.Background())

	var attempts atomic.Int32
	go func() {
		time.Sleep(30 * time.Millisecond)
		cancel()
	}()

	err := policy.Execute(ctx, func() error {
		attempts.Add(1)
		return errors.New("keep failing")
	})

	require.Error(t, err)
	// Should have been cancelled before too many attempts
	assert.Less(t, attempts.Load(), int32(10))
}
