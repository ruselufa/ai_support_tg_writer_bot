package repository

import (
	"ai_support_tg_writer_bot/internal/models"

	"gorm.io/gorm"
)

type FileRepository interface {
	Create(file *models.File) error
	GetByID(id uint) (*models.File, error)
	GetByMessageID(messageID uint) ([]models.File, error)
	Update(file *models.File) error
	Delete(id uint) error
}

type fileRepository struct {
	db *gorm.DB
}

func NewFileRepository(db *gorm.DB) FileRepository {
	return &fileRepository{db: db}
}

func (r *fileRepository) Create(file *models.File) error {
	return r.db.Create(file).Error
}

func (r *fileRepository) GetByID(id uint) (*models.File, error) {
	var file models.File
	err := r.db.Preload("Message").First(&file, id).Error
	if err != nil {
		return nil, err
	}
	return &file, nil
}

func (r *fileRepository) GetByMessageID(messageID uint) ([]models.File, error) {
	var files []models.File
	err := r.db.Where("message_id = ?", messageID).Find(&files).Error
	return files, err
}

func (r *fileRepository) Update(file *models.File) error {
	return r.db.Save(file).Error
}

func (r *fileRepository) Delete(id uint) error {
	return r.db.Delete(&models.File{}, id).Error
}
