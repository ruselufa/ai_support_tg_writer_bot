package service

import (
	"ai_support_tg_writer_bot/internal/models"
	"ai_support_tg_writer_bot/internal/repository"
	"fmt"
	"strings"
)

type UserService interface {
	CreateOrGetUser(telegramID int64, username, firstName, lastName string) (*models.User, error)
	GetUserByTelegramID(telegramID int64) (*models.User, error)
	IsAdmin(telegramID int64) (bool, error)
	SetAdmin(telegramID int64, isAdmin bool) error
	GetAllAdmins() ([]models.User, error)
}

type userService struct {
	userRepo repository.UserRepository
}

func NewUserService(userRepo repository.UserRepository) UserService {
	return &userService{
		userRepo: userRepo,
	}
}

func (s *userService) CreateOrGetUser(telegramID int64, username, firstName, lastName string) (*models.User, error) {
	// Добавляем @ к username если его нет
	if username != "" && !strings.HasPrefix(username, "@") {
		username = "@" + username
	}

	// Сначала пытаемся найти существующего пользователя
	user, err := s.userRepo.GetByTelegramID(telegramID)
	if err == nil {
		// Пользователь найден, обновляем информацию
		user.Username = username
		user.FirstName = firstName
		user.LastName = lastName
		if err := s.userRepo.Update(user); err != nil {
			return nil, fmt.Errorf("failed to update user: %w", err)
		}
		return user, nil
	}

	// Пользователь не найден, создаем нового
	user = &models.User{
		TelegramID: telegramID,
		Username:   username,
		FirstName:  firstName,
		LastName:   lastName,
		IsAdmin:    false,
	}

	if err := s.userRepo.Create(user); err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	return user, nil
}

func (s *userService) GetUserByTelegramID(telegramID int64) (*models.User, error) {
	return s.userRepo.GetByTelegramID(telegramID)
}

func (s *userService) IsAdmin(telegramID int64) (bool, error) {
	return s.userRepo.IsAdmin(telegramID)
}

func (s *userService) SetAdmin(telegramID int64, isAdmin bool) error {
	user, err := s.userRepo.GetByTelegramID(telegramID)
	if err != nil {
		return fmt.Errorf("user not found: %w", err)
	}

	user.IsAdmin = isAdmin
	return s.userRepo.Update(user)
}

func (s *userService) GetAllAdmins() ([]models.User, error) {
	return s.userRepo.GetAllAdmins()
}
