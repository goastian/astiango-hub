package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	mongo3 "github.com/goastian/astiango-hub/core/mongo"

	"github.com/goastian/astiango-hub/core/constants"
	"github.com/goastian/astiango-hub/core/interfaces"
	"github.com/goastian/astiango-hub/core/models/models"
	"github.com/goastian/astiango-hub/core/models/service"
	nodeconfig "github.com/goastian/astiango-hub/core/node/config"
	"github.com/goastian/astiango-hub/core/notification"
	"github.com/goastian/astiango-hub/core/task/stats"
	"github.com/goastian/astiango-hub/core/utils"
	"github.com/goastian/astiango-hub/grpc"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	mongo2 "go.mongodb.org/mongo-driver/mongo"
)

var taskServiceMutex = sync.Mutex{}

type TaskServiceServer struct {
	grpc.UnimplementedTaskServiceServer

	// dependencies
	cfgSvc   interfaces.NodeConfigService
	statsSvc *stats.Service

	// internals
	subs map[primitive.ObjectID]grpc.TaskService_SubscribeServer

	// cleanup mechanism
	cleanupCtx    context.Context
	cleanupCancel context.CancelFunc

	interfaces.Logger
}

func (svr TaskServiceServer) Subscribe(req *grpc.TaskServiceSubscribeRequest, stream grpc.TaskService_SubscribeServer) (err error) {
	// task id
	taskId, err := primitive.ObjectIDFromHex(req.TaskId)
	if err != nil {
		return errors.New("invalid task id")
	}

	// validate stream
	if stream == nil {
		return errors.New("invalid stream")
	}

	svr.Infof("task stream opened: %s", taskId.Hex())

	// Create a context based on client stream
	ctx := stream.Context()

	// add stream and track cancellation function
	taskServiceMutex.Lock()
	svr.subs[taskId] = stream
	taskServiceMutex.Unlock()

	// ensure cleanup on exit
	defer func() {
		taskServiceMutex.Lock()
		delete(svr.subs, taskId)
		taskServiceMutex.Unlock()
		svr.Infof("task stream closed: %s", taskId.Hex())
	}()

	// send periodic heartbeat to detect client disconnection and check for task completion
	heartbeatTicker := time.NewTicker(10 * time.Second) // More frequent for faster completion detection
	defer heartbeatTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			// Stream context cancelled normally (client disconnected or task finished)
			svr.Debugf("task stream context done: %s", taskId.Hex())
			return ctx.Err()

		case <-heartbeatTicker.C:
			// Check if task has finished and close stream if so
			if svr.isTaskFinished(taskId) {
				svr.Infof("task[%s] finished, closing stream", taskId.Hex())
				return nil
			}

			// Check if the context is still valid
			select {
			case <-ctx.Done():
				svr.Debugf("task stream context cancelled during heartbeat check: %s", taskId.Hex())
				return ctx.Err()
			default:
				// Context is still valid, continue
				svr.Debugf("task stream heartbeat check passed: %s", taskId.Hex())
			}
		}
	}
}

// Connect to task stream when a task runner in a node starts
// Connect handles the bidirectional streaming connection from task runners in nodes.
// It receives messages containing task data and logs, processes them, and handles any errors.
func (svr TaskServiceServer) Connect(stream grpc.TaskService_ConnectServer) (err error) {
	// spider id and task id to track which spider/task this connection belongs to
	var spiderId primitive.ObjectID
	var taskId primitive.ObjectID

	// Add timeout protection for the entire connection
	ctx := stream.Context()

	// Log connection start
	svr.Debugf("task connect stream started")

	defer func() {
		if taskId != primitive.NilObjectID {
			svr.Debugf("task connect stream ended for task: %s", taskId.Hex())
		} else {
			svr.Debugf("task connect stream ended")
		}
	}()

	// continuously receive messages from the stream
	for {
		// Check context cancellation before each receive
		select {
		case <-ctx.Done():
			svr.Debugf("task connect stream context cancelled")
			return ctx.Err()
		default:
		}

		// receive next message from stream with timeout
		msg, err := stream.Recv()
		if err == io.EOF {
			// stream has ended normally
			svr.Debugf("task connect stream ended normally (EOF)")
			return nil
		}
		if err != nil {
			// handle graceful context cancellation
			if strings.HasSuffix(err.Error(), "context canceled") ||
				strings.Contains(err.Error(), "context deadline exceeded") ||
				strings.Contains(err.Error(), "transport is closing") {
				svr.Debugf("task connect stream cancelled gracefully: %v", err)
				return nil
			}
			// log other stream receive errors
			svr.Errorf("error receiving stream message: %v", err)
			// Return error instead of continuing to prevent infinite error loops
			return err
		}

		// validate and parse the task ID from the message if not already set
		if taskId.IsZero() {
			taskId, err = primitive.ObjectIDFromHex(msg.TaskId)
			if err != nil {
				svr.Errorf("invalid task id: %s", msg.TaskId)
				continue
			}
			svr.Debugf("task connect stream set task id: %s", taskId.Hex())
		}

		// get spider id if not already set
		// this only needs to be done once per connection
		if spiderId.IsZero() {
			t, err := service.NewModelService[models.Task]().GetById(taskId)
			if err != nil {
				svr.Errorf("error getting spider[%s]: %v", taskId.Hex(), err)
				continue
			}
			spiderId = t.SpiderId
		}

		// handle different message types based on the code
		switch msg.Code {
		case grpc.TaskServiceConnectCode_INSERT_DATA:
			// handle scraped data insertion
			err = svr.handleInsertData(taskId, spiderId, msg)
		case grpc.TaskServiceConnectCode_INSERT_LOGS:
			// handle task log insertion
			err = svr.handleInsertLogs(taskId, msg)
		case grpc.TaskServiceConnectCode_PING:
			// handle connection health check ping - no action needed, just acknowledge
			svr.Debugf("received ping from task[%s]", taskId.Hex())
			err = nil
		default:
			// invalid message code received
			svr.Errorf("invalid stream message code: %d", msg.Code)
			continue
		}
		if err != nil {
			// log any errors from handlers
			svr.Errorf("grpc error[%d]: %v", msg.Code, err)
		}
	}
}

// FetchTask tasks to be executed by a task handler
func (svr TaskServiceServer) FetchTask(ctx context.Context, request *grpc.TaskServiceFetchTaskRequest) (response *grpc.TaskServiceFetchTaskResponse, err error) {
	nodeKey := request.GetNodeKey()
	if nodeKey == "" {
		err = fmt.Errorf("invalid node key")
		svr.Errorf("error fetching task: %v", err)
		return nil, err
	}
	n, err := service.NewModelService[models.Node]().GetOne(bson.M{"key": nodeKey}, nil)
	if err != nil {
		svr.Errorf("error getting node[%s]: %v", nodeKey, err)
		return nil, err
	}
	var tid primitive.ObjectID
	opts := &mongo3.FindOptions{
		Sort: bson.D{
			{Key: "priority", Value: 1},
			{Key: "_id", Value: 1},
		},
		Limit: 1,
	}
	if err := mongo3.RunTransactionWithContext(ctx, func(sc mongo2.SessionContext) (err error) {
		// fetch task for the given node
		t, err := service.NewModelService[models.Task]().GetOne(bson.M{
			"node_id": n.Id,
			"status":  constants.TaskStatusPending,
		}, opts)
		if err == nil {
			tid = t.Id
			t.Status = constants.TaskStatusAssigned
			return svr.saveTask(t)
		} else if !errors.Is(err, mongo2.ErrNoDocuments) {
			svr.Errorf("error fetching task for node[%s]: %v", nodeKey, err)
			return err
		}

		// fetch task for any node
		t, err = service.NewModelService[models.Task]().GetOne(bson.M{
			"node_id": primitive.NilObjectID,
			"status":  constants.TaskStatusPending,
		}, opts)
		if err == nil {
			tid = t.Id
			t.NodeId = n.Id
			t.Status = constants.TaskStatusAssigned
			return svr.saveTask(t)
		} else if !errors.Is(err, mongo2.ErrNoDocuments) {
			svr.Errorf("error fetching task for any node: %v", err)
			return err
		}

		// no task found
		return nil
	}); err != nil {
		return nil, err
	}

	return &grpc.TaskServiceFetchTaskResponse{TaskId: tid.Hex()}, nil
}

func (svr TaskServiceServer) SendNotification(_ context.Context, request *grpc.TaskServiceSendNotificationRequest) (response *grpc.TaskServiceSendNotificationResponse, err error) {
	if !utils.IsPro() {
		return &grpc.TaskServiceSendNotificationResponse{
			Code:    grpc.TaskServiceSendNotificationResponseCode_NOTIFICATION_DISABLED,
			Message: "Notification service is disabled (Pro version required)",
		}, nil
	}

	// task id
	taskId, err := primitive.ObjectIDFromHex(request.TaskId)
	if err != nil {
		svr.Errorf("invalid task id: %s", request.TaskId)
		return nil, err
	}

	// arguments
	var args []any

	// task
	task, err := service.NewModelService[models.Task]().GetById(taskId)
	if err != nil {
		svr.Errorf("error getting task[%s]: %v", request.TaskId, err)
		return nil, err
	}
	args = append(args, task)

	// task stat
	taskStat, err := service.NewModelService[models.TaskStat]().GetById(task.Id)
	if err != nil {
		svr.Errorf("error getting task stat for task[%s]: %v", request.TaskId, err)
		return nil, err
	}
	args = append(args, taskStat)

	// spider
	spider, err := service.NewModelService[models.Spider]().GetById(task.SpiderId)
	if err != nil {
		svr.Errorf("error getting spider[%s]: %v", task.SpiderId.Hex(), err)
		return nil, err
	}
	args = append(args, spider)

	// node
	node, err := service.NewModelService[models.Node]().GetById(task.NodeId)
	if err != nil {
		svr.Errorf("error getting node[%s]: %v", task.NodeId.Hex(), err)
		return nil, err
	}
	args = append(args, node)

	// schedule
	var schedule *models.Schedule
	if !task.ScheduleId.IsZero() {
		schedule, err = service.NewModelService[models.Schedule]().GetById(task.ScheduleId)
		if err != nil {
			svr.Errorf("error getting schedule[%s]: %v", task.ScheduleId.Hex(), err)
			return nil, err
		}
		args = append(args, schedule)
	}

	// settings
	settings, err := service.NewModelService[models.NotificationSetting]().GetMany(bson.M{
		"enabled": true,
		"trigger": bson.M{
			"$regex": constants.NotificationTriggerPatternTask,
		},
	}, nil)
	if err != nil {
		svr.Errorf("error getting notification settings: %v", err)
		return nil, err
	}

	// notification service
	svc := notification.GetNotificationService()

	for _, s := range settings {
		// compatible with old settings
		trigger := s.Trigger
		if trigger == "" {
			trigger = s.TaskTrigger
		}

		// send notification
		switch trigger {
		case constants.NotificationTriggerTaskFinish:
			if task.Status != constants.TaskStatusPending && task.Status != constants.TaskStatusRunning {
				go svc.Send(&s, args...)
			}
		case constants.NotificationTriggerTaskError:
			if task.Status == constants.TaskStatusError || task.Status == constants.TaskStatusAbnormal {
				go svc.Send(&s, args...)
			}
		case constants.NotificationTriggerTaskEmptyResults:
			if task.Status != constants.TaskStatusPending && task.Status != constants.TaskStatusRunning {
				if taskStat.ResultCount == 0 {
					go svc.Send(&s, args...)
				}
			}
		}
	}

	return &grpc.TaskServiceSendNotificationResponse{
		Code:    grpc.TaskServiceSendNotificationResponseCode_NOTIFICATION_SUCCESS,
		Message: "Notification sent successfully",
	}, nil
}

func (svr TaskServiceServer) CheckProcess(_ context.Context, request *grpc.TaskServiceCheckProcessRequest) (response *grpc.TaskServiceCheckProcessResponse, err error) {
	// Validate request
	_, err = primitive.ObjectIDFromHex(request.TaskId)
	if err != nil {
		svr.Errorf("invalid task id: %s", request.TaskId)
		return &grpc.TaskServiceCheckProcessResponse{
			TaskId:       request.TaskId,
			Pid:          request.Pid,
			Status:       grpc.ProcessStatus_PROCESS_UNKNOWN,
			ErrorMessage: "invalid task id",
		}, nil
	}

	pid := int(request.Pid)
	if pid <= 0 {
		return &grpc.TaskServiceCheckProcessResponse{
			TaskId:       request.TaskId,
			Pid:          request.Pid,
			Status:       grpc.ProcessStatus_PROCESS_NOT_FOUND,
			ErrorMessage: "invalid process id",
		}, nil
	}

	// Check if process exists
	processExists := utils.ProcessIdExists(pid)
	if !processExists {
		return &grpc.TaskServiceCheckProcessResponse{
			TaskId:   request.TaskId,
			Pid:      request.Pid,
			Status:   grpc.ProcessStatus_PROCESS_NOT_FOUND,
			ExitCode: -1,
		}, nil
	}

	// Get process details using gopsutil
	processStatus, exitCode, errMsg := svr.getProcessDetails(pid)

	return &grpc.TaskServiceCheckProcessResponse{
		TaskId:       request.TaskId,
		Pid:          request.Pid,
		Status:       processStatus,
		ExitCode:     int32(exitCode),
		ErrorMessage: errMsg,
	}, nil
}

// getProcessDetails queries the process details using gopsutil
func (svr TaskServiceServer) getProcessDetails(pid int) (status grpc.ProcessStatus, exitCode int, errorMessage string) {
	// Import the gopsutil process package
	processLib, err := utils.GetProcesses()
	if err != nil {
		return grpc.ProcessStatus_PROCESS_UNKNOWN, -1, fmt.Sprintf("failed to get processes: %v", err)
	}

	// Find the specific process
	for _, p := range processLib {
		if int(p.Pid) == pid {
			// Get process status
			processStatus, err := p.Status()
			if err != nil {
				return grpc.ProcessStatus_PROCESS_UNKNOWN, -1, fmt.Sprintf("failed to get process status: %v", err)
			}

			// Map process status to our enum
			switch strings.ToLower(processStatus) {
			case "running", "sleep", "disk-sleep":
				return grpc.ProcessStatus_PROCESS_RUNNING, 0, ""
			case "zombie":
				return grpc.ProcessStatus_PROCESS_ZOMBIE, 0, "process is zombie"
			case "stopped", "tracing-stop":
				return grpc.ProcessStatus_PROCESS_FINISHED, 0, "process stopped"
			default:
				return grpc.ProcessStatus_PROCESS_UNKNOWN, -1, fmt.Sprintf("unknown process status: %s", processStatus)
			}
		}
	}

	// Process not found
	return grpc.ProcessStatus_PROCESS_NOT_FOUND, -1, "process not found"
}

func (svr TaskServiceServer) GetSubscribeStream(taskId primitive.ObjectID) (stream grpc.TaskService_SubscribeServer, ok bool) {
	taskServiceMutex.Lock()
	defer taskServiceMutex.Unlock()
	stream, ok = svr.subs[taskId]
	return stream, ok
}

// cleanupStaleStreams periodically checks for and removes stale streams
func (svr TaskServiceServer) cleanupStaleStreams() {
	ticker := time.NewTicker(10 * time.Minute) // Check every 10 minutes
	defer ticker.Stop()

	for {
		select {
		case <-svr.cleanupCtx.Done():
			svr.Debugf("stream cleanup routine shutting down")
			return
		case <-ticker.C:
			svr.performStreamCleanup()
		}
	}
}

// performStreamCleanup checks each stream and removes those that are no longer active
func (svr TaskServiceServer) performStreamCleanup() {
	taskServiceMutex.Lock()
	defer taskServiceMutex.Unlock()

	var staleTaskIds []primitive.ObjectID

	for taskId, stream := range svr.subs {
		// Check if stream context is still active
		select {
		case <-stream.Context().Done():
			// Stream is done, mark for removal
			staleTaskIds = append(staleTaskIds, taskId)
		default:
			// Stream is still active, continue
		}
	}

	// Remove stale streams
	for _, taskId := range staleTaskIds {
		delete(svr.subs, taskId)
		svr.Infof("cleaned up stale stream for task: %s", taskId.Hex())
	}

	if len(staleTaskIds) > 0 {
		svr.Infof("cleaned up %d stale streams", len(staleTaskIds))
	}
}

func (svr TaskServiceServer) handleInsertData(taskId, spiderId primitive.ObjectID, msg *grpc.TaskServiceConnectRequest) (err error) {
	var records []map[string]interface{}
	err = json.Unmarshal(msg.Data, &records)
	if err != nil {
		svr.Errorf("error unmarshalling data: %v", err)
		return err
	}
	for i := range records {
		records[i][constants.TaskKey] = taskId
		records[i][constants.SpiderKey] = spiderId
	}
	return svr.statsSvc.InsertData(taskId, records...)
}

func (svr TaskServiceServer) handleInsertLogs(taskId primitive.ObjectID, msg *grpc.TaskServiceConnectRequest) (err error) {
	var logs []string
	err = json.Unmarshal(msg.Data, &logs)
	if err != nil {
		svr.Errorf("error unmarshalling logs: %v", err)
		return err
	}
	return svr.statsSvc.InsertLogs(taskId, logs...)
}

func (svr TaskServiceServer) saveTask(t *models.Task) (err error) {
	t.SetUpdated(t.CreatedBy)
	return service.NewModelService[models.Task]().ReplaceById(t.Id, *t)
}

func newTaskServiceServer() *TaskServiceServer {
	ctx, cancel := context.WithCancel(context.Background())

	server := &TaskServiceServer{
		cfgSvc:        nodeconfig.GetNodeConfigService(),
		subs:          make(map[primitive.ObjectID]grpc.TaskService_SubscribeServer),
		statsSvc:      stats.GetTaskStatsService(),
		cleanupCtx:    ctx,
		cleanupCancel: cancel,
		Logger:        utils.NewLogger("GrpcTaskServiceServer"),
	}

	// Start the cleanup routine
	go server.cleanupStaleStreams()

	return server
}

// Stop gracefully shuts down the task service server
func (svr TaskServiceServer) Stop() error {
	svr.Infof("stopping task service server...")

	// Cancel cleanup routine
	if svr.cleanupCancel != nil {
		svr.cleanupCancel()
	}

	// Clean up all remaining streams
	taskServiceMutex.Lock()
	streamCount := len(svr.subs)
	for taskId := range svr.subs {
		delete(svr.subs, taskId)
	}
	taskServiceMutex.Unlock()

	if streamCount > 0 {
		svr.Infof("cleaned up %d remaining streams on shutdown", streamCount)
	}

	svr.Infof("task service server stopped")
	return nil
}

// isTaskFinished checks if a task has completed execution
func (svr TaskServiceServer) isTaskFinished(taskId primitive.ObjectID) bool {
	task, err := service.NewModelService[models.Task]().GetById(taskId)
	if err != nil {
		svr.Debugf("error checking task[%s] status: %v", taskId.Hex(), err)
		return false
	}

	// Task is finished if it's not in pending or running state
	return task.Status != constants.TaskStatusPending && task.Status != constants.TaskStatusRunning
}

var _taskServiceServer *TaskServiceServer
var _taskServiceServerOnce sync.Once

func GetTaskServiceServer() *TaskServiceServer {
	_taskServiceServerOnce.Do(func() {
		_taskServiceServer = newTaskServiceServer()
	})
	return _taskServiceServer
}
