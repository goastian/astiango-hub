package interfaces

import "go.mongodb.org/mongo-driver/bson/primitive"

type SpiderTemplateService interface {
	CreateTemplate(id primitive.ObjectID) (err error)
}
