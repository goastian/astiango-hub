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
	RefreshToken           string `json:"refresh_token"`
	AccessExpiresIn        int64  `json:"access_expires_in"`
	RefreshExpiresIn       int64  `json:"refresh_expires_in"`
	PasswordChangeRequired bool   `json:"password_change_required"`
}

type PostRefreshParams struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
}

type PostLogoutParams struct {
	RefreshToken string `json:"refresh_token"`
}

func PostLogin(c *gin.Context, params *PostLoginParams) (response *Response[LoginResponse], err error) {
	userSvc, err := user.GetUserService()
	if err != nil {
		return GetErrorResponse[LoginResponse](err)
	}

	pair, loggedInUser, err := userSvc.LoginWithTokens(params.Username, params.Password)
	if err != nil {
		// Authentication failures are expected client errors. Returning a 401
		// avoids representing invalid credentials as a server malfunction.
		utils.HandleErrorUnauthorized(c, errors.ErrorUserUnauthorized)
		return nil, nil
	}

	c.Set(constants.UserContextKey, loggedInUser)
	return GetDataResponse(LoginResponse{
		Token:                  pair.AccessToken,
		RefreshToken:           pair.RefreshToken,
		AccessExpiresIn:        pair.AccessExpiresIn,
		RefreshExpiresIn:       pair.RefreshExpiresIn,
		PasswordChangeRequired: loggedInUser.MustChangePassword,
	})
}

func PostRefresh(c *gin.Context, params *PostRefreshParams) (response *Response[LoginResponse], err error) {
	userSvc, err := user.GetUserService()
	if err != nil {
		return GetErrorResponse[LoginResponse](err)
	}
	pair, err := userSvc.Refresh(params.RefreshToken)
	if err != nil {
		utils.HandleErrorUnauthorized(c, errors.ErrorUserUnauthorized)
		return nil, nil
	}
	return GetDataResponse(LoginResponse{
		Token: pair.AccessToken, RefreshToken: pair.RefreshToken,
		AccessExpiresIn: pair.AccessExpiresIn, RefreshExpiresIn: pair.RefreshExpiresIn,
	})
}

func PostLogout(c *gin.Context, params *PostLogoutParams) (response *VoidResponse, err error) {
	userSvc, err := user.GetUserService()
	if err != nil {
		return GetErrorVoidResponse(err)
	}
	if err := userSvc.Logout(utils.GetAPITokenFromContext(c), params.RefreshToken); err != nil {
		return GetErrorVoidResponse(err)
	}
	return GetVoidResponse()
}
