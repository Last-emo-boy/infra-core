package snap

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/last-emo-boy/infra-core/pkg/config"
)

const (
	BlockSize = 4 * 1024 * 1024 // 4MB blocks

	// Snapshot status
	StatusPending   = "pending"
	StatusRunning   = "running"
	StatusCompleted = "completed"
	StatusFailed    = "failed"

	// Restore status
	RestoreStatusPending   = "pending"
	RestoreStatusRunning   = "running"
	RestoreStatusCompleted = "completed"
	RestoreStatusFailed    = "failed"
	RestoreStatusCancelled = "cancelled"
)

// SnapManager manages snapshots and restore operations
type SnapManager struct {
	db           *sqlx.DB
	config       config.SnapConfig
	blockStore   *BlockStore
	runningTasks map[string]*Task
	taskMutex    sync.RWMutex
	ctx          context.Context
	cancel       context.CancelFunc
}

// BlockStore manages deduplicated blocks
type BlockStore struct {
	repoDir    string
	blockIndex map[string]string // hash -> filepath
	mutex      sync.RWMutex
}

// Task represents a running operation
type Task struct {
	ID       string
	Type     string // "snapshot" or "restore"
	Status   string
	Progress float64
	Message  string
	Started  time.Time
	ctx      context.Context
	cancel   context.CancelFunc
}

// SnapshotManifest represents the structure of a snapshot
type SnapshotManifest struct {
	ID        string            `json:"id"`
	PlanID    string            `json:"plan_id"`
	Timestamp time.Time         `json:"timestamp"`
	Paths     []string          `json:"paths"`
	Files     []FileEntry       `json:"files"`
	Blocks    map[string]string `json:"blocks"` // hash -> relative path in repo
	Size      int64             `json:"size"`
	FileCount int               `json:"file_count"`
}

// FileEntry represents a file in the snapshot
type FileEntry struct {
	Path     string    `json:"path"`
	Size     int64     `json:"size"`
	Mode     uint32    `json:"mode"`
	ModTime  time.Time `json:"mod_time"`
	IsDir    bool      `json:"is_dir"`
	Blocks   []string  `json:"blocks,omitempty"` // block hashes for files
	Target   string    `json:"target,omitempty"` // symlink target
	Checksum string    `json:"checksum,omitempty"`
}

// RestoreJob represents a restore operation
type RestoreJob struct {
	ID         string    `json:"id"`
	SnapshotID string    `json:"snapshot_id"`
	TargetPath string    `json:"target_path"`
	Status     string    `json:"status"`
	Progress   float64   `json:"progress"`
	Message    string    `json:"message"`
	Started    time.Time `json:"started"`
	Completed  time.Time `json:"completed,omitempty"`
}

// NewSnapManager creates a new snap manager
func NewSnapManager(db *sqlx.DB, config config.SnapConfig) (*SnapManager, error) {
	blockStore, err := NewBlockStore(config.RepoDir)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize block store: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &SnapManager{
		db:           db,
		config:       config,
		blockStore:   blockStore,
		runningTasks: make(map[string]*Task),
		ctx:          ctx,
		cancel:       cancel,
	}, nil
}

// NewBlockStore creates a new block store
func NewBlockStore(repoDir string) (*BlockStore, error) {
	blockDir := filepath.Join(repoDir, "blocks")
	if err := os.MkdirAll(blockDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create block directory: %w", err)
	}

	bs := &BlockStore{
		repoDir:    repoDir,
		blockIndex: make(map[string]string),
	}

	// Load existing blocks
	if err := bs.rebuildIndex(); err != nil {
		return nil, fmt.Errorf("failed to rebuild block index: %w", err)
	}

	return bs, nil
}

// rebuildIndex scans the block directory and rebuilds the index
func (bs *BlockStore) rebuildIndex() error {
	blockDir := filepath.Join(bs.repoDir, "blocks")
	
	return filepath.Walk(blockDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() && strings.HasSuffix(path, ".block") {
			// Extract hash from filename
			filename := filepath.Base(path)
			hash := strings.TrimSuffix(filename, ".block")
			if len(hash) == 64 { // SHA-256 hex length
				bs.blockIndex[hash] = path
			}
		}

		return nil
	})
}

// Start starts the snap manager background tasks
func (sm *SnapManager) Start(ctx context.Context) {
	// Start scheduled snapshots
	go sm.scheduleRunner(ctx)
	
	// Start scrub runner
	go sm.scrubRunner(ctx)
}

// Stop stops the snap manager
func (sm *SnapManager) Stop() {
	sm.cancel()
	
	// Cancel all running tasks
	sm.taskMutex.Lock()
	for _, task := range sm.runningTasks {
		task.cancel()
	}
	sm.taskMutex.Unlock()
}

// scheduleRunner runs scheduled backup plans
func (sm *SnapManager) scheduleRunner(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			sm.checkScheduledPlans()
		}
	}
}

// scrubRunner performs periodic integrity checks
func (sm *SnapManager) scrubRunner(ctx context.Context) {
	ticker := time.NewTicker(24 * time.Hour) // Daily scrub
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			sm.performScrub()
		}
	}
}

// checkScheduledPlans checks for plans that need to run
func (sm *SnapManager) checkScheduledPlans() {
	rows, err := sm.db.Query(`
		SELECT id, name, cron_expression, paths, enabled
		FROM snap_plans 
		WHERE enabled = 1
	`)
	if err != nil {
		return
	}
	defer rows.Close()

	for rows.Next() {
		var id, name, cronExpr, paths string
		var enabled bool
		
		if err := rows.Scan(&id, &name, &cronExpr, &paths, &enabled); err != nil {
			continue
		}

		// Check if plan should run now (simplified cron check)
		if sm.shouldRunPlan(id, cronExpr) {
			go sm.executeScheduledSnapshot(id, name, strings.Split(paths, ","))
		}
	}
}

// shouldRunPlan checks if a plan should run based on its schedule
func (sm *SnapManager) shouldRunPlan(planID, cronExpr string) bool {
	// Simplified check - check if last run was more than the interval
	// In a full implementation, you'd use a proper cron parser
	
	var lastRun sql.NullTime
	err := sm.db.QueryRow(`
		SELECT MAX(timestamp) FROM snapshots WHERE plan_id = ?
	`, planID).Scan(&lastRun)
	
	if err != nil || !lastRun.Valid {
		return true // First run
	}

	// Simple daily check for now
	return time.Since(lastRun.Time) > 24*time.Hour
}

// executeScheduledSnapshot executes a scheduled snapshot
func (sm *SnapManager) executeScheduledSnapshot(planID, name string, paths []string) {
	snapshotID := fmt.Sprintf("snap_%d", time.Now().Unix())
	
	task := &Task{
		ID:      snapshotID,
		Type:    "snapshot",
		Status:  StatusRunning,
		Started: time.Now(),
	}
	
	ctx, cancel := context.WithCancel(sm.ctx)
	task.ctx = ctx
	task.cancel = cancel

	sm.taskMutex.Lock()
	sm.runningTasks[snapshotID] = task
	sm.taskMutex.Unlock()

	defer func() {
		sm.taskMutex.Lock()
		delete(sm.runningTasks, snapshotID)
		sm.taskMutex.Unlock()
	}()

	err := sm.createSnapshotInternal(ctx, snapshotID, planID, paths, task)
	if err != nil {
		task.Status = StatusFailed
		task.Message = err.Error()
	} else {
		task.Status = StatusCompleted
		task.Progress = 100.0
	}
}

// createSnapshotInternal creates a snapshot with progress tracking
func (sm *SnapManager) createSnapshotInternal(ctx context.Context, snapshotID, planID string, paths []string, task *Task) error {
	manifest := &SnapshotManifest{
		ID:        snapshotID,
		PlanID:    planID,
		Timestamp: time.Now(),
		Paths:     paths,
		Files:     []FileEntry{},
		Blocks:    make(map[string]string),
		Size:      0,
		FileCount: 0,
	}

	// Phase 1: Scan files
	task.Message = "Scanning files..."
	totalFiles := 0
	var allFiles []string

	for _, path := range paths {
		err := filepath.Walk(path, func(filePath string, info os.FileInfo, err error) error {
			if err != nil {
				return nil // Skip errors, continue scanning
			}
			allFiles = append(allFiles, filePath)
			totalFiles++
			return nil
		})
		if err != nil {
			return fmt.Errorf("failed to scan path %s: %w", path, err)
		}
	}

	task.Message = fmt.Sprintf("Found %d files", totalFiles)
	processedFiles := 0

	// Phase 2: Process files
	for _, filePath := range allFiles {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		info, err := os.Lstat(filePath)
		if err != nil {
			continue // Skip files that can't be stat'd
		}

		fileEntry := FileEntry{
			Path:    filePath,
			Size:    info.Size(),
			Mode:    uint32(info.Mode()),
			ModTime: info.ModTime(),
			IsDir:   info.IsDir(),
		}

		if info.Mode()&os.ModeSymlink != 0 {
			// Handle symlink
			target, err := os.Readlink(filePath)
			if err == nil {
				fileEntry.Target = target
			}
		} else if !info.IsDir() {
			// Handle regular file - create blocks
			blocks, checksum, err := sm.processFile(filePath)
			if err != nil {
				continue // Skip files that can't be processed
			}
			fileEntry.Blocks = blocks
			fileEntry.Checksum = checksum
			
			// Add blocks to manifest
			for _, blockHash := range blocks {
				if blockPath, exists := sm.blockStore.blockIndex[blockHash]; exists {
					manifest.Blocks[blockHash] = blockPath
				}
			}
		}

		manifest.Files = append(manifest.Files, fileEntry)
		manifest.Size += fileEntry.Size
		manifest.FileCount++

		processedFiles++
		task.Progress = float64(processedFiles) / float64(totalFiles) * 100.0
		task.Message = fmt.Sprintf("Processed %d/%d files", processedFiles, totalFiles)
	}

	// Save manifest
	manifestPath := filepath.Join(sm.config.RepoDir, "manifests", snapshotID+".json")
	if err := os.MkdirAll(filepath.Dir(manifestPath), 0755); err != nil {
		return fmt.Errorf("failed to create manifest directory: %w", err)
	}

	manifestData, err := json.Marshal(manifest)
	if err != nil {
		return fmt.Errorf("failed to marshal manifest: %w", err)
	}

	if err := os.WriteFile(manifestPath, manifestData, 0644); err != nil {
		return fmt.Errorf("failed to write manifest: %w", err)
	}

	// Save to database
	_, err = sm.db.Exec(`
		INSERT INTO snapshots (id, plan_id, timestamp, manifest_path, size_bytes, status)
		VALUES (?, ?, ?, ?, ?, ?)
	`, snapshotID, planID, manifest.Timestamp, manifestPath, manifest.Size, StatusCompleted)

	if err != nil {
		return fmt.Errorf("failed to save snapshot to database: %w", err)
	}

	return nil
}

// processFile processes a file into blocks
func (sm *SnapManager) processFile(filePath string) ([]string, string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, "", err
	}
	defer file.Close()

	var blocks []string
	hasher := sha256.New()
	buffer := make([]byte, BlockSize)

	for {
		n, err := file.Read(buffer)
		if n == 0 {
			break
		}
		if err != nil && err != io.EOF {
			return nil, "", err
		}

		block := buffer[:n]
		hasher.Write(block)

		// Calculate block hash
		blockHasher := sha256.New()
		blockHasher.Write(block)
		blockHash := hex.EncodeToString(blockHasher.Sum(nil))

		// Store block if not exists
		if err := sm.blockStore.storeBlock(blockHash, block); err != nil {
			return nil, "", err
		}

		blocks = append(blocks, blockHash)

		if err == io.EOF {
			break
		}
	}

	checksum := hex.EncodeToString(hasher.Sum(nil))
	return blocks, checksum, nil
}

// storeBlock stores a block in the block store
func (bs *BlockStore) storeBlock(hash string, data []byte) error {
	bs.mutex.Lock()
	defer bs.mutex.Unlock()

	// Check if block already exists
	if _, exists := bs.blockIndex[hash]; exists {
		return nil // Block already stored
	}

	// Create block file path
	blockDir := filepath.Join(bs.repoDir, "blocks", hash[:2])
	if err := os.MkdirAll(blockDir, 0755); err != nil {
		return err
	}

	blockPath := filepath.Join(blockDir, hash+".block")
	
	// Write block data
	if err := os.WriteFile(blockPath, data, 0644); err != nil {
		return err
	}

	// Update index
	bs.blockIndex[hash] = blockPath
	return nil
}

// performScrub performs integrity checking
func (sm *SnapManager) performScrub() {
	// Sample 10% of blocks for verification
	sm.blockStore.mutex.RLock()
	var hashes []string
	for hash := range sm.blockStore.blockIndex {
		hashes = append(hashes, hash)
	}
	sm.blockStore.mutex.RUnlock()

	// Sort and sample
	sort.Strings(hashes)
	sampleSize := len(hashes) / 10
	if sampleSize < 1 {
		sampleSize = len(hashes)
	}

	for i := 0; i < sampleSize; i++ {
		hash := hashes[i*10/sampleSize]
		if err := sm.verifyBlock(hash); err != nil {
			// Log error or trigger alert
			fmt.Printf("Block verification failed for %s: %v\n", hash, err)
		}
	}
}

// verifyBlock verifies a block's integrity
func (sm *SnapManager) verifyBlock(hash string) error {
	sm.blockStore.mutex.RLock()
	blockPath, exists := sm.blockStore.blockIndex[hash]
	sm.blockStore.mutex.RUnlock()

	if !exists {
		return fmt.Errorf("block not found in index")
	}

	data, err := os.ReadFile(blockPath)
	if err != nil {
		return fmt.Errorf("failed to read block: %w", err)
	}

	// Verify hash
	hasher := sha256.New()
	hasher.Write(data)
	computedHash := hex.EncodeToString(hasher.Sum(nil))

	if computedHash != hash {
		return fmt.Errorf("hash mismatch: expected %s, got %s", hash, computedHash)
	}

	return nil
}