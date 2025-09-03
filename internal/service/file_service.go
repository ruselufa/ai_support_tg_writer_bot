package service

import (
	"ai_support_tg_writer_bot/internal/models"
	"ai_support_tg_writer_bot/internal/repository"
	"fmt"
)

type FileService interface {
	CreateFile(messageID uint, fileID, fileName, fileType string, fileSize int64) (*models.File, error)
	GetFileByID(id uint) (*models.File, error)
	GetFilesByMessageID(messageID uint) ([]models.File, error)
	DeleteFile(id uint) error
}

type fileService struct {
	fileRepo repository.FileRepository
}

func NewFileService(fileRepo repository.FileRepository) FileService {
	return &fileService{
		fileRepo: fileRepo,
	}
}

func (s *fileService) CreateFile(messageID uint, fileID, fileName, fileType string, fileSize int64) (*models.File, error) {
	file := &models.File{
		MessageID: messageID,
		FileID:    fileID,
		FileName:  fileName,
		FileType:  fileType,
		FileSize:  fileSize,
	}

	if err := s.fileRepo.Create(file); err != nil {
		return nil, fmt.Errorf("failed to create file: %w", err)
	}

	return file, nil
}

func (s *fileService) GetFileByID(id uint) (*models.File, error) {
	return s.fileRepo.GetByID(id)
}

func (s *fileService) GetFilesByMessageID(messageID uint) ([]models.File, error) {
	return s.fileRepo.GetByMessageID(messageID)
}

func (s *fileService) DeleteFile(id uint) error {
	return s.fileRepo.Delete(id)
}
