#!/bin/bash

echo "🛑 Остановка Social Flow Support Bot"
echo "===================================="

# Останавливаем Docker контейнеры
echo "🗄️ Остановка базы данных..."
docker-compose down

echo "✅ Все сервисы остановлены"
