package initializers

import (
	"fmt"
	"log"
	"log/slog"
	"os"
	"strings"

	"github.com/qdrant/go-client/qdrant"
)

var QdrantClient *qdrant.Client

func ConnectToQdrant() {
	var err error
	host := os.Getenv("QDRANT_HOST")

	// Determine if we're connecting to cloud or local
	useTLS := !strings.Contains(host, "localhost") && !strings.Contains(host, "127.0.0.1")

	QdrantClient, err = qdrant.NewClient(&qdrant.Config{
		Host:   host,
		Port:   6334, // gRPC port
		APIKey: os.Getenv("QDRANT_DB_API_KEY"),
		UseTLS: useTLS,
	})
	if err != nil {
		slog.Error("Failed to connect to qdrant", "error", err)
		log.Fatal(fmt.Errorf("error connecting to qdrant: %w", err))
	}
	slog.Info("Successfully connected to qdrant",
		"host", host,
		"useTLS", useTLS,
		"hasAPIKey", os.Getenv("QDRANT_DB_API_KEY") != "",
	)
}
