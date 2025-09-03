package repository

import (
	"ai_support_tg_writer_bot/internal/models"
	"time"

	"gorm.io/gorm"
)

type ChatRepository interface {
	Create(chat *models.Chat) error
	GetByID(id uint) (*models.Chat, error)
	GetByUserID(userID uint) (*models.Chat, error)
	GetActiveChats() ([]models.Chat, error)
	GetActiveChatsPaginated(limit, offset int) ([]models.Chat, error)
	GetArchivedChats() ([]models.Chat, error)
	GetArchivedChatsPaginated(limit, offset int) ([]models.Chat, error)
	Update(chat *models.Chat) error
	ArchiveChat(chatID uint) error
	MarkAsRead(chatID uint) error
	IncrementUnreadCount(chatID uint) error
	UpdateLastMessageTime(chatID uint) error
}

type chatRepository struct {
	db *gorm.DB
}

func NewChatRepository(db *gorm.DB) ChatRepository {
	return &chatRepository{db: db}
}

func (r *chatRepository) Create(chat *models.Chat) error {
	return r.db.Create(chat).Error
}

func (r *chatRepository) GetByID(id uint) (*models.Chat, error) {
	var chat models.Chat
	err := r.db.Preload("User").Preload("Messages").Preload("Messages.User").Preload("Messages.Files").
		First(&chat, id).Error
	if err != nil {
		return nil, err
	}
	return &chat, nil
}

func (r *chatRepository) GetByUserID(userID uint) (*models.Chat, error) {
	var chat models.Chat
	err := r.db.Preload("User").Preload("Messages").Preload("Messages.User").Preload("Messages.Files").
		Where("user_id = ? AND status = ?", userID, models.ChatStatusActive).First(&chat).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &chat, nil
}

func (r *chatRepository) GetActiveChats() ([]models.Chat, error) {
	var chats []models.Chat
	err := r.db.Preload("User").Preload("Messages").Preload("Messages.User").Preload("Messages.Files").
		Where("status = ?", models.ChatStatusActive).Order("last_message_at DESC NULLS LAST, created_at DESC").Find(&chats).Error
	return chats, err
}

func (r *chatRepository) GetActiveChatsPaginated(limit, offset int) ([]models.Chat, error) {
	var chats []models.Chat
	err := r.db.Preload("User").
		Where("status = ?", models.ChatStatusActive).Order("last_message_at DESC NULLS LAST, created_at DESC").
		Limit(limit).Offset(offset).Find(&chats).Error
	return chats, err
}

func (r *chatRepository) GetArchivedChats() ([]models.Chat, error) {
	var chats []models.Chat
	err := r.db.Preload("User").Preload("Messages").Preload("Messages.User").Preload("Messages.Files").
		Where("status = ?", models.ChatStatusArchived).Order("updated_at DESC").Find(&chats).Error
	return chats, err
}

func (r *chatRepository) GetArchivedChatsPaginated(limit, offset int) ([]models.Chat, error) {
	var chats []models.Chat
	err := r.db.Preload("User").
		Where("status = ?", models.ChatStatusArchived).Order("updated_at DESC").
		Limit(limit).Offset(offset).Find(&chats).Error
	return chats, err
}

func (r *chatRepository) Update(chat *models.Chat) error {
	return r.db.Save(chat).Error
}

func (r *chatRepository) ArchiveChat(chatID uint) error {
	return r.db.Model(&models.Chat{}).Where("id = ?", chatID).Updates(map[string]interface{}{
		"status": models.ChatStatusArchived,
	}).Error
}

func (r *chatRepository) MarkAsRead(chatID uint) error {
	// Обнуляем счетчик непрочитанных сообщений
	err := r.db.Model(&models.Chat{}).Where("id = ?", chatID).Update("unread_count", 0).Error
	if err != nil {
		return err
	}

	// Помечаем все сообщения от пользователя как прочитанные
	return r.db.Model(&models.ChatMessage{}).Where("chat_id = ? AND is_from_user = ?", chatID, true).Update("is_read", true).Error
}

func (r *chatRepository) IncrementUnreadCount(chatID uint) error {
	return r.db.Model(&models.Chat{}).Where("id = ?", chatID).UpdateColumn("unread_count", gorm.Expr("unread_count + 1")).Error
}

func (r *chatRepository) UpdateLastMessageTime(chatID uint) error {
	now := time.Now()
	return r.db.Model(&models.Chat{}).Where("id = ?", chatID).Update("last_message_at", &now).Error
}
