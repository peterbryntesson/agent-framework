// Copyright (c) Microsoft. All rights reserved.

package resilience

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/microsoft/agent-framework-go/agent"
	"github.com/sony/gobreaker/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewCircuitBreaker(t *testing.T) {
	cb := NewCircuitBreaker("test-service")

	assert.NotNil(t, cb)
	assert.Equal(t, "test-service", cb.Name())
	assert.Equal(t, gobreaker.StateClosed, cb.State())
}

func TestCircuitBreaker_WithOptions(t *testing.T) {
	cb := NewCircuitBreaker("test",
		WithMaxRequests(5),
		WithTimeout(10*time.Second),
		WithInterval(30*time.Second),
		WithOnStateChange(func(name string, from, to gobreaker.State) {
			// Callback for state changes
		}),
	)

	assert.NotNil(t, cb)
	assert.Equal(t, "test", cb.Name())
}

func TestCircuitBreaker_Execute_Success(t *testing.T) {
	cb := NewCircuitBreaker("test")

	var called bool
	err := cb.Execute(func() error {
		called = true
		return nil
	})

	assert.NoError(t, err)
	assert.True(t, called)
}

func TestCircuitBreaker_Execute_Failure(t *testing.T) {
	cb := NewCircuitBreaker("test")

	expectedErr := errors.New("test error")
	err := cb.Execute(func() error {
		return expectedErr
	})

	assert.ErrorIs(t, err, expectedErr)
}

func TestCircuitBreaker_TripsAfterConsecutiveFailures(t *testing.T) {
	cb := NewCircuitBreaker("test",
		WithReadyToTrip(func(counts gobreaker.Counts) bool {
			return counts.ConsecutiveFailures >= 3
		}),
	)

	// Cause consecutive failures
	for i := 0; i < 3; i++ {
		_ = cb.Execute(func() error {
			return errors.New("failure")
		})
	}

	// Circuit should now be open
	assert.Equal(t, gobreaker.StateOpen, cb.State())

	// Next execute should fail fast
	err := cb.Execute(func() error {
		return nil
	})
	assert.ErrorIs(t, err, ErrCircuitOpen)
}

func TestCircuitBreaker_RecoverToHalfOpen(t *testing.T) {
	cb := NewCircuitBreaker("test",
		WithTimeout(50*time.Millisecond),
		WithReadyToTrip(func(counts gobreaker.Counts) bool {
			return counts.ConsecutiveFailures >= 2
		}),
		WithMaxRequests(1),
	)

	// Trip the circuit
	for i := 0; i < 2; i++ {
		_ = cb.Execute(func() error {
			return errors.New("failure")
		})
	}
	assert.Equal(t, gobreaker.StateOpen, cb.State())

	// Wait for timeout
	time.Sleep(60 * time.Millisecond)

	// Circuit should be half-open now
	assert.Equal(t, gobreaker.StateHalfOpen, cb.State())
}

func TestCircuitBreaker_ClosesAfterSuccessfulRecovery(t *testing.T) {
	cb := NewCircuitBreaker("test",
		WithTimeout(50*time.Millisecond),
		WithReadyToTrip(func(counts gobreaker.Counts) bool {
			return counts.ConsecutiveFailures >= 2
		}),
		WithMaxRequests(1),
	)

	// Trip the circuit
	for i := 0; i < 2; i++ {
		_ = cb.Execute(func() error {
			return errors.New("failure")
		})
	}

	// Wait for half-open
	time.Sleep(60 * time.Millisecond)

	// Successful request should close circuit
	err := cb.Execute(func() error {
		return nil
	})
	assert.NoError(t, err)
	assert.Equal(t, gobreaker.StateClosed, cb.State())
}

func TestCircuitBreaker_WithIsSuccessful(t *testing.T) {
	// Only certain errors count as failures
	cb := NewCircuitBreaker("test",
		WithIsSuccessful(func(err error) bool {
			return err == nil || errors.Is(err, context.Canceled)
		}),
		WithReadyToTrip(func(counts gobreaker.Counts) bool {
			return counts.ConsecutiveFailures >= 2
		}),
	)

	// Context canceled should not count as failure
	_ = cb.Execute(func() error {
		return context.Canceled
	})

	assert.Equal(t, gobreaker.StateClosed, cb.State())
}

func TestCircuitBreaker_Middleware(t *testing.T) {
	cb := NewCircuitBreaker("test")
	mw := cb.Middleware()

	var called bool
	next := func(ctx context.Context, chatCtx *agent.ChatContext) error {
		called = true
		return nil
	}

	err := mw.Process(context.Background(), &agent.ChatContext{}, next)
	assert.NoError(t, err)
	assert.True(t, called)
}

func TestCircuitBreaker_Middleware_FailsFast(t *testing.T) {
	cb := NewCircuitBreaker("test",
		WithReadyToTrip(func(counts gobreaker.Counts) bool {
			return counts.ConsecutiveFailures >= 2
		}),
	)
	mw := cb.Middleware()

	// Trip the circuit
	for i := 0; i < 2; i++ {
		_ = mw.Process(context.Background(), &agent.ChatContext{}, func(ctx context.Context, chatCtx *agent.ChatContext) error {
			return errors.New("failure")
		})
	}

	// Next request should fail fast without calling handler
	var called bool
	err := mw.Process(context.Background(), &agent.ChatContext{}, func(ctx context.Context, chatCtx *agent.ChatContext) error {
		called = true
		return nil
	})

	assert.ErrorIs(t, err, ErrCircuitOpen)
	assert.False(t, called)
}

func TestCircuitBreaker_Counts(t *testing.T) {
	cb := NewCircuitBreaker("test")

	// Make some requests
	_ = cb.Execute(func() error { return nil })
	_ = cb.Execute(func() error { return nil })
	_ = cb.Execute(func() error { return errors.New("fail") })

	counts := cb.Counts()
	assert.Equal(t, uint32(3), counts.Requests)
	assert.Equal(t, uint32(2), counts.TotalSuccesses)
	assert.Equal(t, uint32(1), counts.TotalFailures)
}

func TestCircuitBreaker_ConcurrentAccess(t *testing.T) {
	cb := NewCircuitBreaker("test")

	var successCount atomic.Int32
	done := make(chan struct{})

	for i := 0; i < 100; i++ {
		go func() {
			for j := 0; j < 10; j++ {
				if err := cb.Execute(func() error { return nil }); err == nil {
					successCount.Add(1)
				}
			}
			done <- struct{}{}
		}()
	}

	for i := 0; i < 100; i++ {
		<-done
	}

	assert.Equal(t, int32(1000), successCount.Load())
}

func TestErrCircuitOpen_Message(t *testing.T) {
	assert.Equal(t, "circuit breaker is open", ErrCircuitOpen.Error())
}

func TestCircuitBreaker_State_ReturnsCorrectStates(t *testing.T) {
	cb := NewCircuitBreaker("test",
		WithTimeout(10*time.Millisecond),
		WithReadyToTrip(func(counts gobreaker.Counts) bool {
			return counts.ConsecutiveFailures >= 1
		}),
	)

	// Initially closed
	require.Equal(t, gobreaker.StateClosed, cb.State())

	// Trip to open
	_ = cb.Execute(func() error { return errors.New("fail") })
	require.Equal(t, gobreaker.StateOpen, cb.State())
}
