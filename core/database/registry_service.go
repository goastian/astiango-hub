package database

import (
	"github.com/goastian/astiango-hub/core/database/interfaces"
)

var serviceInstance interfaces.DatabaseRegistryService

func SetDatabaseRegistryService(svc interfaces.DatabaseRegistryService) {
	serviceInstance = svc
}

func GetDatabaseRegistryService() interfaces.DatabaseRegistryService {
	return serviceInstance
}
