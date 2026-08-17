package interfaces

import (
	"go.mongodb.org/mongo-driver/bson"
	"time"
)

type Export interface {
	GetId() string
	GetType() string
	GetTarget() string
	GetQuery() bson.M
	GetStatus() string
	GetStartTs() time.Time
	GetEndTs() time.Time
	GetDownloadPath() string
}
