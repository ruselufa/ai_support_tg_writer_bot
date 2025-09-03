package repository

import (
	"ai_support_tg_writer_bot/internal/models"

	"gorm.io/gorm"
)

type ChatMessageRepository interface {
	Create(message *models.ChatMessage) error
	GetByID(id uint) (*models.ChatMessage, error)
	GetByChatID(chatID uint) ([]models.ChatMessage, error)
	GetByChatIDPaginated(chatID uint, limit, offset int) ([]models.ChatMessage, error)
	GetCountByChatID(chatID uint) (int64, error)
	GetLastMessageByChatID(chatID uint) (*models.ChatMessage, error)
	Update(message *models.ChatMessage) error
	Delete(id uint) error
}

type chatMessageRepository struct {
	db *gorm.DB
}

func NewChatMessageRepository(db *gorm.DB) ChatMessageRepository {
	return &chatMessageRepository{db: db}
}

func (r *chatMessageRepository) Create(message *models.ChatMessage) error {
	return r.db.Create(message).Error
}

func (r *chatMessageRepository) GetByID(id uint) (*models.ChatMessage, error) {
	var message models.ChatMessage
	err := r.db.Preload("User").Preload("Chat").Preload("Files").First(&message, id).Error
	if err != nil {
		return nil, err
	}
	return &message, nil
}

func (r *chatMessageRepository) GetByChatID(chatID uint) ([]models.ChatMessage, error) {
	var messages []models.ChatMessage
	err := r.db.Preload("User").Preload("Files").
		Where("chat_id = ?", chatID).Order("created_at ASC").Find(&messages).Error
	return messages, err
}

func (r *chatMessageRepository) GetByChatIDPaginated(chatID uint, limit, offset int) ([]models.ChatMessage, error) {
	var messages []models.ChatMessage
	err := r.db.Preload("User").Preload("Files").
		Where("chat_id = ?", chatID).Order("created_at ASC").
		Limit(limit).Offset(offset).Find(&messages).Error
	return messages, err
}

func (r *chatMessageRepository) GetCountByChatID(chatID uint) (int64, error) {
	var count int64
	err := r.db.Model(&models.ChatMessage{}).Where("chat_id = ?", chatID).Count(&count).Error
	return count, err
}

func (r *chatMessageRepository) GetLastMessageByChatID(chatID uint) (*models.ChatMessage, error) {
	var message models.ChatMessage
	err := r.db.Preload("User").Preload("Files").
		Where("chat_id = ?", chatID).Order("created_at DESC").First(&message).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &message, nil
}

func (r *chatMessageRepository) Update(message *models.ChatMessage) error {
	return r.db.Save(message).Error
}

func (r *chatMessageRepository) Delete(id uint) error {
	return r.db.Delete(&models.ChatMessage{}, id).Error
}
