package controllers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/goastian/astiango-hub/core/interfaces"
	"github.com/goastian/astiango-hub/core/models/service"
	"github.com/goastian/astiango-hub/core/mongo"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/juju/errors"
	"github.com/loopfz/gadgeto/tonic"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	mongo2 "go.mongodb.org/mongo-driver/mongo"
)

func init() {
	tonic.SetErrorHook(func(context *gin.Context, err error) (int, interface{}) {
		unwrappedErr := errors.Unwrap(err)
		if unwrappedErr != nil {
			err = unwrappedErr
		}
		response := gin.H{
			"error": err.Error(),
		}
		var status int
		if constErr, ok := errors.AsType[errors.ConstError](err); ok {
			switch {
			case errors.Is(constErr, errors.NotFound):
				status = http.StatusNotFound
			case errors.Is(constErr, errors.BadRequest):
				status = http.StatusBadRequest
			case errors.Is(constErr, errors.Unauthorized):
				status = http.StatusUnauthorized
			case errors.Is(constErr, errors.Forbidden):
				status = http.StatusForbidden
			default:
				status = http.StatusInternalServerError
			}
		} else if _, ok := errors.AsType[tonic.BindError](err); ok {
			status = http.StatusBadRequest
		} else {
			status = http.StatusInternalServerError
		}
		return status, response
	})
}

type Action struct {
	Method      string
	Path        string
	Name        string
	Description string
	HandlerFunc interface{}
}

type BaseController[T any] struct {
	modelSvc *service.ModelService[T]
	actions  []Action
}

type GetListParams struct {
	Filter string `query:"filter" description:"Filter query"`
	Sort   string `query:"sort" default:"-_id" description:"Sort options"`
	Page   int    `query:"page" default:"1" description:"Page number" minimum:"1"`
	Size   int    `query:"size" default:"10" description:"Page size" minimum:"1"`
}

func (ctr *BaseController[T]) GetList(_ *gin.Context, params *GetListParams) (response *ListResponse[T], err error) {
	return ctr.GetWithPagination(params)
}

type GetByIdParams struct {
	Id string `path:"id" description:"The ID of the item to get" format:"objectid" pattern:"^[0-9a-fA-F]{24}$"`
}

func (ctr *BaseController[T]) GetById(_ *gin.Context, params *GetByIdParams) (response *Response[T], err error) {
	id, err := primitive.ObjectIDFromHex(params.Id)
	if err != nil {
		return GetErrorResponse[T](errors.BadRequestf("invalid id format"))
	}

	model, err := ctr.modelSvc.GetById(id)
	if err != nil {
		if errors.Is(err, mongo2.ErrNoDocuments) {
			return nil, errors.NotFoundf("item not found")
		}
		return nil, err
	}

	return GetDataResponse(*model)
}

type PostParams[T any] struct {
	Data T `json:"data" description:"The data to create" validate:"required"`
}

func (ctr *BaseController[T]) Post(c *gin.Context, params *PostParams[T]) (response *Response[T], err error) {
	u := GetUserFromContext(c)
	m := any(&params.Data).(interfaces.Model)
	m.SetId(primitive.NewObjectID())
	m.SetCreated(u.Id)
	m.SetUpdated(u.Id)
	col := ctr.modelSvc.GetCol()
	res, err := col.GetCollection().InsertOne(col.GetContext(), m)
	if err != nil {
		return GetErrorResponse[T](err)
	}

	result, err := ctr.modelSvc.GetById(res.InsertedID.(primitive.ObjectID))
	if err != nil {
		return GetErrorResponse[T](err)
	}

	return GetDataResponse(*result)
}

type PutByIdParams[T any] struct {
	Id   string `path:"id" description:"The ID of the item to update" format:"objectid" pattern:"^[0-9a-fA-F]{24}$"`
	Data T      `json:"data" description:"The data to update" validate:"required"`
}

func (ctr *BaseController[T]) PutById(c *gin.Context, params *PutByIdParams[T]) (response *Response[T], err error) {
	id, err := primitive.ObjectIDFromHex(params.Id)
	if err != nil {
		return GetErrorResponse[T](errors.BadRequestf("invalid id format: %v", err))
	}

	u := GetUserFromContext(c)
	m := any(&params.Data).(interfaces.Model)
	m.SetUpdated(u.Id)
	if m.GetId().IsZero() {
		m.SetId(id)
	}

	// Validate
	if err := validator.New().Struct(params.Data); err != nil {
		return GetErrorResponse[T](errors.BadRequestf("invalid data: %v", err))
	}

	if err := ctr.modelSvc.ReplaceById(id, params.Data); err != nil {
		return GetErrorResponse[T](err)
	}

	result, err := ctr.modelSvc.GetById(id)
	if err != nil {
		return GetErrorResponse[T](err)
	}

	return GetDataResponse(*result)
}

type PatchByIdParams[T any] struct {
	Id   string `path:"id" description:"The ID of the item to update" format:"objectid" pattern:"^[0-9a-fA-F]{24}$"`
	Data T      `json:"data" description:"The data to update" validate:"required"`
}

func (ctr *BaseController[T]) PatchById(c *gin.Context, params *PatchByIdParams[T]) (response *Response[T], err error) {
	id, err := primitive.ObjectIDFromHex(params.Id)
	if err != nil {
		return GetErrorResponse[T](errors.BadRequestf("invalid id format: %v", err))
	}

	u := GetUserFromContext(c)

	// Convert the data to bson.M
	dataJSON, _ := json.Marshal(params.Data)
	var update bson.M
	if err := json.Unmarshal(dataJSON, &update); err != nil {
		return GetErrorResponse[T](errors.BadRequestf("invalid data: %v", err))
	}

	// Remove _id field if present to prevent immutable field error
	delete(update, "_id")

	// Add updated_by and updated_at
	update["updated_by"] = u.Id
	update["updated_at"] = time.Now()

	if err := ctr.modelSvc.UpdateById(id, bson.M{"$set": update}); err != nil {
		return GetErrorResponse[T](err)
	}

	result, err := ctr.modelSvc.GetById(id)
	if err != nil {
		return GetErrorResponse[T](err)
	}

	return GetDataResponse(*result)
}

type PatchParams struct {
	Ids    []string `json:"ids" description:"The IDs of the items to update" validate:"required" items.type:"string" items.format:"objectid" items.pattern:"^[0-9a-fA-F]{24}$"`
	Update bson.M   `json:"update" description:"The update object" validate:"required"`
}

func (ctr *BaseController[T]) PatchList(c *gin.Context, params *PatchParams) (res *Response[T], err error) {
	var ids []primitive.ObjectID
	for _, id := range params.Ids {
		objectId, err := primitive.ObjectIDFromHex(id)
		if err != nil {
			return GetErrorResponse[T](errors.BadRequestf("invalid id format: %v", err))
		}
		ids = append(ids, objectId)
	}

	// Get user from context for updated_by
	u := GetUserFromContext(c)

	// query
	query := bson.M{
		"_id": bson.M{
			"$in": ids,
		},
	}

	// Add updated_by and updated_at to the update object
	updateObj := params.Update
	updateObj["updated_by"] = u.Id
	updateObj["updated_at"] = time.Now()

	// update
	if err := ctr.modelSvc.UpdateMany(query, bson.M{"$set": updateObj}); err != nil {
		return GetErrorResponse[T](err)
	}

	// Return an empty response with success status
	var emptyModel T
	return GetDataResponse(emptyModel)
}

type DeleteByIdParams struct {
	Id string `path:"id" description:"The ID of the item to delete" format:"objectid" pattern:"^[0-9a-fA-F]{24}$"`
}

func (ctr *BaseController[T]) DeleteById(c *gin.Context, params *DeleteByIdParams) (res *Response[T], err error) {
	params.Id = c.Param("id")
	id, err := primitive.ObjectIDFromHex(params.Id)
	if err != nil {
		return GetErrorResponse[T](errors.BadRequestf("invalid id format: %v", err))
	}

	if err := ctr.modelSvc.DeleteById(id); err != nil {
		return GetErrorResponse[T](err)
	}

	var emptyModel T
	return GetDataResponse(emptyModel)
}

type DeleteListParams struct {
	Ids []string `json:"ids" description:"The IDs of the items to delete" items.type:"string" items.format:"objectid" items.pattern:"^[0-9a-fA-F]{24}$"`
}

func (ctr *BaseController[T]) DeleteList(_ *gin.Context, params *DeleteListParams) (res *Response[T], err error) {
	var ids []primitive.ObjectID
	for _, id := range params.Ids {
		objectId, err := primitive.ObjectIDFromHex(id)
		if err != nil {
			return GetErrorResponse[T](errors.BadRequestf("invalid id format: %v", err))
		}
		ids = append(ids, objectId)
	}

	if err := ctr.modelSvc.DeleteMany(bson.M{
		"_id": bson.M{
			"$in": ids,
		},
	}); err != nil {
		return GetErrorResponse[T](err)
	}

	var emptyModel T
	return GetDataResponse(emptyModel)
}

// GetWithPagination retrieves items with pagination
func (ctr *BaseController[T]) GetWithPagination(params *GetListParams) (response *ListResponse[T], err error) {
	// Get filter query
	query := ConvertToBsonMFromListParams(params)

	// Get sort options
	sort, err := GetSortOptionFromString(params.Sort)
	if err != nil {
		return GetErrorListResponse[T](errors.BadRequestf("invalid sort format: %v", err))
	}

	// Get models
	models, err := ctr.modelSvc.GetMany(query, &mongo.FindOptions{
		Sort:  sort,
		Skip:  params.Size * (params.Page - 1),
		Limit: params.Size,
	})
	if err != nil {
		if errors.Is(err, mongo2.ErrNoDocuments) {
			return GetListResponse[T](nil, 0)
		} else {
			return nil, err
		}
	}

	// Total count
	total, err := ctr.modelSvc.Count(query)
	if err != nil {
		return nil, err
	}

	// Response
	return GetListResponse(models, total)
}

func NewController[T any](actions ...Action) *BaseController[T] {
	ctr := &BaseController[T]{
		modelSvc: service.NewModelService[T](),
		actions:  actions,
	}
	return ctr
}
