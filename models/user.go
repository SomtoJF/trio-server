package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type User struct {
	IdUser          uint      `gorm:"primaryKey;column:id_user;autoIncrement" json:"-"`
	ExternalID      uuid.UUID `gorm:"unique;type:uuid;default:gen_random_uuid()" json:"id"`
	Username        string    `gorm:"unique;type:string" json:"userName"`
	FullName        string    `json:"fullName"`
	PasswordHash    string    `json:"-"`
	CreatedAt       time.Time
	UpdatedAt       time.Time
	BasicChats      []BasicChat      `gorm:"foreignKey:UserID"`
	ReflectionChats []ReflectionChat `gorm:"foreignKey:UserID"`
	DeletedAt       gorm.DeletedAt   `gorm:"index"`
}
