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
		qdranttypes.COLLECTION_NAME_REFLECTION_MESSAGES,
	}

	for _, collection := range collections {
		exists, err := client.CollectionExists(ctx, string(collection))
		if err != nil {
			return fmt.Errorf("error checking collection existence: %w", err)
		}

		if exists {
			slog.Info("Collection already exists, skipping", "collection", collection)
			continue // Skip to next collection instead of returning
		}

		// Collection doesn't exist, create it with configuration
		var indexingThreshold uint64
		indexingThreshold = 20000

		err = client.CreateCollection(ctx, &qdrant.CreateCollection{
			CollectionName: string(collection),
			VectorsConfig: qdrant.NewVectorsConfig(&qdrant.VectorParams{
				Size:     uint64(qdranttypes.VECTOR_SIZE_BASIC_MESSAGE),
				Distance: qdrant.Distance_Cosine,
			}),
			// Add optional but recommended configurations
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

		slog.Info("Successfully created collection", "collection", collection)
	}

	return nil
}

func createIndexes(ctx context.Context, client *qdrant.Client, collection qdranttypes.CollectionName) error {
	// Create payload indexes for common search fields
	indexesToCreate := []string{"content", "chat_id", "external_id"}

	for _, field := range indexesToCreate {
		_, err := client.CreateFieldIndex(ctx, &qdrant.CreateFieldIndexCollection{
			CollectionName: string(collection),
			FieldName:      field,
			FieldType:      qdrant.FieldType_FieldTypeText.Enum(),
		})
		if err != nil {
			return fmt.Errorf("error creating index for %s: %w", field, err)
		}
	}

	return nil
}
