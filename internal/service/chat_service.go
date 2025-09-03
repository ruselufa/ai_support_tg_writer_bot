package service

import (
	"ai_support_tg_writer_bot/internal/models"
	"ai_support_tg_writer_bot/internal/repository"
	"fmt"
)

type ChatService interface {
	CreateOrGetChat(userID uint) (*models.Chat, error)
	GetChatByID(id uint) (*models.Chat, error)
	GetActiveChats() ([]models.Chat, error)
	GetActiveChatsPaginated(limit, offset int) ([]models.Chat, error)
	GetArchivedChats() ([]models.Chat, error)
	GetArchivedChatsPaginated(limit, offset int) ([]models.Chat, error)
	ArchiveChat(chatID uint) error
	MarkChatAsRead(chatID uint) error
	AddMessage(chatID uint, userID uint, content string, isFromUser bool) (*models.ChatMessage, error)
	GetUnreadChatsCount() (int, error)
	GetChatMessagesPaginated(chatID uint, limit, offset int) ([]models.ChatMessage, error)
	GetChatMessagesCount(chatID uint) (int64, error)
}

type chatService struct {
	chatRepo        repository.ChatRepository
	chatMessageRepo repository.ChatMessageRepository
	fileRepo        repository.FileRepository
}

func NewChatService(chatRepo repository.ChatRepository, chatMessageRepo repository.ChatMessageRepository, fileRepo repository.FileRepository) ChatService {
	return &chatService{
		chatRepo:        chatRepo,
		chatMessageRepo: chatMessageRepo,
		fileRepo:        fileRepo,
	}
}

func (s *chatService) CreateOrGetChat(userID uint) (*models.Chat, error) {
	// Сначала пытаемся найти активный чат пользователя
	chat, err := s.chatRepo.GetByUserID(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get chat: %w", err)
	}

	if chat != nil {
		return chat, nil
	}

	// Создаем новый чат
	chat = &models.Chat{
		UserID:      userID,
		Status:      models.ChatStatusActive,
		UnreadCount: 0,
	}

	if err := s.chatRepo.Create(chat); err != nil {
		return nil, fmt.Errorf("failed to create chat: %w", err)
	}

	return chat, nil
}

func (s *chatService) GetChatByID(id uint) (*models.Chat, error) {
	return s.chatRepo.GetByID(id)
}

func (s *chatService) GetActiveChats() ([]models.Chat, error) {
	return s.chatRepo.GetActiveChats()
}

func (s *chatService) GetArchivedChats() ([]models.Chat, error) {
	return s.chatRepo.GetArchivedChats()
}

func (s *chatService) ArchiveChat(chatID uint) error {
	return s.chatRepo.ArchiveChat(chatID)
}

func (s *chatService) MarkChatAsRead(chatID uint) error {
	return s.chatRepo.MarkAsRead(chatID)
}

func (s *chatService) AddMessage(chatID uint, userID uint, content string, isFromUser bool) (*models.ChatMessage, error) {
	message := &models.ChatMessage{
		ChatID:     chatID,
		UserID:     userID,
		Content:    content,
		IsFromUser: isFromUser,
		IsRead:     !isFromUser, // Сообщения от админа считаются прочитанными сразу
	}

	if err := s.chatMessageRepo.Create(message); err != nil {
		return nil, fmt.Errorf("failed to create message: %w", err)
	}

	// Обновляем время последнего сообщения
	if err := s.chatRepo.UpdateLastMessageTime(chatID); err != nil {
		return nil, fmt.Errorf("failed to update last message time: %w", err)
	}

	// Если сообщение от пользователя, увеличиваем счетчик непрочитанных
	if isFromUser {
		if err := s.chatRepo.IncrementUnreadCount(chatID); err != nil {
			return nil, fmt.Errorf("failed to increment unread count: %w", err)
		}
	}

	return message, nil
}

func (s *chatService) GetUnreadChatsCount() (int, error) {
	chats, err := s.chatRepo.GetActiveChats()
	if err != nil {
		return 0, err
	}

	count := 0
	for _, chat := range chats {
		if chat.UnreadCount > 0 {
			count++
		}
	}

	return count, nil
}

func (s *chatService) GetActiveChatsPaginated(limit, offset int) ([]models.Chat, error) {
	return s.chatRepo.GetActiveChatsPaginated(limit, offset)
}

func (s *chatService) GetArchivedChatsPaginated(limit, offset int) ([]models.Chat, error) {
	return s.chatRepo.GetArchivedChatsPaginated(limit, offset)
}

func (s *chatService) GetChatMessagesPaginated(chatID uint, limit, offset int) ([]models.ChatMessage, error) {
	return s.chatMessageRepo.GetByChatIDPaginated(chatID, limit, offset)
}

func (s *chatService) GetChatMessagesCount(chatID uint) (int64, error) {
	return s.chatMessageRepo.GetCountByChatID(chatID)
}
