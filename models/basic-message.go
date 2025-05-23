package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type BasicMessage struct {
	IdBasicMessage uint      `gorm:"primaryKey;column:id_basic_message;autoIncrement" json:"-"`
	ExternalID     uuid.UUID `gorm:"unique;type:uuid;default:gen_random_uuid()" json:"id"`
	SenderName     string    `gorm:"column:sender_name" json:"senderName"`
	ChatID         uint      `gorm:"column:id_basic_chat" json:"chatId"`
	Content        string    `gorm:"column:content" json:"content"`
	CreatedAt      time.Time
	UpdatedAt      time.Time
	DeletedAt      gorm.DeletedAt `gorm:"index"`
}
