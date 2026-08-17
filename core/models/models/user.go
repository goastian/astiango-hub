package models

import "go.mongodb.org/mongo-driver/bson/primitive"

type User struct {
	any           `collection:"users"`
	BaseModel     `bson:",inline"`
	Username      string             `json:"username" bson:"username" description:"Username"`
	Password      string             `json:"-" bson:"password" description:"Password"`
	Role          string             `json:"role" bson:"role" description:"Role"`
	RoleId        primitive.ObjectID `json:"role_id" bson:"role_id" description:"Role ID"`
	FirstName     string             `json:"first_name" bson:"first_name" description:"First name"`
	LastName      string             `json:"last_name" bson:"last_name" description:"Last name"`
	Email         string             `json:"email" bson:"email" description:"Email"`
	RootAdmin     bool               `json:"root_admin,omitempty" bson:"root_admin" description:"Root admin"`
	RootAdminRole bool               `json:"root_admin_role,omitempty" bson:"-"`
	Routes        []string           `json:"routes,omitempty" bson:"-"`
}

type UserDTO struct {
	User `json:",inline" bson:",inline"`

	RoleInfo *Role `json:"role_info,omitempty" bson:"_role,omitempty"`
}
