package models

type Project struct {
	any         `collection:"projects"`
	BaseModel   `bson:",inline"`
	Name        string `json:"name" bson:"name" description:"Name"`
	Description string `json:"description" bson:"description" description:"Description"`
}

type ProjectDTO struct {
	Project `json:",inline" bson:",inline"`

	Spiders []Spider `json:"spiders,omitempty" bson:"_spiders,omitempty"`
}
