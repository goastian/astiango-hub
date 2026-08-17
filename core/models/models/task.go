package models

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Task struct {
	any        `collection:"tasks"`
	BaseModel  `bson:",inline"`
	SpiderId   primitive.ObjectID   `json:"spider_id" bson:"spider_id" description:"Spider ID"`
	Status     string               `json:"status" bson:"status" description:"Status: pending, assigned, running, finished, error, cancelled, abnormal."`
	NodeId     primitive.ObjectID   `json:"node_id" bson:"node_id" description:"Node ID"`
	Cmd        string               `json:"cmd" bson:"cmd" description:"Command"`
	Param      string               `json:"param" bson:"param" description:"Parameter"`
	Error      string               `json:"error" bson:"error" description:"Error"`
	Pid        int                  `json:"pid" bson:"pid" description:"Process ID"`
	ScheduleId primitive.ObjectID   `json:"schedule_id" bson:"schedule_id" description:"Schedule ID"`
	Mode       string               `json:"mode" bson:"mode" description:"Mode"`
	Priority   int                  `json:"priority" bson:"priority" description:"Priority"`
	NodeIds    []primitive.ObjectID `json:"node_ids,omitempty" bson:"-"`

	// associated data
	Stat     *TaskStat `json:"stat,omitempty" bson:"_stat,omitempty"`
	Node     *Node     `json:"node,omitempty" bson:"_node,omitempty"`
	Spider   *Spider   `json:"spider,omitempty" bson:"_spider,omitempty"`
	Schedule *Schedule `json:"schedule,omitempty" bson:"_schedule,omitempty"`
}

type TaskDTO struct {
	Task `json:",inline" bson:",inline"`

	Stat     *TaskStat `json:"stat,omitempty" bson:"_stat,omitempty"`
	Node     *Node     `json:"node,omitempty" bson:"_node,omitempty"`
	Spider   *Spider   `json:"spider,omitempty" bson:"_spider,omitempty"`
	Schedule *Schedule `json:"schedule,omitempty" bson:"_schedule,omitempty"`
}
