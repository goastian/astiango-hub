package models

import "go.mongodb.org/mongo-driver/bson/primitive"

type DependencyConfigSetup struct {
	any                `collection:"dependency_config_setups"`
	BaseModel          `bson:",inline"`
	DependencyConfigId primitive.ObjectID `json:"dependency_config_id" bson:"dependency_config_id" description:"Dependency config ID"`
	NodeId             primitive.ObjectID `json:"node_id" bson:"node_id" description:"Node ID"`
	Version            string             `json:"version" bson:"version" description:"Version"`
	Drivers            []DependencyDriver `json:"versions,omitempty" bson:"versions,omitempty" description:"Versions"`
	Status             string             `json:"status" bson:"status" description:"Status"`
	Error              string             `json:"error,omitempty" bson:"error,omitempty" description:"Error"`
	Node               *Node              `json:"node,omitempty" bson:"-" binding:"-"`
}

type DependencyDriver struct {
	Name    string `json:"name" bson:"name" description:"Name"`
	Version string `json:"version" bson:"version" description:"Version"`
}

type DependencyConfigSetupDTO struct {
	DependencyConfigSetup `json:",inline" bson:",inline"`

	Node *Node `json:"node,omitempty" bson:"-"`
}
