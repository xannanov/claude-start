# Single-stage Dockerfile - сборка происходит внутри контейнера
FROM golang:1.21-alpine

WORKDIR /app

# Установить ca-certificates для HTTPS
RUN apk --no-cache add ca-certificates

# Копировать go модули
COPY go.mod go.sum* ./
RUN go mod download

# Копировать весь исходный код
COPY . .

# Сборка приложения внутри контейнера
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o daily-email-sender .

# Создать пользователя для безопасности
RUN addgroup -g 1000 appuser && \
    adduser -D -u 1000 -G appuser appuser

USER appuser

# Установить переменные окружения по умолчанию
ENV DATABASE_URL=postgres://postgres:admin@postgres:5432/daily_email_sender?sslmode=disable&client_encoding=UTF-8
ENV SMTP_CONFIG='{"Host":"smtp.yandex.ru","Port":465,"User":"your-email@yandex.com","Password":"your-password"}'
ENV EMAIL_CONFIG='{"From":"sender@example.com","To":"recipient@example.com"}'

# По умолчанию запускаем scheduler
CMD ["./daily-email-sender", "run-scheduler"]
