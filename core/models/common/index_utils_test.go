package common

import (
	"fmt"
	"github.com/goastian/astiango-hub/core/mongo"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/bson"
	mongo2 "go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func TestRecreateIndexes(t *testing.T) {
	// Setup test collection
	testCol := mongo.GetMongoCol("test_collection")
	defer func() {
		_ = testCol.GetCollection().Drop(testCol.GetContext())
	}()

	// Test cases
	tests := []struct {
		name           string
		desiredIndexes []mongo2.IndexModel
		expectedCount  int64
	}{
		{
			name: "Create new indexes",
			desiredIndexes: []mongo2.IndexModel{
				{Keys: bson.D{{Key: "field1", Value: 1}}},
				{Keys: bson.D{{Key: "field2", Value: -1}}},
			},
			expectedCount: 3, // Including _id index
		},
		{
			name: "Update existing indexes",
			desiredIndexes: []mongo2.IndexModel{
				{Keys: bson.D{{Key: "field1", Value: 1}}},
				{Keys: bson.D{{Key: "field3", Value: 1}}},
			},
			expectedCount: 3,
		},
		{
			name: "Compound indexes",
			desiredIndexes: []mongo2.IndexModel{
				{Keys: bson.D{
					{Key: "field1", Value: 1},
					{Key: "field2", Value: -1},
				}},
				{Keys: bson.D{
					{Key: "field3", Value: 1},
					{Key: "field4", Value: 1},
				}},
			},
			expectedCount: 3,
		},
		{
			name: "Unique and sparse indexes",
			desiredIndexes: []mongo2.IndexModel{
				{
					Keys:    bson.D{{Key: "email", Value: 1}},
					Options: options.Index().SetUnique(true),
				},
				{
					Keys:    bson.D{{Key: "optional_field", Value: 1}},
					Options: options.Index().SetSparse(true),
				},
			},
			expectedCount: 3,
		},
		{
			name: "Mixed index types",
			desiredIndexes: []mongo2.IndexModel{
				{Keys: bson.D{
					{Key: "field1", Value: 1},
					{Key: "field2", Value: -1},
				}},
				{
					Keys:    bson.D{{Key: "unique_field", Value: 1}},
					Options: options.Index().SetUnique(true),
				},
			},
			expectedCount: 3,
		},
		{
			name: "Complex compound index with options",
			desiredIndexes: []mongo2.IndexModel{
				{
					Keys: bson.D{
						{Key: "category", Value: 1},
						{Key: "timestamp", Value: -1},
						{Key: "status", Value: 1},
					},
					Options: options.Index().
						SetUnique(true).
						SetPartialFilterExpression(bson.D{
							{Key: "status", Value: "active"},
						}),
				},
			},
			expectedCount: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Execute CreateIndexes
			CreateIndexes(testCol, tt.desiredIndexes)

			// Verify indexes
			indexes, err := testCol.ListIndexes()
			assert.NoError(t, err)

			// Check total number of indexes (including _id)
			assert.Equal(t, tt.expectedCount, int64(len(indexes)))

			// Verify each desired index exists
			for _, desiredIdx := range tt.desiredIndexes {
				found := false
				// Convert bson.D to normalized string representation
				desiredKeyDoc := desiredIdx.Keys.(bson.D)
				desiredKeyMap := make(map[string]interface{})
				for _, elem := range desiredKeyDoc {
					desiredKeyMap[elem.Key] = elem.Value
				}

				for _, existingIdx := range indexes {
					if existingIdx["name"].(string) != "_id_" {
						existingKey := existingIdx["key"].(map[string]interface{})

						// Compare maps by converting to strings and normalizing
						if compareIndexKeys(desiredKeyMap, existingKey) {
							found = true
							break
						}
					}
				}
				assert.True(t, found, "Index not found: %v", desiredIdx.Keys)
			}
		})
	}
}

// compareIndexKeys compares two index key specifications
func compareIndexKeys(desired, existing map[string]interface{}) bool {
	if len(desired) != len(existing) {
		return false
	}

	for k, v1 := range desired {
		v2, exists := existing[k]
		if !exists {
			return false
		}

		// Convert values to strings for comparison
		// This handles different numeric types (int, float64, etc.)
		str1 := fmt.Sprintf("%v", v1)
		str2 := fmt.Sprintf("%v", v2)
		if str1 != str2 {
			return false
		}
	}
	return true
}
