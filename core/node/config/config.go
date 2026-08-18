package config

import (
	"github.com/goastian/astiango-hub/core/entity"
	"github.com/goastian/astiango-hub/core/utils"
)

func newConfig() (cfg *entity.NodeInfo) {
	authKey, err := utils.NewSecret()
	if err != nil {
		panic(err)
	}
	return &entity.NodeInfo{
		Key:        utils.GetNodeKey(),
		Name:       utils.GetNodeName(),
		IsMaster:   utils.IsMaster(),
		MaxRunners: utils.GetNodeMaxRunners(),
		AuthKey:    authKey,
	}
}
