package repository

import (
	"ai_support_tg_writer_bot/internal/models"

	"gorm.io/gorm"
)

type UserRepository interface {
	Create(user *models.User) error
	GetByTelegramID(telegramID int64) (*models.User, error)
	GetByID(id uint) (*models.User, error)
	Update(user *models.User) error
	IsAdmin(telegramID int64) (bool, error)
	GetAllAdmins() ([]models.User, error)
}

type userRepository struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) UserRepository {
	return &userRepository{db: db}
}

func (r *userRepository) Create(user *models.User) error {
	return r.db.Create(user).Error
}

func (r *userRepository) GetByTelegramID(telegramID int64) (*models.User, error) {
	var user models.User
	err := r.db.Where("telegram_id = ?", telegramID).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *userRepository) GetByID(id uint) (*models.User, error) {
	var user models.User
	err := r.db.First(&user, id).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *userRepository) Update(user *models.User) error {
	return r.db.Save(user).Error
}

func (r *userRepository) IsAdmin(telegramID int64) (bool, error) {
	var user models.User
	err := r.db.Where("telegram_id = ? AND is_admin = ?", telegramID, true).First(&user).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (r *userRepository) GetAllAdmins() ([]models.User, error) {
	var admins []models.User
	err := r.db.Where("is_admin = ?", true).Find(&admins).Error
	return admins, err
}
