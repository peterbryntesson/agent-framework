// Copyright (c) Microsoft. All rights reserved.

package workflow

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
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

// InMemoryCheckpointStore provides an in-memory checkpoint store.
// This is suitable for testing and single-process applications.
type InMemoryCheckpointStore struct {
	checkpoints map[string]*Checkpoint
	byRunID     map[string][]string // runID -> checkpoint IDs
	mu          sync.RWMutex
}

// NewInMemoryCheckpointStore creates a new in-memory checkpoint store.
func NewInMemoryCheckpointStore() *InMemoryCheckpointStore {
	return &InMemoryCheckpointStore{
		checkpoints: make(map[string]*Checkpoint),
		byRunID:     make(map[string][]string),
	}
}

// Save persists a checkpoint to memory.
func (s *InMemoryCheckpointStore) Save(ctx context.Context, checkpoint *Checkpoint) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.checkpoints[checkpoint.ID] = checkpoint
	s.byRunID[checkpoint.RunID] = append(s.byRunID[checkpoint.RunID], checkpoint.ID)
	return nil
}

// Load retrieves a checkpoint by ID.
func (s *InMemoryCheckpointStore) Load(ctx context.Context, checkpointID string) (*Checkpoint, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	cp, ok := s.checkpoints[checkpointID]
	if !ok {
		return nil, fmt.Errorf("checkpoint %q not found", checkpointID)
	}
	return cp, nil
}

// LoadLatest retrieves the most recent checkpoint for a run.
func (s *InMemoryCheckpointStore) LoadLatest(ctx context.Context, runID string) (*Checkpoint, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	ids, ok := s.byRunID[runID]
	if !ok || len(ids) == 0 {
		return nil, fmt.Errorf("no checkpoints found for run %q", runID)
	}

	return s.checkpoints[ids[len(ids)-1]], nil
}

// Delete removes a checkpoint.
func (s *InMemoryCheckpointStore) Delete(ctx context.Context, checkpointID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	cp, ok := s.checkpoints[checkpointID]
	if !ok {
		return nil
	}

	delete(s.checkpoints, checkpointID)

	ids := s.byRunID[cp.RunID]
	for i, id := range ids {
		if id == checkpointID {
			s.byRunID[cp.RunID] = append(ids[:i], ids[i+1:]...)
			break
		}
	}

	return nil
}

// List returns all checkpoints for a run.
func (s *InMemoryCheckpointStore) List(ctx context.Context, runID string) ([]*Checkpoint, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	ids, ok := s.byRunID[runID]
	if !ok {
		return nil, nil
	}

	result := make([]*Checkpoint, 0, len(ids))
	for _, id := range ids {
		result = append(result, s.checkpoints[id])
	}
	return result, nil
}
