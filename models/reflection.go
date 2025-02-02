package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Reflection struct {
	IdReflection      uint                `gorm:"primaryKey;column:id_reflection;autoIncrement" json:"-"`
	ExternalID        uuid.UUID           `gorm:"unique;type:uuid;default:gen_random_uuid()" json:"id"`
	Messages          []ReflectionMessage `gorm:"foreignKey:ReflectionID" json:"messages"`
	EvaluatorMessages []EvaluatorMessage  `gorm:"foreignKey:ReflectionID" json:"evaluatorMessages"`
	ChatID            uint                `gorm:"column:id_reflection_chat" json:"chatId"`
	CreatedAt         time.Time           `gorm:"column:created_at" json:"createdAt"`
	UpdatedAt         time.Time           `gorm:"column:updated_at" json:"updatedAt"`
	DeletedAt         gorm.DeletedAt      `gorm:"index" json:"-"`
}

type EvaluatorMessage struct {
	IdEvaluatorMessage uint           `gorm:"primaryKey;column:id_evaluator_message;autoIncrement" json:"-"`
	ExternalID         uuid.UUID      `gorm:"unique;type:uuid;default:gen_random_uuid()" json:"id"`
	Content            string         `gorm:"column:content" json:"content"`
	IsOptimal          bool           `gorm:"column:is_optimal" default:"false" json:"isOptimal"`
	ReflectionID       uint           `gorm:"column:id_reflection"`
	CreatedAt          time.Time      `json:"createdAt"`
	UpdatedAt          time.Time      `json:"updatedAt"`
	DeletedAt          gorm.DeletedAt `gorm:"index" json:"-"`
}
