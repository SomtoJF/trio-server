package initializers

import (
	"log/slog"
	"os"

	"github.com/qdrant/go-client/qdrant"
)

var QdrantClient *qdrant.Client

func ConnectToQdrant() error {
	var err error

	QdrantClient, err = qdrant.NewClient(&qdrant.Config{
		Host:   os.Getenv("QDRANT_HOST"),
		Port:   6334,
		APIKey: os.Getenv("QDRANT_DB_API_KEY"),
	})
	if err != nil {
		slog.Error("Failed to connect to qdrant", "error", err)
		return err
	}
	slog.Info("Successfully connected to qdrant")
	return nil
}
