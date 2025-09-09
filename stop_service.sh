#!/bin/bash

# Скрипт для остановки Support Bot через systemd

echo "🛑 Остановка Social Flow Support Bot"
echo "===================================="

# Останавливаем сервис
echo "🛑 Остановка сервиса support-bot..."
sudo systemctl stop support-bot.service

# Останавливаем базу данных (опционально)
echo "🗄️ Остановка базы данных..."
cd /root/apps/ai_support_tg_writer_bot
docker-compose down

echo "✅ Support Bot остановлен!"
echo ""
echo "ℹ️  База данных также остановлена"
echo "   Для запуска только базы данных: docker-compose up -d"
