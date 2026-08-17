package controllers

import (
	"github.com/crawlab-team/fizz/openapi"
	"github.com/gin-gonic/gin"
)

func GetOpenAPI(c *gin.Context) {
	f := globalWrapper.GetFizz()

	info := &openapi.Info{
		Title:       "AstianGO Hub API",
		Description: "REST API for AstianGO Hub",
		Version:     "0.7.0",
	}

	handleFunc := f.OpenAPI(info, "json")
	handleFunc(c)
}
