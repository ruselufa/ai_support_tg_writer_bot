package web

import (
	"ai_support_tg_writer_bot/internal/config"
	"ai_support_tg_writer_bot/internal/models"
	"ai_support_tg_writer_bot/internal/service"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

type WebHandlers struct {
	userService   service.UserService
	ticketService service.TicketService
	fileService   service.FileService
	config        *config.Config
}

func NewWebHandlers(userService service.UserService, ticketService service.TicketService, fileService service.FileService, config *config.Config) *WebHandlers {
	return &WebHandlers{
		userService:   userService,
		ticketService: ticketService,
		fileService:   fileService,
		config:        config,
	}
}

// Middleware для проверки админских прав
func (h *WebHandlers) AdminAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// В реальном приложении здесь должна быть проверка JWT токена или сессии
		// Для простоты используем заголовок X-Admin-ID
		adminIDStr := c.GetHeader("X-Admin-ID")
		if adminIDStr == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			c.Abort()
			return
		}

		adminID, err := strconv.ParseInt(adminIDStr, 10, 64)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid admin ID"})
			c.Abort()
			return
		}

		isAdmin, err := h.userService.IsAdmin(adminID)
		if err != nil || !isAdmin {
			c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
			c.Abort()
			return
		}

		c.Set("admin_id", adminID)
		c.Next()
	}
}

// Главная страница админки
func (h *WebHandlers) AdminDashboard(c *gin.Context) {
	openTickets, err := h.ticketService.GetOpenTickets()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get open tickets"})
		return
	}

	answeredTickets, err := h.ticketService.GetAnsweredTickets()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get answered tickets"})
		return
	}

	closedTickets, err := h.ticketService.GetClosedTickets()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get closed tickets"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"open_tickets":     len(openTickets),
		"answered_tickets": len(answeredTickets),
		"closed_tickets":   len(closedTickets),
		"tickets": gin.H{
			"open":     openTickets,
			"answered": answeredTickets,
			"closed":   closedTickets,
		},
	})
}

// Получить все тикеты
func (h *WebHandlers) GetTickets(c *gin.Context) {
	status := c.Query("status")

	var tickets []models.Ticket
	var err error

	switch status {
	case "open":
		tickets, err = h.ticketService.GetOpenTickets()
	case "answered":
		tickets, err = h.ticketService.GetAnsweredTickets()
	case "closed":
		tickets, err = h.ticketService.GetClosedTickets()
	default:
		// Получаем все тикеты
		openTickets, _ := h.ticketService.GetOpenTickets()
		answeredTickets, _ := h.ticketService.GetAnsweredTickets()
		closedTickets, _ := h.ticketService.GetClosedTickets()

		tickets = append(tickets, openTickets...)
		tickets = append(tickets, answeredTickets...)
		tickets = append(tickets, closedTickets...)
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get tickets"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"tickets": tickets})
}

// Получить конкретный тикет
func (h *WebHandlers) GetTicket(c *gin.Context) {
	ticketIDStr := c.Param("id")
	ticketID, err := strconv.ParseUint(ticketIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ticket ID"})
		return
	}

	ticket, err := h.ticketService.GetTicketByID(uint(ticketID))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Ticket not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"ticket": ticket})
}

// Отправить ответ на тикет
func (h *WebHandlers) ReplyToTicket(c *gin.Context) {
	ticketIDStr := c.Param("id")
	ticketID, err := strconv.ParseUint(ticketIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ticket ID"})
		return
	}

	var request struct {
		Message string `json:"message" binding:"required"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	adminID := c.GetInt64("admin_id")
	admin, err := h.userService.GetUserByTelegramID(adminID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Admin not found"})
		return
	}

	// Добавляем сообщение от админа
	_, err = h.ticketService.AddMessage(uint(ticketID), admin.ID, request.Message, false)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to send reply"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Reply sent successfully"})
}

// Закрыть тикет
func (h *WebHandlers) CloseTicket(c *gin.Context) {
	ticketIDStr := c.Param("id")
	ticketID, err := strconv.ParseUint(ticketIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ticket ID"})
		return
	}

	err = h.ticketService.CloseTicket(uint(ticketID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to close ticket"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Ticket closed successfully"})
}

// Получить статистику
func (h *WebHandlers) GetStats(c *gin.Context) {
	openTickets, _ := h.ticketService.GetOpenTickets()
	answeredTickets, _ := h.ticketService.GetAnsweredTickets()
	closedTickets, _ := h.ticketService.GetClosedTickets()

	// Подсчитываем статистику за последние 7 дней
	weekAgo := time.Now().AddDate(0, 0, -7)

	var recentTickets int
	allTickets := append(openTickets, answeredTickets...)
	allTickets = append(allTickets, closedTickets...)

	for _, ticket := range allTickets {
		if ticket.CreatedAt.After(weekAgo) {
			recentTickets++
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"total_tickets":    len(allTickets),
		"open_tickets":     len(openTickets),
		"answered_tickets": len(answeredTickets),
		"closed_tickets":   len(closedTickets),
		"recent_tickets":   recentTickets,
	})
}
