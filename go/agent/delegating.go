// Copyright (c) Microsoft. All rights reserved.

package agent

import (
	"context"
	"encoding/json"
	"reflect"
)

// DelegatingAgent wraps an inner agent to enable the decorator pattern.
// Embed this type and override specific methods to customize behavior.
type DelegatingAgent struct {
	inner Agent
}

// Compile-time check that DelegatingAgent implements Agent.
var _ Agent = (*DelegatingAgent)(nil)

// NewDelegatingAgent creates a new delegating agent wrapping the inner agent.
func NewDelegatingAgent(inner Agent) *DelegatingAgent {
	return &DelegatingAgent{inner: inner}
}

// Inner returns the wrapped agent.
func (d *DelegatingAgent) Inner() Agent {
	return d.inner
}

// ID returns the unique identifier from the inner agent.
func (d *DelegatingAgent) ID() string {
	return d.inner.ID()
}

// Name returns the name from the inner agent.
func (d *DelegatingAgent) Name() string {
	return d.inner.Name()
}

// Description returns the description from the inner agent.
func (d *DelegatingAgent) Description() string {
	return d.inner.Description()
}

// Metadata returns the metadata from the inner agent.
func (d *DelegatingAgent) Metadata() AIAgentMetadata {
	return d.inner.Metadata()
}

// Run forwards to the inner agent's Run method.
func (d *DelegatingAgent) Run(ctx context.Context, messages []Message, opts ...RunOption) (*Response, error) {
	return d.inner.Run(ctx, messages, opts...)
}

// RunStream forwards to the inner agent's RunStream method.
func (d *DelegatingAgent) RunStream(ctx context.Context, messages []Message, opts ...RunOption) (<-chan ResponseUpdate, error) {
	return d.inner.RunStream(ctx, messages, opts...)
}

// NewSession forwards to the inner agent's NewSession method.
func (d *DelegatingAgent) NewSession(ctx context.Context) (Session, error) {
	return d.inner.NewSession(ctx)
}

// RestoreSession forwards to the inner agent's RestoreSession method.
func (d *DelegatingAgent) RestoreSession(ctx context.Context, data json.RawMessage) (Session, error) {
	return d.inner.RestoreSession(ctx, data)
}

// GetService forwards to the inner agent's GetService method.
func (d *DelegatingAgent) GetService(serviceType reflect.Type) interface{} {
	return d.inner.GetService(serviceType)
}
