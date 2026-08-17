package client

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/goastian/astiango-hub/core/grpc/middlewares"
	"github.com/goastian/astiango-hub/core/interfaces"
	"github.com/goastian/astiango-hub/core/utils"
	grpc2 "github.com/goastian/astiango-hub/grpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/keepalive"
)

// Circuit breaker constants
const (
	maxFailures                  = 5
	cbResetTime                  = 2 * time.Minute
	cbHalfOpenRetryInterval      = 30 * time.Second
	healthCheckInterval          = 2 * time.Minute // Reduced frequency from 30 seconds
	stateMonitorInterval         = 5 * time.Second
	registrationCheckInterval    = 100 * time.Millisecond
	idleGracePeriod              = 2 * time.Minute // Increased from 30 seconds
	connectionTimeout            = 30 * time.Second
	defaultClientTimeout         = 15 * time.Second // Increased from 5s for better reconnection handling
	reconnectionClientTimeout    = 90 * time.Second // Extended timeout during reconnection scenarios (must be > worker reset timeout)
	connectionStabilizationDelay = 2 * time.Second  // Wait after reconnection before declaring success
	maxReconnectionWait          = 30 * time.Second // Maximum time to wait for full reconnection completion
	reconnectionCheckInterval    = 500 * time.Millisecond
)

// Circuit breaker states
type circuitBreakerState int

const (
	cbClosed circuitBreakerState = iota
	cbOpen
	cbHalfOpen
)

// min function for calculating backoff
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// RetryWithBackoff retries an operation up to maxAttempts times with exponential backoff.
// It detects "reconnection in progress" errors and retries appropriately.
// Returns the last error if all attempts fail, or nil on success.
func RetryWithBackoff(ctx context.Context, operation func() error, maxAttempts int, logger interfaces.Logger, operationName string) error {
	var lastErr error
	for attempt := 0; attempt < maxAttempts; attempt++ {
		if attempt > 0 {
			// Exponential backoff: 1s, 2s, 4s, ...
			backoffDelay := time.Duration(1<<uint(attempt-1)) * time.Second
			if logger != nil {
				logger.Debugf("retrying %s after %v (attempt %d/%d)", operationName, backoffDelay, attempt+1, maxAttempts)
			}

			select {
			case <-ctx.Done():
				if logger != nil {
					logger.Debugf("%s retry cancelled due to context", operationName)
				}
				return ctx.Err()
			case <-time.After(backoffDelay):
			}
		}

		err := operation()
		if err == nil {
			if attempt > 0 && logger != nil {
				logger.Infof("%s succeeded after %d attempts", operationName, attempt+1)
			}
			return nil
		}

		lastErr = err
		// Check if error indicates reconnection in progress
		if strings.Contains(err.Error(), "reconnection in progress") {
			if logger != nil {
				logger.Debugf("%s waiting for reconnection (attempt %d/%d): %v", operationName, attempt+1, maxAttempts, err)
			}
			continue
		}

		if logger != nil {
			logger.Debugf("%s failed (attempt %d/%d): %v", operationName, attempt+1, maxAttempts, err)
		}
	}

	return fmt.Errorf("%s failed after %d attempts: %v", operationName, maxAttempts, lastErr)
}

// GrpcClient provides a robust gRPC client with connection management and client registration.
//
// The client handles connection lifecycle and ensures that gRPC service clients are properly
// initialized before use. All client fields are private and can only be accessed through
// safe getter methods that ensure registration before returning clients.
//
// Example usage:
//   client := GetGrpcClient()
//
//   // Safe access pattern - always use getter methods
//   nodeClient, err := client.GetNodeClient()
//   if err != nil {
//       return fmt.Errorf("failed to get node client: %v", err)
//   }
//   resp, err := nodeClient.Register(ctx, req)
//
//   // Alternative with timeout
//   taskClient, err := client.GetTaskClientWithTimeout(5 * time.Second)
//   if err != nil {
//       return fmt.Errorf("failed to get task client: %v", err)
//   }
//   resp, err := taskClient.Connect(ctx)

type GrpcClient struct {
	// settings
	address string
	timeout time.Duration

	// internals
	conn    *grpc.ClientConn
	once    sync.Once
	stopped bool
	stop    chan struct{}
	interfaces.Logger

	// clients (private to enforce safe access through getter methods)
	nodeClient             grpc2.NodeServiceClient
	taskClient             grpc2.TaskServiceClient
	modelBaseServiceClient grpc2.ModelBaseServiceClient
	dependencyClient       grpc2.DependencyServiceClient
	metricClient           grpc2.MetricServiceClient
	syncClient             grpc2.SyncServiceClient

	// Add new fields for state management
	state     connectivity.State
	stateMux  sync.RWMutex
	reconnect chan struct{}

	// Circuit breaker fields
	failureCount int
	lastFailure  time.Time
	cbMux        sync.RWMutex

	// Reconnection control
	reconnecting bool
	reconnectMux sync.Mutex

	// Registration status
	registered    bool
	registeredMux sync.RWMutex

	// Health monitoring
	healthClient       grpc_health_v1.HealthClient
	healthCheckEnabled bool
	healthCheckMux     sync.RWMutex

	// Goroutine management
	wg sync.WaitGroup
}

func (c *GrpcClient) Start() {
	c.once.Do(func() {
		// initialize stop channel before any goroutines
		if c.stop == nil {
			c.stop = make(chan struct{})
		}

		// initialize reconnect channel
		c.reconnect = make(chan struct{}, 1) // Make it buffered to prevent blocking

		// connect first, then start monitoring
		err := c.connect()
		if err != nil {
			c.Errorf("failed initial connection, will retry: %v", err)
			// Don't fatal here, let reconnection handle it
		}

		// start monitoring after connection attempt with proper tracking
		c.wg.Add(2) // Track both monitoring goroutines
		go func() {
			defer c.wg.Done()
			c.monitorState()
		}()

		// start health monitoring
		go func() {
			defer c.wg.Done()
			c.startHealthMonitor()
		}()
	})
}

func (c *GrpcClient) Stop() error {
	// Prevent multiple stops
	c.reconnectMux.Lock()
	if c.stopped {
		c.reconnectMux.Unlock()
		return nil
	}
	c.stopped = true
	c.reconnectMux.Unlock()

	c.setRegistered(false)

	// Close channels safely
	select {
	case c.stop <- struct{}{}:
	default:
	}

	// Wait for goroutines to finish
	done := make(chan struct{})
	go func() {
		c.wg.Wait()
		close(done)
	}()

	// Give goroutines time to finish gracefully, then force stop
	select {
	case <-done:
		c.Debugf("all goroutines stopped gracefully")
	case <-time.After(10 * time.Second):
		c.Warnf("some goroutines did not stop gracefully within timeout")
	}

	// Close connection
	if c.conn != nil {
		if err := c.conn.Close(); err != nil {
			c.Errorf("failed to close connection: %v", err)
			return err
		}
	}

	c.Infof("stopped and disconnected from %s", c.address)
	return nil
}

func (c *GrpcClient) WaitForReady() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			if c.IsReady() {
				c.Debugf("client is now ready")
				return
			}
		case <-c.stop:
			c.Errorf("client has stopped")
		}
	}
}

func (c *GrpcClient) register() {
	c.Debugf("registering gRPC service clients")
	c.nodeClient = grpc2.NewNodeServiceClient(c.conn)
	c.modelBaseServiceClient = grpc2.NewModelBaseServiceClient(c.conn)
	c.taskClient = grpc2.NewTaskServiceClient(c.conn)
	c.dependencyClient = grpc2.NewDependencyServiceClient(c.conn)
	c.metricClient = grpc2.NewMetricServiceClient(c.conn)
	c.syncClient = grpc2.NewSyncServiceClient(c.conn)
	c.healthClient = grpc_health_v1.NewHealthClient(c.conn)

	// Enable health checks by default for new connections
	c.setHealthCheckEnabled(true)

	// Mark as registered
	c.setRegistered(true)
	c.Infof("gRPC service clients successfully registered")
}

func (c *GrpcClient) Context() (ctx context.Context, cancel context.CancelFunc) {
	return context.WithTimeout(context.Background(), c.timeout)
}

func (c *GrpcClient) IsReady() (res bool) {
	if c.conn == nil {
		return false
	}
	state := c.conn.GetState()
	return state == connectivity.Ready
}

func (c *GrpcClient) IsReadyAndRegistered() (res bool) {
	return c.IsReady() && c.IsRegistered()
}

func (c *GrpcClient) IsClosed() (res bool) {
	if c.conn != nil {
		return c.conn.GetState() == connectivity.Shutdown
	}
	return false
}

func (c *GrpcClient) monitorState() {
	defer func() {
		if r := recover(); r != nil {
			c.Errorf("state monitor panic recovered: %v", r)
		}
	}()

	var (
		idleStartTime = time.Time{}
		ticker        = time.NewTicker(stateMonitorInterval)
	)
	defer ticker.Stop()

	for {
		select {
		case <-c.stop:
			c.Debugf("state monitor stopping")
			return
		case <-ticker.C:
			if c.stopped {
				return
			}

			c.checkAndHandleStateChange(&idleStartTime)
		}
	}
}

func (c *GrpcClient) checkAndHandleStateChange(idleStartTime *time.Time) {
	if c.conn == nil {
		return
	}

	previous := c.getState()
	current := c.conn.GetState()

	if previous == current {
		// Handle prolonged IDLE state - but be more lenient
		if current == connectivity.Idle && !idleStartTime.IsZero() &&
			time.Since(*idleStartTime) > idleGracePeriod {
			c.Debugf("connection has been IDLE for %v, checking if reconnection is needed", time.Since(*idleStartTime))
			// Only reconnect if we can't make a simple call
			if !c.testConnection() {
				c.triggerReconnection("prolonged IDLE state with failed connection test")
			}
			*idleStartTime = time.Time{}
		}
		return
	}

	// State changed
	c.setState(current)
	c.Infof("connection state: %s -> %s", previous, current)

	switch current {
	case connectivity.TransientFailure:
		c.setRegistered(false)
		c.Warnf("connection in transient failure, will attempt reconnection")
		c.triggerReconnection(fmt.Sprintf("state change to %s", current))

	case connectivity.Shutdown:
		c.setRegistered(false)
		c.Warnf("connection state changed to SHUTDOWN - stopped flag: %v", c.stopped)
		if !c.stopped {
			c.Errorf("connection shutdown unexpectedly")
			c.triggerReconnection(fmt.Sprintf("state change to %s", current))
		} else {
			c.Debugf("connection shutdown expected (client stopped)")
		}

	case connectivity.Idle:
		if previous == connectivity.Ready {
			*idleStartTime = time.Now()
			c.Debugf("connection went IDLE, grace period started")
		}

	case connectivity.Ready:
		*idleStartTime = time.Time{}
		c.recordSuccess()
		if !c.IsRegistered() {
			c.register() // Re-register if needed
		}
	}
}

func (c *GrpcClient) triggerReconnection(reason string) {
	if c.stopped || c.isCircuitBreakerOpen() {
		return
	}

	select {
	case c.reconnect <- struct{}{}:
		c.Infof("reconnection triggered: %s", reason)
	default:
		c.Debugf("reconnection already queued")
	}
}

func (c *GrpcClient) setState(state connectivity.State) {
	c.stateMux.Lock()
	defer c.stateMux.Unlock()
	c.state = state
}

func (c *GrpcClient) getState() connectivity.State {
	c.stateMux.RLock()
	defer c.stateMux.RUnlock()
	return c.state
}

func (c *GrpcClient) setRegistered(registered bool) {
	c.registeredMux.Lock()
	defer c.registeredMux.Unlock()
	c.registered = registered
}

func (c *GrpcClient) IsRegistered() bool {
	c.registeredMux.RLock()
	defer c.registeredMux.RUnlock()
	return c.registered
}

func (c *GrpcClient) setHealthCheckEnabled(enabled bool) {
	c.healthCheckMux.Lock()
	defer c.healthCheckMux.Unlock()
	c.healthCheckEnabled = enabled
}

func (c *GrpcClient) isHealthCheckEnabled() bool {
	c.healthCheckMux.RLock()
	defer c.healthCheckMux.RUnlock()
	return c.healthCheckEnabled
}

func (c *GrpcClient) testConnection() bool {
	if !c.IsReady() || !c.IsRegistered() {
		return false
	}

	// Try a simple health check if available, otherwise just check connection state
	if c.isHealthCheckEnabled() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		_, err := c.healthClient.Check(ctx, &grpc_health_v1.HealthCheckRequest{})
		return err == nil
	}

	// If health checks are disabled, just verify the connection state
	return c.conn != nil && c.conn.GetState() == connectivity.Ready
}

func (c *GrpcClient) WaitForRegistered() {
	ticker := time.NewTicker(registrationCheckInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			if c.IsRegistered() {
				c.Debugf("client is now registered")
				return
			}
		case <-c.stop:
			c.Errorf("client has stopped while waiting for registration")
			return
		}
	}
}

// Safe client getters that ensure registration before returning clients
// These methods will wait for registration to complete or return an error if the client is stopped

func (c *GrpcClient) GetNodeClient() (grpc2.NodeServiceClient, error) {
	// Use longer timeout during reconnection scenarios
	timeout := defaultClientTimeout
	c.reconnectMux.Lock()
	if c.reconnecting {
		timeout = reconnectionClientTimeout
	}
	c.reconnectMux.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	return c.GetNodeClientWithContext(ctx)
}

func (c *GrpcClient) GetTaskClient() (grpc2.TaskServiceClient, error) {
	// Use longer timeout during reconnection scenarios
	timeout := defaultClientTimeout
	c.reconnectMux.Lock()
	if c.reconnecting {
		timeout = reconnectionClientTimeout
	}
	c.reconnectMux.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	return c.GetTaskClientWithContext(ctx)
}

func (c *GrpcClient) GetModelBaseServiceClient() (grpc2.ModelBaseServiceClient, error) {
	// Use longer timeout during reconnection scenarios
	timeout := defaultClientTimeout
	c.reconnectMux.Lock()
	if c.reconnecting {
		timeout = reconnectionClientTimeout
	}
	c.reconnectMux.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	return c.GetModelBaseServiceClientWithContext(ctx)
}

func (c *GrpcClient) GetDependencyClient() (grpc2.DependencyServiceClient, error) {
	// Use longer timeout during reconnection scenarios
	timeout := defaultClientTimeout
	c.reconnectMux.Lock()
	if c.reconnecting {
		timeout = reconnectionClientTimeout
	}
	c.reconnectMux.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	return c.GetDependencyClientWithContext(ctx)
}

func (c *GrpcClient) GetMetricClient() (grpc2.MetricServiceClient, error) {
	// Use longer timeout during reconnection scenarios
	timeout := defaultClientTimeout
	c.reconnectMux.Lock()
	if c.reconnecting {
		timeout = reconnectionClientTimeout
	}
	c.reconnectMux.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	return c.GetMetricClientWithContext(ctx)
}

func (c *GrpcClient) GetSyncClient() (grpc2.SyncServiceClient, error) {
	// Use longer timeout during reconnection scenarios
	timeout := defaultClientTimeout
	c.reconnectMux.Lock()
	if c.reconnecting {
		timeout = reconnectionClientTimeout
	}
	c.reconnectMux.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	return c.GetSyncClientWithContext(ctx)
}

// Safe client getters with timeout - these methods will wait up to the specified timeout
// for registration to complete before returning an error

func (c *GrpcClient) GetNodeClientWithTimeout(timeout time.Duration) (grpc2.NodeServiceClient, error) {
	if c.stopped {
		return nil, fmt.Errorf("grpc client is stopped")
	}
	// Check if connection is in bad state and needs reconnection
	if c.conn != nil && (c.conn.GetState() == connectivity.Shutdown || c.conn.GetState() == connectivity.TransientFailure) {
		c.Debugf("connection in bad state (%s), triggering reconnection", c.conn.GetState())
		c.triggerReconnection(fmt.Sprintf("bad connection state: %s", c.conn.GetState()))
	}
	if !c.IsRegistered() {
		if err := c.waitForRegisteredWithTimeout(timeout); err != nil {
			return nil, fmt.Errorf("failed to get node client: %w", err)
		}
	}
	return c.nodeClient, nil
}

func (c *GrpcClient) GetTaskClientWithTimeout(timeout time.Duration) (grpc2.TaskServiceClient, error) {
	if c.stopped {
		return nil, fmt.Errorf("grpc client is stopped")
	}
	// Check if connection is in bad state and needs reconnection
	if c.conn != nil && (c.conn.GetState() == connectivity.Shutdown || c.conn.GetState() == connectivity.TransientFailure) {
		c.Debugf("connection in bad state (%s), triggering reconnection", c.conn.GetState())
		c.triggerReconnection(fmt.Sprintf("bad connection state: %s", c.conn.GetState()))
	}
	if !c.IsRegistered() {
		if err := c.waitForRegisteredWithTimeout(timeout); err != nil {
			return nil, fmt.Errorf("failed to get task client: %w", err)
		}
	}
	return c.taskClient, nil
}

func (c *GrpcClient) GetModelBaseServiceClientWithTimeout(timeout time.Duration) (grpc2.ModelBaseServiceClient, error) {
	if c.stopped {
		return nil, fmt.Errorf("grpc client is stopped")
	}
	// Check if connection is in bad state and needs reconnection
	if c.conn != nil && (c.conn.GetState() == connectivity.Shutdown || c.conn.GetState() == connectivity.TransientFailure) {
		c.Debugf("connection in bad state (%s), triggering reconnection", c.conn.GetState())
		c.triggerReconnection(fmt.Sprintf("bad connection state: %s", c.conn.GetState()))
	}
	if !c.IsRegistered() {
		if err := c.waitForRegisteredWithTimeout(timeout); err != nil {
			return nil, fmt.Errorf("failed to get model base service client: %w", err)
		}
	}
	return c.modelBaseServiceClient, nil
}

func (c *GrpcClient) GetDependencyClientWithTimeout(timeout time.Duration) (grpc2.DependencyServiceClient, error) {
	if c.stopped {
		return nil, fmt.Errorf("grpc client is stopped")
	}
	// Check if connection is in bad state and needs reconnection
	if c.conn != nil && (c.conn.GetState() == connectivity.Shutdown || c.conn.GetState() == connectivity.TransientFailure) {
		c.Debugf("connection in bad state (%s), triggering reconnection", c.conn.GetState())
		c.triggerReconnection(fmt.Sprintf("bad connection state: %s", c.conn.GetState()))
	}
	if !c.IsRegistered() {
		if err := c.waitForRegisteredWithTimeout(timeout); err != nil {
			return nil, fmt.Errorf("failed to get dependency client: %w", err)
		}
	}
	return c.dependencyClient, nil
}

func (c *GrpcClient) GetMetricClientWithTimeout(timeout time.Duration) (grpc2.MetricServiceClient, error) {
	if c.stopped {
		return nil, fmt.Errorf("grpc client is stopped")
	}
	// Check if connection is in bad state and needs reconnection
	if c.conn != nil && (c.conn.GetState() == connectivity.Shutdown || c.conn.GetState() == connectivity.TransientFailure) {
		c.Debugf("connection in bad state (%s), triggering reconnection", c.conn.GetState())
		c.triggerReconnection(fmt.Sprintf("bad connection state: %s", c.conn.GetState()))
	}
	if !c.IsRegistered() {
		if err := c.waitForRegisteredWithTimeout(timeout); err != nil {
			return nil, fmt.Errorf("failed to get metric client: %w", err)
		}
	}
	return c.metricClient, nil
}

func (c *GrpcClient) waitForRegisteredWithTimeout(timeout time.Duration) error {
	ticker := time.NewTicker(registrationCheckInterval)
	defer ticker.Stop()
	timer := time.NewTimer(timeout)
	defer timer.Stop()

	for {
		select {
		case <-ticker.C:
			if c.IsRegistered() {
				c.Debugf("client is now registered")
				return nil
			}
		case <-timer.C:
			return fmt.Errorf("timeout waiting for client registration after %v", timeout)
		case <-c.stop:
			return fmt.Errorf("client has stopped while waiting for registration")
		}
	}
}

// Context-aware client getters
func (c *GrpcClient) GetNodeClientWithContext(ctx context.Context) (grpc2.NodeServiceClient, error) {
	client, err := c.getClientWithContext(ctx, func() interface{} { return c.nodeClient }, "node")
	if err != nil {
		return nil, err
	}
	return client.(grpc2.NodeServiceClient), nil
}

func (c *GrpcClient) GetTaskClientWithContext(ctx context.Context) (grpc2.TaskServiceClient, error) {
	client, err := c.getClientWithContext(ctx, func() interface{} { return c.taskClient }, "task")
	if err != nil {
		return nil, err
	}
	return client.(grpc2.TaskServiceClient), nil
}

func (c *GrpcClient) GetModelBaseServiceClientWithContext(ctx context.Context) (grpc2.ModelBaseServiceClient, error) {
	client, err := c.getClientWithContext(ctx, func() interface{} { return c.modelBaseServiceClient }, "model base service")
	if err != nil {
		return nil, err
	}
	return client.(grpc2.ModelBaseServiceClient), nil
}

func (c *GrpcClient) GetDependencyClientWithContext(ctx context.Context) (grpc2.DependencyServiceClient, error) {
	client, err := c.getClientWithContext(ctx, func() interface{} { return c.dependencyClient }, "dependency")
	if err != nil {
		return nil, err
	}
	return client.(grpc2.DependencyServiceClient), nil
}

func (c *GrpcClient) GetMetricClientWithContext(ctx context.Context) (grpc2.MetricServiceClient, error) {
	client, err := c.getClientWithContext(ctx, func() interface{} { return c.metricClient }, "metric")
	if err != nil {
		return nil, err
	}
	return client.(grpc2.MetricServiceClient), nil
}

func (c *GrpcClient) GetSyncClientWithContext(ctx context.Context) (grpc2.SyncServiceClient, error) {
	client, err := c.getClientWithContext(ctx, func() interface{} { return c.syncClient }, "sync")
	if err != nil {
		return nil, err
	}
	return client.(grpc2.SyncServiceClient), nil
}

func (c *GrpcClient) getClientWithContext(ctx context.Context, getter func() interface{}, clientType string) (interface{}, error) {
	if c.stopped {
		return nil, fmt.Errorf("grpc client is stopped")
	}

	if c.IsRegistered() {
		return getter(), nil
	}

	// Check if we're reconnecting to provide better error context
	c.reconnectMux.Lock()
	isReconnecting := c.reconnecting
	c.reconnectMux.Unlock()

	// Wait for registration with context
	ticker := time.NewTicker(registrationCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			if isReconnecting {
				return nil, fmt.Errorf("context cancelled while waiting for %s client registration: reconnection in progress, retry recommended", clientType)
			}
			return nil, fmt.Errorf("context cancelled while waiting for %s client registration", clientType)
		case <-c.stop:
			return nil, fmt.Errorf("client stopped while waiting for %s client registration", clientType)
		case <-ticker.C:
			// Update reconnection status in case it changed
			c.reconnectMux.Lock()
			isReconnecting = c.reconnecting
			c.reconnectMux.Unlock()

			if c.IsRegistered() {
				return getter(), nil
			}
		}
	}
}

func (c *GrpcClient) connect() error {
	// Start reconnection handling goroutine with proper tracking
	c.wg.Add(1)
	go func() {
		defer c.wg.Done()
		c.handleReconnections()
	}()

	// Initial connection attempt
	return c.doConnect()
}

func (c *GrpcClient) handleReconnections() {
	defer func() {
		if r := recover(); r != nil {
			c.Errorf("reconnection handler panic: %v", r)
		}
	}()

	for {
		select {
		case <-c.stop:
			c.Debugf("reconnection handler stopping")
			return

		case <-c.reconnect:
			if c.stopped || !c.canAttemptConnection() {
				continue
			}

			c.executeReconnection()
		}
	}
}

func (c *GrpcClient) executeReconnection() {
	c.reconnectMux.Lock()
	if c.reconnecting {
		c.reconnectMux.Unlock()
		return
	}
	c.reconnecting = true
	c.reconnectMux.Unlock()

	// Don't use defer to clear reconnecting flag - we need to control the timing precisely
	// to ensure all dependent services have completed registration before clearing it

	c.Infof("executing reconnection to %s (current state: %s)", c.address, c.getState())

	if err := c.doConnect(); err != nil {
		c.Errorf("reconnection failed: %v", err)
		c.recordFailure()

		// Clear reconnecting flag on failure so we can retry
		c.reconnectMux.Lock()
		c.reconnecting = false
		c.reconnectMux.Unlock()

		// Exponential backoff before allowing next attempt
		backoffDuration := c.calculateBackoff()
		c.Warnf("will retry reconnection after %v backoff", backoffDuration)
		time.Sleep(backoffDuration)

		// Trigger another reconnection attempt after backoff
		// This ensures we keep trying even if network was down during first attempt
		c.Debugf("backoff complete, triggering reconnection retry")
		select {
		case c.reconnect <- struct{}{}:
			c.Debugf("reconnection retry triggered")
		default:
			c.Debugf("reconnection retry already pending")
		}
	} else {
		c.recordSuccess()
		c.Infof("reconnection successful - connection state: %s, registered: %v", c.getState(), c.IsRegistered())

		// Stabilization: wait a moment to ensure connection is truly stable
		// This prevents immediate flapping if the network is still unstable
		c.Debugf("stabilizing connection for %v", connectionStabilizationDelay)
		time.Sleep(connectionStabilizationDelay)

		// Verify connection is still stable after delay
		if c.conn != nil && c.conn.GetState() == connectivity.Ready {
			c.Infof("connection stabilization successful")
		} else {
			c.Warnf("connection became unstable during stabilization")
		}

		// Wait for full registration and service readiness before clearing reconnecting flag
		// This ensures all dependent services can successfully get their clients with extended timeout
		if c.waitForFullReconnectionReady() {
			c.Infof("reconnection fully complete, all services ready")
		} else {
			c.Warnf("reconnection completed but some checks didn't pass within timeout")
		}

		// Now it's safe to clear the reconnecting flag
		c.reconnectMux.Lock()
		c.reconnecting = false
		c.reconnectMux.Unlock()
		c.Infof("resuming normal operation mode")
	}
}

// waitForFullReconnectionReady waits for the client to be fully ready after reconnection
// by verifying all critical service clients can be successfully obtained.
// This ensures dependent services won't fail with "context cancelled" errors.
func (c *GrpcClient) waitForFullReconnectionReady() bool {
	c.Debugf("waiting for full reconnection readiness (max %v)", maxReconnectionWait)
	startTime := time.Now()
	checkCount := 0

	for time.Since(startTime) < maxReconnectionWait {
		checkCount++

		// Check 1: Connection must be in READY state
		if c.conn == nil || c.conn.GetState() != connectivity.Ready {
			c.Debugf("check %d: connection not ready (state: %v)", checkCount, c.getState())
			time.Sleep(reconnectionCheckInterval)
			continue
		}

		// Check 2: Client must be registered
		if !c.IsRegistered() {
			c.Debugf("check %d: client not yet registered", checkCount)
			time.Sleep(reconnectionCheckInterval)
			continue
		}

		// Check 3: Verify we can actually get service clients without timeout
		// This is the critical check that ensures dependent services won't fail
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)

		// Try to get model base service client (most commonly used by TaskHandler)
		_, err1 := c.GetModelBaseServiceClientWithContext(ctx)

		// Try to get task client (used for task operations)
		_, err2 := c.GetTaskClientWithContext(ctx)

		cancel()

		if err1 != nil || err2 != nil {
			c.Debugf("check %d: service clients not ready (model: %v, task: %v)",
				checkCount, err1, err2)
			time.Sleep(reconnectionCheckInterval)
			continue
		}

		// All checks passed!
		elapsed := time.Since(startTime)
		c.Infof("full reconnection readiness achieved after %v (%d checks)", elapsed, checkCount)
		return true
	}

	// Timeout reached
	elapsed := time.Since(startTime)
	c.Warnf("reconnection readiness checks did not complete within %v (elapsed: %v, checks: %d)",
		maxReconnectionWait, elapsed, checkCount)
	return false
}

// Enhanced circuit breaker methods
func (c *GrpcClient) getCircuitBreakerState() circuitBreakerState {
	c.cbMux.RLock()
	defer c.cbMux.RUnlock()

	if c.failureCount < maxFailures {
		return cbClosed
	}

	timeSinceLastFailure := time.Since(c.lastFailure)
	if timeSinceLastFailure > cbResetTime {
		return cbHalfOpen
	}

	return cbOpen
}

func (c *GrpcClient) isCircuitBreakerOpen() bool {
	return c.getCircuitBreakerState() == cbOpen
}

func (c *GrpcClient) canAttemptConnection() bool {
	state := c.getCircuitBreakerState()

	switch state {
	case cbClosed:
		return true
	case cbHalfOpen:
		c.cbMux.RLock()
		canRetry := time.Since(c.lastFailure) > cbHalfOpenRetryInterval
		c.cbMux.RUnlock()
		return canRetry
	case cbOpen:
		return false
	}

	return false
}

func (c *GrpcClient) recordFailure() {
	c.cbMux.Lock()
	defer c.cbMux.Unlock()
	c.failureCount++
	c.lastFailure = time.Now()
	if c.failureCount >= maxFailures {
		c.Warnf("circuit breaker opened after %d consecutive failures", c.failureCount)
	}
}

func (c *GrpcClient) recordSuccess() {
	c.cbMux.Lock()
	defer c.cbMux.Unlock()
	if c.failureCount > 0 {
		c.Infof("connection restored, resetting circuit breaker (was %d failures)", c.failureCount)
	}
	c.failureCount = 0
	c.lastFailure = time.Time{}
}

func (c *GrpcClient) calculateBackoff() time.Duration {
	c.cbMux.RLock()
	failures := c.failureCount
	c.cbMux.RUnlock()

	// Exponential backoff: 1s, 2s, 4s, 8s, max 30s
	backoff := time.Duration(1<<min(failures-1, 5)) * time.Second
	if backoff > 30*time.Second {
		backoff = 30 * time.Second
	}

	return backoff
}

func (c *GrpcClient) doConnect() error {
	c.Debugf("attempting connection to %s", c.address)
	c.setRegistered(false)

	// Close existing connection
	if c.conn != nil {
		c.Debugf("closing existing connection (state: %s)", c.conn.GetState())
		c.conn.Close()
		c.conn = nil
	}

	opts := c.getDialOptions()

	// Create connection with context timeout - using NewClient instead of DialContext
	conn, err := grpc.NewClient(c.address, opts...)
	if err != nil {
		return fmt.Errorf("failed to create client for %s: %w", c.address, err)
	}

	c.conn = conn

	// Connect and wait for ready state with timeout
	ctx, cancel := context.WithTimeout(context.Background(), connectionTimeout)
	defer cancel()

	c.Debugf("initiating connection to %s", c.address)
	c.conn.Connect()
	if err := c.waitForConnectionReady(ctx); err != nil {
		c.Errorf("failed to reach ready state: %v", err)
		c.conn.Close()
		c.conn = nil
		return err
	}

	c.Infof("connected to %s (state: %s)", c.address, c.conn.GetState())
	c.register()

	return nil
}

func (c *GrpcClient) getDialOptions() []grpc.DialOption {
	return []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithChainUnaryInterceptor(middlewares.GetGrpcClientAuthTokenUnaryChainInterceptor()),
		grpc.WithChainStreamInterceptor(middlewares.GetGrpcClientAuthTokenStreamChainInterceptor()),
		grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Time:                20 * time.Second,
			Timeout:             5 * time.Second,
			PermitWithoutStream: true,
		}),
		grpc.WithDefaultCallOptions(
			grpc.WaitForReady(false), // Fail fast for initial connection
			grpc.MaxCallRecvMsgSize(4*1024*1024),
			grpc.MaxCallSendMsgSize(4*1024*1024),
		),
		grpc.WithInitialWindowSize(65535),
		grpc.WithInitialConnWindowSize(65535),
	}
}

func (c *GrpcClient) waitForConnectionReady(ctx context.Context) error {
	for {
		state := c.conn.GetState()
		switch state {
		case connectivity.Ready:
			return nil
		case connectivity.TransientFailure, connectivity.Shutdown:
			return fmt.Errorf("connection failed with state: %s", state)
		}

		if !c.conn.WaitForStateChange(ctx, state) {
			return fmt.Errorf("connection timeout")
		}
	}
}

// Health monitoring methods
func (c *GrpcClient) startHealthMonitor() {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				c.Errorf("health monitor panic: %v", r)
			}
		}()

		ticker := time.NewTicker(healthCheckInterval)
		defer ticker.Stop()

		for {
			select {
			case <-c.stop:
				c.Debugf("health monitor stopping")
				return
			case <-ticker.C:
				if !c.stopped {
					c.performHealthCheck()
				}
			}
		}
	}()
}

func (c *GrpcClient) performHealthCheck() {
	if !c.IsReady() || !c.IsRegistered() || !c.isHealthCheckEnabled() {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := c.healthClient.Check(ctx, &grpc_health_v1.HealthCheckRequest{})

	if err != nil {
		// Check if the error is due to unimplemented health service
		if strings.Contains(err.Error(), "Unimplemented") && strings.Contains(err.Error(), "grpc.health.v1.Health") {
			c.Warnf("health service not implemented on server, disabling health checks")
			c.setHealthCheckEnabled(false)
			// Don't trigger reconnection for unimplemented health service
			return
		}

		c.Warnf("health check failed: %v", err)
		c.triggerReconnection("health check failure")
	} else {
		c.Debugf("health check passed")
	}
}

func newGrpcClient() (c *GrpcClient) {
	client := &GrpcClient{
		address:            utils.GetGrpcAddress(),
		timeout:            10 * time.Second,
		stop:               make(chan struct{}),
		Logger:             utils.NewLogger("GrpcClient"),
		state:              connectivity.Idle,
		healthCheckEnabled: true,
	}

	return client
}

var _client *GrpcClient
var _clientOnce sync.Once
var _clientMux sync.Mutex

func GetGrpcClient() *GrpcClient {
	_clientMux.Lock()
	defer _clientMux.Unlock()

	_clientOnce.Do(func() {
		_client = newGrpcClient()
		go _client.Start()
	})
	return _client
}

// ResetGrpcClient creates a completely new gRPC client instance
// This is needed when the client gets stuck and needs to be fully restarted
func ResetGrpcClient() *GrpcClient {
	_clientMux.Lock()
	defer _clientMux.Unlock()

	// Stop the old client if it exists
	if _client != nil {
		_client.Stop()
	}

	// Reset the sync.Once so we can create a new client
	_clientOnce = sync.Once{}
	_client = nil

	// Create and start the new client
	_clientOnce.Do(func() {
		_client = newGrpcClient()
		go _client.Start()
	})

	return _client
}
