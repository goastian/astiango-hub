package mongo

import (
	"github.com/goastian/astiango-hub/core/config"
	"github.com/spf13/viper"
	"go.mongodb.org/mongo-driver/mongo"
)

func init() {
	config.InitConfig()
}

func GetMongoDb(dbName string) *mongo.Database {
	// Use default database name if not provided
	if dbName == "" {
		if dbName = viper.GetString("mongo.db"); dbName == "" {
			dbName = "test"
		}
	}

	c, err := GetMongoClient()
	if err != nil {
		logger.Errorf("error getting mongo client: %v", err)
		return nil
	}

	return c.Database(dbName)
}
