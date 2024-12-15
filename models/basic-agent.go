package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"gorm.io/gorm"
)

type BasicAgent struct {
	IdBasicAgent uint           `gorm:"primaryKey;column:id_basic_agent;autoIncrement" json:"-"`
	ExternalID   uuid.UUID      `gorm:"unique;type:uuid;default:gen_random_uuid()" json:"id"`
	AgentName    string         `gorm:"column:agent_name;uniqueIndex:idx_agent_name_chat"`
	AgentTraits  pq.StringArray `gorm:"type:text[]"`
	ChatID       uint           `gorm:"column:id_basic_chat;uniqueIndex:idx_agent_name_chat"`
	CreatedAt    time.Time
	UpdatedAt    time.Time
	DeletedAt    gorm.DeletedAt `gorm:"index"`
}
