package handler

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/goastian/astiango-hub/core/constants"
	"github.com/goastian/astiango-hub/core/interfaces"
	"github.com/goastian/astiango-hub/grpc"
	"go.mongodb.org/mongo-driver/bson/primitive"
	grpc2 "google.golang.org/grpc"
)

// Service operations for task management

func (svc *Service) Run(taskId primitive.ObjectID) (err error) {
	return svc.runTask(taskId)
}

func (svc *Service) Cancel(taskId primitive.ObjectID, force bool) (err error) {
	return svc.cancelTask(taskId, force)
}

func (svc *Service) runTask(taskId primitive.ObjectID) (err error) {
	// attempt to get runner from pool
	_, ok := svc.runners.Load(taskId)
	if ok {
		err = fmt.Errorf("task[%s] already exists", taskId.Hex())
		svc.Errorf("run task error: %v", err)
		return err
	}

	// Use worker pool for bounded task execution
	return svc.workerPool.SubmitTask(taskId)
}

// executeTask is the actual task execution logic called by worker pool
func (svc *Service) executeTask(taskId primitive.ObjectID) (err error) {
	// attempt to get runner from pool
	_, ok := svc.runners.Load(taskId)
	if ok {
		err = fmt.Errorf("task[%s] already exists", taskId.Hex())
		svc.Errorf("execute task error: %v", err)
		return err
	}

	// create a new task runner
	r, err := newTaskRunner(taskId, svc)
	if err != nil {
		err = fmt.Errorf("failed to create task runner: %v", err)
		svc.Errorf("execute task error: %v", err)
		return err
	}

	// add runner to pool
	svc.addRunner(taskId, r)

	// Ensure cleanup always happens - CRITICAL for preventing goroutine leaks
	defer func() {
		if rec := recover(); rec != nil {
			svc.Errorf("task[%s] panic recovered: %v", taskId.Hex(), rec)
		}
		// Always cleanup runner from pool and stream
		svc.deleteRunner(taskId)
		svc.streamManager.RemoveTaskStream(taskId)
	}()

	// Add task to stream manager for cancellation support
	if err := svc.streamManager.AddTaskStream(r.GetTaskId()); err != nil {
		svc.Warnf("failed to add task[%s] to stream manager: %v", r.GetTaskId().Hex(), err)
		svc.Warnf("task[%s] will not be able to receive cancellation messages", r.GetTaskId().Hex())
	} else {
		svc.Debugf("task[%s] added to stream manager for cancellation support", r.GetTaskId().Hex())
	}

	// run task process (blocking) error or finish after task runner ends
	if err := r.Run(); err != nil {
		switch {
		case errors.Is(err, constants.ErrTaskError):
			svc.Errorf("task[%s] finished with error: %v", r.GetTaskId().Hex(), err)
		case errors.Is(err, constants.ErrTaskCancelled):
			svc.Infof("task[%s] cancelled", r.GetTaskId().Hex())
		default:
			svc.Errorf("task[%s] finished with unknown error: %v", r.GetTaskId().Hex(), err)
		}
	} else {
		svc.Infof("task[%s] finished successfully", r.GetTaskId().Hex())
	}

	return err
}

// subscribeTaskWithContext attempts to subscribe to task stream with provided context
func (svc *Service) subscribeTaskWithContext(ctx context.Context, taskId primitive.ObjectID) (stream grpc.TaskService_SubscribeClient, err error) {
	req := &grpc.TaskServiceSubscribeRequest{
		TaskId: taskId.Hex(),
	}
	taskClient, err := svc.getGrpcClient().GetTaskClient()
	if err != nil {
		return nil, fmt.Errorf("failed to get task client: %v", err)
	}

	// Use call options to ensure proper cancellation behavior
	opts := []grpc2.CallOption{
		grpc2.WaitForReady(false), // Don't wait for connection if not ready
	}

	stream, err = taskClient.Subscribe(ctx, req, opts...)
	if err != nil {
		svc.Errorf("failed to subscribe task[%s]: %v", taskId.Hex(), err)
		return nil, err
	}
	return stream, nil
}

func (svc *Service) processStreamMessage(taskId primitive.ObjectID, msg *grpc.TaskServiceSubscribeResponse) {
	switch msg.Code {
	case grpc.TaskServiceSubscribeCode_CANCEL:
		svc.Infof("task[%s] received cancel signal", taskId.Hex())
		// Handle cancel synchronously to avoid goroutine accumulation
		svc.handleCancel(msg, taskId)
	default:
		svc.Debugf("task[%s] received unknown stream message code: %v", taskId.Hex(), msg.Code)
	}
}

func (svc *Service) handleCancel(msg *grpc.TaskServiceSubscribeResponse, taskId primitive.ObjectID) {
	// validate task id
	if msg.TaskId != taskId.Hex() {
		svc.Errorf("task[%s] received cancel signal for another task[%s]", taskId.Hex(), msg.TaskId)
		return
	}

	// cancel task
	err := svc.cancelTask(taskId, msg.Force)
	if err != nil {
		svc.Errorf("task[%s] failed to cancel: %v", taskId.Hex(), err)
		return
	}
	svc.Infof("task[%s] cancelled", taskId.Hex())

	// set task status as "cancelled"
	t, err := svc.GetTaskById(taskId)
	if err != nil {
		svc.Errorf("task[%s] failed to get task: %v", taskId.Hex(), err)
		return
	}
	t.Status = constants.TaskStatusCancelled
	err = svc.UpdateTask(t)
	if err != nil {
		svc.Errorf("task[%s] failed to update task: %v", taskId.Hex(), err)
	}
}

func (svc *Service) cancelTask(taskId primitive.ObjectID, force bool) (err error) {
	r, err := svc.getRunner(taskId)
	if err != nil {
		// Runner not found, task might already be finished
		svc.Warnf("runner not found for task[%s]: %v", taskId.Hex(), err)
		return nil
	}

	// Attempt cancellation with timeout - use service context
	cancelCtx, cancelFunc := context.WithTimeout(svc.ctx, 30*time.Second)
	defer cancelFunc()

	cancelDone := make(chan error, 1)
	go func() {
		cancelDone <- r.Cancel(force)
	}()

	select {
	case err = <-cancelDone:
		if err != nil {
			svc.Errorf("failed to cancel task[%s]: %v", taskId.Hex(), err)
			// If cancellation failed and force is not set, try force cancellation
			if !force {
				svc.Warnf("escalating to force cancellation for task[%s]", taskId.Hex())
				return svc.cancelTask(taskId, true)
			}
			return err
		}
		svc.Infof("task[%s] cancelled successfully", taskId.Hex())
	case <-cancelCtx.Done():
		svc.Errorf("timeout cancelling task[%s], removing runner from pool", taskId.Hex())
		// Remove runner from pool to prevent further issues
		svc.runners.Delete(taskId)
		return fmt.Errorf("task cancellation timeout")
	}

	return nil
}

// stopAllRunners gracefully stops all running tasks
func (svc *Service) stopAllRunners() {
	svc.Infof("Stopping all running tasks...")

	var runnerIds []primitive.ObjectID

	// Collect all runner IDs
	svc.runners.Range(func(key, value interface{}) bool {
		if taskId, ok := key.(primitive.ObjectID); ok {
			runnerIds = append(runnerIds, taskId)
		}
		return true
	})

	// Cancel all runners with bounded concurrency to prevent goroutine explosion
	const maxConcurrentCancellations = 10
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, maxConcurrentCancellations)

	for _, taskId := range runnerIds {
		wg.Add(1)

		// Acquire semaphore to limit concurrent cancellations
		semaphore <- struct{}{}

		go func(tid primitive.ObjectID) {
			defer func() {
				<-semaphore // Release semaphore
				wg.Done()
				if r := recover(); r != nil {
					svc.Errorf("stopAllRunners panic for task[%s]: %v", tid.Hex(), r)
				}
			}()

			if err := svc.cancelTask(tid, false); err != nil {
				svc.Errorf("failed to cancel task[%s]: %v", tid.Hex(), err)
				// Force cancel after timeout
				time.Sleep(5 * time.Second)
				_ = svc.cancelTask(tid, true)
			}
		}(taskId)
	}

	// Wait for all cancellations with timeout
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		svc.Infof("All tasks stopped gracefully")
	case <-time.After(30 * time.Second):
		svc.Warnf("Some tasks did not stop within timeout")
	}
}

func (svc *Service) getRunner(taskId primitive.ObjectID) (r interfaces.TaskRunner, err error) {
	svc.Debugf("get runner: taskId[%v]", taskId)
	v, ok := svc.runners.Load(taskId)
	if !ok {
		err = fmt.Errorf("task[%s] not exists", taskId.Hex())
		svc.Errorf("get runner error: %v", err)
		return nil, err
	}
	switch v := v.(type) {
	case interfaces.TaskRunner:
		r = v
	default:
		err = fmt.Errorf("invalid type: %T", v)
		svc.Errorf("get runner error: %v", err)
		return nil, err
	}
	return r, nil
}

func (svc *Service) addRunner(taskId primitive.ObjectID, r interfaces.TaskRunner) {
	svc.Debugf("add runner: taskId[%s]", taskId.Hex())
	svc.runners.Store(taskId, r)
}

func (svc *Service) deleteRunner(taskId primitive.ObjectID) {
	svc.Debugf("delete runner: taskId[%v]", taskId)
	svc.runners.Delete(taskId)
}

// GetTaskRunner returns the task runner for the given task ID (public method for external access)
func (svc *Service) GetTaskRunner(taskId primitive.ObjectID) interfaces.TaskRunner {
	r, err := svc.getRunner(taskId)
	if err != nil {
		return nil
	}
	return r
}
