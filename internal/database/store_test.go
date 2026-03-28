package database

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"daily-email-sender/internal/models"
)

const testSchema = `
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255),
    first_name VARCHAR(100),
    last_name VARCHAR(100),
    age INTEGER,
    gender VARCHAR(20) CHECK (gender IN ('male', 'female', 'other')),
    height_cm INTEGER,
    weight_kg DECIMAL(5, 2),
    goal VARCHAR(50) CHECK (goal IN ('weight_loss', 'muscle_gain', 'maintenance', 'general_fitness')),
    activity_level VARCHAR(50) CHECK (activity_level IN ('sedentary', 'light', 'moderate', 'active', 'very_active')),
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
CREATE TABLE IF NOT EXISTS user_schedules (
    id SERIAL PRIMARY KEY,
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    day_of_week INTEGER CHECK (day_of_week BETWEEN 0 AND 6),
    time_hour INTEGER CHECK (time_hour BETWEEN 0 AND 23),
    time_minute INTEGER CHECK (time_minute BETWEEN 0 AND 59),
    email_type VARCHAR(20) CHECK (email_type IN ('morning', 'afternoon', 'evening')),
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(user_id, day_of_week, time_hour, time_minute)
);
CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);
CREATE INDEX IF NOT EXISTS idx_schedules_user_id ON user_schedules(user_id);
CREATE INDEX IF NOT EXISTS idx_schedules_day_time ON user_schedules(day_of_week, time_hour, time_minute);
`

// setupTestDB запускает PostgreSQL в Docker-контейнере и возвращает Store.
// Пропускает тест если Docker недоступен.
func setupTestDB(t *testing.T) *Store {
	t.Helper()

	ctx := context.Background()

	// testcontainers может паниковать на Windows если Docker недоступен
	var pgContainer *postgres.PostgresContainer
	var containerErr error

	func() {
		defer func() {
			if r := recover(); r != nil {
				containerErr = fmt.Errorf("Docker panic: %v", r)
			}
		}()
		pgContainer, containerErr = postgres.Run(ctx,
			"postgres:16-alpine",
			postgres.WithDatabase("testdb"),
			postgres.WithUsername("test"),
			postgres.WithPassword("test"),
			testcontainers.WithWaitStrategy(
				wait.ForLog("database system is ready to accept connections").
					WithOccurrence(2).
					WithStartupTimeout(30*time.Second),
			),
		)
	}()

	if containerErr != nil {
		t.Skipf("Docker недоступен, пропуск integration-тестов: %v", containerErr)
	}
	t.Cleanup(func() {
		pgContainer.Terminate(ctx)
	})

	connStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("не удалось получить connection string: %v", err)
	}

	store, err := NewStore(connStr)
	if err != nil {
		t.Fatalf("не удалось подключиться к тестовой БД: %v", err)
	}
	t.Cleanup(func() {
		store.Close()
	})

	if err := store.ExecRaw(testSchema); err != nil {
		t.Fatalf("не удалось создать схему: %v", err)
	}

	return store
}

func newTestUser(email string) *models.User {
	return &models.User{
		Email:         email,
		FirstName:     "Тест",
		LastName:      "Тестов",
		Age:           25,
		Gender:        "male",
		HeightCm:      180,
		WeightKg:      80.0,
		Goal:          "muscle_gain",
		ActivityLevel: "moderate",
	}
}

func TestCreateUser(t *testing.T) {
	store := setupTestDB(t)

	user := newTestUser("create@test.com")
	err := store.CreateUser(user)
	if err != nil {
		t.Fatalf("CreateUser error: %v", err)
	}

	if user.ID == "" {
		t.Error("ID should be set after create")
	}
	if user.CreatedAt.IsZero() {
		t.Error("CreatedAt should be set")
	}
	if user.UpdatedAt.IsZero() {
		t.Error("UpdatedAt should be set")
	}
}

func TestCreateUser_DuplicateEmail(t *testing.T) {
	store := setupTestDB(t)

	user1 := newTestUser("dup@test.com")
	if err := store.CreateUser(user1); err != nil {
		t.Fatalf("CreateUser 1 error: %v", err)
	}

	user2 := newTestUser("dup@test.com")
	err := store.CreateUser(user2)
	if err == nil {
		t.Fatal("expected error for duplicate email")
	}
}

func TestGetUserByID(t *testing.T) {
	store := setupTestDB(t)

	user := newTestUser("byid@test.com")
	if err := store.CreateUser(user); err != nil {
		t.Fatalf("CreateUser error: %v", err)
	}

	found, err := store.GetUserByID(user.ID)
	if err != nil {
		t.Fatalf("GetUserByID error: %v", err)
	}

	if found.Email != "byid@test.com" {
		t.Errorf("expected email byid@test.com, got %s", found.Email)
	}
	if found.FirstName != "Тест" {
		t.Errorf("expected FirstName Тест, got %s", found.FirstName)
	}
	if found.WeightKg != 80.0 {
		t.Errorf("expected WeightKg 80, got %f", found.WeightKg)
	}
}

func TestGetUserByID_NotFound(t *testing.T) {
	store := setupTestDB(t)

	_, err := store.GetUserByID("00000000-0000-0000-0000-000000000000")
	if err == nil {
		t.Error("expected error for non-existent user")
	}
}

func TestGetUserByEmail(t *testing.T) {
	store := setupTestDB(t)

	user := newTestUser("byemail@test.com")
	if err := store.CreateUser(user); err != nil {
		t.Fatalf("CreateUser error: %v", err)
	}

	found, err := store.GetUserByEmail("byemail@test.com")
	if err != nil {
		t.Fatalf("GetUserByEmail error: %v", err)
	}
	if found.ID != user.ID {
		t.Errorf("expected ID %s, got %s", user.ID, found.ID)
	}
}

func TestGetUserByEmail_NotFound(t *testing.T) {
	store := setupTestDB(t)

	_, err := store.GetUserByEmail("nonexistent@test.com")
	if err == nil {
		t.Error("expected error for non-existent email")
	}
}

func TestGetAllUsers(t *testing.T) {
	store := setupTestDB(t)

	// Пустая таблица
	users, err := store.GetAllUsers()
	if err != nil {
		t.Fatalf("GetAllUsers error: %v", err)
	}
	if len(users) != 0 {
		t.Errorf("expected 0 users, got %d", len(users))
	}

	// Добавляем двух
	if err := store.CreateUser(newTestUser("all1@test.com")); err != nil {
		t.Fatalf("CreateUser error: %v", err)
	}
	if err := store.CreateUser(newTestUser("all2@test.com")); err != nil {
		t.Fatalf("CreateUser error: %v", err)
	}

	users, err = store.GetAllUsers()
	if err != nil {
		t.Fatalf("GetAllUsers error: %v", err)
	}
	if len(users) != 2 {
		t.Errorf("expected 2 users, got %d", len(users))
	}
}

func TestCreateUserSchedule(t *testing.T) {
	store := setupTestDB(t)

	user := newTestUser("sched@test.com")
	if err := store.CreateUser(user); err != nil {
		t.Fatalf("CreateUser error: %v", err)
	}

	schedule := &models.UserSchedule{
		UserID:     user.ID,
		DayOfWeek:  0, // Понедельник
		TimeHour:   9,
		TimeMinute: 0,
		EmailType:  "morning",
	}
	err := store.CreateUserSchedule(schedule)
	if err != nil {
		t.Fatalf("CreateUserSchedule error: %v", err)
	}
	if schedule.ID == 0 {
		t.Error("schedule ID should be set after create")
	}
}

func TestCreateUserSchedule_Upsert(t *testing.T) {
	store := setupTestDB(t)

	user := newTestUser("upsert@test.com")
	if err := store.CreateUser(user); err != nil {
		t.Fatalf("CreateUser error: %v", err)
	}

	sched1 := &models.UserSchedule{
		UserID: user.ID, DayOfWeek: 1, TimeHour: 10, TimeMinute: 30, EmailType: "morning",
	}
	if err := store.CreateUserSchedule(sched1); err != nil {
		t.Fatalf("CreateUserSchedule 1 error: %v", err)
	}

	// Повторный INSERT с тем же (user_id, day, hour, minute) — upsert
	sched2 := &models.UserSchedule{
		UserID: user.ID, DayOfWeek: 1, TimeHour: 10, TimeMinute: 30, EmailType: "afternoon",
	}
	if err := store.CreateUserSchedule(sched2); err != nil {
		t.Fatalf("CreateUserSchedule 2 (upsert) error: %v", err)
	}

	// Должен быть один и тот же ID
	if sched1.ID != sched2.ID {
		t.Errorf("upsert should return same ID: %d vs %d", sched1.ID, sched2.ID)
	}
}

func TestGetActiveSchedulesForDay(t *testing.T) {
	store := setupTestDB(t)

	user := newTestUser("daytest@test.com")
	if err := store.CreateUser(user); err != nil {
		t.Fatalf("CreateUser error: %v", err)
	}

	// Добавим расписание на понедельник (0) и среду (2)
	for _, day := range []int{0, 2} {
		sched := &models.UserSchedule{
			UserID: user.ID, DayOfWeek: day, TimeHour: 8, TimeMinute: 0, EmailType: "morning",
		}
		if err := store.CreateUserSchedule(sched); err != nil {
			t.Fatalf("CreateUserSchedule day=%d error: %v", day, err)
		}
	}

	// Проверяем понедельник
	schedules, err := store.GetActiveSchedulesForDay(0)
	if err != nil {
		t.Fatalf("GetActiveSchedulesForDay(0) error: %v", err)
	}
	if len(schedules) != 1 {
		t.Errorf("expected 1 schedule for Monday, got %d", len(schedules))
	}
	if len(schedules) > 0 && schedules[0].DayOfWeek != 0 {
		t.Errorf("expected day 0, got %d", schedules[0].DayOfWeek)
	}

	// Вторник — пусто
	schedules, err = store.GetActiveSchedulesForDay(1)
	if err != nil {
		t.Fatalf("GetActiveSchedulesForDay(1) error: %v", err)
	}
	if len(schedules) != 0 {
		t.Errorf("expected 0 schedules for Tuesday, got %d", len(schedules))
	}

	// Среда — одно расписание
	schedules, err = store.GetActiveSchedulesForDay(2)
	if err != nil {
		t.Fatalf("GetActiveSchedulesForDay(2) error: %v", err)
	}
	if len(schedules) != 1 {
		t.Errorf("expected 1 schedule for Wednesday, got %d", len(schedules))
	}
}

func TestGetActiveSchedulesForDay_MultipleUsers(t *testing.T) {
	store := setupTestDB(t)

	// Два пользователя с расписаниями на один день
	user1 := newTestUser("multi1@test.com")
	user2 := newTestUser("multi2@test.com")
	if err := store.CreateUser(user1); err != nil {
		t.Fatalf("CreateUser 1 error: %v", err)
	}
	if err := store.CreateUser(user2); err != nil {
		t.Fatalf("CreateUser 2 error: %v", err)
	}

	for _, uid := range []string{user1.ID, user2.ID} {
		sched := &models.UserSchedule{
			UserID: uid, DayOfWeek: 4, TimeHour: 18, TimeMinute: 0, EmailType: "evening",
		}
		if err := store.CreateUserSchedule(sched); err != nil {
			t.Fatalf("CreateUserSchedule error: %v", err)
		}
	}

	schedules, err := store.GetActiveSchedulesForDay(4)
	if err != nil {
		t.Fatalf("GetActiveSchedulesForDay error: %v", err)
	}
	if len(schedules) != 2 {
		t.Errorf("expected 2 schedules for Friday, got %d", len(schedules))
	}
}
