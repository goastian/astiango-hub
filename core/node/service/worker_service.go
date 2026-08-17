package service

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/goastian/astiango-hub/core/models/models"

	"github.com/cenkalti/backoff/v4"
	"github.com/goastian/astiango-hub/core/grpc/client"
	"github.com/goastian/astiango-hub/core/interfaces"
	client2 "github.com/goastian/astiango-hub/core/models/client"
	nodeconfig "github.com/goastian/astiango-hub/core/node/config"
	"github.com/goastian/astiango-hub/core/task/handler"
	"github.com/goastian/astiango-hub/core/utils"
	"github.com/goastian/astiango-hub/grpc"
	"go.mongodb.org/mongo-driver/bson"
)

type WorkerService struct {
	// dependencies
	cfgSvc     interfaces.NodeConfigService
	handlerSvc *handler.Service
	healthSvc  *HealthService

	// settings
	heartbeatInterval time.Duration

	// context and synchronization
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
	mu     sync.RWMutex

	// internals
	stopped bool
	n       *models.Node
	isReady bool
	interfaces.Logger
}

func (svc *WorkerService) Start() {
	// initialize context for the service
	svc.ctx, svc.cancel = context.WithCancel(context.Background())

	// wait for grpc client ready
	client.GetGrpcClient().WaitForReady()

	// register to master
	svc.register()

	// mark as ready after registration
	svc.mu.Lock()
	svc.isReady = true
	svc.mu.Unlock()
	svc.healthSvc.SetReady(true)

	// start health check server
	svc.wg.Add(1)
	go func() {
		defer svc.wg.Done()
		// Start health service with worker-specific health check
		svc.healthSvc.Start(func() bool {
			svc.mu.RLock()
			defer svc.mu.RUnlock()
			return svc.isReady && !svc.stopped && client.GetGrpcClient() != nil && !client.GetGrpcClient().IsClosed()
		})
	}()

	// subscribe to master
	svc.wg.Add(1)
	go func() {
		defer svc.wg.Done()
		svc.subscribe()
	}()

	// start sending heartbeat to master
	svc.wg.Add(1)
	go func() {
		defer svc.wg.Done()
		svc.reportStatus()
	}()

	// start task handler
	svc.wg.Add(1)
	go func() {
		defer svc.wg.Done()
		svc.handlerSvc.Start()
	}()

	svc.Infof("worker[%s] service started", svc.cfgSvc.GetNodeKey())

	// wait for quit signal
	svc.Wait()

	// stop
	svc.Stop()
}

func (svc *WorkerService) Wait() {
	utils.DefaultWait()
}

func (svc *WorkerService) Stop() {
	svc.mu.Lock()
	if svc.stopped {
		svc.mu.Unlock()
		return
	}
	svc.stopped = true
	svc.mu.Unlock()

	svc.Infof("stopping worker[%s] service...", svc.cfgSvc.GetNodeKey())

	// cancel context to signal all goroutines to stop
	if svc.cancel != nil {
		svc.cancel()
	}

	// stop task handler
	if svc.handlerSvc != nil {
		svc.handlerSvc.Stop()
	}

	// stop health service
	if svc.healthSvc != nil {
		svc.healthSvc.Stop()
	}

	// stop grpc client
	if err := client.GetGrpcClient().Stop(); err != nil {
		svc.Errorf("error stopping grpc client: %v", err)
	}

	// wait for all goroutines to finish with timeout
	done := make(chan struct{})
	go func() {
		svc.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		svc.Infof("worker[%s] service has stopped gracefully", svc.cfgSvc.GetNodeKey())
	case <-time.After(10 * time.Second):
		svc.Warnf("worker[%s] service shutdown timed out", svc.cfgSvc.GetNodeKey())
	}
}

func (svc *WorkerService) register() {
	op := func() (err error) {
		ctx, cancel := client.GetGrpcClient().Context()
		defer cancel()
		nodeClient, err := client.GetGrpcClient().GetNodeClient()
		if err != nil {
			return fmt.Errorf("failed to get node client: %v", err)
		}
		_, err = nodeClient.Register(ctx, &grpc.NodeServiceRegisterRequest{
			NodeKey:    svc.cfgSvc.GetNodeKey(),
			NodeName:   svc.cfgSvc.GetNodeName(),
			MaxRunners: int32(svc.cfgSvc.GetMaxRunners()),
		})
		if err != nil {
			err = fmt.Errorf("failed to register worker[%s]: %v", svc.cfgSvc.GetNodeKey(), err)
			return err
		}
		svc.n, err = client2.NewModelService[models.Node]().GetOne(bson.M{"key": svc.GetConfigService().GetNodeKey()}, nil)
		if err != nil {
			err = fmt.Errorf("failed to get node: %v", err)
			return err
		}
		svc.Infof("worker[%s] registered to master. id: %s", svc.GetConfigService().GetNodeKey(), svc.n.Id.Hex())
		return nil
	}
	b := backoff.NewExponentialBackOff()
	n := func(err error, duration time.Duration) {
		svc.Errorf("register worker[%s] error: %v", svc.cfgSvc.GetNodeKey(), err)
		svc.Infof("retry in %.1f seconds", duration.Seconds())
	}
	err := backoff.RetryNotify(op, b, n)
	if err != nil {
		svc.Fatalf("failed to register worker[%s]: %v", svc.cfgSvc.GetNodeKey(), err)
		panic(err)
	}
}

func (svc *WorkerService) reportStatus() {
	ticker := time.NewTicker(svc.heartbeatInterval)
	defer ticker.Stop()

	for {
		select {
		case <-svc.ctx.Done():
			svc.Debugf("heartbeat goroutine stopping due to context cancellation")
			return
		case <-ticker.C:
			// return if client is closed
			if client.GetGrpcClient().IsClosed() {
				svc.Debugf("heartbeat goroutine stopping due to closed grpc client")
				return
			}
			// send heartbeat
			svc.sendHeartbeat()
		}
	}
}

func (svc *WorkerService) GetConfigService() (cfgSvc interfaces.NodeConfigService) {
	return svc.cfgSvc
}

func (svc *WorkerService) subscribe() {
	// Configure exponential backoff
	b := backoff.NewExponentialBackOff()
	b.InitialInterval = 1 * time.Second
	b.MaxInterval = 30 * time.Second   // Reduced from 1 minute
	b.MaxElapsedTime = 0 * time.Minute // Never give up
	b.Multiplier = 2.0

	for {
		select {
		case <-svc.ctx.Done():
			svc.Infof("subscription stopped due to context cancellation")
			return
		default:
		}

		// Use backoff for connection attempts
		operation := func() error {
			svc.Debugf("attempting to subscribe to master")

			// Wait for gRPC client to be ready and registered after reconnection
			grpcClient := client.GetGrpcClient()

			waitStart := time.Now()
			checkCount := 0
			for !grpcClient.IsReadyAndRegistered() {
				select {
				case <-svc.ctx.Done():
					return svc.ctx.Err()
				case <-time.After(500 * time.Millisecond):
					checkCount++
					// Log periodically while waiting
					if checkCount%20 == 0 { // Every 10 seconds
						svc.Warnf("still waiting for gRPC client (%.1fs)", time.Since(waitStart).Seconds())

						// Force a reconnection attempt if we've been waiting too long
						if time.Since(waitStart) > 15*time.Second {
							svc.Warnf("forcing gRPC client reset due to prolonged wait")
							grpcClient = client.ResetGrpcClient()
							waitStart = time.Now()
							checkCount = 0
						}

						// Check if client is in SHUTDOWN state and force restart
						if !grpcClient.IsReady() && grpcClient.IsClosed() {
							svc.Warnf("gRPC client is in SHUTDOWN state, forcing reset")
							grpcClient = client.ResetGrpcClient()
							waitStart = time.Now()
							checkCount = 0
						}
					}
				}
			}
			svc.Debugf("gRPC client is ready and registered after %.1fs", time.Since(waitStart).Seconds())

			nodeClient, err := grpcClient.GetNodeClient()
			if err != nil {
				svc.Errorf("failed to get node client: %v", err)
				return err
			}

			// Use service context for proper cancellation
			stream, err := nodeClient.Subscribe(svc.ctx, &grpc.NodeServiceSubscribeRequest{
				NodeKey: svc.cfgSvc.GetNodeKey(),
			})
			if err != nil {
				svc.Errorf("failed to subscribe to master: %v", err)
				return err
			}
			svc.Infof("successfully subscribed to master")

			// Handle messages
			for {
				select {
				case <-svc.ctx.Done():
					svc.Debugf("subscription message loop stopped due to context cancellation")
					return nil
				default:
				}

				msg, err := stream.Recv()
				if err != nil {
					if svc.ctx.Err() != nil {
						// Context was cancelled, this is expected
						svc.Debugf("stream receive cancelled due to context")
						return nil
					}
					if client.GetGrpcClient().IsClosed() {
						svc.Errorf("connection to master is closed: %v", err)
						return err
					}
					svc.Errorf("failed to receive message from master: %v", err)
					return err
				}

				switch msg.Code {
				case grpc.NodeServiceSubscribeCode_HEARTBEAT:
					// do nothing
				}
			}
		}

		// Execute with backoff
		err := backoff.Retry(operation, backoff.WithContext(b, svc.ctx))
		if err != nil {
			if svc.ctx.Err() != nil {
				// Context was cancelled, exit gracefully
				svc.Infof("subscription retry cancelled due to context")
				return
			}
			svc.Errorf("subscription attempt failed: %v", err)
			// Reset backoff for next attempt
			b.Reset()
		} else {
			svc.Debugf("subscription completed successfully")
		}

		// Wait before attempting to reconnect, but respect context cancellation
		select {
		case <-svc.ctx.Done():
			return
		case <-time.After(2 * time.Second):
		}
	}
}

func (svc *WorkerService) sendHeartbeat() {
	// Use extended timeout to allow for reconnection scenarios
	ctx, cancel := context.WithTimeout(svc.ctx, 30*time.Second)
	defer cancel()

	// Retry up to 3 times with exponential backoff for reconnection scenarios
	err := client.RetryWithBackoff(ctx, func() error {
		nodeClient, err := client.GetGrpcClient().GetNodeClient()
		if err != nil {
			return err
		}

		_, err = nodeClient.SendHeartbeat(ctx, &grpc.NodeServiceSendHeartbeatRequest{
			NodeKey: svc.cfgSvc.GetNodeKey(),
		})
		return err
	}, 3, svc.Logger, "heartbeat")

	if err != nil {
		svc.Errorf("failed to send heartbeat: %v", err)
	}
}

func newWorkerService() *WorkerService {
	return &WorkerService{
		heartbeatInterval: 15 * time.Second,
		cfgSvc:            nodeconfig.GetNodeConfigService(),
		handlerSvc:        handler.GetTaskHandlerService(),
		healthSvc:         GetHealthService(),
		isReady:           false,
		Logger:            utils.NewLogger("WorkerService"),
	}
}

var workerService *WorkerService
var workerServiceOnce sync.Once

func GetWorkerService() *WorkerService {
	workerServiceOnce.Do(func() {
		workerService = newWorkerService()
	})
	return workerService
}
