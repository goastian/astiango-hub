package interfaces

import "go.mongodb.org/mongo-driver/bson"

type ExportService interface {
	GenerateId() (exportId string, err error)
	Export(exportType, target string, query bson.M) (exportId string, err error)
	GetExport(exportId string) (export Export, err error)
}
