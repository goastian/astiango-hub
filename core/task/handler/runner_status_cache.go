package handler

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/goastian/astiango-hub/core/models/client"
	"github.com/goastian/astiango-hub/core/models/models"
	"github.com/goastian/astiango-hub/core/models/service"
	"github.com/goastian/astiango-hub/core/utils"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// TaskStatusSnapshot represents a point-in-time status of a task for caching
type TaskStatusSnapshot struct {
	TaskId    primitive.ObjectID `json:"task_id"`
	Status    string             `json:"status"`
	Error     string             `json:"error,omitempty"`
	Pid       int                `json:"pid,omitempty"`
	Timestamp time.Time          `json:"timestamp"`
	StartedAt *time.Time         `json:"started_at,omitempty"`
	EndedAt   *time.Time         `json:"ended_at,omitempty"`
}

// TaskStatusCache manages local task status storage for disconnection resilience
type TaskStatusCache struct {
	mu        sync.RWMutex
	snapshots map[primitive.ObjectID]*TaskStatusSnapshot
	filePath  string
	dirty     bool // tracks if cache needs to be persisted
}

func (r *Runner) initStatusCache() error {
	cacheDir := filepath.Join(utils.GetWorkspace(), ".astiango-hub", "task_cache")
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return fmt.Errorf("failed to create cache directory: %w", err)
	}

	r.statusCache = &TaskStatusCache{
		snapshots: make(map[primitive.ObjectID]*TaskStatusSnapshot),
		filePath:  filepath.Join(cacheDir, fmt.Sprintf("task_%s.json", r.tid.Hex())),
		dirty:     false,
	}

	if err := r.loadStatusCache(); err != nil {
		r.Warnf("failed to load existing status cache: %v", err)
	}

	r.pendingUpdates = make([]TaskStatusSnapshot, 0)
	return nil
}

func (r *Runner) loadStatusCache() error {
	if _, err := os.Stat(r.statusCache.filePath); os.IsNotExist(err) {
		return nil
	}
	data, err := os.ReadFile(r.statusCache.filePath)
	if err != nil {
		return fmt.Errorf("failed to read cache file: %w", err)
	}
	var snapshots []TaskStatusSnapshot
	if err := json.Unmarshal(data, &snapshots); err != nil {
		return fmt.Errorf("failed to unmarshal cache data: %w", err)
	}
	r.statusCache.mu.Lock()
	defer r.statusCache.mu.Unlock()
	for _, snapshot := range snapshots {
		r.statusCache.snapshots[snapshot.TaskId] = &snapshot
	}
	r.Debugf("loaded %d task status snapshots from cache", len(snapshots))
	return nil
}

func (r *Runner) persistStatusCache() error {
	r.statusCache.mu.RLock()
	if !r.statusCache.dirty {
		r.statusCache.mu.RUnlock()
		return nil
	}
	snapshots := make([]TaskStatusSnapshot, 0, len(r.statusCache.snapshots))
	for _, snapshot := range r.statusCache.snapshots {
		snapshots = append(snapshots, *snapshot)
	}
	r.statusCache.mu.RUnlock()
	data, err := json.Marshal(snapshots)
	if err != nil {
		return fmt.Errorf("failed to marshal cache data: %w", err)
	}
	if err := os.WriteFile(r.statusCache.filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write cache file: %w", err)
	}
	r.statusCache.mu.Lock()
	r.statusCache.dirty = false
	r.statusCache.mu.Unlock()
	return nil
}

func (r *Runner) cacheTaskStatus(status string, err error) {
	snapshot := &TaskStatusSnapshot{
		TaskId:    r.tid,
		Status:    status,
		Pid:       r.pid,
		Timestamp: time.Now(),
	}
	if err != nil {
		snapshot.Error = err.Error()
	}
	// Store in cache
	r.statusCache.mu.Lock()
	r.statusCache.snapshots[r.tid] = snapshot
	r.statusCache.dirty = true
	r.statusCache.mu.Unlock()
	// Add to pending updates for sync when reconnected
	r.statusCacheMutex.Lock()
	r.pendingUpdates = append(r.pendingUpdates, *snapshot)
	r.statusCacheMutex.Unlock()
	go func() {
		if err := r.persistStatusCache(); err != nil {
			r.Errorf("failed to persist status cache: %v", err)
		}
	}()
	r.Debugf("cached task status: %s (pid: %d)", status, r.pid)
}

func (r *Runner) syncPendingStatusUpdates() error {
	r.statusCacheMutex.Lock()
	pendingCount := len(r.pendingUpdates)
	if pendingCount == 0 {
		r.statusCacheMutex.Unlock()
		return nil
	}
	updates := make([]TaskStatusSnapshot, pendingCount)
	copy(updates, r.pendingUpdates)
	r.pendingUpdates = r.pendingUpdates[:0]
	r.statusCacheMutex.Unlock()
	r.Infof("syncing %d pending status updates to master node", pendingCount)
	for _, update := range updates {
		if err := r.syncStatusUpdate(update); err != nil {
			r.Errorf("failed to sync status update for task %s: %v", update.TaskId.Hex(), err)
			r.statusCacheMutex.Lock()
			r.pendingUpdates = append(r.pendingUpdates, update)
			r.statusCacheMutex.Unlock()
			return err
		}
	}
	r.Infof("successfully synced %d status updates", pendingCount)
	return nil
}

func (r *Runner) syncStatusUpdate(snapshot TaskStatusSnapshot) error {
	task, err := r.svc.GetTaskById(snapshot.TaskId)
	if err != nil {
		return fmt.Errorf("failed to get task %s: %w", snapshot.TaskId.Hex(), err)
	}
	if task.UpdatedAt.After(snapshot.Timestamp) {
		r.Debugf("skipping status sync for task %s - database is newer", snapshot.TaskId.Hex())
		return nil
	}
	task.Status = snapshot.Status
	task.Error = snapshot.Error
	task.Pid = snapshot.Pid
	if utils.IsMaster() {
		err = service.NewModelService[models.Task]().ReplaceById(task.Id, *task)
	} else {
		err = client.NewModelService[models.Task]().ReplaceById(task.Id, *task)
	}
	if err != nil {
		return fmt.Errorf("failed to update task in database: %w", err)
	}
	r.Debugf("synced status update for task %s: %s", snapshot.TaskId.Hex(), snapshot.Status)
	return nil
}

func (r *Runner) getCachedTaskStatus() *TaskStatusSnapshot {
	r.statusCache.mu.RLock()
	defer r.statusCache.mu.RUnlock()
	if snapshot, exists := r.statusCache.snapshots[r.tid]; exists {
		return snapshot
	}
	return nil
}

func (r *Runner) cleanupStatusCache() {
	if r.statusCache != nil && r.statusCache.filePath != "" {
		if err := os.Remove(r.statusCache.filePath); err != nil && !os.IsNotExist(err) {
			r.Warnf("failed to remove status cache file: %v", err)
		}
	}
}

// GetCachedTaskStatus retrieves the cached status for this task (public method for external access)
func (r *Runner) GetCachedTaskStatus() *TaskStatusSnapshot {
	return r.getCachedTaskStatus()
}

// SyncPendingStatusUpdates syncs all pending status updates to the master node (public method for external access)
func (r *Runner) SyncPendingStatusUpdates() error {
	return r.syncPendingStatusUpdates()
}
