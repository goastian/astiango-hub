package server

import (
	"context"
	"sync"
	"time"

	"github.com/goastian/astiango-hub/core/constants"
	"github.com/goastian/astiango-hub/core/errors"
	"github.com/goastian/astiango-hub/core/grpc/middlewares"
	"github.com/goastian/astiango-hub/core/interfaces"
	"github.com/goastian/astiango-hub/core/models/models"
	"github.com/goastian/astiango-hub/core/models/service"
	nodeconfig "github.com/goastian/astiango-hub/core/node/config"
	"github.com/goastian/astiango-hub/core/notification"
	"github.com/goastian/astiango-hub/core/utils"
	"github.com/goastian/astiango-hub/grpc"
	errors2 "github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"google.golang.org/grpc/metadata"
)

var nodeServiceMutex = sync.Mutex{}

type NodeServiceServer struct {
	grpc.UnimplementedNodeServiceServer

	// dependencies
	cfgSvc interfaces.NodeConfigService

	// internals
	subs map[primitive.ObjectID]grpc.NodeService_SubscribeServer
	interfaces.Logger
}

// Register from handler/worker to master
func (svr NodeServiceServer) Register(ctx context.Context, req *grpc.NodeServiceRegisterRequest) (res *grpc.Response, err error) {
	// node key
	if req.NodeKey == "" {
		return HandleError(errors.ErrorModelMissingRequiredData)
	}
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok || len(md.Get(middlewares.GrpcHeaderNodeKey)) != 1 || len(md.Get(middlewares.GrpcHeaderNodeSecret)) != 1 || md.Get(middlewares.GrpcHeaderNodeKey)[0] != req.NodeKey {
		return HandleError(errors.ErrorGrpcUnauthorized)
	}
	syncSecret := md.Get(middlewares.GrpcHeaderNodeSecret)[0]

	// find in db
	var node *models.Node
	node, err = service.NewModelService[models.Node]().GetOne(bson.M{"key": req.NodeKey}, nil)
	if err == nil {
		if node.SyncKeyHash != "" {
			valid, _, verifyErr := utils.VerifyPassword(syncSecret, node.SyncKeyHash)
			if verifyErr != nil || !valid {
				return HandleError(errors.ErrorGrpcUnauthorized)
			}
		} else {
			node.SyncKeyHash, err = utils.HashPassword(syncSecret)
			if err != nil {
				return HandleError(err)
			}
		}
		// register existing
		node.Status = constants.NodeStatusOnline
		node.Active = true
		node.ActiveAt = time.Now()
		err = service.NewModelService[models.Node]().ReplaceById(node.Id, *node)
		if err != nil {
			return HandleError(err)
		}
		svr.Infof("updated worker[%s] in db. id: %s", req.NodeKey, node.Id.Hex())
	} else if errors2.Is(err, mongo.ErrNoDocuments) {
		// register new
		node = &models.Node{
			Key:        req.NodeKey,
			Name:       req.NodeName,
			Status:     constants.NodeStatusOnline,
			Active:     true,
			ActiveAt:   time.Now(),
			Enabled:    true,
			MaxRunners: int(req.MaxRunners),
		}
		node.SyncKeyHash, err = utils.HashPassword(syncSecret)
		if err != nil {
			return HandleError(err)
		}
		node.SetCreated(primitive.NilObjectID)
		node.SetUpdated(primitive.NilObjectID)
		node.Id, err = service.NewModelService[models.Node]().InsertOne(*node)
		if err != nil {
			return HandleError(err)
		}
		svr.Infof("added worker[%s] in db. id: %s", req.NodeKey, node.Id.Hex())
	} else {
		// error
		return HandleError(err)
	}

	svr.Infof("master registered worker[%s]", req.NodeKey)

	return HandleSuccessWithData(node)
}

// SendHeartbeat from worker to master
func (svr NodeServiceServer) SendHeartbeat(_ context.Context, req *grpc.NodeServiceSendHeartbeatRequest) (res *grpc.Response, err error) {
	// find in db
	node, err := service.NewModelService[models.Node]().GetOne(bson.M{"key": req.NodeKey}, nil)
	if err != nil {
		if errors2.Is(err, mongo.ErrNoDocuments) {
			return HandleError(errors.ErrorNodeNotExists)
		}
		return HandleError(err)
	}
	oldStatus := node.Status

	// update status
	node.Status = constants.NodeStatusOnline
	node.Active = true
	node.ActiveAt = time.Now()
	err = service.NewModelService[models.Node]().ReplaceById(node.Id, *node)
	if err != nil {
		return HandleError(err)
	}
	newStatus := node.Status

	// send notification if status changed
	if utils.IsPro() {
		if oldStatus != newStatus {
			go notification.GetNotificationService().SendNodeNotification(node)
		}
	}

	return HandleSuccessWithData(node)
}

func (svr NodeServiceServer) Subscribe(request *grpc.NodeServiceSubscribeRequest, stream grpc.NodeService_SubscribeServer) (err error) {
	svr.Infof("master received subscribe request from node[%s]", request.NodeKey)

	// find in db
	node, err := service.NewModelService[models.Node]().GetOne(bson.M{"key": request.NodeKey}, nil)
	if err != nil {
		svr.Errorf("error getting node: %v", err)
		return err
	}

	// Optimistically mark node as active when subscription succeeds
	// This provides fast recovery after reconnection while master monitor
	// will verify health and revert if the connection is unstable
	oldStatus := node.Status
	node.Status = constants.NodeStatusOnline
	node.Active = true
	node.ActiveAt = time.Now()
	err = service.NewModelService[models.Node]().ReplaceById(node.Id, *node)
	if err != nil {
		svr.Warnf("failed to update node status on subscribe: %v", err)
		// Continue anyway - monitor will fix it
	} else {
		svr.Debugf("marked node[%s] as active on subscription (status: %s -> %s)",
			request.NodeKey, oldStatus, node.Status)
	}

	// subscribe
	nodeServiceMutex.Lock()
	svr.subs[node.Id] = stream
	nodeServiceMutex.Unlock()

	// send notification if status changed
	if utils.IsPro() && oldStatus != node.Status {
		go notification.GetNotificationService().SendNodeNotification(node)
	}

	// wait for stream to close
	<-stream.Context().Done()

	// unsubscribe
	nodeServiceMutex.Lock()
	delete(svr.subs, node.Id)
	nodeServiceMutex.Unlock()
	svr.Infof("master unsubscribed from node[%s]", request.NodeKey)

	return nil
}

func (svr NodeServiceServer) GetSubscribeStream(nodeId primitive.ObjectID) (stream grpc.NodeService_SubscribeServer, ok bool) {
	nodeServiceMutex.Lock()
	defer nodeServiceMutex.Unlock()
	stream, ok = svr.subs[nodeId]
	return stream, ok
}

func newNodeServiceServer() *NodeServiceServer {
	return &NodeServiceServer{
		cfgSvc: nodeconfig.GetNodeConfigService(),
		subs:   make(map[primitive.ObjectID]grpc.NodeService_SubscribeServer),
		Logger: utils.NewLogger("GrpcNodeServiceServer"),
	}
}

var nodeSvr *NodeServiceServer
var nodeSvrOnce sync.Once

func GetNodeServiceServer() *NodeServiceServer {
	nodeSvrOnce.Do(func() {
		nodeSvr = newNodeServiceServer()
	})
	return nodeSvr
}
