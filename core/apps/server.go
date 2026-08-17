package apps

import (
	"fmt"
	"net/http"
	_ "net/http/pprof"
	"sync"

	grpcserver "github.com/goastian/astiango-hub/core/grpc/server"
	"github.com/goastian/astiango-hub/core/interfaces"
	"github.com/goastian/astiango-hub/core/node/service"
	"github.com/goastian/astiango-hub/core/utils"
	"github.com/spf13/viper"
)

type Server struct {
	// modules
	nodeSvc interfaces.NodeService
	api     *Api

	// internals
	interfaces.Logger
}

func (app *Server) Init() {
	// log node info
	app.logNodeInfo()

	// pprof
	app.initPprof()
}

func (app *Server) Start() {
	if utils.IsMaster() {
		// Workers connect to the master through gRPC. Start and register the
		// gRPC server before starting the node service so they can register as
		// soon as the master becomes healthy.
		grpcSvr := grpcserver.GetGrpcServer()
		if err := grpcSvr.Start(); err != nil {
			panic(fmt.Errorf("start gRPC server: %w", err))
		}

		// start api
		go start(app.api)
	}

	// start node service
	go app.nodeSvc.Start()
}

func (app *Server) Wait() {
	utils.DefaultWait()
}

func (app *Server) Stop() {
	app.api.Stop()
}

func (app *Server) GetApi() *Api {
	return app.api
}

func (app *Server) GetNodeService() interfaces.NodeService {
	return app.nodeSvc
}

func (app *Server) logNodeInfo() {
	app.Infof("current node type: %s", utils.GetNodeType())
}

func (app *Server) initPprof() {
	if viper.GetBool("pprof") {
		go func() {
			fmt.Println(http.ListenAndServe("0.0.0.0:6060", nil))
		}()
	}
}

func newServer() App {
	// server
	svr := &Server{
		Logger: utils.NewLogger("Server"),
	}

	// master modules
	if utils.IsMaster() {
		// api
		svr.api = GetApi()
	}

	// node service
	if utils.IsMaster() {
		svr.nodeSvc = service.GetMasterService()
	} else {
		svr.nodeSvc = service.GetWorkerService()
	}

	return svr
}

var server App
var serverOnce sync.Once

func GetServer() App {
	serverOnce.Do(func() {
		server = newServer()
	})
	return server
}
