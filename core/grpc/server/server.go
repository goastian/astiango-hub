package server

import (
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/goastian/astiango-hub/core/grpc/middlewares"
	"github.com/goastian/astiango-hub/core/interfaces"
	"github.com/goastian/astiango-hub/core/utils"
	grpc2 "github.com/goastian/astiango-hub/grpc"
	grpcmiddleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpcauth "github.com/grpc-ecosystem/go-grpc-middleware/auth"
	grpcrecovery "github.com/grpc-ecosystem/go-grpc-middleware/recovery"
	errors2 "github.com/pkg/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
)

type GrpcServer struct {
	// settings
	address string

	// internals
	svr     *grpc.Server
	l       net.Listener
	stopped bool
	interfaces.Logger

	// servers
	NodeSvr             *NodeServiceServer
	TaskSvr             *TaskServiceServer
	ModelBaseServiceSvr *ModelBaseServiceServer
	DependencySvr       *DependencyServiceServer
	MetricSvr           *MetricServiceServer
	SyncSvr             *SyncServiceServer
}

func (svr *GrpcServer) Init() {
	svr.register()
}

func (svr *GrpcServer) Start() (err error) {
	// listener
	svr.l, err = net.Listen("tcp", svr.address)
	if err != nil {
		svr.Errorf("failed to listen: %v", err)
		return err
	}
	svr.Infof("server listens to %s", svr.address)

	// start grpc server
	go func() {
		if err := svr.svr.Serve(svr.l); err != nil {
			if errors2.Is(err, grpc.ErrServerStopped) {
				return
			}
			svr.Errorf("failed to serve: %v", err)
		}
	}()

	return nil
}

func (svr *GrpcServer) Stop() (err error) {
	// skip if listener is nil
	if svr.l == nil {
		return nil
	}

	// graceful stop
	svr.Infof("server stopping...")
	svr.svr.Stop()

	// close listener
	svr.Infof("server closing listener...")
	_ = svr.l.Close()

	// mark as stopped
	svr.stopped = true

	// log
	svr.Infof("server stopped")

	return nil
}

func (svr *GrpcServer) register() {
	grpc2.RegisterNodeServiceServer(svr.svr, svr.NodeSvr)
	grpc2.RegisterModelBaseServiceServer(svr.svr, svr.ModelBaseServiceSvr)
	grpc2.RegisterTaskServiceServer(svr.svr, svr.TaskSvr)
	grpc2.RegisterDependencyServiceServer(svr.svr, svr.DependencySvr)
	grpc2.RegisterMetricServiceServer(svr.svr, svr.MetricSvr)
	grpc2.RegisterSyncServiceServer(svr.svr, svr.SyncSvr)
}

func (svr *GrpcServer) recoveryHandlerFunc(p interface{}) (err error) {
	svr.Errorf("recovered from panic: %v", p)
	return fmt.Errorf("recovered from panic: %v", p)
}

func newGrpcServer() *GrpcServer {
	// server
	svr := &GrpcServer{
		address: utils.GetGrpcServerAddress(),
		Logger:  utils.NewLogger("GrpcServer"),
	}

	// services servers
	svr.NodeSvr = GetNodeServiceServer()
	svr.ModelBaseServiceSvr = GetModelBaseServiceServer()
	svr.TaskSvr = GetTaskServiceServer()
	svr.DependencySvr = GetDependencyServer()
	svr.MetricSvr = GetMetricsServer()
	svr.SyncSvr = GetSyncServiceServer()

	// recovery options
	recoveryOpts := []grpcrecovery.Option{
		grpcrecovery.WithRecoveryHandler(svr.recoveryHandlerFunc),
	}

	// grpc server with keepalive parameters
	// These settings must be compatible with client keepalive parameters
	// to prevent connection timeouts after network disconnection/reconnection
	svr.svr = grpc.NewServer(
		grpcmiddleware.WithUnaryServerChain(
			grpcrecovery.UnaryServerInterceptor(recoveryOpts...),
			grpcauth.UnaryServerInterceptor(middlewares.GetGrpcServerAuthTokenFunc()),
		),
		grpcmiddleware.WithStreamServerChain(
			grpcrecovery.StreamServerInterceptor(recoveryOpts...),
			grpcauth.StreamServerInterceptor(middlewares.GetGrpcServerAuthTokenFunc()),
		),
		// Server-side keepalive enforcement
		grpc.KeepaliveEnforcementPolicy(keepalive.EnforcementPolicy{
			MinTime:             10 * time.Second, // Minimum time clients can send keepalive pings
			PermitWithoutStream: true,             // Allow keepalive pings even without active streams
		}),
		// Server-side keepalive parameters
		grpc.KeepaliveParams(keepalive.ServerParameters{
			Time:    20 * time.Second, // Send keepalive ping if no activity for 20s (matches client)
			Timeout: 5 * time.Second,  // Wait 5s for ping response before closing connection (matches client)
		}),
	)

	// initialize
	svr.Init()

	return svr
}

var _server *GrpcServer
var _serverOnce sync.Once

func GetGrpcServer() *GrpcServer {
	_serverOnce.Do(func() {
		_server = newGrpcServer()
	})
	return _server
}

func NewGrpcServer() *GrpcServer {
	return newGrpcServer()
}
