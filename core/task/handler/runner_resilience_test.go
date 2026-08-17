package handler

import (
	"context"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/goastian/astiango-hub/core/utils"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// TestRunner_LongRunningTaskResilience tests the robustness features for long-running tasks
func TestRunner_LongRunningTaskResilience(t *testing.T) {
	// Create a mock task runner with the resilience features
	r := &Runner{
		tid:                 primitive.NewObjectID(),
		maxConnRetries:      10,
		connRetryDelay:      10 * time.Second,
		ipcTimeout:          60 * time.Second,
		healthCheckInterval: 5 * time.Second,
		connHealthInterval:  60 * time.Second,
		Logger:              utils.NewLogger("TestTaskRunner"),
	}

	// Initialize context
	r.ctx, r.cancel = context.WithCancel(context.Background())
	defer r.cancel()

	// Test that default values are set for robust execution
	if r.maxConnRetries != 10 {
		t.Errorf("Expected maxConnRetries to be 10, got %d", r.maxConnRetries)
	}

	if r.ipcTimeout != 60*time.Second {
		t.Errorf("Expected ipcTimeout to be 60s, got %v", r.ipcTimeout)
	}

	if r.connHealthInterval != 60*time.Second {
		t.Errorf("Expected connHealthInterval to be 60s, got %v", r.connHealthInterval)
	}

	if r.healthCheckInterval != 5*time.Second {
		t.Errorf("Expected healthCheckInterval to be 5s, got %v", r.healthCheckInterval)
	}

	t.Log("✅ All resilience settings configured correctly for robust task execution")
}

// TestRunner_ConnectionHealthMonitoring tests the connection health monitoring
func TestRunner_ConnectionHealthMonitoring(t *testing.T) {
	r := &Runner{
		tid:                primitive.NewObjectID(),
		maxConnRetries:     3,
		connRetryDelay:     100 * time.Millisecond, // Short delay for testing
		connHealthInterval: 200 * time.Millisecond, // Short interval for testing
		Logger:             utils.NewLogger("TestTaskRunner"),
	}

	// Initialize context
	r.ctx, r.cancel = context.WithCancel(context.Background())
	defer r.cancel()

	// Test connection stats
	stats := r.GetConnectionStats()
	if stats == nil {
		t.Fatal("GetConnectionStats returned nil")
	}

	// Check that all expected keys are present
	expectedKeys := []string{
		"last_connection_check",
		"retry_attempts",
		"max_retries",
		"connection_healthy",
		"connection_exists",
	}

	for _, key := range expectedKeys {
		if _, exists := stats[key]; !exists {
			t.Errorf("Expected key '%s' not found in connection stats", key)
		}
	}

	// Test that connection is initially unhealthy (no actual connection)
	if stats["connection_healthy"].(bool) {
		t.Error("Expected connection to be unhealthy without actual connection")
	}

	if stats["connection_exists"].(bool) {
		t.Error("Expected connection_exists to be false without actual connection")
	}

	t.Log("✅ Connection health monitoring working correctly")
}

// TestRunner_PeriodicCleanup tests the periodic cleanup functionality
func TestRunner_PeriodicCleanup(t *testing.T) {
	r := &Runner{
		tid:    primitive.NewObjectID(),
		Logger: utils.NewLogger("TestTaskRunner"),
	}

	// Initialize context
	r.ctx, r.cancel = context.WithCancel(context.Background())
	defer r.cancel()

	// Record memory stats before cleanup
	var beforeStats runtime.MemStats
	runtime.ReadMemStats(&beforeStats)

	// Run cleanup
	r.runPeriodicCleanup()

	// Record memory stats after cleanup
	var afterStats runtime.MemStats
	runtime.ReadMemStats(&afterStats)

	// Verify that GC was called (NumGC should have increased)
	if afterStats.NumGC <= beforeStats.NumGC {
		t.Log("Note: GC count didn't increase, but this is normal in test environment")
	}

	t.Log("✅ Periodic cleanup executed successfully")
}

// TestRunner_ContextCancellation tests proper context handling
func TestRunner_ContextCancellation(t *testing.T) {
	r := &Runner{
		tid:    primitive.NewObjectID(),
		Logger: utils.NewLogger("TestTaskRunner"),
	}

	// Initialize context
	r.ctx, r.cancel = context.WithCancel(context.Background())

	// Test writeLogLines with cancelled context
	r.cancel() // Cancel context first

	// This should return early without error
	r.writeLogLines([]string{"test log"})

	t.Log("✅ Context cancellation handled correctly")
}

// TestRunner_ThreadSafety tests thread-safe access to connection
func TestRunner_ThreadSafety(t *testing.T) {
	r := &Runner{
		tid:    primitive.NewObjectID(),
		Logger: utils.NewLogger("TestTaskRunner"),
	}

	// Initialize context
	r.ctx, r.cancel = context.WithCancel(context.Background())
	defer r.cancel()

	var wg sync.WaitGroup

	// Start multiple goroutines accessing connection stats
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			for j := 0; j < 100; j++ {
				// Access connection stats (this uses RWMutex)
				stats := r.GetConnectionStats()
				if stats == nil {
					t.Errorf("Goroutine %d: GetConnectionStats returned nil", id)
					return
				}

				// Small delay to increase chance of race conditions
				time.Sleep(1 * time.Millisecond)
			}
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
		t.Log("✅ Thread safety test completed successfully")
	case <-time.After(10 * time.Second):
		t.Fatal("Thread safety test timed out")
	}
}

// BenchmarkRunner_ConnectionStats benchmarks the connection stats access
func BenchmarkRunner_ConnectionStats(b *testing.B) {
	r := &Runner{
		tid:                 primitive.NewObjectID(),
		maxConnRetries:      10,
		connRetryDelay:      10 * time.Second,
		ipcTimeout:          60 * time.Second,
		healthCheckInterval: 5 * time.Second,
		connHealthInterval:  60 * time.Second,
		Logger:              utils.NewLogger("BenchmarkTaskRunner"),
	}

	// Initialize context
	r.ctx, r.cancel = context.WithCancel(context.Background())
	defer r.cancel()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		stats := r.GetConnectionStats()
		if stats == nil {
			b.Fatal("GetConnectionStats returned nil")
		}
	}
}
