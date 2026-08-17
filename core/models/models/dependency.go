package models

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Dependency struct {
	any         `collection:"dependencies"`
	BaseModel   `bson:",inline"`
	Name        string               `json:"name" bson:"name" description:"Name"`
	Description string               `json:"description" bson:"description" description:"Description"`
	NodeId      primitive.ObjectID   `json:"node_id" bson:"node_id" description:"Node ID"`
	Type        string               `json:"type" bson:"type" description:"Type"`
	Version     string               `json:"version" bson:"version" description:"Version"`
	Status      string               `json:"status" bson:"status" description:"Status"`
	Error       string               `json:"error,omitempty" bson:"error,omitempty" description:"Error"`
	NodeIds     []primitive.ObjectID `json:"node_ids,omitempty" bson:"-" binding:"-"`
	Versions    []string             `json:"versions,omitempty" bson:"-" binding:"-"`
}

type DependencyDTO struct {
	Dependency `json:",inline" bson:",inline"`

	NodeIds  []primitive.ObjectID `json:"node_ids,omitempty" bson:"-"`
	Versions []string             `json:"versions,omitempty" bson:"-"`
}
