package service

import (
	"context"
	"fmt"
	"reflect"
	"sync"

	"github.com/goastian/astiango-hub/core/mongo"
	"github.com/goastian/astiango-hub/core/utils"

	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/goastian/astiango-hub/core/interfaces"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

var (
	instanceMap    = make(map[string]any)
	onceMap        = make(map[string]*sync.Once)
	onceColNameMap = make(map[string]*sync.Once)
	mu             sync.Mutex
)

type ModelService[T any] struct {
	col *mongo.Col
}

func (svc *ModelService[T]) GetCol() (col *mongo.Col) {
	return svc.col
}

func (svc *ModelService[T]) GetById(id primitive.ObjectID) (model *T, err error) {
	var result T
	err = svc.col.FindId(id).One(&result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func (svc *ModelService[T]) GetByIdContext(ctx context.Context, id primitive.ObjectID) (model *T, err error) {
	var result T
	err = svc.col.GetCollection().FindOne(ctx, bson.M{"_id": id}).Decode(&result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func (svc *ModelService[T]) GetOne(query bson.M, options *mongo.FindOptions) (model *T, err error) {
	var result T
	err = svc.col.Find(query, options).One(&result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func (svc *ModelService[T]) GetOneContext(ctx context.Context, query bson.M, opts *mongo.FindOptions) (model *T, err error) {
	var result T
	_opts := &options.FindOneOptions{}
	if opts != nil {
		if opts.Skip != 0 {
			skipInt64 := int64(opts.Skip)
			_opts.Skip = &skipInt64
		}
		if opts.Sort != nil {
			_opts.Sort = opts.Sort
		}
	}
	err = svc.col.GetCollection().FindOne(ctx, query, _opts).Decode(&result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func (svc *ModelService[T]) GetMany(query bson.M, options *mongo.FindOptions) (models []T, err error) {
	var result []T
	err = svc.col.Find(query, options).All(&result)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (svc *ModelService[T]) GetManyContext(ctx context.Context, query bson.M, opts *mongo.FindOptions) (models []T, err error) {
	var result []T
	_opts := &options.FindOptions{}
	if opts != nil {
		if opts.Skip != 0 {
			skipInt64 := int64(opts.Skip)
			_opts.Skip = &skipInt64
		}
		if opts.Limit != 0 {
			limitInt64 := int64(opts.Limit)
			_opts.Limit = &limitInt64
		}
		if opts.Sort != nil {
			_opts.Sort = opts.Sort
		}
	}
	cur, err := svc.col.GetCollection().Find(ctx, query, _opts)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	for cur.Next(ctx) {
		var model T
		if err := cur.Decode(&model); err != nil {
			return nil, err
		}
		result = append(result, model)
	}
	return result, nil
}

func (svc *ModelService[T]) DeleteById(id primitive.ObjectID) (err error) {
	return svc.col.DeleteId(id)
}

func (svc *ModelService[T]) DeleteByIdContext(ctx context.Context, id primitive.ObjectID) (err error) {
	_, err = svc.col.GetCollection().DeleteOne(ctx, bson.M{"_id": id})
	return err
}

func (svc *ModelService[T]) DeleteOne(query bson.M) (err error) {
	_, err = svc.col.GetCollection().DeleteOne(svc.col.GetContext(), query)
	return err
}

func (svc *ModelService[T]) DeleteOneContext(ctx context.Context, query bson.M) (err error) {
	_, err = svc.col.GetCollection().DeleteOne(ctx, query)
	return err
}

func (svc *ModelService[T]) DeleteMany(query bson.M) (err error) {
	_, err = svc.col.GetCollection().DeleteMany(svc.col.GetContext(), query, nil)
	return err
}

func (svc *ModelService[T]) DeleteManyContext(ctx context.Context, query bson.M) (err error) {
	_, err = svc.col.GetCollection().DeleteMany(ctx, query, nil)
	return err
}

func (svc *ModelService[T]) UpdateById(id primitive.ObjectID, update bson.M) (err error) {
	return svc.col.UpdateId(id, update)
}

func (svc *ModelService[T]) UpdateByIdContext(ctx context.Context, id primitive.ObjectID, update bson.M) (err error) {
	_, err = svc.col.GetCollection().UpdateOne(ctx, bson.M{"_id": id}, update)
	return err
}

func (svc *ModelService[T]) UpdateOne(query bson.M, update bson.M) (err error) {
	_, err = svc.col.GetCollection().UpdateOne(svc.col.GetContext(), query, update)
	return err
}

func (svc *ModelService[T]) UpdateOneContext(ctx context.Context, query bson.M, update bson.M) (err error) {
	_, err = svc.col.GetCollection().UpdateOne(ctx, query, update)
	return err
}

func (svc *ModelService[T]) UpdateMany(query bson.M, update bson.M) (err error) {
	_, err = svc.col.GetCollection().UpdateMany(svc.col.GetContext(), query, update)
	return err
}

func (svc *ModelService[T]) UpdateManyContext(ctx context.Context, query bson.M, update bson.M) (err error) {
	_, err = svc.col.GetCollection().UpdateMany(ctx, query, update)
	return err
}

func (svc *ModelService[T]) ReplaceById(id primitive.ObjectID, model T) (err error) {
	_, err = svc.col.GetCollection().ReplaceOne(svc.col.GetContext(), bson.M{"_id": id}, model)
	return err
}

func (svc *ModelService[T]) ReplaceByIdContext(ctx context.Context, id primitive.ObjectID, model T) (err error) {
	_, err = svc.col.GetCollection().ReplaceOne(ctx, bson.M{"_id": id}, model)
	return err
}

func (svc *ModelService[T]) ReplaceOne(query bson.M, model T) (err error) {
	_, err = svc.col.GetCollection().ReplaceOne(svc.col.GetContext(), query, model)
	return err
}

func (svc *ModelService[T]) ReplaceOneContext(ctx context.Context, query bson.M, model T) (err error) {
	_, err = svc.col.GetCollection().ReplaceOne(ctx, query, model)
	return err
}

func (svc *ModelService[T]) InsertOne(model T) (id primitive.ObjectID, err error) {
	m := any(&model).(interfaces.Model)
	if m.GetId().IsZero() {
		m.SetId(primitive.NewObjectID())
	}
	res, err := svc.col.GetCollection().InsertOne(svc.col.GetContext(), m)
	if err != nil {
		return primitive.NilObjectID, err
	}
	return res.InsertedID.(primitive.ObjectID), nil
}

func (svc *ModelService[T]) InsertOneContext(ctx context.Context, model T) (id primitive.ObjectID, err error) {
	m := any(&model).(interfaces.Model)
	if m.GetId().IsZero() {
		m.SetId(primitive.NewObjectID())
	}
	res, err := svc.col.GetCollection().InsertOne(ctx, m)
	if err != nil {
		return primitive.NilObjectID, err
	}
	return res.InsertedID.(primitive.ObjectID), nil
}

func (svc *ModelService[T]) InsertMany(models []T) (ids []primitive.ObjectID, err error) {
	var _models []any
	for _, model := range models {
		m := any(&model).(interfaces.Model)
		if m.GetId().IsZero() {
			m.SetId(primitive.NewObjectID())
		}
		_models = append(_models, m)
	}
	res, err := svc.col.GetCollection().InsertMany(svc.col.GetContext(), _models)
	if err != nil {
		return nil, err
	}
	for _, v := range res.InsertedIDs {
		ids = append(ids, v.(primitive.ObjectID))
	}
	return ids, nil
}

func (svc *ModelService[T]) InsertManyContext(ctx context.Context, models []T) (ids []primitive.ObjectID, err error) {
	var _models []any
	for _, model := range models {
		m := any(&model).(interfaces.Model)
		if m.GetId().IsZero() {
			m.SetId(primitive.NewObjectID())
		}
		_models = append(_models, m)
	}
	res, err := svc.col.GetCollection().InsertMany(ctx, _models)
	if err != nil {
		return nil, err
	}
	for _, v := range res.InsertedIDs {
		ids = append(ids, v.(primitive.ObjectID))
	}
	return ids, nil
}

func (svc *ModelService[T]) UpsertOne(query bson.M, model T) (id primitive.ObjectID, err error) {
	opts := options.ReplaceOptions{}
	opts.SetUpsert(true)
	result, err := svc.col.GetCollection().ReplaceOne(svc.col.GetContext(), query, model, &opts)
	if err != nil {
		return primitive.NilObjectID, err
	}

	if result.UpsertedID != nil {
		// If document was inserted
		return result.UpsertedID.(primitive.ObjectID), nil
	}

	// If document was updated, get its ID from the model
	m := any(&model).(interfaces.Model)
	return m.GetId(), nil
}

func (svc *ModelService[T]) UpsertOneContext(ctx context.Context, query bson.M, model T) (id primitive.ObjectID, err error) {
	opts := options.ReplaceOptions{}
	opts.SetUpsert(true)
	result, err := svc.col.GetCollection().ReplaceOne(ctx, query, model, &opts)
	if err != nil {
		return primitive.NilObjectID, err
	}

	if result.UpsertedID != nil {
		// If document was inserted
		return result.UpsertedID.(primitive.ObjectID), nil
	}

	// If document was updated, get its ID from the query or model
	if id, ok := query["_id"].(primitive.ObjectID); ok {
		return id, nil
	}
	m := any(&model).(interfaces.Model)
	return m.GetId(), nil
}

func (svc *ModelService[T]) Count(query bson.M) (total int, err error) {
	return svc.col.Count(query)
}

func (svc *ModelService[T]) AggregateAll(pipeline []bson.D, result any) (err error) {
	return svc.AggregateWithOptionsAll(pipeline, nil, result)
}

func (svc *ModelService[T]) AggregateWithOptionsAll(pipeline []bson.D, opts *options.AggregateOptions, result any) (err error) {
	return svc.aggregateWithOptions(pipeline, opts).All(result)
}

func (svc *ModelService[T]) AggregateOne(pipeline []bson.D, result any) (err error) {
	return svc.AggregateWithOptionsOne(pipeline, nil, result)
}

func (svc *ModelService[T]) AggregateWithOptionsOne(pipeline []bson.D, opts *options.AggregateOptions, result any) (err error) {
	return svc.aggregateWithOptions(pipeline, opts).One(result)
}

func (svc *ModelService[T]) aggregateWithOptions(pipeline []bson.D, opts *options.AggregateOptions) (fr *mongo.FindResult) {
	return svc.GetCol().Aggregate(pipeline, opts)
}

func GetCollectionNameByInstance(v any) string {
	t := reflect.TypeOf(v)
	field := t.Field(0)
	return field.Tag.Get("collection")
}

func GetCollectionName[T any]() string {
	var instance T
	return getCollectionNameFromType(reflect.TypeOf(instance))
}

// getCollectionNameFromType recursively searches for collection tag in struct hierarchy
// The function follows the AstianGO Hub pattern where collection tags are typically on the first field
func getCollectionNameFromType(t reflect.Type) string {
	// Handle pointer types
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	// Must be a struct
	if t.Kind() != reflect.Struct {
		return ""
	}

	// Check if struct has any fields
	if t.NumField() == 0 {
		return ""
	}

	// Priority 1: Check the first field for collection tag (AstianGO Hub standard pattern)
	// Most AstianGO Hub models have: `any collection:"collection_name"` as first field
	field := t.Field(0)
	if collectionName := field.Tag.Get("collection"); collectionName != "" {
		return collectionName
	}

	// Priority 2: If first field is an embedded struct, recursively check it
	// This handles DTO patterns like SpiderDTO embedding Spider
	if field.Type.Kind() == reflect.Struct && field.Anonymous {
		if collectionName := getCollectionNameFromType(field.Type); collectionName != "" {
			return collectionName
		}
	}

	// Priority 3: Fallback - check all remaining fields for collection tag
	// This provides robustness for non-standard patterns
	for i := 1; i < t.NumField(); i++ {
		field := t.Field(i)
		if collectionName := field.Tag.Get("collection"); collectionName != "" {
			return collectionName
		}

		// Also recursively check embedded structs in other positions
		if field.Type.Kind() == reflect.Struct && field.Anonymous {
			if collectionName := getCollectionNameFromType(field.Type); collectionName != "" {
				return collectionName
			}
		}
	}

	return ""
}

func GetCollection[T any]() *mongo.Col {
	return mongo.GetMongoCol(GetCollectionName[T]())
}

// NewModelService return singleton instance of ModelService
func NewModelService[T any]() *ModelService[T] {
	typeName := fmt.Sprintf("%T", *new(T))

	mu.Lock()
	defer mu.Unlock()

	if _, exists := onceMap[typeName]; !exists {
		onceMap[typeName] = &sync.Once{}
	}

	var instance *ModelService[T]

	onceMap[typeName].Do(func() {
		collectionName := GetCollectionName[T]()
		collection := mongo.GetMongoCol(collectionName)
		instance = &ModelService[T]{col: collection}
		instanceMap[typeName] = instance
	})

	return instanceMap[typeName].(*ModelService[T])
}

func NewModelServiceWithColName[T any](colName string) *ModelService[T] {
	mu.Lock()
	defer mu.Unlock()

	if _, exists := onceColNameMap[colName]; !exists {
		onceColNameMap[colName] = new(sync.Once)
	}

	var instance *ModelService[T]

	onceColNameMap[colName].Do(func() {
		collection := mongo.GetMongoCol(colName)
		instance = &ModelService[T]{col: collection}
		instanceMap[colName] = instance
	})

	return instanceMap[colName].(*ModelService[T])
}

func GetDefaultJoinPipeline[T any]() []bson.D {
	return []bson.D{
		GetDefaultLookupPipeline[T](),
		GetDefaultUnwindPipeline[T](),
	}
}

func GetJoinPipeline[T any](localField, foreignField, as string) []bson.D {
	return []bson.D{
		GetLookupPipeline[T](localField, foreignField, as),
		GetUnwindPipeline(as),
	}
}

func GetDefaultLookupPipeline[T any]() bson.D {
	var model T
	typ := reflect.TypeOf(model)
	name := utils.ToSnakeCase(typ.Name())
	return GetLookupByNamePipeline[T](name)
}

func GetLookupByNamePipeline[T any](name string) bson.D {
	localField := fmt.Sprintf("%s_id", name)
	foreignField := "_id"
	as := fmt.Sprintf("_%s", name)
	return GetLookupPipeline[T](localField, foreignField, as)
}

func GetLookupPipeline[T any](localField, foreignField, as string) bson.D {
	return bson.D{{
		Key: "$lookup",
		Value: bson.M{
			"from":         GetCollectionName[T](),
			"localField":   localField,
			"foreignField": foreignField,
			"as":           as,
		}},
	}
}

func GetDefaultUnwindPipeline[T any]() bson.D {
	var model T
	typ := reflect.TypeOf(model)
	name := utils.ToSnakeCase(typ.Name())
	as := fmt.Sprintf("_%s", name)
	return GetUnwindPipeline(as)
}

func GetUnwindPipeline(as string) bson.D {
	return bson.D{{
		Key: "$unwind",
		Value: bson.M{
			"path":                       fmt.Sprintf("$%s", as),
			"preserveNullAndEmptyArrays": true,
		}},
	}
}

func GetPaginationPipeline(query bson.M, sort bson.D, skip, limit int) []bson.D {
	if query == nil {
		query = bson.M{}
	}
	if sort == nil {
		sort = bson.D{{"_id", -1}}
	}
	return []bson.D{
		{{Key: "$match", Value: query}},
		{{Key: "$sort", Value: sort}},
		{{Key: "$skip", Value: skip}},
		{{Key: "$limit", Value: limit}},
	}
}

func GetByIdPipeline(id primitive.ObjectID) []bson.D {
	return []bson.D{
		{{Key: "$match", Value: bson.M{"_id": id}}},
		{{Key: "$limit", Value: 1}},
	}
}
