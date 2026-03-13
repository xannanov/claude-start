# Daily Email Sender

Go приложение для отправки персонализированных мотивационных писем пользователям на основе их данных о тренировках и питании.

## Возможности

- 📧 Отправка персонализированных писем (утро, день, вечер)
- 👤 Управление пользователями через CLI
- 📅 Настраиваемое расписание для каждого пользователя
- 🏋️ Персонализированные планы тренировок на основе целей
- 🍎 Персонализированные планы питания на основе параметров
- 🗄️ PostgreSQL база данных для хранения данных
- ⏰ Автоматическая отправка по расписанию

## Установка

1. Установите PostgreSQL 17+
2. Установите Go 1.21 или выше
3. Склонируйте репозиторий

```bash
cd /path/to/claude-start
```

### Установка зависимостей

```bash
make deps
```

Или напрямую:
```bash
go mod download
```

### Сборка

```bash
make build
```

Или напрямую:
```bash
go build -o daily-email-sender.exe .
```

## Настройка

1. Скопируйте `.env.example` в `.env`:

```bash
cp .env.example .env
```

2. Отредактируйте `.env` файл с вашими данными:

```env
SMTP_CONFIG={"Host":"smtp.yandex.ru","Port":465,"User":"your-email@yandex.com","Password":"your-password"}
EMAIL_CONFIG={"From":"sender@example.com","To":"recipient@example.com"}
DATABASE_URL="postgres://postgres:admin@localhost:5432/daily_email_sender?sslmode=disable&client_encoding=UTF-8"
```

### Настройка PostgreSQL

1. Создайте базу данных (используйте пароль от PostgreSQL):

```bash
"C:/Program Files/PostgreSQL/17/bin/psql" -U postgres -c "CREATE DATABASE daily_email_sender;"
```

Или через pgAdmin / pgAdmin 4:
- Откройте pgAdmin
- Создайте новую базу данных: daily_email_sender
- Используйте пользователя postgres

2. Инициализируйте схему:

```bash
make init-db
```

Или напрямую:
```bash
go run . init-db
```

### Настройка SMTP

- **Yandex Mail**: `smtp.yandex.ru:465` (SSL)
- **Gmail**: `smtp.gmail.com:587` (TLS) - нужен App Password
- **Microsoft**: `smtp.office365.com:587` (TLS)

**Советы для Gmail:**
1. Включите 2FA в аккаунте Google
2. Создайте App Password: Settings → Security → 2-Step Verification → App Passwords
3. Используйте этот пароль в .env файле

## Использование

### 1. Создать пользователя

```bash
make add-user
```

Интерактивно введите:
- **Email** (обязательно)
- **Имя (First Name)** (обязательно)
- **Фамилия (Last Name)** (обязательно)
- **Возраст** (обязательно)
- **Пол**: male / female / other
- **Рост** (в сантиметрах)
- **Вес** (в килограммах)
- **Цель**: weight_loss / muscle_gain / maintenance / general_fitness
- **Уровень активности**: sedentary / light / moderate / active / very_active

**Пример ввода:**
```
Email: user@example.com
Имя (First Name): John
Фамилия (Last Name): Doe
Возраст: 30
Пол (male/female/other): male
Рост (cm): 175
Вес (kg): 75
Цель (weight_loss/muscle_gain/maintenance/general_fitness): muscle_gain
Уровень активности (sedentary/light/moderate/active/very_active): active
```

### 2. Посмотреть список пользователей

```bash
make list-users
```

### 3. Добавить расписание для пользователя

```bash
make add-schedule
```

Выберите пользователя из списка и задайте:
- **День недели** (0-6: Пн-Вс)
- **Время** (часы и минуты)
- **Тип email**: morning / afternoon / evening

**Пример:**
```
Доступные пользователи:
1. user@example.com (uuid-here)

Выберите пользователя (1-1): 1

Добавление расписания для: user@example.com
День недели (0-Пн, 1-Вт, 2-Ср, 3-Чт, 4-Пт, 5-Сб, 6-Вс): 0
Час (0-23): 9
Минута (0-59): 0
Тип email (morning/afternoon/evening): morning

✓ Расписание успешно добавлено!
```

### 4. Запустить планировщик

```bash
make run-scheduler
```

Планировщик работает в фоновом режиме:
- Проверяет расписание каждую минуту
- Отправляет письма в назначенное время
- **Важно**: Закройте терминал чтобы остановить планировщик

**Альтернатива**: Запустите в отдельном терминале в фоне:

```bash
# Терминал 1
make run-scheduler
```

```bash
# Терминал 2 (доступ только для теста)
# Откройте базу данных и проверьте отправку
```

### 5. Помощь

```bash
make run
```

## Makefile команды

```bash
make run                # Показать справку по командам
make build              # Сборка приложения
make deps               # Скачать зависимости
make clean              # Очистка (удалить бинарник)
make test               # Тест компиляции (тестирует только main.go)
make init-db            # Инициализация базы данных
make run-scheduler      # Запустить планировщик
make add-user           # Добавить пользователя (интерактивно)
make list-users         # Показать список пользователей
make add-schedule       # Добавить расписание (интерактивно)
make full-setup         # Полный цикл создания пользователя
```

## Структура проекта

```
.
├── main.go           # Точка входа с CLI командами
├── cli.go            # CLI функции для работы с пользователями
├── db.go             # Функции работы с базой данных
├── scheduler.go      # Планировщик email отправок
├── email_templates.go # HTML шаблоны email
├── models.go         # Модели данных (User, UserSchedule, etc.)
├── schema.sql        # SQL схема базы данных
├── .env.example      # Пример .env файла
├── Makefile          # Команды сборки
└── README.md         # Эта документация
```

## База данных

### Таблицы

- **users**: Хранит информацию о пользователях
  - id (UUID, primary key)
  - email (уникальный)
  - password_hash
  - first_name, last_name
  - age, gender
  - height_cm, weight_kg
  - goal, activity_level
  - workout_preferences (JSONB)
  - is_active (boolean)
  - created_at, updated_at

- **user_schedules**: Хранит расписание отправки email
  - id (SERIAL, primary key)
  - user_id (UUID, foreign key)
  - day_of_week (0-6)
  - time_hour (0-23)
  - time_minute (0-59)
  - email_type (morning/afternoon/evening)
  - is_active (boolean)
  - created_at, updated_at
  - UNIQUE(user_id, day_of_week, time_hour, time_minute)

## Логика персонализации

### Тренировки

В зависимости от цели пользователя:

- **weight_loss**: Кардио + базовые упражнения для сжигания калорий
- **muscle_gain**: Силовые упражнения с большими весами
- **maintenance**: Балансированная нагрузка
- **general_fitness**: Универсальная фитнес-тренировка

### Питание

- Рассчитывается на основе уровня активности и возраста
- Учитывается пол (дополнительные 300 ккал для мужчин)
- Уменьшение для возраста 30-50 и 50+
- План питания на завтрак, обед, ужин и перекусы

## Пример работы

### Полный цикл:

```bash
# 1. Инициализация
make init-db

# 2. Создать пользователя
make add-user

# 3. Добавить расписание
make add-schedule

# 4. Запустить планировщик
make run-scheduler

# 5. Проверить email в почтовом клиенте в указанное время
```

### Интерактивный пример:

```bash
# Открыть терминал для планировщика
make run-scheduler

# Открыть новый терминал для создания пользователя
make add-user
# Ввести данные: test@example.com | Test | User | 25 | male | 180 | 75 | muscle_gain | active

# Открыть еще один терминал для добавления расписания
make add-schedule
# Выбрать пользователя и задать расписание
```

## Требования

- **Go** 1.21 или выше
- **PostgreSQL** 17+ (или совместимая версия)
- Пароль от PostgreSQL пользователя (обычно "admin")
- SMTP доступ к email сервису

**Рекомендуемые провайдеры:**
- Yandex Mail (бесплатно, 1000 писем/день)
- Gmail (создайте App Password)
- Microsoft 365 (создайте специальный аккаунт)

## Troubleshooting

### "database not connected" ошибка

Убедитесь что .env файл создан и содержит правильные креды:

```env
DATABASE_URL="postgres://postgres:YOUR_PASSWORD@localhost:5432/daily_email_sender?sslmode=disable&client_encoding=UTF-8"
```

### "cannot determine database owner" ошибка

База данных не создана. Создайте её через pgAdmin или:

```bash
"C:/Program Files/PostgreSQL/17/bin/psql" -U postgres -c "CREATE DATABASE daily_email_sender;"
```

### "redeclared in this block" ошибка

Ошибка сборки при наличии test files. Удалите временные файлы:

```bash
make clean && make build
```

### Кракозябры при подключении к БД

Добавьте `client_encoding=UTF-8` в DATABASE_URL:

```env
DATABASE_URL="...?sslmode=disable&client_encoding=UTF-8"
```

## Лицензия

MIT
