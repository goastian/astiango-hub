package handler

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"time"

	"github.com/goastian/astiango-hub/core/constants"
	grpcclient "github.com/goastian/astiango-hub/core/grpc/client"
	"github.com/goastian/astiango-hub/core/interfaces"
	"github.com/goastian/astiango-hub/core/models/client"
	"github.com/goastian/astiango-hub/core/models/models"
	"github.com/goastian/astiango-hub/core/models/service"
	nodeconfig "github.com/goastian/astiango-hub/core/node/config"
	"github.com/goastian/astiango-hub/core/utils"
	"github.com/goastian/astiango-hub/grpc"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Service struct {
	// dependencies
	cfgSvc interfaces.NodeConfigService
	c      *grpcclient.GrpcClient // grpc client

	// settings
	reportInterval time.Duration
	fetchInterval  time.Duration
	fetchTimeout   time.Duration
	cancelTimeout  time.Duration

	// internals variables
	ctx     context.Context
	cancel  context.CancelFunc
	stopped bool
	mu      sync.RWMutex
	runners sync.Map       // pool of task runners started
	wg      sync.WaitGroup // track background goroutines

	// tickers for cleanup
	fetchTicker  *time.Ticker
	reportTicker *time.Ticker

	// worker pool for bounded task execution
	workerPool *TaskWorkerPool
	maxWorkers int

	// stream manager for leak-free stream handling
	streamManager *StreamManager

	interfaces.Logger
}

func (svc *Service) Start() {
	// wait for grpc client ready
	grpcclient.GetGrpcClient().WaitForReady()

	// Initialize tickers
	svc.fetchTicker = time.NewTicker(svc.fetchInterval)
	svc.reportTicker = time.NewTicker(svc.reportInterval)

	// Get max workers from current node configuration
	svc.maxWorkers = svc.getCurrentNodeMaxRunners()

	// Initialize and start worker pool with dynamic max workers
	svc.workerPool = NewTaskWorkerPool(svc.maxWorkers, svc)
	svc.workerPool.Start()

	// Initialize and start stream manager
	svc.streamManager.Start()

	// Start goroutine monitoring (adds to WaitGroup internally)
	svc.startGoroutineMonitoring()

	// Start background goroutines with WaitGroup tracking
	svc.wg.Add(2)
	go svc.reportStatus()
	go svc.fetchAndRunTasks()

	queueSize := cap(svc.workerPool.taskQueue)
	if svc.maxWorkers == -1 {
		svc.Infof("Task handler service started with unlimited workers (from node config) and queue size %d", queueSize)
	} else {
		svc.Infof("Task handler service started with %d max workers (from node config) and queue size %d", svc.maxWorkers, queueSize)
	}

	// Start the stuck task cleanup routine (adds to WaitGroup internally)
	svc.startStuckTaskCleanup()
}

func (svc *Service) Stop() {
	svc.mu.Lock()
	if svc.stopped {
		svc.mu.Unlock()
		return
	}
	svc.stopped = true
	svc.mu.Unlock()

	svc.Infof("Stopping task handler service...")

	// Cancel context to signal all goroutines to stop
	if svc.cancel != nil {
		svc.cancel()
	}

	// Stop worker pool first
	if svc.workerPool != nil {
		svc.workerPool.Stop()
	}

	// Stop stream manager
	if svc.streamManager != nil {
		svc.streamManager.Stop()
	}

	// Stop tickers to prevent new tasks
	if svc.fetchTicker != nil {
		svc.fetchTicker.Stop()
	}
	if svc.reportTicker != nil {
		svc.reportTicker.Stop()
	}

	// Cancel all running tasks gracefully
	svc.stopAllRunners()

	// Wait for all background goroutines to finish
	done := make(chan struct{})
	go func() {
		svc.wg.Wait()
		close(done)
	}()

	// Give goroutines time to finish gracefully, then force stop
	select {
	case <-done:
		svc.Infof("All goroutines stopped gracefully")
	case <-time.After(30 * time.Second):
		svc.Warnf("Some goroutines did not stop gracefully within timeout")
	}

	svc.Infof("Task handler service stopped")
}

func (svc *Service) startGoroutineMonitoring() {
	svc.wg.Add(1) // Track goroutine monitoring in WaitGroup
	go func() {
		defer svc.wg.Done()
		defer func() {
			if r := recover(); r != nil {
				svc.Errorf("[TaskHandler] goroutine monitoring panic: %v", r)
			}
		}()

		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()

		initialCount := runtime.NumGoroutine()
		svc.Infof("[TaskHandler] initial goroutine count: %d", initialCount)

		for {
			select {
			case <-svc.ctx.Done():
				svc.Infof("[TaskHandler] goroutine monitoring shutting down")
				return
			case <-ticker.C:
				currentCount := runtime.NumGoroutine()
				if currentCount > initialCount+50 { // Alert if 50+ more goroutines than initial
					svc.Warnf("[TaskHandler] potential goroutine leak detected - current: %d, initial: %d, diff: %d",
						currentCount, initialCount, currentCount-initialCount)
				} else {
					svc.Debugf("[TaskHandler] goroutine count: %d (initial: %d)", currentCount, initialCount)
				}
			}
		}
	}()
}

func (svc *Service) fetchAndRunTasks() {
	defer svc.wg.Done()
	defer func() {
		if r := recover(); r != nil {
			svc.Errorf("fetchAndRunTasks panic recovered: %v", r)
		}
	}()

	for {
		select {
		case <-svc.ctx.Done():
			svc.Infof("fetchAndRunTasks stopped by context")
			return
		case <-svc.fetchTicker.C:
			// Use a separate context with timeout for each operation
			if err := svc.processFetchCycle(); err != nil {
				//svc.Debugf("fetch cycle error: %v", err)
			}
		}
	}
}

func (svc *Service) processFetchCycle() error {
	// Check if stopped
	svc.mu.RLock()
	stopped := svc.stopped
	svc.mu.RUnlock()

	if stopped {
		return fmt.Errorf("service stopped")
	}

	// current node
	n, err := svc.GetCurrentNode()
	if err != nil {
		return fmt.Errorf("failed to get current node: %w", err)
	}

	// Log node status for debugging reconnection issues
	svc.Debugf("[TIMING] Node status check: Active=%v, Enabled=%v, Status=%s",
		n.Active, n.Enabled, n.Status)

	// skip if node is not active or enabled
	if !n.Active || !n.Enabled {
		return fmt.Errorf("node not active or enabled (active=%v, enabled=%v)", n.Active, n.Enabled)
	}

	// validate if max runners is reached (max runners = 0 means no limit)
	if n.MaxRunners > 0 && svc.getRunnerCount() >= n.MaxRunners {
		return fmt.Errorf("max runners reached")
	}

	// fetch task id
	tid, err := svc.fetchTask()
	if err != nil {
		return fmt.Errorf("failed to fetch task: %w", err)
	}

	// skip if no task id
	if tid.IsZero() {
		return fmt.Errorf("no task available")
	}

	// run task - now using worker pool instead of unlimited goroutines
	if err := svc.runTask(tid); err != nil {
		// Handle task error
		t, getErr := svc.GetTaskById(tid)
		if getErr == nil && t.Status != constants.TaskStatusCancelled {
			t.Error = err.Error()
			t.Status = constants.TaskStatusError
			t.SetUpdated(t.CreatedBy)
			_ = client.NewModelService[models.Task]().ReplaceById(t.Id, *t)
		}
		return fmt.Errorf("failed to run task: %w", err)
	}

	return nil
}

func (svc *Service) reportStatus() {
	defer svc.wg.Done()
	defer func() {
		if r := recover(); r != nil {
			svc.Errorf("reportStatus panic recovered: %v", r)
		}
	}()

	for {
		select {
		case <-svc.ctx.Done():
			svc.Infof("reportStatus stopped by context")
			return
		case <-svc.reportTicker.C:
			// Update node status with error handling
			if err := svc.updateNodeStatus(); err != nil {
				svc.Errorf("failed to report status: %v", err)
			}
		}
	}
}

func (svc *Service) GetCancelTimeout() (timeout time.Duration) {
	return svc.cancelTimeout
}

func (svc *Service) GetNodeConfigService() (cfgSvc interfaces.NodeConfigService) {
	return svc.cfgSvc
}

// getGrpcClient returns the current gRPC client, ensuring we always have the latest instance
// This is important after ResetGrpcClient() calls which create new client instances
func (svc *Service) getGrpcClient() *grpcclient.GrpcClient {
	return grpcclient.GetGrpcClient()
}

func (svc *Service) getCurrentNodeMaxRunners() int {
	n, err := svc.GetCurrentNode()
	if err != nil {
		svc.Errorf("failed to get current node for max runners: %v", err)
		// Fallback to config default
		return utils.GetNodeMaxRunners()
	}

	// If MaxRunners is 0, it means unlimited workers
	if n.MaxRunners == 0 {
		return -1 // Use -1 internally to represent unlimited
	}

	// If MaxRunners is negative (not set), use config default
	if n.MaxRunners < 0 {
		return utils.GetNodeMaxRunners()
	}

	return n.MaxRunners
}

func (svc *Service) GetCurrentNode() (n *models.Node, err error) {
	// node key
	nodeKey := svc.cfgSvc.GetNodeKey()

	// current node
	if svc.cfgSvc.IsMaster() {
		n, err = service.NewModelService[models.Node]().GetOne(bson.M{"key": nodeKey}, nil)
	} else {
		n, err = client.NewModelService[models.Node]().GetOne(bson.M{"key": nodeKey}, nil)
	}
	if err != nil {
		return nil, err
	}

	return n, nil
}

func (svc *Service) GetTaskById(id primitive.ObjectID) (t *models.Task, err error) {
	if svc.cfgSvc.IsMaster() {
		t, err = service.NewModelService[models.Task]().GetById(id)
	} else {
		t, err = client.NewModelService[models.Task]().GetById(id)
	}
	if err != nil {
		svc.Errorf("failed to get task by id: %v", err)
		return nil, err
	}

	return t, nil
}

func (svc *Service) UpdateTask(t *models.Task) (err error) {
	t.SetUpdated(t.CreatedBy)
	if svc.cfgSvc.IsMaster() {
		err = service.NewModelService[models.Task]().ReplaceById(t.Id, *t)
	} else {
		err = client.NewModelService[models.Task]().ReplaceById(t.Id, *t)
	}
	if err != nil {
		return err
	}
	return nil
}

func (svc *Service) GetSpiderById(id primitive.ObjectID) (s *models.Spider, err error) {
	if svc.cfgSvc.IsMaster() {
		s, err = service.NewModelService[models.Spider]().GetById(id)
	} else {
		s, err = client.NewModelService[models.Spider]().GetById(id)
	}
	if err != nil {
		svc.Errorf("failed to get spider by id: %v", err)
		return nil, err
	}

	return s, nil
}

func (svc *Service) getRunnerCount() (count int) {
	n, err := svc.GetCurrentNode()
	if err != nil {
		svc.Errorf("failed to get current node: %v", err)
		return
	}
	query := bson.M{
		"node_id": n.Id,
		"status": bson.M{
			"$in": []string{constants.TaskStatusAssigned, constants.TaskStatusRunning},
		},
	}
	if svc.cfgSvc.IsMaster() {
		count, err = service.NewModelService[models.Task]().Count(query)
		if err != nil {
			svc.Errorf("failed to count tasks: %v", err)
			return
		}
	} else {
		count, err = client.NewModelService[models.Task]().Count(query)
		if err != nil {
			svc.Errorf("failed to count tasks: %v", err)
			return
		}
	}
	return count
}

func (svc *Service) updateNodeStatus() (err error) {
	// current node
	n, err := svc.GetCurrentNode()
	if err != nil {
		return err
	}

	// Check if max runners configuration has changed and update worker pool
	currentMaxWorkers := n.MaxRunners
	// Handle unlimited workers (0 means unlimited)
	if currentMaxWorkers == 0 {
		currentMaxWorkers = -1 // Use -1 internally to represent unlimited
	} else if currentMaxWorkers < 0 {
		currentMaxWorkers = utils.GetNodeMaxRunners() // Use config default if not set (negative)
	}

	if currentMaxWorkers != svc.maxWorkers {
		if currentMaxWorkers == -1 {
			svc.Infof("Node max runners changed from %d to unlimited, updating worker pool", svc.maxWorkers)
		} else if svc.maxWorkers == -1 {
			svc.Infof("Node max runners changed from unlimited to %d, updating worker pool", currentMaxWorkers)
		} else {
			svc.Infof("Node max runners changed from %d to %d, updating worker pool", svc.maxWorkers, currentMaxWorkers)
		}
		svc.maxWorkers = currentMaxWorkers
		if svc.workerPool != nil {
			svc.workerPool.UpdateMaxWorkers(currentMaxWorkers)
		}
	}

	// set available runners
	n.CurrentRunners = svc.getRunnerCount()

	// Log goroutine count for leak monitoring
	currentGoroutines := runtime.NumGoroutine()
	svc.Debugf("Node status update - runners: %d, goroutines: %d", n.CurrentRunners, currentGoroutines)

	// save node with retry for reconnection scenarios
	n.SetUpdated(n.CreatedBy)
	if svc.cfgSvc.IsMaster() {
		err = service.NewModelService[models.Node]().ReplaceById(n.Id, *n)
		if err != nil {
			return err
		}
	} else {
		// Worker node: use gRPC with retry logic
		ctx, cancel := context.WithTimeout(svc.ctx, 30*time.Second)
		defer cancel()

		err = grpcclient.RetryWithBackoff(ctx, func() error {
			return client.NewModelService[models.Node]().ReplaceById(n.Id, *n)
		}, 3, svc.Logger, "node status update")

		if err != nil {
			return err
		}
	}

	return nil
}

func (svc *Service) fetchTask() (tid primitive.ObjectID, err error) {
	// Use service context with timeout for fetch operation
	ctx, cancel := context.WithTimeout(svc.ctx, svc.fetchTimeout)
	defer cancel()
	taskClient, err := svc.getGrpcClient().GetTaskClient()
	if err != nil {
		return primitive.NilObjectID, fmt.Errorf("failed to get task client: %v", err)
	}
	res, err := taskClient.FetchTask(ctx, &grpc.TaskServiceFetchTaskRequest{
		NodeKey: svc.cfgSvc.GetNodeKey(),
	})
	if err != nil {
		return primitive.NilObjectID, fmt.Errorf("fetchTask task error: %v", err)
	}
	// validate task id
	tid, err = primitive.ObjectIDFromHex(res.GetTaskId())
	if err != nil {
		return primitive.NilObjectID, fmt.Errorf("invalid task id: %s", res.GetTaskId())
	}
	return tid, nil
}

func (svc *Service) startStuckTaskCleanup() {
	svc.wg.Add(1) // Track this goroutine in the WaitGroup
	go func() {
		defer svc.wg.Done() // Ensure WaitGroup is decremented
		defer func() {
			if r := recover(); r != nil {
				svc.Errorf("startStuckTaskCleanup panic recovered: %v", r)
			}
		}()

		ticker := time.NewTicker(5 * time.Minute) // Check every 5 minutes
		defer ticker.Stop()

		for {
			select {
			case <-svc.ctx.Done():
				svc.Debugf("stuck task cleanup routine shutting down")
				return
			case <-ticker.C:
				svc.checkAndCleanupStuckTasks()
			}
		}
	}()
}

// checkAndCleanupStuckTasks checks for tasks that have been trying to cancel for too long
func (svc *Service) checkAndCleanupStuckTasks() {
	defer func() {
		if r := recover(); r != nil {
			svc.Errorf("panic in stuck task cleanup: %v", r)
		}
	}()

	var stuckTasks []primitive.ObjectID

	// Check all running tasks
	svc.runners.Range(func(key, value interface{}) bool {
		taskId, ok := key.(primitive.ObjectID)
		if !ok {
			return true
		}

		// Get task from database to check its state
		t, err := svc.GetTaskById(taskId)
		if err != nil {
			svc.Errorf("failed to get task[%s] during stuck cleanup: %v", taskId.Hex(), err)
			return true
		}

		// Check if task has been in cancelling state too long (15+ minutes)
		if t.Status == constants.TaskStatusCancelled && time.Since(t.UpdatedAt) > 15*time.Minute {
			svc.Warnf("detected stuck cancelled task[%s], will force cleanup", taskId.Hex())
			stuckTasks = append(stuckTasks, taskId)
		}

		return true
	})

	// Force cleanup stuck tasks
	for _, taskId := range stuckTasks {
		svc.Infof("force cleaning up stuck task[%s]", taskId.Hex())

		// Remove from runners map
		svc.runners.Delete(taskId)

		// Update task status to indicate it was force cleaned
		t, err := svc.GetTaskById(taskId)
		if err == nil {
			t.Status = constants.TaskStatusCancelled
			t.Error = "Task was stuck in cancelling state and was force cleaned up"
			if updateErr := svc.UpdateTask(t); updateErr != nil {
				svc.Errorf("failed to update stuck task[%s] status: %v", taskId.Hex(), updateErr)
			}
		}
	}

	if len(stuckTasks) > 0 {
		svc.Infof("cleaned up %d stuck tasks", len(stuckTasks))
	}
}

func newTaskHandlerService() *Service {
	// service
	svc := &Service{
		fetchInterval:  1 * time.Second,
		fetchTimeout:   15 * time.Second,
		reportInterval: 5 * time.Second,
		cancelTimeout:  60 * time.Second,
		maxWorkers:     utils.GetNodeMaxRunners(),
		mu:             sync.RWMutex{},
		runners:        sync.Map{},
		Logger:         utils.NewLogger("TaskHandlerService"),
	}

	// Initialize context for graceful shutdown
	svc.ctx, svc.cancel = context.WithCancel(context.Background())

	// dependency injection
	svc.cfgSvc = nodeconfig.GetNodeConfigService()

	// grpc client
	svc.c = grpcclient.GetGrpcClient()

	// initialize stream manager
	svc.streamManager = NewStreamManager(svc)

	return svc
}

var _service *Service
var _serviceOnce sync.Once

func GetTaskHandlerService() *Service {
	_serviceOnce.Do(func() {
		_service = newTaskHandlerService()
	})
	return _service
}
