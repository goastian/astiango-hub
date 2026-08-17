package service

import (
	"errors"
	"sync"
	"time"

	"github.com/goastian/astiango-hub/core/constants"
	"github.com/goastian/astiango-hub/core/interfaces"
	"github.com/goastian/astiango-hub/core/models/models"
	"github.com/goastian/astiango-hub/core/models/service"
	"github.com/goastian/astiango-hub/core/utils"
	"go.mongodb.org/mongo-driver/bson"
	mongo2 "go.mongodb.org/mongo-driver/mongo"
)

// NodeMonitoringService handles monitoring of worker nodes
type NodeMonitoringService struct {
	cfgSvc interfaces.NodeConfigService
	interfaces.Logger
}

// GetAllWorkerNodes returns all active worker nodes (excluding the master node)
func (svc *NodeMonitoringService) GetAllWorkerNodes() (nodes []models.Node, err error) {
	query := bson.M{
		"key":    bson.M{"$ne": svc.cfgSvc.GetNodeKey()}, // not self
		"active": true,                                   // active
	}
	nodes, err = service.NewModelService[models.Node]().GetMany(query, nil)
	if err != nil {
		if errors.Is(err, mongo2.ErrNoDocuments) {
			return nil, nil
		}
		svc.Errorf("get all worker nodes error: %v", err)
		return nil, err
	}
	return nodes, nil
}

// UpdateMasterNodeStatus updates the master node status in the database
func (svc *NodeMonitoringService) UpdateMasterNodeStatus() (oldStatus, newStatus string, err error) {
	nodeKey := svc.cfgSvc.GetNodeKey()
	node, err := service.NewModelService[models.Node]().GetOne(bson.M{"key": nodeKey}, nil)
	if err != nil {
		return "", "", err
	}
	oldStatus = node.Status

	node.Status = constants.NodeStatusOnline
	node.Active = true
	node.ActiveAt = time.Now()
	newStatus = node.Status

	err = service.NewModelService[models.Node]().ReplaceById(node.Id, *node)
	if err != nil {
		return oldStatus, newStatus, err
	}

	return oldStatus, newStatus, nil
}

// UpdateNodeRunners updates the current runners count for a specific node
func (svc *NodeMonitoringService) UpdateNodeRunners(node *models.Node) (err error) {
	query := bson.M{
		"node_id": node.Id,
		"status":  constants.TaskStatusRunning,
	}
	runningTasksCount, err := service.NewModelService[models.Task]().Count(query)
	if err != nil {
		svc.Errorf("failed to count running tasks for node[%s]: %v", node.Key, err)
		return err
	}
	node.CurrentRunners = runningTasksCount
	err = service.NewModelService[models.Node]().ReplaceById(node.Id, *node)
	if err != nil {
		svc.Errorf("failed to update node runners for node[%s]: %v", node.Key, err)
		return err
	}
	return nil
}

func NewNodeMonitoringService(cfgSvc interfaces.NodeConfigService) *NodeMonitoringService {
	return &NodeMonitoringService{
		cfgSvc: cfgSvc,
		Logger: utils.NewLogger("NodeMonitoringService"),
	}
}

// Singleton pattern
var nodeMonitoringService *NodeMonitoringService
var nodeMonitoringServiceOnce sync.Once

func GetNodeMonitoringService() *NodeMonitoringService {
	nodeMonitoringServiceOnce.Do(func() {
		nodeMonitoringService = NewNodeMonitoringService(nil) // Will be set by the master service
	})
	return nodeMonitoringService
}
