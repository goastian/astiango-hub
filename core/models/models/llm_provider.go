package models

// LLMProvider represents a language model provider such as OpenAI, Anthropic, etc.
type LLMProvider struct {
	any            `collection:"llm_providers"`
	BaseModel      `bson:",inline"`
	Type           string   `json:"type" bson:"type" description:"Provider type (e.g., 'openai', 'azure-openai', 'anthropic', 'gemini')"`
	Name           string   `json:"name" bson:"name" description:"Display name for UI"`
	ApiKey         string   `json:"api_key" bson:"api_key" description:"API key for the provider"`
	ApiBaseUrl     string   `json:"api_base_url" bson:"api_base_url" description:"API base URL for the provider"`
	DeploymentName string   `json:"deployment_name" bson:"deployment_name" description:"Deployment name for the provider"`
	ApiVersion     string   `json:"api_version" bson:"api_version" description:"API version for the provider"`
	Models         []string `json:"models" bson:"models" description:"Models supported by this provider"`
	DefaultModel   string   `json:"default_model" bson:"default_model" description:"Default model for this provider"`
}
