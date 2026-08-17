package utils

import (
	"github.com/goastian/astiango-hub/core/constants"
	"github.com/goastian/astiango-hub/core/entity"
	"github.com/goastian/astiango-hub/trace"
	"github.com/gin-gonic/gin"
	"net/http"
	"time"
)

func handleError(statusCode int, c *gin.Context, err error, print bool) {
	if print {
		trace.PrintError(err)
	}
	c.AbortWithStatusJSON(statusCode, entity.Response{
		Status:  constants.HttpResponseStatusOk,
		Message: constants.HttpResponseMessageError,
		Error:   err.Error(),
	})
}

func HandleError(statusCode int, c *gin.Context, err error) {
	handleError(statusCode, c, err, true)
}

func HandleErrorUnauthorized(c *gin.Context, err error) {
	HandleError(http.StatusUnauthorized, c, err)
}

func HandleErrorInternalServerError(c *gin.Context, err error) {
	HandleError(http.StatusInternalServerError, c, err)
}

func NewHttpClient(timeout time.Duration) *http.Client {
	return &http.Client{
		Timeout: timeout,
	}
}
