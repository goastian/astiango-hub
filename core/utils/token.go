package utils

import (
	"github.com/gin-gonic/gin"
	"strings"
)

func GetAPITokenFromContext(c *gin.Context) string {
	tokenStr := c.GetHeader("Authorization")
	if strings.HasPrefix(tokenStr, "Bearer ") {
		tokenStr = strings.Replace(tokenStr, "Bearer ", "", 1)
	}
	return tokenStr
}
