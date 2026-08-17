package models

import (
	"time"
)

type Node struct {
	any            `collection:"nodes"`
	BaseModel      `bson:",inline"`
	Key            string    `json:"key" bson:"key" description:"Type"`
	Name           string    `json:"name" bson:"name" description:"Name"`
	Ip             string    `json:"ip" bson:"ip" description:"IP"`
	Mac            string    `json:"mac" bson:"mac" description:"MAC"`
	Hostname       string    `json:"hostname" bson:"hostname" description:"Hostname"`
	Description    string    `json:"description" bson:"description" description:"Description"`
	IsMaster       bool      `json:"is_master" bson:"is_master" description:"Is master"`
	Status         string    `json:"status" bson:"status" description:"Status"`
	Enabled        bool      `json:"enabled" bson:"enabled" description:"Enabled"`
	Active         bool      `json:"active" bson:"active" description:"Active"`
	ActiveAt       time.Time `json:"active_at" bson:"active_ts" description:"Active at"`
	CurrentRunners int       `json:"current_runners" bson:"current_runners" description:"Current runners"`
	MaxRunners     int       `json:"max_runners" bson:"max_runners" description:"Max runners"`
}

type NodeDTO struct {
	Node `json:",inline" bson:",inline"`
}
