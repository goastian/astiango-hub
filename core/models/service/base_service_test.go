package service_test

import (
	"context"
	"testing"
	"time"

	"github.com/apex/log"
	"github.com/goastian/astiango-hub/core/models/models"
	"github.com/goastian/astiango-hub/core/mongo"

	"github.com/goastian/astiango-hub/core/models/service"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type TestModel struct {
	any              `collection:"testmodels"`
	Id               primitive.ObjectID `bson:"_id,omitempty"`
	models.BaseModel `bson:",inline"`
	Name             string `bson:"name"`
}

// TestModelBase represents a base model for testing embedded struct scenarios
type TestModelBase struct {
	any              `collection:"testbase"`
	Id               primitive.ObjectID `bson:"_id,omitempty"`
	models.BaseModel `bson:",inline"`
	BaseName         string `bson:"base_name"`
}

// TestModelDTO represents a DTO that embeds TestModelBase (similar to SpiderDTO pattern)
type TestModelDTO struct {
	TestModelBase `json:",inline" bson:",inline"`
	ExtraField    string `json:"extra_field,omitempty" bson:"extra_field,omitempty"`
}

// TestModelNestedDTO represents a deeply nested DTO structure
type TestModelNestedDTO struct {
	TestModelDTO `json:",inline" bson:",inline"`
	NestedField  string `json:"nested_field,omitempty" bson:"nested_field,omitempty"`
}

// TestModelNonStandardPosition tests Priority 3: collection tag not on first field
type TestModelNonStandardPosition struct {
	SomeField        string `bson:"some_field"`
	AnotherField     string `bson:"another_field" collection:"non_standard_collection"`
	models.BaseModel `bson:",inline"`
}

// TestModelEmbeddedInSecondPosition tests Priority 3: embedded struct with collection tag in non-first position
type TestModelWithCollectionTag struct {
	any              `collection:"embedded_collection"`
	models.BaseModel `bson:",inline"`
	EmbeddedName     string `bson:"embedded_name"`
}

type TestModelEmbeddedInSecondPosition struct {
	FirstField                 string           `bson:"first_field"`
	TestModelWithCollectionTag `bson:",inline"` // Embedded struct in second position
	ThirdField                 string           `bson:"third_field"`
}

func setupTestDB() {
	viper.Set("mongo.db", "testdb")
}

func teardownTestDB() {
	db := mongo.GetMongoDb("testdb")
	err := db.Drop(context.Background())
	if err != nil {
		log.Errorf("dropping test db error: %v", err)
		return
	}
	log.Infof("dropped test db")
}

func TestModelService(t *testing.T) {
	setupTestDB()
	defer teardownTestDB()

	t.Run("GetById", func(t *testing.T) {
		svc := service.NewModelService[TestModel]()
		testModel := TestModel{Name: "GetById Test"}

		id, err := svc.InsertOne(testModel)
		require.Nil(t, err)
		time.Sleep(100 * time.Millisecond)

		result, err := svc.GetById(id)
		require.Nil(t, err)
		assert.Equal(t, testModel.Name, result.Name)
	})

	t.Run("GetOne", func(t *testing.T) {
		svc := service.NewModelService[TestModel]()
		testModel := TestModel{Name: "GetOne Test"}

		_, err := svc.InsertOne(testModel)
		require.Nil(t, err)
		time.Sleep(100 * time.Millisecond)

		result, err := svc.GetOne(bson.M{"name": "GetOne Test"}, nil)
		require.Nil(t, err)
		assert.Equal(t, testModel.Name, result.Name)
	})

	t.Run("GetMany", func(t *testing.T) {
		svc := service.NewModelService[TestModel]()
		testModels := []TestModel{
			{Name: "GetMany Test 1"},
			{Name: "GetMany Test 2"},
		}

		_, err := svc.InsertMany(testModels)
		require.Nil(t, err)
		time.Sleep(100 * time.Millisecond)

		results, err := svc.GetMany(bson.M{"name": bson.M{"$regex": "^GetMany Test"}}, nil)
		require.Nil(t, err)
		assert.Equal(t, 2, len(results))
	})

	t.Run("InsertOne", func(t *testing.T) {
		svc := service.NewModelService[TestModel]()
		testModel := TestModel{Name: "InsertOne Test"}

		id, err := svc.InsertOne(testModel)
		require.Nil(t, err)
		assert.NotEqual(t, primitive.NilObjectID, id)
	})

	t.Run("InsertMany", func(t *testing.T) {
		svc := service.NewModelService[TestModel]()
		testModels := []TestModel{
			{Name: "InsertMany Test 1"},
			{Name: "InsertMany Test 2"},
		}

		ids, err := svc.InsertMany(testModels)
		require.Nil(t, err)
		assert.Equal(t, 2, len(ids))
	})

	t.Run("UpdateById", func(t *testing.T) {
		svc := service.NewModelService[TestModel]()
		testModel := TestModel{Name: "UpdateById Test"}

		id, err := svc.InsertOne(testModel)
		require.Nil(t, err)
		time.Sleep(100 * time.Millisecond)

		update := bson.M{"$set": bson.M{"name": "UpdateById Test New Name"}}
		err = svc.UpdateById(id, update)
		require.Nil(t, err)
		time.Sleep(100 * time.Millisecond)

		result, err := svc.GetById(id)
		require.Nil(t, err)
		assert.Equal(t, "UpdateById Test New Name", result.Name)
	})

	t.Run("UpdateOne", func(t *testing.T) {
		svc := service.NewModelService[TestModel]()
		testModel := TestModel{Name: "UpdateOne Test"}

		_, err := svc.InsertOne(testModel)
		require.Nil(t, err)
		time.Sleep(100 * time.Millisecond)

		update := bson.M{"$set": bson.M{"name": "UpdateOne Test New Name"}}
		err = svc.UpdateOne(bson.M{"name": "UpdateOne Test"}, update)
		require.Nil(t, err)
		time.Sleep(100 * time.Millisecond)

		result, err := svc.GetOne(bson.M{"name": "UpdateOne Test New Name"}, nil)
		require.Nil(t, err)
		assert.Equal(t, "UpdateOne Test New Name", result.Name)
	})

	t.Run("UpdateMany", func(t *testing.T) {
		svc := service.NewModelService[TestModel]()
		testModels := []TestModel{
			{Name: "UpdateMany Test 1"},
			{Name: "UpdateMany Test 2"},
		}

		_, err := svc.InsertMany(testModels)
		require.Nil(t, err)
		time.Sleep(100 * time.Millisecond)

		update := bson.M{"$set": bson.M{"name": "UpdateMany Test New Name"}}
		err = svc.UpdateMany(bson.M{"name": bson.M{"$regex": "^UpdateMany Test"}}, update)
		require.Nil(t, err)
		time.Sleep(100 * time.Millisecond)

		results, err := svc.GetMany(bson.M{"name": "UpdateMany Test New Name"}, nil)
		require.Nil(t, err)
		assert.Equal(t, 2, len(results))
	})

	t.Run("DeleteById", func(t *testing.T) {
		svc := service.NewModelService[TestModel]()
		testModel := TestModel{Name: "DeleteById Test"}

		id, err := svc.InsertOne(testModel)
		require.Nil(t, err)
		time.Sleep(100 * time.Millisecond)

		err = svc.DeleteById(id)
		require.Nil(t, err)
		time.Sleep(100 * time.Millisecond)

		result, err := svc.GetById(id)
		assert.NotNil(t, err)
		assert.Nil(t, result)
	})

	t.Run("DeleteOne", func(t *testing.T) {
		svc := service.NewModelService[TestModel]()
		testModel := TestModel{Name: "DeleteOne Test"}

		_, err := svc.InsertOne(testModel)
		require.Nil(t, err)
		time.Sleep(100 * time.Millisecond)

		err = svc.DeleteOne(bson.M{"name": "DeleteOne Test"})
		require.Nil(t, err)
		time.Sleep(100 * time.Millisecond)

		result, err := svc.GetOne(bson.M{"name": "DeleteOne Test"}, nil)
		assert.NotNil(t, err)
		assert.Nil(t, result)
	})

	t.Run("DeleteMany", func(t *testing.T) {
		svc := service.NewModelService[TestModel]()
		testModels := []TestModel{
			{Name: "DeleteMany Test 1"},
			{Name: "DeleteMany Test 2"},
		}

		_, err := svc.InsertMany(testModels)
		require.Nil(t, err)
		time.Sleep(100 * time.Millisecond)

		err = svc.DeleteMany(bson.M{"name": bson.M{"$regex": "^DeleteMany Test"}})
		require.Nil(t, err)
		time.Sleep(100 * time.Millisecond)

		results, err := svc.GetMany(bson.M{"name": bson.M{"$regex": "^DeleteMany Test"}}, nil)
		require.Nil(t, err)
		assert.Equal(t, 0, len(results))
	})

	t.Run("Count", func(t *testing.T) {
		svc := service.NewModelService[TestModel]()
		testModels := []TestModel{
			{Name: "Count Test 1"},
			{Name: "Count Test 2"},
		}

		_, err := svc.InsertMany(testModels)
		require.Nil(t, err)
		time.Sleep(100 * time.Millisecond)

		total, err := svc.Count(bson.M{"name": bson.M{"$regex": "^Count Test"}})
		require.Nil(t, err)
		assert.Equal(t, 2, total)
	})
}

func TestEmbeddedStructHandling(t *testing.T) {
	setupTestDB()
	defer teardownTestDB()

	t.Run("GetCollectionName for embedded struct DTO", func(t *testing.T) {
		// Test that GetCollectionName correctly handles embedded struct (DTO pattern)
		collectionName := service.GetCollectionName[TestModelDTO]()
		assert.Equal(t, "testbase", collectionName, "Should extract collection name from embedded struct")
	})

	t.Run("GetCollectionName for deeply nested embedded struct", func(t *testing.T) {
		// Test deeply nested embedded struct
		collectionName := service.GetCollectionName[TestModelNestedDTO]()
		assert.Equal(t, "testbase", collectionName, "Should extract collection name from deeply nested embedded struct")
	})

	t.Run("GetCollectionName for regular struct", func(t *testing.T) {
		// Test that regular struct (non-embedded) still works
		collectionName := service.GetCollectionName[TestModel]()
		assert.Equal(t, "testmodels", collectionName, "Should get collection name from regular struct")

		collectionName = service.GetCollectionName[TestModelBase]()
		assert.Equal(t, "testbase", collectionName, "Should get collection name from base struct")
	})

	t.Run("GetCollectionName follows AstianGO Hub pattern", func(t *testing.T) {
		// Test that the function correctly identifies the AstianGO Hub pattern:
		// First field is `any` with collection tag (no field name)
		// This matches the actual pattern used in Spider, Node, Project, etc.

		// TestModel follows: any `collection:"testmodels"`
		collectionName := service.GetCollectionName[TestModel]()
		assert.Equal(t, "testmodels", collectionName, "Should extract collection from first anonymous any field")

		// TestModelBase follows: any `collection:"testbase"`
		collectionName = service.GetCollectionName[TestModelBase]()
		assert.Equal(t, "testbase", collectionName, "Should extract collection from first anonymous any field")
	})

	t.Run("ModelService with embedded struct DTO", func(t *testing.T) {
		// Test that ModelService works correctly with embedded struct DTO
		svc := service.NewModelService[TestModelDTO]()
		assert.NotNil(t, svc, "Should create service for DTO")

		// Verify the service uses the correct collection
		col := svc.GetCol()
		assert.NotNil(t, col, "Should have collection")
		// The collection name should be extracted from the embedded struct
		assert.Equal(t, "testbase", col.GetName(), "Should use collection name from embedded struct")
	})

	t.Run("CRUD operations with embedded struct DTO", func(t *testing.T) {
		// Test actual CRUD operations with DTO
		svc := service.NewModelService[TestModelDTO]()

		testDTO := TestModelDTO{
			TestModelBase: TestModelBase{BaseName: "Base Test"},
			ExtraField:    "Extra Test",
		}

		// Insert
		id, err := svc.InsertOne(testDTO)
		require.Nil(t, err)
		assert.NotEqual(t, primitive.NilObjectID, id)
		time.Sleep(100 * time.Millisecond)

		// Get by ID
		result, err := svc.GetById(id)
		require.Nil(t, err)
		assert.Equal(t, "Base Test", result.BaseName)
		assert.Equal(t, "Extra Test", result.ExtraField)

		// Update
		update := bson.M{"$set": bson.M{"extra_field": "Updated Extra"}}
		err = svc.UpdateById(id, update)
		require.Nil(t, err)
		time.Sleep(100 * time.Millisecond)

		// Verify update
		updated, err := svc.GetById(id)
		require.Nil(t, err)
		assert.Equal(t, "Updated Extra", updated.ExtraField)

		// Delete
		err = svc.DeleteById(id)
		require.Nil(t, err)
		time.Sleep(100 * time.Millisecond)

		// Verify deletion
		deleted, err := svc.GetById(id)
		assert.NotNil(t, err)
		assert.Nil(t, deleted)
	})

	t.Run("GetCollection function with embedded struct", func(t *testing.T) {
		// Test GetCollection function directly
		col := service.GetCollection[TestModelDTO]()
		assert.NotNil(t, col, "Should get collection for DTO")
		assert.Equal(t, "testbase", col.GetName(), "Should use collection name from embedded struct")

		// Test with regular struct
		col = service.GetCollection[TestModel]()
		assert.NotNil(t, col, "Should get collection for regular struct")
		assert.Equal(t, "testmodels", col.GetName(), "Should use collection name from struct")
	})

	t.Run("Multiple DTO instances share same collection", func(t *testing.T) {
		// Test that both base model and DTO use the same collection
		baseSvc := service.NewModelService[TestModelBase]()
		dtoSvc := service.NewModelService[TestModelDTO]()

		baseCol := baseSvc.GetCol()
		dtoCol := dtoSvc.GetCol()

		assert.Equal(t, baseCol.GetName(), dtoCol.GetName(), "Base and DTO should use same collection")
		assert.Equal(t, "testbase", baseCol.GetName(), "Should use correct collection name")
	})

	t.Run("Priority 3: Collection tag on non-first field", func(t *testing.T) {
		// Test Priority 3: fallback to check remaining fields for collection tag
		// TestModelNonStandardPosition has collection tag on second field
		collectionName := service.GetCollectionName[TestModelNonStandardPosition]()
		assert.Equal(t, "non_standard_collection", collectionName, "Should find collection tag on non-first field")
	})

	t.Run("Priority 3: Embedded struct in non-first position", func(t *testing.T) {
		// Test Priority 3: embedded struct with collection tag in non-first position
		// TestModelEmbeddedInSecondPosition has embedded struct as second field
		collectionName := service.GetCollectionName[TestModelEmbeddedInSecondPosition]()
		assert.Equal(t, "embedded_collection", collectionName, "Should find collection tag in embedded struct at non-first position")
	})

	t.Run("Priority 3: CRUD operations with non-standard position", func(t *testing.T) {
		// Test that ModelService works with non-standard collection tag position
		svc := service.NewModelService[TestModelNonStandardPosition]()

		testModel := TestModelNonStandardPosition{
			SomeField:    "Test Field",
			AnotherField: "Another Field",
		}

		// Insert
		id, err := svc.InsertOne(testModel)
		require.Nil(t, err)
		assert.NotEqual(t, primitive.NilObjectID, id)
		time.Sleep(100 * time.Millisecond)

		// Get by ID
		result, err := svc.GetById(id)
		require.Nil(t, err)
		assert.Equal(t, "Test Field", result.SomeField)
		assert.Equal(t, "Another Field", result.AnotherField)

		// Verify the service uses the correct collection
		col := svc.GetCol()
		assert.Equal(t, "non_standard_collection", col.GetName(), "Should use collection from non-first field")

		// Delete
		err = svc.DeleteById(id)
		require.Nil(t, err)
	})

	t.Run("Priority order verification", func(t *testing.T) {
		// This test verifies that priorities work as expected:
		// Priority 1: First field collection tag
		// Priority 2: First field embedded struct
		// Priority 3: Other fields

		// TestModel has collection tag on first field (Priority 1)
		collectionName := service.GetCollectionName[TestModel]()
		assert.Equal(t, "testmodels", collectionName, "Priority 1: First field collection tag")

		// TestModelDTO has embedded struct as first field (Priority 2)
		collectionName = service.GetCollectionName[TestModelDTO]()
		assert.Equal(t, "testbase", collectionName, "Priority 2: First field embedded struct")

		// TestModelNonStandardPosition has collection tag on second field (Priority 3)
		collectionName = service.GetCollectionName[TestModelNonStandardPosition]()
		assert.Equal(t, "non_standard_collection", collectionName, "Priority 3: Non-first field collection tag")

		// TestModelEmbeddedInSecondPosition has embedded struct in second position (Priority 3)
		collectionName = service.GetCollectionName[TestModelEmbeddedInSecondPosition]()
		assert.Equal(t, "embedded_collection", collectionName, "Priority 3: Embedded struct in non-first position")
	})
}
