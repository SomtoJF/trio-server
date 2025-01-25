package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ReflectionChat struct {
	IdReflectionChat uint           `gorm:"primaryKey;column:id_reflection_chat;autoIncrement" json:"-"`
	ExternalID       uuid.UUID      `gorm:"unique;type:uuid;default:gen_random_uuid()" json:"id"`
	ChatName         string         `gorm:"column:chat_name" json:"chatName"`
	UserID           uint           `gorm:"column:user_id" json:"userId"`
	User             User           `gorm:"foreignKey:UserID" json:"user"`
	Reflections      []Reflection   `gorm:"foreignKey:ChatID" json:"reflections"`
	CreatedAt        time.Time      `gorm:"column:created_at" json:"createdAt"`
	UpdatedAt        time.Time      `gorm:"column:updated_at" json:"updatedAt"`
	DeletedAt        gorm.DeletedAt `gorm:"index"`
}
