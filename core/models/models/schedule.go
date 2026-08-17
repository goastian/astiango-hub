package models

import (
	"github.com/robfig/cron/v3"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Schedule struct {
	any         `collection:"schedules"`
	BaseModel   `bson:",inline"`
	Name        string               `json:"name" bson:"name" description:"Name"`
	Description string               `json:"description" bson:"description" description:"Description"`
	SpiderId    primitive.ObjectID   `json:"spider_id" bson:"spider_id" description:"Spider ID"`
	Cron        string               `json:"cron" bson:"cron" description:"Cron"`
	EntryId     cron.EntryID         `json:"entry_id" bson:"entry_id" description:"Entry ID"`
	Cmd         string               `json:"cmd" bson:"cmd" description:"Cmd"`
	Param       string               `json:"param" bson:"param" description:"Param"`
	Mode        string               `json:"mode" bson:"mode" description:"Mode"`
	NodeIds     []primitive.ObjectID `json:"node_ids" bson:"node_ids" description:"Node IDs"`
	Priority    int                  `json:"priority" bson:"priority" description:"Priority"`
	Enabled     bool                 `json:"enabled" bson:"enabled" description:"Enabled"`

	// associated data
	Spider *Spider `json:"spider" bson:"_spider" description:"Spider"`
}

type ScheduleDTO struct {
	Schedule `json:",inline" bson:",inline"`

	Spider *Spider `json:"spider,omitempty" bson:"_spider,omitempty"`
}
