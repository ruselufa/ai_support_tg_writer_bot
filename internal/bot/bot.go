package bot

import (
	"ai_support_tg_writer_bot/internal/config"
	"ai_support_tg_writer_bot/internal/models"
	"ai_support_tg_writer_bot/internal/service"
	"fmt"
	"log"
	"strconv"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type Bot struct {
	api           *tgbotapi.BotAPI
	config        *config.Config
	userService   service.UserService
	ticketService service.TicketService
	fileService   service.FileService
	stateManager  *StateManager
}

func NewBot(cfg *config.Config, userService service.UserService, ticketService service.TicketService, fileService service.FileService) (*Bot, error) {
	bot, err := tgbotapi.NewBotAPI(cfg.TelegramBotToken)
	if err != nil {
		return nil, fmt.Errorf("failed to create bot: %w", err)
	}

	bot.Debug = false

	log.Printf("Authorized on account %s", bot.Self.UserName)

	return &Bot{
		api:           bot,
		config:        cfg,
		userService:   userService,
		ticketService: ticketService,
		fileService:   fileService,
		stateManager:  NewStateManager(),
	}, nil
}

func (b *Bot) Start() error {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := b.api.GetUpdatesChan(u)

	for update := range updates {
		if update.Message != nil {
			b.handleMessage(update.Message)
		} else if update.CallbackQuery != nil {
			b.handleCallbackQuery(update.CallbackQuery)
		}
	}

	return nil
}

func (b *Bot) handleMessage(message *tgbotapi.Message) {
	// Получаем или создаем пользователя
	user, err := b.userService.CreateOrGetUser(
		int64(message.From.ID),
		message.From.UserName,
		message.From.FirstName,
		message.From.LastName,
	)
	if err != nil {
		log.Printf("Failed to create/get user: %v", err)
		return
	}

	// Проверяем, является ли пользователь админом
	isAdmin := b.isUserAdmin(int64(message.From.ID))

	// Если пользователь админ по конфигурации, но не в базе - обновляем базу
	if isAdmin && !user.IsAdmin {
		b.userService.SetAdmin(int64(message.From.ID), true)
	}

	// Обрабатываем команды
	if message.IsCommand() {
		b.handleCommand(message, user, isAdmin)
		return
	}

	// Обрабатываем обычные сообщения
	b.handleRegularMessage(message, user, isAdmin)
}

func (b *Bot) handleCommand(message *tgbotapi.Message, user *models.User, isAdmin bool) {
	switch message.Command() {
	case "start":
		b.handleStartCommand(message, user)
	case "help":
		b.handleHelpCommand(message, user)
	case "admin":
		if isAdmin {
			b.handleAdminCommand(message, user)
		} else {
			b.sendMessage(message.Chat.ID, "У вас нет прав для выполнения этой команды.")
		}
	case "cancel":
		if isAdmin {
			b.handleCancelCommand(message, user)
		} else {
			b.sendMessage(message.Chat.ID, "У вас нет прав для выполнения этой команды.")
		}
	case "tickets":
		if isAdmin {
			b.handleTicketsCommand(message, user)
		} else {
			b.sendMessage(message.Chat.ID, "У вас нет прав для выполнения этой команды.")
		}
	default:
		b.sendMessage(message.Chat.ID, "Неизвестная команда. Используйте /help для получения списка команд.")
	}
}

func (b *Bot) handleStartCommand(message *tgbotapi.Message, user *models.User) {
	welcomeText := `🤖 Добро пожаловать в службу технической поддержки Social Flow!

Здесь вы можете:
• Задать вопросы по работе с ботом
• Отправить отзывы о багах или ошибках
• Приложить скриншоты или записи экрана

Просто напишите ваш вопрос или проблему, и мы обязательно поможем!`

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📝 Создать тикет", "create_ticket"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📋 Мои тикеты", "my_tickets"),
		),
	)

	msg := tgbotapi.NewMessage(message.Chat.ID, welcomeText)
	msg.ReplyMarkup = keyboard
	b.api.Send(msg)
}

func (b *Bot) handleHelpCommand(message *tgbotapi.Message, user *models.User) {
	helpText := `📖 Доступные команды:

/start - Начать работу с ботом
/help - Показать эту справку`

	// Добавляем админские команды если пользователь админ
	if b.isUserAdmin(int64(message.From.ID)) {
		helpText += `

👨‍💼 Админские команды:
/admin - Админская панель
/cancel - Отменить режим ответа на тикет`
	}

	helpText += `

Для создания тикета поддержки просто напишите ваш вопрос или проблему. Вы также можете приложить скриншоты или видео для лучшего понимания проблемы.`

	b.sendMessage(message.Chat.ID, helpText)
}

func (b *Bot) handleAdminCommand(message *tgbotapi.Message, user *models.User) {
	// Очищаем состояние ответа при входе в админку
	b.stateManager.ClearUserState(int64(message.From.ID))

	adminText := `👨‍💼 Админская панель

Выберите действие:`

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📋 Открытые тикеты", "admin_open_tickets"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("💬 Отвеченные тикеты", "admin_answered_tickets"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("✅ Закрытые тикеты", "admin_closed_tickets"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📊 Статистика", "admin_stats"),
		),
	)

	msg := tgbotapi.NewMessage(message.Chat.ID, adminText)
	msg.ReplyMarkup = keyboard
	b.api.Send(msg)
}

func (b *Bot) handleCancelCommand(message *tgbotapi.Message, user *models.User) {
	// Очищаем состояние ответа
	b.stateManager.ClearUserState(int64(message.From.ID))
	b.sendMessage(message.Chat.ID, "✅ Режим ответа отменен. Используйте /admin для доступа к админской панели.")
}

func (b *Bot) handleTicketsCommand(message *tgbotapi.Message, user *models.User) {
	// Админская команда для просмотра тикетов
	openTickets, err := b.ticketService.GetOpenTickets()
	if err != nil {
		b.sendMessage(message.Chat.ID, "Ошибка при получении тикетов.")
		return
	}

	if len(openTickets) == 0 {
		b.sendMessage(message.Chat.ID, "Нет открытых тикетов.")
		return
	}

	text := "📋 Открытые тикеты:\n\n"
	for _, ticket := range openTickets {
		text += fmt.Sprintf("🔸 Тикет #%d\n", ticket.ID)
		text += fmt.Sprintf("👤 Пользователь: %s\n", ticket.User.FirstName)
		text += fmt.Sprintf("📝 Тема: %s\n", ticket.Subject)
		text += fmt.Sprintf("📅 Создан: %s\n\n", ticket.CreatedAt.Format("02.01.2006 15:04"))
	}

	b.sendMessage(message.Chat.ID, text)
}

func (b *Bot) handleRegularMessage(message *tgbotapi.Message, user *models.User, isAdmin bool) {
	// Проверяем, находится ли админ в режиме ответа на тикет
	if isAdmin {
		isReplying, ticketID := b.stateManager.IsReplyingToTicket(int64(message.From.ID))
		if isReplying {
			// Админ отвечает на тикет
			admin, err := b.userService.GetUserByTelegramID(int64(message.From.ID))
			if err != nil {
				log.Printf("Failed to get admin user: %v", err)
				b.sendMessage(message.Chat.ID, "Произошла ошибка.")
				return
			}

			// Добавляем ответ от админа
			_, err = b.ticketService.AddMessage(ticketID, admin.ID, message.Text, false)
			if err != nil {
				log.Printf("Failed to add admin message: %v", err)
				b.sendMessage(message.Chat.ID, "Произошла ошибка при отправке ответа.")
				return
			}

			// Обрабатываем файлы если есть
			if err := b.handleMessageFiles(message, ticketID, admin.ID); err != nil {
				log.Printf("Failed to handle files: %v", err)
			}

			b.sendMessage(message.Chat.ID, "✅ Ответ отправлен в тикет #"+strconv.Itoa(int(ticketID)))

			// Очищаем состояние ответа
			b.stateManager.ClearUserState(int64(message.From.ID))
			return
		}
	}

	// Обычная логика для клиентов
	// Проверяем, есть ли у пользователя активный тикет
	activeTicket, err := b.ticketService.GetActiveTicketByUserID(user.ID)
	if err != nil {
		log.Printf("Failed to get active ticket: %v", err)
		b.sendMessage(message.Chat.ID, "Произошла ошибка. Попробуйте позже.")
		return
	}

	if activeTicket == nil {
		// Создаем новый тикет
		subject := "Новый тикет"
		if len(message.Text) > 50 {
			subject = message.Text[:50] + "..."
		} else if message.Text != "" {
			subject = message.Text
		}

		ticket, err := b.ticketService.CreateTicket(user.ID, subject)
		if err != nil {
			log.Printf("Failed to create ticket: %v", err)
			b.sendMessage(message.Chat.ID, "Не удалось создать тикет. У вас уже есть активный тикет.")
			return
		}

		// Добавляем первое сообщение
		_, err = b.ticketService.AddMessage(ticket.ID, user.ID, message.Text, true)
		if err != nil {
			log.Printf("Failed to add message: %v", err)
		}

		// Обрабатываем файлы если есть
		if err := b.handleMessageFiles(message, ticket.ID, user.ID); err != nil {
			log.Printf("Failed to handle files: %v", err)
		}

		b.sendMessage(message.Chat.ID, "✅ Тикет создан! Мы получили ваше сообщение и скоро ответим.")
	} else {
		// Добавляем сообщение к существующему тикету
		_, err = b.ticketService.AddMessage(activeTicket.ID, user.ID, message.Text, true)
		if err != nil {
			log.Printf("Failed to add message: %v", err)
			b.sendMessage(message.Chat.ID, "Произошла ошибка при отправке сообщения.")
			return
		}

		// Обрабатываем файлы если есть
		if err := b.handleMessageFiles(message, activeTicket.ID, user.ID); err != nil {
			log.Printf("Failed to handle files: %v", err)
		}

		b.sendMessage(message.Chat.ID, "✅ Сообщение добавлено к тикету #"+strconv.Itoa(int(activeTicket.ID)))
	}
}

func (b *Bot) handleMessageFiles(message *tgbotapi.Message, ticketID uint, userID uint) error {
	// Получаем последнее сообщение тикета
	messages, err := b.ticketService.GetTicketByID(ticketID)
	if err != nil {
		return err
	}

	if len(messages.Messages) == 0 {
		return fmt.Errorf("no messages found for ticket")
	}

	lastMessage := messages.Messages[len(messages.Messages)-1]

	// Обрабатываем различные типы файлов
	if message.Photo != nil && len(message.Photo) > 0 {
		photo := message.Photo[len(message.Photo)-1] // Берем самое большое фото
		_, err = b.fileService.CreateFile(
			lastMessage.ID,
			photo.FileID,
			"photo.jpg",
			"photo",
			int64(photo.FileSize),
		)
		return err
	}

	if message.Video != nil {
		_, err = b.fileService.CreateFile(
			lastMessage.ID,
			message.Video.FileID,
			message.Video.FileName,
			"video",
			int64(message.Video.FileSize),
		)
		return err
	}

	if message.Document != nil {
		_, err = b.fileService.CreateFile(
			lastMessage.ID,
			message.Document.FileID,
			message.Document.FileName,
			"document",
			int64(message.Document.FileSize),
		)
		return err
	}

	if message.Voice != nil {
		_, err = b.fileService.CreateFile(
			lastMessage.ID,
			message.Voice.FileID,
			"voice.ogg",
			"voice",
			int64(message.Voice.FileSize),
		)
		return err
	}

	if message.VideoNote != nil {
		_, err = b.fileService.CreateFile(
			lastMessage.ID,
			message.VideoNote.FileID,
			"video_note.mp4",
			"video_note",
			int64(message.VideoNote.FileSize),
		)
		return err
	}

	return nil
}

func (b *Bot) handleCallbackQuery(query *tgbotapi.CallbackQuery) {
	// Обработка callback запросов от inline кнопок
	switch query.Data {
	case "create_ticket":
		b.handleCreateTicketCallback(query)
	case "my_tickets":
		b.handleMyTicketsCallback(query)
	case "admin_open_tickets":
		b.handleAdminOpenTicketsCallback(query)
	case "admin_answered_tickets":
		b.handleAdminAnsweredTicketsCallback(query)
	case "admin_closed_tickets":
		b.handleAdminClosedTicketsCallback(query)
	case "admin_stats":
		b.handleAdminStatsCallback(query)
	default:
		if strings.HasPrefix(query.Data, "reply_") {
			b.handleReplyCallback(query)
		} else if strings.HasPrefix(query.Data, "close_") {
			b.handleCloseTicketCallback(query)
		} else if strings.HasPrefix(query.Data, "view_ticket_") {
			b.handleViewTicketCallback(query)
		} else if strings.HasPrefix(query.Data, "admin_reply_") {
			b.handleAdminReplyCallback(query)
		}
	}
}

func (b *Bot) handleCreateTicketCallback(query *tgbotapi.CallbackQuery) {
	user, err := b.userService.GetUserByTelegramID(int64(query.From.ID))
	if err != nil {
		b.answerCallbackQuery(query.ID, "Ошибка при получении данных пользователя.")
		return
	}

	activeTicket, err := b.ticketService.GetActiveTicketByUserID(user.ID)
	if err != nil {
		b.answerCallbackQuery(query.ID, "Произошла ошибка.")
		return
	}

	if activeTicket != nil {
		b.answerCallbackQuery(query.ID, "У вас уже есть активный тикет #"+strconv.Itoa(int(activeTicket.ID)))
		return
	}

	b.answerCallbackQuery(query.ID, "Просто напишите ваш вопрос или проблему!")
}

func (b *Bot) handleMyTicketsCallback(query *tgbotapi.CallbackQuery) {
	user, err := b.userService.GetUserByTelegramID(int64(query.From.ID))
	if err != nil {
		b.answerCallbackQuery(query.ID, "Ошибка при получении данных пользователя.")
		return
	}

	tickets, err := b.ticketService.GetUserTickets(user.ID)
	if err != nil {
		b.answerCallbackQuery(query.ID, "Ошибка при получении тикетов.")
		return
	}

	if len(tickets) == 0 {
		b.answerCallbackQuery(query.ID, "У вас пока нет тикетов.")
		return
	}

	text := "📋 Ваши тикеты:\n\n"
	for _, ticket := range tickets {
		status := "🟢 Открыт"
		if ticket.Status == models.TicketStatusAnswered {
			status = "🟡 Отвечен"
		} else if ticket.Status == models.TicketStatusClosed {
			status = "🔴 Закрыт"
		}

		text += fmt.Sprintf("🔸 Тикет #%d - %s\n", ticket.ID, status)
		text += fmt.Sprintf("📝 %s\n", ticket.Subject)
		text += fmt.Sprintf("📅 %s\n\n", ticket.CreatedAt.Format("02.01.2006 15:04"))
	}

	msg := tgbotapi.NewMessage(query.Message.Chat.ID, text)
	b.api.Send(msg)
	b.answerCallbackQuery(query.ID, "")
}

func (b *Bot) handleReplyCallback(query *tgbotapi.CallbackQuery) {
	// Обработка ответа на тикет (для админов)
	parts := strings.Split(query.Data, "_")
	if len(parts) != 2 {
		return
	}

	_, err := strconv.ParseUint(parts[1], 10, 32)
	if err != nil {
		return
	}

	// Здесь можно реализовать логику для админов
	b.answerCallbackQuery(query.ID, "Функция ответа будет реализована в админской панели.")
}

func (b *Bot) handleCloseTicketCallback(query *tgbotapi.CallbackQuery) {
	// Обработка закрытия тикета
	parts := strings.Split(query.Data, "_")
	if len(parts) != 2 {
		return
	}

	ticketID, err := strconv.ParseUint(parts[1], 10, 32)
	if err != nil {
		return
	}

	user, err := b.userService.GetUserByTelegramID(int64(query.From.ID))
	if err != nil {
		b.answerCallbackQuery(query.ID, "Ошибка при получении данных пользователя.")
		return
	}

	ticket, err := b.ticketService.GetTicketByID(uint(ticketID))
	if err != nil {
		b.answerCallbackQuery(query.ID, "Тикет не найден.")
		return
	}

	if ticket.UserID != user.ID {
		b.answerCallbackQuery(query.ID, "У вас нет прав для закрытия этого тикета.")
		return
	}

	if err := b.ticketService.CloseTicket(uint(ticketID)); err != nil {
		b.answerCallbackQuery(query.ID, "Ошибка при закрытии тикета.")
		return
	}

	b.answerCallbackQuery(query.ID, "Тикет закрыт.")
}

func (b *Bot) handleAdminOpenTicketsCallback(query *tgbotapi.CallbackQuery) {
	// Проверяем права админа
	if !b.isUserAdmin(int64(query.From.ID)) {
		b.answerCallbackQuery(query.ID, "У вас нет прав администратора.")
		return
	}

	tickets, err := b.ticketService.GetOpenTickets()
	if err != nil {
		b.answerCallbackQuery(query.ID, "Ошибка при получении тикетов.")
		return
	}

	if len(tickets) == 0 {
		msg := tgbotapi.NewMessage(query.Message.Chat.ID, "📋 Открытых тикетов нет.")
		b.api.Send(msg)
		b.answerCallbackQuery(query.ID, "")
		return
	}

	text := "📋 Открытые тикеты:\n\n"
	for _, ticket := range tickets {
		text += fmt.Sprintf("🔸 Тикет #%d\n", ticket.ID)
		text += fmt.Sprintf("👤 %s %s\n", ticket.User.FirstName, ticket.User.LastName)
		text += fmt.Sprintf("📝 %s\n", ticket.Subject)
		text += fmt.Sprintf("📅 %s\n\n", ticket.CreatedAt.Format("02.01.2006 15:04"))
	}

	// Создаем кнопки для каждого тикета
	var buttons [][]tgbotapi.InlineKeyboardButton
	for _, ticket := range tickets {
		button := tgbotapi.NewInlineKeyboardButtonData(
			fmt.Sprintf("🔸 #%d - %s", ticket.ID, ticket.Subject[:min(30, len(ticket.Subject))]),
			fmt.Sprintf("view_ticket_%d", ticket.ID),
		)
		buttons = append(buttons, tgbotapi.NewInlineKeyboardRow(button))
	}

	keyboard := tgbotapi.NewInlineKeyboardMarkup(buttons...)

	msg := tgbotapi.NewMessage(query.Message.Chat.ID, text)
	msg.ReplyMarkup = keyboard
	b.api.Send(msg)
	b.answerCallbackQuery(query.ID, "")
}

func (b *Bot) handleAdminAnsweredTicketsCallback(query *tgbotapi.CallbackQuery) {
	// Проверяем права админа
	if !b.isUserAdmin(int64(query.From.ID)) {
		b.answerCallbackQuery(query.ID, "У вас нет прав администратора.")
		return
	}

	tickets, err := b.ticketService.GetAnsweredTickets()
	if err != nil {
		b.answerCallbackQuery(query.ID, "Ошибка при получении тикетов.")
		return
	}

	if len(tickets) == 0 {
		msg := tgbotapi.NewMessage(query.Message.Chat.ID, "💬 Отвеченных тикетов нет.")
		b.api.Send(msg)
		b.answerCallbackQuery(query.ID, "")
		return
	}

	text := "💬 Отвеченные тикеты:\n\n"
	for _, ticket := range tickets {
		text += fmt.Sprintf("🔸 Тикет #%d\n", ticket.ID)
		text += fmt.Sprintf("👤 %s %s\n", ticket.User.FirstName, ticket.User.LastName)
		text += fmt.Sprintf("📝 %s\n", ticket.Subject)
		text += fmt.Sprintf("📅 %s\n\n", ticket.UpdatedAt.Format("02.01.2006 15:04"))
	}

	// Создаем кнопки для каждого тикета
	var buttons [][]tgbotapi.InlineKeyboardButton
	for _, ticket := range tickets {
		button := tgbotapi.NewInlineKeyboardButtonData(
			fmt.Sprintf("🔸 #%d - %s", ticket.ID, ticket.Subject[:min(30, len(ticket.Subject))]),
			fmt.Sprintf("view_ticket_%d", ticket.ID),
		)
		buttons = append(buttons, tgbotapi.NewInlineKeyboardRow(button))
	}

	keyboard := tgbotapi.NewInlineKeyboardMarkup(buttons...)

	msg := tgbotapi.NewMessage(query.Message.Chat.ID, text)
	msg.ReplyMarkup = keyboard
	b.api.Send(msg)
	b.answerCallbackQuery(query.ID, "")
}

func (b *Bot) handleAdminClosedTicketsCallback(query *tgbotapi.CallbackQuery) {
	// Проверяем права админа
	if !b.isUserAdmin(int64(query.From.ID)) {
		b.answerCallbackQuery(query.ID, "У вас нет прав администратора.")
		return
	}

	tickets, err := b.ticketService.GetClosedTickets()
	if err != nil {
		b.answerCallbackQuery(query.ID, "Ошибка при получении тикетов.")
		return
	}

	if len(tickets) == 0 {
		msg := tgbotapi.NewMessage(query.Message.Chat.ID, "✅ Закрытых тикетов нет.")
		b.api.Send(msg)
		b.answerCallbackQuery(query.ID, "")
		return
	}

	text := "✅ Закрытые тикеты:\n\n"
	for _, ticket := range tickets {
		text += fmt.Sprintf("🔸 Тикет #%d\n", ticket.ID)
		text += fmt.Sprintf("👤 %s %s\n", ticket.User.FirstName, ticket.User.LastName)
		text += fmt.Sprintf("📝 %s\n", ticket.Subject)
		if ticket.ClosedAt != nil {
			text += fmt.Sprintf("📅 Закрыт: %s\n\n", ticket.ClosedAt.Format("02.01.2006 15:04"))
		}
	}

	// Создаем кнопки для каждого тикета
	var buttons [][]tgbotapi.InlineKeyboardButton
	for _, ticket := range tickets {
		button := tgbotapi.NewInlineKeyboardButtonData(
			fmt.Sprintf("🔸 #%d - %s", ticket.ID, ticket.Subject[:min(30, len(ticket.Subject))]),
			fmt.Sprintf("view_ticket_%d", ticket.ID),
		)
		buttons = append(buttons, tgbotapi.NewInlineKeyboardRow(button))
	}

	keyboard := tgbotapi.NewInlineKeyboardMarkup(buttons...)

	msg := tgbotapi.NewMessage(query.Message.Chat.ID, text)
	msg.ReplyMarkup = keyboard
	b.api.Send(msg)
	b.answerCallbackQuery(query.ID, "")
}

func (b *Bot) handleAdminStatsCallback(query *tgbotapi.CallbackQuery) {
	// Проверяем права админа
	if !b.isUserAdmin(int64(query.From.ID)) {
		b.answerCallbackQuery(query.ID, "У вас нет прав администратора.")
		return
	}

	openTickets, _ := b.ticketService.GetOpenTickets()
	answeredTickets, _ := b.ticketService.GetAnsweredTickets()
	closedTickets, _ := b.ticketService.GetClosedTickets()

	text := "📊 Статистика тикетов:\n\n"
	text += fmt.Sprintf("🟢 Открытые: %d\n", len(openTickets))
	text += fmt.Sprintf("🟡 Отвеченные: %d\n", len(answeredTickets))
	text += fmt.Sprintf("🔴 Закрытые: %d\n", len(closedTickets))
	text += fmt.Sprintf("📈 Всего: %d\n", len(openTickets)+len(answeredTickets)+len(closedTickets))

	msg := tgbotapi.NewMessage(query.Message.Chat.ID, text)
	b.api.Send(msg)
	b.answerCallbackQuery(query.ID, "")
}

func (b *Bot) handleViewTicketCallback(query *tgbotapi.CallbackQuery) {
	// Проверяем права админа
	if !b.isUserAdmin(int64(query.From.ID)) {
		b.answerCallbackQuery(query.ID, "У вас нет прав администратора.")
		return
	}

	// Извлекаем ID тикета
	parts := strings.Split(query.Data, "_")
	if len(parts) != 3 {
		return
	}

	ticketID, err := strconv.ParseUint(parts[2], 10, 32)
	if err != nil {
		return
	}

	ticket, err := b.ticketService.GetTicketByID(uint(ticketID))
	if err != nil {
		b.answerCallbackQuery(query.ID, "Тикет не найден.")
		return
	}

	// Формируем текст с информацией о тикете
	text := fmt.Sprintf("🔸 Тикет #%d\n", ticket.ID)
	text += fmt.Sprintf("👤 Пользователь: %s %s\n", ticket.User.FirstName, ticket.User.LastName)
	text += fmt.Sprintf("📝 Тема: %s\n", ticket.Subject)
	text += fmt.Sprintf("📅 Создан: %s\n", ticket.CreatedAt.Format("02.01.2006 15:04"))

	status := "🟢 Открыт"
	if ticket.Status == models.TicketStatusAnswered {
		status = "🟡 Отвечен"
	} else if ticket.Status == models.TicketStatusClosed {
		status = "🔴 Закрыт"
	}
	text += fmt.Sprintf("📊 Статус: %s\n\n", status)

	// Добавляем сообщения
	text += "💬 Сообщения:\n"
	for _, message := range ticket.Messages {
		sender := "👤 Клиент"
		if !message.IsFromUser {
			sender = "👨‍💼 Админ"
		}
		text += fmt.Sprintf("%s: %s\n", sender, message.Content)
		text += fmt.Sprintf("📅 %s\n\n", message.CreatedAt.Format("02.01.2006 15:04"))
	}

	// Создаем кнопки
	var buttons [][]tgbotapi.InlineKeyboardButton

	if ticket.Status != models.TicketStatusClosed {
		replyButton := tgbotapi.NewInlineKeyboardButtonData(
			"💬 Ответить",
			fmt.Sprintf("admin_reply_%d", ticket.ID),
		)
		buttons = append(buttons, tgbotapi.NewInlineKeyboardRow(replyButton))
	}

	closeButton := tgbotapi.NewInlineKeyboardButtonData(
		"✅ Закрыть тикет",
		fmt.Sprintf("close_%d", ticket.ID),
	)
	buttons = append(buttons, tgbotapi.NewInlineKeyboardRow(closeButton))

	backButton := tgbotapi.NewInlineKeyboardButtonData(
		"🔙 Назад к админке",
		"admin_open_tickets",
	)
	buttons = append(buttons, tgbotapi.NewInlineKeyboardRow(backButton))

	keyboard := tgbotapi.NewInlineKeyboardMarkup(buttons...)

	msg := tgbotapi.NewMessage(query.Message.Chat.ID, text)
	msg.ReplyMarkup = keyboard
	b.api.Send(msg)
	b.answerCallbackQuery(query.ID, "")
}

func (b *Bot) handleAdminReplyCallback(query *tgbotapi.CallbackQuery) {
	// Проверяем права админа
	if !b.isUserAdmin(int64(query.From.ID)) {
		b.answerCallbackQuery(query.ID, "У вас нет прав администратора.")
		return
	}

	// Извлекаем ID тикета
	parts := strings.Split(query.Data, "_")
	if len(parts) != 3 {
		return
	}

	ticketID, err := strconv.ParseUint(parts[2], 10, 32)
	if err != nil {
		return
	}

	// Устанавливаем состояние ответа на тикет
	b.stateManager.SetReplyingState(int64(query.From.ID), uint(ticketID))

	text := fmt.Sprintf("💬 Ответ на тикет #%d\n\nНапишите ваш ответ:", ticketID)

	msg := tgbotapi.NewMessage(query.Message.Chat.ID, text)
	b.api.Send(msg)
	b.answerCallbackQuery(query.ID, "Напишите ответ на тикет.")
}

// Вспомогательная функция для min
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func (b *Bot) sendMessage(chatID int64, text string) {
	msg := tgbotapi.NewMessage(chatID, text)
	b.api.Send(msg)
}

func (b *Bot) answerCallbackQuery(queryID, text string) {
	callback := tgbotapi.NewCallback(queryID, text)
	b.api.Request(callback)
}

func (b *Bot) GetBotUsername() string {
	return b.api.Self.UserName
}

func (b *Bot) isUserAdmin(telegramID int64) bool {
	for _, adminID := range b.config.AdminIDs {
		if adminID == telegramID {
			return true
		}
	}
	return false
}
