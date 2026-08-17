package controllers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func GetHealthFn(healthFn func() bool) func(c *gin.Context) {
	return func(c *gin.Context) {
		if healthFn() {
			_, _ = c.Writer.Write([]byte("ok"))
			c.AbortWithStatus(http.StatusOK)
			return
		}
		_, _ = c.Writer.Write([]byte("not ready"))
		c.AbortWithStatus(http.StatusServiceUnavailable)
	}
}
