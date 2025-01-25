package response

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"time"

	"github.com/somtojf/trio-server/aipi"
	"github.com/somtojf/trio-server/aipi/aipitypes"
	"gorm.io/gorm"
)

type SenderName string

const (
	HistoryMessageSenderNameAnswerer SenderName = "Answerer"
)

type HistoryMessage struct {
	SenderName string
	Content    string
	SentAt     time.Time
}

type AnswererInfoBank struct {
	IdUser            uint
	ChatHistory       []HistoryMessage
	Context           []HistoryMessage
	PreviousResponses []PreviousResponse
	Message           string
}

type PreviousResponse struct {
	AnswererResponse  AnswererResponse
	EvaluatorResponse EvaluatorResponse
}

type AnswererResponse struct {
	Content string
}

type EvaluatorInfoBank struct {
	IdUser            uint
	ChatHistory       []HistoryMessage
	Context           []HistoryMessage
	Message           string
	PreviousResponses []PreviousResponse
	AnswererResponse  AnswererResponse
}

type EvaluatorResponse struct {
	Content   string `json:"content"`
	IsOptimal bool   `json:"isOptimal"`
}

type Response struct {
	db   *gorm.DB
	aipi *aipi.Provider
}

func NewResponse(db *gorm.DB, aipi *aipi.Provider) *Response {
	return &Response{db: db, aipi: aipi}
}

func (r *Response) RunAnswerer(ctx context.Context, infoBank AnswererInfoBank, model string) (AnswererResponse, error) {
	systemTmpl, err := template.ParseFiles("controllers/reflection-chat/reflection-message/response/prompt/answerer/system/prompt.go.tmpl")
	if err != nil {
		log.Printf("Error parsing system template: %v", err)
		return AnswererResponse{}, fmt.Errorf("error parsing system template: %w", err)
	}
	var systemBuf bytes.Buffer
	if err := systemTmpl.Execute(&systemBuf, infoBank); err != nil {
		log.Printf("Error executing system template: %v", err)
		return AnswererResponse{}, fmt.Errorf("error executing system template: %w", err)
	}

	userTmpl, err := template.ParseFiles("controllers/reflection-chat/reflection-message/response/prompt/answerer/user/prompt.go.tmpl")
	if err != nil {
		log.Printf("Error parsing user template: %v", err)
		return AnswererResponse{}, fmt.Errorf("error parsing user template: %w", err)
	}
	var userBuf bytes.Buffer
	if err := userTmpl.Execute(&userBuf, infoBank); err != nil {
		log.Printf("Error executing user template: %v", err)
		return AnswererResponse{}, fmt.Errorf("error executing user template: %w", err)
	}

	request := &aipitypes.AIPIRequest{
		Model:         model,
		SystemMessage: systemBuf.String(),
		UserMessage:   userBuf.String(),
		IdUser:        infoBank.IdUser,
	}
	response, err := r.aipi.GetCompletion(ctx, *request)
	if err != nil {
		return AnswererResponse{}, err
	}

	return AnswererResponse{Content: response.Data}, nil
}

func (r *Response) RunEvaluator(ctx context.Context, infoBank EvaluatorInfoBank, model string) (EvaluatorResponse, error) {
	systemTmpl, err := template.ParseFiles("controllers/reflection-chat/reflection-message/response/prompt/answerer/system/prompt.go.tmpl")
	if err != nil {
		log.Printf("Error parsing system template: %v", err)
		return EvaluatorResponse{}, fmt.Errorf("error parsing system template: %w", err)
	}
	var systemBuf bytes.Buffer
	if err := systemTmpl.Execute(&systemBuf, infoBank); err != nil {
		log.Printf("Error executing system template: %v", err)
		return EvaluatorResponse{}, fmt.Errorf("error executing system template: %w", err)
	}

	userTmpl, err := template.ParseFiles("controllers/reflection-chat/reflection-message/response/prompt/answerer/user/prompt.go.tmpl")
	if err != nil {
		log.Printf("Error parsing user template: %v", err)
		return EvaluatorResponse{}, fmt.Errorf("error parsing user template: %w", err)
	}
	var userBuf bytes.Buffer
	if err := userTmpl.Execute(&userBuf, infoBank); err != nil {
		log.Printf("Error executing user template: %v", err)
		return EvaluatorResponse{}, fmt.Errorf("error executing user template: %w", err)
	}

	request := &aipitypes.AIPIRequest{
		Model:          model,
		SystemMessage:  systemBuf.String(),
		UserMessage:    userBuf.String(),
		IdUser:         infoBank.IdUser,
		ResponseFormat: aipitypes.AIPI_RESPONSE_FORMAT_JSON,
	}

	response, err := r.aipi.GetCompletion(ctx, *request)
	if err != nil {
		return EvaluatorResponse{}, err
	}

	var evaluatorResponse EvaluatorResponse

	err = json.Unmarshal([]byte(response.Data), &evaluatorResponse)
	if err != nil {
		log.Printf("Error unmarshalling evaluator response: %v", err)
		return EvaluatorResponse{}, fmt.Errorf("error unmarshalling evaluator response: %w", err)
	}

	return evaluatorResponse, nil
}
