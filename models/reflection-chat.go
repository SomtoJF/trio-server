package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ReflectionChat struct {
	IdReflectionChat uint         `gorm:"primaryKey;column:id_reflection_chat;autoIncrement" json:"-"`
	ExternalID       uuid.UUID    `gorm:"unique;type:uuid;default:gen_random_uuid()" json:"id"`
	UserID           uint         `gorm:"column:user_id"`
	User             User         `gorm:"foreignKey:UserID"`
	Reflections      []Reflection `gorm:"foreignKey:ChatID"`
	CreatedAt        time.Time
	UpdatedAt        time.Time
	DeletedAt        gorm.DeletedAt `gorm:"index"`
}
