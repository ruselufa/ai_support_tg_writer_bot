package main

import (
	"ai_support_tg_writer_bot/internal/bot"
	"ai_support_tg_writer_bot/internal/config"
	"ai_support_tg_writer_bot/internal/database"
	"ai_support_tg_writer_bot/internal/repository"
	"ai_support_tg_writer_bot/internal/service"
	"log"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	// Загружаем конфигурацию
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Проверяем обязательные параметры
	if cfg.TelegramBotToken == "" {
		log.Fatal("TELEGRAM_BOT_TOKEN is required")
	}

	// Подключаемся к базе данных
	db, err := database.Connect(cfg)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// Инициализируем репозитории
	userRepo := repository.NewUserRepository(db)
	chatRepo := repository.NewChatRepository(db)
	chatMessageRepo := repository.NewChatMessageRepository(db)
	fileRepo := repository.NewFileRepository(db)

	// Инициализируем сервисы
	userService := service.NewUserService(userRepo)
	chatService := service.NewChatService(chatRepo, chatMessageRepo, fileRepo)
	fileService := service.NewFileService(fileRepo)

	// Инициализируем Telegram бота
	telegramBot, err := bot.NewChatBot(cfg, userService, chatService, fileService)
	if err != nil {
		log.Fatalf("Failed to create bot: %v", err)
	}

	// Инициализируем веб-сервер (временно отключен)
	// webServer := web.NewServer(cfg, userService, chatService, fileService)

	// Канал для graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// Веб-сервер временно отключен
	log.Println("Web admin panel disabled. Use /admin command in Telegram for admin functions.")

	// Запускаем Telegram бота в горутине
	go func() {
		log.Println("Starting Telegram bot...")
		if err := telegramBot.Start(); err != nil {
			log.Printf("Telegram bot error: %v", err)
		}
	}()

	log.Println("🚀 Social Flow Support Bot started successfully!")
	log.Printf("📱 Telegram bot: @%s", telegramBot.GetBotUsername())
	log.Println("👨‍💼 Admin panel: Use /admin command in Telegram")
	log.Println("Press Ctrl+C to stop...")

	// Ждем сигнал завершения
	<-quit
	log.Println("Shutting down...")
}
