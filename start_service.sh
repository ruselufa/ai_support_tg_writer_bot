#!/bin/bash

# Скрипт для запуска Support Bot через systemd

echo "🚀 Запуск Social Flow Support Bot через systemd"
echo "=============================================="

# Сначала запускаем базу данных
echo "🗄️ Запуск базы данных..."
cd /root/apps/ai_support_tg_writer_bot
docker-compose up -d

# Ждем пока база данных запустится
echo "⏳ Ожидание запуска базы данных..."
sleep 10

# Запускаем сервис
echo "🚀 Запуск сервиса support-bot..."
sudo systemctl start support-bot.service

# Проверяем статус
echo "📊 Статус сервиса:"
sudo systemctl status support-bot.service --no-pager

echo ""
echo "✅ Support Bot запущен!"
echo ""
echo "🌐 Доступные сервисы:"
echo "   - Support Bot: http://localhost:8080"
echo "   - PostgreSQL: localhost:6432"
echo "   - Redis: localhost:6389"
echo ""
echo "📱 Telegram бот: используйте токен из TELEGRAM_BOT_TOKEN"
echo ""
echo "🛠️ Управление сервисом:"
echo "   - Остановить: ./stop_service.sh"
echo "   - Перезапустить: ./restart_service.sh"
echo "   - Логи: sudo journalctl -u support-bot -f"
