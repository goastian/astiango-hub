package controllers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"testing"
	"time"

	"github.com/crawlab-team/fizz"
	"github.com/goastian/astiango-hub/core/controllers"
	"github.com/goastian/astiango-hub/core/entity"
	"github.com/goastian/astiango-hub/core/middlewares"
	"github.com/goastian/astiango-hub/core/models/models"
	"github.com/goastian/astiango-hub/core/models/service"
	"github.com/goastian/astiango-hub/core/mongo"
	"github.com/goastian/astiango-hub/core/user"
	"github.com/loopfz/gadgeto/tonic"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"

	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func init() {

}

// TestModel is a simple struct to be used as a model in tests
type TestModel models.TestModel

//type TestModel struct {
//	Name string `json:"name" bson:"name"`
//}

var TestToken string
var TestUserId primitive.ObjectID

// SetupTestDB sets up the test database
func SetupTestDB() {
	viper.Set("mongo.db", "testdb")
	viper.Set("node.master", true) // Configure as master node for tests
	viper.Set("jwt.keyset", `{"active_kid":"test-2026","keys":{"test-2025":"ZmVkY2JhOTg3NjU0MzIxMGZlZGNiYTk4NzY1NDMyMTA=","test-2026":"MDEyMzQ1Njc4OWFiY2RlZjAxMjM0NTY3ODlhYmNkZWY="}}`)
	viper.Set("jwt.issuer", "astiango-hub-test")
	viper.Set("jwt.audience", "astiango-hub-test-api")
	viper.Set("jwt.access_ttl", "15m")
	viper.Set("jwt.refresh_ttl", "168h")
	viper.Set("jwt.leeway", "0s")

	modelSvc := service.NewModelService[models.User]()
	u := models.User{
		Username:  "admin",
		RootAdmin: true,
	}
	id, err := modelSvc.InsertOne(u)
	if err != nil {
		panic(err)
	}
	u.SetId(id)

	userSvc, err := user.GetUserService()
	if err != nil {
		panic(err)
	}
	token, err := userSvc.MakeToken(&u)
	if err != nil {
		panic(err)
	}
	TestToken = token
	TestUserId = u.Id
}

// SetupRouter sets up the gin router for testing
func SetupRouter() *fizz.Fizz {
	router := fizz.New()
	return router
}

// CleanupTestDB cleans up the test database
func CleanupTestDB() {
	mongo.GetMongoDb("testdb").Drop(context.Background())
}

func TestBaseController_GetById(t *testing.T) {
	SetupTestDB()
	defer CleanupTestDB()

	// Insert a test document
	id, err := service.NewModelService[TestModel]().InsertOne(TestModel{Name: "test"})
	assert.NoError(t, err)

	// Initialize the controller
	ctr := controllers.NewController[TestModel]()

	// Set up the router
	router := SetupRouter()
	router.Use(middlewares.AuthorizationMiddleware())
	router.GET("/testmodels/:id", nil, tonic.Handler(ctr.GetById, 200))

	// Create a test request
	req, _ := http.NewRequest("GET", "/testmodels/"+id.Hex(), nil)
	req.Header.Set("Authorization", TestToken)
	w := httptest.NewRecorder()

	// Serve the request
	router.ServeHTTP(w, req)

	// Check the response
	assert.Equal(t, http.StatusOK, w.Code)

	var response controllers.Response[TestModel]
	err = json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "test", response.Data.Name)
}

func TestBaseController_Post(t *testing.T) {
	SetupTestDB()
	defer CleanupTestDB()

	// Initialize the controller
	ctr := controllers.NewController[TestModel]()

	// Set up the router
	router := SetupRouter()
	router.Use(middlewares.AuthorizationMiddleware())
	router.POST("/testmodels", nil, tonic.Handler(ctr.Post, 200))

	// Create a test request
	testModel := TestModel{Name: "test"}
	requestBody := controllers.PostParams[TestModel]{
		Data: testModel,
	}
	jsonValue, _ := json.Marshal(requestBody)
	req, _ := http.NewRequest("POST", "/testmodels", bytes.NewBuffer(jsonValue))
	req.Header.Set("Authorization", TestToken)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// Serve the request
	router.ServeHTTP(w, req)

	// Check the response
	require.Equal(t, http.StatusOK, w.Code)

	var response controllers.Response[TestModel]
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	require.Equal(t, "test", response.Data.Name)

	// Check if the document was inserted into the database
	result, err := service.NewModelService[TestModel]().GetById(response.Data.Id)
	require.NoError(t, err)
	require.Equal(t, "test", result.Name)
}

func TestBaseController_DeleteById(t *testing.T) {
	SetupTestDB()
	defer CleanupTestDB()

	// Insert a test document
	id, err := service.NewModelService[TestModel]().InsertOne(TestModel{Name: "test"})
	assert.NoError(t, err)

	// Initialize the controller
	ctr := controllers.NewController[TestModel]()

	// Set up the router
	router := SetupRouter()
	router.Use(middlewares.AuthorizationMiddleware())
	router.DELETE("/testmodels/:id", nil, tonic.Handler(ctr.DeleteById, 200))

	// Create a test request
	req, _ := http.NewRequest("DELETE", "/testmodels/"+id.Hex(), nil)
	req.Header.Set("Authorization", TestToken)
	w := httptest.NewRecorder()

	// Serve the request
	router.ServeHTTP(w, req)

	// Check the response
	assert.Equal(t, http.StatusOK, w.Code)

	// Check if the document was deleted from the database
	_, err = service.NewModelService[TestModel]().GetById(id)
	assert.Error(t, err)
}

func TestBaseController_GetList(t *testing.T) {
	SetupTestDB()
	defer CleanupTestDB()

	// Insert test documents
	modelSvc := service.NewModelService[TestModel]()
	for i := 0; i < 15; i++ {
		_, err := modelSvc.InsertOne(TestModel{Name: fmt.Sprintf("test%d", i)})
		assert.NoError(t, err)
	}

	// Initialize the controller
	ctr := controllers.NewController[TestModel]()

	// Set up the router
	router := SetupRouter()
	router.Use(middlewares.AuthorizationMiddleware())
	router.GET("/testmodels/list", nil, tonic.Handler(ctr.GetList, 200))

	// Test case 1: Get with pagination
	t.Run("test_get_with_pagination", func(t *testing.T) {
		var testData = []struct {
			Page               int
			ExpectedDataCount  int
			ExpectedTotalCount int
		}{
			{1, 10, 15},
			{2, 5, 15},
		}
		for _, data := range testData {
			params := url.Values{}
			params.Add("page", strconv.Itoa(data.Page))
			params.Add("size", "10")
			requestUrl := url.URL{Path: "/testmodels/list", RawQuery: params.Encode()}
			req, _ := http.NewRequest("GET", requestUrl.String(), nil)
			req.Header.Set("Authorization", TestToken)
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			// Serve the request
			router.ServeHTTP(w, req)

			// Check the response
			assert.Equal(t, http.StatusOK, w.Code)

			var response controllers.ListResponse[TestModel]
			err := json.Unmarshal(w.Body.Bytes(), &response)
			assert.NoError(t, err)
			assert.Equal(t, data.ExpectedDataCount, len(response.Data))
			assert.Equal(t, data.ExpectedTotalCount, response.Total)
		}
	})

	// Test case 2: Get all
	t.Run("test_get_all", func(t *testing.T) {
		params := url.Values{}
		params.Add("all", "true")
		requestUrl := url.URL{Path: "/testmodels/list", RawQuery: params.Encode()}
		req, _ := http.NewRequest("GET", requestUrl.String(), nil)
		req.Header.Set("Authorization", TestToken)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		// Serve the request
		router.ServeHTTP(w, req)

		// Check the response
		assert.Equal(t, http.StatusOK, w.Code)

		var response controllers.ListResponse[TestModel]
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, 15, len(response.Data))
		assert.Equal(t, 15, response.Total)
	})

	// Test case 3: Get with query filter
	t.Run("test_get_with_query_filter", func(t *testing.T) {
		cond := []entity.Condition{
			{Key: "name", Op: "eq", Value: "test1"},
		}
		condBytes, err := json.Marshal(cond)
		require.Nil(t, err)
		params := url.Values{}
		params.Add("filter", string(condBytes))
		params.Add("page", "1")
		params.Add("size", "10")
		requestUrl := url.URL{Path: "/testmodels/list", RawQuery: params.Encode()}
		req, _ := http.NewRequest("GET", requestUrl.String(), nil)
		req.Header.Set("Authorization", TestToken)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		// Serve the request
		router.ServeHTTP(w, req)

		// Check the response
		assert.Equal(t, http.StatusOK, w.Code)

		var response controllers.ListResponse[TestModel]
		err = json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, 1, len(response.Data))
		assert.Equal(t, 1, response.Total)
	})
}

func TestBaseController_PutById(t *testing.T) {
	SetupTestDB()
	defer CleanupTestDB()

	// Insert a test document
	id, err := service.NewModelService[TestModel]().InsertOne(TestModel{Name: "test"})
	assert.NoError(t, err)

	// Initialize the controller
	ctr := controllers.NewController[TestModel]()

	// Set up the router
	router := SetupRouter()
	router.Use(middlewares.AuthorizationMiddleware())
	router.PUT("/testmodels/:id", nil, tonic.Handler(ctr.PutById, 200))

	// Create a test request
	updatedModel := TestModel{Name: "updated"}
	requestParams := controllers.PutByIdParams[TestModel]{
		Data: updatedModel,
	}
	jsonValue, _ := json.Marshal(requestParams)
	req, _ := http.NewRequest("PUT", "/testmodels/"+id.Hex(), bytes.NewBuffer(jsonValue))
	req.Header.Set("Authorization", TestToken)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// Serve the request
	router.ServeHTTP(w, req)

	// Check the response
	assert.Equal(t, http.StatusOK, w.Code)

	var response controllers.Response[TestModel]
	err = json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "updated", response.Data.Name)

	// Check if the document was updated in the database
	result, err := service.NewModelService[TestModel]().GetById(id)
	assert.NoError(t, err)
	assert.Equal(t, "updated", result.Name)

	// Test with invalid ID
	req, _ = http.NewRequest("PUT", "/testmodels/invalid-id", bytes.NewBuffer(jsonValue))
	req.Header.Set("Authorization", TestToken)
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()

	// Serve the request
	router.ServeHTTP(w, req)

	// Check the response
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestBaseController_PatchList(t *testing.T) {
	SetupTestDB()
	defer CleanupTestDB()

	// Insert test documents
	modelSvc := service.NewModelService[TestModel]()
	var ids []primitive.ObjectID
	for i := 0; i < 3; i++ {
		id, err := modelSvc.InsertOne(TestModel{Name: fmt.Sprintf("test%d", i)})
		assert.NoError(t, err)
		ids = append(ids, id)
	}

	// Initialize the controller
	ctr := controllers.NewController[TestModel]()

	// Set up the router
	router := SetupRouter()
	router.Use(middlewares.AuthorizationMiddleware())
	router.PATCH("/testmodels", nil, tonic.Handler(ctr.PatchList, 200))

	// Create a test request
	t.Run("test_patch_list", func(t *testing.T) {
		var idStrings []string
		for _, id := range ids {
			idStrings = append(idStrings, id.Hex())
		}
		requestBody := controllers.PatchParams{
			Ids:    idStrings,
			Update: bson.M{"name": "patched"},
		}
		jsonValue, _ := json.Marshal(requestBody)
		req, _ := http.NewRequest("PATCH", "/testmodels", bytes.NewBuffer(jsonValue))
		req.Header.Set("Authorization", TestToken)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		// Get the user ID
		userId := TestUserId

		// Record time before the update
		beforeUpdate := time.Now()

		time.Sleep(100 * time.Millisecond)

		// Serve the request
		router.ServeHTTP(w, req)

		// Check the response
		assert.Equal(t, http.StatusOK, w.Code)

		time.Sleep(100 * time.Millisecond)

		// Record time after the update
		afterUpdate := time.Now()

		// Check if the documents were updated in the database
		for _, id := range ids {
			result, err := modelSvc.GetById(id)
			assert.NoError(t, err)
			assert.Equal(t, "patched", result.Name)

			// Verify updated_by is set to the current user's ID
			assert.Equal(t, userId, result.UpdatedBy)

			// Verify updated_at is set to a recent timestamp
			assert.GreaterOrEqual(t, result.UpdatedAt.UnixMilli(), beforeUpdate.UnixMilli())
			assert.LessOrEqual(t, result.UpdatedAt.UnixMilli(), afterUpdate.UnixMilli())
		}
	})

	// Test with invalid ID
	t.Run("test_patch_list_with_invalid_id", func(t *testing.T) {
		requestBody := controllers.PatchParams{
			Ids:    []string{"invalid-id"},
			Update: bson.M{"name": "patched"},
		}
		jsonValue, _ := json.Marshal(requestBody)
		req, _ := http.NewRequest("PATCH", "/testmodels", bytes.NewBuffer(jsonValue))
		req.Header.Set("Authorization", TestToken)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		// Serve the request
		router.ServeHTTP(w, req)

		// Check the response
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestBaseController_DeleteList(t *testing.T) {
	SetupTestDB()
	defer CleanupTestDB()

	// Insert test documents
	modelSvc := service.NewModelService[TestModel]()
	var ids []primitive.ObjectID
	for i := 0; i < 3; i++ {
		id, err := modelSvc.InsertOne(TestModel{Name: fmt.Sprintf("test%d", i)})
		assert.NoError(t, err)
		ids = append(ids, id)
	}

	// Initialize the controller
	ctr := controllers.NewController[TestModel]()

	// Set up the router
	router := SetupRouter()
	router.Use(middlewares.AuthorizationMiddleware())
	router.DELETE("/testmodels", nil, tonic.Handler(ctr.DeleteList, 200))

	// Create a test request
	var idStrings []string
	for _, id := range ids {
		idStrings = append(idStrings, id.Hex())
	}
	requestBody := controllers.DeleteListParams{
		Ids: idStrings,
	}
	jsonValue, _ := json.Marshal(requestBody)
	req, _ := http.NewRequest("DELETE", "/testmodels", bytes.NewBuffer(jsonValue))
	req.Header.Set("Authorization", TestToken)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// Serve the request
	router.ServeHTTP(w, req)

	// Check the response
	assert.Equal(t, http.StatusOK, w.Code)

	// Check if the documents were deleted from the database
	for _, id := range ids {
		_, err := modelSvc.GetById(id)
		assert.Error(t, err)
	}

	// Test with invalid ID
	requestBody = controllers.DeleteListParams{
		Ids: []string{"invalid-id"},
	}
	jsonValue, _ = json.Marshal(requestBody)
	req, _ = http.NewRequest("DELETE", "/testmodels", bytes.NewBuffer(jsonValue))
	req.Header.Set("Authorization", TestToken)
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()

	// Serve the request
	router.ServeHTTP(w, req)

	// Check the response
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestBaseController_PatchById(t *testing.T) {
	SetupTestDB()
	defer CleanupTestDB()

	// Insert a test document
	id, err := service.NewModelService[TestModel]().InsertOne(TestModel{Name: "test"})
	assert.NoError(t, err)

	// Initialize the controller
	ctr := controllers.NewController[TestModel]()

	// Set up the router
	router := SetupRouter()
	router.Use(middlewares.AuthorizationMiddleware())
	router.PATCH("/testmodels/:id", nil, tonic.Handler(ctr.PatchById, 200))

	// Test case 1: Successful patch
	t.Run("test_patch_success", func(t *testing.T) {
		// Create update data
		updateData := TestModel{
			Name: "patched",
		}

		requestBody := controllers.PatchByIdParams[TestModel]{
			Id:   id.Hex(),
			Data: updateData,
		}

		jsonValue, _ := json.Marshal(requestBody)
		req, _ := http.NewRequest("PATCH", "/testmodels/"+id.Hex(), bytes.NewBuffer(jsonValue))
		req.Header.Set("Authorization", TestToken)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		// Get the user ID
		userId := TestUserId

		// Record time before the update
		beforeUpdate := time.Now()

		time.Sleep(time.Millisecond) // Ensure time difference

		// Serve the request
		router.ServeHTTP(w, req)

		// Check the response
		assert.Equal(t, http.StatusOK, w.Code)

		// Check if the document was updated in the database
		result, err := service.NewModelService[TestModel]().GetById(id)
		assert.NoError(t, err)
		assert.Equal(t, "patched", result.Name)
		assert.Equal(t, userId, result.UpdatedBy)
		assert.True(t, result.UpdatedAt.After(beforeUpdate))
	})

	// Test case 2: Invalid ID
	t.Run("test_patch_invalid_id", func(t *testing.T) {
		updateData := TestModel{
			Name: "patched",
		}

		requestBody := controllers.PatchByIdParams[TestModel]{
			Id:   "invalid-id",
			Data: updateData,
		}

		jsonValue, _ := json.Marshal(requestBody)
		req, _ := http.NewRequest("PATCH", "/testmodels/invalid-id", bytes.NewBuffer(jsonValue))
		req.Header.Set("Authorization", TestToken)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		// Serve the request
		router.ServeHTTP(w, req)

		// Check the response
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestBaseController_Post_Validation(t *testing.T) {
	SetupTestDB()
	defer CleanupTestDB()

	// Initialize the controller
	ctr := controllers.NewController[TestModel]()

	// Set up the router
	router := SetupRouter()
	router.Use(middlewares.AuthorizationMiddleware())
	router.POST("/testmodels", nil, tonic.Handler(ctr.Post, 200))

	// Test case: Empty data
	t.Run("test_post_empty_data", func(t *testing.T) {
		requestBody := controllers.PostParams[TestModel]{
			Data: TestModel{},
		}
		jsonValue, _ := json.Marshal(requestBody)
		req, _ := http.NewRequest("POST", "/testmodels", bytes.NewBuffer(jsonValue))
		req.Header.Set("Authorization", TestToken)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		// Serve the request
		router.ServeHTTP(w, req)

		// Check the response
		assert.Equal(t, http.StatusOK, w.Code)

		var response controllers.Response[TestModel]
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.NotEmpty(t, response.Data.Id)
		assert.Equal(t, TestUserId, response.Data.CreatedBy)
		assert.Equal(t, TestUserId, response.Data.UpdatedBy)
	})
}

func TestBaseController_GetList_Sort(t *testing.T) {
	SetupTestDB()
	defer CleanupTestDB()

	// Insert test documents
	modelSvc := service.NewModelService[TestModel]()
	for i := 0; i < 3; i++ {
		_, err := modelSvc.InsertOne(TestModel{Name: fmt.Sprintf("test%d", i)})
		assert.NoError(t, err)
	}

	// Initialize the controller
	ctr := controllers.NewController[TestModel]()

	// Set up the router
	router := SetupRouter()
	router.Use(middlewares.AuthorizationMiddleware())
	router.GET("/testmodels/list", nil, tonic.Handler(ctr.GetList, 200))

	t.Run("test_get_list_valid_sort", func(t *testing.T) {
		params := url.Values{}
		params.Add("sort", "-name") // Sort by name descending
		requestUrl := url.URL{Path: "/testmodels/list", RawQuery: params.Encode()}
		req, _ := http.NewRequest("GET", requestUrl.String(), nil)
		req.Header.Set("Authorization", TestToken)
		w := httptest.NewRecorder()

		// Serve the request
		router.ServeHTTP(w, req)

		// Check the response
		assert.Equal(t, http.StatusOK, w.Code)

		var response controllers.ListResponse[TestModel]
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, 3, len(response.Data))

		// Verify descending order
		for i := 0; i < len(response.Data)-1; i++ {
			assert.True(t, response.Data[i].Name > response.Data[i+1].Name)
		}
	})

	// Test case: Invalid sort format
	t.Run("test_get_list_invalid_sort", func(t *testing.T) {
		// Use an invalid sort format (missing field name)
		params := url.Values{}
		params.Add("sort", "-") // Invalid: hyphen without field name
		requestUrl := url.URL{Path: "/testmodels/list", RawQuery: params.Encode()}
		req, _ := http.NewRequest("GET", requestUrl.String(), nil)
		req.Header.Set("Authorization", TestToken)
		w := httptest.NewRecorder()

		// Serve the request
		router.ServeHTTP(w, req)

		// Check the response
		assert.Equal(t, http.StatusBadRequest, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Contains(t, response, "error")
		assert.Contains(t, response["error"].(string), "invalid sort format")
	})
}

func TestBaseController_GetList_Pagination_Edge_Cases(t *testing.T) {
	SetupTestDB()
	defer CleanupTestDB()

	// Insert test documents
	modelSvc := service.NewModelService[TestModel]()
	for i := 0; i < 5; i++ {
		_, err := modelSvc.InsertOne(TestModel{Name: fmt.Sprintf("test%d", i)})
		assert.NoError(t, err)
	}

	// Initialize the controller
	ctr := controllers.NewController[TestModel]()

	// Set up the router
	router := SetupRouter()
	router.Use(middlewares.AuthorizationMiddleware())
	router.GET("/testmodels/list", nil, tonic.Handler(ctr.GetList, 200))

	// Test cases
	testCases := []struct {
		name          string
		page          int
		size          int
		expectedCount int
		expectedTotal int
	}{
		{"test_empty_page", 10, 10, 0, 5},      // Page beyond data
		{"test_large_page_size", 1, 100, 5, 5}, // Page size larger than data
		{"test_exact_page_size", 1, 5, 5, 5},   // Page size equals data size
		{"test_last_page_partial", 2, 3, 2, 5}, // Last page with partial data
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			params := url.Values{}
			params.Add("page", strconv.Itoa(tc.page))
			params.Add("size", strconv.Itoa(tc.size))
			requestUrl := url.URL{Path: "/testmodels/list", RawQuery: params.Encode()}
			req, _ := http.NewRequest("GET", requestUrl.String(), nil)
			req.Header.Set("Authorization", TestToken)
			w := httptest.NewRecorder()

			// Serve the request
			router.ServeHTTP(w, req)

			// Check the response
			assert.Equal(t, http.StatusOK, w.Code)

			var response controllers.ListResponse[TestModel]
			err := json.Unmarshal(w.Body.Bytes(), &response)
			assert.NoError(t, err)
			assert.Equal(t, tc.expectedCount, len(response.Data))
			assert.Equal(t, tc.expectedTotal, response.Total)
		})
	}
}
