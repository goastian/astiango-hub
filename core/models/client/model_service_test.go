package client_test

import (
	"context"
	client2 "github.com/goastian/astiango-hub/core/grpc/client"
	"github.com/goastian/astiango-hub/core/models/models"
	"github.com/goastian/astiango-hub/core/mongo"
	"testing"
	"time"

	"github.com/apex/log"
	"github.com/goastian/astiango-hub/core/grpc/server"
	"github.com/goastian/astiango-hub/core/models/client"
	"github.com/goastian/astiango-hub/core/models/service"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson"
)

func setupTestDB() {
	viper.Set("mongo.db", "testdb")
}

func teardownTestDB() {
	db := mongo.GetMongoDb("testdb")
	db.Drop(context.Background())
}

func startSvr(svr *server.GrpcServer) {
	err := svr.Start()
	if err != nil {
		log.Errorf("failed to start grpc server: %v", err)
	}
}

func stopSvr(svr *server.GrpcServer) {
	err := svr.Stop()
	if err != nil {
		log.Errorf("failed to stop grpc server: %v", err)
	}
}

func TestModelService_GetById(t *testing.T) {
	setupTestDB()
	defer teardownTestDB()
	svr := server.NewGrpcServer()
	go startSvr(svr)
	defer stopSvr(svr)

	m := models.TestModel{
		Name: "Test Name",
	}
	modelSvc := service.NewModelService[models.TestModel]()
	id, err := modelSvc.InsertOne(m)
	require.Nil(t, err)
	m.SetId(id)
	time.Sleep(100 * time.Millisecond)

	client2.GetGrpcClient().Start()

	clientSvc := client.NewModelService[models.TestModel]()
	res, err := clientSvc.GetById(m.Id)
	require.Nil(t, err)
	assert.Equal(t, res.Id, m.Id)
	assert.Equal(t, res.Name, m.Name)
}

func TestModelService_GetOne(t *testing.T) {
	setupTestDB()
	defer teardownTestDB()
	svr := server.NewGrpcServer()
	go startSvr(svr)
	defer stopSvr(svr)

	m := models.TestModel{
		Name: "Test Name",
	}
	modelSvc := service.NewModelService[models.TestModel]()
	id, err := modelSvc.InsertOne(m)
	require.Nil(t, err)
	m.SetId(id)
	time.Sleep(100 * time.Millisecond)

	client2.GetGrpcClient().Start()

	clientSvc := client.NewModelService[models.TestModel]()
	res, err := clientSvc.GetOne(bson.M{"name": m.Name}, nil)
	require.Nil(t, err)
	assert.Equal(t, res.Id, m.Id)
	assert.Equal(t, res.Name, m.Name)
}

func TestModelService_GetMany(t *testing.T) {
	setupTestDB()
	defer teardownTestDB()
	svr := server.NewGrpcServer()
	go startSvr(svr)
	defer stopSvr(svr)

	m := models.TestModel{
		Name: "Test Name",
	}
	modelSvc := service.NewModelService[models.TestModel]()
	id, err := modelSvc.InsertOne(m)
	require.Nil(t, err)
	m.SetId(id)
	time.Sleep(100 * time.Millisecond)

	client2.GetGrpcClient().Start()

	clientSvc := client.NewModelService[models.TestModel]()
	res, err := clientSvc.GetMany(bson.M{"name": m.Name}, nil)
	require.Nil(t, err)
	assert.Equal(t, len(res), 1)
	assert.Equal(t, res[0].Id, m.Id)
	assert.Equal(t, res[0].Name, m.Name)
}

func TestModelService_DeleteById(t *testing.T) {
	setupTestDB()
	defer teardownTestDB()
	svr := server.NewGrpcServer()
	go startSvr(svr)
	defer stopSvr(svr)

	m := models.TestModel{
		Name: "Test Name",
	}
	modelSvc := service.NewModelService[models.TestModel]()
	id, err := modelSvc.InsertOne(m)
	require.Nil(t, err)
	m.SetId(id)
	time.Sleep(100 * time.Millisecond)

	client2.GetGrpcClient().Start()

	clientSvc := client.NewModelService[models.TestModel]()
	err = clientSvc.DeleteById(m.Id)
	require.Nil(t, err)

	res, err := clientSvc.GetById(m.Id)
	assert.NotNil(t, err)
	require.Nil(t, res)
}

func TestModelService_DeleteOne(t *testing.T) {
	setupTestDB()
	defer teardownTestDB()
	svr := server.NewGrpcServer()
	go startSvr(svr)
	defer stopSvr(svr)

	m := models.TestModel{
		Name: "Test Name",
	}
	modelSvc := service.NewModelService[models.TestModel]()
	id, err := modelSvc.InsertOne(m)
	require.Nil(t, err)
	m.SetId(id)
	time.Sleep(100 * time.Millisecond)

	client2.GetGrpcClient().Start()

	clientSvc := client.NewModelService[models.TestModel]()
	err = clientSvc.DeleteOne(bson.M{"name": m.Name})
	require.Nil(t, err)

	res, err := clientSvc.GetOne(bson.M{"name": m.Name}, nil)
	assert.NotNil(t, err)
	require.Nil(t, res)
}

func TestModelService_DeleteMany(t *testing.T) {
	setupTestDB()
	defer teardownTestDB()
	svr := server.NewGrpcServer()
	go startSvr(svr)
	defer stopSvr(svr)

	m := models.TestModel{
		Name: "Test Name",
	}
	modelSvc := service.NewModelService[models.TestModel]()
	id, err := modelSvc.InsertOne(m)
	require.Nil(t, err)
	m.SetId(id)
	time.Sleep(100 * time.Millisecond)

	client2.GetGrpcClient().Start()

	clientSvc := client.NewModelService[models.TestModel]()
	err = clientSvc.DeleteMany(bson.M{"name": m.Name})
	require.Nil(t, err)

	res, err := clientSvc.GetMany(bson.M{"name": m.Name}, nil)
	require.Nil(t, err)
	assert.Equal(t, len(res), 0)
}

func TestModelService_UpdateById(t *testing.T) {
	setupTestDB()
	defer teardownTestDB()
	svr := server.NewGrpcServer()
	go startSvr(svr)
	defer stopSvr(svr)

	m := models.TestModel{
		Name: "Test Name",
	}
	modelSvc := service.NewModelService[models.TestModel]()
	id, err := modelSvc.InsertOne(m)
	require.Nil(t, err)
	m.SetId(id)
	time.Sleep(100 * time.Millisecond)

	client2.GetGrpcClient().Start()

	clientSvc := client.NewModelService[models.TestModel]()
	err = clientSvc.UpdateById(m.Id, bson.M{"$set": bson.M{"name": "New Name"}})
	require.Nil(t, err)

	res, err := clientSvc.GetById(m.Id)
	require.Nil(t, err)
	assert.Equal(t, res.Name, "New Name")
}

func TestModelService_UpdateOne(t *testing.T) {
	setupTestDB()
	defer teardownTestDB()
	svr := server.NewGrpcServer()
	go startSvr(svr)
	defer stopSvr(svr)

	m := models.TestModel{
		Name: "Test Name",
	}
	modelSvc := service.NewModelService[models.TestModel]()
	id, err := modelSvc.InsertOne(m)
	require.Nil(t, err)
	m.SetId(id)
	time.Sleep(100 * time.Millisecond)

	client2.GetGrpcClient().Start()

	clientSvc := client.NewModelService[models.TestModel]()
	err = clientSvc.UpdateOne(bson.M{"name": m.Name}, bson.M{"$set": bson.M{"name": "New Name"}})
	require.Nil(t, err)

	res, err := clientSvc.GetOne(bson.M{"name": "New Name"}, nil)
	require.Nil(t, err)
	assert.Equal(t, res.Name, "New Name")
}

func TestModelService_UpdateMany(t *testing.T) {
	setupTestDB()
	defer teardownTestDB()
	svr := server.NewGrpcServer()
	go startSvr(svr)
	defer stopSvr(svr)

	m1 := models.TestModel{
		Name: "Test Name",
	}
	m2 := models.TestModel{
		Name: "Test Name",
	}
	modelSvc := service.NewModelService[models.TestModel]()
	_, err := modelSvc.InsertOne(m1)
	require.Nil(t, err)
	_, err = modelSvc.InsertOne(m2)
	require.Nil(t, err)
	time.Sleep(100 * time.Millisecond)

	client2.GetGrpcClient().Start()

	clientSvc := client.NewModelService[models.TestModel]()
	err = clientSvc.UpdateMany(bson.M{"name": "Test Name"}, bson.M{"$set": bson.M{"name": "New Name"}})
	require.Nil(t, err)

	res, err := clientSvc.GetMany(bson.M{"name": "New Name"}, nil)
	require.Nil(t, err)
	assert.Equal(t, len(res), 2)
}

func TestModelService_ReplaceById(t *testing.T) {
	setupTestDB()
	defer teardownTestDB()
	svr := server.NewGrpcServer()
	go startSvr(svr)
	defer stopSvr(svr)

	m := models.TestModel{
		Name: "Test Name",
	}
	modelSvc := service.NewModelService[models.TestModel]()
	id, err := modelSvc.InsertOne(m)
	require.Nil(t, err)
	m.SetId(id)
	time.Sleep(100 * time.Millisecond)

	client2.GetGrpcClient().Start()

	clientSvc := client.NewModelService[models.TestModel]()
	m.Name = "New Name"
	err = clientSvc.ReplaceById(m.Id, m)
	require.Nil(t, err)

	res, err := clientSvc.GetById(m.Id)
	require.Nil(t, err)
	assert.Equal(t, res.Name, "New Name")
}

func TestModelService_ReplaceOne(t *testing.T) {
	setupTestDB()
	defer teardownTestDB()
	svr := server.NewGrpcServer()
	go startSvr(svr)
	defer stopSvr(svr)

	m := models.TestModel{
		Name: "Test Name",
	}
	modelSvc := service.NewModelService[models.TestModel]()
	id, err := modelSvc.InsertOne(m)
	require.Nil(t, err)
	m.SetId(id)
	time.Sleep(100 * time.Millisecond)

	client2.GetGrpcClient().Start()

	clientSvc := client.NewModelService[models.TestModel]()
	m.Name = "New Name"
	err = clientSvc.ReplaceOne(bson.M{"name": "Test Name"}, m)
	require.Nil(t, err)

	res, err := clientSvc.GetOne(bson.M{"name": "New Name"}, nil)
	require.Nil(t, err)
	assert.Equal(t, res.Name, "New Name")
}

func TestModelService_InsertOne(t *testing.T) {
	setupTestDB()
	defer teardownTestDB()
	svr := server.NewGrpcServer()
	go startSvr(svr)
	defer stopSvr(svr)

	client2.GetGrpcClient().Start()

	clientSvc := client.NewModelService[models.TestModel]()
	m := models.TestModel{
		Name: "Test Name",
	}
	id, err := clientSvc.InsertOne(m)
	require.Nil(t, err)

	res, err := clientSvc.GetById(id)
	require.Nil(t, err)
	assert.Equal(t, res.Name, m.Name)
}

func TestModelService_InsertMany(t *testing.T) {
	setupTestDB()
	defer teardownTestDB()
	svr := server.NewGrpcServer()
	go startSvr(svr)
	defer stopSvr(svr)

	client2.GetGrpcClient().Start()

	clientSvc := client.NewModelService[models.TestModel]()
	testModels := []models.TestModel{
		{Name: "Test Name 1"},
		{Name: "Test Name 2"},
	}
	ids, err := clientSvc.InsertMany(testModels)
	require.Nil(t, err)

	for i, id := range ids {
		res, err := clientSvc.GetById(id)
		require.Nil(t, err)
		assert.Equal(t, res.Name, testModels[i].Name)
	}
}

func TestModelService_Count(t *testing.T) {
	setupTestDB()
	defer teardownTestDB()
	svr := server.NewGrpcServer()
	go startSvr(svr)
	defer stopSvr(svr)

	modelSvc := service.NewModelService[models.TestModel]()
	for i := 0; i < 5; i++ {
		_, err := modelSvc.InsertOne(models.TestModel{
			Name: "Test Name",
		})
		require.Nil(t, err)
	}
	time.Sleep(100 * time.Millisecond)

	client2.GetGrpcClient().Start()

	clientSvc := client.NewModelService[models.TestModel]()
	count, err := clientSvc.Count(bson.M{})
	require.Nil(t, err)

	assert.Equal(t, count, 5)
}
