package controllers_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/goastian/astiango-hub/core/constants"
	"github.com/goastian/astiango-hub/core/controllers"
	"github.com/goastian/astiango-hub/core/middlewares"
	"github.com/goastian/astiango-hub/core/models/models"
	"github.com/goastian/astiango-hub/core/models/service"
	"github.com/gin-gonic/gin"
	"github.com/loopfz/gadgeto/tonic"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Helper function to create a test task
func createTestTask(t *testing.T) (task *models.Task, spiderId primitive.ObjectID) {
	// First, create a test spider
	spider := models.Spider{
		Name:    "Test Spider for Task",
		ColName: "test_spider_for_task",
	}
	spiderSvc := service.NewModelService[models.Spider]()
	var err error
	spiderId, err = spiderSvc.InsertOne(spider)
	require.NoError(t, err)
	require.False(t, spiderId.IsZero())

	// Now create a task associated with the spider
	task = &models.Task{
		SpiderId: spiderId,
		Status:   constants.TaskStatusPending,
		Priority: 10,
		Mode:     constants.RunTypeAllNodes,
		Param:    "test param",
		Cmd:      "python main.py",
	}

	// Set timestamps
	now := time.Now()
	task.CreatedAt = now
	task.UpdatedAt = now

	taskSvc := service.NewModelService[models.Task]()
	taskId, err := taskSvc.InsertOne(*task)
	require.NoError(t, err)
	require.False(t, taskId.IsZero())

	task.Id = taskId
	return task, spiderId
}

// Test GetTaskById endpoint
func TestGetTaskById(t *testing.T) {
	SetupTestDB()
	defer CleanupTestDB()

	gin.SetMode(gin.TestMode)

	// Create a test task
	task, _ := createTestTask(t)

	// Set up router
	router := SetupRouter()
	router.Use(middlewares.AuthorizationMiddleware())
	router.GET("/tasks/:id", nil, tonic.Handler(controllers.GetTaskById, 200))

	// Create test request
	req, err := http.NewRequest("GET", "/tasks/"+task.Id.Hex(), nil)
	req.Header.Set("Authorization", TestToken)
	require.NoError(t, err)

	// Execute request
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	// Verify response
	assert.Equal(t, http.StatusOK, resp.Code)

	var response controllers.Response[models.Task]
	err = json.Unmarshal(resp.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.True(t, response.Status == "ok")
	assert.Equal(t, task.Id, response.Data.Id)
	assert.Equal(t, task.SpiderId, response.Data.SpiderId)
	assert.Equal(t, task.Status, response.Data.Status)
}

// Test GetTaskList endpoint
func TestGetTaskList(t *testing.T) {
	SetupTestDB()
	defer CleanupTestDB()

	gin.SetMode(gin.TestMode)

	// Create several test tasks
	task1, _ := createTestTask(t)
	task2, _ := createTestTask(t)
	task2.Status = constants.TaskStatusRunning

	// Use ReplaceById instead of UpdateById with the model
	taskSvc := service.NewModelService[models.Task]()
	err := taskSvc.ReplaceById(task2.Id, *task2)
	require.NoError(t, err)

	// Set up router
	router := SetupRouter()
	router.Use(middlewares.AuthorizationMiddleware())
	router.GET("/tasks", nil, tonic.Handler(controllers.GetTaskList, 200))

	// Create test request
	req, err := http.NewRequest("GET", "/tasks?page=1&size=10", nil)
	req.Header.Set("Authorization", TestToken)
	require.NoError(t, err)

	// Execute request
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	// Verify response
	assert.Equal(t, http.StatusOK, resp.Code)

	var response controllers.ListResponse[models.Task]
	err = json.Unmarshal(resp.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.True(t, response.Status == "ok")
	assert.Equal(t, 2, response.Total) // We created 2 tasks
	assert.Equal(t, 2, len(response.Data))

	// Verify both tasks (including task1) are in the response
	foundTask1 := false
	foundTask2 := false
	for _, task := range response.Data {
		if task.Id == task1.Id {
			foundTask1 = true
			assert.Equal(t, constants.TaskStatusPending, task.Status)
		}
		if task.Id == task2.Id {
			foundTask2 = true
			assert.Equal(t, constants.TaskStatusRunning, task.Status)
		}
	}
	assert.True(t, foundTask1, "task1 should be in the response")
	assert.True(t, foundTask2, "task2 should be in the response")
}

// Test DeleteTaskById endpoint
func TestDeleteTaskById(t *testing.T) {
	SetupTestDB()
	defer CleanupTestDB()

	gin.SetMode(gin.TestMode)

	// Create a test task
	task, _ := createTestTask(t)

	// Set up router
	router := SetupRouter()
	router.Use(middlewares.AuthorizationMiddleware())
	router.DELETE("/tasks/:id", nil, tonic.Handler(controllers.DeleteTaskById, 200))

	// Create test request
	req, err := http.NewRequest("DELETE", "/tasks/"+task.Id.Hex(), nil)
	req.Header.Set("Authorization", TestToken)
	require.NoError(t, err)

	// Execute request
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	// Verify response
	assert.Equal(t, http.StatusOK, resp.Code)

	// Verify task is deleted from database
	taskSvc := service.NewModelService[models.Task]()
	_, err = taskSvc.GetById(task.Id)
	assert.Error(t, err) // Should return error as the task is deleted
}

// Test PostTaskRun endpoint
func TestPostTaskRun(t *testing.T) {
	SetupTestDB()
	defer CleanupTestDB()

	gin.SetMode(gin.TestMode)

	// Create a test spider
	spider := models.Spider{
		Name:    "Test Spider for Run",
		ColName: "test_spider_for_run",
	}
	spiderSvc := service.NewModelService[models.Spider]()
	spiderId, err := spiderSvc.InsertOne(spider)
	require.NoError(t, err)

	// Set up router
	router := SetupRouter()
	router.Use(middlewares.AuthorizationMiddleware())
	router.POST("/tasks/run", nil, tonic.Handler(controllers.PostTaskRun, 200))

	// Create payload
	payload := controllers.PostTaskRunParams{
		SpiderId: spiderId.Hex(),
		Mode:     constants.RunTypeAllNodes,
		Cmd:      "python main.py",
		Param:    "test param",
		Priority: 1,
	}
	jsonValue, _ := json.Marshal(payload)

	// Create test request
	req, err := http.NewRequest("POST", "/tasks/run", bytes.NewBuffer(jsonValue))
	req.Header.Set("Authorization", TestToken)
	req.Header.Set("Content-Type", "application/json")
	require.NoError(t, err)

	// Execute request
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	// Verify response - it may fail if the scheduler service is not properly initialized in test environment
	// This is more of an integration test, so we'll check the status code but not the exact response
	assert.Equal(t, http.StatusOK, resp.Code)
}

// Test PostTaskCancel endpoint
func TestPostTaskCancel(t *testing.T) {
	SetupTestDB()
	defer CleanupTestDB()

	gin.SetMode(gin.TestMode)

	// Create a test task
	task, _ := createTestTask(t)

	// Set status to running to make it cancellable
	task.Status = constants.TaskStatusRunning

	// Use ReplaceById instead of UpdateById with the model
	taskSvc := service.NewModelService[models.Task]()
	err := taskSvc.ReplaceById(task.Id, *task)
	require.NoError(t, err)

	// Set up router
	router := SetupRouter()
	router.Use(middlewares.AuthorizationMiddleware())
	router.POST("/tasks/:id/cancel", nil, tonic.Handler(controllers.PostTaskCancel, 200))

	// Create payload
	payload := controllers.PostTaskCancelParams{
		Force: true,
	}
	jsonValue, _ := json.Marshal(payload)

	// Create test request
	req, err := http.NewRequest("POST", "/tasks/"+task.Id.Hex()+"/cancel", bytes.NewBuffer(jsonValue))
	req.Header.Set("Authorization", TestToken)
	req.Header.Set("Content-Type", "application/json")
	require.NoError(t, err)

	// Execute request
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	// Verify response - it may fail if the scheduler service is not properly initialized in test environment
	// This is more of an integration test, so we'll check the status code but not the exact response
	assert.Equal(t, http.StatusOK, resp.Code)
}

// Test PostTaskRestart endpoint
func TestPostTaskRestart(t *testing.T) {
	SetupTestDB()
	defer CleanupTestDB()

	gin.SetMode(gin.TestMode)

	// Create a test task
	task, _ := createTestTask(t)

	// Set up router
	router := SetupRouter()
	router.Use(middlewares.AuthorizationMiddleware())
	router.POST("/tasks/:id/restart", nil, tonic.Handler(controllers.PostTaskRestart, 200))

	// Create test request
	req, err := http.NewRequest("POST", "/tasks/"+task.Id.Hex()+"/restart", nil)
	req.Header.Set("Authorization", TestToken)
	require.NoError(t, err)

	// Execute request
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	// Verify response - it may fail if the scheduler service is not properly initialized in test environment
	// This is more of an integration test, so we'll check the status code but not the exact response
	assert.Equal(t, http.StatusOK, resp.Code)
}

// Test GetTaskLogs endpoint
func TestGetTaskLogs(t *testing.T) {
	SetupTestDB()
	defer CleanupTestDB()

	gin.SetMode(gin.TestMode)

	// Create a test task
	task, _ := createTestTask(t)

	// Set up router
	router := SetupRouter()
	router.Use(middlewares.AuthorizationMiddleware())
	router.GET("/tasks/:id/logs", nil, tonic.Handler(controllers.GetTaskLogs, 200))

	// Create test request
	req, err := http.NewRequest("GET", "/tasks/"+task.Id.Hex()+"/logs?page=1&size=100", nil)
	req.Header.Set("Authorization", TestToken)
	require.NoError(t, err)

	// Execute request
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	// Verify response
	assert.Equal(t, http.StatusOK, resp.Code)

	var response controllers.ListResponse[string]
	err = json.Unmarshal(resp.Body.Bytes(), &response)
	require.NoError(t, err)

	// Check status is ok - the logs might be empty since we didn't create any,
	// but the endpoint should still function correctly
	assert.Equal(t, "ok", response.Status)
}
