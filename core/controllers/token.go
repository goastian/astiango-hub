package controllers

import (
	"github.com/goastian/astiango-hub/core/models/models"
	"github.com/goastian/astiango-hub/core/models/service"
	"github.com/goastian/astiango-hub/core/mongo"
	"github.com/goastian/astiango-hub/core/user"
	"github.com/gin-gonic/gin"
	"github.com/juju/errors"
	mongo2 "go.mongodb.org/mongo-driver/mongo"
)

type PostTokenParams struct {
	Data models.Token `json:"data" description:"The data to create" validate:"required"`
}

func PostToken(c *gin.Context, params *PostTokenParams) (response *Response[models.Token], err error) {
	t := params.Data
	svc, err := user.GetUserService()
	if err != nil {
		return GetErrorResponse[models.Token](err)
	}

	u := GetUserFromContext(c)
	t.SetCreated(u.Id)
	t.SetUpdated(u.Id)
	t.Token, err = svc.MakeToken(u)
	if err != nil {
		return GetErrorResponse[models.Token](err)
	}

	id, err := service.NewModelService[models.Token]().InsertOne(t)
	if err != nil {
		return GetErrorResponse[models.Token](err)
	}
	t.Id = id

	return GetDataResponse(t)
}

func GetTokenList(c *gin.Context, params *GetListParams) (response *ListResponse[models.Token], err error) {
	// Get current user from context
	u := GetUserFromContext(c)

	// Get filter query
	query := ConvertToBsonMFromListParams(params)

	// Add filter for tokens created by the current user
	query["created_by"] = u.Id

	// Get sort options
	sort, err := GetSortOptionFromString(params.Sort)
	if err != nil {
		return GetErrorListResponse[models.Token](errors.BadRequestf("invalid request parameters: %v", err))
	}

	// Get tokens with pagination
	tokens, err := service.NewModelService[models.Token]().GetMany(query, &mongo.FindOptions{
		Sort:  sort,
		Skip:  params.Size * (params.Page - 1),
		Limit: params.Size,
	})
	if err != nil {
		if err == mongo2.ErrNoDocuments {
			return GetListResponse([]models.Token{}, 0)
		}
		return GetErrorListResponse[models.Token](err)
	}

	// Count total tokens for pagination
	total, err := service.NewModelService[models.Token]().Count(query)
	if err != nil {
		return GetErrorListResponse[models.Token](err)
	}

	return GetListResponse(tokens, total)
}
