package initializers

import (
	"fmt"
	"log"
	"log/slog"
	"os"

	"github.com/qdrant/go-client/qdrant"
)

var QdrantClient *qdrant.Client

func ConnectToQdrant() {
	var err error

	QdrantClient, err = qdrant.NewClient(&qdrant.Config{
		Host:   os.Getenv("QDRANT_HOST"),
		Port:   6334,
		APIKey: os.Getenv("QDRANT_DB_API_KEY"),
		UseTLS: true,
	})
	if err != nil {
		slog.Error("Failed to connect to qdrant", "error", err)
		log.Fatal(fmt.Errorf("error connecting to qdrant: %w", err))
	}
	slog.Info("Successfully connected to qdrant")
}
