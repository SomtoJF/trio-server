package reflectionchat

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/somtojf/trio-server/models"
	"gorm.io/gorm"
)

type Endpoint struct {
	db *gorm.DB
}

type CreateReflectionChatRequest struct {
	ChatName string `json:"chatName" binding:"required"`
}

func NewEndpoint(db *gorm.DB) *Endpoint {
	return &Endpoint{db: db}
}

func (e *Endpoint) GetReflectionChats(c *gin.Context) {
	currentUser, exists := c.Get("currentUser")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found in context"})
		return
	}
	user := currentUser.(models.User)

	var reflectionChats []models.ReflectionChat
	if err := e.db.Where("user_id = ?", user.IdUser).Find(&reflectionChats).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch reflection chats"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": reflectionChats})
}

type GetReflectionChatResponse struct {
	ID        string    `json:"id"`
	ChatName  string    `json:"chatName"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

func (e *Endpoint) GetReflectionChat(c *gin.Context) {
	chatID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid chat ID"})
		return
	}

	var reflectionChats models.ReflectionChat
	if err := e.db.Where("external_id = ?", chatID).First(&reflectionChats).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch reflection chats"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": GetReflectionChatResponse{
		ID:        reflectionChats.ExternalID.String(),
		ChatName:  reflectionChats.ChatName,
		CreatedAt: reflectionChats.CreatedAt,
		UpdatedAt: reflectionChats.UpdatedAt,
	}})
}

func (e *Endpoint) GetChatReflections(c *gin.Context) {
	chatID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid chat ID"})
		return
	}

	var reflectionChats models.ReflectionChat
	if err := e.db.Where("external_id = ?", chatID).First(&reflectionChats).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch reflection chats"})
		return
	}

	var reflections []models.Reflection
	if err := e.db.
		Preload("Messages").
		Preload("EvaluatorMessages").
		Where("id_reflection_chat = ?", reflectionChats.IdReflectionChat).
		Find(&reflections).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch reflections"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": reflections})
}

func (e *Endpoint) CreateReflectionChat(c *gin.Context) {
	currentUser, exists := c.Get("currentUser")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found in context"})
		return
	}
	user := currentUser.(models.User)

	var body CreateReflectionChatRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	newChat := models.ReflectionChat{
		ChatName: body.ChatName,
		UserID:   user.IdUser,
	}

	if err := e.db.Create(&newChat).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create reflection chat"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"data": newChat})
}

func (e *Endpoint) DeleteReflectionChat(c *gin.Context) {
	currentUser, exists := c.Get("currentUser")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found in context"})
		return
	}
	user := currentUser.(models.User)

	chatID := c.Param("id")

	var chat models.ReflectionChat
	if err := e.db.Where("id_reflection_chat = ? AND user_id = ?", chatID, user.IdUser).First(&chat).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Chat not found or unauthorized"})
		return
	}

	if err := e.db.Delete(&chat).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete chat"})
		return
	}

	c.JSON(http.StatusNoContent, gin.H{"message": "Chat deleted successfully"})
}
