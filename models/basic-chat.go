package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type BasicChat struct {
	IdBasicChat uint           `gorm:"primaryKey;column:id_basic_chat;autoIncrement" json:"-"`
	ExternalID  uuid.UUID      `gorm:"unique;type:uuid;default:gen_random_uuid()" json:"id"`
	UserID      uint           `gorm:"column:user_id"`
	User        User           `gorm:"foreignKey:UserID"`
	Messages    []BasicMessage `gorm:"foreignKey:ChatID"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   gorm.DeletedAt `gorm:"index"`
}
