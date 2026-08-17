package handler

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/goastian/astiango-hub/grpc"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// StreamManager manages task streams without goroutine leaks
type StreamManager struct {
	streams      sync.Map // map[primitive.ObjectID]*TaskStream
	ctx          context.Context
	cancel       context.CancelFunc
	wg           sync.WaitGroup
	service      *Service
	messageQueue chan *StreamMessage
	maxStreams   int
}

// TaskStream represents a single task's stream
type TaskStream struct {
	taskId     primitive.ObjectID
	stream     grpc.TaskService_SubscribeClient
	ctx        context.Context
	cancel     context.CancelFunc
	lastActive time.Time
	mu         sync.RWMutex
}

// StreamMessage represents a message from a stream
type StreamMessage struct {
	taskId primitive.ObjectID
	msg    *grpc.TaskServiceSubscribeResponse
	err    error
}

func NewStreamManager(service *Service) *StreamManager {
	// Use service context for proper cancellation chain
	ctx, cancel := context.WithCancel(service.ctx)
	return &StreamManager{
		ctx:          ctx,
		cancel:       cancel,
		service:      service,
		messageQueue: make(chan *StreamMessage, 100), // Buffered channel for messages
		maxStreams:   50,                             // Limit concurrent streams
	}
}

func (sm *StreamManager) Start() {
	sm.wg.Add(2)
	go sm.messageProcessor()
	go sm.streamCleaner()
}

func (sm *StreamManager) Stop() {
	sm.cancel()
	close(sm.messageQueue)

	// Close all active streams
	sm.streams.Range(func(key, value interface{}) bool {
		if ts, ok := value.(*TaskStream); ok {
			ts.Close()
		}
		return true
	})

	sm.wg.Wait()
}

func (sm *StreamManager) AddTaskStream(taskId primitive.ObjectID) error {
	// Check if stream already exists
	if _, exists := sm.streams.Load(taskId); exists {
		sm.service.Debugf("stream already exists for task[%s], skipping", taskId.Hex())
		return nil
	}

	// Check if we're at the stream limit
	streamCount := sm.getStreamCount()
	if streamCount >= sm.maxStreams {
		sm.service.Warnf("stream limit reached (%d/%d), rejecting new stream for task[%s]",
			streamCount, sm.maxStreams, taskId.Hex())
		return fmt.Errorf("stream limit reached (%d)", sm.maxStreams)
	}

	// Create a context for this specific stream that can be cancelled
	ctx, cancel := context.WithCancel(sm.ctx)

	// Create stream with the cancellable context
	stream, err := sm.service.subscribeTaskWithContext(ctx, taskId)
	if err != nil {
		cancel() // Clean up the context if stream creation fails
		return fmt.Errorf("failed to subscribe to task stream: %v", err)
	}

	taskStream := &TaskStream{
		taskId:     taskId,
		stream:     stream,
		ctx:        ctx,
		cancel:     cancel,
		lastActive: time.Now(),
	}

	sm.streams.Store(taskId, taskStream)
	sm.service.Infof("created stream for task[%s], total streams: %d/%d",
		taskId.Hex(), streamCount+1, sm.maxStreams)

	// Start listening for messages in a single goroutine per stream
	sm.wg.Add(1)
	go sm.streamListener(taskStream)

	return nil
}

func (sm *StreamManager) RemoveTaskStream(taskId primitive.ObjectID) {
	if value, ok := sm.streams.LoadAndDelete(taskId); ok {
		if ts, ok := value.(*TaskStream); ok {
			sm.service.Debugf("stream removed, total streams: %d/%d", sm.getStreamCount(), sm.maxStreams)
			ts.Close()
		}
	}
}

func (sm *StreamManager) getStreamCount() int {
	streamCount := 0
	sm.streams.Range(func(key, value interface{}) bool {
		streamCount++
		return true
	})
	return streamCount
}

func (sm *StreamManager) streamListener(ts *TaskStream) {
	defer sm.wg.Done()
	defer func() {
		if r := recover(); r != nil {
			sm.service.Errorf("stream listener panic for task[%s]: %v", ts.taskId.Hex(), r)
		}
		ts.Close()
		sm.streams.Delete(ts.taskId)
		sm.service.Debugf("stream listener finished cleanup for task[%s]", ts.taskId.Hex())
	}()

	sm.service.Debugf("stream listener started for task[%s]", ts.taskId.Hex())

	for {
		select {
		case <-ts.ctx.Done():
			sm.service.Debugf("stream listener stopped for task[%s]", ts.taskId.Hex())
			return
		case <-sm.ctx.Done():
			sm.service.Debugf("stream manager shutdown, stopping listener for task[%s]", ts.taskId.Hex())
			return
		default:
			// Use a timeout wrapper to handle cases where Recv() might hang
			resultChan := make(chan struct {
				msg *grpc.TaskServiceSubscribeResponse
				err error
			}, 1)

			// Start receive operation in a separate goroutine with proper cleanup
			go func() {
				defer func() {
					if r := recover(); r != nil {
						sm.service.Errorf("stream recv goroutine panic for task[%s]: %v", ts.taskId.Hex(), r)
					}
				}()
				
				msg, err := ts.stream.Recv()
				
				// Use select to ensure we don't block if the main goroutine has exited
				select {
				case resultChan <- struct {
					msg *grpc.TaskServiceSubscribeResponse
					err error
				}{msg, err}:
				case <-ts.ctx.Done():
					// Parent context cancelled, just return without sending
					return
				case <-sm.ctx.Done():
					// Manager context cancelled, just return without sending  
					return
				}
			}()

			// Wait for result, timeout, or cancellation
			select {
			case result := <-resultChan:
				if result.err != nil {
					if errors.Is(result.err, io.EOF) {
						sm.service.Debugf("stream EOF for task[%s] - server closed stream", ts.taskId.Hex())
						return
					}
					if errors.Is(result.err, context.Canceled) || errors.Is(result.err, context.DeadlineExceeded) {
						sm.service.Debugf("stream context cancelled for task[%s]", ts.taskId.Hex())
						return
					}
					sm.service.Debugf("stream error for task[%s]: %v - likely server closed", ts.taskId.Hex(), result.err)
					return
				}

				// Update last active time
				ts.mu.Lock()
				ts.lastActive = time.Now()
				ts.mu.Unlock()

				// Send message to processor (non-blocking)
				select {
				case sm.messageQueue <- &StreamMessage{
					taskId: ts.taskId,
					msg:    result.msg,
					err:    nil,
				}:
				case <-ts.ctx.Done():
					return
				case <-sm.ctx.Done():
					return
				default:
					sm.service.Warnf("message queue full, dropping message for task[%s]", ts.taskId.Hex())
				}

			case <-ts.ctx.Done():
				sm.service.Debugf("stream listener stopped for task[%s]", ts.taskId.Hex())
				return

			case <-sm.ctx.Done():
				sm.service.Debugf("stream manager shutdown, stopping listener for task[%s]", ts.taskId.Hex())
				return
			}
		}
	}
}

func (sm *StreamManager) messageProcessor() {
	defer sm.wg.Done()
	defer func() {
		if r := recover(); r != nil {
			sm.service.Errorf("message processor panic: %v", r)
		}
	}()

	sm.service.Debugf("stream message processor started")

	for {
		select {
		case <-sm.ctx.Done():
			sm.service.Debugf("stream message processor shutting down")
			return
		case msg, ok := <-sm.messageQueue:
			if !ok {
				return
			}
			sm.processMessage(msg)
		}
	}
}

func (sm *StreamManager) processMessage(streamMsg *StreamMessage) {
	if streamMsg.err != nil {
		sm.service.Errorf("stream message error for task[%s]: %v", streamMsg.taskId.Hex(), streamMsg.err)
		return
	}

	// Process the actual message
	sm.service.processStreamMessage(streamMsg.taskId, streamMsg.msg)
}

func (sm *StreamManager) streamCleaner() {
	defer sm.wg.Done()
	defer func() {
		if r := recover(); r != nil {
			sm.service.Errorf("stream cleaner panic: %v", r)
		}
	}()

	ticker := time.NewTicker(2 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-sm.ctx.Done():
			return
		case <-ticker.C:
			sm.cleanupInactiveStreams()
		}
	}
}

func (sm *StreamManager) cleanupInactiveStreams() {
	now := time.Now()
	inactiveThreshold := 10 * time.Minute

	sm.streams.Range(func(key, value interface{}) bool {
		taskId := key.(primitive.ObjectID)
		ts := value.(*TaskStream)

		ts.mu.RLock()
		lastActive := ts.lastActive
		ts.mu.RUnlock()

		if now.Sub(lastActive) > inactiveThreshold {
			sm.service.Debugf("cleaning up inactive stream for task[%s]", taskId.Hex())
			sm.RemoveTaskStream(taskId)
		}

		return true
	})
}

func (ts *TaskStream) Close() {
	// Cancel the context first - this should interrupt any blocking operations
	ts.cancel()

	if ts.stream != nil {
		// Try to close send direction
		err := ts.stream.CloseSend()
		if err != nil {
			fmt.Printf("failed to close stream send for task[%s]: %v\n", ts.taskId.Hex(), err)
		}

		// Note: The stream.Recv() should now fail with context.Canceled
		// due to the cancelled context passed to subscribeTaskWithContext
	}
}
