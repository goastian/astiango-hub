package handler

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"github.com/goastian/astiango-hub/core/dependency"
	"github.com/goastian/astiango-hub/core/fs"
	"github.com/goastian/astiango-hub/core/models/models"
	"github.com/hashicorp/go-multierror"

	"github.com/goastian/astiango-hub/core/constants"
	"github.com/goastian/astiango-hub/core/entity"
	client2 "github.com/goastian/astiango-hub/core/grpc/client"
	"github.com/goastian/astiango-hub/core/interfaces"
	"github.com/goastian/astiango-hub/core/models/client"
	"github.com/goastian/astiango-hub/core/models/service"
	"github.com/goastian/astiango-hub/core/utils"
	"github.com/goastian/astiango-hub/grpc"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// newTaskRunner creates a new task runner instance with the specified task ID
// It initializes all necessary components and establishes required connections
func newTaskRunner(id primitive.ObjectID, svc *Service) (r *Runner, err error) {
	// validate options
	if id.IsZero() {
		err = fmt.Errorf("invalid task id: %s", id.Hex())
		return nil, err
	}

	// runner
	r = &Runner{
		subscribeTimeout: 30 * time.Second,
		bufferSize:       1024 * 1024,
		svc:              svc,
		tid:              id,
		ch:               make(chan constants.TaskSignal),
		logBatchSize:     20,
		Logger:           utils.NewLogger("TaskRunner"),
		// treat all tasks as potentially long-running
		maxConnRetries:      10,
		connRetryDelay:      10 * time.Second,
		ipcTimeout:          60 * time.Second, // generous timeout for all tasks
		healthCheckInterval: 5 * time.Second,  // check process every 5 seconds
		connHealthInterval:  60 * time.Second, // check connection health every minute
		// initialize circuit breaker for log connections
		logConnHealthy:         true,
		logCircuitOpenDuration: 30 * time.Second, // keep circuit open for 30 seconds after failures
	}

	// multi error
	var errs multierror.Error

	// task
	r.t, err = svc.GetTaskById(id)
	if err != nil {
		errs.Errors = append(errs.Errors, err)
	} else {
		// spider
		r.s, err = svc.GetSpiderById(r.t.SpiderId)
		if err != nil {
			errs.Errors = append(errs.Errors, err)
		} else {
			// task fs service
			r.fsSvc = fs.NewFsService(filepath.Join(utils.GetWorkspace(), r.s.Id.Hex()))
		}
	}

	// Initialize context and done channel - use service context for proper cancellation chain
	r.ctx, r.cancel = context.WithCancel(svc.ctx)
	r.done = make(chan struct{})

	// Initialize status cache for disconnection resilience
	if err := r.initStatusCache(); err != nil {
		r.Errorf("error initializing status cache: %v", err)
		errs.Errors = append(errs.Errors, err)
	}

	// initialize task runner
	if err := r.Init(); err != nil {
		r.Errorf("error initializing task runner: %v", err)
		errs.Errors = append(errs.Errors, err)
	}

	return r, errs.ErrorOrNil()
}

// Runner represents a task execution handler that manages the lifecycle of a running task
type Runner struct {
	// dependencies
	svc   *Service    // task handler service
	fsSvc *fs.Service // task fs service

	// settings
	subscribeTimeout time.Duration // maximum time to wait for task subscription
	bufferSize       int           // buffer size for reading process output

	// internals
	cmd  *exec.Cmd                      // process command instance
	pid  int                            // process id
	tid  primitive.ObjectID             // task id
	t    *models.Task                   // task model instance
	s    *models.Spider                 // spider model instance
	ch   chan constants.TaskSignal      // channel for task status communication
	err  error                          // captures any process execution errors
	cwd  string                         // current working directory for task
	conn grpc.TaskService_ConnectClient // gRPC stream connection for task service
	interfaces.Logger

	// log handling
	readerStdout *bufio.Reader // reader for process stdout
	readerStderr *bufio.Reader // reader for process stderr
	logBatchSize int           // number of log lines to batch before sending

	// IPC (Inter-Process Communication)
	stdinPipe  io.WriteCloser          // pipe for writing to child process
	stdoutPipe io.ReadCloser           // pipe for reading from child process
	ipcChan    chan entity.IPCMessage  // channel for sending IPC messages
	ipcHandler func(entity.IPCMessage) // callback for handling received IPC messages

	// goroutine management
	ctx    context.Context    // context for controlling goroutine lifecycle
	cancel context.CancelFunc // function to cancel the context
	done   chan struct{}      // channel to signal completion
	wg     sync.WaitGroup     // wait group for goroutine synchronization

	// connection management for robust task execution
	connMutex         sync.RWMutex  // mutex for connection access
	connHealthTicker  *time.Ticker  // ticker for connection health checks
	lastConnCheck     time.Time     // last successful connection check
	connRetryAttempts int           // current retry attempts
	maxConnRetries    int           // maximum connection retry attempts
	connRetryDelay    time.Duration // delay between connection retries
	resourceCleanup   *time.Ticker  // periodic resource cleanup

	// circuit breaker for log connections to prevent cascading failures
	logConnHealthy         bool          // tracks if log connection is healthy
	logConnMutex           sync.RWMutex  // mutex for log connection health state
	lastLogSendFailure     time.Time     // last time log send failed
	logCircuitOpenTime     time.Time     // when circuit breaker was opened
	logFailureCount        int           // consecutive log send failures
	logCircuitOpenDuration time.Duration // how long to keep circuit open after failures

	// configurable timeouts for robust task execution
	ipcTimeout          time.Duration // timeout for IPC operations
	healthCheckInterval time.Duration // interval for health checks
	connHealthInterval  time.Duration // interval for connection health checks

	// status cache for disconnection resilience
	statusCache      *TaskStatusCache     // local status cache that survives disconnections
	pendingUpdates   []TaskStatusSnapshot // status updates to sync when reconnected
	statusCacheMutex sync.RWMutex         // mutex for status cache operations
	taskEnv          []string
	secretValues     []string
	sandboxName      string
}

// Init initializes the task runner by updating the task status and establishing gRPC connections
func (r *Runner) Init() (err error) {
	// wait for grpc client ready
	client2.GetGrpcClient().WaitForReady()

	// update task
	if err := r.updateTask("", nil); err != nil {
		return err
	}

	// grpc task service stream client
	if err := r.initConnection(); err != nil {
		return err
	}

	return nil
}

// Run executes the task and manages its lifecycle, including file synchronization, process execution,
// and status monitoring. Returns an error if the task execution fails.
func (r *Runner) Run() (err error) {
	// log task started
	r.Infof("task[%s] started", r.tid.Hex())

	// update task status (processing)
	if err := r.updateTask(constants.TaskStatusRunning, nil); err != nil {
		return err
	}

	// configure working directory
	r.configureCwd()

	// sync files worker nodes
	if !utils.IsMaster() {
		if err := r.syncFiles(); err != nil {
			r.Warnf("error synchronizing files: %v", err)
		}
	}

	// configure environment variables
	r.configureEnv()

	// configure cmd
	err = r.configureCmd()
	if err != nil {
		return r.updateTask(constants.TaskStatusError, err)
	}

	// start process
	if err := r.cmd.Start(); err != nil {
		return r.updateTask(constants.TaskStatusError, err)
	}

	// process id
	if r.cmd.Process == nil {
		return r.updateTask(constants.TaskStatusError, constants.ErrNotExists)
	}
	r.pid = r.cmd.Process.Pid
	r.t.Pid = r.pid

	// start health check
	go r.startHealthCheck()

	// Start IPC reader
	go r.startIPCReader()

	// Start IPC handler
	go r.handleIPC()

	// ZOMBIE PREVENTION: Start zombie process monitor
	go r.startZombieMonitor()

	// Ensure cleanup when Run() exits
	defer func() {
		utils.RemoveSandbox(r.sandboxName)
		// 1. Signal all goroutines to stop
		r.cancel()

		// 2. Stop tickers to prevent resource leaks
		if r.connHealthTicker != nil {
			r.connHealthTicker.Stop()
		}
		if r.resourceCleanup != nil {
			r.resourceCleanup.Stop()
		}

		// 3. Wait for all goroutines to finish with timeout
		done := make(chan struct{})
		go func() {
			r.wg.Wait()
			close(done)
		}()

		select {
		case <-done:
			// All goroutines finished normally
		case <-time.After(10 * time.Second): // Increased timeout for long-running tasks
			// Timeout waiting for goroutines, proceed with cleanup
			r.Warnf("timeout waiting for goroutines to finish, proceeding with cleanup")
		}

		// 4. Close gRPC connection after all goroutines have stopped
		r.connMutex.Lock()
		if r.conn != nil {
			_ = r.conn.CloseSend()
			r.conn = nil
		}
		r.connMutex.Unlock()

		// 5. Close channels after everything has stopped
		close(r.done)
		if r.ipcChan != nil {
			close(r.ipcChan)
		}

		// 6. Clean up status cache for completed tasks
		r.cleanupStatusCache()
	}()

	// wait for process to finish
	return r.wait()
}

// Cancel terminates the running task. If force is true, the process will be killed immediately
// without waiting for graceful shutdown.
func (r *Runner) Cancel(force bool) (err error) {
	r.Debugf("attempting to cancel task (force: %v)", force)

	// Signal goroutines to stop
	r.cancel()

	// Stop health check ticker immediately to prevent interference
	if r.connHealthTicker != nil {
		r.connHealthTicker.Stop()
		r.Debugf("stopped connection health ticker")
	}

	// Close gRPC connection to stop health check messages
	r.connMutex.Lock()
	if r.conn != nil {
		_ = r.conn.CloseSend()
		r.conn = nil
		r.Debugf("closed gRPC connection to stop health checks")
	}
	r.connMutex.Unlock()

	// Wait a moment for background goroutines to respond to cancellation signal
	time.Sleep(100 * time.Millisecond)
	utils.StopSandbox(r.sandboxName, force)

	// If force is not requested, try graceful termination first
	if !force {
		r.Debugf("attempting graceful termination of process[%d]", r.pid)
		if err = utils.KillProcess(r.cmd, false); err != nil {
			r.Warnf("graceful termination failed: %v, escalating to force", err)
			force = true
		} else {
			// Wait for graceful termination with shorter timeout
			ctx, cancel := context.WithTimeout(r.ctx, 15*time.Second)
			defer cancel()

			ticker := time.NewTicker(500 * time.Millisecond)
			defer ticker.Stop()

			for {
				select {
				case <-ctx.Done():
					r.Warnf("graceful termination timeout, escalating to force")
					force = true
					goto forceKill
				case <-ticker.C:
					if !utils.ProcessIdExists(r.pid) {
						r.Debugf("process[%d] terminated gracefully", r.pid)
						return nil
					}
				}
			}
		}
	}

forceKill:
	if force {
		r.Debugf("force killing process[%d]", r.pid)
		if err = utils.KillProcess(r.cmd, true); err != nil {
			r.Errorf("force kill failed: %v", err)
			return err
		}
	}

	// Wait for process to be killed with timeout
	ctx, cancel := context.WithTimeout(r.ctx, r.svc.GetCancelTimeout())
	defer cancel()

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			r.Errorf("timeout waiting for task to stop after %v", r.svc.GetCancelTimeout())
			// At this point, process might be completely stuck, log and return error
			return fmt.Errorf("task cancellation timeout: process may be stuck")
		case <-ticker.C:
			if !utils.ProcessIdExists(r.pid) {
				r.Debugf("process[%d] terminated successfully", r.pid)
				// Wait for background goroutines to finish with timeout
				done := make(chan struct{})
				go func() {
					r.wg.Wait()
					close(done)
				}()

				select {
				case <-done:
					r.Debugf("all background goroutines stopped")
				case <-time.After(5 * time.Second):
					r.Warnf("some background goroutines did not stop within timeout")
				}
				return nil
			}
		}
	}
}

func (r *Runner) SetSubscribeTimeout(timeout time.Duration) {
	r.subscribeTimeout = timeout
}

func (r *Runner) GetTaskId() (id primitive.ObjectID) {
	return r.tid
}

// startHealthCheck periodically verifies that the process is still running
// If the process disappears unexpectedly, it signals a task lost condition
func (r *Runner) startHealthCheck() {
	r.wg.Add(1)
	defer r.wg.Done()

	if r.cmd.ProcessState == nil || r.cmd.ProcessState.Exited() {
		return
	}

	ticker := time.NewTicker(r.healthCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-r.ctx.Done():
			return
		case <-ticker.C:
			if !utils.ProcessIdExists(r.pid) {
				// process lost
				r.ch <- constants.TaskSignalLost
				return
			}
		}
	}
}

// wait monitors the process execution and sends appropriate signals based on the exit status:
// - TaskSignalFinish for successful completion
// - TaskSignalCancel for cancellation
// - TaskSignalError for execution errors
func (r *Runner) wait() (err error) {
	// start a goroutine to wait for process to finish
	go func() {
		r.Debugf("waiting for process[%d] to finish", r.pid)
		err = r.cmd.Wait()
		if err != nil {
			var exitError *exec.ExitError
			if !errors.As(err, &exitError) {
				r.ch <- constants.TaskSignalError
				r.Debugf("process[%d] exited with error: %v", r.pid, err)
				return
			}
			exitCode := exitError.ExitCode()
			if exitCode == -1 {
				// cancel error
				r.ch <- constants.TaskSignalCancel
				r.Debugf("process[%d] cancelled", r.pid)
				return
			}

			// standard error
			r.err = err
			r.ch <- constants.TaskSignalError
			r.Debugf("process[%d] exited with error: %v", r.pid, err)
			return
		}

		// success
		r.ch <- constants.TaskSignalFinish
		r.Debugf("process[%d] exited successfully", r.pid)
	}()

	// declare task status
	status := ""

	// wait for signal
	signal := <-r.ch
	switch signal {
	case constants.TaskSignalFinish:
		err = nil
		status = constants.TaskStatusFinished
	case constants.TaskSignalCancel:
		err = constants.ErrTaskCancelled
		status = constants.TaskStatusCancelled
	case constants.TaskSignalError:
		err = r.err
		status = constants.TaskStatusError
	case constants.TaskSignalLost:
		err = constants.ErrTaskLost
		status = constants.TaskStatusError
		// ZOMBIE PREVENTION: Clean up any remaining processes when task is lost
		go r.cleanupOrphanedProcesses()
	default:
		err = constants.ErrInvalidSignal
		status = constants.TaskStatusError
	}

	// update task status
	if err := r.updateTask(status, err); err != nil {
		r.Errorf("error updating task status: %v", err)
		return err
	}

	// log according to status
	switch status {
	case constants.TaskStatusFinished:
		r.Infof("task[%s] finished", r.tid.Hex())
	case constants.TaskStatusCancelled:
		r.Infof("task[%s] cancelled", r.tid.Hex())
	case constants.TaskStatusError:
		r.Errorf("task[%s] error: %v", r.tid.Hex(), err)
	default:
		r.Errorf("invalid task status: %s", status)
	}

	return nil
}

// updateTask updates the task status and related statistics in the database
// If running on a worker node, updates are sent to the master
func (r *Runner) updateTask(status string, e error) (err error) {
	if status != "" {
		r.Debugf("updating task status to: %s", status)
	}

	if r.t != nil && status != "" {
		// Cache status locally first (always succeeds)
		r.cacheTaskStatus(status, e)

		// update task status
		r.t.Status = status
		if e != nil {
			r.t.Error = e.Error()
		}
		if utils.IsMaster() {
			err = service.NewModelService[models.Task]().ReplaceById(r.t.Id, *r.t)
			if err != nil {
				r.Warnf("failed to update task in database, but cached locally: %v", err)
				// Don't return error - the status is cached and will be synced later
			}
		} else {
			err = client.NewModelService[models.Task]().ReplaceById(r.t.Id, *r.t)
			if err != nil {
				r.Warnf("failed to update task in database, but cached locally: %v", err)
				// Don't return error - the status is cached and will be synced later
			}
		}

		// update stats (only if database update succeeded)
		if err == nil {
			r.updateTaskStat(status)
			r.updateSpiderStat(status)
		}

		// send notification
		go r.sendNotification()
	}

	// get task
	r.Debugf("fetching updated task from database")
	r.t, err = r.svc.GetTaskById(r.tid)
	if err != nil {
		r.Errorf("failed to get updated task: %v", err)
		return err
	}

	return nil
}

// initConnection establishes a gRPC connection to the task service with retry logic
func (r *Runner) initConnection() (err error) {
	r.connMutex.Lock()
	defer r.connMutex.Unlock()

	taskClient, err := client2.GetGrpcClient().GetTaskClient()
	if err != nil {
		r.Errorf("failed to get task client: %v", err)
		return err
	}
	r.conn, err = taskClient.Connect(r.ctx)
	if err != nil {
		r.Errorf("error connecting to task service: %v", err)
		return err
	}

	r.lastConnCheck = time.Now()
	r.connRetryAttempts = 0
	// Start connection health monitoring for all tasks (potentially long-running)
	go r.monitorConnectionHealth()

	// Start periodic resource cleanup for all tasks
	go r.performPeriodicCleanup()

	return nil
}

// monitorConnectionHealth periodically checks gRPC connection health and reconnects if needed
func (r *Runner) monitorConnectionHealth() {
	r.wg.Add(1)
	defer r.wg.Done()

	r.connHealthTicker = time.NewTicker(r.connHealthInterval)
	defer r.connHealthTicker.Stop()

	for {
		select {
		case <-r.ctx.Done():
			return
		case <-r.connHealthTicker.C:
			if r.isConnectionHealthy() {
				r.lastConnCheck = time.Now()
				r.connRetryAttempts = 0
			} else {
				r.Warnf("gRPC connection unhealthy, attempting reconnection (attempt %d/%d)",
					r.connRetryAttempts+1, r.maxConnRetries)
				if err := r.reconnectWithRetry(); err != nil {
					r.Errorf("failed to reconnect after %d attempts: %v", r.maxConnRetries, err)
				}
			}
		}
	}
}

// isConnectionHealthy checks if the gRPC connection is still healthy
// Uses a non-blocking approach to prevent interfering with log streams
func (r *Runner) isConnectionHealthy() bool {
	r.connMutex.RLock()
	conn := r.conn
	r.connMutex.RUnlock()

	if conn == nil {
		return false
	}

	// Check if context is already cancelled - don't do health checks during cancellation
	select {
	case <-r.ctx.Done():
		r.Debugf("skipping health check - task is being cancelled")
		return false
	default:
	}

	// FIXED: Use a completely non-blocking approach to prevent stream interference
	// Instead of sending data that could block the log stream, just check connection state
	// and use timing-based health assessment

	// Check if we've had recent successful operations
	timeSinceLastCheck := time.Since(r.lastConnCheck)

	// If we haven't checked recently, consider it healthy if not too old
	// This prevents health checks from interfering with active log streaming
	if timeSinceLastCheck < 2*time.Minute {
		r.Debugf("connection considered healthy based on recent activity")
		return true
	}

	// For older connections, try a non-blocking ping only if no active log streaming
	// This is a compromise to avoid blocking the critical log data flow
	pingMsg := &grpc.TaskServiceConnectRequest{
		Code:   grpc.TaskServiceConnectCode_PING,
		TaskId: r.tid.Hex(),
		Data:   nil,
	}

	// Use a very short timeout and non-blocking approach
	done := make(chan error, 1)
	go func() {
		// Re-acquire lock only for the send operation
		r.connMutex.RLock()
		defer r.connMutex.RUnlock()
		if r.conn != nil {
			done <- r.conn.Send(pingMsg)
		} else {
			done <- fmt.Errorf("connection is nil")
		}
	}()

	// Very short timeout to prevent blocking log operations
	select {
	case err := <-done:
		if err != nil {
			r.Debugf("connection health check failed: %v", err)
			return false
		}
		r.Debugf("connection health check successful")
		return true
	case <-time.After(1 * time.Second): // Much shorter timeout
		r.Debugf("connection health check timed out quickly - assume healthy to avoid blocking logs")
		return true // Assume healthy to avoid disrupting log flow
	case <-r.ctx.Done():
		r.Debugf("connection health check cancelled")
		return false
	}
}

// reconnectWithRetry attempts to reconnect to the gRPC service with exponential backoff
func (r *Runner) reconnectWithRetry() error {
	r.connMutex.Lock()
	defer r.connMutex.Unlock()

	for attempt := 0; attempt < r.maxConnRetries; attempt++ {
		r.connRetryAttempts = attempt + 1

		// Close existing connection
		if r.conn != nil {
			_ = r.conn.CloseSend()
			r.conn = nil
		}

		// Wait before retry (exponential backoff)
		if attempt > 0 {
			backoffDelay := time.Duration(attempt) * r.connRetryDelay
			r.Debugf("waiting %v before retry attempt %d", backoffDelay, attempt+1)

			select {
			case <-r.ctx.Done():
				return fmt.Errorf("context cancelled during reconnection")
			case <-time.After(backoffDelay):
			}
		}

		// Attempt reconnection
		taskClient, err := client2.GetGrpcClient().GetTaskClient()
		if err != nil {
			r.Warnf("reconnection attempt %d failed to get task client: %v", attempt+1, err)
			continue
		}
		conn, err := taskClient.Connect(r.ctx)
		if err != nil {
			r.Warnf("reconnection attempt %d failed: %v", attempt+1, err)
			continue
		}

		r.conn = conn
		r.lastConnCheck = time.Now()
		r.connRetryAttempts = 0
		r.Infof("successfully reconnected to task service after %d attempts", attempt+1)

		// Reset log circuit breaker when connection is restored
		r.logConnMutex.Lock()
		if !r.logConnHealthy {
			r.logConnHealthy = true
			r.logFailureCount = 0
			r.Logger.Info("log circuit breaker reset after successful reconnection")
		}
		r.logConnMutex.Unlock()

		// Sync pending status updates after successful reconnection
		go func() {
			if err := r.syncPendingStatusUpdates(); err != nil {
				r.Errorf("failed to sync pending status updates after reconnection: %v", err)
			}
		}()

		return nil
	}

	return fmt.Errorf("failed to reconnect after %d attempts", r.maxConnRetries)
}

// updateTaskStat updates task statistics based on the current status:
// - For running tasks: sets start time and wait duration
// - For completed tasks: sets end time and calculates durations
func (r *Runner) updateTaskStat(status string) {
	if status != "" {
		r.Debugf("updating task statistics for status: %s", status)
	}

	ts, err := client.NewModelService[models.TaskStat]().GetById(r.tid)
	if err != nil {
		r.Errorf("error getting task stat: %v", err)
		return
	}

	r.Debugf("current task statistics - wait_duration: %dms, runtime_duration: %dms", ts.WaitDuration, ts.RuntimeDuration)

	switch status {
	case constants.TaskStatusPending:
		// do nothing
	case constants.TaskStatusRunning:
		ts.StartedAt = time.Now()
		ts.WaitDuration = ts.StartedAt.Sub(ts.CreatedAt).Milliseconds()
	case constants.TaskStatusFinished, constants.TaskStatusError, constants.TaskStatusCancelled:
		if ts.StartedAt.IsZero() {
			ts.StartedAt = time.Now()
			ts.WaitDuration = ts.StartedAt.Sub(ts.CreatedAt).Milliseconds()
		}
		ts.EndedAt = time.Now()
		ts.RuntimeDuration = ts.EndedAt.Sub(ts.StartedAt).Milliseconds()
		ts.TotalDuration = ts.EndedAt.Sub(ts.CreatedAt).Milliseconds()
	}
	if utils.IsMaster() {
		err = service.NewModelService[models.TaskStat]().ReplaceById(ts.Id, *ts)
		if err != nil {
			r.Errorf("error updating task stat: %v", err)
			return
		}
	} else {
		err = client.NewModelService[models.TaskStat]().ReplaceById(ts.Id, *ts)
		if err != nil {
			r.Errorf("error updating task stat: %v", err)
			return
		}
	}
}

// sendNotification sends a notification to the task service
func (r *Runner) sendNotification() {
	req := &grpc.TaskServiceSendNotificationRequest{
		NodeKey: r.svc.GetNodeConfigService().GetNodeKey(),
		TaskId:  r.tid.Hex(),
	}
	taskClient, err := client2.GetGrpcClient().GetTaskClient()
	if err != nil {
		r.Errorf("failed to get task client: %v", err)
		return
	}

	// Use independent context for async notification - prevents cancellation due to task lifecycle
	// This ensures notifications are sent even if the task runner is being cleaned up
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err = taskClient.SendNotification(ctx, req)
	if err != nil {
		if !errors.Is(ctx.Err(), context.DeadlineExceeded) {
			r.Errorf("error sending notification: %v", err)
		}
		return
	}
}

// updateSpiderStat updates spider statistics based on task completion:
// - Updates last task ID
// - Increments task counts
// - Updates duration metrics
func (r *Runner) updateSpiderStat(status string) {
	// task stat
	ts, err := client.NewModelService[models.TaskStat]().GetById(r.tid)
	if err != nil {
		r.Errorf("error getting task stat: %v", err)
		return
	}

	// update
	var update bson.M
	switch status {
	case constants.TaskStatusPending, constants.TaskStatusRunning:
		update = bson.M{
			"$set": bson.M{
				"last_task_id": r.tid, // last task id
			},
			"$inc": bson.M{
				"tasks":         1,               // task count
				"wait_duration": ts.WaitDuration, // wait duration
			},
		}
	case constants.TaskStatusFinished, constants.TaskStatusError, constants.TaskStatusCancelled:
		update = bson.M{
			"$set": bson.M{
				"last_task_id": r.tid, // last task id
			},
			"$inc": bson.M{
				"results":          ts.ResultCount,            // results
				"runtime_duration": ts.RuntimeDuration / 1000, // runtime duration
				"total_duration":   ts.TotalDuration / 1000,   // total duration
			},
		}
	default:
		r.Errorf("Invalid task status: %s", status)
		return
	}

	// perform update
	if utils.IsMaster() {
		err = service.NewModelService[models.SpiderStat]().UpdateById(r.s.Id, update)
		if err != nil {
			r.Errorf("error updating spider stat: %v", err)
			return
		}
	} else {
		err = client.NewModelService[models.SpiderStat]().UpdateById(r.s.Id, update)
		if err != nil {
			r.Errorf("error updating spider stat: %v", err)
			return
		}
	}
}

func (r *Runner) installDependenciesIfAvailable() (err error) {
	if !utils.IsPro() {
		return nil
	}

	// Get dependency installer service
	depSvc := dependency.GetDependencyInstallerRegistryService()
	if depSvc == nil {
		r.Warnf("dependency installer service not available")
		return nil
	}

	// Check if auto install is enabled
	if !depSvc.IsAutoInstallEnabled() {
		r.Debug("auto dependency installation is disabled")
		return nil
	}

	// Get install command
	cmd, err := depSvc.GetInstallDependencyRequirementsCmdBySpiderId(r.s.Id)
	if err != nil {
		return err
	}
	if cmd == nil {
		return nil
	}

	// Set up pipes for stdout and stderr
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		r.Errorf("error creating stdout pipe for dependency installation: %v", err)
		return err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		r.Errorf("error creating stderr pipe for dependency installation: %v", err)
		return err
	}

	// Start the command
	r.Infof("installing dependencies for spider: %s", r.s.Id.Hex())
	r.Infof("command for dependencies installation: %s", cmd.String())
	if err := cmd.Start(); err != nil {
		r.Errorf("error starting dependency installation command: %v", err)
		return err
	}

	// Create wait group for log readers
	var wg sync.WaitGroup
	wg.Add(2)

	// Read stdout
	go func() {
		defer wg.Done()
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			line := scanner.Text()
			r.Info(line)
		}
	}()

	// Read stderr
	go func() {
		defer wg.Done()
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			line := scanner.Text()
			r.Error(line)
		}
	}()

	// Wait for command to complete
	if err := cmd.Wait(); err != nil {
		r.Errorf("dependency installation failed: %v", err)
		return err
	}

	// Wait for log readers to finish
	wg.Wait()

	return nil
}

// GetConnectionStats returns connection health statistics for monitoring
func (r *Runner) GetConnectionStats() map[string]interface{} {
	r.connMutex.RLock()
	defer r.connMutex.RUnlock()

	return map[string]interface{}{
		"last_connection_check": r.lastConnCheck,
		"retry_attempts":        r.connRetryAttempts,
		"max_retries":           r.maxConnRetries,
		"connection_healthy":    r.isConnectionHealthy(),
		"connection_exists":     r.conn != nil,
	}
}
