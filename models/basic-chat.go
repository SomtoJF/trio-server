package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type BasicChat struct {
	IdBasicChat uint           `gorm:"primaryKey;column:id_basic_chat;autoIncrement" json:"-"`
	ExternalID  uuid.UUID      `gorm:"unique;type:uuid;default:gen_random_uuid()" json:"id"`
	ChatName    string         `gorm:"column:chat_name" json:"chatName"`
	ChatAgents  []BasicAgent   `gorm:"foreignKey:ChatID" json:"chatAgents"`
	UserID      uint           `gorm:"column:user_id" json:"userId"`
	User        User           `gorm:"foreignKey:UserID" json:"user"`
	Messages    []BasicMessage `gorm:"foreignKey:ChatID" json:"messages"`
	CreatedAt   time.Time      `gorm:"column:created_at" json:"createdAt"`
	UpdatedAt   time.Time      `gorm:"column:updated_at" json:"updatedAt"`
	DeletedAt   gorm.DeletedAt `gorm:"index"`
}
