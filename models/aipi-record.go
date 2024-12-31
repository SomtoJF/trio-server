package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type AIPIRecord struct {
	IdAIPIRecord     uint      `gorm:"primaryKey;column:id_aipi_record;autoIncrement" json:"-"`
	ExternalID       uuid.UUID `gorm:"unique;type:uuid;default:gen_random_uuid()" json:"id"`
	ModelName        string    `gorm:"column:model_name"`
	InputTokenCount  int       `json:"inputTokenCount"`
	InputCost        float64   `json:"inputCost"`
	OutputCost       float64   `json:"outputCost"`
	TotalCost        float64   `json:"totalCost"`
	OutputTokenCount int       `json:"outputTokenCount"`
	Streamed         bool      `gorm:"type:bool;default:false" json:"streamed"`
	User             User      `gorm:"foreignKey:UserID"`
	UserID           uint      `gorm:"column:id_user"`
	CreatedAt        time.Time
	UpdatedAt        time.Time
	DeletedAt        gorm.DeletedAt `gorm:"index"`
}
