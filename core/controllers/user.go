package controllers

import (
	"regexp"

	"github.com/goastian/astiango-hub/core/constants"
	"github.com/goastian/astiango-hub/core/mongo"
	"github.com/juju/errors"

	"github.com/gin-gonic/gin"
	"github.com/goastian/astiango-hub/core/models/models"
	"github.com/goastian/astiango-hub/core/models/service"
	"github.com/goastian/astiango-hub/core/utils"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	mongo2 "go.mongodb.org/mongo-driver/mongo"
)

func GetUserById(c *gin.Context, params *GetByIdParams) (response *Response[models.User], err error) {
	id, err := primitive.ObjectIDFromHex(params.Id)
	if err != nil {
		return GetErrorResponse[models.User](errors.BadRequestf("invalid user id: %v", err))
	}
	target, err := service.NewModelService[models.User]().GetById(id)
	if err != nil {
		return getUserById(id)
	}
	if err := requireUserManagementAccess(GetUserFromContext(c), target); err != nil {
		return GetErrorResponse[models.User](err)
	}
	return getUserById(id)
}

func GetUserList(c *gin.Context, params *GetListParams) (response *ListResponse[models.User], err error) {
	actor := GetUserFromContext(c)
	if err := requireTenantAdministrator(actor); err != nil {
		return GetErrorListResponse[models.User](err)
	}

	query := ConvertToBsonMFromListParams(params)
	if query == nil {
		query = bson.M{}
	}
	if !isRootAdmin(actor) {
		query["tenant_id"] = actor.TenantId
	}

	sort, err := GetSortOptionFromString(params.Sort)
	if err != nil {
		return GetErrorListResponse[models.User](err)
	}

	users, err := service.NewModelService[models.User]().GetMany(query, &mongo.FindOptions{
		Sort:  sort,
		Skip:  params.Size * (params.Page - 1),
		Limit: params.Size,
	})
	if err != nil {
		if errors.Is(err, mongo2.ErrNoDocuments) {
			return GetListResponse[models.User](nil, 0)
		} else {
			return GetErrorListResponse[models.User](err)
		}
	}

	// get roles
	if utils.IsPro() {
		var roleIds []primitive.ObjectID
		for _, user := range users {
			if !user.RoleId.IsZero() {
				roleIds = append(roleIds, user.RoleId)
			}
		}
		if len(roleIds) > 0 {
			roles, err := service.NewModelService[models.Role]().GetMany(bson.M{
				"_id": bson.M{"$in": roleIds},
			}, nil)
			if err != nil {
				return GetErrorListResponse[models.User](err)
			}
			rolesMap := make(map[primitive.ObjectID]models.Role)
			for _, role := range roles {
				rolesMap[role.Id] = role
			}
			for i, user := range users {
				if user.RoleId.IsZero() {
					continue
				}
				if role, ok := rolesMap[user.RoleId]; ok {
					users[i].Role = role.Name
					users[i].RootAdminRole = role.RootAdmin
				}
			}
		}
	}

	// total count
	total, err := service.NewModelService[models.User]().Count(query)
	if err != nil {
		return GetErrorListResponse[models.User](err)
	}

	// response
	return GetListResponse(users, total)
}

type PostUserParams struct {
	Data struct {
		Username string `json:"username" description:"Username" validate:"required"`
		Password string `json:"password" description:"Password" validate:"required"`
		Role     string `json:"role" description:"Role"`
		RoleId   string `json:"role_id" description:"Role ID" format:"objectid" pattern:"^[0-9a-fA-F]{24}$"`
		TenantId string `json:"tenant_id" description:"Tenant ID" format:"objectid" pattern:"^[0-9a-fA-F]{24}$"`
		Email    string `json:"email" description:"Email"`
	} `json:"data" validate:"required"`
}

func PostUser(c *gin.Context, params *PostUserParams) (response *Response[models.User], err error) {
	actor := GetUserFromContext(c)
	if err := requireTenantAdministrator(actor); err != nil {
		return GetErrorResponse[models.User](err)
	}

	// Validate email format
	if params.Data.Email != "" {
		emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)
		if !emailRegex.MatchString(params.Data.Email) {
			return GetErrorResponse[models.User](errors.BadRequestf("invalid email format"))
		}
	}

	var roleId primitive.ObjectID
	if params.Data.RoleId != "" {
		roleId, err = primitive.ObjectIDFromHex(params.Data.RoleId)
		if err != nil {
			return GetErrorResponse[models.User](errors.BadRequestf("invalid role id: %v", err))
		}
		_, err = service.NewModelService[models.Role]().GetById(roleId)
		if err != nil {
			return GetErrorResponse[models.User](errors.BadRequestf("role not found: %v", err))
		}
	}
	tenantId := actor.TenantId
	if params.Data.TenantId != "" {
		requestedTenantId, err := primitive.ObjectIDFromHex(params.Data.TenantId)
		if err != nil {
			return GetErrorResponse[models.User](errors.BadRequestf("invalid tenant id: %v", err))
		}
		if !isRootAdmin(actor) && requestedTenantId != actor.TenantId {
			return GetErrorResponse[models.User](errors.Forbiddenf("cannot create users outside your tenant"))
		}
		tenantId = requestedTenantId
	}
	role := params.Data.Role
	if role == "" {
		role = constants.RoleNormal
	}
	model := models.User{
		Username: params.Data.Username,
		Password: utils.EncryptMd5(params.Data.Password),
		Role:     role,
		RoleId:   roleId,
		TenantId: tenantId,
		Email:    params.Data.Email,
	}
	model.SetCreated(actor.Id)
	model.SetUpdated(actor.Id)
	id, err := service.NewModelService[models.User]().InsertOne(model)
	if err != nil {
		return GetErrorResponse[models.User](err)
	}

	result, err := service.NewModelService[models.User]().GetById(id)
	if err != nil {
		return GetErrorResponse[models.User](err)
	}

	return GetDataResponse(*result)
}

func PutUserById(c *gin.Context, params *PutByIdParams[models.User]) (response *Response[models.User], err error) {
	id, err := primitive.ObjectIDFromHex(params.Id)
	if err != nil {
		return GetErrorResponse[models.User](errors.BadRequestf("invalid user id: %v", err))
	}
	actor := GetUserFromContext(c)
	target, err := service.NewModelService[models.User]().GetById(id)
	if err != nil {
		return GetErrorResponse[models.User](err)
	}
	if err := requireUserManagementAccess(actor, target); err != nil {
		return GetErrorResponse[models.User](err)
	}
	return putUser(id, actor, params.Data)
}

// PatchUserById is intentionally disabled. The generic patch controller can
// modify privilege-bearing fields directly, bypassing the invariants enforced
// by PutUserById. Use the guarded replacement endpoint instead.
func PatchUserById(_ *gin.Context, _ *PatchByIdParams[models.User]) (response *Response[models.User], err error) {
	return GetErrorResponse[models.User](errors.Forbiddenf("user patch endpoint is disabled; use PUT"))
}

// PatchUserList is intentionally disabled for the same reason as
// PatchUserById: bulk partial updates are not safe for identity records.
func PatchUserList(_ *gin.Context, _ *PatchParams) (response *Response[models.User], err error) {
	return GetErrorResponse[models.User](errors.Forbiddenf("bulk user patch endpoint is disabled"))
}

type PostUserChangePasswordParams struct {
	Id       string `path:"id" description:"User ID" format:"objectid" pattern:"^[0-9a-fA-F]{24}$"`
	Password string `json:"password" description:"Password" validate:"required"`
}

func PostUserChangePassword(c *gin.Context, params *PostUserChangePasswordParams) (response *Response[models.User], err error) {
	id, err := primitive.ObjectIDFromHex(params.Id)
	if err != nil {
		return GetErrorResponse[models.User](errors.BadRequestf("invalid user id: %v", err))
	}
	return postUserChangePassword(id, GetUserFromContext(c), params.Password)
}

func DeleteUserById(c *gin.Context, params *DeleteByIdParams) (response *Response[models.User], err error) {
	id, err := primitive.ObjectIDFromHex(params.Id)
	if err != nil {
		return GetErrorResponse[models.User](errors.BadRequestf("invalid user id: %v", err))
	}

	user, err := service.NewModelService[models.User]().GetById(id)
	if err != nil {
		return GetErrorResponse[models.User](err)
	}
	if err := requireUserManagementAccess(GetUserFromContext(c), user); err != nil {
		return GetErrorResponse[models.User](err)
	}
	if user.RootAdmin {
		return GetErrorResponse[models.User](errors.Forbiddenf("root admin cannot be deleted"))
	}

	if err := service.NewModelService[models.User]().DeleteById(id); err != nil {
		return GetErrorResponse[models.User](err)
	}

	return GetDataResponse(models.User{})
}

func DeleteUserList(c *gin.Context, params *DeleteListParams) (response *Response[models.User], err error) {
	// Convert string IDs to ObjectIDs
	var ids []primitive.ObjectID
	for _, id := range params.Ids {
		objectId, err := primitive.ObjectIDFromHex(id)
		if err != nil {
			return GetErrorResponse[models.User](errors.BadRequestf("invalid user id: %v", err))
		}
		ids = append(ids, objectId)
	}
	actor := GetUserFromContext(c)
	for _, id := range ids {
		target, err := service.NewModelService[models.User]().GetById(id)
		if err != nil {
			return GetErrorResponse[models.User](err)
		}
		if err := requireUserManagementAccess(actor, target); err != nil {
			return GetErrorResponse[models.User](err)
		}
	}

	// Check if root admin is in the list
	_, err = service.NewModelService[models.User]().GetOne(bson.M{
		"_id": bson.M{
			"$in": ids,
		},
		"root_admin": true,
	}, nil)
	if err == nil {
		return GetErrorResponse[models.User](errors.Forbiddenf("root admin cannot be deleted"))
	}
	if !errors.Is(err, mongo2.ErrNoDocuments) {
		return GetErrorResponse[models.User](err)
	}

	// Delete users
	if err := service.NewModelService[models.User]().DeleteMany(bson.M{
		"_id": bson.M{
			"$in": ids,
		},
	}); err != nil {
		return GetErrorResponse[models.User](err)
	}

	return GetDataResponse(models.User{})
}

func GetUserMe(c *gin.Context) (response *Response[models.User], err error) {
	u := GetUserFromContext(c)
	return getUserByIdWithRoutes(u.Id)
}

type PutUserMeParams struct {
	Data models.User `json:"data"`
}

func PutUserMe(c *gin.Context, params *PutUserMeParams) (response *Response[models.User], err error) {
	u := GetUserFromContext(c)
	return putUser(u.Id, u, params.Data)
}

type PostUserMeChangePasswordParams struct {
	Password string `json:"password" description:"Password" validate:"required"`
}

func PostUserMeChangePassword(c *gin.Context, params *PostUserMeChangePasswordParams) (response *Response[models.User], err error) {
	u := GetUserFromContext(c)
	return postUserChangePassword(u.Id, u, params.Password)
}

func getUserById(userId primitive.ObjectID) (response *Response[models.User], err error) {
	// get user
	user, err := service.NewModelService[models.User]().GetById(userId)
	if err != nil {
		if errors.Is(err, mongo2.ErrNoDocuments) {
			return GetErrorResponse[models.User](errors.BadRequestf("user not found: %v", err))
		}
		return GetErrorResponse[models.User](err)
	}

	// get role
	if utils.IsPro() {
		if !user.RoleId.IsZero() {
			role, err := service.NewModelService[models.Role]().GetById(user.RoleId)
			if err != nil {
				return GetErrorResponse[models.User](errors.BadRequestf("role not found: %v", err))
			}
			user.Role = role.Name
			user.RootAdminRole = role.RootAdmin
		}
	}

	return GetDataResponse(*user)
}

func getUserByIdWithRoutes(userId primitive.ObjectID) (response *Response[models.User], err error) {
	if !utils.IsPro() {
		return getUserById(userId)
	}

	// get user
	user, err := service.NewModelService[models.User]().GetById(userId)
	if err != nil {
		if errors.Is(err, mongo2.ErrNoDocuments) {
			return GetErrorResponse[models.User](errors.BadRequestf("user not found: %v", err))
		}
		return GetErrorResponse[models.User](err)
	}

	// get role
	if !user.RoleId.IsZero() {
		role, err := service.NewModelService[models.Role]().GetById(user.RoleId)
		if err != nil {
			if errors.Is(err, mongo2.ErrNoDocuments) {
				return GetErrorResponse[models.User](errors.BadRequestf("role not found: %v", err))
			}
			return GetErrorResponse[models.User](err)
		}
		user.Role = role.Name
		user.RootAdminRole = role.RootAdmin
		user.Routes = role.Routes
	}

	return GetDataResponse(*user)
}

func putUser(userId primitive.ObjectID, actor *models.User, user models.User) (response *Response[models.User], err error) {
	// model service
	modelSvc := service.NewModelService[models.User]()

	// update user
	userDb, err := modelSvc.GetById(userId)
	if err != nil {
		if errors.Is(err, mongo2.ErrNoDocuments) {
			return GetErrorResponse[models.User](errors.BadRequestf("user not found: %v", err))
		}
		return GetErrorResponse[models.User](err)
	}

	// Root administrator status is assigned only during bootstrap, never by a
	// client-controlled update. Non-root administrators cannot move users
	// across tenants or elevate roles.
	user.RootAdmin = userDb.RootAdmin
	if !isRootAdmin(actor) {
		user.Role = userDb.Role
		user.RoleId = userDb.RoleId
		user.TenantId = userDb.TenantId
	}

	// if root admin, disallow changing username and role
	if userDb.RootAdmin {
		user.Username = userDb.Username
		user.RoleId = userDb.RoleId
		user.Role = userDb.Role
		user.TenantId = userDb.TenantId
	}

	// disallow changing password
	user.Password = userDb.Password

	// update user
	user.SetUpdated(actor.Id)
	if user.Id.IsZero() {
		user.Id = userId
	}
	if err := modelSvc.ReplaceById(userId, user); err != nil {
		return GetErrorResponse[models.User](err)
	}

	// handle success
	return GetDataResponse(user)
}

func postUserChangePassword(userId primitive.ObjectID, actor *models.User, password string) (response *Response[models.User], err error) {
	if len(password) < 5 {
		return GetErrorResponse[models.User](errors.BadRequestf("password must be at least 5 characters"))
	}

	// update password
	userDb, err := service.NewModelService[models.User]().GetById(userId)
	if err != nil {
		return GetErrorResponse[models.User](err)
	}
	if err := requireUserManagementAccess(actor, userDb); err != nil {
		return GetErrorResponse[models.User](err)
	}
	userDb.SetUpdated(actor.Id)
	userDb.Password = utils.EncryptMd5(password)
	if err := service.NewModelService[models.User]().ReplaceById(userDb.Id, *userDb); err != nil {
		return GetErrorResponse[models.User](err)
	}

	return GetDataResponse(models.User{})
}
