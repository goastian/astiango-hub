package models

import (
	"time"

	"github.com/goastian/astiango-hub/vcs"
)

type Git struct {
	any           `collection:"gits"`
	BaseModel     `bson:",inline"`
	Url           string       `json:"url" bson:"url" description:"URL"`
	Name          string       `json:"name" bson:"name" description:"Name"`
	AuthType      string       `json:"auth_type" bson:"auth_type" description:"Auth type"`
	Username      string       `json:"username" bson:"username" description:"Username"`
	Password      string       `json:"password" bson:"password" description:"Password"`
	CurrentBranch string       `json:"current_branch" bson:"current_branch" description:"Current branch"`
	Status        string       `json:"status" bson:"status" description:"Status"`
	Error         string       `json:"error" bson:"error" description:"Error"`
	Refs          []vcs.GitRef `json:"refs" bson:"refs" description:"Refs"`
	RefsUpdatedAt time.Time    `json:"refs_updated_at" bson:"refs_updated_at" description:"Refs updated at"`
	CloneLogs     []string     `json:"clone_logs,omitempty" bson:"clone_logs" description:"Clone logs"`

	// settings
	AutoPull bool `json:"auto_pull" bson:"auto_pull" description:"Auto pull"`
}

type GitDTO struct {
	Git `json:",inline" bson:",inline"`

	Spiders []Spider `json:"spiders,omitempty" bson:"_spiders,omitempty" description:"Spiders"`
}
