package config

import (
	"encoding/json"
	"github.com/goastian/astiango-hub/core/entity"
	"github.com/goastian/astiango-hub/core/interfaces"
	"github.com/goastian/astiango-hub/core/utils"
	"os"
	"path/filepath"
	"sync"
)

type Service struct {
	cfg *entity.NodeInfo
	interfaces.Logger
}

func (svc *Service) Init() (err error) {
	metadataConfigPath := utils.GetMetadataConfigPath()

	// check config directory path
	configDirPath := filepath.Dir(metadataConfigPath)
	if !utils.Exists(configDirPath) {
		if err := os.MkdirAll(configDirPath, os.FileMode(0700)); err != nil {
			svc.Errorf("create config directory error: %v", err)
			return err
		}
	}

	if !utils.Exists(metadataConfigPath) {
		// not exists, set to default config, and create a config file for persistence
		svc.cfg = newConfig()
		data, err := json.Marshal(svc.cfg)
		if err != nil {
			svc.Errorf("marshal config error: %v", err)
			return err
		}
		if err := os.WriteFile(metadataConfigPath, data, os.FileMode(0600)); err != nil {
			svc.Errorf("write config file error: %v", err)
			return err
		}
	} else {
		// exists, read and set to config
		data, err := os.ReadFile(metadataConfigPath)
		if err != nil {
			svc.Errorf("read config file error: %v", err)
			return err
		}
		if err := json.Unmarshal(data, svc.cfg); err != nil {
			svc.Errorf("unmarshal config error: %v", err)
			return err
		}
	}
	if svc.cfg.AuthKey == "" {
		secret, err := utils.NewSecret()
		if err != nil {
			return err
		}
		svc.cfg.AuthKey = secret
		data, err := json.Marshal(svc.cfg)
		if err != nil {
			return err
		}
		if err := os.WriteFile(metadataConfigPath, data, os.FileMode(0600)); err != nil {
			return err
		}
	}

	return nil
}

func (svc *Service) Reload() (err error) {
	return svc.Init()
}

func (svc *Service) GetBasicNodeInfo() (res interfaces.Entity) {
	res = &entity.NodeInfo{
		Key:        svc.GetNodeKey(),
		Name:       svc.GetNodeName(),
		IsMaster:   svc.IsMaster(),
		AuthKey:    svc.GetAuthKey(),
		MaxRunners: svc.GetMaxRunners(),
	}
	return res
}

func (svc *Service) GetNodeKey() (res string) {
	return svc.cfg.Key
}

func (svc *Service) GetNodeName() (res string) {
	return svc.cfg.Name
}

func (svc *Service) IsMaster() (res bool) {
	return svc.cfg.IsMaster
}

func (svc *Service) GetAuthKey() (res string) {
	return svc.cfg.AuthKey
}

func (svc *Service) GetMaxRunners() (res int) {
	return svc.cfg.MaxRunners
}

func newNodeConfigService() (svc2 interfaces.NodeConfigService, err error) {
	// config service
	svc := &Service{
		cfg:    newConfig(),
		Logger: utils.NewLogger("NodeConfigService"),
	}

	// init
	if err := svc.Init(); err != nil {
		return nil, err
	}

	return svc, nil
}

var _service interfaces.NodeConfigService
var _serviceOnce sync.Once

func GetNodeConfigService() interfaces.NodeConfigService {
	_serviceOnce.Do(func() {
		var err error
		_service, err = newNodeConfigService()
		if err != nil {
			panic(err)
		}
	})
	return _service
}
