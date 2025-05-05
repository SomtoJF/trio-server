package basicchat

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/somtojf/trio-server/models"
	"gorm.io/gorm"
)

type Endpoint struct {
	db *gorm.DB
}

type CreateAgentRequest struct {
	AgentName   string   `json:"agentName" binding:"required,max=50"`
	AgentTraits []string `json:"agentTraits" binding:"required"`
}

type CreateBasicChatRequest struct {
	ChatName string               `json:"chatName" binding:"required,max=100"`
	Agents   []CreateAgentRequest `json:"agents"`
}

type UpdateBasicChatRequest struct {
	ChatName string               `json:"chatName" binding:"required,max=100"`
	Agents   []CreateAgentRequest `json:"agents"`
}

func NewEndpoint(db *gorm.DB) *Endpoint {
	return &Endpoint{db: db}
}

func (e *Endpoint) CreateBasicChat(c *gin.Context) {
	currentUser, exists := c.Get("currentUser")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}
	user := currentUser.(models.User)

	var body CreateBasicChatRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if len(body.Agents) > 2 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Maximum of 2 agents allowed per chat"})
		return
	}

	chat := models.BasicChat{
		ChatName: body.ChatName,
		UserID:   user.IdUser,
	}

	tx := e.db.Begin()
	if tx.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start transaction"})
		return
	}

	if err := tx.Create(&chat).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create chat"})
		return
	}

	if len(body.Agents) > 0 {
		for _, agentReq := range body.Agents {
			agent := models.BasicAgent{
				AgentName:   agentReq.AgentName,
				AgentTraits: agentReq.AgentTraits,
				ChatID:      chat.IdBasicChat,
			}
			if err := tx.Create(&agent).Error; err != nil {
				tx.Rollback()
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create agent"})
				return
			}
		}
	}

	if err := tx.Commit().Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit transaction"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"data": chat})
}

func (e *Endpoint) GetBasicChats(c *gin.Context) {

	currentUser, exists := c.Get("currentUser")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}
	user := currentUser.(models.User)

	var chats []models.BasicChat

	if err := e.db.Where("user_id = ?", user.IdUser).Find(&chats).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch chats"})
		return
	}

	fmt.Println(chats)

	c.JSON(http.StatusOK, gin.H{"data": chats})
}

func (e *Endpoint) UpdateBasicChat(c *gin.Context) {
	currentUser, exists := c.Get("currentUser")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}
	user := currentUser.(models.User)

	chatID := c.Param("id")

	var body UpdateBasicChatRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if len(body.Agents) > 2 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Maximum of 2 agents allowed per chat"})
		return
	}

	var existingChat models.BasicChat
	if err := e.db.Where("id_basic_chat = ? AND user_id = ?", chatID, user.IdUser).First(&existingChat).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Chat not found or unauthorized"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch chat"})
		return
	}

	tx := e.db.Begin()
	if tx.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start transaction"})
		return
	}

	existingChat.ChatName = body.ChatName
	if err := tx.Save(&existingChat).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update chat"})
		return
	}

	if err := tx.Where("chat_id = ?", chatID).Delete(&models.BasicAgent{}).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update agents"})
		return
	}

	for _, agentReq := range body.Agents {
		agent := models.BasicAgent{
			AgentName:   agentReq.AgentName,
			AgentTraits: agentReq.AgentTraits,
			ChatID:      existingChat.IdBasicChat,
		}
		if err := tx.Create(&agent).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create agent"})
			return
		}
	}

	if err := tx.Commit().Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit transaction"})
		return
	}

	var updatedChat models.BasicChat
	if err := e.db.Preload("Agents").First(&updatedChat, chatID).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch updated chat"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": updatedChat})
}

func (e *Endpoint) DeleteBasicChat(c *gin.Context) {
	currentUser, exists := c.Get("currentUser")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}
	user := currentUser.(models.User)

	chatID := c.Param("id")

	var chat models.BasicChat
	if err := e.db.Where("id_basic_chat = ? AND user_id = ?", chatID, user.IdUser).First(&chat).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Chat not found or unauthorized"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch chat"})
		return
	}

	tx := e.db.Begin()
	if tx.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start transaction"})
		return
	}

	// Delete associated agents first (due to foreign key constraint)
	if err := tx.Where("chat_id = ?", chatID).Delete(&models.BasicAgent{}).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete associated agents"})
		return
	}

	if err := tx.Delete(&chat).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete chat"})
		return
	}

	if err := tx.Commit().Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit transaction"})
		return
	}

	c.JSON(http.StatusNoContent, gin.H{"message": "Chat deleted successfully"})
}

func (e *Endpoint) GetBasicChat(c *gin.Context) {
	chatID := c.Param("id")

	var chat models.BasicChat
	if err := e.db.Where("external_id = ?", chatID).Preload("ChatAgents").First(&chat).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch chat"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": chat})
}
