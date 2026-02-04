// Copyright (c) Microsoft. All rights reserved.

package resilience

import (
	"crypto/tls"
	"net"
	"net/http"
	"time"
)

// PoolConfig configures HTTP client connection pooling for production use.
// These settings optimize connection reuse, prevent resource exhaustion,
// and ensure responsive timeout behavior.
type PoolConfig struct {
	// MaxIdleConns controls the maximum number of idle connections across all hosts.
	// Zero means no limit.
	MaxIdleConns int

	// MaxIdleConnsPerHost controls the maximum number of idle connections per host.
	// Zero uses DefaultMaxIdleConnsPerHost (typically 2).
	MaxIdleConnsPerHost int

	// MaxConnsPerHost limits the total connections per host including active and idle.
	// Zero means no limit.
	MaxConnsPerHost int

	// IdleConnTimeout is the maximum time an idle connection remains open.
	// Zero means no timeout.
	IdleConnTimeout time.Duration

	// TLSHandshakeTimeout is the maximum time for TLS handshake.
	// Zero means no timeout.
	TLSHandshakeTimeout time.Duration

	// ResponseHeaderTimeout is the time to wait for response headers after writing the request.
	// Zero means no timeout.
	ResponseHeaderTimeout time.Duration

	// DialTimeout is the maximum time to establish a connection.
	// Zero means no timeout.
	DialTimeout time.Duration

	// KeepAlive specifies the keep-alive period for network connections.
	// Zero uses the default keep-alive behavior.
	KeepAlive time.Duration

	// ExpectContinueTimeout is the time to wait for a 100 Continue response.
	// Zero means no timeout.
	ExpectContinueTimeout time.Duration

	// DisableKeepAlives disables HTTP keep-alives and uses a new connection for each request.
	// This is useful for testing or when connection reuse causes issues.
	DisableKeepAlives bool

	// DisableCompression disables automatic decompression of gzip responses.
	DisableCompression bool

	// ForceHTTP2 enables HTTP/2 support even without TLS (h2c).
	ForceHTTP2 bool
}

// DefaultPoolConfig returns sensible defaults for production use.
// These values are tuned for high-throughput API clients.
func DefaultPoolConfig() PoolConfig {
	return PoolConfig{
		MaxIdleConns:          100,
		MaxIdleConnsPerHost:   10,
		MaxConnsPerHost:       100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ResponseHeaderTimeout: 30 * time.Second,
		DialTimeout:           30 * time.Second,
		KeepAlive:             30 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		DisableKeepAlives:     false,
		DisableCompression:    false,
		ForceHTTP2:            false,
	}
}

// HighThroughputPoolConfig returns settings optimized for high-throughput scenarios.
// Use this when making many concurrent requests to the same host.
func HighThroughputPoolConfig() PoolConfig {
	return PoolConfig{
		MaxIdleConns:          500,
		MaxIdleConnsPerHost:   100,
		MaxConnsPerHost:       0, // No limit
		IdleConnTimeout:       120 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ResponseHeaderTimeout: 60 * time.Second,
		DialTimeout:           30 * time.Second,
		KeepAlive:             30 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		DisableKeepAlives:     false,
		DisableCompression:    false,
		ForceHTTP2:            true,
	}
}

// LowLatencyPoolConfig returns settings optimized for low-latency scenarios.
// Use this when response time is critical.
func LowLatencyPoolConfig() PoolConfig {
	return PoolConfig{
		MaxIdleConns:          100,
		MaxIdleConnsPerHost:   20,
		MaxConnsPerHost:       50,
		IdleConnTimeout:       60 * time.Second,
		TLSHandshakeTimeout:   5 * time.Second,
		ResponseHeaderTimeout: 15 * time.Second,
		DialTimeout:           10 * time.Second,
		KeepAlive:             15 * time.Second,
		ExpectContinueTimeout: 500 * time.Millisecond,
		DisableKeepAlives:     false,
		DisableCompression:    true, // Avoid decompression overhead
		ForceHTTP2:            true,
	}
}

// NewHTTPClient creates an http.Client with optimized connection pooling.
func NewHTTPClient(config PoolConfig) *http.Client {
	return &http.Client{
		Transport: NewHTTPTransport(config),
	}
}

// NewHTTPTransport creates an http.Transport with optimized settings.
// This can be used directly or wrapped with additional functionality.
func NewHTTPTransport(config PoolConfig) *http.Transport {
	dialer := &net.Dialer{
		Timeout:   config.DialTimeout,
		KeepAlive: config.KeepAlive,
	}

	transport := &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		DialContext:           dialer.DialContext,
		MaxIdleConns:          config.MaxIdleConns,
		MaxIdleConnsPerHost:   config.MaxIdleConnsPerHost,
		MaxConnsPerHost:       config.MaxConnsPerHost,
		IdleConnTimeout:       config.IdleConnTimeout,
		TLSHandshakeTimeout:   config.TLSHandshakeTimeout,
		ResponseHeaderTimeout: config.ResponseHeaderTimeout,
		ExpectContinueTimeout: config.ExpectContinueTimeout,
		DisableKeepAlives:     config.DisableKeepAlives,
		DisableCompression:    config.DisableCompression,
		ForceAttemptHTTP2:     config.ForceHTTP2,
		TLSClientConfig: &tls.Config{
			MinVersion: tls.VersionTLS12,
		},
	}

	return transport
}

// NewHTTPClientWithTimeout creates an http.Client with connection pooling and a request timeout.
// The timeout applies to the entire request-response cycle.
func NewHTTPClientWithTimeout(config PoolConfig, timeout time.Duration) *http.Client {
	client := NewHTTPClient(config)
	client.Timeout = timeout
	return client
}
