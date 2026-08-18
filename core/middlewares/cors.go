package middlewares

import (
	"strings"

	"github.com/goastian/astiango-hub/core/utils"
	"github.com/gin-gonic/gin"
	"net/http"
)

func CORSMiddleware() gin.HandlerFunc {
	allowedOrigins := make(map[string]struct{})
	for _, origin := range utils.GetAllowedOrigins() {
		allowedOrigins[origin] = struct{}{}
	}
	allowCredentials := utils.GetAllowCredentials() == "true"

	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if origin == "" {
			c.Next()
			return
		}
		if _, ok := allowedOrigins[origin]; !ok {
			if c.Request.Method == http.MethodOptions {
				c.AbortWithStatus(http.StatusForbidden)
				return
			}
			c.Next()
			return
		}

		c.Writer.Header().Set("Vary", "Origin")
		c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
		if allowCredentials {
			c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		}
		c.Writer.Header().Set("Access-Control-Allow-Headers", utils.GetAllowHeaders())
		c.Writer.Header().Set("Access-Control-Allow-Methods", utils.GetAllowMethods())
		c.Writer.Header().Set("Access-Control-Max-Age", "600")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

func isAllowedOrigin(origin string, allowed []string) bool {
	for _, value := range allowed {
		if strings.TrimSpace(value) == origin {
			return true
		}
	}
	return false
}
