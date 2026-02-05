// Copyright (c) Microsoft. All rights reserved.

package mcp

import (
	"errors"
	"io"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNextRequestID(t *testing.T) {
	t.Run("generates unique IDs", func(t *testing.T) {
		// Act
		id1 := nextRequestID()
		id2 := nextRequestID()
		id3 := nextRequestID()

		// Assert
		assert.NotEqual(t, id1, id2)
		assert.NotEqual(t, id2, id3)
		assert.NotEqual(t, id1, id3)
	})

	t.Run("IDs are sequential", func(t *testing.T) {
		// Act
		id1 := nextRequestID()
		id2 := nextRequestID()

		// Assert
		assert.Equal(t, id1+1, id2)
	})

	t.Run("IDs are positive", func(t *testing.T) {
		// Act
		id := nextRequestID()

		// Assert
		assert.Greater(t, id, int64(0))
	})
}

func TestTransportOptions(t *testing.T) {
	t.Run("WithTransportTimeout sets timeout", func(t *testing.T) {
		// Arrange
		cfg := defaultTransportConfig()

		// Act
		WithTransportTimeout(10 * time.Second)(cfg)

		// Assert
		assert.Equal(t, 10*time.Second, cfg.timeout)
	})

	t.Run("WithReadTimeout sets read timeout", func(t *testing.T) {
		// Arrange
		cfg := defaultTransportConfig()

		// Act
		WithReadTimeout(2 * time.Minute)(cfg)

		// Assert
		assert.Equal(t, 2*time.Minute, cfg.readTimeout)
	})

	t.Run("multiple options are applied in order", func(t *testing.T) {
		// Arrange
		cfg := defaultTransportConfig()
		opts := []TransportOption{
			WithTransportTimeout(5 * time.Second),
			WithReadTimeout(1 * time.Minute),
			WithTransportTimeout(15 * time.Second), // Overrides first
		}

		// Act
		cfg.applyOptions(opts)

		// Assert
		assert.Equal(t, 15*time.Second, cfg.timeout)
		assert.Equal(t, 1*time.Minute, cfg.readTimeout)
	})
}

func TestDefaultTransportConfig(t *testing.T) {
	t.Run("has reasonable defaults", func(t *testing.T) {
		// Act
		cfg := defaultTransportConfig()

		// Assert
		assert.Equal(t, 30*time.Second, cfg.timeout)
		assert.Equal(t, 5*time.Minute, cfg.readTimeout)
	})
}

func TestTransportError(t *testing.T) {
	t.Run("Error includes operation and cause", func(t *testing.T) {
		// Arrange
		cause := errors.New("connection refused")
		err := newTransportError("start", cause)

		// Act
		msg := err.Error()

		// Assert
		assert.Contains(t, msg, "mcp transport start")
		assert.Contains(t, msg, "connection refused")
	})

	t.Run("Error handles nil cause", func(t *testing.T) {
		// Arrange
		err := &TransportError{Op: "close"}

		// Act
		msg := err.Error()

		// Assert
		assert.Equal(t, "mcp transport close failed", msg)
	})

	t.Run("Unwrap returns cause", func(t *testing.T) {
		// Arrange
		cause := errors.New("underlying error")
		err := newTransportError("send", cause)

		// Act
		unwrapped := err.Unwrap()

		// Assert
		assert.Equal(t, cause, unwrapped)
	})

	t.Run("Unwrap returns nil for nil cause", func(t *testing.T) {
		// Arrange
		err := &TransportError{Op: "receive"}

		// Act
		unwrapped := err.Unwrap()

		// Assert
		assert.Nil(t, unwrapped)
	})

	t.Run("Is matches EOF", func(t *testing.T) {
		// Arrange
		err := newTransportError("receive", io.EOF)

		// Act & Assert
		assert.True(t, err.Is(io.EOF))
	})

	t.Run("Is does not match non-EOF", func(t *testing.T) {
		// Arrange
		err := newTransportError("send", errors.New("other"))

		// Act & Assert
		assert.False(t, err.Is(io.EOF))
	})

	t.Run("errors.Is works with TransportError", func(t *testing.T) {
		// Arrange
		err := newTransportError("receive", io.EOF)

		// Act & Assert
		assert.True(t, errors.Is(err, io.EOF))
	})

	t.Run("errors.As works with TransportError", func(t *testing.T) {
		// Arrange
		cause := errors.New("wrapped")
		err := newTransportError("start", cause)

		// Act
		var transportErr *TransportError
		ok := errors.As(err, &transportErr)

		// Assert
		require.True(t, ok)
		assert.Equal(t, "start", transportErr.Op)
		assert.Equal(t, cause, transportErr.Cause)
	})
}

func TestNewTransportError(t *testing.T) {
	t.Run("creates error with operation and cause", func(t *testing.T) {
		// Arrange
		cause := errors.New("test error")

		// Act
		err := newTransportError("test_op", cause)

		// Assert
		assert.Equal(t, "test_op", err.Op)
		assert.Equal(t, cause, err.Cause)
	})

	t.Run("creates error with nil cause", func(t *testing.T) {
		// Act
		err := newTransportError("empty", nil)

		// Assert
		assert.Equal(t, "empty", err.Op)
		assert.Nil(t, err.Cause)
	})
}

func TestTransportConfig_ApplyOptions(t *testing.T) {
	t.Run("applies empty options list", func(t *testing.T) {
		// Arrange
		cfg := defaultTransportConfig()
		originalTimeout := cfg.timeout

		// Act
		cfg.applyOptions(nil)

		// Assert
		assert.Equal(t, originalTimeout, cfg.timeout)
	})

	t.Run("applies single option", func(t *testing.T) {
		// Arrange
		cfg := defaultTransportConfig()

		// Act
		cfg.applyOptions([]TransportOption{
			WithTransportTimeout(42 * time.Second),
		})

		// Assert
		assert.Equal(t, 42*time.Second, cfg.timeout)
	})
}
