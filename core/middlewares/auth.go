package middlewares

import (
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/goastian/astiango-hub/core/constants"
	"github.com/goastian/astiango-hub/core/models/models"
	"github.com/goastian/astiango-hub/core/models/service"
	"github.com/goastian/astiango-hub/core/user"
	"github.com/goastian/astiango-hub/core/utils"
	"go.mongodb.org/mongo-driver/bson"
)

func AuthorizationMiddleware() gin.HandlerFunc {
	userSvc, _ := user.GetUserService()
	return func(c *gin.Context) {
		// disable auth for test
		if utils.IsAuthDisabled() {
			u, err := service.NewModelService[models.User]().GetOne(bson.M{"root_admin": true}, nil)
			if err != nil {
				utils.HandleErrorInternalServerError(c, err)
				return
			}
			c.Set(constants.UserContextKey, u)
			c.Next()
			return
		}

		// token string
		tokenStr := utils.GetAPITokenFromContext(c)

		// validate token
		u, err := userSvc.CheckToken(tokenStr)
		if err != nil {
			// validation failed, return error response
			utils.HandleErrorUnauthorized(c, errors.New("invalid token"))
			return
		}

		// set user in context
		c.Set(constants.UserContextKey, u)

		// The bootstrap account is deliberately constrained until its password is
		// replaced. Enforcing this server-side prevents API clients from bypassing
		// the UI prompt.
		if u.MustChangePassword && !(c.Request.Method == "POST" && c.Request.URL.Path == "/users/me/change-password") {
			utils.HandleError(403, c, errors.New("password change required"))
			return
		}

		// validation success
		c.Next()
	}
}

func SyncAuthorizationMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if utils.IsAuthDisabled() {
			c.Next()
			return
		}

		authKey := c.GetHeader("Authorization")

		if authKey != utils.GetAuthKey() {
			utils.HandleErrorUnauthorized(c, errors.New("invalid auth key"))
			return
		}

		c.Next()
	}
}
