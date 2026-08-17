package models

import (
	"reflect"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type BaseModel struct {
	Id        primitive.ObjectID `json:"_id" bson:"_id" description:"ID"`
	CreatedAt time.Time          `json:"created_at,omitempty" bson:"created_at,omitempty" description:"Created timestamp"`
	CreatedBy primitive.ObjectID `json:"created_by,omitempty" bson:"created_by,omitempty" description:"Created by"`
	UpdatedAt time.Time          `json:"updated_at,omitempty" bson:"updated_at,omitempty" description:"Updated timestamp"`
	UpdatedBy primitive.ObjectID `json:"updated_by,omitempty" bson:"updated_by,omitempty" description:"Updated by"`
}

func (m *BaseModel) GetId() primitive.ObjectID {
	return m.Id
}

func (m *BaseModel) SetId(id primitive.ObjectID) {
	m.Id = id
}

func (m *BaseModel) GetCreatedAt() time.Time {
	return m.CreatedAt
}

func (m *BaseModel) SetCreatedAt(t time.Time) {
	m.CreatedAt = t
}

func (m *BaseModel) GetCreatedBy() primitive.ObjectID {
	return m.CreatedBy
}

func (m *BaseModel) SetCreatedBy(id primitive.ObjectID) {
	m.CreatedBy = id
}

func (m *BaseModel) GetUpdatedAt() time.Time {
	return m.UpdatedAt
}

func (m *BaseModel) SetUpdatedAt(t time.Time) {
	m.UpdatedAt = t
}

func (m *BaseModel) GetUpdatedBy() primitive.ObjectID {
	return m.UpdatedBy
}

func (m *BaseModel) SetUpdatedBy(id primitive.ObjectID) {
	m.UpdatedBy = id
}

func (m *BaseModel) SetCreated(id primitive.ObjectID) {
	m.SetCreatedAt(time.Now())
	m.SetCreatedBy(id)
}

func (m *BaseModel) SetUpdated(id primitive.ObjectID) {
	m.SetUpdatedAt(time.Now())
	m.SetUpdatedBy(id)
}

func GetModelInstances() []any {
	return []any{
		*new(TestModel),
		*new(DataCollection),
		*new(Database),
		*new(DatabaseMetric),
		*new(Dependency),
		*new(DependencyLog),
		*new(DependencyConfig),
		*new(DependencyConfigSetup),
		*new(DependencyRepo),
		*new(Environment),
		*new(Git),
		*new(Metric),
		*new(Node),
		*new(NotificationAlert),
		*new(NotificationChannel),
		*new(NotificationRequest),
		*new(NotificationSetting),
		*new(Permission),
		*new(Project),
		*new(Role),
		*new(Schedule),
		*new(Setting),
		*new(Spider),
		*new(SpiderStat),
		*new(TaskStat),
		*new(Task),
		*new(Token),
		*new(UserRole),
		*new(User),
	}
}

func GetCollectionNameByInstance(v any) string {
	t := reflect.TypeOf(v)
	field := t.Field(0)
	return field.Tag.Get("collection")
}

func GetSystemModelColNamesMap() map[string]bool {
	colNamesMap := make(map[string]bool)
	for _, instance := range GetModelInstances() {
		colName := GetCollectionNameByInstance(instance)
		if colName != "" {
			colNamesMap[colName] = true
		}
	}
	return colNamesMap
}

func GetSystemModelColNames() []string {
	colNames := make([]string, 0)
	for _, instance := range GetModelInstances() {
		colName := GetCollectionNameByInstance(instance)
		if colName != "" {
			colNames = append(colNames, colName)
		}
	}
	return colNames
}
