package repository

import (
	"ai_support_tg_writer_bot/internal/models"

	"gorm.io/gorm"
)

type MessageRepository interface {
	Create(message *models.Message) error
	GetByID(id uint) (*models.Message, error)
	GetByTicketID(ticketID uint) ([]models.Message, error)
	GetLastMessageByTicketID(ticketID uint) (*models.Message, error)
	Update(message *models.Message) error
	Delete(id uint) error
}

type messageRepository struct {
	db *gorm.DB
}

func NewMessageRepository(db *gorm.DB) MessageRepository {
	return &messageRepository{db: db}
}

func (r *messageRepository) Create(message *models.Message) error {
	return r.db.Create(message).Error
}

func (r *messageRepository) GetByID(id uint) (*models.Message, error) {
	var message models.Message
	err := r.db.Preload("User").Preload("Ticket").Preload("Files").First(&message, id).Error
	if err != nil {
		return nil, err
	}
	return &message, nil
}

func (r *messageRepository) GetByTicketID(ticketID uint) ([]models.Message, error) {
	var messages []models.Message
	err := r.db.Preload("User").Preload("Files").
		Where("ticket_id = ?", ticketID).Order("created_at ASC").Find(&messages).Error
	return messages, err
}

func (r *messageRepository) GetLastMessageByTicketID(ticketID uint) (*models.Message, error) {
	var message models.Message
	err := r.db.Preload("User").Preload("Files").
		Where("ticket_id = ?", ticketID).Order("created_at DESC").First(&message).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &message, nil
}

func (r *messageRepository) Update(message *models.Message) error {
	return r.db.Save(message).Error
}

func (r *messageRepository) Delete(id uint) error {
	return r.db.Delete(&models.Message{}, id).Error
}
