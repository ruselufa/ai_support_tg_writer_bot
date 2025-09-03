package service

import (
	"ai_support_tg_writer_bot/internal/models"
	"ai_support_tg_writer_bot/internal/repository"
	"fmt"
)

type TicketService interface {
	CreateTicket(userID uint, subject string) (*models.Ticket, error)
	GetTicketByID(id uint) (*models.Ticket, error)
	GetUserTickets(userID uint) ([]models.Ticket, error)
	GetOpenTickets() ([]models.Ticket, error)
	GetAnsweredTickets() ([]models.Ticket, error)
	GetClosedTickets() ([]models.Ticket, error)
	GetActiveTicketByUserID(userID uint) (*models.Ticket, error)
	CloseTicket(ticketID uint) error
	AddMessage(ticketID uint, userID uint, content string, isFromUser bool) (*models.Message, error)
	UpdateTicketStatus(ticketID uint, status models.TicketStatus) error
}

type ticketService struct {
	ticketRepo  repository.TicketRepository
	messageRepo repository.MessageRepository
}

func NewTicketService(ticketRepo repository.TicketRepository, messageRepo repository.MessageRepository) TicketService {
	return &ticketService{
		ticketRepo:  ticketRepo,
		messageRepo: messageRepo,
	}
}

func (s *ticketService) CreateTicket(userID uint, subject string) (*models.Ticket, error) {
	// Проверяем, есть ли у пользователя активный тикет
	activeTicket, err := s.ticketRepo.GetActiveTicketByUserID(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to check active ticket: %w", err)
	}

	if activeTicket != nil {
		return nil, fmt.Errorf("user already has an active ticket")
	}

	ticket := &models.Ticket{
		UserID:  userID,
		Status:  models.TicketStatusOpen,
		Subject: subject,
	}

	if err := s.ticketRepo.Create(ticket); err != nil {
		return nil, fmt.Errorf("failed to create ticket: %w", err)
	}

	return ticket, nil
}

func (s *ticketService) GetTicketByID(id uint) (*models.Ticket, error) {
	return s.ticketRepo.GetByID(id)
}

func (s *ticketService) GetUserTickets(userID uint) ([]models.Ticket, error) {
	return s.ticketRepo.GetByUserID(userID)
}

func (s *ticketService) GetOpenTickets() ([]models.Ticket, error) {
	return s.ticketRepo.GetOpenTickets()
}

func (s *ticketService) GetAnsweredTickets() ([]models.Ticket, error) {
	return s.ticketRepo.GetAnsweredTickets()
}

func (s *ticketService) GetClosedTickets() ([]models.Ticket, error) {
	return s.ticketRepo.GetClosedTickets()
}

func (s *ticketService) GetActiveTicketByUserID(userID uint) (*models.Ticket, error) {
	return s.ticketRepo.GetActiveTicketByUserID(userID)
}

func (s *ticketService) CloseTicket(ticketID uint) error {
	return s.ticketRepo.CloseTicket(ticketID)
}

func (s *ticketService) AddMessage(ticketID uint, userID uint, content string, isFromUser bool) (*models.Message, error) {
	message := &models.Message{
		TicketID:   ticketID,
		UserID:     userID,
		Content:    content,
		IsFromUser: isFromUser,
	}

	if err := s.messageRepo.Create(message); err != nil {
		return nil, fmt.Errorf("failed to create message: %w", err)
	}

	// Обновляем статус тикета
	var newStatus models.TicketStatus
	if isFromUser {
		newStatus = models.TicketStatusOpen
	} else {
		newStatus = models.TicketStatusAnswered
	}

	if err := s.UpdateTicketStatus(ticketID, newStatus); err != nil {
		return nil, fmt.Errorf("failed to update ticket status: %w", err)
	}

	return message, nil
}

func (s *ticketService) UpdateTicketStatus(ticketID uint, status models.TicketStatus) error {
	ticket, err := s.ticketRepo.GetByID(ticketID)
	if err != nil {
		return fmt.Errorf("failed to get ticket: %w", err)
	}

	ticket.Status = status
	return s.ticketRepo.Update(ticket)
}
