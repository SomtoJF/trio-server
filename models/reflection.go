package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Reflection struct {
	IdReflection uint                `gorm:"primaryKey;column:id_reflection;autoIncrement" json:"-"`
	ExternalID   uuid.UUID           `gorm:"unique;type:uuid;default:gen_random_uuid()" json:"id"`
	Messages     []ReflectionMessage `gorm:"foreignKey:ReflectionID"`
	ChatID       uint                `gorm:"column:id_reflection_chat"`
	CreatedAt    time.Time
	UpdatedAt    time.Time
	DeletedAt    gorm.DeletedAt `gorm:"index"`
}
