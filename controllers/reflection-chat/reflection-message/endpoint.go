package reflectionmessage

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/qdrant/go-client/qdrant"
	"github.com/sashabaranov/go-openai"
	"github.com/somtojf/trio-server/aipi"
	"github.com/somtojf/trio-server/aipi/aipitypes"
	"github.com/somtojf/trio-server/controllers/reflection-chat/reflection-message/response"
	"github.com/somtojf/trio-server/types/qdranttypes"

	"github.com/somtojf/trio-server/models"
	"gorm.io/gorm"
)

type Endpoint struct {
	db           *gorm.DB
	qdrantDB     *qdrant.Client
	aipi         *aipi.Provider
	streamOutput *SendReflectionMessageResponse
}

func NewEndpoint(db *gorm.DB, aipi *aipi.Provider, qdrantDB *qdrant.Client) *Endpoint {
	return &Endpoint{db: db, aipi: aipi, qdrantDB: qdrantDB}
}

type SendReflectionMessageResponse struct {
	Reflection *models.Reflection `json:"reflection"`
	Status     []string           `json:"status"`
	Error      string             `json:"error"`
}

// type MessageData struct {
// 	ID         uuid.UUID `json:"id"`
// 	Content    string    `json:"content"`
// 	SenderName string    `json:"senderName"`
// 	IsOptimal  bool      `json:"isOptimal"`
// 	SentAt     time.Time `json:"sentAt"`
// }

// type EvaluatorMessageData struct {
// 	ID        uuid.UUID `json:"id"`
// 	Content   string    `json:"content"`
// 	IsOptimal bool      `json:"isOptimal"`
// 	SentAt    time.Time `json:"sentAt"`
// }

// type ReflectionData struct {
// 	ID                uuid.UUID              `json:"id"`
// 	Messages          []MessageData          `json:"messages"`
// 	EvaluatorMessages []EvaluatorMessageData `json:"evaluatorMessages"`
// }

type SendMessageRequest struct {
	Message string `json:"message" binding:"required"`
}

const EMBEDDING_MODEL = string(openai.SmallEmbedding3)
const ANSWERER_MODEL = "gpt-4.1-nano-2025-04-14"
const EVALUATOR_MODEL = "gpt-4.1-nano-2025-04-14"
const MAX_MESSAGE_LENGTH = 400

func (e *Endpoint) SendMessage(c *gin.Context) {
	e.streamOutput = &SendReflectionMessageResponse{
		Status: make([]string, 0),
	}

	defer func() {
		if r := recover(); r != nil {
			log.Printf("Recovered from panic: %v", r)
			e.streamError(c, "An unexpected error occurred")
		}
	}()

	ctx, cancel := context.WithTimeout(c.Request.Context(), 300*time.Second)
	defer cancel()

	go func() {
		<-ctx.Done()
		if ctx.Err() == context.DeadlineExceeded {
			e.streamError(c, "Request timeout exceeded")
		}
	}()

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("Transfer-Encoding", "chunked")

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

	var request SendMessageRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		e.streamError(c, err.Error())
		return
	}

	var chat models.ReflectionChat
	if err := e.db.First(&chat, "external_id = ?", chatId).Error; err != nil {
		e.streamError(c, "Chat not found")
		return
	}

	e.streamStatus(c, "Reading chat history...")
	chatHistory, err := e.getChatHistory(chat.IdReflectionChat, 10, user)
	if err != nil {
		e.streamError(c, err.Error())
		return
	}

	time.Sleep(1 * time.Second)

	e.streamStatus(c, "Getting relevant context...")

	time.Sleep(1 * time.Second)
	// TODO: Uncomment this
	// relevantContext, err := e.getRelevantContext(ctx, request.Message, chat.ExternalID, 10)
	// if err != nil {
	// 	e.streamError(c, err.Error())
	// 	return
	// }
	relevantContext := []response.HistoryMessage{}

	optimalResponseGotten := false
	numberOfIterations := 0

	tx := e.db.Begin()
	reflection := models.Reflection{
		ChatID: chat.IdReflectionChat,
	}
	if err := tx.Create(&reflection).Error; err != nil {
		tx.Rollback()
		log.Printf("Failed to create reflection: %v", err)
		e.streamError(c, "An error occurred")
		return
	}

	userMessage := models.ReflectionMessage{
		ReflectionID: reflection.IdReflection,
		SenderName:   user.Username,
		Content:      request.Message,
	}
	if err := tx.Create(&userMessage).Error; err != nil {
		tx.Rollback()
		log.Printf("Failed to create user message: %v", err)
		e.streamError(c, "An error occured while sending your message")
		return
	}

	previousResponses := []response.PreviousResponse{}

	for !optimalResponseGotten {
		log.Printf("Iteration %d", numberOfIterations)

		reflectionMessage := models.ReflectionMessage{
			ReflectionID: reflection.IdReflection,
			SenderName:   "Reflector",
		}
		evaluatorMessage := models.EvaluatorMessage{
			ReflectionID: reflection.IdReflection,
		}

		/*
			1. Get the context
			2. Get the chat history
			3. Prompt the answerer
			4. Prompt the evaluator with the chat history and the answerer's response
			4. Evaluator should return if the response is optimal or not as well as an explanation
			5. If the response is optimal, break by assigning the optimality value to optimalResponseGotten
		*/

		responseGenerator := response.NewResponse(e.db, e.aipi)
		answererInfoBank := response.AnswererInfoBank{
			IdUser:            chat.UserID,
			ChatHistory:       chatHistory,
			Context:           relevantContext,
			Message:           request.Message,
			PreviousResponses: previousResponses,
		}

		if numberOfIterations > 0 {
			e.streamStatus(c, fmt.Sprintf("Improving on response %d", numberOfIterations))
		} else {
			e.streamStatus(c, fmt.Sprintf("Generating response %d", numberOfIterations+1))
		}

		answererResponse, err := responseGenerator.RunAnswerer(ctx, answererInfoBank, ANSWERER_MODEL)
		if err != nil {
			tx.Rollback()
			log.Printf("Failed to generate response: %v", err)
			e.streamError(c, "An error occured while sending your message")
			return
		}

		reflectionMessage.Content = answererResponse.Content
		reflectionMessage.Title = answererResponse.Title
		if err := tx.Create(&reflectionMessage).Error; err != nil {
			tx.Rollback()
			log.Printf("Failed to create reflection message: %v", err)
			e.streamError(c, "An error occured while sending your message")
			return
		}

		// Reload reflection to get the latest messages
		if err := e.refreshReflection(tx, &reflection); err != nil {
			tx.Rollback()
			log.Printf("Failed to reload reflection: %v", err)
			e.streamError(c, "An error occured while sending your message")
			return
		}

		e.streamReflection(c, &reflection)

		evaluatorInfoBank := response.EvaluatorInfoBank{
			IdUser:            chat.UserID,
			ChatHistory:       chatHistory,
			Context:           relevantContext,
			Message:           request.Message,
			IterationCount:    numberOfIterations + 1,
			AnswererResponse:  answererResponse,
			PreviousResponses: previousResponses,
		}

		e.streamStatus(c, fmt.Sprintf("Evaluating response %d", numberOfIterations+1))
		evaluatorResponse, err := responseGenerator.RunEvaluator(ctx, evaluatorInfoBank, EVALUATOR_MODEL)
		if err != nil {
			tx.Rollback()
			log.Printf("evaluator failed to evaluate response: %v", err)
			e.streamError(c, "An error occured while sending your message")
			return
		}

		// Update previous responses
		currentResponse := response.PreviousResponse{
			AnswererResponse:  answererResponse,
			EvaluatorResponse: evaluatorResponse,
		}
		previousResponses = append(previousResponses, currentResponse)

		if numberOfIterations >= 5 {
			optimalResponseGotten = true

			evaluatorMessage.Content = evaluatorResponse.Content
			evaluatorMessage.IsOptimal = true
			if err := tx.Create(&evaluatorMessage).Error; err != nil {
				tx.Rollback()
				log.Printf("Failed to create evaluator message: %v", err)
				e.streamError(c, "An error occured while sending your message")
				return
			}

			if err := tx.Model(&reflectionMessage).Update("is_optimal", true).Error; err != nil {
				tx.Rollback()
				log.Printf("Failed to update reflection message optimal status: %v", err)
				e.streamError(c, "An error occurred while sending your message")
				return
			}

			// Reload reflection again to get the latest messages including evaluator message
			if err := e.refreshReflection(tx, &reflection); err != nil {
				tx.Rollback()
				log.Printf("Failed to reload reflection: %v", err)
				e.streamError(c, "An error occured while sending your message")
				return
			}

			e.streamReflection(c, &reflection)
			continue
		} else {
			optimalResponseGotten = evaluatorResponse.IsOptimal

		}

		evaluatorMessage.Content = evaluatorResponse.Content
		evaluatorMessage.IsOptimal = evaluatorResponse.IsOptimal
		if err := tx.Create(&evaluatorMessage).Error; err != nil {
			tx.Rollback()
			log.Printf("Failed to create evaluator message: %v", err)
			e.streamError(c, "An error occured while sending your message")
			return
		}

		if evaluatorMessage.IsOptimal {
			if err := tx.Model(&reflectionMessage).Update("is_optimal", true).Error; err != nil {
				tx.Rollback()
				log.Printf("Failed to update reflection message optimal status: %v", err)
				e.streamError(c, "An error occurred while sending your message")
				return
			}
		}

		// Reload reflection again to get the latest messages including evaluator message
		if err := e.refreshReflection(tx, &reflection); err != nil {
			tx.Rollback()
			log.Printf("Failed to reload reflection: %v", err)
			e.streamError(c, "An error occured while sending your message")
			return
		}

		e.streamReflection(c, &reflection)

		numberOfIterations += 1
	}

	// Save reflection messages to Qdrant
	// TODO: Uncomment this
	// for _, message := range reflection.Messages {
	// 	if err := e.saveToQdrant(ctx, message, chat.ExternalID); err != nil {
	// 		tx.Rollback()
	// 		log.Printf("Failed to save message to qdrant: %v", err)
	// 		e.streamError(c, "An error occured while sending your message")
	// 		return
	// 	}
	// }

	if err := tx.Commit().Error; err != nil {
		tx.Rollback()
		log.Printf("Failed to commit transaction: %v", err)
		e.streamError(c, "An error occured while sending your message")
		return
	}
}

func (e *Endpoint) refreshReflection(tx *gorm.DB, reflection *models.Reflection) error {
	if err := tx.Preload("Messages").Preload("EvaluatorMessages").Where("id_reflection = ?", reflection.IdReflection).First(reflection).Error; err != nil {
		log.Printf("Failed to load reflection with associations: %v", err)
		return err
	}

	return nil
}

// Get the chat history for the reflection chat. Only get the optimal messages
func (e *Endpoint) getChatHistory(chatId uint, limit int, user models.User) ([]response.HistoryMessage, error) {
	var messages []models.ReflectionMessage
	if err := e.db.Where("id_reflection IN (SELECT id_reflection FROM reflections WHERE id_reflection_chat = ?) AND is_optimal = ?", chatId, true).
		Order("created_at DESC").
		Limit(limit).
		Find(&messages).Error; err != nil {
		return nil, err
	}

	var chatHistory []response.HistoryMessage
	for _, message := range messages {
		var senderName string
		if message.SenderName == user.Username {
			senderName = user.Username
		} else {
			senderName = string(response.HistoryMessageSenderNameAnswerer)
		}

		historyMessage := response.HistoryMessage{
			SenderName: senderName,
			Content:    message.Content,
			SentAt:     message.CreatedAt,
		}
		chatHistory = append(chatHistory, historyMessage)
	}

	return chatHistory, nil
}

func (e *Endpoint) getRelevantContext(c context.Context, message string, chatId uuid.UUID, limit int) ([]response.HistoryMessage, error) {
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
		CollectionName: string(qdranttypes.COLLECTION_NAME_REFLECTION_MESSAGES),
		Query:          qdrant.NewQuery(embedding...),
		Limit:          &limitUint64,
		Filter: &qdrant.Filter{
			Must: []*qdrant.Condition{
				qdrant.NewMatch("chat_id", chatId.String()),
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
		sentAt := payload["created_at"]

		timeValue, err := time.Parse(time.RFC1123, sentAt.String())
		if err != nil {
			return nil, err
		}

		historyMessage := response.HistoryMessage{

			Content: content.String(),
			SentAt:  timeValue,
		}
		relevantContext = append(relevantContext, historyMessage)
	}

	return relevantContext, nil
}

func (e *Endpoint) saveToQdrant(c context.Context, message models.ReflectionMessage, chatId uuid.UUID) error {
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
		"chat_id":     chatId.String(),
		"content":     message.Content,
		"external_id": message.ExternalID,
		"created_at":  message.CreatedAt.String(),
	}

	_, err = e.qdrantDB.Upsert(c, &qdrant.UpsertPoints{
		CollectionName: string(qdranttypes.COLLECTION_NAME_BASIC_MESSAGES),
		Points: []*qdrant.PointStruct{
			{
				Id:      qdrant.NewIDUUID(message.ExternalID.String()),
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

func (e *Endpoint) streamReflection(c *gin.Context, reflection *models.Reflection) {
	e.streamOutput.Reflection = reflection
	e.updateStream(c, *e.streamOutput)
}

func (e *Endpoint) streamStatus(c *gin.Context, status string) {
	e.streamOutput.Status = append(e.streamOutput.Status, status)
	e.updateStream(c, *e.streamOutput)
}

func (e *Endpoint) streamError(c *gin.Context, error string) {
	if e.streamOutput == nil {
		e.streamOutput = &SendReflectionMessageResponse{}
	}
	e.streamOutput.Error = error
	e.updateStream(c, *e.streamOutput)
}

func (e *Endpoint) updateStream(c *gin.Context, response SendReflectionMessageResponse) {
	data, err := json.Marshal(response)
	if err == nil {
		c.SSEvent("message", string(data))
		c.Writer.Flush()
	}
}
