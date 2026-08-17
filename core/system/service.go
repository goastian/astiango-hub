package system

import (
	"errors"
	"github.com/goastian/astiango-hub/core/interfaces"
	"github.com/goastian/astiango-hub/core/models/models"
	"github.com/goastian/astiango-hub/core/models/service"
	"github.com/goastian/astiango-hub/core/utils"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"sync"
)

type Service struct {
	interfaces.Logger
}

func (svc *Service) Init() (err error) {
	// initialize data
	if err := svc.initData(); err != nil {
		return err
	}

	return nil
}

func (svc *Service) initData() (err error) {
	// initial settings data
	initData := []models.Setting{
		{
			Key: "dependency",
			Value: bson.M{
				"auto_install": true,
			},
		},
	}

	for _, setting := range initData {
		_, err := service.NewModelService[models.Setting]().GetOne(bson.M{"key": setting.Key}, nil)
		if err != nil {
			if !errors.Is(err, mongo.ErrNoDocuments) {
				svc.Errorf("error getting setting: %v", err)
				continue
			}

			// not found, insert
			_, err := service.NewModelService[models.Setting]().InsertOne(setting)
			if err != nil {
				svc.Errorf("error inserting setting: %v", err)
				continue
			}
		}
	}

	return nil
}

func newSystemService() *Service {
	// service
	svc := &Service{
		Logger: utils.NewLogger("SystemService"),
	}

	if err := svc.Init(); err != nil {
		panic(err)
	}

	return svc
}

var _service *Service
var _serviceOnce sync.Once

func GetSystemService() *Service {
	if _service == nil {
		_service = newSystemService()
	}
	_serviceOnce.Do(func() {
		_service = newSystemService()
	})
	return _service
}
