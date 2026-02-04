// Copyright (c) Microsoft. All rights reserved.

package workflow

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFileCheckpointStore_SaveAndLoad(t *testing.T) {
	// Arrange
	tempDir := t.TempDir()
	store, err := NewFileCheckpointStore(tempDir)
	require.NoError(t, err, "failed to create store")

	ctx := context.Background()
	cp := &Checkpoint{
		ID:        "cp-001",
		RunID:     "run-001",
		Superstep: 5,
		State:     json.RawMessage(`{"key": "value"}`),
		CreatedAt: time.Now(),
	}

	// Act - Save
	err = store.Save(ctx, cp)
	require.NoError(t, err, "failed to save checkpoint")

	// Assert - Verify file exists
	expectedPath := filepath.Join(tempDir, "run-001_cp-001.json")
	_, err = os.Stat(expectedPath)
	assert.False(t, os.IsNotExist(err), "checkpoint file not created")

	// Act - Load
	loaded, err := store.Load(ctx, "cp-001")
	require.NoError(t, err, "failed to load checkpoint")

	// Assert
	assert.Equal(t, cp.ID, loaded.ID)
	assert.Equal(t, cp.RunID, loaded.RunID)
	assert.Equal(t, cp.Superstep, loaded.Superstep)
}

func TestFileCheckpointStore_LoadLatest(t *testing.T) {
	// Arrange
	tempDir := t.TempDir()
	store, err := NewFileCheckpointStore(tempDir)
	require.NoError(t, err, "failed to create store")

	ctx := context.Background()
	now := time.Now()

	// Save multiple checkpoints for same run
	for i := 0; i < 3; i++ {
		cp := &Checkpoint{
			ID:        fmt.Sprintf("cp-%03d", i),
			RunID:     "run-001",
			Superstep: i,
			CreatedAt: now.Add(time.Duration(i) * time.Minute),
		}
		err := store.Save(ctx, cp)
		require.NoError(t, err, "failed to save checkpoint %d", i)
	}

	// Act
	latest, err := store.LoadLatest(ctx, "run-001")
	require.NoError(t, err, "failed to load latest")

	// Assert - Should return the most recent
	assert.Equal(t, "cp-002", latest.ID)
}

func TestFileCheckpointStore_Delete(t *testing.T) {
	// Arrange
	tempDir := t.TempDir()
	store, err := NewFileCheckpointStore(tempDir)
	require.NoError(t, err, "failed to create store")

	ctx := context.Background()
	cp := &Checkpoint{
		ID:        "cp-001",
		RunID:     "run-001",
		CreatedAt: time.Now(),
	}

	err = store.Save(ctx, cp)
	require.NoError(t, err)

	// Act
	err = store.Delete(ctx, "cp-001")
	require.NoError(t, err, "failed to delete")

	// Assert
	_, err = store.Load(ctx, "cp-001")
	assert.Error(t, err, "expected error loading deleted checkpoint")
}

func TestFileCheckpointStore_List(t *testing.T) {
	// Arrange
	tempDir := t.TempDir()
	store, err := NewFileCheckpointStore(tempDir)
	require.NoError(t, err, "failed to create store")

	ctx := context.Background()

	// Save checkpoints for two runs
	for i := 0; i < 3; i++ {
		err := store.Save(ctx, &Checkpoint{
			ID:        fmt.Sprintf("cp-a%d", i),
			RunID:     "run-a",
			CreatedAt: time.Now(),
		})
		require.NoError(t, err)

		err = store.Save(ctx, &Checkpoint{
			ID:        fmt.Sprintf("cp-b%d", i),
			RunID:     "run-b",
			CreatedAt: time.Now(),
		})
		require.NoError(t, err)
	}

	// Act & Assert
	listA, err := store.List(ctx, "run-a")
	require.NoError(t, err)
	assert.Len(t, listA, 3, "expected 3 checkpoints for run-a")

	listB, err := store.List(ctx, "run-b")
	require.NoError(t, err)
	assert.Len(t, listB, 3, "expected 3 checkpoints for run-b")
}

func TestFileCheckpointStore_CreateDirectory(t *testing.T) {
	// Arrange
	tempDir := t.TempDir()
	nestedPath := filepath.Join(tempDir, "nested", "checkpoints")

	// Act
	store, err := NewFileCheckpointStore(nestedPath)

	// Assert
	require.NoError(t, err, "failed to create store with nested path")
	assert.NotNil(t, store)

	info, err := os.Stat(nestedPath)
	require.NoError(t, err, "directory not created")
	assert.True(t, info.IsDir())
}

func TestFileCheckpointStore_LoadNotFound(t *testing.T) {
	// Arrange
	tempDir := t.TempDir()
	store, err := NewFileCheckpointStore(tempDir)
	require.NoError(t, err)

	// Act
	_, err = store.Load(context.Background(), "nonexistent")

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestFileCheckpointStore_LoadLatestNoCheckpoints(t *testing.T) {
	// Arrange
	tempDir := t.TempDir()
	store, err := NewFileCheckpointStore(tempDir)
	require.NoError(t, err)

	// Act
	_, err = store.LoadLatest(context.Background(), "nonexistent-run")

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no checkpoints found")
}

func TestParseFilename(t *testing.T) {
	tests := []struct {
		name         string
		filename     string
		wantRunID    string
		wantCpID     string
		wantOK       bool
	}{
		{
			name:      "valid filename",
			filename:  "run-001_cp-001.json",
			wantRunID: "run-001",
			wantCpID:  "cp-001",
			wantOK:    true,
		},
		{
			name:      "valid with nested path",
			filename:  "/some/path/run-001_cp-001.json",
			wantRunID: "run-001",
			wantCpID:  "cp-001",
			wantOK:    true,
		},
		{
			name:     "not json file",
			filename: "run-001_cp-001.txt",
			wantOK:   false,
		},
		{
			name:     "no underscore",
			filename: "checkpoint.json",
			wantOK:   false,
		},
		{
			name:      "multiple underscores",
			filename:  "run_001_cp_001.json",
			wantRunID: "run",
			wantCpID:  "001_cp_001",
			wantOK:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runID, cpID, ok := parseFilename(tt.filename)
			assert.Equal(t, tt.wantOK, ok)
			if ok {
				assert.Equal(t, tt.wantRunID, runID)
				assert.Equal(t, tt.wantCpID, cpID)
			}
		})
	}
}
