.PHONY: all build test clean init-db run-scheduler deps add-user list-users add-schedule full-setup help

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
	@echo "  make build         - Собрать бинарный файл"
	@echo "  make deps          - Скачать зависимости"
	@echo "  make test          - Запустить тесты"
	@echo "  make test-coverage - Тесты с отчётом о покрытии"
	@echo "  make lint          - Запустить линтер"
	@echo "  make clean         - Удалить артефакты сборки"
	@echo ""

build:
	@echo "Сборка..."
	go build -o daily-email-sender.exe ./cmd/server/
	@echo "Готово: daily-email-sender.exe"

deps:
	@echo "Скачивание зависимостей..."
	go mod download
	@echo "Готово"

clean:
	@echo "Очистка..."
	rm -f daily-email-sender.exe
	@echo "Готово"

test:
	@echo "Запуск тестов..."
	go test ./... -v
	@echo "Тесты завершены"

test-coverage:
	@echo "Запуск тестов с покрытием..."
	go test ./... -coverprofile=coverage.out
	go tool cover -html=coverage.out -o coverage.html
	@echo "Отчёт: coverage.html"

lint:
	golangci-lint run ./...

init-db:
	@echo "Инициализация базы данных..."
	go run ./cmd/server/ init-db
	@echo "База данных инициализирована"

run-scheduler:
	@echo "Запуск планировщика..."
	go run ./cmd/server/ run-scheduler

add-user:
	@echo "Добавление пользователя..."
	go run ./cmd/server/ add-user

list-users:
	@echo "Список пользователей..."
	go run ./cmd/server/ list-users

add-schedule:
	@echo "Добавление расписания..."
	go run ./cmd/server/ add-schedule

full-setup:
	@echo "Полная настройка..."
	$(MAKE) add-user
