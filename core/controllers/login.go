package controllers

import (
	"github.com/goastian/astiango-hub/core/constants"
	"github.com/goastian/astiango-hub/core/errors"
	"github.com/goastian/astiango-hub/core/user"
	"github.com/gin-gonic/gin"
)

type PostLoginParams struct {
	Username string `json:"username" description:"Username" validate:"required"`
	Password string `json:"password" description:"Password" validate:"required"`
}

func PostLogin(c *gin.Context, params *PostLoginParams) (response *Response[string], err error) {
	userSvc, err := user.GetUserService()
	if err != nil {
		return GetErrorResponse[string](err)
	}

	token, loggedInUser, err := userSvc.Login(params.Username, params.Password)
	if err != nil {
		return GetErrorResponse[string](errors.ErrorUserUnauthorized)
	}

	c.Set(constants.UserContextKey, loggedInUser)
	return GetDataResponse(token)
}

func PostLogout(_ *gin.Context) (response *VoidResponse, err error) {
	return GetVoidResponse()
}
