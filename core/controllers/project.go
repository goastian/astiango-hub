package controllers

import (
	"github.com/goastian/astiango-hub/core/models/models"
	"github.com/goastian/astiango-hub/core/models/service"
	"github.com/gin-gonic/gin"
	"github.com/juju/errors"
	"go.mongodb.org/mongo-driver/bson"
)

func GetProjectList(_ *gin.Context, params *GetListParams) (response *ListResponse[models.ProjectDTO], err error) {
	query := ConvertToBsonMFromListParams(params)
	sort, err := GetSortOptionFromString(params.Sort)
	if err != nil {
		return GetErrorListResponse[models.ProjectDTO](errors.BadRequestf("invalid request parameters: %v", err))
	}
	skip, limit := GetSkipLimitFromListParams(params)

	// total count
	total, err := service.NewModelService[models.Project]().Count(query)
	if err != nil {
		return GetErrorListResponse[models.ProjectDTO](err)
	}

	// check total
	if total == 0 {
		return GetEmptyListResponse[models.ProjectDTO]()
	}

	// aggregation pipelines
	pipelines := service.GetPaginationPipeline(query, sort, skip, limit)
	pipelines = addProjectPipelines(pipelines)

	// perform query
	var projects []models.ProjectDTO
	err = service.GetCollection[models.Project]().Aggregate(pipelines, nil).All(&projects)
	if err != nil {
		return GetErrorListResponse[models.ProjectDTO](err)
	}

	return GetListResponse[models.ProjectDTO](projects, total)
}

func addProjectPipelines(pipelines []bson.D) []bson.D {
	pipelines = append(pipelines, service.GetLookupPipeline[models.Spider]("_id", "project_id", "_spiders"))
	return pipelines
}
