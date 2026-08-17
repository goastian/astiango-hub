package models

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Spider struct {
	any            `collection:"spiders"`
	BaseModel      `bson:",inline"`
	Name           string                `json:"name" bson:"name" description:"Spider name"`
	ColId          primitive.ObjectID    `json:"col_id" bson:"col_id" description:"Data collection id" deprecated:"true"`
	ColName        string                `json:"col_name,omitempty" bson:"col_name" description:"Data collection name"`
	DbName         string                `json:"db_name,omitempty" bson:"db_name" description:"Database name"`
	Description    string                `json:"description" bson:"description" description:"Description"`
	DatabaseId     primitive.ObjectID    `json:"database_id" bson:"database_id" description:"Database Id"`
	ProjectId      primitive.ObjectID    `json:"project_id" bson:"project_id" description:"Project ID"`
	Mode           string                `json:"mode" bson:"mode" description:"Default task mode" enum:"random,all,selected-nodes"`
	NodeIds        []primitive.ObjectID  `json:"node_ids" bson:"node_ids" description:"Default node ids, used in selected-nodes mode"`
	GitId          primitive.ObjectID    `json:"git_id" bson:"git_id" description:"Related Git ID"`
	GitRootPath    string                `json:"git_root_path" bson:"git_root_path" description:"Git root path"`
	Template       string                `json:"template,omitempty" bson:"template,omitempty" description:"Spider template"`
	TemplateParams *SpiderTemplateParams `json:"template_params,omitempty" bson:"template_params,omitempty" description:"Spider template params"`

	// execution
	Cmd      string `json:"cmd" bson:"cmd" description:"Execute command"`
	Param    string `json:"param" bson:"param" description:"Default task param"`
	Priority int    `json:"priority" bson:"priority" description:"Priority" default:"5" minimum:"1" maximum:"10"`
}

type SpiderTemplateParams struct {
	ProjectName    string `json:"project_name,omitempty" bson:"project_name,omitempty" description:"Project name"`
	SpiderName     string `json:"spider_name,omitempty" bson:"spider_name,omitempty" description:"Spider name"`
	StartUrls      string `json:"start_urls,omitempty" bson:"start_urls,omitempty" description:"Start urls"`
	AllowedDomains string `json:"allowed_domains,omitempty" bson:"allowed_domains,omitempty" description:"Allowed domains"`
}

type SpiderDTO struct {
	Spider `json:",inline" bson:",inline"`

	Stat     *SpiderStat `json:"stat,omitempty" bson:"_stat,omitempty"`
	LastTask *Task       `json:"last_task,omitempty" bson:"_last_task,omitempty"`
	Project  *Project    `json:"project,omitempty" bson:"_project,omitempty"`
	Git      *Git        `json:"git,omitempty" bson:"_git,omitempty"`
	Database *Database   `json:"database,omitempty" bson:"_database,omitempty"`
}
