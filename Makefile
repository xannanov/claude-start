.PHONY: run build test clean

# Запуск приложения
run:
	go run main.go

# Сборка исполняемого файла
build:
	go build -o daily-email-sender main.go

# Установка зависимостей
deps:
	go mod download

# Очистка
clean:
	rm -f daily-email-sender

# Тест (просто проверка компиляции)
test:
	go build -o /dev/null main.go
