# Single-stage Dockerfile - сборка внутри контейнера
FROM golang:1.22-alpine

WORKDIR /app

# ca-certificates для HTTPS-соединений
RUN apk --no-cache add ca-certificates tzdata

# Копируем модули и скачиваем зависимости
COPY go.mod go.sum ./
RUN go mod download

# Копируем исходный код
COPY . .

# Статическая сборка
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o daily-email-sender ./cmd/server/

# Безопасный пользователь
RUN addgroup -g 1000 appuser && \
    adduser -D -u 1000 -G appuser appuser

USER appuser

# По умолчанию запускаем планировщик
CMD ["./daily-email-sender", "run-scheduler"]
