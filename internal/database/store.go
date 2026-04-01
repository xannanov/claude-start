package database

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/lib/pq"

	"daily-email-sender/internal/models"
)

// Store предоставляет все операции с базой данных.
type Store struct {
	db *sql.DB
}

// NewStore создаёт подключение к PostgreSQL с настройкой пула соединений.
func NewStore(databaseURL string) (*Store, error) {
	db, err := sql.Open("postgres", databaseURL)
	if err != nil {
		return nil, fmt.Errorf("ошибка открытия БД: %w", err)
	}

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	if err = db.Ping(); err != nil {
		return nil, fmt.Errorf("ошибка подключения к БД: %w", err)
	}

	return &Store{db: db}, nil
}

// Close закрывает соединение с БД.
func (s *Store) Close() {
	if s.db != nil {
		s.db.Close()
	}
}

// GetUserByID возвращает активного пользователя по UUID.
func (s *Store) GetUserByID(id string) (*models.User, error) {
	query := `
		SELECT id, email, first_name, last_name, age, gender,
		       height_cm, weight_kg, goal, activity_level
		FROM users
		WHERE id = $1 AND is_active = true
	`

	row := s.db.QueryRow(query, id)
	user := &models.User{}

	err := row.Scan(
		&user.ID, &user.Email, &user.FirstName, &user.LastName,
		&user.Age, &user.Gender, &user.HeightCm, &user.WeightKg,
		&user.Goal, &user.ActivityLevel,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("пользователь не найден")
	}
	if err != nil {
		return nil, fmt.Errorf("ошибка запроса пользователя: %w", err)
	}

	return user, nil
}

// GetUserByEmail возвращает активного пользователя по email.
func (s *Store) GetUserByEmail(email string) (*models.User, error) {
	query := `
		SELECT id, email, first_name, last_name, age, gender,
		       height_cm, weight_kg, goal, activity_level
		FROM users
		WHERE email = $1 AND is_active = true
	`

	row := s.db.QueryRow(query, email)
	user := &models.User{}

	err := row.Scan(
		&user.ID, &user.Email, &user.FirstName, &user.LastName,
		&user.Age, &user.Gender, &user.HeightCm, &user.WeightKg,
		&user.Goal, &user.ActivityLevel,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("пользователь не найден")
	}
	if err != nil {
		return nil, fmt.Errorf("ошибка запроса пользователя: %w", err)
	}

	return user, nil
}

// GetAllUsers возвращает всех активных пользователей.
func (s *Store) GetAllUsers() ([]models.User, error) {
	query := `
		SELECT id, email, first_name, last_name, age, gender,
		       height_cm, weight_kg, goal, activity_level,
		       created_at, updated_at
		FROM users
		WHERE is_active = true
		ORDER BY email
	`

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("ошибка запроса пользователей: %w", err)
	}
	defer rows.Close()

	var users []models.User
	for rows.Next() {
		var u models.User
		err := rows.Scan(
			&u.ID, &u.Email, &u.FirstName, &u.LastName,
			&u.Age, &u.Gender, &u.HeightCm, &u.WeightKg,
			&u.Goal, &u.ActivityLevel,
			&u.CreatedAt, &u.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("ошибка чтения пользователя: %w", err)
		}
		users = append(users, u)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("ошибка итерации пользователей: %w", err)
	}

	return users, nil
}

// CreateUser создаёт нового пользователя и заполняет ID, CreatedAt, UpdatedAt.
// Возвращает ошибку "Пользователь с таким email уже существует" если email занят.
func (s *Store) CreateUser(user *models.User) error {
	return s.CreateUserWithPassword(user, "")
}

// CreateUserWithPassword создаёт пользователя с хешем пароля.
// Если passwordHash пустой — поле password_hash будет NULL.
// Если пользователь с таким email деактивирован — реактивирует его с новыми данными.
func (s *Store) CreateUserWithPassword(user *models.User, passwordHash string) error {
	// Проверяем всех пользователей (включая неактивных) чтобы не упасть на UNIQUE constraint
	var existingID string
	var isActive bool
	err := s.db.QueryRow(
		`SELECT id, is_active FROM users WHERE email = $1`, user.Email,
	).Scan(&existingID, &isActive)

	if err == nil && isActive {
		return fmt.Errorf("пользователь с email '%s' уже существует", user.Email)
	}

	if err == nil && !isActive {
		// Реактивируем деактивированного пользователя с новыми данными
		return s.db.QueryRow(`
			UPDATE users SET
				password_hash = NULLIF($2, ''), first_name = $3, last_name = $4,
				age = $5, gender = $6, height_cm = $7, weight_kg = $8,
				goal = $9, activity_level = $10, is_active = true, updated_at = CURRENT_TIMESTAMP
			WHERE id = $1
			RETURNING id, created_at, updated_at`,
			existingID, passwordHash, user.FirstName, user.LastName, user.Age, user.Gender,
			user.HeightCm, user.WeightKg, user.Goal, user.ActivityLevel,
		).Scan(&user.ID, &user.CreatedAt, &user.UpdatedAt)
	}

	query := `
		INSERT INTO users (
			email, password_hash, first_name, last_name, age, gender,
			height_cm, weight_kg, goal, activity_level, is_active
		) VALUES ($1, NULLIF($2, ''), $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING id, created_at, updated_at
	`

	err = s.db.QueryRow(
		query,
		user.Email, passwordHash, user.FirstName, user.LastName, user.Age, user.Gender,
		user.HeightCm, user.WeightKg, user.Goal, user.ActivityLevel, true,
	).Scan(&user.ID, &user.CreatedAt, &user.UpdatedAt)

	if err != nil {
		return fmt.Errorf("ошибка создания пользователя: %w", err)
	}
	return nil
}

// GetPasswordHashByEmail возвращает ID и хеш пароля активного пользователя по email.
// Используется для аутентификации при логине.
func (s *Store) GetPasswordHashByEmail(email string) (string, string, error) {
	query := `
		SELECT id, COALESCE(password_hash, '')
		FROM users
		WHERE email = $1 AND is_active = true
	`

	var userID, hash string
	err := s.db.QueryRow(query, email).Scan(&userID, &hash)
	if err == sql.ErrNoRows {
		return "", "", fmt.Errorf("пользователь не найден")
	}
	if err != nil {
		return "", "", fmt.Errorf("ошибка запроса пользователя: %w", err)
	}

	return userID, hash, nil
}

// UpdatePasswordHash обновляет хеш пароля пользователя.
func (s *Store) UpdatePasswordHash(userID, passwordHash string) error {
	query := `
		UPDATE users SET password_hash = $1, updated_at = CURRENT_TIMESTAMP
		WHERE id = $2 AND is_active = true
	`

	result, err := s.db.Exec(query, passwordHash, userID)
	if err != nil {
		return fmt.Errorf("ошибка обновления пароля: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("ошибка проверки обновления: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("пользователь не найден")
	}

	return nil
}

// CountActiveSchedulesByUserID возвращает количество активных расписаний пользователя.
func (s *Store) CountActiveSchedulesByUserID(userID string) (int, error) {
	var count int
	err := s.db.QueryRow(`
		SELECT COUNT(*) FROM user_schedules
		WHERE user_id = $1 AND is_active = true
	`, userID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("ошибка подсчёта расписаний: %w", err)
	}
	return count, nil
}

// CreateUserSchedule создаёт расписание. Возвращает ошибку если такое уже существует.
func (s *Store) CreateUserSchedule(schedule *models.UserSchedule) error {
	// Проверяем дубликат до вставки
	var exists bool
	err := s.db.QueryRow(`
		SELECT EXISTS(
			SELECT 1 FROM user_schedules
			WHERE user_id=$1 AND day_of_week=$2 AND time_hour=$3 AND time_minute=$4 AND is_active=true
		)`,
		schedule.UserID, schedule.DayOfWeek, schedule.TimeHour, schedule.TimeMinute,
	).Scan(&exists)
	if err != nil {
		return fmt.Errorf("Ошибка проверки дубликата: %w", err)
	}
	if exists {
		return fmt.Errorf("Расписание на этот день и время уже существует")
	}

	return s.db.QueryRow(`
		INSERT INTO user_schedules (user_id, day_of_week, time_hour, time_minute, email_type, is_active)
		VALUES ($1, $2, $3, $4, $5, true)
		RETURNING id`,
		schedule.UserID, schedule.DayOfWeek, schedule.TimeHour,
		schedule.TimeMinute, schedule.EmailType,
	).Scan(&schedule.ID)
}

// ExecRaw выполняет произвольный SQL-запрос (используется для инициализации схемы).
func (s *Store) ExecRaw(sql string) error {
	_, err := s.db.Exec(sql)
	return err
}

// GetActiveSchedulesForDay возвращает все активные расписания на указанный день.
// dayOfWeek: 0 = понедельник, 6 = воскресенье.
func (s *Store) GetActiveSchedulesForDay(dayOfWeek int) ([]models.UserSchedule, error) {
	query := `
		SELECT id, user_id, day_of_week, time_hour, time_minute, email_type
		FROM user_schedules
		WHERE day_of_week = $1 AND is_active = true
		ORDER BY time_hour, time_minute
	`

	rows, err := s.db.Query(query, dayOfWeek)
	if err != nil {
		return nil, fmt.Errorf("ошибка запроса расписаний: %w", err)
	}
	defer rows.Close()

	var schedules []models.UserSchedule
	for rows.Next() {
		var sc models.UserSchedule
		if err := rows.Scan(&sc.ID, &sc.UserID, &sc.DayOfWeek,
			&sc.TimeHour, &sc.TimeMinute, &sc.EmailType); err != nil {
			return nil, fmt.Errorf("ошибка чтения расписания: %w", err)
		}
		schedules = append(schedules, sc)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("ошибка итерации расписаний: %w", err)
	}

	return schedules, nil
}

// UpdateUser обновляет профиль пользователя (без email и пароля).
func (s *Store) UpdateUser(user *models.User) error {
	query := `
		UPDATE users SET
			first_name = $1, last_name = $2, age = $3, gender = $4,
			height_cm = $5, weight_kg = $6, goal = $7, activity_level = $8,
			updated_at = CURRENT_TIMESTAMP
		WHERE id = $9 AND is_active = true
	`

	result, err := s.db.Exec(query,
		user.FirstName, user.LastName, user.Age, user.Gender,
		user.HeightCm, user.WeightKg, user.Goal, user.ActivityLevel,
		user.ID,
	)
	if err != nil {
		return fmt.Errorf("ошибка обновления пользователя: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("ошибка проверки обновления: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("пользователь не найден")
	}
	return nil
}

// GetSchedulesByUserID возвращает все активные расписания пользователя.
func (s *Store) GetSchedulesByUserID(userID string) ([]models.UserSchedule, error) {
	query := `
		SELECT id, user_id, day_of_week, time_hour, time_minute, email_type, is_active
		FROM user_schedules
		WHERE user_id = $1 AND is_active = true
		ORDER BY day_of_week, time_hour, time_minute
	`

	rows, err := s.db.Query(query, userID)
	if err != nil {
		return nil, fmt.Errorf("ошибка запроса расписаний: %w", err)
	}
	defer rows.Close()

	var schedules []models.UserSchedule
	for rows.Next() {
		var sc models.UserSchedule
		if err := rows.Scan(&sc.ID, &sc.UserID, &sc.DayOfWeek,
			&sc.TimeHour, &sc.TimeMinute, &sc.EmailType, &sc.IsActive); err != nil {
			return nil, fmt.Errorf("ошибка чтения расписания: %w", err)
		}
		schedules = append(schedules, sc)
	}
	return schedules, rows.Err()
}

// GetScheduleByID возвращает расписание по ID.
func (s *Store) GetScheduleByID(id int) (*models.UserSchedule, error) {
	query := `
		SELECT id, user_id, day_of_week, time_hour, time_minute, email_type, is_active
		FROM user_schedules
		WHERE id = $1
	`

	sc := &models.UserSchedule{}
	err := s.db.QueryRow(query, id).Scan(
		&sc.ID, &sc.UserID, &sc.DayOfWeek,
		&sc.TimeHour, &sc.TimeMinute, &sc.EmailType, &sc.IsActive,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("расписание не найдено")
	}
	if err != nil {
		return nil, fmt.Errorf("ошибка запроса расписания: %w", err)
	}
	return sc, nil
}

// UpdateSchedule обновляет расписание с проверкой владельца.
func (s *Store) UpdateSchedule(schedule *models.UserSchedule) error {
	query := `
		UPDATE user_schedules SET
			day_of_week = $1, time_hour = $2, time_minute = $3,
			email_type = $4, updated_at = CURRENT_TIMESTAMP
		WHERE id = $5 AND user_id = $6 AND is_active = true
	`

	result, err := s.db.Exec(query,
		schedule.DayOfWeek, schedule.TimeHour, schedule.TimeMinute,
		schedule.EmailType, schedule.ID, schedule.UserID,
	)
	if err != nil {
		return fmt.Errorf("ошибка обновления расписания: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("ошибка проверки обновления: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("расписание не найдено или нет доступа")
	}
	return nil
}

// DeleteSchedule деактивирует расписание с проверкой владельца.
func (s *Store) DeleteSchedule(scheduleID int, userID string) error {
	query := `
		UPDATE user_schedules SET is_active = false, updated_at = CURRENT_TIMESTAMP
		WHERE id = $1 AND user_id = $2 AND is_active = true
	`

	result, err := s.db.Exec(query, scheduleID, userID)
	if err != nil {
		return fmt.Errorf("ошибка удаления расписания: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("ошибка проверки удаления: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("расписание не найдено или нет доступа")
	}
	return nil
}

// DeactivateScheduleByID деактивирует расписание по ID (для отписки из письма).
func (s *Store) DeactivateScheduleByID(scheduleID int) error {
	query := `
		UPDATE user_schedules SET is_active = false, updated_at = CURRENT_TIMESTAMP
		WHERE id = $1 AND is_active = true
	`

	result, err := s.db.Exec(query, scheduleID)
	if err != nil {
		return fmt.Errorf("ошибка деактивации расписания: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("ошибка проверки деактивации: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("расписание не найдено или уже деактивировано")
	}
	return nil
}

// GetRecentWorkoutHistory возвращает последние N записей тренировок пользователя.
func (s *Store) GetRecentWorkoutHistory(userID string, limit int) ([]models.WorkoutHistory, error) {
	query := `
		SELECT id, user_id, date, muscle_group
		FROM workout_history
		WHERE user_id = $1
		ORDER BY date DESC
		LIMIT $2
	`

	rows, err := s.db.Query(query, userID, limit)
	if err != nil {
		return nil, fmt.Errorf("ошибка получения истории тренировок: %w", err)
	}
	defer rows.Close()

	var history []models.WorkoutHistory
	for rows.Next() {
		var h models.WorkoutHistory
		if err := rows.Scan(&h.ID, &h.UserID, &h.Date, &h.MuscleGroup); err != nil {
			return nil, fmt.Errorf("ошибка чтения записи тренировки: %w", err)
		}
		history = append(history, h)
	}
	return history, rows.Err()
}

// SaveWorkoutHistory сохраняет мышечную группу тренировки текущего дня.
// При повторном вызове в тот же день обновляет запись (ON CONFLICT).
func (s *Store) SaveWorkoutHistory(userID, muscleGroup string) error {
	query := `
		INSERT INTO workout_history (user_id, date, muscle_group)
		VALUES ($1, CURRENT_DATE, $2)
		ON CONFLICT (user_id, date) DO UPDATE SET muscle_group = $2
	`

	_, err := s.db.Exec(query, userID, muscleGroup)
	if err != nil {
		return fmt.Errorf("ошибка сохранения истории тренировки: %w", err)
	}
	return nil
}

// Ping проверяет доступность базы данных.
func (s *Store) Ping() error {
	return s.db.Ping()
}
