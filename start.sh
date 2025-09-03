#!/bin/bash

echo "🚀 Запуск Social Flow Support Bot"
echo "================================="

# Проверяем наличие .env файла
if [ ! -f .env ]; then
    echo "❌ Файл .env не найден!"
    echo "📝 Скопируйте env.example в .env и заполните необходимые параметры:"
    echo "   cp env.example .env"
    echo "   nano .env"
    exit 1
fi

# Проверяем наличие Docker
if ! command -v docker &> /dev/null; then
    echo "❌ Docker не установлен!"
    echo "📦 Установите Docker: https://docs.docker.com/get-docker/"
    exit 1
fi

# Проверяем наличие Docker Compose
if ! command -v docker-compose &> /dev/null; then
    echo "❌ Docker Compose не установлен!"
    echo "📦 Установите Docker Compose: https://docs.docker.com/compose/install/"
    exit 1
fi

# Проверяем наличие Go
if ! command -v go &> /dev/null; then
    echo "❌ Go не установлен!"
    echo "📦 Установите Go: https://golang.org/doc/install"
    exit 1
fi

echo "✅ Все зависимости найдены"

# Запускаем базу данных
echo "🗄️ Запуск базы данных..."
docker-compose up -d

# Ждем пока база данных запустится
echo "⏳ Ожидание запуска PostgreSQL..."
sleep 5

# Проверяем что база данных запустилась
if ! docker-compose ps | grep -q "Up"; then
    echo "❌ Ошибка запуска базы данных!"
    echo "🔍 Проверьте логи: docker-compose logs"
    exit 1
fi

echo "✅ База данных запущена"

# Устанавливаем зависимости
echo "📦 Установка зависимостей..."
go mod tidy

# Запускаем приложение
echo "🚀 Запуск бота..."
echo "================================="
echo "📱 Админка: http://localhost:8080"
echo "🛑 Для остановки нажмите Ctrl+C"
echo "================================="

go run main.go
