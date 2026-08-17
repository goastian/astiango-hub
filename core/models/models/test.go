package models

type TestModel struct {
	any       `collection:"testmodels"`
	BaseModel `bson:",inline"`
	Name      string `json:"name" bson:"name"`
}
