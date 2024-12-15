package reflectionchat

import (
	"net/http"

	"github.com/gin-gonic/gin"
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

	c.JSON(http.StatusOK, reflectionChats)
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

	c.JSON(http.StatusCreated, newChat)
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
