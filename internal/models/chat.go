package models

import (
	"time"

	"gorm.io/gorm"
)

type ChatStatus string

const (
	ChatStatusActive   ChatStatus = "active"   // Активный чат
	ChatStatusArchived ChatStatus = "archived" // Архивированный чат
)

type Chat struct {
	ID            uint           `json:"id" gorm:"primaryKey"`
	UserID        uint           `json:"user_id" gorm:"not null"`
	User          User           `json:"user" gorm:"foreignKey:UserID"`
	Status        ChatStatus     `json:"status" gorm:"default:'active'"`
	LastMessageAt *time.Time     `json:"last_message_at"`
	UnreadCount   int            `json:"unread_count" gorm:"default:0"` // Количество непрочитанных сообщений от клиента
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
	DeletedAt     gorm.DeletedAt `json:"deleted_at" gorm:"index"`
	Messages      []ChatMessage  `json:"messages" gorm:"foreignKey:ChatID"`
}

type ChatMessage struct {
	ID         uint           `json:"id" gorm:"primaryKey"`
	ChatID     uint           `json:"chat_id" gorm:"not null"`
	Chat       Chat           `json:"chat" gorm:"foreignKey:ChatID"`
	UserID     uint           `json:"user_id" gorm:"not null"`
	User       User           `json:"user" gorm:"foreignKey:UserID"`
	Content    string         `json:"content"`
	IsFromUser bool           `json:"is_from_user" gorm:"not null"` // true = от клиента, false = от админа
	IsRead     bool           `json:"is_read" gorm:"default:false"` // Прочитано ли сообщение админом
	CreatedAt  time.Time      `json:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at"`
	DeletedAt  gorm.DeletedAt `json:"deleted_at" gorm:"index"`
	Files      []File         `json:"files" gorm:"foreignKey:MessageID"`
}

