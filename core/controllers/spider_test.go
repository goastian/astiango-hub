package controllers_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/goastian/astiango-hub/core/models/models"
	"github.com/loopfz/gadgeto/tonic"

	"github.com/goastian/astiango-hub/core/controllers"
	"github.com/goastian/astiango-hub/core/middlewares"
	"github.com/goastian/astiango-hub/core/models/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestCreateSpider(t *testing.T) {
	SetupTestDB()
	defer CleanupTestDB()

	gin.SetMode(gin.TestMode)

	router := SetupRouter()
	router.Use(middlewares.AuthorizationMiddleware())
	router.POST("/spiders", nil, tonic.Handler(controllers.PostSpider, 200))

	payload := models.Spider{
		Name:    "Test Spider",
		ColName: "test_spiders",
	}
	requestParams := controllers.PostParams[models.Spider]{
		Data: payload,
	}
	jsonValue, _ := json.Marshal(requestParams)
	req, _ := http.NewRequest("POST", "/spiders", bytes.NewBuffer(jsonValue))
	req.Header.Set("Authorization", TestToken)
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusOK, resp.Code)

	var response controllers.Response[models.Spider]
	err := json.Unmarshal(resp.Body.Bytes(), &response)
	require.Nil(t, err)
	assert.False(t, response.Data.Id.IsZero())
	assert.Equal(t, payload.Name, response.Data.Name)
}

func TestGetSpiderById(t *testing.T) {
	SetupTestDB()
	defer CleanupTestDB()

	gin.SetMode(gin.TestMode)

	router := SetupRouter()
	router.Use(middlewares.AuthorizationMiddleware())
	router.GET("/spiders/:id", nil, tonic.Handler(controllers.GetSpiderById, 200))

	model := models.Spider{
		Name:    "Test Spider",
		ColName: "test_spiders",
	}
	id, err := service.NewModelService[models.Spider]().InsertOne(model)
	require.Nil(t, err)
	ts := models.SpiderStat{}
	ts.SetId(id)
	_, err = service.NewModelService[models.SpiderStat]().InsertOne(ts)
	require.Nil(t, err)

	req, _ := http.NewRequest("GET", "/spiders/"+id.Hex(), nil)
	req.Header.Set("Authorization", TestToken)
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusOK, resp.Code)

	var response controllers.Response[models.Spider]
	err = json.Unmarshal(resp.Body.Bytes(), &response)
	require.Nil(t, err)
	assert.Equal(t, model.Name, response.Data.Name)
}

func TestDeleteSpiderById(t *testing.T) {
	SetupTestDB()
	defer CleanupTestDB()

	gin.SetMode(gin.TestMode)

	router := SetupRouter()
	router.Use(middlewares.AuthorizationMiddleware())
	router.DELETE("/spiders/:id", nil, tonic.Handler(controllers.DeleteSpiderById, 200))

	model := models.Spider{
		Name:    "Test Spider",
		ColName: "test_spiders",
	}
	id, err := service.NewModelService[models.Spider]().InsertOne(model)
	require.Nil(t, err)
	ts := models.SpiderStat{}
	ts.SetId(id)
	_, err = service.NewModelService[models.SpiderStat]().InsertOne(ts)
	require.Nil(t, err)
	task := models.Task{}
	task.SpiderId = id
	taskId, err := service.NewModelService[models.Task]().InsertOne(task)
	require.Nil(t, err)
	taskStat := models.TaskStat{}
	taskStat.SetId(taskId)
	_, err = service.NewModelService[models.TaskStat]().InsertOne(taskStat)
	require.Nil(t, err)

	req, _ := http.NewRequest("DELETE", "/spiders/"+id.Hex(), nil)
	req.Header.Set("Authorization", TestToken)
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusOK, resp.Code)

	_, err = service.NewModelService[models.Spider]().GetById(id)
	assert.NotNil(t, err)
	_, err = service.NewModelService[models.SpiderStat]().GetById(id)
	assert.NotNil(t, err)
	taskCount, err := service.NewModelService[models.Task]().Count(bson.M{"spider_id": id})
	require.Nil(t, err)
	assert.Equal(t, 0, taskCount)
	taskStatCount, err := service.NewModelService[models.TaskStat]().Count(bson.M{"_id": taskId})
	require.Nil(t, err)
	assert.Equal(t, 0, taskStatCount)

}

func TestDeleteSpiderList(t *testing.T) {
	SetupTestDB()
	defer CleanupTestDB()

	gin.SetMode(gin.TestMode)

	router := SetupRouter()
	router.Use(middlewares.AuthorizationMiddleware())
	router.DELETE("/spiders", nil, tonic.Handler(controllers.DeleteSpiderList, 200))

	modelList := []models.Spider{
		{
			Name:    "Test Name 1",
			ColName: "test_spiders",
		}, {
			Name:    "Test Name 2",
			ColName: "test_spiders",
		},
	}
	var ids []primitive.ObjectID
	var taskIds []primitive.ObjectID
	for _, model := range modelList {
		id, err := service.NewModelService[models.Spider]().InsertOne(model)
		require.Nil(t, err)
		ts := models.SpiderStat{}
		ts.SetId(id)
		_, err = service.NewModelService[models.SpiderStat]().InsertOne(ts)
		require.Nil(t, err)
		task := models.Task{}
		task.SpiderId = id
		taskId, err := service.NewModelService[models.Task]().InsertOne(task)
		require.Nil(t, err)
		taskStat := models.TaskStat{}
		taskStat.SetId(taskId)
		_, err = service.NewModelService[models.TaskStat]().InsertOne(taskStat)
		require.Nil(t, err)
		ids = append(ids, id)
		taskIds = append(taskIds, taskId)
	}

	payload := struct {
		Ids []primitive.ObjectID `json:"ids"`
	}{
		Ids: ids,
	}
	jsonValue, _ := json.Marshal(payload)
	req, _ := http.NewRequest("DELETE", "/spiders", bytes.NewBuffer(jsonValue))
	req.Header.Set("Authorization", TestToken)
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusOK, resp.Code)

	spiderCount, err := service.NewModelService[models.Spider]().Count(bson.M{"_id": bson.M{"$in": ids}})
	require.Nil(t, err)
	assert.Equal(t, 0, spiderCount)
	spiderStatCount, err := service.NewModelService[models.SpiderStat]().Count(bson.M{"_id": bson.M{"$in": ids}})
	require.Nil(t, err)
	assert.Equal(t, 0, spiderStatCount)
	taskCount, err := service.NewModelService[models.Task]().Count(bson.M{"_id": bson.M{"$in": taskIds}})
	require.Nil(t, err)
	assert.Equal(t, 0, taskCount)
	taskStatCount, err := service.NewModelService[models.TaskStat]().Count(bson.M{"_id": bson.M{"$in": taskIds}})
	require.Nil(t, err)
	assert.Equal(t, 0, taskStatCount)
}
