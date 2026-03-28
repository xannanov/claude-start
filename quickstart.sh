#!/bin/bash

echo "=== Daily Email Sender - Быстрый старт ==="
echo ""

if ! command -v go &> /dev/null; then
    echo "Go не установлен. Скачайте с https://go.dev/dl/"
    exit 1
fi

echo "Go установлен"

echo "Установка зависимостей..."
go mod download

if [ ! -f .env ]; then
    echo "Файл .env не найден — скопируйте .env.example и заполните настройки"
    exit 1
fi

echo ""
echo "Запустите:"
echo "  make init-db       - инициализировать БД"
echo "  make add-user      - добавить пользователя"
echo "  make run-scheduler - запустить планировщик"
echo ""
read -p "Press enter to exit..."
