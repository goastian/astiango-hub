package models

type Role struct {
	any         `collection:"roles"`
	BaseModel   `bson:",inline"`
	Name        string   `json:"name" bson:"name" description:"Name"`
	Description string   `json:"description" bson:"description" description:"Description"`
	Routes      []string `json:"routes" bson:"routes" description:"Routes"`
	RootAdmin   bool     `json:"-" bson:"root_admin,omitempty" description:"Root admin"`
	IsRootAdmin bool     `json:"root_admin" bson:"-"`
	Users       int      `json:"users" bson:"-"`
}
