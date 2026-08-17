package controllers_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

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

// Helper function to create a test schedule
func createTestSchedule(t *testing.T) (*models.Schedule, primitive.ObjectID) {
	// First, create a test spider
	spider := models.Spider{
		Name:    "Test Spider for Schedule",
		ColName: "test_spider_for_schedule",
	}
	spiderSvc := service.NewModelService[models.Spider]()
	spiderId, err := spiderSvc.InsertOne(spider)
	require.NoError(t, err)
	require.False(t, spiderId.IsZero())

	// Now create a schedule associated with the spider
	schedule := &models.Schedule{
		Name:     "Test Schedule",
		SpiderId: spiderId,
		Cron:     "0 0 * * *", // Run daily at midnight
		Cmd:      "python main.py",
		Param:    "test param",
		Priority: 5,
		Enabled:  false, // Disabled initially
	}

	// Set timestamps
	now := time.Now()
	schedule.CreatedAt = now
	schedule.UpdatedAt = now

	scheduleSvc := service.NewModelService[models.Schedule]()
	scheduleId, err := scheduleSvc.InsertOne(*schedule)
	require.NoError(t, err)
	require.False(t, scheduleId.IsZero())

	schedule.Id = scheduleId
	return schedule, spiderId
}

// Test PostSchedule endpoint
func TestPostSchedule(t *testing.T) {
	SetupTestDB()
	defer CleanupTestDB()

	gin.SetMode(gin.TestMode)

	// Create a test spider first
	spider := models.Spider{
		Name:    "Test Spider for Schedule Post",
		ColName: "test_spider_for_schedule_post",
	}
	spiderSvc := service.NewModelService[models.Spider]()
	spiderId, err := spiderSvc.InsertOne(spider)
	require.NoError(t, err)

	// Set up router
	router := SetupRouter()
	router.Use(middlewares.AuthorizationMiddleware())
	router.POST("/schedules", nil, tonic.Handler(controllers.PostSchedule, 200))

	// Create payload
	schedule := models.Schedule{
		Name:     "Test Schedule for Post",
		SpiderId: spiderId,
		Cron:     "0 0 * * *",
		Cmd:      "python main.py",
		Param:    "test schedule param",
		Priority: 3,
		Enabled:  false,
	}

	payload := controllers.PostScheduleParams{
		Data: schedule,
	}
	jsonValue, _ := json.Marshal(payload)

	// Create test request
	req, err := http.NewRequest("POST", "/schedules", bytes.NewBuffer(jsonValue))
	req.Header.Set("Authorization", TestToken)
	req.Header.Set("Content-Type", "application/json")
	require.NoError(t, err)

	// Execute request
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	// Verify response
	assert.Equal(t, http.StatusOK, resp.Code)

	var response controllers.Response[models.Schedule]
	err = json.Unmarshal(resp.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "ok", response.Status)
	assert.False(t, response.Data.Id.IsZero())
	assert.Equal(t, schedule.Name, response.Data.Name)
	assert.Equal(t, schedule.SpiderId, response.Data.SpiderId)
	assert.Equal(t, schedule.Cron, response.Data.Cron)
	assert.Equal(t, schedule.Enabled, response.Data.Enabled)
}

// Test GetScheduleById endpoint
func TestGetScheduleById(t *testing.T) {
	SetupTestDB()
	defer CleanupTestDB()

	gin.SetMode(gin.TestMode)

	// Create a test schedule
	schedule, _ := createTestSchedule(t)

	// Set up router
	router := SetupRouter()
	router.Use(middlewares.AuthorizationMiddleware())
	router.GET("/schedules/:id", nil, tonic.Handler(controllers.NewController[models.Schedule]().GetById, 200))

	// Create test request
	req, err := http.NewRequest("GET", "/schedules/"+schedule.Id.Hex(), nil)
	req.Header.Set("Authorization", TestToken)
	require.NoError(t, err)

	// Execute request
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	// Verify response
	assert.Equal(t, http.StatusOK, resp.Code)

	var response controllers.Response[models.Schedule]
	err = json.Unmarshal(resp.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "ok", response.Status)
	assert.Equal(t, schedule.Id, response.Data.Id)
	assert.Equal(t, schedule.Name, response.Data.Name)
	assert.Equal(t, schedule.Cron, response.Data.Cron)
}

// Test PutScheduleById endpoint
func TestPutScheduleById(t *testing.T) {
	SetupTestDB()
	defer CleanupTestDB()

	gin.SetMode(gin.TestMode)

	// Create a test schedule
	schedule, _ := createTestSchedule(t)

	// Set up router
	router := SetupRouter()
	router.Use(middlewares.AuthorizationMiddleware())
	router.PUT("/schedules/:id", nil, tonic.Handler(controllers.PutScheduleById, 200))

	// Update the schedule
	updatedSchedule := *schedule
	updatedSchedule.Name = "Updated Schedule Name"
	updatedSchedule.Cron = "*/5 * * * *" // Every 5 minutes
	updatedSchedule.Enabled = true

	payload := controllers.PutScheduleByIdParams{
		Id:   schedule.Id.Hex(),
		Data: updatedSchedule,
	}
	jsonValue, _ := json.Marshal(payload)

	// Create test request
	req, err := http.NewRequest("PUT", "/schedules/"+schedule.Id.Hex(), bytes.NewBuffer(jsonValue))
	req.Header.Set("Authorization", TestToken)
	req.Header.Set("Content-Type", "application/json")
	require.NoError(t, err)

	// Execute request
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	// Verify response
	assert.Equal(t, http.StatusOK, resp.Code)

	var response controllers.Response[models.Schedule]
	err = json.Unmarshal(resp.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "ok", response.Status)

	// Fetch the updated schedule to verify changes
	scheduleSvc := service.NewModelService[models.Schedule]()
	updatedScheduleFromDB, err := scheduleSvc.GetById(schedule.Id)
	require.NoError(t, err)

	assert.Equal(t, "Updated Schedule Name", updatedScheduleFromDB.Name)
	assert.Equal(t, "*/5 * * * *", updatedScheduleFromDB.Cron)
	assert.True(t, updatedScheduleFromDB.Enabled)
}

// Test DeleteScheduleById endpoint
func TestDeleteScheduleById(t *testing.T) {
	SetupTestDB()
	defer CleanupTestDB()

	gin.SetMode(gin.TestMode)

	// Create a test schedule
	schedule, _ := createTestSchedule(t)

	// Set up router
	router := SetupRouter()
	router.Use(middlewares.AuthorizationMiddleware())
	router.DELETE("/schedules/:id", nil, tonic.Handler(controllers.NewController[models.Schedule]().DeleteById, 200))

	// Create test request
	req, err := http.NewRequest("DELETE", "/schedules/"+schedule.Id.Hex(), nil)
	req.Header.Set("Authorization", TestToken)
	require.NoError(t, err)

	// Execute request
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	// Verify response
	assert.Equal(t, http.StatusOK, resp.Code)

	// Verify schedule is deleted from database
	scheduleSvc := service.NewModelService[models.Schedule]()
	_, err = scheduleSvc.GetById(schedule.Id)
	assert.Error(t, err) // Should return error as the schedule is deleted
}

// Test GetScheduleList endpoint
func TestGetScheduleList(t *testing.T) {
	SetupTestDB()
	defer CleanupTestDB()

	gin.SetMode(gin.TestMode)

	// Create several test schedules
	schedule1, _ := createTestSchedule(t)
	schedule2, _ := createTestSchedule(t)
	schedule2.Name = "Second Test Schedule"

	scheduleSvc := service.NewModelService[models.Schedule]()
	err := scheduleSvc.ReplaceById(schedule2.Id, *schedule2)
	require.NoError(t, err)

	// Set up router
	router := SetupRouter()
	router.Use(middlewares.AuthorizationMiddleware())
	router.GET("/schedules", nil, tonic.Handler(controllers.NewController[models.Schedule]().GetList, 200))

	// Create test request
	req, err := http.NewRequest("GET", "/schedules?page=1&size=10", nil)
	req.Header.Set("Authorization", TestToken)
	require.NoError(t, err)

	// Execute request
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	// Verify response
	assert.Equal(t, http.StatusOK, resp.Code)

	var response controllers.ListResponse[models.Schedule]
	err = json.Unmarshal(resp.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "ok", response.Status)
	assert.Equal(t, 2, response.Total) // We created 2 schedules
	assert.Equal(t, 2, len(response.Data))

	// Verify both schedules are in the response
	foundSchedule1 := false
	foundSchedule2 := false
	for _, schedule := range response.Data {
		if schedule.Id == schedule1.Id {
			foundSchedule1 = true
			assert.Equal(t, schedule1.Name, schedule.Name)
		}
		if schedule.Id == schedule2.Id {
			foundSchedule2 = true
			assert.Equal(t, "Second Test Schedule", schedule.Name)
		}
	}
	assert.True(t, foundSchedule1, "schedule1 should be in the response")
	assert.True(t, foundSchedule2, "schedule2 should be in the response")
}

// Test EnableSchedule endpoint
func TestEnableSchedule(t *testing.T) {
	SetupTestDB()
	defer CleanupTestDB()

	gin.SetMode(gin.TestMode)

	// Create a test schedule (disabled by default)
	schedule, _ := createTestSchedule(t)
	assert.False(t, schedule.Enabled)

	// Set up router
	router := SetupRouter()
	router.Use(middlewares.AuthorizationMiddleware())
	router.POST("/schedules/:id/enable", nil, tonic.Handler(controllers.PostScheduleEnable, 200))

	// Create test request
	req, err := http.NewRequest("POST", "/schedules/"+schedule.Id.Hex()+"/enable", nil)
	req.Header.Set("Authorization", TestToken)
	require.NoError(t, err)

	// Execute request
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	// Verify response
	assert.Equal(t, http.StatusOK, resp.Code)

	// Verify schedule is now enabled
	scheduleSvc := service.NewModelService[models.Schedule]()
	updatedSchedule, err := scheduleSvc.GetById(schedule.Id)
	require.NoError(t, err)
	assert.True(t, updatedSchedule.Enabled)
}

// Test DisableSchedule endpoint
func TestDisableSchedule(t *testing.T) {
	SetupTestDB()
	defer CleanupTestDB()

	gin.SetMode(gin.TestMode)

	// Create a test schedule and enable it
	schedule, _ := createTestSchedule(t)
	schedule.Enabled = true

	scheduleSvc := service.NewModelService[models.Schedule]()
	err := scheduleSvc.ReplaceById(schedule.Id, *schedule)
	require.NoError(t, err)

	// Set up router
	router := SetupRouter()
	router.Use(middlewares.AuthorizationMiddleware())
	router.POST("/schedules/:id/disable", nil, tonic.Handler(controllers.PostScheduleDisable, 200))

	// Create test request
	req, err := http.NewRequest("POST", "/schedules/"+schedule.Id.Hex()+"/disable", nil)
	req.Header.Set("Authorization", TestToken)
	require.NoError(t, err)

	// Execute request
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	// Verify response
	assert.Equal(t, http.StatusOK, resp.Code)

	// Verify schedule is now disabled
	updatedSchedule, err := scheduleSvc.GetById(schedule.Id)
	require.NoError(t, err)
	assert.False(t, updatedSchedule.Enabled)
}
