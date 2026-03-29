# Daily Email Sender

Go-приложение для отправки персонализированных мотивационных писем с тренировками и питанием. Пользователь регистрируется, задаёт расписание — и получает уникальные письма с AI-сгенерированным контентом.

**Статус:** Фазы 1–2 выполнены. Текущая — Фаза 3 (рефакторинг архитектуры). Полный план: `docs/plan.md`.

---

## Требования

- Go 1.21+
- PostgreSQL 17+
- SMTP-аккаунт (Yandex, Gmail, Microsoft)

---

## Установка

```bash
cp .env.example .env
# Отредактируйте .env: DATABASE_URL, SMTP_CONFIG
go mod download
go run ./cmd/server/ init-db
go build -o server ./cmd/server/
```

### Пример .env

```env
SMTP_CONFIG={"Host":"smtp.yandex.ru","Port":465,"User":"you@yandex.com","Password":"pass"}
EMAIL_CONFIG={"From":"you@yandex.com"}
DATABASE_URL="postgres://postgres:PASSWORD@localhost:5432/daily_email_sender?sslmode=disable&client_encoding=UTF-8"
```

---

## Использование

```bash
go run ./cmd/server/ add-user        # Добавить пользователя
go run ./cmd/server/ list-users      # Список пользователей
go run ./cmd/server/ add-schedule    # Задать расписание отправки
go run ./cmd/server/ run-scheduler   # Запустить планировщик
```

---

## Структура проекта

```
cmd/server/          — точка входа
internal/            — бизнес-логика
migrations/          — схема БД
docs/                — план разработки
```

---

## Лицензия

MIT
