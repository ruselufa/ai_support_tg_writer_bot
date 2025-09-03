# Social Flow Support Bot

Telegram бот для технической поддержки проекта Social Flow с админской панелью.

## Возможности

### Для клиентов:
- 🤖 Создание тикетов поддержки
- 📝 Отправка вопросов и отзывов
- 📎 Прикрепление скриншотов и видео
- 💬 Чат с администраторами
- 📋 Просмотр истории своих тикетов

### Для администраторов:
- 🌐 Веб-панель управления
- 📊 Статистика тикетов
- 💬 Ответы на сообщения клиентов
- 🎯 Фильтрация по статусам
- ✅ Закрытие тикетов

## Архитектура

Проект использует многослойную архитектуру:

```
├── internal/
│   ├── bot/          # Telegram бот
│   ├── config/       # Конфигурация
│   ├── database/     # Подключение к БД
│   ├── models/       # Модели данных
│   ├── repository/   # Слой доступа к данным
│   ├── service/      # Бизнес-логика
│   └── web/          # Веб-сервер и API
├── web/
│   ├── static/       # Статические файлы
│   └── templates/    # HTML шаблоны
└── main.go           # Точка входа
```

## Установка и запуск

### 1. Клонирование репозитория

```bash
git clone <repository-url>
cd ai_support_tg_writer_bot
```

### 2. Настройка окружения

Скопируйте файл `env.example` в `.env` и заполните необходимые параметры:

```bash
cp env.example .env
```

Отредактируйте `.env` файл:

```env
# Telegram Bot Configuration
TELEGRAM_BOT_TOKEN=1234567890:ABCdefGHIjklMNOpqrsTUVwxyz
TELEGRAM_WEBHOOK_URL=https://yourdomain.com/webhook

# Database Configuration
DB_HOST=localhost
DB_PORT=6432
DB_USER=postgres
DB_PASSWORD=postgres
DB_NAME=support_bot

# Admin Configuration (замените на ваши Telegram ID)
ADMIN_IDS=123456789,987654321

# Server Configuration
SERVER_PORT=8080
WEBHOOK_SECRET=your_webhook_secret_here

# Redis Configuration
REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_PASSWORD=
```

**📋 Подробная инструкция по настройке:** См. файл [SETUP.md](SETUP.md)

### 3. Запуск базы данных

```bash
docker-compose up -d
```

### 4. Установка зависимостей

```bash
go mod tidy
```

### 5. Запуск приложения

**Быстрый запуск:**
```bash
./start.sh
```

**Или вручную:**
```bash
go run main.go
```

## Создание Telegram бота

1. Найдите [@BotFather](https://t.me/botfather) в Telegram
2. Отправьте команду `/newbot`
3. Следуйте инструкциям для создания бота
4. Скопируйте полученный токен в файл `.env`

## Настройка администраторов

1. Узнайте Telegram ID пользователей, которые должны быть администраторами
2. Добавьте их ID в переменную `ADMIN_IDS` в файле `.env` (через запятую)

## Использование

### Для клиентов:

1. Найдите вашего бота в Telegram
2. Отправьте команду `/start`
3. Нажмите "Создать тикет" или просто напишите ваш вопрос
4. При необходимости прикрепите скриншоты или видео
5. Дождитесь ответа от администратора

### Для администраторов:

1. Откройте админскую панель: `http://localhost:8080`
2. Используйте заголовок `X-Admin-ID` с вашим Telegram ID для авторизации
3. Просматривайте тикеты в разделе "Открытые"
4. Отвечайте на сообщения клиентов
5. Закрывайте тикеты после решения проблем

## API Endpoints

### Админские маршруты (требуют заголовок X-Admin-ID):

- `GET /api/v1/admin/dashboard` - Статистика
- `GET /api/v1/admin/tickets` - Список тикетов
- `GET /api/v1/admin/tickets/:id` - Детали тикета
- `POST /api/v1/admin/tickets/:id/reply` - Ответ на тикет
- `POST /api/v1/admin/tickets/:id/close` - Закрытие тикета
- `GET /api/v1/admin/stats` - Статистика

## Структура базы данных

### Таблицы:

- `users` - Пользователи Telegram
- `tickets` - Тикеты поддержки
- `messages` - Сообщения в тикетах
- `files` - Прикрепленные файлы

## Разработка

### Добавление новых функций:

1. Создайте модель в `internal/models/`
2. Добавьте репозиторий в `internal/repository/`
3. Создайте сервис в `internal/service/`
4. Обновите обработчики в `internal/bot/` или `internal/web/`

### Тестирование:

```bash
go test ./...
```

## Деплой

### Docker:

```bash
# Сборка образа
docker build -t social-flow-support-bot .

# Запуск
docker run -d --env-file .env social-flow-support-bot
```

### Systemd (Linux):

Создайте файл `/etc/systemd/system/social-flow-bot.service`:

```ini
[Unit]
Description=Social Flow Support Bot
After=network.target

[Service]
Type=simple
User=your-user
WorkingDirectory=/path/to/project
ExecStart=/path/to/project/social-flow-bot
Restart=always
EnvironmentFile=/path/to/project/.env

[Install]
WantedBy=multi-user.target
```

## Лицензия

MIT License

## Поддержка

Если у вас возникли вопросы или проблемы, создайте issue в репозитории.
