package models

import "go.mongodb.org/mongo-driver/bson/primitive"

// Environment is explicitly scoped; unscoped legacy records are never
// injected into a crawler process.
type Environment struct {
	any       `collection:"environments"`
	BaseModel `bson:",inline"`
	Key       string             `json:"key" bson:"key" description:"Environment variable name"`
	Value     string             `json:"value" bson:"value" description:"Secret value"`
	TenantId  primitive.ObjectID `json:"tenant_id,omitempty" bson:"tenant_id,omitempty"`
	ProjectId primitive.ObjectID `json:"project_id,omitempty" bson:"project_id,omitempty"`
	TaskId    primitive.ObjectID `json:"task_id,omitempty" bson:"task_id,omitempty"`
}

// SecretAccessAudit stores only metadata about injection, never secret values.
type SecretAccessAudit struct {
	any       `collection:"secret_access_audits"`
	BaseModel `bson:",inline"`
	TaskId    primitive.ObjectID `json:"task_id" bson:"task_id"`
	ProjectId primitive.ObjectID `json:"project_id,omitempty" bson:"project_id,omitempty"`
	TenantId  primitive.ObjectID `json:"tenant_id,omitempty" bson:"tenant_id,omitempty"`
	Key       string             `json:"key" bson:"key"`
}
