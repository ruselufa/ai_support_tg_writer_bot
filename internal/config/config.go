package config

import (
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	TelegramBotToken   string
	TelegramWebhookURL string
	ServerPort         string
	WebhookSecret      string
	AdminIDs           []int64
	Database           DatabaseConfig
	Redis              RedisConfig
	EnableWebAdmin     bool
}

type DatabaseConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	Name     string
}

type RedisConfig struct {
	Host     string
	Port     string
	Password string
}

func Load() (*Config, error) {
	// Загружаем .env файл если он существует
	_ = godotenv.Load()

	adminIDsStr := os.Getenv("ADMIN_IDS")
	var adminIDs []int64
	if adminIDsStr != "" {
		for _, idStr := range strings.Split(adminIDsStr, ",") {
			if id, err := strconv.ParseInt(strings.TrimSpace(idStr), 10, 64); err == nil {
				adminIDs = append(adminIDs, id)
			}
		}
	}

	return &Config{
		TelegramBotToken:   os.Getenv("TELEGRAM_BOT_TOKEN"),
		TelegramWebhookURL: os.Getenv("TELEGRAM_WEBHOOK_URL"),
		ServerPort:         getEnv("SERVER_PORT", "8080"),
		WebhookSecret:      os.Getenv("WEBHOOK_SECRET"),
		AdminIDs:           adminIDs,
		EnableWebAdmin:     getEnv("ENABLE_WEB_ADMIN", "false") == "true",
		Database: DatabaseConfig{
			Host:     getEnv("DB_HOST", "localhost"),
			Port:     getEnv("DB_PORT", "6432"),
			User:     getEnv("DB_USER", "postgres"),
			Password: getEnv("DB_PASSWORD", "postgres"),
			Name:     getEnv("DB_NAME", "support_bot"),
		},
		Redis: RedisConfig{
			Host:     getEnv("REDIS_HOST", "localhost"),
			Port:     getEnv("REDIS_PORT", "6379"),
			Password: getEnv("REDIS_PASSWORD", ""),
		},
	}, nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
