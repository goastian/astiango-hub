package models

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type UserRole struct {
	any       `collection:"user_roles"`
	BaseModel `bson:",inline"`
	RoleId    primitive.ObjectID `json:"role_id" bson:"role_id" description:"Role ID"`
	UserId    primitive.ObjectID `json:"user_id" bson:"user_id" description:"User ID"`
}
