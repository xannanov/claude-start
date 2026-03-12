#!/bin/bash

echo "=== Daily Email Sender - Быстрый старт ==="
echo ""

# Проверка, установлен ли Go
if ! command -v go &> /dev/null; then
    echo "❌ Go не установлен. Скачайте с https://go.dev/dl/"
    exit 1
fi

echo "✅ Go установлен"

# Установка зависимостей
echo "📦 Установка зависимостей..."
go mod download

# Проверка .env файла
if [ ! -f .env ]; then
    echo "⚠️  Файл .env не найден"
    echo "📋 Создал файл .env с примерами настроек"
fi

echo ""
echo "📝 Настройте email в файле .env:"
echo "   EMAIL_FROM=your-email@gmail.com"
echo "   EMAIL_PASSWORD=your-app-password"
echo "   EMAIL_TO=target-email@example.com"
echo ""
echo "💡 Для Gmail:"
echo "   1. Включите 2FA в Google Account"
echo "   2. Создайте App Password: Security > App Passwords"
echo "   3. Используйте полученный пароль (без пробелов)"
echo ""
echo "🚀 Запустите: make run"
