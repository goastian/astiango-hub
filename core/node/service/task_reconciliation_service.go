package service

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/goastian/astiango-hub/core/constants"
	"github.com/goastian/astiango-hub/core/grpc/server"
	"github.com/goastian/astiango-hub/core/interfaces"
	"github.com/goastian/astiango-hub/core/models/models"
	"github.com/goastian/astiango-hub/core/models/service"
	"github.com/goastian/astiango-hub/core/task/handler"
	"github.com/goastian/astiango-hub/core/utils"
	"github.com/goastian/astiango-hub/grpc"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	grpc2 "google.golang.org/grpc"
)

const (
	staleReconciliationThreshold = 15 * time.Minute
	assignedTaskTimeout          = 10 * time.Minute
	pendingTaskTimeout           = 30 * time.Minute
	reconciliationInterval       = 30 * time.Second
	assignedCleanupInterval      = 2 * time.Minute
)

// TaskReconciliationService handles task status reconciliation for node disconnection scenarios.
// It ensures task statuses remain accurate even when nodes go offline/online unexpectedly.
type TaskReconciliationService struct {
	server         *server.GrpcServer
	taskHandlerSvc *handler.Service
	interfaces.Logger
}

// ============================================================================
// Section 1: Public API - Node Lifecycle Events
// ============================================================================

// HandleTasksForOfflineNode marks all running tasks as disconnected when node goes offline
func (svc *TaskReconciliationService) HandleTasksForOfflineNode(node *models.Node) {
	runningTasks := svc.findTasksByStatus(node.Id, constants.TaskStatusRunning)
	if len(runningTasks) == 0 {
		svc.Debugf("no running tasks found for offline node[%s]", node.Key)
		return
	}

	svc.Infof("marking %d running tasks as disconnected for offline node[%s]", len(runningTasks), node.Key)
	for _, task := range runningTasks {
		svc.markTaskDisconnected(&task, node.Key)
	}
}

// HandleNodeReconnection reconciles tasks when node comes back online
func (svc *TaskReconciliationService) HandleNodeReconnection(node *models.Node) {
	svc.reconcileDisconnectedTasks(node)
	svc.reconcileAbandonedAssignedTasks(node)
	svc.reconcileStalePendingTasks(node)
}

// StartPeriodicReconciliation starts background goroutines for periodic task reconciliation
func (svc *TaskReconciliationService) StartPeriodicReconciliation() {
	go svc.runPeriodicReconciliation()
	go svc.runPeriodicAssignedTaskCleanup()
}

// ForceReconcileTask forces reconciliation of a specific task (for manual intervention)
func (svc *TaskReconciliationService) ForceReconcileTask(taskId primitive.ObjectID) error {
	task, err := service.NewModelService[models.Task]().GetById(taskId)
	if err != nil {
		return fmt.Errorf("failed to get task[%s]: %w", taskId.Hex(), err)
	}
	return svc.reconcileTaskStatus(task)
}

// ValidateTaskStatus ensures task status is consistent with actual process state
func (svc *TaskReconciliationService) ValidateTaskStatus(task *models.Task) error {
	if task == nil {
		return fmt.Errorf("task is nil")
	}

	node, err := service.NewModelService[models.Node]().GetById(task.NodeId)
	if err != nil {
		return fmt.Errorf("failed to get node for task: %w", err)
	}

	actualStatus, err := svc.getActualTaskStatus(node, task)
	if err != nil {
		return fmt.Errorf("failed to get actual task status: %w", err)
	}

	if actualStatus != task.Status {
		svc.Warnf("task[%s] status inconsistency: database='%s', actual='%s'",
			task.Id.Hex(), task.Status, actualStatus)
		return svc.updateTaskStatus(task, actualStatus)
	}

	return nil
}

// ============================================================================
// Section 2: Reconciliation Logic - Node Reconnection Handlers
// ============================================================================

func (svc *TaskReconciliationService) reconcileDisconnectedTasks(node *models.Node) {
	tasks := svc.findTasksByStatus(node.Id, constants.TaskStatusNodeDisconnected)
	if len(tasks) == 0 {
		svc.Debugf("no disconnected tasks found for node[%s]", node.Key)
		return
	}

	svc.Infof("reconciling %d disconnected tasks for node[%s]", len(tasks), node.Key)
	for _, task := range tasks {
		svc.reconcileDisconnectedTask(node, &task)
	}
}

func (svc *TaskReconciliationService) reconcileDisconnectedTask(node *models.Node, task *models.Task) {
	// Try to sync from worker cache first
	_ = svc.triggerWorkerStatusSync(task)

	actualStatus, err := svc.getActualTaskStatus(node, task)
	if err != nil {
		svc.Warnf("cannot determine actual status for task[%s]: %v", task.Id.Hex(), err)
		if task.Error == "" {
			task.Error = "Unable to verify task status after node reconnection"
		}
		return
	}

	// Update with actual status
	task.Status = actualStatus
	if actualStatus == constants.TaskStatusFinished {
		task.Error = ""
	} else if actualStatus == constants.TaskStatusError {
		task.Error = "Task encountered an error during node disconnection"
	}

	if err := svc.saveTask(task); err != nil {
		svc.Errorf("failed to save reconciled task[%s]: %v", task.Id.Hex(), err)
	} else {
		svc.Infof("reconciled task[%s] from disconnected to '%s'", task.Id.Hex(), actualStatus)
	}
}

func (svc *TaskReconciliationService) reconcileAbandonedAssignedTasks(node *models.Node) {
	tasks := svc.findTasksByStatus(node.Id, constants.TaskStatusAssigned)
	if len(tasks) == 0 {
		return
	}

	svc.Infof("resetting %d abandoned assigned tasks for node[%s]", len(tasks), node.Key)
	for _, task := range tasks {
		task.Status = constants.TaskStatusPending
		task.Error = "Task reset from 'assigned' to 'pending' after node reconnection"
		if err := svc.saveTask(&task); err != nil {
			svc.Errorf("failed to reset assigned task[%s]: %v", task.Id.Hex(), err)
		}
	}
}

func (svc *TaskReconciliationService) reconcileStalePendingTasks(node *models.Node) {
	tasks := svc.findTasksByStatus(node.Id, constants.TaskStatusPending)
	if len(tasks) == 0 {
		return
	}

	staleCount := 0
	for _, task := range tasks {
		if time.Since(task.CreatedAt) > pendingTaskTimeout {
			staleCount++
			svc.handleStalePendingTask(node, &task)
		}
	}

	if staleCount > 0 {
		svc.Infof("handled %d stale pending tasks for node[%s]", staleCount, node.Key)
	}
}

func (svc *TaskReconciliationService) handleStalePendingTask(node *models.Node, task *models.Task) {
	svc.Warnf("task[%s] pending for %v, attempting recovery", task.Id.Hex(), time.Since(task.CreatedAt))

	originalNodeKey := node.Key
	originalNodeId := task.NodeId

	availableNode, err := svc.findAvailableNodeForTask(task)
	if err == nil && availableNode != nil {
		// Re-assign to available node
		task.NodeId = availableNode.Id
		task.Error = fmt.Sprintf("Pending for %v on node %s (%s), re-assigned to %s",
			time.Since(task.CreatedAt), originalNodeKey, originalNodeId.Hex(), availableNode.Key)
		svc.Infof("re-assigned stale task[%s] to node[%s]", task.Id.Hex(), availableNode.Key)
	} else {
		// Mark as abnormal
		task.Status = constants.TaskStatusAbnormal
		task.Error = fmt.Sprintf("Pending for %v on node %s (%s), no available nodes for re-assignment",
			time.Since(task.CreatedAt), originalNodeKey, originalNodeId.Hex())
		svc.Warnf("marked stale task[%s] as abnormal", task.Id.Hex())
	}

	if err := svc.saveTask(task); err != nil {
		svc.Errorf("failed to handle stale pending task[%s]: %v", task.Id.Hex(), err)
	}
}

// ============================================================================
// Section 3: Status Querying - Get Actual Task Status from Worker
// ============================================================================

func (svc *TaskReconciliationService) getActualTaskStatus(node *models.Node, task *models.Task) (string, error) {
	// Priority 1: Worker cache (most accurate)
	if cachedStatus, err := svc.getStatusFromWorkerCache(task); err == nil && cachedStatus != "" {
		svc.Debugf("retrieved cached status for task[%s]: %s", task.Id.Hex(), cachedStatus)
		return cachedStatus, nil
	}

	// Priority 2: Query process status from worker
	processStatus, err := svc.queryProcessStatus(node, task)
	if err != nil {
		return "", fmt.Errorf("unable to determine task status: %w", err)
	}

	return processStatus, nil
}

func (svc *TaskReconciliationService) getStatusFromWorkerCache(task *models.Task) (string, error) {
	if svc.taskHandlerSvc == nil {
		return "", fmt.Errorf("task handler service not available")
	}

	taskRunner := svc.taskHandlerSvc.GetTaskRunner(task.Id)
	if taskRunner == nil {
		return "", fmt.Errorf("no active task runner")
	}

	runner, ok := taskRunner.(*handler.Runner)
	if !ok {
		return "", fmt.Errorf("unexpected task runner type")
	}

	cachedSnapshot := runner.GetCachedTaskStatus()
	if cachedSnapshot == nil {
		return "", fmt.Errorf("no cached status available")
	}

	return cachedSnapshot.Status, nil
}

func (svc *TaskReconciliationService) triggerWorkerStatusSync(task *models.Task) error {
	if svc.taskHandlerSvc == nil {
		return fmt.Errorf("task handler service not available")
	}

	taskRunner := svc.taskHandlerSvc.GetTaskRunner(task.Id)
	if taskRunner == nil {
		return fmt.Errorf("no active task runner")
	}

	runner, ok := taskRunner.(*handler.Runner)
	if !ok {
		return fmt.Errorf("unexpected task runner type")
	}

	if err := runner.SyncPendingStatusUpdates(); err != nil {
		return fmt.Errorf("failed to sync: %w", err)
	}

	svc.Infof("triggered status sync for task[%s]", task.Id.Hex())
	return nil
}

func (svc *TaskReconciliationService) queryProcessStatus(node *models.Node, task *models.Task) (string, error) {
	// Check task stream exists
	_, hasActiveStream := svc.server.TaskSvr.GetSubscribeStream(task.Id)
	if hasActiveStream {
		return constants.TaskStatusRunning, nil
	}

	// Check node connection
	nodeStream, nodeConnected := svc.server.NodeSvr.GetSubscribeStream(node.Id)
	if !nodeConnected {
		return "", fmt.Errorf("node not connected")
	}

	// Query worker for process status
	if nodeStream != nil && task.Pid > 0 {
		return svc.requestProcessStatusFromWorker(node, task, 5*time.Second)
	}

	return "", fmt.Errorf("unable to determine process status")
}

func (svc *TaskReconciliationService) requestProcessStatusFromWorker(node *models.Node, task *models.Task, timeout time.Duration) (string, error) {
	if task.Pid <= 0 {
		return "", fmt.Errorf("invalid PID: %d", task.Pid)
	}

	// TODO: Implement actual gRPC call to worker
	// For now, this is a placeholder that demonstrates the intended architecture
	client, err := svc.createWorkerClient(node)
	if err != nil {
		return "", fmt.Errorf("failed to create worker client: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	req := &grpc.TaskServiceCheckProcessRequest{
		TaskId: task.Id.Hex(),
		Pid:    int32(task.Pid),
	}

	resp, err := client.CheckProcess(ctx, req)
	if err != nil {
		return "", fmt.Errorf("worker query failed: %w", err)
	}

	taskStatus := svc.mapProcessStatusToTaskStatus(resp.Status, resp.ExitCode, task)
	svc.Infof("worker reported status for task[%s]: %s (exit=%d)", task.Id.Hex(), taskStatus, resp.ExitCode)

	return taskStatus, nil
}

func (svc *TaskReconciliationService) mapProcessStatusToTaskStatus(processStatus grpc.ProcessStatus, exitCode int32, task *models.Task) string {
	switch processStatus {
	case grpc.ProcessStatus_PROCESS_RUNNING:
		return constants.TaskStatusRunning

	case grpc.ProcessStatus_PROCESS_FINISHED:
		if exitCode == 0 {
			return constants.TaskStatusFinished
		}
		return constants.TaskStatusError

	case grpc.ProcessStatus_PROCESS_ERROR, grpc.ProcessStatus_PROCESS_ZOMBIE:
		return constants.TaskStatusError

	case grpc.ProcessStatus_PROCESS_NOT_FOUND:
		// Recently active task with missing process likely completed
		if time.Since(task.UpdatedAt) < 5*time.Minute {
			if task.Error != "" {
				return constants.TaskStatusError
			}
			return constants.TaskStatusFinished
		}
		return constants.TaskStatusError

	default:
		svc.Warnf("unknown process status %v for task[%s]", processStatus, task.Id.Hex())
		return constants.TaskStatusError
	}
}

func (svc *TaskReconciliationService) createWorkerClient(node *models.Node) (grpc.TaskServiceClient, error) {
	// Check if we have an active gRPC stream to this node
	// This indicates the node is connected and can receive requests
	_, hasStream := svc.server.NodeSvr.GetSubscribeStream(node.Id)
	if !hasStream {
		return nil, fmt.Errorf("node[%s] not connected via gRPC stream", node.Key)
	}

	// Use the existing gRPC server's task service server
	// The master node has a TaskServiceServer that can handle CheckProcess requests
	// We'll use it through the server's registered service
	taskClient := &workerTaskClient{
		server: svc.server.TaskSvr,
		nodeId: node.Id,
		logger: svc.Logger,
	}

	return taskClient, nil
}

// workerTaskClient wraps the TaskServiceServer to provide TaskServiceClient interface
// This allows the reconciliation service to query worker process status through the gRPC server
type workerTaskClient struct {
	server *server.TaskServiceServer
	nodeId primitive.ObjectID
	logger interfaces.Logger
}

func (c *workerTaskClient) Subscribe(ctx context.Context, in *grpc.TaskServiceSubscribeRequest, opts ...grpc2.CallOption) (grpc2.ServerStreamingClient[grpc.TaskServiceSubscribeResponse], error) {
	return nil, fmt.Errorf("Subscribe not implemented for worker task client")
}

func (c *workerTaskClient) Connect(ctx context.Context, opts ...grpc2.CallOption) (grpc2.BidiStreamingClient[grpc.TaskServiceConnectRequest, grpc.TaskServiceConnectResponse], error) {
	return nil, fmt.Errorf("Connect not implemented for worker task client")
}

func (c *workerTaskClient) FetchTask(ctx context.Context, in *grpc.TaskServiceFetchTaskRequest, opts ...grpc2.CallOption) (*grpc.TaskServiceFetchTaskResponse, error) {
	return nil, fmt.Errorf("FetchTask not implemented for worker task client")
}

func (c *workerTaskClient) SendNotification(ctx context.Context, in *grpc.TaskServiceSendNotificationRequest, opts ...grpc2.CallOption) (*grpc.TaskServiceSendNotificationResponse, error) {
	return nil, fmt.Errorf("SendNotification not implemented for worker task client")
}

func (c *workerTaskClient) CheckProcess(ctx context.Context, in *grpc.TaskServiceCheckProcessRequest, opts ...grpc2.CallOption) (*grpc.TaskServiceCheckProcessResponse, error) {
	// Call the server's CheckProcess method directly
	// This works because all gRPC requests from workers go through the same server
	c.logger.Debugf("checking process for task[%s] on node[%s]", in.TaskId, c.nodeId.Hex())
	return c.server.CheckProcess(ctx, in)
}

// ============================================================================
// Section 4: Periodic Background Tasks
// ============================================================================

func (svc *TaskReconciliationService) runPeriodicReconciliation() {
	ticker := time.NewTicker(reconciliationInterval)
	defer ticker.Stop()

	for range ticker.C {
		if err := svc.reconcileAllRunningTasks(); err != nil {
			svc.Errorf("periodic reconciliation failed: %v", err)
		}
	}
}

func (svc *TaskReconciliationService) reconcileAllRunningTasks() error {
	query := bson.M{
		"status": bson.M{
			"$in": []string{constants.TaskStatusRunning, constants.TaskStatusNodeDisconnected},
		},
	}

	tasks, err := service.NewModelService[models.Task]().GetMany(query, nil)
	if err != nil {
		return err
	}

	svc.Debugf("reconciling %d tasks", len(tasks))
	for _, task := range tasks {
		if err := svc.reconcileTaskStatus(&task); err != nil {
			svc.Errorf("failed to reconcile task[%s]: %v", task.Id.Hex(), err)
		}
	}

	return nil
}

func (svc *TaskReconciliationService) reconcileTaskStatus(task *models.Task) error {
	node, err := service.NewModelService[models.Node]().GetById(task.NodeId)
	if err != nil {
		return fmt.Errorf("failed to get node: %w", err)
	}

	actualStatus, err := svc.getActualTaskStatus(node, task)
	if err != nil {
		if svc.shouldMarkTaskAbnormal(task) {
			return svc.markTaskAbnormal(task, err)
		}
		return err
	}

	if actualStatus != task.Status {
		svc.Infof("reconciling task[%s]: '%s' -> '%s'", task.Id.Hex(), task.Status, actualStatus)
		return svc.updateTaskStatus(task, actualStatus)
	}

	return nil
}

func (svc *TaskReconciliationService) runPeriodicAssignedTaskCleanup() {
	ticker := time.NewTicker(assignedCleanupInterval)
	defer ticker.Stop()

	for range ticker.C {
		if err := svc.cleanupStuckAssignedTasks(); err != nil {
			svc.Errorf("assigned task cleanup failed: %v", err)
		}
	}
}

func (svc *TaskReconciliationService) cleanupStuckAssignedTasks() error {
	cutoff := time.Now().Add(-assignedTaskTimeout)
	query := bson.M{
		"status": constants.TaskStatusAssigned,
		"$or": []bson.M{
			{"updated_at": bson.M{"$lt": cutoff}},
			{"updated_at": bson.M{"$exists": false}},
		},
	}

	tasks, err := service.NewModelService[models.Task]().GetMany(query, nil)
	if err != nil {
		return err
	}

	if len(tasks) == 0 {
		return nil
	}

	svc.Infof("cleaning up %d stuck assigned tasks", len(tasks))
	for _, task := range tasks {
		svc.handleStuckAssignedTask(&task)
	}

	return nil
}

func (svc *TaskReconciliationService) handleStuckAssignedTask(task *models.Task) {
	node, err := service.NewModelService[models.Node]().GetById(task.NodeId)
	if err != nil {
		svc.Warnf("failed to get node for task[%s]: %v", task.Id.Hex(), err)
		return
	}

	// Only handle if node is offline or inactive
	if node.Status == constants.NodeStatusOnline && node.Active {
		svc.Warnf("task[%s] assigned to online node[%s] for %v without starting",
			task.Id.Hex(), node.Key, time.Since(task.UpdatedAt))
		return
	}

	originalNodeKey := node.Key
	originalNodeId := task.NodeId

	availableNode, err := svc.findAvailableNodeForTask(task)
	if err == nil && availableNode != nil {
		// Re-assign to available node
		task.Status = constants.TaskStatusPending
		task.NodeId = availableNode.Id
		task.Error = fmt.Sprintf("Stuck in assigned for 10+ min on node %s (%s), re-assigned to %s",
			originalNodeKey, originalNodeId.Hex(), availableNode.Key)
		svc.Infof("re-assigned stuck task[%s] to node[%s]", task.Id.Hex(), availableNode.Key)
	} else {
		// Mark as abnormal
		task.Status = constants.TaskStatusAbnormal
		task.Error = fmt.Sprintf("Stuck in assigned for 10+ min on node %s (%s), no available nodes",
			originalNodeKey, originalNodeId.Hex())
		svc.Infof("marked stuck task[%s] as abnormal", task.Id.Hex())
	}

	if err := svc.saveTask(task); err != nil {
		svc.Errorf("failed to handle stuck assigned task[%s]: %v", task.Id.Hex(), err)
	}
}

// ============================================================================
// Section 5: Database Operations & Utilities
// ============================================================================

func (svc *TaskReconciliationService) findTasksByStatus(nodeId primitive.ObjectID, status string) []models.Task {
	query := bson.M{"node_id": nodeId, "status": status}
	tasks, err := service.NewModelService[models.Task]().GetMany(query, nil)
	if err != nil {
		svc.Errorf("failed to query tasks with status[%s] for node[%s]: %v", status, nodeId.Hex(), err)
		return nil
	}
	return tasks
}

func (svc *TaskReconciliationService) markTaskDisconnected(task *models.Task, nodeKey string) {
	task.Status = constants.TaskStatusNodeDisconnected
	task.Error = "Task temporarily disconnected due to worker node offline"
	if err := svc.saveTask(task); err != nil {
		svc.Errorf("failed to mark task[%s] disconnected for node[%s]: %v", task.Id.Hex(), nodeKey, err)
	} else {
		svc.Debugf("marked task[%s] as disconnected for node[%s]", task.Id.Hex(), nodeKey)
	}
}

func (svc *TaskReconciliationService) updateTaskStatus(task *models.Task, newStatus string) error {
	task.Status = newStatus

	switch newStatus {
	case constants.TaskStatusFinished:
		task.Error = ""
	case constants.TaskStatusError:
		if task.Error == "" {
			task.Error = "Task status synchronized from actual process state"
		}
	case constants.TaskStatusAbnormal:
		if task.Error == "" {
			task.Error = "Task marked as abnormal during status reconciliation"
		}
	}

	return svc.saveTask(task)
}

func (svc *TaskReconciliationService) saveTask(task *models.Task) error {
	task.SetUpdated(primitive.NilObjectID)
	return backoff.Retry(func() error {
		return service.NewModelService[models.Task]().ReplaceById(task.Id, *task)
	}, backoff.WithMaxRetries(backoff.NewConstantBackOff(500*time.Millisecond), 3))
}

func (svc *TaskReconciliationService) shouldMarkTaskAbnormal(task *models.Task) bool {
	if task == nil || svc.IsTaskStatusFinal(task.Status) {
		return false
	}

	if task.Status != constants.TaskStatusNodeDisconnected {
		return false
	}

	lastUpdated := task.UpdatedAt
	if lastUpdated.IsZero() {
		lastUpdated = task.CreatedAt
	}

	return !lastUpdated.IsZero() && time.Since(lastUpdated) >= staleReconciliationThreshold
}

func (svc *TaskReconciliationService) markTaskAbnormal(task *models.Task, cause error) error {
	if task == nil {
		return fmt.Errorf("task is nil")
	}

	reasonParts := []string{}
	if cause != nil {
		reasonParts = append(reasonParts, fmt.Sprintf("reconciliation error: %v", cause))
	}
	reasonParts = append(reasonParts, fmt.Sprintf("not reconciled for %s", staleReconciliationThreshold))
	reason := strings.Join(reasonParts, "; ")

	if task.Error == "" {
		task.Error = reason
	} else if !strings.Contains(task.Error, reason) {
		task.Error = fmt.Sprintf("%s; %s", task.Error, reason)
	}

	if err := svc.updateTaskStatus(task, constants.TaskStatusAbnormal); err != nil {
		svc.Errorf("failed to mark task[%s] abnormal: %v", task.Id.Hex(), err)
		return err
	}

	svc.Warnf("marked task[%s] as abnormal after %s", task.Id.Hex(), staleReconciliationThreshold)
	return nil
}

func (svc *TaskReconciliationService) findAvailableNodeForTask(task *models.Task) (*models.Node, error) {
	query := bson.M{
		"status":  constants.NodeStatusOnline,
		"active":  true,
		"enabled": true,
	}

	nodes, err := service.NewModelService[models.Node]().GetMany(query, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to query nodes: %w", err)
	}

	if len(nodes) == 0 {
		return nil, fmt.Errorf("no available nodes")
	}

	// For selected nodes mode, try to find from selected list
	if task.Mode == constants.RunTypeSelectedNodes && len(task.NodeIds) > 0 {
		for _, node := range nodes {
			for _, selectedId := range task.NodeIds {
				if node.Id == selectedId {
					svc.Debugf("found selected node[%s] for task[%s]", node.Key, task.Id.Hex())
					return &node, nil
				}
			}
		}
		svc.Debugf("no selected nodes available for task[%s], using first available", task.Id.Hex())
	}

	// Use first available node
	return &nodes[0], nil
}

// IsTaskStatusFinal returns true if the task is in a terminal state
func (svc *TaskReconciliationService) IsTaskStatusFinal(status string) bool {
	switch status {
	case constants.TaskStatusFinished, constants.TaskStatusError,
		constants.TaskStatusCancelled, constants.TaskStatusAbnormal:
		return true
	default:
		return false
	}
}

// ShouldReconcileTask determines if a task needs reconciliation
func (svc *TaskReconciliationService) ShouldReconcileTask(task *models.Task) bool {
	if svc.IsTaskStatusFinal(task.Status) {
		return false
	}

	switch task.Status {
	case constants.TaskStatusRunning, constants.TaskStatusNodeDisconnected:
		return true
	case constants.TaskStatusPending, constants.TaskStatusAssigned:
		return time.Since(task.CreatedAt) > 10*time.Minute
	default:
		return false
	}
}

// ============================================================================
// Section 6: Constructor & Singleton
// ============================================================================

func NewTaskReconciliationService(server *server.GrpcServer, taskHandlerSvc *handler.Service) *TaskReconciliationService {
	return &TaskReconciliationService{
		server:         server,
		taskHandlerSvc: taskHandlerSvc,
		Logger:         utils.NewLogger("TaskReconciliationService"),
	}
}

var (
	taskReconciliationService     *TaskReconciliationService
	taskReconciliationServiceOnce sync.Once
)

func GetTaskReconciliationService() *TaskReconciliationService {
	taskReconciliationServiceOnce.Do(func() {
		grpcServer := server.GetGrpcServer()
		var taskHandlerSvc *handler.Service
		if !utils.IsMaster() {
			taskHandlerSvc = handler.GetTaskHandlerService()
		}
		taskReconciliationService = NewTaskReconciliationService(grpcServer, taskHandlerSvc)
	})
	return taskReconciliationService
}
