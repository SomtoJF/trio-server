package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ReflectionMessage struct {
	IdReflectionMessage uint           `gorm:"primaryKey;column:id_reflection_message;autoIncrement" json:"-"`
	ExternalID          uuid.UUID      `gorm:"unique;type:uuid;default:gen_random_uuid()" json:"id"`
	SenderName          string         `gorm:"column:sender_name" json:"senderName"`
	IsOptimal           bool           `gorm:"column:is_optimal" default:"false" json:"isOptimal"`
	Title               string         `gorm:"column:title" json:"title"`
	Content             string         `gorm:"column:content" json:"content"`
	ReflectionID        uint           `gorm:"column:id_reflection" json:"reflectionId"`
	CreatedAt           time.Time      `json:"createdAt"`
	UpdatedAt           time.Time      `json:"updatedAt"`
	DeletedAt           gorm.DeletedAt `gorm:"index" json:"-"`
}
