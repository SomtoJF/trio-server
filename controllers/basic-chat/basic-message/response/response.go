package response

import (
	"bytes"
	"context"
	"fmt"
	"html/template"
	"log"
	"time"

	"github.com/somtojf/trio-server/aipi"
	"github.com/somtojf/trio-server/aipi/aipitypes"
	"gorm.io/gorm"
)

type AgentInformation struct {
	AgentName   string   `json:"agentName"`
	AgentTraits []string `json:"agentTraits"`
}

type HistoryMessage struct {
	SenderName string    `json:"senderName"`
	Content    string    `json:"content"`
	SentAt     time.Time `json:"sentAt"`
}

type InfoBank struct {
	IdUser           uint               `json:"idUser"`
	NewMessage       string             `json:"newMessage"`
	AgentInformation AgentInformation   `json:"agentInformation"`
	OtherAgents      []AgentInformation `json:"otherAgents"`
	ChatHistory      []HistoryMessage   `json:"chatHistory"`
	RelevantContext  []HistoryMessage   `json:"relevantContext"`
}

type RunResponse struct {
	Content string `json:"content"`
}

type Response struct {
	db   *gorm.DB
	aipi *aipi.Provider
}

func NewResponse(db *gorm.DB, aipi *aipi.Provider) *Response {
	return &Response{db: db, aipi: aipi}
}

func (r *Response) Run(ctx context.Context, infoBank InfoBank, model string) (RunResponse, error) {
	systemTmpl, err := template.ParseFiles("controllers/basic-chat/basic-message/response/prompt/system/prompt.go.tmpl")
	if err != nil {
		log.Printf("Error parsing system template: %v", err)
		return RunResponse{}, fmt.Errorf("error parsing system template: %w", err)
	}
	var systemBuf bytes.Buffer
	if err := systemTmpl.Execute(&systemBuf, infoBank); err != nil {
		log.Printf("Error executing system template: %v", err)
		return RunResponse{}, fmt.Errorf("error executing system template: %w", err)
	}

	userTmpl, err := template.ParseFiles("controllers/basic-chat/basic-message/response/prompt/user/prompt.go.tmpl")
	if err != nil {
		log.Printf("Error parsing user template: %v", err)
		return RunResponse{}, fmt.Errorf("error parsing user template: %w", err)
	}
	var userBuf bytes.Buffer
	if err := userTmpl.Execute(&userBuf, infoBank); err != nil {
		log.Printf("Error executing user template: %v", err)
		return RunResponse{}, fmt.Errorf("error executing user template: %w", err)
	}

	request := &aipitypes.AIPIRequest{
		Model:         model,
		SystemMessage: systemBuf.String(),
		UserMessage:   userBuf.String(),
		IdUser:        infoBank.IdUser,
	}
	response, err := r.aipi.GetCompletion(ctx, *request)
	if err != nil {
		return RunResponse{}, err
	}

	return RunResponse{Content: response.Data}, nil
}
