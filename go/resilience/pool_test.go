// Copyright (c) Microsoft. All rights reserved.

package resilience

import (
	"crypto/tls"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultPoolConfig(t *testing.T) {
	config := DefaultPoolConfig()

	assert.Equal(t, 100, config.MaxIdleConns)
	assert.Equal(t, 10, config.MaxIdleConnsPerHost)
	assert.Equal(t, 100, config.MaxConnsPerHost)
	assert.Equal(t, 90*time.Second, config.IdleConnTimeout)
	assert.Equal(t, 10*time.Second, config.TLSHandshakeTimeout)
	assert.Equal(t, 30*time.Second, config.ResponseHeaderTimeout)
	assert.Equal(t, 30*time.Second, config.DialTimeout)
	assert.Equal(t, 30*time.Second, config.KeepAlive)
	assert.Equal(t, 1*time.Second, config.ExpectContinueTimeout)
	assert.False(t, config.DisableKeepAlives)
	assert.False(t, config.DisableCompression)
	assert.False(t, config.ForceHTTP2)
}

func TestHighThroughputPoolConfig(t *testing.T) {
	config := HighThroughputPoolConfig()

	assert.Equal(t, 500, config.MaxIdleConns)
	assert.Equal(t, 100, config.MaxIdleConnsPerHost)
	assert.Equal(t, 0, config.MaxConnsPerHost) // No limit
	assert.True(t, config.ForceHTTP2)
}

func TestLowLatencyPoolConfig(t *testing.T) {
	config := LowLatencyPoolConfig()

	assert.Equal(t, 5*time.Second, config.TLSHandshakeTimeout)
	assert.Equal(t, 10*time.Second, config.DialTimeout)
	assert.True(t, config.DisableCompression)
	assert.True(t, config.ForceHTTP2)
}

func TestNewHTTPClient(t *testing.T) {
	config := DefaultPoolConfig()
	client := NewHTTPClient(config)

	require.NotNil(t, client)
	require.NotNil(t, client.Transport)
}

func TestNewHTTPClientWithTimeout(t *testing.T) {
	config := DefaultPoolConfig()
	timeout := 45 * time.Second
	client := NewHTTPClientWithTimeout(config, timeout)

	require.NotNil(t, client)
	assert.Equal(t, timeout, client.Timeout)
}

func TestNewHTTPTransport(t *testing.T) {
	config := PoolConfig{
		MaxIdleConns:          50,
		MaxIdleConnsPerHost:   5,
		MaxConnsPerHost:       25,
		IdleConnTimeout:       60 * time.Second,
		TLSHandshakeTimeout:   5 * time.Second,
		ResponseHeaderTimeout: 15 * time.Second,
		DialTimeout:           10 * time.Second,
		KeepAlive:             20 * time.Second,
		ExpectContinueTimeout: 2 * time.Second,
		DisableKeepAlives:     true,
		DisableCompression:    true,
		ForceHTTP2:            true,
	}

	transport := NewHTTPTransport(config)

	require.NotNil(t, transport)
	assert.Equal(t, 50, transport.MaxIdleConns)
	assert.Equal(t, 5, transport.MaxIdleConnsPerHost)
	assert.Equal(t, 25, transport.MaxConnsPerHost)
	assert.Equal(t, 60*time.Second, transport.IdleConnTimeout)
	assert.Equal(t, 5*time.Second, transport.TLSHandshakeTimeout)
	assert.Equal(t, 15*time.Second, transport.ResponseHeaderTimeout)
	assert.Equal(t, 2*time.Second, transport.ExpectContinueTimeout)
	assert.True(t, transport.DisableKeepAlives)
	assert.True(t, transport.DisableCompression)
	assert.True(t, transport.ForceAttemptHTTP2)
}

func TestNewHTTPTransport_TLSConfig(t *testing.T) {
	config := DefaultPoolConfig()
	transport := NewHTTPTransport(config)

	require.NotNil(t, transport.TLSClientConfig)
	assert.Equal(t, uint16(tls.VersionTLS12), transport.TLSClientConfig.MinVersion)
}

func TestNewHTTPTransport_HasDialContext(t *testing.T) {
	config := DefaultPoolConfig()
	transport := NewHTTPTransport(config)

	assert.NotNil(t, transport.DialContext)
}

func TestPoolConfig_ZeroValues(t *testing.T) {
	config := PoolConfig{} // All zero values
	transport := NewHTTPTransport(config)

	// Should not panic and should have reasonable defaults
	require.NotNil(t, transport)
}

func TestNewHTTPClient_CanMakeRequests(t *testing.T) {
	config := DefaultPoolConfig()
	client := NewHTTPClient(config)

	// Just verify the client is usable (not making actual requests)
	require.NotNil(t, client)
	require.NotNil(t, client.Transport)
}

func TestPoolConfig_CustomValues(t *testing.T) {
	config := PoolConfig{
		MaxIdleConns:        200,
		MaxIdleConnsPerHost: 20,
		MaxConnsPerHost:     200,
		IdleConnTimeout:     120 * time.Second,
	}

	client := NewHTTPClient(config)
	require.NotNil(t, client)
}

func TestHighThroughputPoolConfig_NoConnectionLimit(t *testing.T) {
	config := HighThroughputPoolConfig()

	// MaxConnsPerHost = 0 means no limit
	assert.Equal(t, 0, config.MaxConnsPerHost)

	transport := NewHTTPTransport(config)
	assert.Equal(t, 0, transport.MaxConnsPerHost)
}

func TestLowLatencyPoolConfig_DisablesCompression(t *testing.T) {
	config := LowLatencyPoolConfig()

	// Compression is disabled to avoid decompression overhead
	assert.True(t, config.DisableCompression)

	transport := NewHTTPTransport(config)
	assert.True(t, transport.DisableCompression)
}

func TestNewHTTPTransport_ProxyFromEnvironment(t *testing.T) {
	config := DefaultPoolConfig()
	transport := NewHTTPTransport(config)

	// Should use proxy from environment by default
	assert.NotNil(t, transport.Proxy)
}
