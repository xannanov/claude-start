# Daily Email Sender

Go-приложение для отправки персонализированных мотивационных писем с тренировками и питанием. Пользователь регистрируется, задаёт расписание — и получает уникальные письма с AI-сгенерированным контентом.

**Статус:** Фазы 1–2 выполнены. Текущая — Фаза 3 (рефакторинг архитектуры). Полный план: `plan.md`.

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
make deps
make init-db
make build
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
make add-user        # Добавить пользователя
make list-users      # Список пользователей
make add-schedule    # Задать расписание отправки
make run-scheduler   # Запустить планировщик
```

---

## Docker

```bash
cp .env.example .env   # Настройте SMTP
make docker-build
make docker-up
make docker-add-user
```

| Команда | Описание |
|---------|----------|
| `make docker-up` | Запустить контейнеры |
| `make docker-down` | Остановить |
| `make docker-logs` | Логи в реальном времени |
| `make docker-shell` | Shell в контейнере приложения |
| `make docker-clean` | Удалить контейнеры и volumes |

---

## Структура проекта

```
main.go              — точка входа
cli.go               — CLI-команды
db.go                — работа с базой данных
scheduler.go         — планировщик отправок
email_templates.go   — HTML-шаблоны писем
models.go            — модели данных
schema.sql           — схема БД
```

> После Фазы 3 код переедет в `cmd/` и `internal/` (см. `plan.md`).

---

## Лицензия

MIT
