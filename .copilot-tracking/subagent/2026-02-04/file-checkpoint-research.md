# FileCheckpointStore Research for Go

**Date:** 2026-02-04  
**Status:** Complete  
**Purpose:** Research .NET and Python FileCheckpointStore implementations to guide Go implementation

---

## Executive Summary

Both .NET and Python implementations use JSON-based file storage with similar patterns:
- One JSON file per checkpoint
- File naming: `{checkpointId}.json` (Python) or `{runId}_{checkpointId}.json` (.NET)
- Atomic writes via temp file + rename
- Index file for .NET only (JSONL format)

---

## 1. Interface Comparison

### Python: `CheckpointStorage` Protocol

```python
class CheckpointStorage(Protocol):
    async def save_checkpoint(self, checkpoint: WorkflowCheckpoint) -> str
    async def load_checkpoint(self, checkpoint_id: str) -> WorkflowCheckpoint | None
    async def list_checkpoint_ids(self, workflow_id: str | None = None) -> list[str]
    async def list_checkpoints(self, workflow_id: str | None = None) -> list[WorkflowCheckpoint]
    async def delete_checkpoint(self, checkpoint_id: str) -> bool
```

### .NET: `ICheckpointStore<TStoreObject>`

```csharp
public interface ICheckpointStore<TStoreObject>
{
    ValueTask<IEnumerable<CheckpointInfo>> RetrieveIndexAsync(string runId, CheckpointInfo? withParent = null);
    ValueTask<CheckpointInfo> CreateCheckpointAsync(string runId, TStoreObject value, CheckpointInfo? parent = null);
    ValueTask<TStoreObject> RetrieveCheckpointAsync(string runId, CheckpointInfo key);
}
```

### Existing Go: `CheckpointStore`

```go
type CheckpointStore interface {
    Save(ctx context.Context, checkpoint *Checkpoint) error
    Load(ctx context.Context, checkpointID string) (*Checkpoint, error)
    LoadLatest(ctx context.Context, runID string) (*Checkpoint, error)
    Delete(ctx context.Context, checkpointID string) error
    List(ctx context.Context, runID string) ([]*Checkpoint, error)
}
```

### Key Differences

| Feature | Go | Python | .NET |
|---------|-----|--------|------|
| LoadLatest | ✓ | ✗ | ✗ |
| Delete | ✓ | ✓ | ✗ |
| List by run | ✓ | ✓ (via filter) | ✓ (via index) |
| Parent tracking | ✗ | ✗ | ✓ |
| Context support | ✓ | N/A (async) | N/A (ValueTask) |

---

## 2. File/Directory Structure

### Python (`FileCheckpointStorage`)

```
storage_path/
├── {checkpoint_id_1}.json
├── {checkpoint_id_2}.json
└── {checkpoint_id_n}.json
```

- **File naming:** `{checkpoint_id}.json`
- **Flat structure:** All checkpoints in single directory
- **No index file:** Lists by scanning directory with `*.json` glob

### .NET (`FileSystemJsonCheckpointStore`)

```
directory/
├── index.jsonl
├── {runId}_{checkpointId_1}.json
├── {runId}_{checkpointId_2}.json
└── {runId}_{checkpointId_n}.json
```

- **File naming:** `{runId}_{checkpointId}.json`
- **Index file:** `index.jsonl` (JSON Lines format with CheckpointInfo records)
- **File locking:** Exclusive lock on index.jsonl for single-process guarantee

### Recommended Go Structure

```
storage_path/
├── index.jsonl           # Optional: for fast listing (like .NET)
├── {runId}/              # Subdirectory per run for organization
│   ├── {checkpointId_1}.json
│   ├── {checkpointId_2}.json
│   └── latest            # Symlink or file with latest checkpoint ID
└── ...
```

**Alternative (simpler, like Python):**

```
storage_path/
├── {runId}_{checkpointId_1}.json
├── {runId}_{checkpointId_2}.json
└── ...
```

---

## 3. Serialization Approach

### Python

- **Format:** JSON (via `json.dump`/`json.load`)
- **Indentation:** 2 spaces (`indent=2`)
- **Encoding:** UTF-8 (`ensure_ascii=False`)
- **Checkpoint structure:**

```json
{
  "checkpoint_id": "uuid",
  "workflow_id": "string",
  "timestamp": "ISO8601",
  "messages": {},
  "shared_state": {},
  "pending_request_info_events": {},
  "iteration_count": 0,
  "metadata": {},
  "version": "1.0"
}
```

### .NET

- **Format:** JSON (via `System.Text.Json`)
- **Indentation:** None (`Indented = false`)
- **Uses `JsonElement` as intermediate format for flexibility
- **Custom converters for complex types (CheckpointInfo, EdgeId, etc.)

### Recommended Go Approach

```go
// Use encoding/json with struct tags
type Checkpoint struct {
    ID              string          `json:"id"`
    RunID           string          `json:"run_id"`
    Superstep       int             `json:"superstep"`
    State           json.RawMessage `json:"state"`
    PendingMessages []WorkflowMessage `json:"pending_messages"`
    CreatedAt       time.Time       `json:"created_at"`
}

// Serialize with json.MarshalIndent for debugging, json.Marshal for production
```

**Why JSON over gob/protobuf:**
1. Human-readable for debugging
2. Consistent with .NET and Python
3. Cross-platform compatibility
4. No schema compilation needed

---

## 4. Atomic Write Strategy

### Python Implementation

```python
async def save_checkpoint(self, checkpoint: WorkflowCheckpoint) -> str:
    file_path = self.storage_path / f"{checkpoint.checkpoint_id}.json"
    
    def _write_atomic() -> None:
        tmp_path = file_path.with_suffix(".json.tmp")
        with open(tmp_path, "w") as f:
            json.dump(checkpoint_dict, f, indent=2, ensure_ascii=False)
        os.replace(tmp_path, file_path)  # Atomic on POSIX

    await asyncio.to_thread(_write_atomic)
```

### .NET Implementation

```csharp
using Stream checkpointStream = File.Open(fileName, FileMode.Create, FileAccess.Write, FileShare.None);
using Utf8JsonWriter jsonWriter = new(checkpointStream, new JsonWriterOptions() { Indented = false });
value.WriteTo(jsonWriter);
// Note: .NET uses exclusive file lock via FileShare.None
```

### Recommended Go Approach

```go
func (s *FileCheckpointStore) atomicWrite(path string, data []byte) error {
    dir := filepath.Dir(path)
    
    // Create temp file in same directory (ensures same filesystem for rename)
    tmp, err := os.CreateTemp(dir, ".checkpoint-*.tmp")
    if err != nil {
        return fmt.Errorf("create temp file: %w", err)
    }
    tmpPath := tmp.Name()
    
    // Cleanup on failure
    success := false
    defer func() {
        if !success {
            os.Remove(tmpPath)
        }
    }()
    
    // Write and sync
    if _, err := tmp.Write(data); err != nil {
        tmp.Close()
        return fmt.Errorf("write temp file: %w", err)
    }
    if err := tmp.Sync(); err != nil {
        tmp.Close()
        return fmt.Errorf("sync temp file: %w", err)
    }
    if err := tmp.Close(); err != nil {
        return fmt.Errorf("close temp file: %w", err)
    }
    
    // Atomic rename
    if err := os.Rename(tmpPath, path); err != nil {
        return fmt.Errorf("rename to final path: %w", err)
    }
    
    success = true
    return nil
}
```

---

## 5. File Locking Mechanism

### Python

- **No file locking:** Relies on atomic rename for safety
- **Thread safety:** Uses `asyncio.to_thread` for I/O operations
- **Process safety:** Not guaranteed (possible data corruption with concurrent processes)

### .NET

- **Exclusive lock on index file:** `FileShare.None` when opening index.jsonl
- **Checkpoint files:** `FileShare.None` for writes, `FileShare.Read` for reads
- **Process safety:** Single-process design (throws if already in use)

```csharp
this._indexFile = File.Open(
    Path.Combine(directory.FullName, "index.jsonl"), 
    FileMode.OpenOrCreate, 
    FileAccess.ReadWrite, 
    FileShare.None  // Exclusive lock
);
```

### Recommended Go Approach

**Option 1: Simple (like Python)**
- No file locking
- Rely on atomic rename
- Document single-process limitation

**Option 2: Advisory locking (recommended)**
```go
import "golang.org/x/sys/unix"  // or syscall

func (s *FileCheckpointStore) acquireLock() error {
    lockPath := filepath.Join(s.dir, ".lock")
    f, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0644)
    if err != nil {
        return err
    }
    s.lockFile = f
    
    // Try exclusive lock (non-blocking)
    if err := unix.Flock(int(f.Fd()), unix.LOCK_EX|unix.LOCK_NB); err != nil {
        f.Close()
        return fmt.Errorf("store already in use by another process")
    }
    return nil
}
```

**Option 3: Cross-platform lock file**
```go
// Use github.com/gofrs/flock for cross-platform support
import "github.com/gofrs/flock"

type FileCheckpointStore struct {
    dir      string
    fileLock *flock.Flock
}
```

---

## 6. Checkpoint ID to File Path Mapping

### Python

```python
file_path = self.storage_path / f"{checkpoint.checkpoint_id}.json"
```

### .NET

```csharp
private string GetFileNameForCheckpoint(string runId, CheckpointInfo key)
    => Path.Combine(this.Directory.FullName, $"{runId}_{key.CheckpointId}.json");
```

### Recommended Go Implementation

```go
func (s *FileCheckpointStore) checkpointPath(checkpoint *Checkpoint) string {
    // Include runID in filename for easy filtering
    filename := fmt.Sprintf("%s_%s.json", checkpoint.RunID, checkpoint.ID)
    return filepath.Join(s.dir, filename)
}

func (s *FileCheckpointStore) parseFilename(name string) (runID, checkpointID string, ok bool) {
    // Parse "{runID}_{checkpointID}.json"
    if !strings.HasSuffix(name, ".json") {
        return "", "", false
    }
    base := strings.TrimSuffix(name, ".json")
    parts := strings.SplitN(base, "_", 2)
    if len(parts) != 2 {
        return "", "", false
    }
    return parts[0], parts[1], true
}
```

---

## 7. Cleanup/Pruning Strategies

### Python

- **Manual delete only:** `delete_checkpoint(checkpoint_id)` method
- **No automatic pruning**

### .NET

- **No delete method in ICheckpointStore**
- **Cleanup only on write failure** (attempts to delete failed checkpoint file)

### Recommended Go Approach

```go
// Pruning options for FileCheckpointStore
type PruneOptions struct {
    // MaxCheckpoints keeps only the N most recent checkpoints per run
    MaxCheckpoints int
    
    // MaxAge removes checkpoints older than this duration
    MaxAge time.Duration
    
    // DryRun reports what would be deleted without deleting
    DryRun bool
}

func (s *FileCheckpointStore) Prune(ctx context.Context, runID string, opts PruneOptions) ([]string, error)

// Automatic pruning on save (optional)
type FileCheckpointStoreOptions struct {
    Dir            string
    MaxCheckpoints int  // If > 0, prune after each save
}
```

---

## 8. Error Handling for I/O Failures

### Python

```python
try:
    with open(file_path) as f:
        data = json.load(f)
except Exception as e:
    logger.warning(f"Failed to read checkpoint file {file_path}: {e}")
```

### .NET

```csharp
try {
    // ... create checkpoint
} catch (Exception ex) {
    this.CheckpointIndex.Remove(key);
    try {
        File.Delete(fileName);  // Cleanup on failure
    } catch { }
    throw new InvalidOperationException($"Could not create checkpoint...", ex);
}
```

### Recommended Go Error Handling

```go
var (
    ErrCheckpointNotFound = errors.New("checkpoint not found")
    ErrStoreInUse         = errors.New("checkpoint store already in use")
    ErrCorruptedCheckpoint = errors.New("corrupted checkpoint data")
)

func (s *FileCheckpointStore) Load(ctx context.Context, checkpointID string) (*Checkpoint, error) {
    path := s.pathForID(checkpointID)
    
    data, err := os.ReadFile(path)
    if errors.Is(err, os.ErrNotExist) {
        return nil, fmt.Errorf("%w: %s", ErrCheckpointNotFound, checkpointID)
    }
    if err != nil {
        return nil, fmt.Errorf("read checkpoint file: %w", err)
    }
    
    var cp Checkpoint
    if err := json.Unmarshal(data, &cp); err != nil {
        return nil, fmt.Errorf("%w: %s: %v", ErrCorruptedCheckpoint, checkpointID, err)
    }
    
    return &cp, nil
}
```

---

## 9. Recommended Go Implementation

### Complete FileCheckpointStore

```go
// Copyright (c) Microsoft. All rights reserved.

package workflow

import (
    "context"
    "encoding/json"
    "errors"
    "fmt"
    "os"
    "path/filepath"
    "sort"
    "strings"
    "sync"
    "time"
)

var (
    ErrCheckpointNotFound  = errors.New("checkpoint not found")
    ErrNoCheckpointsForRun = errors.New("no checkpoints found for run")
    ErrStoreInUse          = errors.New("checkpoint store already in use")
)

// FileCheckpointStore provides file-based checkpoint persistence.
// Checkpoints are stored as JSON files in a directory structure.
type FileCheckpointStore struct {
    dir string
    mu  sync.RWMutex // Protects concurrent access within process
}

// FileCheckpointStoreOptions configures the file checkpoint store.
type FileCheckpointStoreOptions struct {
    // Dir is the directory path for storing checkpoints.
    Dir string
}

// NewFileCheckpointStore creates a new file-based checkpoint store.
func NewFileCheckpointStore(opts FileCheckpointStoreOptions) (*FileCheckpointStore, error) {
    if opts.Dir == "" {
        return nil, errors.New("directory path is required")
    }
    
    // Create directory if it doesn't exist
    if err := os.MkdirAll(opts.Dir, 0755); err != nil {
        return nil, fmt.Errorf("create checkpoint directory: %w", err)
    }
    
    return &FileCheckpointStore{dir: opts.Dir}, nil
}

// Save persists a checkpoint to a file.
func (s *FileCheckpointStore) Save(ctx context.Context, checkpoint *Checkpoint) error {
    s.mu.Lock()
    defer s.mu.Unlock()
    
    if err := ctx.Err(); err != nil {
        return err
    }
    
    data, err := json.MarshalIndent(checkpoint, "", "  ")
    if err != nil {
        return fmt.Errorf("marshal checkpoint: %w", err)
    }
    
    path := s.checkpointPath(checkpoint.RunID, checkpoint.ID)
    if err := s.atomicWrite(path, data); err != nil {
        return fmt.Errorf("write checkpoint: %w", err)
    }
    
    return nil
}

// Load retrieves a checkpoint by ID.
func (s *FileCheckpointStore) Load(ctx context.Context, checkpointID string) (*Checkpoint, error) {
    s.mu.RLock()
    defer s.mu.RUnlock()
    
    if err := ctx.Err(); err != nil {
        return nil, err
    }
    
    // Search for checkpoint file (need to scan since we don't know runID)
    pattern := filepath.Join(s.dir, "*_"+checkpointID+".json")
    matches, err := filepath.Glob(pattern)
    if err != nil {
        return nil, fmt.Errorf("search for checkpoint: %w", err)
    }
    if len(matches) == 0 {
        return nil, fmt.Errorf("%w: %s", ErrCheckpointNotFound, checkpointID)
    }
    
    return s.loadFromFile(matches[0])
}

// LoadLatest retrieves the most recent checkpoint for a run.
func (s *FileCheckpointStore) LoadLatest(ctx context.Context, runID string) (*Checkpoint, error) {
    s.mu.RLock()
    defer s.mu.RUnlock()
    
    if err := ctx.Err(); err != nil {
        return nil, err
    }
    
    checkpoints, err := s.listForRun(runID)
    if err != nil {
        return nil, err
    }
    if len(checkpoints) == 0 {
        return nil, fmt.Errorf("%w: %s", ErrNoCheckpointsForRun, runID)
    }
    
    // Sort by creation time (newest last)
    sort.Slice(checkpoints, func(i, j int) bool {
        return checkpoints[i].CreatedAt.Before(checkpoints[j].CreatedAt)
    })
    
    return checkpoints[len(checkpoints)-1], nil
}

// Delete removes a checkpoint.
func (s *FileCheckpointStore) Delete(ctx context.Context, checkpointID string) error {
    s.mu.Lock()
    defer s.mu.Unlock()
    
    if err := ctx.Err(); err != nil {
        return err
    }
    
    pattern := filepath.Join(s.dir, "*_"+checkpointID+".json")
    matches, err := filepath.Glob(pattern)
    if err != nil {
        return fmt.Errorf("search for checkpoint: %w", err)
    }
    
    for _, path := range matches {
        if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
            return fmt.Errorf("delete checkpoint file: %w", err)
        }
    }
    
    return nil
}

// List returns all checkpoints for a run.
func (s *FileCheckpointStore) List(ctx context.Context, runID string) ([]*Checkpoint, error) {
    s.mu.RLock()
    defer s.mu.RUnlock()
    
    if err := ctx.Err(); err != nil {
        return nil, err
    }
    
    return s.listForRun(runID)
}

func (s *FileCheckpointStore) checkpointPath(runID, checkpointID string) string {
    filename := fmt.Sprintf("%s_%s.json", runID, checkpointID)
    return filepath.Join(s.dir, filename)
}

func (s *FileCheckpointStore) atomicWrite(path string, data []byte) error {
    dir := filepath.Dir(path)
    
    tmp, err := os.CreateTemp(dir, ".checkpoint-*.tmp")
    if err != nil {
        return fmt.Errorf("create temp file: %w", err)
    }
    tmpPath := tmp.Name()
    
    success := false
    defer func() {
        if !success {
            os.Remove(tmpPath)
        }
    }()
    
    if _, err := tmp.Write(data); err != nil {
        tmp.Close()
        return fmt.Errorf("write temp file: %w", err)
    }
    if err := tmp.Sync(); err != nil {
        tmp.Close()
        return fmt.Errorf("sync temp file: %w", err)
    }
    if err := tmp.Close(); err != nil {
        return fmt.Errorf("close temp file: %w", err)
    }
    
    if err := os.Rename(tmpPath, path); err != nil {
        return fmt.Errorf("rename to final path: %w", err)
    }
    
    success = true
    return nil
}

func (s *FileCheckpointStore) loadFromFile(path string) (*Checkpoint, error) {
    data, err := os.ReadFile(path)
    if err != nil {
        return nil, fmt.Errorf("read checkpoint file: %w", err)
    }
    
    var cp Checkpoint
    if err := json.Unmarshal(data, &cp); err != nil {
        return nil, fmt.Errorf("unmarshal checkpoint: %w", err)
    }
    
    return &cp, nil
}

func (s *FileCheckpointStore) listForRun(runID string) ([]*Checkpoint, error) {
    pattern := filepath.Join(s.dir, runID+"_*.json")
    matches, err := filepath.Glob(pattern)
    if err != nil {
        return nil, fmt.Errorf("list checkpoint files: %w", err)
    }
    
    result := make([]*Checkpoint, 0, len(matches))
    for _, path := range matches {
        cp, err := s.loadFromFile(path)
        if err != nil {
            // Log warning but continue
            continue
        }
        result = append(result, cp)
    }
    
    return result, nil
}
```

---

## 10. Test Strategy

### Unit Tests

```go
func TestFileCheckpointStore_Save(t *testing.T) {
    // Create temp directory
    dir := t.TempDir()
    
    store, err := NewFileCheckpointStore(FileCheckpointStoreOptions{Dir: dir})
    require.NoError(t, err)
    
    cp := &Checkpoint{
        ID:        "cp-1",
        RunID:     "run-1",
        Superstep: 1,
        CreatedAt: time.Now(),
    }
    
    err = store.Save(context.Background(), cp)
    require.NoError(t, err)
    
    // Verify file exists
    path := filepath.Join(dir, "run-1_cp-1.json")
    _, err = os.Stat(path)
    require.NoError(t, err)
}

func TestFileCheckpointStore_Load(t *testing.T) {
    // Similar setup, save then load
}

func TestFileCheckpointStore_LoadLatest(t *testing.T) {
    // Save multiple checkpoints with different timestamps
    // Verify LoadLatest returns the most recent
}

func TestFileCheckpointStore_Delete(t *testing.T) {
    // Save, delete, verify file is gone
}

func TestFileCheckpointStore_List(t *testing.T) {
    // Save multiple checkpoints for same run
    // Verify List returns all
}

func TestFileCheckpointStore_AtomicWrite(t *testing.T) {
    // Verify temp file is cleaned up on failure
    // Verify final file contains correct data
}

func TestFileCheckpointStore_ConcurrentAccess(t *testing.T) {
    // Run multiple goroutines saving/loading
    // Verify no data corruption
}

func TestFileCheckpointStore_ContextCancellation(t *testing.T) {
    // Verify operations respect context cancellation
}
```

### Integration Tests

```go
func TestFileCheckpointStore_LargeCheckpoint(t *testing.T) {
    // Test with large state data (MB+)
}

func TestFileCheckpointStore_ManyCheckpoints(t *testing.T) {
    // Test with hundreds of checkpoints
    // Verify List performance
}

func TestFileCheckpointStore_InvalidJSON(t *testing.T) {
    // Manually write invalid JSON to file
    // Verify Load returns appropriate error
}
```

---

## 11. Implementation Recommendations

### Priority Order

1. **Core implementation** (Save, Load, LoadLatest, Delete, List)
2. **Atomic write with temp file**
3. **Basic error handling**
4. **Unit tests**
5. **Optional: Index file for faster listing**
6. **Optional: File locking for multi-process safety**
7. **Optional: Pruning/cleanup API**

### Dependencies

- `os` - File operations
- `filepath` - Path manipulation
- `encoding/json` - Serialization
- `sync` - Mutex for thread safety
- `time` - Timestamps
- `errors` - Error wrapping

### No External Dependencies Required

The implementation can be done entirely with Go standard library.

---

## References

- [Python FileCheckpointStorage](python/packages/core/agent_framework/_workflows/_checkpoint.py)
- [.NET FileSystemJsonCheckpointStore](dotnet/src/Microsoft.Agents.AI.Workflows/Checkpointing/FileSystemJsonCheckpointStore.cs)
- [Go InMemoryCheckpointStore](go/workflow/checkpoint.go)
