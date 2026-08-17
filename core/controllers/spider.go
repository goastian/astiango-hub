package controllers

import (
	"github.com/goastian/astiango-hub/core/entity"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"sync"

	"github.com/goastian/astiango-hub/core/constants"
	"github.com/goastian/astiango-hub/core/fs"
	"github.com/goastian/astiango-hub/core/interfaces"
	"github.com/goastian/astiango-hub/core/models/models"
	"github.com/goastian/astiango-hub/core/models/service"
	mongo2 "github.com/goastian/astiango-hub/core/mongo"
	"github.com/goastian/astiango-hub/core/spider"
	"github.com/goastian/astiango-hub/core/spider/admin"
	"github.com/goastian/astiango-hub/core/utils"
	"github.com/gin-gonic/gin"
	"github.com/juju/errors"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

// GetSpiderById handles getting a spider by ID
func GetSpiderById(_ *gin.Context, params *GetByIdParams) (response *Response[models.SpiderDTO], err error) {
	id, err := primitive.ObjectIDFromHex(params.Id)
	if err != nil {
		return GetErrorResponse[models.SpiderDTO](errors.BadRequestf("invalid id format"))
	}

	// aggregation pipelines
	pipelines := service.GetByIdPipeline(id)
	pipelines = addSpiderPipelines(pipelines)

	// perform query
	var spiders []models.SpiderDTO
	err = service.GetCollection[models.SpiderDTO]().Aggregate(pipelines, nil).All(&spiders)
	if err != nil {
		return GetErrorResponse[models.SpiderDTO](err)
	}

	// check results
	if len(spiders) == 0 {
		return nil, errors.NotFoundf("spider %s not found", params.Id)
	}

	return GetDataResponse(spiders[0])
}

// GetSpiderList handles getting a list of spiders with optional stats
func GetSpiderList(_ *gin.Context, params *GetListParams) (response *ListResponse[models.SpiderDTO], err error) {
	// query parameters
	query := ConvertToBsonMFromListParams(params)
	sort, err := GetSortOptionFromString(params.Sort)
	if err != nil {
		return GetErrorListResponse[models.SpiderDTO](errors.BadRequestf("invalid request parameters: %v", err))
	}
	skip, limit := GetSkipLimitFromListParams(params)

	// total count
	total, err := service.NewModelService[models.SpiderDTO]().Count(query)
	if err != nil {
		return GetErrorListResponse[models.SpiderDTO](err)
	}

	// check total
	if total == 0 {
		return GetEmptyListResponse[models.SpiderDTO]()
	}

	// aggregation pipelines
	pipelines := service.GetPaginationPipeline(query, sort, skip, limit)
	pipelines = addSpiderPipelines(pipelines)

	// perform query
	var spiders []models.SpiderDTO
	err = service.GetCollection[models.Spider]().Aggregate(pipelines, nil).All(&spiders)
	if err != nil {
		return GetErrorListResponse[models.SpiderDTO](err)
	}

	return GetListResponse(spiders, total)
}

// PostSpider handles creating a new spider
func PostSpider(c *gin.Context, params *PostParams[models.Spider]) (response *Response[models.Spider], err error) {
	s := params.Data

	if s.Mode == "" {
		s.Mode = constants.RunTypeRandom
	}
	if s.Priority == 0 {
		s.Priority = 5
	}

	// user
	u := GetUserFromContext(c)

	// add
	s.SetCreated(u.Id)
	s.SetUpdated(u.Id)
	id, err := service.NewModelService[models.Spider]().InsertOne(s)
	if err != nil {
		return GetErrorResponse[models.Spider](err)
	}
	s.SetId(id)

	// add stat
	st := models.SpiderStat{}
	st.SetId(id)
	st.SetCreated(u.Id)
	st.SetUpdated(u.Id)
	_, err = service.NewModelService[models.SpiderStat]().InsertOne(st)
	if err != nil {
		return GetErrorResponse[models.Spider](err)
	}

	// create folder
	fsSvc, err := getSpiderFsSvcById(id)
	if err != nil {
		return GetErrorResponse[models.Spider](err)
	}
	err = fsSvc.CreateDir(".")
	if err != nil {
		return GetErrorResponse[models.Spider](err)
	}

	// create template if available
	if utils.IsPro() && s.Template != "" {
		if templateSvc := spider.GetSpiderTemplateRegistryService(); templateSvc != nil {
			err = templateSvc.CreateTemplate(s.Id)
			if err != nil {
				return GetErrorResponse[models.Spider](err)
			}
		}
	}

	return GetDataResponse(s)
}

// DeleteSpiderById handles deleting a spider by ID
func DeleteSpiderById(_ *gin.Context, params *DeleteByIdParams) (response *Response[models.Spider], err error) {
	id, err := primitive.ObjectIDFromHex(params.Id)
	if err != nil {
		return GetErrorResponse[models.Spider](errors.BadRequestf("invalid id format"))
	}

	s, err := service.NewModelService[models.Spider]().GetById(id)
	if err != nil {
		return GetErrorResponse[models.Spider](errors.NotFoundf("spider not found"))
	}

	if err := mongo2.RunTransaction(func(context mongo.SessionContext) (err error) {
		// delete spider
		err = service.NewModelService[models.Spider]().DeleteById(id)
		if err != nil {
			return err
		}

		// delete spider stat
		err = service.NewModelService[models.SpiderStat]().DeleteById(id)
		if err != nil {
			return err
		}

		// related tasks
		tasks, err := service.NewModelService[models.Task]().GetMany(bson.M{"spider_id": id}, nil)
		if err != nil {
			return err
		}

		if len(tasks) == 0 {
			return nil
		}

		// task ids
		var taskIds []primitive.ObjectID
		for _, t := range tasks {
			taskIds = append(taskIds, t.Id)
		}

		// delete related tasks
		err = service.NewModelService[models.Task]().DeleteMany(bson.M{"_id": bson.M{"$in": taskIds}})
		if err != nil {
			return err
		}

		// delete related task stats
		err = service.NewModelService[models.TaskStat]().DeleteMany(bson.M{"_id": bson.M{"$in": taskIds}})
		if err != nil {
			return err
		}

		// delete tasks logs
		wg := sync.WaitGroup{}
		wg.Add(len(taskIds))
		for _, id := range taskIds {
			go func(id string) {
				// delete task logs
				logPath := filepath.Join(utils.GetTaskLogPath(), id)
				if err := os.RemoveAll(logPath); err != nil {
					logger.Warnf("failed to remove task log directory: %s", logPath)
				}
				wg.Done()
			}(id.Hex())
		}
		wg.Wait()

		return nil
	}); err != nil {
		return GetErrorResponse[models.Spider](err)
	}

	// Delete spider directory synchronously to prevent goroutine leaks
	if !s.GitId.IsZero() {
		// delete spider directory
		fsSvc, err := getSpiderFsSvcById(s.Id)
		if err != nil {
			logger.Errorf("failed to get spider fs service: %v", err)
		} else {
			err = fsSvc.Delete(".")
			if err != nil {
				logger.Errorf("failed to delete spider directory: %v", err)
			}
		}
	}

	return GetDataResponse(models.Spider{})
}

type DeleteSpiderListParams struct {
	Ids []string `json:"ids" validate:"required"`
}

// DeleteSpiderList handles deleting multiple spiders
func DeleteSpiderList(_ *gin.Context, params *DeleteSpiderListParams) (response *Response[models.Spider], err error) {
	var ids []primitive.ObjectID
	for _, id := range params.Ids {
		_id, err := primitive.ObjectIDFromHex(id)
		if err != nil {
			return GetErrorResponse[models.Spider](errors.BadRequestf("invalid id format"))
		}
		ids = append(ids, _id)
	}

	// Fetch spiders before deletion
	spiders, err := service.NewModelService[models.Spider]().GetMany(bson.M{
		"_id": bson.M{
			"$in": ids,
		},
	}, nil)
	if err != nil {
		return nil, err
	}

	if err := mongo2.RunTransaction(func(context mongo.SessionContext) (err error) {
		// delete spiders
		if err := service.NewModelService[models.Spider]().DeleteMany(bson.M{
			"_id": bson.M{
				"$in": ids,
			},
		}); err != nil {
			return err
		}

		// delete spider stats
		if err := service.NewModelService[models.SpiderStat]().DeleteMany(bson.M{
			"_id": bson.M{
				"$in": ids,
			},
		}); err != nil {
			return err
		}

		// related tasks
		tasks, err := service.NewModelService[models.Task]().GetMany(bson.M{"spider_id": bson.M{"$in": ids}}, nil)
		if err != nil {
			return err
		}

		if len(tasks) == 0 {
			return nil
		}

		// task ids
		var taskIds []primitive.ObjectID
		for _, t := range tasks {
			taskIds = append(taskIds, t.Id)
		}

		// delete related tasks
		if err := service.NewModelService[models.Task]().DeleteMany(bson.M{"_id": bson.M{"$in": taskIds}}); err != nil {
			return err
		}

		// delete related task stats
		if err := service.NewModelService[models.TaskStat]().DeleteMany(bson.M{"_id": bson.M{"$in": taskIds}}); err != nil {
			return err
		}

		// delete tasks logs
		wg := sync.WaitGroup{}
		wg.Add(len(taskIds))
		for _, id := range taskIds {
			go func(id string) {
				// delete task logs
				logPath := filepath.Join(utils.GetTaskLogPath(), id)
				if err := os.RemoveAll(logPath); err != nil {
					logger.Warnf("failed to remove task log directory: %s", logPath)
				}
				wg.Done()
			}(id.Hex())
		}
		wg.Wait()

		return nil
	}); err != nil {
		return GetErrorResponse[models.Spider](err)
	}

	// Delete spider directories synchronously to prevent goroutine leaks
	wg := sync.WaitGroup{}
	semaphore := make(chan struct{}, 5) // Limit concurrent operations
	
	for i := range spiders {
		// Skip spider with git
		if !spiders[i].GitId.IsZero() {
			continue
		}

		wg.Add(1)
		semaphore <- struct{}{} // Acquire semaphore
		
		func(s *models.Spider) {
			defer func() {
				<-semaphore // Release semaphore
				wg.Done()
			}()

			// Delete spider directory
			fsSvc, err := getSpiderFsSvcById(s.Id)
			if err != nil {
				logger.Errorf("failed to get spider fs service: %v", err)
				return
			}
			err = fsSvc.Delete(".")
			if err != nil {
				logger.Errorf("failed to delete spider directory: %v", err)
				return
			}
		}(&spiders[i])
	}
	wg.Wait()

	return GetDataResponse(models.Spider{})
}

type GetSpiderFilesParams struct {
	Id   string `path:"id" description:"Spider ID" format:"objectid" pattern:"^[0-9a-fA-F]{24}$"`
	Path string `query:"path" description:"Directory path"`
}

func GetSpiderFiles(c *gin.Context, params *GetSpiderFilesParams) (response *Response[[]entity.FsFileInfo], err error) {
	rootPath, err := getSpiderRootPathByContext(c)
	if err != nil {
		return GetErrorResponse[[]entity.FsFileInfo](err)
	}
	return GetBaseFileListDir(rootPath, params.Path)
}

type GetSpiderFileContentParams struct {
	Id   string `path:"id" description:"Spider ID" format:"objectid" pattern:"^[0-9a-fA-F]{24}$"`
	Path string `query:"path" description:"File path"`
}

func GetSpiderFileContent(c *gin.Context, params *GetSpiderFileContentParams) (response *Response[string], err error) {
	rootPath, err := getSpiderRootPathByContext(c)
	if err != nil {
		return GetErrorResponse[string](err)
	}
	return GetBaseFileContent(rootPath, params.Path)
}

type GetSpiderFileInfoParams struct {
	Id   string `path:"id" description:"Spider ID" format:"objectid" pattern:"^[0-9a-fA-F]{24}$"`
	Path string `query:"path" description:"File path"`
}

func GetSpiderFileInfo(c *gin.Context, params *GetSpiderFileInfoParams) (response *Response[*entity.FsFileInfo], err error) {
	rootPath, err := getSpiderRootPathByContext(c)
	if err != nil {
		return GetErrorResponse[*entity.FsFileInfo](err)
	}
	return GetBaseFileInfo(rootPath, params.Path)
}

type PostSpiderSaveFileParams struct {
	Id   string                `path:"id" description:"Spider ID" format:"objectid" pattern:"^[0-9a-fA-F]{24}$"`
	Path string                `json:"path" description:"File path to save"`
	Data string                `json:"data" description:"File content"`
	File *multipart.FileHeader `form:"file"`
}

func PostSpiderSaveFile(c *gin.Context, params *PostSpiderSaveFileParams) (response *VoidResponse, err error) {
	rootPath, err := getSpiderRootPathByContext(c)
	if err != nil {
		return GetErrorVoidResponse(err)
	}
	if c.GetHeader("Content-Type") == "application/json" {
		return PostBaseFileSaveOne(rootPath, params.Path, params.Data)
	} else {
		return PostBaseFileSaveOneForm(rootPath, params.Path, params.File)
	}
}

type PostSpiderSaveFilesParams struct {
	Id              string `path:"id" description:"Spider ID" format:"objectid" pattern:"^[0-9a-fA-F]{24}$"`
	TargetDirectory string `form:"targetDirectory" description:"Target directory path"`
}

func PostSpiderSaveFiles(c *gin.Context, params *PostSpiderSaveFilesParams) (response *VoidResponse, err error) {
	rootPath, err := getSpiderRootPathByContext(c)
	if err != nil {
		return GetErrorVoidResponse(err)
	}
	form, err := c.MultipartForm()
	if err != nil {
		return GetErrorVoidResponse(err)
	}
	return PostBaseFileSaveMany(filepath.Join(rootPath, params.TargetDirectory), form)
}

// PostSpiderSaveFilesGin handles saving multiple files to a spider's directory via Gin context TODO: temporary solution
func PostSpiderSaveFilesGin(c *gin.Context) {
	targetDirectory := c.PostForm("targetDirectory")
	rootPath, err := getSpiderRootPathByContext(c)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	form, err := c.MultipartForm()
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	_, err = PostBaseFileSaveMany(filepath.Join(rootPath, targetDirectory), form)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	HandleSuccessWithData(c, nil)
}

type PostSpiderSaveDirParams struct {
	Id   string `path:"id" description:"Spider ID" format:"objectid" pattern:"^[0-9a-fA-F]{24}$"`
	Path string `json:"path" description:"File path to save"`
}

func PostSpiderSaveDir(c *gin.Context, params *PostSpiderSaveDirParams) (response *VoidResponse, err error) {
	rootPath, err := getSpiderRootPathByContext(c)
	if err != nil {
		return GetErrorVoidResponse(err)
	}
	return PostBaseFileSaveDir(rootPath, params.Path)
}

type PostSpiderRenameFileParams struct {
	Id      string `path:"id" description:"Spider ID" format:"objectid" pattern:"^[0-9a-fA-F]{24}$"`
	Path    string `json:"path" description:"File path to rename"`
	NewPath string `json:"newPath" description:"New file path"`
}

func PostSpiderRenameFile(c *gin.Context, params *PostSpiderRenameFileParams) (response *VoidResponse, err error) {
	rootPath, err := getSpiderRootPathByContext(c)
	if err != nil {
		return GetErrorVoidResponse(err)
	}
	return PostBaseFileRename(rootPath, params.Path, params.NewPath)
}

type DeleteSpiderFileParams struct {
	Id   string `path:"id" description:"Spider ID" format:"objectid" pattern:"^[0-9a-fA-F]{24}$"`
	Path string `json:"path" description:"File path to delete"`
}

func DeleteSpiderFile(c *gin.Context, params *DeleteSpiderFileParams) (response *VoidResponse, err error) {
	rootPath, err := getSpiderRootPathByContext(c)
	if err != nil {
		return GetErrorVoidResponse(err)
	}
	return DeleteBaseFile(rootPath, params.Path)
}

type PostSpiderCopyFileParams struct {
	Id      string `path:"id" description:"Spider ID" format:"objectid" pattern:"^[0-9a-fA-F]{24}$"`
	Path    string `json:"path" description:"File path to copy"`
	NewPath string `json:"new_path" description:"New file path"`
}

func PostSpiderCopyFile(c *gin.Context, params *PostSpiderCopyFileParams) (response *VoidResponse, err error) {
	rootPath, err := getSpiderRootPathByContext(c)
	if err != nil {
		return GetErrorVoidResponse(err)
	}
	return PostBaseFileCopy(rootPath, params.Path, params.NewPath)
}

type PostSpiderExportParams struct {
	Id string `path:"id" description:"Spider ID" format:"objectid" pattern:"^[0-9a-fA-F]{24}$"`
}

func PostSpiderExport(c *gin.Context, _ *PostSpiderExportParams) (err error) {
	rootPath, err := getSpiderRootPathByContext(c)
	if err != nil {
		return err
	}
	return PostBaseFileExport(rootPath, c)
}

type PostSpiderRunParams struct {
	Id         string   `path:"id" description:"Spider ID" format:"objectid" pattern:"^[0-9a-fA-F]{24}$"`
	Mode       string   `json:"mode" description:"Run mode: random,all,selected-nodes" default:"random" enum:"random,all,selected-nodes"`
	NodeIds    []string `json:"node_ids" description:"Node IDs, used in selected-nodes mode"`
	Cmd        string   `json:"cmd" description:"Command"`
	Param      string   `json:"param" description:"Parameters"`
	ScheduleId string   `json:"schedule_id" description:"Schedule ID" format:"objectid" pattern:"^[0-9a-fA-F]{24}$"`
	Priority   int      `json:"priority" description:"Priority" default:"5" minimum:"1" maximum:"10"`
}

func PostSpiderRun(c *gin.Context, params *PostSpiderRunParams) (response *Response[[]primitive.ObjectID], err error) {
	id, err := primitive.ObjectIDFromHex(params.Id)
	if err != nil {
		return GetErrorResponse[[]primitive.ObjectID](errors.BadRequestf("invalid id format"))
	}

	// get spider
	s, err := service.NewModelService[models.Spider]().GetById(id)
	if err != nil {
		return GetErrorResponse[[]primitive.ObjectID](errors.NotFoundf("spider not found"))
	}

	// options
	var nodeIds []primitive.ObjectID
	if len(params.NodeIds) > 0 {
		for _, id := range params.NodeIds {
			nodeId, err := primitive.ObjectIDFromHex(id)
			if err != nil {
				return GetErrorResponse[[]primitive.ObjectID](errors.BadRequestf("invalid node id format"))
			}
			nodeIds = append(nodeIds, nodeId)
		}
	}
	var scheduleId primitive.ObjectID
	if params.ScheduleId != "" {
		scheduleId, err = primitive.ObjectIDFromHex(params.ScheduleId)
		if err != nil {
			return GetErrorResponse[[]primitive.ObjectID](errors.BadRequestf("invalid schedule id format"))
		}
	}
	opts := interfaces.SpiderRunOptions{
		Mode:       params.Mode,
		NodeIds:    nodeIds,
		Cmd:        params.Cmd,
		Param:      params.Param,
		ScheduleId: scheduleId,
		Priority:   params.Priority,
	}
	if opts.Mode == "" {
		opts.Mode = s.Mode
	}
	if opts.Cmd == "" {
		opts.Cmd = s.Cmd
	}
	if opts.Param == "" {
		opts.Param = s.Param
	}
	if opts.Priority == 0 {
		opts.Priority = s.Priority
	}

	// user
	if u := GetUserFromContext(c); u != nil {
		opts.UserId = u.GetId()
	}

	// schedule tasks
	taskIds, err := admin.GetSpiderAdminService().Schedule(id, &opts)
	if err != nil {
		return GetErrorResponse[[]primitive.ObjectID](err)
	}

	return GetDataResponse(taskIds)
}

type GetSpiderResultsParams struct {
	Id   string `path:"id" description:"Spider ID" format:"objectid" pattern:"^[0-9a-fA-F]{24}$"`
	Page int    `query:"page" description:"Page" default:"1" minimum:"1"`
	Size int    `query:"size" description:"Size" default:"10" minimum:"1"`
}

func GetSpiderResults(c *gin.Context, params *GetSpiderResultsParams) (response *ListResponse[bson.M], err error) {
	id, err := primitive.ObjectIDFromHex(params.Id)
	if err != nil {
		return GetErrorListResponse[bson.M](errors.BadRequestf("invalid id format"))
	}

	s, err := service.NewModelService[models.Spider]().GetById(id)
	if err != nil {
		return GetErrorListResponse[bson.M](err)
	}

	query := ConvertToBsonMFromContext(c)
	if query == nil {
		query = bson.M{}
	}
	query["_sid"] = s.Id

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

func getSpiderFsSvc(s *models.Spider) (svc *fs.Service, err error) {
	workspacePath := utils.GetWorkspace()
	fsSvc := fs.NewFsService(filepath.Join(workspacePath, s.Id.Hex()))

	return fsSvc, nil
}

func getSpiderFsSvcById(id primitive.ObjectID) (svc *fs.Service, err error) {
	s, err := service.NewModelService[models.Spider]().GetById(id)
	if err != nil {
		logger.Errorf("failed to get spider: %v", err)
		return nil, err
	}
	return getSpiderFsSvc(s)
}

func getSpiderRootPathByContext(c *gin.Context) (rootPath string, err error) {
	id, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		return "", err
	}
	s, err := service.NewModelService[models.Spider]().GetById(id)
	if err != nil {
		return "", err
	}
	return utils.GetSpiderRootPath(s)
}

func addSpiderPipelines(pipelines []bson.D) []bson.D {
	pipelines = append(pipelines, service.GetJoinPipeline[models.SpiderStat]("_id", "_id", "_stat")...)
	pipelines = append(pipelines, service.GetJoinPipeline[models.Task]("_stat.last_task_id", "_id", "_last_task")...)
	pipelines = append(pipelines, service.GetJoinPipeline[models.TaskStat]("_last_task._id", "_id", "_last_task._stat")...)
	pipelines = append(pipelines, service.GetDefaultJoinPipeline[models.Project]()...)
	if utils.IsPro() {
		pipelines = append(pipelines, service.GetDefaultJoinPipeline[models.Git]()...)
		pipelines = append(pipelines, service.GetDefaultJoinPipeline[models.Database]()...)
	}
	return pipelines
}
