package server

import (
	"context"
	"errors"
	"fmt"
	"github.com/cenkalti/backoff/v4"
	"github.com/goastian/astiango-hub/core/interfaces"
	mongo2 "github.com/goastian/astiango-hub/core/mongo"
	"github.com/goastian/astiango-hub/core/utils"
	"io"
	"sync"
	"time"

	"github.com/goastian/astiango-hub/core/constants"
	"github.com/goastian/astiango-hub/core/models/models"
	"github.com/goastian/astiango-hub/core/models/service"
	"github.com/goastian/astiango-hub/grpc"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type DependencyServiceServer struct {
	grpc.UnimplementedDependencyServiceServer
	mu      *sync.Mutex
	streams map[string]*grpc.DependencyService_ConnectServer
	interfaces.Logger
}

func (svr DependencyServiceServer) Connect(req *grpc.DependencyServiceConnectRequest, stream grpc.DependencyService_ConnectServer) (err error) {
	svr.mu.Lock()
	svr.streams[req.NodeKey] = &stream
	svr.mu.Unlock()
	svr.Info("[DependencyServiceServer] connected: " + req.NodeKey)

	// Keep this scope alive because once this scope exits - the stream is closed
	<-stream.Context().Done()
	svr.Info("[DependencyServiceServer] disconnected: " + req.NodeKey)

	return nil
}

// Sync handles synchronization of dependencies between nodes and the database
func (svr DependencyServiceServer) Sync(_ context.Context, request *grpc.DependencyServiceSyncRequest) (response *grpc.Response, err error) {
	// Get node by node key
	n, err := service.NewModelService[models.Node]().GetOne(bson.M{"key": request.NodeKey}, nil)
	if err != nil {
		return nil, err
	}

	// Get existing dependencies from database for this node and language
	depsDb, err := service.NewModelService[models.Dependency]().GetMany(bson.M{
		"node_id": n.Id,
		"type":    request.Lang,
	}, nil)
	if err != nil {
		if !errors.Is(err, mongo.ErrNoDocuments) {
			svr.Errorf("[DependencyService] get dependencies from db error: %v", err)
			return nil, err
		}
	}

	// Create map of existing dependencies for faster lookup
	depsDbMap := make(map[string]*models.Dependency)
	for _, d := range depsDb {
		depsDbMap[d.Name] = &d
	}

	// Process new dependencies from request
	var depsToInsert []models.Dependency
	var depsToUpdate []models.Dependency
	depsMap := make(map[string]*models.Dependency)
	for _, dep := range request.Dependencies {
		// Create dependency model
		d := models.Dependency{
			Name:    dep.Name,
			NodeId:  n.Id,
			Type:    request.Lang,
			Version: dep.Version,
			Status:  constants.DependencyStatusInstalled,
		}
		d.SetCreatedAt(time.Now())
		d.SetUpdatedAt(time.Now())

		// Add to map
		depsMap[d.Name] = &d

		// Check if dependency exists in DB
		existingDep, ok := depsDbMap[d.Name]
		if !ok {
			// If dependency doesn't exist, add to insert list
			depsToInsert = append(depsToInsert, d)
		} else if existingDep.Version != d.Version || existingDep.Status != constants.DependencyStatusInstalled {
			// If dependency exists but version is different or status is not installed, add to update list
			d.Id = existingDep.Id
			d.SetUpdatedAt(time.Now())
			depsToUpdate = append(depsToUpdate, d)
		}
	}

	// Find dependencies to delete (exist in DB but not in request)
	var depIdsToDelete []primitive.ObjectID
	for _, d := range depsDb {
		_, ok := depsMap[d.Name]
		if !ok {
			// Only delete dependencies that are uninstalled/error/abnormal and older than 7 days
			if d.Status != constants.DependencyStatusInstalled && time.Since(d.UpdatedAt) > 7*24*time.Hour {
				depIdsToDelete = append(depIdsToDelete, d.Id)
			}
		}
	}

	// Run database operations in a transaction
	err = mongo2.RunTransaction(func(ctx mongo.SessionContext) (err error) {
		// Delete old dependencies if any
		if len(depIdsToDelete) > 0 {
			err = service.NewModelService[models.Dependency]().DeleteMany(bson.M{
				"_id": bson.M{"$in": depIdsToDelete},
			})
			if err != nil {
				svr.Errorf("[DependencyServiceServer] delete dependencies in db error: %v", err)
				return err
			}
		}

		// Insert new dependencies if any
		if len(depsToInsert) > 0 {
			_, err = service.NewModelService[models.Dependency]().InsertMany(depsToInsert)
			if err != nil {
				svr.Errorf("[DependencyServiceServer] insert dependencies in db error: %v", err)
				return err
			}
		}

		// Update dependencies with different versions
		for _, d := range depsToUpdate {
			err = service.NewModelService[models.Dependency]().ReplaceById(d.Id, d)
			if err != nil {
				svr.Errorf("[DependencyServiceServer] update dependency in db error: %v", err)
				return err
			}
		}

		return nil
	})

	return nil, err
}

func (svr DependencyServiceServer) UpdateLogs(stream grpc.DependencyService_UpdateLogsServer) (err error) {
	for {
		// receive message
		req, err := stream.Recv()
		if err == io.EOF {
			// all messages have been received
			return stream.SendAndClose(&grpc.Response{Message: "update task log finished"})
		}
		if err != nil {
			return err
		}

		// get id
		id, err := primitive.ObjectIDFromHex(req.TargetId)
		if err != nil {
			svr.Errorf("[DependencyServiceServer] convert dependency id error: %v", err)
			return err
		}

		// insert dependency logs
		var depLogs []models.DependencyLog
		for _, line := range req.Logs {
			depLog := models.DependencyLog{
				TargetId: id,
				Content:  line,
			}
			depLogs = append(depLogs, depLog)
		}
		_, err = service.NewModelService[models.DependencyLog]().InsertMany(depLogs)
		if err != nil {
			svr.Errorf("[DependencyServiceServer] insert dependency logs error: %v", err)
			return err
		}
	}
}

func (svr DependencyServiceServer) SyncConfigSetup(_ context.Context, request *grpc.DependencyServiceSyncConfigSetupRequest) (response *grpc.Response, err error) {
	// Get node by node key
	n, err := service.NewModelService[models.Node]().GetOne(bson.M{"key": request.NodeKey}, nil)
	if err != nil {
		return nil, err
	}

	// Get config
	cfg, err := service.NewModelService[models.DependencyConfig]().GetOne(bson.M{"key": request.Lang}, nil)
	if err != nil {
		return nil, err
	}

	// Get config setup for the node
	cs, err := service.NewModelService[models.DependencyConfigSetup]().GetOne(bson.M{
		"node_id":              n.Id,
		"dependency_config_id": cfg.Id,
	}, nil)
	if err != nil {
		if !errors.Is(err, mongo.ErrNoDocuments) {
			svr.Errorf("[DependencyService] get dependency config setup from db error: %v", err)
			return nil, err
		}
	}

	// drivers
	var drivers []models.DependencyDriver
	if request.Drivers != nil && len(request.Drivers) > 0 {
		for _, d := range request.Drivers {
			drivers = append(drivers, models.DependencyDriver{
				Name:    d.Name,
				Version: d.Version,
			})
		}
	}

	if cs == nil {
		// Create new config setup
		cs = &models.DependencyConfigSetup{
			NodeId:             n.Id,
			DependencyConfigId: cfg.Id,
			Status:             request.Status,
			Error:              request.Error,
			Version:            request.Version,
			Drivers:            drivers,
		}
		_, err = service.NewModelService[models.DependencyConfigSetup]().InsertOne(*cs)
		if err != nil {
			svr.Errorf("[DependencyService] insert dependency config setup error: %v", err)
			return nil, err
		}
	} else {
		// Update existing config setup
		if cs.Status == constants.DependencyStatusUninstalled || request.Status == constants.DependencyStatusInstalled {
			cs.Status = request.Status
		}
		cs.Error = request.Error
		cs.Version = request.Version
		cs.Drivers = drivers
		err = service.NewModelService[models.DependencyConfigSetup]().ReplaceById(cs.Id, *cs)
		if err != nil {
			svr.Errorf("[DependencyService] update dependency config setup error: %v", err)
			return nil, err
		}
	}
	return nil, nil
}

func (svr DependencyServiceServer) GetStream(nodeKey string) (stream *grpc.DependencyService_ConnectServer, err error) {
	b := backoff.WithMaxRetries(backoff.NewConstantBackOff(1*time.Second), 30)
	err = backoff.Retry(func() error {
		stream, err = svr.getStream(nodeKey)
		return err
	}, b)
	if err != nil {
		svr.Errorf("get stream error: %v", err)
		return nil, err
	}
	return stream, nil
}

func (svr DependencyServiceServer) getStream(nodeKey string) (stream *grpc.DependencyService_ConnectServer, err error) {
	svr.mu.Lock()
	defer svr.mu.Unlock()
	stream, ok := svr.streams[nodeKey]
	if !ok {
		return nil, fmt.Errorf("stream not found for node: %s", nodeKey)
	}
	return stream, nil
}

func newDependencyServer() *DependencyServiceServer {
	return &DependencyServiceServer{
		mu:      new(sync.Mutex),
		streams: make(map[string]*grpc.DependencyService_ConnectServer),
		Logger:  utils.NewLogger("DependencyServiceServer"),
	}
}

var depSvc *DependencyServiceServer
var depSvcOnce sync.Once

func GetDependencyServer() *DependencyServiceServer {
	depSvcOnce.Do(func() {
		depSvc = newDependencyServer()
	})
	return depSvc
}
