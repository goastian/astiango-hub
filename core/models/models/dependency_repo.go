package models

type DependencyRepo struct {
	any           `collection:"dependency_repos"`
	BaseModel     `bson:",inline"`
	Name          string   `json:"name" bson:"name" description:"Name"`
	Type          string   `json:"type" bson:"type" description:"Type"`
	LatestVersion string   `json:"latest_version" bson:"latest_version" description:"Latest version"`
	AllVersions   []string `json:"all_versions" bson:"all_versions" description:"All versions"`
}
