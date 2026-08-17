package admin

import (
	"github.com/goastian/astiango-hub/core/constants"
	"github.com/goastian/astiango-hub/core/interfaces"
	"github.com/goastian/astiango-hub/core/models/models"
	"github.com/goastian/astiango-hub/core/models/service"
	"github.com/goastian/astiango-hub/core/node/config"
	"github.com/goastian/astiango-hub/core/task/scheduler"
	"github.com/goastian/astiango-hub/trace"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"sync"
)

type Service struct {
	// dependencies
	schedulerSvc *scheduler.Service
}

func (svc *Service) Schedule(id primitive.ObjectID, opts *interfaces.SpiderRunOptions) (taskIds []primitive.ObjectID, err error) {
	// spider
	s, err := service.NewModelService[models.Spider]().GetById(id)
	if err != nil {
		return nil, err
	}

	// assign tasks
	return svc.scheduleTasks(s, opts)
}

func (svc *Service) scheduleTasks(s *models.Spider, opts *interfaces.SpiderRunOptions) (taskIds []primitive.ObjectID, err error) {
	// get node ids
	nodeIds, err := svc.getNodeIds(opts)
	if err != nil {
		return nil, err
	}

	// iterate node ids
	for _, nodeId := range nodeIds {
		// task
		t := &models.Task{
			SpiderId:   s.Id,
			NodeId:     nodeId,
			NodeIds:    opts.NodeIds,
			Mode:       opts.Mode,
			Cmd:        opts.Cmd,
			Param:      opts.Param,
			ScheduleId: opts.ScheduleId,
			Priority:   opts.Priority,
		}

		// normalize
		if t.Mode == "" {
			t.Mode = s.Mode
		}
		if t.NodeIds == nil {
			t.NodeIds = s.NodeIds
		}
		if t.Cmd == "" {
			t.Cmd = s.Cmd
		}
		if t.Param == "" {
			t.Param = s.Param
		}
		if t.Priority == 0 {
			t.Priority = s.Priority
		}

		// enqueue task
		t, err = svc.schedulerSvc.Enqueue(t, opts.UserId)
		if err != nil {
			return nil, err
		}

		// append task id
		taskIds = append(taskIds, t.Id)
	}

	return taskIds, nil
}

func (svc *Service) getNodeIds(opts *interfaces.SpiderRunOptions) (nodeIds []primitive.ObjectID, err error) {
	switch opts.Mode {
	case constants.RunTypeAllNodes:
		query := bson.M{
			"active":  true,
			"enabled": true,
			"status":  constants.NodeStatusOnline,
		}
		nodes, err := service.NewModelService[models.Node]().GetMany(query, nil)
		if err != nil {
			return nil, err
		}
		for _, node := range nodes {
			nodeIds = append(nodeIds, node.Id)
		}
	case constants.RunTypeSelectedNodes:
		nodeIds = opts.NodeIds
	default:
		nodeIds = []primitive.ObjectID{primitive.NilObjectID}
	}
	return nodeIds, nil
}

func (svc *Service) isMultiTask(opts *interfaces.SpiderRunOptions) (res bool) {
	if opts.Mode == constants.RunTypeAllNodes {
		query := bson.M{
			"active":  true,
			"enabled": true,
			"status":  constants.NodeStatusOnline,
		}
		nodes, err := service.NewModelService[models.Node]().GetMany(query, nil)
		if err != nil {
			trace.PrintError(err)
			return false
		}
		return len(nodes) > 1
	} else if opts.Mode == constants.RunTypeRandom {
		return false
	} else if opts.Mode == constants.RunTypeSelectedNodes {
		return len(opts.NodeIds) > 1
	} else {
		return false
	}
}

func newSpiderAdminService() *Service {
	nodeCfgSvc := config.GetNodeConfigService()
	// validate node type
	if !nodeCfgSvc.IsMaster() {
		panic("only master node can run spider admin service")
	}
	return &Service{
		schedulerSvc: scheduler.GetTaskSchedulerService(),
	}
}

var _service *Service
var _serviceOnce sync.Once

func GetSpiderAdminService() *Service {
	_serviceOnce.Do(func() {
		_service = newSpiderAdminService()
	})
	return _service
}
