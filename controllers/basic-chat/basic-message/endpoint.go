package basicmessage

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"math/rand"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/somtojf/trio-server/aipi"
	"github.com/somtojf/trio-server/controllers/basic-chat/basic-message/response"
	"github.com/somtojf/trio-server/models"
	"gorm.io/gorm"
)

type Endpoint struct {
	db           *gorm.DB
	aipi         *aipi.Provider
	streamMx     sync.RWMutex
	streamOutput *SendBasicMessageResponse
}

type ResponseStatus string

const (
	ResponseStatusTyping   ResponseStatus = "typing"
	ResponseStatusThinking ResponseStatus = "thinking"
)

type SendBasicMessageRequest struct {
	Message string `json:"message" binding:"required"`
}

type AgentResponse struct {
	AgentName string
	Content   string
	CreatedAt time.Time
}

type Status struct {
	Status    ResponseStatus
	AgentName string
}

type SendBasicMessageResponse struct {
	AgentResponses []AgentResponse `json:"agentResponses"`
	Status         []Status        `json:"status"`
	Error          string          `json:"error"`
}

func NewEndpoint(db *gorm.DB, aipi *aipi.Provider) *Endpoint {
	return &Endpoint{db, aipi, sync.RWMutex{}, nil}
}

func (e *Endpoint) GetBasicMessages(c *gin.Context) {
	currentUser, exists := c.Get("currentUser")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}
	user := currentUser.(models.User)

	chatId, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid chatId"})
		return
	}

	var chat models.BasicChat
	if err := e.db.Where("external_id = ? AND user_id = ?", chatId, user.IdUser).First(&chat).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var messages []models.BasicMessage
	if err := e.db.Where("chat_id = ?", chat.IdBasicChat).Find(&messages).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": messages})
}

func (e *Endpoint) SendBasicMessage(c *gin.Context) {
	const HISTORYLIMIT = 10

	chatId, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid chat id"})
		return
	}

	currentUser, exists := c.Get("currentUser")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}
	user := currentUser.(models.User)

	var request SendBasicMessageRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var chat models.BasicChat
	if err := e.db.Where("external_id = ? AND user_id = ?", chatId, user.IdUser).First(&chat).Preload("ChatAgents").Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var agentInformation []response.AgentInformation
	for _, agent := range chat.ChatAgents {
		info := response.AgentInformation{
			AgentName:   agent.AgentName,
			AgentTraits: agent.AgentTraits,
		}
		agentInformation = append(agentInformation, info)
	}

	relevantContext, err := e.getRelevantContext(chat.IdBasicChat, HISTORYLIMIT)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	agentInformation = shuffleArray(agentInformation)

	startTime := time.Now()

	for _, agent := range agentInformation {
		agentStartTime := time.Now()

		chatHistory, err := e.getChatHistory(chat.IdBasicChat, HISTORYLIMIT)
		if err != nil {
			e.streamError(c, err.Error())
			return
		}

		var otherAgents []response.AgentInformation
		for _, chatAgent := range agentInformation {
			if chatAgent.AgentName != agent.AgentName {
				otherAgents = append(otherAgents, chatAgent)
			}
		}

		infoBank := response.InfoBank{
			IdUser:           user.IdUser,
			NewMessage:       request.Message,
			AgentInformation: agent,
			OtherAgents:      otherAgents,
			ChatHistory:      chatHistory,
			RelevantContext:  relevantContext,
		}

		response := response.NewResponse(e.db, e.aipi)
		data, err := response.Run(c.Request.Context(), infoBank)
		if err != nil {
			e.streamError(c, fmt.Sprintf("Agent %s response error: %s", agent.AgentName, err.Error()))
			return
		}

		newMessage := &models.BasicMessage{
			SenderName: agent.AgentName,
			ChatID:     chat.IdBasicChat,
			Content:    data.Content,
		}

		if err := e.db.Create(newMessage).Error; err != nil {
			e.streamError(c, err.Error())
			return
		}

		agentResponse := AgentResponse{
			AgentName: agent.AgentName,
			Content:   data.Content,
			CreatedAt: newMessage.CreatedAt,
		}

		e.streamAgentResponses(c, agentResponse)
		agentElapsedTime := time.Since(agentStartTime)
		slog.Info("Agent %s responded in %vs", agent.AgentName, agentElapsedTime.Seconds())
	}

	elapsedTime := time.Since(startTime)
	slog.Info("Total time taken: %vs", elapsedTime.Seconds())
}

func (e *Endpoint) getChatHistory(chatId uint, limit int) ([]response.HistoryMessage, error) {
	var messages []models.BasicMessage
	if err := e.db.Where("chat_id = ?", chatId).
		Order("created_at DESC").
		Limit(limit).
		Find(&messages).Error; err != nil {
		return nil, err
	}

	var chatHistory []response.HistoryMessage
	for _, message := range messages {
		historyMessage := response.HistoryMessage{
			SenderName: message.SenderName,
			Content:    message.Content,
			SentAt:     message.CreatedAt,
		}
		chatHistory = append(chatHistory, historyMessage)
	}

	return chatHistory, nil
}

// TODO: Implement this using a vector database i.e qdrant
func (e *Endpoint) getRelevantContext(chatId uint, limit int) ([]response.HistoryMessage, error) {
	return nil, nil
}

func shuffleArray[T any](array []T) []T {
	shuffled := make([]T, len(array))
	copy(shuffled, array)

	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	// Fisher-Yates shuffle algorithm
	for i := len(shuffled) - 1; i > 0; i-- {
		j := r.Intn(i + 1)
		shuffled[i], shuffled[j] = shuffled[j], shuffled[i]
	}

	return shuffled
}

func (e *Endpoint) streamAgentResponses(c *gin.Context, response AgentResponse) {
	e.streamMx.Lock()
	defer e.streamMx.Unlock()

	found := false
	for i, existing := range e.streamOutput.AgentResponses {
		if existing.AgentName == response.AgentName {
			e.streamOutput.AgentResponses[i] = response
			found = true
			break
		}
	}
	if !found {
		e.streamOutput.AgentResponses = append(e.streamOutput.AgentResponses, response)
	}

	e.updateStream(c, *e.streamOutput)
}

func (e *Endpoint) streamStatus(c *gin.Context, status Status) {
	e.streamMx.Lock()
	defer e.streamMx.Unlock()

	found := false
	for i, existing := range e.streamOutput.Status {
		if existing.AgentName == status.AgentName {
			e.streamOutput.Status[i] = status
			found = true
			break
		}
	}
	if !found {
		e.streamOutput.Status = append(e.streamOutput.Status, status)
	}

	e.updateStream(c, *e.streamOutput)
}

func (e *Endpoint) streamError(c *gin.Context, error string) {
	e.streamMx.Lock()
	defer e.streamMx.Unlock()

	e.streamOutput.Error = error
	e.updateStream(c, *e.streamOutput)
}

func (e *Endpoint) updateStream(c *gin.Context, response SendBasicMessageResponse) {
	data, err := json.Marshal(response)
	if err == nil {
		c.SSEvent("message", string(data))
		c.Writer.Flush()
	}
}

/*
	Send Message Psuedocode ---

	1. Get the chat history
	2. Get the relevant context
	3. For each agent:
		1. Get the agent information
		2. Get the Chat History
		3. Send the message to the agent (response.Run())
		4. Get the response from the agent
		5. Stream the response
*/
