package controllers

import (
	"github.com/goastian/astiango-hub/core/entity"
	"github.com/goastian/astiango-hub/core/mongo"
	"github.com/gin-gonic/gin"
	"github.com/juju/errors"
	"go.mongodb.org/mongo-driver/bson"
	mongo2 "go.mongodb.org/mongo-driver/mongo"
)

type GetFilterColFieldOptionsParams struct {
	Col    string `path:"col" validate:"required"`
	Filter string `query:"filter" description:"Filter query"`
}

func GetFilterColFieldOptions(c *gin.Context, params *GetFilterColFieldOptionsParams) (response *Response[[]entity.FilterSelectOption], err error) {
	return GetFilterColFieldOptionsWithValueLabel(c, &GetFilterColFieldOptionsWithValueLabelParams{
		Col:    params.Col,
		Filter: params.Filter,
	})
}

type GetFilterColFieldOptionsWithValueParams struct {
	Col    string `path:"col" validate:"required"`
	Value  string `path:"value"`
	Filter string `query:"filter" description:"Filter query"`
}

func GetFilterColFieldOptionsWithValue(c *gin.Context, params *GetFilterColFieldOptionsWithValueParams) (response *Response[[]entity.FilterSelectOption], err error) {
	return GetFilterColFieldOptionsWithValueLabel(c, &GetFilterColFieldOptionsWithValueLabelParams{
		Col:    params.Col,
		Value:  params.Value,
		Filter: params.Filter,
	})
}

type GetFilterColFieldOptionsWithValueLabelParams struct {
	Col    string `path:"col" validate:"required"`
	Value  string `path:"value"`
	Label  string `path:"label"`
	Filter string `query:"filter" description:"Filter query"`
}

func GetFilterColFieldOptionsWithValueLabel(_ *gin.Context, params *GetFilterColFieldOptionsWithValueLabelParams) (response *Response[[]entity.FilterSelectOption], err error) {
	value := params.Value
	if value == "" {
		value = "_id"
	}
	label := params.Label
	if label == "" {
		label = "name"
	}

	pipelines := mongo2.Pipeline{}
	if params.Filter != "" {
		query := ConvertToFilter(params.Filter)
		pipelines = append(pipelines, bson.D{{"$match", query}})
	}
	pipelines = append(
		pipelines,
		bson.D{
			{
				"$group",
				bson.M{
					"_id": bson.M{
						"value": "$" + value,
						"label": "$" + label,
					},
				},
			},
		},
	)
	pipelines = append(
		pipelines,
		bson.D{
			{
				"$project",
				bson.M{
					"value": "$_id.value",
					"label": bson.M{"$toString": "$_id.label"},
				},
			},
		},
	)
	pipelines = append(
		pipelines,
		bson.D{
			{
				"$sort",
				bson.D{
					{"value", 1},
				},
			},
		},
	)

	var options []entity.FilterSelectOption
	if err := mongo.GetMongoCol(params.Col).Aggregate(pipelines, nil).All(&options); err != nil {
		return GetErrorResponse[[]entity.FilterSelectOption](errors.Trace(err))
	}

	return GetDataResponse(options)
}
