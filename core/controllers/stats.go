package controllers

import (
	"errors"
	"time"

	"github.com/goastian/astiango-hub/core/models/service"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/goastian/astiango-hub/core/constants"
	"github.com/goastian/astiango-hub/core/entity"
	"github.com/goastian/astiango-hub/core/models/models"
	mongo2 "go.mongodb.org/mongo-driver/mongo"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
)

var statsDefaultQuery = bson.M{
	"created_at": bson.M{
		"$gte": time.Now().Add(-30 * 24 * time.Hour),
	},
}

type GetStatsOverviewParams struct {
	Query bson.M `json:"query" description:"Query"`
}

type GetStatsOverviewResponse struct {
	Nodes      int `json:"nodes" description:"Number of nodes"`
	Projects   int `json:"projects" description:"Number of projects"`
	Spiders    int `json:"spiders" description:"Number of spiders"`
	Schedules  int `json:"schedules" description:"Number of schedules"`
	Tasks      int `json:"tasks" description:"Number of tasks"`
	ErrorTasks int `json:"error_tasks" description:"Number of error tasks"`
	Results    int `json:"results" description:"Number of results"`
	Users      int `json:"users" description:"Number of users"`
}

func GetStatsOverview(_ *gin.Context, params *GetStatsOverviewParams) (response *Response[GetStatsOverviewResponse], err error) {
	query := statsDefaultQuery
	if params.Query != nil {
		query = params.Query
	}

	var data GetStatsOverviewResponse

	// nodes
	data.Nodes, err = service.GetCollection[models.Node]().Count(bson.M{"active": true})
	if err != nil {
		if err.Error() != mongo2.ErrNoDocuments.Error() {
			return nil, err
		}
		data.Nodes = 0
	}

	// projects
	data.Projects, err = service.GetCollection[models.Project]().Count(nil)
	if err != nil {
		if err.Error() != mongo2.ErrNoDocuments.Error() {
			return nil, err
		}
		data.Projects = 0
	}

	// spiders
	data.Spiders, err = service.GetCollection[models.Spider]().Count(nil)
	if err != nil {
		if err.Error() != mongo2.ErrNoDocuments.Error() {
			return nil, err
		}
		data.Spiders = 0
	}

	// schedules
	data.Schedules, err = service.GetCollection[models.Schedule]().Count(nil)
	if err != nil {
		if err.Error() != mongo2.ErrNoDocuments.Error() {
			return nil, err
		}
		data.Schedules = 0
	}

	// tasks
	data.Tasks, err = service.GetCollection[models.Task]().Count(nil)
	if err != nil {
		if err.Error() != mongo2.ErrNoDocuments.Error() {
			return nil, err
		}
		data.Tasks = 0
	}

	// error tasks
	data.ErrorTasks, err = service.GetCollection[models.Task]().Count(bson.M{"status": constants.TaskStatusError})
	if err != nil {
		if err.Error() != mongo2.ErrNoDocuments.Error() {
			return nil, err
		}
		data.ErrorTasks = 0
	}

	// results
	pipeline := mongo2.Pipeline{
		{{"$match", query}},
		{{
			"$group",
			bson.M{
				"_id":     nil,
				"results": bson.M{"$sum": "$result_count"},
			},
		}},
	}
	var res struct {
		Results int `bson:"results"`
	}
	if err := service.GetCollection[models.TaskStat]().Aggregate(pipeline, nil).One(&res); err != nil {
		if !errors.Is(err, mongo2.ErrNoDocuments) {
			return nil, err
		}
		res.Results = 0
	}
	data.Results = res.Results

	// users
	data.Users, err = service.GetCollection[models.User]().Count(nil)
	if err != nil {
		if !errors.Is(err, mongo2.ErrNoDocuments) {
			return nil, err
		}
		data.Users = 0
	}

	return GetDataResponse(data)
}

type GetStatsDailyParams struct {
	Query bson.M `json:"query" description:"Query"`
}

func GetStatsDaily(_ *gin.Context, params *GetStatsDailyParams) (response *Response[[]entity.StatsDailyItem], err error) {
	query := statsDefaultQuery
	if params.Query != nil {
		query = params.Query
	}

	pipeline := mongo2.Pipeline{
		{{
			"$match", query,
		}},
		{{
			"$addFields",
			bson.M{
				"date": bson.M{
					"$dateToString": bson.M{
						"date":     bson.M{"$toDate": "$_id"},
						"format":   "%Y-%m-%d",
						"timezone": "Asia/Shanghai", // TODO: parameterization
					},
				},
			},
		}},
		{{
			"$group",
			bson.M{
				"_id":     "$date",
				"tasks":   bson.M{"$sum": 1},
				"results": bson.M{"$sum": "$result_count"},
			},
		}},
		{{
			"$sort",
			bson.D{{"_id", 1}},
		}},
	}
	var results []entity.StatsDailyItem
	if err := service.GetCollection[models.TaskStat]().Aggregate(pipeline, nil).All(&results); err != nil {
		return nil, err
	}
	return GetDataResponse(results)
}

type GetStatsTasksParams struct {
	Query bson.M `json:"query" description:"Query"`
}

type GetStatsTaskResponse struct {
	ByStatus []GetStatsTaskResponseByStatusItem `json:"by_status"`
	ByNode   []GetStatsTaskResponseByNodeItem   `json:"by_node"`
	BySpider []GetStatsTaskResponseBySpiderItem `json:"by_spider"`
}

type GetStatsTaskResponseByStatusItem struct {
	Status string `json:"status" bson:"status"`
	Tasks  int    `json:"tasks" bson:"tasks"`
}
type GetStatsTaskResponseByNodeItem struct {
	NodeId   primitive.ObjectID `json:"node_id" bson:"node_id"`
	Node     models.Node        `json:"node" bson:"node"`
	NodeName string             `json:"node_name" bson:"node_name"`
	Tasks    int                `json:"tasks" bson:"tasks"`
}
type GetStatsTaskResponseBySpiderItem struct {
	SpiderId   primitive.ObjectID `json:"spider_id" bson:"spider_id"`
	Spider     models.Spider      `json:"spider" bson:"spider"`
	SpiderName string             `json:"spider_name" bson:"spider_name"`
	Tasks      int                `json:"tasks" bson:"tasks"`
}

func GetStatsTasks(_ *gin.Context, params *GetStatsTasksParams) (response *Response[GetStatsTaskResponse], err error) {
	query := statsDefaultQuery
	if params.Query != nil {
		query = params.Query
	}

	var data GetStatsTaskResponse

	// by status
	data.ByStatus, err = getTaskStatsByStatus(query)
	if err != nil {
		return nil, err
	}

	// by node
	data.ByNode, err = getTaskStatsByNode(query)
	if err != nil {
		return nil, err
	}

	// by spider
	data.BySpider, err = getTaskStatsBySpider(query)
	if err != nil {
		return nil, err
	}

	return GetDataResponse(data)
}

func getTaskStatsByStatus(query bson.M) (data []GetStatsTaskResponseByStatusItem, err error) {
	pipeline := mongo2.Pipeline{
		{{"$match", query}},
		{{
			"$group",
			bson.M{
				"_id":   "$status",
				"tasks": bson.M{"$sum": 1},
			},
		}},
		{{
			"$project",
			bson.M{
				"status": "$_id",
				"tasks":  "$tasks",
			},
		}},
	}
	if err := service.GetCollection[models.Task]().Aggregate(pipeline, nil).All(&data); err != nil {
		return nil, err
	}
	return data, nil
}

func getTaskStatsByNode(query bson.M) (data []GetStatsTaskResponseByNodeItem, err error) {
	pipeline := mongo2.Pipeline{
		{{"$match", query}},
		{{
			"$group",
			bson.M{
				"_id":   "$node_id",
				"tasks": bson.M{"$sum": 1},
			},
		}},
		{{
			"$lookup",
			bson.M{
				"from":         models.GetCollectionNameByInstance(models.Node{}),
				"localField":   "_id",
				"foreignField": "_id",
				"as":           "_n",
			},
		}},
		{{
			"$project",
			bson.M{
				"node_id":   "$_id",
				"node":      bson.M{"$arrayElemAt": bson.A{"$_n", 0}},
				"node_name": bson.M{"$arrayElemAt": bson.A{"$_n.name", 0}},
				"tasks":     "$tasks",
			},
		}},
	}
	if err := service.GetCollection[models.Task]().Aggregate(pipeline, nil).All(&data); err != nil {
		return nil, err
	}
	return data, nil
}

func getTaskStatsBySpider(query bson.M) (data []GetStatsTaskResponseBySpiderItem, err error) {
	pipeline := mongo2.Pipeline{
		{{"$match", query}},
		{{
			"$group",
			bson.M{
				"_id":   "$spider_id",
				"tasks": bson.M{"$sum": 1},
			},
		}},
		{{
			"$lookup",
			bson.M{
				"from":         models.GetCollectionNameByInstance(models.Spider{}),
				"localField":   "_id",
				"foreignField": "_id",
				"as":           "_s",
			},
		}},
		{{
			"$project",
			bson.M{
				"spider_id":   "$_id",
				"spider":      bson.M{"$arrayElemAt": bson.A{"$_s", 0}},
				"spider_name": bson.M{"$arrayElemAt": bson.A{"$_s.name", 0}},
				"tasks":       "$tasks",
			},
		}},
		{{"$limit", 10}},
	}
	if err := service.GetCollection[models.Task]().Aggregate(pipeline, nil).All(&data); err != nil {
		return nil, err
	}
	return data, nil
}
