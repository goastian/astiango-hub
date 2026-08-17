package entity

type LLMResponseUsage struct {
	PromptTokens     int `json:"prompt_tokens" bson:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens" bson:"completion_tokens"`
	TotalTokens      int `json:"total_tokens" bson:"total_tokens"`
}
