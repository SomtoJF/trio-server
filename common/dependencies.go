package common

import (
	"context"
	"os"

	"github.com/google/generative-ai-go/genai"
	"github.com/sashabaranov/go-openai"
	"github.com/somtojf/trio-server/aipi"
	"google.golang.org/api/option"
)

type Dependencies struct {
	AIPIProvider *aipi.Provider
}

func NewDependencies(ctx context.Context) (*Dependencies, error) {
	openaiClient := openai.NewClient(os.Getenv("OPENAI_API_KEY"))

	genaiClient, err := genai.NewClient(ctx, option.WithAPIKey(os.Getenv("GEMINI_API_KEY")))
	if err != nil {
		return nil, err
	}

	// Create the AIPI provider with both clients
	aipiProvider := aipi.NewProvider(genaiClient, openaiClient)

	return &Dependencies{
		AIPIProvider: aipiProvider,
	}, nil
}
