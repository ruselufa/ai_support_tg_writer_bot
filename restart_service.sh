#!/bin/bash

# Скрипт для перезапуска Support Bot через systemd

echo "🔄 Перезапуск Social Flow Support Bot"
echo "====================================="

# Перезапускаем сервис
echo "🔄 Перезапуск сервиса support-bot..."
sudo systemctl restart support-bot.service

# Проверяем статус
echo "📊 Статус сервиса:"
sudo systemctl status support-bot.service --no-pager

echo ""
echo "✅ Support Bot перезапущен!"
echo ""
echo "📋 Полезные команды:"
echo "   - Логи: sudo journalctl -u support-bot -f"
echo "   - Статус: sudo systemctl status support-bot"
echo "   - Остановить: ./stop_service.sh"
