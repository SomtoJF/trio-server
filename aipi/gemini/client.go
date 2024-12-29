package gemini

import (
	"context"
	"fmt"

	"github.com/google/generative-ai-go/genai"
	"github.com/somtojf/trio-server/aipi/aipitypes"
)

type Client struct {
	client *genai.Client
}

func NewClient(genaiClient *genai.Client) *Client {
	return &Client{client: genaiClient}
}

func (c *Client) GetCompletion(ctx context.Context, request aipitypes.AIPIRequest) (aipitypes.AIPIResponse, error) {
	model := c.client.GenerativeModel(request.Model)
	prompt := []genai.Part{
		genai.Text(request.SystemMessage + "\n" + request.UserMessage),
	}

	resp, err := model.GenerateContent(ctx, prompt...)
	if err != nil {
		return aipitypes.AIPIResponse{}, err
	}

	if len(resp.Candidates) == 0 {
		return aipitypes.AIPIResponse{}, fmt.Errorf("no response generated")
	}

	return aipitypes.AIPIResponse{
		Data: string(resp.Candidates[0].Content.Parts[0].(genai.Text)),
	}, nil
}
