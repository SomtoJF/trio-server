package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"

	"github.com/qdrant/go-client/qdrant"
	"github.com/somtojf/trio-server/initializers"
	"github.com/somtojf/trio-server/types/qdranttypes"
)

func init() {
	initializers.LoadEnvVariables()
	initializers.ConnectToQdrant()
}

func main() {
	err := createQdrantCollections(initializers.QdrantClient)
	if err != nil {
		log.Fatal(err)
	}
}

func createQdrantCollections(client *qdrant.Client) error {
	ctx := context.Background()
	collections := []qdranttypes.CollectionName{
		qdranttypes.COLLECTION_NAME_BASIC_MESSAGES,
	}

	for _, collection := range collections {
		exists, err := client.CollectionExists(ctx, string(collection))
		if err != nil {
			return fmt.Errorf("error checking collection existence: %w", err)
		}

		if exists {
			// Delete existing collection to recreate with proper indexes
			err = client.DeleteCollection(ctx, string(collection))
			if err != nil {
				return fmt.Errorf("error deleting collection %s: %w", collection, err)
			}
			slog.Info("Deleted existing collection", "collection", collection)
		}

		// Create collection with configuration
		var indexingThreshold uint64
		indexingThreshold = 20000

		err = client.CreateCollection(ctx, &qdrant.CreateCollection{
			CollectionName: string(collection),
			VectorsConfig: qdrant.NewVectorsConfig(&qdrant.VectorParams{
				Size:     uint64(qdranttypes.VECTOR_SIZE_BASIC_MESSAGE),
				Distance: qdrant.Distance_Cosine,
			}),
			OptimizersConfig: &qdrant.OptimizersConfigDiff{
				IndexingThreshold: &indexingThreshold,
			},
		})

		if err != nil {
			return fmt.Errorf("error creating collection %s: %w", collection, err)
		}

		// Create indexes for faster searching
		err = createIndexes(ctx, client, collection)
		if err != nil {
			return fmt.Errorf("error creating indexes for collection %s: %w", collection, err)
		}

		slog.Info("Successfully created collection with indexes", "collection", collection)
	}

	return nil
}

func createIndexes(ctx context.Context, client *qdrant.Client, collection qdranttypes.CollectionName) error {
	// Define field types for different fields
	fieldConfigs := map[string]*qdrant.FieldType{
		"content":     qdrant.FieldType_FieldTypeText.Enum(),
		"chat_id":     qdrant.FieldType_FieldTypeKeyword.Enum(),
		"external_id": qdrant.FieldType_FieldTypeKeyword.Enum(),
	}

	// Create payload indexes for common search fields
	for field, fieldType := range fieldConfigs {
		_, err := client.CreateFieldIndex(ctx, &qdrant.CreateFieldIndexCollection{
			CollectionName: string(collection),
			FieldName:      field,
			FieldType:      fieldType,
		})
		if err != nil {
			return fmt.Errorf("error creating index for %s: %w", field, err)
		}
		slog.Info("Created index", "field", field, "type", fieldType)
	}

	return nil
}
