.PHONY: all run build build-linux test clean init-db run-scheduler deps add-user list-users add-schedule full-setup help docker-build docker-up docker-down docker-logs docker-ps docker-clean docker-init-db docker-add-user docker-shell docker-restart docker-up-dev

# По умолчанию показываем справку
.DEFAULT_GOAL := help

help:
	@echo ""
	@echo "========================================"
	@echo "  Daily Email Sender - Makefile Commands"
	@echo "========================================"
	@echo ""
	@echo "  make init-db       - Инициализировать базу данных"
	@echo "  make run-scheduler - Запустить планировщик"
	@echo "  make add-user      - Добавить пользователя через CLI"
	@echo "  make list-users    - Показать список пользователей"
	@echo "  make add-schedule  - Добавить расписание через CLI"
	@echo "  make full-setup    - Полный цикл создания пользователя"
	@echo "  make build         - Собрать бинарный файл (Windows)"
	@echo "  make build-linux   - Собрать Linux-бинарник для Docker"
	@echo "  make deps          - Скачать зависимости"
	@echo "  make test          - Запустить тесты"
	@echo "  make test-coverage - Тесты с отчётом о покрытии"
	@echo "  make lint          - Запустить линтер"
	@echo "  make clean         - Удалить артефакты сборки"
	@echo ""

# Собрать бинарный файл (Windows)
build:
	@echo "Сборка..."
	go build -o daily-email-sender.exe ./cmd/server/
	@echo "Готово: daily-email-sender.exe"

# Собрать Linux-бинарник для Docker dev
build-linux:
	@echo "Сборка для Linux..."
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o daily-email-sender-linux ./cmd/server/
	@echo "Готово: daily-email-sender-linux"

# Скачать зависимости
deps:
	@echo "Скачивание зависимостей..."
	go mod download
	@echo "Готово"

# Удалить артефакты
clean:
	@echo "Очистка..."
	rm -f daily-email-sender.exe daily-email-sender-linux
	@echo "Готово"

# Запустить тесты
test:
	@echo "Запуск тестов..."
	go test ./... -v
	@echo "Тесты завершены"

# Тесты с покрытием
test-coverage:
	@echo "Запуск тестов с покрытием..."
	go test ./... -coverprofile=coverage.out
	go tool cover -html=coverage.out -o coverage.html
	@echo "Отчёт: coverage.html"

# Линтер
lint:
	golangci-lint run ./...

# Инициализировать БД
init-db:
	@echo "Инициализация базы данных..."
	go run ./cmd/server/ init-db
	@echo "База данных инициализирована"

# Запустить планировщик
run-scheduler:
	@echo "Запуск планировщика..."
	go run ./cmd/server/ run-scheduler

# Добавить пользователя
add-user:
	@echo "Добавление пользователя..."
	go run ./cmd/server/ add-user

# Показать список пользователей
list-users:
	@echo "Список пользователей..."
	go run ./cmd/server/ list-users

# Добавить расписание
add-schedule:
	@echo "Добавление расписания..."
	go run ./cmd/server/ add-schedule

# Полный цикл создания
full-setup:
	@echo "Полная настройка..."
	$(MAKE) add-user

# Docker
docker-build:
	@echo "Сборка Docker-образов..."
	docker-compose build
	@echo "Готово!"

docker-up:
	@echo "Запуск контейнеров..."
	docker-compose up -d
	@echo "Контейнеры запущены! PostgreSQL доступен на localhost:5432"

docker-down:
	@echo "Остановка контейнеров..."
	docker-compose down
	@echo "Контейнеры остановлены!"

docker-logs:
	docker-compose logs -f

docker-ps:
	docker-compose ps

docker-clean:
	@echo "Очистка контейнеров и томов..."
	docker-compose down -v
	@echo "Готово!"

docker-init-db:
	@echo "Инициализация БД в контейнере..."
	docker-compose exec app ./daily-email-sender init-db
	@echo "БД инициализирована!"

docker-add-user:
	@echo "Добавление пользователя..."
	docker-compose exec app ./daily-email-sender add-user

docker-shell:
	docker-compose exec app sh

docker-restart:
	@echo "Перезапуск контейнеров..."
	docker-compose restart
	@echo "Готово!"

docker-up-dev:
	@echo "Запуск в режиме разработки..."
	docker-compose -f docker-compose.dev.yml up -d
	@echo "Контейнеры запущены в dev-режиме!"
