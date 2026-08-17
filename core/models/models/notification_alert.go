package models

import "go.mongodb.org/mongo-driver/bson/primitive"

type NotificationAlert struct {
	any             `collection:"notification_alerts"`
	BaseModel       `bson:",inline"`
	Name            string             `json:"name" bson:"name" description:"Name"`
	Description     string             `json:"description" bson:"description" description:"Description"`
	Enabled         bool               `json:"enabled" bson:"enabled" description:"Enabled"`
	HasMetricTarget bool               `json:"has_metric_target" bson:"has_metric_target" description:"Has metric target"`
	MetricTargetId  primitive.ObjectID `json:"metric_target_id,omitempty" bson:"metric_target_id,omitempty" description:"Metric target ID"`
	MetricName      string             `json:"metric_name" bson:"metric_name" description:"Metric name"`
	Operator        string             `json:"operator" bson:"operator" description:"Operator"`
	LastingSeconds  int                `json:"lasting_seconds" bson:"lasting_seconds" description:"Lasting seconds"`
	TargetValue     float32            `json:"target_value" bson:"target_value" description:"Target value"`
	Level           string             `json:"level" bson:"level" description:"Level"`
	TemplateKey     string             `json:"template_key,omitempty" bson:"template_key,omitempty" description:"Template key"`
}
