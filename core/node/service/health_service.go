package service

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/goastian/astiango-hub/core/grpc/client"
	"github.com/goastian/astiango-hub/core/grpc/server"
	"github.com/goastian/astiango-hub/core/interfaces"
	nodeconfig "github.com/goastian/astiango-hub/core/node/config"
	"github.com/goastian/astiango-hub/core/utils"
)

type HealthService struct {
	// settings
	healthFilePath string
	updateInterval time.Duration

	// context and synchronization
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
	mu     sync.RWMutex

	// state
	isReady bool
	stopped bool

	// dependencies
	cfgSvc interfaces.NodeConfigService
	interfaces.Logger
}

type HealthCheckFunc func() bool

func (svc *HealthService) Start(customHealthCheck HealthCheckFunc) {
	svc.ctx, svc.cancel = context.WithCancel(context.Background())

	svc.wg.Add(1)
	go func() {
		defer svc.wg.Done()
		svc.writeHealthFile(customHealthCheck)
	}()
}

func (svc *HealthService) Stop() {
	svc.mu.Lock()
	if svc.stopped {
		svc.mu.Unlock()
		return
	}
	svc.stopped = true
	svc.mu.Unlock()

	if svc.cancel != nil {
		svc.cancel()
	}

	// Clean up health file
	if svc.healthFilePath != "" {
		os.Remove(svc.healthFilePath)
	}

	// Wait for goroutines to finish with timeout
	done := make(chan struct{})
	go func() {
		svc.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		svc.Debugf("health service stopped gracefully")
	case <-time.After(5 * time.Second):
		svc.Warnf("health service shutdown timed out")
	}
}

func (svc *HealthService) SetReady(ready bool) {
	svc.mu.Lock()
	svc.isReady = ready
	svc.mu.Unlock()
}

func (svc *HealthService) IsReady() bool {
	svc.mu.RLock()
	defer svc.mu.RUnlock()
	return svc.isReady
}

func (svc *HealthService) writeHealthFile(customHealthCheck HealthCheckFunc) {
	ticker := time.NewTicker(svc.updateInterval)
	defer ticker.Stop()

	for {
		select {
		case <-svc.ctx.Done():
			svc.Debugf("health file writer stopping due to context cancellation")
			return
		case <-ticker.C:
			svc.updateHealthFile(customHealthCheck)
		}
	}
}

func (svc *HealthService) updateHealthFile(customHealthCheck HealthCheckFunc) {
	if svc.healthFilePath == "" {
		return
	}

	svc.mu.RLock()
	isReady := svc.isReady
	stopped := svc.stopped
	svc.mu.RUnlock()

	// Determine node type and health status
	nodeType := utils.GetNodeType()
	isHealthy := isReady && !stopped

	// Add node-type-specific health checks
	if customHealthCheck != nil {
		isHealthy = isHealthy && customHealthCheck()
	} else {
		// Default health checks based on node type
		if utils.IsMaster() {
			// Master node health: check if gRPC server is running
			grpcServer := server.GetGrpcServer()
			isHealthy = isHealthy && grpcServer != nil
		} else {
			// Worker node health: check if gRPC client is connected
			grpcClient := client.GetGrpcClient()
			isHealthy = isHealthy && grpcClient != nil && !grpcClient.IsClosed()
		}
	}

	healthData := fmt.Sprintf(`{
  "healthy": %t,
  "timestamp": "%s",
  "node_type": "%s",
  "node_key": "%s",
  "ready": %t,
  "stopped": %t
}
`, isHealthy, time.Now().Format(time.RFC3339), nodeType, svc.cfgSvc.GetNodeKey(), isReady, stopped)

	// Write to temporary file first, then rename for atomicity
	tmpPath := svc.healthFilePath + ".tmp"
	if err := os.WriteFile(tmpPath, []byte(healthData), 0644); err != nil {
		svc.Errorf("failed to write health file: %v", err)
		return
	}

	if err := os.Rename(tmpPath, svc.healthFilePath); err != nil {
		svc.Errorf("failed to rename health file: %v", err)
		os.Remove(tmpPath) // Clean up temp file
		return
	}
}

func newHealthService() *HealthService {
	return &HealthService{
		healthFilePath: "/tmp/astiango_hub_health",
		updateInterval: 30 * time.Second,
		cfgSvc:         nodeconfig.GetNodeConfigService(),
		Logger:         utils.NewLogger("HealthService"),
	}
}

var healthService *HealthService
var healthServiceOnce sync.Once

func GetHealthService() *HealthService {
	healthServiceOnce.Do(func() {
		healthService = newHealthService()
	})
	return healthService
}
