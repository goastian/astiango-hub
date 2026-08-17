package controllers_test

import (
	"sort"
	"testing"

	"github.com/goastian/astiango-hub/core/controllers"
	"github.com/goastian/astiango-hub/core/entity"
	"github.com/goastian/astiango-hub/core/models/models"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestGetFilterFromConditionString(t *testing.T) {
	// Simple condition with string value
	condStr := `[{"key":"name","op":"eq","value":"test"}]`
	filter := controllers.ConvertToFilter(condStr)
	require.NotNil(t, filter)
	require.Len(t, filter.Conditions, 1)
	assert.Equal(t, "name", filter.Conditions[0].Key)
	assert.Equal(t, "eq", filter.Conditions[0].Op)
	assert.Equal(t, "test", filter.Conditions[0].Value)

	// Multiple conditions with different types
	condStr = `[{"key":"name","op":"eq","value":"test"},{"key":"priority","op":"gt","value":5}]`
	filter = controllers.ConvertToFilter(condStr)
	require.NotNil(t, filter)
	require.Len(t, filter.Conditions, 2)
	assert.Equal(t, "name", filter.Conditions[0].Key)
	assert.Equal(t, "eq", filter.Conditions[0].Op)
	assert.Equal(t, "test", filter.Conditions[0].Value)
	assert.Equal(t, "priority", filter.Conditions[1].Key)
	assert.Equal(t, "gt", filter.Conditions[1].Op)
	assert.Equal(t, int64(5), filter.Conditions[1].Value)

	// Invalid JSON should return error
	condStr = `[{"key":"name","op":"eq","value":"test"`
	filter = controllers.ConvertToFilter(condStr)
	assert.Nil(t, filter)
}

func TestGetFilterQueryFromConditionString(t *testing.T) {
	// Simple equality condition
	condStr := `[{"key":"name","op":"eq","value":"test"}]`
	query := controllers.ConvertToBsonMFromFilter(condStr)
	require.NotNil(t, query)
	expected := bson.M{"name": "test"}
	assert.Equal(t, expected, query)

	// Greater than condition
	condStr = `[{"key":"priority","op":"gt","value":5}]`
	query = controllers.ConvertToBsonMFromFilter(condStr)
	require.NotNil(t, query)
	expected = bson.M{"priority": bson.M{"$gt": int64(5)}}
	assert.Equal(t, expected, query)

	// Multiple conditions
	condStr = `[{"key":"name","op":"eq","value":"test"},{"key":"priority","op":"gt","value":5}]`
	query = controllers.ConvertToBsonMFromFilter(condStr)
	require.NotNil(t, query)
	expected = bson.M{"name": "test", "priority": bson.M{"$gt": int64(5)}}
	assert.Equal(t, expected, query)

	// Contains operator
	condStr = `[{"key":"name","op":"c","value":"test"}]`
	query = controllers.ConvertToBsonMFromFilter(condStr)
	require.NotNil(t, query)
	expectedRegex := bson.M{"name": bson.M{"$regex": "test", "$options": "i"}}
	assert.Equal(t, expectedRegex, query)

	// Invalid condition should return error
	condStr = `[{"key":"name","op":"invalid_op","value":"test"}]`
	query = controllers.ConvertToBsonMFromFilter(condStr)
	assert.Nil(t, query)
}

func TestGetFilterQueryFromListParams(t *testing.T) {
	// No conditions
	params := &controllers.GetListParams{}
	query := controllers.ConvertToBsonMFromListParams(params)
	assert.Nil(t, query)

	// With conditions
	params.Filter = `[{"key":"name","op":"eq","value":"test"}]`
	query = controllers.ConvertToBsonMFromListParams(params)
	require.NotNil(t, query)
	expected := bson.M{"name": "test"}
	assert.Equal(t, expected, query)
}

func TestGetUserFromContext(t *testing.T) {
	// Empty context should return nil
	c := &gin.Context{}
	user := controllers.GetUserFromContext(c)
	assert.Nil(t, user)

	// Context with non-user value should return nil
	c = &gin.Context{}
	c.Set("user", "not a user object")
	user = controllers.GetUserFromContext(c)
	assert.Nil(t, user)

	// Context with user should return the user
	c = &gin.Context{}
	expectedUser := &models.User{Username: "test_user"}
	expectedUser.Id = primitive.NewObjectID()
	c.Set("user", expectedUser)
	user = controllers.GetUserFromContext(c)
	assert.NotNil(t, user)
	assert.Equal(t, expectedUser.Id, user.Id)
	assert.Equal(t, expectedUser.Username, user.Username)
}

func TestGetErrorResponse(t *testing.T) {
	// Error response test
	err := assert.AnError
	resp, _ := controllers.GetErrorResponse[models.Task](err)
	assert.Equal(t, err.Error(), resp.Error)
	assert.Equal(t, models.Task{}, resp.Data)
}

func TestGetDataResponse(t *testing.T) {
	// Data response test
	task := models.Task{
		Status: "running",
		Cmd:    "python main.py",
		Param:  "test param",
	}
	task.Id = primitive.NewObjectID()

	resp, err := controllers.GetDataResponse(task)
	require.NoError(t, err)
	assert.Equal(t, "ok", resp.Status)
	assert.Equal(t, task, resp.Data)
	assert.Empty(t, resp.Error)
}

func TestGetListResponse(t *testing.T) {
	// List response test
	tasks := []models.Task{
		{
			Status: "running",
			Cmd:    "python main.py",
		},
		{
			Status: "pending",
			Cmd:    "python main.py",
		},
	}
	tasks[0].Id = primitive.NewObjectID()
	tasks[1].Id = primitive.NewObjectID()

	total := 2
	resp, err := controllers.GetListResponse(tasks, total)
	require.NoError(t, err)
	assert.Equal(t, "ok", resp.Status)
	assert.Equal(t, tasks, resp.Data)
	assert.Equal(t, total, resp.Total)
	assert.Empty(t, resp.Error)
}

func TestGetErrorListResponse(t *testing.T) {
	// Error list response test
	err := assert.AnError
	resp, _ := controllers.GetErrorListResponse[models.Task](err)
	assert.Equal(t, err.Error(), resp.Error)
	assert.Nil(t, resp.Data)
	assert.Equal(t, 0, resp.Total)
}

func TestConvertToFilterMap(t *testing.T) {
	// Simple map with string value
	condStr := `{"name": "test"}`
	filter := controllers.ConvertToFilter(condStr)
	require.NotNil(t, filter)
	require.Len(t, filter.Conditions, 1)
	assert.Equal(t, "name", filter.Conditions[0].Key)
	assert.Equal(t, "eq", filter.Conditions[0].Op)
	assert.Equal(t, "test", filter.Conditions[0].Value)

	// Map with multiple fields of different types
	condStr = `{"name": "test", "priority": 5, "active": true}`
	filter = controllers.ConvertToFilter(condStr)
	require.NotNil(t, filter)
	require.Len(t, filter.Conditions, 3)
	// Sort conditions to ensure consistent test results
	sortConditions(filter.Conditions)
	assert.Equal(t, "active", filter.Conditions[0].Key)
	assert.Equal(t, "eq", filter.Conditions[0].Op)
	assert.Equal(t, true, filter.Conditions[0].Value)
	assert.Equal(t, "name", filter.Conditions[1].Key)
	assert.Equal(t, "eq", filter.Conditions[1].Op)
	assert.Equal(t, "test", filter.Conditions[1].Value)
	assert.Equal(t, "priority", filter.Conditions[2].Key)
	assert.Equal(t, "eq", filter.Conditions[2].Op)
	assert.Equal(t, int64(5), filter.Conditions[2].Value)

	// Map with ObjectID string
	id := primitive.NewObjectID()
	condStr = `{"_id": "` + id.Hex() + `"}`
	filter = controllers.ConvertToFilter(condStr)
	require.NotNil(t, filter)
	require.Len(t, filter.Conditions, 1)
	assert.Equal(t, "_id", filter.Conditions[0].Key)
	assert.Equal(t, "eq", filter.Conditions[0].Op)
	assert.Equal(t, id, filter.Conditions[0].Value)

	// Invalid JSON should return nil
	condStr = `{"name": "test"`
	filter = controllers.ConvertToFilter(condStr)
	assert.Nil(t, filter)
}

func sortConditions(conditions []*entity.Condition) {
	sort.Slice(conditions, func(i, j int) bool {
		return conditions[i].Key < conditions[j].Key
	})
}
