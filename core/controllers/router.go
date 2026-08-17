package controllers

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/goastian/astiango-hub/core/utils"

	"github.com/crawlab-team/fizz"

	"github.com/goastian/astiango-hub/core/middlewares"
	"github.com/goastian/astiango-hub/core/models/models"
	"github.com/goastian/astiango-hub/core/openapi"
	"github.com/gin-gonic/gin"
)

// RouterGroups defines the different authentication levels for API routes
type RouterGroups struct {
	AuthGroup      *fizz.RouterGroup    // Routes requiring full authentication
	AnonymousGroup *fizz.RouterGroup    // Public routes that don't require auth
	SyncAuthGroup  *fizz.RouterGroup    // Routes for sync operations with special auth
	Wrapper        *openapi.FizzWrapper // OpenAPI wrapper for documentation
}

// Global variable to store the OpenAPI wrapper
// This is a workaround since we can't easily pass it through the Gin context
var globalWrapper *openapi.FizzWrapper

func GetGlobalFizzWrapper() *openapi.FizzWrapper {
	return globalWrapper
}

// NewRouterGroups initializes the router groups with their respective middleware
func NewRouterGroups(app *gin.Engine) (groups *RouterGroups) {
	// Create OpenAPI wrapper
	globalWrapper = openapi.GetFizzWrapper(app)

	f := globalWrapper.GetFizz()

	return &RouterGroups{
		AuthGroup:      f.Group("/", "AuthGroup", "Router group that requires authentication", middlewares.AuthorizationMiddleware()),
		AnonymousGroup: f.Group("/", "AnonymousGroup", "Router group that doesn't require authentication"),
		SyncAuthGroup:  f.Group("/", "SyncAuthGroup", "Router group for sync operations with special auth", middlewares.SyncAuthorizationMiddleware()),
		Wrapper:        globalWrapper,
	}
}

// RegisterController registers a generic controller with standard CRUD endpoints
// and any additional custom actions
func RegisterController[T any](group *fizz.RouterGroup, basePath string, ctr *BaseController[T]) {
	// Track registered paths to avoid duplicates
	actionPaths := make(map[string]bool)
	for _, action := range ctr.actions {
		fullPath := basePath + action.Path
		key := action.Method + " - " + fullPath
		actionPaths[key] = true

		// Create appropriate model response based on the action
		responses := globalWrapper.BuildModelResponse()

		id := getIDForAction(action.Method, fullPath, action.Name)
		summary := getSummaryForAction(action.Method, basePath, action.Path, action.Name)
		description := getDescriptionForAction(action.Method, basePath, action.Path, action.Description)

		globalWrapper.RegisterRoute(action.Method, fullPath, group, action.HandlerFunc, id, summary, description, responses)
	}

	// Register built-in handlers if they haven't been overridden
	resource := getResourceName(basePath)
	registerBuiltinHandler(group, globalWrapper, basePath, http.MethodGet, "", ctr.GetList, actionPaths, fmt.Sprintf("Get %s List", resource), "Get a list of items")
	registerBuiltinHandler(group, globalWrapper, basePath, http.MethodGet, "/:id", ctr.GetById, actionPaths, fmt.Sprintf("Get %s by ID", resource), "Get a single item by ID")
	registerBuiltinHandler(group, globalWrapper, basePath, http.MethodPost, "", ctr.Post, actionPaths, fmt.Sprintf("Create %s", resource), "Create a new item")
	registerBuiltinHandler(group, globalWrapper, basePath, http.MethodPut, "/:id", ctr.PutById, actionPaths, fmt.Sprintf("Replace %s by ID", resource), "Replace an item by ID with (full update)")
	registerBuiltinHandler(group, globalWrapper, basePath, http.MethodPatch, "/:id", ctr.PatchById, actionPaths, fmt.Sprintf("Update %s by ID", resource), "Update an item by ID with (partial update)")
	registerBuiltinHandler(group, globalWrapper, basePath, http.MethodPatch, "", ctr.PatchList, actionPaths, fmt.Sprintf("Batch Update %s List", resource), "Batch update multiple items with partial fields")
	registerBuiltinHandler(group, globalWrapper, basePath, http.MethodDelete, "/:id", ctr.DeleteById, actionPaths, fmt.Sprintf("Delete %s by ID", resource), "Delete an item by ID")
	registerBuiltinHandler(group, globalWrapper, basePath, http.MethodDelete, "", ctr.DeleteList, actionPaths, fmt.Sprintf("Delete %s List", resource), "Delete multiple items")
}

// RegisterActions registers a list of custom action handlers to a route group
func RegisterActions(group *fizz.RouterGroup, basePath string, actions []Action) {
	for _, action := range actions {
		fullPath := basePath + action.Path

		// Create generic response
		responses := globalWrapper.BuildModelResponse()

		id := getIDForAction(action.Method, fullPath, action.Name)
		summary := getSummaryForAction(action.Method, basePath, action.Path, action.Name)
		description := getDescriptionForAction(action.Method, basePath, action.Path, action.Description)

		globalWrapper.RegisterRoute(action.Method, fullPath, group, action.HandlerFunc, id, summary, description, responses)
	}
}

// registerBuiltinHandler registers a standard handler if it hasn't been overridden
// by a custom action
func registerBuiltinHandler(group *fizz.RouterGroup, wrapper *openapi.FizzWrapper, basePath, method, pathSuffix string, handlerFunc interface{}, existingActionPaths map[string]bool, summary, description string) {
	path := basePath + pathSuffix
	key := method + " - " + path
	_, ok := existingActionPaths[key]
	if ok {
		return
	}

	id := getIDForAction(method, path, summary)

	// Create appropriate response based on the method
	responses := wrapper.BuildModelResponse()

	wrapper.RegisterRoute(method, path, group, handlerFunc, id, summary, description, responses)
}

// Helper functions to generate OpenAPI documentation
func getIDForAction(method, path, summary string) string {
	if summary != "" {
		return utils.ToSnakeCase(summary)
	}

	// Remove leading slash and convert remaining slashes to underscores
	cleanPath := strings.TrimPrefix(path, "/")
	cleanPath = strings.ReplaceAll(cleanPath, "/", "_")
	cleanPath = strings.ReplaceAll(cleanPath, ":", "_")
	cleanPath = strings.ReplaceAll(cleanPath, "__", "_")

	return method + "_" + cleanPath
}

func getSummaryForAction(method, basePath, path, summary string) string {
	if summary != "" {
		return summary
	}

	resource := getResourceName(basePath)

	switch method {
	case http.MethodGet:
		if path == "" {
			return "List " + resource
		} else if path == "/:id" {
			return "Get " + resource + " by ID"
		}
	case http.MethodPost:
		if path == "" {
			return "Create " + resource
		}
	case http.MethodPut:
		if path == "/:id" {
			return "Update " + resource + " by ID"
		}
	case http.MethodPatch:
		if path == "" {
			return "Patch " + resource + " List"
		}
	case http.MethodDelete:
		if path == "/:id" {
			return "Delete " + resource + " by ID"
		} else if path == "" {
			return "Delete " + resource + " List"
		}
	}

	// For custom actions, use a more descriptive summary
	if path != "" && path != "/:id" {
		return method + " " + resource + path
	}

	return method + " " + resource
}

func getDescriptionForAction(method, basePath, path, description string) string {
	if description != "" {
		return description
	}

	resource := getResourceName(basePath)

	switch method {
	case http.MethodGet:
		if path == "" {
			return "Get a list of " + resource + " items"
		} else if path == "/:id" {
			return "Get a single " + resource + " by ID"
		}
	case http.MethodPost:
		if path == "" {
			return "Create a new " + resource
		}
	case http.MethodPut:
		if path == "/:id" {
			return "Update a " + resource + " by ID"
		}
	case http.MethodPatch:
		if path == "" {
			return "Patch multiple " + resource + " items"
		}
	case http.MethodDelete:
		if path == "/:id" {
			return "Delete a " + resource + " by ID"
		} else if path == "" {
			return "Delete multiple " + resource + " items"
		}
	}

	// For custom actions, use a more descriptive description
	if path != "" && path != "/:id" {
		return "Perform " + method + " operation on " + resource + " with path " + path
	}

	return "Perform " + method + " operation on " + resource
}

func getResourceName(basePath string) string {
	resource := basePath

	// Remove leading slash and get the last part of the path
	if len(resource) > 0 && resource[0] == '/' {
		resource = resource[1:]
	}

	// Remove trailing slash if present
	if len(resource) > 0 && resource[len(resource)-1] == '/' {
		resource = resource[:len(resource)-1]
	}

	// Convert to capitalized form
	resource = utils.Capitalize(resource)

	// Convert to non-plural form
	resource = strings.TrimSuffix(resource, "s")

	return resource
}

// InitRoutes configures all API routes for the application
func InitRoutes(app *gin.Engine) (err error) {
	// Initialize route groups with different auth levels
	groups := NewRouterGroups(app)

	// Store the wrapper in the global variable for later use
	globalWrapper = groups.Wrapper

	// Register resource controllers with their respective endpoints
	// Each RegisterController call sets up standard CRUD operations
	// Additional custom actions can be specified in the controller initialization
	RegisterController(groups.AuthGroup.Group("", "Nodes", "APIs for nodes management"), "/nodes", NewController[models.Node]())
	RegisterController(groups.AuthGroup.Group("", "Projects", "APIs for projects management"), "/projects", NewController[models.Project]([]Action{
		{
			Method:      http.MethodGet,
			Path:        "",
			Name:        "Get Project List",
			Description: "Get a list of projects",
			HandlerFunc: GetProjectList,
		},
	}...))
	RegisterController(groups.AuthGroup.Group("", "Spiders", "APIs for spiders management"), "/spiders", NewController[models.Spider]([]Action{
		{
			Method:      http.MethodGet,
			Path:        "/:id",
			Name:        "Get Spider by ID",
			Description: "Get a single spider by ID",
			HandlerFunc: GetSpiderById,
		},
		{
			Method:      http.MethodGet,
			Path:        "",
			Name:        "Get Spider List",
			Description: "Get a list of spiders",
			HandlerFunc: GetSpiderList,
		},
		{
			Method:      http.MethodPost,
			Path:        "",
			Name:        "Create Spider",
			Description: "Create a new spider",
			HandlerFunc: PostSpider,
		},
		{
			Method:      http.MethodDelete,
			Path:        "/:id",
			Name:        "Delete Spider by ID",
			Description: "Delete a spider by ID",
			HandlerFunc: DeleteSpiderById,
		},
		{
			Method:      http.MethodDelete,
			Path:        "",
			Name:        "Delete Spider List",
			Description: "Delete a list of spiders",
			HandlerFunc: DeleteSpiderList,
		},
		{
			Method:      http.MethodGet,
			Path:        "/:id/files/list",
			Name:        "List Spider Files",
			Description: "List the files given a directory path in the spider project",
			HandlerFunc: GetSpiderFiles,
		},
		{
			Method:      http.MethodGet,
			Path:        "/:id/files/get",
			Name:        "Get Spider File Content",
			Description: "Get the content of a spider file",
			HandlerFunc: GetSpiderFileContent,
		},
		{
			Method:      http.MethodGet,
			Path:        "/:id/files/info",
			Name:        "Get Spider File Info",
			Description: "Get the info of a spider file",
			HandlerFunc: GetSpiderFileInfo,
		},
		{
			Method:      http.MethodPost,
			Path:        "/:id/files/save",
			Name:        "Save Spider File",
			Description: "Save a spider file",
			HandlerFunc: PostSpiderSaveFile,
		},
		// TODO: temporarily disabled due to compatibility issue
		//{
		//	Method:      http.MethodPost,
		//	Path:        "/:id/files/save/batch",
		//	Name:        "Save Spider Files",
		//	Description: "Save multiple spider files",
		//	HandlerFunc: PostSpiderSaveFiles,
		//},
		{
			Method:      http.MethodPost,
			Path:        "/:id/files/save/dir",
			Name:        "Save Spider Dir",
			Description: "Save a spider directory",
			HandlerFunc: PostSpiderSaveDir,
		},
		{
			Method:      http.MethodPost,
			Path:        "/:id/files/rename",
			Name:        "Rename Spider File",
			Description: "Rename a spider file",
			HandlerFunc: PostSpiderRenameFile,
		},
		{
			Method:      http.MethodDelete,
			Path:        "/:id/files",
			Name:        "Delete Spider File",
			Description: "Delete a spider file",
			HandlerFunc: DeleteSpiderFile,
		},
		{
			Method:      http.MethodPost,
			Path:        "/:id/files/copy",
			Name:        "Copy Spider File",
			Description: "Copy a spider file",
			HandlerFunc: PostSpiderCopyFile,
		},
		{
			Method:      http.MethodPost,
			Path:        "/:id/files/export",
			Name:        "Export Spider Files",
			Description: "Export spider files to a zip file",
			HandlerFunc: PostSpiderExport,
		},
		{
			Method:      http.MethodPost,
			Path:        "/:id/run",
			Name:        "Run Spider",
			Description: "Run a task for the given spider",
			HandlerFunc: PostSpiderRun,
		},
		{
			Method:      http.MethodGet,
			Path:        "/:id/results",
			Name:        "Get Spider Results",
			Description: "Get the scraped or crawled results data of a spider",
			HandlerFunc: GetSpiderResults,
		},
	}...))
	groups.AuthGroup.GinRouterGroup().POST("/spiders/:id/files/save/batch", PostSpiderSaveFilesGin) // TODO: temporarily use this due to compatibility issue
	RegisterController(groups.AuthGroup.Group("", "Schedules", "APIs for schedules management"), "/schedules", NewController[models.Schedule]([]Action{
		{
			Method:      http.MethodGet,
			Path:        "/:id",
			Name:        "Get Schedule by ID",
			Description: "Get a single schedule by ID",
			HandlerFunc: GetScheduleById,
		},
		{
			Method:      http.MethodGet,
			Path:        "",
			Name:        "Get Schedule List",
			Description: "Get a list of schedules",
			HandlerFunc: GetScheduleList,
		},
		{
			Method:      http.MethodPost,
			Path:        "",
			Name:        "Create Schedule",
			Description: "Create a new schedule",
			HandlerFunc: PostSchedule,
		},
		{
			Method:      http.MethodPut,
			Path:        "/:id",
			Name:        "Replace Schedule by ID",
			Description: "Replace a schedule by ID (full update)",
			HandlerFunc: PutScheduleById,
		},
		{
			Method:      http.MethodPost,
			Path:        "/:id/enable",
			Name:        "Enable Schedule",
			Description: "Enable a schedule",
			HandlerFunc: PostScheduleEnable,
		},
		{
			Method:      http.MethodPost,
			Path:        "/:id/disable",
			Name:        "Disable Schedule",
			Description: "Disable a schedule",
			HandlerFunc: PostScheduleDisable,
		},
		{
			Method:      http.MethodPost,
			Path:        "/:id/run",
			Name:        "Run Schedule",
			Description: "Run a schedule",
			HandlerFunc: PostScheduleRun,
		},
	}...))
	RegisterController(groups.AuthGroup.Group("", "Tasks", "APIs for tasks management"), "/tasks", NewController[models.Task]([]Action{
		{
			Method:      http.MethodGet,
			Path:        "/:id",
			Name:        "Get Task by ID",
			Description: "Get a single task by ID",
			HandlerFunc: GetTaskById,
		},
		{
			Method:      http.MethodGet,
			Path:        "",
			Name:        "Get Task List",
			Description: "Get a list of tasks (default sorted in descending order)",
			HandlerFunc: GetTaskList,
		},
		{
			Method:      http.MethodDelete,
			Path:        "/:id",
			Name:        "Delete Task by ID",
			Description: "Delete a task by ID",
			HandlerFunc: DeleteTaskById,
		},
		{
			Method:      http.MethodDelete,
			Path:        "",
			Name:        "Delete Task List",
			Description: "Delete a list of tasks",
			HandlerFunc: DeleteTaskList,
		},
		{
			Method:      http.MethodPost,
			Path:        "/run",
			Name:        "Run Task",
			Description: "Run a task",
			HandlerFunc: PostTaskRun,
		},
		{
			Method:      http.MethodPost,
			Path:        "/:id/restart",
			Name:        "Restart Task",
			Description: "Restart a task",
			HandlerFunc: PostTaskRestart,
		},
		{
			Method:      http.MethodPost,
			Path:        "/:id/cancel",
			Name:        "Cancel Task",
			Description: "Cancel a task",
			HandlerFunc: PostTaskCancel,
		},
		{
			Method:      http.MethodGet,
			Path:        "/:id/logs",
			Name:        "Get Task Logs",
			Description: "Get the logs of a task",
			HandlerFunc: GetTaskLogs,
		},
		{
			Method:      http.MethodGet,
			Path:        "/:id/results",
			Name:        "Get Task Results",
			Description: "Get the scraped or crawled results data of a task",
			HandlerFunc: GetTaskResults,
		},
	}...))
	RegisterController(groups.AuthGroup.Group("", "Users", "APIs for users management"), "/users", NewController[models.User]([]Action{
		{
			Method:      http.MethodGet,
			Path:        "/:id",
			Name:        "Get User by ID",
			Description: "Get a single user by ID",
			HandlerFunc: GetUserById,
		},
		{
			Method:      http.MethodGet,
			Path:        "",
			Name:        "Get User List",
			Description: "Get a list of users",
			HandlerFunc: GetUserList,
		},
		{
			Method:      http.MethodPost,
			Path:        "",
			Name:        "Create User",
			Description: "Create a new user",
			HandlerFunc: PostUser,
		},
		{
			Method:      http.MethodPut,
			Path:        "/:id",
			Name:        "Replace User by ID",
			Description: "Replace a user by ID (full update)",
			HandlerFunc: PutUserById,
		},
		{
			Method:      http.MethodPost,
			Path:        "/:id/change-password",
			Name:        "Change User Password",
			Description: "Change a user's password",
			HandlerFunc: PostUserChangePassword,
		},
		{
			Method:      http.MethodDelete,
			Path:        "/:id",
			Name:        "Delete User by ID",
			Description: "Delete a user by ID",
			HandlerFunc: DeleteUserById,
		},
		{
			Method:      http.MethodDelete,
			Path:        "",
			Name:        "Delete User List",
			Description: "Delete a list of users",
			HandlerFunc: DeleteUserList,
		},
		{
			Method:      http.MethodGet,
			Path:        "/me",
			Name:        "Get Me",
			Description: "Get the current user",
			HandlerFunc: GetUserMe,
		},
		{
			Method:      http.MethodPut,
			Path:        "/me",
			Name:        "Replace Me",
			Description: "Update the current user (full update)",
			HandlerFunc: PutUserMe,
		},
		{
			Method:      http.MethodPost,
			Path:        "/me/change-password",
			Name:        "Change My Password",
			Description: "Change the current user's password",
			HandlerFunc: PostUserMeChangePassword,
		},
	}...))
	RegisterController(groups.AuthGroup.Group("", "Tokens", "APIs for PAT management"), "/tokens", NewController[models.Token]([]Action{
		{
			Method:      http.MethodPost,
			Path:        "",
			Name:        "Create Token",
			Description: "Create a new token",
			HandlerFunc: PostToken,
		},
		{
			Method:      http.MethodGet,
			Path:        "",
			Name:        "Get Token List",
			Description: "Get a list of tokens",
			HandlerFunc: GetTokenList,
		},
	}...))
	RegisterController(groups.AuthGroup.Group("", "Environments", "APIs for environment variables management"), "/environments", NewController[models.Environment]())
	RegisterController(groups.AuthGroup.Group("", "Data Collections", "APIs for data collections management"), "/data/collections", NewController[models.DataCollection]())

	// Register standalone action routes that don't fit the standard CRUD pattern
	RegisterActions(groups.AuthGroup.Group("", "Export", "APIs for exporting data"), "/export", []Action{
		{
			Method:      http.MethodPost,
			Path:        "/:type",
			Name:        "Export Data",
			Description: "Export data",
			HandlerFunc: PostExport,
		},
		{
			Method:      http.MethodGet,
			Path:        "/:type/:id",
			Name:        "Get Export",
			Description: "Get an export",
			HandlerFunc: GetExport,
		},
		{
			Method:      http.MethodGet,
			Path:        "/:type/:id/download",
			Name:        "Get Export Download",
			Description: "Get an export download",
			HandlerFunc: GetExportDownload,
		},
	})
	RegisterActions(groups.AuthGroup.Group("", "Filters", "APIs for data collections filters management"), "/filters", []Action{
		{
			Method:      http.MethodGet,
			Path:        "/:col",
			Name:        "Get Filter Column Field Options",
			Description: "Get the field options of a collection",
			HandlerFunc: GetFilterColFieldOptions,
		},
		{
			Method:      http.MethodGet,
			Path:        "/:col/:value",
			Name:        "Get Filter Col Field Options With Value",
			Description: "Get the field options of a collection with a value",
			HandlerFunc: GetFilterColFieldOptionsWithValue,
		},
		{
			Method:      http.MethodGet,
			Path:        "/:col/:value/:label",
			Name:        "Get Filter Col Field Options With Value And Label",
			Description: "Get the field options of a collection with a value and label",
			HandlerFunc: GetFilterColFieldOptionsWithValueLabel,
		},
	})
	RegisterActions(groups.AuthGroup.Group("", "Settings", "APIs for settings management"), "/settings", []Action{
		{
			Method:      http.MethodGet,
			Path:        "/:key",
			Name:        "Get Setting",
			Description: "Get a setting",
			HandlerFunc: GetSetting,
		},
		{
			Method:      http.MethodPost,
			Path:        "/:key",
			Name:        "Create Setting",
			Description: "Create a new setting",
			HandlerFunc: PostSetting,
		},
		{
			Method:      http.MethodPut,
			Path:        "/:key",
			Name:        "Replace Setting by Type",
			Description: "Replace a setting by key (full update)",
			HandlerFunc: PutSetting,
		},
	})
	RegisterActions(groups.AuthGroup.Group("", "Stats", "APIs for data stats"), "/stats", []Action{
		{
			Method:      http.MethodGet,
			Path:        "/overview",
			Name:        "Get Stats Overview",
			Description: "Get the overview of the stats",
			HandlerFunc: GetStatsOverview,
		},
		{
			Method:      http.MethodGet,
			Path:        "/daily",
			Name:        "Get Stats Daily",
			Description: "Get the daily stats",
			HandlerFunc: GetStatsDaily,
		},
		{
			Method:      http.MethodGet,
			Path:        "/tasks",
			Name:        "Get Stats Tasks",
			Description: "Get the tasks stats",
			HandlerFunc: GetStatsTasks,
		},
	})

	// Register public routes that don't require authentication
	RegisterActions(groups.AnonymousGroup.Group("", "System", "APIs for system info"), "/system-info", []Action{
		{
			Path:        "",
			Method:      http.MethodGet,
			Name:        "Get System Info",
			Description: "Get the system info",
			HandlerFunc: GetSystemInfo,
		},
	})
	RegisterActions(groups.AnonymousGroup.Group("", "Auth", "APIs for authentication"), "/", []Action{
		{
			Method:      http.MethodPost,
			Path:        "/login",
			Name:        "Login",
			Description: "Login",
			HandlerFunc: PostLogin,
		},
		{
			Method:      http.MethodPost,
			Path:        "/logout",
			Name:        "Logout",
			Description: "Logout",
			HandlerFunc: PostLogout,
		},
	})

	// Register sync routes that require special authentication
	RegisterActions(groups.SyncAuthGroup.Group("", "Sync", "APIs for sync operations"), "/sync/:id", []Action{
		{
			Method:      http.MethodGet,
			Path:        "/scan",
			Name:        "Scan files (sync)",
			Description: "Scan files for a specific ID (Spider/Git)",
			HandlerFunc: GetSyncScan,
		},
		{
			Method:      http.MethodGet,
			Path:        "/download",
			Name:        "Download file (sync)",
			Description: "Download a file for a specific ID (Spider/Git)",
			HandlerFunc: GetSyncDownload,
		},
	})

	// Register health check route
	groups.AnonymousGroup.GinRouterGroup().GET("/health", GetHealthFn(func() bool { return true }))

	// Register OpenAPI documentation route
	groups.AnonymousGroup.GinRouterGroup().GET("/openapi.json", GetOpenAPI)

	return nil
}
