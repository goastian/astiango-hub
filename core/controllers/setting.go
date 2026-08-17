package controllers

import (
	"errors"
	"github.com/goastian/astiango-hub/core/models/models"
	"github.com/goastian/astiango-hub/core/models/service"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type GetSettingParams struct {
	Key string `path:"key" description:"Setting key" validate:"required"`
}

func GetSetting(_ *gin.Context, params *GetSettingParams) (response *Response[models.Setting], err error) {
	// setting
	s, err := service.NewModelService[models.Setting]().GetOne(bson.M{"key": params.Key}, nil)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return GetDataResponse(models.Setting{})
		}
		return GetErrorResponse[models.Setting](err)
	}

	return GetDataResponse(*s)
}

type PostSettingParams struct {
	Key  string         `path:"key" description:"Setting key" validate:"required"`
	Data models.Setting `json:"data" description:"The data to create" validate:"required"`
}

func PostSetting(c *gin.Context, params *PostSettingParams) (response *Response[models.Setting], err error) {
	s := params.Data
	if s.Key == "" {
		s.Key = params.Key
	}

	u := GetUserFromContext(c)
	s.SetCreated(u.Id)
	s.SetUpdated(u.Id)

	id, err := service.NewModelService[models.Setting]().InsertOne(s)
	if err != nil {
		return GetErrorResponse[models.Setting](err)
	}
	s.Id = id

	return GetDataResponse(s)
}

type PutSettingParams struct {
	Key  string         `path:"key" description:"Setting key" validate:"required"`
	Data models.Setting `json:"data" description:"The data to update" validate:"required"`
}

func PutSetting(c *gin.Context, params *PutSettingParams) (response *Response[models.Setting], err error) {
	modelSvc := service.NewModelService[models.Setting]()

	// setting
	existingSetting, err := modelSvc.GetOne(bson.M{"key": params.Key}, nil)
	if err != nil {
		return GetErrorResponse[models.Setting](err)
	}

	u := GetUserFromContext(c)

	// save
	existingSetting.Value = params.Data.Value
	existingSetting.SetUpdated(u.Id)
	err = modelSvc.ReplaceOne(bson.M{"key": params.Key}, *existingSetting)
	if err != nil {
		return GetErrorResponse[models.Setting](err)
	}

	return GetDataResponse(*existingSetting)
}
