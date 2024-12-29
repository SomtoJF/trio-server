package basicmessage

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/somtojf/trio-server/aipi"
	"github.com/somtojf/trio-server/models"
	"gorm.io/gorm"
)

type Endpoint struct {
	db   *gorm.DB
	aipi *aipi.Provider
}

func NewEndpoint(db *gorm.DB, aipi *aipi.Provider) *Endpoint {
	return &Endpoint{db, aipi}
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
