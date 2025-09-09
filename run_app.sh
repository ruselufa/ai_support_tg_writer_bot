#!/bin/bash

# Скрипт для запуска Support Bot приложения

# Переходим в директорию приложения
cd /root/apps/ai_support_tg_writer_bot

# Проверяем наличие .env файла
if [ ! -f .env ]; then
    echo "❌ Файл .env не найден!"
    echo "📝 Создайте файл .env на основе env.example"
    exit 1
fi

# Загружаем переменные окружения
export $(cat .env | grep -v '^#' | xargs)

# Проверяем что база данных запущена
echo "🔍 Проверка подключения к базе данных..."
if ! nc -z localhost 6432; then
    echo "❌ База данных не доступна на порту 6432"
    echo "🗄️ Запускаем базу данных..."
    docker-compose up -d postgres redis
    sleep 10
fi

# Проверяем что Redis запущен
if ! nc -z localhost 6389; then
    echo "❌ Redis не доступен на порту 6389"
    echo "🗄️ Запускаем Redis..."
    docker-compose up -d redis
    sleep 5
fi

echo "✅ База данных и Redis доступны"

# Устанавливаем зависимости
echo "📦 Установка зависимостей..."
go mod tidy

# Запускаем приложение
echo "🚀 Запуск Support Bot..."
exec go run main.go
