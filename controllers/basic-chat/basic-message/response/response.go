package response

import (
	"context"
	"time"

	"github.com/somtojf/trio-server/aipi"
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

func (r *Response) Run(ctx context.Context, infoBank InfoBank) (RunResponse, error) {

	return RunResponse{}, nil
}
