package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type User struct {
	IdUser          uint             `gorm:"primaryKey;column:id_user;autoIncrement" json:"-"`
	ExternalID      uuid.UUID        `gorm:"unique;type:uuid;default:gen_random_uuid()" json:"id"`
	Username        string           `gorm:"unique;type:string" json:"userName"`
	FullName        string           `json:"fullName"`
	PasswordHash    string           `json:"-"`
	IsGuest         bool             `gorm:"default:false" json:"isGuest"`
	CreatedAt       time.Time        `json:"createdAt"`
	UpdatedAt       time.Time        `json:"updatedAt"`
	BasicChats      []BasicChat      `gorm:"foreignKey:UserID"`
	ReflectionChats []ReflectionChat `gorm:"foreignKey:UserID"`
	DeletedAt       gorm.DeletedAt   `gorm:"index"`
}
