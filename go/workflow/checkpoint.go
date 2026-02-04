// Copyright (c) Microsoft. All rights reserved.

package workflow

import (
	"context"
	"encoding/json"
	"time"
)

// Checkpoint represents a saved workflow state.
type Checkpoint struct {
	// ID is the unique checkpoint identifier
	ID string `json:"id"`

	// RunID is the workflow run identifier
	RunID string `json:"run_id"`

	// Superstep is the superstep number when saved
	Superstep int `json:"superstep"`

	// State is the serialized workflow state
	State json.RawMessage `json:"state"`

	// PendingMessages are messages waiting for delivery
	PendingMessages []WorkflowMessage `json:"pending_messages"`

	// CreatedAt is when the checkpoint was created
	CreatedAt time.Time `json:"created_at"`
}

// CheckpointStore defines the interface for checkpoint persistence.
// Full implementation is in Phase 4.
type CheckpointStore interface {
	// Save persists a checkpoint.
	Save(ctx context.Context, checkpoint *Checkpoint) error

	// Load retrieves a checkpoint by ID.
	Load(ctx context.Context, checkpointID string) (*Checkpoint, error)

	// LoadLatest retrieves the most recent checkpoint for a run.
	LoadLatest(ctx context.Context, runID string) (*Checkpoint, error)

	// Delete removes a checkpoint.
	Delete(ctx context.Context, checkpointID string) error

	// List returns all checkpoints for a run.
	List(ctx context.Context, runID string) ([]*Checkpoint, error)
}
