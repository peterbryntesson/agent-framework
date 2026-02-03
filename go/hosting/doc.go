// Copyright (c) Microsoft. All rights reserved.

// Package hosting provides infrastructure components for hosted agent scenarios.
//
// This package includes:
//   - SessionStore: Interface for persisting agent sessions across requests
//   - InMemorySessionStore: Thread-safe in-memory session storage
//   - NoopSessionStore: No-op implementation for testing
//
// These components are typically used when building HTTP or gRPC servers
// that expose agents as network services.
package hosting
