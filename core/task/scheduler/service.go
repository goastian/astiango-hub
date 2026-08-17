package scheduler

import (
	"context"
	errors2 "errors"
	"fmt"
	"sync"
	"time"

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
	mongo2 "go.mongodb.org/mongo-driver/mongo"
)

type Service struct {
	// dependencies
	svr        *server.GrpcServer
	handlerSvc *handler.Service

	// settings
	interval time.Duration

	// internals for lifecycle management
	ctx     context.Context
	cancel  context.CancelFunc
	stopped bool
	mu      sync.RWMutex
	wg      sync.WaitGroup

	interfaces.Logger
}

func (svc *Service) Start() {
	// Initialize context for graceful shutdown
	svc.ctx, svc.cancel = context.WithCancel(context.Background())

	// Start background goroutines with proper tracking
	svc.wg.Add(2)
	go svc.initTaskStatus()
	go svc.cleanupTasks()

	svc.Infof("Task scheduler service started")
	utils.DefaultWait()
}

func (svc *Service) Stop() {
	svc.mu.Lock()
	if svc.stopped {
		svc.mu.Unlock()
		return
	}
	svc.stopped = true
	svc.mu.Unlock()

	svc.Infof("Stopping task scheduler service...")

	// Cancel context to signal all goroutines to stop
	if svc.cancel != nil {
		svc.cancel()
	}

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

	svc.Infof("Task scheduler service stopped")
}

func (svc *Service) Enqueue(t *models.Task, by primitive.ObjectID) (t2 *models.Task, err error) {
	// set task status
	t.Status = constants.TaskStatusPending
	t.SetCreated(by)
	t.SetUpdated(by)

	// add task
	taskModelSvc := service.NewModelService[models.Task]()
	t.Id, err = taskModelSvc.InsertOne(*t)
	if err != nil {
		return nil, err
	}

	// task stat
	ts := models.TaskStat{}
	ts.SetId(t.Id)
	ts.SetCreated(by)
	ts.SetUpdated(by)

	// add task stat
	_, err = service.NewModelService[models.TaskStat]().InsertOne(ts)
	if err != nil {
		svc.Errorf("failed to add task stat: %s", t.Id.Hex())
		return nil, err
	}

	// success
	return t, nil
}

func (svc *Service) Cancel(id, by primitive.ObjectID, force bool) (err error) {
	// task
	t, err := service.NewModelService[models.Task]().GetById(id)
	if err != nil {
		svc.Errorf("task not found: %s", id.Hex())
		return err
	}

	// initial status
	initialStatus := t.Status

	// set status of pending tasks as "cancelled"
	if initialStatus == constants.TaskStatusPending {
		t.Status = constants.TaskStatusCancelled
		return svc.SaveTask(t, by)
	}

	// whether task is running on master node
	isMasterTask, err := svc.isMasterNode(t)
	if err != nil {
		err := fmt.Errorf("failed to check if task is running on master node: %s", t.Id.Hex())
		t.Status = constants.TaskStatusAbnormal
		t.Error = err.Error()
		return svc.SaveTask(t, by)
	}

	if isMasterTask {
		// cancel task on master
		return svc.cancelOnMaster(t, by, force)
	} else {
		// send to cancel task on worker nodes
		return svc.cancelOnWorker(t, by, force)
	}
}

func (svc *Service) cancelOnMaster(t *models.Task, by primitive.ObjectID, force bool) (err error) {
	if err := svc.handlerSvc.Cancel(t.Id, force); err != nil {
		svc.Errorf("failed to cancel task (%s) on master: %v", t.Id.Hex(), err)
		return err
	}

	// set task status as "cancelled"
	t.Status = constants.TaskStatusCancelled
	return svc.SaveTask(t, by)
}

func (svc *Service) cancelOnWorker(t *models.Task, by primitive.ObjectID, force bool) (err error) {
	// get subscribe stream
	stream, ok := svc.svr.TaskSvr.GetSubscribeStream(t.Id)
	if !ok {
		svc.Warnf("stream not found for task (%s), task may already be finished or connection lost", t.Id.Hex())
		// Task might have finished or connection lost, mark as cancelled anyway
		t.Status = constants.TaskStatusCancelled
		t.Error = "cancel signal could not be delivered - stream not found"
		return svc.SaveTask(t, by)
	}

	// send cancel request with timeout - use service context
	ctx, cancel := context.WithTimeout(svc.ctx, 30*time.Second)
	defer cancel()

	// Create a channel to handle the send operation
	sendDone := make(chan error, 1)
	go func() {
		err := stream.Send(&grpc.TaskServiceSubscribeResponse{
			Code:   grpc.TaskServiceSubscribeCode_CANCEL,
			TaskId: t.Id.Hex(),
			Force:  force,
		})
		sendDone <- err
	}()

	select {
	case err = <-sendDone:
		if err != nil {
			svc.Errorf("failed to send cancel task (%s) request to worker: %v", t.Id.Hex(), err)
			// If sending failed, still mark task as cancelled to avoid stuck state
			t.Status = constants.TaskStatusCancelled
			t.Error = fmt.Sprintf("cancel signal delivery failed: %v", err)
			return svc.SaveTask(t, by)
		}
		svc.Infof("cancel signal sent for task (%s) with force=%v", t.Id.Hex(), force)
	case <-ctx.Done():
		svc.Errorf("timeout sending cancel request for task (%s)", t.Id.Hex())
		// Mark as cancelled even if signal couldn't be delivered
		t.Status = constants.TaskStatusCancelled
		t.Error = "cancel signal delivery timeout"
		return svc.SaveTask(t, by)
	}

	// For force cancellation, wait a bit and verify cancellation
	if force {
		time.Sleep(5 * time.Second)
		// Re-fetch task to check current status
		currentTask, fetchErr := service.NewModelService[models.Task]().GetById(t.Id)
		if fetchErr == nil && currentTask.Status == constants.TaskStatusRunning {
			svc.Warnf("task (%s) still running after force cancel, marking as cancelled", t.Id.Hex())
			currentTask.Status = constants.TaskStatusCancelled
			currentTask.Error = "forced cancellation - task was unresponsive"
			return svc.SaveTask(currentTask, by)
		}
	}

	return nil
}

func (svc *Service) SetInterval(interval time.Duration) {
	svc.interval = interval
}

func (svc *Service) SaveTask(t *models.Task, by primitive.ObjectID) (err error) {
	if t.Id.IsZero() {
		t.SetCreated(by)
		t.SetUpdated(by)
		_, err = service.NewModelService[models.Task]().InsertOne(*t)
		return err
	} else {
		t.SetUpdated(by)
		return service.NewModelService[models.Task]().ReplaceById(t.Id, *t)
	}
}

// initTaskStatus initialize task status of existing tasks
func (svc *Service) initTaskStatus() {
	defer svc.wg.Done()
	defer func() {
		if r := recover(); r != nil {
			svc.Errorf("initTaskStatus panic recovered: %v", r)
		}
	}()

	// set status of running tasks as TaskStatusAbnormal
	runningTasks, err := service.NewModelService[models.Task]().GetMany(bson.M{
		"status": bson.M{
			"$in": []string{
				constants.TaskStatusPending,
				constants.TaskStatusAssigned,
				constants.TaskStatusRunning,
			},
		},
	}, nil)
	if err != nil {
		if errors2.Is(err, mongo2.ErrNoDocuments) {
			return
		}
		svc.Errorf("failed to get running tasks: %v", err)
		return
	}

	// Use bounded worker pool for task updates
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, 10) // Limit concurrent updates

	for _, t := range runningTasks {
		select {
		case <-svc.ctx.Done():
			svc.Infof("initTaskStatus stopped by context")
			return
		case semaphore <- struct{}{}:
			wg.Add(1)
			go func(task models.Task) {
				defer func() {
					<-semaphore
					wg.Done()
					if r := recover(); r != nil {
						svc.Errorf("task status update panic for task %s: %v", task.Id.Hex(), r)
					}
				}()

				task.Status = constants.TaskStatusAbnormal
				if err := svc.SaveTask(&task, primitive.NilObjectID); err != nil {
					svc.Errorf("failed to set task status as TaskStatusAbnormal: %s", task.Id.Hex())
				}
			}(t)
		}
	}

	wg.Wait()
	svc.Infof("initTaskStatus completed")
}

func (svc *Service) isMasterNode(t *models.Task) (ok bool, err error) {
	if t.NodeId.IsZero() {
		err = fmt.Errorf("task %s has no node id", t.Id.Hex())
		svc.Errorf("%v", err)
		return false, err
	}
	n, err := service.NewModelService[models.Node]().GetById(t.NodeId)
	if err != nil {
		if errors2.Is(err, mongo2.ErrNoDocuments) {
			svc.Errorf("node not found: %s", t.NodeId.Hex())
			return false, err
		}
		svc.Errorf("failed to get node: %s", t.NodeId.Hex())
		return false, err
	}
	return n.IsMaster, nil
}

func (svc *Service) cleanupTasks() {
	defer svc.wg.Done()
	defer func() {
		if r := recover(); r != nil {
			svc.Errorf("cleanupTasks panic recovered: %v", r)
		}
	}()

	ticker := time.NewTicker(30 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-svc.ctx.Done():
			svc.Infof("cleanupTasks stopped by context")
			return
		case <-ticker.C:
			svc.performCleanup()
		}
	}
}

func (svc *Service) performCleanup() {
	defer func() {
		if r := recover(); r != nil {
			svc.Errorf("performCleanup panic recovered: %v", r)
		}
	}()

	// task stats over 30 days ago
	taskStats, err := service.NewModelService[models.TaskStat]().GetMany(bson.M{
		"created_at": bson.M{
			"$lt": time.Now().Add(-30 * 24 * time.Hour),
		},
	}, nil)
	if err != nil {
		svc.Errorf("failed to get old task stats: %v", err)
		return
	}

	// task ids
	var ids []primitive.ObjectID
	for _, ts := range taskStats {
		ids = append(ids, ts.Id)
	}

	if len(ids) > 0 {
		// remove tasks
		if err := service.NewModelService[models.Task]().DeleteMany(bson.M{
			"_id": bson.M{"$in": ids},
		}); err != nil {
			svc.Warnf("failed to remove tasks: %v", err)
		}

		// remove task stats
		if err := service.NewModelService[models.TaskStat]().DeleteMany(bson.M{
			"_id": bson.M{"$in": ids},
		}); err != nil {
			svc.Warnf("failed to remove task stats: %v", err)
		}

		svc.Infof("cleaned up %d old tasks", len(ids))
	}
}

func newTaskSchedulerService() *Service {
	return &Service{
		interval:   5 * time.Second,
		svr:        server.GetGrpcServer(),
		handlerSvc: handler.GetTaskHandlerService(),
		Logger:     utils.NewLogger("TaskScheduler"),
	}
}

var _service *Service
var _serviceOnce sync.Once

func GetTaskSchedulerService() *Service {
	_serviceOnce.Do(func() {
		_service = newTaskSchedulerService()
	})
	return _service
}
