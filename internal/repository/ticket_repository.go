package repository

import (
	"ai_support_tg_writer_bot/internal/models"
	"time"

	"gorm.io/gorm"
)

type TicketRepository interface {
	Create(ticket *models.Ticket) error
	GetByID(id uint) (*models.Ticket, error)
	GetByUserID(userID uint) ([]models.Ticket, error)
	GetOpenTickets() ([]models.Ticket, error)
	GetAnsweredTickets() ([]models.Ticket, error)
	GetClosedTickets() ([]models.Ticket, error)
	Update(ticket *models.Ticket) error
	CloseTicket(ticketID uint) error
	GetActiveTicketByUserID(userID uint) (*models.Ticket, error)
}

type ticketRepository struct {
	db *gorm.DB
}

func NewTicketRepository(db *gorm.DB) TicketRepository {
	return &ticketRepository{db: db}
}

func (r *ticketRepository) Create(ticket *models.Ticket) error {
	return r.db.Create(ticket).Error
}

func (r *ticketRepository) GetByID(id uint) (*models.Ticket, error) {
	var ticket models.Ticket
	err := r.db.Preload("User").Preload("Messages").Preload("Messages.User").Preload("Messages.Files").First(&ticket, id).Error
	if err != nil {
		return nil, err
	}
	return &ticket, nil
}

func (r *ticketRepository) GetByUserID(userID uint) ([]models.Ticket, error) {
	var tickets []models.Ticket
	err := r.db.Preload("User").Preload("Messages").Preload("Messages.User").Preload("Messages.Files").
		Where("user_id = ?", userID).Order("created_at DESC").Find(&tickets).Error
	return tickets, err
}

func (r *ticketRepository) GetOpenTickets() ([]models.Ticket, error) {
	var tickets []models.Ticket
	err := r.db.Preload("User").Preload("Messages").Preload("Messages.User").Preload("Messages.Files").
		Where("status = ?", models.TicketStatusOpen).Order("created_at ASC").Find(&tickets).Error
	return tickets, err
}

func (r *ticketRepository) GetAnsweredTickets() ([]models.Ticket, error) {
	var tickets []models.Ticket
	err := r.db.Preload("User").Preload("Messages").Preload("Messages.User").Preload("Messages.Files").
		Where("status = ?", models.TicketStatusAnswered).Order("updated_at DESC").Find(&tickets).Error
	return tickets, err
}

func (r *ticketRepository) GetClosedTickets() ([]models.Ticket, error) {
	var tickets []models.Ticket
	err := r.db.Preload("User").Preload("Messages").Preload("Messages.User").Preload("Messages.Files").
		Where("status = ?", models.TicketStatusClosed).Order("closed_at DESC").Find(&tickets).Error
	return tickets, err
}

func (r *ticketRepository) Update(ticket *models.Ticket) error {
	return r.db.Save(ticket).Error
}

func (r *ticketRepository) CloseTicket(ticketID uint) error {
	now := time.Now()
	return r.db.Model(&models.Ticket{}).Where("id = ?", ticketID).Updates(map[string]interface{}{
		"status":    models.TicketStatusClosed,
		"closed_at": &now,
	}).Error
}

func (r *ticketRepository) GetActiveTicketByUserID(userID uint) (*models.Ticket, error) {
	var ticket models.Ticket
	err := r.db.Preload("User").Preload("Messages").Preload("Messages.User").Preload("Messages.Files").
		Where("user_id = ? AND status IN ?", userID, []models.TicketStatus{models.TicketStatusOpen, models.TicketStatusAnswered}).
		Order("created_at DESC").First(&ticket).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &ticket, nil
}
