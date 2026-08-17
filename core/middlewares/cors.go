package middlewares

import (
	"github.com/goastian/astiango-hub/core/utils"
	"github.com/gin-gonic/gin"
	"net/http"
)

func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", utils.GetAllowOrigin())
		c.Writer.Header().Set("Access-Control-Allow-Credentials", utils.GetAllowCredentials())
		c.Writer.Header().Set("Access-Control-Allow-Headers", utils.GetAllowHeaders())
		c.Writer.Header().Set("Access-Control-Allow-Methods", utils.GetAllowMethods())

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}
