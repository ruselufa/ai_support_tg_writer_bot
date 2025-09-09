# Support Bot - Systemd + Docker Setup

## Архитектура

- **Docker контейнеры** (PostgreSQL + Redis) - автозапуск при перезагрузке системы
- **Systemd сервис** - основное приложение бота с автозапуском

## Быстрый запуск

1. **Создайте файл .env** с вашими настройками:
```bash
cp env.example .env
nano .env
```

2. **Запустите через systemd:**
```bash
./start_service.sh
```

3. **Остановите:**
```bash
./stop_service.sh
```

## Управление сервисом

```bash
# Запуск
./start_service.sh

# Остановка
./stop_service.sh

# Перезапуск
./restart_service.sh

# Статус
sudo systemctl status support-bot

# Логи
sudo journalctl -u support-bot -f
```

## Управление базой данных

```bash
# Запуск только базы данных
docker-compose up -d

# Остановка базы данных
docker-compose down

# Статус контейнеров
docker-compose ps
```

## Доступные сервисы

- **Support Bot**: http://localhost:8080
- **PostgreSQL**: localhost:6432
- **Redis**: localhost:6389

## Автозапуск

- **Docker контейнеры** настроены на `restart: always`
- **Systemd сервис** включен в автозапуск (`systemctl enable support-bot`)

## Логи

```bash
# Логи приложения
sudo journalctl -u support-bot -f

# Логи базы данных
docker-compose logs -f postgres

# Логи Redis
docker-compose logs -f redis
```

## Перезагрузка системы

После перезагрузки системы:
1. Docker контейнеры (PostgreSQL + Redis) запустятся автоматически
2. Systemd сервис support-bot запустится автоматически
3. Приложение подключится к базе данных и начнет работу
