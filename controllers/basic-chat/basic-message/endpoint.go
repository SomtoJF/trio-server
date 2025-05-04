package basicmessage

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"math/rand"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/qdrant/go-client/qdrant"
	"github.com/sashabaranov/go-openai"
	"github.com/somtojf/trio-server/aipi"
	"github.com/somtojf/trio-server/aipi/aipitypes"
	"github.com/somtojf/trio-server/controllers/basic-chat/basic-message/response"
	"github.com/somtojf/trio-server/models"
	"github.com/somtojf/trio-server/types/qdranttypes"
	"gorm.io/gorm"
)

type Endpoint struct {
	db           *gorm.DB
	qdrantDB     *qdrant.Client
	aipi         *aipi.Provider
	streamMx     sync.RWMutex
	streamOutput *SendBasicMessageResponse
}

type ResponseStatus string

const (
	ResponseStatusTyping               ResponseStatus = "typing"
	ResponseStatusThinking             ResponseStatus = "thinking"
	ResponseStatusUnderstandingContext ResponseStatus = "understanding context"
)

type SendBasicMessageRequest struct {
	Message string `json:"message" binding:"required"`
}

type AgentResponse struct {
	AgentName string    `json:"agentName"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"createdAt"`
}

type Status struct {
	Status    string `json:"status"`
	AgentName string `json:"agentName"`
}

type SendBasicMessageResponse struct {
	AgentResponses []AgentResponse `json:"agentResponses"`
	Status         []Status        `json:"status"`
	Error          string          `json:"error"`
}

func NewEndpoint(db *gorm.DB, aipi *aipi.Provider, qdrantDB *qdrant.Client) *Endpoint {
	return &Endpoint{db, qdrantDB, aipi, sync.RWMutex{}, nil}
}

const EMBEDDING_MODEL = string(openai.SmallEmbedding3)
const RESPONSE_MODEL = string(openai.GPT4oMini)
const MAX_MESSAGE_LENGTH = 400
const HISTORYLIMIT = 10

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
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Chat not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var messages []models.BasicMessage
	if err := e.db.Where("id_basic_chat = ?", chat.IdBasicChat).Find(&messages).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": messages})
}

func (e *Endpoint) SendBasicMessage(c *gin.Context) {
	// Initialize stream output for this request
	e.streamMx.Lock()
	e.streamOutput = &SendBasicMessageResponse{
		AgentResponses: make([]AgentResponse, 0),
		Status:         make([]Status, 0),
	}
	e.streamMx.Unlock()

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("Transfer-Encoding", "chunked")

	done := make(chan struct{})
	defer func() {
		select {
		case done <- struct{}{}:
		default:
		}
		close(done)
	}()

	ctx, cancel := context.WithTimeout(c.Request.Context(), 300*time.Second)
	defer cancel()

	go func() {
		select {
		case <-ctx.Done():
			if ctx.Err() == context.DeadlineExceeded {
				e.streamError(c, "Request timeout exceeded")
			}
		case <-done:
			return
		}
	}()

	defer func() {
		c.SSEvent("done", "done")
		c.Writer.Flush()
	}()

	chatId, err := uuid.Parse(c.Param("id"))
	if err != nil {
		e.streamError(c, "Invalid chat id")
		return
	}

	currentUser, exists := c.Get("currentUser")
	if !exists {
		e.streamError(c, "User not authenticated")
		return
	}
	user := currentUser.(models.User)

	var request SendBasicMessageRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		e.streamError(c, err.Error())
		return
	}

	if len(request.Message) > MAX_MESSAGE_LENGTH {
		e.streamError(c, "Message too long")
		return
	}

	var chat models.BasicChat
	if err := e.db.Where("external_id = ? AND user_id = ?", chatId, user.IdUser).Preload("ChatAgents").First(&chat).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			e.streamError(c, "Chat not found")
			return
		}
		e.streamError(c, err.Error())
		return
	}

	if len(chat.ChatAgents) < 1 {
		e.streamError(c, "There are no agents to respond")
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

	e.streamStatus(c, Status{
		Status:    fmt.Sprintf("%s is trying to understand the context", agentInformation[0].AgentName),
		AgentName: agentInformation[0].AgentName,
	})

	var relevantContext []response.HistoryMessage
	// TODO: Uncomment this
	relevantContext, err = e.getRelevantContext(c.Request.Context(), request.Message, chat.IdBasicChat, HISTORYLIMIT)
	if err != nil {
		e.streamError(c, err.Error())
		return
	}

	agentInformation = shuffleArray(agentInformation)

	startTime := time.Now()

	userMessage := &models.BasicMessage{
		SenderName: user.Username,
		Content:    request.Message,
		ChatID:     chat.IdBasicChat,
	}

	if err := e.db.Create(userMessage).Error; err != nil {
		e.streamError(c, err.Error())
		return
	}

	for _, agent := range agentInformation {
		agentStartTime := time.Now()

		// Update agent status to thinking
		e.streamStatus(c, Status{
			Status:    fmt.Sprintf("%s is thinking", agent.AgentName),
			AgentName: agent.AgentName,
		})

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
		data, err := response.Run(c.Request.Context(), infoBank, RESPONSE_MODEL)
		if err != nil {
			e.streamError(c, fmt.Sprintf("Agent %s response error: %s", agent.AgentName, err.Error()))
			return
		}

		// Skip empty responses
		if data.Content == "" {
			slog.Info("Agent %s skipped response as message was directed to another agent", agent.AgentName)
			continue
		}

		newMessage := &models.BasicMessage{
			SenderName: agent.AgentName,
			ChatID:     chat.IdBasicChat,
			Content:    data.Content,
		}

		tx := e.db.Begin()
		if tx.Error != nil {
			e.streamError(c, tx.Error.Error())
			return
		}

		if err := tx.Create(newMessage).Error; err != nil {
			tx.Rollback()
			e.streamError(c, err.Error())
			return
		}

		if err := e.saveToQdrant(c, *newMessage); err != nil {
			tx.Rollback()
			e.streamError(c, err.Error())
			return
		}

		if err := tx.Commit().Error; err != nil {
			tx.Rollback()
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
	if err := e.db.Where("id_basic_chat = ?", chatId).
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

func (e *Endpoint) getRelevantContext(c context.Context, message string, chatId uint, limit int) ([]response.HistoryMessage, error) {
	limitUint64 := uint64(limit)
	embeddingRequest := aipitypes.EmbeddingRequest{
		Input:          message,
		Model:          EMBEDDING_MODEL,
		EncodingFormat: string(openai.EmbeddingEncodingFormatFloat),
		Dimensions:     int(qdranttypes.VECTOR_SIZE_BASIC_MESSAGE),
	}
	embedding, err := e.aipi.GetEmbedding(c, embeddingRequest)
	if err != nil {
		return nil, err
	}

	searchResult, err := e.qdrantDB.Query(c, &qdrant.QueryPoints{
		CollectionName: string(qdranttypes.COLLECTION_NAME_BASIC_MESSAGES),
		Query:          qdrant.NewQuery(embedding...),
		Limit:          &limitUint64,
		Filter: &qdrant.Filter{
			Must: []*qdrant.Condition{
				qdrant.NewMatch("chat_id", string(chatId)),
			},
		},
		WithPayload: qdrant.NewWithPayload(true),
	})
	if err != nil {
		return nil, err
	}

	var relevantContext []response.HistoryMessage
	for _, point := range searchResult {
		payload := point.GetPayload()
		content := payload["content"]
		senderName := payload["sender_name"]
		sentAt := payload["created_at"]

		timeValue, err := time.Parse(time.RFC1123, sentAt.String())
		if err != nil {
			return nil, err
		}

		historyMessage := response.HistoryMessage{
			SenderName: senderName.String(),
			Content:    content.String(),
			SentAt:     timeValue,
		}
		relevantContext = append(relevantContext, historyMessage)
	}

	return relevantContext, nil
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

func (e *Endpoint) saveToQdrant(c context.Context, message models.BasicMessage) error {
	embeddingRequest := aipitypes.EmbeddingRequest{
		Input:          message.Content,
		Model:          EMBEDDING_MODEL,
		EncodingFormat: string(openai.EmbeddingEncodingFormatFloat),
		Dimensions:     int(qdranttypes.VECTOR_SIZE_BASIC_MESSAGE),
	}
	embedding, err := e.aipi.GetEmbedding(c, embeddingRequest)
	if err != nil {
		return err
	}

	payload := map[string]any{
		"chat_id":     message.ChatID,
		"content":     message.Content,
		"sender_name": message.SenderName,
		"external_id": message.ExternalID.String(),
		"created_at":  message.CreatedAt.String(),
	}

	_, err = e.qdrantDB.Upsert(c, &qdrant.UpsertPoints{
		CollectionName: string(qdranttypes.COLLECTION_NAME_BASIC_MESSAGES),
		Points: []*qdrant.PointStruct{
			{
				Id:      qdrant.NewIDNum(uint64(message.IdBasicMessage)),
				Vectors: qdrant.NewVectors(embedding...),
				Payload: qdrant.NewValueMap(payload),
			},
		},
	})
	if err != nil {
		return err
	}

	return nil
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

	if e.streamOutput == nil {
		e.streamOutput = &SendBasicMessageResponse{}
	}
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
