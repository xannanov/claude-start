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
func (s *Store) CreateUser(user *models.User) error {
	query := `
		INSERT INTO users (
			email, first_name, last_name, age, gender,
			height_cm, weight_kg, goal, activity_level, is_active
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id, created_at, updated_at
	`

	err := s.db.QueryRow(
		query,
		user.Email, user.FirstName, user.LastName, user.Age, user.Gender,
		user.HeightCm, user.WeightKg, user.Goal, user.ActivityLevel, true,
	).Scan(&user.ID, &user.CreatedAt, &user.UpdatedAt)

	if err != nil {
		return fmt.Errorf("ошибка создания пользователя: %w", err)
	}
	return nil
}

// CreateUserSchedule создаёт расписание; при дубликате активирует существующее.
func (s *Store) CreateUserSchedule(schedule *models.UserSchedule) error {
	query := `
		INSERT INTO user_schedules (user_id, day_of_week, time_hour, time_minute, email_type, is_active)
		VALUES ($1, $2, $3, $4, $5, true)
		ON CONFLICT (user_id, day_of_week, time_hour, time_minute)
		DO UPDATE SET is_active = true, updated_at = CURRENT_TIMESTAMP
		RETURNING id
	`

	return s.db.QueryRow(
		query,
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
