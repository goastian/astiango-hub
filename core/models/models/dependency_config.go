package models

type DependencyConfig struct {
	any               `collection:"dependency_configs"`
	BaseModel         `bson:",inline"`
	Key               string `json:"key" bson:"key" description:"Type"`
	Name              string `json:"name" bson:"name" description:"Name"`
	ExecCmd           string `json:"exec_cmd" bson:"exec_cmd" description:"Exec cmd"`
	PkgCmd            string `json:"pkg_cmd" bson:"pkg_cmd" description:"Pkg cmd"`
	PkgSrcURL         string `json:"pkg_src_url" bson:"pkg_src_url" description:"Pkg src url"`
	Setup             bool   `json:"setup" bson:"-" binding:"-"`
	TotalDependencies int    `json:"total_dependencies" bson:"-" binding:"-"`
	SearchReady       bool   `json:"search_ready" bson:"-" binding:"-"`
}

type DependencyConfigDTO struct {
	DependencyConfig `json:",inline" bson:",inline"`

	Setup             bool `json:"setup" bson:"-"`
	TotalDependencies int  `json:"total_dependencies" bson:"-"`
	SearchReady       bool `json:"search_ready" bson:"-"`
}
