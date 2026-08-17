package models

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type SpiderStat struct {
	any                    `collection:"spider_stats"`
	BaseModel              `bson:",inline"`
	LastTaskId             primitive.ObjectID `json:"last_task_id" bson:"last_task_id,omitempty" description:"Last task ID"`
	Tasks                  int                `json:"tasks" bson:"tasks" description:"Task count"`
	Results                int                `json:"results" bson:"results" description:"Result count"`
	WaitDuration           int64              `json:"wait_duration" bson:"wait_duration,omitempty" description:"Wait duration (in second)"`
	RuntimeDuration        int64              `json:"runtime_duration" bson:"runtime_duration,omitempty" description:"Runtime duration (in second)"`
	TotalDuration          int64              `json:"total_duration" bson:"total_duration,omitempty" description:"Total duration (in second)"`
	AverageWaitDuration    int64              `json:"average_wait_duration" bson:"-"`    // in second
	AverageRuntimeDuration int64              `json:"average_runtime_duration" bson:"-"` // in second
	AverageTotalDuration   int64              `json:"average_total_duration" bson:"-"`   // in second
}
