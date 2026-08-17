package server

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/goastian/astiango-hub/core/utils"
	"github.com/goastian/astiango-hub/grpc"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"google.golang.org/grpc/metadata"
)

// Mock stream for testing
type mockSubscribeStream struct {
	ctx    context.Context
	cancel context.CancelFunc
}

func (m *mockSubscribeStream) Context() context.Context {
	return m.ctx
}

func (m *mockSubscribeStream) Send(*grpc.TaskServiceSubscribeResponse) error {
	return nil
}

func (m *mockSubscribeStream) SetHeader(metadata.MD) error  { return nil }
func (m *mockSubscribeStream) SendHeader(metadata.MD) error { return nil }
func (m *mockSubscribeStream) SetTrailer(metadata.MD)       {}
func (m *mockSubscribeStream) RecvMsg(interface{}) error    { return nil }
func (m *mockSubscribeStream) SendMsg(interface{}) error    { return nil }

func newMockSubscribeStream() *mockSubscribeStream {
	ctx, cancel := context.WithCancel(context.Background())
	return &mockSubscribeStream{
		ctx:    ctx,
		cancel: cancel,
	}
}

func TestTaskServiceServer_Subscribe_Timeout(t *testing.T) {
	server := &TaskServiceServer{
		subs:   make(map[primitive.ObjectID]grpc.TaskService_SubscribeServer),
		Logger: utils.NewLogger("TestTaskServiceServer"),
	}

	taskId := primitive.NewObjectID()
	mockStream := newMockSubscribeStream()

	req := &grpc.TaskServiceSubscribeRequest{
		TaskId: taskId.Hex(),
	}

	// Start subscribe in goroutine
	done := make(chan error, 1)
	go func() {
		err := server.Subscribe(req, mockStream)
		done <- err
	}()

	// Wait a moment for subscription to be added
	time.Sleep(100 * time.Millisecond)

	// Verify stream was added
	taskServiceMutex.Lock()
	_, exists := server.subs[taskId]
	taskServiceMutex.Unlock()

	if !exists {
		t.Fatal("Stream was not added to subscription map")
	}

	// Cancel the mock stream context
	mockStream.cancel()

	// Wait for subscribe to complete
	select {
	case err := <-done:
		if err == nil {
			t.Error("Expected error from cancelled context")
		}
		t.Logf("✅ Subscribe returned with error as expected: %v", err)
	case <-time.After(5 * time.Second):
		t.Fatal("Subscribe didn't return within timeout")
	}

	// Verify stream was cleaned up
	taskServiceMutex.Lock()
	_, exists = server.subs[taskId]
	taskServiceMutex.Unlock()

	if exists {
		t.Error("Stream was not cleaned up from subscription map")
	} else {
		t.Log("✅ Stream properly cleaned up")
	}
}

func TestTaskServiceServer_StreamCleanup(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	server := &TaskServiceServer{
		subs:          make(map[primitive.ObjectID]grpc.TaskService_SubscribeServer),
		cleanupCtx:    ctx,
		cleanupCancel: cancel,
		Logger:        utils.NewLogger("TestTaskServiceServer"),
	}

	// Add some mock streams
	taskId1 := primitive.NewObjectID()
	taskId2 := primitive.NewObjectID()

	mockStream1 := newMockSubscribeStream()
	mockStream2 := newMockSubscribeStream()

	taskServiceMutex.Lock()
	server.subs[taskId1] = mockStream1
	server.subs[taskId2] = mockStream2
	taskServiceMutex.Unlock()

	// Cancel one stream
	mockStream1.cancel()

	// Wait a moment
	time.Sleep(100 * time.Millisecond)

	// Perform cleanup
	server.performStreamCleanup()

	// Verify only the cancelled stream was removed
	taskServiceMutex.Lock()
	_, exists1 := server.subs[taskId1]
	_, exists2 := server.subs[taskId2]
	taskServiceMutex.Unlock()

	if exists1 {
		t.Error("Cancelled stream was not cleaned up")
	} else {
		t.Log("✅ Cancelled stream properly cleaned up")
	}

	if !exists2 {
		t.Error("Active stream was incorrectly removed")
	} else {
		t.Log("✅ Active stream preserved")
	}

	// Clean up remaining
	mockStream2.cancel()
}

func TestTaskServiceServer_Stop(t *testing.T) {
	server := newTaskServiceServer()

	// Add some mock streams
	taskId := primitive.NewObjectID()
	mockStream := newMockSubscribeStream()

	taskServiceMutex.Lock()
	server.subs[taskId] = mockStream
	taskServiceMutex.Unlock()

	// Stop the server
	err := server.Stop()
	if err != nil {
		t.Fatalf("Stop returned error: %v", err)
	}

	// Verify all streams are cleaned up
	taskServiceMutex.Lock()
	streamCount := len(server.subs)
	taskServiceMutex.Unlock()

	if streamCount != 0 {
		t.Errorf("Expected 0 streams after stop, got %d", streamCount)
	} else {
		t.Log("✅ All streams cleaned up on stop")
	}

	// Verify cleanup context is cancelled
	select {
	case <-server.cleanupCtx.Done():
		t.Log("✅ Cleanup context properly cancelled")
	default:
		t.Error("Cleanup context not cancelled")
	}
}

func TestTaskServiceServer_ConcurrentAccess(t *testing.T) {
	server := &TaskServiceServer{
		subs:   make(map[primitive.ObjectID]grpc.TaskService_SubscribeServer),
		Logger: utils.NewLogger("TestTaskServiceServer"),
	}

	var wg sync.WaitGroup

	// Start multiple goroutines adding/removing streams
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			taskId := primitive.NewObjectID()
			mockStream := newMockSubscribeStream()
			defer mockStream.cancel()

			// Add stream
			taskServiceMutex.Lock()
			server.subs[taskId] = mockStream
			taskServiceMutex.Unlock()

			// Do some work
			time.Sleep(10 * time.Millisecond)

			// Remove stream
			taskServiceMutex.Lock()
			delete(server.subs, taskId)
			taskServiceMutex.Unlock()
		}(i)
	}

	// Wait for all goroutines to complete
	done := make(chan bool)
	go func() {
		wg.Wait()
		done <- true
	}()

	select {
	case <-done:
		t.Log("✅ Concurrent access test completed successfully")
	case <-time.After(5 * time.Second):
		t.Fatal("Concurrent access test timed out")
	}
}
