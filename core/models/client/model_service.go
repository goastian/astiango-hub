package client

import (
	"encoding/json"
	"fmt"
	"reflect"
	"sync"

	"github.com/goastian/astiango-hub/core/grpc/client"
	"github.com/goastian/astiango-hub/core/interfaces"
	"github.com/goastian/astiango-hub/core/mongo"
	nodeconfig "github.com/goastian/astiango-hub/core/node/config"
	"github.com/goastian/astiango-hub/grpc"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

var (
	instanceMap = make(map[string]interface{})
	onceMap     = make(map[string]*sync.Once)
	mu          sync.Mutex
)

type ModelService[T any] struct {
	cfg       interfaces.NodeConfigService
	modelType string
}

func (svc *ModelService[T]) GetById(id primitive.ObjectID) (model *T, err error) {
	ctx, cancel := client.GetGrpcClient().Context()
	defer cancel()
	modelClient, err := client.GetGrpcClient().GetModelBaseServiceClient()
	if err != nil {
		return nil, fmt.Errorf("failed to get model base service client: %v", err)
	}
	res, err := modelClient.GetById(ctx, &grpc.ModelServiceGetByIdRequest{
		NodeKey:   svc.cfg.GetNodeKey(),
		ModelType: svc.modelType,
		Id:        id.Hex(),
	})
	if err != nil {
		return nil, err
	}
	return svc.deserializeOne(res)
}

func (svc *ModelService[T]) GetOne(query bson.M, options *mongo.FindOptions) (model *T, err error) {
	ctx, cancel := client.GetGrpcClient().Context()
	defer cancel()
	queryData, err := json.Marshal(query)
	if err != nil {
		return nil, err
	}
	findOptionsData, err := json.Marshal(options)
	if err != nil {
		return nil, err
	}
	modelClient, err := client.GetGrpcClient().GetModelBaseServiceClient()
	if err != nil {
		return nil, fmt.Errorf("failed to get model base service client: %v", err)
	}
	res, err := modelClient.GetOne(ctx, &grpc.ModelServiceGetOneRequest{
		NodeKey:     svc.cfg.GetNodeKey(),
		ModelType:   svc.modelType,
		Query:       queryData,
		FindOptions: findOptionsData,
	})
	if err != nil {
		return nil, err
	}
	return svc.deserializeOne(res)
}

func (svc *ModelService[T]) GetMany(query bson.M, options *mongo.FindOptions) (models []T, err error) {
	ctx, cancel := client.GetGrpcClient().Context()
	defer cancel()
	queryData, err := json.Marshal(query)
	if err != nil {
		return nil, err
	}
	findOptionsData, err := json.Marshal(options)
	if err != nil {
		return nil, err
	}
	modelClient, err := client.GetGrpcClient().GetModelBaseServiceClient()
	if err != nil {
		return nil, fmt.Errorf("failed to get model base service client: %v", err)
	}
	res, err := modelClient.GetMany(ctx, &grpc.ModelServiceGetManyRequest{
		NodeKey:     svc.cfg.GetNodeKey(),
		ModelType:   svc.modelType,
		Query:       queryData,
		FindOptions: findOptionsData,
	})
	if err != nil {
		return nil, err
	}
	return svc.deserializeMany(res)
}

func (svc *ModelService[T]) DeleteById(id primitive.ObjectID) (err error) {
	ctx, cancel := client.GetGrpcClient().Context()
	defer cancel()
	modelClient, err := client.GetGrpcClient().GetModelBaseServiceClient()
	if err != nil {
		return fmt.Errorf("failed to get model base service client: %v", err)
	}
	_, err = modelClient.DeleteById(ctx, &grpc.ModelServiceDeleteByIdRequest{
		NodeKey:   svc.cfg.GetNodeKey(),
		ModelType: svc.modelType,
		Id:        id.Hex(),
	})
	if err != nil {
		return err
	}
	return nil
}

func (svc *ModelService[T]) DeleteOne(query bson.M) (err error) {
	ctx, cancel := client.GetGrpcClient().Context()
	defer cancel()
	queryData, err := json.Marshal(query)
	if err != nil {
		return err
	}
	modelClient, err := client.GetGrpcClient().GetModelBaseServiceClient()
	if err != nil {
		return fmt.Errorf("failed to get model base service client: %v", err)
	}
	_, err = modelClient.DeleteOne(ctx, &grpc.ModelServiceDeleteOneRequest{
		NodeKey:   svc.cfg.GetNodeKey(),
		ModelType: svc.modelType,
		Query:     queryData,
	})
	if err != nil {
		return err
	}
	return nil
}

func (svc *ModelService[T]) DeleteMany(query bson.M) (err error) {
	ctx, cancel := client.GetGrpcClient().Context()
	defer cancel()
	queryData, err := json.Marshal(query)
	if err != nil {
		return err
	}
	modelClient, err := client.GetGrpcClient().GetModelBaseServiceClient()
	if err != nil {
		return fmt.Errorf("failed to get model base service client: %v", err)
	}
	_, err = modelClient.DeleteMany(ctx, &grpc.ModelServiceDeleteManyRequest{
		NodeKey:   svc.cfg.GetNodeKey(),
		ModelType: svc.modelType,
		Query:     queryData,
	})
	if err != nil {
		return err
	}
	return nil
}

func (svc *ModelService[T]) UpdateById(id primitive.ObjectID, update bson.M) (err error) {
	ctx, cancel := client.GetGrpcClient().Context()
	defer cancel()
	updateData, err := json.Marshal(update)
	if err != nil {
		return err
	}
	modelClient, err := client.GetGrpcClient().GetModelBaseServiceClient()
	if err != nil {
		return fmt.Errorf("failed to get model base service client: %v", err)
	}
	_, err = modelClient.UpdateById(ctx, &grpc.ModelServiceUpdateByIdRequest{
		NodeKey:   svc.cfg.GetNodeKey(),
		ModelType: svc.modelType,
		Id:        id.Hex(),
		Update:    updateData,
	})
	if err != nil {
		return err
	}
	return nil
}

func (svc *ModelService[T]) UpdateOne(query bson.M, update bson.M) (err error) {
	ctx, cancel := client.GetGrpcClient().Context()
	defer cancel()
	queryData, err := json.Marshal(query)
	if err != nil {
		return err
	}
	updateData, err := json.Marshal(update)
	if err != nil {
		return err
	}
	modelClient, err := client.GetGrpcClient().GetModelBaseServiceClient()
	if err != nil {
		return fmt.Errorf("failed to get model base service client: %v", err)
	}
	_, err = modelClient.UpdateOne(ctx, &grpc.ModelServiceUpdateOneRequest{
		NodeKey:   svc.cfg.GetNodeKey(),
		ModelType: svc.modelType,
		Query:     queryData,
		Update:    updateData,
	})
	if err != nil {
		return err
	}
	return nil
}

func (svc *ModelService[T]) UpdateMany(query bson.M, update bson.M) (err error) {
	ctx, cancel := client.GetGrpcClient().Context()
	defer cancel()
	queryData, err := json.Marshal(query)
	if err != nil {
		return err
	}
	updateData, err := json.Marshal(update)
	if err != nil {
		return err
	}
	modelClient, err := client.GetGrpcClient().GetModelBaseServiceClient()
	if err != nil {
		return fmt.Errorf("failed to get model base service client: %v", err)
	}
	_, err = modelClient.UpdateMany(ctx, &grpc.ModelServiceUpdateManyRequest{
		NodeKey:   svc.cfg.GetNodeKey(),
		ModelType: svc.modelType,
		Query:     queryData,
		Update:    updateData,
	})
	return nil
}

func (svc *ModelService[T]) ReplaceById(id primitive.ObjectID, model T) (err error) {
	ctx, cancel := client.GetGrpcClient().Context()
	defer cancel()
	modelData, err := json.Marshal(model)
	if err != nil {
		return err
	}
	modelClient, err := client.GetGrpcClient().GetModelBaseServiceClient()
	if err != nil {
		return fmt.Errorf("failed to get model base service client: %v", err)
	}
	_, err = modelClient.ReplaceById(ctx, &grpc.ModelServiceReplaceByIdRequest{
		NodeKey:   svc.cfg.GetNodeKey(),
		ModelType: svc.modelType,
		Id:        id.Hex(),
		Model:     modelData,
	})
	if err != nil {
		return err
	}
	return nil
}

func (svc *ModelService[T]) ReplaceOne(query bson.M, model T) (err error) {
	ctx, cancel := client.GetGrpcClient().Context()
	defer cancel()
	queryData, err := json.Marshal(query)
	if err != nil {
		return err
	}
	modelData, err := json.Marshal(model)
	if err != nil {
		return err
	}
	modelClient, err := client.GetGrpcClient().GetModelBaseServiceClient()
	if err != nil {
		return fmt.Errorf("failed to get model base service client: %v", err)
	}
	_, err = modelClient.ReplaceOne(ctx, &grpc.ModelServiceReplaceOneRequest{
		NodeKey:   svc.cfg.GetNodeKey(),
		ModelType: svc.modelType,
		Query:     queryData,
		Model:     modelData,
	})
	if err != nil {
		return err
	}
	return nil
}

func (svc *ModelService[T]) InsertOne(model T) (id primitive.ObjectID, err error) {
	ctx, cancel := client.GetGrpcClient().Context()
	defer cancel()
	modelData, err := json.Marshal(model)
	if err != nil {
		return primitive.NilObjectID, err
	}
	modelClient, err := client.GetGrpcClient().GetModelBaseServiceClient()
	if err != nil {
		return primitive.NilObjectID, fmt.Errorf("failed to get model base service client: %v", err)
	}
	res, err := modelClient.InsertOne(ctx, &grpc.ModelServiceInsertOneRequest{
		NodeKey:   svc.cfg.GetNodeKey(),
		ModelType: svc.modelType,
		Model:     modelData,
	})
	if err != nil {
		return primitive.NilObjectID, err
	}
	return deserialize[primitive.ObjectID](res)
}

func (svc *ModelService[T]) InsertMany(models []T) (ids []primitive.ObjectID, err error) {
	ctx, cancel := client.GetGrpcClient().Context()
	defer cancel()
	modelsData, err := json.Marshal(models)
	if err != nil {
		return nil, err
	}
	modelClient, err := client.GetGrpcClient().GetModelBaseServiceClient()
	if err != nil {
		return nil, fmt.Errorf("failed to get model base service client: %v", err)
	}
	res, err := modelClient.InsertMany(ctx, &grpc.ModelServiceInsertManyRequest{
		NodeKey:   svc.cfg.GetNodeKey(),
		ModelType: svc.modelType,
		Models:    modelsData,
	})
	if err != nil {
		return nil, err
	}
	return deserialize[[]primitive.ObjectID](res)
}

func (svc *ModelService[T]) UpsertOne(query bson.M, model T) (id primitive.ObjectID, err error) {
	ctx, cancel := client.GetGrpcClient().Context()
	defer cancel()
	queryData, err := json.Marshal(query)
	if err != nil {
		return primitive.NilObjectID, err
	}
	modelData, err := json.Marshal(model)
	if err != nil {
		return primitive.NilObjectID, err
	}
	modelClient, err := client.GetGrpcClient().GetModelBaseServiceClient()
	if err != nil {
		return primitive.NilObjectID, fmt.Errorf("failed to get model base service client: %v", err)
	}
	res, err := modelClient.UpsertOne(ctx, &grpc.ModelServiceUpsertOneRequest{
		NodeKey:   svc.cfg.GetNodeKey(),
		ModelType: svc.modelType,
		Query:     queryData,
		Model:     modelData,
	})
	if err != nil {
		return primitive.NilObjectID, err
	}

	return deserialize[primitive.ObjectID](res)
}

func (svc *ModelService[T]) Count(query bson.M) (total int, err error) {
	ctx, cancel := client.GetGrpcClient().Context()
	defer cancel()
	queryData, err := json.Marshal(query)
	if err != nil {
		return 0, err
	}
	modelClient, err := client.GetGrpcClient().GetModelBaseServiceClient()
	if err != nil {
		return 0, fmt.Errorf("failed to get model base service client: %v", err)
	}
	res, err := modelClient.Count(ctx, &grpc.ModelServiceCountRequest{
		NodeKey:   svc.cfg.GetNodeKey(),
		ModelType: svc.modelType,
		Query:     queryData,
	})
	if err != nil {
		return 0, err
	}
	return deserialize[int](res)
}

func (svc *ModelService[T]) GetCol() (col *mongo.Col) {
	return nil
}

func (svc *ModelService[T]) deserializeOne(res *grpc.Response) (result *T, err error) {
	r, err := deserialize[T](res)
	if err != nil {
		return nil, err
	}
	return &r, err
}

func (svc *ModelService[T]) deserializeMany(res *grpc.Response) (results []T, err error) {
	return deserialize[[]T](res)
}

func deserialize[T any](res *grpc.Response) (result T, err error) {
	err = json.Unmarshal(res.Data, &result)
	if err != nil {
		return result, err
	}
	return result, nil
}

func NewModelService[T any]() *ModelService[T] {
	mu.Lock()
	defer mu.Unlock()

	var v T
	t := reflect.TypeOf(v)
	typeName := t.Name()

	if _, exists := onceMap[typeName]; !exists {
		onceMap[typeName] = new(sync.Once)
	}

	var instance *ModelService[T]

	onceMap[typeName].Do(func() {
		instance = &ModelService[T]{
			cfg:       nodeconfig.GetNodeConfigService(),
			modelType: typeName,
		}
		instanceMap[typeName] = instance
	})

	return instanceMap[typeName].(*ModelService[T])
}
