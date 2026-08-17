package controllers

import (
	"fmt"

	"github.com/goastian/astiango-hub/core/constants"
	"github.com/goastian/astiango-hub/core/export"
	"github.com/goastian/astiango-hub/core/interfaces"
	"github.com/gin-gonic/gin"
	"github.com/juju/errors"
)

type PostExportParams struct {
	Type   string `path:"type" validate:"required"`
	Target string `query:"target" validate:"required"`
	Filter string `query:"filter" description:"Filter query"`
}

func PostExport(_ *gin.Context, params *PostExportParams) (response *Response[string], err error) {
	query := ConvertToBsonMFromFilter(params.Filter)
	var exportId string
	switch params.Type {
	case constants.ExportTypeCsv:
		exportId, err = export.GetCsvService().Export(params.Type, params.Target, query)
	case constants.ExportTypeJson:
		exportId, err = export.GetJsonService().Export(params.Type, params.Target, query)
	default:
		return GetErrorResponse[string](errors.BadRequestf("invalid export type: %s", params.Type))
	}
	if err != nil {
		return GetErrorResponse[string](err)
	}

	return GetDataResponse(exportId)
}

type GetExportParams struct {
	Type string `path:"type" validate:"required"`
	Id   string `path:"id" validate:"required"`
}

func GetExport(_ *gin.Context, params *GetExportParams) (response *Response[interfaces.Export], err error) {
	var exp interfaces.Export
	switch params.Type {
	case constants.ExportTypeCsv:
		exp, err = export.GetCsvService().GetExport(params.Id)
	case constants.ExportTypeJson:
		exp, err = export.GetJsonService().GetExport(params.Id)
	default:
		return GetErrorResponse[interfaces.Export](errors.BadRequestf("invalid export type: %s", params.Type))
	}
	if err != nil {
		return GetErrorResponse[interfaces.Export](err)
	}

	return GetDataResponse(exp)
}

type GetExportDownloadParams struct {
	Type string `path:"type" validate:"required"`
	Id   string `path:"id" validate:"required"`
}

func GetExportDownload(c *gin.Context, params *GetExportDownloadParams) (err error) {
	var exp interfaces.Export
	switch params.Type {
	case constants.ExportTypeCsv:
		exp, err = export.GetCsvService().GetExport(params.Id)
	case constants.ExportTypeJson:
		exp, err = export.GetJsonService().GetExport(params.Id)
	default:
		return errors.BadRequestf("invalid export type: %s", params.Type)
	}
	if err != nil {
		return err
	}

	switch params.Type {
	case constants.ExportTypeCsv:
		c.Header("Content-Type", "text/csv")
	case constants.ExportTypeJson:
		c.Header("Content-Type", "text/plain")
	default:
		return errors.BadRequestf("invalid export type: %s", params.Type)
	}
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", exp.GetDownloadPath()))
	c.File(exp.GetDownloadPath())
	return nil
}
