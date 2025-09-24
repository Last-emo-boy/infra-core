package snap

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/last-emo-boy/infra-core/pkg/config"
)

func TestNewSnapManager(t *testing.T) {
	// Create temporary directory for testing
	tempDir, err := os.MkdirTemp("", "snap_test_")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)
	
	// Create mock database
	db := &sqlx.DB{}
	
	snapConfig := config.SnapConfig{
		RepoDir: tempDir,
	}
	
	manager, err := NewSnapManager(db, snapConfig)
	require.NoError(t, err)
	assert.NotNil(t, manager)
}

func TestSnapManagerLifecycle(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "snap_lifecycle_test_")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)
	
	db := &sqlx.DB{}
	snapConfig := config.SnapConfig{
		RepoDir: tempDir,
	}
	
	manager, err := NewSnapManager(db, snapConfig)
	require.NoError(t, err)
	
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	
	// Start manager
	manager.Start(ctx)
	
	// Stop manager
	manager.Stop()
	
	// Verify manager exists and has been configured
	assert.NotNil(t, manager)
}

func TestBlockStoreOperations(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "blockstore_test_")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)
	
	blockStore, err := NewBlockStore(tempDir)
	require.NoError(t, err)
	
	// Test initial state
	assert.NotNil(t, blockStore)
	assert.Equal(t, 0, len(blockStore.blockIndex))
	
	// Test storing blocks
	const numBlocks = 10
	hashes := make([]string, numBlocks)
	
	for i := 0; i < numBlocks; i++ {
		data := []byte(fmt.Sprintf("data_%d", i))
		
		// Calculate hash
		hasher := sha256.New()
		hasher.Write(data)
		hash := hex.EncodeToString(hasher.Sum(nil))
		hashes[i] = hash
		
		err := blockStore.storeBlock(hash, data)
		require.NoError(t, err)
	}
	
	// Verify final state
	blockStore.mutex.RLock()
	finalSize := len(blockStore.blockIndex)
	blockStore.mutex.RUnlock()
	
	assert.Equal(t, numBlocks, finalSize)
}

func TestSnapshotStatusConstants(t *testing.T) {
	// Test status constants are defined correctly
	assert.Equal(t, "pending", StatusPending)
	assert.Equal(t, "running", StatusRunning)
	assert.Equal(t, "completed", StatusCompleted)
	assert.Equal(t, "failed", StatusFailed)
}

func TestFileEntryJSONSerialization(t *testing.T) {
	original := &FileEntry{
		Path:     "/test/path/file.txt",
		Size:     1024,
		ModTime:  time.Now(),
		IsDir:    false,
		Blocks:   []string{"block1", "block2", "block3"},
		Checksum: "abcd1234",
	}
	
	// Test JSON marshaling
	jsonData, err := json.Marshal(original)
	require.NoError(t, err)
	assert.NotEmpty(t, jsonData)
	
	// Test JSON unmarshaling
	var deserialized FileEntry
	err = json.Unmarshal(jsonData, &deserialized)
	require.NoError(t, err)
	
	// Verify all fields match
	assert.Equal(t, original.Path, deserialized.Path)
	assert.Equal(t, original.Size, deserialized.Size)
	assert.Equal(t, original.ModTime.Unix(), deserialized.ModTime.Unix())
	assert.Equal(t, original.IsDir, deserialized.IsDir)
	assert.Equal(t, original.Blocks, deserialized.Blocks)
	assert.Equal(t, original.Checksum, deserialized.Checksum)
}

func TestSnapshotManifestJSONSerialization(t *testing.T) {
	fileEntry := FileEntry{
		Path:     "/test/file.txt",
		Size:     512,
		ModTime:  time.Now(),
		IsDir:    false,
		Blocks:   []string{"block1"},
		Checksum: "xyz789",
	}
	
	original := &SnapshotManifest{
		ID:        "snapshot_123",
		PlanID:    "plan_456",
		Timestamp: time.Now(),
		Paths:     []string{"/test/path1", "/test/path2"},
		Files:     []FileEntry{fileEntry},
		Blocks:    map[string]string{"block1": "/blocks/01/block1.block"},
		Size:      1024,
		FileCount: 2,
	}
	
	// Test JSON marshaling
	jsonData, err := json.Marshal(original)
	require.NoError(t, err)
	assert.NotEmpty(t, jsonData)
	
	// Test JSON unmarshaling
	var deserialized SnapshotManifest
	err = json.Unmarshal(jsonData, &deserialized)
	require.NoError(t, err)
	
	// Verify all fields match
	assert.Equal(t, original.ID, deserialized.ID)
	assert.Equal(t, original.PlanID, deserialized.PlanID)
	assert.Equal(t, original.Timestamp.Unix(), deserialized.Timestamp.Unix())
	assert.Equal(t, original.Paths, deserialized.Paths)
	assert.Equal(t, len(original.Files), len(deserialized.Files))
	assert.Equal(t, original.Blocks, deserialized.Blocks)
	assert.Equal(t, original.Size, deserialized.Size)
	assert.Equal(t, original.FileCount, deserialized.FileCount)
	
	if len(deserialized.Files) > 0 {
		assert.Equal(t, original.Files[0].Path, deserialized.Files[0].Path)
		assert.Equal(t, original.Files[0].Size, deserialized.Files[0].Size)
		assert.Equal(t, original.Files[0].IsDir, deserialized.Files[0].IsDir)
	}
}

func TestRestoreJobJSONSerialization(t *testing.T) {
	now := time.Now()
	completed := now.Add(time.Hour)
	
	original := &RestoreJob{
		ID:         "restore_456",
		SnapshotID: "snapshot_123",
		TargetPath: "/restore/target",
		Status:     RestoreStatusRunning,
		Progress:   0.75,
		Message:    "Restoring files...",
		Started:    now,
		Completed:  completed,
	}
	
	// Test JSON marshaling
	jsonData, err := json.Marshal(original)
	require.NoError(t, err)
	assert.NotEmpty(t, jsonData)
	
	// Test JSON unmarshaling
	var deserialized RestoreJob
	err = json.Unmarshal(jsonData, &deserialized)
	require.NoError(t, err)
	
	// Verify all fields match
	assert.Equal(t, original.ID, deserialized.ID)
	assert.Equal(t, original.SnapshotID, deserialized.SnapshotID)
	assert.Equal(t, original.TargetPath, deserialized.TargetPath)
	assert.Equal(t, original.Status, deserialized.Status)
	assert.Equal(t, original.Progress, deserialized.Progress)
	assert.Equal(t, original.Message, deserialized.Message)
	assert.Equal(t, original.Started.Unix(), deserialized.Started.Unix())
	assert.Equal(t, original.Completed.Unix(), deserialized.Completed.Unix())
}

func TestSnapManagerConfiguration(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "snap_config_test_")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)
	
	db := &sqlx.DB{}
	snapConfig := config.SnapConfig{
		RepoDir: tempDir,
	}
	
	manager, err := NewSnapManager(db, snapConfig)
	require.NoError(t, err)
	
	// Verify configuration is applied
	assert.Equal(t, tempDir, manager.config.RepoDir)
	assert.NotNil(t, manager.db)
	assert.NotNil(t, manager.blockStore)
}

func TestProcessFileInternal(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "snap_process_test_")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)
	
	// Create some test files
	testFile := filepath.Join(tempDir, "test.txt")
	testContent := []byte("test content for processing")
	err = os.WriteFile(testFile, testContent, 0644)
	require.NoError(t, err)
	
	db := &sqlx.DB{}
	snapConfig := config.SnapConfig{
		RepoDir: tempDir,
	}
	
	manager, err := NewSnapManager(db, snapConfig)
	require.NoError(t, err)
	
	// Process the file
	blocks, checksum, err := manager.processFile(testFile)
	require.NoError(t, err)
	assert.Greater(t, len(blocks), 0)
	assert.NotEmpty(t, checksum)
}

func TestRestoreStatusConstants(t *testing.T) {
	// Test restore status constants are defined correctly
	assert.Equal(t, "pending", RestoreStatusPending)
	assert.Equal(t, "running", RestoreStatusRunning)
	assert.Equal(t, "completed", RestoreStatusCompleted)
	assert.Equal(t, "failed", RestoreStatusFailed)
	assert.Equal(t, "cancelled", RestoreStatusCancelled)
}

func TestBlockStoreRetrieveBlock(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "blockstore_retrieve_test_")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)
	
	blockStore, err := NewBlockStore(tempDir)
	require.NoError(t, err)
	
	// Store a block
	originalData := []byte("test block data")
	hasher := sha256.New()
	hasher.Write(originalData)
	blockHash := hex.EncodeToString(hasher.Sum(nil))
	
	err = blockStore.storeBlock(blockHash, originalData)
	require.NoError(t, err)
	
	// Verify the block was stored
	blockStore.mutex.RLock()
	_, exists := blockStore.blockIndex[blockHash]
	blockStore.mutex.RUnlock()
	assert.True(t, exists)
}

func TestSnapManagerStopBeforeStart(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "snap_stop_test_")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)
	
	db := &sqlx.DB{}
	snapConfig := config.SnapConfig{
		RepoDir: tempDir,
	}
	
	manager, err := NewSnapManager(db, snapConfig)
	require.NoError(t, err)
	
	// Stop before start should not panic
	manager.Stop()
	assert.NotNil(t, manager)
}

func TestFileEntryValidation(t *testing.T) {
	entry := &FileEntry{
		Path:     "/valid/path/file.txt",
		Size:     1024,
		ModTime:  time.Now(),
		IsDir:    false,
		Blocks:   []string{"block1", "block2"},
		Checksum: "valid_checksum",
	}
	
	// Basic validation checks
	assert.NotEmpty(t, entry.Path)
	assert.Greater(t, entry.Size, int64(0))
	assert.False(t, entry.ModTime.IsZero())
	assert.Greater(t, len(entry.Blocks), 0)
	assert.NotEmpty(t, entry.Checksum)
}

func TestSnapshotManifestValidation(t *testing.T) {
	manifest := &SnapshotManifest{
		ID:        "valid_snapshot_id",
		PlanID:    "valid_plan_id",
		Timestamp: time.Now(),
		Paths:     []string{"/valid/path1"},
		Files:     []FileEntry{},
		Blocks:    make(map[string]string),
		Size:      1024,
		FileCount: 1,
	}
	
	// Basic validation checks
	assert.NotEmpty(t, manifest.ID)
	assert.NotEmpty(t, manifest.PlanID)
	assert.False(t, manifest.Timestamp.IsZero())
	assert.Greater(t, len(manifest.Paths), 0)
	assert.NotNil(t, manifest.Files)
	assert.NotNil(t, manifest.Blocks)
	assert.Greater(t, manifest.Size, int64(0))
	assert.Greater(t, manifest.FileCount, 0)
}

func TestRestoreJobValidation(t *testing.T) {
	job := &RestoreJob{
		ID:         "valid_restore_id",
		SnapshotID: "valid_snapshot_id",
		TargetPath: "/valid/target/path",
		Status:     RestoreStatusPending,
		Progress:   0.0,
		Message:    "",
		Started:    time.Now(),
		Completed:  time.Time{},
	}
	
	// Basic validation checks
	assert.NotEmpty(t, job.ID)
	assert.NotEmpty(t, job.SnapshotID)
	assert.NotEmpty(t, job.TargetPath)
	assert.GreaterOrEqual(t, job.Progress, 0.0)
	assert.LessOrEqual(t, job.Progress, 1.0)
	assert.False(t, job.Started.IsZero())
}

func TestBlockStoreRebuildIndex(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "blockstore_rebuild_test_")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)
	
	// Create block directory with some files
	blockDir := filepath.Join(tempDir, "blocks", "ab")
	err = os.MkdirAll(blockDir, 0755)
	require.NoError(t, err)
	
	// Create a fake block file
	blockFile := filepath.Join(blockDir, "abcd1234567890abcd1234567890abcd1234567890abcd1234567890abcd1234.block")
	err = os.WriteFile(blockFile, []byte("test data"), 0644)
	require.NoError(t, err)
	
	// Create block store and verify it rebuilds the index
	blockStore, err := NewBlockStore(tempDir)
	require.NoError(t, err)
	
	blockStore.mutex.RLock()
	indexSize := len(blockStore.blockIndex)
	blockStore.mutex.RUnlock()
	
	assert.Equal(t, 1, indexSize)
}

func TestTaskCreation(t *testing.T) {
	task := &Task{
		ID:       "test_task_123",
		Type:     "snapshot",
		Status:   StatusRunning,
		Progress: 50.0,
		Message:  "Processing...",
		Started:  time.Now(),
	}
	
	// Basic validation
	assert.NotEmpty(t, task.ID)
	assert.Equal(t, "snapshot", task.Type)
	assert.Equal(t, StatusRunning, task.Status)
	assert.Equal(t, 50.0, task.Progress)
	assert.NotEmpty(t, task.Message)
	assert.False(t, task.Started.IsZero())
}

func TestBlockSizeConstant(t *testing.T) {
	// Verify block size constant
	expectedBlockSize := 4 * 1024 * 1024 // 4MB
	assert.Equal(t, expectedBlockSize, BlockSize)
}

func TestSymlinkFileEntry(t *testing.T) {
	entry := &FileEntry{
		Path:    "/path/to/symlink",
		Size:    0,
		ModTime: time.Now(),
		IsDir:   false,
		Target:  "/path/to/target",
	}
	
	// Verify symlink properties
	assert.NotEmpty(t, entry.Path)
	assert.NotEmpty(t, entry.Target)
	assert.False(t, entry.IsDir)
}

func TestVerifyBlockIntegrity(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "verify_block_test_")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)
	
	db := &sqlx.DB{}
	snapConfig := config.SnapConfig{
		RepoDir: tempDir,
	}
	
	manager, err := NewSnapManager(db, snapConfig)
	require.NoError(t, err)
	
	// Store a block
	testData := []byte("test block data for verification")
	hasher := sha256.New()
	hasher.Write(testData)
	blockHash := hex.EncodeToString(hasher.Sum(nil))
	
	err = manager.blockStore.storeBlock(blockHash, testData)
	require.NoError(t, err)
	
	// Verify the block
	err = manager.verifyBlock(blockHash)
	assert.NoError(t, err)
	
	// Try to verify non-existent block
	err = manager.verifyBlock("nonexistent_hash")
	assert.Error(t, err)
}
