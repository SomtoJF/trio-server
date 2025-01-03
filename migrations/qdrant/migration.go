package main

import (
	"context"
	"fmt"
	"log"

	"github.com/qdrant/go-client/qdrant"
	"github.com/somtojf/trio-server/initializers"
	"github.com/somtojf/trio-server/types/qdranttypes"
)

func main() {
	err := createQdrantCollections(initializers.QdrantClient)
	if err != nil {
		log.Fatal(err)
	}
}

func createQdrantCollections(client *qdrant.Client) error {
	ctx := context.Background()

	exists, err := client.CollectionExists(ctx, string(qdranttypes.COLLECTION_NAME_BASIC_MESSAGES))
	if err != nil {
		return fmt.Errorf("error checking collection existence: %w", err)
	}

	if exists {
		return nil
	}

	// Collection doesn't exist, create it
	err = client.CreateCollection(ctx, &qdrant.CreateCollection{
		CollectionName: string(qdranttypes.COLLECTION_NAME_BASIC_MESSAGES),
		VectorsConfig: qdrant.NewVectorsConfig(&qdrant.VectorParams{
			Size:     uint64(qdranttypes.VECTOR_SIZE_BASIC_MESSAGE),
			Distance: qdrant.Distance_Cosine,
		}),
	})

	if err != nil {
		return fmt.Errorf("error creating collection: %w", err)
	}

	return nil
}
