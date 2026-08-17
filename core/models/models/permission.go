package models

import "go.mongodb.org/mongo-driver/bson/primitive"

type Permission struct {
	any         `collection:"permissions"`
	BaseModel   `bson:",inline"`
	Key         string             `json:"key" bson:"key" description:"Type"`
	Name        string             `json:"name" bson:"name" description:"Name"`
	Description string             `json:"description" bson:"description" description:"Description"`
	RoleId      primitive.ObjectID `json:"role_id" bson:"role_id" description:"Role ID"`
	Type        string             `json:"type" bson:"type" description:"Type"`
	Routes      []string           `json:"routes" bson:"routes" description:"Routes"`
}
