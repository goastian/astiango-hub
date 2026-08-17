package controllers_test

import (
	"testing"

	"github.com/goastian/astiango-hub/core/controllers"
	"github.com/goastian/astiango-hub/core/models/models"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestRouterGroups(t *testing.T) {
	router := gin.Default()
	groups := controllers.NewRouterGroups(router)

	assertions := []struct {
		group *gin.RouterGroup
		name  string
	}{
		{groups.AuthGroup.GinRouterGroup(), "AuthGroup"},
		{groups.AnonymousGroup.GinRouterGroup(), "AnonymousGroup"},
	}

	for _, a := range assertions {
		assert.NotNil(t, a.group, a.name+" should not be nil")
	}
}

func TestRegisterController_Routes(t *testing.T) {
	router := gin.Default()
	groups := controllers.NewRouterGroups(router)
	ctr := controllers.NewController[models.TestModel]()
	basePath := "/testmodels"

	controllers.RegisterController(groups.AuthGroup, basePath, ctr)

	// Check if all routes are registered
	routes := controllers.GetGlobalFizzWrapper().GetFizz().Engine().Routes()

	var methodPaths []string
	for _, route := range routes {
		methodPaths = append(methodPaths, route.Method+" - "+route.Path)
	}

	expectedRoutes := []gin.RouteInfo{
		{Method: "GET", Path: basePath},
		{Method: "GET", Path: basePath + "/:id"},
		{Method: "POST", Path: basePath},
		{Method: "PUT", Path: basePath + "/:id"},
		{Method: "PATCH", Path: basePath + "/:id"},
		{Method: "PATCH", Path: basePath},
		{Method: "DELETE", Path: basePath + "/:id"},
		{Method: "DELETE", Path: basePath},
	}

	assert.Equal(t, len(expectedRoutes), len(routes))
	for _, route := range expectedRoutes {
		assert.Contains(t, methodPaths, route.Method+" - "+route.Path)
	}
}

func TestMain(m *testing.M) {
	gin.SetMode(gin.TestMode)
	m.Run()
}
