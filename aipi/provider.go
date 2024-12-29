package aipi

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/generative-ai-go/genai"
	"github.com/sashabaranov/go-openai"
	"github.com/somtojf/trio-server/aipi/aipitypes"
	"github.com/somtojf/trio-server/aipi/gemini"
	openaiHelper "github.com/somtojf/trio-server/aipi/openai"
)

type Provider struct {
	genaiClient  *gemini.Client
	openaiClient *openaiHelper.Client
}

func NewProvider(genaiClient *genai.Client, openaiClient *openai.Client) *Provider {
	return &Provider{
		genaiClient:  gemini.NewClient(genaiClient),
		openaiClient: openaiHelper.NewClient(openaiClient),
	}
}

type AIPIClient interface {
	GetCompletion(ctx context.Context, request aipitypes.AIPIRequest) (aipitypes.AIPIResponse, error)
	GetCompletionAsync(ctx context.Context, request aipitypes.AIPIRequest) (string, error)
}

type AIPIClientFactory struct {
	geminiClient *genai.Client
	openaiClient *openai.Client
}

func (p *Provider) GetCompletion(ctx context.Context, request aipitypes.AIPIRequest) (aipitypes.AIPIResponse, error) {
	if strings.HasPrefix(request.Model, "gemini") {
		return p.genaiClient.GetCompletion(ctx, request)
	} else if strings.HasPrefix(request.Model, "gpt") {
		return p.openaiClient.GetCompletion(ctx, request)
	}
	return aipitypes.AIPIResponse{}, fmt.Errorf("unsupported model: %s", request.Model)
}

func (p *Provider) GetCompletionAsync(ctx context.Context, request aipitypes.AIPIRequest) (string, error) {
	return "", nil
}
