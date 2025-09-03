package models

import (
	"time"

	"gorm.io/gorm"
)

type TicketStatus string

const (
	TicketStatusOpen     TicketStatus = "open"
	TicketStatusAnswered TicketStatus = "answered"
	TicketStatusClosed   TicketStatus = "closed"
)

type Ticket struct {
	ID        uint           `json:"id" gorm:"primaryKey"`
	UserID    uint           `json:"user_id" gorm:"not null"`
	User      User           `json:"user" gorm:"foreignKey:UserID"`
	Status    TicketStatus   `json:"status" gorm:"default:'open'"`
	Subject   string         `json:"subject"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	ClosedAt  *time.Time     `json:"closed_at"`
	DeletedAt gorm.DeletedAt `json:"deleted_at" gorm:"index"`
	Messages  []Message      `json:"messages" gorm:"foreignKey:TicketID"`
}

type Message struct {
	ID         uint           `json:"id" gorm:"primaryKey"`
	TicketID   uint           `json:"ticket_id" gorm:"not null"`
	Ticket     Ticket         `json:"ticket" gorm:"foreignKey:TicketID"`
	UserID     uint           `json:"user_id" gorm:"not null"`
	User       User           `json:"user" gorm:"foreignKey:UserID"`
	Content    string         `json:"content"`
	IsFromUser bool           `json:"is_from_user" gorm:"not null"`
	CreatedAt  time.Time      `json:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at"`
	DeletedAt  gorm.DeletedAt `json:"deleted_at" gorm:"index"`
	Files      []File         `json:"files" gorm:"foreignKey:MessageID"`
}

type File struct {
	ID        uint           `json:"id" gorm:"primaryKey"`
	MessageID uint           `json:"message_id" gorm:"not null"`
	Message   ChatMessage    `json:"message" gorm:"foreignKey:MessageID;references:ID"`
	FileID    string         `json:"file_id" gorm:"not null"`
	FileName  string         `json:"file_name"`
	FileType  string         `json:"file_type"`
	FileSize  int64          `json:"file_size"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"deleted_at" gorm:"index"`
}
