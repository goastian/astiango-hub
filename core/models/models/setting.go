package models

import (
	"go.mongodb.org/mongo-driver/bson"
)

type Setting struct {
	any       `collection:"settings"`
	BaseModel `bson:",inline"`
	Key       string `json:"key" bson:"key" description:"Type"`
	Value     bson.M `json:"value" bson:"value" description:"Value"`
}
