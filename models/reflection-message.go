package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ReflectionMessage struct {
	IdReflectionMessage uint      `gorm:"primaryKey;column:id_reflection_message;autoIncrement" json:"-"`
	ExternalID          uuid.UUID `gorm:"unique;type:uuid;default:gen_random_uuid()" json:"id"`
	SenderName          string    `gorm:"column:sender_name"`
	IsOptimal           bool      `gorm:"column:is_optimal"`
	Content             string    `gorm:"column:content"`
	ReflectionID        uint      `gorm:"column:id_reflection"`
	CreatedAt           time.Time
	UpdatedAt           time.Time
	DeletedAt           gorm.DeletedAt `gorm:"index"`
}
