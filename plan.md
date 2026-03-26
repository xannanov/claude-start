# План доведения проекта до релиза

> Цель: рабочий продукт на ~1000 пользователей/месяц.
> Таймзона: Moscow (UTC+3). Язык: русский. SMTP: Yandex. Авторизация: логин + пароль.
> AI-персонализация: DeepSeek API. Защита от ботов: rate limit + CSRF.

---

## Фаза 1. Чистка — удаление мусора, дубликатов, мёртвого кода

### 1.1 Удалить файл `make` (пустой, 0 байт — создан случайно)

### 1.2 Удалить дубликаты в Makefile
В `Makefile` строки ~150-192 — **точная копия** строк ~105-148 (все docker-* таргеты продублированы). Удалить вторую копию.

### 1.3 Удалить мёртвый код
- Функция `getNextRuns()` в `scheduler.go:178-220` — определена, но нигде не вызывается.
- Поле `Message.Time` в `scheduler.go:37` — устанавливается, но не используется по назначению.
- `EmailConfig.To` в `.env` — читается из конфига, но полностью игнорируется (перезаписывается `toEmail` из БД). Убрать из `.env.example` и из кода чтения конфига, либо задокументировать что это fallback.

### 1.4 Убрать захардкоженный email
Убрать `voda2600@gmail.com` из `schema.sql` — это тестовый email, не должен быть в схеме. Дефолтный INSERT пользователя удалить полностью.

### 1.5 Поле `password_hash`
Сейчас оно в схеме, но не используется. **Не удаляем** — оно понадобится в Фазе 5 (авторизация). Но убираем из модели до тех пор, пока авторизация не будет реализована, чтобы не путать.

### 1.6 `WorkoutPreferences` (JSONB)
Поле в модели `User` — всегда пустой map, нигде не заполняется. Удалить из Go-модели и из `CREATE TABLE`. Если понадобится потом — добавим миграцией.

---

## Фаза 2. Фикс критических багов

### 2.1 Day-of-week сдвиг (КРИТИЧНО — scheduler не работает)
**Файл:** `scheduler.go:117`
**Баг:** `int(now.Weekday())` в Go возвращает 0=Sunday, а в БД и CLI подразумевается 0=Monday.
**Результат:** письма уходят не в тот день недели.

**Фикс:** Заменить на:
```go
moscowTime := now.In(moscowTZ)
dayOfWeek := (int(moscowTime.Weekday()) + 6) % 7 // 0=Пн, 6=Вс
```
Применить эту же формулу **везде**, где используется `Weekday()`:
- `scheduler.go:117` — основной цикл проверки
- `scheduler.go:227` — `displayNextRuns()`
- `scheduler.go:262-273` — расчёт daysUntil

### 2.2 DB connection закрывается до работы scheduler (КРИТИЧНО)
**Файл:** `main.go:88-91`
**Баг:** `defer CloseDatabase()` срабатывает при выходе из функции `runScheduler()`, а scheduler крутится в горутине. Все запросы к БД падают.

**Фикс:** Убрать `defer CloseDatabase()`. Закрывать соединение в обработчике сигналов (SIGINT/SIGTERM), **после** остановки scheduler:
```go
scheduler.Stop()
CloseDatabase()
```

### 2.3 Фикс schema.sql
**Баг:** `OR (SELECT COUNT(*) = 0)` — всегда false, дефолтный INSERT не работает.
**Фикс:** Удалить весь блок INSERT с захардкоженным пользователем (см. 1.4). Оставить только CREATE TABLE + индексы.

### 2.4 Московское время во всём scheduler
**Файл:** `scheduler.go:114`
**Баг:** `time.Now()` использует серверный часовой пояс.

**Фикс:** Загружать timezone один раз при старте:
```go
var moscowTZ *time.Location

func init() {
    moscowTZ, _ = time.LoadLocation("Europe/Moscow")
}
```
Использовать `time.Now().In(moscowTZ)` во всех местах, где сравнивается время с расписанием.

---

## Фаза 3. Рефакторинг архитектуры

### 3.1 Убрать глобальную переменную `db`
Сейчас `var db *sql.DB` — глобальная в `db.go`. Это создаёт проблемы с конкурентностью и тестированием.

**Фикс:** Создать структуру:
```go
type Store struct {
    db *sql.DB
}

func NewStore(databaseURL string) (*Store, error) { ... }
func (s *Store) GetUserByID(id string) (*User, error) { ... }
// и т.д.
```
Передавать `*Store` в scheduler, CLI и другие компоненты.

### 3.2 Разбить на пакеты
Текущая структура — всё в `package main`. Разбить на:
```
cmd/
  server/
    main.go           — точка входа
internal/
  config/
    config.go         — загрузка и валидация конфигурации
  database/
    store.go          — Store + все DB операции
    migrations/       — SQL миграции
  scheduler/
    scheduler.go      — логика расписания
  email/
    sender.go         — SMTP отправка
    templates.go      — HTML шаблоны
  models/
    user.go           — User, UserSchedule
    message.go        — Message, WorkoutPlan, NutritionPlan
  auth/
    auth.go           — (Фаза 5) хеширование паролей, сессии
  api/
    handler.go        — (Фаза 6) HTTP хендлеры
    middleware.go      — (Фаза 6) auth middleware
```

### 3.3 Конфиг-структура с валидацией
Собрать все настройки в одну структуру:
```go
type Config struct {
    DatabaseURL string
    SMTP        SMTPConfig
    EmailFrom   string
    Timezone    string // всегда "Europe/Moscow"
}

func LoadConfig() (*Config, error) {
    // загрузка из .env + валидация обязательных полей
}
```
Проверять при старте: если нет `DATABASE_URL` или `SMTP_CONFIG` — выходим с понятной ошибкой, а не паникуем в рантайме.

### 3.4 HTML-шаблоны через `html/template`
Заменить `fmt.Sprintf` с HTML в `email_templates.go` на Go templates:
- Защита от HTML-injection (имя пользователя может содержать `<script>`)
- Удобство редактирования шаблонов
- Разделение логики и представления

---

## Фаза 4. Валидация и обработка ошибок

### 4.1 Валидация email
Добавить проверку формата email через `net/mail.ParseAddress()` или регулярку. Проверять при:
- Регистрации (CLI и будущий API)
- Обновлении профиля

### 4.2 Валидация числовых полей
| Поле | Допустимые значения |
|------|-------------------|
| Возраст | 13–120 |
| Рост (см) | 100–250 |
| Вес (кг) | 30–300 |
| День недели | 0–6 |
| Час | 0–23 |
| Минута | 0–59 |

### 4.3 Проверка дубликатов email
Перед INSERT вызывать `GetUserByEmail()`. Если найден — показать "Пользователь с таким email уже существует", а не generic database error.

### 4.4 Проверка SMTP при старте
При запуске scheduler — подключиться к SMTP серверу и тут же отключиться. Если не получилось — не запускать scheduler, вывести ошибку.

### 4.5 Обёртка ошибок с контекстом
Заменить:
```go
return fmt.Errorf("ошибка создания: %w", err)
```
На:
```go
return fmt.Errorf("создание пользователя %s: %w", email, err)
```
Чтобы в логах было видно **что** упало и **для кого**.

### 4.6 Graceful error handling в scheduler
Если `GetActiveSchedulesForDay()` вернул ошибку — логировать WARNING и **продолжать** работу (retry на следующем тике), а не падать молча. Добавить счётчик ошибок подряд — если > 10, выходить с FATAL.

---

## Фаза 5. Авторизация (простая, логин + пароль)

### 5.1 Хеширование паролей
Использовать `golang.org/x/crypto/bcrypt`:
```go
func HashPassword(password string) (string, error)
func CheckPassword(hash, password string) bool
```
Хранить в существующем поле `password_hash` в таблице `users`.

### 5.2 Регистрация
При создании пользователя — запрашивать пароль (мин. 6 символов). Хешировать bcrypt и сохранять.

### 5.3 Сессии
Простейший вариант — JWT токен или session cookie:
- `POST /api/login` — проверка email + пароль → выдача токена
- `POST /api/register` — регистрация → выдача токена
- Токен передаётся в cookie или `Authorization: Bearer <token>`
- Middleware проверяет токен на всех защищённых эндпоинтах

### 5.4 Middleware авторизации
```go
func AuthMiddleware(next http.Handler) http.Handler {
    // проверить токен/сессию
    // если невалидный — 401
    // если валидный — положить userID в context
}
```
Пользователь может редактировать **только свои** данные (проверка `userID` из токена == `userID` в запросе).

---

## Фаза 6. Тесты

### 6.1 Unit-тесты (покрытие ≥70% критических путей)

| Что тестировать | Файл теста | Приоритет |
|----------------|-----------|-----------|
| Конвертация day-of-week | `scheduler_test.go` | Высокий — был баг |
| Генерация workout по цели | `email_templates_test.go` | Средний |
| Генерация nutrition по уровню активности | `email_templates_test.go` | Средний |
| Калорийные корректировки (возраст, пол) | `email_templates_test.go` | Средний |
| Валидация email | `validation_test.go` | Высокий |
| Валидация числовых полей | `validation_test.go` | Средний |
| Хеширование/проверка пароля | `auth_test.go` | Высокий |
| Конфиг: загрузка и валидация | `config_test.go` | Средний |

### 6.2 Integration-тесты

| Что тестировать | Как |
|----------------|-----|
| CRUD пользователей в БД | Тестовая PostgreSQL (testcontainers-go) |
| CRUD расписаний в БД | Тестовая PostgreSQL |
| Scheduler находит нужные расписания | Mock time + тестовая БД |
| Отправка email | Mock SMTP сервер (mailhog) |
| API endpoints | `httptest.NewServer` + реальный хендлер |
| Auth flow: register → login → access | `httptest` + тестовая БД |

### 6.3 Makefile цели для тестов
```makefile
test:
	go test ./... -v
test-coverage:
	go test ./... -coverprofile=coverage.out
	go tool cover -html=coverage.out
```

---

## Фаза 7. Web API + интерфейс управления

### 7.1 HTTP сервер
Использовать стандартный `net/http` + `http.ServeMux` (Go 1.22+ поддерживает паттерны с методами).

### 7.2 API endpoints

| Метод | Путь | Описание | Auth |
|-------|------|----------|------|
| POST | `/api/register` | Регистрация | Нет |
| POST | `/api/login` | Вход | Нет |
| GET | `/api/profile` | Получить свой профиль | Да |
| PUT | `/api/profile` | Обновить профиль (имя, возраст, рост, вес, цель, уровень активности) | Да |
| GET | `/api/schedules` | Список своих расписаний | Да |
| POST | `/api/schedules` | Создать расписание | Да |
| PUT | `/api/schedules/{id}` | Изменить расписание (день, время, тип) | Да |
| DELETE | `/api/schedules/{id}` | Удалить расписание | Да |
| POST | `/api/unsubscribe?token=xxx` | Отписка по ссылке из письма | Нет (токен) |

### 7.5 Защита от спама и ботов

Необходимо защитить публичные эндпоинты (`/api/register`, `/api/login`) от автоматизированных атак.

#### 7.5.1 Rate Limiting по IP
Ограничение количества запросов с одного IP-адреса:
```
/api/register — макс. 3 запроса в час с одного IP
/api/login    — макс. 10 запросов в 15 минут с одного IP
Все остальные — макс. 60 запросов в минуту с одного IP
```

Реализация — `golang.org/x/time/rate` + map по IP с TTL (или middleware с `sync.Map`):
```go
type RateLimiter struct {
    visitors map[string]*rate.Limiter
    mu       sync.RWMutex
}

func RateLimitMiddleware(limit rate.Limit, burst int) func(http.Handler) http.Handler
```
Раз в 10 минут чистить просроченные записи (goroutine с ticker), чтобы map не рос бесконечно.

#### 7.5.2 CSRF-токены для форм
Все HTML-формы (регистрация, логин, редактирование) должны содержать CSRF-токен:
- Генерировать `crypto/rand` токен при загрузке страницы
- Сохранять в cookie (`HttpOnly`, `SameSite=Strict`)
- Вставлять в `<input type="hidden" name="_csrf" value="...">`
- При POST проверять совпадение cookie и скрытого поля

Библиотека: `gorilla/csrf` или самописный middleware (простой вариант — ~50 строк).

#### 7.5.3 Honeypot-поля (защита от простых ботов)
В формы регистрации и логина добавить скрытое поле:
```html
<div style="display:none">
    <input type="text" name="website" tabindex="-1" autocomplete="off">
</div>
```
На сервере: если `website` заполнен — это бот, отклонить запрос с 200 OK (не давать боту понять что его поймали).

#### 7.5.4 Задержка при неудачном логине
- После 3 неудачных попыток входа для одного email — задержка 30 секунд
- После 5 попыток — блокировка на 15 минут
- Хранить счётчик в памяти (map email → attempts + timestamp)
- Не раскрывать, существует ли email ("Неверный email или пароль" — одинаковое сообщение)

#### 7.5.5 Заголовки безопасности
Middleware, который добавляет ко всем ответам:
```
X-Content-Type-Options: nosniff
X-Frame-Options: DENY
X-XSS-Protection: 1; mode=block
Content-Security-Policy: default-src 'self'
Referrer-Policy: strict-origin-when-cross-origin
```

### 7.3 Веб-страницы (минимальный UI)
Простые HTML-страницы с формами (server-side rendering, Go templates):

| Страница | Путь | Описание |
|----------|------|----------|
| Регистрация | `/register` | Форма: email, пароль, имя, возраст, пол, рост, вес, цель, уровень активности |
| Вход | `/login` | Форма: email + пароль |
| Личный кабинет | `/dashboard` | Профиль + список расписаний |
| Редактирование профиля | `/profile/edit` | Форма изменения данных |
| Управление расписанием | `/schedules` | Добавить/изменить/удалить дни и время отправки |
| Отписка | `/unsubscribe` | Подтверждение отписки по ссылке из письма |

Стилизация: минимальный CSS (можно classless CSS типа Water.css или Pico.css — подключается одной строкой).

### 7.4 Кнопка отписки в каждом email
В каждое письмо добавить ссылку:
```
Не хочешь больше получать письма? <a href="https://your-domain/unsubscribe?token=xxx">Отписаться</a>
```
Токен — подписанный JWT с userID + scheduleID. При клике — деактивация расписания.

---

## Фаза 8. AI-персонализация контента через DeepSeek

Сейчас тренировки и питание — **захардкоженные шаблоны** в `email_templates.go`. Всего 4 варианта тренировок (weight_loss, muscle_gain, maintenance, general_fitness) и 3 варианта питания. Пользователь с одинаковой целью получает **идентичные письма каждый день**. Это нужно исправить.

### 8.1 Интеграция DeepSeek API

Добавить клиент для DeepSeek API (OpenAI-совместимый формат):
```go
// internal/ai/deepseek.go
type DeepSeekClient struct {
    apiKey  string
    baseURL string // https://api.deepseek.com/v1
    client  *http.Client
}

func NewDeepSeekClient(apiKey string) *DeepSeekClient
func (c *DeepSeekClient) GenerateContent(prompt string) (string, error)
```

**Конфигурация:**
```env
DEEPSEEK_API_KEY=sk-xxx
DEEPSEEK_MODEL=deepseek-chat    # дешёвая модель для массовой генерации
```

DeepSeek Chat — самая дешёвая модель (~$0.14/1M input tokens, ~$0.28/1M output tokens). При 1000 пользователях × 3 письма/неделю × ~500 токенов на ответ ≈ $0.5-1/месяц.

### 8.2 Промпт-система для персонализации

Создать структурированные промпты, которые учитывают **все** данные профиля пользователя:

```go
func BuildWorkoutPrompt(user User, dayOfWeek string, emailType string) string {
    return fmt.Sprintf(`Ты — персональный фитнес-тренер. Составь тренировку на сегодня.

Данные пользователя:
- Имя: %s
- Пол: %s
- Возраст: %d лет
- Рост: %d см
- Вес: %.1f кг
- Цель: %s
- Уровень активности: %s
- День недели: %s
- Время суток: %s

Требования:
- Ответ ТОЛЬКО на русском языке
- Формат: JSON с полями title, exercises (массив строк), duration, description
- 4-6 упражнений с подходами и повторениями
- Учитывай день недели (не повторяй группы мышц подряд)
- Разнообразие: не повторяй одни и те же упражнения каждый день
- Учитывай время суток: утром — бодрящие, вечером — не слишком интенсивные`, ...)
}
```

Аналогичный промпт для питания:
```go
func BuildNutritionPrompt(user User) string {
    return fmt.Sprintf(`Ты — диетолог. Составь план питания на сегодня.

Данные пользователя:
- Пол: %s, Возраст: %d, Рост: %d см, Вес: %.1f кг
- Цель: %s
- Уровень активности: %s

Требования:
- Ответ ТОЛЬКО на русском
- JSON: { breakfast, lunch, dinner, snacks[], calories, protein_grams, water_ml }
- Конкретные блюда с граммовками
- Реалистичные продукты, доступные в РФ
- Калории соответствуют цели пользователя
- Разнообразие: не повторяй одни блюда каждый день`, ...)
}
```

### 8.3 Мотивационные сообщения от AI

Помимо тренировки и питания, добавить **уникальное мотивационное сообщение** в каждое письмо:
```go
func BuildMotivationPrompt(user User, dayOfWeek string) string {
    return fmt.Sprintf(`Напиши короткое мотивационное сообщение для %s.
Цель: %s. День: %s.

Требования:
- 2-3 предложения
- Юмор и лёгкая подколка (как друг, который тащит в зал)
- На русском языке
- Без банальных фраз типа "Ты можешь всё"
- Можно использовать эмодзи`, ...)
}
```

Это **ключевая фишка приложения** — каждое письмо уникальное, с юмором и персональным подходом.

### 8.4 Retry и обработка ошибок DeepSeek

Без кэша — каждый запрос идёт напрямую в DeepSeek API. Если API недоступен, пользователь должен об этом узнать.

**Стратегия retry:**
- 3 попытки с exponential backoff: 2с → 5с → 10с
- Таймаут на один запрос: 30 секунд
- Между пользователями пауза 300ms (защита от rate limit DeepSeek)

```go
func (c *DeepSeekClient) GenerateWithRetry(prompt string) (string, error) {
    delays := []time.Duration{2 * time.Second, 5 * time.Second, 10 * time.Second}
    var lastErr error

    for i, delay := range delays {
        content, err := c.GenerateContent(prompt)
        if err == nil {
            return content, nil
        }
        lastErr = err
        slog.Warn("deepseek attempt failed",
            "attempt", i+1, "error", err, "retry_in", delay)
        time.Sleep(delay)
    }
    return "", fmt.Errorf("deepseek failed after 3 attempts: %w", lastErr)
}
```

**При исчерпании всех попыток — отправить пользователю письмо с ошибкой:**
```go
func (s *EmailService) SendForSchedule(user User, day string, emailType string) error {
    // 1. Генерация через AI (с retry)
    content, err := s.generateAIContent(user, day, emailType)
    if err != nil {
        slog.Error("AI generation failed completely", "user", user.Email, "error", err)
        // 2. Отправить письмо-заглушку с извинением
        return s.sendErrorEmail(user, err)
    }
    // 3. Отправить нормальное письмо
    return s.sendEmail(user, content)
}
```

**Письмо-заглушка при ошибке:**
```
Тема: ⚠️ Сегодня без тренировки — технические неполадки

Привет, {Имя}!

К сожалению, сегодня не получилось сгенерировать
персональную тренировку — у нас технические проблемы.

Но это не повод пропускать зал! 💪
Сделай свою любимую тренировку или просто побегай 30 минут.

Завтра всё будет как обычно!
```

Это честно по отношению к пользователю — он знает что подписка работает, но была временная проблема. Лучше чем молча пропустить письмо.

### 8.5 Парсинг JSON-ответа от AI

DeepSeek возвращает текст. Нужно извлечь JSON:
```go
func ParseWorkoutResponse(response string) (*WorkoutPlan, error) {
    // Найти JSON в ответе (может быть обёрнут в ```json ... ```)
    jsonStr := extractJSON(response)
    var plan WorkoutPlan
    if err := json.Unmarshal([]byte(jsonStr), &plan); err != nil {
        return nil, fmt.Errorf("failed to parse AI response: %w", err)
    }
    // Валидация: есть ли exercises, duration и т.д.
    if len(plan.Exercises) == 0 {
        return nil, fmt.Errorf("AI returned empty workout")
    }
    return &plan, nil
}
```

При ошибке парсинга — fallback на шаблоны (см. 8.4).

---

## Фаза 9. Подготовка к нагрузке (1000 пользователей)

### 9.1 Rate limiting SMTP
Yandex SMTP ограничен ~500 писем/день. При 1000 пользователях, если у каждого 3 дня в неделю — это ~430 писем/день (укладываемся, но впритык).

Добавить очередь отправки:
- Между письмами пауза 200ms
- Если ошибка 429/rate limit — exponential backoff
- Лог количества отправленных за день

### 9.2 Connection pooling PostgreSQL
```go
db.SetMaxOpenConns(25)
db.SetMaxIdleConns(5)
db.SetConnMaxLifetime(5 * time.Minute)
```

### 9.3 Retry при ошибках SMTP
3 попытки с паузами 1с → 5с → 15с. Если все 3 провалились — записать в `email_logs` со статусом `failed`.

### 9.4 Таблица `email_logs`
```sql
CREATE TABLE email_logs (
    id SERIAL PRIMARY KEY,
    user_id UUID REFERENCES users(id),
    schedule_id INTEGER REFERENCES user_schedules(id),
    sent_at TIMESTAMP DEFAULT NOW(),
    status VARCHAR(20), -- 'sent', 'failed', 'retrying'
    error_message TEXT,
    email_type VARCHAR(20)
);
```
Также предотвращает дублирование — перед отправкой проверить, не было ли уже письма этому пользователю сегодня по этому расписанию.

### 9.5 Structured logging
Заменить `log.Printf` на `log/slog`:
```go
slog.Info("email sent", "user", user.Email, "schedule", schedule.ID)
slog.Error("smtp failed", "error", err, "user", user.Email)
```

### 9.6 Health check
```go
http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
    if err := db.Ping(); err != nil {
        http.Error(w, "db down", 503)
        return
    }
    w.Write([]byte("ok"))
})
```

---

## Фаза 10. Финальная сборка и проверка

### 10.1 Миграции БД
Вместо единого `schema.sql` — пронумерованные миграции:
```
migrations/
  001_create_users.up.sql
  001_create_users.down.sql
  002_create_schedules.up.sql
  002_create_schedules.down.sql
  003_create_email_logs.up.sql
  003_create_email_logs.down.sql
  004_add_password_hash.up.sql
  ...
```
Использовать `golang-migrate/migrate`.

### 10.2 CI/CD (GitHub Actions)
```yaml
on: [push, pull_request]
jobs:
  test:
    - go vet ./...
    - golangci-lint run
    - go test ./... -race -coverprofile=coverage.out
  build:
    - docker build -t daily-email-sender .
```

### 10.3 Обновить Dockerfile
- Multi-stage build (уже есть, проверить актуальность)
- Использовать `CMD ["./app", "serve"]` — новая команда, запускающая HTTP сервер + scheduler одновременно

### 10.4 Обновить docker-compose.yml
- Добавить `email_logs` в schema
- Убрать захардкоженные пароли
- Добавить resource limits

### 10.5 Обновить README
- Актуальная инструкция по запуску
- Описание API endpoints
- Скриншоты веб-интерфейса

### 10.6 Smoke test полного цикла
Чеклист перед релизом:
- [ ] Регистрация через веб-форму
- [ ] Логин
- [ ] Редактирование профиля
- [ ] Создание расписания
- [ ] Изменение расписания
- [ ] Удаление расписания
- [ ] Scheduler находит расписание и отправляет письмо
- [ ] Письмо приходит с правильным контентом
- [ ] Кнопка отписки работает
- [ ] Health check отвечает 200
- [ ] Логи пишутся корректно
- [ ] email_logs заполняется
- [ ] Повторная отправка не происходит (дедупликация)
- [ ] Graceful shutdown: scheduler останавливается корректно
- [ ] AI-генерация: письмо содержит уникальную тренировку/питание от DeepSeek
- [ ] AI-ошибка: при недоступности DeepSeek после 3 retry — приходит письмо-заглушка с извинением
- [ ] Rate limit: 4-я регистрация с одного IP за час блокируется
- [ ] CSRF: POST без токена возвращает 403
- [ ] Honeypot: заполненное скрытое поле → тихий отказ
- [ ] Блокировка логина после 5 неудачных попыток

---

## Порядок выполнения (рекомендация)

```
Фаза 1  (чистка)            ████░░░░░░  — 1 день
Фаза 2  (баги)              ████░░░░░░  — 1 день
Фаза 3  (рефакторинг)       ████████░░  — 2-3 дня
Фаза 4  (валидация)         ████░░░░░░  — 1 день
Фаза 5  (авторизация)       ██████░░░░  — 1-2 дня
Фаза 6  (тесты)             ████████░░  — 2-3 дня
Фаза 7  (web UI + антиспам) ██████████  — 4-5 дней
Фаза 8  (AI персонализация) ████████░░  — 2-3 дня
Фаза 9  (масштабирование)   ██████░░░░  — 1-2 дня
Фаза 10 (финал)             ██████░░░░  — 1-2 дня
```

**Итого: ~16-23 рабочих дня до полностью рабочего релиза.**
