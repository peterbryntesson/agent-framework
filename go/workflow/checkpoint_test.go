// Copyright (c) Microsoft. All rights reserved.

package workflow

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/microsoft/agent-framework-go/agent"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// checkpointTestExecutor is a simple executor for checkpoint tests.
type checkpointTestExecutor struct {
	id string
}

func newCheckpointTestExecutor(id string) *checkpointTestExecutor {
	return &checkpointTestExecutor{id: id}
}

func (e *checkpointTestExecutor) ID() string {
	return e.id
}

func (e *checkpointTestExecutor) Execute(ctx context.Context, wCtx *WorkflowContext) error {
	return nil
}

func createTestWorkflow() *Workflow {
	start := newCheckpointTestExecutor("start")
	wf, _ := NewBuilder(start).Build()
	return wf
}

func TestInMemoryCheckpointStore_Save(t *testing.T) {
	store := NewInMemoryCheckpointStore()
	ctx := context.Background()

	checkpoint := &Checkpoint{
		ID:        "cp-1",
		RunID:     "run-1",
		Superstep: 5,
		State:     json.RawMessage(`{"key": "value"}`),
		CreatedAt: time.Now(),
	}

	err := store.Save(ctx, checkpoint)
	require.NoError(t, err)

	// Verify it was saved
	loaded, err := store.Load(ctx, "cp-1")
	require.NoError(t, err)
	assert.Equal(t, checkpoint.ID, loaded.ID)
	assert.Equal(t, checkpoint.RunID, loaded.RunID)
	assert.Equal(t, checkpoint.Superstep, loaded.Superstep)
}

func TestInMemoryCheckpointStore_Load(t *testing.T) {
	store := NewInMemoryCheckpointStore()
	ctx := context.Background()

	// Test loading non-existent checkpoint
	_, err := store.Load(ctx, "non-existent")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")

	// Save and load
	checkpoint := &Checkpoint{
		ID:        "cp-1",
		RunID:     "run-1",
		Superstep: 3,
		State:     json.RawMessage(`{}`),
		CreatedAt: time.Now(),
	}

	err = store.Save(ctx, checkpoint)
	require.NoError(t, err)

	loaded, err := store.Load(ctx, "cp-1")
	require.NoError(t, err)
	assert.Equal(t, checkpoint, loaded)
}

func TestInMemoryCheckpointStore_LoadLatest(t *testing.T) {
	store := NewInMemoryCheckpointStore()
	ctx := context.Background()

	// Test with no checkpoints
	_, err := store.LoadLatest(ctx, "run-1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no checkpoints found")

	// Save multiple checkpoints
	cp1 := &Checkpoint{ID: "cp-1", RunID: "run-1", Superstep: 1, CreatedAt: time.Now()}
	cp2 := &Checkpoint{ID: "cp-2", RunID: "run-1", Superstep: 2, CreatedAt: time.Now()}
	cp3 := &Checkpoint{ID: "cp-3", RunID: "run-1", Superstep: 3, CreatedAt: time.Now()}

	require.NoError(t, store.Save(ctx, cp1))
	require.NoError(t, store.Save(ctx, cp2))
	require.NoError(t, store.Save(ctx, cp3))

	// LoadLatest should return the last one saved
	latest, err := store.LoadLatest(ctx, "run-1")
	require.NoError(t, err)
	assert.Equal(t, "cp-3", latest.ID)
	assert.Equal(t, 3, latest.Superstep)
}

func TestInMemoryCheckpointStore_Delete(t *testing.T) {
	store := NewInMemoryCheckpointStore()
	ctx := context.Background()

	// Delete non-existent should not error
	err := store.Delete(ctx, "non-existent")
	require.NoError(t, err)

	// Save and delete
	checkpoint := &Checkpoint{
		ID:        "cp-1",
		RunID:     "run-1",
		Superstep: 1,
		CreatedAt: time.Now(),
	}

	require.NoError(t, store.Save(ctx, checkpoint))

	// Verify it exists
	_, err = store.Load(ctx, "cp-1")
	require.NoError(t, err)

	// Delete
	err = store.Delete(ctx, "cp-1")
	require.NoError(t, err)

	// Verify it's gone
	_, err = store.Load(ctx, "cp-1")
	require.Error(t, err)

	// Verify it's removed from the byRunID index
	checkpoints, err := store.List(ctx, "run-1")
	require.NoError(t, err)
	assert.Empty(t, checkpoints)
}

func TestInMemoryCheckpointStore_List(t *testing.T) {
	store := NewInMemoryCheckpointStore()
	ctx := context.Background()

	// List empty run
	checkpoints, err := store.List(ctx, "run-1")
	require.NoError(t, err)
	assert.Nil(t, checkpoints)

	// Save checkpoints for different runs
	cp1 := &Checkpoint{ID: "cp-1", RunID: "run-1", Superstep: 1}
	cp2 := &Checkpoint{ID: "cp-2", RunID: "run-1", Superstep: 2}
	cp3 := &Checkpoint{ID: "cp-3", RunID: "run-2", Superstep: 1}

	require.NoError(t, store.Save(ctx, cp1))
	require.NoError(t, store.Save(ctx, cp2))
	require.NoError(t, store.Save(ctx, cp3))

	// List run-1
	checkpoints, err = store.List(ctx, "run-1")
	require.NoError(t, err)
	assert.Len(t, checkpoints, 2)
	assert.Equal(t, "cp-1", checkpoints[0].ID)
	assert.Equal(t, "cp-2", checkpoints[1].ID)

	// List run-2
	checkpoints, err = store.List(ctx, "run-2")
	require.NoError(t, err)
	assert.Len(t, checkpoints, 1)
	assert.Equal(t, "cp-3", checkpoints[0].ID)
}

func TestInMemoryCheckpointStore_WithPendingMessages(t *testing.T) {
	store := NewInMemoryCheckpointStore()
	ctx := context.Background()

	messages := []WorkflowMessage{
		{From: "a", To: "b", Content: agent.NewUserMessage("hello"), Superstep: 1},
		{From: "b", To: "c", Content: agent.NewUserMessage("world"), Superstep: 1},
	}

	checkpoint := &Checkpoint{
		ID:              "cp-1",
		RunID:           "run-1",
		Superstep:       1,
		State:           json.RawMessage(`{"count": 42}`),
		PendingMessages: messages,
		CreatedAt:       time.Now(),
	}

	err := store.Save(ctx, checkpoint)
	require.NoError(t, err)

	loaded, err := store.Load(ctx, "cp-1")
	require.NoError(t, err)
	assert.Equal(t, 2, len(loaded.PendingMessages))
	assert.Equal(t, "a", loaded.PendingMessages[0].From)
	assert.Equal(t, "b", loaded.PendingMessages[0].To)
}

func TestWorkflowRunner_SaveCheckpoint_NoStore(t *testing.T) {
	wf := createTestWorkflow()
	runner := NewRunner(wf) // No checkpoint store configured

	ctx := context.Background()
	_, err := runner.SaveCheckpoint(ctx, "run-1", 0, nil, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "checkpoint store not configured")
}

func TestWorkflowRunner_SaveCheckpoint(t *testing.T) {
	store := NewInMemoryCheckpointStore()
	wf := createTestWorkflow()
	runner := NewRunner(wf, WithCheckpointStore(store))

	ctx := context.Background()
	state := map[string]interface{}{
		"counter": 10,
		"name":    "test",
	}
	messages := []WorkflowMessage{
		{From: "a", To: "b", Content: agent.NewUserMessage("pending"), Superstep: 2},
	}

	checkpointID, err := runner.SaveCheckpoint(ctx, "run-1", 5, state, messages)
	require.NoError(t, err)
	assert.NotEmpty(t, checkpointID)

	// Verify the checkpoint was saved
	loaded, err := store.Load(ctx, checkpointID)
	require.NoError(t, err)
	assert.Equal(t, "run-1", loaded.RunID)
	assert.Equal(t, 5, loaded.Superstep)
	assert.Equal(t, 1, len(loaded.PendingMessages))

	// Verify state was serialized
	var loadedState map[string]interface{}
	err = json.Unmarshal(loaded.State, &loadedState)
	require.NoError(t, err)
	assert.Equal(t, float64(10), loadedState["counter"]) // JSON numbers are float64
	assert.Equal(t, "test", loadedState["name"])
}

func TestWorkflowRunner_ResumeFromCheckpoint_NoStore(t *testing.T) {
	wf := createTestWorkflow()
	runner := NewRunner(wf) // No checkpoint store configured

	ctx := context.Background()
	_, err := runner.ResumeFromCheckpoint(ctx, "cp-1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "checkpoint store not configured")
}

func TestWorkflowRunner_ResumeFromCheckpoint_NotFound(t *testing.T) {
	store := NewInMemoryCheckpointStore()
	wf := createTestWorkflow()
	runner := NewRunner(wf, WithCheckpointStore(store))

	ctx := context.Background()
	_, err := runner.ResumeFromCheckpoint(ctx, "non-existent")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to load checkpoint")
}

func TestWorkflowRunner_ResumeFromLatestCheckpoint_NoStore(t *testing.T) {
	wf := createTestWorkflow()
	runner := NewRunner(wf) // No checkpoint store configured

	ctx := context.Background()
	_, err := runner.ResumeFromLatestCheckpoint(ctx, "run-1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "checkpoint store not configured")
}

func TestWorkflowRunner_ResumeFromLatestCheckpoint_NoCheckpoints(t *testing.T) {
	store := NewInMemoryCheckpointStore()
	wf := createTestWorkflow()
	runner := NewRunner(wf, WithCheckpointStore(store))

	ctx := context.Background()
	_, err := runner.ResumeFromLatestCheckpoint(ctx, "run-1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to load latest checkpoint")
}

func TestInMemoryCheckpointStore_ThreadSafety(t *testing.T) {
	store := NewInMemoryCheckpointStore()
	ctx := context.Background()

	// Run concurrent operations
	done := make(chan bool)

	// Writer goroutine
	go func() {
		for i := 0; i < 100; i++ {
			cp := &Checkpoint{
				ID:        "cp-" + string(rune(i)),
				RunID:     "run-1",
				Superstep: i,
				CreatedAt: time.Now(),
			}
			_ = store.Save(ctx, cp)
		}
		done <- true
	}()

	// Reader goroutine
	go func() {
		for i := 0; i < 100; i++ {
			_, _ = store.List(ctx, "run-1")
			_, _ = store.LoadLatest(ctx, "run-1")
		}
		done <- true
	}()

	// Wait for both to complete (test passes if no race condition panic)
	<-done
	<-done
}

func TestInMemoryCheckpointStore_DeleteMiddleCheckpoint(t *testing.T) {
	store := NewInMemoryCheckpointStore()
	ctx := context.Background()

	// Save three checkpoints
	cp1 := &Checkpoint{ID: "cp-1", RunID: "run-1", Superstep: 1}
	cp2 := &Checkpoint{ID: "cp-2", RunID: "run-1", Superstep: 2}
	cp3 := &Checkpoint{ID: "cp-3", RunID: "run-1", Superstep: 3}

	require.NoError(t, store.Save(ctx, cp1))
	require.NoError(t, store.Save(ctx, cp2))
	require.NoError(t, store.Save(ctx, cp3))

	// Delete the middle one
	err := store.Delete(ctx, "cp-2")
	require.NoError(t, err)

	// Verify the list is correct
	checkpoints, err := store.List(ctx, "run-1")
	require.NoError(t, err)
	assert.Len(t, checkpoints, 2)
	assert.Equal(t, "cp-1", checkpoints[0].ID)
	assert.Equal(t, "cp-3", checkpoints[1].ID)

	// LoadLatest should still work
	latest, err := store.LoadLatest(ctx, "run-1")
	require.NoError(t, err)
	assert.Equal(t, "cp-3", latest.ID)
}
