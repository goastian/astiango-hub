package handler

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/goastian/astiango-hub/core/utils"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// taskRequest represents a task execution request
type taskRequest struct {
	taskId primitive.ObjectID
}

// TaskWorkerPool manages a dynamic pool of workers for task execution
type TaskWorkerPool struct {
	maxWorkers int
	ctx        context.Context
	cancel     context.CancelFunc
	wg         sync.WaitGroup
	taskQueue  chan taskRequest
	service    *Service
	
	// Track active workers
	activeWorkers int
	workerMutex   sync.RWMutex
}

func NewTaskWorkerPool(maxWorkers int, service *Service) *TaskWorkerPool {
	// Use service context for proper cancellation chain
	ctx, cancel := context.WithCancel(service.ctx)
	
	var queueSize int
	if maxWorkers == -1 {
		// Unlimited workers - use configured queue size or larger default
		configuredSize := utils.GetTaskQueueSize()
		if configuredSize > 0 {
			queueSize = configuredSize
		} else {
			queueSize = 1000 // Large default for unlimited workers
		}
	} else {
		// Limited workers - use configured size or calculate based on max workers
		configuredSize := utils.GetTaskQueueSize()
		if configuredSize > 0 {
			queueSize = configuredSize
		} else {
			// Use a more generous queue size to handle task bursts
			// Queue size is maxWorkers * 5 to allow for better buffering
			queueSize = maxWorkers * 5
			if queueSize < 50 {
				queueSize = 50 // Minimum queue size
			}
		}
	}

	return &TaskWorkerPool{
		maxWorkers:    maxWorkers,
		ctx:           ctx,
		cancel:        cancel,
		taskQueue:     make(chan taskRequest, queueSize),
		service:       service,
		activeWorkers: 0,
		workerMutex:   sync.RWMutex{},
	}
}

func (pool *TaskWorkerPool) Start() {
	// Don't pre-create workers - they will be created on-demand
	if pool.maxWorkers == -1 {
		pool.service.Debugf("Task worker pool started with unlimited workers")
	} else {
		pool.service.Debugf("Task worker pool started with max workers: %d", pool.maxWorkers)
	}
}

func (pool *TaskWorkerPool) Stop() {
	pool.cancel()
	close(pool.taskQueue)
	pool.wg.Wait()
}

func (pool *TaskWorkerPool) SubmitTask(taskId primitive.ObjectID) error {
	req := taskRequest{
		taskId: taskId,
	}

	select {
	case pool.taskQueue <- req:
		pool.service.Debugf("task[%s] queued for parallel execution, queue usage: %d/%d",
			taskId.Hex(), len(pool.taskQueue), cap(pool.taskQueue))
		
		// Try to create a new worker if we haven't reached the limit
		pool.maybeCreateWorker()
		
		return nil // Return immediately - task will execute in parallel
	case <-pool.ctx.Done():
		return fmt.Errorf("worker pool is shutting down")
	default:
		queueLen := len(pool.taskQueue)
		queueCap := cap(pool.taskQueue)
		if pool.maxWorkers == -1 {
			pool.service.Warnf("task queue is full (%d/%d), consider increasing system resources",
				queueLen, queueCap)
			return fmt.Errorf("task queue is full (%d/%d), consider increasing system resources",
				queueLen, queueCap)
		} else {
			pool.service.Warnf("task queue is full (%d/%d), consider increasing node max_runners configuration",
				queueLen, queueCap)
			return fmt.Errorf("task queue is full (%d/%d), consider increasing node max_runners configuration",
				queueLen, queueCap)
		}
	}
}

func (pool *TaskWorkerPool) UpdateMaxWorkers(newMaxWorkers int) {
	pool.workerMutex.Lock()
	defer pool.workerMutex.Unlock()
	
	oldMax := pool.maxWorkers
	pool.maxWorkers = newMaxWorkers
	
	// Update queue size if needed
	var needQueueResize bool
	var newQueueSize int
	
	configuredSize := utils.GetTaskQueueSize()
	
	if newMaxWorkers == -1 {
		// Unlimited workers
		if configuredSize > 0 {
			newQueueSize = configuredSize
		} else {
			newQueueSize = 1000
		}
		needQueueResize = newQueueSize > cap(pool.taskQueue)
	} else if oldMax == -1 {
		// From unlimited to limited
		if configuredSize > 0 {
			newQueueSize = configuredSize
		} else {
			newQueueSize = newMaxWorkers * 5
			if newQueueSize < 50 {
				newQueueSize = 50
			}
		}
		needQueueResize = true // Always resize when going from unlimited to limited
	} else if newMaxWorkers > oldMax {
		// Increase queue capacity
		if configuredSize > 0 {
			newQueueSize = configuredSize
		} else {
			newQueueSize = newMaxWorkers * 5
			if newQueueSize < 50 {
				newQueueSize = 50
			}
		}
		needQueueResize = newQueueSize > cap(pool.taskQueue)
	}
	
	if needQueueResize {
		oldQueue := pool.taskQueue
		pool.taskQueue = make(chan taskRequest, newQueueSize)
		
		// Copy existing tasks to new queue
		close(oldQueue)
		for req := range oldQueue {
			select {
			case pool.taskQueue <- req:
			default:
				// If new queue is somehow full, log the issue but don't block
				pool.service.Warnf("Lost task during queue resize: %s", req.taskId.Hex())
			}
		}
	}
	
	if oldMax == -1 && newMaxWorkers != -1 {
		pool.service.Infof("Updated worker pool from unlimited to max workers: %d", newMaxWorkers)
	} else if oldMax != -1 && newMaxWorkers == -1 {
		pool.service.Infof("Updated worker pool from max workers %d to unlimited", oldMax)
	} else {
		pool.service.Infof("Updated worker pool max workers from %d to %d", oldMax, newMaxWorkers)
	}
}

func (pool *TaskWorkerPool) maybeCreateWorker() {
	pool.workerMutex.Lock()
	defer pool.workerMutex.Unlock()
	
	// Only create a worker if we have tasks queued and haven't reached the limit
	// For unlimited workers (maxWorkers == -1), always create if there are queued tasks
	hasQueuedTasks := len(pool.taskQueue) > 0
	underLimit := pool.maxWorkers == -1 || pool.activeWorkers < pool.maxWorkers
	
	if hasQueuedTasks && underLimit {
		pool.activeWorkers++
		workerID := pool.activeWorkers
		pool.wg.Add(1)
		go pool.worker(workerID)
		
		if pool.maxWorkers == -1 {
			pool.service.Debugf("created on-demand worker %d (unlimited workers mode, total active: %d)", 
				workerID, pool.activeWorkers)
		} else {
			pool.service.Debugf("created on-demand worker %d (total active: %d/%d)", 
				workerID, pool.activeWorkers, pool.maxWorkers)
		}
	}
}

func (pool *TaskWorkerPool) worker(workerID int) {
	defer pool.wg.Done()
	defer func() {
		// Decrement active worker count when worker exits
		pool.workerMutex.Lock()
		pool.activeWorkers--
		pool.workerMutex.Unlock()
		
		if r := recover(); r != nil {
			pool.service.Errorf("worker %d panic recovered: %v", workerID, r)
		}
	}()

	pool.service.Debugf("worker %d started", workerID)
	idleTimeout := 5 * time.Minute // Worker will exit after 5 minutes of idleness
	timer := time.NewTimer(idleTimeout)
	defer timer.Stop()

	for {
		timer.Reset(idleTimeout)
		
		select {
		case <-pool.ctx.Done():
			pool.service.Debugf("worker %d shutting down", workerID)
			return
		case req, ok := <-pool.taskQueue:
			if !ok {
				pool.service.Debugf("worker %d: task queue closed", workerID)
				return
			}

			// Execute task asynchronously - each worker handles one task at a time
			// but multiple workers can process different tasks simultaneously
			pool.service.Debugf("worker %d processing task[%s]", workerID, req.taskId.Hex())
			err := pool.service.executeTask(req.taskId)
			if err != nil {
				pool.service.Errorf("worker %d failed to execute task[%s]: %v",
					workerID, req.taskId.Hex(), err)
			} else {
				pool.service.Debugf("worker %d completed task[%s]", workerID, req.taskId.Hex())
			}
		case <-timer.C:
			// Worker has been idle for too long, exit to save resources
			pool.service.Debugf("worker %d exiting due to inactivity", workerID)
			return
		}
	}
}
