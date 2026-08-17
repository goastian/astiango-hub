package models

type DependencyPypiProject struct {
	any        `collection:"dependency_pypi_projects"`
	BaseModel  `bson:",inline"`
	Name       string `json:"name" bson:"name" description:"Name"`
	Version    string `json:"version" bson:"version" description:"Version"`
	LastSerial int    `json:"_last-serial" bson:"last_serial" description:"Last serial"`
}
