// Copyright (c) Microsoft. All rights reserved.

package workflow

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

// FileCheckpointStore persists checkpoints to the filesystem as JSON files.
// Each checkpoint is stored as a separate file with naming pattern:
// {basePath}/{runID}_{checkpointID}.json
//
// This store uses atomic writes (temp file + rename) for durability and
// an in-process mutex for thread safety. It does not provide cross-process
// locking, so concurrent access from multiple processes is not supported.
type FileCheckpointStore struct {
	basePath string
	mu       sync.RWMutex
}

// NewFileCheckpointStore creates a new file-based checkpoint store.
// The basePath directory is created if it does not exist.
func NewFileCheckpointStore(basePath string) (*FileCheckpointStore, error) {
	// Ensure the base directory exists
	if err := os.MkdirAll(basePath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create checkpoint directory: %w", err)
	}
	return &FileCheckpointStore{
		basePath: basePath,
	}, nil
}

// checkpointFilename generates the filename for a checkpoint.
func (s *FileCheckpointStore) checkpointFilename(runID, checkpointID string) string {
	return filepath.Join(s.basePath, fmt.Sprintf("%s_%s.json", runID, checkpointID))
}

// parseFilename extracts runID and checkpointID from a filename.
func parseFilename(filename string) (runID, checkpointID string, ok bool) {
	base := filepath.Base(filename)
	if !strings.HasSuffix(base, ".json") {
		return "", "", false
	}
	name := strings.TrimSuffix(base, ".json")
	parts := strings.SplitN(name, "_", 2)
	if len(parts) != 2 {
		return "", "", false
	}
	return parts[0], parts[1], true
}

// Save persists a checkpoint to the filesystem atomically.
// It writes to a temporary file first, then renames to the target path.
func (s *FileCheckpointStore) Save(ctx context.Context, checkpoint *Checkpoint) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Serialize checkpoint to JSON
	data, err := json.MarshalIndent(checkpoint, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal checkpoint: %w", err)
	}

	targetPath := s.checkpointFilename(checkpoint.RunID, checkpoint.ID)

	// Write to temporary file first for atomic operation
	tempPath := targetPath + ".tmp"
	if err := os.WriteFile(tempPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write checkpoint file: %w", err)
	}

	// Atomic rename (works on POSIX systems)
	if err := os.Rename(tempPath, targetPath); err != nil {
		// Clean up temp file on failure
		os.Remove(tempPath)
		return fmt.Errorf("failed to finalize checkpoint file: %w", err)
	}

	return nil
}

// Load retrieves a checkpoint by ID.
func (s *FileCheckpointStore) Load(ctx context.Context, checkpointID string) (*Checkpoint, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Search for checkpoint file with matching ID
	files, err := os.ReadDir(s.basePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read checkpoint directory: %w", err)
	}

	for _, file := range files {
		if file.IsDir() {
			continue
		}
		_, cpID, ok := parseFilename(file.Name())
		if ok && cpID == checkpointID {
			return s.loadFile(filepath.Join(s.basePath, file.Name()))
		}
	}

	return nil, fmt.Errorf("checkpoint %q not found", checkpointID)
}

// LoadLatest retrieves the most recent checkpoint for a run.
func (s *FileCheckpointStore) LoadLatest(ctx context.Context, runID string) (*Checkpoint, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	checkpoints, err := s.listForRun(runID)
	if err != nil {
		return nil, err
	}

	if len(checkpoints) == 0 {
		return nil, fmt.Errorf("no checkpoints found for run %q", runID)
	}

	// Sort by CreatedAt descending and return the most recent
	sort.Slice(checkpoints, func(i, j int) bool {
		return checkpoints[i].CreatedAt.After(checkpoints[j].CreatedAt)
	})

	return checkpoints[0], nil
}

// Delete removes a checkpoint by ID.
func (s *FileCheckpointStore) Delete(ctx context.Context, checkpointID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Search for and delete checkpoint file
	files, err := os.ReadDir(s.basePath)
	if err != nil {
		return fmt.Errorf("failed to read checkpoint directory: %w", err)
	}

	for _, file := range files {
		if file.IsDir() {
			continue
		}
		_, cpID, ok := parseFilename(file.Name())
		if ok && cpID == checkpointID {
			path := filepath.Join(s.basePath, file.Name())
			if err := os.Remove(path); err != nil {
				return fmt.Errorf("failed to delete checkpoint: %w", err)
			}
			return nil
		}
	}

	return fmt.Errorf("checkpoint %q not found", checkpointID)
}

// List returns all checkpoints for a run.
func (s *FileCheckpointStore) List(ctx context.Context, runID string) ([]*Checkpoint, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.listForRun(runID)
}

// listForRun is the internal implementation without locking.
func (s *FileCheckpointStore) listForRun(runID string) ([]*Checkpoint, error) {
	files, err := os.ReadDir(s.basePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read checkpoint directory: %w", err)
	}

	var checkpoints []*Checkpoint
	for _, file := range files {
		if file.IsDir() {
			continue
		}
		fileRunID, _, ok := parseFilename(file.Name())
		if ok && fileRunID == runID {
			cp, err := s.loadFile(filepath.Join(s.basePath, file.Name()))
			if err != nil {
				continue // Skip corrupted files
			}
			checkpoints = append(checkpoints, cp)
		}
	}

	return checkpoints, nil
}

// loadFile reads and parses a checkpoint file.
func (s *FileCheckpointStore) loadFile(path string) (*Checkpoint, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read checkpoint file: %w", err)
	}

	var checkpoint Checkpoint
	if err := json.Unmarshal(data, &checkpoint); err != nil {
		return nil, fmt.Errorf("failed to parse checkpoint file: %w", err)
	}

	return &checkpoint, nil
}
