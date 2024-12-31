package openai

import (
	"context"
	"fmt"

	"github.com/sashabaranov/go-openai"
	"github.com/somtojf/trio-server/aipi/aipitypes"
)

type Client struct {
	client *openai.Client
}

func NewClient(openaiClient *openai.Client) *Client {
	return &Client{client: openaiClient}
}

func (c *Client) GetCompletion(ctx context.Context, request aipitypes.AIPIRequest) (aipitypes.AIPIResponse, error) {
	resp, err := c.client.CreateChatCompletion(
		ctx,
		openai.ChatCompletionRequest{
			Model: request.Model,
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    "system",
					Content: request.SystemMessage,
				},
				{
					Role:    "user",
					Content: request.UserMessage,
				},
			},
		},
	)
	if err != nil {
		return aipitypes.AIPIResponse{}, err
	}

	return aipitypes.AIPIResponse{
		Data: resp.Choices[0].Message.Content,
	}, nil
}

func (p *Client) GetEmbedding(ctx context.Context, request aipitypes.EmbeddingRequest) ([]float32, error) {
	embReq := &openai.EmbeddingRequest{
		Input:          request.Input,
		Model:          openai.EmbeddingModel(request.Model),
		EncodingFormat: openai.EmbeddingEncodingFormat(request.EncodingFormat),
		Dimensions:     request.Dimensions,
	}

	response, err := p.client.CreateEmbeddings(ctx, embReq)
	if err != nil {
		return nil, fmt.Errorf("error creating embedding: %w", err)
	}
	return response.Data[0].Embedding, nil
}
