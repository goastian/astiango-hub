package stats

import (
	"github.com/goastian/astiango-hub/core/constants"
	"github.com/goastian/astiango-hub/core/database"
	interfaces2 "github.com/goastian/astiango-hub/core/database/interfaces"
	"github.com/goastian/astiango-hub/core/interfaces"
	"github.com/goastian/astiango-hub/core/models/models"
	"github.com/goastian/astiango-hub/core/models/service"
	"github.com/goastian/astiango-hub/core/mongo"
	nodeconfig "github.com/goastian/astiango-hub/core/node/config"
	"github.com/goastian/astiango-hub/core/task/log"
	"github.com/goastian/astiango-hub/core/utils"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"sync"
	"time"
)

type databaseServiceItem struct {
	taskId    primitive.ObjectID
	spiderId  primitive.ObjectID
	dbId      primitive.ObjectID
	dbSvc     interfaces2.DatabaseService
	tableName string
	time      time.Time
}

type Service struct {
	// dependencies
	nodeCfgSvc interfaces.NodeConfigService

	// internals
	mu                   sync.Mutex
	databaseServiceItems map[string]*databaseServiceItem
	databaseServiceTll   time.Duration
	logDriver            log.Driver
	interfaces.Logger
}

func (svc *Service) Init() (err error) {
	go svc.cleanup()
	return nil
}

func (svc *Service) InsertData(taskId primitive.ObjectID, records ...map[string]interface{}) (err error) {
	count := 0

	item, err := svc.getDatabaseServiceItem(taskId)
	if err != nil {
		return err
	}
	dbId := item.dbId
	dbSvc := item.dbSvc
	tableName := item.tableName
	if utils.IsPro() && dbSvc != nil {
		for _, record := range records {
			if err := dbSvc.CreateRow(dbId, "", tableName, svc.normalizeRecord(item, record)); err != nil {
				svc.Errorf("failed to insert data: %v", err)
				continue
			}
			count++
		}
	} else {
		var recordsToInsert []interface{}
		for _, record := range records {
			recordsToInsert = append(recordsToInsert, svc.normalizeRecord(item, record))
		}
		_, err = mongo.GetMongoCol(tableName).InsertMany(recordsToInsert)
		if err != nil {
			svc.Errorf("failed to insert data: %v", err)
			return err
		}
		count = len(records)
	}

	go svc.updateTaskStats(taskId, count)

	return nil
}

func (svc *Service) InsertLogs(id primitive.ObjectID, logs ...string) (err error) {
	return svc.logDriver.WriteLines(id.Hex(), logs)
}

func (svc *Service) getDatabaseServiceItem(taskId primitive.ObjectID) (item *databaseServiceItem, err error) {
	// atomic operation
	svc.mu.Lock()
	defer svc.mu.Unlock()

	// attempt to get from cache
	item, ok := svc.databaseServiceItems[taskId.Hex()]
	if ok {
		// hit in cache
		item.time = time.Now()
		return item, nil
	}

	// task
	t, err := service.NewModelService[models.Task]().GetById(taskId)
	if err != nil {
		return nil, err
	}

	// spider
	s, err := service.NewModelService[models.Spider]().GetById(t.SpiderId)
	if err != nil {
		return nil, err
	}

	// database service
	var dbSvc interfaces2.DatabaseService
	if utils.IsPro() {
		if dbRegSvc := database.GetDatabaseRegistryService(); dbRegSvc != nil {
			dbSvc, err = dbRegSvc.GetDatabaseService(s.DatabaseId)
			if err != nil {
				return nil, err
			}
		}
	}

	// item
	item = &databaseServiceItem{
		taskId:    taskId,
		spiderId:  s.Id,
		dbId:      s.DatabaseId,
		dbSvc:     dbSvc,
		tableName: s.ColName,
		time:      time.Now(),
	}

	// store in cache
	svc.databaseServiceItems[taskId.Hex()] = item

	return item, nil
}

func (svc *Service) updateTaskStats(id primitive.ObjectID, resultCount int) {
	err := service.NewModelService[models.TaskStat]().UpdateById(id, bson.M{
		"$inc": bson.M{
			"result_count": resultCount,
		},
	})
	if err != nil {
		svc.Errorf("failed to update task stats: %v", err)
		return
	}
}

func (svc *Service) cleanup() {
	for {
		// atomic operation
		svc.mu.Lock()

		for k, v := range svc.databaseServiceItems {
			if time.Now().After(v.time.Add(svc.databaseServiceTll)) {
				delete(svc.databaseServiceItems, k)
			}
		}

		svc.mu.Unlock()

		time.Sleep(10 * time.Minute)
	}
}

func (svc *Service) normalizeRecord(item *databaseServiceItem, record map[string]interface{}) (res map[string]interface{}) {
	res = record

	// set task id
	res[constants.TaskKey] = item.taskId

	// set spider id
	res[constants.SpiderKey] = item.spiderId

	return res
}

func NewTaskStatsService() *Service {
	// service
	svc := &Service{
		mu:                   sync.Mutex{},
		databaseServiceItems: map[string]*databaseServiceItem{},
		databaseServiceTll:   10 * time.Minute,
		Logger:               utils.NewLogger("TaskStatsService"),
	}

	svc.nodeCfgSvc = nodeconfig.GetNodeConfigService()

	// log driver
	svc.logDriver = log.GetLogDriver(log.DriverTypeFile)

	return svc
}

var _service *Service
var _serviceOnce sync.Once

func GetTaskStatsService() *Service {
	_serviceOnce.Do(func() {
		_service = NewTaskStatsService()
	})
	return _service
}
