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

const (
	CHATS_PER_PAGE    = 5
	MESSAGES_PER_PAGE = 10
)

type ChatBot struct {
	api          *tgbotapi.BotAPI
	config       *config.Config
	userService  service.UserService
	chatService  service.ChatService
	fileService  service.FileService
	stateManager *StateManager
}

func NewChatBot(cfg *config.Config, userService service.UserService, chatService service.ChatService, fileService service.FileService) (*ChatBot, error) {
	bot, err := tgbotapi.NewBotAPI(cfg.TelegramBotToken)
	if err != nil {
		return nil, fmt.Errorf("failed to create bot: %w", err)
	}

	bot.Debug = false

	log.Printf("Authorized on account %s", bot.Self.UserName)

	return &ChatBot{
		api:          bot,
		config:       cfg,
		userService:  userService,
		chatService:  chatService,
		fileService:  fileService,
		stateManager: NewStateManager(),
	}, nil
}

func (b *ChatBot) Start() error {
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

func (b *ChatBot) handleMessage(message *tgbotapi.Message) {
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

func (b *ChatBot) handleCommand(message *tgbotapi.Message, user *models.User, isAdmin bool) {
	switch message.Command() {
	case "start":
		b.handleStartCommand(message, user)
	case "help":
		b.handleHelpCommand(message, user, isAdmin)
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
	default:
		b.sendMessage(message.Chat.ID, "Неизвестная команда. Используйте /help для получения списка команд.")
	}
}

func (b *ChatBot) handleStartCommand(message *tgbotapi.Message, user *models.User) {
	welcomeText := `🤖 Добро пожаловать в службу технической поддержки Social Flow!

Здесь вы можете:
• Задать вопросы по работе с ботом
• Отправить отзывы о багах или ошибках
• Приложить скриншоты или записи экрана

Просто напишите ваш вопрос или проблему, и мы обязательно поможем!`

	b.sendMessage(message.Chat.ID, welcomeText)
}

func (b *ChatBot) handleHelpCommand(message *tgbotapi.Message, user *models.User, isAdmin bool) {
	helpText := `📖 Доступные команды:

/start - Начать работу с ботом
/help - Показать эту справку`

	// Добавляем админские команды если пользователь админ
	if isAdmin {
		helpText += `

👨‍💼 Админские команды:
/admin - Админская панель
/cancel - Отменить режим ответа на чат`
	}

	helpText += `

Для создания чата поддержки просто напишите ваш вопрос или проблему. Вы также можете приложить скриншоты или видео для лучшего понимания проблемы.`

	b.sendMessage(message.Chat.ID, helpText)
}

func (b *ChatBot) handleAdminCommand(message *tgbotapi.Message, user *models.User) {
	// Очищаем состояние ответа при входе в админку
	b.stateManager.ClearUserState(int64(message.From.ID))

	// Получаем количество непрочитанных чатов
	unreadCount, _ := b.chatService.GetUnreadChatsCount()

	adminText := fmt.Sprintf(`👨‍💼 Админская панель

Непрочитанных чатов: %d

Выберите действие:`, unreadCount)

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("💬 Активные чаты", "admin_active_chats"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📁 Архивированные чаты", "admin_archived_chats"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📊 Статистика", "admin_stats"),
		),
	)

	msg := tgbotapi.NewMessage(message.Chat.ID, adminText)
	msg.ReplyMarkup = keyboard
	b.api.Send(msg)
}

func (b *ChatBot) handleCancelCommand(message *tgbotapi.Message, user *models.User) {
	// Очищаем состояние ответа
	b.stateManager.ClearUserState(int64(message.From.ID))
	b.sendMessage(message.Chat.ID, "✅ Режим ответа отменен. Используйте /admin для доступа к админской панели.")
}

func (b *ChatBot) handleRegularMessage(message *tgbotapi.Message, user *models.User, isAdmin bool) {
	// Проверяем, находится ли админ в режиме ответа на чат
	if isAdmin {
		isReplying, chatID := b.stateManager.IsReplyingToTicket(int64(message.From.ID))
		if isReplying {
			// Админ отвечает в чат
			admin, err := b.userService.GetUserByTelegramID(int64(message.From.ID))
			if err != nil {
				log.Printf("Failed to get admin user: %v", err)
				b.sendMessage(message.Chat.ID, "Произошла ошибка.")
				return
			}

			// Определяем содержимое сообщения (текст или подпись к медиа)
			content := message.Text
			if content == "" && message.Caption != "" {
				content = message.Caption
			}

			// Добавляем ответ от админа
			adminMessage, err := b.chatService.AddMessage(chatID, admin.ID, content, false)
			if err != nil {
				log.Printf("Failed to add admin message: %v", err)
				b.sendMessage(message.Chat.ID, "Произошла ошибка при отправке ответа.")
				return
			}

			// Обрабатываем файлы если есть
			if err := b.handleMessageFiles(message, adminMessage.ID); err != nil {
				log.Printf("Failed to handle files: %v", err)
			}

			// Отправляем ответ клиенту
			if err := b.sendResponseToClient(chatID, content, message); err != nil {
				log.Printf("Failed to send response to client: %v", err)
			}

			// Показываем кнопку "Закончить разговор"
			b.showFinishConversationButton(message.Chat.ID, chatID)

			// НЕ очищаем состояние ответа - админ остается в режиме ответа
			return
		}
	}

	// Если это админ без выбора меню - игнорируем
	if isAdmin {
		// Админ пишет без выбора чата - ничего не делаем
		return
	}

	// Обычная логика для клиентов
	// Создаем или получаем чат пользователя
	chat, err := b.chatService.CreateOrGetChat(user.ID)
	if err != nil {
		log.Printf("Failed to create/get chat: %v", err)
		b.sendMessage(message.Chat.ID, "Произошла ошибка. Попробуйте позже.")
		return
	}

	// Определяем содержимое сообщения (текст или подпись к медиа)
	content := message.Text
	if content == "" && message.Caption != "" {
		content = message.Caption
	}

	// Добавляем сообщение в чат
	chatMessage, err := b.chatService.AddMessage(chat.ID, user.ID, content, true)
	if err != nil {
		log.Printf("Failed to add message: %v", err)
		b.sendMessage(message.Chat.ID, "Произошла ошибка при отправке сообщения.")
		return
	}

	// Обрабатываем файлы если есть
	if err := b.handleMessageFiles(message, chatMessage.ID); err != nil {
		log.Printf("Failed to handle files: %v", err)
	}

	// Отправляем уведомление админам о новом сообщении
	b.notifyAdminsAboutNewMessage(chat.ID, user, content)

	// Отправляем медиа файлы админам если есть
	if err := b.sendMediaToAdmins(chat.ID, user, message); err != nil {
		log.Printf("Failed to send media to admins: %v", err)
	}

	b.sendMessage(message.Chat.ID, "✅ Сообщение отправлено! Мы получили ваше сообщение и скоро ответим.")
}

func (b *ChatBot) handleMessageFiles(message *tgbotapi.Message, messageID uint) error {

	// Обрабатываем различные типы файлов
	if message.Photo != nil && len(message.Photo) > 0 {
		photo := message.Photo[len(message.Photo)-1] // Берем самое большое фото
		_, err := b.fileService.CreateFile(
			messageID,
			photo.FileID,
			"photo.jpg",
			"photo",
			int64(photo.FileSize),
		)
		if err != nil {
			return err
		}
	}

	if message.Video != nil {
		_, err := b.fileService.CreateFile(
			messageID,
			message.Video.FileID,
			message.Video.FileName,
			"video",
			int64(message.Video.FileSize),
		)
		if err != nil {
			return err
		}
	}

	if message.Document != nil {
		_, err := b.fileService.CreateFile(
			messageID,
			message.Document.FileID,
			message.Document.FileName,
			"document",
			int64(message.Document.FileSize),
		)
		if err != nil {
			return err
		}
	}

	if message.Voice != nil {
		_, err := b.fileService.CreateFile(
			messageID,
			message.Voice.FileID,
			"voice.ogg",
			"voice",
			int64(message.Voice.FileSize),
		)
		if err != nil {
			return err
		}
	}

	if message.VideoNote != nil {
		_, err := b.fileService.CreateFile(
			messageID,
			message.VideoNote.FileID,
			"video_note.mp4",
			"video_note",
			int64(message.VideoNote.FileSize),
		)
		if err != nil {
			return err
		}
	}

	return nil
}

func (b *ChatBot) handleCallbackQuery(query *tgbotapi.CallbackQuery) {
	// Обработка callback запросов от inline кнопок
	switch query.Data {
	case "admin_active_chats":
		b.handleAdminActiveChatsCallback(query)
	case "admin_archived_chats":
		b.handleAdminArchivedChatsCallback(query)
	case "admin_stats":
		b.handleAdminStatsCallback(query)
	default:
		if strings.HasPrefix(query.Data, "view_chat_") {
			if strings.Contains(query.Data, "_page_") {
				b.handleViewChatPageCallback(query)
			} else {
				b.handleViewChatCallback(query)
			}
		} else if strings.HasPrefix(query.Data, "admin_reply_") {
			b.handleAdminReplyCallback(query)
		} else if strings.HasPrefix(query.Data, "archive_chat_") {
			b.handleArchiveChatCallback(query)
		} else if strings.HasPrefix(query.Data, "finish_conversation_") {
			b.handleFinishConversationCallback(query)
		} else if query.Data == "continue_chat" {
			b.handleContinueChatCallback(query)
		} else if strings.HasPrefix(query.Data, "active_chats_page_") {
			b.handleActiveChatsPageCallback(query)
		} else if strings.HasPrefix(query.Data, "archived_chats_page_") {
			b.handleArchivedChatsPageCallback(query)
		} else if strings.HasPrefix(query.Data, "detailed_history_") {
			b.handleDetailedHistoryCallback(query)
		} else if query.Data == "admin_menu" {
			b.handleAdminMenuCallback(query)
		}
	}
}

func (b *ChatBot) handleAdminActiveChatsCallback(query *tgbotapi.CallbackQuery) {
	b.handleAdminActiveChatsCallbackWithPage(query, 0)
}

func (b *ChatBot) handleAdminActiveChatsCallbackWithPage(query *tgbotapi.CallbackQuery, page int) {
	// Проверяем права админа
	if !b.isUserAdmin(int64(query.From.ID)) {
		b.answerCallbackQuery(query.ID, "У вас нет прав администратора.")
		return
	}

	offset := page * CHATS_PER_PAGE
	chats, err := b.chatService.GetActiveChatsPaginated(CHATS_PER_PAGE, offset)
	if err != nil {
		b.answerCallbackQuery(query.ID, "Ошибка при получении чатов.")
		return
	}

	if len(chats) == 0 && page == 0 {
		msg := tgbotapi.NewMessage(query.Message.Chat.ID, "💬 Активных чатов нет.")
		b.api.Send(msg)
		b.answerCallbackQuery(query.ID, "")
		return
	}

	text := fmt.Sprintf("💬 Активные чаты (страница %d):\n\n", page+1)
	for _, chat := range chats {
		unreadBadge := ""
		if chat.UnreadCount > 0 {
			unreadBadge = fmt.Sprintf(" 🔴(%d)", chat.UnreadCount)
		}

		text += fmt.Sprintf("🔸 Чат #%d%s\n", chat.ID, unreadBadge)
		text += fmt.Sprintf("👤 %s\n", b.formatUserName(&chat.User))

		if chat.LastMessageAt != nil {
			text += fmt.Sprintf("📅 %s\n\n", chat.LastMessageAt.Format("02.01.2006 15:04"))
		} else {
			text += fmt.Sprintf("📅 %s\n\n", chat.CreatedAt.Format("02.01.2006 15:04"))
		}
	}

	// Создаем кнопки для каждого чата
	var buttons [][]tgbotapi.InlineKeyboardButton
	for _, chat := range chats {
		unreadBadge := ""
		if chat.UnreadCount > 0 {
			unreadBadge = " 🔴"
		}

		button := tgbotapi.NewInlineKeyboardButtonData(
			fmt.Sprintf("🔸 #%d%s - %s", chat.ID, unreadBadge, chat.User.FirstName),
			fmt.Sprintf("view_chat_%d", chat.ID),
		)
		buttons = append(buttons, tgbotapi.NewInlineKeyboardRow(button))
	}

	// Добавляем кнопки навигации
	var navButtons []tgbotapi.InlineKeyboardButton
	if page > 0 {
		navButtons = append(navButtons, tgbotapi.NewInlineKeyboardButtonData("⬅️ Назад", fmt.Sprintf("active_chats_page_%d", page-1)))
	}
	if len(chats) == CHATS_PER_PAGE {
		navButtons = append(navButtons, tgbotapi.NewInlineKeyboardButtonData("Вперед ➡️", fmt.Sprintf("active_chats_page_%d", page+1)))
	}
	if len(navButtons) > 0 {
		buttons = append(buttons, navButtons)
	}

	// Добавляем кнопку "Назад в админку"
	backButton := tgbotapi.NewInlineKeyboardButtonData("🔙 Назад в админку", "admin_menu")
	buttons = append(buttons, tgbotapi.NewInlineKeyboardRow(backButton))

	keyboard := tgbotapi.NewInlineKeyboardMarkup(buttons...)

	msg := tgbotapi.NewMessage(query.Message.Chat.ID, text)
	msg.ReplyMarkup = keyboard
	b.api.Send(msg)
	b.answerCallbackQuery(query.ID, "")
}

func (b *ChatBot) handleAdminArchivedChatsCallback(query *tgbotapi.CallbackQuery) {
	b.handleAdminArchivedChatsCallbackWithPage(query, 0)
}

func (b *ChatBot) handleAdminArchivedChatsCallbackWithPage(query *tgbotapi.CallbackQuery, page int) {
	// Проверяем права админа
	if !b.isUserAdmin(int64(query.From.ID)) {
		b.answerCallbackQuery(query.ID, "У вас нет прав администратора.")
		return
	}

	offset := page * CHATS_PER_PAGE
	chats, err := b.chatService.GetArchivedChatsPaginated(CHATS_PER_PAGE, offset)
	if err != nil {
		b.answerCallbackQuery(query.ID, "Ошибка при получении чатов.")
		return
	}

	if len(chats) == 0 && page == 0 {
		msg := tgbotapi.NewMessage(query.Message.Chat.ID, "📁 Архивированных чатов нет.")
		b.api.Send(msg)
		b.answerCallbackQuery(query.ID, "")
		return
	}

	text := fmt.Sprintf("📁 Архивированные чаты (страница %d):\n\n", page+1)
	for _, chat := range chats {
		text += fmt.Sprintf("🔸 Чат #%d\n", chat.ID)
		text += fmt.Sprintf("👤 %s\n", b.formatUserName(&chat.User))
		text += fmt.Sprintf("📅 %s\n\n", chat.UpdatedAt.Format("02.01.2006 15:04"))
	}

	// Создаем кнопки для каждого чата
	var buttons [][]tgbotapi.InlineKeyboardButton
	for _, chat := range chats {
		button := tgbotapi.NewInlineKeyboardButtonData(
			fmt.Sprintf("🔸 #%d - %s", chat.ID, chat.User.FirstName),
			fmt.Sprintf("view_chat_%d", chat.ID),
		)
		buttons = append(buttons, tgbotapi.NewInlineKeyboardRow(button))
	}

	// Добавляем кнопки навигации
	var navButtons []tgbotapi.InlineKeyboardButton
	if page > 0 {
		navButtons = append(navButtons, tgbotapi.NewInlineKeyboardButtonData("⬅️ Назад", fmt.Sprintf("archived_chats_page_%d", page-1)))
	}
	if len(chats) == CHATS_PER_PAGE {
		navButtons = append(navButtons, tgbotapi.NewInlineKeyboardButtonData("Вперед ➡️", fmt.Sprintf("archived_chats_page_%d", page+1)))
	}
	if len(navButtons) > 0 {
		buttons = append(buttons, navButtons)
	}

	// Добавляем кнопку "Назад в админку"
	backButton := tgbotapi.NewInlineKeyboardButtonData("🔙 Назад в админку", "admin_menu")
	buttons = append(buttons, tgbotapi.NewInlineKeyboardRow(backButton))

	keyboard := tgbotapi.NewInlineKeyboardMarkup(buttons...)

	msg := tgbotapi.NewMessage(query.Message.Chat.ID, text)
	msg.ReplyMarkup = keyboard
	b.api.Send(msg)
	b.answerCallbackQuery(query.ID, "")
}

func (b *ChatBot) handleAdminStatsCallback(query *tgbotapi.CallbackQuery) {
	// Проверяем права админа
	if !b.isUserAdmin(int64(query.From.ID)) {
		b.answerCallbackQuery(query.ID, "У вас нет прав администратора.")
		return
	}

	activeChats, _ := b.chatService.GetActiveChats()
	archivedChats, _ := b.chatService.GetArchivedChats()
	unreadCount, _ := b.chatService.GetUnreadChatsCount()

	text := "📊 Статистика чатов:\n\n"
	text += fmt.Sprintf("💬 Активные: %d\n", len(activeChats))
	text += fmt.Sprintf("📁 Архивированные: %d\n", len(archivedChats))
	text += fmt.Sprintf("🔴 Непрочитанные: %d\n", unreadCount)
	text += fmt.Sprintf("📈 Всего: %d\n", len(activeChats)+len(archivedChats))

	msg := tgbotapi.NewMessage(query.Message.Chat.ID, text)
	b.api.Send(msg)
	b.answerCallbackQuery(query.ID, "")
}

func (b *ChatBot) handleViewChatCallback(query *tgbotapi.CallbackQuery) {
	b.handleViewChatCallbackWithPage(query, 0)
}

func (b *ChatBot) handleViewChatCallbackWithPage(query *tgbotapi.CallbackQuery, page int) {
	// Проверяем права админа
	if !b.isUserAdmin(int64(query.From.ID)) {
		b.answerCallbackQuery(query.ID, "У вас нет прав администратора.")
		return
	}

	// Извлекаем ID чата
	parts := strings.Split(query.Data, "_")
	if len(parts) != 3 {
		return
	}

	chatID, err := strconv.ParseUint(parts[2], 10, 32)
	if err != nil {
		return
	}

	chat, err := b.chatService.GetChatByID(uint(chatID))
	if err != nil {
		b.answerCallbackQuery(query.ID, "Чат не найден.")
		return
	}

	// Помечаем чат как прочитанный
	b.chatService.MarkChatAsRead(uint(chatID))

	// Получаем сообщения с пагинацией
	offset := page * MESSAGES_PER_PAGE
	messages, err := b.chatService.GetChatMessagesPaginated(uint(chatID), MESSAGES_PER_PAGE, offset)
	if err != nil {
		b.answerCallbackQuery(query.ID, "Ошибка при получении сообщений.")
		return
	}

	// Получаем общее количество сообщений
	totalMessages, err := b.chatService.GetChatMessagesCount(uint(chatID))
	if err != nil {
		b.answerCallbackQuery(query.ID, "Ошибка при получении количества сообщений.")
		return
	}

	// Формируем текст с информацией о чате
	text := fmt.Sprintf("🔸 Чат #%d\n", chat.ID)
	text += fmt.Sprintf("👤 Пользователь: %s\n", b.formatUserName(&chat.User))
	text += fmt.Sprintf("📅 Создан: %s\n", chat.CreatedAt.Format("02.01.2006 15:04"))

	status := "💬 Активный"
	if chat.Status == models.ChatStatusArchived {
		status = "📁 Архивированный"
	}
	text += fmt.Sprintf("📊 Статус: %s\n", status)
	text += fmt.Sprintf("💬 Сообщений: %d\n\n", totalMessages)

	// Добавляем сообщения
	if len(messages) > 0 {
		text += fmt.Sprintf("💬 Сообщения (страница %d):\n", page+1)
		for _, message := range messages {
			sender := "👤 Клиент"
			if !message.IsFromUser {
				sender = "👨‍💼 Админ"
			}

			// Добавляем информацию о файлах если есть
			content := message.Content
			if len(message.Files) > 0 {
				content += " 📎"
			}

			text += fmt.Sprintf("%s: %s\n", sender, content)
			text += fmt.Sprintf("📅 %s\n\n", message.CreatedAt.Format("02.01.2006 15:04"))
		}
	} else {
		text += "💬 Сообщений пока нет.\n"
	}

	// Создаем кнопки
	var buttons [][]tgbotapi.InlineKeyboardButton

	// Кнопки навигации по сообщениям
	var navButtons []tgbotapi.InlineKeyboardButton
	if page > 0 {
		navButtons = append(navButtons, tgbotapi.NewInlineKeyboardButtonData("⬅️ Назад", fmt.Sprintf("view_chat_%d_page_%d", chatID, page-1)))
	}
	if int64(offset+MESSAGES_PER_PAGE) < totalMessages {
		navButtons = append(navButtons, tgbotapi.NewInlineKeyboardButtonData("Вперед ➡️", fmt.Sprintf("view_chat_%d_page_%d", chatID, page+1)))
	}
	if len(navButtons) > 0 {
		buttons = append(buttons, navButtons)
	}

	// Кнопка для детального просмотра истории
	historyButton := tgbotapi.NewInlineKeyboardButtonData(
		"📋 Детальная история",
		fmt.Sprintf("detailed_history_%d", chat.ID),
	)
	buttons = append(buttons, tgbotapi.NewInlineKeyboardRow(historyButton))

	if chat.Status == models.ChatStatusActive {
		replyButton := tgbotapi.NewInlineKeyboardButtonData(
			"💬 Ответить",
			fmt.Sprintf("admin_reply_%d", chat.ID),
		)
		buttons = append(buttons, tgbotapi.NewInlineKeyboardRow(replyButton))

		archiveButton := tgbotapi.NewInlineKeyboardButtonData(
			"📁 Архивировать",
			fmt.Sprintf("archive_chat_%d", chat.ID),
		)
		buttons = append(buttons, tgbotapi.NewInlineKeyboardRow(archiveButton))
	}

	// Определяем откуда пришли (активные или архивированные чаты)
	backButton := tgbotapi.NewInlineKeyboardButtonData(
		"🔙 Назад к чатам",
		"admin_active_chats",
	)
	if chat.Status == models.ChatStatusArchived {
		backButton = tgbotapi.NewInlineKeyboardButtonData(
			"🔙 Назад к чатам",
			"admin_archived_chats",
		)
	}
	buttons = append(buttons, tgbotapi.NewInlineKeyboardRow(backButton))

	keyboard := tgbotapi.NewInlineKeyboardMarkup(buttons...)

	msg := tgbotapi.NewMessage(query.Message.Chat.ID, text)
	msg.ReplyMarkup = keyboard
	b.api.Send(msg)
	b.answerCallbackQuery(query.ID, "")
}

func (b *ChatBot) handleAdminReplyCallback(query *tgbotapi.CallbackQuery) {
	// Проверяем права админа
	if !b.isUserAdmin(int64(query.From.ID)) {
		b.answerCallbackQuery(query.ID, "У вас нет прав администратора.")
		return
	}

	// Извлекаем ID чата
	parts := strings.Split(query.Data, "_")
	if len(parts) != 3 {
		return
	}

	chatID, err := strconv.ParseUint(parts[2], 10, 32)
	if err != nil {
		return
	}

	// Устанавливаем состояние ответа на чат
	b.stateManager.SetReplyingState(int64(query.From.ID), uint(chatID))

	text := fmt.Sprintf("💬 Ответ в чат #%d\n\nНапишите ваш ответ:", chatID)

	msg := tgbotapi.NewMessage(query.Message.Chat.ID, text)
	b.api.Send(msg)
	b.answerCallbackQuery(query.ID, "Напишите ответ в чат.")
}

func (b *ChatBot) handleArchiveChatCallback(query *tgbotapi.CallbackQuery) {
	// Проверяем права админа
	if !b.isUserAdmin(int64(query.From.ID)) {
		b.answerCallbackQuery(query.ID, "У вас нет прав администратора.")
		return
	}

	// Извлекаем ID чата
	parts := strings.Split(query.Data, "_")
	if len(parts) != 3 {
		return
	}

	chatID, err := strconv.ParseUint(parts[2], 10, 32)
	if err != nil {
		return
	}

	if err := b.chatService.ArchiveChat(uint(chatID)); err != nil {
		b.answerCallbackQuery(query.ID, "Ошибка при архивировании чата.")
		return
	}

	b.answerCallbackQuery(query.ID, "Чат архивирован.")
}

func (b *ChatBot) handleFinishConversationCallback(query *tgbotapi.CallbackQuery) {
	// Проверяем права админа
	if !b.isUserAdmin(int64(query.From.ID)) {
		b.answerCallbackQuery(query.ID, "У вас нет прав администратора.")
		return
	}

	// Извлекаем ID чата
	parts := strings.Split(query.Data, "_")
	if len(parts) != 3 {
		return
	}

	chatID, err := strconv.ParseUint(parts[2], 10, 32)
	if err != nil {
		return
	}

	// Архивируем чат
	if err := b.chatService.ArchiveChat(uint(chatID)); err != nil {
		b.answerCallbackQuery(query.ID, "Ошибка при архивировании чата.")
		return
	}

	// Очищаем состояние ответа
	b.stateManager.ClearUserState(int64(query.From.ID))

	// Возвращаемся в главное меню админа
	b.handleAdminCommand(&tgbotapi.Message{
		Chat: &tgbotapi.Chat{ID: query.Message.Chat.ID},
		From: query.From,
	}, &models.User{})

	b.answerCallbackQuery(query.ID, "Разговор завершен. Чат архивирован.")
}

func (b *ChatBot) handleContinueChatCallback(query *tgbotapi.CallbackQuery) {
	// Проверяем права админа
	if !b.isUserAdmin(int64(query.From.ID)) {
		b.answerCallbackQuery(query.ID, "У вас нет прав администратора.")
		return
	}

	// Просто подтверждаем, что админ может продолжать общение
	msg := tgbotapi.NewMessage(query.Message.Chat.ID, "💬 Продолжайте общение! Ваши сообщения будут отправляться в активный чат.")
	b.api.Send(msg)

	b.answerCallbackQuery(query.ID, "Продолжайте общение.")
}

func (b *ChatBot) handleViewChatPageCallback(query *tgbotapi.CallbackQuery) {
	// Извлекаем ID чата и номер страницы
	// Формат: view_chat_1_page_0
	parts := strings.Split(query.Data, "_")
	if len(parts) != 5 {
		return
	}

	chatID, err := strconv.ParseUint(parts[2], 10, 32)
	if err != nil {
		return
	}

	page, err := strconv.Atoi(parts[4])
	if err != nil {
		return
	}

	// Создаем новый callback query с правильным форматом
	newQuery := *query
	newQuery.Data = fmt.Sprintf("view_chat_%d", chatID)

	b.handleViewChatCallbackWithPage(&newQuery, page)
}

func (b *ChatBot) handleActiveChatsPageCallback(query *tgbotapi.CallbackQuery) {
	// Извлекаем номер страницы
	// Формат: active_chats_page_1
	parts := strings.Split(query.Data, "_")
	if len(parts) != 4 {
		return
	}

	page, err := strconv.Atoi(parts[3])
	if err != nil {
		return
	}

	b.handleAdminActiveChatsCallbackWithPage(query, page)
}

func (b *ChatBot) handleArchivedChatsPageCallback(query *tgbotapi.CallbackQuery) {
	// Извлекаем номер страницы
	// Формат: archived_chats_page_1
	parts := strings.Split(query.Data, "_")
	if len(parts) != 4 {
		return
	}

	page, err := strconv.Atoi(parts[3])
	if err != nil {
		return
	}

	b.handleAdminArchivedChatsCallbackWithPage(query, page)
}

func (b *ChatBot) handleAdminMenuCallback(query *tgbotapi.CallbackQuery) {
	// Возвращаемся в главное меню админа
	b.handleAdminCommand(&tgbotapi.Message{
		Chat: &tgbotapi.Chat{ID: query.Message.Chat.ID},
		From: query.From,
	}, &models.User{})

	b.answerCallbackQuery(query.ID, "Возврат в главное меню.")
}

func (b *ChatBot) handleDetailedHistoryCallback(query *tgbotapi.CallbackQuery) {
	// Проверяем права админа
	if !b.isUserAdmin(int64(query.From.ID)) {
		b.answerCallbackQuery(query.ID, "У вас нет прав администратора.")
		return
	}

	// Извлекаем ID чата
	parts := strings.Split(query.Data, "_")
	if len(parts) != 3 {
		return
	}

	chatID, err := strconv.ParseUint(parts[2], 10, 32)
	if err != nil {
		return
	}

	// Получаем все сообщения чата
	messages, err := b.chatService.GetChatMessagesPaginated(uint(chatID), 1000, 0) // Получаем все сообщения
	if err != nil {
		b.answerCallbackQuery(query.ID, "Ошибка при получении истории.")
		return
	}

	// Получаем информацию о чате
	chat, err := b.chatService.GetChatByID(uint(chatID))
	if err != nil {
		b.answerCallbackQuery(query.ID, "Чат не найден.")
		return
	}

	// Отправляем заголовок
	headerText := fmt.Sprintf("📋 Детальная история чата #%d\n👤 %s\n\n", chat.ID, b.formatUserName(&chat.User))
	msg := tgbotapi.NewMessage(query.Message.Chat.ID, headerText)
	b.api.Send(msg)

	// Отправляем каждое сообщение отдельно с улучшенным форматированием
	for _, message := range messages {
		var senderIcon, senderName string
		if message.IsFromUser {
			senderIcon = "👤"
			senderName = "КЛИЕНТ"
		} else {
			senderIcon = "👨‍💼"
			senderName = "АДМИН"
		}

		// Формируем информацию об отправителе
		senderInfo := fmt.Sprintf("%s %s (%s)", senderIcon, senderName, b.formatUserName(&message.User))

		// Отправляем медиа файлы если есть
		if len(message.Files) > 0 {
			for _, file := range message.Files {
				caption := fmt.Sprintf("%s\n📅 %s", senderInfo, message.CreatedAt.Format("02.01.2006 15:04:05"))

				switch file.FileType {
				case "photo":
					photoMsg := tgbotapi.NewPhoto(query.Message.Chat.ID, tgbotapi.FileID(file.FileID))
					photoMsg.Caption = caption
					b.api.Send(photoMsg)
				case "video":
					videoMsg := tgbotapi.NewVideo(query.Message.Chat.ID, tgbotapi.FileID(file.FileID))
					videoMsg.Caption = caption
					b.api.Send(videoMsg)
				case "document":
					docMsg := tgbotapi.NewDocument(query.Message.Chat.ID, tgbotapi.FileID(file.FileID))
					docMsg.Caption = caption
					b.api.Send(docMsg)
				case "voice":
					voiceMsg := tgbotapi.NewVoice(query.Message.Chat.ID, tgbotapi.FileID(file.FileID))
					voiceMsg.Caption = caption
					b.api.Send(voiceMsg)
				case "video_note":
					videoNoteMsg := tgbotapi.NewVideoNote(query.Message.Chat.ID, 0, tgbotapi.FileID(file.FileID))
					b.api.Send(videoNoteMsg)
				}
			}
		}

		// Отправляем текстовое сообщение если есть
		if message.Content != "" {
			messageText := fmt.Sprintf("%s:\n%s\n📅 %s", senderInfo, message.Content, message.CreatedAt.Format("02.01.2006 15:04:05"))
			msg := tgbotapi.NewMessage(query.Message.Chat.ID, messageText)
			b.api.Send(msg)
		} else if len(message.Files) == 0 {
			// Только если нет ни текста, ни файлов
			messageText := fmt.Sprintf("%s:\n[Сообщение без текста]\n📅 %s", senderInfo, message.CreatedAt.Format("02.01.2006 15:04:05"))
			msg := tgbotapi.NewMessage(query.Message.Chat.ID, messageText)
			b.api.Send(msg)
		}
	}

	// Отправляем кнопку возврата
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔙 Назад к чату", fmt.Sprintf("view_chat_%d", chatID)),
		),
	)

	backMsg := tgbotapi.NewMessage(query.Message.Chat.ID, "📋 История чата отправлена выше.")
	backMsg.ReplyMarkup = keyboard
	b.api.Send(backMsg)

	b.answerCallbackQuery(query.ID, "История отправлена.")
}

func (b *ChatBot) isUserAdmin(telegramID int64) bool {
	for _, adminID := range b.config.AdminIDs {
		if adminID == telegramID {
			return true
		}
	}
	return false
}

func (b *ChatBot) sendMessage(chatID int64, text string) {
	msg := tgbotapi.NewMessage(chatID, text)
	b.api.Send(msg)
}

// formatUserName форматирует имя пользователя с username
func (b *ChatBot) formatUserName(user *models.User) string {
	name := fmt.Sprintf("%s %s", user.FirstName, user.LastName)
	if user.Username != "" {
		name += fmt.Sprintf(" %s", user.Username)
	}
	return name
}

func (b *ChatBot) answerCallbackQuery(queryID, text string) {
	callback := tgbotapi.NewCallback(queryID, text)
	b.api.Request(callback)
}

func (b *ChatBot) GetBotUsername() string {
	return b.api.Self.UserName
}

// sendResponseToClient отправляет ответ админа клиенту
func (b *ChatBot) sendResponseToClient(chatID uint, messageText string, originalMessage *tgbotapi.Message) error {
	// Получаем чат
	chat, err := b.chatService.GetChatByID(chatID)
	if err != nil {
		return err
	}

	// Отправляем сообщение клиенту
	clientMsg := tgbotapi.NewMessage(int64(chat.User.TelegramID), "👨‍💼 Ответ от поддержки:\n\n"+messageText)

	// Если есть файлы в оригинальном сообщении, отправляем их тоже
	if originalMessage.Photo != nil && len(originalMessage.Photo) > 0 {
		photo := originalMessage.Photo[len(originalMessage.Photo)-1]
		photoMsg := tgbotapi.NewPhoto(int64(chat.User.TelegramID), tgbotapi.FileID(photo.FileID))
		photoMsg.Caption = "👨‍💼 Ответ от поддержки:\n\n" + messageText
		_, err = b.api.Send(photoMsg)
	} else if originalMessage.Video != nil {
		videoMsg := tgbotapi.NewVideo(int64(chat.User.TelegramID), tgbotapi.FileID(originalMessage.Video.FileID))
		videoMsg.Caption = "👨‍💼 Ответ от поддержки:\n\n" + messageText
		_, err = b.api.Send(videoMsg)
	} else if originalMessage.Document != nil {
		docMsg := tgbotapi.NewDocument(int64(chat.User.TelegramID), tgbotapi.FileID(originalMessage.Document.FileID))
		docMsg.Caption = "👨‍💼 Ответ от поддержки:\n\n" + messageText
		_, err = b.api.Send(docMsg)
	} else {
		_, err = b.api.Send(clientMsg)
	}

	return err
}

// showFinishConversationButton показывает кнопку "Закончить разговор"
func (b *ChatBot) showFinishConversationButton(adminChatID int64, chatID uint) {
	text := "✅ Ответ отправлен клиенту!\n\nВыберите действие:"

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("✅ Закончить разговор", fmt.Sprintf("finish_conversation_%d", chatID)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("💬 Продолжить общение", "continue_chat"),
		),
	)

	msg := tgbotapi.NewMessage(adminChatID, text)
	msg.ReplyMarkup = keyboard
	b.api.Send(msg)
}

// notifyAdminsAboutNewMessage отправляет уведомления админам о новом сообщении
func (b *ChatBot) notifyAdminsAboutNewMessage(chatID uint, user *models.User, messageText string) {
	// Получаем количество непрочитанных чатов
	unreadCount, _ := b.chatService.GetUnreadChatsCount()

	// Формируем текст уведомления
	notificationText := fmt.Sprintf("🔔 Новое сообщение!\n\n")
	notificationText += fmt.Sprintf("👤 От: %s\n", b.formatUserName(user))
	notificationText += fmt.Sprintf("💬 Чат #%d\n", chatID)
	notificationText += fmt.Sprintf("📝 Сообщение: %s\n\n", messageText)
	notificationText += fmt.Sprintf("📊 Непрочитанных чатов: %d", unreadCount)

	// Отправляем уведомление всем админам
	for _, adminID := range b.config.AdminIDs {
		msg := tgbotapi.NewMessage(adminID, notificationText)

		// Добавляем кнопку для быстрого перехода к чату
		keyboard := tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("💬 Открыть чат", fmt.Sprintf("view_chat_%d", chatID)),
			),
		)
		msg.ReplyMarkup = keyboard

		b.api.Send(msg)
	}
}

// sendMediaToAdmins отправляет медиа файлы от клиента всем админам
func (b *ChatBot) sendMediaToAdmins(chatID uint, user *models.User, message *tgbotapi.Message) error {
	// Отправляем медиа всем админам
	for _, adminID := range b.config.AdminIDs {
		// Определяем тип медиа и создаем соответствующее сообщение
		if message.Photo != nil && len(message.Photo) > 0 {
			photo := message.Photo[len(message.Photo)-1]
			photoMsg := tgbotapi.NewPhoto(adminID, tgbotapi.FileID(photo.FileID))

			// Формируем подпись с username
			caption := fmt.Sprintf("📷 Фото от %s", b.formatUserName(user))
			caption += fmt.Sprintf(" (Чат #%d)", chatID)

			// Добавляем подпись к фото если есть
			if message.Caption != "" {
				caption += fmt.Sprintf("\n\n%s", message.Caption)
			}

			photoMsg.Caption = caption
			b.api.Send(photoMsg)
		} else if message.Video != nil {
			videoMsg := tgbotapi.NewVideo(adminID, tgbotapi.FileID(message.Video.FileID))

			// Формируем подпись с username
			caption := fmt.Sprintf("🎥 Видео от %s", b.formatUserName(user))
			caption += fmt.Sprintf(" (Чат #%d)", chatID)

			// Добавляем подпись к видео если есть
			if message.Caption != "" {
				caption += fmt.Sprintf("\n\n%s", message.Caption)
			}

			videoMsg.Caption = caption
			b.api.Send(videoMsg)
		} else if message.Document != nil {
			docMsg := tgbotapi.NewDocument(adminID, tgbotapi.FileID(message.Document.FileID))

			// Формируем подпись с username
			caption := fmt.Sprintf("📄 Документ от %s", b.formatUserName(user))
			caption += fmt.Sprintf(" (Чат #%d)", chatID)

			// Добавляем подпись к документу если есть
			if message.Caption != "" {
				caption += fmt.Sprintf("\n\n%s", message.Caption)
			}

			docMsg.Caption = caption
			b.api.Send(docMsg)
		} else if message.Voice != nil {
			voiceMsg := tgbotapi.NewVoice(adminID, tgbotapi.FileID(message.Voice.FileID))

			// Формируем подпись с username
			caption := fmt.Sprintf("🎤 Голосовое от %s", b.formatUserName(user))
			caption += fmt.Sprintf(" (Чат #%d)", chatID)

			// Добавляем подпись к голосовому если есть
			if message.Caption != "" {
				caption += fmt.Sprintf("\n\n%s", message.Caption)
			}

			voiceMsg.Caption = caption
			b.api.Send(voiceMsg)
		} else if message.VideoNote != nil {
			videoNoteMsg := tgbotapi.NewVideoNote(adminID, 0, tgbotapi.FileID(message.VideoNote.FileID))
			b.api.Send(videoNoteMsg)
		}
	}

	return nil
}
