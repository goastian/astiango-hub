package controllers

import (
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/goastian/astiango-hub/core/constants"
	"github.com/goastian/astiango-hub/core/interfaces"
	"github.com/goastian/astiango-hub/core/models/models"
	"github.com/goastian/astiango-hub/core/models/service"
	mongo2 "github.com/goastian/astiango-hub/core/mongo"
	"github.com/goastian/astiango-hub/core/spider/admin"
	"github.com/goastian/astiango-hub/core/task/log"
	"github.com/goastian/astiango-hub/core/task/scheduler"
	"github.com/goastian/astiango-hub/core/utils"
	"github.com/gin-gonic/gin"
	"github.com/juju/errors"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type GetTaskByIdParams struct {
	Id string `path:"id" description:"Task ID" format:"objectid" pattern:"^[0-9a-fA-F]{24}$"`
}

func GetTaskById(_ *gin.Context, params *GetTaskByIdParams) (response *Response[models.TaskDTO], err error) {
	// id
	id, err := primitive.ObjectIDFromHex(params.Id)
	if err != nil {
		return GetErrorResponse[models.TaskDTO](err)
	}

	// aggregation pipelines
	pipelines := service.GetByIdPipeline(id)
	pipelines = addTaskPipelines(pipelines)

	// perform query
	var tasks []models.TaskDTO
	err = service.GetCollection[models.TaskDTO]().Aggregate(pipelines, nil).All(&tasks)
	if err != nil {
		return GetErrorResponse[models.TaskDTO](err)
	}

	// check results
	if len(tasks) == 0 {
		return nil, errors.NotFoundf("task %s not found", params.Id)
	}

	return GetDataResponse(tasks[0])
}

func GetTaskList(_ *gin.Context, params *GetListParams) (response *ListResponse[models.TaskDTO], err error) {
	// query parameters
	query := ConvertToBsonMFromListParams(params)
	sort, err := GetSortOptionFromString(params.Sort)
	if err != nil {
		return GetErrorListResponse[models.TaskDTO](err)
	}
	skip, limit := GetSkipLimitFromListParams(params)

	// total
	total, err := service.NewModelService[models.TaskDTO]().Count(query)
	if err != nil {
		return GetErrorListResponse[models.TaskDTO](err)
	}

	// check total
	if total == 0 {
		return GetEmptyListResponse[models.TaskDTO]()
	}

	// aggregation pipelines
	pipelines := service.GetPaginationPipeline(query, sort, skip, limit)
	pipelines = addTaskPipelines(pipelines)

	// perform query
	var tasks []models.TaskDTO
	err = service.GetCollection[models.TaskDTO]().Aggregate(pipelines, nil).All(&tasks)
	if err != nil {
		return GetErrorListResponse[models.TaskDTO](err)
	}

	return GetListResponse(tasks, total)
}

type DeleteTaskByIdParams struct {
	Id string `path:"id" description:"Task ID" format:"objectid" pattern:"^[0-9a-fA-F]{24}$"`
}

func DeleteTaskById(_ *gin.Context, params *DeleteTaskByIdParams) (response *VoidResponse, err error) {
	id, err := primitive.ObjectIDFromHex(params.Id)
	if err != nil {
		return GetErrorVoidResponse(err)
	}

	// delete in db
	if err := mongo2.RunTransaction(func(context mongo.SessionContext) (err error) {
		// delete task
		_, err = service.NewModelService[models.Task]().GetById(id)
		if err != nil {
			return err
		}
		err = service.NewModelService[models.Task]().DeleteById(id)
		if err != nil {
			return err
		}

		// delete task stat
		_, err = service.NewModelService[models.TaskStat]().GetById(id)
		if err != nil {
			logger.Warnf("delete task stat error: %s", err.Error())
			return nil
		}
		err = service.NewModelService[models.TaskStat]().DeleteById(id)
		if err != nil {
			logger.Warnf("delete task stat error: %s", err.Error())
			return nil
		}

		return nil
	}); err != nil {
		return GetErrorVoidResponse(err)
	}

	// delete task logs
	logPath := filepath.Join(utils.GetTaskLogPath(), id.Hex())
	if err := os.RemoveAll(logPath); err != nil {
		logger.Warnf("failed to remove task log directory: %s", logPath)
	}

	return GetVoidResponse()
}

type DeleteTaskListParams struct {
	Ids []string `json:"ids"`
}

func DeleteTaskList(_ *gin.Context, params *DeleteTaskListParams) (response *VoidResponse, err error) {
	var ids []primitive.ObjectID
	for _, id := range params.Ids {
		id, err := primitive.ObjectIDFromHex(id)
		if err != nil {
			return GetErrorVoidResponse(err)
		}
		ids = append(ids, id)
	}

	if err := mongo2.RunTransaction(func(context mongo.SessionContext) error {
		// delete tasks
		if err := service.NewModelService[models.Task]().DeleteMany(bson.M{
			"_id": bson.M{
				"$in": ids,
			},
		}); err != nil {
			return err
		}

		// delete task stats
		if err := service.NewModelService[models.Task]().DeleteMany(bson.M{
			"_id": bson.M{
				"$in": ids,
			},
		}); err != nil {
			logger.Warnf("delete task stat error: %s", err.Error())
			return nil
		}

		return nil
	}); err != nil {
		return GetErrorVoidResponse(err)
	}

	// delete tasks logs
	wg := sync.WaitGroup{}
	wg.Add(len(ids))
	for _, id := range ids {
		go func(taskId primitive.ObjectID) {
			// delete task logs
			logPath := filepath.Join(utils.GetTaskLogPath(), taskId.Hex())
			if err := os.RemoveAll(logPath); err != nil {
				logger.Warnf("failed to remove task log directory: %s", logPath)
			}
			wg.Done()
		}(id)
	}
	wg.Wait()

	return GetVoidResponse()
}

type PostTaskRunParams struct {
	SpiderId string   `json:"spider_id" validate:"required"`
	Mode     string   `json:"mode"`
	NodeIds  []string `json:"node_ids"`
	Cmd      string   `json:"cmd"`
	Param    string   `json:"param"`
	Priority int      `json:"priority"`
}

func PostTaskRun(c *gin.Context, params *PostTaskRunParams) (response *Response[[]primitive.ObjectID], err error) {
	spiderId, err := primitive.ObjectIDFromHex(params.SpiderId)
	if err != nil {
		return GetErrorResponse[[]primitive.ObjectID](err)
	}

	var nodeIds []primitive.ObjectID
	if params.NodeIds != nil {
		for _, nodeId := range params.NodeIds {
			nodeId, err := primitive.ObjectIDFromHex(nodeId)
			if err != nil {
				return GetErrorResponse[[]primitive.ObjectID](err)
			}
			nodeIds = append(nodeIds, nodeId)
		}
	}

	// spider
	s, err := service.NewModelService[models.Spider]().GetById(spiderId)
	if err != nil {
		return GetErrorResponse[[]primitive.ObjectID](err)
	}

	// options
	opts := &interfaces.SpiderRunOptions{
		Mode:     params.Mode,
		NodeIds:  nodeIds,
		Cmd:      params.Cmd,
		Param:    params.Param,
		Priority: params.Priority,
	}

	// user
	if u := GetUserFromContext(c); u != nil {
		opts.UserId = u.Id
	}

	// run
	adminSvc := admin.GetSpiderAdminService()
	taskIds, err := adminSvc.Schedule(s.Id, opts)
	if err != nil {
		return GetErrorResponse[[]primitive.ObjectID](err)
	}

	return GetDataResponse(taskIds)
}

type PostTaskRestartParams struct {
	Id string `path:"id" description:"Task ID" format:"objectid" pattern:"^[0-9a-fA-F]{24}$"`
}

func PostTaskRestart(c *gin.Context, params *PostTaskRestartParams) (response *Response[[]primitive.ObjectID], err error) {
	// id
	id, err := primitive.ObjectIDFromHex(params.Id)
	if err != nil {
		return GetErrorResponse[[]primitive.ObjectID](err)
	}

	// task
	t, err := service.NewModelService[models.Task]().GetById(id)
	if err != nil {
		return GetErrorResponse[[]primitive.ObjectID](err)
	}

	// options
	opts := &interfaces.SpiderRunOptions{
		Mode:     t.Mode,
		NodeIds:  t.NodeIds,
		Cmd:      t.Cmd,
		Param:    t.Param,
		Priority: t.Priority,
	}

	// normalize options for tasks with assigned node
	if !t.NodeId.IsZero() {
		opts.NodeIds = []primitive.ObjectID{t.NodeId}
		opts.Mode = constants.RunTypeSelectedNodes
	}

	// user
	if u := GetUserFromContext(c); u != nil {
		opts.UserId = u.Id
	}

	// run
	adminSvc := admin.GetSpiderAdminService()
	taskIds, err := adminSvc.Schedule(t.SpiderId, opts)
	if err != nil {
		return GetErrorResponse[[]primitive.ObjectID](err)
	}

	return GetDataResponse(taskIds)
}

type PostTaskCancelParams struct {
	Id    string `path:"id" description:"Task ID" format:"objectid" pattern:"^[0-9a-fA-F]{24}$"`
	Force bool   `json:"force,omitempty" description:"Force cancel" default:"false"`
}

func PostTaskCancel(c *gin.Context, params *PostTaskCancelParams) (response *VoidResponse, err error) {
	// id
	id, err := primitive.ObjectIDFromHex(params.Id)
	if err != nil {
		return GetErrorVoidResponse(err)
	}

	// task
	t, err := service.NewModelService[models.Task]().GetById(id)
	if err != nil {
		return GetErrorVoidResponse(err)
	}

	// validate - only check if task is cancellable when force is false
	if !params.Force && !utils.IsCancellable(t.Status) {
		return GetErrorVoidResponse(errors.New("task is not cancellable"))
	}

	u := GetUserFromContext(c)

	// cancel
	schedulerSvc := scheduler.GetTaskSchedulerService()
	err = schedulerSvc.Cancel(id, u.Id, params.Force)
	if err != nil {
		return GetErrorVoidResponse(err)
	}

	return GetVoidResponse()
}

type GetTaskLogsParams struct {
	Id     string `path:"id" description:"Task ID" format:"objectid" pattern:"^[0-9a-fA-F]{24}$"`
	Page   int    `query:"page" description:"Page (default: 1)" default:"1" minimum:"1"`
	Size   int    `query:"size" description:"Size (default: 10000)" default:"10000" minimum:"1"`
	Latest bool   `query:"latest" description:"Whether to get latest logs (default: true)" default:"true"`
}

func GetTaskLogs(_ *gin.Context, params *GetTaskLogsParams) (response *ListResponse[string], err error) {
	// id
	id, err := primitive.ObjectIDFromHex(params.Id)
	if err != nil {
		return GetErrorListResponse[string](err)
	}

	// Get total count first for pagination
	logDriver := log.GetFileLogDriver()
	total, err := logDriver.Count(id.Hex(), "")
	if err != nil {
		return GetErrorListResponse[string](err)
	}

	// Skip calculation depends on whether we're getting latest logs or not
	skip := (params.Page - 1) * params.Size
	if params.Latest {
		// For latest logs (tail mode), skip is from the end
		// No adjustment needed as our implementation handles it correctly
	} else {
		// For oldest logs (normal mode), skip is from the beginning
		// No adjustment needed as it's already the default behavior
	}

	// Get logs
	logs, err := logDriver.Find(id.Hex(), "", skip, params.Size, params.Latest)
	if err != nil {
		if strings.HasSuffix(err.Error(), "Status:404 Not Found") {
			return GetListResponse[string](nil, 0)
		}
		return GetErrorListResponse[string](err)
	}

	return GetListResponse(logs, total)
}

type GetTaskResultsParams struct {
	Id   string `path:"id" description:"Task ID" format:"objectid" pattern:"^[0-9a-fA-F]{24}$"`
	Page int    `query:"page" description:"Page" default:"1" minimum:"1"`
	Size int    `query:"size" description:"Size" default:"10" minimum:"1"`
}

func GetTaskResults(c *gin.Context, params *GetSpiderResultsParams) (response *ListResponse[bson.M], err error) {
	id, err := primitive.ObjectIDFromHex(params.Id)
	if err != nil {
		return GetErrorListResponse[bson.M](errors.BadRequestf("invalid id format"))
	}

	t, err := service.NewModelService[models.Task]().GetById(id)
	if err != nil {
		return GetErrorListResponse[bson.M](err)
	}

	s, err := service.NewModelService[models.Spider]().GetById(t.SpiderId)
	if err != nil {
		return GetErrorListResponse[bson.M](err)
	}

	query := ConvertToBsonMFromContext(c)
	if query == nil {
		query = bson.M{}
	}
	query["_tid"] = t.Id

	col := mongo2.GetMongoCol(s.ColName)

	var results []bson.M
	err = col.Find(query, mongo2.GetMongoOpts(&mongo2.ListOptions{
		Sort:  []mongo2.ListSort{{"_id", mongo2.SortDirectionDesc}},
		Skip:  params.Size * (params.Page - 1),
		Limit: params.Size,
	})).All(&results)
	if err != nil {
		return GetErrorListResponse[bson.M](err)
	}

	total, err := mongo2.GetMongoCol(s.ColName).Count(query)
	if err != nil {
		return GetErrorListResponse[bson.M](err)
	}

	return GetListResponse(results, total)
}

func addTaskPipelines(pipelines []bson.D) []bson.D {
	pipelines = append(pipelines, service.GetJoinPipeline[models.TaskStat]("_id", "_id", "_stat")...)
	pipelines = append(pipelines, service.GetDefaultJoinPipeline[models.Node]()...)
	pipelines = append(pipelines, service.GetDefaultJoinPipeline[models.Spider]()...)
	pipelines = append(pipelines, service.GetDefaultJoinPipeline[models.Schedule]()...)
	return pipelines
}
