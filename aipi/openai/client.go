package openai

import (
	"context"

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
