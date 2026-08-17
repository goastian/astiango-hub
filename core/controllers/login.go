package controllers

import (
	"github.com/gin-gonic/gin"
	"github.com/goastian/astiango-hub/core/constants"
	"github.com/goastian/astiango-hub/core/errors"
	"github.com/goastian/astiango-hub/core/user"
	"github.com/goastian/astiango-hub/core/utils"
)

type PostLoginParams struct {
	Username string `json:"username" description:"Username" validate:"required"`
	Password string `json:"password" description:"Password" validate:"required"`
}

type LoginResponse struct {
	Token                  string `json:"token"`
	PasswordChangeRequired bool   `json:"password_change_required"`
}

func PostLogin(c *gin.Context, params *PostLoginParams) (response *Response[LoginResponse], err error) {
	userSvc, err := user.GetUserService()
	if err != nil {
		return GetErrorResponse[LoginResponse](err)
	}

	token, loggedInUser, err := userSvc.Login(params.Username, params.Password)
	if err != nil {
		// Authentication failures are expected client errors. Returning a 401
		// avoids representing invalid credentials as a server malfunction.
		utils.HandleErrorUnauthorized(c, errors.ErrorUserUnauthorized)
		return nil, nil
	}

	c.Set(constants.UserContextKey, loggedInUser)
	return GetDataResponse(LoginResponse{
		Token:                  token,
		PasswordChangeRequired: loggedInUser.MustChangePassword,
	})
}

func PostLogout(_ *gin.Context) (response *VoidResponse, err error) {
	return GetVoidResponse()
}
